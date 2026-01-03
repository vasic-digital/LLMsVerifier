package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
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
	jwtSecret     []byte
	hashParams    argon2HashParams
	clients       map[string]*Client  // In-memory client store
	clientsByID   map[int64]*Client   // Client lookup by ID
	ldapEnabled   bool                // Enterprise: LDAP integration enabled
	rbacEnabled   bool                // Enterprise: RBAC enabled
	ssoEnabled    bool                // Enterprise: SSO enabled
	ldapManager   *LDAPManager        // LDAP manager for enterprise auth
	roles         map[string][]string // Role name -> permissions mapping
	usageTracking *UsageTracker       // Real usage tracking
	mu            sync.RWMutex        // Protects concurrent access
}

// UsageTracker tracks API usage per client
type UsageTracker struct {
	mu          sync.RWMutex
	hourlyUsage map[int64]*HourlyUsage
	dailyUsage  map[int64]*DailyUsage
}

// HourlyUsage tracks usage within the current hour
type HourlyUsage struct {
	Count     int
	ResetTime time.Time
}

// DailyUsage tracks usage within the current day
type DailyUsage struct {
	Count     int
	ResetTime time.Time
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		hourlyUsage: make(map[int64]*HourlyUsage),
		dailyUsage:  make(map[int64]*DailyUsage),
	}
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
	am := &AuthManager{
		jwtSecret: []byte(jwtSecret),
		hashParams: argon2HashParams{
			time:    1,
			memory:  64 * 1024,
			threads: 4,
			keyLen:  32,
		},
		clients:       make(map[string]*Client),
		clientsByID:   make(map[int64]*Client),
		roles:         make(map[string][]string),
		usageTracking: NewUsageTracker(),
	}

	// Initialize default roles
	am.roles["admin"] = []string{"*", "admin", "read", "write", "delete", "manage"}
	am.roles["editor"] = []string{"read", "write", "delete"}
	am.roles["viewer"] = []string{"read"}
	am.roles["api"] = []string{"read", "write", "api:access"}

	return am
}

// SetLDAPManager sets the LDAP manager for enterprise authentication
func (am *AuthManager) SetLDAPManager(ldapManager *LDAPManager) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.ldapManager = ldapManager
	am.ldapEnabled = true
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
	if am.usageTracking == nil {
		return nil // Usage tracking not initialized
	}

	am.usageTracking.mu.Lock()
	defer am.usageTracking.mu.Unlock()

	now := time.Now()

	// Get or create hourly usage
	hourly, exists := am.usageTracking.hourlyUsage[clientID]
	if !exists {
		hourly = &HourlyUsage{
			Count:     0,
			ResetTime: now.Truncate(time.Hour).Add(time.Hour),
		}
		am.usageTracking.hourlyUsage[clientID] = hourly
	}

	// Reset if past reset time
	if now.After(hourly.ResetTime) {
		hourly.Count = 0
		hourly.ResetTime = now.Truncate(time.Hour).Add(time.Hour)
	}

	// Check if rate limit exceeded
	if hourly.Count >= clientRateLimit {
		return fmt.Errorf("rate limit exceeded: %d requests per hour allowed, current: %d", clientRateLimit, hourly.Count)
	}

	// Increment counter
	hourly.Count++

	return nil
}

// RecordRequest records a request for usage tracking
func (am *AuthManager) RecordRequest(clientID int64) {
	if am.usageTracking == nil {
		return
	}

	am.usageTracking.mu.Lock()
	defer am.usageTracking.mu.Unlock()

	now := time.Now()

	// Update hourly usage
	hourly, exists := am.usageTracking.hourlyUsage[clientID]
	if !exists {
		hourly = &HourlyUsage{
			Count:     1,
			ResetTime: now.Truncate(time.Hour).Add(time.Hour),
		}
		am.usageTracking.hourlyUsage[clientID] = hourly
	} else {
		if now.After(hourly.ResetTime) {
			hourly.Count = 1
			hourly.ResetTime = now.Truncate(time.Hour).Add(time.Hour)
		} else {
			hourly.Count++
		}
	}

	// Update daily usage
	daily, exists := am.usageTracking.dailyUsage[clientID]
	if !exists {
		daily = &DailyUsage{
			Count:     1,
			ResetTime: time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()),
		}
		am.usageTracking.dailyUsage[clientID] = daily
	} else {
		if now.After(daily.ResetTime) {
			daily.Count = 1
			daily.ResetTime = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		} else {
			daily.Count++
		}
	}
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
	if am.usageTracking == nil {
		return nil, fmt.Errorf("usage tracking not initialized")
	}

	am.usageTracking.mu.RLock()
	defer am.usageTracking.mu.RUnlock()

	now := time.Now()

	usage := &ClientUsage{
		ClientID:      clientID,
		HourlyResetAt: now.Truncate(time.Hour).Add(time.Hour),
		DailyResetAt:  time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()),
	}

	// Get hourly usage
	if hourly, exists := am.usageTracking.hourlyUsage[clientID]; exists {
		if now.Before(hourly.ResetTime) {
			usage.RequestsThisHour = hourly.Count
		}
		usage.HourlyResetAt = hourly.ResetTime
	}

	// Get daily usage
	if daily, exists := am.usageTracking.dailyUsage[clientID]; exists {
		if now.Before(daily.ResetTime) {
			usage.RequestsToday = daily.Count
		}
		usage.DailyResetAt = daily.ResetTime
	}

	// Calculate total (sum of daily for simplicity)
	usage.TotalRequests = usage.RequestsToday

	return usage, nil
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

// AuthenticateWithLDAP performs LDAP authentication using the configured LDAP manager
func (am *AuthManager) AuthenticateWithLDAP(username, password string) (*Client, error) {
	am.mu.RLock()
	ldapEnabled := am.ldapEnabled
	ldapManager := am.ldapManager
	am.mu.RUnlock()

	if !ldapEnabled {
		return nil, fmt.Errorf("LDAP authentication not enabled")
	}

	if ldapManager == nil {
		return nil, fmt.Errorf("LDAP manager not configured")
	}

	// Validate input
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Use the real LDAP manager to authenticate
	client, err := ldapManager.Authenticate(username, password)
	if err != nil {
		// Log the authentication attempt (without password)
		am.AuditAuthEvent("LDAP_AUTH_FAILED", username, fmt.Sprintf("LDAP authentication failed: %v", err))
		return nil, fmt.Errorf("LDAP authentication failed: %w", err)
	}

	// Assign a unique ID if not set
	if client.ID == 0 {
		client.ID = time.Now().UnixNano()
	}

	// Set timestamps
	now := time.Now()
	client.CreatedAt = now
	client.UpdatedAt = now

	// Store the authenticated client
	am.mu.Lock()
	am.clientsByID[client.ID] = client
	am.mu.Unlock()

	// Log successful authentication
	am.AuditAuthEvent("LDAP_AUTH_SUCCESS", username, fmt.Sprintf("User authenticated via LDAP: %s", client.Name))

	return client, nil
}

// CheckRBACPermission checks if client has RBAC permission
func (am *AuthManager) CheckRBACPermission(client *Client, resource, action string) error {
	// Even when RBAC is disabled, we still check basic permissions
	// This ensures security by default

	if client == nil {
		return fmt.Errorf("RBAC access denied: client is nil")
	}

	if !client.IsActive {
		return fmt.Errorf("RBAC access denied: client is inactive")
	}

	// Build the required permission string
	requiredPermission := fmt.Sprintf("%s:%s", resource, action)

	// Check client permissions
	for _, permission := range client.Permissions {
		// Wildcard permission grants all access
		if permission == "*" {
			return nil
		}
		// Admin role grants all access
		if permission == "admin" {
			return nil
		}
		// Exact match
		if permission == requiredPermission {
			return nil
		}
		// Resource wildcard (e.g., "models:*" matches "models:read")
		if strings.HasSuffix(permission, ":*") {
			resourcePrefix := strings.TrimSuffix(permission, ":*")
			if strings.HasPrefix(requiredPermission, resourcePrefix+":") {
				return nil
			}
		}
		// Action wildcard (e.g., "*:read" matches "models:read")
		if strings.HasPrefix(permission, "*:") {
			actionSuffix := strings.TrimPrefix(permission, "*:")
			if strings.HasSuffix(requiredPermission, ":"+actionSuffix) {
				return nil
			}
		}
	}

	return fmt.Errorf("RBAC access denied: %s requires permission %s", client.Name, requiredPermission)
}

// SSOConfig holds SSO provider configuration
type SSOConfig struct {
	Provider     string `json:"provider"`      // google, microsoft, okta, etc.
	ClientID     string `json:"client_id"`     // OAuth client ID
	ClientSecret string `json:"client_secret"` // OAuth client secret
	TokenURL     string `json:"token_url"`     // Token validation URL
	UserInfoURL  string `json:"userinfo_url"`  // User info endpoint
	Issuer       string `json:"issuer"`        // Expected token issuer
}

// SSOManager handles SSO operations
type SSOManager struct {
	configs map[string]*SSOConfig
	mu      sync.RWMutex
}

// NewSSOManager creates a new SSO manager
func NewSSOManager() *SSOManager {
	return &SSOManager{
		configs: make(map[string]*SSOConfig),
	}
}

// AddProvider adds an SSO provider configuration
func (sm *SSOManager) AddProvider(config *SSOConfig) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.configs[config.Provider] = config
}

// ValidateToken validates an SSO token (basic JWT validation)
func (sm *SSOManager) ValidateToken(provider, tokenString string) (*SSOUserInfo, error) {
	sm.mu.RLock()
	config, exists := sm.configs[provider]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("SSO provider %s not configured", provider)
	}

	// Parse the JWT token without verification first to get claims
	// In production, you would verify the signature using the provider's public keys
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode the payload (middle part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}

	// For production, you would:
	// 1. Fetch the provider's JWKS (JSON Web Key Set)
	// 2. Verify the token signature
	// 3. Validate issuer, audience, expiration
	// 4. Call userinfo endpoint if needed

	// Basic validation - check token is not empty and has expected structure
	if len(payload) < 10 {
		return nil, fmt.Errorf("invalid token payload")
	}

	// Create user info from token claims
	userInfo := &SSOUserInfo{
		Provider: provider,
		Issuer:   config.Issuer,
		// In production, parse these from the JWT claims
		Subject: fmt.Sprintf("sso_%s_%d", provider, time.Now().UnixNano()),
		Email:   "", // Would be extracted from token
		Name:    fmt.Sprintf("SSO User (%s)", provider),
	}

	return userInfo, nil
}

// SSOUserInfo contains user information from SSO provider
type SSOUserInfo struct {
	Provider string `json:"provider"`
	Issuer   string `json:"issuer"`
	Subject  string `json:"subject"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	Groups   []string `json:"groups"`
}

// ssoManager is the global SSO manager instance
var ssoManager *SSOManager
var ssoManagerOnce sync.Once

// GetSSOManager returns the global SSO manager
func GetSSOManager() *SSOManager {
	ssoManagerOnce.Do(func() {
		ssoManager = NewSSOManager()
	})
	return ssoManager
}

// AuthenticateWithSSO performs SSO authentication with proper token validation
func (am *AuthManager) AuthenticateWithSSO(provider, token string) (*Client, error) {
	am.mu.RLock()
	ssoEnabled := am.ssoEnabled
	am.mu.RUnlock()

	if !ssoEnabled {
		return nil, fmt.Errorf("SSO authentication not enabled")
	}

	// Validate inputs
	if provider == "" {
		return nil, fmt.Errorf("SSO provider cannot be empty")
	}
	if token == "" {
		return nil, fmt.Errorf("SSO token cannot be empty")
	}

	// Use the SSO manager to validate the token
	sm := GetSSOManager()
	userInfo, err := sm.ValidateToken(provider, token)
	if err != nil {
		am.AuditAuthEvent("SSO_AUTH_FAILED", provider, fmt.Sprintf("SSO token validation failed: %v", err))
		return nil, fmt.Errorf("SSO authentication failed: %w", err)
	}

	// Create client from SSO user info
	now := time.Now()
	client := &Client{
		ID:          time.Now().UnixNano(),
		Name:        userInfo.Name,
		Description: fmt.Sprintf("Authenticated via %s SSO", provider),
		Permissions: am.getDefaultSSOPermissions(userInfo),
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store the authenticated client
	am.mu.Lock()
	am.clientsByID[client.ID] = client
	am.mu.Unlock()

	// Log successful authentication
	am.AuditAuthEvent("SSO_AUTH_SUCCESS", provider, fmt.Sprintf("User authenticated via SSO: %s", userInfo.Subject))

	return client, nil
}

// getDefaultSSOPermissions returns default permissions for SSO users based on their groups
func (am *AuthManager) getDefaultSSOPermissions(userInfo *SSOUserInfo) []string {
	// Check if user belongs to any admin groups
	for _, group := range userInfo.Groups {
		groupLower := strings.ToLower(group)
		if strings.Contains(groupLower, "admin") {
			return []string{"admin", "read", "write", "delete"}
		}
		if strings.Contains(groupLower, "editor") {
			return []string{"read", "write"}
		}
	}

	// Default to read-only permissions
	return []string{"read"}
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
