package videos

import (
	"context"
	"fmt"
	"sync"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/notifications"
)

const bytesInMegabyte = 1024 * 1024

type StorageInfo struct {
	TotalAvailableBytes int64   // Total storage available in bytes (0 means unlimited)
	TotalUsedBytes      int64   // Total storage used in bytes
	UsagePercent        float64 // Percentage of storage used (0-100)
}

type StorageManager interface {
	StoreClip(ctx context.Context, clip *Clip) error
	GetStorageInfo(ctx context.Context, clientID string) (*StorageInfo, error)
}

type storageManager struct {
	logger         logging.Logger
	clipRepo       ClipRepository
	clientRepo     clients.ClientRepository
	notifier       notifications.StorageNotifier
	motionNotifier notifications.MotionNotifier

	// Mutex map for per-client storage limit operations only
	clientStorageMutexes sync.Map // map[string]*sync.Mutex
}

func NewStorageManager(logger logging.Logger, clipRepo ClipRepository, clientRepo clients.ClientRepository, notifier notifications.StorageNotifier, motionNotifier notifications.MotionNotifier) StorageManager {
	if logger == nil {
		logger = logging.NopLogger
	}
	if notifier == nil {
		notifier = notifications.NopStorageNotifier
	}
	if motionNotifier == nil {
		motionNotifier = notifications.NopMotionNotifier
	}
	return &storageManager{
		logger:               logger,
		clipRepo:             clipRepo,
		clientRepo:           clientRepo,
		notifier:             notifier,
		motionNotifier:       motionNotifier,
		clientStorageMutexes: sync.Map{},
	}
}

// getStorageMutex returns a mutex for storage operations for the given client ID
func (s *storageManager) getStorageMutex(clientID string) *sync.Mutex {
	mutex, _ := s.clientStorageMutexes.LoadOrStore(clientID, &sync.Mutex{})
	return mutex.(*sync.Mutex)
}

func (s *storageManager) StoreClip(ctx context.Context, clip *Clip) error {
	client, err := s.clientRepo.GetByID(ctx, clip.ClientID)
	if err != nil {
		s.logger.Error("failed to get client for storage management", "error", err, "client_id", clip.ClientID)
		return err
	}

	if client == nil {
		s.logger.Error("client not found", "client_id", clip.ClientID)
		return fmt.Errorf("client not found: %s", clip.ClientID)
	}

	if client.StorageLimitMegabytes <= 0 {
		s.logger.Info("client has unlimited storage, storing clip directly", "client_id", clip.ClientID)
		err := s.clipRepo.Add(ctx, clip)
		if err != nil {
			return err
		}

		// Send motion notification if clip has motion
		if clip.HasMotion {
			err = s.motionNotifier.NotifyMotionDetected(clip.ClientID, clip.Title, clip.TimeStamp)
			if err != nil {
				s.logger.Warn("failed to send motion detection notification", "error", err, "client_id", clip.ClientID)
			}
		}

		return nil
	}

	// For clients with storage limits, use a mutex to make storage check-and-store atomic
	storageMutex := s.getStorageMutex(clip.ClientID)
	storageMutex.Lock()
	defer storageMutex.Unlock()

	usageBytes, err := s.clipRepo.GetTotalStorageUsage(ctx, clip.ClientID)
	if err != nil {
		s.logger.Error("failed to get total storage usage", "error", err, "client_id", clip.ClientID)
		return err
	}

	usageMegaBytes := usageBytes / bytesInMegabyte
	totalMegaBytes := int64(client.StorageLimitMegabytes)
	newClipSizeMegaBytes := int64(len(clip.EncryptedVideo)) / bytesInMegabyte

	capacityExceeded := (usageMegaBytes + newClipSizeMegaBytes) > totalMegaBytes

	if s.notifier.ShouldWarn(usageMegaBytes, totalMegaBytes) && !capacityExceeded {
		err := s.notifier.NotifyCapacityWarning(clip.ClientID, usageMegaBytes, totalMegaBytes)
		if err != nil {
			s.logger.Warn("failed to send capacity warning", "error", err, "client_id", clip.ClientID)
		}
	}

	if capacityExceeded {
		s.logger.Warn("storage capacity exceeded, deleting oldest clips", "client_id", clip.ClientID)
		err := s.notifier.NotifyCapacityReached(clip.ClientID, usageMegaBytes, totalMegaBytes)
		if err != nil {
			s.logger.Warn("failed to send capacity reached notification", "error", err, "client_id", clip.ClientID)
		}
	}

	for (usageMegaBytes + newClipSizeMegaBytes) > totalMegaBytes {
		oldestClips, err := s.clipRepo.GetOldestClips(ctx, clip.ClientID, 1)
		if err != nil {
			s.logger.Error("failed to get oldest clips for deletion", "error", err, "client_id", clip.ClientID)
			return err
		}

		if len(oldestClips) == 0 {
			s.logger.Warn("no more clips to delete, but capacity still exceeded", "client_id", clip.ClientID)
			break
		}

		oldestClip := oldestClips[0]

		// Try to delete the oldest clip
		err = s.clipRepo.Delete(ctx, oldestClip.ID)
		if err != nil {
			s.logger.Error("failed to delete oldest clip", "error", err, "clip_id", oldestClip.ID)
			// Continue anyway - we don't want deletion failures to prevent storing new clips
			// Just log and break out of the cleanup loop
			s.logger.Warn("stopping cleanup due to deletion failure, proceeding with storage", "client_id", clip.ClientID)
			break
		}
		s.logger.Info("deleted oldest clip to free up space", "clip_id", oldestClip.ID, "client_id", clip.ClientID)

		// Refresh usage after deletion
		usageBytes, err = s.clipRepo.GetTotalStorageUsage(ctx, clip.ClientID)
		if err != nil {
			s.logger.Error("failed to get updated total storage usage", "error", err, "client_id", clip.ClientID)
			return err
		}
		usageMegaBytes = usageBytes / bytesInMegabyte
	}

	// Store the clip
	err = s.clipRepo.Add(ctx, clip)
	if err != nil {
		return err
	}

	// Send motion notification if clip has motion
	if clip.HasMotion {
		err = s.motionNotifier.NotifyMotionDetected(clip.ClientID, clip.Title, clip.TimeStamp)
		if err != nil {
			s.logger.Warn("failed to send motion detection notification", "error", err, "client_id", clip.ClientID)
		}
	}

	return nil
}

func (s *storageManager) GetStorageInfo(ctx context.Context, clientID string) (*StorageInfo, error) {
	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		s.logger.Error("failed to get client for storage info", "error", err, "client_id", clientID)
		return nil, err
	}

	if client == nil {
		s.logger.Error("client not found", "client_id", clientID)
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	usageBytes, err := s.clipRepo.GetTotalStorageUsage(ctx, clientID)
	if err != nil {
		s.logger.Error("failed to get total storage usage", "error", err, "client_id", clientID)
		return nil, err
	}

	var totalAvailableBytes int64
	var usagePercent float64

	if client.StorageLimitMegabytes <= 0 {
		// Unlimited storage
		totalAvailableBytes = 0
		usagePercent = 0.0
	} else {
		totalAvailableBytes = int64(client.StorageLimitMegabytes) * bytesInMegabyte
		if totalAvailableBytes > 0 {
			usagePercent = (float64(usageBytes) / float64(totalAvailableBytes)) * 100.0
		}
	}

	return &StorageInfo{
		TotalAvailableBytes: totalAvailableBytes,
		TotalUsedBytes:      usageBytes,
		UsagePercent:        usagePercent,
	}, nil
}
