package gitprotocol

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"runtime/trace"
	"sync"
	"time"

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

	b := &Backend{
		endpoint:   endpoint,
		auth:       auth,
		lockWrites: true,

		ep:        ep,
		transport: c,
	}

	err = b.newSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
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
	auth     transport.AuthMethod

	transport transport.Transport
	ep        *transport.Endpoint
	storeMtx  sync.Mutex
	store     *memory.Storage

	session    transport.Session
	sessionMtx sync.RWMutex

	writeMtx   sync.Mutex
	lockWrites bool
}

func (b *Backend) newSession() error {
	b.sessionMtx.Lock()
	defer b.sessionMtx.Unlock()

	store := memory.NewStorage()

	sess, err := b.transport.NewSession(store, b.ep, b.auth)
	if err != nil {
		return err
	}

	b.store = store
	b.session = sess

	go func() {
		// Set aggressive GC for this backend
		debug.SetGCPercent(50) // More aggressive than default 100

		// Set soft memory limit (e.g., 200MB)
		debug.SetMemoryLimit(200 * 1024 * 1024)

		// Clean up objects every 10s
		for range time.Tick(10 * time.Second) {
			b.sessionMtx.Lock()

			// Clear storage objects
			b.store.ObjectStorage.Objects = make(map[plumbing.Hash]plumbing.EncodedObject)
			b.store.ObjectStorage.Commits = make(map[plumbing.Hash]plumbing.EncodedObject)
			b.store.ObjectStorage.Trees = make(map[plumbing.Hash]plumbing.EncodedObject)
			b.store.ObjectStorage.Blobs = make(map[plumbing.Hash]plumbing.EncodedObject)
			b.store.ObjectStorage.Tags = make(map[plumbing.Hash]plumbing.EncodedObject)

			b.sessionMtx.Unlock()

			// Force garbage collection after cleanup
			runtime.GC()

			// Log memory usage for debugging
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			fmt.Printf("Memory after cleanup - HeapAlloc: %d KB, HeapSys: %d KB\n",
				m.HeapAlloc/1024, m.HeapSys/1024)
		}
	}()

	return nil
}

// GetEndpoint returns the endpoint used by the backend.
func (b *Backend) GetEndpoint() string {
	return b.endpoint
}

// DELETE implements gitbackedrest.APIBackend.
func (b *Backend) DELETE(ctx context.Context, path string) (context.Context, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "DELETE").End()

	b.sessionMtx.RLock()
	defer b.sessionMtx.RUnlock()

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
			return ctx, gitbackedrest.ErrNotFound
		}
		fmt.Println("Error:", err)
		return ctx, gitbackedrest.ErrInternalServerError
	}

	return ctx, nil
}

// GET implements gitbackedrest.APIBackend.
func (b *Backend) GET(ctx context.Context, path string) (context.Context, []byte, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "GET").End()

	b.sessionMtx.RLock()
	defer b.sessionMtx.RUnlock()

	result, err := b.simpleGET(ctx, path)
	if err != nil {
		return ctx, nil, gitbackedrest.ErrInternalServerError
	}
	if result == nil {
		return ctx, nil, gitbackedrest.ErrNotFound
	}
	return ctx, result, nil
}

// POST implements gitbackedrest.APIBackend.
func (b *Backend) POST(ctx context.Context, path string, body []byte) (context.Context, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "POST").End()

	b.sessionMtx.RLock()
	defer b.sessionMtx.RUnlock()

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
			if err == gitbackedrest.ErrInternalServerError {
				return plumbing.ZeroHash, backoff.Permanent(gitbackedrest.ErrInternalServerError)
			}
			return plumbing.ZeroHash, err
		}
		return commit, nil
	}

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		if errors.Is(err, gitbackedrest.ErrConflict) {
			return ctx, gitbackedrest.ErrConflict
		}
		fmt.Println("Error:", err)
		return ctx, gitbackedrest.ErrInternalServerError
	}

	return ctx, nil
}

// PUT implements gitbackedrest.APIBackend.
func (b *Backend) PUT(ctx context.Context, path string, body []byte) (context.Context, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "PUT").End()

	b.sessionMtx.RLock()
	defer b.sessionMtx.RUnlock()

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
			return ctx, gitbackedrest.ErrNotFound
		}
		fmt.Println("Error:", err)
		return ctx, gitbackedrest.ErrInternalServerError
	}

	return ctx, nil
}

func (b *Backend) Close() error {
	return nil
}
