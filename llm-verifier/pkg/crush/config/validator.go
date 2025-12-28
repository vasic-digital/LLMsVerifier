package crush_config

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

// SchemaValidator handles Crush configuration validation
type SchemaValidator struct {
	loader ConfigLoader
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		loader: ConfigLoader{},
	}
}

// ValidateFile validates a Crush configuration file
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

	// Check if file is JSON
	if !strings.HasSuffix(path, ".json") {
		return nil, fmt.Errorf("crush configuration must be in JSON format")
	}

	return sv.ValidateFromReader(bytes.NewReader(content))
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
	result := sv.validateStructure(&config)
	
	return result, nil
}

// ValidateDirectory validates all Crush configuration files in a directory
func (sv *SchemaValidator) ValidateDirectory(dir string) (map[string]*ValidationResult, error) {
	results := make(map[string]*ValidationResult)

	// Check for crush.json
	for _, filename := range []string{"crush.json"} {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			result, err := sv.ValidateFile(path)
			if err != nil {
				return nil, err
			}
			results[path] = result
		}
	}

	// Check for .crush directory
	crushDir := filepath.Join(dir, ".crush")
	if info, err := os.Stat(crushDir); err == nil && info.IsDir() {
		// Validate config files in .crush directory
		for _, filename := range []string{"crush.json"} {
			path := filepath.Join(crushDir, filename)
			if _, err := os.Stat(path); err == nil {
				result, err := sv.ValidateFile(path)
				if err != nil {
					return nil, err
				}
				results[path] = result
			}
		}
	}

	return results, nil
}

// validateStructure validates the structure of a configuration
func (sv *SchemaValidator) validateStructure(config *map[string]interface{}) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Check for required fields
	if _, ok := (*config)["providers"]; !ok {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "providers",
			Message: "at least one provider must be configured",
		})
		result.Valid = false
	}

	// Validate schema if present
	if schema, ok := (*config)["$schema"]; ok {
		if schemaStr, ok := schema.(string); ok {
			if !strings.Contains(schemaStr, "charm.land/crush") {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Field:   "$schema",
					Message: "schema should reference charm.land/crush.json",
				})
			}
		}
	}

	// Validate provider configurations
	if providers, ok := (*config)["providers"].(map[string]interface{}); ok {
		sv.validateProviders(providers, &result.Errors)
	}

	// Validate LSP configurations
	if lsp, ok := (*config)["lsp"].(map[string]interface{}); ok {
		sv.validateLSPs(lsp, &result.Errors)
	}

	return result
}

func (sv *SchemaValidator) validateProviders(providers map[string]interface{}, errors *[]ValidationError) {
	for name, provider := range providers {
		if p, ok := provider.(map[string]interface{}); ok {
			// Validate required provider fields
			requiredFields := []string{"name", "type", "base_url", "models"}
			for _, field := range requiredFields {
				if _, exists := p[field]; !exists {
					*errors = append(*errors, ValidationError{
						Field:   fmt.Sprintf("providers.%s.%s", name, field),
						Message: fmt.Sprintf("%s is required for providers", field),
					})
				}
			}

			// Validate models
			if models, ok := p["models"].([]interface{}); ok {
				if len(models) == 0 {
					*errors = append(*errors, ValidationError{
						Field:   fmt.Sprintf("providers.%s.models", name),
						Message: "at least one model must be configured",
					})
				}
				
				for i, model := range models {
					if m, ok := model.(map[string]interface{}); ok {
						sv.validateModel(name, i, m, errors)
					}
				}
			}

			// Validate base_url format
			if baseURL, ok := p["base_url"].(string); ok {
				if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
					*errors = append(*errors, ValidationError{
						Field:   fmt.Sprintf("providers.%s.base_url", name),
						Message: "base_url must start with http:// or https://",
					})
				}
			}
		}
	}
}

func (sv *SchemaValidator) validateModel(providerName string, index int, model map[string]interface{}, errors *[]ValidationError) {
	prefix := fmt.Sprintf("providers.%s.models[%d]", providerName, index)
	
	requiredFields := []string{"id", "name", "cost_per_1m_in", "cost_per_1m_out", "context_window", "default_max_tokens"}
	for _, field := range requiredFields {
		if _, exists := model[field]; !exists {
			*errors = append(*errors, ValidationError{
				Field:   fmt.Sprintf("%s.%s", prefix, field),
				Message: fmt.Sprintf("%s is required for models", field),
			})
		}
	}

	// Validate numeric fields
	if costIn, ok := model["cost_per_1m_in"].(float64); ok {
		if costIn < 0 {
			*errors = append(*errors, ValidationError{
				Field:   fmt.Sprintf("%s.cost_per_1m_in", prefix),
				Message: "cost_per_1m_in must be non-negative",
			})
		}
	}

	if costOut, ok := model["cost_per_1m_out"].(float64); ok {
		if costOut < 0 {
			*errors = append(*errors, ValidationError{
				Field:   fmt.Sprintf("%s.cost_per_1m_out", prefix),
				Message: "cost_per_1m_out must be non-negative",
			})
		}
	}

	if contextWindow, ok := model["context_window"].(float64); ok {
		if contextWindow <= 0 {
			*errors = append(*errors, ValidationError{
				Field:   fmt.Sprintf("%s.context_window", prefix),
				Message: "context_window must be positive",
			})
		}
	}

	if maxTokens, ok := model["default_max_tokens"].(float64); ok {
		if maxTokens <= 0 {
			*errors = append(*errors, ValidationError{
				Field:   fmt.Sprintf("%s.default_max_tokens", prefix),
				Message: "default_max_tokens must be positive",
			})
		}
	}

	// Validate boolean fields
	boolFields := []string{"can_reason", "supports_attachments", "streaming", "supports_brotli"}
	for _, field := range boolFields {
		if value, exists := model[field]; exists {
			if _, ok := value.(bool); !ok {
				*errors = append(*errors, ValidationError{
					Field:   fmt.Sprintf("%s.%s", prefix, field),
					Message: fmt.Sprintf("%s must be a boolean", field),
				})
			}
		}
	}
}

func (sv *SchemaValidator) validateLSPs(lsp map[string]interface{}, errors *[]ValidationError) {
	for name, config := range lsp {
		if c, ok := config.(map[string]interface{}); ok {
			// Validate required LSP fields
			if _, exists := c["command"]; !exists {
				*errors = append(*errors, ValidationError{
					Field:   fmt.Sprintf("lsp.%s.command", name),
					Message: "command is required for LSP configurations",
				})
			}

			// Validate enabled field
			if enabled, exists := c["enabled"]; exists {
				if _, ok := enabled.(bool); !ok {
					*errors = append(*errors, ValidationError{
						Field:   fmt.Sprintf("lsp.%s.enabled", name),
						Message: "enabled must be a boolean",
					})
				}
			}

			// Validate args if present
			if args, exists := c["args"]; exists {
				if _, ok := args.([]interface{}); !ok {
					*errors = append(*errors, ValidationError{
						Field:   fmt.Sprintf("lsp.%s.args", name),
						Message: "args must be an array",
					})
				}
			}
		}
	}
}