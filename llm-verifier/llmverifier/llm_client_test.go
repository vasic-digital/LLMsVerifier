package llmverifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewLLMClient(t *testing.T) {
	client := NewLLMClient("https://api.example.com/v1", "test-key", nil)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.endpoint != "https://api.example.com/v1" {
		t.Errorf("Expected endpoint 'https://api.example.com/v1', got '%s'", client.endpoint)
	}

	if client.apiKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", client.apiKey)
	}

	if client.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", client.httpClient.Timeout)
	}

	client2 := NewLLMClient("https://api.example.com/v1/", "test-key", nil)
	if client2.endpoint != "https://api.example.com/v1" {
		t.Errorf("Expected trailing slash to be trimmed, got '%s'", client2.endpoint)
	}

	headers := map[string]string{
		"Custom-Header": "value",
		"X-API-Version": "2024-01-01",
	}
	client3 := NewLLMClient("https://api.example.com/v1", "test-key", headers)

	if client3.headers == nil {
		t.Error("Expected headers to be set")
	}

	if client3.headers["Custom-Header"] != "value" {
		t.Errorf("Expected custom header 'value', got '%s'", client3.headers["Custom-Header"])
	}
}

func TestLLMClient_ListModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/models" {
			t.Errorf("Expected path /models, got %s", r.URL.Path)
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Error("Expected Bearer token in Authorization header")
		}

		response := `{
			"object": "list",
			"data": [
				{
					"id": "gpt-4",
					"object": "model",
					"created": 1687882411,
					"owned_by": "openai"
				},
				{
					"id": "gpt-3.5-turbo",
					"object": "model",
					"created": 1677610602,
					"owned_by": "openai"
				}
			]
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", nil)

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0].ID != "gpt-4" {
		t.Errorf("Expected first model ID 'gpt-4', got '%s'", models[0].ID)
	}

	if models[1].ID != "gpt-3.5-turbo" {
		t.Errorf("Expected second model ID 'gpt-3.5-turbo', got '%s'", models[1].ID)
	}
}

func TestLLMClient_ListModels_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customHeader := r.Header.Get("Custom-Header")
		if customHeader != "value" {
			t.Errorf("Expected Custom-Header 'value', got '%s'", customHeader)
		}

		response := `{"object": "list", "data": []}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	headers := map[string]string{
		"Custom-Header": "value",
	}
	client := NewLLMClient(server.URL, "test-key", headers)

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) != 0 {
		t.Errorf("Expected 0 models, got %d", len(models))
	}
}

func TestLLMClient_ListModels_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", nil)

	models, err := client.ListModels(context.Background())
	if err == nil {
		t.Error("Expected error for server error response")
	}

	if models != nil {
		t.Error("Expected nil models on error")
	}
}

func TestLLMClient_ListModels_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", nil)

	models, err := client.ListModels(context.Background())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if models != nil {
		t.Error("Expected nil models on JSON parse error")
	}
}

func TestLLMClient_CheckModelExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			response := `{
				"object": "list",
				"data": [
					{"id": "gpt-4", "object": "model"},
					{"id": "gpt-3.5-turbo", "object": "model"}
				]
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		}
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", nil)

	exists, err := client.CheckModelExists(context.Background(), "gpt-4")
	if err != nil {
		t.Fatalf("CheckModelExists failed: %v", err)
	}

	if !exists {
		t.Error("Expected model 'gpt-4' to exist")
	}

	exists, err = client.CheckModelExists(context.Background(), "non-existent-model")
	if err != nil {
		t.Fatalf("CheckModelExists failed: %v", err)
	}

	if exists {
		t.Error("Expected model 'non-existent-model' to not exist")
	}
}

func TestLLMClient_ChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		var req ChatCompletionRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("Failed to unmarshal request: %v", err)
		}

		if req.Model != "gpt-4" {
			t.Errorf("Expected model 'gpt-4', got '%s'", req.Model)
		}

		response := `{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-4",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! How can I help you today?"
					},
					"finish_reason": "stop"
				}
			],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 8,
				"total_tokens": 18
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", nil)

	req := ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []Message{
			{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		},
	}

	resp, err := client.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response to be non-nil")
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("Expected response ID 'chatcmpl-123', got '%s'", resp.ID)
	}

	if resp.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", resp.Model)
	}
}

func TestLLMClient_ChatCompletion_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "Invalid request"}}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", nil)

	req := ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
	}

	resp, err := client.ChatCompletion(context.Background(), req)
	if err == nil {
		t.Error("Expected error for bad request")
	}

	if resp != nil {
		t.Error("Expected nil response on error")
	}
}
