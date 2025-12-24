package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type CLIChallengeResult struct {
	ChallengeName string       `json:"challenge_name"`
	ChallengeDate string       `json:"challenge_date"`
	StartTime     string       `json:"start_time"`
	EndTime       string       `json:"end_time"`
	Duration      string       `json:"duration"`
	TestResults   []TestResult `json:"test_results"`
	Summary       CLISummary   `json:"summary"`
}

type TestResult struct {
	TestName   string `json:"test_name"`
	Command    string `json:"command"`
	Success    bool   `json:"success"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

type CLISummary struct {
	TotalTests      int     `json:"total_tests"`
	SuccessfulTests int     `json:"successful_tests"`
	FailedTests     int     `json:"failed_tests"`
	SuccessRate     float64 `json:"success_rate"`
}

type TestCase struct {
	name string
	args []string
	desc string
}

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "CLI-CHALLENGE: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	logger.SetOutput(io.MultiWriter(os.Stdout))
}

func main() {
	challengeDir := ""
	if len(os.Args) > 1 {
		challengeDir = os.Args[1]
	}
	if challengeDir == "" {
		challengeDir = fmt.Sprintf("challenges/cli_platform_challenge/%s/%s/%s/%d",
			time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), time.Now().Unix())
	}

	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(resultsDir, 0755)

	logFile, err := os.Create(filepath.Join(logDir, "challenge.log"))
	if err != nil {
		logger.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()
	logger.SetOutput(io.MultiWriter(os.Stdout, logFile))

	startTime := time.Now()
	logger.Printf("Starting CLI Platform Challenge")
	logger.Printf("Challenge directory: %s", challengeDir)

	cliBinary := findBinary("llm-verifier")
	if cliBinary == "" {
		logger.Fatalf("llm-verifier binary not found")
	}

	result := CLIChallengeResult{
		ChallengeName: "CLI Platform Challenge",
		ChallengeDate: time.Now().Format("2006-01-02"),
		StartTime:     startTime.Format(time.RFC3339),
		TestResults:   []TestResult{},
	}

	results := runAllTests(cliBinary, logDir)
	result.TestResults = results
	result.EndTime = time.Now().Format(time.RFC3339)
	result.Duration = time.Since(startTime).String()

	result.Summary = CLISummary{
		TotalTests:      len(results),
		SuccessfulTests: countSuccessful(results),
		FailedTests:     countFailed(results),
		SuccessRate:     float64(countSuccessful(results)) / float64(len(results)) * 100,
	}

	saveResults(resultsDir, result)
	generateSummary(result, resultsDir)
	logger.Printf("Challenge completed in %s", result.Duration)
}

func runAllTests(cliBinary string, logDir string) []TestResult {
	var results []TestResult

	tests := []TestCase{
		{"Basic Model Discovery", []string{"discover", "--providers", "openai,anthropic", "--output-file", filepath.Join(logDir, "discover.json")}, "Test model discovery via CLI"},
		{"Model Verification", []string{"verify", "--model", "gpt-4", "--features", "streaming,function_calling"}, "Test model verification"},
		{"Database Query", []string{"query", "--sort-by", "score", "--top", "10", "--output", filepath.Join(logDir, "query.json")}, "Test database query"},
		{"Limits Check", []string{"limits", "--model", "gpt-4", "--provider", "openai"}, "Test limits checking"},
		{"Config Export OpenCode", []string{"export", "--target", "opencode", "--output", filepath.Join(logDir, "opencode_config.json")}, "Test OpenCode export"},
		{"Event Subscribe", []string{"events", "subscribe", "--type", "score_change", "--channel", "stdout"}, "Test event subscription"},
		{"Schedule Task", []string{"schedule", "--create", "--interval", "daily", "--all-models", "--time", "02:00"}, "Test task scheduling"},
		{"Report Generation", []string{"report", "--format", "markdown", "--output", filepath.Join(logDir, "report.md"), "--all-models"}, "Test report generation"},
	}

	for _, test := range tests {
		result := runTest(test.name, cliBinary, test.args, logDir)
		results = append(results, result)
		time.Sleep(1 * time.Second)
	}

	return results
}

func runTest(name string, binary string, args []string, logDir string) TestResult {
	logger.Printf("Running test: %s", name)
	start := time.Now()

	cmd := exec.Command(binary, args...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	success := err == nil
	var errMsg string
	if err != nil {
		errMsg = err.Error()
		logger.Printf("Test failed: %v", errMsg)
	} else {
		logger.Printf("Test succeeded")
	}

	testLogFile := filepath.Join(logDir, fmt.Sprintf("%s.log", strings.ReplaceAll(name, " ", "_")))
	os.WriteFile(testLogFile, output, 0644)

	return TestResult{
		TestName:   name,
		Command:    fmt.Sprintf("%s %s", binary, strings.Join(args, " ")),
		Success:    success,
		Output:     string(output),
		Error:      errMsg,
		DurationMs: duration.Milliseconds(),
	}
}

func findBinary(name string) string {
	paths := []string{
		filepath.Join("..", "..", "cmd", "cmd"),
		filepath.Join("..", "..", "llm-verifier", "cmd", "cmd"),
		"llm-verifier",
		filepath.Join("..", "..", "build", "llm-verifier"),
		filepath.Join("..", "..", "llm-verifier", "llm-verifier"),
	}

	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			mode := info.Mode()
			if mode.Perm()&0111 != 0 {
				return path
			}
		}
	}

	return ""
}

func countSuccessful(results []TestResult) int {
	count := 0
	for _, r := range results {
		if r.Success {
			count++
		}
	}
	return count
}

func countFailed(results []TestResult) int {
	count := 0
	for _, r := range results {
		if !r.Success {
			count++
		}
	}
	return count
}

func saveResults(resultsDir string, result CLIChallengeResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(resultsDir, "challenge_result.json"), data, 0644)
}

func generateSummary(result CLIChallengeResult, resultsDir string) {
	summary := fmt.Sprintf(`# CLI Platform Challenge Summary

## Challenge Information
- **Name**: %s
- **Date**: %s
- **Duration**: %s

## Test Results
- **Total Tests**: %d
- **Successful**: %d
- **Failed**: %d
- **Success Rate**: %.2f%%

## Test Details
`, result.ChallengeName, result.ChallengeDate, result.Duration,
		result.Summary.TotalTests, result.Summary.SuccessfulTests,
		result.Summary.FailedTests, result.Summary.SuccessRate)

	for _, test := range result.TestResults {
		status := "✓ PASSED"
		if !test.Success {
			status = "✗ FAILED"
		}
		summary += fmt.Sprintf("\n### %s %s\n\n", test.TestName, status)
		summary += fmt.Sprintf("**Command**: `%s`\n\n", test.Command)
		if test.Success {
			summary += fmt.Sprintf("**Duration**: %dms\n\n", test.DurationMs)
		} else {
			summary += fmt.Sprintf("**Error**: %s\n\n", test.Error)
		}
	}

	os.WriteFile(filepath.Join(resultsDir, "summary.md"), []byte(summary), 0644)
}
