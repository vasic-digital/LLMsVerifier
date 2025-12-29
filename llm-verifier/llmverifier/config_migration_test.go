package llmverifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMigrateOpenCodeConfig tests the migration of old OpenCode configs to new format
func TestMigrateOpenCodeConfig(t *testing.T) {
	// Create an old-style config that would cause ProviderInitError
	oldConfig := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json", // Old internal schema
		"provider": map[string]interface{}{ // Old "provider" (singular)
			"openai": map[string]interface{}{
				"options": map[string]interface{}{
					"apiKey": "${OPENAI_API_KEY}",
				},
				"models": []interface{}{}, // Empty models array
			},
			"anthropic": map[string]interface{}{
				"options": map[string]interface{}{
					"apiKey": "${ANTHROPIC_API_KEY}",
				},
				"models": []interface{}{},
			},
		},
	}

	// Migrate the config
	newConfig, err := migrateOpenCodeConfig(oldConfig)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify the migration worked

	// 1. Schema should be updated
	if schema, ok := newConfig["$schema"].(string); !ok || schema != "./opencode-schema.json" {
		t.Errorf("Expected schema './opencode-schema.json', got '%v'", newConfig["$schema"])
	}

	// 2. Should have "providers" (plural) instead of "provider" (singular)
	if _, hasProviders := newConfig["providers"]; !hasProviders {
		t.Error("Missing 'providers' section in migrated config")
	}
	if _, hasProvider := newConfig["provider"]; hasProvider {
		t.Error("Old 'provider' section should be removed")
	}

	// 3. Providers should have new format
	providers, ok := newConfig["providers"].(map[string]interface{})
	if !ok {
		t.Fatal("providers section should be an object")
	}

	// Check OpenAI provider
	openaiProvider, exists := providers["openai"]
	if !exists {
		t.Error("Expected openai provider to exist")
	}
	if openaiMap, ok := openaiProvider.(map[string]interface{}); ok {
		if apiKey, hasKey := openaiMap["apiKey"]; !hasKey || apiKey != "${OPENAI_API_KEY}" {
			t.Errorf("OpenAI provider missing or incorrect apiKey: %v", apiKey)
		}
		if _, hasDisabled := openaiMap["disabled"]; !hasDisabled {
			t.Error("OpenAI provider missing disabled field")
		}
		if _, hasProvider := openaiMap["provider"]; !hasProvider {
			t.Error("OpenAI provider missing provider field")
		}
	}

	// 4. Should have agents section
	agents, ok := newConfig["agents"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing agents section in migrated config")
	}

	requiredAgents := []string{"coder", "task", "title"}
	for _, agentName := range requiredAgents {
		if _, exists := agents[agentName]; !exists {
			t.Errorf("Missing required agent: %s", agentName)
		}
	}

	// 5. Should have all required OpenCode sections
	requiredSections := []string{"data", "tui", "shell", "autoCompact", "debug", "debugLSP"}
	for _, section := range requiredSections {
		if _, exists := newConfig[section]; !exists {
			t.Errorf("Missing required section: %s", section)
		}
	}

	t.Log("✅ Migration test passed - old config successfully converted to OpenCode-compatible format")
}

// TestMigrateOpenCodeConfigFile tests file-based migration
func TestMigrateOpenCodeConfigFile(t *testing.T) {
	// Create a temporary old config file
	oldConfig := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]interface{}{
			"openai": map[string]interface{}{
				"options": map[string]interface{}{
					"apiKey": "${OPENAI_API_KEY}",
				},
				"models": []interface{}{},
			},
		},
	}

	oldData, err := json.MarshalIndent(oldConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	// Create temp files
	tmpDir := t.TempDir()
	oldFile := filepath.Join(tmpDir, "old_config.json")
	newFile := filepath.Join(tmpDir, "new_config.json")

	if err := os.WriteFile(oldFile, oldData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test migration
	if err := MigrateOpenCodeConfigFile(oldFile, newFile); err != nil {
		t.Fatalf("File migration failed: %v", err)
	}

	// Verify new file exists and has correct content
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Fatal("Migrated file was not created")
	}

	newData, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("Failed to read migrated file: %v", err)
	}

	var migratedConfig map[string]interface{}
	if err := json.Unmarshal(newData, &migratedConfig); err != nil {
		t.Fatalf("Failed to parse migrated config: %v", err)
	}

	// Verify schema was updated
	if schema, ok := migratedConfig["$schema"].(string); !ok || schema != "./opencode-schema.json" {
		t.Errorf("Migrated file has wrong schema: %v", schema)
	}

	t.Log("✅ File migration test passed")
}

// TestFindBestModelForAgent tests the model selection logic
func TestFindBestModelForAgent(t *testing.T) {
	providers := map[string]interface{}{
		"openai":    map[string]interface{}{},
		"anthropic": map[string]interface{}{},
	}

	// Test coder model selection
	coderModel := findBestModelForAgent(providers, []string{"gpt-4", "claude-3"})
	if coderModel == "" {
		t.Error("Should have found a coder model")
	}
	if !strings.Contains(coderModel, "gpt-4") && !strings.Contains(coderModel, "claude-3") {
		t.Errorf("Coder model should be gpt-4 or claude-3, got: %s", coderModel)
	}

	// Test fallback
	emptyProviders := map[string]interface{}{}
	fallbackModel := findBestModelForAgent(emptyProviders, []string{"nonexistent"})
	if fallbackModel != "" {
		t.Error("Should return empty string when no providers available")
	}

	t.Log("✅ Model selection test passed")
}
