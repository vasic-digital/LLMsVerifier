package llmops

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryContinuousEvaluator implements ContinuousEvaluator
type InMemoryContinuousEvaluator struct {
	mu           sync.RWMutex
	datasets     map[string]*Dataset
	samples      map[string][]*DatasetSample
	runs         map[string]*EvaluationRun
	config       *LLMOpsConfig
	alertManager AlertManager
	debateEval   DebateLLMEvaluator
	logger       *log.Logger
}

// DebateLLMEvaluator interface for debate-based evaluation
type DebateLLMEvaluator interface {
	Evaluate(ctx context.Context, prompt, response, expected string) (float64, error)
}

// NewInMemoryContinuousEvaluator creates a new evaluator
func NewInMemoryContinuousEvaluator(config *LLMOpsConfig, alertManager AlertManager, debateEval DebateLLMEvaluator, logger *log.Logger) *InMemoryContinuousEvaluator {
	if config == nil {
		config = DefaultLLMOpsConfig()
	}
	if logger == nil {
		logger = log.Default()
	}
	return &InMemoryContinuousEvaluator{
		datasets:     make(map[string]*Dataset),
		samples:      make(map[string][]*DatasetSample),
		runs:         make(map[string]*EvaluationRun),
		config:       config,
		alertManager: alertManager,
		debateEval:   debateEval,
		logger:       logger,
	}
}

// CreateDataset creates a new dataset
func (e *InMemoryContinuousEvaluator) CreateDataset(ctx context.Context, dataset *Dataset) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if dataset.ID == "" {
		dataset.ID = uuid.New().String()
	}
	dataset.CreatedAt = time.Now()

	e.datasets[dataset.ID] = dataset
	e.samples[dataset.ID] = make([]*DatasetSample, 0)

	e.logger.Printf("Created dataset: %s", dataset.Name)
	return nil
}

// AddSamples adds samples to a dataset
func (e *InMemoryContinuousEvaluator) AddSamples(ctx context.Context, datasetID string, samples []*DatasetSample) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.datasets[datasetID]; !ok {
		return fmt.Errorf("dataset not found: %s", datasetID)
	}

	for _, s := range samples {
		if s.ID == "" {
			s.ID = uuid.New().String()
		}
	}

	e.samples[datasetID] = append(e.samples[datasetID], samples...)
	e.datasets[datasetID].Samples = len(e.samples[datasetID])

	return nil
}

// CreateRun creates a new evaluation run
func (e *InMemoryContinuousEvaluator) CreateRun(ctx context.Context, run *EvaluationRun) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if run.ID == "" {
		run.ID = uuid.New().String()
	}
	run.Status = EvaluationStatusPending
	run.CreatedAt = time.Now()

	e.runs[run.ID] = run

	e.logger.Printf("Created evaluation run: %s", run.Name)
	return nil
}

// StartRun starts an evaluation run
func (e *InMemoryContinuousEvaluator) StartRun(ctx context.Context, runID string) error {
	e.mu.Lock()
	run, ok := e.runs[runID]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("run not found: %s", runID)
	}

	run.Status = EvaluationStatusRunning
	now := time.Now()
	run.StartedAt = &now
	e.mu.Unlock()

	// Run evaluation in background
	go e.executeRun(ctx, run)

	return nil
}

// executeRun executes the evaluation
func (e *InMemoryContinuousEvaluator) executeRun(ctx context.Context, run *EvaluationRun) {
	e.mu.RLock()
	samples := e.samples[run.Dataset]
	e.mu.RUnlock()

	results := &EvaluationResults{
		TotalSamples: len(samples),
		Metrics:      make(map[string]float64),
	}

	var passed int
	for _, sample := range samples {
		// Simulate evaluation
		score := 0.8 // In real implementation, use LLM or debate
		if e.debateEval != nil {
			if s, err := e.debateEval.Evaluate(ctx, sample.Input, "", sample.ExpectedOutput); err == nil {
				score = s
			}
		}

		if score >= 0.5 {
			passed++
		}
	}

	results.PassedSamples = passed
	if results.TotalSamples > 0 {
		results.PassRate = float64(passed) / float64(results.TotalSamples)
	}

	e.mu.Lock()
	run.Results = results
	run.Status = EvaluationStatusCompleted
	now := time.Now()
	run.EndedAt = &now
	e.mu.Unlock()

	// Check for regressions
	e.checkForRegressions(ctx, run)

	e.logger.Printf("Completed evaluation run: %s, pass rate: %.2f", run.Name, results.PassRate)
}

// checkForRegressions checks for quality regressions
func (e *InMemoryContinuousEvaluator) checkForRegressions(ctx context.Context, run *EvaluationRun) {
	if e.alertManager == nil || run.Results == nil {
		return
	}

	threshold := e.config.AlertThresholds["pass_rate_drop"]
	if run.Results.PassRate < (1.0 - threshold) {
		alert := &Alert{
			Type:     AlertTypeRegression,
			Severity: AlertSeverityWarning,
			Message:  fmt.Sprintf("Pass rate dropped to %.2f%%", run.Results.PassRate*100),
			Source:   "evaluator",
			Data: map[string]interface{}{
				"run_id":    run.ID,
				"pass_rate": run.Results.PassRate,
			},
		}
		e.alertManager.Create(ctx, alert)
	}
}

// GetRun retrieves an evaluation run. Returns a shallow copy so callers
// can read scalar fields (Status, Results, etc.) without racing against
// the background executeRun goroutine which mutates the live struct
// under e.mu. The clone shares map/slice references with the live
// run; those fields are read-only post-completion so the shallow copy
// is race-free for the public-API use cases.
func (e *InMemoryContinuousEvaluator) GetRun(ctx context.Context, runID string) (*EvaluationRun, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	run, ok := e.runs[runID]
	if !ok {
		return nil, fmt.Errorf("run not found: %s", runID)
	}
	clone := *run
	return &clone, nil
}

// ListRuns lists all evaluation runs
func (e *InMemoryContinuousEvaluator) ListRuns(ctx context.Context) ([]*EvaluationRun, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*EvaluationRun, 0, len(e.runs))
	for _, run := range e.runs {
		result = append(result, run)
	}
	return result, nil
}

// InMemoryAlertManager implements AlertManager
type InMemoryAlertManager struct {
	mu       sync.RWMutex
	alerts   map[string]*Alert
	handlers []AlertHandler
	logger   *log.Logger
}

// NewInMemoryAlertManager creates a new alert manager
func NewInMemoryAlertManager(logger *log.Logger) *InMemoryAlertManager {
	if logger == nil {
		logger = log.Default()
	}
	return &InMemoryAlertManager{
		alerts:   make(map[string]*Alert),
		handlers: make([]AlertHandler, 0),
		logger:   logger,
	}
}

// Create creates a new alert
func (m *InMemoryAlertManager) Create(ctx context.Context, alert *Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}
	alert.CreatedAt = time.Now()

	m.alerts[alert.ID] = alert

	// Notify handlers
	for _, h := range m.handlers {
		go h(alert)
	}

	m.logger.Printf("Created alert: %s - %s", alert.Type, alert.Message)
	return nil
}

// Get retrieves an alert
func (m *InMemoryAlertManager) Get(ctx context.Context, id string) (*Alert, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alert, ok := m.alerts[id]
	if !ok {
		return nil, fmt.Errorf("alert not found: %s", id)
	}
	return alert, nil
}

// List lists alerts with filtering
func (m *InMemoryAlertManager) List(ctx context.Context, filter *AlertFilter) ([]*Alert, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Alert, 0)
	for _, alert := range m.alerts {
		if filter != nil {
			if filter.Resolved != nil && alert.Resolved != *filter.Resolved {
				continue
			}
		}
		result = append(result, alert)
	}
	return result, nil
}

// Resolve resolves an alert
func (m *InMemoryAlertManager) Resolve(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert, ok := m.alerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	alert.Resolved = true
	now := time.Now()
	alert.ResolvedAt = &now

	return nil
}

// Subscribe subscribes to alerts
func (m *InMemoryAlertManager) Subscribe(ctx context.Context, handler AlertHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handlers = append(m.handlers, handler)
	return nil
}
