package encryption

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yeti47/cryospy/server/core/ccc/db"
)

type MekRepository interface {
	Create(mek *Mek) error
	Get() (*Mek, error)
	Update(mek *Mek) error
	Delete() error
}

// SQLiteMekRepository implements MekRepository using SQLite
type SQLiteMekRepository struct {
	db *sql.DB
}

// NewSQLiteMekRepository creates a new SQLite-based MekRepository
func NewSQLiteMekRepository(db *sql.DB) (*SQLiteMekRepository, error) {
	repo := &SQLiteMekRepository{db: db}
	if err := repo.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return repo, nil
}

// createTables ensures that the required tables exist
func (r *SQLiteMekRepository) createTables() error {
	createMekTable := `
	CREATE TABLE IF NOT EXISTS meks (
		id TEXT PRIMARY KEY,
		encrypted_encryption_key TEXT NOT NULL,
		encryption_key_salt TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`

	_, err := r.db.Exec(createMekTable)
	return err
}

// Create adds a new MEK to the repository
// Since there can only be one MEK, this will fail if one already exists
func (r *SQLiteMekRepository) Create(mek *Mek) error {
	// Check if a MEK already exists
	existing, err := r.Get()
	if err != nil {
		return fmt.Errorf("failed to check for existing MEK: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("MEK already exists with ID: %s", existing.ID)
	}

	query := `
	INSERT INTO meks (id, encrypted_encryption_key, encryption_key_salt, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?)`

	_, err = r.db.Exec(query,
		mek.ID, mek.EncryptedEncryptionKey, mek.EncryptionKeySalt,
		db.TimeToString(mek.CreatedAt), db.TimeToString(mek.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to create MEK: %w", err)
	}

	return nil
}

// Get retrieves the single MEK from the repository
// Returns nil if no MEK exists (this is not an error)
func (r *SQLiteMekRepository) Get() (*Mek, error) {
	query := `
	SELECT id, encrypted_encryption_key, encryption_key_salt, created_at, updated_at
	FROM meks LIMIT 1`

	row := r.db.QueryRow(query)

	mek := &Mek{}
	var createdAtStr, updatedAtStr string
	err := row.Scan(
		&mek.ID, &mek.EncryptedEncryptionKey, &mek.EncryptionKeySalt,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No MEK found, not an error
		}
		return nil, fmt.Errorf("failed to get MEK: %w", err)
	}

	// Convert string timestamps back to time.Time
	mek.CreatedAt, err = db.StringToTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at timestamp: %w", err)
	}

	mek.UpdatedAt, err = db.StringToTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at timestamp: %w", err)
	}

	return mek, nil
}

// Update modifies the existing MEK in the repository
func (r *SQLiteMekRepository) Update(mek *Mek) error {
	query := `
	UPDATE meks 
	SET encrypted_encryption_key = ?, encryption_key_salt = ?, updated_at = ?
	WHERE id = ?`

	result, err := r.db.Exec(query,
		mek.EncryptedEncryptionKey, mek.EncryptionKeySalt, db.TimeToString(mek.UpdatedAt),
		mek.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update MEK: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("MEK with ID %s not found", mek.ID)
	}

	return nil
}

// Delete removes the MEK from the repository
func (r *SQLiteMekRepository) Delete() error {
	query := `DELETE FROM meks`

	result, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete MEK: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no MEK found to delete")
	}

	return nil
}
