package gitprotocol

import (
	"context"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/trace"
	"testing"

	"github.com/go-git/go-git/v6/plumbing/transport"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/google/go-github/v79/github"
	"github.com/joho/godotenv"
	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
	"github.com/tjarratt/babble"
)

const alwaysCleanup = true

func TestGet(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test.
	ctx, task := trace.NewTask(ctx, "SetupTestGet")

	reg := trace.StartRegion(ctx, "createTestGitHubRepo")
	remote, cleanup := createTestGitHubRepo(t)
	defer ifPassed(t, cleanup)
	reg.End()

	reg = trace.StartRegion(ctx, "NewBackend")
	auth := getAuthForEndpoint(t, remote)
	backend, err := NewBackendWithAuth(remote, auth)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	reg.End()

	defer task.End()

	docPath := "doc1"
	docContent := "content1"

	ctx, task = trace.NewTask(ctx, "TestGET")
	defer task.End()

	_, _, getErr := backend.GET(ctx, docPath)
	if getErr == nil {
		t.Fatal("expected error for missing document")
	}
	statusCode := gitbackedrest.GetHTTPStatusCode(getErr, 0)
	if statusCode != http.StatusNotFound {
		t.Fatalf("expected not found status, got %d", statusCode)
	}

	if _, err := backend.POST(ctx, docPath, []byte(docContent)); err != nil {
		t.Fatal(err)
	}

	_, body, getErr := backend.GET(ctx, docPath)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if string(body) != docContent {
		t.Errorf("expected body %s, got %s", docContent, string(body))
	}

	_, postErr := backend.POST(ctx, docPath, []byte(docContent))
	if postErr == nil {
		t.Fatal("expected conflict error on post to existing path")
	}
	statusCode = gitbackedrest.GetHTTPStatusCode(postErr, 0)
	if statusCode != http.StatusConflict {
		t.Fatalf("expected conflict status, got %d", statusCode)
	}
}

func TestGetPreexisting(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test.
	ctx, task := trace.NewTask(ctx, "SetupTestGetPreexisting")

	reg := trace.StartRegion(ctx, "createTestGitHubRepo")
	remote, cleanup := createTestGitHubRepo(t)
	defer ifPassed(t, cleanup)
	reg.End()

	reg = trace.StartRegion(ctx, "NewBackend")
	auth := getAuthForEndpoint(t, remote)
	backend, err := NewBackendWithAuth(remote, auth)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	reg.End()

	defer task.End()

	docPath := "LICENSE"

	ctx, task = trace.NewTask(ctx, "TestGETPreexisting")
	defer task.End()

	_, _, getErr := backend.GET(ctx, docPath)
	if getErr != nil {
		t.Fatal(getErr)
	}
}

func TestPut(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test.
	ctx, task := trace.NewTask(ctx, "TestPut")
	defer task.End()

	remote, cleanup := createTestGitHubRepo(t)
	defer ifPassed(t, cleanup)

	auth := getAuthForEndpoint(t, remote)
	backend, err := NewBackendWithAuth(remote, auth)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	docPath := "doc1"
	docContentPost := "content1"
	docContentPut := "content2"

	t.Log("First PUT - should fail")
	_, putErr := backend.PUT(ctx, docPath, []byte(docContentPut))
	if putErr == nil {
		t.Fatal("expected not found error on put to missing path")
	}
	statusCode := gitbackedrest.GetHTTPStatusCode(putErr, 0)
	if statusCode != http.StatusNotFound {
		t.Fatalf("expected not found status, got %d", statusCode)
	}

	t.Log("POST")
	if _, err := backend.POST(ctx, docPath, []byte(docContentPost)); err != nil {
		t.Fatal(err)
	}

	t.Log("Second PUT - should succeed")
	if _, err := backend.PUT(ctx, docPath, []byte(docContentPut)); err != nil {
		t.Fatal(err)
	}

	t.Log("GET for confirmation")
	_, body, getErr := backend.GET(ctx, docPath)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if string(body) != docContentPut {
		t.Errorf("expected body %s, got %s", docContentPut, string(body))
	}
}

func TestDelete(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test.
	ctx, task := trace.NewTask(ctx, "SetupTestDelete")

	reg := trace.StartRegion(ctx, "createTestGitHubRepo")
	remote, cleanup := createTestGitHubRepo(t)
	defer ifPassed(t, cleanup)
	reg.End()

	reg = trace.StartRegion(ctx, "NewBackend")
	auth := getAuthForEndpoint(t, remote)
	backend, err := NewBackendWithAuth(remote, auth)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	reg.End()

	defer task.End()

	docPath := "doc1"
	docContent := "content1"

	ctx, task = trace.NewTask(ctx, "TestDelete")
	defer task.End()

	_, _, getErr := backend.GET(ctx, docPath)
	if getErr == nil {
		t.Fatal("expected error for missing document")
	}
	statusCode := gitbackedrest.GetHTTPStatusCode(getErr, 0)
	if statusCode != http.StatusNotFound {
		t.Fatalf("expected not found status, got %d", statusCode)
	}

	if _, err := backend.POST(ctx, docPath, []byte(docContent)); err != nil {
		t.Fatal(err)
	}

	_, body, getErr := backend.GET(ctx, docPath)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if string(body) != docContent {
		t.Errorf("expected body %s, got %s", docContent, string(body))
	}

	if _, err := backend.DELETE(ctx, docPath); err != nil {
		t.Fatal(err)
	}

	_, _, getErr = backend.GET(ctx, docPath)
	if getErr == nil {
		t.Fatal("expected error for missing document after delete")
	}
	statusCode = gitbackedrest.GetHTTPStatusCode(getErr, 0)
	if statusCode != http.StatusNotFound {
		t.Fatalf("expected not found status, got %d", statusCode)
	}
}

func init() {
	runtime.SetBlockProfileRate(1)

	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

var ifPassed = func(t *testing.T, f func()) {
	if t.Failed() && !alwaysCleanup {
		return
	}
	f()
}

// getAuthForEndpoint returns appropriate authentication for the endpoint.
// For HTTP/HTTPS endpoints, it uses BasicAuth with TEST_GITHUB_PAT_TOKEN.
// For SSH endpoints, it uses the SSH agent (default SSH configuration).
func getAuthForEndpoint(t *testing.T, endpoint string) transport.AuthMethod {
	// Parse the endpoint to determine the scheme
	ep, err := transport.NewEndpoint(endpoint)
	if err != nil {
		t.Fatalf("parsing endpoint: %v", err)
	}

	// For SSH, use SSH agent authentication
	if ep.Scheme == "ssh" {
		auth, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			t.Fatalf("creating SSH agent auth: %v", err)
		}
		return auth
	}

	// For HTTP/HTTPS, use token from environment
	if ep.Scheme == "https" || ep.Scheme == "http" {
		githubToken := os.Getenv("TEST_GITHUB_PAT_TOKEN")
		if githubToken == "" {
			t.Fatal("TEST_GITHUB_PAT_TOKEN must be set for HTTP/HTTPS endpoints")
		}

		// GitHub uses BasicAuth with token as password
		return &githttp.BasicAuth{
			Username: "git", // Can be any non-empty string for token auth
			Password: githubToken,
		}
	}

	// For other protocols, return nil
	return nil
}

func createTestGitHubRepo(t *testing.T) (remote string, cleanup func()) {
	org := os.Getenv("TEST_GITHUB_ORG")
	if org == "" {
		t.Fatal("TEST_GITHUB_ORG must be set")
	}
	githubToken := os.Getenv("TEST_GITHUB_PAT_TOKEN")
	if githubToken == "" {
		t.Fatal("TEST_GITHUB_PAT_TOKEN must be set")
	}

	babbler := babble.NewBabbler()
	babbler.Count = 3
	babbler.Separator = "-"
	repoName := babbler.Babble()

	client := github.NewClient(nil).WithAuthToken(githubToken)
	repo, _, err := client.Repositories.Create(t.Context(), org, &github.Repository{
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
		t.Fatal(err)
	}

	return repo.GetHTMLURL(), func() {
		_, err := client.Repositories.Delete(context.Background(), org, repoName)
		if err != nil {
			t.Fatal(err)
		}
	}
}
