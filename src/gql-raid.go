package miner

func (gql *GraphQL) JoinRaid(raidID string) error {
	req := GraphQLRequest{
		OperationName: "JoinRaid",
		Variables: map[string]any{
			"input": map[string]any{
				"raidID": raidID,
			},
		},
		Extensions: GraphQLRequestExtensions{
			PersistedQuery: GraphQLRequestExtensionsPersistedQuery{
				Version:    1,
				Sha256Hash: "c6a332a86d1087fbbb1a8623aa01bd1313d2386e7c63be60fdb2d1901f01a4ae",
			},
		},
	}

	var res any
	return gql.SendRequest(req, &res)
}
