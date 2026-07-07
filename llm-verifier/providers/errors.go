package providers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ProviderError represents a provider-specific error
type ProviderError struct {
	Provider    string
	Type        ErrorType
	Code        string
	Message     string
	HTTPStatus  int
	Retryable   bool
	RetryAfter  time.Duration
	RawResponse []byte
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s API error [%s]: %s", e.Provider, e.Code, e.Message)
}

// ErrorType categorizes different types of errors
type ErrorType int

const (
	ErrorTypeNetwork ErrorType = iota
	ErrorTypeAuth
	ErrorTypeRateLimit
	ErrorTypeQuota
	ErrorTypeInvalidRequest
	ErrorTypeServer
	ErrorTypeTimeout
	ErrorTypeUnknown
)

// ErrorClassifier analyzes HTTP responses and classifies errors
type ErrorClassifier struct {
	provider string
}

// NewErrorClassifier creates a new error classifier for a provider
func NewErrorClassifier(provider string) *ErrorClassifier {
	return &ErrorClassifier{provider: provider}
}

// ClassifyError analyzes an HTTP response and returns a classified error
func (ec *ErrorClassifier) ClassifyError(resp *http.Response, body []byte) *ProviderError {
	if resp == nil {
		return &ProviderError{
			Provider:  ec.provider,
			Type:      ErrorTypeNetwork,
			Code:      "NETWORK_ERROR",
			Message:   tr("llmsverifier_provider_err_network_connection_failed"),
			Retryable: true,
		}
	}

	// Extract error details based on provider
	var errorCode, errorMessage string
	var retryable bool
	var retryAfter time.Duration

	switch strings.ToLower(ec.provider) {
	case "openai":
		errorCode, errorMessage, retryable = ec.classifyOpenAIError(resp.StatusCode, body)
	case "anthropic":
		errorCode, errorMessage, retryable = ec.classifyAnthropicError(resp.StatusCode, body)
	case "deepseek":
		errorCode, errorMessage, retryable = ec.classifyDeepSeekError(resp.StatusCode, body)
	case "gemini", "google":
		errorCode, errorMessage, retryable = ec.classifyGoogleError(resp.StatusCode, body)
	default:
		errorCode, errorMessage, retryable = ec.classifyGenericError(resp.StatusCode, body)
	}

	// Check for rate limit headers
	if resp.StatusCode == 429 {
		if retryAfterHeader := resp.Header.Get("Retry-After"); retryAfterHeader != "" {
			if seconds, err := time.ParseDuration(retryAfterHeader + "s"); err == nil {
				retryAfter = seconds
			}
		}
	}

	// PWU-2 (§11.4.69 sink-side positive-evidence taxonomy): HTTP 402
	// (Payment Required) is a provider-agnostic quota/subscription-cap
	// signal. None of the per-provider classifiers above (nor
	// classifyGenericError) carry a 402 case, so without this override a
	// 402 falls through to "UNKNOWN_ERROR" -- the exact opaque-failed
	// collapse PWU-2 closes. Captured FACT (§11.4.6, live 2026-07-06):
	// qa-results/multitrack/logs/T4_iter2_20260706T223846Z.log --
	// `API Error: 402 ... {"detail":"Subscription usage cap exceeded.
	// Please add balance to continue."}`. A subscription/usage-cap
	// condition will not clear on a short retry, so it is never marked
	// Retryable (unlike 429/5xx).
	if resp.StatusCode == http.StatusPaymentRequired {
		errorCode = "QUOTA_EXCEEDED"
		errorMessage = tr("llmsverifier_provider_err_quota_exceeded")
		retryable = false
	}

	errorType := ec.getErrorType(resp.StatusCode, errorCode)

	return &ProviderError{
		Provider:    ec.provider,
		Type:        errorType,
		Code:        errorCode,
		Message:     errorMessage,
		HTTPStatus:  resp.StatusCode,
		Retryable:   retryable,
		RetryAfter:  retryAfter,
		RawResponse: body,
	}
}

// classifyOpenAIError classifies OpenAI-specific errors
func (ec *ErrorClassifier) classifyOpenAIError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400:
		return "INVALID_REQUEST", tr("llmsverifier_provider_err_invalid_request_params"), false
	case 401:
		return "AUTHENTICATION_FAILED", tr("llmsverifier_provider_err_invalid_api_key"), false
	case 403:
		return "PERMISSION_DENIED", tr("llmsverifier_provider_err_access_forbidden"), false
	case 404:
		return "NOT_FOUND", tr("llmsverifier_provider_err_resource_not_found"), false
	case 429:
		return "RATE_LIMIT_EXCEEDED", tr("llmsverifier_provider_err_rate_limit_exceeded"), true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", tr("llmsverifier_provider_err_internal_server"), true
	default:
		return "UNKNOWN_ERROR", tr("llmsverifier_provider_err_unknown_occurred"), false
	}
}

// classifyAnthropicError classifies Anthropic-specific errors
func (ec *ErrorClassifier) classifyAnthropicError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400:
		return "INVALID_REQUEST", tr("llmsverifier_provider_err_invalid_request_params"), false
	case 401:
		return "AUTHENTICATION_FAILED", tr("llmsverifier_provider_err_invalid_api_key"), false
	case 403:
		return "PERMISSION_DENIED", tr("llmsverifier_provider_err_access_forbidden"), false
	case 404:
		return "NOT_FOUND", tr("llmsverifier_provider_err_resource_not_found"), false
	case 429:
		return "RATE_LIMIT_EXCEEDED", tr("llmsverifier_provider_err_rate_limit_exceeded"), true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", tr("llmsverifier_provider_err_internal_server"), true
	case 529:
		return "OVERLOADED", tr("llmsverifier_provider_err_service_overloaded"), true
	default:
		return "UNKNOWN_ERROR", tr("llmsverifier_provider_err_unknown_occurred"), false
	}
}

// classifyDeepSeekError classifies DeepSeek-specific errors
func (ec *ErrorClassifier) classifyDeepSeekError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400:
		return "INVALID_REQUEST", tr("llmsverifier_provider_err_invalid_request_params"), false
	case 401:
		return "AUTHENTICATION_FAILED", tr("llmsverifier_provider_err_invalid_api_key"), false
	case 403:
		return "PERMISSION_DENIED", tr("llmsverifier_provider_err_access_forbidden"), false
	case 404:
		return "NOT_FOUND", tr("llmsverifier_provider_err_resource_not_found"), false
	case 429:
		return "RATE_LIMIT_EXCEEDED", tr("llmsverifier_provider_err_rate_limit_exceeded"), true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", tr("llmsverifier_provider_err_internal_server"), true
	default:
		return "UNKNOWN_ERROR", tr("llmsverifier_provider_err_unknown_occurred"), false
	}
}

// classifyGoogleError classifies Google AI-specific errors
func (ec *ErrorClassifier) classifyGoogleError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400:
		return "INVALID_REQUEST", tr("llmsverifier_provider_err_invalid_request_params"), false
	case 401:
		return "AUTHENTICATION_FAILED", tr("llmsverifier_provider_err_invalid_api_key"), false
	case 403:
		return "PERMISSION_DENIED", tr("llmsverifier_provider_err_access_forbidden"), false
	case 404:
		return "NOT_FOUND", tr("llmsverifier_provider_err_resource_not_found"), false
	case 429:
		return "RATE_LIMIT_EXCEEDED", tr("llmsverifier_provider_err_rate_limit_exceeded"), true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", tr("llmsverifier_provider_err_internal_server"), true
	default:
		return "UNKNOWN_ERROR", tr("llmsverifier_provider_err_unknown_occurred"), false
	}
}

// classifyGenericError provides generic error classification
func (ec *ErrorClassifier) classifyGenericError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400, 422:
		return "INVALID_REQUEST", tr("llmsverifier_provider_err_invalid_request"), false
	case 401, 403:
		return "AUTHENTICATION_FAILED", tr("llmsverifier_provider_err_authentication_failed"), false
	case 404:
		return "NOT_FOUND", tr("llmsverifier_provider_err_resource_not_found"), false
	case 408:
		return "TIMEOUT", tr("llmsverifier_provider_err_request_timeout"), true
	case 429:
		return "RATE_LIMIT_EXCEEDED", tr("llmsverifier_provider_err_rate_limit_exceeded"), true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", tr("llmsverifier_provider_err_server_error"), true
	default:
		return "UNKNOWN_ERROR", tr("llmsverifier_provider_err_unknown"), false
	}
}

// getErrorType maps HTTP status codes and error codes to error types
func (ec *ErrorClassifier) getErrorType(statusCode int, errorCode string) ErrorType {
	switch {
	case statusCode >= 500:
		return ErrorTypeServer
	case statusCode == 408 || strings.Contains(errorCode, "TIMEOUT"):
		return ErrorTypeTimeout
	case statusCode == 401 || statusCode == 403 || strings.Contains(errorCode, "AUTH"):
		return ErrorTypeAuth
	case statusCode == 429 || strings.Contains(errorCode, "RATE_LIMIT"):
		return ErrorTypeRateLimit
	case statusCode == 402 || strings.Contains(errorCode, "QUOTA"):
		return ErrorTypeQuota
	case statusCode >= 400 && statusCode < 500:
		return ErrorTypeInvalidRequest
	case statusCode >= 200 && statusCode < 300:
		return ErrorTypeUnknown // Should not happen for errors
	default:
		return ErrorTypeUnknown
	}
}

// ErrorHandler provides error handling and retry logic
type ErrorHandler struct {
	classifier  *ErrorClassifier
	retryConfig RetryConfig
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(provider string, retryConfig RetryConfig) *ErrorHandler {
	return &ErrorHandler{
		classifier:  NewErrorClassifier(provider),
		retryConfig: retryConfig,
	}
}

// HandleError processes an error and determines retry behavior
func (eh *ErrorHandler) HandleError(resp *http.Response, body []byte, attempt int) (*ProviderError, bool, time.Duration) {
	providerError := eh.classifier.ClassifyError(resp, body)

	// Check if we should retry
	shouldRetry := providerError.Retryable &&
		attempt < eh.retryConfig.MaxRetries &&
		eh.isRetryableError(providerError)

	// Calculate retry delay
	var retryDelay time.Duration
	if shouldRetry {
		retryDelay = eh.calculateRetryDelay(attempt, providerError.RetryAfter)
	}

	return providerError, shouldRetry, retryDelay
}

// isRetryableError checks if an error is retryable based on configuration
func (eh *ErrorHandler) isRetryableError(err *ProviderError) bool {
	errorCode := err.Code

	for _, retryableCode := range eh.retryConfig.RetryableErrors {
		if strings.Contains(errorCode, retryableCode) {
			return true
		}
		// Also check HTTP status as string
		if strings.Contains(fmt.Sprintf("%d", err.HTTPStatus), retryableCode) {
			return true
		}
	}

	return false
}

// calculateRetryDelay calculates the delay before retrying
func (eh *ErrorHandler) calculateRetryDelay(attempt int, retryAfter time.Duration) time.Duration {
	// If server specified Retry-After, use it
	if retryAfter > 0 {
		return retryAfter
	}

	// Exponential backoff
	delay := time.Duration(float64(eh.retryConfig.InitialDelay) * pow(eh.retryConfig.BackoffFactor, attempt))

	// Cap at max delay
	if delay > eh.retryConfig.MaxDelay {
		delay = eh.retryConfig.MaxDelay
	}

	return delay
}

// pow calculates base^exponent for float64
func pow(base float64, exp int) float64 {
	result := 1.0
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}

// RecoveryStrategies provides different recovery strategies
type RecoveryStrategies struct {
	handler *ErrorHandler
}

// NewRecoveryStrategies creates recovery strategies
func NewRecoveryStrategies(provider string, retryConfig RetryConfig) *RecoveryStrategies {
	return &RecoveryStrategies{
		handler: NewErrorHandler(provider, retryConfig),
	}
}

// ExecuteWithRetry executes a function with retry logic
func (rs *RecoveryStrategies) ExecuteWithRetry(fn func() (*http.Response, []byte, error)) (*http.Response, []byte, error) {
	var lastResp *http.Response
	var lastBody []byte
	var lastErr error

	for attempt := 0; attempt <= rs.handler.retryConfig.MaxRetries; attempt++ {
		resp, body, err := fn()

		if err != nil || resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, body, err
		}

		// Classify error and check if retryable
		providerError, shouldRetry, retryDelay := rs.handler.HandleError(resp, body, attempt)

		lastResp = resp
		lastBody = body
		lastErr = providerError

		if !shouldRetry {
			break
		}

		if retryDelay > 0 {
			time.Sleep(retryDelay)
		}
	}

	return lastResp, lastBody, lastErr
}

// CircuitBreakerRecovery provides circuit breaker-based recovery
func (rs *RecoveryStrategies) CircuitBreakerRecovery(cb CircuitBreaker, fn func() (*http.Response, []byte, error)) (*http.Response, []byte, error) {
	// Execute with circuit breaker protection
	err := cb.Call(func() error {
		resp, _, err := fn()
		if err != nil {
			return err
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// Get the actual response
	return fn()
}

// FallbackRecovery provides fallback to alternative endpoints
func (rs *RecoveryStrategies) FallbackRecovery(endpoints []string, fn func(endpoint string) (*http.Response, []byte, error)) (*http.Response, []byte, error) {
	var lastErr error

	for _, endpoint := range endpoints {
		resp, body, err := fn(endpoint)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, body, nil
		}
		lastErr = err
	}

	return nil, nil, lastErr
}

// CircuitBreaker interface for circuit breaker functionality
type CircuitBreaker interface {
	Call(func() error) error
}
