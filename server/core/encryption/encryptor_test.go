package encryption

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func TestNewAESEncryptor(t *testing.T) {
	encryptor := NewAESEncryptor()
	if encryptor == nil {
		t.Fatal("NewAESEncryptor() returned nil")
	}
}

func TestGenerateKey(t *testing.T) {
	encryptor := NewAESEncryptor()

	key, err := encryptor.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() failed: %v", err)
	}

	if len(key) != keyLength {
		t.Errorf("Expected key length %d, got %d", keyLength, len(key))
	}

	// Generate another key and ensure they're different
	key2, err := encryptor.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() failed on second call: %v", err)
	}

	if bytes.Equal(key, key2) {
		t.Error("GenerateKey() produced identical keys, should be random")
	}
}

func TestGenerateSalt(t *testing.T) {
	encryptor := NewAESEncryptor()

	salt, err := encryptor.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() failed: %v", err)
	}

	if len(salt) != saltLength {
		t.Errorf("Expected salt length %d, got %d", saltLength, len(salt))
	}

	// Generate another salt and ensure they're different
	salt2, err := encryptor.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() failed on second call: %v", err)
	}

	if bytes.Equal(salt, salt2) {
		t.Error("GenerateSalt() produced identical salts, should be random")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	encryptor := NewAESEncryptor()

	// Generate a key
	key, err := encryptor.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() failed: %v", err)
	}

	testData := []byte("Hello, World! This is a test message.")

	// Encrypt the data
	encrypted, err := encryptor.Encrypt(testData, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	if bytes.Equal(testData, encrypted) {
		t.Error("Encrypted data should not equal original data")
	}

	// Decrypt the data
	decrypted, err := encryptor.Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if !bytes.Equal(testData, decrypted) {
		t.Errorf("Decrypted data does not match original. Expected %s, got %s", testData, decrypted)
	}
}

func TestEncryptWithInvalidKey(t *testing.T) {
	encryptor := NewAESEncryptor()
	testData := []byte("test data")

	// Test with wrong key length
	invalidKey := []byte("short")
	_, err := encryptor.Encrypt(testData, invalidKey)
	if err == nil {
		t.Error("Encrypt() should fail with invalid key length")
	}

	// Test with empty key
	_, err = encryptor.Encrypt(testData, []byte{})
	if err == nil {
		t.Error("Encrypt() should fail with empty key")
	}
}

func TestDecryptWithInvalidKey(t *testing.T) {
	encryptor := NewAESEncryptor()

	// First create valid encrypted data
	key, _ := encryptor.GenerateKey()
	testData := []byte("test data")
	encrypted, _ := encryptor.Encrypt(testData, key)

	// Test with wrong key length
	invalidKey := []byte("short")
	_, err := encryptor.Decrypt(encrypted, invalidKey)
	if err == nil {
		t.Error("Decrypt() should fail with invalid key length")
	}

	// Test with wrong key
	wrongKey, _ := encryptor.GenerateKey()
	_, err = encryptor.Decrypt(encrypted, wrongKey)
	if err == nil {
		t.Error("Decrypt() should fail with wrong key")
	}
}

func TestDecryptWithInvalidData(t *testing.T) {
	encryptor := NewAESEncryptor()
	key, _ := encryptor.GenerateKey()

	// Test with too short data
	shortData := []byte("short")
	_, err := encryptor.Decrypt(shortData, key)
	if err == nil {
		t.Error("Decrypt() should fail with data too short")
	}

	// Test with empty data
	_, err = encryptor.Decrypt([]byte{}, key)
	if err == nil {
		t.Error("Decrypt() should fail with empty data")
	}
}

func TestEncryptDeterminism(t *testing.T) {
	encryptor := NewAESEncryptor()
	key, _ := encryptor.GenerateKey()
	testData := []byte("test data")

	// Encrypt the same data twice
	encrypted1, err := encryptor.Encrypt(testData, key)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	encrypted2, err := encryptor.Encrypt(testData, key)
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	// Results should be different due to random nonces
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("Encryption should not be deterministic (results should differ due to random nonces)")
	}

	// But both should decrypt to the same original data
	decrypted1, _ := encryptor.Decrypt(encrypted1, key)
	decrypted2, _ := encryptor.Decrypt(encrypted2, key)

	if !bytes.Equal(decrypted1, testData) || !bytes.Equal(decrypted2, testData) {
		t.Error("Both encrypted versions should decrypt to original data")
	}
}

func TestDeriveKeyFromSecret(t *testing.T) {
	encryptor := NewAESEncryptor()

	secret := []byte("my secret password")
	salt, err := encryptor.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() failed: %v", err)
	}

	// Derive key
	key, err := encryptor.DeriveKeyFromSecret(secret, salt)
	if err != nil {
		t.Fatalf("DeriveKeyFromSecret() failed: %v", err)
	}

	if len(key) != keyLength {
		t.Errorf("Expected derived key length %d, got %d", keyLength, len(key))
	}

	// Derive the same key again with same inputs
	key2, err := encryptor.DeriveKeyFromSecret(secret, salt)
	if err != nil {
		t.Fatalf("DeriveKeyFromSecret() failed on second call: %v", err)
	}

	if !bytes.Equal(key, key2) {
		t.Error("DeriveKeyFromSecret() should produce identical results with same inputs")
	}

	// Test with different salt should produce different key
	salt2, _ := encryptor.GenerateSalt()
	key3, err := encryptor.DeriveKeyFromSecret(secret, salt2)
	if err != nil {
		t.Fatalf("DeriveKeyFromSecret() failed with different salt: %v", err)
	}

	if bytes.Equal(key, key3) {
		t.Error("DeriveKeyFromSecret() should produce different keys with different salts")
	}
}

func TestDeriveKeyFromSecretWithEmptySalt(t *testing.T) {
	encryptor := NewAESEncryptor()
	secret := []byte("my secret password")

	_, err := encryptor.DeriveKeyFromSecret(secret, []byte{})
	if err == nil {
		t.Error("DeriveKeyFromSecret() should fail with empty salt")
	}

	_, err = encryptor.DeriveKeyFromSecret(secret, nil)
	if err == nil {
		t.Error("DeriveKeyFromSecret() should fail with nil salt")
	}
}

func TestDeriveKeyCompatibility(t *testing.T) {
	encryptor := NewAESEncryptor()

	secret := []byte("test secret")
	salt := []byte("testsalt12345678") // 16 bytes

	// Our implementation
	ourKey, err := encryptor.DeriveKeyFromSecret(secret, salt)
	if err != nil {
		t.Fatalf("DeriveKeyFromSecret() failed: %v", err)
	}

	// Direct PBKDF2 call for comparison
	expectedKey := pbkdf2.Key(secret, salt, iterationCount, keyLength, sha256.New)

	if !bytes.Equal(ourKey, expectedKey) {
		t.Error("DeriveKeyFromSecret() does not match direct PBKDF2 implementation")
	}
}

func TestHash(t *testing.T) {
	encryptor := NewAESEncryptor()
	testData := []byte("test data to hash")

	hash, salt, err := encryptor.Hash(testData)
	if err != nil {
		t.Fatalf("Hash() failed: %v", err)
	}

	if len(hash) != sha256.Size {
		t.Errorf("Expected hash length %d, got %d", sha256.Size, len(hash))
	}

	if len(salt) != saltLength {
		t.Errorf("Expected salt length %d, got %d", saltLength, len(salt))
	}

	// Hash the same data again
	hash2, salt2, err := encryptor.Hash(testData)
	if err != nil {
		t.Fatalf("Hash() failed on second call: %v", err)
	}

	// Salts should be different
	if bytes.Equal(salt, salt2) {
		t.Error("Hash() should generate different salts")
	}

	// Hashes should be different due to different salts
	if bytes.Equal(hash, hash2) {
		t.Error("Hash() should produce different hashes with different salts")
	}
}

func TestHashVerification(t *testing.T) {
	encryptor := NewAESEncryptor()
	testData := []byte("test data to hash")

	hash, salt, err := encryptor.Hash(testData)
	if err != nil {
		t.Fatalf("Hash() failed: %v", err)
	}

	// Manually verify the hash
	hasher := sha256.New()
	hasher.Write(salt)
	hasher.Write(testData)
	expectedHash := hasher.Sum(nil)

	if !bytes.Equal(hash, expectedHash) {
		t.Error("Hash() does not produce expected SHA-256 hash")
	}
}

func TestIntegrationEncryptionWithDerivedKey(t *testing.T) {
	encryptor := NewAESEncryptor()

	// Use key derivation in a realistic scenario
	password := []byte("user password 123")
	salt, err := encryptor.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() failed: %v", err)
	}

	key, err := encryptor.DeriveKeyFromSecret(password, salt)
	if err != nil {
		t.Fatalf("DeriveKeyFromSecret() failed: %v", err)
	}

	// Encrypt some data
	testData := []byte("Sensitive user data that needs encryption")
	encrypted, err := encryptor.Encrypt(testData, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Decrypt with the same derived key
	decrypted, err := encryptor.Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if !bytes.Equal(testData, decrypted) {
		t.Error("Integration test failed: decrypted data does not match original")
	}

	// Verify we can derive the same key again with the same password and salt
	key2, err := encryptor.DeriveKeyFromSecret(password, salt)
	if err != nil {
		t.Fatalf("DeriveKeyFromSecret() failed on second derivation: %v", err)
	}

	// Should be able to decrypt with the re-derived key
	decrypted2, err := encryptor.Decrypt(encrypted, key2)
	if err != nil {
		t.Fatalf("Decrypt() with re-derived key failed: %v", err)
	}

	if !bytes.Equal(testData, decrypted2) {
		t.Error("Integration test failed: cannot decrypt with re-derived key")
	}
}

func TestEmptyData(t *testing.T) {
	encryptor := NewAESEncryptor()
	key, _ := encryptor.GenerateKey()

	// Test encrypting empty data
	emptyData := []byte{}
	encrypted, err := encryptor.Encrypt(emptyData, key)
	if err != nil {
		t.Fatalf("Encrypt() failed with empty data: %v", err)
	}

	decrypted, err := encryptor.Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() failed with empty data: %v", err)
	}

	if !bytes.Equal(emptyData, decrypted) {
		t.Error("Empty data encryption/decryption failed")
	}
}

func TestLargeData(t *testing.T) {
	encryptor := NewAESEncryptor()
	key, _ := encryptor.GenerateKey()

	// Test with large data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	encrypted, err := encryptor.Encrypt(largeData, key)
	if err != nil {
		t.Fatalf("Encrypt() failed with large data: %v", err)
	}

	decrypted, err := encryptor.Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() failed with large data: %v", err)
	}

	if !bytes.Equal(largeData, decrypted) {
		t.Error("Large data encryption/decryption failed")
	}
}
