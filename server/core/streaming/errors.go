package streaming

import "fmt"

type SegmentNotFoundError struct {
	ClipID   string
	ClientID string
}

func (e *SegmentNotFoundError) Error() string {
	return fmt.Sprintf("Segment not found for clip %s and client %s", e.ClipID, e.ClientID)
}

func IsSegmentNotFoundError(err error) bool {
	_, ok := err.(*SegmentNotFoundError)
	return ok
}

func NewSegmentNotFoundError(clipID, clientID string) error {
	return &SegmentNotFoundError{
		ClipID:   clipID,
		ClientID: clientID,
	}
}
