// Package api contains HTTP API handlers
// Temporarily simplified due to dependency issues
package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthHandler handles health check requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// ListModelsHandler handles listing all models
func (s *Server) ListModelsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement proper database query
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]interface{}{})
}

// GetModelHandler handles getting a single model
func (s *Server) GetModelHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement proper database query
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   r.PathValue("id"),
		"name": "placeholder",
	})
}

// VerifyModelHandler handles model verification
func (s *Server) VerifyModelHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement model verification
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "verification_pending",
		"message": "Verification system temporarily disabled",
	})
}

// AddProviderHandler handles adding a new provider
func (s *Server) AddProviderHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement provider addition
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "provider_added",
		"name":   "placeholder",
	})
}
