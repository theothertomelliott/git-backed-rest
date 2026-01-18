package gitprotocol

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestConcurrentAccessNoLock(t *testing.T) {
	if true {
		t.Skip("Skipping concurrent checks")
	}

	remote, cleanup := createTestGitHubRepo(t)
	if os.Getenv("SKIP_CLEANUP") == "" {
		t.Cleanup(cleanup)
	} else {
		t.Logf("Remote will be retained at %s", remote)
	}

	auth := getAuthForEndpoint(t, remote)

	backend, err := NewBackendWithAuth(remote, auth)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	backend.lockWrites = false

	fname := func(i int) string {
		return fmt.Sprintf("file%d", i)
	}

	ctx := t.Context()
	wg := sync.WaitGroup{}
	for i := range 10 {
		wg.Go(func() {
			//i := i

			start := time.Now()

			if _, err := backend.POST(ctx, fname(i), fmt.Appendf(nil, "content%d", i)); err != nil {
				t.Errorf("%d: Error on POST: %v", i, err)
				return
			}

			result, err := backend.GET(ctx, fname(i))
			if err != nil {
				t.Errorf("%d: Error on GET: %v", i, err)
				return
			}
			if string(result.Data) != fmt.Sprintf("content%d", i) {
				t.Errorf("%d: Expected content%d, got %s", i, i, string(result.Data))
			}

			t.Logf("%d: successful after %s", i, time.Since(start))
		})
	}
	wg.Wait()

	t.Log("Verifying with GETs")

	for i := range 10 {
		result, err := backend.GET(ctx, fname(i))
		if err != nil {
			t.Errorf("%d: Error on GET: %v", i, err)
		}

		if string(result.Data) != fmt.Sprintf("content%d", i) {
			t.Errorf("%d: Content mismatch: expected %s, got %s", i, fmt.Sprintf("content%d", i), string(result.Data))
		}
	}

}

func TestConcurrentAccessWithLock(t *testing.T) {
	if true {
		t.Skip("Skipping concurrent checks")
	}

	remote, cleanup := createTestGitHubRepo(t)
	if os.Getenv("SKIP_CLEANUP") == "" {
		t.Cleanup(cleanup)
	} else {
		t.Logf("Remote will be retained at %s", remote)
	}

	auth := getAuthForEndpoint(t, remote)

	backend, err := NewBackendWithAuth(remote, auth)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	fname := func(i int) string {
		return fmt.Sprintf("file%d", i)
	}

	ctx := t.Context()
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Go(func() {
			i := i

			start := time.Now()

			if _, err := backend.POST(ctx, fname(i), []byte(fmt.Sprintf("content%d", i))); err != nil {
				t.Errorf("%d: Error on POST: %v", i, err)
				return
			}

			result, err := backend.GET(ctx, fname(i))
			if err != nil {
				t.Errorf("%d: Error on GET: %v", i, err)
				return
			}

			if string(result.Data) != fmt.Sprintf("content%d", i) {
				t.Errorf("%d: Content mismatch: expected %s, got %s", i, fmt.Sprintf("content%d", i), string(result.Data))
				return
			}

			t.Logf("%d: successful after %s", i, time.Since(start))
		})
	}
	wg.Wait()

	t.Log("Verifying with GETs")

	for i := 0; i < 10; i++ {
		result, err := backend.GET(ctx, fname(i))
		if err != nil {
			t.Errorf("%d: Error on GET: %v", i, err)
		}

		if string(result.Data) != fmt.Sprintf("content%d", i) {
			t.Errorf("%d: Content mismatch: expected %s, got %s", i, fmt.Sprintf("content%d", i), string(result.Data))
		}
	}

}
