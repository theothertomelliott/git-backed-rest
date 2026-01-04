# Create Repo Tool

This tool creates a test repository for the git-backed-rest system and outputs only the endpoint URL for use with the server.

## Usage

```bash
# Create a test repository and assign to environment variable
export GIT_REPO_URL=$(GITHUB_ORG=your-org GITHUB_PAT_TOKEN=ghp_your_token go run ./cmd/create_repo)

# Or use directly
GITHUB_ORG=your-org GITHUB_PAT_TOKEN=ghp_your_token go run ./cmd/create_repo
```

## Environment Variables

### Required:
- `GITHUB_ORG` - GitHub organization name where the repository will be created
- `GITHUB_PAT_TOKEN` - GitHub personal access token with repo creation permissions

## Output

The tool outputs only the repository URL (no additional text), making it perfect for command substitution:

```
https://github.com/your-org/Jaime-Lucuma-anthorine
```

## Workflow

1. **Create a test repository and assign to variable:**
   ```bash
   export GIT_REPO_URL=$(GITHUB_ORG=myorg GITHUB_PAT_TOKEN=ghp_xxx go run ./cmd/create_repo)
   ```

2. **Use the variable with the server:**
   ```bash
   BACKEND_TYPE=git docker-compose up server
   ```

3. **Clean up when done:**
   ```bash
   GITHUB_ORG=myorg GITHUB_PAT_TOKEN=ghp_xxx REPO_URL=$GIT_REPO_URL go run ./cmd/gitcleanup
   ```

## Notes

- The repository persists until manually cleaned up
- Repository names are randomly generated using the babble library
- Each repository has a unique name to avoid conflicts
- Output is clean (just the URL) for easy command substitution
