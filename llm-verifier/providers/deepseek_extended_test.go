package providers

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepSeekAdapter_IsRetryableError_WithNetworkError(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	// Network errors should be retryable
	networkErr := errors.New("connection refused")
	result := adapter.isRetryableError(networkErr, 0)
	assert.True(t, result)
}

func TestDeepSeekAdapter_IsRetryableError_TooManyRequests(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	// 429 should be retryable
	result := adapter.isRetryableError(nil, 429)
	assert.True(t, result)
}

func TestDeepSeekAdapter_IsRetryableError_ServerErrors(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	serverErrors := []int{500, 502, 503, 504}
	for _, code := range serverErrors {
		t.Run(string(rune(code)), func(t *testing.T) {
			result := adapter.isRetryableError(nil, code)
			assert.True(t, result)
		})
	}
}

func TestDeepSeekAdapter_IsRetryableError_RequestTimeout(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	// 408 Request Timeout should be retryable
	result := adapter.isRetryableError(nil, 408)
	assert.True(t, result)
}

func TestDeepSeekAdapter_IsRetryableError_ClientErrors(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	clientErrors := []int{400, 401, 403, 404, 422}
	for _, code := range clientErrors {
		t.Run(string(rune(code)), func(t *testing.T) {
			result := adapter.isRetryableError(nil, code)
			assert.False(t, result)
		})
	}
}

func TestDeepSeekAdapter_IsRetryableError_Success(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	// Success codes should not be retryable
	result := adapter.isRetryableError(nil, 200)
	assert.False(t, result)

	result = adapter.isRetryableError(nil, 201)
	assert.False(t, result)
}

func TestDeepSeekAdapter_Headers(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	headers := adapter.GetHeaders()
	assert.Contains(t, headers, "Authorization")
	assert.Equal(t, "Bearer sk-test-key", headers["Authorization"])
	assert.Contains(t, headers, "Content-Type")
	assert.Equal(t, "application/json", headers["Content-Type"])
}

func TestDeepSeekAdapter_ValidateRequest_EmptyMessages(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	req := DeepSeekChatRequest{
		Model:    "deepseek-chat",
		Messages: []Message{},
	}

	err := adapter.ValidateRequest(req)
	assert.Error(t, err)
}

func TestDeepSeekAdapter_ValidateRequest_EmptyModel(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	req := DeepSeekChatRequest{
		Model: "",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	err := adapter.ValidateRequest(req)
	assert.Error(t, err)
}

func TestDeepSeekAdapter_ValidateRequest_NegativeMaxTokens(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	req := DeepSeekChatRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: -1,
	}

	err := adapter.ValidateRequest(req)
	assert.Error(t, err)
}

func TestDeepSeekAdapter_ValidateRequest_InvalidTemperature(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	// Temperature too high
	req := DeepSeekChatRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 3.0,
	}

	err := adapter.ValidateRequest(req)
	assert.Error(t, err)

	// Temperature negative
	req.Temperature = -0.5
	err = adapter.ValidateRequest(req)
	assert.Error(t, err)
}

func TestDeepSeekAdapter_ValidateRequest_ValidRequest(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewDeepSeekAdapter(client, "https://api.deepseek.com", "sk-test-key")

	req := DeepSeekChatRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	err := adapter.ValidateRequest(req)
	assert.NoError(t, err)
}

func TestDeepSeekRetryConfig(t *testing.T) {
	assert.Equal(t, 3, deepSeekRetryConfig.MaxAttempts)
	assert.Equal(t, 1*time.Second, deepSeekRetryConfig.BaseDelay)
	assert.Equal(t, 30*time.Second, deepSeekRetryConfig.MaxDelay)
}

func TestSSEScanner_NewScanner(t *testing.T) {
	reader := strings.NewReader("data: test\n\ndata: test2\n\n")
	scanner := NewSSEScanner(reader)

	require.NotNil(t, scanner)

	// First line
	assert.True(t, scanner.Scan())
	assert.Equal(t, "data: test", scanner.Text())
}

func TestSSEScanner_EmptyReader(t *testing.T) {
	reader := strings.NewReader("")
	scanner := NewSSEScanner(reader)

	assert.False(t, scanner.Scan())
	assert.NoError(t, scanner.Err())
}

func TestSSEScanner_DataWithJSONContent(t *testing.T) {
	reader := strings.NewReader("data: {\"message\": \"hello\"}\n")
	scanner := NewSSEScanner(reader)

	assert.True(t, scanner.Scan())
	assert.Equal(t, "data: {\"message\": \"hello\"}", scanner.Text())
	assert.NoError(t, scanner.Err())
}

func TestSSEScanner_MultipleLines(t *testing.T) {
	reader := strings.NewReader("line1\nline2\nline3\n")
	scanner := NewSSEScanner(reader)

	// First line
	assert.True(t, scanner.Scan())
	assert.Equal(t, "line1", scanner.Text())

	// Second line
	assert.True(t, scanner.Scan())
	assert.Equal(t, "line2", scanner.Text())

	// Third line
	assert.True(t, scanner.Scan())
	assert.Equal(t, "line3", scanner.Text())

	// No more
	assert.False(t, scanner.Scan())
	assert.NoError(t, scanner.Err())
}

// New Cohere adapter tests
func TestCohereAdapter_Headers(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewCohereAdapter(client, "https://api.cohere.ai", "co-test-key")

	headers := adapter.GetHeaders()
	assert.Contains(t, headers, "Authorization")
	assert.Equal(t, "Bearer co-test-key", headers["Authorization"])
}

// Test Mistral adapter headers
func TestMistralAdapter_Headers(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewMistralAdapter(client, "https://api.mistral.ai", "mistral-key")

	headers := adapter.GetHeaders()
	assert.Contains(t, headers, "Authorization")
}

// Test SiliconFlow adapter headers
func TestSiliconFlowAdapter_Headers(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewSiliconFlowAdapter(client, "https://api.siliconflow.cn", "sf-key")

	headers := adapter.GetHeaders()
	assert.Contains(t, headers, "Authorization")
}

// Test TogetherAI adapter headers
func TestTogetherAIAdapter_Headers(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewTogetherAIAdapter(client, "https://api.together.xyz", "tog-key")

	headers := adapter.GetHeaders()
	assert.Contains(t, headers, "Authorization")
}

// Test xAI adapter headers
func TestXAIAdapter_Headers(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := NewxAIAdapter(client, "https://api.x.ai", "xai-key")

	headers := adapter.GetHeaders()
	assert.Contains(t, headers, "Authorization")
}
