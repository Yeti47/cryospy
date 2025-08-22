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
		storage_limit_megabytes INTEGER NOT NULL,
		is_disabled INTEGER NOT NULL DEFAULT 0,
		clip_duration_seconds INTEGER NOT NULL DEFAULT 60,
		motion_only INTEGER NOT NULL DEFAULT 0,
		grayscale INTEGER NOT NULL DEFAULT 0,
		downscale_resolution TEXT NOT NULL DEFAULT '',
		output_format TEXT NOT NULL DEFAULT 'mp4',
		output_codec TEXT NOT NULL DEFAULT 'libx264',
		video_bitrate TEXT NOT NULL DEFAULT '1000k',
		motion_min_area INTEGER NOT NULL DEFAULT 1000,
		motion_max_frames INTEGER NOT NULL DEFAULT 300,
		motion_warm_up_frames INTEGER NOT NULL DEFAULT 30,
		capture_codec TEXT NOT NULL DEFAULT 'MJPG',
		capture_frame_rate REAL NOT NULL DEFAULT 15.0
	);`

	_, err := r.db.Exec(createClientsTable)
	if err != nil {
		return err
	}

	// Add new columns if they don't exist, to support migration from older versions
	db.AddColumn(r.db, "clients", "is_disabled", "INTEGER NOT NULL DEFAULT 0")
	db.AddColumn(r.db, "clients", "clip_duration_seconds", "INTEGER NOT NULL DEFAULT 60")
	db.AddColumn(r.db, "clients", "motion_only", "INTEGER NOT NULL DEFAULT 0")
	db.AddColumn(r.db, "clients", "grayscale", "INTEGER NOT NULL DEFAULT 0")
	db.AddColumn(r.db, "clients", "downscale_resolution", "TEXT NOT NULL DEFAULT ''")
	db.AddColumn(r.db, "clients", "output_format", "TEXT NOT NULL DEFAULT 'mp4'")
	db.AddColumn(r.db, "clients", "output_codec", "TEXT NOT NULL DEFAULT 'libx264'")
	db.AddColumn(r.db, "clients", "video_bitrate", "TEXT NOT NULL DEFAULT '1000k'")
	db.AddColumn(r.db, "clients", "motion_min_area", "INTEGER NOT NULL DEFAULT 1000")
	db.AddColumn(r.db, "clients", "motion_max_frames", "INTEGER NOT NULL DEFAULT 300")
	db.AddColumn(r.db, "clients", "motion_warm_up_frames", "INTEGER NOT NULL DEFAULT 30")
	db.AddColumn(r.db, "clients", "capture_codec", "TEXT NOT NULL DEFAULT 'MJPG'")
	db.AddColumn(r.db, "clients", "capture_frame_rate", "REAL NOT NULL DEFAULT 15.0")

	return nil
}

// GetByID retrieves a Client by its ID
func (r *SQLiteClientRepository) GetByID(ctx context.Context, id string) (*Client, error) {
	query := `
	SELECT id, secret_hash, secret_salt, created_at, updated_at, encrypted_mek, key_derivation_salt, storage_limit_megabytes,
		is_disabled, clip_duration_seconds, motion_only, grayscale, downscale_resolution,
		output_format, output_codec, video_bitrate,
		motion_min_area, motion_max_frames, motion_warm_up_frames,
		capture_codec, capture_frame_rate
	FROM clients WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)

	client := &Client{}
	var createdAtStr, updatedAtStr string
	err := row.Scan(
		&client.ID, &client.SecretHash, &client.SecretSalt, &createdAtStr, &updatedAtStr,
		&client.EncryptedMek, &client.KeyDerivationSalt, &client.StorageLimitMegabytes,
		&client.IsDisabled, &client.ClipDurationSeconds, &client.MotionOnly, &client.Grayscale, &client.DownscaleResolution,
		&client.OutputFormat, &client.OutputCodec, &client.VideoBitRate,
		&client.MotionMinArea, &client.MotionMaxFrames, &client.MotionWarmUpFrames,
		&client.CaptureCodec, &client.CaptureFrameRate,
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
	SELECT id, secret_hash, secret_salt, created_at, updated_at, encrypted_mek, key_derivation_salt, storage_limit_megabytes,
		is_disabled, clip_duration_seconds, motion_only, grayscale, downscale_resolution,
		output_format, output_codec, video_bitrate,
		motion_min_area, motion_max_frames, motion_warm_up_frames,
		capture_codec, capture_frame_rate
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
			&client.IsDisabled, &client.ClipDurationSeconds, &client.MotionOnly, &client.Grayscale, &client.DownscaleResolution,
			&client.OutputFormat, &client.OutputCodec, &client.VideoBitRate,
			&client.MotionMinArea, &client.MotionMaxFrames, &client.MotionWarmUpFrames,
			&client.CaptureCodec, &client.CaptureFrameRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client row: %w", err)
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
	INSERT INTO clients (id, secret_hash, secret_salt, created_at, updated_at, encrypted_mek, key_derivation_salt, storage_limit_megabytes,
		is_disabled, clip_duration_seconds, motion_only, grayscale, downscale_resolution,
		output_format, output_codec, video_bitrate,
		motion_min_area, motion_max_frames, motion_warm_up_frames,
		capture_codec, capture_frame_rate)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		client.ID, client.SecretHash, client.SecretSalt,
		db.TimeToString(client.CreatedAt), db.TimeToString(client.UpdatedAt),
		client.EncryptedMek, client.KeyDerivationSalt, client.StorageLimitMegabytes,
		client.IsDisabled, client.ClipDurationSeconds, client.MotionOnly, client.Grayscale, client.DownscaleResolution,
		client.OutputFormat, client.OutputCodec, client.VideoBitRate,
		client.MotionMinArea, client.MotionMaxFrames, client.MotionWarmUpFrames,
		client.CaptureCodec, client.CaptureFrameRate,
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
		key_derivation_salt = ?, storage_limit_megabytes = ?, is_disabled = ?,
		clip_duration_seconds = ?, motion_only = ?, grayscale = ?, downscale_resolution = ?,
		output_format = ?, output_codec = ?, video_bitrate = ?,
		motion_min_area = ?, motion_max_frames = ?, motion_warm_up_frames = ?,
		capture_codec = ?, capture_frame_rate = ?
	WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		client.SecretHash, client.SecretSalt, db.TimeToString(client.UpdatedAt),
		client.EncryptedMek, client.KeyDerivationSalt, client.StorageLimitMegabytes, client.IsDisabled,
		client.ClipDurationSeconds, client.MotionOnly, client.Grayscale, client.DownscaleResolution,
		client.OutputFormat, client.OutputCodec, client.VideoBitRate,
		client.MotionMinArea, client.MotionMaxFrames, client.MotionWarmUpFrames,
		client.CaptureCodec, client.CaptureFrameRate,
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
