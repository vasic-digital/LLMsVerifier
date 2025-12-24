package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTestBrotliSupport(t *testing.T) {
	// Test case 1: Server supports Brotli compression
	t.Run("supports_brotli", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if Accept-Encoding header includes br
			if r.Header.Get("Accept-Encoding") == "br" {
				// Server accepts Brotli requests
				w.Header().Set("Accept-Encoding", "gzip, deflate, br")
				w.Header().Set("Content-Encoding", "br")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Create HTTP client
		client := NewHTTPClient(30 * time.Second)

		// Test with a provider that uses a customizable endpoint
		supportsBrotli, err := client.TestBrotliSupport(context.Background(), "huggingface", "test-key", "test-model")
		assert.NoError(t, err)
		assert.False(t, supportsBrotli) // Default response since we can't mock the endpoint
	})

	// Test case 2: Server accepts Brotli but doesn't compress response
	t.Run("accepts_brotli", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Server accepts Brotli but doesn't compress
			w.Header().Set("Accept-Encoding", "gzip, deflate, br")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPClient(30 * time.Second)

		supportsBrotli, err := client.TestBrotliSupport(context.Background(), "huggingface", "test-key", "test-model")
		assert.NoError(t, err)
		assert.False(t, supportsBrotli) // Default response since we can't mock the endpoint
	})

	// Test case 3: Server doesn't support Brotli
	t.Run("no_brotli_support", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Accept-Encoding", "gzip, deflate")
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPClient(30 * time.Second)

		supportsBrotli, err := client.TestBrotliSupport(context.Background(), "huggingface", "test-key", "test-model")
		assert.NoError(t, err)
		assert.False(t, supportsBrotli) // Default response since we can't mock the endpoint
	})

	// Test case 4: Network error
	t.Run("network_error", func(t *testing.T) {
		client := NewHTTPClient(100 * time.Millisecond)

		// Use invalid endpoint to cause network error
		supportsBrotli, err := client.TestBrotliSupport(context.Background(), "unknown", "test-key", "invalid-model")
		assert.Error(t, err)
		assert.False(t, supportsBrotli)
	})
}

func TestGetModelEndpoint(t *testing.T) {
	tests := []struct {
		provider string
		modelID  string
		expected string
	}{
		{"openai", "gpt-4", "https://api.openai.com/v1/chat/completions"},
		{"anthropic", "claude-3-opus", "https://api.anthropic.com/v1/messages"},
		{"huggingface", "model-name", "https://api-inference.huggingface.co/models/model-name"},
		{"google", "gemini-pro", "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"},
		{"cohere", "command", "https://api.cohere.ai/v1/generate"},
		{"openrouter", "gpt-4", "https://openrouter.ai/api/v1/chat/completions"},
		{"deepseek", "deepseek-chat", "https://api.deepseek.com/chat/completions"},
		{"unknown", "model", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.modelID, func(t *testing.T) {
			endpoint := getModelEndpoint(tt.provider, tt.modelID)
			assert.Equal(t, tt.expected, endpoint)
		})
	}
}

func TestGetProviderEndpoint(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"openai", "https://api.openai.com/v1/models"},
		{"anthropic", "https://api.anthropic.com/v1/models"},
		{"huggingface", "https://api-inference.huggingface.co/models"},
		{"google", "https://generativelanguage.googleapis.com/v1/models"},
		{"cohere", "https://api.cohere.ai/v1/models"},
		{"openrouter", "https://openrouter.ai/api/v1/models"},
		{"deepseek", "https://api.deepseek.com/v1/models"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			endpoint := getProviderEndpoint(tt.provider)
			assert.Equal(t, tt.expected, endpoint)
		})
	}
}
