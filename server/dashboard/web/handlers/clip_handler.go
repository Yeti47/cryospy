package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/videos"
	"github.com/yeti47/cryospy/server/dashboard/sessions"
)

const defaultPageSize = 20

type ClipHandler struct {
	logger          logging.Logger
	clipReader      videos.ClipReader
	mekStoreFactory sessions.MekStoreFactory
}

func NewClipHandler(logger logging.Logger, clipReader videos.ClipReader, mekStoreFactory sessions.MekStoreFactory) *ClipHandler {
	return &ClipHandler{
		logger:          logger,
		clipReader:      clipReader,
		mekStoreFactory: mekStoreFactory,
	}
}

func (h *ClipHandler) ListClips(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", strconv.Itoa(defaultPageSize)))

	query := videos.ClipQuery{
		Page:     page,
		PageSize: pageSize,
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

	c.HTML(http.StatusOK, "clips", gin.H{
		"Title":      "Clips",
		"Clips":      clips,
		"Total":      total,
		"Page":       page,
		"PageSize":   pageSize,
		"TotalPages": (total + pageSize - 1) / pageSize,
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
