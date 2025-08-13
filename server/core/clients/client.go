package clients

import "time"

type Client struct {
	ID                    string    // Unique identifier for the client
	SecretHash            string    // Hashed secret for authentication (base 64 encoded)
	SecretSalt            string    // Salt used for hashing the secret (base 64 encoded)
	CreatedAt             time.Time // Timestamp when the client was created
	UpdatedAt             time.Time // Timestamp when the client was last updated
	EncryptedMek          string    // MEK encrypted with key derived from client secret (base 64 encoded)
	KeyDerivationSalt     string    // Salt used for deriving encryption key from secret (base 64 encoded)
	StorageLimitMegabytes int       // Storage limit in megabytes
	ClipDurationSeconds   int       // The duration in seconds (integer) of each clip that the client captures
	MotionOnly            bool      // A flag that describes whether the client should only upload video clips in which motion was detected
	Grayscale             bool      // A flag that describes whether clips should be optimized to use grayscale to reduce size
	DownscaleResolution   string    // Optional resolution to which captured video clips should be downscaled (on the client side). Example value: "360p", "480p", "720p"...
}
