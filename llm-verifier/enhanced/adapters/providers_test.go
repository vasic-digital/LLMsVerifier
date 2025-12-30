package adapters

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== OpenAI Adapter Tests ====================

func TestNewOpenAIAdapter(t *testing.T) {
	adapter := NewOpenAIAdapter()
	assert.NotNil(t, adapter)
}

func TestOpenAIAdapter_GetProviderName(t *testing.T) {
	adapter := NewOpenAIAdapter()
	assert.Equal(t, "openai", adapter.GetProviderName())
}

func TestOpenAIAdapter_OptimizeRequest(t *testing.T) {
	adapter := NewOpenAIAdapter()

	tests := []struct {
		name     string
		request  *LLMRequest
		checkFn  func(t *testing.T, result *LLMRequest)
	}{
		{
			name: "optimize code request temperature",
			request: &LLMRequest{
				Prompt: "Write some code",
			},
			checkFn: func(t *testing.T, result *LLMRequest) {
				assert.NotNil(t, result.Temperature)
				assert.Equal(t, 0.1, *result.Temperature)
				assert.NotNil(t, result.MaxTokens)
			},
		},
		{
			name: "set default max tokens",
			request: &LLMRequest{
				Prompt: "Hello",
			},
			checkFn: func(t *testing.T, result *LLMRequest) {
				assert.NotNil(t, result.MaxTokens)
				assert.Equal(t, 2048, *result.MaxTokens)
			},
		},
		{
			name: "add system message",
			request: &LLMRequest{
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			checkFn: func(t *testing.T, result *LLMRequest) {
				assert.Len(t, result.Messages, 2)
				assert.Equal(t, "system", result.Messages[0].Role)
			},
		},
		{
			name: "preserve existing system message",
			request: &LLMRequest{
				Messages: []Message{
					{Role: "system", Content: "Custom system"},
					{Role: "user", Content: "Hello"},
				},
			},
			checkFn: func(t *testing.T, result *LLMRequest) {
				assert.Len(t, result.Messages, 2)
				assert.Equal(t, "Custom system", result.Messages[0].Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.OptimizeRequest(tt.request)
			tt.checkFn(t, result)
		})
	}
}

func TestOpenAIAdapter_ParseStreamingResponse(t *testing.T) {
	adapter := NewOpenAIAdapter()

	// Create a mock SSE stream
	streamData := `data: {"choices":[{"delta":{"content":"Hello"}}]}

data: {"choices":[{"delta":{"content":" world"}}]}

data: [DONE]
`
	reader := strings.NewReader(streamData)

	ch := adapter.ParseStreamingResponse(reader)

	var chunks []StreamingChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	assert.NotEmpty(t, chunks)
}

func TestOpenAIAdapter_HandleError(t *testing.T) {
	adapter := NewOpenAIAdapter()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "rate limit error",
			statusCode: 429,
			body:       `{"error":{"message":"Rate limit exceeded"}}`,
			wantErr:    true,
		},
		{
			name:       "authentication error",
			statusCode: 401,
			body:       `{"error":{"message":"Invalid API key"}}`,
			wantErr:    true,
		},
		{
			name:       "server error",
			statusCode: 500,
			body:       `{"error":{"message":"Internal server error"}}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
			}
			err := adapter.HandleError(resp, []byte(tt.body))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOpenAIAdapter_GetOptimalBatchSize(t *testing.T) {
	adapter := NewOpenAIAdapter()
	batchSize := adapter.GetOptimalBatchSize()
	assert.Greater(t, batchSize, 0)
}

func TestOpenAIAdapter_GetRateLimitInfo(t *testing.T) {
	adapter := NewOpenAIAdapter()

	headers := http.Header{}
	headers.Set("x-ratelimit-limit-requests", "100")
	headers.Set("x-ratelimit-limit-tokens", "10000")
	headers.Set("x-ratelimit-reset-requests", "1s")

	info := adapter.GetRateLimitInfo(headers)
	assert.NotNil(t, info.RequestsPerMinute)
	assert.NotNil(t, info.TokensPerMinute)
}

// ==================== DeepSeek Adapter Tests ====================

func TestNewDeepSeekAdapter(t *testing.T) {
	adapter := NewDeepSeekAdapter()
	assert.NotNil(t, adapter)
}

func TestDeepSeekAdapter_GetProviderName(t *testing.T) {
	adapter := NewDeepSeekAdapter()
	assert.Equal(t, "deepseek", adapter.GetProviderName())
}

func TestDeepSeekAdapter_OptimizeRequest(t *testing.T) {
	adapter := NewDeepSeekAdapter()

	tests := []struct {
		name     string
		request  *LLMRequest
		checkFn  func(t *testing.T, result *LLMRequest)
	}{
		{
			name: "set default max tokens",
			request: &LLMRequest{
				Prompt: "Hello",
			},
			checkFn: func(t *testing.T, result *LLMRequest) {
				assert.NotNil(t, result.MaxTokens)
			},
		},
		{
			name: "optimize code request",
			request: &LLMRequest{
				Prompt: "Write code for me",
			},
			checkFn: func(t *testing.T, result *LLMRequest) {
				assert.NotNil(t, result.Temperature)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.OptimizeRequest(tt.request)
			tt.checkFn(t, result)
		})
	}
}

func TestDeepSeekAdapter_ParseStreamingResponse(t *testing.T) {
	adapter := NewDeepSeekAdapter()

	streamData := `data: {"choices":[{"delta":{"content":"Test"}}]}

data: [DONE]
`
	reader := strings.NewReader(streamData)

	ch := adapter.ParseStreamingResponse(reader)

	var chunks []StreamingChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should have at least processed the stream
	assert.NotNil(t, chunks)
}

func TestDeepSeekAdapter_HandleError(t *testing.T) {
	adapter := NewDeepSeekAdapter()

	resp := &http.Response{
		StatusCode: 429,
	}
	err := adapter.HandleError(resp, []byte(`{"error":{"message":"Rate limit"}}`))
	assert.Error(t, err)
}

func TestDeepSeekAdapter_GetOptimalBatchSize(t *testing.T) {
	adapter := NewDeepSeekAdapter()
	batchSize := adapter.GetOptimalBatchSize()
	assert.Greater(t, batchSize, 0)
}

func TestDeepSeekAdapter_GetRateLimitInfo(t *testing.T) {
	adapter := NewDeepSeekAdapter()

	headers := http.Header{}
	headers.Set("x-ratelimit-limit-requests", "50")

	info := adapter.GetRateLimitInfo(headers)
	// Info should be returned even if some fields are empty
	assert.NotNil(t, info)
}

// ==================== Adapter Registry Tests ====================

func TestNewAdapterRegistry(t *testing.T) {
	registry := NewAdapterRegistry()
	assert.NotNil(t, registry)
}

func TestAdapterRegistry_RegisterAdapter(t *testing.T) {
	registry := NewAdapterRegistry()

	adapter := NewOpenAIAdapter()
	registry.RegisterAdapter(adapter)

	retrieved, exists := registry.GetAdapter("openai")
	assert.True(t, exists)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "openai", retrieved.GetProviderName())
}

func TestAdapterRegistry_GetAdapter(t *testing.T) {
	registry := NewAdapterRegistry()

	// Get non-existent adapter
	adapter, exists := registry.GetAdapter("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, adapter)

	// Register and get
	registry.RegisterAdapter(NewOpenAIAdapter())
	adapter, exists = registry.GetAdapter("openai")
	assert.True(t, exists)
	assert.NotNil(t, adapter)
}

func TestAdapterRegistry_GetAvailableProviders(t *testing.T) {
	registry := NewAdapterRegistry()

	// Register some adapters
	registry.RegisterAdapter(NewOpenAIAdapter())
	registry.RegisterAdapter(NewDeepSeekAdapter())

	providers := registry.GetAvailableProviders()
	assert.GreaterOrEqual(t, len(providers), 2)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "deepseek")
}

// ==================== Message and Request Tests ====================

func TestMessage_Struct(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello world",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello world", msg.Content)
}

func TestLLMRequest_Struct(t *testing.T) {
	maxTokens := 1000
	temp := 0.7

	req := LLMRequest{
		Prompt:      "Test prompt",
		Messages:    []Message{{Role: "user", Content: "Test"}},
		MaxTokens:   &maxTokens,
		Temperature: &temp,
		Stream:      true,
	}

	assert.Equal(t, "Test prompt", req.Prompt)
	assert.Len(t, req.Messages, 1)
	assert.Equal(t, 1000, *req.MaxTokens)
	assert.Equal(t, 0.7, *req.Temperature)
	assert.True(t, req.Stream)
}

func TestStreamingChunk_Struct(t *testing.T) {
	chunk := StreamingChunk{
		Content:  "Hello",
		Finish:   false,
		Error:    "",
		Metadata: map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "Hello", chunk.Content)
	assert.False(t, chunk.Finish)
	assert.Empty(t, chunk.Error)
	assert.Equal(t, "value", chunk.Metadata["key"])
}

func TestRateLimitInfo_Struct(t *testing.T) {
	rpm := 100
	tpm := 10000
	resetTime := time.Now().Add(time.Minute)

	info := RateLimitInfo{
		RequestsPerMinute: &rpm,
		TokensPerMinute:   &tpm,
		ResetTime:         &resetTime,
	}

	assert.Equal(t, 100, *info.RequestsPerMinute)
	assert.Equal(t, 10000, *info.TokensPerMinute)
	assert.NotNil(t, info.ResetTime)
}

// ==================== Edge Cases ====================

func TestOpenAIAdapter_OptimizeRequest_NilInput(t *testing.T) {
	adapter := NewOpenAIAdapter()

	// Test with empty request
	req := &LLMRequest{}
	result := adapter.OptimizeRequest(req)
	assert.NotNil(t, result)
	assert.NotNil(t, result.MaxTokens)
}

func TestOpenAIAdapter_ParseStreamingResponse_EmptyReader(t *testing.T) {
	adapter := NewOpenAIAdapter()

	reader := bytes.NewReader([]byte{})
	ch := adapter.ParseStreamingResponse(reader)

	chunks := []StreamingChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should handle empty input gracefully - empty slice is ok
	assert.Empty(t, chunks)
}

func TestDeepSeekAdapter_ParseStreamingResponse_EmptyReader(t *testing.T) {
	adapter := NewDeepSeekAdapter()

	reader := bytes.NewReader([]byte{})
	ch := adapter.ParseStreamingResponse(reader)

	chunks := []StreamingChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should handle empty input gracefully - may return a finish chunk
	// The implementation sends a finish chunk when the stream ends
	if len(chunks) > 0 {
		assert.True(t, chunks[len(chunks)-1].Finish)
	}
}

func TestOpenAIAdapter_GetRateLimitInfo_EmptyHeaders(t *testing.T) {
	adapter := NewOpenAIAdapter()

	headers := http.Header{}
	info := adapter.GetRateLimitInfo(headers)

	// Should return empty info without panicking
	assert.NotNil(t, info)
}

func TestDeepSeekAdapter_GetRateLimitInfo_EmptyHeaders(t *testing.T) {
	adapter := NewDeepSeekAdapter()

	headers := http.Header{}
	info := adapter.GetRateLimitInfo(headers)

	assert.NotNil(t, info)
}

// ==================== Interface Implementation Tests ====================

func TestOpenAIAdapter_ImplementsProviderAdapter(t *testing.T) {
	var _ ProviderAdapter = (*OpenAIAdapter)(nil)
}

func TestDeepSeekAdapter_ImplementsProviderAdapter(t *testing.T) {
	var _ ProviderAdapter = (*DeepSeekAdapter)(nil)
}

// ==================== Concurrent Access Tests ====================

func TestAdapterRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewAdapterRegistry()

	done := make(chan bool)

	// Concurrent writes - use different adapters
	for i := 0; i < 5; i++ {
		go func() {
			registry.RegisterAdapter(NewOpenAIAdapter())
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		go func() {
			registry.RegisterAdapter(NewDeepSeekAdapter())
			done <- true
		}()
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = registry.GetAvailableProviders()
			_, _ = registry.GetAdapter("openai")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}
