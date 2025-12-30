package enterprise

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== handleLogout Tests ====================

func TestHandleLogout_Success(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	testUser := &User{
		ID:       "test_user_123",
		Username: "testuser",
		Email:    "testuser@example.com",
		Roles:    []RBACRole{RBACRoleAnalyst},
		Enabled:  true,
	}
	err := manager.RBAC.CreateUser(testUser)
	require.NoError(t, err)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	// Generate valid token
	token, _, err := api.generateToken(testUser)
	require.NoError(t, err)

	// Create request with user in context
	req := httptest.NewRequest(http.MethodPost, "/api/enterprise/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := context.WithValue(req.Context(), "user", testUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	api.handleLogout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Logged out successfully", response["message"])

	// Token should now be blacklisted
	assert.True(t, api.isTokenBlacklisted(token))
}

func TestHandleLogout_WrongMethod(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)
	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/enterprise/auth/logout", nil)
			w := httptest.NewRecorder()

			api.handleLogout(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

// ==================== handleTokenRefresh Tests ====================

func TestHandleTokenRefresh_Success(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	testUser := &User{
		ID:       "test_user_123",
		Username: "testuser",
		Email:    "testuser@example.com",
		Roles:    []RBACRole{RBACRoleAnalyst},
		Enabled:  true,
	}
	err := manager.RBAC.CreateUser(testUser)
	require.NoError(t, err)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	// Generate initial token
	oldToken, _, err := api.generateToken(testUser)
	require.NoError(t, err)

	// Create request with user in context
	req := httptest.NewRequest(http.MethodPost, "/api/enterprise/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+oldToken)
	ctx := context.WithValue(req.Context(), "user", testUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	api.handleTokenRefresh(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotEmpty(t, response["token"])
	assert.Equal(t, "Bearer", response["token_type"])
	assert.NotEmpty(t, response["expires_in"])

	// Old token should be blacklisted
	assert.True(t, api.isTokenBlacklisted(oldToken))
}

func TestHandleTokenRefresh_WrongMethod(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)
	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/enterprise/auth/refresh", nil)
			w := httptest.NewRecorder()

			api.handleTokenRefresh(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

// ==================== handleUsers Tests ====================

func TestHandleUsers_GetSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	// Create some test users
	testUsers := []*User{
		{ID: "user1", Username: "user1", Email: "user1@test.com", Enabled: true},
		{ID: "user2", Username: "user2", Email: "user2@test.com", Enabled: true},
	}
	for _, u := range testUsers {
		err := manager.RBAC.CreateUser(u)
		require.NoError(t, err)
	}

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/users", nil)
	w := httptest.NewRecorder()

	api.handleUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleUsers_PostSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	body := `{"id": "new_user", "username": "newuser", "email": "new@test.com", "enabled": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/enterprise/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "User created successfully", response["message"])
}

func TestHandleUsers_PostInvalidJSON(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodPost, "/api/enterprise/users", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	api.handleUsers(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleUsers_WrongMethod(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodDelete, "/api/enterprise/users", nil)
	w := httptest.NewRecorder()

	api.handleUsers(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ==================== handleUser Tests ====================

func TestHandleUser_GetSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	testUser := &User{
		ID:       "test_user_123",
		Username: "testuser",
		Email:    "testuser@example.com",
		Enabled:  true,
	}
	err := manager.RBAC.CreateUser(testUser)
	require.NoError(t, err)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/users/test_user_123", nil)
	w := httptest.NewRecorder()

	api.handleUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleUser_GetNotFound(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/users/nonexistent", nil)
	w := httptest.NewRecorder()

	api.handleUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleUser_PutSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	body := `{"id": "user123", "username": "updated", "email": "updated@test.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/enterprise/users/user123", strings.NewReader(body))
	w := httptest.NewRecorder()

	api.handleUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleUser_DeleteSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodDelete, "/api/enterprise/users/user123", nil)
	w := httptest.NewRecorder()

	api.handleUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleUser_EmptyUserID(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/users/", nil)
	w := httptest.NewRecorder()

	api.handleUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== handleRoles Tests ====================

func TestHandleRoles_GetSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/roles", nil)
	w := httptest.NewRecorder()

	api.handleRoles(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleRoles_WrongMethod(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodPost, "/api/enterprise/roles", nil)
	w := httptest.NewRecorder()

	api.handleRoles(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ==================== handleRole Tests ====================

func TestHandleRole_Success(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/roles/admin", nil)
	w := httptest.NewRecorder()

	api.handleRole(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ==================== handleTenants Tests ====================

func TestHandleTenants_NoMultiTenant(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
		// MultiTenant is not enabled
	}
	manager := NewEnterpriseManager(config)
	// Explicitly set MultiTenant to nil to simulate disabled state
	manager.MultiTenant = nil

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/tenants", nil)
	w := httptest.NewRecorder()

	api.handleTenants(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleTenants_GetSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC:        RBACConfig{Enabled: true},
		MultiTenant: MultiTenantConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/tenants", nil)
	w := httptest.NewRecorder()

	api.handleTenants(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleTenants_PostSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC:        RBACConfig{Enabled: true},
		MultiTenant: MultiTenantConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	body := `{"id": "tenant1", "name": "Test Tenant"}`
	req := httptest.NewRequest(http.MethodPost, "/api/enterprise/tenants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTenants(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleTenants_PostInvalidJSON(t *testing.T) {
	config := EnterpriseConfig{
		RBAC:        RBACConfig{Enabled: true},
		MultiTenant: MultiTenantConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodPost, "/api/enterprise/tenants", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	api.handleTenants(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== handleTenant Tests ====================

func TestHandleTenant_Success(t *testing.T) {
	config := EnterpriseConfig{
		RBAC:        RBACConfig{Enabled: true},
		MultiTenant: MultiTenantConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/tenants/tenant1", nil)
	w := httptest.NewRecorder()

	api.handleTenant(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ==================== handleAudit Tests ====================

func TestHandleAudit_GetSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/audit", nil)
	w := httptest.NewRecorder()

	api.handleAudit(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAudit_WrongMethod(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodPost, "/api/enterprise/audit", nil)
	w := httptest.NewRecorder()

	api.handleAudit(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ==================== handleMetrics Tests ====================

func TestHandleMetrics_GetSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/metrics", nil)
	w := httptest.NewRecorder()

	api.handleMetrics(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleMetrics_WrongMethod(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodPost, "/api/enterprise/metrics", nil)
	w := httptest.NewRecorder()

	api.handleMetrics(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ==================== handleHealth Tests ====================

func TestHandleHealth_GetSuccess(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/health", nil)
	w := httptest.NewRecorder()

	api.handleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

// ==================== writeJSON Tests ====================

func TestWriteJSON_Success(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	api.writeJSON(w, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "value", response["key"])
}

// ==================== getClientIP Tests ====================

func TestGetClientIP_FromXForwardedFor(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1")

	ip := api.getClientIP(req)
	assert.Equal(t, "192.168.1.100", ip)
}

func TestGetClientIP_FromXRealIP(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.50")

	ip := api.getClientIP(req)
	assert.Equal(t, "192.168.1.50", ip)
}

func TestGetClientIP_FromRemoteAddr(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// RemoteAddr is set by httptest.NewRequest

	ip := api.getClientIP(req)
	assert.NotEmpty(t, ip)
}

// ==================== corsMiddleware Tests ====================

func TestCorsMiddleware_PreflightRequest(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := api.corsMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:4200")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// OPTIONS should return without calling the handler
	assert.False(t, handlerCalled)
	// CORS preflight returns 204 No Content
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCorsMiddleware_RegularRequest(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := api.corsMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:4200")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// ==================== rateLimitMiddleware Tests ====================

func TestRateLimitMiddleware_PassThrough(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := api.rateLimitMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ==================== auditMiddleware Tests ====================

func TestAuditMiddleware_PassThrough(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := api.auditMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.True(t, handlerCalled)
}

// ==================== rbacMiddleware Tests ====================

func TestRbacMiddleware_NoUser(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{Enabled: true},
	}
	manager := NewEnterpriseManager(config)

	jwtSecret := []byte("test-jwt-secret")
	api := NewEnterpriseAPIWithSecret(manager, jwtSecret, time.Hour)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := api.rbacMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/enterprise/users", nil)
	// No user in context
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
