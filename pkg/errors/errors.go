package errors

import "errors"

// Standard error types
var (
	ErrProxyNotFound      = errors.New("proxy not found")
	ErrProxyAlreadyExists = errors.New("proxy already exists")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrPortInUse          = errors.New("port already in use")
	ErrConnectionFailed   = errors.New("connection failed")
	ErrTimeout            = errors.New("operation timeout")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
	ErrBadRequest         = errors.New("bad request")
	ErrInternalServer     = errors.New("internal server error")
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string
	Message string
	Err     error
}

func (e *ValidationError) Error() string {
	if e.Err != nil {
		return e.Field + ": " + e.Message + " (" + e.Err.Error() + ")"
	}
	return e.Field + ": " + e.Message
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string, err error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Err:     err,
	}
}

// ProxyError represents a proxy-specific error
type ProxyError struct {
	ProxyID string
	Message string
	Err     error
}

func (e *ProxyError) Error() string {
	if e.Err != nil {
		return "proxy[" + e.ProxyID + "]: " + e.Message + " (" + e.Err.Error() + ")"
	}
	return "proxy[" + e.ProxyID + "]: " + e.Message
}

func (e *ProxyError) Unwrap() error {
	return e.Err
}

// NewProxyError creates a new proxy error
func NewProxyError(proxyID, message string, err error) *ProxyError {
	return &ProxyError{
		ProxyID: proxyID,
		Message: message,
		Err:     err,
	}
}
