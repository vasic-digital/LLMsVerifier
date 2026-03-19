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

	"digital.vasic.llmsverifier/auth"
)

// QwenAdapter provides Qwen/DashScope-specific functionality
type QwenAdapter struct {
	BaseAdapter
	authType        AuthType
	oauthCredReader *auth.OAuthCredentialReader
}

// NewQwenAdapter creates a new Qwen adapter
func NewQwenAdapter(client *http.Client, endpoint, apiKey string) *QwenAdapter {
	return &QwenAdapter{
		BaseAdapter: BaseAdapter{
			client:   client,
			endpoint: strings.TrimSuffix(endpoint, "/"),
			apiKey:   apiKey,
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + apiKey,
			},
		},
		authType: AuthTypeAPIKey,
	}
}

// NewQwenAdapterWithOAuth creates a new Qwen adapter using OAuth credentials from Qwen Code CLI
func NewQwenAdapterWithOAuth(client *http.Client, endpoint string) (*QwenAdapter, error) {
	credReader := auth.GetGlobalOAuthReader()

	// Verify credentials are available
	if !credReader.HasValidQwenCredentials() {
		return nil, fmt.Errorf("no valid Qwen OAuth credentials available: ensure you are logged in via Qwen Code CLI")
	}

	token, err := credReader.GetQwenAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get Qwen OAuth token: %w", err)
	}

	// Use Qwen Chat API for OAuth instead of DashScope
	if endpoint == "" {
		endpoint = "https://chat.qwen.ai/api/v1"
	}

	return &QwenAdapter{
		BaseAdapter: BaseAdapter{
			client:   client,
			endpoint: strings.TrimSuffix(endpoint, "/"),
			apiKey:   "", // Will use OAuth token instead
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		authType:        AuthTypeOAuth,
		oauthCredReader: credReader,
	}, nil
}

// NewQwenAdapterAuto creates a Qwen adapter, automatically choosing OAuth if enabled and available
func NewQwenAdapterAuto(client *http.Client, endpoint, apiKey string) (*QwenAdapter, error) {
	// Check if OAuth is enabled and credentials are available
	if auth.IsQwenOAuthEnabled() {
		credReader := auth.GetGlobalOAuthReader()
		if credReader.HasValidQwenCredentials() {
			return NewQwenAdapterWithOAuth(client, endpoint)
		}
	}

	// Fall back to API key authentication
	if apiKey == "" {
		return nil, fmt.Errorf("no API key provided and OAuth credentials not available")
	}
	return NewQwenAdapter(client, endpoint, apiKey), nil
}

// GetAuthType returns the authentication type being used
func (q *QwenAdapter) GetAuthType() AuthType {
	return q.authType
}

// refreshAuthHeaders refreshes the authentication headers if using OAuth
func (q *QwenAdapter) refreshAuthHeaders() error {
	if q.authType != AuthTypeOAuth || q.oauthCredReader == nil {
		return nil
	}

	token, err := q.oauthCredReader.GetQwenAccessToken()
	if err != nil {
		return fmt.Errorf("failed to refresh OAuth token: %w", err)
	}

	q.headers["Authorization"] = "Bearer " + token
	return nil
}

// QwenChatRequest represents a chat completion request for Qwen
type QwenChatRequest struct {
	Model       string        `json:"model"`
	Messages    []QwenMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
}

// QwenMessage represents a message in Qwen format
type QwenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// QwenChatResponse represents a chat completion response from Qwen
type QwenChatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []QwenChoice `json:"choices"`
	Usage   QwenUsage    `json:"usage"`
}

// QwenChoice represents a choice in the Qwen response
type QwenChoice struct {
	Index        int         `json:"index"`
	Message      QwenMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// QwenUsage represents token usage in the Qwen response
type QwenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// QwenStreamChunk represents a streaming chunk from Qwen
type QwenStreamChunk struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []QwenStreamChoice `json:"choices"`
}

// QwenStreamChoice represents a choice in a streaming response
type QwenStreamChoice struct {
	Index        int             `json:"index"`
	Delta        QwenStreamDelta `json:"delta"`
	FinishReason *string         `json:"finish_reason"`
}

// QwenStreamDelta represents the delta content in a streaming response
type QwenStreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// convertToQwenRequest converts OpenAI format to Qwen format
func (q *QwenAdapter) convertToQwenRequest(openaiReq OpenAIChatRequest) QwenChatRequest {
	qwenReq := QwenChatRequest{
		Model:       openaiReq.Model,
		MaxTokens:   openaiReq.MaxTokens,
		Stream:      openaiReq.Stream,
		Temperature: openaiReq.Temperature,
		TopP:        openaiReq.TopP,
	}

	// Convert messages
	for _, msg := range openaiReq.Messages {
		qwenReq.Messages = append(qwenReq.Messages, QwenMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return qwenReq
}

// convertFromQwenResponse converts Qwen format to OpenAI format
func (q *QwenAdapter) convertFromQwenResponse(qwenResp QwenChatResponse) OpenAIChatResponse {
	openaiResp := OpenAIChatResponse{
		ID:      qwenResp.ID,
		Object:  "chat.completion",
		Created: qwenResp.Created,
		Model:   qwenResp.Model,
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
			PromptTokens:     qwenResp.Usage.PromptTokens,
			CompletionTokens: qwenResp.Usage.CompletionTokens,
			TotalTokens:      qwenResp.Usage.TotalTokens,
		},
	}

	// Convert choices
	for _, choice := range qwenResp.Choices {
		openaiResp.Choices = append(openaiResp.Choices, struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{
			Index: choice.Index,
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
		})
	}

	return openaiResp
}

// ChatCompletion performs a non-streaming chat completion
func (q *QwenAdapter) ChatCompletion(ctx context.Context, request OpenAIChatRequest) (*OpenAIChatResponse, error) {
	// Refresh OAuth headers if needed
	if err := q.refreshAuthHeaders(); err != nil {
		return nil, fmt.Errorf("failed to refresh auth: %w", err)
	}

	qwenReq := q.convertToQwenRequest(request)

	// Prepare request body
	requestBody, err := json.Marshal(qwenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/services/aigc/text-generation/generation", q.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range q.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var qwenResp QwenChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&qwenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	openaiResp := q.convertFromQwenResponse(qwenResp)
	return &openaiResp, nil
}

// StreamChatCompletion streams a chat completion from Qwen
func (q *QwenAdapter) StreamChatCompletion(ctx context.Context, request OpenAIChatRequest) (<-chan OpenAIStreamResponse, <-chan error) {
	responseChan := make(chan OpenAIStreamResponse, 10)
	errorChan := make(chan error, 1)

	// Refresh OAuth headers if needed
	if err := q.refreshAuthHeaders(); err != nil {
		go func() {
			defer close(responseChan)
			defer close(errorChan)
			errorChan <- fmt.Errorf("failed to refresh auth: %w", err)
		}()
		return responseChan, errorChan
	}

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		qwenReq := q.convertToQwenRequest(request)
		qwenReq.Stream = true

		// Prepare request body
		requestBody, err := json.Marshal(qwenReq)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Create HTTP request
		url := fmt.Sprintf("%s/services/aigc/text-generation/generation", q.endpoint)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		// Set headers
		for key, value := range q.headers {
			req.Header.Set(key, value)
		}
		req.Header.Set("Accept", "text/event-stream")

		// Send request
		resp, err := q.client.Do(req)
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

				var qwenStream QwenStreamChunk
				if err := json.Unmarshal([]byte(data), &qwenStream); err != nil {
					continue // Skip malformed lines
				}

				// Convert to OpenAI format
				openaiStream := OpenAIStreamResponse{
					ID:      qwenStream.ID,
					Object:  "chat.completion.chunk",
					Created: qwenStream.Created,
					Model:   qwenStream.Model,
					Choices: []OpenAIChoice{},
				}

				for _, choice := range qwenStream.Choices {
					if choice.Delta.Content != "" {
						openaiStream.Choices = append(openaiStream.Choices, OpenAIChoice{
							Index: choice.Index,
							Delta: OpenAIDelta{
								Content: choice.Delta.Content,
							},
						})
					}
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

// ListModels retrieves available models from Qwen
func (q *QwenAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	// Qwen doesn't have a public models endpoint, so we return a curated list
	availableModels := q.discoverAvailableModels(ctx)

	modelsResp := &OpenAIModelsResponse{
		Object: "list",
		Data:   availableModels,
	}

	return modelsResp, nil
}

// discoverAvailableModels returns the list of known Qwen models
func (q *QwenAdapter) discoverAvailableModels(ctx context.Context) []struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
} {
	// Known Qwen models
	knownModels := []struct {
		ID      string
		Created int64
	}{
		{ID: "qwen-turbo", Created: 1704067200},          // Jan 2024
		{ID: "qwen-plus", Created: 1704067200},           // Jan 2024
		{ID: "qwen-max", Created: 1704067200},            // Jan 2024
		{ID: "qwen-max-longcontext", Created: 1704067200}, // Jan 2024
		{ID: "qwen-coder-turbo", Created: 1709251200},    // Feb 2024
		{ID: "qwen2.5-coder-32b-instruct", Created: 1729036800}, // Oct 2024
		{ID: "qwen2.5-72b-instruct", Created: 1729036800},       // Oct 2024
	}

	var availableModels []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	// Try to verify API access
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	apiAccessible := q.verifyAPIAccess(testCtx)

	for _, model := range knownModels {
		availableModels = append(availableModels, struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			ID:      model.ID,
			Object:  "model",
			Created: model.Created,
			OwnedBy: "alibaba",
		})
	}

	if !apiAccessible {
		log.Printf("Warning: Qwen API access could not be verified. Model list may be stale.")
	}

	return availableModels
}

// verifyAPIAccess checks if the API is accessible with current credentials
func (q *QwenAdapter) verifyAPIAccess(ctx context.Context) bool {
	// Refresh OAuth headers if needed
	if err := q.refreshAuthHeaders(); err != nil {
		return false
	}

	// Make a minimal request to verify API access
	req, err := http.NewRequestWithContext(ctx, "GET", q.endpoint+"/models", nil)
	if err != nil {
		return false
	}

	for key, value := range q.headers {
		req.Header.Set(key, value)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Any response means API is accessible
	return resp.StatusCode != 0
}
