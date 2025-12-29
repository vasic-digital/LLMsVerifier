package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"llm-verifier/client"
	"llm-verifier/database"
	"llm-verifier/logging"
	"llm-verifier/providers"
	"llm-verifier/verification"
)

// Configuration for the verification run
type VerificationConfig struct {
	ProviderFilter []string `json:"provider_filter"`
	ModelFilter    []string `json:"model_filter"`
	MaxConcurrency int      `json:"max_concurrency"`
	Timeout        int      `json:"timeout_seconds"`
	OutputFormat   string   `json:"output_format"` // json, csv, markdown
}

// VerificationReport represents the complete verification report
type VerificationReport struct {
	Timestamp      time.Time                         `json:"timestamp"`
	TotalModels    int                               `json:"total_models"`
	VerifiedModels int                               `json:"verified_models"`
	FailedModels   int                               `json:"failed_models"`
	ErrorModels    int                               `json:"error_models"`
	AverageScore   float64                           `json:"average_score"`
	Results        []verification.VerificationResult `json:"results"`
	Summary        VerificationSummary               `json:"summary"`
}

// VerificationSummary provides a summary of the verification run
type VerificationSummary struct {
	ByProvider map[string]ProviderSummary `json:"by_provider"`
	ByStatus   map[string]int             `json:"by_status"`
	TopScoring []ModelScore               `json:"top_scoring"`
	LowScoring []ModelScore               `json:"low_scoring"`
}

// ProviderSummary summarizes verification results by provider
type ProviderSummary struct {
	TotalModels    int     `json:"total_models"`
	VerifiedModels int     `json:"verified_models"`
	FailedModels   int     `json:"failed_models"`
	AverageScore   float64 `json:"average_score"`
}

// ModelScore represents a model's verification score
type ModelScore struct {
	ModelID           string  `json:"model_id"`
	ProviderID        string  `json:"provider_id"`
	Status            string  `json:"status"`
	VerificationScore float64 `json:"verification_score"`
}

func main() {
	var (
		configFile   = flag.String("config", "", "Path to configuration file")
		outputDir    = flag.String("output", "verification_results", "Output directory for results")
		providerFlag = flag.String("providers", "", "Comma-separated list of providers to verify")
		modelFlag    = flag.String("models", "", "Comma-separated list of models to verify")
		concurrency  = flag.Int("concurrency", 5, "Maximum number of concurrent verifications")
		timeout      = flag.Int("timeout", 60, "Timeout in seconds for each verification")
		format       = flag.String("format", "json", "Output format: json, csv, markdown")
		dbPath       = flag.String("db", "../llm-verifier.db", "Database path")
		help         = flag.Bool("help", false, "Show help information")
	)

	flag.Parse()

	if *help {
		printHelp()
		return
	}

	// Load configuration
	config := loadConfig(*configFile, *providerFlag, *modelFlag, *concurrency, *timeout, *format)

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Set up logging
	loggerConfig := map[string]any{
		"console_level": "info",
		"file_level":    "debug",
		"component":     "code_verification",
	}

	// Initialize database
	db, err := database.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize logger
	logger, err := logging.NewLogger(db, loggerConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	logger.Info("Starting mandatory code verification process", map[string]interface{}{
		"config": config,
	})

	// Initialize services
	httpClient := client.NewHTTPClient(time.Duration(config.Timeout) * time.Second)
	providerService := providers.NewModelProviderService("config.yaml", logger)

	// Register all providers
	providerService.RegisterAllProviders()

	// Create verification service
	verificationService := verification.NewCodeVerificationService(httpClient, logger)

	// Create integration with adapter
	providerAdapter := providers.NewProviderServiceAdapter(providerService)
	integration := verification.NewCodeVerificationIntegration(verificationService, db, logger, providerAdapter)

	// Run verification
	ctx := context.Background()
	results, err := runVerification(ctx, integration, config, logger)
	if err != nil {
		log.Fatalf("Verification failed: %v", err)
	}

	// Generate report
	report := generateReport(results)

	// Save results
	if err := saveResults(report, *outputDir, config.OutputFormat); err != nil {
		log.Fatalf("Failed to save results: %v", err)
	}

	// Print summary
	printSummary(report)

	logger.Info("Code verification completed successfully", map[string]interface{}{
		"total_models":    report.TotalModels,
		"verified_models": report.VerifiedModels,
		"failed_models":   report.FailedModels,
		"average_score":   report.AverageScore,
	})
}

func printHelp() {
	fmt.Println("LLM Verifier - Mandatory Code Verification Tool")
	fmt.Println()
	fmt.Println("This tool verifies that coding models can actually see and process code with tooling support.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  code-verification [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -config string       Path to configuration file")
	fmt.Println("  -output string       Output directory for results (default \"verification_results\")")
	fmt.Println("  -providers string    Comma-separated list of providers to verify")
	fmt.Println("  -models string       Comma-separated list of models to verify")
	fmt.Println("  -concurrency int     Maximum number of concurrent verifications (default 5)")
	fmt.Println("  -timeout int         Timeout in seconds for each verification (default 60)")
	fmt.Println("  -format string       Output format: json, csv, markdown (default \"json\")")
	fmt.Println("  -db string           Database path (default \"../llm-verifier.db\")")
	fmt.Println("  -help                Show this help information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Verify all models from all providers")
	fmt.Println("  code-verification")
	fmt.Println()
	fmt.Println("  # Verify only OpenAI and Anthropic models")
	fmt.Println("  code-verification -providers openai,anthropic")
	fmt.Println()
	fmt.Println("  # Verify specific models")
	fmt.Println("  code-verification -models gpt-4,claude-3.5-sonnet")
	fmt.Println()
	fmt.Println("  # Custom output directory and format")
	fmt.Println("  code-verification -output ./results -format markdown")
}

func loadConfig(configFile, providers, models string, concurrency, timeout int, format string) VerificationConfig {
	config := VerificationConfig{
		MaxConcurrency: concurrency,
		Timeout:        timeout,
		OutputFormat:   format,
	}

	// Load from file if provided
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Printf("Warning: Failed to read config file: %v", err)
		} else {
			if err := json.Unmarshal(data, &config); err != nil {
				log.Printf("Warning: Failed to parse config file: %v", err)
			}
		}
	}

	// Override with command line flags
	if providers != "" {
		config.ProviderFilter = strings.Split(providers, ",")
	}
	if models != "" {
		config.ModelFilter = strings.Split(models, ",")
	}

	return config
}

func runVerification(ctx context.Context, integration *verification.CodeVerificationIntegration, config VerificationConfig, logger *logging.Logger) ([]verification.VerificationResult, error) {
	logger.Info("Starting verification process", map[string]interface{}{
		"provider_filter": config.ProviderFilter,
		"model_filter":    config.ModelFilter,
		"concurrency":     config.MaxConcurrency,
		"timeout":         config.Timeout,
	})

	// Get all verification results
	results, err := integration.VerifyAllModelsWithCodeSupport(ctx)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	// Filter results based on configuration
	filteredResults := filterResults(results, config)

	logger.Info(fmt.Sprintf("Verification completed for %d models", len(filteredResults)), map[string]interface{}{
		"total_results": len(filteredResults),
	})

	return filteredResults, nil
}

func filterResults(results []verification.VerificationResult, config VerificationConfig) []verification.VerificationResult {
	if len(config.ProviderFilter) == 0 && len(config.ModelFilter) == 0 {
		return results
	}

	var filtered []verification.VerificationResult

	for _, result := range results {
		// Filter by provider
		if len(config.ProviderFilter) > 0 {
			found := false
			for _, provider := range config.ProviderFilter {
				if strings.EqualFold(result.ProviderID, provider) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by model
		if len(config.ModelFilter) > 0 {
			found := false
			for _, model := range config.ModelFilter {
				if strings.EqualFold(result.ModelID, model) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, result)
	}

	return filtered
}

func generateReport(results []verification.VerificationResult) VerificationReport {
	byProvider := make(map[string]ProviderSummary)
	byStatus := make(map[string]int)

	var totalScore float64
	var verifiedCount, failedCount, errorCount int

	for _, result := range results {
		// Count by status
		switch result.Status {
		case "verified":
			verifiedCount++
			byStatus["verified"]++
		case "failed":
			failedCount++
			byStatus["failed"]++
		case "error":
			errorCount++
			byStatus["error"]++
		}

		// Sum scores
		totalScore += result.VerificationScore

		// Group by provider
		summary, exists := byProvider[result.ProviderID]
		if !exists {
			summary = ProviderSummary{}
		}
		summary.TotalModels++
		if result.Status == "verified" {
			summary.VerifiedModels++
		} else if result.Status == "failed" {
			summary.FailedModels++
		}
		byProvider[result.ProviderID] = summary
	}

	// Calculate provider averages
	for provider, summary := range byProvider {
		if summary.TotalModels > 0 {
			summary.AverageScore = float64(summary.VerifiedModels) / float64(summary.TotalModels) * 10
			byProvider[provider] = summary
		}
	}

	report := VerificationReport{
		Timestamp:      time.Now(),
		TotalModels:    len(results),
		VerifiedModels: verifiedCount,
		FailedModels:   failedCount,
		ErrorModels:    errorCount,
		Results:        results,
		Summary: VerificationSummary{
			ByProvider: byProvider,
			ByStatus:   byStatus,
			TopScoring: getTopScoringModels(results, 5),
			LowScoring: getLowScoringModels(results, 5),
		},
	}

	if report.TotalModels > 0 {
		report.AverageScore = totalScore / float64(report.TotalModels)
	}

	return report
}

func getTopScoringModels(results []verification.VerificationResult, count int) []ModelScore {
	var scores []ModelScore
	for _, result := range results {
		if result.Status == "verified" {
			scores = append(scores, ModelScore{
				ModelID:           result.ModelID,
				ProviderID:        result.ProviderID,
				Status:            result.Status,
				VerificationScore: result.VerificationScore,
			})
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].VerificationScore > scores[i].VerificationScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	if len(scores) > count {
		return scores[:count]
	}
	return scores
}

func getLowScoringModels(results []verification.VerificationResult, count int) []ModelScore {
	var scores []ModelScore
	for _, result := range results {
		if result.Status == "failed" || result.Status == "error" {
			scores = append(scores, ModelScore{
				ModelID:           result.ModelID,
				ProviderID:        result.ProviderID,
				Status:            result.Status,
				VerificationScore: result.VerificationScore,
			})
		}
	}

	// Sort by score (ascending)
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].VerificationScore < scores[i].VerificationScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	if len(scores) > count {
		return scores[:count]
	}
	return scores
}

func saveResults(report VerificationReport, outputDir, format string) error {
	timestamp := time.Now().Format("20060102_150405")

	switch format {
	case "json":
		return saveJSONResults(report, outputDir, timestamp)
	case "csv":
		return saveCSVResults(report, outputDir, timestamp)
	case "markdown":
		return saveMarkdownResults(report, outputDir, timestamp)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func saveJSONResults(report VerificationReport, outputDir, timestamp string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("code_verification_report_%s.json", timestamp))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Results saved to: %s", filename)
	return nil
}

func saveCSVResults(report VerificationReport, outputDir, timestamp string) error {
	var csv strings.Builder
	csv.WriteString("Provider,Model,Status,VerificationScore,CodeVisibility,ToolSupport,VerifiedAt\n")

	for _, result := range report.Results {
		csv.WriteString(fmt.Sprintf("%s,%s,%s,%.2f,%t,%t,%s\n",
			result.ProviderID,
			result.ModelID,
			result.Status,
			result.VerificationScore,
			result.CodeVisibility,
			result.ToolSupport,
			result.VerifiedAt.Format(time.RFC3339),
		))
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("code_verification_results_%s.csv", timestamp))
	if err := os.WriteFile(filename, []byte(csv.String()), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Results saved to: %s", filename)
	return nil
}

func saveMarkdownResults(report VerificationReport, outputDir, timestamp string) error {
	var md strings.Builder

	md.WriteString("# Code Verification Report\n\n")
	md.WriteString(fmt.Sprintf("**Generated:** %s  \n", report.Timestamp.Format(time.RFC3339)))
	md.WriteString(fmt.Sprintf("**Total Models:** %d  \n", report.TotalModels))
	md.WriteString(fmt.Sprintf("**Verified Models:** %d  \n", report.VerifiedModels))
	md.WriteString(fmt.Sprintf("**Failed Models:** %d  \n", report.FailedModels))
	md.WriteString(fmt.Sprintf("**Error Models:** %d  \n", report.ErrorModels))
	md.WriteString(fmt.Sprintf("**Average Score:** %.2f  \n\n", report.AverageScore))

	md.WriteString("## Summary by Provider\n\n")
	md.WriteString("| Provider | Total | Verified | Failed | Average Score |\n")
	md.WriteString("|----------|-------|----------|--------|---------------|\n")

	for provider, summary := range report.Summary.ByProvider {
		md.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %.2f |\n",
			provider, summary.TotalModels, summary.VerifiedModels, summary.FailedModels, summary.AverageScore))
	}

	md.WriteString("\n## Detailed Results\n\n")
	md.WriteString("| Provider | Model | Status | Score | Code Visibility | Tool Support |\n")
	md.WriteString("|----------|-------|--------|-------|-----------------|--------------|\n")

	for _, result := range report.Results {
		md.WriteString(fmt.Sprintf("| %s | %s | %s | %.2f | %t | %t |\n",
			result.ProviderID,
			result.ModelID,
			result.Status,
			result.VerificationScore,
			result.CodeVisibility,
			result.ToolSupport,
		))
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("code_verification_report_%s.md", timestamp))
	if err := os.WriteFile(filename, []byte(md.String()), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Results saved to: %s", filename)
	return nil
}

func printSummary(report VerificationReport) {
	fmt.Println("\n=== Code Verification Summary ===")
	fmt.Printf("Total Models: %d\n", report.TotalModels)
	fmt.Printf("Verified Models: %d\n", report.VerifiedModels)
	fmt.Printf("Failed Models: %d\n", report.FailedModels)
	fmt.Printf("Error Models: %d\n", report.ErrorModels)
	fmt.Printf("Average Score: %.2f\n", report.AverageScore)
	fmt.Printf("Success Rate: %.1f%%\n", float64(report.VerifiedModels)/float64(report.TotalModels)*100)
}
