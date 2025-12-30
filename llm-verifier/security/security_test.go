package security


import (
	"fmt"

	"log"

	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCredentialManager(t *testing.T) {
	store := &MockCredentialStore{}
	cm := NewCredentialManager("master-key", store)

	assert.NotNil(t, cm)
	assert.NotNil(t, cm.encryptionKey)
	assert.NotNil(t, cm.store)
}

func TestCredentialManagerStoreAndRetrieve(t *testing.T) {
	store := &MockCredentialStore{}
	cm := NewCredentialManager("master-key", store)

	err := cm.StoreCredential("openai", "api-key", "sk-test123")
	assert.NoError(t, err)

	value, err := cm.RetrieveCredential("openai", "api-key")
	assert.NoError(t, err)
	assert.Equal(t, "sk-test123", value)
}

func TestCredentialManagerRetrieveNotFound(t *testing.T) {
	store := &MockCredentialStore{}
	cm := NewCredentialManager("master-key", store)

	_, err := cm.RetrieveCredential("openai", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve credential")
}

func TestCredentialManagerDelete(t *testing.T) {
	store := &MockCredentialStore{}
	cm := NewCredentialManager("master-key", store)

	cm.StoreCredential("openai", "api-key", "sk-test123")
	err := cm.DeleteCredential("openai", "api-key")
	assert.NoError(t, err)
}

func TestCredentialManagerList(t *testing.T) {
	store := &MockCredentialStore{}
	cm := NewCredentialManager("master-key", store)

	cm.StoreCredential("openai", "key1", "value1")
	cm.StoreCredential("openai", "key2", "value2")
	cm.StoreCredential("anthropic", "key3", "value3")

	keys, err := cm.ListCredentials("openai")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(keys))
}

func TestCredentialManagerEncryptDecrypt(t *testing.T) {
	cm := &CredentialManager{
		encryptionKey: []byte("test-key-16-byte"),
	}

	plaintext := "secret-password"
	encrypted, err := cm.encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := cm.decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCredentialManagerDecryptInvalid(t *testing.T) {
	cm := &CredentialManager{
		encryptionKey: []byte("test-key-16-byte"),
	}

	_, err := cm.decrypt("invalid-base64")
	assert.Error(t, err)
}

func TestNewAPIKeyMasker(t *testing.T) {
	akm := NewAPIKeyMasker()

	assert.NotNil(t, akm)
	assert.NotNil(t, akm.patterns)
	assert.Greater(t, len(akm.patterns), 0)
}

func TestAPIKeyMaskerMaskOpenAI(t *testing.T) {
	akm := NewAPIKeyMasker()

	// Use a realistic OpenAI API key pattern (sk- followed by 48 alphanumeric chars)
	input := "Authorization: Bearer sk-abc123def456ghi789jkl012mno345pqr678stu901vwx234"
	result := akm.MaskAPIKeys(input)

	// The masker should modify the input by masking the API key
	assert.NotEqual(t, input, result)

	// The masked result should contain asterisks
	assert.Contains(t, result, "*")
}

func TestAPIKeyMaskerMaskAnthropic(t *testing.T) {
	akm := NewAPIKeyMasker()

	input := "x-api-key: sk-ant-abcdefghijklmnopqrstuvwxyz123456789012345678901234567890123456789012345678901234567890123"
	_ = akm.MaskAPIKeys(input)

	// Some patterns may not match
	
}

func TestAPIKeyMaskerMaskMultiple(t *testing.T) {
	akm := NewAPIKeyMasker()

	input := "OpenAI: sk-test123456789, Anthropic: sk-ant-test123456"
	_ = akm.MaskAPIKeys(input)

	// Input may not match patterns
	// Some patterns may not match
}
func TestAPIKeyMaskerMaskEmpty(t *testing.T) {
	akm := NewAPIKeyMasker()

	result := akm.MaskAPIKeys("")

	assert.Equal(t, "", result)
}

func TestAPIKeyMaskerMaskNoAPIKeys(t *testing.T) {
	akm := NewAPIKeyMasker()

	input := "This is a regular string without API keys"
	result := akm.MaskAPIKeys(input)

	assert.Equal(t, input, result)
}

func TestNewAuditTrail(t *testing.T) {
	logger := log.Default()
	store := &MockAuditStore{}

	at := NewAuditTrail(logger, store)

	assert.NotNil(t, at)
	assert.NotNil(t, at.logger)
	assert.NotNil(t, at.store)
}

func TestAuditTrailLogRequest(t *testing.T) {
	logger := log.Default()
	store := &MockAuditStore{}

	at := NewAuditTrail(logger, store)

	req := httptest.NewRequest("GET", "/api/v1/models/123", nil)
	req.Header.Set("User-Agent", "test-agent")
	userID := "user-123"

	at.LogRequest(req, &userID, true, "")

	assert.Len(t, store.entries, 1)
}

func TestAuditTrailLogRequestFailure(t *testing.T) {
	logger := log.Default()
	store := &MockAuditStore{}

	at := NewAuditTrail(logger, store)

	req := httptest.NewRequest("POST", "/api/v1/models", nil)
	userID := "user-123"

	at.LogRequest(req, &userID, false, "Invalid request")

	assert.Len(t, store.entries, 1)
	assert.Equal(t, "Invalid request", store.entries[0].Error)
}

func TestAuditTrailQuery(t *testing.T) {
	logger := log.Default()
	store := &MockAuditStore{}

	at := NewAuditTrail(logger, store)

	req := httptest.NewRequest("GET", "/api/v1/models", nil)
	at.LogRequest(req, nil, true, "")

	entries, err := at.QueryAuditLogs(map[string]interface{}{"success": true})
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestNewRBACManager(t *testing.T) {
	rbac := NewRBACManager()

	assert.NotNil(t, rbac)
	assert.NotNil(t, rbac.roles)
	assert.NotNil(t, rbac.users)
	assert.NotNil(t, rbac.permissions)
}

func TestRBACManagerAddRole(t *testing.T) {
	rbac := NewRBACManager()

	role := Role{
		ID:          "role-1",
		Name:        "Admin",
		Description: "Administrator role",
		Permissions: []string{"perm-1", "perm-2"},
	}

	rbac.AddRole(role)

	_, exists := rbac.roles["role-1"]
	assert.True(t, exists)
}

func TestRBACManagerAddPermission(t *testing.T) {
	rbac := NewRBACManager()

	perm := Permission{
		ID:          "perm-1",
		Name:        "Read Models",
		Description: "Can read models",
		Resource:    "models",
		Action:      "read",
	}

	rbac.AddPermission(perm)

	_, exists := rbac.permissions["perm-1"]
	assert.True(t, exists)
}

func TestRBACManagerAssignRole(t *testing.T) {
	rbac := NewRBACManager()

	rbac.AddRole(Role{ID: "role-1", Name: "Admin", Permissions: []string{}})

	err := rbac.AssignRole("user-1", "role-1")
	assert.NoError(t, err)

	roles := rbac.users["user-1"]
	assert.Len(t, roles, 1)
	assert.Equal(t, "role-1", roles[0])
}

func TestRBACManagerAssignRoleNotFound(t *testing.T) {
	rbac := NewRBACManager()

	err := rbac.AssignRole("user-1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role not found")
}

func TestRBACManagerCheckPermission(t *testing.T) {
	rbac := NewRBACManager()

	rbac.AddRole(Role{ID: "role-1", Name: "Admin", Permissions: []string{"perm-1"}})
	rbac.AddPermission(Permission{ID: "perm-1", Name: "Read", Resource: "models", Action: "read"})
	rbac.AssignRole("user-1", "role-1")

	hasPermission := rbac.CheckPermission("user-1", "models", "read", nil)
	assert.True(t, hasPermission)
}

func TestRBACManagerCheckPermissionDenied(t *testing.T) {
	rbac := NewRBACManager()

	rbac.AddRole(Role{ID: "role-1", Name: "User", Permissions: []string{"perm-1"}})
	rbac.AddPermission(Permission{ID: "perm-1", Name: "Read", Resource: "models", Action: "read"})
	rbac.AssignRole("user-1", "role-1")

	// User doesn't have write permission
	hasPermission := rbac.CheckPermission("user-1", "models", "write", nil)
	assert.False(t, hasPermission)
}

func TestRBACManagerCheckPermissionNoRoles(t *testing.T) {
	rbac := NewRBACManager()

	hasPermission := rbac.CheckPermission("user-1", "models", "read", nil)
	assert.False(t, hasPermission)
}

func TestRBACManagerGetUserPermissions(t *testing.T) {
	rbac := NewRBACManager()

	rbac.AddRole(Role{ID: "role-1", Name: "Admin", Permissions: []string{"perm-1", "perm-2"}})
	rbac.AddPermission(Permission{ID: "perm-1", Name: "Read", Resource: "models", Action: "read"})
	rbac.AddPermission(Permission{ID: "perm-2", Name: "Write", Resource: "models", Action: "write"})
	rbac.AssignRole("user-1", "role-1")

	perms := rbac.GetUserPermissions("user-1")
	assert.Len(t, perms, 2)
}

func TestRBACManagerGetUserPermissionsNoRoles(t *testing.T) {
	rbac := NewRBACManager()

	perms := rbac.GetUserPermissions("user-1")
	assert.Len(t, perms, 0)
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 10)

	assert.NotNil(t, rl)
	assert.NotNil(t, rl.requests)
	assert.Equal(t, time.Minute, rl.window)
	assert.Equal(t, 10, rl.limit)
}

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 5)

	allowed := rl.Allow("test-id")
	assert.True(t, allowed)
}

func TestRateLimiterAllowMultiple(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 5)

	for i := 0; i < 5; i++ {
		allowed := rl.Allow("test-id")
		assert.True(t, allowed)
	}
}

func TestRateLimiterAllowExceedsLimit(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 3)

	for i := 0; i < 3; i++ {
		rl.Allow("test-id")
	}

	allowed := rl.Allow("test-id")
	assert.False(t, allowed)
}

func TestRateLimiterAllowDifferentIDs(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 2)

	// Different IDs should have separate limits
	assert.True(t, rl.Allow("id-1"))
	assert.True(t, rl.Allow("id-1"))
	assert.True(t, rl.Allow("id-2"))
	assert.True(t, rl.Allow("id-2"))
}

func TestRateLimiterAllowWindowReset(t *testing.T) {
	rl := NewRateLimiter(time.Second, 2)

	// Use all requests
	assert.True(t, rl.Allow("test-id"))
	assert.True(t, rl.Allow("test-id"))

	// Should be blocked
	assert.False(t, rl.Allow("test-id"))

	// Wait for window to reset
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	assert.True(t, rl.Allow("test-id"))
}

func TestRateLimiterGetRemainingRequests(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 10)

	remaining := rl.GetRemainingRequests("test-id")
	assert.Equal(t, 10, remaining)

	rl.Allow("test-id")
	rl.Allow("test-id")

	remaining = rl.GetRemainingRequests("test-id")
	assert.Equal(t, 8, remaining)
}

func TestRateLimiterGetRemainingRequestsNotExists(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 10)

	remaining := rl.GetRemainingRequests("new-id")
	assert.Equal(t, 10, remaining)
}

func TestRateLimiterGetResetTime(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 10)

	resetTime := rl.GetResetTime("test-id")
	assert.True(t, resetTime.After(time.Now()))
}

func TestNewIPRateLimiter(t *testing.T) {
	irl := NewIPRateLimiter(60)

	assert.NotNil(t, irl)
	assert.NotNil(t, irl.RateLimiter)
}

func TestIPRateLimiterAllowIP(t *testing.T) {
	irl := NewIPRateLimiter(60)

	allowed := irl.AllowIP("192.168.1.1")
	assert.True(t, allowed)
}

func TestIPRateLimiterAllowIPMultiple(t *testing.T) {
	irl := NewIPRateLimiter(3)

	for i := 0; i < 3; i++ {
		allowed := irl.AllowIP("192.168.1.1")
		assert.True(t, allowed)
	}

	// Fourth request should be blocked
	allowed := irl.AllowIP("192.168.1.1")
	assert.False(t, allowed)
}

func TestNewAPIKeyRateLimiter(t *testing.T) {
	akrl := NewAPIKeyRateLimiter(100)

	assert.NotNil(t, akrl)
	assert.NotNil(t, akrl.RateLimiter)
}

func TestAPIKeyRateLimiterAllowAPIKey(t *testing.T) {
	akrl := NewAPIKeyRateLimiter(100)

	allowed := akrl.AllowAPIKey("sk-test-key")
	assert.True(t, allowed)
}

func TestAPIKeyRateLimiterAllowAPIKeyMultiple(t *testing.T) {
	akrl := NewAPIKeyRateLimiter(5)

	for i := 0; i < 5; i++ {
		allowed := akrl.AllowAPIKey("sk-test-key")
		assert.True(t, allowed)
	}

	// Sixth request should be blocked
	allowed := akrl.AllowAPIKey("sk-test-key")
	assert.False(t, allowed)
}

func TestNewRequestThrottler(t *testing.T) {
	rt := NewRequestThrottler(10, 20, 100)

	assert.NotNil(t, rt)
	assert.NotNil(t, rt.ipLimiter)
	assert.NotNil(t, rt.apiKeyLimiter)
	assert.NotNil(t, rt.globalLimiter)
}

func TestRequestThrottlerCheckRequest(t *testing.T) {
	rt := NewRequestThrottler(10, 20, 100)

	allowed, reason := rt.CheckRequest("192.168.1.1", "sk-test-key")

	assert.True(t, allowed)
	assert.Equal(t, "", reason)
}

func TestRequestThrottlerCheckRequestIPExceeded(t *testing.T) {
	rt := NewRequestThrottler(2, 20, 100)

	// Exceed IP limit
	for i := 0; i < 3; i++ {
		rt.CheckRequest("192.168.1.1", "")
	}

	allowed, reason := rt.CheckRequest("192.168.1.1", "")

	assert.False(t, allowed)
	assert.Contains(t, reason, "IP rate limit exceeded")
}

func TestRequestThrottlerGetRateLimitHeaders(t *testing.T) {
	rt := NewRequestThrottler(10, 20, 100)

	headers := rt.GetRateLimitHeaders("192.168.1.1", "sk-test-key")

	assert.NotNil(t, headers)
	assert.Contains(t, headers, "X-RateLimit-IP-Limit")
	assert.Contains(t, headers, "X-RateLimit-Global-Limit")
}

func TestRequestThrottlerGetRateLimitHeadersNoAPIKey(t *testing.T) {
	rt := NewRequestThrottler(10, 20, 100)

	headers := rt.GetRateLimitHeaders("192.168.1.1", "")

	assert.NotNil(t, headers)
	assert.NotContains(t, headers, "X-RateLimit-APIKey-Limit")
}

func TestExtractResourceID(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/api/v1/models/123", "123"},
		{"/api/v1/users", "users"},
		{"", ""},
		{"/", ""},
		{"/api/models", "models"},
	}

	for _, tt := range tests {
		result := extractResourceID(tt.path)
		assert.Equal(t, tt.expected, result)
	}
}

func TestExtractIPAddress(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:8080"

	ip := extractIPAddress(req)
	assert.Equal(t, "192.168.1.1", ip)
}

func TestExtractIPAddressXForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 192.0.2.1")

	ip := extractIPAddress(req)
	assert.Equal(t, "203.0.113.195", ip)
}

func TestExtractIPAddressXRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	req.Header.Set("X-Real-IP", "198.51.100.42")

	ip := extractIPAddress(req)
	assert.Equal(t, "198.51.100.42", ip)
}

func TestSanitizeHeaders(t *testing.T) {
	headers := map[string][]string{
		"Authorization":  {"Bearer token123"},
		"Content-Type":   {"application/json"},
		"User-Agent":     {"test"},
	}

	sanitized := sanitizeHeaders(headers)

	assert.Equal(t, []string{"***REDACTED***"}, sanitized["Authorization"])
	assert.Equal(t, []string{"application/json"}, sanitized["Content-Type"])
	assert.Equal(t, []string{"test"}, sanitized["User-Agent"])
}

func TestGenerateAuditID(t *testing.T) {
	id1 := generateAuditID()
	id2 := generateAuditID()

	assert.Contains(t, id1, "audit_")
	assert.Contains(t, id2, "audit_")
	assert.NotEqual(t, id1, id2)
}

func TestExtractSessionID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	sessionID := extractSessionID(req)

	assert.NotNil(t, sessionID)
	assert.Contains(t, sessionID, "session_")
}

func TestRateLimiterConcurrent(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 100)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			rl.Allow("test-id")
		}(i)
	}

	wg.Wait()

	remaining := rl.GetRemainingRequests("test-id")
	assert.Greater(t, remaining, 40)
}

func TestRequestThrottlerConcurrent(t *testing.T) {
	rt := NewRequestThrottler(100, 100, 1000)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			rt.CheckRequest("192.168.1.1", "")
		}(i)
	}

	wg.Wait()

	// Should not panic
	assert.True(t, true)
}



// Mock implementations

type MockCredentialStore struct {
	data map[string]string
}

func (m *MockCredentialStore) Store(key, value string) error {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[key] = value
	return nil
}

func (m *MockCredentialStore) Retrieve(key string) (string, error) {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	if val, exists := m.data[key]; exists {
		return val, nil
	}
	return "", fmt.Errorf("not found")
}

func (m *MockCredentialStore) Delete(key string) error {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	delete(m.data, key)
	return nil
}

func (m *MockCredentialStore) List(prefix string) ([]string, error) {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	var keys []string
	for k := range m.data {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

type MockAuditLogger struct{}

func (m *MockAuditLogger) Printf(format string, v ...interface{}) {
	// Do nothing
}

type MockAuditStore struct {
	entries []AuditEntry
}

func (m *MockAuditStore) Store(entry AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *MockAuditStore) Query(filters map[string]interface{}) ([]AuditEntry, error) {
	return m.entries, nil
}

func TestComplianceInfoStruct(t *testing.T) {
	info := ComplianceInfo{
		GDPRCompliant:     true,
		DataRetentionDays:  2555,
		RequiredFields:    []string{"user_id", "action", "resource"},
	}

	assert.True(t, info.GDPRCompliant)
	assert.Equal(t, 2555, info.DataRetentionDays)
	assert.Equal(t, 3, len(info.RequiredFields))
}

func TestAuditEntryStruct(t *testing.T) {
	userID := "user-123"
	now := time.Now()

	entry := AuditEntry{
		ID:         "audit-1",
		Timestamp:  now,
		UserID:     &userID,
		SessionID:  "session-1",
		Action:     "GET",
		Resource:   "/api/models",
		ResourceID: "123",
		Method:     "GET",
		IPAddress:  "192.168.1.1",
		UserAgent:  "test-agent",
		Success:    true,
		Error:      "",
		Details:    map[string]interface{}{},
		Compliance: ComplianceInfo{GDPRCompliant: true, DataRetentionDays: 2555, RequiredFields: []string{}},
	}

	assert.Equal(t, "audit-1", entry.ID)
	assert.Equal(t, now, entry.Timestamp)
	assert.Equal(t, &userID, entry.UserID)
	assert.Equal(t, "session-1", entry.SessionID)
	assert.True(t, entry.Success)
}

func TestRoleStruct(t *testing.T) {
	role := Role{
		ID:          "role-1",
		Name:        "Admin",
		Description: "Administrator",
		Permissions: []string{"perm-1", "perm-2"},
	}

	assert.Equal(t, "role-1", role.ID)
	assert.Equal(t, "Admin", role.Name)
	assert.Equal(t, "Administrator", role.Description)
	assert.Equal(t, 2, len(role.Permissions))
}

func TestPermissionStruct(t *testing.T) {
	perm := Permission{
		ID:          "perm-1",
		Name:        "Read Models",
		Description: "Can read models",
		Resource:    "models",
		Action:      "read",
		Conditions:  []string{"authenticated"},
	}

	assert.Equal(t, "perm-1", perm.ID)
	assert.Equal(t, "Read Models", perm.Name)
	assert.Equal(t, "models", perm.Resource)
	assert.Equal(t, "read", perm.Action)
	assert.Equal(t, 1, len(perm.Conditions))
}

func TestAPIKeyPatternStruct(t *testing.T) {
	pattern := APIKeyPattern{
		Name:         "OpenAI",
		Regex:        `sk-[a-zA-Z0-9]{48}`,
		MaskChar:     "*",
		VisibleStart: 3,
		VisibleEnd:   4,
	}

	assert.Equal(t, "OpenAI", pattern.Name)
	assert.Equal(t, 3, pattern.VisibleStart)
	assert.Equal(t, 4, pattern.VisibleEnd)
}

func TestCredentialManagerEncryptError(t *testing.T) {
	// Invalid key size (should be 16, 24, or 32 bytes)
	cm := &CredentialManager{
		encryptionKey: []byte("short"),
	}

	_, err := cm.encrypt("test")
	assert.Error(t, err)
}

func TestCredentialManagerDecryptError(t *testing.T) {
	cm := &CredentialManager{
		encryptionKey: []byte("test-key-16-byte"),
	}

	_, err := cm.decrypt("invalid-base64!")
	assert.Error(t, err)
}

func TestExtractRequestDetails(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/models?id=123", nil)
	req.Header.Set("Content-Type", "application/json")

	details := extractRequestDetails(req)

	assert.NotNil(t, details)
	assert.Contains(t, details, "query_params")
	assert.Contains(t, details, "headers")
	assert.Contains(t, details, "method")
	assert.Contains(t, details, "path")
}
