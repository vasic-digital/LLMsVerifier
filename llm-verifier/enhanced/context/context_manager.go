package context

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	llmverifier "llm-verifier/llmverifier"
)

// VerifierInterface defines the interface for LLM verifiers
type VerifierInterface interface {
	SummarizeConversation(messages []string) (*llmverifier.ConversationSummary, error)
}

// ContextManagerInterface defines the interface for context managers
type ContextManagerInterface interface {
	AddMessage(role, content string, metadata map[string]interface{}) error
	GetContext(query string, maxMessages int) ([]*Message, []*Summary, error)
	GetFullContext() ([]*Message, []*Summary)
	SearchContext(query string, includeSummaries bool) ([]*Message, []*Summary)
	GetStats() ContextStats
	ClearContext() error
	ExportContext() ([]byte, error)
	ImportContext(data []byte) error
}

// ContextType represents the type of context storage
type ContextType int

const (
	ContextTypeShortTerm ContextType = iota
	ContextTypeLongTerm
)

// ContextManager manages both short-term and long-term conversation context
type ContextManager struct {
	shortTerm     *Conversation
	longTerm      *LongTermMemory
	verifier      VerifierInterface
	mu            sync.RWMutex
	storage       ContextStorage
	backupEnabled bool
}

// ContextStorage interface for persisting context data
type ContextStorage interface {
	SaveContext(ctx context.Context, conversationID string, data []byte) error
	LoadContext(ctx context.Context, conversationID string) ([]byte, error)
	DeleteContext(ctx context.Context, conversationID string) error
	ListConversations(ctx context.Context) ([]string, error)
}

// ContextStats provides statistics about context usage
type ContextStats struct {
	ShortTermMessages    int                    `json:"short_term_messages"`
	ShortTermMaxMessages int                    `json:"short_term_max_messages"`
	LongTermSummaries    int                    `json:"long_term_summaries"`
	LongTermMaxSummaries int                    `json:"long_term_max_summaries"`
	TotalMemoryUsage     int64                  `json:"total_memory_usage_bytes"`
	ConversationAge      time.Duration          `json:"conversation_age_days"`
	LastActivity         time.Time              `json:"last_activity"`
	MemoryPressure       float64                `json:"memory_pressure"`
	CustomStats          map[string]interface{} `json:"custom_stats,omitempty"`
}

// ContextConfig holds configuration for context management
type ContextConfig struct {
	ShortTermMaxMessages    int                    `yaml:"short_term_max_messages"`
	ShortTermWindowDuration time.Duration          `yaml:"short_term_window_duration"`
	LongTermMaxSummaries    int                    `yaml:"long_term_max_summaries"`
	SummarizationThreshold  int                    `yaml:"summarization_threshold"`
	BackupEnabled           bool                   `yaml:"backup_enabled"`
	BackupInterval          time.Duration          `yaml:"backup_interval"`
	StorageConfig           map[string]interface{} `yaml:"storage_config"`
}

// NewContextManager creates a new context manager with both short and long-term memory
func NewContextManager(conversationID string, config ContextConfig, verifier VerifierInterface, storage ContextStorage) *ContextManager {
	// Create short-term conversation
	shortTerm := NewConversation(conversationID, config.ShortTermMaxMessages, config.ShortTermWindowDuration)

	// Create long-term memory
	longTerm := NewLongTermMemory(config.LongTermMaxSummaries, config.SummarizationThreshold, verifier)

	return &ContextManager{
		shortTerm:     shortTerm,
		longTerm:      longTerm,
		verifier:      verifier,
		storage:       storage,
		backupEnabled: config.BackupEnabled,
	}
}

// AddMessage adds a new message to the conversation context
func (cm *ContextManager) AddMessage(role, content string, metadata map[string]interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Add to short-term memory
	newMsg := &Message{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
	cm.shortTerm.AddMessage(newMsg)

	// Check if we need to create a long-term summary
	messages := cm.shortTerm.GetRecentMessages(0) // Get all messages for potential summarization
	if len(messages) >= cm.longTerm.summarizationThreshold {
		// Create summary from oldest messages that are about to be pushed out
		summaryMessages := make([]*Message, 0, cm.longTerm.summarizationThreshold)
		// Get the messages that would be removed from short-term memory
		for i := 0; i < minInt(len(messages), cm.longTerm.summarizationThreshold); i++ {
			if messages[i] != nil {
				summaryMessages = append(summaryMessages, &Message{
					ID:        messages[i].ID,
					Role:      messages[i].Role,
					Content:   messages[i].Content,
					Timestamp: messages[i].Timestamp,
					Metadata:  messages[i].Metadata,
				})
			}
		}

		if err := cm.longTerm.AddMessages(summaryMessages); err != nil {
			return fmt.Errorf("failed to add messages to long-term memory: %w", err)
		}
	}

	// Backup if enabled
	if cm.backupEnabled && cm.storage != nil {
		go cm.backupContext()
	}

	return nil
}

// GetContext retrieves relevant context for a given query
func (cm *ContextManager) GetContext(query string, maxMessages int) ([]*Message, []*Summary, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Get recent messages from short-term memory
	shortTermMessages := cm.shortTerm.GetRecentMessages(maxMessages)

	// Get relevant summaries from long-term memory
	relevantSummaries := cm.longTerm.GetRelevantSummaries(query, 5) // Limit to 5 most relevant

	// Convert short-term messages to the expected format
	messages := make([]*Message, len(shortTermMessages))
	for i, msg := range shortTermMessages {
		messages[i] = &Message{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			Metadata:  msg.Metadata,
		}
	}

	return messages, relevantSummaries, nil
}

// GetFullContext returns the complete context including all messages and summaries
func (cm *ContextManager) GetFullContext() ([]*Message, []*Summary) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Get all short-term messages
	shortTermMessages := cm.shortTerm.GetRecentMessages(0) // 0 means get all
	messages := make([]*Message, len(shortTermMessages))
	for i, msg := range shortTermMessages {
		messages[i] = &Message{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			Metadata:  msg.Metadata,
		}
	}

	// Get all long-term summaries
	summaries := cm.longTerm.GetAllSummaries()

	return messages, summaries
}

// SearchContext searches through both short-term and long-term context
func (cm *ContextManager) SearchContext(query string, includeSummaries bool) ([]*Message, []*Summary) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var matchingMessages []*Message
	var matchingSummaries []*Summary

	queryLower := strings.ToLower(query)

	// Search in short-term messages
	shortTermMessages := cm.shortTerm.GetRecentMessages(0)
	for _, msg := range shortTermMessages {
		if strings.Contains(strings.ToLower(msg.Content), queryLower) {
			matchingMessages = append(matchingMessages, &Message{
				ID:        msg.ID,
				Role:      msg.Role,
				Content:   msg.Content,
				Timestamp: msg.Timestamp,
				Metadata:  msg.Metadata,
			})
		}
	}

	// Search in long-term summaries if requested
	if includeSummaries {
		allSummaries := cm.longTerm.GetAllSummaries()
		for _, summary := range allSummaries {
			score := cm.longTerm.calculateRelevanceScore(summary, query)
			if score > 0.1 { // Same threshold as in long-term memory
				summaryCopy := *summary
				matchingSummaries = append(matchingSummaries, &summaryCopy)
			}
		}
	}

	return matchingMessages, matchingSummaries
}

// GetStats returns comprehensive statistics about the context manager
func (cm *ContextManager) GetStats() ContextStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	shortTermInfo := cm.shortTerm.GetWindowInfo()
	longTermStats := cm.longTerm.GetMemoryStats()

	// Calculate memory pressure (0.0 to 1.0)
	shortTermPressure := float64(shortTermInfo["message_count"].(int)) / float64(shortTermInfo["max_messages"].(int))
	longTermPressure := float64(longTermStats["summary_count"].(int)) / float64(longTermStats["max_summaries"].(int))
	memoryPressure := maxf(shortTermPressure, longTermPressure)

	// Calculate conversation age
	conversationAge := time.Since(cm.shortTerm.CreatedAt)

	stats := ContextStats{
		ShortTermMessages:    shortTermInfo["message_count"].(int),
		ShortTermMaxMessages: shortTermInfo["max_messages"].(int),
		LongTermSummaries:    longTermStats["summary_count"].(int),
		LongTermMaxSummaries: longTermStats["max_summaries"].(int),
		TotalMemoryUsage:     cm.calculateMemoryUsage(),
		ConversationAge:      conversationAge,
		LastActivity:         cm.shortTerm.UpdatedAt,
		MemoryPressure:       memoryPressure,
		CustomStats: map[string]interface{}{
			"short_term_window_duration": cm.shortTerm.WindowDuration,
			"total_messages_processed":   longTermStats["total_messages"],
			"average_importance":         longTermStats["average_importance"],
			"memory_span_days":           longTermStats["memory_span_days"],
		},
	}

	return stats
}

// ClearContext clears all context data
func (cm *ContextManager) ClearContext() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Clear short-term memory
	cm.shortTerm.Clear()

	// Clear long-term memory
	cm.longTerm.Clear()

	// Clear persistent storage if available
	if cm.storage != nil {
		err := cm.storage.DeleteContext(context.Background(), cm.shortTerm.ID)
		if err != nil {
			return fmt.Errorf("failed to clear persistent context: %w", err)
		}
	}

	return nil
}

// ExportContext exports the full context for backup or migration
func (cm *ContextManager) ExportContext() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	messages, summaries := cm.GetFullContext()

	exportData := map[string]interface{}{
		"conversation_id": cm.shortTerm.ID,
		"created_at":      cm.shortTerm.CreatedAt,
		"updated_at":      cm.shortTerm.UpdatedAt,
		"messages":        messages,
		"summaries":       summaries,
		"stats":           cm.GetStats(),
	}

	return json.MarshalIndent(exportData, "", "  ")
}

// ImportContext imports context data from backup or migration
func (cm *ContextManager) ImportContext(data []byte) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var importData struct {
		ConversationID string       `json:"conversation_id"`
		CreatedAt      time.Time    `json:"created_at"`
		UpdatedAt      time.Time    `json:"updated_at"`
		Messages       []*Message   `json:"messages"`
		Summaries      []*Summary   `json:"summaries"`
		Stats          ContextStats `json:"stats"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("failed to unmarshal import data: %w", err)
	}

	// Import messages to short-term memory
	cm.shortTerm.Clear()
	for _, msg := range importData.Messages {
		cm.shortTerm.AddMessage(msg)
	}

	// Import summaries to long-term memory
	summaryData, err := json.Marshal(importData.Summaries)
	if err != nil {
		return fmt.Errorf("failed to marshal summaries for import: %w", err)
	}

	if err := cm.longTerm.ImportMemory(summaryData); err != nil {
		return fmt.Errorf("failed to import summaries: %w", err)
	}

	return nil
}

// backupContext periodically backs up the context to persistent storage
func (cm *ContextManager) backupContext() {
	if !cm.backupEnabled || cm.storage == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data, err := cm.ExportContext()
	if err != nil {
		fmt.Printf("Failed to export context for backup: %v\n", err)
		return
	}

	if err := cm.storage.SaveContext(ctx, cm.shortTerm.ID, data); err != nil {
		fmt.Printf("Failed to backup context: %v\n", err)
	}
}

// calculateMemoryUsage estimates the memory usage of the context manager
func (cm *ContextManager) calculateMemoryUsage() int64 {
	var usage int64

	// Estimate short-term memory usage
	messages := cm.shortTerm.GetRecentMessages(0)
	for _, msg := range messages {
		usage += int64(len(msg.ID) + len(msg.Role) + len(msg.Content) + 100) // 100 bytes overhead estimate
	}

	// Estimate long-term memory usage
	summaries := cm.longTerm.GetAllSummaries()
	for _, summary := range summaries {
		usage += int64(len(summary.ID) + len(summary.Content) + 200) // 200 bytes overhead estimate
		for _, topic := range summary.Topics {
			usage += int64(len(topic))
		}
		for _, point := range summary.KeyPoints {
			usage += int64(len(point))
		}
	}

	return usage
}

// Helper functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
