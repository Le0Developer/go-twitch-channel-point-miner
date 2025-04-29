package miner

import "github.com/davecgh/go-spew/spew"

func (gql *GraphQL) MakePrediction(eventID string, outcomeID string, points int) error {
	request := GraphQLRequest{
		OperationName: "MakePrediction",
		Variables: map[string]any{
			"input": map[string]any{
				"eventID":       eventID,
				"outcomeID":     outcomeID,
				"points":        points,
				"transactionID": createRandomString(16, hexAlphabet),
			},
		},
		Extensions: GraphQLRequestExtensions{
			PersistedQuery: GraphQLRequestExtensionsPersistedQuery{
				Version:    1,
				Sha256Hash: "b44682ecc88358817009f20e69d75081b1e58825bb40aa53d5dbadcc17c881d8",
			},
		},
	}

	var response any
	err := gql.SendRequest(request, &response)
	spew.Dump(response)
	return err
}
