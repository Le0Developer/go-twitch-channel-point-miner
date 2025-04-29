package miner

func (gql *GraphQL) GetFollows() ([]string, error) {
	req := GraphQLRequest{
		OperationName: "ChannelFollows",
		Variables: map[string]any{
			"limit": 100,
			"order": "ASC",
		},
		Extensions: GraphQLRequestExtensions{
			PersistedQuery: GraphQLRequestExtensionsPersistedQuery{
				Version:    1,
				Sha256Hash: "eecf815273d3d949e5cf0085cc5084cd8a1b5b7b6f7990cf43cb0beadf546907",
			},
		},
	}

	follows := []string{}
	cursor := ""
	var res channelFollowsResponse
	for {
		req.Variables["cursor"] = cursor
		if err := gql.SendRequest(req, &res); err != nil {
			return nil, err
		}

		followsResponse := res.Data.User.Follows
		for _, follow := range followsResponse.Edges {
			follows = append(follows, follow.Node.Login)
			cursor = follow.Cursor
		}

		if !followsResponse.PageInfo.HasNextPage {
			break
		}
	}

	return follows, nil
}

type channelFollowsResponse struct {
	Data struct {
		User struct {
			Follows struct {
				Edges []struct {
					Node struct {
						Login string `json:"login"`
					} `json:"node"`
					Cursor string `json:"cursor"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool `json:"hasNextPage"`
				} `json:"pageInfo"`
			} `json:"follows"`
		} `json:"user"`
	} `json:"data"`
}
