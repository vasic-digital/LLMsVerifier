package opencode_verifier

import (
	"encoding/json"
	"fmt"
	"log"
	
	"llm-verifier/database"
)

// Integration with LLM Verifier

// OpenCodeIntegration handles integration with the main LLM Verifier
type OpenCodeIntegration struct {
	db *database.Database
}

// NewOpenCodeIntegration creates a new integration instance
func NewOpenCodeIntegration(db *database.Database) *OpenCodeIntegration {
	return &OpenCodeIntegration{
		db: db,
	}
}

// VerifyOpenCodeConfig verifies an OpenCode configuration and stores the result
func (oci *OpenCodeIntegration) VerifyOpenCodeConfig(configPath string) error {
	verifier := NewOpenCodeVerifier(oci.db, configPath)
	
	result, err := verifier.VerifyConfiguration()
	if err != nil {
		return fmt.Errorf("failed to verify OpenCode config: %w", err)
	}

	// Store the result in the database
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	data := map[string]interface{}{
		"config_path": configPath,
		"valid":       result.Valid,
		"score":       result.OverallScore,
		"result":      string(resultJSON),
		"errors":      len(result.Errors),
		"warnings":    len(result.Warnings),
		"providers":   len(result.ProviderStatus),
		"agents":      len(result.AgentStatus),
		"mcps":        len(result.McpStatus),
	}

	if err := oci.db.Create("verifications", data); err != nil {
		return fmt.Errorf("failed to store verification: %w", err)
	}

	log.Printf("OpenCode verification completed: %s (valid: %v, score: %.1f)",
		configPath, result.Valid, result.OverallScore)

	return nil
}

// GetOpenCodeStatus returns the current OpenCode verification status
func (oci *OpenCodeIntegration) GetOpenCodeStatus() (map[string]interface{}, error) {
	return GetVerificationStatus(oci.db)
}

// SaveOpenCodeResult saves an OpenCode verification result to the database
func (oci *OpenCodeIntegration) SaveOpenCodeResult(result *VerificationResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	data := map[string]interface{}{
		"config_path": result.ConfigFile,
		"valid":       result.Valid,
		"score":       result.OverallScore,
		"result":      string(resultJSON),
		"created_at":  "",
		"errors":      len(result.Errors),
		"warnings":    len(result.Warnings),
		"providers":   len(result.ProviderStatus),
		"agents":      len(result.AgentStatus),
		"mcps":        len(result.McpStatus),
	}

	if err := oci.db.Create("open_code_results", data); err != nil {
		return fmt.Errorf("failed to store result: %w", err)
	}

	return nil
}

// ListOpenCodeResults returns all OpenCode verification results
func (oci *OpenCodeIntegration) ListOpenCodeResults() ([]map[string]interface{}, error) {
	query := map[string]interface{}{
		"table": "open_code_results",
		"limit": 100,
	}

	results, err := oci.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
	}

	return results, nil
}

// DeleteOpenCodeResult deletes a specific OpenCode verification result
func (oci *OpenCodeIntegration) DeleteOpenCodeResult(id int64) error {
	if err := oci.db.Delete("open_code_results", map[string]interface{}{"id": id}); err != nil {
		return fmt.Errorf("failed to delete result: %w", err)
	}

	return nil
}

// GetHighScoringConfigs returns configs that scored above a threshold
func (oci *OpenCodeIntegration) GetHighScoringConfigs(threshold float64) ([]map[string]interface{}, error) {
	query := map[string]interface{}{
		"table": "open_code_results",
		"where": map[string]interface{}{
			"score >": threshold,
		},
		"order_by": "score DESC",
		"limit": 50,
	}

	configs, err := oci.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query high scoring configs: %w", err)
	}

	return configs, nil
}

// GetOpenCodeStats returns statistics about OpenCode verifications
func (oci *OpenCodeIntegration) GetOpenCodeStats() (map[string]interface{}, error) {
	total, err := oci.db.Count("open_code_results", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	valid, err := oci.db.Count("open_code_results", map[string]interface{}{
		"valid": true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get valid count: %w", err)
	}

	highScoring, err := oci.db.Count("open_code_results", map[string]interface{}{
		"score >=": 80.0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get high scoring count: %w", err)
	}

	averageScore := 0.0
	if total > 0 {
		// Calculate average score - simplified for demonstration
		averageScore = 75.0 // Would query actual average from database
	}

	return map[string]interface{}{
		"total_configs":    total,
		"valid_configs":    valid,
		"valid_percentage": float64(valid) / float64(total) * 100,
		"high_scoring":     highScoring,
		"average_score":    averageScore,
	}, nil
}