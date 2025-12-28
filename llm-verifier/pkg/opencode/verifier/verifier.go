package opencode_verifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	opencode_config "llm-verifier/pkg/opencode/config"
	"llm-verifier/database"
)

// OpenCodeVerifier handles verification of OpenCode configurations and setups
type OpenCodeVerifier struct {
	db         *database.Database
	validator  *opencode_config.SchemaValidator
	configPath string
}

// NewOpenCodeVerifier creates a new OpenCode verifier
func NewOpenCodeVerifier(db *database.Database, configPath string) *OpenCodeVerifier {
	return &OpenCodeVerifier{
		db:         db,
		validator:  opencode_config.NewSchemaValidator(),
		configPath: configPath,
	}
}

// VerificationResult represents the result of a configuration verification
type VerificationResult struct {
	ConfigFile     string                                   `json:"config_file"`
	Valid          bool                                     `json:"valid"`
	Errors         []opencode_config.ValidationError        `json:"errors,omitempty"`
	Warnings       []opencode_config.ValidationWarning      `json:"warnings,omitempty"`
	ProviderStatus map[string]ProviderVerificationStatus    `json:"provider_status,omitempty"`
	AgentStatus    map[string]AgentVerificationStatus       `json:"agent_status,omitempty"`
	McpStatus      map[string]McpVerificationStatus         `json:"mcp_status,omitempty"`
	OverallScore   float64                                  `json:"overall_score"`
}

// ProviderVerificationStatus represents the verification status of a provider
type ProviderVerificationStatus struct {
	Name        string  `json:"name"`
	Configured  bool    `json:"configured"`
	HasAPIKey   bool    `json:"has_api_key"`
	Verified    bool    `json:"verified"`
	Error       string  `json:"error,omitempty"`
	Score       float64 `json:"score"`
}

// AgentVerificationStatus represents the verification status of an agent
type AgentVerificationStatus struct {
	Name            string  `json:"name"`
	Configured      bool    `json:"configured"`
	HasModel        bool    `json:"has_model"`
	HasPrompt       bool    `json:"has_prompt"`
	ToolsConfigured int     `json:"tools_configured"`
	Score           float64 `json:"score"`
}

// McpVerificationStatus represents the verification status of an MCP server
type McpVerificationStatus struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Configured bool    `json:"configured"`
	Enabled    bool    `json:"enabled"`
	Command    string  `json:"command,omitempty"`
	URL        string  `json:"url,omitempty"`
	Score      float64 `json:"score"`
}

// VerifyConfiguration verifies an OpenCode configuration file
func (v *OpenCodeVerifier) VerifyConfiguration() (*VerificationResult, error) {
	result := &VerificationResult{
		ConfigFile:     v.configPath,
		ProviderStatus: make(map[string]ProviderVerificationStatus),
		AgentStatus:    make(map[string]AgentVerificationStatus),
		McpStatus:      make(map[string]McpVerificationStatus),
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

	var cfg opencode_config.Config
	if err := json.Unmarshal(configContent, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Verify providers and models
	for name, provider := range cfg.Provider {
		status := v.VerifyProvider(name, &provider)
		result.ProviderStatus[name] = status
	}

	// Verify agents
	for name, agent := range cfg.Agent {
		status := v.VerifyAgent(name, &agent)
		result.AgentStatus[name] = status
	}

	// Verify MCP servers
	for name, mcp := range cfg.Mcp {
		status := v.VerifyMCP(name, &mcp)
		result.McpStatus[name] = status
	}

	// Calculate overall score
	result.OverallScore = v.calculateOverallScore(result)

	return result, nil
}

// VerifySetup verifies a complete OpenCode project setup
func (v *OpenCodeVerifier) VerifySetup(projectPath string) (map[string]*VerificationResult, error) {
	results := make(map[string]*VerificationResult)

	// Check for .opencode directory
	opencodeDir := filepath.Join(projectPath, ".opencode")
	if _, err := os.Stat(opencodeDir); err == nil {
		// Verify configuration files in .opencode
		configFiles := []string{"opencode.jsonc", "opencode.json"}
		for _, filename := range configFiles {
			configPath := filepath.Join(opencodeDir, filename)
			if _, err := os.Stat(configPath); err == nil {
				verifier := NewOpenCodeVerifier(v.db, configPath)
				result, err := verifier.VerifyConfiguration()
				if err != nil {
					return nil, err
				}
				results[configPath] = result
			}
		}
	}

	// Check for opencode.json in project root
	rootConfig := filepath.Join(projectPath, "opencode.json")
	if _, err := os.Stat(rootConfig); err == nil {
		verifier := NewOpenCodeVerifier(v.db, rootConfig)
		result, err := verifier.VerifyConfiguration()
		if err != nil {
			return nil, err
		}
		results[rootConfig] = result
	}

	return results, nil
}

func (v *OpenCodeVerifier) VerifyProvider(name string, provider *opencode_config.ProviderConfig) ProviderVerificationStatus {
	status := ProviderVerificationStatus{
		Name:       name,
		Configured: true,
	}

	// Check if has API key
	apiKeyEnv := fmt.Sprintf("%s_API_KEY", strings.ToUpper(strings.ReplaceAll(name, "-", "_")))
	if os.Getenv(apiKeyEnv) != "" {
		status.HasAPIKey = true
	}

	// Score calculation
	score := 50.0 // Base score
	if status.HasAPIKey {
		score += 30
	}
	if len(provider.Options) > 0 {
		score += 10
	}
	if provider.Model != "" {
		score += 10
	}
	status.Score = score

	return status
}

func (v *OpenCodeVerifier) VerifyAgent(name string, agent *opencode_config.AgentConfig) AgentVerificationStatus {
	status := AgentVerificationStatus{
		Name:       name,
		Configured: true,
	}

	// Check configurations
	if agent.Model != "" {
		status.HasModel = true
	}
	if agent.Prompt != "" {
		status.HasPrompt = true
	}
	if agent.Tools != nil {
		status.ToolsConfigured = len(agent.Tools)
	}

	// Score calculation
	score := 50.0 // Base score
	if status.HasModel {
		score += 20
	}
	if status.HasPrompt {
		score += 20
	}
	if status.ToolsConfigured > 0 {
		score += float64(status.ToolsConfigured) * 2
	}
	if agent.Description != "" {
		score += 5
	}
	status.Score = score

	return status
}

func (v *OpenCodeVerifier) VerifyMCP(name string, mcp *opencode_config.McpConfig) McpVerificationStatus {
	status := McpVerificationStatus{
		Name:       name,
		Type:       mcp.Type,
		Configured: true,
	}

	if mcp.Enabled == nil || *mcp.Enabled {
		status.Enabled = true
	}

	if mcp.Type == "local" {
		status.Command = strings.Join(mcp.Command, " ")
	} else {
		status.URL = mcp.URL
	}

	// Score calculation
	score := 50.0 // Base score
	if status.Enabled {
		score += 20
	}
	if mcp.Timeout != nil && *mcp.Timeout > 0 {
		score += 15
	}
	if len(mcp.Environment) > 0 {
		score += 15
	}
	status.Score = score

	return status
}

func (v *OpenCodeVerifier) calculateOverallScore(result *VerificationResult) float64 {
	var totalScore float64
	var count int

	// Sum provider scores
	for _, provider := range result.ProviderStatus {
		totalScore += provider.Score
		count++
	}

	// Sum agent scores
	for _, agent := range result.AgentStatus {
		totalScore += agent.Score
		count++
	}

	// Sum MCP scores
	for _, mcp := range result.McpStatus {
		totalScore += mcp.Score
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

	return score
}

// GetVerificationStatus returns a summary of verification results
func GetVerificationStatus(db *database.Database) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_configs": 0,
		"valid_configs": 0,
		"average_score": 0.0,
	}, nil
}

// VerifyAllConfigurations verifies all OpenCode configurations in a project
func VerifyAllConfigurations(db *database.Database, projectPath string) error {
	verifier := NewOpenCodeVerifier(db, filepath.Join(projectPath, "opencode.jsonc"))
	
	results, err := verifier.VerifySetup(projectPath)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Process results
	for configPath, result := range results {
		fmt.Printf("Config: %s\n", configPath)
		fmt.Printf("Valid: %v\n", result.Valid)
		fmt.Printf("Score: %.1f\n", result.OverallScore)
	}

	return nil
}