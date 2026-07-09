package benchmark

import (
	"context"
	"time"
)

// BenchmarkType represents the type of benchmark
type BenchmarkType string

const (
	BenchmarkTypeSWEBench  BenchmarkType = "swe-bench"
	BenchmarkTypeHumanEval BenchmarkType = "human-eval"
	BenchmarkTypeMBPP      BenchmarkType = "mbpp"
	BenchmarkTypeLMSYS     BenchmarkType = "lmsys"
	BenchmarkTypeHellaSwag BenchmarkType = "hellaswag"
	BenchmarkTypeMMLU      BenchmarkType = "mmlu"
	BenchmarkTypeGSM8K     BenchmarkType = "gsm8k"
	BenchmarkTypeMATH      BenchmarkType = "math"
	BenchmarkTypeCustom    BenchmarkType = "custom"
)

// DifficultyLevel represents task difficulty
type DifficultyLevel string

const (
	DifficultyEasy   DifficultyLevel = "easy"
	DifficultyMedium DifficultyLevel = "medium"
	DifficultyHard   DifficultyLevel = "hard"
)

// BenchmarkStatus represents the status of a benchmark run
type BenchmarkStatus string

const (
	BenchmarkStatusPending   BenchmarkStatus = "pending"
	BenchmarkStatusRunning   BenchmarkStatus = "running"
	BenchmarkStatusCompleted BenchmarkStatus = "completed"
	BenchmarkStatusFailed    BenchmarkStatus = "failed"
	BenchmarkStatusCancelled BenchmarkStatus = "cancelled"
)

// Benchmark represents a benchmark definition
type Benchmark struct {
	ID          string        `json:"id"`
	Type        BenchmarkType `json:"type"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Version     string        `json:"version"`
	TaskCount   int           `json:"task_count"`
}

// BenchmarkTask represents a single benchmark task
type BenchmarkTask struct {
	ID          string                 `json:"id"`
	BenchmarkID string                 `json:"benchmark_id"`
	Type        BenchmarkType          `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Prompt      string                 `json:"prompt"`
	Expected    string                 `json:"expected,omitempty"`
	TestCases   []*TestCase            `json:"test_cases,omitempty"`
	Difficulty  DifficultyLevel        `json:"difficulty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TestCase represents a test case for code benchmarks
type TestCase struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

// BenchmarkResult represents the result of a task
type BenchmarkResult struct {
	TaskID      string                 `json:"task_id"`
	Response    string                 `json:"response"`
	Passed      bool                   `json:"passed"`
	Score       float64                `json:"score"`
	Latency     time.Duration          `json:"latency"`
	TokensUsed  int                    `json:"tokens_used"`
	Error       string                 `json:"error,omitempty"`
	DebateScore float64                `json:"debate_score,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BenchmarkRun represents a benchmark run
type BenchmarkRun struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	BenchmarkType BenchmarkType      `json:"benchmark_type"`
	ProviderName  string             `json:"provider_name"`
	ModelName     string             `json:"model_name,omitempty"`
	Config        *BenchmarkConfig   `json:"config"`
	Status        BenchmarkStatus    `json:"status"`
	Results       []*BenchmarkResult `json:"results,omitempty"`
	Summary       *BenchmarkSummary  `json:"summary,omitempty"`
	StartedAt     *time.Time         `json:"started_at,omitempty"`
	EndedAt       *time.Time         `json:"ended_at,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
}

// BenchmarkConfig holds configuration for a benchmark run
type BenchmarkConfig struct {
	MaxTasks         int               `json:"max_tasks,omitempty"`
	Difficulties     []DifficultyLevel `json:"difficulties,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	Timeout          time.Duration     `json:"timeout"`
	Concurrency      int               `json:"concurrency"`
	SaveResponses    bool              `json:"save_responses"`
	UseDebateForEval bool              `json:"use_debate_for_eval"`
}

// DefaultBenchmarkConfig returns default configuration
func DefaultBenchmarkConfig() *BenchmarkConfig {
	return &BenchmarkConfig{
		MaxTasks:         100,
		Timeout:          5 * time.Minute,
		Concurrency:      4,
		SaveResponses:    true,
		UseDebateForEval: false,
	}
}

// BenchmarkSummary represents summary statistics
type BenchmarkSummary struct {
	TotalTasks     int                                  `json:"total_tasks"`
	PassedTasks    int                                  `json:"passed_tasks"`
	FailedTasks    int                                  `json:"failed_tasks"`
	PassRate       float64                              `json:"pass_rate"`
	AverageScore   float64                              `json:"average_score"`
	AverageLatency time.Duration                        `json:"average_latency"`
	TotalTokens    int                                  `json:"total_tokens"`
	ByDifficulty   map[DifficultyLevel]*DifficultyStats `json:"by_difficulty,omitempty"`
	ByTag          map[string]*TagStats                 `json:"by_tag,omitempty"`
}

// DifficultyStats represents stats per difficulty
type DifficultyStats struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
}

// TagStats represents stats per tag
type TagStats struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
}

// RunComparison represents comparison between runs
type RunComparison struct {
	Run1ID      string                 `json:"run1_id"`
	Run2ID      string                 `json:"run2_id"`
	Summary     string                 `json:"summary"`
	Improvement float64                `json:"improvement"`
	Details     map[string]interface{} `json:"details"`
}

// RunFilter for filtering runs
type RunFilter struct {
	BenchmarkType BenchmarkType   `json:"benchmark_type,omitempty"`
	Status        BenchmarkStatus `json:"status,omitempty"`
	ProviderName  string          `json:"provider_name,omitempty"`
	Limit         int             `json:"limit,omitempty"`
}

// Leaderboard represents benchmark leaderboard
type Leaderboard struct {
	BenchmarkType BenchmarkType       `json:"benchmark_type"`
	Entries       []*LeaderboardEntry `json:"entries"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

// LeaderboardEntry represents a leaderboard entry
type LeaderboardEntry struct {
	Rank         int     `json:"rank"`
	ProviderName string  `json:"provider_name"`
	ModelName    string  `json:"model_name"`
	Score        float64 `json:"score"`
	PassRate     float64 `json:"pass_rate"`
	RunID        string  `json:"run_id"`
}

// BenchmarkRunner runs benchmarks
type BenchmarkRunner interface {
	ListBenchmarks(ctx context.Context) ([]*Benchmark, error)
	GetTasks(ctx context.Context, benchmarkID string, config *BenchmarkConfig) ([]*BenchmarkTask, error)
	CreateRun(ctx context.Context, run *BenchmarkRun) error
	StartRun(ctx context.Context, runID string) error
	GetRun(ctx context.Context, runID string) (*BenchmarkRun, error)
	ListRuns(ctx context.Context, filter *RunFilter) ([]*BenchmarkRun, error)
	CancelRun(ctx context.Context, runID string) error
	CompareRuns(ctx context.Context, run1ID, run2ID string) (*RunComparison, error)
}

// LLMProvider interface for LLM providers
type LLMProvider interface {
	Complete(ctx context.Context, prompt, systemPrompt string) (string, int, error)
	GetName() string
}

// DebateEvaluator evaluates responses using debate
type DebateEvaluator interface {
	EvaluateResponse(ctx context.Context, task *BenchmarkTask, response string) (float64, bool, error)
}
