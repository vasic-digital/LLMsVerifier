# Scheduling and Periodic Re-test Comprehensive Challenge

## Overview
This challenge validates scheduling system that allows periodic re-testing of models and providers at configurable intervals (hourly, daily, weekly, monthly).

## Challenge Type
Integration Test + Automation Test + Cron Test

## Test Scenarios

### 1. Scheduled Task Creation Challenge
**Objective**: Verify scheduled tasks can be created

**Intervals**:
- Hourly
- Daily
- Weekly
- Monthly

**Steps**:
1. Create hourly scheduled task
2. Create daily scheduled task
3. Create weekly scheduled task
4. Create monthly scheduled task
5. Verify task properties

**Expected Results**:
- All tasks created successfully
- Intervals set correctly
- Tasks persisted to database

**Test Code**:
```go
func TestScheduledTaskCreation(t *testing.T) {
    scheduler := NewScheduler()

    // Hourly
    task1, err := scheduler.CreateTask(TaskConfig{
        Name:      "Hourly Verification",
        Interval:  "hourly",
        Providers: []string{"openai"},
        Models:    []string{"gpt-4"},
    })
    assert.NoError(t, err)
    assert.Equal(t, "hourly", task1.Interval)

    // Daily
    task2, err := scheduler.CreateTask(TaskConfig{
        Name:     "Daily Verification",
        Interval: "daily",
        Time:     "02:00",
        AllModels: true,
    })
    assert.NoError(t, err)
    assert.Equal(t, "02:00", task2.Time)
}
```

---

### 2. Multiple Scheduling Configuration Challenge
**Objective**: Verify multiple scheduling configurations can coexist

**Steps**:
1. Schedule daily verification for all providers
2. Schedule hourly verification for specific provider
3. Schedule weekly verification for specific models
4. Verify all schedules active

**Expected Results**:
- Multiple schedules created
- Each schedule independent
- All schedules active

**Test Code**:
```go
func TestMultipleSchedulingConfigurations(t *testing.T) {
    scheduler := NewScheduler()

    // Daily for all
    _, _ = scheduler.CreateTask(TaskConfig{
        Name:      "Daily All",
        Interval:  "daily",
        Time:      "02:00",
        AllModels: true,
    })

    // Hourly for OpenAI
    _, _ = scheduler.CreateTask(TaskConfig{
        Name:      "Hourly OpenAI",
        Interval:  "hourly",
        Providers: []string{"openai"},
    })

    // Weekly for specific models
    _, _ = scheduler.CreateTask(TaskConfig{
        Name:     "Weekly High Priority",
        Interval: "weekly",
        Day:      "Monday",
        Time:     "03:00",
        Models:   []string{"gpt-4", "claude-3-opus"},
    })

    tasks := scheduler.ListTasks()
    assert.Equal(t, 3, len(tasks))
}
```

---

### 3. Scheduled Task Execution Challenge
**Objective**: Verify scheduled tasks execute at correct times

**Steps**:
1. Create task
2. Advance time
3. Verify task executed
4. Check results

**Expected Results**:
- Tasks execute at scheduled time
- Results recorded
- Logs updated

**Test Code**:
```go
func TestScheduledTaskExecution(t *testing.T) {
    scheduler := NewScheduler()
    executor := NewMockExecutor()

    task, _ := scheduler.CreateTask(TaskConfig{
        Name:     "Test Task",
        Interval: "daily",
        Time:     "02:00",
        AllModels: true,
    })

    // Advance time to 02:00
    scheduler.AdvanceTimeTo("02:00")

    result := <-executor.Results
    assert.Equal(t, task.ID, result.TaskID)
    assert.True(t, result.Success)
}
```

---

### 4. Task Cancellation Challenge
**Objective**: Verify tasks can be cancelled

**Steps**:
1. Create scheduled task
2. Cancel task
3. Verify task stopped
4. Verify task removed from list

**Expected Results**:
- Task cancelled
- No further executions
- Task removed

**Test Code**:
```go
func TestTaskCancellation(t *testing.T) {
    scheduler := NewScheduler()

    task, _ := scheduler.CreateTask(TaskConfig{
        Name:     "Cancellable Task",
        Interval: "hourly",
    })

    err := scheduler.CancelTask(task.ID)
    assert.NoError(t, err)

    tasks := scheduler.ListTasks()
    assert.Equal(t, 0, len(tasks))
}
```

---

### 5. Task Rescheduling Challenge
**Objective**: Verify tasks can be rescheduled

**Steps**:
1. Create task
2. Modify schedule
3. Verify new schedule applied
4. Verify next run time updated

**Expected Results**:
- Schedule modified
- Next run time updated
- Task continues with new schedule

**Test Code**:
```go
func TestTaskRescheduling(t *testing.T) {
    scheduler := NewScheduler()

    task, _ := scheduler.CreateTask(TaskConfig{
        Name:     "Reschedulable Task",
        Interval: "daily",
        Time:     "02:00",
    })

    // Reschedule to weekly
    err := scheduler.RescheduleTask(task.ID, TaskConfig{
        Name:     task.Name,
        Interval: "weekly",
        Day:      "Friday",
        Time:     "03:00",
    })
    assert.NoError(t, err)

    updated, _ := scheduler.GetTask(task.ID)
    assert.Equal(t, "weekly", updated.Interval)
    assert.Equal(t, "Friday", updated.Day)
}
```

---

### 6. Score Change Re-trigger Challenge
**Objective**: Verify re-tests triggered on score changes

**Steps**:
1. Enable regenerate_configurations_on_score_changes
2. Change model score
3. Verify re-test triggered
4. Verify configuration regenerated

**Expected Results**:
- Score change detected
- Re-test triggered
- Configuration regenerated
- Event emitted

**Test Code**:
```go
func TestScoreChangeRetrigger(t *testing.T) {
    scheduler := NewScheduler()
    scorer := NewMockScorer()

    // Enable score change re-trigger
    scheduler.SetScoreChangeTrigger(true)

    // Change score
    scorer.SetScore("gpt-4", 95, 90)

    // Wait for re-trigger
    result := <-scheduler.ReTriggerQueue
    assert.Equal(t, "score_change", result.Reason)
    assert.Equal(t, "gpt-4", result.ModelID)
}
```

---

### 7. Scheduled Task History Challenge
**Objective**: Verify task execution history is maintained

**Steps**:
1. Execute scheduled task multiple times
2. Query execution history
3. Verify records
4. Check timestamps

**Expected Results**:
- History maintained
- Records accurate
- Timestamps correct

**Test Code**:
```go
func TestScheduledTaskHistory(t *testing.T) {
    scheduler := NewScheduler()

    task, _ := scheduler.CreateTask(TaskConfig{
        Name:     "History Task",
        Interval: "hourly",
    })

    // Execute task 3 times
    for i := 0; i < 3; i++ {
        scheduler.ExecuteTask(task.ID)
    }

    history := scheduler.GetTaskHistory(task.ID)
    assert.Equal(t, 3, len(history))
}
```

---

### 8. Task Dependencies Challenge
**Objective**: Verify tasks can have dependencies

**Steps**:
1. Create task A
2. Create task B that depends on A
3. Execute A
4. Verify B executes after A

**Expected Results**:
- Dependency configured
- B waits for A
- Execution order correct

**Test Code**:
```go
func TestTaskDependencies(t *testing.T) {
    scheduler := NewScheduler()

    taskA, _ := scheduler.CreateTask(TaskConfig{Name: "Task A"})
    taskB, _ := scheduler.CreateTask(TaskConfig{
        Name:        "Task B",
        DependsOn:   []string{taskA.ID},
    })

    // Execute A
    scheduler.ExecuteTask(taskA.ID)

    // B should execute after A
    result := <-scheduler.Results
    assert.Equal(t, taskB.ID, result.TaskID)
}
```

---

### 9. Task Timezone Handling Challenge
**Objective**: Verify tasks handle timezones correctly

**Steps**:
1. Create task with timezone
2. Verify execution in correct timezone
3. Test daylight saving transitions
4. Verify DST handling

**Expected Results**:
- Timezones respected
- DST handled correctly
- Execution times accurate

**Test Code**:
```go
func TestTaskTimezoneHandling(t *testing.T) {
    scheduler := NewScheduler()

    // Create task in UTC
    task1, _ := scheduler.CreateTask(TaskConfig{
        Name:     "UTC Task",
        Interval: "daily",
        Time:     "02:00",
        Timezone: "UTC",
    })

    // Create task in EST
    task2, _ := scheduler.CreateTask(TaskConfig{
        Name:     "EST Task",
        Interval: "daily",
        Time:     "02:00",
        Timezone: "America/New_York",
    })

    // Both should execute at 02:00 in respective timezones
}
```

---

### 10. Maximal Flexibility Challenge
**Objective**: Verify scheduling system is maximally flexible

**Scenarios**:
- All providers/all models
- Specific provider/all models
- Specific provider/specific models
- Multiple providers/specific models
- Custom cron expressions

**Steps**:
1. Test all scheduling combinations
2. Verify flexibility
3. Test custom cron
4. Verify complex schedules

**Expected Results**:
- All combinations work
- Custom cron expressions supported
- Maximum flexibility achieved

**Test Code**:
```go
func TestMaximalFlexibility(t *testing.T) {
    scheduler := NewScheduler()

    // Custom cron expression
    task, err := scheduler.CreateTask(TaskConfig{
        Name:     "Custom Cron Task",
        Schedule: "0 2 * * 1", // Every Monday at 2 AM
    })
    assert.NoError(t, err)

    assert.Equal(t, "0 2 * * 1", task.CronExpression)
}
```

---

## Success Criteria

### Functional Requirements
- [ ] Tasks created successfully
- [ ] Multiple schedules coexist
- [ ] Tasks execute at correct times
- [ ] Tasks can be cancelled
- [ ] Tasks can be rescheduled
- [ ] Score changes trigger re-tests
- [ ] History maintained
- [ ] Dependencies work
- [ ] Timezones handled
- [ ] Maximum flexibility achieved

### Accuracy Requirements
- [ ] Execution time accuracy within 1 second
- [ ] Schedule accuracy 100%
- [ ] History accuracy 100%

### Performance Requirements
- [ ] Task creation < 100ms
- [ ] Task cancellation < 50ms
- [ ] Can handle 1000 concurrent tasks

## Dependencies
- Scheduler service running
- Database initialized

## Cleanup
- Remove test tasks
- Clear history
