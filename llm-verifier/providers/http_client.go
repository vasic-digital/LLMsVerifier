package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient provides HTTP client functionality for provider requests
type HTTPClient struct {
	client      *http.Client
	baseURL     string
	apiKey      string
	headers     map[string]string
	retryConfig *RetryConfig
}

// HTTPClientConfig configures the HTTP client
type HTTPClientConfig struct {
	BaseURL      string
	APIKey       string
	Headers      map[string]string
	Timeout      time.Duration
	RetryConfig  *RetryConfig
	MaxIdleConns int
	IdleTimeout  time.Duration
}

// NewHTTPClient creates a new HTTP client for provider requests
func NewHTTPClient(config *HTTPClientConfig) *HTTPClient {
	if config == nil {
		config = &HTTPClientConfig{}
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	maxIdleConns := config.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 100
	}

	idleTimeout := config.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = 90 * time.Second
	}

	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     idleTimeout,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return &HTTPClient{
		client:      client,
		baseURL:     config.BaseURL,
		apiKey:      config.APIKey,
		headers:     config.Headers,
		retryConfig: config.RetryConfig,
	}
}

// Request represents an HTTP request
type Request struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do executes an HTTP request
func (c *HTTPClient) Do(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	url := c.baseURL + req.Path

	var body io.Reader
	if req.Body != nil {
		data, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = &bytesReader{data: data}
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Set configured headers
	for k, v := range c.headers {
		httpReq.Header.Set(k, v)
	}

	// Set request-specific headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Execute request with retry logic
	var resp *http.Response
	var lastErr error

	maxRetries := 1
	if c.retryConfig != nil {
		maxRetries = c.retryConfig.MaxRetries + 1
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, lastErr = c.client.Do(httpReq)
		if lastErr == nil && resp.StatusCode < 500 {
			break
		}

		if attempt < maxRetries-1 && c.retryConfig != nil {
			delay := c.retryConfig.InitialDelay
			for i := 0; i < attempt; i++ {
				delay = time.Duration(float64(delay) * c.retryConfig.BackoffFactor)
				if delay > c.retryConfig.MaxDelay {
					delay = c.retryConfig.MaxDelay
					break
				}
			}
			time.Sleep(delay)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}, nil
}

// Get performs a GET request
func (c *HTTPClient) Get(ctx context.Context, path string) (*Response, error) {
	return c.Do(ctx, &Request{
		Method: http.MethodGet,
		Path:   path,
	})
}

// Post performs a POST request
func (c *HTTPClient) Post(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, &Request{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	})
}

// SetBaseURL sets the base URL
func (c *HTTPClient) SetBaseURL(url string) {
	c.baseURL = url
}

// SetAPIKey sets the API key
func (c *HTTPClient) SetAPIKey(key string) {
	c.apiKey = key
}

// SetHeader sets a header
func (c *HTTPClient) SetHeader(key, value string) {
	if c.headers == nil {
		c.headers = make(map[string]string)
	}
	c.headers[key] = value
}

// bytesReader is an io.Reader that reads from a byte slice
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
