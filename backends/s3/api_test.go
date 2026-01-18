package s3

import (
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/trace"
	"testing"

	"github.com/joho/godotenv"
	"github.com/tjarratt/babble"

	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)

func TestGET(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test
	ctx, task := trace.NewTask(ctx, "SetupTestGET")

	reg := trace.StartRegion(ctx, "loadConfig")
	cfg := loadTestConfig(t)
	reg.End()

	reg = trace.StartRegion(ctx, "newBackend")
	backend, err := NewBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	defer ifPassed(t, func() {
		if err := backend.CleanupPrefix(ctx); err != nil {
			t.Errorf("cleanup prefix: %v", err)
		}
	})
	reg.End()
	task.End()

	ctx, task = trace.NewTask(ctx, "TestGET")
	defer task.End()
	docPath := "doc1"
	docContent := "content1"

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

func TestPOST(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test
	ctx, task := trace.NewTask(ctx, "TestPOST")
	defer task.End()

	reg := trace.StartRegion(ctx, "setup")
	cfg := loadTestConfig(t)
	backend, err := NewBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	defer ifPassed(t, func() {
		if err := backend.CleanupPrefix(ctx); err != nil {
			t.Errorf("cleanup prefix: %v", err)
		}
	})
	reg.End()

	docPath := "doc_post"
	docContent := "content_post"

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

	// Try to POST again to same path
	_, postErr2 := backend.POST(ctx, docPath, []byte("different"))
	if postErr2 == nil {
		t.Fatal("expected conflict error on post to existing path")
	}
	statusCode := gitbackedrest.GetHTTPStatusCode(postErr2, 0)
	if statusCode != http.StatusConflict {
		t.Fatalf("expected conflict status, got %d", statusCode)
	}
}

func TestPUT(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test
	ctx, task := trace.NewTask(ctx, "TestPUT")
	defer task.End()

	reg := trace.StartRegion(ctx, "setup")
	cfg := loadTestConfig(t)
	backend, err := NewBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	defer ifPassed(t, func() {
		if err := backend.CleanupPrefix(ctx); err != nil {
			t.Errorf("cleanup prefix: %v", err)
		}
	})
	reg.End()

	docPath := "doc_put"
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

func TestDELETE(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test
	ctx, task := trace.NewTask(ctx, "TestDELETE")
	defer task.End()

	reg := trace.StartRegion(ctx, "setup")
	cfg := loadTestConfig(t)
	backend, err := NewBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	defer ifPassed(t, func() {
		if err := backend.CleanupPrefix(ctx); err != nil {
			t.Errorf("cleanup prefix: %v", err)
		}
	})
	reg.End()

	docPath := "doc_delete"
	docContent := "content_delete"

	t.Log("DELETE - should fail on non-existent path")
	_, deleteErr := backend.DELETE(ctx, docPath)
	if deleteErr == nil {
		t.Fatal("expected not found error on delete of non-existent path")
	}
	statusCode := gitbackedrest.GetHTTPStatusCode(deleteErr, 0)
	if statusCode != http.StatusNotFound {
		t.Fatalf("expected not found status, got %d", statusCode)
	}

	if _, err := backend.POST(ctx, docPath, []byte(docContent)); err != nil {
		t.Fatal(err)
	}

	if _, err := backend.DELETE(ctx, docPath); err != nil {
		t.Fatal(err)
	}

	_, _, getErr := backend.GET(ctx, docPath)
	if getErr == nil {
		t.Fatal("expected error for missing document after delete")
	}
	statusCode = gitbackedrest.GetHTTPStatusCode(getErr, 0)
	if statusCode != http.StatusNotFound {
		t.Fatalf("expected not found status, got %d", statusCode)
	}
}

func TestPrefixIsolation(t *testing.T) {
	ctx := t.Context()

	// Create a logical task for this test
	ctx, task := trace.NewTask(ctx, "TestPrefixIsolation")
	defer task.End()

	reg := trace.StartRegion(ctx, "setup")
	baseCfg := loadTestConfig(t)

	// Create two backends with different prefixes
	cfg1 := baseCfg
	cfg1.Prefix = baseCfg.Prefix + "/isolation-1"
	backend1, err := NewBackend(cfg1)
	if err != nil {
		t.Fatal(err)
	}
	defer backend1.Close()
	defer ifPassed(t, func() {
		if err := backend1.CleanupPrefix(ctx); err != nil {
			t.Errorf("cleanup backend1 prefix: %v", err)
		}
	})

	cfg2 := baseCfg
	cfg2.Prefix = baseCfg.Prefix + "/isolation-2"
	backend2, err := NewBackend(cfg2)
	if err != nil {
		t.Fatal(err)
	}
	defer backend2.Close()
	defer ifPassed(t, func() {
		if err := backend2.CleanupPrefix(ctx); err != nil {
			t.Errorf("cleanup backend2 prefix: %v", err)
		}
	})
	reg.End()

	docPath := "shared-doc"
	content1 := "content-from-backend1"
	content2 := "content-from-backend2"

	// POST to backend1
	if _, err := backend1.POST(ctx, docPath, []byte(content1)); err != nil {
		t.Fatal(err)
	}

	// POST to backend2 with same path should succeed (different prefix)
	if _, err := backend2.POST(ctx, docPath, []byte(content2)); err != nil {
		t.Fatal(err)
	}

	// Verify isolation
	_, body1, getErr := backend1.GET(ctx, docPath)
	if getErr != nil {
		t.Fatalf("backend1.GET failed: %v", getErr)
	}
	if string(body1) != content1 {
		t.Errorf("backend1: expected %s, got %s", content1, string(body1))
	}

	_, body2, getErr := backend2.GET(ctx, docPath)
	if getErr != nil {
		t.Fatalf("backend2.GET failed: %v", getErr)
	}
	if string(body2) != content2 {
		t.Errorf("backend2: expected %s, got %s", content2, string(body2))
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

func loadTestConfig(t *testing.T) Config {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Fatal("TEST_S3_ENDPOINT must be set")
	}
	accessKeyID := os.Getenv("TEST_S3_ACCESS_KEY_ID")
	if accessKeyID == "" {
		t.Fatal("TEST_S3_ACCESS_KEY_ID must be set")
	}
	secretAccessKey := os.Getenv("TEST_S3_SECRET_ACCESS_KEY")
	if secretAccessKey == "" {
		t.Fatal("TEST_S3_SECRET_ACCESS_KEY must be set")
	}
	bucket := os.Getenv("TEST_S3_BUCKET")
	if bucket == "" {
		t.Fatal("TEST_S3_BUCKET must be set")
	}

	// Generate unique prefix for this test run to avoid conflicts
	babbler := babble.NewBabbler()
	babbler.Count = 3
	babbler.Separator = "-"
	prefix := "test/" + babbler.Babble()

	return Config{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Bucket:          bucket,
		Prefix:          prefix,
		Region:          "auto",
	}
}
