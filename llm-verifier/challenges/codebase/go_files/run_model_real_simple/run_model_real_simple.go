// Package main implements the Model Real Simple challenge.
// This challenge performs simple, fast model testing with basic prompts
// to quickly verify model availability and basic functionality.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ProviderConfig holds configuration for an LLM provider.
type ProviderConfig struct {
	Name    string `json:"name"`
	APIKey  string `json:"-"` // Never serialize
	BaseURL string `json:"base_url,omitempty"`
	Enabled bool   `json:"enabled"`
}

// SimpleTestResult holds the result of a simple model test.
type SimpleTestResult struct {
	ModelID      string    `json:"model_id"`
	Provider     string    `json:"provider"`
	Success      bool      `json:"success"`
	Response     string    `json:"response"`
	ResponseLen  int       `json:"response_length"`
	LatencyMs    int64     `json:"latency_ms"`
	HasContent   bool      `json:"has_content"`
	IsCoherent   bool      `json:"is_coherent"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ProviderTestResult holds all test results for a provider.
type ProviderTestResult struct {
	Provider       string             `json:"provider"`
	Success        bool               `json:"success"`
	ModelsTested   int                `json:"models_tested"`
	ModelsPassed   int                `json:"models_passed"`
	Results        []SimpleTestResult `json:"results"`
	AverageLatency float64            `json:"average_latency_ms"`
	ResponseTimeMs int64              `json:"response_time_ms"`
	Error          string             `json:"error,omitempty"`
	Timestamp      time.Time          `json:"timestamp"`
}

// ChallengeResult holds the complete challenge output.
type ChallengeResult struct {
	ChallengeID   string               `json:"challenge_id"`
	ChallengeName string               `json:"challenge_name"`
	Timestamp     time.Time            `json:"timestamp"`
	Duration      time.Duration        `json:"duration"`
	Status        string               `json:"status"`
	Results       []ProviderTestResult `json:"results"`
	AllTests      []SimpleTestResult   `json:"all_tests"`
	Summary       ChallengeSummary     `json:"summary"`
}

// ChallengeSummary provides aggregated statistics.
type ChallengeSummary struct {
	TotalProviders      int     `json:"total_providers"`
	SuccessfulProviders int     `json:"successful_providers"`
	FailedProviders     int     `json:"failed_providers"`
	TotalModels         int     `json:"total_models"`
	PassedModels        int     `json:"passed_models"`
	FailedModels        int     `json:"failed_models"`
	AverageLatency      float64 `json:"average_latency_ms"`
	FastestModel        string  `json:"fastest_model"`
	FastestLatency      int64   `json:"fastest_latency_ms"`
}

// API structures
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

// Anthropic structures
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// Gemini structures
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

type GeminiGenerationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// Simple test prompt
const simplePrompt = "Say 'Hello, I am working correctly!' in exactly those words."

// Known models per provider (using fast/cheap models)
var knownModels = map[string][]string{
	"openai":     {"gpt-4o-mini"},
	"anthropic":  {"claude-3-haiku-20240307"},
	"openrouter": {"openai/gpt-4o-mini"},
	"deepseek":   {"deepseek-chat"},
	"gemini":     {"gemini-1.5-flash"},
	"groq":       {"llama-3.1-8b-instant"},
	"ollama":     {"llama3"},
}

func sendSimplePrompt(ctx context.Context, config ProviderConfig, modelID string) (string, error) {
	switch config.Name {
	case "anthropic":
		return sendAnthropicSimple(ctx, config, modelID)
	case "gemini":
		return sendGeminiSimple(ctx, config, modelID)
	case "ollama":
		return sendOllamaSimple(ctx, config, modelID)
	default:
		return sendOpenAICompatibleSimple(ctx, config, modelID)
	}
}

func sendOpenAICompatibleSimple(ctx context.Context, config ProviderConfig, modelID string) (string, error) {
	baseURL := getBaseURL(config)

	reqBody := ChatCompletionRequest{
		Model:       modelID,
		Messages:    []ChatMessage{{Role: "user", Content: simplePrompt}},
		MaxTokens:   50,
		Temperature: 0,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func sendAnthropicSimple(ctx context.Context, config ProviderConfig, modelID string) (string, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	reqBody := AnthropicRequest{
		Model:     modelID,
		MaxTokens: 50,
		Messages:  []AnthropicMessage{{Role: "user", Content: simplePrompt}},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return anthropicResp.Content[0].Text, nil
}

func sendGeminiSimple(ctx context.Context, config ProviderConfig, modelID string) (string, error) {
	baseURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelID, config.APIKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: simplePrompt}}},
		},
		GenerationConfig: GeminiGenerationConfig{
			MaxOutputTokens: 50,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func sendOllamaSimple(ctx context.Context, config ProviderConfig, modelID string) (string, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	type OllamaRequest struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
	}

	type OllamaResponse struct {
		Response string `json:"response"`
	}

	reqBody := OllamaRequest{
		Model:  modelID,
		Prompt: simplePrompt,
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Response, nil
}

func getBaseURL(config ProviderConfig) string {
	if config.BaseURL != "" {
		return config.BaseURL
	}

	switch config.Name {
	case "openai":
		return "https://api.openai.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "deepseek":
		return "https://api.deepseek.com"
	case "groq":
		return "https://api.groq.com/openai/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

func testModel(ctx context.Context, config ProviderConfig, modelID string) *SimpleTestResult {
	result := &SimpleTestResult{
		ModelID:   modelID,
		Provider:  config.Name,
		Timestamp: time.Now(),
	}

	start := time.Now()
	response, err := sendSimplePrompt(ctx, config, modelID)
	result.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Response = response
	result.ResponseLen = len(response)

	// Check basic quality
	result.HasContent = len(strings.TrimSpace(response)) > 0

	// Check if response is coherent (contains expected keywords)
	responseLower := strings.ToLower(response)
	hasHello := strings.Contains(responseLower, "hello")
	hasWorking := strings.Contains(responseLower, "working")
	result.IsCoherent = hasHello || hasWorking || len(response) > 10

	return result
}

func testProvider(ctx context.Context, config ProviderConfig) *ProviderTestResult {
	result := &ProviderTestResult{
		Provider:  config.Name,
		Timestamp: time.Now(),
	}

	if config.APIKey == "" && config.Name != "ollama" {
		result.Error = "API key not configured"
		return result
	}

	start := time.Now()

	models, exists := knownModels[config.Name]
	if !exists || len(models) == 0 {
		result.Error = "No known models for provider"
		return result
	}

	var testResults []SimpleTestResult
	var totalLatency int64
	passedCount := 0

	// Test first model only
	modelID := models[0]
	testResult := testModel(ctx, config, modelID)
	testResults = append(testResults, *testResult)

	if testResult.Success && testResult.HasContent {
		passedCount++
		totalLatency += testResult.LatencyMs
	}

	result.ResponseTimeMs = time.Since(start).Milliseconds()
	result.Results = testResults
	result.ModelsTested = len(testResults)
	result.ModelsPassed = passedCount

	if passedCount > 0 {
		result.AverageLatency = float64(totalLatency) / float64(passedCount)
		result.Success = true
	}

	return result
}

func main() {
	resultsDir := flag.String("results-dir", "", "Directory to store results")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	if *resultsDir == "" {
		log.Fatal("--results-dir is required")
	}

	ctx := context.Background()
	start := time.Now()

	// Create results directory
	resultsPath := filepath.Join(*resultsDir, "results")
	logsPath := filepath.Join(*resultsDir, "logs")
	if err := os.MkdirAll(resultsPath, 0755); err != nil {
		log.Fatalf("Failed to create results directory: %v", err)
	}
	if err := os.MkdirAll(logsPath, 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Configure providers from environment
	providers := []ProviderConfig{
		{Name: "anthropic", APIKey: os.Getenv("ANTHROPIC_API_KEY"), Enabled: true},
		{Name: "openai", APIKey: os.Getenv("OPENAI_API_KEY"), Enabled: true},
		{Name: "openrouter", APIKey: os.Getenv("OPENROUTER_API_KEY"), Enabled: true},
		{Name: "deepseek", APIKey: os.Getenv("DEEPSEEK_API_KEY"), Enabled: true},
		{Name: "gemini", APIKey: os.Getenv("GEMINI_API_KEY"), Enabled: true},
		{Name: "groq", APIKey: os.Getenv("GROQ_API_KEY"), Enabled: true},
		{Name: "ollama", BaseURL: os.Getenv("OLLAMA_BASE_URL"), Enabled: true},
	}

	// Filter to configured providers
	var configuredProviders []ProviderConfig
	for _, p := range providers {
		if p.APIKey != "" || p.Name == "ollama" {
			configuredProviders = append(configuredProviders, p)
		}
	}

	if *verbose {
		log.Printf("Found %d configured providers", len(configuredProviders))
	}

	// Test each provider concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var testResults []ProviderTestResult
	var allTests []SimpleTestResult

	for _, provider := range configuredProviders {
		wg.Add(1)
		go func(p ProviderConfig) {
			defer wg.Done()

			if *verbose {
				log.Printf("Testing provider: %s", p.Name)
			}

			result := testProvider(ctx, p)

			mu.Lock()
			testResults = append(testResults, *result)
			if result.Success {
				allTests = append(allTests, result.Results...)
			}
			mu.Unlock()
		}(provider)
	}

	wg.Wait()

	// Sort tests by latency (ascending)
	sort.Slice(allTests, func(i, j int) bool {
		return allTests[i].LatencyMs < allTests[j].LatencyMs
	})

	// Calculate summary
	successfulCount := 0
	failedCount := 0
	var totalLatency int64
	passedModels := 0
	failedModels := 0
	var fastestModel string
	var fastestLatency int64 = -1

	for _, r := range testResults {
		if r.Success {
			successfulCount++
		} else {
			failedCount++
		}
	}

	for _, t := range allTests {
		if t.Success && t.HasContent {
			passedModels++
			totalLatency += t.LatencyMs
			if fastestLatency == -1 || t.LatencyMs < fastestLatency {
				fastestLatency = t.LatencyMs
				fastestModel = fmt.Sprintf("%s/%s", t.Provider, t.ModelID)
			}
		} else {
			failedModels++
		}
	}

	avgLatency := float64(0)
	if passedModels > 0 {
		avgLatency = float64(totalLatency) / float64(passedModels)
	}

	// Build final result
	challengeResult := ChallengeResult{
		ChallengeID:   "run_model_real_simple",
		ChallengeName: "Model Real Simple",
		Timestamp:     time.Now(),
		Duration:      time.Since(start),
		Status:        "passed",
		Results:       testResults,
		AllTests:      allTests,
		Summary: ChallengeSummary{
			TotalProviders:      len(configuredProviders),
			SuccessfulProviders: successfulCount,
			FailedProviders:     failedCount,
			TotalModels:         len(allTests),
			PassedModels:        passedModels,
			FailedModels:        failedModels,
			AverageLatency:      avgLatency,
			FastestModel:        fastestModel,
			FastestLatency:      fastestLatency,
		},
	}

	if successfulCount == 0 {
		challengeResult.Status = "failed"
	}

	// Write results
	resultFile := filepath.Join(resultsPath, "simple_test_results.json")
	resultData, _ := json.MarshalIndent(challengeResult, "", "  ")
	if err := os.WriteFile(resultFile, resultData, 0644); err != nil {
		log.Printf("Warning: Failed to write results: %v", err)
	}

	testsFile := filepath.Join(resultsPath, "model_tests.json")
	testsData, _ := json.MarshalIndent(allTests, "", "  ")
	if err := os.WriteFile(testsFile, testsData, 0644); err != nil {
		log.Printf("Warning: Failed to write tests: %v", err)
	}

	// Write report
	reportFile := filepath.Join(resultsPath, "simple_test_report.md")
	report := generateReport(challengeResult)
	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		log.Printf("Warning: Failed to write report: %v", err)
	}

	// Print summary
	fmt.Printf("\n=== Model Real Simple Complete ===\n")
	fmt.Printf("Status: %s\n", strings.ToUpper(challengeResult.Status))
	fmt.Printf("Providers: %d successful, %d failed\n", successfulCount, failedCount)
	fmt.Printf("Models: %d passed, %d failed\n", passedModels, failedModels)
	if fastestModel != "" {
		fmt.Printf("Fastest: %s (%dms)\n", fastestModel, fastestLatency)
	}
	fmt.Printf("Average Latency: %.0fms\n", avgLatency)
	fmt.Printf("Duration: %v\n", challengeResult.Duration)
	fmt.Printf("Results: %s\n", resultsPath)

	if challengeResult.Status == "failed" {
		os.Exit(1)
	}
}

func generateReport(result ChallengeResult) string {
	var sb strings.Builder

	sb.WriteString("# Model Real Simple Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", result.Timestamp.Format(time.RFC3339)))

	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Status | %s |\n", strings.ToUpper(result.Status)))
	sb.WriteString(fmt.Sprintf("| Total Providers | %d |\n", result.Summary.TotalProviders))
	sb.WriteString(fmt.Sprintf("| Successful Providers | %d |\n", result.Summary.SuccessfulProviders))
	sb.WriteString(fmt.Sprintf("| Failed Providers | %d |\n", result.Summary.FailedProviders))
	sb.WriteString(fmt.Sprintf("| Total Models | %d |\n", result.Summary.TotalModels))
	sb.WriteString(fmt.Sprintf("| Passed Models | %d |\n", result.Summary.PassedModels))
	sb.WriteString(fmt.Sprintf("| Failed Models | %d |\n", result.Summary.FailedModels))
	sb.WriteString(fmt.Sprintf("| Average Latency | %.0fms |\n", result.Summary.AverageLatency))
	sb.WriteString(fmt.Sprintf("| Fastest Model | %s |\n", result.Summary.FastestModel))
	sb.WriteString(fmt.Sprintf("| Fastest Latency | %dms |\n", result.Summary.FastestLatency))
	sb.WriteString(fmt.Sprintf("| Duration | %v |\n", result.Duration))

	sb.WriteString("\n## Test Prompt\n\n")
	sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", simplePrompt))

	sb.WriteString("\n## Provider Results\n\n")
	sb.WriteString("| Provider | Status | Models Tested | Models Passed | Avg Latency |\n")
	sb.WriteString("|----------|--------|---------------|---------------|-------------|\n")
	for _, r := range result.Results {
		status := "Failed"
		if r.Success {
			status = "Success"
		}
		latency := "-"
		if r.AverageLatency > 0 {
			latency = fmt.Sprintf("%.0fms", r.AverageLatency)
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %s |\n",
			r.Provider, status, r.ModelsTested, r.ModelsPassed, latency))
	}

	sb.WriteString("\n## Model Test Results\n\n")
	sb.WriteString("| Provider | Model | Status | Latency | Has Content | Coherent |\n")
	sb.WriteString("|----------|-------|--------|---------|-------------|----------|\n")
	for _, t := range result.AllTests {
		status := "Failed"
		if t.Success {
			status = "Passed"
		}
		hasContent := "No"
		if t.HasContent {
			hasContent = "Yes"
		}
		coherent := "No"
		if t.IsCoherent {
			coherent = "Yes"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %dms | %s | %s |\n",
			t.Provider, t.ModelID, status, t.LatencyMs, hasContent, coherent))
	}

	sb.WriteString("\n## Sample Responses\n\n")
	for _, t := range result.AllTests {
		if t.Success && len(t.Response) > 0 {
			response := t.Response
			if len(response) > 200 {
				response = response[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("### %s/%s\n\n", t.Provider, t.ModelID))
			sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", response))
		}
	}

	sb.WriteString("---\n\n")
	sb.WriteString("*Generated by LLMsVerifier Model Real Simple Challenge*\n")

	return sb.String()
}
