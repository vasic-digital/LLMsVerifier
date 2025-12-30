package llmverifier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-verifier/database"
	"llm-verifier/events"
)

func TestNewIssueDetector(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	eventManager := events.NewEventManager(ctx, 100, 2)
	detector := NewIssueDetector(db, eventManager)

	require.NotNil(t, detector)
	assert.NotNil(t, detector.db)
	assert.NotNil(t, detector.eventManager)
}

func TestIssueDetector_detectIssues_LowOverallScore(t *testing.T) {
	detector := &IssueDetector{}

	result := &VerificationResult{
		PerformanceScores: PerformanceScore{
			OverallScore:   20.0, // Below 30 threshold
			CodeCapability: 50.0,
			Reliability:    60.0,
			Responsiveness: 70.0,
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:         true,
			FunctionCalling: true,
		},
	}

	issues := detector.detectIssues(result)
	require.NotEmpty(t, issues)

	// Should have a critical performance issue
	found := false
	for _, issue := range issues {
		if issue.IssueType == "performance" && issue.Severity == "critical" {
			found = true
			assert.Equal(t, "Severely Underperforming Model", issue.Title)
			break
		}
	}
	assert.True(t, found, "Expected critical performance issue for low overall score")
}

func TestIssueDetector_detectIssues_LowCodeCapability(t *testing.T) {
	detector := &IssueDetector{}

	result := &VerificationResult{
		PerformanceScores: PerformanceScore{
			OverallScore:   50.0,
			CodeCapability: 30.0, // Below 40 threshold
			Reliability:    60.0,
			Responsiveness: 70.0,
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:         true,
			FunctionCalling: true,
		},
	}

	issues := detector.detectIssues(result)
	require.NotEmpty(t, issues)

	found := false
	for _, issue := range issues {
		if issue.IssueType == "capability" && issue.Title == "Poor Code Generation Capability" {
			found = true
			assert.Equal(t, "high", issue.Severity)
			assert.Contains(t, issue.AffectedFeatures, "code_generation")
			break
		}
	}
	assert.True(t, found, "Expected capability issue for low code capability")
}

func TestIssueDetector_detectIssues_LowReliability(t *testing.T) {
	detector := &IssueDetector{}

	result := &VerificationResult{
		PerformanceScores: PerformanceScore{
			OverallScore:   50.0,
			CodeCapability: 50.0,
			Reliability:    40.0, // Below 50 threshold
			Responsiveness: 70.0,
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:         true,
			FunctionCalling: true,
		},
	}

	issues := detector.detectIssues(result)
	require.NotEmpty(t, issues)

	found := false
	for _, issue := range issues {
		if issue.IssueType == "reliability" {
			found = true
			assert.Equal(t, "high", issue.Severity)
			assert.Equal(t, "Unreliable Model Responses", issue.Title)
			break
		}
	}
	assert.True(t, found, "Expected reliability issue for low reliability score")
}

func TestIssueDetector_detectIssues_SlowResponseTimes(t *testing.T) {
	detector := &IssueDetector{}

	result := &VerificationResult{
		PerformanceScores: PerformanceScore{
			OverallScore:   50.0,
			CodeCapability: 50.0,
			Reliability:    60.0,
			Responsiveness: 50.0, // Below 60 threshold
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:         true,
			FunctionCalling: true,
		},
	}

	issues := detector.detectIssues(result)
	require.NotEmpty(t, issues)

	found := false
	for _, issue := range issues {
		if issue.IssueType == "performance" && issue.Severity == "medium" {
			found = true
			assert.Equal(t, "Slow Response Times", issue.Title)
			break
		}
	}
	assert.True(t, found, "Expected performance issue for slow response times")
}

func TestIssueDetector_detectIssues_MissingToolUse(t *testing.T) {
	detector := &IssueDetector{}

	result := &VerificationResult{
		PerformanceScores: PerformanceScore{
			OverallScore:   50.0,
			CodeCapability: 50.0,
			Reliability:    60.0,
			Responsiveness: 70.0,
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:         false, // No tool use
			FunctionCalling: false, // No function calling
		},
	}

	issues := detector.detectIssues(result)
	require.NotEmpty(t, issues)

	found := false
	for _, issue := range issues {
		if issue.IssueType == "capability" && issue.Title == "Missing Tool Use Capabilities" {
			found = true
			assert.Equal(t, "medium", issue.Severity)
			assert.Contains(t, issue.AffectedFeatures, "tool_use")
			assert.Contains(t, issue.AffectedFeatures, "function_calling")
			break
		}
	}
	assert.True(t, found, "Expected capability issue for missing tool use")
}

func TestIssueDetector_detectIssues_NetworkIssues(t *testing.T) {
	detector := &IssueDetector{}

	testCases := []struct {
		name       string
		error      string
		shouldFind bool
	}{
		{"timeout error", "Request timeout after 30s", true},
		{"connection error", "Connection refused", true},
		{"normal error", "Invalid request", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &VerificationResult{
				PerformanceScores: PerformanceScore{
					OverallScore:   80.0,
					CodeCapability: 80.0,
					Reliability:    80.0,
					Responsiveness: 80.0,
				},
				FeatureDetection: FeatureDetectionResult{
					ToolUse:         true,
					FunctionCalling: true,
				},
				Error: tc.error,
			}

			issues := detector.detectIssues(result)

			found := false
			for _, issue := range issues {
				if issue.IssueType == "connectivity" {
					found = true
					break
				}
			}

			if tc.shouldFind {
				assert.True(t, found, "Expected connectivity issue for error: %s", tc.error)
			} else {
				assert.False(t, found, "Should not detect connectivity issue for error: %s", tc.error)
			}
		})
	}
}

func TestIssueDetector_detectIssues_AuthenticationIssues(t *testing.T) {
	detector := &IssueDetector{}

	testCases := []struct {
		name       string
		error      string
		shouldFind bool
	}{
		{"auth error", "Authentication failed", true},
		{"unauthorized error", "Unauthorized: invalid API key", true},
		{"normal error", "Model not found", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &VerificationResult{
				PerformanceScores: PerformanceScore{
					OverallScore:   80.0,
					CodeCapability: 80.0,
					Reliability:    80.0,
					Responsiveness: 80.0,
				},
				FeatureDetection: FeatureDetectionResult{
					ToolUse:         true,
					FunctionCalling: true,
				},
				Error: tc.error,
			}

			issues := detector.detectIssues(result)

			found := false
			for _, issue := range issues {
				if issue.IssueType == "authentication" {
					found = true
					assert.Equal(t, "critical", issue.Severity)
					break
				}
			}

			if tc.shouldFind {
				assert.True(t, found, "Expected authentication issue for error: %s", tc.error)
			} else {
				assert.False(t, found, "Should not detect authentication issue for error: %s", tc.error)
			}
		})
	}
}

func TestIssueDetector_detectIssues_MultipleIssues(t *testing.T) {
	detector := &IssueDetector{}

	// Create a result with multiple issues
	result := &VerificationResult{
		PerformanceScores: PerformanceScore{
			OverallScore:   20.0, // Critical
			CodeCapability: 30.0, // High
			Reliability:    40.0, // High
			Responsiveness: 50.0, // Medium
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:         false, // Medium
			FunctionCalling: false,
		},
		Error: "Connection timeout", // High
	}

	issues := detector.detectIssues(result)
	// Should have multiple issues
	assert.GreaterOrEqual(t, len(issues), 4)
}

func TestIssueDetector_detectIssues_NoIssues(t *testing.T) {
	detector := &IssueDetector{}

	// Create a healthy result
	result := &VerificationResult{
		PerformanceScores: PerformanceScore{
			OverallScore:   95.0,
			CodeCapability: 90.0,
			Reliability:    95.0,
			Responsiveness: 90.0,
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:         true,
			FunctionCalling: true,
		},
		Error: "",
	}

	issues := detector.detectIssues(result)
	assert.Empty(t, issues)
}

func TestIssueDetector_mapSeverityToEventSeverity(t *testing.T) {
	detector := &IssueDetector{}

	testCases := []struct {
		issueSeverity string
		expected      events.Severity
	}{
		{"critical", events.SeverityCritical},
		{"Critical", events.SeverityCritical},
		{"CRITICAL", events.SeverityCritical},
		{"high", events.SeverityError},
		{"High", events.SeverityError},
		{"medium", events.SeverityWarning},
		{"Medium", events.SeverityWarning},
		{"low", events.SeverityInfo},
		{"Low", events.SeverityInfo},
		{"unknown", events.SeverityInfo},
		{"", events.SeverityInfo},
	}

	for _, tc := range testCases {
		t.Run(tc.issueSeverity, func(t *testing.T) {
			result := detector.mapSeverityToEventSeverity(tc.issueSeverity)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIssueDetector_isDecliningTrend(t *testing.T) {
	detector := &IssueDetector{}

	t.Run("insufficient data", func(t *testing.T) {
		result := detector.isDecliningTrend([]float64{90.0, 85.0})
		assert.False(t, result)
	})

	t.Run("stable trend", func(t *testing.T) {
		result := detector.isDecliningTrend([]float64{90.0, 90.0, 90.0, 90.0, 90.0})
		assert.False(t, result)
	})

	t.Run("upward trend", func(t *testing.T) {
		result := detector.isDecliningTrend([]float64{80.0, 85.0, 90.0, 95.0, 100.0})
		assert.False(t, result)
	})

	t.Run("declining trend", func(t *testing.T) {
		result := detector.isDecliningTrend([]float64{100.0, 90.0, 80.0, 70.0, 60.0})
		assert.True(t, result)
	})

	t.Run("slight decline - not significant", func(t *testing.T) {
		result := detector.isDecliningTrend([]float64{90.0, 89.0, 88.5, 88.0, 87.5})
		assert.False(t, result) // Slope is not steep enough
	})
}

func TestIssueDetector_AnalyzeMultipleResults(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	eventManager := events.NewEventManager(ctx, 100, 2)
	detector := NewIssueDetector(db, eventManager)

	// Test with empty results
	err = detector.AnalyzeMultipleResults([]*VerificationResult{})
	assert.NoError(t, err)

	// Test with some results
	results := []*VerificationResult{
		{
			ModelInfo: ModelInfo{ID: "model-1"},
			PerformanceScores: PerformanceScore{
				OverallScore: 90.0,
			},
		},
		{
			ModelInfo: ModelInfo{ID: "model-1"},
			PerformanceScores: PerformanceScore{
				OverallScore: 85.0,
			},
		},
		{
			ModelInfo: ModelInfo{ID: "model-2"},
			PerformanceScores: PerformanceScore{
				OverallScore: 80.0,
			},
			Error: "Some error", // Will be skipped
		},
	}

	err = detector.AnalyzeMultipleResults(results)
	assert.NoError(t, err)
}

func TestIssueDetector_analyzeModelPatterns(t *testing.T) {
	detector := &IssueDetector{}

	t.Run("insufficient results", func(t *testing.T) {
		results := []*VerificationResult{
			{PerformanceScores: PerformanceScore{OverallScore: 90.0}},
			{PerformanceScores: PerformanceScore{OverallScore: 85.0}},
		}
		err := detector.analyzeModelPatterns("model-1", results)
		assert.NoError(t, err)
	})

	t.Run("sufficient results", func(t *testing.T) {
		results := []*VerificationResult{
			{PerformanceScores: PerformanceScore{OverallScore: 90.0}},
			{PerformanceScores: PerformanceScore{OverallScore: 85.0}},
			{PerformanceScores: PerformanceScore{OverallScore: 80.0}},
		}
		err := detector.analyzeModelPatterns("model-1", results)
		assert.NoError(t, err)
	})
}
