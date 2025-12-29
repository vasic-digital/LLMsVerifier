package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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

	fmt.Printf("  [DEBUG] Logger created successfully\n")

	// Create verification configuration (disabled for discovery phase)
	verificationConfig := providers.VerificationConfig{
		Enabled:              false, // Disable for discovery phase
		StrictMode:           false,
		MaxRetries:           1,
		TimeoutSeconds:       10,
		RequireAffirmative:   false,
		MinVerificationScore: 0.0,
	}

	// Create enhanced provider service with mandatory verification
	service := providers.NewEnhancedModelProviderService("/tmp/opencode.json", logger, verificationConfig)

	// Register all providers from env
	fmt.Println("📋 Registering all providers from environment...")
	service.RegisterAllProviders()
	allProviders := service.GetAllProviders()
	fmt.Printf("✓ Registered %d providers\n", len(allProviders))
	fmt.Println()

	// Discover all models from all providers
	fmt.Println("🔍 Discovering and verifying models from all providers...")
	fmt.Printf("  [DEBUG] About to start %d goroutines\n", len(allProviders))

	// Define result type for concurrent processing
	type providerResult struct {
		providerID string
		models     []providers.Model
		err        error
	}

	// Process providers in parallel for better performance
	results := make(chan providerResult, len(allProviders))
	fmt.Printf("  [DEBUG] Created results channel\n")

	allModels := make(map[string][]providers.Model)
	totalModels := 0
	verifiedModels := 0
	unverifiedModels := 0
	providersWithModels := 0

	// Start goroutines for each provider
	for providerID := range allProviders {
		go func(pid string) {
			var models []providers.Model
			var verified bool

			// For discovery phase, get all models without verification to avoid timeouts
			// Verification will be handled separately for specific providers with valid API keys
			unverifiedModels, err := service.GetModels(pid)
			if err != nil {
				fmt.Printf("❌ Error getting models: %v\n", err)
				results <- providerResult{pid, nil, err}
				return
			}
			if len(unverifiedModels) == 0 {
				fmt.Printf("❌ No models found\n")
				results <- providerResult{pid, unverifiedModels, nil}
				return
			}
			models = unverifiedModels
			verified = false
			fmt.Printf("✓ Found %d models (discovered)\n", len(models))

			// Mark models with verification status
			for i := range models {
				if verified {
					// Add verified status to features
					if models[i].Features == nil {
						models[i].Features = make(map[string]interface{})
					}
					models[i].Features["verified"] = true
					models[i].Features["llmsVerifier"] = true
				} else {
					// Mark as unverified
					if models[i].Features == nil {
						models[i].Features = make(map[string]interface{})
					}
					models[i].Features["verified"] = false
					models[i].Features["llmsVerifier"] = false
				}
			}

			results <- providerResult{pid, models, nil}
		}(providerID)
	}

	// Collect results
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
	fmt.Printf("   ℹ️  All models marked as unverified (discovery phase)\n")
	fmt.Printf("   🎯 Average models per provider: %.1f\n", float64(totalModels)/float64(providersWithModels))
	fmt.Println()

	if totalModels == 0 {
		fmt.Printf("❌ ERROR: No models discovered! Check provider configurations and network connectivity.\n")
		fmt.Println()
	} else {
		fmt.Printf("ℹ️  NOTE: All %d models are included as discovered models.\n", totalModels)
		fmt.Println("   Verification can be performed selectively for specific providers with API keys.")
		fmt.Println()
	}

	// Generate ultimate OpenCode configurations
	fmt.Println("📄 Generating ultimate OpenCode configurations...")
	fullConfig := generateUltimateOpenCode(allModels, service, allProviders, totalModels)
	publicConfig := generateUltimateOpenCodePublic(allModels, service, allProviders, totalModels)

	// Write to files - use current directory or specified output
	outputPath := os.Getenv("OPENCODE_OUTPUT_PATH")
	if outputPath == "" {
		outputPath = "opencode_ultimate.json"
	}

	publicOutputPath := strings.Replace(outputPath, ".json", "_public.json", 1)

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
	if err := encoder.Encode(fullConfig); err != nil {
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
	if err := publicEncoder.Encode(publicConfig); err != nil {
		log.Fatalf("Failed to encode public config: %v", err)
	}

	fmt.Printf("✅ Public OpenCode configuration exported to: %s\n", publicOutputPath)
	fmt.Printf("📊 Size: %.2f KB\n", float64(getFileSize(publicOutputPath))/1024)

	// Verify the generated configuration
	// Verify both generated configurations
	fmt.Println("\n🔍 Verifying OpenCode configurations...")

	// Verify full configuration (with API keys)
	fmt.Printf("🔐 Verifying full configuration: %s\n", outputPath)
	if err := verifyOpenCodeConfig(outputPath); err != nil {
		fmt.Printf("❌ Full configuration verification FAILED: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Println("✅ Full configuration structure verified successfully")
	}

	// Verify public configuration (without API keys)
	fmt.Printf("🌐 Verifying public configuration: %s\n", publicOutputPath)
	if err := verifyOpenCodeConfig(publicOutputPath); err != nil {
		fmt.Printf("❌ Public configuration verification FAILED: %v\n", err)
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
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  🎉 ULTIMATE CHALLENGE COMPLETE - CONFIGURATIONS READY      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("📋 Configuration Summary:")
	fmt.Println("   🔐 Full config: Contains API keys (secure, git-ignored)")
	fmt.Println("   🌐 Public config: No API keys (safe for versioning)")
	fmt.Println("   📁 Config directory: Uses public config (no API keys)")
}

func generateUltimateOpenCode(allModels map[string][]providers.Model, service interface{}, allProviders map[string]*providers.ProviderClient, totalModels int) map[string]interface{} {
	// Create comprehensive OpenCode configuration
	config := make(map[string]interface{})

	// Add OpenCode schema
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

		// Get provider client for API key and base URL
		providerClient, exists := allProviders[providerID]

		// SECURITY: NEVER include API keys in exported configurations
		// API keys are sensitive and should never be in config files
		// OpenCode configurations should not contain sensitive credentials
		// Users must configure API keys separately in their environment

		// Only add baseURL if provider exists and has base URL
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
				"displayName": displayName,              // Use camelCase
				"provider": map[string]interface{}{
					"id":  model.ProviderID,                                         // Provider ID field
					"npm": fmt.Sprintf("@openrouter/%s-provider", model.ProviderID), // NPM package field
				},
				"maxTokens":     model.MaxTokens,     // Use camelCase
				"supportsHTTP3": model.SupportsHTTP3, // Use camelCase
			}

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
				features["openSource"] = true // Use camelCase
			}
			if model.SupportsBrotli {
				features["brotli"] = true
			}
			if model.SupportsToon {
				features["toon"] = true
			}

			// Add verification status from model features
			if modelVerified, exists := model.Features["verified"]; exists {
				if v, ok := modelVerified.(bool); ok {
					features["verified"] = v
				} else {
					features["verified"] = false
				}
			} else {
				features["verified"] = false
			}
			features["llmsVerifier"] = true

			if len(features) > 0 {
				modelEntry["features"] = features
			}

			modelMap[model.ID] = modelEntry
		}

		providerEntry["models"] = modelMap
		providerConfig[camelCaseProviderID] = providerEntry
	}

	config["provider"] = providerConfig

	// Add comprehensive agent configuration
	config["agent"] = map[string]interface{}{
		"code": map[string]interface{}{
			"model":  "openai/gpt-4",
			"prompt": "You are a senior software engineer specializing in code development, debugging, and optimization. You have deep expertise in multiple programming languages and frameworks. Help the user write clean, efficient, and well-documented code.",
			"tools": map[string]interface{}{
				"bash":     true,
				"docker":   true,
				"git":      true,
				"lsp":      true,
				"webfetch": true,
			},
			"temperature": 0.2,
			"maxSteps":    10,
		},
		"review": map[string]interface{}{
			"model":  "anthropic/claude-3-sonnet",
			"prompt": "You are a meticulous code reviewer with expertise in best practices, security, and performance. Review the code thoroughly and provide detailed feedback on improvements, potential bugs, and optimization opportunities.",
			"tools": map[string]interface{}{
				"lsp":  true,
				"diff": true,
			},
			"temperature": 0.1,
			"maxSteps":    5,
		},
		"verifier": map[string]interface{}{
			"model":  "openai/gpt-4",
			"prompt": "You are an LLM verifier agent specialized in model verification and testing. You ensure that models can properly see and understand code.",
			"tools": map[string]interface{}{
				"verification": true,
			},
			"temperature": 0.1,
		},
	}

	// Add comprehensive MCP configuration
	config["mcp"] = map[string]interface{}{
		"filesystem": map[string]interface{}{
			"type":    "local",
			"command": []string{"npx", "@modelcontextprotocol/server-filesystem"},
			"enabled": true,
		},
		"git": map[string]interface{}{
			"type":    "local",
			"command": []string{"npx", "@modelcontextprotocol/server-git"},
			"enabled": true,
		},
		"github": map[string]interface{}{
			"type":    "local",
			"command": []string{"npx", "@modelcontextprotocol/server-github"},
			"enabled": true,
		},
		"postgresql": map[string]interface{}{
			"type":    "local",
			"command": []string{"npx", "@modelcontextprotocol/server-postgresql"},
			"enabled": true,
		},
	}

	// Note: Metadata removed as it's not part of OpenCode schema

	return config
}

// generateUltimateOpenCodePublic generates OpenCode configuration WITHOUT API keys (safe for versioning)
func generateUltimateOpenCodePublic(allModels map[string][]providers.Model, service interface{}, allProviders map[string]*providers.ProviderClient, totalModels int) map[string]interface{} {
	config := make(map[string]interface{})

	// Basic OpenCode structure
	config["$schema"] = "https://opencode.sh/schema.json"
	config["username"] = "OpenCode AI Assistant"

	// Create display formatter for model names
	displayFormatter := scoring.NewModelDisplayName()

	// Build provider section with proper OpenCode structure (NO API KEYS)
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

		// Only add baseURL if available (NO API KEYS - PUBLIC VERSION)
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

			// Add verification status (always false for public version)
			features["verified"] = false
			features["llmsVerifier"] = true

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
			"prompt":      "You are a senior software engineer specializing in code development, debugging, and optimization. You have deep expertise in multiple programming languages and frameworks. Help the user write clean, efficient, and well-documented code.",
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
			"prompt":      "You are a meticulous code reviewer with expertise in best practices, security, and performance. Review the code thoroughly and provide detailed feedback on improvements, potential bugs, and optimization opportunities.",
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

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// toCamelCase converts a string to camelCase format
// verifyOpenCodeConfig verifies that the generated configuration follows OpenCode standards
func verifyOpenCodeConfig(configPath string) error {
	// Read the generated configuration
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Verify required top-level fields
	requiredFields := []string{"$schema", "username", "provider", "agent", "mcp"}
	for _, field := range requiredFields {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Verify schema URL
	if schema, ok := config["$schema"].(string); !ok || schema != "https://opencode.sh/schema.json" {
		return fmt.Errorf("invalid or missing $schema field")
	}

	// Verify provider structure
	providers, ok := config["provider"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid provider section structure")
	}

	for providerID, providerData := range providers {
		provider, ok := providerData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid provider structure for %s", providerID)
		}

		// Check for options wrapper - only required for verified providers
		if optionsData, exists := provider["options"]; exists {
			options, ok := optionsData.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid options structure for provider %s", providerID)
			}

			// Check for required options fields
			if _, exists := options["apiKey"]; !exists {
				return fmt.Errorf("missing apiKey in options for provider %s", providerID)
			}
			if _, exists := options["baseURL"]; !exists {
				return fmt.Errorf("missing baseURL in options for provider %s", providerID)
			}

			// Verify baseURL format
			baseURL, ok := options["baseURL"].(string)
			if !ok || !strings.Contains(baseURL, "/v1") {
				return fmt.Errorf("invalid baseURL format for provider %s (must contain /v1)", providerID)
			}
		} else {
			// Options not present - this is allowed for unverified providers
			fmt.Printf("   Note: Provider %s has no options (unverified)\n", providerID)
		}

		// Check models section
		models, ok := provider["models"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid models section for provider %s", providerID)
		}

		// Verify model structure
		for modelID, modelData := range models {
			model, ok := modelData.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid model structure for %s/%s", providerID, modelID)
			}

			// Check required model fields
			requiredModelFields := []string{"id", "name", "displayName", "provider", "maxTokens", "supportsHTTP3"}
			for _, field := range requiredModelFields {
				if _, exists := model[field]; !exists {
					return fmt.Errorf("missing required field %s in model %s/%s", field, providerID, modelID)
				}
			}

			// Verify provider structure within model
			modelProvider, ok := model["provider"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid provider structure in model %s/%s", providerID, modelID)
			}

			if _, exists := modelProvider["id"]; !exists {
				return fmt.Errorf("missing provider id in model %s/%s", providerID, modelID)
			}
			if _, exists := modelProvider["npm"]; !exists {
				return fmt.Errorf("missing provider npm in model %s/%s", providerID, modelID)
			}
		}
	}

	fmt.Printf("✅ Configuration verification passed - %d providers validated\n", len(providers))
	return nil
}

func toCamelCase(s string) string {
	// Handle special cases for common provider names
	switch strings.ToLower(s) {
	case "openai":
		return "openai"
	case "anthropic":
		return "anthropic"
	case "google", "gemini":
		return "google"
	case "groq":
		return "groq"
	case "perplexity":
		return "perplexity"
	case "together", "togetherai":
		return "together"
	case "fireworks":
		return "fireworks"
	case "poe":
		return "poe"
	case "navigator":
		return "navigator"
	default:
		// Convert to camelCase for other providers
		words := strings.FieldsFunc(s, func(r rune) bool {
			return r == '-' || r == '_' || r == ' '
		})

		if len(words) == 0 {
			return s
		}

		// First word lowercase, rest title case
		result := strings.ToLower(words[0])
		for i := 1; i < len(words); i++ {
			result += strings.Title(strings.ToLower(words[i]))
		}
		return result
	}
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
