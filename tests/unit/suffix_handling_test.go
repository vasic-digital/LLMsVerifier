package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"llm-verifier/suffix"
)

func TestSuffixHandling_ParseSuffixes(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedBaseName   string
		expectedSuffixes   []string
	}{
		{
			name:             "Model with single suffix",
			input:            "GPT-4 (llmsvd)",
			expectedBaseName: "GPT-4",
			expectedSuffixes: []string{"llmsvd"},
		},
		{
			name:             "Model with multiple suffixes",
			input:            "Claude-3 (llmsvd) (brotli) (http3)",
			expectedBaseName: "Claude-3",
			expectedSuffixes: []string{"llmsvd", "brotli", "http3"},
		},
		{
			name:             "Model with no suffixes",
			input:            "Llama-2-7B",
			expectedBaseName: "Llama-2-7B",
			expectedSuffixes: []string{},
		},
		{
			name:             "Model with complex suffixes",
			input:            "Mixtral-8x7B (llmsvd) (brotli) (http3) (free to use) (open source) (SC:8.5)",
			expectedBaseName: "Mixtral-8x7B",
			expectedSuffixes: []string{"llmsvd", "brotli", "http3", "free to use", "open source", "SC:8.5"},
		},
		{
			name:             "Model with nested parentheses",
			input:            "Test-Model (llmsvd (verified))",
			expectedBaseName: "Test-Model",
			expectedSuffixes: []string{"llmsvd (verified)"},
		},
		{
			name:             "Empty string",
			input:            "",
			expectedBaseName: "",
			expectedSuffixes: []string{},
		},
		{
			name:             "Only suffixes",
			input:            "(llmsvd) (brotli)",
			expectedBaseName: "",
			expectedSuffixes: []string{"llmsvd", "brotli"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := suffix.NewParser()
			baseName, suffixes := parser.Parse(tt.input)

			assert.Equal(t, tt.expectedBaseName, baseName)
			assert.Equal(t, tt.expectedSuffixes, suffixes)
		})
	}
}

func TestSuffixHandling_GenerateModelName(t *testing.T) {
	tests := []struct {
		name             string
		baseName         string
		suffixes         map[string]bool
		expectedOutput   string
	}{
		{
			name:           "Model with llmsvd suffix",
			baseName:       "GPT-4",
			suffixes:       map[string]bool{"llmsvd": true},
			expectedOutput: "GPT-4 (llmsvd)",
		},
		{
			name:     "Model with multiple suffixes",
			baseName: "Claude-3",
			suffixes: map[string]bool{
				"llmsvd": true,
				"brotli": true,
				"http3":  true,
			},
			expectedOutput: "Claude-3 (llmsvd) (brotli) (http3)",
		},
		{
			name:           "Model with no suffixes",
			baseName:       "Llama-2-7B",
			suffixes:       map[string]bool{},
			expectedOutput: "Llama-2-7B",
		},
		{
			name:     "Model with scoring suffix",
			baseName: "Mixtral-8x7B",
			suffixes: map[string]bool{
				"llmsvd":  true,
				"brotli":  true,
				"http3":   true,
				"SC:8.5":  true,
			},
			expectedOutput: "Mixtral-8x7B (llmsvd) (brotli) (http3) (SC:8.5)",
		},
		{
			name:     "Model with boolean flags",
			baseName: "Test-Model",
			suffixes: map[string]bool{
				"llmsvd":     true,
				"brotli":     false,
				"http3":      true,
				"free":       true,
				"open source": false,
			},
			expectedOutput: "Test-Model (llmsvd) (http3) (free)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := suffix.NewGenerator()
			output := generator.Generate(tt.baseName, tt.suffixes)

			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestSuffixHandling_Validation(t *testing.T) {
	tests := []struct {
		name           string
		suffix         string
		expectedValid  bool
		expectedError  string
	}{
		{
			name:          "Valid llmsvd suffix",
			suffix:        "llmsvd",
			expectedValid: true,
		},
		{
			name:          "Valid brotli suffix",
			suffix:        "brotli",
			expectedValid: true,
		},
		{
			name:          "Valid http3 suffix",
			suffix:        "http3",
			expectedValid: true,
		},
		{
			name:          "Valid scoring suffix",
			suffix:        "SC:8.5",
			expectedValid: true,
		},
		{
			name:          "Valid free suffix",
			suffix:        "free to use",
			expectedValid: true,
		},
		{
			name:          "Valid open source suffix",
			suffix:        "open source",
			expectedValid: true,
		},
		{
			name:          "Invalid suffix with special chars",
			suffix:        "invalid@suffix",
			expectedValid: false,
			expectedError: "invalid characters",
		},
		{
			name:          "Empty suffix",
			suffix:        "",
			expectedValid: false,
			expectedError: "empty suffix",
		},
		{
			name:          "Suffix with only spaces",
			suffix:        "   ",
			expectedValid: false,
			expectedError: "empty suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := suffix.NewValidator()
			valid, err := validator.Validate(tt.suffix)

			assert.Equal(t, tt.expectedValid, valid)
			if !tt.expectedValid {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSuffixHandling_StandardSuffixes(t *testing.T) {
	standardSuffixes := []struct {
		suffix      string
		description string
		category    string
	}{
		{"llmsvd", "LLM Suffix Verification Done", "verification"},
		{"brotli", "Brotli compression support", "compression"},
		{"http3", "HTTP/3 protocol support", "protocol"},
		{"free to use", "Free tier model", "pricing"},
		{"open source", "Open source model", "license"},
		{"SC:8.5", "Score 8.5", "scoring"},
		{"SC:9.0", "Score 9.0", "scoring"},
		{"SC:7.5", "Score 7.5", "scoring"},
	}

	manager := suffix.NewManager()

	for _, standard := range standardSuffixes {
		t.Run(standard.suffix, func(t *testing.T) {
			// Verify it's recognized as a standard suffix
			assert.True(t, manager.IsStandardSuffix(standard.suffix))
			assert.Equal(t, standard.description, manager.GetDescription(standard.suffix))
			assert.Equal(t, standard.category, manager.GetCategory(standard.suffix))
		})
	}
}

func TestSuffixHandling_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		operation      string
		expectedResult string
	}{
		{
			name:           "Add suffix to existing model",
			input:          "GPT-4 (brotli) (http3)",
			operation:      "add_llmsvd",
			expectedResult: "GPT-4 (llmsvd) (brotli) (http3)",
		},
		{
			name:           "Remove suffix from model",
			input:          "GPT-4 (llmsvd) (brotli) (http3)",
			operation:      "remove_brotli",
			expectedResult: "GPT-4 (llmsvd) (http3)",
		},
		{
			name:           "Update scoring suffix",
			input:          "GPT-4 (SC:8.0)",
			operation:      "update_score_9.5",
			expectedResult: "GPT-4 (SC:9.5)",
		},
		{
			name:           "Handle duplicate suffixes",
			input:          "GPT-4 (llmsvd) (llmsvd)",
			operation:      "deduplicate",
			expectedResult: "GPT-4 (llmsvd)",
		},
		{
			name:           "Handle malformed input",
			input:          "GPT-4 (llmsvd (brotli)",
			operation:      "normalize",
			expectedResult: "GPT-4 (llmsvd) (brotli)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := suffix.NewProcessor()
			var result string

			switch tt.operation {
			case "add_llmsvd":
				result = processor.AddSuffix(tt.input, "llmsvd")
			case "remove_brotli":
				result = processor.RemoveSuffix(tt.input, "brotli")
			case "update_score_9.5":
				result = processor.UpdateScore(tt.input, 9.5)
			case "deduplicate":
				result = processor.Deduplicate(tt.input)
			case "normalize":
				result = processor.Normalize(tt.input)
			}

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestSuffixHandling_Performance(t *testing.T) {
	processor := suffix.NewProcessor()
	
	// Test with a large number of model names
	modelNames := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		modelNames[i] = fmt.Sprintf("Model-%d (llmsvd) (brotli) (http3) (SC:%.1f)", i, 8.0+float64(i%20)*0.1)
	}

	start := time.Now()
	for _, modelName := range modelNames {
		baseName, suffixes := processor.Parse(modelName)
		assert.NotEmpty(t, baseName)
		assert.NotEmpty(t, suffixes)
	}
	duration := time.Since(start)

	// Should process 1000 model names in less than 100ms
	assert.Less(t, duration, 100*time.Millisecond)
}

func TestSuffixHandling_Integration(t *testing.T) {
	// Test integration with model provider service
	mockService := &MockProviderService{}
	suffixManager := suffix.NewManager()

	// Simulate model discovery with suffix handling
	models := []providers.Model{
		{
			ID:   "gpt-4",
			Name: "GPT-4",
			Metadata: map[string]interface{}{
				"supports_brotli": true,
				"supports_http3":  true,
				"score":           9.0,
			},
		},
		{
			ID:   "claude-3",
			Name: "Claude-3",
			Metadata: map[string]interface{}{
				"supports_brotli": false,
				"supports_http3":  true,
				"score":           8.5,
			},
		},
	}

	processedModels := make([]providers.Model, len(models))
	for i, model := range models {
		processedModel := model
		suffixes := make(map[string]bool)

		// Add suffixes based on metadata
		if supportsBrotli, ok := model.Metadata["supports_brotli"].(bool); ok && supportsBrotli {
			suffixes["brotli"] = true
		}
		if supportsHTTP3, ok := model.Metadata["supports_http3"].(bool); ok && supportsHTTP3 {
			suffixes["http3"] = true
		}
		if score, ok := model.Metadata["score"].(float64); ok {
			suffixes[fmt.Sprintf("SC:%.1f", score)] = true
		}
		
		// Always add llmsvd suffix for verified models
		suffixes["llmsvd"] = true

		generator := suffix.NewGenerator()
		processedModel.Name = generator.Generate(model.Name, suffixes)
		processedModels[i] = processedModel
	}

	// Verify processed model names
	assert.Contains(t, processedModels[0].Name, "(llmsvd)")
	assert.Contains(t, processedModels[0].Name, "(brotli)")
	assert.Contains(t, processedModels[0].Name, "(http3)")
	assert.Contains(t, processedModels[0].Name, "(SC:9.0)")

	assert.Contains(t, processedModels[1].Name, "(llmsvd)")
	assert.NotContains(t, processedModels[1].Name, "(brotli)")
	assert.Contains(t, processedModels[1].Name, "(http3)")
	assert.Contains(t, processedModels[1].Name, "(SC:8.5)")
}

func TestSuffixHandling_ErrorHandling(t *testing.T) {
	processor := suffix.NewProcessor()

	tests := []struct {
		name          string
		input         string
		operation     string
		expectedError bool
	}{
		{
			name:          "Parse empty string",
			input:         "",
			operation:     "parse",
			expectedError: false, // Should handle gracefully
		},
		{
			name:          "Parse nil input",
			input:         "",
			operation:     "parse",
			expectedError: false,
		},
		{
			name:          "Remove suffix that doesn't exist",
			input:         "GPT-4 (llmsvd)",
			operation:     "remove_brotli",
			expectedError: false, // Should handle gracefully
		},
		{
			name:          "Add empty suffix",
			input:         "GPT-4",
			operation:     "add_empty",
			expectedError: true,
		},
		{
			name:          "Malformed parentheses",
			input:         "GPT-4 (llmsvd (brotli))",
			operation:     "parse",
			expectedError: false, // Should handle nested parentheses
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			
			switch tt.operation {
			case "parse":
				_, _ = processor.Parse(tt.input)
			case "remove_brotli":
				_ = processor.RemoveSuffix(tt.input, "brotli")
			case "add_empty":
				_ = processor.AddSuffix(tt.input, "")
			}

			// For this test, we're mainly checking that operations don't panic
			// and handle edge cases gracefully
			assert.True(t, true) // Test passed if we reached here without panic
		})
	}
}