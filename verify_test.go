package main

import (
	"fmt"
	"os"
	"path/filepath"

	crush_config "llm-verifier/pkg/crush/config"
	crush_verifier "llm-verifier/pkg/crush/verifier"
)

func main() {
	// Test Crush configuration verification with full provider/model validation
	fmt.Println("=== Crush Configuration Verifier Test ===")
	fmt.Println()

	// Get the current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// Test with the full Crush config
	configPath := filepath.Join(currentDir, "..", "test_crush_full.json")
	
	// Create verifier
	verifier := crush_verifier.NewCrushVerifier(nil, configPath)
	
	// Verify configuration
	result, err := verifier.VerifyConfiguration()
	if err != nil {
		fmt.Printf("Error during verification: %v\n", err)
		return
	}

	// Print detailed results
	fmt.Printf("Config File: %s\n", result.ConfigFile)
	fmt.Printf("Valid: %v\n", result.Valid)
	fmt.Printf("Overall Score: %.1f/100\n", result.OverallScore)
	fmt.Printf("Errors: %d\n", len(result.Errors))
	fmt.Printf("Warnings: %d\n", len(result.Warnings))
	fmt.Println()

	// Print provider status
	fmt.Println("=== Provider Verification Status ===")
	for name, provider := range result.ProviderStatus {
		fmt.Printf("\nProvider: %s\n", name)
		fmt.Printf("  Name: %s\n", provider.Name)
		fmt.Printf("  Type: %s\n", provider.Type)
		fmt.Printf("  Has API Key: %v\n", provider.HasAPIKey)
		fmt.Printf("  Model Count: %d\n", provider.ModelCount)
		fmt.Printf("  Score: %.1f/100\n", provider.Score)
	}

	// Print model status
	fmt.Println("\n=== Model Verification Status ===")
	for providerName, models := range result.ModelStatus {
		fmt.Printf("\nProvider: %s\n", providerName)
		for modelID, model := range models {
			fmt.Printf("  Model: %s (%s)\n", modelID, model.Name)
			fmt.Printf("    Has Cost Info: %v\n", model.HasCostInfo)
			fmt.Printf("    Has Context Info: %v\n", model.HasContextInfo)
			fmt.Printf("    Has Feature Flags: %v\n", model.HasFeatureFlags)
			fmt.Printf("    Score: %.1f/100\n", model.Score)
		}
	}

	// Print LSP status
	fmt.Println("\n=== LSP Verification Status ===")
	for name, lsp := range result.LspStatus {
		fmt.Printf("\nLSP: %s\n", name)
		fmt.Printf("  Command: %s\n", lsp.Command)
		fmt.Printf("  Enabled: %v\n", lsp.Enabled)
		fmt.Printf("  Args: %v\n", lsp.Args)
		fmt.Printf("  Score: %.1f/100\n", lsp.Score)
	}

	// Test individual provider verification
	fmt.Println("\n=== Individual Provider Tests ===")
	cfg, err := crush_config.LoadAndParse(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	for name, provider := range cfg.Providers {
		status := verifier.VerifyProvider(name, &provider)
		fmt.Printf("\nProvider '%s':\n", name)
		fmt.Printf("  Type: %s\n", status.Type)
		fmt.Printf("  API Key Present: %v\n", status.HasAPIKey)
		fmt.Printf("  Models Configured: %d\n", status.ModelCount)
		fmt.Printf("  Verification Score: %.1f/100\n", status.Score)
	}

	// Test individual model verification
	fmt.Println("\n=== Individual Model Tests ===")
	for providerName, provider := range cfg.Providers {
		fmt.Printf("\nProvider: %s\n", providerName)
		for _, model := range provider.Models {
			status := verifier.VerifyModel(&model)
			fmt.Printf("  Model '%s' (%s):\n", model.ID, model.Name)
			fmt.Printf("    Cost Configuration: %v\n", status.HasCostInfo)
			fmt.Printf("    Context Configuration: %v\n", status.HasContextInfo)
			fmt.Printf("    Feature Flags: %v\n", status.HasFeatureFlags)
			fmt.Printf("    Model Score: %.1f/100\n", status.Score)
		}
	}

	fmt.Println("\n=== Test Complete ===")
}