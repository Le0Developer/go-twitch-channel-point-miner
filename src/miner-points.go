package miner

import (
	"cmp"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

func (miner *Miner) OnPointsUpdate(message WebsocketMessage) {
	userID := message.Topics[1]

	var event pointsUpdateEvent
	if err := json.Unmarshal(message.Data, &event); err != nil {
		fmt.Println("Failed to unmarshal points update event", err)
		return
	}

	balance := event.Data.Balance
	streamer := miner.GetStreamerByID(balance.ChannelID)

	if streamer == nil {
		fmt.Println("Streamer not found for points update:", balance.ChannelID)
		return
	}

	for _, user := range miner.Users {
		if user.ID == userID {
			miner.Lock.Lock()
			streamer.Points[user] = int(balance.Balance)
			if event.Data.PointGain != nil && event.Data.PointGain.ReasonCode == "WATCH" {
				streamer.GotPointsOnce[user] = true
			}
			miner.Lock.Unlock()
		}
	}
}

type pointsUpdateEvent struct {
	Data struct {
		Balance struct {
			ChannelID string  `json:"channel_id"`
			Balance   float64 `json:"balance"`
		} `json:"balance"`
		PointGain *struct {
			ReasonCode string `json:"reason_code"`
		} `json:"point_gain"`
	} `json:"data"`
}

func (miner *Miner) OnClaimAvailable(message WebsocketMessage) {
	var event claimAvailableEvent
	if err := json.Unmarshal(message.Data, &event); err != nil {
		fmt.Println("Failed to unmarshal claim available event", err)
		return
	}

	userID := message.Topics[1]
	claim := event.Data.Claim
	streamer := miner.GetStreamerByID(claim.ChannelID)

	if streamer == nil {
		fmt.Println("Streamer not found for claim available:", claim.ChannelID)
		return
	}

	for _, user := range miner.Users {
		if user.ID == userID {
			user.GraphQL.ClaimBonus(streamer, claim.ID)
		}
	}
}

type claimAvailableEvent struct {
	Data struct {
		Claim struct {
			ID        string `json:"id"`
			ChannelID string `json:"channel_id"`
		} `json:"claim"`
	} `json:"data"`
}

func (miner *Miner) MinePoints(user *User) error {
	streamers := make([]*Streamer, 0, len(user.Streamers))
	for _, streamer := range user.Streamers {
		streamers = append(streamers, streamer)
	}

	slices.SortStableFunc(streamers, func(a, b *Streamer) int {
		// prioritize streamers who havent been mined yet (to get the streak bonus)
		if miner.Options.PrioritizeStreaks && a.GotPointsOnce[user] != b.GotPointsOnce[user] {
			if a.GotPointsOnce[user] {
				return 1
			}
			return -1
		}
		// alternatively, mine whoever has the lowest points
		return cmp.Compare(a.Points[user], b.Points[user])
	})

	for _, streamer := range streamers {
		if streamer.IsLive() {
			fmt.Printf("Streamer %s (%s): %d points, mined=%v\n", streamer.Username, streamer.ID, streamer.Points[user], streamer.GotPointsOnce[user])
		}
	}

	mined := 0
	for _, streamer := range streamers {
		// no points => points disabled usually. TODO: better check if points are disabled
		if !streamer.IsLive() || streamer.Points[user] == 0 {
			continue
		}
		if miner.Options.ConcurrentStreamLimit > 0 && mined >= miner.Options.ConcurrentStreamLimit {
			break
		}

		if err := miner.minePoints(streamer, user); err != nil {
			fmt.Println("Error mining points for", streamer.Username, ":", err)
			continue
		}
		mined++
	}

	return nil
}

func (miner *Miner) minePoints(streamer *Streamer, user *User) error {
	fmt.Println("Mining points for", streamer.Username, streamer.ID, "mined=", streamer.GotPointsOnce[user], "points=", streamer.Points[user])
	if err := miner.minePointsPlayback(streamer, user); err != nil {
		return fmt.Errorf("failed to mine points on playback: %w", err)
	}
	if err := miner.minePointsSpade(streamer, user); err != nil {
		return fmt.Errorf("failed to mine points on spade: %w", err)
	}
	return nil
}

// send a request to the current HLS playlist to make them think we're watching
func (miner *Miner) minePointsPlayback(streamer *Streamer, user *User) error {
	// get auth token
	signature, value, err := user.GraphQL.PlaybackAccessToken(streamer)
	if err != nil {
		return fmt.Errorf("failed to get playback access token: %w", err)
	}

	requestBroadcastQualitiesURL := fmt.Sprintf("https://usher.ttvnw.net/api/channel/hls/%s.m3u8?sig=%s&token=%s", streamer.Username, signature, value)
	request, err := http.NewRequest("GET", requestBroadcastQualitiesURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("User-Agent", userAgent)

	response, err := user.GraphQL.Client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// response is m3u8 list of qualities
	defer response.Body.Close()
	responseText, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to mine points, status code: %d", response.StatusCode)
	}

	// lowest quality should be the last one
	qualities := strings.Split(string(responseText), "\n")
	playlistURL := qualities[len(qualities)-1]

	// get the playlist
	request, err = http.NewRequest("GET", playlistURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("User-Agent", userAgent)
	response, err = user.GraphQL.Client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// response is m3u8 playlist
	defer response.Body.Close()
	responseText, err = io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// get the last segment
	segments := strings.Split(string(responseText), "\n")
	segmentURL := segments[len(segments)-2]

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to mine points, status code: %d", response.StatusCode)
	}

	// send a HEAD request to the segment
	request, err = http.NewRequest("HEAD", segmentURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("User-Agent", userAgent)
	response, err = user.GraphQL.Client.Do(request)
	response.Body.Close()

	fmt.Println("Mined points for", streamer.Username, "on playback", segmentURL)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to mine points, status code: %d", response.StatusCode)
	}
	return nil
}

func (miner *Miner) minePointsSpade(streamer *Streamer, user *User) error {
	if streamer.BroadcastID == "" {
		fmt.Println("Don't have broadcast id, getting it")
		if err := user.GraphQL.GetStreamBroadcastID(streamer); err != nil {
			return fmt.Errorf("failed to get broadcast id: %w", err)
		}
	}

	payload := []map[string]any{
		{
			"event": "minute-watched",
			"properties": map[string]any{
				"channel_id":   streamer.ID,
				"broadcast_id": streamer.BroadcastID,
				"player":       "site",
				"user_id":      user.ID,
				"live":         true,
				"channel":      streamer.Username,
			},
		},
	}

	payloadEncoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	payloadBase64encoded := base64.StdEncoding.EncodeToString(payloadEncoded)

	data := url.Values{
		"data": []string{payloadBase64encoded},
	}

	body := strings.NewReader(data.Encode())

	request, err := http.NewRequest("POST", miner.SpadeUrl, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("User-Agent", userAgent)

	response, err := user.GraphQL.Client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	_, err = io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Mined points for", streamer.Username, "on spade")
	return nil
}
