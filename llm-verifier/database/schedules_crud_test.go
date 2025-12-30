package database

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScheduleTestDB(t *testing.T) *Database {
	dbFile := "/tmp/test_schedules_" + time.Now().Format("20060102150405") + ".db"
	db, err := New(dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbFile)
	})
	return db
}

func createTestSchedule() *Schedule {
	desc := "Test schedule description"
	cronExpr := "0 0 * * *"
	return &Schedule{
		Name:           "test-schedule",
		Description:    &desc,
		ScheduleType:   "cron",
		CronExpression: &cronExpr,
		TargetType:     "all_models",
		IsActive:       true,
		RunCount:       0,
	}
}

// ==================== Schedule CRUD Tests ====================

func TestCreateSchedule(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)
	assert.NotZero(t, schedule.ID)
}

func TestCreateSchedule_Interval(t *testing.T) {
	db := setupScheduleTestDB(t)

	interval := 3600 // 1 hour
	schedule := &Schedule{
		Name:            "interval-schedule",
		ScheduleType:    "interval",
		IntervalSeconds: &interval,
		TargetType:      "all_models",
		IsActive:        true,
	}

	err := db.CreateSchedule(schedule)
	require.NoError(t, err)
	assert.NotZero(t, schedule.ID)
}

func TestCreateSchedule_WithDates(t *testing.T) {
	db := setupScheduleTestDB(t)

	lastRun := time.Now().Add(-24 * time.Hour)
	nextRun := time.Now().Add(24 * time.Hour)
	maxRuns := 10
	createdBy := "admin"

	schedule := &Schedule{
		Name:         "dated-schedule",
		ScheduleType: "cron",
		TargetType:   "all_models",
		IsActive:     true,
		LastRun:      &lastRun,
		NextRun:      &nextRun,
		MaxRuns:      &maxRuns,
		CreatedBy:    &createdBy,
	}

	err := db.CreateSchedule(schedule)
	require.NoError(t, err)
	assert.NotZero(t, schedule.ID)
}

func TestGetSchedule(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)

	retrieved, err := db.GetSchedule(schedule.ID)
	require.NoError(t, err)
	assert.Equal(t, schedule.ID, retrieved.ID)
	assert.Equal(t, schedule.Name, retrieved.Name)
	assert.Equal(t, schedule.ScheduleType, retrieved.ScheduleType)
	assert.Equal(t, schedule.TargetType, retrieved.TargetType)
}

func TestGetSchedule_NotFound(t *testing.T) {
	db := setupScheduleTestDB(t)

	_, err := db.GetSchedule(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateSchedule(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)

	schedule.Name = "updated-schedule"
	schedule.IsActive = false
	err = db.UpdateSchedule(schedule)
	require.NoError(t, err)

	retrieved, err := db.GetSchedule(schedule.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-schedule", retrieved.Name)
	assert.False(t, retrieved.IsActive)
}

func TestDeleteSchedule(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)

	err = db.DeleteSchedule(schedule.ID)
	require.NoError(t, err)

	_, err = db.GetSchedule(schedule.ID)
	assert.Error(t, err)
}

func TestListSchedules_NoFilters(t *testing.T) {
	db := setupScheduleTestDB(t)

	for i := 0; i < 3; i++ {
		schedule := createTestSchedule()
		schedule.Name = "schedule-" + string(rune('A'+i))
		err := db.CreateSchedule(schedule)
		require.NoError(t, err)
	}

	schedules, err := db.ListSchedules(nil)
	require.NoError(t, err)
	assert.Len(t, schedules, 3)
}

func TestListSchedules_FilterByActive(t *testing.T) {
	db := setupScheduleTestDB(t)

	// Create active schedules
	for i := 0; i < 2; i++ {
		schedule := createTestSchedule()
		schedule.Name = "active-" + string(rune('A'+i))
		schedule.IsActive = true
		err := db.CreateSchedule(schedule)
		require.NoError(t, err)
	}

	// Create inactive schedule
	inactiveSchedule := createTestSchedule()
	inactiveSchedule.Name = "inactive"
	inactiveSchedule.IsActive = false
	err := db.CreateSchedule(inactiveSchedule)
	require.NoError(t, err)

	schedules, err := db.ListSchedules(map[string]interface{}{"is_active": true})
	require.NoError(t, err)
	assert.Len(t, schedules, 2)
}

func TestListSchedules_FilterByType(t *testing.T) {
	db := setupScheduleTestDB(t)

	// Create cron schedule
	cronSchedule := createTestSchedule()
	err := db.CreateSchedule(cronSchedule)
	require.NoError(t, err)

	// Create interval schedule
	interval := 3600
	intervalSchedule := &Schedule{
		Name:            "interval",
		ScheduleType:    "interval",
		IntervalSeconds: &interval,
		TargetType:      "all_models",
		IsActive:        true,
	}
	err = db.CreateSchedule(intervalSchedule)
	require.NoError(t, err)

	schedules, err := db.ListSchedules(map[string]interface{}{"schedule_type": "cron"})
	require.NoError(t, err)
	assert.Len(t, schedules, 1)
	assert.Equal(t, "cron", schedules[0].ScheduleType)
}

func TestGetActiveSchedules(t *testing.T) {
	db := setupScheduleTestDB(t)

	// Create active schedules
	for i := 0; i < 2; i++ {
		schedule := createTestSchedule()
		schedule.IsActive = true
		err := db.CreateSchedule(schedule)
		require.NoError(t, err)
	}

	// Create inactive schedule
	inactiveSchedule := createTestSchedule()
	inactiveSchedule.IsActive = false
	err := db.CreateSchedule(inactiveSchedule)
	require.NoError(t, err)

	schedules, err := db.GetActiveSchedules()
	require.NoError(t, err)
	assert.Len(t, schedules, 2)
}

func TestUpdateScheduleRunInfo(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)

	now := time.Now()
	next := now.Add(24 * time.Hour)
	err = db.UpdateScheduleRunInfo(schedule.ID, now, &next, 5)
	require.NoError(t, err)

	retrieved, err := db.GetSchedule(schedule.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, retrieved.RunCount)
	assert.NotNil(t, retrieved.LastRun)
	assert.NotNil(t, retrieved.NextRun)
}

// ==================== ScheduleRun CRUD Tests ====================

func TestCreateScheduleRun(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)

	run := &ScheduleRun{
		ScheduleID: schedule.ID,
		StartedAt:  time.Now(),
		Status:     "running",
	}

	err = db.CreateScheduleRun(run)
	require.NoError(t, err)
	assert.NotZero(t, run.ID)
}

func TestGetScheduleRun(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)

	run := &ScheduleRun{
		ScheduleID: schedule.ID,
		StartedAt:  time.Now(),
		Status:     "running",
	}
	err = db.CreateScheduleRun(run)
	require.NoError(t, err)

	retrieved, err := db.GetScheduleRun(run.ID)
	require.NoError(t, err)
	assert.Equal(t, run.ID, retrieved.ID)
	assert.Equal(t, run.ScheduleID, retrieved.ScheduleID)
	assert.Equal(t, "running", retrieved.Status)
}

func TestGetScheduleRun_NotFound(t *testing.T) {
	db := setupScheduleTestDB(t)

	_, err := db.GetScheduleRun(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateScheduleRun(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)

	run := &ScheduleRun{
		ScheduleID: schedule.ID,
		StartedAt:  time.Now(),
		Status:     "running",
	}
	err = db.CreateScheduleRun(run)
	require.NoError(t, err)

	// Update
	completedAt := time.Now()
	run.CompletedAt = &completedAt
	run.Status = "completed"
	run.ResultsCount = 10
	run.ErrorsCount = 2

	err = db.UpdateScheduleRun(run)
	require.NoError(t, err)

	retrieved, err := db.GetScheduleRun(run.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", retrieved.Status)
	assert.Equal(t, 10, retrieved.ResultsCount)
	assert.Equal(t, 2, retrieved.ErrorsCount)
}

func TestDeleteScheduleRun(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)

	run := &ScheduleRun{
		ScheduleID: schedule.ID,
		StartedAt:  time.Now(),
		Status:     "completed",
	}
	err = db.CreateScheduleRun(run)
	require.NoError(t, err)

	err = db.DeleteScheduleRun(run.ID)
	require.NoError(t, err)

	_, err = db.GetScheduleRun(run.ID)
	assert.Error(t, err)
}

func TestGetScheduleRuns(t *testing.T) {
	db := setupScheduleTestDB(t)

	schedule := createTestSchedule()
	err := db.CreateSchedule(schedule)
	require.NoError(t, err)

	// Create multiple runs
	for i := 0; i < 5; i++ {
		run := &ScheduleRun{
			ScheduleID: schedule.ID,
			StartedAt:  time.Now(),
			Status:     "completed",
		}
		err = db.CreateScheduleRun(run)
		require.NoError(t, err)
	}

	runs, err := db.GetScheduleRuns(schedule.ID)
	require.NoError(t, err)
	assert.Len(t, runs, 5)
}

// ==================== Struct Tests ====================

func TestSchedule_Struct(t *testing.T) {
	desc := "Test description"
	cron := "0 0 * * *"
	interval := 3600
	targetID := int64(123)
	maxRuns := 100
	createdBy := "admin"
	now := time.Now()

	schedule := Schedule{
		ID:              1,
		Name:            "test-schedule",
		Description:     &desc,
		ScheduleType:    "cron",
		CronExpression:  &cron,
		IntervalSeconds: &interval,
		TargetType:      "provider",
		TargetID:        &targetID,
		IsActive:        true,
		LastRun:         &now,
		NextRun:         &now,
		RunCount:        5,
		MaxRuns:         &maxRuns,
		CreatedAt:       now,
		UpdatedAt:       now,
		CreatedBy:       &createdBy,
	}

	assert.Equal(t, int64(1), schedule.ID)
	assert.Equal(t, "test-schedule", schedule.Name)
	assert.Equal(t, "cron", schedule.ScheduleType)
	assert.Equal(t, 5, schedule.RunCount)
}

func TestScheduleRun_Struct(t *testing.T) {
	now := time.Now()
	errMsg := "Test error"

	run := ScheduleRun{
		ID:           1,
		ScheduleID:   100,
		StartedAt:    now,
		CompletedAt:  &now,
		Status:       "failed",
		ResultsCount: 8,
		ErrorsCount:  2,
		ErrorMessage: &errMsg,
		CreatedAt:    now,
	}

	assert.Equal(t, int64(1), run.ID)
	assert.Equal(t, int64(100), run.ScheduleID)
	assert.Equal(t, "failed", run.Status)
	assert.Equal(t, 2, run.ErrorsCount)
}
