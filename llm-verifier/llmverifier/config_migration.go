package llmverifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// migrateOpenCodeConfig converts old LLM Verifier OpenCode configs to the new OpenCode-compatible format
func migrateOpenCodeConfig(oldConfig map[string]interface{}) (map[string]interface{}, error) {
	newConfig := make(map[string]interface{})

	// Convert schema
	if schema, ok := oldConfig["$schema"].(string); ok {
		if schema == "https://opencode.ai/config.json" {
			newConfig["$schema"] = "./opencode-schema.json"
		} else {
			newConfig["$schema"] = schema
		}
	} else {
		newConfig["$schema"] = "./opencode-schema.json"
	}

	// Add required data section
	newConfig["data"] = map[string]interface{}{
		"directory": ".opencode",
	}

	// Convert provider section to providers section
	if oldProviderSection, ok := oldConfig["provider"].(map[string]interface{}); ok {
		providersSection := make(map[string]interface{})

		for providerName, providerData := range oldProviderSection {
			if providerMap, ok := providerData.(map[string]interface{}); ok {
				// Extract API key from old options structure
				var apiKey string
				if options, hasOptions := providerMap["options"].(map[string]interface{}); hasOptions {
					if key, hasKey := options["apiKey"].(string); hasKey {
						apiKey = key
					}
				}

				// Create new provider format
				newProvider := map[string]interface{}{
					"apiKey":   apiKey,
					"disabled": false,
					"provider": providerName,
				}
				providersSection[providerName] = newProvider
			}
		}

		newConfig["providers"] = providersSection
	}

	// Add agents section with smart model assignment
	agentsSection := make(map[string]interface{})
	if providers, ok := newConfig["providers"].(map[string]interface{}); ok {
		// Find best models for agents
		bestCoderModel := findBestModelForAgent(providers, []string{"gpt-4", "claude-3", "gpt-4o", "claude-3-5-sonnet"})
		bestTaskModel := findBestModelForAgent(providers, []string{"gpt-4", "claude-3", "gpt-3.5-turbo", "claude-3-haiku"})
		bestTitleModel := findBestModelForAgent(providers, []string{"gpt-3.5-turbo", "claude-3-haiku", "gpt-4o-mini"})

		if bestCoderModel != "" {
			agentsSection["coder"] = map[string]interface{}{
				"model":     bestCoderModel,
				"maxTokens": 5000,
			}
		}
		if bestTaskModel != "" {
			agentsSection["task"] = map[string]interface{}{
				"model":     bestTaskModel,
				"maxTokens": 5000,
			}
		}
		if bestTitleModel != "" {
			agentsSection["title"] = map[string]interface{}{
				"model":     bestTitleModel,
				"maxTokens": 80,
			}
		}
	}

	newConfig["agents"] = agentsSection

	// Add other required sections
	newConfig["tui"] = map[string]interface{}{
		"theme": "opencode",
	}
	newConfig["shell"] = map[string]interface{}{
		"path": "/bin/bash",
		"args": []string{"-l"},
	}
	newConfig["autoCompact"] = true
	newConfig["debug"] = false
	newConfig["debugLSP"] = false

	return newConfig, nil
}

// findBestModelForAgent finds the best available model from providers for a specific agent role
func findBestModelForAgent(providers map[string]interface{}, preferredModels []string) string {
	for _, preferredModel := range preferredModels {
		for providerName := range providers {
			// Check if this provider has the preferred model
			modelRef := fmt.Sprintf("%s.%s", providerName, preferredModel)
			// For migration, we'll assume the model exists if the provider does
			// In a real implementation, you might want to validate against actual model lists
			return modelRef
		}
	}
	// Fallback to first available provider with a generic model
	for providerName := range providers {
		return fmt.Sprintf("%s.gpt-4o", providerName)
	}
	return ""
}

// MigrateOpenCodeConfigFile migrates a single OpenCode config file
func MigrateOpenCodeConfigFile(inputPath, outputPath string) error {
	// Read old config
	oldData, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var oldConfig map[string]interface{}
	if err := json.Unmarshal(oldData, &oldConfig); err != nil {
		return fmt.Errorf("failed to parse input JSON: %w", err)
	}

	// Migrate config
	newConfig, err := migrateOpenCodeConfig(oldConfig)
	if err != nil {
		return fmt.Errorf("failed to migrate config: %w", err)
	}

	// Write new config
	newData, err := json.MarshalIndent(newConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal new config: %w", err)
	}

	if err := os.WriteFile(outputPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// MigrateOpenCodeConfigsInDirectory migrates all OpenCode config files in a directory
func MigrateOpenCodeConfigsInDirectory(inputDir, outputDir string) error {
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read input directory: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	migratedCount := 0
	for _, file := range files {
		if strings.Contains(file.Name(), "opencode") && strings.HasSuffix(file.Name(), ".json") {
			inputPath := filepath.Join(inputDir, file.Name())
			outputPath := filepath.Join(outputDir, "migrated_"+file.Name())

			if err := MigrateOpenCodeConfigFile(inputPath, outputPath); err != nil {
				fmt.Printf("Warning: Failed to migrate %s: %v\n", file.Name(), err)
				continue
			}

			migratedCount++
			fmt.Printf("✅ Migrated %s → %s\n", file.Name(), "migrated_"+file.Name())
		}
	}

	fmt.Printf("Migration complete: %d files migrated\n", migratedCount)
	return nil
}
