package miner

func (gql *GraphQL) LoadChannelPoints(streamer *Streamer) error {
	req := GraphQLRequest{
		OperationName: "ChannelPointsContext",
		Variables: map[string]any{
			"channelLogin": streamer.Username,
		},
		Extensions: GraphQLRequestExtensions{
			PersistedQuery: GraphQLRequestExtensionsPersistedQuery{
				Version:    1,
				Sha256Hash: "1530a003a7d374b0380b79db0be0534f30ff46e61cffa2bc0e2468a909fbc024",
			},
		},
	}

	var res channelPointsContextResponse
	if err := gql.SendRequest(req, &res); err != nil {
		return err
	}

	communityPoints := res.Data.Community.Channel.Self.CommunityPoints
	gql.User.Miner.Lock.Lock()
	streamer.Points[gql.User] = int(communityPoints.Balance)
	gql.User.Miner.Lock.Unlock()

	if communityPoints.AvailableClaim != nil {
		if err := gql.ClaimBonus(streamer, communityPoints.AvailableClaim.ID); err != nil {
			return err
		}
	}

	return nil
}

type channelPointsContextResponse struct {
	Data struct {
		Community struct {
			Channel struct {
				Self struct {
					CommunityPoints struct {
						Balance        float64 `json:"balance"`
						AvailableClaim *struct {
							ID string `json:"id"`
						} `json:"availableClaim"`
					} `json:"communityPoints"`
				} `json:"self"`
			} `json:"channel"`
		} `json:"community"`
	} `json:"data"`
}

func (gql *GraphQL) ClaimBonus(streamer *Streamer, claimID string) error {
	req := GraphQLRequest{
		OperationName: "ClaimCommunityPoints",
		Variables: map[string]any{
			"input": map[string]any{
				"channelID": streamer.ID,
				"claimID":   claimID,
			},
		},
		Extensions: GraphQLRequestExtensions{
			PersistedQuery: GraphQLRequestExtensionsPersistedQuery{
				Version:    1,
				Sha256Hash: "46aaeebe02c99afdf4fc97c7c0cba964124bf6b0af229395f1f6d1feed05b3d0",
			},
		},
	}

	var res any
	return gql.SendRequest(req, &res)
}
