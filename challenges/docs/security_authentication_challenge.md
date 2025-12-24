# Security and Authentication Comprehensive Challenge

## Overview
This challenge validates security and authentication including RBAC, multi-tenancy, audit logging, SSO integration, and API key management.

## Challenge Type
Security Test + Authentication Test + Authorization Test

## Test Scenarios

### 1. API Key Authentication Challenge
**Objective**: Verify API key authentication works

**Steps**:
1. Generate API key
2. Make request with API key
3. Verify authentication succeeds
4. Make request with invalid key
5. Verify authentication fails

**Expected Results**:
- Valid keys authenticate
- Invalid keys rejected
- Proper error messages

**Test Code**:
```go
func TestAPIKeyAuthentication(t *testing.T) {
    client := NewAPIClient()

    // Valid key
    client.SetAPIKey("valid-api-key")
    resp, err := client.GetModels()
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Invalid key
    client.SetAPIKey("invalid-key")
    resp, err = client.GetModels()
    assert.Error(t, err)
    assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
```

---

### 2. JWT Token Authentication Challenge
**Objective**: Verify JWT token authentication works

**Steps**:
1. Login and get JWT token
2. Make request with token
3. Verify authentication succeeds
4. Wait for token to expire
5. Verify authentication fails
6. Refresh token

**Expected Results**:
- Tokens issued correctly
- Valid tokens authenticate
- Expired tokens rejected
- Token refresh works

**Test Code**:
```go
func TestJWTAuthentication(t *testing.T) {
    auth := NewAuthService()

    // Login
    token, err := auth.Login("admin", "password")
    assert.NoError(t, err)
    assert.NotEmpty(t, token)

    // Use token
    client := NewAPIClient()
    client.SetJWTToken(token)
    resp, err := client.GetModels()
    assert.NoError(t, err)

    // Verify token
    claims, err := auth.VerifyToken(token)
    assert.NoError(t, err)
    assert.Equal(t, "admin", claims.Username)
}
```

---

### 3. Role-Based Access Control (RBAC) Challenge
**Objective**: Verify RBAC works correctly

**Roles**:
- Admin: Full access
- User: Read + write own data
- Viewer: Read-only access

**Steps**:
1. Create users with different roles
2. Test admin access
3. Test user access
4. Test viewer access
5. Test unauthorized actions

**Expected Results**:
- Admins have full access
- Users limited to own data
- Viewers only read
- Unauthorized actions blocked

**Test Code**:
```go
func TestRBAC(t *testing.T) {
    adminClient := NewClient("admin", "password")
    userClient := NewClient("user", "password")
    viewerClient := NewClient("viewer", "password")

    // Admin can do everything
    _, err := adminClient.CreateProvider("openai")
    assert.NoError(t, err)

    // User can't create providers
    _, err = userClient.CreateProvider("anthropic")
    assert.Error(t, err)

    // Viewer can't write
    _, err = viewerClient.UpdateModel("gpt-4", Model{Score: 95})
    assert.Error(t, err)
}
```

---

### 4. Multi-Tenancy Challenge
**Objective**: Verify multi-tenancy isolation

**Steps**:
1. Create tenant A
2. Create tenant B
3. Add data to tenant A
4. Query from tenant B
5. Verify isolation
6. Verify tenant-specific API keys

**Expected Results**:
- Tenants isolated
- Cross-tenant access blocked
- Tenant-specific keys work only for that tenant

**Test Code**:
```go
func TestMultiTenancy(t *testing.T) {
    clientA := NewClient("tenant-a", "key-a")
    clientB := NewClient("tenant-b", "key-b")

    // Tenant A creates model
    clientA.CreateModel(Model{ID: "gpt-4"})

    // Tenant B can't access
    models, err := clientB.GetModels()
    assert.NoError(t, err)
    assert.Equal(t, 0, len(models)) // Can't see A's models
}
```

---

### 5. Audit Logging Challenge
**Objective**: Verify all actions are logged

**Actions to Log**:
- Login/logout
- API key generation/revocation
- Model discovery
- Configuration exports
- Score changes

**Steps**:
1. Perform various actions
2. Query audit logs
3. Verify all actions logged
4. Verify metadata captured
5. Verify user and timestamp

**Expected Results**:
- All actions logged
- Metadata captured
- User and timestamp recorded
- Logs searchable

**Test Code**:
```go
func TestAuditLogging(t *testing.T) {
    audit := NewAuditLogger()

    // Perform action
    performAction("discover_models", map[string]interface{}{
        "provider": "openai",
    })

    // Query logs
    logs := audit.Query(AuditQuery{
        User: "testuser",
        Action: "discover_models",
    })

    assert.Equal(t, 1, len(logs))
    assert.Equal(t, "discover_models", logs[0].Action)
    assert.Contains(t, logs[0].Metadata, "provider")
}
```

---

### 6. SSO Integration Challenge
**Objective**: Verify Single Sign-On integration

**Providers**:
- LDAP
- SAML
- OAuth2 (Google, GitHub)

**Steps**:
1. Configure SSO provider
2. Initiate SSO login
3. Complete SSO flow
4. Verify user logged in
5. Verify session created

**Expected Results**:
- SSO flow completes
- User authenticated
- Session created
- Logout works

**Test Code**:
```go
func TestSSOIntegration(t *testing.T) {
    sso := NewSSOProvider("saml", "saml-config.xml")

    // Initiate login
    redirectURL := sso.InitiateLogin("user@example.com")
    assert.NotEmpty(t, redirectURL)

    // Complete SSO
    session, err := sso.CompleteLogin(redirectURL)
    assert.NoError(t, err)
    assert.NotEmpty(t, session.Token)
}
```

---

### 7. API Key Management Challenge
**Objective**: Verify API key lifecycle

**Steps**:
1. Generate API key
2. List API keys
3. Set expiration
4. Revoke API key
5. Verify revoked key fails

**Expected Results**:
- Keys generated securely
- Keys listed
- Expiration enforced
- Revocation works
- Revoked keys invalid

**Test Code**:
```go
func TestAPIKeyManagement(t *testing.T) {
    manager := NewAPIKeyManager()

    // Generate key
    key, err := manager.GenerateKey("test-key", 24*time.Hour)
    assert.NoError(t, err)
    assert.NotEmpty(t, key.Key)
    assert.False(t, key.IsExpired())

    // Revoke
    err = manager.RevokeKey(key.ID)
    assert.NoError(t, err)

    // Verify revoked
    keys, _ := manager.ListKeys()
    revoked := findKey(keys, key.ID)
    assert.True(t, revoked.Revoked)
}
```

---

### 8. Password Security Challenge
**Objective**: Verify password security

**Steps**:
1. Create user with weak password
2. Verify rejected
3. Create user with strong password
4. Verify accepted
5. Verify password hashed
6. Test password change

**Expected Results**:
- Weak passwords rejected
- Strong passwords accepted
- Passwords hashed
- Passwords never stored plain

**Test Code**:
```go
func TestPasswordSecurity(t *testing.T) {
    auth := NewAuthService()

    // Weak password rejected
    err := auth.CreateUser("user", "password123")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "weak password")

    // Strong password accepted
    err = auth.CreateUser("user", "Str0ng!P@ssw0rd")
    assert.NoError(t, err)

    // Verify hashed
    user := auth.GetUser("user")
    assert.NotEqual(t, "Str0ng!P@ssw0rd", user.PasswordHash)
}
```

---

### 9. Session Management Challenge
**Objective**: Verify session security

**Steps**:
1. Login and get session
2. Use session for requests
3. Verify session expires
4. Verify session invalidated on logout
5. Verify session revoked

**Expected Results**:
- Sessions created
- Sessions used
- Sessions expire
- Logout invalidates
- Revocation works

**Test Code**:
```go
func TestSessionManagement(t *testing.T) {
    auth := NewAuthService()

    // Login
    session, err := auth.Login("user", "password")
    assert.NoError(t, err)

    // Use session
    resp, err := makeRequest(session.Token)
    assert.NoError(t, err)

    // Logout
    err = auth.Logout(session.Token)
    assert.NoError(t, err)

    // Session invalid
    resp, err = makeRequest(session.Token)
    assert.Error(t, err)
    assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
```

---

### 10. Security Headers Challenge
**Objective**: Verify security headers are set

**Headers**:
- X-Frame-Options
- X-Content-Type-Options
- X-XSS-Protection
- Content-Security-Policy
- Strict-Transport-Security

**Steps**:
1. Make API request
2. Check response headers
3. Verify all security headers present
4. Verify header values correct

**Expected Results**:
- All security headers present
- Header values correct
- Headers applied to all endpoints

**Test Code**:
```go
func TestSecurityHeaders(t *testing.T) {
    resp, err := http.Get("http://localhost:8080/api/v1/models")
    assert.NoError(t, err)

    headers := resp.Header

    assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
    assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
    assert.Contains(t, headers.Get("Content-Security-Policy"), "default-src")
}
```

---

## Success Criteria

### Functional Requirements
- [ ] API key authentication works
- [ ] JWT authentication works
- [ ] RBAC works
- [ ] Multi-tenancy works
- [ ] Audit logging works
- [ ] SSO integration works
- [ ] API key management works
- [ ] Password security works
- [ ] Session management works
- [ ] Security headers present

### Security Requirements
- [ ] Passwords hashed with bcrypt/argon2
- [ ] JWT tokens signed with RS256
- [ ] API keys encrypted at rest
- [ ] All actions audited
- [ ] Sessions have expiration
- [ ] TLS enforced in production

### Compliance Requirements
- [ ] GDPR compliance (data deletion)
- [ ] Data encryption at rest
- [ ] Data encryption in transit
- [ ] Audit retention policy
- [ ] Access control enforcement

## Dependencies
- SSO provider configured
- TLS certificates
- Encryption keys

## Cleanup
- Delete test users
- Revoke test keys
- Clear audit logs
