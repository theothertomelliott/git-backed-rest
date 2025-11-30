package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theothertomelliott/git-backed-rest/backends/memory"
)

func BenchmarkInMemoryServer(b *testing.B) {
	backend := memory.NewBackend()
	server := &Server{
		backend: backend,
	}

	req, err := http.NewRequest("POST", "/doc1", nil)
	if err != nil {
		b.Fatal(err)
	}
	req.Body = io.NopCloser(bytes.NewBufferString("blank"))

	resp := httptest.NewRecorder()
	server.handleRequest(resp, req)

	for i := 0; i < b.N; i++ {
		req, err = http.NewRequest("PUT", "/doc1", nil)
		if err != nil {
			b.Fatal(err)
		}
		req.Body = io.NopCloser(bytes.NewBufferString(fmt.Sprintf("content%v", i)))
		server.handleRequest(resp, req)

		req, err = http.NewRequest("GET", "/doc1", nil)
		if err != nil {
			b.Fatal(err)
		}
		server.handleRequest(resp, req)
	}
}
