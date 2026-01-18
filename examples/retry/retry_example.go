package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)

// RetryExampleBackend demonstrates how to implement retry tracking with Approach 1
type RetryExampleBackend struct {
	attempts map[string]int // Track attempts per path
}

func NewRetryExampleBackend() *RetryExampleBackend {
	return &RetryExampleBackend{
		attempts: make(map[string]int),
	}
}

func (b *RetryExampleBackend) GET(ctx context.Context, path string) (context.Context, []byte, error) {
	// Simulate a GET that might need retries
	retries := 0
	maxRetries := 3

	for retries < maxRetries {
		// Simulate operation that fails first 2 times
		if retries >= 2 {
			// Success! Set retry count in context and return result
			ctx = gitbackedrest.SetRetryCount(ctx, retries)
			return ctx, []byte(fmt.Sprintf("data for %s (after %d retries)", path, retries)), nil
		}
		retries++
		time.Sleep(10 * time.Millisecond) // Simulate delay
	}

	// All retries failed
	err := gitbackedrest.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("GET failed after %d retries", retries))
	userErr := gitbackedrest.NewUserError(fmt.Sprintf("GET failed after %d retries", retries), err)
	return ctx, nil, userErr
}

func (b *RetryExampleBackend) POST(ctx context.Context, path string, body []byte) (context.Context, error) {
	// Simulate a POST that might need retries
	retries := 0
	maxRetries := 2

	for retries < maxRetries {
		// Simulate operation that fails first time
		if retries >= 1 {
			// Success! Set retry count in context
			ctx = gitbackedrest.SetRetryCount(ctx, retries)
			return ctx, nil
		}
		retries++
		time.Sleep(10 * time.Millisecond) // Simulate delay
	}

	// All retries failed
	err := gitbackedrest.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("POST failed after %d retries", retries))
	userErr := gitbackedrest.NewUserError(fmt.Sprintf("POST failed after %d retries", retries), err)
	return ctx, userErr
}

func (b *RetryExampleBackend) PUT(ctx context.Context, path string, body []byte) (context.Context, error) {
	// Simulate a PUT that succeeds immediately
	ctx = gitbackedrest.SetRetryCount(ctx, 0)
	return ctx, nil
}

func (b *RetryExampleBackend) DELETE(ctx context.Context, path string) (context.Context, error) {
	// Simulate a DELETE that fails after retries
	retries := 0
	maxRetries := 3

	for retries < maxRetries {
		// This operation always fails
		retries++
		time.Sleep(10 * time.Millisecond)
	}

	err := gitbackedrest.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("DELETE failed after %d retries", retries))
	userErr := gitbackedrest.NewUserError(fmt.Sprintf("DELETE failed after %d retries", retries), err)
	return ctx, userErr
}

func main() {
	fmt.Println("Retry Tracking Example (Approach 1)")
	fmt.Println("=====================================")

	backend := NewRetryExampleBackend()
	ctx := context.Background()

	// Test GET with retries
	fmt.Println("\n1. Testing GET with retries:")
	newCtx, body, err := backend.GET(ctx, "/test")
	if err != nil {
		userMessage := gitbackedrest.GetUserMessage(err)
		statusCode := gitbackedrest.GetHTTPStatusCode(err, 0)
		fmt.Printf("   GET failed: %s (status: %d)\n", userMessage, statusCode)
	} else {
		retries := gitbackedrest.GetRetryCount(newCtx)
		fmt.Printf("   GET succeeded: %s (retries: %d)\n", string(body), retries)
	}

	// Test POST with retries
	fmt.Println("\n2. Testing POST with retries:")
	newCtx, err = backend.POST(ctx, "/test", []byte("test data"))
	if err != nil {
		userMessage := gitbackedrest.GetUserMessage(err)
		statusCode := gitbackedrest.GetHTTPStatusCode(err, 0)
		fmt.Printf("   POST failed: %s (status: %d)\n", userMessage, statusCode)
	} else {
		retries := gitbackedrest.GetRetryCount(newCtx)
		fmt.Printf("   POST succeeded (retries: %d)\n", retries)
	}

	// Test PUT without retries
	fmt.Println("\n3. Testing PUT without retries:")
	newCtx, err = backend.PUT(ctx, "/test", []byte("updated data"))
	if err != nil {
		userMessage := gitbackedrest.GetUserMessage(err)
		statusCode := gitbackedrest.GetHTTPStatusCode(err, 0)
		fmt.Printf("   PUT failed: %s (status: %d)\n", userMessage, statusCode)
	} else {
		retries := gitbackedrest.GetRetryCount(newCtx)
		fmt.Printf("   PUT succeeded (retries: %d)\n", retries)
	}

	// Test DELETE that fails
	fmt.Println("\n4. Testing DELETE that fails:")
	newCtx, err = backend.DELETE(ctx, "/test")
	if err != nil {
		userMessage := gitbackedrest.GetUserMessage(err)
		statusCode := gitbackedrest.GetHTTPStatusCode(err, 0)
		fmt.Printf("   DELETE failed: %s (status: %d)\n", userMessage, statusCode)
	} else {
		retries := gitbackedrest.GetRetryCount(newCtx)
		fmt.Printf("   DELETE succeeded (retries: %d)\n", retries)
	}

	fmt.Println("\n=====================================")
	fmt.Println("Key Points:")
	fmt.Println("- Success after retries: Context contains retry count")
	fmt.Println("- Failure after retries: Error contains HTTP status and user message")
	fmt.Println("- Immediate success: Context contains retry count (0)")
	fmt.Println("- Server can distinguish retry vs non-retry operations")
}
