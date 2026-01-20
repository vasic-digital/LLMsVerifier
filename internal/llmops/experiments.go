package llmops

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryExperimentManager implements ExperimentManager
type InMemoryExperimentManager struct {
	mu            sync.RWMutex
	experiments   map[string]*Experiment
	assignments   map[string]map[string]string // expID -> userID -> variantID
	metrics       map[string]map[string][]float64 // expID -> variantID+metric -> values
	logger        *log.Logger
}

// NewInMemoryExperimentManager creates a new experiment manager
func NewInMemoryExperimentManager(logger *log.Logger) *InMemoryExperimentManager {
	if logger == nil {
		logger = log.Default()
	}
	return &InMemoryExperimentManager{
		experiments: make(map[string]*Experiment),
		assignments: make(map[string]map[string]string),
		metrics:     make(map[string]map[string][]float64),
		logger:      logger,
	}
}

// Create creates a new experiment
func (m *InMemoryExperimentManager) Create(ctx context.Context, exp *Experiment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if exp.ID == "" {
		exp.ID = uuid.New().String()
	}
	exp.Status = ExperimentStatusDraft
	exp.CreatedAt = time.Now()
	exp.UpdatedAt = time.Now()

	// Assign IDs to variants
	for i, v := range exp.Variants {
		if v.ID == "" {
			v.ID = uuid.New().String()
		}
		exp.Variants[i] = v
	}

	// Initialize traffic split if not set
	if exp.TrafficSplit == nil {
		exp.TrafficSplit = make(map[string]float64)
		split := 1.0 / float64(len(exp.Variants))
		for _, v := range exp.Variants {
			exp.TrafficSplit[v.ID] = split
		}
	}

	m.experiments[exp.ID] = exp
	m.assignments[exp.ID] = make(map[string]string)
	m.metrics[exp.ID] = make(map[string][]float64)

	m.logger.Printf("Created experiment: %s", exp.Name)
	return nil
}

// Get retrieves an experiment by ID
func (m *InMemoryExperimentManager) Get(ctx context.Context, id string) (*Experiment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exp, ok := m.experiments[id]
	if !ok {
		return nil, fmt.Errorf("experiment not found: %s", id)
	}
	return exp, nil
}

// List lists all experiments
func (m *InMemoryExperimentManager) List(ctx context.Context) ([]*Experiment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Experiment, 0, len(m.experiments))
	for _, exp := range m.experiments {
		result = append(result, exp)
	}
	return result, nil
}

// Start starts an experiment
func (m *InMemoryExperimentManager) Start(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, ok := m.experiments[id]
	if !ok {
		return fmt.Errorf("experiment not found: %s", id)
	}

	if exp.Status == ExperimentStatusCompleted {
		return fmt.Errorf("cannot start completed experiment")
	}

	exp.Status = ExperimentStatusRunning
	now := time.Now()
	exp.StartedAt = &now
	exp.UpdatedAt = time.Now()

	m.logger.Printf("Started experiment: %s", exp.Name)
	return nil
}

// Pause pauses an experiment
func (m *InMemoryExperimentManager) Pause(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, ok := m.experiments[id]
	if !ok {
		return fmt.Errorf("experiment not found: %s", id)
	}

	if exp.Status != ExperimentStatusRunning {
		return fmt.Errorf("cannot pause non-running experiment")
	}

	exp.Status = ExperimentStatusPaused
	exp.UpdatedAt = time.Now()

	m.logger.Printf("Paused experiment: %s", exp.Name)
	return nil
}

// Complete completes an experiment
func (m *InMemoryExperimentManager) Complete(ctx context.Context, id, winner string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, ok := m.experiments[id]
	if !ok {
		return fmt.Errorf("experiment not found: %s", id)
	}

	exp.Status = ExperimentStatusCompleted
	exp.Winner = winner
	now := time.Now()
	exp.EndedAt = &now
	exp.UpdatedAt = time.Now()

	m.logger.Printf("Completed experiment: %s, winner: %s", exp.Name, winner)
	return nil
}

// AssignVariant assigns a user to a variant
func (m *InMemoryExperimentManager) AssignVariant(ctx context.Context, expID, userID string) (*Variant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, ok := m.experiments[expID]
	if !ok {
		return nil, fmt.Errorf("experiment not found: %s", expID)
	}

	if exp.Status != ExperimentStatusRunning {
		return nil, fmt.Errorf("experiment not running: %s", expID)
	}

	// Check if already assigned
	if variantID, ok := m.assignments[expID][userID]; ok {
		for _, v := range exp.Variants {
			if v.ID == variantID {
				return v, nil
			}
		}
	}

	// Deterministic assignment based on hash
	h := fnv.New32a()
	h.Write([]byte(exp.ID + userID))
	hashValue := float64(h.Sum32()) / float64(^uint32(0))

	var cumulative float64
	for _, v := range exp.Variants {
		cumulative += exp.TrafficSplit[v.ID]
		if hashValue <= cumulative {
			m.assignments[expID][userID] = v.ID
			v.Assignments++
			return v, nil
		}
	}

	// Fallback to last variant
	last := exp.Variants[len(exp.Variants)-1]
	m.assignments[expID][userID] = last.ID
	last.Assignments++
	return last, nil
}

// RecordMetric records a metric value for a variant
func (m *InMemoryExperimentManager) RecordMetric(ctx context.Context, expID, variantID, metric string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.metrics[expID]; !ok {
		m.metrics[expID] = make(map[string][]float64)
	}

	key := variantID + ":" + metric
	m.metrics[expID][key] = append(m.metrics[expID][key], value)
	return nil
}

// GetResults gets experiment results
func (m *InMemoryExperimentManager) GetResults(ctx context.Context, expID string) (*ExperimentResults, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exp, ok := m.experiments[expID]
	if !ok {
		return nil, fmt.Errorf("experiment not found: %s", expID)
	}

	results := &ExperimentResults{
		ExperimentID:   expID,
		VariantResults: make(map[string]*VariantResult),
	}

	expMetrics := m.metrics[expID]
	var controlAvg float64

	for _, v := range exp.Variants {
		vr := &VariantResult{
			VariantID:    v.ID,
			MetricValues: make(map[string]float64),
		}

		for _, metric := range exp.Metrics {
			key := v.ID + ":" + metric
			values := expMetrics[key]
			if len(values) > 0 {
				avg := average(values)
				vr.MetricValues[metric] = avg
				vr.SampleCount = len(values)
				if metric == exp.TargetMetric {
					vr.AverageScore = avg
				}
			}
		}

		if v.IsControl {
			controlAvg = vr.AverageScore
		}

		results.VariantResults[v.ID] = vr
		results.TotalSamples += vr.SampleCount
	}

	// Calculate improvement vs control
	for _, vr := range results.VariantResults {
		if controlAvg > 0 {
			vr.Improvement = (vr.AverageScore - controlAvg) / controlAvg
		}
	}

	return results, nil
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
