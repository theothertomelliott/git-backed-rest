package gitbackedrest

import (
	"errors"
)

// HTTPError wraps an error with an HTTP status code.
type HTTPError struct {
	Err  error
	Code int
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *HTTPError) Unwrap() error {
	return e.Err
}

// NewHTTPError creates a new HTTPError with the given status code and underlying error.
func NewHTTPError(code int, err error) error {
	return &HTTPError{
		Err:  err,
		Code: code,
	}
}

// UserError wraps an error with a user-friendly message suitable for HTTP responses or UI display.
type UserError struct {
	Err         error
	UserMessage string
}

// Error implements the error interface.
func (e *UserError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *UserError) Unwrap() error {
	return e.Err
}

// NewUserError creates a new UserError with the given user-friendly message and underlying error.
func NewUserError(userMessage string, err error) error {
	return &UserError{
		Err:         err,
		UserMessage: userMessage,
	}
}

// GetHTTPStatusCode extracts the HTTP status code from an error if it's an HTTPError.
// Returns the provided default code if no HTTPError is found.
func GetHTTPStatusCode(err error, defaultCode int) int {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Code
	}
	return defaultCode
}

// GetUserMessage extracts the user-friendly message from an error if it's a UserError.
// Returns the error's Error() message if no UserError is found.
func GetUserMessage(err error) string {
	var userErr *UserError
	if errors.As(err, &userErr) {
		return userErr.UserMessage
	}
	return err.Error()
}

// HasHTTPStatusCode checks if an error contains any of the provided HTTP status codes.
// Returns true if the error has an HTTPError with a status code that matches any in the provided set.
func HasHTTPStatusCode(err error, codes ...int) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		for _, code := range codes {
			if httpErr.Code == code {
				return true
			}
		}
	}
	return false
}
