package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v79/github"
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

	// Check for specific repo URL
	repoURL := os.Getenv("REPO_URL")

	// Create GitHub client
	ctx := context.Background()
	client := github.NewClient(nil).WithAuthToken(token)

	if repoURL != "" {
		// Delete specific repository
		deleteSpecificRepo(ctx, client, org, repoURL)
	} else {
		// Delete all test repositories
		deleteAllTestRepos(ctx, client, org)
	}
}

func deleteSpecificRepo(ctx context.Context, client *github.Client, org, repoURL string) {
	// Extract repo name from URL
	parts := strings.Split(repoURL, "/")
	if len(parts) < 1 {
		log.Fatalf("Invalid repository URL: %s", repoURL)
	}
	repoName := parts[len(parts)-1]

	log.Printf("Deleting specific repository: %s/%s", org, repoName)

	// Delete the repository
	_, err := client.Repositories.Delete(ctx, org, repoName)
	if err != nil {
		log.Fatalf("Error deleting repository %s/%s: %v", org, repoName, err)
	}

	log.Printf("Successfully deleted repository: %s/%s", org, repoName)
}

func deleteAllTestRepos(ctx context.Context, client *github.Client, org string) {
	log.Printf("Searching for test repositories in organization: %s", org)

	// Get all repositories in the organization
	opt := &github.RepositoryListByOrgOptions{
		Type:        "all",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			log.Fatalf("Error listing repositories: %v", err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	log.Printf("Found %d repositories in organization", len(allRepos))

	// Find test repositories
	var testRepos []*github.Repository
	testDescription := "test repo created by git-backed-rest"

	for _, repo := range allRepos {
		if repo.Description != nil && *repo.Description == testDescription {
			testRepos = append(testRepos, repo)
		}
	}

	if len(testRepos) == 0 {
		log.Printf("No test repositories found with description: %s", testDescription)
		return
	}

	log.Printf("Found %d test repositories to delete:", len(testRepos))
	for _, repo := range testRepos {
		log.Printf("- %s (%s)", *repo.Name, *repo.HTMLURL)
	}

	// Ask for confirmation unless FORCE_DELETE is set
	forceDelete := os.Getenv("FORCE_DELETE")
	if forceDelete != "true" {
		fmt.Print("Do you want to delete these repositories? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			log.Println("Deletion cancelled")
			return
		}
	}

	// Delete the test repositories
	for _, repo := range testRepos {
		log.Printf("Deleting repository: %s", *repo.Name)
		_, err := client.Repositories.Delete(ctx, org, *repo.Name)
		if err != nil {
			log.Printf("Error deleting repository %s: %v", *repo.Name, err)
			continue
		}
		log.Printf("Successfully deleted repository: %s", *repo.Name)
	}

	log.Printf("Cleanup completed. Deleted %d test repositories.", len(testRepos))
}
