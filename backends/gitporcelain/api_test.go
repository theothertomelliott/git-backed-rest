package gitporcelain

import (
	"log"
	"os"
	"runtime"
	"runtime/trace"
	"testing"

	"github.com/google/go-github/v79/github"
	"github.com/joho/godotenv"
	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
	"github.com/tjarratt/babble"
)

func TestGet(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test.
	ctx, task := trace.NewTask(ctx, "SetupTestGet")

	reg := trace.StartRegion(ctx, "createTestGitHubRepo")
	remote, cleanup := createTestGitHubRepo(t)
	defer ifPassed(t, cleanup)
	reg.End()

	reg = trace.StartRegion(ctx, "createTestDir")
	testDir := createTestDir(t)
	defer ifPassed(t, func() {
		os.RemoveAll(testDir)
	})
	reg.End()

	reg = trace.StartRegion(ctx, "NewBackend")
	backend, err := NewBackend(remote, testDir)
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
	if getErr != gitbackedrest.ErrNotFound {
		t.Fatal(getErr)
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
	if _, err := backend.POST(ctx, docPath, []byte(docContent)); err == nil || err != gitbackedrest.ErrConflict {
		t.Errorf("expected conflict error on post to existing path, got %v", err)
	}
}

func TestPut(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test.
	ctx, task := trace.NewTask(ctx, "TestPut")
	defer task.End()

	remote, cleanup := createTestGitHubRepo(t)
	defer ifPassed(t, cleanup)

	testDir := createTestDir(t)
	defer ifPassed(t, func() {
		os.RemoveAll(testDir)
	})

	backend, err := NewBackend(remote, testDir)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	docPath := "doc1"
	docContentPost := "content1"
	docContentPut := "content2"

	if _, err := backend.PUT(ctx, docPath, []byte(docContentPut)); err == nil || err != gitbackedrest.ErrNotFound {
		t.Errorf("expected not found error on put to missing path, got %v", err)
	}

	if _, err := backend.POST(ctx, docPath, []byte(docContentPost)); err != nil {
		t.Fatal(err)
	}

	if _, err := backend.PUT(ctx, docPath, []byte(docContentPut)); err != nil {
		t.Fatal(err)
	}

	_, body, getErr := backend.GET(ctx, docPath)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if string(body) != docContentPut {
		t.Errorf("expected body %s, got %s", docContentPut, string(body))
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
	if t.Failed() {
		return
	}
	f()
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

	return repo.GetSSHURL(), func() {
		_, err := client.Repositories.Delete(t.Context(), org, repoName)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func createTestDir(t *testing.T) (dirPath string) {
	_ = os.MkdirAll("testdata", os.ModePerm)

	tmpDir, err := os.MkdirTemp("testdata", "tmp-*")
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}
