package miner

func (gql *GraphQL) PlaybackAccessToken(streamer *Streamer) (string, string, error) {
	req := GraphQLRequest{
		OperationName: "PlaybackAccessToken",
		Variables: map[string]any{
			"login":  streamer.Username,
			"isLive": true,
			"isVod":  false,
			"vodID":  "",
			// "playerType": "picture-by-picture",
			"playerType": "site",
		},
		Extensions: GraphQLRequestExtensions{
			PersistedQuery: GraphQLRequestExtensionsPersistedQuery{
				Version:    1,
				Sha256Hash: "3093517e37e4f4cb48906155bcd894150aef92617939236d2508f3375ab732ce",
			},
		},
	}

	var res playbackAccessTokenResponse
	if err := gql.SendRequest(req, &res); err != nil {
		return "", "", err
	}

	signature := res.Data.StreamPlaybackAccessToken.Signature
	value := res.Data.StreamPlaybackAccessToken.Value

	return signature, value, nil
}

type playbackAccessTokenResponse struct {
	Data struct {
		StreamPlaybackAccessToken struct {
			Signature string `json:"signature"`
			Value     string `json:"value"`
		} `json:"streamPlaybackAccessToken"`
	} `json:"data"`
}
