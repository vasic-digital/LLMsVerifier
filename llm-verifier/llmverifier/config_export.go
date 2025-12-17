package llmverifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"llm-verifier/config"
	"llm-verifier/database"
)

// AIConfig represents the base structure for AI CLI agent configurations
type AIConfig struct {
	Version     string      `json:"version"`
	CreatedAt   time.Time   `json:"created_at"`
	GeneratedBy string      `json:"generated_by"`
	Models      []AIModel   `json:"models"`
	Preferences Preferences `json:"preferences"`
	Metadata    Metadata    `json:"metadata"`
}

// AIModel represents a model in AI CLI configuration
type AIModel struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Provider     string         `json:"provider"`
	Endpoint     string         `json:"endpoint"`
	APIKey       string         `json:"api_key,omitempty"`
	Capabilities []string       `json:"capabilities"`
	Score        float64        `json:"score"`
	Category     string         `json:"category"`
	Tags         []string       `json:"tags"`
	Description  string         `json:"description,omitempty"`
	Settings     map[string]any `json:"settings,omitempty"`
}

// Preferences contains user preferences for AI tools
type Preferences struct {
	PrimaryModel    string   `json:"primary_model"`
	FallbackModels  []string `json:"fallback_models"`
	MaxTokens       int      `json:"max_tokens,omitempty"`
	Temperature     float64  `json:"temperature,omitempty"`
	AutoSave        bool     `json:"auto_save"`
	StreamResponses bool     `json:"stream_responses"`
	Language        string   `json:"language,omitempty"`
}

// Metadata contains additional information about the configuration
type Metadata struct {
	TotalModels        int       `json:"total_models"`
	AverageScore       float64   `json:"average_score"`
	ExportCriteria     string    `json:"export_criteria"`
	LLMVerifierVersion string    `json:"llm_verifier_version"`
	LastUpdated        time.Time `json:"last_updated"`
}

// ExportOptions controls how models are selected for export
type ExportOptions struct {
	Format        string             `json:"format"`
	Top           int                `json:"top,omitempty"`
	MinScore      float64            `json:"min_score,omitempty"`
	MaxModels     int                `json:"max_models,omitempty"`
	Categories    []string           `json:"categories,omitempty"`
	Providers     []string           `json:"providers,omitempty"`
	Models        []string           `json:"models,omitempty"`
	ScoreWeight   map[string]float64 `json:"score_weight,omitempty"`
	IncludeAPIKey bool               `json:"include_api_key"`
}

// ExportConfig exports configuration to various formats (enhanced)
func ExportConfig(db *database.Database, cfg *config.Config, format, outputPath string) error {
	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(cfg, "", "  ")
	case "yaml", "yml":
		data, err = yaml.Marshal(cfg)
	case "opencode":
		return ExportAIConfig(db, cfg, "opencode", outputPath, &ExportOptions{})
	case "crush":
		return ExportAIConfig(db, cfg, "crush", outputPath, &ExportOptions{})
	case "claude-code":
		return ExportAIConfig(db, cfg, "claude-code", outputPath, &ExportOptions{})
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ExportAIConfig exports AI CLI agent configurations
func ExportAIConfig(db *database.Database, cfg *config.Config, aiFormat, outputPath string, options *ExportOptions) error {
	// Fetch real verification results from database
	results, err := fetchVerificationResults(db, options)
	if err != nil {
		return fmt.Errorf("failed to fetch verification results: %w", err)
	}

	// Filter models based on options
	filteredModels := filterModels(results, options)

	// Create AI-specific configuration
	var aiConfig *AIConfig

	switch strings.ToLower(aiFormat) {
	case "opencode":
		aiConfig, err = createOpenCodeConfig(filteredModels, options)
	case "crush":
		aiConfig, err = createCrushConfig(filteredModels, options)
	case "claude-code":
		aiConfig, err = createClaudeCode(filteredModels, options)
	default:
		return fmt.Errorf("unsupported AI format: %s", aiFormat)
	}

	if err != nil {
		return fmt.Errorf("failed to create %s config: %w", aiFormat, err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(aiConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal AI config: %w", err)
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write AI config file: %w", err)
	}

	return nil
}

// createOpenCodeConfig creates configuration for OpenCode
func createOpenCodeConfig(results []VerificationResult, options *ExportOptions) (*AIConfig, error) {
	models := make([]AIModel, 0, len(results))
	var totalScore float64

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		capabilities := extractCapabilities(result)
		category := categorizeModel(result)
		provider := extractProvider(result.ModelInfo.Endpoint)

		model := AIModel{
			ID:           result.ModelInfo.ID,
			Name:         result.ModelInfo.ID,
			Provider:     provider,
			Endpoint:     result.ModelInfo.Endpoint,
			Capabilities: capabilities,
			Score:        result.PerformanceScores.OverallScore,
			Category:     category,
			Tags:         result.ModelInfo.Tags,
			Description:  result.ModelInfo.Description,
			Settings:     createOpenCodeModelSettings(result, provider),
		}

		// Add API key if requested
		if options != nil && options.IncludeAPIKey {
			model.APIKey = "YOUR_API_KEY_HERE" // In real implementation, this would be encrypted/obfuscated
		}

		models = append(models, model)
		totalScore += result.PerformanceScores.OverallScore
	}

	// Sort by score (highest first)
	slices.SortFunc(models, func(a, b AIModel) int {
		if b.Score > a.Score {
			return 1
		}
		if b.Score < a.Score {
			return -1
		}
		return 0
	})

	// Select primary model (highest scoring)
	var primaryModel string
	var fallbackModels []string

	if len(models) > 0 {
		primaryModel = models[0].ID
		for i := 1; i < len(models) && i < 3; i++ {
			fallbackModels = append(fallbackModels, models[i].ID)
		}
	}

	avgScore := float64(0)
	if len(models) > 0 {
		avgScore = totalScore / float64(len(models))
	}

	return &AIConfig{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		GeneratedBy: "LLM Verifier",
		Models:      models,
		Preferences: Preferences{
			PrimaryModel:    primaryModel,
			FallbackModels:  fallbackModels,
			MaxTokens:       4096,
			Temperature:     0.7,
			AutoSave:        true,
			StreamResponses: true,
			Language:        "english",
		},
		Metadata: Metadata{
			TotalModels:        len(models),
			AverageScore:       avgScore,
			ExportCriteria:     getExportCriteriaDescription(options),
			LLMVerifierVersion: "1.0.0",
			LastUpdated:        time.Now(),
		},
	}, nil
}

// createOpenCodeModelSettings creates provider-specific settings for OpenCode
func createOpenCodeModelSettings(result VerificationResult, provider string) map[string]any {
	baseSettings := map[string]any{
		"max_tokens":         result.ModelInfo.MaxOutputTokens,
		"context_window":     result.ModelInfo.ContextWindow.TotalMaxTokens,
		"supports_vision":    result.ModelInfo.SupportsVision,
		"supports_audio":     result.ModelInfo.SupportsAudio,
		"supports_video":     result.ModelInfo.SupportsVideo,
		"supports_reasoning": result.ModelInfo.SupportsReasoning,
		"temperature":        0.7,
		"top_p":              0.9,
		"frequency_penalty":  0.0,
		"presence_penalty":   0.0,
	}

	// Provider-specific optimizations
	switch provider {
	case "OpenAI":
		baseSettings["temperature"] = 0.7
		baseSettings["model_family"] = "gpt"
		baseSettings["streaming_supported"] = true
	case "Anthropic":
		baseSettings["temperature"] = 0.8
		baseSettings["model_family"] = "claude"
		baseSettings["streaming_supported"] = true
		baseSettings["max_tokens"] = min(4096, result.ModelInfo.MaxOutputTokens)
	case "DeepSeek":
		baseSettings["temperature"] = 0.6
		baseSettings["model_family"] = "deepseek"
		baseSettings["streaming_supported"] = true
	case "Google":
		baseSettings["temperature"] = 0.9
		baseSettings["model_family"] = "gemini"
		baseSettings["streaming_supported"] = true
	}

	// Add reasoning-specific settings
	if result.ModelInfo.SupportsReasoning {
		baseSettings["reasoning_enabled"] = true
		baseSettings["reasoning_budget"] = 2048
		baseSettings["reasoning_temperature"] = 0.3
	}

	return baseSettings
}

// createCrushConfig creates configuration for Crush
func createCrushConfig(results []VerificationResult, options *ExportOptions) (*AIConfig, error) {
	models := make([]AIModel, 0, len(results))
	var totalScore float64

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		// Crush focuses on coding capabilities - filter for high coding scores
		if result.PerformanceScores.CodeCapability < 75 {
			continue // Skip models with poor coding capability
		}

		capabilities := extractCapabilities(result)
		category := categorizeModel(result)
		provider := extractProvider(result.ModelInfo.Endpoint)

		model := AIModel{
			ID:           result.ModelInfo.ID,
			Name:         result.ModelInfo.ID,
			Provider:     provider,
			Endpoint:     result.ModelInfo.Endpoint,
			Capabilities: capabilities,
			Score:        result.PerformanceScores.OverallScore,
			Category:     category,
			Tags:         result.ModelInfo.Tags,
			Description:  result.ModelInfo.Description,
			Settings:     createCrushModelSettings(result, provider),
		}

		if options != nil && options.IncludeAPIKey {
			model.APIKey = "YOUR_API_KEY_HERE"
		}

		models = append(models, model)
		totalScore += result.PerformanceScores.OverallScore
	}

	// Sort by code capability score (highest first)
	slices.SortFunc(models, func(a, b AIModel) int {
		aCodeScore := a.Score // Assuming we can access code score, using overall for now
		bCodeScore := b.Score
		if bCodeScore > aCodeScore {
			return 1
		}
		if bCodeScore < aCodeScore {
			return -1
		}
		return 0
	})

	var primaryModel string
	var fallbackModels []string

	if len(models) > 0 {
		primaryModel = models[0].ID
		for i := 1; i < len(models) && i < 2; i++ {
			fallbackModels = append(fallbackModels, models[i].ID)
		}
	}

	avgScore := float64(0)
	if len(models) > 0 {
		avgScore = totalScore / float64(len(models))
	}

	return &AIConfig{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		GeneratedBy: "LLM Verifier",
		Models:      models,
		Preferences: Preferences{
			PrimaryModel:    primaryModel,
			FallbackModels:  fallbackModels,
			MaxTokens:       2048, // Crush prefers shorter responses for coding
			Temperature:     0.3,  // Lower temperature for deterministic code
			AutoSave:        true,
			StreamResponses: true,
			Language:        "english",
		},
		Metadata: Metadata{
			TotalModels:        len(models),
			AverageScore:       avgScore,
			ExportCriteria:     "High coding capability models (code score >= 75)",
			LLMVerifierVersion: "1.0.0",
			LastUpdated:        time.Now(),
		},
	}, nil
}

// createCrushModelSettings creates coding-focused settings for Crush
func createCrushModelSettings(result VerificationResult, provider string) map[string]any {
	baseSettings := map[string]any{
		"max_tokens":        result.ModelInfo.MaxOutputTokens,
		"context_window":    result.ModelInfo.ContextWindow.TotalMaxTokens,
		"temperature":       0.3, // Crush prefers lower temperature for coding
		"top_p":             0.95,
		"frequency_penalty": 0.1,
		"presence_penalty":  0.1,
		"code_focus":        true,
		"debug_mode":        false,
		"auto_debug":        true,
		"test_generation":   true,
		"linting_enabled":   true,
		"format_on_save":    true,
	}

	// Provider-specific coding optimizations
	switch provider {
	case "OpenAI":
		baseSettings["model_family"] = "gpt"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	case "Anthropic":
		baseSettings["model_family"] = "claude"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
		baseSettings["max_tokens"] = min(2048, result.ModelInfo.MaxOutputTokens)
	case "DeepSeek":
		baseSettings["model_family"] = "deepseek"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
		baseSettings["temperature"] = 0.2 // Even lower for DeepSeek coding
	case "Google":
		baseSettings["model_family"] = "gemini"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	}

	// Add advanced coding features
	if result.FeatureDetection.CodeGeneration && result.FeatureDetection.CodeReview {
		baseSettings["advanced_coding"] = true
		baseSettings["code_review_enabled"] = true
		baseSettings["refactoring_suggestions"] = true
	}

	return baseSettings
}

// createClaudeCode creates configuration for Claude Code
func createClaudeCode(results []VerificationResult, options *ExportOptions) (*AIConfig, error) {
	models := make([]AIModel, 0, len(results))
	var totalScore float64

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		capabilities := extractCapabilities(result)
		category := categorizeModel(result)
		provider := extractProvider(result.ModelInfo.Endpoint)

		model := AIModel{
			ID:           result.ModelInfo.ID,
			Name:         result.ModelInfo.ID,
			Provider:     provider,
			Endpoint:     result.ModelInfo.Endpoint,
			Capabilities: capabilities,
			Score:        result.PerformanceScores.OverallScore,
			Category:     category,
			Tags:         result.ModelInfo.Tags,
			Description:  result.ModelInfo.Description,
			Settings:     createClaudeCodeModelSettings(result, provider),
		}

		if options != nil && options.IncludeAPIKey {
			model.APIKey = "YOUR_API_KEY_HERE"
		}

		models = append(models, model)
		totalScore += result.PerformanceScores.OverallScore
	}

	// Sort by overall score with preference for reasoning capability
	slices.SortFunc(models, func(a, b AIModel) int {
		// Prefer models with reasoning capability
		aReasoning := slices.Contains(a.Capabilities, "reasoning")
		bReasoning := slices.Contains(b.Capabilities, "reasoning")

		if aReasoning && !bReasoning {
			return -1
		}
		if !aReasoning && bReasoning {
			return 1
		}

		return int(b.Score*1000 - a.Score*1000)
	})

	var primaryModel string
	var fallbackModels []string

	if len(models) > 0 {
		primaryModel = models[0].ID
		for i := 1; i < len(models) && i < 3; i++ {
			fallbackModels = append(fallbackModels, models[i].ID)
		}
	}

	avgScore := float64(0)
	if len(models) > 0 {
		avgScore = totalScore / float64(len(models))
	}

	return &AIConfig{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		GeneratedBy: "LLM Verifier",
		Models:      models,
		Preferences: Preferences{
			PrimaryModel:    primaryModel,
			FallbackModels:  fallbackModels,
			MaxTokens:       4096,
			Temperature:     0.5, // Moderate temperature for balanced responses
			AutoSave:        true,
			StreamResponses: true,
			Language:        "english",
		},
		Metadata: Metadata{
			TotalModels:        len(models),
			AverageScore:       avgScore,
			ExportCriteria:     "All models with preference for reasoning capability",
			LLMVerifierVersion: "1.0.0",
			LastUpdated:        time.Now(),
		},
	}, nil
}

// createClaudeCodeModelSettings creates Claude Code specific settings
func createClaudeCodeModelSettings(result VerificationResult, provider string) map[string]any {
	baseSettings := map[string]any{
		"max_tokens":         result.ModelInfo.MaxOutputTokens,
		"context_window":     result.ModelInfo.ContextWindow.TotalMaxTokens,
		"temperature":        0.5, // Claude Code prefers moderate temperature
		"top_p":              0.9,
		"frequency_penalty":  0.0,
		"presence_penalty":   0.0,
		"conversation_style": "professional",
		"code_style":         "clean",
		"explanation_level":  "detailed",
		"auto_format":        true,
		"import_suggestions": true,
		"error_handling":     true,
		"context_awareness":  true,
	}

	// Provider-specific conversation settings
	switch provider {
	case "Anthropic":
		baseSettings["model_family"] = "claude"
		baseSettings["anthropic_style"] = true
		baseSettings["streaming_supported"] = true
		baseSettings["conversation_memory"] = true
	case "OpenAI":
		baseSettings["model_family"] = "gpt"
		baseSettings["openai_style"] = true
		baseSettings["streaming_supported"] = true
		baseSettings["conversation_memory"] = true
	case "DeepSeek":
		baseSettings["model_family"] = "deepseek"
		baseSettings["streaming_supported"] = true
		baseSettings["temperature"] = 0.4 // Slightly lower for DeepSeek
	case "Google":
		baseSettings["model_family"] = "gemini"
		baseSettings["google_style"] = true
		baseSettings["streaming_supported"] = true
	}

	// Add reasoning-specific features
	if result.ModelInfo.SupportsReasoning {
		baseSettings["reasoning_assistance"] = true
		baseSettings["step_by_step_explanation"] = true
		baseSettings["complexity_analysis"] = true
	}

	return baseSettings
}

// filterModels filters verification results based on export options
func filterModels(results []VerificationResult, options *ExportOptions) []VerificationResult {
	if options == nil {
		return results
	}

	filtered := make([]VerificationResult, 0, len(results))

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		// Filter by minimum score
		if options.MinScore > 0 && result.PerformanceScores.OverallScore < options.MinScore {
			continue
		}

		// Filter by categories
		if len(options.Categories) > 0 {
			category := categorizeModel(result)
			if !slices.Contains(options.Categories, category) {
				continue
			}
		}

		// Filter by providers
		if len(options.Providers) > 0 {
			provider := extractProvider(result.ModelInfo.Endpoint)
			if !slices.Contains(options.Providers, provider) {
				continue
			}
		}

		// Filter by specific models
		if len(options.Models) > 0 {
			if !slices.Contains(options.Models, result.ModelInfo.ID) {
				continue
			}
		}

		filtered = append(filtered, result)
	}

	// Limit by max models
	if options.MaxModels > 0 && len(filtered) > options.MaxModels {
		filtered = filtered[:options.MaxModels]
	}

	// Sort by score (highest first)
	slices.SortFunc(filtered, func(a, b VerificationResult) int {
		return int(b.PerformanceScores.OverallScore*1000 - a.PerformanceScores.OverallScore*1000)
	})

	// Limit by top models
	if options.Top > 0 && len(filtered) > options.Top {
		filtered = filtered[:options.Top]
	}

	return filtered
}

// extractCapabilities extracts model capabilities from verification result
func extractCapabilities(result VerificationResult) []string {
	capabilities := make([]string, 0)

	if result.FeatureDetection.ToolUse {
		capabilities = append(capabilities, "tool_use")
	}
	if result.FeatureDetection.FunctionCalling {
		capabilities = append(capabilities, "function_calling")
	}
	if result.FeatureDetection.CodeGeneration {
		capabilities = append(capabilities, "code_generation")
	}
	if result.FeatureDetection.CodeCompletion {
		capabilities = append(capabilities, "code_completion")
	}
	if result.FeatureDetection.CodeReview {
		capabilities = append(capabilities, "code_review")
	}
	if result.FeatureDetection.CodeExplanation {
		capabilities = append(capabilities, "code_explanation")
	}
	if result.FeatureDetection.Embeddings {
		capabilities = append(capabilities, "embeddings")
	}
	if result.FeatureDetection.Reranking {
		capabilities = append(capabilities, "reranking")
	}
	if result.FeatureDetection.ImageGeneration {
		capabilities = append(capabilities, "image_generation")
	}
	if result.FeatureDetection.AudioGeneration {
		capabilities = append(capabilities, "audio_generation")
	}
	if result.FeatureDetection.VideoGeneration {
		capabilities = append(capabilities, "video_generation")
	}
	if result.FeatureDetection.MCPs {
		capabilities = append(capabilities, "mcps")
	}
	if result.FeatureDetection.LSPs {
		capabilities = append(capabilities, "lsps")
	}
	if result.FeatureDetection.Multimodal {
		capabilities = append(capabilities, "multimodal")
	}
	if result.FeatureDetection.Streaming {
		capabilities = append(capabilities, "streaming")
	}
	if result.FeatureDetection.JSONMode {
		capabilities = append(capabilities, "json_mode")
	}
	if result.FeatureDetection.StructuredOutput {
		capabilities = append(capabilities, "structured_output")
	}
	if result.FeatureDetection.Reasoning {
		capabilities = append(capabilities, "reasoning")
	}
	if result.FeatureDetection.ParallelToolUse {
		capabilities = append(capabilities, "parallel_tool_use")
	}

	return capabilities
}

// categorizeModel categorizes model based on capabilities
func categorizeModel(result VerificationResult) string {
	// Check for coding-focused models (highest priority)
	if result.CodeCapabilities.CodeGeneration ||
		result.CodeCapabilities.CodeReview ||
		result.CodeCapabilities.CodeExplanation ||
		result.FeatureDetection.CodeGeneration ||
		result.FeatureDetection.CodeReview ||
		result.FeatureDetection.CodeExplanation {
		return "coding"
	}

	// Check for multimodal models
	if result.FeatureDetection.Multimodal ||
		(result.FeatureDetection.ImageGeneration && result.FeatureDetection.CodeGeneration) {
		return "multimodal"
	}

	// Check for reasoning models
	if result.FeatureDetection.Reasoning && result.PerformanceScores.OverallScore >= 80 {
		return "reasoning"
	}

	// Check for chat models
	if result.FeatureDetection.ToolUse && result.FeatureDetection.FunctionCalling {
		return "chat"
	}

	// Check for generative models
	if result.GenerativeCapabilities.CreativeWriting ||
		result.GenerativeCapabilities.ContentGeneration {
		return "generative"
	}

	// Check for specialized models
	if result.FeatureDetection.Embeddings || result.FeatureDetection.Reranking {
		return "specialized"
	}

	return "general"
}

// extractProvider extracts provider name from endpoint
func extractProvider(endpoint string) string {
	if strings.Contains(endpoint, "openai.com") {
		return "OpenAI"
	}
	if strings.Contains(endpoint, "anthropic.com") {
		return "Anthropic"
	}
	if strings.Contains(endpoint, "deepseek.com") {
		return "DeepSeek"
	}
	if strings.Contains(endpoint, "google.com") {
		return "Google"
	}
	if strings.Contains(endpoint, "azure.com") {
		return "Azure"
	}
	if strings.Contains(endpoint, "aws") || strings.Contains(endpoint, "bedrock") {
		return "AWS"
	}

	return "Unknown"
}

// getExportCriteriaDescription generates description of export criteria
func getExportCriteriaDescription(options *ExportOptions) string {
	if options == nil {
		return "All models"
	}

	criteria := make([]string, 0)

	if options.MinScore > 0 {
		criteria = append(criteria, fmt.Sprintf("min score %.1f", options.MinScore))
	}

	if options.Top > 0 {
		criteria = append(criteria, fmt.Sprintf("top %d models", options.Top))
	}

	if len(options.Categories) > 0 {
		criteria = append(criteria, fmt.Sprintf("categories: %s", strings.Join(options.Categories, ", ")))
	}

	if len(options.Providers) > 0 {
		criteria = append(criteria, fmt.Sprintf("providers: %s", strings.Join(options.Providers, ", ")))
	}

	if len(options.Models) > 0 {
		criteria = append(criteria, fmt.Sprintf("specific models: %s", strings.Join(options.Models, ", ")))
	}

	if len(criteria) == 0 {
		return "All models"
	}

	return strings.Join(criteria, "; ")
}

// ExportBulkConfig exports configurations for multiple AI CLI agents in bulk
func ExportBulkConfig(db *database.Database, cfg *config.Config, outputPath string, options *ExportOptions) error {
	if options == nil {
		options = &ExportOptions{}
	}

	// Create all supported formats
	formats := []string{"opencode", "crush", "claude-code"}
	exportResults := make([]ExportResult, 0, len(formats))

	for _, format := range formats {
		// Generate filename for this format
		safeFormat := strings.ReplaceAll(format, "-", "_")
		filename := fmt.Sprintf("export_%s.json", safeFormat)
		fullPath := filepath.Join(outputPath, filename)

		// Export configuration for this format
		startTime := time.Now()
		err := ExportAIConfig(db, cfg, format, fullPath, options)
		duration := time.Since(startTime)

		result := ExportResult{
			Format:   format,
			Filename: filename,
			Path:     fullPath,
			Success:  err == nil,
			Duration: duration,
			Error:    "",
		}

		if err != nil {
			result.Error = err.Error()
			return fmt.Errorf("failed to export %s format: %w", format, err)
		}

		exportResults = append(exportResults, result)
	}

	// Create summary and manifest files
	if err := createExportSummary(outputPath, formats, options); err != nil {
		return fmt.Errorf("failed to create export summary: %w", err)
	}

	if err := createExportManifest(outputPath, exportResults, options); err != nil {
		return fmt.Errorf("failed to create export manifest: %w", err)
	}

	return nil
}

// ExportResult represents the result of a single export operation
type ExportResult struct {
	Format   string        `json:"format"`
	Filename string        `json:"filename"`
	Path     string        `json:"path"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// createExportSummary creates a summary of all exported configurations
func createExportSummary(outputPath string, formats []string, options *ExportOptions) error {
	summary := struct {
		Version     string         `json:"version"`
		ExportedAt  time.Time      `json:"exported_at"`
		GeneratedBy string         `json:"generated_by"`
		Formats     []string       `json:"formats"`
		Options     *ExportOptions `json:"options"`
		Description string         `json:"description"`
	}{
		Version:     "1.0",
		ExportedAt:  time.Now(),
		GeneratedBy: "LLM Verifier",
		Formats:     formats,
		Options:     options,
		Description: "Bulk export of AI CLI agent configurations",
	}

	// Marshal summary
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	// Write summary file
	summaryPath := filepath.Join(outputPath, "export_summary.json")
	if err := os.WriteFile(summaryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}

// createExportManifest creates a detailed manifest of all exported files
func createExportManifest(outputPath string, results []ExportResult, options *ExportOptions) error {
	manifest := struct {
		Version       string         `json:"version"`
		CreatedAt     time.Time      `json:"created_at"`
		GeneratedBy   string         `json:"generated_by"`
		OutputPath    string         `json:"output_path"`
		Options       *ExportOptions `json:"options"`
		TotalExports  int            `json:"total_exports"`
		Successful    int            `json:"successful"`
		Failed        int            `json:"failed"`
		TotalDuration time.Duration  `json:"total_duration"`
		Results       []ExportResult `json:"results"`
	}{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		GeneratedBy: "LLM Verifier",
		OutputPath:  outputPath,
		Options:     options,
		Results:     results,
	}

	// Calculate statistics
	totalDuration := time.Duration(0)
	successful := 0
	for _, result := range results {
		totalDuration += result.Duration
		if result.Success {
			successful++
		}
	}

	manifest.TotalExports = len(results)
	manifest.Successful = successful
	manifest.Failed = len(results) - successful
	manifest.TotalDuration = totalDuration

	// Marshal manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write manifest file
	manifestPath := filepath.Join(outputPath, "export_manifest.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// ValidateExportedConfig validates an exported AI CLI configuration
func ValidateExportedConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read exported config: %w", err)
	}

	var config AIConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse exported config: %w", err)
	}

	// Validate basic structure
	if err := validateAIConfigStructure(&config); err != nil {
		return fmt.Errorf("configuration structure validation failed: %w", err)
	}

	// Determine format from filename and validate format-specific requirements
	if err := validateFormatSpecific(configPath, &config); err != nil {
		return fmt.Errorf("format-specific validation failed: %w", err)
	}

	// Validate model configurations
	if err := validateModelConfigurations(&config); err != nil {
		return fmt.Errorf("model configuration validation failed: %w", err)
	}

	// Validate preferences
	if err := validatePreferences(&config); err != nil {
		return fmt.Errorf("preferences validation failed: %w", err)
	}

	return nil
}

// validateAIConfigStructure validates the basic AI config structure
func validateAIConfigStructure(config *AIConfig) error {
	if config.Version == "" {
		return fmt.Errorf("missing version field")
	}

	if len(config.Models) == 0 {
		return fmt.Errorf("models array cannot be empty")
	}

	if config.Preferences.PrimaryModel == "" {
		return fmt.Errorf("missing primary model in preferences")
	}

	return nil
}

// validateFormatSpecific validates format-specific requirements
func validateFormatSpecific(configPath string, config *AIConfig) error {
	filename := filepath.Base(configPath)

	switch {
	case strings.Contains(filename, "opencode"):
		return validateOpenCodeConfig(config)
	case strings.Contains(filename, "crush"):
		return validateCrushConfig(config)
	case strings.Contains(filename, "claude_code"):
		return validateClaudeCodeConfig(config)
	default:
		return fmt.Errorf("unknown configuration format")
	}
}

// validateOpenCodeConfig validates OpenCode-specific requirements
func validateOpenCodeConfig(config *AIConfig) error {
	// OpenCode requires at least one model with coding capabilities
	hasCodingModel := false
	for _, model := range config.Models {
		if slices.Contains(model.Capabilities, "code_generation") ||
			slices.Contains(model.Capabilities, "code_review") {
			hasCodingModel = true
			break
		}
	}

	if !hasCodingModel {
		return fmt.Errorf("OpenCode config must include at least one model with coding capabilities")
	}

	// Validate OpenCode-specific settings
	for i, model := range config.Models {
		if model.Settings == nil {
			continue
		}

		// Check for required OpenCode settings
		requiredSettings := []string{"max_tokens", "context_window", "temperature"}
		for _, setting := range requiredSettings {
			if _, ok := model.Settings[setting]; !ok {
				return fmt.Errorf("model %d missing required OpenCode setting: %s", i, setting)
			}
		}
	}

	return nil
}

// validateCrushConfig validates Crush-specific requirements
func validateCrushConfig(config *AIConfig) error {
	// Crush focuses on coding - all models should have coding capabilities
	for i, model := range config.Models {
		hasCoding := slices.Contains(model.Capabilities, "code_generation") ||
			slices.Contains(model.Capabilities, "code_review") ||
			slices.Contains(model.Capabilities, "code_explanation")

		if !hasCoding {
			return fmt.Errorf("model %d in Crush config must have coding capabilities", i)
		}

		// Check for Crush-specific settings
		if model.Settings != nil {
			if temp, ok := model.Settings["temperature"].(float64); ok && temp > 0.5 {
				return fmt.Errorf("model %d in Crush config has temperature too high (should be ≤ 0.5)", i)
			}
		}
	}

	return nil
}

// validateClaudeCodeConfig validates Claude Code-specific requirements
func validateClaudeCodeConfig(config *AIConfig) error {
	// Claude Code prefers models with reasoning capabilities
	hasReasoningModel := false
	for _, model := range config.Models {
		if slices.Contains(model.Capabilities, "reasoning") {
			hasReasoningModel = true
			break
		}
	}

	if !hasReasoningModel {
		return fmt.Errorf("Claude Code config should include at least one model with reasoning capabilities")
	}

	return nil
}

// validateModelConfigurations validates individual model configurations
func validateModelConfigurations(config *AIConfig) error {
	for i, model := range config.Models {
		// Validate required fields
		if model.ID == "" {
			return fmt.Errorf("model %d missing ID", i)
		}
		if model.Name == "" {
			return fmt.Errorf("model %d missing name", i)
		}
		if model.Provider == "" {
			return fmt.Errorf("model %d missing provider", i)
		}
		if model.Endpoint == "" {
			return fmt.Errorf("model %d missing endpoint", i)
		}

		// Validate score range
		if model.Score < 0 || model.Score > 100 {
			return fmt.Errorf("model %d has invalid score: %.2f (must be 0-100)", i, model.Score)
		}

		// Validate endpoint format (basic check)
		if !strings.HasPrefix(model.Endpoint, "http") {
			return fmt.Errorf("model %d has invalid endpoint format: %s", i, model.Endpoint)
		}

		// Validate capabilities
		validCapabilities := []string{
			"tool_use", "function_calling", "code_generation", "code_completion",
			"code_review", "code_explanation", "embeddings", "reranking",
			"image_generation", "audio_generation", "video_generation",
			"mcps", "lsps", "multimodal", "streaming", "json_mode",
			"structured_output", "reasoning", "parallel_tool_use",
		}

		for _, cap := range model.Capabilities {
			if !slices.Contains(validCapabilities, cap) {
				return fmt.Errorf("model %d has invalid capability: %s", i, cap)
			}
		}

		// Validate settings
		if model.Settings != nil {
			if maxTokens, ok := model.Settings["max_tokens"].(float64); ok && maxTokens <= 0 {
				return fmt.Errorf("model %d has invalid max_tokens setting", i)
			}
			if temp, ok := model.Settings["temperature"].(float64); ok && (temp < 0 || temp > 2) {
				return fmt.Errorf("model %d has invalid temperature setting (must be 0-2)", i)
			}
		}
	}

	return nil
}

// validatePreferences validates the preferences section
func validatePreferences(config *AIConfig) error {
	prefs := &config.Preferences

	// Validate primary model exists
	primaryExists := false
	for _, model := range config.Models {
		if model.ID == prefs.PrimaryModel {
			primaryExists = true
			break
		}
	}
	if !primaryExists {
		return fmt.Errorf("primary model '%s' not found in models list", prefs.PrimaryModel)
	}

	// Validate fallback models exist
	for _, fallback := range prefs.FallbackModels {
		exists := false
		for _, model := range config.Models {
			if model.ID == fallback {
				exists = true
				break
			}
		}
		if !exists {
			return fmt.Errorf("fallback model '%s' not found in models list", fallback)
		}
	}

	// Validate temperature range
	if prefs.Temperature < 0 || prefs.Temperature > 2 {
		return fmt.Errorf("invalid temperature in preferences (must be 0-2)")
	}

	// Validate max tokens
	if prefs.MaxTokens < 0 {
		return fmt.Errorf("invalid max_tokens in preferences (must be >= 0)")
	}

	return nil
}

// fetchVerificationResults fetches real verification results from database
func fetchVerificationResults(db *database.Database, options *ExportOptions) ([]VerificationResult, error) {
	filters := make(map[string]interface{})

	// Apply filters based on options
	if options != nil {
		if options.MinScore > 0 {
			filters["min_score"] = options.MinScore
		}
		// Add limit to prevent fetching too many results
		if options.MaxModels > 0 {
			filters["limit"] = options.MaxModels * 2 // Get more to allow for filtering
		} else {
			filters["limit"] = 50 // Default reasonable limit
		}
	}

	dbResults, err := db.ListVerificationResults(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch verification results from database: %w", err)
	}

	// Convert database results to VerificationResult format
	results := make([]VerificationResult, 0, len(dbResults))
	for _, dbResult := range dbResults {
		if dbResult.Status != "completed" || dbResult.ErrorMessage != nil {
			continue // Skip failed or incomplete results
		}

		// Get model information
		model, err := db.GetModel(dbResult.ModelID)
		if err != nil {
			continue // Skip if model not found
		}

		// Get provider information for endpoint
		provider, err := db.GetProvider(model.ProviderID)
		if err != nil {
			continue // Skip if provider not found
		}

		result := VerificationResult{
			ModelInfo: ModelInfo{
				ID:                model.Name,
				Description:       model.Description,
				Endpoint:          provider.Endpoint,
				Tags:              model.Tags,
				MaxOutputTokens:   int(*model.MaxOutputTokens),
				ContextWindow:     ContextWindow{TotalMaxTokens: int(*model.ContextWindowTokens)},
				SupportsVision:    model.SupportsVision,
				SupportsAudio:     model.SupportsAudio,
				SupportsVideo:     model.SupportsVideo,
				SupportsReasoning: model.SupportsReasoning,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     dbResult.OverallScore,
				CodeCapability:   dbResult.CodeCapabilityScore,
				Responsiveness:   dbResult.ResponsivenessScore,
				Reliability:      dbResult.ReliabilityScore,
				FeatureRichness:  dbResult.FeatureRichnessScore,
				ValueProposition: dbResult.ValuePropositionScore,
			},
			FeatureDetection: FeatureDetectionResult{
				ToolUse:          dbResult.SupportsToolUse,
				FunctionCalling:  dbResult.SupportsFunctionCalling,
				CodeGeneration:   dbResult.SupportsCodeGeneration,
				CodeCompletion:   dbResult.SupportsCodeCompletion,
				CodeReview:       dbResult.SupportsCodeReview,
				CodeExplanation:  dbResult.SupportsCodeExplanation,
				Embeddings:       dbResult.SupportsEmbeddings,
				Reranking:        dbResult.SupportsReranking,
				ImageGeneration:  dbResult.SupportsImageGeneration,
				AudioGeneration:  dbResult.SupportsAudioGeneration,
				VideoGeneration:  dbResult.SupportsVideoGeneration,
				MCPs:             dbResult.SupportsMCPs,
				LSPs:             dbResult.SupportsLSPs,
				Multimodal:       dbResult.SupportsMultimodal,
				Streaming:        dbResult.SupportsStreaming,
				JSONMode:         dbResult.SupportsJSONMode,
				StructuredOutput: dbResult.SupportsStructuredOutput,
				Reasoning:        dbResult.SupportsReasoning,
				ParallelToolUse:  dbResult.SupportsParallelToolUse,
			},
			CodeCapabilities: CodeCapabilityResult{
				LanguageSupport:    model.LanguageSupport,
				CodeDebugging:      dbResult.CodeDebugging,
				CodeOptimization:   dbResult.CodeOptimization,
				TestGeneration:     dbResult.TestGeneration,
				Refactoring:        dbResult.Refactoring,
				ErrorResolution:    dbResult.ErrorResolution,
				Architecture:       dbResult.ArchitectureDesign,
				SecurityAssessment: dbResult.SecurityAssessment,
				PatternRecognition: dbResult.PatternRecognition,
				DebuggingAccuracy:  dbResult.DebuggingAccuracy,
				ComplexityHandling: ComplexityMetrics{
					MaxHandledDepth:   int(dbResult.MaxHandledDepth),
					CodeQuality:       dbResult.CodeQualityScore,
					LogicCorrectness:  dbResult.LogicCorrectnessScore,
					RuntimeEfficiency: dbResult.RuntimeEfficiencyScore,
				},
			},
			Timestamp: dbResult.CreatedAt,
		}

		results = append(results, result)
	}

	return results, nil
}

// parseJSONField parses JSON string fields, returns empty slice if invalid
func parseJSONField(jsonStr *string) []string {
	if jsonStr == nil || *jsonStr == "" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(*jsonStr), &result); err != nil {
		return []string{}
	}
	return result
}

// createMockVerificationResults creates mock verification results for testing
func createMockVerificationResults() []VerificationResult {
	now := time.Now()

	return []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:                "gpt-4-turbo",
				Description:       "Latest GPT-4 model with improved capabilities",
				Endpoint:          "https://api.openai.com/v1",
				Tags:              []string{"coding", "reasoning", "multimodal"},
				MaxOutputTokens:   4096,
				ContextWindow:     ContextWindow{TotalMaxTokens: 128000},
				SupportsVision:    true,
				SupportsReasoning: true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     92.5,
				CodeCapability:   95.0,
				Responsiveness:   90.0,
				Reliability:      94.0,
				FeatureRichness:  93.0,
				ValueProposition: 85.0,
			},
			FeatureDetection: FeatureDetectionResult{
				CodeGeneration:   true,
				CodeReview:       true,
				CodeExplanation:  true,
				ToolUse:          true,
				FunctionCalling:  true,
				Reasoning:        true,
				Multimodal:       true,
				Streaming:        true,
				JSONMode:         true,
				StructuredOutput: true,
				ParallelToolUse:  true,
			},
			Timestamp: now,
		},
		{
			ModelInfo: ModelInfo{
				ID:                "gpt-3.5-turbo",
				Description:       "Fast and efficient model for general tasks",
				Endpoint:          "https://api.openai.com/v1",
				Tags:              []string{"coding", "chat", "fast"},
				MaxOutputTokens:   4096,
				ContextWindow:     ContextWindow{TotalMaxTokens: 16384},
				SupportsReasoning: true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     78.3,
				CodeCapability:   75.0,
				Responsiveness:   85.0,
				Reliability:      80.0,
				FeatureRichness:  78.0,
				ValueProposition: 90.0,
			},
			FeatureDetection: FeatureDetectionResult{
				CodeGeneration:   true,
				CodeReview:       true,
				CodeExplanation:  true,
				ToolUse:          true,
				FunctionCalling:  true,
				Reasoning:        true,
				Streaming:        true,
				JSONMode:         true,
				StructuredOutput: true,
			},
			Timestamp: now,
		},
		{
			ModelInfo: ModelInfo{
				ID:                "claude-3-5-sonnet-20241022",
				Description:       "Anthropic's most capable model",
				Endpoint:          "https://api.anthropic.com/v1",
				Tags:              []string{"reasoning", "coding", "safety"},
				MaxOutputTokens:   8192,
				ContextWindow:     ContextWindow{TotalMaxTokens: 200000},
				SupportsReasoning: true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     89.7,
				CodeCapability:   88.0,
				Responsiveness:   87.0,
				Reliability:      92.0,
				FeatureRichness:  91.0,
				ValueProposition: 82.0,
			},
			FeatureDetection: FeatureDetectionResult{
				CodeGeneration:   true,
				CodeReview:       true,
				CodeExplanation:  true,
				ToolUse:          true,
				FunctionCalling:  true,
				Reasoning:        true,
				Streaming:        true,
				JSONMode:         true,
				StructuredOutput: true,
			},
			Timestamp: now,
		},
	}
}

// Import existing functions from config_loader.go to avoid duplication
