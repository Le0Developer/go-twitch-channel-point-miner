package miner

func (gql *GraphQL) ClaimMoment(momentID string) error {
	req := GraphQLRequest{
		OperationName: "CommunityMomentCallout_Claim",
		Variables: map[string]any{
			"input": map[string]any{
				"momentID": momentID,
			},
		},
		Extensions: GraphQLRequestExtensions{
			PersistedQuery: GraphQLRequestExtensionsPersistedQuery{
				Version:    1,
				Sha256Hash: "e2d67415aead910f7f9ceb45a77b750a1e1d9622c936d832328a0689e054db62",
			},
		},
	}

	var res any
	return gql.SendRequest(req, &res)
}
