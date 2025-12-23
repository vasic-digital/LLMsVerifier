package enhanced

import (
	"fmt"
	"log"
	"sync"
	"time"

	ctxt "llm-verifier/enhanced/context"
	llmverifier "llm-verifier/llmverifier"
)

// ContextManager coordinates short-term and long-term context management
type ContextManager struct {
	shortTerm    map[string]*ctxt.Conversation // conversation_id -> conversation
	longTerm     *ctxt.LongTermMemory
	verifier     *llmverifier.Verifier
	maxContexts  int
	contextTTL   time.Duration // Time to live for inactive contexts
	stopCh       chan struct{}
	mu           sync.RWMutex
	lastActivity map[string]time.Time // Track last activity per conversation
}

// NewContextManager creates a new context manager
func NewContextManager(verifier *llmverifier.Verifier) *ContextManager {
	cm := &ContextManager{
		shortTerm:    make(map[string]*ctxt.Conversation),
		longTerm:     ctxt.NewLongTermMemory(100, 50, verifier), // Max 100 summaries, summarize every 50 messages
		verifier:     verifier,
		maxContexts:  1000,           // Max 1000 concurrent conversations
		contextTTL:   24 * time.Hour, // 24 hours TTL for inactive contexts
		lastActivity: make(map[string]time.Time),
		stopCh:       make(chan struct{}),
	}

	return cm
}

// StartConversation starts a new conversation with context management
func (cm *ContextManager) StartConversation(conversationID string) (*ctxt.Conversation, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if we already have this conversation
	if conv, exists := cm.shortTerm[conversationID]; exists {
		return conv, nil
	}

	// Check if we're at capacity
	if len(cm.shortTerm) >= cm.maxContexts {
		return nil, fmt.Errorf("maximum number of concurrent conversations reached")
	}

	// Create new conversation (6-10 messages window, 1 hour duration)
	conv := ctxt.NewConversation(conversationID, 10, time.Hour)
	cm.shortTerm[conversationID] = conv
	cm.lastActivity[conversationID] = time.Now()

	log.Printf("Started conversation %s", conversationID)
	return conv, nil
}

// AddMessage adds a message to a conversation and processes context
func (cm *ContextManager) AddMessage(conversationID string, role, content string, metadata map[string]interface{}) error {
	cm.mu.RLock()
	conv, exists := cm.shortTerm[conversationID]
	cm.mu.RUnlock()

	if !exists {
		// Auto-start conversation if it doesn't exist
		var err error
		conv, err = cm.StartConversation(conversationID)
		if err != nil {
			return err
		}
	}

	// Create message
	msg := &ctxt.Message{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	// Add to short-term context
	conv.AddMessage(msg)

	// Update last activity
	cm.mu.Lock()
	cm.lastActivity[conversationID] = time.Now()
	cm.mu.Unlock()

	// Process for long-term memory
	messages := conv.GetRecentMessages(20) // Get last 20 messages
	if err := cm.longTerm.AddMessages(messages); err != nil {
		log.Printf("Failed to process long-term memory for conversation %s: %v", conversationID, err)
	}

	log.Printf("Added message to conversation %s", conversationID)
	return nil
}

// GetContext retrieves relevant context for a query
func (cm *ContextManager) GetContext(conversationID, query string, maxResults int) (map[string]interface{}, error) {
	cm.mu.RLock()
	conv, exists := cm.shortTerm[conversationID]
	cm.mu.RUnlock()

	result := map[string]interface{}{
		"short_term":      []interface{}{},
		"long_term":       []interface{}{},
		"conversation_id": conversationID,
	}

	// Get short-term context
	if exists {
		messages := conv.GetRecentMessages(10)
		shortTermMsgs := make([]interface{}, len(messages))
		for i, msg := range messages {
			shortTermMsgs[i] = map[string]interface{}{
				"role":      msg.Role,
				"content":   msg.Content,
				"timestamp": msg.Timestamp,
			}
		}
		result["short_term"] = shortTermMsgs
	}

	// Get long-term context
	summaries := cm.longTerm.GetRelevantSummaries(query, 3)
	longTermSummaries := make([]interface{}, len(summaries))
	for i, summary := range summaries {
		longTermSummaries[i] = map[string]interface{}{
			"content":    summary.Content,
			"topics":     summary.Topics,
			"importance": summary.Importance,
			"start_time": summary.StartTime,
			"end_time":   summary.EndTime,
		}
	}
	result["long_term"] = longTermSummaries

	return result, nil
}

// EndConversation ends a conversation and cleans up resources
func (cm *ContextManager) EndConversation(conversationID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.shortTerm[conversationID]; !exists {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	delete(cm.shortTerm, conversationID)
	delete(cm.lastActivity, conversationID)
	log.Printf("Ended conversation %s", conversationID)
	return nil
}

// GetConversationStats returns statistics about active conversations
func (cm *ContextManager) GetConversationStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	totalMessages := 0
	activeConversations := len(cm.shortTerm)

	for _, conv := range cm.shortTerm {
		totalMessages += conv.GetMessageCount()
	}

	return map[string]interface{}{
		"active_conversations": activeConversations,
		"total_messages":       totalMessages,
		"max_conversations":    cm.maxContexts,
		"context_ttl_hours":    cm.contextTTL.Hours(),
		"long_term_summaries":  len(cm.longTerm.GetAllSummaries()),
	}
}

// Cleanup removes expired conversations
func (cm *ContextManager) Cleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for id, lastActivity := range cm.lastActivity {
		if now.Sub(lastActivity) > cm.contextTTL {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		delete(cm.shortTerm, id)
		delete(cm.lastActivity, id)
		log.Printf("Cleaned up expired conversation: %s", id)
	}

	if len(expired) > 0 {
		log.Printf("Cleaned up %d expired conversations", len(expired))
	}
}

// Shutdown gracefully shuts down the context manager
func (cm *ContextManager) Shutdown() {
	// Protect against double close
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	select {
	case <-cm.stopCh:
		// Already closed
		return
	default:
		close(cm.stopCh)
		log.Println("Context manager shutdown complete")
	}
}
