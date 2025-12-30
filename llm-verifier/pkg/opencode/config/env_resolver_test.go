package opencode_config

import (
	"os"
	"strings"
	"testing"
)

// TestEnvResolver_ResolveInString tests string resolution
func TestEnvResolver_ResolveInString(t *testing.T) {
	// Set up test environment
	os.Setenv("TEST_API_KEY", "sk-test123")
	os.Setenv("TEST_BASE_URL", "https://api.test.com")
	defer os.Unsetenv("TEST_API_KEY")
	defer os.Unsetenv("TEST_BASE_URL")

	tests := []struct {
		name     string
		input    string
		expected string
		strict   bool
		wantErr  bool
	}{
		{
			name:     "simple variable",
			input:    "${TEST_API_KEY}",
			expected: "sk-test123",
			strict:   true,
			wantErr:  false,
		},
		{
			name:     "variable with default",
			input:    "${MISSING_VAR:-default_value}",
			expected: "default_value",
			strict:   true,
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			input:    "Key: ${TEST_API_KEY}, URL: ${TEST_BASE_URL}",
			expected: "Key: sk-test123, URL: https://api.test.com",
			strict:   true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewEnvResolver(tt.strict)
			result, err := resolver.ResolveInString(tt.input)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveInString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if result != tt.expected {
				t.Errorf("ResolveInString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestEnvResolver_ResolveConfig tests full config resolution
func TestEnvResolver_ResolveConfig(t *testing.T) {
	// Set up test environment
	os.Setenv("OPENAI_API_KEY", "sk-openai123")
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant456")
	defer os.Unsetenv("OPENAI_API_KEY")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	config := &Config{
		Provider: map[string]ProviderConfig{
			"openai": {
				Options: map[string]interface{}{
					"api_key": "${OPENAI_API_KEY}",
					"baseURL": "https://api.openai.com/v1",
				},
				Model: "gpt-4",
			},
			"anthropic": {
				Options: map[string]interface{}{
					"api_key": "${ANTHROPIC_API_KEY}",
				},
				Model: "claude-3",
			},
		},
	}

	resolver := NewEnvResolver(true)
	resolved, err := resolver.ResolveConfig(config)
	if err != nil {
		t.Fatalf("ResolveConfig() error = %v", err)
	}

	// Check OpenAI provider
	openaiProvider := resolved.Provider["openai"]
	if openaiProvider.Options["api_key"] != "sk-openai123" {
		t.Errorf("OpenAI api_key = %v, want sk-openai123", openaiProvider.Options["api_key"])
	}

	// Check Anthropic provider
	anthropicProvider := resolved.Provider["anthropic"]
	if anthropicProvider.Options["api_key"] != "sk-ant456" {
		t.Errorf("Anthropic api_key = %v, want sk-ant456", anthropicProvider.Options["api_key"])
	}
}

// TestEnvResolver_RealWorldScenario tests a real-world scenario
func TestEnvResolver_RealWorldScenario(t *testing.T) {
	// Set up environment
	os.Setenv("HUGGINGFACE_API_KEY", "hf_test123456")
	os.Setenv("OPENAI_API_KEY", "sk-openai789")
	defer os.Unsetenv("HUGGINGFACE_API_KEY")
	defer os.Unsetenv("OPENAI_API_KEY")

	// Create a temporary config file with real structure
	testJSON := `{
		"provider": {
			"huggingface": {
				"options": {
					"apiKey": "${HUGGINGFACE_API_KEY}",
					"baseURL": "https://api-inference.huggingface.co"
				}
			},
			"openai": {
				"options": {
					"apiKey": "${OPENAI_API_KEY}",
					"baseURL": "https://api.openai.com/v1"
				}
			}
		}
	}`

	// Create temp file
	tmpFile, err := os.CreateTemp("", "opencode-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testJSON); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Load and resolve
	resolved, err := LoadAndResolveConfig(tmpFile.Name(), true)
	if err != nil {
		t.Fatalf("LoadAndResolveConfig() error = %v", err)
	}

	// Verify
	hfProvider := resolved.Provider["huggingface"]
	if hfProvider.Options["apiKey"] != "hf_test123456" {
		t.Errorf("HuggingFace apiKey = %v, want hf_test123456", hfProvider.Options["apiKey"])
	}

	openaiProvider := resolved.Provider["openai"]
	if openaiProvider.Options["apiKey"] != "sk-openai789" {
		t.Errorf("OpenAI apiKey = %v, want sk-openai789", openaiProvider.Options["apiKey"])
	}

	// Verify URLs preserved
	if hfProvider.Options["baseURL"] != "https://api-inference.huggingface.co" {
		t.Errorf("HuggingFace baseURL = %v", hfProvider.Options["baseURL"])
	}
}

// TestEnvResolver_NoProviderInitError tests the fix for ProviderInitError
func TestEnvResolver_NoProviderInitError(t *testing.T) {
	os.Setenv("TEST_PROVIDER_KEY", "sk-validkey123")
	defer os.Unsetenv("TEST_PROVIDER_KEY")

	config := &Config{
		Provider: map[string]ProviderConfig{
			"test-provider": {
				Options: map[string]interface{}{
					"api_key": "${TEST_PROVIDER_KEY}",
					"baseURL": "https://api.test.com/v1",
				},
			},
		},
	}

	resolver := NewEnvResolver(true)
	resolved, err := resolver.ResolveConfig(config)
	if err != nil {
		t.Fatalf("ResolveConfig() error = %v", err)
	}

	// Verify that the placeholder was resolved to actual value
	testProvider := resolved.Provider["test-provider"]
	apiKey := testProvider.Options["api_key"]
	
	if apiKey == "${TEST_PROVIDER_KEY}" {
		t.Error("API key still contains placeholder, should be resolved!")
	}
	
	if apiKey != "sk-validkey123" {
		t.Errorf("API key = %v, want sk-validkey123", apiKey)
	}

	// This test simulates what happens in OpenCode:
	// Before fix: apiKey would be "${TEST_PROVIDER_KEY}" → ProviderInitError
	// After fix: apiKey is "sk-validkey123" → Success!
	t.Logf("✓ API key successfully resolved to: %s (no ProviderInitError)", apiKey)
}

// TestValidateEnvVars tests environment variable validation
func TestValidateEnvVars(t *testing.T) {
	os.Setenv("EXISTING_KEY", "test123")
	defer os.Unsetenv("EXISTING_KEY")

	config := &Config{
		Provider: map[string]ProviderConfig{
			"test": {
				Options: map[string]interface{}{
					"api_key": "${EXISTING_KEY}",
					"missing": "${NONEXISTENT_KEY}",
				},
			},
		},
	}

	missingVars := ValidateEnvVars(config)
	if len(missingVars) == 0 {
		t.Error("Expected to find missing env var, got none")
	}

	expectedMsg := "NONEXISTENT_KEY"
	found := false
	for _, msg := range missingVars {
		if strings.Contains(msg, expectedMsg) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected message containing '%s', got %v", expectedMsg, missingVars)
	}
}

// TestLoadAndResolveConfigIntegration tests full integration
func TestLoadAndResolveConfigIntegration(t *testing.T) {
	// Create temp config file
	tmpFile, err := os.CreateTemp("", "opencode-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `{
		"provider": {
			"test": {
				"options": {
					"api_key": "${TEST_API_KEY}",
					"baseURL": "https://api.test.com"
				}
			}
		}
	}`
	
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Set environment
	os.Setenv("TEST_API_KEY", "sk-integration123")
	defer os.Unsetenv("TEST_API_KEY")

	// Load and resolve
	resolved, err := LoadAndResolveConfig(tmpFile.Name(), true)
	if err != nil {
		t.Fatalf("LoadAndResolveConfig() error = %v", err)
	}

	// Verify
	testProvider := resolved.Provider["test"]
	if testProvider.Options["api_key"] != "sk-integration123" {
		t.Errorf("Resolved api_key = %v, want sk-integration123", testProvider.Options["api_key"])
	}
}

// TestStripJSONCComments tests JSONC comment stripping
// Note: Current implementation only strips single-line (//) comments
func TestStripJSONCComments(t *testing.T) {
	input := `{
		// This is a comment
		"provider": {
			"test": {} // inline comment
		}
	}`

	// Single-line comments are stripped, leaving the line content before //
	// Note: Leading whitespace is preserved, and trailing space before // is kept
	expected := "{" + "\n" +
		"\t\t" + "\n" + // Line with comment removed, tabs preserved
		"\t\t" + "\"provider\": {" + "\n" +
		"\t\t\t" + "\"test\": {} " + "\n" + // Trailing space before // is kept
		"\t\t}" + "\n" +
		"\t}" + "\n"

	result := stripJSONCComments(input)
	if result != expected {
		t.Errorf("stripJSONCComments() result mismatch\nGot: %q\nWant: %q", result, expected)
	}
}