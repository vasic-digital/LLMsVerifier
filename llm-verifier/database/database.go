package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents the main database interface
type Database struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*Database, error) {
	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure database connection
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{conn: db}

	// Initialize schema
	if err := database.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

// initializeSchema creates the database tables and indexes
func (d *Database) initializeSchema() error {
	schema := `
	-- Enable foreign keys
	PRAGMA foreign_keys = ON;

	-- Providers table
	CREATE TABLE IF NOT EXISTS providers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		endpoint TEXT NOT NULL,
		api_key_encrypted TEXT,
		description TEXT,
		website TEXT,
		support_email TEXT,
		documentation_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_checked TIMESTAMP,
		is_active BOOLEAN DEFAULT 1,
		reliability_score REAL DEFAULT 0.0,
		average_response_time_ms INTEGER DEFAULT 0
	);

	-- Models table
	CREATE TABLE IF NOT EXISTS models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id INTEGER NOT NULL,
		model_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		version TEXT,
		architecture TEXT,
		parameter_count INTEGER,
		context_window_tokens INTEGER,
		max_output_tokens INTEGER,
		training_data_cutoff DATE,
		release_date DATE,
		is_multimodal BOOLEAN DEFAULT 0,
		supports_vision BOOLEAN DEFAULT 0,
		supports_audio BOOLEAN DEFAULT 0,
		supports_video BOOLEAN DEFAULT 0,
		supports_reasoning BOOLEAN DEFAULT 0,
		open_source BOOLEAN DEFAULT 0,
		deprecated BOOLEAN DEFAULT 0,
		tags TEXT,
		language_support TEXT,
		use_case TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_verified TIMESTAMP,
		verification_status TEXT DEFAULT 'pending',
		overall_score REAL DEFAULT 0.0,
		code_capability_score REAL DEFAULT 0.0,
		responsiveness_score REAL DEFAULT 0.0,
		reliability_score REAL DEFAULT 0.0,
		feature_richness_score REAL DEFAULT 0.0,
		value_proposition_score REAL DEFAULT 0.0,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
	);

	-- Verification results table
	CREATE TABLE IF NOT EXISTS verification_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		verification_type TEXT NOT NULL,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		status TEXT DEFAULT 'running',
		error_message TEXT,
		"exists" BOOLEAN,
		responsive BOOLEAN,
		overloaded BOOLEAN,
		latency_ms INTEGER,
		supports_tool_use BOOLEAN DEFAULT 0,
		supports_function_calling BOOLEAN DEFAULT 0,
		supports_code_generation BOOLEAN DEFAULT 0,
		supports_code_completion BOOLEAN DEFAULT 0,
		supports_code_review BOOLEAN DEFAULT 0,
		supports_code_explanation BOOLEAN DEFAULT 0,
		supports_embeddings BOOLEAN DEFAULT 0,
		supports_reranking BOOLEAN DEFAULT 0,
		supports_image_generation BOOLEAN DEFAULT 0,
		supports_audio_generation BOOLEAN DEFAULT 0,
		supports_video_generation BOOLEAN DEFAULT 0,
		supports_mcps BOOLEAN DEFAULT 0,
		supports_lsps BOOLEAN DEFAULT 0,
		supports_multimodal BOOLEAN DEFAULT 0,
		supports_streaming BOOLEAN DEFAULT 0,
		supports_json_mode BOOLEAN DEFAULT 0,
		supports_structured_output BOOLEAN DEFAULT 0,
		supports_reasoning BOOLEAN DEFAULT 0,
		supports_parallel_tool_use BOOLEAN DEFAULT 0,
		max_parallel_calls INTEGER DEFAULT 0,
		supports_batch_processing BOOLEAN DEFAULT 0,
		code_language_support TEXT,
		code_debugging BOOLEAN DEFAULT 0,
		code_optimization BOOLEAN DEFAULT 0,
		test_generation BOOLEAN DEFAULT 0,
		documentation_generation BOOLEAN DEFAULT 0,
		refactoring BOOLEAN DEFAULT 0,
		error_resolution BOOLEAN DEFAULT 0,
		architecture_design BOOLEAN DEFAULT 0,
		security_assessment BOOLEAN DEFAULT 0,
		pattern_recognition BOOLEAN DEFAULT 0,
		debugging_accuracy REAL DEFAULT 0.0,
		max_handled_depth INTEGER DEFAULT 0,
		code_quality_score REAL DEFAULT 0.0,
		logic_correctness_score REAL DEFAULT 0.0,
		runtime_efficiency_score REAL DEFAULT 0.0,
		overall_score REAL DEFAULT 0.0,
		code_capability_score REAL DEFAULT 0.0,
		responsiveness_score REAL DEFAULT 0.0,
		reliability_score REAL DEFAULT 0.0,
		feature_richness_score REAL DEFAULT 0.0,
		value_proposition_score REAL DEFAULT 0.0,
		score_details TEXT,
		avg_latency_ms INTEGER DEFAULT 0,
		p95_latency_ms INTEGER DEFAULT 0,
		min_latency_ms INTEGER DEFAULT 0,
		max_latency_ms INTEGER DEFAULT 0,
		throughput_rps REAL DEFAULT 0.0,
		raw_request TEXT,
		raw_response TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Events table
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		severity TEXT DEFAULT 'info',
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		details TEXT,
		model_id INTEGER,
		provider_id INTEGER,
		verification_result_id INTEGER,
		issue_id INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
		FOREIGN KEY (verification_result_id) REFERENCES verification_results(id) ON DELETE CASCADE,
		FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
	);

	-- Configuration exports table
	CREATE TABLE IF NOT EXISTS config_exports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		export_type TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		config_data TEXT NOT NULL,
		target_models TEXT,
		target_providers TEXT,
		filters TEXT,
		is_verified BOOLEAN DEFAULT 0,
		verification_notes TEXT,
		created_by TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		download_count INTEGER DEFAULT 0
	);

	-- Logs table
	CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		level TEXT NOT NULL,
		logger TEXT NOT NULL,
		message TEXT NOT NULL,
		details TEXT,
		request_id TEXT,
		model_id INTEGER,
		provider_id INTEGER,
		verification_result_id INTEGER,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
		FOREIGN KEY (verification_result_id) REFERENCES verification_results(id) ON DELETE CASCADE
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_providers_endpoint ON providers(endpoint);
	CREATE INDEX IF NOT EXISTS idx_providers_active ON providers(is_active);
	CREATE INDEX IF NOT EXISTS idx_models_provider ON models(provider_id);
	CREATE INDEX IF NOT EXISTS idx_models_model_id ON models(model_id);
	CREATE INDEX IF NOT EXISTS idx_models_verification_status ON models(verification_status);
	CREATE INDEX IF NOT EXISTS idx_models_overall_score ON models(overall_score);
	CREATE INDEX IF NOT EXISTS idx_verification_results_model ON verification_results(model_id);
	CREATE INDEX IF NOT EXISTS idx_verification_results_status ON verification_results(status);
	CREATE INDEX IF NOT EXISTS idx_verification_results_timestamp ON verification_results(created_at);
	CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(created_at);
	CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
	`

	_, err := d.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.conn.Close()
}

// Transaction executes a function within a database transaction
func (d *Database) Transaction(fn func(*sql.Tx) error) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Provider represents a model provider
type Provider struct {
	ID                    int64      `json:"id"`
	Name                  string     `json:"name"`
	Endpoint              string     `json:"endpoint"`
	APIKeyEncrypted       string     `json:"api_key_encrypted"`
	Description           string     `json:"description"`
	Website               string     `json:"website"`
	SupportEmail          string     `json:"support_email"`
	DocumentationURL      string     `json:"documentation_url"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastChecked           *time.Time `json:"last_checked"`
	IsActive              bool       `json:"is_active"`
	ReliabilityScore      float64    `json:"reliability_score"`
	AverageResponseTimeMs int        `json:"average_response_time_ms"`
}

// Model represents an LLM model
type Model struct {
	ID                    int64      `json:"id"`
	ProviderID            int64      `json:"provider_id"`
	ModelID               string     `json:"model_id"`
	Name                  string     `json:"name"`
	Description           string     `json:"description"`
	Version               string     `json:"version"`
	Architecture          string     `json:"architecture"`
	ParameterCount        *int64     `json:"parameter_count"`
	ContextWindowTokens   *int       `json:"context_window_tokens"`
	MaxOutputTokens       *int       `json:"max_output_tokens"`
	TrainingDataCutoff    *time.Time `json:"training_data_cutoff"`
	ReleaseDate           *time.Time `json:"release_date"`
	IsMultimodal          bool       `json:"is_multimodal"`
	SupportsVision        bool       `json:"supports_vision"`
	SupportsAudio         bool       `json:"supports_audio"`
	SupportsVideo         bool       `json:"supports_video"`
	SupportsReasoning     bool       `json:"supports_reasoning"`
	OpenSource            bool       `json:"open_source"`
	Deprecated            bool       `json:"deprecated"`
	Tags                  []string   `json:"tags"`
	LanguageSupport       []string   `json:"language_support"`
	UseCase               string     `json:"use_case"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastVerified          *time.Time `json:"last_verified"`
	VerificationStatus    string     `json:"verification_status"`
	OverallScore          float64    `json:"overall_score"`
	CodeCapabilityScore   float64    `json:"code_capability_score"`
	ResponsivenessScore   float64    `json:"responsiveness_score"`
	ReliabilityScore      float64    `json:"reliability_score"`
	FeatureRichnessScore  float64    `json:"feature_richness_score"`
	ValuePropositionScore float64    `json:"value_proposition_score"`
}

// VerificationResult represents a verification run result
type VerificationResult struct {
	ID                       int64      `json:"id"`
	ModelID                  int64      `json:"model_id"`
	VerificationType         string     `json:"verification_type"`
	StartedAt                time.Time  `json:"started_at"`
	CompletedAt              *time.Time `json:"completed_at"`
	Status                   string     `json:"status"`
	ErrorMessage             *string    `json:"error_message"`
	Exists                   *bool      `json:"exists"`
	Responsive               *bool      `json:"responsive"`
	Overloaded               *bool      `json:"overloaded"`
	LatencyMs                *int       `json:"latency_ms"`
	SupportsToolUse          bool       `json:"supports_tool_use"`
	SupportsFunctionCalling  bool       `json:"supports_function_calling"`
	SupportsCodeGeneration   bool       `json:"supports_code_generation"`
	SupportsCodeCompletion   bool       `json:"supports_code_completion"`
	SupportsCodeReview       bool       `json:"supports_code_review"`
	SupportsCodeExplanation  bool       `json:"supports_code_explanation"`
	SupportsEmbeddings       bool       `json:"supports_embeddings"`
	SupportsReranking        bool       `json:"supports_reranking"`
	SupportsImageGeneration  bool       `json:"supports_image_generation"`
	SupportsAudioGeneration  bool       `json:"supports_audio_generation"`
	SupportsVideoGeneration  bool       `json:"supports_video_generation"`
	SupportsMCPs             bool       `json:"supports_mcps"`
	SupportsLSPs             bool       `json:"supports_lsps"`
	SupportsMultimodal       bool       `json:"supports_multimodal"`
	SupportsStreaming        bool       `json:"supports_streaming"`
	SupportsJSONMode         bool       `json:"supports_json_mode"`
	SupportsStructuredOutput bool       `json:"supports_structured_output"`
	SupportsReasoning        bool       `json:"supports_reasoning"`
	SupportsParallelToolUse  bool       `json:"supports_parallel_tool_use"`
	MaxParallelCalls         int        `json:"max_parallel_calls"`
	SupportsBatchProcessing  bool       `json:"supports_batch_processing"`
	CodeLanguageSupport      []string   `json:"code_language_support"`
	CodeDebugging            bool       `json:"code_debugging"`
	CodeOptimization         bool       `json:"code_optimization"`
	TestGeneration           bool       `json:"test_generation"`
	DocumentationGeneration  bool       `json:"documentation_generation"`
	Refactoring              bool       `json:"refactoring"`
	ErrorResolution          bool       `json:"error_resolution"`
	ArchitectureDesign       bool       `json:"architecture_design"`
	SecurityAssessment       bool       `json:"security_assessment"`
	PatternRecognition       bool       `json:"pattern_recognition"`
	DebuggingAccuracy        float64    `json:"debugging_accuracy"`
	MaxHandledDepth          int        `json:"max_handled_depth"`
	CodeQualityScore         float64    `json:"code_quality_score"`
	LogicCorrectnessScore    float64    `json:"logic_correctness_score"`
	RuntimeEfficiencyScore   float64    `json:"runtime_efficiency_score"`
	OverallScore             float64    `json:"overall_score"`
	CodeCapabilityScore      float64    `json:"code_capability_score"`
	ResponsivenessScore      float64    `json:"responsiveness_score"`
	ReliabilityScore         float64    `json:"reliability_score"`
	FeatureRichnessScore     float64    `json:"feature_richness_score"`
	ValuePropositionScore    float64    `json:"value_proposition_score"`
	ScoreDetails             string     `json:"score_details"`
	AvgLatencyMs             int        `json:"avg_latency_ms"`
	P95LatencyMs             int        `json:"p95_latency_ms"`
	MinLatencyMs             int        `json:"min_latency_ms"`
	MaxLatencyMs             int        `json:"max_latency_ms"`
	ThroughputRPS            float64    `json:"throughput_rps"`
	RawRequest               *string    `json:"raw_request"`
	RawResponse              *string    `json:"raw_response"`
	CreatedAt                time.Time  `json:"created_at"`
}

// Event represents a system event
type Event struct {
	ID                   int64     `json:"id"`
	EventType            string    `json:"event_type"`
	Severity             string    `json:"severity"`
	Title                string    `json:"title"`
	Message              string    `json:"message"`
	Details              *string   `json:"details"`
	ModelID              *int64    `json:"model_id"`
	ProviderID           *int64    `json:"provider_id"`
	VerificationResultID *int64    `json:"verification_result_id"`
	IssueID              *int64    `json:"issue_id"`
	CreatedAt            time.Time `json:"created_at"`
}

// Schedule represents a scheduled verification task
type Schedule struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Description     *string    `json:"description"`
	ScheduleType    string     `json:"schedule_type"` // cron, interval, manual
	CronExpression  *string    `json:"cron_expression"`
	IntervalSeconds *int       `json:"interval_seconds"`
	TargetType      string     `json:"target_type"` // all_models, provider, specific_model
	TargetID        *int64     `json:"target_id"`   // provider_id or model_id depending on target_type
	IsActive        bool       `json:"is_active"`
	LastRun         *time.Time `json:"last_run"`
	NextRun         *time.Time `json:"next_run"`
	RunCount        int        `json:"run_count"`
	MaxRuns         *int       `json:"max_runs"` // NULL for unlimited
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedBy       *string    `json:"created_by"` // For future multi-user support
}

// ConfigExport represents a configuration export
type ConfigExport struct {
	ID                int64     `json:"id"`
	ExportType        string    `json:"export_type"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	ConfigData        string    `json:"config_data"`
	TargetModels      *string   `json:"target_models"`
	TargetProviders   *string   `json:"target_providers"`
	Filters           *string   `json:"filters"`
	IsVerified        bool      `json:"is_verified"`
	VerificationNotes *string   `json:"verification_notes"`
	CreatedBy         *string   `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DownloadCount     int       `json:"download_count"`
}

// LogEntry represents a log entry
type LogEntry struct {
	ID                   int64     `json:"id"`
	Timestamp            time.Time `json:"timestamp"`
	Level                string    `json:"level"`
	Logger               string    `json:"logger"`
	Message              string    `json:"message"`
	Details              *string   `json:"details"`
	RequestID            *string   `json:"request_id"`
	UserID               *int64    `json:"user_id"` // For future multi-user support
	ModelID              *int64    `json:"model_id"`
	ProviderID           *int64    `json:"provider_id"`
	VerificationResultID *int64    `json:"verification_result_id"`
}

// Helper function to scan nullable time
func scanNullableTime(nullTime sql.NullTime) *time.Time {
	if nullTime.Valid {
		return &nullTime.Time
	}
	return nil
}

// Helper function to scan nullable string
func scanNullableString(nullString sql.NullString) *string {
	if nullString.Valid {
		return &nullString.String
	}
	return nil
}

// Helper function to scan nullable int64
func scanNullableInt64(nullInt64 sql.NullInt64) *int64 {
	if nullInt64.Valid {
		return &nullInt64.Int64
	}
	return nil
}

// Helper function to scan nullable int
func scanNullableInt(nullInt sql.NullInt32) *int {
	if nullInt.Valid {
		val := int(nullInt.Int32)
		return &val
	}
	return nil
}

// Helper function to scan nullable float64
func scanNullableFloat64(nullFloat sql.NullFloat64) *float64 {
	if nullFloat.Valid {
		return &nullFloat.Float64
	}
	return nil
}

// Helper function to scan nullable bool
func scanNullableBool(nullBool sql.NullBool) *bool {
	if nullBool.Valid {
		return &nullBool.Bool
	}
	return nil
}

// Helper function to scan nullable bool from string
func scanNullableBoolFromString(nullString sql.NullString) *bool {
	if !nullString.Valid || nullString.String == "" {
		return nil
	}

	if nullString.String == "true" || nullString.String == "1" {
		val := true
		return &val
	} else if nullString.String == "false" || nullString.String == "0" {
		val := false
		return &val
	}

	return nil
}

// Helper function to scan nullable time from string
func scanNullableTimeFromString(nullString sql.NullString) *time.Time {
	if !nullString.Valid || nullString.String == "" {
		return nil
	}

	// Try to parse as RFC3339 timestamp
	if t, err := time.Parse(time.RFC3339, nullString.String); err == nil {
		return &t
	}

	// Try to parse as Unix timestamp
	if timestamp, err := strconv.ParseInt(nullString.String, 10, 64); err == nil {
		t := time.Unix(timestamp, 0)
		return &t
	}

	return nil
}

// Helper function to scan nullable int from int64
func scanNullableIntFromInt64(nullInt64 sql.NullInt64) *int {
	if nullInt64.Valid {
		val := int(nullInt64.Int64)
		return &val
	}
	return nil
}

// Helper function to scan JSON string
func scanJSONString(nullString sql.NullString) []string {
	if !nullString.Valid || nullString.String == "" {
		return []string{}
	}

	var result []string
	if err := json.Unmarshal([]byte(nullString.String), &result); err != nil {
		return []string{}
	}
	return result
}

// Pricing represents model pricing information
type Pricing struct {
	ID                   int64      `json:"id"`
	ModelID              int64      `json:"model_id"`
	InputTokenCost       float64    `json:"input_token_cost"`
	OutputTokenCost      float64    `json:"output_token_cost"`
	CachedInputTokenCost float64    `json:"cached_input_token_cost"`
	StorageCost          float64    `json:"storage_cost"`
	RequestCost          float64    `json:"request_cost"`
	Currency             string     `json:"currency"`
	PricingModel         string     `json:"pricing_model"`
	EffectiveFrom        *time.Time `json:"effective_from"`
	EffectiveTo          *time.Time `json:"effective_to"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// Limit represents rate limits and quotas
type Limit struct {
	ID           int64      `json:"id"`
	ModelID      int64      `json:"model_id"`
	LimitType    string     `json:"limit_type"`
	LimitValue   int        `json:"limit_value"`
	CurrentUsage int        `json:"current_usage"`
	ResetPeriod  string     `json:"reset_period"`
	ResetTime    *time.Time `json:"reset_time"`
	IsHardLimit  bool       `json:"is_hard_limit"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Issue represents documented problems with models
type Issue struct {
	ID                   int64      `json:"id"`
	ModelID              int64      `json:"model_id"`
	IssueType            string     `json:"issue_type"`
	Severity             string     `json:"severity"`
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	Symptoms             *string    `json:"symptoms"`
	Workarounds          *string    `json:"workarounds"`
	AffectedFeatures     []string   `json:"affected_features"`
	FirstDetected        time.Time  `json:"first_detected"`
	LastOccurred         *time.Time `json:"last_occurred"`
	ResolvedAt           *time.Time `json:"resolved_at"`
	ResolutionNotes      *string    `json:"resolution_notes"`
	VerificationResultID *int64     `json:"verification_result_id"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
