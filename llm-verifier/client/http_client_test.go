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
	// §11.4.50 determinism / §11.4.1 anti-FAIL-bluff: the previous version of
	// this test spun up an httptest server but never pointed the client at it,
	// so every subtest dialed the hardcoded LIVE huggingface host. It "passed"
	// only when the network + DNS + the real endpoint happened to cooperate and
	// returned no brotli header — an environment accident, not a product-behaviour
	// assertion. Each subtest below now redirects the client to its controllable
	// httptest server via SetEndpointResolver and asserts the REAL outcome of the
	// brotli-detection logic (Content-Encoding "br" OR Accept-Encoding "br").

	// Test case 1: Server compresses the response with Brotli → supported.
	t.Run("supports_brotli", func(t *testing.T) {
		var gotAcceptEncoding string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAcceptEncoding = r.Header.Get("Accept-Encoding")
			w.Header().Set("Content-Encoding", "br")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPClient(30 * time.Second)
		client.SetEndpointResolver(func(_, _ string) string { return server.URL })

		supportsBrotli, err := client.TestBrotliSupport(context.Background(), "huggingface", "test-key", "test-model")
		assert.NoError(t, err)
		assert.Equal(t, "br", gotAcceptEncoding, "client must request Brotli via Accept-Encoding")
		assert.True(t, supportsBrotli, "Content-Encoding: br must be detected as Brotli support")
	})

	// Test case 2: Server advertises Brotli acceptance but does not compress → still supported.
	t.Run("accepts_brotli", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Accept-Encoding", "gzip, deflate, br")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPClient(30 * time.Second)
		client.SetEndpointResolver(func(_, _ string) string { return server.URL })

		supportsBrotli, err := client.TestBrotliSupport(context.Background(), "huggingface", "test-key", "test-model")
		assert.NoError(t, err)
		assert.True(t, supportsBrotli, "Accept-Encoding containing br must be detected as Brotli support")
	})

	// Test case 3: Server neither compresses with nor accepts Brotli → not supported.
	t.Run("no_brotli_support", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Accept-Encoding", "gzip, deflate")
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPClient(30 * time.Second)
		client.SetEndpointResolver(func(_, _ string) string { return server.URL })

		supportsBrotli, err := client.TestBrotliSupport(context.Background(), "huggingface", "test-key", "test-model")
		assert.NoError(t, err)
		assert.False(t, supportsBrotli, "no br in Content-Encoding/Accept-Encoding must be detected as no Brotli support")
	})

	// Test case 4: Network error → error surfaced, support false.
	t.Run("network_error", func(t *testing.T) {
		client := NewHTTPClient(100 * time.Millisecond)
		// Resolve to a closed/unroutable port to force a deterministic dial error
		// (no live-DNS dependence).
		client.SetEndpointResolver(func(_, _ string) string { return "http://127.0.0.1:1" })

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
		{"huggingface", "model-name", "https://router.huggingface.co/v1/chat/completions"},
		{"google", "gemini-pro", "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"},
		{"cohere", "command", "https://api.cohere.com/v2/chat"},
		{"openrouter", "gpt-4", "https://openrouter.ai/api/v1/chat/completions"},
		{"deepseek", "deepseek-chat", "https://api.deepseek.com/v1/chat/completions"},
		{"groq", "llama2-70b", "https://api.groq.com/openai/v1/chat/completions"},
		{"togetherai", "llama-2-70b", "https://api.together.xyz/v1/chat/completions"},
		{"fireworks", "llama-v2-7b", "https://api.fireworks.ai/v1/chat/completions"},
		{"poe", "GPT-4", "https://api.poe.com/v1/chat/completions"},
		{"navigator", "mistral-small-3.1", "https://api.ai.it.ufl.edu/v1/chat/completions"},
		{"replicate", "llama-2-70b", "https://api.replicate.com/v1/predictions"},
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
		{"huggingface", "https://router.huggingface.co/v1/models"},
		{"google", "https://generativelanguage.googleapis.com/v1/models"},
		{"cohere", "https://api.cohere.com/v2/models"},
		{"openrouter", "https://openrouter.ai/api/v1/models"},
		{"deepseek", "https://api.deepseek.com/v1/models"},
		{"groq", "https://api.groq.com/openai/v1/models"},
		{"togetherai", "https://api.together.xyz/v1/models"},
		{"fireworks", "https://api.fireworks.ai/v1/models"},
		{"poe", "https://api.poe.com/v1/models"},
		{"navigator", "https://api.ai.it.ufl.edu/v1/models"},
		{"replicate", "https://api.replicate.com/v1/models"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			endpoint := getProviderEndpoint(tt.provider)
			assert.Equal(t, tt.expected, endpoint)
		})
	}
}
