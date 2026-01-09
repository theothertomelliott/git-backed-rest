package gitbackedrest

import (
	"context"
	"net/http"
)

// Context key for storing retry information
type contextKey string

const RetryCountKey contextKey = "retry_count"

// SetRetryCount sets the retry count in the context
func SetRetryCount(ctx context.Context, retries int) context.Context {
	return context.WithValue(ctx, RetryCountKey, retries)
}

// GetRetryCount gets the retry count from the context
func GetRetryCount(ctx context.Context) int {
	if retries, ok := ctx.Value(RetryCountKey).(int); ok {
		return retries
	}
	return 0
}

// APIBackend defines the interface for REST API storage backends.
// Methods return updated context first for retry tracking, followed by results, with errors last.
type APIBackend interface {
	GET(ctx context.Context, path string) (context.Context, []byte, *APIError)
	POST(ctx context.Context, path string, body []byte) (context.Context, *APIError)
	PUT(ctx context.Context, path string, body []byte) (context.Context, *APIError)
	DELETE(ctx context.Context, path string) (context.Context, *APIError)
}

var _ error = new(APIError)

type APIError struct {
	Message string
	Code    int
	Retries int // Number of retry attempts made
}

// Error implements error.
func (a *APIError) Error() string {
	return a.Message
}

// Client error sentinel values (4xx)
var (
	// ErrBadRequest indicates invalid syntax or validation errors in the request
	ErrBadRequest = &APIError{Message: "Bad Request", Code: http.StatusBadRequest, Retries: 0}

	// ErrUnauthorized indicates missing or invalid authentication
	ErrUnauthorized = &APIError{Message: "Unauthorized", Code: http.StatusUnauthorized, Retries: 0}

	// ErrForbidden indicates authenticated but not allowed to access this resource
	ErrForbidden = &APIError{Message: "Forbidden", Code: http.StatusForbidden, Retries: 0}

	// ErrNotFound indicates resource or endpoint does not exist
	ErrNotFound = &APIError{Message: "Not Found", Code: http.StatusNotFound, Retries: 0}

	// ErrMethodNotAllowed indicates HTTP method not supported for this endpoint
	ErrMethodNotAllowed = &APIError{Message: "Method Not Allowed", Code: http.StatusMethodNotAllowed, Retries: 0}

	// ErrConflict indicates request conflicts with current state
	ErrConflict = &APIError{Message: "Conflict", Code: http.StatusConflict, Retries: 0}

	// ErrGone indicates resource has been intentionally removed
	ErrGone = &APIError{Message: "Gone", Code: http.StatusGone, Retries: 0}

	// ErrUnsupportedMediaType indicates request body format not supported
	ErrUnsupportedMediaType = &APIError{Message: "Unsupported Media Type", Code: http.StatusUnsupportedMediaType, Retries: 0}

	// ErrUnprocessableEntity indicates domain-specific validation failed
	ErrUnprocessableEntity = &APIError{Message: "Unprocessable Entity", Code: http.StatusUnprocessableEntity, Retries: 0}

	// ErrTooManyRequests indicates rate limiting has been triggered
	ErrTooManyRequests = &APIError{Message: "Too Many Requests", Code: http.StatusTooManyRequests, Retries: 0}
)

// Server error sentinel values (5xx)
var (
	// ErrInternalServerError indicates unexpected server error
	ErrInternalServerError = &APIError{Message: "Internal Server Error", Code: http.StatusInternalServerError, Retries: 0}

	// ErrNotImplemented indicates endpoint or method not implemented
	ErrNotImplemented = &APIError{Message: "Not Implemented", Code: http.StatusNotImplemented, Retries: 0}

	// ErrBadGateway indicates upstream dependency failure
	ErrBadGateway = &APIError{Message: "Bad Gateway", Code: http.StatusBadGateway, Retries: 0}

	// ErrServiceUnavailable indicates API is temporarily unavailable
	ErrServiceUnavailable = &APIError{Message: "Service Unavailable", Code: http.StatusServiceUnavailable, Retries: 0}

	// ErrGatewayTimeout indicates upstream service timeout
	ErrGatewayTimeout = &APIError{Message: "Gateway Timeout", Code: http.StatusGatewayTimeout, Retries: 0}
)
