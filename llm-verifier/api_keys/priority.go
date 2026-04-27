package api_keys

import (
	"sort"
	"strings"
)

type ProviderPriority struct {
	ProviderID    string
	APIKeyName    string
	IsFaulty      bool
	IsUnsupported bool
}

func SortProvidersByPriority(providers []ProviderPriority) []ProviderPriority {
	sort.Slice(providers, func(i, j int) bool {
		if providers[i].IsFaulty != providers[j].IsFaulty {
			return !providers[i].IsFaulty
		}
		return providers[i].ProviderID < providers[j].ProviderID
	})
	return providers
}

func GetProviderAPIKeyName(providerID string) string {
	providerToKey := map[string]string{
		"openai":      "OPENAI_API_KEY",
		"anthropic":   "ANTHROPIC_API_KEY",
		"huggingface": "HUGGINGFACE_API_KEY",
		"groq":        "GROQ_API_KEY",
		"gemini":      "GEMINI_API_KEY",
		"deepseek":    "DEEPSEEK_API_KEY",
		"nvidia":      "NVIDIA_API_KEY",
		"openrouter":  "OPENROUTER_API_KEY",
		"replicate":   "REPLICATE_API_KEY",
		"fireworks":   "FIREWORKS_API_KEY",
		"together":    "TOGETHER_API_KEY",
		"perplexity":  "PERPLEXITY_API_KEY",
		"mistral":     "MISTRAL_API_KEY",
		"codestral":   "CODESTRAL_API_KEY",
		"cloudflare":  "CLOUDFLARE_API_KEY",
		"sambanova":   "SAMBANOVA_API_KEY",
		"cerebras":    "CEREBRAS_API_KEY",
		"modal":       "MODAL_API_KEY",
		"inference":   "INFERENCE_API_KEY",
		"siliconflow": "SILICONFLOW_API_KEY",
		"novita":      "NOVITA_API_KEY",
		"upstage":     "UPSTAGE_API_KEY",
		"nlpcloud":    "NLP_API_KEY",
		"hyperbolic":  "HYPERBOLIC_API_KEY",
		"zai":         "ZAI_API_KEY",
		"baseten":     "BASETEN_API_KEY",
		"twelvelabs":  "TWELVELABS_API_KEY",
		"chutes":      "CHUTES_API_KEY",
		"kimi":        "KIMI_API_KEY",
		"sarvam":      "SARVAM_API_KEY",
		"vulavula":    "VULAVULA_API_KEY",
		"vercel":      "VERCEL_API_KEY",
		"cohere":      "COHERE_API_KEY",
		"ai21":        "AI21_API_KEY",
		"aleph-alpha": "ALEPH_ALPHA_API_KEY",
		"writer":      "WRITER_API_KEY",
		"gooseai":     "GOOSECAI_API_KEY",
		"stability":   "STABILITY_API_KEY",
		"midjourney":  "MIDJOURNEY_API_KEY",
		"runway":      "RUNWAY_API_KEY",
		"elevenlabs":  "ELEVENLABS_API_KEY",
		"assemblyai":  "ASSEMBLYAI_API_KEY",
		"gladia":      "GLADIA_API_KEY",
		"fal":         "FAL_API_KEY",
		"relevance":   "RELEVANCE_API_KEY",
		"xai":         "XAI_API_KEY",
		"togetherai":  "TOGETHERAI_API_KEY",
		"opencode":    "OPENCODE_API_KEY",
		"claude-code": "CLAUDE_CODE_API_KEY",
		"qwen":        "QWEN_API_KEY",
		"mcp":         "MCP_API_KEY",
		"lsp":         "LSP_API_KEY",
	}

	if key, ok := providerToKey[providerID]; ok {
		return key
	}

	upperID := strings.ToUpper(providerID)
	return upperID + "_API_KEY"
}

func NewProviderPriority(providerID string) ProviderPriority {
	return ProviderPriority{
		ProviderID:    providerID,
		APIKeyName:    GetProviderAPIKeyName(providerID),
		IsFaulty:      false,
		IsUnsupported: false,
	}
}
