package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// ==================== ConfigExport CRUD Operations ====================

// CreateConfigExport creates a new config export
func (d *Database) CreateConfigExport(configExport *ConfigExport) error {
	query := `
		INSERT INTO config_exports (
			export_type, name, description, config_data, target_models,
			target_providers, filters, is_verified, verification_notes, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.conn.Exec(query,
		configExport.ExportType,
		configExport.Name,
		configExport.Description,
		configExport.ConfigData,
		configExport.TargetModels,
		configExport.TargetProviders,
		configExport.Filters,
		configExport.IsVerified,
		configExport.VerificationNotes,
		configExport.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create config export: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	configExport.ID = id
	return nil
}

// GetConfigExport retrieves a config export by ID
func (d *Database) GetConfigExport(id int64) (*ConfigExport, error) {
	query := `
		SELECT id, export_type, name, description, config_data, target_models,
			target_providers, filters, is_verified, verification_notes, created_by,
			created_at, updated_at, download_count
		FROM config_exports WHERE id = ?
	`

	var configExport ConfigExport
	var targetModels, targetProviders, filters, verificationNotes, createdBy sql.NullString

	err := d.conn.QueryRow(query, id).Scan(
		&configExport.ID,
		&configExport.ExportType,
		&configExport.Name,
		&configExport.Description,
		&configExport.ConfigData,
		&targetModels,
		&targetProviders,
		&filters,
		&configExport.IsVerified,
		&verificationNotes,
		&createdBy,
		&configExport.CreatedAt,
		&configExport.UpdatedAt,
		&configExport.DownloadCount,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("config export not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get config export: %w", err)
	}

	// Handle nullable fields
	if targetModels.Valid {
		configExport.TargetModels = &targetModels.String
	}
	if targetProviders.Valid {
		configExport.TargetProviders = &targetProviders.String
	}
	if filters.Valid {
		configExport.Filters = &filters.String
	}
	if verificationNotes.Valid {
		configExport.VerificationNotes = &verificationNotes.String
	}
	if createdBy.Valid {
		configExport.CreatedBy = &createdBy.String
	}

	return &configExport, nil
}

// UpdateConfigExport updates an existing config export
func (d *Database) UpdateConfigExport(configExport *ConfigExport) error {
	query := `
		UPDATE config_exports SET
			export_type = ?, name = ?, description = ?, config_data = ?,
			target_models = ?, target_providers = ?, filters = ?, is_verified = ?,
			verification_notes = ?, created_by = ?, download_count = ?
		WHERE id = ?
	`

	_, err := d.conn.Exec(query,
		configExport.ExportType,
		configExport.Name,
		configExport.Description,
		configExport.ConfigData,
		configExport.TargetModels,
		configExport.TargetProviders,
		configExport.Filters,
		configExport.IsVerified,
		configExport.VerificationNotes,
		configExport.CreatedBy,
		configExport.DownloadCount,
		configExport.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update config export: %w", err)
	}

	return nil
}

// DeleteConfigExport deletes a config export by ID
func (d *Database) DeleteConfigExport(id int64) error {
	query := `DELETE FROM config_exports WHERE id = ?`

	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete config export: %w", err)
	}

	return nil
}

// ListConfigExports retrieves config exports with optional filtering
func (d *Database) ListConfigExports(filters map[string]any) ([]*ConfigExport, error) {
	query := `
		SELECT id, export_type, name, description, config_data, target_models,
			target_providers, filters, is_verified, verification_notes, created_by,
			created_at, updated_at, download_count
		FROM config_exports
	`

	var conditions []string
	var args []any

	// Add conditions based on filters
	if exportType, ok := filters["export_type"]; ok {
		conditions = append(conditions, "export_type = ?")
		args = append(args, exportType)
	}

	if isVerified, ok := filters["is_verified"]; ok {
		conditions = append(conditions, "is_verified = ?")
		args = append(args, isVerified)
	}

	if createdBy, ok := filters["created_by"]; ok {
		conditions = append(conditions, "created_by = ?")
		args = append(args, createdBy)
	}

	if search, ok := filters["search"]; ok {
		conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
		searchPattern := fmt.Sprintf("%%%s%%", search)
		args = append(args, searchPattern, searchPattern)
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
		return nil, fmt.Errorf("failed to list config exports: %w", err)
	}
	defer rows.Close()

	var configExports []*ConfigExport
	for rows.Next() {
		var configExport ConfigExport
		var targetModels, targetProviders, filters, verificationNotes, createdBy sql.NullString

		err := rows.Scan(
			&configExport.ID,
			&configExport.ExportType,
			&configExport.Name,
			&configExport.Description,
			&configExport.ConfigData,
			&targetModels,
			&targetProviders,
			&filters,
			&configExport.IsVerified,
			&verificationNotes,
			&createdBy,
			&configExport.CreatedAt,
			&configExport.UpdatedAt,
			&configExport.DownloadCount,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan config export: %w", err)
		}

		// Handle nullable fields
		if targetModels.Valid {
			configExport.TargetModels = &targetModels.String
		}
		if targetProviders.Valid {
			configExport.TargetProviders = &targetProviders.String
		}
		if filters.Valid {
			configExport.Filters = &filters.String
		}
		if verificationNotes.Valid {
			configExport.VerificationNotes = &verificationNotes.String
		}
		if createdBy.Valid {
			configExport.CreatedBy = &createdBy.String
		}

		configExports = append(configExports, &configExport)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating config exports: %w", err)
	}

	return configExports, nil
}

// IncrementDownloadCount increments the download count for a config export
func (d *Database) IncrementDownloadCount(id int64) error {
	query := `UPDATE config_exports SET download_count = download_count + 1 WHERE id = ?`

	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to increment download count: %w", err)
	}

	return nil
}

// GetConfigExportsByType gets config exports filtered by type
func (d *Database) GetConfigExportsByType(exportType string) ([]*ConfigExport, error) {
	filters := map[string]any{
		"export_type": exportType,
	}

	return d.ListConfigExports(filters)
}

// GetVerifiedConfigExports gets all verified config exports
func (d *Database) GetVerifiedConfigExports() ([]*ConfigExport, error) {
	filters := map[string]any{
		"is_verified": true,
	}

	return d.ListConfigExports(filters)
}
