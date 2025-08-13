package sessions

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/yeti47/cryospy/server/core/encryption"
)

const (
	sessionName = "cryospy-dashboard-session"
	mekKey      = "mek"
)

// GorillaMekStore implements the MekStore interface using gorilla sessions
type GorillaMekStore struct {
	store   sessions.Store
	request *gin.Context
}

// NewGorillaMekStore creates a new GorillaMekStore for a specific request
func NewGorillaMekStore(store sessions.Store, c *gin.Context) encryption.MekStore {
	return &GorillaMekStore{
		store:   store,
		request: c,
	}
}

// GetMek retrieves the MEK from the session
func (s *GorillaMekStore) GetMek() ([]byte, error) {
	session, err := s.store.Get(s.request.Request, sessionName)
	if err != nil {
		return nil, err
	}

	value, ok := session.Values[mekKey]
	if !ok {
		return nil, errors.New("MEK not found in session")
	}

	mek, ok := value.([]byte)
	if !ok {
		return nil, errors.New("invalid MEK format in session")
	}

	return mek, nil
}

// SetMek sets the MEK in the session
func (s *GorillaMekStore) SetMek(mekValue []byte) error {
	session, err := s.store.Get(s.request.Request, sessionName)
	if err != nil {
		return err
	}

	session.Values[mekKey] = mekValue
	return session.Save(s.request.Request, s.request.Writer)
}

// ClearMek removes the MEK from the session
func (s *GorillaMekStore) ClearMek() error {
	session, err := s.store.Get(s.request.Request, sessionName)
	if err != nil {
		return err
	}

	delete(session.Values, mekKey)
	return session.Save(s.request.Request, s.request.Writer)
}
