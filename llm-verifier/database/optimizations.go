package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ==================== Query Optimizations ====================

// QueryOptimizer provides query optimization and performance monitoring
type QueryOptimizer struct {
	db *Database
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *Database) *QueryOptimizer {
	return &QueryOptimizer{db: db}
}

// AnalyzeQueryPerformance analyzes the performance of a query
func (qo *QueryOptimizer) AnalyzeQueryPerformance(ctx context.Context, query string, args ...interface{}) (time.Duration, error) {
	start := time.Now()

	// Execute query with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := qo.db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	// Consume all rows to complete the query
	for rows.Next() {
		// Just iterate, don't process data for analysis
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	duration := time.Since(start)
	return duration, nil
}

// OptimizeModelQueries provides optimized queries for model operations
func (qo *QueryOptimizer) OptimizeModelQueries() {
	// Analyze slow queries and suggest optimizations
	log.Println("Analyzing model query performance...")

	// Example: Check if composite indexes are being used effectively
	ctx := context.Background()

	// Test common model queries
	queries := []struct {
		name  string
		query string
		args  []interface{}
	}{
		{"ListModels", "SELECT COUNT(*) FROM models WHERE verification_status = ?", []interface{}{"verified"}},
		{"GetModelByProvider", "SELECT COUNT(*) FROM models WHERE provider_id = ?", []interface{}{1}},
		{"SearchModels", "SELECT COUNT(*) FROM models WHERE name LIKE ?", []interface{}{"%gpt%"}},
	}

	for _, q := range queries {
		duration, err := qo.AnalyzeQueryPerformance(ctx, q.query, q.args...)
		if err != nil {
			log.Printf("Error analyzing query %s: %v", q.name, err)
			continue
		}

		if duration > 100*time.Millisecond {
			log.Printf("Slow query detected: %s took %v", q.name, duration)
		} else {
			log.Printf("Query %s performed well: %v", q.name, duration)
		}
	}
}

// BatchInsert performs optimized batch inserts
func (d *Database) BatchInsertModels(models []*Model) error {
	return d.WithTransaction(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(`
			INSERT INTO models (
				provider_id, model_id, name, description, version, architecture,
				parameter_count, context_window_tokens, max_output_tokens,
				training_data_cutoff, release_date, is_multimodal, supports_vision,
				supports_audio, supports_video, supports_reasoning, open_source,
				deprecated, tags, language_support, use_case, verification_status,
				overall_score, code_capability_score, responsiveness_score,
				reliability_score, feature_richness_score, value_proposition_score
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, model := range models {
			tagsJSON, _ := json.Marshal(model.Tags)
			langSupportJSON, _ := json.Marshal(model.LanguageSupport)

			_, err = stmt.Exec(
				model.ProviderID,
				model.ModelID,
				model.Name,
				model.Description,
				model.Version,
				model.Architecture,
				model.ParameterCount,
				model.ContextWindowTokens,
				model.MaxOutputTokens,
				model.TrainingDataCutoff,
				model.ReleaseDate,
				model.IsMultimodal,
				model.SupportsVision,
				model.SupportsAudio,
				model.SupportsVideo,
				model.SupportsReasoning,
				model.OpenSource,
				model.Deprecated,
				string(tagsJSON),
				string(langSupportJSON),
				model.UseCase,
				model.VerificationStatus,
				model.OverallScore,
				model.CodeCapabilityScore,
				model.ResponsivenessScore,
				model.ReliabilityScore,
				model.FeatureRichnessScore,
				model.ValuePropositionScore,
			)
			if err != nil {
				return fmt.Errorf("failed to insert model %s: %w", model.Name, err)
			}
		}

		return nil
	})
}

// GetModelsWithStats retrieves models with aggregated statistics (optimized)
func (d *Database) GetModelsWithStats(limit int) ([]*ModelWithStats, error) {
	query := `
		SELECT
			m.id, m.provider_id, m.model_id, m.name, m.description, m.version,
			m.architecture, m.parameter_count, m.context_window_tokens, m.max_output_tokens,
			m.training_data_cutoff, m.release_date, m.is_multimodal, m.supports_vision,
			m.supports_audio, m.supports_video, m.supports_reasoning, m.open_source,
			m.deprecated, m.tags, m.language_support, m.use_case, m.created_at,
			m.updated_at, m.last_verified, m.verification_status, m.overall_score,
			m.code_capability_score, m.responsiveness_score, m.reliability_score,
			m.feature_richness_score, m.value_proposition_score,
			p.name as provider_name,
			COUNT(vr.id) as verification_count,
			MAX(vr.overall_score) as latest_score,
			AVG(vr.avg_latency_ms) as avg_latency,
			COUNT(CASE WHEN vr.status = 'completed' THEN 1 END) as completed_verifications,
			COUNT(i.id) as open_issues
		FROM models m
		JOIN providers p ON m.provider_id = p.id
		LEFT JOIN verification_results vr ON m.id = vr.model_id
		LEFT JOIN issues i ON m.id = i.model_id AND i.resolved_at IS NULL
		GROUP BY m.id, p.name
		ORDER BY m.overall_score DESC
		LIMIT ?
	`

	rows, err := d.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get models with stats: %w", err)
	}
	defer rows.Close()

	var models []*ModelWithStats
	for rows.Next() {
		var model ModelWithStats
		var tagsJSON, langSupportJSON sql.NullString
		var trainingDataCutoff, releaseDate, lastVerified sql.NullTime
		var parameterCount, contextWindowTokens, maxOutputTokens sql.NullInt64
		var latestScore, avgLatency sql.NullFloat64

		err := rows.Scan(
			&model.ID,
			&model.ProviderID,
			&model.ModelID,
			&model.Name,
			&model.Description,
			&model.Version,
			&model.Architecture,
			&parameterCount,
			&contextWindowTokens,
			&maxOutputTokens,
			&trainingDataCutoff,
			&releaseDate,
			&model.IsMultimodal,
			&model.SupportsVision,
			&model.SupportsAudio,
			&model.SupportsVideo,
			&model.SupportsReasoning,
			&model.OpenSource,
			&model.Deprecated,
			&tagsJSON,
			&langSupportJSON,
			&model.UseCase,
			&model.CreatedAt,
			&model.UpdatedAt,
			&lastVerified,
			&model.VerificationStatus,
			&model.OverallScore,
			&model.CodeCapabilityScore,
			&model.ResponsivenessScore,
			&model.ReliabilityScore,
			&model.FeatureRichnessScore,
			&model.ValuePropositionScore,
			&model.ProviderName,
			&model.VerificationCount,
			&latestScore,
			&avgLatency,
			&model.CompletedVerifications,
			&model.OpenIssues,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan model with stats: %w", err)
		}

		model.ParameterCount = scanNullableInt64(parameterCount)
		model.ContextWindowTokens = scanNullableIntFromInt64(contextWindowTokens)
		model.MaxOutputTokens = scanNullableIntFromInt64(maxOutputTokens)
		model.TrainingDataCutoff = scanNullableTime(trainingDataCutoff)
		model.ReleaseDate = scanNullableTime(releaseDate)
		model.Tags = scanJSONString(tagsJSON)
		model.LanguageSupport = scanJSONString(langSupportJSON)
		model.LastVerified = scanNullableTime(lastVerified)
		model.LatestScore = latestScore
		model.AvgLatency = avgLatency

		models = append(models, &model)
	}

	return models, nil
}

// ModelWithStats represents a model with aggregated statistics
type ModelWithStats struct {
	Model
	ProviderName           string          `json:"provider_name"`
	VerificationCount      int             `json:"verification_count"`
	LatestScore            sql.NullFloat64 `json:"latest_score"`
	AvgLatency             sql.NullFloat64 `json:"avg_latency"`
	CompletedVerifications int             `json:"completed_verifications"`
	OpenIssues             int             `json:"open_issues"`
}

// VacuumDatabase performs database maintenance and optimization
func (d *Database) VacuumDatabase() error {
	log.Println("Starting database vacuum and optimization...")

	// Run VACUUM to reclaim space and optimize database
	if _, err := d.conn.Exec("VACUUM"); err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	// Run ANALYZE to update query planner statistics
	if _, err := d.conn.Exec("ANALYZE"); err != nil {
		return fmt.Errorf("failed to analyze database: %w", err)
	}

	log.Println("Database vacuum and optimization completed")
	return nil
}

// GetDatabaseStats returns database performance statistics
func (d *Database) GetDatabaseStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get table sizes
	tables := []string{
		"providers", "models", "pricing", "limits", "verification_results",
		"issues", "events", "schedules", "config_exports", "logs",
	}

	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err := d.conn.QueryRow(query).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to get count for %s: %w", table, err)
		}
		stats[table+"_count"] = count
	}

	// Get database file size (approximate)
	var pageCount int
	err := d.conn.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	var pageSize int
	err = d.conn.QueryRow("PRAGMA page_size").Scan(&pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get page size: %w", err)
	}

	stats["database_size_bytes"] = pageCount * pageSize
	stats["database_size_mb"] = float64(pageCount*pageSize) / (1024 * 1024)

	return stats, nil
}
