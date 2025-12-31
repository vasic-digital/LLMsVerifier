package providers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Extended Recovery Strategies Tests ====================

func TestRecoveryStrategies_ExecuteWithRetry_Success(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"429", "500"},
	}
	strategies := NewRecoveryStrategies("test", retryConfig)

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
	assert.Equal(t, 1, callCount) // Should only be called once on success
}

func TestRecoveryStrategies_ExecuteWithRetry_RetryThenSuccess(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"429", "500"},
	}
	strategies := NewRecoveryStrategies("test", retryConfig)

	callCount := 0
	fn := func() (*http.Response, []byte, error) {
		callCount++
		if callCount < 3 {
			return &http.Response{StatusCode: 429}, []byte("rate limit"), nil
		}
		return &http.Response{StatusCode: 200}, []byte("success"), nil
	}

	resp, body, err := strategies.ExecuteWithRetry(fn)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, []byte("success"), body)
	assert.Equal(t, 3, callCount) // Should be called 3 times
}

func TestRecoveryStrategies_ExecuteWithRetry_FunctionError(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"429"},
	}
	strategies := NewRecoveryStrategies("test", retryConfig)

	expectedErr := errors.New("network error")
	fn := func() (*http.Response, []byte, error) {
		return nil, nil, expectedErr
	}

	resp, body, err := strategies.ExecuteWithRetry(fn)

	assert.Nil(t, resp)
	assert.Nil(t, body)
	assert.Equal(t, expectedErr, err)
}

func TestRecoveryStrategies_FallbackRecovery_FirstSuccess(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   2.0,
	}
	strategies := NewRecoveryStrategies("test", retryConfig)

	endpoints := []string{
		"https://primary.api.com",
		"https://secondary.api.com",
		"https://tertiary.api.com",
	}

	calledEndpoint := ""
	fn := func(endpoint string) (*http.Response, []byte, error) {
		calledEndpoint = endpoint
		return &http.Response{StatusCode: 200}, []byte("success"), nil
	}

	resp, body, err := strategies.FallbackRecovery(endpoints, fn)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, []byte("success"), body)
	assert.Equal(t, "https://primary.api.com", calledEndpoint)
}

func TestRecoveryStrategies_FallbackRecovery_SecondEndpoint(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   2.0,
	}
	strategies := NewRecoveryStrategies("test", retryConfig)

	endpoints := []string{
		"https://primary.api.com",
		"https://secondary.api.com",
	}

	callCount := 0
	fn := func(endpoint string) (*http.Response, []byte, error) {
		callCount++
		if endpoint == "https://primary.api.com" {
			return &http.Response{StatusCode: 500}, []byte("error"), nil
		}
		return &http.Response{StatusCode: 200}, []byte("success"), nil
	}

	resp, body, err := strategies.FallbackRecovery(endpoints, fn)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, []byte("success"), body)
	assert.Equal(t, 2, callCount)
}

func TestRecoveryStrategies_FallbackRecovery_AllFail(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   2.0,
	}
	strategies := NewRecoveryStrategies("test", retryConfig)

	endpoints := []string{
		"https://primary.api.com",
		"https://secondary.api.com",
	}

	expectedErr := errors.New("all endpoints failed")
	fn := func(endpoint string) (*http.Response, []byte, error) {
		return nil, nil, expectedErr
	}

	resp, body, err := strategies.FallbackRecovery(endpoints, fn)

	assert.Nil(t, resp)
	assert.Nil(t, body)
	assert.Equal(t, expectedErr, err)
}

// ==================== Mock Circuit Breaker for Testing ====================

type mockCircuitBreaker struct {
	shouldFail bool
	callCount  int
}

func (m *mockCircuitBreaker) Call(fn func() error) error {
	m.callCount++
	if m.shouldFail {
		return errors.New("circuit breaker open")
	}
	return fn()
}

func TestRecoveryStrategies_CircuitBreakerRecovery_MockSuccess(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   2.0,
	}
	strategies := NewRecoveryStrategies("test", retryConfig)

	cb := &mockCircuitBreaker{shouldFail: false}

	fn := func() (*http.Response, []byte, error) {
		return &http.Response{StatusCode: 200}, []byte("success"), nil
	}

	resp, body, err := strategies.CircuitBreakerRecovery(cb, fn)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, []byte("success"), body)
}

func TestRecoveryStrategies_CircuitBreakerRecovery_MockCircuitOpen(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   2.0,
	}
	strategies := NewRecoveryStrategies("test", retryConfig)

	cb := &mockCircuitBreaker{shouldFail: true}

	fn := func() (*http.Response, []byte, error) {
		return &http.Response{StatusCode: 200}, []byte("success"), nil
	}

	resp, body, err := strategies.CircuitBreakerRecovery(cb, fn)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "circuit breaker open")
}

// ==================== Extended Error Classification Tests ====================

func TestErrorClassifier_AllOpenAIStatusCodes(t *testing.T) {
	classifier := NewErrorClassifier("openai")

	testCases := []struct {
		statusCode int
		expectCode string
		retryable  bool
	}{
		{400, "INVALID_REQUEST", false},
		{401, "AUTHENTICATION_FAILED", false},
		{403, "PERMISSION_DENIED", false},
		{404, "NOT_FOUND", false},
		{429, "RATE_LIMIT_EXCEEDED", true},
		{500, "SERVER_ERROR", true},
		{502, "SERVER_ERROR", true},
		{503, "SERVER_ERROR", true},
		{504, "SERVER_ERROR", true},
		{999, "UNKNOWN_ERROR", false},
	}

	for _, tc := range testCases {
		t.Run(string(rune(tc.statusCode)), func(t *testing.T) {
			code, _, retryable := classifier.classifyOpenAIError(tc.statusCode, nil)
			assert.Equal(t, tc.expectCode, code)
			assert.Equal(t, tc.retryable, retryable)
		})
	}
}

func TestErrorClassifier_AllAnthropicStatusCodes(t *testing.T) {
	classifier := NewErrorClassifier("anthropic")

	testCases := []struct {
		statusCode int
		expectCode string
		retryable  bool
	}{
		{400, "INVALID_REQUEST", false},
		{401, "AUTHENTICATION_FAILED", false},
		{403, "PERMISSION_DENIED", false},
		{404, "NOT_FOUND", false},
		{429, "RATE_LIMIT_EXCEEDED", true},
		{500, "SERVER_ERROR", true},
		{502, "SERVER_ERROR", true},
		{503, "SERVER_ERROR", true},
		{504, "SERVER_ERROR", true},
		{529, "OVERLOADED", true}, // Anthropic specific
		{999, "UNKNOWN_ERROR", false},
	}

	for _, tc := range testCases {
		t.Run(string(rune(tc.statusCode)), func(t *testing.T) {
			code, _, retryable := classifier.classifyAnthropicError(tc.statusCode, nil)
			assert.Equal(t, tc.expectCode, code)
			assert.Equal(t, tc.retryable, retryable)
		})
	}
}

func TestErrorClassifier_AllGenericStatusCodes(t *testing.T) {
	classifier := NewErrorClassifier("generic")

	testCases := []struct {
		statusCode int
		expectCode string
		retryable  bool
	}{
		{400, "INVALID_REQUEST", false},
		{401, "AUTHENTICATION_FAILED", false},
		{403, "AUTHENTICATION_FAILED", false},
		{404, "NOT_FOUND", false},
		{408, "TIMEOUT", true},
		{422, "INVALID_REQUEST", false},
		{429, "RATE_LIMIT_EXCEEDED", true},
		{500, "SERVER_ERROR", true},
		{502, "SERVER_ERROR", true},
		{503, "SERVER_ERROR", true},
		{504, "SERVER_ERROR", true},
		{999, "UNKNOWN_ERROR", false},
	}

	for _, tc := range testCases {
		t.Run(string(rune(tc.statusCode)), func(t *testing.T) {
			code, _, retryable := classifier.classifyGenericError(tc.statusCode, nil)
			assert.Equal(t, tc.expectCode, code)
			assert.Equal(t, tc.retryable, retryable)
		})
	}
}

func TestGetErrorType_AllCases(t *testing.T) {
	classifier := NewErrorClassifier("test")

	testCases := []struct {
		name       string
		statusCode int
		errorCode  string
		expected   ErrorType
	}{
		{"server 500", 500, "", ErrorTypeServer},
		{"server 503", 503, "", ErrorTypeServer},
		{"timeout 408", 408, "", ErrorTypeTimeout},
		{"timeout code", 200, "TIMEOUT_ERROR", ErrorTypeTimeout},
		{"auth 401", 401, "", ErrorTypeAuth},
		{"auth 403", 403, "", ErrorTypeAuth},
		{"auth code", 200, "AUTH_ERROR", ErrorTypeAuth},
		{"rate limit 429", 429, "", ErrorTypeRateLimit},
		{"rate limit code", 200, "RATE_LIMIT_EXCEEDED", ErrorTypeRateLimit},
		{"quota code", 200, "QUOTA_EXCEEDED", ErrorTypeQuota},
		{"invalid 400", 400, "", ErrorTypeInvalidRequest},
		{"invalid 404", 404, "", ErrorTypeInvalidRequest},
		{"success 200", 200, "", ErrorTypeUnknown}, // Should not happen for errors
		{"unknown 600", 600, "", ErrorTypeServer},  // 600 >= 500, treated as server error
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := classifier.getErrorType(tc.statusCode, tc.errorCode)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ==================== Provider Config Extended Tests ====================

func TestProviderRegistry_AllDefaultProviders(t *testing.T) {
	registry := NewProviderRegistry()

	expectedProviders := []string{
		"openai",
		"deepseek",
		"anthropic",
		"google",
		"groq",
		"togetherai",
		"generic",
	}

	for _, provider := range expectedProviders {
		t.Run(provider, func(t *testing.T) {
			config, exists := registry.GetConfig(provider)
			require.True(t, exists, "Provider %s should exist", provider)
			require.NotNil(t, config)
			assert.Equal(t, provider, config.Name)
			// Generic provider has no endpoint (it's a fallback for unknown providers)
			if provider != "generic" {
				assert.NotEmpty(t, config.Endpoint)
			}
			assert.NotEmpty(t, config.AuthType)
			assert.NotEmpty(t, config.StreamingFormat)
		})
	}
}

func TestProviderRegistry_GroqConfig(t *testing.T) {
	registry := NewProviderRegistry()
	config, exists := registry.GetConfig("groq")

	require.True(t, exists)
	assert.Equal(t, "groq", config.Name)
	assert.Equal(t, "https://api.groq.com/openai/v1", config.Endpoint)
	assert.Equal(t, "bearer", config.AuthType)
	assert.Equal(t, "llama3-8b-8192", config.DefaultModel)
	assert.True(t, config.Features["supports_streaming"].(bool))
	assert.False(t, config.Features["supports_vision"].(bool))
}

func TestProviderRegistry_TogetherAIConfig(t *testing.T) {
	registry := NewProviderRegistry()
	config, exists := registry.GetConfig("togetherai")

	require.True(t, exists)
	assert.Equal(t, "togetherai", config.Name)
	assert.Equal(t, "https://api.together.xyz/v1", config.Endpoint)
	assert.Equal(t, "bearer", config.AuthType)
	assert.Contains(t, config.DefaultModel, "llama")
	assert.True(t, config.Features["supports_streaming"].(bool))
}

func TestProviderRegistry_GenericConfig(t *testing.T) {
	registry := NewProviderRegistry()
	config, exists := registry.GetConfig("generic")

	require.True(t, exists)
	assert.Equal(t, "generic", config.Name)
	assert.Equal(t, "unknown", config.DefaultModel)
	assert.Equal(t, 30*time.Second, config.Timeouts.RequestTimeout)
	assert.Equal(t, 2, config.RetryConfig.MaxRetries)
}

func TestProviderConfig_RateLimits(t *testing.T) {
	registry := NewProviderRegistry()

	providers := []string{"openai", "anthropic", "deepseek", "google"}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			config, _ := registry.GetConfig(provider)
			assert.Greater(t, config.RateLimits.RequestsPerMinute, 0)
			assert.Greater(t, config.RateLimits.RequestsPerHour, 0)
			assert.Greater(t, config.RateLimits.BurstLimit, 0)
		})
	}
}

func TestProviderConfig_Timeouts(t *testing.T) {
	registry := NewProviderRegistry()

	providers := []string{"openai", "anthropic", "deepseek", "google"}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			config, _ := registry.GetConfig(provider)
			assert.Greater(t, config.Timeouts.RequestTimeout, time.Duration(0))
			assert.Greater(t, config.Timeouts.StreamTimeout, time.Duration(0))
			assert.Greater(t, config.Timeouts.ConnectTimeout, time.Duration(0))
		})
	}
}

func TestProviderConfig_RetryConfig(t *testing.T) {
	registry := NewProviderRegistry()

	providers := []string{"openai", "anthropic", "deepseek", "google"}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			config, _ := registry.GetConfig(provider)
			assert.Greater(t, config.RetryConfig.MaxRetries, 0)
			assert.Greater(t, config.RetryConfig.InitialDelay, time.Duration(0))
			assert.Greater(t, config.RetryConfig.MaxDelay, time.Duration(0))
			assert.Greater(t, config.RetryConfig.BackoffFactor, 0.0)
			assert.NotEmpty(t, config.RetryConfig.RetryableErrors)
		})
	}
}

// ==================== ProviderError Extended Tests ====================

func TestProviderError_AllFields(t *testing.T) {
	err := &ProviderError{
		Provider:    "openai",
		Type:        ErrorTypeRateLimit,
		Code:        "RATE_LIMIT_EXCEEDED",
		Message:     "You have exceeded your rate limit",
		HTTPStatus:  429,
		Retryable:   true,
		RetryAfter:  30 * time.Second,
		RawResponse: []byte(`{"error": "rate limit"}`),
	}

	assert.Equal(t, "openai", err.Provider)
	assert.Equal(t, ErrorTypeRateLimit, err.Type)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", err.Code)
	assert.Equal(t, "You have exceeded your rate limit", err.Message)
	assert.Equal(t, 429, err.HTTPStatus)
	assert.True(t, err.Retryable)
	assert.Equal(t, 30*time.Second, err.RetryAfter)
	assert.NotEmpty(t, err.RawResponse)

	// Test Error() method
	errStr := err.Error()
	assert.Contains(t, errStr, "openai")
	assert.Contains(t, errStr, "RATE_LIMIT_EXCEEDED")
	assert.Contains(t, errStr, "rate limit")
}

// ==================== BaseAdapter Extended Tests ====================

func TestBaseAdapter_AddHeaderInitializesMap(t *testing.T) {
	adapter := &BaseAdapter{} // headers is nil
	assert.Nil(t, adapter.headers)

	adapter.AddHeader("X-First", "value1")
	assert.NotNil(t, adapter.headers)
	assert.Equal(t, "value1", adapter.headers["X-First"])

	adapter.AddHeader("X-Second", "value2")
	assert.Equal(t, "value2", adapter.headers["X-Second"])
	assert.Len(t, adapter.headers, 2)
}

func TestBaseAdapter_FullInitialization(t *testing.T) {
	client := &http.Client{Timeout: 60 * time.Second}
	headers := map[string]string{
		"Authorization":  "Bearer token",
		"Content-Type":   "application/json",
		"X-Custom-Header": "custom-value",
	}

	adapter := &BaseAdapter{
		client:   client,
		endpoint: "https://api.example.com/v1",
		apiKey:   "sk-test-key-12345",
		headers:  headers,
	}

	assert.Equal(t, client, adapter.GetClient())
	assert.Equal(t, "https://api.example.com/v1", adapter.GetEndpoint())
	assert.Equal(t, "sk-test-key-12345", adapter.GetAPIKey())
	assert.Len(t, adapter.GetHeaders(), 3)
}
