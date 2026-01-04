package main

import (
	"context"
	"log"
	"time"

	"github.com/theothertomelliott/git-backed-rest/client"
)

func main() {
	// Run client operations
	if err := runClientOperations(); err != nil {
		log.Printf("Client operations failed: %v", err)
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
