# Git Cleanup Tool

This tool finds and deletes test repositories created by the git-backed-rest system in a GitHub organization. It can delete all test repositories or a specific one.

## Usage

```bash
# Delete all test repositories (interactive mode)
GITHUB_ORG=your-org GITHUB_PAT_TOKEN=ghp_your_token go run ./cmd/gitcleanup

# Delete all test repositories (force mode)
GITHUB_ORG=your-org GITHUB_PAT_TOKEN=ghp_your_token FORCE_DELETE=true go run ./cmd/gitcleanup

# Delete a specific repository
GITHUB_ORG=your-org GITHUB_PAT_TOKEN=ghp_your_token REPO_URL=https://github.com/your-org/specific-repo go run ./cmd/gitcleanup
```

## Environment Variables

### Required:
- `GITHUB_ORG` - GitHub organization name to search
- `GITHUB_PAT_TOKEN` - GitHub personal access token with repo deletion permissions

### Optional:
- `FORCE_DELETE` - Set to "true" to skip confirmation prompt (only for bulk deletion)
- `REPO_URL` - Specific repository URL to delete (bypasses bulk deletion)

## What it does

### Mode 1: Delete All Test Repositories (default)
1. Lists all repositories in the specified organization
2. Finds repositories with description: "test repo created by git-backed-rest"
3. Shows the found repositories
4. Asks for confirmation (unless FORCE_DELETE=true)
5. Deletes the test repositories
6. Reports success/failure for each deletion

### Mode 2: Delete Specific Repository
1. Extracts repository name from REPO_URL
2. Deletes the specified repository directly
3. Reports success/failure

## Safety Features

- Requires explicit confirmation before bulk deletion (unless FORCE_DELETE is set)
- Only deletes repositories with the exact test description (bulk mode)
- Reports any errors during deletion
- Continues with remaining repos if one deletion fails (bulk mode)

## Example Output

### Bulk Deletion:
```
2026/01/04 11:14:27 Searching for test repositories in organization: myorg
2026/01/04 11:14:28 Found 150 repositories in organization
2026/01/04 11:14:28 Found 3 test repositories to delete:
- test-repo-abc123 (https://github.com/myorg/test-repo-abc123)
- test-repo-def456 (https://github.com/myorg/test-repo-def456)
- test-repo-ghi789 (https://github.com/myorg/test-repo-ghi789)
Do you want to delete these repositories? (y/N): y
2026/01/04 11:14:30 Deleting repository: test-repo-abc123
2026/01/04 11:14:31 Successfully deleted repository: test-repo-abc123
2026/01/04 11:14:32 Deleting repository: test-repo-def456
2026/01/04 11:14:33 Successfully deleted repository: test-repo-def456
2026/01/04 11:14:34 Deleting repository: test-repo-ghi789
2026/01/04 11:14:35 Successfully deleted repository: test-repo-ghi789
2026/01/04 11:14:35 Cleanup completed. Deleted 3 test repositories.
```

### Specific Repository Deletion:
```
2026/01/04 11:15:00 Deleting specific repository: myorg/test-repo-abc123
2026/01/04 11:15:01 Successfully deleted repository: myorg/test-repo-abc123
```
