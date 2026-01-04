package main

import (
	"context"
	"log"
	"net/http"
	"os"

	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
	"github.com/theothertomelliott/git-backed-rest/backends/gitprotocol"
	"github.com/theothertomelliott/git-backed-rest/backends/memory"
	"github.com/theothertomelliott/git-backed-rest/backends/s3"
	"github.com/theothertomelliott/git-backed-rest/server"
)

func main() {
	// Get configuration from environment variables
	port := getEnv("PORT", "8080")
	backendType := getEnv("BACKEND_TYPE", "memory")

	log.Printf("Starting server on port %s with %s backend", port, backendType)

	// Create backend based on type
	backend, cleanup, err := createBackend(backendType)
	if err != nil {
		log.Fatalf("Failed to create backend: %v", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Create server
	srv := server.New(backend)
	http.HandleFunc("/", srv.HandleRequest)

	// Create http.Server for proper shutdown
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: nil, // Use default ServeMux
	}

	log.Printf("Server ready on http://localhost:%s", port)

	// Start server
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func createBackend(backendType string) (gitbackedrest.APIBackend, func(), error) {
	switch backendType {
	case "memory":
		return memory.NewBackend(), nil, nil

	case "git":
		return createGitBackend()

	case "s3":
		return createS3Backend()

	default:
		log.Fatalf("Unknown backend type: %s. Supported: memory, git, s3", backendType)
		return nil, nil, nil // This line won't be reached due to log.Fatalf
	}
}

func createGitBackend() (gitbackedrest.APIBackend, func(), error) {
	// For git backend, we can either create a test repo or use an existing one
	testRepoURL := getEnv("GIT_REPO_URL", "")
	if testRepoURL != "" {
		// Use existing repository
		auth, err := gitprotocol.GetAuthForEndpoint(testRepoURL)
		if err != nil {
			return nil, nil, err
		}

		backend, err := gitprotocol.NewBackendWithAuth(testRepoURL, auth)
		if err != nil {
			return nil, nil, err
		}

		return backend, nil, nil
	}

	// Create test repository (requires TEST_GITHUB_ORG and TEST_GITHUB_PAT_TOKEN)
	ctx := context.Background()
	backend, cleanup, err := gitprotocol.NewTestBackend(ctx)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("Created test repository: %s", backend.GetEndpoint())

	return backend, cleanup, nil
}

func createS3Backend() (gitbackedrest.APIBackend, func(), error) {
	// Required S3 environment variables
	endpoint := getEnv("S3_ENDPOINT", "")
	accessKeyID := getEnv("S3_ACCESS_KEY_ID", "")
	secretAccessKey := getEnv("S3_SECRET_ACCESS_KEY", "")
	bucket := getEnv("S3_BUCKET", "")

	if endpoint == "" || accessKeyID == "" || secretAccessKey == "" || bucket == "" {
		log.Fatalf("S3 backend requires: S3_ENDPOINT, S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY, S3_BUCKET")
		return nil, nil, nil // This line won't be reached due to log.Fatalf
	}

	// Optional prefix for namespace isolation
	prefix := getEnv("S3_PREFIX", "")

	backend, err := s3.NewBackend(s3.Config{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Bucket:          bucket,
		Prefix:          prefix,
	})
	if err != nil {
		return nil, nil, err
	}

	return backend, nil, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
