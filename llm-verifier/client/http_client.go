package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPClient represents an HTTP client for making LLM API requests
type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// TestModelExists checks if a model is available on provider's API
func (c *HTTPClient) TestModelExists(ctx context.Context, provider, apiKey, modelID string) (bool, error) {
	endpoint := getProviderEndpoint(provider)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err == nil {
		// Check if model exists in response
		if models, ok := data["data"].([]interface{}); ok {
			for _, m := range models {
				if model, ok := m.(map[string]interface{}); ok {
					if id, ok := model["id"].(string); ok && id == modelID {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// TestResponsiveness measures how quickly a model responds to a test prompt
func (c *HTTPClient) TestResponsiveness(ctx context.Context, provider, apiKey, modelID, prompt string) (time.Duration, time.Duration, error, string, bool, int, error) {
	endpoint := getModelEndpoint(provider, modelID)

	requestBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": 10,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return time.Duration(0), time.Duration(0), err, "", false, 0, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return time.Duration(0), time.Duration(0), err, "", false, 0, nil
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return time.Duration(0), time.Duration(0), err, "", false, 0, err
	}
	defer resp.Body.Close()

	totalTime := time.Since(start)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return time.Duration(0), time.Duration(0), err, "", false, 0, nil
	}

	// Parse response for TTFT (time to first token)
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err == nil {
		// Estimate TTFT as 20% of total time for non-streaming
		ttft := time.Duration(float64(totalTime) * 0.2)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return totalTime, ttft, nil, "", true, resp.StatusCode, nil
		}

		return totalTime, ttft, nil, fmt.Sprintf("HTTP %d", resp.StatusCode), false, resp.StatusCode, nil
	}

	return totalTime, time.Duration(0), nil, "Invalid response format", false, resp.StatusCode, nil
}

// TestStreaming tests if a model supports streaming responses
func (c *HTTPClient) TestStreaming(ctx context.Context, provider, apiKey, modelID, prompt string) (bool, error) {
	endpoint := getModelEndpoint(provider, modelID)

	requestBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"stream":     true,
		"max_tokens": 50,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Count chunks in streaming response
	chunkCount := 0
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "data: ") {
			chunkCount++
		}
		if line == "[DONE]" {
			break
		}
	}

	return chunkCount > 0, nil
}

// getProviderEndpoint returns the models list endpoint for a provider
func getProviderEndpoint(provider string) string {
	providerEndpoints := map[string]string{
		"openai":      "https://api.openai.com/v1/models",
		"anthropic":   "https://api.anthropic.com/v1/models",
		"huggingface": "https://api-inference.huggingface.co/models",
		"google":      "https://generativelanguage.googleapis.com/v1/models",
		"cohere":      "https://api.cohere.ai/v1/models",
		"openrouter":  "https://openrouter.ai/api/v1/models",
		"deepseek":    "https://api.deepseek.com/v1/models",
	}

	if endpoint, ok := providerEndpoints[strings.ToLower(provider)]; ok {
		return endpoint
	}
	return ""
}

// getModelEndpoint returns the chat/completion endpoint for a provider
func getModelEndpoint(provider, modelID string) string {
	providerEndpoints := map[string]string{
		"openai":      "https://api.openai.com/v1/chat/completions",
		"anthropic":   "https://api.anthropic.com/v1/messages",
		"huggingface": "https://api-inference.huggingface.co/models/" + modelID,
		"google":      "https://generativelanguage.googleapis.com/v1beta/models/" + modelID + ":generateContent",
		"cohere":      "https://api.cohere.ai/v1/generate",
		"openrouter":  "https://openrouter.ai/api/v1/chat/completions",
		"deepseek":    "https://api.deepseek.com/chat/completions",
	}

	if endpoint, ok := providerEndpoints[strings.ToLower(provider)]; ok {
		return endpoint
	}
	return ""
}

// DetectErrorType categorizes HTTP errors
func DetectErrorType(statusCode int, body []byte) string {
	switch statusCode {
	case 401:
		return "authentication_error"
	case 429:
		return "rate_limit_exceeded"
	case 404:
		return "model_not_found"
	case 500:
		return "server_error"
	default:
		if statusCode >= 400 && statusCode < 500 {
			return "client_error"
		}
		return "unknown_error"
	}
}
