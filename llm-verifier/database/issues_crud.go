package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ==================== Issue CRUD Operations ====================

// CreateIssue creates a new issue
func (d *Database) CreateIssue(issue *Issue) error {
	query := `
		INSERT INTO issues (
			model_id, issue_type, severity, title, description, symptoms,
			workarounds, affected_features, first_detected, last_occurred,
			resolved_at, resolution_notes, verification_result_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	var lastOccurred, resolvedAt sql.NullTime
	if issue.LastOccurred != nil {
		lastOccurred.Valid = true
		lastOccurred.Time = *issue.LastOccurred
	}
	
	if issue.ResolvedAt != nil {
		resolvedAt.Valid = true
		resolvedAt.Time = *issue.ResolvedAt
	}
	
	affectedFeaturesJSON, err := json.Marshal(issue.AffectedFeatures)
	if err != nil {
		return fmt.Errorf("failed to marshal affected features: %w", err)
	}
	
	result, err := d.conn.Exec(query,
		issue.ModelID,
		issue.IssueType,
		issue.Severity,
		issue.Title,
		issue.Description,
		issue.Symptoms,
		issue.Workarounds,
		string(affectedFeaturesJSON),
		issue.FirstDetected,
		lastOccurred,
		resolvedAt,
		issue.ResolutionNotes,
		issue.VerificationResultID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	
	issue.ID = id
	return nil
}

// GetIssue retrieves an issue by ID
func (d *Database) GetIssue(id int64) (*Issue, error) {
	query := `
		SELECT id, model_id, issue_type, severity, title, description, symptoms,
			workarounds, affected_features, first_detected, last_occurred,
			resolved_at, resolution_notes, verification_result_id, created_at, updated_at
		FROM issues WHERE id = ?
	`
	
	var issue Issue
	var lastOccurred, resolvedAt sql.NullTime
	var affectedFeaturesJSON sql.NullString
	
	err := d.conn.QueryRow(query, id).Scan(
		&issue.ID,
		&issue.ModelID,
		&issue.IssueType,
		&issue.Severity,
		&issue.Title,
		&issue.Description,
		&issue.Symptoms,
		&issue.Workarounds,
		&affectedFeaturesJSON,
		&issue.FirstDetected,
		&lastOccurred,
		&resolvedAt,
		&issue.ResolutionNotes,
		&issue.VerificationResultID,
		&issue.CreatedAt,
		&issue.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("issue not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}
	
	issue.LastOccurred = scanNullableTime(lastOccurred)
	issue.ResolvedAt = scanNullableTime(resolvedAt)
	
	// Parse affected features
	if affectedFeaturesJSON.Valid && affectedFeaturesJSON.String != "" {
		var affectedFeatures []string
		if err := json.Unmarshal([]byte(affectedFeaturesJSON.String), &affectedFeatures); err == nil {
			issue.AffectedFeatures = affectedFeatures
		}
	}
	
	return &issue, nil
}

// UpdateIssue updates an existing issue
func (d *Database) UpdateIssue(issue *Issue) error {
	query := `
		UPDATE issues SET
			model_id = ?, issue_type = ?, severity = ?, title = ?, description = ?,
			symptoms = ?, workarounds = ?, affected_features = ?, last_occurred = ?,
			resolved_at = ?, resolution_notes = ?, verification_result_id = ?
		WHERE id = ?
	`
	
	var lastOccurred, resolvedAt sql.NullTime
	if issue.LastOccurred != nil {
		lastOccurred.Valid = true
		lastOccurred.Time = *issue.LastOccurred
	}
	
	if issue.ResolvedAt != nil {
		resolvedAt.Valid = true
		resolvedAt.Time = *issue.ResolvedAt
	}
	
	affectedFeaturesJSON, err := json.Marshal(issue.AffectedFeatures)
	if err != nil {
		return fmt.Errorf("failed to marshal affected features: %w", err)
	}
	
	_, err = d.conn.Exec(query,
		issue.ModelID,
		issue.IssueType,
		issue.Severity,
		issue.Title,
		issue.Description,
		issue.Symptoms,
		issue.Workarounds,
		string(affectedFeaturesJSON),
		lastOccurred,
		resolvedAt,
		issue.ResolutionNotes,
		issue.VerificationResultID,
		issue.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}
	
	return nil
}

// ListIssues retrieves issues with optional filtering
func (d *Database) ListIssues(filters map[string]interface{}) ([]*Issue, error) {
	query := `
		SELECT id, model_id, issue_type, severity, title, description, symptoms,
			workarounds, affected_features, first_detected, last_occurred,
			resolved_at, resolution_notes, verification_result_id, created_at, updated_at
		FROM issues
	`
	
	var conditions []string
	var args []interface{}
	
	// Add conditions based on filters
	if modelID, ok := filters["model_id"]; ok {
		conditions = append(conditions, "model_id = ?")
		args = append(args, modelID)
	}
	
	if severity, ok := filters["severity"]; ok {
		conditions = append(conditions, "severity = ?")
		args = append(args, severity)
	}
	
	if issueType, ok := filters["issue_type"]; ok {
		conditions = append(conditions, "issue_type = ?")
		args = append(args, issueType)
	}
	
	if resolved, ok := filters["resolved"]; ok {
		if resolved.(bool) {
			conditions = append(conditions, "resolved_at IS NOT NULL")
		} else {
			conditions = append(conditions, "resolved_at IS NULL")
		}
	}
	
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	query += " ORDER BY first_detected DESC"
	
	if limit, ok := filters["limit"]; ok {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	
	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}
	defer rows.Close()
	
	var issues []*Issue
	for rows.Next() {
		var issue Issue
		var lastOccurred, resolvedAt sql.NullTime
		var affectedFeaturesJSON sql.NullString
		
		err := rows.Scan(
			&issue.ID,
			&issue.ModelID,
			&issue.IssueType,
			&issue.Severity,
			&issue.Title,
			&issue.Description,
			&issue.Symptoms,
			&issue.Workarounds,
			&affectedFeaturesJSON,
			&issue.FirstDetected,
			&lastOccurred,
			&resolvedAt,
			&issue.ResolutionNotes,
			&issue.VerificationResultID,
			&issue.CreatedAt,
			&issue.UpdatedAt,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}
		
		issue.LastOccurred = scanNullableTime(lastOccurred)
		issue.ResolvedAt = scanNullableTime(resolvedAt)
		
		// Parse affected features
		if affectedFeaturesJSON.Valid && affectedFeaturesJSON.String != "" {
			var affectedFeatures []string
			if err := json.Unmarshal([]byte(affectedFeaturesJSON.String), &affectedFeatures); err == nil {
				issue.AffectedFeatures = affectedFeatures
			}
		}
		
		issues = append(issues, &issue)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating issues: %w", err)
	}
	
	return issues, nil
}

// GetIssuesBySeverity gets issues filtered by severity
func (d *Database) GetIssuesBySeverity(severity string, includeResolved bool) ([]*Issue, error) {
	filters := map[string]interface{}{
		"severity": severity,
	}
	
	if !includeResolved {
		filters["resolved"] = false
	}
	
	return d.ListIssues(filters)
}

// GetIssuesByType gets issues filtered by type
func (d *Database) GetIssuesByType(issueType string, includeResolved bool) ([]*Issue, error) {
	filters := map[string]interface{}{
		"issue_type": issueType,
	}
	
	if !includeResolved {
		filters["resolved"] = false
	}
	
	return d.ListIssues(filters)
}

// GetIssuesForModel gets all issues for a specific model
func (d *Database) GetIssuesForModel(modelID int64, includeResolved bool) ([]*Issue, error) {
	filters := map[string]interface{}{
		"model_id": modelID,
	}
	
	if !includeResolved {
		filters["resolved"] = false
	}
	
	return d.ListIssues(filters)
}

// GetUnresolvedIssues gets all unresolved issues
func (d *Database) GetUnresolvedIssues() ([]*Issue, error) {
	return d.ListIssues(map[string]interface{}{"resolved": false})
}

// UpdateIssueLastOccurred updates the last_occurred timestamp for an issue
func (d *Database) UpdateIssueLastOccurred(issueID int64) error {
	query := `UPDATE issues SET last_occurred = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	
	_, err := d.conn.Exec(query, issueID)
	if err != nil {
		return fmt.Errorf("failed to update issue last occurred: %w", err)
	}
	
	return nil
}

// GetIssueStatistics gets statistics about issues
func (d *Database) GetIssueStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Total count
	var totalCount int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM issues").Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total_count"] = totalCount
	
	// By severity
	severityQuery := `
		SELECT severity, COUNT(*) as count 
		FROM issues 
		GROUP BY severity
		ORDER BY count DESC
	`
	
	severityRows, err := d.conn.Query(severityQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get severity statistics: %w", err)
	}
	defer severityRows.Close()
	
	severityStats := make(map[string]int)
	for severityRows.Next() {
		var severity string
		var count int
		if err := severityRows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity row: %w", err)
		}
		severityStats[severity] = count
	}
	stats["by_severity"] = severityStats
	
	// By type
	typeQuery := `
		SELECT issue_type, COUNT(*) as count 
		FROM issues 
		GROUP BY issue_type
		ORDER BY count DESC
	`
	
	typeRows, err := d.conn.Query(typeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get type statistics: %w", err)
	}
	defer typeRows.Close()
	
	typeStats := make(map[string]int)
	for typeRows.Next() {
		var issueType string
		var count int
		if err := typeRows.Scan(&issueType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan type row: %w", err)
		}
		typeStats[issueType] = count
	}
	stats["by_type"] = typeStats
	
	// Open vs resolved
	var openCount int
	err = d.conn.QueryRow("SELECT COUNT(*) FROM issues WHERE resolved_at IS NULL").Scan(&openCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get open count: %w", err)
	}
	
	var resolvedCount int
	err = d.conn.QueryRow("SELECT COUNT(*) FROM issues WHERE resolved_at IS NOT NULL").Scan(&resolvedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get resolved count: %w", err)
	}
	
	stats["open_count"] = openCount
	stats["resolved_count"] = resolvedCount
	
	// By model (top 10)
	modelQuery := `
		SELECT model_id, COUNT(*) as count 
		FROM issues 
		GROUP BY model_id
		ORDER BY count DESC
		LIMIT 10
	`
	
	modelRows, err := d.conn.Query(modelQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get model statistics: %w", err)
	}
	defer modelRows.Close()
	
	modelStats := make(map[int64]int)
	for modelRows.Next() {
		var modelID int64
		var count int
		if err := modelRows.Scan(&modelID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan model row: %w", err)
		}
		modelStats[modelID] = count
	}
	stats["by_model"] = modelStats
	
	// Average resolution time
	var avgResolutionTime sql.NullFloat64
	resolutionQuery := `
		SELECT AVG(JULIANDAY(resolved_at) - JULIANDAY(first_detected)) * 24 * 60 as avg_minutes
		FROM issues 
		WHERE resolved_at IS NOT NULL
	`
	err = d.conn.QueryRow(resolutionQuery).Scan(&avgResolutionTime)
	if err == nil && avgResolutionTime.Valid {
		stats["avg_resolution_hours"] = avgResolutionTime.Float64 / 60.0 // Convert minutes to hours
	}
	
	return stats, nil
}

// DeleteIssue deletes an issue by ID
func (d *Database) DeleteIssue(id int64) error {
	query := `DELETE FROM issues WHERE id = ?`
	
	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}
	
	return nil
}