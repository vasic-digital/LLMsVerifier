// Package providers implements LLM provider adapters
package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
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
// Cohere has a models endpoint at /v1/models - let's use it
func (c *CohereAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	// Try to fetch models from Cohere's API
	models, err := c.fetchModelsFromAPI(ctx)
	if err != nil {
		// Fall back to known models if API call fails
		log.Printf("Warning: Could not fetch Cohere models from API: %v. Using known models.", err)
		models = c.getKnownModels()
	}

	return &OpenAIModelsResponse{
		Object: "list",
		Data:   models,
	}, nil
}

// fetchModelsFromAPI attempts to fetch models from Cohere's API
func (c *CohereAdapter) fetchModelsFromAPI(ctx context.Context) ([]struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint+"/models", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Cohere API response structure
	var cohereResp struct {
		Models []struct {
			Name            string   `json:"name"`
			Endpoints       []string `json:"endpoints"`
			ContextLength   int      `json:"context_length"`
			TokenizerURL    string   `json:"tokenizer_url,omitempty"`
			DefaultEndpoint string   `json:"default_endpoint,omitempty"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&cohereResp); err != nil {
		return nil, err
	}

	var models []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	now := time.Now().Unix()
	for _, m := range cohereResp.Models {
		// Only include models that support chat
		hasChat := false
		for _, ep := range m.Endpoints {
			if ep == "chat" || ep == "generate" {
				hasChat = true
				break
			}
		}
		if hasChat {
			models = append(models, struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				OwnedBy string `json:"owned_by"`
			}{
				ID:      m.Name,
				Object:  "model",
				Created: now,
				OwnedBy: "cohere",
			})
		}
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no chat models found")
	}

	return models, nil
}

// getKnownModels returns a curated list of known Cohere models
// Updated based on Cohere's documentation - Last verified: 2025-01
func (c *CohereAdapter) getKnownModels() []struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
} {
	return []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}{
		{ID: "command-r-plus", Object: "model", Created: 1712102400, OwnedBy: "cohere"},  // Apr 2024
		{ID: "command-r", Object: "model", Created: 1709510400, OwnedBy: "cohere"},       // Mar 2024
		{ID: "command", Object: "model", Created: 1672531200, OwnedBy: "cohere"},         // Jan 2023
		{ID: "command-light", Object: "model", Created: 1672531200, OwnedBy: "cohere"},   // Jan 2023
		{ID: "command-nightly", Object: "model", Created: 1704067200, OwnedBy: "cohere"}, // Jan 2024
		{ID: "command-light-nightly", Object: "model", Created: 1704067200, OwnedBy: "cohere"},
	}
}
