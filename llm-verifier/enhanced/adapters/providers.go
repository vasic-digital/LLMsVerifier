package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMRequest represents a request to an LLM
type LLMRequest struct {
	Prompt      string    `json:"prompt,omitempty"`
	Messages    []Message `json:"messages,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ProviderAdapter interface for provider-specific optimizations
type ProviderAdapter interface {
	// GetProviderName returns the provider name
	GetProviderName() string

	// OptimizeRequest optimizes the request for this provider
	OptimizeRequest(req *LLMRequest) *LLMRequest

	// ParseStreamingResponse parses streaming responses
	ParseStreamingResponse(reader io.Reader) <-chan StreamingChunk

	// HandleError handles provider-specific errors
	HandleError(resp *http.Response, body []byte) error

	// GetOptimalBatchSize returns the optimal batch size for this provider
	GetOptimalBatchSize() int

	// GetRateLimitInfo extracts rate limit information from headers
	GetRateLimitInfo(headers http.Header) RateLimitInfo
}

// StreamingChunk represents a chunk of streaming response
type StreamingChunk struct {
	Content  string                 `json:"content"`
	Finish   bool                   `json:"finish"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RateLimitInfo contains rate limiting information
type RateLimitInfo struct {
	RequestsPerMinute *int       `json:"requests_per_minute,omitempty"`
	TokensPerMinute   *int       `json:"tokens_per_minute,omitempty"`
	ResetTime         *time.Time `json:"reset_time,omitempty"`
}

// OpenAIAdapter provides OpenAI-specific optimizations
type OpenAIAdapter struct{}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter() *OpenAIAdapter {
	return &OpenAIAdapter{}
}

// GetProviderName returns the provider name
func (oa *OpenAIAdapter) GetProviderName() string {
	return "openai"
}

// OptimizeRequest optimizes requests for OpenAI
func (oa *OpenAIAdapter) OptimizeRequest(req *LLMRequest) *LLMRequest {
	optimized := *req // Copy the request

	// Optimize temperature for different tasks
	if strings.Contains(strings.ToLower(req.Prompt), "code") {
		if optimized.Temperature == nil || *optimized.Temperature > 0.3 {
			temp := 0.1
			optimized.Temperature = &temp
		}
	}

	// Ensure max_tokens is set appropriately
	if optimized.MaxTokens == nil {
		maxTokens := 2048
		optimized.MaxTokens = &maxTokens
	}

	// Add system message for better responses
	if len(optimized.Messages) > 0 && optimized.Messages[0].Role != "system" {
		systemMessage := Message{
			Role:    "system",
			Content: "You are a helpful AI assistant. Provide accurate and well-structured responses.",
		}
		optimized.Messages = append([]Message{systemMessage}, optimized.Messages...)
	}

	return &optimized
}

// ParseStreamingResponse parses OpenAI streaming responses
func (oa *OpenAIAdapter) ParseStreamingResponse(reader io.Reader) <-chan StreamingChunk {
	ch := make(chan StreamingChunk, 10)

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines and comments
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// OpenAI SSE format: "data: {...}"
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				if data == "[DONE]" {
					ch <- StreamingChunk{Finish: true}
					return
				}

				var event map[string]interface{}
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					ch <- StreamingChunk{Error: fmt.Sprintf("Failed to parse event: %v", err)}
					continue
				}

				// Extract content from choices
				if choices, ok := event["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if delta, ok := choice["delta"].(map[string]interface{}); ok {
							if content, ok := delta["content"].(string); ok {
								ch <- StreamingChunk{
									Content: content,
									Finish:  false,
								}
							}
						}

						// Check if this is the final message
						if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
							ch <- StreamingChunk{Finish: true}
							return
						}
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamingChunk{Error: fmt.Sprintf("Scanner error: %v", err)}
		}
	}()

	return ch
}

// HandleError handles OpenAI-specific errors
func (oa *OpenAIAdapter) HandleError(resp *http.Response, body []byte) error {
	if resp.StatusCode < 400 {
		return nil
	}

	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	switch resp.StatusCode {
	case 401:
		return fmt.Errorf("authentication failed: %s", errorResp.Error.Message)
	case 429:
		return fmt.Errorf("rate limit exceeded: %s", errorResp.Error.Message)
	case 500, 502, 503, 504:
		return fmt.Errorf("server error: %s", errorResp.Error.Message)
	default:
		return fmt.Errorf("API error (%s): %s", errorResp.Error.Type, errorResp.Error.Message)
	}
}

// GetOptimalBatchSize returns optimal batch size for OpenAI
func (oa *OpenAIAdapter) GetOptimalBatchSize() int {
	return 20 // OpenAI allows up to 50, but 20 is safer
}

// GetRateLimitInfo extracts rate limit info from OpenAI headers
func (oa *OpenAIAdapter) GetRateLimitInfo(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	if rpm := headers.Get("x-ratelimit-limit-requests"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil {
			info.RequestsPerMinute = &val
		}
	}

	if tpm := headers.Get("x-ratelimit-limit-tokens"); tpm != "" {
		if val, err := strconv.Atoi(tpm); err == nil {
			info.TokensPerMinute = &val
		}
	}

	if reset := headers.Get("x-ratelimit-reset-requests"); reset != "" {
		if timestamp, err := strconv.ParseInt(reset, 10, 64); err == nil {
			resetTime := time.Unix(timestamp, 0)
			info.ResetTime = &resetTime
		}
	}

	return info
}

// DeepSeekAdapter provides DeepSeek-specific optimizations
type DeepSeekAdapter struct{}

// NewDeepSeekAdapter creates a new DeepSeek adapter
func NewDeepSeekAdapter() *DeepSeekAdapter {
	return &DeepSeekAdapter{}
}

// GetProviderName returns the provider name
func (da *DeepSeekAdapter) GetProviderName() string {
	return "deepseek"
}

// OptimizeRequest optimizes requests for DeepSeek
func (da *DeepSeekAdapter) OptimizeRequest(req *LLMRequest) *LLMRequest {
	optimized := *req // Copy the request

	// DeepSeek performs well with slightly higher temperature for creative tasks
	if strings.Contains(strings.ToLower(req.Prompt), "creative") || strings.Contains(strings.ToLower(req.Prompt), "write") {
		if optimized.Temperature == nil || *optimized.Temperature < 0.7 {
			temp := 0.8
			optimized.Temperature = &temp
		}
	}

	// Optimize max_tokens for DeepSeek's context window
	if optimized.MaxTokens == nil || *optimized.MaxTokens > 4096 {
		maxTokens := 4096
		optimized.MaxTokens = &maxTokens
	}

	return &optimized
}

// ParseStreamingResponse parses DeepSeek streaming responses
func (da *DeepSeekAdapter) ParseStreamingResponse(reader io.Reader) <-chan StreamingChunk {
	ch := make(chan StreamingChunk, 10)

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()

			// DeepSeek uses similar SSE format to OpenAI
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				if data == "[DONE]" {
					ch <- StreamingChunk{Finish: true}
					return
				}

				var event map[string]interface{}
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					ch <- StreamingChunk{Error: fmt.Sprintf("Failed to parse event: %v", err)}
					continue
				}

				// Extract content from choices
				if choices, ok := event["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if delta, ok := choice["delta"].(map[string]interface{}); ok {
							if content, ok := delta["content"].(string); ok && content != "" {
								ch <- StreamingChunk{
									Content: content,
									Finish:  false,
								}
							}
						}
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamingChunk{Error: fmt.Sprintf("Scanner error: %v", err)}
		}

		// Send finish signal
		ch <- StreamingChunk{Finish: true}
	}()

	return ch
}

// HandleError handles DeepSeek-specific errors
func (da *DeepSeekAdapter) HandleError(resp *http.Response, body []byte) error {
	if resp.StatusCode < 400 {
		return nil
	}

	// DeepSeek error format might be similar to OpenAI
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("DeepSeek API error: %s", errorResp.Error.Message)
}

// GetOptimalBatchSize returns optimal batch size for DeepSeek
func (da *DeepSeekAdapter) GetOptimalBatchSize() int {
	return 10 // More conservative for DeepSeek
}

// GetRateLimitInfo extracts rate limit info from DeepSeek headers
func (da *DeepSeekAdapter) GetRateLimitInfo(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	// DeepSeek might use different header names
	if rpm := headers.Get("x-rpm-limit"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil {
			info.RequestsPerMinute = &val
		}
	}

	if tpm := headers.Get("x-tpm-limit"); tpm != "" {
		if val, err := strconv.Atoi(tpm); err == nil {
			info.TokensPerMinute = &val
		}
	}

	return info
}

// AdapterRegistry manages provider adapters
type AdapterRegistry struct {
	adapters map[string]ProviderAdapter
}

// NewAdapterRegistry creates a new adapter registry
func NewAdapterRegistry() *AdapterRegistry {
	registry := &AdapterRegistry{
		adapters: make(map[string]ProviderAdapter),
	}

	// Register built-in adapters
	registry.RegisterAdapter(NewOpenAIAdapter())
	registry.RegisterAdapter(NewDeepSeekAdapter())

	return registry
}

// RegisterAdapter registers a provider adapter
func (ar *AdapterRegistry) RegisterAdapter(adapter ProviderAdapter) {
	ar.adapters[adapter.GetProviderName()] = adapter
}

// GetAdapter returns the adapter for a provider
func (ar *AdapterRegistry) GetAdapter(providerName string) (ProviderAdapter, bool) {
	adapter, exists := ar.adapters[strings.ToLower(providerName)]
	return adapter, exists
}

// GetAvailableProviders returns all registered provider names
func (ar *AdapterRegistry) GetAvailableProviders() []string {
	providers := make([]string, 0, len(ar.adapters))
	for provider := range ar.adapters {
		providers = append(providers, provider)
	}
	return providers
}
