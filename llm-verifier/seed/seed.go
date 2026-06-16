// Package seed populates the verifier model database from the verifier's own
// configuration (config.yaml `llms:` list).
//
// It is fully project-agnostic and decoupled (per the Submodule-As-Equal-Codebase
// + Decoupling mandate): it reads ONLY the generic config.Config structure and
// writes via the public database CRUD API. No consuming-project name, path, host,
// model name, or provider name is hardcoded — every value comes from the supplied
// configuration.
//
// The seed is idempotent: a provider/model already present in the database is
// reused, never duplicated. This makes it safe to run at every server boot and
// also as an explicit `seed` subcommand.
package seed

import (
	"fmt"

	"digital.vasic.llmsverifier/config"
	"digital.vasic.llmsverifier/database"
)

// Result reports what a seed pass did. Counts are derived from real DB
// operations so a caller can emit honest, captured evidence (no bluff).
type Result struct {
	ProvidersCreated int      // providers newly inserted this pass
	ProvidersReused  int      // providers already present (idempotent skip)
	ModelsCreated    int      // models newly inserted this pass
	ModelsReused     int      // models already present (idempotent skip)
	Skipped          []string // config entries skipped, with reason
}

// dbWriter is the minimal database surface seed needs. *database.Database
// satisfies it. Declaring it as an interface keeps seed unit-testable and
// decoupled from the concrete connection.
type dbWriter interface {
	GetProviderByName(name string) (*database.Provider, error)
	CreateProvider(provider *database.Provider) error
	ListModels(filters map[string]any) ([]*database.Model, error)
	CreateModel(model *database.Model) error
}

// FromConfig idempotently seeds the database from cfg.LLMs.
//
// For each LLM config entry: a provider is keyed by its Name (created if absent,
// reused if present), and a model is keyed by (provider_id, model_id) where
// model_id is the entry's Model field (falling back to the entry Name when Model
// is empty). An entry whose Name is empty is skipped with a recorded reason
// rather than producing a malformed row.
func FromConfig(cfg *config.Config, db dbWriter) (*Result, error) {
	if cfg == nil {
		return nil, fmt.Errorf("seed: nil config")
	}
	if db == nil {
		return nil, fmt.Errorf("seed: nil database")
	}

	res := &Result{}

	for i := range cfg.LLMs {
		llm := cfg.LLMs[i]

		if llm.Name == "" {
			res.Skipped = append(res.Skipped, fmt.Sprintf("entry[%d]: empty name", i))
			continue
		}

		providerID, created, err := upsertProvider(db, llm)
		if err != nil {
			return res, fmt.Errorf("seed: provider %q: %w", llm.Name, err)
		}
		if created {
			res.ProvidersCreated++
		} else {
			res.ProvidersReused++
		}

		modelCreated, err := upsertModel(db, providerID, llm)
		if err != nil {
			return res, fmt.Errorf("seed: model for provider %q: %w", llm.Name, err)
		}
		if modelCreated {
			res.ModelsCreated++
		} else {
			res.ModelsReused++
		}
	}

	return res, nil
}

// upsertProvider returns the provider id for the given config entry, creating
// the provider if it does not already exist. Returns (id, created, error).
func upsertProvider(db dbWriter, llm config.LLMConfig) (int64, bool, error) {
	existing, err := db.GetProviderByName(llm.Name)
	if err == nil && existing != nil {
		return existing.ID, false, nil
	}
	// GetProviderByName returns an error for "not found"; treat that as absent
	// and proceed to create. Any other error path still results in a create
	// attempt whose own error (e.g. UNIQUE violation on a race) surfaces below.

	provider := &database.Provider{
		Name:     llm.Name,
		Endpoint: llm.Endpoint,
		IsActive: true,
	}
	if err := db.CreateProvider(provider); err != nil {
		// A concurrent creator may have won the race; re-resolve by name.
		if again, gerr := db.GetProviderByName(llm.Name); gerr == nil && again != nil {
			return again.ID, false, nil
		}
		return 0, false, err
	}
	return provider.ID, true, nil
}

// modelIdentifier derives the stable model_id for a config entry: the explicit
// Model field, or the entry Name when Model is empty.
func modelIdentifier(llm config.LLMConfig) string {
	if llm.Model != "" {
		return llm.Model
	}
	return llm.Name
}

// upsertModel creates the model for the entry under providerID if a model with
// the same (provider_id, model_id) does not already exist. Returns (created, error).
func upsertModel(db dbWriter, providerID int64, llm config.LLMConfig) (bool, error) {
	modelID := modelIdentifier(llm)

	existing, err := db.ListModels(map[string]any{
		"provider_id": providerID,
		"model_id":    modelID,
	})
	if err != nil {
		return false, fmt.Errorf("list existing models: %w", err)
	}
	if len(existing) > 0 {
		return false, nil
	}

	model := &database.Model{
		ProviderID:         providerID,
		ModelID:            modelID,
		Name:               llm.Name,
		VerificationStatus: "pending",
	}
	if err := db.CreateModel(model); err != nil {
		return false, err
	}
	return true, nil
}
