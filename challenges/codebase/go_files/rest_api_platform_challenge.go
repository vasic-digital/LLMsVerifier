package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type APIChallengeResult struct {
	ChallengeName string       `json:"challenge_name"`
	ChallengeDate string       `json:"challenge_date"`
	StartTime     string       `json:"start_time"`
	EndTime       string       `json:"end_time"`
	Duration      string       `json:"duration"`
	TestResults   []TestResult `json:"test_results"`
	Summary       APISummary  `json:"summary"`
}

type TestResult struct {
	TestName   string `json:"test_name"`
	Endpoint   string `json:"endpoint"`
	Method     string `json:"method"`
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type APISummary struct {
	TotalTests      int     `json:"total_tests"`
	SuccessfulTests int     `json:"successful_tests"`
	FailedTests     int     `json:"failed_tests"`
	SuccessRate     float64 `json:"success_rate"`
}

var logger *log.Logger

func main() {
	challengeDir := ""
	if len(os.Args) > 1 {
		challengeDir = os.Args[1]
	}
	if challengeDir == "" {
		challengeDir = fmt.Sprintf("challenges/rest_api_platform_challenge/%s/%s/%s/%d",
			time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), time.Now().Unix())
	}

	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(resultsDir, 0755)

	logFile, _ := os.Create(filepath.Join(logDir, "challenge.log"))
	logger = log.New(io.MultiWriter(os.Stdout, logFile), "API-CHALLENGE: ", log.Ldate|log.Ltime|log.Lmicroseconds)

	startTime := time.Now()
	logger.Printf("Starting REST API Platform Challenge")

	baseURL := "http://localhost:8080/api/v1"

	result := APIChallengeResult{
		ChallengeName: "REST API Platform Challenge",
		ChallengeDate: time.Now().Format("2006-01-02"),
		StartTime:     startTime.Format(time.RFC3339),
		TestResults:   runAllTests(baseURL, logDir),
	}

	result.EndTime = time.Now().Format(time.RFC3339)
	result.Duration = time.Since(startTime).String()

	successCount := 0
	for _, test := range result.TestResults {
		if test.Success {
			successCount++
		}
	}

	result.Summary = APISummary{
		TotalTests:      len(result.TestResults),
		SuccessfulTests: successCount,
		FailedTests:     len(result.TestResults) - successCount,
		SuccessRate:     float64(successCount) / float64(len(result.TestResults)) * 100,
	}

	saveResults(resultsDir, result)
	generateSummary(result, resultsDir)
	logger.Printf("Challenge completed in %s", result.Duration)
}

func runAllTests(baseURL, logDir string) []TestResult {
	var results []TestResult

	tests := []struct {
		name     string
		endpoint string
		method   string
		body     string
	}{
		{"Auth Test", "/auth/login", "POST", `{"username":"admin","password":"password"}`},
		{"Model Discovery", "/discover", "POST", `{"providers":["openai","anthropic"]}`},
		{"Model Verification", "/verify", "POST", `{"model_id":"gpt-4","provider":"openai"}`},
		{"Database Query", "/models", "GET", ""},
		{"Health Check", "/health", "GET", ""},
		{"Metrics", "/metrics", "GET", ""},
	}

	for _, test := range tests {
		result := runAPITest(test.name, baseURL+test.endpoint, test.method, test.body)
		results = append(results, result)
		time.Sleep(500 * time.Millisecond)
	}

	return results
}

func runAPITest(name, url, method, body string) TestResult {
	logger.Printf("Running API test: %s", name)
	start := time.Now()

	var resp *http.Response
	var err error

	if method == "GET" {
		resp, err = http.Get(url)
	} else {
		resp, err = http.Post(url, "application/json", strings.NewReader(body))
	}

	duration := time.Since(start)

	success := err == nil
	var statusCode int
	var errMsg string
	if err != nil {
		errMsg = err.Error()
		logger.Printf("Test failed: %v", errMsg)
	} else {
		statusCode = resp.StatusCode
		if statusCode >= 200 && statusCode < 300 {
			logger.Printf("Test succeeded: %d", statusCode)
		} else {
			success = false
			errMsg = fmt.Sprintf("HTTP %d", statusCode)
			logger.Printf("Test failed with status: %d", statusCode)
		}
		resp.Body.Close()
	}

	return TestResult{
		TestName:   name,
		Endpoint:   url,
		Method:     method,
		Success:    success,
		StatusCode: statusCode,
		DurationMs: duration.Milliseconds(),
		Error:      errMsg,
	}
}

func saveResults(resultsDir string, result APIChallengeResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(resultsDir, "challenge_result.json"), data, 0644)
}

func generateSummary(result APIChallengeResult, resultsDir string) {
	summary := fmt.Sprintf("# REST API Platform Challenge Summary\n\n## Challenge Information\n- **Name**: %s\n- **Date**: %s\n- **Duration**: %s\n\n## Test Results\n- **Total Tests**: %d\n- **Successful**: %d\n- **Failed**: %d\n- **Success Rate**: %.2f%%\n\n",
		result.ChallengeName, result.ChallengeDate, result.Duration,
		result.Summary.TotalTests, result.Summary.SuccessfulTests,
		result.Summary.FailedTests, result.Summary.SuccessRate)

	summary += "## Test Details\n"
	for _, test := range result.TestResults {
		status := "✓ PASSED"
		if !test.Success {
			status = "✗ FAILED"
		}
		summary += fmt.Sprintf("\n### %s %s\n\n**Endpoint**: %s\n**Method**: %s\n", test.TestName, status, test.Endpoint, test.Method)
		if test.Success {
			summary += fmt.Sprintf("**Status Code**: %d\n**Duration**: %dms\n\n", test.StatusCode, test.DurationMs)
		} else {
			summary += fmt.Sprintf("**Error**: %s\n\n", test.Error)
		}
	}

	os.WriteFile(filepath.Join(resultsDir, "summary.md"), []byte(summary), 0644)
}
