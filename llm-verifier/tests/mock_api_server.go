package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"llm-verifier/config"
	"llm-verifier/providers"
)

// MockAPIServer provides a comprehensive mock API server for testing
type MockAPIServer struct {
	server          *httptest.Server
	models          []providers.ModelInfo
	responses       map[string]interface{}
	rateLimitConfig RateLimitConfig
	errorConfig     ErrorConfig
}

// RateLimitConfig configures rate limiting behavior
type RateLimitConfig struct {
	RequestsPerMinute int
	TokensPerMinute   int
	EnableRateLimit   bool
}

// ErrorConfig configures error simulation
type ErrorConfig struct {
	SimulateErrors      bool
	ErrorRate           float64 // 0.0 to 1.0
	TimeoutRate         float64 // 0.0 to 1.0
	UnauthorizedRate    float64 // 0.0 to 1.0
	ServerErrorRate     float64 // 0.0 to 1.0
}

// NewMockAPIServer creates a new mock API server with comprehensive endpoints
func NewMockAPIServer() *MockAPIServer {
	server := &MockAPIServer{
		models: generateDefaultModels(),
		responses: make(map[string]interface{}),
		rateLimitConfig: RateLimitConfig{
			RequestsPerMinute: 100,
			TokensPerMinute:   10000,
			EnableRateLimit:   true,
		},
		errorConfig: ErrorConfig{
			SimulateErrors:   false,
			ErrorRate:        0.0,
			TimeoutRate:      0.0,
			UnauthorizedRate: 0.0,
			ServerErrorRate:  0.0,
		},
	}
	server.setupDefaultResponses()
	return server
}

// Start starts the mock server
func (m *MockAPIServer) Start() *httptest.Server {
	mux := http.NewServeMux()
	
	// OpenAI API endpoints
	mux.HandleFunc("/v1/models", m.handleModels)
	mux.HandleFunc("/v1/models/", m.handleModel)
	mux.HandleFunc("/v1/chat/completions", m.handleChatCompletions)
	mux.HandleFunc("/v1/completions", m.handleCompletions)
	mux.HandleFunc("/v1/embeddings", m.handleEmbeddings)
	mux.HandleFunc("/v1/moderations", m.handleModerations)
	
	// Image endpoints
	mux.HandleFunc("/v1/images/generations", m.handleImageGenerations)
	mux.HandleFunc("/v1/images/edits", m.handleImageEdits)
	mux.HandleFunc("/v1/images/variations", m.handleImageVariations)
	
	// Audio endpoints
	mux.HandleFunc("/v1/audio/transcriptions", m.handleAudioTranscriptions)
	mux.HandleFunc("/v1/audio/speech", m.handleAudioSpeech)
	
	// Advanced endpoints
	mux.HandleFunc("/v1/fine-tuning/jobs", m.handleFineTuningJobs)
	mux.HandleFunc("/v1/assistants", m.handleAssistants)
	mux.HandleFunc("/v1/threads", m.handleThreads)
	mux.HandleFunc("/v1/files", m.handleFiles)
	
	// Admin endpoints for testing
	mux.HandleFunc("/test/reset", m.handleTestReset)
	mux.HandleFunc("/test/config", m.handleTestConfig)
	
	m.server = httptest.NewServer(mux)
	return m.server
}

// Stop stops the mock server
func (m *MockAPIServer) Stop() {
	if m.server != nil {
		m.server.Close()
	}
}

// URL returns the server URL
func (m *MockAPIServer) URL() string {
	return m.server.URL
}

// SetErrorConfig configures error simulation
func (m *MockAPIServer) SetErrorConfig(config ErrorConfig) {
	m.errorConfig = config
}

// SetRateLimitConfig configures rate limiting
func (m *MockAPIServer) SetRateLimitConfig(config RateLimitConfig) {
	m.rateLimitConfig = config
}

// AddCustomResponse adds a custom response for testing
func (m *MockAPIServer) AddCustomResponse(endpoint string, response interface{}) {
	m.responses[endpoint] = response
}

// generateDefaultModels generates the default model list
func generateDefaultModels() []providers.ModelInfo {
	return []providers.ModelInfo{
		{
			ID:       "gpt-4-turbo",
			Object:   "model",
			Created:  1677649963,
			OwnedBy:  "openai",
		},
		{
			ID:       "gpt-3.5-turbo",
			Object:   "model",
			Created:  1677649963,
			OwnedBy:  "openai",
		},
		{
			ID:       "gpt-4",
			Object:   "model",
			Created:  1687882410,
			OwnedBy:  "openai",
		},
		{
			ID:       "text-embedding-3-small",
			Object:   "model",
			Created:  1695267458,
			OwnedBy:  "openai",
		},
		{
			ID:       "text-embedding-3-large",
			Object:   "model",
			Created:  1695267458,
			OwnedBy:  "openai",
		},
		{
			ID:       "text-embedding-ada-002",
			Object:   "model",
			Created:  1671217299,
			OwnedBy:  "openai",
		},
		{
			ID:       "dall-e-3",
			Object:   "model",
			Created:  1698789744,
			OwnedBy:  "openai",
		},
		{
			ID:       "dall-e-2",
			Object:   "model",
			Created:  1677649963,
			OwnedBy:  "openai",
		},
		{
			ID:       "whisper-1",
			Object:   "model",
			Created:  1677649963,
			OwnedBy:  "openai",
		},
		{
			ID:       "tts-1",
			Object:   "model",
			Created:  1698797732,
			OwnedBy:  "openai",
		},
		{
			ID:       "tts-1-hd",
			Object:   "model",
			Created:  1698797732,
			OwnedBy:  "openai",
		},
	}
}

// setupDefaultResponses sets up default responses for testing
func (m *MockAPIServer) setupDefaultResponses() {
	// Add default responses that can be customized for testing
	m.responses["default_chat"] = map[string]interface{}{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "gpt-3.5-turbo",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "I can help with various tasks including coding, analysis, and general questions.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": 20,
			"total_tokens":      30,
		},
	}
}

// Common request handling
func (m *MockAPIServer) authenticateRequest(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}
	
	// Check for Bearer token
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		return token != "" && token != "invalid-token"
	}
	
	return false
}

func (m *MockAPIServer) setRateLimitHeaders(w http.ResponseWriter) {
	if !m.rateLimitConfig.EnableRateLimit {
		return
	}
	
	w.Header().Set("x-ratelimit-limit-requests", strconv.Itoa(m.rateLimitConfig.RequestsPerMinute))
	w.Header().Set("x-ratelimit-limit-tokens", strconv.Itoa(m.rateLimitConfig.TokensPerMinute))
	w.Header().Set("x-ratelimit-remaining-requests", "90")
	w.Header().Set("x-ratelimit-remaining-tokens", "9000")
	w.Header().Set("x-ratelimit-reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
}

func (m *MockAPIServer) simulateErrors(w http.ResponseWriter, r *http.Request) bool {
	if !m.errorConfig.SimulateErrors {
		return false
	}
	
	// Simulate different types of errors based on rates
	random := float64(time.Now().UnixNano()%1000) / 1000.0
	
	if random < m.errorConfig.TimeoutRate {
		// Simulate timeout
		time.Sleep(5 * time.Second)
		return true
	}
	
	if random < m.errorConfig.UnauthorizedRate {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return true
	}
	
	if random < m.errorConfig.ServerErrorRate {
		http.Error(w, `{"error":{"code":"internal_server_error","message":"Internal server error"}}`, http.StatusInternalServerError)
		return true
	}
	
	if random < m.errorConfig.ErrorRate {
		http.Error(w, `{"error":{"code":"api_error","message":"API error occurred"}}`, http.StatusBadRequest)
		return true
	}
	
	return false
}

func (m *MockAPIServer) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// Models endpoints
func (m *MockAPIServer) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	response := map[string]interface{}{
		"object": "list",
		"data":   m.models,
	}
	
	m.writeJSONResponse(w, response)
}

func (m *MockAPIServer) handleModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	// Extract model ID from URL path
	modelID := strings.TrimPrefix(r.URL.Path, "/v1/models/")
	if modelID == "" {
		http.Error(w, `{"error":{"code":"not_found","message":"Model not found"}}`, http.StatusNotFound)
		return
	}
	
	// Find the model
	var foundModel *providers.ModelInfo
	for _, model := range m.models {
		if model.ID == modelID {
			foundModel = &model
			break
		}
	}
	
	if foundModel == nil {
		http.Error(w, `{"error":{"code":"not_found","message":"Model not found"}}`, http.StatusNotFound)
		return
	}
	
	m.setRateLimitHeaders(w)
	m.writeJSONResponse(w, foundModel)
}

// Chat completions endpoint
func (m *MockAPIServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Invalid request body"}}`, http.StatusBadRequest)
		return
	}
	
	model, ok := request["model"].(string)
	if !ok {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Model is required"}}`, http.StatusBadRequest)
		return
	}
	
	// Generate response based on model
	response := m.generateChatCompletionResponse(model, request)
	m.writeJSONResponse(w, response)
}

func (m *MockAPIServer) generateChatCompletionResponse(model string, request map[string]interface{}) map[string]interface{} {
	// Default response
	response := m.responses["default_chat"].(map[string]interface{})
	
	// Create a copy to avoid modifying the default
	result := make(map[string]interface{})
	for k, v := range response {
		result[k] = v
	}
	result["model"] = model
	
	// Customize based on model type
	switch {
	case strings.Contains(model, "gpt-4"):
		result["choices"] = []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "As an advanced AI model, I can help with complex coding tasks, analysis, and problem-solving. Here's a sophisticated Python solution:\n\n```python\ndef factorial(n):\n    \"\"\"Calculate factorial using memoization for optimization.\"\"\"\n    memo = {}\n    \n    def fact(k):\n        if k in memo:\n            return memo[k]\n        if k <= 1:\n            return 1\n        memo[k] = k * fact(k - 1)\n        return memo[k]\n    \n    return fact(n)\n```\n\nThis implementation includes memoization for better performance on repeated calls.",
				},
				"finish_reason": "stop",
			},
		}
		result["usage"] = map[string]interface{}{
			"prompt_tokens":     25,
			"completion_tokens": 80,
			"total_tokens":      105,
		}
		
		// Add tool calls if requested
		if tools, ok := request["tools"].([]interface{}); ok && len(tools) > 0 {
			if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := choice["message"].(map[string]interface{}); ok {
						message["tool_calls"] = []map[string]interface{}{
							{
								"id":   "call_test_gpt4",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "get_current_weather",
									"arguments": `{"location": "San Francisco, CA"}`,
								},
							},
						}
					}
				}
			}
		}
		
	case strings.Contains(model, "gpt-3.5"):
		result["choices"] = []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "I can help with coding tasks. Here's a Python factorial function:\n\n```python\ndef factorial(n):\n    result = 1\n    for i in range(1, n + 1):\n        result *= i\n    return result\n```\n\nThis uses an iterative approach which is straightforward and efficient.",
				},
				"finish_reason": "stop",
			},
		}
		result["usage"] = map[string]interface{}{
			"prompt_tokens":     20,
			"completion_tokens": 60,
			"total_tokens":      80,
		}
	}
	
	// Handle streaming if requested
	if stream, ok := request["stream"].(bool); ok && stream {
		// For streaming, we'll return a simple marker
		result["stream"] = true
	}
	
	return result
}

// Completions endpoint (legacy)
func (m *MockAPIServer) handleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	response := map[string]interface{}{
		"id":      "cmpl-test",
		"object":  "text_completion",
		"created": time.Now().Unix(),
		"model":   "text-davinci-003",
		"choices": []map[string]interface{}{
			{
				"text":  "This is a legacy completion response for testing purposes.",
				"index": 0,
				"logprobs": map[string]interface{}{
					"tokens": []string{"This", " is", " a", " legacy", " completion"},
					"token_logprobs": []float64{-0.1, -0.2, -0.3, -0.4, -0.5},
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     5,
			"completion_tokens": 10,
			"total_tokens":      15,
		},
	}
	
	m.writeJSONResponse(w, response)
}

// Embeddings endpoint
func (m *MockAPIServer) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	var request map[string]interface{}
	json.NewDecoder(r.Body).Decode(&request)
	
	model, _ := request["model"].(string)
	input := request["input"]
	
	var embeddingSize int
	switch model {
	case "text-embedding-3-large":
		embeddingSize = 3072
	case "text-embedding-3-small":
		embeddingSize = 1536
	default:
		embeddingSize = 1536
	}
	
	var data []map[string]interface{}
	
	switch v := input.(type) {
	case string:
		data = []map[string]interface{}{
			{
				"object":    "embedding",
				"embedding": generateEmbedding(embeddingSize),
				"index":     0,
			},
		}
	case []interface{}:
		data = make([]map[string]interface{}, len(v))
		for i, item := range v {
			data[i] = map[string]interface{}{
				"object":    "embedding",
				"embedding": generateEmbedding(embeddingSize),
				"index":     i,
			}
		}
	}
	
	response := map[string]interface{}{
		"object": "list",
		"data":   data,
		"model":  model,
		"usage": map[string]interface{}{
			"prompt_tokens": 10,
			"total_tokens":  10,
		},
	}
	
	m.writeJSONResponse(w, response)
}

// Moderations endpoint
func (m *MockAPIServer) handleModerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	response := map[string]interface{}{
		"id":     "modr-test",
		"model":  "text-moderation-latest",
		"results": []map[string]interface{}{
			{
				"flagged": false,
				"categories": map[string]bool{
					"sexual":                false,
					"violence":              false,
					"harassment":            false,
					"self-harm":             false,
					"sexual/minors":         false,
					"hate":                  false,
					"violence/graphic":      false,
					"self-harm/intent":      false,
					"self-harm/instructions": false,
					"harassment/threatening": false,
				},
				"category_scores": map[string]float64{
					"sexual":                0.01,
					"violence":              0.01,
					"harassment":            0.01,
					"self-harm":             0.01,
					"sexual/minors":         0.01,
					"hate":                  0.01,
					"violence/graphic":      0.01,
					"self-harm/intent":      0.01,
					"self-harm/instructions": 0.01,
					"harassment/threatening": 0.01,
				},
			},
		},
	}
	
	m.writeJSONResponse(w, response)
}

// Image generation endpoint
func (m *MockAPIServer) handleImageGenerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	response := map[string]interface{}{
		"created": time.Now().Unix(),
		"data": []map[string]interface{}{
			{
				"url":         "https://example.com/generated-image.png",
				"revised_prompt": "A revised version of the input prompt",
			},
		},
	}
	
	m.writeJSONResponse(w, response)
}

// Image edits endpoint
func (m *MockAPIServer) handleImageEdits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	response := map[string]interface{}{
		"created": time.Now().Unix(),
		"data": []map[string]interface{}{
			{
				"url": "https://example.com/edited-image.png",
			},
		},
	}
	
	m.writeJSONResponse(w, response)
}

// Image variations endpoint
func (m *MockAPIServer) handleImageVariations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	response := map[string]interface{}{
		"created": time.Now().Unix(),
		"data": []map[string]interface{}{
			{
				"url": "https://example.com/image-variation-1.png",
			},
			{
				"url": "https://example.com/image-variation-2.png",
			},
		},
	}
	
	m.writeJSONResponse(w, response)
}

// Audio transcription endpoint
func (m *MockAPIServer) handleAudioTranscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	response := map[string]interface{}{
		"text": "This is a transcribed version of the audio file for testing purposes.",
	}
	
	m.writeJSONResponse(w, response)
}

// Audio speech endpoint
func (m *MockAPIServer) handleAudioSpeech(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}
	
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	// Return a mock audio file (just for testing)
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Write([]byte("mock audio data"))
}

// Fine-tuning jobs endpoint
func (m *MockAPIServer) handleFineTuningJobs(w http.ResponseWriter, r *http.Request) {
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	switch r.Method {
	case http.MethodGet:
		response := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"id":      "ftjob-test-1",
					"object":  "fine_tuning.job",
					"model":   "gpt-3.5-turbo-0613",
					"created":  time.Now().Unix(),
					"status":  "succeeded",
				},
			},
		}
		m.writeJSONResponse(w, response)
	case http.MethodPost:
		response := map[string]interface{}{
			"id":      "ftjob-test-new",
			"object":  "fine_tuning.job",
			"model":   "gpt-3.5-turbo-0613",
			"created":  time.Now().Unix(),
			"status":  "running",
		}
		m.writeJSONResponse(w, response)
	default:
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
	}
}

// Assistants endpoint
func (m *MockAPIServer) handleAssistants(w http.ResponseWriter, r *http.Request) {
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	switch r.Method {
	case http.MethodGet:
		response := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"id":      "asst_test_123",
					"object":  "assistant",
					"created": time.Now().Unix(),
					"name":    "Test Assistant",
					"model":   "gpt-4-turbo-preview",
				},
			},
		}
		m.writeJSONResponse(w, response)
	case http.MethodPost:
		response := map[string]interface{}{
			"id":      "asst_new_456",
			"object":  "assistant",
			"created": time.Now().Unix(),
			"name":    "New Assistant",
			"model":   "gpt-4-turbo-preview",
		}
		m.writeJSONResponse(w, response)
	default:
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
	}
}

// Threads endpoint
func (m *MockAPIServer) handleThreads(w http.ResponseWriter, r *http.Request) {
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	response := map[string]interface{}{
		"id":      "thread_test_789",
		"object":  "thread",
		"created": time.Now().Unix(),
	}
	
	m.writeJSONResponse(w, response)
}

// Files endpoint
func (m *MockAPIServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	if !m.authenticateRequest(r) {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	if m.simulateErrors(w, r) {
		return
	}
	
	m.setRateLimitHeaders(w)
	
	response := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"id":        "file_test_abc",
				"object":    "file",
				"bytes":     1024,
				"created":   time.Now().Unix(),
				"filename":  "test.jsonl",
				"purpose":   "fine-tune",
			},
		},
	}
	
	m.writeJSONResponse(w, response)
}

// Test control endpoints
func (m *MockAPIServer) handleTestReset(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Reset server to default state
		m.models = generateDefaultModels()
		m.setupDefaultResponses()
		m.errorConfig = ErrorConfig{
			SimulateErrors:   false,
			ErrorRate:        0.0,
			TimeoutRate:      0.0,
			UnauthorizedRate: 0.0,
			ServerErrorRate:  0.0,
		}
		
		response := map[string]interface{}{
			"status": "reset",
			"time":   time.Now().Unix(),
		}
		m.writeJSONResponse(w, response)
	} else {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
	}
}

func (m *MockAPIServer) handleTestConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		response := map[string]interface{}{
			"rate_limit": m.rateLimitConfig,
			"error_config": m.errorConfig,
			"models_count": len(m.models),
		}
		m.writeJSONResponse(w, response)
	} else if r.Method == http.MethodPost {
		var config map[string]interface{}
		json.NewDecoder(r.Body).Decode(&config)
		
		if rateLimit, ok := config["rate_limit"].(map[string]interface{}); ok {
			if rpm, ok := rateLimit["requests_per_minute"].(float64); ok {
				m.rateLimitConfig.RequestsPerMinute = int(rpm)
			}
			if tpm, ok := rateLimit["tokens_per_minute"].(float64); ok {
				m.rateLimitConfig.TokensPerMinute = int(tpm)
			}
			if enable, ok := rateLimit["enable"].(bool); ok {
				m.rateLimitConfig.EnableRateLimit = enable
			}
		}
		
		if errorConf, ok := config["error_config"].(map[string]interface{}); ok {
			if simulate, ok := errorConf["simulate"].(bool); ok {
				m.errorConfig.SimulateErrors = simulate
			}
			if rate, ok := errorConf["error_rate"].(float64); ok {
				m.errorConfig.ErrorRate = rate
			}
		}
		
		response := map[string]interface{}{
			"status": "updated",
			"time":   time.Now().Unix(),
		}
		m.writeJSONResponse(w, response)
	} else {
		http.Error(w, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed)
	}
}

// Helper functions
func generateEmbedding(size int) []float64 {
	embedding := make([]float64, size)
	for i := 0; i < size; i++ {
		embedding[i] = float64(i%1000) * 0.001 - 0.5 // Simple deterministic embedding
	}
	return embedding
}

// Utility function for creating a mock server with custom configuration
func CreateMockServerWithConfig(rateLimit RateLimitConfig, errors ErrorConfig) *httptest.Server {
	server := NewMockAPIServer()
	server.SetRateLimitConfig(rateLimit)
	server.SetErrorConfig(errors)
	return server.Start()
}