package context

import (
	"container/list"
	"sync"
	"time"
)

// Message represents a single message in the conversation
type Message struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"` // user, assistant, system
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Conversation represents a conversation with sliding window context
type Conversation struct {
	ID             string        `json:"id"`
	Messages       *list.List    `json:"-"` // Not serialized
	MaxMessages    int           `json:"max_messages"`
	WindowDuration time.Duration `json:"window_duration"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	mu             sync.RWMutex
}

// NewConversation creates a new conversation with sliding window
func NewConversation(id string, maxMessages int, windowDuration time.Duration) *Conversation {
	return &Conversation{
		ID:             id,
		Messages:       list.New(),
		MaxMessages:    maxMessages,
		WindowDuration: windowDuration,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// AddMessage adds a message to the conversation and manages the sliding window
func (c *Conversation) AddMessage(msg *Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set timestamp if not provided
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Add message to the end
	c.Messages.PushBack(msg)
	c.UpdatedAt = time.Now()

	// Apply sliding window: remove messages outside the time window
	cleanupTime := time.Now().Add(-c.WindowDuration)
	for e := c.Messages.Front(); e != nil; {
		msg := e.Value.(*Message)
		if msg.Timestamp.Before(cleanupTime) {
			next := e.Next()
			c.Messages.Remove(e)
			e = next
		} else {
			break // Messages are ordered by time, so we can stop
		}
	}

	// Apply sliding window: limit by message count
	for c.Messages.Len() > c.MaxMessages {
		c.Messages.Remove(c.Messages.Front())
	}
}

// GetMessages returns all messages in the current window
func (c *Conversation) GetMessages() []*Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	messages := make([]*Message, 0, c.Messages.Len())
	for e := c.Messages.Front(); e != nil; e = e.Next() {
		messages = append(messages, e.Value.(*Message))
	}
	return messages
}

// GetRecentMessages returns the most recent N messages
func (c *Conversation) GetRecentMessages(limit int) []*Message {
	allMessages := c.GetMessages()
	count := len(allMessages)

	if limit <= 0 || count <= limit {
		return allMessages
	}

	// Return the most recent messages
	return allMessages[count-limit:]
}

// Clear removes all messages from the conversation
func (c *Conversation) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Messages = list.New()
	c.UpdatedAt = time.Now()
}

// GetMessageCount returns the current number of messages
func (c *Conversation) GetMessageCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Messages.Len()
}

// GetWindowInfo returns information about the current sliding window
func (c *Conversation) GetWindowInfo() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	oldest := time.Now()
	newest := time.Time{}

	for e := c.Messages.Front(); e != nil; e = e.Next() {
		msg := e.Value.(*Message)
		if msg.Timestamp.Before(oldest) {
			oldest = msg.Timestamp
		}
		if msg.Timestamp.After(newest) {
			newest = msg.Timestamp
		}
	}

	return map[string]interface{}{
		"message_count":   c.Messages.Len(),
		"max_messages":    c.MaxMessages,
		"window_duration": c.WindowDuration.String(),
		"oldest_message":  oldest,
		"newest_message":  newest,
		"window_span":     newest.Sub(oldest).String(),
	}
}

// ConversationManager manages multiple conversations
type ConversationManager struct {
	conversations         map[string]*Conversation
	defaultMaxMessages    int
	defaultWindowDuration time.Duration
	mu                    sync.RWMutex
}

// NewConversationManager creates a new conversation manager
func NewConversationManager(defaultMaxMessages int, defaultWindowDuration time.Duration) *ConversationManager {
	return &ConversationManager{
		conversations:         make(map[string]*Conversation),
		defaultMaxMessages:    defaultMaxMessages,
		defaultWindowDuration: defaultWindowDuration,
	}
}

// CreateConversation creates a new conversation
func (cm *ConversationManager) CreateConversation(id string) *Conversation {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv := NewConversation(id, cm.defaultMaxMessages, cm.defaultWindowDuration)
	cm.conversations[id] = conv
	return conv
}

// GetConversation retrieves a conversation by ID
func (cm *ConversationManager) GetConversation(id string) *Conversation {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.conversations[id]
}

// DeleteConversation removes a conversation
func (cm *ConversationManager) DeleteConversation(id string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.conversations, id)
}

// ListConversations returns all conversation IDs
func (cm *ConversationManager) ListConversations() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ids := make([]string, 0, len(cm.conversations))
	for id := range cm.conversations {
		ids = append(ids, id)
	}
	return ids
}

// CleanupOldConversations removes conversations that haven't been updated recently
func (cm *ConversationManager) CleanupOldConversations(maxAge time.Duration) int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, conv := range cm.conversations {
		if conv.UpdatedAt.Before(cutoff) {
			delete(cm.conversations, id)
			removed++
		}
	}

	return removed
}
