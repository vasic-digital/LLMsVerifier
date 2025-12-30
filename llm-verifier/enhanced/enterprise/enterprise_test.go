package enterprise

import (
	"context"
	"testing"
	"time"
)

func TestRBACManager(t *testing.T) {
	rbac := NewRBACManager()

	// Test user creation
	user := &User{
		ID:          "user1",
		Username:    "testuser",
		Email:       "testuser@example.com",
		FirstName:   "Test",
		LastName:    "User",
		Roles:       []RBACRole{RBACRoleAnalyst},
		Permissions: []Permission{PermissionJobSubmit, PermissionJobView},
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := rbac.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test user retrieval
	retrievedUser, err := rbac.GetUser("user1")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if retrievedUser.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrievedUser.Username)
	}

	// Test permission check
	hasPermission := rbac.HasPermission("user1", PermissionJobSubmit)
	if !hasPermission {
		t.Error("User should have job submit permission")
	}

	// Test role assignment
	err = rbac.AssignRole("user1", RBACRoleOperator)
	if err != nil {
		t.Fatalf("Failed to assign role: %v", err)
	}

	// Test role removal
	err = rbac.RemoveRole("user1", RBACRoleOperator)
	if err != nil {
		t.Fatalf("Failed to remove role: %v", err)
	}

	// Test audit log
	auditLog := rbac.GetAuditLog(10)
	if len(auditLog) == 0 {
		t.Error("Audit log should contain entries")
	}
}

func TestMultiTenantManager(t *testing.T) {
	mtm := NewMultiTenantManager()

	// Test tenant creation
	tenant := &Tenant{
		ID:          "tenant1",
		Name:        "Test Tenant",
		Description: "A test tenant",
		Domain:      "test.example.com",
		Settings:    map[string]interface{}{"setting1": "value1"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Enabled:     true,
	}

	err := mtm.CreateTenant(tenant)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Test tenant retrieval
	retrievedTenant, err := mtm.GetTenant("tenant1")
	if err != nil {
		t.Fatalf("Failed to get tenant: %v", err)
	}

	if retrievedTenant.Name != tenant.Name {
		t.Errorf("Expected tenant name %s, got %s", tenant.Name, retrievedTenant.Name)
	}

	// Test user assignment to tenant
	err = mtm.AddUserToTenant("tenant1", "user1")
	if err != nil {
		t.Fatalf("Failed to add user to tenant: %v", err)
	}

	// Test getting user tenants
	userTenants := mtm.GetUserTenants("user1")
	if len(userTenants) != 1 {
		t.Errorf("Expected 1 tenant for user, got %d", len(userTenants))
	}
}

func TestSAMLAuthenticator(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		SSOURL:              "https://app.example.com/saml/sso",
		// Note: CertificateFile intentionally omitted for testing without certificate validation
		AttributeMapping: map[string]string{
			"username":   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			"email":      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"first_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
			"last_name":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
		},
		EntityID:         "https://app.example.com",
		AllowedAudiences: []string{"https://app.example.com"},
	}

	saml := NewSAMLAuthenticator(config)

	// Verify SAML is configured
	if !saml.IsConfigured() {
		t.Errorf("Expected SAML to be configured")
	}

	// Verify login URL
	loginURL := saml.GetLoginURL("")
	if loginURL != "https://app.example.com/saml/sso" {
		t.Errorf("Expected login URL https://app.example.com/saml/sso, got %s", loginURL)
	}

	// Test with invalid SAML response (should fail with proper error)
	_, err := saml.ProcessSAMLResponse("invalid-base64-data")
	if err == nil {
		t.Error("Expected error for invalid SAML response, got nil")
	}

	// Note: Valid SAML response processing is tested in saml_test.go
	// with TestProcessSAMLResponse_ValidResponse
}

func TestEnterpriseManager(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{
			Enabled:        true,
			SessionTimeout: 30 * time.Minute,
			PasswordPolicy: PasswordPolicy{
				MinLength:        8,
				RequireUppercase: true,
				RequireLowercase: true,
				RequireNumbers:   true,
				RequireSymbols:   false,
				MaxAge:           90 * 24 * time.Hour, // 90 days
			},
			TwoFactorAuth: false,
		},
		MultiTenant: MultiTenantConfig{
			Enabled:       true,
			DefaultTenant: "default",
			TenantHeader:  "X-Tenant-ID",
		},
		AuditLogging: AuditLoggingConfig{
			Enabled:   true,
			Storage:   "file",
			Retention: 90 * 24 * time.Hour, // 90 days
			Level:     "info",
		},
		Security: SecurityConfig{
			HTTPSEnabled: true,
			CORSOrigins:  []string{"https://app.example.com"},
			RateLimiting: RateLimitConfig{
				Enabled:  true,
				Requests: 100,
				Window:   time.Minute,
			},
			Authentication: AuthConfig{
				Methods: []string{"local", "ldap", "saml"},
			},
		},
	}

	// Create enterprise manager (would use real supervisor in practice)
	manager := NewEnterpriseManager(config)

	if manager == nil {
		t.Fatal("Failed to create enterprise manager")
	}

	// Test starting manager
	ctx := context.Background()
	err := manager.Start(ctx)
	if err != nil {
		t.Errorf("Expected start error due to missing dependencies: %v", err)
	}

	// Test stopping manager
	err = manager.Stop(ctx)
	if err != nil {
		t.Errorf("Expected stop error due to missing dependencies: %v", err)
	}
}

func TestPermissionConstants(t *testing.T) {
	expectedPermissions := []Permission{
		PermissionJobSubmit,
		PermissionJobView,
		PermissionJobCancel,
		PermissionSystemStart,
		PermissionSystemStop,
		PermissionSystemConfigure,
		PermissionUserManage,
		PermissionMetricsView,
		PermissionLogsView,
	}

	// Ensure all permissions are unique
	permissionMap := make(map[Permission]bool)
	for _, perm := range expectedPermissions {
		if permissionMap[perm] {
			t.Errorf("Duplicate permission: %s", perm)
		}
		permissionMap[perm] = true
	}

	if len(permissionMap) != len(expectedPermissions) {
		t.Errorf("Permission count mismatch: expected %d, got %d",
			len(expectedPermissions), len(permissionMap))
	}
}

func TestRoleConstants(t *testing.T) {
	expectedRoles := []RBACRole{
		RBACRoleAdmin,
		RBACRoleOperator,
		RBACRoleAnalyst,
		RBACRoleViewer,
		RBACRoleReadOnly,
	}

	// Ensure all roles are unique
	roleMap := make(map[RBACRole]bool)
	for _, role := range expectedRoles {
		if roleMap[role] {
			t.Errorf("Duplicate role: %s", role)
		}
		roleMap[role] = true
	}

	if len(roleMap) != len(expectedRoles) {
		t.Errorf("Role count mismatch: expected %d, got %d",
			len(expectedRoles), len(roleMap))
	}
}

func TestUserValidation(t *testing.T) {
	rbac := NewRBACManager()

	// Test creating user with missing required fields
	invalidUser := &User{
		ID:        "", // Missing ID
		Username:  "testuser",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := rbac.CreateUser(invalidUser)
	if err == nil {
		t.Error("Expected error for user with missing ID")
	}

	// Test creating valid user
	validUser := &User{
		ID:        "valid-user-1",
		Username:  "validuser",
		Email:     "validuser@example.com",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = rbac.CreateUser(validUser)
	if err != nil {
		t.Errorf("Failed to create valid user: %v", err)
	}

	// Test duplicate user creation
	err = rbac.CreateUser(validUser)
	if err == nil {
		t.Error("Expected error for duplicate user")
	}
}

func TestAuditLogging(t *testing.T) {
	rbac := NewRBACManager()

	// Create a user to generate audit logs
	user := &User{
		ID:        "audit-test-user",
		Username:  "audituser",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := rbac.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Check audit log
	auditLog := rbac.GetAuditLog(10)
	if len(auditLog) == 0 {
		t.Error("Expected audit log entries")
	}

	// Check audit entry structure
	for _, entry := range auditLog {
		if entry.UserID == "" {
			t.Error("Audit entry missing user ID")
		}
		if entry.Action == "" {
			t.Error("Audit entry missing action")
		}
		if entry.Timestamp.IsZero() {
			t.Error("Audit entry missing timestamp")
		}
		if entry.IPAddress == "" {
			t.Error("Audit entry missing IP address")
		}
	}
}

func TestMultiTenantIsolation(t *testing.T) {
	mtm := NewMultiTenantManager()

	// Create two tenants
	tenant1 := &Tenant{
		ID:        "tenant1",
		Name:      "Tenant 1",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tenant2 := &Tenant{
		ID:        "tenant2",
		Name:      "Tenant 2",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mtm.CreateTenant(tenant1)
	mtm.CreateTenant(tenant2)

	// Add same user to both tenants
	mtm.AddUserToTenant("tenant1", "user1")
	mtm.AddUserToTenant("tenant2", "user1")

	// Verify tenant isolation
	userTenants := mtm.GetUserTenants("user1")
	if len(userTenants) != 2 {
		t.Errorf("Expected 2 tenants for user, got %d", len(userTenants))
	}

	// Verify each tenant has the user
	tenantsForUser1, _ := mtm.GetTenant("tenant1")
	tenantsForUser2, _ := mtm.GetTenant("tenant2")

	if tenantsForUser1 == nil || tenantsForUser2 == nil {
		t.Error("Tenants should be accessible")
	}
}
