package auth

import (
	"context"
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

// auditLogCapacity bounds the in-memory ring buffer of AuditEvent entries.
// On overflow, the oldest entry is dropped (FIFO). 1024 entries is the
// agreed baseline per round-28 §11.4 sweep; durable persistence to a real
// audit database is a separate follow-up tracked outside this anchor.
const auditLogCapacity = 1024

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
	auditLog      []AuditEvent        // Bounded FIFO buffer of audit events
	auditMu       sync.Mutex          // Protects auditLog (separate from mu to avoid contention with hot auth path)
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
		auditLog:      make([]AuditEvent, 0, auditLogCapacity),
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

	// Store in memory (in production, this would be in database).
	// Round-28 §11.4 fix: also index by numeric ID so AssignRoleToClient
	// (and other ID-based lookups added subsequently) can find the client.
	// Previously the by-ID map was populated only by SSO authentication —
	// which made the public RBAC API unusable for normally-registered
	// clients and hid the RBAC bluff.
	am.mu.Lock()
	am.clients[apiKey] = client
	am.clientsByID[client.ID] = client
	am.mu.Unlock()

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

	// Round-59 JWKS-based signature verification fields. When JWKSURL
	// is non-empty, SSOManager.ValidateToken performs real asymmetric
	// signature verification (closes round-28 JWT-bypass gap). When
	// empty, ValidateToken refuses with the round-28 sentinel
	// ErrJWTSignatureVerificationNotImplemented (preserved semantics:
	// "SSO configured but JWKS not wired up").
	JWKSURL     string        `json:"jwks_url,omitempty"`     // IdP's published jwks_uri
	Audience    string        `json:"audience,omitempty"`     // Expected aud claim
	AllowedAlgs []string      `json:"allowed_algs,omitempty"` // Sanitised against forbiddenAlgs; defaults to asymmetric-only set
	CacheTTL    time.Duration `json:"cache_ttl,omitempty"`    // JWKS cache TTL; 0 -> 1h, capped at 24h
	ClockSkew   time.Duration `json:"clock_skew,omitempty"`   // exp/iat/nbf skew tolerance; 0 -> 60s
}

// ErrJWTSignatureVerificationNotImplemented is returned by SSOManager.ValidateToken
// when an SSO provider is registered WITHOUT a JWKSURL configured.
//
// History:
//   - Round-28 introduced this sentinel as the original "refuse to parse"
//     guard after discovering the silent-auth-bypass bluff: the previous
//     ValidateToken base64-decoded the JWT payload directly, fabricated an
//     SSOUserInfo, and returned success — accepting unsigned tokens as
//     authenticated. §11.4 PASS-bluff at the authentication layer.
//   - Round-59 wired real JWKS-based asymmetric signature verification via
//     verifyJWTWithJWKS (see jwks.go). The sentinel is PRESERVED — its
//     meaning is now narrowed to "SSO provider configured WITHOUT JWKS
//     endpoint", which keeps the round-28 refusal path live for any
//     deployment that registered an SSO provider but never supplied a
//     jwks_uri. This guarantees the pre-round-59 test
//     TestSSOManager_ValidateToken_RefusesUnsignedJWT keeps passing.
//
// Adjacency: CONST-035 (anti-bluff), CONST-042 (No-Secret-Leak — a
// successful bypass would have effectively leaked auth scope),
// Article XI §11.9 (anti-bluff forensic anchor).
var ErrJWTSignatureVerificationNotImplemented = fmt.Errorf("llmsverifier auth: SSO provider configured without JWKS endpoint — refusing to parse SSO token because accepting unsigned tokens is an authentication bypass (§11.4 PASS-bluff: security gap). Configure SSOConfig.JWKSURL with the IdP's published jwks_uri to enable round-59 JWKS-based signature verification")

// SSOManager handles SSO operations
type SSOManager struct {
	configs map[string]*SSOConfig
	// Round-59: per-provider JWKS cache. Populated lazily on first
	// AddProvider call when the supplied SSOConfig has a JWKSURL.
	keysets map[string]*jwksKeySet
	mu      sync.RWMutex
}

// NewSSOManager creates a new SSO manager
func NewSSOManager() *SSOManager {
	return &SSOManager{
		configs: make(map[string]*SSOConfig),
		keysets: make(map[string]*jwksKeySet),
	}
}

// AddProvider adds an SSO provider configuration
//
// Round-59: when config.JWKSURL is non-empty, eagerly builds the
// jwksKeySet cache so the first ValidateToken call doesn't have to
// hold sm.mu during the IdP roundtrip. The cache itself is empty until
// the first verification triggers a refresh — we only stand up the
// container struct here.
func (sm *SSOManager) AddProvider(config *SSOConfig) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.configs[config.Provider] = config
	if config.JWKSURL != "" {
		sm.keysets[config.Provider] = newJWKSKeySet(&JWKSConfig{
			JWKSURL:     config.JWKSURL,
			Issuer:      config.Issuer,
			Audience:    config.Audience,
			AllowedAlgs: config.AllowedAlgs,
			CacheTTL:    config.CacheTTL,
			ClockSkew:   config.ClockSkew,
		})
	}
}

// ValidateToken verifies an SSO JWT.
//
// History:
//   - Pre-round-28: base64-decoded the payload, fabricated SSOUserInfo,
//     accepted ANY well-formed 3-part dot-delimited string as
//     "authenticated". Silent auth bypass. §11.4 PASS-bluff at the
//     authentication layer.
//   - Round-28: refused the SSO flow entirely with
//     ErrJWTSignatureVerificationNotImplemented sentinel. Secure but
//     blocked all SSO use.
//   - Round-59 (current): wires real JWKS-based asymmetric signature
//     verification via verifyJWTWithJWKS (see jwks.go). Re-enables SSO
//     with full crypto:
//       1. Provider-existence check (preserved).
//       2. Token format sanity (preserved).
//       3. If SSOConfig.JWKSURL is empty -> round-28 sentinel
//          (preserves "unconfigured" semantics for any deployment that
//          registered a provider but never supplied jwks_uri).
//       4. Else -> full JWKS path: fetch JWKS (cached per kid),
//          enforce alg allowlist (no `none`, no HMAC for SSO), verify
//          signature against IdP public key, validate iss/aud/exp/iat.
//       5. On success: synthesise SSOUserInfo from verified claims.
//
// CONST-035 / CONST-042 / CONST-050(A) / Article XI §11.9 anchors apply.
func (sm *SSOManager) ValidateToken(provider, tokenString string) (*SSOUserInfo, error) {
	sm.mu.RLock()
	cfg, exists := sm.configs[provider]
	keySet := sm.keysets[provider]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("SSO provider %s not configured", provider)
	}

	// Preserve format check so callers receive specific feedback first.
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Format-sanity probe (base64 must decode). This catches obvious
	// garbage before either path (JWKS or round-28 sentinel) runs.
	if _, err := base64.RawURLEncoding.DecodeString(parts[1]); err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}

	// Round-28 preserved sentinel path: SSO provider configured WITHOUT
	// JWKS endpoint. We will NEVER fabricate a verified identity from
	// an unsigned token. Surface the original sentinel so deployments
	// upgrading from round-28 see identical refusal behaviour.
	if cfg.JWKSURL == "" || keySet == nil {
		return nil, ErrJWTSignatureVerificationNotImplemented
	}

	// Round-59 JWKS path: real asymmetric signature verification.
	// 60s context bound: the IdP JWKS endpoint must respond promptly;
	// the http.Client also enforces its own timeout (default 10s).
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	jwksCfg := &JWKSConfig{
		JWKSURL:     cfg.JWKSURL,
		Issuer:      cfg.Issuer,
		Audience:    cfg.Audience,
		AllowedAlgs: cfg.AllowedAlgs,
		CacheTTL:    cfg.CacheTTL,
		ClockSkew:   cfg.ClockSkew,
		HTTPClient:  keySet.cfg.HTTPClient,
	}
	claims, err := verifyJWTWithJWKS(ctx, jwksCfg, keySet, tokenString)
	if err != nil {
		return nil, err
	}

	// Synthesise SSOUserInfo from verified claims. We pull only fields
	// the IdP signed — we do not invent any.
	info := &SSOUserInfo{
		Provider: provider,
	}
	if v, ok := claims["iss"].(string); ok {
		info.Issuer = v
	}
	if v, ok := claims["sub"].(string); ok {
		info.Subject = v
	}
	if v, ok := claims["email"].(string); ok {
		info.Email = v
	}
	if v, ok := claims["name"].(string); ok {
		info.Name = v
	}
	if v, ok := claims["picture"].(string); ok {
		info.Picture = v
	}
	if v, ok := claims["groups"].([]interface{}); ok {
		groups := make([]string, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
		info.Groups = groups
	}
	return info, nil
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

// CreateRole creates (or replaces) a role and stores its permission set in
// the in-memory roles map. Previously this method only fmt.Printf'd — role
// data never persisted anywhere, so subsequent AssignRoleToClient calls
// could not consult it (a §11.4 PASS-bluff at the RBAC layer: the API
// advertised role management while delivering nothing). The in-memory map
// already lived on AuthManager (initialised in NewAuthManager); this
// method now actually writes to it. Durable DB-backed persistence is a
// documented follow-up — not relevant to closing the bluff.
func (am *AuthManager) CreateRole(name string, permissions []string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	// Defensive copy so the caller's slice cannot be mutated under us.
	perms := make([]string, len(permissions))
	copy(perms, permissions)
	am.roles[name] = perms
}

// GetRolePermissions returns the permission set for a role (defensive copy)
// or nil if the role has not been registered. Added in round-28 to support
// CONST-035 anti-bluff assertions and to give callers a way to inspect what
// CreateRole actually persisted.
func (am *AuthManager) GetRolePermissions(name string) []string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	perms, ok := am.roles[name]
	if !ok {
		return nil
	}
	out := make([]string, len(perms))
	copy(out, perms)
	return out
}

// AssignRoleToClient assigns a registered role's permissions to a client by
// numeric ID. Previously this method hardcoded `am.clients["dummy"]` — the
// lookup ALWAYS missed (no such key existed), so the role-assignment API was
// effectively a no-op disguised as a working feature. §11.4 PASS-bluff at
// the RBAC layer.
//
// Round-28 fix:
//   1. Look up the client via the existing clientsByID map (the correct
//      data structure for the numeric clientID argument the API accepts).
//   2. Resolve the role's permission set from the in-memory roles map
//      (populated by CreateRole and by NewAuthManager's default roles).
//   3. Merge the role's permissions into the client's permission set
//      (deduplicating) and stamp UpdatedAt.
func (am *AuthManager) AssignRoleToClient(clientID int64, roleName string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	client, ok := am.clientsByID[clientID]
	if !ok {
		return fmt.Errorf("client not found")
	}

	rolePerms, ok := am.roles[roleName]
	if !ok {
		return fmt.Errorf("unknown role: %s", roleName)
	}

	// Merge into client.Permissions, deduplicating.
	existing := make(map[string]struct{}, len(client.Permissions))
	for _, p := range client.Permissions {
		existing[p] = struct{}{}
	}
	for _, p := range rolePerms {
		if _, ok := existing[p]; !ok {
			client.Permissions = append(client.Permissions, p)
			existing[p] = struct{}{}
		}
	}

	client.UpdatedAt = time.Now()
	return nil
}

// AuditAuthEvent appends an authentication event to the in-memory audit log.
//
// §11.4 anti-bluff (round-28): the previous body was a single fmt.Printf with
// the comment "In production, this would log to audit database". No event
// was ever persisted; every audit query saw an empty store; the security
// claim that authentication is auditable was a bluff at the persistence
// layer (CONST-035, Article XI §11.9).
//
// The fix introduces an in-memory bounded FIFO ring of AuditEvent (capacity
// auditLogCapacity = 1024). This is an HONEST BASELINE — durable persistence
// to a real audit database remains a documented follow-up, but no caller
// can now make a false audit-coverage claim because every call leaves a
// retrievable record.
//
// `eventType` is interpreted as the event name; success is inferred as
// true unless the eventType contains "FAIL" (case-insensitive). Callers
// needing finer control should use the lower-level AppendAuditEvent.
func (am *AuthManager) AuditAuthEvent(eventType, clientID, details string) {
	status := "success"
	if strings.Contains(strings.ToUpper(eventType), "FAIL") {
		status = "failure"
	}
	evt := AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		ClientID:  clientID,
		Status:    status,
		Details:   map[string]interface{}{"message": details},
	}
	am.AppendAuditEvent(evt)
}

// AppendAuditEvent is the lower-level audit-trail entry point. Appends `evt`
// to the in-memory ring buffer, dropping the oldest entry on overflow.
func (am *AuthManager) AppendAuditEvent(evt AuditEvent) {
	am.auditMu.Lock()
	defer am.auditMu.Unlock()
	if len(am.auditLog) >= auditLogCapacity {
		// Drop oldest (FIFO).
		am.auditLog = append(am.auditLog[1:], evt)
		return
	}
	am.auditLog = append(am.auditLog, evt)
}

// GetAuditLog returns a defensive copy of the current in-memory audit log.
// Added in round-28 to support CONST-035 anti-bluff assertions and to give
// callers a way to inspect what AuditAuthEvent actually persisted.
func (am *AuthManager) GetAuditLog() []AuditEvent {
	am.auditMu.Lock()
	defer am.auditMu.Unlock()
	out := make([]AuditEvent, len(am.auditLog))
	copy(out, am.auditLog)
	return out
}
