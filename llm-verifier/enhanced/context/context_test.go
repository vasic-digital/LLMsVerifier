package context

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	llmverifier "digital.vasic.llmsverifier/llmverifier"
)

// MockVerifier implements a mock LLM verifier for testing
type MockVerifier struct{}

func NewMockVerifier() *MockVerifier {
	return &MockVerifier{}
}

func (m *MockVerifier) SummarizeConversation(messages []string) (*llmverifier.ConversationSummary, error) {
	// Return a mock summary
	return &llmverifier.ConversationSummary{
		Summary:    "Test conversation about AI and related topics",
		Topics:     []string{"AI", "testing", "conversation"},
		KeyPoints:  []string{"Important point 1", "Important point 2", "Test discussion"},
		Importance: 0.8,
	}, nil
}

// MockStorage implements mock storage for testing.
//
// ROOT CAUSE (independent-audit finding, 2026-07-10): MockStorage's `data`
// map was read/written with no synchronization. HybridStorage.SaveContext
// (enhanced/context/storage.go) deliberately replicates to every replica
// backend CONCURRENTLY via unsynchronized goroutines (fire-and-forget async
// replication, by design -- the caller is not meant to block on replica
// writes), and HybridStorage.LoadContext similarly fires a background
// restore-to-primary goroutine on a replica hit. Any ContextStorage
// implementation MUST therefore tolerate concurrent SaveContext/LoadContext/
// DeleteContext/ListConversations calls -- exactly like a real backing store
// (SQL database, Redis, etc.) would via its own internal locking/connection
// pooling. A bare, unsynchronized map does NOT satisfy that implicit
// contract: confirmed under `go test -race` (WARNING: DATA RACE,
// MockStorage.SaveContext's map write vs MockStorage.LoadContext's map read,
// TestHybridStorage_SaveContext / TestHybridStorage_LoadContext_
// FallbackToReplica). A test relying on `time.Sleep` to "wait for" the async
// replication goroutine does NOT establish a happens-before edge under Go's
// memory model, so the race is real even though the sleep usually makes it
// look correct. Fixed here (test-only change; HybridStorage's intentional
// async-replication design in storage.go is untouched) by guarding MockStorage
// with a mutex, matching what any real, well-behaved concurrent-safe backend
// must already do.
type MockStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string][]byte),
	}
}

func (m *MockStorage) SaveContext(ctx context.Context, conversationID string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[conversationID] = data
	return nil
}

func (m *MockStorage) LoadContext(ctx context.Context, conversationID string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, exists := m.data[conversationID]; exists {
		return data, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *MockStorage) DeleteContext(ctx context.Context, conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, conversationID)
	return nil
}

func (m *MockStorage) ListConversations(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var conversations []string
	for id := range m.data {
		conversations = append(conversations, id)
	}
	return conversations, nil
}

// Test the long-term memory functionality
func TestLongTermMemoryBasic(t *testing.T) {
	mockVerifier := &MockVerifier{}
	ltm := NewLongTermMemory(5, 2, mockVerifier)

	// Create test messages
	messages := []*Message{
		{
			ID:        "msg1",
			Role:      "user",
			Content:   "Hello AI",
			Timestamp: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:        "msg2",
			Role:      "assistant",
			Content:   "Hello! How can I help?",
			Timestamp: time.Now().Add(-1 * time.Hour),
		},
	}

	// Test adding messages
	err := ltm.AddMessages(messages)
	if err != nil {
		t.Fatalf("Failed to add messages: %v", err)
	}

	// Test getting summaries
	summaries := ltm.GetAllSummaries()
	if len(summaries) == 0 {
		t.Error("Expected at least one summary")
	}

	// Test relevant summaries
	relevant := ltm.GetRelevantSummaries("AI", 5)
	if len(relevant) == 0 {
		t.Error("Expected relevant summaries for AI query")
	}

	// Test stats
	stats := ltm.GetMemoryStats()
	if summaryCount, ok := stats["summary_count"].(int); !ok || summaryCount == 0 {
		t.Error("Expected non-zero summary count")
	}

	// Test clear
	ltm.Clear()
	clearedSummaries := ltm.GetAllSummaries()
	if len(clearedSummaries) != 0 {
		t.Error("Expected no summaries after clearing")
	}
}

// Test the short-term conversation functionality
func TestShortTermConversation(t *testing.T) {
	conv := NewConversation("test-conv", 3, 10*time.Minute)

	// Add some messages
	msg1 := &Message{
		ID:        "msg1",
		Role:      "user",
		Content:   "Hello",
		Timestamp: time.Now(),
	}
	msg2 := &Message{
		ID:        "msg2",
		Role:      "assistant",
		Content:   "Hi there!",
		Timestamp: time.Now(),
	}

	conv.AddMessage(msg1)
	conv.AddMessage(msg2)

	// Test message count
	if conv.GetMessageCount() != 2 {
		t.Errorf("Expected 2 messages, got %d", conv.GetMessageCount())
	}

	// Test getting messages
	messages := conv.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// Test window info
	info := conv.GetWindowInfo()
	if messageCount, ok := info["message_count"].(int); !ok || messageCount != 2 {
		t.Error("Expected message_count to be 2")
	}

	// Test clear
	conv.Clear()
	if conv.GetMessageCount() != 0 {
		t.Error("Expected 0 messages after clearing")
	}
}

// Test file system storage
func TestFileSystemStorageBasic(t *testing.T) {
	tempDir := t.TempDir()

	fs, err := NewFileSystemStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem storage: %v", err)
	}

	ctx := context.Background()
	conversationID := "test-conv"
	testData := []byte(`{"test": "data"}`)

	// Test save
	err = fs.SaveContext(ctx, conversationID, testData)
	if err != nil {
		t.Fatalf("Failed to save context: %v", err)
	}

	// Test load
	loadedData, err := fs.LoadContext(ctx, conversationID)
	if err != nil {
		t.Fatalf("Failed to load context: %v", err)
	}

	if string(loadedData) != string(testData) {
		t.Error("Loaded data doesn't match saved data")
	}

	// Test list
	conversations, err := fs.ListConversations(ctx)
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}

	if len(conversations) != 1 || conversations[0] != conversationID {
		t.Error("Expected one conversation with correct ID")
	}

	// Test delete
	err = fs.DeleteContext(ctx, conversationID)
	if err != nil {
		t.Fatalf("Failed to delete context: %v", err)
	}

	// Verify deletion
	_, err = fs.LoadContext(ctx, conversationID)
	if err == nil {
		t.Error("Expected error when loading deleted conversation")
	}
}

// Test mock storage
func TestMockStorage(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()
	conversationID := "test-conv"
	testData := []byte(`{"test": "data"}`)

	// Test save
	err := storage.SaveContext(ctx, conversationID, testData)
	if err != nil {
		t.Fatalf("Failed to save context: %v", err)
	}

	// Test load
	loadedData, err := storage.LoadContext(ctx, conversationID)
	if err != nil {
		t.Fatalf("Failed to load context: %v", err)
	}

	if string(loadedData) != string(testData) {
		t.Error("Loaded data doesn't match saved data")
	}

	// Test list
	conversations, err := storage.ListConversations(ctx)
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}

	if len(conversations) != 1 || conversations[0] != conversationID {
		t.Error("Expected one conversation with correct ID")
	}

	// Test delete
	err = storage.DeleteContext(ctx, conversationID)
	if err != nil {
		t.Fatalf("Failed to delete context: %v", err)
	}

	// Verify deletion
	_, err = storage.LoadContext(ctx, conversationID)
	if err == nil {
		t.Error("Expected error when loading deleted conversation")
	}
}

// Test the message search functionality
func TestMessageSearch(t *testing.T) {
	conv := NewConversation("test-conv", 10, 10*time.Minute)

	// Add test messages
	msg1 := &Message{
		ID:        "msg1",
		Role:      "user",
		Content:   "What is machine learning?",
		Timestamp: time.Now(),
	}
	msg2 := &Message{
		ID:        "msg2",
		Role:      "assistant",
		Content:   "Machine learning is a subset of AI",
		Timestamp: time.Now(),
	}
	msg3 := &Message{
		ID:        "msg3",
		Role:      "user",
		Content:   "Tell me about neural networks",
		Timestamp: time.Now(),
	}

	conv.AddMessage(msg1)
	conv.AddMessage(msg2)
	conv.AddMessage(msg3)

	// Test search functionality
	messages := conv.GetMessages()
	queryLower := "machine"
	var matchingMessages []*Message

	for _, msg := range messages {
		contentLower := strings.ToLower(msg.Content)
		if strings.Contains(contentLower, queryLower) {
			matchingMessages = append(matchingMessages, msg)
		}
	}

	if len(matchingMessages) != 2 {
		t.Errorf("Expected 2 matching messages for 'machine', got %d", len(matchingMessages))
	}
}

// Test context manager with mock verifier
func TestContextManagerWithMock(t *testing.T) {
	mockVerifier := &MockVerifier{}
	mockStorage := NewMockStorage()

	config := ContextConfig{
		ShortTermMaxMessages:    3,
		ShortTermWindowDuration: 5 * time.Minute,
		LongTermMaxSummaries:    5,
		SummarizationThreshold:  2,
		BackupEnabled:           false, // Disable backup for test
	}

	cm := NewContextManager("test-conv", config, mockVerifier, mockStorage)

	// Add messages
	err := cm.AddMessage("user", "Hello AI", nil)
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	err = cm.AddMessage("assistant", "Hello! How can I help you today?", nil)
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Test getting context
	messages, _, err := cm.GetContext("AI", 10)
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if len(messages) == 0 {
		t.Error("Expected messages but got none")
	}

	// Test stats
	stats := cm.GetStats()
	if stats.ShortTermMessages == 0 {
		t.Error("Expected non-zero short term messages")
	}

	// Test search
	searchMessages, _ := cm.SearchContext("AI", false)
	if len(searchMessages) == 0 {
		t.Error("Expected search results for 'AI'")
	}

	// Test clear
	err = cm.ClearContext()
	if err != nil {
		t.Fatalf("Failed to clear context: %v", err)
	}

	// Verify cleared
	clearedMessages, _, err := cm.GetContext("AI", 10)
	if err != nil {
		t.Fatalf("Failed to get cleared context: %v", err)
	}

	if len(clearedMessages) != 0 {
		t.Error("Expected no messages after clearing")
	}
}
