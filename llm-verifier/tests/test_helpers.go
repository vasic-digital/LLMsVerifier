package tests

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"llm-verifier/config"
	"llm-verifier/database"
	"llm-verifier/llmverifier"
)

// TestHelper provides common test utilities
type TestHelper struct {
	DB       *database.Database
	MockServer *httptest.Server
	Config   *config.Config
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	// Create temp database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Create mock server
	mockServer := createMockServer(t)
	
	// Create test configuration
	testConfig := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:      mockServer.URL + "/v1",
			APIKey:       "test-api-key",
			MaxRetries:   3,
			RequestDelay: 100 * time.Millisecond,
			Timeout:      10 * time.Second,
		},
		LLMs:        []config.LLMConfig{},
		Concurrency: 2,
		Timeout:     15 * time.Second,
	}
	
	return &TestHelper{
		DB:         db,
		MockServer: mockServer,
		Config:     testConfig,
	}
}

// Cleanup cleans up test resources
func (th *TestHelper) Cleanup() {
	if th.DB != nil {
		th.DB.Close()
	}
	if th.MockServer != nil {
		th.MockServer.Close()
	}
}

// createMockServer creates a mock OpenAI API server
func createMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			handleModelsRequest(w, r)
		case "/v1/chat/completions":
			handleChatCompletionsRequest(w, r)
		case "/v1/embeddings":
			handleEmbeddingsRequest(w, r)
		default:
			handleDefaultRequest(w, r)
		}
	}))
}

// handleModelsRequest handles the models endpoint
func handleModelsRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Check API key
	auth := r.Header.Get("Authorization")
	if auth != "Bearer test-api-key" {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	response := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"id":       "gpt-4-turbo",
				"object":   "model",
				"created":  1677649963,
				"owned_by": "openai",
			},
			{
				"id":       "gpt-3.5-turbo",
				"object":   "model",
				"created":  1677649963,
				"owned_by": "openai",
			},
			{
				"id":       "text-embedding-3-small",
				"object":   "model",
				"created":  1677649963,
				"owned_by": "openai",
			},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleChatCompletionsRequest handles chat completions
func handleChatCompletionsRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Check API key
	auth := r.Header.Get("Authorization")
	if auth != "Bearer test-api-key" {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	// Add rate limit headers
	w.Header().Set("x-ratelimit-limit-requests", "100")
	w.Header().Set("x-ratelimit-limit-tokens", "10000")
	w.Header().Set("x-ratelimit-remaining-requests", "95")
	w.Header().Set("x-ratelimit-remaining-tokens", "9500")
	w.Header().Set("x-ratelimit-reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
	
	// Simulate different responses based on model
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Invalid request body"}}`, http.StatusBadRequest)
		return
	}
	
	model := request["model"].(string)
	
	// Simulate different behaviors
	switch model {
	case "gpt-4-turbo":
		handleGPT4Response(w, r, request)
	case "gpt-3.5-turbo":
		handleGPT35Response(w, r, request)
	default:
		handleDefaultModelResponse(w, r, request)
	}
}

// handleGPT4Response simulates GPT-4 responses
func handleGPT4Response(w http.ResponseWriter, r *http.Request, request map[string]interface{}) {
	// Simulate high capability responses
	response := map[string]interface{}{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "gpt-4-turbo",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "I can help you with coding tasks. Here's a Python function to calculate factorial:\n\n```python\ndef factorial(n):\n    if n == 0 or n == 1:\n        return 1\n    return n * factorial(n - 1)\n```\n\nThis function uses recursion to calculate the factorial of a number.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     20,
			"completion_tokens": 50,
			"total_tokens":      70,
		},
	}
	
	// Simulate tool use if requested
	if tools, ok := request["tools"].([]interface{}); ok && len(tools) > 0 {
		response["choices"].([]map[string]interface{})[0]["message"]["tool_calls"] = []map[string]interface{}{
			{
				"id":       "call_test",
				"type":     "function",
				"function": map[string]interface{}{
					"name":      "get_current_weather",
					"arguments": `{"location": "New York, NY"}`,
				},
			},
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGPT35Response simulates GPT-3.5 responses
func handleGPT35Response(w http.ResponseWriter, r *http.Request, request map[string]interface{}) {
	// Simulate moderate capability responses
	response := map[string]interface{}{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "gpt-3.5-turbo",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "I can help with coding tasks. For calculating factorial:\n\n```python\ndef factorial(n):\n    result = 1\n    for i in range(1, n + 1):\n        result *= i\n    return result\n```\n\nThis uses an iterative approach.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     20,
			"completion_tokens": 40,
			"total_tokens":      60,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleEmbeddingsRequest handles embeddings requests
func handleEmbeddingsRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Check API key
	auth := r.Header.Get("Authorization")
	if auth != "Bearer test-api-key" {
		http.Error(w, `{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}
	
	response := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"object":       "embedding",
				"embedding":    generateRandomEmbedding(1536), // Standard embedding size
				"index":        0,
			},
		},
		"model":   "text-embedding-3-small",
		"usage": map[string]interface{}{
			"prompt_tokens": 10,
			"total_tokens":  10,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDefaultRequest handles other requests
func handleDefaultRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":{"code":"not_found","message":"Not found"}}`, http.StatusNotFound)
}

// handleDefaultModelResponse handles responses for unknown models
func handleDefaultModelResponse(w http.ResponseWriter, r *http.Request, request map[string]interface{}) {
	response := map[string]interface{}{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   request["model"],
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "I can help with general questions and tasks.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": 10,
			"total_tokens":      20,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateRandomEmbedding generates a random embedding vector
func generateRandomEmbedding(size int) []float32 {
	embedding := make([]float32, size)
	for i := range embedding {
		embedding[i] = float32(i) * 0.01 // Simple deterministic embedding for testing
	}
	return embedding
}

// CreateTestConfig creates a test configuration
func CreateTestConfig() *config.Config {
	return &config.Config{
		Global: config.GlobalConfig{
			BaseURL:      "https://api.openai.com/v1",
			APIKey:       "test-api-key",
			MaxRetries:   3,
			RequestDelay: 100 * time.Millisecond,
			Timeout:      10 * time.Second,
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "Test GPT-4",
				Endpoint: "https://api.openai.com/v1",
				APIKey:   "test-api-key",
				Model:    "gpt-4-turbo",
			},
			{
				Name:     "Test GPT-3.5",
				Endpoint: "https://api.openai.com/v1",
				APIKey:   "test-api-key",
				Model:    "gpt-3.5-turbo",
			},
		},
		Concurrency: 2,
		Timeout:     15 * time.Second,
	}
}

// CreateTestVerifier creates a test verifier with mocked dependencies
func CreateTestVerifier(config *config.Config) *llmverifier.Verifier {
	return llmverifier.New(config)
}

// SetupTestEnvironment sets up a complete test environment
func SetupTestEnvironment(t *testing.T) (*TestHelper, func()) {
	helper := NewTestHelper(t)
	
	cleanup := func() {
		helper.Cleanup()
	}
	
	return helper, cleanup
}

// AssertNoError asserts that an error is nil
func AssertNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// AssertError asserts that an error is not nil
func AssertError(t *testing.T, err error) {
	if err == nil {
		t.Error("Expected an error, got nil")
	}
}

// AssertEquals asserts that two values are equal
func AssertEquals(t *testing.T, expected, actual interface{}) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertTrue asserts that a condition is true
func AssertTrue(t *testing.T, condition bool, message string) {
	if !condition {
		t.Errorf("Expected true, got false: %s", message)
	}
}

// AssertFalse asserts that a condition is false
func AssertFalse(t *testing.T, condition bool, message string) {
	if condition {
		t.Errorf("Expected false, got true: %s", message)
	}
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// GenerateTestModels generates test models
func GenerateTestModels(count int) []map[string]interface{} {
	models := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		models[i] = map[string]interface{}{
			"id":       fmt.Sprintf("test-model-%d", i),
			"object":   "model",
			"created":  1677649963 + int64(i),
			"owned_by": "test-provider",
		}
	}
	return models
}

// GenerateTestVerificationResults generates test verification results
func GenerateTestVerificationResults(count int) []*llmverifier.VerificationResult {
	results := make([]*llmverifier.VerificationResult, count)
	for i := 0; i < count; i++ {
		now := time.Now()
		results[i] = &llmverifier.VerificationResult{
			ModelInfo: llmverifier.ModelInfo{
				ID:       fmt.Sprintf("test-model-%d", i),
				Object:   "model",
				Created:  now.Unix(),
				Endpoint: "https://api.test.com/v1",
			},
			Availability: llmverifier.AvailabilityResult{
				Exists:      true,
				Responsive:  true,
				Overloaded:  false,
				Latency:     time.Duration(100+i*10) * time.Millisecond,
				LastChecked: now,
			},
			Timestamp: now,
			OverallScore: 80.0 + float64(i%20),
		}
	}
	return results
}