# Database Comprehensive Challenge

## Overview
This challenge validates SQLite database with SQL Cipher encryption, log database, proper indexing, and all database operations.

## Challenge Type
Integration Test + Security Test + Performance Test

## Test Scenarios

### 1. SQLite with SQL Cipher Challenge
**Objective**: Verify database encryption with SQL Cipher

**Steps**:
1. Create encrypted database
2. Verify data is encrypted
3. Test decryption with correct password
4. Test with incorrect password
5. Verify encryption strength

**Expected Results**:
- Database is encrypted
- Data cannot be read without password
- Correct password decrypts data
- Incorrect password fails

**Test Code**:
```go
func TestSQLCipherEncryption(t *testing.T) {
    db := NewEncryptedDatabase("test.db", "strong-password-123")

    // Insert test data
    err := db.InsertModel(Model{ID: "gpt-4", Provider: "openai"})
    assert.NoError(t, err)

    // Close and reopen with correct password
    db.Close()
    db2 := NewEncryptedDatabase("test.db", "strong-password-123")

    model, err := db2.GetModel("gpt-4")
    assert.NoError(t, err)
    assert.Equal(t, "gpt-4", model.ID)

    // Try with incorrect password
    db3 := NewEncryptedDatabase("test.db", "wrong-password")
    _, err = db3.GetModel("gpt-4")
    assert.Error(t, err)
}
```

---

### 2. Database Schema Challenge
**Objective**: Verify database schema is correct

**Tables**:
- providers
- models
- verifications
- scores
- limits
- pricing
- events
- logs

**Steps**:
1. Verify all tables exist
2. Verify schema structure
3. Verify constraints
4. Verify relationships

**Expected Results**:
- All tables created
- Schema matches specification
- Constraints work
- Relationships enforced

**Test Code**:
```go
func TestDatabaseSchema(t *testing.T) {
    db := NewDatabase("test.db")

    // Check tables exist
    tables, err := db.GetTables()
    assert.NoError(t, err)
    assert.Contains(t, tables, "models")
    assert.Contains(t, tables, "verifications")

    // Check schema
    schema, err := db.GetTableSchema("models")
    assert.NoError(t, err)
    assert.Contains(t, schema, "id")
    assert.Contains(t, schema, "provider")
    assert.Contains(t, schema, "score")
}
```

---

### 3. Database Indexing Challenge
**Objective**: Verify database has proper indexes

**Required Indexes**:
- models: provider, score, features
- verifications: model_id, timestamp
- scores: model_id, timestamp
- logs: timestamp, level

**Steps**:
1. Verify indexes exist
2. Test query performance with indexes
3. Test without indexes
4. Compare performance

**Expected Results**:
- All indexes created
- Queries use indexes
- Performance is optimal

**Test Code**:
```go
func TestDatabaseIndexing(t *testing.T) {
    db := NewDatabase("test.db")

    // Check indexes
    indexes, err := db.GetIndexes("models")
    assert.NoError(t, err)
    assert.Contains(t, indexes, "idx_models_provider")
    assert.Contains(t, indexes, "idx_models_score")

    // Test query performance
    start := time.Now()
    models, _ := db.QueryModels("provider = 'openai'")
    duration := time.Since(start)

    assert.Less(t, duration, 10*time.Millisecond)
}
```

---

### 4. CRUD Operations Challenge
**Objective**: Verify all CRUD operations work

**Steps**:
1. Create model record
2. Read model record
3. Update model record
4. Delete model record
5. Verify all operations

**Expected Results**:
- Create works
- Read returns correct data
- Update modifies data
- Delete removes data

**Test Code**:
```go
func TestCRUDOperations(t *testing.T) {
    db := NewDatabase("test.db")

    // Create
    model := Model{
        ID:      "test-model",
        Provider: "test-provider",
        Score:   85,
    }
    err := db.InsertModel(model)
    assert.NoError(t, err)

    // Read
    retrieved, err := db.GetModel("test-model")
    assert.NoError(t, err)
    assert.Equal(t, 85, retrieved.Score)

    // Update
    retrieved.Score = 90
    err = db.UpdateModel(retrieved)
    assert.NoError(t, err)

    updated, _ := db.GetModel("test-model")
    assert.Equal(t, 90, updated.Score)

    // Delete
    err = db.DeleteModel("test-model")
    assert.NoError(t, err)

    _, err = db.GetModel("test-model")
    assert.Error(t, err)
}
```

---

### 5. Log Database Challenge
**Objective**: Verify separate log database works

**Steps**:
1. Initialize log database
2. Log messages at different levels
3. Query logs by level
4. Query logs by time range
5. Verify log format

**Expected Results**:
- Log database created
- Messages logged correctly
- Queries work
- Format is consistent

**Test Code**:
```go
func TestLogDatabase(t *testing.T) {
    logDB := NewLogDatabase("logs.db")

    // Log messages
    logDB.Log("INFO", "Test info message", map[string]interface{}{"key": "value"})
    logDB.Log("ERROR", "Test error message", nil)
    logDB.Log("WARN", "Test warning message", nil)

    // Query by level
    errors := logDB.QueryByLevel("ERROR")
    assert.Equal(t, 1, len(errors))
    assert.Contains(t, errors[0].Message, "Test error message")

    // Query by time range
    logs := logDB.QueryByTimeRange(time.Now().Add(-1*time.Hour), time.Now())
    assert.GreaterOrEqual(t, len(logs), 3)
}
```

---

### 6. Database Migration Challenge
**Objective**: Verify database migrations work

**Steps**:
1. Create initial schema
2. Add migration to add column
3. Run migration
4. Verify new column exists
5. Test rollback

**Expected Results**:
- Migrations apply correctly
- New columns added
- Data preserved
- Rollback works

**Test Code**:
```go
func TestDatabaseMigration(t *testing.T) {
    migrator := NewMigrator("test.db")

    // Initial migration
    err := migrator.Up("001_initial_schema.sql")
    assert.NoError(t, err)

    // Add column migration
    err = migrator.Up("002_add_category.sql")
    assert.NoError(t, err)

    // Verify column exists
    db := NewDatabase("test.db")
    models, _ := db.GetTableSchema("models")
    assert.Contains(t, models, "category")

    // Rollback
    err = migrator.Down("002_add_category.sql")
    assert.NoError(t, err)

    models, _ = db.GetTableSchema("models")
    assert.NotContains(t, models, "category")
}
```

---

### 7. Database Backup and Restore Challenge
**Objective**: Verify backup and restore operations

**Steps**:
1. Create backup
2. Verify backup file exists
3. Modify database
4. Restore from backup
5. Verify data restored

**Expected Results**:
- Backup created
- Backup file valid
- Restore works
- Data matches backup

**Test Code**:
```go
func TestDatabaseBackupRestore(t *testing.T) {
    db := NewDatabase("test.db")
    db.InsertModel(Model{ID: "gpt-4", Score: 95})

    // Backup
    backupManager := NewBackupManager(db)
    err := backupManager.Backup("backup.db")
    assert.NoError(t, err)
    assert.FileExists(t, "backup.db")

    // Modify
    db.UpdateModel(Model{ID: "gpt-4", Score: 85})

    // Restore
    err = backupManager.Restore("backup.db")
    assert.NoError(t, err)

    // Verify restored
    model, _ := db.GetModel("gpt-4")
    assert.Equal(t, 95, model.Score)
}
```

---

### 8. Database Query Optimization Challenge
**Objective**: Verify queries are optimized

**Steps**:
1. Run EXPLAIN on queries
2. Verify index usage
3. Identify slow queries
4. Optimize queries

**Expected Results**:
- Queries use indexes
- Query plans are optimal
- No full table scans

**Test Code**:
```go
func TestQueryOptimization(t *testing.T) {
    db := NewDatabase("test.db")

    // Explain query
    plan, err := db.ExplainQuery("SELECT * FROM models WHERE provider = 'openai'")
    assert.NoError(t, err)

    // Verify index is used
    assert.Contains(t, plan, "idx_models_provider")

    // Check for table scan
    assert.NotContains(t, plan, "SCAN TABLE")
}
```

---

### 9. Database Concurrency Challenge
**Objective**: Verify database handles concurrent operations

**Steps**:
1. Perform concurrent reads
2. Perform concurrent writes
3. Test with multiple goroutines
4. Verify data integrity

**Expected Results**:
- Concurrent reads work
- Concurrent writes work
- No data corruption
- No deadlocks

**Test Code**:
```go
func TestDatabaseConcurrency(t *testing.T) {
    db := NewDatabase("test.db")

    // Concurrent writes
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            db.InsertModel(Model{ID: fmt.Sprintf("model-%d", id), Score: id})
        }(i)
    }
    wg.Wait()

    // Verify all inserted
    count, _ := db.CountModels()
    assert.Equal(t, 100, count)
}
```

---

### 10. Database Performance Challenge
**Objective**: Verify database performance is acceptable

**Metrics**:
- Insert time
- Query time
- Index creation time
- Backup time

**Steps**:
1. Measure insert performance
2. Measure query performance
3. Measure bulk operations
4. Verify within limits

**Expected Results**:
- Inserts < 10ms
- Queries < 50ms
- Bulk operations scale linearly

**Test Code**:
```go
func TestDatabasePerformance(t *testing.T) {
    db := NewDatabase("test.db")

    // Insert performance
    start := time.Now()
    for i := 0; i < 1000; i++ {
        db.InsertModel(Model{ID: fmt.Sprintf("model-%d", i), Score: i % 100})
    }
    duration := time.Since(start)
    avgDuration := duration / 1000
    assert.Less(t, avgDuration, 10*time.Millisecond)

    // Query performance
    start = time.Now()
    models, _ := db.QueryModels("score > 50")
    duration = time.Since(start)
    assert.Less(t, duration, 50*time.Millisecond)
}
```

---

## Success Criteria

### Functional Requirements
- [ ] Encryption works (SQL Cipher)
- [ ] Schema is correct
- [ ] Indexes exist and work
- [ ] CRUD operations work
- [ ] Log database works
- [ ] Migrations work
- [ ] Backup/restore works
- [ ] Queries are optimized
- [ ] Concurrency handled
- [ ] Performance is acceptable

### Security Requirements
- [ ] Database encrypted
- [ ] Data cannot be read without password
- [ ] No SQL injection vulnerabilities
- [ ] Access controls work

### Performance Requirements
- [ ] Insert < 10ms
- [ ] Query < 50ms
- [ ] Bulk insert 1000 records < 1 second

## Dependencies
- SQLite with SQL Cipher
- Test database path

## Cleanup
- Remove test databases
- Remove backup files
