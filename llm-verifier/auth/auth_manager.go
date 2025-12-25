package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

// Client represents a registered client application
type Client struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	APIKey      string     `json:"api_key,omitempty"` // Only for internal use
	APIKeyHash  string     `json:"-"`                 // Stored hashed version
	Permissions []string   `json:"permissions"`
	RateLimit   int        `json:"rate_limit"` // requests per minute
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
}

// ClientUsage tracks usage statistics for clients
type ClientUsage struct {
	ClientID         int64      `json:"client_id"`
	RequestsToday    int        `json:"requests_today"`
	RequestsThisHour int        `json:"requests_this_hour"`
	TotalRequests    int        `json:"total_requests"`
	LastRequestAt    *time.Time `json:"last_request_at"`
	DailyResetAt     time.Time  `json:"daily_reset_at"`
	HourlyResetAt    time.Time  `json:"hourly_reset_at"`
}

// AuthManager handles authentication and authorization
type AuthManager struct {
	jwtSecret   []byte
	hashParams  argon2HashParams
	clients     map[string]*Client // In-memory client store for demo
	ldapEnabled bool               // Enterprise: LDAP integration enabled
	rbacEnabled bool               // Enterprise: RBAC enabled
	ssoEnabled  bool               // Enterprise: SSO enabled
}

// argon2HashParams defines parameters for Argon2 hashing
type argon2HashParams struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	ClientID    int64    `json:"client_id"`
	ClientName  string   `json:"client_name"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(jwtSecret string) *AuthManager {
	return &AuthManager{
		jwtSecret: []byte(jwtSecret),
		hashParams: argon2HashParams{
			time:    1,
			memory:  64 * 1024,
			threads: 4,
			keyLen:  32,
		},
		clients: make(map[string]*Client),
	}
}

// RegisterClient creates a new client with generated API key
func (am *AuthManager) RegisterClient(name, description string, permissions []string, rateLimit int) (*Client, string, error) {
	// Generate secure API key
	apiKey, err := am.generateSecureAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the API key for storage
	hashedKey, err := am.hashAPIKey(apiKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash API key: %w", err)
	}

	client := &Client{
		ID:          int64(len(am.clients) + 1),
		Name:        name,
		Description: description,
		APIKeyHash:  hashedKey,
		Permissions: permissions,
		RateLimit:   rateLimit,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store in memory (in production, this would be in database)
	am.clients[apiKey] = client

	return client, apiKey, nil
}

// AuthenticateClient authenticates a client using API key
func (am *AuthManager) AuthenticateClient(apiKey string) (*Client, error) {
	client, exists := am.clients[apiKey]
	if !exists {
		return nil, fmt.Errorf("authentication failed: invalid API key")
	}

	if !client.IsActive {
		return nil, fmt.Errorf("client is not active")
	}

	// Update last used timestamp
	now := time.Now()
	client.LastUsedAt = &now

	return client, nil
}

// AuthorizeRequest checks if client has permission for the requested operation
func (am *AuthManager) AuthorizeRequest(client *Client, requiredPermission string) error {
	if !client.IsActive {
		return fmt.Errorf("client is not active")
	}

	// Check if client has the required permission
	for _, permission := range client.Permissions {
		if permission == requiredPermission || permission == "*" {
			return nil
		}
	}

	return fmt.Errorf("insufficient permissions: %s required", requiredPermission)
}

// CheckRateLimit checks if client has exceeded rate limit
func (am *AuthManager) CheckRateLimit(clientID int64, clientRateLimit int) error {
	// In-memory rate limiting (in production, this would use Redis or database)
	// For demo purposes, we'll allow all requests
	return nil
}

// GenerateJWTToken generates a JWT token for authenticated client
func (am *AuthManager) GenerateJWTToken(client *Client, ttl time.Duration) (string, error) {
	claims := JWTClaims{
		ClientID:    client.ID,
		ClientName:  client.Name,
		Permissions: client.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "llm-verifier",
			Subject:   fmt.Sprintf("client-%d", client.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(am.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	return tokenString, nil
}

// ValidateJWTToken validates a JWT token and returns client information
func (am *AuthManager) ValidateJWTToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// generateSecureAPIKey generates a cryptographically secure API key
func (am *AuthManager) generateSecureAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Format as readable API key: prefix-random-suffix
	randomPart := base64.RawURLEncoding.EncodeToString(bytes)[:32]
	return fmt.Sprintf("lv_%s_%s", randomPart[:16], randomPart[16:]), nil
}

// hashAPIKey hashes an API key using Argon2
func (am *AuthManager) hashAPIKey(apiKey string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(apiKey), salt, am.hashParams.time, am.hashParams.memory, am.hashParams.threads, am.hashParams.keyLen)

	// Format: $argon2id$v=19$m=memory,t=time,p=threads$salt$hash
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		am.hashParams.memory, am.hashParams.time, am.hashParams.threads,
		encodedSalt, encodedHash), nil
}

// verifyAPIKey verifies an API key against its hash
func (am *AuthManager) verifyAPIKey(apiKey, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	// Extract parameters
	var memory, time uint32
	var threads uint8
	var salt, expectedHash []byte
	var err error

	for i, part := range parts[3:] {
		switch i {
		case 0: // salt
			salt, err = base64.RawStdEncoding.DecodeString(part)
			if err != nil {
				return false, err
			}
		case 1: // hash
			expectedHash, err = base64.RawStdEncoding.DecodeString(part)
			if err != nil {
				return false, err
			}
		}
	}

	// Parse parameters from version string (this is simplified)
	memory = am.hashParams.memory
	time = am.hashParams.time
	threads = am.hashParams.threads

	// Hash the provided key
	computedHash := argon2.IDKey([]byte(apiKey), salt, time, memory, threads, uint32(len(expectedHash)))

	// Compare hashes
	return subtle.ConstantTimeCompare(computedHash, expectedHash) == 1, nil
}

// GetClients returns all registered clients (without sensitive data)
func (am *AuthManager) GetClients() []*Client {
	clients := make([]*Client, 0, len(am.clients))
	for _, client := range am.clients {
		// Return copy without API key hash
		clientCopy := *client
		clientCopy.APIKeyHash = ""
		clients = append(clients, &clientCopy)
	}
	return clients
}

// GetClientUsage returns usage statistics for a client
func (am *AuthManager) GetClientUsage(clientID int64) (*ClientUsage, error) {
	// In-memory usage tracking (in production, this would be in database)
	now := time.Now()
	return &ClientUsage{
		ClientID:         clientID,
		RequestsToday:    0,
		RequestsThisHour: 0,
		TotalRequests:    0,
		LastRequestAt:    nil,
		DailyResetAt:     now.AddDate(0, 0, 1),
		HourlyResetAt:    now.Add(time.Hour),
	}, nil
}

// Middleware helper functions

// ExtractAPIKey extracts API key from request headers
func ExtractAPIKeyFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header missing")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

// RequirePermission is a middleware helper for checking permissions
func (am *AuthManager) RequirePermission(requiredPermission string) func(*Client) error {
	return func(client *Client) error {
		return am.AuthorizeRequest(client, requiredPermission)
	}
}

// ValidateAndExtractClaims extracts and validates JWT token
func (am *AuthManager) ValidateAndExtractClaims(tokenString string) (*Client, error) {
	claims, err := am.ValidateJWTToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Find client by ID (in production, this would be from database)
	for _, client := range am.clients {
		if client.ID == claims.ClientID {
			return client, nil
		}
	}

	return nil, fmt.Errorf("client not found")
}

// Enterprise Authentication Methods

// EnableLDAP enables LDAP authentication
func (am *AuthManager) EnableLDAP() {
	am.ldapEnabled = true
}

// EnableRBAC enables role-based access control
func (am *AuthManager) EnableRBAC() {
	am.rbacEnabled = true
}

// EnableSSO enables single sign-on
func (am *AuthManager) EnableSSO() {
	am.ssoEnabled = true
}

// AuthenticateWithLDAP performs LDAP authentication (placeholder)
func (am *AuthManager) AuthenticateWithLDAP(username, password string) (*Client, error) {
	if !am.ldapEnabled {
		return nil, fmt.Errorf("LDAP authentication not enabled")
	}

	// Placeholder LDAP authentication logic
	// In production, this would connect to LDAP server
	if username == "ldap-user" && password == "ldap-pass" {
		return &Client{
			ID:          999,
			Name:        "LDAP User",
			Description: "Authenticated via LDAP",
			Permissions: []string{"read", "write"},
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("LDAP authentication failed")
}

// CheckRBACPermission checks if client has RBAC permission
func (am *AuthManager) CheckRBACPermission(client *Client, resource, action string) error {
	if !am.rbacEnabled {
		return nil // RBAC disabled, allow all
	}

	// Simple RBAC logic - check if client has permission for resource:action
	requiredPermission := fmt.Sprintf("%s:%s", resource, action)

	for _, permission := range client.Permissions {
		if permission == requiredPermission || permission == "*" || permission == "admin" {
			return nil
		}
	}

	return fmt.Errorf("RBAC access denied: %s requires permission %s", client.Name, requiredPermission)
}

// AuthenticateWithSSO performs SSO authentication (placeholder)
func (am *AuthManager) AuthenticateWithSSO(provider, token string) (*Client, error) {
	if !am.ssoEnabled {
		return nil, fmt.Errorf("SSO authentication not enabled")
	}

	// Placeholder SSO authentication logic
	if provider == "google" && strings.HasPrefix(token, "google-token-") {
		return &Client{
			ID:          1000,
			Name:        "SSO User",
			Description: "Authenticated via Google SSO",
			Permissions: []string{"read"},
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("SSO authentication failed")
}

// CreateRole creates a role with permissions (RBAC)
func (am *AuthManager) CreateRole(name string, permissions []string) {
	// In production, this would store roles in database
	// For demo, just log the role creation
	fmt.Printf("Created role: %s with permissions: %v\n", name, permissions)
}

// AssignRoleToClient assigns a role to a client
func (am *AuthManager) AssignRoleToClient(clientID int64, roleName string) error {
	client, exists := am.clients["dummy"] // This is simplified
	if !exists {
		return fmt.Errorf("client not found")
	}

	// Add role-based permissions
	switch roleName {
	case "admin":
		client.Permissions = append(client.Permissions, "admin", "read", "write", "delete")
	case "editor":
		client.Permissions = append(client.Permissions, "read", "write")
	case "viewer":
		client.Permissions = append(client.Permissions, "read")
	default:
		return fmt.Errorf("unknown role: %s", roleName)
	}

	client.UpdatedAt = time.Now()
	return nil
}

// AuditAuthEvent logs authentication events
func (am *AuthManager) AuditAuthEvent(eventType, clientID, details string) {
	// In production, this would log to audit database
	fmt.Printf("AUDIT [%s] Client: %s, Details: %s\n", eventType, clientID, details)
}
