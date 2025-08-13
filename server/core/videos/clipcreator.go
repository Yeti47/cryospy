package videos

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/encryption"
)

type CreateClipRequest struct {
	TimeStamp time.Time     `json:"time_stamp"`
	Duration  time.Duration `json:"duration"`
	HasMotion bool          `json:"has_motion"`
	Video     []byte        `json:"video"`
}

type ClipCreator interface {
	// CreateClip creates a new video clip with the given details
	CreateClip(req CreateClipRequest, clientID, clientSecret string) (*Clip, error)
}

type clipCreator struct {
	logger             logging.Logger
	storageManager     StorageManager
	encryptor          encryption.Encryptor
	mekProvider        clients.ClientMekProvider
	metadataExtractor  VideoMetadataExtractor
	thumbnailGenerator ThumbnailGenerator
}

func NewClipCreator(logger logging.Logger, storageManager StorageManager, encryptor encryption.Encryptor, mekProvider clients.ClientMekProvider, metadataExtractor VideoMetadataExtractor, thumbnailGenerator ThumbnailGenerator) *clipCreator {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &clipCreator{
		logger:             logger,
		storageManager:     storageManager,
		encryptor:          encryptor,
		mekProvider:        mekProvider,
		metadataExtractor:  metadataExtractor,
		thumbnailGenerator: thumbnailGenerator,
	}
}

func (s *clipCreator) CreateClip(req CreateClipRequest, clientID, clientSecret string) (*Clip, error) {
	// Validate request
	if req.Duration <= 0 {
		return nil, errors.New("invalid duration")
	}
	if len(req.Video) == 0 {
		return nil, errors.New("video data is required")
	}

	// Uncover the MEK for the client (verify client first before expensive operations)
	mek, err := s.mekProvider.UncoverMek(clientID, clientSecret)
	if err != nil {
		s.logger.Error("Failed to uncover MEK", err)
		return nil, err
	}

	// Extract video metadata (expensive operation, only after client verification)
	videoMeta, err := s.metadataExtractor.ExtractMetadata(req.Video)
	if err != nil {
		s.logger.Error("Failed to extract video metadata", err)
		return nil, err
	}

	// Generate UUID for clip ID
	clipID := uuid.New().String()

	// Create title in the specified format
	timestampUtc := req.TimeStamp.UTC().Format("2006-01-02T15-04-05")
	durationSeconds := fmt.Sprintf("%.0f", req.Duration.Seconds())
	motionStr := "nomotion"
	if req.HasMotion {
		motionStr = "motion"
	}

	title := fmt.Sprintf("%s_%ss_%s.%s", timestampUtc, durationSeconds, motionStr, videoMeta.Extension)

	// Encrypt video data
	encryptedVideo, err := s.encryptor.Encrypt(req.Video, mek)
	if err != nil {
		s.logger.Error("Failed to encrypt video", err)
		return nil, err
	}

	// Extract thumbnail from video
	thumbnail, err := s.thumbnailGenerator.GenerateThumbnail(req.Video, videoMeta)
	if err != nil {
		s.logger.Warn("Failed to extract thumbnail, proceeding without thumbnail", err)
		// Continue without thumbnail
		thumbnail = nil
	}

	// Encrypt thumbnail if one was extracted
	var encryptedThumbnail []byte
	var thumbnailWidth, thumbnailHeight int
	var thumbnailMimeType string

	if thumbnail != nil {
		encryptedThumbnail, err = s.encryptor.Encrypt(thumbnail.Data, mek)
		if err != nil {
			s.logger.Error("Failed to encrypt thumbnail", err)
			return nil, err
		}
		thumbnailWidth = thumbnail.Width
		thumbnailHeight = thumbnail.Height
		thumbnailMimeType = thumbnail.MimeType
	}

	// Create clip object
	clip := &Clip{
		ID:                 clipID,
		ClientID:           clientID,
		Title:              title,
		TimeStamp:          req.TimeStamp,
		Duration:           req.Duration,
		HasMotion:          req.HasMotion,
		EncryptedVideo:     encryptedVideo,
		VideoWidth:         videoMeta.Width,
		VideoHeight:        videoMeta.Height,
		VideoMimeType:      videoMeta.MimeType,
		EncryptedThumbnail: encryptedThumbnail,
		ThumbnailWidth:     thumbnailWidth,
		ThumbnailHeight:    thumbnailHeight,
		ThumbnailMimeType:  thumbnailMimeType,
	}

	// Save clip to repository
	err = s.storageManager.StoreClip(context.Background(), clip)
	if err != nil {
		s.logger.Error("Failed to save clip", "error", err)
		return nil, err
	}

	s.logger.Info(fmt.Sprintf("Successfully created clip %s for client %s", clipID, clientID))
	return clip, nil
}
