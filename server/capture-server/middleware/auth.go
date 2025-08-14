package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
)

// AuthMiddleware provides client authentication middleware for Gin
type AuthMiddleware struct {
	logger   logging.Logger
	verifier clients.ClientVerifier
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(logger logging.Logger, verifier clients.ClientVerifier) *AuthMiddleware {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &AuthMiddleware{
		logger:   logger,
		verifier: verifier,
	}
}

// RequireAuth middleware that requires client authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract client ID and secret from Authorization header
		// Expected format: "Basic <base64(clientId:clientSecret)>"
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.logger.Warn("Missing Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
			c.Abort()
			return
		}

		// Check if it's Basic auth
		if !strings.HasPrefix(authHeader, "Basic ") {
			m.logger.Warn("Invalid Authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			c.Abort()
			return
		}

		// Extract and decode credentials
		clientID, clientSecret, ok := c.Request.BasicAuth()
		if !ok {
			m.logger.Warn("Failed to parse Basic Auth credentials")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials format"})
			c.Abort()
			return
		}

		// Verify client credentials
		valid, client, err := m.verifier.VerifyClient(clientID, clientSecret)
		if err != nil {
			// Check if it's a client verification error (authentication failure)
			if clients.IsClientVerificationError(err) {
				m.logger.Warn("Client verification failed", "clientID", clientID)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
				c.Abort()
				return
			}
			// Other errors are internal server errors
			m.logger.Error("Error verifying client", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication error"})
			c.Abort()
			return
		}

		if !valid {
			m.logger.Warn("Invalid client credentials", "clientID", clientID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			c.Abort()
			return
		}

		// Store client information in context
		c.Set("client", client)
		c.Set("clientID", clientID)
		c.Set("clientSecret", clientSecret)

		c.Next()
	}
}
