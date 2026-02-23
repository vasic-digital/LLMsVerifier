package selfimprove

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// rebuildIndexes (triggered via buffer overflow)
// ============================================================================

func TestInMemoryFeedbackCollector_RebuildIndexes_OnOverflow(t *testing.T) {
	// Create a collector with a small buffer (5 entries max)
	collector := NewInMemoryFeedbackCollector(nil, 5)
	ctx := context.Background()

	// Fill the buffer to capacity
	for i := 0; i < 5; i++ {
		feedback := &Feedback{
			SessionID: "session-1",
			PromptID:  "prompt-1",
			Type:      FeedbackTypePositive,
			Source:    FeedbackSourceHuman,
			Score:     0.8,
		}
		require.NoError(t, collector.Collect(ctx, feedback))
	}

	// This 6th entry triggers overflow → calls rebuildIndexes
	overflow := &Feedback{
		SessionID: "session-2",
		PromptID:  "prompt-2",
		Type:      FeedbackTypeNegative,
		Source:    FeedbackSourceAI,
		Score:     0.3,
	}
	require.NoError(t, collector.Collect(ctx, overflow))

	// After overflow, buffer was trimmed and re-indexed
	// The session-2 entry should be accessible
	results, err := collector.GetBySession(ctx, "session-2")
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

// ============================================================================
// GetByPrompt
// ============================================================================

func TestInMemoryFeedbackCollector_GetByPrompt(t *testing.T) {
	collector := NewInMemoryFeedbackCollector(nil, 100)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		feedback := &Feedback{
			PromptID: "prompt-abc",
			Type:     FeedbackTypePositive,
			Score:    0.8,
		}
		require.NoError(t, collector.Collect(ctx, feedback))
	}

	results, err := collector.GetByPrompt(ctx, "prompt-abc")
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestInMemoryFeedbackCollector_GetByPrompt_NotFound(t *testing.T) {
	collector := NewInMemoryFeedbackCollector(nil, 100)
	ctx := context.Background()

	results, err := collector.GetByPrompt(ctx, "nonexistent-prompt")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ============================================================================
// NewAutoFeedbackCollector + CollectAuto
// ============================================================================

type mockRewardModelForAuto struct {
	score float64
}

func (m *mockRewardModelForAuto) Score(ctx context.Context, prompt, response string) (float64, error) {
	return m.score, nil
}

func (m *mockRewardModelForAuto) Compare(ctx context.Context, prompt, responseA, responseB string) (*PreferencePair, error) {
	return &PreferencePair{Chosen: responseA, Rejected: responseB}, nil
}

func (m *mockRewardModelForAuto) ScoreWithDimensions(ctx context.Context, prompt, response string) (map[DimensionType]float64, error) {
	return map[DimensionType]float64{DimensionAccuracy: 0.8}, nil
}

func (m *mockRewardModelForAuto) Train(ctx context.Context, examples []*TrainingExample) error {
	return nil
}

func TestNewAutoFeedbackCollector(t *testing.T) {
	innerCollector := NewInMemoryFeedbackCollector(nil, 100)
	rewardModel := &mockRewardModelForAuto{score: 0.8}
	config := DefaultSelfImprovementConfig()

	afc := NewAutoFeedbackCollector(innerCollector, rewardModel, config, nil)
	assert.NotNil(t, afc)
}

func TestAutoFeedbackCollector_CollectAuto_Positive(t *testing.T) {
	innerCollector := NewInMemoryFeedbackCollector(nil, 100)
	rewardModel := &mockRewardModelForAuto{score: 0.85} // >= 0.7 → positive
	config := DefaultSelfImprovementConfig()
	afc := NewAutoFeedbackCollector(innerCollector, rewardModel, config, nil)

	ctx := context.Background()
	feedback, err := afc.CollectAuto(ctx, "session-1", "prompt-1", "What is 2+2?", "4", "claude", "claude-3")
	require.NoError(t, err)
	assert.NotNil(t, feedback)
	assert.Equal(t, FeedbackTypePositive, feedback.Type)
	assert.InDelta(t, 0.85, feedback.Score, 0.01)
}

func TestAutoFeedbackCollector_CollectAuto_Negative(t *testing.T) {
	innerCollector := NewInMemoryFeedbackCollector(nil, 100)
	rewardModel := &mockRewardModelForAuto{score: 0.2} // <= 0.3 → negative
	config := DefaultSelfImprovementConfig()
	afc := NewAutoFeedbackCollector(innerCollector, rewardModel, config, nil)

	ctx := context.Background()
	feedback, err := afc.CollectAuto(ctx, "s", "p", "Q", "A", "provider", "model")
	require.NoError(t, err)
	assert.Equal(t, FeedbackTypeNegative, feedback.Type)
}

func TestAutoFeedbackCollector_CollectAuto_Neutral(t *testing.T) {
	innerCollector := NewInMemoryFeedbackCollector(nil, 100)
	rewardModel := &mockRewardModelForAuto{score: 0.5} // between 0.3 and 0.7 → neutral
	config := DefaultSelfImprovementConfig()
	afc := NewAutoFeedbackCollector(innerCollector, rewardModel, config, nil)

	ctx := context.Background()
	feedback, err := afc.CollectAuto(ctx, "s", "p", "Q", "A", "provider", "model")
	require.NoError(t, err)
	assert.Equal(t, FeedbackTypeNeutral, feedback.Type)
}

// ============================================================================
// SelfImprovementSystem - uncovered methods
// ============================================================================

func TestSelfImprovementSystem_IsInitialized_False(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	assert.False(t, system.IsInitialized())
}

func TestSelfImprovementSystem_IsInitialized_True(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	require.NoError(t, system.Initialize(NewMockLLMProvider(), nil))
	assert.True(t, system.IsInitialized())
}

func TestSelfImprovementSystem_SetVerifier(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	// verifier can be nil — just make sure it doesn't panic
	system.SetVerifier(nil)
}

func TestSelfImprovementSystem_GetFeedbackCollector(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	assert.Nil(t, system.GetFeedbackCollector()) // not initialized yet

	require.NoError(t, system.Initialize(NewMockLLMProvider(), nil))
	assert.NotNil(t, system.GetFeedbackCollector())
}

func TestSelfImprovementSystem_GetPolicyOptimizer(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	assert.Nil(t, system.GetPolicyOptimizer())

	require.NoError(t, system.Initialize(NewMockLLMProvider(), nil))
	assert.NotNil(t, system.GetPolicyOptimizer())
}

func TestSelfImprovementSystem_OptimizePolicy_NoCollector(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	// Not initialized → no collector → returns nil, nil
	updates, err := system.OptimizePolicy(context.Background())
	require.NoError(t, err)
	assert.Nil(t, updates)
}

func TestSelfImprovementSystem_ApplyPolicyUpdate_NoOptimizer(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	err := system.ApplyPolicyUpdate(context.Background(), &PolicyUpdate{})
	require.NoError(t, err)
}

func TestSelfImprovementSystem_GetCurrentPolicy_NoOptimizer(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	policy := system.GetCurrentPolicy()
	assert.Empty(t, policy)
}

func TestSelfImprovementSystem_SetCurrentPolicy_NoOptimizer(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	// Should not panic
	system.SetCurrentPolicy("some policy")
}

func TestSelfImprovementSystem_GetFeedbackStats_NoCollector(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	stats, err := system.GetFeedbackStats(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestSelfImprovementSystem_GetFeedbackStats_WithData(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	require.NoError(t, system.Initialize(NewMockLLMProvider(), nil))

	ctx := context.Background()
	require.NoError(t, system.CollectFeedback(ctx, &Feedback{
		Type:  FeedbackTypePositive,
		Score: 0.9,
	}))

	stats, err := system.GetFeedbackStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalCount)
}

func TestSelfImprovementSystem_ScoreResponse_NoModel(t *testing.T) {
	system := NewSelfImprovementSystem(nil, nil)
	// Not initialized → returns 0.5, nil
	score, err := system.ScoreResponse(context.Background(), "q", "a")
	require.NoError(t, err)
	assert.Equal(t, 0.5, score)
}

// ============================================================================
// LLMPolicyOptimizer - uncovered methods
// ============================================================================

func TestLLMPolicyOptimizer_parseUpdates_Valid(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)
	opt.SetCurrentPolicy("Be helpful.")

	response := `[{"type": "guideline_addition", "change": "Be concise", "reason": "Users prefer short", "improvement_score": 0.8}]`
	updates, err := opt.parseUpdates(response)
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, PolicyUpdateGuidelineAddition, updates[0].UpdateType)
	assert.Equal(t, "Be concise", updates[0].Change)
}

func TestLLMPolicyOptimizer_parseUpdates_PromptRefinement(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)

	response := `[{"type": "prompt_refinement", "change": "Add context", "reason": "Clarity", "improvement_score": 0.6}]`
	updates, err := opt.parseUpdates(response)
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, PolicyUpdatePromptRefinement, updates[0].UpdateType)
}

func TestLLMPolicyOptimizer_parseUpdates_ConstraintUpdate(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)

	response := `[{"type": "constraint_update", "change": "Limit length", "reason": "Verbosity", "improvement_score": 0.5}]`
	updates, err := opt.parseUpdates(response)
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, PolicyUpdateConstraintUpdate, updates[0].UpdateType)
}

func TestLLMPolicyOptimizer_parseUpdates_InvalidJSON(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)

	// Invalid JSON → returns nil, nil (not error)
	updates, err := opt.parseUpdates("not valid json")
	require.NoError(t, err)
	assert.Nil(t, updates)
}

func TestLLMPolicyOptimizer_applyChange(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)
	opt.SetCurrentPolicy("Base policy")

	update := &PolicyUpdate{Change: "New guideline"}
	newPolicy := opt.applyChange(update)
	assert.Contains(t, newPolicy, "Base policy")
	assert.Contains(t, newPolicy, "New guideline")
}

func TestLLMPolicyOptimizer_applySelfCritique_NoPrinciples(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	config.ConstitutionalPrinciples = nil
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)

	updates := []*PolicyUpdate{{Change: "test change"}}
	result := opt.applySelfCritique(context.Background(), updates)
	assert.Len(t, result, 1)
}

func TestLLMPolicyOptimizer_applySelfCritique_WithPrinciples(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	config.ConstitutionalPrinciples = []string{"no harm"}
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)

	updates := []*PolicyUpdate{
		{Change: "be helpful"},
		{Change: "cause harm"}, // containsViolation always returns false, so both pass
	}
	result := opt.applySelfCritique(context.Background(), updates)
	assert.Len(t, result, 2) // containsViolation is always false
}

func TestLLMPolicyOptimizer_Apply(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)
	opt.SetCurrentPolicy("Old policy")

	update := &PolicyUpdate{
		ID:        "update-1",
		NewPolicy: "New policy",
	}

	err := opt.Apply(context.Background(), update)
	require.NoError(t, err)
	assert.Equal(t, "New policy", opt.GetCurrentPolicy())
}

func TestLLMPolicyOptimizer_Rollback(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)
	opt.SetCurrentPolicy("Original policy")

	update := &PolicyUpdate{
		ID:        "update-rollback",
		OldPolicy: "Original policy",
		NewPolicy: "Updated policy",
	}

	require.NoError(t, opt.Apply(context.Background(), update))
	assert.Equal(t, "Updated policy", opt.GetCurrentPolicy())

	require.NoError(t, opt.Rollback(context.Background(), "update-rollback"))
	assert.Equal(t, "Original policy", opt.GetCurrentPolicy())
}

func TestLLMPolicyOptimizer_Rollback_NotFound(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)

	err := opt.Rollback(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestLLMPolicyOptimizer_GetHistory(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)

	for i := 0; i < 3; i++ {
		update := &PolicyUpdate{
			ID:        "update",
			NewPolicy: "policy",
		}
		require.NoError(t, opt.Apply(context.Background(), update))
	}

	history, err := opt.GetHistory(context.Background(), 2)
	require.NoError(t, err)
	assert.Len(t, history, 2)
}

func TestLLMPolicyOptimizer_GetHistory_All(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	opt := NewLLMPolicyOptimizer(provider, nil, config, nil)

	for i := 0; i < 3; i++ {
		update := &PolicyUpdate{ID: "u", NewPolicy: "p"}
		require.NoError(t, opt.Apply(context.Background(), update))
	}

	history, err := opt.GetHistory(context.Background(), 0) // 0 means all
	require.NoError(t, err)
	assert.Len(t, history, 3)
}

// ============================================================================
// FeedbackFilter with time ranges
// ============================================================================

func TestFeedbackFilter_WithTimeRange(t *testing.T) {
	collector := NewInMemoryFeedbackCollector(nil, 100)
	ctx := context.Background()

	require.NoError(t, collector.Collect(ctx, &Feedback{
		Type:  FeedbackTypePositive,
		Score: 0.8,
	}))

	agg, err := collector.GetAggregated(ctx, &FeedbackFilter{})
	require.NoError(t, err)
	assert.Equal(t, 1, agg.TotalCount)
}

// ============================================================================
// AIRewardModel - ScoreWithDimensions and parseDimensionsFromJSON
// ============================================================================

func TestAIRewardModel_ScoreWithDimensions(t *testing.T) {
	provider := NewMockLLMProvider()
	// Override "score" so parseDimensionsFromJSON can unmarshal all-float64 JSON
	provider.responses["score"] = `{"accuracy": 0.9, "relevance": 0.8, "coherence": 0.7}`
	config := DefaultSelfImprovementConfig()
	config.UseDebateForReward = false

	model := NewAIRewardModel(provider, nil, config, nil)

	dims, err := model.ScoreWithDimensions(context.Background(), "What is 2+2?", "4")
	require.NoError(t, err)
	assert.NotNil(t, dims)
	assert.Contains(t, dims, DimensionAccuracy)
}
