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

// AnthropicAdapter provides Anthropic-specific functionality
type AnthropicAdapter struct {
	BaseAdapter
}

// NewAnthropicAdapter creates a new Anthropic adapter
func NewAnthropicAdapter(client *http.Client, endpoint, apiKey string) *AnthropicAdapter {
	return &AnthropicAdapter{
		BaseAdapter: BaseAdapter{
			client:   client,
			endpoint: strings.TrimSuffix(endpoint, "/"),
			apiKey:   apiKey,
			headers: map[string]string{
				"Content-Type":      "application/json",
				"anthropic-version": "2023-06-01",
				"x-api-key":         apiKey,
			},
		},
	}
}

// AnthropicChatRequest represents a chat completion request for Anthropic
type AnthropicChatRequest struct {
	Model         string             `json:"model"`
	Messages      []AnthropicMessage `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	TopK          *int               `json:"top_k,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	System        string             `json:"system,omitempty"`
	Metadata      *AnthropicMetadata `json:"metadata,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicMetadata represents metadata for Anthropic requests
type AnthropicMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// AnthropicChatResponse represents a chat completion response from Anthropic
type AnthropicChatResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []AnthropicContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Usage      AnthropicUsage     `json:"usage"`
}

// AnthropicContent represents content in Anthropic responses
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// AnthropicUsage represents token usage information
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamResponse represents a streaming response from Anthropic
type AnthropicStreamResponse struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta struct {
		Type string `json:"type,omitempty"`
		Text string `json:"text,omitempty"`
	} `json:"delta,omitempty"`
}

// convertToAnthropicRequest converts OpenAI format to Anthropic format
func (a *AnthropicAdapter) convertToAnthropicRequest(openaiReq OpenAIChatRequest) AnthropicChatRequest {
	anthropicReq := AnthropicChatRequest{
		Model:     openaiReq.Model,
		MaxTokens: openaiReq.MaxTokens,
		Stream:    openaiReq.Stream,
	}

	// Convert messages from OpenAI to Anthropic format
	for _, msg := range openaiReq.Messages {
		anthropicMsg := AnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		anthropicReq.Messages = append(anthropicReq.Messages, anthropicMsg)
	}

	// Set optional parameters if they are non-zero
	if openaiReq.Temperature != 0 {
		anthropicReq.Temperature = &openaiReq.Temperature
	}
	if openaiReq.TopP != 0 {
		anthropicReq.TopP = &openaiReq.TopP
	}

	return anthropicReq
}

// convertFromAnthropicResponse converts Anthropic format to OpenAI format
func (a *AnthropicAdapter) convertFromAnthropicResponse(anthropicResp AnthropicChatResponse) OpenAIChatResponse {
	openaiResp := OpenAIChatResponse{
		ID:      anthropicResp.ID,
		Object:  "chat.completion",
		Created: 0, // Anthropic doesn't provide creation time
		Model:   anthropicResp.Model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}

	// Convert content
	if len(anthropicResp.Content) > 0 {
		content := ""
		for _, c := range anthropicResp.Content {
			if c.Type == "text" {
				content += c.Text
			}
		}
		openaiResp.Choices = append(openaiResp.Choices, struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{
			Index: 0,
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    anthropicResp.Role,
				Content: content,
			},
		})
	}

	return openaiResp
}

// StreamChatCompletion streams a chat completion from Anthropic
func (a *AnthropicAdapter) StreamChatCompletion(ctx context.Context, request OpenAIChatRequest) (<-chan OpenAIStreamResponse, <-chan error) {
	responseChan := make(chan OpenAIStreamResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		anthropicReq := a.convertToAnthropicRequest(request)
		anthropicReq.Stream = true

		// Prepare request body
		requestBody, err := json.Marshal(anthropicReq)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Create HTTP request
		url := fmt.Sprintf("%s/messages", a.endpoint)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		// Set headers
		for key, value := range a.headers {
			req.Header.Set(key, value)
		}

		// Send request
		resp, err := a.client.Do(req)
		if err != nil {
			errorChan <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorChan <- fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
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

				var anthropicStream AnthropicStreamResponse
				if err := json.Unmarshal([]byte(data), &anthropicStream); err != nil {
					continue // Skip malformed lines
				}

				// Convert to OpenAI format
				openaiStream := OpenAIStreamResponse{
					ID:      "anthropic-stream",
					Object:  "chat.completion.chunk",
					Created: 0,
					Model:   request.Model,
					Choices: []OpenAIChoice{},
				}

				if anthropicStream.Delta.Text != "" {
					openaiStream.Choices = append(openaiStream.Choices, OpenAIChoice{
						Index: anthropicStream.Index,
						Delta: OpenAIDelta{
							Content: anthropicStream.Delta.Text,
						},
					})
				}

				select {
				case responseChan <- openaiStream:
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
func (a *AnthropicAdapter) ChatCompletion(ctx context.Context, request OpenAIChatRequest) (*OpenAIChatResponse, error) {
	anthropicReq := a.convertToAnthropicRequest(request)

	// Prepare request body
	requestBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/messages", a.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range a.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anthropicResp AnthropicChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	openaiResp := a.convertFromAnthropicResponse(anthropicResp)
	return &openaiResp, nil
}

// ListModels retrieves available models from Anthropic
func (a *AnthropicAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	// Anthropic doesn't have a models endpoint like OpenAI, so we'll return known models
	modelsResp := &OpenAIModelsResponse{
		Object: "list",
		Data: []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			{ID: "claude-3-opus-20240229", Object: "model", Created: 1707955200, OwnedBy: "anthropic"},
			{ID: "claude-3-sonnet-20240229", Object: "model", Created: 1707955200, OwnedBy: "anthropic"},
			{ID: "claude-3-haiku-20240307", Object: "model", Created: 1709856000, OwnedBy: "anthropic"},
			{ID: "claude-3-5-sonnet-20240620", Object: "model", Created: 1718841600, OwnedBy: "anthropic"},
		},
	}

	return modelsResp, nil
}
