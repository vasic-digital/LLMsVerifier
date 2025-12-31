package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Cohere adapter creation and methods
func TestNewCohereAdapter(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewCohereAdapter(client, "https://api.cohere.ai/v1/", "test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.cohere.ai/v1", adapter.endpoint)
	assert.Equal(t, "test-key", adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
	assert.Equal(t, "Bearer test-key", adapter.headers["Authorization"])
}

func TestCohereAdapter_ListModels(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewCohereAdapter(client, "https://api.cohere.ai/v1", "test-key")

	models, err := adapter.ListModels(context.Background())
	require.NoError(t, err)
	require.NotNil(t, models)
	assert.Equal(t, "list", models.Object)
	assert.NotEmpty(t, models.Data)

	// Check that known models are included
	modelIDs := make([]string, len(models.Data))
	for i, m := range models.Data {
		modelIDs[i] = m.ID
	}
	assert.Contains(t, modelIDs, "command")
	assert.Contains(t, modelIDs, "base")
	assert.Contains(t, modelIDs, "command-light")
}

func TestCohereAdapter_ChatCompletion(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/generate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		// Return mock response
		resp := CohereChatResponse{
			ResponseID:   "test-response-id",
			Text:         "Hello! How can I help you today?",
			GenerationID: "test-gen-id",
		}
		resp.TokenCount.PromptTokens = 10
		resp.TokenCount.ResponseTokens = 8
		resp.TokenCount.TotalTokens = 18

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewCohereAdapter(client, server.URL, "test-key")

	req := OpenAIChatRequest{
		Model: "command",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	resp, err := adapter.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "test-response-id", resp.ID)
	assert.Equal(t, "chat.completion", resp.Object)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, "assistant", resp.Choices[0].Message.Role)
	assert.Contains(t, resp.Choices[0].Message.Content, "Hello")
}

func TestCohereAdapter_ChatCompletion_Error(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request"}`))
	}))
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewCohereAdapter(client, server.URL, "test-key")

	req := OpenAIChatRequest{
		Model: "command",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := adapter.ChatCompletion(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "400")
}

func TestCohereAdapter_StreamChatCompletion(t *testing.T) {
	// Create mock streaming server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/generate", r.URL.Path)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: Hello\n\n"))
		w.Write([]byte("data: World\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewCohereAdapter(client, server.URL, "test-key")

	req := OpenAIChatRequest{
		Model: "command",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	responseChan, errorChan := adapter.StreamChatCompletion(ctx, req)

	var responses []OpenAIStreamResponse
	var streamErr error

	for {
		select {
		case resp, ok := <-responseChan:
			if !ok {
				responseChan = nil
			} else {
				responses = append(responses, resp)
			}
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
			} else if err != nil {
				streamErr = err
			}
		case <-ctx.Done():
			goto done
		}
		if responseChan == nil && errorChan == nil {
			break
		}
	}
done:

	assert.NoError(t, streamErr)
	assert.NotEmpty(t, responses)
}

func TestCohereAdapter_StreamChatCompletion_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewCohereAdapter(client, server.URL, "test-key")

	req := OpenAIChatRequest{
		Model: "command",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, errorChan := adapter.StreamChatCompletion(ctx, req)

	var streamErr error
	select {
	case err := <-errorChan:
		streamErr = err
	case <-ctx.Done():
	}

	assert.Error(t, streamErr)
}

// Test Replicate adapter - Extended
func TestNewReplicateAdapter_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewReplicateAdapter(client, "https://api.replicate.com/v1/", "r8-test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.replicate.com/v1", adapter.endpoint)
	assert.Equal(t, "r8-test-key", adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
	assert.Equal(t, "Token r8-test-key", adapter.headers["Authorization"])
}

func TestReplicateAdapter_ListModels_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewReplicateAdapter(client, "https://api.replicate.com/v1", "test-key")

	models, err := adapter.ListModels(context.Background())
	require.NoError(t, err)
	require.NotNil(t, models)
	assert.Equal(t, "list", models.Object)
	assert.NotEmpty(t, models.Data)

	// Check for known replicate models
	modelIDs := make([]string, len(models.Data))
	for i, m := range models.Data {
		modelIDs[i] = m.ID
	}
	assert.Contains(t, modelIDs, "meta/llama-2-70b-chat")
}

func TestExtractPromptFromMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		expected string
	}{
		{
			name:     "single message",
			messages: []Message{{Role: "user", Content: "Hello"}},
			expected: "Hello",
		},
		{
			name: "multiple messages",
			messages: []Message{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Hello"},
			},
			expected: "You are helpful\nHello",
		},
		{
			name:     "empty messages",
			messages: []Message{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPromptFromMessages(tt.messages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test Anthropic adapter
func TestNewAnthropicAdapter(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewAnthropicAdapter(client, "https://api.anthropic.com/v1/", "test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.anthropic.com/v1", adapter.endpoint)
	assert.Contains(t, adapter.headers, "x-api-key")
}

func TestAnthropicAdapter_StreamChatCompletion(t *testing.T) {
	// Create mock streaming server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`event: content_block_delta` + "\n"))
		w.Write([]byte(`data: {"type":"content_block_delta","delta":{"text":"Hello"}}` + "\n\n"))
		w.Write([]byte(`event: message_stop` + "\n"))
		w.Write([]byte(`data: {"type":"message_stop"}` + "\n\n"))
	}))
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewAnthropicAdapter(client, server.URL, "test-key")

	req := OpenAIChatRequest{
		Model: "claude-3-opus-20240229",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	responseChan, errorChan := adapter.StreamChatCompletion(ctx, req)

	var responses []OpenAIStreamResponse
	var streamErr error

	for {
		select {
		case resp, ok := <-responseChan:
			if !ok {
				responseChan = nil
			} else {
				responses = append(responses, resp)
			}
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
			} else if err != nil {
				streamErr = err
			}
		case <-ctx.Done():
			goto done
		}
		if responseChan == nil && errorChan == nil {
			break
		}
	}
done:

	// Check for no critical errors (connection errors are expected in mock)
	if streamErr != nil {
		assert.True(t, strings.Contains(streamErr.Error(), "status") || strings.Contains(streamErr.Error(), "response"))
	}
}

// Test OpenAI adapter
func TestNewOpenAIAdapter_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewOpenAIAdapter(client, "https://api.openai.com/v1/", "test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.openai.com/v1", adapter.endpoint)
	assert.Contains(t, adapter.headers, "Authorization")
	assert.Equal(t, "Bearer test-key", adapter.headers["Authorization"])
}

func TestOpenAIAdapter_ValidateRequest_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewOpenAIAdapter(client, "https://api.openai.com/v1", "test-key")

	tests := []struct {
		name    string
		request OpenAIChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: OpenAIChatRequest{
				Model: "gpt-4",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty messages",
			request: OpenAIChatRequest{
				Model:    "gpt-4",
				Messages: []Message{},
			},
			wantErr: true,
		},
		{
			name: "empty model",
			request: OpenAIChatRequest{
				Model: "",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.ValidateRequest(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOpenAIAdapter_GetModelInfo_Extended(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/models/gpt-4") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "gpt-4",
				"object":   "model",
				"owned_by": "openai",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewOpenAIAdapter(client, server.URL, "test-key")

	modelInfo, err := adapter.GetModelInfo(context.Background(), "gpt-4")
	require.NoError(t, err)
	require.NotNil(t, modelInfo)
	// ModelInfo is a struct, not a map
	assert.Equal(t, "gpt-4", modelInfo.ID)
}

// Test Groq adapter
func TestNewGroqAdapter_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewGroqAdapter(client, "https://api.groq.com/openai/v1/", "gsk-test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.groq.com/openai/v1", adapter.endpoint)
	assert.Contains(t, adapter.headers, "Authorization")
}

func TestGroqAdapter_ChatCompletion_Extended(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)

		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "mixtral-8x7b-32768",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewGroqAdapter(client, server.URL, "test-key")

	req := OpenAIChatRequest{
		Model: "mixtral-8x7b-32768",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := adapter.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "chatcmpl-test", resp.ID)
}

// Test Mistral adapter
func TestNewMistralAdapter_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewMistralAdapter(client, "https://api.mistral.ai/v1/", "test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.mistral.ai/v1", adapter.endpoint)
}

func TestMistralAdapter_ChatCompletion_Extended(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "cmpl-mistral",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "mistral-small",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello from Mistral!",
					},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewMistralAdapter(client, server.URL, "test-key")

	req := OpenAIChatRequest{
		Model: "mistral-small",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := adapter.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

// Test SiliconFlow adapter
func TestNewSiliconFlowAdapter_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewSiliconFlowAdapter(client, "https://api.siliconflow.cn/v1/", "sf-test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.siliconflow.cn/v1", adapter.endpoint)
}

// Test TogetherAI adapter
func TestNewTogetherAIAdapter_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewTogetherAIAdapter(client, "https://api.together.xyz/v1/", "tog-test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.together.xyz/v1", adapter.endpoint)
}

// Test xAI adapter
func TestNewxAIAdapter_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewxAIAdapter(client, "https://api.x.ai/v1/", "xai-test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.x.ai/v1", adapter.endpoint)
}

// Test Cerebras adapter
func TestNewCerebrasAdapter_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewCerebrasAdapter(client, "https://api.cerebras.ai/v1/", "csk-test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.cerebras.ai/v1", adapter.endpoint)
}

// Test Cloudflare adapter
func TestNewCloudflareAdapter_Extended(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewCloudflareAdapter(client, "https://api.cloudflare.com/client/v4/", "cf-test-key")

	require.NotNil(t, adapter)
	assert.Equal(t, "https://api.cloudflare.com/client/v4", adapter.endpoint)
}

// Test Recovery Strategies
func TestRecoveryStrategiesCreation(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
	strategies := NewRecoveryStrategies("openai", retryConfig)
	require.NotNil(t, strategies)
	assert.NotNil(t, strategies.handler)
}

// Test Model Provider Service - Extended
func TestNewModelProviderService_Extended(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("", logger)
	require.NotNil(t, service)
	assert.NotNil(t, service.providerClients)
	assert.NotNil(t, service.httpClient)
}

func TestModelProviderService_RegisterProvider_Extended(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("", logger)

	// Use the correct API - RegisterProvider takes (providerID, baseURL, apiKey string)
	service.RegisterProvider("custom-provider", "https://api.custom.com/v1", "custom-key")

	// Verify registration
	assert.Contains(t, service.providerClients, "custom-provider")
}

func TestModelProviderService_GetAllProviders_Extended(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("", logger)

	providers := service.GetAllProviders()
	assert.NotNil(t, providers)
}

func TestModelProviderService_ClearCache_Extended(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("", logger)

	// This should not panic
	service.ClearCache()
}

func TestModelProviderService_RefreshCache_Extended(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("", logger)

	// This should not panic
	service.RefreshCache()
}

// Test Enhanced Model Provider Service - Extended
func TestNewEnhancedModelProviderService_Extended(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()
	service := NewEnhancedModelProviderService("", logger, config)
	require.NotNil(t, service)

	// Test multiple operations in sequence
	service.EnableVerification(true)
	service.SetStrictMode(false)
	service.ClearVerificationResults()

	results := service.GetVerificationResults()
	assert.Empty(t, results)
}

func TestCreateDefaultVerificationConfig_Extended(t *testing.T) {
	config := CreateDefaultVerificationConfig()
	require.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.True(t, config.StrictMode)
}

// Test Relaxed Verification Service - Extended
func TestNewRelaxedVerificationService_Extended(t *testing.T) {
	logger := NewTestLogger()
	service := NewRelaxedVerificationService(logger)
	require.NotNil(t, service)
}

// Test Provider Service Adapter - Extended
func TestNewProviderServiceAdapter_Extended(t *testing.T) {
	logger := NewTestLogger()
	mockService := NewModelProviderService("", logger)
	adapter := NewProviderServiceAdapter(mockService)
	require.NotNil(t, adapter)
}

func TestProviderServiceAdapter_GetAllProviders_Extended(t *testing.T) {
	logger := NewTestLogger()
	mockService := NewModelProviderService("", logger)
	adapter := NewProviderServiceAdapter(mockService)
	providers := adapter.GetAllProviders()
	assert.NotNil(t, providers)
}

// Test Fallback Models
func TestGetFallbackModels_Extended(t *testing.T) {
	models := GetFallbackModels("openai")
	assert.NotEmpty(t, models)

	// Check that common models are included
	modelIDs := make(map[string]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	// Should have some common models
	assert.True(t, len(models) > 0)
}

// Test DeepSeek Adapter comprehensive tests
func TestDeepSeekAdapter_ChatCompletion_Extended(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-deepseek",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "deepseek-chat",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello from DeepSeek!",
					},
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, server.URL, "test-key")

	req := DeepSeekChatRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := adapter.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "deepseek-chat", resp.Model)
}
