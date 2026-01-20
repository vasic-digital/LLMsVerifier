package selfimprove

import (
	"context"
	"log"
	"sync"
)

// SelfImprovementSystem orchestrates the self-improvement components
type SelfImprovementSystem struct {
	rewardModel       RewardModel
	feedbackCollector FeedbackCollector
	policyOptimizer   PolicyOptimizer
	debateAdapter     *DebateServiceAdapter
	verifier          ProviderVerifier
	config            *SelfImprovementConfig
	logger            *log.Logger
	mu                sync.RWMutex
	initialized       bool
}

// NewSelfImprovementSystem creates a new self-improvement system
func NewSelfImprovementSystem(config *SelfImprovementConfig, logger *log.Logger) *SelfImprovementSystem {
	if config == nil {
		config = DefaultSelfImprovementConfig()
	}
	if logger == nil {
		logger = log.Default()
	}
	return &SelfImprovementSystem{
		config: config,
		logger: logger,
	}
}

// Initialize sets up the self-improvement system
func (s *SelfImprovementSystem) Initialize(provider LLMProvider, debateService DebateService) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create reward model
	s.rewardModel = NewAIRewardModel(provider, debateService, s.config, s.logger)

	// Create feedback collector
	s.feedbackCollector = NewInMemoryFeedbackCollector(s.logger, s.config.MaxBufferSize)

	// Create policy optimizer
	s.policyOptimizer = NewLLMPolicyOptimizer(provider, debateService, s.config, s.logger)

	// Create debate adapter if debate service available
	if debateService != nil {
		s.debateAdapter = NewDebateServiceAdapter(debateService, s.logger)
	}

	s.initialized = true
	s.logger.Println("Self-improvement system initialized")
	return nil
}

// SetVerifier sets the provider verifier for integration
func (s *SelfImprovementSystem) SetVerifier(verifier ProviderVerifier) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verifier = verifier
}

// GetRewardModel returns the reward model
func (s *SelfImprovementSystem) GetRewardModel() RewardModel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rewardModel
}

// GetFeedbackCollector returns the feedback collector
func (s *SelfImprovementSystem) GetFeedbackCollector() FeedbackCollector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.feedbackCollector
}

// GetPolicyOptimizer returns the policy optimizer
func (s *SelfImprovementSystem) GetPolicyOptimizer() PolicyOptimizer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policyOptimizer
}

// GetDebateAdapter returns the debate adapter
func (s *SelfImprovementSystem) GetDebateAdapter() *DebateServiceAdapter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.debateAdapter
}

// CollectFeedback collects feedback on a response
func (s *SelfImprovementSystem) CollectFeedback(ctx context.Context, feedback *Feedback) error {
	s.mu.RLock()
	collector := s.feedbackCollector
	s.mu.RUnlock()

	if collector == nil {
		return nil
	}
	return collector.Collect(ctx, feedback)
}

// ScoreResponse scores a response using the reward model
func (s *SelfImprovementSystem) ScoreResponse(ctx context.Context, prompt, response string) (float64, error) {
	s.mu.RLock()
	model := s.rewardModel
	s.mu.RUnlock()

	if model == nil {
		return 0.5, nil
	}
	return model.Score(ctx, prompt, response)
}

// OptimizePolicy triggers policy optimization
func (s *SelfImprovementSystem) OptimizePolicy(ctx context.Context) ([]*PolicyUpdate, error) {
	s.mu.RLock()
	collector := s.feedbackCollector
	optimizer := s.policyOptimizer
	s.mu.RUnlock()

	if collector == nil || optimizer == nil {
		return nil, nil
	}

	examples, err := collector.Export(ctx, nil)
	if err != nil {
		return nil, err
	}

	return optimizer.Optimize(ctx, examples)
}

// ApplyPolicyUpdate applies a policy update
func (s *SelfImprovementSystem) ApplyPolicyUpdate(ctx context.Context, update *PolicyUpdate) error {
	s.mu.RLock()
	optimizer := s.policyOptimizer
	s.mu.RUnlock()

	if optimizer == nil {
		return nil
	}
	return optimizer.Apply(ctx, update)
}

// GetCurrentPolicy returns the current policy
func (s *SelfImprovementSystem) GetCurrentPolicy() string {
	s.mu.RLock()
	optimizer := s.policyOptimizer
	s.mu.RUnlock()

	if optimizer == nil {
		return ""
	}
	return optimizer.GetCurrentPolicy()
}

// SetCurrentPolicy sets the current policy
func (s *SelfImprovementSystem) SetCurrentPolicy(policy string) {
	s.mu.RLock()
	optimizer := s.policyOptimizer
	s.mu.RUnlock()

	if optimizer != nil {
		optimizer.SetCurrentPolicy(policy)
	}
}

// GetFeedbackStats returns aggregated feedback statistics
func (s *SelfImprovementSystem) GetFeedbackStats(ctx context.Context) (*AggregatedFeedback, error) {
	s.mu.RLock()
	collector := s.feedbackCollector
	s.mu.RUnlock()

	if collector == nil {
		return &AggregatedFeedback{}, nil
	}
	return collector.GetAggregated(ctx, nil)
}

// IsInitialized returns whether the system is initialized
func (s *SelfImprovementSystem) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}
