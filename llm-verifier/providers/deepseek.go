package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

// DeepSeekAdapter provides DeepSeek-specific functionality
type DeepSeekAdapter struct {
	BaseAdapter
}

// NewDeepSeekAdapter creates a new DeepSeek adapter
func NewDeepSeekAdapter(client *http.Client, endpoint, apiKey string) *DeepSeekAdapter {
	return &DeepSeekAdapter{
		BaseAdapter: BaseAdapter{
			client:   client,
			endpoint: strings.TrimSuffix(endpoint, "/"),
			apiKey:   apiKey,
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", apiKey),
			},
		},
	}
}

// ChatCompletion performs a chat completion with DeepSeek
func (d *DeepSeekAdapter) ChatCompletion(ctx context.Context, request DeepSeekChatRequest) (*DeepSeekChatResponse, error) {
	var response DeepSeekChatResponse

	err := d.doWithRetry(ctx, func() error {
		// Prepare request body
		requestBody, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		log.Printf("API REQUEST: POST %s/chat/completions", d.endpoint)
		log.Printf("API REQUEST HEADERS: %v", d.headers)
		log.Printf("API REQUEST BODY: %s", string(requestBody))

		// Create HTTP request
		url := fmt.Sprintf("%s/chat/completions", d.endpoint)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		for key, value := range d.headers {
			req.Header.Set(key, value)
		}

		// Make request
		resp, err := d.client.Do(req)
		if err != nil {
			log.Printf("API RESPONSE ERROR: %v", err)
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		log.Printf("API RESPONSE STATUS: %d", resp.StatusCode)
		log.Printf("API RESPONSE HEADERS: %v", resp.Header)
		log.Printf("API RESPONSE BODY LENGTH: %d bytes", len(body))

		if resp.StatusCode != http.StatusOK {
			if d.isRetryableError(nil, resp.StatusCode) {
				log.Printf("API RESPONSE BODY: %s", string(body))
				return fmt.Errorf("API returned retryable status %d", resp.StatusCode)
			}
			return fmt.Errorf("API returned permanent status %d: %s", resp.StatusCode, string(body))
		}

		// Parse response
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// StreamChatCompletion streams a chat completion from DeepSeek
func (d *DeepSeekAdapter) StreamChatCompletion(ctx context.Context, request DeepSeekChatRequest) (<-chan DeepSeekStreamResponse, <-chan error) {
	responseChan := make(chan DeepSeekStreamResponse, 10)
	errorChan := make(chan error, 1)

	// Enable streaming
	request.Stream = true

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		// Prepare request body
		requestBody, err := json.Marshal(request)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Create HTTP request
		url := fmt.Sprintf("%s/chat/completions", d.endpoint)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		// Set headers
		for key, value := range d.headers {
			req.Header.Set(key, value)
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")

		// Make request
		resp, err := d.client.Do(req)
		if err != nil {
			errorChan <- fmt.Errorf("failed to make request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorChan <- fmt.Errorf("DeepSeek API error: %d - %s", resp.StatusCode, string(body))
			return
		}

		// Parse SSE stream (similar to OpenAI but may have different format)
		scanner := NewSSEScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				if data == "[DONE]" {
					break
				}

				var streamResp DeepSeekStreamResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					continue
				}

				select {
				case responseChan <- streamResp:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errorChan <- fmt.Errorf("error reading stream: %w", err)
		}
	}()

	return responseChan, errorChan
}

// ValidateRequest validates a DeepSeek chat request
func (d *DeepSeekAdapter) ValidateRequest(request DeepSeekChatRequest) error {
	if request.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(request.Messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}
	if request.MaxTokens < 0 {
		return fmt.Errorf("max_tokens cannot be negative")
	}
	if request.Temperature < 0 || request.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	return nil
}

// Retry configuration
var deepSeekRetryConfig = struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}{
	MaxAttempts: 3,
	BaseDelay:   1 * time.Second,
	MaxDelay:    30 * time.Second,
}

// isRetryableError determines if an error is worth retrying
func (d *DeepSeekAdapter) isRetryableError(err error, statusCode int) bool {
	if err != nil {
		// Network errors are retryable
		return true
	}

	// HTTP status codes that are retryable
	switch statusCode {
	case 429: // Too Many Requests
		return true
	case 500, 502, 503, 504: // Server errors
		return true
	case 408: // Request Timeout
		return true
	}

	return false
}

// doWithRetry executes a function with exponential backoff retry
func (d *DeepSeekAdapter) doWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= deepSeekRetryConfig.MaxAttempts; attempt++ {
		log.Printf("DeepSeek attempt %d/%d", attempt, deepSeekRetryConfig.MaxAttempts)

		err := operation()
		if err == nil {
			if attempt > 1 {
				log.Printf("DeepSeek operation succeeded on attempt %d", attempt)
			}
			return nil
		}

		lastErr = err
		log.Printf("DeepSeek attempt %d failed: %v", attempt, err)

		// Don't retry on last attempt
		if attempt == deepSeekRetryConfig.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff
		delay := time.Duration(float64(deepSeekRetryConfig.BaseDelay) * math.Pow(2, float64(attempt-1)))
		if delay > deepSeekRetryConfig.MaxDelay {
			delay = deepSeekRetryConfig.MaxDelay
		}

		// Add jitter (±25%)
		jitterRange := int64(delay) / 4 // 25% of delay
		jitter := time.Duration((time.Now().UnixNano() % (2 * jitterRange)) - jitterRange)
		totalDelay := delay + jitter

		log.Printf("DeepSeek retrying in %v...", totalDelay)
		select {
		case <-time.After(totalDelay):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("DeepSeek operation failed after %d attempts, last error: %w", deepSeekRetryConfig.MaxAttempts, lastErr)
}

// GetModelInfo retrieves model information from DeepSeek
func (d *DeepSeekAdapter) GetModelInfo(ctx context.Context, model string) (*ModelInfo, error) {
	var modelResp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	err := d.doWithRetry(ctx, func() error {
		url := fmt.Sprintf("%s/models/%s", d.endpoint, model)
		log.Printf("API REQUEST: GET %s", url)
		log.Printf("API REQUEST HEADERS: %v", d.headers)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		for key, value := range d.headers {
			req.Header.Set(key, value)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			log.Printf("API RESPONSE ERROR: %v", err)
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		log.Printf("API RESPONSE STATUS: %d", resp.StatusCode)
		log.Printf("API RESPONSE HEADERS: %v", resp.Header)
		log.Printf("API RESPONSE BODY LENGTH: %d bytes", len(body))

		if resp.StatusCode != http.StatusOK {
			if d.isRetryableError(nil, resp.StatusCode) {
				return fmt.Errorf("API returned retryable status %d", resp.StatusCode)
			}
			return fmt.Errorf("API returned permanent status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, &modelResp); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ModelInfo{
		ID:      modelResp.ID,
		Object:  modelResp.Object,
		Created: modelResp.Created,
		OwnedBy: modelResp.OwnedBy,
	}, nil
}

// DeepSeekChatRequest represents a chat completion request for DeepSeek
type DeepSeekChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// DeepSeekChatResponse represents a chat completion response from DeepSeek
type DeepSeekChatResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []DeepSeekChoice `json:"choices"`
	Usage   DeepSeekUsage    `json:"usage"`
}

// DeepSeekChoice represents a choice in the DeepSeek response
type DeepSeekChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// DeepSeekUsage represents token usage information
type DeepSeekUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// DeepSeekStreamResponse represents a streaming response from DeepSeek
type DeepSeekStreamResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []DeepSeekChoice `json:"choices"`
}

// SSEScanner provides SSE (Server-Sent Events) scanning functionality
type SSEScanner struct {
	scanner *bufio.Scanner
}

// NewSSEScanner creates a new SSE scanner
func NewSSEScanner(reader io.Reader) *SSEScanner {
	return &SSEScanner{
		scanner: bufio.NewScanner(reader),
	}
}

// Scan advances to the next line
func (s *SSEScanner) Scan() bool {
	return s.scanner.Scan()
}

// Text returns the current line
func (s *SSEScanner) Text() string {
	return s.scanner.Text()
}

// Err returns the scanner error
func (s *SSEScanner) Err() error {
	return s.scanner.Err()
}
