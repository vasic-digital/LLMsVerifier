// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- WeightConfig Tests ---

func TestWeightConfig_Validate_SumsToOne(t *testing.T) {
	wc := WeightConfig{
		Responsiveness:       0.30,
		CodeCapability:       0.25,
		FeatureRichness:      0.25,
		Reliability:          0.20,
		VisionCapability:     0.00,
		InstructionFollowing: 0.00,
	}
	assert.NoError(t, wc.Validate())
}

func TestWeightConfig_Validate_SumsToOneWithNewDimensions(t *testing.T) {
	wc := WeightConfig{
		Responsiveness:       0.15,
		CodeCapability:       0.10,
		FeatureRichness:      0.15,
		Reliability:          0.20,
		VisionCapability:     0.25,
		InstructionFollowing: 0.15,
	}
	assert.NoError(t, wc.Validate())
}

func TestWeightConfig_Validate_FailsWhenNotSumToOne(t *testing.T) {
	wc := WeightConfig{
		Responsiveness: 0.50,
		CodeCapability: 0.30,
		// Sum = 0.80, not 1.0
	}
	assert.Error(t, wc.Validate())
}

func TestWeightConfig_Validate_FailsWhenExceedsOne(t *testing.T) {
	wc := WeightConfig{
		Responsiveness:  0.50,
		CodeCapability:  0.50,
		FeatureRichness: 0.10,
	}
	assert.Error(t, wc.Validate())
}

func TestWeightConfig_Validate_AllZeroFails(t *testing.T) {
	wc := WeightConfig{}
	assert.Error(t, wc.Validate())
}

func TestWeightConfig_Validate_NegativeWeightFails(t *testing.T) {
	wc := WeightConfig{
		Responsiveness: -0.5,
		CodeCapability: 1.5,
	}
	assert.Error(t, wc.Validate())
}

// --- StrategyScore Tests ---

func TestStrategyScore_ToPerformanceScore(t *testing.T) {
	ss := StrategyScore{
		Overall: 0.85,
		Breakdown: map[string]float64{
			"responsiveness":    0.90,
			"code_capability":   0.80,
			"feature_richness":  0.85,
			"reliability":       0.82,
			"value_proposition": 0.75,
		},
		Passed: true,
	}
	ps := ss.ToPerformanceScore()
	assert.Equal(t, 0.85, ps.OverallScore)
	assert.Equal(t, 0.90, ps.Responsiveness)
	assert.Equal(t, 0.80, ps.CodeCapability)
	assert.Equal(t, 0.85, ps.FeatureRichness)
	assert.Equal(t, 0.82, ps.Reliability)
	assert.Equal(t, 0.75, ps.ValueProposition)
}

func TestStrategyScore_ToPerformanceScore_MissingKeys(t *testing.T) {
	ss := StrategyScore{
		Overall:   0.50,
		Breakdown: map[string]float64{},
	}
	ps := ss.ToPerformanceScore()
	assert.Equal(t, 0.50, ps.OverallScore)
	assert.Equal(t, 0.0, ps.Responsiveness)
}

// --- Thresholds Tests ---

func TestThresholds_Check_AllPass(t *testing.T) {
	th := Thresholds{
		MinOverallScore:      0.6,
		MaxLatency:           10 * time.Second,
		MinContextWindow:     50000,
		RequiredCapabilities: []string{"vision"},
	}
	assert.True(t, th.Check(0.7, 5*time.Second, 100000, []string{"vision", "streaming"}))
}

func TestThresholds_Check_ScoreTooLow(t *testing.T) {
	th := Thresholds{MinOverallScore: 0.6}
	assert.False(t, th.Check(0.5, 0, 0, nil))
}

func TestThresholds_Check_LatencyTooHigh(t *testing.T) {
	th := Thresholds{MaxLatency: 5 * time.Second}
	assert.False(t, th.Check(1.0, 10*time.Second, 0, nil))
}

func TestThresholds_Check_ContextWindowTooSmall(t *testing.T) {
	th := Thresholds{MinContextWindow: 100000}
	assert.False(t, th.Check(1.0, 0, 50000, nil))
}

func TestThresholds_Check_MissingCapability(t *testing.T) {
	th := Thresholds{RequiredCapabilities: []string{"vision", "streaming"}}
	assert.False(t, th.Check(1.0, 0, 0, []string{"vision"}))
}

func TestThresholds_Check_ZeroLatencyThreshold_SkipsCheck(t *testing.T) {
	th := Thresholds{MaxLatency: 0}
	assert.True(t, th.Check(0.0, 999*time.Second, 0, nil))
}

func TestThresholds_Check_EmptyRequirements(t *testing.T) {
	th := Thresholds{}
	assert.True(t, th.Check(0.0, 0, 0, nil))
}

// --- TestCategory Constants ---

func TestTestCategory_Values(t *testing.T) {
	categories := []TestCategory{
		TestCategoryExistence,
		TestCategoryResponsiveness,
		TestCategoryLatency,
		TestCategoryFeature,
		TestCategoryCode,
		TestCategoryVision,
		TestCategoryInstruction,
		TestCategoryStability,
	}
	assert.Len(t, categories, 8)

	// Ensure uniqueness
	seen := make(map[TestCategory]bool)
	for _, c := range categories {
		assert.False(t, seen[c], "duplicate category: %s", c)
		seen[c] = true
	}
}

// --- TestResult ---

func TestTestResult_Fields(t *testing.T) {
	tr := TestResult{
		TestID:    "test-vision-1",
		Category:  TestCategoryVision,
		Passed:    true,
		Score:     0.95,
		Latency:   2 * time.Second,
		Error:     nil,
		Details:   map[string]any{"resolution": "1080p"},
		Timestamp: time.Now(),
	}
	assert.Equal(t, "test-vision-1", tr.TestID)
	assert.Equal(t, TestCategoryVision, tr.Category)
	assert.True(t, tr.Passed)
	assert.Equal(t, 0.95, tr.Score)
}

// --- ScoringStrategy Interface Compliance ---

func TestScoringStrategy_InterfaceExists(t *testing.T) {
	// Verify the interface can be used as a type
	var _ ScoringStrategy = (*mockStrategy)(nil)
}

// mockStrategy for interface compliance test
type mockStrategy struct{}

func (m *mockStrategy) Name() string                    { return "mock" }
func (m *mockStrategy) Description() string             { return "mock strategy" }
func (m *mockStrategy) WeightConfig() WeightConfig      { return WeightConfig{Responsiveness: 1.0} }
func (m *mockStrategy) CustomTests() []VerificationTest { return nil }
func (m *mockStrategy) ScoreModel(_ context.Context, _ ModelInfo, _ []TestResult) (StrategyScore, error) {
	return StrategyScore{Overall: 0.5, Passed: true}, nil
}
func (m *mockStrategy) FilterModels(models []ModelInfo) []ModelInfo { return models }
func (m *mockStrategy) MinimumThresholds() Thresholds               { return Thresholds{} }

// --- VerificationTest Interface Compliance ---

func TestVerificationTest_InterfaceExists(t *testing.T) {
	var _ VerificationTest = (*mockVerificationTest)(nil)
}

// --- DefaultStrategy Tests ---

func TestDefaultStrategy_Name(t *testing.T) {
	ds := NewDefaultStrategy()
	assert.Equal(t, "default", ds.Name())
}

func TestDefaultStrategy_Description(t *testing.T) {
	ds := NewDefaultStrategy()
	assert.NotEmpty(t, ds.Description())
}

func TestDefaultStrategy_WeightConfig_MatchesExistingWeights(t *testing.T) {
	ds := NewDefaultStrategy()
	wc := ds.WeightConfig()
	assert.NoError(t, wc.Validate())
	assert.Equal(t, 0.30, wc.Responsiveness)
	assert.Equal(t, 0.25, wc.CodeCapability)
	assert.Equal(t, 0.25, wc.FeatureRichness)
	assert.Equal(t, 0.20, wc.Reliability)
	assert.Equal(t, 0.0, wc.VisionCapability)
	assert.Equal(t, 0.0, wc.InstructionFollowing)
}

func TestDefaultStrategy_CustomTests_Empty(t *testing.T) {
	ds := NewDefaultStrategy()
	assert.Empty(t, ds.CustomTests())
}

func TestDefaultStrategy_FilterModels_PassesAll(t *testing.T) {
	ds := NewDefaultStrategy()
	models := []ModelInfo{{ID: "m1"}, {ID: "m2"}, {ID: "m3"}}
	filtered := ds.FilterModels(models)
	assert.Len(t, filtered, 3)
}

func TestDefaultStrategy_MinimumThresholds_Zero(t *testing.T) {
	ds := NewDefaultStrategy()
	th := ds.MinimumThresholds()
	assert.Equal(t, 0.0, th.MinOverallScore)
	assert.Equal(t, time.Duration(0), th.MaxLatency)
	assert.Empty(t, th.RequiredCapabilities)
}

func TestDefaultStrategy_ScoreModel_WithResults(t *testing.T) {
	ds := NewDefaultStrategy()
	model := ModelInfo{ID: "test-model"}
	results := []TestResult{
		{Category: TestCategoryResponsiveness, Score: 0.90, Passed: true},
		{Category: TestCategoryCode, Score: 0.80, Passed: true},
		{Category: TestCategoryFeature, Score: 0.85, Passed: true},
		{Category: TestCategoryExistence, Score: 0.82, Passed: true},
	}
	score, err := ds.ScoreModel(context.Background(), model, results)
	assert.NoError(t, err)
	assert.True(t, score.Overall > 0)
	assert.True(t, score.Passed)
	// Verify weighted calculation: 0.90*0.30 + 0.80*0.25 + 0.85*0.25 + 0.82*0.20
	expected := 0.90*0.30 + 0.80*0.25 + 0.85*0.25 + 0.82*0.20
	assert.InDelta(t, expected, score.Overall, 0.001)
}

func TestDefaultStrategy_ScoreModel_EmptyResults(t *testing.T) {
	ds := NewDefaultStrategy()
	score, err := ds.ScoreModel(context.Background(), ModelInfo{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, score.Overall)
	assert.True(t, score.Passed)
}

func TestDefaultStrategy_ScoreModel_MultipleSameCategory(t *testing.T) {
	ds := NewDefaultStrategy()
	results := []TestResult{
		{Category: TestCategoryResponsiveness, Score: 0.80},
		{Category: TestCategoryResponsiveness, Score: 1.00},
	}
	score, err := ds.ScoreModel(context.Background(), ModelInfo{}, results)
	assert.NoError(t, err)
	// Average of 0.80 and 1.00 = 0.90, weighted by 0.30 = 0.27
	assert.InDelta(t, 0.90*0.30, score.Overall, 0.001)
}

func TestDefaultStrategy_ScoreModel_BreakdownPopulated(t *testing.T) {
	ds := NewDefaultStrategy()
	results := []TestResult{
		{Category: TestCategoryResponsiveness, Score: 0.90},
		{Category: TestCategoryCode, Score: 0.80},
	}
	score, err := ds.ScoreModel(context.Background(), ModelInfo{}, results)
	assert.NoError(t, err)
	assert.Equal(t, 0.90, score.Breakdown["responsiveness"])
	assert.Equal(t, 0.80, score.Breakdown["code_capability"])
}

func TestDefaultStrategy_ImplementsScoringStrategy(t *testing.T) {
	var _ ScoringStrategy = NewDefaultStrategy()
}

type mockVerificationTest struct{}

func (m *mockVerificationTest) ID() string             { return "mock-test" }
func (m *mockVerificationTest) Name() string           { return "Mock Test" }
func (m *mockVerificationTest) Category() TestCategory { return TestCategoryVision }
func (m *mockVerificationTest) Run(_ context.Context, _ *LLMClient) (TestResult, error) {
	return TestResult{Passed: true}, nil
}
func (m *mockVerificationTest) Weight() float64 { return 1.0 }
func (m *mockVerificationTest) Required() bool  { return false }
