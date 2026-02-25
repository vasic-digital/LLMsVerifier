package providers

import (
	"strings"
	"sync"
	"time"
)

// FallbackModelsVersion tracks when this list was last verified
// Last updated: 2026-02 based on provider documentation
const FallbackModelsVersion = "2026-02"

// ModelCache provides thread-safe caching for discovered models
type ModelCache struct {
	mu         sync.RWMutex
	models     map[string][]Model
	timestamps map[string]time.Time
	ttl        time.Duration
}

var modelCache = &ModelCache{
	models:     make(map[string][]Model),
	timestamps: make(map[string]time.Time),
	ttl:        24 * time.Hour, // Cache models for 24 hours
}

// GetCachedModels returns cached models for a provider if available and not expired
func GetCachedModels(providerID string) ([]Model, bool) {
	modelCache.mu.RLock()
	defer modelCache.mu.RUnlock()

	if models, exists := modelCache.models[providerID]; exists {
		if ts, ok := modelCache.timestamps[providerID]; ok {
			if time.Since(ts) < modelCache.ttl {
				return models, true
			}
		}
	}
	return nil, false
}

// SetCachedModels stores discovered models in the cache
func SetCachedModels(providerID string, models []Model) {
	modelCache.mu.Lock()
	defer modelCache.mu.Unlock()

	modelCache.models[providerID] = models
	modelCache.timestamps[providerID] = time.Now()
}

// GetFallbackModels returns common models for providers that don't have API access
// These are curated fallbacks verified against provider documentation as of 2025-01
func GetFallbackModels(providerID string) []Model {
	// First check if we have cached models from a previous successful API call
	if cached, ok := GetCachedModels(providerID); ok {
		return cached
	}

	// Updated fallback lists - verified against provider docs 2025-01
	commonModels := map[string][]Model{
		"openai": {
			{ID: "gpt-4o", Name: "GPT-4o", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 128000},
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 128000},
			{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 128000},
			{ID: "gpt-4", Name: "GPT-4", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 8192},
			{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 16385},
			{ID: "o1-preview", Name: "O1 Preview", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 128000},
			{ID: "o1-mini", Name: "O1 Mini", ProviderID: "openai", ProviderName: "OpenAI", MaxTokens: 128000},
		},
		"anthropic": {
			{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-opus-4-5-20251101", Name: "Claude Opus 4.5", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-sonnet-4-5-20250929", Name: "Claude Sonnet 4.5", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-opus-4-20250514", Name: "Claude Opus 4", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-3-sonnet-20240229", Name: "Claude 3 Sonnet", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
			{ID: "claude-3-haiku-20240307", Name: "Claude 3 Haiku", ProviderID: "anthropic", ProviderName: "Anthropic", MaxTokens: 200000},
		},
		"groq": {
			{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B Versatile", ProviderID: "groq", ProviderName: "Groq", MaxTokens: 128000},
			{ID: "llama-3.1-70b-versatile", Name: "Llama 3.1 70B Versatile", ProviderID: "groq", ProviderName: "Groq", MaxTokens: 131072},
			{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B Instant", ProviderID: "groq", ProviderName: "Groq", MaxTokens: 131072},
			{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", ProviderID: "groq", ProviderName: "Groq", MaxTokens: 32768},
			{ID: "gemma2-9b-it", Name: "Gemma 2 9B IT", ProviderID: "groq", ProviderName: "Groq", MaxTokens: 8192},
		},
		"gemini": {
			{ID: "gemini-2.0-flash-exp", Name: "Gemini 2.0 Flash (Exp)", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 1048576},
			{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 2097152},
			{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 1048576},
			{ID: "gemini-pro", Name: "Gemini Pro", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 30720},
		},
		"deepseek": {
			{ID: "deepseek-chat", Name: "DeepSeek Chat", ProviderID: "deepseek", ProviderName: "DeepSeek", MaxTokens: 64000},
			{ID: "deepseek-coder", Name: "DeepSeek Coder", ProviderID: "deepseek", ProviderName: "DeepSeek", MaxTokens: 64000},
			{ID: "deepseek-reasoner", Name: "DeepSeek Reasoner", ProviderID: "deepseek", ProviderName: "DeepSeek", MaxTokens: 64000},
		},
		"cohere": {
			{ID: "command-r-plus", Name: "Command R+", ProviderID: "cohere", ProviderName: "Cohere", MaxTokens: 128000},
			{ID: "command-r", Name: "Command R", ProviderID: "cohere", ProviderName: "Cohere", MaxTokens: 128000},
			{ID: "command", Name: "Command", ProviderID: "cohere", ProviderName: "Cohere", MaxTokens: 4096},
			{ID: "command-light", Name: "Command Light", ProviderID: "cohere", ProviderName: "Cohere", MaxTokens: 4096},
		},
		"nvidia": {
			{ID: "llama-3.1-nemotron-70b-instruct", Name: "Llama 3.1 Nemotron 70B", ProviderID: "nvidia", ProviderName: "NVIDIA", MaxTokens: 128000},
			{ID: "mixtral-8x22b-instruct-v0.1", Name: "Mixtral 8x22B Instruct", ProviderID: "nvidia", ProviderName: "NVIDIA", MaxTokens: 65536},
		},
		"openrouter": {
			{ID: "openai/gpt-4o", Name: "GPT-4o", ProviderID: "openrouter", ProviderName: "OpenRouter", MaxTokens: 128000},
			{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", ProviderID: "openrouter", ProviderName: "OpenRouter", MaxTokens: 200000},
			{ID: "google/gemini-pro-1.5", Name: "Gemini Pro 1.5", ProviderID: "openrouter", ProviderName: "OpenRouter", MaxTokens: 1000000},
			{ID: "meta-llama/llama-3.1-405b-instruct", Name: "Llama 3.1 405B", ProviderID: "openrouter", ProviderName: "OpenRouter", MaxTokens: 131072},
		},
		"together": {
			{ID: "meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo", Name: "Llama 3.1 405B Instruct", ProviderID: "together", ProviderName: "Together AI", MaxTokens: 4096},
			{ID: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo", Name: "Llama 3.1 70B Instruct", ProviderID: "together", ProviderName: "Together AI", MaxTokens: 131072},
			{ID: "mistralai/Mixtral-8x22B-Instruct-v0.1", Name: "Mixtral 8x22B Instruct", ProviderID: "together", ProviderName: "Together AI", MaxTokens: 65536},
		},
		"mistral": {
			{ID: "mistral-large-latest", Name: "Mistral Large", ProviderID: "mistral", ProviderName: "Mistral AI", MaxTokens: 128000},
			{ID: "mistral-medium-latest", Name: "Mistral Medium", ProviderID: "mistral", ProviderName: "Mistral AI", MaxTokens: 32000},
			{ID: "mistral-small-latest", Name: "Mistral Small", ProviderID: "mistral", ProviderName: "Mistral AI", MaxTokens: 32000},
			{ID: "codestral-latest", Name: "Codestral", ProviderID: "mistral", ProviderName: "Mistral AI", MaxTokens: 32000},
		},
		"fireworks": {
			{ID: "accounts/fireworks/models/llama-v3p1-405b-instruct", Name: "Llama 3.1 405B", ProviderID: "fireworks", ProviderName: "Fireworks", MaxTokens: 131072},
			{ID: "accounts/fireworks/models/llama-v3p1-70b-instruct", Name: "Llama 3.1 70B", ProviderID: "fireworks", ProviderName: "Fireworks", MaxTokens: 131072},
			{ID: "accounts/fireworks/models/mixtral-8x22b-instruct", Name: "Mixtral 8x22B", ProviderID: "fireworks", ProviderName: "Fireworks", MaxTokens: 65536},
		},
		"perplexity": {
			{ID: "llama-3.1-sonar-huge-128k-online", Name: "Sonar Huge 128K Online", ProviderID: "perplexity", ProviderName: "Perplexity", MaxTokens: 127072},
			{ID: "llama-3.1-sonar-large-128k-online", Name: "Sonar Large 128K Online", ProviderID: "perplexity", ProviderName: "Perplexity", MaxTokens: 127072},
			{ID: "llama-3.1-sonar-small-128k-online", Name: "Sonar Small 128K Online", ProviderID: "perplexity", ProviderName: "Perplexity", MaxTokens: 127072},
		},
		"huggingface": {
			{ID: "meta-llama/Llama-3.1-70B-Instruct", Name: "Llama 3.1 70B Instruct", ProviderID: "huggingface", ProviderName: "Hugging Face", MaxTokens: 8192},
			{ID: "mistralai/Mixtral-8x7B-Instruct-v0.1", Name: "Mixtral 8x7B Instruct", ProviderID: "huggingface", ProviderName: "Hugging Face", MaxTokens: 32768},
		},
	}

	if models, exists := commonModels[providerID]; exists {
		return models
	}

	// Generic fallback for unknown providers
	return []Model{
		{ID: providerID + "-model", Name: toTitle(providerID) + " Model", ProviderID: providerID, ProviderName: toTitle(providerID), MaxTokens: 4096},
	}
}

// toTitle is a simple replacement for strings.Title which is deprecated
func toTitle(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
