package main

import (
	"context"
	"testing"
	"time"

	"digital.vasic.llmsverifier/llmverifier"
)

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{"OpenAI GPT-4", "gpt-4", "openai"},
		{"OpenAI GPT-4o", "gpt-4o", "openai"},
		{"OpenAI o1", "o1-preview", "openai"},
		{"Claude model", "claude-3-opus", "anthropic"},
		{"Claude Sonnet", "claude-3-5-sonnet-latest", "anthropic"},
		{"DeepSeek chat", "deepseek-chat", "deepseek"},
		{"DeepSeek coder", "deepseek-coder", "deepseek"},
		{"Gemini", "gemini-pro", "google"},
		{"Unknown defaults to OpenAI", "unknown-model", "openai"},
		{"Llama via OpenAI", "llama-2-70b", "openai"},
		{"Case insensitive Claude", "CLAUDE-3-OPUS", "anthropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectProvider(tt.model)
			if result != tt.expected {
				t.Errorf("detectProvider(%q) = %q, want %q", tt.model, result, tt.expected)
			}
		})
	}
}

func TestGetKnownModelsForProvider(t *testing.T) {
	tests := []struct {
		provider    string
		minExpected int // At least this many models
	}{
		{"openai", 5},
		{"anthropic", 3},
		{"deepseek", 2},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			models := getKnownModelsForProvider(tt.provider)
			if len(models) < tt.minExpected {
				t.Errorf("getKnownModelsForProvider(%q) returned %d models, want at least %d",
					tt.provider, len(models), tt.minExpected)
			}
		})
	}
}

func TestCalculateVerificationScore_NotSupported(t *testing.T) {
	// When ACP is not supported, score should be 0
	score := calculateVerificationScore(false, nil, "test-model", context.Background())
	if score != 0.0 {
		t.Errorf("calculateVerificationScore(false, ...) = %f, want 0.0", score)
	}
}

func TestCalculateVerificationScore_Supported(t *testing.T) {
	// When ACP is supported but client is nil, should return base score
	score := calculateVerificationScore(true, nil, "test-model", context.Background())
	if score != 50.0 {
		t.Errorf("calculateVerificationScore(true, nil, ...) = %f, want 50.0", score)
	}
}

func TestBatchResult_Fields(t *testing.T) {
	result := BatchResult{
		Model:     "gpt-4",
		Provider:  "openai",
		Supported: true,
		Score:     85.0,
		Duration:  2 * time.Second,
	}

	if result.Model != "gpt-4" {
		t.Errorf("BatchResult.Model = %q, want %q", result.Model, "gpt-4")
	}
	if result.Provider != "openai" {
		t.Errorf("BatchResult.Provider = %q, want %q", result.Provider, "openai")
	}
	if !result.Supported {
		t.Error("BatchResult.Supported = false, want true")
	}
	if result.Score != 85.0 {
		t.Errorf("BatchResult.Score = %f, want 85.0", result.Score)
	}
}

func TestProviderModels_Fields(t *testing.T) {
	pm := ProviderModels{
		Provider: "openai",
		Models:   []string{"gpt-4", "gpt-3.5-turbo"},
		Error:    "",
	}

	if pm.Provider != "openai" {
		t.Errorf("ProviderModels.Provider = %q, want %q", pm.Provider, "openai")
	}
	if len(pm.Models) != 2 {
		t.Errorf("len(ProviderModels.Models) = %d, want 2", len(pm.Models))
	}
	if pm.Error != "" {
		t.Errorf("ProviderModels.Error = %q, want empty", pm.Error)
	}
}

func TestProviderModels_WithError(t *testing.T) {
	pm := ProviderModels{
		Provider: "anthropic",
		Models:   []string{},
		Error:    "API key not found",
	}

	if pm.Error == "" {
		t.Error("ProviderModels.Error should not be empty")
	}
	if len(pm.Models) != 0 {
		t.Errorf("len(ProviderModels.Models) = %d, want 0 when error present", len(pm.Models))
	}
}

// MockClient for testing
type MockLLMClient struct {
	ShouldFail bool
}

func (m *MockLLMClient) ChatCompletion(ctx context.Context, req llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	if m.ShouldFail {
		return nil, context.DeadlineExceeded
	}
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.ChatCompletionChoice{
			{
				Message: llmverifier.Message{
					Role:    "assistant",
					Content: "verified",
				},
			},
		},
	}, nil
}

func TestACPMockClient_ChatCompletion(t *testing.T) {
	client := &ACPMockClient{Provider: "test-provider"}
	ctx := context.Background()

	req := llmverifier.ChatCompletionRequest{
		Model: "test-model",
		Messages: []llmverifier.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("Expected at least one choice in response")
	}

	if resp.Choices[0].Message.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got %q", resp.Choices[0].Message.Role)
	}
}
