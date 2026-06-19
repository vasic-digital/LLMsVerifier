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

func TestNewXiaomiMiMoAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.xiaomimimo.com/v1"
	apiKey := "test-key"

	adapter := NewXiaomiMiMoAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
	assert.Equal(t, "Bearer test-key", adapter.headers["Authorization"])
}

func TestXiaomiMiMoAdapterBaseAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.xiaomimimo.com/v1"
	apiKey := "test-key"

	adapter := NewXiaomiMiMoAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter.GetClient())
	assert.Equal(t, endpoint, adapter.GetEndpoint())
	assert.Equal(t, apiKey, adapter.GetAPIKey())
	assert.NotNil(t, adapter.GetHeaders())
}

func TestXiaomiMiMoAdapter_EndpointTrailingSlash(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.xiaomimimo.com/v1/"
	apiKey := "test-key"

	adapter := NewXiaomiMiMoAdapter(client, endpoint, apiKey)

	assert.Equal(t, "https://api.xiaomimimo.com/v1", adapter.endpoint)
}

// createXiaomiMiMoMockServer creates a mock HTTP server for Xiaomi MiMo tests
func createXiaomiMiMoMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			if r.Method == "POST" {
				response := map[string]interface{}{
					"id":      "chatcmpl-mimo-123",
					"object":  "chat.completion",
					"created": time.Now().Unix(),
					"model":   "mimo-v2.5-pro",
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"message": map[string]string{
								"role":    "assistant",
								"content": "Hello from Xiaomi MiMo!",
							},
							"finish_reason": "stop",
						},
					},
					"usage": map[string]int{
						"prompt_tokens":     15,
						"completion_tokens": 25,
						"total_tokens":      40,
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
							"id":       "mimo-v2.5-pro",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "xiaomi",
						},
						{
							"id":       "mimo-v2.5",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "xiaomi",
						},
						{
							"id":       "mimo-v2-flash",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "xiaomi",
						},
						{
							"id":       "mimo-v2.5-asr",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "xiaomi",
						},
						{
							"id":       "mimo-v2.5-tts",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "xiaomi",
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

// createXiaomiMiMoStreamingMockServer creates a mock server that returns SSE stream
func createXiaomiMiMoStreamingMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			streamResp1 := `{"id":"chatcmpl-mimo-123","object":"chat.completion.chunk","created":1234567890,"model":"mimo-v2.5-pro","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`
			streamResp2 := `{"id":"chatcmpl-mimo-123","object":"chat.completion.chunk","created":1234567890,"model":"mimo-v2.5-pro","choices":[{"index":0,"delta":{"content":" from"},"finish_reason":null}]}`
			streamResp3 := `{"id":"chatcmpl-mimo-123","object":"chat.completion.chunk","created":1234567890,"model":"mimo-v2.5-pro","choices":[{"index":0,"delta":{"content":" MiMo!"},"finish_reason":null}]}`
			streamResp4 := `{"id":"chatcmpl-mimo-123","object":"chat.completion.chunk","created":1234567890,"model":"mimo-v2.5-pro","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`

			fmt.Fprintf(w, "data: %s\n\n", streamResp1)
			fmt.Fprintf(w, "data: %s\n\n", streamResp2)
			fmt.Fprintf(w, "data: %s\n\n", streamResp3)
			fmt.Fprintf(w, "data: %s\n\n", streamResp4)
			fmt.Fprintf(w, "data: [DONE]\n\n")
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// createXiaomiMiMoErrorMockServer creates a mock server that returns errors
func createXiaomiMiMoErrorMockServer(statusCode int, errorMessage string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(errorMessage))
	}))
}

func TestXiaomiMiMoAdapter_ChatCompletion(t *testing.T) {
	server := createXiaomiMiMoMockServer()
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "mimo-v2.5-pro",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "chatcmpl-mimo-123", response.ID)
	assert.Len(t, response.Choices, 1)
	assert.Equal(t, "assistant", response.Choices[0].Message.Role)
	assert.Equal(t, "Hello from Xiaomi MiMo!", response.Choices[0].Message.Content)
}

func TestXiaomiMiMoAdapter_ChatCompletion_Error(t *testing.T) {
	server := createXiaomiMiMoErrorMockServer(http.StatusUnauthorized, "Invalid API key")
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "invalid-key")

	request := OpenAIChatRequest{
		Model:    "mimo-v2.5-pro",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "401")
}

func TestXiaomiMiMoAdapter_ChatCompletion_WithOptions(t *testing.T) {
	server := createXiaomiMiMoMockServer()
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "mimo-v2.5-pro",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		TopP:        0.9,
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestXiaomiMiMoAdapter_ListModels(t *testing.T) {
	server := createXiaomiMiMoMockServer()
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "list", response.Object)
	assert.Len(t, response.Data, 5)
}

func TestXiaomiMiMoAdapter_ListModels_Error(t *testing.T) {
	server := createXiaomiMiMoErrorMockServer(http.StatusInternalServerError, "Server error")
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestXiaomiMiMoAdapter_StreamChatCompletion(t *testing.T) {
	server := createXiaomiMiMoStreamingMockServer()
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "mimo-v2.5-pro",
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

func TestXiaomiMiMoAdapter_StreamChatCompletion_Error(t *testing.T) {
	server := createXiaomiMiMoErrorMockServer(http.StatusBadRequest, "Bad request")
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "invalid-model",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	for range responseChan {
	}

	select {
	case err := <-errorChan:
		assert.Error(t, err)
	case <-time.After(time.Second):
	}
}

func TestXiaomiMiMoAdapter_StreamChatCompletion_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"test\":\"data\"}\n\n")
	}))
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "mimo-v2.5-pro",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	responseChan, errorChan := adapter.StreamChatCompletion(ctx, request)

	for range responseChan {
	}
	for range errorChan {
	}
}

func TestXiaomiMiMoAdapter_MultimodalModel(t *testing.T) {
	server := createXiaomiMiMoMockServer()
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "mimo-v2.5",
		Messages: []Message{
			{Role: "user", Content: "Describe this image"},
		},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestXiaomiMiMoAdapter_FlashModel(t *testing.T) {
	server := createXiaomiMiMoMockServer()
	defer server.Close()

	adapter := NewXiaomiMiMoAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "mimo-v2-flash",
		Messages: []Message{
			{Role: "user", Content: "Quick question"},
		},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestXiaomiMiMoProviderConfigRegistration(t *testing.T) {
	registry := NewProviderRegistry()

	config, exists := registry.GetConfig("xiaomi")
	require.True(t, exists, "xiaomi provider should be registered")
	assert.Equal(t, "xiaomi", config.Name)
	assert.Equal(t, "https://api.xiaomimimo.com/v1", config.Endpoint)
	assert.Equal(t, "bearer", config.AuthType)
	assert.Equal(t, "mimo-v2.5-pro", config.DefaultModel)
	assert.Equal(t, 1048576, config.Features["max_context_length"])
	assert.Equal(t, 131072, config.Features["max_output_tokens"])
	assert.Equal(t, true, config.Features["supports_streaming"])
	assert.Equal(t, true, config.Features["supports_functions"])
	assert.Equal(t, true, config.Features["supports_vision"])

	// Verify all 5 models are listed
	models, ok := config.Features["supported_models"].([]string)
	require.True(t, ok)
	assert.Len(t, models, 5)
	assert.Contains(t, models, "mimo-v2.5-pro")
	assert.Contains(t, models, "mimo-v2.5")
	assert.Contains(t, models, "mimo-v2-flash")
	assert.Contains(t, models, "mimo-v2.5-asr")
	assert.Contains(t, models, "mimo-v2.5-tts")
}
