package llmops

import (
	"context"
	"log"
	"sync"
)

// LLMOpsSystem orchestrates all LLMOps components
type LLMOpsSystem struct {
	promptRegistry    PromptRegistry
	experimentManager ExperimentManager
	evaluator         ContinuousEvaluator
	alertManager      AlertManager
	config            *LLMOpsConfig
	logger            *log.Logger
	mu                sync.RWMutex
	initialized       bool
}

// NewLLMOpsSystem creates a new LLMOps system
func NewLLMOpsSystem(config *LLMOpsConfig, logger *log.Logger) *LLMOpsSystem {
	if config == nil {
		config = DefaultLLMOpsConfig()
	}
	if logger == nil {
		logger = log.Default()
	}
	return &LLMOpsSystem{
		config: config,
		logger: logger,
	}
}

// Initialize sets up all components
func (s *LLMOpsSystem) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.alertManager = NewInMemoryAlertManager(s.logger)
	s.promptRegistry = NewInMemoryPromptRegistry(s.logger)
	s.experimentManager = NewInMemoryExperimentManager(s.logger)
	s.evaluator = NewInMemoryContinuousEvaluator(s.config, s.alertManager, nil, s.logger)

	s.initialized = true
	s.logger.Println("LLMOps system initialized")
	return nil
}

// GetPromptRegistry returns the prompt registry
func (s *LLMOpsSystem) GetPromptRegistry() PromptRegistry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.promptRegistry
}

// GetExperimentManager returns the experiment manager
func (s *LLMOpsSystem) GetExperimentManager() ExperimentManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.experimentManager
}

// GetEvaluator returns the continuous evaluator
func (s *LLMOpsSystem) GetEvaluator() ContinuousEvaluator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.evaluator
}

// GetAlertManager returns the alert manager
func (s *LLMOpsSystem) GetAlertManager() AlertManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.alertManager
}

// CreatePromptExperiment creates an A/B test between two prompts
func (s *LLMOpsSystem) CreatePromptExperiment(ctx context.Context, name string, control, treatment *PromptVersion, trafficRatio float64) (*Experiment, error) {
	// Register prompts
	if err := s.promptRegistry.Create(ctx, control); err != nil {
		return nil, err
	}
	if err := s.promptRegistry.Create(ctx, treatment); err != nil {
		return nil, err
	}

	exp := &Experiment{
		Name:        name,
		Description: "A/B test between prompt versions",
		Variants: []*Variant{
			{Name: "Control", PromptID: control.ID, IsControl: true},
			{Name: "Treatment", PromptID: treatment.ID},
		},
		TrafficSplit: map[string]float64{},
		Metrics:      []string{"quality", "latency"},
		TargetMetric: "quality",
	}

	// Set traffic split
	if err := s.experimentManager.Create(ctx, exp); err != nil {
		return nil, err
	}

	exp.TrafficSplit[exp.Variants[0].ID] = 1.0 - trafficRatio
	exp.TrafficSplit[exp.Variants[1].ID] = trafficRatio

	return exp, nil
}

// IsInitialized returns whether the system is initialized
func (s *LLMOpsSystem) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}
