package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthManager(t *testing.T) {
	am := NewAuthManager("test-secret")

	assert.NotNil(t, am)
	assert.NotNil(t, am.clients)
	assert.Equal(t, []byte("test-secret"), am.jwtSecret)
}

func TestAuthManager_RegisterClient(t *testing.T) {
	am := NewAuthManager("test-secret")

	client, apiKey, err := am.RegisterClient(
		"Test Client",
		"Test Description",
		[]string{"read", "write"},
		100,
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotEmpty(t, apiKey)

	assert.Equal(t, "Test Client", client.Name)
	assert.Equal(t, "Test Description", client.Description)
	assert.Equal(t, []string{"read", "write"}, client.Permissions)
	assert.Equal(t, 100, client.RateLimit)
	assert.True(t, client.IsActive)
	assert.NotEmpty(t, client.APIKeyHash)
	assert.Contains(t, apiKey, "lv_")
}

func TestAuthManager_AuthenticateClient_Success(t *testing.T) {
	am := NewAuthManager("test-secret")

	_, apiKey, err := am.RegisterClient("Test Client", "desc", []string{"read"}, 100)
	require.NoError(t, err)

	client, err := am.AuthenticateClient(apiKey)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "Test Client", client.Name)
	assert.NotNil(t, client.LastUsedAt)
}

func TestAuthManager_AuthenticateClient_InvalidKey(t *testing.T) {
	am := NewAuthManager("test-secret")

	_, err := am.AuthenticateClient("invalid-api-key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

func TestAuthManager_AuthenticateClient_InactiveClient(t *testing.T) {
	am := NewAuthManager("test-secret")

	_, apiKey, err := am.RegisterClient("Test Client", "desc", []string{"read"}, 100)
	require.NoError(t, err)

	// Deactivate client
	am.clients[apiKey].IsActive = false

	_, err = am.AuthenticateClient(apiKey)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestAuthManager_AuthorizeRequest_WithPermission(t *testing.T) {
	am := NewAuthManager("test-secret")

	client := &Client{
		Name:        "Test Client",
		Permissions: []string{"read", "write"},
		IsActive:    true,
	}

	err := am.AuthorizeRequest(client, "read")
	assert.NoError(t, err)

	err = am.AuthorizeRequest(client, "write")
	assert.NoError(t, err)
}

func TestAuthManager_AuthorizeRequest_WildcardPermission(t *testing.T) {
	am := NewAuthManager("test-secret")

	client := &Client{
		Name:        "Admin Client",
		Permissions: []string{"*"},
		IsActive:    true,
	}

	err := am.AuthorizeRequest(client, "read")
	assert.NoError(t, err)

	err = am.AuthorizeRequest(client, "any-permission")
	assert.NoError(t, err)
}

func TestAuthManager_AuthorizeRequest_InsufficientPermissions(t *testing.T) {
	am := NewAuthManager("test-secret")

	client := &Client{
		Name:        "Read-Only Client",
		Permissions: []string{"read"},
		IsActive:    true,
	}

	err := am.AuthorizeRequest(client, "write")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
}

func TestAuthManager_AuthorizeRequest_InactiveClient(t *testing.T) {
	am := NewAuthManager("test-secret")

	client := &Client{
		Name:        "Test Client",
		Permissions: []string{"read"},
		IsActive:    false,
	}

	err := am.AuthorizeRequest(client, "read")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestAuthManager_GenerateJWTToken(t *testing.T) {
	am := NewAuthManager("test-secret-key-12345")

	client := &Client{
		ID:          1,
		Name:        "Test Client",
		Permissions: []string{"read", "write"},
		IsActive:    true,
	}

	token, err := am.GenerateJWTToken(client, 24*time.Hour)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthManager_ValidateJWTToken_Success(t *testing.T) {
	am := NewAuthManager("test-secret-key-12345")

	client := &Client{
		ID:          1,
		Name:        "Test Client",
		Permissions: []string{"read", "write"},
		IsActive:    true,
	}

	token, err := am.GenerateJWTToken(client, 24*time.Hour)
	require.NoError(t, err)

	claims, err := am.ValidateJWTToken(token)

	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, int64(1), claims.ClientID)
	assert.Equal(t, "Test Client", claims.ClientName)
	assert.Equal(t, []string{"read", "write"}, claims.Permissions)
}

func TestAuthManager_ValidateJWTToken_InvalidToken(t *testing.T) {
	am := NewAuthManager("test-secret")

	_, err := am.ValidateJWTToken("invalid-token")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse token")
}

func TestAuthManager_ValidateJWTToken_WrongSecret(t *testing.T) {
	am1 := NewAuthManager("secret-1")
	am2 := NewAuthManager("secret-2")

	client := &Client{
		ID:          1,
		Name:        "Test",
		Permissions: []string{"read"},
	}

	token, err := am1.GenerateJWTToken(client, time.Hour)
	require.NoError(t, err)

	_, err = am2.ValidateJWTToken(token)
	require.Error(t, err)
}

func TestAuthManager_CheckRateLimit(t *testing.T) {
	am := NewAuthManager("test-secret")

	// Current implementation always returns nil
	err := am.CheckRateLimit(1, 100)
	assert.NoError(t, err)
}

func TestAuthManager_GetClients(t *testing.T) {
	am := NewAuthManager("test-secret")

	// Register multiple clients
	_, _, _ = am.RegisterClient("Client 1", "desc", []string{"read"}, 100)
	_, _, _ = am.RegisterClient("Client 2", "desc", []string{"write"}, 200)

	clients := am.GetClients()

	assert.Len(t, clients, 2)
	for _, c := range clients {
		assert.Empty(t, c.APIKeyHash) // Sensitive data should be stripped
	}
}

func TestAuthManager_GetClientUsage(t *testing.T) {
	am := NewAuthManager("test-secret")

	usage, err := am.GetClientUsage(1)

	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, int64(1), usage.ClientID)
	assert.Equal(t, 0, usage.RequestsToday)
	assert.Equal(t, 0, usage.RequestsThisHour)
}

func TestExtractAPIKeyFromHeader(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		expectedKey string
		expectError bool
	}{
		{"valid bearer", "Bearer test-api-key", "test-api-key", false},
		{"valid Bearer uppercase", "BEARER test-api-key", "test-api-key", false},
		{"empty header", "", "", true},
		{"no bearer prefix", "test-api-key", "", true},
		{"basic auth", "Basic abc123", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ExtractAPIKeyFromHeader(tt.header)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedKey, key)
			}
		})
	}
}

func TestAuthManager_RequirePermission(t *testing.T) {
	am := NewAuthManager("test-secret")

	checkFunc := am.RequirePermission("admin")

	client := &Client{
		Permissions: []string{"admin"},
		IsActive:    true,
	}

	err := checkFunc(client)
	assert.NoError(t, err)

	clientNoAdmin := &Client{
		Permissions: []string{"read"},
		IsActive:    true,
	}

	err = checkFunc(clientNoAdmin)
	assert.Error(t, err)
}

func TestAuthManager_ValidateAndExtractClaims(t *testing.T) {
	am := NewAuthManager("test-secret")

	// Register and authenticate a client
	_, apiKey, err := am.RegisterClient("Test Client", "desc", []string{"read"}, 100)
	require.NoError(t, err)

	client, err := am.AuthenticateClient(apiKey)
	require.NoError(t, err)

	// Generate token
	token, err := am.GenerateJWTToken(client, time.Hour)
	require.NoError(t, err)

	// Validate and extract
	extractedClient, err := am.ValidateAndExtractClaims(token)
	require.NoError(t, err)
	assert.Equal(t, client.ID, extractedClient.ID)
}

func TestAuthManager_ValidateAndExtractClaims_ClientNotFound(t *testing.T) {
	am := NewAuthManager("test-secret")

	// Create a token for a non-existent client
	client := &Client{
		ID:          999999,
		Name:        "Non-existent",
		Permissions: []string{"read"},
	}

	token, err := am.GenerateJWTToken(client, time.Hour)
	require.NoError(t, err)

	_, err = am.ValidateAndExtractClaims(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not found")
}

func TestAuthManager_EnableLDAP(t *testing.T) {
	am := NewAuthManager("test-secret")
	assert.False(t, am.ldapEnabled)

	am.EnableLDAP()
	assert.True(t, am.ldapEnabled)
}

func TestAuthManager_EnableRBAC(t *testing.T) {
	am := NewAuthManager("test-secret")
	assert.False(t, am.rbacEnabled)

	am.EnableRBAC()
	assert.True(t, am.rbacEnabled)
}

func TestAuthManager_EnableSSO(t *testing.T) {
	am := NewAuthManager("test-secret")
	assert.False(t, am.ssoEnabled)

	am.EnableSSO()
	assert.True(t, am.ssoEnabled)
}

func TestAuthManager_AuthenticateWithLDAP_Disabled(t *testing.T) {
	am := NewAuthManager("test-secret")

	_, err := am.AuthenticateWithLDAP("user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestAuthManager_AuthenticateWithLDAP_NotConfigured(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableLDAP()

	// Without configuring an LDAP manager, authentication fails
	_, err := am.AuthenticateWithLDAP("ldap-user", "ldap-pass")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestAuthManager_AuthenticateWithLDAP_WithManager(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableLDAP()

	// Configure an LDAP manager (will fail to connect but tests the flow)
	config := &LDAPConfig{
		Host:   "ldap.example.com",
		Port:   389,
		BaseDN: "dc=example,dc=com",
	}
	ldapMgr, _ := NewLDAPManager(config)
	am.SetLDAPManager(ldapMgr)

	// Will fail because there's no actual LDAP server
	_, err := am.AuthenticateWithLDAP("wrong-user", "wrong-pass")
	require.Error(t, err)
}

func TestAuthManager_CheckRBACPermission_NoPermissions(t *testing.T) {
	am := NewAuthManager("test-secret")

	client := &Client{
		Name:        "Test",
		Permissions: []string{},
		IsActive:    true,
	}

	// Even with RBAC disabled, permissions are still checked (security by default)
	// A client with no permissions will be denied
	err := am.CheckRBACPermission(client, "resource", "action")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RBAC access denied")
}

func TestAuthManager_CheckRBACPermission_Success(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableRBAC()

	client := &Client{
		Name:        "Test",
		Permissions: []string{"resource:action"},
		IsActive:    true,
	}

	err := am.CheckRBACPermission(client, "resource", "action")
	assert.NoError(t, err)
}

func TestAuthManager_CheckRBACPermission_Admin(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableRBAC()

	client := &Client{
		Name:        "Admin",
		Permissions: []string{"admin"},
		IsActive:    true,
	}

	err := am.CheckRBACPermission(client, "any-resource", "any-action")
	assert.NoError(t, err)
}

func TestAuthManager_CheckRBACPermission_Wildcard(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableRBAC()

	client := &Client{
		Name:        "Super User",
		Permissions: []string{"*"},
		IsActive:    true,
	}

	err := am.CheckRBACPermission(client, "any-resource", "any-action")
	assert.NoError(t, err)
}

func TestAuthManager_CheckRBACPermission_Denied(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableRBAC()

	client := &Client{
		Name:        "Limited User",
		Permissions: []string{"read"},
		IsActive:    true,
	}

	err := am.CheckRBACPermission(client, "resource", "write")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RBAC access denied")
}

func TestAuthManager_CheckRBACPermission_InactiveClient(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableRBAC()

	client := &Client{
		Name:        "Inactive User",
		Permissions: []string{"*"},
		IsActive:    false,
	}

	err := am.CheckRBACPermission(client, "resource", "action")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is inactive")
}

func TestAuthManager_AuthenticateWithSSO_Disabled(t *testing.T) {
	am := NewAuthManager("test-secret")

	_, err := am.AuthenticateWithSSO("google", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestAuthManager_AuthenticateWithSSO_NotConfigured(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableSSO()

	// Without configuring an SSO provider, authentication fails
	_, err := am.AuthenticateWithSSO("google", "google-token-12345")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestAuthManager_AuthenticateWithSSO_WithProvider(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableSSO()

	// Register a Google SSO provider via the SSO manager
	ssoMgr := GetSSOManager()
	ssoMgr.AddProvider(&SSOConfig{
		Provider:     "google",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://www.googleapis.com/oauth2/v3/userinfo",
		Issuer:       "https://accounts.google.com",
	})

	// Will fail because token is not a valid JWT (wrong format)
	_, err := am.AuthenticateWithSSO("google", "invalid-token")
	require.Error(t, err)
}

func TestAuthManager_AuthenticateWithSSO_UnknownProvider(t *testing.T) {
	am := NewAuthManager("test-secret")
	am.EnableSSO()

	_, err := am.AuthenticateWithSSO("unknown-provider", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestAuthManager_CreateRole(t *testing.T) {
	// §11.4 anti-bluff (round-28): previously CreateRole only fmt.Printf'd,
	// so the test "just verify it doesn't panic" was a bluff — there was
	// nothing to verify. The contract now persists the role + permissions
	// to the in-memory roles map, retrievable via GetRolePermissions.
	am := NewAuthManager("test-secret")

	am.CreateRole("custom_role", []string{"read", "write", "delete"})
	perms := am.GetRolePermissions("custom_role")
	require.NotNil(t, perms, "CreateRole must persist the role (round-28 anti-bluff fix)")
	assert.ElementsMatch(t, []string{"read", "write", "delete"}, perms)

	// Defensive-copy invariant: mutating the returned slice must not
	// corrupt the stored permission set.
	perms[0] = "MUTATED"
	freshPerms := am.GetRolePermissions("custom_role")
	assert.NotContains(t, freshPerms, "MUTATED", "GetRolePermissions must return defensive copy")

	// Unknown role → nil.
	assert.Nil(t, am.GetRolePermissions("no-such-role"))
}

func TestAuthManager_AssignRoleToClient_ClientNotFound(t *testing.T) {
	am := NewAuthManager("test-secret")

	err := am.AssignRoleToClient(1, "admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not found")
}

// TestAuthManager_AssignRoleToClient_AppliesPermissions confirms the
// round-28 RBAC fix: with a real client registered, assigning a default
// role must merge that role's permissions into the client's permission
// set (deduplicated). Previously the method looked up `am.clients["dummy"]`
// — a key that never existed — so this path never produced any effect.
func TestAuthManager_AssignRoleToClient_AppliesPermissions(t *testing.T) {
	am := NewAuthManager("test-secret")

	// Register a real client via the existing registration API.
	client, _, err := am.RegisterClient("test-client", "desc", []string{"read"}, 60)
	require.NoError(t, err)

	require.NoError(t, am.AssignRoleToClient(client.ID, "editor"))

	// editor default permissions per NewAuthManager: {"read", "write", "delete"}
	// merged with the client's existing {"read"} (dedup) → {"read","write","delete"}.
	am.mu.RLock()
	stored := am.clientsByID[client.ID]
	am.mu.RUnlock()
	require.NotNil(t, stored)
	assert.Contains(t, stored.Permissions, "read")
	assert.Contains(t, stored.Permissions, "write")
	assert.Contains(t, stored.Permissions, "delete")

	// No duplicates: "read" should appear exactly once.
	readCount := 0
	for _, p := range stored.Permissions {
		if p == "read" {
			readCount++
		}
	}
	assert.Equal(t, 1, readCount, "dedup must collapse duplicate 'read'")
}

// TestAuthManager_AssignRoleToClient_UnknownRole asserts the contract for
// roles that have never been registered. Default roles registered in
// NewAuthManager ("admin","editor","viewer","api") are known; anything else
// must be refused.
func TestAuthManager_AssignRoleToClient_UnknownRole(t *testing.T) {
	am := NewAuthManager("test-secret")
	client, _, err := am.RegisterClient("test-client", "desc", []string{"read"}, 60)
	require.NoError(t, err)

	err = am.AssignRoleToClient(client.ID, "definitely-not-a-role")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown role")
}

func TestAuthManager_AuditAuthEvent(t *testing.T) {
	// §11.4 anti-bluff (round-28): previously AuditAuthEvent only fmt.Printf'd
	// the event. No record persisted. This test asserts the new in-memory
	// ring-buffer contract: every call leaves a retrievable record.
	am := NewAuthManager("test-secret")

	am.AuditAuthEvent("LOGIN", "client-1", "successful login")
	am.AuditAuthEvent("LOGIN_FAIL", "client-2", "bad credentials")

	log := am.GetAuditLog()
	require.Len(t, log, 2)
	assert.Equal(t, "LOGIN", log[0].EventType)
	assert.Equal(t, "client-1", log[0].ClientID)
	assert.Equal(t, "success", log[0].Status, "LOGIN without FAIL must be Status=success")
	assert.Equal(t, "LOGIN_FAIL", log[1].EventType)
	assert.Equal(t, "failure", log[1].Status, "event containing 'FAIL' must be Status=failure")
	assert.Equal(t, "successful login", log[0].Details["message"])
}

// TestAuthManager_AuditLog_FIFOOverflow asserts the bounded-buffer behaviour
// added in round-28. With capacity 1024, appending 1025 entries must drop
// the oldest and retain the newest.
func TestAuthManager_AuditLog_FIFOOverflow(t *testing.T) {
	am := NewAuthManager("test-secret")

	for i := 0; i < auditLogCapacity+5; i++ {
		am.AuditAuthEvent("EVT", fmt.Sprintf("client-%d", i), "details")
	}

	log := am.GetAuditLog()
	assert.Len(t, log, auditLogCapacity, "ring buffer must cap at auditLogCapacity")
	// Oldest 5 should have been dropped; first surviving entry's clientID
	// must reflect the 5th-from-start (clientID "client-5").
	assert.Equal(t, "client-5", log[0].ClientID, "FIFO overflow must drop oldest entries first")
	assert.Equal(t, fmt.Sprintf("client-%d", auditLogCapacity+4), log[len(log)-1].ClientID,
		"newest entry must be preserved at tail")
}

// TestSSOManager_ValidateToken_RefusesUnsignedJWT pins the round-28 fix for
// the JWT-signature-bypass bluff: ValidateToken must refuse any well-formed
// JWT-shaped string with ErrJWTSignatureVerificationNotImplemented rather
// than fabricating an SSOUserInfo.
func TestSSOManager_ValidateToken_RefusesUnsignedJWT(t *testing.T) {
	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{Provider: "google", Issuer: "https://accounts.google.com"})

	// A well-formed 3-part JWT-shaped string with base64-valid payload would
	// previously have been accepted as a verified SSO authentication.
	// header.payload.signature — payload is base64url("{\"sub\":\"victim\"}").
	wellFormed := "aGVhZGVy.eyJzdWIiOiJ2aWN0aW0ifQ.c2ln"

	info, err := sm.ValidateToken("google", wellFormed)
	require.Error(t, err, "round-28 anti-bluff: ValidateToken MUST refuse unsigned JWT")
	assert.Nil(t, info, "ValidateToken MUST NOT fabricate SSOUserInfo when signature is unverified")
	assert.ErrorIs(t, err, ErrJWTSignatureVerificationNotImplemented,
		"refusal MUST surface the specific JWKS-not-implemented sentinel")
}

func TestAuthManager_HashAPIKey(t *testing.T) {
	am := NewAuthManager("test-secret")

	// Generate and hash an API key
	apiKey := "lv_testkey12345678_suffix1234567"
	hash, err := am.hashAPIKey(apiKey)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Contains(t, hash, "$argon2id$")
}

func TestAuthManager_VerifyAPIKey_InvalidHashFormat(t *testing.T) {
	am := NewAuthManager("test-secret")

	_, err := am.verifyAPIKey("key", "invalid-hash-format")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hash format")
}

func TestClient_Struct(t *testing.T) {
	now := time.Now()
	client := Client{
		ID:          1,
		Name:        "Test Client",
		Description: "Test Description",
		APIKey:      "api-key",
		APIKeyHash:  "hash",
		Permissions: []string{"read", "write"},
		RateLimit:   100,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
		LastUsedAt:  &now,
	}

	assert.Equal(t, int64(1), client.ID)
	assert.Equal(t, "Test Client", client.Name)
	assert.Equal(t, 100, client.RateLimit)
	assert.True(t, client.IsActive)
}

func TestClientUsage_Struct(t *testing.T) {
	now := time.Now()
	usage := ClientUsage{
		ClientID:         1,
		RequestsToday:    50,
		RequestsThisHour: 10,
		TotalRequests:    1000,
		LastRequestAt:    &now,
		DailyResetAt:     now.AddDate(0, 0, 1),
		HourlyResetAt:    now.Add(time.Hour),
	}

	assert.Equal(t, int64(1), usage.ClientID)
	assert.Equal(t, 50, usage.RequestsToday)
	assert.Equal(t, 10, usage.RequestsThisHour)
	assert.Equal(t, 1000, usage.TotalRequests)
}

func TestJWTClaims_Struct(t *testing.T) {
	claims := JWTClaims{
		ClientID:    1,
		ClientName:  "Test",
		Permissions: []string{"read"},
	}

	assert.Equal(t, int64(1), claims.ClientID)
	assert.Equal(t, "Test", claims.ClientName)
	assert.Len(t, claims.Permissions, 1)
}
