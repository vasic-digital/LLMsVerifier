package enhanced

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"digital.vasic.llmsverifier/database"
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

// IssueTemplate represents a template for common issues.
//
// CONST-046: the Name, Title, Description, Symptoms, and Workarounds
// fields hold i18n message IDs, not English literals. They are resolved
// to locale-aware strings by Localized() at consumption time (e.g. inside
// CreateIssueFromTemplate). The non-text fields (ID, IssueType, Severity)
// remain stable identifier tokens.
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

// Localized returns a copy of the template with its Name, Title,
// Description, Symptoms, and Workarounds resolved through the active i18n
// translator. The receiver is unchanged so the package-level
// IssueTemplates registry keeps storing stable message IDs.
func (t IssueTemplate) Localized() IssueTemplate {
	loc := t
	loc.Name = tr(t.Name)
	loc.Title = tr(t.Title)
	loc.Description = tr(t.Description)
	loc.Symptoms = trList(t.Symptoms...)
	loc.Workarounds = trList(t.Workarounds...)
	return loc
}

// Common issue templates. Text fields hold i18n message IDs per CONST-046;
// resolve to display strings via IssueTemplate.Localized().
var IssueTemplates = []IssueTemplate{
	{
		ID:          "high_latency",
		Name:        "issue.high_latency.name",
		IssueType:   IssueTypePerformance,
		Severity:    SeverityMedium,
		Title:       "issue.high_latency.title",
		Description: "issue.high_latency.description",
		Symptoms: []string{
			"issue.high_latency.symptom.response_time",
			"issue.high_latency.symptom.timeouts",
			"issue.high_latency.symptom.degradation",
		},
		Workarounds: []string{
			"issue.high_latency.workaround.retry",
			"issue.high_latency.workaround.context_window",
			"issue.high_latency.workaround.different_model",
		},
	},
	{
		ID:          "rate_limit_exceeded",
		Name:        "issue.rate_limit_exceeded.name",
		IssueType:   IssueTypeAvailability,
		Severity:    SeverityHigh,
		Title:       "issue.rate_limit_exceeded.title",
		Description: "issue.rate_limit_exceeded.description",
		Symptoms: []string{
			"issue.rate_limit_exceeded.symptom.http_429",
			"issue.rate_limit_exceeded.symptom.rejected",
			"issue.rate_limit_exceeded.symptom.quota",
		},
		Workarounds: []string{
			"issue.rate_limit_exceeded.workaround.backoff",
			"issue.rate_limit_exceeded.workaround.frequency",
			"issue.rate_limit_exceeded.workaround.upgrade_tier",
			"issue.rate_limit_exceeded.workaround.queuing",
		},
	},
	{
		ID:          "model_unresponsive",
		Name:        "issue.model_unresponsive.name",
		IssueType:   IssueTypeAvailability,
		Severity:    SeverityCritical,
		Title:       "issue.model_unresponsive.title",
		Description: "issue.model_unresponsive.description",
		Symptoms: []string{
			"issue.model_unresponsive.symptom.timeouts",
			"issue.model_unresponsive.symptom.http_5xx",
			"issue.model_unresponsive.symptom.unavailable",
		},
		Workarounds: []string{
			"issue.model_unresponsive.workaround.status_page",
			"issue.model_unresponsive.workaround.backup_model",
			"issue.model_unresponsive.workaround.support",
			"issue.model_unresponsive.workaround.retry_later",
		},
	},
	{
		ID:          "accuracy_degradation",
		Name:        "issue.accuracy_degradation.name",
		IssueType:   IssueTypeAccuracy,
		Severity:    SeverityMedium,
		Title:       "issue.accuracy_degradation.title",
		Description: "issue.accuracy_degradation.description",
		Symptoms: []string{
			"issue.accuracy_degradation.symptom.low_scores",
			"issue.accuracy_degradation.symptom.incorrect",
			"issue.accuracy_degradation.symptom.complaints",
		},
		Workarounds: []string{
			"issue.accuracy_degradation.workaround.prompt_engineering",
			"issue.accuracy_degradation.workaround.specific_instructions",
			"issue.accuracy_degradation.workaround.fine_tuning",
			"issue.accuracy_degradation.workaround.capable_model",
		},
	},
	{
		ID:          "security_concern",
		Name:        "issue.security_concern.name",
		IssueType:   IssueTypeSecurity,
		Severity:    SeverityHigh,
		Title:       "issue.security_concern.title",
		Description: "issue.security_concern.description",
		Symptoms: []string{
			"issue.security_concern.symptom.sensitive_info",
			"issue.security_concern.symptom.injection",
			"issue.security_concern.symptom.unauthorized",
		},
		Workarounds: []string{
			"issue.security_concern.workaround.input_validation",
			"issue.security_concern.workaround.secure_templates",
			"issue.security_concern.workaround.monitoring",
			"issue.security_concern.workaround.content_filtering",
		},
	},
	{
		ID:          "cost_spike",
		Name:        "issue.cost_spike.name",
		IssueType:   IssueTypeCost,
		Severity:    SeverityMedium,
		Title:       "issue.cost_spike.title",
		Description: "issue.cost_spike.description",
		Symptoms: []string{
			"issue.cost_spike.symptom.billing",
			"issue.cost_spike.symptom.cost_per_request",
			"issue.cost_spike.symptom.budget_alerts",
		},
		Workarounds: []string{
			"issue.cost_spike.workaround.optimize_patterns",
			"issue.cost_spike.workaround.cost_monitoring",
			"issue.cost_spike.workaround.cost_effective_model",
			"issue.cost_spike.workaround.budget_alerts",
		},
	},
}

// CreateIssueFromTemplate creates an issue from a template
func (im *IssueManager) CreateIssueFromTemplate(modelID int64, templateID string, verificationResultID *int64, additionalDetails map[string]interface{}) error {
	// Find the template, then resolve its CONST-046 message IDs to
	// locale-aware display strings before persisting the issue.
	var template *IssueTemplate
	for _, t := range IssueTemplates {
		if t.ID == templateID {
			localized := t.Localized()
			template = &localized
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

	report := fmt.Sprintf("# %s\n\n", tr("issue.report.title"))
	report += fmt.Sprintf("%s: %s\n\n", tr("issue.report.generated"), time.Now().Format(time.RFC3339))

	if len(filters) > 0 {
		report += fmt.Sprintf("## %s\n\n", tr("issue.report.section.filters_applied"))
		for key, value := range filters {
			report += fmt.Sprintf("- %s: %v\n", key, value)
		}
		report += "\n"
	}

	// Summary statistics
	stats, err := im.GetIssueStatistics()
	if err == nil {
		report += fmt.Sprintf("## %s\n\n", tr("issue.report.section.summary_statistics"))
		report += fmt.Sprintf("- %s: %d\n", tr("issue.report.field.total_issues"), stats["total_issues"])
		report += fmt.Sprintf("- %s: %d\n", tr("issue.report.field.open_issues"), stats["open_issues"])
		report += fmt.Sprintf("- %s: %d\n", tr("issue.report.field.resolved_issues"), stats["resolved_issues"])
		report += "\n"
	}

	// Issues by severity
	if bySeverity, ok := stats["by_severity"].(map[string]int); ok {
		report += fmt.Sprintf("## %s\n\n", tr("issue.report.section.by_severity"))
		for severity, count := range bySeverity {
			report += fmt.Sprintf("- %s: %d\n", severity, count)
		}
		report += "\n"
	}

	// Issues by type
	if byType, ok := stats["by_type"].(map[string]int); ok {
		report += fmt.Sprintf("## %s\n\n", tr("issue.report.section.by_type"))
		for issueType, count := range byType {
			report += fmt.Sprintf("- %s: %d\n", issueType, count)
		}
		report += "\n"
	}

	// Detailed issues
	report += fmt.Sprintf("## %s\n\n", tr("issue.report.section.detailed_issues"))

	for _, issue := range issues {
		report += fmt.Sprintf("### %s (ID: %d)\n\n", issue.Title, issue.ID)
		report += fmt.Sprintf("**%s:** %d\n\n", tr("issue.report.field.model_id"), issue.ModelID)
		report += fmt.Sprintf("**%s:** %s\n\n", tr("issue.report.field.type"), issue.IssueType)
		report += fmt.Sprintf("**%s:** %s\n\n", tr("issue.report.field.severity"), issue.Severity)
		report += fmt.Sprintf("**%s:** %s\n\n", tr("issue.report.field.description"), issue.Description)

		if issue.Symptoms != nil && *issue.Symptoms != "" {
			report += fmt.Sprintf("**%s:** %s\n\n", tr("issue.report.field.symptoms"), *issue.Symptoms)
		}

		if issue.Workarounds != nil && *issue.Workarounds != "" {
			report += fmt.Sprintf("**%s:** %s\n\n", tr("issue.report.field.workarounds"), *issue.Workarounds)
		}

		if len(issue.AffectedFeatures) > 0 {
			report += fmt.Sprintf("**%s:**\n", tr("issue.report.field.affected_features"))
			for _, feature := range issue.AffectedFeatures {
				report += fmt.Sprintf("- %s\n", feature)
			}
			report += "\n"
		}

		report += fmt.Sprintf("**%s:** %s\n", tr("issue.report.field.first_detected"), issue.FirstDetected.Format(time.RFC3339))

		if issue.LastOccurred != nil {
			report += fmt.Sprintf("**%s:** %s\n", tr("issue.report.field.last_occurred"), issue.LastOccurred.Format(time.RFC3339))
		}

		if issue.ResolvedAt != nil {
			report += fmt.Sprintf("**%s:** %s\n", tr("issue.report.field.resolved"), issue.ResolvedAt.Format(time.RFC3339))
			if issue.ResolutionNotes != nil {
				report += fmt.Sprintf("**%s:** %s\n", tr("issue.report.field.resolution_notes"), *issue.ResolutionNotes)
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
			resolutionNotes := tr("issue.auto_resolution.note")
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

// GitHubIssueReporter handles automatic issue reporting to GitHub
type GitHubIssueReporter struct {
	token      string
	repository string
	baseURL    string
	httpClient *http.Client
}

// NewGitHubIssueReporter creates a new GitHub issue reporter
func NewGitHubIssueReporter(token, repository string) *GitHubIssueReporter {
	return &GitHubIssueReporter{
		token:      token,
		repository: repository,
		baseURL:    "https://api.github.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ReportIssue creates a GitHub issue for a detected problem
func (gir *GitHubIssueReporter) ReportIssue(issue *database.Issue, modelInfo map[string]interface{}) error {
	if gir.token == "" {
		return fmt.Errorf("GitHub token not configured")
	}

	severity := IssueSeverity(issue.Severity)
	issueType := IssueType(issue.IssueType)

	issueData := map[string]interface{}{
		"title": gir.generateIssueTitle(issue),
		"body":  gir.generateIssueBody(issue, modelInfo),
		"labels": []string{
			gir.getSeverityLabel(severity),
			gir.getTypeLabel(issueType),
			"automated",
			"llm-verifier",
		},
	}

	jsonData, err := json.Marshal(issueData)
	if err != nil {
		return fmt.Errorf("failed to marshal issue data: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/issues", gir.baseURL, gir.repository)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gir.token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := gir.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	return nil
}

// generateIssueTitle creates a descriptive title for the GitHub issue
func (gir *GitHubIssueReporter) generateIssueTitle(issue *database.Issue) string {
	severity := IssueSeverity(issue.Severity)
	return fmt.Sprintf("[%s] %s: %s", gir.getSeverityEmoji(severity), issue.IssueType, issue.Title)
}

// generateIssueBody creates detailed issue body
func (gir *GitHubIssueReporter) generateIssueBody(issue *database.Issue, modelInfo map[string]interface{}) string {
	lastOccurred := "N/A"
	if issue.LastOccurred != nil {
		lastOccurred = issue.LastOccurred.Format(time.RFC3339)
	}

	symptoms := "N/A"
	if issue.Symptoms != nil {
		symptoms = *issue.Symptoms
	}

	workarounds := "N/A"
	if issue.Workarounds != nil {
		workarounds = *issue.Workarounds
	}

	body := fmt.Sprintf(`## Issue Details

**Type:** %s
**Severity:** %s
**First Detected:** %s
**Last Occurred:** %s

## Description

%s

## Symptoms

%s

## Workarounds

%s

## Affected Features

%s

## Model Information

`+"```json"+`
%s
`+"```"+`

---

*This issue was automatically reported by LLM Verifier*
*Detection Time: %s*
`,
		issue.IssueType,
		issue.Severity,
		issue.FirstDetected.Format(time.RFC3339),
		lastOccurred,
		issue.Description,
		symptoms,
		workarounds,
		strings.Join(issue.AffectedFeatures, ", "),
		gir.formatModelInfo(modelInfo),
		time.Now().Format(time.RFC3339))

	return body
}

// Helper methods for GitHub integration
func (gir *GitHubIssueReporter) getSeverityLabel(severity IssueSeverity) string {
	switch severity {
	case SeverityCritical:
		return "severity-critical"
	case SeverityHigh:
		return "severity-high"
	case SeverityMedium:
		return "severity-medium"
	case SeverityLow:
		return "severity-low"
	default:
		return "severity-unknown"
	}
}

func (gir *GitHubIssueReporter) getSeverityEmoji(severity IssueSeverity) string {
	switch severity {
	case SeverityCritical:
		return "🚨"
	case SeverityHigh:
		return "⚠️"
	case SeverityMedium:
		return "⚡"
	case SeverityLow:
		return "ℹ️"
	default:
		return "❓"
	}
}

func (gir *GitHubIssueReporter) getTypeLabel(issueType IssueType) string {
	return fmt.Sprintf("type-%s", issueType)
}

func (gir *GitHubIssueReporter) formatIssueDetails(details map[string]interface{}) string {
	if details == nil {
		return "{}"
	}
	jsonBytes, _ := json.MarshalIndent(details, "", "  ")
	return string(jsonBytes)
}

func (gir *GitHubIssueReporter) formatModelInfo(modelInfo map[string]interface{}) string {
	if modelInfo == nil {
		return "{}"
	}
	jsonBytes, _ := json.MarshalIndent(modelInfo, "", "  ")
	return string(jsonBytes)
}

func (gir *GitHubIssueReporter) generateRecommendations(issue *database.Issue) string {
	var recommendations []string

	switch IssueType(issue.IssueType) {
	case IssueTypeAvailability:
		recommendations = trList(
			"issue.recommendation.availability.api_status",
			"issue.recommendation.availability.credentials",
			"issue.recommendation.availability.fallback",
			"issue.recommendation.availability.status_pages",
		)
	case IssueTypePerformance:
		recommendations = trList(
			"issue.recommendation.performance.rate_limiting",
			"issue.recommendation.performance.upgrade_tier",
			"issue.recommendation.performance.batching",
			"issue.recommendation.performance.timeouts",
		)
	case IssueTypeAccuracy:
		recommendations = trList(
			"issue.recommendation.accuracy.prompt_engineering",
			"issue.recommendation.accuracy.model_versions",
			"issue.recommendation.accuracy.validation",
			"issue.recommendation.accuracy.fine_tuning",
		)
	case IssueTypeSecurity:
		recommendations = trList(
			"issue.recommendation.security.api_key_audit",
			"issue.recommendation.security.content_filtering",
			"issue.recommendation.security.privacy_compliance",
			"issue.recommendation.security.policies",
		)
	default:
		recommendations = trList(
			"issue.recommendation.default.monitor_trends",
			"issue.recommendation.default.review_config",
			"issue.recommendation.default.error_handling",
			"issue.recommendation.default.provider_support",
		)
	}

	return strings.Join(recommendations, "\n")
}

// SlackIssueReporter handles Slack notifications for issues
type SlackIssueReporter struct {
	webhookURL string
	channel    string
	httpClient *http.Client
}

// NewSlackIssueReporter creates a new Slack issue reporter
func NewSlackIssueReporter(webhookURL, channel string) *SlackIssueReporter {
	return &SlackIssueReporter{
		webhookURL: webhookURL,
		channel:    channel,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ReportIssue sends issue notification to Slack
func (sir *SlackIssueReporter) ReportIssue(issue *database.Issue, modelInfo map[string]interface{}) error {
	if sir.webhookURL == "" {
		return fmt.Errorf("Slack webhook URL not configured")
	}

	color := sir.getSlackColor(IssueSeverity(issue.Severity))

	payload := map[string]interface{}{
		"channel": sir.channel,
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": fmt.Sprintf("%s: %s", issue.IssueType, issue.Title),
				"text":  issue.Description,
				"fields": []map[string]interface{}{
					{
						"title": tr("issue.report.field.severity"),
						"value": issue.Severity,
						"short": true,
					},
					{
						"title": tr("issue.report.field.type"),
						"value": issue.IssueType,
						"short": true,
					},
					{
						"title": tr("issue.report.field.affected_features"),
						"value": strings.Join(issue.AffectedFeatures, ", "),
						"short": false,
					},
					{
						"title": tr("issue.report.field.first_detected"),
						"value": issue.FirstDetected.Format("2006-01-02 15:04:05"),
						"short": true,
					},
				},
				"footer": tr("issue.report.footer.llm_verifier"),
				"ts":     time.Now().Unix(),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	resp, err := http.Post(sir.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned status: %d", resp.StatusCode)
	}

	return nil
}

func (sir *SlackIssueReporter) getSlackColor(severity IssueSeverity) string {
	switch severity {
	case SeverityCritical:
		return "danger"
	case SeverityHigh:
		return "warning"
	case SeverityMedium:
		return "#ff9900"
	case SeverityLow:
		return "good"
	default:
		return "#808080"
	}
}
