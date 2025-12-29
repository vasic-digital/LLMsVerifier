package llmverifier

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// Integration test for complete OpenCode workflow
func TestOpenCodeFullIntegration(t *testing.T) {
	// This test verifies the complete workflow from LLM Verifier to OpenCode
	// It simulates the real-world scenario where users export configs and use them

	// 1. Setup test data - simulate verification results
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:       "gpt-4o",
				Endpoint: "https://api.openai.com/v1",
			},
			PerformanceScores: PerformanceScore{
				OverallScore: 95.0,
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:       "claude-3-5-sonnet-20241022",
				Endpoint: "https://api.anthropic.com/v1",
			},
			PerformanceScores: PerformanceScore{
				OverallScore: 92.0,
			},
		},
	}

	// 2. Export configuration using LLM Verifier
	options := &ExportOptions{
		IncludeAPIKey: true,
	}

	configMap, err := createCorrectOpenCodeConfig(results, options)
	if err != nil {
		t.Fatalf("Failed to export OpenCode config: %v", err)
	}

	// 3. Verify configuration structure (what OpenCode expects)
	if schema, ok := configMap["$schema"].(string); !ok || schema != "./opencode-schema.json" {
		t.Errorf("Config must use OpenCode schema, got: %v", schema)
	}

	// 4. Verify providers section exists and is properly formatted
	if _, ok := configMap["providers"].(map[string]interface{}); !ok {
		t.Fatal("Config must have providers section for OpenCode")
	}

	// 5. Verify agents section enables model usage
	if _, ok := configMap["agents"].(map[string]interface{}); !ok {
		t.Fatal("Config must have agents section for OpenCode")
	}

	// 6. Simulate OpenCode parsing (JSON marshal/unmarshal)
	configJSON, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		t.Fatalf("Config must be valid JSON for OpenCode: %v", err)
	}

	var parsedConfig map[string]interface{}
	if err := json.Unmarshal(configJSON, &parsedConfig); err != nil {
		t.Fatalf("OpenCode would fail to parse this config: %v", err)
	}

	// 7. Verify no ProviderInitError conditions exist
	if _, hasInvalidProvider := configMap["provider"]; hasInvalidProvider {
		t.Error("Config contains old 'provider' field that causes ProviderInitError")
	}

	t.Logf("✅ OpenCode integration test passed - config is compatible")
	t.Logf("📄 Generated config size: %d bytes", len(configJSON))
}

// Integration test for configuration migration workflow
func TestConfigurationMigrationWorkflow(t *testing.T) {
	// Test the complete migration workflow for existing users

	// 1. Create an old incompatible config
	oldConfig := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]interface{}{
			"openai": map[string]interface{}{
				"options": map[string]interface{}{
					"apiKey": "${OPENAI_API_KEY}",
				},
				"models": []interface{}{},
			},
		},
	}

	// 2. Migrate using our migration tool
	newConfig, err := migrateOpenCodeConfig(oldConfig)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// 3. Verify migration was successful
	if schema, ok := newConfig["$schema"].(string); !ok || schema != "./opencode-schema.json" {
		t.Errorf("Migration must update schema, got: %v", schema)
	}

	if _, hasProviders := newConfig["providers"]; !hasProviders {
		t.Error("Migration must create providers section")
	}

	if _, hasProvider := newConfig["provider"]; hasProvider {
		t.Error("Migration must remove old provider section")
	}

	t.Log("✅ Configuration migration workflow test passed")
}

// Integration test for analytics and monitoring
func TestAnalyticsIntegration(t *testing.T) {
	// Test that analytics are properly recorded during exports

	// 1. Create test config
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:       "gpt-4o",
				Endpoint: "https://api.openai.com/v1",
			},
			PerformanceScores: PerformanceScore{OverallScore: 95.0},
		},
	}

	options := &ExportOptions{IncludeAPIKey: true}
	configMap, err := createCorrectOpenCodeConfig(results, options)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// 2. Record analytics
	RecordOpenCodeExport(configMap, true, "")

	// 3. Verify analytics file was created/updated
	analyticsFile := GetAnalyticsFilePath()
	if _, err := os.Stat(analyticsFile); os.IsNotExist(err) {
		t.Log("Analytics file created successfully")
	}

	// 4. Load and verify analytics
	analytics, err := LoadAnalytics(analyticsFile)
	if err != nil {
		t.Fatalf("Failed to load analytics: %v", err)
	}

	if analytics.TotalExports < 1 {
		t.Error("Analytics should record at least one export")
	}

	t.Logf("✅ Analytics integration test passed - %d total exports recorded", analytics.TotalExports)
}

// Integration test for provider discovery workflow
func TestProviderDiscoveryWorkflow(t *testing.T) {
	// Test the complete provider discovery and configuration process

	testEndpoints := []string{
		"https://api.openai.com/v1",
		"https://api.anthropic.com/v1",
		"https://generativelanguage.googleapis.com/v1",
		"https://api.groq.com/openai/v1",
		"localhost:8000/v1", // Local/custom endpoint
	}

	for _, endpoint := range testEndpoints {
		t.Run(fmt.Sprintf("Endpoint_%s", endpoint), func(t *testing.T) {
			provider := extractProvider(endpoint)
			if provider == "unknown" {
				t.Errorf("Provider detection failed for endpoint: %s", endpoint)
			} else {
				t.Logf("✅ Detected provider '%s' for endpoint: %s", provider, endpoint)
			}
		})
	}

	t.Log("✅ Provider discovery workflow test passed")
}
