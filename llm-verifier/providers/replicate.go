// Package providers implements LLM provider adapters
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ReplicateAdapter provides Replicate-specific functionality
type ReplicateAdapter struct {
	BaseAdapter
}

// NewReplicateAdapter creates a new Replicate adapter
func NewReplicateAdapter(client *http.Client, endpoint, apiKey string) *ReplicateAdapter {
	return &ReplicateAdapter{
		BaseAdapter: BaseAdapter{
			client:   client,
			endpoint: strings.TrimSuffix(endpoint, "/"),
			apiKey:   apiKey,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Token %s", apiKey),
				"Content-Type":  "application/json",
			},
		},
	}
}

// StreamChatCompletion streams a chat completion from Replicate
func (r *ReplicateAdapter) StreamChatCompletion(ctx context.Context, request OpenAIChatRequest) (<-chan OpenAIStreamResponse, <-chan error) {
	responseChan := make(chan OpenAIStreamResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		// Convert OpenAI format to Replicate format
		replicateRequest := map[string]interface{}{
			"input": map[string]interface{}{
				"prompt":             extractPromptFromMessages(request.Messages),
				"max_new_tokens":     request.MaxTokens,
				"temperature":        request.Temperature,
				"top_p":              0.9,
				"repetition_penalty": 1.0,
			},
		}

		// Prepare request body
		requestBody, err := json.Marshal(replicateRequest)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Create HTTP request
		url := fmt.Sprintf("%s/predictions", r.endpoint)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		// Set headers
		for key, value := range r.headers {
			req.Header.Set(key, value)
		}

		// Send request
		resp, err := r.client.Do(req)
		if err != nil {
			errorChan <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			errorChan <- fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			return
		}

		// Parse prediction response to get prediction URL
		var predictionResp struct {
			URLs struct {
				Get string `json:"get"`
			} `json:"urls"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&predictionResp); err != nil {
			errorChan <- fmt.Errorf("failed to decode prediction response: %w", err)
			return
		}

		// Poll for completion
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Check prediction status
				statusReq, err := http.NewRequestWithContext(ctx, "GET", predictionResp.URLs.Get, nil)
				if err != nil {
					errorChan <- fmt.Errorf("failed to create status request: %w", err)
					return
				}

				for key, value := range r.headers {
					statusReq.Header.Set(key, value)
				}

				statusResp, err := r.client.Do(statusReq)
				if err != nil {
					errorChan <- fmt.Errorf("failed to check status: %w", err)
					return
				}

				var statusData struct {
					Status string   `json:"status"`
					Output []string `json:"output"`
				}

				if err := json.NewDecoder(statusResp.Body).Decode(&statusData); err != nil {
					statusResp.Body.Close()
					continue
				}
				statusResp.Body.Close()

				if statusData.Status == "succeeded" && len(statusData.Output) > 0 {
					// Convert to OpenAI format and send
					finishReason := "stop"
					streamResp := OpenAIStreamResponse{
						ID:      "replicate-" + fmt.Sprintf("%d", time.Now().Unix()),
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   request.Model,
						Choices: []OpenAIChoice{
							{
								Index: 0,
								Delta: OpenAIDelta{
									Content: statusData.Output[0],
								},
								FinishReason: &finishReason,
							},
						},
					}

					select {
					case responseChan <- streamResp:
					case <-ctx.Done():
						return
					}
					return
				} else if statusData.Status == "failed" {
					errorChan <- fmt.Errorf("prediction failed")
					return
				}
			}
		}
	}()

	return responseChan, errorChan
}

// ChatCompletion performs a non-streaming chat completion
func (r *ReplicateAdapter) ChatCompletion(ctx context.Context, request OpenAIChatRequest) (*OpenAIChatResponse, error) {
	// For non-streaming, we can use the streaming method and collect results
	streamChan, errChan := r.StreamChatCompletion(ctx, request)

	var fullResponse strings.Builder

	for {
		select {
		case response := <-streamChan:
			if len(response.Choices) > 0 {
				fullResponse.WriteString(response.Choices[0].Delta.Content)
			}
		case err := <-errChan:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Check if stream is done (empty channels)
		if streamChan == nil && errChan == nil {
			break
		}
	}

	return &OpenAIChatResponse{
		ID:      "replicate-" + fmt.Sprintf("%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
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
					Content: fullResponse.String(),
				},
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     len(extractPromptFromMessages(request.Messages)),
			CompletionTokens: len(fullResponse.String()),
			TotalTokens:      len(extractPromptFromMessages(request.Messages)) + len(fullResponse.String()),
		},
	}, nil
}

// ListModels retrieves available models from Replicate
func (r *ReplicateAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	// Replicate has a different model listing approach
	// For now, return a static list of popular models
	models := []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}{
		{
			ID:      "meta/llama-2-70b-chat",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "replicate",
		},
		{
			ID:      "meta/llama-2-13b-chat",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "replicate",
		},
		{
			ID:      "mistralai/mistral-7b-instruct-v0.1",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "replicate",
		},
	}

	return &OpenAIModelsResponse{
		Object: "list",
		Data:   models,
	}, nil
}

// Helper function to extract prompt from OpenAI messages
func extractPromptFromMessages(messages []Message) string {
	var prompt strings.Builder
	for _, msg := range messages {
		prompt.WriteString(msg.Content)
		prompt.WriteString("\n")
	}
	return strings.TrimSpace(prompt.String())
}
