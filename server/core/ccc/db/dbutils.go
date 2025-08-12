package db

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TimeToString converts a time.Time to RFC3339Nano string for database storage
func TimeToString(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}

// StringToTime converts an RFC3339Nano string from database to time.Time
func StringToTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}

// BoolToInt converts a boolean to integer for database storage (1 for true, 0 for false)
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// IntToBool converts an integer from database to boolean (1 = true, 0 = false)
func IntToBool(i int) bool {
	return i == 1
}

// BoolPtrToInt converts a *bool to integer for database storage
// Returns nil if the pointer is nil, otherwise converts the boolean value
func BoolPtrToInt(b *bool) *int {
	if b == nil {
		return nil
	}
	result := BoolToInt(*b)
	return &result
}

// TimePtrToString converts a *time.Time to string for database storage
// Returns nil if the pointer is nil, otherwise converts the time value
func TimePtrToString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	result := TimeToString(*t)
	return &result
}

// NewInMemoryDB creates a new in-memory SQLite database for testing
func NewInMemoryDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
