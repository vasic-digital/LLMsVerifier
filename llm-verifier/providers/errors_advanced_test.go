package providers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Error Classification Tests ====================

func TestErrorClassifier_ClassifyOpenAIErrorAdvanced(t *testing.T) {
	classifier := NewErrorClassifier("openai")

	tests := []struct {
		name         string
		statusCode   int
		expectedCode string
		retryable    bool
	}{
		{name: "bad request", statusCode: 400, expectedCode: "INVALID_REQUEST", retryable: false},
		{name: "unauthorized", statusCode: 401, expectedCode: "AUTHENTICATION_FAILED", retryable: false},
		{name: "forbidden", statusCode: 403, expectedCode: "PERMISSION_DENIED", retryable: false},
		{name: "not found", statusCode: 404, expectedCode: "NOT_FOUND", retryable: false},
		{name: "rate limit", statusCode: 429, expectedCode: "RATE_LIMIT_EXCEEDED", retryable: true},
		{name: "server error", statusCode: 500, expectedCode: "SERVER_ERROR", retryable: true},
		{name: "bad gateway", statusCode: 502, expectedCode: "SERVER_ERROR", retryable: true},
		{name: "service unavailable", statusCode: 503, expectedCode: "SERVER_ERROR", retryable: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode, Header: http.Header{}}
			result := classifier.ClassifyError(resp, nil)
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.Equal(t, tt.retryable, result.Retryable)
		})
	}
}

func TestErrorClassifier_ClassifyAnthropicErrorAdvanced(t *testing.T) {
	classifier := NewErrorClassifier("anthropic")

	tests := []struct {
		name         string
		statusCode   int
		expectedCode string
	}{
		{name: "bad request", statusCode: 400, expectedCode: "INVALID_REQUEST"},
		{name: "overloaded", statusCode: 529, expectedCode: "OVERLOADED"},
		{name: "rate limit", statusCode: 429, expectedCode: "RATE_LIMIT_EXCEEDED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode, Header: http.Header{}}
			result := classifier.ClassifyError(resp, nil)
			assert.Equal(t, tt.expectedCode, result.Code)
		})
	}
}

func TestErrorClassifier_ClassifyGoogleErrorAdvanced(t *testing.T) {
	classifier := NewErrorClassifier("google")

	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "bad request", statusCode: 400},
		{name: "unauthorized", statusCode: 401},
		{name: "rate limit", statusCode: 429},
		{name: "server error", statusCode: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode, Header: http.Header{}}
			result := classifier.ClassifyError(resp, nil)
			assert.NotEmpty(t, result.Code)
		})
	}
}

func TestErrorClassifier_ClassifyDeepSeekErrorAdvanced(t *testing.T) {
	classifier := NewErrorClassifier("deepseek")

	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "bad request", statusCode: 400},
		{name: "unauthorized", statusCode: 401},
		{name: "rate limit", statusCode: 429},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode, Header: http.Header{}}
			result := classifier.ClassifyError(resp, nil)
			assert.NotEmpty(t, result.Code)
		})
	}
}

func TestErrorClassifier_ClassifyGenericErrorAdvanced(t *testing.T) {
	classifier := NewErrorClassifier("unknown_provider")

	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "bad request", statusCode: 400},
		{name: "server error", statusCode: 500},
		{name: "service unavailable", statusCode: 503},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode, Header: http.Header{}}
			result := classifier.ClassifyError(resp, nil)
			assert.NotEmpty(t, result.Code)
		})
	}
}

func TestErrorClassifier_NilResponseAdvanced(t *testing.T) {
	classifier := NewErrorClassifier("openai")
	result := classifier.ClassifyError(nil, nil)
	assert.Equal(t, ErrorTypeNetwork, result.Type)
	assert.Equal(t, "NETWORK_ERROR", result.Code)
}

func TestErrorClassifier_RateLimitWithRetryAfterAdvanced(t *testing.T) {
	classifier := NewErrorClassifier("openai")

	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"60"}},
	}
	result := classifier.ClassifyError(resp, nil)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", result.Code)
	assert.Equal(t, 60*time.Second, result.RetryAfter)
}

// ==================== Error Handler Tests ====================

func TestErrorHandler_HandleErrorAdvanced(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		RetryableErrors: []string{"RATE_LIMIT", "SERVER_ERROR", "429", "500"},
	}
	handler := NewErrorHandler("openai", config)

	// Rate limit should be retryable
	resp := &http.Response{StatusCode: 429, Header: http.Header{}}
	provErr, shouldRetry, _ := handler.HandleError(resp, nil, 0)
	assert.NotNil(t, provErr)
	assert.True(t, shouldRetry)

	// Auth error should not be retryable
	resp = &http.Response{StatusCode: 401, Header: http.Header{}}
	provErr, shouldRetry, _ = handler.HandleError(resp, nil, 0)
	assert.NotNil(t, provErr)
	assert.False(t, shouldRetry)
}

func TestErrorHandler_HandleError_MaxRetriesAdvanced(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		RetryableErrors: []string{"429"},
	}
	handler := NewErrorHandler("openai", config)

	resp := &http.Response{StatusCode: 429, Header: http.Header{}}
	// At attempt 3 (which is max), should not retry
	_, shouldRetry, _ := handler.HandleError(resp, nil, 3)
	assert.False(t, shouldRetry)
}

func TestCalculateRetryDelay_WithRetryAfterAdvanced(t *testing.T) {
	config := RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
	}
	handler := NewErrorHandler("openai", config)

	// With Retry-After, should use that value
	delay := handler.calculateRetryDelay(0, 30*time.Second)
	assert.Equal(t, 30*time.Second, delay)
}

func TestCalculateRetryDelay_ExponentialBackoffAdvanced(t *testing.T) {
	config := RetryConfig{
		MaxRetries:    5,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
	}
	handler := NewErrorHandler("openai", config)

	delay0 := handler.calculateRetryDelay(0, 0)
	delay1 := handler.calculateRetryDelay(1, 0)
	delay2 := handler.calculateRetryDelay(2, 0)

	assert.Equal(t, 100*time.Millisecond, delay0)
	assert.Equal(t, 200*time.Millisecond, delay1)
	assert.Equal(t, 400*time.Millisecond, delay2)
}

func TestCalculateRetryDelay_MaxDelayCapAdvanced(t *testing.T) {
	config := RetryConfig{
		MaxRetries:    10,
		InitialDelay:  1 * time.Second,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 3.0,
	}
	handler := NewErrorHandler("openai", config)

	delay := handler.calculateRetryDelay(10, 0) // Would be 1s * 3^10 without cap
	assert.Equal(t, 5*time.Second, delay)
}

func TestIsRetryableErrorAdvanced(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		RetryableErrors: []string{"RATE_LIMIT", "SERVER_ERROR"},
	}
	handler := NewErrorHandler("openai", config)

	retryable := handler.isRetryableError(&ProviderError{
		Code:       "RATE_LIMIT_EXCEEDED",
		HTTPStatus: 429,
	})
	assert.True(t, retryable)

	notRetryable := handler.isRetryableError(&ProviderError{
		Code:       "AUTHENTICATION_FAILED",
		HTTPStatus: 401,
	})
	assert.False(t, notRetryable)
}

func TestPowAdvanced(t *testing.T) {
	assert.Equal(t, 1.0, pow(2.0, 0))
	assert.Equal(t, 2.0, pow(2.0, 1))
	assert.Equal(t, 4.0, pow(2.0, 2))
	assert.Equal(t, 8.0, pow(2.0, 3))
	assert.Equal(t, 27.0, pow(3.0, 3))
}

// ==================== Recovery Strategies Tests ====================

func TestExecuteWithRetry_SuccessAdvanced(t *testing.T) {
	config := RetryConfig{MaxRetries: 3, InitialDelay: 10 * time.Millisecond}
	strategies := NewRecoveryStrategies("openai", config)

	callCount := 0
	fn := func() (*http.Response, []byte, error) {
		callCount++
		return &http.Response{StatusCode: 200}, []byte("success"), nil
	}

	resp, body, err := strategies.ExecuteWithRetry(fn)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, []byte("success"), body)
	assert.Equal(t, 1, callCount)
}

func TestExecuteWithRetry_RetryOnServerErrorAdvanced(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Millisecond,
		BackoffFactor:   1.0,
		RetryableErrors: []string{"SERVER_ERROR", "500"},
	}
	strategies := NewRecoveryStrategies("openai", config)

	callCount := 0
	fn := func() (*http.Response, []byte, error) {
		callCount++
		if callCount < 3 {
			return &http.Response{StatusCode: 500, Header: http.Header{}}, nil, nil
		}
		return &http.Response{StatusCode: 200}, []byte("success"), nil
	}

	resp, _, _ := strategies.ExecuteWithRetry(fn)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 3, callCount)
}

// ==================== Fallback Models Tests ====================

func TestGetFallbackModels_OpenAI(t *testing.T) {
	models := GetFallbackModels("openai")
	require.NotNil(t, models)
	assert.NotEmpty(t, models)

	for _, model := range models {
		assert.NotEmpty(t, model.ID, "Model ID should not be empty")
		assert.NotEmpty(t, model.Name, "Model name should not be empty")
		assert.Equal(t, "openai", model.ProviderID)
	}
}

func TestGetFallbackModels_Anthropic(t *testing.T) {
	models := GetFallbackModels("anthropic")
	require.NotNil(t, models)
	assert.NotEmpty(t, models)

	foundClaude := false
	for _, model := range models {
		if model.ID == "claude-3-5-sonnet" || model.ID == "claude-3-haiku" {
			foundClaude = true
		}
	}
	assert.True(t, foundClaude, "Should have Claude models")
}

func TestGetFallbackModels_Unknown(t *testing.T) {
	models := GetFallbackModels("unknown_provider")
	require.NotNil(t, models)
	assert.Len(t, models, 1)
	assert.Equal(t, "unknown_provider-model", models[0].ID)
}

func TestGetFallbackModels_AllProviders(t *testing.T) {
	providers := []string{"openai", "anthropic", "groq", "gemini", "deepseek", "mistral"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			models := GetFallbackModels(provider)
			assert.NotEmpty(t, models)
		})
	}
}

// ==================== Provider Error Tests ====================

func TestProviderError_ErrorAdvanced(t *testing.T) {
	err := &ProviderError{
		Provider: "openai",
		Code:     "RATE_LIMIT_EXCEEDED",
		Message:  "Too many requests",
	}

	errorStr := err.Error()
	assert.Contains(t, errorStr, "openai")
	assert.Contains(t, errorStr, "RATE_LIMIT_EXCEEDED")
	assert.Contains(t, errorStr, "Too many requests")
}

func TestProviderError_EmptyFieldsAdvanced(t *testing.T) {
	err := &ProviderError{}
	errorStr := err.Error()
	assert.NotEmpty(t, errorStr)
}

// ==================== Mock Server Tests ====================

func TestErrorClassifier_WithMockServerAdvanced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("Retry-After", "30")
		w.Write([]byte(`{"error": "rate limit exceeded"}`))
	}))
	defer server.Close()

	classifier := NewErrorClassifier("openai")

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	result := classifier.ClassifyError(resp, nil)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", result.Code)
	assert.True(t, result.Retryable)
}

func TestErrorHandler_WithMockServerAdvanced(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	config := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    1 * time.Millisecond,
		BackoffFactor:   1.0,
		RetryableErrors: []string{"SERVER_ERROR", "500"},
	}
	strategies := NewRecoveryStrategies("openai", config)

	fn := func() (*http.Response, []byte, error) {
		resp, err := http.Get(server.URL)
		if err != nil {
			return nil, nil, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, body, nil
	}

	resp, _, err := strategies.ExecuteWithRetry(fn)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 3, callCount)
}

// ==================== RetryConfig Tests ====================

func TestRetryConfig_DefaultsAdvanced(t *testing.T) {
	config := RetryConfig{}
	// Zero values should work
	handler := NewErrorHandler("openai", config)
	assert.NotNil(t, handler)
}

func TestRetryConfig_WithAllFields(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.5,
		RetryableErrors: []string{"RATE_LIMIT", "SERVER_ERROR", "TIMEOUT"},
	}

	handler := NewErrorHandler("anthropic", config)
	assert.NotNil(t, handler)
	assert.Equal(t, 5, handler.retryConfig.MaxRetries)
	assert.Equal(t, 2.5, handler.retryConfig.BackoffFactor)
}

// ==================== Recovery Strategies Tests ====================

// MockCircuitBreaker implements CircuitBreaker for testing
type MockCircuitBreaker struct {
	ShouldFail bool
	CallCount  int
}

func (m *MockCircuitBreaker) Call(fn func() error) error {
	m.CallCount++
	if m.ShouldFail {
		return fmt.Errorf("circuit breaker open")
	}
	return fn()
}

func TestRecoveryStrategies_CircuitBreakerRecovery_Success(t *testing.T) {
	rs := &RecoveryStrategies{}
	cb := &MockCircuitBreaker{ShouldFail: false}

	successResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("success")),
	}

	fn := func() (*http.Response, []byte, error) {
		return successResp, []byte("success"), nil
	}

	resp, body, err := rs.CircuitBreakerRecovery(cb, fn)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, []byte("success"), body)
	assert.Equal(t, 1, cb.CallCount)
}

func TestRecoveryStrategies_CircuitBreakerRecovery_CircuitOpen(t *testing.T) {
	rs := &RecoveryStrategies{}
	cb := &MockCircuitBreaker{ShouldFail: true}

	fn := func() (*http.Response, []byte, error) {
		return &http.Response{StatusCode: 200}, nil, nil
	}

	resp, body, err := rs.CircuitBreakerRecovery(cb, fn)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker open")
	assert.Nil(t, resp)
	assert.Nil(t, body)
}

func TestRecoveryStrategies_CircuitBreakerRecovery_ServerError(t *testing.T) {
	rs := &RecoveryStrategies{}
	cb := &MockCircuitBreaker{ShouldFail: false}

	serverErrorResp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader("server error")),
	}

	fn := func() (*http.Response, []byte, error) {
		return serverErrorResp, []byte("server error"), nil
	}

	resp, body, err := rs.CircuitBreakerRecovery(cb, fn)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server error: 500")
	assert.Nil(t, resp)
	assert.Nil(t, body)
}

func TestRecoveryStrategies_CircuitBreakerRecovery_FunctionError(t *testing.T) {
	rs := &RecoveryStrategies{}
	cb := &MockCircuitBreaker{ShouldFail: false}

	fn := func() (*http.Response, []byte, error) {
		return nil, nil, fmt.Errorf("connection refused")
	}

	resp, body, err := rs.CircuitBreakerRecovery(cb, fn)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
	assert.Nil(t, resp)
	assert.Nil(t, body)
}

func TestRecoveryStrategies_FallbackRecovery_FirstEndpointSuccess(t *testing.T) {
	rs := &RecoveryStrategies{}
	endpoints := []string{"http://primary.api", "http://backup.api"}

	callCount := 0
	fn := func(endpoint string) (*http.Response, []byte, error) {
		callCount++
		return &http.Response{StatusCode: 200}, []byte(endpoint), nil
	}

	resp, body, err := rs.FallbackRecovery(endpoints, fn)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, []byte("http://primary.api"), body)
	assert.Equal(t, 1, callCount)
}

func TestRecoveryStrategies_FallbackRecovery_FallbackToSecond(t *testing.T) {
	rs := &RecoveryStrategies{}
	endpoints := []string{"http://primary.api", "http://backup.api"}

	callCount := 0
	fn := func(endpoint string) (*http.Response, []byte, error) {
		callCount++
		if endpoint == "http://primary.api" {
			return &http.Response{StatusCode: 500}, nil, nil
		}
		return &http.Response{StatusCode: 200}, []byte(endpoint), nil
	}

	resp, body, err := rs.FallbackRecovery(endpoints, fn)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, []byte("http://backup.api"), body)
	assert.Equal(t, 2, callCount)
}

func TestRecoveryStrategies_FallbackRecovery_AllEndpointsFail(t *testing.T) {
	rs := &RecoveryStrategies{}
	endpoints := []string{"http://primary.api", "http://backup.api", "http://tertiary.api"}

	fn := func(endpoint string) (*http.Response, []byte, error) {
		return nil, nil, fmt.Errorf("connection refused to %s", endpoint)
	}

	resp, body, err := rs.FallbackRecovery(endpoints, fn)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestRecoveryStrategies_FallbackRecovery_EmptyEndpoints(t *testing.T) {
	rs := &RecoveryStrategies{}
	endpoints := []string{}

	fn := func(endpoint string) (*http.Response, []byte, error) {
		return &http.Response{StatusCode: 200}, nil, nil
	}

	resp, body, err := rs.FallbackRecovery(endpoints, fn)

	// With empty endpoints, lastErr is nil
	assert.NoError(t, err)
	assert.Nil(t, resp)
	assert.Nil(t, body)
}
