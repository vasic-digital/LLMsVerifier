package enhanced

import (
	"encoding/json"
	"fmt"
	"time"

	"llm-verifier/database"
)

// IssueManager manages issues and problems with LLM models
type IssueManager struct {
	db *database.Database
}

// NewIssueManager creates a new issue manager
func NewIssueManager(db *database.Database) *IssueManager {
	return &IssueManager{
		db: db,
	}
}

// IssueSeverity represents the severity level of an issue
type IssueSeverity string

const (
	SeverityCritical IssueSeverity = "critical"
	SeverityHigh     IssueSeverity = "high"
	SeverityMedium   IssueSeverity = "medium"
	SeverityLow      IssueSeverity = "low"
)

// IssueType represents the type of issue
type IssueType string

const (
	IssueTypeAvailability  IssueType = "availability"
	IssueTypePerformance   IssueType = "performance"
	IssueTypeAccuracy      IssueType = "accuracy"
	IssueTypeSecurity      IssueType = "security"
	IssueTypeCompliance    IssueType = "compliance"
	IssueTypeCost          IssueType = "cost"
	IssueTypeCompatibility IssueType = "compatibility"
)

// IssueTemplate represents a template for common issues
type IssueTemplate struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	IssueType   IssueType     `json:"issue_type"`
	Severity    IssueSeverity `json:"severity"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Symptoms    []string      `json:"symptoms"`
	Workarounds []string      `json:"workarounds"`
}

// Common issue templates
var IssueTemplates = []IssueTemplate{
	{
		ID:          "high_latency",
		Name:        "High Latency",
		IssueType:   IssueTypePerformance,
		Severity:    SeverityMedium,
		Title:       "Model showing consistently high response times",
		Description: "The model is taking significantly longer than expected to respond to requests",
		Symptoms: []string{
			"Response times consistently above 5 seconds",
			"Timeouts occurring frequently",
			"Performance degradation compared to baseline",
		},
		Workarounds: []string{
			"Implement request timeout with retry logic",
			"Use smaller context windows to reduce processing time",
			"Consider using a different model for time-sensitive operations",
		},
	},
	{
		ID:          "rate_limit_exceeded",
		Name:        "Rate Limit Exceeded",
		IssueType:   IssueTypeAvailability,
		Severity:    SeverityHigh,
		Title:       "Rate limit being exceeded frequently",
		Description: "The model is hitting rate limits, causing requests to be rejected",
		Symptoms: []string{
			"HTTP 429 errors occurring frequently",
			"Requests being rejected due to rate limits",
			"API quota being exhausted quickly",
		},
		Workarounds: []string{
			"Implement exponential backoff retry logic",
			"Reduce request frequency",
			"Upgrade to higher tier plan if available",
			"Implement request queuing",
		},
	},
	{
		ID:          "model_unresponsive",
		Name:        "Model Unresponsive",
		IssueType:   IssueTypeAvailability,
		Severity:    SeverityCritical,
		Title:       "Model not responding to requests",
		Description: "The model is completely unresponsive or returning errors for all requests",
		Symptoms: []string{
			"All requests timing out",
			"HTTP 5xx errors for all requests",
			"Model status showing as unavailable",
		},
		Workarounds: []string{
			"Check provider status page for outages",
			"Switch to backup model if available",
			"Contact provider support",
			"Monitor for recovery and retry later",
		},
	},
	{
		ID:          "accuracy_degradation",
		Name:        "Accuracy Degradation",
		IssueType:   IssueTypeAccuracy,
		Severity:    SeverityMedium,
		Title:       "Model accuracy has degraded",
		Description: "The model's responses are less accurate or relevant than expected",
		Symptoms: []string{
			"Lower accuracy scores in verification results",
			"Increased irrelevant or incorrect responses",
			"User complaints about response quality",
		},
		Workarounds: []string{
			"Improve prompt engineering",
			"Use more specific instructions",
			"Consider fine-tuning if available",
			"Switch to a more capable model",
		},
	},
	{
		ID:          "security_concern",
		Name:        "Security Concern",
		IssueType:   IssueTypeSecurity,
		Severity:    SeverityHigh,
		Title:       "Potential security vulnerability detected",
		Description: "The model may be vulnerable to prompt injection or other security issues",
		Symptoms: []string{
			"Model revealing sensitive information",
			"Prompt injection attacks succeeding",
			"Unauthorized access attempts",
		},
		Workarounds: []string{
			"Implement input validation and sanitization",
			"Use secure prompt templates",
			"Monitor for suspicious activity",
			"Implement content filtering",
		},
	},
	{
		ID:          "cost_spike",
		Name:        "Cost Spike",
		IssueType:   IssueTypeCost,
		Severity:    SeverityMedium,
		Title:       "Unexpected increase in usage costs",
		Description: "The model's usage costs have increased significantly without corresponding usage increase",
		Symptoms: []string{
			"Higher than expected billing charges",
			"Cost per request increasing",
			"Budget alerts being triggered",
		},
		Workarounds: []string{
			"Review and optimize request patterns",
			"Implement cost monitoring",
			"Consider switching to more cost-effective model",
			"Set up budget alerts",
		},
	},
}

// CreateIssueFromTemplate creates an issue from a template
func (im *IssueManager) CreateIssueFromTemplate(modelID int64, templateID string, verificationResultID *int64, additionalDetails map[string]interface{}) error {
	// Find the template
	var template *IssueTemplate
	for _, t := range IssueTemplates {
		if t.ID == templateID {
			template = &t
			break
		}
	}

	if template == nil {
		return fmt.Errorf("issue template not found: %s", templateID)
	}

	// Create issue from template
	symptomsJSON, _ := json.Marshal(template.Symptoms)
	workaroundsJSON, _ := json.Marshal(template.Workarounds)

	issue := &database.Issue{
		ModelID:              modelID,
		IssueType:            string(template.IssueType),
		Severity:             string(template.Severity),
		Title:                template.Title,
		Description:          template.Description,
		Symptoms:             scanNullableStringFromBytes(symptomsJSON),
		Workarounds:          scanNullableStringFromBytes(workaroundsJSON),
		AffectedFeatures:     template.Symptoms, // Use symptoms as affected features
		VerificationResultID: verificationResultID,
	}

	// Add additional details if provided
	if len(additionalDetails) > 0 {
		detailsJSON, _ := json.Marshal(additionalDetails)
		if issue.Symptoms == nil {
			issue.Symptoms = scanNullableStringFromBytes(detailsJSON)
		}
	}

	return im.db.CreateIssue(issue)
}

// CreateCustomIssue creates a custom issue
func (im *IssueManager) CreateCustomIssue(modelID int64, issueType IssueType, severity IssueSeverity, title, description string, symptoms, workarounds []string, affectedFeatures []string, verificationResultID *int64) error {
	symptomsJSON, _ := json.Marshal(symptoms)
	workaroundsJSON, _ := json.Marshal(workarounds)

	issue := &database.Issue{
		ModelID:              modelID,
		IssueType:            string(issueType),
		Severity:             string(severity),
		Title:                title,
		Description:          description,
		Symptoms:             scanNullableStringFromBytes(symptomsJSON),
		Workarounds:          scanNullableStringFromBytes(workaroundsJSON),
		AffectedFeatures:     affectedFeatures,
		VerificationResultID: verificationResultID,
	}

	return im.db.CreateIssue(issue)
}

// AutoDetectIssues automatically detects issues based on verification results
func (im *IssueManager) AutoDetectIssues(verificationResult *database.VerificationResult) error {
	var issuesDetected []string

	// Check for availability issues
	if verificationResult.ModelExists != nil && !*verificationResult.ModelExists {
		issuesDetected = append(issuesDetected, "model_not_found")
	}

	if verificationResult.Responsive != nil && !*verificationResult.Responsive {
		issuesDetected = append(issuesDetected, "model_unresponsive")
	}

	if verificationResult.Overloaded != nil && *verificationResult.Overloaded {
		issuesDetected = append(issuesDetected, "model_overloaded")
	}

	// Check for performance issues
	if verificationResult.LatencyMs != nil && *verificationResult.LatencyMs > 5000 {
		issuesDetected = append(issuesDetected, "high_latency")
	}

	// Check for accuracy issues
	if verificationResult.OverallScore < 30.0 {
		issuesDetected = append(issuesDetected, "accuracy_degradation")
	}

	// Check for specific capability issues
	if !verificationResult.SupportsCodeGeneration && verificationResult.CodeCapabilityScore > 0 {
		issuesDetected = append(issuesDetected, "code_capability_issue")
	}

	// Create issues for detected problems
	for _, issueID := range issuesDetected {
		if err := im.CreateIssueFromTemplate(verificationResult.ModelID, issueID, &verificationResult.ID, nil); err != nil {
			// Log error but continue detecting other issues
			fmt.Printf("Warning: Failed to create issue %s: %v\n", issueID, err)
		}
	}

	return nil
}

// UpdateIssueStatus updates the status of an issue
func (im *IssueManager) UpdateIssueStatus(issueID int64, resolved bool, resolutionNotes string) error {
	issue, err := im.db.GetIssue(issueID)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	now := time.Now()
	if resolved {
		issue.ResolvedAt = &now
		issue.ResolutionNotes = &resolutionNotes
	} else {
		issue.ResolvedAt = nil
		issue.ResolutionNotes = nil
	}

	return im.db.UpdateIssue(issue)
}

// GetIssuesForModel gets all issues for a specific model
func (im *IssueManager) GetIssuesForModel(modelID int64, includeResolved bool) ([]*database.Issue, error) {
	filters := map[string]interface{}{
		"model_id": modelID,
	}

	if !includeResolved {
		filters["resolved"] = false
	}

	return im.db.ListIssues(filters)
}

// GetCriticalIssues gets all critical issues across all models
func (im *IssueManager) GetCriticalIssues() ([]*database.Issue, error) {
	filters := map[string]interface{}{
		"severity": string(SeverityCritical),
		"resolved": false,
	}

	return im.db.ListIssues(filters)
}

// GetIssueStatistics gets statistics about issues
func (im *IssueManager) GetIssueStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total issues by severity
	severities := []IssueSeverity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
	severityStats := make(map[string]int)

	for _, severity := range severities {
		filters := map[string]interface{}{"severity": string(severity)}
		issues, err := im.db.ListIssues(filters)
		if err != nil {
			return nil, fmt.Errorf("failed to get issues for severity %s: %w", severity, err)
		}
		severityStats[string(severity)] = len(issues)
	}

	stats["by_severity"] = severityStats

	// Total issues by type
	issueTypes := []IssueType{
		IssueTypeAvailability, IssueTypePerformance, IssueTypeAccuracy,
		IssueTypeSecurity, IssueTypeCompliance, IssueTypeCost, IssueTypeCompatibility,
	}
	typeStats := make(map[string]int)

	for _, issueType := range issueTypes {
		filters := map[string]interface{}{"issue_type": string(issueType)}
		issues, err := im.db.ListIssues(filters)
		if err != nil {
			return nil, fmt.Errorf("failed to get issues for type %s: %w", issueType, err)
		}
		typeStats[string(issueType)] = len(issues)
	}

	stats["by_type"] = typeStats

	// Open vs resolved issues
	openIssues, err := im.db.ListIssues(map[string]interface{}{"resolved": false})
	if err != nil {
		return nil, fmt.Errorf("failed to get open issues: %w", err)
	}

	resolvedIssues, err := im.db.ListIssues(map[string]interface{}{"resolved": true})
	if err != nil {
		return nil, fmt.Errorf("failed to get resolved issues: %w", err)
	}

	stats["open_issues"] = len(openIssues)
	stats["resolved_issues"] = len(resolvedIssues)
	stats["total_issues"] = len(openIssues) + len(resolvedIssues)

	// Issues by model (top 10)
	modelStats := make(map[string]int)
	allIssues := append(openIssues, resolvedIssues...)

	for _, issue := range allIssues {
		modelKey := fmt.Sprintf("model_%d", issue.ModelID)
		modelStats[modelKey]++
	}

	stats["by_model"] = modelStats

	return stats, nil
}

// GenerateIssueReport generates a comprehensive issue report
func (im *IssueManager) GenerateIssueReport(filters map[string]interface{}) (string, error) {
	issues, err := im.db.ListIssues(filters)
	if err != nil {
		return "", fmt.Errorf("failed to get issues: %w", err)
	}

	report := fmt.Sprintf("# Issue Report\n\n")
	report += fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339))

	if len(filters) > 0 {
		report += "## Filters Applied\n\n"
		for key, value := range filters {
			report += fmt.Sprintf("- %s: %v\n", key, value)
		}
		report += "\n"
	}

	// Summary statistics
	stats, err := im.GetIssueStatistics()
	if err == nil {
		report += "## Summary Statistics\n\n"
		report += fmt.Sprintf("- Total Issues: %d\n", stats["total_issues"])
		report += fmt.Sprintf("- Open Issues: %d\n", stats["open_issues"])
		report += fmt.Sprintf("- Resolved Issues: %d\n", stats["resolved_issues"])
		report += "\n"
	}

	// Issues by severity
	if bySeverity, ok := stats["by_severity"].(map[string]int); ok {
		report += "## Issues by Severity\n\n"
		for severity, count := range bySeverity {
			report += fmt.Sprintf("- %s: %d\n", severity, count)
		}
		report += "\n"
	}

	// Issues by type
	if byType, ok := stats["by_type"].(map[string]int); ok {
		report += "## Issues by Type\n\n"
		for issueType, count := range byType {
			report += fmt.Sprintf("- %s: %d\n", issueType, count)
		}
		report += "\n"
	}

	// Detailed issues
	report += "## Detailed Issues\n\n"

	for _, issue := range issues {
		report += fmt.Sprintf("### %s (ID: %d)\n\n", issue.Title, issue.ID)
		report += fmt.Sprintf("**Model ID:** %d\n\n", issue.ModelID)
		report += fmt.Sprintf("**Type:** %s\n\n", issue.IssueType)
		report += fmt.Sprintf("**Severity:** %s\n\n", issue.Severity)
		report += fmt.Sprintf("**Description:** %s\n\n", issue.Description)

		if issue.Symptoms != nil && *issue.Symptoms != "" {
			report += fmt.Sprintf("**Symptoms:** %s\n\n", *issue.Symptoms)
		}

		if issue.Workarounds != nil && *issue.Workarounds != "" {
			report += fmt.Sprintf("**Workarounds:** %s\n\n", *issue.Workarounds)
		}

		if len(issue.AffectedFeatures) > 0 {
			report += "**Affected Features:**\n"
			for _, feature := range issue.AffectedFeatures {
				report += fmt.Sprintf("- %s\n", feature)
			}
			report += "\n"
		}

		report += fmt.Sprintf("**First Detected:** %s\n", issue.FirstDetected.Format(time.RFC3339))

		if issue.LastOccurred != nil {
			report += fmt.Sprintf("**Last Occurred:** %s\n", issue.LastOccurred.Format(time.RFC3339))
		}

		if issue.ResolvedAt != nil {
			report += fmt.Sprintf("**Resolved:** %s\n", issue.ResolvedAt.Format(time.RFC3339))
			if issue.ResolutionNotes != nil {
				report += fmt.Sprintf("**Resolution Notes:** %s\n", *issue.ResolutionNotes)
			}
		}

		report += "\n---\n\n"
	}

	return report, nil
}

// Helper function to convert bytes to nullable string
func scanNullableStringFromBytes(data []byte) *string {
	if len(data) == 0 {
		return nil
	}
	str := string(data)
	return &str
}

// AutoResolutionChecker checks if issues can be automatically resolved
func (im *IssueManager) AutoResolutionChecker() error {
	// Get all open issues
	openIssues, err := im.db.ListIssues(map[string]interface{}{"resolved": false})
	if err != nil {
		return fmt.Errorf("failed to get open issues: %w", err)
	}

	for _, issue := range openIssues {
		// Check if issue should be auto-resolved based on criteria
		shouldResolve := im.checkAutoResolutionCriteria(issue)

		if shouldResolve {
			resolutionNotes := "Automatically resolved based on system criteria"
			if err := im.UpdateIssueStatus(issue.ID, true, resolutionNotes); err != nil {
				fmt.Printf("Warning: Failed to auto-resolve issue %d: %v\n", issue.ID, err)
			}
		}
	}

	return nil
}

// checkAutoResolutionCriteria determines if an issue should be auto-resolved
func (im *IssueManager) checkAutoResolutionCriteria(issue *database.Issue) bool {
	// Don't auto-resolve critical issues
	if issue.Severity == string(SeverityCritical) {
		return false
	}

	// Auto-resolve if issue hasn't occurred in the last 7 days
	if issue.LastOccurred != nil {
		sevenDaysAgo := time.Now().AddDate(0, 0, -7)
		if issue.LastOccurred.Before(sevenDaysAgo) {
			return true
		}
	}

	// Auto-resolve low severity issues after 30 days
	if issue.Severity == string(SeverityLow) {
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		if issue.FirstDetected.Before(thirtyDaysAgo) {
			return true
		}
	}

	return false
}
