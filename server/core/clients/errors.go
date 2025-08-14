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

type ClientValidationError struct {
	Message string
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

func (e *ClientValidationError) Error() string {
	return "Client validation failed: " + e.Message
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

func IsClientValidationError(err error) bool {
	_, ok := err.(*ClientValidationError)
	return ok
}

func IsClientVerificationError(err error) bool {
	_, ok := err.(*ClientVerificationError)
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

func NewClientValidationError(message string) error {
	return &ClientValidationError{Message: message}
}
