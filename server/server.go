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
	defer func() {
		duration := time.Since(start).Seconds()
		RequestDuration.WithLabelValues(r.Method, status).Observe(duration)
		RequestCount.WithLabelValues(r.Method, status).Inc()
	}()

	switch r.Method {
	case http.MethodGet:
		status = s.handleGET(w, r)
	case http.MethodPost:
		status = s.handlePOST(w, r)
	case http.MethodPut:
		status = s.handlePUT(w, r)
	case http.MethodDelete:
		status = s.handleDELETE(w, r)
	default:
		log.Printf("Server: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		status = "error"
	}
}

func (s *Server) handleGET(w http.ResponseWriter, r *http.Request) string {
	body, err := s.backend.GET(r.Context(), r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), err.Code)
		return "error"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", string(body))
	return "success"
}

func (s *Server) handlePOST(w http.ResponseWriter, r *http.Request) string {
	log.Printf("Server: handlePOST started for %s", r.URL.Path)

	if r.Body == nil {
		log.Printf("Server: Request body is nil")
		http.Error(w, "Request body is required", http.StatusBadRequest)
		return "error"
	}

	log.Printf("Server: Reading request body...")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Server: Failed to read body: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "error"
	}

	log.Printf("Server: Calling backend.POST for %s", r.URL.Path)
	if err := s.backend.POST(r.Context(), r.URL.Path, body); err != nil {
		log.Printf("Server: backend.POST failed: %v", err)
		http.Error(w, err.Error(), err.Code)
		return "error"
	}

	log.Printf("Server: backend.POST successful, sending response")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	log.Printf("Server: handlePOST completed for %s", r.URL.Path)
	return "success"
}

func (s *Server) handlePUT(w http.ResponseWriter, r *http.Request) string {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "error"
	}
	if err := s.backend.PUT(r.Context(), r.URL.Path, body); err != nil {
		http.Error(w, err.Error(), err.Code)
		return "error"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	return "success"
}

func (s *Server) handleDELETE(w http.ResponseWriter, r *http.Request) string {
	if err := s.backend.DELETE(r.Context(), r.URL.Path); err != nil {
		http.Error(w, err.Error(), err.Code)
		return "error"
	}
	w.WriteHeader(http.StatusNoContent)
	return "success"
}
