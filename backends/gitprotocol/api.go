package gitprotocol

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"sync"

	"github.com/cenkalti/backoff/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/go-git/go-git/v6/storage/memory"
	gitbackedrest "github.com/theothertomelliott/git-backed-rest"

	_ "github.com/go-git/go-git/v6/plumbing/transport/file"
	_ "github.com/go-git/go-git/v6/plumbing/transport/git"
	_ "github.com/go-git/go-git/v6/plumbing/transport/ssh"
)

var _ gitbackedrest.APIBackend = (*Backend)(nil)

func NewBackend(endpoint string) (*Backend, error) {
	return NewBackendWithAuth(endpoint, nil)
}

// NewBackendWithAuth creates a new Backend with authentication.
// For HTTP/HTTPS endpoints, you can provide:
//   - *http.BasicAuth for username/password or token authentication
//     (GitHub, GitLab, and Bitbucket use BasicAuth with tokens as the password)
//   - *http.TokenAuth for bearer token authentication
//   - nil for no authentication
func NewBackendWithAuth(endpoint string, auth transport.AuthMethod) (*Backend, error) {
	ep, err := transport.NewEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("creating transport endpoint: %w", err)
	}

	c, err := transport.Get(ep.Scheme)
	if err != nil {
		return nil, fmt.Errorf("getting transport: %w", err)
	}

	store := memory.NewStorage()

	sess, err := c.NewSession(store, ep, auth)
	if err != nil {
		return nil, err
	}

	b := &Backend{
		endpoint:   endpoint,
		transport:  c,
		ep:         ep,
		store:      store,
		session:    sess,
		lockWrites: true,
	}

	// Create connections to warm up the transport
	_, err = b.getReadConnection(context.Background())
	if err != nil {
		return nil, fmt.Errorf("warming up: %w", err)
	}
	if auth != nil {
		_, err = b.getWriteConnection(context.Background())
		if err != nil {
			return nil, fmt.Errorf("warming up: %w", err)
		}
	}

	return b, nil
}

type Backend struct {
	endpoint string

	transport transport.Transport
	ep        *transport.Endpoint
	storeMtx  sync.Mutex
	store     *memory.Storage

	session transport.Session

	writeMtx   sync.Mutex
	lockWrites bool
}

// GetEndpoint returns the endpoint used by the backend.
func (b *Backend) GetEndpoint() string {
	return b.endpoint
}

// DELETE implements gitbackedrest.APIBackend.
func (b *Backend) DELETE(ctx context.Context, path string) *gitbackedrest.APIError {
	defer trace.StartRegion(ctx, "DELETE").End()

	if b.lockWrites {
		b.writeMtx.Lock()
		defer b.writeMtx.Unlock()
	}

	operation := func() (plumbing.Hash, error) {
		commit, err := b.simplePOST(ctx, path, nil, false)
		if err != nil {
			if err == gitbackedrest.ErrNotFound || err == gitbackedrest.ErrInternalServerError {
				return plumbing.ZeroHash, backoff.Permanent(err)
			}
			fmt.Println("Error, will retry:", err)
			return plumbing.ZeroHash, err
		}
		return commit, nil
	}

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		if errors.Is(err, gitbackedrest.ErrNotFound) {
			return gitbackedrest.ErrNotFound
		}
		fmt.Println("Error:", err)
		return gitbackedrest.ErrInternalServerError
	}

	return nil
}

// GET implements gitbackedrest.APIBackend.
func (b *Backend) GET(ctx context.Context, path string) ([]byte, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "GET").End()

	result, err := b.simpleGET(ctx, path)
	if err != nil {
		return nil, gitbackedrest.ErrInternalServerError
	}
	if result == nil {
		return nil, gitbackedrest.ErrNotFound
	}
	return result, nil
}

// POST implements gitbackedrest.APIBackend.
func (b *Backend) POST(ctx context.Context, path string, body []byte) *gitbackedrest.APIError {
	defer trace.StartRegion(ctx, "POST").End()

	if b.lockWrites {
		b.writeMtx.Lock()
		defer b.writeMtx.Unlock()
	}

	operation := func() (plumbing.Hash, error) {
		commit, err := b.simplePOST(ctx, path, body, true)
		if err != nil {
			if err == gitbackedrest.ErrConflict {
				return plumbing.ZeroHash, backoff.Permanent(gitbackedrest.ErrConflict)
			}
			return plumbing.ZeroHash, err
		}
		return commit, nil
	}

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		if errors.Is(err, gitbackedrest.ErrConflict) {
			return gitbackedrest.ErrConflict
		}
		fmt.Println("Error:", err)
		return gitbackedrest.ErrInternalServerError
	}

	return nil
}

// PUT implements gitbackedrest.APIBackend.
func (b *Backend) PUT(ctx context.Context, path string, body []byte) *gitbackedrest.APIError {
	defer trace.StartRegion(ctx, "PUT").End()

	if b.lockWrites {
		b.writeMtx.Lock()
		defer b.writeMtx.Unlock()
	}

	operation := func() (plumbing.Hash, error) {
		commit, err := b.simplePOST(ctx, path, body, false)
		if err != nil {
			if err == gitbackedrest.ErrNotFound || err == gitbackedrest.ErrInternalServerError {
				return plumbing.ZeroHash, backoff.Permanent(err)
			}
			fmt.Println("Error, will retry:", err)
			return plumbing.ZeroHash, err
		}
		return commit, nil
	}

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		if errors.Is(err, gitbackedrest.ErrNotFound) {
			return gitbackedrest.ErrNotFound
		}
		fmt.Println("Error:", err)
		return gitbackedrest.ErrInternalServerError
	}

	return nil
}

func (b *Backend) Close() error {
	return nil
}
