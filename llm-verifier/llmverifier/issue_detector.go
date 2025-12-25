package llmverifier

import (
	"fmt"
	"log"
	"strings"
	"time"

	"llm-verifier/database"
	"llm-verifier/events"
)

// IssueDetector automatically detects issues from verification results
type IssueDetector struct {
	db           *database.Database
	eventManager *events.EventManager
}

// NewIssueDetector creates a new issue detector
func NewIssueDetector(db *database.Database, eventManager *events.EventManager) *IssueDetector {
	return &IssueDetector{
		db:           db,
		eventManager: eventManager,
	}
}

// AnalyzeVerificationResult analyzes a verification result and detects issues
func (id *IssueDetector) AnalyzeVerificationResult(result *VerificationResult) error {
	issues := id.detectIssues(result)

	for _, issue := range issues {
		if err := id.createOrUpdateIssue(issue, result); err != nil {
			log.Printf("Failed to create/update issue: %v", err)
		}
	}

	return nil
}

// detectIssues analyzes verification results and returns detected issues
func (id *IssueDetector) detectIssues(result *VerificationResult) []*database.Issue {
	var issues []*database.Issue

	// Issue 1: Very low overall score
	if result.PerformanceScores.OverallScore < 30 {
		now := time.Now()
		issues = append(issues, &database.Issue{
			ModelID:          0, // Will be set when creating
			IssueType:        "performance",
			Severity:         "critical",
			Title:            "Severely Underperforming Model",
			Description:      "Model has critically low performance scores",
			AffectedFeatures: []string{"all"},
			FirstDetected:    now,
			LastOccurred:     &now,
		})
	}

	// Issue 2: Code capability issues
	if result.PerformanceScores.CodeCapability < 40 {
		now := time.Now()
		issues = append(issues, &database.Issue{
			ModelID:          0,
			IssueType:        "capability",
			Severity:         "high",
			Title:            "Poor Code Generation Capability",
			Description:      "Model struggles significantly with code-related tasks",
			AffectedFeatures: []string{"code_generation", "code_review", "code_explanation"},
			FirstDetected:    now,
			LastOccurred:     &now,
		})
	}

	// Issue 3: High error rates
	if result.PerformanceScores.Reliability < 50 {
		issues = append(issues, &database.Issue{
			ModelID:          0,
			IssueType:        "reliability",
			Severity:         "high",
			Title:            "Unreliable Model Responses",
			Description:      "Model exhibits high error rates or inconsistent behavior",
			AffectedFeatures: []string{"reliability"},
		})
	}

	// Issue 4: Slow response times
	if result.PerformanceScores.Responsiveness < 60 {
		issues = append(issues, &database.Issue{
			ModelID:          0,
			IssueType:        "performance",
			Severity:         "medium",
			Title:            "Slow Response Times",
			Description:      "Model has slow response times affecting user experience",
			AffectedFeatures: []string{"responsiveness"},
		})
	}

	// Issue 5: Feature detection failures
	if !result.FeatureDetection.ToolUse && !result.FeatureDetection.FunctionCalling {
		issues = append(issues, &database.Issue{
			ModelID:          0,
			IssueType:        "capability",
			Severity:         "medium",
			Title:            "Missing Tool Use Capabilities",
			Description:      "Model does not support tool use or function calling",
			AffectedFeatures: []string{"tool_use", "function_calling"},
		})
	}

	// Issue 6: Network/API issues
	if strings.Contains(strings.ToLower(result.Error), "timeout") ||
		strings.Contains(strings.ToLower(result.Error), "connection") {
		issues = append(issues, &database.Issue{
			ModelID:          0,
			IssueType:        "connectivity",
			Severity:         "high",
			Title:            "Network Connectivity Issues",
			Description:      "Model experiences network or API connectivity problems",
			AffectedFeatures: []string{"connectivity"},
		})
	}

	// Issue 7: Authentication failures
	if strings.Contains(strings.ToLower(result.Error), "auth") ||
		strings.Contains(strings.ToLower(result.Error), "unauthorized") {
		issues = append(issues, &database.Issue{
			ModelID:          0,
			IssueType:        "authentication",
			Severity:         "critical",
			Title:            "Authentication Problems",
			Description:      "Model has authentication or authorization issues",
			AffectedFeatures: []string{"authentication"},
		})
	}

	return issues
}

// createOrUpdateIssue creates a new issue or updates an existing one
func (id *IssueDetector) createOrUpdateIssue(issue *database.Issue, result *VerificationResult) error {
	// Set model ID from verification result
	// This would need to be passed in or looked up
	// For now, we'll assume it's set elsewhere

	now := time.Now()
	issue.FirstDetected = now
	issue.LastOccurred = &now

	// Check if similar issue already exists
	existingIssues, err := id.db.ListIssues(map[string]interface{}{
		"model_id":   issue.ModelID,
		"issue_type": issue.IssueType,
		"title":      issue.Title,
	})

	if err == nil && len(existingIssues) > 0 {
		// Update existing issue
		existing := existingIssues[0]
		now := time.Now()
		existing.LastOccurred = &now
		existing.UpdatedAt = time.Now()

		return id.db.UpdateIssue(existing)
	}

	// Create new issue
	issue.CreatedAt = time.Now()
	issue.UpdatedAt = time.Now()

	if err := id.db.CreateIssue(issue); err != nil {
		return err
	}

	// Publish event about new issue
	event := events.CreateEventWithDetails(
		events.EventIssueDetected,
		id.mapSeverityToEventSeverity(issue.Severity),
		fmt.Sprintf("Issue Detected: %s", issue.Title),
		issue.Description,
		map[string]interface{}{
			"model_id":          issue.ModelID,
			"issue_type":        issue.IssueType,
			"severity":          issue.Severity,
			"affected_features": issue.AffectedFeatures,
		},
	)

	return id.eventManager.PublishEvent(event)
}

// mapSeverityToEventSeverity converts issue severity to event severity
func (id *IssueDetector) mapSeverityToEventSeverity(issueSeverity string) events.Severity {
	switch strings.ToLower(issueSeverity) {
	case "critical":
		return events.SeverityCritical
	case "high":
		return events.SeverityError
	case "medium":
		return events.SeverityWarning
	case "low":
		return events.SeverityInfo
	default:
		return events.SeverityInfo
	}
}

// AnalyzeMultipleResults analyzes multiple verification results and detects patterns
func (id *IssueDetector) AnalyzeMultipleResults(results []*VerificationResult) error {
	// Group results by model ID (assuming we can derive model ID from results)
	modelResults := make(map[string][]*VerificationResult)

	for _, result := range results {
		if result.Error == "" { // Only analyze successful results
			modelKey := result.ModelInfo.ID // Use model ID string as key
			modelResults[modelKey] = append(modelResults[modelKey], result)
		}
	}

	// Analyze patterns for each model
	for modelKey, modelResults := range modelResults {
		if err := id.analyzeModelPatterns(modelKey, modelResults); err != nil {
			log.Printf("Failed to analyze patterns for model %s: %v", modelKey, err)
		}
	}

	return nil
}

// analyzeModelPatterns analyzes patterns in multiple results for a model
func (id *IssueDetector) analyzeModelPatterns(modelKey string, results []*VerificationResult) error {
	if len(results) < 3 {
		return nil // Need at least 3 results for pattern analysis
	}

	// For now, skip pattern analysis as we need model ID mapping
	// This would require looking up the actual model ID from the database
	// based on the model key (name/ID string)

	return nil
}

// isDecliningTrend checks if scores show a declining trend
func (id *IssueDetector) isDecliningTrend(scores []float64) bool {
	if len(scores) < 3 {
		return false
	}

	// Simple linear regression to detect downward trend
	n := float64(len(scores))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, score := range scores {
		x := float64(i)
		sumX += x
		sumY += score
		sumXY += x * score
		sumX2 += x * x
	}

	// Calculate slope
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Negative slope indicates declining trend
	return slope < -2.0 // Threshold for significant decline
}
