package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

const sessionKeyFileName = "cryospy_session.txt"
const sessionKeyLength = 64 // 64 bytes for a strong key

// GetOrCreateSessionKey retrieves the session key from a file in the user's home directory.
// If the file doesn't exist, it generates a new key, saves it, and returns it.
func GetOrCreateSessionKey() ([]byte, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	keyPath := filepath.Join(homeDir, "cryospy", sessionKeyFileName)

	// Try to read the key from the file
	key, err := os.ReadFile(keyPath)
	if err == nil {
		// Key file exists, decode it from base64
		decodedKey, err := base64.StdEncoding.DecodeString(string(key))
		if err != nil {
			return nil, fmt.Errorf("failed to decode existing session key: %w", err)
		}
		return decodedKey, nil
	}

	// If the file does not exist, create a new key
	if os.IsNotExist(err) {
		newKey := make([]byte, sessionKeyLength)
		if _, err := rand.Read(newKey); err != nil {
			return nil, fmt.Errorf("failed to generate new session key: %w", err)
		}

		// Encode the key to base64 to store as a string
		encodedKey := base64.StdEncoding.EncodeToString(newKey)

		// Write the new key to the file
		if err := os.WriteFile(keyPath, []byte(encodedKey), 0600); err != nil {
			return nil, fmt.Errorf("failed to save new session key: %w", err)
		}

		return newKey, nil
	}

	// For any other error, return it
	return nil, fmt.Errorf("failed to read session key file: %w", err)
}
