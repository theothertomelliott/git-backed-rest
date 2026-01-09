package gitprotocol_test

import (
	"context"
	"fmt"
	"log"

	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/theothertomelliott/git-backed-rest/backends/gitprotocol"
)

// Example demonstrates how to use the gitprotocol backend with HTTP authentication
func Example() {
	// For GitHub, GitLab, Bitbucket: use BasicAuth with token as password
	auth := &http.BasicAuth{
		Username: "git", // Can be any non-empty string for token auth
		Password: "your-personal-access-token",
	}

	backend, err := gitprotocol.NewBackendWithAuth(
		"https://github.com/owner/repo",
		auth,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()

	// Use the backend
	ctx := context.Background()
	_, content, apiErr := backend.GET(ctx, "path/to/file")
	if apiErr != nil {
		log.Fatal(apiErr)
	}

	fmt.Println(string(content))
}

// Example_publicRepo demonstrates accessing a public repository without authentication
func Example_publicRepo() {
	backend, err := gitprotocol.NewBackend("https://github.com/owner/public-repo")
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()

	ctx := context.Background()
	_, content, apiErr := backend.GET(ctx, "README.md")
	if apiErr != nil {
		log.Fatal(apiErr)
	}

	fmt.Println(string(content))
}

// Example_usernamePassword demonstrates username/password authentication
func Example_usernamePassword() {
	auth := &http.BasicAuth{
		Username: "your-username",
		Password: "your-password",
	}

	backend, err := gitprotocol.NewBackendWithAuth(
		"https://git.example.com/repo",
		auth,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()

	// Use the backend...
}

// Example_bearerToken demonstrates bearer token authentication
func Example_bearerToken() {
	auth := &http.TokenAuth{
		Token: "your-bearer-token",
	}

	backend, err := gitprotocol.NewBackendWithAuth(
		"https://git.example.com/repo",
		auth,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()

	// Use the backend...
}
