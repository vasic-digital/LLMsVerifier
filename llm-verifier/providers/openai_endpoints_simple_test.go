package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestOpenAIEndpoint_BasicChat tests basic chat completion functionality
func TestOpenAIEndpoint_BasicChat(t *testing.T) {
	// Mock server for chat completions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "chat-1",
			"object": "chat.completion",
			"choices": [{"message": {"content": "Hello!"}}]
		}`))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(&http.Client{Timeout: 10 * time.Second}, server.URL, "test-key")

	t.Run("AdapterCreation", func(t *testing.T) {
		assert.NotNil(t, adapter)
		assert.Equal(t, "Bearer test-key", adapter.headers["Authorization"])
		assert.Equal(t, "application/json", adapter.headers["Content-Type"])
	})
}

// TestOpenAIEndpoint_BasicStreaming tests basic streaming functionality
func TestOpenAIEndpoint_BasicStreaming(t *testing.T) {
	// Mock server for streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(&http.Client{Timeout: 10 * time.Second}, server.URL, "test-key")

	t.Run("StreamingSupport", func(t *testing.T) {
		request := OpenAIChatRequest{
			Model: "gpt-3.5-turbo",
			Messages: []Message{{Role: "user", Content: "Hello"}},
			Stream: true,
		}

		respChan, errChan := adapter.StreamChatCompletion(context.Background(), request)
		assert.NotNil(t, respChan)
		assert.NotNil(t, errChan)

		// Collect response
		count := 0
		for range respChan {
			count++
		}
		assert.Greater(t, count, 0)
	})
}

// TestOpenAIEndpoint_ValidationError tests request validation
func TestOpenAIEndpoint_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "Invalid request"}}`))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(&http.Client{Timeout: 10 * time.Second}, server.URL, "test-key")

	t.Run("ValidationError", func(t *testing.T) {
		request := OpenAIChatRequest{
			Model: "", // Empty model should fail validation
			Messages: []Message{{Role: "user", Content: "Hello"}},
		}

		err := adapter.ValidateRequest(request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model is required")
	})
}

// TestOpenAIEndpoint_NetworkError tests network error handling
func TestOpenAIEndpoint_NetworkError(t *testing.T) {
	// Use invalid URL to simulate network error
	adapter := NewOpenAIAdapter(&http.Client{Timeout: 1 * time.Second}, "http://invalid-url", "test-key")

	t.Run("NetworkError", func(t *testing.T) {
		request := OpenAIChatRequest{
			Model: "gpt-3.5-turbo",
			Messages: []Message{{Role: "user", Content: "Hello"}},
		}

		respChan, errChan := adapter.StreamChatCompletion(context.Background(), request)
		
		// Should receive error
		select {
		case err := <-errChan:
			assert.Error(t, err)
		case <-respChan:
			assert.Fail(t, "Should receive error instead of response")
		case <-time.After(5 * time.Second):
			assert.Fail(t, "Should receive error within timeout")
		}
	})
}

// TestOpenAIEndpoint_CorrectHeaders tests correct header handling
func TestOpenAIEndpoint_CorrectHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(&http.Client{Timeout: 10 * time.Second}, server.URL, "test-key")

	t.Run("CorrectHeaders", func(t *testing.T) {
		request := OpenAIChatRequest{
			Model: "gpt-3.5-turbo",
			Messages: []Message{{Role: "user", Content: "Hello"}},
		}

		respChan, errChan := adapter.StreamChatCompletion(context.Background(), request)
		assert.NotNil(t, respChan)
		assert.NotNil(t, errChan)
		
		// Drain channels to avoid blocking
		for range respChan {}
		select {
		case <-errChan:
		default:
		}
	})
}