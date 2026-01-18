package gitbackedrest

import (
	"context"
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
	GET(ctx context.Context, path string) (context.Context, []byte, error)
	POST(ctx context.Context, path string, body []byte) (context.Context, error)
	PUT(ctx context.Context, path string, body []byte) (context.Context, error)
	DELETE(ctx context.Context, path string) (context.Context, error)
}
