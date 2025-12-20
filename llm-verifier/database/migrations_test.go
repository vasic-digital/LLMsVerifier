package database

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMigrationManager(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	assert.NotNil(t, mm)
	assert.Equal(t, db, mm.db)
	assert.Empty(t, mm.migrations)
}

func TestAddMigration(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Add first migration
	migration1 := Migration{
		Version:     1,
		Description: "Test migration 1",
		Up:          func(tx *sql.Tx) error { return nil },
		Down:        func(tx *sql.Tx) error { return nil },
	}
	mm.AddMigration(migration1)

	assert.Len(t, mm.migrations, 1)
	assert.Equal(t, 1, mm.migrations[0].Version)

	// Add second migration with lower version (should be sorted)
	migration0 := Migration{
		Version:     0,
		Description: "Test migration 0",
		Up:          func(tx *sql.Tx) error { return nil },
		Down:        func(tx *sql.Tx) error { return nil },
	}
	mm.AddMigration(migration0)

	assert.Len(t, mm.migrations, 2)
	assert.Equal(t, 0, mm.migrations[0].Version) // Should be sorted
	assert.Equal(t, 1, mm.migrations[1].Version)
}

func TestInitializeMigrationTable(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Should not error
	err := mm.InitializeMigrationTable()
	require.NoError(t, err)

	// Check that table exists
	var tableName string
	err = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "schema_migrations", tableName)
}

func TestGetCurrentVersion(t *testing.T) {
	db := setupEmptyTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Initialize migration table
	err := mm.InitializeMigrationTable()
	require.NoError(t, err)

	// Should return 0 for new database
	version, err := mm.GetCurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 0, version)

	// Insert a migration record
	_, err = db.conn.Exec("INSERT INTO schema_migrations (version, description) VALUES (?, ?)", 1, "Test migration")
	require.NoError(t, err)

	// Should return inserted version
	version, err = mm.GetCurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestMigrateUp(t *testing.T) {
	db := setupEmptyTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Initialize migration table
	err := mm.InitializeMigrationTable()
	require.NoError(t, err)

	// Add a test migration
	applied := false
	migration := Migration{
		Version:     1,
		Description: "Test migration",
		Up: func(tx *sql.Tx) error {
			applied = true
			// Create a test table
			_, err := tx.Exec("CREATE TABLE test_table (id INTEGER)")
			return err
		},
		Down: func(tx *sql.Tx) error {
			_, err := tx.Exec("DROP TABLE test_table")
			return err
		},
	}
	mm.AddMigration(migration)

	// Migrate up
	err = mm.MigrateUp()
	require.NoError(t, err)
	assert.True(t, applied)

	// Check that migration was recorded
	version, err := mm.GetCurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 1, version)

	// Check that test table exists
	var tableName string
	err = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "test_table", tableName)
}

func TestMigrateUp_SkipApplied(t *testing.T) {
	db := setupEmptyTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Initialize migration table
	err := mm.InitializeMigrationTable()
	require.NoError(t, err)

	// Record migration as already applied
	_, err = db.conn.Exec("INSERT INTO schema_migrations (version, description) VALUES (?, ?)", 1, "Test migration")
	require.NoError(t, err)

	// Add test migration
	applied := false
	migration := Migration{
		Version:     1,
		Description: "Test migration",
		Up: func(tx *sql.Tx) error {
			applied = true
			return nil
		},
		Down: func(tx *sql.Tx) error { return nil },
	}
	mm.AddMigration(migration)

	// Migrate up
	err = mm.MigrateUp()
	require.NoError(t, err)
	assert.False(t, applied) // Should not be applied again
}

func TestMigrateDown(t *testing.T) {
	db := setupEmptyTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Initialize migration table
	err := mm.InitializeMigrationTable()
	require.NoError(t, err)

	// Add and apply a test migration
	migration := Migration{
		Version:     1,
		Description: "Test migration",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE test_table (id INTEGER)")
			return err
		},
		Down: func(tx *sql.Tx) error {
			_, err := tx.Exec("DROP TABLE test_table")
			return err
		},
	}
	mm.AddMigration(migration)

	// Apply migration first
	err = mm.MigrateUp()
	require.NoError(t, err)

	// Verify table exists
	var tableName string
	err = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "test_table", tableName)

	// Roll back migration
	err = mm.MigrateDown()
	require.NoError(t, err)

	// Verify table no longer exists
	err = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	assert.Error(t, err) // Should error since table doesn't exist

	// Check that migration was removed from records
	version, err := mm.GetCurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 0, version)
}

func TestMigrateDown_NoMigrations(t *testing.T) {
	db := setupEmptyTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Initialize migration table
	err := mm.InitializeMigrationTable()
	require.NoError(t, err)

	// Try to roll back with no migrations
	err = mm.MigrateDown()
	require.NoError(t, err)
}

func TestGetMigrationStatus(t *testing.T) {
	db := setupEmptyTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Initialize migration table
	err := mm.InitializeMigrationTable()
	require.NoError(t, err)

	// Add test migrations
	migration1 := Migration{
		Version:     1,
		Description: "Test migration 1",
		Up:          func(tx *sql.Tx) error { return nil },
		Down:        func(tx *sql.Tx) error { return nil },
	}
	migration2 := Migration{
		Version:     2,
		Description: "Test migration 2",
		Up:          func(tx *sql.Tx) error { return nil },
		Down:        func(tx *sql.Tx) error { return nil },
	}
	mm.AddMigration(migration1)
	mm.AddMigration(migration2)

	// Apply first migration
	err = mm.MigrateUp()
	require.NoError(t, err)

	// Get status
	status, err := mm.GetMigrationStatus()
	require.NoError(t, err)
	assert.Len(t, status, 2)

	// Check first migration status
	assert.Equal(t, 1, status[0].Version)
	assert.Equal(t, "Test migration 1", status[0].Description)
	assert.True(t, status[0].Applied)

	// Check second migration status
	assert.Equal(t, 2, status[1].Version)
	assert.Equal(t, "Test migration 2", status[1].Description)
	assert.True(t, status[1].Applied) // Both migrations should be applied after MigrateUp()
}

func TestSetupDefaultMigrations(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Setup default migrations
	mm.SetupDefaultMigrations()

	// Should have 4 default migrations
	assert.Len(t, mm.migrations, 4)

	// Check migration versions
	versions := []int{}
	for _, migration := range mm.migrations {
		versions = append(versions, migration.Version)
	}
	assert.Contains(t, versions, 1)
	assert.Contains(t, versions, 2)
	assert.Contains(t, versions, 3)
	assert.Contains(t, versions, 4)
}

func TestMigrationError_Up(t *testing.T) {
	db := setupEmptyTestDatabase(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	// Initialize migration table
	err := mm.InitializeMigrationTable()
	require.NoError(t, err)

	// Add a failing migration
	migration := Migration{
		Version:     1,
		Description: "Failing migration",
		Up: func(tx *sql.Tx) error {
			return assert.AnError
		},
		Down: func(tx *sql.Tx) error { return nil },
	}
	mm.AddMigration(migration)

	// Migrate up should fail
	err = mm.MigrateUp()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to apply migration 1")

	// Migration should not be recorded
	version, err := mm.GetCurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 0, version)
}

func TestMigrationError_Down(t *testing.T) {
	db := setupEmptyTestDatabase(t)  // Use empty DB to avoid default migrations
	defer db.Close()

	mm := NewMigrationManager(db)

	// Initialize migration table
	err := mm.InitializeMigrationTable()
	require.NoError(t, err)

	// Add and apply a migration
	migration := Migration{
		Version:     1,
		Description: "Test migration",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE test_table (id INTEGER)")
			return err
		},
		Down: func(tx *sql.Tx) error {
			return assert.AnError
		},
	}
	mm.AddMigration(migration)

	// Apply migration
	err = mm.MigrateUp()
	require.NoError(t, err)

	// Try to roll back failing migration
	err = mm.MigrateDown()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to roll back migration 1")

	// Migration should still be recorded (rollback failed)
	version, err := mm.GetCurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}

// Helper function to set up test database
func setupTestDatabase(t *testing.T) *Database {
	// Create in-memory database for testing
	db, err := New(":memory:")
	require.NoError(t, err)
	return db
}

// setupEmptyTestDatabase creates a database without running migrations
func setupEmptyTestDatabase(t *testing.T) *Database {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	database := &Database{
		conn: sqlDB,
	}

	return database
}
