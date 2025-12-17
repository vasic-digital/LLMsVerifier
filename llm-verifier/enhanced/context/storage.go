package context

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// FileSystemStorage implements ContextStorage using local filesystem
type FileSystemStorage struct {
	basePath string
}

// NewFileSystemStorage creates a new filesystem-based storage
func NewFileSystemStorage(basePath string) (*FileSystemStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return &FileSystemStorage{
		basePath: basePath,
	}, nil
}

// SaveContext saves context to a file
func (fs *FileSystemStorage) SaveContext(ctx context.Context, conversationID string, data []byte) error {
	filename := filepath.Join(fs.basePath, fmt.Sprintf("%s.json", conversationID))

	// Write to temporary file first
	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename to final filename (atomic operation)
	if err := os.Rename(tempFile, filename); err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// LoadContext loads context from a file
func (fs *FileSystemStorage) LoadContext(ctx context.Context, conversationID string) ([]byte, error) {
	filename := filepath.Join(fs.basePath, fmt.Sprintf("%s.json", conversationID))

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("conversation not found: %s", conversationID)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// DeleteContext deletes a context file
func (fs *FileSystemStorage) DeleteContext(ctx context.Context, conversationID string) error {
	filename := filepath.Join(fs.basePath, fmt.Sprintf("%s.json", conversationID))

	err := os.Remove(filename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// ListConversations returns a list of all conversation IDs
func (fs *FileSystemStorage) ListConversations(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var conversations []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			conversationID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
			conversations = append(conversations, conversationID)
		}
	}

	return conversations, nil
}

// DatabaseStorage implements ContextStorage using SQL database
type DatabaseStorage struct {
	db *sql.DB
}

// NewDatabaseStorage creates a new database-based storage
func NewDatabaseStorage(db *sql.DB) (*DatabaseStorage, error) {
	// Create table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS conversation_contexts (
		id VARCHAR(255) PRIMARY KEY,
		data JSONB NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_conversation_contexts_updated_at 
	ON conversation_contexts(updated_at);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &DatabaseStorage{
		db: db,
	}, nil
}

// SaveContext saves context to the database
func (ds *DatabaseStorage) SaveContext(ctx context.Context, conversationID string, data []byte) error {
	query := `
	INSERT INTO conversation_contexts (id, data, updated_at)
	VALUES ($1, $2, CURRENT_TIMESTAMP)
	ON CONFLICT (id) 
	DO UPDATE SET 
		data = EXCLUDED.data,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := ds.db.ExecContext(ctx, query, conversationID, string(data))
	if err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return nil
}

// LoadContext loads context from the database
func (ds *DatabaseStorage) LoadContext(ctx context.Context, conversationID string) ([]byte, error) {
	var data string
	query := "SELECT data FROM conversation_contexts WHERE id = $1"

	err := ds.db.QueryRowContext(ctx, query, conversationID).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conversation not found: %s", conversationID)
		}
		return nil, fmt.Errorf("failed to query context: %w", err)
	}

	return []byte(data), nil
}

// DeleteContext deletes context from the database
func (ds *DatabaseStorage) DeleteContext(ctx context.Context, conversationID string) error {
	query := "DELETE FROM conversation_contexts WHERE id = $1"

	_, err := ds.db.ExecContext(ctx, query, conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete context: %w", err)
	}

	return nil
}

// ListConversations returns a list of all conversation IDs
func (ds *DatabaseStorage) ListConversations(ctx context.Context) ([]string, error) {
	query := "SELECT id FROM conversation_contexts ORDER BY updated_at DESC"

	rows, err := ds.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}
	defer rows.Close()

	var conversations []string
	for rows.Next() {
		var conversationID string
		if err := rows.Scan(&conversationID); err != nil {
			return nil, fmt.Errorf("failed to scan conversation ID: %w", err)
		}
		conversations = append(conversations, conversationID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversations: %w", err)
	}

	return conversations, nil
}

// HybridStorage implements ContextStorage with multiple backends for redundancy
type HybridStorage struct {
	primary  ContextStorage
	replicas []ContextStorage
}

// NewHybridStorage creates a new hybrid storage with primary and replica backends
func NewHybridStorage(primary ContextStorage, replicas ...ContextStorage) *HybridStorage {
	return &HybridStorage{
		primary:  primary,
		replicas: replicas,
	}
}

// SaveContext saves to primary and replicates to all replicas
func (hs *HybridStorage) SaveContext(ctx context.Context, conversationID string, data []byte) error {
	// Save to primary first
	if err := hs.primary.SaveContext(ctx, conversationID, data); err != nil {
		return fmt.Errorf("failed to save to primary storage: %w", err)
	}

	// Replicate to all replicas asynchronously
	for i, replica := range hs.replicas {
		go func(index int, r ContextStorage) {
			if err := r.SaveContext(ctx, conversationID, data); err != nil {
				log.Printf("Failed to replicate to replica %d: %v", index, err)
			}
		}(i, replica)
	}

	return nil
}

// LoadContext attempts to load from primary, falls back to replicas if needed
func (hs *HybridStorage) LoadContext(ctx context.Context, conversationID string) ([]byte, error) {
	// Try primary first
	data, err := hs.primary.LoadContext(ctx, conversationID)
	if err == nil {
		return data, nil
	}

	// Try replicas
	for i, replica := range hs.replicas {
		data, err := replica.LoadContext(ctx, conversationID)
		if err == nil {
			// Successfully loaded from replica, restore to primary
			go func() {
				if restoreErr := hs.primary.SaveContext(ctx, conversationID, data); restoreErr != nil {
					log.Printf("Failed to restore to primary from replica %d: %v", i, restoreErr)
				}
			}()
			return data, nil
		}
		log.Printf("Failed to load from replica %d: %v", i, err)
	}

	return nil, fmt.Errorf("failed to load from all storage backends")
}

// DeleteContext deletes from all backends
func (hs *HybridStorage) DeleteContext(ctx context.Context, conversationID string) error {
	var lastError error

	// Delete from primary
	if err := hs.primary.DeleteContext(ctx, conversationID); err != nil {
		lastError = fmt.Errorf("failed to delete from primary: %w", err)
	}

	// Delete from all replicas
	for i, replica := range hs.replicas {
		if err := replica.DeleteContext(ctx, conversationID); err != nil {
			log.Printf("Failed to delete from replica %d: %v", i, err)
			lastError = fmt.Errorf("failed to delete from replica %d: %w", i, err)
		}
	}

	return lastError
}

// ListConversations returns from primary storage
func (hs *HybridStorage) ListConversations(ctx context.Context) ([]string, error) {
	return hs.primary.ListConversations(ctx)
}

// StorageConfig holds configuration for different storage backends
type StorageConfig struct {
	Type     string                 `yaml:"type"` // "filesystem", "database", "hybrid"
	Settings map[string]interface{} `yaml:"settings"`
}

// NewStorageFromConfig creates storage backend based on configuration
func NewStorageFromConfig(config StorageConfig, db *sql.DB) (ContextStorage, error) {
	switch config.Type {
	case "filesystem":
		basePath, ok := config.Settings["base_path"].(string)
		if !ok {
			return nil, fmt.Errorf("base_path required for filesystem storage")
		}
		return NewFileSystemStorage(basePath)

	case "database":
		if db == nil {
			return nil, fmt.Errorf("database connection required for database storage")
		}
		return NewDatabaseStorage(db)

	case "hybrid":
		primaryType, ok := config.Settings["primary"].(string)
		if !ok {
			return nil, fmt.Errorf("primary storage type required for hybrid storage")
		}

		var primary ContextStorage
		var err error

		// Create primary storage
		primaryConfig := StorageConfig{
			Type:     primaryType,
			Settings: config.Settings["primary_settings"].(map[string]interface{}),
		}
		primary, err = NewStorageFromConfig(primaryConfig, db)
		if err != nil {
			return nil, fmt.Errorf("failed to create primary storage: %w", err)
		}

		// Create replica storages
		var replicas []ContextStorage
		if replicaConfigs, exists := config.Settings["replicas"].([]interface{}); exists {
			for i, replicaConfig := range replicaConfigs {
				replicaMap := replicaConfig.(map[string]interface{})
				replicaType := replicaMap["type"].(string)
				replicaSettings := replicaMap["settings"].(map[string]interface{})

				replica := StorageConfig{
					Type:     replicaType,
					Settings: replicaSettings,
				}

				replicaStorage, err := NewStorageFromConfig(replica, db)
				if err != nil {
					return nil, fmt.Errorf("failed to create replica %d: %w", i, err)
				}
				replicas = append(replicas, replicaStorage)
			}
		}

		return NewHybridStorage(primary, replicas...), nil

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}
}
