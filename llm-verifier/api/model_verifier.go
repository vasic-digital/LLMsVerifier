package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"digital.vasic.llmsverifier/client"
	"digital.vasic.llmsverifier/config"
	"digital.vasic.llmsverifier/database"
	"digital.vasic.llmsverifier/logging"
	"digital.vasic.llmsverifier/providers"
)

// ModelVerifier runs real model verification for the REST verify endpoint and
// returns the verification status + score so the handler can persist them onto
// the model. It is a seam: the production implementation (liveModelVerifier)
// calls the real providers.ModelVerificationService against the provider's live
// API; unit tests substitute a stub so the handler can be exercised with a
// known outcome and no network.
type ModelVerifier interface {
	// Verify runs verification for dbModel served by provider. It returns the
	// resolved verification status ("verified" / "failed" / "error" / "skipped"),
	// the verification score, and an error only for an internal failure that
	// prevented verification from running at all. A model that runs but does not
	// pass returns a non-"verified" status with a nil error — the caller persists
	// the honest outcome either way.
	Verify(ctx context.Context, dbModel *database.Model, provider *database.Provider) (status string, score float64, err error)
}

// liveModelVerifier is the production ModelVerifier. It builds a
// providers.ProviderClient for the model's provider (BaseURL from the provider's
// stored endpoint, API key resolved from config/environment) and delegates to
// providers.ModelVerificationService.VerifyModel — the real verification that
// makes live API calls and sets the verification status + score.
type liveModelVerifier struct {
	cfg     *config.Config
	svc     *providers.ModelVerificationService
	timeout time.Duration
}

// newLiveModelVerifier constructs the production verifier. Verification is
// enabled and non-strict so a genuinely-passing model is marked "verified".
// logger may be nil; a nil logger results in a real but DB-less logger so the
// verification service never dereferences a nil receiver.
func newLiveModelVerifier(cfg *config.Config, logger *logging.Logger) *liveModelVerifier {
	timeout := 60 * time.Second
	if cfg != nil && cfg.Global.Timeout > 0 {
		timeout = cfg.Global.Timeout
	}

	if logger == nil {
		// NewLogger never fails for an in-memory logger; ignore the error and
		// proceed — verification logging is best-effort, not load-bearing.
		logger, _ = logging.NewLogger(nil, nil)
	}

	httpClient := client.NewHTTPClient(timeout)
	svc := providers.NewModelVerificationService(httpClient, logger, providers.VerificationConfig{
		Enabled:        true,
		StrictMode:     false,
		MaxRetries:     1,
		TimeoutSeconds: int(timeout / time.Second),
	})

	return &liveModelVerifier{cfg: cfg, svc: svc, timeout: timeout}
}

// Verify implements ModelVerifier against the real verification service.
func (l *liveModelVerifier) Verify(
	ctx context.Context,
	dbModel *database.Model,
	provider *database.Provider,
) (string, float64, error) {
	pClient := &providers.ProviderClient{
		ProviderID: provider.Name,
		BaseURL:    strings.TrimSuffix(provider.Endpoint, "/"),
		APIKey:     l.resolveAPIKey(provider.Name),
		HTTPClient: &http.Client{Timeout: l.timeout},
	}

	pModel := providers.Model{
		ID:         dbModel.ModelID,
		Name:       dbModel.Name,
		ProviderID: provider.Name,
		Provider:   provider.Name,
	}

	result, err := l.svc.VerifyModel(ctx, pModel, pClient)
	if err != nil {
		return "error", 0, err
	}
	return result.VerificationStatus, result.VerificationScore, nil
}

// resolveAPIKey resolves the API key for a provider name without ever logging
// it (§11.4.10). Resolution order, decoupled from any consumer (§11.4.28):
// (1) a matching config.LLMs entry by name (the same source the seed used),
// (2) the conventional <UPPER_SNAKE_NAME>_API_KEY environment variable.
func (l *liveModelVerifier) resolveAPIKey(providerName string) string {
	if l.cfg != nil {
		for _, llm := range l.cfg.LLMs {
			if strings.EqualFold(llm.Name, providerName) && llm.APIKey != "" {
				return llm.APIKey
			}
		}
	}

	envName := strings.ToUpper(providerName)
	envName = strings.NewReplacer(" ", "_", "-", "_", ".", "_").Replace(envName)
	if v := os.Getenv(envName + "_API_KEY"); v != "" {
		return v
	}
	return ""
}
