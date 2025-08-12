package clients

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yeti47/cryospy/server/core/ccc/db"
)

type ClientRepository interface {
	// GetByID retrieves a Client by its ID
	GetByID(ctx context.Context, id string) (*Client, error)
	// GetAll retrieves all Clients
	GetAll(ctx context.Context) ([]*Client, error)
	// Create adds a new Client to the repository
	Create(ctx context.Context, client *Client) error
	// Update modifies an existing Client in the repository
	Update(ctx context.Context, client *Client) error
	// Delete removes a Client from the repository
	Delete(ctx context.Context, id string) error
}

// SQLiteClientRepository implements ClientRepository using SQLite
type SQLiteClientRepository struct {
	db *sql.DB
}

// NewSQLiteClientRepository creates a new SQLite-based ClientRepository
func NewSQLiteClientRepository(db *sql.DB) (*SQLiteClientRepository, error) {
	repo := &SQLiteClientRepository{db: db}
	if err := repo.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return repo, nil
}

// createTables ensures that the required tables exist
func (r *SQLiteClientRepository) createTables() error {
	createClientsTable := `
	CREATE TABLE IF NOT EXISTS clients (
		id TEXT PRIMARY KEY,
		secret_hash TEXT NOT NULL,
		secret_salt TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		encrypted_mek TEXT NOT NULL,
		key_derivation_salt TEXT NOT NULL,
		storage_limit_megabytes INTEGER NOT NULL
	);`

	_, err := r.db.Exec(createClientsTable)
	return err
}

// GetByID retrieves a Client by its ID
func (r *SQLiteClientRepository) GetByID(ctx context.Context, id string) (*Client, error) {
	query := `
	SELECT id, secret_hash, secret_salt, created_at, updated_at, encrypted_mek, key_derivation_salt, storage_limit_megabytes
	FROM clients WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)

	client := &Client{}
	var createdAtStr, updatedAtStr string
	err := row.Scan(
		&client.ID, &client.SecretHash, &client.SecretSalt, &createdAtStr, &updatedAtStr,
		&client.EncryptedMek, &client.KeyDerivationSalt, &client.StorageLimitMegabytes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get client by ID: %w", err)
	}

	// Convert string timestamps back to time.Time
	client.CreatedAt, err = db.StringToTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at timestamp: %w", err)
	}

	client.UpdatedAt, err = db.StringToTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at timestamp: %w", err)
	}

	return client, nil
}

// GetAll retrieves all Clients
func (r *SQLiteClientRepository) GetAll(ctx context.Context) ([]*Client, error) {
	query := `
	SELECT id, secret_hash, secret_salt, created_at, updated_at, encrypted_mek, key_derivation_salt, storage_limit_megabytes
	FROM clients ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query clients: %w", err)
	}
	defer rows.Close()

	var clients []*Client
	for rows.Next() {
		client := &Client{}
		var createdAtStr, updatedAtStr string
		err := rows.Scan(
			&client.ID, &client.SecretHash, &client.SecretSalt, &createdAtStr, &updatedAtStr,
			&client.EncryptedMek, &client.KeyDerivationSalt, &client.StorageLimitMegabytes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}

		// Convert string timestamps back to time.Time
		client.CreatedAt, err = db.StringToTime(createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at timestamp: %w", err)
		}

		client.UpdatedAt, err = db.StringToTime(updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at timestamp: %w", err)
		}

		clients = append(clients, client)
	}

	return clients, rows.Err()
}

// Create adds a new Client to the repository
func (r *SQLiteClientRepository) Create(ctx context.Context, client *Client) error {
	query := `
	INSERT INTO clients (id, secret_hash, secret_salt, created_at, updated_at, encrypted_mek, key_derivation_salt, storage_limit_megabytes)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		client.ID, client.SecretHash, client.SecretSalt,
		db.TimeToString(client.CreatedAt), db.TimeToString(client.UpdatedAt),
		client.EncryptedMek, client.KeyDerivationSalt, client.StorageLimitMegabytes,
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	return nil
}

// Update modifies an existing Client in the repository
func (r *SQLiteClientRepository) Update(ctx context.Context, client *Client) error {
	query := `
	UPDATE clients 
	SET secret_hash = ?, secret_salt = ?, updated_at = ?, encrypted_mek = ?, 
		key_derivation_salt = ?, storage_limit_megabytes = ?
	WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		client.SecretHash, client.SecretSalt, db.TimeToString(client.UpdatedAt),
		client.EncryptedMek, client.KeyDerivationSalt, client.StorageLimitMegabytes,
		client.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update client: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("client with ID %s not found", client.ID)
	}

	return nil
}

// Delete removes a Client from the repository
func (r *SQLiteClientRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM clients WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("client with ID %s not found", id)
	}

	return nil
}
