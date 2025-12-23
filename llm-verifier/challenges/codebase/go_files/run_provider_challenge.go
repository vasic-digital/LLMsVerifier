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
	"strings"
	"time"
)

// Challenge results structures
type ProviderResult struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Endpoint  string      `json:"endpoint"`
	Status    string      `json:"status"`
	Error     string      `json:"error,omitempty"`
	Models    []ModelInfo `json:"models"`
	Features  Features    `json:"features"`
	FreeToUse bool        `json:"free_to_use"`
	Latency   string      `json:"latency,omitempty"`
	TestTime  string      `json:"test_time"`
}

type ModelInfo struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	ContextSize  int           `json:"context_size,omitempty"`
	Capabilities []string      `json:"capabilities"`
	Features     ModelFeatures `json:"features"`
	FreeToUse    bool          `json:"free_to_use"`
}

type ModelFeatures struct {
	MCPs            []string `json:"mcps,omitempty"`
	LSPs            []string `json:"lsps,omitempty"`
	Embeddings      []string `json:"embeddings,omitempty"`
	Streaming       bool     `json:"streaming"`
	FunctionCalling bool     `json:"function_calling"`
	Vision          bool     `json:"vision"`
	Tools           bool     `json:"tools"`
}

type Features struct {
	MCPs            bool `json:"mcps"`
	LSPs            bool `json:"lsps"`
	Embeddings      bool `json:"embeddings"`
	Streaming       bool `json:"streaming"`
	FunctionCalling bool `json:"function_calling"`
	Vision          bool `json:"vision"`
	Tools           bool `json:"tools"`
}

type ChallengeResult struct {
	ChallengeName string           `json:"challenge_name"`
	StartTime     string           `json:"start_time"`
	EndTime       string           `json:"end_time"`
	Duration      string           `json:"duration"`
	Providers     []ProviderResult `json:"providers"`
	Summary       ChallengeSummary `json:"summary"`
}

type ChallengeSummary struct {
	TotalProviders int `json:"total_providers"`
	SuccessCount   int `json:"success_count"`
	ErrorCount     int `json:"error_count"`
	SkippedCount   int `json:"skipped_count"`
	TotalModels    int `json:"total_models"`
	FreeModels     int `json:"free_models"`
}

var (
	logger       *log.Logger
	verboseLevel int = 3
)

func initLogger(logDir string) *log.Logger {
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "challenge.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return log.New(os.Stdout, "[CHALLENGE] ", log.Ldate|log.Ltime)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	return log.New(multiWriter, "[CHALLENGE] ", log.Ldate|log.Ltime|log.Lmicroseconds)
}

func main() {
	timestamp := time.Now().Unix()
	challengeDir := filepath.Join("challenges", "provider_models_discovery",
		time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"),
		fmt.Sprintf("%d", timestamp))

	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")

	logger = initLogger(logDir)

	logger.Println("======================================================")
	logger.Println("PROVIDER MODELS DISCOVERY CHALLENGE")
	logger.Println("======================================================")
	logger.Printf("Challenge Directory: %s", challengeDir)
	logger.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))

	apiKeys := map[string]string{
		"huggingface": os.Getenv("ApiKey_HuggingFace"),
		"nvidia":      os.Getenv("ApiKey_Nvidia"),
		"chutes":      os.Getenv("ApiKey_Chutes"),
		"siliconflow": os.Getenv("ApiKey_SiliconFlow"),
		"kimi":        os.Getenv("ApiKey_Kimi"),
		"gemini":      os.Getenv("ApiKey_Gemini"),
		"openrouter":  os.Getenv("ApiKey_OpenRouter"),
		"zai":         os.Getenv("ApiKey_Z_AI"),
		"deepseek":    os.Getenv("ApiKey_DeepSeek"),
	}

	logger.Printf("Loaded %d API keys", len(apiKeys))
	loggerVerbose(1, "API Keys for: %s", strings.Join(getMapKeys(apiKeys), ", "))

	result := runChallenge(apiKeys)

	if err := saveResults(resultsDir, result); err != nil {
		logger.Printf("Failed to save results: %v", err)
	}

	logger.Println("======================================================")
	logger.Println("CHALLENGE COMPLETE")
	logger.Println("======================================================")
	logger.Printf("Duration: %s", result.Duration)
	logger.Printf("Results saved to: %s", resultsDir)
}

func runChallenge(apiKeys map[string]string) ChallengeResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTime := time.Now()
	result := ChallengeResult{
		ChallengeName: "provider_models_discovery",
		StartTime:     startTime.Format(time.RFC3339),
	}

	providersToTest := []struct {
		Name      string
		Endpoint  string
		APIKey    string
		FreeToUse bool
	}{
		{Name: "HuggingFace", Endpoint: "https://api-inference.huggingface.co", APIKey: apiKeys["huggingface"], FreeToUse: true},
		{Name: "Nvidia", Endpoint: "https://integrate.api.nvidia.com/v1", APIKey: apiKeys["nvidia"], FreeToUse: true},
		{Name: "Chutes", Endpoint: "https://api.chutes.ai/v1", APIKey: apiKeys["chutes"], FreeToUse: true},
		{Name: "SiliconFlow", Endpoint: "https://api.siliconflow.cn/v1", APIKey: apiKeys["siliconflow"], FreeToUse: true},
		{Name: "Kimi", Endpoint: "https://api.moonshot.cn/v1", APIKey: apiKeys["kimi"], FreeToUse: true},
		{Name: "Gemini", Endpoint: "https://generativelanguage.googleapis.com/v1", APIKey: apiKeys["gemini"], FreeToUse: true},
		{Name: "OpenRouter", Endpoint: "https://openrouter.ai/api/v1", APIKey: apiKeys["openrouter"], FreeToUse: false},
		{Name: "Z.AI", Endpoint: "https://api.z.ai/v1", APIKey: apiKeys["zai"], FreeToUse: false},
		{Name: "DeepSeek", Endpoint: "https://api.deepseek.com", APIKey: apiKeys["deepseek"], FreeToUse: false},
		{Name: "Qwen", Endpoint: "", APIKey: "", FreeToUse: false},
		{Name: "Claude", Endpoint: "", APIKey: "", FreeToUse: false},
	}

	loggerVerbose(1, "Testing %d providers", len(providersToTest))

	for _, provider := range providersToTest {
		providerResult := testProvider(ctx, provider)
		result.Providers = append(result.Providers, providerResult)
	}

	result.Summary = generateSummary(result)
	result.EndTime = time.Now().Format(time.RFC3339)
	result.Duration = time.Since(startTime).String()

	return result
}

func testProvider(ctx context.Context, provider struct {
	Name      string
	Endpoint  string
	APIKey    string
	FreeToUse bool
}) ProviderResult {
	logger.Printf("\n======================================================")
	logger.Printf("Testing Provider: %s", provider.Name)
	logger.Printf("======================================================")

	result := ProviderResult{
		Name:      provider.Name,
		Type:      "llm",
		Endpoint:  provider.Endpoint,
		FreeToUse: provider.FreeToUse,
		TestTime:  time.Now().Format(time.RFC3339),
	}

	if provider.APIKey == "" {
		loggerVerbose(2, "No API key for %s - marking as skipped", provider.Name)
		result.Status = "skipped"
		result.Error = "no_api_key"
		return result
	}

	startTime := time.Now()
	models, features, err := discoverProviderModels(ctx, provider)
	if err != nil {
		loggerVerbose(2, "Failed to discover models for %s: %v", provider.Name, err)
		result.Status = "error"
		result.Error = fmt.Sprintf("discovery_failed: %v", err)
		result.Latency = time.Since(startTime).String()
		return result
	}

	result.Models = models
	result.Features = features
	result.Status = "success"
	result.Latency = time.Since(startTime).String()

	loggerVerbose(1, "Successfully tested %s", provider.Name)
	loggerVerbose(1, "  Latency: %s", result.Latency)
	loggerVerbose(1, "  Models discovered: %d", len(models))
	loggerVerbose(1, "  Features: MCPs=%t, LSPs=%t, Embeddings=%t, Streaming=%t",
		features.MCPs, features.LSPs, features.Embeddings, features.Streaming)

	return result
}

func discoverProviderModels(ctx context.Context, provider struct {
	Name      string
	Endpoint  string
	APIKey    string
	FreeToUse bool
}) ([]ModelInfo, Features, error) {
	var models []ModelInfo
	var features Features

	providerName := strings.ToLower(provider.Name)

	// Attempt to discover models via API
	switch providerName {
	case "huggingface":
		// HuggingFace has a models API: https://huggingface.co/api/models
		modelsURL := "https://huggingface.co/api/models?sort=downloads&direction=-1&limit=50"
		req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
		if err != nil {
			return nil, Features{}, fmt.Errorf("failed to create request: %w", err)
		}
		if provider.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+provider.APIKey)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, Features{}, fmt.Errorf("failed to fetch models: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, Features{}, fmt.Errorf("API returned status %d", resp.StatusCode)
		}
		var hfModels []struct {
			ID       string `json:"id"`
			Pipeline string `json:"pipeline_tag"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&hfModels); err != nil {
			return nil, Features{}, fmt.Errorf("failed to decode models: %w", err)
		}
		for _, m := range hfModels {
			caps := []string{}
			switch m.Pipeline {
			case "text-generation":
				caps = []string{"text-generation"}
			case "feature-extraction":
				caps = []string{"feature-extraction", "embeddings"}
			}
			models = append(models, ModelInfo{
				ID:           m.ID,
				Name:         m.ID,
				Capabilities: caps,
				FreeToUse:    provider.FreeToUse,
			})
		}
		features = Features{Embeddings: true}

	default:
		// For OpenAI-compatible providers, try /v1/models
		modelsURL := provider.Endpoint + "/v1/models"
		req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
		if err != nil {
			return nil, Features{}, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, Features{}, fmt.Errorf("failed to fetch models: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, Features{}, fmt.Errorf("API returned status %d", resp.StatusCode)
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
			return nil, Features{}, fmt.Errorf("failed to decode models: %w", err)
		}
		for _, m := range openaiResp.Data {
			models = append(models, ModelInfo{
				ID:           m.ID,
				Name:         m.ID,
				Capabilities: []string{"chat"}, // Assume chat for now
				FreeToUse:    provider.FreeToUse,
			})
		}
		// Set features based on provider knowledge
		switch providerName {
		case "nvidia":
			features = Features{Streaming: true, FunctionCalling: true, Vision: true}
		case "chutes":
			features = Features{Streaming: true, FunctionCalling: true, Vision: true}
		case "siliconflow":
			features = Features{Streaming: true, FunctionCalling: true}
		case "kimi":
			features = Features{Streaming: true, FunctionCalling: true}
		case "gemini":
			features = Features{Streaming: true, FunctionCalling: true, Vision: true, Tools: true}
		case "openrouter":
			features = Features{Streaming: true, Vision: true}
		case "z.ai":
			features = Features{Streaming: true}
		case "deepseek":
			features = Features{Streaming: true, FunctionCalling: true}
		default:
			features = Features{}
		}
	}

	return models, features, nil
}

func generateSummary(result ChallengeResult) ChallengeSummary {
	successCount := 0
	errorCount := 0
	skippedCount := 0
	totalModels := 0
	freeModels := 0

	for _, provider := range result.Providers {
		switch provider.Status {
		case "success":
			successCount++
			totalModels += len(provider.Models)
			for _, model := range provider.Models {
				if model.FreeToUse {
					freeModels++
				}
			}
		case "skipped":
			skippedCount++
		default:
			errorCount++
		}
	}

	return ChallengeSummary{
		TotalProviders: len(result.Providers),
		SuccessCount:   successCount,
		ErrorCount:     errorCount,
		SkippedCount:   skippedCount,
		TotalModels:    totalModels,
		FreeModels:     freeModels,
	}
}

func saveResults(resultsDir string, result ChallengeResult) error {
	os.MkdirAll(resultsDir, 0755)

	opencodeData := map[string]interface{}{
		"challenge_name": result.ChallengeName,
		"date":           time.Now().Format("2006-01-02"),
		"providers":      make([]interface{}, 0),
	}

	for _, provider := range result.Providers {
		providerMap := map[string]interface{}{
			"name":        provider.Name,
			"type":        provider.Type,
			"endpoint":    provider.Endpoint,
			"status":      provider.Status,
			"free_to_use": provider.FreeToUse,
			"features":    provider.Features,
			"models":      provider.Models,
		}
		opencodeData["providers"] = append(opencodeData["providers"].([]interface{}), providerMap)
	}

	opencodeBytes, err := json.MarshalIndent(opencodeData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode: %w", err)
	}

	if err := os.WriteFile(filepath.Join(resultsDir, "providers_opencode.json"), opencodeBytes, 0644); err != nil {
		return fmt.Errorf("failed to write opencode: %w", err)
	}

	crushBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal crush: %w", err)
	}

	if err := os.WriteFile(filepath.Join(resultsDir, "providers_crush.json"), crushBytes, 0644); err != nil {
		return fmt.Errorf("failed to write crush: %w", err)
	}

	loggerVerbose(1, "Results saved:")
	loggerVerbose(1, "  - providers_opencode.json")
	loggerVerbose(1, "  - providers_crush.json")

	return nil
}

func loggerVerbose(level int, format string, args ...interface{}) {
	if level <= verboseLevel {
		logger.Printf(format, args...)
	}
}

func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
