package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// CredentialManager handles secure storage and retrieval of credentials
type CredentialManager struct {
	encryptionKey []byte
	store         CredentialStore
}

// CredentialStore interface for different storage backends
type CredentialStore interface {
	Store(key string, value string) error
	Retrieve(key string) (string, error)
	Delete(key string) error
	List(prefix string) ([]string, error)
}

// EncryptedFileStore implements file-based encrypted credential storage
type EncryptedFileStore struct {
	filepath string
	data     map[string]string
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager(masterKey string, store CredentialStore) *CredentialManager {
	// Derive encryption key from master key
	hash := sha256.Sum256([]byte(masterKey))
	key := hash[:]

	return &CredentialManager{
		encryptionKey: key,
		store:         store,
	}
}

// StoreCredential securely stores a credential
func (cm *CredentialManager) StoreCredential(service, key, value string) error {
	// Create composite key
	compositeKey := fmt.Sprintf("%s:%s", service, key)

	// Encrypt the value
	encrypted, err := cm.encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt credential: %w", err)
	}

	// Store encrypted value
	return cm.store.Store(compositeKey, encrypted)
}

// RetrieveCredential securely retrieves a credential
func (cm *CredentialManager) RetrieveCredential(service, key string) (string, error) {
	// Create composite key
	compositeKey := fmt.Sprintf("%s:%s", service, key)

	// Retrieve encrypted value
	encrypted, err := cm.store.Retrieve(compositeKey)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve credential: %w", err)
	}

	// Decrypt the value
	decrypted, err := cm.decrypt(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt credential: %w", err)
	}

	return decrypted, nil
}

// DeleteCredential removes a credential
func (cm *CredentialManager) DeleteCredential(service, key string) error {
	compositeKey := fmt.Sprintf("%s:%s", service, key)
	return cm.store.Delete(compositeKey)
}

// ListCredentials lists all credentials for a service
func (cm *CredentialManager) ListCredentials(service string) ([]string, error) {
	keys, err := cm.store.List(service + ":")
	if err != nil {
		return nil, err
	}

	// Extract key names (remove service prefix)
	var result []string
	for _, key := range keys {
		if strings.HasPrefix(key, service+":") {
			result = append(result, strings.TrimPrefix(key, service+":"))
		}
	}

	return result, nil
}

// Encryption/Decryption methods
func (cm *CredentialManager) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (cm *CredentialManager) decrypt(encrypted string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// APIKeyMasker handles masking and protection of API keys in logs and responses
type APIKeyMasker struct {
	patterns []APIKeyPattern
}

type APIKeyPattern struct {
	Name         string
	Regex        string // Compiled regex pattern
	MaskChar     string
	VisibleStart int
	VisibleEnd   int
}

// NewAPIKeyMasker creates a new API key masker
func NewAPIKeyMasker() *APIKeyMasker {
	return &APIKeyMasker{
		patterns: []APIKeyPattern{
			{
				Name:         "OpenAI",
				Regex:        `sk-[a-zA-Z0-9]{48}`,
				MaskChar:     "*",
				VisibleStart: 3,
				VisibleEnd:   4,
			},
			{
				Name:         "Anthropic",
				Regex:        `sk-ant-[a-zA-Z0-9_-]{95,}`,
				MaskChar:     "*",
				VisibleStart: 7,
				VisibleEnd:   4,
			},
			{
				Name:         "Google AI",
				Regex:        `AIza[0-9A-Za-z-_]{35}`,
				MaskChar:     "*",
				VisibleStart: 4,
				VisibleEnd:   4,
			},
			{
				Name:         "Generic Bearer Token",
				Regex:        `Bearer [a-zA-Z0-9_-]{20,}`,
				MaskChar:     "*",
				VisibleStart: 7,
				VisibleEnd:   4,
			},
		},
	}
}

// MaskAPIKeys masks API keys in a string
func (akm *APIKeyMasker) MaskAPIKeys(input string) string {
	result := input

	// Apply each pattern
	for _, pattern := range akm.patterns {
		result = akm.maskPattern(result, pattern)
	}

	return result
}

// maskPattern applies a single masking pattern
func (akm *APIKeyMasker) maskPattern(input string, pattern APIKeyPattern) string {
	// Simple string replacement for demo
	// In real implementation, use regexp.ReplaceAllStringFunc
	return strings.ReplaceAll(input, pattern.Regex, akm.createMask(pattern))
}

// createMask creates a masked version of a key
func (akm *APIKeyMasker) createMask(pattern APIKeyPattern) string {
	// This is a simplified implementation
	// Real implementation would dynamically create masks based on matched content
	return pattern.Name + "_KEY_MASKED"
}

// AuditTrail provides comprehensive audit logging
type AuditTrail struct {
	logger *log.Logger
	store  AuditStore
}

type AuditStore interface {
	Store(entry AuditEntry) error
	Query(filters map[string]interface{}) ([]AuditEntry, error)
}

type AuditEntry struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	UserID     *string                `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	ResourceID string                 `json:"resource_id"`
	Method     string                 `json:"method"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Compliance ComplianceInfo         `json:"compliance"`
}

type ComplianceInfo struct {
	GDPRCompliant     bool     `json:"gdpr_compliant"`
	DataRetentionDays int      `json:"data_retention_days"`
	RequiredFields    []string `json:"required_fields"`
}

// NewAuditTrail creates a new audit trail
func NewAuditTrail(logger *log.Logger, store AuditStore) *AuditTrail {
	return &AuditTrail{
		logger: logger,
		store:  store,
	}
}

// LogRequest logs an API request
func (at *AuditTrail) LogRequest(r *http.Request, userID *string, success bool, errorMsg string) {
	entry := AuditEntry{
		ID:         generateAuditID(),
		Timestamp:  time.Now(),
		UserID:     userID,
		SessionID:  extractSessionID(r),
		Action:     r.Method,
		Resource:   r.URL.Path,
		ResourceID: extractResourceID(r.URL.Path),
		Method:     r.Method,
		IPAddress:  extractIPAddress(r),
		UserAgent:  r.Header.Get("User-Agent"),
		Success:    success,
		Error:      errorMsg,
		Details:    extractRequestDetails(r),
		Compliance: ComplianceInfo{
			GDPRCompliant:     true,
			DataRetentionDays: 2555, // 7 years
			RequiredFields:    []string{"user_id", "action", "resource", "timestamp"},
		},
	}

	// Store in audit store
	if err := at.store.Store(entry); err != nil {
		at.logger.Printf("Failed to store audit entry: %v", err)
	}

	// Log to standard logger
	at.logger.Printf("AUDIT: %s %s %s %s %v",
		entry.Action, entry.Resource, entry.IPAddress,
		entry.UserAgent, entry.Success)
}

// QueryAuditLogs queries audit logs with filters
func (at *AuditTrail) QueryAuditLogs(filters map[string]interface{}) ([]AuditEntry, error) {
	return at.store.Query(filters)
}

// RBACManager provides Role-Based Access Control
type RBACManager struct {
	roles       map[string]Role
	users       map[string][]string // userID -> []roleIDs
	permissions map[string]Permission
}

// Role represents a user role
type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"` // Permission IDs
}

// Permission represents a system permission
type Permission struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Resource    string   `json:"resource"`
	Action      string   `json:"action"`
	Conditions  []string `json:"conditions,omitempty"`
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager() *RBACManager {
	return &RBACManager{
		roles:       make(map[string]Role),
		users:       make(map[string][]string),
		permissions: make(map[string]Permission),
	}
}

// AddRole adds a new role
func (rbac *RBACManager) AddRole(role Role) {
	rbac.roles[role.ID] = role
}

// AddPermission adds a new permission
func (rbac *RBACManager) AddPermission(perm Permission) {
	rbac.permissions[perm.ID] = perm
}

// AssignRole assigns a role to a user
func (rbac *RBACManager) AssignRole(userID, roleID string) error {
	if _, exists := rbac.roles[roleID]; !exists {
		return fmt.Errorf("role not found: %s", roleID)
	}

	rbac.users[userID] = append(rbac.users[userID], roleID)
	return nil
}

// CheckPermission checks if a user has permission for an action
func (rbac *RBACManager) CheckPermission(userID, resource, action string, conditions map[string]interface{}) bool {
	userRoles := rbac.users[userID]
	if len(userRoles) == 0 {
		return false
	}

	for _, roleID := range userRoles {
		role, exists := rbac.roles[roleID]
		if !exists {
			continue
		}

		for _, permID := range role.Permissions {
			perm, exists := rbac.permissions[permID]
			if !exists {
				continue
			}

			if perm.Resource == resource && perm.Action == action {
				// Check conditions if any
				if len(perm.Conditions) > 0 {
					if rbac.checkConditions(conditions, perm.Conditions) {
						return true
					}
				} else {
					return true
				}
			}
		}
	}

	return false
}

// GetUserPermissions returns all permissions for a user
func (rbac *RBACManager) GetUserPermissions(userID string) []Permission {
	userRoles := rbac.users[userID]
	var permissions []Permission

	for _, roleID := range userRoles {
		role, exists := rbac.roles[roleID]
		if !exists {
			continue
		}

		for _, permID := range role.Permissions {
			if perm, exists := rbac.permissions[permID]; exists {
				permissions = append(permissions, perm)
			}
		}
	}

	return permissions
}

func (rbac *RBACManager) checkConditions(requestConditions map[string]interface{}, requiredConditions []string) bool {
	// Simplified condition checking
	// Real implementation would have more sophisticated logic
	for _, condition := range requiredConditions {
		if _, exists := requestConditions[condition]; !exists {
			return false
		}
	}
	return true
}

// Helper functions
func generateAuditID() string {
	return fmt.Sprintf("audit_%d", time.Now().UnixNano())
}

func extractSessionID(r *http.Request) string {
	// Extract from JWT token or session cookie
	// Simplified implementation
	return "session_" + fmt.Sprintf("%d", time.Now().Unix())
}

func extractResourceID(path string) string {
	// Extract ID from paths like /api/v1/models/123
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractIPAddress(r *http.Request) string {
	// Try X-Forwarded-For first, then X-Real-IP, then RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

func extractRequestDetails(r *http.Request) map[string]interface{} {
	return map[string]interface{}{
		"query_params": r.URL.Query(),
		"headers":      sanitizeHeaders(r.Header),
		"method":       r.Method,
		"path":         r.URL.Path,
	}
}

func sanitizeHeaders(headers map[string][]string) map[string][]string {
	sanitized := make(map[string][]string)
	for k, v := range headers {
		if strings.ToLower(k) == "authorization" {
			sanitized[k] = []string{"***REDACTED***"}
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}
