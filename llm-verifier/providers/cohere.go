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

// CohereAdapter provides Cohere-specific functionality
type CohereAdapter struct {
	BaseAdapter
}

// NewCohereAdapter creates a new Cohere adapter
func NewCohereAdapter(client *http.Client, endpoint, apiKey string) *CohereAdapter {
	return &CohereAdapter{
		BaseAdapter: BaseAdapter{
			client:   client,
			endpoint: strings.TrimSuffix(endpoint, "/"),
			apiKey:   apiKey,
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", apiKey),
				"Accept":        "application/json",
			},
		},
	}
}

// CohereChatRequest represents a chat completion request for Cohere
type CohereChatRequest struct {
	Message     string  `json:"message"`
	Model       string  `json:"model,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// CohereChatResponse represents a chat completion response from Cohere
type CohereChatResponse struct {
	ResponseID   string `json:"response_id"`
	Text         string `json:"text"`
	GenerationID string `json:"generation_id"`
	TokenCount   struct {
		PromptTokens   int `json:"prompt_tokens"`
		ResponseTokens int `json:"response_tokens"`
		TotalTokens    int `json:"total_tokens"`
		BilledTokens   int `json:"billed_tokens"`
	} `json:"token_count"`
	Meta struct {
		APIVersion struct {
			Version string `json:"version"`
		} `json:"api_version"`
	} `json:"meta"`
}

// StreamChatCompletion streams a chat completion from Cohere
func (c *CohereAdapter) StreamChatCompletion(ctx context.Context, request OpenAIChatRequest) (<-chan OpenAIStreamResponse, <-chan error) {
	responseChan := make(chan OpenAIStreamResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		cohereReq := CohereChatRequest{
			Message:     request.Messages[len(request.Messages)-1].Content, // Use last message as prompt
			Model:       request.Model,
			MaxTokens:   request.MaxTokens,
			Temperature: request.Temperature,
			Stream:      true,
		}

		// Prepare request body
		requestBody, err := json.Marshal(cohereReq)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Create HTTP request
		url := fmt.Sprintf("%s/generate", c.endpoint)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		// Set headers
		for key, value := range c.headers {
			req.Header.Set(key, value)
		}

		// Send request
		resp, err := c.client.Do(req)
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

				// Cohere streaming format is different, simplified for this example
				if strings.TrimSpace(data) != "" {
					openaiStream := OpenAIStreamResponse{
						ID:      "cohere-stream",
						Object:  "chat.completion.chunk",
						Created: 0,
						Model:   request.Model,
						Choices: []OpenAIChoice{
							{
								Index: 0,
								Delta: OpenAIDelta{
									Content: data,
								},
							},
						},
					}

					select {
					case responseChan <- openaiStream:
					case <-ctx.Done():
						return
					}
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
func (c *CohereAdapter) ChatCompletion(ctx context.Context, request OpenAIChatRequest) (*OpenAIChatResponse, error) {
	cohereReq := CohereChatRequest{
		Message:     request.Messages[len(request.Messages)-1].Content, // Use last message as prompt
		Model:       request.Model,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		Stream:      false,
	}

	// Prepare request body
	requestBody, err := json.Marshal(cohereReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/generate", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var cohereResp CohereChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cohereResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	openaiResp := OpenAIChatResponse{
		ID:      cohereResp.ResponseID,
		Object:  "chat.completion",
		Created: 0,
		Model:   request.Model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: cohereResp.Text,
				},
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     cohereResp.TokenCount.PromptTokens,
			CompletionTokens: cohereResp.TokenCount.ResponseTokens,
			TotalTokens:      cohereResp.TokenCount.TotalTokens,
		},
	}

	return &openaiResp, nil
}

// ListModels retrieves available models from Cohere
func (c *CohereAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	// Cohere doesn't have a models endpoint, so we'll return known models
	modelsResp := &OpenAIModelsResponse{
		Object: "list",
		Data: []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			{ID: "command", Object: "model", Created: 1640000000, OwnedBy: "cohere"},
			{ID: "base", Object: "model", Created: 1640000000, OwnedBy: "cohere"},
			{ID: "command-light", Object: "model", Created: 1640000000, OwnedBy: "cohere"},
		},
	}

	return modelsResp, nil
}
