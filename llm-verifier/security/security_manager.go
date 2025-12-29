package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

// SecurityManager handles all security-related operations
type SecurityManager struct {
	apiKeys       map[string]*APIKeyInfo
	sessions      map[string]*SessionInfo
	rateLimiters  map[string]*rate.Limiter
	blockedIPs    map[string]time.Time
	encryptionKey []byte
}

// APIKeyInfo stores information about API keys
type APIKeyInfo struct {
	Key         string
	UserID      string
	Permissions []string
	CreatedAt   time.Time
	LastUsedAt  time.Time
	ExpiresAt   *time.Time
	RateLimit   int // requests per minute
}

// SessionInfo stores session information
type SessionInfo struct {
	SessionID string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	IP        string
	UserAgent string
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(encryptionKey string) *SecurityManager {
	key := sha256.Sum256([]byte(encryptionKey))
	return &SecurityManager{
		apiKeys:       make(map[string]*APIKeyInfo),
		sessions:      make(map[string]*SessionInfo),
		rateLimiters:  make(map[string]*rate.Limiter),
		blockedIPs:    make(map[string]time.Time),
		encryptionKey: key[:],
	}
}

// GenerateAPIKey generates a new API key for a user
func (sm *SecurityManager) GenerateAPIKey(userID string, permissions []string, expiresIn time.Duration) (string, error) {
	// Generate random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Create API key
	apiKey := "lv_" + hex.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	expiresAt := time.Now().Add(expiresIn)

	sm.apiKeys[keyHash] = &APIKeyInfo{
		Key:         keyHash,
		UserID:      userID,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ExpiresAt:   &expiresAt,
		RateLimit:   100, // 100 requests per minute default
	}

	return apiKey, nil
}

// ValidateAPIKey validates an API key and returns user information
func (sm *SecurityManager) ValidateAPIKey(apiKey string) (*APIKeyInfo, error) {
	// Remove prefix if present
	cleanKey := strings.TrimPrefix(apiKey, "lv_")

	// Hash the key
	hash := sha256.Sum256([]byte(cleanKey))
	keyHash := hex.EncodeToString(hash[:])

	info, exists := sm.apiKeys[keyHash]
	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}

	// Check expiration
	if info.ExpiresAt != nil && time.Now().After(*info.ExpiresAt) {
		delete(sm.apiKeys, keyHash)
		return nil, fmt.Errorf("API key expired")
	}

	// Update last used time
	info.LastUsedAt = time.Now()

	return info, nil
}

// CheckPermission checks if an API key has a specific permission
func (sm *SecurityManager) CheckPermission(apiKey, permission string) bool {
	info, err := sm.ValidateAPIKey(apiKey)
	if err != nil {
		return false
	}

	for _, p := range info.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

// CreateSession creates a new session for a user
func (sm *SecurityManager) CreateSession(userID, ip, userAgent string) (string, error) {
	sessionID := generateRandomString(32)
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hours

	sm.sessions[sessionID] = &SessionInfo{
		SessionID: sessionID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		IP:        ip,
		UserAgent: userAgent,
	}

	return sessionID, nil
}

// ValidateSession validates a session
func (sm *SecurityManager) ValidateSession(sessionID string) (*SessionInfo, error) {
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("invalid session")
	}

	if time.Now().After(session.ExpiresAt) {
		delete(sm.sessions, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// RateLimitMiddleware creates a rate limiting middleware
func (sm *SecurityManager) RateLimitMiddleware(rps float64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client identifier (IP + API key)
			clientID := r.RemoteAddr
			if auth := r.Header.Get("Authorization"); auth != "" {
				if strings.HasPrefix(auth, "Bearer ") {
					clientID += ":" + strings.TrimPrefix(auth, "Bearer ")
				}
			}

			// Check if IP is blocked
			if blockedUntil, blocked := sm.blockedIPs[r.RemoteAddr]; blocked && time.Now().Before(blockedUntil) {
				http.Error(w, "IP temporarily blocked", http.StatusTooManyRequests)
				return
			}

			// Get or create rate limiter
			limiter, exists := sm.rateLimiters[clientID]
			if !exists {
				limiter = rate.NewLimiter(rate.Limit(rps), int(rps*2))
				sm.rateLimiters[clientID] = limiter
			}

			if !limiter.Allow() {
				// Block IP for 5 minutes on rate limit violation
				sm.blockedIPs[r.RemoteAddr] = time.Now().Add(5 * time.Minute)
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SanitizeInput sanitizes user input to prevent injection attacks
func SanitizeInput(input string) string {
	// Basic sanitization - remove potentially dangerous characters
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	return input
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
