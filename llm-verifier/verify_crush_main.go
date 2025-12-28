package main

import (
	"encoding/json"
	"fmt"
	"os"

	crush_config "llm-verifier/pkg/crush/config"
	crush_verifier "llm-verifier/pkg/crush/verifier"
	"llm-verifier/database"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║        CRUSH CONFIGURATION VERIFIER - FULL TEST              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Initialize database
	db, err := database.New("./crush_verifications.db")
	if err != nil {
		fmt.Printf("❌ Database initialization failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test with the full Crush config
	configPath := "../test_crush_full.json"
	
	fmt.Printf("📁 Loading configuration: %s\n", configPath)
	
	// Load configuration
	cfg, err := crush_config.LoadAndParse(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Configuration loaded successfully")
	fmt.Println()

	// Create verifier
	verifier := crush_verifier.NewCrushVerifier(db, configPath)
	
	fmt.Println("🔍 Verifying configuration...")
	fmt.Println()
	
	// Verify configuration
	result, err := verifier.VerifyConfiguration()
	if err != nil {
		fmt.Printf("❌ Verification failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    VERIFICATION SUMMARY                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("📊 Config File: %s\n", result.ConfigFile)
	fmt.Printf("✅ Valid: %v\n", result.Valid)
	fmt.Printf("📈 Overall Score: %.1f/100\n", result.OverallScore)
	fmt.Printf("⚠️  Errors: %d\n", len(result.Errors))
	fmt.Printf("🔔 Warnings: %d\n", len(result.Warnings))
	fmt.Println()

	// Detailed Provider Analysis
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                 PROVIDER VERIFICATION STATUS                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	for name, provider := range result.ProviderStatus {
		statusIcon := "✅"
		if provider.Score < 70 {
			statusIcon = "⚠️"
		}
		if provider.Score < 50 {
			statusIcon = "❌"
		}
		
		fmt.Printf("%s Provider: %s (%s)\n", statusIcon, name, provider.Type)
		fmt.Printf("   ├─ Name: %s\n", provider.Name)
		fmt.Printf("   ├─ Type: %s\n", provider.Type)
		fmt.Printf("   ├─ API Key Present: %v\n", provider.HasAPIKey)
		fmt.Printf("   ├─ Models Configured: %d\n", provider.ModelCount)
		fmt.Printf("   └─ Score: %.1f/100\n", provider.Score)
		fmt.Println()
	}

	// Detailed Model Analysis (per provider)
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  MODEL VERIFICATION STATUS                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	totalModels := 0
	for providerName, models := range result.ModelStatus {
		fmt.Printf("📦 Provider: %s\n", providerName)
		fmt.Printf("   Models: %d\n", len(models))
		fmt.Println()
		
		for modelID, model := range models {
			totalModels++
			statusIcon := "✅"
			if model.Score < 70 {
				statusIcon = "⚠️"
			}
			if model.Score < 50 {
				statusIcon = "❌"
			}
			
			fmt.Printf("   %s Model: %s\n", statusIcon, modelID)
			fmt.Printf("      ├─ Name: %s\n", model.Name)
			fmt.Printf("      ├─ Cost Configuration: %v\n", model.HasCostInfo)
			fmt.Printf("      ├─ Context Configuration: %v\n", model.HasContextInfo)
			fmt.Printf("      ├─ Feature Flags: %v\n", model.HasFeatureFlags)
			fmt.Printf("      └─ Score: %.1f/100\n", model.Score)
		}
		fmt.Println()
	}

	// LSP Status
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    LSP VERIFICATION STATUS                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	if len(result.LspStatus) > 0 {
		for name, lsp := range result.LspStatus {
			statusIcon := "✅"
			if !lsp.Enabled {
				statusIcon = "⚠️"
			}
			if lsp.Score < 50 {
				statusIcon = "❌"
			}
			
			fmt.Printf("%s LSP: %s\n", statusIcon, name)
			fmt.Printf("   ├─ Command: %s\n", lsp.Command)
			fmt.Printf("   ├─ Enabled: %v\n", lsp.Enabled)
			fmt.Printf("   ├─ Args: %v\n", lsp.Args)
			fmt.Printf("   └─ Score: %.1f/100\n", lsp.Score)
			fmt.Println()
		}
	} else {
		fmt.Println("ℹ️  No LSP configurations found")
		fmt.Println()
	}

	// Individual Verification Tests
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║               INDIVIDUAL VERIFICATION TESTS                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Test each provider individually
	fmt.Println("🔍 Testing Individual Providers:")
	fmt.Println()
	
	for name, provider := range cfg.Providers {
		fmt.Printf("   Testing '%s'... ", name)
		status := verifier.VerifyProvider(name, &provider)
		
		if status.Score >= 80 {
			fmt.Printf("✅ Score: %.1f/100\n", status.Score)
		} else if status.Score >= 60 {
			fmt.Printf("⚠️  Score: %.1f/100\n", status.Score)
		} else {
			fmt.Printf("❌ Score: %.1f/100\n", status.Score)
		}
		
		// Test each model in the provider
		for _, model := range provider.Models {
			fmt.Printf("      └─ Model '%s'... ", model.ID)
			modelStatus := verifier.VerifyModel(&model)
			
			if modelStatus.Score >= 80 {
				fmt.Printf("✅ Score: %.1f/100\n", modelStatus.Score)
			} else if modelStatus.Score >= 60 {
				fmt.Printf("⚠️  Score: %.1f/100\n", modelStatus.Score)
			} else {
				fmt.Printf("❌ Score: %.1f/100\n", modelStatus.Score)
			}
		}
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    FINAL SUMMARY                             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("Total Providers: %d\n", len(result.ProviderStatus))
	fmt.Printf("Total Models: %d\n", totalModels)
	fmt.Printf("Total LSPs: %d\n", len(result.LspStatus))
	fmt.Printf("Overall Quality Score: %.1f/100\n", result.OverallScore)
	fmt.Println()

	if result.Valid && result.OverallScore >= 80 {
		fmt.Println("🎉 Configuration is VALID and OPTIMIZED!")
	} else if result.Valid && result.OverallScore >= 60 {
		fmt.Println("✅ Configuration is VALID with room for improvement.")
	} else if result.Valid {
		fmt.Println("⚠️  Configuration is VALID but needs significant improvements.")
	} else {
		fmt.Println("❌ Configuration is INVALID - please fix the errors above.")
	}

	fmt.Println()
	
	// Store verification result
	verificationData := map[string]interface{}{
		"config_path": result.ConfigFile,
		"valid":       result.Valid,
		"score":       result.OverallScore,
		"providers":   len(result.ProviderStatus),
		"models":      totalModels,
		"lsps":        len(result.LspStatus),
		"errors":      len(result.Errors),
		"warnings":    len(result.Warnings),
		"timestamp":   "2025-12-28T19:30:00Z",
	}

	resultJSON, _ := json.MarshalIndent(verificationData, "", "  ")
	fmt.Println("💾 Verification result stored in database")
	fmt.Printf("📋 Raw Data: %s\n", string(resultJSON))
}