package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/videos"
	"github.com/yeti47/cryospy/server/dashboard/sessions"
)

const defaultPageSize = 20

type ClipHandler struct {
	logger          logging.Logger
	clipReader      videos.ClipReader
	clientService   clients.ClientService
	mekStoreFactory sessions.MekStoreFactory
}

func NewClipHandler(logger logging.Logger, clipReader videos.ClipReader, clientService clients.ClientService, mekStoreFactory sessions.MekStoreFactory) *ClipHandler {
	return &ClipHandler{
		logger:          logger,
		clipReader:      clipReader,
		clientService:   clientService,
		mekStoreFactory: mekStoreFactory,
	}
}

func (h *ClipHandler) ListClips(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", strconv.Itoa(defaultPageSize)))

	// Parse filter parameters
	query := videos.ClipQuery{
		Page:     page,
		PageSize: pageSize,
	}

	// ClientID filter
	if clientID := c.Query("clientId"); clientID != "" {
		query.ClientID = clientID
	}

	// Start time filter
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse("2006-01-02T15:04", startTimeStr); err == nil {
			query.StartTime = &startTime
		}
	}

	// End time filter
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse("2006-01-02T15:04", endTimeStr); err == nil {
			query.EndTime = &endTime
		}
	}

	// Motion filter
	if hasMotionStr := c.Query("hasMotion"); hasMotionStr != "" {
		if hasMotion, err := strconv.ParseBool(hasMotionStr); err == nil {
			query.HasMotion = &hasMotion
		}
	}

	mekStore := h.mekStoreFactory(c)
	clips, total, err := h.clipReader.QueryClips(query, mekStore)
	if err != nil {
		h.logger.Error("Failed to query clips", err)
		c.HTML(http.StatusInternalServerError, "clips", gin.H{
			"Title": "Clips",
			"Error": "Failed to load clips.",
		})
		return
	}

	// Get clients for filter dropdown
	clientList, err := h.clientService.GetClients()
	if err != nil {
		h.logger.Error("Failed to get clients", err)
		// Don't fail completely, just log the error and continue without clients
		clientList = []*clients.Client{}
	}

	// Prepare current filter values for the template
	filterValues := gin.H{
		"ClientID":  c.Query("clientId"),
		"StartTime": c.Query("startTime"),
		"EndTime":   c.Query("endTime"),
		"HasMotion": c.Query("hasMotion"),
	}

	c.HTML(http.StatusOK, "clips", gin.H{
		"Title":        "Clips",
		"Clips":        clips,
		"Total":        total,
		"Page":         page,
		"PageSize":     pageSize,
		"TotalPages":   (total + pageSize - 1) / pageSize,
		"Clients":      clientList,
		"FilterValues": filterValues,
	})
}

func (h *ClipHandler) GetThumbnail(c *gin.Context) {
	clipID := c.Param("id")
	if clipID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	mekStore := h.mekStoreFactory(c)
	thumbnail, err := h.clipReader.GetClipThumbnail(clipID, mekStore)
	if err != nil {
		h.logger.Error("Failed to get thumbnail", err, "clipID", clipID)
		c.Status(http.StatusInternalServerError)
		return
	}

	if thumbnail == nil || len(thumbnail.Data) == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	c.Data(http.StatusOK, thumbnail.MimeType, thumbnail.Data)
}

func (h *ClipHandler) ViewClip(c *gin.Context) {
	clipID := c.Param("id")
	if clipID == "" {
		c.HTML(http.StatusBadRequest, "clip-detail", gin.H{
			"Title": "Clip Detail",
			"Error": "Clip ID is required.",
		})
		return
	}

	clipInfo, err := h.clipReader.GetClipInfoByID(clipID)
	if err != nil {
		h.logger.Error("Failed to get clip info", err, "clipID", clipID)
		c.HTML(http.StatusInternalServerError, "clip-detail", gin.H{
			"Title": "Clip Detail",
			"Error": "Failed to load clip.",
		})
		return
	}

	if clipInfo == nil {
		c.HTML(http.StatusNotFound, "clip-detail", gin.H{
			"Title": "Clip Detail",
			"Error": "Clip not found.",
		})
		return
	}

	c.HTML(http.StatusOK, "clip-detail", gin.H{
		"Title": "Clip Detail - " + clipInfo.Title,
		"Clip":  clipInfo,
	})
}

func (h *ClipHandler) GetVideo(c *gin.Context) {
	clipID := c.Param("id")
	if clipID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	mekStore := h.mekStoreFactory(c)
	clip, err := h.clipReader.GetClipByID(clipID, mekStore)
	if err != nil {
		h.logger.Error("Failed to get video", err, "clipID", clipID)
		c.Status(http.StatusInternalServerError)
		return
	}

	if clip == nil || len(clip.Video) == 0 {
		h.logger.Warn("Video not found or empty", "clipID", clipID)
		c.Status(http.StatusNotFound)
		return
	}

	h.logger.Debug("Serving video", "clipID", clipID, "mimeType", clip.VideoMimeType, "size", len(clip.Video))

	// Set appropriate headers for video streaming
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", strconv.Itoa(len(clip.Video)))
	c.Data(http.StatusOK, clip.VideoMimeType, clip.Video)
}
