package context

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"llm-verifier/llmverifier"
)

// Summary represents a compressed summary of conversation content
type Summary struct {
	ID           string    `json:"id"`
	Content      string    `json:"content"`
	Topics       []string  `json:"topics"`
	KeyPoints    []string  `json:"key_points"`
	Importance   float64   `json:"importance"` // 0.0 to 1.0
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// LongTermMemory manages long-term conversation memory with summarization
type LongTermMemory struct {
	summaries              []*Summary
	maxSummaries           int
	summarizationThreshold int // Number of messages before creating a summary
	verifier               *llmverifier.Verifier
	mu                     sync.RWMutex
}

// NewLongTermMemory creates a new long-term memory system
func NewLongTermMemory(maxSummaries int, summarizationThreshold int, verifier *llmverifier.Verifier) *LongTermMemory {
	return &LongTermMemory{
		summaries:              make([]*Summary, 0),
		maxSummaries:           maxSummaries,
		summarizationThreshold: summarizationThreshold,
		verifier:               verifier,
	}
}

// AddMessages processes messages and creates summaries when threshold is reached
func (ltm *LongTermMemory) AddMessages(messages []*Message) error {
	if len(messages) < ltm.summarizationThreshold {
		return nil // Not enough messages to summarize
	}

	// Create summary from messages
	summary, err := ltm.createSummary(messages)
	if err != nil {
		return fmt.Errorf("failed to create summary: %w", err)
	}

	ltm.mu.Lock()
	defer ltm.mu.Unlock()

	// Add summary
	ltm.summaries = append(ltm.summaries, summary)

	// Maintain maximum number of summaries
	if len(ltm.summaries) > ltm.maxSummaries {
		// Remove oldest summaries (lowest importance first)
		ltm.consolidateSummaries()
	}

	return nil
}

// createSummary generates a summary from a batch of messages
func (ltm *LongTermMemory) createSummary(messages []*Message) (*Summary, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to summarize")
	}

	// Prepare messages for summarization
	var messageTexts []string
	startTime := messages[0].Timestamp
	endTime := messages[0].Timestamp

	for _, msg := range messages {
		messageTexts = append(messageTexts, fmt.Sprintf("%s: %s", msg.Role, msg.Content))
		if msg.Timestamp.Before(startTime) {
			startTime = msg.Timestamp
		}
		if msg.Timestamp.After(endTime) {
			endTime = msg.Timestamp
		}
	}

	conversationText := strings.Join(messageTexts, "\n")

	// TODO: Use LLM to generate summary from conversation text
	// For now, create a basic summary

	// This would normally call the LLM, but for now we'll create a mock summary
	summary := &Summary{
		ID:           fmt.Sprintf("summary_%d", time.Now().UnixNano()),
		Content:      "Conversation summary: " + conversationText[:min(200, len(conversationText))],
		Topics:       []string{"general", "discussion"},
		KeyPoints:    []string{"Key point 1", "Key point 2"},
		Importance:   0.5,
		StartTime:    startTime,
		EndTime:      endTime,
		MessageCount: len(messages),
		CreatedAt:    time.Now(),
	}

	return summary, nil
}

// GetRelevantSummaries returns summaries relevant to a query
func (ltm *LongTermMemory) GetRelevantSummaries(query string, limit int) []*Summary {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()

	var relevant []*Summary

	// Simple relevance scoring based on topic and content matching
	for _, summary := range ltm.summaries {
		score := ltm.calculateRelevanceScore(summary, query)
		if score > 0.1 { // Minimum relevance threshold
			relevant = append(relevant, summary)
		}
	}

	// Sort by relevance (would need more sophisticated scoring)
	// For now, just return the most recent relevant summaries
	if len(relevant) > limit {
		relevant = relevant[len(relevant)-limit:]
	}

	return relevant
}

// calculateRelevanceScore calculates how relevant a summary is to a query
func (ltm *LongTermMemory) calculateRelevanceScore(summary *Summary, query string) float64 {
	score := 0.0
	queryLower := strings.ToLower(query)

	// Check topics
	for _, topic := range summary.Topics {
		if strings.Contains(strings.ToLower(topic), queryLower) {
			score += 0.3
		}
	}

	// Check content
	if strings.Contains(strings.ToLower(summary.Content), queryLower) {
		score += 0.4
	}

	// Check key points
	for _, point := range summary.KeyPoints {
		if strings.Contains(strings.ToLower(point), queryLower) {
			score += 0.2
		}
	}

	// Recency boost (newer summaries are more relevant)
	daysSince := time.Since(summary.CreatedAt).Hours() / 24
	recencyBoost := 1.0 / (1.0 + daysSince/30.0) // Decay over 30 days
	score *= recencyBoost

	return score
}

// consolidateSummaries reduces the number of summaries by merging less important ones
func (ltm *LongTermMemory) consolidateSummaries() {
	if len(ltm.summaries) <= ltm.maxSummaries {
		return
	}

	// Sort by importance (keep most important)
	// This is a simplified version - in practice, you'd want more sophisticated merging
	keepCount := ltm.maxSummaries
	if keepCount > len(ltm.summaries) {
		keepCount = len(ltm.summaries)
	}

	// For now, just keep the most recent summaries
	ltm.summaries = ltm.summaries[len(ltm.summaries)-keepCount:]
}

// GetAllSummaries returns all stored summaries
func (ltm *LongTermMemory) GetAllSummaries() []*Summary {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()

	// Return copies to prevent external modification
	summaries := make([]*Summary, len(ltm.summaries))
	for i, summary := range ltm.summaries {
		summaryCopy := *summary
		summaries[i] = &summaryCopy
	}
	return summaries
}

// Clear removes all summaries
func (ltm *LongTermMemory) Clear() {
	ltm.mu.Lock()
	defer ltm.mu.Unlock()
	ltm.summaries = make([]*Summary, 0)
}

// GetMemoryStats returns statistics about the long-term memory
func (ltm *LongTermMemory) GetMemoryStats() map[string]interface{} {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()

	totalMessages := 0
	totalImportance := 0.0
	oldestTime := time.Now()
	newestTime := time.Time{}

	for _, summary := range ltm.summaries {
		totalMessages += summary.MessageCount
		totalImportance += summary.Importance

		if summary.CreatedAt.Before(oldestTime) {
			oldestTime = summary.CreatedAt
		}
		if summary.CreatedAt.After(newestTime) {
			newestTime = summary.CreatedAt
		}
	}

	avgImportance := 0.0
	if len(ltm.summaries) > 0 {
		avgImportance = totalImportance / float64(len(ltm.summaries))
	}

	return map[string]interface{}{
		"summary_count":      len(ltm.summaries),
		"max_summaries":      ltm.maxSummaries,
		"total_messages":     totalMessages,
		"average_importance": avgImportance,
		"oldest_summary":     oldestTime,
		"newest_summary":     newestTime,
		"memory_span_days":   newestTime.Sub(oldestTime).Hours() / 24,
	}
}

// ExportMemory exports the memory as JSON for persistence
func (ltm *LongTermMemory) ExportMemory() ([]byte, error) {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()

	return json.MarshalIndent(ltm.summaries, "", "  ")
}

// ImportMemory imports memory from JSON
func (ltm *LongTermMemory) ImportMemory(data []byte) error {
	var summaries []*Summary
	if err := json.Unmarshal(data, &summaries); err != nil {
		return err
	}

	ltm.mu.Lock()
	defer ltm.mu.Unlock()

	ltm.summaries = summaries
	return nil
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
