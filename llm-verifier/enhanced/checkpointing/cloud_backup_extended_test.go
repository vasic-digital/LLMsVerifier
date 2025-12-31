package checkpointing

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCloudBackupManager_PrefixHandling(t *testing.T) {
	mockProvider := &mockCloudProvider{
		name: "test",
		data: make(map[string][]byte),
	}

	tests := []struct {
		name           string
		inputPrefix    string
		expectedPrefix string
	}{
		{"empty prefix", "", ""},
		{"prefix without slash", "backups", "backups/"},
		{"prefix with slash", "backups/", "backups/"},
		{"nested prefix", "test/nested", "test/nested/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewCloudBackupManager(mockProvider, tt.inputPrefix)
			assert.Equal(t, tt.expectedPrefix, manager.prefix)
		})
	}
}

func TestCloudBackupManager_CleanupOldBackups(t *testing.T) {
	mockProvider := &mockCloudProvider{
		name: "test",
		data: make(map[string][]byte),
	}
	manager := NewCloudBackupManager(mockProvider, "backup/")

	ctx := context.Background()

	// Create multiple checkpoints
	for i := 1; i <= 5; i++ {
		checkpoint := &Checkpoint{
			ID:        "checkpoint-" + strconv.Itoa(i),
			AgentID:   "agent-1",
			Timestamp: time.Now(),
		}
		err := manager.BackupCheckpoint(ctx, checkpoint)
		require.NoError(t, err)
	}

	// Verify 5 checkpoints exist
	checkpoints, err := manager.ListCheckpoints(ctx, "agent-1")
	require.NoError(t, err)
	assert.Len(t, checkpoints, 5)

	// Cleanup, keeping only 2
	err = manager.CleanupOldBackups(ctx, "agent-1", 2)
	require.NoError(t, err)

	// Verify only 2 checkpoints remain
	checkpoints, err = manager.ListCheckpoints(ctx, "agent-1")
	require.NoError(t, err)
	assert.Len(t, checkpoints, 2)
}

func TestCloudBackupManager_CleanupOldBackups_NoCleanupNeeded(t *testing.T) {
	mockProvider := &mockCloudProvider{
		name: "test",
		data: make(map[string][]byte),
	}
	manager := NewCloudBackupManager(mockProvider, "backup/")

	ctx := context.Background()

	// Create 2 checkpoints
	for i := 1; i <= 2; i++ {
		checkpoint := &Checkpoint{
			ID:        "checkpoint-" + strconv.Itoa(i),
			AgentID:   "agent-1",
			Timestamp: time.Now(),
		}
		err := manager.BackupCheckpoint(ctx, checkpoint)
		require.NoError(t, err)
	}

	// Cleanup with maxBackups=5 (no cleanup needed)
	err := manager.CleanupOldBackups(ctx, "agent-1", 5)
	require.NoError(t, err)

	// Verify all 2 checkpoints still exist
	checkpoints, err := manager.ListCheckpoints(ctx, "agent-1")
	require.NoError(t, err)
	assert.Len(t, checkpoints, 2)
}

func TestCloudBackupManager_SyncCheckpoints(t *testing.T) {
	mockProvider := &mockCloudProvider{
		name: "test",
		data: make(map[string][]byte),
	}
	cloudManager := NewCloudBackupManager(mockProvider, "sync/")

	// Create local checkpoint manager
	localManager := NewCheckpointManager("/tmp/test-checkpoints", 100)

	// Create checkpoints in local manager
	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		checkpoint := &Checkpoint{
			ID:        "local-checkpoint-" + strconv.Itoa(i),
			AgentID:   "sync-agent",
			Timestamp: time.Now(),
			Progress: AgentProgress{
				TaskID:   "task-" + strconv.Itoa(i),
				TaskName: "Task " + strconv.Itoa(i),
				Status:   "completed",
			},
		}
		localManager.checkpoints[checkpoint.ID] = checkpoint
	}

	// Sync to cloud
	err := cloudManager.SyncCheckpoints(ctx, localManager)
	require.NoError(t, err)

	// Verify all checkpoints were synced
	checkpoints, err := cloudManager.ListCheckpoints(ctx, "sync-agent")
	require.NoError(t, err)
	assert.Len(t, checkpoints, 3)
}

func TestCloudBackupManager_GetBackupStats(t *testing.T) {
	mockProvider := &mockCloudProvider{
		name: "test",
		data: make(map[string][]byte),
	}
	manager := NewCloudBackupManager(mockProvider, "stats/")

	ctx := context.Background()

	// Create checkpoints for multiple agents
	for idx, agentID := range []string{"agent-1", "agent-2", "agent-1"} {
		checkpoint := &Checkpoint{
			ID:        "checkpoint-" + agentID + "-" + strconv.Itoa(idx),
			AgentID:   agentID,
			Timestamp: time.Now(),
		}
		err := manager.BackupCheckpoint(ctx, checkpoint)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := manager.GetBackupStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, "test", stats["provider"])
	assert.Equal(t, "stats/", stats["prefix"])
	assert.Equal(t, 3, stats["total_checkpoints"])

	agents := stats["agents"].([]string)
	assert.GreaterOrEqual(t, len(agents), 1)
}

func TestCloudBackupManager_GetBackupStats_Empty(t *testing.T) {
	mockProvider := &mockCloudProvider{
		name: "empty-test",
		data: make(map[string][]byte),
	}
	manager := NewCloudBackupManager(mockProvider, "empty/")

	ctx := context.Background()

	stats, err := manager.GetBackupStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, "empty-test", stats["provider"])
	assert.Equal(t, 0, stats["total_checkpoints"])
}

func TestCloudBackupManager_BackupCheckpoint_Error(t *testing.T) {
	errorProvider := &errorMockProvider{
		uploadError: errors.New("upload failed"),
	}
	manager := NewCloudBackupManager(errorProvider, "error/")

	ctx := context.Background()
	checkpoint := &Checkpoint{
		ID:        "error-checkpoint",
		AgentID:   "error-agent",
		Timestamp: time.Now(),
	}

	err := manager.BackupCheckpoint(ctx, checkpoint)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upload failed")
}

func TestCloudBackupManager_RestoreCheckpoint_Error(t *testing.T) {
	errorProvider := &errorMockProvider{
		downloadError: errors.New("download failed"),
	}
	manager := NewCloudBackupManager(errorProvider, "error/")

	ctx := context.Background()
	_, err := manager.RestoreCheckpoint(ctx, "nonexistent", "agent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download failed")
}

func TestCloudBackupManager_ListCheckpoints_AllAgents(t *testing.T) {
	mockProvider := &mockCloudProvider{
		name: "test",
		data: make(map[string][]byte),
	}
	manager := NewCloudBackupManager(mockProvider, "list/")

	ctx := context.Background()

	// Create checkpoints for multiple agents
	agents := []string{"agent-a", "agent-b", "agent-c"}
	for _, agentID := range agents {
		checkpoint := &Checkpoint{
			ID:        "checkpoint-" + agentID,
			AgentID:   agentID,
			Timestamp: time.Now(),
		}
		err := manager.BackupCheckpoint(ctx, checkpoint)
		require.NoError(t, err)
	}

	// List all checkpoints (empty agentID)
	checkpoints, err := manager.ListCheckpoints(ctx, "")
	require.NoError(t, err)
	assert.Len(t, checkpoints, 3)
}

func TestCloudBackupManager_getCheckpointKey(t *testing.T) {
	mockProvider := &mockCloudProvider{
		name: "test",
		data: make(map[string][]byte),
	}
	manager := NewCloudBackupManager(mockProvider, "keys/")

	checkpoint := &Checkpoint{
		ID:      "checkpoint-123",
		AgentID: "agent-456",
	}

	key := manager.getCheckpointKey(checkpoint)
	assert.Equal(t, "keys/agent-456/checkpoint-123.json", key)
}

func TestCloudBackupManager_getCheckpointKeyByID(t *testing.T) {
	mockProvider := &mockCloudProvider{
		name: "test",
		data: make(map[string][]byte),
	}
	manager := NewCloudBackupManager(mockProvider, "keys/")

	key := manager.getCheckpointKeyByID("checkpoint-abc", "agent-xyz")
	assert.Equal(t, "keys/agent-xyz/checkpoint-abc.json", key)
}

// errorMockProvider is a mock that returns errors for testing error paths
type errorMockProvider struct {
	uploadError   error
	downloadError error
	listError     error
	deleteError   error
}

func (e *errorMockProvider) Upload(ctx context.Context, key string, data []byte) error {
	return e.uploadError
}

func (e *errorMockProvider) Download(ctx context.Context, key string) ([]byte, error) {
	return nil, e.downloadError
}

func (e *errorMockProvider) List(ctx context.Context, prefix string) ([]string, error) {
	if e.listError != nil {
		return nil, e.listError
	}
	return []string{}, nil
}

func (e *errorMockProvider) Delete(ctx context.Context, key string) error {
	return e.deleteError
}

func (e *errorMockProvider) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (e *errorMockProvider) GetProviderName() string {
	return "error-provider"
}

func (e *errorMockProvider) HealthCheck(ctx context.Context) error {
	return nil
}
