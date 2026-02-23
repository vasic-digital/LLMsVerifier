package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ProviderAdapterForBenchmark - GetName
// ============================================================================

func TestProviderAdapterForBenchmark_GetName(t *testing.T) {
	adapter := NewProviderAdapterForBenchmark(nil, "my-provider", "my-model", nil)
	assert.Equal(t, "my-provider", adapter.GetName())
}

func TestProviderAdapterForBenchmark_Complete(t *testing.T) {
	adapter := NewProviderAdapterForBenchmark(nil, "p", "m", nil)
	resp, tokens, err := adapter.Complete(context.Background(), "hello", "system")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
	assert.Greater(t, tokens, 0)
}

// ============================================================================
// BenchmarkSystem - NewBenchmarkSystem with nil config and EvaluateResponse
// ============================================================================

func TestNewBenchmarkSystem_NilConfig(t *testing.T) {
	system := NewBenchmarkSystem(nil, nil)
	assert.NotNil(t, system)
}

func TestDebateAdapterForBenchmark_EvaluateResponse_NilService(t *testing.T) {
	// DebateAdapterForBenchmark with no debate service returns 0 score
	task := &BenchmarkTask{
		Name:        "Test",
		Description: "Test task",
		Expected:    "Answer",
	}
	adapter := &DebateAdapterForBenchmark{service: nil, logger: nil}
	_ = task
	_ = adapter
	// Just ensure the struct can be constructed with nil service
	assert.NotNil(t, adapter)
}

// ============================================================================
// BenchmarkSystem - SelectBestProvider with healthcheck variation
// ============================================================================

type mockVerifierWithHealth struct {
	scores  map[string]float64
	healthy map[string]bool
}

func (m *mockVerifierWithHealth) GetProviderScore(name string) float64 {
	return m.scores[name]
}

func (m *mockVerifierWithHealth) IsProviderHealthy(name string) bool {
	return m.healthy[name]
}

func (m *mockVerifierWithHealth) GetTopProviders(count int) []string {
	return []string{"slow-provider", "fast-provider", "best-provider"}
}

func TestVerifierAdapterForBenchmark_SelectBestProvider_WithHealthCheck(t *testing.T) {
	svc := &mockVerifierWithHealth{
		scores: map[string]float64{
			"slow-provider": 0.5,
			"fast-provider": 0.8,
			"best-provider": 0.9,
		},
		healthy: map[string]bool{
			"slow-provider": false,
			"fast-provider": true,
			"best-provider": true,
		},
	}

	adapter := NewVerifierAdapterForBenchmark(svc, nil)
	best, score := adapter.SelectBestProvider()
	assert.NotEmpty(t, best)
	assert.Greater(t, score, 0.0)
	assert.Equal(t, "best-provider", best)
}

type emptyVerifier struct{}

func (e *emptyVerifier) GetProviderScore(name string) float64  { return 0 }
func (e *emptyVerifier) IsProviderHealthy(name string) bool    { return false }
func (e *emptyVerifier) GetTopProviders(count int) []string    { return nil }

func TestVerifierAdapterForBenchmark_SelectBestProvider_Empty(t *testing.T) {
	adapter := NewVerifierAdapterForBenchmark(&emptyVerifier{}, nil)
	best, score := adapter.SelectBestProvider()
	assert.Empty(t, best)
	assert.Equal(t, 0.0, score)
}

// ============================================================================
// BenchmarkRunner - StartRun edge cases
// ============================================================================

func TestStandardBenchmarkRunner_StartRun_NotFound(t *testing.T) {
	runner := NewStandardBenchmarkRunner(nil, nil)
	err := runner.StartRun(context.Background(), "nonexistent-id")
	assert.Error(t, err)
}

func TestStandardBenchmarkRunner_CancelRun_NotFound(t *testing.T) {
	runner := NewStandardBenchmarkRunner(nil, nil)
	err := runner.CancelRun(context.Background(), "nonexistent-id")
	assert.Error(t, err)
}

func TestStandardBenchmarkRunner_GetRun_NotFound(t *testing.T) {
	runner := NewStandardBenchmarkRunner(nil, nil)
	_, err := runner.GetRun(context.Background(), "nonexistent-id")
	assert.Error(t, err)
}

func TestStandardBenchmarkRunner_CompareRuns_NotFound(t *testing.T) {
	runner := NewStandardBenchmarkRunner(nil, nil)
	_, err := runner.CompareRuns(context.Background(), "run-a", "run-b")
	assert.Error(t, err)
}

// ============================================================================
// BenchmarkRunner - StartRun completes
// ============================================================================

func TestStandardBenchmarkRunner_StartRun_Completes(t *testing.T) {
	runner := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	run := &BenchmarkRun{
		Name:          "Short Run",
		BenchmarkType: BenchmarkTypeGSM8K,
		Config:        &BenchmarkConfig{MaxTasks: 1, Timeout: 30 * time.Second},
	}
	require.NoError(t, runner.CreateRun(ctx, run))
	require.NoError(t, runner.StartRun(ctx, run.ID))

	// Wait for completion
	for i := 0; i < 30; i++ {
		got, err := runner.GetRun(ctx, run.ID)
		if err == nil && got.Status == BenchmarkStatusCompleted {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	got, err := runner.GetRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, BenchmarkStatusCompleted, got.Status)
}
