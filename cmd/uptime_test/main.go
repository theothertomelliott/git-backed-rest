package main

import (
	"context"
	"log"
	"net/http"
	"time"

	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
	"github.com/theothertomelliott/git-backed-rest/backends/gitprotocol"
	"github.com/theothertomelliott/git-backed-rest/client"
	"github.com/theothertomelliott/git-backed-rest/server"
)

func main() {
	ctx := context.Background()

	// Create backend with cleanup
	backend, cleanup, err := gitprotocol.NewTestBackend(ctx)
	if err != nil {
		log.Fatalf("Failed to create backend: %v", err)
	}
	// Cleanup the repo when done
	defer cleanup()

	// Start server and get shutdown function
	shutdown := runServer(backend)

	// Give the server a moment to start
	log.Printf("Waiting for server to be ready...")
	time.Sleep(1 * time.Second)

	// Run client operations
	if err := runClientOperations(); err != nil {
		log.Printf("Client operations failed: %v", err)
	}

	// Shutdown server
	log.Println("Shutting down server...")
	if err := shutdown(); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}

	log.Println("Test complete, exiting")
}

func runClientOperations() error {
	// Create context with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := client.New("http://localhost:8080")

	// Test basic connectivity first
	log.Printf("Testing server connectivity...")
	testCtx, testCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer testCancel()

	// Try a simple GET to see if server responds
	_, err := client.GET(testCtx, "/nonexistent")
	if err != nil {
		log.Printf("Connectivity test failed (expected for nonexistent): %v", err)
	} else {
		log.Printf("✓ Server is responding")
	}

	testPath := "/test.json"
	originalData := []byte(`{"message": "hello world", "timestamp": "2025-01-02"}`)
	updatedData := []byte(`{"message": "hello updated world", "timestamp": "2025-01-02"}`)

	log.Printf("Creating resource at %s", testPath)
	if err := client.POST(ctx, testPath, originalData); err != nil {
		return err
	}
	log.Printf("✓ Created resource")

	log.Printf("Getting resource at %s", testPath)
	result, err := client.GET(ctx, testPath)
	if err != nil {
		return err
	}
	log.Printf("✓ Got resource: %s", string(result))

	log.Printf("Updating resource at %s", testPath)
	if err := client.PUT(ctx, testPath, updatedData); err != nil {
		return err
	}
	log.Printf("✓ Updated resource")

	log.Printf("Getting updated resource at %s", testPath)
	result, err = client.GET(ctx, testPath)
	if err != nil {
		return err
	}
	log.Printf("✓ Got updated resource: %s", string(result))

	log.Printf("Deleting resource at %s", testPath)
	if err := client.DELETE(ctx, testPath); err != nil {
		return err
	}
	log.Printf("✓ Deleted resource")

	log.Printf("Verifying deletion at %s", testPath)
	_, err = client.GET(ctx, testPath)
	if err == nil {
		log.Printf("✗ Resource still exists after deletion")
		return err
	}
	log.Printf("✓ Resource successfully deleted")

	return nil
}

func runServer(backend gitbackedrest.APIBackend) func() error {
	server := server.New(backend)
	http.HandleFunc("/", server.HandleRequest)

	port := ":8080"
	log.Printf("Starting server on %s", port)

	// Create http.Server for proper shutdown
	httpServer := &http.Server{
		Addr:    port,
		Handler: nil, // Use default ServeMux
	}

	// Start server in goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Return shutdown function
	return func() error {
		log.Println("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}
