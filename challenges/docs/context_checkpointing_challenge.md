# Context Management and Checkpointing Comprehensive Challenge

## Overview
This challenge validates context management, summarization, long-term memory (Cognee), and checkpointing system from OPTIMIZATIONS.md.

## Challenge Type
Integration Test + State Management Test + Memory Test

## Test Scenarios

### 1. Short-Term Context Challenge
**Objective**: Verify short-term context management (6-10 messages)

**Steps**:
1. Start conversation
2. Add 6-10 messages
3. Verify messages kept verbatim
4. Add 11th message
5. Verify oldest removed (sliding window)
6. Verify recent messages retained

**Expected Results**:
- 6-10 messages kept
- Sliding window works
- Recent messages verbatim
- Oldest removed at threshold

**Test Code**:
```go
func TestShortTermContext(t *testing.T) {
    ctx := NewContextManager(ContextConfig{
        MaxMessages: 10,
    })

    // Add 10 messages
    for i := 0; i < 10; i++ {
        ctx.AddMessage(User, fmt.Sprintf("Message %d", i))
    }

    messages := ctx.GetRecentMessages(10)
    assert.Equal(t, 10, len(messages))
    assert.Equal(t, "Message 0", messages[0].Content)

    // Add 11th message
    ctx.AddMessage(User, "Message 10")

    // Oldest should be removed
    messages = ctx.GetRecentMessages(10)
    assert.Equal(t, 10, len(messages))
    assert.Equal(t, "Message 1", messages[0].Content)
}
```

---

### 2. Conversation Summarization Challenge
**Objective**: Verify summarization every 8-12 turns

**Steps**:
1. Add 10 messages to conversation
2. Trigger summarization
3. Verify summary generated
4. Verify only recent messages kept
5. Verify summary reference added

**Expected Results**:
- Summary generated at threshold
- Summary stored in memory
- Recent messages kept
- Summary reference in context

**Test Code**:
```go
func TestConversationSummarization(t *testing.T) {
    ctx := NewContextManager(ContextConfig{
        SummaryThreshold: 10,
    })

    // Add 10 messages
    for i := 0; i < 10; i++ {
        ctx.AddMessage(User, fmt.Sprintf("Message %d", i))
    }

    // Should trigger summarization
    summary, err := ctx.GenerateSummary()
    assert.NoError(t, err)
    assert.NotEmpty(t, summary.Content)

    // Only recent messages + summary should remain
    messages := ctx.GetRecentMessages(20)
    assert.Equal(t, 6, len(messages)) // 5 recent + 1 summary
    assert.Contains(t, messages[0].Content, "SUMMARY OF EARLIER CONVERSATION")
}
```

---

### 3. Long-Term Memory Integration Challenge
**Objective**: Verify Cognee/vector database integration

**Steps**:
1. Generate summary
2. Store in Cognee
3. Retrieve relevant context on-demand
4. Verify retrieved context relevant

**Expected Results**:
- Summary stored in vector DB
- Retrieval returns relevant summaries
- Context not full history
- On-demand retrieval works

**Test Code**:
```go
func TestLongTermMemoryIntegration(t *testing.T) {
    memory := NewCogneeMemory("mongodb://localhost", "redis://localhost")
    ctx := NewContextManager(ContextConfig{
        MemoryClient: memory,
    })

    // Generate and store summary
    summary, _ := ctx.GenerateSummary()
    err := memory.StoreSummary(summary)
    assert.NoError(t, err)

    // Retrieve relevant context
    relevant, err := memory.RetrieveRelevantContext("What was decided about feature X?")
    assert.NoError(t, err)
    assert.Greater(t, len(relevant), 0)
}
```

---

### 4. Context Trimming Challenge
**Objective**: Verify context trimming for token limits

**Steps**:
1. Set max tokens to 8000
2. Add messages that exceed limit
3. Verify trimming
4. Verify most recent retained
5. Verify summary included if available

**Expected Results**:
- Context trimmed to limit
- Most recent retained
- Low-value messages removed
- Summaries used to preserve context

**Test Code**:
```go
func TestContextTrimming(t *testing.T) {
    ctx := NewContextManager(ContextConfig{
        MaxTokens: 8000,
    })

    // Add messages exceeding limit
    for i := 0; i < 100; i++ {
        ctx.AddMessage(User, strings.Repeat("Hello ", 200))
    }

    trimmed := ctx.TrimToTokenLimit(8000)
    tokenCount := ctx.CountTokens(trimmed)
    assert.LessOrEqual(t, tokenCount, 8000)

    // Most recent should be preserved
    assert.Equal(t, trimmed[len(trimmed)-1].Content, strings.Repeat("Hello ", 200))
}
```

---

### 5. Checkpoint Creation Challenge
**Objective**: Verify checkpoint creation

**Checkpoint Contents**:
- Agent progress (task ID, step index)
- Memory snapshot (Cognee pointers)
- Open files (filenames, cursor positions)
- Timestamp and provider used

**Steps**:
1. Create agent state
2. Create checkpoint
3. Verify saved to database
4. Verify saved to S3 backup
5. Verify checkpoint ID

**Expected Results**:
- Checkpoint created
- State saved
- Backup created
- Checkpoint ID returned

**Test Code**:
```go
func TestCheckpointCreation(t *testing.T) {
    manager := NewCheckpointManager(CheckpointConfig{
        DB:      testDB,
        S3Bucket: "test-bucket",
    })

    state := AgentState{
        TaskID:    "task-001",
        StepIndex: 5,
        LastOutputHash: "abc123",
    }

    checkpointID, err := manager.CreateCheckpoint(state, "Step 5 completed")
    assert.NoError(t, err)
    assert.NotEmpty(t, checkpointID)

    // Verify in database
    dbCheckpoint, _ := testDB.GetCheckpoint(checkpointID)
    assert.Equal(t, "task-001", dbCheckpoint.TaskID)
}
```

---

### 6. Checkpoint Frequency Challenge
**Objective**: Verify checkpointing at appropriate intervals

**Frequencies**:
- Short tasks: After each step
- Long tasks: Every 5-15 minutes
- Critical operations: Before and after

**Steps**:
1. Start long-running task
2. Create checkpoint every 10 minutes
3. Verify checkpoint frequency
4. Verify checkpoints include milestones

**Expected Results**:
- Checkpoints created periodically
- Milestones recorded
- All steps covered

**Test Code**:
```go
func TestCheckpointFrequency(t *testing.T) {
    manager := NewCheckpointManager(CheckpointConfig{
        Interval: 10 * time.Minute,
    })

    manager.StartAutoCheckpointing("task-001")

    // Simulate 30 minutes of work
    for i := 0; i < 3; i++ {
        time.Sleep(10 * time.Millisecond) // Faster for test
    }

    manager.StopAutoCheckpointing()

    checkpoints := manager.ListCheckpoints("task-001")
    assert.Equal(t, 3, len(checkpoints))
}
```

---

### 7. Checkpoint Restore Challenge
**Objective**: Verify restoration from checkpoint

**Steps**:
1. Create checkpoint
2. Modify agent state
3. Restore from checkpoint
4. Verify state matches
5. Verify S3 backup used if needed

**Expected Results**:
- State restored exactly
- Progress resumed
- Files reopened at positions
- Memory pointers restored

**Test Code**:
```go
func TestCheckpointRestore(t *testing.T) {
    manager := NewCheckpointManager(CheckpointConfig{})

    // Create checkpoint
    state := AgentState{TaskID: "task-001", StepIndex: 5}
    checkpointID, _ := manager.CreateCheckpoint(state, "")

    // Modify state
    state.StepIndex = 10

    // Restore
    restored, err := manager.RestoreFromCheckpoint("task-001")
    assert.NoError(t, err)
    assert.Equal(t, 5, restored.StepIndex)
}
```

---

### 8. Disaster Recovery Challenge
**Objective**: Verify S3 backup and disaster recovery

**Steps**:
1. Create checkpoint
2. Verify in database
3. Verify in S3
4. Simulate database loss
5. Restore from S3
6. Verify recovery

**Expected Results**:
- Backup in S3
- Recovery possible
- State identical
- No data loss

**Test Code**:
```go
func TestDisasterRecovery(t *testing.T) {
    manager := NewCheckpointManager(CheckpointConfig{
        DB:      testDB,
        S3Bucket: "test-bucket",
    })

    // Create checkpoint
    state := AgentState{TaskID: "task-001", StepIndex: 5}
    checkpointID, _ := manager.CreateCheckpoint(state, "")

    // Simulate database loss
    testDB.Drop()

    // Restore from S3
    manager = NewCheckpointManager(CheckpointConfig{
        S3Bucket: "test-bucket",
    })
    restored, err := manager.RestoreFromS3("task-001", checkpointID)
    assert.NoError(t, err)
    assert.Equal(t, 5, restored.StepIndex)
}
```

---

### 9. Memory Summarization Challenge
**Objective**: Verify memory is summarized correctly

**Steps**:
1. Add complex conversation
2. Generate summary
3. Verify key facts preserved
4. Verify decisions recorded
5. Verify summary concise (2-4 sentences)

**Expected Results**:
- Key facts preserved
- Decisions recorded
- Summary concise
- Context sufficient for continuation

**Test Code**:
```go
func TestMemorySummarization(t *testing.T) {
    ctx := NewContextManager(ContextConfig{})

    // Add complex conversation
    ctx.AddMessage(User, "We need to implement feature X")
    ctx.AddMessage(Assistant, "I'll start by creating the base class")
    ctx.AddMessage(User, "Good, also ensure it supports plugin architecture")
    ctx.AddMessage(Assistant, "Understood, I'll add plugin support")

    summary, _ := ctx.GenerateSummary()

    // Should contain key decisions
    assert.Contains(t, summary.Content, "feature X")
    assert.Contains(t, summary.Content, "plugin architecture")

    // Should be concise
    sentences := strings.Split(summary.Content, ".")
    assert.LessOrEqual(t, len(sentences), 4)
}
```

---

### 10. Checkpoint Cleanup Challenge
**Objective**: Verify old checkpoints are cleaned up

**Steps**:
1. Create multiple checkpoints
2. Configure retention policy
3. Trigger cleanup
4. Verify old checkpoints removed
5. Verify recent checkpoints kept

**Expected Results**:
- Old checkpoints removed
- Recent kept
- Retention policy enforced
- Space reclaimed

**Test Code**:
```go
func TestCheckpointCleanup(t *testing.T) {
    manager := NewCheckpointManager(CheckpointConfig{
        RetentionDays: 7,
    })

    // Create checkpoints at different times
    now := time.Now()
    manager.CreateCheckpoint(AgentState{}, "Old 1", now.Add(-10*24*time.Hour))
    manager.CreateCheckpoint(AgentState{}, "Old 2", now.Add(-8*24*time.Hour))
    manager.CreateCheckpoint(AgentState{}, "Recent", now.Add(-1*24*time.Hour))

    // Cleanup
    manager.CleanupOldCheckpoints()

    checkpoints := manager.ListCheckpoints("task-001")
    assert.Equal(t, 1, len(checkpoints)) // Only recent
}
```

---

## Success Criteria

### Functional Requirements
- [ ] Short-term context works
- [ ] Summarization works
- [ ] Long-term memory works
- [ ] Context trimming works
- [ ] Checkpoint creation works
- [ ] Checkpoint frequency works
- [ ] Restore works
- [ ] Disaster recovery works
- [ ] Memory summarization works
- [ ] Cleanup works

### Reliability Requirements
- [ ] No context loss
- [ ] Checkpoints consistent
- [ ] Restores accurate
- [ ] S3 backup successful

### Performance Requirements
- [ ] Context operations < 10ms
- [ ] Summarization < 5 seconds
- [ ] Checkpoint creation < 1 second
- [ ] Restore < 2 seconds

## Dependencies
- Cognee/vector DB
- S3 or MinIO
- Database

## Cleanup
- Remove checkpoints
- Clear memory
- Delete S3 objects
