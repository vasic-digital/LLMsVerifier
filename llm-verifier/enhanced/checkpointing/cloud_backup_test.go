package checkpointing

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCloudBackupManager(t *testing.T) {
	// Create a mock provider for testing
	mockProvider := &mockCloudProvider{
		name: "Mock Provider",
		data: make(map[string][]byte),
	}

	manager := NewCloudBackupManager(mockProvider, "test/")

	ctx := context.Background()

	t.Run("BackupCheckpoint", func(t *testing.T) {
		checkpoint := &Checkpoint{
			ID:        "test-checkpoint",
			AgentID:   "test-agent",
			Timestamp: time.Now(),
			Progress: AgentProgress{
				TaskID:   "test-task",
				TaskName: "Test Task",
				Status:   "running",
			},
		}

		err := manager.BackupCheckpoint(ctx, checkpoint)
		if err != nil {
			t.Errorf("BackupCheckpoint failed: %v", err)
		}

		// Verify data was stored
		key := "test/test-agent/test-checkpoint.json"
		if _, exists := mockProvider.data[key]; !exists {
			t.Errorf("Checkpoint data not found in mock provider")
		}
	})

	t.Run("RestoreCheckpoint", func(t *testing.T) {
		restored, err := manager.RestoreCheckpoint(ctx, "test-checkpoint", "test-agent")
		if err != nil {
			t.Errorf("RestoreCheckpoint failed: %v", err)
		}

		if restored.ID != "test-checkpoint" {
			t.Errorf("Restored checkpoint ID mismatch: got %s, want %s", restored.ID, "test-checkpoint")
		}

		if restored.AgentID != "test-agent" {
			t.Errorf("Restored checkpoint agent ID mismatch: got %s, want %s", restored.AgentID, "test-agent")
		}
	})

	t.Run("ListCheckpoints", func(t *testing.T) {
		checkpoints, err := manager.ListCheckpoints(ctx, "test-agent")
		if err != nil {
			t.Errorf("ListCheckpoints failed: %v", err)
		}

		if len(checkpoints) != 1 {
			t.Errorf("Expected 1 checkpoint, got %d", len(checkpoints))
		}

		if checkpoints[0].ID != "test-checkpoint" {
			t.Errorf("Checkpoint ID mismatch: got %s, want %s", checkpoints[0].ID, "test-checkpoint")
		}
	})

	t.Run("DeleteCheckpoint", func(t *testing.T) {
		err := manager.DeleteCheckpoint(ctx, "test-checkpoint", "test-agent")
		if err != nil {
			t.Errorf("DeleteCheckpoint failed: %v", err)
		}

		// Verify data was deleted
		key := "test/test-agent/test-checkpoint.json"
		if _, exists := mockProvider.data[key]; exists {
			t.Errorf("Checkpoint data still exists after deletion")
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		err := manager.HealthCheck(ctx)
		if err != nil {
			t.Errorf("HealthCheck failed: %v", err)
		}
	})
}

// mockCloudProvider implements CloudBackupProvider for testing
type mockCloudProvider struct {
	name string
	data map[string][]byte
}

func (m *mockCloudProvider) Upload(ctx context.Context, key string, data []byte) error {
	m.data[key] = make([]byte, len(data))
	copy(m.data[key], data)
	return nil
}

func (m *mockCloudProvider) Download(ctx context.Context, key string) ([]byte, error) {
	data, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

func (m *mockCloudProvider) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	for key := range m.data {
		if len(prefix) == 0 || strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (m *mockCloudProvider) Delete(ctx context.Context, key string) error {
	if _, exists := m.data[key]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}
	delete(m.data, key)
	return nil
}

func (m *mockCloudProvider) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := m.data[key]
	return exists, nil
}

func (m *mockCloudProvider) GetProviderName() string {
	return m.name
}

func (m *mockCloudProvider) HealthCheck(ctx context.Context) error {
	return nil
}
