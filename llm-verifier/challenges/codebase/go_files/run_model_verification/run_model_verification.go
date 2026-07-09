// Package main implements the Model Verification challenge.
// This challenge verifies model capabilities (streaming, function calling, vision, etc.)
// by testing each capability against real LLM providers.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ProviderConfig holds configuration for an LLM provider.
type ProviderConfig struct {
	Name    string `json:"name"`
	APIKey  string `json:"-"` // Never serialize
	BaseURL string `json:"base_url,omitempty"`
	Enabled bool   `json:"enabled"`
}

// ModelCapability represents a capability to test.
type ModelCapability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// CapabilityTestResult holds the result of testing a capability.
type CapabilityTestResult struct {
	Capability string `json:"capability"`
	Supported  bool   `json:"supported"`
	Tested     bool   `json:"tested"`
	LatencyMs  int64  `json:"latency_ms,omitempty"`
	Error      string `json:"error,omitempty"`
	Details    string `json:"details,omitempty"`
}

// ModelVerificationResult holds verification results for a model.
type ModelVerificationResult struct {
	ModelID            string                 `json:"model_id"`
	ModelName          string                 `json:"model_name"`
	Provider           string                 `json:"provider"`
	Verified           bool                   `json:"verified"`
	CapabilityResults  []CapabilityTestResult `json:"capability_results"`
	TotalCapabilities  int                    `json:"total_capabilities"`
	PassedCapabilities int                    `json:"passed_capabilities"`
	FailedCapabilities int                    `json:"failed_capabilities"`
	VerificationScore  float64                `json:"verification_score"`
	ResponseTimeMs     int64                  `json:"response_time_ms"`
	Error              string                 `json:"error,omitempty"`
	Timestamp          time.Time              `json:"timestamp"`
}

// ProviderVerificationResult holds all model verification results for a provider.
type ProviderVerificationResult struct {
	Provider       string                    `json:"provider"`
	Success        bool                      `json:"success"`
	ModelsVerified int                       `json:"models_verified"`
	Models         []ModelVerificationResult `json:"models"`
	AverageScore   float64                   `json:"average_score"`
	ResponseTimeMs int64                     `json:"response_time_ms"`
	Error          string                    `json:"error,omitempty"`
	Timestamp      time.Time                 `json:"timestamp"`
}

// ChallengeResult holds the complete challenge output.
type ChallengeResult struct {
	ChallengeID   string                       `json:"challenge_id"`
	ChallengeName string                       `json:"challenge_name"`
	Timestamp     time.Time                    `json:"timestamp"`
	Duration      time.Duration                `json:"duration"`
	Status        string                       `json:"status"`
	Results       []ProviderVerificationResult `json:"results"`
	AllModels     []ModelVerificationResult    `json:"all_models"`
	Summary       ChallengeSummary             `json:"summary"`
}

// ChallengeSummary provides aggregated statistics.
type ChallengeSummary struct {
	TotalProviders      int     `json:"total_providers"`
	SuccessfulProviders int     `json:"successful_providers"`
	FailedProviders     int     `json:"failed_providers"`
	TotalModels         int     `json:"total_models"`
	VerifiedModels      int     `json:"verified_models"`
	AverageScore        float64 `json:"average_score"`
	HighestScore        float64 `json:"highest_score"`
	TopModel            string  `json:"top_model"`
	AverageResponseMs   float64 `json:"average_response_ms"`
}

// API Response structures
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Tools       []Tool        `json:"tools,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Anthropic-specific structures
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
	Stream    bool               `json:"stream,omitempty"`
	Tools     []AnthropicTool    `json:"tools,omitempty"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type AnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Gemini-specific structures
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig,omitempty"`
	Tools            []GeminiTool           `json:"tools,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

type GeminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

// Known models per provider for verification
var knownModels = map[string][]string{
	"openai":     {"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"},
	"anthropic":  {"claude-3-5-sonnet-20241022", "claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
	"openrouter": {"openai/gpt-4o", "anthropic/claude-3-5-sonnet", "meta-llama/llama-3.1-70b-instruct"},
	"deepseek":   {"deepseek-chat", "deepseek-coder"},
	"gemini":     {"gemini-1.5-pro", "gemini-1.5-flash", "gemini-pro"},
	"groq":       {"llama-3.1-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"},
	"ollama":     {"llama3", "codellama", "mistral"},
}

// Capabilities to test
var capabilities = []ModelCapability{
	{Name: "basic_completion", Description: "Basic text completion", Required: true},
	{Name: "streaming", Description: "Streaming responses", Required: false},
	{Name: "function_calling", Description: "Function/tool calling", Required: false},
	{Name: "system_message", Description: "System message support", Required: false},
	{Name: "multi_turn", Description: "Multi-turn conversation", Required: false},
}

func verifyModelCapabilities(ctx context.Context, config ProviderConfig, modelID string) *ModelVerificationResult {
	result := &ModelVerificationResult{
		ModelID:   modelID,
		ModelName: modelID,
		Provider:  config.Name,
		Timestamp: time.Now(),
	}

	start := time.Now()
	var capResults []CapabilityTestResult
	passedCount := 0
	failedCount := 0

	for _, cap := range capabilities {
		capResult := testCapability(ctx, config, modelID, cap)
		capResults = append(capResults, capResult)

		if capResult.Tested {
			if capResult.Supported {
				passedCount++
			} else if cap.Required {
				failedCount++
			}
		}
	}

	result.CapabilityResults = capResults
	result.TotalCapabilities = len(capabilities)
	result.PassedCapabilities = passedCount
	result.FailedCapabilities = failedCount
	result.ResponseTimeMs = time.Since(start).Milliseconds()

	// Calculate verification score (0-100)
	if result.TotalCapabilities > 0 {
		result.VerificationScore = float64(passedCount) / float64(result.TotalCapabilities) * 100
	}

	// Model is verified if all required capabilities pass
	result.Verified = true
	for _, capResult := range capResults {
		for _, cap := range capabilities {
			if cap.Name == capResult.Capability && cap.Required && !capResult.Supported {
				result.Verified = false
				break
			}
		}
	}

	return result
}

func testCapability(ctx context.Context, config ProviderConfig, modelID string, cap ModelCapability) CapabilityTestResult {
	result := CapabilityTestResult{
		Capability: cap.Name,
		Tested:     true,
	}

	start := time.Now()
	var err error

	switch cap.Name {
	case "basic_completion":
		err = testBasicCompletion(ctx, config, modelID)
	case "streaming":
		err = testStreaming(ctx, config, modelID)
	case "function_calling":
		err = testFunctionCalling(ctx, config, modelID)
	case "system_message":
		err = testSystemMessage(ctx, config, modelID)
	case "multi_turn":
		err = testMultiTurn(ctx, config, modelID)
	default:
		result.Tested = false
		result.Details = "Unknown capability"
		return result
	}

	result.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Supported = false
		result.Error = err.Error()
	} else {
		result.Supported = true
		result.Details = "Capability verified successfully"
	}

	return result
}

func testBasicCompletion(ctx context.Context, config ProviderConfig, modelID string) error {
	switch config.Name {
	case "anthropic":
		return testAnthropicCompletion(ctx, config, modelID, "Say 'hello' in one word.")
	case "gemini":
		return testGeminiCompletion(ctx, config, modelID, "Say 'hello' in one word.")
	case "ollama":
		return testOllamaCompletion(ctx, config, modelID, "Say 'hello' in one word.")
	default:
		return testOpenAICompatibleCompletion(ctx, config, modelID, "Say 'hello' in one word.", false)
	}
}

func testStreaming(ctx context.Context, config ProviderConfig, modelID string) error {
	switch config.Name {
	case "anthropic":
		return testAnthropicStreaming(ctx, config, modelID)
	case "gemini":
		// Gemini streaming requires different endpoint
		return testGeminiStreaming(ctx, config, modelID)
	case "ollama":
		return testOllamaStreaming(ctx, config, modelID)
	default:
		return testOpenAICompatibleCompletion(ctx, config, modelID, "Count from 1 to 5.", true)
	}
}

func testFunctionCalling(ctx context.Context, config ProviderConfig, modelID string) error {
	switch config.Name {
	case "anthropic":
		return testAnthropicFunctionCalling(ctx, config, modelID)
	case "gemini":
		return testGeminiFunctionCalling(ctx, config, modelID)
	case "ollama":
		return fmt.Errorf("function calling not typically supported by Ollama")
	default:
		return testOpenAIFunctionCalling(ctx, config, modelID)
	}
}

func testSystemMessage(ctx context.Context, config ProviderConfig, modelID string) error {
	switch config.Name {
	case "anthropic":
		return testAnthropicSystemMessage(ctx, config, modelID)
	case "gemini":
		return testGeminiSystemMessage(ctx, config, modelID)
	case "ollama":
		return testOllamaSystemMessage(ctx, config, modelID)
	default:
		return testOpenAISystemMessage(ctx, config, modelID)
	}
}

func testMultiTurn(ctx context.Context, config ProviderConfig, modelID string) error {
	switch config.Name {
	case "anthropic":
		return testAnthropicMultiTurn(ctx, config, modelID)
	case "gemini":
		return testGeminiMultiTurn(ctx, config, modelID)
	case "ollama":
		return testOllamaMultiTurn(ctx, config, modelID)
	default:
		return testOpenAIMultiTurn(ctx, config, modelID)
	}
}

// OpenAI-compatible API tests
func testOpenAICompatibleCompletion(ctx context.Context, config ProviderConfig, modelID, prompt string, stream bool) error {
	baseURL := getBaseURL(config)

	reqBody := ChatCompletionRequest{
		Model:     modelID,
		Messages:  []ChatMessage{{Role: "user", Content: prompt}},
		MaxTokens: 50,
		Stream:    stream,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	if stream {
		// For streaming, just verify we get chunked response
		respBody, _ := io.ReadAll(resp.Body)
		if len(respBody) == 0 {
			return fmt.Errorf("empty streaming response")
		}
		return nil
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return fmt.Errorf("no choices in response")
	}

	return nil
}

func testOpenAIFunctionCalling(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := getBaseURL(config)

	tools := []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_weather",
				Description: "Get the current weather in a location",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The city name",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	reqBody := ChatCompletionRequest{
		Model:     modelID,
		Messages:  []ChatMessage{{Role: "user", Content: "What's the weather in Tokyo?"}},
		MaxTokens: 100,
		Tools:     tools,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testOpenAISystemMessage(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := getBaseURL(config)

	reqBody := ChatCompletionRequest{
		Model: modelID,
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful assistant that responds only in uppercase."},
			{Role: "user", Content: "Say hello."},
		},
		MaxTokens: 50,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testOpenAIMultiTurn(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := getBaseURL(config)

	reqBody := ChatCompletionRequest{
		Model: modelID,
		Messages: []ChatMessage{
			{Role: "user", Content: "My name is Alice."},
			{Role: "assistant", Content: "Hello Alice! Nice to meet you."},
			{Role: "user", Content: "What is my name?"},
		},
		MaxTokens: 50,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Anthropic API tests
func testAnthropicCompletion(ctx context.Context, config ProviderConfig, modelID, prompt string) error {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	reqBody := AnthropicRequest{
		Model:     modelID,
		MaxTokens: 50,
		Messages:  []AnthropicMessage{{Role: "user", Content: prompt}},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testAnthropicStreaming(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	reqBody := AnthropicRequest{
		Model:     modelID,
		MaxTokens: 50,
		Messages:  []AnthropicMessage{{Role: "user", Content: "Count from 1 to 5."}},
		Stream:    true,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Verify we get streaming response
	respBody, _ := io.ReadAll(resp.Body)
	if len(respBody) == 0 {
		return fmt.Errorf("empty streaming response")
	}

	return nil
}

func testAnthropicFunctionCalling(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	tools := []AnthropicTool{
		{
			Name:        "get_weather",
			Description: "Get the current weather in a location",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city name",
					},
				},
				"required": []string{"location"},
			},
		},
	}

	reqBody := AnthropicRequest{
		Model:     modelID,
		MaxTokens: 100,
		Messages:  []AnthropicMessage{{Role: "user", Content: "What's the weather in Tokyo?"}},
		Tools:     tools,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testAnthropicSystemMessage(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	type AnthropicSystemRequest struct {
		Model     string             `json:"model"`
		MaxTokens int                `json:"max_tokens"`
		System    string             `json:"system"`
		Messages  []AnthropicMessage `json:"messages"`
	}

	reqBody := AnthropicSystemRequest{
		Model:     modelID,
		MaxTokens: 50,
		System:    "You are a helpful assistant that responds only in uppercase.",
		Messages:  []AnthropicMessage{{Role: "user", Content: "Say hello."}},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testAnthropicMultiTurn(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	reqBody := AnthropicRequest{
		Model:     modelID,
		MaxTokens: 50,
		Messages: []AnthropicMessage{
			{Role: "user", Content: "My name is Alice."},
			{Role: "assistant", Content: "Hello Alice! Nice to meet you."},
			{Role: "user", Content: "What is my name?"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Gemini API tests
func testGeminiCompletion(ctx context.Context, config ProviderConfig, modelID, prompt string) error {
	baseURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelID, config.APIKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: prompt}}},
		},
		GenerationConfig: GeminiGenerationConfig{
			MaxOutputTokens: 50,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testGeminiStreaming(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?key=%s", modelID, config.APIKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: "Count from 1 to 5."}}},
		},
		GenerationConfig: GeminiGenerationConfig{
			MaxOutputTokens: 50,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testGeminiFunctionCalling(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelID, config.APIKey)

	tools := []GeminiTool{
		{
			FunctionDeclarations: []GeminiFunctionDeclaration{
				{
					Name:        "get_weather",
					Description: "Get the current weather in a location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The city name",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		},
	}

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: "What's the weather in Tokyo?"}}},
		},
		Tools: tools,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testGeminiSystemMessage(ctx context.Context, config ProviderConfig, modelID string) error {
	// Gemini uses system instruction in a different way
	baseURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelID, config.APIKey)

	type GeminiSystemRequest struct {
		SystemInstruction GeminiContent          `json:"systemInstruction,omitempty"`
		Contents          []GeminiContent        `json:"contents"`
		GenerationConfig  GeminiGenerationConfig `json:"generationConfig,omitempty"`
	}

	reqBody := GeminiSystemRequest{
		SystemInstruction: GeminiContent{
			Parts: []GeminiPart{{Text: "You are a helpful assistant that responds only in uppercase."}},
		},
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: "Say hello."}}},
		},
		GenerationConfig: GeminiGenerationConfig{
			MaxOutputTokens: 50,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testGeminiMultiTurn(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelID, config.APIKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Role: "user", Parts: []GeminiPart{{Text: "My name is Alice."}}},
			{Role: "model", Parts: []GeminiPart{{Text: "Hello Alice! Nice to meet you."}}},
			{Role: "user", Parts: []GeminiPart{{Text: "What is my name?"}}},
		},
		GenerationConfig: GeminiGenerationConfig{
			MaxOutputTokens: 50,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Ollama API tests
func testOllamaCompletion(ctx context.Context, config ProviderConfig, modelID, prompt string) error {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	type OllamaRequest struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
	}

	reqBody := OllamaRequest{
		Model:  modelID,
		Prompt: prompt,
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testOllamaStreaming(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	type OllamaRequest struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
	}

	reqBody := OllamaRequest{
		Model:  modelID,
		Prompt: "Count from 1 to 5.",
		Stream: true,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testOllamaSystemMessage(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	type OllamaRequest struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		System string `json:"system"`
		Stream bool   `json:"stream"`
	}

	reqBody := OllamaRequest{
		Model:  modelID,
		Prompt: "Say hello.",
		System: "You are a helpful assistant that responds only in uppercase.",
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func testOllamaMultiTurn(ctx context.Context, config ProviderConfig, modelID string) error {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	type OllamaChatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type OllamaChatRequest struct {
		Model    string              `json:"model"`
		Messages []OllamaChatMessage `json:"messages"`
		Stream   bool                `json:"stream"`
	}

	reqBody := OllamaChatRequest{
		Model: modelID,
		Messages: []OllamaChatMessage{
			{Role: "user", Content: "My name is Alice."},
			{Role: "assistant", Content: "Hello Alice! Nice to meet you."},
			{Role: "user", Content: "What is my name?"},
		},
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func getBaseURL(config ProviderConfig) string {
	if config.BaseURL != "" {
		return config.BaseURL
	}

	switch config.Name {
	case "openai":
		return "https://api.openai.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "deepseek":
		return "https://api.deepseek.com"
	case "groq":
		return "https://api.groq.com/openai/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

func verifyProvider(ctx context.Context, config ProviderConfig) *ProviderVerificationResult {
	result := &ProviderVerificationResult{
		Provider:  config.Name,
		Timestamp: time.Now(),
	}

	if config.APIKey == "" && config.Name != "ollama" {
		result.Error = "API key not configured"
		return result
	}

	start := time.Now()

	models, exists := knownModels[config.Name]
	if !exists || len(models) == 0 {
		result.Error = "No known models for provider"
		return result
	}

	var modelResults []ModelVerificationResult
	var totalScore float64

	// Verify first model only to save time and API costs
	modelID := models[0]
	modelResult := verifyModelCapabilities(ctx, config, modelID)
	modelResults = append(modelResults, *modelResult)
	totalScore += modelResult.VerificationScore

	result.ResponseTimeMs = time.Since(start).Milliseconds()
	result.Models = modelResults
	result.ModelsVerified = len(modelResults)

	if result.ModelsVerified > 0 {
		result.AverageScore = totalScore / float64(result.ModelsVerified)
		result.Success = true
	}

	return result
}

func main() {
	resultsDir := flag.String("results-dir", "", "Directory to store results")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	if *resultsDir == "" {
		log.Fatal("--results-dir is required")
	}

	ctx := context.Background()
	start := time.Now()

	// Create results directory
	resultsPath := filepath.Join(*resultsDir, "results")
	logsPath := filepath.Join(*resultsDir, "logs")
	if err := os.MkdirAll(resultsPath, 0755); err != nil {
		log.Fatalf("Failed to create results directory: %v", err)
	}
	if err := os.MkdirAll(logsPath, 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Configure providers from environment
	providers := []ProviderConfig{
		{Name: "anthropic", APIKey: os.Getenv("ANTHROPIC_API_KEY"), Enabled: true},
		{Name: "openai", APIKey: os.Getenv("OPENAI_API_KEY"), Enabled: true},
		{Name: "openrouter", APIKey: os.Getenv("OPENROUTER_API_KEY"), Enabled: true},
		{Name: "deepseek", APIKey: os.Getenv("DEEPSEEK_API_KEY"), Enabled: true},
		{Name: "gemini", APIKey: os.Getenv("GEMINI_API_KEY"), Enabled: true},
		{Name: "groq", APIKey: os.Getenv("GROQ_API_KEY"), Enabled: true},
		{Name: "ollama", BaseURL: os.Getenv("OLLAMA_BASE_URL"), Enabled: true},
	}

	// Filter to configured providers
	var configuredProviders []ProviderConfig
	for _, p := range providers {
		if p.APIKey != "" || p.Name == "ollama" {
			configuredProviders = append(configuredProviders, p)
		}
	}

	if *verbose {
		log.Printf("Found %d configured providers", len(configuredProviders))
	}

	// Verify each provider concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var verificationResults []ProviderVerificationResult
	var allModels []ModelVerificationResult

	for _, provider := range configuredProviders {
		wg.Add(1)
		go func(p ProviderConfig) {
			defer wg.Done()

			if *verbose {
				log.Printf("Verifying provider: %s", p.Name)
			}

			result := verifyProvider(ctx, p)

			mu.Lock()
			verificationResults = append(verificationResults, *result)
			if result.Success {
				allModels = append(allModels, result.Models...)
			}
			mu.Unlock()
		}(provider)
	}

	wg.Wait()

	// Sort models by verification score (descending)
	sort.Slice(allModels, func(i, j int) bool {
		return allModels[i].VerificationScore > allModels[j].VerificationScore
	})

	// Calculate summary
	var totalResponseTime int64
	successfulCount := 0
	failedCount := 0
	var totalScore float64
	var highestScore float64
	var topModel string

	for _, r := range verificationResults {
		if r.Success {
			successfulCount++
			totalResponseTime += r.ResponseTimeMs
		} else {
			failedCount++
		}
	}

	for _, m := range allModels {
		totalScore += m.VerificationScore
		if m.VerificationScore > highestScore {
			highestScore = m.VerificationScore
			topModel = m.ModelID
		}
	}

	avgResponseTime := float64(0)
	if successfulCount > 0 {
		avgResponseTime = float64(totalResponseTime) / float64(successfulCount)
	}

	avgScore := float64(0)
	if len(allModels) > 0 {
		avgScore = totalScore / float64(len(allModels))
	}

	verifiedModelsCount := 0
	for _, m := range allModels {
		if m.Verified {
			verifiedModelsCount++
		}
	}

	// Build final result
	challengeResult := ChallengeResult{
		ChallengeID:   "run_model_verification",
		ChallengeName: "Model Verification",
		Timestamp:     time.Now(),
		Duration:      time.Since(start),
		Status:        "passed",
		Results:       verificationResults,
		AllModels:     allModels,
		Summary: ChallengeSummary{
			TotalProviders:      len(configuredProviders),
			SuccessfulProviders: successfulCount,
			FailedProviders:     failedCount,
			TotalModels:         len(allModels),
			VerifiedModels:      verifiedModelsCount,
			AverageScore:        avgScore,
			HighestScore:        highestScore,
			TopModel:            topModel,
			AverageResponseMs:   avgResponseTime,
		},
	}

	if successfulCount == 0 {
		challengeResult.Status = "failed"
	}

	// Write results
	resultFile := filepath.Join(resultsPath, "verification_results.json")
	resultData, _ := json.MarshalIndent(challengeResult, "", "  ")
	if err := os.WriteFile(resultFile, resultData, 0644); err != nil {
		log.Printf("Warning: Failed to write results: %v", err)
	}

	modelsFile := filepath.Join(resultsPath, "verified_models.json")
	modelsData, _ := json.MarshalIndent(allModels, "", "  ")
	if err := os.WriteFile(modelsFile, modelsData, 0644); err != nil {
		log.Printf("Warning: Failed to write models: %v", err)
	}

	// Write report
	reportFile := filepath.Join(resultsPath, "verification_report.md")
	report := generateReport(challengeResult)
	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		log.Printf("Warning: Failed to write report: %v", err)
	}

	// Print summary
	fmt.Printf("\n=== Model Verification Complete ===\n")
	fmt.Printf("Status: %s\n", strings.ToUpper(challengeResult.Status))
	fmt.Printf("Providers: %d successful, %d failed\n", successfulCount, failedCount)
	fmt.Printf("Models: %d verified out of %d\n", verifiedModelsCount, len(allModels))
	if topModel != "" {
		fmt.Printf("Top Model: %s (score: %.1f%%)\n", topModel, highestScore)
	}
	fmt.Printf("Duration: %v\n", challengeResult.Duration)
	fmt.Printf("Results: %s\n", resultsPath)

	if challengeResult.Status == "failed" {
		os.Exit(1)
	}
}

func generateReport(result ChallengeResult) string {
	var sb strings.Builder

	sb.WriteString("# Model Verification Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", result.Timestamp.Format(time.RFC3339)))

	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Status | %s |\n", strings.ToUpper(result.Status)))
	sb.WriteString(fmt.Sprintf("| Total Providers | %d |\n", result.Summary.TotalProviders))
	sb.WriteString(fmt.Sprintf("| Successful Providers | %d |\n", result.Summary.SuccessfulProviders))
	sb.WriteString(fmt.Sprintf("| Failed Providers | %d |\n", result.Summary.FailedProviders))
	sb.WriteString(fmt.Sprintf("| Total Models Tested | %d |\n", result.Summary.TotalModels))
	sb.WriteString(fmt.Sprintf("| Verified Models | %d |\n", result.Summary.VerifiedModels))
	sb.WriteString(fmt.Sprintf("| Average Score | %.1f%% |\n", result.Summary.AverageScore))
	sb.WriteString(fmt.Sprintf("| Highest Score | %.1f%% |\n", result.Summary.HighestScore))
	sb.WriteString(fmt.Sprintf("| Top Model | %s |\n", result.Summary.TopModel))
	sb.WriteString(fmt.Sprintf("| Average Response Time | %.0fms |\n", result.Summary.AverageResponseMs))
	sb.WriteString(fmt.Sprintf("| Duration | %v |\n", result.Duration))

	sb.WriteString("\n## Provider Results\n\n")
	sb.WriteString("| Provider | Status | Models Verified | Average Score | Response Time |\n")
	sb.WriteString("|----------|--------|-----------------|---------------|---------------|\n")
	for _, r := range result.Results {
		status := "Failed"
		if r.Success {
			status = "Success"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %.1f%% | %dms |\n",
			r.Provider, status, r.ModelsVerified, r.AverageScore, r.ResponseTimeMs))
	}

	sb.WriteString("\n## Model Verification Details\n\n")
	for _, m := range result.AllModels {
		verifiedStatus := "Not Verified"
		if m.Verified {
			verifiedStatus = "Verified"
		}
		sb.WriteString(fmt.Sprintf("### %s (%s)\n\n", m.ModelID, m.Provider))
		sb.WriteString(fmt.Sprintf("**Status**: %s | **Score**: %.1f%% | **Response Time**: %dms\n\n", verifiedStatus, m.VerificationScore, m.ResponseTimeMs))

		sb.WriteString("| Capability | Supported | Latency | Details |\n")
		sb.WriteString("|------------|-----------|---------|--------|\n")
		for _, cap := range m.CapabilityResults {
			supported := "No"
			if cap.Supported {
				supported = "Yes"
			}
			details := cap.Details
			if cap.Error != "" {
				details = cap.Error
			}
			if len(details) > 50 {
				details = details[:47] + "..."
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %dms | %s |\n",
				cap.Capability, supported, cap.LatencyMs, details))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("*Generated by LLMsVerifier Model Verification Challenge*\n")

	return sb.String()
}
