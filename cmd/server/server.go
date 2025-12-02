package main

import (
	"fmt"
	"io"
	"net/http"

	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)

type Server struct {
	backend gitbackedrest.APIBackend
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
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
	if r.Body == nil {
		http.Error(w, "Request body is required", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.backend.POST(r.Context(), r.URL.Path, body); err != nil {
		http.Error(w, err.Error(), err.Code)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
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
