package miner

import "math/rand"

const (
	hexAlphabet   = "0123456789abcdef"
	nonceAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func createRandomString(length int, alphabetOptional ...string) string {
	alphabet := nonceAlphabet
	if len(alphabetOptional) > 0 {
		alphabet = alphabetOptional[0]
	}

	result := make([]byte, length)
	for i := range result {
		result[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(result)
}
