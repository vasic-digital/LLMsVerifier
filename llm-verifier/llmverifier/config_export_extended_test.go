package llmverifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== formatProviderNameWithSuffix Tests ====================

func TestFormatProviderNameWithSuffix_Extended(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		isVerified bool
		expected   string
	}{
		{
			name:       "Verified provider",
			provider:   "OpenAI",
			isVerified: true,
			expected:   "OpenAI (llmsvd)",
		},
		{
			name:       "Unverified provider",
			provider:   "OpenAI",
			isVerified: false,
			expected:   "OpenAI",
		},
		{
			name:       "Empty provider verified",
			provider:   "",
			isVerified: true,
			expected:   " (llmsvd)",
		},
		{
			name:       "Anthropic verified",
			provider:   "Anthropic",
			isVerified: true,
			expected:   "Anthropic (llmsvd)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatProviderNameWithSuffix(tt.provider, tt.isVerified)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== formatModelNameWithSuffixes Tests ====================

func TestFormatModelNameWithSuffixes_Extended(t *testing.T) {
	tests := []struct {
		name       string
		modelID    string
		result     VerificationResult
		isVerified bool
		shouldHave []string
	}{
		{
			name:    "Basic verified model",
			modelID: "gpt-4",
			result: VerificationResult{
				FeatureDetection: FeatureDetectionResult{},
				ModelInfo:        ModelInfo{},
			},
			isVerified: true,
			shouldHave: []string{"(llmsvd)"},
		},
		{
			name:    "Model with brotli support",
			modelID: "claude-3",
			result: VerificationResult{
				FeatureDetection: FeatureDetectionResult{
					SupportsBrotli: true,
				},
				ModelInfo: ModelInfo{},
			},
			isVerified: true,
			shouldHave: []string{"(brotli)", "(llmsvd)"},
		},
		{
			name:    "Model with streaming",
			modelID: "gpt-4-turbo",
			result: VerificationResult{
				FeatureDetection: FeatureDetectionResult{
					Streaming: true,
				},
				ModelInfo: ModelInfo{},
			},
			isVerified: true,
			shouldHave: []string{"(streaming)", "(llmsvd)"},
		},
		{
			name:    "Unverified model",
			modelID: "test-model",
			result: VerificationResult{
				FeatureDetection: FeatureDetectionResult{},
				ModelInfo:        ModelInfo{},
			},
			isVerified: false,
			shouldHave: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatModelNameWithSuffixes(tt.modelID, tt.result, tt.isVerified)
			for _, expected := range tt.shouldHave {
				assert.Contains(t, result, expected)
			}
		})
	}
}

// ==================== detectBrotliSupport Tests ====================

func TestDetectBrotliSupport_Extended(t *testing.T) {
	tests := []struct {
		endpoint string
		expected bool
	}{
		{"https://api.anthropic.com/v1", true},
		{"https://api.openai.com/v1", true},
		{"https://googleapis.com/v1", true},
		{"https://api.deepseek.com/v1", true},
		{"https://api.mistral.ai/v1", true},
		{"https://api.cohere.com/v1", true},
		{"https://api.groq.com/v1", false},
		{"https://api.unknown.com/v1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			result := detectBrotliSupport(tt.endpoint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== detectHTTP3Support Tests ====================

func TestDetectHTTP3Support_Extended(t *testing.T) {
	tests := []struct {
		endpoint string
		expected bool
	}{
		{"https://cloudflare.com/api", true},
		{"https://api.google.com/v1", true},
		{"https://fastly.com/api", true},
		{"https://api.openai.com/v1", false},
		{"https://api.anthropic.com/v1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			result := detectHTTP3Support(tt.endpoint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== detectToonSupport Tests ====================

func TestDetectToonSupport_Extended(t *testing.T) {
	tests := []struct {
		modelID  string
		expected bool
	}{
		{"toon-model", true},
		{"creative-writer", true},
		{"art-model", true},
		{"dalle-3", true},
		{"dall-e-3", true},
		{"stable-diffusion-xl", true},
		{"midjourney", true},
		{"imagen", true},
		{"gpt-4", false},
		{"claude-3", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			result := detectToonSupport(tt.modelID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== filterModels Tests ====================

func TestFilterModels_Extended(t *testing.T) {
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:       "gpt-4",
				Endpoint: "https://api.openai.com/v1",
			},
			PerformanceScores: PerformanceScore{
				OverallScore: 90.0,
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:       "claude-3",
				Endpoint: "https://api.anthropic.com/v1",
			},
			PerformanceScores: PerformanceScore{
				OverallScore: 85.0,
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:       "gpt-3.5",
				Endpoint: "https://api.openai.com/v1",
			},
			PerformanceScores: PerformanceScore{
				OverallScore: 70.0,
			},
		},
		{
			Error: "Connection failed",
			ModelInfo: ModelInfo{
				ID: "failed-model",
			},
		},
	}

	t.Run("Nil options returns all valid results", func(t *testing.T) {
		filtered := filterModels(results, nil)
		assert.Len(t, filtered, 4) // All results
	})

	t.Run("Filter by minimum score", func(t *testing.T) {
		options := &ExportOptions{
			MinScore: 80.0,
		}
		filtered := filterModels(results, options)
		assert.Len(t, filtered, 2) // gpt-4 and claude-3
	})

	t.Run("Filter by providers", func(t *testing.T) {
		options := &ExportOptions{
			Providers: []string{"openai"},
		}
		filtered := filterModels(results, options)
		assert.Len(t, filtered, 2) // gpt-4 and gpt-3.5
	})

	t.Run("Filter by specific models", func(t *testing.T) {
		options := &ExportOptions{
			Models: []string{"gpt-4"},
		}
		filtered := filterModels(results, options)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "gpt-4", filtered[0].ModelInfo.ID)
	})

	t.Run("Limit by max models", func(t *testing.T) {
		options := &ExportOptions{
			MaxModels: 2,
		}
		filtered := filterModels(results, options)
		assert.Len(t, filtered, 2)
	})

	t.Run("Top N models by score", func(t *testing.T) {
		options := &ExportOptions{
			Top: 1,
		}
		filtered := filterModels(results, options)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "gpt-4", filtered[0].ModelInfo.ID)
	})
}

// ==================== extractCapabilities Tests ====================

func TestExtractCapabilities_Extended(t *testing.T) {
	result := VerificationResult{
		FeatureDetection: FeatureDetectionResult{
			ToolUse:          true,
			FunctionCalling:  true,
			CodeGeneration:   true,
			CodeCompletion:   true,
			CodeReview:       true,
			CodeExplanation:  true,
			Embeddings:       true,
			Reranking:        true,
			ImageGeneration:  true,
			AudioGeneration:  true,
			VideoGeneration:  true,
			MCPs:             true,
			LSPs:             true,
			Multimodal:       true,
			Streaming:        true,
			JSONMode:         true,
			StructuredOutput: true,
			Reasoning:        true,
			ParallelToolUse:  true,
		},
	}

	capabilities := extractCapabilities(result)

	expectedCapabilities := []string{
		"tool_use",
		"function_calling",
		"code_generation",
		"code_completion",
		"code_review",
		"code_explanation",
		"embeddings",
		"reranking",
		"image_generation",
		"audio_generation",
		"video_generation",
		"mcps",
		"lsps",
		"multimodal",
		"streaming",
		"json_mode",
		"structured_output",
		"reasoning",
		"parallel_tool_use",
	}

	for _, cap := range expectedCapabilities {
		assert.Contains(t, capabilities, cap)
	}
}

func TestExtractCapabilities_Empty(t *testing.T) {
	result := VerificationResult{
		FeatureDetection: FeatureDetectionResult{},
	}

	capabilities := extractCapabilities(result)
	assert.Empty(t, capabilities)
}

// ==================== categorizeModel Tests ====================

func TestCategorizeModel_Extended(t *testing.T) {
	tests := []struct {
		name     string
		result   VerificationResult
		expected string
	}{
		{
			name: "Coding model from CodeCapabilities",
			result: VerificationResult{
				CodeCapabilities: CodeCapabilityResult{
					CodeGeneration: true,
				},
			},
			expected: "coding",
		},
		{
			name: "Coding model from FeatureDetection",
			result: VerificationResult{
				FeatureDetection: FeatureDetectionResult{
					CodeReview: true,
				},
			},
			expected: "coding",
		},
		{
			name: "Multimodal model",
			result: VerificationResult{
				FeatureDetection: FeatureDetectionResult{
					Multimodal: true,
				},
			},
			expected: "multimodal",
		},
		{
			name: "Reasoning model",
			result: VerificationResult{
				FeatureDetection: FeatureDetectionResult{
					Reasoning: true,
				},
				PerformanceScores: PerformanceScore{
					OverallScore: 85,
				},
			},
			expected: "reasoning",
		},
		{
			name: "Chat model",
			result: VerificationResult{
				FeatureDetection: FeatureDetectionResult{
					ToolUse:         true,
					FunctionCalling: true,
				},
			},
			expected: "chat",
		},
		{
			name: "Generative model",
			result: VerificationResult{
				GenerativeCapabilities: GenerativeCapabilityResult{
					CreativeWriting: true,
				},
			},
			expected: "generative",
		},
		{
			name: "Specialized model - embeddings",
			result: VerificationResult{
				FeatureDetection: FeatureDetectionResult{
					Embeddings: true,
				},
			},
			expected: "specialized",
		},
		{
			name: "General model",
			result: VerificationResult{},
			expected: "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeModel(tt.result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== extractProvider Tests ====================

func TestExtractProvider_Extended(t *testing.T) {
	tests := []struct {
		endpoint string
		expected string
	}{
		{"https://api.openai.com/v1", "openai"},
		{"https://api.anthropic.com/v1", "anthropic"},
		{"https://generativelanguage.googleapis.com/v1", "gemini"},
		{"https://api.groq.com/v1", "groq"},
		{"https://api.deepseek.com/v1", "deepseek"},
		{"https://api.mistral.ai/v1", "mistral"},
		{"https://api.cohere.com/v1", "cohere"},
		{"https://api.together.xyz/v1", "together"},
		{"https://api.fireworks.ai/v1", "fireworks"},
		{"https://openrouter.ai/api/v1", "openrouter"},
		{"https://api.perplexity.ai/v1", "perplexity"},
		{"https://localhost:8080/v1", "local"},
		{"https://api.unknown.com/v1", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			result := extractProvider(tt.endpoint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== getProviderBaseURL Tests ====================

func TestGetProviderBaseURL_Extended(t *testing.T) {
	tests := []struct {
		provider string
		models   []VerificationResult
		expected string
	}{
		{"openai", nil, "https://api.openai.com/v1"},
		{"anthropic", nil, "https://api.anthropic.com/v1"},
		{"deepseek", nil, "https://api.deepseek.com/v1"},
		{"google", nil, "https://generativelanguage.googleapis.com/v1"},
		{"unknown", nil, "https://api.example.com/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := getProviderBaseURL(tt.provider, tt.models)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetProviderBaseURL_WithModels(t *testing.T) {
	models := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:       "gpt-4",
				Endpoint: "https://api.openai.com/v1/chat/completions",
			},
		},
	}

	result := getProviderBaseURL("openai", models)
	assert.Equal(t, "https://api.openai.com/v1", result)
}

func TestGetProviderEndpoint_Extended(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"openai", "https://api.openai.com/v1"},
		{"anthropic", "https://api.anthropic.com/v1"},
		{"gemini", "https://generativelanguage.googleapis.com/v1"},
		{"groq", "https://api.groq.com/openai/v1"},
		{"together", "https://api.together.xyz/v1"},
		{"unknown", "https://api.unknown.com/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := getProviderEndpoint(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== getCrushProviderType Tests ====================

func TestGetCrushProviderType_Extended(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"openai", "openai"},
		{"anthropic", "anthropic"},
		{"deepseek", "openai-compat"},
		{"google", "openai-compat"},
		{"unknown", "openai-compat"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := getCrushProviderType(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== AIConfig Structure Tests ====================

func TestAIConfigStructure_Extended(t *testing.T) {
	config := AIConfig{
		Version:     "1.0.0",
		GeneratedBy: "LLM Verifier",
		Models: []AIModel{
			{
				ID:       "gpt-4",
				Name:     "GPT-4",
				Provider: "openai",
				Score:    90.0,
			},
		},
		Preferences: Preferences{
			PrimaryModel:    "gpt-4",
			StreamResponses: true,
			AutoSave:        true,
		},
		Metadata: Metadata{
			TotalModels:  1,
			AverageScore: 90.0,
		},
	}

	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, "LLM Verifier", config.GeneratedBy)
	assert.Len(t, config.Models, 1)
	assert.Equal(t, "gpt-4", config.Preferences.PrimaryModel)
	assert.True(t, config.Preferences.StreamResponses)
}

// ==================== ExportOptions Tests ====================

func TestExportOptionsDefaults_Extended(t *testing.T) {
	options := &ExportOptions{}

	// Verify default values
	assert.Equal(t, 0.0, options.MinScore)
	assert.Equal(t, 0, options.Top)
	assert.False(t, options.IncludeAPIKey)
	assert.Empty(t, options.Categories)
	assert.Empty(t, options.Providers)
}

func TestExportOptionsWithValues_Extended(t *testing.T) {
	options := &ExportOptions{
		MinScore:      80.0,
		Top:           10,
		IncludeAPIKey: true,
		Categories:    []string{"coding", "reasoning"},
		Providers:     []string{"openai", "anthropic"},
		MaxModels:     100,
		Models:        []string{"gpt-4"},
	}

	assert.Equal(t, 80.0, options.MinScore)
	assert.Equal(t, 10, options.Top)
	assert.True(t, options.IncludeAPIKey)
	assert.Len(t, options.Categories, 2)
	assert.Len(t, options.Providers, 2)
}

// ==================== FeatureSuffixes Tests ====================

func TestFeatureSuffixes_Extended(t *testing.T) {
	assert.Contains(t, FeatureSuffixes, "(brotli)")
	assert.Contains(t, FeatureSuffixes, "(http3)")
	assert.Contains(t, FeatureSuffixes, "(toon)")
	assert.Contains(t, FeatureSuffixes, "(streaming)")
	assert.Contains(t, FeatureSuffixes, "(free to use)")
	assert.Contains(t, FeatureSuffixes, "(open source)")
	assert.Contains(t, FeatureSuffixes, "(fast)")
	assert.Contains(t, FeatureSuffixes, "(llmsvd)")

	// Verify (llmsvd) is last
	assert.Equal(t, "(llmsvd)", FeatureSuffixes[len(FeatureSuffixes)-1])
}

// ==================== getExportCriteriaDescription Tests ====================

func TestGetExportCriteriaDescription_Extended(t *testing.T) {
	tests := []struct {
		options  *ExportOptions
		expected string
	}{
		{
			options:  nil,
			expected: "All models",
		},
		{
			options:  &ExportOptions{},
			expected: "All models",
		},
		{
			options: &ExportOptions{
				MinScore: 80.0,
			},
			expected: "min score 80.0",
		},
		{
			options: &ExportOptions{
				Top: 10,
			},
			expected: "top 10 models",
		},
	}

	for _, tt := range tests {
		result := getExportCriteriaDescription(tt.options)
		assert.Contains(t, result, tt.expected)
	}
}

// ==================== FilterModels by Category Tests ====================

func TestFilterModels_ByCategory(t *testing.T) {
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:       "gpt-4-coding",
				Endpoint: "https://api.openai.com/v1",
			},
			PerformanceScores: PerformanceScore{
				OverallScore: 90.0,
			},
			CodeCapabilities: CodeCapabilityResult{
				CodeGeneration: true,
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:       "claude-chat",
				Endpoint: "https://api.anthropic.com/v1",
			},
			PerformanceScores: PerformanceScore{
				OverallScore: 85.0,
			},
			FeatureDetection: FeatureDetectionResult{
				ToolUse:         true,
				FunctionCalling: true,
			},
		},
	}

	t.Run("Filter by coding category", func(t *testing.T) {
		options := &ExportOptions{
			Categories: []string{"coding"},
		}
		filtered := filterModels(results, options)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "gpt-4-coding", filtered[0].ModelInfo.ID)
	})

	t.Run("Filter by chat category", func(t *testing.T) {
		options := &ExportOptions{
			Categories: []string{"chat"},
		}
		filtered := filterModels(results, options)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "claude-chat", filtered[0].ModelInfo.ID)
	})
}
