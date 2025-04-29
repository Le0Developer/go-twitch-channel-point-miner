package miner

import (
	"bytes"
	"encoding/json"
	"net/http"
)

const (
	userAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; rv:122) Gecko/20100101 Firefox/122.0"
	defaultClientID = "ue6666qo983tsx6so1t0vnawi233wa" // TV
)

type GraphQL struct {
	User          *User
	ClientSession string
	ClientVersion string
	DeviceID      string
	Client        *http.Client
}

func (gql *GraphQL) SendRequest(payload GraphQLRequest, ptr any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", "https://gql.twitch.tv/gql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "OAuth "+gql.User.AuthToken)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Client-ID", defaultClientID)
	request.Header.Set("Client-Session-ID", gql.ClientSession)
	request.Header.Set("Client-Version", gql.ClientVersion)
	request.Header.Set("X-Device-ID", gql.DeviceID)

	response, err := gql.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return json.NewDecoder(response.Body).Decode(ptr)
}

type GraphQLRequest struct {
	OperationName string                   `json:"operationName"`
	Variables     map[string]any           `json:"variables"`
	Extensions    GraphQLRequestExtensions `json:"extensions"`
}

type GraphQLRequestExtensions struct {
	PersistedQuery GraphQLRequestExtensionsPersistedQuery `json:"persistedQuery"`
}

type GraphQLRequestExtensionsPersistedQuery struct {
	Version    int    `json:"version"`
	Sha256Hash string `json:"sha256Hash"`
}

func NewGraphQL(user *User) *GraphQL {
	client := &http.Client{}
	deviceID := createRandomString(32)
	clientSession := createRandomString(16, hexAlphabet)

	gql := &GraphQL{
		User:          user,
		ClientSession: clientSession,
		ClientVersion: "",
		DeviceID:      deviceID,
		Client:        client,
	}
	return gql
}
