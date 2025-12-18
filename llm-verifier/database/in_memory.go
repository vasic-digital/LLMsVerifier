package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// InMemoryDatabase provides an in-memory database for testing
type InMemoryDatabase struct {
	conn *sql.DB
}

// NewInMemoryDatabase creates a new in-memory database
func NewInMemoryDatabase() *InMemoryDatabase {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("Failed to create in-memory database: %v", err)
	}

	// Initialize schema
	initSchema(conn)

	return &InMemoryDatabase{conn: conn}
}

// initSchema creates tables in the in-memory database
func initSchema(conn *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		schedule_type TEXT NOT NULL,
		cron_expression TEXT,
		interval_seconds INTEGER,
		target_type TEXT,
		target_id TEXT,
		is_active BOOLEAN DEFAULT 1,
		last_run TIMESTAMP,
		next_run TIMESTAMP,
		run_count INTEGER DEFAULT 0,
		max_runs INTEGER,
		created_by TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS schedule_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		schedule_id INTEGER NOT NULL,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		status TEXT NOT NULL,
		results_count INTEGER DEFAULT 0,
		errors_count INTEGER DEFAULT 0,
		error_message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	// Execute schema
	_, err := conn.Exec(schema)
	if err != nil {
		log.Printf("Warning: Failed to initialize in-memory schema: %v", err)
	}
}

// Implement Database interface methods
func (db *InMemoryDatabase) CreateSchedule(schedule *Schedule) error {
	// Implementation for testing would go here
	return nil
}

func (db *InMemoryDatabase) GetSchedule(id int64) (*Schedule, error) {
	// Implementation for testing would go here
	return nil, nil
}

func (db *InMemoryDatabase) ListSchedules(filters map[string]interface{}) ([]*Schedule, error) {
	// Implementation for testing would go here
	return []*Schedule{}, nil
}

func (db *InMemoryDatabase) CreateScheduleRun(run *ScheduleRun) error {
	// Implementation for testing would go here
	return nil
}

func (db *InMemoryDatabase) GetScheduleRun(id int64) (*ScheduleRun, error) {
	// Implementation for testing would go here
	return nil, nil
}

func (db *InMemoryDatabase) ListScheduleRuns(scheduleID string, limit int) ([]ScheduleRun, error) {
	// Implementation for testing would go here
	return []ScheduleRun{}, nil
}

func (db *InMemoryDatabase) UpdateScheduleRunInfo(scheduleID int64, lastRun time.Time, nextRun *time.Time, runCount int) error {
	// Implementation for testing would go here
	return nil
}

func (db *InMemoryDatabase) Close() error {
	return db.conn.Close()
}

func (db *InMemoryDatabase) begin() error {
	_, err := db.conn.Exec("BEGIN")
	return err
}

func (db *InMemoryDatabase) commit() error {
	_, err := db.conn.Exec("COMMIT")
	return err
}

func (db *InMemoryDatabase) rollback() error {
	_, err := db.conn.Exec("ROLLBACK")
	return err
}
