package database

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          func(*sql.Tx) error
	Down        func(*sql.Tx) error
}

// MigrationManager handles database schema migrations
type MigrationManager struct {
	db         *Database
	migrations []Migration
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *Database) *MigrationManager {
	return &MigrationManager{
		db:         db,
		migrations: []Migration{},
	}
}

// AddMigration adds a new migration to the manager
func (mm *MigrationManager) AddMigration(migration Migration) {
	mm.migrations = append(mm.migrations, migration)
	// Sort migrations by version
	sort.Slice(mm.migrations, func(i, j int) bool {
		return mm.migrations[i].Version < mm.migrations[j].Version
	})
}

// InitializeMigrationTable creates the migrations table if it doesn't exist
func (mm *MigrationManager) InitializeMigrationTable() error {
	return mm.db.WithTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				description TEXT NOT NULL,
				applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		return err
	})
}

// GetCurrentVersion returns the current database schema version
func (mm *MigrationManager) GetCurrentVersion() (int, error) {
	var version int
	err := mm.db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}
	return version, nil
}

// MigrateUp applies all pending migrations
func (mm *MigrationManager) MigrateUp() error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return err
	}

	log.Printf("Current database version: %d", currentVersion)

	for _, migration := range mm.migrations {
		if migration.Version > currentVersion {
			log.Printf("Applying migration %d: %s", migration.Version, migration.Description)

			err := mm.db.WithTransaction(func(tx *sql.Tx) error {
				// Apply the migration
				if err := migration.Up(tx); err != nil {
					return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
				}

				// Record the migration
				_, err := tx.Exec(
					"INSERT INTO schema_migrations (version, description) VALUES (?, ?)",
					migration.Version, migration.Description,
				)
				if err != nil {
					return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
				}

				return nil
			})

			if err != nil {
				return err
			}

			log.Printf("Successfully applied migration %d", migration.Version)
		}
	}

	log.Println("All migrations applied successfully")
	return nil
}

// MigrateDown rolls back the last migration
func (mm *MigrationManager) MigrateDown() error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return err
	}

	if currentVersion == 0 {
		log.Println("No migrations to roll back")
		return nil
	}

	// Find the migration to roll back
	var migrationToRollback *Migration
	for i := len(mm.migrations) - 1; i >= 0; i-- {
		if mm.migrations[i].Version == currentVersion {
			migrationToRollback = &mm.migrations[i]
			break
		}
	}

	if migrationToRollback == nil {
		return fmt.Errorf("migration %d not found", currentVersion)
	}

	log.Printf("Rolling back migration %d: %s", migrationToRollback.Version, migrationToRollback.Description)

	err = mm.db.WithTransaction(func(tx *sql.Tx) error {
		// Roll back the migration
		if err := migrationToRollback.Down(tx); err != nil {
			return fmt.Errorf("failed to roll back migration %d: %w", migrationToRollback.Version, err)
		}

		// Remove the migration record
		_, err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", migrationToRollback.Version)
		if err != nil {
			return fmt.Errorf("failed to remove migration record %d: %w", migrationToRollback.Version, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("Successfully rolled back migration %d", migrationToRollback.Version)
	return nil
}

// GetMigrationStatus returns the status of all migrations
func (mm *MigrationManager) GetMigrationStatus() ([]MigrationStatus, error) {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return nil, err
	}

	var status []MigrationStatus
	for _, migration := range mm.migrations {
		status = append(status, MigrationStatus{
			Version:     migration.Version,
			Description: migration.Description,
			Applied:     migration.Version <= currentVersion,
		})
	}

	return status, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version     int    `json:"version"`
	Description string `json:"description"`
	Applied     bool   `json:"applied"`
}

// SetupDefaultMigrations adds the default migrations for the LLM Verifier
func (mm *MigrationManager) SetupDefaultMigrations() {
	// Migration 1: Initial schema
	mm.AddMigration(Migration{
		Version:     1,
		Description: "Initial schema with providers, models, and verification_results tables",
		Up: func(tx *sql.Tx) error {
			// This would contain the initial schema creation
			// For now, we'll assume the schema is already created by initializeSchema
			return nil
		},
		Down: func(tx *sql.Tx) error {
			// Drop all tables in reverse order
			tables := []string{
				"verification_results", "models", "providers",
				"pricing", "limits", "issues", "events", "schedules",
				"config_exports", "logs",
			}
			for _, table := range tables {
				if _, err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
					return err
				}
			}
			return nil
		},
	})

	// Migration 2: Add performance indexes
	mm.AddMigration(Migration{
		Version:     2,
		Description: "Add performance indexes for common queries",
		Up: func(tx *sql.Tx) error {
			indexes := []string{
				"CREATE INDEX IF NOT EXISTS idx_models_provider_id ON models(provider_id)",
				"CREATE INDEX IF NOT EXISTS idx_models_verification_status ON models(verification_status)",
				"CREATE INDEX IF NOT EXISTS idx_models_overall_score ON models(overall_score)",
				"CREATE INDEX IF NOT EXISTS idx_verification_results_model_id ON verification_results(model_id)",
				"CREATE INDEX IF NOT EXISTS idx_verification_results_status ON verification_results(status)",
				"CREATE INDEX IF NOT EXISTS idx_verification_results_created_at ON verification_results(created_at)",
				"CREATE INDEX IF NOT EXISTS idx_issues_model_id ON issues(model_id)",
				"CREATE INDEX IF NOT EXISTS idx_issues_resolved ON issues(resolved_at IS NULL)",
				"CREATE INDEX IF NOT EXISTS idx_pricing_model_id ON pricing(model_id)",
				"CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at)",
				"CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp)",
			}

			for _, index := range indexes {
				if _, err := tx.Exec(index); err != nil {
					return fmt.Errorf("failed to create index: %s, error: %w", index, err)
				}
			}
			return nil
		},
		Down: func(tx *sql.Tx) error {
			indexes := []string{
				"DROP INDEX IF EXISTS idx_models_provider_id",
				"DROP INDEX IF EXISTS idx_models_verification_status",
				"DROP INDEX IF EXISTS idx_models_overall_score",
				"DROP INDEX IF EXISTS idx_verification_results_model_id",
				"DROP INDEX IF EXISTS idx_verification_results_status",
				"DROP INDEX IF EXISTS idx_verification_results_created_at",
				"DROP INDEX IF EXISTS idx_issues_model_id",
				"DROP INDEX IF EXISTS idx_issues_resolved",
				"DROP INDEX IF EXISTS idx_pricing_model_id",
				"DROP INDEX IF EXISTS idx_events_created_at",
				"DROP INDEX IF EXISTS idx_logs_timestamp",
			}

			for _, index := range indexes {
				if _, err := tx.Exec(index); err != nil {
					return fmt.Errorf("failed to drop index: %s, error: %w", index, err)
				}
			}
			return nil
		},
	})

	// Migration 3: Add composite indexes for complex queries
	mm.AddMigration(Migration{
		Version:     3,
		Description: "Add composite indexes for complex query optimization",
		Up: func(tx *sql.Tx) error {
			compositeIndexes := []string{
				"CREATE INDEX IF NOT EXISTS idx_models_provider_status ON models(provider_id, verification_status)",
				"CREATE INDEX IF NOT EXISTS idx_models_score_status ON models(overall_score DESC, verification_status)",
				"CREATE INDEX IF NOT EXISTS idx_verification_results_model_status ON verification_results(model_id, status)",
				"CREATE INDEX IF NOT EXISTS idx_verification_results_model_created ON verification_results(model_id, created_at DESC)",
				"CREATE INDEX IF NOT EXISTS idx_issues_model_resolved ON issues(model_id, resolved_at IS NULL)",

				"CREATE INDEX IF NOT EXISTS idx_events_type_created ON events(event_type, created_at DESC)",
			}

			for _, index := range compositeIndexes {
				if _, err := tx.Exec(index); err != nil {
					return fmt.Errorf("failed to create composite index: %s, error: %w", index, err)
				}
			}
			return nil
		},
		Down: func(tx *sql.Tx) error {
			compositeIndexes := []string{
				"DROP INDEX IF EXISTS idx_models_provider_status",
				"DROP INDEX IF EXISTS idx_models_score_status",
				"DROP INDEX IF EXISTS idx_verification_results_model_status",
				"DROP INDEX IF EXISTS idx_verification_results_model_created",
				"DROP INDEX IF EXISTS idx_issues_model_resolved",

				"DROP INDEX IF EXISTS idx_events_type_created",
			}

			for _, index := range compositeIndexes {
				if _, err := tx.Exec(index); err != nil {
					return fmt.Errorf("failed to drop composite index: %s, error: %w", index, err)
				}
			}
			return nil
		},
	})

	// Migration 4: Add notifications table
	mm.AddMigration(Migration{
		Version:     4,
		Description: "Add notifications table for system notifications",
		Up: func(tx *sql.Tx) error {
			// Create notifications table
			if _, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS notifications (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					type TEXT NOT NULL, -- verification_completed, verification_failed, score_changed, etc.
					channel TEXT NOT NULL, -- slack, email, telegram, matrix, whatsapp
					priority TEXT NOT NULL DEFAULT 'normal', -- low, normal, high, critical
					title TEXT NOT NULL,
					message TEXT NOT NULL,
					data TEXT, -- JSON additional data
					recipient TEXT, -- recipient identifier (email, chat_id, etc.)
					sent BOOLEAN DEFAULT 0,
					error TEXT,
					retry_count INTEGER DEFAULT 0,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					sent_at TIMESTAMP,
					updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				)
			`); err != nil {
				return fmt.Errorf("failed to create notifications table: %w", err)
			}

			// Create indexes for notifications table
			indexes := []string{
				"CREATE INDEX IF NOT EXISTS idx_notifications_type_created ON notifications(type, created_at DESC)",
				"CREATE INDEX IF NOT EXISTS idx_notifications_channel_sent ON notifications(channel, sent)",
				"CREATE INDEX IF NOT EXISTS idx_notifications_sent_at ON notifications(sent_at)",
				"CREATE INDEX IF NOT EXISTS idx_notifications_priority_created ON notifications(priority, created_at DESC)",
			}

			for _, index := range indexes {
				if _, err := tx.Exec(index); err != nil {
					return fmt.Errorf("failed to create notification index: %s, error: %w", index, err)
				}
			}

			// Create trigger for updating notifications timestamp
			if _, err := tx.Exec(`
				CREATE TRIGGER IF NOT EXISTS update_notifications_timestamp
				AFTER UPDATE ON notifications
				BEGIN
					UPDATE notifications SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
				END
			`); err != nil {
				return fmt.Errorf("failed to create notifications timestamp trigger: %w", err)
			}

			return nil
		},
		Down: func(tx *sql.Tx) error {
			// Drop trigger
			if _, err := tx.Exec("DROP TRIGGER IF EXISTS update_notifications_timestamp"); err != nil {
				return fmt.Errorf("failed to drop notifications timestamp trigger: %w", err)
			}

			// Drop indexes
			indexes := []string{
				"DROP INDEX IF EXISTS idx_notifications_type_created",
				"DROP INDEX IF EXISTS idx_notifications_channel_sent",
				"DROP INDEX IF EXISTS idx_notifications_sent_at",
				"DROP INDEX IF EXISTS idx_notifications_priority_created",
			}

			for _, index := range indexes {
				if _, err := tx.Exec(index); err != nil {
					return fmt.Errorf("failed to drop notification index: %s, error: %w", index, err)
				}
			}

			// Drop table
			if _, err := tx.Exec("DROP TABLE IF EXISTS notifications"); err != nil {
				return fmt.Errorf("failed to drop notifications table: %w", err)
			}

			return nil
		},
	})
}
