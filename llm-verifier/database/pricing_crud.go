package database

import (
	"database/sql"
	"fmt"
	"time"
)

// ==================== Pricing CRUD Operations ====================

// CreatePricing creates a new pricing record
func (d *Database) CreatePricing(pricing *Pricing) error {
	query := `
		INSERT INTO pricing (
			model_id, input_token_cost, output_token_cost, cached_input_token_cost,
			storage_cost, request_cost, currency, pricing_model, effective_from,
			effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	result, err := d.conn.Exec(query,
		pricing.ModelID,
		pricing.InputTokenCost,
		pricing.OutputTokenCost,
		pricing.CachedInputTokenCost,
		pricing.StorageCost,
		pricing.RequestCost,
		pricing.Currency,
		pricing.PricingModel,
		pricing.EffectiveFrom,
		pricing.EffectiveTo,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create pricing: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	
	pricing.ID = id
	return nil
}

// GetPricing retrieves a pricing record by ID
func (d *Database) GetPricing(id int64) (*Pricing, error) {
	query := `
		SELECT id, model_id, input_token_cost, output_token_cost, cached_input_token_cost,
			storage_cost, request_cost, currency, pricing_model, effective_from,
			effective_to, created_at, updated_at
		FROM pricing WHERE id = ?
	`
	
	var pricing Pricing
	var effectiveFrom, effectiveTo sql.NullTime
	
	err := d.conn.QueryRow(query, id).Scan(
		&pricing.ID,
		&pricing.ModelID,
		&pricing.InputTokenCost,
		&pricing.OutputTokenCost,
		&pricing.CachedInputTokenCost,
		&pricing.StorageCost,
		&pricing.RequestCost,
		&pricing.Currency,
		&pricing.PricingModel,
		&effectiveFrom,
		&effectiveTo,
		&pricing.CreatedAt,
		&pricing.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pricing not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}
	
	pricing.EffectiveFrom = scanNullableTime(effectiveFrom)
	pricing.EffectiveTo = scanNullableTime(effectiveTo)
	
	return &pricing, nil
}

// GetLatestPricing gets the latest pricing for a model
func (d *Database) GetLatestPricing(modelID int64) (*Pricing, error) {
	query := `
		SELECT id, model_id, input_token_cost, output_token_cost, cached_input_token_cost,
			storage_cost, request_cost, currency, pricing_model, effective_from,
			effective_to, created_at, updated_at
		FROM pricing
		WHERE model_id = ? AND (effective_to IS NULL OR effective_to >= ?)
		ORDER BY created_at DESC
		LIMIT 1
	`
	
	var pricing Pricing
	var effectiveFrom, effectiveTo sql.NullTime
	currentTime := time.Now()
	
	err := d.conn.QueryRow(query, modelID, currentTime).Scan(
		&pricing.ID,
		&pricing.ModelID,
		&pricing.InputTokenCost,
		&pricing.OutputTokenCost,
		&pricing.CachedInputTokenCost,
		&pricing.StorageCost,
		&pricing.RequestCost,
		&pricing.Currency,
		&pricing.PricingModel,
		&effectiveFrom,
		&effectiveTo,
		&pricing.CreatedAt,
		&pricing.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no valid pricing found for model: %d", modelID)
		}
		return nil, fmt.Errorf("failed to get latest pricing: %w", err)
	}
	
	pricing.EffectiveFrom = scanNullableTime(effectiveFrom)
	pricing.EffectiveTo = scanNullableTime(effectiveTo)
	
	return &pricing, nil
}

// ListPricing gets all pricing records for a model
func (d *Database) ListPricing(modelID int64) ([]*Pricing, error) {
	query := `
		SELECT id, model_id, input_token_cost, output_token_cost, cached_input_token_cost,
			storage_cost, request_cost, currency, pricing_model, effective_from,
			effective_to, created_at, updated_at
		FROM pricing
		WHERE model_id = ?
		ORDER BY created_at DESC
	`
	
	rows, err := d.conn.Query(query, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pricing: %w", err)
	}
	defer rows.Close()
	
	var pricingList []*Pricing
	for rows.Next() {
		var pricing Pricing
		var effectiveFrom, effectiveTo sql.NullTime
		
		err := rows.Scan(
			&pricing.ID,
			&pricing.ModelID,
			&pricing.InputTokenCost,
			&pricing.OutputTokenCost,
			&pricing.CachedInputTokenCost,
			&pricing.StorageCost,
			&pricing.RequestCost,
			&pricing.Currency,
			&pricing.PricingModel,
			&effectiveFrom,
			&effectiveTo,
			&pricing.CreatedAt,
			&pricing.UpdatedAt,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan pricing: %w", err)
		}
		
		pricing.EffectiveFrom = scanNullableTime(effectiveFrom)
		pricing.EffectiveTo = scanNullableTime(effectiveTo)
		
		pricingList = append(pricingList, &pricing)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pricing: %w", err)
	}
	
	return pricingList, nil
}

// ==================== Limits CRUD Operations ====================

// CreateLimit creates a new limit record
func (d *Database) CreateLimit(limit *Limit) error {
	query := `
		INSERT INTO limits (
			model_id, limit_type, limit_value, current_usage, reset_period,
			reset_time, is_hard_limit
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	result, err := d.conn.Exec(query,
		limit.ModelID,
		limit.LimitType,
		limit.LimitValue,
		limit.CurrentUsage,
		limit.ResetPeriod,
		limit.ResetTime,
		limit.IsHardLimit,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create limit: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	
	limit.ID = id
	return nil
}

// GetLimit retrieves a limit record by ID
func (d *Database) GetLimit(id int64) (*Limit, error) {
	query := `
		SELECT id, model_id, limit_type, limit_value, current_usage, reset_period,
			reset_time, is_hard_limit, created_at, updated_at
		FROM limits WHERE id = ?
	`
	
	var limit Limit
	var resetTime sql.NullTime
	
	err := d.conn.QueryRow(query, id).Scan(
		&limit.ID,
		&limit.ModelID,
		&limit.LimitType,
		&limit.LimitValue,
		&limit.CurrentUsage,
		&limit.ResetPeriod,
		&resetTime,
		&limit.IsHardLimit,
		&limit.CreatedAt,
		&limit.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("limit not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get limit: %w", err)
	}
	
	limit.ResetTime = scanNullableTime(resetTime)
	
	return &limit, nil
}

// GetLimitsForModel gets all limits for a specific model
func (d *Database) GetLimitsForModel(modelID int64) ([]*Limit, error) {
	query := `
		SELECT id, model_id, limit_type, limit_value, current_usage, reset_period,
			reset_time, is_hard_limit, created_at, updated_at
		FROM limits
		WHERE model_id = ?
		ORDER BY limit_type
	`
	
	rows, err := d.conn.Query(query, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get limits for model: %w", err)
	}
	defer rows.Close()
	
	var limits []*Limit
	for rows.Next() {
		var limit Limit
		var resetTime sql.NullTime
		
		err := rows.Scan(
			&limit.ID,
			&limit.ModelID,
			&limit.LimitType,
			&limit.LimitValue,
			&limit.CurrentUsage,
			&limit.ResetPeriod,
			&resetTime,
			&limit.IsHardLimit,
			&limit.CreatedAt,
			&limit.UpdatedAt,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan limit: %w", err)
		}
		
		limit.ResetTime = scanNullableTime(resetTime)
		limits = append(limits, &limit)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating limits: %w", err)
	}
	
	return limits, nil
}

// UpdateLimitCurrentUsage updates the current usage for a limit
func (d *Database) UpdateLimitCurrentUsage(limitID int64, currentUsage int) error {
	query := `UPDATE limits SET current_usage = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	
	_, err := d.conn.Exec(query, currentUsage, limitID)
	if err != nil {
		return fmt.Errorf("failed to update limit current usage: %w", err)
	}
	
	return nil
}

// ResetLimitUsage resets usage for limits that have expired
func (d *Database) ResetLimitUsage() error {
	query := `
		UPDATE limits 
		SET current_usage = 0, updated_at = CURRENT_TIMESTAMP 
		WHERE reset_time IS NOT NULL AND reset_time <= CURRENT_TIMESTAMP
	`
	
	_, err := d.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to reset limit usage: %w", err)
	}
	
	return nil
}

// DeleteLimit deletes a limit by ID
func (d *Database) DeleteLimit(id int64) error {
	query := `DELETE FROM limits WHERE id = ?`
	
	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete limit: %w", err)
	}
	
	return nil
}