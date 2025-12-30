package llmverifier

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFeatureSuffixes tests that all expected feature suffixes are defined
func TestFeatureSuffixes(t *testing.T) {
	expectedSuffixes := []string{
		"(brotli)",
		"(http3)",
		"(toon)",
		"(streaming)",
		"(free to use)",
		"(open source)",
		"(fast)",
		"(llmsvd)",
	}

	for _, suffix := range expectedSuffixes {
		found := false
		for _, fs := range FeatureSuffixes {
			if fs == suffix {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected suffix %s not found in FeatureSuffixes", suffix)
	}
}

// TestFormatModelNameWithSuffixes tests model name formatting with feature suffixes
func TestFormatModelNameWithSuffixes(t *testing.T) {
	tests := []struct {
		name          string
		modelID       string
		result        VerificationResult
		isVerified    bool
		wantContains  []string
		wantNotContain []string
	}{
		{
			name:    "verified model with brotli support",
			modelID: "gpt-4o",
			result: VerificationResult{
				ModelInfo: ModelInfo{
					Endpoint:       "https://api.openai.com/v1",
					SupportsBrotli: true,
				},
				FeatureDetection: FeatureDetectionResult{
					SupportsBrotli: true,
					Streaming:      true,
				},
			},
			isVerified:   true,
			wantContains: []string{"gpt-4o", "(brotli)", "(streaming)", "(llmsvd)"},
		},
		{
			name:    "verified model with http3 support",
			modelID: "gemini-pro",
			result: VerificationResult{
				ModelInfo: ModelInfo{
					Endpoint:      "https://generativelanguage.googleapis.com/v1",
					SupportsHTTP3: true,
				},
				FeatureDetection: FeatureDetectionResult{
					SupportsHTTP3: true,
				},
			},
			isVerified:   true,
			wantContains: []string{"gemini-pro", "(http3)", "(llmsvd)"},
		},
		{
			name:    "verified model with toon support",
			modelID: "dall-e-3",
			result: VerificationResult{
				ModelInfo: ModelInfo{
					Endpoint:     "https://api.openai.com/v1",
					SupportsToon: true,
				},
				FeatureDetection: FeatureDetectionResult{
					SupportsToon: true,
				},
			},
			isVerified:   true,
			wantContains: []string{"dall-e-3", "(toon)", "(llmsvd)"},
		},
		{
			name:    "unverified model - no llmsvd suffix",
			modelID: "unknown-model",
			result: VerificationResult{
				ModelInfo: ModelInfo{
					Endpoint: "https://api.unknown.com/v1",
				},
				FeatureDetection: FeatureDetectionResult{},
			},
			isVerified:     false,
			wantContains:   []string{"unknown-model"},
			wantNotContain: []string{"(llmsvd)"},
		},
		{
			name:    "free provider model",
			modelID: "free-model",
			result: VerificationResult{
				ModelInfo: ModelInfo{
					Endpoint: "https://api.chutes.ai/v1", // Chutes is a free provider
				},
				FeatureDetection: FeatureDetectionResult{
					Streaming: true,
				},
			},
			isVerified:   true,
			wantContains: []string{"free-model", "(free to use)", "(llmsvd)"},
		},
		{
			name:    "all features enabled",
			modelID: "super-model",
			result: VerificationResult{
				ModelInfo: ModelInfo{
					Endpoint:       "https://api.openai.com/v1",
					SupportsBrotli: true,
					SupportsHTTP3:  true,
					SupportsToon:   true,
				},
				FeatureDetection: FeatureDetectionResult{
					SupportsBrotli: true,
					SupportsHTTP3:  true,
					SupportsToon:   true,
					Streaming:      true,
				},
			},
			isVerified:   true,
			wantContains: []string{"super-model", "(brotli)", "(http3)", "(toon)", "(streaming)", "(llmsvd)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatModelNameWithSuffixes(tt.modelID, tt.result, tt.isVerified)

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Model name should contain %s", want)
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, result, notWant, "Model name should not contain %s", notWant)
			}

			// Verify (llmsvd) is always last if present
			if tt.isVerified {
				assert.True(t, strings.HasSuffix(result, "(llmsvd)"), "(llmsvd) should be the last suffix")
			}
		})
	}
}

// TestDetectBrotliSupport tests brotli support detection
func TestDetectBrotliSupport(t *testing.T) {
	tests := []struct {
		endpoint string
		want     bool
	}{
		{"https://api.anthropic.com/v1", true},
		{"https://api.openai.com/v1", true},
		{"https://generativelanguage.googleapis.com/v1", true},
		{"https://api.deepseek.com/v1", true},
		{"https://api.mistral.ai/v1", true},
		{"https://api.cohere.com/v1", true},
		{"https://api.unknown.com/v1", false},
		{"https://api.groq.com/openai/v1", false},
		{"https://api.together.xyz/v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			got := detectBrotliSupport(tt.endpoint)
			assert.Equal(t, tt.want, got, "detectBrotliSupport(%s)", tt.endpoint)
		})
	}
}

// TestDetectHTTP3Support tests HTTP/3 support detection
func TestDetectHTTP3Support(t *testing.T) {
	tests := []struct {
		endpoint string
		want     bool
	}{
		{"https://api.cloudflare.com/v1", true},
		{"https://generativelanguage.googleapis.com/v1", true},
		{"https://api.fastly.com/v1", true},
		{"https://api.openai.com/v1", false},
		{"https://api.anthropic.com/v1", false},
		{"https://api.unknown.com/v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			got := detectHTTP3Support(tt.endpoint)
			assert.Equal(t, tt.want, got, "detectHTTP3Support(%s)", tt.endpoint)
		})
	}
}

// TestDetectToonSupport tests toon/creative style detection
func TestDetectToonSupport(t *testing.T) {
	tests := []struct {
		modelID string
		want    bool
	}{
		{"dall-e-3", true},
		{"stable-diffusion-xl", true},
		{"midjourney-v5", true},
		{"creative-model", true},
		{"art-generator", true},
		{"toon-style", true},
		{"gpt-4o", false},
		{"claude-3-opus", false},
		{"llama-3-70b", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got := detectToonSupport(tt.modelID)
			assert.Equal(t, tt.want, got, "detectToonSupport(%s)", tt.modelID)
		})
	}
}

// TestCrushModelFeatureFlags tests that Crush model has all feature flags
func TestCrushModelFeatureFlags(t *testing.T) {
	model := CrushModel{
		ID:                "test-model",
		Name:              "test-model (brotli) (http3) (streaming) (llmsvd)",
		ContextWindow:     128000,
		DefaultMaxTokens:  4096,
		CanReason:         true,
		SupportsAttachments: true,
		SupportsHTTP3:     true,
		SupportsToon:      true,
		SupportsBrotli:    true,
		SupportsStreaming: true,
		Verified:          true,
	}

	// Marshal and unmarshal to verify JSON tags
	data, err := json.Marshal(model)
	require.NoError(t, err)

	var unmarshalled map[string]interface{}
	err = json.Unmarshal(data, &unmarshalled)
	require.NoError(t, err)

	// Verify all expected fields are present
	expectedFields := []string{
		"id", "name", "context_window", "default_max_tokens",
		"can_reason", "supports_attachments", "supports_http3",
		"supports_toon", "supports_brotli", "supports_streaming", "verified",
	}

	for _, field := range expectedFields {
		_, exists := unmarshalled[field]
		assert.True(t, exists, "Field %s should be present in JSON", field)
	}
}

// TestExportCrushConfigWithFeatures tests Crush config export with feature detection
func TestExportCrushConfigWithFeatures(t *testing.T) {
	// Create test verification results with various features
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:             "gpt-4o",
				Endpoint:       "https://api.openai.com/v1",
				SupportsBrotli: true,
				ContextWindow:  ContextWindow{TotalMaxTokens: 128000},
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsBrotli: true,
				Streaming:      true,
				CodeGeneration: true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:   90.0,
				CodeCapability: 85.0,
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:            "gemini-pro",
				Endpoint:      "https://generativelanguage.googleapis.com/v1",
				SupportsHTTP3: true,
				ContextWindow: ContextWindow{TotalMaxTokens: 32000},
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsHTTP3: true,
				Streaming:     true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:   85.0,
				CodeCapability: 80.0,
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:           "dall-e-3",
				Endpoint:     "https://api.openai.com/v1",
				SupportsToon: true,
				ContextWindow: ContextWindow{TotalMaxTokens: 4000},
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsToon:   true,
				SupportsBrotli: true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:   80.0,
				CodeCapability: 30.0,
			},
		},
	}

	// Create temporary output file
	tmpFile, err := os.CreateTemp("", "crush_config_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Export config
	err = exportCrushConfig(results, tmpFile.Name(), nil)
	require.NoError(t, err)

	// Read and parse exported config
	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var config CrushConfig
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	// Verify providers and models
	assert.NotEmpty(t, config.Providers, "Config should have providers")

	// Check OpenAI provider models
	if openai, exists := config.Providers["openai"]; exists {
		for _, model := range openai.Models {
			if model.ID == "gpt-4o" {
				assert.True(t, model.SupportsBrotli, "gpt-4o should support brotli")
				assert.True(t, model.SupportsStreaming, "gpt-4o should support streaming")
				assert.Contains(t, model.Name, "(brotli)", "gpt-4o name should contain (brotli)")
				assert.Contains(t, model.Name, "(llmsvd)", "gpt-4o name should contain (llmsvd)")
			}
			if model.ID == "dall-e-3" {
				assert.True(t, model.SupportsToon, "dall-e-3 should support toon")
				assert.Contains(t, model.Name, "(toon)", "dall-e-3 name should contain (toon)")
			}
		}
	}

	// Check Google provider models
	if google, exists := config.Providers["gemini"]; exists {
		for _, model := range google.Models {
			if model.ID == "gemini-pro" {
				assert.True(t, model.SupportsHTTP3, "gemini-pro should support HTTP/3")
				assert.Contains(t, model.Name, "(http3)", "gemini-pro name should contain (http3)")
			}
		}
	}
}

// TestLLMSVDSuffixAlwaysLast tests that (llmsvd) suffix is always the last suffix
func TestLLMSVDSuffixAlwaysLast(t *testing.T) {
	testCases := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				Endpoint:       "https://api.openai.com/v1",
				SupportsBrotli: true,
				SupportsHTTP3:  true,
				SupportsToon:   true,
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsBrotli: true,
				SupportsHTTP3:  true,
				SupportsToon:   true,
				Streaming:      true,
			},
		},
		{
			ModelInfo: ModelInfo{
				Endpoint: "https://api.groq.com/openai/v1",
			},
			FeatureDetection: FeatureDetectionResult{
				Streaming: true,
			},
		},
		{
			ModelInfo: ModelInfo{
				Endpoint:       "https://generativelanguage.googleapis.com/v1",
				SupportsHTTP3:  true,
				SupportsBrotli: true,
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsHTTP3:  true,
				SupportsBrotli: true,
			},
		},
	}

	for i, result := range testCases {
		name := formatModelNameWithSuffixes("model", result, true)
		assert.True(t, strings.HasSuffix(name, "(llmsvd)"),
			"Test case %d: (llmsvd) should be the last suffix in '%s'", i, name)
	}
}

// TestVerificationResultFeaturePopulation tests that VerificationResult has feature fields populated
func TestVerificationResultFeaturePopulation(t *testing.T) {
	result := VerificationResult{
		ModelInfo: ModelInfo{
			ID:             "test-model",
			Endpoint:       "https://api.openai.com/v1",
			SupportsBrotli: true,
			SupportsHTTP3:  false,
			SupportsToon:   false,
		},
		FeatureDetection: FeatureDetectionResult{
			SupportsBrotli: true,
			SupportsHTTP3:  false,
			SupportsToon:   false,
			Streaming:      true,
			CodeGeneration: true,
		},
		PerformanceScores: PerformanceScore{
			OverallScore:   85.0,
			CodeCapability: 80.0,
		},
		Timestamp: time.Now(),
	}

	// Test that all feature fields are accessible
	assert.True(t, result.ModelInfo.SupportsBrotli)
	assert.False(t, result.ModelInfo.SupportsHTTP3)
	assert.False(t, result.ModelInfo.SupportsToon)
	assert.True(t, result.FeatureDetection.SupportsBrotli)
	assert.True(t, result.FeatureDetection.Streaming)
	assert.True(t, result.FeatureDetection.CodeGeneration)
}

// TestFeatureSuffixOrder tests that suffixes are added in the correct order
func TestFeatureSuffixOrder(t *testing.T) {
	result := VerificationResult{
		ModelInfo: ModelInfo{
			Endpoint:       "https://api.openai.com/v1",
			SupportsBrotli: true,
			SupportsHTTP3:  true,
			SupportsToon:   true,
		},
		FeatureDetection: FeatureDetectionResult{
			SupportsBrotli: true,
			SupportsHTTP3:  true,
			SupportsToon:   true,
			Streaming:      true,
		},
	}

	name := formatModelNameWithSuffixes("test-model", result, true)

	// Find positions of each suffix
	brotliPos := strings.Index(name, "(brotli)")
	http3Pos := strings.Index(name, "(http3)")
	toonPos := strings.Index(name, "(toon)")
	streamingPos := strings.Index(name, "(streaming)")
	llmsvdPos := strings.Index(name, "(llmsvd)")

	// Verify (llmsvd) is last
	assert.Greater(t, llmsvdPos, brotliPos, "(llmsvd) should come after (brotli)")
	assert.Greater(t, llmsvdPos, http3Pos, "(llmsvd) should come after (http3)")
	assert.Greater(t, llmsvdPos, toonPos, "(llmsvd) should come after (toon)")
	assert.Greater(t, llmsvdPos, streamingPos, "(llmsvd) should come after (streaming)")
}

// TestProviderBasedFeatureDetection tests provider-based automatic feature detection
func TestProviderBasedFeatureDetection(t *testing.T) {
	providers := []struct {
		name     string
		endpoint string
		brotli   bool
		http3    bool
	}{
		{"anthropic", "https://api.anthropic.com/v1", true, false},
		{"openai", "https://api.openai.com/v1", true, false},
		{"google", "https://generativelanguage.googleapis.com/v1", true, true},
		{"deepseek", "https://api.deepseek.com/v1", true, false},
		{"cloudflare", "https://api.cloudflare.com/v1", false, true},
		{"groq", "https://api.groq.com/openai/v1", false, false},
		{"together", "https://api.together.xyz/v1", false, false},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			brotli := detectBrotliSupport(p.endpoint)
			http3 := detectHTTP3Support(p.endpoint)

			assert.Equal(t, p.brotli, brotli, "%s should have brotli=%v", p.name, p.brotli)
			assert.Equal(t, p.http3, http3, "%s should have http3=%v", p.name, p.http3)
		})
	}
}

// TestOpenCodeConfigFormat tests that OpenCode config has correct structure
func TestOpenCodeConfigFormat(t *testing.T) {
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:            "gpt-4o",
				Endpoint:      "https://api.openai.com/v1",
				ContextWindow: ContextWindow{TotalMaxTokens: 128000},
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsBrotli: true,
				Streaming:      true,
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:            "claude-3-opus",
				Endpoint:      "https://api.anthropic.com/v1",
				ContextWindow: ContextWindow{TotalMaxTokens: 200000},
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsBrotli: true,
			},
		},
	}

	tmpFile, err := os.CreateTemp("", "opencode_config_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Export config
	err = exportOpenCodeConfigCorrect(results, tmpFile.Name(), nil)
	require.NoError(t, err)

	// Read and parse exported config
	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var config map[string]interface{}
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	// Verify OpenCode-specific structure
	assert.Contains(t, config, "$schema", "OpenCode config should have $schema")
	assert.Contains(t, config, "agents", "OpenCode config should have agents")
	assert.Contains(t, config, "providers", "OpenCode config should have providers")
	assert.Contains(t, config, "data", "OpenCode config should have data section")
	assert.Contains(t, config, "tui", "OpenCode config should have tui section")
	assert.Contains(t, config, "shell", "OpenCode config should have shell section")

	// Verify providers don't have models array (OpenCode format)
	providers := config["providers"].(map[string]interface{})
	for providerName, providerData := range providers {
		provider := providerData.(map[string]interface{})
		assert.Contains(t, provider, "apiKey", "Provider %s should have apiKey", providerName)
		assert.Contains(t, provider, "disabled", "Provider %s should have disabled", providerName)
		assert.Contains(t, provider, "provider", "Provider %s should have provider field", providerName)
		assert.NotContains(t, provider, "models", "OpenCode provider %s should NOT have models array", providerName)
	}

	// Verify agents have model references
	agents := config["agents"].(map[string]interface{})
	for agentName, agentData := range agents {
		agent := agentData.(map[string]interface{})
		assert.Contains(t, agent, "model", "Agent %s should have model", agentName)
		assert.Contains(t, agent, "maxTokens", "Agent %s should have maxTokens", agentName)

		// Model reference should be in format "provider.model"
		modelRef := agent["model"].(string)
		assert.Contains(t, modelRef, ".", "Model reference should be in provider.model format")
	}
}

// TestCrushConfigFormat tests that Crush config has correct structure with models
func TestCrushConfigFormat(t *testing.T) {
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:             "gpt-4o",
				Endpoint:       "https://api.openai.com/v1",
				SupportsBrotli: true,
				ContextWindow:  ContextWindow{TotalMaxTokens: 128000},
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsBrotli: true,
				Streaming:      true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:   85.0,
				CodeCapability: 80.0,
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:            "gemini-pro",
				Endpoint:      "https://generativelanguage.googleapis.com/v1",
				SupportsHTTP3: true,
				ContextWindow: ContextWindow{TotalMaxTokens: 32000},
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsHTTP3: true,
				Streaming:     true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:   80.0,
				CodeCapability: 75.0,
			},
		},
	}

	tmpFile, err := os.CreateTemp("", "crush_config_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = exportCrushConfig(results, tmpFile.Name(), nil)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var config CrushConfig
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	// Verify Crush-specific structure
	assert.Contains(t, config.Schema, "crush", "Crush config should have crush schema")
	assert.NotEmpty(t, config.Providers, "Crush config should have providers")

	// Verify providers HAVE models array (Crush format)
	for providerName, provider := range config.Providers {
		assert.NotEmpty(t, provider.Name, "Provider %s should have name", providerName)
		assert.NotEmpty(t, provider.BaseURL, "Provider %s should have base_url", providerName)
		assert.NotNil(t, provider.Models, "Crush provider %s SHOULD have models array", providerName)

		// Verify models have required fields
		for _, model := range provider.Models {
			assert.NotEmpty(t, model.ID, "Model should have ID")
			assert.NotEmpty(t, model.Name, "Model should have name")
			assert.Contains(t, model.Name, "(llmsvd)", "Verified model name should contain (llmsvd)")
		}
	}
}

// TestOpenCodeProviderWithoutModels tests that OpenCode providers don't include models
func TestOpenCodeProviderWithoutModels(t *testing.T) {
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:       "test-model-1",
				Endpoint: "https://api.openai.com/v1",
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:       "test-model-2",
				Endpoint: "https://api.openai.com/v1",
			},
		},
	}

	tmpFile, err := os.CreateTemp("", "opencode_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = exportOpenCodeConfigCorrect(results, tmpFile.Name(), nil)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	// Check raw JSON doesn't contain "models" key in providers
	var rawConfig map[string]interface{}
	json.Unmarshal(data, &rawConfig)

	providers := rawConfig["providers"].(map[string]interface{})
	openaiProvider := providers["openai"].(map[string]interface{})

	_, hasModels := openaiProvider["models"]
	assert.False(t, hasModels, "OpenCode provider should NOT have models key")
}

// TestCrushProviderWithModels tests that Crush providers include models with features
func TestCrushProviderWithModels(t *testing.T) {
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:             "claude-3-opus",
				Endpoint:       "https://api.anthropic.com/v1",
				SupportsBrotli: true,
				ContextWindow:  ContextWindow{TotalMaxTokens: 200000},
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsBrotli: true,
				Streaming:      true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:   90.0,
				CodeCapability: 85.0,
			},
		},
	}

	tmpFile, err := os.CreateTemp("", "crush_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = exportCrushConfig(results, tmpFile.Name(), nil)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var config CrushConfig
	json.Unmarshal(data, &config)

	anthropic, exists := config.Providers["anthropic"]
	assert.True(t, exists, "Crush config should have anthropic provider")
	assert.NotEmpty(t, anthropic.Models, "Crush provider should have models")

	// Check model has feature flags
	model := anthropic.Models[0]
	assert.True(t, model.SupportsBrotli, "Model should have supports_brotli=true")
	assert.True(t, model.Verified, "Model should have verified=true")
	assert.Contains(t, model.Name, "(brotli)", "Model name should contain (brotli)")
	assert.Contains(t, model.Name, "(llmsvd)", "Model name should contain (llmsvd)")
}

// TestExtractProviderMapping tests provider extraction from endpoints
func TestExtractProviderMapping(t *testing.T) {
	tests := []struct {
		endpoint string
		expected string
	}{
		{"https://api.openai.com/v1", "openai"},
		{"https://api.anthropic.com/v1", "anthropic"},
		{"https://generativelanguage.googleapis.com/v1", "gemini"},
		{"https://api.groq.com/openai/v1", "groq"},
		{"https://api.deepseek.com/v1", "deepseek"},
		{"https://api.together.xyz/v1", "together"},
		{"https://api.fireworks.ai/inference/v1", "fireworks"},
		{"https://api.cloudflare.com/v1", "cloudflare"},
		{"https://api.mistral.ai/v1", "mistral"},
		{"https://api.cohere.com/v1", "cohere"},
		{"https://api.moonshot.cn/v1", "kimi"},
		{"https://integrate.api.nvidia.com/v1", "nvidia"},
		{"https://api.ai21.com/v1", "ai21"},
		{"https://api.stability.com/v1", "stability"},
		{"https://api.modal.com/v1", "modal"},
		{"https://api.sarvam.ai/v1", "sarvam"},
		{"https://api.nlpcloud.com/v1", "nlpcloud"},
		{"https://codestral.mistral.ai/v1", "codestral"},
		{"https://api.sambanova.ai/v1", "sambanova"},
		{"https://api.hyperbolic.xyz/v1", "hyperbolic"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := extractProvider(tt.endpoint)
			assert.Equal(t, tt.expected, result, "extractProvider(%s) should return %s", tt.endpoint, tt.expected)
		})
	}
}

// TestConfigExportWithAllFeatures tests export with all feature types
func TestConfigExportWithAllFeatures(t *testing.T) {
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:             "full-featured-model",
				Endpoint:       "https://generativelanguage.googleapis.com/v1",
				SupportsBrotli: true,
				SupportsHTTP3:  true,
				ContextWindow:  ContextWindow{TotalMaxTokens: 128000},
			},
			FeatureDetection: FeatureDetectionResult{
				SupportsBrotli: true,
				SupportsHTTP3:  true,
				Streaming:      true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:   85.0,
				CodeCapability: 80.0,
			},
		},
	}

	tmpFile, err := os.CreateTemp("", "features_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = exportCrushConfig(results, tmpFile.Name(), nil)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var config CrushConfig
	json.Unmarshal(data, &config)

	gemini := config.Providers["gemini"]
	require.NotNil(t, gemini)
	require.NotEmpty(t, gemini.Models)

	model := gemini.Models[0]

	// Verify all features detected
	assert.True(t, model.SupportsBrotli, "Should detect brotli support")
	assert.True(t, model.SupportsHTTP3, "Should detect http3 support")
	assert.True(t, model.SupportsStreaming, "Should detect streaming support")
	assert.True(t, model.Verified, "Should be verified")

	// Verify model name has all suffixes
	assert.Contains(t, model.Name, "(brotli)")
	assert.Contains(t, model.Name, "(http3)")
	assert.Contains(t, model.Name, "(streaming)")
	assert.Contains(t, model.Name, "(llmsvd)")

	// Verify (llmsvd) is last
	assert.True(t, strings.HasSuffix(model.Name, "(llmsvd)"), "(llmsvd) should be last suffix")
}

// TestFreeProviderDetection tests free provider detection
func TestFreeProviderDetection(t *testing.T) {
	freeProviders := []string{"chutes", "gemini", "nvidia", "huggingface", "siliconflow", "kimi"}
	paidProviders := []string{"openai", "anthropic", "deepseek", "groq"}

	for _, provider := range freeProviders {
		assert.True(t, isProviderFree(provider), "%s should be detected as free", provider)
	}

	for _, provider := range paidProviders {
		assert.False(t, isProviderFree(provider), "%s should NOT be detected as free", provider)
	}
}

// exportOpenCodeConfigCorrect is a helper that calls the correct OpenCode export function
func exportOpenCodeConfigCorrect(results []VerificationResult, outputPath string, options *ExportOptions) error {
	config, err := createCorrectOpenCodeConfig(results, options)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}
