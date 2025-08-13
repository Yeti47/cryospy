package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/encryption"
	"github.com/yeti47/cryospy/server/dashboard/sessions"
)

type AuthMiddleware struct {
	logger          logging.Logger
	mekService      encryption.MekService
	mekStoreFactory sessions.MekStoreFactory
}

func NewAuthMiddleware(logger logging.Logger, mekService encryption.MekService, mekStoreFactory sessions.MekStoreFactory) *AuthMiddleware {
	return &AuthMiddleware{
		logger:          logger,
		mekService:      mekService,
		mekStoreFactory: mekStoreFactory,
	}
}

func (m *AuthMiddleware) RequireAuth(c *gin.Context) {
	// Check if a MEK exists in the database. If not, redirect to setup.
	_, err := m.mekService.GetMek()
	if err != nil {
		if _, ok := err.(*encryption.MekNotFoundError); ok {
			m.logger.Info("No MEK found, redirecting to setup.")
			c.Redirect(http.StatusFound, "/auth/setup")
			c.Abort()
			return
		}
		m.logger.Error("Failed to get MEK for auth check", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Check if the user is authenticated (MEK in session)
	mekStore := m.mekStoreFactory(c)
	sessionMek, err := mekStore.GetMek()
	if err != nil || len(sessionMek) == 0 {
		m.logger.Info("User not authenticated, redirecting to login.")
		c.Redirect(http.StatusFound, "/auth/login")
		c.Abort()
		return
	}

	c.Next()
}

func (m *AuthMiddleware) RedirectIfAuth(c *gin.Context) {
	// If a MEK is in the session, the user is authenticated. Redirect to dashboard.
	mekStore := m.mekStoreFactory(c)
	sessionMek, err := mekStore.GetMek()
	if err == nil && len(sessionMek) > 0 {
		c.Redirect(http.StatusFound, "/")
		c.Abort()
		return
	}

	c.Next()
}
