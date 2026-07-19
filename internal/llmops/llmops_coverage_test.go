package llmops

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// InMemoryContinuousEvaluator - GetRun, ListRuns, StartRun
// ============================================================================

func TestInMemoryContinuousEvaluator_GetRun(t *testing.T) {
	evaluator := NewInMemoryContinuousEvaluator(nil, nil, nil, nil)
	ctx := context.Background()

	dataset := &Dataset{Name: "Test Dataset"}
	require.NoError(t, evaluator.CreateDataset(ctx, dataset))

	run := &EvaluationRun{
		Name:       "Test Run",
		Dataset:    dataset.ID,
		PromptName: "prompt-1",
		Metrics:    []string{"accuracy"},
	}
	require.NoError(t, evaluator.CreateRun(ctx, run))

	got, err := evaluator.GetRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, run.ID, got.ID)
	assert.Equal(t, "Test Run", got.Name)
}

func TestInMemoryContinuousEvaluator_GetRun_NotFound(t *testing.T) {
	evaluator := NewInMemoryContinuousEvaluator(nil, nil, nil, nil)
	ctx := context.Background()

	_, err := evaluator.GetRun(ctx, "nonexistent-run")
	assert.Error(t, err)
}

func TestInMemoryContinuousEvaluator_ListRuns(t *testing.T) {
	evaluator := NewInMemoryContinuousEvaluator(nil, nil, nil, nil)
	ctx := context.Background()

	dataset := &Dataset{Name: "D"}
	require.NoError(t, evaluator.CreateDataset(ctx, dataset))

	for i := 0; i < 3; i++ {
		run := &EvaluationRun{Name: "Run", Dataset: dataset.ID, PromptName: "p"}
		require.NoError(t, evaluator.CreateRun(ctx, run))
	}

	runs, err := evaluator.ListRuns(ctx)
	require.NoError(t, err)
	assert.Len(t, runs, 3)
}

func TestInMemoryContinuousEvaluator_StartRun(t *testing.T) {
	evaluator := NewInMemoryContinuousEvaluator(nil, nil, nil, nil)
	ctx := context.Background()

	dataset := &Dataset{Name: "D"}
	require.NoError(t, evaluator.CreateDataset(ctx, dataset))

	samples := []*DatasetSample{
		{Input: "Test 1", ExpectedOutput: "Output 1"},
	}
	require.NoError(t, evaluator.AddSamples(ctx, dataset.ID, samples))

	run := &EvaluationRun{Name: "Run", Dataset: dataset.ID, PromptName: "p"}
	require.NoError(t, evaluator.CreateRun(ctx, run))

	// StartRun triggers background goroutine
	err := evaluator.StartRun(ctx, run.ID)
	require.NoError(t, err)

	got, err := evaluator.GetRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, EvaluationStatusRunning, got.Status)
}

func TestInMemoryContinuousEvaluator_StartRun_NotFound(t *testing.T) {
	evaluator := NewInMemoryContinuousEvaluator(nil, nil, nil, nil)
	err := evaluator.StartRun(context.Background(), "nonexistent")
	assert.Error(t, err)
}

// TestInMemoryContinuousEvaluator_StartRun_CompletesInBackground:
// per round-18 §11.4 fix to evaluator.executeRun (commit
// `348ecd55`), the evaluator refuses to score with a hardcoded 0.8
// when debateEval is nil; instead the run terminates with
// EvaluationStatusFailed and a `error_debate_eval_nil: -1` metric.
// The previous test asserted EvaluationStatusCompleted — that was
// CERTIFYING the hardcoded-score bluff (per CONST-035 §11.4.4).
// Tightened to assert the honest "Failed" terminal state when
// running without a debateEval injection.
func TestInMemoryContinuousEvaluator_StartRun_CompletesInBackground(t *testing.T) {
	evaluator := NewInMemoryContinuousEvaluator(nil, nil, nil, nil)
	ctx := context.Background()

	dataset := &Dataset{Name: "D"}
	require.NoError(t, evaluator.CreateDataset(ctx, dataset))

	samples := []*DatasetSample{
		{Input: "Q1", ExpectedOutput: "A1"},
		{Input: "Q2", ExpectedOutput: "A2"},
	}
	require.NoError(t, evaluator.AddSamples(ctx, dataset.ID, samples))

	run := &EvaluationRun{Name: "Run", Dataset: dataset.ID}
	require.NoError(t, evaluator.CreateRun(ctx, run))
	require.NoError(t, evaluator.StartRun(ctx, run.ID))

	// Wait for background goroutine to reach terminal state. With nil
	// debateEval the round-18 fix terminates as Failed.
	var got *EvaluationRun
	var err error
	for i := 0; i < 50; i++ {
		got, err = evaluator.GetRun(ctx, run.ID)
		require.NoError(t, err)
		if got.Status == EvaluationStatusFailed || got.Status == EvaluationStatusCompleted {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	assert.Equal(t, EvaluationStatusFailed, got.Status,
		"expect Failed terminal state — running without debateEval injection now refuses per §11.4 (was: silent hardcoded 0.8 score)")
	require.NotNil(t, got.Results)
	assert.Equal(t, 2, got.Results.TotalSamples)
	assert.Equal(t, float64(-1), got.Results.Metrics["error_debate_eval_nil"])
}

func TestInMemoryContinuousEvaluator_CheckForRegressions_WithAlertManager(t *testing.T) {
	alertManager := NewInMemoryAlertManager(nil)
	config := DefaultLLMOpsConfig()
	// Set low threshold so regression is triggered (pass rate < 1 - 0.1 = 0.9)
	config.AlertThresholds["pass_rate_drop"] = 0.1

	evaluator := NewInMemoryContinuousEvaluator(config, alertManager, nil, nil)
	ctx := context.Background()

	dataset := &Dataset{Name: "D"}
	require.NoError(t, evaluator.CreateDataset(ctx, dataset))

	// Add 10 samples
	for i := 0; i < 10; i++ {
		samples := []*DatasetSample{{Input: "q", ExpectedOutput: "a"}}
		require.NoError(t, evaluator.AddSamples(ctx, dataset.ID, samples))
	}

	run := &EvaluationRun{Name: "Run", Dataset: dataset.ID}
	require.NoError(t, evaluator.CreateRun(ctx, run))
	require.NoError(t, evaluator.StartRun(ctx, run.ID))

	// Wait for completion
	for i := 0; i < 50; i++ {
		got, _ := evaluator.GetRun(ctx, run.ID)
		if got != nil && got.Status == EvaluationStatusCompleted {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ============================================================================
// InMemoryExperimentManager - List
// ============================================================================

func TestInMemoryExperimentManager_List(t *testing.T) {
	manager := NewInMemoryExperimentManager(nil)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		exp := &Experiment{
			Name: "Experiment",
			Variants: []*Variant{
				{Name: "Control", IsControl: true},
				{Name: "Treatment"},
			},
		}
		require.NoError(t, manager.Create(ctx, exp))
	}

	list, err := manager.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestInMemoryExperimentManager_List_Empty(t *testing.T) {
	manager := NewInMemoryExperimentManager(nil)
	ctx := context.Background()

	list, err := manager.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}

// ============================================================================
// InMemoryPromptRegistry - List
// ============================================================================

func TestInMemoryPromptRegistry_List(t *testing.T) {
	registry := NewInMemoryPromptRegistry(nil)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		prompt := &PromptVersion{
			Name:    "prompt",
			Version: fmt.Sprintf("1.%d.0", i),
			Content: "Test content",
		}
		require.NoError(t, registry.Create(ctx, prompt))
	}

	list, err := registry.List(ctx, "prompt")
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestInMemoryPromptRegistry_List_Empty(t *testing.T) {
	registry := NewInMemoryPromptRegistry(nil)
	ctx := context.Background()

	list, err := registry.List(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, list)
}

// ============================================================================
// LLMOpsSystem - IsInitialized
// ============================================================================

func TestLLMOpsSystem_IsInitialized_False(t *testing.T) {
	system := NewLLMOpsSystem(nil, nil)
	assert.False(t, system.IsInitialized())
}

func TestLLMOpsSystem_IsInitialized_True(t *testing.T) {
	system := NewLLMOpsSystem(nil, nil)
	require.NoError(t, system.Initialize())
	assert.True(t, system.IsInitialized())
}

// ============================================================================
// InMemoryContinuousEvaluator - GetRun edge cases
// ============================================================================

func TestInMemoryContinuousEvaluator_Resolve_Alert(t *testing.T) {
	alertMgr := NewInMemoryAlertManager(nil)
	ctx := context.Background()

	alert := &Alert{
		Type:    AlertTypeRegression,
		Message: "Regression detected",
	}
	require.NoError(t, alertMgr.Create(ctx, alert))

	// Resolve it
	err := alertMgr.Resolve(ctx, alert.ID)
	require.NoError(t, err)

	got, err := alertMgr.Get(ctx, alert.ID)
	require.NoError(t, err)
	assert.True(t, got.Resolved)
}

// TestInMemoryContinuousEvaluator_ConcurrentStartRunGetRun_NoRace is the
// explicit regression guard for the executeRun↔GetRun data race (fixed by
// moving the nil-debateEval `run.Status` write inside the mutex). It hammers
// StartRun (whose background executeRun mutates run.Status) and GetRun (which
// reads run.Status/Results) concurrently across many runs. Under `-race` this
// reproduces the pre-fix write/read race; it MUST stay green.
func TestInMemoryContinuousEvaluator_ConcurrentStartRunGetRun_NoRace(t *testing.T) {
	evaluator := NewInMemoryContinuousEvaluator(nil, nil, nil, nil)
	ctx := context.Background()

	dataset := &Dataset{Name: "D"}
	require.NoError(t, evaluator.CreateDataset(ctx, dataset))
	require.NoError(t, evaluator.AddSamples(ctx, dataset.ID,
		[]*DatasetSample{{Input: "Q", ExpectedOutput: "A"}}))

	const n = 50
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		run := &EvaluationRun{Name: "Run", Dataset: dataset.ID}
		require.NoError(t, evaluator.CreateRun(ctx, run))
		ids[i] = run.ID
	}

	var wg sync.WaitGroup
	for _, id := range ids {
		id := id
		wg.Add(2)
		// Writer: StartRun launches executeRun (background) which mutates run.Status.
		go func() { defer wg.Done(); _ = evaluator.StartRun(ctx, id) }()
		// Reader: GetRun reads run.Status/Results concurrently with that mutation.
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				if r, err := evaluator.GetRun(ctx, id); err == nil {
					_ = r.Status
				}
			}
		}()
	}
	wg.Wait()

	// Every run reaches the honest terminal Failed state (debateEval is nil).
	for _, id := range ids {
		var got *EvaluationRun
		for i := 0; i < 50; i++ {
			var err error
			got, err = evaluator.GetRun(ctx, id)
			require.NoError(t, err)
			if got.Status == EvaluationStatusFailed || got.Status == EvaluationStatusCompleted {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		require.NotNil(t, got)
		assert.Equal(t, EvaluationStatusFailed, got.Status)
	}
}
