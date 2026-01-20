package selfimprove

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMProvider for testing
type MockLLMProvider struct {
	responses map[string]string
}

func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		responses: map[string]string{
			"score":   `{"score": 0.8, "reasoning": "Good response"}`,
			"compare": `{"preferred": "A", "margin": 0.3, "reasoning": "A is better"}`,
		},
	}
}

func (m *MockLLMProvider) Complete(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	if containsAny(prompt, "compare", "Compare") {
		return m.responses["compare"], nil
	}
	return m.responses["score"], nil
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// MockDebateService for testing
type MockDebateService struct{}

func (m *MockDebateService) RunDebate(ctx context.Context, topic string, participants []string) (*DebateResult, error) {
	return &DebateResult{
		ID:         "debate-123",
		Consensus:  `{"score": 0.85, "reasoning": "Consensus reached"}`,
		Confidence: 0.85,
		Participants: map[string]string{
			"claude":   "Score: 0.8",
			"deepseek": "Score: 0.9",
		},
		Votes: map[string]float64{
			"claude":   0.8,
			"deepseek": 0.9,
		},
	}, nil
}

func TestAIRewardModel_Score(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	config.UseDebateForReward = false

	model := NewAIRewardModel(provider, nil, config, nil)

	score, err := model.Score(context.Background(), "What is 2+2?", "The answer is 4.")
	require.NoError(t, err)
	assert.InDelta(t, 0.8, score, 0.01)
}

func TestAIRewardModel_ScoreWithDebate(t *testing.T) {
	provider := NewMockLLMProvider()
	debateService := &MockDebateService{}
	config := DefaultSelfImprovementConfig()
	config.UseDebateForReward = true

	model := NewAIRewardModel(provider, debateService, config, nil)

	score, err := model.Score(context.Background(), "What is 2+2?", "The answer is 4.")
	require.NoError(t, err)
	assert.InDelta(t, 0.85, score, 0.01)
}

func TestAIRewardModel_Compare(t *testing.T) {
	provider := NewMockLLMProvider()
	config := DefaultSelfImprovementConfig()
	config.UseDebateForReward = false

	model := NewAIRewardModel(provider, nil, config, nil)

	pair, err := model.Compare(context.Background(), "What is 2+2?", "4", "5")
	require.NoError(t, err)
	assert.Equal(t, "4", pair.Chosen)
	assert.Equal(t, "5", pair.Rejected)
	assert.Greater(t, pair.ChosenScore, pair.RejectedScore)
}

func TestInMemoryFeedbackCollector_Collect(t *testing.T) {
	collector := NewInMemoryFeedbackCollector(nil, 100)

	feedback := &Feedback{
		SessionID:    "session-1",
		PromptID:     "prompt-1",
		Type:         FeedbackTypePositive,
		Source:       FeedbackSourceHuman,
		Score:        0.9,
		ProviderName: "claude",
	}

	err := collector.Collect(context.Background(), feedback)
	require.NoError(t, err)
	assert.NotEmpty(t, feedback.ID)

	results, err := collector.GetBySession(context.Background(), "session-1")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, feedback.ID, results[0].ID)
}

func TestInMemoryFeedbackCollector_GetAggregated(t *testing.T) {
	collector := NewInMemoryFeedbackCollector(nil, 100)

	for i := 0; i < 10; i++ {
		feedback := &Feedback{
			SessionID:    "session-1",
			Type:         FeedbackTypePositive,
			Source:       FeedbackSourceAI,
			Score:        0.7 + float64(i)*0.02,
			ProviderName: "claude",
		}
		require.NoError(t, collector.Collect(context.Background(), feedback))
	}

	agg, err := collector.GetAggregated(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 10, agg.TotalCount)
	assert.InDelta(t, 0.79, agg.AverageScore, 0.01)
	assert.Equal(t, 10, agg.TypeDistribution[FeedbackTypePositive])
}

func TestInMemoryFeedbackCollector_Export(t *testing.T) {
	collector := NewInMemoryFeedbackCollector(nil, 100)

	for i := 0; i < 3; i++ {
		feedback := &Feedback{
			PromptID:     "prompt-1",
			Type:         FeedbackTypePositive,
			Source:       FeedbackSourceHuman,
			Score:        0.8,
			ProviderName: "claude",
		}
		require.NoError(t, collector.Collect(context.Background(), feedback))
	}

	examples, err := collector.Export(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, examples, 1)
	assert.Len(t, examples[0].Feedback, 3)
}

func TestLLMPolicyOptimizer_Optimize(t *testing.T) {
	provider := NewMockLLMProvider()
	provider.responses["optimize"] = `[{"type": "guideline_addition", "change": "Be more concise", "reason": "Users prefer shorter answers", "improvement_score": 0.7}]`

	config := DefaultSelfImprovementConfig()
	config.MinExamplesForUpdate = 2
	config.UseDebateForOptimize = false

	optimizer := NewLLMPolicyOptimizer(provider, nil, config, nil)
	optimizer.SetCurrentPolicy("Be helpful.")

	examples := []*TrainingExample{
		{RewardScore: 0.3, Prompt: "Question 1"},
		{RewardScore: 0.4, Prompt: "Question 2"},
		{RewardScore: 0.5, Prompt: "Question 3"},
	}

	updates, err := optimizer.Optimize(context.Background(), examples)
	if err == nil && len(updates) > 0 {
		assert.NotEmpty(t, updates[0].NewPolicy)
	}
}

func TestSelfImprovementSystem_Initialize(t *testing.T) {
	config := DefaultSelfImprovementConfig()
	system := NewSelfImprovementSystem(config, nil)

	provider := NewMockLLMProvider()
	debateService := &MockDebateService{}

	err := system.Initialize(provider, debateService)
	require.NoError(t, err)

	assert.NotNil(t, system.GetRewardModel())
	assert.NotNil(t, system.GetDebateAdapter())
}

func TestSelfImprovementSystem_CollectFeedback(t *testing.T) {
	config := DefaultSelfImprovementConfig()
	system := NewSelfImprovementSystem(config, nil)

	provider := NewMockLLMProvider()
	require.NoError(t, system.Initialize(provider, nil))

	feedback := &Feedback{
		SessionID: "session-1",
		Type:      FeedbackTypePositive,
		Score:     0.9,
	}

	err := system.CollectFeedback(context.Background(), feedback)
	require.NoError(t, err)
}

func TestSelfImprovementSystem_ScoreResponse(t *testing.T) {
	config := DefaultSelfImprovementConfig()
	config.UseDebateForReward = false
	system := NewSelfImprovementSystem(config, nil)

	provider := NewMockLLMProvider()
	require.NoError(t, system.Initialize(provider, nil))

	score, err := system.ScoreResponse(context.Background(), "Test prompt", "Test response")
	require.NoError(t, err)
	assert.Greater(t, score, 0.0)
}

func TestDefaultSelfImprovementConfig(t *testing.T) {
	config := DefaultSelfImprovementConfig()

	assert.NotEmpty(t, config.RewardModelProvider)
	assert.True(t, config.AutoCollectFeedback)
	assert.Greater(t, config.FeedbackBatchSize, 0)
	assert.NotEmpty(t, config.ConstitutionalPrinciples)
	assert.True(t, config.UseDebateForReward)
}

func TestFeedbackFilter(t *testing.T) {
	collector := NewInMemoryFeedbackCollector(nil, 100)

	types := []FeedbackType{FeedbackTypePositive, FeedbackTypeNegative, FeedbackTypeNeutral}
	for i, ft := range types {
		feedback := &Feedback{
			SessionID: "session-1",
			Type:      ft,
			Source:    FeedbackSourceHuman,
			Score:     0.5 + float64(i)*0.2,
		}
		require.NoError(t, collector.Collect(context.Background(), feedback))
	}

	filter := &FeedbackFilter{
		Types: []FeedbackType{FeedbackTypePositive},
	}
	agg, err := collector.GetAggregated(context.Background(), filter)
	require.NoError(t, err)
	assert.Equal(t, 1, agg.TotalCount)
}

func TestDebateServiceAdapter(t *testing.T) {
	mockService := &MockDebateService{}
	adapter := NewDebateServiceAdapter(mockService, nil)

	eval, err := adapter.EvaluateWithDebate(context.Background(), "Test prompt", "Test response")
	require.NoError(t, err)
	assert.Greater(t, eval.Score, 0.0)
	assert.NotEmpty(t, eval.DebateID)
}

func TestPreferencePair(t *testing.T) {
	pair := &PreferencePair{
		ID:            "pair-1",
		Prompt:        "Test prompt",
		Chosen:        "Good response",
		Rejected:      "Bad response",
		ChosenScore:   0.9,
		RejectedScore: 0.3,
		Margin:        0.6,
		Source:        FeedbackSourceDebate,
		CreatedAt:     time.Now(),
	}

	assert.Greater(t, pair.ChosenScore, pair.RejectedScore)
	assert.Equal(t, pair.Margin, pair.ChosenScore-pair.RejectedScore)
}
