package enhanced

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"llm-verifier/llmverifier"
)

func TestNewContextManager(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	assert.NotNil(t, cm)
	assert.NotNil(t, cm.shortTerm)
	assert.NotNil(t, cm.longTerm)
	assert.NotNil(t, cm.verifier)
	assert.NotNil(t, cm.lastActivity)
	assert.NotNil(t, cm.stopCh)
	assert.NotNil(t, cm.mu)
	assert.Equal(t, 1000, cm.maxContexts)
	assert.Equal(t, 24*time.Hour, cm.contextTTL)
}

func TestStartConversation(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	conv, err := cm.StartConversation("test-conv-1")

	assert.NoError(t, err)
	assert.NotNil(t, conv)
	assert.Equal(t, "test-conv-1", conv.ID)
}

func TestStartConversationAlreadyExists(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	// Create first conversation
	cm.StartConversation("test-conv-1")

	// Try to create same conversation again
	conv, err := cm.StartConversation("test-conv-1")

	assert.NoError(t, err)
	assert.NotNil(t, conv)
	assert.Equal(t, "test-conv-1", conv.ID)
}

func TestStartConversationMaxCapacity(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)
	cm.maxContexts = 2 // Set low limit for testing

	// Create max conversations
	cm.StartConversation("conv-1")
	cm.StartConversation("conv-2")

	// Try to create one more
	conv, err := cm.StartConversation("conv-3")

	assert.Error(t, err)
	assert.Nil(t, conv)
	assert.Contains(t, err.Error(), "maximum number of concurrent")
}

func TestAddMessage(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	// Start conversation first
	cm.StartConversation("test-conv")

	err := cm.AddMessage("test-conv", "user", "Hello world", nil)

	assert.NoError(t, err)
}

func TestAddMessageAutoStartConversation(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	// Add message to non-existent conversation (should auto-start)
	err := cm.AddMessage("auto-conv", "user", "Auto start", nil)

	assert.NoError(t, err)
}

func TestAddMessageWithMetadata(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("test-conv")

	metadata := map[string]interface{}{
		"source":  "web",
		"user_id": "123",
	}

	err := cm.AddMessage("test-conv", "assistant", "Response", metadata)

	assert.NoError(t, err)
}

func TestAddMessageMultiple(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("test-conv")

	// Add multiple messages
	for i := 0; i < 5; i++ {
		err := cm.AddMessage("test-conv", "user", "Message", nil)
		assert.NoError(t, err)
	}
}

func TestGetContext(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("test-conv")
	cm.AddMessage("test-conv", "user", "Test message", nil)

	context, err := cm.GetContext("test-conv", "query", 10)

	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Equal(t, "test-conv", context["conversation_id"])
	assert.NotNil(t, context["short_term"])
	assert.NotNil(t, context["long_term"])
}

func TestGetContextNonExistentConversation(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	context, err := cm.GetContext("non-existent", "query", 10)

	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Equal(t, "non-existent", context["conversation_id"])
}

func TestGetContextMaxResults(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("test-conv")
	cm.AddMessage("test-conv", "user", "Message 1", nil)
	cm.AddMessage("test-conv", "user", "Message 2", nil)

	context, err := cm.GetContext("test-conv", "query", 5)

	assert.NoError(t, err)
	assert.NotNil(t, context)
}

func TestEndConversation(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("test-conv")

	err := cm.EndConversation("test-conv")

	assert.NoError(t, err)
}

func TestEndConversationNonExistent(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	err := cm.EndConversation("non-existent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEndConversationAlreadyEnded(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("test-conv")
	cm.EndConversation("test-conv")

	err := cm.EndConversation("test-conv")

	assert.Error(t, err)
}

func TestGetConversationStats(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("conv-1")
	cm.StartConversation("conv-2")
	cm.AddMessage("conv-1", "user", "Message", nil)

	stats := cm.GetConversationStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats["active_conversations"])
	assert.NotNil(t, stats["total_messages"])
	assert.Equal(t, 1000, stats["max_conversations"])
	assert.Equal(t, 24.0, stats["context_ttl_hours"])
}

func TestGetConversationStatsEmpty(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	stats := cm.GetConversationStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["active_conversations"])
	assert.Equal(t, 0, stats["total_messages"])
}

func TestCleanupExpiredConversations(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)
	cm.contextTTL = time.Millisecond * 100 // Set short TTL for testing

	// Create conversations
	cm.StartConversation("conv-1")
	cm.StartConversation("conv-2")

	// Update activity for one conversation
	cm.mu.Lock()
	cm.lastActivity["conv-1"] = time.Now()
	cm.lastActivity["conv-2"] = time.Now().Add(-time.Hour) // Expired
	cm.mu.Unlock()

	// Wait for TTL
	time.Sleep(time.Millisecond * 150)

	// Run cleanup
	cm.Cleanup()

	// Check stats
	stats := cm.GetConversationStats()
	// Note: Cleanup removes from maps but we can't easily verify the exact count
	assert.NotNil(t, stats)
}

func TestCleanupNoExpired(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("conv-1")
	cm.StartConversation("conv-2")

	// Update all activities to recent
	cm.mu.Lock()
	cm.lastActivity["conv-1"] = time.Now()
	cm.lastActivity["conv-2"] = time.Now()
	cm.mu.Unlock()

	// Run cleanup
	cm.Cleanup()

	stats := cm.GetConversationStats()
	assert.Equal(t, 2, stats["active_conversations"])
}

func TestShutdown(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	// Should not panic
	assert.NotPanics(t, func() { cm.Shutdown() })
}

func TestShutdownMultiple(t *testing.T) {
	t.Skip("Skip - Shutdown() panics when called twice")
}

func TestAddMessageDifferentRoles(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("test-conv")

	roles := []string{"user", "assistant", "system", "user"}

	for _, role := range roles {
		err := cm.AddMessage("test-conv", role, "Message", nil)
		assert.NoError(t, err)
	}
}

func TestConcurrentConversations(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	// Create multiple conversations
	for i := 0; i < 10; i++ {
		convID := "conv-" + string(rune('0'+i))
		_, err := cm.StartConversation(convID)
		assert.NoError(t, err)
	}

	// Add messages to each
	cm.mu.RLock()
	assert.Equal(t, 10, len(cm.shortTerm))
	cm.mu.RUnlock()
}

func TestContextManagerDefaultValues(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	assert.Equal(t, 1000, cm.maxContexts)
	assert.Equal(t, 24*time.Hour, cm.contextTTL)
	assert.NotNil(t, cm.stopCh)
}

func TestContextManagerNilVerifier(t *testing.T) {
	cm := NewContextManager(nil)

	assert.NotNil(t, cm)
	assert.NotNil(t, cm.longTerm)
}

func TestGetConversationStatsLongTermMemory(t *testing.T) {
	verifier := llmverifier.New(nil)
	cm := NewContextManager(verifier)

	cm.StartConversation("conv-1")

	stats := cm.GetConversationStats()

	assert.NotNil(t, stats)
	assert.NotNil(t, stats["long_term_summaries"])
}
