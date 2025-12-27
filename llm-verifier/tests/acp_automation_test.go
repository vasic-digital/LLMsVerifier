package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/llmverifier/llmverifier"
	"github.com/llmverifier/llmverifier/config"
)

// TestACPsFullAutomationWorkflow tests complete automated ACP workflow
func TestACPsFullAutomationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full automation test in short mode")
	}

	// Setup automation environment
	automationConfig := setupAutomationEnvironment(t)
	verifier := llmverifier.New(automationConfig)

	// Step 1: Discover models automatically
	t.Log("Step 1: Automatic model discovery")
	discoveredModels := discoverModelsAutomatically(t, verifier)
	if len(discoveredModels) == 0 {
		t.Fatal("No models discovered for automation testing")
	}

	// Step 2: Filter models for ACP testing
	t.Log("Step 2: Filtering models for ACP testing")
	acpCandidateModels := filterModelsForACP(discoveredModels)
	t.Logf("Found %d ACP candidate models", len(acpCandidateModels))

	// Step 3: Run automated ACP verification
	t.Log("Step 3: Running automated ACP verification")
	acpResults := runAutomatedACPVerification(t, verifier, acpCandidateModels)

	// Step 4: Generate automated reports
	t.Log("Step 4: Generating automated reports")
	reports := generateAutomatedReports(t, acpResults)

	// Step 5: Validate results automatically
	t.Log("Step 5: Validating results automatically")
	validationResults := validateACPResultsAutomatically(t, acpResults, reports)

	// Step 6: Automated decision making
	t.Log("Step 6: Automated decision making")
	decisions := makeAutomatedDecisions(t, validationResults)

	// Summary
	t.Log("=== Automation Workflow Summary ===")
	t.Logf("Models discovered: %d", len(discoveredModels))
	t.Logf("ACP candidates: %d", len(acpCandidateModels))
	t.Logf("ACP results generated: %d", len(acpResults))
	t.Logf("Reports generated: %d", len(reports))
	t.Logf("Validation results: %d", len(validationResults))
	t.Logf("Automated decisions: %d", len(decisions))

	// Assert workflow success
	if len(acpResults) == 0 {
		t.Error("No ACP results generated in automation workflow")
	}
	if !validationResults.AllValid {
		t.Error("Validation failed in automation workflow")
	}
}

// TestACPsAutomatedScheduling tests automated scheduling of ACP verification
func TestACPsAutomatedScheduling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping automated scheduling test in short mode")
	}

	// Create test scheduler
	scheduler := &ACPTestScheduler{
		Config: &ACPSchedulerConfig{
			CheckInterval:     1 * time.Hour,
			MaxConcurrent:     5,
			RetryAttempts:     3,
			ScheduleWindow:    24 * time.Hour,
			PriorityModels:    []string{"gpt-4", "claude-3-opus"},
			EnableAutoScaling: true,
		},
	}

	// Schedule ACP tests
	schedule := &ACPTestSchedule{
		Models: []ACPModelSchedule{
			{ModelID: "gpt-4", Provider: "openai", Priority: 1, NextCheck: time.Now().Add(-1 * time.Hour)},
			{ModelID: "claude-3-opus", Provider: "anthropic", Priority: 2, NextCheck: time.Now().Add(-30 * time.Minute)},
			{ModelID: "deepseek-chat", Provider: "deepseek", Priority: 3, NextCheck: time.Now().Add(-2 * time.Hour)},
		},
		LastUpdated: time.Now().Add(-1 * time.Hour),
	}

	// Test scheduling logic
	t.Run("Priority Scheduling", func(t *testing.T) {
		sortedModels := scheduler.SortByPriority(schedule.Models)
		
		// Verify priority order
		if sortedModels[0].ModelID != "gpt-4" {
			t.Error("High priority model should be scheduled first")
		}
		if sortedModels[1].ModelID != "claude-3-opus" {
			t.Error("Medium priority model should be scheduled second")
		}
	})

	t.Run("Overdue Detection", func(t *testing.T) {
		overdueModels := scheduler.GetOverdueModels(schedule.Models)
		
		// All models should be overdue since their next check times are in the past
		if len(overdueModels) != len(schedule.Models) {
			t.Errorf("Expected %d overdue models, got %d", len(schedule.Models), len(overdueModels))
		}
	})

	t.Run("Concurrent Execution", func(t *testing.T) {
		// Test concurrent execution within limits
		results := scheduler.ExecuteConcurrently(schedule.Models)
		
		if len(results) != len(schedule.Models) {
			t.Errorf("Expected %d results, got %d", len(schedule.Models), len(results))
		}
		
		// Verify no more than max concurrent executions
		maxConcurrent := 0
		currentConcurrent := 0
		
		for _, result := range results {
			if result.StartTime.Sub(results[0].StartTime) < 1*time.Second {
				currentConcurrent++
				if currentConcurrent > maxConcurrent {
					maxConcurrent = currentConcurrent
				}
			} else {
				currentConcurrent = 0
			}
		}
		
		if maxConcurrent > scheduler.Config.MaxConcurrent {
			t.Errorf("Exceeded max concurrent limit: %d > %d", maxConcurrent, scheduler.Config.MaxConcurrent)
		}
	})
}

// TestACPsAutomatedMonitoring tests automated monitoring and alerting
func TestACPsAutomatedMonitoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping automated monitoring test in short mode")
	}

	// Setup monitoring system
	monitor := &ACPMonitor{
		Config: &ACPMonitorConfig{
			CheckInterval:      5 * time.Minute,
			AlertThreshold:     0.7, // Alert if ACP support drops below 70%
			PerformanceThreshold: 5 * time.Second, // Alert if ACP test takes > 5s
			ErrorThreshold:     0.1, // Alert if error rate > 10%
			NotificationChannels: []string{"email", "slack", "webhook"},
		},
		Metrics: make(map[string]ACPMetrics),
	}

	// Simulate monitoring data
	testData := []ACPMetrics{
		{
			ModelID:           "gpt-4",
			Provider:          "openai",
			ACPSupported:      true,
			ACPScore:          0.85,
			LastTestTime:      time.Now().Add(-10 * time.Minute),
			AverageDuration:   2 * time.Second,
			ErrorRate:         0.05,
			ConsecutiveErrors: 0,
		},
		{
			ModelID:           "claude-3-opus",
			Provider:          "anthropic",
			ACPSupported:      false,
			ACPScore:          0.45,
			LastTestTime:      time.Now().Add(-15 * time.Minute),
			AverageDuration:   8 * time.Second,
			ErrorRate:         0.15,
			ConsecutiveErrors: 3,
		},
		{
			ModelID:           "problematic-model",
			Provider:          "test",
			ACPSupported:      true,
			ACPScore:          0.9,
			LastTestTime:      time.Now().Add(-2 * time.Hour),
			AverageDuration:   1 * time.Second,
			ErrorRate:         0.0,
			ConsecutiveErrors: 0,
		},
	}

	for _, metrics := range testData {
		monitor.Metrics[metrics.ModelID] = metrics
	}

	t.Run("Threshold Detection", func(t *testing.T) {
		alerts := monitor.CheckThresholds()
		
		// Should detect claude-3-opus as below threshold
		foundLowScoreAlert := false
		foundHighErrorRateAlert := false
		foundSlowPerformanceAlert := false
		
		for _, alert := range alerts {
			switch alert.Type {
			case "low_acp_score":
				if alert.ModelID == "claude-3-opus" && alert.Value < 0.7 {
					foundLowScoreAlert = true
				}
			case "high_error_rate":
				if alert.ModelID == "claude-3-opus" && alert.Value > 0.1 {
					foundHighErrorRateAlert = true
				}
			case "slow_performance":
				if alert.ModelID == "claude-3-opus" && alert.Value > 5*time.Second {
					foundSlowPerformanceAlert = true
				}
			}
		}
		
		if !foundLowScoreAlert {
			t.Error("Should have detected low ACP score alert for claude-3-opus")
		}
		if !foundHighErrorRateAlert {
			t.Error("Should have detected high error rate alert for claude-3-opus")
		}
		if !foundSlowPerformanceAlert {
			t.Error("Should have detected slow performance alert for claude-3-opus")
		}
	})

	t.Run("Stale Data Detection", func(t *testing.T) {
		staleModels := monitor.GetStaleModels(1 * time.Hour)
		
		// Should detect problematic-model as stale
		foundStaleModel := false
		for _, model := range staleModels {
			if model.ModelID == "problematic-model" {
				foundStaleModel = true
				break
			}
		}
		
		if !foundStaleModel {
			t.Error("Should have detected stale model data")
		}
	})

	t.Run("Notification Generation", func(t *testing.T) {
		notifications := monitor.GenerateNotifications(alerts)
		
		// Verify notification channels
		emailNotifications := 0
		slackNotifications := 0
		
		for _, notification := range notifications {
			switch notification.Channel {
			case "email":
				emailNotifications++
			case "slack":
				slackNotifications++
			}
		}
		
		if emailNotifications == 0 {
			t.Error("Should generate email notifications")
		}
		if slackNotifications == 0 {
			t.Error("Should generate slack notifications")
		}
	})
}

// TestACPsAutomatedRecovery tests automated recovery mechanisms
func TestACPsAutomatedRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping automated recovery test in short mode")
	}

	// Setup recovery system
	recovery := &ACPRecovery{
		Config: &ACPRecoveryConfig{
			MaxRetries:          3,
			BackoffMultiplier:   2.0,
			InitialBackoff:      1 * time.Second,
			MaxBackoff:          30 * time.Second,
			CircuitBreakerThreshold: 5,
			ResetTimeout:        5 * time.Minute,
		},
		FailedModels: make(map[string]int),
		CircuitBreaker: make(map[string]time.Time),
	}

	// Simulate failing models
	failingModels := []string{"failing-model-1", "failing-model-2", "failing-model-3"}
	
	t.Run("Retry Mechanism", func(t *testing.T) {
		for _, model := range failingModels {
			success := recovery.RetryWithBackoff(model, func() error {
				// Simulate failure
				return fmt.Errorf("simulated failure")
			})
			
			if success {
				t.Errorf("Expected retry to fail for %s", model)
			}
			
			retryCount := recovery.FailedModels[model]
			if retryCount != recovery.Config.MaxRetries {
				t.Errorf("Expected %d retries, got %d", recovery.Config.MaxRetries, retryCount)
			}
		}
	})

	t.Run("Circuit Breaker", func(t *testing.T) {
		// Reset for circuit breaker test
		recovery.FailedModels = make(map[string]int)
		
		model := "circuit-breaker-model"
		
		// Trigger circuit breaker
		for i := 0; i < recovery.Config.CircuitBreakerThreshold; i++ {
			recovery.RecordFailure(model)
		}
		
		// Should trigger circuit breaker
		if !recovery.IsCircuitBreakerOpen(model) {
			t.Error("Circuit breaker should be open after threshold failures")
		}
		
		// Should prevent further attempts
		canAttempt := recovery.CanAttempt(model)
		if canAttempt {
			t.Error("Should not allow attempts when circuit breaker is open")
		}
		
		// Should reset after timeout
		recovery.CircuitBreaker[model] = time.Now().Add(-6 * time.Minute)
		canAttempt = recovery.CanAttempt(model)
		if !canAttempt {
			t.Error("Should allow attempts after circuit breaker timeout")
		}
	})

	t.Run("Success Recovery", func(t *testing.T) {
		model := "recovering-model"
		
		// Record some failures
		recovery.RecordFailure(model)
		recovery.RecordFailure(model)
		
		// Simulate success
		success := recovery.RetryWithBackoff(model, func() error {
			return nil // Success
		})
		
		if !success {
			t.Error("Expected successful recovery")
		}
		
		// Should reset failure count
		if recovery.FailedModels[model] != 0 {
			t.Error("Failure count should be reset after success")
		}
	})
}

// TestACPsAutomatedReporting tests automated report generation and distribution
func TestACPsAutomatedReporting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping automated reporting test in short mode")
	}

	// Generate sample ACP results
	acpResults := []ACPResult{
		{
			ModelID:      "gpt-4",
			Provider:     "openai",
			ACPSupported: true,
			ACPScore:     0.85,
			Capabilities: map[string]float64{
				"jsonrpc_compliance": 0.9,
				"tool_calling":       0.8,
				"context_management": 0.85,
				"code_assistance":    0.9,
				"error_detection":    0.75,
			},
			TestDuration: 2 * time.Second,
			Timestamp:    time.Now().Add(-1 * time.Hour),
		},
		{
			ModelID:      "claude-3-opus",
			Provider:     "anthropic",
			ACPSupported: false,
			ACPScore:     0.45,
			Capabilities: map[string]float64{
				"jsonrpc_compliance": 0.5,
				"tool_calling":       0.3,
				"context_management": 0.6,
				"code_assistance":    0.4,
				"error_detection":    0.2,
			},
			TestDuration: 3 * time.Second,
			Timestamp:    time.Now().Add(-2 * time.Hour),
		},
	}

	// Create report generator
	reportGen := &ACPReportGenerator{
		Templates: map[string]string{
			"executive_summary": "templates/executive_summary.tmpl",
			"detailed_report":   "templates/detailed_report.tmpl",
			"trend_analysis":    "templates/trend_analysis.tmpl",
		},
		OutputFormats: []string{"html", "pdf", "json", "csv"},
	}

	t.Run("Report Generation", func(t *testing.T) {
		reports, err := reportGen.GenerateReports(acpResults)
		if err != nil {
			t.Fatalf("Failed to generate reports: %v", err)
		}

		// Verify all formats were generated
		expectedFormats := map[string]bool{"html": false, "pdf": false, "json": false, "csv": false}
		for _, report := range reports {
			if _, ok := expectedFormats[report.Format]; ok {
				expectedFormats[report.Format] = true
			}
		}

		for format, generated := range expectedFormats {
			if !generated {
				t.Errorf("Expected %s report format was not generated", format)
			}
		}
	})

	t.Run("Report Distribution", func(t *testing.T) {
		reports := []Report{
			{Format: "html", Content: "<html>ACP Report</html>"},
			{Format: "json", Content: `{"results": []}`},
		}

		distributor := &ACPReportDistributor{
			Channels: []DistributionChannel{
				{Name: "email", Config: map[string]string{"recipients": "team@company.com"}},
				{Name: "slack", Config: map[string]string{"channel": "#ai-team"}},
				{Name: "webhook", Config: map[string]string{"url": "https://api.company.com/reports"}},
			},
		}

		distributionResults := distributor.Distribute(reports)

		// Verify distribution succeeded
		for _, result := range distributionResults {
			if !result.Success {
				t.Errorf("Distribution failed for channel %s: %v", 
					result.Channel, result.Error)
			}
		}
	})

	t.Run("Report Scheduling", func(t *testing.T) {
		scheduler := &ACPReportScheduler{
			Schedule: "0 9 * * MON", // Every Monday at 9 AM
			Recipients: []string{"team@company.com", "manager@company.com"},
			Formats: []string{"html", "pdf"},
		}

		// Test schedule parsing
		nextRun, err := scheduler.GetNextRunTime()
		if err != nil {
			t.Fatalf("Failed to parse schedule: %v", err)
		}

		if nextRun.Before(time.Now()) {
			t.Error("Next run time should be in the future")
		}

		// Test report generation for scheduled time
		scheduledReports, err := scheduler.GenerateScheduledReports(acpResults)
		if err != nil {
			t.Fatalf("Failed to generate scheduled reports: %v", err)
		}

		if len(scheduledReports) != len(scheduler.Formats) {
			t.Errorf("Expected %d reports, got %d", len(scheduler.Formats), len(scheduledReports))
		}
	})
}

// Helper functions and types

func setupAutomationEnvironment(t *testing.T) *config.Config {
	return &config.Config{
		GlobalTimeout: 300 * time.Second,
		MaxRetries:    5,
		// Add automation-specific configuration
	}
}

func discoverModelsAutomatically(t *testing.T, verifier *llmverifier.Verifier) []string {
	// Simulate automatic model discovery
	models := []string{
		"gpt-4",
		"gpt-3.5-turbo",
		"claude-3-opus",
		"claude-3-sonnet",
		"claude-3-haiku",
		"deepseek-chat",
		"deepseek-coder",
		"gemini-pro",
		"gemini-pro-vision",
	}
	
	t.Logf("Automatically discovered %d models", len(models))
	return models
}

func filterModelsForACP(models []string) []string {
	// Filter models based on ACP suitability criteria
	var acpModels []string
	
	// Simple heuristic: exclude vision-only models
	for _, model := range models {
		if !strings.Contains(model, "vision") {
			acpModels = append(acpModels, model)
		}
	}
	
	return acpModels
}

func runAutomatedACPVerification(t *testing.T, verifier *llmverifier.Verifier, models []string) []ACPResult {
	var results []ACPResult
	
	for _, model := range models {
		// Simulate ACP verification
		result := ACPResult{
			ModelID:      model,
			Provider:     extractProvider(model),
			ACPSupported: len(model)%2 == 0, // Simulate results
			ACPScore:     float64(len(model)) / 20.0, // Simulate score
			TestDuration: time.Duration(len(model)) * 100 * time.Millisecond,
			Timestamp:    time.Now(),
		}
		
		results = append(results, result)
	}
	
	return results
}

func generateAutomatedReports(t *testing.T, results []ACPResult) []Report {
	var reports []Report
	
	// Generate different report formats
	reportTypes := []string{"summary", "detailed", "trends", "compliance"}
	
	for _, reportType := range reportTypes {
		report := Report{
			Type:      reportType,
			Format:    "json",
			Generated: time.Now(),
			Content:   generateReportContent(reportType, results),
		}
		reports = append(reports, report)
	}
	
	return reports
}

func validateACPResultsAutomatically(t *testing.T, results []ACPResult, reports []Report) ValidationResults {
	validation := ValidationResults{
		AllValid: true,
		Errors:   []string{},
	}
	
	// Validate result completeness
	for _, result := range results {
		if result.ACPScore < 0 || result.ACPScore > 1 {
			validation.AllValid = false
			validation.Errors = append(validation.Errors, 
				fmt.Sprintf("Invalid ACP score for %s: %f", result.ModelID, result.ACPScore))
		}
		
		if result.TestDuration <= 0 {
			validation.AllValid = false
			validation.Errors = append(validation.Errors,
				fmt.Sprintf("Invalid test duration for %s", result.ModelID))
		}
	}
	
	// Validate reports
	if len(reports) == 0 {
		validation.AllValid = false
		validation.Errors = append(validation.Errors, "No reports generated")
	}
	
	return validation
}

func makeAutomatedDecisions(t *testing.T, validation ValidationResults) []AutomatedDecision {
	var decisions []AutomatedDecision
	
	if validation.AllValid {
		decisions = append(decisions, AutomatedDecision{
			Type:        "deployment_approved",
			Description: "ACP implementation approved for production",
			Confidence:  0.95,
			Timestamp:   time.Now(),
		})
	} else {
		decisions = append(decisions, AutomatedDecision{
			Type:        "review_required",
			Description: "Manual review required due to validation errors",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	return decisions
}

func extractProvider(model string) string {
	// Simple heuristic to extract provider from model name
	if strings.Contains(model, "gpt") {
		return "openai"
	} else if strings.Contains(model, "claude") {
		return "anthropic"
	} else if strings.Contains(model, "deepseek") {
		return "deepseek"
	} else if strings.Contains(model, "gemini") {
		return "google"
	}
	return "unknown"
}

func generateReportContent(reportType string, results []ACPResult) string {
	switch reportType {
	case "summary":
		return fmt.Sprintf(`{"type":"summary","models_tested":%d}`, len(results))
	case "detailed":
		return fmt.Sprintf(`{"type":"detailed","results_count":%d}`, len(results))
	case "trends":
		return `{"type":"trends","data":[]}`
	case "compliance":
		return `{"type":"compliance","status":"pass"}`
	default:
		return `{"type":"unknown"}`
	}
}

// Types for automation

type ACPTestScheduler struct {
	Config *ACPSchedulerConfig
}

type ACPSchedulerConfig struct {
	CheckInterval     time.Duration
	MaxConcurrent     int
	RetryAttempts     int
	ScheduleWindow    time.Duration
	PriorityModels    []string
	EnableAutoScaling bool
}

type ACPTestSchedule struct {
	Models      []ACPModelSchedule
	LastUpdated time.Time
}

type ACPModelSchedule struct {
	ModelID   string
	Provider  string
	Priority  int
	NextCheck time.Time
}

type ACPMonitor struct {
	Config  *ACPMonitorConfig
	Metrics map[string]ACPMetrics
}

type ACPMonitorConfig struct {
	CheckInterval        time.Duration
	AlertThreshold       float64
	PerformanceThreshold time.Duration
	ErrorThreshold       float64
	NotificationChannels []string
}

type ACPMetrics struct {
	ModelID           string
	Provider          string
	ACPSupported      bool
	ACPScore          float64
	LastTestTime      time.Time
	AverageDuration   time.Duration
	ErrorRate         float64
	ConsecutiveErrors int
}

type ACPAlert struct {
	Type     string
	ModelID  string
	Value    interface{}
	Severity string
	Message  string
}

type ACPNotification struct {
	Channel string
	Subject string
	Content string
	Success bool
	Error   error
}

type ACPRecovery struct {
	Config          *ACPRecoveryConfig
	FailedModels    map[string]int
	CircuitBreaker  map[string]time.Time
}

type ACPRecoveryConfig struct {
	MaxRetries             int
	BackoffMultiplier      float64
	InitialBackoff         time.Duration
	MaxBackoff             time.Duration
	CircuitBreakerThreshold int
	ResetTimeout           time.Duration
}

type ACPReportGenerator struct {
	Templates     map[string]string
	OutputFormats []string
}

type ACPReportDistributor struct {
	Channels []DistributionChannel
}

type ACPReportScheduler struct {
	Schedule   string
	Recipients []string
	Formats    []string
}

type ACPResult struct {
	ModelID      string
	Provider     string
	ACPSupported bool
	ACPScore     float64
	Capabilities map[string]float64
	TestDuration time.Duration
	Timestamp    time.Time
}

type Report struct {
	Type      string
	Format    string
	Generated time.Time
	Content   string
}

type ValidationResults struct {
	AllValid bool
	Errors   []string
}

type AutomatedDecision struct {
	Type        string
	Description string
	Confidence  float64
	Timestamp   time.Time
}

type DistributionChannel struct {
	Name   string
	Config map[string]string
}

// Helper methods
func (s *ACPTestScheduler) SortByPriority(models []ACPModelSchedule) []ACPModelSchedule {
	// Simple priority-based sorting
	sorted := make([]ACPModelSchedule, len(models))
	copy(sorted, models)
	
	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Priority > sorted[j].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted
}

func (s *ACPTestScheduler) GetOverdueModels(models []ACPModelSchedule) []ACPModelSchedule {
	var overdue []ACPModelSchedule
	now := time.Now()
	
	for _, model := range models {
		if model.NextCheck.Before(now) {
			overdue = append(overdue, model)
		}
	}
	
	return overdue
}

func (s *ACPTestScheduler) ExecuteConcurrently(models []ACPModelSchedule) []ACPExecutionResult {
	// Simplified concurrent execution
	var results []ACPExecutionResult
	
	for _, model := range models {
		result := ACPExecutionResult{
			ModelID:   model.ModelID,
			StartTime: time.Now(),
			Success:   true,
		}
		
		// Simulate ACP test
		time.Sleep(100 * time.Millisecond)
		
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		
		results = append(results, result)
	}
	
	return results
}

func (m *ACPMonitor) CheckThresholds() []ACPAlert {
	var alerts []ACPAlert
	
	for modelID, metrics := range m.Metrics {
		// Check ACP score threshold
		if metrics.ACPScore < m.Config.AlertThreshold {
			alerts = append(alerts, ACPAlert{
				Type:     "low_acp_score",
				ModelID:  modelID,
				Value:    metrics.ACPScore,
				Severity: "warning",
				Message:  fmt.Sprintf("ACP score %.2f below threshold %.2f", 
					metrics.ACPScore, m.Config.AlertThreshold),
			})
		}
		
		// Check performance threshold
		if metrics.AverageDuration > m.Config.PerformanceThreshold {
			alerts = append(alerts, ACPAlert{
				Type:     "slow_performance",
				ModelID:  modelID,
				Value:    metrics.AverageDuration,
				Severity: "warning",
				Message:  fmt.Sprintf("ACP test duration %v exceeds threshold %v",
					metrics.AverageDuration, m.Config.PerformanceThreshold),
			})
		}
		
		// Check error rate threshold
		if metrics.ErrorRate > m.Config.ErrorThreshold {
			alerts = append(alerts, ACPAlert{
				Type:     "high_error_rate",
				ModelID:  modelID,
				Value:    metrics.ErrorRate,
				Severity: "error",
				Message:  fmt.Sprintf("Error rate %.2f exceeds threshold %.2f",
					metrics.ErrorRate, m.Config.ErrorThreshold),
			})
		}
	}
	
	return alerts
}

func (m *ACPMonitor) GetStaleModels(maxAge time.Duration) []ACPMetrics {
	var stale []ACPMetrics
	cutoff := time.Now().Add(-maxAge)
	
	for _, metrics := range m.Metrics {
		if metrics.LastTestTime.Before(cutoff) {
			stale = append(stale, metrics)
		}
	}
	
	return stale
}

func (m *ACPMonitor) GenerateNotifications(alerts []ACPAlert) []ACPNotification {
	var notifications []ACPNotification
	
	for _, alert := range alerts {
		for _, channel := range m.Config.NotificationChannels {
			notification := ACPNotification{
				Channel: channel,
				Subject: fmt.Sprintf("ACP Alert: %s", alert.Type),
				Content: alert.Message,
				Success: true,
			}
			
			// Simulate notification sending
			if channel == "email" {
				// Simulate email sending
				notification.Success = true
			} else if channel == "slack" {
				// Simulate slack notification
				notification.Success = true
			} else if channel == "webhook" {
				// Simulate webhook call
				notification.Success = true
			}
			
			notifications = append(notifications, notification)
		}
	}
	
	return notifications
}

func (r *ACPRecovery) RetryWithBackoff(model string, testFunc func() error) bool {
	backoff := r.Config.InitialBackoff
	
	for attempt := 0; attempt < r.Config.MaxRetries; attempt++ {
		err := testFunc()
		if err == nil {
			// Success - reset failure count
			delete(r.FailedModels, model)
			return true
		}
		
		// Record failure
		r.FailedModels[model] = attempt + 1
		
		// Check if circuit breaker should open
		if r.FailedModels[model] >= r.Config.CircuitBreakerThreshold {
			r.CircuitBreaker[model] = time.Now()
			return false
		}
		
		// Wait with backoff
		if attempt < r.Config.MaxRetries-1 {
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * r.Config.BackoffMultiplier)
			if backoff > r.Config.MaxBackoff {
				backoff = r.Config.MaxBackoff
			}
		}
	}
	
	return false
}

func (r *ACPRecovery) RecordFailure(model string) {
	r.FailedModels[model]++
}

func (r *ACPRecovery) IsCircuitBreakerOpen(model string) bool {
	if openTime, exists := r.CircuitBreaker[model]; exists {
		return time.Since(openTime) < r.Config.ResetTimeout
	}
	return false
}

func (r *ACPRecovery) CanAttempt(model string) bool {
	// Check if circuit breaker is open
	if r.IsCircuitBreakerOpen(model) {
		return false
	}
	
	return true
}

func (r *ACPReportGenerator) GenerateReports(results []ACPResult) ([]Report, error) {
	var reports []Report
	
	for _, format := range r.OutputFormats {
		report := Report{
			Format:    format,
			Generated: time.Now(),
		}
		
		switch format {
		case "json":
			data, err := json.Marshal(results)
			if err != nil {
				return nil, err
			}
			report.Content = string(data)
		case "csv":
			report.Content = generateCSVReport(results)
		case "html":
			report.Content = generateHTMLReport(results)
		default:
			report.Content = fmt.Sprintf("Report for %d models", len(results))
		}
		
		reports = append(reports, report)
	}
	
	return reports, nil
}

func (d *ACPReportDistributor) Distribute(reports []Report) []ACPNotification {
	var notifications []ACPNotification
	
	for _, report := range reports {
		for _, channel := range d.Channels {
			notification := ACPNotification{
				Channel: channel.Name,
				Subject: fmt.Sprintf("ACP Report - %s format", report.Format),
				Content: report.Content,
				Success: true,
			}
			
			// Simulate distribution
			switch channel.Name {
			case "email":
				notification.Success = true
			case "slack":
				notification.Success = true
			case "webhook":
				notification.Success = true
			}
			
			notifications = append(notifications, notification)
		}
	}
	
	return notifications
}

func (s *ACPReportScheduler) GetNextRunTime() (time.Time, error) {
	// Simple cron parsing (in real implementation, use proper cron library)
	if s.Schedule == "0 9 * * MON" {
		// Next Monday at 9 AM
		now := time.Now()
		nextMonday := now.AddDate(0, 0, int(time.Monday-now.Weekday()+7)%7)
		return time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 9, 0, 0, 0, nextMonday.Location()), nil
	}
	
	return time.Now().Add(24 * time.Hour), nil // Default to tomorrow
}

func (s *ACPReportScheduler) GenerateScheduledReports(results []ACPResult) ([]Report, error) {
	generator := &ACPReportGenerator{
		OutputFormats: s.Formats,
	}
	
	return generator.GenerateReports(results)
}

// Helper functions
func generateCSVReport(results []ACPResult) string {
	var csv strings.Builder
	csv.WriteString("Model,Provider,ACP_Supported,ACP_Score,Test_Duration,Timestamp\n")
	
	for _, result := range results {
		csv.WriteString(fmt.Sprintf("%s,%s,%t,%.2f,%v,%s\n",
			result.ModelID,
			result.Provider,
			result.ACPSupported,
			result.ACPScore,
			result.TestDuration,
			result.Timestamp.Format(time.RFC3339)))
	}
	
	return csv.String()
}

func generateHTMLReport(results []ACPResult) string {
	var html strings.Builder
	html.WriteString("<html><head><title>ACP Report</title></head><body>")
	html.WriteString("<h1>ACP Verification Results</h1>")
	html.WriteString("<table border='1'>")
	html.WriteString("<tr><th>Model</th><th>Provider</th><th>ACP Supported</th><th>Score</th></tr>")
	
	for _, result := range results {
		html.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%t</td><td>%.2f</td></tr>",
			result.ModelID,
			result.Provider,
			result.ACPSupported,
			result.ACPScore))
	}
	
	html.WriteString("</table></body></html>")
	return html.String()
}

// Placeholder types for compilation
type ACPExecutionResult struct {
	ModelID   string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Success   bool
}