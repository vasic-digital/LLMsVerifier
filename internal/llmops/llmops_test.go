package llmops

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryPromptRegistry_Create(t *testing.T) {
	registry := NewInMemoryPromptRegistry(nil)

	prompt := &PromptVersion{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello, {{name}}!",
		Variables: []PromptVariable{
			{Name: "name", Type: "string", Required: true},
		},
	}

	err := registry.Create(context.Background(), prompt)
	require.NoError(t, err)
	assert.NotEmpty(t, prompt.ID)
	assert.True(t, prompt.IsActive)
}

func TestInMemoryPromptRegistry_GetLatest(t *testing.T) {
	registry := NewInMemoryPromptRegistry(nil)

	v1 := &PromptVersion{Name: "test", Version: "1.0.0", Content: "V1"}
	v2 := &PromptVersion{Name: "test", Version: "2.0.0", Content: "V2"}

	require.NoError(t, registry.Create(context.Background(), v1))
	require.NoError(t, registry.Create(context.Background(), v2))

	// V1 is active by default
	latest, err := registry.GetLatest(context.Background(), "test")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", latest.Version)

	// Activate V2
	require.NoError(t, registry.Activate(context.Background(), "test", "2.0.0"))

	latest, err = registry.GetLatest(context.Background(), "test")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", latest.Version)
}

func TestInMemoryPromptRegistry_Render(t *testing.T) {
	registry := NewInMemoryPromptRegistry(nil)

	prompt := &PromptVersion{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello, {{name}}! You are {{age}} years old.",
		Variables: []PromptVariable{
			{Name: "name", Type: "string", Required: true},
			{Name: "age", Type: "int", Required: false, Default: 25},
		},
	}

	require.NoError(t, registry.Create(context.Background(), prompt))

	// Render with all variables
	rendered, err := registry.Render(context.Background(), "greeting", "1.0.0", map[string]interface{}{
		"name": "John",
		"age":  30,
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello, John! You are 30 years old.", rendered)

	// Render with default
	rendered, err = registry.Render(context.Background(), "greeting", "1.0.0", map[string]interface{}{
		"name": "Jane",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello, Jane! You are 25 years old.", rendered)
}

func TestInMemoryPromptRegistry_Delete(t *testing.T) {
	registry := NewInMemoryPromptRegistry(nil)

	v1 := &PromptVersion{Name: "test", Version: "1.0.0", Content: "V1"}
	v2 := &PromptVersion{Name: "test", Version: "2.0.0", Content: "V2"}

	require.NoError(t, registry.Create(context.Background(), v1))
	require.NoError(t, registry.Create(context.Background(), v2))

	// Cannot delete active version
	err := registry.Delete(context.Background(), "test", "1.0.0")
	assert.Error(t, err)

	// Activate V2 and delete V1
	require.NoError(t, registry.Activate(context.Background(), "test", "2.0.0"))
	require.NoError(t, registry.Delete(context.Background(), "test", "1.0.0"))

	_, err = registry.Get(context.Background(), "test", "1.0.0")
	assert.Error(t, err)
}

func TestInMemoryExperimentManager_Create(t *testing.T) {
	manager := NewInMemoryExperimentManager(nil)

	exp := &Experiment{
		Name: "Test Experiment",
		Variants: []*Variant{
			{Name: "Control", IsControl: true},
			{Name: "Treatment"},
		},
		Metrics:      []string{"quality", "latency"},
		TargetMetric: "quality",
	}

	err := manager.Create(context.Background(), exp)
	require.NoError(t, err)
	assert.NotEmpty(t, exp.ID)
	assert.Equal(t, ExperimentStatusDraft, exp.Status)
}

func TestInMemoryExperimentManager_Lifecycle(t *testing.T) {
	manager := NewInMemoryExperimentManager(nil)

	exp := &Experiment{
		Name: "Test Experiment",
		Variants: []*Variant{
			{Name: "Control", IsControl: true},
			{Name: "Treatment"},
		},
	}

	require.NoError(t, manager.Create(context.Background(), exp))
	assert.Equal(t, ExperimentStatusDraft, exp.Status)

	// Start
	require.NoError(t, manager.Start(context.Background(), exp.ID))
	exp, _ = manager.Get(context.Background(), exp.ID)
	assert.Equal(t, ExperimentStatusRunning, exp.Status)

	// Pause
	require.NoError(t, manager.Pause(context.Background(), exp.ID))
	exp, _ = manager.Get(context.Background(), exp.ID)
	assert.Equal(t, ExperimentStatusPaused, exp.Status)

	// Resume
	require.NoError(t, manager.Start(context.Background(), exp.ID))

	// Complete
	require.NoError(t, manager.Complete(context.Background(), exp.ID, exp.Variants[1].ID))
	exp, _ = manager.Get(context.Background(), exp.ID)
	assert.Equal(t, ExperimentStatusCompleted, exp.Status)
	assert.Equal(t, exp.Variants[1].ID, exp.Winner)
}

func TestInMemoryExperimentManager_AssignVariant(t *testing.T) {
	manager := NewInMemoryExperimentManager(nil)

	exp := &Experiment{
		Name: "Test Experiment",
		Variants: []*Variant{
			{Name: "Control", IsControl: true},
			{Name: "Treatment"},
		},
		TrafficSplit: map[string]float64{},
	}

	require.NoError(t, manager.Create(context.Background(), exp))
	require.NoError(t, manager.Start(context.Background(), exp.ID))

	// Assign variant to user
	variant, err := manager.AssignVariant(context.Background(), exp.ID, "user-1")
	require.NoError(t, err)
	assert.NotNil(t, variant)

	// Same user gets same variant
	variant2, err := manager.AssignVariant(context.Background(), exp.ID, "user-1")
	require.NoError(t, err)
	assert.Equal(t, variant.ID, variant2.ID)
}

func TestInMemoryExperimentManager_RecordMetric(t *testing.T) {
	manager := NewInMemoryExperimentManager(nil)

	exp := &Experiment{
		Name: "Test Experiment",
		Variants: []*Variant{
			{Name: "Control", IsControl: true},
			{Name: "Treatment"},
		},
	}

	require.NoError(t, manager.Create(context.Background(), exp))
	require.NoError(t, manager.Start(context.Background(), exp.ID))

	// Record metrics
	for i := 0; i < 50; i++ {
		err := manager.RecordMetric(context.Background(), exp.ID, exp.Variants[0].ID, "quality", 0.7)
		require.NoError(t, err)
		err = manager.RecordMetric(context.Background(), exp.ID, exp.Variants[1].ID, "quality", 0.8)
		require.NoError(t, err)
	}

	// Get results
	results, err := manager.GetResults(context.Background(), exp.ID)
	require.NoError(t, err)
	assert.Equal(t, 100, results.TotalSamples)
	assert.Len(t, results.VariantResults, 2)
}

func TestInMemoryContinuousEvaluator_CreateRun(t *testing.T) {
	evaluator := NewInMemoryContinuousEvaluator(nil, nil, nil, nil)

	dataset := &Dataset{Name: "Test Dataset", Type: DatasetTypeGolden}
	require.NoError(t, evaluator.CreateDataset(context.Background(), dataset))

	samples := []*DatasetSample{
		{Input: "Test 1", ExpectedOutput: "Output 1"},
		{Input: "Test 2", ExpectedOutput: "Output 2"},
	}
	require.NoError(t, evaluator.AddSamples(context.Background(), dataset.ID, samples))

	run := &EvaluationRun{
		Name:       "Test Run",
		Dataset:    dataset.ID,
		PromptName: "test-prompt",
		Metrics:    []string{"accuracy"},
	}

	err := evaluator.CreateRun(context.Background(), run)
	require.NoError(t, err)
	assert.NotEmpty(t, run.ID)
	assert.Equal(t, EvaluationStatusPending, run.Status)
}

func TestInMemoryAlertManager_Create(t *testing.T) {
	manager := NewInMemoryAlertManager(nil)

	alert := &Alert{
		Type:     AlertTypeRegression,
		Severity: AlertSeverityWarning,
		Message:  "Pass rate dropped by 5%",
		Source:   "evaluation",
	}

	err := manager.Create(context.Background(), alert)
	require.NoError(t, err)
	assert.NotEmpty(t, alert.ID)
}

func TestInMemoryAlertManager_Subscribe(t *testing.T) {
	manager := NewInMemoryAlertManager(nil)

	received := make(chan *Alert, 1)
	err := manager.Subscribe(context.Background(), func(alert *Alert) error {
		received <- alert
		return nil
	})
	require.NoError(t, err)

	alert := &Alert{
		Type:     AlertTypeThreshold,
		Severity: AlertSeverityCritical,
		Message:  "Latency exceeded threshold",
	}

	err = manager.Create(context.Background(), alert)
	require.NoError(t, err)

	select {
	case a := <-received:
		assert.Equal(t, alert.ID, a.ID)
	case <-time.After(time.Second):
		t.Fatal("Did not receive alert")
	}
}

func TestLLMOpsSystem_Initialize(t *testing.T) {
	config := DefaultLLMOpsConfig()
	system := NewLLMOpsSystem(config, nil)

	err := system.Initialize()
	require.NoError(t, err)

	assert.NotNil(t, system.GetPromptRegistry())
	assert.NotNil(t, system.GetExperimentManager())
	assert.NotNil(t, system.GetEvaluator())
	assert.NotNil(t, system.GetAlertManager())
}

func TestLLMOpsSystem_CreatePromptExperiment(t *testing.T) {
	config := DefaultLLMOpsConfig()
	system := NewLLMOpsSystem(config, nil)
	require.NoError(t, system.Initialize())

	control := &PromptVersion{
		Name:    "control-prompt",
		Version: "1.0.0",
		Content: "Be helpful",
	}

	treatment := &PromptVersion{
		Name:    "treatment-prompt",
		Version: "1.0.0",
		Content: "Be very helpful and concise",
	}

	exp, err := system.CreatePromptExperiment(context.Background(), "Prompt A/B Test", control, treatment, 0.5)
	require.NoError(t, err)
	assert.NotEmpty(t, exp.ID)
	assert.Len(t, exp.Variants, 2)
}

func TestPromptVersionComparator_Compare(t *testing.T) {
	registry := NewInMemoryPromptRegistry(nil)

	v1 := &PromptVersion{
		Name:    "test",
		Version: "1.0.0",
		Content: "Line 1\nLine 2",
		Variables: []PromptVariable{
			{Name: "name", Type: "string"},
		},
	}

	v2 := &PromptVersion{
		Name:    "test",
		Version: "2.0.0",
		Content: "Line 1\nLine 2\nLine 3",
		Variables: []PromptVariable{
			{Name: "name", Type: "string"},
			{Name: "age", Type: "int"},
		},
	}

	require.NoError(t, registry.Create(context.Background(), v1))
	require.NoError(t, registry.Create(context.Background(), v2))

	comparator := NewPromptVersionComparator(registry, nil)
	diff, err := comparator.Compare(context.Background(), "test", "1.0.0", "2.0.0")
	require.NoError(t, err)

	assert.Contains(t, diff.AddedVars, "age")
	assert.NotEmpty(t, diff.ContentDiff)
}

func TestDefaultLLMOpsConfig(t *testing.T) {
	config := DefaultLLMOpsConfig()

	assert.True(t, config.EnableAutoEvaluation)
	assert.Greater(t, config.MinSamplesForSignif, 0)
	assert.NotEmpty(t, config.AlertThresholds)
}

func TestEvaluationRun_Status(t *testing.T) {
	run := &EvaluationRun{
		ID:     "run-1",
		Name:   "Test Run",
		Status: EvaluationStatusPending,
	}

	assert.Equal(t, EvaluationStatusPending, run.Status)

	run.Status = EvaluationStatusRunning
	assert.Equal(t, EvaluationStatusRunning, run.Status)

	run.Status = EvaluationStatusCompleted
	assert.Equal(t, EvaluationStatusCompleted, run.Status)
}

func TestExperimentStatus_Transitions(t *testing.T) {
	tests := []struct {
		from  ExperimentStatus
		to    ExperimentStatus
		valid bool
	}{
		{ExperimentStatusDraft, ExperimentStatusRunning, true},
		{ExperimentStatusRunning, ExperimentStatusPaused, true},
		{ExperimentStatusPaused, ExperimentStatusRunning, true},
		{ExperimentStatusRunning, ExperimentStatusCompleted, true},
		{ExperimentStatusCompleted, ExperimentStatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			manager := NewInMemoryExperimentManager(nil)

			exp := &Experiment{
				Name: "Test",
				Variants: []*Variant{
					{Name: "A"},
					{Name: "B"},
				},
			}

			require.NoError(t, manager.Create(context.Background(), exp))

			// Set initial status
			if tt.from == ExperimentStatusRunning {
				manager.Start(context.Background(), exp.ID)
			} else if tt.from == ExperimentStatusPaused {
				manager.Start(context.Background(), exp.ID)
				manager.Pause(context.Background(), exp.ID)
			} else if tt.from == ExperimentStatusCompleted {
				manager.Start(context.Background(), exp.ID)
				manager.Complete(context.Background(), exp.ID, "")
			}

			// Try transition
			var err error
			switch tt.to {
			case ExperimentStatusRunning:
				err = manager.Start(context.Background(), exp.ID)
			case ExperimentStatusPaused:
				err = manager.Pause(context.Background(), exp.ID)
			case ExperimentStatusCompleted:
				err = manager.Complete(context.Background(), exp.ID, "")
			}

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
