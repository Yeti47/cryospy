package encryption

import (
	"encoding/base64"
	"fmt"
)

// DecryptMek decrypts the Master Encryption Key (MEK) using a password.
// It derives a key from the password and the MEK's salt, then decrypts the
// encrypted MEK value.
func DecryptMek(mek *Mek, password string, encryptor Encryptor) ([]byte, error) {
	// Decode the salt from base64
	salt, err := base64.StdEncoding.DecodeString(mek.EncryptionKeySalt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode MEK salt: %w", err)
	}

	// Derive the key from the password
	key, err := encryptor.DeriveKeyFromSecret([]byte(password), salt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key from password: %w", err)
	}

	// Decode the encrypted MEK from base64
	encryptedMek, err := base64.StdEncoding.DecodeString(mek.EncryptedEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted MEK: %w", err)
	}

	// Decrypt the MEK
	decryptedMek, err := encryptor.Decrypt(encryptedMek, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt MEK: %w", err)
	}

	return decryptedMek, nil
}
