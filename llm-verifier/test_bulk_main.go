package main

import (
	"fmt"
	"log"
	"os"

	"llm-verifier/llmverifier"
)

func main() {
	fmt.Println("Testing Bulk Export functionality...")

	// Create output directory
	outputDir := "bulk_exports"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Test bulk export with custom options
	fmt.Println("\n1. Testing bulk export with custom options...")
	err := llmverifier.ExportBulkConfig(nil, outputDir, &llmverifier.ExportOptions{
		Top:           5,
		MinScore:      70.0,
		Categories:    []string{"coding", "reasoning"},
		IncludeAPIKey: false,
	})
	if err != nil {
		log.Printf("Bulk export failed: %v", err)
	} else {
		fmt.Println("✅ Bulk export successful")
	}

	// Test bulk export with default options
	fmt.Println("\n2. Testing bulk export with default options...")
	err = llmverifier.ExportBulkConfig(nil, outputDir+"default", nil)
	if err != nil {
		log.Printf("Default bulk export failed: %v", err)
	} else {
		fmt.Println("✅ Default bulk export successful")
	}

	// Validate exported configurations
	fmt.Println("\n3. Validating exported configurations...")
	configFiles := []string{
		"bulk_exports/export_opencode.json",
		"bulk_exports/export_crush.json",
		"bulk_exports/export_claude_code.json",
	}

	for _, configFile := range configFiles {
		if _, err := os.Stat(configFile); err == nil {
			err = llmverifier.ValidateExportedConfig(configFile)
			if err != nil {
				log.Printf("❌ Validation failed for %s: %v", configFile, err)
			} else {
				fmt.Printf("✅ %s is valid\n", configFile)
			}
		} else {
			fmt.Printf("❌ %s not found\n", configFile)
		}
	}

	// Show export summary
	fmt.Println("\n4. Export summary...")
	if _, err := os.Stat("bulk_exports/export_summary.json"); err == nil {
		fmt.Println("✅ Export summary created")
		data, _ := os.ReadFile("bulk_exports/export_summary.json")
		fmt.Printf("Summary: %s\n", string(data))
	} else {
		fmt.Println("❌ Export summary not found")
	}

	// List all generated files
	fmt.Println("\n5. Generated files:")
	entries, err := os.ReadDir("bulk_exports")
	if err != nil {
		log.Printf("Failed to read output directory: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				fmt.Printf("   📄 %s (%d bytes)\n", entry.Name(), info.Size())
			} else {
				fmt.Printf("   📄 %s\n", entry.Name())
			}
		}
	}

	fmt.Println("\nBulk export test completed!")
}
