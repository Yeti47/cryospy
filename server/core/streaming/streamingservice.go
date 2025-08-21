package streaming

import (
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/config"
	"github.com/yeti47/cryospy/server/core/encryption"
	"github.com/yeti47/cryospy/server/core/videos"
)

type StreamingService interface {
	GetPlaylist(clientID string, startTime time.Time, referenceTime time.Time) (string, error)
	GetSegment(clientID, clipID string, mekStore encryption.MekStore) ([]byte, error)
}

type streamingService struct {
	logger            logging.Logger
	clipReader        videos.ClipReader
	clipNormalizer    ClipNormalizer
	playlistGenerator PlaylistGenerator
	config            config.StreamingSettings
}

func NewStreamingService(logger logging.Logger, clipReader videos.ClipReader, clipNormalizer ClipNormalizer, playlistGenerator PlaylistGenerator, config config.StreamingSettings) *streamingService {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &streamingService{
		logger:            logger,
		clipReader:        clipReader,
		clipNormalizer:    clipNormalizer,
		playlistGenerator: playlistGenerator,
		config:            config,
	}
}

func (s *streamingService) virtualizeNow(startTime time.Time, referenceTime time.Time) time.Time {

	elapsed := time.Since(startTime)
	virtualNow := referenceTime.Add(elapsed)

	return virtualNow
}

func (s *streamingService) GetPlaylist(clientID string, startTime time.Time, referenceTime time.Time) (string, error) {

	virtualNow := s.virtualizeNow(startTime, referenceTime)

	lookAhead := s.config.LookAhead

	if lookAhead <= 0 {
		lookAhead = 10 // Default to 10 clips if not configured
	}

	s.logger.Debug("Getting clip infos for playlist", "clientID", clientID, "virtualNow", virtualNow, "lookAhead", lookAhead)

	// Get clip infos based on the virtual now
	clipInfos, err := s.clipReader.GetClipInfosByReferenceTime(clientID, virtualNow, lookAhead)
	if err != nil {
		s.logger.Error("Failed to get clip infos for playlist", err)
		return "", err
	}

	s.logger.Debug("Found clip infos", "clientID", clientID, "clipCount", len(clipInfos))

	// Generate the playlist from clip infos
	playlist, err := s.playlistGenerator.GeneratePlaylist(clipInfos, true) // we always want the behavior of a "live" playlist, since new clips may be added
	if err != nil {
		s.logger.Error("Failed to generate playlist", err)
		return "", err
	}

	return playlist, nil
}

func (s *streamingService) GetSegment(clientID, clipID string, mekStore encryption.MekStore) ([]byte, error) {
	s.logger.Debug("Getting segment", "clientID", clientID, "clipID", clipID)

	// Get the clip by ID
	clip, err := s.clipReader.GetClipByID(clipID, mekStore)
	if err != nil {
		s.logger.Error("Failed to get clip by ID", err, "clientID", clientID, "clipID", clipID)
		return nil, err
	}

	if clip == nil || clip.ClientID != clientID {
		s.logger.Error("Clip does not belong to the client", nil)
		return nil, NewSegmentNotFoundError(clipID, clientID)
	}

	s.logger.Debug("Found clip, normalizing", "clientID", clientID, "clipID", clipID, "clipSize", len(clip.Video))

	// Normalize the clip data for streaming
	normalizedClip, err := s.clipNormalizer.NormalizeClip(clip)
	if err != nil {
		s.logger.Error("Failed to normalize clip for streaming", err, "clientID", clientID, "clipID", clipID)
		return nil, err
	}

	s.logger.Debug("Normalized clip successfully", "clientID", clientID, "clipID", clipID, "normalizedSize", len(normalizedClip))

	return normalizedClip, nil
}
