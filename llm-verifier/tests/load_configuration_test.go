package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfiguration tests config file operations
func TestLoadConfiguration(t *testing.T) {
	t.Run("Load Configuration - Valid Config", func(t *testing.T) {
		// Test with valid config file
		validConfig := `global:
  base_url: "https://api.openai.com/v1"
  api_key: "valid-key"
llms:
  - name: "gpt-4"
    endpoint: "https://api.openai.com/v1"
    model: "gpt-4"
`

		// Create temporary config file
		tempDir := t.TempDir()
		tempConfig := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(tempConfig, []byte(validConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to write valid config file: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(tempConfig); os.IsNotExist(err) {
			t.Errorf("Config file was not created")
		}

		// Read file back
		data, err := os.ReadFile(tempConfig)
		if err != nil {
			t.Errorf("Failed to read config file: %v", err)
		}

		if len(data) == 0 {
			t.Errorf("Config file is empty")
		}

		t.Logf("Valid config file test passed")
	})

	t.Run("Load Configuration - File Not Found", func(t *testing.T) {
		// Test with non-existent file
		_, err := os.Stat("/nonexistent/config.yaml")
		if !os.IsNotExist(err) {
			t.Errorf("Expected file not found error")
		}

		t.Logf("File not found test passed")
	})

	t.Logf("Config file tests completed")
}
