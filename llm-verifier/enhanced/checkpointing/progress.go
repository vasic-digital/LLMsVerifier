package checkpointing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	ctxt "llm-verifier/enhanced/context"
)

// Checkpoint represents a snapshot of agent state
type Checkpoint struct {
	ID          string                 `json:"id"`
	AgentID     string                 `json:"agent_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Progress    AgentProgress          `json:"progress"`
	MemoryState MemoryState            `json:"memory_state"`
	OpenFiles   []OpenFile             `json:"open_files"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AgentProgress tracks the progress of an agent task
type AgentProgress struct {
	TaskID         string    `json:"task_id"`
	TaskName       string    `json:"task_name"`
	Status         string    `json:"status"`   // running, completed, failed, paused
	Progress       float64   `json:"progress"` // 0.0 to 1.0
	Step           string    `json:"current_step"`
	TotalSteps     int       `json:"total_steps"`
	CompletedSteps int       `json:"completed_steps"`
	StartTime      time.Time `json:"start_time"`
	LastUpdate     time.Time `json:"last_update"`
	Error          string    `json:"error,omitempty"`
}

// MemoryState represents the state of agent memory
type MemoryState struct {
	ShortTermConversations map[string][]ctxt.Message `json:"short_term_conversations"`
	LongTermSummaries      []*ctxt.Summary           `json:"long_term_summaries"`
	WorkingMemory          map[string]interface{}    `json:"working_memory"`
}

// OpenFile represents an open file being worked on
type OpenFile struct {
	Path         string    `json:"path"`
	Content      string    `json:"content"`
	CursorPos    int       `json:"cursor_position"`
	LastModified time.Time `json:"last_modified"`
	IsModified   bool      `json:"is_modified"`
}

// CheckpointCache provides LRU caching for checkpoints
type CheckpointCache struct {
	cache   map[string]*Checkpoint
	maxSize int
	mu      sync.RWMutex
}

// NewCheckpointCache creates a new checkpoint cache
func NewCheckpointCache(maxSize int) *CheckpointCache {
	return &CheckpointCache{
		cache:   make(map[string]*Checkpoint),
		maxSize: maxSize,
	}
}

// Get retrieves a checkpoint from cache
func (cc *CheckpointCache) Get(id string) (*Checkpoint, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	checkpoint, exists := cc.cache[id]
	return checkpoint, exists
}

// Put stores a checkpoint in cache with LRU eviction
func (cc *CheckpointCache) Put(checkpoint *Checkpoint) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// If cache is full, remove oldest entry (simple implementation)
	if len(cc.cache) >= cc.maxSize {
		for key := range cc.cache {
			delete(cc.cache, key)
			break // Remove just one for simplicity
		}
	}

	cc.cache[checkpoint.ID] = checkpoint
}

// Clear removes all entries from cache
func (cc *CheckpointCache) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.cache = make(map[string]*Checkpoint)
}

// CheckpointManager manages agent checkpoints
type CheckpointManager struct {
	checkpoints    map[string]*Checkpoint
	storagePath    string
	maxCheckpoints int
	cache          *CheckpointCache
	mu             sync.RWMutex
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(storagePath string, maxCheckpoints int) *CheckpointManager {
	return &CheckpointManager{
		checkpoints:    make(map[string]*Checkpoint),
		storagePath:    storagePath,
		maxCheckpoints: maxCheckpoints,
		cache:          NewCheckpointCache(100), // Cache up to 100 checkpoints
	}
}

// CreateCheckpoint creates a new checkpoint for an agent
func (cm *CheckpointManager) CreateCheckpoint(agentID string, progress AgentProgress, memoryState MemoryState, openFiles []OpenFile) (*Checkpoint, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	checkpoint := &Checkpoint{
		ID:          fmt.Sprintf("chk_%s_%d", agentID, time.Now().UnixNano()),
		AgentID:     agentID,
		Timestamp:   time.Now(),
		Progress:    progress,
		MemoryState: memoryState,
		OpenFiles:   openFiles,
		Metadata:    make(map[string]interface{}),
	}

	// Store checkpoint
	cm.checkpoints[checkpoint.ID] = checkpoint

	// Save to disk
	if err := cm.saveCheckpoint(checkpoint); err != nil {
		return nil, fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// Cleanup old checkpoints
	cm.cleanupOldCheckpoints(agentID)

	return checkpoint, nil
}

// GetLatestCheckpoint gets the most recent checkpoint for an agent
func (cm *CheckpointManager) GetLatestCheckpoint(agentID string) (*Checkpoint, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var latest *Checkpoint
	var latestTime time.Time

	for _, checkpoint := range cm.checkpoints {
		if checkpoint.AgentID == agentID && checkpoint.Timestamp.After(latestTime) {
			latest = checkpoint
			latestTime = checkpoint.Timestamp
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no checkpoint found for agent %s", agentID)
	}

	// Cache the result for faster future access
	cm.cache.Put(latest)

	return latest, nil
}

// GetCheckpointsForAgent gets all checkpoints for an agent
func (cm *CheckpointManager) GetCheckpointsForAgent(agentID string) []*Checkpoint {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var agentCheckpoints []*Checkpoint
	for _, checkpoint := range cm.checkpoints {
		if checkpoint.AgentID == agentID {
			agentCheckpoints = append(agentCheckpoints, checkpoint)
		}
	}

	return agentCheckpoints
}

// RestoreFromCheckpoint restores agent state from a checkpoint
func (cm *CheckpointManager) RestoreFromCheckpoint(checkpointID string) (*Checkpoint, error) {
	// First check cache
	if cached, exists := cm.cache.Get(checkpointID); exists {
		// Return a copy to prevent external modification
		checkpointCopy := *cached
		return &checkpointCopy, nil
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	checkpoint, exists := cm.checkpoints[checkpointID]
	if !exists {
		return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	// Cache for future use
	cm.cache.Put(checkpoint)

	// Return a copy to prevent external modification
	checkpointCopy := *checkpoint
	return &checkpointCopy, nil
}

// UpdateProgress updates the progress of an agent
func (cm *CheckpointManager) UpdateProgress(agentID string, progress AgentProgress) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Find the latest checkpoint for this agent
	var latestCheckpoint *Checkpoint
	var latestTime time.Time

	for _, checkpoint := range cm.checkpoints {
		if checkpoint.AgentID == agentID && checkpoint.Timestamp.After(latestTime) {
			latestCheckpoint = checkpoint
			latestTime = checkpoint.Timestamp
		}
	}

	if latestCheckpoint == nil {
		return fmt.Errorf("no checkpoint found for agent %s", agentID)
	}

	// Update progress
	latestCheckpoint.Progress = progress
	latestCheckpoint.Timestamp = time.Now()

	// Save updated checkpoint
	return cm.saveCheckpoint(latestCheckpoint)
}

// AddOpenFile adds an open file to the current checkpoint
func (cm *CheckpointManager) AddOpenFile(agentID, filePath, content string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Find the latest checkpoint for this agent
	var latestCheckpoint *Checkpoint
	var latestTime time.Time

	for _, checkpoint := range cm.checkpoints {
		if checkpoint.AgentID == agentID && checkpoint.Timestamp.After(latestTime) {
			latestCheckpoint = checkpoint
			latestTime = checkpoint.Timestamp
		}
	}

	if latestCheckpoint == nil {
		return fmt.Errorf("no checkpoint found for agent %s", agentID)
	}

	// Check if file already exists
	fileExists := false
	for i, file := range latestCheckpoint.OpenFiles {
		if file.Path == filePath {
			// Update existing file
			latestCheckpoint.OpenFiles[i].Content = content
			latestCheckpoint.OpenFiles[i].LastModified = time.Now()
			latestCheckpoint.OpenFiles[i].IsModified = true
			fileExists = true
			break
		}
	}

	if !fileExists {
		// Add new file
		openFile := OpenFile{
			Path:         filePath,
			Content:      content,
			CursorPos:    0,
			LastModified: time.Now(),
			IsModified:   true,
		}
		latestCheckpoint.OpenFiles = append(latestCheckpoint.OpenFiles, openFile)
	}

	latestCheckpoint.Timestamp = time.Now()
	return cm.saveCheckpoint(latestCheckpoint)
}

// RemoveOpenFile removes an open file from the current checkpoint
func (cm *CheckpointManager) RemoveOpenFile(agentID, filePath string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Find the latest checkpoint for this agent
	var latestCheckpoint *Checkpoint
	var latestTime time.Time

	for _, checkpoint := range cm.checkpoints {
		if checkpoint.AgentID == agentID && checkpoint.Timestamp.After(latestTime) {
			latestCheckpoint = checkpoint
			latestTime = checkpoint.Timestamp
		}
	}

	if latestCheckpoint == nil {
		return fmt.Errorf("no checkpoint found for agent %s", agentID)
	}

	// Remove file
	for i, file := range latestCheckpoint.OpenFiles {
		if file.Path == filePath {
			latestCheckpoint.OpenFiles = append(latestCheckpoint.OpenFiles[:i], latestCheckpoint.OpenFiles[i+1:]...)
			latestCheckpoint.Timestamp = time.Now()
			return cm.saveCheckpoint(latestCheckpoint)
		}
	}

	return fmt.Errorf("file not found in checkpoint: %s", filePath)
}

// saveCheckpoint saves a checkpoint to disk
func (cm *CheckpointManager) saveCheckpoint(checkpoint *Checkpoint) error {
	// Ensure storage directory exists
	if err := os.MkdirAll(cm.storagePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Serialize checkpoint
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("%s.json", checkpoint.ID)
	filepath := filepath.Join(cm.storagePath, filename)

	if err := ioutil.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	return nil
}

// loadCheckpoints loads all checkpoints from disk
func (cm *CheckpointManager) loadCheckpoints() error {
	if _, err := os.Stat(cm.storagePath); os.IsNotExist(err) {
		return nil // Directory doesn't exist, no checkpoints to load
	}

	files, err := ioutil.ReadDir(cm.storagePath)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filepath := filepath.Join(cm.storagePath, file.Name())
		data, err := ioutil.ReadFile(filepath)
		if err != nil {
			continue // Skip files that can't be read
		}

		var checkpoint Checkpoint
		if err := json.Unmarshal(data, &checkpoint); err != nil {
			continue // Skip invalid files
		}

		cm.checkpoints[checkpoint.ID] = &checkpoint
	}

	return nil
}

// cleanupOldCheckpoints removes old checkpoints to maintain the maximum limit
func (cm *CheckpointManager) cleanupOldCheckpoints(agentID string) {
	var agentCheckpoints []*Checkpoint

	// Collect all checkpoints for this agent
	for _, checkpoint := range cm.checkpoints {
		if checkpoint.AgentID == agentID {
			agentCheckpoints = append(agentCheckpoints, checkpoint)
		}
	}

	// If we have too many, remove the oldest ones
	if len(agentCheckpoints) > cm.maxCheckpoints {
		// Sort by timestamp (oldest first)
		for i := 0; i < len(agentCheckpoints)-1; i++ {
			for j := i + 1; j < len(agentCheckpoints); j++ {
				if agentCheckpoints[i].Timestamp.After(agentCheckpoints[j].Timestamp) {
					agentCheckpoints[i], agentCheckpoints[j] = agentCheckpoints[j], agentCheckpoints[i]
				}
			}
		}

		// Remove excess checkpoints
		toRemove := len(agentCheckpoints) - cm.maxCheckpoints
		for i := 0; i < toRemove; i++ {
			delete(cm.checkpoints, agentCheckpoints[i].ID)

			// Delete file
			filename := fmt.Sprintf("%s.json", agentCheckpoints[i].ID)
			filepath := filepath.Join(cm.storagePath, filename)
			os.Remove(filepath)
		}
	}
}

// GetCheckpointStats returns statistics about checkpoints
func (cm *CheckpointManager) GetCheckpointStats() map[string]any {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := map[string]any{
		"total_checkpoints": len(cm.checkpoints),
		"storage_path":      cm.storagePath,
		"max_checkpoints":   cm.maxCheckpoints,
		"cache_size":        len(cm.cache.cache),
		"cache_max_size":    cm.cache.maxSize,
	}

	// Count checkpoints per agent
	agentCounts := make(map[string]int)
	for _, checkpoint := range cm.checkpoints {
		agentCounts[checkpoint.AgentID]++
	}
	stats["checkpoints_per_agent"] = agentCounts

	return stats
}

// ClearCache clears the checkpoint cache
func (cm *CheckpointManager) ClearCache() {
	cm.cache.Clear()
}

// SetCacheSize sets the maximum cache size
func (cm *CheckpointManager) SetCacheSize(maxSize int) {
	cm.cache.mu.Lock()
	defer cm.cache.mu.Unlock()
	cm.cache.maxSize = maxSize
}
