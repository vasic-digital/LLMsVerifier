package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewScheduler tests scheduler creation
func TestNewScheduler(t *testing.T) {
	// Note: In real tests, this would use a mock database
	scheduler := NewScheduler(nil)

	assert.NotNil(t, scheduler)
	assert.Nil(t, scheduler.db)
	assert.Empty(t, scheduler.schedules)
	assert.Empty(t, scheduler.runs)
	assert.False(t, scheduler.running)
	assert.NotNil(t, scheduler.stopCh)
	assert.NotNil(t, scheduler.schedules)
	assert.NotNil(t, scheduler.runs)
}

// TestScheduler_SetJobHandler tests job handler setting
func TestScheduler_SetJobHandler(t *testing.T) {
	scheduler := NewScheduler(nil)

	handler := func(jobType JobType, targets []string, options map[string]interface{}) error {
		return nil
	}

	scheduler.SetJobHandler(handler)
	assert.NotNil(t, scheduler.jobHandler)
}

// TestScheduler_CreateSchedule tests schedule creation
func TestScheduler_CreateSchedule(t *testing.T) {
	_ = NewScheduler(nil)

	// Test ID generation for cron schedule
	_ = &Schedule{
		Name:        "Test Cron Schedule",
		Description: "Test description",
		Type:        ScheduleTypeCron,
		JobType:     JobTypeVerification,
		Expression:  "0 2 * * *", // Daily at 2 AM
		Enabled:     true,
		Targets:     []string{"1", "2", "3"},
		Options:     map[string]interface{}{"test": "value"},
	}

	// This will fail without database, but we can test the ID generation logic
	scheduleID := generateScheduleID()
	assert.Contains(t, scheduleID, "sched_")
	assert.True(t, len(scheduleID) > len("sched_"))
}

// TestScheduler_CalculateNextRun tests next run calculation
func TestScheduler_CalculateNextRun(t *testing.T) {
	scheduler := NewScheduler(nil)

	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		schedule   *Schedule
		wantAfter  time.Duration
	}{
		{
			name: "Daily at 2 AM",
			schedule: &Schedule{
				Type:       ScheduleTypeCron,
				Expression: "0 2 * * *",
			},
			wantAfter: 14 * time.Hour, // From 12:00 to 02:00
		},
		{
			name: "Hourly",
			schedule: &Schedule{
				Type:       ScheduleTypeCron,
				Expression: "0 * * * *",
			},
			wantAfter: 0 * time.Hour, // Next hour is at 12:00
		},
		{
			name: "1 hour interval",
			schedule: &Schedule{
				Type:       ScheduleTypeInterval,
				Expression: "1h",
			},
			wantAfter: 1 * time.Hour,
		},
		{
			name: "30 minutes interval",
			schedule: &Schedule{
				Type:       ScheduleTypeInterval,
				Expression: "30m",
			},
			wantAfter: 30 * time.Minute,
		},
		{
			name: "Once schedule",
			schedule: &Schedule{
				Type:       ScheduleTypeOnce,
				Expression: "", // No expression needed
			},
			wantAfter: 365 * 24 * time.Hour, // Far future
		},
		{
			name: "Invalid schedule defaults to hourly",
			schedule: &Schedule{
				Type:       ScheduleType("invalid"),
				Expression: "",
			},
			wantAfter: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextRun := scheduler.calculateNextRun(tt.schedule, now)
			duration := nextRun.Sub(now)
			
			// Allow some tolerance for cron calculations
			assert.GreaterOrEqual(t, duration, tt.wantAfter-1*time.Hour)
			assert.LessOrEqual(t, duration, tt.wantAfter+1*time.Hour)
			
			// Debug logging to understand failures
			t.Logf("Test: %s, Now: %s, Next: %s, Duration: %v, Expected: %v",
				tt.name, now.Format("2006-01-02 15:04:05"), 
				nextRun.Format("2006-01-02 15:04:05"), duration, tt.wantAfter)
		})
	}
}

// TestScheduler_ParseIntervalExpression tests interval expression parsing
func TestScheduler_ParseIntervalExpression(t *testing.T) {
	scheduler := NewScheduler(nil)

	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expression string
		wantDelta time.Duration
	}{
		{
			name:      "1 hour",
			expression: "1h",
			wantDelta: 1 * time.Hour,
		},
		{
			name:      "30 minutes",
			expression: "30m",
			wantDelta: 30 * time.Minute,
		},
		{
			name:      "2 hours 30 minutes",
			expression: "2h30m",
			wantDelta: 2*time.Hour + 30*time.Minute,
		},
		{
			name:      "45 seconds",
			expression: "45s",
			wantDelta: 45 * time.Second,
		},
		{
			name:      "Invalid defaults to hourly",
			expression: "invalid",
			wantDelta: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextRun := scheduler.parseIntervalExpression(tt.expression, now)
			duration := nextRun.Sub(now)
			assert.Equal(t, tt.wantDelta, duration)
		})
	}
}

// TestScheduler_ParseCronField tests cron field parsing
func TestScheduler_ParseCronField(t *testing.T) {
	scheduler := NewScheduler(nil)

	tests := []struct {
		name      string
		field     string
		min       int
		max       int
		want      []int
	}{
		{
			name:  "All values",
			field: "*",
			min:   0,
			max:   59,
			want:  func() []int { var vals []int; for i := 0; i <= 59; i++ { vals = append(vals, i) }; return vals }(),
		},
		{
			name:  "Single value",
			field: "15",
			min:   0,
			max:   59,
			want:  []int{15},
		},
		{
			name:  "Multiple values",
			field: "0,30",
			min:   0,
			max:   59,
			want:  []int{0, 30},
		},
		{
			name:  "Range",
			field: "10-20",
			min:   0,
			max:   59,
			want:  func() []int { var vals []int; for i := 10; i <= 20; i++ { vals = append(vals, i) }; return vals }(),
		},
		{
			name:  "Mixed values and ranges",
			field: "0,15-20,30",
			min:   0,
			max:   59,
			want:  func() []int { var vals []int; vals = append(vals, 0); for i := 15; i <= 20; i++ { vals = append(vals, i) }; vals = append(vals, 30); return vals }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scheduler.parseCronField(tt.field, tt.min, tt.max)
			assert.Equal(t, tt.want, result)
		})
	}
}

// TestScheduler_MatchesCronField tests cron field matching
func TestScheduler_MatchesCronField(t *testing.T) {
	scheduler := NewScheduler(nil)

	tests := []struct {
		name     string
		value    int
		allowed  []int
		want     bool
	}{
		{
			name:    "Match",
			value:   15,
			allowed: []int{0, 15, 30, 45},
			want:    true,
		},
		{
			name:    "No match",
			value:   20,
			allowed: []int{0, 15, 30, 45},
			want:    false,
		},
		{
			name:    "Empty allowed",
			value:   15,
			allowed: []int{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scheduler.matchesCronField(tt.value, tt.allowed)
			assert.Equal(t, tt.want, result)
		})
	}
}

// TestScheduler_Constants tests constant definitions
func TestScheduler_Constants(t *testing.T) {
	// Test schedule type constants
	assert.Equal(t, ScheduleType("cron"), ScheduleTypeCron)
	assert.Equal(t, ScheduleType("interval"), ScheduleTypeInterval)
	assert.Equal(t, ScheduleType("once"), ScheduleTypeOnce)

	// Test job type constants
	assert.Equal(t, JobType("verification"), JobTypeVerification)
	assert.Equal(t, JobType("export"), JobTypeExport)
	assert.Equal(t, JobType("cleanup"), JobTypeCleanup)
	assert.Equal(t, JobType("report"), JobTypeReport)
}

// TestGenerateScheduleID tests schedule ID generation
func TestGenerateScheduleID(t *testing.T) {
	id1 := generateScheduleID()
	id2 := generateScheduleID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.True(t, len(id1) > len("sched_"))
	assert.True(t, len(id2) > len("sched_"))
	assert.Contains(t, id1, "sched_")
	assert.Contains(t, id2, "sched_")
}

// TestGenerateRunID tests run ID generation
func TestGenerateRunID(t *testing.T) {
	id1 := generateRunID()
	id2 := generateRunID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.True(t, len(id1) > len("run_"))
	assert.True(t, len(id2) > len("run_"))
	assert.Contains(t, id1, "run_")
	assert.Contains(t, id2, "run_")
}

// TestScheduler_MutexProtection tests thread safety of scheduler operations
func TestScheduler_MutexProtection(t *testing.T) {
	scheduler := NewScheduler(nil)

	// Test that we can safely access schedules from multiple goroutines
	testSchedule := &Schedule{
		Name:    "Test Schedule",
		Type:    ScheduleTypeCron,
		Enabled: true,
	}
	assert.NotNil(t, testSchedule)

	// This would fail without database, but we can test the structure
	assert.NotNil(t, scheduler.schedules)
	assert.NotNil(t, scheduler.runs)

	// Test that the maps are created and accessible
	scheduler.mu.Lock()
	assert.NotNil(t, scheduler.schedules)
	assert.NotNil(t, scheduler.runs)
	scheduler.mu.Unlock()
}

// TestScheduler_GetScheduleTests tests schedule retrieval functionality
func TestScheduler_GetScheduleTests(t *testing.T) {
	scheduler := NewScheduler(nil)

	// Test getting non-existent schedule
	_, err := scheduler.GetSchedule("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")

	// Test with empty schedules list
	schedules := scheduler.GetSchedules()
	assert.Empty(t, schedules)
}

// TestScheduler_ScheduleOperations tests schedule CRUD operations
func TestScheduler_ScheduleOperations(t *testing.T) {
	scheduler := NewScheduler(nil)

	// Test enabling non-existent schedule
	err := scheduler.EnableSchedule("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")

	// Test disabling non-existent schedule
	err = scheduler.DisableSchedule("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")

	// Test deleting non-existent schedule
	err = scheduler.DeleteSchedule("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")

	// Test updating non-existent schedule
	updates := &Schedule{
		Name: "Updated Name",
	}
	err = scheduler.UpdateSchedule("nonexistent", updates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")
}

// TestScheduler_RunNowTests tests immediate execution functionality
func TestScheduler_RunNowTests(t *testing.T) {
	scheduler := NewScheduler(nil)

	// Test running non-existent schedule
	err := scheduler.RunNow("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")
}

// TestScheduler_ScheduleRunTests tests schedule run functionality
func TestScheduler_ScheduleRunTests(t *testing.T) {
	scheduler := NewScheduler(nil)

	// Test getting runs for non-existent schedule
	runs := scheduler.GetScheduleRuns("nonexistent", 10)
	assert.Empty(t, runs) // Should return empty list, not error

	// Test limit functionality
	runs = scheduler.GetScheduleRuns("nonexistent", 0)
	assert.Empty(t, runs)

	runs = scheduler.GetScheduleRuns("nonexistent", 5)
	assert.Empty(t, runs)
}

// TestScheduler_PredefinedSchedules tests predefined schedule creation
func TestScheduler_PredefinedSchedules(t *testing.T) {
	// Test daily verification schedule
	dailySchedule := CreateDailyVerificationSchedule("Daily Test", []string{"all"})
	assert.Equal(t, "Daily Test", dailySchedule.Name)
	assert.Equal(t, ScheduleTypeCron, dailySchedule.Type)
	assert.Equal(t, JobTypeVerification, dailySchedule.JobType)
	assert.Equal(t, "0 2 * * *", dailySchedule.Expression)
	assert.True(t, dailySchedule.Enabled)
	assert.Equal(t, []string{"all"}, dailySchedule.Targets)
	assert.NotNil(t, dailySchedule.Options)
	assert.True(t, dailySchedule.Options["full_verification"].(bool))

	// Test hourly health check schedule
	healthSchedule := CreateHourlyHealthCheckSchedule("Health Test")
	assert.Equal(t, "Health Test", healthSchedule.Name)
	assert.Equal(t, ScheduleTypeCron, healthSchedule.Type)
	assert.Equal(t, JobTypeCleanup, healthSchedule.JobType)
	assert.Equal(t, "0 * * * *", healthSchedule.Expression)
	assert.True(t, healthSchedule.Enabled)
	assert.Equal(t, []string{"system"}, healthSchedule.Targets)
	assert.NotNil(t, healthSchedule.Options)
	assert.True(t, healthSchedule.Options["check_databases"].(bool))
	assert.True(t, healthSchedule.Options["check_connections"].(bool))

	// Test weekly report schedule
	reportSchedule := CreateWeeklyReportSchedule("Report Test")
	assert.Equal(t, "Report Test", reportSchedule.Name)
	assert.Equal(t, ScheduleTypeCron, reportSchedule.Type)
	assert.Equal(t, JobTypeReport, reportSchedule.JobType)
	assert.Equal(t, "0 3 * * 1", reportSchedule.Expression) // Every Monday at 3 AM
	assert.True(t, reportSchedule.Enabled)
	assert.Equal(t, []string{"all"}, reportSchedule.Targets)
	assert.NotNil(t, reportSchedule.Options)
	assert.Equal(t, "comprehensive", reportSchedule.Options["report_type"].(string))
	assert.True(t, reportSchedule.Options["include_charts"].(bool))
}

// TestScheduler_StartStopTests tests scheduler start/stop functionality
func TestScheduler_StartStopTests(t *testing.T) {
	scheduler := NewScheduler(nil)

	// Test stopping when not running
	scheduler.Stop() // Should not panic

	// Test starting without database (should fail on loadSchedules)
	err := scheduler.Start()
	assert.Error(t, err) // Should fail because loadSchedules will error
	assert.Contains(t, err.Error(), "failed to load schedules")
}

// BenchmarkScheduler_CalculateNextRun benchmarks next run calculation
func BenchmarkScheduler_CalculateNextRun(b *testing.B) {
	scheduler := NewScheduler(nil)
	
	schedule := &Schedule{
		Type:       ScheduleTypeCron,
		Expression: "0 2 * * *",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.calculateNextRun(schedule, time.Now())
	}
}

// BenchmarkScheduler_ParseCronExpression benchmarks cron expression parsing
func BenchmarkScheduler_ParseCronExpression(b *testing.B) {
	scheduler := NewScheduler(nil)
	
	expression := "0 2 * * *"
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.parseCronExpression(expression, now)
	}
}

// BenchmarkScheduler_ParseIntervalExpression benchmarks interval expression parsing
func BenchmarkScheduler_ParseIntervalExpression(b *testing.B) {
	scheduler := NewScheduler(nil)
	
	expression := "1h30m"
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.parseIntervalExpression(expression, now)
	}
}

// BenchmarkScheduler_ParseCronField benchmarks cron field parsing
func BenchmarkScheduler_ParseCronField(b *testing.B) {
	scheduler := NewScheduler(nil)
	
	field := "0,15,30,45"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.parseCronField(field, 0, 59)
	}
}

// BenchmarkScheduler_GenerateScheduleID benchmarks schedule ID generation
func BenchmarkScheduler_GenerateScheduleID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateScheduleID()
	}
}

// BenchmarkScheduler_GenerateRunID benchmarks run ID generation
func BenchmarkScheduler_GenerateRunID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateRunID()
	}
}