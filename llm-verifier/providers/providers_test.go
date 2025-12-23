package providers

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBaseAdapter(t *testing.T) {
	adapter := &BaseAdapter{
		endpoint: "https://api.openai.com/v1",
		apiKey:   "test-key",
		headers:  map[string]string{"X-Custom": "value"},
	}
	assert.NotNil(t, adapter)
}

func TestBaseAdapterSetClient(t *testing.T) {
	adapter := &BaseAdapter{}
	client := &http.Client{Timeout: 30 * time.Second}
	adapter.SetClient(client)
	assert.Equal(t, client, adapter.client)
}

func TestBaseAdapterGetClient(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}
	adapter := &BaseAdapter{client: client}
	result := adapter.GetClient()
	assert.Equal(t, client, result)
}

func TestBaseAdapterSetEndpoint(t *testing.T) {
	adapter := &BaseAdapter{}
	adapter.SetEndpoint("https://api.anthropic.com/v1")
	assert.Equal(t, "https://api.anthropic.com/v1", adapter.endpoint)
}

func TestBaseAdapterGetEndpoint(t *testing.T) {
	adapter := &BaseAdapter{endpoint: "https://api.test.com/v1"}
	result := adapter.GetEndpoint()
	assert.Equal(t, "https://api.test.com/v1", result)
}

func TestBaseAdapterSetAPIKey(t *testing.T) {
	adapter := &BaseAdapter{}
	adapter.SetAPIKey("new-api-key")
	assert.Equal(t, "new-api-key", adapter.apiKey)
}

func TestBaseAdapterGetAPIKey(t *testing.T) {
	adapter := &BaseAdapter{apiKey: "test-api-key"}
	result := adapter.GetAPIKey()
	assert.Equal(t, "test-api-key", result)
}

func TestBaseAdapterSetHeaders(t *testing.T) {
	adapter := &BaseAdapter{}
	headers := map[string]string{"X-A": "1", "X-B": "2"}
	adapter.SetHeaders(headers)
	assert.Equal(t, headers, adapter.headers)
}

func TestBaseAdapterGetHeaders(t *testing.T) {
	headers := map[string]string{"X-Auth": "token"}
	adapter := &BaseAdapter{headers: headers}
	result := adapter.GetHeaders()
	assert.Equal(t, headers, result)
}

func TestBaseAdapterAddHeader(t *testing.T) {
	adapter := &BaseAdapter{}
	adapter.AddHeader("X-Custom", "value")
	assert.NotNil(t, adapter.headers)
	assert.Equal(t, "value", adapter.headers["X-Custom"])
}

func TestBaseAdapterAddHeaderExisting(t *testing.T) {
	adapter := &BaseAdapter{headers: map[string]string{"X-Test": "old"}}
	adapter.AddHeader("X-Test", "new")
	assert.Equal(t, "new", adapter.headers["X-Test"])
}

func TestProviderError(t *testing.T) {
	err := &ProviderError{
		Provider:   "openai",
		Type:       ErrorTypeRateLimit,
		Code:       "RATE_LIMIT_EXCEEDED",
		Message:    "Rate limit exceeded",
		HTTPStatus: 429,
		Retryable:  true,
	}
	result := err.Error()
	assert.Contains(t, result, "openai")
	assert.Contains(t, result, "RATE_LIMIT_EXCEEDED")
	assert.Contains(t, result, "Rate limit exceeded")
}

func TestErrorTypeConstants(t *testing.T) {
	assert.Equal(t, ErrorType(0), ErrorTypeNetwork)
	assert.Equal(t, ErrorType(1), ErrorTypeAuth)
	assert.Equal(t, ErrorType(2), ErrorTypeRateLimit)
	assert.Equal(t, ErrorType(3), ErrorTypeQuota)
	assert.Equal(t, ErrorType(4), ErrorTypeInvalidRequest)
	assert.Equal(t, ErrorType(5), ErrorTypeServer)
	assert.Equal(t, ErrorType(6), ErrorTypeTimeout)
	assert.Equal(t, ErrorType(7), ErrorTypeUnknown)
}

func TestNewErrorClassifier(t *testing.T) {
	classifier := NewErrorClassifier("openai")
	assert.NotNil(t, classifier)
	assert.Equal(t, "openai", classifier.provider)
}

func TestErrorClassifierClassifyErrorNilResponse(t *testing.T) {
	classifier := NewErrorClassifier("test")
	err := classifier.ClassifyError(nil, []byte{})
	assert.Error(t, err)
	assert.Equal(t, ErrorTypeNetwork, err.Type)
	assert.Equal(t, "NETWORK_ERROR", err.Code)
	assert.True(t, err.Retryable)
}

func TestErrorClassifierClassifyErrorOpenAI(t *testing.T) {
	classifier := NewErrorClassifier("openai")
	resp := &http.Response{StatusCode: 429}
	err := classifier.ClassifyError(resp, []byte{})
	assert.Equal(t, "openai", err.Provider)
	assert.Equal(t, ErrorTypeRateLimit, err.Type)
	assert.True(t, err.Retryable)
}

func TestErrorClassifierClassifyErrorAnthropic(t *testing.T) {
	classifier := NewErrorClassifier("anthropic")
	resp := &http.Response{StatusCode: 401}
	err := classifier.ClassifyError(resp, []byte{})
	assert.Equal(t, "anthropic", err.Provider)
	assert.Equal(t, ErrorTypeAuth, err.Type)
	assert.False(t, err.Retryable)
}

func TestErrorClassifierClassifyErrorDeepSeek(t *testing.T) {
	classifier := NewErrorClassifier("deepseek")
	resp := &http.Response{StatusCode: 500}
	err := classifier.ClassifyError(resp, []byte{})
	assert.Equal(t, "deepseek", err.Provider)
	assert.Equal(t, ErrorTypeServer, err.Type)
	assert.True(t, err.Retryable)
}

func TestErrorClassifierClassifyErrorGoogle(t *testing.T) {
	classifier := NewErrorClassifier("google")
	resp := &http.Response{StatusCode: 404}
	err := classifier.ClassifyError(resp, []byte{})
	assert.Equal(t, "google", err.Provider)
	assert.Equal(t, ErrorTypeInvalidRequest, err.Type)
	assert.False(t, err.Retryable)
}

func TestErrorClassifierClassifyErrorGeneric(t *testing.T) {
	classifier := NewErrorClassifier("unknown")
	resp := &http.Response{StatusCode: 503}
	err := classifier.ClassifyError(resp, []byte{})
	assert.Equal(t, "unknown", err.Provider)
	assert.Equal(t, ErrorTypeServer, err.Type)
	assert.True(t, err.Retryable)
}

func TestErrorClassifierClassifyErrorWithRetryAfter(t *testing.T) {
	classifier := NewErrorClassifier("openai")
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"5"}},
	}
	err := classifier.ClassifyError(resp, []byte{})
	assert.Equal(t, 5*time.Second, err.RetryAfter)
}

func TestClassifyOpenAIError(t *testing.T) {
	classifier := NewErrorClassifier("openai")
	code, _, retryable := classifier.classifyOpenAIError(400, nil)
	assert.Equal(t, "INVALID_REQUEST", code)
	assert.False(t, retryable)
	code, _, retryable = classifier.classifyOpenAIError(401, nil)
	assert.Equal(t, "AUTHENTICATION_FAILED", code)
	assert.False(t, retryable)
	code, _, retryable = classifier.classifyOpenAIError(429, nil)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", code)
	assert.True(t, retryable)
	code, _, retryable = classifier.classifyOpenAIError(500, nil)
	assert.Equal(t, "SERVER_ERROR", code)
	assert.True(t, retryable)
}

func TestClassifyAnthropicError(t *testing.T) {
	classifier := NewErrorClassifier("anthropic")
	code, _, retryable := classifier.classifyAnthropicError(400, nil)
	assert.Equal(t, "INVALID_REQUEST", code)
	assert.False(t, retryable)
	code, _, retryable = classifier.classifyAnthropicError(429, nil)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", code)
	assert.True(t, retryable)
	code, _, retryable = classifier.classifyAnthropicError(529, nil)
	assert.Equal(t, "OVERLOADED", code)
	assert.True(t, retryable)
}

func TestClassifyDeepSeekError(t *testing.T) {
	classifier := NewErrorClassifier("deepseek")
	code, _, retryable := classifier.classifyDeepSeekError(403, nil)
	assert.Equal(t, "PERMISSION_DENIED", code)
	assert.False(t, retryable)
	code, _, retryable = classifier.classifyDeepSeekError(500, nil)
	assert.Equal(t, "SERVER_ERROR", code)
	assert.True(t, retryable)
}

func TestClassifyGoogleError(t *testing.T) {
	classifier := NewErrorClassifier("google")
	code, _, retryable := classifier.classifyGoogleError(401, nil)
	assert.Equal(t, "AUTHENTICATION_FAILED", code)
	assert.False(t, retryable)
	code, _, retryable = classifier.classifyGoogleError(429, nil)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", code)
	assert.True(t, retryable)
}

func TestClassifyGenericError(t *testing.T) {
	classifier := NewErrorClassifier("generic")
	code, _, retryable := classifier.classifyGenericError(422, nil)
	assert.Equal(t, "INVALID_REQUEST", code)
	assert.False(t, retryable)
	code, _, retryable = classifier.classifyGenericError(408, nil)
	assert.Equal(t, "TIMEOUT", code)
	assert.True(t, retryable)
}

func TestGetErrorTypeServer(t *testing.T) {
	classifier := NewErrorClassifier("test")
	errType := classifier.getErrorType(500, "")
	assert.Equal(t, ErrorTypeServer, errType)
	errType = classifier.getErrorType(503, "")
	assert.Equal(t, ErrorTypeServer, errType)
}

func TestGetErrorTypeTimeout(t *testing.T) {
	classifier := NewErrorClassifier("test")
	errType := classifier.getErrorType(408, "")
	assert.Equal(t, ErrorTypeTimeout, errType)
	errType = classifier.getErrorType(400, "TIMEOUT_ERROR")
	assert.Equal(t, ErrorTypeTimeout, errType)
}

func TestGetErrorTypeAuth(t *testing.T) {
	classifier := NewErrorClassifier("test")
	errType := classifier.getErrorType(401, "")
	assert.Equal(t, ErrorTypeAuth, errType)
	errType = classifier.getErrorType(403, "")
	assert.Equal(t, ErrorTypeAuth, errType)
	errType = classifier.getErrorType(400, "AUTH_FAILED")
	assert.Equal(t, ErrorTypeAuth, errType)
}

func TestGetErrorTypeRateLimit(t *testing.T) {
	classifier := NewErrorClassifier("test")
	errType := classifier.getErrorType(429, "")
	assert.Equal(t, ErrorTypeRateLimit, errType)
	errType = classifier.getErrorType(400, "RATE_LIMIT_EXCEEDED")
	assert.Equal(t, ErrorTypeRateLimit, errType)
}

func TestGetErrorTypeQuota(t *testing.T) {
	classifier := NewErrorClassifier("test")
	errType := classifier.getErrorType(400, "QUOTA_EXCEEDED")
	assert.Equal(t, ErrorTypeQuota, errType)
}

func TestGetErrorTypeInvalidRequest(t *testing.T) {
	classifier := NewErrorClassifier("test")
	errType := classifier.getErrorType(404, "")
	assert.Equal(t, ErrorTypeInvalidRequest, errType)
	errType = classifier.getErrorType(400, "")
	assert.Equal(t, ErrorTypeInvalidRequest, errType)
}

func TestNewErrorHandler(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"429", "500"},
	}
	handler := NewErrorHandler("openai", retryConfig)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.classifier)
	assert.Equal(t, retryConfig, handler.retryConfig)
}

func TestErrorHandlerHandleError(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"429"},
	}
	handler := NewErrorHandler("openai", retryConfig)
	resp := &http.Response{StatusCode: 429}
	err, shouldRetry, delay := handler.HandleError(resp, []byte{}, 0)
	assert.Error(t, err)
	assert.True(t, shouldRetry)
	assert.Greater(t, delay, time.Duration(0))
}

func TestErrorHandlerHandleErrorNotRetryable(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"500"},
	}
	handler := NewErrorHandler("openai", retryConfig)
	resp := &http.Response{StatusCode: 401}
	err, shouldRetry, delay := handler.HandleError(resp, []byte{}, 0)
	assert.Error(t, err)
	assert.False(t, shouldRetry)
	assert.Equal(t, time.Duration(0), delay)
}

func TestNewRecoveryStrategies(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
	}
	strategies := NewRecoveryStrategies("openai", retryConfig)
	assert.NotNil(t, strategies)
	assert.NotNil(t, strategies.handler)
}

func TestRetryConfig(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    2 * time.Second,
		MaxDelay:        60 * time.Second,
		BackoffFactor:   3.0,
		RetryableErrors: []string{"429", "500", "503"},
	}
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 2*time.Second, config.InitialDelay)
	assert.Equal(t, 60*time.Second, config.MaxDelay)
	assert.Equal(t, 3.0, config.BackoffFactor)
	assert.Len(t, config.RetryableErrors, 3)
}

func TestRateLimitConfig(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerMinute: 60,
		RequestsPerHour:   1000,
		BurstLimit:        10,
	}
	assert.Equal(t, 60, config.RequestsPerMinute)
	assert.Equal(t, 1000, config.RequestsPerHour)
	assert.Equal(t, 10, config.BurstLimit)
}

func TestTimeoutConfig(t *testing.T) {
	config := TimeoutConfig{
		RequestTimeout: 30 * time.Second,
		StreamTimeout:  300 * time.Second,
		ConnectTimeout: 10 * time.Second,
	}
	assert.Equal(t, 30*time.Second, config.RequestTimeout)
	assert.Equal(t, 300*time.Second, config.StreamTimeout)
	assert.Equal(t, 10*time.Second, config.ConnectTimeout)
}

func TestProviderConfig(t *testing.T) {
	config := ProviderConfig{
		Name:            "test",
		Endpoint:        "https://api.test.com/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "gpt-4",
	}
	assert.Equal(t, "test", config.Name)
	assert.Equal(t, "https://api.test.com/v1", config.Endpoint)
	assert.Equal(t, "bearer", config.AuthType)
	assert.Equal(t, "sse", config.StreamingFormat)
	assert.Equal(t, "gpt-4", config.DefaultModel)
}

func TestNewProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.providers)
}

func TestProviderRegistryGetConfig(t *testing.T) {
	registry := NewProviderRegistry()
	config, exists := registry.GetConfig("openai")
	assert.True(t, exists)
	assert.NotNil(t, config)
	assert.Equal(t, "openai", config.Name)
}

func TestProviderRegistryGetConfigNotFound(t *testing.T) {
	registry := NewProviderRegistry()
	config, exists := registry.GetConfig("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, config)
}

func TestProviderRegistryRegisterProvider(t *testing.T) {
	registry := NewProviderRegistry()
	customConfig := &ProviderConfig{
		Name:            "custom",
		Endpoint:        "https://api.custom.com/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "custom-model",
	}
	registry.RegisterProvider(customConfig)
	config, exists := registry.GetConfig("custom")
	assert.True(t, exists)
	assert.Equal(t, "custom", config.Name)
}

func TestProviderRegistryGetProviderNames(t *testing.T) {
	registry := NewProviderRegistry()
	names := registry.GetProviderNames()
	assert.NotEmpty(t, names)
	assert.Contains(t, names, "openai")
	assert.Contains(t, names, "anthropic")
	assert.Contains(t, names, "deepseek")
}

func TestProviderRegistryIsProviderSupported(t *testing.T) {
	registry := NewProviderRegistry()
	assert.True(t, registry.IsProviderSupported("openai"))
	assert.True(t, registry.IsProviderSupported("anthropic"))
	assert.False(t, registry.IsProviderSupported("nonexistent"))
}

func TestProviderRegistryGetDefaultConfig(t *testing.T) {
	registry := NewProviderRegistry()
	config := registry.GetDefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "generic", config.Name)
}

func TestPow(t *testing.T) {
	assert.Equal(t, 1.0, pow(2, 0))
	assert.Equal(t, 2.0, pow(2, 1))
	assert.Equal(t, 4.0, pow(2, 2))
	assert.Equal(t, 8.0, pow(2, 3))
	assert.Equal(t, 16.0, pow(2, 4))
	assert.Equal(t, 27.0, pow(3, 3))
	assert.Equal(t, 1.0, pow(10, 0))
}

func TestCalculateRetryDelayWithRetryAfter(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
	}
	handler := NewErrorHandler("openai", retryConfig)
	delay := handler.calculateRetryDelay(0, 5*time.Second)
	assert.Equal(t, 5*time.Second, delay)
}

func TestCalculateRetryDelayExponentialBackoff(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
	}
	handler := NewErrorHandler("openai", retryConfig)
	delay := handler.calculateRetryDelay(1, 0)
	assert.Equal(t, 2*time.Second, delay)
}

func TestCalculateRetryDelayMaxDelayCap(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      10,
		InitialDelay:    1 * time.Second,
		MaxDelay:        5 * time.Second,
		BackoffFactor:   3.0,
	}
	handler := NewErrorHandler("openai", retryConfig)
	delay := handler.calculateRetryDelay(10, 0)
	assert.Equal(t, 5*time.Second, delay)
}

func TestIsRetryableError(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		RetryableErrors: []string{"429", "500", "503"},
	}
	handler := NewErrorHandler("openai", retryConfig)
	err1 := &ProviderError{Code: "429"}
	assert.True(t, handler.isRetryableError(err1))
	err2 := &ProviderError{Code: "401"}
	assert.False(t, handler.isRetryableError(err2))
}

func TestIsRetryableErrorByStatusCode(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      3,
		RetryableErrors: []string{"429", "500"},
	}
	handler := NewErrorHandler("openai", retryConfig)
	err := &ProviderError{HTTPStatus: 429}
	assert.True(t, handler.isRetryableError(err))
}

func TestRetryAfterHeaderParsing(t *testing.T) {
	classifier := NewErrorClassifier("openai")
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"10"}},
	}
	err := classifier.ClassifyError(resp, []byte{})
	assert.Equal(t, 10*time.Second, err.RetryAfter)
}

func TestDefaultOpenAIConfig(t *testing.T) {
	registry := NewProviderRegistry()
	config, exists := registry.GetConfig("openai")
	assert.True(t, exists)
	assert.Equal(t, "openai", config.Name)
	assert.Equal(t, "bearer", config.AuthType)
	assert.Equal(t, "gpt-4", config.DefaultModel)
	assert.Equal(t, 60, config.RateLimits.RequestsPerMinute)
}

func TestDefaultDeepSeekConfig(t *testing.T) {
	registry := NewProviderRegistry()
	config, exists := registry.GetConfig("deepseek")
	assert.True(t, exists)
	assert.Equal(t, "deepseek", config.Name)
	assert.Equal(t, "bearer", config.AuthType)
	assert.Equal(t, "deepseek-chat", config.DefaultModel)
}

func TestDefaultAnthropicConfig(t *testing.T) {
	registry := NewProviderRegistry()
	config, exists := registry.GetConfig("anthropic")
	assert.True(t, exists)
	assert.Equal(t, "anthropic", config.Name)
	assert.Equal(t, "bearer", config.AuthType)
	assert.Contains(t, config.DefaultModel, "claude")
}

func TestDefaultGoogleConfig(t *testing.T) {
	registry := NewProviderRegistry()
	config, exists := registry.GetConfig("google")
	assert.True(t, exists)
	assert.Equal(t, "google", config.Name)
	assert.Equal(t, "api_key", config.AuthType)
	assert.Equal(t, "gemini-pro", config.DefaultModel)
}

func TestProviderFeatures(t *testing.T) {
	registry := NewProviderRegistry()
	openaiConfig, _ := registry.GetConfig("openai")
	assert.True(t, openaiConfig.Features["supports_streaming"].(bool))
	assert.True(t, openaiConfig.Features["supports_functions"].(bool))
	deepseekConfig, _ := registry.GetConfig("deepseek")
	assert.False(t, deepseekConfig.Features["supports_functions"].(bool))
}
