package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
)

// ClientHandler handles client-related operations
type ClientHandler struct {
	logger        logging.Logger
	clientService clients.ClientService
}

// NewClientHandler creates a new client handler
func NewClientHandler(logger logging.Logger, clientService clients.ClientService) *ClientHandler {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &ClientHandler{
		logger:        logger,
		clientService: clientService,
	}
}

// ClientSettingsResponse represents the client settings response
type ClientSettingsResponse struct {
	ID                    string `json:"id"`
	StorageLimitMegabytes int    `json:"storage_limit_megabytes"`
	ClipDurationSeconds   int    `json:"clip_duration_seconds"`
	MotionOnly            bool   `json:"motion_only"`
	Grayscale             bool   `json:"grayscale"`
	DownscaleResolution   string `json:"downscale_resolution"`
}

// GetClientSettings handles GET /api/client/settings
func (h *ClientHandler) GetClientSettings(c *gin.Context) {
	h.logger.Info("Received client settings request")

	// Get client information from middleware
	clientInterface, exists := c.Get("client")
	if !exists {
		h.logger.Error("Client not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	client, ok := clientInterface.(*clients.Client)
	if !ok {
		h.logger.Error("Invalid client type in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	h.logger.Info("Returning client settings", "clientID", client.ID)

	// Create response
	response := ClientSettingsResponse{
		ID:                    client.ID,
		StorageLimitMegabytes: client.StorageLimitMegabytes,
		ClipDurationSeconds:   client.ClipDurationSeconds,
		MotionOnly:            client.MotionOnly,
		Grayscale:             client.Grayscale,
		DownscaleResolution:   client.DownscaleResolution,
	}

	c.JSON(http.StatusOK, response)
}
