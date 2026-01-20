package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMProviderForBenchmark for testing
type MockLLMProviderForBenchmark struct {
	responses map[string]string
}

func NewMockLLMProviderForBenchmark() *MockLLMProviderForBenchmark {
	return &MockLLMProviderForBenchmark{
		responses: map[string]string{
			"default":     "The answer is B",
			"binary":      "B",
			"math":        "18",
			"code":        "def has_close_elements(numbers, threshold): return any(abs(a-b) < threshold for i, a in enumerate(numbers) for b in numbers[i+1:])",
		},
	}
}

func (m *MockLLMProviderForBenchmark) Complete(ctx context.Context, prompt, systemPrompt string) (string, int, error) {
	if containsStr(prompt, "binary search") || containsStr(prompt, "time complexity") {
		return m.responses["binary"], 10, nil
	}
	if containsStr(prompt, "duck eggs") || containsStr(prompt, "farmers' market") {
		return m.responses["math"], 15, nil
	}
	if containsStr(prompt, "has_close_elements") {
		return m.responses["code"], 100, nil
	}
	return m.responses["default"], 20, nil
}

func (m *MockLLMProviderForBenchmark) GetName() string {
	return "mock-provider"
}

// MockDebateEvaluatorForBenchmark for testing
type MockDebateEvaluatorForBenchmark struct{}

func (m *MockDebateEvaluatorForBenchmark) EvaluateResponse(ctx context.Context, task *BenchmarkTask, response string) (float64, bool, error) {
	if len(response) > 0 {
		return 0.8, true, nil
	}
	return 0.3, false, nil
}

func TestStandardBenchmarkRunner_ListBenchmarks(t *testing.T) {
	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)

	benchmarks, err := runner.ListBenchmarks(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, benchmarks)

	types := make(map[BenchmarkType]bool)
	for _, b := range benchmarks {
		types[b.Type] = true
	}
	assert.True(t, types[BenchmarkTypeSWEBench])
	assert.True(t, types[BenchmarkTypeHumanEval])
	assert.True(t, types[BenchmarkTypeMMLU])
	assert.True(t, types[BenchmarkTypeGSM8K])
}

func TestStandardBenchmarkRunner_GetTasks(t *testing.T) {
	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)

	benchmarks, _ := runner.ListBenchmarks(context.Background())
	require.NotEmpty(t, benchmarks)

	tasks, err := runner.GetTasks(context.Background(), benchmarks[0].ID, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tasks)
}

func TestStandardBenchmarkRunner_GetTasks_WithFilter(t *testing.T) {
	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)

	config := &BenchmarkConfig{
		Difficulties: []DifficultyLevel{DifficultyEasy},
		MaxTasks:     2,
	}

	tasks, err := runner.GetTasks(context.Background(), "swe-bench-lite", config)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(tasks), 2)

	for _, task := range tasks {
		assert.Equal(t, DifficultyEasy, task.Difficulty)
	}
}

func TestStandardBenchmarkRunner_CreateAndStartRun(t *testing.T) {
	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)

	run := &BenchmarkRun{
		Name:          "Test Run",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "mock-provider",
		Config: &BenchmarkConfig{
			MaxTasks:    2,
			Timeout:     30 * time.Second,
			Concurrency: 1,
		},
	}

	err := runner.CreateRun(context.Background(), run)
	require.NoError(t, err)
	assert.NotEmpty(t, run.ID)
	assert.Equal(t, BenchmarkStatusPending, run.Status)

	err = runner.StartRun(context.Background(), run.ID)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	run, err = runner.GetRun(context.Background(), run.ID)
	require.NoError(t, err)
	assert.Equal(t, BenchmarkStatusCompleted, run.Status)
	assert.NotNil(t, run.Summary)
}

func TestStandardBenchmarkRunner_RunWithDebateEvaluation(t *testing.T) {
	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)
	runner.SetDebateEvaluator(&MockDebateEvaluatorForBenchmark{})

	run := &BenchmarkRun{
		Name:          "Debate Eval Test",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "mock-provider",
		Config: &BenchmarkConfig{
			MaxTasks:         1,
			Timeout:          30 * time.Second,
			UseDebateForEval: true,
		},
	}

	require.NoError(t, runner.CreateRun(context.Background(), run))
	require.NoError(t, runner.StartRun(context.Background(), run.ID))

	time.Sleep(time.Second)

	run, _ = runner.GetRun(context.Background(), run.ID)
	assert.Equal(t, BenchmarkStatusCompleted, run.Status)
}

func TestStandardBenchmarkRunner_CompareRuns(t *testing.T) {
	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)

	run1 := &BenchmarkRun{
		Name:          "Run 1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "provider-1",
		Config:        &BenchmarkConfig{MaxTasks: 2, Timeout: 30 * time.Second},
	}
	run2 := &BenchmarkRun{
		Name:          "Run 2",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "provider-2",
		Config:        &BenchmarkConfig{MaxTasks: 2, Timeout: 30 * time.Second},
	}

	require.NoError(t, runner.CreateRun(context.Background(), run1))
	require.NoError(t, runner.CreateRun(context.Background(), run2))

	require.NoError(t, runner.StartRun(context.Background(), run1.ID))
	require.NoError(t, runner.StartRun(context.Background(), run2.ID))

	time.Sleep(2 * time.Second)

	comparison, err := runner.CompareRuns(context.Background(), run1.ID, run2.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, comparison.Summary)
}

func TestStandardBenchmarkRunner_ListRuns(t *testing.T) {
	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)

	for i := 0; i < 3; i++ {
		run := &BenchmarkRun{
			Name:          "Test Run",
			BenchmarkType: BenchmarkTypeMMLU,
			Config:        &BenchmarkConfig{MaxTasks: 1},
		}
		require.NoError(t, runner.CreateRun(context.Background(), run))
	}

	runs, err := runner.ListRuns(context.Background(), nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(runs), 3)

	filter := &RunFilter{Status: BenchmarkStatusPending}
	runs, err = runner.ListRuns(context.Background(), filter)
	require.NoError(t, err)
	for _, run := range runs {
		assert.Equal(t, BenchmarkStatusPending, run.Status)
	}
}

func TestStandardBenchmarkRunner_CancelRun(t *testing.T) {
	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)

	run := &BenchmarkRun{
		Name:          "To Cancel",
		BenchmarkType: BenchmarkTypeMMLU,
		Config:        &BenchmarkConfig{MaxTasks: 10, Timeout: time.Minute},
	}

	require.NoError(t, runner.CreateRun(context.Background(), run))

	err := runner.CancelRun(context.Background(), run.ID)
	require.NoError(t, err)

	run, _ = runner.GetRun(context.Background(), run.ID)
	assert.Equal(t, BenchmarkStatusCancelled, run.Status)
}

func TestStandardBenchmarkRunner_AddCustomBenchmark(t *testing.T) {
	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)

	customBenchmark := &Benchmark{
		ID:          "custom-1",
		Type:        BenchmarkTypeCustom,
		Name:        "Custom Benchmark",
		Description: "Custom test benchmark",
		Version:     "1.0.0",
	}

	customTasks := []*BenchmarkTask{
		{
			ID:         "custom-task-1",
			Name:       "Custom Task",
			Prompt:     "What is 1+1?",
			Expected:   "2",
			Difficulty: DifficultyEasy,
		},
	}

	runner.AddBenchmark(customBenchmark, customTasks)

	tasks, err := runner.GetTasks(context.Background(), "custom-1", nil)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestBenchmarkSummary_Calculation(t *testing.T) {
	results := []*BenchmarkResult{
		{TaskID: "1", Passed: true, Score: 1.0, Latency: 100 * time.Millisecond, TokensUsed: 50},
		{TaskID: "2", Passed: true, Score: 0.8, Latency: 200 * time.Millisecond, TokensUsed: 60},
		{TaskID: "3", Passed: false, Score: 0.3, Latency: 150 * time.Millisecond, TokensUsed: 70},
	}

	tasks := []*BenchmarkTask{
		{ID: "1", Difficulty: DifficultyEasy, Tags: []string{"math"}},
		{ID: "2", Difficulty: DifficultyMedium, Tags: []string{"math", "logic"}},
		{ID: "3", Difficulty: DifficultyHard, Tags: []string{"code"}},
	}

	provider := NewMockLLMProviderForBenchmark()
	runner := NewStandardBenchmarkRunner(provider, nil)

	summary := runner.calculateSummary(results, tasks)

	assert.Equal(t, 3, summary.TotalTasks)
	assert.Equal(t, 2, summary.PassedTasks)
	assert.Equal(t, 1, summary.FailedTasks)
	assert.InDelta(t, 0.667, summary.PassRate, 0.01)
	assert.Equal(t, 180, summary.TotalTokens)
}

func TestBenchmarkSystem_Initialize(t *testing.T) {
	config := DefaultBenchmarkSystemConfig()
	system := NewBenchmarkSystem(config, nil)

	provider := NewProviderAdapterForBenchmark(nil, "test", "model", nil)
	err := system.Initialize(provider)
	require.NoError(t, err)

	assert.NotNil(t, system.GetRunner())
}

func TestBenchmarkSystem_GenerateLeaderboard(t *testing.T) {
	config := DefaultBenchmarkSystemConfig()
	system := NewBenchmarkSystem(config, nil)

	provider := &ProviderAdapterForBenchmark{
		providerName: "mock",
		modelName:    "mock-model",
	}
	require.NoError(t, system.Initialize(provider))

	runner := system.GetRunner().(*StandardBenchmarkRunner)

	for i := 0; i < 3; i++ {
		run := &BenchmarkRun{
			Name:          "Run",
			BenchmarkType: BenchmarkTypeMMLU,
			ProviderName:  "provider-" + string(rune('A'+i)),
			Config:        &BenchmarkConfig{MaxTasks: 1, Timeout: 30 * time.Second},
		}
		runner.CreateRun(context.Background(), run)
		runner.StartRun(context.Background(), run.ID)
	}

	time.Sleep(2 * time.Second)

	leaderboard, err := system.GenerateLeaderboard(context.Background(), BenchmarkTypeMMLU)
	require.NoError(t, err)
	assert.NotNil(t, leaderboard)
	assert.Equal(t, BenchmarkTypeMMLU, leaderboard.BenchmarkType)
}

func TestDebateAdapterForBenchmark_EvaluateResponse(t *testing.T) {
	mockService := &MockDebateServiceForBenchmark{}
	adapter := NewDebateAdapterForBenchmark(mockService, nil)

	task := &BenchmarkTask{
		Name:        "Test Task",
		Description: "Test description",
		Expected:    "Expected output",
	}

	score, passed, err := adapter.EvaluateResponse(context.Background(), task, "Test response")
	require.NoError(t, err)
	assert.Greater(t, score, 0.0)
	assert.True(t, passed)
}

// MockDebateServiceForBenchmark implements DebateServiceForBenchmark
type MockDebateServiceForBenchmark struct{}

func (m *MockDebateServiceForBenchmark) RunDebate(ctx context.Context, topic string) (*DebateResultForBenchmark, error) {
	return &DebateResultForBenchmark{
		ID:         "debate-123",
		Consensus:  `{"score": 0.85, "passed": true, "reasoning": "Good response"}`,
		Confidence: 0.85,
	}, nil
}

func TestVerifierAdapterForBenchmark_SelectBestProvider(t *testing.T) {
	mockService := &MockVerifierServiceForBenchmark{
		scores: map[string]float64{
			"claude":   8.5,
			"deepseek": 7.8,
			"gemini":   8.2,
		},
	}

	adapter := NewVerifierAdapterForBenchmark(mockService, nil)

	best, score := adapter.SelectBestProvider()
	assert.Equal(t, "claude", best)
	assert.Equal(t, 8.5, score)
}

// MockVerifierServiceForBenchmark implements VerifierServiceForBenchmark
type MockVerifierServiceForBenchmark struct {
	scores map[string]float64
}

func (m *MockVerifierServiceForBenchmark) GetProviderScore(name string) float64 {
	return m.scores[name]
}

func (m *MockVerifierServiceForBenchmark) IsProviderHealthy(name string) bool {
	return true
}

func (m *MockVerifierServiceForBenchmark) GetTopProviders(count int) []string {
	providers := make([]string, 0)
	for p := range m.scores {
		providers = append(providers, p)
	}
	return providers
}

func TestDefaultBenchmarkConfig(t *testing.T) {
	config := DefaultBenchmarkConfig()

	assert.Greater(t, config.Timeout, time.Duration(0))
	assert.Greater(t, config.Concurrency, 0)
	assert.True(t, config.SaveResponses)
}

func TestBenchmarkTask_Types(t *testing.T) {
	types := []BenchmarkType{
		BenchmarkTypeSWEBench,
		BenchmarkTypeHumanEval,
		BenchmarkTypeMBPP,
		BenchmarkTypeLMSYS,
		BenchmarkTypeHellaSwag,
		BenchmarkTypeMMLU,
		BenchmarkTypeGSM8K,
		BenchmarkTypeMATH,
		BenchmarkTypeCustom,
	}

	for _, bt := range types {
		task := &BenchmarkTask{
			ID:   "test",
			Type: bt,
		}
		assert.Equal(t, bt, task.Type)
	}
}
