package clients

import (
	"context"
	"encoding/base64"
	"encoding/hex"

	"github.com/yeti47/cryospy/server/core/encryption"
)

type ClientVerifier interface {
	// VerifyClient verifies the client using its ID and secret
	// The clientSecret parameter should be hex-encoded
	VerifyClient(clientID, clientSecret string) (bool, *Client, error)
}

type clientVerifier struct {
	clientRepo ClientRepository
	encryptor  encryption.Encryptor
}

func NewClientVerifier(clientRepo ClientRepository, encryptor encryption.Encryptor) *clientVerifier {
	return &clientVerifier{
		clientRepo: clientRepo,
		encryptor:  encryptor,
	}
}

// VerifyClient verifies the client using its ID and secret
// The clientSecret parameter should be hex-encoded
func (v *clientVerifier) VerifyClient(clientID, clientSecret string) (bool, *Client, error) {
	// Retrieve the client by ID
	client, err := v.clientRepo.GetByID(context.Background(), clientID)
	if err != nil {
		return false, nil, err
	}
	if client == nil {
		return false, nil, NewClientVerificationError(clientID) // don't specify the reason to avoid leaking information
	}

	// Verify the client secret
	// First decode the client secret from hex (since it was sent as hex-encoded string)
	decodedClientSecret, err := hex.DecodeString(clientSecret)
	if err != nil {
		return false, nil, NewClientVerificationError(clientID) // don't specify the reason to avoid leaking information
	}

	// Base64-decode the secret salt from the client
	decodedSecretSalt, err := base64.StdEncoding.DecodeString(client.SecretSalt)
	if err != nil {
		return false, nil, err
	}

	// Also decode the secret hash from the client
	decodedSecretHash, err := base64.StdEncoding.DecodeString(client.SecretHash)
	if err != nil {
		return false, nil, err
	}

	// Compare hash
	if !v.encryptor.CompareHash(decodedSecretHash, decodedClientSecret, decodedSecretSalt) {
		return false, nil, NewClientVerificationError(clientID) // don't specify the reason to avoid leaking information
	}

	return true, client, nil
}
