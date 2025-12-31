package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModelNaming(t *testing.T) {
	mn := NewModelNaming()

	require.NotNil(t, mn)
	assert.NotNil(t, mn.scoreSuffixPattern)
}

func TestModelNaming_GenerateScoreSuffix(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name     string
		score    float64
		expected string
	}{
		{"perfect score", 10.0, "(SC:10.0)"},
		{"high score", 9.5, "(SC:9.5)"},
		{"medium score", 7.3, "(SC:7.3)"},
		{"low score", 3.2, "(SC:3.2)"},
		{"zero score", 0.0, "(SC:0.0)"},
		{"rounded score", 8.55, "(SC:8.6)"},   // Rounds to 8.6
		{"rounded down", 8.54, "(SC:8.5)"},    // Rounds to 8.5
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suffix := mn.GenerateScoreSuffix(tc.score)
			assert.Equal(t, tc.expected, suffix)
		})
	}
}

func TestModelNaming_AddScoreSuffix(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name      string
		modelName string
		score     float64
		expected  string
	}{
		{"simple model name", "GPT-4", 8.5, "GPT-4 (SC:8.5)"},
		{"model with spaces", "Claude 3 Opus", 9.0, "Claude 3 Opus (SC:9.0)"},
		{"model with existing suffix", "GPT-4 (SC:7.0)", 8.5, "GPT-4 (SC:8.5)"},
		{"model with trailing space", "GPT-4 ", 8.5, "GPT-4 (SC:8.5)"},
		{"empty model name", "", 8.0, " (SC:8.0)"},  // Empty string trimmed + space + suffix
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mn.AddScoreSuffix(tc.modelName, tc.score)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestModelNaming_RemoveScoreSuffix(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name      string
		modelName string
		expected  string
	}{
		{"with suffix", "GPT-4 (SC:8.5)", "GPT-4"},
		{"without suffix", "GPT-4", "GPT-4"},
		{"with suffix and spaces", "Claude 3 Opus (SC:9.0)", "Claude 3 Opus"},
		{"multiple spaces before suffix", "GPT-4  (SC:8.5)", "GPT-4"},  // Trailing spaces removed
		{"integer score suffix", "Model (SC:9.0)", "Model"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mn.RemoveScoreSuffix(tc.modelName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestModelNaming_HasScoreSuffix(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name      string
		modelName string
		expected  bool
	}{
		{"with valid suffix", "GPT-4 (SC:8.5)", true},
		{"without suffix", "GPT-4", false},
		{"with malformed suffix", "GPT-4 (SC:abc)", false},
		{"with partial suffix", "GPT-4 (SC:", false},
		{"with similar but wrong format", "GPT-4 (8.5)", false},
		{"integer score", "GPT-4 (SC:9.0)", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mn.HasScoreSuffix(tc.modelName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestModelNaming_ExtractScoreFromName(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name          string
		modelName     string
		expectedScore float64
		expectedValid bool
	}{
		{"with valid suffix", "GPT-4 (SC:8.5)", 8.5, true},
		{"without suffix", "GPT-4", 0, false},
		{"high score", "Claude (SC:9.9)", 9.9, true},
		{"low score", "Model (SC:1.2)", 1.2, true},
		{"zero score", "Model (SC:0.0)", 0.0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, valid := mn.ExtractScoreFromName(tc.modelName)
			assert.Equal(t, tc.expectedValid, valid)
			if valid {
				assert.InDelta(t, tc.expectedScore, score, 0.01)
			}
		})
	}
}

func TestModelNaming_UpdateModelNameWithScore(t *testing.T) {
	mn := NewModelNaming()

	// Test that update is same as add
	result1 := mn.UpdateModelNameWithScore("GPT-4", 8.5)
	result2 := mn.AddScoreSuffix("GPT-4", 8.5)
	assert.Equal(t, result1, result2)

	// Test updating existing suffix
	result := mn.UpdateModelNameWithScore("GPT-4 (SC:7.0)", 9.0)
	assert.Equal(t, "GPT-4 (SC:9.0)", result)
}

func TestModelNaming_ValidateScoreSuffix(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name     string
		suffix   string
		expected bool
	}{
		{"valid suffix", "(SC:8.5)", true},
		{"valid with spaces", " (SC:8.5) ", true},
		{"invalid format", "(8.5)", false},
		{"missing SC", "(8.5)", false},
		{"missing colon", "(SC8.5)", false},
		{"missing parentheses", "SC:8.5", false},
		{"empty string", "", false},
		{"integer only", "(SC:9)", false}, // Requires decimal
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mn.ValidateScoreSuffix(tc.suffix)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestModelNaming_GetScoreSuffixFormat(t *testing.T) {
	mn := NewModelNaming()

	format := mn.GetScoreSuffixFormat()
	assert.Contains(t, format, "(SC:X.X)")
	assert.Contains(t, format, "0.0")
	assert.Contains(t, format, "10.0")
}

func TestModelNaming_BatchUpdateModelNames(t *testing.T) {
	mn := NewModelNaming()

	modelScores := map[string]float64{
		"GPT-4":         8.5,
		"Claude 3":      9.0,
		"Gemini Pro":    7.8,
		"Llama 2 (SC:5.0)": 8.2, // Should update existing
	}

	results := mn.BatchUpdateModelNames(modelScores)

	assert.Len(t, results, 4)
	assert.Equal(t, "GPT-4 (SC:8.5)", results["GPT-4"])
	assert.Equal(t, "Claude 3 (SC:9.0)", results["Claude 3"])
	assert.Equal(t, "Gemini Pro (SC:7.8)", results["Gemini Pro"])
	assert.Equal(t, "Llama 2 (SC:8.2)", results["Llama 2 (SC:5.0)"])
}

func TestModelNaming_NormalizeModelName(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"extra spaces", "GPT-4   Turbo", "GPT-4 Turbo"},
		{"leading spaces", "  GPT-4", "GPT-4"},
		{"trailing spaces", "GPT-4  ", "GPT-4"},
		{"newlines", "GPT-4\nTurbo", "GPT-4 Turbo"},
		{"tabs", "GPT-4\tTurbo", "GPT-4 Turbo"},
		{"normal name", "GPT-4", "GPT-4"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mn.NormalizeModelName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestModelNaming_CompareModelNames(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name     string
		name1    string
		name2    string
		expected bool
	}{
		{"same names", "GPT-4", "GPT-4", true},
		{"different scores", "GPT-4 (SC:8.5)", "GPT-4 (SC:9.0)", true},
		{"one with score", "GPT-4", "GPT-4 (SC:8.5)", true},
		{"different case", "gpt-4", "GPT-4", true},
		{"different models", "GPT-4", "Claude 3", false},
		{"similar names different suffix", "GPT-4 Turbo (SC:8.5)", "GPT-4 Turbo (SC:7.0)", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mn.CompareModelNames(tc.name1, tc.name2)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestModelNaming_ExtractBaseName(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name      string
		modelName string
		expected  string
	}{
		{"with suffix", "GPT-4 (SC:8.5)", "GPT-4"},
		{"without suffix", "GPT-4", "GPT-4"},
		{"complex name with suffix", "Claude 3 Opus (SC:9.0)", "Claude 3 Opus"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mn.ExtractBaseName(tc.modelName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestModelNaming_GenerateScoreSuffixWithConfidence(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name       string
		score      float64
		confidence float64
		expected   string
	}{
		{"high confidence", 8.5, 0.95, "(SC:8.5)★"},
		{"medium confidence", 8.5, 0.75, "(SC:8.5)◆"},
		{"low confidence", 8.5, 0.55, "(SC:8.5)▲"},
		{"very low confidence", 8.5, 0.3, "(SC:8.5)?"},
		{"boundary 0.9", 8.5, 0.9, "(SC:8.5)★"},
		{"boundary 0.7", 8.5, 0.7, "(SC:8.5)◆"},
		{"boundary 0.5", 8.5, 0.5, "(SC:8.5)▲"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mn.GenerateScoreSuffixWithConfidence(tc.score, tc.confidence)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestModelNaming_ParseScoreSuffix(t *testing.T) {
	mn := NewModelNaming()

	testCases := []struct {
		name               string
		suffix             string
		expectedScore      float64
		expectedConfidence string
		expectedValid      bool
	}{
		{"with high confidence", "(SC:8.5)★", 8.5, "★", true},
		{"with medium confidence", "(SC:9.0)◆", 9.0, "◆", true},
		{"with low confidence", "(SC:7.5)▲", 7.5, "▲", true},
		{"with very low confidence", "(SC:6.0)?", 6.0, "?", true},
		{"without confidence", "(SC:8.5)", 8.5, "", true},
		{"invalid suffix", "invalid", 0, "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, confidence, valid := mn.ParseScoreSuffix(tc.suffix)
			assert.Equal(t, tc.expectedValid, valid)
			if valid {
				assert.InDelta(t, tc.expectedScore, score, 0.01)
				assert.Equal(t, tc.expectedConfidence, confidence)
			}
		})
	}
}

// ==================== ScoreSuffixFormatter Tests ====================

func TestNewScoreSuffixFormatter(t *testing.T) {
	sf := NewScoreSuffixFormatter()

	require.NotNil(t, sf)
	assert.NotNil(t, sf.modelNaming)
}

func TestScoreSuffixFormatter_FormatWithColor(t *testing.T) {
	sf := NewScoreSuffixFormatter()

	testCases := []struct {
		name           string
		score          float64
		expectedPrefix string // ANSI color code
	}{
		{"excellent - green", 9.5, "\033[32m"},
		{"very good - yellow", 7.5, "\033[33m"},
		{"average - magenta", 5.5, "\033[35m"},
		{"poor - red", 3.0, "\033[31m"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sf.FormatWithColor(tc.score)
			assert.Contains(t, result, tc.expectedPrefix)
			assert.Contains(t, result, "\033[0m") // Reset code
		})
	}
}

func TestScoreSuffixFormatter_FormatWithDescription(t *testing.T) {
	sf := NewScoreSuffixFormatter()

	testCases := []struct {
		name                string
		score               float64
		expectedDescription string
	}{
		{"exceptional", 9.5, "Exceptional"},
		{"excellent", 8.5, "Excellent"},
		{"very good", 7.5, "Very Good"},
		{"good", 6.5, "Good"},
		{"average", 5.5, "Average"},
		{"below average", 4.5, "Below Average"},
		{"poor", 3.5, "Poor"},
		{"unacceptable", 2.5, "Unacceptable"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sf.FormatWithDescription(tc.score)
			assert.Contains(t, result, tc.expectedDescription)
			assert.Contains(t, result, "SC:")
		})
	}
}

func TestScoreSuffixFormatter_FormatForDisplay(t *testing.T) {
	sf := NewScoreSuffixFormatter()

	t.Run("with description", func(t *testing.T) {
		result := sf.FormatForDisplay(8.5, true)
		assert.Contains(t, result, "(SC:8.5)")
		assert.Contains(t, result, "Excellent")
	})

	t.Run("without description", func(t *testing.T) {
		result := sf.FormatForDisplay(8.5, false)
		assert.Equal(t, "(SC:8.5)", result)
	})
}
