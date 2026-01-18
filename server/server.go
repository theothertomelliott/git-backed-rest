package server

import (
	"errors"
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

// handleError handles API errors by extracting status code and user message, then writing HTTP error response
func (s *Server) handleError(w http.ResponseWriter, err error) (string, int) {
	statusCode := gitbackedrest.GetHTTPStatusCode(err, http.StatusInternalServerError)
	userMessage := gitbackedrest.GetUserMessage(err)
	http.Error(w, userMessage, statusCode)
	return "error", 0
}

func (s *Server) handleGET(w http.ResponseWriter, r *http.Request) (string, int) {
	result, err := s.backend.GET(r.Context(), r.URL.Path)
	if err != nil {
		return s.handleError(w, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", string(result.Data))
	return "success", result.Retries
}

func (s *Server) handlePOST(w http.ResponseWriter, r *http.Request) (string, int) {
	log.Printf("Server: handlePOST started for %s", r.URL.Path)

	if r.Body == nil {
		log.Printf("Server: Request body is nil")
		err := gitbackedrest.NewUserError(
			"Request body is required",
			gitbackedrest.NewHTTPError(
				http.StatusBadRequest,
				errors.New("request body is required"),
			),
		)
		return s.handleError(w, err)
	}

	log.Printf("Server: Reading request body...")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Server: Error reading request body: %v", err)
		err = gitbackedrest.NewUserError(
			"Error reading request body",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("reading request body: %w", err),
			),
		)
		return s.handleError(w, err)
	}

	log.Printf("Server: Calling backend.POST...")
	result, apiErr := s.backend.POST(r.Context(), r.URL.Path, body)
	if apiErr != nil {
		log.Printf("Server: backend.POST failed: %v", apiErr)
		return s.handleError(w, apiErr)
	}

	log.Printf("Server: backend.POST succeeded with %d retries", result.Retries)
	w.WriteHeader(http.StatusCreated)
	return "success", result.Retries
}

func (s *Server) handlePUT(w http.ResponseWriter, r *http.Request) (string, int) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		err = gitbackedrest.NewUserError(
			"Error reading request body",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("reading request body: %w", err),
			),
		)
		return s.handleError(w, err)
	}
	result, apiErr := s.backend.PUT(r.Context(), r.URL.Path, body)
	if apiErr != nil {
		return s.handleError(w, apiErr)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	return "success", result.Retries
}

func (s *Server) handleDELETE(w http.ResponseWriter, r *http.Request) (string, int) {
	result, apiErr := s.backend.DELETE(r.Context(), r.URL.Path)
	if apiErr != nil {
		return s.handleError(w, apiErr)
	}

	w.WriteHeader(http.StatusNoContent)
	return "success", result.Retries
}
