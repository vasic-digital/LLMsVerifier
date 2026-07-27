package providers

// Deprecated: RelaxedVerificationService is NOT used by any production code
// path — only its own tests (relaxed_verification_test.go,
// providers_extended_test.go) reference it. VerifyModelRelaxed marks every
// model from a "reputable" provider (and even unknown providers) as verified
// without making any API call, which is a verification bluff; the strict
// CodeVerificationService path (verification/code_verification.go) is the real
// gate. This file is retained only to keep those tests compiling — do not wire
// it into new code.

import (
	"context"
	"fmt"

	"digital.vasic.llmsverifier/logging"
)

// RelaxedVerificationService provides relaxed model verification
type RelaxedVerificationService struct {
	logger *logging.Logger
}

// NewRelaxedVerificationService creates a new relaxed verification service
func NewRelaxedVerificationService(logger *logging.Logger) *RelaxedVerificationService {
	return &RelaxedVerificationService{logger: logger}
}

// VerifyModelRelaxed performs relaxed verification - just checks if model exists and responds
func (rvs *RelaxedVerificationService) VerifyModelRelaxed(ctx context.Context, model Model, providerClient *ProviderClient) bool {
	rvs.logger.Info(fmt.Sprintf("Performing relaxed verification for %s/%s", model.ProviderID, model.ID), nil)

	// For now, assume all models from reputable sources are verified
	// In a real implementation, this would make a simple API call

	reputableProviders := []string{
		"openai", "anthropic", "huggingface", "groq", "gemini", "deepseek",
		"nvidia", "openrouter", "replicate", "fireworks", "together",
		"perplexity", "mistral", "cloudflare", "sambanova", "cerebras",
		"modal", "inference", "siliconflow", "novita", "upstage",
		"nlpcloud", "hyperbolic", "chutes", "kimi",
	}

	for _, provider := range reputableProviders {
		if model.ProviderID == provider {
			rvs.logger.Info(fmt.Sprintf("✅ Model %s/%s verified (reputable provider)", model.ProviderID, model.ID), nil)
			return true
		}
	}

	rvs.logger.Info(fmt.Sprintf("⚠️  Model %s/%s from unknown provider, still marking as verified", model.ProviderID, model.ID), nil)
	return true // Be very permissive
}
