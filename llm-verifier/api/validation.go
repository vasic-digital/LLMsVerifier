package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
)

func init() {
	validate = validator.New()
	validate.SetTagName("binding")

	// Register custom validations
	validate.RegisterValidation("alphanumspace", validateAlphaNumSpace)
	validate.RegisterValidation("url", validateURL)
	validate.RegisterValidation("email", validateEmail)
	validate.RegisterValidation("cron", validateCron)
	validate.RegisterValidation("severity", validateSeverity)
	validate.RegisterValidation("event_type", validateEventType)
	validate.RegisterValidation("verification_type", validateVerificationType)
	validate.RegisterValidation("status", validateStatus)
	validate.RegisterValidation("schedule_type", validateScheduleType)
	validate.RegisterValidation("target_type", validateTargetType)
	validate.RegisterValidation("export_type", validateExportType)
	validate.RegisterValidation("issue_type", validateIssueType)
	validate.RegisterValidation("pricing_model", validatePricingModel)
	validate.RegisterValidation("limit_type", validateLimitType)
	validate.RegisterValidation("reset_period", validateResetPeriod)
	validate.RegisterValidation("port", validatePort)
}

// Custom validation functions
func validateAlphaNumSpace(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	re := regexp.MustCompile(`^[a-zA-Z0-9\s\-_\.]+$`)
	return re.MatchString(value)
}

func validateURL(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Optional field
	}
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func validateEmail(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Optional field
	}
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(value)
}

func validateCron(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Optional field
	}
	// Basic cron validation - can be enhanced
	parts := strings.Fields(value)
	return len(parts) == 5
}

func validateSeverity(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validSeverities := []string{"info", "warning", "error", "critical"}
	for _, s := range validSeverities {
		if value == s {
			return true
		}
	}
	return false
}

func validateEventType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validTypes := []string{
		"verification_started", "verification_completed", "verification_failed",
		"model_added", "model_updated", "model_deleted",
		"provider_added", "provider_updated", "provider_deleted",
		"schedule_created", "schedule_updated", "schedule_deleted", "schedule_executed",
		"system_error", "rate_limit_exceeded", "authentication_failed",
	}
	for _, t := range validTypes {
		if value == t {
			return true
		}
	}
	return false
}

func validateVerificationType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validTypes := []string{"basic", "comprehensive", "performance", "security", "custom"}
	for _, t := range validTypes {
		if value == t {
			return true
		}
	}
	return false
}

func validateStatus(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validStatuses := []string{
		"pending", "running", "completed", "failed", "cancelled",
		"success", "error", "warning", "info",
	}
	for _, s := range validStatuses {
		if value == s {
			return true
		}
	}
	return false
}

func validateScheduleType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validTypes := []string{"cron", "interval", "manual"}
	for _, t := range validTypes {
		if value == t {
			return true
		}
	}
	return false
}

func validateTargetType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validTypes := []string{"all_models", "provider", "specific_model"}
	for _, t := range validTypes {
		if value == t {
			return true
		}
	}
	return false
}

func validateExportType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validTypes := []string{"json", "yaml", "csv", "html", "pdf", "opencode", "claude", "crush", "vscode"}
	for _, t := range validTypes {
		if value == t {
			return true
		}
	}
	return false
}

func validateIssueType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validTypes := []string{
		"availability", "performance", "accuracy", "security", "reliability",
		"compatibility", "documentation", "pricing", "limit", "other",
	}
	for _, t := range validTypes {
		if value == t {
			return true
		}
	}
	return false
}

func validatePricingModel(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validModels := []string{
		"per_token", "per_request", "subscription", "tiered", "hybrid", "custom",
	}
	for _, m := range validModels {
		if value == m {
			return true
		}
	}
	return false
}

func validateLimitType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validTypes := []string{
		"requests_per_minute", "requests_per_hour", "requests_per_day",
		"tokens_per_minute", "tokens_per_hour", "tokens_per_day",
		"concurrent_requests", "rate_limit", "quota", "other",
	}
	for _, t := range validTypes {
		if value == t {
			return true
		}
	}
	return false
}

func validateResetPeriod(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validPeriods := []string{
		"minute", "hour", "day", "week", "month", "year", "never",
	}
	for _, p := range validPeriods {
		if value == p {
			return true
		}
	}
	return false
}

func validatePort(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Optional field
	}
	// Check if it's a valid port number (1-65535)
	re := regexp.MustCompile(`^[1-9][0-9]*$`)
	if !re.MatchString(value) {
		return false
	}
	// Check if it's within valid port range
	portNum := 0
	fmt.Sscanf(value, "%d", &portNum)
	return portNum >= 1 && portNum <= 65535
}

// Request structs with validation tags

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50,alphanumspace"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

// CreateProviderRequest represents provider creation request
type CreateProviderRequest struct {
	Name                  string  `json:"name" binding:"required,min=2,max=100,alphanumspace"`
	Endpoint              string  `json:"endpoint" binding:"required,url"`
	APIKeyEncrypted       string  `json:"api_key_encrypted" binding:"required,min=10"`
	Description           string  `json:"description" binding:"max=500"`
	Website               string  `json:"website" binding:"url"`
	SupportEmail          string  `json:"support_email" binding:"email"`
	DocumentationURL      string  `json:"documentation_url" binding:"url"`
	IsActive              bool    `json:"is_active"`
	ReliabilityScore      float64 `json:"reliability_score" binding:"omitempty,min=0,max=100"`
	AverageResponseTimeMs int     `json:"average_response_time_ms" binding:"omitempty,min=0"`
}

// UpdateProviderRequest represents provider update request
type UpdateProviderRequest struct {
	Name                  string  `json:"name" binding:"omitempty,min=2,max=100,alphanumspace"`
	Endpoint              string  `json:"endpoint" binding:"omitempty,url"`
	APIKeyEncrypted       string  `json:"api_key_encrypted" binding:"omitempty,min=10"`
	Description           string  `json:"description" binding:"omitempty,max=500"`
	Website               string  `json:"website" binding:"omitempty,url"`
	SupportEmail          string  `json:"support_email" binding:"omitempty,email"`
	DocumentationURL      string  `json:"documentation_url" binding:"omitempty,url"`
	IsActive              *bool   `json:"is_active"`
	ReliabilityScore      float64 `json:"reliability_score" binding:"omitempty,min=0,max=100"`
	AverageResponseTimeMs int     `json:"average_response_time_ms" binding:"omitempty,min=0"`
}

// CreateModelRequest represents model creation request
type CreateModelRequest struct {
	ProviderID            int64      `json:"provider_id" binding:"required,min=1"`
	ModelID               string     `json:"model_id" binding:"required,min=1,max=100,alphanumspace"`
	Name                  string     `json:"name" binding:"required,min=2,max=100,alphanumspace"`
	Description           string     `json:"description" binding:"max=500"`
	Version               string     `json:"version" binding:"max=50"`
	Architecture          string     `json:"architecture" binding:"max=100"`
	ParameterCount        *int64     `json:"parameter_count" binding:"omitempty,min=0"`
	ContextWindowTokens   *int       `json:"context_window_tokens" binding:"omitempty,min=0"`
	MaxOutputTokens       *int       `json:"max_output_tokens" binding:"omitempty,min=0"`
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
	UseCase               string     `json:"use_case" binding:"max=200"`
	VerificationStatus    string     `json:"verification_status" binding:"oneof=pending verified failed"`
	OverallScore          float64    `json:"overall_score" binding:"min=0,max=100"`
	CodeCapabilityScore   float64    `json:"code_capability_score" binding:"min=0,max=100"`
	ResponsivenessScore   float64    `json:"responsiveness_score" binding:"min=0,max=100"`
	ReliabilityScore      float64 `json:"reliability_score" binding:"omitempty,min=0,max=100"`
	FeatureRichnessScore  float64    `json:"feature_richness_score" binding:"min=0,max=100"`
	ValuePropositionScore float64    `json:"value_proposition_score" binding:"min=0,max=100"`
}

// UpdateModelRequest represents model update request
type UpdateModelRequest struct {
	ProviderID            int64      `json:"provider_id" binding:"omitempty,min=1"`
	ModelID               string     `json:"model_id" binding:"omitempty,min=1,max=100,alphanumspace"`
	Name                  string     `json:"name" binding:"omitempty,min=2,max=100,alphanumspace"`
	Description           string     `json:"description" binding:"omitempty,max=500"`
	Version               string     `json:"version" binding:"omitempty,max=50"`
	Architecture          string     `json:"architecture" binding:"omitempty,max=100"`
	ParameterCount        *int64     `json:"parameter_count" binding:"omitempty,min=0"`
	ContextWindowTokens   *int       `json:"context_window_tokens" binding:"omitempty,min=0"`
	MaxOutputTokens       *int       `json:"max_output_tokens" binding:"omitempty,min=0"`
	TrainingDataCutoff    *time.Time `json:"training_data_cutoff"`
	ReleaseDate           *time.Time `json:"release_date"`
	IsMultimodal          *bool      `json:"is_multimodal"`
	SupportsVision        *bool      `json:"supports_vision"`
	SupportsAudio         *bool      `json:"supports_audio"`
	SupportsVideo         *bool      `json:"supports_video"`
	SupportsReasoning     *bool      `json:"supports_reasoning"`
	OpenSource            *bool      `json:"open_source"`
	Deprecated            *bool      `json:"deprecated"`
	Tags                  []string   `json:"tags"`
	LanguageSupport       []string   `json:"language_support"`
	UseCase               string     `json:"use_case" binding:"omitempty,max=200"`
	VerificationStatus    string     `json:"verification_status" binding:"omitempty,oneof=pending verified failed"`
	OverallScore          float64    `json:"overall_score" binding:"omitempty,min=0,max=100"`
	CodeCapabilityScore   float64    `json:"code_capability_score" binding:"omitempty,min=0,max=100"`
	ResponsivenessScore   float64    `json:"responsiveness_score" binding:"omitempty,min=0,max=100"`
	ReliabilityScore      float64    `json:"reliability_score" binding:"omitempty,min=0,max=100"`
	FeatureRichnessScore  float64    `json:"feature_richness_score" binding:"omitempty,min=0,max=100"`
	ValuePropositionScore float64    `json:"value_proposition_score" binding:"omitempty,min=0,max=100"`
}

// CreateVerificationResultRequest represents verification result creation request
type CreateVerificationResultRequest struct {
	ModelID                  int64      `json:"model_id" binding:"required,min=1"`
	VerificationType         string     `json:"verification_type" binding:"required,verification_type"`
	StartedAt                time.Time  `json:"started_at" binding:"required"`
	CompletedAt              *time.Time `json:"completed_at"`
	Status                   string     `json:"status" binding:"required,status"`
	ErrorMessage             *string    `json:"error_message" binding:"omitempty,max=1000"`
	ModelExists              *bool      `json:"model_exists"`
	Responsive               *bool      `json:"responsive"`
	Overloaded               *bool      `json:"overloaded"`
	LatencyMs                *int       `json:"latency_ms" binding:"omitempty,min=0"`
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
	MaxParallelCalls         int        `json:"max_parallel_calls" binding:"min=0"`
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
	DebuggingAccuracy        float64    `json:"debugging_accuracy" binding:"min=0,max=100"`
	MaxHandledDepth          int        `json:"max_handled_depth" binding:"min=0"`
	CodeQualityScore         float64    `json:"code_quality_score" binding:"min=0,max=100"`
	LogicCorrectnessScore    float64    `json:"logic_correctness_score" binding:"min=0,max=100"`
	RuntimeEfficiencyScore   float64    `json:"runtime_efficiency_score" binding:"min=0,max=100"`
	OverallScore             float64    `json:"overall_score" binding:"min=0,max=100"`
	CodeCapabilityScore      float64    `json:"code_capability_score" binding:"min=0,max=100"`
	ResponsivenessScore      float64    `json:"responsiveness_score" binding:"min=0,max=100"`
	ReliabilityScore      float64 `json:"reliability_score" binding:"omitempty,min=0,max=100"`
	FeatureRichnessScore     float64    `json:"feature_richness_score" binding:"min=0,max=100"`
	ValuePropositionScore    float64    `json:"value_proposition_score" binding:"min=0,max=100"`
	ScoreDetails             string     `json:"score_details" binding:"omitempty,max=5000"`
	AvgLatencyMs             int        `json:"avg_latency_ms" binding:"min=0"`
	P95LatencyMs             int        `json:"p95_latency_ms" binding:"min=0"`
	MinLatencyMs             int        `json:"min_latency_ms" binding:"min=0"`
	MaxLatencyMs             int        `json:"max_latency_ms" binding:"min=0"`
	ThroughputRPS            float64    `json:"throughput_rps" binding:"min=0"`
	RawRequest               *string    `json:"raw_request" binding:"omitempty,max=10000"`
	RawResponse              *string    `json:"raw_response" binding:"omitempty,max=10000"`
}

// UpdateVerificationResultRequest represents verification result update request
type UpdateVerificationResultRequest struct {
	ModelID                  int64      `json:"model_id" binding:"omitempty,min=1"`
	VerificationType         string     `json:"verification_type" binding:"omitempty,verification_type"`
	StartedAt                time.Time  `json:"started_at"`
	CompletedAt              *time.Time `json:"completed_at"`
	Status                   string     `json:"status" binding:"omitempty,status"`
	ErrorMessage             *string    `json:"error_message" binding:"omitempty,max=1000"`
	ModelExists              *bool      `json:"model_exists"`
	Responsive               *bool      `json:"responsive"`
	Overloaded               *bool      `json:"overloaded"`
	LatencyMs                *int       `json:"latency_ms" binding:"omitempty,min=0"`
	SupportsToolUse          *bool      `json:"supports_tool_use"`
	SupportsFunctionCalling  *bool      `json:"supports_function_calling"`
	SupportsCodeGeneration   *bool      `json:"supports_code_generation"`
	SupportsCodeCompletion   *bool      `json:"supports_code_completion"`
	SupportsCodeReview       *bool      `json:"supports_code_review"`
	SupportsCodeExplanation  *bool      `json:"supports_code_explanation"`
	SupportsEmbeddings       *bool      `json:"supports_embeddings"`
	SupportsReranking        *bool      `json:"supports_reranking"`
	SupportsImageGeneration  *bool      `json:"supports_image_generation"`
	SupportsAudioGeneration  *bool      `json:"supports_audio_generation"`
	SupportsVideoGeneration  *bool      `json:"supports_video_generation"`
	SupportsMCPs             *bool      `json:"supports_mcps"`
	SupportsLSPs             *bool      `json:"supports_lsps"`
	SupportsMultimodal       *bool      `json:"supports_multimodal"`
	SupportsStreaming        *bool      `json:"supports_streaming"`
	SupportsJSONMode         *bool      `json:"supports_json_mode"`
	SupportsStructuredOutput *bool      `json:"supports_structured_output"`
	SupportsReasoning        *bool      `json:"supports_reasoning"`
	SupportsParallelToolUse  *bool      `json:"supports_parallel_tool_use"`
	MaxParallelCalls         int        `json:"max_parallel_calls" binding:"omitempty,min=0"`
	SupportsBatchProcessing  *bool      `json:"supports_batch_processing"`
	CodeLanguageSupport      []string   `json:"code_language_support"`
	CodeDebugging            *bool      `json:"code_debugging"`
	CodeOptimization         *bool      `json:"code_optimization"`
	TestGeneration           *bool      `json:"test_generation"`
	DocumentationGeneration  *bool      `json:"documentation_generation"`
	Refactoring              *bool      `json:"refactoring"`
	ErrorResolution          *bool      `json:"error_resolution"`
	ArchitectureDesign       *bool      `json:"architecture_design"`
	SecurityAssessment       *bool      `json:"security_assessment"`
	PatternRecognition       *bool      `json:"pattern_recognition"`
	DebuggingAccuracy        float64    `json:"debugging_accuracy" binding:"omitempty,min=0,max=100"`
	MaxHandledDepth          int        `json:"max_handled_depth" binding:"omitempty,min=0"`
	CodeQualityScore         float64    `json:"code_quality_score" binding:"omitempty,min=0,max=100"`
	LogicCorrectnessScore    float64    `json:"logic_correctness_score" binding:"omitempty,min=0,max=100"`
	RuntimeEfficiencyScore   float64    `json:"runtime_efficiency_score" binding:"omitempty,min=0,max=100"`
	OverallScore             float64    `json:"overall_score" binding:"omitempty,min=0,max=100"`
	CodeCapabilityScore      float64    `json:"code_capability_score" binding:"omitempty,min=0,max=100"`
	ResponsivenessScore      float64    `json:"responsiveness_score" binding:"omitempty,min=0,max=100"`
	ReliabilityScore      float64 `json:"reliability_score" binding:"omitempty,min=0,max=100"`
	FeatureRichnessScore     float64    `json:"feature_richness_score" binding:"omitempty,min=0,max=100"`
	ValuePropositionScore    float64    `json:"value_proposition_score" binding:"omitempty,min=0,max=100"`
	ScoreDetails             string     `json:"score_details" binding:"omitempty,max=5000"`
	AvgLatencyMs             int        `json:"avg_latency_ms" binding:"omitempty,min=0"`
	P95LatencyMs             int        `json:"p95_latency_ms" binding:"omitempty,min=0"`
	MinLatencyMs             int        `json:"min_latency_ms" binding:"omitempty,min=0"`
	MaxLatencyMs             int        `json:"max_latency_ms" binding:"omitempty,min=0"`
	ThroughputRPS            float64    `json:"throughput_rps" binding:"omitempty,min=0"`
	RawRequest               *string    `json:"raw_request" binding:"omitempty,max=10000"`
	RawResponse              *string    `json:"raw_response" binding:"omitempty,max=10000"`
}

// CreateEventRequest represents event creation request
type CreateEventRequest struct {
	EventType            string  `json:"event_type" binding:"required,event_type"`
	Severity             string  `json:"severity" binding:"required,severity"`
	Title                string  `json:"title" binding:"required,min=2,max=200"`
	Message              string  `json:"message" binding:"required,min=2,max=1000"`
	Details              *string `json:"details" binding:"omitempty,max=5000"`
	ModelID              *int64  `json:"model_id" binding:"omitempty,min=1"`
	ProviderID           *int64  `json:"provider_id" binding:"omitempty,min=1"`
	VerificationResultID *int64  `json:"verification_result_id" binding:"omitempty,min=1"`
	IssueID              *int64  `json:"issue_id" binding:"omitempty,min=1"`
}

// UpdateEventRequest represents event update request
type UpdateEventRequest struct {
	EventType            string  `json:"event_type" binding:"omitempty,event_type"`
	Severity             string  `json:"severity" binding:"omitempty,severity"`
	Title                string  `json:"title" binding:"omitempty,min=2,max=200"`
	Message              string  `json:"message" binding:"omitempty,min=2,max=1000"`
	Details              *string `json:"details" binding:"omitempty,max=5000"`
	ModelID              *int64  `json:"model_id" binding:"omitempty,min=1"`
	ProviderID           *int64  `json:"provider_id" binding:"omitempty,min=1"`
	VerificationResultID *int64  `json:"verification_result_id" binding:"omitempty,min=1"`
	IssueID              *int64  `json:"issue_id" binding:"omitempty,min=1"`
}

// CreateScheduleRequest represents schedule creation request
type CreateScheduleRequest struct {
	Name            string  `json:"name" binding:"required,min=2,max=100,alphanumspace"`
	Description     *string `json:"description" binding:"omitempty,max=500"`
	ScheduleType    string  `json:"schedule_type" binding:"required,schedule_type"`
	CronExpression  *string `json:"cron_expression" binding:"omitempty,cron"`
	IntervalSeconds *int    `json:"interval_seconds" binding:"omitempty,min=60"`
	TargetType      string  `json:"target_type" binding:"required,target_type"`
	TargetID        *int64  `json:"target_id" binding:"omitempty,min=1"`
	IsActive        bool    `json:"is_active"`
	MaxRuns         *int    `json:"max_runs" binding:"omitempty,min=1"`
}

// UpdateScheduleRequest represents schedule update request
type UpdateScheduleRequest struct {
	Name            string  `json:"name" binding:"omitempty,min=2,max=100,alphanumspace"`
	Description     *string `json:"description" binding:"omitempty,max=500"`
	ScheduleType    string  `json:"schedule_type" binding:"omitempty,schedule_type"`
	CronExpression  *string `json:"cron_expression" binding:"omitempty,cron"`
	IntervalSeconds *int    `json:"interval_seconds" binding:"omitempty,min=60"`
	TargetType      string  `json:"target_type" binding:"omitempty,target_type"`
	TargetID        *int64  `json:"target_id" binding:"omitempty,min=1"`
	IsActive        *bool   `json:"is_active"`
	MaxRuns         *int    `json:"max_runs" binding:"omitempty,min=1"`
}

// GenerateReportRequest represents report generation request
type GenerateReportRequest struct {
	ReportType string  `json:"report_type" binding:"required,oneof=summary detailed comparison"`
	ModelIDs   []int64 `json:"model_ids,omitempty"`
	StartDate  string  `json:"start_date,omitempty"`
	EndDate    string  `json:"end_date,omitempty"`
	Format     string  `json:"format" binding:"required,oneof=json html pdf"`
}

// UpdateConfigRequest represents configuration update request
type UpdateConfigRequest struct {
	Key   string      `json:"key" binding:"required,min=2,max=100,alphanumspace"`
	Value interface{} `json:"value" binding:"required"`
}

// UpdateSystemConfigRequest represents system configuration update request
type UpdateSystemConfigRequest struct {
	Concurrency *int           `json:"concurrency,omitempty" binding:"omitempty,min=1,max=100"`
	Timeout     *time.Duration `json:"timeout,omitempty" binding:"omitempty,min=1000000000,max=600000000000"` // 1s to 10m in nanoseconds
	API         *struct {
		Port       *string `json:"port,omitempty" binding:"omitempty,min=2,max=10,port"`
		RateLimit  *int    `json:"rate_limit,omitempty" binding:"omitempty,min=1,max=1000"`
		EnableCORS *bool   `json:"enable_cors,omitempty"`
	} `json:"api,omitempty"`
}

// CreateIssueRequest represents issue creation request
type CreateIssueRequest struct {
	ModelID              int64      `json:"model_id" binding:"required,min=1"`
	IssueType            string     `json:"issue_type" binding:"required,issue_type"`
	Severity             string     `json:"severity" binding:"required,severity"`
	Title                string     `json:"title" binding:"required,min=2,max=200"`
	Description          string     `json:"description" binding:"required,min=2,max=1000"`
	Symptoms             *string    `json:"symptoms" binding:"omitempty,max=1000"`
	Workarounds          *string    `json:"workarounds" binding:"omitempty,max=1000"`
	AffectedFeatures     []string   `json:"affected_features"`
	FirstDetected        time.Time  `json:"first_detected" binding:"required"`
	LastOccurred         *time.Time `json:"last_occurred"`
	ResolvedAt           *time.Time `json:"resolved_at"`
	ResolutionNotes      *string    `json:"resolution_notes" binding:"omitempty,max=1000"`
	VerificationResultID *int64     `json:"verification_result_id" binding:"omitempty,min=1"`
}

// UpdateIssueRequest represents issue update request
type UpdateIssueRequest struct {
	ModelID              int64      `json:"model_id" binding:"omitempty,min=1"`
	IssueType            string     `json:"issue_type" binding:"omitempty,issue_type"`
	Severity             string     `json:"severity" binding:"omitempty,severity"`
	Title                string     `json:"title" binding:"omitempty,min=2,max=200"`
	Description          string     `json:"description" binding:"omitempty,min=2,max=1000"`
	Symptoms             *string    `json:"symptoms" binding:"omitempty,max=1000"`
	Workarounds          *string    `json:"workarounds" binding:"omitempty,max=1000"`
	AffectedFeatures     []string   `json:"affected_features"`
	FirstDetected        time.Time  `json:"first_detected"`
	LastOccurred         *time.Time `json:"last_occurred"`
	ResolvedAt           *time.Time `json:"resolved_at"`
	ResolutionNotes      *string    `json:"resolution_notes" binding:"omitempty,max=1000"`
	VerificationResultID *int64     `json:"verification_result_id" binding:"omitempty,min=1"`
}

// CreatePricingRequest represents pricing creation request
type CreatePricingRequest struct {
	ModelID              int64      `json:"model_id" binding:"required,min=1"`
	InputTokenCost       float64    `json:"input_token_cost" binding:"required,min=0"`
	OutputTokenCost      float64    `json:"output_token_cost" binding:"required,min=0"`
	CachedInputTokenCost float64    `json:"cached_input_token_cost" binding:"min=0"`
	StorageCost          float64    `json:"storage_cost" binding:"min=0"`
	RequestCost          float64    `json:"request_cost" binding:"min=0"`
	Currency             string     `json:"currency" binding:"required,min=3,max=3"`
	PricingModel         string     `json:"pricing_model" binding:"required,pricing_model"`
	EffectiveFrom        *time.Time `json:"effective_from"`
	EffectiveTo          *time.Time `json:"effective_to"`
}

// UpdatePricingRequest represents pricing update request
type UpdatePricingRequest struct {
	ModelID              int64      `json:"model_id" binding:"omitempty,min=1"`
	InputTokenCost       float64    `json:"input_token_cost" binding:"omitempty,min=0"`
	OutputTokenCost      float64    `json:"output_token_cost" binding:"omitempty,min=0"`
	CachedInputTokenCost float64    `json:"cached_input_token_cost" binding:"omitempty,min=0"`
	StorageCost          float64    `json:"storage_cost" binding:"omitempty,min=0"`
	RequestCost          float64    `json:"request_cost" binding:"omitempty,min=0"`
	Currency             string     `json:"currency" binding:"omitempty,min=3,max=3"`
	PricingModel         string     `json:"pricing_model" binding:"omitempty,pricing_model"`
	EffectiveFrom        *time.Time `json:"effective_from"`
	EffectiveTo          *time.Time `json:"effective_to"`
}

// CreateLimitRequest represents limit creation request
type CreateLimitRequest struct {
	ModelID      int64      `json:"model_id" binding:"required,min=1"`
	LimitType    string     `json:"limit_type" binding:"required,limit_type"`
	LimitValue   int        `json:"limit_value" binding:"required,min=0"`
	CurrentUsage int        `json:"current_usage" binding:"min=0"`
	ResetPeriod  string     `json:"reset_period" binding:"required,reset_period"`
	ResetTime    *time.Time `json:"reset_time"`
	IsHardLimit  bool       `json:"is_hard_limit"`
}

// UpdateLimitRequest represents limit update request
type UpdateLimitRequest struct {
	ModelID      int64      `json:"model_id" binding:"omitempty,min=1"`
	LimitType    string     `json:"limit_type" binding:"omitempty,limit_type"`
	LimitValue   int        `json:"limit_value" binding:"omitempty,min=0"`
	CurrentUsage int        `json:"current_usage" binding:"omitempty,min=0"`
	ResetPeriod  string     `json:"reset_period" binding:"omitempty,reset_period"`
	ResetTime    *time.Time `json:"reset_time"`
	IsHardLimit  *bool      `json:"is_hard_limit"`
}

// CreateConfigExportRequest represents config export creation request
type CreateConfigExportRequest struct {
	ExportType        string  `json:"export_type" binding:"required,export_type"`
	Name              string  `json:"name" binding:"required,min=2,max=100,alphanumspace"`
	Description       string  `json:"description" binding:"max=500"`
	ConfigData        string  `json:"config_data" binding:"required,min=2"`
	TargetModels      *string `json:"target_models" binding:"max=1000"`
	TargetProviders   *string `json:"target_providers" binding:"max=1000"`
	Filters           *string `json:"filters" binding:"max=1000"`
	IsVerified        bool    `json:"is_verified"`
	VerificationNotes *string `json:"verification_notes" binding:"max=1000"`
	CreatedBy         *string `json:"created_by" binding:"max=100"`
}

// UpdateConfigExportRequest represents config export update request
type UpdateConfigExportRequest struct {
	ExportType        string  `json:"export_type" binding:"omitempty,export_type"`
	Name              string  `json:"name" binding:"omitempty,min=2,max=100,alphanumspace"`
	Description       string  `json:"description" binding:"omitempty,max=500"`
	ConfigData        string  `json:"config_data" binding:"omitempty,min=2"`
	TargetModels      *string `json:"target_models" binding:"omitempty,max=1000"`
	TargetProviders   *string `json:"target_providers" binding:"omitempty,max=1000"`
	Filters           *string `json:"filters" binding:"omitempty,max=1000"`
	IsVerified        *bool   `json:"is_verified"`
	VerificationNotes *string `json:"verification_notes" binding:"omitempty,max=1000"`
	CreatedBy         *string `json:"created_by" binding:"omitempty,max=100"`
}

// Helper function to validate request
func ValidateRequest(data interface{}) error {
	return validate.Struct(data)
}

// Helper function to get validation errors in user-friendly format
func GetValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			field := fieldError.Field()
			tag := fieldError.Tag()
			param := fieldError.Param()

			switch tag {
			case "required":
				errors[field] = "This field is required"
			case "min":
				errors[field] = fmt.Sprintf("Value must be at least %s", param)
			case "max":
				errors[field] = fmt.Sprintf("Value must be at most %s", param)
			case "len":
				errors[field] = fmt.Sprintf("Value must be exactly %s characters", param)
			case "email":
				errors[field] = "Invalid email format"
			case "url":
				errors[field] = "Invalid URL format"
			case "alphanumspace":
				errors[field] = "Only alphanumeric characters, spaces, hyphens, underscores, and periods are allowed"
			case "oneof":
				errors[field] = "Invalid value. Must be one of the allowed values"
			case "gt":
				errors[field] = fmt.Sprintf("Value must be greater than %s", param)
			case "gte":
				errors[field] = fmt.Sprintf("Value must be greater than or equal to %s", param)
			case "lt":
				errors[field] = fmt.Sprintf("Value must be less than %s", param)
			case "lte":
				errors[field] = fmt.Sprintf("Value must be less than or equal to %s", param)
			case "eq":
				errors[field] = fmt.Sprintf("Value must be equal to %s", param)
			case "ne":
				errors[field] = fmt.Sprintf("Value must not be equal to %s", param)
			default:
				errors[field] = "Invalid value"
			}
		}
	}

	return errors
}

// ValidateAndSanitizeString validates and sanitizes a string input
func ValidateAndSanitizeString(input string, minLen, maxLen int, allowHTML bool) (string, error) {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Check length
	if len(input) < minLen {
		return "", fmt.Errorf("input must be at least %d characters", minLen)
	}
	if len(input) > maxLen {
		return "", fmt.Errorf("input must be at most %d characters", maxLen)
	}

	// Sanitize based on HTML allowance
	if allowHTML {
		input = SanitizeHTML(input)
	} else {
		input = SanitizeInput(input)
	}

	return input, nil
}

// ValidateAndSanitizeEmail validates and sanitizes an email address
func ValidateAndSanitizeEmail(email string) (string, error) {
	email = strings.TrimSpace(email)

	// Use validator for email validation
	err := validate.Var(email, "required,email")
	if err != nil {
		return "", fmt.Errorf("invalid email format")
	}

	// Sanitize email
	sanitized, valid := SanitizeEmail(email)
	if !valid {
		return "", fmt.Errorf("invalid email address")
	}

	return sanitized, nil
}

// ValidateAndSanitizeURL validates and sanitizes a URL
func ValidateAndSanitizeURL(url string) (string, error) {
	url = strings.TrimSpace(url)

	// Use validator for URL validation
	err := validate.Var(url, "url")
	if err != nil {
		return "", fmt.Errorf("invalid URL format")
	}

	// Sanitize URL
	sanitized, valid := SanitizeURL(url)
	if !valid {
		return "", fmt.Errorf("invalid URL")
	}

	return sanitized, nil
}

// ValidateAndSanitizeInteger validates and sanitizes an integer
func ValidateAndSanitizeInteger(input string, min, max int64) (int64, error) {
	// Sanitize and parse integer
	value, valid := SanitizeInteger(input)
	if !valid {
		return 0, fmt.Errorf("invalid integer value")
	}

	// Validate range
	if value < min {
		return 0, fmt.Errorf("value must be at least %d", min)
	}
	if value > max {
		return 0, fmt.Errorf("value must be at most %d", max)
	}

	return value, nil
}

// ValidateAndSanitizeFloat validates and sanitizes a float
func ValidateAndSanitizeFloat(input string, min, max float64) (float64, error) {
	// Sanitize and parse float
	value, valid := SanitizeFloat(input)
	if !valid {
		return 0, fmt.Errorf("invalid float value")
	}

	// Validate range
	if value < min {
		return 0, fmt.Errorf("value must be at least %f", min)
	}
	if value > max {
		return 0, fmt.Errorf("value must be at most %f", max)
	}

	return value, nil
}

// ValidateAndSanitizeBool validates and sanitizes a boolean
func ValidateAndSanitizeBool(input string) (bool, error) {
	value, valid := SanitizeBool(input)
	if !valid {
		return false, fmt.Errorf("invalid boolean value")
	}

	return value, nil
}

// ValidateDateRange validates that start date is before end date
func ValidateDateRange(startDate, endDate time.Time) error {
	if startDate.After(endDate) {
		return fmt.Errorf("start date must be before end date")
	}

	// Optional: Add maximum date range validation
	maxRange := 365 * 24 * time.Hour // 1 year
	if endDate.Sub(startDate) > maxRange {
		return fmt.Errorf("date range cannot exceed 1 year")
	}

	return nil
}

// ValidateStringSlice validates a slice of strings
func ValidateStringSlice(slice []string, minItems, maxItems int, minLen, maxLen int) error {
	if len(slice) < minItems {
		return fmt.Errorf("must have at least %d items", minItems)
	}
	if len(slice) > maxItems {
		return fmt.Errorf("cannot have more than %d items", maxItems)
	}

	for i, item := range slice {
		item = strings.TrimSpace(item)
		if len(item) < minLen {
			return fmt.Errorf("item %d must be at least %d characters", i+1, minLen)
		}
		if len(item) > maxLen {
			return fmt.Errorf("item %d must be at most %d characters", i+1, maxLen)
		}
	}

	return nil
}

// ValidateIntegerSlice validates a slice of integers
func ValidateIntegerSlice(slice []int64, minItems, maxItems int, minVal, maxVal int64) error {
	if len(slice) < minItems {
		return fmt.Errorf("must have at least %d items", minItems)
	}
	if len(slice) > maxItems {
		return fmt.Errorf("cannot have more than %d items", maxItems)
	}

	for i, item := range slice {
		if item < minVal {
			return fmt.Errorf("item %d must be at least %d", i+1, minVal)
		}
		if item > maxVal {
			return fmt.Errorf("item %d must be at most %d", i+1, maxVal)
		}
	}

	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Check for at least one uppercase letter
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	// Check for at least one lowercase letter
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	// Check for at least one digit
	if !regexp.MustCompile(`\d`).MatchString(password) {
		return fmt.Errorf("password must contain at least one digit")
	}

	// Check for at least one special character
	if !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// ValidateUsername validates username format
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)

	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if len(username) > 50 {
		return fmt.Errorf("username must be at most 50 characters")
	}

	// Only allow alphanumeric characters, underscores, and hyphens
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(username) {
		return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
	}

	return nil
}

// ValidatePhoneNumber validates phone number format
func ValidatePhoneNumber(phone string) error {
	phone = strings.TrimSpace(phone)

	// Remove all non-digit characters
	re := regexp.MustCompile(`\D`)
	digits := re.ReplaceAllString(phone, "")

	// Validate length (10 digits for US numbers)
	if len(digits) < 10 || len(digits) > 15 {
		return fmt.Errorf("phone number must be between 10 and 15 digits")
	}

	return nil
}

// ValidateJSON validates JSON string
func ValidateJSON(jsonStr string) error {
	// Check for balanced braces/brackets
	braceCount := 0
	bracketCount := 0

	for _, char := range jsonStr {
		switch char {
		case '{':
			braceCount++
		case '}':
			braceCount--
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		}

		// If counts go negative, JSON is malformed
		if braceCount < 0 || bracketCount < 0 {
			return fmt.Errorf("malformed JSON: unbalanced braces or brackets")
		}
	}

	// Check for balanced braces/brackets
	if braceCount != 0 || bracketCount != 0 {
		return fmt.Errorf("malformed JSON: unbalanced braces or brackets")
	}

	return nil
}

// ValidateCronExpression validates cron expression format
func ValidateCronExpression(cron string) error {
	if cron == "" {
		return nil // Empty cron is allowed for non-cron schedules
	}

	parts := strings.Fields(cron)
	if len(parts) != 5 {
		return fmt.Errorf("cron expression must have exactly 5 fields")
	}

	// Basic validation - could be enhanced with more detailed checks
	for i, part := range parts {
		if part == "" {
			return fmt.Errorf("cron field %d cannot be empty", i+1)
		}
	}

	return nil
}

// customValidator implements Gin's StructValidator interface
type customValidator struct {
	validator *validator.Validate
}

// ValidateStruct validates a struct using our custom validator
func (cv *customValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}
	return cv.validator.Struct(obj)
}

// Engine returns the underlying validator engine
func (cv *customValidator) Engine() any {
	return cv.validator
}

// SetupGinValidator sets up Gin to use our custom validator
func SetupGinValidator() {
	cv := &customValidator{validator: validate}
	binding.Validator = cv
}
