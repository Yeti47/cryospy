package clients

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/yeti47/cryospy/server/core/encryption"
)

type ClientMekProvider interface {
	// UncoverMek decrypts the MEK for the client using the provided secret.
	// The clientSecret parameter must be hex-encoded.
	UncoverMek(clientID, clientSecret string) ([]byte, error)
}

type clientMekProvider struct {
	encryptor      encryption.Encryptor
	clientRepo     ClientRepository
	clientVerifier ClientVerifier
}

func NewClientMekProvider(encryptor encryption.Encryptor, clientRepo ClientRepository, clientVerifier ClientVerifier) *clientMekProvider {
	return &clientMekProvider{
		encryptor:      encryptor,
		clientRepo:     clientRepo,
		clientVerifier: clientVerifier,
	}
}

// UncoverMek decrypts the MEK for the client using the provided secret.
// The clientSecret parameter must be hex-encoded.
func (p *clientMekProvider) UncoverMek(clientID, clientSecret string) ([]byte, error) {

	// Verify the client secret
	isValid, client, err := p.clientVerifier.VerifyClient(clientID, clientSecret)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, NewClientVerificationError(clientID)
	}

	// Decode the base64 encoded values
	encryptedMek, err := base64.StdEncoding.DecodeString(client.EncryptedMek)
	if err != nil {
		return nil, err
	}

	keyDerivationSalt, err := base64.StdEncoding.DecodeString(client.KeyDerivationSalt)
	if err != nil {
		return nil, err
	}

	// Derive the encryption key from the client secret using the salt.
	// The client secret is hex-encoded, so we need to decode it first
	clientSecretBytes, err := hex.DecodeString(clientSecret)
	if err != nil {
		return nil, err
	}

	derivedKey, err := p.encryptor.DeriveKeyFromSecret(clientSecretBytes, keyDerivationSalt)
	if err != nil {
		return nil, err
	}

	// Decrypt the MEK using the derived key
	mek, err := p.encryptor.Decrypt(encryptedMek, derivedKey)
	if err != nil {
		return nil, err
	}

	return mek, nil
}
