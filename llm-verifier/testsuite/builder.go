// Package testsuite provides custom test suite creation and execution
package testsuite

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TestSuite represents a custom test suite
type TestSuite struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Version       string                 `json:"version"`
	Author        string                 `json:"author"`
	Tags          []string               `json:"tags"`
	TestCases     []TestCase             `json:"test_cases"`
	Configuration SuiteConfig            `json:"configuration"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// TestCase represents an individual test case
type TestCase struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Type          TestCaseType           `json:"type"`
	Priority      TestPriority           `json:"priority"`
	Category      string                 `json:"category"`
	Tags          []string               `json:"tags"`
	Configuration TestConfig             `json:"configuration"`
	Inputs        TestInputs             `json:"inputs"`
	Expected      TestExpected           `json:"expected"`
	Metadata      map[string]interface{} `json:"metadata"`
	Enabled       bool                   `json:"enabled"`
}

// TestCaseType represents the type of test case
type TestCaseType string

const (
	TestCaseTypeBasic      TestCaseType = "basic"
	TestCaseTypeComparison TestCaseType = "comparison"
	TestCaseTypeLoad       TestCaseType = "load"
	TestCaseTypeStress     TestCaseType = "stress"
	TestCaseTypeSecurity   TestCaseType = "security"
	TestCaseTypeCompliance TestCaseType = "compliance"
	TestCaseTypeMultiModal TestCaseType = "multimodal"
	TestCaseTypeCustom     TestCaseType = "custom"
)

// TestPriority represents test priority levels
type TestPriority string

const (
	TestPriorityLow      TestPriority = "low"
	TestPriorityMedium   TestPriority = "medium"
	TestPriorityHigh     TestPriority = "high"
	TestPriorityCritical TestPriority = "critical"
)

// SuiteConfig represents test suite configuration
type SuiteConfig struct {
	ExecutionMode   ExecutionMode          `json:"execution_mode"`
	Parallelism     int                    `json:"parallelism"`
	Timeout         time.Duration          `json:"timeout"`
	RetryPolicy     RetryPolicy            `json:"retry_policy"`
	Environment     map[string]string      `json:"environment"`
	GlobalVariables map[string]interface{} `json:"global_variables"`
	Reporting       ReportingConfig        `json:"reporting"`
	Providers       []string               `json:"providers"`
}

// ExecutionMode represents how tests should be executed
type ExecutionMode string

const (
	ExecutionModeSequential  ExecutionMode = "sequential"
	ExecutionModeParallel    ExecutionMode = "parallel"
	ExecutionModeDistributed ExecutionMode = "distributed"
)

// RetryPolicy represents retry configuration
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// ReportingConfig represents reporting configuration
type ReportingConfig struct {
	Format        []string               `json:"format"`
	OutputDir     string                 `json:"output_dir"`
	IncludeRaw    bool                   `json:"include_raw"`
	Metrics       []string               `json:"metrics"`
	CustomReports map[string]interface{} `json:"custom_reports"`
}

// TestConfig represents test case configuration
type TestConfig struct {
	Provider    string                 `json:"provider"`
	Model       string                 `json:"model"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// TestInputs represents test inputs
type TestInputs struct {
	Prompt        string                 `json:"prompt,omitempty"`
	SystemMessage string                 `json:"system_message,omitempty"`
	Messages      []Message              `json:"messages,omitempty"`
	Files         []FileInput            `json:"files,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// FileInput represents a file input for multi-modal tests
type FileInput struct {
	Name    string `json:"name"`
	Content []byte `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
	Type    string `json:"type"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TestExpected represents expected test results
type TestExpected struct {
	ResponsePattern  string                 `json:"response_pattern,omitempty"`
	Contains         []string               `json:"contains,omitempty"`
	NotContains      []string               `json:"not_contains,omitempty"`
	MinLength        int                    `json:"min_length,omitempty"`
	MaxLength        int                    `json:"max_length,omitempty"`
	ResponseTime     time.Duration          `json:"response_time,omitempty"`
	SuccessRate      float64                `json:"success_rate,omitempty"`
	CustomValidators []CustomValidator      `json:"custom_validators,omitempty"`
	ScoreThreshold   float64                `json:"score_threshold,omitempty"`
	MetadataChecks   map[string]interface{} `json:"metadata_checks,omitempty"`
}

// CustomValidator represents a custom validation function
type CustomValidator struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// TestSuiteBuilder provides a fluent interface for building test suites
type TestSuiteBuilder struct {
	suite *TestSuite
}

// NewTestSuiteBuilder creates a new test suite builder
func NewTestSuiteBuilder(name, description string) *TestSuiteBuilder {
	suite := &TestSuite{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Version:     "1.0.0",
		TestCases:   []TestCase{},
		Configuration: SuiteConfig{
			ExecutionMode: ExecutionModeParallel,
			Parallelism:   5,
			Timeout:       300 * time.Second,
			RetryPolicy: RetryPolicy{
				MaxRetries:    3,
				InitialDelay:  1 * time.Second,
				MaxDelay:      30 * time.Second,
				BackoffFactor: 2.0,
			},
			Environment:     make(map[string]string),
			GlobalVariables: make(map[string]interface{}),
			Reporting: ReportingConfig{
				Format:     []string{"json", "html"},
				OutputDir:  "./reports",
				IncludeRaw: true,
				Metrics:    []string{"latency", "success_rate", "cost"},
			},
		},
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return &TestSuiteBuilder{suite: suite}
}

// WithVersion sets the suite version
func (tsb *TestSuiteBuilder) WithVersion(version string) *TestSuiteBuilder {
	tsb.suite.Version = version
	return tsb
}

// WithAuthor sets the suite author
func (tsb *TestSuiteBuilder) WithAuthor(author string) *TestSuiteBuilder {
	tsb.suite.Author = author
	return tsb
}

// WithTags adds tags to the suite
func (tsb *TestSuiteBuilder) WithTags(tags ...string) *TestSuiteBuilder {
	tsb.suite.Tags = append(tsb.suite.Tags, tags...)
	return tsb
}

// WithExecutionMode sets the execution mode
func (tsb *TestSuiteBuilder) WithExecutionMode(mode ExecutionMode) *TestSuiteBuilder {
	tsb.suite.Configuration.ExecutionMode = mode
	return tsb
}

// WithParallelism sets the parallelism level
func (tsb *TestSuiteBuilder) WithParallelism(parallelism int) *TestSuiteBuilder {
	tsb.suite.Configuration.Parallelism = parallelism
	return tsb
}

// WithProviders sets the providers to test against
func (tsb *TestSuiteBuilder) WithProviders(providers ...string) *TestSuiteBuilder {
	tsb.suite.Configuration.Providers = providers
	return tsb
}

// AddTestCase adds a test case to the suite
func (tsb *TestSuiteBuilder) AddTestCase(testCase TestCase) *TestSuiteBuilder {
	testCase.ID = uuid.New().String()
	if testCase.Enabled == false && testCase.ID == "" {
		testCase.Enabled = true
	}
	tsb.suite.TestCases = append(tsb.suite.TestCases, testCase)
	return tsb
}

// AddBasicTestCase adds a basic prompt-response test case
func (tsb *TestSuiteBuilder) AddBasicTestCase(name, prompt string, expectedContains []string) *TestSuiteBuilder {
	testCase := TestCase{
		Name:        name,
		Description: fmt.Sprintf("Basic test case: %s", name),
		Type:        TestCaseTypeBasic,
		Priority:    TestPriorityMedium,
		Category:    "basic",
		Configuration: TestConfig{
			MaxTokens:  1000,
			Timeout:    30 * time.Second,
			RetryCount: 2,
		},
		Inputs: TestInputs{
			Prompt: prompt,
		},
		Expected: TestExpected{
			Contains:       expectedContains,
			ResponseTime:   10 * time.Second,
			ScoreThreshold: 0.8,
		},
		Enabled: true,
	}

	return tsb.AddTestCase(testCase)
}

// AddComparisonTestCase adds a comparison test case between providers
func (tsb *TestSuiteBuilder) AddComparisonTestCase(name, prompt string) *TestSuiteBuilder {
	testCase := TestCase{
		Name:        name,
		Description: fmt.Sprintf("Provider comparison: %s", name),
		Type:        TestCaseTypeComparison,
		Priority:    TestPriorityHigh,
		Category:    "comparison",
		Configuration: TestConfig{
			MaxTokens:  2000,
			Timeout:    60 * time.Second,
			RetryCount: 3,
		},
		Inputs: TestInputs{
			Prompt: prompt,
		},
		Expected: TestExpected{
			MinLength:      100,
			ResponseTime:   30 * time.Second,
			ScoreThreshold: 0.7,
		},
		Enabled: true,
	}

	return tsb.AddTestCase(testCase)
}

// AddLoadTestCase adds a load testing case
func (tsb *TestSuiteBuilder) AddLoadTestCase(name string, concurrentUsers, duration int) *TestSuiteBuilder {
	testCase := TestCase{
		Name:        name,
		Description: fmt.Sprintf("Load test: %d users for %d seconds", concurrentUsers, duration),
		Type:        TestCaseTypeLoad,
		Priority:    TestPriorityHigh,
		Category:    "performance",
		Configuration: TestConfig{
			MaxTokens:  500,
			Timeout:    time.Duration(duration) * time.Second,
			RetryCount: 1,
		},
		Inputs: TestInputs{
			Prompt: "Generate a short summary about artificial intelligence.",
		},
		Expected: TestExpected{
			ResponseTime: 5 * time.Second,
			SuccessRate:  0.95,
		},
		Metadata: map[string]interface{}{
			"concurrent_users": concurrentUsers,
			"duration_seconds": duration,
		},
		Enabled: true,
	}

	return tsb.AddTestCase(testCase)
}

// AddMultiModalTestCase adds a multi-modal test case
func (tsb *TestSuiteBuilder) AddMultiModalTestCase(name, prompt string, fileType string) *TestSuiteBuilder {
	testCase := TestCase{
		Name:        name,
		Description: fmt.Sprintf("Multi-modal test: %s", name),
		Type:        TestCaseTypeMultiModal,
		Priority:    TestPriorityMedium,
		Category:    "multimodal",
		Configuration: TestConfig{
			MaxTokens:  1000,
			Timeout:    60 * time.Second,
			RetryCount: 2,
		},
		Inputs: TestInputs{
			Prompt: prompt,
			Files: []FileInput{
				{
					Name: "test_file",
					Type: fileType,
				},
			},
		},
		Expected: TestExpected{
			ResponseTime:   45 * time.Second,
			ScoreThreshold: 0.75,
		},
		Enabled: true,
	}

	return tsb.AddTestCase(testCase)
}

// Build returns the constructed test suite
func (tsb *TestSuiteBuilder) Build() *TestSuite {
	tsb.suite.UpdatedAt = time.Now()
	return tsb.suite
}

// TestSuiteExecutor executes test suites
type TestSuiteExecutor struct {
	suite     *TestSuite
	results   []TestResult
	startTime time.Time
	endTime   time.Time
	isRunning bool
}

// TestResult represents the result of a test execution
type TestResult struct {
	TestCaseID   string                 `json:"test_case_id"`
	TestCaseName string                 `json:"test_case_name"`
	Status       TestStatus             `json:"status"`
	Duration     time.Duration          `json:"duration"`
	Response     string                 `json:"response,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Score        float64                `json:"score,omitempty"`
	Provider     string                 `json:"provider,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// TestStatus represents the status of a test execution
type TestStatus string

const (
	TestStatusPending TestStatus = "pending"
	TestStatusRunning TestStatus = "running"
	TestStatusPassed  TestStatus = "passed"
	TestStatusFailed  TestStatus = "failed"
	TestStatusSkipped TestStatus = "skipped"
	TestStatusError   TestStatus = "error"
)

// NewTestSuiteExecutor creates a new test suite executor
func NewTestSuiteExecutor(suite *TestSuite) *TestSuiteExecutor {
	return &TestSuiteExecutor{
		suite:   suite,
		results: []TestResult{},
	}
}

// Execute runs the test suite
func (tse *TestSuiteExecutor) Execute(ctx context.Context) (*ExecutionReport, error) {
	tse.startTime = time.Now()
	tse.isRunning = true
	defer func() {
		tse.endTime = time.Now()
		tse.isRunning = false
	}()

	report := &ExecutionReport{
		SuiteID:     tse.suite.ID,
		SuiteName:   tse.suite.Name,
		StartTime:   tse.startTime,
		TestResults: []TestResult{},
		Summary:     ExecutionSummary{},
	}

	// Execute test cases based on execution mode
	switch tse.suite.Configuration.ExecutionMode {
	case ExecutionModeSequential:
		if err := tse.executeSequential(ctx, report); err != nil {
			return nil, err
		}
	case ExecutionModeParallel:
		if err := tse.executeParallel(ctx, report); err != nil {
			return nil, err
		}
	case ExecutionModeDistributed:
		if err := tse.executeDistributed(ctx, report); err != nil {
			return nil, err
		}
	default:
		if err := tse.executeSequential(ctx, report); err != nil {
			return nil, err
		}
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)
	report.Summary = tse.generateSummary(report.TestResults)

	return report, nil
}

// executeSequential executes test cases sequentially
func (tse *TestSuiteExecutor) executeSequential(ctx context.Context, report *ExecutionReport) error {
	for _, testCase := range tse.suite.TestCases {
		if !testCase.Enabled {
			continue
		}

		result := tse.executeTestCase(ctx, testCase)
		report.TestResults = append(report.TestResults, result)
	}

	return nil
}

// executeParallel executes test cases in parallel
func (tse *TestSuiteExecutor) executeParallel(ctx context.Context, report *ExecutionReport) error {
	semaphore := make(chan struct{}, tse.suite.Configuration.Parallelism)
	resultsChan := make(chan TestResult, len(tse.suite.TestCases))

	// Start workers
	for _, testCase := range tse.suite.TestCases {
		if !testCase.Enabled {
			continue
		}

		go func(tc TestCase) {
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			result := tse.executeTestCase(ctx, tc)
			resultsChan <- result
		}(testCase)
	}

	// Collect results
	for i := 0; i < len(tse.suite.TestCases); i++ {
		if result := <-resultsChan; result.TestCaseID != "" {
			report.TestResults = append(report.TestResults, result)
		}
	}

	close(resultsChan)
	return nil
}

// executeDistributed executes test cases in distributed mode (placeholder)
func (tse *TestSuiteExecutor) executeDistributed(ctx context.Context, report *ExecutionReport) error {
	// In a real implementation, this would distribute tests across multiple nodes
	return tse.executeParallel(ctx, report)
}

// executeTestCase executes a single test case
func (tse *TestSuiteExecutor) executeTestCase(ctx context.Context, testCase TestCase) TestResult {
	startTime := time.Now()

	result := TestResult{
		TestCaseID:   testCase.ID,
		TestCaseName: testCase.Name,
		Status:       TestStatusRunning,
		Timestamp:    startTime,
		Metrics:      make(map[string]interface{}),
	}

	// Simulate test execution (in real implementation, this would call LLM providers)
	time.Sleep(time.Duration(rand.Intn(2000)+500) * time.Millisecond) // Random delay 500-2500ms

	// Mock response based on test type
	switch testCase.Type {
	case TestCaseTypeBasic:
		result = tse.executeBasicTest(testCase, result)
	case TestCaseTypeComparison:
		result = tse.executeComparisonTest(testCase, result)
	case TestCaseTypeLoad:
		result = tse.executeLoadTest(testCase, result)
	case TestCaseTypeMultiModal:
		result = tse.executeMultiModalTest(testCase, result)
	default:
		result.Status = TestStatusError
		result.Error = "Unsupported test case type"
	}

	result.Duration = time.Since(startTime)
	return result
}

// executeBasicTest executes a basic test
func (tse *TestSuiteExecutor) executeBasicTest(testCase TestCase, result TestResult) TestResult {
	// Mock LLM response
	result.Response = fmt.Sprintf("Response to: %s", testCase.Inputs.Prompt)
	result.Provider = "mock-provider"
	result.Model = "mock-model"
	result.Score = 0.85 + rand.Float64()*0.1 // Random score 0.85-0.95

	// Check expectations
	if tse.checkExpectations(testCase, result) {
		result.Status = TestStatusPassed
	} else {
		result.Status = TestStatusFailed
	}

	return result
}

// executeComparisonTest executes a comparison test
func (tse *TestSuiteExecutor) executeComparisonTest(testCase TestCase, result TestResult) TestResult {
	providers := []string{"openai", "anthropic", "google"}
	result.Provider = providers[rand.Intn(len(providers))]
	result.Model = "comparison-model"
	result.Response = fmt.Sprintf("Comparison response from %s", result.Provider)
	result.Score = 0.75 + rand.Float64()*0.2 // Random score 0.75-0.95

	if tse.checkExpectations(testCase, result) {
		result.Status = TestStatusPassed
	} else {
		result.Status = TestStatusFailed
	}

	return result
}

// executeLoadTest executes a load test
func (tse *TestSuiteExecutor) executeLoadTest(testCase TestCase, result TestResult) TestResult {
	// Mock load test execution
	concurrentUsers := testCase.Metadata["concurrent_users"].(int)
	result.Response = fmt.Sprintf("Load test completed with %d concurrent users", concurrentUsers)
	result.Score = 0.9 + rand.Float64()*0.05 // High score for load tests

	if tse.checkExpectations(testCase, result) {
		result.Status = TestStatusPassed
	} else {
		result.Status = TestStatusFailed
	}

	return result
}

// executeMultiModalTest executes a multi-modal test
func (tse *TestSuiteExecutor) executeMultiModalTest(testCase TestCase, result TestResult) TestResult {
	result.Response = "Multi-modal content processed successfully"
	result.Score = 0.8 + rand.Float64()*0.15 // Score 0.8-0.95

	if tse.checkExpectations(testCase, result) {
		result.Status = TestStatusPassed
	} else {
		result.Status = TestStatusFailed
	}

	return result
}

// checkExpectations validates test expectations
func (tse *TestSuiteExecutor) checkExpectations(testCase TestCase, result TestResult) bool {
	expected := testCase.Expected

	// Check response time
	if expected.ResponseTime > 0 && result.Duration > expected.ResponseTime {
		return false
	}

	// Check score threshold
	if expected.ScoreThreshold > 0 && result.Score < expected.ScoreThreshold {
		return false
	}

	// Check content expectations
	for _, contain := range expected.Contains {
		if !strings.Contains(result.Response, contain) {
			return false
		}
	}

	for _, notContain := range expected.NotContains {
		if strings.Contains(result.Response, notContain) {
			return false
		}
	}

	// Check length constraints
	if expected.MinLength > 0 && len(result.Response) < expected.MinLength {
		return false
	}

	if expected.MaxLength > 0 && len(result.Response) > expected.MaxLength {
		return false
	}

	return true
}

// generateSummary generates execution summary
func (tse *TestSuiteExecutor) generateSummary(results []TestResult) ExecutionSummary {
	summary := ExecutionSummary{
		TotalTests:    len(results),
		PassedTests:   0,
		FailedTests:   0,
		SkippedTests:  0,
		ErrorTests:    0,
		AvgScore:      0,
		AvgDuration:   0,
		TotalDuration: 0,
	}

	totalScore := 0.0
	totalDuration := time.Duration(0)

	for _, result := range results {
		switch result.Status {
		case TestStatusPassed:
			summary.PassedTests++
		case TestStatusFailed:
			summary.FailedTests++
		case TestStatusSkipped:
			summary.SkippedTests++
		case TestStatusError:
			summary.ErrorTests++
		}

		totalScore += result.Score
		totalDuration += result.Duration
	}

	if len(results) > 0 {
		summary.AvgScore = totalScore / float64(len(results))
		summary.AvgDuration = totalDuration / time.Duration(len(results))
	}

	summary.TotalDuration = totalDuration

	// Calculate percentiles
	if len(results) > 0 {
		durations := make([]time.Duration, len(results))
		for i, result := range results {
			durations[i] = result.Duration
		}
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

		summary.P50Duration = durations[len(durations)/2]
		p95Index := int(float64(len(durations)) * 0.95)
		if p95Index < len(durations) {
			summary.P95Duration = durations[p95Index]
		}
	}

	return summary
}

// ExecutionReport represents a test execution report
type ExecutionReport struct {
	SuiteID     string           `json:"suite_id"`
	SuiteName   string           `json:"suite_name"`
	StartTime   time.Time        `json:"start_time"`
	EndTime     time.Time        `json:"end_time"`
	Duration    time.Duration    `json:"duration"`
	TestResults []TestResult     `json:"test_results"`
	Summary     ExecutionSummary `json:"summary"`
}

// ExecutionSummary represents execution summary statistics
type ExecutionSummary struct {
	TotalTests    int           `json:"total_tests"`
	PassedTests   int           `json:"passed_tests"`
	FailedTests   int           `json:"failed_tests"`
	SkippedTests  int           `json:"skipped_tests"`
	ErrorTests    int           `json:"error_tests"`
	AvgScore      float64       `json:"avg_score"`
	AvgDuration   time.Duration `json:"avg_duration"`
	P50Duration   time.Duration `json:"p50_duration"`
	P95Duration   time.Duration `json:"p95_duration"`
	TotalDuration time.Duration `json:"total_duration"`
}

// TestSuiteManager manages test suites
type TestSuiteManager struct {
	suites map[string]*TestSuite
}

// NewTestSuiteManager creates a new test suite manager
func NewTestSuiteManager() *TestSuiteManager {
	return &TestSuiteManager{
		suites: make(map[string]*TestSuite),
	}
}

// SaveSuite saves a test suite
func (tsm *TestSuiteManager) SaveSuite(suite *TestSuite) error {
	tsm.suites[suite.ID] = suite
	return nil
}

// GetSuite retrieves a test suite by ID
func (tsm *TestSuiteManager) GetSuite(id string) (*TestSuite, error) {
	suite, exists := tsm.suites[id]
	if !exists {
		return nil, fmt.Errorf("test suite not found: %s", id)
	}
	return suite, nil
}

// ListSuites returns all test suites
func (tsm *TestSuiteManager) ListSuites() []*TestSuite {
	suites := make([]*TestSuite, 0, len(tsm.suites))
	for _, suite := range tsm.suites {
		suites = append(suites, suite)
	}
	return suites
}

// DeleteSuite deletes a test suite
func (tsm *TestSuiteManager) DeleteSuite(id string) error {
	if _, exists := tsm.suites[id]; !exists {
		return fmt.Errorf("test suite not found: %s", id)
	}
	delete(tsm.suites, id)
	return nil
}

// ExportSuite exports a test suite to JSON
func (tsm *TestSuiteManager) ExportSuite(id string) ([]byte, error) {
	suite, err := tsm.GetSuite(id)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(suite, "", "  ")
}

// ImportSuite imports a test suite from JSON
func (tsm *TestSuiteManager) ImportSuite(data []byte) (*TestSuite, error) {
	var suite TestSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, err
	}

	// Generate new ID to avoid conflicts
	suite.ID = uuid.New().String()
	suite.CreatedAt = time.Now()
	suite.UpdatedAt = time.Now()

	if err := tsm.SaveSuite(&suite); err != nil {
		return nil, err
	}

	return &suite, nil
}

// CreateTemplateSuites creates predefined template test suites
func CreateTemplateSuites() []*TestSuite {
	suites := []*TestSuite{}

	// Basic functionality test suite
	basicSuite := NewTestSuiteBuilder("Basic LLM Tests", "Basic functionality tests for LLM providers").
		WithAuthor("LLM Verifier").
		WithTags("basic", "functional", "smoke").
		WithProviders("openai", "anthropic", "google").
		AddBasicTestCase("Greeting Test", "Say hello in a friendly way", []string{"hello", "hi"}).
		AddBasicTestCase("Math Test", "What is 2 + 2?", []string{"4"}).
		AddBasicTestCase("Code Test", "Write a simple Python function to add two numbers", []string{"def", "return"}).
		Build()

	suites = append(suites, basicSuite)

	// Performance test suite
	perfSuite := NewTestSuiteBuilder("Performance Tests", "Performance and load testing suite").
		WithAuthor("LLM Verifier").
		WithTags("performance", "load", "stress").
		WithExecutionMode(ExecutionModeParallel).
		WithParallelism(10).
		AddLoadTestCase("Light Load Test", 5, 30).
		AddLoadTestCase("Medium Load Test", 25, 60).
		AddLoadTestCase("Heavy Load Test", 100, 120).
		Build()

	suites = append(suites, perfSuite)

	// Comparison test suite
	comparisonSuite := NewTestSuiteBuilder("Provider Comparison", "Compare responses across different providers").
		WithAuthor("LLM Verifier").
		WithTags("comparison", "quality", "consistency").
		WithProviders("openai", "anthropic", "google", "groq", "together").
		AddComparisonTestCase("Creative Writing", "Write a short story about a robot learning to paint").
		AddComparisonTestCase("Technical Explanation", "Explain how quantum computing works in simple terms").
		AddComparisonTestCase("Code Review", "Review this Python function and suggest improvements: def fibonacci(n): return n if n <= 1 else fibonacci(n-1) + fibonacci(n-2)").
		Build()

	suites = append(suites, comparisonSuite)

	// Multi-modal test suite
	multimodalSuite := NewTestSuiteBuilder("Multi-Modal Tests", "Test multi-modal capabilities").
		WithAuthor("LLM Verifier").
		WithTags("multimodal", "vision", "audio").
		AddMultiModalTestCase("Image Description", "Describe this image in detail", "image/jpeg").
		AddMultiModalTestCase("Audio Transcription", "Transcribe this audio file", "audio/mpeg").
		Build()

	suites = append(suites, multimodalSuite)

	return suites
}
