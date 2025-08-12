package clients

import (
	"encoding/base64"

	"github.com/yeti47/cryospy/server/core/encryption"
)

type ClientMekProvider interface {
	// UncoverMek decrypts the MEK for the client using the provided secret
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

	// Derive the encryption key from the client secret using the salt
	derivedKey, err := p.encryptor.DeriveKeyFromSecret([]byte(clientSecret), keyDerivationSalt)
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
