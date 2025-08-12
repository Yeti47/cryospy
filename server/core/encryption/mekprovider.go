package encryption

import (
	"encoding/base64"
)

type MekProvider interface {
	// UncoverMek decrypts the MEK using the provided password
	UncoverMek(password string) ([]byte, error)
}

type mekProvider struct {
	encryptor Encryptor
	repo      MekRepository
}

func NewMekProvider(encryptor Encryptor, repo MekRepository) mekProvider {
	return mekProvider{
		encryptor: encryptor,
		repo:      repo,
	}
}

func (p *mekProvider) UncoverMek(password string) ([]byte, error) {
	mek, err := p.repo.Get()
	if err != nil {
		return nil, err
	}
	if mek == nil {
		return nil, NewMekNotFoundError()
	}

	// Get the password-derived key. First basse64-decode the salt
	salt, err := base64.StdEncoding.DecodeString(mek.EncryptionKeySalt)
	if err != nil {
		return nil, err
	}

	// Derive the key from the password and salt
	key, err := p.encryptor.DeriveKeyFromSecret([]byte(password), salt)
	if err != nil {
		return nil, err
	}

	// Decrypt the MEK using the provided password
	decryptedMek, err := p.encryptor.Decrypt([]byte(mek.EncryptedEncryptionKey), key)
	if err != nil {
		return nil, err
	}

	return decryptedMek, nil
}
