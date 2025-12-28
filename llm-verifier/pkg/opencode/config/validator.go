package opencode_config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ValidationResult represents the result of a configuration validation
type ValidationResult struct {
	Valid    bool             `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

// ValidationWarning represents a non-critical validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SchemaValidator handles JSON schema validation for OpenCode configurations
type SchemaValidator struct {
	loader ConfigLoader
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		loader: ConfigLoader{},
	}
}

// ValidateFile validates a configuration file
func (sv *SchemaValidator) ValidateFile(path string) (*ValidationResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Strip JSONC comments
	cleanContent := stripJSONCComments(string(content))

	return sv.ValidateFromReader(bytes.NewReader([]byte(cleanContent)))
}

// ValidateFromReader validates configuration from a reader
func (sv *SchemaValidator) ValidateFromReader(r io.Reader) (*ValidationResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	// Parse JSON to validate structure
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Basic validation
	result := sv.validateStructure(config)
	
	return result, nil
}

// ValidateDirectory validates all configuration files in a directory
func (sv *SchemaValidator) ValidateDirectory(dir string) (map[string]*ValidationResult, error) {
	results := make(map[string]*ValidationResult)

	// Check for main config files
	for _, filename := range []string{"opencode.jsonc", "opencode.json"} {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			result, err := sv.ValidateFile(path)
			if err != nil {
				return nil, err
			}
			results[path] = result
		}
	}

	// Check for .opencode directory
	opencodeDir := filepath.Join(dir, ".opencode")
	if info, err := os.Stat(opencodeDir); err == nil && info.IsDir() {
		// Validate config files in .opencode directory
		for _, filename := range []string{"opencode.jsonc", "opencode.json"} {
			path := filepath.Join(opencodeDir, filename)
			if _, err := os.Stat(path); err == nil {
				result, err := sv.ValidateFile(path)
				if err != nil {
					return nil, err
				}
				results[path] = result
			}
		}

		// Additional validation for .opencode structure
		if err := sv.validateOpenCodeDirectoryStructure(opencodeDir); err != nil {
			return nil, err
		}
	}

	return results, nil
}

// validateStructure validates the structure of a configuration
func (sv *SchemaValidator) validateStructure(config map[string]interface{}) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Check for required fields
	if _, ok := config["provider"]; !ok {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "provider",
			Message: "at least one provider must be configured",
		})
		result.Valid = false
	}

	// Validate provider configurations
	if providers, ok := config["provider"].(map[string]interface{}); ok {
		sv.validateProviders(providers, &result.Errors)
	}

	// Validate agent configurations
	if agents, ok := config["agent"].(map[string]interface{}); ok {
		sv.validateAgents(agents, &result.Errors)
	}

	// Validate MCP configurations
	if mcps, ok := config["mcp"].(map[string]interface{}); ok {
		sv.validateMCPs(mcps, &result.Errors)
	}

	return result
}

func (sv *SchemaValidator) validateProviders(providers map[string]interface{}, errors *[]ValidationError) {
	for name, provider := range providers {
		if p, ok := provider.(map[string]interface{}); ok {
			// Validate provider structure
			if _, hasOptions := p["options"]; !hasOptions && p["model"] == nil {
				*errors = append(*errors, ValidationError{
					Field:   fmt.Sprintf("provider.%s", name),
					Message: "provider must have either options or model configured",
				})
			}
		}
	}
}

func (sv *SchemaValidator) validateAgents(agents map[string]interface{}, errors *[]ValidationError) {
	for name, agent := range agents {
		if a, ok := agent.(map[string]interface{}); ok {
			// Validate agent structure
			hasModel := a["model"] != nil
			hasPrompt := a["prompt"] != nil
			
			if !hasModel && !hasPrompt {
				*errors = append(*errors, ValidationError{
					Field:   fmt.Sprintf("agent.%s", name),
					Message: "agent must have either model or prompt configured",
				})
			}
		}
	}
}

func (sv *SchemaValidator) validateMCPs(mcps map[string]interface{}, errors *[]ValidationError) {
	for name, mcp := range mcps {
		if m, ok := mcp.(map[string]interface{}); ok {
			typeVal, hasType := m["type"].(string)
			if !hasType || (typeVal != "local" && typeVal != "remote") {
				*errors = append(*errors, ValidationError{
					Field:   fmt.Sprintf("mcp.%s.type", name),
					Message: "type must be either 'local' or 'remote'",
				})
			}

			if typeVal == "local" && m["command"] == nil {
				*errors = append(*errors, ValidationError{
					Field:   fmt.Sprintf("mcp.%s.command", name),
					Message: "command is required for local MCP servers",
				})
			}

			if typeVal == "remote" && m["url"] == nil {
				*errors = append(*errors, ValidationError{
					Field:   fmt.Sprintf("mcp.%s.url", name),
					Message: "url is required for remote MCP servers",
				})
			}
		}
	}
}

func (sv *SchemaValidator) validateOpenCodeDirectoryStructure(dir string) error {
	// Check for required subdirectories
	requiredDirs := []string{"agent", "command"}
	for _, reqDir := range requiredDirs {
		path := filepath.Join(dir, reqDir)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			// Not an error if they don't exist, but log a warning
			continue
		}
	}
	return nil
}

func stripJSONCComments(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		// Remove single line comments
		if idx := strings.Index(line, "//"); idx >= 0 {
			// Check if it's inside a string
			inString := false
			for i := 0; i < idx; i++ {
				if line[i] == '"' && (i == 0 || line[i-1] != '\\') {
					inString = !inString
				}
			}
			if !inString {
				line = line[:idx]
			}
		}
		result.WriteString(line)
		result.WriteString("\n")
	}
	
	return result.String()
}

// LoadAndParse loads and parses a configuration file
func LoadAndParse(path string) (*Config, error) {
	loader := ConfigLoader{}
	return loader.LoadFromFile(path)
}

// LoadAndParseResolved loads and parses a configuration file with environment variable resolution
func LoadAndParseResolved(path string, strict bool) (*Config, error) {
	return LoadAndResolveConfig(path, strict)
}

// SaveConfig saves a configuration to a file in JSON format
func SaveConfig(config *Config, path string) error {
	loader := ConfigLoader{}
	return loader.SaveToFile(config, path)
}

// CreateDefaultConfig creates a default OpenCode configuration
func CreateDefaultConfig() *Config {
	return &Config{
		Provider: map[string]ProviderConfig{
			"openai": {
				Options: map[string]interface{}{
					"api_key": "${OPENAI_API_KEY}",
				},
			},
		},
		Agent: map[string]AgentConfig{
			"build": {
				Model: "openai/gpt-4",
				Prompt: "You are a build agent. Help the user with development tasks.",
			},
		},
	}
}