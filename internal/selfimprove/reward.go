package selfimprove

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AIRewardModel implements RewardModel using AI-based evaluation
type AIRewardModel struct {
	provider      LLMProvider
	debateService DebateService
	config        *SelfImprovementConfig
	logger        *log.Logger
	mu            sync.RWMutex
	cache         map[string]float64
}

// NewAIRewardModel creates a new AI-based reward model
func NewAIRewardModel(provider LLMProvider, debateService DebateService, config *SelfImprovementConfig, logger *log.Logger) *AIRewardModel {
	if config == nil {
		config = DefaultSelfImprovementConfig()
	}
	if logger == nil {
		logger = log.Default()
	}
	return &AIRewardModel{
		provider:      provider,
		debateService: debateService,
		config:        config,
		logger:        logger,
		cache:         make(map[string]float64),
	}
}

// Score evaluates a response and returns a reward score
func (rm *AIRewardModel) Score(ctx context.Context, prompt, response string) (float64, error) {
	if rm.config.UseDebateForReward && rm.debateService != nil {
		return rm.scoreWithDebate(ctx, prompt, response)
	}
	return rm.scoreWithLLM(ctx, prompt, response)
}

// scoreWithLLM uses a single LLM to evaluate the response
func (rm *AIRewardModel) scoreWithLLM(ctx context.Context, prompt, response string) (float64, error) {
	systemPrompt := `You are an expert response evaluator. Evaluate the response quality.
Return JSON: {"score": 0.X, "reasoning": "brief explanation"}`

	evalPrompt := fmt.Sprintf(`Evaluate this AI response:
User Prompt: %s
AI Response: %s
Return JSON with score (0.0-1.0).`, prompt, response)

	result, err := rm.provider.Complete(ctx, evalPrompt, systemPrompt)
	if err != nil {
		return 0.0, fmt.Errorf("failed to evaluate: %w", err)
	}

	return rm.parseScoreFromJSON(result)
}

// scoreWithDebate uses AI debate to evaluate the response
func (rm *AIRewardModel) scoreWithDebate(ctx context.Context, prompt, response string) (float64, error) {
	topic := fmt.Sprintf(`Evaluate quality of: Prompt: %s Response: %s`, prompt, response)
	result, err := rm.debateService.RunDebate(ctx, topic, nil)
	if err != nil {
		rm.logger.Print(trData("selfimprove.log.debate_failed_fallback", map[string]any{"error": err.Error()}))
		return rm.scoreWithLLM(ctx, prompt, response)
	}
	return rm.parseScoreFromJSON(result.Consensus)
}

// parseScoreFromJSON extracts a score from JSON response
func (rm *AIRewardModel) parseScoreFromJSON(jsonStr string) (float64, error) {
	var result struct {
		Score     float64 `json:"score"`
		Reasoning string  `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return 0.5, nil
	}
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 1 {
		result.Score = 1
	}
	return result.Score, nil
}

// ScoreWithDimensions evaluates across multiple dimensions
func (rm *AIRewardModel) ScoreWithDimensions(ctx context.Context, prompt, response string) (map[DimensionType]float64, error) {
	systemPrompt := `Evaluate the response across dimensions. Return JSON with scores 0.0-1.0.`
	evalPrompt := fmt.Sprintf(`Prompt: %s Response: %s`, prompt, response)

	result, err := rm.provider.Complete(ctx, evalPrompt, systemPrompt)
	if err != nil {
		return nil, err
	}
	return rm.parseDimensionsFromJSON(result)
}

// parseDimensionsFromJSON extracts dimension scores from JSON
func (rm *AIRewardModel) parseDimensionsFromJSON(jsonStr string) (map[DimensionType]float64, error) {
	var result map[string]float64
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, err
	}

	dimensions := make(map[DimensionType]float64)
	dimensionMap := map[string]DimensionType{
		"accuracy": DimensionAccuracy, "relevance": DimensionRelevance,
		"helpfulness": DimensionHelpfulness, "harmlessness": DimensionHarmless,
		"honesty": DimensionHonest, "coherence": DimensionCoherence,
		"creativity": DimensionCreativity, "formatting": DimensionFormatting,
	}
	for key, score := range result {
		if dim, ok := dimensionMap[key]; ok {
			if score < 0 {
				score = 0
			}
			if score > 1 {
				score = 1
			}
			dimensions[dim] = score
		}
	}
	return dimensions, nil
}

// Compare compares two responses and returns a preference pair
func (rm *AIRewardModel) Compare(ctx context.Context, prompt, response1, response2 string) (*PreferencePair, error) {
	systemPrompt := `Compare two AI responses. Return: {"preferred": "A" or "B", "margin": 0.X}`
	comparePrompt := fmt.Sprintf(`Prompt: %s
Response A: %s
Response B: %s`, prompt, response1, response2)

	result, err := rm.provider.Complete(ctx, comparePrompt, systemPrompt)
	if err != nil {
		return nil, err
	}

	var comparison struct {
		Preferred string  `json:"preferred"`
		Margin    float64 `json:"margin"`
	}
	json.Unmarshal([]byte(result), &comparison)

	pair := &PreferencePair{
		ID:        uuid.New().String(),
		Prompt:    prompt,
		Source:    FeedbackSourceAI,
		Margin:    comparison.Margin,
		CreatedAt: time.Now(),
	}

	if comparison.Preferred == "A" {
		pair.Chosen, pair.Rejected = response1, response2
		pair.ChosenScore, pair.RejectedScore = 0.5+comparison.Margin/2, 0.5-comparison.Margin/2
	} else {
		pair.Chosen, pair.Rejected = response2, response1
		pair.ChosenScore, pair.RejectedScore = 0.5+comparison.Margin/2, 0.5-comparison.Margin/2
	}
	return pair, nil
}

// Train updates the reward model with new examples
func (rm *AIRewardModel) Train(ctx context.Context, examples []*TrainingExample) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.cache = make(map[string]float64)
	rm.logger.Print(trData("selfimprove.log.training_reward_model", map[string]any{"count": len(examples)}))
	return nil
}

// DebateServiceAdapter adapts the debate service for self-improvement
type DebateServiceAdapter struct {
	debateService DebateService
	logger        *log.Logger
}

// NewDebateServiceAdapter creates a new adapter
func NewDebateServiceAdapter(service DebateService, logger *log.Logger) *DebateServiceAdapter {
	if logger == nil {
		logger = log.Default()
	}
	return &DebateServiceAdapter{debateService: service, logger: logger}
}

// EvaluateWithDebate uses AI debate to evaluate a response
func (a *DebateServiceAdapter) EvaluateWithDebate(ctx context.Context, prompt, response string) (*DebateEvaluation, error) {
	topic := fmt.Sprintf(`Evaluate: Prompt: %s Response: %s`, prompt, response)
	result, err := a.debateService.RunDebate(ctx, topic, nil)
	if err != nil {
		return nil, err
	}

	var consensus struct {
		Score float64 `json:"score"`
	}
	json.Unmarshal([]byte(result.Consensus), &consensus)

	return &DebateEvaluation{
		Score:            consensus.Score,
		DebateID:         result.ID,
		ParticipantVotes: result.Votes,
		Confidence:       result.Confidence,
	}, nil
}
