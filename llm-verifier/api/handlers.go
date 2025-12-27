// Package api contains HTTP API handlers
package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// HealthHandler handles health check requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// ListModelsHandler handles listing all models
func (s *Server) ListModelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	demoModels := []map[string]any{
		{
			"id":           "1",
			"name":         "GPT-4",
			"provider":     "OpenAI",
			"status":       "verified",
			"score":        95,
			"capabilities": []string{"text", "code", "reasoning"},
		},
		{
			"id":           "2",
			"name":         "Claude-3",
			"provider":     "Anthropic",
			"status":       "verified",
			"score":        92,
			"capabilities": []string{"text", "analysis", "safety"},
		},
		{
			"id":           "3",
			"name":         "Gemini Pro",
			"provider":     "Google",
			"status":       "pending",
			"score":        0,
			"capabilities": []string{"text", "multimodal", "reasoning"},
		},
	}
	json.NewEncoder(w).Encode(map[string]any{
		"models": demoModels,
	})
}

// GetModelHandler handles getting a single model
func (s *Server) GetModelHandler(w http.ResponseWriter, r *http.Request) {
	// Extract model ID from path: /api/models/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/models/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	modelID := path

	w.Header().Set("Content-Type", "application/json")

	demoModel := map[string]any{
		"id":           modelID,
		"name":         "GPT-4",
		"provider":     "OpenAI",
		"status":       "verified",
		"score":        95,
		"capabilities": []string{"text", "code", "reasoning"},
		"description":  "OpenAI's most capable model for complex tasks",
		"parameters":   "1.76 trillion",
		"context":      "128K tokens",
	}

	json.NewEncoder(w).Encode(demoModel)
}

// VerifyModelHandler handles model verification
func (s *Server) VerifyModelHandler(w http.ResponseWriter, r *http.Request) {
	// Extract model ID from path: /api/models/{id}/verify
	path := strings.TrimPrefix(r.URL.Path, "/api/models/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	modelID := parts[0]

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"status":  "verification_started",
		"model":   modelID,
		"message": "Verification process initiated",
		"job_id":  "verification_" + modelID,
	})
}

// ListProvidersHandler handles listing all providers
func (s *Server) ListProvidersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	demoProviders := []map[string]any{
		{
			"id":      "1",
			"name":    "OpenAI",
			"status":  "active",
			"models":  15,
			"api_url": "https://api.openai.com/v1",
		},
		{
			"id":      "2",
			"name":    "Anthropic",
			"status":  "active",
			"models":  8,
			"api_url": "https://api.anthropic.com/v1",
		},
		{
			"id":      "3",
			"name":    "Google",
			"status":  "active",
			"models":  12,
			"api_url": "https://generativelanguage.googleapis.com/v1",
		},
	}

	json.NewEncoder(w).Encode(demoProviders)
}

// ProvidersHandler handles both GET (list) and POST (add) for providers
func (s *Server) ProvidersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.ListProvidersHandler(w, r)
	case http.MethodPost:
		s.AddProviderHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// AddProviderHandler handles adding a new provider
func (s *Server) AddProviderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var providerData map[string]any
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&providerData); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]any{
		"status": "provider_added",
		"id":     "new_provider",
		"name":   providerData["name"],
	})
}
