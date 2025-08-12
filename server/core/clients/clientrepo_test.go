package clients

import (
	"context"
	"testing"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/db"
)

func setupTestClientRepo(t *testing.T) (*SQLiteClientRepository, func()) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	repo, err := NewSQLiteClientRepository(testDB)
	if err != nil {
		testDB.Close()
		t.Fatalf("Failed to create repository: %v", err)
	}

	cleanup := func() {
		testDB.Close()
	}

	return repo, cleanup
}

func createTestClient() *Client {
	now := time.Now().UTC()
	return &Client{
		ID:                    "test-client-1",
		SecretHash:            "hashedSecret123",
		SecretSalt:            "saltValue456",
		CreatedAt:             now,
		UpdatedAt:             now,
		EncryptedMek:          "encryptedMekValue789",
		KeyDerivationSalt:     "keyDerivationSalt012",
		StorageLimitMegabytes: 1024,
	}
}

func TestSQLiteClientRepository_Create(t *testing.T) {
	repo, cleanup := setupTestClientRepo(t)
	defer cleanup()

	ctx := context.Background()
	client := createTestClient()

	err := repo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify the client was created by retrieving it
	retrieved, err := repo.GetByID(ctx, client.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve client: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved client is nil")
	}

	// Compare all fields
	if retrieved.ID != client.ID {
		t.Errorf("Expected ID %s, got %s", client.ID, retrieved.ID)
	}
	if retrieved.SecretHash != client.SecretHash {
		t.Errorf("Expected SecretHash %s, got %s", client.SecretHash, retrieved.SecretHash)
	}
	if retrieved.SecretSalt != client.SecretSalt {
		t.Errorf("Expected SecretSalt %s, got %s", client.SecretSalt, retrieved.SecretSalt)
	}
	if !retrieved.CreatedAt.Equal(client.CreatedAt) {
		t.Errorf("Expected CreatedAt %v, got %v", client.CreatedAt, retrieved.CreatedAt)
	}
	if !retrieved.UpdatedAt.Equal(client.UpdatedAt) {
		t.Errorf("Expected UpdatedAt %v, got %v", client.UpdatedAt, retrieved.UpdatedAt)
	}
	if retrieved.EncryptedMek != client.EncryptedMek {
		t.Errorf("Expected EncryptedMek %s, got %s", client.EncryptedMek, retrieved.EncryptedMek)
	}
	if retrieved.KeyDerivationSalt != client.KeyDerivationSalt {
		t.Errorf("Expected KeyDerivationSalt %s, got %s", client.KeyDerivationSalt, retrieved.KeyDerivationSalt)
	}
	if retrieved.StorageLimitMegabytes != client.StorageLimitMegabytes {
		t.Errorf("Expected StorageLimitMegabytes %d, got %d", client.StorageLimitMegabytes, retrieved.StorageLimitMegabytes)
	}
}

func TestSQLiteClientRepository_GetByID_NotFound(t *testing.T) {
	repo, cleanup := setupTestClientRepo(t)
	defer cleanup()

	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, "non-existent-id")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected nil for non-existent client, got a client")
	}
}

func TestSQLiteClientRepository_GetAll(t *testing.T) {
	repo, cleanup := setupTestClientRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()

	// Create multiple test clients
	clients := []*Client{
		{
			ID:                    "client-1",
			SecretHash:            "hash1",
			SecretSalt:            "salt1",
			CreatedAt:             now.Add(-2 * time.Hour),
			UpdatedAt:             now.Add(-2 * time.Hour),
			EncryptedMek:          "mek1",
			KeyDerivationSalt:     "kds1",
			StorageLimitMegabytes: 512,
		},
		{
			ID:                    "client-2",
			SecretHash:            "hash2",
			SecretSalt:            "salt2",
			CreatedAt:             now.Add(-1 * time.Hour),
			UpdatedAt:             now.Add(-1 * time.Hour),
			EncryptedMek:          "mek2",
			KeyDerivationSalt:     "kds2",
			StorageLimitMegabytes: 1024,
		},
		{
			ID:                    "client-3",
			SecretHash:            "hash3",
			SecretSalt:            "salt3",
			CreatedAt:             now,
			UpdatedAt:             now,
			EncryptedMek:          "mek3",
			KeyDerivationSalt:     "kds3",
			StorageLimitMegabytes: 2048,
		},
	}

	for _, client := range clients {
		err := repo.Create(ctx, client)
		if err != nil {
			t.Fatalf("Failed to create client %s: %v", client.ID, err)
		}
	}

	// Test GetAll (should return all clients, ordered by created_at DESC)
	allClients, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all clients: %v", err)
	}

	if len(allClients) != 3 {
		t.Errorf("Expected 3 clients, got %d", len(allClients))
	}

	// Verify order (newest first)
	if allClients[0].ID != "client-3" || allClients[1].ID != "client-2" || allClients[2].ID != "client-1" {
		t.Error("Clients not ordered correctly by created_at DESC")
	}
}

func TestSQLiteClientRepository_Update(t *testing.T) {
	repo, cleanup := setupTestClientRepo(t)
	defer cleanup()

	ctx := context.Background()
	client := createTestClient()

	// Create the client
	err := repo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Update the client
	client.SecretHash = "newHashedSecret"
	client.SecretSalt = "newSaltValue"
	client.EncryptedMek = "newEncryptedMek"
	client.KeyDerivationSalt = "newKeyDerivationSalt"
	client.StorageLimitMegabytes = 2048
	client.UpdatedAt = time.Now().UTC()

	err = repo.Update(ctx, client)
	if err != nil {
		t.Fatalf("Failed to update client: %v", err)
	}

	// Retrieve and verify the update
	retrieved, err := repo.GetByID(ctx, client.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated client: %v", err)
	}

	if retrieved.SecretHash != "newHashedSecret" {
		t.Errorf("Expected updated SecretHash 'newHashedSecret', got %s", retrieved.SecretHash)
	}
	if retrieved.SecretSalt != "newSaltValue" {
		t.Errorf("Expected updated SecretSalt 'newSaltValue', got %s", retrieved.SecretSalt)
	}
	if retrieved.EncryptedMek != "newEncryptedMek" {
		t.Errorf("Expected updated EncryptedMek 'newEncryptedMek', got %s", retrieved.EncryptedMek)
	}
	if retrieved.KeyDerivationSalt != "newKeyDerivationSalt" {
		t.Errorf("Expected updated KeyDerivationSalt 'newKeyDerivationSalt', got %s", retrieved.KeyDerivationSalt)
	}
	if retrieved.StorageLimitMegabytes != 2048 {
		t.Errorf("Expected updated StorageLimitMegabytes 2048, got %d", retrieved.StorageLimitMegabytes)
	}
	// CreatedAt should remain unchanged
	if !retrieved.CreatedAt.Equal(client.CreatedAt) {
		t.Errorf("CreatedAt should not change during update")
	}
}

func TestSQLiteClientRepository_Update_NotFound(t *testing.T) {
	repo, cleanup := setupTestClientRepo(t)
	defer cleanup()

	ctx := context.Background()
	client := createTestClient()
	client.ID = "non-existent-client"

	err := repo.Update(ctx, client)
	if err == nil {
		t.Error("Expected error when updating non-existent client, got nil")
	}
	if err.Error() != "client with ID non-existent-client not found" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestSQLiteClientRepository_Delete(t *testing.T) {
	repo, cleanup := setupTestClientRepo(t)
	defer cleanup()

	ctx := context.Background()
	client := createTestClient()

	// Create the client
	err := repo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify it exists
	retrieved, err := repo.GetByID(ctx, client.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve client: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Client not found after creating")
	}

	// Delete it
	err = repo.Delete(ctx, client.ID)
	if err != nil {
		t.Fatalf("Failed to delete client: %v", err)
	}

	// Verify it's gone
	retrieved, err = repo.GetByID(ctx, client.ID)
	if err != nil {
		t.Fatalf("Unexpected error after deletion: %v", err)
	}
	if retrieved != nil {
		t.Error("Client still exists after deletion")
	}
}

func TestSQLiteClientRepository_Delete_NotFound(t *testing.T) {
	repo, cleanup := setupTestClientRepo(t)
	defer cleanup()

	ctx := context.Background()

	err := repo.Delete(ctx, "non-existent-client")
	if err == nil {
		t.Error("Expected error when deleting non-existent client, got nil")
	}
	if err.Error() != "client with ID non-existent-client not found" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestSQLiteClientRepository_TimeConversion(t *testing.T) {
	repo, cleanup := setupTestClientRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()

	client := &Client{
		ID:                    "time-test-client",
		SecretHash:            "hash",
		SecretSalt:            "salt",
		CreatedAt:             now,
		UpdatedAt:             now.Add(5 * time.Minute),
		EncryptedMek:          "mek",
		KeyDerivationSalt:     "kds",
		StorageLimitMegabytes: 1024,
	}

	err := repo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, client.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve client: %v", err)
	}

	// Verify timestamps are preserved with precision
	if !retrieved.CreatedAt.Equal(client.CreatedAt) {
		t.Errorf("CreatedAt not preserved: expected %v, got %v", client.CreatedAt, retrieved.CreatedAt)
	}
	if !retrieved.UpdatedAt.Equal(client.UpdatedAt) {
		t.Errorf("UpdatedAt not preserved: expected %v, got %v", client.UpdatedAt, retrieved.UpdatedAt)
	}
}
