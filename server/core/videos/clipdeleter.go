package videos

import (
	"context"
	"errors"
	"fmt"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

type DeleteClipsRequest struct {
	ClipIDs []string `json:"clip_ids"`
}

type DeleteClipsResponse struct {
	DeletedClips []string `json:"deleted_clips"`
	FailedClips  []string `json:"failed_clips"`
	Errors       []string `json:"errors"`
}

type ClipDeleter interface {
	// DeleteClips deletes one or more video clips by their IDs
	// Returns information about which clips were successfully deleted and which failed
	DeleteClips(req DeleteClipsRequest) (*DeleteClipsResponse, error)
}

type clipDeleter struct {
	logger   logging.Logger
	clipRepo ClipRepository
}

func NewClipDeleter(logger logging.Logger, clipRepo ClipRepository) *clipDeleter {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &clipDeleter{
		logger:   logger,
		clipRepo: clipRepo,
	}
}

func (d *clipDeleter) DeleteClips(req DeleteClipsRequest) (*DeleteClipsResponse, error) {
	// Validate request
	if len(req.ClipIDs) == 0 {
		return nil, errors.New("no clip IDs provided")
	}

	response := &DeleteClipsResponse{
		DeletedClips: make([]string, 0),
		FailedClips:  make([]string, 0),
		Errors:       make([]string, 0),
	}

	ctx := context.Background()

	// Process each clip ID
	for _, clipID := range req.ClipIDs {
		// Attempt to delete the clip directly
		err := d.clipRepo.Delete(ctx, clipID)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to delete clip %s: %v", clipID, err)
			d.logger.Error("Failed to delete clip", err, "clip_id", clipID)
			response.FailedClips = append(response.FailedClips, clipID)
			response.Errors = append(response.Errors, errorMsg)
			continue
		}

		// Success - clip was found and deleted
		response.DeletedClips = append(response.DeletedClips, clipID)
		d.logger.Info("Successfully deleted clip", "clip_id", clipID)
	}

	d.logger.Info("Clip deletion completed", "requested", len(req.ClipIDs), "deleted", len(response.DeletedClips), "failed", len(response.FailedClips))
	return response, nil
}
