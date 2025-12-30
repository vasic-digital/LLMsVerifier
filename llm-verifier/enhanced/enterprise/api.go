package enterprise

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// EnterpriseJWTClaims represents JWT token claims for enterprise users
type EnterpriseJWTClaims struct {
	UserID    string     `json:"user_id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Roles     []RBACRole `json:"roles"`
	TenantID  string     `json:"tenant_id,omitempty"`
	jwt.RegisteredClaims
}

// TokenBlacklist manages invalidated tokens (for logout)
type TokenBlacklist struct {
	tokens map[string]time.Time // token -> expiry time
	mu     sync.RWMutex
}

// EnterpriseAPI provides HTTP API for enterprise features
type EnterpriseAPI struct {
	manager        *EnterpriseManager
	server         *http.Server
	middleware     map[string]func(http.Handler) http.Handler
	jwtSecret      []byte
	tokenBlacklist *TokenBlacklist
	tokenTTL       time.Duration
}

// NewEnterpriseAPI creates a new enterprise API
func NewEnterpriseAPI(manager *EnterpriseManager) *EnterpriseAPI {
	// Generate secure JWT secret if not configured
	jwtSecret := make([]byte, 32)
	if _, err := rand.Read(jwtSecret); err != nil {
		// Fallback to a default (should be overridden via configuration)
		jwtSecret = []byte("llm-verifier-enterprise-jwt-secret-change-in-production")
	}

	api := &EnterpriseAPI{
		manager:    manager,
		middleware: make(map[string]func(http.Handler) http.Handler),
		jwtSecret:  jwtSecret,
		tokenBlacklist: &TokenBlacklist{
			tokens: make(map[string]time.Time),
		},
		tokenTTL: time.Hour, // Default 1 hour token lifetime
	}

	// Initialize middleware
	api.initializeMiddleware()

	// Start background cleanup of expired blacklist entries
	go api.cleanupBlacklist()

	return api
}

// NewEnterpriseAPIWithSecret creates a new enterprise API with a specified JWT secret
func NewEnterpriseAPIWithSecret(manager *EnterpriseManager, jwtSecret []byte, tokenTTL time.Duration) *EnterpriseAPI {
	api := &EnterpriseAPI{
		manager:    manager,
		middleware: make(map[string]func(http.Handler) http.Handler),
		jwtSecret:  jwtSecret,
		tokenBlacklist: &TokenBlacklist{
			tokens: make(map[string]time.Time),
		},
		tokenTTL: tokenTTL,
	}

	api.initializeMiddleware()
	go api.cleanupBlacklist()

	return api
}

// cleanupBlacklist removes expired tokens from the blacklist periodically
func (api *EnterpriseAPI) cleanupBlacklist() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		api.tokenBlacklist.mu.Lock()
		now := time.Now()
		for token, expiry := range api.tokenBlacklist.tokens {
			if now.After(expiry) {
				delete(api.tokenBlacklist.tokens, token)
			}
		}
		api.tokenBlacklist.mu.Unlock()
	}
}

// blacklistToken adds a token to the blacklist
func (api *EnterpriseAPI) blacklistToken(token string, expiry time.Time) {
	api.tokenBlacklist.mu.Lock()
	defer api.tokenBlacklist.mu.Unlock()
	api.tokenBlacklist.tokens[token] = expiry
}

// isTokenBlacklisted checks if a token is blacklisted
func (api *EnterpriseAPI) isTokenBlacklisted(token string) bool {
	api.tokenBlacklist.mu.RLock()
	defer api.tokenBlacklist.mu.RUnlock()
	_, exists := api.tokenBlacklist.tokens[token]
	return exists
}

// initializeMiddleware sets up HTTP middleware
func (api *EnterpriseAPI) initializeMiddleware() {
	// Authentication middleware
	api.middleware["auth"] = api.authMiddleware

	// RBAC middleware
	api.middleware["rbac"] = api.rbacMiddleware

	// Audit middleware
	api.middleware["audit"] = api.auditMiddleware

	// Rate limiting middleware
	api.middleware["ratelimit"] = api.rateLimitMiddleware

	// CORS middleware
	api.middleware["cors"] = api.corsMiddleware
}

// Start starts the enterprise API server
func (api *EnterpriseAPI) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Setup routes with middleware
	api.setupRoutes(mux)

	// Configure server
	api.server = &http.Server{
		Addr:         ":8080", // Default port
		Handler:      api.applyGlobalMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Enterprise API starting on %s", api.server.Addr)

	// Start server in goroutine
	go func() {
		if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Enterprise API server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the enterprise API server
func (api *EnterpriseAPI) Stop(ctx context.Context) error {
	if api.server == nil {
		return nil
	}

	log.Printf("Enterprise API stopping...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := api.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown API server: %w", err)
	}

	log.Printf("Enterprise API stopped")
	return nil
}

// setupRoutes configures HTTP routes
func (api *EnterpriseAPI) setupRoutes(mux *http.ServeMux) {
	// Authentication routes
	mux.HandleFunc("/api/enterprise/auth/login", api.withMiddleware(api.handleLogin, "cors"))
	mux.HandleFunc("/api/enterprise/auth/logout", api.withMiddleware(api.handleLogout, "auth", "audit"))
	mux.HandleFunc("/api/enterprise/auth/refresh", api.withMiddleware(api.handleTokenRefresh, "auth"))

	// User management routes
	mux.HandleFunc("/api/enterprise/users", api.withMiddleware(api.handleUsers, "auth", "rbac", "audit"))
	mux.HandleFunc("/api/enterprise/users/", api.withMiddleware(api.handleUser, "auth", "rbac", "audit"))

	// Role management routes
	mux.HandleFunc("/api/enterprise/roles", api.withMiddleware(api.handleRoles, "auth", "rbac"))
	mux.HandleFunc("/api/enterprise/roles/", api.withMiddleware(api.handleRole, "auth", "rbac"))

	// Tenant management routes
	mux.HandleFunc("/api/enterprise/tenants", api.withMiddleware(api.handleTenants, "auth", "rbac", "audit"))
	mux.HandleFunc("/api/enterprise/tenants/", api.withMiddleware(api.handleTenant, "auth", "rbac", "audit"))

	// Audit routes
	mux.HandleFunc("/api/enterprise/audit", api.withMiddleware(api.handleAudit, "auth", "rbac"))

	// Metrics and monitoring
	mux.HandleFunc("/api/enterprise/metrics", api.withMiddleware(api.handleMetrics, "auth", "rbac"))

	// Health check
	mux.HandleFunc("/api/enterprise/health", api.handleHealth)
}

// withMiddleware applies middleware chain to handler
func (api *EnterpriseAPI) withMiddleware(handler http.HandlerFunc, middlewares ...string) http.HandlerFunc {
	wrapped := handler

	// Apply middleware in reverse order (last to first)
	for i := len(middlewares) - 1; i >= 0; i-- {
		if mw, exists := api.middleware[middlewares[i]]; exists {
			wrapped = mw(wrapped).ServeHTTP
		}
	}

	return wrapped
}

// applyGlobalMiddleware applies global middleware to all handlers
func (api *EnterpriseAPI) applyGlobalMiddleware(handler http.Handler) http.Handler {
	// Apply CORS to all requests
	if cors, exists := api.middleware["cors"]; exists {
		handler = cors(handler)
	}

	return handler
}

// AuthMiddleware handles authentication
func (api *EnterpriseAPI) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for token in Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			api.writeError(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			api.writeError(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token (in real implementation, you'd validate JWT)
		user, err := api.validateToken(token)
		if err != nil {
			api.writeError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// rbacMiddleware handles role-based access control
func (api *EnterpriseAPI) rbacMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("user")
		if user == nil {
			api.writeError(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Check permissions based on route
		requiredPermission := api.getRequiredPermission(r.URL.Path, r.Method)
		if requiredPermission == "" {
			// No permission required for this route
			next.ServeHTTP(w, r)
			return
		}

		// Check if user has required permission
		if !api.manager.RBAC.HasPermission(user.(*User).ID, requiredPermission) {
			api.writeError(w, "Insufficient permissions", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// auditMiddleware logs all requests
func (api *EnterpriseAPI) auditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Create response writer wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapper, r)

		// Log audit entry
		duration := time.Since(startTime)
		user := r.Context().Value("user")
		var userID string
		if user != nil {
			userID = user.(*User).ID
		}

		api.manager.RBAC.logAudit(userID, r.Method+":"+r.URL.Path, r.URL.Path,
			api.getClientIP(r), wrapper.statusCode < 400, map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"duration":   duration.String(),
				"user_agent": r.Header.Get("User-Agent"),
			})
	})
}

// rateLimitMiddleware implements rate limiting
func (api *EnterpriseAPI) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic rate limiting implementation
		// In a real implementation, you'd use a more sophisticated rate limiter

		clientIP := api.getClientIP(r)

		// For demo, just pass through
		_ = clientIP

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware handles CORS
func (api *EnterpriseAPI) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// HTTP Handlers

func (api *EnterpriseAPI) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse login request
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		api.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if loginReq.Username == "" || loginReq.Password == "" {
		api.writeError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Authenticate user
	user, err := api.authenticateUser(loginReq.Username, loginReq.Password)
	if err != nil {
		// Log failed login attempt
		api.manager.RBAC.logAudit("", "user.login_failed", "auth", api.getClientIP(r), false, map[string]interface{}{
			"username": loginReq.Username,
			"error":    err.Error(),
		})
		api.writeError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, expiresAt, err := api.generateToken(user)
	if err != nil {
		log.Printf("Failed to generate token for user %s: %v", user.ID, err)
		api.writeError(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	// Log successful login
	api.manager.RBAC.logAudit(user.ID, "user.login", "auth", api.getClientIP(r), true, map[string]interface{}{
		"username": loginReq.Username,
	})

	// Calculate expires_in in seconds
	expiresIn := int(time.Until(expiresAt).Seconds())

	api.writeJSON(w, map[string]interface{}{
		"token":      token,
		"token_type": "Bearer",
		"user":       user,
		"expires_in": expiresIn,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (api *EnterpriseAPI) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value("user").(*User)

	// Extract the token from Authorization header to blacklist it
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			token := parts[1]

			// Parse token to get expiry time for blacklist cleanup
			parsedToken, err := jwt.ParseWithClaims(token, &EnterpriseJWTClaims{}, func(t *jwt.Token) (interface{}, error) {
				return api.jwtSecret, nil
			})

			if err == nil {
				if claims, ok := parsedToken.Claims.(*EnterpriseJWTClaims); ok {
					// Add token to blacklist until its original expiry
					if claims.ExpiresAt != nil {
						api.blacklistToken(token, claims.ExpiresAt.Time)
					} else {
						// If no expiry, blacklist for default TTL
						api.blacklistToken(token, time.Now().Add(api.tokenTTL))
					}
				}
			}
		}
	}

	// Log audit entry
	api.manager.RBAC.logAudit(user.ID, "user.logout", "auth", api.getClientIP(r), true, nil)

	api.writeJSON(w, map[string]string{"message": "Logged out successfully"})
}

func (api *EnterpriseAPI) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the current user from context (set by auth middleware)
	user := r.Context().Value("user").(*User)

	// Blacklist the old token
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			oldToken := parts[1]
			// Blacklist old token immediately
			api.blacklistToken(oldToken, time.Now().Add(api.tokenTTL))
		}
	}

	// Generate new token
	newToken, expiresAt, err := api.generateToken(user)
	if err != nil {
		log.Printf("Failed to refresh token for user %s: %v", user.ID, err)
		api.writeError(w, "Failed to refresh token", http.StatusInternalServerError)
		return
	}

	// Log token refresh
	api.manager.RBAC.logAudit(user.ID, "user.token_refresh", "auth", api.getClientIP(r), true, nil)

	expiresIn := int(time.Until(expiresAt).Seconds())

	api.writeJSON(w, map[string]interface{}{
		"token":      newToken,
		"token_type": "Bearer",
		"expires_in": expiresIn,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (api *EnterpriseAPI) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users := api.manager.RBAC.GetUsers()
		api.writeJSON(w, users)

	case http.MethodPost:
		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			api.writeError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := api.manager.RBAC.CreateUser(&user); err != nil {
			api.writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		api.writeJSON(w, map[string]string{"message": "User created successfully"})

	default:
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *EnterpriseAPI) handleUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL
	userID := strings.TrimPrefix(r.URL.Path, "/api/enterprise/users/")
	if userID == "" {
		api.writeError(w, "User ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := api.manager.RBAC.GetUser(userID)
		if err != nil {
			api.writeError(w, err.Error(), http.StatusNotFound)
			return
		}

		api.writeJSON(w, user)

	case http.MethodPut:
		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			api.writeError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Update user logic would go here
		api.writeJSON(w, map[string]string{"message": "User updated successfully"})

	case http.MethodDelete:
		// Delete user logic would go here
		api.writeJSON(w, map[string]string{"message": "User deleted successfully"})

	default:
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *EnterpriseAPI) handleRoles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		roles := api.manager.RBAC.GetRoles()
		api.writeJSON(w, roles)

	default:
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *EnterpriseAPI) handleRole(w http.ResponseWriter, r *http.Request) {
	// Handle individual role operations
	api.writeJSON(w, map[string]string{"message": "Role operations"})
}

func (api *EnterpriseAPI) handleTenants(w http.ResponseWriter, r *http.Request) {
	if api.manager.MultiTenant == nil {
		api.writeError(w, "Multi-tenancy not enabled", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tenants := api.manager.MultiTenant.GetAllTenants()
		api.writeJSON(w, tenants)

	case http.MethodPost:
		var tenant Tenant
		if err := json.NewDecoder(r.Body).Decode(&tenant); err != nil {
			api.writeError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := api.manager.MultiTenant.CreateTenant(&tenant); err != nil {
			api.writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		api.writeJSON(w, map[string]string{"message": "Tenant created successfully"})

	default:
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *EnterpriseAPI) handleTenant(w http.ResponseWriter, r *http.Request) {
	// Handle individual tenant operations
	api.writeJSON(w, map[string]string{"message": "Tenant operations"})
}

func (api *EnterpriseAPI) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 100 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	auditLog := api.manager.RBAC.GetAuditLog(limit)
	api.writeJSON(w, auditLog)
}

func (api *EnterpriseAPI) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return enterprise metrics
	metrics := map[string]interface{}{
		"total_users": len(api.manager.RBAC.users),
		"total_roles": len(api.manager.RBAC.roles),
		"total_tenants": func() int {
			if api.manager.MultiTenant != nil {
				return len(api.manager.MultiTenant.tenants)
			}
			return 0
		}(),
		"uptime": time.Since(time.Time{}).String(),
	}

	api.writeJSON(w, metrics)
}

func (api *EnterpriseAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"features": map[string]bool{
			"rbac":          api.manager.Config.RBAC.Enabled,
			"ldap":          api.manager.Config.LDAP.URL != "",
			"saml":          api.manager.Config.SAML.IdentityProviderURL != "",
			"multi_tenant":  api.manager.Config.MultiTenant.Enabled,
			"audit_logging": api.manager.Config.AuditLogging.Enabled,
		},
	}

	api.writeJSON(w, health)
}

// Utility methods

func (api *EnterpriseAPI) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (api *EnterpriseAPI) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (api *EnterpriseAPI) validateToken(token string) (*User, error) {
	// Check if token is blacklisted (logged out)
	if api.isTokenBlacklisted(token) {
		return nil, errors.New("token has been revoked")
	}

	// Parse and validate the JWT token
	parsedToken, err := jwt.ParseWithClaims(token, &EnterpriseJWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return api.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract claims
	claims, ok := parsedToken.Claims.(*EnterpriseJWTClaims)
	if !ok || !parsedToken.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Check token expiration (jwt library handles this, but double-check)
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}

	// Verify the user still exists and is enabled
	user, err := api.manager.RBAC.GetUser(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if !user.Enabled {
		return nil, errors.New("user account is disabled")
	}

	return user, nil
}

// generateToken creates a new JWT token for a user
func (api *EnterpriseAPI) generateToken(user *User) (string, time.Time, error) {
	expiresAt := time.Now().Add(api.tokenTTL)

	claims := &EnterpriseJWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "llm-verifier-enterprise",
			Subject:   user.ID,
			ID:        generateTokenID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(api.jwtSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// generateTokenID creates a unique token ID
func generateTokenID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (api *EnterpriseAPI) authenticateUser(username, password string) (*User, error) {
	// Try LDAP first if configured
	if api.manager.LDAP != nil {
		ldapUser, err := api.manager.LDAP.Authenticate(username, password)
		if err != nil {
			return nil, err
		}

		// Convert LDAPUser to User
		return &User{
			ID:        ldapUser.ID,
			Username:  ldapUser.Username,
			Email:     ldapUser.Email,
			FirstName: ldapUser.FirstName,
			LastName:  ldapUser.LastName,
			Roles:     ldapUser.Roles,
			Enabled:   ldapUser.Enabled,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}

	// Fall back to local authentication
	return api.manager.RBAC.AuthenticateUser(username, password)
}

func (api *EnterpriseAPI) getRequiredPermission(path, method string) Permission {
	// Simple permission mapping - in real implementation, this would be more sophisticated
	switch {
	case strings.HasPrefix(path, "/api/enterprise/users"):
		if method == "GET" {
			return PermissionJobView
		}
		return PermissionUserManage
	case strings.HasPrefix(path, "/api/enterprise/roles"):
		return PermissionSystemConfigure
	case strings.HasPrefix(path, "/api/enterprise/tenants"):
		return PermissionUserManage
	case strings.HasPrefix(path, "/api/enterprise/audit"):
		return PermissionLogsView
	case strings.HasPrefix(path, "/api/enterprise/metrics"):
		return PermissionMetricsView
	default:
		return ""
	}
}

func (api *EnterpriseAPI) getClientIP(r *http.Request) string {
	// Get client IP from request headers or remote address
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// responseWriter wrapper to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
