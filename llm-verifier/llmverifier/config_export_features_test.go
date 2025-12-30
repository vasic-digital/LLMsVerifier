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
