package scoring

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"llm-verifier/database"
)

// ScoringDatabaseExtensions extends the database with scoring-specific functionality
type ScoringDatabaseExtensions struct {
	db *database.Database
}

// NewScoringDatabaseExtensions creates new scoring database extensions
func NewScoringDatabaseExtensions(db *database.Database) *ScoringDatabaseExtensions {
	return &ScoringDatabaseExtensions{db: db}
}

// InitializeScoringSchema creates the scoring-related database tables
func (sde *ScoringDatabaseExtensions) InitializeScoringSchema() error {
	schema := `
	-- Comprehensive model scores table
	CREATE TABLE IF NOT EXISTS model_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		overall_score REAL NOT NULL,
		speed_score REAL NOT NULL,
		efficiency_score REAL NOT NULL,
		cost_score REAL NOT NULL,
		capability_score REAL NOT NULL,
		recency_score REAL NOT NULL,
		score_suffix TEXT NOT NULL,
		calculation_hash TEXT NOT NULL,
		calculation_details TEXT,
		last_calculated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		valid_until TIMESTAMP,
		is_active BOOLEAN DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Model performance metrics table
	CREATE TABLE IF NOT EXISTS model_performance_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		metric_type TEXT NOT NULL,
		metric_value REAL NOT NULL,
		metric_unit TEXT,
		sample_count INTEGER DEFAULT 1,
		p50_value REAL,
		p95_value REAL,
		p99_value REAL,
		min_value REAL,
		max_value REAL,
		std_dev REAL,
		measured_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		measurement_window_seconds INTEGER DEFAULT 3600,
		metadata TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Model cost tracking table
	CREATE TABLE IF NOT EXISTS model_cost_tracking (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		cost_type TEXT NOT NULL,
		cost_per_unit REAL NOT NULL,
		currency TEXT DEFAULT 'USD',
		unit_type TEXT NOT NULL,
		effective_from TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		effective_to TIMESTAMP,
		source TEXT,
		is_current BOOLEAN DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Model feature capabilities table
	CREATE TABLE IF NOT EXISTS model_capabilities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		capability_type TEXT NOT NULL,
		capability_name TEXT NOT NULL,
		is_supported BOOLEAN DEFAULT 0,
		support_level TEXT DEFAULT 'none',
		detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		confidence_score REAL DEFAULT 1.0,
		test_results TEXT,
		last_verified TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Model scoring history table
	CREATE TABLE IF NOT EXISTS model_scoring_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		previous_score REAL,
		new_score REAL,
		score_change REAL,
		change_reason TEXT,
		components_changed TEXT,
		triggered_by TEXT,
		calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Scoring configuration table
	CREATE TABLE IF NOT EXISTS scoring_configuration (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		config_name TEXT NOT NULL UNIQUE,
		config_data TEXT NOT NULL,
		is_active BOOLEAN DEFAULT 1,
		created_by TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Model metadata from external sources
	CREATE TABLE IF NOT EXISTS model_external_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		source_name TEXT NOT NULL,
		source_id TEXT NOT NULL,
		metadata_type TEXT NOT NULL,
		metadata_data TEXT NOT NULL,
		fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		valid_until TIMESTAMP,
		confidence_score REAL DEFAULT 1.0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_model_scores_model ON model_scores(model_id);
	CREATE INDEX IF NOT EXISTS idx_model_scores_overall ON model_scores(overall_score);
	CREATE INDEX IF NOT EXISTS idx_model_scores_active ON model_scores(is_active);
	CREATE INDEX IF NOT EXISTS idx_model_scores_calculated ON model_scores(last_calculated);
	CREATE INDEX IF NOT EXISTS idx_model_performance_metrics_model ON model_performance_metrics(model_id);
	CREATE INDEX IF NOT EXISTS idx_model_performance_metrics_type ON model_performance_metrics(metric_type);
	CREATE INDEX IF NOT EXISTS idx_model_performance_metrics_measured ON model_performance_metrics(measured_at);
	CREATE INDEX IF NOT EXISTS idx_model_cost_tracking_model ON model_cost_tracking(model_id);
	CREATE INDEX IF NOT EXISTS idx_model_cost_tracking_current ON model_cost_tracking(is_current);
	CREATE INDEX IF NOT EXISTS idx_model_capabilities_model ON model_capabilities(model_id);
	CREATE INDEX IF NOT EXISTS idx_model_capabilities_type ON model_capabilities(capability_type);
	CREATE INDEX IF NOT EXISTS idx_model_scoring_history_model ON model_scoring_history(model_id);
	CREATE INDEX IF NOT EXISTS idx_model_scoring_history_calculated ON model_scoring_history(calculated_at);
	CREATE INDEX IF EXISTS idx_model_external_metadata_model ON model_external_metadata(model_id);
	CREATE INDEX IF NOT EXISTS idx_model_external_metadata_source ON model_external_metadata(source_name, source_id);
	`

	_, err := sde.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize scoring schema: %w", err)
	}

	// Insert default scoring configuration
	defaultConfig := DefaultScoringConfig()
	configData, err := json.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	query := `INSERT OR REPLACE INTO scoring_configuration (config_name, config_data, is_active, created_by) VALUES (?, ?, ?, ?)`
	_, err = sde.db.Exec(query, "default", string(configData), true, "system")
	if err != nil {
		return fmt.Errorf("failed to insert default config: %w", err)
	}

	return nil
}

// ModelScore represents a comprehensive model score
type ModelScore struct {
	ID                int64     `json:"id"`
	ModelID           int64     `json:"model_id"`
	OverallScore      float64   `json:"overall_score"`
	SpeedScore        float64   `json:"speed_score"`
	EfficiencyScore   float64   `json:"efficiency_score"`
	CostScore         float64   `json:"cost_score"`
	CapabilityScore   float64   `json:"capability_score"`
	RecencyScore      float64   `json:"recency_score"`
	ScoreSuffix       string    `json:"score_suffix"`
	CalculationHash   string    `json:"calculation_hash"`
	CalculationDetails string   `json:"calculation_details,omitempty"`
	LastCalculated    time.Time `json:"last_calculated"`
	ValidUntil        *time.Time `json:"valid_until,omitempty"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// PerformanceMetric represents a performance measurement
type PerformanceMetric struct {
	ID                     int64      `json:"id"`
	ModelID                int64      `json:"model_id"`
	MetricType             string     `json:"metric_type"`
	MetricValue            float64    `json:"metric_value"`
	MetricUnit             *string    `json:"metric_unit,omitempty"`
	SampleCount            int        `json:"sample_count"`
	P50Value               *float64   `json:"p50_value,omitempty"`
	P95Value               *float64   `json:"p95_value,omitempty"`
	P99Value               *float64   `json:"p99_value,omitempty"`
	MinValue               *float64   `json:"min_value,omitempty"`
	MaxValue               *float64   `json:"max_value,omitempty"`
	StdDev                 *float64   `json:"std_dev,omitempty"`
	MeasuredAt             time.Time  `json:"measured_at"`
	MeasurementWindowSeconds int      `json:"measurement_window_seconds"`
	Metadata               *string    `json:"metadata,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
}

// ModelCost represents cost information for a model
type ModelCost struct {
	ID            int64      `json:"id"`
	ModelID       int64      `json:"model_id"`
	CostType      string     `json:"cost_type"`
	CostPerUnit   float64    `json:"cost_per_unit"`
	Currency      string     `json:"currency"`
	UnitType      string     `json:"unit_type"`
	EffectiveFrom time.Time  `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty"`
	Source        *string    `json:"source,omitempty"`
	IsCurrent     bool       `json:"is_current"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// CreateModelScore creates a new model score record
func (sde *ScoringDatabaseExtensions) CreateModelScore(score *ModelScore) error {
	query := `
		INSERT INTO model_scores (
			model_id, overall_score, speed_score, efficiency_score, cost_score,
			capability_score, recency_score, score_suffix, calculation_hash,
			calculation_details, last_calculated, valid_until, is_active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := sde.db.Exec(query,
		score.ModelID,
		score.OverallScore,
		score.SpeedScore,
		score.EfficiencyScore,
		score.CostScore,
		score.CapabilityScore,
		score.RecencyScore,
		score.ScoreSuffix,
		score.CalculationHash,
		score.CalculationDetails,
		score.LastCalculated,
		score.ValidUntil,
		score.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to create model score: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	score.ID = id
	return nil
}

// GetLatestModelScore retrieves the latest score for a model
func (sde *ScoringDatabaseExtensions) GetLatestModelScore(modelID int64) (*ModelScore, error) {
	query := `
		SELECT id, model_id, overall_score, speed_score, efficiency_score, cost_score,
			   capability_score, recency_score, score_suffix, calculation_hash,
			   calculation_details, last_calculated, valid_until, is_active,
			   created_at, updated_at
		FROM model_scores
		WHERE model_id = ? AND is_active = 1
		ORDER BY last_calculated DESC
		LIMIT 1
	`

	var score ModelScore
	var validUntil sql.NullTime

	err := sde.db.QueryRow(query, modelID).Scan(
		&score.ID, &score.ModelID, &score.OverallScore, &score.SpeedScore,
		&score.EfficiencyScore, &score.CostScore, &score.CapabilityScore,
		&score.RecencyScore, &score.ScoreSuffix, &score.CalculationHash,
		&score.CalculationDetails, &score.LastCalculated, &validUntil,
		&score.IsActive, &score.CreatedAt, &score.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No score found
		}
		return nil, fmt.Errorf("failed to get latest model score: %w", err)
	}

	if validUntil.Valid {
		score.ValidUntil = &validUntil.Time
	}

	return &score, nil
}

// GetModelScoresByRange retrieves models within a score range
func (sde *ScoringDatabaseExtensions) GetModelScoresByRange(minScore, maxScore float64, limit int) ([]ModelScore, error) {
	query := `
		SELECT ms.id, ms.model_id, ms.overall_score, ms.speed_score, ms.efficiency_score,
			   ms.cost_score, ms.capability_score, ms.recency_score, ms.score_suffix,
			   ms.calculation_hash, ms.calculation_details, ms.last_calculated,
			   ms.valid_until, ms.is_active, ms.created_at, ms.updated_at,
			   m.name as model_name, m.model_id as model_id_string
		FROM model_scores ms
		JOIN models m ON ms.model_id = m.id
		WHERE ms.overall_score >= ? AND ms.overall_score <= ? AND ms.is_active = 1
		ORDER BY ms.overall_score DESC
		LIMIT ?
	`

	rows, err := sde.db.Query(query, minScore, maxScore, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get model scores by range: %w", err)
	}
	defer rows.Close()

	var scores []ModelScore
	for rows.Next() {
		var score ModelScore
		var validUntil sql.NullTime
		var modelName string
		var modelIDString string

		err := rows.Scan(
			&score.ID, &score.ModelID, &score.OverallScore, &score.SpeedScore,
			&score.EfficiencyScore, &score.CostScore, &score.CapabilityScore,
			&score.RecencyScore, &score.ScoreSuffix, &score.CalculationHash,
			&score.CalculationDetails, &score.LastCalculated, &validUntil,
			&score.IsActive, &score.CreatedAt, &score.UpdatedAt,
			&modelName, &modelIDString,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model score: %w", err)
		}

		if validUntil.Valid {
			score.ValidUntil = &validUntil.Time
		}

		scores = append(scores, score)
	}

	return scores, rows.Err()
}

// CreatePerformanceMetric creates a new performance metric record
func (sde *ScoringDatabaseExtensions) CreatePerformanceMetric(metric *PerformanceMetric) error {
	query := `
		INSERT INTO model_performance_metrics (
			model_id, metric_type, metric_value, metric_unit, sample_count,
			p50_value, p95_value, p99_value, min_value, max_value, std_dev,
			measured_at, measurement_window_seconds, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := sde.db.Exec(query,
		metric.ModelID,
		metric.MetricType,
		metric.MetricValue,
		metric.MetricUnit,
		metric.SampleCount,
		metric.P50Value,
		metric.P95Value,
		metric.P99Value,
		metric.MinValue,
		metric.MaxValue,
		metric.StdDev,
		metric.MeasuredAt,
		metric.MeasurementWindowSeconds,
		metric.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create performance metric: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	metric.ID = id
	return nil
}

// GetPerformanceMetrics retrieves performance metrics for a model
func (sde *ScoringDatabaseExtensions) GetPerformanceMetrics(modelID int64, metricType string, timeWindow time.Duration) ([]PerformanceMetric, error) {
	query := `
		SELECT id, model_id, metric_type, metric_value, metric_unit, sample_count,
			   p50_value, p95_value, p99_value, min_value, max_value, std_dev,
			   measured_at, measurement_window_seconds, metadata, created_at
		FROM model_performance_metrics
		WHERE model_id = ? AND metric_type = ?
			AND measured_at >= datetime('now', '-' || ? || ' seconds')
		ORDER BY measured_at DESC
	`

	rows, err := sde.db.Query(query, modelID, metricType, int(timeWindow.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("failed to get performance metrics: %w", err)
	}
	defer rows.Close()

	var metrics []PerformanceMetric
	for rows.Next() {
		var metric PerformanceMetric
		var p50, p95, p99, min, max, stdDev, unit, metadata sql.NullFloat64
		var unitStr, metadataStr *string

		err := rows.Scan(
			&metric.ID, &metric.ModelID, &metric.MetricType, &metric.MetricValue,
			&unit, &metric.SampleCount, &p50, &p95, &p99, &min, &max, &stdDev,
			&metric.MeasuredAt, &metric.MeasurementWindowSeconds, &metadata,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan performance metric: %w", err)
		}

		if unit.Valid {
			unitStr = new(string)
			*unitStr = fmt.Sprintf("%.2f", unit.Float64)
			metric.MetricUnit = unitStr
		}

		if p50.Valid {
			metric.P50Value = new(float64)
			*metric.P50Value = p50.Float64
		}
		if p95.Valid {
			metric.P95Value = new(float64)
			*metric.P95Value = p95.Float64
		}
		if p99.Valid {
			metric.P99Value = new(float64)
			*metric.P99Value = p99.Float64
		}
		if min.Valid {
			metric.MinValue = new(float64)
			*metric.MinValue = min.Float64
		}
		if max.Valid {
			metric.MaxValue = new(float64)
			*metric.MaxValue = max.Float64
		}
		if stdDev.Valid {
			metric.StdDev = new(float64)
			*metric.StdDev = stdDev.Float64
		}
		if metadata.Valid {
			metadataStr = new(string)
			*metadataStr = fmt.Sprintf("%.2f", metadata.Float64)
			metric.Metadata = metadataStr
		}

		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}

// CreateModelCost creates a new model cost record
func (sde *ScoringDatabaseExtensions) CreateModelCost(cost *ModelCost) error {
	query := `
		INSERT INTO model_cost_tracking (
			model_id, cost_type, cost_per_unit, currency, unit_type,
			effective_from, effective_to, source, is_current
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := sde.db.Exec(query,
		cost.ModelID,
		cost.CostType,
		cost.CostPerUnit,
		cost.Currency,
		cost.UnitType,
		cost.EffectiveFrom,
		cost.EffectiveTo,
		cost.Source,
		cost.IsCurrent,
	)

	if err != nil {
		return fmt.Errorf("failed to create model cost: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	cost.ID = id
	return nil
}

// GetCurrentModelCosts retrieves current costs for a model
func (sde *ScoringDatabaseExtensions) GetCurrentModelCosts(modelID int64) ([]ModelCost, error) {
	query := `
		SELECT id, model_id, cost_type, cost_per_unit, currency, unit_type,
			   effective_from, effective_to, source, is_current, created_at, updated_at
		FROM model_cost_tracking
		WHERE model_id = ? AND is_current = 1
		ORDER BY cost_type
	`

	rows, err := sde.db.Query(query, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current model costs: %w", err)
	}
	defer rows.Close()

	var costs []ModelCost
	for rows.Next() {
		var cost ModelCost
		var effectiveTo sql.NullTime
		var source sql.NullString

		err := rows.Scan(
			&cost.ID, &cost.ModelID, &cost.CostType, &cost.CostPerUnit,
			&cost.Currency, &cost.UnitType, &cost.EffectiveFrom, &effectiveTo,
			&source, &cost.IsCurrent, &cost.CreatedAt, &cost.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model cost: %w", err)
		}

		if effectiveTo.Valid {
			cost.EffectiveTo = &effectiveTo.Time
		}
		if source.Valid {
			cost.Source = &source.String
		}

		costs = append(costs, cost)
	}

	return costs, rows.Err()
}

// GetScoringConfiguration retrieves scoring configuration by name
func (sde *ScoringDatabaseExtensions) GetScoringConfiguration(configName string) (*ScoringConfig, error) {
	query := `
		SELECT config_data
		FROM scoring_configuration
		WHERE config_name = ? AND is_active = 1
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var configData string
	err := sde.db.QueryRow(query, configName).Scan(&configData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No config found
		}
		return nil, fmt.Errorf("failed to get scoring configuration: %w", err)
	}

	var config ScoringConfig
	if err := json.Unmarshal([]byte(configData), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scoring configuration: %w", err)
	}

	return &config, nil
}

// StoreModelExternalMetadata stores external metadata for a model
func (sde *ScoringDatabaseExtensions) StoreModelExternalMetadata(modelID int64, sourceName, sourceID, metadataType string, metadataData interface{}, validUntil *time.Time, confidenceScore float64) error {
	metadataJSON, err := json.Marshal(metadataData)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO model_external_metadata (
			model_id, source_name, source_id, metadata_type, metadata_data,
			valid_until, confidence_score
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = sde.db.Exec(query,
		modelID, sourceName, sourceID, metadataType, string(metadataJSON),
		validUntil, confidenceScore,
	)

	if err != nil {
		return fmt.Errorf("failed to store model external metadata: %w", err)
	}

	return nil
}

// GetModelExternalMetadata retrieves external metadata for a model
func (sde *ScoringDatabaseExtensions) GetModelExternalMetadata(modelID int64, sourceName, metadataType string) (interface{}, error) {
	query := `
		SELECT metadata_data
		FROM model_external_metadata
		WHERE model_id = ? AND source_name = ? AND metadata_type = ?
			AND (valid_until IS NULL OR valid_until > datetime('now'))
		ORDER BY fetched_at DESC
		LIMIT 1
	`

	var metadataData string
	err := sde.db.QueryRow(query, modelID, sourceName, metadataType).Scan(&metadataData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No metadata found
		}
		return nil, fmt.Errorf("failed to get model external metadata: %w", err)
	}

	var metadata interface{}
	if err := json.Unmarshal([]byte(metadataData), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return metadata, nil
}