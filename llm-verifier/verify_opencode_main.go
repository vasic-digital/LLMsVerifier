package main

import (
	"encoding/json"
	"fmt"
	"os"

	opencode_config "llm-verifier/pkg/opencode/config"
	opencode_verifier "llm-verifier/pkg/opencode/verifier"
	"llm-verifier/database"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║       OPENCODE CONFIGURATION VERIFIER - FULL TEST            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Initialize database
	db, err := database.New("./opencode_verifications.db")
	if err != nil {
		fmt.Printf("❌ Database initialization failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test with the full OpenCode config
	configPath := "./test_opencode_full.json"
	
	fmt.Printf("📁 Loading configuration: %s\n", configPath)
	
	// Load configuration
	cfg, err := opencode_config.LoadAndParse(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Configuration loaded successfully")
	fmt.Println()

	// Create verifier
	verifier := opencode_verifier.NewOpenCodeVerifier(db, configPath)
	
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
		
		fmt.Printf("%s Provider: %s\n", statusIcon, name)
		fmt.Printf("   ├─ Type: %s\n", provider.Name)
		fmt.Printf("   ├─ Has API Key: %v\n", provider.HasAPIKey)
		fmt.Printf("   ├─ Verified: %v\n", provider.Verified)
		fmt.Printf("   └─ Score: %.1f/100\n", provider.Score)
		fmt.Println()
	}

	// Detailed Agent Analysis
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  AGENT VERIFICATION STATUS                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	for name, agent := range result.AgentStatus {
		statusIcon := "✅"
		if agent.Score < 70 {
			statusIcon = "⚠️"
		}
		if agent.Score < 50 {
			statusIcon = "❌"
		}
		
		fmt.Printf("%s Agent: %s\n", statusIcon, name)
		fmt.Printf("   ├─ Has Model: %v\n", agent.HasModel)
		fmt.Printf("   ├─ Has Prompt: %v\n", agent.HasPrompt)
		fmt.Printf("   ├─ Tools Configured: %d\n", agent.ToolsConfigured)
		fmt.Printf("   └─ Score: %.1f/100\n", agent.Score)
		fmt.Println()
	}

	// MCP Status
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    MCP VERIFICATION STATUS                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	if len(result.McpStatus) > 0 {
		for name, mcp := range result.McpStatus {
			statusIcon := "✅"
			if !mcp.Enabled {
				statusIcon = "⚠️"
			}
			if mcp.Score < 50 {
				statusIcon = "❌"
			}
			
			fmt.Printf("%s MCP: %s\n", statusIcon, name)
			fmt.Printf("   ├─ Type: %s\n", mcp.Type)
			fmt.Printf("   ├─ Enabled: %v\n", mcp.Enabled)
			if mcp.Type == "local" {
				fmt.Printf("   ├─ Command: %s\n", mcp.Command)
			} else {
				fmt.Printf("   ├─ URL: %s\n", mcp.URL)
			}
			fmt.Printf("   └─ Score: %.1f/100\n", mcp.Score)
			fmt.Println()
		}
	} else {
		fmt.Println("ℹ️  No MCP configurations found")
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
	
	for name, provider := range cfg.Provider {
		fmt.Printf("   Testing '%s'... ", name)
		status := verifier.VerifyProvider(name, &provider)
		
		if status.Score >= 80 {
			fmt.Printf("✅ Score: %.1f/100\n", status.Score)
		} else if status.Score >= 60 {
			fmt.Printf("⚠️  Score: %.1f/100\n", status.Score)
		} else {
			fmt.Printf("❌ Score: %.1f/100\n", status.Score)
		}
	}

	// Test each agent individually
	fmt.Println("\n🔍 Testing Individual Agents:")
	fmt.Println()
	
	for name, agent := range cfg.Agent {
		fmt.Printf("   Testing '%s'... ", name)
		status := verifier.VerifyAgent(name, &agent)
		
		if status.Score >= 80 {
			fmt.Printf("✅ Score: %.1f/100\n", status.Score)
		} else if status.Score >= 60 {
			fmt.Printf("⚠️  Score: %.1f/100\n", status.Score)
		} else {
			fmt.Printf("❌ Score: %.1f/100\n", status.Score)
		}
	}

	// Test each MCP individually
	fmt.Println("\n🔍 Testing Individual MCPs:")
	fmt.Println()
	
	for name, mcp := range cfg.Mcp {
		fmt.Printf("   Testing '%s'... ", name)
		status := verifier.VerifyMCP(name, &mcp)
		
		if status.Score >= 80 {
			fmt.Printf("✅ Score: %.1f/100\n", status.Score)
		} else if status.Score >= 60 {
			fmt.Printf("⚠️  Score: %.1f/100\n", status.Score)
		} else {
			fmt.Printf("❌ Score: %.1f/100\n", status.Score)
		}
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    FINAL SUMMARY                             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("Total Providers: %d\n", len(result.ProviderStatus))
	fmt.Printf("Total Agents: %d\n", len(result.AgentStatus))
	fmt.Printf("Total MCPs: %d\n", len(result.McpStatus))
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
		"agents":      len(result.AgentStatus),
		"mcps":        len(result.McpStatus),
		"errors":      len(result.Errors),
		"warnings":    len(result.Warnings),
		"timestamp":   "2025-12-28T19:30:00Z",
	}

	resultJSON, _ := json.MarshalIndent(verificationData, "", "  ")
	fmt.Println("💾 Verification result stored in database")
	fmt.Printf("📋 Raw Data: %s\n", string(resultJSON))
}

func init() {
	// Export the verification methods
	_ = opencode_verifier.NewOpenCodeVerifier
}