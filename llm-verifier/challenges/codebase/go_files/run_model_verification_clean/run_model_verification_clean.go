// Package main implements a clean, simplified Model Verification challenge.
// This is a minimal version that focuses on basic verification without
// provider-specific complexities.
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
	"time"
)

// ChallengeResult represents the result of the verification challenge
type ChallengeResult struct {
	ChallengeID   string              `json:"challenge_id"`
	ChallengeName string              `json:"challenge_name"`
	Success       bool                `json:"success"`
	Message       string              `json:"message"`
	Timestamp     time.Time           `json:"timestamp"`
	Duration      string              `json:"duration"`
	Models        []ModelResult       `json:"models,omitempty"`
	Error         string              `json:"error,omitempty"`
}

// ModelResult represents verification result for a single model
type ModelResult struct {
	ModelID   string  `json:"model_id"`
	Provider  string  `json:"provider"`
	Verified  bool    `json:"verified"`
	Score     float64 `json:"score"`
	LatencyMs int64   `json:"latency_ms"`
	Error     string  `json:"error,omitempty"`
}

// ChatRequest represents an OpenAI-compatible chat request
type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

// ChatMessage represents a message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents an OpenAI-compatible chat response
type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

// verifyModel performs a basic verification test on a model
func verifyModel(ctx context.Context, baseURL, apiKey, model string) *ModelResult {
	result := &ModelResult{
		ModelID:  model,
		Provider: "openai-compatible",
	}

	start := time.Now()

	// Create verification request
	reqBody := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "user", Content: "Say 'verified' if you can read this message."},
		},
		MaxTokens: 50,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		result.Error = fmt.Sprintf("failed to marshal request: %v", err)
		return result
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.LatencyMs = time.Since(start).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(respBody))
		return result
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		result.Error = fmt.Sprintf("failed to decode response: %v", err)
		return result
	}

	// Check for valid response
	if len(chatResp.Choices) == 0 {
		result.Error = "no choices in response"
		return result
	}

	// Model is verified if we got a valid response
	result.Verified = true
	result.Score = 100.0

	return result
}

func main() {
	resultsDir := flag.String("results-dir", "", "Directory to store results")
	model := flag.String("model", "gpt-3.5-turbo", "Model to verify")
	baseURL := flag.String("base-url", "https://api.openai.com/v1", "API base URL")
	flag.Parse()

	start := time.Now()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		result := ChallengeResult{
			ChallengeID:   "run_model_verification_clean",
			ChallengeName: "Clean Model Verification",
			Success:       false,
			Message:       "OPENAI_API_KEY environment variable not set",
			Timestamp:     time.Now(),
			Duration:      time.Since(start).String(),
			Error:         "missing API key",
		}
		outputResult(result, *resultsDir)
		log.Fatal("OPENAI_API_KEY not set")
	}

	// Run verification
	ctx := context.Background()
	modelResult := verifyModel(ctx, *baseURL, apiKey, *model)

	// Build challenge result
	result := ChallengeResult{
		ChallengeID:   "run_model_verification_clean",
		ChallengeName: "Clean Model Verification",
		Success:       modelResult.Verified,
		Message:       fmt.Sprintf("Model %s verification complete", *model),
		Timestamp:     time.Now(),
		Duration:      time.Since(start).String(),
		Models:        []ModelResult{*modelResult},
	}

	if !modelResult.Verified {
		result.Error = modelResult.Error
	}

	// Output results
	outputResult(result, *resultsDir)

	// Print summary
	fmt.Printf("\n=== Clean Model Verification ===\n")
	fmt.Printf("Model: %s\n", *model)
	fmt.Printf("Verified: %v\n", modelResult.Verified)
	if modelResult.Verified {
		fmt.Printf("Score: %.1f%%\n", modelResult.Score)
		fmt.Printf("Latency: %dms\n", modelResult.LatencyMs)
	} else {
		fmt.Printf("Error: %s\n", modelResult.Error)
	}
	fmt.Printf("Duration: %s\n", result.Duration)

	if !result.Success {
		os.Exit(1)
	}
}

func outputResult(result ChallengeResult, resultsDir string) {
	data, _ := json.MarshalIndent(result, "", "  ")

	if resultsDir != "" {
		resultsPath := filepath.Join(resultsDir, "results")
		if err := os.MkdirAll(resultsPath, 0755); err != nil {
			log.Printf("Warning: Failed to create results directory: %v", err)
			return
		}
		resultFile := filepath.Join(resultsPath, "verification_clean_results.json")
		if err := os.WriteFile(resultFile, data, 0644); err != nil {
			log.Printf("Warning: Failed to write results: %v", err)
		}
	}

	// Also print to stdout for testing
	fmt.Println(string(data))
}
