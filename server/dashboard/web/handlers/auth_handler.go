package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/encryption"
	"github.com/yeti47/cryospy/server/dashboard/sessions"
)

type AuthHandler struct {
	logger          logging.Logger
	mekService      encryption.MekService
	mekStoreFactory sessions.MekStoreFactory
}

func NewAuthHandler(logger logging.Logger, mekService encryption.MekService, mekStoreFactory sessions.MekStoreFactory) *AuthHandler {
	return &AuthHandler{
		logger:          logger,
		mekService:      mekService,
		mekStoreFactory: mekStoreFactory,
	}
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login", gin.H{
		"Title": "Login",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	password := c.PostForm("password")
	if password == "" {
		c.HTML(http.StatusBadRequest, "login", gin.H{
			"Title": "Login",
			"Error": "Password is required",
		})
		return
	}

	// This is a simplified login. We just need to decrypt the MEK with the password.
	// The MEK is not stored in the session yet. We need to get it from the database.
	mek, err := h.mekService.GetMek()
	if err != nil {
		h.logger.Error("Failed to get MEK during login", err)
		c.HTML(http.StatusInternalServerError, "login", gin.H{
			"Title": "Login",
			"Error": "An internal error occurred.",
		})
		return
	}

	// Decrypt the MEK to verify the password
	encryptor := encryption.NewAESEncryptor()
	decryptedMek, err := encryption.DecryptMek(mek, password, encryptor)
	if err != nil {
		h.logger.Warn("Failed login attempt", "error", err)
		c.HTML(http.StatusUnauthorized, "login", gin.H{
			"Title": "Login",
			"Error": "Invalid password",
		})
		return
	}

	// On success, store the DECRYPTED MEK in the session
	mekStore := h.mekStoreFactory(c)
	if err := mekStore.SetMek(decryptedMek); err != nil {
		h.logger.Error("Failed to set MEK in session", err)
		c.HTML(http.StatusInternalServerError, "login", gin.H{
			"Title": "Login",
			"Error": "Failed to start session.",
		})
		return
	}

	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	mekStore := h.mekStoreFactory(c)
	if err := mekStore.ClearMek(); err != nil {
		h.logger.Error("Failed to clear MEK from session", err)
		// Don't block logout, just log the error.
	}
	c.Redirect(http.StatusFound, "/auth/login")
}

func (h *AuthHandler) ShowSetup(c *gin.Context) {
	c.HTML(http.StatusOK, "setup", gin.H{
		"Title": "Setup",
	})
}

func (h *AuthHandler) Setup(c *gin.Context) {
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	if password == "" || confirmPassword == "" {
		c.HTML(http.StatusBadRequest, "setup", gin.H{
			"Title": "Setup",
			"Error": "All fields are required",
		})
		return
	}

	if password != confirmPassword {
		c.HTML(http.StatusBadRequest, "setup", gin.H{
			"Title": "Setup",
			"Error": "Passwords do not match",
		})
		return
	}

	// Create a new MEK
	mek, err := h.mekService.CreateMek(password)
	if err != nil {
		h.logger.Error("Failed to create MEK", err)
		c.HTML(http.StatusInternalServerError, "setup", gin.H{
			"Title": "Setup",
			"Error": "Failed to create encryption key.",
		})
		return
	}

	// Decrypt the MEK for session storage
	encryptor := encryption.NewAESEncryptor()
	decryptedMek, err := encryption.DecryptMek(mek, password, encryptor)
	if err != nil {
		h.logger.Error("Failed to decrypt newly created MEK", err)
		c.HTML(http.StatusInternalServerError, "setup", gin.H{
			"Title": "Setup",
			"Error": "Failed to initialize encryption.",
		})
		return
	}

	// Store the DECRYPTED MEK in the session to log the user in
	mekStore := h.mekStoreFactory(c)
	if err := mekStore.SetMek(decryptedMek); err != nil {
		h.logger.Error("Failed to set MEK in session after setup", err)
		c.HTML(http.StatusInternalServerError, "setup", gin.H{
			"Title": "Setup",
			"Error": "Failed to start session.",
		})
		return
	}

	c.Redirect(http.StatusFound, "/")
}
