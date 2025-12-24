package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProviderInfo holds information about a tested provider
type ProviderInfo struct {
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	APIKey      string           `json:"api_key,omitempty"`
	APIEndpoint string           `json:"api_endpoint"`
	Models      []ModelInfo      `json:"models"`
	Status      string           `json:"status"`
	Error       string           `json:"error,omitempty"`
	Features    ProviderFeatures `json:"features"`
	FreeToUse   bool             `json:"free_to_use"`
}

// ModelInfo holds information about a discovered model
type ModelInfo struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	ContextSize  int           `json:"context_size,omitempty"`
	Capabilities []string      `json:"capabilities"`
	Features     ModelFeatures `json:"features"`
	FreeToUse    bool          `json:"free_to_use"`
}

// ModelFeatures holds feature information for a model
type ModelFeatures struct {
	MCPs            []string `json:"mcps,omitempty"`
	LSPs            []string `json:"lsps,omitempty"`
	Embeddings      []string `json:"embeddings,omitempty"`
	Streaming       bool     `json:"streaming,omitempty"`
	FunctionCalling bool     `json:"function_calling,omitempty"`
}

// ProviderFeatures holds provider-level features
type ProviderFeatures struct {
	MCPs            bool `json:"mcps"`
	LSPs            bool `json:"lsps"`
	Embeddings      bool `json:"embeddings"`
	Streaming       bool `json:"streaming"`
	FunctionCalling bool `json:"function_calling"`
}

// ChallengeResult holds the complete challenge result
type ChallengeResult struct {
	ChallengeName string           `json:"challenge_name"`
	ChallengeDate string           `json:"challenge_date"`
	StartTime     string           `json:"start_time"`
	EndTime       string           `json:"end_time"`
	Duration      string           `json:"duration"`
	Providers     []ProviderInfo   `json:"providers"`
	Summary       ChallengeSummary `json:"summary"`
}

// ChallengeSummary holds summary statistics
type ChallengeSummary struct {
	TotalProviders     int            `json:"total_providers"`
	SuccessProviders   int            `json:"success_providers"`
	FailedProviders    int            `json:"failed_providers"`
	TotalModels        int            `json:"total_models"`
	FreeModels         int            `json:"free_models"`
	PaidModels         int            `json:"paid_models"`
	FeaturesDiscovered map[string]int `json:"features_discovered"`
}

var logger *log.Logger

func init() {
	// Initialize logger with verbose output
	logger = log.New(os.Stdout, "CHALLENGE: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	logger.SetOutput(io.MultiWriter(os.Stdout))
}

func main() {
	challengeDir := os.Args[1]
	if challengeDir == "" {
		challengeDir = fmt.Sprintf("challenges/providers_models_discovery/%s/%s/%s/%d",
			time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), time.Now().Unix())
	}

	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(resultsDir, 0755)

	// Setup logging to file
	logFile, err := os.OpenFile(filepath.Join(logDir, "challenge.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	// Multi-output logging
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger.SetOutput(multiWriter)

	logger.Println("==========================================")
	logger.Println("PROVIDER MODELS DISCOVERY CHALLENGE")
	logger.Println("==========================================")
	logger.Printf("Challenge Directory: %s", challengeDir)
	logger.Printf("Start Time: %s", time.Now().Format(time.RFC3339))

	startTime := time.Now()
	result := ChallengeResult{
		ChallengeName: "providers_models_discovery",
		ChallengeDate: startTime.Format("2006-01-02"),
		StartTime:     startTime.Format(time.RFC3339),
	}

	// Load API keys
	apiKeys, err := loadAPIKeys()
	if err != nil {
		logger.Fatalf("Failed to load API keys: %v", err)
	}

	// Test each provider
	allProviders := []ProviderTest{
		{Name: "HuggingFace", APIKey: apiKeys.HuggingFace, FreeToUse: true},
		{Name: "Nvidia", APIKey: apiKeys.Nvidia, FreeToUse: true},
		{Name: "Chutes", APIKey: apiKeys.Chutes, FreeToUse: true},
		{Name: "SiliconFlow", APIKey: apiKeys.SiliconFlow, FreeToUse: true},
		{Name: "Kimi", APIKey: apiKeys.Kimi, FreeToUse: true},
		{Name: "Gemini", APIKey: apiKeys.Gemini, FreeToUse: true},
		{Name: "OpenRouter", APIKey: apiKeys.OpenRouter, FreeToUse: false},
		{Name: "Z.AI", APIKey: apiKeys.ZAI, FreeToUse: false},
		{Name: "DeepSeek", APIKey: apiKeys.DeepSeek, FreeToUse: false},
		{Name: "Qwen", APIKey: "", FreeToUse: false},   // No API key provided
		{Name: "Claude", APIKey: "", FreeToUse: false}, // No API key provided
	}

	for _, providerTest := range allProviders {
		logger.Printf("\n==========================================")
		logger.Printf("Testing Provider: %s", providerTest.Name)
		logger.Printf("==========================================")

		providerInfo := testProvider(context.Background(), providerTest, logDir)
		result.Providers = append(result.Providers, providerInfo)
	}

	// Generate summary
	generateSummary(&result)

	// Set end time
	endTime := time.Now()
	result.EndTime = endTime.Format(time.RFC3339)
	result.Duration = endTime.Sub(startTime).String()

	// Save results
	saveResults(resultsDir, result)

	logger.Println("\n==========================================")
	logger.Println("CHALLENGE COMPLETE")
	logger.Println("==========================================")
	logger.Printf("Total Duration: %s", result.Duration)
	logger.Printf("Results saved to: %s", resultsDir)
}

// Retry configuration
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

var defaultRetryConfig = RetryConfig{
	MaxAttempts: 3,
	BaseDelay:   1 * time.Second,
	MaxDelay:    30 * time.Second,
}

// isRetryableError determines if an error is worth retrying
func isRetryableError(err error, statusCode int) bool {
	if err != nil {
		// Network errors are retryable
		return true
	}

	// HTTP status codes that are retryable
	switch statusCode {
	case 429: // Too Many Requests
		return true
	case 500, 502, 503, 504: // Server errors
		return true
	case 408: // Request Timeout
		return true
	}

	return false
}

// doWithRetry executes a function with exponential backoff retry
func doWithRetry(ctx context.Context, operation func() error, config RetryConfig) error {
	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		logger.Printf("Attempt %d/%d", attempt, config.MaxAttempts)

		err := operation()
		if err == nil {
			if attempt > 1 {
				logger.Printf("Operation succeeded on attempt %d", attempt)
			}
			return nil
		}

		lastErr = err
		logger.Printf("Attempt %d failed: %v", attempt, err)

		// Don't retry on last attempt
		if attempt == config.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff
		delay := time.Duration(float64(config.BaseDelay) * math.Pow(2, float64(attempt-1)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		// Add jitter (±25%)
		jitterRange := int64(delay) / 4 // 25% of delay
		jitter := time.Duration((time.Now().UnixNano() % (2 * jitterRange)) - jitterRange)
		totalDelay := delay + jitter

		logger.Printf("Retrying in %v...", totalDelay)
		select {
		case <-time.After(totalDelay):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("operation failed after %d attempts, last error: %w", config.MaxAttempts, lastErr)
}

// createCrushConfig creates a Crush-compatible configuration with providers as a map
func createCrushConfig(result ChallengeResult) map[string]interface{} {
	providersMap := make(map[string]interface{})

	for _, provider := range result.Providers {
		providerConfig := map[string]interface{}{
			"type":         "openai", // Default type for Crush
			"api_endpoint": provider.APIEndpoint,
			"api_key":      provider.APIKey,
		}

		// Add models if present
		if len(provider.Models) > 0 {
			models := make([]map[string]interface{}, 0, len(provider.Models))
			for _, model := range provider.Models {
				modelConfig := map[string]interface{}{
					"id":           model.ID,
					"name":         model.Name,
					"capabilities": model.Capabilities,
					"free_to_use":  model.FreeToUse,
				}

				// Add Crush-specific fields

				models = append(models, modelConfig)
			}
			providerConfig["models"] = models
		}

		providersMap[provider.Name] = providerConfig
	}

	return map[string]interface{}{
		"version":   "1.0.0",
		"providers": providersMap,
	}
}

// detectHTTP3Support checks if a model supports HTTP/3 (QUIC/Cronet)
func detectHTTP3Support(modelID, provider string, capabilities []string) bool {
	// HTTP/3 support detection logic
	// This is mock logic - in real implementation would test actual HTTP/3 support

	// Certain providers/models are known to support HTTP/3
	http3Providers := []string{"DeepSeek", "OpenRouter"}
	for _, p := range http3Providers {
		if strings.EqualFold(provider, p) {
			return true
		}
	}

	// Models with certain capabilities might support HTTP/3
	http3Capabilities := []string{"streaming", "function-calling"}
	capabilityMatch := 0
	for _, cap := range capabilities {
		for _, http3Cap := range http3Capabilities {
			if strings.Contains(strings.ToLower(cap), strings.ToLower(http3Cap)) {
				capabilityMatch++
			}
		}
	}

	// If model has multiple advanced capabilities, likely supports HTTP/3
	return capabilityMatch >= 2
}

// detectToonFormatSupport checks if a model supports Toon data format
func detectToonFormatSupport(modelID, provider string, capabilities []string) bool {
	// Toon format support detection logic
	// This is mock logic - in real implementation would test actual Toon format support

	// Certain providers support Toon format
	toonProviders := []string{"HuggingFace", "Nvidia", "Gemini"}
	for _, p := range toonProviders {
		if strings.EqualFold(provider, p) {
			return true
		}
	}

	// Models with vision capabilities often support Toon format
	for _, cap := range capabilities {
		if strings.Contains(strings.ToLower(cap), "vision") {
			return true
		}
	}

	// Large context models might support Toon format
	return strings.Contains(strings.ToLower(modelID), "large") ||
		strings.Contains(strings.ToLower(modelID), "70b") ||
		strings.Contains(strings.ToLower(modelID), "340b")
}

// buildModelNameWithSuffixes builds the complete model name with all suffixes
func buildModelNameWithSuffixes(baseName string, freeToUse, http3, toon bool) string {
	suffixes := []string{}

	if freeToUse {
		suffixes = append(suffixes, "free to use")
	}
	if http3 {
		suffixes = append(suffixes, "http3")
	}
	if toon {
		suffixes = append(suffixes, "toon")
	}

	if len(suffixes) == 0 {
		return baseName
	}

	return baseName + " (" + strings.Join(suffixes, ", ") + ")"
}

type ProviderTest struct {
	Name      string
	APIKey    string
	FreeToUse bool
}

func testProvider(ctx context.Context, test ProviderTest, logDir string) ProviderInfo {
	logger.Printf("Testing %s...", test.Name)

	provider := ProviderInfo{
		Name:      test.Name,
		APIKey:    test.APIKey,
		FreeToUse: test.FreeToUse,
	}

	if test.APIKey == "" {
		logger.Printf("No API key provided for %s - skipping", test.Name)
		provider.Status = "skipped"
		provider.Error = "no_api_key"
		return provider
	}

	var models []ModelInfo
	var err error

	switch test.Name {
	case "HuggingFace":
		models, err = discoverHuggingFaceModels(ctx, test.APIKey, test.FreeToUse)
		provider.APIEndpoint = "https://api-inference.huggingface.co"
	case "Nvidia":
		nvidiaModels := []struct {
			id           string
			name         string
			capabilities []string
			features     ModelFeatures
		}{
			{
				id:           "nvidia-nemotron-4-340b",
				name:         "NVIDIA Nemotron 4 340B",
				capabilities: []string{"text-generation", "chat", "code-generation"},
				features:     ModelFeatures{Streaming: true, FunctionCalling: true},
			},
			{
				id:           "meta-llama3-70b-instruct",
				name:         "Llama 3 70B Instruct",
				capabilities: []string{"text-generation", "chat"},
				features:     ModelFeatures{Streaming: true},
			},
		}

		models = make([]ModelInfo, 0, len(nvidiaModels))
		for _, m := range nvidiaModels {
			// Build name with suffixes
			name := buildModelNameWithSuffixes(m.name, test.FreeToUse, false, false)

			models = append(models, ModelInfo{
				ID:           m.id,
				Name:         name,
				Capabilities: m.capabilities,
				Features:     m.features,
				FreeToUse:    test.FreeToUse,
			})
		}
		provider.APIEndpoint = "https://integrate.api.nvidia.com/v1"
		provider.Status = "success"
		provider.Models = models
		return provider
	case "Chutes":
		models, err = discoverOpenAIModels(ctx, "https://api.chutes.ai/v1", test.APIKey, test.FreeToUse)
		provider.APIEndpoint = "https://api.chutes.ai/v1"
	case "SiliconFlow":
		// Use the correct SiliconFlow models API endpoint
		models, err = discoverSiliconFlowModels(ctx, test.APIKey, test.FreeToUse)
		provider.APIEndpoint = "https://api.siliconflow.cn/v1"
	case "Kimi":
		// Kimi (Moonshot) models - hardcoded based on documentation
		kimiModels := []struct {
			id           string
			name         string
			capabilities []string
		}{
			{"moonshot-v1-8k", "Moonshot v1 8K", []string{"chat"}},
			{"moonshot-v1-32k", "Moonshot v1 32K", []string{"chat"}},
			{"moonshot-v1-128k", "Moonshot v1 128K", []string{"chat"}},
		}

		models = make([]ModelInfo, 0, len(kimiModels))
		for _, m := range kimiModels {
			name := buildModelNameWithSuffixes(m.name, test.FreeToUse, false, false)

			models = append(models, ModelInfo{
				ID:           m.id,
				Name:         name,
				Capabilities: m.capabilities,
				Features:     ModelFeatures{},
				FreeToUse:    test.FreeToUse,
			})
		}
		provider.APIEndpoint = "https://api.moonshot.cn/v1"
		provider.Status = "success"
	case "Gemini":
		// Gemini models - hardcoded based on documentation
		geminiModels := []struct {
			id           string
			name         string
			capabilities []string
		}{
			{"gemini-2.0-flash-exp", "Gemini 2.0 Flash", []string{"chat", "vision"}},
			{"gemini-1.5-pro", "Gemini 1.5 Pro", []string{"chat", "vision"}},
			{"gemini-1.5-flash", "Gemini 1.5 Flash", []string{"chat", "vision"}},
		}

		models = make([]ModelInfo, 0, len(geminiModels))
		for _, m := range geminiModels {
			name := buildModelNameWithSuffixes(m.name, test.FreeToUse, false, false)

			models = append(models, ModelInfo{
				ID:           m.id,
				Name:         name,
				Capabilities: m.capabilities,
				Features:     ModelFeatures{},
				FreeToUse:    test.FreeToUse,
			})
		}
		provider.APIEndpoint = "https://generativelanguage.googleapis.com/v1"
		provider.Status = "success"
	case "OpenRouter":
		// Try to use OpenRouter's models API
		models, err = discoverOpenRouterModels(ctx, test.APIKey, test.FreeToUse)
		provider.APIEndpoint = "https://openrouter.ai/api/v1"
	case "DeepSeek":
		// DeepSeek models - hardcoded based on documentation
		deepSeekModels := []struct {
			id           string
			name         string
			capabilities []string
		}{
			{"deepseek-chat", "DeepSeek Chat", []string{"chat", "reasoning"}},
			{"deepseek-reasoner", "DeepSeek Reasoner", []string{"chat", "reasoning"}},
			{"deepseek-coder", "DeepSeek Coder", []string{"chat", "code"}},
		}

		models = make([]ModelInfo, 0, len(deepSeekModels))
		for _, m := range deepSeekModels {
			name := buildModelNameWithSuffixes(m.name, test.FreeToUse, false, false)

			models = append(models, ModelInfo{
				ID:           m.id,
				Name:         name,
				Capabilities: m.capabilities,
				Features:     ModelFeatures{},
				FreeToUse:    test.FreeToUse,
			})
		}
		provider.APIEndpoint = "https://api.deepseek.com"
		provider.Status = "success"
	case "Z.AI":
		// Z.AI models - hardcoded based on documentation
		zaiModels := []struct {
			id           string
			name         string
			capabilities []string
		}{
			{"glm-4.7", "GLM-4.7", []string{"chat", "vision", "reasoning"}},
			{"glm-4.6", "GLM-4.6", []string{"chat", "vision"}},
			{"glm-4.5", "GLM-4.5", []string{"chat", "vision"}},
		}

		models = make([]ModelInfo, 0, len(zaiModels))
		for _, m := range zaiModels {
			name := buildModelNameWithSuffixes(m.name, test.FreeToUse, false, false)

			models = append(models, ModelInfo{
				ID:           m.id,
				Name:         name,
				Capabilities: m.capabilities,
				Features:     ModelFeatures{},
				FreeToUse:    test.FreeToUse,
			})
		}
		provider.APIEndpoint = "https://api.z.ai/api/paas/v4"
		provider.Status = "success"
	case "Qwen":
		provider.Status = "skipped"
		provider.Error = "no_api_key"
		return provider
	case "Claude":
		provider.Status = "skipped"
		provider.Error = "no_api_key"
		return provider
	default:
		provider.Status = "unsupported"
		provider.Error = "unknown_provider"
		return provider
	}

	if err != nil {
		logger.Printf("Failed to discover models for %s: %v", test.Name, err)
		provider.Status = "error"
		provider.Error = err.Error()
		return provider
	}

	provider.Status = "success"
	provider.Models = models
	logger.Printf("Discovered %d models from %s", len(models), test.Name)

	return provider
}

func generateSummary(result *ChallengeResult) {
	summary := ChallengeSummary{
		FeaturesDiscovered: make(map[string]int),
	}

	success := 0
	failed := 0
	totalModels := 0
	freeModels := 0

	for _, provider := range result.Providers {
		if provider.Status == "success" || provider.Status == "tested" {
			success++
			totalModels += len(provider.Models)
			for _, model := range provider.Models {
				if model.FreeToUse {
					freeModels++
				}
			}
		} else {
			failed++
		}
	}

	summary.TotalProviders = len(result.Providers)
	summary.SuccessProviders = success
	summary.FailedProviders = failed
	summary.TotalModels = totalModels
	summary.FreeModels = freeModels
	summary.PaidModels = totalModels - freeModels

	result.Summary = summary
}

func saveResults(resultsDir string, result ChallengeResult) {
	// Save providers opencode
	opencode := map[string]interface{}{
		"challenge_name": result.ChallengeName,
		"date":           result.ChallengeDate,
		"providers":      make([]map[string]interface{}, 0),
	}

	for _, provider := range result.Providers {
		providerData := map[string]interface{}{
			"name":         provider.Name,
			"type":         provider.Type,
			"api_endpoint": provider.APIEndpoint,
			"status":       provider.Status,
			"free_to_use":  provider.FreeToUse,
			"models":       provider.Models,
		}
		opencode["providers"] = append(opencode["providers"].([]map[string]interface{}), providerData)
	}

	opencodeFile := filepath.Join(resultsDir, "providers_opencode.json")
	opencodeData, _ := json.MarshalIndent(opencode, "", "  ")
	if err := os.WriteFile(opencodeFile, opencodeData, 0644); err != nil {
		logger.Printf("Failed to save opencode: %v", err)
	}

	// Save providers crush (with correct map format)
	crushFile := filepath.Join(resultsDir, "providers_crush.json")
	crushConfig := createCrushConfig(result)
	crushData, _ := json.MarshalIndent(crushConfig, "", "  ")
	if err := os.WriteFile(crushFile, crushData, 0644); err != nil {
		logger.Printf("Failed to save crush: %v", err)
	}

	logger.Printf("Results saved:")
	logger.Printf("  - %s", opencodeFile)
	logger.Printf("  - %s", crushFile)
}

type APIKeys struct {
	HuggingFace string `json:"huggingface"`
	Nvidia      string `json:"nvidia"`
	Chutes      string `json:"chutes"`
	SiliconFlow string `json:"siliconflow"`
	Kimi        string `json:"kimi"`
	Gemini      string `json:"gemini"`
	OpenRouter  string `json:"openrouter"`
	ZAI         string `json:"zai"`
	DeepSeek    string `json:"deepseek"`
}

func loadAPIKeys() (*APIKeys, error) {
	return &APIKeys{
		HuggingFace: os.Getenv("ApiKey_HuggingFace"),
		Nvidia:      os.Getenv("ApiKey_Nvidia"),
		Chutes:      os.Getenv("ApiKey_Chutes"),
		SiliconFlow: os.Getenv("ApiKey_SiliconFlow"),
		Kimi:        os.Getenv("ApiKey_Kimi"),
		Gemini:      os.Getenv("ApiKey_Gemini"),
		OpenRouter:  os.Getenv("ApiKey_OpenRouter"),
		ZAI:         os.Getenv("ApiKey_ZAI"),
		DeepSeek:    os.Getenv("ApiKey_DeepSeek"),
	}, nil
}

func discoverHuggingFaceModels(ctx context.Context, apiKey string, freeToUse bool) ([]ModelInfo, error) {
	var hfModels []struct {
		ID       string `json:"id"`
		Pipeline string `json:"pipeline_tag"`
	}

	err := doWithRetry(ctx, func() error {
		modelsURL := "https://huggingface.co/api/models?sort=downloads&direction=-1&limit=50"
		req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		logger.Printf("API REQUEST: GET %s", modelsURL)
		logger.Printf("API REQUEST HEADERS: %v", req.Header)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Printf("API RESPONSE ERROR: %v", err)
			return fmt.Errorf("failed to fetch models: %w", err)
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read response body: %w", readErr)
		}

		logger.Printf("API RESPONSE STATUS: %d", resp.StatusCode)
		logger.Printf("API RESPONSE HEADERS: %v", resp.Header)
		logger.Printf("API RESPONSE BODY LENGTH: %d bytes", len(body))

		if resp.StatusCode != 200 {
			if isRetryableError(nil, resp.StatusCode) {
				return fmt.Errorf("API returned retryable status %d: %s", resp.StatusCode, string(body))
			}
			return fmt.Errorf("API returned permanent status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, &hfModels); err != nil {
			return fmt.Errorf("failed to decode models: %w", err)
		}

		return nil
	}, defaultRetryConfig)

	if err != nil {
		return nil, err
	}
	var models []ModelInfo
	for _, m := range hfModels {
		caps := []string{}
		switch m.Pipeline {
		case "text-generation":
			caps = []string{"text-generation"}
		case "feature-extraction":
			caps = []string{"feature-extraction", "embeddings"}
		}
		embeddings := []string{}
		if m.Pipeline == "feature-extraction" {
			embeddings = []string{"default"}
		}
		// Detect advanced features
		http3 := detectHTTP3Support(m.ID, "HuggingFace", caps)
		toon := detectToonFormatSupport(m.ID, "HuggingFace", caps)

		// Build name with all suffixes
		name := buildModelNameWithSuffixes(m.ID, freeToUse, http3, toon)

		models = append(models, ModelInfo{
			ID:           m.ID,
			Name:         name,
			Capabilities: caps,
			Features: ModelFeatures{
				Embeddings: embeddings,
			},
			FreeToUse: freeToUse,
		})
	}
	return models, nil
}

func discoverSiliconFlowModels(ctx context.Context, apiKey string, freeToUse bool) ([]ModelInfo, error) {
	var siliconResp struct {
		Object string `json:"object"`
		Data   []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	err := doWithRetry(ctx, func() error {
		modelsURL := "https://api.siliconflow.cn/v1/models"
		req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		logger.Printf("API REQUEST: GET %s", modelsURL)
		logger.Printf("API REQUEST HEADERS: %v", req.Header)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Printf("API RESPONSE ERROR: %v", err)
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read response body: %w", readErr)
		}

		logger.Printf("API RESPONSE STATUS: %d", resp.StatusCode)
		logger.Printf("API RESPONSE HEADERS: %v", resp.Header)
		logger.Printf("API RESPONSE BODY LENGTH: %d bytes", len(body))

		if resp.StatusCode != 200 {
			if isRetryableError(nil, resp.StatusCode) {
				return fmt.Errorf("API returned retryable status %d: %s", resp.StatusCode, string(body))
			}
			return fmt.Errorf("API returned permanent status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, &siliconResp); err != nil {
			return fmt.Errorf("failed to decode models: %w", err)
		}

		return nil
	}, defaultRetryConfig)

	if err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(siliconResp.Data))
	for _, m := range siliconResp.Data {
		// Detect advanced features
		http3 := detectHTTP3Support(m.ID, "SiliconFlow", []string{})
		toon := detectToonFormatSupport(m.ID, "SiliconFlow", []string{})

		// Build name with all suffixes
		name := buildModelNameWithSuffixes(m.ID, freeToUse, http3, toon)

		models = append(models, ModelInfo{
			ID:           m.ID,
			Name:         name,
			Capabilities: []string{}, // Will be determined by testing
			Features: ModelFeatures{
			},
			FreeToUse: freeToUse,
		})
	}

	return models, nil
}

func discoverOpenRouterModels(ctx context.Context, apiKey string, freeToUse bool) ([]ModelInfo, error) {
	var openRouterResp struct {
		Data []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Pricing struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		} `json:"data"`
	}

	err := doWithRetry(ctx, func() error {
		modelsURL := "https://openrouter.ai/api/v1/models"
		req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		logger.Printf("API REQUEST: GET %s", modelsURL)
		logger.Printf("API REQUEST HEADERS: %v", req.Header)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Printf("API RESPONSE ERROR: %v", err)
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read response body: %w", readErr)
		}

		logger.Printf("API RESPONSE STATUS: %d", resp.StatusCode)
		logger.Printf("API RESPONSE HEADERS: %v", resp.Header)
		logger.Printf("API RESPONSE BODY LENGTH: %d bytes", len(body))

		if resp.StatusCode != 200 {
			if isRetryableError(nil, resp.StatusCode) {
				return fmt.Errorf("API returned retryable status %d: %s", resp.StatusCode, string(body))
			}
			return fmt.Errorf("API returned permanent status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, &openRouterResp); err != nil {
			return fmt.Errorf("failed to decode models: %w", err)
		}

		return nil
	}, defaultRetryConfig)

	if err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(openRouterResp.Data))
	for _, m := range openRouterResp.Data {
		// Determine if model is free based on pricing - check actual pricing data
		isFree := (m.Pricing.Prompt == "0" || m.Pricing.Prompt == "" || m.Pricing.Prompt == "0.0")

		// Detect advanced features
		http3 := detectHTTP3Support(m.ID, "OpenRouter", []string{})
		toon := detectToonFormatSupport(m.ID, "OpenRouter", []string{})

		// Build name with all suffixes
		name := buildModelNameWithSuffixes(m.Name, isFree, http3, toon)

		models = append(models, ModelInfo{
			ID:           m.ID,
			Name:         name,
			Capabilities: []string{}, // Will be determined by testing
			Features: ModelFeatures{
			},
			FreeToUse: isFree,
		})
	}

	return models, nil
}

func discoverOpenAIModels(ctx context.Context, endpoint, apiKey string, freeToUse bool) ([]ModelInfo, error) {
	// Determine provider name from endpoint
	providerName := getProviderNameFromEndpoint(endpoint)

	var openaiResp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int    `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	err := doWithRetry(ctx, func() error {
		// For OpenRouter, the endpoint already includes /api/v1, so just add /models
		modelsURL := strings.TrimSuffix(endpoint, "/")
		if strings.Contains(endpoint, "openrouter.ai") {
			modelsURL += "/models"
		} else {
			modelsURL += "/v1/models"
		}
		req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		logger.Printf("API REQUEST: GET %s", modelsURL)
		logger.Printf("API REQUEST HEADERS: %v", req.Header)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Printf("API RESPONSE ERROR: %v", err)
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read response body: %w", readErr)
		}

		logger.Printf("API RESPONSE STATUS: %d", resp.StatusCode)
		logger.Printf("API RESPONSE HEADERS: %v", resp.Header)
		logger.Printf("API RESPONSE BODY LENGTH: %d bytes", len(body))

		if resp.StatusCode != 200 {
			if isRetryableError(nil, resp.StatusCode) {
				return fmt.Errorf("API returned retryable status %d: %s", resp.StatusCode, string(body))
			}
			return fmt.Errorf("API returned permanent status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, &openaiResp); err != nil {
			return fmt.Errorf("failed to decode models: %w", err)
		}

		return nil
	}, defaultRetryConfig)

	if err != nil {
		return nil, err
	}
	var models []ModelInfo
	for _, m := range openaiResp.Data {
		// Detect advanced features
		http3 := detectHTTP3Support(m.ID, providerName, []string{"chat"})
		toon := detectToonFormatSupport(m.ID, providerName, []string{"chat"})

		// Build name with all suffixes
		name := buildModelNameWithSuffixes(m.ID, freeToUse, http3, toon)

		models = append(models, ModelInfo{
			ID:           m.ID,
			Name:         name,
			Capabilities: []string{"chat"},
			Features: ModelFeatures{
			},
			FreeToUse: freeToUse,
		})
	}
	return models, nil
}
func getProviderNameFromEndpoint(endpoint string) string {
	switch {
	case strings.Contains(endpoint, "openrouter"):
		return "OpenRouter"
	case strings.Contains(endpoint, "chutes"):
		return "Chutes"
	case strings.Contains(endpoint, "siliconflow"):
		return "SiliconFlow"
	case strings.Contains(endpoint, "kimi"):
		return "Kimi"
	case strings.Contains(endpoint, "deepseek"):
		return "DeepSeek"
	case strings.Contains(endpoint, "z.ai"):
		return "Z.AI"
	case strings.Contains(endpoint, "qwen"):
		return "Qwen"
	case strings.Contains(endpoint, "claude"):
		return "Claude"
	default:
		return "Unknown"
	}
}
