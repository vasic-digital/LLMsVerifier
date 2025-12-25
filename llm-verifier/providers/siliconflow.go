// Package providers implements LLM provider adapters
package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SiliconFlowAdapter provides SiliconFlow-specific functionality
type SiliconFlowAdapter struct {
	BaseAdapter
}

// NewSiliconFlowAdapter creates a new SiliconFlow adapter
func NewSiliconFlowAdapter(client *http.Client, endpoint, apiKey string) *SiliconFlowAdapter {
	return &SiliconFlowAdapter{
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

// StreamChatCompletion streams a chat completion from SiliconFlow
func (s *SiliconFlowAdapter) StreamChatCompletion(ctx context.Context, request OpenAIChatRequest) (<-chan OpenAIStreamResponse, <-chan error) {
	responseChan := make(chan OpenAIStreamResponse, 10)
	errorChan := make(chan error, 1)

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
		url := fmt.Sprintf("%s/chat/completions", s.endpoint)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		// Set headers
		for key, value := range s.headers {
			req.Header.Set(key, value)
		}

		// Send request
		resp, err := s.client.Do(req)
		if err != nil {
			errorChan <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errorChan <- fmt.Errorf("API request failed with status: %d", resp.StatusCode)
			return
		}

		// Parse streaming response
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					break
				}

				var streamResp OpenAIStreamResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					continue // Skip malformed lines
				}

				select {
				case responseChan <- streamResp:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errorChan <- fmt.Errorf("error reading response: %w", err)
		}
	}()

	return responseChan, errorChan
}

// ChatCompletion performs a non-streaming chat completion
func (s *SiliconFlowAdapter) ChatCompletion(ctx context.Context, request OpenAIChatRequest) (*OpenAIChatResponse, error) {
	// Prepare request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", s.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range s.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp OpenAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// ListModels retrieves available models from SiliconFlow
func (s *SiliconFlowAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/models", s.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range s.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var modelsResp OpenAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &modelsResp, nil
}
