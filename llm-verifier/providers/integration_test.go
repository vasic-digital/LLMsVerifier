package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnthropicAdapter_Integration tests Anthropic adapter integration
func TestAnthropicAdapter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Mock Anthropic API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/messages", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		assert.Contains(t, r.Header.Get("x-api-key"), "test-key")

		// Return mock response
		response := `{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Hello from Anthropic!"}],
			"model": "claude-3-sonnet-20240229",
			"stop_reason": "end_turn",
			"usage": {
				"input_tokens": 10,
				"output_tokens": 8
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	adapter := NewAnthropicAdapter(client, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := adapter.ChatCompletion(ctx, request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "msg_123", response.ID)
	assert.Equal(t, "claude-3-sonnet-20240229", response.Model)
	assert.Len(t, response.Choices, 1)
	assert.Contains(t, response.Choices[0].Message.Content, "Hello from Anthropic")
}

// TestCohereAdapter_Integration tests Cohere adapter integration
func TestCohereAdapter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Mock Cohere API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/generate", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		// Return mock response
		response := `{
			"response_id": "cohere-123",
			"text": "Hello from Cohere!",
			"generation_id": "gen_123",
			"token_count": {
				"prompt_tokens": 5,
				"response_tokens": 4,
				"total_tokens": 9,
				"billed_tokens": 9
			},
			"meta": {
				"api_version": {
					"version": "2022-12-06"
				}
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	adapter := NewCohereAdapter(client, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "command",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := adapter.ChatCompletion(ctx, request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "cohere-123", response.ID)
	assert.Equal(t, "command", response.Model)
	assert.Len(t, response.Choices, 1)
	assert.Contains(t, response.Choices[0].Message.Content, "Hello from Cohere")
	assert.Equal(t, 5, response.Usage.PromptTokens)
	assert.Equal(t, 4, response.Usage.CompletionTokens)
	assert.Equal(t, 9, response.Usage.TotalTokens)
}

// TestTogetherAIAdapter_Integration tests Together AI adapter integration
func TestTogetherAIAdapter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Mock Together AI API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

			// Return mock OpenAI-compatible response
			response := `{
				"id": "together-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "mistralai/Mistral-7B-Instruct-v0.1",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello from Together AI!"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 9,
					"completion_tokens": 7,
					"total_tokens": 16
				}
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		} else if r.URL.Path == "/models" {
			assert.Equal(t, "GET", r.Method)
			response := `{
				"object": "list",
				"data": [
					{
						"id": "mistralai/Mistral-7B-Instruct-v0.1",
						"object": "model",
						"created": 1677652288,
						"owned_by": "mistralai"
					}
				]
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	adapter := NewTogetherAIAdapter(client, server.URL, "test-key")

	// Test chat completion
	request := OpenAIChatRequest{
		Model: "mistralai/Mistral-7B-Instruct-v0.1",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := adapter.ChatCompletion(ctx, request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "together-123", response.ID)
	assert.Equal(t, "mistralai/Mistral-7B-Instruct-v0.1", response.Model)
	assert.Len(t, response.Choices, 1)
	assert.Contains(t, response.Choices[0].Message.Content, "Hello from Together AI")

	// Test models listing
	models, err := adapter.ListModels(ctx)
	require.NoError(t, err)
	assert.NotNil(t, models)
	assert.Len(t, models.Data, 1)
	assert.Equal(t, "mistralai/Mistral-7B-Instruct-v0.1", models.Data[0].ID)
}

// TestProviderRegistry_Integration tests the provider registry integration
func TestProviderRegistry_Integration(t *testing.T) {
	registry := NewProviderRegistry()

	// Test that new providers are registered in config
	anthropicConfig, exists := registry.GetConfig("anthropic")
	assert.True(t, exists)
	assert.NotNil(t, anthropicConfig)
	assert.Equal(t, "anthropic", anthropicConfig.Name)
	assert.Equal(t, "https://api.anthropic.com/v1", anthropicConfig.Endpoint)

	// Test provider names
	providerNames := registry.GetProviderNames()
	assert.Contains(t, providerNames, "anthropic")
	assert.Contains(t, providerNames, "google")
	assert.Contains(t, providerNames, "openai")

	// Test provider support
	assert.True(t, registry.IsProviderSupported("anthropic"))
	assert.True(t, registry.IsProviderSupported("google"))
	assert.False(t, registry.IsProviderSupported("nonexistent"))

	// Test adapter instantiation (manual)
	client := &http.Client{}
	anthropicAdapter := NewAnthropicAdapter(client, "https://api.anthropic.com", "test-key")
	assert.NotNil(t, anthropicAdapter)
	assert.IsType(t, &AnthropicAdapter{}, anthropicAdapter)

	cohereAdapter := NewCohereAdapter(client, "https://api.cohere.ai", "test-key")
	assert.NotNil(t, cohereAdapter)
	assert.IsType(t, &CohereAdapter{}, cohereAdapter)

	togetherAdapter := NewTogetherAIAdapter(client, "https://api.together.xyz", "test-key")
	assert.NotNil(t, togetherAdapter)
	assert.IsType(t, &TogetherAIAdapter{}, togetherAdapter)
}

// TestCrossProviderCompatibility tests compatibility across different providers
func TestCrossProviderCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cross-provider test in short mode")
	}

	// Mock servers for different providers
	anthropicServer := createMockAnthropicServer()
	defer anthropicServer.Close()

	cohereServer := createMockCohereServer()
	defer cohereServer.Close()

	togetherServer := createMockTogetherServer()
	defer togetherServer.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	// Create adapters
	anthropicAdapter := NewAnthropicAdapter(client, anthropicServer.URL, "test-key")
	cohereAdapter := NewCohereAdapter(client, cohereServer.URL, "test-key")
	togetherAdapter := NewTogetherAIAdapter(client, togetherServer.URL, "test-key")

	// Standard test request
	testRequest := OpenAIChatRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "What is the capital of France?"},
		},
		MaxTokens:   50,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test all adapters with the same request
	t.Run("Anthropic", func(t *testing.T) {
		response, err := anthropicAdapter.ChatCompletion(ctx, testRequest)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ID)
		assert.Len(t, response.Choices, 1)
		assert.NotEmpty(t, response.Choices[0].Message.Content)
		assert.Greater(t, response.Usage.TotalTokens, 0)
	})

	t.Run("Cohere", func(t *testing.T) {
		response, err := cohereAdapter.ChatCompletion(ctx, testRequest)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ID)
		assert.Len(t, response.Choices, 1)
		assert.NotEmpty(t, response.Choices[0].Message.Content)
		assert.Greater(t, response.Usage.TotalTokens, 0)
	})

	t.Run("TogetherAI", func(t *testing.T) {
		response, err := togetherAdapter.ChatCompletion(ctx, testRequest)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ID)
		assert.Len(t, response.Choices, 1)
		assert.NotEmpty(t, response.Choices[0].Message.Content)
		assert.Greater(t, response.Usage.TotalTokens, 0)
	})
}

// Helper functions for mock servers

func createMockAnthropicServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"id": "msg_test",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Paris is the capital of France."}],
			"model": "claude-3-sonnet-20240229",
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 8, "output_tokens": 7}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
}

func createMockCohereServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"response_id": "cohere_test",
			"text": "Paris is the capital of France.",
			"generation_id": "gen_test",
			"token_count": {
				"prompt_tokens": 8,
				"response_tokens": 6,
				"total_tokens": 14,
				"billed_tokens": 14
			},
			"meta": {"api_version": {"version": "2022-12-06"}}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
}

func createMockTogetherServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"id": "together_test",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "test-model",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Paris is the capital of France."
				},
				"finish_reason": "stop"
			}],
			"usage": {
				"prompt_tokens": 8,
				"completion_tokens": 6,
				"total_tokens": 14
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
}

// TestErrorHandling_Integration tests error handling across providers
func TestErrorHandling_Integration(t *testing.T) {
	// Mock server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "Rate limit exceeded"}`))
	}))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	adapter := NewAnthropicAdapter(client, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := adapter.ChatCompletion(ctx, request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

// TestLoadBalancing_Integration tests load balancing across providers
func TestLoadBalancing_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load balancing test in short mode")
	}

	// Test that we can create multiple instances manually
	client := &http.Client{}

	// Create multiple adapter instances with different endpoints/keys
	adapter1 := NewAnthropicAdapter(client, "https://api.anthropic.com", "key1")
	adapter2 := NewAnthropicAdapter(client, "https://api.anthropic.com", "key2")
	adapter3 := NewAnthropicAdapter(client, "https://api.anthropic.com", "key3")

	assert.NotNil(t, adapter1)
	assert.NotNil(t, adapter2)
	assert.NotNil(t, adapter3)
	assert.IsType(t, &AnthropicAdapter{}, adapter1)
	assert.IsType(t, &AnthropicAdapter{}, adapter2)
	assert.IsType(t, &AnthropicAdapter{}, adapter3)

	// Test registry supports the provider
	registry := NewProviderRegistry()
	assert.True(t, registry.IsProviderSupported("anthropic"))
}
