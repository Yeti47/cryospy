package handlers

import (
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/dashboard/sessions"
)

type ClientHandler struct {
	logger          logging.Logger
	clientService   clients.ClientService
	mekStoreFactory sessions.MekStoreFactory
}

func NewClientHandler(logger logging.Logger, clientService clients.ClientService, mekStoreFactory sessions.MekStoreFactory) *ClientHandler {
	return &ClientHandler{
		logger:          logger,
		clientService:   clientService,
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

	c.HTML(http.StatusOK, "clients", gin.H{
		"Title":                "Clients",
		"Clients":              clientList,
		"SupportedResolutions": h.clientService.GetSupportedDownscaleResolutions(),
	})
}

func (h *ClientHandler) ShowNewClientForm(c *gin.Context) {
	c.HTML(http.StatusOK, "new-client", gin.H{
		"Title":                "New Client",
		"SupportedResolutions": h.clientService.GetSupportedDownscaleResolutions(),
	})
}

func (h *ClientHandler) CreateClient(c *gin.Context) {
	id := c.PostForm("id")
	storageLimitStr := c.PostForm("storage_limit")
	clipDurationStr := c.PostForm("clip_duration")
	motionOnly := c.PostForm("motion_only") == "on"
	grayscale := c.PostForm("grayscale") == "on"
	downscaleResolution := c.PostForm("downscale_resolution")

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

	req := clients.CreateClientRequest{
		ID:                    id,
		StorageLimitMegabytes: storageLimit,
		ClipDurationSeconds:   clipDuration,
		MotionOnly:            motionOnly,
		Grayscale:             grayscale,
		DownscaleResolution:   downscaleResolution,
	}

	mekStore := h.mekStoreFactory(c)
	client, secret, err := h.clientService.CreateClient(req, mekStore)
	if err != nil {
		if clients.IsClientValidationError(err) {
			c.HTML(http.StatusBadRequest, "new-client", gin.H{
				"Title":                "New Client",
				"Error":                err.Error(),
				"SupportedResolutions": h.clientService.GetSupportedDownscaleResolutions(),
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

	req := clients.UpdateClientSettingsRequest{
		ID:                    id,
		StorageLimitMegabytes: storageLimit,
		ClipDurationSeconds:   clipDuration,
		MotionOnly:            motionOnly,
		Grayscale:             grayscale,
		DownscaleResolution:   downscaleResolution,
	}

	err = h.clientService.UpdateClientSettings(req)
	if err != nil {
		if clients.IsClientValidationError(err) {
			clientList, _ := h.clientService.GetClients()
			c.HTML(http.StatusBadRequest, "clients", gin.H{
				"Title":                "Clients",
				"Error":                err.Error(),
				"Clients":              clientList,
				"SupportedResolutions": h.clientService.GetSupportedDownscaleResolutions(),
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
