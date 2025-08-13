package clients

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/encryption"
)

type ClientService interface {
	// CreateClient creates a new client with the given details
	CreateClient(id string, storageLimitMegabytes int, mekStore encryption.MekStore) (client *Client, secret []byte, err error)
	// GetClient retrieves a client by its ID
	GetClient(id string) (*Client, error)
	// GetClients retrieves all clients
	GetClients() ([]*Client, error)
	// SetStorageLimit sets the storage limit for a client
	SetStorageLimit(id string, newLimitMegabytes int) error
	// DeleteClient deletes a client by its ID
	DeleteClient(id string) error
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

func (s *clientService) CreateClient(id string, storageLimitMegabytes int, mekStore encryption.MekStore) (*Client, []byte, error) {
	// trim the id
	id = strings.TrimSpace(id)

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
		StorageLimitMegabytes: storageLimitMegabytes,
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

func (s *clientService) SetStorageLimit(id string, newLimitMegabytes int) error {
	s.logger.Info("Setting storage limit for client", "id", id, "newLimitMegabytes", newLimitMegabytes)

	// validate the new limit
	if newLimitMegabytes <= 0 {
		s.logger.Error("Invalid storage limit", "id", id, "newLimitMegabytes", newLimitMegabytes)
		return fmt.Errorf("invalid storage limit: %d", newLimitMegabytes)
	}

	ctx := context.Background()

	// Retrieve the client by ID
	client, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to retrieve client", err)
		return err
	}
	if client == nil {
		s.logger.Info("Client not found", "id", id)
		return nil // No error if the client does not exist
	}

	// Update the storage limit
	client.StorageLimitMegabytes = newLimitMegabytes
	client.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, client); err != nil {
		s.logger.Error("Failed to update client storage limit", err)
		return err
	}

	return nil
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
