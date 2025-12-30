package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// CreateAPIKey creates a new API key for a user
func (d *Database) CreateAPIKey(apiKey *APIKey) (string, error) {
	// Generate a random API key
	rawKey, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the API key for storage
	keyHash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}

	// Convert scopes to JSON
	var scopesJSON []byte
	if apiKey.Scopes != nil {
		scopesJSON, err = json.Marshal(apiKey.Scopes)
		if err != nil {
			return "", fmt.Errorf("failed to marshal scopes: %w", err)
		}
	}

	query := `
		INSERT INTO api_keys (
			user_id, name, key_hash, scopes, expires_at, is_active
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := d.conn.Exec(query,
		apiKey.UserID,
		apiKey.Name,
		string(keyHash),
		scopesJSON,
		apiKey.ExpiresAt,
		apiKey.IsActive,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return "", fmt.Errorf("failed to get last insert ID: %w", err)
	}

	apiKey.ID = id
	apiKey.KeyHash = string(keyHash)
	return rawKey, nil
}

// GetAPIKey retrieves an API key by ID
func (d *Database) GetAPIKey(id int64) (*APIKey, error) {
	query := `
		SELECT id, user_id, name, key_hash, scopes, expires_at, last_used,
		       is_active, created_at
		FROM api_keys WHERE id = ?
	`

	row := d.conn.QueryRow(query, id)
	return d.scanAPIKey(row)
}

// GetAPIKeyByHash retrieves an API key by its hash
func (d *Database) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
	query := `
		SELECT id, user_id, name, key_hash, scopes, expires_at, last_used,
		       is_active, created_at
		FROM api_keys WHERE key_hash = ?
	`

	row := d.conn.QueryRow(query, keyHash)
	return d.scanAPIKey(row)
}

// UpdateAPIKey updates an existing API key
func (d *Database) UpdateAPIKey(apiKey *APIKey) error {
	// Convert scopes to JSON
	var scopesJSON []byte
	if apiKey.Scopes != nil {
		var err error
		scopesJSON, err = json.Marshal(apiKey.Scopes)
		if err != nil {
			return fmt.Errorf("failed to marshal scopes: %w", err)
		}
	}

	query := `
		UPDATE api_keys SET
			name = ?,
			scopes = ?,
			expires_at = ?,
			is_active = ?,
			last_used = ?
		WHERE id = ?
	`

	_, err := d.conn.Exec(query,
		apiKey.Name,
		scopesJSON,
		apiKey.ExpiresAt,
		apiKey.IsActive,
		apiKey.LastUsed,
		apiKey.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	return nil
}

// DeleteAPIKey deletes an API key by ID
func (d *Database) DeleteAPIKey(id int64) error {
	query := `DELETE FROM api_keys WHERE id = ?`
	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	return nil
}

// ListAPIKeys retrieves API keys for a user with optional filtering
func (d *Database) ListAPIKeys(userID int64, filters map[string]interface{}) ([]*APIKey, error) {
	query := `
		SELECT id, user_id, name, key_hash, scopes, expires_at, last_used,
		       is_active, created_at
		FROM api_keys
		WHERE user_id = ?
	`

	args := []interface{}{userID}

	// Apply filters
	if isActive, ok := filters["is_active"]; ok {
		query += " AND is_active = ?"
		args = append(args, isActive)
	}

	if expired, ok := filters["expired"]; ok && expired.(bool) {
		query += " AND (expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP)"
	} else if expired, ok := filters["expired"]; ok && !expired.(bool) {
		query += " AND (expires_at IS NULL OR expires_at >= CURRENT_TIMESTAMP)"
	}

	// Add pagination
	limit := 50
	offset := 0
	if l, ok := filters["limit"]; ok {
		limit = l.(int)
	}
	if o, ok := filters["offset"]; ok {
		offset = o.(int)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*APIKey
	for rows.Next() {
		apiKey, err := d.scanAPIKeyFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

// UpdateAPIKeyLastUsed updates the last used timestamp for an API key
func (d *Database) UpdateAPIKeyLastUsed(apiKeyID int64) error {
	query := `UPDATE api_keys SET last_used = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.conn.Exec(query, apiKeyID)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}
	return nil
}

// VerifyAPIKey verifies an API key and returns the associated user
func (d *Database) VerifyAPIKey(apiKey string) (*User, *APIKey, error) {
	// Get all active API keys
	query := `
		SELECT id, user_id, name, key_hash, scopes, expires_at, last_used,
		       is_active, created_at
		FROM api_keys
		WHERE is_active = 1
		AND (expires_at IS NULL OR expires_at >= CURRENT_TIMESTAMP)
	`

	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query API keys: %w", err)
	}

	// Collect all API keys first, then close the rows cursor
	// This is necessary to avoid nested query issues with SQLite
	var apiKeys []*APIKey
	for rows.Next() {
		dbAPIKey, err := d.scanAPIKeyFromRows(rows)
		if err != nil {
			continue
		}
		apiKeys = append(apiKeys, dbAPIKey)
	}
	rows.Close()

	// Check each API key
	for _, dbAPIKey := range apiKeys {
		// Compare the provided key with the stored hash
		err = bcrypt.CompareHashAndPassword([]byte(dbAPIKey.KeyHash), []byte(apiKey))
		if err == nil {
			// Key matches, get the user
			user, err := d.GetUser(dbAPIKey.UserID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get user: %w", err)
			}

			// Update last used timestamp
			if err := d.UpdateAPIKeyLastUsed(dbAPIKey.ID); err != nil {
				// Log but don't fail authentication
				fmt.Printf("Failed to update API key last used: %v\n", err)
			}

			return user, dbAPIKey, nil
		}
	}

	return nil, nil, fmt.Errorf("invalid API key")
}

// scanAPIKey scans an API key from a sql.Row
func (d *Database) scanAPIKey(row *sql.Row) (*APIKey, error) {
	var apiKey APIKey
	var scopesJSON []byte
	var expiresAt sql.NullTime
	var lastUsed sql.NullTime

	err := row.Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.Name,
		&apiKey.KeyHash,
		&scopesJSON,
		&expiresAt,
		&lastUsed,
		&apiKey.IsActive,
		&apiKey.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to scan API key: %w", err)
	}

	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}
	if lastUsed.Valid {
		apiKey.LastUsed = &lastUsed.Time
	}

	if len(scopesJSON) > 0 {
		err = json.Unmarshal(scopesJSON, &apiKey.Scopes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
		}
	}

	return &apiKey, nil
}

// scanAPIKeyFromRows scans an API key from sql.Rows
func (d *Database) scanAPIKeyFromRows(rows *sql.Rows) (*APIKey, error) {
	var apiKey APIKey
	var scopesJSON []byte
	var expiresAt sql.NullTime
	var lastUsed sql.NullTime

	err := rows.Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.Name,
		&apiKey.KeyHash,
		&scopesJSON,
		&expiresAt,
		&lastUsed,
		&apiKey.IsActive,
		&apiKey.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan API key: %w", err)
	}

	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}
	if lastUsed.Valid {
		apiKey.LastUsed = &lastUsed.Time
	}

	if len(scopesJSON) > 0 {
		err = json.Unmarshal(scopesJSON, &apiKey.Scopes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
		}
	}

	return &apiKey, nil
}

// generateSecureToken generates a secure random token
func generateSecureToken(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)

	// Use crypto/rand for secure random generation
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes), nil
}
