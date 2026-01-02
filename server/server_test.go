package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theothertomelliott/git-backed-rest/backends/memory"
)

func TestServerGET(t *testing.T) {
	server := &Server{
		backend: memory.NewBackend(),
	}

	req, err := http.NewRequest("GET", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp := httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, resp.Code)
	}

	if err := server.backend.POST(req.Context(), "/doc1", []byte("content1")); err != nil {
		t.Fatal(err)
	}

	resp = httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.Code)
	}
}

func TestServerPOST(t *testing.T) {
	server := &Server{
		backend: memory.NewBackend(),
	}

	req, err := http.NewRequest("POST", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = io.NopCloser(bytes.NewBufferString("content1"))

	resp := httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status code %d, got %d: %v", http.StatusCreated, resp.Code, resp.Body)
	}

	resp = httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusConflict {
		t.Errorf("expected status code %d, got %d: %v", http.StatusConflict, resp.Code, resp.Body)
	}
}

func TestServerPUT(t *testing.T) {
	server := &Server{
		backend: memory.NewBackend(),
	}

	req, err := http.NewRequest("PUT", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = io.NopCloser(bytes.NewBufferString("content2"))

	resp := httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d: %v", http.StatusNotFound, resp.Code, resp.Body)
	}

	req, err = http.NewRequest("POST", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = io.NopCloser(bytes.NewBufferString("content1"))

	resp = httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status code %d, got %d: %v", http.StatusCreated, resp.Code, resp.Body)
	}

	req, err = http.NewRequest("GET", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp = httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d: %v", http.StatusOK, resp.Code, resp.Body)
	}
	if resp.Body.String() != "content1" {
		t.Errorf("expected body %s, got %s", "content1", resp.Body.String())
	}

	req, err = http.NewRequest("PUT", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = io.NopCloser(bytes.NewBufferString("content2"))

	resp = httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Errorf("expected status code %d, got %d: %v", http.StatusNoContent, resp.Code, resp.Body)
	}

	req, err = http.NewRequest("GET", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp = httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d: %v", http.StatusOK, resp.Code, resp.Body)
	}
	if resp.Body.String() != "content2" {
		t.Errorf("expected body %s, got %s", "content2", resp.Body.String())
	}
}

func TestServerDELETE(t *testing.T) {
	server := &Server{
		backend: memory.NewBackend(),
	}

	req, err := http.NewRequest("DELETE", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp := httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d: %v", http.StatusNotFound, resp.Code, resp.Body)
	}

	req, err = http.NewRequest("POST", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = io.NopCloser(bytes.NewBufferString("content1"))

	resp = httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status code %d, got %d: %v", http.StatusCreated, resp.Code, resp.Body)
	}

	req, err = http.NewRequest("DELETE", "/doc1", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp = httptest.NewRecorder()
	server.HandleRequest(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Errorf("expected status code %d, got %d: %v", http.StatusNoContent, resp.Code, resp.Body)
	}
}
