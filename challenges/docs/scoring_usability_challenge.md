# Scoring and Usability Comprehensive Challenge

## Overview
This challenge validates the scoring system that evaluates model usability from 0-100% based on multiple criteria.

## Challenge Type
Functional Test + Scoring Algorithm Test + Validation Test

## Test Scenarios

### 1. Scoring Algorithm Validation Challenge
**Objective**: Verify scoring algorithm produces accurate 0-100% scores

**Steps**:
1. Provide model metrics (TTFT, success rate, feature support)
2. Calculate score
3. Verify score is within 0-100 range
4. Verify score distribution

**Expected Results**:
- Scores are in valid range
- Higher quality models score higher
- Consistency in scoring

**Test Code**:
```go
func TestScoringAlgorithm(t *testing.T) {
    scorer := NewModelScorer()

    testCases := []struct {
        metrics    ModelMetrics
        minScore   float64
        maxScore   float64
    }{
        {
            ModelMetrics{
                TTFT:           100 * time.Millisecond,
                SuccessRate:    1.0,
                FeatureSupport: 1.0,
                Reliability:    1.0,
            },
            90, 100,
        },
        {
            ModelMetrics{
                TTFT:           2000 * time.Millisecond,
                SuccessRate:    0.95,
                FeatureSupport: 0.8,
                Reliability:    0.85,
            },
            70, 85,
        },
    }

    for _, tc := range testCases {
        score := scorer.CalculateScore(tc.metrics)
        assert.GreaterOrEqual(t, score, tc.minScore)
        assert.LessOrEqual(t, score, tc.maxScore)
    }
}
```

---

### 2. Multi-Criteria Scoring Challenge
**Objective**: Verify scoring considers all criteria

**Criteria**:
- Strength (capability power)
- Speed (response time, TTFT)
- Reliability (success rate, uptime)
- Feature support (streaming, tools, vision, etc.)
- Cost-effectiveness

**Steps**:
1. Calculate individual scores per criterion
2. Combine scores with weights
3. Verify weighted average

**Expected Results**:
- All criteria contribute to score
- Weights are applied correctly
- Final score reflects all factors

**Test Code**:
```go
func TestMultiCriteriaScoring(t *testing.T) {
    scorer := NewModelScorer()

    criteria := CriteriaScores{
        Strength:       85,
        Speed:          70,
        Reliability:    90,
        FeatureSupport: 75,
        CostEffectiveness: 80,
    }

    score := scorer.CalculateMultiCriteriaScore(criteria)
    assert.InDelta(t, 79.5, score, 1.0) // Weighted average
}
```

---

### 3. Score Ranking Challenge
**Objective**: Verify models can be ranked by score

**Steps**:
1. Calculate scores for multiple models
2. Sort by score
3. Verify ranking order

**Expected Results**:
- Models are sorted correctly
- Ties are handled appropriately
- Ranking is stable

**Test Code**:
```go
func TestScoreRanking(t *testing.T) {
    scorer := NewModelScorer()

    models := []Model{
        {ID: "gpt-4", Score: 95},
        {ID: "gpt-3.5", Score: 80},
        {ID: "claude-3-opus", Score: 92},
    }

    ranked := scorer.RankModels(models)

    assert.Equal(t, "gpt-4", ranked[0].ID)
    assert.Equal(t, "claude-3-opus", ranked[1].ID)
    assert.Equal(t, "gpt-3.5", ranked[2].ID)
}
```

---

### 4. Score Calculation Edge Cases Challenge
**Objective**: Verify scoring handles edge cases

**Steps**:
1. Test with missing metrics
2. Test with extreme values
3. Test with partial data

**Expected Results**:
- Missing data is handled
- Extreme values don't break scoring
- Partial data produces reasonable scores

**Test Code**:
```go
func TestScoreEdgeCases(t *testing.T) {
    scorer := NewModelScorer()

    // Missing metrics
    score1 := scorer.CalculateScore(ModelMetrics{})
    assert.Greater(t, score1, 0.0)

    // Extreme values
    score2 := scorer.CalculateScore(ModelMetrics{
        TTFT:        0,
        SuccessRate: 0,
    })
    assert.GreaterOrEqual(t, score2, 0.0)
    assert.LessOrEqual(t, score2, 100.0)
}
```

---

### 5. Usability Classification Challenge
**Objective**: Verify usability classification based on score

**Categories**:
- 90-100: Excellent (production-ready, best-in-class)
- 80-89: Good (production-ready)
- 70-79: Acceptable (usable with limitations)
- 60-69: Fair (limited use cases)
- 0-59: Poor (not recommended)

**Steps**:
1. Calculate scores
2. Classify by score range
3. Verify categories

**Expected Results**:
- Classification matches score ranges
- Categories are accurate

**Test Code**:
```go
func TestUsabilityClassification(t *testing.T) {
    scorer := NewModelScorer()

    tests := []struct {
        score     float64
        category  string
    }{
        {95, "Excellent"},
        {82, "Good"},
        {75, "Acceptable"},
        {65, "Fair"},
        {50, "Poor"},
    }

    for _, tt := range tests {
        category := scorer.ClassifyUsability(tt.score)
        assert.Equal(t, tt.category, category)
    }
}
```

---

### 6. Score Trend Analysis Challenge
**Objective**: Verify score changes over time are tracked

**Steps**:
1. Calculate initial scores
2. Re-run verification later
3. Compare scores
4. Track trends

**Expected Results**:
- Score history is maintained
- Trends are calculated correctly
- Significant changes trigger events

**Test Code**:
```go
func TestScoreTrendAnalysis(t *testing.T) {
    scorer := NewModelScorer()

    // Initial score
    score1 := scorer.CalculateScore(ModelMetrics{TTFT: 100 * time.Millisecond})
    scorer.RecordScore("gpt-4", score1, time.Now())

    // New score after 24 hours
    score2 := scorer.CalculateScore(ModelMetrics{TTFT: 150 * time.Millisecond})
    scorer.RecordScore("gpt-4", score2, time.Now().Add(24*time.Hour))

    trend := scorer.GetScoreTrend("gpt-4")
    assert.Equal(t, -1, trend.Direction) // Score decreased
    assert.Equal(t, 24*time.Hour, trend.Duration)
}
```

---

### 7. Real-World Usability Score Challenge
**Objective**: Verify scores reflect real-world coding utility

**Steps**:
1. Run real coding tasks
2. Measure success rate
3. Calculate usability score
4. Verify correlation with coding tasks

**Expected Results**:
- High-scoring models perform well in coding tasks
- Score correlates with actual utility

**Test Code**:
```go
func TestRealWorldUsabilityScore(t *testing.T) {
    scorer := NewModelScorer()

    codingResults := []struct {
        model   string
        tasks   int
        success int
    }{
        {"gpt-4", 100, 95},
        {"gpt-3.5", 100, 80},
        {"claude-3-opus", 100, 92},
    }

    for _, result := range codingResults {
        score := scorer.CalculateUsabilityScore(result.tasks, result.success)
        assert.Greater(t, score, float64(result.success))
    }
}
```

---

### 8. Confidence Score Challenge
**Objective**: Verify confidence level in scores

**Steps**:
1. Calculate score
2. Calculate confidence based on data points
3. Verify confidence metrics

**Expected Results**:
- More data = higher confidence
- Less data = lower confidence
- Confidence is reported

**Test Code**:
```go
func TestConfidenceScore(t *testing.T) {
    scorer := NewModelScorer()

    // High confidence (many data points)
    score1, conf1 := scorer.CalculateScoreWithConfidence(ModelMetrics{}, 100)
    assert.Greater(t, conf1, 0.9)

    // Low confidence (few data points)
    score2, conf2 := scorer.CalculateScoreWithConfidence(ModelMetrics{}, 3)
    assert.Less(t, conf2, 0.7)
}
```

---

### 9. Score Aggregation Challenge
**Objective**: Verify scores can be aggregated across providers

**Steps**:
1. Calculate scores per provider
2. Aggregate across all providers
3. Calculate provider average
4. Calculate overall statistics

**Expected Results**:
- Aggregation works correctly
- Statistics are accurate

**Test Code**:
```go
func TestScoreAggregation(t *testing.T) {
    scorer := NewModelScorer()

    scores := []float64{90, 85, 92, 88, 95}

    avg := scorer.AverageScore(scores)
    assert.InDelta(t, 90.0, avg, 0.1)

    median := scorer.MedianScore(scores)
    assert.InDelta(t, 90.0, median, 0.1)
}
```

---

### 10. Score Report Generation Challenge
**Objective**: Verify scores are included in reports

**Steps**:
1. Generate Markdown report
2. Generate JSON report
3. Verify scores are present
4. Verify score breakdown

**Expected Results**:
- Scores are in all report formats
- Score breakdown is clear
- Ranking is shown

**Test Code**:
```go
func TestScoreReportGeneration(t *testing.T) {
    reporter := NewScoreReporter()
    scorer := NewModelScorer()

    scores := scorer.CalculateAllScores(testModels)

    mdReport := reporter.GenerateMarkdown(scores)
    assert.Contains(t, mdReport, "Usability Score")
    assert.Contains(t, mdReport, "95%")

    jsonReport := reporter.GenerateJSON(scores)
    assert.JSONEq(t, `{"gpt-4": {"score": 95, "category": "Excellent"}}`, jsonReport)
}
```

---

## Success Criteria

### Functional Requirements
- [ ] Scores are in valid 0-100 range
- [ ] Scoring algorithm is consistent
- [ ] All criteria are considered
- [ ] Edge cases handled correctly
- [ ] Classification is accurate
- [ ] Trends tracked correctly
- [ ] Real-world utility correlates
- [ ] Confidence scores calculated
- [ ] Aggregation works correctly
- [ ] Reports include scores

### Accuracy Requirements
- [ ] Scoring variance < 2% for same metrics
- [ ] Ranking accuracy > 99%
- [ ] Trend detection accuracy > 95%
- [ ] Real-world correlation > 0.8

### Performance Requirements
- [ ] Score calculation < 10ms
- [ ] 1000 model scoring < 1 second

## Dependencies
- Model metrics data
- Historical score data

## Cleanup
- No cleanup needed
