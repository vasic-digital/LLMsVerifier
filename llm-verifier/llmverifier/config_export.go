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
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Provider     string                 `json:"provider"`
	Endpoint     string                 `json:"endpoint"`
	APIKey       string                 `json:"api_key,omitempty"`
	Capabilities []string               `json:"capabilities"`
	Score        float64                `json:"score"`
	Category     string                 `json:"category"`
	Tags         []string               `json:"tags"`
	Description  string                 `json:"description,omitempty"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
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
func ExportConfig(cfg *config.Config, format, outputPath string) error {
	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(cfg, "", "  ")
	case "yaml", "yml":
		data, err = yaml.Marshal(cfg)
	case "opencode":
		return ExportAIConfig(cfg, "opencode", outputPath, &ExportOptions{})
	case "crush":
		return ExportAIConfig(cfg, "crush", outputPath, &ExportOptions{})
	case "claude-code":
		return ExportAIConfig(cfg, "claude-code", outputPath, &ExportOptions{})
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
func ExportAIConfig(cfg *config.Config, aiFormat, outputPath string, options *ExportOptions) error {
	// For now, we'll create a mock verification results structure
	// In real implementation, this would come from database
	results := createMockVerificationResults()

	// Filter models based on options
	filteredModels := filterModels(results, options)

	// Create AI-specific configuration
	var aiConfig *AIConfig
	var err error

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

		model := AIModel{
			ID:           result.ModelInfo.ID,
			Name:         result.ModelInfo.ID,
			Provider:     extractProvider(result.ModelInfo.Endpoint),
			Endpoint:     result.ModelInfo.Endpoint,
			Capabilities: capabilities,
			Score:        result.PerformanceScores.OverallScore,
			Category:     category,
			Tags:         result.ModelInfo.Tags,
			Description:  result.ModelInfo.Description,
			Settings: map[string]interface{}{
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
			},
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

// createCrushConfig creates configuration for Crush
func createCrushConfig(results []VerificationResult, options *ExportOptions) (*AIConfig, error) {
	models := make([]AIModel, 0, len(results))
	var totalScore float64

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		// Crush focuses on coding capabilities - use overall score for now
		if result.PerformanceScores.OverallScore < 70 {
			continue // Skip models with poor overall score
		}

		capabilities := extractCapabilities(result)
		category := categorizeModel(result)

		model := AIModel{
			ID:           result.ModelInfo.ID,
			Name:         result.ModelInfo.ID,
			Provider:     extractProvider(result.ModelInfo.Endpoint),
			Endpoint:     result.ModelInfo.Endpoint,
			Capabilities: capabilities,
			Score:        result.PerformanceScores.OverallScore,
			Category:     category,
			Tags:         result.ModelInfo.Tags,
			Description:  result.ModelInfo.Description,
			Settings: map[string]interface{}{
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
			},
		}

		if options != nil && options.IncludeAPIKey {
			model.APIKey = "YOUR_API_KEY_HERE"
		}

		models = append(models, model)
		totalScore += result.PerformanceScores.OverallScore
	}

	// Sort by overall score (highest first)
	slices.SortFunc(models, func(a, b AIModel) int {
		if b.Score > a.Score {
			return 1
		}
		if b.Score < a.Score {
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
			AutoSave:        true,
			StreamResponses: true,
			Language:        "english",
		},
		Metadata: Metadata{
			TotalModels:        len(models),
			AverageScore:       avgScore,
			ExportCriteria:     "Coding-focused models (score >= 70)",
			LLMVerifierVersion: "1.0.0",
			LastUpdated:        time.Now(),
		},
	}, nil
}

// createClaudeCodeConfig creates configuration for Claude Code
func createClaudeCode(results []VerificationResult, options *ExportOptions) (*AIConfig, error) {
	models := make([]AIModel, 0, len(results))
	var totalScore float64

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		capabilities := extractCapabilities(result)
		category := categorizeModel(result)

		model := AIModel{
			ID:           result.ModelInfo.ID,
			Name:         result.ModelInfo.ID,
			Provider:     extractProvider(result.ModelInfo.Endpoint),
			Endpoint:     result.ModelInfo.Endpoint,
			Capabilities: capabilities,
			Score:        result.PerformanceScores.OverallScore,
			Category:     category,
			Tags:         result.ModelInfo.Tags,
			Description:  result.ModelInfo.Description,
			Settings: map[string]interface{}{
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
			},
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
func ExportBulkConfig(cfg *config.Config, outputPath string, options *ExportOptions) error {
	if options == nil {
		options = &ExportOptions{}
	}

	// Create all supported formats
	formats := []string{"opencode", "crush", "claude-code"}

	for _, format := range formats {
		// Generate filename for this format
		safeFormat := strings.ReplaceAll(format, "-", "_")
		filename := fmt.Sprintf("export_%s.json", safeFormat)
		fullPath := filepath.Join(outputPath, filename)

		// Export configuration for this format
		err := ExportAIConfig(cfg, format, fullPath, options)
		if err != nil {
			return fmt.Errorf("failed to export %s format: %w", format, err)
		}
	}

	// Create summary file
	return createExportSummary(outputPath, formats, options)
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

// ValidateExportedConfig validates an exported AI CLI configuration
func ValidateExportedConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read exported config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse exported config: %w", err)
	}

	// Validate required fields
	if _, ok := config["version"]; !ok {
		return fmt.Errorf("missing version field")
	}

	if _, ok := config["models"]; !ok {
		return fmt.Errorf("missing models field")
	}

	if _, ok := config["preferences"]; !ok {
		return fmt.Errorf("missing preferences field")
	}

	// Validate models array
	models, ok := config["models"].([]interface{})
	if !ok {
		return fmt.Errorf("models field must be an array")
	}

	if len(models) == 0 {
		return fmt.Errorf("models array cannot be empty")
	}

	// Validate each model
	for i, model := range models {
		modelMap, ok := model.(map[string]interface{})
		if !ok {
			return fmt.Errorf("model %d must be an object", i)
		}

		// Validate required model fields
		requiredFields := []string{"id", "name", "provider", "endpoint", "score", "category"}
		for _, field := range requiredFields {
			if _, ok := modelMap[field]; !ok {
				return fmt.Errorf("model %d missing required field: %s", i, field)
			}
		}

		// Validate score
		if score, ok := modelMap["score"].(float64); !ok || score < 0 || score > 100 {
			return fmt.Errorf("model %d has invalid score: %v", i, modelMap["score"])
		}
	}

	return nil
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
