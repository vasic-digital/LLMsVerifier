package main

import (
	"fmt"
	"log"
	"os"

	"llm-verifier/llmverifier"
)

func main() {
	// Test export functionality
	fmt.Println("Testing AI CLI Export functionality...")

	// Test OpenCode export
	fmt.Println("\n1. Testing OpenCode export...")
	err := llmverifier.ExportAIConfig(nil, "opencode", "test_opencode.json", &llmverifier.ExportOptions{
		Top:           3,
		MinScore:      70.0,
		IncludeAPIKey: false,
	})
	if err != nil {
		log.Printf("OpenCode export failed: %v", err)
	} else {
		fmt.Println("✅ OpenCode export successful")
	}

	// Test Crush export
	fmt.Println("\n2. Testing Crush export...")
	err = llmverifier.ExportAIConfig(nil, "crush", "test_crush.json", &llmverifier.ExportOptions{
		Top:           2,
		MinScore:      75.0,
		Categories:    []string{"coding"},
		IncludeAPIKey: false,
	})
	if err != nil {
		log.Printf("Crush export failed: %v", err)
	} else {
		fmt.Println("✅ Crush export successful")
	}

	// Test Claude Code export
	fmt.Println("\n3. Testing Claude Code export...")
	err = llmverifier.ExportAIConfig(nil, "claude-code", "test_claude_code.json", &llmverifier.ExportOptions{
		Top:           5,
		Categories:    []string{"reasoning", "coding"},
		IncludeAPIKey: false,
	})
	if err != nil {
		log.Printf("Claude Code export failed: %v", err)
	} else {
		fmt.Println("✅ Claude Code export successful")
	}

	// Show generated files
	fmt.Println("\n4. Generated configuration files:")
	files := []string{"test_opencode.json", "test_crush.json", "test_claude_code.json"}

	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			fmt.Printf("   ✅ %s\n", file)
		} else {
			fmt.Printf("   ❌ %s (not found)\n", file)
		}
	}

	fmt.Println("\nExport functionality test completed!")
}
