package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests updated for 2026-02 model lists

func TestFallbackModels_OpenAI_Details(t *testing.T) {
	models := GetFallbackModels("openai")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 5) // Updated: OpenAI has gpt-4o, gpt-4o-mini, gpt-4-turbo, gpt-4, gpt-3.5-turbo, o1-preview, o1-mini

	// Check model IDs - all these should be present
	modelIDs := make([]string, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
	}
	assert.Contains(t, modelIDs, "gpt-4")
	assert.Contains(t, modelIDs, "gpt-4o")
	assert.Contains(t, modelIDs, "gpt-3.5-turbo")

	// Check model properties
	for _, m := range models {
		assert.Equal(t, "openai", m.ProviderID)
		assert.Equal(t, "OpenAI", m.ProviderName)
		assert.Greater(t, m.MaxTokens, 0)
	}
}

func TestFallbackModels_Anthropic_Details(t *testing.T) {
	models := GetFallbackModels("anthropic")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 3) // Updated: Anthropic has claude-3.5-sonnet, claude-3.5-haiku, claude-3-opus, etc.

	// Check for Claude model presence
	modelIDs := make([]string, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
	}
	assert.Contains(t, modelIDs, "claude-opus-4-6")
	assert.Contains(t, modelIDs, "claude-sonnet-4-6")
	assert.Contains(t, modelIDs, "claude-3-5-sonnet-20241022")

	for _, m := range models {
		assert.Equal(t, "anthropic", m.ProviderID)
		assert.Equal(t, "Anthropic", m.ProviderName)
		assert.Equal(t, 200000, m.MaxTokens)
	}
}

func TestGetFallbackModels_Groq(t *testing.T) {
	models := GetFallbackModels("groq")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 3) // Updated: Groq has llama-3.3-70b, llama-3.1-70b, llama-3.1-8b, mixtral, gemma

	for _, m := range models {
		assert.Equal(t, "groq", m.ProviderID)
		assert.Equal(t, "Groq", m.ProviderName)
	}
}

func TestGetFallbackModels_Gemini(t *testing.T) {
	models := GetFallbackModels("gemini")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 7) // Updated: Gemini has 7 current models

	// Check for at least one Gemini model
	modelIDs := make([]string, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
	}
	assert.Contains(t, modelIDs, "gemini-2.5-flash")
}

func TestGetFallbackModels_DeepSeek(t *testing.T) {
	models := GetFallbackModels("deepseek")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 2) // Updated: DeepSeek has deepseek-chat, deepseek-coder, deepseek-reasoner

	modelIDs := make([]string, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
	}
	assert.Contains(t, modelIDs, "deepseek-chat")
}

func TestGetFallbackModels_NVIDIA(t *testing.T) {
	models := GetFallbackModels("nvidia")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 1)

	for _, m := range models {
		assert.Equal(t, "nvidia", m.ProviderID)
		assert.Equal(t, "NVIDIA", m.ProviderName)
	}
}

func TestGetFallbackModels_OpenRouter(t *testing.T) {
	models := GetFallbackModels("openrouter")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 2)

	for _, m := range models {
		assert.Equal(t, "openrouter", m.ProviderID)
		assert.Equal(t, "OpenRouter", m.ProviderName)
	}
}

func TestGetFallbackModels_Together(t *testing.T) {
	models := GetFallbackModels("together")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 2) // Updated: Together has llama-3.1-405b, llama-3.1-70b, mixtral

	for _, m := range models {
		assert.Equal(t, "together", m.ProviderID)
		assert.Equal(t, "Together AI", m.ProviderName)
	}
}

func TestGetFallbackModels_Mistral(t *testing.T) {
	models := GetFallbackModels("mistral")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 3) // Updated: Mistral has large, medium, small, codestral

	modelIDs := make([]string, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
	}
	assert.Contains(t, modelIDs, "mistral-large-latest")
	assert.Contains(t, modelIDs, "mistral-small-latest")
}

func TestGetFallbackModels_Fireworks(t *testing.T) {
	models := GetFallbackModels("fireworks")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 2) // Updated: Fireworks has llama-3.1-405b, llama-3.1-70b, mixtral

	for _, m := range models {
		assert.Equal(t, "fireworks", m.ProviderID)
		assert.Equal(t, "Fireworks", m.ProviderName)
	}
}

func TestGetFallbackModels_Perplexity(t *testing.T) {
	models := GetFallbackModels("perplexity")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 2) // Updated: Perplexity has sonar-huge, sonar-large, sonar-small

	for _, m := range models {
		assert.Equal(t, "perplexity", m.ProviderID)
		assert.Equal(t, "Perplexity", m.ProviderName)
	}
}

func TestGetFallbackModels_HuggingFace(t *testing.T) {
	models := GetFallbackModels("huggingface")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 1)

	for _, m := range models {
		assert.Equal(t, "huggingface", m.ProviderID)
		assert.Equal(t, "Hugging Face", m.ProviderName)
	}
}

func TestGetFallbackModels_Cohere(t *testing.T) {
	models := GetFallbackModels("cohere")

	require.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 3) // Cohere has command-r-plus, command-r, command, command-light

	modelIDs := make([]string, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
	}
	assert.Contains(t, modelIDs, "command-r-plus")
	assert.Contains(t, modelIDs, "command")
}

func TestGetFallbackModels_UnknownProvider(t *testing.T) {
	models := GetFallbackModels("unknown-provider")

	require.NotEmpty(t, models)
	assert.Len(t, models, 1)

	// Should return a generic fallback model
	assert.Equal(t, "unknown-provider-model", models[0].ID)
	assert.Equal(t, "unknown-provider", models[0].ProviderID)
	assert.Equal(t, 4096, models[0].MaxTokens)
}

func TestGetFallbackModels_EmptyProvider(t *testing.T) {
	models := GetFallbackModels("")

	require.NotEmpty(t, models)
	assert.Len(t, models, 1)
	assert.Equal(t, "-model", models[0].ID) // Empty provider ID
}

func TestGetFallbackModels_AllProvidersHaveValidModels(t *testing.T) {
	providers := []string{
		"openai", "anthropic", "groq", "gemini", "deepseek",
		"nvidia", "openrouter", "together", "mistral",
		"fireworks", "perplexity", "huggingface", "cohere",
	}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			models := GetFallbackModels(provider)
			require.NotEmpty(t, models, "Provider %s should have models", provider)

			for _, m := range models {
				assert.NotEmpty(t, m.ID, "Model should have ID")
				assert.NotEmpty(t, m.Name, "Model should have Name")
				assert.Equal(t, provider, m.ProviderID, "Model should have correct ProviderID")
				assert.NotEmpty(t, m.ProviderName, "Model should have ProviderName")
				assert.Greater(t, m.MaxTokens, 0, "Model should have positive MaxTokens")
			}
		})
	}
}

func TestModelCache_SetAndGet(t *testing.T) {
	testModels := []Model{
		{ID: "test-model-1", Name: "Test Model 1", ProviderID: "test", ProviderName: "Test", MaxTokens: 4096},
		{ID: "test-model-2", Name: "Test Model 2", ProviderID: "test", ProviderName: "Test", MaxTokens: 8192},
	}

	// Set cached models
	SetCachedModels("test-provider", testModels)

	// Get cached models
	cached, ok := GetCachedModels("test-provider")
	require.True(t, ok, "Should find cached models")
	assert.Equal(t, testModels, cached)
}

func TestModelCache_NotFound(t *testing.T) {
	cached, ok := GetCachedModels("nonexistent-provider")
	assert.False(t, ok, "Should not find cached models for nonexistent provider")
	assert.Nil(t, cached)
}

func TestFallbackModelsVersion(t *testing.T) {
	assert.NotEmpty(t, FallbackModelsVersion)
	assert.Equal(t, "2026-02", FallbackModelsVersion)
}
