package clients

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"slices"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/encryption"
)

type CreateClientRequest struct {
	ID                    string
	StorageLimitMegabytes int
	ClipDurationSeconds   int
	MotionOnly            bool
	Grayscale             bool
	DownscaleResolution   string
	OutputFormat          string
	OutputCodec           string
	VideoBitRate          string
	MotionMinArea         int
	MotionMaxFrames       int
	MotionWarmUpFrames    int
	CaptureCodec          string
	CaptureFrameRate      float64
}

type UpdateClientSettingsRequest struct {
	ID                    string
	StorageLimitMegabytes int
	ClipDurationSeconds   int
	MotionOnly            bool
	Grayscale             bool
	DownscaleResolution   string
	OutputFormat          string
	OutputCodec           string
	VideoBitRate          string
	MotionMinArea         int
	MotionMaxFrames       int
	MotionWarmUpFrames    int
	CaptureCodec          string
	CaptureFrameRate      float64
}

var supportedDownscaleResolutions = []string{"", "360p", "480p", "640x480", "720p", "800x600", "1024x768", "1080p"}
var supportedCaptureCodecs = []string{"MJPG", "YUYV", "H264"}
var supportedOutputCodecs = []string{"libx264", "libopenh264", "libx265", "libvpx-vp9", "ffv1"}
var supportedOutputFormats = []string{"mp4", "avi", "mkv", "webm", "mov"}
var supportedVideoBitrates = []string{"500k", "1000k", "1500k", "4000k", "8000k", "15000k"}

type ClientService interface {
	// CreateClient creates a new client with the given details
	CreateClient(req CreateClientRequest, mekStore encryption.MekStore) (client *Client, secret []byte, err error)
	// GetClient retrieves a client by its ID
	GetClient(id string) (*Client, error)
	// GetClients retrieves all clients
	GetClients() ([]*Client, error)
	// UpdateClientSettings updates the settings for a client
	UpdateClientSettings(req UpdateClientSettingsRequest) error
	// DeleteClient deletes a client by its ID
	DeleteClient(id string) error
	// DisableClient disables a client (soft delete)
	DisableClient(id string) error
	// EnableClient enables a disabled client
	EnableClient(id string) error
	// GetSupportedDownscaleResolutions returns a list of supported downscale resolutions
	GetSupportedDownscaleResolutions() []string
	// GetSupportedCaptureCodecs returns a list of supported capture codecs
	GetSupportedCaptureCodecs() []string
	// GetSupportedOutputCodecs returns a list of supported output codecs
	GetSupportedOutputCodecs() []string
	// GetSupportedOutputFormats returns a list of supported output formats
	GetSupportedOutputFormats() []string
	// GetSupportedVideoBitrates returns a list of supported video bitrates
	GetSupportedVideoBitrates() []string
}

type clientService struct {
	logger    logging.Logger
	repo      ClientRepository
	encryptor encryption.Encryptor
}

func NewClientService(logger logging.Logger, repo ClientRepository, encryptor encryption.Encryptor) *clientService {

	if logger == nil {
		logger = logging.NopLogger
	}

	return &clientService{
		logger:    logger,
		repo:      repo,
		encryptor: encryptor,
	}
}

func (s *clientService) validateClientSettings(req UpdateClientSettingsRequest) error {
	if req.ClipDurationSeconds < 30 || req.ClipDurationSeconds > 1800 { // 30 minutes = 1800 seconds
		return NewClientValidationError("clip duration must be between 30 and 1800 seconds")
	}

	if !slices.Contains(supportedDownscaleResolutions, req.DownscaleResolution) {
		return NewClientValidationError("unsupported downscale resolution")
	}

	if !slices.Contains(supportedCaptureCodecs, req.CaptureCodec) {
		return NewClientValidationError("unsupported capture codec")
	}

	if !slices.Contains(supportedOutputCodecs, req.OutputCodec) {
		return NewClientValidationError("unsupported output codec")
	}

	if !slices.Contains(supportedOutputFormats, req.OutputFormat) {
		return NewClientValidationError("unsupported output format")
	}

	if !slices.Contains(supportedVideoBitrates, req.VideoBitRate) {
		return NewClientValidationError("unsupported video bitrate")
	}

	return nil
}

func (s *clientService) CreateClient(req CreateClientRequest, mekStore encryption.MekStore) (*Client, []byte, error) {
	updateReq := UpdateClientSettingsRequest(req)
	if err := s.validateClientSettings(updateReq); err != nil {
		return nil, nil, err
	}

	// trim the id
	id := strings.TrimSpace(req.ID)

	if id == "" {
		return nil, nil, NewClientValidationError("client ID cannot be empty")
	}

	s.logger.Info("Creating client", "id", id)

	ctx := context.Background()

	// Check if the client already exists
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to check for existing client", err)
		return nil, nil, err
	}
	if existing != nil {
		s.logger.Error("Client already exists", "id", id)
		return nil, nil, NewClientAlreadyExistsError(id)
	}

	// Generate a new secret for the client
	secret, err := s.encryptor.GenerateKey()
	if err != nil {
		s.logger.Error("Failed to generate client secret", err)
		return nil, nil, err
	}

	// hash the secret
	hashedSecret, salt, err := s.encryptor.Hash(secret)
	if err != nil {
		s.logger.Error("Failed to hash client secret", err)
		return nil, nil, err
	}

	hashedSecretBase64 := base64.StdEncoding.EncodeToString(hashedSecret)
	saltBase64 := base64.StdEncoding.EncodeToString(salt)

	now := time.Now().UTC()

	// Get the MEK from the store
	// The MEK is guaranteed to not be nil. If it is not found, an error will be returned.
	mek, err := mekStore.GetMek()
	if err != nil {
		s.logger.Error("Failed to get MEK from store", err)
		return nil, nil, err
	}

	// Generate a key-derivation salt
	keyDerivationSalt, err := s.encryptor.GenerateSalt()
	if err != nil {
		s.logger.Error("Failed to generate key-derivation salt", err)
		return nil, nil, err
	}

	// Derive a key from the secret
	secretDerivedKey, err := s.encryptor.DeriveKeyFromSecret(secret, keyDerivationSalt)
	if err != nil {
		s.logger.Error("Failed to derive key from secret", err)
		return nil, nil, err
	}

	// Re-encrypt the MEK using the new client's secret
	encryptedMek, err := s.encryptor.Encrypt(mek, secretDerivedKey)
	if err != nil {
		s.logger.Error("Failed to encrypt MEK", err)
		return nil, nil, err
	}

	// creat the client
	client := &Client{
		ID:                    id,
		SecretHash:            hashedSecretBase64,
		SecretSalt:            saltBase64,
		CreatedAt:             now,
		UpdatedAt:             now,
		EncryptedMek:          base64.StdEncoding.EncodeToString(encryptedMek),
		KeyDerivationSalt:     base64.StdEncoding.EncodeToString(keyDerivationSalt),
		StorageLimitMegabytes: req.StorageLimitMegabytes,
		ClipDurationSeconds:   req.ClipDurationSeconds,
		MotionOnly:            req.MotionOnly,
		Grayscale:             req.Grayscale,
		DownscaleResolution:   req.DownscaleResolution,
		OutputFormat:          req.OutputFormat,
		OutputCodec:           req.OutputCodec,
		VideoBitRate:          req.VideoBitRate,
		MotionMinArea:         req.MotionMinArea,
		MotionMaxFrames:       req.MotionMaxFrames,
		MotionWarmUpFrames:    req.MotionWarmUpFrames,
		CaptureCodec:          req.CaptureCodec,
		CaptureFrameRate:      req.CaptureFrameRate,
	}

	// Save the client to the repository
	if err := s.repo.Create(ctx, client); err != nil {
		s.logger.Error("Failed to save client to repository", err)
		return nil, nil, err
	}

	s.logger.Info("Successfully created client", "id", client.ID)
	return client, secret, nil
}

func (s *clientService) GetClient(id string) (*Client, error) {
	s.logger.Info("Retrieving client", "id", id)

	ctx := context.Background()

	// Retrieve the client by ID
	client, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to retrieve client", err)
		return nil, err
	}
	if client == nil {
		s.logger.Info("Client not found", "id", id)
		return nil, nil // No error if the client does not exist
	}

	return client, nil
}

func (s *clientService) GetClients() ([]*Client, error) {
	s.logger.Info("Retrieving all clients")

	ctx := context.Background()

	// Retrieve all clients
	clients, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.Error("Failed to retrieve clients", err)
		return nil, err
	}

	return clients, nil
}

func (s *clientService) UpdateClientSettings(req UpdateClientSettingsRequest) error {
	if err := s.validateClientSettings(req); err != nil {
		return err
	}

	s.logger.Info("Updating client settings", "id", req.ID)

	ctx := context.Background()

	// Retrieve the client by ID
	client, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("Failed to retrieve client", err)
		return err
	}
	if client == nil {
		s.logger.Info("Client not found", "id", req.ID)
		return NewClientNotFoundError(req.ID)
	}

	// Update the client's settings
	client.StorageLimitMegabytes = req.StorageLimitMegabytes
	client.ClipDurationSeconds = req.ClipDurationSeconds
	client.MotionOnly = req.MotionOnly
	client.Grayscale = req.Grayscale
	client.DownscaleResolution = req.DownscaleResolution
	client.OutputFormat = req.OutputFormat
	client.OutputCodec = req.OutputCodec
	client.VideoBitRate = req.VideoBitRate
	client.MotionMinArea = req.MotionMinArea
	client.MotionMaxFrames = req.MotionMaxFrames
	client.MotionWarmUpFrames = req.MotionWarmUpFrames
	client.CaptureCodec = req.CaptureCodec
	client.CaptureFrameRate = req.CaptureFrameRate
	client.UpdatedAt = time.Now().UTC()

	// Save the updated client to the repository
	if err := s.repo.Update(ctx, client); err != nil {
		s.logger.Error("Failed to update client in repository", err)
		return err
	}

	s.logger.Info("Successfully updated client settings", "id", client.ID)
	return nil
}

func (s *clientService) GetSupportedDownscaleResolutions() []string {
	return supportedDownscaleResolutions
}

func (s *clientService) GetSupportedCaptureCodecs() []string {
	return supportedCaptureCodecs
}

func (s *clientService) GetSupportedOutputCodecs() []string {
	return supportedOutputCodecs
}

func (s *clientService) GetSupportedOutputFormats() []string {
	return supportedOutputFormats
}

func (s *clientService) GetSupportedVideoBitrates() []string {
	return supportedVideoBitrates
}

func (s *clientService) DeleteClient(id string) error {
	s.logger.Info("Deleting client", "id", id)

	ctx := context.Background()

	// Check if the client exists
	client, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to retrieve client", err)
		return err
	}
	if client == nil {
		s.logger.Info("Client not found", "id", id)
		return nil // No error if the client does not exist
	}

	// Delete the client
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete client", err)
		return err
	}

	s.logger.Info("Successfully deleted client", "id", id)
	return nil
}

func (s *clientService) DisableClient(id string) error {
	s.logger.Info("Disabling client", "id", id)

	ctx := context.Background()

	// Check if the client exists
	client, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to retrieve client", err)
		return err
	}
	if client == nil {
		s.logger.Info("Client not found", "id", id)
		return nil // No error if the client does not exist
	}

	// Update the client to disable it
	client.IsDisabled = true
	client.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, client); err != nil {
		s.logger.Error("Failed to disable client", err)
		return err
	}

	s.logger.Info("Successfully disabled client", "id", id)
	return nil
}

func (s *clientService) EnableClient(id string) error {
	s.logger.Info("Enabling client", "id", id)

	ctx := context.Background()

	// Check if the client exists
	client, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to retrieve client", err)
		return err
	}
	if client == nil {
		s.logger.Info("Client not found", "id", id)
		return nil // No error if the client does not exist
	}

	// Update the client to enable it
	client.IsDisabled = false
	client.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, client); err != nil {
		s.logger.Error("Failed to enable client", err)
		return err
	}

	s.logger.Info("Successfully enabled client", "id", id)
	return nil
}
