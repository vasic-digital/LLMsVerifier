package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"llm-verifier/config"
	"llm-verifier/providers"
)

// MockAPIServer provides a comprehensive mock API server for testing
type MockAPIServer struct {
	server         *http.Server
	models         []providers.ModelInfo
	mu             sync.RWMutex
	simulateErrors bool
	rateLimitCount map[string]int
	errorConfig    ErrorConfig
}

type ErrorConfig struct {
	SimulateErrors bool     `json:"simulate_errors"`
	ErrorRate      float64  `json:"error_rate"`
	ErrorTypes     []string `json:"error_types"`
}

// EmbeddingResponse represents embedding generation response
type EmbeddingResponse struct {
	Object string                 `json:"object"`
	Data   []float64              `json:"data"`
	Model  string                 `json:"model"`
	Usage  map[string]interface{} `json:"usage"`
}

// ModerationRequest represents moderation request
type ModerationRequest struct {
	Input string `json:"input"`
}

// ModerationResponse represents moderation response
type ModerationResponse struct {
	Results []struct {
		Categories map[string]interface{} `json:"categories"`
		Flagged    bool                   `json:"flagged"`
		Scores     map[string]interface{} `json:"scores"`
	} `json:"results"`
}

// NewMockAPIServer creates a new mock API server
func NewMockAPIServer(models []providers.ModelInfo, errorConfig ErrorConfig) *MockAPIServer {
	// Default error config if not provided
	defaultConfig := ErrorConfig{
		SimulateErrors: false,
		ErrorRate:      0.05,
		ErrorTypes:     []string{"rate_limit", "timeout", "internal_error"},
	}

	if errorConfig.ErrorRate == 0 {
		errorConfig = defaultConfig
	}

	return &MockAPIServer{
		models:         models,
		simulateErrors: errorConfig.SimulateErrors,
		rateLimitCount: make(map[string]int),
		errorConfig:    errorConfig,
	}
}

// Start starts the mock API server
func (m *MockAPIServer) Start(port int) error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/", m.handleRoot)
	mux.HandleFunc("/v1/models", m.handleModels)
	mux.HandleFunc("/v1/embeddings", m.handleEmbeddings)
	mux.HandleFunc("/v1/chat/completions", m.handleChatCompletions)
	mux.HandleFunc("/v1/moderations", m.handleModerations)

	m.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Printf("Mock API server starting on port %d", port)
	log.Printf("Simulating errors: %v", m.simulateErrors)
	log.Printf("Error rate: %.2f", m.errorConfig.ErrorRate)
	log.Printf("Error types: %v", m.errorConfig.ErrorTypes)

	return m.server.ListenAndServe()
}

// Stop stops the mock API server
func (m *MockAPIServer) Stop() {
	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		m.server.Shutdown(ctx)
		cancel()
	}
}

// URL returns the server URL
func (m *MockAPIServer) URL() string {
	return m.server.Addr
}

// authenticateRequest validates API key
func (m *MockAPIServer) authenticateRequest(r *http.Request) bool {
	apiKey := r.Header.Get("Authorization")
	if apiKey == "" {
		return false
	}

	// Simple API key validation (in real implementation, this would check against a database)
	expectedKeys := []string{"sk-mock-key-1", "sk-mock-key-2", "sk-mock-key-3"}
	for _, key := range expectedKeys {
		if strings.Contains(apiKey, key) {
			return true
		}
	}

	return false
}

// simulateRateLimitCheck simulates rate limiting
func (m *MockAPIServer) simulateRateLimitCheck(r *http.Request) bool {
	clientIP := strings.Split(r.RemoteAddr, ":")[0]

	m.mu.Lock()
	count := m.rateLimitCount[clientIP]
	m.rateLimitCount[clientIP]++
	m.mu.Unlock()

	// Allow 10 requests per minute, then rate limit
	if count > 10 {
		return true
	}

	return false
}

// simulateError simulates different types of errors
func (m *MockAPIServer) simulateError(w http.ResponseWriter, r *http.Request) bool {
	if !m.simulateErrors {
		return false
	}

	// Simulate error based on configured rate
	if m.errorConfig.ErrorRate > 0 && rand.Float64() < m.errorConfig.ErrorRate {
		errorType := m.errorConfig.ErrorTypes[rand.Intn(len(m.errorConfig.ErrorTypes))]
		m.writeError(w, errorType)
		return true
	}

	// Simulate rate limiting
	if m.simulateRateLimitCheck(r) {
		m.writeError(w, "rate_limit")
		return true
	}

	return false
}

// writeError writes an error response
func (m *MockAPIServer) writeError(w http.ResponseWriter, errorType string) {
	var statusCode int
	var errorResponse map[string]interface{}

	switch errorType {
	case "rate_limit":
		statusCode = http.StatusTooManyRequests
		errorResponse = map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Rate limit exceeded",
				"type":    "rate_limit_error",
				"code":    "rate_limit_exceeded",
			},
		}
	case "timeout":
		statusCode = http.StatusGatewayTimeout
		errorResponse = map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Request timeout",
				"type":    "timeout_error",
				"code":    "request_timeout",
			},
		}
	case "internal_error":
		statusCode = http.StatusInternalServerError
		errorResponse = map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Internal server error",
				"type":    "internal_error",
				"code":    "internal_server_error",
			},
		}
	default:
		statusCode = http.StatusInternalServerError
		errorResponse = map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Unknown error",
				"type":    "unknown_error",
				"code":    "unknown_error",
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

// setRateLimitHeaders sets rate limiting headers
func (m *MockAPIServer) setRateLimitHeaders(w http.ResponseWriter) {
	w.Header().Set("X-RateLimit-Limit", "100")
	w.Header().Set("X-RateLimit-Remaining", "90")
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(1*time.Minute).Unix()))
}

// handleRoot handles root endpoint
func (m *MockAPIServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message": "LLM Verifier Mock API Server",
		"version": "1.0.0",
		"endpoints": []string{
			"GET /v1/models",
			"POST /v1/embeddings",
			"POST /v1/chat/completions",
			"POST /v1/moderations",
		},
		"status": "running",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleModels handles models listing
func (m *MockAPIServer) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		m.writeError(w, "method_not_allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   m.models,
	})
}

// handleEmbeddings handles embedding generation
func (m *MockAPIServer) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		m.writeError(w, "method_not_allowed")
		return
	}

	if m.simulateError(w, r) {
		return
	}

	var req EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, "invalid_json")
		return
	}

	// Generate mock embeddings based on input
	response := EmbeddingResponse{
		Object: "list",
		Data:   m.generateEmbeddings(req.Input, len(req.Input), req.Model),
		Model:  req.Model,
		Usage: map[string]interface{}{
			"prompt_tokens": len(req.Input),
			"total_tokens":  len(req.Input),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateEmbeddings generates mock embedding data
func (m *MockAPIServer) generateEmbeddings(input string, model string) []float64 {
	// Simple mock: generate random values based on input hash
	embeddings := make([]float64, len(input))
	for i := range embeddings {
		embeddings[i] = float64(i+1) * 0.1 // Simple mock values
	}
	return embeddings
}

// handleChatCompletions handles chat completions
func (m *MockAPIServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		m.writeError(w, "method_not_allowed")
		return
	}

	if m.simulateError(w, r) {
		return
	}

	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, "invalid_json")
		return
	}

	// Generate mock completion based on last message
	response := map[string]interface{}{
		"id":      "chatcmpl-" + fmt.Sprintf("%d", time.Now().Unix()),
		"object":  "chat.completion",
		"created": time.Now().Format(time.RFC3339),
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": fmt.Sprintf("This is a mock response from %s", req.Model),
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     50,
			"completion_tokens": 30,
			"total_tokens":      80,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleModerations handles content moderation
func (m *MockAPIServer) handleModerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		m.writeError(w, "method_not_allowed")
		return
	}

	if m.simulateError(w, r) {
		return
	}

	var req ModerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, "invalid_json")
		return
	}

	// Generate mock moderation response
	response := ModerationResponse{
		Results: []struct {
			Categories map[string]interface{} `json:"categories"`
			Flagged    bool                   `json:"flagged"`
			Scores     map[string]interface{} `json:"scores"`
		}{
			{
				Categories: map[string]interface{}{
					"sexual":    false,
					"violence":  false,
					"self_harm": false,
				},
				Flagged: false,
				Scores: map[string]interface{}{
					"sexual":    0.0,
					"violence":  0.0,
					"self_harm": 0.0,
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
