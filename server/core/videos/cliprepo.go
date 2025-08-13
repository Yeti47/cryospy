package videos

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yeti47/cryospy/server/core/ccc/db"
)

// ClipRepository defines the interface for CRUD operations on Clip entities
type ClipRepository interface {
	// GetByID retrieves a Clip by its ID
	GetByID(ctx context.Context, id string) (*Clip, error)

	// Query retrieves Clips based on the provided query parameters
	// Returns clips and total count of matching records (before pagination)
	Query(ctx context.Context, query ClipQuery) ([]*Clip, int, error)

	// Add stores a new Clip in the repository
	Add(ctx context.Context, clip *Clip) error

	// Delete removes a Clip by its ID
	Delete(ctx context.Context, id string) error

	// QueryInfo retrieves ClipInfo (metadata only) based on the provided query parameters
	// Returns clip infos and total count of matching records (before pagination)
	QueryInfo(ctx context.Context, query ClipQuery) ([]*ClipInfo, int, error)

	// GetThumbnailByID retrieves the thumbnail data with metadata for a Clip by its ID
	GetThumbnailByID(ctx context.Context, id string) (*Thumbnail, error)
}

// SQLiteClipRepository implements ClipRepository using SQLite
type SQLiteClipRepository struct {
	db *sql.DB
}

// NewSQLiteClipRepository creates a new SQLite-based ClipRepository
func NewSQLiteClipRepository(db *sql.DB) (*SQLiteClipRepository, error) {
	repo := &SQLiteClipRepository{db: db}
	if err := repo.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return repo, nil
}

// createTables ensures that the required tables exist
func (r *SQLiteClipRepository) createTables() error {
	createClipsTable := `
	CREATE TABLE IF NOT EXISTS clips (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		title TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		duration INTEGER NOT NULL,
		has_motion INTEGER NOT NULL,
		encrypted_video BLOB,
		video_width INTEGER NOT NULL,
		video_height INTEGER NOT NULL,
		video_mime_type TEXT NOT NULL,
		encrypted_thumbnail BLOB,
		thumbnail_width INTEGER NOT NULL,
		thumbnail_height INTEGER NOT NULL,
		thumbnail_mime_type TEXT NOT NULL
	);`

	_, err := r.db.Exec(createClipsTable)
	return err
}

// GetByID retrieves a Clip by its ID
func (r *SQLiteClipRepository) GetByID(ctx context.Context, id string) (*Clip, error) {
	query := `
	SELECT id, client_id, title, timestamp, duration, has_motion, encrypted_video, video_width, video_height, video_mime_type,
		   encrypted_thumbnail, thumbnail_width, thumbnail_height, thumbnail_mime_type
	FROM clips WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)

	clip := &Clip{}
	var durationNanos int64
	var timestampStr string
	var hasMotionInt int
	err := row.Scan(
		&clip.ID, &clip.ClientID, &clip.Title, &timestampStr, &durationNanos, &hasMotionInt, &clip.EncryptedVideo,
		&clip.VideoWidth, &clip.VideoHeight, &clip.VideoMimeType,
		&clip.EncryptedThumbnail, &clip.ThumbnailWidth, &clip.ThumbnailHeight, &clip.ThumbnailMimeType,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get clip by ID: %w", err)
	}

	// Convert string timestamp back to time.Time
	clip.TimeStamp, err = db.StringToTime(timestampStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	clip.Duration = time.Duration(durationNanos)
	clip.HasMotion = db.IntToBool(hasMotionInt)
	return clip, nil
}

// Query retrieves Clips based on the provided query parameters
func (r *SQLiteClipRepository) Query(ctx context.Context, query ClipQuery) ([]*Clip, int, error) {
	// First, get the total count without pagination
	totalCount, err := r.getQueryCount(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Then get the actual data with pagination
	sqlQuery, args := r.buildQuerySQL(query, false)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query clips: %w", err)
	}
	defer rows.Close()

	var clips []*Clip
	for rows.Next() {
		clip := &Clip{}
		var durationNanos int64
		var timestampStr string
		var hasMotionInt int
		err := rows.Scan(
			&clip.ID, &clip.ClientID, &clip.Title, &timestampStr, &durationNanos, &hasMotionInt, &clip.EncryptedVideo,
			&clip.VideoWidth, &clip.VideoHeight, &clip.VideoMimeType,
			&clip.EncryptedThumbnail, &clip.ThumbnailWidth, &clip.ThumbnailHeight, &clip.ThumbnailMimeType,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan clip: %w", err)
		}

		// Convert string timestamp back to time.Time
		clip.TimeStamp, err = db.StringToTime(timestampStr)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		clip.Duration = time.Duration(durationNanos)
		clip.HasMotion = db.IntToBool(hasMotionInt)
		clips = append(clips, clip)
	}

	return clips, totalCount, rows.Err()
}

// Add stores a new Clip in the repository
func (r *SQLiteClipRepository) Add(ctx context.Context, clip *Clip) error {
	query := `
	INSERT INTO clips (id, client_id, title, timestamp, duration, has_motion, encrypted_video, video_width, video_height, video_mime_type,
					   encrypted_thumbnail, thumbnail_width, thumbnail_height, thumbnail_mime_type)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Convert bool to int for has_motion
	hasMotionInt := db.BoolToInt(clip.HasMotion)

	_, err := r.db.ExecContext(ctx, query,
		clip.ID, clip.ClientID, clip.Title, db.TimeToString(clip.TimeStamp), int64(clip.Duration), hasMotionInt, clip.EncryptedVideo,
		clip.VideoWidth, clip.VideoHeight, clip.VideoMimeType,
		clip.EncryptedThumbnail, clip.ThumbnailWidth, clip.ThumbnailHeight, clip.ThumbnailMimeType,
	)
	if err != nil {
		return fmt.Errorf("failed to add clip: %w", err)
	}

	return nil
}

// Delete removes a Clip by its ID
func (r *SQLiteClipRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM clips WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete clip: %w", err)
	}

	return nil
}

// QueryInfo retrieves ClipInfo (metadata only) based on the provided query parameters
func (r *SQLiteClipRepository) QueryInfo(ctx context.Context, query ClipQuery) ([]*ClipInfo, int, error) {
	// First, get the total count without pagination
	totalCount, err := r.getQueryCount(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Then get the actual data with pagination
	sqlQuery, args := r.buildQuerySQL(query, true)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query clip info: %w", err)
	}
	defer rows.Close()

	var clipInfos []*ClipInfo
	for rows.Next() {
		clipInfo := &ClipInfo{}
		var durationNanos int64
		var timestampStr string
		var hasMotionInt int
		err := rows.Scan(
			&clipInfo.ID, &clipInfo.ClientID, &clipInfo.Title, &timestampStr, &durationNanos, &hasMotionInt,
			&clipInfo.VideoWidth, &clipInfo.VideoHeight, &clipInfo.VideoMimeType,
			&clipInfo.ThumbnailWidth, &clipInfo.ThumbnailHeight, &clipInfo.ThumbnailMimeType,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan clip info: %w", err)
		}

		// Convert string timestamp back to time.Time
		clipInfo.TimeStamp, err = db.StringToTime(timestampStr)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		clipInfo.Duration = time.Duration(durationNanos)
		clipInfo.HasMotion = db.IntToBool(hasMotionInt)
		clipInfos = append(clipInfos, clipInfo)
	}

	return clipInfos, totalCount, rows.Err()
}

// GetThumbnailByID retrieves the thumbnail data with metadata for a Clip by its ID
func (r *SQLiteClipRepository) GetThumbnailByID(ctx context.Context, id string) (*Thumbnail, error) {
	query := `
	SELECT encrypted_thumbnail, thumbnail_width, thumbnail_height, thumbnail_mime_type
	FROM clips WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)

	thumbnail := &Thumbnail{}
	err := row.Scan(&thumbnail.Data, &thumbnail.Width, &thumbnail.Height, &thumbnail.MimeType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get thumbnail by ID: %w", err)
	}

	return thumbnail, nil
}

// getQueryCount returns the total count of records matching the query (without pagination)
func (r *SQLiteClipRepository) getQueryCount(ctx context.Context, query ClipQuery) (int, error) {
	sqlQuery := "SELECT COUNT(*) FROM clips"
	var conditions []string
	var args []any

	if query.ClientID != "" {
		conditions = append(conditions, "client_id = ?")
		args = append(args, query.ClientID)
	}

	if query.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, db.TimeToString(*query.StartTime))
	}

	if query.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, db.TimeToString(*query.EndTime))
	}

	if query.HasMotion != nil {
		conditions = append(conditions, "has_motion = ?")
		args = append(args, db.BoolToInt(*query.HasMotion))
	}

	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %w", err)
	}

	return count, nil
}

// buildQuerySQL builds the SQL query and arguments based on ClipQuery parameters
func (r *SQLiteClipRepository) buildQuerySQL(query ClipQuery, metadataOnly bool) (string, []interface{}) {
	var selectClause string
	if metadataOnly {
		selectClause = `SELECT id, client_id, title, timestamp, duration, has_motion, video_width, video_height, video_mime_type,
						thumbnail_width, thumbnail_height, thumbnail_mime_type`
	} else {
		selectClause = `SELECT id, client_id, title, timestamp, duration, has_motion, encrypted_video, video_width, video_height, video_mime_type,
						encrypted_thumbnail, thumbnail_width, thumbnail_height, thumbnail_mime_type`
	}

	sqlQuery := selectClause + " FROM clips"
	var conditions []string
	var args []any

	if query.ClientID != "" {
		conditions = append(conditions, "client_id = ?")
		args = append(args, query.ClientID)
	}

	if query.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, db.TimeToString(*query.StartTime))
	}

	if query.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, db.TimeToString(*query.EndTime))
	}

	if query.HasMotion != nil {
		conditions = append(conditions, "has_motion = ?")
		args = append(args, db.BoolToInt(*query.HasMotion))
	}

	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	sqlQuery += " ORDER BY timestamp DESC"

	// Add pagination if specified
	if query.PageSize > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, query.PageSize)
		if query.Page > 1 {
			sqlQuery += " OFFSET ?"
			args = append(args, (query.Page-1)*query.PageSize)
		}
	}

	return sqlQuery, args
}
