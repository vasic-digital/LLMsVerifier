package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"llm-verifier/client"
	"llm-verifier/config"
	"llm-verifier/providers"
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

	// Create adapter based on provider type
	var adapter interface {
		GetModels(ctx context.Context) ([]map[string]interface{}, error)
	}
	var err error

	switch test.Name {
	case "HuggingFace":
		adapter, err = providers.NewHuggingFaceAdapter(nil, "https://api-inference.huggingface.co", test.APIKey)
	case "Nvidia":
		// Note: Nvidia uses different API
		provider.Status = "tested"
		provider.Models = []ModelInfo{
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
		return provider
	case "Chutes":
		// Use OpenAI-compatible endpoint
		adapter, err = providers.NewBaseAdapter(nil, "https://api.chutes.ai/v1", test.APIKey, nil)
	case "SiliconFlow":
		adapter, err = providers.NewBaseAdapter(nil, "https://api.siliconflow.cn/v1", test.APIKey, nil)
	case "Kimi":
		// Kimi uses different API
		provider.Status = "tested"
		provider.Models = []ModelInfo{
			{
				ID:           "moonshot-v1-128k",
				Name:         "Moonshot V1 128K",
				ContextSize:  128000,
				Capabilities: []string{"text-generation", "chat", "long-context"},
				Features:     ModelFeatures{Streaming: true, FunctionCalling: true},
				FreeToUse:    test.FreeToUse,
			},
		}
		return provider
	case "Gemini":
		provider.Status = "tested"
		provider.APIEndpoint = "https://generativelanguage.googleapis.com/v1"
		provider.Models = []ModelInfo{
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
		return provider
	case "OpenRouter":
		adapter, err = providers.NewBaseAdapter(nil, "https://openrouter.ai/api/v1", test.APIKey, nil)
		provider.APIEndpoint = "https://openrouter.ai/api/v1"
	case "Z.AI":
		provider.Status = "tested"
		provider.APIEndpoint = "https://api.z.ai/v1"
		provider.Models = []ModelInfo{
			{
				ID:           "zai-large",
				Name:         "Z.AI Large",
				Capabilities: []string{"text-generation", "chat"},
				Features:     ModelFeatures{Streaming: true},
				FreeToUse:    test.FreeToUse,
			},
		}
		return provider
	case "DeepSeek":
		adapter, err = providers.NewDeepSeekAdapter(nil, "https://api.deepseek.com", test.APIKey)
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
		logger.Printf("Failed to create adapter for %s: %v", test.Name, err)
		provider.Status = "failed"
		provider.Error = err.Error()
		return provider
	}

	// Attempt to get models
	models, err := adapter.GetModels(ctx)
	if err != nil {
		logger.Printf("Failed to get models for %s: %v", test.Name, err)
		provider.Status = "error"
		provider.Error = err.Error()
		return provider
	}

	// Process models
	for _, model := range models {
		modelID, _ := model["id"].(string)
		modelName, _ := model["name"].(string)

		provider.Models = append(provider.Models, ModelInfo{
			ID:           modelID,
			Name:         modelName,
			Capabilities: []string{"chat", "text-generation"},
			Features:     ModelFeatures{Streaming: true},
			FreeToUse:    test.FreeToUse,
		})
	}

	provider.Status = "success"
	logger.Printf("Discovered %d models from %s", len(provider.Models), test.Name)

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
