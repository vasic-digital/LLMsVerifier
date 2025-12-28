#!/usr/bin/env python3
"""
Update http_client.go with comprehensive endpoint mappings
"""

import re

# Read the original file
with open('/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/llm-verifier/client/http_client.go', 'r') as f:
    content = f.read()

# New endpoint mappings
new_provider_endpoint = """// getProviderEndpoint returns the models list endpoint for a provider
func getProviderEndpoint(provider string) string {
	providerEndpoints := map[string]string{
		// Core providers
		"openai":      "https://api.openai.com/v1/models",
		"anthropic":   "https://api.anthropic.com/v1/models",
		"google":      "https://generativelanguage.googleapis.com/v1/models",
		"gemini":      "https://generativelanguage.googleapis.com/v1beta/models",
		
		// OpenAI-compatible providers
		"openrouter":  "https://openrouter.ai/api/v1/models",
		"deepseek":    "https://api.deepseek.com/v1/models",
		"mistral":     "https://api.mistral.ai/v1/models",
		"mistralaistudio": "https://api.mistral.ai/v1/models",
		"groq":        "https://api.groq.com/openai/v1/models",
		"togetherai":  "https://api.together.xyz/v1/models",
		"fireworksai": "https://api.fireworks.ai/v1/models",
		"fireworks":   "https://api.fireworks.ai/v1/models",
		"chutes":      "https://api.chutes.ai/v1/models",
		"siliconflow": "https://api.siliconflow.cn/v1/models",
		"kimi":        "https://api.moonshot.cn/v1/models",
		"zai":         "https://api.studio.nebius.ai/v1/models",
		"nebius":      "https://api.studio.nebius.ai/v1/models",
		"hyperbolic":  "https://api.hyperbolic.xyz/v1/models",
		"baseten":     "https://inference.baseten.co/v1/models",
		"novita":      "https://api.novita.ai/v1/models",
		"upstage":     "https://api.upstage.ai/v1/models",
		"inference":   "https://api.inference.net/v1/models",
		"cerebras":    "https://api.cerebras.ai/v1/models",
		"modal":       "https://api.modal.com/v1/models",
		"sambanova":   "https://api.sambanova.ai/v1/models",
		
		// Special API providers
		"huggingface": "https://api-inference.huggingface.co/models",
		"cohere":      "https://api.cohere.ai/v1/models",
		"replicate":   "https://api.replicate.com/v1/models",
		"nlpcloud":    "https://api.nlpcloud.com/v1/models",
		"poe":         "https://api.poe.com/v1/models",
		"navigator":   "https://api.ai.it.ufl.edu/v1/models",
		"codestral":   "https://api.mistral.ai/v1/models",
		"nvidia":      "https://integrate.api.nvidia.com/v1/models",
		
		// Cloud providers
		"cloudflare":  "https://api.cloudflare.com/client/v4/accounts/{{account_id}}/ai/models",
		
		// Vercel AI Gateway
		"vercelai":    "https://api.vercel.com/v1/ai/models",
		"vercel":      "https://api.vercel.com/v1/ai/models",
		"vercelaigateway": "https://api.vercel.com/v1/ai/models",
	}

	if endpoint, ok := providerEndpoints[strings.ToLower(provider)]; ok {
		return endpoint
	}
	
	// Return empty string for unknown providers
	// This allows the caller to handle the error gracefully
	return ""
}"""

new_model_endpoint = """// getModelEndpoint returns the chat/completion endpoint for a provider
func getModelEndpoint(provider, modelID string) string {
	providerEndpoints := map[string]string{
		// Core providers
		"openai":      "https://api.openai.com/v1/chat/completions",
		"anthropic":   "https://api.anthropic.com/v1/messages",
		"google":      "https://generativelanguage.googleapis.com/v1beta/models/" + modelID + ":generateContent",
		"gemini":      "https://generativelanguage.googleapis.com/v1beta/models/" + modelID + ":generateContent",
		
		// OpenAI-compatible providers
		"openrouter":  "https://openrouter.ai/api/v1/chat/completions",
		"deepseek":    "https://api.deepseek.com/v1/chat/completions",
		"mistral":     "https://api.mistral.ai/v1/chat/completions",
		"mistralaistudio": "https://api.mistral.ai/v1/chat/completions",
		"groq":        "https://api.groq.com/openai/v1/chat/completions",
		"togetherai":  "https://api.together.xyz/v1/chat/completions",
		"fireworksai": "https://api.fireworks.ai/v1/chat/completions",
		"fireworks":   "https://api.fireworks.ai/v1/chat/completions",
		"chutes":      "https://api.chutes.ai/v1/chat/completions",
		"siliconflow": "https://api.siliconflow.cn/v1/chat/completions",
		"kimi":        "https://api.moonshot.cn/v1/chat/completions",
		"zai":         "https://api.studio.nebius.ai/v1/chat/completions",
		"nebius":      "https://api.studio.nebius.ai/v1/chat/completions",
		"hyperbolic":  "https://api.hyperbolic.xyz/v1/chat/completions",
		"baseten":     "https://inference.baseten.co/v1/chat/completions",
		"novita":      "https://api.novita.ai/v1/chat/completions",
		"upstage":     "https://api.upstage.ai/v1/chat/completions",
		"inference":   "https://api.inference.net/v1/chat/completions",
		"cerebras":    "https://api.cerebras.ai/v1/chat/completions",
		"modal":       "https://api.modal.com/v1/chat/completions",
		"sambanova":   "https://api.sambanova.ai/v1/chat/completions",
		
		// Special API providers
		"huggingface": "https://api-inference.huggingface.co/models/" + modelID,
		"cohere":      "https://api.cohere.ai/v1/generate",
		"replicate":   "https://api.replicate.com/v1/predictions",
		"nlpcloud":    "https://api.nlpcloud.com/v1/gpu",
		"poe":         "https://api.poe.com/v1/chat/completions",
		"navigator":   "https://api.ai.it.ufl.edu/v1/chat/completions",
		"codestral":   "https://api.mistral.ai/v1/fim/completions",
		"nvidia":      "https://integrate.api.nvidia.com/v1/chat/completions",
		
		// Cloud providers (special handling needed)
		"cloudflare":  "https://api.cloudflare.com/client/v4/accounts/{{account_id}}/ai/run/" + modelID,
		
		// Vercel AI Gateway
		"vercelai":    "https://api.vercel.com/v1/ai/chat/completions",
		"vercel":      "https://api.vercel.com/v1/ai/chat/completions",
		"vercelaigateway": "https://api.vercel.com/v1/ai/chat/completions",
	}

	if endpoint, ok := providerEndpoints[strings.ToLower(provider)]; ok {
		return endpoint
	}
	
	// Return empty string for unknown providers
	return ""
}"""

# Replace the functions using regex
pattern = r'// getProviderEndpoint returns the models list endpoint for a provider\s+func getProviderEndpoint[^}]+}\s+}'
replacement = new_provider_endpoint

content_new = re.sub(pattern, replacement, content, flags=re.DOTALL)

if content_new == content:
    print("❌ Failed to replace getProviderEndpoint")
    exit(1)

print("✅ Updated getProviderEndpoint")

# Now replace getModelEndpoint
pattern2 = r'// getModelEndpoint returns the chat/completion endpoint for a provider\s+func getModelEndpoint[^}]+}\s+}'
replacement2 = new_model_endpoint

content_final = re.sub(pattern2, replacement2, content_new, flags=re.DOTALL)

if content_final == content_new:
    print("❌ Failed to replace getModelEndpoint")
    exit(1)

print("✅ Updated getModelEndpoint")

# Write the updated file
with open('/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/llm-verifier/client/http_client.go', 'w') as f:
    f.write(content_final)

print("✅ File updated successfully")
print("✅ Added 22 new provider endpoint mappings")