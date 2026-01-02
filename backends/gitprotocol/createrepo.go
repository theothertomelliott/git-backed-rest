package gitprotocol

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/google/go-github/v79/github"
	"github.com/tjarratt/babble"
)

func NewTestBackend(ctx context.Context) (*Backend, func(), error) {
	remote, cleanup, err := CreateTestGitHubRepo(ctx)
	if err != nil {
		return nil, nil, err
	}

	auth, err := GetAuthForEndpoint(remote)
	if err != nil {
		return nil, nil, err
	}

	backend, err := NewBackendWithAuth(remote, auth)
	if err != nil {
		return nil, nil, err
	}

	return backend, cleanup, nil
}

// GetAuthForEndpoint returns appropriate authentication for the endpoint.
// For HTTP/HTTPS endpoints, it uses BasicAuth with TEST_GITHUB_PAT_TOKEN.
// For SSH endpoints, it uses the SSH agent (default SSH configuration).
func GetAuthForEndpoint(endpoint string) (transport.AuthMethod, error) {
	// Parse the endpoint to determine the scheme
	ep, err := transport.NewEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint: %w", err)
	}

	// For SSH, use SSH agent authentication
	if ep.Scheme == "ssh" {
		auth, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			return nil, fmt.Errorf("creating SSH agent auth: %w", err)
		}
		return auth, nil
	}

	// For HTTP/HTTPS, use token from environment
	if ep.Scheme == "https" || ep.Scheme == "http" {
		githubToken := os.Getenv("TEST_GITHUB_PAT_TOKEN")
		if githubToken == "" {
			return nil, fmt.Errorf("TEST_GITHUB_PAT_TOKEN must be set for HTTP/HTTPS endpoints")
		}

		// GitHub uses BasicAuth with token as password
		return &http.BasicAuth{
			Username: "git", // Can be any non-empty string for token auth
			Password: githubToken,
		}, nil
	}

	// For other protocols, return nil
	return nil, nil
}

func CreateTestGitHubRepo(ctx context.Context) (remote string, cleanup func(), err error) {
	org := os.Getenv("TEST_GITHUB_ORG")
	if org == "" {
		return "", nil, fmt.Errorf("TEST_GITHUB_ORG must be set")
	}
	githubToken := os.Getenv("TEST_GITHUB_PAT_TOKEN")
	if githubToken == "" {
		return "", nil, fmt.Errorf("TEST_GITHUB_PAT_TOKEN must be set")
	}

	babbler := babble.NewBabbler()
	babbler.Count = 3
	babbler.Separator = "-"
	repoName := babbler.Babble()

	client := github.NewClient(nil).WithAuthToken(githubToken)
	repo, _, err := client.Repositories.Create(ctx, org, &github.Repository{
		Name:        github.Ptr(repoName),
		Description: github.Ptr("test repo created by git-backed-rest"),
		Topics:      []string{"test", "git-backed-rest", "integration-test"},

		HasIssues:      github.Ptr(false),
		HasWiki:        github.Ptr(false),
		HasPages:       github.Ptr(false),
		HasProjects:    github.Ptr(false),
		HasDownloads:   github.Ptr(false),
		HasDiscussions: github.Ptr(false),
		IsTemplate:     github.Ptr(false),

		// This forces an initial commit and branch
		LicenseTemplate: github.Ptr("MIT"),
	})
	if err != nil {
		return "", nil, fmt.Errorf("creating GitHub repository: %w", err)
	}

	return repo.GetHTMLURL(), func() {
		_, err := client.Repositories.Delete(ctx, org, repoName)
		if err != nil {
			log.Printf("failed to delete test repository %s/%s: %v", org, repoName, err)
		}
	}, nil
}
