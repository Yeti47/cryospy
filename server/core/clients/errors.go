package clients

// Error types for client operations
type ClientAlreadyExistsError struct {
	ID string
}

type ClientNotFoundError struct {
	ID string
}

type ClientVerificationError struct {
	ClientID string
}

func (e *ClientAlreadyExistsError) Error() string {
	return "Client already exists: " + e.ID
}

func (e *ClientNotFoundError) Error() string {
	return "Client not found: " + e.ID
}

func (e *ClientVerificationError) Error() string {
	return "Client verification failed for ID: " + e.ClientID
}

// helper functions for error handling

func IsClientAlreadyExistsError(err error) bool {
	_, ok := err.(*ClientAlreadyExistsError)
	return ok
}

func IsClientNotFoundError(err error) bool {
	_, ok := err.(*ClientNotFoundError)
	return ok
}

// helper function to create a new ClientAlreadyExistsError
func NewClientAlreadyExistsError(id string) error {
	return &ClientAlreadyExistsError{ID: id}
}

func NewClientNotFoundError(id string) error {
	return &ClientNotFoundError{ID: id}
}

func NewClientVerificationError(clientID string) error {
	return &ClientVerificationError{ClientID: clientID}
}
