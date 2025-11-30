package gitbackedrest

import "net/http"

type APIBackend interface {
	GET(path string) ([]byte, *APIError)
	POST(path string, body []byte) *APIError
	PUT(path string, body []byte) *APIError
	DELETE(path string) *APIError

	// TODO: This signature probably doesn't represent patching effectively
	//PATCH(path string, body []byte) error

	// TODO: what options should be available?
	//OPTIONS(path string) error
}

var _ error = new(APIError)

type APIError struct {
	Message string
	Code    int
}

// Error implements error.
func (a *APIError) Error() string {
	return a.Message
}

// Client error sentinel values (4xx)
var (
	// ErrBadRequest indicates invalid syntax or validation errors in the request
	ErrBadRequest = &APIError{Message: "Bad Request", Code: http.StatusBadRequest}

	// ErrUnauthorized indicates missing or invalid authentication
	ErrUnauthorized = &APIError{Message: "Unauthorized", Code: http.StatusUnauthorized}

	// ErrForbidden indicates authenticated but not allowed to access this resource
	ErrForbidden = &APIError{Message: "Forbidden", Code: http.StatusForbidden}

	// ErrNotFound indicates resource or endpoint does not exist
	ErrNotFound = &APIError{Message: "Not Found", Code: http.StatusNotFound}

	// ErrMethodNotAllowed indicates HTTP method not supported for this endpoint
	ErrMethodNotAllowed = &APIError{Message: "Method Not Allowed", Code: http.StatusMethodNotAllowed}

	// ErrConflict indicates request conflicts with current state
	ErrConflict = &APIError{Message: "Conflict", Code: http.StatusConflict}

	// ErrGone indicates resource has been intentionally removed
	ErrGone = &APIError{Message: "Gone", Code: http.StatusGone}

	// ErrUnsupportedMediaType indicates request body format not supported
	ErrUnsupportedMediaType = &APIError{Message: "Unsupported Media Type", Code: http.StatusUnsupportedMediaType}

	// ErrUnprocessableEntity indicates domain-specific validation failed
	ErrUnprocessableEntity = &APIError{Message: "Unprocessable Entity", Code: http.StatusUnprocessableEntity}

	// ErrTooManyRequests indicates rate limiting has been triggered
	ErrTooManyRequests = &APIError{Message: "Too Many Requests", Code: http.StatusTooManyRequests}
)

// Server error sentinel values (5xx)
var (
	// ErrInternalServerError indicates unexpected server error
	ErrInternalServerError = &APIError{Message: "Internal Server Error", Code: http.StatusInternalServerError}

	// ErrNotImplemented indicates endpoint or method not implemented
	ErrNotImplemented = &APIError{Message: "Not Implemented", Code: http.StatusNotImplemented}

	// ErrBadGateway indicates upstream dependency failure
	ErrBadGateway = &APIError{Message: "Bad Gateway", Code: http.StatusBadGateway}

	// ErrServiceUnavailable indicates API is temporarily unavailable
	ErrServiceUnavailable = &APIError{Message: "Service Unavailable", Code: http.StatusServiceUnavailable}

	// ErrGatewayTimeout indicates upstream service timeout
	ErrGatewayTimeout = &APIError{Message: "Gateway Timeout", Code: http.StatusGatewayTimeout}
)
