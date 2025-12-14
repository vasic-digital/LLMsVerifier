package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ==================== Schedule CRUD Operations ====================

// CreateSchedule creates a new schedule
func (d *Database) CreateSchedule(schedule *Schedule) error {
	query := `
		INSERT INTO schedules (
			name, description, schedule_type, cron_expression, interval_seconds,
			target_type, target_id, is_active, last_run, next_run, run_count, max_runs, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var lastRun, nextRun sql.NullTime
	if schedule.LastRun != nil {
		lastRun.Valid = true
		lastRun.Time = *schedule.LastRun
	}
	if schedule.NextRun != nil {
		nextRun.Valid = true
		nextRun.Time = *schedule.NextRun
	}

	result, err := d.conn.Exec(query,
		schedule.Name,
		schedule.Description,
		schedule.ScheduleType,
		schedule.CronExpression,
		schedule.IntervalSeconds,
		schedule.TargetType,
		schedule.TargetID,
		schedule.IsActive,
		lastRun,
		nextRun,
		schedule.RunCount,
		schedule.MaxRuns,
		schedule.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	schedule.ID = id
	return nil
}

// GetSchedule retrieves a schedule by ID
func (d *Database) GetSchedule(id int64) (*Schedule, error) {
	query := `
		SELECT id, name, description, schedule_type, cron_expression, interval_seconds,
			target_type, target_id, is_active, last_run, next_run, run_count, max_runs,
			created_at, updated_at, created_by
		FROM schedules WHERE id = ?
	`

	var schedule Schedule
	var description, cronExpression, createdBy sql.NullString
	var intervalSeconds, maxRuns sql.NullInt32
	var lastRun, nextRun sql.NullTime

	err := d.conn.QueryRow(query, id).Scan(
		&schedule.ID,
		&schedule.Name,
		&description,
		&schedule.ScheduleType,
		&cronExpression,
		&intervalSeconds,
		&schedule.TargetType,
		&schedule.TargetID,
		&schedule.IsActive,
		&lastRun,
		&nextRun,
		&schedule.RunCount,
		&maxRuns,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
		&createdBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("schedule not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	// Handle nullable fields
	if description.Valid {
		schedule.Description = &description.String
	}
	if cronExpression.Valid {
		schedule.CronExpression = &cronExpression.String
	}
	if intervalSeconds.Valid {
		interval := int(intervalSeconds.Int32)
		schedule.IntervalSeconds = &interval
	}
	if maxRuns.Valid {
		max := int(maxRuns.Int32)
		schedule.MaxRuns = &max
	}
	if lastRun.Valid {
		schedule.LastRun = &lastRun.Time
	}
	if nextRun.Valid {
		schedule.NextRun = &nextRun.Time
	}
	if createdBy.Valid {
		schedule.CreatedBy = &createdBy.String
	}

	return &schedule, nil
}

// UpdateSchedule updates an existing schedule
func (d *Database) UpdateSchedule(schedule *Schedule) error {
	query := `
		UPDATE schedules SET
			name = ?, description = ?, schedule_type = ?, cron_expression = ?,
			interval_seconds = ?, target_type = ?, target_id = ?, is_active = ?,
			last_run = ?, next_run = ?, run_count = ?, max_runs = ?, created_by = ?
		WHERE id = ?
	`

	var lastRun, nextRun sql.NullTime
	if schedule.LastRun != nil {
		lastRun.Valid = true
		lastRun.Time = *schedule.LastRun
	}
	if schedule.NextRun != nil {
		nextRun.Valid = true
		nextRun.Time = *schedule.NextRun
	}

	_, err := d.conn.Exec(query,
		schedule.Name,
		schedule.Description,
		schedule.ScheduleType,
		schedule.CronExpression,
		schedule.IntervalSeconds,
		schedule.TargetType,
		schedule.TargetID,
		schedule.IsActive,
		lastRun,
		nextRun,
		schedule.RunCount,
		schedule.MaxRuns,
		schedule.CreatedBy,
		schedule.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	return nil
}

// DeleteSchedule deletes a schedule by ID
func (d *Database) DeleteSchedule(id int64) error {
	query := `DELETE FROM schedules WHERE id = ?`

	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}

// ListSchedules retrieves schedules with optional filtering
func (d *Database) ListSchedules(filters map[string]interface{}) ([]*Schedule, error) {
	query := `
		SELECT id, name, description, schedule_type, cron_expression, interval_seconds,
			target_type, target_id, is_active, last_run, next_run, run_count, max_runs,
			created_at, updated_at, created_by
		FROM schedules
	`

	var conditions []string
	var args []interface{}

	// Add conditions based on filters
	if scheduleType, ok := filters["schedule_type"]; ok {
		conditions = append(conditions, "schedule_type = ?")
		args = append(args, scheduleType)
	}

	if targetType, ok := filters["target_type"]; ok {
		conditions = append(conditions, "target_type = ?")
		args = append(args, targetType)
	}

	if targetID, ok := filters["target_id"]; ok {
		conditions = append(conditions, "target_id = ?")
		args = append(args, targetID)
	}

	if isActive, ok := filters["is_active"]; ok {
		conditions = append(conditions, "is_active = ?")
		args = append(args, isActive)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"]; ok {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		var schedule Schedule
		var description, cronExpression, createdBy sql.NullString
		var intervalSeconds, maxRuns sql.NullInt32
		var lastRun, nextRun sql.NullTime

		err := rows.Scan(
			&schedule.ID,
			&schedule.Name,
			&description,
			&schedule.ScheduleType,
			&cronExpression,
			&intervalSeconds,
			&schedule.TargetType,
			&schedule.TargetID,
			&schedule.IsActive,
			&lastRun,
			&nextRun,
			&schedule.RunCount,
			&maxRuns,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
			&createdBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		// Handle nullable fields
		if description.Valid {
			schedule.Description = &description.String
		}
		if cronExpression.Valid {
			schedule.CronExpression = &cronExpression.String
		}
		if intervalSeconds.Valid {
			interval := int(intervalSeconds.Int32)
			schedule.IntervalSeconds = &interval
		}
		if maxRuns.Valid {
			max := int(maxRuns.Int32)
			schedule.MaxRuns = &max
		}
		if lastRun.Valid {
			schedule.LastRun = &lastRun.Time
		}
		if nextRun.Valid {
			schedule.NextRun = &nextRun.Time
		}
		if createdBy.Valid {
			schedule.CreatedBy = &createdBy.String
		}

		schedules = append(schedules, &schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedules: %w", err)
	}

	return schedules, nil
}

// GetActiveSchedules gets all active schedules that are due to run
func (d *Database) GetActiveSchedules() ([]*Schedule, error) {
	query := `
		SELECT id, name, description, schedule_type, cron_expression, interval_seconds,
			target_type, target_id, is_active, last_run, next_run, run_count, max_runs,
			created_at, updated_at, created_by
		FROM schedules
		WHERE is_active = 1 AND (next_run IS NULL OR next_run <= ?)
		ORDER BY next_run ASC
	`

	currentTime := time.Now()
	rows, err := d.conn.Query(query, currentTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get active schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		var schedule Schedule
		var description, cronExpression, createdBy sql.NullString
		var intervalSeconds, maxRuns sql.NullInt32
		var lastRun, nextRun sql.NullTime

		err := rows.Scan(
			&schedule.ID,
			&schedule.Name,
			&description,
			&schedule.ScheduleType,
			&cronExpression,
			&intervalSeconds,
			&schedule.TargetType,
			&schedule.TargetID,
			&schedule.IsActive,
			&lastRun,
			&nextRun,
			&schedule.RunCount,
			&maxRuns,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
			&createdBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan active schedule: %w", err)
		}

		// Handle nullable fields
		if description.Valid {
			schedule.Description = &description.String
		}
		if cronExpression.Valid {
			schedule.CronExpression = &cronExpression.String
		}
		if intervalSeconds.Valid {
			interval := int(intervalSeconds.Int32)
			schedule.IntervalSeconds = &interval
		}
		if maxRuns.Valid {
			max := int(maxRuns.Int32)
			schedule.MaxRuns = &max
		}
		if lastRun.Valid {
			schedule.LastRun = &lastRun.Time
		}
		if nextRun.Valid {
			schedule.NextRun = &nextRun.Time
		}
		if createdBy.Valid {
			schedule.CreatedBy = &createdBy.String
		}

		schedules = append(schedules, &schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active schedules: %w", err)
	}

	return schedules, nil
}

// UpdateScheduleRunInfo updates the run information for a schedule
func (d *Database) UpdateScheduleRunInfo(scheduleID int64, lastRun time.Time, nextRun *time.Time, runCount int) error {
	query := `
		UPDATE schedules
		SET last_run = ?, next_run = ?, run_count = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	var nextRunNull sql.NullTime
	if nextRun != nil {
		nextRunNull.Valid = true
		nextRunNull.Time = *nextRun
	}

	_, err := d.conn.Exec(query, lastRun, nextRunNull, runCount, scheduleID)
	if err != nil {
		return fmt.Errorf("failed to update schedule run info: %w", err)
	}

	return nil
}
