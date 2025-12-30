package providers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-verifier/logging"
)

func createTestLogger(t *testing.T) *logging.Logger {
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": "error"})
	require.NoError(t, err)
	return logger
}

func TestNewRelaxedVerificationService(t *testing.T) {
	logger := createTestLogger(t)
	service := NewRelaxedVerificationService(logger)

	require.NotNil(t, service)
	assert.NotNil(t, service.logger)
}

func TestRelaxedVerificationService_VerifyModelRelaxed_ReputableProviders(t *testing.T) {
	logger := createTestLogger(t)
	service := NewRelaxedVerificationService(logger)
	ctx := context.Background()

	reputableProviders := []string{
		"openai", "anthropic", "huggingface", "groq", "gemini", "deepseek",
		"nvidia", "openrouter", "replicate", "fireworks", "together",
		"perplexity", "mistral", "cloudflare", "sambanova", "cerebras",
		"modal", "inference", "siliconflow", "novita", "upstage",
		"nlpcloud", "hyperbolic", "chutes", "kimi",
	}

	for _, provider := range reputableProviders {
		t.Run(provider, func(t *testing.T) {
			model := Model{
				ID:         "test-model",
				Name:       "Test Model",
				ProviderID: provider,
			}

			result := service.VerifyModelRelaxed(ctx, model, nil)

			assert.True(t, result, "Model from reputable provider %s should be verified", provider)
		})
	}
}

func TestRelaxedVerificationService_VerifyModelRelaxed_UnknownProvider(t *testing.T) {
	logger := createTestLogger(t)
	service := NewRelaxedVerificationService(logger)
	ctx := context.Background()

	model := Model{
		ID:         "test-model",
		Name:       "Test Model",
		ProviderID: "unknown-provider",
	}

	result := service.VerifyModelRelaxed(ctx, model, nil)

	// Relaxed verification is permissive - should still return true
	assert.True(t, result)
}

func TestRelaxedVerificationService_VerifyModelRelaxed_EmptyProvider(t *testing.T) {
	logger := createTestLogger(t)
	service := NewRelaxedVerificationService(logger)
	ctx := context.Background()

	model := Model{
		ID:         "test-model",
		Name:       "Test Model",
		ProviderID: "",
	}

	result := service.VerifyModelRelaxed(ctx, model, nil)

	// Should still be true due to permissive nature
	assert.True(t, result)
}

func TestRelaxedVerificationService_VerifyModelRelaxed_WithProviderClient(t *testing.T) {
	logger := createTestLogger(t)
	service := NewRelaxedVerificationService(logger)
	ctx := context.Background()

	model := Model{
		ID:         "gpt-4",
		Name:       "GPT-4",
		ProviderID: "openai",
	}

	// Provider client is not used in current implementation but should not cause errors
	client := &ProviderClient{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
	}

	result := service.VerifyModelRelaxed(ctx, model, client)

	assert.True(t, result)
}

func TestRelaxedVerificationService_VerifyModelRelaxed_AllModelsVerified(t *testing.T) {
	logger := createTestLogger(t)
	service := NewRelaxedVerificationService(logger)
	ctx := context.Background()

	testModels := []Model{
		{ID: "gpt-4", ProviderID: "openai"},
		{ID: "claude-3", ProviderID: "anthropic"},
		{ID: "llama-2", ProviderID: "unknown"},
		{ID: "custom-model", ProviderID: "custom-provider"},
		{ID: "local-model", ProviderID: "local"},
	}

	for _, model := range testModels {
		result := service.VerifyModelRelaxed(ctx, model, nil)
		assert.True(t, result, "All models should be verified in relaxed mode: %s/%s", model.ProviderID, model.ID)
	}
}

func TestRelaxedVerificationService_VerifyModelRelaxed_CanceledContext(t *testing.T) {
	logger := createTestLogger(t)
	service := NewRelaxedVerificationService(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	model := Model{
		ID:         "test-model",
		ProviderID: "openai",
	}

	// Should still work even with canceled context since it's synchronous
	result := service.VerifyModelRelaxed(ctx, model, nil)

	assert.True(t, result)
}
