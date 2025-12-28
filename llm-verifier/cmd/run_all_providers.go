package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"llm-verifier/cmd/model-verification"
	"llm-verifier/config"
)

func runAllProvidersVerification() error {
	fmt.Println("=== LLM Verifier: Dynamic Model Discovery ===")
	fmt.Printf("Starting verification of all providers from .env at %s\n\n", time.Now().Format(time.RFC3339))

	// Load configuration from .env
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize verification runner
	runner := model_verification.NewVerificationRunner(cfg)

	// Get all provider API keys from environment
	providers := []struct {
		name   string
		envKey string
	}{
		{"openai", "OPENAI_API_KEY"},
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"google", "GOOGLE_API_KEY"},
		{"deepseek", "DEEPSEEK_API_KEY"},
		{"mistral", "MISTRAL_API_KEY"},
		{"cohere", "COHERE_API_KEY"},
		{"huggingface", "HUGGINGFACE_API_KEY"},
		{"together", "TOGETHER_API_KEY"},
		{"fireworks", "FIREWORKS_API_KEY"},
		{"replicate", "REPLICATE_API_KEY"},
		{"groq", "GROQ_API_KEY"},
		{"perplexity", "PERPLEXITY_API_KEY"},
		{"nvidia", "NVIDIA_API_KEY"},
		{"chutes", "CHUTES_API_KEY"},
		{"siliconflow", "SILICONFLOW_API_KEY"},
		{"kimi", "KIMI_API_KEY"},
		{"openrouter", "OPENROUTER_API_KEY"},
		{"zai", "ZAI_API_KEY"},
		{"cerebras", "CEREBRAS_API_KEY"},
		{"cloudflare", "CLOUDFLARE_API_KEY"},
		{"vercel", "VERCEL_API_KEY"},
		{"baseten", "BASETEN_API_KEY"},
		{"novita", "NOVITA_API_KEY"},
		{"upstage", "UPSTAGE_API_KEY"},
		{"nlpcloud", "NLPCLOUD_API_KEY"},
		{"modal", "MODAL_API_KEY"},
		{"inference", "INFERENCE_API_KEY"},
		{"hyperbolic", "HYPERBOLIC_API_KEY"},
		{"sambanova", "SAMBANOVA_API_KEY"},
		{"vertex", "VERTEX_API_KEY"},
	}

	// Check which providers have API keys
	hasKeys := 0
	for _, p := range providers {
		apiKey := os.Getenv(p.envKey)
		if apiKey != "" {
			hasKeys++
			log.Printf("Found API key for %s (env: %s)", p.name, p.envKey)
		}
	}

	fmt.Printf("\nFound %d providers with API keys\n\n", hasKeys)

	// Build provider data with API keys
	providerData := make(map[string]model_verification.ProviderInfo)
	for _, p := range providers {
		apiKey := os.Getenv(p.envKey)
		if apiKey == "" {
			continue // Skip providers without API keys
		}

		endpoint := getProviderEndpoint(p.name)
		providerData[p.name] = model_verification.ProviderInfo{
			endpoint: endpoint,
			apiKey:   apiKey,
			models:   []string{}, // Empty means fetch dynamically
		}
	}

	// Initialize runner with provider data
	if err := runner.Initialize(providerData); err != nil {
		return fmt.Errorf("failed to initialize verification runner: %w", err)
	}

	// Run verification
	if err := runner.VerifyAllProviders(); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Print summary
	results := runner.GetResults()
	fmt.Printf("\n=== Verification Complete ===\n")
	fmt.Printf("Total Providers: %d\n", results.Summary.TotalProviders)
	fmt.Printf("Providers with Keys: %d\n", results.Summary.ProvidersWithKeys)
	fmt.Printf("Total Models: %d\n", results.Summary.TotalModels)
	fmt.Printf("Verified Models: %d\n", results.Summary.VerifiedModels)
	fmt.Printf("Failed Models: %d\n", results.Summary.FailedModels)

	return nil
}

func getProviderEndpoint(provider string) string {
	// Map of known provider endpoints
	endpoints := map[string]string{
		"openai":      "https://api.openai.com/v1",
		"anthropic":   "https://api.anthropic.com/v1",
		"google":      "https://generativelanguage.googleapis.com/v1",
		"deepseek":    "https://api.deepseek.com/v1",
		"mistal":      "https://api.mistral.ai/v1",
		"cohere":      "https://api.cohere.ai/v1",
		"huggingface": "https://api-inference.huggingface.co",
		"together":    "https://api.together.xyz/v1",
		"fireworks":   "https://api.fireworks.ai/v1",
		"replicate":   "https://api.replicate.com/v1",
		"groq":        "https://api.groq.com/openai/v1",
		"perplexity":  "https://api.perplexity.ai/v1",
		"nvidia":      "https://integrate.api.nvidia.com/v1",
		"chutes":      "https://api.chutes.ai/v1",
		"siliconflow": "https://api.siliconflow.cn/v1",
		"kimi":        "https://api.moonshot.cn/v1",
		"openrouter":  "https://openrouter.ai/api/v1",
		"zai":         "https://api.zai.dev/v1",
		"cerebras":    "https://api.cerebras.ai/v1",
		"cloudflare":  "https://api.cloudflare.com/client/v4/accounts",
		"vercel":      "https://api.vercel.com/v1",
		"baseten":     "https://inference.baseten.co/v1",
		"novita":      "https://api.novita.ai/v3/openai",
		"upstage":     "https://api.upstage.ai/v1",
		"nlpcloud":    "https://api.nlpcloud.com/v1",
		"modal":       "https://api.modal.com/v1",
		"inference":   "https://api.inference.net/v1",
		"hyperbolic":  "https://api.hyperbolic.xyz/v1",
		"sambanova":   "https://api.sambanova.ai/v1",
		"vertex":      "https://us-central1-aiplatform.googleapis.com/v1",
	}

	if endpoint, ok := endpoints[provider]; ok {
		return endpoint
	}

	// Default OpenAI-compatible endpoint
	return fmt.Sprintf("https://api.%s.com/v1", provider)
}

func init() {
	// Register this command with the root command in main.go
}
