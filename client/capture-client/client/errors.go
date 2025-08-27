package client

import "fmt"

// UploadServerError represents an error response from the server during upload
type UploadServerError struct {
	IsRecoverable bool
	InnerError    error
}

func (e *UploadServerError) Error() string {
	if e.InnerError != nil {
		return fmt.Sprintf("Error during upload: %v", e.InnerError)
	}
	return "Error during upload"
}

// NewUploadServerError creates a new UploadServerError
func NewUploadServerError(isRecoverable bool, inner error) *UploadServerError {
	return &UploadServerError{
		IsRecoverable: isRecoverable,
		InnerError:    inner,
	}
}

// NewRecoverableUploadError creates a new recoverable UploadServerError
func NewRecoverableUploadError(inner error) *UploadServerError {
	return &UploadServerError{
		IsRecoverable: true,
		InnerError:    inner,
	}
}

// NewNonRecoverableUploadError creates a new non-recoverable UploadServerError
func NewNonRecoverableUploadError(inner error) *UploadServerError {
	return &UploadServerError{
		IsRecoverable: false,
		InnerError:    inner,
	}
}

// IsUploadServerError checks if the error is an UploadServerError
func IsUploadServerError(err error) bool {
	_, ok := err.(*UploadServerError)
	return ok
}

// IsRecoverableUploadError returns true if the error is recoverable (not a client-side error)
func IsRecoverableUploadError(err error) bool {
	if e, ok := err.(*UploadServerError); ok {
		return e.IsRecoverable
	}
	return false
}
