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
			{ID: "gemini-3.1-pro-preview", Name: "Gemini 3.1 Pro Preview", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 1048576},
			{ID: "gemini-3-pro-preview", Name: "Gemini 3 Pro Preview", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 1048576},
			{ID: "gemini-3-flash-preview", Name: "Gemini 3 Flash Preview", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 1048576},
			{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 1048576},
			{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 1048576},
			{ID: "gemini-2.5-flash-lite", Name: "Gemini 2.5 Flash Lite", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 1048576},
			{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", ProviderID: "gemini", ProviderName: "Google Gemini", MaxTokens: 1048576},
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
		"publicai": {
			{ID: "swiss-ai/apertus-8b-instruct", Name: "Apertus 8B Instruct", ProviderID: "publicai", ProviderName: "Public AI", MaxTokens: 65536},
		},
		"kimi": {
			{ID: "moonshot-v1-8k", Name: "Moonshot V1 8K", ProviderID: "kimi", ProviderName: "Kimi", MaxTokens: 8192},
			{ID: "moonshot-v1-32k", Name: "Moonshot V1 32K", ProviderID: "kimi", ProviderName: "Kimi", MaxTokens: 32768},
			{ID: "moonshot-v1-128k", Name: "Moonshot V1 128K", ProviderID: "kimi", ProviderName: "Kimi", MaxTokens: 131072},
		},
		"kimi-code": {
			{ID: "kimi-for-coding", Name: "Kimi For Coding", ProviderID: "kimi-code", ProviderName: "Kimi Code", MaxTokens: 262144},
		},
		"sambanova": {
			{ID: "Meta-Llama-3.1-8B-Instruct", Name: "Llama 3.1 8B Instruct", ProviderID: "sambanova", ProviderName: "SambaNova", MaxTokens: 131072},
			{ID: "Meta-Llama-3.1-70B-Instruct", Name: "Llama 3.1 70B Instruct", ProviderID: "sambanova", ProviderName: "SambaNova", MaxTokens: 131072},
			{ID: "Meta-Llama-3.2-1B-Instruct", Name: "Llama 3.2 1B Instruct", ProviderID: "sambanova", ProviderName: "SambaNova", MaxTokens: 131072},
			{ID: "Meta-Llama-3.2-3B-Instruct", Name: "Llama 3.2 3B Instruct", ProviderID: "sambanova", ProviderName: "SambaNova", MaxTokens: 131072},
		},
		"upstage": {
			{ID: "solar-1-mini-chat", Name: "Solar 1 Mini Chat", ProviderID: "upstage", ProviderName: "Upstage", MaxTokens: 32768},
			{ID: "solar-1-mini-chat-ja", Name: "Solar 1 Mini Chat Japanese", ProviderID: "upstage", ProviderName: "Upstage", MaxTokens: 32768},
			{ID: "solar-pro", Name: "Solar Pro", ProviderID: "upstage", ProviderName: "Upstage", MaxTokens: 32768},
		},
		"sarvam": {
			{ID: "sarvam-m", Name: "Sarvam M", ProviderID: "sarvam", ProviderName: "Sarvam AI", MaxTokens: 8192},
			{ID: "sarvam-2b", Name: "Sarvam 2B", ProviderID: "sarvam", ProviderName: "Sarvam AI", MaxTokens: 4096},
			{ID: "sarvam-7b", Name: "Sarvam 7B", ProviderID: "sarvam", ProviderName: "Sarvam AI", MaxTokens: 8192},
		},
		"zhipu": {
			{ID: "glm-4-flash", Name: "GLM 4 Flash", ProviderID: "zhipu", ProviderName: "Zhipu AI", MaxTokens: 128000},
			{ID: "glm-4", Name: "GLM 4", ProviderID: "zhipu", ProviderName: "Zhipu AI", MaxTokens: 128000},
			{ID: "glm-4-plus", Name: "GLM 4 Plus", ProviderID: "zhipu", ProviderName: "Zhipu AI", MaxTokens: 128000},
			{ID: "glm-4-air", Name: "GLM 4 Air", ProviderID: "zhipu", ProviderName: "Zhipu AI", MaxTokens: 128000},
			{ID: "glm-4-airx", Name: "GLM 4 AirX", ProviderID: "zhipu", ProviderName: "Zhipu AI", MaxTokens: 8192},
			{ID: "glm-4-long", Name: "GLM 4 Long", ProviderID: "zhipu", ProviderName: "Zhipu AI", MaxTokens: 1048576},
		},
		"hyperbolic": {
			{ID: "meta-llama/Meta-Llama-3-8B-Instruct", Name: "Llama 3 8B Instruct", ProviderID: "hyperbolic", ProviderName: "Hyperbolic", MaxTokens: 8192},
			{ID: "meta-llama/Meta-Llama-3-70B-Instruct", Name: "Llama 3 70B Instruct", ProviderID: "hyperbolic", ProviderName: "Hyperbolic", MaxTokens: 8192},
			{ID: "mistralai/Mistral-7B-Instruct-v0.2", Name: "Mistral 7B Instruct", ProviderID: "hyperbolic", ProviderName: "Hyperbolic", MaxTokens: 32768},
		},
		"siliconflow": {
			{ID: "Qwen/Qwen2.5-7B-Instruct", Name: "Qwen 2.5 7B Instruct", ProviderID: "siliconflow", ProviderName: "SiliconFlow", MaxTokens: 32768},
			{ID: "Qwen/Qwen2.5-72B-Instruct", Name: "Qwen 2.5 72B Instruct", ProviderID: "siliconflow", ProviderName: "SiliconFlow", MaxTokens: 32768},
			{ID: "deepseek-ai/DeepSeek-V2.5", Name: "DeepSeek V2.5", ProviderID: "siliconflow", ProviderName: "SiliconFlow", MaxTokens: 64000},
		},
		"novita": {
			{ID: "meta-llama/llama-3-8b-instruct", Name: "Llama 3 8B Instruct", ProviderID: "novita", ProviderName: "Novita", MaxTokens: 8192},
			{ID: "meta-llama/llama-3-70b-instruct", Name: "Llama 3 70B Instruct", ProviderID: "novita", ProviderName: "Novita", MaxTokens: 8192},
			{ID: "mistralai/mistral-7b-instruct", Name: "Mistral 7B Instruct", ProviderID: "novita", ProviderName: "Novita", MaxTokens: 32768},
		},
		"kilo": {
			{ID: "kilocode-1.5", Name: "KiloCode 1.5", ProviderID: "kilo", ProviderName: "Kilo", MaxTokens: 32768},
			{ID: "kilocode-1", Name: "KiloCode 1", ProviderID: "kilo", ProviderName: "Kilo", MaxTokens: 32768},
		},
		"modal": {
			{ID: "llama-3.1-8b", Name: "Llama 3.1 8B", ProviderID: "modal", ProviderName: "Modal", MaxTokens: 131072},
			{ID: "llama-3.1-70b", Name: "Llama 3.1 70B", ProviderID: "modal", ProviderName: "Modal", MaxTokens: 131072},
			{ID: "mistral-7b", Name: "Mistral 7B", ProviderID: "modal", ProviderName: "Modal", MaxTokens: 32768},
		},
		"nia": {
			{ID: "nia-1.5", Name: "Nia 1.5", ProviderID: "nia", ProviderName: "Nia", MaxTokens: 32768},
			{ID: "nia-1", Name: "Nia 1", ProviderID: "nia", ProviderName: "Nia", MaxTokens: 32768},
		},
		"nlpcloud": {
			{ID: "finetuned-llama-3-70b", Name: "Finetuned Llama 3 70B", ProviderID: "nlpcloud", ProviderName: "NLP Cloud", MaxTokens: 8192},
			{ID: "llama-3-70b", Name: "Llama 3 70B", ProviderID: "nlpcloud", ProviderName: "NLP Cloud", MaxTokens: 8192},
			{ID: "mixtral-8x7b", Name: "Mixtral 8x7B", ProviderID: "nlpcloud", ProviderName: "NLP Cloud", MaxTokens: 32768},
			{ID: "openchat-3-5", Name: "OpenChat 3.5", ProviderID: "nlpcloud", ProviderName: "NLP Cloud", MaxTokens: 8192},
		},
		"vulavula": {
			{ID: "vulavula-1.5", Name: "Vulavula 1.5", ProviderID: "vulavula", ProviderName: "Vulavula", MaxTokens: 32768},
			{ID: "vulavula-1", Name: "Vulavula 1", ProviderID: "vulavula", ProviderName: "Vulavula", MaxTokens: 32768},
		},
		"cloudflare": {
			{ID: "@cf/meta/llama-3.1-8b-instruct", Name: "Llama 3.1 8B Instruct", ProviderID: "cloudflare", ProviderName: "Cloudflare", MaxTokens: 8192},
			{ID: "@cf/meta/llama-3.1-70b-instruct", Name: "Llama 3.1 70B Instruct", ProviderID: "cloudflare", ProviderName: "Cloudflare", MaxTokens: 8192},
			{ID: "@cf/mistral/mistral-7b-instruct", Name: "Mistral 7B Instruct", ProviderID: "cloudflare", ProviderName: "Cloudflare", MaxTokens: 8192},
			{ID: "@cf/qwen/qwen1.5-14b-chat-awq", Name: "Qwen 1.5 14B Chat", ProviderID: "cloudflare", ProviderName: "Cloudflare", MaxTokens: 8192},
		},
		"qwen": {
			{ID: "qwen-turbo", Name: "Qwen Turbo", ProviderID: "qwen", ProviderName: "Qwen", MaxTokens: 131072},
			{ID: "qwen-plus", Name: "Qwen Plus", ProviderID: "qwen", ProviderName: "Qwen", MaxTokens: 131072},
			{ID: "qwen-max", Name: "Qwen Max", ProviderID: "qwen", ProviderName: "Qwen", MaxTokens: 32768},
			{ID: "qwen2.5-72b-instruct", Name: "Qwen 2.5 72B Instruct", ProviderID: "qwen", ProviderName: "Qwen", MaxTokens: 131072},
			{ID: "qwen2.5-32b-instruct", Name: "Qwen 2.5 32B Instruct", ProviderID: "qwen", ProviderName: "Qwen", MaxTokens: 131072},
			{ID: "qwen2.5-coder-32b-instruct", Name: "Qwen 2.5 Coder 32B", ProviderID: "qwen", ProviderName: "Qwen", MaxTokens: 131072},
		},
		"qwen-code": {
			{ID: "coder-model", Name: "Coder Model (Qwen 3.5 Plus)", ProviderID: "qwen-code", ProviderName: "Qwen Code", MaxTokens: 1048576},
			{ID: "vision-model", Name: "Vision Model (Qwen VL)", ProviderID: "qwen-code", ProviderName: "Qwen Code", MaxTokens: 262144},
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
