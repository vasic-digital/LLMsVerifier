package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"llm-verifier/logging"
	"llm-verifier/providers"
)

func main() {
	var (
		configPath     = flag.String("config", "", "Path to configuration file")
		outputDir      = flag.String("output", "./verified-configs", "Output directory for verified configurations")
		verifyAll      = flag.Bool("verify-all", false, "Verify all available models")
		provider       = flag.String("provider", "", "Specific provider to verify (e.g., openai, anthropic)")
		model          = flag.String("model", "", "Specific model to verify")
		disableVerification = flag.Bool("no-verify", false, "Disable verification (for testing)")
		strictMode     = flag.Bool("strict", true, "Enable strict mode (only verified models are usable)")
		listProviders  = flag.Bool("list-providers", false, "List all available providers")
		statistics     = flag.Bool("stats", false, "Show verification statistics")
		verbose        = flag.Bool("verbose", false, "Enable verbose logging")
	)
	
	flag.Parse()
	
	// Setup logging
	logLevel := "info"
	if *verbose {
		logLevel = "debug"
	}
	
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": logLevel})
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("🔍 LLM Model Verification System - Mandatory 'Do you see my code?' Verification")
	fmt.Println(strings.Repeat("=", 80))
	
	// Create verification configuration
	verificationConfig := providers.VerificationConfig{
		Enabled:               !*disableVerification,
		StrictMode:            *strictMode,
		MaxRetries:            3,
		TimeoutSeconds:        30,
		RequireAffirmative:    true,
		MinVerificationScore:  0.7,
	}
	
	// Create enhanced model provider service
	enhancedService := providers.NewEnhancedModelProviderService(*configPath, logger, verificationConfig)
	
	// Register all providers from environment variables
	logger.Info("Registering all providers from environment variables", nil)
	enhancedService.RegisterAllProviders()
	
	// List providers if requested
	if *listProviders {
		listAllProviders(enhancedService, logger)
		return
	}
	
	// Show statistics if requested
	if *statistics {
		showVerificationStatistics(enhancedService, logger)
		return
	}
	
	// Perform verification based on command line arguments
	ctx := context.Background()
	
	if *model != "" && *provider != "" {
		// Verify specific model
		verifySpecificModel(ctx, enhancedService, *provider, *model, logger)
	} else if *provider != "" {
		// Verify all models for specific provider
		verifyProviderModels(ctx, enhancedService, *provider, logger)
	} else if *verifyAll {
		// Verify all models for all providers
		verifyAllModels(ctx, enhancedService, logger)
	} else {
		// Generate verified configuration
		generateVerifiedConfiguration(enhancedService, *outputDir, logger)
	}
}

func listAllProviders(service *providers.EnhancedModelProviderService, logger *logging.Logger) {
	fmt.Println("\n📋 Available Providers:")
	fmt.Println(strings.Repeat("-", 40))
	
	allProviders := service.GetAllProviders()
	if len(allProviders) == 0 {
		fmt.Println("No providers registered. Please set API keys in environment variables.")
		return
	}
	
	for providerID, client := range allProviders {
		status := "❌ No API Key"
		if client.APIKey != "" {
			status = "✅ Configured"
		}
		fmt.Printf("%-20s %s\n", providerID, status)
	}
	
	fmt.Printf("\nTotal providers: %d\n", len(allProviders))
}

func verifySpecificModel(ctx context.Context, service *providers.EnhancedModelProviderService, providerID, modelID string, logger *logging.Logger) {
	fmt.Printf("\n🔍 Verifying specific model: %s from provider: %s\n", modelID, providerID)
	fmt.Println(strings.Repeat("-", 60))
	
	// Get models for the provider
	models, err := service.GetModels(providerID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get models for provider %s: %v", providerID, err), nil)
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	
	// Find the specific model
	var targetModel *providers.Model
	for _, model := range models {
		if model.ID == modelID {
			targetModel = &model
			break
		}
	}
	
	if targetModel == nil {
		fmt.Printf("❌ Model %s not found for provider %s\n", modelID, providerID)
		return
	}
	
	// Get provider client
	providerClient, exists := service.GetAllProviders()[providerID]
	if !exists {
		fmt.Printf("❌ Provider client not found for %s\n", providerID)
		return
	}
	
	// Perform verification
	startTime := time.Now()
	result, err := service.GetVerificationService().VerifyModel(ctx, *targetModel, providerClient)
	duration := time.Since(startTime)
	
	if err != nil {
		fmt.Printf("❌ Verification failed: %v\n", err)
		return
	}
	
	// Display results
	fmt.Printf("✅ Verification completed in %v\n\n", duration)
	fmt.Printf("Model: %s\n", result.ModelID)
	fmt.Printf("Provider: %s\n", result.ProviderID)
	fmt.Printf("Status: %s\n", result.VerificationStatus)
	fmt.Printf("Can See Code: %t\n", result.CanSeeCode)
	fmt.Printf("Affirmative Response: %t\n", result.AffirmativeResponse)
	fmt.Printf("Verification Score: %.2f\n", result.VerificationScore)
	fmt.Printf("Last Verified: %s\n", result.LastVerifiedAt.Format(time.RFC3339))
	
	if result.ErrorMessage != "" {
		fmt.Printf("Error: %s\n", result.ErrorMessage)
	}
	
	// Determine overall result
	if result.VerificationStatus == "verified" && result.CanSeeCode && result.AffirmativeResponse {
		fmt.Println("\n🎉 Model PASSED mandatory verification!")
	} else {
		fmt.Println("\n❌ Model FAILED mandatory verification!")
	}
}

func verifyProviderModels(ctx context.Context, service *providers.EnhancedModelProviderService, providerID string, logger *logging.Logger) {
	fmt.Printf("\n🔍 Verifying all models for provider: %s\n", providerID)
	fmt.Println(strings.Repeat("-", 60))
	
	// Get models with verification
	models, err := service.GetModelsWithVerification(ctx, providerID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get verified models for provider %s: %v", providerID, err), nil)
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Found %d verified models for provider %s\n\n", len(models), providerID)
	
	// Display verification results
	verificationResults := service.GetVerificationResults()
	
	for _, model := range models {
		verificationKey := fmt.Sprintf("%s:%s", providerID, model.ID)
		result := verificationResults[verificationKey]
		
		if result != nil {
			fmt.Printf("✅ %s (Score: %.2f, Can See Code: %t)\n", 
				model.DisplayName, result.VerificationScore, result.CanSeeCode)
		}
	}
	
	fmt.Printf("\n📊 Summary: %d/%d models verified successfully\n", len(models), len(models))
}

func verifyAllModels(ctx context.Context, service *providers.EnhancedModelProviderService, logger *logging.Logger) {
	fmt.Println("\n🔍 Verifying all models across all providers")
	fmt.Println(strings.Repeat("-", 60))
	
	startTime := time.Now()
	
	// Get all models with verification
	allModels, err := service.GetAllModelsWithVerification(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get all verified models: %v", err), nil)
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	
	duration := time.Since(startTime)
	
	// Calculate statistics
	totalModels := 0
	verifiedModels := 0
	
	for _, models := range allModels {
		totalModels += len(models)
		verifiedModels += len(models)
	}
	
	fmt.Printf("✅ Verification completed in %v\n", duration)
	fmt.Printf("📊 Summary: %d verified models across %d providers\n\n", verifiedModels, len(allModels))
	
	// Display breakdown by provider
	verificationResults := service.GetVerificationResults()
	
	for providerID, models := range allModels {
		fmt.Printf("📋 %s: %d verified models\n", providerID, len(models))
		
		for _, model := range models {
			verificationKey := fmt.Sprintf("%s:%s", providerID, model.ID)
			result := verificationResults[verificationKey]
			
			if result != nil {
				fmt.Printf("  ✅ %s (Score: %.2f)\n", model.DisplayName, result.VerificationScore)
			}
		}
		fmt.Println()
	}
	
	fmt.Printf("🎯 Overall Verification Rate: %d/%d (%.1f%%)\n", 
		verifiedModels, totalModels, float64(verifiedModels)/float64(totalModels)*100)
}

func generateVerifiedConfiguration(service *providers.EnhancedModelProviderService, outputDir string, logger *logging.Logger) {
	fmt.Println("\n🔧 Generating Verified Configuration")
	fmt.Println(strings.Repeat("-", 60))
	
	// Create config generator
	configGenerator := providers.NewVerifiedConfigGenerator(service, logger, outputDir)
	
	// Generate and save verified configuration
	startTime := time.Now()
	err := configGenerator.GenerateAndSaveVerifiedConfig("llm_verifier")
	duration := time.Since(startTime)
	
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to generate verified configuration: %v", err), nil)
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Verified configuration generated in %v\n", duration)
	fmt.Printf("📁 Configuration files saved to: %s\n", outputDir)
	
	// Show statistics
	statistics, err := configGenerator.GetVerificationStatistics()
	if err != nil {
		logger.Warning(fmt.Sprintf("Failed to get verification statistics: %v", err), nil)
		return
	}
	
	fmt.Println("\n📊 Verification Statistics:")
	fmt.Printf("  Total Models Scanned: %v\n", statistics["total_models_scanned"])
	fmt.Printf("  Verified Models: %v\n", statistics["verified_models"])
	fmt.Printf("  Verification Rate: %.1f%%\n", statistics["verification_rate"])
	fmt.Printf("  Providers with Models: %v\n", statistics["providers_with_models"])
	fmt.Printf("  Verification Enabled: %v\n", statistics["verification_enabled"])
	fmt.Printf("  Strict Mode: %v\n", statistics["strict_mode"])
	
	fmt.Println("\n🎉 Verified configuration generation complete!")
}

func showVerificationStatistics(service *providers.EnhancedModelProviderService, logger *logging.Logger) {
	fmt.Println("\n📊 Verification Statistics")
	fmt.Println(strings.Repeat("-", 60))
	
	configGenerator := providers.NewVerifiedConfigGenerator(service, logger, "./temp")
	
	statistics, err := configGenerator.GetVerificationStatistics()
	if err != nil {
		fmt.Printf("❌ Failed to get statistics: %v\n", err)
		return
	}
	
	fmt.Printf("Total Models Scanned: %v\n", statistics["total_models_scanned"])
	fmt.Printf("Verified Models: %v\n", statistics["verified_models"])
	fmt.Printf("Verification Rate: %.1f%%\n", statistics["verification_rate"])
	fmt.Printf("Providers with Models: %v\n", statistics["providers_with_models"])
	fmt.Printf("Verification Enabled: %v\n", statistics["verification_enabled"])
	fmt.Printf("Strict Mode: %v\n", statistics["strict_mode"])
	
	if providerBreakdown, ok := statistics["provider_breakdown"].(map[string]interface{}); ok {
		fmt.Println("\nProvider Breakdown:")
		for provider, stats := range providerBreakdown {
			if providerStats, ok := stats.(map[string]interface{}); ok {
				fmt.Printf("  %s: %v/%v models (%.1f%% success rate)\n", 
					provider, 
					providerStats["verified_count"], 
					providerStats["total_models"], 
					providerStats["success_rate"])
			}
		}
	}
}