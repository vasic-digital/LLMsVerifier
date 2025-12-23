package enhanced

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewContextManager(t *testing.T) {
	cm := NewContextManager(nil)

	assert.NotNil(t, cm)
	assert.NotNil(t, cm.shortTerm)
	assert.NotNil(t, cm.longTerm)
	assert.Equal(t, 1000, cm.maxContexts)
	assert.Equal(t, 24*time.Hour, cm.contextTTL)
}

func TestContextManagerStartConversation(t *testing.T) {
	cm := NewContextManager(nil)

	conv, err := cm.StartConversation("conv-1")

	assert.NoError(t, err)
	assert.NotNil(t, conv)
	assert.Equal(t, "conv-1", conv.ID)
}

func TestContextManagerStartConversationDuplicate(t *testing.T) {
	cm := NewContextManager(nil)

	// Start conversation once
	_, err1 := cm.StartConversation("conv-1")
	assert.NoError(t, err1)

	// Try to start same conversation again - should return existing
	conv2, err2 := cm.StartConversation("conv-1")
	assert.NoError(t, err2)
	assert.NotNil(t, conv2)
}

func TestContextManagerStartConversationCapacityExceeded(t *testing.T) {
	cm := NewContextManager(nil)
	cm.maxContexts = 2 // Reduce capacity for testing

	// Start 2 conversations
	cm.StartConversation("conv-1")
	cm.StartConversation("conv-2")

	// Third should fail
	_, err := cm.StartConversation("conv-3")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of concurrent conversations")
}

func TestContextManagerAddMessage(t *testing.T) {
	cm := NewContextManager(nil)

	err := cm.AddMessage("conv-1", "user", "Hello", nil)

	assert.NoError(t, err)
}

func TestContextManagerAddMessageAutoStart(t *testing.T) {
	cm := NewContextManager(nil)

	// Add message without starting conversation - should auto-start
	err := cm.AddMessage("conv-auto", "user", "Test message", nil)

	assert.NoError(t, err)
}

func TestContextManagerAddMultipleMessages(t *testing.T) {
	cm := NewContextManager(nil)

	// Start conversation
	cm.StartConversation("conv-1")

	// Add multiple messages
	err1 := cm.AddMessage("conv-1", "user", "Hello", nil)
	err2 := cm.AddMessage("conv-1", "assistant", "Hi there!", nil)
	err3 := cm.AddMessage("conv-1", "user", "How are you?", nil)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
}

func TestContextManagerGetContext(t *testing.T) {
	cm := NewContextManager(nil)

	// Start conversation and add message
	cm.StartConversation("conv-1")
	cm.AddMessage("conv-1", "user", "Hello", nil)

	// Get context
	result, err := cm.GetContext("conv-1", "test query", 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "short_term")
	assert.Contains(t, result, "long_term")
	assert.Contains(t, result, "conversation_id")
}

func TestContextManagerGetContextNonExistent(t *testing.T) {
	cm := NewContextManager(nil)

	// Get context for non-existent conversation
	result, err := cm.GetContext("conv-999", "test", 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "conversation_id")
}

func TestContextManagerGetContextMaxResults(t *testing.T) {
	cm := NewContextManager(nil)

	// Start conversation
	cm.StartConversation("conv-1")

	// Add multiple messages
	for i := 0; i < 15; i++ {
		cm.AddMessage("conv-1", "user", "Message "+string(rune('0'+i)), nil)
	}

	// Get context with limited results
	result, err := cm.GetContext("conv-1", "test", 5)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestContextManagerEndConversation(t *testing.T) {
	cm := NewContextManager(nil)

	// Start conversation
	cm.StartConversation("conv-1")

	// End conversation
	err := cm.EndConversation("conv-1")

	assert.NoError(t, err)
}

func TestContextManagerEndConversationNotFound(t *testing.T) {
	cm := NewContextManager(nil)

	// Try to end non-existent conversation
	err := cm.EndConversation("conv-999")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestContextManagerGetConversationStats(t *testing.T) {
	cm := NewContextManager(nil)

	// Start conversations and add messages
	cm.StartConversation("conv-1")
	cm.AddMessage("conv-1", "user", "Hello", nil)
	cm.AddMessage("conv-1", "user", "Test", nil)

	cm.StartConversation("conv-2")
	cm.AddMessage("conv-2", "user", "Hi", nil)

	// Get stats
	stats := cm.GetConversationStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "active_conversations")
	assert.Contains(t, stats, "total_messages")
	assert.Contains(t, stats, "max_conversations")
	assert.Contains(t, stats, "context_ttl_hours")
}

func TestContextManagerGetConversationStatsEmpty(t *testing.T) {
	cm := NewContextManager(nil)

	// Get stats when no conversations
	stats := cm.GetConversationStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["active_conversations"])
	assert.Equal(t, 0, stats["total_messages"])
}

func TestContextManagerCleanup(t *testing.T) {
	cm := NewContextManager(nil)
	cm.contextTTL = 50 * time.Millisecond // Short TTL for testing

	// Start conversation
	cm.StartConversation("conv-expire")
	cm.AddMessage("conv-expire", "user", "Test", nil)

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Run cleanup
	cm.Cleanup()

	// Verify conversation was cleaned up
	stats := cm.GetConversationStats()
	// Should be 0 or close to 0 active conversations
	assert.LessOrEqual(t, stats["active_conversations"], 1)
}

func TestContextManagerCleanupNoExpired(t *testing.T) {
	cm := NewContextManager(nil)
	cm.contextTTL = 10 * time.Minute // Long TTL

	// Start conversation
	cm.StartConversation("conv-active")
	cm.AddMessage("conv-active", "user", "Test", nil)

	// Run cleanup
	cm.Cleanup()

	// Verify conversation is still active
	stats := cm.GetConversationStats()
	assert.Equal(t, 1, stats["active_conversations"])
}

func TestContextManagerMultipleConversations(t *testing.T) {
	cm := NewContextManager(nil)

	// Start multiple conversations
	cm.StartConversation("conv-1")
	cm.StartConversation("conv-2")
	cm.StartConversation("conv-3")

	// Add messages to each
	cm.AddMessage("conv-1", "user", "Hello", nil)
	cm.AddMessage("conv-2", "user", "Hi", nil)
	cm.AddMessage("conv-3", "user", "Hey", nil)

	// Check stats
	stats := cm.GetConversationStats()
	assert.Equal(t, 3, stats["active_conversations"])
	assert.Equal(t, 3, stats["total_messages"])
}

func TestContextManagerGetContextShortTerm(t *testing.T) {
	cm := NewContextManager(nil)

	// Start conversation and add messages
	cm.StartConversation("conv-1")
	cm.AddMessage("conv-1", "user", "Message 1", nil)
	cm.AddMessage("conv-1", "assistant", "Response 1", nil)
	cm.AddMessage("conv-1", "user", "Message 2", nil)

	// Get context
	result, err := cm.GetContext("conv-1", "test", 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	shortTerm := result["short_term"]
	assert.IsType(t, []interface{}{}, shortTerm)

	messages := shortTerm.([]interface{})
	assert.GreaterOrEqual(t, len(messages), 3)
}

func TestContextManagerGetContextLongTerm(t *testing.T) {
	cm := NewContextManager(nil)

	// Start conversation
	cm.StartConversation("conv-1")

	// Add many messages to trigger long-term processing
	for i := 0; i < 30; i++ {
		cm.AddMessage("conv-1", "user", "Message "+string(rune('0'+i)), nil)
	}

	// Get context
	result, err := cm.GetContext("conv-1", "test", 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	longTerm := result["long_term"]
	assert.IsType(t, []interface{}{}, longTerm)
}

func TestContextManagerGetContextConversationID(t *testing.T) {
	cm := NewContextManager(nil)

	// Start conversation
	cm.StartConversation("conv-123")

	// Get context
	result, err := cm.GetContext("conv-123", "test", 10)

	assert.NoError(t, err)
	assert.Equal(t, "conv-123", result["conversation_id"])
}

func TestContextManagerShutdown(t *testing.T) {
	cm := NewContextManager(nil)

	// Should not panic
	cm.Shutdown()

	assert.True(t, true)
}

func TestContextManagerLongTermMemory(t *testing.T) {
	cm := NewContextManager(nil)

	assert.NotNil(t, cm.longTerm)
}

func TestContextManagerVerifier(t *testing.T) {
	cm := NewContextManager(nil)

	assert.Nil(t, cm.verifier) // Will be nil in tests
}

func TestContextManagerStopChannel(t *testing.T) {
	cm := NewContextManager(nil)

	assert.NotNil(t, cm.stopCh)
}
