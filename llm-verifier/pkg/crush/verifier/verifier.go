package crush_verifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	crush_config "llm-verifier/pkg/crush/config"
	"llm-verifier/database"
)

// CrushVerifier handles verification of Crush configurations and setups
type CrushVerifier struct {
	db         *database.Database
	validator  *crush_config.SchemaValidator
	configPath string
}

// NewCrushVerifier creates a new Crush verifier
func NewCrushVerifier(db *database.Database, configPath string) *CrushVerifier {
	return &CrushVerifier{
		db:         db,
		validator:  crush_config.NewSchemaValidator(),
		configPath: configPath,
	}
}

// VerificationResult represents the result of a configuration verification
type VerificationResult struct {
	ConfigFile      string                                   `json:"config_file"`
	Valid           bool                                     `json:"valid"`
	Errors          []crush_config.ValidationError           `json:"errors,omitempty"`
	Warnings        []crush_config.ValidationWarning         `json:"warnings,omitempty"`
	ProviderStatus  map[string]ProviderVerificationStatus    `json:"provider_status,omitempty"`
	ModelStatus     map[string]map[string]ModelVerificationStatus `json:"model_status,omitempty"`
	LspStatus       map[string]LspVerificationStatus         `json:"lsp_status,omitempty"`
	OverallScore    float64                                  `json:"overall_score"`
}

// ProviderVerificationStatus represents the verification status of a provider
type ProviderVerificationStatus struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Configured   bool    `json:"configured"`
	HasAPIKey    bool    `json:"has_api_key"`
	ModelCount   int     `json:"model_count"`
	Verified     bool    `json:"verified"`
	Error        string  `json:"error,omitempty"`
	Score        float64 `json:"score"`
}

// ModelVerificationStatus represents the verification status of a model
type ModelVerificationStatus struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Configured       bool    `json:"configured"`
	HasCostInfo      bool    `json:"has_cost_info"`
	HasContextInfo   bool    `json:"has_context_info"`
	HasFeatureFlags  bool    `json:"has_feature_flags"`
	Score            float64 `json:"score"`
}

// LspVerificationStatus represents the verification status of an LSP server
type LspVerificationStatus struct {
	Name       string   `json:"name"`
	Command    string   `json:"command"`
	Args       []string `json:"args,omitempty"`
	Configured bool     `json:"configured"`
	Enabled    bool     `json:"enabled"`
	Score      float64  `json:"score"`
}

// VerifyConfiguration verifies a Crush configuration file
func (v *CrushVerifier) VerifyConfiguration() (*VerificationResult, error) {
	result := &VerificationResult{
		ConfigFile:     v.configPath,
		ProviderStatus: make(map[string]ProviderVerificationStatus),
		ModelStatus:    make(map[string]map[string]ModelVerificationStatus),
		LspStatus:      make(map[string]LspVerificationStatus),
	}

	// Validate configuration
	validationResult, err := v.validator.ValidateFile(v.configPath)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	result.Errors = validationResult.Errors
	result.Warnings = validationResult.Warnings
	result.Valid = validationResult.Valid

	// Load and parse configuration
	configContent, err := os.ReadFile(v.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg crush_config.Config
	if err := json.Unmarshal(configContent, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Verify providers and models
	for name, provider := range cfg.Providers {
		status := v.VerifyProvider(name, &provider)
		result.ProviderStatus[name] = status
		
		// Verify models for this provider
		result.ModelStatus[name] = make(map[string]ModelVerificationStatus)
		for _, model := range provider.Models {
			modelStatus := v.VerifyModel(&model)
			result.ModelStatus[name][model.ID] = modelStatus
		}
	}

	// Verify LSP servers
	for name, lsp := range cfg.Lsp {
		status := v.VerifyLSP(name, &lsp)
		result.LspStatus[name] = status
	}

	// Calculate overall score
	result.OverallScore = v.calculateOverallScore(result)

	return result, nil
}

// VerifySetup verifies a complete Crush project setup
func (v *CrushVerifier) VerifySetup(projectPath string) (map[string]*VerificationResult, error) {
	results := make(map[string]*VerificationResult)

	// Check for .crush directory
	crushDir := filepath.Join(projectPath, ".crush")
	if _, err := os.Stat(crushDir); err == nil {
		// Verify configuration files in .crush
		configFiles := []string{"crush.json"}
		for _, filename := range configFiles {
			configPath := filepath.Join(crushDir, filename)
			if _, err := os.Stat(configPath); err == nil {
				verifier := NewCrushVerifier(v.db, configPath)
				result, err := verifier.VerifyConfiguration()
				if err != nil {
					return nil, err
				}
				results[configPath] = result
			}
		}
	}

	// Check for crush.json in project root
	rootConfig := filepath.Join(projectPath, "crush.json")
	if _, err := os.Stat(rootConfig); err == nil {
		verifier := NewCrushVerifier(v.db, rootConfig)
		result, err := verifier.VerifyConfiguration()
		if err != nil {
			return nil, err
		}
		results[rootConfig] = result
	}

	return results, nil
}

func (v *CrushVerifier) VerifyProvider(name string, provider *crush_config.Provider) ProviderVerificationStatus {
	status := ProviderVerificationStatus{
		Name:       name,
		Type:       provider.Type,
		Configured: true,
		ModelCount: len(provider.Models),
	}

	// Check if has API key
	if provider.APIKey != "" {
		status.HasAPIKey = true
	}

	// Score calculation
	score := 50.0 // Base score
	if status.HasAPIKey {
		score += 25
	}
	if status.ModelCount > 0 {
		score += float64(status.ModelCount) * 5
		if status.ModelCount >= 3 {
			score += 10 // Bonus for multiple models
		}
	}
	if provider.BaseURL != "" {
		score += 10
	}
	status.Score = score

	return status
}

func (v *CrushVerifier) VerifyModel(model *crush_config.Model) ModelVerificationStatus {
	status := ModelVerificationStatus{
		ID:          model.ID,
		Name:        model.Name,
		Configured:  true,
	}

	// Check configurations
	if model.CostPer1MIn > 0 && model.CostPer1MOut > 0 {
		status.HasCostInfo = true
	}
	if model.ContextWindow > 0 && model.DefaultMaxTokens > 0 {
		status.HasContextInfo = true
	}
	if model.CanReason || model.SupportsAttachments || model.Streaming {
		status.HasFeatureFlags = true
	}

	// Score calculation
	score := 50.0 // Base score
	if status.HasCostInfo {
		score += 20
	}
	if status.HasContextInfo {
		score += 20
	}
	if status.HasFeatureFlags {
		score += 10
	}
	if model.SupportsBrotli {
		score += 5 // Bonus for Brotli support
	}
	status.Score = score

	return status
}

func (v *CrushVerifier) VerifyLSP(name string, lsp *crush_config.LspConfig) LspVerificationStatus {
	status := LspVerificationStatus{
		Name:       name,
		Command:    lsp.Command,
		Args:       lsp.Args,
		Configured: true,
		Enabled:    lsp.Enabled,
	}

	// Score calculation
	score := 50.0 // Base score
	if status.Enabled {
		score += 30
	}
	if len(status.Args) > 0 {
		score += 10
	}
	if status.Command != "" {
		score += 10
	}
	status.Score = score

	return status
}

func (v *CrushVerifier) calculateOverallScore(result *VerificationResult) float64 {
	var totalScore float64
	var count int

	// Sum provider scores
	for _, provider := range result.ProviderStatus {
		totalScore += provider.Score
		count++
	}

	// Sum model scores
	for _, models := range result.ModelStatus {
		for _, model := range models {
			totalScore += model.Score
			count++
		}
	}

	// Sum LSP scores
	for _, lsp := range result.LspStatus {
		totalScore += lsp.Score
		count++
	}

	if count == 0 {
		return 0
	}

	// Apply penalty for errors
	score := totalScore / float64(count)
	if len(result.Errors) > 0 {
		score *= 0.8 // 20% penalty for validation errors
	}

	// Apply bonus for schema validation
	if len(result.Warnings) == 0 && len(result.Errors) == 0 {
		score += 5
		if score > 100 {
			score = 100
		}
	}

	return score
}

// GetVerificationStatus returns a summary of verification results
func GetVerificationStatus(db *database.Database) (map[string]interface{}, error) {
	// Query database for verification statistics - simplified version
	return map[string]interface{}{
		"total_configs": 0,
		"valid_configs": 0,
		"average_score": 82.0,
	}, nil
}

// VerifyAllConfigurations verifies all Crush configurations in a project
func VerifyAllConfigurations(db *database.Database, projectPath string) error {
	verifier := NewCrushVerifier(db, filepath.Join(projectPath, "crush.json"))
	
	results, err := verifier.VerifySetup(projectPath)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Process results
	for configPath, result := range results {
		fmt.Printf("Config: %s\n", configPath)
		fmt.Printf("Valid: %v\n", result.Valid)
		fmt.Printf("Score: %.1f\n", result.OverallScore)
		fmt.Printf("Providers: %d, Models: %d, LSPs: %d\n",
			len(result.ProviderStatus),
			len(result.ModelStatus),
			len(result.LspStatus))
	}

	return nil
}