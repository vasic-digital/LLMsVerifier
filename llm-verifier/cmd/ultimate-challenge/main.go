package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"llm-verifier/logging"
	"llm-verifier/providers"
	"llm-verifier/scoring"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  ULTIMATE OPENCODE CHALLENGE - MODEL DISCOVERY              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create logger
	logger, err := logging.NewLogger(nil, nil)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// Create provider service
	service := providers.NewModelProviderService("/tmp/opencode.json", logger)

	// Register all providers from env
	fmt.Println("📋 Registering all providers from environment...")
	service.RegisterAllProviders()
	allProviders := service.GetAllProviders()
	fmt.Printf("✓ Registered %d providers\n", len(allProviders))
	fmt.Println()

	// Discover models for all providers
	fmt.Println("🔍 Discovering models from all providers...")
	allModels := make(map[string][]providers.Model)
	totalModels := 0

	for providerID := range allProviders {
		
		fmt.Printf("  Testing %s... ", providerID)
		models, err := service.GetModels(providerID)
		if err != nil {
			fmt.Printf("⚠️  Error: %v\n", err)
		} else {
			fmt.Printf("✓ Found %d models\n", len(models))
			allModels[providerID] = models
			totalModels += len(models)
		}
	}

	fmt.Println()
	fmt.Printf("✅ Total: %d providers, %d models discovered\n", len(allModels), totalModels)
	fmt.Println()

	// Generate ultimate OpenCode configuration
	fmt.Println("📄 Generating ultimate OpenCode configuration...")
	config := generateUltimateOpenCode(allModels, service, allProviders)

	// Write to file
	outputPath := "/home/milosvasic/Downloads/opencode.json"
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		log.Fatalf("Failed to encode config: %v", err)
	}

	fmt.Printf("✅ Ultimate OpenCode configuration exported to: %s\n", outputPath)
	fmt.Printf("📊 Size: %.2f KB\n", float64(getFileSize(outputPath))/1024)
	
	// Copy to config directory
	configDir := "/home/milosvasic/.config/opencode"
	if err := os.MkdirAll(configDir, 0755); err == nil {
		configPath := configDir + "/opencode.json"
		os.Remove(configPath) // Remove old symlink
		if err := copyFile(outputPath, configPath); err == nil {
			fmt.Printf("✅ Configuration copied to: %s\n", configPath)
		}
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  🎉 ULTIMATE CHALLENGE COMPLETE - CONFIGURATION READY       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}

func generateUltimateOpenCode(allModels map[string][]providers.Model, service *providers.ModelProviderService, allProviders map[string]*providers.ProviderClient) map[string]interface{} {
	config := make(map[string]interface{})
	
	// Create display formatter for model names
	displayFormatter := scoring.NewModelDisplayName()
	
	// Build provider section
	providerConfig := make(map[string]interface{})
	
	for providerID, models := range allModels {
		if len(models) == 0 {
			continue // Skip providers with no models
		}
		
		// Get provider client for API key and base URL
		providerClient, exists := allProviders[providerID]
		if !exists || providerClient.APIKey == "" {
			continue // Skip if no provider client or API key
		}
		
		// Create provider entry with options and models
		providerEntry := make(map[string]interface{})
		
		// Add options (API key and base URL)
		providerEntry["options"] = map[string]interface{}{
			"api_key":  providerClient.APIKey,
			"base_url": providerClient.BaseURL,
		}
		
		// Create model map for this provider
		modelMap := make(map[string]interface{})
		
		for _, model := range models {
			// Extract features for display name
			featureData := map[string]interface{}{
				"supports_brotli": model.SupportsBrotli,
				"supports_http3":  model.SupportsHTTP3,
				"supports_toon":   model.SupportsToon,
				"is_free":         model.IsFree,
				"is_open_source":  model.IsOpenSource,
			}
			
			// Generate display name with suffixes
			displayName := displayFormatter.FormatWithFeatureSuffixes(model.Name, featureData)
			
			modelEntry := map[string]interface{}{
				"id":             model.ID,
				"name":           model.Name,
				"display_name":   displayName,
				"provider":       model.ProviderID,
				"max_tokens":     model.MaxTokens,
				"supports_http3": model.SupportsHTTP3,
			}
			
			// Add cost if available
			if model.CostPer1MInput > 0 || model.CostPer1MOutput > 0 {
				modelEntry["cost_per_1m_input"] = model.CostPer1MInput
				modelEntry["cost_per_1m_output"] = model.CostPer1MOutput
			}
			
			// Add features
			features := make(map[string]interface{})
			if model.SupportsHTTP3 {
				features["http3"] = true
			}
			if model.IsFree {
				features["free_to_use"] = true
			}
			if model.IsOpenSource {
				features["open_source"] = true
			}
			if model.SupportsBrotli {
				features["brotli"] = true
			}
			if model.SupportsToon {
				features["toon"] = true
			}
			
			if len(features) > 0 {
				modelEntry["features"] = features
			}
			
			modelMap[model.ID] = modelEntry
		}
		
		providerEntry["models"] = modelMap
		providerConfig[providerID] = providerEntry
	}
	
	config["provider"] = providerConfig
	
	// Add agent
	config["agent"] = map[string]interface{}{
		"verifier": map[string]interface{}{
			"model": "openai/gpt-4",
			"prompt": "You are an LLM verifier agent",
		},
	}
	
	// Add mcp
	config["mcp"] = map[string]interface{}{
		"filesystem": map[string]interface{}{
			"type": "local",
			"command": []string{"npx", "@modelcontextprotocol/server-filesystem"},
			"enabled": true,
		},
	}
	
	return config
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = destination.ReadFrom(source)
	return err
}
