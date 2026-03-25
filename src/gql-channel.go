package miner

func (gql *GraphQL) GetSteamerID(name string) (string, error) {
	req := GraphQLRequest{
		OperationName: "GetIDFromLogin",
		Variables: map[string]any{
			"login": name,
		},
		Extensions: GraphQLRequestExtensions{
			PersistedQuery: GraphQLRequestExtensionsPersistedQuery{
				Version:    1,
				Sha256Hash: "94e82a7b1e3c21e186daa73ee2afc4b8f23bade1fbbff6fe8ac133f50a2f58ca",
			},
		},
	}

	var res reportMenuItemResponse
	if err := gql.SendRequest(req, &res); err != nil {
		return "", err
	}

	return res.Data.User.ID, nil
}

type reportMenuItemResponse struct {
	Data struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	} `json:"data"`
}

func (gql *GraphQL) GetStreamBroadcastID(streamer *Streamer) error {
	req := GraphQLRequest{
		OperationName: "VideoPlayerStreamInfoOverlayChannel",
		Variables: map[string]any{
			"channel": streamer.Username,
		},
		Extensions: GraphQLRequestExtensions{
			PersistedQuery: GraphQLRequestExtensionsPersistedQuery{
				Version:    1,
				Sha256Hash: "a5f2e34d626a9f4f5c0204f910bab2194948a9502089be558bb6e779a9e1b3d2",
			},
		},
	}

	var res videoPlayerStreamInfoOverlayChannelResponse
	if err := gql.SendRequest(req, &res); err != nil {
		return err
	}

	if res.Data.User != nil && res.Data.User.Stream != nil {
		streamer.BroadcastID = res.Data.User.Stream.ID
	}

	return nil
}

type videoPlayerStreamInfoOverlayChannelResponse struct {
	Data struct {
		User *struct {
			Stream *struct {
				ID string `json:"id"`
			} `json:"stream"`
		} `json:"user"`
	} `json:"data"`
}
