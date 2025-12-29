package providers

import (
	"context"
	"fmt"
	"log"
	"time"

	"llm-verifier/logging"
)

// ExampleIntegration demonstrates how to integrate the mandatory verification system
// into existing code that uses the ModelProviderService
func ExampleIntegration() {
	// Setup logging
	logger, err := logging.NewLogger(nil, nil)
	if err != nil {
		log.Printf("Failed to create logger: %v", err)
		return
	}
	
	// Create verification configuration
	verificationConfig := VerificationConfig{
		Enabled:               true,  // Enable mandatory verification
		StrictMode:            true,  // Only verified models are usable
		MaxRetries:            3,
		TimeoutSeconds:        30,
		RequireAffirmative:    true,
		MinVerificationScore:  0.7,
	}
	
	// Create enhanced service with verification
	enhancedService := NewEnhancedModelProviderService("./config.yaml", logger, verificationConfig)
	
	// Register providers (this would normally come from config or environment)
	enhancedService.RegisterProvider("openai", "https://api.openai.com/v1", "your-openai-api-key")
	enhancedService.RegisterProvider("anthropic", "https://api.anthropic.com/v1", "your-anthropic-api-key")
	
	// Example 1: Get verified models for a specific provider
	fmt.Println("Example 1: Getting verified models for OpenAI")
	ctx := context.Background()
	
	verifiedModels, err := enhancedService.GetModelsWithVerification(ctx, "openai")
	if err != nil {
		log.Printf("Error getting verified models: %v", err)
		return
	}
	
	fmt.Printf("Found %d verified models for OpenAI:\n", len(verifiedModels))
	for _, model := range verifiedModels {
		verificationResult := enhancedService.GetModelVerificationResult("openai", model.ID)
		fmt.Printf("- %s (Verification Score: %.2f, Can See Code: %t)\n", 
			model.DisplayName, verificationResult.VerificationScore, verificationResult.CanSeeCode)
	}
	
	// Example 2: Get all verified models across all providers
	fmt.Println("\nExample 2: Getting all verified models")
	
	allVerifiedModels, err := enhancedService.GetAllModelsWithVerification(ctx)
	if err != nil {
		log.Printf("Error getting all verified models: %v", err)
		return
	}
	
	totalVerified := 0
	for providerID, models := range allVerifiedModels {
		fmt.Printf("Provider %s: %d verified models\n", providerID, len(models))
		totalVerified += len(models)
	}
	fmt.Printf("Total verified models: %d\n", totalVerified)
	
	// Example 3: Check if a specific model is verified
	fmt.Println("\nExample 3: Checking model verification status")
	
	modelID := "gpt-4"
	providerID := "openai"
	isVerified := enhancedService.IsModelVerified(providerID, modelID)
	
	if isVerified {
		fmt.Printf("✅ Model %s from %s is verified and can see code\n", modelID, providerID)
	} else {
		fmt.Printf("❌ Model %s from %s is NOT verified or cannot see code\n", modelID, providerID)
		
		// Get detailed verification result
		result := enhancedService.GetModelVerificationResult(providerID, modelID)
		if result != nil {
			fmt.Printf("   Status: %s, Score: %.2f, Can See Code: %t\n", 
				result.VerificationStatus, result.VerificationScore, result.CanSeeCode)
		}
	}
	
	// Example 4: Generate verified configuration
	fmt.Println("\nExample 4: Generating verified configuration")
	
	configGenerator := NewVerifiedConfigGenerator(enhancedService, logger, "./verified-configs")
	
	verifiedConfig, err := configGenerator.GenerateVerifiedConfig()
	if err != nil {
		log.Printf("Error generating verified config: %v", err)
		return
	}
	
	fmt.Printf("Generated verified configuration:\n")
	fmt.Printf("- Total models scanned: %d\n", verifiedConfig.TotalModels)
	fmt.Printf("- Verified models: %d\n", verifiedConfig.VerifiedModels)
	fmt.Printf("- Providers with verified models: %d\n", len(verifiedConfig.Providers))
	fmt.Printf("- Verification rate: %.1f%%\n", float64(verifiedConfig.VerifiedModels)/float64(verifiedConfig.TotalModels)*100)
	
	// Save the configuration
	err = configGenerator.SaveVerifiedConfig(verifiedConfig, "example")
	if err != nil {
		log.Printf("Error saving verified config: %v", err)
		return
	}
	
	fmt.Println("✅ Verified configuration saved successfully")
}

// ExampleWithErrorHandling shows how to handle verification errors and edge cases
func ExampleWithErrorHandling() {
	logger, err := logging.NewLogger(nil, nil)
	if err != nil {
		log.Printf("Failed to create logger: %v", err)
		return
	}
	
	// Create verification configuration with error handling settings
	verificationConfig := VerificationConfig{
		Enabled:               true,
		StrictMode:            false, // More lenient - allow models even if verification fails
		MaxRetries:            2,     // Fewer retries for faster failure
		TimeoutSeconds:        15,    // Shorter timeout
		RequireAffirmative:    true,
		MinVerificationScore:  0.5,   // Lower threshold
	}
	
	enhancedService := NewEnhancedModelProviderService("./config.yaml", logger, verificationConfig)
	
	// Register providers
	enhancedService.RegisterAllProviders()
	
	ctx := context.Background()
	
	// Example: Handle provider with no API key
	fmt.Println("Handling provider with no API key:")
	models, err := enhancedService.GetModelsWithVerification(ctx, "nonexistent-provider")
	if err != nil {
		fmt.Printf("Expected error for nonexistent provider: %v\n", err)
	} else {
		fmt.Printf("No models found for nonexistent provider (as expected)\n")
	}
	
	// Example: Handle verification timeout
	fmt.Println("\nHandling verification timeout:")
	
	// Create a context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	models, err = enhancedService.GetModelsWithVerification(ctxWithTimeout, "openai")
	if err != nil {
		fmt.Printf("Verification timed out or failed: %v\n", err)
	} else {
		fmt.Printf("Got %d models before timeout\n", len(models))
	}
	
	// Example: Handle mixed verification results
	fmt.Println("\nHandling mixed verification results:")
	
	allModels, err := enhancedService.GetAllModelsWithVerification(ctx)
	if err != nil {
		fmt.Printf("Error getting all models: %v\n", err)
		return
	}
	
	verificationResults := enhancedService.GetVerificationResults()
	
	var verifiedCount, failedCount, errorCount int
	
	for providerID, models := range allModels {
		for _, model := range models {
			verificationKey := fmt.Sprintf("%s:%s", providerID, model.ID)
			result := verificationResults[verificationKey]
			
			if result != nil {
				switch result.VerificationStatus {
				case "verified":
					verifiedCount++
				case "failed":
					failedCount++
				case "error":
					errorCount++
				}
			}
		}
	}
	
	fmt.Printf("Verification Results Summary:\n")
	fmt.Printf("- Verified: %d\n", verifiedCount)
	fmt.Printf("- Failed: %d\n", failedCount)
	fmt.Printf("- Error: %d\n", errorCount)
	fmt.Printf("- Total Processed: %d\n", verifiedCount+failedCount+errorCount)
}

// ExampleConfigurationIntegration shows how to integrate with existing configuration systems
func ExampleConfigurationIntegration() {
	logger, err := logging.NewLogger(nil, nil)
	if err != nil {
		log.Printf("Failed to create logger: %v", err)
		return
	}
	
	// Create verification configuration
	verificationConfig := VerificationConfig{
		Enabled:               true,
		StrictMode:            true,
		MaxRetries:            3,
		TimeoutSeconds:        30,
		RequireAffirmative:    true,
		MinVerificationScore:  0.7,
	}
	
	enhancedService := NewEnhancedModelProviderService("./config.yaml", logger, verificationConfig)
	enhancedService.RegisterAllProviders()
	
	// Generate verified configuration for different platforms
	configGenerator := NewVerifiedConfigGenerator(enhancedService, logger, "./configs")
	
	// Generate configuration for Crush platform
	fmt.Println("Generating verified configuration for Crush platform...")
	
	verifiedConfig, err := configGenerator.GenerateVerifiedConfig()
	if err != nil {
		log.Printf("Error generating config: %v", err)
		return
	}
	
	// Convert to Crush format (this would be platform-specific)
	crushConfig := convertToCrushFormat(verifiedConfig)
	
	// Save Crush configuration
	crushConfigPath := "./configs/crush_verified_config.json"
	if err := saveConfig(crushConfig, crushConfigPath); err != nil {
		log.Printf("Error saving Crush config: %v", err)
		return
	}
	
	fmt.Printf("✅ Crush configuration saved to: %s\n", crushConfigPath)
	
	// Generate configuration for OpenCode platform
	fmt.Println("Generating verified configuration for OpenCode platform...")
	
	opencodeConfig := convertToOpenCodeFormat(verifiedConfig)
	
	// Save OpenCode configuration
	opencodeConfigPath := "./configs/opencode_verified_config.json"
	if err := saveConfig(opencodeConfig, opencodeConfigPath); err != nil {
		log.Printf("Error saving OpenCode config: %v", err)
		return
	}
	
	fmt.Printf("✅ OpenCode configuration saved to: %s\n", opencodeConfigPath)
	
	// Show final statistics
	statistics, err := configGenerator.GetVerificationStatistics()
	if err != nil {
		log.Printf("Error getting statistics: %v", err)
		return
	}
	
	fmt.Println("\n📊 Final Verification Statistics:")
	fmt.Printf("Configuration generated at: %s\n", verifiedConfig.GeneratedAt.Format(time.RFC3339))
	fmt.Printf("Total models: %v\n", statistics["total_models_scanned"])
	fmt.Printf("Verified models: %v\n", statistics["verified_models"])
	fmt.Printf("Verification rate: %.1f%%\n", statistics["verification_rate"])
	fmt.Printf("Providers included: %v\n", statistics["providers_with_models"])
}

// Helper functions for configuration conversion
func convertToCrushFormat(config *VerifiedConfig) interface{} {
	// This would convert to Crush-specific format
	return map[string]interface{}{
		"version": "1.0",
		"generated": config.GeneratedAt,
		"models":    config.Providers,
	}
}

func convertToOpenCodeFormat(config *VerifiedConfig) interface{} {
	// This would convert to OpenCode-specific format
	return map[string]interface{}{
		"$schema":   "https://opencode.org/schema.json",
		"version":   "1.0",
		"generated": config.GeneratedAt,
		"providers": config.Providers,
	}
}

func saveConfig(config interface{}, filepath string) error {
	// Implementation would save the configuration to file
	// This is a placeholder
	return nil
}