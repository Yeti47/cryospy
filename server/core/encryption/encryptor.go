package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// Constants for encryption parameters
const (
	iterationCount = 10000 // PBKDF2 iterations
	keyLength      = 32    // 256 bits for AES-256
	saltLength     = 16    // 128 bits for salt
	nonceLength    = 12    // 96 bits for GCM nonce
)

type Encryptor interface {
	// Encrypt encrypts the given data using the provided key
	Encrypt(data []byte, key []byte) ([]byte, error)
	// Decrypt decrypts the given data using the provided key
	Decrypt(data []byte, key []byte) ([]byte, error)
	// GenerateKey generates a new encryption key
	GenerateKey() ([]byte, error)
	// GenerateSalt generates a new salt for key derivation
	GenerateSalt() ([]byte, error)
	// DeriveKeyFromSecret derives an encryption key from a secret and salt
	DeriveKeyFromSecret(secret []byte, salt []byte) ([]byte, error)
	// Hash hashes the given data using a secure hash function
	Hash(data []byte) (hashedData, salt []byte, err error)
	// CompareHash compares a hashed value with a plain value using the provided salt
	CompareHash(hashedValue, plainValue, salt []byte) bool
}

// AESEncryptor implements the Encryptor interface using AES-GCM
type AESEncryptor struct{}

// NewAESEncryptor creates a new AESEncryptor instance
func NewAESEncryptor() *AESEncryptor {
	return &AESEncryptor{}
}

// Encrypt encrypts data using AES-GCM with the provided key
func (e *AESEncryptor) Encrypt(data []byte, key []byte) ([]byte, error) {
	// Validate key length
	if len(key) != keyLength {
		return nil, errors.New("invalid key length")
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM with the provided key
func (e *AESEncryptor) Decrypt(data []byte, key []byte) ([]byte, error) {
	// Validate key length
	if len(key) != keyLength {
		return nil, errors.New("invalid key length")
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Check minimum data length
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GenerateKey generates a new random encryption key
func (e *AESEncryptor) GenerateKey() ([]byte, error) {
	key := make([]byte, keyLength)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// GenerateSalt generates a new random salt
func (e *AESEncryptor) GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// DeriveKeyFromSecret derives an encryption key from a secret and salt using PBKDF2
func (e *AESEncryptor) DeriveKeyFromSecret(secret []byte, salt []byte) ([]byte, error) {
	if len(salt) == 0 {
		return nil, errors.New("salt cannot be empty")
	}

	key := pbkdf2.Key(secret, salt, iterationCount, keyLength, sha256.New)
	return key, nil
}

// Hash hashes the given data using SHA-256 with a random salt
func (e *AESEncryptor) Hash(data []byte) (hashedData, salt []byte, err error) {
	// Generate salt
	salt, err = e.GenerateSalt()
	if err != nil {
		return nil, nil, err
	}

	// Create hash with salt
	hasher := sha256.New()
	hasher.Write(salt)
	hasher.Write(data)
	hashedData = hasher.Sum(nil)

	return hashedData, salt, nil
}

// CompareHash compares a hashed value with a plain value using the provided salt
func (e *AESEncryptor) CompareHash(hashedValue, plainValue, salt []byte) bool {
	// Derive the hash of the plain value with the same salt
	hasher := sha256.New()
	hasher.Write(salt)
	hasher.Write(plainValue)
	computedHash := hasher.Sum(nil)

	// compare using constant time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(hashedValue, computedHash) == 1
}
