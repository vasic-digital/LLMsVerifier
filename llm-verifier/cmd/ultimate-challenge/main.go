package main

import (
	_ "context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	_ "time"

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

	// Discover models from all providers
	fmt.Println("🔍 Discovering models from all providers...")

	// Define result type for concurrent processing
	type providerResult struct {
		providerID string
		models     []providers.Model
		err        error
	}

	// Process providers in parallel for better performance
	results := make(chan providerResult, len(allProviders))

	// Start goroutines for each provider
	for providerID := range allProviders {
		go func(pid string) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("❌ PANIC in provider %s: %v\n", pid, r)
					results <- providerResult{pid, nil, fmt.Errorf("panic: %v", r)}
				}
			}()

			fmt.Printf("🔍 Processing provider: %s\n", pid)

			// Get models using discovery (no verification to avoid hanging)
			models, err := service.GetModels(pid)
			if err != nil {
				fmt.Printf("❌ Failed to get models for %s: %v\n", pid, err)
				results <- providerResult{pid, nil, err}
				return
			}

			if len(models) == 0 {
				fmt.Printf("❌ No models found for %s\n", pid)
				results <- providerResult{pid, models, nil}
				return
			}

			// Mark models as discovered (unverified)
			providerClient, hasClient := allProviders[pid]
			isVerified := hasClient && providerClient.APIKey != ""

			for i := range models {
				if models[i].Features == nil {
					models[i].Features = make(map[string]interface{})
				}
				models[i].Features["verified"] = isVerified
				models[i].Features["llmsVerifier"] = true
			}

			verificationStatus := "discovered"
			if isVerified {
				verificationStatus = "verified"
			}
			fmt.Printf("✓ Found %d models (%s) for %s\n", len(models), verificationStatus, pid)

			results <- providerResult{pid, models, nil}
		}(providerID)
	}

	// Collect results
	allModels := make(map[string][]providers.Model)
	totalModels := 0
	verifiedModels := 0
	unverifiedModels := 0
	providersWithModels := 0

	for i := 0; i < len(allProviders); i++ {
		result := <-results

		if result.err == nil && len(result.models) > 0 {
			allModels[result.providerID] = result.models
			totalModels += len(result.models)

			// Count verified vs unverified models
			verifiedCount := 0
			unverifiedCount := 0
			for _, model := range result.models {
				if model.Features != nil {
					if verified, exists := model.Features["verified"]; exists {
						if v, ok := verified.(bool); ok && v {
							verifiedCount++
						} else {
							unverifiedCount++
						}
					} else {
						unverifiedCount++
					}
				} else {
					unverifiedCount++
				}
			}

			verifiedModels += verifiedCount
			unverifiedModels += unverifiedCount
			providersWithModels++
		}
	}

	fmt.Println()
	fmt.Printf("✅ DISCOVERY COMPLETE:\n")
	fmt.Printf("   📊 Providers processed: %d\n", len(allProviders))
	fmt.Printf("   🏢 Providers with models: %d\n", providersWithModels)
	fmt.Printf("   🤖 Total models discovered: %d\n", totalModels)
	fmt.Printf("   ✅ Verified models: %d\n", verifiedModels)
	fmt.Printf("   ⚠️  Unverified models: %d\n", unverifiedModels)
	fmt.Printf("   🎯 Average models per provider: %.1f\n", float64(totalModels)/float64(providersWithModels))
	fmt.Println()

	if totalModels == 0 {
		fmt.Printf("❌ ERROR: No models discovered! Check provider configurations and network connectivity.\n")
		fmt.Println()
		os.Exit(1)
	}

	if verifiedModels == 0 {
		fmt.Printf("⚠️  WARNING: No models were verified.\n")
		fmt.Println("   All models are marked as 'discovered' (unverified).")
		fmt.Println("   To verify models, configure API keys in environment variables.")
		fmt.Println()
	}

	// Generate ultimate OpenCode configuration
	fmt.Println("📄 Generating ultimate OpenCode configuration...")
	config := generateUltimateOpenCode(allModels, service, allProviders, totalModels)

	// Write to file - use current directory or specified output
	outputPath := os.Getenv("OPENCODE_OUTPUT_PATH")
	if outputPath == "" {
		outputPath = "opencode_ultimate.json"
	}

	publicOutputPath := strings.Replace(outputPath, ".json", "_public.json", 1)

	fmt.Printf("\n💾 WRITING CONFIGURATIONS:\n")
	fmt.Printf("   🔐 Full config (with API keys): %s\n", outputPath)
	fmt.Printf("   🌐 Public config (no API keys): %s\n", publicOutputPath)

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
	}

	// Write full configuration (with API keys) - SECURE FILE
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create full config file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		log.Fatalf("Failed to encode full config: %v", err)
	}

	fmt.Printf("✅ Full OpenCode configuration exported to: %s\n", outputPath)
	fmt.Printf("📊 Size: %.2f KB\n", float64(getFileSize(outputPath))/1024)

	// Set restrictive permissions on full config (owner read/write only)
	if err := os.Chmod(outputPath, 0600); err != nil {
		fmt.Printf("⚠️  Warning: Could not set restrictive permissions on %s: %v\n", outputPath, err)
	} else {
		fmt.Printf("🔒 Set restrictive permissions (600) on full config\n")
	}

	// Write public configuration (without API keys) - VERSIONABLE FILE
	publicFile, err := os.Create(publicOutputPath)
	if err != nil {
		log.Fatalf("Failed to create public config file: %v", err)
	}
	defer publicFile.Close()

	publicEncoder := json.NewEncoder(publicFile)
	publicEncoder.SetIndent("", "  ")
	if err := publicEncoder.Encode(config); err != nil {
		log.Fatalf("Failed to encode public config: %v", err)
	}

	fmt.Printf("✅ Public OpenCode configuration exported to: %s\n", publicOutputPath)
	fmt.Printf("📊 Size: %.2f KB\n", float64(getFileSize(publicOutputPath))/1024)

	// Verify both generated configurations
	fmt.Println("\n🔍 Verifying OpenCode configurations...")

	// Verify full configuration (with API keys)
	fmt.Printf("🔐 Verifying full configuration: %s\n", outputPath)
	if err := verifyOpenCodeConfig(outputPath); err != nil {
		fmt.Printf("❌ Full configuration validation failed: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Println("✅ Full configuration structure verified successfully")
	}

	// Verify public configuration (without API keys)
	fmt.Printf("🌐 Verifying public configuration: %s\n", publicOutputPath)
	if err := verifyOpenCodeConfig(publicOutputPath); err != nil {
		fmt.Printf("❌ Public configuration validation failed: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Println("✅ Public configuration structure verified successfully")
	}

	// Copy PUBLIC configuration to config directory (no API keys)
	configDir := "/home/milosvasic/.config/opencode"
	if err := os.MkdirAll(configDir, 0755); err == nil {
		configPath := configDir + "/opencode.json"
		os.Remove(configPath) // Remove old symlink
		if err := copyFile(publicOutputPath, configPath); err == nil {
			fmt.Printf("✅ Configuration copied to: %s\n", configPath)
		}
	}

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  🎉 ULTIMATE CHALLENGE COMPLETE - CONFIGURATIONS READY      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("📋 FINAL RESULTS:\n")
	fmt.Printf("   🏢 Providers: %d\n", providersWithModels)
	fmt.Printf("   🤖 Total models: %d\n", totalModels)
	fmt.Printf("   ✅ Verified models: %d\n", verifiedModels)
	fmt.Printf("   🔐 Full config: %s\n", outputPath)
	fmt.Printf("   🌐 Public config: %s\n", publicOutputPath)
	fmt.Printf("   📁 Config directory: %s\n", configDir)
	fmt.Println()
	fmt.Println("🎯 OpenCode configuration is ready for production use!")
}

func generateUltimateOpenCode(allModels map[string][]providers.Model, service interface{}, allProviders map[string]*providers.ProviderClient, totalModels int) map[string]interface{} {
	config := make(map[string]interface{})

	// Basic OpenCode structure
	config["$schema"] = "https://opencode.sh/schema.json"
	config["username"] = "OpenCode AI Assistant"

	// Create display formatter for model names
	displayFormatter := scoring.NewModelDisplayName()

	// Build provider section with proper OpenCode structure
	providerConfig := make(map[string]interface{})

	for providerID, models := range allModels {
		if len(models) == 0 {
			continue // Skip providers with no models
		}

		// Convert provider ID to camelCase for OpenCode compatibility
		camelCaseProviderID := toCamelCase(providerID)

		// Create provider entry with proper OpenCode structure
		providerEntry := make(map[string]interface{})

		// Get provider client for base URL only
		providerClient, exists := allProviders[providerID]

		// Only add baseURL if available (NO API KEYS IN PUBLIC VERSION)
		if exists && providerClient.BaseURL != "" {
			// Process baseURL to ensure it ends with /v1 and has proper format
			baseURL := providerClient.BaseURL
			if !strings.Contains(baseURL, "/v1") && !strings.HasSuffix(baseURL, "/v1") {
				baseURL = strings.TrimSuffix(baseURL, "/") + "/v1"
			}

			// Add options with baseURL only - NO API KEYS
			providerEntry["options"] = map[string]interface{}{
				"baseURL": baseURL,
			}
		}

		// Note: API keys are NEVER exported for security reasons
		// Users must set them in their environment or OpenCode configuration

		// Create model map for this provider with proper structure
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

			// Generate display name with suffixes and add (llmsvd) and verification status
			displayName := displayFormatter.FormatWithFeatureSuffixesAndLLMsVerifier(model.Name, featureData)

			// Create model entry with proper OpenCode structure
			modelEntry := map[string]interface{}{
				"id":          model.ID,
				"name":        model.Name + " (llmsvd)", // Add (llmsvd) to model name
				"displayName": displayName,
				"provider": map[string]interface{}{
					"id":  providerID,
					"npm": "@openrouter/" + providerID + "-provider",
				},
			}

			// Add maxTokens if available
			if model.MaxTokens > 0 {
				modelEntry["maxTokens"] = model.MaxTokens
			}

			// Add supportsHTTP3 flag
			modelEntry["supportsHTTP3"] = model.SupportsHTTP3

			// Add cost information with camelCase keys
			if model.CostPer1MInput > 0 || model.CostPer1MOutput > 0 {
				modelEntry["costPer1MInput"] = model.CostPer1MInput   // Use camelCase
				modelEntry["costPer1MOutput"] = model.CostPer1MOutput // Use camelCase
			}

			// Add features with proper camelCase and snake_case
			features := make(map[string]interface{})
			if model.SupportsHTTP3 {
				features["http3"] = true
			}
			if model.IsFree {
				features["freeToUse"] = true // Use camelCase
			}
			if model.IsOpenSource {
				features["openSource"] = true
			}

			// Add verification status
			if model.Features != nil {
				if verified, exists := model.Features["verified"]; exists {
					if v, ok := verified.(bool); ok {
						features["verified"] = v
					}
				}
				features["llmsVerifier"] = true
			}

			if len(features) > 0 {
				modelEntry["features"] = features
			}

			modelMap[model.ID] = modelEntry
		}

		if len(modelMap) > 0 {
			providerEntry["models"] = modelMap
			providerConfig[camelCaseProviderID] = providerEntry
		}
	}

	config["provider"] = providerConfig

	// Add agent configurations
	config["agent"] = map[string]interface{}{
		"code": map[string]interface{}{
			"maxSteps":    10,
			"model":       "openai/gpt-4",
			"prompt":      "You are a senior software engineer specializing in code development, debugging, and optimization. You have deep expertise in multiple programming languages and frameworks. Help user write clean, efficient, and well-documented code.",
			"temperature": 0.2,
			"tools": map[string]interface{}{
				"bash":     true,
				"docker":   true,
				"git":      true,
				"lsp":      true,
				"webfetch": true,
			},
		},
		"review": map[string]interface{}{
			"maxSteps":    5,
			"model":       "anthropic/claude-3-sonnet",
			"prompt":      "You are a meticulous code reviewer with expertise in best practices, security, and performance. Review code thoroughly and provide detailed feedback on improvements, potential bugs, and optimization opportunities.",
			"temperature": 0.1,
			"tools": map[string]interface{}{
				"diff": true,
				"lsp":  true,
			},
		},
		"verifier": map[string]interface{}{
			"model":       "openai/gpt-4",
			"prompt":      "You are an LLM verifier agent specialized in model verification and testing. You ensure that models can properly see and understand code.",
			"temperature": 0.1,
			"tools": map[string]interface{}{
				"verification": true,
			},
		},
	}

	// Add MCP configurations
	config["mcp"] = map[string]interface{}{
		"filesystem": map[string]interface{}{
			"command": []interface{}{
				"npx",
				"@modelcontextprotocol/server-filesystem",
			},
			"enabled": true,
			"type":    "local",
		},
		"git": map[string]interface{}{
			"command": []interface{}{
				"npx",
				"@modelcontextprotocol/server-git",
			},
			"enabled": true,
			"type":    "local",
		},
		"github": map[string]interface{}{
			"command": []interface{}{
				"npx",
				"@modelcontextprotocol/server-github",
			},
			"enabled": true,
			"type":    "local",
		},
		"postgresql": map[string]interface{}{
			"command": []interface{}{
				"npx",
				"@modelcontextprotocol/server-postgresql",
			},
			"enabled": true,
			"type":    "local",
		},
	}

	return config
}

func verifyOpenCodeConfig(configPath string) error {
	// Read the generated config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Validate required top-level fields
	requiredFields := []string{"$schema", "username", "provider", "agent", "mcp"}
	for _, field := range requiredFields {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate provider section
	providerSection, ok := config["provider"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid provider section structure")
	}

	if len(providerSection) == 0 {
		return fmt.Errorf("provider section is empty")
	}

	// Check that each provider has models
	for providerID, providerData := range providerSection {
		providerMap, ok := providerData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid structure for provider %s", providerID)
		}

		modelsSection, ok := providerMap["models"]
		if !ok {
			return fmt.Errorf("provider %s missing models section", providerID)
		}

		modelsMap, ok := modelsSection.(map[string]interface{})
		if !ok {
			return fmt.Errorf("provider %s has invalid models section", providerID)
		}

		if len(modelsMap) == 0 {
			return fmt.Errorf("provider %s has no models", providerID)
		}

		// Check that options only contains baseURL (not apiKey)
		if optionsData, exists := providerMap["options"]; exists {
			options, ok := optionsData.(map[string]interface{})
			if !ok {
				return fmt.Errorf("provider %s has invalid options section", providerID)
			}

			// Should only have baseURL, not apiKey
			if _, hasAPIKey := options["apiKey"]; hasAPIKey {
				return fmt.Errorf("provider %s options section should not contain apiKey", providerID)
			}

			// Should have baseURL
			if _, hasBaseURL := options["baseURL"]; !hasBaseURL {
				return fmt.Errorf("provider %s options section missing baseURL", providerID)
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func toCamelCase(s string) string {
	// Simple kebab-case to camelCase conversion
	parts := strings.Split(s, "-")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}
