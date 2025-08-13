package encryption

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

type MekService interface {
	// CreateMek creates a new MEK and persists it
	CreateMek(password string) (*Mek, error)
	// GetMek retrieves the MEK from the repository
	GetMek() (*Mek, error)
	// ChangeMekPassword updates the existing MEK with a new password (requires old password to decrypt)
	ChangeMekPassword(oldPassword, newPassword string) (*Mek, error)
	// DeleteMek deletes the MEK from the database
	DeleteMek() error
}

type mekService struct {
	logger    logging.Logger
	repo      MekRepository
	encryptor Encryptor
}

func NewMekService(logger logging.Logger, repo MekRepository, encryptor Encryptor) *mekService {

	if logger == nil {
		logger = logging.NopLogger
	}

	return &mekService{
		logger:    logger,
		repo:      repo,
		encryptor: encryptor,
	}
}

func (s *mekService) CreateMek(password string) (*Mek, error) {
	s.logger.Info("Creating MEK")

	// We cannot create a MEK if one already exists.
	// In that case, UpdateMek should be used instead.
	existing, err := s.repo.Get()
	if err != nil {
		s.logger.Error("Failed to check for existing MEK", err)
		return nil, err
	}
	if existing != nil {
		s.logger.Error("MEK already exists, use UpdateMek instead")
		return nil, NewMekAlreadyExistsError(existing.ID)
	}

	mekID := uuid.NewString()

	// Generate a random MEK
	mekValue, err := s.encryptor.GenerateKey()
	if err != nil {
		s.logger.Error("Failed to generate MEK", err)
		return nil, err
	}

	// Derive a key from the password to encrypt the MEK with
	salt, err := s.encryptor.GenerateSalt()
	if err != nil {
		s.logger.Error("Failed to generate salt for MEK encryption", err)
		return nil, err
	}
	key, err := s.encryptor.DeriveKeyFromSecret([]byte(password), salt)
	if err != nil {
		s.logger.Error("Failed to derive key from password for MEK encryption", err)
		return nil, err
	}

	// Encrypt the MEK
	encryptedKey, err := s.encryptor.Encrypt(mekValue, key)
	if err != nil {
		s.logger.Error("Failed to encrypt MEK", err)
		return nil, err
	}

	now := time.Now().UTC()

	// base64 encode the encrypted key for storage
	encryptedKeyBase64 := base64.StdEncoding.EncodeToString(encryptedKey)

	// same for the salt
	saltBase64 := base64.StdEncoding.EncodeToString(salt)

	// Create the MEK object
	mek := &Mek{
		ID:                     mekID,
		EncryptedEncryptionKey: encryptedKeyBase64,
		EncryptionKeySalt:      saltBase64,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	// Persist the MEK
	if err := s.repo.Create(mek); err != nil {
		s.logger.Error("Failed to create MEK in repository", err)
		return nil, fmt.Errorf("failed to create MEK: %w", err)
	}

	return mek, nil
}

func (s *mekService) GetMek() (*Mek, error) {
	s.logger.Info("Retrieving MEK")

	// Retrieve the MEK from the repository
	mek, err := s.repo.Get()
	if err != nil {
		s.logger.Error("Failed to get MEK from repository", err)
		return nil, fmt.Errorf("failed to get MEK: %w", err)
	}

	if mek == nil {
		s.logger.Info("No MEK found")
		return nil, NewMekNotFoundError() // A specific error helps enforcing the existence of a MEK
	}

	return mek, nil
}

func (s *mekService) ChangeMekPassword(oldPassword, newPassword string) (*Mek, error) {
	s.logger.Info("Updating MEK password")

	// Retrieve the existing MEK
	mek, err := s.GetMek()
	if err != nil {
		s.logger.Error("Failed to retrieve existing MEK", err)
		return nil, err
	}

	if mek == nil {
		s.logger.Error("No existing MEK found to update")
		return nil, NewMekNotFoundError()
	}

	// Decrypt the current MEK using the old password
	oldSaltBytes, err := base64.StdEncoding.DecodeString(mek.EncryptionKeySalt)
	if err != nil {
		s.logger.Error("Failed to decode old MEK salt", err)
		return nil, fmt.Errorf("failed to decode old MEK salt: %w", err)
	}

	oldKey, err := s.encryptor.DeriveKeyFromSecret([]byte(oldPassword), oldSaltBytes)
	if err != nil {
		s.logger.Error("Failed to derive key from old password", err)
		return nil, fmt.Errorf("failed to derive key from old password: %w", err)
	}

	encryptedMekBytes, err := base64.StdEncoding.DecodeString(mek.EncryptedEncryptionKey)
	if err != nil {
		s.logger.Error("Failed to decode encrypted MEK", err)
		return nil, fmt.Errorf("failed to decode encrypted MEK: %w", err)
	}

	// Decrypt the MEK value using the old password
	mekValue, err := s.encryptor.Decrypt(encryptedMekBytes, oldKey)
	if err != nil {
		s.logger.Error("Failed to decrypt MEK with old password", err)
		return nil, fmt.Errorf("failed to decrypt MEK with old password (invalid old password?): %w", err)
	}

	// Generate a new salt and derive a key from the new password
	newSalt, err := s.encryptor.GenerateSalt()
	if err != nil {
		s.logger.Error("Failed to generate new salt for MEK encryption", err)
		return nil, fmt.Errorf("failed to generate new salt: %w", err)
	}

	newKey, err := s.encryptor.DeriveKeyFromSecret([]byte(newPassword), newSalt)
	if err != nil {
		s.logger.Error("Failed to derive key from new password", err)
		return nil, fmt.Errorf("failed to derive key from new password: %w", err)
	}

	// Re-encrypt the same MEK value with the new password
	newEncryptedKey, err := s.encryptor.Encrypt(mekValue, newKey)
	if err != nil {
		s.logger.Error("Failed to encrypt MEK with new password", err)
		return nil, fmt.Errorf("failed to encrypt MEK with new password: %w", err)
	}

	// Update the MEK fields with new encryption but preserve the MEK value
	mek.EncryptedEncryptionKey = base64.StdEncoding.EncodeToString(newEncryptedKey)
	mek.EncryptionKeySalt = base64.StdEncoding.EncodeToString(newSalt)
	mek.UpdatedAt = time.Now().UTC()

	// Persist the updated MEK
	if err := s.repo.Update(mek); err != nil {
		s.logger.Error("Failed to update MEK in repository", err)
		return nil, fmt.Errorf("failed to update MEK: %w", err)
	}

	s.logger.Info("MEK password updated successfully")
	return mek, nil
}

func (s *mekService) DeleteMek() error {
	s.logger.Info("Deleting MEK")

	// Delete the MEK from the repository
	if err := s.repo.Delete(); err != nil {
		s.logger.Error("Failed to delete MEK from repository", err)
		return fmt.Errorf("failed to delete MEK: %w", err)
	}

	s.logger.Info("MEK deleted successfully")
	return nil
}
