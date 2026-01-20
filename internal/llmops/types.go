package llmops

import (
	"context"
	"time"
)

// PromptVariable represents a variable in a prompt template
type PromptVariable struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// PromptVersion represents a versioned prompt template
type PromptVersion struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	Content     string           `json:"content"`
	Variables   []PromptVariable `json:"variables,omitempty"`
	SystemPrompt string          `json:"system_prompt,omitempty"`
	Model       string           `json:"model,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// ExperimentStatus represents the status of an experiment
type ExperimentStatus string

const (
	ExperimentStatusDraft     ExperimentStatus = "draft"
	ExperimentStatusRunning   ExperimentStatus = "running"
	ExperimentStatusPaused    ExperimentStatus = "paused"
	ExperimentStatusCompleted ExperimentStatus = "completed"
	ExperimentStatusCancelled ExperimentStatus = "cancelled"
)

// Variant represents an experiment variant
type Variant struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	PromptID    string                 `json:"prompt_id,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	IsControl   bool                   `json:"is_control"`
	Assignments int                    `json:"assignments"`
}

// Experiment represents an A/B testing experiment
type Experiment struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Variants     []*Variant         `json:"variants"`
	TrafficSplit map[string]float64 `json:"traffic_split"`
	Status       ExperimentStatus   `json:"status"`
	Metrics      []string           `json:"metrics,omitempty"`
	TargetMetric string             `json:"target_metric,omitempty"`
	StartedAt    *time.Time         `json:"started_at,omitempty"`
	EndedAt      *time.Time         `json:"ended_at,omitempty"`
	Winner       string             `json:"winner,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// ExperimentResults represents results of an experiment
type ExperimentResults struct {
	ExperimentID    string                  `json:"experiment_id"`
	TotalSamples    int                     `json:"total_samples"`
	VariantResults  map[string]*VariantResult `json:"variant_results"`
	WinningVariant  string                  `json:"winning_variant,omitempty"`
	Confidence      float64                 `json:"confidence"`
	IsSignificant   bool                    `json:"is_significant"`
}

// VariantResult represents results for a single variant
type VariantResult struct {
	VariantID     string             `json:"variant_id"`
	SampleCount   int                `json:"sample_count"`
	MetricValues  map[string]float64 `json:"metric_values"`
	AverageScore  float64            `json:"average_score"`
	Improvement   float64            `json:"improvement"` // vs control
}

// EvaluationStatus represents the status of an evaluation run
type EvaluationStatus string

const (
	EvaluationStatusPending   EvaluationStatus = "pending"
	EvaluationStatusRunning   EvaluationStatus = "running"
	EvaluationStatusCompleted EvaluationStatus = "completed"
	EvaluationStatusFailed    EvaluationStatus = "failed"
)

// DatasetType represents types of evaluation datasets
type DatasetType string

const (
	DatasetTypeGolden     DatasetType = "golden"
	DatasetTypeProduction DatasetType = "production"
	DatasetTypeSynthetic  DatasetType = "synthetic"
)

// Dataset represents an evaluation dataset
type Dataset struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Type        DatasetType `json:"type"`
	Samples     int         `json:"samples"`
	CreatedAt   time.Time   `json:"created_at"`
}

// DatasetSample represents a single sample in a dataset
type DatasetSample struct {
	ID             string                 `json:"id"`
	Input          string                 `json:"input"`
	ExpectedOutput string                 `json:"expected_output,omitempty"`
	Context        string                 `json:"context,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// EvaluationRun represents a continuous evaluation run
type EvaluationRun struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Dataset    string           `json:"dataset"`
	PromptName string           `json:"prompt_name"`
	Metrics    []string         `json:"metrics"`
	Status     EvaluationStatus `json:"status"`
	Results    *EvaluationResults `json:"results,omitempty"`
	StartedAt  *time.Time       `json:"started_at,omitempty"`
	EndedAt    *time.Time       `json:"ended_at,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}

// EvaluationResults represents results of an evaluation
type EvaluationResults struct {
	Metrics       map[string]float64 `json:"metrics"`
	PassRate      float64            `json:"pass_rate"`
	TotalSamples  int                `json:"total_samples"`
	PassedSamples int                `json:"passed_samples"`
}

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertType represents types of alerts
type AlertType string

const (
	AlertTypeRegression AlertType = "regression"
	AlertTypeThreshold  AlertType = "threshold"
	AlertTypeAnomaly    AlertType = "anomaly"
)

// Alert represents an LLMOps alert
type Alert struct {
	ID         string                 `json:"id"`
	Type       AlertType              `json:"type"`
	Severity   AlertSeverity          `json:"severity"`
	Message    string                 `json:"message"`
	Source     string                 `json:"source"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// AlertHandler handles alert notifications
type AlertHandler func(*Alert) error

// PromptRegistry manages prompt versions
type PromptRegistry interface {
	Create(ctx context.Context, prompt *PromptVersion) error
	Get(ctx context.Context, name, version string) (*PromptVersion, error)
	GetLatest(ctx context.Context, name string) (*PromptVersion, error)
	List(ctx context.Context, name string) ([]*PromptVersion, error)
	Activate(ctx context.Context, name, version string) error
	Delete(ctx context.Context, name, version string) error
	Render(ctx context.Context, name, version string, vars map[string]interface{}) (string, error)
}

// ExperimentManager manages A/B experiments
type ExperimentManager interface {
	Create(ctx context.Context, exp *Experiment) error
	Get(ctx context.Context, id string) (*Experiment, error)
	List(ctx context.Context) ([]*Experiment, error)
	Start(ctx context.Context, id string) error
	Pause(ctx context.Context, id string) error
	Complete(ctx context.Context, id, winner string) error
	AssignVariant(ctx context.Context, expID, userID string) (*Variant, error)
	RecordMetric(ctx context.Context, expID, variantID, metric string, value float64) error
	GetResults(ctx context.Context, expID string) (*ExperimentResults, error)
}

// ContinuousEvaluator runs continuous evaluations
type ContinuousEvaluator interface {
	CreateDataset(ctx context.Context, dataset *Dataset) error
	AddSamples(ctx context.Context, datasetID string, samples []*DatasetSample) error
	CreateRun(ctx context.Context, run *EvaluationRun) error
	StartRun(ctx context.Context, runID string) error
	GetRun(ctx context.Context, runID string) (*EvaluationRun, error)
	ListRuns(ctx context.Context) ([]*EvaluationRun, error)
}

// AlertManager manages alerts
type AlertManager interface {
	Create(ctx context.Context, alert *Alert) error
	Get(ctx context.Context, id string) (*Alert, error)
	List(ctx context.Context, filter *AlertFilter) ([]*Alert, error)
	Resolve(ctx context.Context, id string) error
	Subscribe(ctx context.Context, handler AlertHandler) error
}

// AlertFilter for filtering alerts
type AlertFilter struct {
	Types      []AlertType     `json:"types,omitempty"`
	Severities []AlertSeverity `json:"severities,omitempty"`
	Resolved   *bool           `json:"resolved,omitempty"`
	Limit      int             `json:"limit,omitempty"`
}

// LLMOpsConfig holds configuration for LLMOps
type LLMOpsConfig struct {
	EnableAutoEvaluation  bool               `json:"enable_auto_evaluation"`
	EvaluationInterval    time.Duration      `json:"evaluation_interval"`
	MinSamplesForSignif   int                `json:"min_samples_for_significance"`
	SignificanceLevel     float64            `json:"significance_level"`
	AlertThresholds       map[string]float64 `json:"alert_thresholds"`
	EnableDebateEval      bool               `json:"enable_debate_eval"`
}

// DefaultLLMOpsConfig returns default configuration
func DefaultLLMOpsConfig() *LLMOpsConfig {
	return &LLMOpsConfig{
		EnableAutoEvaluation: true,
		EvaluationInterval:   time.Hour,
		MinSamplesForSignif:  100,
		SignificanceLevel:    0.95,
		AlertThresholds: map[string]float64{
			"pass_rate_drop": 0.05,
			"latency_spike":  1.5,
		},
		EnableDebateEval: true,
	}
}
