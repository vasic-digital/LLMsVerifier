package scoring

import (
	"testing"
)

// TestNewModelDisplayName tests creation of ModelDisplayName
func TestNewModelDisplayName(t *testing.T) {
	md := NewModelDisplayName()
	if md == nil {
		t.Fatal("NewModelDisplayName() returned nil")
	}
	if md.naming == nil {
		t.Error("ModelDisplayName.naming should not be nil")
	}
}

// TestFormatWithFeatureSuffixes tests feature suffix formatting
func TestFormatWithFeatureSuffixes(t *testing.T) {
	md := NewModelDisplayName()
	
	tests := []struct {
		name        string
		modelName   string
		features    map[string]interface{}
		expected    string
	}{
		{
			name:      "brotli only",
			modelName: "GPT-4",
			features: map[string]interface{}{
				"supports_brotli": true,
			},
			expected: "GPT-4 (brotli)",
		},
		{
			name:      "multiple features",
			modelName: "Llama-2-70B",
			features: map[string]interface{}{
				"supports_brotli": true,
				"supports_http3":  true,
				"open_weights":    true,
			},
			expected: "Llama-2-70B (brotli) (http3) (open source)",
		},
		{
			name:      "free model",
			modelName: "Free-Model",
			features: map[string]interface{}{
				"cost": map[string]interface{}{
					"input":  0.0,
					"output": 0.0,
				},
			},
			expected: "Free-Model (free to use)",
		},
		{
			name:      "toon feature",
			modelName: "Toon-Model",
			features: map[string]interface{}{
				"supports_toon": true,
			},
			expected: "Toon-Model (toon)",
		},
		{
			name:      "no features",
			modelName: "Standard-Model",
			features:  map[string]interface{}{},
			expected:  "Standard-Model",
		},
		{
			name:      "fast model",
			modelName: "Fast-Model",
			features: map[string]interface{}{
				"response_time_ms": 500.0,
			},
			expected: "Fast-Model (fast)",
		},
		{
			name:      "all features",
			modelName: "Premium-Model",
			features: map[string]interface{}{
				"supports_brotli": true,
				"supports_http3":  true,
				"supports_toon":   true,
				"open_weights":    true,
				"response_time_ms": 300.0,
				"cost": map[string]interface{}{
					"input":  0.0,
					"output": 0.0,
				},
			},
			expected: "Premium-Model (brotli) (http3) (toon) (open source) (free to use) (fast)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := md.FormatWithFeatureSuffixes(tt.modelName, tt.features)
			if result != tt.expected {
				t.Errorf("FormatWithFeatureSuffixes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestRemoveFeatureSuffixes tests removing existing suffixes
func TestRemoveFeatureSuffixes(t *testing.T) {
	md := NewModelDisplayName()
	
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "GPT-4 (brotli)",
			expected: "GPT-4",
		},
		{
			input:    "Llama-2 (brotli) (http3) (toon)",
			expected: "Llama-2",
		},
		{
			input:    "Model (free to use) (open source)",
			expected: "Model",
		},
		{
			input:    "No Suffix Model",
			expected: "No Suffix Model",
		},
		{
			input:    "Model (fast)",
			expected: "Model",
		},
		{
			input:    "Complex Model (brotli) (http3) (fast) (free to use)",
			expected: "Complex Model",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := md.removeFeatureSuffixes(tt.input)
			if result != tt.expected {
				t.Errorf("removeFeatureSuffixes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestExtractFeatures tests feature extraction from model data
func TestExtractFeatures(t *testing.T) {
	md := NewModelDisplayName()
	
	tests := []struct {
		name     string
		modelData map[string]interface{}
		expected ModelFeatures
	}{
		{
			name: "brotli support",
			modelData: map[string]interface{}{
				"supports_brotli": true,
			},
			expected: ModelFeatures{
				supportsBrotli: true,
			},
		},
		{
			name: "multiple bool flags",
			modelData: map[string]interface{}{
				"supports_brotli": true,
				"supports_http3":  true,
				"supports_toon":   true,
				"open_weights":    true,
			},
			expected: ModelFeatures{
				supportsBrotli: true,
				supportsHTTP3:  true,
				supportsToon:   true,
				isOpenSource:   true,
			},
		},
		{
			name: "free model",
			modelData: map[string]interface{}{
				"cost": map[string]interface{}{
					"input":  0.0,
					"output": 0.0,
				},
			},
			expected: ModelFeatures{
				isFree:          true,
				costPer1MInput:  0.0,
				costPer1MOutput: 0.0,
			},
		},
		{
			name: "paid model",
			modelData: map[string]interface{}{
				"cost": map[string]interface{}{
					"input":  0.05,
					"output": 0.15,
				},
			},
			expected: ModelFeatures{
				isFree:          false,
				costPer1MInput:  0.05,
				costPer1MOutput: 0.15,
			},
		},
		{
			name: "fast model",
			modelData: map[string]interface{}{
				"response_time_ms": 500.0,
			},
			expected: ModelFeatures{
				responseTime: 500.0,
				isFast:       true,
			},
		},
		{
			name: "slow model",
			modelData: map[string]interface{}{
				"response_time_ms": 2000.0,
			},
			expected: ModelFeatures{
				responseTime: 2000.0,
				isFast:       false,
			},
		},
		{
			name: "empty data",
			modelData: map[string]interface{}{},
			expected: ModelFeatures{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := md.extractFeatures(tt.modelData)
			
			// Compare individual fields
			if result.supportsBrotli != tt.expected.supportsBrotli {
				t.Errorf("supportsBrotli = %v, want %v", result.supportsBrotli, tt.expected.supportsBrotli)
			}
			if result.supportsHTTP3 != tt.expected.supportsHTTP3 {
				t.Errorf("supportsHTTP3 = %v, want %v", result.supportsHTTP3, tt.expected.supportsHTTP3)
			}
			if result.supportsToon != tt.expected.supportsToon {
				t.Errorf("supportsToon = %v, want %v", result.supportsToon, tt.expected.supportsToon)
			}
			if result.isOpenSource != tt.expected.isOpenSource {
				t.Errorf("isOpenSource = %v, want %v", result.isOpenSource, tt.expected.isOpenSource)
			}
			if result.isFree != tt.expected.isFree {
				t.Errorf("isFree = %v, want %v", result.isFree, tt.expected.isFree)
			}
			if result.isFast != tt.expected.isFast {
				t.Errorf("isFast = %v, want %v", result.isFast, tt.expected.isFast)
			}
			if result.costPer1MInput != tt.expected.costPer1MInput {
				t.Errorf("costPer1MInput = %v, want %v", result.costPer1MInput, tt.expected.costPer1MInput)
			}
		})
	}
}

// TestGetAllFeatureSuffixes tests getting all suffixes
func TestGetAllFeatureSuffixes(t *testing.T) {
	suffixes := GetAllFeatureSuffixes()
	
	expectedSuffixes := []string{
		"(brotli)", "(http3)", "(toon)", "(free to use)", "(open source)",
		"(fast)", "(optimized)", "(premium)", "(experimental)", "(beta)",
		"(prod)", "(stable)", "(deprecated)", "(legacy)",
	}
	
	if len(suffixes) != len(expectedSuffixes) {
		t.Errorf("Expected %d suffixes, got %d", len(expectedSuffixes), len(suffixes))
	}
	
	for _, expected := range expectedSuffixes {
		found := false
		for _, actual := range suffixes {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected suffix: %s", expected)
		}
	}
}

// TestFormatModelNameWithScoreAndFeatures tests combined formatting
func TestFormatModelNameWithScoreAndFeatures(t *testing.T) {
	md := NewModelDisplayName()
	
	tests := []struct {
		name         string
		modelName    string
		score        float64
		features     map[string]interface{}
		includeScore bool
		expected     string
	}{
		{
			name:         "score and features",
			modelName:    "GPT-4",
			score:        8.5,
			features:     map[string]interface{}{"supports_brotli": true},
			includeScore: true,
			expected:     "GPT-4 (brotli) (SC:8.5)",
		},
		{
			name:         "features only",
			modelName:    "Llama-2",
			score:        7.2,
			features:     map[string]interface{}{"open_weights": true},
			includeScore: false,
			expected:     "Llama-2 (open source)",
		},
		{
			name:         "score only",
			modelName:    "Standard-Model",
			score:        6.0,
			features:     map[string]interface{}{},
			includeScore: true,
			expected:     "Standard-Model (SC:6.0)",
		},
		{
			name:         "complex model",
			modelName:    "Premium-Model",
			score:        9.2,
			features: map[string]interface{}{
				"supports_brotli": true,
				"supports_http3":  true,
				"open_weights":    true,
				"response_time_ms": 200.0,
			},
			includeScore: true,
			expected:     "Premium-Model (brotli) (http3) (open source) (fast) (SC:9.2)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := md.FormatModelNameWithScoreAndFeatures(
				tt.modelName, tt.score, tt.features, tt.includeScore,
			)
			if result != tt.expected {
				t.Errorf("FormatModelNameWithScoreAndFeatures() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseFeatureSuffixes tests parsing suffixes from model names
func TestParseFeatureSuffixes(t *testing.T) {
	md := NewModelDisplayName()
	
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "Model (brotli) (http3)",
			expected: []string{"(brotli)", "(http3)"},
		},
		{
			input:    "Free Model (free to use) (open source)",
			expected: []string{"(free to use)", "(open source)"},
		},
		{
			input:    "Complex (brotli) (http3) (fast) (open source)",
			expected: []string{"(brotli)", "(http3)", "(fast)", "(open source)"},
		},
		{
			input:    "No Suffix Model",
			expected: []string{},
		},
		{
			input:    "Model (brotli) (SC:8.5)",
			expected: []string{"(brotli)"}, // Score suffix not included
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := md.ParseFeatureSuffixes(tt.input)
			
			if len(result) != len(tt.expected) {
				t.Errorf("ParseFeatureSuffixes() returned %d suffixes, want %d", len(result), len(tt.expected))
			}
			
			// Check all expected suffixes are present
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Missing expected suffix: %s in result %v", expected, result)
				}
			}
		})
	}
}

// TestHasFeatureSuffix tests checking for specific suffixes
func TestHasFeatureSuffix(t *testing.T) {
	md := NewModelDisplayName()
	
	modelName := "GPT-4 (brotli) (http3) (fast)"
	
	tests := []struct {
		suffix   string
		expected bool
	}{
		{"(brotli)", true},
		{"(http3)", true},
		{"(fast)", true},
		{"(toon)", false},
		{"(free to use)", false},
		{"(open source)", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.suffix, func(t *testing.T) {
			result := md.HasFeatureSuffix(modelName, tt.suffix)
			if result != tt.expected {
				t.Errorf("HasFeatureSuffix(%s) = %v, want %v", tt.suffix, result, tt.expected)
			}
		})
	}
}

// TestValidateFeatureSuffix tests suffix validation
func TestValidateFeatureSuffix(t *testing.T) {
	md := NewModelDisplayName()
	
	tests := []struct {
		suffix      string
		shouldValid bool
	}{
		{"(brotli)", true},
		{"(http3)", true},
		{"(toon)", true},
		{"(free to use)", true},
		{"(open source)", true},
		{"(fast)", true},
		{"(invalid)", false},
		{"(unknown)", false},
		{"brotli", false}, // Missing parentheses
		{"(BROTLI)", false}, // Wrong case
		{"", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.suffix, func(t *testing.T) {
			valid, suggestion := md.ValidateFeatureSuffix(tt.suffix)
			if valid != tt.shouldValid {
				t.Errorf("ValidateFeatureSuffix(%s) = %v, want %v", tt.suffix, valid, tt.shouldValid)
			}
			if !valid && suggestion == "" {
				t.Error("Expected suggestion when validation fails")
			}
		})
	}
}

// TestComplexRealWorldScenario tests a complex real-world scenario
func TestComplexRealWorldScenario(t *testing.T) {
	md := NewModelDisplayName()
	
	// Simulate a real model from OpenCode config
	modelData := map[string]interface{}{
		"name":            "GPT-4 Turbo",
		"maxTokens":       128000,
		"supports_brotli": true,
		"supports_http3":  true,
		"supports_toon":   false,
		"open_weights":    false,
		"cost": map[string]interface{}{
			"input":  0.01,
			"output": 0.03,
		},
		"response_time_ms": 800.0,
	}
	
	// Format with all features and score
	result := md.FormatModelNameWithScoreAndFeatures(
		"GPT-4 Turbo",
		8.7,
		modelData,
		true,
	)
	
	expected := "GPT-4 Turbo (brotli) (http3) (fast) (SC:8.7)"
	
	if result != expected {
		t.Errorf("Complex real-world scenario failed\nGot:      %v\nExpected: %v", result, expected)
	}
	
	// Verify suffixes are present
	if !md.HasFeatureSuffix(result, "(brotli)") {
		t.Error("Missing (brotli) suffix")
	}
	if !md.HasFeatureSuffix(result, "(http3)") {
		t.Error("Missing (http3) suffix")
	}
	if !md.HasFeatureSuffix(result, "(fast)") {
		t.Error("Missing (fast) suffix")
	}
	if md.HasFeatureSuffix(result, "(free to use)") {
		t.Error("Should not have (free to use) suffix")
	}
}

// TestEmptyAndEdgeCases tests empty and edge cases
func TestEmptyAndEdgeCases(t *testing.T) {
	md := NewModelDisplayName()
	
	// Test with empty model name
	result := md.FormatWithFeatureSuffixes("", map[string]interface{}{})
	if result != "" {
		t.Errorf("Empty model name should return empty, got: %v", result)
	}
	
	// Test with nil features
	result = md.FormatWithFeatureSuffixes("Model", nil)
	if result != "Model" {
		t.Errorf("Nil features should return clean name, got: %v", result)
	}
	
	// Test with only cost.output = 0 (not free)
	modelData := map[string]interface{}{
		"cost": map[string]interface{}{
			"input":  0.05,
			"output": 0.0,
		},
	}
	features := md.extractFeatures(modelData)
	if features.isFree {
		t.Error("Model with only output=0 should not be free");
	}
}