package encryption

import (
	"testing"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/db"
)

func TestNewSQLiteMekRepository(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo, err := NewSQLiteMekRepository(testDB)
	if err != nil {
		t.Fatalf("NewSQLiteMekRepository() failed: %v", err)
	}

	if repo == nil {
		t.Fatal("NewSQLiteMekRepository() returned nil")
	}

	if repo.db != testDB {
		t.Error("Repository database reference is incorrect")
	}
}

func TestMekRepository_Create(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo, err := NewSQLiteMekRepository(testDB)
	if err != nil {
		t.Fatalf("NewSQLiteMekRepository() failed: %v", err)
	}

	now := time.Now().UTC()
	mek := &Mek{
		ID:                     "test-mek-id",
		EncryptedEncryptionKey: "encrypted-key-data",
		EncryptionKeySalt:      "key-salt-data",
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	err = repo.Create(mek)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify the MEK was created
	retrieved, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() failed after Create(): %v", err)
	}

	if retrieved == nil {
		t.Fatal("Get() returned nil after Create()")
	}

	if retrieved.ID != mek.ID {
		t.Errorf("Expected ID %s, got %s", mek.ID, retrieved.ID)
	}

	if retrieved.EncryptedEncryptionKey != mek.EncryptedEncryptionKey {
		t.Errorf("Expected EncryptedEncryptionKey %s, got %s", mek.EncryptedEncryptionKey, retrieved.EncryptedEncryptionKey)
	}
}

func TestMekRepository_Create_DuplicateError(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo, err := NewSQLiteMekRepository(testDB)
	if err != nil {
		t.Fatalf("NewSQLiteMekRepository() failed: %v", err)
	}

	now := time.Now().UTC()
	mek := &Mek{
		ID:                     "test-mek-id",
		EncryptedEncryptionKey: "encrypted-key-data",
		EncryptionKeySalt:      "key-salt-data",
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	// Create first MEK
	err = repo.Create(mek)
	if err != nil {
		t.Fatalf("First Create() failed: %v", err)
	}

	// Try to create second MEK - should fail
	mek2 := &Mek{
		ID:                     "test-mek-id-2",
		EncryptedEncryptionKey: "encrypted-key-data-2",
		EncryptionKeySalt:      "key-salt-data-2",
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	err = repo.Create(mek2)
	if err == nil {
		t.Error("Create() should fail when MEK already exists")
	}
}

func TestMekRepository_Get_Empty(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo, err := NewSQLiteMekRepository(testDB)
	if err != nil {
		t.Fatalf("NewSQLiteMekRepository() failed: %v", err)
	}

	// Get from empty database
	mek, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if mek != nil {
		t.Error("Get() should return nil when no MEK exists")
	}
}

func TestMekRepository_Update(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo, err := NewSQLiteMekRepository(testDB)
	if err != nil {
		t.Fatalf("NewSQLiteMekRepository() failed: %v", err)
	}

	now := time.Now().UTC()
	mek := &Mek{
		ID:                     "test-mek-id",
		EncryptedEncryptionKey: "encrypted-key-data",
		EncryptionKeySalt:      "key-salt-data",
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	// Create MEK first
	err = repo.Create(mek)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Update MEK
	laterTime := time.Now().Add(time.Hour).UTC()
	mek.EncryptedEncryptionKey = "updated-encrypted-key-data"
	mek.EncryptionKeySalt = "updated-key-salt-data"
	mek.UpdatedAt = laterTime

	err = repo.Update(mek)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify the update
	retrieved, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() failed after Update(): %v", err)
	}

	if retrieved.EncryptedEncryptionKey != "updated-encrypted-key-data" {
		t.Errorf("Expected updated EncryptedEncryptionKey %s, got %s", "updated-encrypted-key-data", retrieved.EncryptedEncryptionKey)
	}

	if retrieved.EncryptionKeySalt != "updated-key-salt-data" {
		t.Errorf("Expected updated EncryptionKeySalt %s, got %s", "updated-key-salt-data", retrieved.EncryptionKeySalt)
	}

	if retrieved.UpdatedAt != laterTime {
		t.Errorf("Expected updated UpdatedAt %s, got %s", laterTime, retrieved.UpdatedAt)
	}

	// CreatedAt should remain unchanged
	if retrieved.CreatedAt != now {
		t.Errorf("CreatedAt should not change during update. Expected %s, got %s", now, retrieved.CreatedAt)
	}
}

func TestMekRepository_Update_NotFound(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo, err := NewSQLiteMekRepository(testDB)
	if err != nil {
		t.Fatalf("NewSQLiteMekRepository() failed: %v", err)
	}

	now := time.Now().UTC()
	mek := &Mek{
		ID:                     "non-existent-mek-id",
		EncryptedEncryptionKey: "encrypted-key-data",
		EncryptionKeySalt:      "key-salt-data",
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	err = repo.Update(mek)
	if err == nil {
		t.Error("Update() should fail when MEK does not exist")
	}
}

func TestMekRepository_Delete(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo, err := NewSQLiteMekRepository(testDB)
	if err != nil {
		t.Fatalf("NewSQLiteMekRepository() failed: %v", err)
	}

	now := time.Now().UTC()
	mek := &Mek{
		ID:                     "test-mek-id",
		EncryptedEncryptionKey: "encrypted-key-data",
		EncryptionKeySalt:      "key-salt-data",
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	// Create MEK first
	err = repo.Create(mek)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify MEK exists
	retrieved, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("MEK should exist before delete")
	}

	// Delete MEK
	err = repo.Delete()
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify MEK is gone
	retrieved, err = repo.Get()
	if err != nil {
		t.Fatalf("Get() failed after Delete(): %v", err)
	}
	if retrieved != nil {
		t.Error("MEK should be nil after delete")
	}
}

func TestMekRepository_Delete_NotFound(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo, err := NewSQLiteMekRepository(testDB)
	if err != nil {
		t.Fatalf("NewSQLiteMekRepository() failed: %v", err)
	}

	// Try to delete from empty database
	err = repo.Delete()
	if err == nil {
		t.Error("Delete() should fail when no MEK exists")
	}
}

func TestMekRepository_CreateTables(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo := &SQLiteMekRepository{db: testDB}
	err = repo.createTables()
	if err != nil {
		t.Fatalf("createTables() failed: %v", err)
	}

	// Verify table exists by trying to query it
	_, err = testDB.Exec("SELECT COUNT(*) FROM meks")
	if err != nil {
		t.Fatalf("meks table was not created properly: %v", err)
	}
}

func TestMekRepository_IntegrationCRUD(t *testing.T) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	repo, err := NewSQLiteMekRepository(testDB)
	if err != nil {
		t.Fatalf("NewSQLiteMekRepository() failed: %v", err)
	}

	// 1. Verify empty state
	mek, err := repo.Get()
	if err != nil {
		t.Fatalf("Initial Get() failed: %v", err)
	}
	if mek != nil {
		t.Error("Database should be empty initially")
	}

	// 2. Create MEK
	now := time.Now().UTC()
	originalMek := &Mek{
		ID:                     "integration-test-mek",
		EncryptedEncryptionKey: "original-encrypted-key",
		EncryptionKeySalt:      "original-salt",
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	err = repo.Create(originalMek)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// 3. Read MEK
	retrieved, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() after Create() failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get() should return MEK after Create()")
	}

	// 4. Update MEK
	laterTime := time.Now().Add(time.Hour).UTC()
	retrieved.EncryptedEncryptionKey = "updated-encrypted-key"
	retrieved.EncryptionKeySalt = "updated-salt"
	retrieved.UpdatedAt = laterTime

	err = repo.Update(retrieved)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// 5. Verify update
	updated, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() after Update() failed: %v", err)
	}
	if updated.EncryptedEncryptionKey != "updated-encrypted-key" {
		t.Error("MEK was not updated properly")
	}

	// 6. Delete MEK
	err = repo.Delete()
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// 7. Verify deletion
	deleted, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() after Delete() failed: %v", err)
	}
	if deleted != nil {
		t.Error("MEK should be nil after deletion")
	}
}
