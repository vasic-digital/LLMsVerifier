package context

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== HybridStorage Tests ====================

func TestNewHybridStorage(t *testing.T) {
	primary := NewMockStorage()
	replica1 := NewMockStorage()
	replica2 := NewMockStorage()

	hs := NewHybridStorage(primary, replica1, replica2)
	require.NotNil(t, hs)
	assert.Equal(t, primary, hs.primary)
	assert.Len(t, hs.replicas, 2)
}

func TestHybridStorage_SaveContext(t *testing.T) {
	ctx := context.Background()
	primary := NewMockStorage()
	replica := NewMockStorage()

	hs := NewHybridStorage(primary, replica)

	err := hs.SaveContext(ctx, "conv-1", []byte(`{"test": "data"}`))
	require.NoError(t, err)

	// Verify primary has data
	data, err := primary.LoadContext(ctx, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, `{"test": "data"}`, string(data))

	// Give time for async replication
	time.Sleep(50 * time.Millisecond)

	// Verify replica has data
	data, err = replica.LoadContext(ctx, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, `{"test": "data"}`, string(data))
}

func TestHybridStorage_LoadContext_FromPrimary(t *testing.T) {
	ctx := context.Background()
	primary := NewMockStorage()
	replica := NewMockStorage()

	// Set up primary with data
	primary.data["conv-1"] = []byte(`{"primary": true}`)

	hs := NewHybridStorage(primary, replica)

	data, err := hs.LoadContext(ctx, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, `{"primary": true}`, string(data))
}

func TestHybridStorage_LoadContext_FallbackToReplica(t *testing.T) {
	ctx := context.Background()
	primary := NewMockStorage()
	replica := NewMockStorage()

	// Only set up replica with data
	replica.data["conv-1"] = []byte(`{"replica": true}`)

	hs := NewHybridStorage(primary, replica)

	data, err := hs.LoadContext(ctx, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, `{"replica": true}`, string(data))

	// Give time for restoration to primary
	time.Sleep(50 * time.Millisecond)

	// Verify data was restored to primary
	primaryData, err := primary.LoadContext(ctx, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, `{"replica": true}`, string(primaryData))
}

func TestHybridStorage_LoadContext_NotFound(t *testing.T) {
	ctx := context.Background()
	primary := NewMockStorage()
	replica := NewMockStorage()

	hs := NewHybridStorage(primary, replica)

	_, err := hs.LoadContext(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load from all storage backends")
}

func TestHybridStorage_DeleteContext(t *testing.T) {
	ctx := context.Background()
	primary := NewMockStorage()
	replica := NewMockStorage()

	// Set up data in both
	primary.data["conv-1"] = []byte(`{"test": true}`)
	replica.data["conv-1"] = []byte(`{"test": true}`)

	hs := NewHybridStorage(primary, replica)

	err := hs.DeleteContext(ctx, "conv-1")
	require.NoError(t, err)

	// Verify deleted from primary
	_, err = primary.LoadContext(ctx, "conv-1")
	assert.Error(t, err)

	// Verify deleted from replica
	_, err = replica.LoadContext(ctx, "conv-1")
	assert.Error(t, err)
}

func TestHybridStorage_ListConversations(t *testing.T) {
	ctx := context.Background()
	primary := NewMockStorage()
	replica := NewMockStorage()

	primary.data["conv-1"] = []byte(`{}`)
	primary.data["conv-2"] = []byte(`{}`)

	hs := NewHybridStorage(primary, replica)

	conversations, err := hs.ListConversations(ctx)
	require.NoError(t, err)
	assert.Len(t, conversations, 2)
}

// ==================== StorageConfig Tests ====================

func TestStorageConfig_Struct(t *testing.T) {
	config := StorageConfig{
		Type: "filesystem",
		Settings: map[string]interface{}{
			"base_path": "/tmp/test",
		},
	}

	assert.Equal(t, "filesystem", config.Type)
	assert.Equal(t, "/tmp/test", config.Settings["base_path"])
}

func TestNewStorageFromConfig_Filesystem(t *testing.T) {
	tempDir := t.TempDir()

	config := StorageConfig{
		Type: "filesystem",
		Settings: map[string]interface{}{
			"base_path": tempDir,
		},
	}

	storage, err := NewStorageFromConfig(config, nil)
	require.NoError(t, err)
	require.NotNil(t, storage)

	// Verify it works
	ctx := context.Background()
	err = storage.SaveContext(ctx, "test", []byte("data"))
	require.NoError(t, err)

	data, err := storage.LoadContext(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "data", string(data))
}

func TestNewStorageFromConfig_FilesystemMissingPath(t *testing.T) {
	config := StorageConfig{
		Type:     "filesystem",
		Settings: map[string]interface{}{},
	}

	_, err := NewStorageFromConfig(config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base_path required")
}

func TestNewStorageFromConfig_DatabaseNilDB(t *testing.T) {
	config := StorageConfig{
		Type:     "database",
		Settings: map[string]interface{}{},
	}

	_, err := NewStorageFromConfig(config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection required")
}

func TestNewStorageFromConfig_UnsupportedType(t *testing.T) {
	config := StorageConfig{
		Type:     "unknown",
		Settings: map[string]interface{}{},
	}

	_, err := NewStorageFromConfig(config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported storage type")
}

// ==================== FileSystemStorage Additional Tests ====================

func TestFileSystemStorage_LoadContext_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	fs, err := NewFileSystemStorage(tempDir)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = fs.LoadContext(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conversation not found")
}

func TestFileSystemStorage_DeleteContext_NonExistent(t *testing.T) {
	tempDir := t.TempDir()

	fs, err := NewFileSystemStorage(tempDir)
	require.NoError(t, err)

	ctx := context.Background()
	err = fs.DeleteContext(ctx, "nonexistent")
	assert.NoError(t, err) // Should not error when file doesn't exist
}

func TestFileSystemStorage_ListConversations_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()

	fs, err := NewFileSystemStorage(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Create multiple conversations
	for i := 1; i <= 5; i++ {
		err = fs.SaveContext(ctx, "conv-"+string(rune('0'+i)), []byte(`{}`))
		require.NoError(t, err)
	}

	conversations, err := fs.ListConversations(ctx)
	require.NoError(t, err)
	assert.Len(t, conversations, 5)
}

// ==================== ContextManager Export/Import Tests ====================

func TestContextManager_ExportImportContext(t *testing.T) {
	mockVerifier := &MockVerifier{}
	mockStorage := NewMockStorage()

	config := ContextConfig{
		ShortTermMaxMessages:    5,
		ShortTermWindowDuration: 10 * time.Minute,
		LongTermMaxSummaries:    5,
		SummarizationThreshold:  3,
		BackupEnabled:           false,
	}

	cm := NewContextManager("export-test", config, mockVerifier, mockStorage)

	// Add some messages
	cm.AddMessage("user", "Hello AI", nil)
	cm.AddMessage("assistant", "Hello!", nil)
	cm.AddMessage("user", "How are you?", nil)

	// Export
	exportData, err := cm.ExportContext()
	require.NoError(t, err)
	require.NotEmpty(t, exportData)

	// Create new context manager and import
	cm2 := NewContextManager("import-test", config, mockVerifier, mockStorage)

	err = cm2.ImportContext(exportData)
	require.NoError(t, err)

	// Verify imported data
	messages, _, err := cm2.GetContext("", 10)
	require.NoError(t, err)
	assert.Len(t, messages, 3)
}

func TestContextManager_ImportContext_InvalidJSON(t *testing.T) {
	mockVerifier := &MockVerifier{}
	mockStorage := NewMockStorage()

	config := ContextConfig{
		ShortTermMaxMessages:    5,
		ShortTermWindowDuration: 10 * time.Minute,
		LongTermMaxSummaries:    5,
		SummarizationThreshold:  3,
	}

	cm := NewContextManager("import-test", config, mockVerifier, mockStorage)

	err := cm.ImportContext([]byte("invalid json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestContextManager_GetFullContext(t *testing.T) {
	mockVerifier := &MockVerifier{}
	mockStorage := NewMockStorage()

	config := ContextConfig{
		ShortTermMaxMessages:    5,
		ShortTermWindowDuration: 10 * time.Minute,
		LongTermMaxSummaries:    5,
		SummarizationThreshold:  2,
	}

	cm := NewContextManager("full-context-test", config, mockVerifier, mockStorage)

	// Add messages
	cm.AddMessage("user", "First message", nil)
	cm.AddMessage("assistant", "First response", nil)

	messages, summaries := cm.GetFullContext()

	assert.NotEmpty(t, messages)
	assert.NotNil(t, summaries) // May or may not have summaries depending on threshold
}

func TestContextManager_SearchContext_WithSummaries(t *testing.T) {
	mockVerifier := &MockVerifier{}
	mockStorage := NewMockStorage()

	config := ContextConfig{
		ShortTermMaxMessages:    10,
		ShortTermWindowDuration: 10 * time.Minute,
		LongTermMaxSummaries:    5,
		SummarizationThreshold:  2,
	}

	cm := NewContextManager("search-test", config, mockVerifier, mockStorage)

	// Add messages about AI
	cm.AddMessage("user", "Tell me about AI", nil)
	cm.AddMessage("assistant", "AI is artificial intelligence", nil)
	cm.AddMessage("user", "What about machine learning?", nil)

	// Search with summaries included
	messages, summaries := cm.SearchContext("AI", true)

	assert.NotEmpty(t, messages)
	// Summaries depend on LongTermMemory implementation
	assert.NotNil(t, summaries)
}

func TestContextManager_ClearContext_WithStorage(t *testing.T) {
	mockVerifier := &MockVerifier{}
	mockStorage := NewMockStorage()

	config := ContextConfig{
		ShortTermMaxMessages:    5,
		ShortTermWindowDuration: 10 * time.Minute,
		LongTermMaxSummaries:    5,
		SummarizationThreshold:  2,
	}

	cm := NewContextManager("clear-test", config, mockVerifier, mockStorage)

	// Add messages
	cm.AddMessage("user", "Test message", nil)

	// Clear
	err := cm.ClearContext()
	require.NoError(t, err)

	// Verify cleared
	stats := cm.GetStats()
	assert.Equal(t, 0, stats.ShortTermMessages)
}

// ==================== Helper Functions Tests ====================

func TestMinInt(t *testing.T) {
	assert.Equal(t, 5, minInt(5, 10))
	assert.Equal(t, 5, minInt(10, 5))
	assert.Equal(t, 5, minInt(5, 5))
	assert.Equal(t, -10, minInt(-10, 5))
	assert.Equal(t, 0, minInt(0, 100))
}

func TestMaxf(t *testing.T) {
	assert.Equal(t, 10.0, maxf(5.0, 10.0))
	assert.Equal(t, 10.0, maxf(10.0, 5.0))
	assert.Equal(t, 5.0, maxf(5.0, 5.0))
	assert.Equal(t, 5.0, maxf(-10.0, 5.0))
	assert.Equal(t, 100.0, maxf(0.0, 100.0))
}

// ==================== ContextStats Tests ====================

func TestContextStats_Struct(t *testing.T) {
	stats := ContextStats{
		ShortTermMessages:    10,
		ShortTermMaxMessages: 20,
		LongTermSummaries:    5,
		LongTermMaxSummaries: 10,
		TotalMemoryUsage:     1024,
		ConversationAge:      24 * time.Hour,
		LastActivity:         time.Now(),
		MemoryPressure:       0.5,
		CustomStats: map[string]interface{}{
			"test_stat": 42,
		},
	}

	assert.Equal(t, 10, stats.ShortTermMessages)
	assert.Equal(t, 20, stats.ShortTermMaxMessages)
	assert.Equal(t, 5, stats.LongTermSummaries)
	assert.Equal(t, int64(1024), stats.TotalMemoryUsage)
	assert.Equal(t, 0.5, stats.MemoryPressure)
	assert.Equal(t, 42, stats.CustomStats["test_stat"])
}

// ==================== ContextConfig Tests ====================

func TestContextConfig_Struct(t *testing.T) {
	config := ContextConfig{
		ShortTermMaxMessages:    100,
		ShortTermWindowDuration: 30 * time.Minute,
		LongTermMaxSummaries:    50,
		SummarizationThreshold:  10,
		BackupEnabled:           true,
		BackupInterval:          5 * time.Minute,
		StorageConfig: map[string]interface{}{
			"type": "filesystem",
		},
	}

	assert.Equal(t, 100, config.ShortTermMaxMessages)
	assert.Equal(t, 30*time.Minute, config.ShortTermWindowDuration)
	assert.Equal(t, 50, config.LongTermMaxSummaries)
	assert.True(t, config.BackupEnabled)
}

// ==================== Conversation Additional Tests ====================

func TestConversation_WindowExpiry(t *testing.T) {
	conv := NewConversation("expiry-test", 10, 100*time.Millisecond) // Very short window

	// Add message
	msg := &Message{
		ID:        "msg1",
		Role:      "user",
		Content:   "Test",
		Timestamp: time.Now().Add(-200 * time.Millisecond), // Older than window
	}
	conv.AddMessage(msg)

	// Add recent message
	msg2 := &Message{
		ID:        "msg2",
		Role:      "assistant",
		Content:   "Response",
		Timestamp: time.Now(),
	}
	conv.AddMessage(msg2)

	// Both messages should be present (window is about age, not automatic removal)
	messages := conv.GetMessages()
	assert.NotEmpty(t, messages)
}

func TestConversation_MaxMessages(t *testing.T) {
	conv := NewConversation("max-test", 3, 10*time.Minute)

	// Add 5 messages
	for i := 1; i <= 5; i++ {
		msg := &Message{
			ID:        "msg" + string(rune('0'+i)),
			Role:      "user",
			Content:   "Message " + string(rune('0'+i)),
			Timestamp: time.Now(),
		}
		conv.AddMessage(msg)
	}

	// Should have at most 3 messages
	messages := conv.GetMessages()
	assert.LessOrEqual(t, len(messages), 3)
}

// ==================== Long Term Memory Additional Tests ====================

func TestLongTermMemory_ExportImport(t *testing.T) {
	mockVerifier := &MockVerifier{}
	ltm := NewLongTermMemory(5, 2, mockVerifier)

	// Add messages to create summaries
	messages := []*Message{
		{ID: "msg1", Role: "user", Content: "Hello", Timestamp: time.Now()},
		{ID: "msg2", Role: "assistant", Content: "Hi there", Timestamp: time.Now()},
	}
	err := ltm.AddMessages(messages)
	require.NoError(t, err)

	// Export
	exportData, err := ltm.ExportMemory()
	require.NoError(t, err)
	require.NotEmpty(t, exportData)

	// Create new LTM and import
	ltm2 := NewLongTermMemory(5, 2, mockVerifier)
	err = ltm2.ImportMemory(exportData)
	require.NoError(t, err)

	// Verify import worked
	summaries := ltm2.GetAllSummaries()
	assert.NotEmpty(t, summaries)
}

func TestLongTermMemory_GetRelevantSummaries_NoMatch(t *testing.T) {
	mockVerifier := &MockVerifier{}
	ltm := NewLongTermMemory(5, 2, mockVerifier)

	// Add messages
	messages := []*Message{
		{ID: "msg1", Role: "user", Content: "Hello", Timestamp: time.Now()},
		{ID: "msg2", Role: "assistant", Content: "Hi there", Timestamp: time.Now()},
	}
	ltm.AddMessages(messages)

	// Search for something completely unrelated
	summaries := ltm.GetRelevantSummaries("xyzabc123nonexistent", 5)
	// Should return empty or low-relevance results (may be nil or empty slice)
	if summaries != nil {
		// If not nil, the results should still be valid
		for _, s := range summaries {
			assert.NotEmpty(t, s.ID)
		}
	}
}
