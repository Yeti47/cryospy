package handlers

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/capture-server/utils"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/videos"
)

// ClipHandler handles video clip upload operations
type ClipHandler struct {
	logger      logging.Logger
	clipCreator videos.ClipCreator
}

// NewClipHandler creates a new clip handler
func NewClipHandler(logger logging.Logger, clipCreator videos.ClipCreator) *ClipHandler {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &ClipHandler{
		logger:      logger,
		clipCreator: clipCreator,
	}
}

// UploadClipRequest represents the expected form data for clip upload
type UploadClipRequest struct {
	Timestamp string `form:"timestamp" binding:"required"`
	Duration  string `form:"duration" binding:"required"`
	HasMotion string `form:"has_motion"`
}

// UploadClip handles POST /api/clips
func (h *ClipHandler) UploadClip(c *gin.Context) {
	h.logger.Info("Received clip upload request")

	// Get client information from middleware
	clientID, exists := c.Get("clientID")
	if !exists {
		h.logger.Error("Client ID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	clientSecret, exists := c.Get("clientSecret")
	if !exists {
		h.logger.Error("Client secret not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	clientIDStr := clientID.(string)
	clientSecretStr := clientSecret.(string)

	// Parse form data
	var req UploadClipRequest
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Warn("Invalid form data", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data: " + err.Error()})
		return
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		h.logger.Warn("Invalid timestamp format", "timestamp", req.Timestamp, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format. Expected RFC3339 format"})
		return
	}

	// Parse duration (expects seconds as float)
	durationSeconds, err := strconv.ParseFloat(req.Duration, 64)
	if err != nil {
		h.logger.Warn("Invalid duration format", "duration", req.Duration, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid duration format. Expected number of seconds"})
		return
	}

	if durationSeconds <= 0 {
		h.logger.Warn("Invalid duration value", "duration", durationSeconds)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Duration must be greater than 0"})
		return
	}

	duration := time.Duration(durationSeconds * float64(time.Second))

	// Parse has_motion (optional, defaults to false)
	hasMotion := false
	if req.HasMotion != "" {
		hasMotion, err = strconv.ParseBool(req.HasMotion)
		if err != nil {
			h.logger.Warn("Invalid has_motion format", "has_motion", req.HasMotion, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid has_motion format. Expected boolean"})
			return
		}
	}

	// Get uploaded file
	fileHeader, err := c.FormFile("video")
	if err != nil {
		h.logger.Warn("Failed to get uploaded file", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Video file is required"})
		return
	}

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error("Failed to open uploaded file", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process uploaded file"})
		return
	}
	defer file.Close()

	// Read file data
	videoData, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read uploaded file", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read uploaded file"})
		return
	}

	// Validate that it's a video file
	isVideo, format, err := utils.IsVideoFile(videoData)
	if err != nil {
		h.logger.Warn("Failed to validate file type", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to validate file type"})
		return
	}

	if !isVideo {
		h.logger.Warn("Uploaded file is not a video", "filename", fileHeader.Filename)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Uploaded file is not a valid video format"})
		return
	}

	h.logger.Info("Video file validated", "format", format, "size", len(videoData), "filename", fileHeader.Filename)

	// Create clip request
	createReq := videos.CreateClipRequest{
		TimeStamp: timestamp,
		Duration:  duration,
		HasMotion: hasMotion,
		Video:     videoData,
	}

	// Create the clip
	clip, err := h.clipCreator.CreateClip(createReq, clientIDStr, clientSecretStr)
	if err != nil {
		h.logger.Error("Failed to create clip", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create clip"})
		return
	}

	h.logger.Info("Successfully created clip", "clipID", clip.ID, "clientID", clientIDStr)

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"message": "Clip uploaded successfully",
		"clip_id": clip.ID,
		"title":   clip.Title,
	})
}
