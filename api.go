package gitbackedrest

import "context"

// GetResult represents the result of a GET operation with data and retry count
type GetResult struct {
	Data    []byte
	Retries int
}

// Result represents the result of POST, PUT, DELETE operations with retry count
type Result struct {
	Retries int
}

// APIBackend defines the interface for REST API storage backends.
// Methods return result structs that include retry counts and data where applicable.
type APIBackend interface {
	GET(ctx context.Context, path string) (*GetResult, error)
	POST(ctx context.Context, path string, body []byte) (*Result, error)
	PUT(ctx context.Context, path string, body []byte) (*Result, error)
	DELETE(ctx context.Context, path string) (*Result, error)
}
