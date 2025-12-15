package enhanced

import (
	"time"

	"llm-verifier/database"
)

// MockDatabase is a mock implementation of the database for testing
type MockDatabase struct {
	*database.Database
	issues []*database.Issue
}

// NewMockDatabase creates a new mock database
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		issues: []*database.Issue{},
	}
}

// CreateIssue mocks the CreateIssue method
func (m *MockDatabase) CreateIssue(issue *database.Issue) error {
	// Simulate database auto-increment
	issue.ID = int64(len(m.issues) + 1)
	issue.CreatedAt = time.Now()
	issue.UpdatedAt = time.Now()

	m.issues = append(m.issues, issue)
	return nil
}

// GetIssue mocks the GetIssue method
func (m *MockDatabase) GetIssue(id int64) (*database.Issue, error) {
	for _, issue := range m.issues {
		if issue.ID == id {
			return issue, nil
		}
	}
	return nil, nil
}

// UpdateIssue mocks the UpdateIssue method
func (m *MockDatabase) UpdateIssue(issue *database.Issue) error {
	for i, existing := range m.issues {
		if existing.ID == issue.ID {
			m.issues[i] = issue
			m.issues[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return nil
}

// ListIssues mocks the ListIssues method
func (m *MockDatabase) ListIssues(filters map[string]interface{}) ([]*database.Issue, error) {
	var filtered []*database.Issue

	for _, issue := range m.issues {
		matches := true

		// Apply filters
		if modelID, ok := filters["model_id"]; ok {
			if issue.ModelID != modelID.(int64) {
				matches = false
			}
		}

		if severity, ok := filters["severity"]; ok {
			if issue.Severity != severity.(string) {
				matches = false
			}
		}

		if issueType, ok := filters["issue_type"]; ok {
			if issue.IssueType != issueType.(string) {
				matches = false
			}
		}

		if resolved, ok := filters["resolved"]; ok {
			isResolved := issue.ResolvedAt != nil
			if isResolved != resolved.(bool) {
				matches = false
			}
		}

		if matches {
			filtered = append(filtered, issue)
		}
	}

	return filtered, nil
}

// GetIssueStatistics mocks the GetIssueStatistics method
func (m *MockDatabase) GetIssueStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Simple statistics
	severityStats := make(map[string]int)
	typeStats := make(map[string]int)
	openCount := 0
	resolvedCount := 0

	for _, issue := range m.issues {
		severityStats[issue.Severity]++
		typeStats[issue.IssueType]++

		if issue.ResolvedAt != nil {
			resolvedCount++
		} else {
			openCount++
		}
	}

	stats["by_severity"] = severityStats
	stats["by_type"] = typeStats
	stats["open_count"] = openCount
	stats["resolved_count"] = resolvedCount
	stats["total_count"] = len(m.issues)

	return stats, nil
}

// Helper function to create a mock verification result
func CreateMockVerificationResult(modelID int64, latencyMs *int, errorMsg *string, overallScore float64) *database.VerificationResult {
	return &database.VerificationResult{
		ID:                     1,
		ModelID:                modelID,
		LatencyMs:              latencyMs,
		ErrorMessage:           errorMsg,
		OverallScore:           overallScore,
		ModelExists:            boolPtr(true),
		Responsive:             boolPtr(true),
		Overloaded:             boolPtr(false),
		SupportsCodeGeneration: true,
		CodeCapabilityScore:    85.0,
	}
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
