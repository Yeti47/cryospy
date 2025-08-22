package handlers

import (
	"context"
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/videos"
	"github.com/yeti47/cryospy/server/dashboard/sessions"
)

type ClientHandler struct {
	logger          logging.Logger
	clientService   clients.ClientService
	storageManager  videos.StorageManager
	mekStoreFactory sessions.MekStoreFactory
}

func NewClientHandler(logger logging.Logger, clientService clients.ClientService, storageManager videos.StorageManager, mekStoreFactory sessions.MekStoreFactory) *ClientHandler {
	return &ClientHandler{
		logger:          logger,
		clientService:   clientService,
		storageManager:  storageManager,
		mekStoreFactory: mekStoreFactory,
	}
}

func (h *ClientHandler) ListClients(c *gin.Context) {
	clientList, err := h.clientService.GetClients()
	if err != nil {
		h.logger.Error("Failed to get clients", err)
		c.HTML(http.StatusInternalServerError, "clients", gin.H{
			"Title": "Clients",
			"Error": "Failed to load clients.",
		})
		return
	}

	// Create a structure to hold client and storage info
	type ClientWithStorage struct {
		*clients.Client
		StorageInfo *videos.StorageInfo
	}

	clientsWithStorage := make([]ClientWithStorage, len(clientList))
	for i, client := range clientList {
		storageInfo, err := h.storageManager.GetStorageInfo(context.Background(), client.ID)
		if err != nil {
			h.logger.Warn("Failed to get storage info for client", "client_id", client.ID, "error", err)
			// Continue with nil storage info - we'll handle this in the template
			storageInfo = nil
		}
		clientsWithStorage[i] = ClientWithStorage{
			Client:      client,
			StorageInfo: storageInfo,
		}
	}

	c.HTML(http.StatusOK, "clients", gin.H{
		"Title":                  "Clients",
		"Clients":                clientsWithStorage,
		"SupportedResolutions":   h.clientService.GetSupportedDownscaleResolutions(),
		"SupportedCaptureCodecs": h.clientService.GetSupportedCaptureCodecs(),
		"SupportedOutputCodecs":  h.clientService.GetSupportedOutputCodecs(),
		"SupportedOutputFormats": h.clientService.GetSupportedOutputFormats(),
		"SupportedVideoBitrates": h.clientService.GetSupportedVideoBitrates(),
	})
}

func (h *ClientHandler) ShowNewClientForm(c *gin.Context) {
	c.HTML(http.StatusOK, "new-client", gin.H{
		"Title":                  "New Client",
		"SupportedResolutions":   h.clientService.GetSupportedDownscaleResolutions(),
		"SupportedCaptureCodecs": h.clientService.GetSupportedCaptureCodecs(),
		"SupportedOutputCodecs":  h.clientService.GetSupportedOutputCodecs(),
		"SupportedOutputFormats": h.clientService.GetSupportedOutputFormats(),
		"SupportedVideoBitrates": h.clientService.GetSupportedVideoBitrates(),
	})
}

func (h *ClientHandler) CreateClient(c *gin.Context) {
	id := c.PostForm("id")
	storageLimitStr := c.PostForm("storage_limit")
	clipDurationStr := c.PostForm("clip_duration")
	motionOnly := c.PostForm("motion_only") == "on"
	grayscale := c.PostForm("grayscale") == "on"
	downscaleResolution := c.PostForm("downscale_resolution")
	outputFormat := c.PostForm("output_format")
	outputCodec := c.PostForm("output_codec")
	videoBitRate := c.PostForm("video_bitrate")
	motionMinAreaStr := c.PostForm("motion_min_area")
	motionMaxFramesStr := c.PostForm("motion_max_frames")
	motionWarmUpFramesStr := c.PostForm("motion_warm_up_frames")
	captureCodec := c.PostForm("capture_codec")
	captureFrameRateStr := c.PostForm("capture_frame_rate")

	if id == "" || storageLimitStr == "" || clipDurationStr == "" {
		c.HTML(http.StatusBadRequest, "new-client", gin.H{
			"Title": "New Client",
			"Error": "All fields are required.",
		})
		return
	}

	storageLimit, err := strconv.Atoi(storageLimitStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "new-client", gin.H{
			"Title": "New Client",
			"Error": "Invalid storage limit.",
		})
		return
	}

	clipDuration, err := strconv.Atoi(clipDurationStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "new-client", gin.H{
			"Title": "New Client",
			"Error": "Invalid clip duration.",
		})
		return
	}

	motionMinArea, err := strconv.Atoi(motionMinAreaStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "new-client", gin.H{
			"Title": "New Client",
			"Error": "Invalid motion min area.",
		})
		return
	}

	motionMaxFrames, err := strconv.Atoi(motionMaxFramesStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "new-client", gin.H{
			"Title": "New Client",
			"Error": "Invalid motion max frames.",
		})
		return
	}

	motionWarmUpFrames, err := strconv.Atoi(motionWarmUpFramesStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "new-client", gin.H{
			"Title": "New Client",
			"Error": "Invalid motion warm up frames.",
		})
		return
	}

	captureFrameRate, err := strconv.ParseFloat(captureFrameRateStr, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "new-client", gin.H{
			"Title": "New Client",
			"Error": "Invalid capture frame rate.",
		})
		return
	}

	req := clients.CreateClientRequest{
		ID:                    id,
		StorageLimitMegabytes: storageLimit,
		ClipDurationSeconds:   clipDuration,
		MotionOnly:            motionOnly,
		Grayscale:             grayscale,
		DownscaleResolution:   downscaleResolution,
		OutputFormat:          outputFormat,
		OutputCodec:           outputCodec,
		VideoBitRate:          videoBitRate,
		MotionMinArea:         motionMinArea,
		MotionMaxFrames:       motionMaxFrames,
		MotionWarmUpFrames:    motionWarmUpFrames,
		CaptureCodec:          captureCodec,
		CaptureFrameRate:      captureFrameRate,
	}

	mekStore := h.mekStoreFactory(c)
	client, secret, err := h.clientService.CreateClient(req, mekStore)
	if err != nil {
		if clients.IsClientValidationError(err) {
			c.HTML(http.StatusBadRequest, "new-client", gin.H{
				"Title":                  "New Client",
				"Error":                  err.Error(),
				"SupportedResolutions":   h.clientService.GetSupportedDownscaleResolutions(),
				"SupportedCaptureCodecs": h.clientService.GetSupportedCaptureCodecs(),
				"SupportedOutputCodecs":  h.clientService.GetSupportedOutputCodecs(),
				"SupportedOutputFormats": h.clientService.GetSupportedOutputFormats(),
				"SupportedVideoBitrates": h.clientService.GetSupportedVideoBitrates(),
			})
			return
		}
		h.logger.Error("Failed to create client", err)
		c.HTML(http.StatusInternalServerError, "new-client", gin.H{
			"Title": "New Client",
			"Error": "Failed to create client.",
		})
		return
	}

	c.HTML(http.StatusOK, "new-client", gin.H{
		"Title":  "New Client",
		"Client": client,
		"Secret": hex.EncodeToString(secret),
	})
}

func (h *ClientHandler) UpdateClientSettings(c *gin.Context) {
	id := c.Param("id")
	storageLimitStr := c.PostForm("storage_limit")
	clipDurationStr := c.PostForm("clip_duration")
	motionOnly := c.PostForm("motion_only") == "on"
	grayscale := c.PostForm("grayscale") == "on"
	downscaleResolution := c.PostForm("downscale_resolution")
	outputFormat := c.PostForm("output_format")
	outputCodec := c.PostForm("output_codec")
	videoBitRate := c.PostForm("video_bitrate")
	motionMinAreaStr := c.PostForm("motion_min_area")
	motionMaxFramesStr := c.PostForm("motion_max_frames")
	motionWarmUpFramesStr := c.PostForm("motion_warm_up_frames")
	captureCodec := c.PostForm("capture_codec")
	captureFrameRateStr := c.PostForm("capture_frame_rate")

	if id == "" || clipDurationStr == "" || storageLimitStr == "" {
		c.HTML(http.StatusBadRequest, "clients", gin.H{
			"Title": "Clients",
			"Error": "All fields are required.",
		})
		return
	}

	storageLimit, err := strconv.Atoi(storageLimitStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "clients", gin.H{
			"Title": "Clients",
			"Error": "Invalid storage limit.",
		})
		return
	}

	clipDuration, err := strconv.Atoi(clipDurationStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "clients", gin.H{
			"Title": "Clients",
			"Error": "Invalid clip duration.",
		})
		return
	}

	motionMinArea, err := strconv.Atoi(motionMinAreaStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "clients", gin.H{
			"Title": "Clients",
			"Error": "Invalid motion min area.",
		})
		return
	}

	motionMaxFrames, err := strconv.Atoi(motionMaxFramesStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "clients", gin.H{
			"Title": "Clients",
			"Error": "Invalid motion max frames.",
		})
		return
	}

	motionWarmUpFrames, err := strconv.Atoi(motionWarmUpFramesStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "clients", gin.H{
			"Title": "Clients",
			"Error": "Invalid motion warm up frames.",
		})
		return
	}

	captureFrameRate, err := strconv.ParseFloat(captureFrameRateStr, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "clients", gin.H{
			"Title": "Clients",
			"Error": "Invalid capture frame rate.",
		})
		return
	}

	req := clients.UpdateClientSettingsRequest{
		ID:                    id,
		StorageLimitMegabytes: storageLimit,
		ClipDurationSeconds:   clipDuration,
		MotionOnly:            motionOnly,
		Grayscale:             grayscale,
		DownscaleResolution:   downscaleResolution,
		OutputFormat:          outputFormat,
		OutputCodec:           outputCodec,
		VideoBitRate:          videoBitRate,
		MotionMinArea:         motionMinArea,
		MotionMaxFrames:       motionMaxFrames,
		MotionWarmUpFrames:    motionWarmUpFrames,
		CaptureCodec:          captureCodec,
		CaptureFrameRate:      captureFrameRate,
	}

	err = h.clientService.UpdateClientSettings(req)
	if err != nil {
		if clients.IsClientValidationError(err) {
			clientList, _ := h.clientService.GetClients()
			c.HTML(http.StatusBadRequest, "clients", gin.H{
				"Title":                  "Clients",
				"Error":                  err.Error(),
				"Clients":                clientList,
				"SupportedResolutions":   h.clientService.GetSupportedDownscaleResolutions(),
				"SupportedCaptureCodecs": h.clientService.GetSupportedCaptureCodecs(),
				"SupportedOutputCodecs":  h.clientService.GetSupportedOutputCodecs(),
				"SupportedOutputFormats": h.clientService.GetSupportedOutputFormats(),
				"SupportedVideoBitrates": h.clientService.GetSupportedVideoBitrates(),
			})
			return
		}
		h.logger.Error("Failed to update client settings", err)
		c.HTML(http.StatusInternalServerError, "clients", gin.H{
			"Title": "Clients",
			"Error": "Failed to update client settings.",
		})
		return
	}

	c.Redirect(http.StatusFound, "/clients")
}

func (h *ClientHandler) DeleteClient(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.HTML(http.StatusBadRequest, "clients", gin.H{
			"Title": "Clients",
			"Error": "Client ID is required.",
		})
		return
	}

	if err := h.clientService.DeleteClient(id); err != nil {
		h.logger.Error("Failed to delete client", err)
		c.HTML(http.StatusInternalServerError, "clients", gin.H{
			"Title": "Clients",
			"Error": "Failed to delete client.",
		})
		return
	}

	c.Redirect(http.StatusFound, "/clients")
}

func (h *ClientHandler) DisableClient(c *gin.Context) {
	id := c.Param("id")

	if err := h.clientService.DisableClient(id); err != nil {
		h.logger.Error("Failed to disable client", err)
		c.HTML(http.StatusInternalServerError, "clients", gin.H{
			"Title": "Clients",
			"Error": "Failed to disable client.",
		})
		return
	}

	h.logger.Info("Client disabled", "clientId", id)
	c.Redirect(http.StatusFound, "/clients")
}

func (h *ClientHandler) EnableClient(c *gin.Context) {
	id := c.Param("id")

	if err := h.clientService.EnableClient(id); err != nil {
		h.logger.Error("Failed to enable client", err)
		c.HTML(http.StatusInternalServerError, "clients", gin.H{
			"Title": "Clients",
			"Error": "Failed to enable client.",
		})
		return
	}

	h.logger.Info("Client enabled", "clientId", id)
	c.Redirect(http.StatusFound, "/clients")
}
