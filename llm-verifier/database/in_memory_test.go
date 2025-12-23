package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewInMemoryDatabase(t *testing.T) {
	db := NewInMemoryDatabase()
	assert.NotNil(t, db)
	assert.NotNil(t, db.conn)
	defer db.Close()
}

func TestInMemoryDatabaseClose(t *testing.T) {
	db := NewInMemoryDatabase()
	err := db.Close()
	assert.NoError(t, err)
}

func TestInMemoryDatabaseCreateSchedule(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	desc := "Test Description"
	expr := "0 * * * *"
	schedule := &Schedule{
		Name:           "Test Schedule",
		Description:    &desc,
		ScheduleType:    "cron",
		CronExpression:  &expr,
		IsActive:       true,
	}
	err := db.CreateSchedule(schedule)
	assert.NoError(t, err)
}

func TestInMemoryDatabaseGetSchedule(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	schedule, err := db.GetSchedule(1)
	assert.NoError(t, err)
	assert.Nil(t, schedule)
}

func TestInMemoryDatabaseListSchedules(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	schedules, err := db.ListSchedules(nil)
	assert.NoError(t, err)
	assert.NotNil(t, schedules)
}

func TestInMemoryDatabaseCreateScheduleRun(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	started := time.Now()
	status := "running"
	run := &ScheduleRun{
		ScheduleID:   1,
		StartedAt:    started,
		Status:       status,
		ResultsCount: 0,
		ErrorsCount:  0,
	}
	err := db.CreateScheduleRun(run)
	assert.NoError(t, err)
}

func TestInMemoryDatabaseGetScheduleRun(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	run, err := db.GetScheduleRun(1)
	assert.NoError(t, err)
	assert.Nil(t, run)
}

func TestInMemoryDatabaseListScheduleRuns(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	runs, err := db.ListScheduleRuns("1", 10)
	assert.NoError(t, err)
	assert.NotNil(t, runs)
}

func TestInMemoryDatabaseUpdateScheduleRunInfo(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	lastRun := time.Now()
	nextRun := time.Now().Add(time.Hour)
	err := db.UpdateScheduleRunInfo(1, lastRun, &nextRun, 5)
	assert.NoError(t, err)
}

func TestInMemoryDatabaseBegin(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	err := db.begin()
	assert.NoError(t, err)
}

func TestInMemoryDatabaseCommit(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	err := db.begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	err = db.commit()
	assert.NoError(t, err)
}

func TestInMemoryDatabaseRollback(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	err := db.begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	err = db.rollback()
	assert.NoError(t, err)
}

func TestInMemoryDatabaseConnection(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	err := db.conn.Ping()
	assert.NoError(t, err)
}

func TestInMemoryDatabaseSchema(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	var tables []string
	rows, err := db.conn.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Failed to scan table name: %v", err)
		}
		tables = append(tables, name)
	}

	assert.NotEmpty(t, tables)
	assert.Contains(t, tables, "schedules")
	assert.Contains(t, tables, "schedule_runs")
}

func TestScheduleStruct(t *testing.T) {
	desc := "Test Description"
	expr := "0 * * * *"
	interval := 3600
	targetID := int64(1)
	lastRun := time.Now()
	nextRun := time.Now().Add(time.Hour)
	maxRuns := 100
	createdBy := "admin"

	schedule := Schedule{
		ID:              1,
		Name:            "Daily Test",
		Description:      &desc,
		ScheduleType:     "cron",
		CronExpression:   &expr,
		IntervalSeconds:  &interval,
		TargetType:      "model",
		TargetID:        &targetID,
		IsActive:        true,
		LastRun:         &lastRun,
		NextRun:         &nextRun,
		RunCount:        10,
		MaxRuns:         &maxRuns,
		CreatedBy:       &createdBy,
	}

	assert.Equal(t, int64(1), schedule.ID)
	assert.Equal(t, "Daily Test", schedule.Name)
	assert.Equal(t, "cron", schedule.ScheduleType)
}

func TestScheduleRunStruct(t *testing.T) {
	started := time.Now()
	completed := time.Now().Add(time.Hour)
	status := "completed"
	msg := ""

	run := ScheduleRun{
		ID:            1,
		ScheduleID:    1,
		StartedAt:     started,
		CompletedAt:   &completed,
		Status:        status,
		ResultsCount:  5,
		ErrorsCount:   0,
		ErrorMessage:  &msg,
	}

	assert.Equal(t, int64(1), run.ID)
	assert.Equal(t, int64(1), run.ScheduleID)
	assert.Equal(t, "completed", run.Status)
}

func TestInMemoryDatabaseMultipleSchedules(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	expr := "0 * * * *"
	for i := 0; i < 3; i++ {
		schedule := &Schedule{
			Name:           "Test Schedule",
			ScheduleType:    "cron",
			CronExpression:  &expr,
			IsActive:       true,
		}
		err := db.CreateSchedule(schedule)
		assert.NoError(t, err)
	}
}

func TestInMemoryDatabaseMultipleRuns(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	for i := 0; i < 3; i++ {
		run := &ScheduleRun{
			ScheduleID:   1,
			StartedAt:    time.Now(),
			Status:       "running",
			ResultsCount: 0,
			ErrorsCount:  0,
		}
		err := db.CreateScheduleRun(run)
		assert.NoError(t, err)
	}
}

func TestInMemoryDatabaseEmptySchedule(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	schedule := &Schedule{}
	err := db.CreateSchedule(schedule)
	assert.NoError(t, err)
}

func TestInMemoryDatabaseNilSchedule(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	err := db.CreateSchedule(nil)
	assert.NoError(t, err)
}

func TestInMemoryDatabaseEmptyList(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	schedules, err := db.ListSchedules(map[string]interface{}{})
	assert.NoError(t, err)
	assert.NotNil(t, schedules)
}

func TestInMemoryDatabaseListSchedulesWithFilters(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	filters := map[string]interface{}{
		"schedule_type": "cron",
		"is_active":      true,
	}

	schedules, err := db.ListSchedules(filters)
	assert.NoError(t, err)
	assert.NotNil(t, schedules)
}

func TestScheduleStatusValues(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	msg := ""
	statuses := []string{"active", "inactive", "paused", "completed"}

	for _, status := range statuses {
		run := &ScheduleRun{
			ScheduleID:   1,
			StartedAt:     time.Now(),
			Status:        status,
			ResultsCount:  0,
			ErrorsCount:   0,
			ErrorMessage:  &msg,
		}
		err := db.CreateScheduleRun(run)
		assert.NoError(t, err)
	}
}

func TestInMemoryDatabaseType(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	assert.IsType(t, &InMemoryDatabase{}, db)
}

func TestInMemoryDatabaseScheduleID(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	schedule, err := db.GetSchedule(0)
	assert.NoError(t, err)
	assert.Nil(t, schedule)

	schedule, err = db.GetSchedule(-1)
	assert.NoError(t, err)
	assert.Nil(t, schedule)
}

func TestInMemoryDatabaseListSchedulesWithLimit(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	runs, err := db.ListScheduleRuns("1", 5)
	assert.NoError(t, err)
	assert.NotNil(t, runs)

	runs, err = db.ListScheduleRuns("1", 0)
	assert.NoError(t, err)
	assert.NotNil(t, runs)

	runs, err = db.ListScheduleRuns("1", -1)
	assert.NoError(t, err)
	assert.NotNil(t, runs)
}

func TestInMemoryDatabaseUpdateScheduleRunInfoNilNextRun(t *testing.T) {
	db := NewInMemoryDatabase()
	defer db.Close()

	lastRun := time.Now()
	err := db.UpdateScheduleRunInfo(1, lastRun, nil, 5)
	assert.NoError(t, err)
}
