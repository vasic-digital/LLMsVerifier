package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ==================== Verification Scores CRUD Operations ====================

// VerificationScore represents a model verification score
type VerificationScore struct {
	ID                   int64  `json:"id"`
	ModelID              int64  `json:"model_id"`
	VerificationResultID *int64 `json:"verification_result_id,omitempty"`
	Score                int    `json:"score"`          // 0-100 overall score
	ScoreType            string `json:"score_type"`     // coding_capability, accuracy, speed, reliability
	ScoringMethod        string `json:"scoring_method"` // benchmark, manual, auto, hybrid
	Category             string `json:"category"`       // fully_coding_capable, coding_with_tools, chat_with_tooling, chat_only

	// Detailed benchmark scores (0-100 each)
	CodeCorrectnessScore      *int `json:"code_correctness_score,omitempty"`
	CodeQualityScore          *int `json:"code_quality_score,omitempty"`
	CodeSpeedScore            *int `json:"code_speed_score,omitempty"`
	ErrorHandlingScore        *int `json:"error_handling_score,omitempty"`
	ContextUnderstandingScore *int `json:"context_understanding_score,omitempty"`

	// Evidence and metadata
	Evidence         string `json:"evidence,omitempty"` // JSON with test results
	BenchmarkVersion string `json:"benchmark_version"`
	ScoredBy         string `json:"scored_by"`
	ConfidenceLevel  int    `json:"confidence_level"` // 0-100 confidence in score

	ScoredAt  time.Time  `json:"scored_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CreateVerificationScore creates a new verification score
func (d *Database) CreateVerificationScore(score *VerificationScore) (int64, error) {
	query := `
		INSERT INTO verification_scores (
			model_id, verification_result_id, score, score_type, scoring_method, category,
			code_correctness_score, code_quality_score, code_speed_score, error_handling_score,
			context_understanding_score, evidence, benchmark_version, scored_by, confidence_level,
			scored_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	expiresAt := sql.NullTime{}
	if score.ExpiresAt != nil {
		expiresAt.Valid = true
		expiresAt.Time = *score.ExpiresAt
	}

	result, err := d.conn.Exec(query,
		score.ModelID,
		score.VerificationResultID,
		score.Score,
		score.ScoreType,
		score.ScoringMethod,
		score.Category,
		score.CodeCorrectnessScore,
		score.CodeQualityScore,
		score.CodeSpeedScore,
		score.ErrorHandlingScore,
		score.ContextUnderstandingScore,
		score.Evidence,
		score.BenchmarkVersion,
		score.ScoredBy,
		score.ConfidenceLevel,
		score.ScoredAt,
		expiresAt,
	)

	if err != nil {
		return 0, fmt.Errorf("failed to create verification score: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return id, nil
}

// GetVerificationScore retrieves a verification score by ID
func (d *Database) GetVerificationScore(id int64) (*VerificationScore, error) {
	query := `
		SELECT id, model_id, verification_result_id, score, score_type, scoring_method, category,
			code_correctness_score, code_quality_score, code_speed_score, error_handling_score,
			context_understanding_score, evidence, benchmark_version, scored_by, confidence_level,
			scored_at, expires_at, created_at, updated_at
		FROM verification_scores WHERE id = ?
	`

	var score VerificationScore
	var expiresAt sql.NullTime

	err := d.conn.QueryRow(query, id).Scan(
		&score.ID,
		&score.ModelID,
		&score.VerificationResultID,
		&score.Score,
		&score.ScoreType,
		&score.ScoringMethod,
		&score.Category,
		&score.CodeCorrectnessScore,
		&score.CodeQualityScore,
		&score.CodeSpeedScore,
		&score.ErrorHandlingScore,
		&score.ContextUnderstandingScore,
		&score.Evidence,
		&score.BenchmarkVersion,
		&score.ScoredBy,
		&score.ConfidenceLevel,
		&score.ScoredAt,
		&expiresAt,
		&score.CreatedAt,
		&score.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get verification score: %w", err)
	}

	if expiresAt.Valid {
		score.ExpiresAt = &expiresAt.Time
	}

	return &score, nil
}

// GetLatestVerificationScore gets the latest score for a model
func (d *Database) GetLatestVerificationScore(modelID int64, scoreType string) (*VerificationScore, error) {
	query := `
		SELECT id, model_id, verification_result_id, score, score_type, scoring_method, category,
			code_correctness_score, code_quality_score, code_speed_score, error_handling_score,
			context_understanding_score, evidence, benchmark_version, scored_by, confidence_level,
			scored_at, expires_at, created_at, updated_at
		FROM verification_scores
		WHERE model_id = ? AND score_type = ?
		ORDER BY scored_at DESC LIMIT 1
	`

	var score VerificationScore
	var expiresAt sql.NullTime

	err := d.conn.QueryRow(query, modelID, scoreType).Scan(
		&score.ID,
		&score.ModelID,
		&score.VerificationResultID,
		&score.Score,
		&score.ScoreType,
		&score.ScoringMethod,
		&score.Category,
		&score.CodeCorrectnessScore,
		&score.CodeQualityScore,
		&score.CodeSpeedScore,
		&score.ErrorHandlingScore,
		&score.ContextUnderstandingScore,
		&score.Evidence,
		&score.BenchmarkVersion,
		&score.ScoredBy,
		&score.ConfidenceLevel,
		&score.ScoredAt,
		&expiresAt,
		&score.CreatedAt,
		&score.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get latest verification score: %w", err)
	}

	if expiresAt.Valid {
		score.ExpiresAt = &expiresAt.Time
	}

	return &score, nil
}

// ListVerificationScores retrieves scores with optional filtering
func (d *Database) ListVerificationScores(filters map[string]interface{}) ([]*VerificationScore, error) {
	query := `
		SELECT id, model_id, verification_result_id, score, score_type, scoring_method, category,
			code_correctness_score, code_quality_score, code_speed_score, error_handling_score,
			context_understanding_score, evidence, benchmark_version, scored_by, confidence_level,
			scored_at, expires_at, created_at, updated_at
		FROM verification_scores
	`

	var conditions []string
	var args []interface{}

	if modelID, ok := filters["model_id"]; ok && modelID != nil {
		conditions = append(conditions, "model_id = ?")
		args = append(args, modelID)
	}

	if scoreType, ok := filters["score_type"]; ok && scoreType != "" {
		conditions = append(conditions, "score_type = ?")
		args = append(args, scoreType)
	}

	if category, ok := filters["category"]; ok && category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, category)
	}

	if minScore, ok := filters["min_score"]; ok && minScore != nil {
		conditions = append(conditions, "score >= ?")
		args = append(args, minScore)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY scored_at DESC"

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list verification scores: %w", err)
	}
	defer rows.Close()

	var scores []*VerificationScore
	for rows.Next() {
		var score VerificationScore
		var expiresAt sql.NullTime

		err := rows.Scan(
			&score.ID,
			&score.ModelID,
			&score.VerificationResultID,
			&score.Score,
			&score.ScoreType,
			&score.ScoringMethod,
			&score.Category,
			&score.CodeCorrectnessScore,
			&score.CodeQualityScore,
			&score.CodeSpeedScore,
			&score.ErrorHandlingScore,
			&score.ContextUnderstandingScore,
			&score.Evidence,
			&score.BenchmarkVersion,
			&score.ScoredBy,
			&score.ConfidenceLevel,
			&score.ScoredAt,
			&expiresAt,
			&score.CreatedAt,
			&score.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan verification score: %w", err)
		}

		if expiresAt.Valid {
			score.ExpiresAt = &expiresAt.Time
		}

		scores = append(scores, &score)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating verification scores: %w", err)
	}

	return scores, nil
}

// UpdateVerificationScore updates an existing verification score
func (d *Database) UpdateVerificationScore(score *VerificationScore) error {
	query := `
		UPDATE verification_scores SET
			score = ?, score_type = ?, scoring_method = ?, category = ?,
			code_correctness_score = ?, code_quality_score = ?, code_speed_score = ?,
			error_handling_score = ?, context_understanding_score = ?,
			evidence = ?, benchmark_version = ?, scored_by = ?, confidence_level = ?,
			scored_at = ?, expires_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	expiresAt := sql.NullTime{}
	if score.ExpiresAt != nil {
		expiresAt.Valid = true
		expiresAt.Time = *score.ExpiresAt
	}

	_, err := d.conn.Exec(query,
		score.Score,
		score.ScoreType,
		score.ScoringMethod,
		score.Category,
		score.CodeCorrectnessScore,
		score.CodeQualityScore,
		score.CodeSpeedScore,
		score.ErrorHandlingScore,
		score.ContextUnderstandingScore,
		score.Evidence,
		score.BenchmarkVersion,
		score.ScoredBy,
		score.ConfidenceLevel,
		score.ScoredAt,
		expiresAt,
		score.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update verification score: %w", err)
	}

	return nil
}

// DeleteVerificationScore removes a verification score
func (d *Database) DeleteVerificationScore(id int64) error {
	query := `DELETE FROM verification_scores WHERE id = ?`

	result, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete verification score: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("verification score not found: %d", id)
	}

	return nil
}

// GetModelCodingCapabilityScore gets the latest coding capability score for a model
func (d *Database) GetModelCodingCapabilityScore(modelID int64) (*VerificationScore, error) {
	return d.GetLatestVerificationScore(modelID, "coding_capability")
}

// CalculateCodingCapabilityScore calculates a coding capability score from benchmark results
func CalculateCodingCapabilityScore(benchmarks map[string]int) (int, string) {
	// Weights for different benchmark categories
	weights := map[string]float64{
		"code_correctness": 0.40,
		"code_quality":     0.30,
		"code_speed":       0.20,
		"error_handling":   0.10,
	}

	totalScore := 0.0
	evidence := make(map[string]int)

	for category, weight := range weights {
		if score, ok := benchmarks[category]; ok {
			totalScore += float64(score) * weight
			evidence[category] = score
		}
	}

	// Convert to integer 0-100
	finalScore := int(totalScore)

	// Determine category
	var category string
	switch {
	case finalScore >= 80:
		category = "fully_coding_capable"
	case finalScore >= 60:
		category = "coding_with_tools"
	case finalScore >= 40:
		category = "chat_with_tooling"
	default:
		category = "chat_only"
	}

	return finalScore, category
}

// GetTopScoringModels returns top N models by score type
func (d *Database) GetTopScoringModels(scoreType string, limit int) ([]*VerificationScore, error) {
	query := `
		SELECT id, model_id, verification_result_id, score, score_type, scoring_method, category,
			code_correctness_score, code_quality_score, code_speed_score, error_handling_score,
			context_understanding_score, evidence, benchmark_version, scored_by, confidence_level,
			scored_at, expires_at, created_at, updated_at
		FROM verification_scores
		WHERE score_type = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY score DESC, scored_at DESC
		LIMIT ?
	`

	rows, err := d.conn.Query(query, scoreType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top scoring models: %w", err)
	}
	defer rows.Close()

	var scores []*VerificationScore
	for rows.Next() {
		var score VerificationScore
		var expiresAt sql.NullTime

		err := rows.Scan(
			&score.ID,
			&score.ModelID,
			&score.VerificationResultID,
			&score.Score,
			&score.ScoreType,
			&score.ScoringMethod,
			&score.Category,
			&score.CodeCorrectnessScore,
			&score.CodeQualityScore,
			&score.CodeSpeedScore,
			&score.ErrorHandlingScore,
			&score.ContextUnderstandingScore,
			&score.Evidence,
			&score.BenchmarkVersion,
			&score.ScoredBy,
			&score.ConfidenceLevel,
			&score.ScoredAt,
			&expiresAt,
			&score.CreatedAt,
			&score.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan verification score: %w", err)
		}

		if expiresAt.Valid {
			score.ExpiresAt = &expiresAt.Time
		}

		scores = append(scores, &score)
	}

	return scores, nil
}
