package server

import (
	"fmt"
	"io"
	"log"
	"net/http"

	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)

type Server struct {
	backend gitbackedrest.APIBackend
}

func New(backend gitbackedrest.APIBackend) *Server {
	return &Server{
		backend: backend,
	}
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Server: Received %s %s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		s.handleGET(w, r)
	case http.MethodPost:
		s.handlePOST(w, r)
	case http.MethodPut:
		s.handlePUT(w, r)
	case http.MethodDelete:
		s.handleDELETE(w, r)
	default:
		log.Printf("Server: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGET(w http.ResponseWriter, r *http.Request) {
	body, err := s.backend.GET(r.Context(), r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), err.Code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", string(body))
}

func (s *Server) handlePOST(w http.ResponseWriter, r *http.Request) {
	log.Printf("Server: handlePOST started for %s", r.URL.Path)

	if r.Body == nil {
		log.Printf("Server: Request body is nil")
		http.Error(w, "Request body is required", http.StatusBadRequest)
		return
	}

	log.Printf("Server: Reading request body...")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Server: Failed to read body: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Server: Calling backend.POST for %s", r.URL.Path)
	if err := s.backend.POST(r.Context(), r.URL.Path, body); err != nil {
		log.Printf("Server: backend.POST failed: %v", err)
		http.Error(w, err.Error(), err.Code)
		return
	}

	log.Printf("Server: backend.POST successful, sending response")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	log.Printf("Server: handlePOST completed for %s", r.URL.Path)
}

func (s *Server) handlePUT(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.backend.PUT(r.Context(), r.URL.Path, body); err != nil {
		http.Error(w, err.Error(), err.Code)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDELETE(w http.ResponseWriter, r *http.Request) {
	if err := s.backend.DELETE(r.Context(), r.URL.Path); err != nil {
		http.Error(w, err.Error(), err.Code)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
