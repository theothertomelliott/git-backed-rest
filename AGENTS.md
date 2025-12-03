# AGENTS.md

Guide for AI agents working in the git-backed-rest codebase.

## Project Overview

**Purpose**: Exploring whether Git can be used as an effective backing store for a REST API.

**Language**: Go 1.25.1

**Module**: `github.com/theothertomelliott/git-backed-rest`

This project implements a REST API abstraction (`APIBackend`) with multiple backend implementations:
- `memory`: In-memory storage (for testing/benchmarking)
- `gitporcelain`: Git-backed storage using git CLI commands
- `s3`: S3-compatible object storage (works with R2, S3, MinIO, etc.)

## Project Structure

```
├── api.go                      # Core APIBackend interface and error definitions
├── backends/
│   ├── memory/                 # In-memory backend implementation
│   │   └── api.go
│   ├── gitporcelain/          # Git-backed implementation
│   │   ├── api.go
│   │   ├── api_test.go
│   │   └── testdata/
│   └── s3/                    # S3-compatible storage implementation
│       ├── api.go
│       ├── api_test.go
│       └── README.md
├── cmd/
│   └── server/                # HTTP server implementation
│       ├── main.go            # Entry point (uses memory backend)
│       ├── server.go          # HTTP handlers
│       ├── server_test.go     # Server tests
│       └── server_benchmark_test.go
├── go.mod
└── .github/
    └── codesherlock.yaml      # Code review instructions
```

## Essential Commands

### Building
```bash
go build ./...                  # Build all packages
go build ./cmd/server          # Build server binary
```

### Testing
```bash
go test ./...                  # Run all tests
go test ./cmd/server           # Test server package
go test ./backends/memory      # Test memory backend
go test ./backends/gitporcelain # Test git backend (requires .env)
go test ./backends/s3          # Test S3 backend (requires .env)
```

### Benchmarking
```bash
go test -bench=. ./cmd/server  # Run benchmarks
```

### Running
```bash
go run ./cmd/server            # Run server on :8080
```

## Architecture

### Core Interface: APIBackend

Located in `api.go`, defines the contract all backends must implement:

```go
type APIBackend interface {
    GET(ctx context.Context, path string) ([]byte, *APIError)
    POST(ctx context.Context, path string, body []byte) *APIError
    PUT(ctx context.Context, path string, body []byte) *APIError
    DELETE(ctx context.Context, path string) *APIError
}
```

**REST Semantics**:
- **POST**: Create new resource (returns `ErrConflict` if exists)
- **PUT**: Update existing resource (returns `ErrNotFound` if missing)
- **DELETE**: Remove resource (returns `ErrNotFound` if missing)
- **GET**: Retrieve resource (returns `ErrNotFound` if missing)

### Error Handling

Custom error type: `APIError` (implements `error` interface)
- Has `Message` (string) and `Code` (int, HTTP status code)
- Sentinel error values defined in `api.go`:
  - Client errors (4xx): `ErrBadRequest`, `ErrNotFound`, `ErrConflict`, etc.
  - Server errors (5xx): `ErrInternalServerError`, `ErrNotImplemented`, etc.

**Pattern**: Backend methods return `*APIError` (or `([]byte, *APIError)` for GET). Nil means success.

### Backend Implementations

#### Memory Backend (`backends/memory/api.go`)
- Simple map-based storage: `map[string][]byte`
- No persistence
- Used for testing and benchmarking
- Follows REST semantics strictly

#### Git Porcelain Backend (`backends/gitporcelain/api.go`)
- Uses git CLI commands via `os/exec`
- Each operation:
  1. Pulls from remote
  2. Performs file operation
  3. Commits and pushes
- Files stored as `{repoPath}/{path}`
- Commit messages: `"write {path}"`, `"delete {path}"`
- Uses `runtime/trace` for performance profiling

#### S3 Backend (`backends/s3/api.go`)
- Uses AWS SDK v2 for S3-compatible storage
- Works with any S3-compatible service (R2, S3, MinIO, etc.)
- Supports prefix for namespace isolation within a bucket
- Each operation checks existence (via HeadObject) before modifying
- Key structure: `{prefix}/{path}` (prefix is optional)
- Uses `runtime/trace` for performance profiling (similar to gitporcelain)
- See `backends/s3/README.md` for detailed configuration
- See `backends/s3/TRACING.md` for performance comparison with git backend

### HTTP Server (`cmd/server/`)

**Structure**:
- `Server` struct holds an `APIBackend`
- Single handler: `handleRequest` dispatches by HTTP method
- Method-specific handlers: `handleGET`, `handlePOST`, `handlePUT`, `handleDELETE`

**HTTP Response Codes**:
- GET success: 200 OK
- POST success: 201 Created
- PUT success: 204 No Content
- DELETE success: 204 No Content
- Errors: Use `APIError.Code` field

**Content Type**: Always sets `Content-Type: application/json` on success responses

## Code Conventions

### Error Messages
**Critical**: From `.github/codesherlock.yaml`:
- Go error messages should NOT include redundant language like "failed" or "error"
- Just describe what was being done, using `%w` for wrapping
- Example: `fmt.Errorf("opening file: %w", err)` NOT `fmt.Errorf("failed to open file: %w", err)`

### Logging
**Critical**: No `fmt.Print*` for logging. Use `log` package from standard library.

### Imports
Standard pattern used throughout:
```go
import (
    // Standard library first
    "context"
    "fmt"
    
    // External dependencies
    "github.com/google/go-github/v79/github"
    
    // Local imports (with alias when needed)
    gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)
```

### Interface Verification
Pattern used to ensure types implement interfaces:
```go
var _ gitbackedrest.APIBackend = (*Backend)(nil)
```

### Context
All backend methods accept `context.Context` as first parameter. Used for:
- Request cancellation
- Tracing (in gitporcelain backend)

## Testing Patterns

### Standard Tests
- Use `testing.T`
- Pattern: create backend → call methods → verify responses
- Check both error cases and success cases
- Use `httptest.NewRecorder()` for HTTP tests

### Git Backend Tests
Located in `backends/gitporcelain/api_test.go`:

**Requirements**:
- Needs `.env` file in repo root with:
  - `TEST_GITHUB_ORG`: GitHub organization for test repos
  - `TEST_GITHUB_PAT_TOKEN`: GitHub PAT with repo permissions

**Test Setup**:
- Creates temporary GitHub repository via API
- Clones to local `testdata/tmp-*` directory
- Runs tests
- Cleans up repo and directory if test passes (uses `ifPassed` helper)

**Tracing**:
- Uses `runtime/trace` API for performance profiling
- `trace.NewTask` creates logical task for test
- `trace.StartRegion` marks regions within test
- See `testdata/trace.out` for output

### S3 Backend Tests
Located in `backends/s3/api_test.go`:

**Requirements**:
- Needs `.env` file in repo root with:
  - `TEST_S3_ENDPOINT`: S3-compatible endpoint (e.g., `https://<account-id>.r2.cloudflarestorage.com`)
  - `TEST_S3_ACCESS_KEY_ID`: Access key ID
  - `TEST_S3_SECRET_ACCESS_KEY`: Secret access key
  - `TEST_S3_BUCKET`: Bucket name

**Test Setup**:
- Generates unique random prefix for each test run using `babble`
- All objects stored under `test/{random-prefix}/*`
- No cleanup needed between runs (isolated by prefix)
- Tests include prefix isolation verification

**Cloudflare R2 Setup**:
1. Create R2 bucket in Cloudflare dashboard
2. Generate API token with read/write permissions
3. Get account ID and construct endpoint URL
4. Set environment variables in `.env`

### Benchmarks
Located in `cmd/server/server_benchmark_test.go`:
- Pattern: setup → create initial resource with POST → loop: PUT + GET
- Measures realistic read/write patterns

## Dependencies

Key external dependencies (from `go.mod`):
- `github.com/google/go-github/v79` - GitHub API client (for git backend tests)
- `github.com/joho/godotenv` - Load .env files (for tests)
- `github.com/tjarratt/babble` - Generate random names (for test repos/prefixes)
- `github.com/aws/aws-sdk-go-v2/*` - AWS SDK v2 (for S3 backend)

## Ignored Files

From `.gitignore`:
- Test artifacts: `*.test`, `*.out`, coverage files
- Environment: `.env`
- Binaries: `*.exe`, `*.dll`, `*.so`, `*.dylib`
- Go workspace: `go.work`, `go.work.sum`

## Current State

**Recent commits** (from git log):
- `30a807d`: Add gitporcelain backend
- `48c10ab`: Create simple harness for http server
- `adcfaf5`: Initial commit

**Modified files** (from git status):
- `backends/gitporcelain/api.go` - Currently has uncommitted changes

**Untracked**:
- `.github/` directory (contains codesherlock.yaml)

## Performance Considerations

### Git Backend Performance
- Each operation does: pull → modify → commit → push
- Potentially slow for high-throughput scenarios
- Tracing instrumentation added to measure bottlenecks
- See `backends/gitporcelain/trace_test.sh` for trace generation

### S3 Backend Performance
- Each POST/PUT/DELETE does two round-trips: HeadObject + main operation
- GET is single round-trip
- Performance depends on endpoint latency and network
- No local caching implemented

### Benchmarking
- Memory backend provides baseline performance
- Use `go test -bench=.` to compare backends
- Benchmarks in `cmd/server/server_benchmark_test.go`

## Future Work / TODOs

Noted in `api.go`:
- PATCH method signature needs better design
- OPTIONS method needs design (what options to expose?)

## Gotchas

1. **Git Backend Test Requirements**: Tests in `backends/gitporcelain` require GitHub credentials in `.env`. They will fail without proper setup.

2. **S3 Backend Test Requirements**: Tests in `backends/s3` require S3/R2 credentials in `.env`. They will fail without proper setup.

3. **REST Semantics**: POST vs PUT semantics are strict:
   - POST on existing resource = Conflict
   - PUT on missing resource = Not Found
   
4. **Directory Handling**: Git backend returns `ErrNotFound` when path points to directory (only files are resources).

5. **Server Backend**: `cmd/server/main.go` currently hardcoded to use memory backend. To use other backends, modify main.go.

6. **Error Wrapping**: Always use `%w` verb when wrapping errors, never `%v`.

7. **Commit Messages**: Git backend uses simple messages like "write {path}" and "delete {path}". No user-provided commit messages.

8. **S3 Prefix Isolation**: Different backends with different prefixes in the same bucket are completely isolated. Useful for testing but be aware when debugging.

9. **Tracing**: Git backend uses runtime tracing extensively. To analyze: run with trace output, use `go tool trace` to view.

## Working with this Codebase

### Adding a New Backend
1. Create package in `backends/{name}/`
2. Implement `gitbackedrest.APIBackend` interface
3. Add interface verification: `var _ gitbackedrest.APIBackend = (*Backend)(nil)`
4. Follow REST semantics (POST=create, PUT=update)
5. Return appropriate sentinel errors from `api.go`
6. Add tests following patterns in `backends/memory` or `backends/gitporcelain`

### Modifying the HTTP Server
- Server is backend-agnostic (uses `APIBackend` interface)
- Test changes against memory backend first (faster)
- Ensure proper HTTP status codes match `APIError.Code`
- Keep `Content-Type: application/json` on success responses

### Running Integration Tests
```bash
# Git backend - Create .env file with:
# TEST_GITHUB_ORG=your-org
# TEST_GITHUB_PAT_TOKEN=ghp_xxxxx
go test ./backends/gitporcelain -v

# S3 backend - Create .env file with:
# TEST_S3_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
# TEST_S3_ACCESS_KEY_ID=your_key
# TEST_S3_SECRET_ACCESS_KEY=your_secret
# TEST_S3_BUCKET=your_bucket
go test ./backends/s3 -v
```

### Debugging Performance
```bash
# Generate Git backend trace
cd backends/gitporcelain
./trace_test.sh

# Generate S3 backend trace
cd backends/s3
./trace_test.sh

# View trace
go tool trace testdata/trace.out
```

See `backends/s3/TRACING.md` for detailed comparison guide.
