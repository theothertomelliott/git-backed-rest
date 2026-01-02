package client

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/theothertomelliott/git-backed-rest/backends/memory"
	"github.com/theothertomelliott/git-backed-rest/server"
)

func TestClientOperations(t *testing.T) {
	// Create a test server with memory backend
	backend := memory.NewBackend()
	srv := server.New(backend)

	// Start test server
	httpServer := httptest.NewServer(http.HandlerFunc(srv.HandleRequest))
	defer httpServer.Close()

	// Create client pointing to test server
	client := New(httpServer.URL)
	ctx := context.Background()

	// Test POST (create)
	testData := []byte(`{"test": "data"}`)
	err := client.POST(ctx, "/test/resource", testData)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	// Test GET (retrieve)
	result, err := client.GET(ctx, "/test/resource")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	if !bytes.Equal(result, testData) {
		t.Errorf("GET returned unexpected data: got %s, want %s", string(result), string(testData))
	}

	// Test PUT (update)
	updatedData := []byte(`{"test": "updated"}`)
	err = client.PUT(ctx, "/test/resource", updatedData)
	if err != nil {
		t.Fatalf("PUT failed: %v", err)
	}

	// Verify update with GET
	result, err = client.GET(ctx, "/test/resource")
	if err != nil {
		t.Fatalf("GET after PUT failed: %v", err)
	}

	if !bytes.Equal(result, updatedData) {
		t.Errorf("GET after PUT returned unexpected data: got %s, want %s", string(result), string(updatedData))
	}

	// Test DELETE
	err = client.DELETE(ctx, "/test/resource")
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}

	// Verify deletion with GET (should fail)
	_, err = client.GET(ctx, "/test/resource")
	if err == nil {
		t.Error("GET after DELETE should have failed but didn't")
	}
}

func TestClientErrorHandling(t *testing.T) {
	backend := memory.NewBackend()
	srv := server.New(backend)

	server := httptest.NewServer(http.HandlerFunc(srv.HandleRequest))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	// Test GET non-existent resource
	_, err := client.GET(ctx, "/nonexistent")
	if err == nil {
		t.Error("GET non-existent resource should have failed")
	}

	// Test PUT non-existent resource
	err = client.PUT(ctx, "/nonexistent", []byte("data"))
	if err == nil {
		t.Error("PUT non-existent resource should have failed")
	}

	// Test POST to existing resource
	testData := []byte("test data")
	err = client.POST(ctx, "/conflict", testData)
	if err != nil {
		t.Fatalf("Initial POST failed: %v", err)
	}

	// Try POST again to same resource (should conflict)
	err = client.POST(ctx, "/conflict", testData)
	if err == nil {
		t.Error("POST to existing resource should have failed")
	}

	// Test DELETE non-existent resource
	err = client.DELETE(ctx, "/nonexistent")
	if err == nil {
		t.Error("DELETE non-existent resource should have failed")
	}
}

func TestClientWithContext(t *testing.T) {
	backend := memory.NewBackend()
	srv := server.New(backend)

	server := httptest.NewServer(http.HandlerFunc(srv.HandleRequest))
	defer server.Close()

	client := New(server.URL)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(10 * time.Millisecond)

	// Try operation with expired context
	_, err := client.GET(ctx, "/test")
	if err == nil {
		t.Error("Operation with expired context should have failed")
	}

	// Check if error mentions context
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Error should mention context: %v", err)
	}
}
