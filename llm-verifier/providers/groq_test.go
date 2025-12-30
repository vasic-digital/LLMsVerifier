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

func TestNewGroqAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.groq.com/openai/v1"
	apiKey := "test-key"

	adapter := NewGroqAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
	assert.Equal(t, "Bearer test-key", adapter.headers["Authorization"])
}

func TestGroqAdapterBaseAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.groq.com/openai/v1"
	apiKey := "test-key"

	adapter := NewGroqAdapter(client, endpoint, apiKey)

	// Test base adapter methods
	assert.NotNil(t, adapter.GetClient())
	assert.Equal(t, endpoint, adapter.GetEndpoint())
	assert.Equal(t, apiKey, adapter.GetAPIKey())
	assert.NotNil(t, adapter.GetHeaders())
}

func TestGroqAdapter_EndpointTrailingSlash(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.groq.com/openai/v1/"
	apiKey := "test-key"

	adapter := NewGroqAdapter(client, endpoint, apiKey)

	// Should strip trailing slash
	assert.Equal(t, "https://api.groq.com/openai/v1", adapter.endpoint)
}

// createGroqMockServer creates a mock HTTP server for Groq tests
func createGroqMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			if r.Method == "POST" {
				response := map[string]interface{}{
					"id":      "chatcmpl-groq-123",
					"object":  "chat.completion",
					"created": time.Now().Unix(),
					"model":   "llama-3.1-70b-versatile",
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"message": map[string]string{
								"role":    "assistant",
								"content": "Hello from Groq!",
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
							"id":       "llama-3.1-70b-versatile",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "groq",
						},
						{
							"id":       "llama-3.1-8b-instant",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "groq",
						},
						{
							"id":       "mixtral-8x7b-32768",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "groq",
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

// createGroqStreamingMockServer creates a mock server that returns SSE stream
func createGroqStreamingMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Write streaming response
			streamResp1 := `{"id":"chatcmpl-groq-123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`
			streamResp2 := `{"id":"chatcmpl-groq-123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{"content":" from"},"finish_reason":null}]}`
			streamResp3 := `{"id":"chatcmpl-groq-123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{"content":" Groq!"},"finish_reason":null}]}`
			streamResp4 := `{"id":"chatcmpl-groq-123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`

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

// createGroqErrorMockServer creates a mock server that returns errors
func createGroqErrorMockServer(statusCode int, errorMessage string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(errorMessage))
	}))
}

func TestGroqAdapter_ChatCompletion(t *testing.T) {
	server := createGroqMockServer()
	defer server.Close()

	adapter := NewGroqAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "llama-3.1-70b-versatile",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "chatcmpl-groq-123", response.ID)
	assert.Len(t, response.Choices, 1)
	assert.Equal(t, "assistant", response.Choices[0].Message.Role)
	assert.Equal(t, "Hello from Groq!", response.Choices[0].Message.Content)
}

func TestGroqAdapter_ChatCompletion_Error(t *testing.T) {
	server := createGroqErrorMockServer(http.StatusUnauthorized, "Invalid API key")
	defer server.Close()

	adapter := NewGroqAdapter(&http.Client{}, server.URL, "invalid-key")

	request := OpenAIChatRequest{
		Model:    "llama-3.1-70b-versatile",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	response, err := adapter.ChatCompletion(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "401")
}

func TestGroqAdapter_ChatCompletion_WithOptions(t *testing.T) {
	server := createGroqMockServer()
	defer server.Close()

	adapter := NewGroqAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model: "llama-3.1-70b-versatile",
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

func TestGroqAdapter_ListModels(t *testing.T) {
	server := createGroqMockServer()
	defer server.Close()

	adapter := NewGroqAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "list", response.Object)
	assert.Len(t, response.Data, 3)
}

func TestGroqAdapter_ListModels_Error(t *testing.T) {
	server := createGroqErrorMockServer(http.StatusInternalServerError, "Server error")
	defer server.Close()

	adapter := NewGroqAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	response, err := adapter.ListModels(ctx)

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestGroqAdapter_StreamChatCompletion(t *testing.T) {
	server := createGroqStreamingMockServer()
	defer server.Close()

	adapter := NewGroqAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "llama-3.1-70b-versatile",
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

func TestGroqAdapter_StreamChatCompletion_Error(t *testing.T) {
	server := createGroqErrorMockServer(http.StatusBadRequest, "Bad request")
	defer server.Close()

	adapter := NewGroqAdapter(&http.Client{}, server.URL, "test-key")

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

func TestGroqAdapter_StreamChatCompletion_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay response to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"test\":\"data\"}\n\n")
	}))
	defer server.Close()

	adapter := NewGroqAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "llama-3.1-70b-versatile",
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
