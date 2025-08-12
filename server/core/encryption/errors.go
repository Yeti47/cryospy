package encryption

import "fmt"

// create an error type that indicates that no MEK exists
type MekNotFoundError struct {
}

func (e *MekNotFoundError) Error() string {
	return "MEK not found"
}

// create an error type that indicates that a MEK already exists
type MekAlreadyExistsError struct {
	ID string
}

func (e *MekAlreadyExistsError) Error() string {
	return fmt.Sprintf("MEK already exists with ID: %s", e.ID)
}

// helper functions for error handling
func IsMekNotFoundError(err error) bool {
	_, ok := err.(*MekNotFoundError)
	return ok
}
func IsMekAlreadyExistsError(err error) bool {
	_, ok := err.(*MekAlreadyExistsError)
	return ok
}

// factory functions for mek-related errors
func NewMekNotFoundError() error {
	return &MekNotFoundError{}
}
func NewMekAlreadyExistsError(id string) error {
	return &MekAlreadyExistsError{ID: id}
}
