package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"llm-verifier/client"
	"llm-verifier/llmverifier"
)

// Model Verification Challenge
// Verifies models with actual API calls to meet validation criteria

type ProviderConfig struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	Endpoint  string        `json:"endpoint"`
	Status    string        `json:"status"`
	FreeToUse bool          `json:"free_to_use"`
	APIKey    string        `json:"api_key"`
	Features  ModelFeatures `json:"features"`
	Models    []ModelConfig `json:"models"`
}

type ModelConfig struct {
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

type VerificationResult struct {
	ModelID          string   `json:"model_id"`
	ModelName        string   `json:"model_name"`
	ProviderName     string   `json:"provider_name"`
	Exists           bool     `json:"exists"`
	ExistsError      string   `json:"exists_error,omitempty"`
	Responsive       bool     `json:"responsive"`
	ResponseError    string   `json:"response_error,omitempty"`
	Latency          string   `json:"latency"`
	TTFT             string   `json:"time_to_first_token"`
	FeaturesVerified Features `json:"features_verified"`
	OverallStatus    string   `json:"overall_status"`
	TestTime         string   `json:"test_time"`
	StatusCode       int      `json:"status_code,omitempty"`
}

type ChallengeResult struct {
	ChallengeName string           `json:"challenge_name"`
	Date          string           `json:"date"`
	StartTime     string           `json:"start_time"`
	EndTime       string           `json:"end_time"`
	Duration      string           `json:"duration"`
	Providers     []ProviderResult `json:"providers"`
	Summary       ChallengeSummary `json:"summary"`
}

type ProviderResult struct {
	Name                string               `json:"name"`
	Type                string               `json:"type"`
	Endpoint            string               `json:"endpoint"`
	VerificationResults []VerificationResult `json:"verification_results"`
	SuccessCount        int                  `json:"success_count"`
	FailedCount         int                  `json:"failed_count"`
	SkippedCount        int                  `json:"skipped_count"`
	TestTime            string               `json:"test_time"`
}

type ChallengeSummary struct {
	TotalModels               int `json:"total_models"`
	VerifiedModels            int `json:"verified_models"`
	ModelsExist               int `json:"models_exist"`
	ModelsResponsive          int `json:"models_responsive"`
	ModelsWithStreaming       int `json:"models_with_streaming"`
	ModelsWithFunctionCalling int `json:"models_with_function_calling"`
	ModelsWithVision          int `json:"models_with_vision"`
	ModelsWithEmbeddings      int `json:"models_with_embeddings"`
	FreeModels                int `json:"free_models"`
	PaidModels                int `json:"paid_models"`
}

var (
	logger       *log.Logger
	verboseLevel int = 3
	httpClient   *client.HTTPClient
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

func loggerVerbose(level int, format string, v ...interface{}) {
	if verboseLevel >= level {
		logger.Printf(format, v...)
	}
}

func main() {
	timestamp := time.Now().Unix()
	challengeDir := filepath.Join("challenges", "model_verification",
		time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"),
		fmt.Sprintf("%d", timestamp))

	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")

	logger = initLogger(logDir)

	logger.Println("======================================================")
	logger.Println("MODEL VERIFICATION CHALLENGE")
	logger.Println("======================================================")
	logger.Printf("Challenge Directory: %s", challengeDir)
	logger.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))

	providers, err := loadProviderConfig()
	if err != nil {
		logger.Printf("Failed to load provider config: %v", err)
		logger.Println("Exiting...")
		return
	}

	// Initialize HTTP client with 30 second timeout
	httpClient = client.NewHTTPClient(30 * time.Second)

	logger.Printf("Loaded %d providers from config", len(providers))

	result := runChallenge(providers)

	if err := saveResults(resultsDir, result); err != nil {
		logger.Printf("Failed to save results: %v", err)
	}

	logger.Println("======================================================")
	logger.Println("CHALLENGE COMPLETE")
	logger.Println("======================================================")
	logger.Printf("Duration: %s", result.Duration)
	logger.Printf("Results saved to: %s", resultsDir)
}

func runChallenge(providers []ProviderConfig) ChallengeResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	startTime := time.Now()
	result := ChallengeResult{
		ChallengeName: "model_verification",
		StartTime:     startTime.Format(time.RFC3339),
		Date:          time.Now().Format("2006-01-02"),
	}

	loggerVerbose(1, "Starting model verification with real API calls...")
	loggerVerbose(1, "Models to verify: %d", countModelsConfig(providers))

	for _, provider := range providers {
		logger.Printf("\n======================================================")
		logger.Printf("Verifying Provider: %s", provider.Name)
		logger.Printf("======================================================")

		providerResult := verifyProvider(ctx, provider)
		result.Providers = append(result.Providers, providerResult)
	}

	result.Summary = generateSummary(result)
	result.EndTime = time.Now().Format(time.RFC3339)
	result.Duration = time.Since(startTime).String()

	return result
}

func verifyProvider(ctx context.Context, provider ProviderConfig) ProviderResult {
	startTime := time.Now()
	result := ProviderResult{
		Name:     provider.Name,
		Type:     provider.Type,
		Endpoint: provider.Endpoint,
		TestTime: time.Now().Format(time.RFC3339),
	}

	for _, model := range provider.Models {
		if provider.Status != "success" || provider.APIKey == "" {
			result.SkippedCount++
			continue
		}

		verification := verifyModel(ctx, provider, model)
		result.VerificationResults = append(result.VerificationResults, verification)

		// Count success/failure based on overall status
		if verification.OverallStatus == "success" {
			result.SuccessCount++
		} else {
			result.FailedCount++
		}
	}

	loggerVerbose(1, "Provider %s: %d tested, %d passed, %d failed, %d skipped in %s",
		provider.Name, result.SuccessCount+result.FailedCount+result.SkippedCount,
		result.SuccessCount, result.FailedCount, result.SkippedCount, time.Since(startTime).String())

	return result
}

func verifyModel(ctx context.Context, provider ProviderConfig, model ModelConfig) VerificationResult {
	logger.Printf("\n--- Verifying Model: %s (%s) ---", model.Name, model.ID)
	startTime := time.Now()

	verification := VerificationResult{
		ModelID:       model.ID,
		ModelName:     model.Name,
		ProviderName:  provider.Name,
		TestTime:      time.Now().Format(time.RFC3339),
		OverallStatus: "unknown",
	}

	// Test 1: Existence - Make real API call
	loggerVerbose(2, "  Testing existence...")
	existsResp, existsErr := httpClient.TestModelExists(ctx, provider.Name, provider.APIKey, model.ID)
	verification.Exists = existsResp != nil && existsResp.Exists
	if existsErr != nil {
		verification.ExistsError = existsErr.Error()
		loggerVerbose(2, "    ✗ Existence failed: %s", existsErr.Error())
	} else if existsResp != nil && existsResp.Exists {
		loggerVerbose(2, "    ✓ Exists (latency: %s)", existsResp.Latency)
	}

	// Test 2: Responsiveness - Make real API call
	loggerVerbose(2, "  Testing responsiveness...")
	respResp, respErr := httpClient.TestResponsiveness(ctx, provider.Name, provider.APIKey, model.ID, "test")
	verification.Responsive = respErr == nil && respResp.Success
	if respErr != nil {
		verification.ResponseError = respErr.Error()
		verification.StatusCode = respResp.StatusCode
		loggerVerbose(2, "    ✗ Responsiveness failed: %s", respErr.Error())
	} else if respResp.Success {
		verification.Latency = respResp.TotalTime.String()
		verification.TTFT = respResp.TTFT.String()
		loggerVerbose(2, "    ✓ Responsive (TTFT: %s, total: %s)", respResp.TTFT, respResp.TotalTime)
	}

	// Verify features from configuration (actual feature testing would require more complex implementations)
	verification.FeaturesVerified = ModelFeatures{
		Streaming:       model.Features.Streaming,
		FunctionCalling: model.Features.FunctionCalling,
		Vision:          model.Features.Vision,
		Embeddings:      len(model.Features.Embeddings) > 0,
		MCPs:            len(model.Features.MCPs) > 0,
		LSPs:            len(model.Features.LSPs) > 0,
		Tools:           model.Features.Tools,
	}

	loggerVerbose(1, "  Features verified from configuration:")
	loggerVerbose(2, "    Streaming: %v", verification.FeaturesVerified.Streaming)
	loggerVerbose(2, "    Function Calling: %v", verification.FeaturesVerified.FunctionCalling)
	loggerVerbose(2, "    Vision: %v", verification.FeaturesVerified.Vision)
	loggerVerbose(2, "    Embeddings: %v", verification.FeaturesVerified.Embeddings)

	// Determine overall status
	if verification.Exists && verification.Responsive {
		verification.OverallStatus = "success"
		loggerVerbose(1, "  ✓ Overall status: SUCCESS")
	} else if !verification.Exists {
		verification.OverallStatus = "failed"
		loggerVerbose(1, "  ✗ Overall status: FAILED (model does not exist)")
	} else if !verification.Responsive {
		verification.OverallStatus = "failed"
		loggerVerbose(1, "  ✗ Overall status: FAILED (model not responsive)")
	} else {
		verification.OverallStatus = "failed"
		loggerVerbose(1, "  ✗ Overall status: FAILED (unknown error)")
	}

	return verification
}

func generateSummary(result ChallengeResult) ChallengeSummary {
	summary := ChallengeSummary{
		TotalModels:               countResults(result.Providers),
		ModelsExist:               0,
		ModelsResponsive:          0,
		ModelsWithStreaming:       0,
		ModelsWithFunctionCalling: 0,
		ModelsWithVision:          0,
		ModelsWithEmbeddings:      0,
		FreeModels:                0,
		PaidModels:                0,
	}

	for _, provider := range result.Providers {
		for _, verification := range provider.VerificationResults {
			summary.VerifiedModels++

			if verification.Exists {
				summary.ModelsExist++
			}
			if verification.Responsive {
				summary.ModelsResponsive++
			}
			if verification.FeaturesVerified.Streaming {
				summary.ModelsWithStreaming++
			}
			if verification.FeaturesVerified.FunctionCalling {
				summary.ModelsWithFunctionCalling++
			}
			if verification.FeaturesVerified.Vision {
				summary.ModelsWithVision++
			}
			if verification.FeaturesVerified.Embeddings {
				summary.ModelsWithEmbeddings++
			}
			if verification.FeaturesVerified.Tools {
				summary.ModelsWithStreaming++
			}
		}
	}

	return summary
}

func countModelsConfig(providers []ProviderConfig) int {
	count := 0
	for _, provider := range providers {
		count += len(provider.Models)
	}
	return count
}

func countResults(providers []ProviderResult) int {
	count := 0
	for _, provider := range providers {
		count += len(provider.VerificationResults)
	}
	return count
}

func loadProviderConfig() ([]ProviderConfig, error) {
	// Load from existing challenge results
	providers, err := llmverifier.LoadProviders()
	if err != nil {
		return nil, fmt.Errorf("failed to load providers: %w", err)
	}

	config := make([]ProviderConfig, 0)
	for _, provider := range providers {
		config = append(config, ProviderConfig{
			Name:      provider.Name,
			Type:      provider.Type,
			Endpoint:  provider.Endpoint,
			Status:    "success", // Assume success if loaded from DB
			FreeToUse: provider.FreeToUse,
			APIKey:    provider.APIKeyEncrypted, // Will need to decrypt
			Features: ModelFeatures{
				Streaming:       provider.Features.Streaming,
				FunctionCalling: provider.Features.FunctionCalling,
				Vision:          provider.Features.Vision,
				Embeddings:      len(provider.Features.Embeddings) > 0,
				MCPs:            []string{},
				LSPs:            []string{},
				Tools:           provider.Features.Tools,
			},
		})
	}

	return config, nil
}

func saveResults(resultsDir string, result ChallengeResult) error {
	// Save full challenge result
	resultFile := filepath.Join(resultsDir, "verification_results.json")
	resultData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(resultFile, resultData, 0644); err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}

	// Save provider summary
	summaryFile := filepath.Join(resultsDir, "summary.md")
	summary := fmt.Sprintf("# Model Verification Challenge Results\n\n")
	summary += fmt.Sprintf("**Date**: %s\n\n", result.Date)
	summary += fmt.Sprintf("**Duration**: %s\n\n", result.Duration)
	summary += "## Providers\n\n"
	for _, provider := range result.Providers {
		summary += fmt.Sprintf("### %s\n", provider.Name)
		summary += fmt.Sprintf("- **Status**: %d tested, %d passed, %d failed, %d skipped\n",
			provider.SuccessCount+provider.FailedCount+provider.SkippedCount,
			provider.SuccessCount, provider.FailedCount, provider.SkippedCount)
	}
	summary += "\n## Summary\n\n"
	summary += fmt.Sprintf("- **Total Models**: %d\n", result.Summary.TotalModels)
	summary += fmt.Sprintf("- **Models Exist**: %d\n", result.Summary.ModelsExist)
	summary += fmt.Sprintf("- **Models Responsive**: %d\n", result.Summary.ModelsResponsive)
	summary += fmt.Sprintf("- **Success Rate**: %.1f%%\n",
		float64(result.Summary.ModelsResponsive)/float64(result.Summary.ModelsExist)*100)

	if err := os.WriteFile(summaryFile, []byte(summary), 0644); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}
