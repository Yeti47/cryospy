package encryption

// MekStore provides access to MEKs for authenticated admin users
// The implementation assumes the user has already been authenticated
// and retrieves the MEK from secure storage (e.g., session, cookie)
type MekStore interface {
	// GetMek retrieves the MEK for the authenticated admin user
	GetMek() ([]byte, error)
	// SetMek sets the MEK for the authenticated admin user
	SetMek(mek []byte) error
	// ClearMek removes the MEK for the authenticated admin user
	ClearMek() error
}
