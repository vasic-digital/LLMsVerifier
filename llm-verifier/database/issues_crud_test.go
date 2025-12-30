package database

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupIssueTestDB(t *testing.T) *Database {
	dbFile := "/tmp/test_issues_" + time.Now().Format("20060102150405") + ".db"
	db, err := New(dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbFile)
	})
	return db
}

func createTestIssue(modelID int64) *Issue {
	symptoms := "Test symptoms"
	workarounds := "Test workarounds"
	return &Issue{
		ModelID:          modelID,
		IssueType:        "performance",
		Severity:         "medium",
		Title:            "Test Issue",
		Description:      "Test description",
		Symptoms:         &symptoms,
		Workarounds:      &workarounds,
		AffectedFeatures: []string{"feature1", "feature2"},
		FirstDetected:    time.Now(),
	}
}

func TestCreateIssue(t *testing.T) {
	db := setupIssueTestDB(t)

	// Create a model first
	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create issue
	issue := createTestIssue(model.ID)
	err = db.CreateIssue(issue)
	require.NoError(t, err)
	assert.NotZero(t, issue.ID)
}

func TestCreateIssue_WithOptionalFields(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create issue with resolved_at and last_occurred
	now := time.Now()
	resolutionNotes := "Fixed by update"
	issue := &Issue{
		ModelID:         model.ID,
		IssueType:       "bug",
		Severity:        "critical",
		Title:           "Critical Bug",
		Description:     "A critical bug",
		FirstDetected:   now.Add(-24 * time.Hour),
		LastOccurred:    &now,
		ResolvedAt:      &now,
		ResolutionNotes: &resolutionNotes,
	}

	err = db.CreateIssue(issue)
	require.NoError(t, err)
	assert.NotZero(t, issue.ID)
}

func TestGetIssue(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	issue := createTestIssue(model.ID)
	err = db.CreateIssue(issue)
	require.NoError(t, err)

	// Retrieve issue
	retrieved, err := db.GetIssue(issue.ID)
	require.NoError(t, err)
	assert.Equal(t, issue.ID, retrieved.ID)
	assert.Equal(t, issue.Title, retrieved.Title)
	assert.Equal(t, issue.Severity, retrieved.Severity)
	assert.Equal(t, issue.IssueType, retrieved.IssueType)
	assert.Equal(t, issue.AffectedFeatures, retrieved.AffectedFeatures)
}

func TestGetIssue_NotFound(t *testing.T) {
	db := setupIssueTestDB(t)

	_, err := db.GetIssue(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateIssue(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	issue := createTestIssue(model.ID)
	err = db.CreateIssue(issue)
	require.NoError(t, err)

	// Update issue
	issue.Title = "Updated Title"
	issue.Severity = "critical"
	now := time.Now()
	issue.ResolvedAt = &now
	resolutionNotes := "Fixed in v2.0"
	issue.ResolutionNotes = &resolutionNotes

	err = db.UpdateIssue(issue)
	require.NoError(t, err)

	// Verify update
	retrieved, err := db.GetIssue(issue.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrieved.Title)
	assert.Equal(t, "critical", retrieved.Severity)
	assert.NotNil(t, retrieved.ResolvedAt)
	assert.NotNil(t, retrieved.ResolutionNotes)
}

func TestListIssues_NoFilters(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create multiple issues
	for i := 0; i < 3; i++ {
		issue := createTestIssue(model.ID)
		issue.Title = "Issue " + string(rune('A'+i))
		err = db.CreateIssue(issue)
		require.NoError(t, err)
	}

	// List all
	issues, err := db.ListIssues(nil)
	require.NoError(t, err)
	assert.Len(t, issues, 3)
}

func TestListIssues_WithFilters(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create issues with different severities
	severities := []string{"low", "medium", "high", "critical"}
	for _, sev := range severities {
		issue := createTestIssue(model.ID)
		issue.Severity = sev
		err = db.CreateIssue(issue)
		require.NoError(t, err)
	}

	// Filter by severity
	issues, err := db.ListIssues(map[string]interface{}{"severity": "critical"})
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.Equal(t, "critical", issues[0].Severity)

	// Filter by model_id
	issues, err = db.ListIssues(map[string]interface{}{"model_id": model.ID})
	require.NoError(t, err)
	assert.Len(t, issues, 4)

	// Filter with limit
	issues, err = db.ListIssues(map[string]interface{}{"limit": 2})
	require.NoError(t, err)
	assert.Len(t, issues, 2)
}

func TestListIssues_FilterByResolved(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create unresolved issue
	issue1 := createTestIssue(model.ID)
	issue1.Title = "Unresolved"
	err = db.CreateIssue(issue1)
	require.NoError(t, err)

	// Create resolved issue
	issue2 := createTestIssue(model.ID)
	issue2.Title = "Resolved"
	now := time.Now()
	issue2.ResolvedAt = &now
	err = db.CreateIssue(issue2)
	require.NoError(t, err)

	// Filter unresolved
	unresolvedIssues, err := db.ListIssues(map[string]interface{}{"resolved": false})
	require.NoError(t, err)
	assert.Len(t, unresolvedIssues, 1)
	assert.Equal(t, "Unresolved", unresolvedIssues[0].Title)

	// Filter resolved
	resolvedIssues, err := db.ListIssues(map[string]interface{}{"resolved": true})
	require.NoError(t, err)
	assert.Len(t, resolvedIssues, 1)
	assert.Equal(t, "Resolved", resolvedIssues[0].Title)
}

func TestGetIssuesBySeverity(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create issues with different severities
	severities := []string{"low", "medium", "high", "high"}
	for _, sev := range severities {
		issue := createTestIssue(model.ID)
		issue.Severity = sev
		err = db.CreateIssue(issue)
		require.NoError(t, err)
	}

	issues, err := db.GetIssuesBySeverity("high", true)
	require.NoError(t, err)
	assert.Len(t, issues, 2)
}

func TestGetIssuesByType(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create issues with different types
	types := []string{"bug", "bug", "performance", "compatibility"}
	for _, issueType := range types {
		issue := createTestIssue(model.ID)
		issue.IssueType = issueType
		err = db.CreateIssue(issue)
		require.NoError(t, err)
	}

	issues, err := db.GetIssuesByType("bug", true)
	require.NoError(t, err)
	assert.Len(t, issues, 2)
}

func TestGetIssuesForModel(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model1 := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model-1",
		Name:       "Test Model 1",
	}
	err = db.CreateModel(model1)
	require.NoError(t, err)

	model2 := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model-2",
		Name:       "Test Model 2",
	}
	err = db.CreateModel(model2)
	require.NoError(t, err)

	// Create issues for model1
	for i := 0; i < 3; i++ {
		issue := createTestIssue(model1.ID)
		err = db.CreateIssue(issue)
		require.NoError(t, err)
	}

	// Create issue for model2
	issue := createTestIssue(model2.ID)
	err = db.CreateIssue(issue)
	require.NoError(t, err)

	issues, err := db.GetIssuesForModel(model1.ID, true)
	require.NoError(t, err)
	assert.Len(t, issues, 3)
}

func TestGetUnresolvedIssues(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create 2 unresolved issues
	for i := 0; i < 2; i++ {
		issue := createTestIssue(model.ID)
		err = db.CreateIssue(issue)
		require.NoError(t, err)
	}

	// Create 1 resolved issue
	issue := createTestIssue(model.ID)
	now := time.Now()
	issue.ResolvedAt = &now
	err = db.CreateIssue(issue)
	require.NoError(t, err)

	issues, err := db.GetUnresolvedIssues()
	require.NoError(t, err)
	assert.Len(t, issues, 2)
}

func TestUpdateIssueLastOccurred(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	issue := createTestIssue(model.ID)
	err = db.CreateIssue(issue)
	require.NoError(t, err)

	// Update last occurred
	err = db.UpdateIssueLastOccurred(issue.ID)
	require.NoError(t, err)

	// Verify
	retrieved, err := db.GetIssue(issue.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.LastOccurred)
}

func TestGetIssueStatistics(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create issues with varying properties
	severities := []string{"low", "medium", "high", "critical"}
	types := []string{"bug", "performance", "bug", "compatibility"}
	for i := 0; i < 4; i++ {
		issue := createTestIssue(model.ID)
		issue.Severity = severities[i]
		issue.IssueType = types[i]
		if i == 0 { // Resolve first one
			now := time.Now()
			issue.ResolvedAt = &now
		}
		err = db.CreateIssue(issue)
		require.NoError(t, err)
	}

	stats, err := db.GetIssueStatistics()
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 4, stats["total_count"])
	assert.Equal(t, 3, stats["open_count"])
	assert.Equal(t, 1, stats["resolved_count"])
	assert.NotNil(t, stats["by_severity"])
	assert.NotNil(t, stats["by_type"])
	assert.NotNil(t, stats["by_model"])
}

func TestDeleteIssue(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	issue := createTestIssue(model.ID)
	err = db.CreateIssue(issue)
	require.NoError(t, err)

	// Delete issue
	err = db.DeleteIssue(issue.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = db.GetIssue(issue.ID)
	assert.Error(t, err)
}

func TestListIssues_FilterByIssueType(t *testing.T) {
	db := setupIssueTestDB(t)

	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create issues with different types
	types := []string{"bug", "performance", "compatibility"}
	for _, issueType := range types {
		issue := createTestIssue(model.ID)
		issue.IssueType = issueType
		err = db.CreateIssue(issue)
		require.NoError(t, err)
	}

	// Filter by issue_type
	issues, err := db.ListIssues(map[string]interface{}{"issue_type": "bug"})
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.Equal(t, "bug", issues[0].IssueType)
}
