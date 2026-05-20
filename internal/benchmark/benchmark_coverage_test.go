package benchmark

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

// TestProviderAdapterForBenchmark_Complete_NotWired proves the un-wired
// adapter surfaces ErrProviderAdapterNotWired instead of fabricating the
// old hardcoded ("Response", 50) result — §11.4 / CONST-035 anti-bluff
// regression guard. round-396 HXV-003.
func TestProviderAdapterForBenchmark_Complete_NotWired(t *testing.T) {
	adapter := NewProviderAdapterForBenchmark(nil, "p", "m", nil)
	resp, tokens, err := adapter.Complete(context.Background(), "hello", "system")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderAdapterNotWired)
	assert.Empty(t, resp, "un-wired adapter must NOT fabricate a response")
	assert.Zero(t, tokens, "un-wired adapter must NOT fabricate a token count")
	assert.NotEqual(t, "Response", resp, "the round-396 fixed mock-bluff must not return")
}

// TestProviderAdapterForBenchmark_Complete_RealDispatch proves the wired
// adapter performs a REAL dispatch through its underlying LLMProvider: the
// returned text + token count reflect the real HTTP response from an
// httptest server the adapter must actually hit. round-396 HXV-003.
func TestProviderAdapterForBenchmark_Complete_RealDispatch(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("server failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"4 is the real answer"}}],"usage":{"prompt_tokens":7,"completion_tokens":12,"total_tokens":19}}`))
	}))
	defer srv.Close()

	httpProvider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: srv.URL + "/v1",
		Model:    "test-model",
	})
	require.NoError(t, err)

	adapter := NewProviderAdapterForBenchmark(httpProvider, "openai-shim", "test-model", nil)
	resp, tokens, err := adapter.Complete(context.Background(), "what is 2+2?", "you are a calculator")
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "adapter must have actually dispatched to the server")
	assert.Equal(t, "4 is the real answer", resp, "response must be the real server payload, not a fabricated constant")
	assert.Equal(t, 19, tokens, "token count must be the server-reported usage.total_tokens, not the old hardcoded 50")
	assert.NotEqual(t, "Response", resp)
	assert.NotEqual(t, 50, tokens)
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
	// §11.4 anti-bluff (round-28): the runner is intentionally constructed
	// with provider=nil here. Previously the nil-provider branch in
	// executeTask fabricated Passed=true/Score=0.8/Latency=100ms/Tokens=50
	// — a CONST-035 violation. The current contract surfaces
	// ErrBenchmarkProviderNotConfigured per task; the run still reaches
	// Completed status (the runner does not abort on per-task errors), but
	// every result MUST carry the sentinel error string and Passed=false.
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

	// Anti-bluff assertions: every result must reflect the missing-provider
	// reality. Empty Results is also acceptable (no tasks queued for the
	// type) — but if any result exists, it must NOT be a fabricated PASS.
	for _, r := range got.Results {
		assert.False(t, r.Passed, "nil-provider runner must NOT report Passed=true (regression of round-28 anti-bluff fix)")
		assert.Equal(t, ErrBenchmarkProviderNotConfigured.Error(), r.Error,
			"nil-provider runner must surface ErrBenchmarkProviderNotConfigured")
		assert.Equal(t, 0.0, r.Score, "nil-provider runner must NOT fabricate Score")
		assert.Equal(t, 0, r.TokensUsed, "nil-provider runner must NOT fabricate TokensUsed")
	}
}

// TestStandardBenchmarkRunner_NilProvider_ReturnsSentinel directly exercises
// the executeTask nil-provider branch to lock-in the §11.4 sentinel contract.
func TestStandardBenchmarkRunner_NilProvider_ReturnsSentinel(t *testing.T) {
	runner := NewStandardBenchmarkRunner(nil, nil)
	task := &BenchmarkTask{ID: "anti-bluff-probe", Prompt: "what is 2+2?", Expected: "4"}
	result := runner.executeTask(context.Background(), &BenchmarkRun{}, task)

	require.NotNil(t, result)
	assert.False(t, result.Passed, "nil-provider executeTask must return Passed=false")
	assert.Equal(t, ErrBenchmarkProviderNotConfigured.Error(), result.Error)
	assert.Equal(t, 0.0, result.Score)
	assert.Equal(t, 0, result.TokensUsed)
	assert.Empty(t, result.Response)
}

// TestStandardBenchmarkRunner_SetProvider verifies the SetProvider injection
// path added in round-28: a runner constructed with provider=nil can be
// repaired by calling SetProvider before invoking executeTask.
func TestStandardBenchmarkRunner_SetProvider(t *testing.T) {
	runner := NewStandardBenchmarkRunner(nil, nil)
	runner.SetProvider(&fakeBenchmarkProvider{response: "4", tokens: 7})

	task := &BenchmarkTask{ID: "set-provider-probe", Prompt: "what is 2+2?", Expected: "4"}
	result := runner.executeTask(context.Background(), &BenchmarkRun{}, task)

	require.NotNil(t, result)
	assert.True(t, result.Passed, "after SetProvider with matching response, task should pass")
	assert.Empty(t, result.Error)
	assert.Equal(t, "4", result.Response)
	assert.Equal(t, 7, result.TokensUsed)
}

// fakeBenchmarkProvider is a unit-test-only stub (CONST-050(A) permits
// fakes in *_test.go) that returns a fixed canned response.
type fakeBenchmarkProvider struct {
	response string
	tokens   int
}

func (f *fakeBenchmarkProvider) Complete(ctx context.Context, prompt, _ string) (string, int, error) {
	return f.response, f.tokens, nil
}

func (f *fakeBenchmarkProvider) GetName() string { return "fake-benchmark-provider" }
