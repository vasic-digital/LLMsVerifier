package benchmark

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrBenchmarkProviderNotConfigured is returned by executeTask when the runner
// was constructed without an LLMProvider (or had it nil-set). Previously the
// nil-provider branch fabricated Passed=true / Score=0.8 / Latency=100ms /
// TokensUsed=50 for every task regardless of input — a §11.4 PASS-bluff at the
// benchmark-runner default layer. The sentinel makes the absence of wiring an
// honest, surfaced failure instead of a silent green PASS.
var ErrBenchmarkProviderNotConfigured = fmt.Errorf("llmsverifier benchmark: provider has not been wired into the runner — call NewBenchmarkRunner with a non-nil provider or use SetProvider before invoking ExecuteRun (the previous nil-provider branch fabricated Passed=true/Score=0.8/Latency=100ms/TokensUsed=50 regardless of input; §11.4 PASS-bluff removed)")

// StandardBenchmarkRunner implements BenchmarkRunner
type StandardBenchmarkRunner struct {
	mu             sync.RWMutex
	provider       LLMProvider
	debateEval     DebateEvaluator
	benchmarks     map[string]*Benchmark
	tasks          map[string][]*BenchmarkTask
	runs           map[string]*BenchmarkRun
	runCancels     map[string]context.CancelFunc
	logger         *log.Logger
}

// NewStandardBenchmarkRunner creates a new benchmark runner
func NewStandardBenchmarkRunner(provider LLMProvider, logger *log.Logger) *StandardBenchmarkRunner {
	if logger == nil {
		logger = log.Default()
	}
	r := &StandardBenchmarkRunner{
		provider:   provider,
		benchmarks: make(map[string]*Benchmark),
		tasks:      make(map[string][]*BenchmarkTask),
		runs:       make(map[string]*BenchmarkRun),
		runCancels: make(map[string]context.CancelFunc),
		logger:     logger,
	}
	r.initBuiltInBenchmarks()
	return r
}

// SetDebateEvaluator sets the debate evaluator
func (r *StandardBenchmarkRunner) SetDebateEvaluator(eval DebateEvaluator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.debateEval = eval
}

// SetProvider injects an LLMProvider after construction. Required when the
// runner was created with provider=nil (e.g. for fast unit-test bootstrap of
// the benchmark catalogue). Without a provider, executeTask returns
// ErrBenchmarkProviderNotConfigured per the §11.4 anti-bluff posture.
func (r *StandardBenchmarkRunner) SetProvider(provider LLMProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.provider = provider
}

// initBuiltInBenchmarks initializes built-in benchmarks
func (r *StandardBenchmarkRunner) initBuiltInBenchmarks() {
	// SWE-Bench
	r.addBenchmark(&Benchmark{
		ID: "swe-bench-lite", Type: BenchmarkTypeSWEBench,
		Name: "SWE-Bench Lite", Version: "1.0",
	}, r.createSWEBenchTasks())

	// HumanEval
	r.addBenchmark(&Benchmark{
		ID: "human-eval", Type: BenchmarkTypeHumanEval,
		Name: "HumanEval", Version: "1.0",
	}, r.createHumanEvalTasks())

	// MMLU
	r.addBenchmark(&Benchmark{
		ID: "mmlu", Type: BenchmarkTypeMMLU,
		Name: "MMLU", Version: "1.0",
	}, r.createMMLUTasks())

	// GSM8K
	r.addBenchmark(&Benchmark{
		ID: "gsm8k", Type: BenchmarkTypeGSM8K,
		Name: "GSM8K", Version: "1.0",
	}, r.createGSM8KTasks())
}

func (r *StandardBenchmarkRunner) addBenchmark(b *Benchmark, tasks []*BenchmarkTask) {
	b.TaskCount = len(tasks)
	r.benchmarks[b.ID] = b
	r.tasks[b.ID] = tasks
}

// AddBenchmark adds a custom benchmark
func (r *StandardBenchmarkRunner) AddBenchmark(b *Benchmark, tasks []*BenchmarkTask) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addBenchmark(b, tasks)
}

func (r *StandardBenchmarkRunner) createSWEBenchTasks() []*BenchmarkTask {
	return []*BenchmarkTask{
		{ID: "swe-1", Type: BenchmarkTypeSWEBench, Name: "Fix NPE in User Service",
			Prompt: "Fix the null pointer exception in the UserService.getUser() method",
			Difficulty: DifficultyEasy, Tags: []string{"bug-fix", "java"}},
		{ID: "swe-2", Type: BenchmarkTypeSWEBench, Name: "Implement Caching",
			Prompt: "Add Redis caching to the product catalog service",
			Difficulty: DifficultyMedium, Tags: []string{"feature", "caching"}},
		{ID: "swe-3", Type: BenchmarkTypeSWEBench, Name: "Optimize Query",
			Prompt: "Optimize the slow SQL query in the reports module",
			Difficulty: DifficultyHard, Tags: []string{"optimization", "sql"}},
	}
}

func (r *StandardBenchmarkRunner) createHumanEvalTasks() []*BenchmarkTask {
	return []*BenchmarkTask{
		{ID: "he-1", Type: BenchmarkTypeHumanEval, Name: "has_close_elements",
			Prompt: `def has_close_elements(numbers: List[float], threshold: float) -> bool:
    """ Check if in given list of numbers, are any two numbers closer to each other than
    given threshold. """`,
			Expected: "True for [1.0, 2.0, 3.0] with threshold 1.5",
			Difficulty: DifficultyEasy, Tags: []string{"code"}},
		{ID: "he-2", Type: BenchmarkTypeHumanEval, Name: "separate_paren_groups",
			Prompt: `def separate_paren_groups(paren_string: str) -> List[str]:
    """ Separate balanced parentheses groups. """`,
			Expected: "['(()())', '((()))', '()()']",
			Difficulty: DifficultyMedium, Tags: []string{"code"}},
	}
}

func (r *StandardBenchmarkRunner) createMMLUTasks() []*BenchmarkTask {
	return []*BenchmarkTask{
		{ID: "mmlu-1", Type: BenchmarkTypeMMLU, Name: "Binary Search Complexity",
			Prompt: "What is the time complexity of binary search? A) O(n) B) O(log n) C) O(n^2) D) O(1)",
			Expected: "B", Difficulty: DifficultyEasy, Tags: []string{"cs", "algorithms"}},
		{ID: "mmlu-2", Type: BenchmarkTypeMMLU, Name: "Physics: Newton's Law",
			Prompt: "Which law states F=ma? A) First B) Second C) Third D) None",
			Expected: "B", Difficulty: DifficultyEasy, Tags: []string{"physics"}},
		{ID: "mmlu-3", Type: BenchmarkTypeMMLU, Name: "World History",
			Prompt: "In which year did World War II end? A) 1943 B) 1944 C) 1945 D) 1946",
			Expected: "C", Difficulty: DifficultyMedium, Tags: []string{"history"}},
	}
}

func (r *StandardBenchmarkRunner) createGSM8KTasks() []*BenchmarkTask {
	return []*BenchmarkTask{
		{ID: "gsm-1", Type: BenchmarkTypeGSM8K, Name: "Eggs Problem",
			Prompt: "Janet sells duck eggs. She sells 16 eggs per day. How many eggs per week?",
			Expected: "112", Difficulty: DifficultyEasy, Tags: []string{"math"}},
		{ID: "gsm-2", Type: BenchmarkTypeGSM8K, Name: "Shopping Problem",
			Prompt: "Tom bought 3 apples at $2 each and 4 oranges at $1.5 each. What is the total?",
			Expected: "12", Difficulty: DifficultyEasy, Tags: []string{"math"}},
		{ID: "gsm-3", Type: BenchmarkTypeGSM8K, Name: "Workers Problem",
			Prompt: "If 5 workers can build a wall in 10 days, how many days for 10 workers?",
			Expected: "5", Difficulty: DifficultyMedium, Tags: []string{"math"}},
	}
}

// ListBenchmarks lists available benchmarks
func (r *StandardBenchmarkRunner) ListBenchmarks(ctx context.Context) ([]*Benchmark, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Benchmark, 0, len(r.benchmarks))
	for _, b := range r.benchmarks {
		result = append(result, b)
	}
	return result, nil
}

// GetTasks gets tasks for a benchmark
func (r *StandardBenchmarkRunner) GetTasks(ctx context.Context, benchmarkID string, config *BenchmarkConfig) ([]*BenchmarkTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks, ok := r.tasks[benchmarkID]
	if !ok {
		return nil, fmt.Errorf("%s", trData("benchmark.err.benchmark_not_found", map[string]any{"id": benchmarkID}))
	}

	if config == nil {
		return tasks, nil
	}

	// Filter tasks
	var filtered []*BenchmarkTask
	for _, t := range tasks {
		if len(config.Difficulties) > 0 && !containsDifficulty(config.Difficulties, t.Difficulty) {
			continue
		}
		filtered = append(filtered, t)
		if config.MaxTasks > 0 && len(filtered) >= config.MaxTasks {
			break
		}
	}
	return filtered, nil
}

func containsDifficulty(list []DifficultyLevel, d DifficultyLevel) bool {
	for _, l := range list {
		if l == d {
			return true
		}
	}
	return false
}

// CreateRun creates a new benchmark run
func (r *StandardBenchmarkRunner) CreateRun(ctx context.Context, run *BenchmarkRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if run.ID == "" {
		run.ID = uuid.New().String()
	}
	run.Status = BenchmarkStatusPending
	run.CreatedAt = time.Now()

	if run.Config == nil {
		run.Config = DefaultBenchmarkConfig()
	}

	r.runs[run.ID] = run
	r.logger.Print(trData("benchmark.log.run_created", map[string]any{"id": run.ID}))
	return nil
}

// StartRun starts a benchmark run
func (r *StandardBenchmarkRunner) StartRun(ctx context.Context, runID string) error {
	r.mu.Lock()
	run, ok := r.runs[runID]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("%s", trData("benchmark.err.run_not_found", map[string]any{"id": runID}))
	}

	benchmarkID := string(run.BenchmarkType)
	if run.BenchmarkType == BenchmarkTypeMMLU {
		benchmarkID = "mmlu"
	} else if run.BenchmarkType == BenchmarkTypeSWEBench {
		benchmarkID = "swe-bench-lite"
	} else if run.BenchmarkType == BenchmarkTypeHumanEval {
		benchmarkID = "human-eval"
	} else if run.BenchmarkType == BenchmarkTypeGSM8K {
		benchmarkID = "gsm8k"
	}

	tasks := r.tasks[benchmarkID]
	run.Status = BenchmarkStatusRunning
	now := time.Now()
	run.StartedAt = &now

	ctx, cancel := context.WithCancel(ctx)
	r.runCancels[runID] = cancel
	r.mu.Unlock()

	go r.executeRun(ctx, run, tasks)
	return nil
}

// executeRun executes the benchmark run
func (r *StandardBenchmarkRunner) executeRun(ctx context.Context, run *BenchmarkRun, tasks []*BenchmarkTask) {
	if run.Config != nil && run.Config.MaxTasks > 0 && len(tasks) > run.Config.MaxTasks {
		tasks = tasks[:run.Config.MaxTasks]
	}

	results := make([]*BenchmarkResult, 0, len(tasks))
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			r.mu.Lock()
			run.Status = BenchmarkStatusCancelled
			r.mu.Unlock()
			return
		default:
		}

		result := r.executeTask(ctx, run, task)
		results = append(results, result)
	}

	r.mu.Lock()
	run.Results = results
	run.Summary = r.calculateSummary(results, tasks)
	run.Status = BenchmarkStatusCompleted
	now := time.Now()
	run.EndedAt = &now
	r.mu.Unlock()

	r.logger.Print(trData("benchmark.log.run_completed", map[string]any{
		"id":        run.ID,
		"pass_rate": fmt.Sprintf("%.2f", run.Summary.PassRate),
	}))
}

// executeTask executes a single task
func (r *StandardBenchmarkRunner) executeTask(ctx context.Context, run *BenchmarkRun, task *BenchmarkTask) *BenchmarkResult {
	start := time.Now()

	result := &BenchmarkResult{TaskID: task.ID}

	if r.provider != nil {
		response, tokens, err := r.provider.Complete(ctx, task.Prompt, "")
		result.Latency = time.Since(start)
		result.TokensUsed = tokens

		if err != nil {
			result.Error = err.Error()
			result.Passed = false
			return result
		}

		result.Response = response
		result.Passed, result.Score = r.evaluateResponse(ctx, run, task, response)
	} else {
		// §11.4 anti-bluff: the previous else-branch hardcoded
		// Passed=true / Score=0.8 / Latency=100ms / TokensUsed=50, so every
		// task PASSed regardless of input whenever the runner had no LLM
		// provider wired in. That is a fabricated success — exactly the
		// failure mode Article XI §11.9 (forensic anchor) and CONST-035
		// (zero-bluff mandate) forbid. Surface the missing-wiring as an
		// honest failure instead. Do NOT populate fake Score/Latency/Tokens.
		result.Latency = time.Since(start)
		result.Passed = false
		result.Error = ErrBenchmarkProviderNotConfigured.Error()
	}

	return result
}

// evaluateResponse evaluates a response
func (r *StandardBenchmarkRunner) evaluateResponse(ctx context.Context, run *BenchmarkRun, task *BenchmarkTask, response string) (bool, float64) {
	// Use debate evaluation if configured
	if run.Config != nil && run.Config.UseDebateForEval && r.debateEval != nil {
		score, passed, err := r.debateEval.EvaluateResponse(ctx, task, response)
		if err == nil {
			return passed, score
		}
	}

	// Simple evaluation - check if expected is in response
	if task.Expected != "" && containsStr(response, task.Expected) {
		return true, 1.0
	}

	return len(response) > 0, 0.5
}

func containsStr(s, substr string) bool {
	if len(substr) == 0 || len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// calculateSummary calculates run summary
func (r *StandardBenchmarkRunner) calculateSummary(results []*BenchmarkResult, tasks []*BenchmarkTask) *BenchmarkSummary {
	summary := &BenchmarkSummary{
		TotalTasks:   len(results),
		ByDifficulty: make(map[DifficultyLevel]*DifficultyStats),
		ByTag:        make(map[string]*TagStats),
	}

	taskMap := make(map[string]*BenchmarkTask)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	var totalLatency time.Duration
	var totalScore float64

	for _, r := range results {
		if r.Passed {
			summary.PassedTasks++
		} else {
			summary.FailedTasks++
		}
		totalLatency += r.Latency
		totalScore += r.Score
		summary.TotalTokens += r.TokensUsed

		if task, ok := taskMap[r.TaskID]; ok {
			// By difficulty
			if _, ok := summary.ByDifficulty[task.Difficulty]; !ok {
				summary.ByDifficulty[task.Difficulty] = &DifficultyStats{}
			}
			summary.ByDifficulty[task.Difficulty].Total++
			if r.Passed {
				summary.ByDifficulty[task.Difficulty].Passed++
			}
		}
	}

	if summary.TotalTasks > 0 {
		summary.PassRate = float64(summary.PassedTasks) / float64(summary.TotalTasks)
		summary.AverageScore = totalScore / float64(summary.TotalTasks)
		summary.AverageLatency = totalLatency / time.Duration(summary.TotalTasks)
	}

	// Calculate pass rates per difficulty
	for _, ds := range summary.ByDifficulty {
		if ds.Total > 0 {
			ds.PassRate = float64(ds.Passed) / float64(ds.Total)
		}
	}

	return summary
}

// GetRun retrieves a run. Returns a shallow copy so callers can read its
// scalar fields (Status, EndedAt, Summary) without racing against the
// runner's executeRun goroutine which mutates the live struct under the
// mutex. Callers that need a pointer into the live instance must reach
// for an internal helper that holds the lock for the duration of their
// read; external test code goes through this safe path.
func (r *StandardBenchmarkRunner) GetRun(ctx context.Context, runID string) (*BenchmarkRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	run, ok := r.runs[runID]
	if !ok {
		return nil, fmt.Errorf("%s", trData("benchmark.err.run_not_found", map[string]any{"id": runID}))
	}
	clone := *run
	return &clone, nil
}

// ListRuns lists runs
func (r *StandardBenchmarkRunner) ListRuns(ctx context.Context, filter *RunFilter) ([]*BenchmarkRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*BenchmarkRun
	for _, run := range r.runs {
		if filter != nil && filter.Status != "" && run.Status != filter.Status {
			continue
		}
		result = append(result, run)
	}
	return result, nil
}

// CancelRun cancels a run
func (r *StandardBenchmarkRunner) CancelRun(ctx context.Context, runID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	run, ok := r.runs[runID]
	if !ok {
		return fmt.Errorf("%s", trData("benchmark.err.run_not_found", map[string]any{"id": runID}))
	}

	if cancel, ok := r.runCancels[runID]; ok {
		cancel()
	}

	run.Status = BenchmarkStatusCancelled
	return nil
}

// CompareRuns compares two runs
func (r *StandardBenchmarkRunner) CompareRuns(ctx context.Context, run1ID, run2ID string) (*RunComparison, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	run1, ok := r.runs[run1ID]
	if !ok {
		return nil, fmt.Errorf("%s", trData("benchmark.err.run_not_found", map[string]any{"id": run1ID}))
	}
	run2, ok := r.runs[run2ID]
	if !ok {
		return nil, fmt.Errorf("%s", trData("benchmark.err.run_not_found", map[string]any{"id": run2ID}))
	}

	comparison := &RunComparison{
		Run1ID:  run1ID,
		Run2ID:  run2ID,
		Details: make(map[string]interface{}),
	}

	if run1.Summary != nil && run2.Summary != nil {
		diff := run2.Summary.PassRate - run1.Summary.PassRate
		comparison.Improvement = diff
		comparison.Summary = trData("benchmark.summary.run_comparison", map[string]any{
			"improvement_pct": fmt.Sprintf("%.2f", diff*100),
		})
	}

	return comparison, nil
}
