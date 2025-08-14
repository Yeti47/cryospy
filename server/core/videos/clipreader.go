package videos

import (
	"context"
	"fmt"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/encryption"
)

// ClipReader handles reading and decrypting video clips for admin access
type ClipReader interface {
	// QueryClips retrieves clips with decrypted video and thumbnail data
	// Returns clips and total count of matching records (before pagination)
	QueryClips(query ClipQuery, mekStore encryption.MekStore) ([]*DecryptedClip, int, error)
	// QueryClipInfos retrieves clip metadata without decrypted video/thumbnail data
	// Returns clip infos and total count of matching records (before pagination)
	QueryClipInfos(query ClipQuery) ([]*ClipInfo, int, error)
	// GetClipByID retrieves a single clip by ID with decrypted data
	GetClipByID(clipID string, mekStore encryption.MekStore) (*DecryptedClip, error)
	// GetClipInfoByID retrieves clip metadata by ID without decrypted data
	GetClipInfoByID(clipID string) (*ClipInfo, error)
	// GetClipThumbnail retrieves the thumbnail for a clip by ID with decrypted data
	GetClipThumbnail(clipID string, mekStore encryption.MekStore) (*Thumbnail, error)
}

type clipReader struct {
	logger    logging.Logger
	clipRepo  ClipRepository
	encryptor encryption.Encryptor
}

// NewClipReader creates a new ClipReader service
func NewClipReader(logger logging.Logger, clipRepo ClipRepository, encryptor encryption.Encryptor) *clipReader {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &clipReader{
		logger:    logger,
		clipRepo:  clipRepo,
		encryptor: encryptor,
	}
}

func (r *clipReader) QueryClips(query ClipQuery, mekStore encryption.MekStore) ([]*DecryptedClip, int, error) {
	// Get the admin's MEK for decryption
	mek, err := mekStore.GetMek()
	if err != nil {
		r.logger.Error("Failed to get MEK for clip query", err)
		return nil, 0, err
	}

	// Query encrypted clips from repository
	clips, totalCount, err := r.clipRepo.Query(context.Background(), query)
	if err != nil {
		r.logger.Error("Failed to query clips from repository", err)
		return nil, 0, err
	}

	// Decrypt clips
	decryptedClips := make([]*DecryptedClip, 0, len(clips))
	for _, clip := range clips {
		decryptedClip, err := r.decryptClip(clip, mek)
		if err != nil {
			r.logger.Error(fmt.Sprintf("Failed to decrypt clip %s", clip.ID), err)
			// Skip this clip and continue with others
			continue
		}
		decryptedClips = append(decryptedClips, decryptedClip)
	}

	r.logger.Info(fmt.Sprintf("Successfully retrieved %d decrypted clips", len(decryptedClips)))
	return decryptedClips, totalCount, nil
}

func (r *clipReader) QueryClipInfos(query ClipQuery) ([]*ClipInfo, int, error) {
	// Query clip infos from repository (no decryption needed)
	clipInfos, totalCount, err := r.clipRepo.QueryInfo(context.Background(), query)
	if err != nil {
		r.logger.Error("Failed to query clip infos from repository", err)
		return nil, 0, err
	}

	r.logger.Info(fmt.Sprintf("Successfully retrieved %d clip infos", len(clipInfos)))
	return clipInfos, totalCount, nil
}

func (r *clipReader) GetClipByID(clipID string, mekStore encryption.MekStore) (*DecryptedClip, error) {
	// Get the admin's MEK for decryption
	mek, err := mekStore.GetMek()
	if err != nil {
		r.logger.Error("Failed to get MEK for clip retrieval", err)
		return nil, err
	}

	// Get clip from repository
	clip, err := r.clipRepo.GetByID(context.Background(), clipID)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to get clip %s from repository", clipID), err)
		return nil, err
	}

	// Decrypt clip
	decryptedClip, err := r.decryptClip(clip, mek)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to decrypt clip %s", clipID), err)
		return nil, err
	}

	r.logger.Info(fmt.Sprintf("Successfully retrieved and decrypted clip %s", clipID))
	return decryptedClip, nil
}

func (r *clipReader) GetClipInfoByID(clipID string) (*ClipInfo, error) {
	// Get clip info from repository (no decryption needed)
	clipInfo, err := r.clipRepo.GetInfoByID(context.Background(), clipID)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to get clip info %s from repository", clipID), err)
		return nil, err
	}

	if clipInfo == nil {
		r.logger.Warn(fmt.Sprintf("Clip info not found for ID %s", clipID))
		return nil, nil
	}

	r.logger.Info(fmt.Sprintf("Successfully retrieved clip info %s", clipID))
	return clipInfo, nil
}

// decryptClip decrypts a clip's video and thumbnail data
func (r *clipReader) decryptClip(clip *Clip, mek []byte) (*DecryptedClip, error) {
	// Decrypt video data
	video, err := r.encryptor.Decrypt(clip.EncryptedVideo, mek)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt video: %w", err)
	}

	// Decrypt thumbnail data if present
	var thumbnail []byte
	if len(clip.EncryptedThumbnail) > 0 {
		thumbnail, err = r.encryptor.Decrypt(clip.EncryptedThumbnail, mek)
		if err != nil {
			r.logger.Warn(fmt.Sprintf("Failed to decrypt thumbnail for clip %s, proceeding without thumbnail", clip.ID))
			// Continue without thumbnail rather than failing the entire clip
			thumbnail = nil
		}
	}

	return &DecryptedClip{
		ID:                clip.ID,
		ClientID:          clip.ClientID,
		Title:             clip.Title,
		TimeStamp:         clip.TimeStamp,
		Duration:          clip.Duration,
		HasMotion:         clip.HasMotion,
		Video:             video,
		VideoWidth:        clip.VideoWidth,
		VideoHeight:       clip.VideoHeight,
		VideoMimeType:     clip.VideoMimeType,
		Thumbnail:         thumbnail,
		ThumbnailWidth:    clip.ThumbnailWidth,
		ThumbnailHeight:   clip.ThumbnailHeight,
		ThumbnailMimeType: clip.ThumbnailMimeType,
	}, nil
}

func (r *clipReader) GetClipThumbnail(clipID string, mekStore encryption.MekStore) (*Thumbnail, error) {
	// Get the admin's MEK for decryption
	mek, err := mekStore.GetMek()
	if err != nil {
		r.logger.Error("Failed to get MEK for thumbnail retrieval", err)
		return nil, err
	}

	// Get the thumbnail for the clip
	thumb, err := r.clipRepo.GetThumbnailByID(context.Background(), clipID)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to get thumbnail for clip %s", clipID), err)
		return nil, err
	}
	// If no thumbnail is found, return nil
	if thumb == nil {
		r.logger.Warn(fmt.Sprintf("No thumbnail found for clip %s", clipID))
		return nil, nil
	}

	// Decrypt thumbnail data
	if len(thumb.Data) == 0 {
		r.logger.Warn(fmt.Sprintf("No thumbnail available for clip %s", clipID))
		return nil, nil // No thumbnail available
	}

	thumbnailData, err := r.encryptor.Decrypt(thumb.Data, mek)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to decrypt thumbnail for clip %s", clipID), err)
		return nil, err
	}

	return &Thumbnail{
		Data:     thumbnailData,
		MimeType: thumb.MimeType,
	}, nil
}
