package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/streaming"
	"github.com/yeti47/cryospy/server/dashboard/sessions"
)

type StreamHandler struct {
	logger           logging.Logger
	streamingService streaming.StreamingService
	clientService    clients.ClientService
	mekStoreFactory  sessions.MekStoreFactory
}

func NewStreamHandler(logger logging.Logger, streamingService streaming.StreamingService, clientService clients.ClientService, mekStoreFactory sessions.MekStoreFactory) *StreamHandler {
	return &StreamHandler{
		logger:           logger,
		streamingService: streamingService,
		clientService:    clientService,
		mekStoreFactory:  mekStoreFactory,
	}
}

// ShowStreamSelection displays the stream selection page
func (h *StreamHandler) ShowStreamSelection(c *gin.Context) {
	// Get all clients for selection
	clients, err := h.clientService.GetClients()
	if err != nil {
		h.logger.Error("Failed to get clients", err)
		c.HTML(http.StatusInternalServerError, "error", gin.H{
			"Title":   "Error",
			"Message": "Failed to load clients",
		})
		return
	}

	c.HTML(http.StatusOK, "stream-selection", gin.H{
		"Title":   "Stream Selection",
		"Clients": clients,
	})
}

// ShowStream displays the streaming page for a specific client
func (h *StreamHandler) ShowStream(c *gin.Context) {
	clientID := c.Param("clientId")
	if clientID == "" {
		c.HTML(http.StatusBadRequest, "error", gin.H{
			"Title":   "Error",
			"Message": "Client ID is required",
		})
		return
	}

	// Verify client exists
	client, err := h.clientService.GetClient(clientID)
	if err != nil {
		h.logger.Error("Failed to get client", err)
		c.HTML(http.StatusNotFound, "error", gin.H{
			"Title":   "Error",
			"Message": "Client not found",
		})
		return
	}

	// Get start time and reference time from query parameters
	startTime := time.Now()
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = parsed
		}
	}

	refTime := time.Now()
	if refTimeStr := c.Query("refTime"); refTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, refTimeStr); err == nil {
			refTime = parsed
		}
	}

	c.HTML(http.StatusOK, "stream", gin.H{
		"Title":     "Stream - " + client.ID,
		"Client":    client,
		"StartTime": startTime.Format(time.RFC3339),
		"RefTime":   refTime.Format(time.RFC3339),
	})
}

// GetPlaylist serves the HLS playlist for a client
func (h *StreamHandler) GetPlaylist(c *gin.Context) {
	clientID := c.Param("clientId")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID is required"})
		return
	}

	// Parse query parameters
	startTimeStr := c.Query("startTime")
	refTimeStr := c.Query("refTime")

	if startTimeStr == "" || refTimeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "startTime and refTime parameters are required"})
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid startTime format"})
		return
	}

	refTime, err := time.Parse(time.RFC3339, refTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid refTime format"})
		return
	}

	h.logger.Debug("Generating playlist", "clientID", clientID, "startTime", startTime, "refTime", refTime)

	// Generate playlist
	playlist, err := h.streamingService.GetPlaylist(clientID, startTime, refTime)
	if err != nil {
		h.logger.Error("Failed to generate playlist", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate playlist"})
		return
	}

	h.logger.Debug("Generated playlist", "clientID", clientID, "playlistLength", len(playlist))

	// Set appropriate headers for HLS
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "no-cache")
	c.String(http.StatusOK, playlist)
}

// GetSegment serves a video segment for streaming
func (h *StreamHandler) GetSegment(c *gin.Context) {
	clientID := c.Param("clientId")
	clipID := c.Param("clipId")

	h.logger.Debug("Segment request", "method", c.Request.Method, "url", c.Request.URL.String(), "clientID", clientID, "clipID", clipID)

	if clientID == "" || clipID == "" {
		h.logger.Error("Missing required parameters", nil, "clientID", clientID, "clipID", clipID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID and clip ID are required"})
		return
	}

	h.logger.Debug("Getting segment", "clientID", clientID, "clipID", clipID)

	mekStore := h.mekStoreFactory(c)

	// Get video segment
	segmentData, err := h.streamingService.GetSegment(clientID, clipID, mekStore)
	if err != nil {
		h.logger.Error("Failed to get video segment", err, "clientID", clientID, "clipID", clipID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Segment not found"})
		return
	}

	h.logger.Debug("Serving segment", "clientID", clientID, "clipID", clipID, "segmentSize", len(segmentData))

	// Set appropriate headers for video content
	c.Header("Content-Type", "video/mp2t")
	c.Header("Cache-Control", "public, max-age=3600") // Cache segments for 1 hour
	c.Header("Content-Length", strconv.Itoa(len(segmentData)))

	c.Data(http.StatusOK, "video/mp2t", segmentData)
}
