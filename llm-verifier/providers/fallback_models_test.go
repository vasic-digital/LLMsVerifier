package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallbackModels_OpenAI_Details(t *testing.T) {
	models := GetFallbackModels("openai")

	require.NotEmpty(t, models)
	assert.Len(t, models, 3)

	// Check model IDs
	modelIDs := make([]string, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
	}
	assert.Contains(t, modelIDs, "gpt-4")
	assert.Contains(t, modelIDs, "gpt-4-turbo")
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
	assert.Len(t, models, 2)

	for _, m := range models {
		assert.Equal(t, "anthropic", m.ProviderID)
		assert.Equal(t, "Anthropic", m.ProviderName)
		assert.Equal(t, 200000, m.MaxTokens)
	}
}

func TestGetFallbackModels_Groq(t *testing.T) {
	models := GetFallbackModels("groq")

	require.NotEmpty(t, models)
	assert.Len(t, models, 2)

	for _, m := range models {
		assert.Equal(t, "groq", m.ProviderID)
		assert.Equal(t, "Groq", m.ProviderName)
	}
}

func TestGetFallbackModels_Gemini(t *testing.T) {
	models := GetFallbackModels("gemini")

	require.NotEmpty(t, models)
	assert.Len(t, models, 1)
	assert.Equal(t, "gemini-pro", models[0].ID)
	assert.Equal(t, 30720, models[0].MaxTokens)
}

func TestGetFallbackModels_DeepSeek(t *testing.T) {
	models := GetFallbackModels("deepseek")

	require.NotEmpty(t, models)
	assert.Len(t, models, 1)
	assert.Equal(t, "deepseek-chat", models[0].ID)
}

func TestGetFallbackModels_NVIDIA(t *testing.T) {
	models := GetFallbackModels("nvidia")

	require.NotEmpty(t, models)
	assert.Len(t, models, 1)
	assert.Equal(t, "llama2-70b", models[0].ID)
	assert.Equal(t, "NVIDIA", models[0].ProviderName)
}

func TestGetFallbackModels_OpenRouter(t *testing.T) {
	models := GetFallbackModels("openrouter")

	require.NotEmpty(t, models)
	assert.Len(t, models, 2)

	for _, m := range models {
		assert.Equal(t, "openrouter", m.ProviderID)
		assert.Equal(t, "OpenRouter", m.ProviderName)
	}
}

func TestGetFallbackModels_Together(t *testing.T) {
	models := GetFallbackModels("together")

	require.NotEmpty(t, models)
	assert.Len(t, models, 1)
	assert.Equal(t, "llama-2-70b-chat", models[0].ID)
}

func TestGetFallbackModels_Mistral(t *testing.T) {
	models := GetFallbackModels("mistral")

	require.NotEmpty(t, models)
	assert.Len(t, models, 2)

	modelIDs := make([]string, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
		assert.Equal(t, 32000, m.MaxTokens)
	}
	assert.Contains(t, modelIDs, "mistral-tiny")
	assert.Contains(t, modelIDs, "mistral-small")
}

func TestGetFallbackModels_Fireworks(t *testing.T) {
	models := GetFallbackModels("fireworks")

	require.NotEmpty(t, models)
	assert.Len(t, models, 1)
	assert.Equal(t, "llama-v2-7b-chat", models[0].ID)
}

func TestGetFallbackModels_Perplexity(t *testing.T) {
	models := GetFallbackModels("perplexity")

	require.NotEmpty(t, models)
	assert.Len(t, models, 1)
	assert.Equal(t, "pplx-70b-online", models[0].ID)
}

func TestGetFallbackModels_HuggingFace(t *testing.T) {
	models := GetFallbackModels("huggingface")

	require.NotEmpty(t, models)
	assert.Len(t, models, 1)
	assert.Equal(t, "llama-2-7b-chat", models[0].ID)
	assert.Equal(t, "Hugging Face", models[0].ProviderName)
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
		"fireworks", "perplexity", "huggingface",
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
