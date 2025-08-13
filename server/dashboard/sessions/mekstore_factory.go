package sessions

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/yeti47/cryospy/server/core/encryption"
)

// MekStoreFactory is a function that creates a MekStore for a given request context.
type MekStoreFactory func(c *gin.Context) encryption.MekStore

// NewMekStoreFactory creates a new MekStoreFactory.
func NewMekStoreFactory(store sessions.Store) MekStoreFactory {
	return func(c *gin.Context) encryption.MekStore {
		return NewGorillaMekStore(store, c)
	}
}
