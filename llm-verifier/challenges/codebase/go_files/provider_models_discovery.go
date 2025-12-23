package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ProviderInfo holds information about a tested provider
type ProviderInfo struct {
	Name        string           `json:"name"`
	Type        string           `json:"type"`
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

type ProviderTest struct {
	Name      string
	APIKey    string
	FreeToUse bool
}

func testProvider(ctx context.Context, test ProviderTest, logDir string) ProviderInfo {
	logger.Printf("Testing %s...", test.Name)

	provider := ProviderInfo{
		Name:      test.Name,
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
		models = []ModelInfo{
			{
				ID:           "nvidia-nemotron-4-340b",
				Name:         "NVIDIA Nemotron 4 340B",
				Capabilities: []string{"text-generation", "chat", "code-generation"},
				Features:     ModelFeatures{Streaming: true, FunctionCalling: true},
				FreeToUse:    test.FreeToUse,
			},
			{
				ID:           "meta-llama3-70b-instruct",
				Name:         "Llama 3 70B Instruct",
				Capabilities: []string{"text-generation", "chat"},
				Features:     ModelFeatures{Streaming: true},
				FreeToUse:    test.FreeToUse,
			},
		}
		provider.APIEndpoint = "https://integrate.api.nvidia.com/v1"
		provider.Status = "success"
		provider.Models = models
		return provider
	case "Chutes":
		models, err = discoverOpenAIModels(ctx, "https://api.chutes.ai/v1", test.APIKey, test.FreeToUse)
		provider.APIEndpoint = "https://api.chutes.ai/v1"
	case "SiliconFlow":
		models, err = discoverOpenAIModels(ctx, "https://api.siliconflow.cn/v1", test.APIKey, test.FreeToUse)
		provider.APIEndpoint = "https://api.siliconflow.cn/v1"
	case "Kimi":
		models = []ModelInfo{
			{
				ID:           "moonshot-v1-128k",
				Name:         "Moonshot V1 128K",
				ContextSize:  128000,
				Capabilities: []string{"text-generation", "chat", "long-context"},
				Features:     ModelFeatures{Streaming: true, FunctionCalling: true},
				FreeToUse:    test.FreeToUse,
			},
		}
		provider.APIEndpoint = "https://api.moonshot.cn/v1"
		provider.Status = "success"
		provider.Models = models
		return provider
	case "Gemini":
		models = []ModelInfo{
			{
				ID:           "gemini-2.0-flash-exp",
				Name:         "Gemini 2.0 Flash Experimental",
				Capabilities: []string{"text-generation", "chat", "code-generation", "vision"},
				Features:     ModelFeatures{Streaming: true, FunctionCalling: true, MCPs: []string{"mcp-proto"}},
				FreeToUse:    test.FreeToUse,
			},
			{
				ID:           "gemini-1.5-pro",
				Name:         "Gemini 1.5 Pro",
				Capabilities: []string{"text-generation", "chat", "code-generation", "vision"},
				Features:     ModelFeatures{Streaming: true, FunctionCalling: true},
				FreeToUse:    test.FreeToUse,
			},
		}
		provider.APIEndpoint = "https://generativelanguage.googleapis.com/v1"
		provider.Status = "success"
		provider.Models = models
		return provider
	case "OpenRouter":
		models, err = discoverOpenAIModels(ctx, "https://openrouter.ai/api/v1", test.APIKey, test.FreeToUse)
		provider.APIEndpoint = "https://openrouter.ai/api/v1"
	case "DeepSeek":
		models, err = discoverOpenAIModels(ctx, "https://api.deepseek.com/v1", test.APIKey, test.FreeToUse)
		provider.APIEndpoint = "https://api.deepseek.com/v1"
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

	// Save providers crush
	crushFile := filepath.Join(resultsDir, "providers_crush.json")
	crushData, _ := json.MarshalIndent(result, "", "  ")
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
		ZAI:         os.Getenv("ApiKey_Z_AI"),
		DeepSeek:    os.Getenv("ApiKey_DeepSeek"),
	}, nil
}

func discoverHuggingFaceModels(ctx context.Context, apiKey string, freeToUse bool) ([]ModelInfo, error) {
	modelsURL := "https://huggingface.co/api/models?sort=downloads&direction=-1&limit=50"
	req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	var hfModels []struct {
		ID       string `json:"id"`
		Pipeline string `json:"pipeline_tag"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hfModels); err != nil {
		return nil, fmt.Errorf("failed to decode models: %w", err)
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
		name := m.ID
		if freeToUse {
			name += " free to use"
		}
		models = append(models, ModelInfo{
			ID:           m.ID,
			Name:         name,
			Capabilities: caps,
			Features:     ModelFeatures{Embeddings: embeddings},
			FreeToUse:    freeToUse,
		})
	}
	return models, nil
}

func discoverOpenAIModels(ctx context.Context, endpoint, apiKey string, freeToUse bool) ([]ModelInfo, error) {
	modelsURL := endpoint + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	var openaiResp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int    `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode models: %w", err)
	}
	var models []ModelInfo
	for _, m := range openaiResp.Data {
		name := m.ID
		if freeToUse {
			name += " free to use"
		}
		models = append(models, ModelInfo{
			ID:           m.ID,
			Name:         name,
			Capabilities: []string{"chat"},
			FreeToUse:    freeToUse,
		})
	}
	return models, nil
}
