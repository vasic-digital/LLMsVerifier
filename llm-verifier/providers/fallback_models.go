package providers

import "strings"

// GetFallbackModels returns common models for providers that don't have API access
func GetFallbackModels(providerID string) []Model {
	commonModels := map[string][]Model{
		"openai": {
			{ID: "gpt-4", Name: "GPT-4", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 8192},
			{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 128000},
			{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 4096},
		},
		"anthropic": {
			{ID: "claude-3-5-sonnet", Name: "Claude 3.5 Sonnet", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-3-haiku", Name: "Claude 3 Haiku", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
		},
		"groq": {
			{ID: "llama2-70b", Name: "Llama 2 70B", ProviderID: "groq", ProviderName: "Groq", MaxTokens: 4096},
			{ID: "mixtral-8x7b", Name: "Mixtral 8x7B", ProviderID: "groq", ProviderName: "Groq", MaxTokens: 32768},
		},
		"gemini": {
			{ID: "gemini-pro", Name: "Gemini Pro", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 30720},
		},
		"deepseek": {
			{ID: "deepseek-chat", Name: "DeepSeek Chat", ProviderID: "deepseek", ProviderName: "DeepSeek", MaxTokens: 4096},
		},
		"nvidia": {
			{ID: "llama2-70b", Name: "Llama 2 70B", ProviderID: "nvidia", ProviderName: "NVIDIA", MaxTokens: 4096},
		},
		"openrouter": {
			{ID: "gpt-4", Name: "GPT-4", ProviderID: "openrouter", ProviderName: "OpenRouter", MaxTokens: 8192},
			{ID: "claude-3-sonnet", Name: "Claude 3 Sonnet", ProviderID: "openrouter", ProviderName: "OpenRouter", MaxTokens: 200000},
		},
		"together": {
			{ID: "llama-2-70b-chat", Name: "Llama 2 70B Chat", ProviderID: "together", ProviderName: "Together AI", MaxTokens: 4096},
		},
		"mistral": {
			{ID: "mistral-tiny", Name: "Mistral Tiny", ProviderID: "mistral", ProviderName: "Mistral AI", MaxTokens: 32000},
			{ID: "mistral-small", Name: "Mistral Small", ProviderID: "mistral", ProviderName: "Mistral AI", MaxTokens: 32000},
		},
		"fireworks": {
			{ID: "llama-v2-7b-chat", Name: "Llama 2 7B Chat", ProviderID: "fireworks", ProviderName: "Fireworks", MaxTokens: 4096},
		},
		"perplexity": {
			{ID: "pplx-70b-online", Name: "Perplexity 70B Online", ProviderID: "perplexity", ProviderName: "Perplexity", MaxTokens: 4096},
		},
		"huggingface": {
			{ID: "llama-2-7b-chat", Name: "Llama 2 7B Chat", ProviderID: "huggingface", ProviderName: "Hugging Face", MaxTokens: 4096},
		},
	}
	
	if models, exists := commonModels[providerID]; exists {
		return models
	}
	
	// Generic fallback
	return []Model{
		{ID: providerID + "-model", Name: strings.Title(providerID) + " Model", ProviderID: providerID, ProviderName: strings.Title(providerID), MaxTokens: 4096},
	}
}
