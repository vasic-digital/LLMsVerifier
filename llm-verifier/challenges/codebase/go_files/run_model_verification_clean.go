package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

"github.com/milosvasic/LLMsVerifier/llm-verifier/client"
)

type VerificationResult struct {
	ModelID       string   `json:"model_id"`
	ModelName     string   `json:"model_name"`
	ProviderName  string   `json:"provider_name"`
	Exists      bool     `json:"exists"`
	Responsive  bool     `json:"responsive"`
	StatusCode  int      `json:"status_code,omitempty"`
	OverallStatus string   `json:"overall_status"`
	Latency      string   `json:"latency"`
	TTFT         string   `json:"time_to_first_token"`
}

type ProviderResult struct {
	Name                string               `json:"name"`
	VerificationResults []VerificationResult `json:"verification_results"`
	SuccessCount        int                  `json:"success_count"`
	FailedCount         int                  `json:"failed_count"`
	SkippedCount       int                  `json:"skipped_count"`
}

type ChallengeResult struct {
	ChallengeName string   `json:"challenge_name"`
	Date          string   `json:"date"`
	Duration      string   `json:"duration"`
	Providers     []ProviderResult `json:"providers"`
	Summary       ChallengeSummary `json:"summary"`
}

type ChallengeSummary struct {
	TotalModels   int `json:"total_models"`
	ModelsExist  int `json:"models_exist"`
	ModelsResponsive int   `json:"models_responsive"`
	SuccessRate   float64 `json:"success_rate"`
}

var logger *log.Logger
var httpClient *client.HTTPClient

func initLogger(logDir string) *log.Logger {
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "challenge.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return log.New(os.Stdout, "[CHALLENGE] ", log.Ldate|log.Ltime|log.Lmicroseconds)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	return log.New(multiWriter, "[CHALLENGE] ", log.Ldate|log.Ltime|log.Lmicroseconds)
}

func main() {
	timestamp := time.Now().Unix()
	challengeDir := filepath.Join("challenges", "model_verification",
		time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"),
		fmt.Sprintf("%d", timestamp))
	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")

	logger = initLogger(logDir)
	httpClient = client.NewHTTPClient(30 * time.Second)

	logger.Println("======================================================")
	logger.Println("MODEL VERIFICATION CHALLENGE (SIMPLE - READY)")
	logger.Println("======================================================")
	logger.Printf("Challenge Directory: %s", challengeDir)
	logger.Printf("Timestamp: %s\n", time.Now().Format(time.RFC3339))

	providers := createSimulatedProviders()
	logger.Printf("Loaded %d providers with simulated data, starting verification\n", len(providers))

	startTime := time.Now()
	result := runChallenge(providers)

	result.Date = time.Now().Format("2006-01-02")
	result.Duration = time.Since(startTime).String()

	if err := saveResults(resultsDir, result); err != nil {
		logger.Printf("Failed to save results: %v\n", err)
	}

	logger.Println("======================================================")
	logger.Println("CHALLENGE COMPLETE")
	logger.Println("======================================================")
	logger.Printf("Duration: %s\n", result.Duration)
	logger.Printf("Results saved to: %s\n", resultsDir)
}

func runChallenge(providers []ProviderConfig) ChallengeResult {
	result := ChallengeResult{
			ChallengeName: "model_verification",
		Providers:     []ProviderResult{},
		Summary:       ChallengeSummary{},
	}

	for _, provider := range providers {
		logger.Printf("\n======================================================")
		logger.Printf("Verifying Provider: %s (%d models)\n", provider.Name, len(provider.Models))
		logger.Printf("======================================================")

			providerResult := verifyProvider(ctx, provider)
		result.Providers = append(result.Providers, providerResult)
	}

	result.Summary = generateSummary(result)
	return result
}

func verifyProvider(ctx context.Context, provider ProviderConfig) ProviderResult {
	startTime := time.Now()
	result := ProviderResult{
		Name:     provider.Name,
		VerificationResults: []VerificationResult{},
		TestTime: time.Now().Format(time.RFC3339),
	}

	for _, model := range provider.Models {
	verification := verifyModel(ctx, provider, model)
		result.VerificationResults = append(result.VerificationResults, verification)

		if verification.OverallStatus == "success" {
			result.SuccessCount++
		} else {
			result.FailedCount++
		}
	}

	duration := time.Since(startTime)
	logger.Printf("Provider %s: %d verified, %d failed (duration: %s)\n",
		provider.Name, result.SuccessCount, result.FailedCount, duration.String())

	return result
}

func verifyModel(ctx context.Context, provider ProviderConfig, model ModelConfig) VerificationResult {
	logger.Printf("--- Verifying Model: %s (%s)\n", model.Name, model.ID)

	return VerificationResult{
		ModelID:       model.ID,
		ModelName:     model.Name,
			ProviderName:  provider.Name,
		OverallStatus: "simulated_success",
		Latency:      "simulated_ms",
		TTFT:         "simulated",
		StatusCode:   200,
	}
}

func generateSummary(result ChallengeResult) ChallengeSummary {
	summary := ChallengeSummary{
		TotalModels:  0,
		ModelsExist: 0,
		ModelsResponsive: 0,
		SuccessRate:     0.0,
	}

	for _, provider := range result.Providers {
		for _, verification := range provider.VerificationResults {
			summary.TotalModels++
			if verification.Exists {
				summary.ModelsExist++
			}
			if verification.Responsive {
				summary.ModelsResponsive++
			}
		}
	}

	if summary.TotalModels > 0 {
			summary.SuccessRate = float64(summary.ModelsResponsive) / float64(summary.TotalModels) * 100
	}

	return summary
}

func createSimulatedProviders() []ProviderConfig {
	providers := []ProviderConfig{
		{
			Name: "huggingface",
			Type:  "openai-compatible",
			Models: []ModelConfig{
				{ID: "sentence-transformers/all-MiniLM-L6-v2", Name: "Sentence Transformers MiniLM L6 v2", FreeToUse: true},
				{ID: "Falconsai/nsfw_image_detection", Name: "NSFW Image Detection", FreeToUse: true},
			},
		},
	},
		{
			Name: "deepseek",
			Type: "openai-compatible",
			Models: []ModelConfig{
				{ID: "deepseek-chat", Name: "DeepSeek Chat", FreeToUse: false},
				{ID: "deepseek-coder", Name: "DeepSeek Coder", FreeToUse: false},
			},
		},
	}

	return providers
}

func saveResults(resultsDir string, result ChallengeResult) error {
	os.MkdirAll(resultsDir, 0755)

	resultFile := filepath.Join(resultsDir, "verification_results.json")
	resultData, err := json.MarshalIndent(result, "", "  ")

	if err := os.WriteFile(resultFile, resultData, 0644); err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}

	summaryFile := filepath.Join(resultsDir, "summary.md")
	summary := fmt.Sprintf("# Model Verification Challenge Results (Simple Version)\n\n")
	summary += fmt.Sprintf("**Date**: %s\n\n", result.Date)
	summary += fmt.Sprintf("**Duration**: %s\n", result.Duration)
	summary += "## Providers\n\n"
	for _, provider := range result.Providers {
		summary += fmt.Sprintf("### %s\n", provider.Name)
		summary += fmt.Sprintf("- **Status**: %d tested, %d failed\n",
			provider.SuccessCount+provider.FailedCount+provider.SkippedCount,
			provider.SuccessCount, provider.FailedCount, provider.SkippedCount)
	}
	summary += "\n## Summary\n\n"
	summary += fmt.Sprintf("- **Total Models**: %d\n", result.Summary.TotalModels)
	summary += fmt.Sprintf("- **Models Exist**: %d\n", result.Summary.ModelsExist)
	summary += fmt.Sprintf("- **Models Responsive**: %d\n", result.Summary.ModelsResponsive)
	summary += fmt.Sprintf("- **Success Rate**: %.1f%%\n", result.Summary.SuccessRate)

	if err := os.WriteFile(summaryFile, []byte(summary), 064); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}
