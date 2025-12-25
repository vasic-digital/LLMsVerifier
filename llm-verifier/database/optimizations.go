package database

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// ==================== Query Optimizations ====================

// QueryOptimizer provides query optimization and performance monitoring
type QueryOptimizer struct {
	db           *Database
	queryStats   map[string]*QueryStat
	slowQueryLog []*SlowQuery
	mutex        sync.RWMutex
}

// QueryStat holds statistics for a query
type QueryStat struct {
	Query     string
	Count     int64
	TotalTime time.Duration
	AvgTime   time.Duration
	MaxTime   time.Duration
	LastRun   time.Time
	SlowCount int64
}

// SlowQuery represents a slow query entry
type SlowQuery struct {
	Query     string
	Duration  time.Duration
	Timestamp time.Time
	Args      []interface{}
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *Database) *QueryOptimizer {
	return &QueryOptimizer{
		db:           db,
		queryStats:   make(map[string]*QueryStat),
		slowQueryLog: make([]*SlowQuery, 0),
	}
}

// AnalyzeQueryPerformance analyzes the performance of a query and tracks statistics
func (qo *QueryOptimizer) AnalyzeQueryPerformance(ctx context.Context, query string, args ...interface{}) (time.Duration, error) {
	start := time.Now()

	// Execute query with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := qo.db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		qo.recordQueryStat(query, time.Since(start), true)
		return 0, err
	}
	defer rows.Close()

	// Consume all rows to complete the query
	for rows.Next() {
		// Just iterate, don't process data for analysis
	}

	if err := rows.Err(); err != nil {
		qo.recordQueryStat(query, time.Since(start), true)
		return 0, err
	}

	duration := time.Since(start)
	qo.recordQueryStat(query, duration, false)

	// Log slow queries
	if duration > 100*time.Millisecond {
		qo.logSlowQuery(query, duration, args...)
	}

	return duration, nil
}

// recordQueryStat records query performance statistics
func (qo *QueryOptimizer) recordQueryStat(query string, duration time.Duration, hadError bool) {
	qo.mutex.Lock()
	defer qo.mutex.Unlock()

	queryHash := qo.hashQuery(query)
	stat, exists := qo.queryStats[queryHash]
	if !exists {
		stat = &QueryStat{
			Query:     query,
			Count:     0,
			TotalTime: 0,
			MaxTime:   0,
		}
		qo.queryStats[queryHash] = stat
	}

	stat.Count++
	stat.TotalTime += duration
	stat.AvgTime = stat.TotalTime / time.Duration(stat.Count)
	stat.LastRun = time.Now()

	if duration > stat.MaxTime {
		stat.MaxTime = duration
	}

	if hadError || duration > 500*time.Millisecond {
		stat.SlowCount++
	}
}

// logSlowQuery logs a slow query for analysis
func (qo *QueryOptimizer) logSlowQuery(query string, duration time.Duration, args ...interface{}) {
	qo.mutex.Lock()
	defer qo.mutex.Unlock()

	slowQuery := &SlowQuery{
		Query:     query,
		Duration:  duration,
		Timestamp: time.Now(),
		Args:      args,
	}

	// Keep only last 100 slow queries
	qo.slowQueryLog = append(qo.slowQueryLog, slowQuery)
	if len(qo.slowQueryLog) > 100 {
		qo.slowQueryLog = qo.slowQueryLog[1:]
	}

	log.Printf("SLOW QUERY: %v - %s", duration, qo.truncateQuery(query))
}

// hashQuery creates a hash of the query for indexing
func (qo *QueryOptimizer) hashQuery(query string) string {
	// Normalize query by removing extra whitespace
	normalized := strings.Join(strings.Fields(query), " ")
	hash := md5.Sum([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}

// truncateQuery truncates a query for logging
func (qo *QueryOptimizer) truncateQuery(query string) string {
	if len(query) > 100 {
		return query[:100] + "..."
	}
	return query
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
			qo.suggestOptimization(q.name, q.query)
		} else {
			log.Printf("Query %s performed well: %v", q.name, duration)
		}
	}
}

// GetQueryStats returns query performance statistics
func (qo *QueryOptimizer) GetQueryStats() map[string]*QueryStat {
	qo.mutex.RLock()
	defer qo.mutex.RUnlock()

	stats := make(map[string]*QueryStat)
	for k, v := range qo.queryStats {
		statCopy := *v
		stats[k] = &statCopy
	}

	return stats
}

// GetSlowQueries returns the slow query log
func (qo *QueryOptimizer) GetSlowQueries() []*SlowQuery {
	qo.mutex.RLock()
	defer qo.mutex.RUnlock()

	queries := make([]*SlowQuery, len(qo.slowQueryLog))
	for i, q := range qo.slowQueryLog {
		queryCopy := *q
		queries[i] = &queryCopy
	}

	return queries
}

// suggestOptimization suggests optimization for a slow query
func (qo *QueryOptimizer) suggestOptimization(queryName, query string) {
	log.Printf("Optimization suggestion for %s:", queryName)

	switch {
	case strings.Contains(query, "verification_status = ?"):
		log.Println("  - Consider adding index on verification_status")
		log.Println("  - Consider partitioning table by verification_status")

	case strings.Contains(query, "provider_id = ?"):
		log.Println("  - Ensure index exists on provider_id")
		log.Println("  - Consider composite index (provider_id, verification_status)")

	case strings.Contains(query, "name LIKE ?"):
		log.Println("  - Full table scan detected - consider full-text search index")
		log.Println("  - Consider normalizing search terms")

	case strings.Contains(query, "LEFT JOIN") || strings.Contains(query, "JOIN"):
		log.Println("  - Complex join detected - ensure foreign key indexes exist")
		log.Println("  - Consider denormalization for read-heavy queries")

	default:
		log.Println("  - Enable query execution plan analysis: PRAGMA vdbe_trace=ON")
		log.Println("  - Check for missing indexes on WHERE clause columns")
	}
}

// AnalyzeTableStatistics analyzes table statistics for optimization
func (qo *QueryOptimizer) AnalyzeTableStatistics() error {
	tables := []string{"models", "verification_results", "providers", "events"}

	for _, table := range tables {
		if err := qo.analyzeTable(table); err != nil {
			log.Printf("Failed to analyze table %s: %v", table, err)
		}
	}

	return nil
}

// analyzeTable analyzes a specific table for optimization opportunities
func (qo *QueryOptimizer) analyzeTable(tableName string) error {
	// Get row count
	var count int64
	err := qo.db.conn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	if err != nil {
		return err
	}

	log.Printf("Table %s: %d rows", tableName, count)

	// Analyze index usage (SQLite specific)
	rows, err := qo.db.conn.Query(fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	indexes := 0
	for rows.Next() {
		indexes++
	}

	log.Printf("Table %s: %d indexes", tableName, indexes)

	// Suggest optimizations based on analysis
	if count > 10000 && indexes == 0 {
		log.Printf("OPTIMIZATION: Table %s has %d rows but no indexes - consider adding primary key index", tableName, count)
	}

	if count > 100000 && indexes < 3 {
		log.Printf("OPTIMIZATION: Table %s has %d rows - consider adding more indexes for query performance", tableName, count)
	}

	return nil
}

// IndexManager provides index management and optimization
type IndexManager struct {
	db *Database
}

// NewIndexManager creates a new index manager
func NewIndexManager(db *Database) *IndexManager {
	return &IndexManager{db: db}
}

// AnalyzeIndexUsage analyzes how indexes are being used
func (im *IndexManager) AnalyzeIndexUsage() error {
	// Get all tables
	tables := []string{"models", "verification_results", "providers", "events", "issues", "pricing"}

	for _, table := range tables {
		if err := im.analyzeTableIndexes(table); err != nil {
			log.Printf("Failed to analyze indexes for table %s: %v", table, err)
		}
	}

	return nil
}

// analyzeTableIndexes analyzes indexes for a specific table
func (im *IndexManager) analyzeTableIndexes(tableName string) error {
	// Get index information
	rows, err := im.db.conn.Query("PRAGMA index_list(?)", tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	indexes := []map[string]interface{}{}
	for rows.Next() {
		var seq int
		var name, unique string
		var origin, partial string

		err := rows.Scan(&seq, &name, &unique, &origin, &partial)
		if err != nil {
			continue
		}

		indexInfo := map[string]interface{}{
			"name":    name,
			"unique":  unique,
			"origin":  origin,
			"partial": partial,
		}

		// Get index columns
		colRows, err := im.db.conn.Query("PRAGMA index_info(?)", name)
		if err == nil {
			defer colRows.Close()

			columns := []string{}
			for colRows.Next() {
				var seqno, cid int
				var name string
				colRows.Scan(&seqno, &cid, &name)
				columns = append(columns, name)
			}
			indexInfo["columns"] = columns
		}

		indexes = append(indexes, indexInfo)
	}

	// Analyze index effectiveness
	for _, index := range indexes {
		if name, ok := index["name"].(string); ok {
			im.analyzeIndexEffectiveness(tableName, name, index)
		}
	}

	return nil
}

// analyzeIndexEffectiveness analyzes how effective an index is
func (im *IndexManager) analyzeIndexEffectiveness(tableName, indexName string, indexInfo map[string]interface{}) {
	// This is a simplified analysis - in production, you'd analyze actual query plans
	// and index usage statistics

	log.Printf("Analyzing index %s on table %s", indexName, tableName)

	// Check if index is on commonly queried columns
	if columns, ok := indexInfo["columns"].([]string); ok {
		for _, col := range columns {
			switch col {
			case "verification_status", "status", "provider_id", "model_id", "created_at":
				log.Printf("  ✓ Index %s covers frequently queried column: %s", indexName, col)
			}
		}
	}

	// Check if index is unique (generally good for performance)
	if unique, ok := indexInfo["unique"].(string); ok && unique == "1" {
		log.Printf("  ✓ Index %s is unique - good for performance", indexName)
	}
}

// OptimizeIndexes creates additional indexes based on query patterns
func (im *IndexManager) OptimizeIndexes() error {
	optimizations := []string{
		// Text search indexes
		"CREATE INDEX IF NOT EXISTS idx_models_name_search ON models(name COLLATE NOCASE)",
		"CREATE INDEX IF NOT EXISTS idx_providers_name_search ON providers(name COLLATE NOCASE)",

		// Range query optimizations
		"CREATE INDEX IF NOT EXISTS idx_verification_results_score_range ON verification_results(overall_score DESC)",
		"CREATE INDEX IF NOT EXISTS idx_events_type_time ON events(event_type, timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_issues_severity_status ON issues(severity, resolved_at)",

		// Foreign key optimizations
		"CREATE INDEX IF NOT EXISTS idx_verification_results_provider ON verification_results(provider_id)",
		"CREATE INDEX IF NOT EXISTS idx_pricing_provider ON pricing(provider_id)",

		// Partial indexes for active records
		"CREATE INDEX IF NOT EXISTS idx_models_active ON models(id) WHERE verification_status != 'deprecated'",
		"CREATE INDEX IF NOT EXISTS idx_issues_unresolved ON issues(id) WHERE resolved_at IS NULL",
	}

	for _, sql := range optimizations {
		if _, err := im.db.conn.Exec(sql); err != nil {
			log.Printf("Failed to create optimization index: %v", err)
			continue
		}
		log.Printf("Created optimization index: %s", sql[:50]+"...")
	}

	return nil
}

// GetIndexStatistics returns index usage statistics
func (im *IndexManager) GetIndexStatistics() (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"total_indexes":    0,
		"indexes_by_table": make(map[string]int),
		"largest_indexes":  []string{},
	}

	// Get all tables
	rows, err := im.db.conn.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := []string{}
	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		tables = append(tables, tableName)
	}

	// Analyze each table
	for _, table := range tables {
		indexRows, err := im.db.conn.Query("PRAGMA index_list(?)", table)
		if err != nil {
			continue
		}

		indexCount := 0
		for indexRows.Next() {
			indexCount++
		}
		indexRows.Close()

		stats["total_indexes"] = stats["total_indexes"].(int) + indexCount
		stats["indexes_by_table"].(map[string]int)[table] = indexCount
	}

	return stats, nil
}

// CleanupUnusedIndexes identifies and suggests removal of unused indexes
func (im *IndexManager) CleanupUnusedIndexes() error {
	// In SQLite, it's harder to track index usage than in other databases
	// This provides basic suggestions based on index analysis

	log.Println("Analyzing indexes for potential cleanup...")

	tables := []string{"models", "verification_results", "providers"}

	for _, table := range tables {
		rows, err := im.db.conn.Query("PRAGMA index_list(?)", table)
		if err != nil {
			continue
		}

		for rows.Next() {
			var seq int
			var name, unique, origin, partial string
			rows.Scan(&seq, &name, &unique, &origin, &partial)

			// Skip primary key and unique indexes
			if unique == "1" || strings.Contains(name, "_pk") {
				continue
			}

			// This is where you'd check actual usage statistics
			// For now, just log potential candidates
			log.Printf("Potential cleanup candidate: index %s on table %s", name, table)
		}
		rows.Close()
	}

	return nil
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
		// Validate table name to prevent SQL injection
		quotedTable, err := QuoteTableName(table)
		if err != nil {
			return nil, fmt.Errorf("invalid table name: %w", err)
		}

		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable)
		err = d.conn.QueryRow(query).Scan(&count)
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
