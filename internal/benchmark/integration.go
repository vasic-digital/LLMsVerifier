package benchmark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// BenchmarkSystemConfig holds system configuration
type BenchmarkSystemConfig struct {
	DefaultConcurrency int           `json:"default_concurrency"`
	DefaultTimeout     time.Duration `json:"default_timeout"`
	EnableDebateEval   bool          `json:"enable_debate_eval"`
	LeaderboardSize    int           `json:"leaderboard_size"`
}

// DefaultBenchmarkSystemConfig returns default config
func DefaultBenchmarkSystemConfig() *BenchmarkSystemConfig {
	return &BenchmarkSystemConfig{
		DefaultConcurrency: 4,
		DefaultTimeout:     5 * time.Minute,
		EnableDebateEval:   true,
		LeaderboardSize:    20,
	}
}

// BenchmarkSystem orchestrates benchmarking
type BenchmarkSystem struct {
	runner BenchmarkRunner
	config *BenchmarkSystemConfig
	logger *log.Logger
	mu     sync.RWMutex
}

// NewBenchmarkSystem creates a new benchmark system
func NewBenchmarkSystem(config *BenchmarkSystemConfig, logger *log.Logger) *BenchmarkSystem {
	if config == nil {
		config = DefaultBenchmarkSystemConfig()
	}
	if logger == nil {
		logger = log.Default()
	}
	return &BenchmarkSystem{
		config: config,
		logger: logger,
	}
}

// Initialize initializes the system with a provider
func (s *BenchmarkSystem) Initialize(provider LLMProvider) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runner = NewStandardBenchmarkRunner(provider, s.logger)
	s.logger.Print(tr("benchmark.log.system_initialized"))
	return nil
}

// GetRunner returns the benchmark runner
func (s *BenchmarkSystem) GetRunner() BenchmarkRunner {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runner
}

// GenerateLeaderboard generates a leaderboard for a benchmark type
func (s *BenchmarkSystem) GenerateLeaderboard(ctx context.Context, benchmarkType BenchmarkType) (*Leaderboard, error) {
	s.mu.RLock()
	runner := s.runner
	s.mu.RUnlock()

	if runner == nil {
		return nil, fmt.Errorf("%s", tr("benchmark.err.system_not_initialized"))
	}

	runs, err := runner.ListRuns(ctx, &RunFilter{
		BenchmarkType: benchmarkType,
		Status:        BenchmarkStatusCompleted,
	})
	if err != nil {
		return nil, err
	}

	// Sort by score
	sort.Slice(runs, func(i, j int) bool {
		if runs[i].Summary == nil || runs[j].Summary == nil {
			return false
		}
		return runs[i].Summary.PassRate > runs[j].Summary.PassRate
	})

	leaderboard := &Leaderboard{
		BenchmarkType: benchmarkType,
		Entries:       make([]*LeaderboardEntry, 0),
		UpdatedAt:     time.Now(),
	}

	for i, run := range runs {
		if i >= s.config.LeaderboardSize {
			break
		}
		if run.Summary == nil {
			continue
		}
		entry := &LeaderboardEntry{
			Rank:         i + 1,
			ProviderName: run.ProviderName,
			ModelName:    run.ModelName,
			Score:        run.Summary.AverageScore,
			PassRate:     run.Summary.PassRate,
			RunID:        run.ID,
		}
		leaderboard.Entries = append(leaderboard.Entries, entry)
	}

	return leaderboard, nil
}

// DebateAdapterForBenchmark adapts debate service for benchmarks
type DebateAdapterForBenchmark struct {
	service DebateServiceForBenchmark
	logger  *log.Logger
}

// DebateServiceForBenchmark interface for debate service
type DebateServiceForBenchmark interface {
	RunDebate(ctx context.Context, topic string) (*DebateResultForBenchmark, error)
}

// DebateResultForBenchmark represents debate result
type DebateResultForBenchmark struct {
	ID         string  `json:"id"`
	Consensus  string  `json:"consensus"`
	Confidence float64 `json:"confidence"`
}

// NewDebateAdapterForBenchmark creates a new adapter
func NewDebateAdapterForBenchmark(service DebateServiceForBenchmark, logger *log.Logger) *DebateAdapterForBenchmark {
	if logger == nil {
		logger = log.Default()
	}
	return &DebateAdapterForBenchmark{
		service: service,
		logger:  logger,
	}
}

// EvaluateResponse evaluates a response using debate
func (a *DebateAdapterForBenchmark) EvaluateResponse(ctx context.Context, task *BenchmarkTask, response string) (float64, bool, error) {
	topic := fmt.Sprintf(`Evaluate this response for task "%s":
Task: %s
Expected: %s
Response: %s

Is the response correct? Score 0-1.`, task.Name, task.Description, task.Expected, response)

	result, err := a.service.RunDebate(ctx, topic)
	if err != nil {
		return 0, false, err
	}

	var consensus struct {
		Score  float64 `json:"score"`
		Passed bool    `json:"passed"`
	}
	json.Unmarshal([]byte(result.Consensus), &consensus)

	if consensus.Score == 0 {
		consensus.Score = result.Confidence
	}
	consensus.Passed = consensus.Score >= 0.5

	return consensus.Score, consensus.Passed, nil
}

// VerifierAdapterForBenchmark adapts verifier for benchmarks
type VerifierAdapterForBenchmark struct {
	service VerifierServiceForBenchmark
	logger  *log.Logger
}

// VerifierServiceForBenchmark interface for verifier
type VerifierServiceForBenchmark interface {
	GetProviderScore(name string) float64
	IsProviderHealthy(name string) bool
	GetTopProviders(count int) []string
}

// NewVerifierAdapterForBenchmark creates a new adapter
func NewVerifierAdapterForBenchmark(service VerifierServiceForBenchmark, logger *log.Logger) *VerifierAdapterForBenchmark {
	if logger == nil {
		logger = log.Default()
	}
	return &VerifierAdapterForBenchmark{
		service: service,
		logger:  logger,
	}
}

// SelectBestProvider selects the best provider for benchmarking
func (a *VerifierAdapterForBenchmark) SelectBestProvider() (string, float64) {
	providers := a.service.GetTopProviders(10)
	if len(providers) == 0 {
		return "", 0
	}

	best := providers[0]
	bestScore := a.service.GetProviderScore(best)

	for _, p := range providers[1:] {
		score := a.service.GetProviderScore(p)
		if score > bestScore && a.service.IsProviderHealthy(p) {
			best = p
			bestScore = score
		}
	}

	return best, bestScore
}

// ErrProviderAdapterNotWired fires from ProviderAdapterForBenchmark.Complete
// when the adapter holds no real underlying LLMProvider. Previously the
// Complete method fabricated a hardcoded ("Response", 50, nil) result
// regardless of input — a §11.4 / CONST-035 PASS-bluff (BLUFF-001 pattern)
// in production code: the adapter claimed "calls the underlying provider"
// but dispatched nothing. The sentinel makes the absence of a wired provider
// an honest, surfaced failure instead of a silent fabricated completion.
var ErrProviderAdapterNotWired = errors.New(
	"llmsverifier benchmark: ProviderAdapterForBenchmark has no underlying LLMProvider wired — pass a non-nil value implementing benchmark.LLMProvider (e.g. *HTTPBenchmarkProvider) to NewProviderAdapterForBenchmark; the previous nil-provider branch fabricated a hardcoded (\"Response\", 50) completion regardless of input — §11.4 / CONST-035 PASS-bluff removed",
)

// ProviderAdapterForBenchmark adapts a real LLMProvider for use as the
// benchmark runner's provider. The underlying provider field MUST hold a
// value implementing the benchmark.LLMProvider contract (Complete + GetName);
// Complete dispatches directly to it. Construct the underlying provider with
// NewHTTPBenchmarkProvider (OpenAI-compatible HTTP dispatch) or any other
// concrete LLMProvider implementation. NO completion value is ever
// fabricated — every response, token count, and error returned by Complete
// originates from the real underlying provider.
type ProviderAdapterForBenchmark struct {
	provider     LLMProvider
	providerName string
	modelName    string
	logger       *log.Logger
}

// NewProviderAdapterForBenchmark creates a new adapter wrapping a real
// LLMProvider. The provider argument is accepted as interface{} for caller
// convenience but MUST satisfy the benchmark.LLMProvider interface for
// Complete to dispatch; a nil provider (or a non-LLMProvider value) leaves
// the adapter un-wired and Complete returns ErrProviderAdapterNotWired
// rather than fabricating a result.
func NewProviderAdapterForBenchmark(provider interface{}, name, model string, logger *log.Logger) *ProviderAdapterForBenchmark {
	if logger == nil {
		logger = log.Default()
	}
	a := &ProviderAdapterForBenchmark{
		providerName: name,
		modelName:    model,
		logger:       logger,
	}
	if lp, ok := provider.(LLMProvider); ok && lp != nil {
		a.provider = lp
	}
	return a
}

// Complete dispatches the prompt to the real underlying LLMProvider and
// returns its genuine response text, token count, and error verbatim. When
// no underlying provider is wired, it returns ErrProviderAdapterNotWired —
// it NEVER fabricates a completion.
func (a *ProviderAdapterForBenchmark) Complete(ctx context.Context, prompt, systemPrompt string) (string, int, error) {
	if a.provider == nil {
		return "", 0, fmt.Errorf("%w (provider=%q model=%q)", ErrProviderAdapterNotWired, a.providerName, a.modelName)
	}
	return a.provider.Complete(ctx, prompt, systemPrompt)
}

// GetName returns provider name
func (a *ProviderAdapterForBenchmark) GetName() string {
	return a.providerName
}
