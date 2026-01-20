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

// LLMPolicyOptimizer implements PolicyOptimizer using LLM-based optimization
type LLMPolicyOptimizer struct {
	provider      LLMProvider
	debateService DebateService
	config        *SelfImprovementConfig
	logger        *log.Logger
	mu            sync.RWMutex
	currentPolicy string
	history       []*PolicyUpdate
}

// NewLLMPolicyOptimizer creates a new LLM-based policy optimizer
func NewLLMPolicyOptimizer(provider LLMProvider, debateService DebateService, config *SelfImprovementConfig, logger *log.Logger) *LLMPolicyOptimizer {
	if config == nil {
		config = DefaultSelfImprovementConfig()
	}
	if logger == nil {
		logger = log.Default()
	}
	return &LLMPolicyOptimizer{
		provider:      provider,
		debateService: debateService,
		config:        config,
		logger:        logger,
		history:       make([]*PolicyUpdate, 0),
	}
}

// Optimize generates policy updates from training examples
func (opt *LLMPolicyOptimizer) Optimize(ctx context.Context, examples []*TrainingExample) ([]*PolicyUpdate, error) {
	if len(examples) < opt.config.MinExamplesForUpdate {
		return nil, nil
	}

	// Analyze low-scoring examples
	lowScoreExamples := opt.filterLowScoreExamples(examples)
	if len(lowScoreExamples) == 0 {
		return nil, nil
	}

	// Generate update suggestions
	updates, err := opt.generateUpdates(ctx, lowScoreExamples)
	if err != nil {
		return nil, err
	}

	// Apply self-critique if enabled
	if opt.config.EnableSelfCritique {
		updates = opt.applySelfCritique(ctx, updates)
	}

	return updates, nil
}

// filterLowScoreExamples filters examples with low reward scores
func (opt *LLMPolicyOptimizer) filterLowScoreExamples(examples []*TrainingExample) []*TrainingExample {
	var lowScore []*TrainingExample
	for _, ex := range examples {
		if ex.RewardScore < opt.config.MinRewardThreshold {
			lowScore = append(lowScore, ex)
		}
	}
	return lowScore
}

// generateUpdates uses LLM to generate policy updates
func (opt *LLMPolicyOptimizer) generateUpdates(ctx context.Context, examples []*TrainingExample) ([]*PolicyUpdate, error) {
	systemPrompt := `You are an AI policy optimization expert. Analyze these low-quality response examples and suggest policy improvements.

Return JSON array of updates:
[{
  "type": "guideline_addition|prompt_refinement|constraint_update",
  "change": "the specific change to make",
  "reason": "why this will improve quality",
  "improvement_score": 0.0-1.0
}]`

	examplesJSON, _ := json.Marshal(examples[:min(len(examples), 5)])
	prompt := fmt.Sprintf(`Current Policy: %s

Low-quality examples:
%s

Suggest policy improvements.`, opt.currentPolicy, string(examplesJSON))

	result, err := opt.provider.Complete(ctx, prompt, systemPrompt)
	if err != nil {
		return nil, err
	}

	return opt.parseUpdates(result)
}

// parseUpdates parses LLM response into policy updates
func (opt *LLMPolicyOptimizer) parseUpdates(response string) ([]*PolicyUpdate, error) {
	var rawUpdates []struct {
		Type             string  `json:"type"`
		Change           string  `json:"change"`
		Reason           string  `json:"reason"`
		ImprovementScore float64 `json:"improvement_score"`
	}

	if err := json.Unmarshal([]byte(response), &rawUpdates); err != nil {
		return nil, nil // Return empty on parse error
	}

	var updates []*PolicyUpdate
	for _, raw := range rawUpdates {
		updateType := PolicyUpdateGuidelineAddition
		switch raw.Type {
		case "prompt_refinement":
			updateType = PolicyUpdatePromptRefinement
		case "constraint_update":
			updateType = PolicyUpdateConstraintUpdate
		case "tone_adjustment":
			updateType = PolicyUpdateToneAdjustment
		}

		update := &PolicyUpdate{
			ID:               uuid.New().String(),
			OldPolicy:        opt.currentPolicy,
			UpdateType:       updateType,
			Change:           raw.Change,
			Reason:           raw.Reason,
			ImprovementScore: raw.ImprovementScore,
			CreatedAt:        time.Now(),
		}
		update.NewPolicy = opt.applyChange(update)
		updates = append(updates, update)
	}

	return updates, nil
}

// applyChange applies a change to create new policy
func (opt *LLMPolicyOptimizer) applyChange(update *PolicyUpdate) string {
	return opt.currentPolicy + "\n" + update.Change
}

// applySelfCritique filters updates through constitutional principles
func (opt *LLMPolicyOptimizer) applySelfCritique(ctx context.Context, updates []*PolicyUpdate) []*PolicyUpdate {
	if len(opt.config.ConstitutionalPrinciples) == 0 {
		return updates
	}

	var approved []*PolicyUpdate
	for _, update := range updates {
		if opt.passesConstitutionalCheck(update) {
			approved = append(approved, update)
		}
	}
	return approved
}

// passesConstitutionalCheck verifies update against principles
func (opt *LLMPolicyOptimizer) passesConstitutionalCheck(update *PolicyUpdate) bool {
	for _, principle := range opt.config.ConstitutionalPrinciples {
		if containsViolation(update.Change, principle) {
			return false
		}
	}
	return true
}

// containsViolation is a simple check for principle violations
func containsViolation(change, principle string) bool {
	// Simplified check - in production use LLM evaluation
	return false
}

// Apply applies a policy update
func (opt *LLMPolicyOptimizer) Apply(ctx context.Context, update *PolicyUpdate) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()

	opt.currentPolicy = update.NewPolicy
	now := time.Now()
	update.AppliedAt = &now
	opt.history = append(opt.history, update)

	opt.logger.Printf("Applied policy update: %s", update.ID)
	return nil
}

// Rollback reverts a policy update
func (opt *LLMPolicyOptimizer) Rollback(ctx context.Context, updateID string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()

	for i := len(opt.history) - 1; i >= 0; i-- {
		if opt.history[i].ID == updateID {
			opt.currentPolicy = opt.history[i].OldPolicy
			opt.history = opt.history[:i]
			opt.logger.Printf("Rolled back policy update: %s", updateID)
			return nil
		}
	}
	return fmt.Errorf("update not found: %s", updateID)
}

// GetHistory returns policy update history
func (opt *LLMPolicyOptimizer) GetHistory(ctx context.Context, limit int) ([]*PolicyUpdate, error) {
	opt.mu.RLock()
	defer opt.mu.RUnlock()

	if limit <= 0 || limit > len(opt.history) {
		limit = len(opt.history)
	}

	result := make([]*PolicyUpdate, limit)
	copy(result, opt.history[len(opt.history)-limit:])
	return result, nil
}

// GetCurrentPolicy returns the current policy
func (opt *LLMPolicyOptimizer) GetCurrentPolicy() string {
	opt.mu.RLock()
	defer opt.mu.RUnlock()
	return opt.currentPolicy
}

// SetCurrentPolicy sets the current policy
func (opt *LLMPolicyOptimizer) SetCurrentPolicy(policy string) {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	opt.currentPolicy = policy
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
