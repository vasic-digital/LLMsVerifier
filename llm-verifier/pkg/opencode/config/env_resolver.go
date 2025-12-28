package opencode_config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// EnvResolver handles resolution of environment variable placeholders
type EnvResolver struct {
	strict bool // If true, fail on missing env vars. If false, leave unchanged.
}

// NewEnvResolver creates a new environment variable resolver
func NewEnvResolver(strict bool) *EnvResolver {
	return &EnvResolver{strict: strict}
}

// ResolveInString resolves environment variables in a string
// Supports formats: ${VAR}, ${VAR:-default}, $VAR
func (er *EnvResolver) ResolveInString(s string) (string, error) {
	// Pattern to match ${VAR} or ${VAR:-default}
	pattern := regexp.MustCompile(`\$\{([^}]+)\}`)
	
	result := pattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract content between ${ and }
		content := match[2 : len(match)-1]
		
		// Check for default value: VAR:-default
		if idx := strings.Index(content, ":-"); idx != -1 {
			varName := content[:idx]
			defaultValue := content[idx+2:]
			
			if value := os.Getenv(varName); value != "" {
				return value
			}
			return defaultValue
		}
		
		// Simple variable: VAR
		varName := content
		value := os.Getenv(varName)
		
		if value == "" && er.strict {
			// In strict mode, return the original to indicate error
			return match
		}
		
		return value
	})
	
	// Also support $VAR format (without braces)
	pattern2 := regexp.MustCompile(`\$([A-Z_][A-Z0-9_]*)`)
	result = pattern2.ReplaceAllStringFunc(result, func(match string) string {
		varName := match[1:]
		value := os.Getenv(varName)
		
		if value == "" && er.strict {
			return match
		}
		
		return value
	})
	
	return result, nil
}

// ResolveInMap resolves environment variables in a map
func (er *EnvResolver) ResolveInMap(m map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for key, value := range m {
		resolvedValue, err := er.ResolveInterface(value)
		if err != nil {
			return nil, fmt.Errorf("error resolving %s: %w", key, err)
		}
		result[key] = resolvedValue
	}
	
	return result, nil
}

// ResolveInterface resolves environment variables in an interface{}
func (er *EnvResolver) ResolveInterface(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return er.ResolveInString(v)
	case map[string]interface{}:
		return er.ResolveInMap(v)
	case []interface{}:
		return er.ResolveInSlice(v)
	default:
		return value, nil
	}
}

// ResolveInSlice resolves environment variables in a slice
func (er *EnvResolver) ResolveInSlice(slice []interface{}) ([]interface{}, error) {
	result := make([]interface{}, len(slice))
	
	for i, value := range slice {
		resolvedValue, err := er.ResolveInterface(value)
		if err != nil {
			return nil, fmt.Errorf("error resolving slice element %d: %w", i, err)
		}
		result[i] = resolvedValue
	}
	
	return result, nil
}

// ResolveConfig resolves environment variables in a Config object
func (er *EnvResolver) ResolveConfig(config *Config) (*Config, error) {
	// Marshal to JSON
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Resolve in string
	resolvedJSON, err := er.ResolveInString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve env vars: %w", err)
	}
	
	// Check for unresolved placeholders in strict mode
	if er.strict {
		if strings.Contains(resolvedJSON, "${") {
			return nil, fmt.Errorf("unresolved environment variable placeholders found in config")
		}
	}
	
	// Unmarshal back to Config
	var resolvedConfig Config
	if err := json.Unmarshal([]byte(resolvedJSON), &resolvedConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resolved config: %w", err)
	}
	
	return &resolvedConfig, nil
}

// LoadAndResolveConfig loads a config file and resolves environment variables
func LoadAndResolveConfig(path string, strict bool) (*Config, error) {
	// Load raw config
	loader := ConfigLoader{}
	config, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	
	// Resolve environment variables
	resolver := NewEnvResolver(strict)
	resolvedConfig, err := resolver.ResolveConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve env vars: %w", err)
	}
	
	return resolvedConfig, nil
}

// ValidateEnvVars checks if all required environment variables are set
func ValidateEnvVars(config *Config) []string {
	var missingVars []string
	
	// Check provider options for env var references
	for providerName, provider := range config.Provider {
		for key, value := range provider.Options {
			if str, ok := value.(string); ok {
				if strings.Contains(str, "${") {
					// Extract var name
					pattern := regexp.MustCompile(`\$\{([^:}]+)`)
					matches := pattern.FindStringSubmatch(str)
					if len(matches) > 1 {
						varName := matches[1]
						if os.Getenv(varName) == "" {
							missingVars = append(missingVars, 
								fmt.Sprintf("Provider '%s' option '%s' references unset env var: %s", 
									providerName, key, varName))
						}
					}
				}
			}
		}
	}
	
	return missingVars
}