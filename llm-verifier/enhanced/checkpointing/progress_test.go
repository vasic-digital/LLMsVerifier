package checkpointing

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== CheckpointCache Tests ====================

func TestNewCheckpointCache(t *testing.T) {
	cache := NewCheckpointCache(10)
	assert.NotNil(t, cache)
	assert.Equal(t, 10, cache.maxSize)
}

func TestCheckpointCache_GetPut(t *testing.T) {
	cache := NewCheckpointCache(10)

	checkpoint := &Checkpoint{
		ID:        "test-id",
		AgentID:   "agent-1",
		Timestamp: time.Now(),
	}

	// Get non-existent
	_, exists := cache.Get("test-id")
	assert.False(t, exists)

	// Put and get
	cache.Put(checkpoint)
	retrieved, exists := cache.Get("test-id")
	assert.True(t, exists)
	assert.Equal(t, checkpoint.ID, retrieved.ID)
}

func TestCheckpointCache_Eviction(t *testing.T) {
	cache := NewCheckpointCache(2)

	cache.Put(&Checkpoint{ID: "chk-1"})
	cache.Put(&Checkpoint{ID: "chk-2"})
	cache.Put(&Checkpoint{ID: "chk-3"}) // Should evict one

	// At least one should be evicted
	count := 0
	if _, exists := cache.Get("chk-1"); exists {
		count++
	}
	if _, exists := cache.Get("chk-2"); exists {
		count++
	}
	if _, exists := cache.Get("chk-3"); exists {
		count++
	}
	assert.LessOrEqual(t, count, 2)
}

func TestCheckpointCache_Clear(t *testing.T) {
	cache := NewCheckpointCache(10)

	cache.Put(&Checkpoint{ID: "chk-1"})
	cache.Put(&Checkpoint{ID: "chk-2"})

	cache.Clear()

	_, exists := cache.Get("chk-1")
	assert.False(t, exists)
}

// ==================== CheckpointManager Tests ====================

func setupTestManager(t *testing.T) (*CheckpointManager, string) {
	tmpDir, err := os.MkdirTemp("", "checkpoint_test")
	require.NoError(t, err)

	manager := NewCheckpointManager(tmpDir, 5)
	return manager, tmpDir
}

func cleanupTestManager(t *testing.T, tmpDir string) {
	os.RemoveAll(tmpDir)
}

func TestNewCheckpointManager(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	assert.NotNil(t, manager)
	assert.Equal(t, 5, manager.maxCheckpoints)
	assert.NotNil(t, manager.cache)
}

func TestCheckpointManager_CreateCheckpoint(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	progress := AgentProgress{
		TaskID:         "task-1",
		TaskName:       "Test Task",
		Status:         "running",
		Progress:       0.5,
		Step:           "step 1",
		TotalSteps:     10,
		CompletedSteps: 5,
		StartTime:      time.Now(),
		LastUpdate:     time.Now(),
	}

	memoryState := MemoryState{
		WorkingMemory: map[string]interface{}{"key": "value"},
	}

	openFiles := []OpenFile{
		{Path: "/test/file.go", Content: "package main", IsModified: true},
	}

	checkpoint, err := manager.CreateCheckpoint("agent-1", progress, memoryState, openFiles)
	require.NoError(t, err)
	assert.NotEmpty(t, checkpoint.ID)
	assert.Equal(t, "agent-1", checkpoint.AgentID)
	assert.Equal(t, "running", checkpoint.Progress.Status)

	// Verify file was created
	expectedFile := filepath.Join(tmpDir, checkpoint.ID+".json")
	_, err = os.Stat(expectedFile)
	assert.NoError(t, err)
}

func TestCheckpointManager_GetLatestCheckpoint(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	// Create multiple checkpoints
	progress := AgentProgress{Status: "running"}

	_, err := manager.CreateCheckpoint("agent-1", progress, MemoryState{}, nil)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	secondProgress := AgentProgress{Status: "completed"}
	second, err := manager.CreateCheckpoint("agent-1", secondProgress, MemoryState{}, nil)
	require.NoError(t, err)

	// Get latest
	latest, err := manager.GetLatestCheckpoint("agent-1")
	require.NoError(t, err)
	assert.Equal(t, second.ID, latest.ID)
	assert.Equal(t, "completed", latest.Progress.Status)
}

func TestCheckpointManager_GetLatestCheckpointNotFound(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	_, err := manager.GetLatestCheckpoint("nonexistent-agent")
	assert.Error(t, err)
}

func TestCheckpointManager_GetCheckpointsForAgent(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	progress := AgentProgress{Status: "running"}

	manager.CreateCheckpoint("agent-1", progress, MemoryState{}, nil)
	manager.CreateCheckpoint("agent-1", progress, MemoryState{}, nil)
	manager.CreateCheckpoint("agent-2", progress, MemoryState{}, nil)

	agent1Checkpoints := manager.GetCheckpointsForAgent("agent-1")
	assert.Len(t, agent1Checkpoints, 2)

	agent2Checkpoints := manager.GetCheckpointsForAgent("agent-2")
	assert.Len(t, agent2Checkpoints, 1)
}

func TestCheckpointManager_RestoreFromCheckpoint(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	progress := AgentProgress{Status: "running", TaskID: "task-1"}
	created, err := manager.CreateCheckpoint("agent-1", progress, MemoryState{}, nil)
	require.NoError(t, err)

	restored, err := manager.RestoreFromCheckpoint(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, restored.ID)
	assert.Equal(t, "task-1", restored.Progress.TaskID)
}

func TestCheckpointManager_RestoreFromCheckpointNotFound(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	_, err := manager.RestoreFromCheckpoint("nonexistent")
	assert.Error(t, err)
}

func TestCheckpointManager_UpdateProgress(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	progress := AgentProgress{Status: "running", Progress: 0.5}
	_, err := manager.CreateCheckpoint("agent-1", progress, MemoryState{}, nil)
	require.NoError(t, err)

	newProgress := AgentProgress{Status: "completed", Progress: 1.0}
	err = manager.UpdateProgress("agent-1", newProgress)
	require.NoError(t, err)

	latest, err := manager.GetLatestCheckpoint("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "completed", latest.Progress.Status)
	assert.Equal(t, 1.0, latest.Progress.Progress)
}

func TestCheckpointManager_UpdateProgressNotFound(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	err := manager.UpdateProgress("nonexistent", AgentProgress{})
	assert.Error(t, err)
}

func TestCheckpointManager_AddOpenFile(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	_, err := manager.CreateCheckpoint("agent-1", AgentProgress{}, MemoryState{}, nil)
	require.NoError(t, err)

	err = manager.AddOpenFile("agent-1", "/test/file.go", "package main")
	require.NoError(t, err)

	latest, err := manager.GetLatestCheckpoint("agent-1")
	require.NoError(t, err)
	assert.Len(t, latest.OpenFiles, 1)
	assert.Equal(t, "/test/file.go", latest.OpenFiles[0].Path)
}

func TestCheckpointManager_AddOpenFileUpdate(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	openFiles := []OpenFile{
		{Path: "/test/file.go", Content: "package main"},
	}
	_, err := manager.CreateCheckpoint("agent-1", AgentProgress{}, MemoryState{}, openFiles)
	require.NoError(t, err)

	// Update existing file
	err = manager.AddOpenFile("agent-1", "/test/file.go", "package main\n\nfunc main() {}")
	require.NoError(t, err)

	latest, err := manager.GetLatestCheckpoint("agent-1")
	require.NoError(t, err)
	assert.Len(t, latest.OpenFiles, 1) // Should not add duplicate
	assert.Contains(t, latest.OpenFiles[0].Content, "func main()")
}

func TestCheckpointManager_AddOpenFileNotFound(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	err := manager.AddOpenFile("nonexistent", "/test/file.go", "content")
	assert.Error(t, err)
}

func TestCheckpointManager_RemoveOpenFile(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	openFiles := []OpenFile{
		{Path: "/test/file1.go", Content: "package main"},
		{Path: "/test/file2.go", Content: "package test"},
	}
	_, err := manager.CreateCheckpoint("agent-1", AgentProgress{}, MemoryState{}, openFiles)
	require.NoError(t, err)

	err = manager.RemoveOpenFile("agent-1", "/test/file1.go")
	require.NoError(t, err)

	latest, err := manager.GetLatestCheckpoint("agent-1")
	require.NoError(t, err)
	assert.Len(t, latest.OpenFiles, 1)
	assert.Equal(t, "/test/file2.go", latest.OpenFiles[0].Path)
}

func TestCheckpointManager_RemoveOpenFileNotFound(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	_, err := manager.CreateCheckpoint("agent-1", AgentProgress{}, MemoryState{}, nil)
	require.NoError(t, err)

	err = manager.RemoveOpenFile("agent-1", "/nonexistent/file.go")
	assert.Error(t, err)
}

func TestCheckpointManager_RemoveOpenFileNoCheckpoint(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	err := manager.RemoveOpenFile("nonexistent", "/test/file.go")
	assert.Error(t, err)
}

func TestCheckpointManager_GetCheckpointStats(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	manager.CreateCheckpoint("agent-1", AgentProgress{}, MemoryState{}, nil)
	manager.CreateCheckpoint("agent-1", AgentProgress{}, MemoryState{}, nil)
	manager.CreateCheckpoint("agent-2", AgentProgress{}, MemoryState{}, nil)

	stats := manager.GetCheckpointStats()

	assert.Equal(t, 3, stats["total_checkpoints"])
	assert.Equal(t, tmpDir, stats["storage_path"])
	assert.Equal(t, 5, stats["max_checkpoints"])

	agentCounts := stats["checkpoints_per_agent"].(map[string]int)
	assert.Equal(t, 2, agentCounts["agent-1"])
	assert.Equal(t, 1, agentCounts["agent-2"])
}

func TestCheckpointManager_ClearCache(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	// Create and access to cache
	created, _ := manager.CreateCheckpoint("agent-1", AgentProgress{}, MemoryState{}, nil)
	manager.GetLatestCheckpoint("agent-1")

	// Clear cache
	manager.ClearCache()

	// Verify cache is cleared by checking cache size
	stats := manager.GetCheckpointStats()
	assert.Equal(t, 0, stats["cache_size"])

	// Checkpoint should still be accessible from disk/memory
	restored, err := manager.RestoreFromCheckpoint(created.ID)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, restored.ID)
}

func TestCheckpointManager_SetCacheSize(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer cleanupTestManager(t, tmpDir)

	manager.SetCacheSize(50)
	assert.Equal(t, 50, manager.cache.maxSize)
}

func TestCheckpointManager_CleanupOldCheckpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpoint_cleanup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager := NewCheckpointManager(tmpDir, 2) // Max 2 checkpoints

	// Create 4 checkpoints
	for i := 0; i < 4; i++ {
		manager.CreateCheckpoint("agent-1", AgentProgress{}, MemoryState{}, nil)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Should only have 2 checkpoints
	checkpoints := manager.GetCheckpointsForAgent("agent-1")
	assert.Len(t, checkpoints, 2)
}

// ==================== Struct Tests ====================

func TestAgentProgress_Struct(t *testing.T) {
	now := time.Now()
	progress := AgentProgress{
		TaskID:         "task-1",
		TaskName:       "Test Task",
		Status:         "running",
		Progress:       0.75,
		Step:           "Processing",
		TotalSteps:     4,
		CompletedSteps: 3,
		StartTime:      now,
		LastUpdate:     now,
		Error:          "",
	}

	assert.Equal(t, "task-1", progress.TaskID)
	assert.Equal(t, 0.75, progress.Progress)
	assert.Equal(t, 3, progress.CompletedSteps)
}

func TestOpenFile_Struct(t *testing.T) {
	now := time.Now()
	file := OpenFile{
		Path:         "/test/file.go",
		Content:      "package main",
		CursorPos:    10,
		LastModified: now,
		IsModified:   true,
	}

	assert.Equal(t, "/test/file.go", file.Path)
	assert.Equal(t, 10, file.CursorPos)
	assert.True(t, file.IsModified)
}

func TestCheckpoint_Struct(t *testing.T) {
	now := time.Now()
	checkpoint := Checkpoint{
		ID:        "chk-1",
		AgentID:   "agent-1",
		Timestamp: now,
		Progress: AgentProgress{
			Status: "running",
		},
		MemoryState: MemoryState{
			WorkingMemory: map[string]interface{}{"key": "value"},
		},
		OpenFiles: []OpenFile{
			{Path: "/test.go"},
		},
		Metadata: map[string]interface{}{"meta": "data"},
	}

	assert.Equal(t, "chk-1", checkpoint.ID)
	assert.Equal(t, "running", checkpoint.Progress.Status)
	assert.Equal(t, "value", checkpoint.MemoryState.WorkingMemory["key"])
	assert.Len(t, checkpoint.OpenFiles, 1)
}
