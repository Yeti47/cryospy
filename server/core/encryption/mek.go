package encryption

import "time"

type Mek struct {
	ID                     string    // Unique identifier for the MEK
	EncryptedEncryptionKey string    // MEK encrypted with key derived from a password (base 64 encoded)
	EncryptionKeySalt      string    // Salt used for deriving the encryption key (base 64 encoded)
	CreatedAt              time.Time // Timestamp when the MEK was created
	UpdatedAt              time.Time // Timestamp when the MEK was last updated
}
