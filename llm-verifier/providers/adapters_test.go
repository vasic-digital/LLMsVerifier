// Package providers tests for LLM provider adapters
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMockProviderServer creates a mock HTTP server for provider tests
func createMockProviderServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			if r.Method == "POST" {
				response := map[string]interface{}{
					"id":      "chatcmpl-123",
					"object":  "chat.completion",
					"created": time.Now().Unix(),
					"model":   "test-model",
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"message": map[string]string{
								"role":    "assistant",
								"content": "Hello! How can I help you?",
							},
						},
					},
					"usage": map[string]int{
						"prompt_tokens":     10,
						"completion_tokens": 20,
						"total_tokens":      30,
					},
				}
				json.NewEncoder(w).Encode(response)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case strings.HasSuffix(r.URL.Path, "/models"):
			if r.Method == "GET" {
				response := map[string]interface{}{
					"object": "list",
					"data": []map[string]interface{}{
						{
							"id":       "model-1",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "test-provider",
						},
						{
							"id":       "model-2",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "test-provider",
						},
					},
				}
				json.NewEncoder(w).Encode(response)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// createStreamingMockServer creates a mock server that returns SSE stream
func createStreamingMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Write streaming response
			streamResp1 := `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"test-model","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`
			streamResp2 := `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"test-model","choices":[{"index":0,"delta":{"content":" World"},"finish_reason":null}]}`
			streamResp3 := `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"test-model","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`

			fmt.Fprintf(w, "data: %s\n\n", streamResp1)
			fmt.Fprintf(w, "data: %s\n\n", streamResp2)
			fmt.Fprintf(w, "data: %s\n\n", streamResp3)
			fmt.Fprintf(w, "data: [DONE]\n\n")
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// createErrorMockServer creates a mock server that returns errors
func createErrorMockServer(statusCode int, errorMessage string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(errorMessage))
	}))
}

// ==================== Cerebras Adapter Tests ====================

func TestNewCerebrasAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.cerebras.ai/v1"
	apiKey := "test-key"

	adapter := NewCerebrasAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
	assert.Equal(t, "Bearer test-key", adapter.headers["Authorization"])
}

func TestCerebrasAdapter_EndpointTrailingSlash(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.cerebras.ai/v1/"
	apiKey := "test-key"

	adapter := NewCerebrasAdapter(client, endpoint, apiKey)

	// Should strip trailing slash
	assert.Equal(t, "https://api.cerebras.ai/v1", adapter.endpoint)
}

func TestCerebrasAdapter_ChatCompletion(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewCerebrasAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "llama3.1-8b",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "chatcmpl-123", response.ID)
	assert.Len(t, response.Choices, 1)
	assert.Equal(t, "assistant", response.Choices[0].Message.Role)
}

func TestCerebrasAdapter_ChatCompletion_Error(t *testing.T) {
	server := createErrorMockServer(http.StatusUnauthorized, "Invalid API key")
	defer server.Close()

	adapter := NewCerebrasAdapter(&http.Client{}, server.URL, "invalid-key")

	request := OpenAIChatRequest{
		Model:    "llama3.1-8b",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "401")
}

func TestCerebrasAdapter_ListModels(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewCerebrasAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "list", response.Object)
	assert.Len(t, response.Data, 2)
}

func TestCerebrasAdapter_ListModels_Error(t *testing.T) {
	server := createErrorMockServer(http.StatusInternalServerError, "Server error")
	defer server.Close()

	adapter := NewCerebrasAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestCerebrasAdapter_StreamChatCompletion(t *testing.T) {
	server := createStreamingMockServer()
	defer server.Close()

	adapter := NewCerebrasAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "llama3.1-8b",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	var responses []OpenAIStreamResponse
	for resp := range responseChan {
		responses = append(responses, resp)
	}

	// Check for errors
	select {
	case err := <-errorChan:
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	default:
	}

	assert.NotEmpty(t, responses)
}

func TestCerebrasAdapter_StreamChatCompletion_Error(t *testing.T) {
	server := createErrorMockServer(http.StatusBadRequest, "Bad request")
	defer server.Close()

	adapter := NewCerebrasAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "invalid-model",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	// Drain response channel
	for range responseChan {
	}

	// Check for errors
	select {
	case err := <-errorChan:
		assert.Error(t, err)
	case <-time.After(time.Second):
		// No error is acceptable if the channel was closed
	}
}

func TestCerebrasAdapter_StreamChatCompletion_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay response to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"test\":\"data\"}\n\n")
	}))
	defer server.Close()

	adapter := NewCerebrasAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "llama3.1-8b",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	// Drain channels
	for range responseChan {
	}
	for range errorChan {
	}
}

// ==================== Cloudflare Adapter Tests ====================

func TestNewCloudflareAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.cloudflare.com/client/v4/accounts/ACCOUNT/ai/v1"
	apiKey := "test-key"

	adapter := NewCloudflareAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
}

func TestCloudflareAdapter_ChatCompletion(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewCloudflareAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "@cf/meta/llama-2-7b-chat-int8",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "chatcmpl-123", response.ID)
}

func TestCloudflareAdapter_ListModels(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewCloudflareAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Data, 2)
}

func TestCloudflareAdapter_StreamChatCompletion(t *testing.T) {
	server := createStreamingMockServer()
	defer server.Close()

	adapter := NewCloudflareAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "@cf/meta/llama-2-7b-chat-int8",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	var responses []OpenAIStreamResponse
	for resp := range responseChan {
		responses = append(responses, resp)
	}

	select {
	case err := <-errorChan:
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	default:
	}

	assert.NotEmpty(t, responses)
}

// ==================== DeepSeek Adapter Tests ====================

func TestNewDeepSeekAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.deepseek.com/v1"
	apiKey := "test-key"

	adapter := NewDeepSeekAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
}

func TestDeepSeekAdapter_ChatCompletion(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewDeepSeekAdapter(&http.Client{}, server.URL, "test-key")

	request := DeepSeekChatRequest{
		Model:    "deepseek-chat",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "chatcmpl-123", response.ID)
}

func TestDeepSeekAdapter_ChatCompletion_Error(t *testing.T) {
	server := createErrorMockServer(http.StatusBadRequest, "Bad request") // Use non-retryable error
	defer server.Close()

	adapter := NewDeepSeekAdapter(&http.Client{}, server.URL, "test-key")

	request := DeepSeekChatRequest{
		Model:    "deepseek-chat",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestDeepSeekAdapter_StreamChatCompletion(t *testing.T) {
	server := createStreamingMockServer()
	defer server.Close()

	adapter := NewDeepSeekAdapter(&http.Client{}, server.URL, "test-key")

	request := DeepSeekChatRequest{
		Model:    "deepseek-chat",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	var responses []DeepSeekStreamResponse
	for resp := range responseChan {
		responses = append(responses, resp)
	}

	select {
	case err := <-errorChan:
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	default:
	}

	assert.NotEmpty(t, responses)
}

func TestDeepSeekAdapter_ValidateRequest(t *testing.T) {
	adapter := NewDeepSeekAdapter(&http.Client{}, "https://api.deepseek.com/v1", "test-key")

	tests := []struct {
		name    string
		request DeepSeekChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: DeepSeekChatRequest{
				Model:    "deepseek-chat",
				Messages: []Message{{Role: "user", Content: "Hello"}},
			},
			wantErr: false,
		},
		{
			name: "empty model",
			request: DeepSeekChatRequest{
				Model:    "",
				Messages: []Message{{Role: "user", Content: "Hello"}},
			},
			wantErr: true,
		},
		{
			name: "empty messages",
			request: DeepSeekChatRequest{
				Model:    "deepseek-chat",
				Messages: []Message{},
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

func TestDeepSeekAdapter_GetModelInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/models/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "deepseek-chat",
				"object":   "model",
				"created":  1234567890,
				"owned_by": "deepseek",
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	adapter := NewDeepSeekAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	info, err := adapter.GetModelInfo(ctx, "deepseek-chat")

	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "deepseek-chat", info.ID)
}

func TestDeepSeekAdapter_SSEScanner(t *testing.T) {
	input := "data: {\"test\":\"value\"}\ndata: [DONE]\n"
	reader := strings.NewReader(input)
	scanner := NewSSEScanner(reader)

	assert.True(t, scanner.Scan())
	assert.Equal(t, "data: {\"test\":\"value\"}", scanner.Text())
	assert.True(t, scanner.Scan())
	assert.NoError(t, scanner.Err())
}

// ==================== Mistral Adapter Tests ====================

func TestNewMistralAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.mistral.ai/v1"
	apiKey := "test-key"

	adapter := NewMistralAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
}

func TestMistralAdapter_ChatCompletion(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewMistralAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "mistral-medium",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "chatcmpl-123", response.ID)
}

func TestMistralAdapter_ListModels(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewMistralAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Data, 2)
}

func TestMistralAdapter_StreamChatCompletion(t *testing.T) {
	server := createStreamingMockServer()
	defer server.Close()

	adapter := NewMistralAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "mistral-medium",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	var responses []OpenAIStreamResponse
	for resp := range responseChan {
		responses = append(responses, resp)
	}

	select {
	case err := <-errorChan:
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	default:
	}

	assert.NotEmpty(t, responses)
}

// ==================== SiliconFlow Adapter Tests ====================

func TestNewSiliconFlowAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.siliconflow.cn/v1"
	apiKey := "test-key"

	adapter := NewSiliconFlowAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
}

func TestSiliconFlowAdapter_ChatCompletion(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewSiliconFlowAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "Qwen/Qwen2-7B-Instruct",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "chatcmpl-123", response.ID)
}

func TestSiliconFlowAdapter_ListModels(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewSiliconFlowAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Data, 2)
}

func TestSiliconFlowAdapter_StreamChatCompletion(t *testing.T) {
	server := createStreamingMockServer()
	defer server.Close()

	adapter := NewSiliconFlowAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "Qwen/Qwen2-7B-Instruct",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	var responses []OpenAIStreamResponse
	for resp := range responseChan {
		responses = append(responses, resp)
	}

	select {
	case err := <-errorChan:
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	default:
	}

	assert.NotEmpty(t, responses)
}

// ==================== xAI Adapter Tests ====================

func TestNewxAIAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.x.ai/v1"
	apiKey := "test-key"

	adapter := NewxAIAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
}

func TestXAIAdapter_ChatCompletion(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewxAIAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "grok-beta",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "chatcmpl-123", response.ID)
}

func TestXAIAdapter_ListModels(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewxAIAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Data, 2)
}

func TestXAIAdapter_StreamChatCompletion(t *testing.T) {
	server := createStreamingMockServer()
	defer server.Close()

	adapter := NewxAIAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "grok-beta",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	var responses []OpenAIStreamResponse
	for resp := range responseChan {
		responses = append(responses, resp)
	}

	select {
	case err := <-errorChan:
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	default:
	}

	assert.NotEmpty(t, responses)
}

// ==================== TogetherAI Adapter Tests ====================

func TestNewTogetherAIAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.together.xyz/v1"
	apiKey := "test-key"

	adapter := NewTogetherAIAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
}

func TestTogetherAIAdapter_ChatCompletion(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewTogetherAIAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "meta-llama/Llama-3-8b-chat-hf",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "chatcmpl-123", response.ID)
}

func TestTogetherAIAdapter_ListModels(t *testing.T) {
	server := createMockProviderServer()
	defer server.Close()

	adapter := NewTogetherAIAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Data, 2)
}

func TestTogetherAIAdapter_StreamChatCompletion(t *testing.T) {
	server := createStreamingMockServer()
	defer server.Close()

	adapter := NewTogetherAIAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "meta-llama/Llama-3-8b-chat-hf",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	var responses []OpenAIStreamResponse
	for resp := range responseChan {
		responses = append(responses, resp)
	}

	select {
	case err := <-errorChan:
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	default:
	}

	assert.NotEmpty(t, responses)
}

// ==================== Cohere Adapter Tests ====================

func TestNewCohereAdapter_Full(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.cohere.ai/v1"
	apiKey := "test-key"

	adapter := NewCohereAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
}

// ==================== Anthropic Adapter Tests ====================

func TestNewAnthropicAdapter_Full(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.anthropic.com/v1"
	apiKey := "test-key"

	adapter := NewAnthropicAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "x-api-key")
	assert.Contains(t, adapter.headers, "anthropic-version")
}

func TestAnthropicAdapter_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			w.Header().Set("Content-Type", "application/json")
			// Anthropic returns models in a different format
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "claude-3-opus-20240229", "type": "model"},
					{"id": "claude-3-sonnet-20240229", "type": "model"},
				},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	adapter := NewAnthropicAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
}

// ==================== Replicate Adapter Tests ====================

func TestReplicateAdapter_ListModels(t *testing.T) {
	// Replicate ListModels returns a static list, no server needed
	adapter := NewReplicateAdapter(&http.Client{}, "https://api.replicate.com/v1", "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "list", response.Object)
	assert.NotEmpty(t, response.Data)
	// Check expected models are present
	var foundLlama bool
	for _, model := range response.Data {
		if strings.Contains(model.ID, "llama") {
			foundLlama = true
			break
		}
	}
	assert.True(t, foundLlama, "Should have llama model in list")
}

// ==================== Edge Cases Tests ====================

func TestAdapters_EmptyAPIKey(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.test.com/v1"

	adapters := []struct {
		name    string
		adapter interface{}
	}{
		{"Cerebras", NewCerebrasAdapter(client, endpoint, "")},
		{"Cloudflare", NewCloudflareAdapter(client, endpoint, "")},
		{"DeepSeek", NewDeepSeekAdapter(client, endpoint, "")},
		{"Mistral", NewMistralAdapter(client, endpoint, "")},
		{"SiliconFlow", NewSiliconFlowAdapter(client, endpoint, "")},
		{"xAI", NewxAIAdapter(client, endpoint, "")},
		{"TogetherAI", NewTogetherAIAdapter(client, endpoint, "")},
	}

	for _, tt := range adapters {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.adapter)
		})
	}
}

func TestAdapters_NilHTTPClient(t *testing.T) {
	endpoint := "https://api.test.com/v1"
	apiKey := "test-key"

	// These should not panic even with nil client
	adapters := []struct {
		name    string
		adapter interface{}
	}{
		{"Cerebras", NewCerebrasAdapter(nil, endpoint, apiKey)},
		{"Cloudflare", NewCloudflareAdapter(nil, endpoint, apiKey)},
		{"DeepSeek", NewDeepSeekAdapter(nil, endpoint, apiKey)},
		{"Mistral", NewMistralAdapter(nil, endpoint, apiKey)},
		{"SiliconFlow", NewSiliconFlowAdapter(nil, endpoint, apiKey)},
		{"xAI", NewxAIAdapter(nil, endpoint, apiKey)},
		{"TogetherAI", NewTogetherAIAdapter(nil, endpoint, apiKey)},
	}

	for _, tt := range adapters {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.adapter)
		})
	}
}

func TestAdapters_MalformedJSON(t *testing.T) {
	// Server that returns malformed JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json}"))
	}))
	defer server.Close()

	adapter := NewCerebrasAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestAdapters_NetworkError(t *testing.T) {
	// Use an invalid URL to simulate network error
	adapter := NewCerebrasAdapter(&http.Client{Timeout: time.Millisecond}, "http://invalid.local:99999", "test-key")

	request := OpenAIChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
}
