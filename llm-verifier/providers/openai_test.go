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

func TestNewOpenAIAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.openai.com/v1"
	apiKey := "test-key"

	adapter := NewOpenAIAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
	assert.Equal(t, "Bearer test-key", adapter.headers["Authorization"])
}

func TestOpenAIAdapter_EndpointTrailingSlash(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.openai.com/v1/"
	apiKey := "test-key"

	adapter := NewOpenAIAdapter(client, endpoint, apiKey)

	// Should strip trailing slash
	assert.Equal(t, "https://api.openai.com/v1", adapter.endpoint)
}

// createOpenAIMockServer creates a mock HTTP server for OpenAI tests
func createOpenAIMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			if r.Method == "POST" {
				response := map[string]interface{}{
					"id":      "chatcmpl-openai-123",
					"object":  "chat.completion",
					"created": time.Now().Unix(),
					"model":   "gpt-4",
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"message": map[string]string{
								"role":    "assistant",
								"content": "Hello from OpenAI!",
							},
							"finish_reason": "stop",
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
		case strings.Contains(r.URL.Path, "/models/"):
			if r.Method == "GET" {
				modelID := strings.TrimPrefix(r.URL.Path, "/models/")
				response := map[string]interface{}{
					"id":       modelID,
					"object":   "model",
					"created":  time.Now().Unix(),
					"owned_by": "openai",
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
							"id":       "gpt-4",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "openai",
						},
						{
							"id":       "gpt-3.5-turbo",
							"object":   "model",
							"created":  time.Now().Unix(),
							"owned_by": "openai",
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

// createOpenAIStreamingMockServer creates a mock server that returns SSE stream
func createOpenAIStreamingMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Write streaming response
			streamResp1 := `{"id":"chatcmpl-openai-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`
			streamResp2 := `{"id":"chatcmpl-openai-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" from"},"finish_reason":null}]}`
			streamResp3 := `{"id":"chatcmpl-openai-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" OpenAI!"},"finish_reason":null}]}`
			streamResp4 := `{"id":"chatcmpl-openai-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`

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

// createOpenAIErrorMockServer creates a mock server that returns errors
func createOpenAIErrorMockServer(statusCode int, errorMessage string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(errorMessage))
	}))
}

func TestOpenAIAdapter_StreamChatCompletion(t *testing.T) {
	server := createOpenAIStreamingMockServer()
	defer server.Close()

	adapter := NewOpenAIAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "gpt-4",
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

func TestOpenAIAdapter_StreamChatCompletion_Error(t *testing.T) {
	server := createOpenAIErrorMockServer(http.StatusUnauthorized, "Invalid API key")
	defer server.Close()

	adapter := NewOpenAIAdapter(&http.Client{}, server.URL, "invalid-key")

	request := OpenAIChatRequest{
		Model:    "gpt-4",
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
		assert.Contains(t, err.Error(), "401")
	case <-time.After(time.Second):
		t.Fatal("Expected error but got none")
	}
}

func TestOpenAIAdapter_StreamChatCompletion_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay response to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"test\":\"data\"}\n\n")
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(&http.Client{}, server.URL, "test-key")

	request := OpenAIChatRequest{
		Model:    "gpt-4",
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

func TestOpenAIAdapter_ValidateRequest(t *testing.T) {
	adapter := NewOpenAIAdapter(&http.Client{}, "https://api.openai.com/v1", "test-key")

	tests := []struct {
		name    string
		request OpenAIChatRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: OpenAIChatRequest{
				Model:       "gpt-4",
				Messages:    []Message{{Role: "user", Content: "Hello"}},
				Temperature: 0.7,
				MaxTokens:   100,
			},
			wantErr: false,
		},
		{
			name: "missing model",
			request: OpenAIChatRequest{
				Messages: []Message{{Role: "user", Content: "Hello"}},
			},
			wantErr: true,
			errMsg:  "model is required",
		},
		{
			name: "missing messages",
			request: OpenAIChatRequest{
				Model:    "gpt-4",
				Messages: []Message{},
			},
			wantErr: true,
			errMsg:  "at least one message is required",
		},
		{
			name: "negative max_tokens",
			request: OpenAIChatRequest{
				Model:     "gpt-4",
				Messages:  []Message{{Role: "user", Content: "Hello"}},
				MaxTokens: -1,
			},
			wantErr: true,
			errMsg:  "max_tokens cannot be negative",
		},
		{
			name: "temperature too high",
			request: OpenAIChatRequest{
				Model:       "gpt-4",
				Messages:    []Message{{Role: "user", Content: "Hello"}},
				Temperature: 2.5,
			},
			wantErr: true,
			errMsg:  "temperature must be between 0 and 2",
		},
		{
			name: "temperature too low",
			request: OpenAIChatRequest{
				Model:       "gpt-4",
				Messages:    []Message{{Role: "user", Content: "Hello"}},
				Temperature: -0.5,
			},
			wantErr: true,
			errMsg:  "temperature must be between 0 and 2",
		},
		{
			name: "top_p too high",
			request: OpenAIChatRequest{
				Model:    "gpt-4",
				Messages: []Message{{Role: "user", Content: "Hello"}},
				TopP:     1.5,
			},
			wantErr: true,
			errMsg:  "top_p must be between 0 and 1",
		},
		{
			name: "top_p too low",
			request: OpenAIChatRequest{
				Model:    "gpt-4",
				Messages: []Message{{Role: "user", Content: "Hello"}},
				TopP:     -0.1,
			},
			wantErr: true,
			errMsg:  "top_p must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.ValidateRequest(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOpenAIAdapter_GetModelInfo(t *testing.T) {
	server := createOpenAIMockServer()
	defer server.Close()

	adapter := NewOpenAIAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	modelInfo, err := adapter.GetModelInfo(ctx, "gpt-4")

	require.NoError(t, err)
	assert.NotNil(t, modelInfo)
	assert.Equal(t, "gpt-4", modelInfo.ID)
	assert.Equal(t, "model", modelInfo.Object)
	assert.Equal(t, "openai", modelInfo.OwnedBy)
}

func TestOpenAIAdapter_GetModelInfo_Error(t *testing.T) {
	server := createOpenAIErrorMockServer(http.StatusNotFound, "Model not found")
	defer server.Close()

	adapter := NewOpenAIAdapter(&http.Client{}, server.URL, "test-key")

	ctx := context.Background()
	modelInfo, err := adapter.GetModelInfo(ctx, "nonexistent-model")

	assert.Error(t, err)
	assert.Nil(t, modelInfo)
}

// ==================== OpenAI Types Tests ====================

func TestOpenAIChatRequest_JSON(t *testing.T) {
	request := OpenAIChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
		TopP:        0.9,
		Stream:      true,
	}

	data, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded OpenAIChatRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, request.Model, decoded.Model)
	assert.Equal(t, len(request.Messages), len(decoded.Messages))
	assert.Equal(t, request.MaxTokens, decoded.MaxTokens)
	assert.Equal(t, request.Temperature, decoded.Temperature)
}

func TestOpenAIStreamResponse_JSON(t *testing.T) {
	jsonData := `{
		"id": "chatcmpl-123",
		"object": "chat.completion.chunk",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"delta": {
				"role": "assistant",
				"content": "Hello"
			},
			"finish_reason": null
		}]
	}`

	var resp OpenAIStreamResponse
	err := json.Unmarshal([]byte(jsonData), &resp)
	require.NoError(t, err)

	assert.Equal(t, "chatcmpl-123", resp.ID)
	assert.Equal(t, "chat.completion.chunk", resp.Object)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, "assistant", resp.Choices[0].Delta.Role)
	assert.Equal(t, "Hello", resp.Choices[0].Delta.Content)
}

func TestMessage_JSON(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello, how are you?",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.Role, decoded.Role)
	assert.Equal(t, msg.Content, decoded.Content)
}

func TestModelInfo_Struct(t *testing.T) {
	info := ModelInfo{
		ID:      "gpt-4",
		Object:  "model",
		Created: 1234567890,
		OwnedBy: "openai",
	}

	assert.Equal(t, "gpt-4", info.ID)
	assert.Equal(t, "model", info.Object)
	assert.Equal(t, int64(1234567890), info.Created)
	assert.Equal(t, "openai", info.OwnedBy)
}
