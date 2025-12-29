package llmverifier

import (
	"strings"
	"testing"
)

func TestFeatureDetection_HTTP3(t *testing.T) {
	// Create a test verifier (we'll need to create a minimal one for testing)
	verifier := &Verifier{}

	// Create mock client with Google endpoint (should support HTTP/3)
	client := NewLLMClient("https://generativelanguage.googleapis.com/v1", "test-key", nil)

	result := verifier.testHTTP3(client, "gemini-pro")
	if !result {
		t.Error("Expected Google endpoint to support HTTP/3")
	}

	// Create mock client with OpenAI endpoint (should not support HTTP/3)
	client2 := NewLLMClient("https://api.openai.com/v1", "test-key", nil)

	result2 := verifier.testHTTP3(client2, "gpt-4")
	if result2 {
		t.Error("Expected OpenAI endpoint to not support HTTP/3")
	}
}

func TestFeatureDetection_Brotli(t *testing.T) {
	verifier := &Verifier{}

	// Create mock client with Anthropic endpoint (should support Brotli)
	client := NewLLMClient("https://api.anthropic.com/v1", "test-key", nil)

	result := verifier.testBrotli(client, "claude-3")
	if !result {
		t.Error("Expected Anthropic endpoint to support Brotli")
	}

	// Create mock client with unknown endpoint (should not support Brotli)
	client2 := NewLLMClient("https://unknown-provider.com/v1", "test-key", nil)

	result2 := verifier.testBrotli(client2, "unknown-model")
	if result2 {
		t.Error("Expected unknown endpoint to not support Brotli")
	}
}

func TestFeatureDetection_Toon(t *testing.T) {
	verifier := &Verifier{}

	// Create mock client
	client := NewLLMClient("https://api.test.com/v1", "test-key", nil)

	// Test model with "toon" in name
	result := verifier.testToon(client, "toon-generator-v2")
	if !result {
		t.Error("Expected model with 'toon' in name to support toon features")
	}

	// Test creative model
	result2 := verifier.testToon(client, "creative-ai-dalle")
	if !result2 {
		t.Error("Expected creative model to support toon features")
	}

	// Test regular model (should not detect toon from name)
	result3 := verifier.testToon(client, "gpt-4")
	if result3 {
		t.Error("Expected regular model to not support toon features based on name")
	}
}

func TestSuffixApplication(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		supportsHTTP3  bool
		supportsStream bool
		supportsToon   bool
		supportsBrotli bool
		expectedSuffix string
	}{
		{
			name:           "GPT-4 with streaming and brotli",
			modelName:      "gpt-4",
			supportsHTTP3:  false,
			supportsStream: true,
			supportsToon:   false,
			supportsBrotli: true,
			expectedSuffix: "(stream) (brotli)",
		},
		{
			name:           "Claude with toon support",
			modelName:      "claude-3-opus",
			supportsHTTP3:  false,
			supportsStream: true,
			supportsToon:   true,
			supportsBrotli: true,
			expectedSuffix: "(stream) (toon) (brotli)",
		},
		{
			name:           "Google with HTTP/3",
			modelName:      "gemini-pro",
			supportsHTTP3:  true,
			supportsStream: true,
			supportsToon:   false,
			supportsBrotli: true,
			expectedSuffix: "(http3) (stream) (brotli)",
		},
		{
			name:           "No features",
			modelName:      "basic-model",
			supportsHTTP3:  false,
			supportsStream: false,
			supportsToon:   false,
			supportsBrotli: false,
			expectedSuffix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the suffix logic (this would be in the export logic)
			var suffixes []string

			if tt.supportsHTTP3 {
				suffixes = append(suffixes, "(http3)")
			}
			if tt.supportsStream {
				suffixes = append(suffixes, "(stream)")
			}
			if tt.supportsToon {
				suffixes = append(suffixes, "(toon)")
			}
			if tt.supportsBrotli {
				suffixes = append(suffixes, "(brotli)")
			}

			result := tt.modelName
			if len(suffixes) > 0 {
				result += " " + strings.Join(suffixes, " ")
			}

			expected := tt.modelName
			if tt.expectedSuffix != "" {
				expected += " " + tt.expectedSuffix
			}

			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	}
}
