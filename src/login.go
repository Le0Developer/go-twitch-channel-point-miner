package miner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TV
func NewLoginSession(clientID ...string) *LoginSession {
	finalClientID := defaultClientID
	if len(clientID) > 0 {
		finalClientID = clientID[0]
	}

	return &LoginSession{
		clientID: finalClientID,
		deviceID: createRandomString(32),
	}
}

type LoginSession struct {
	clientID string
	deviceID string

	deviceCode string
	interval   time.Duration
	expiration time.Time
}

func (l *LoginSession) GetCode() (string, error) {
	body := url.Values{
		"client_id": {l.clientID},
		"scopes":    {"channel_read chat:read user_blocks_edit user_blocks_read user_follows_edit user_read"},
	}

	req, err := http.NewRequest("POST", "https://id.twitch.tv/oauth2/device", strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := l.sendRequest(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get code: %s", res.Status)
	}

	var response map[string]any
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", err
	}

	l.deviceCode = response["device_code"].(string)
	l.interval = time.Duration(response["interval"].(float64)) * time.Second
	l.expiration = time.Now().Add(time.Duration(response["expires_in"].(float64)) * time.Second)

	return response["user_code"].(string), nil
}

func (l *LoginSession) CheckCode() (string, error) {
	if time.Now().After(l.expiration) {
		return "", fmt.Errorf("device code expired")
	}

	body := url.Values{
		"client_id":   {l.clientID},
		"device_code": {l.deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	req, err := http.NewRequest("POST", "https://id.twitch.tv/oauth2/token", strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := l.sendRequest(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode == http.StatusOK {
		var response map[string]any
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			return "", err
		}
		return response["access_token"].(string), nil
	} else if res.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("authorization pending")
	} else if res.StatusCode == http.StatusBadRequest {
		var response map[string]any
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			return "", err
		}
		if res := response["message"]; res != nil {
			return "", fmt.Errorf("error: %s", res)
		}
		return "", fmt.Errorf("invalid request")
	}

	return "", fmt.Errorf("failed to check code: %s", res.Status)
}

func (l *LoginSession) WaitForToken() (string, error) {
	for time.Now().Before(l.expiration) {
		time.Sleep(l.interval)
		token, err := l.CheckCode()
		if err != nil {
			if err.Error() != "error: authorization_pending" {
				return "", err
			}
			continue
		}

		return token, nil
	}
	return "", fmt.Errorf("device code expired")
}

func (l *LoginSession) sendRequest(request *http.Request) (*http.Response, error) {
	request.Header.Set("Client-ID", l.clientID)
	request.Header.Set("X-Device-ID", l.deviceID)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Origin", "https://android.tv.twitch.tv")

	return http.DefaultClient.Do(request)
}
