package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/theothertomelliott/git-backed-rest/backends/gitprotocol"
)

func main() {
	// Get required environment variables
	org := os.Getenv("GITHUB_ORG")
	if org == "" {
		log.Fatalf("GITHUB_ORG environment variable must be set")
	}

	token := os.Getenv("GITHUB_PAT_TOKEN")
	if token == "" {
		log.Fatalf("GITHUB_PAT_TOKEN environment variable must be set")
	}

	// Create test repository
	ctx := context.Background()
	backend, cleanup, err := gitprotocol.NewTestBackend(ctx)
	if err != nil {
		log.Fatalf("Failed to create test repository: %v", err)
	}

	// Output only the endpoint URL
	endpoint := backend.GetEndpoint()
	fmt.Print(endpoint)

	// Keep the cleanup function available but don't call it automatically
	// The repository will persist until manually cleaned up
	_ = cleanup
}
