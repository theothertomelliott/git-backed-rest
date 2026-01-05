package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)

type Server struct {
	backend gitbackedrest.APIBackend
	metrics *MetricsUpdater
}

func New(backend gitbackedrest.APIBackend) *Server {
	return &Server{
		backend: backend,
		metrics: NewMetricsUpdater(),
	}
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Handle metrics endpoint first - don't update uptime for metrics scraping
	if r.URL.Path == "/metrics" {
		promhttp.Handler().ServeHTTP(w, r)
		return
	}

	// Update uptime metric for API requests only
	s.metrics.UpdateUptime()

	start := time.Now()
	log.Printf("Server: Received %s %s", r.Method, r.URL.Path)

	var status string
	var retries int
	defer func() {
		duration := time.Since(start).Seconds()
		retryLabel := "false"
		if retries > 0 {
			retryLabel = "true"
		}

		RequestDuration.WithLabelValues(r.Method, status, retryLabel).Observe(duration)
		RequestCount.WithLabelValues(r.Method, status, retryLabel).Inc()

		// Track retry attempts if any
		if retries > 0 {
			RetryCount.WithLabelValues(r.Method, status).Add(float64(retries))
		}
	}()

	switch r.Method {
	case http.MethodGet:
		status, retries = s.handleGET(w, r)
	case http.MethodPost:
		status, retries = s.handlePOST(w, r)
	case http.MethodPut:
		status, retries = s.handlePUT(w, r)
	case http.MethodDelete:
		status, retries = s.handleDELETE(w, r)
	default:
		log.Printf("Server: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		status = "error"
		retries = 0
	}
}

func (s *Server) handleGET(w http.ResponseWriter, r *http.Request) (string, int) {
	newCtx, body, err := s.backend.GET(r.Context(), r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), err.Code)
		return "error", err.Retries
	}

	// Use updated context to get retry count
	retries := gitbackedrest.GetRetryCount(newCtx)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", string(body))
	return "success", retries
}

func (s *Server) handlePOST(w http.ResponseWriter, r *http.Request) (string, int) {
	log.Printf("Server: handlePOST started for %s", r.URL.Path)

	if r.Body == nil {
		log.Printf("Server: Request body is nil")
		http.Error(w, "Request body is required", http.StatusBadRequest)
		return "error", 0
	}

	log.Printf("Server: Reading request body...")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Server: Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return "error", 0
	}

	log.Printf("Server: Calling backend.POST...")
	newCtx, apiErr := s.backend.POST(r.Context(), r.URL.Path, body)
	if apiErr != nil {
		log.Printf("Server: backend.POST failed: %v", apiErr)
		http.Error(w, apiErr.Error(), apiErr.Code)
		return "error", apiErr.Retries
	}

	// Use updated context to get retry count
	retries := gitbackedrest.GetRetryCount(newCtx)

	log.Printf("Server: backend.POST succeeded with %d retries", retries)
	w.WriteHeader(http.StatusCreated)
	return "success", retries
}

func (s *Server) handlePUT(w http.ResponseWriter, r *http.Request) (string, int) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "error", 0
	}
	newCtx, apiErr := s.backend.PUT(r.Context(), r.URL.Path, body)
	if apiErr != nil {
		http.Error(w, apiErr.Error(), apiErr.Code)
		return "error", apiErr.Retries
	}

	// Use updated context to get retry count
	retries := gitbackedrest.GetRetryCount(newCtx)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	return "success", retries
}

func (s *Server) handleDELETE(w http.ResponseWriter, r *http.Request) (string, int) {
	newCtx, apiErr := s.backend.DELETE(r.Context(), r.URL.Path)
	if apiErr != nil {
		http.Error(w, apiErr.Error(), apiErr.Code)
		return "error", apiErr.Retries
	}

	// Use updated context to get retry count
	retries := gitbackedrest.GetRetryCount(newCtx)

	w.WriteHeader(http.StatusNoContent)
	return "success", retries
}
