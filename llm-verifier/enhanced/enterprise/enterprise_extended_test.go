package enterprise

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"llm-verifier/monitoring"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== EnterpriseMonitor Format Tests ====================

func TestEnterpriseMonitor_FormatForSplunk(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		Splunk: SplunkConfig{
			Source:     "llm-verifier",
			Sourcetype: "llm:metrics",
			Fields:     map[string]string{"env": "test"},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	data := map[string]interface{}{
		"health_score": 95.5,
		"status":       "healthy",
	}

	result := monitor.formatForSplunk(data)
	assert.NotEmpty(t, result)

	// Parse and verify JSON structure
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "llm-verifier", parsed["host"])
	assert.Equal(t, "llm-verifier", parsed["source"])
	assert.Equal(t, "llm:metrics", parsed["sourcetype"])
	assert.Equal(t, "test", parsed["env"])
}

func TestEnterpriseMonitor_FormatForSplunk_Empty(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		Splunk:  SplunkConfig{},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	result := monitor.formatForSplunk(nil)
	assert.NotEmpty(t, result) // Should still produce valid JSON
}

func TestEnterpriseMonitor_FormatForDataDog(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		DataDog: DataDogConfig{
			ServiceName: "llm-verifier",
			Environment: "production",
			Tags:        map[string]string{"version": "1.0"},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	data := map[string]interface{}{
		"health_score": 95.5,
	}

	result := monitor.formatForDataDog(data)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "llm.health_score")
}

func TestEnterpriseMonitor_FormatForNewRelic(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		NewRelic: NewRelicConfig{
			LicenseKey: "test-key",
			AppName:    "LLM Verifier",
			Labels:     map[string]string{"version": "1.0"},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	data := map[string]interface{}{
		"health_score": 88.0,
	}

	result := monitor.formatForNewRelic(data)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "llm.health_score")
}

func TestEnterpriseMonitor_FormatForNewRelic_Empty(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled:  true,
		NewRelic: NewRelicConfig{},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	data := map[string]interface{}{
		"health_score": 90.0,
	}

	result := monitor.formatForNewRelic(data)
	assert.NotEmpty(t, result)
}

func TestEnterpriseMonitor_FormatForELK(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		ELK: ELKConfig{
			ElasticsearchURL: "http://es:9200",
			IndexName:        "llm-logs",
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	data := map[string]interface{}{
		"status": "healthy",
	}

	result := monitor.formatForELK(data)
	assert.NotEmpty(t, result)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "llm-verifier", parsed["service"])
	assert.NotEmpty(t, parsed["@timestamp"])
}

func TestEnterpriseMonitor_FormatForWebhook(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	data := map[string]interface{}{
		"event": "test_event",
	}

	result := monitor.formatForWebhook(data, "default")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "timestamp")
	assert.Contains(t, result, "data")
}

func TestEnterpriseMonitor_FormatAlertForSplunk(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		Splunk: SplunkConfig{
			Host:   "splunk.test.com",
			Source: "llm-verifier",
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	alert := &monitoring.Alert{
		ID:       "alert-1",
		Name:     "High CPU",
		Message:  "CPU usage above 90%",
		Severity: monitoring.AlertSeverityCritical,
	}

	result := monitor.formatAlertForSplunk(alert)
	assert.NotEmpty(t, result)
}

func TestEnterpriseMonitor_FormatAlertForDataDog(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		DataDog: DataDogConfig{
			ServiceName: "llm-verifier",
			Environment: "prod",
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	alert := &monitoring.Alert{
		ID:       "alert-1",
		Name:     "Memory Warning",
		Message:  "Memory usage high",
		Severity: monitoring.AlertSeverityWarning,
	}

	result := monitor.formatAlertForDataDog(alert)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Memory Warning")
}

func TestEnterpriseMonitor_FormatAlertForNewRelic(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		NewRelic: NewRelicConfig{
			Labels: map[string]string{"env": "test"},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	alert := &monitoring.Alert{
		ID:       "alert-2",
		Name:     "Disk Alert",
		Message:  "Disk space low",
		Severity: monitoring.AlertSeverityCritical,
	}

	result := monitor.formatAlertForNewRelic(alert)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "LLMAlert")
}

func TestEnterpriseMonitor_FormatAlertForELK(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		ELK: ELKConfig{
			ElasticsearchURL: "http://es:9200",
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	alert := &monitoring.Alert{
		ID:      "alert-3",
		Name:    "Error Alert",
		Message: "Error occurred",
	}

	result := monitor.formatAlertForELK(alert)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "alert")
}

func TestEnterpriseMonitor_FormatDataDogTags(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		DataDog: DataDogConfig{
			ServiceName: "llm-service",
			Environment: "staging",
			Tags: map[string]string{
				"version": "2.0",
				"region":  "us-west",
			},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	tags := monitor.formatDataDogTags()
	assert.Contains(t, tags, "service:llm-service")
	assert.Contains(t, tags, "env:staging")
	assert.GreaterOrEqual(t, len(tags), 4) // base tags + custom tags
}

func TestEnterpriseMonitor_WebhookSupportsEventType(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	tests := []struct {
		name       string
		webhook    WebhookConfig
		eventType  string
		shouldPass bool
	}{
		{
			name:       "empty event types allows all",
			webhook:    WebhookConfig{EventTypes: []string{}},
			eventType:  "alert",
			shouldPass: true,
		},
		{
			name:       "matching event type",
			webhook:    WebhookConfig{EventTypes: []string{"alert", "metric"}},
			eventType:  "alert",
			shouldPass: true,
		},
		{
			name:       "non-matching event type",
			webhook:    WebhookConfig{EventTypes: []string{"metric"}},
			eventType:  "alert",
			shouldPass: false,
		},
		{
			name:       "nil event types allows all",
			webhook:    WebhookConfig{},
			eventType:  "any",
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.webhookSupportsEventType(tt.webhook, tt.eventType)
			assert.Equal(t, tt.shouldPass, result)
		})
	}
}

func TestEnterpriseMonitor_GetStatus(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled:       true,
		BatchInterval: 10 * time.Second,
		RetryAttempts: 3,
		Splunk: SplunkConfig{
			Host: "splunk.example.com",
		},
		DataDog: DataDogConfig{
			APIKey: "test-key",
		},
		CustomWebhooks: []WebhookConfig{{URL: "https://hook.example.com"}},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	status := monitor.GetStatus()
	assert.True(t, status["enabled"].(bool))
	assert.Equal(t, "10s", status["batch_interval"])
	assert.Equal(t, 3, status["retry_attempts"])

	integrations := status["integrations"].(map[string]bool)
	assert.True(t, integrations["splunk"])
	assert.True(t, integrations["datadog"])
	assert.False(t, integrations["newrelic"])
	assert.True(t, integrations["webhooks"])
}

func TestEnterpriseMonitor_GetStatus_AllDisabled(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: false,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	status := monitor.GetStatus()
	assert.False(t, status["enabled"].(bool))

	integrations := status["integrations"].(map[string]bool)
	assert.False(t, integrations["splunk"])
	assert.False(t, integrations["datadog"])
	assert.False(t, integrations["newrelic"])
	assert.False(t, integrations["elk"])
	assert.False(t, integrations["webhooks"])
}

func TestEnterpriseMonitor_Start_Disabled(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: false,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	err := monitor.Start()
	assert.NoError(t, err)
}

func TestEnterpriseMonitor_Stop(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	// Stop should not panic
	monitor.Stop()
}

// ==================== RBAC Manager Extended Tests ====================

func TestRBACManager_AuthenticateUser(t *testing.T) {
	rbac := NewRBACManager()

	// Create a user
	user := &User{
		ID:       "user-1",
		Username: "testuser",
		Email:    "test@example.com",
		Enabled:  true,
		Roles:    []RBACRole{RBACRoleViewer},
	}
	err := rbac.CreateUser(user)
	require.NoError(t, err)

	// Test authentication (note: actual password validation not implemented in stub)
	authUser, err := rbac.AuthenticateUser("testuser", "password")
	require.NoError(t, err)
	assert.Equal(t, "user-1", authUser.ID)
	assert.NotNil(t, authUser.LastLogin)
}

func TestRBACManager_AuthenticateUser_NotFound(t *testing.T) {
	rbac := NewRBACManager()

	_, err := rbac.AuthenticateUser("nonexistent", "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestRBACManager_AuthenticateUser_Disabled(t *testing.T) {
	rbac := NewRBACManager()

	// Create a disabled user
	user := &User{
		ID:       "user-2",
		Username: "disableduser",
		Email:    "disabled@example.com",
		Enabled:  false,
		Roles:    []RBACRole{RBACRoleViewer},
	}
	err := rbac.CreateUser(user)
	require.NoError(t, err)

	_, err = rbac.AuthenticateUser("disableduser", "password")
	assert.Error(t, err)
}

func TestRBACManager_HasPermission_DirectPermission(t *testing.T) {
	rbac := NewRBACManager()

	user := &User{
		ID:          "user-3",
		Username:    "testuser",
		Enabled:     true,
		Permissions: []Permission{PermissionJobSubmit},
	}
	err := rbac.CreateUser(user)
	require.NoError(t, err)

	// Test direct permission
	assert.True(t, rbac.HasPermission("user-3", PermissionJobSubmit))
	assert.False(t, rbac.HasPermission("user-3", PermissionUserManage))
}

func TestRBACManager_HasPermission_RolePermission(t *testing.T) {
	rbac := NewRBACManager()

	user := &User{
		ID:       "user-4",
		Username: "adminuser",
		Enabled:  true,
		Roles:    []RBACRole{RBACRoleAdmin},
	}
	err := rbac.CreateUser(user)
	require.NoError(t, err)

	// Admin has all permissions
	assert.True(t, rbac.HasPermission("user-4", PermissionJobSubmit))
	assert.True(t, rbac.HasPermission("user-4", PermissionUserManage))
	assert.True(t, rbac.HasPermission("user-4", PermissionSystemConfigure))
}

func TestRBACManager_HasPermission_DisabledUser(t *testing.T) {
	rbac := NewRBACManager()

	user := &User{
		ID:       "user-5",
		Username: "disabledadmin",
		Enabled:  false,
		Roles:    []RBACRole{RBACRoleAdmin},
	}
	err := rbac.CreateUser(user)
	require.NoError(t, err)

	// Disabled user has no permissions
	assert.False(t, rbac.HasPermission("user-5", PermissionJobSubmit))
}

func TestRBACManager_HasPermission_NonexistentUser(t *testing.T) {
	rbac := NewRBACManager()

	assert.False(t, rbac.HasPermission("nonexistent", PermissionJobSubmit))
}

// ==================== DecodePEM Tests ====================

func TestDecodePEM_ValidPEM(t *testing.T) {
	pemData := `-----BEGIN CERTIFICATE-----
TWFuIGlzIGRpc3Rpbmd1aXNoZWQ=
-----END CERTIFICATE-----`

	result, err := decodePEM([]byte(pemData))
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestDecodePEM_NoHeader(t *testing.T) {
	pemData := `TWFuIGlzIGRpc3Rpbmd1aXNoZWQ=
-----END CERTIFICATE-----`

	_, err := decodePEM([]byte(pemData))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no PEM header found")
}

func TestDecodePEM_NoFooter(t *testing.T) {
	pemData := `-----BEGIN CERTIFICATE-----
TWFuIGlzIGRpc3Rpbmd1aXNoZWQ=`

	_, err := decodePEM([]byte(pemData))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no PEM footer found")
}

func TestDecodePEM_InvalidFormat(t *testing.T) {
	pemData := `-----BEGIN CERTIFICATE----------END CERTIFICATE-----`

	_, err := decodePEM([]byte(pemData))
	assert.Error(t, err)
}

// ==================== EnterpriseAPI Permission Tests ====================

func TestEnterpriseAPI_GetRequiredPermission(t *testing.T) {
	manager := &EnterpriseManager{
		RBAC: NewRBACManager(),
	}
	api := NewEnterpriseAPI(manager)

	tests := []struct {
		path       string
		method     string
		expected   Permission
	}{
		{"/api/enterprise/users", "GET", PermissionJobView},
		{"/api/enterprise/users", "POST", PermissionUserManage},
		{"/api/enterprise/users/123", "GET", PermissionJobView},
		{"/api/enterprise/users/123", "DELETE", PermissionUserManage},
		{"/api/enterprise/roles", "GET", PermissionSystemConfigure},
		{"/api/enterprise/roles/admin", "PUT", PermissionSystemConfigure},
		{"/api/enterprise/tenants", "GET", PermissionUserManage},
		{"/api/enterprise/audit", "GET", PermissionLogsView},
		{"/api/enterprise/metrics", "GET", PermissionMetricsView},
		{"/api/unknown", "GET", Permission("")},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.method, func(t *testing.T) {
			result := api.getRequiredPermission(tt.path, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== LDAP Validation Tests ====================

func TestValidateLDAPConfig_Disabled(t *testing.T) {
	config := LDAPConfig{
		Enabled: false,
	}

	err := ValidateLDAPConfig(config)
	assert.NoError(t, err)
}

func TestValidateLDAPConfig_MissingURL(t *testing.T) {
	config := LDAPConfig{
		Enabled: true,
		URL:     "",
		BaseDN:  "dc=example,dc=com",
	}

	err := ValidateLDAPConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestValidateLDAPConfig_MissingBaseDN(t *testing.T) {
	config := LDAPConfig{
		Enabled: true,
		URL:     "ldap://localhost:389",
		BaseDN:  "",
	}

	err := ValidateLDAPConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BaseDN is required")
}

func TestValidateLDAPConfig_MissingUserFilter(t *testing.T) {
	config := LDAPConfig{
		Enabled:    true,
		URL:        "ldap://localhost:389",
		BaseDN:     "dc=example,dc=com",
		UserFilter: "",
	}

	err := ValidateLDAPConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UserFilter is required")
}

func TestValidateLDAPConfig_Valid(t *testing.T) {
	config := LDAPConfig{
		Enabled:    true,
		URL:        "ldap://localhost:389",
		BaseDN:     "dc=example,dc=com",
		UserFilter: "(uid=%s)",
	}

	err := ValidateLDAPConfig(config)
	assert.NoError(t, err)
}

// ==================== Multi-Tenant Manager Extended Tests ====================

func TestMultiTenantManager_AddUserToTenant_TenantNotFound(t *testing.T) {
	mtm := NewMultiTenantManager()

	err := mtm.AddUserToTenant("nonexistent-tenant", "user-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant not found")
}

func TestMultiTenantManager_AddUserToTenant_UserAlreadyExists(t *testing.T) {
	mtm := NewMultiTenantManager()

	tenant := &Tenant{
		ID:      "tenant-1",
		Name:    "Test Tenant",
		Enabled: true,
	}
	err := mtm.CreateTenant(tenant)
	require.NoError(t, err)

	// Add user first time
	err = mtm.AddUserToTenant("tenant-1", "user-1")
	require.NoError(t, err)

	// Try to add same user again
	err = mtm.AddUserToTenant("tenant-1", "user-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user already in tenant")
}

func TestMultiTenantManager_GetUserTenants(t *testing.T) {
	mtm := NewMultiTenantManager()

	// Create tenants
	tenant1 := &Tenant{ID: "tenant-1", Name: "Tenant 1", Enabled: true}
	tenant2 := &Tenant{ID: "tenant-2", Name: "Tenant 2", Enabled: true}
	tenant3 := &Tenant{ID: "tenant-3", Name: "Tenant 3", Enabled: true}

	require.NoError(t, mtm.CreateTenant(tenant1))
	require.NoError(t, mtm.CreateTenant(tenant2))
	require.NoError(t, mtm.CreateTenant(tenant3))

	// Add user to some tenants
	require.NoError(t, mtm.AddUserToTenant("tenant-1", "user-1"))
	require.NoError(t, mtm.AddUserToTenant("tenant-3", "user-1"))

	tenants := mtm.GetUserTenants("user-1")
	assert.Len(t, tenants, 2)
}

func TestMultiTenantManager_GetUserTenants_NoTenants(t *testing.T) {
	mtm := NewMultiTenantManager()

	tenants := mtm.GetUserTenants("nonexistent-user")
	assert.Empty(t, tenants)
}

// ==================== Enterprise Manager Tests ====================

func TestEnterpriseManager_StartStop(t *testing.T) {
	config := EnterpriseConfig{
		RBAC: RBACConfig{
			Enabled: true,
		},
	}

	manager := NewEnterpriseManager(config)
	require.NotNil(t, manager)
	require.NotNil(t, manager.RBAC)
	require.NotNil(t, manager.MultiTenant)
	require.NotNil(t, manager.API)

	ctx := context.Background()

	// Start (will fail due to no server address configured, but tests the code path)
	err := manager.Start(ctx)
	// May return error due to missing server config
	if err != nil {
		assert.Contains(t, err.Error(), "failed to start")
	}

	// Stop should not panic
	err = manager.Stop(ctx)
	assert.NoError(t, err)
}

// ==================== SAML Authenticator Tests ====================

func TestSAMLAuthenticator_IsConfigured_Extended(t *testing.T) {
	tests := []struct {
		name       string
		config     SAMLConfig
		configured bool
	}{
		{
			name:       "not configured",
			config:     SAMLConfig{},
			configured: false,
		},
		{
			name: "configured",
			config: SAMLConfig{
				IdentityProviderURL: "https://idp.example.com/saml",
			},
			configured: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewSAMLAuthenticator(tt.config)
			assert.Equal(t, tt.configured, auth.IsConfigured())
		})
	}
}

func TestSAMLAuthenticator_GetLoginURL_Extended(t *testing.T) {
	tests := []struct {
		name     string
		config   SAMLConfig
		expected string
	}{
		{
			name: "uses SSO URL when available",
			config: SAMLConfig{
				IdentityProviderURL: "https://idp.example.com",
				SSOURL:              "https://sso.example.com/login",
			},
			expected: "https://sso.example.com/login",
		},
		{
			name: "falls back to IdP URL",
			config: SAMLConfig{
				IdentityProviderURL: "https://idp.example.com",
			},
			expected: "https://idp.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewSAMLAuthenticator(tt.config)
			url := auth.GetLoginURL("relay-state")
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestSAMLAuthenticator_ProcessSAMLResponse_NotConfigured(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{})

	_, err := auth.ProcessSAMLResponse("base64encodedresponse")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not properly configured")
}

func TestSAMLAuthenticator_ProcessSAMLResponse_InvalidBase64(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	_, err := auth.ProcessSAMLResponse("!!!invalid-base64!!!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestSAMLAuthenticator_DefaultAttributeMapping(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	assert.NotNil(t, auth.config.AttributeMapping)
	assert.Contains(t, auth.config.AttributeMapping, "email")
	assert.Contains(t, auth.config.AttributeMapping, "username")
}

func TestSAMLAuthenticator_MaxClockSkew(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	// Default clock skew is 5 minutes
	assert.Equal(t, 5*time.Minute, auth.config.MaxClockSkew)

	// Custom clock skew
	auth2 := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		MaxClockSkew:        10 * time.Minute,
	})
	assert.Equal(t, 10*time.Minute, auth2.config.MaxClockSkew)
}

// ==================== MapGroupsToRoles Tests ====================

func TestMapGroupsToRoles_Extended(t *testing.T) {
	tests := []struct {
		name     string
		groups   []string
		expected []RBACRole
	}{
		{
			name:     "admin group",
			groups:   []string{"admin"},
			expected: []RBACRole{RBACRoleAdmin},
		},
		{
			name:     "admins group",
			groups:   []string{"admins"},
			expected: []RBACRole{RBACRoleAdmin},
		},
		{
			name:     "administrator group",
			groups:   []string{"administrator"},
			expected: []RBACRole{RBACRoleAdmin},
		},
		{
			name:     "operator group",
			groups:   []string{"operator"},
			expected: []RBACRole{RBACRoleOperator},
		},
		{
			name:     "analyst group",
			groups:   []string{"analyst"},
			expected: []RBACRole{RBACRoleAnalyst},
		},
		{
			name:     "viewer group",
			groups:   []string{"viewer"},
			expected: []RBACRole{RBACRoleViewer},
		},
		{
			name:     "readonly group",
			groups:   []string{"readonly"},
			expected: []RBACRole{RBACRoleReadOnly},
		},
		{
			name:     "multiple groups",
			groups:   []string{"Admin", "Operator"},
			expected: []RBACRole{RBACRoleAdmin, RBACRoleOperator},
		},
		{
			name:     "unknown groups",
			groups:   []string{"unknown", "custom"},
			expected: nil,
		},
		{
			name:     "empty groups",
			groups:   []string{},
			expected: nil,
		},
		{
			name:     "groups with spaces",
			groups:   []string{"  admin  ", " viewer "},
			expected: []RBACRole{RBACRoleAdmin, RBACRoleViewer},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapGroupsToRoles(tt.groups)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== SanitizeUserID Tests ====================

func TestSanitizeUserID_Extended(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "user_at_example.com"},
		{"domain\\user", "domain_user"},
		{"domain/user", "domain_user"},
		{"user name", "user_name"},
		{"normaluser", "normaluser"},
		{"user@domain/path", "user_at_domain_path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeUserID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== HTTP Handler Tests ====================

func TestEnterpriseAPI_SendHTTPRequest(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := EnterpriseMonitorConfig{
		Enabled:       true,
		RetryAttempts: 1,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	err := monitor.sendHTTPRequest("test", server.URL, `{"test": true}`, "")
	assert.NoError(t, err)
}

func TestEnterpriseAPI_SendHTTPRequest_WithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		assert.Equal(t, "Splunk test-token", auth)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := EnterpriseMonitorConfig{
		Enabled: true,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	err := monitor.sendHTTPRequest("test", server.URL, `{"test": true}`, "test-token")
	assert.NoError(t, err)
}

func TestEnterpriseAPI_SendHTTPRequest_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := EnterpriseMonitorConfig{
		Enabled: true,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	err := monitor.sendHTTPRequest("test", server.URL, `{"test": true}`, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestEnterpriseAPI_SendWebhookRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := EnterpriseMonitorConfig{
		Enabled: true,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	webhook := WebhookConfig{
		URL:     server.URL,
		Method:  "POST",
		Headers: map[string]string{"Authorization": "Bearer token123"},
	}

	err := monitor.sendWebhookRequest(webhook, `{"event": "test"}`)
	assert.NoError(t, err)
}

func TestEnterpriseAPI_SendWebhookRequest_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	config := EnterpriseMonitorConfig{
		Enabled: true,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)

	webhook := WebhookConfig{
		URL:    server.URL,
		Method: "POST",
	}

	err := monitor.sendWebhookRequest(webhook, `{"event": "test"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

// ==================== SAML Response Processing Tests ====================

func TestSAMLAuthenticator_ValidateConditions_Expired(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		MaxClockSkew:        5 * time.Minute,
	})

	expiredTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	conditions := &SAMLConditions{
		NotOnOrAfter: expiredTime,
	}

	err := auth.validateConditions(conditions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestSAMLAuthenticator_ValidateConditions_NotYetValid(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		MaxClockSkew:        5 * time.Minute,
	})

	futureTime := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	conditions := &SAMLConditions{
		NotBefore: futureTime,
	}

	err := auth.validateConditions(conditions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet valid")
}

func TestSAMLAuthenticator_ValidateConditions_Valid(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		MaxClockSkew:        5 * time.Minute,
	})

	notBefore := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	notOnOrAfter := time.Now().Add(10 * time.Minute).Format(time.RFC3339)
	conditions := &SAMLConditions{
		NotBefore:    notBefore,
		NotOnOrAfter: notOnOrAfter,
	}

	err := auth.validateConditions(conditions)
	assert.NoError(t, err)
}

func TestSAMLAuthenticator_ValidateConditions_InvalidTimeFormat(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	conditions := &SAMLConditions{
		NotBefore: "invalid-time-format",
	}

	err := auth.validateConditions(conditions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid NotBefore time format")
}

func TestSAMLAuthenticator_ValidateResponse_InvalidVersion(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	response := &SAMLResponse{
		Version: "1.0",
	}

	err := auth.validateResponse(response)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported SAML version")
}

func TestSAMLAuthenticator_ValidateResponse_FailedStatus(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	response := &SAMLResponse{
		Version: "2.0",
		Status: SAMLStatus{
			StatusCode: SAMLStatusCode{Value: "urn:oasis:names:tc:SAML:2.0:status:Requester"},
		},
	}

	err := auth.validateResponse(response)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestSAMLAuthenticator_ValidateResponse_NoAssertion(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	response := &SAMLResponse{
		Version: "2.0",
		Issuer:  "https://idp.example.com",
		Status: SAMLStatus{
			StatusCode: SAMLStatusCode{Value: "urn:oasis:names:tc:SAML:2.0:status:Success"},
		},
		Assertion: nil,
	}

	err := auth.validateResponse(response)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no assertion")
}

func TestSAMLAuthenticator_ExtractUserFromAssertion_NoNameID(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	assertion := &SAMLAssertion{
		Subject: SAMLSubject{
			NameID: SAMLNameID{Value: ""},
		},
	}

	_, err := auth.extractUserFromAssertion(assertion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no NameID")
}

func TestSAMLAuthenticator_ExtractUserFromAssertion_Success(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	assertion := &SAMLAssertion{
		Subject: SAMLSubject{
			NameID: SAMLNameID{Value: "user@example.com"},
		},
		AttributeStatement: SAMLAttributeStmt{
			Attributes: []SAMLAttribute{
				{
					Name:   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
					Values: []string{"user@example.com"},
				},
				{
					Name:   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
					Values: []string{"Test"},
				},
				{
					Name:   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
					Values: []string{"User"},
				},
				{
					Name:   "http://schemas.xmlsoap.org/claims/Group",
					Values: []string{"admin", "operator"},
				},
			},
		},
	}

	user, err := auth.extractUserFromAssertion(assertion)
	require.NoError(t, err)

	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "user", user.Username)
	assert.Equal(t, "Test", user.FirstName)
	assert.Equal(t, "User", user.LastName)
	assert.Contains(t, user.Roles, RBACRoleAdmin)
	assert.Contains(t, user.Roles, RBACRoleOperator)
}

func TestSAMLAuthenticator_ExtractUserFromAssertion_NilAssertion(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	_, err := auth.extractUserFromAssertion(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "assertion is nil")
}

// ==================== Process Full SAML Response Tests ====================

func TestSAMLAuthenticator_ProcessSAMLResponse_ValidResponse(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		MaxClockSkew:        5 * time.Minute,
	})

	// Create a valid SAML response
	notBefore := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)
	notOnOrAfter := time.Now().Add(5 * time.Minute).Format(time.RFC3339)

	samlXML := `<?xml version="1.0"?>
<Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol" ID="resp123" Version="2.0" IssueInstant="2024-01-01T00:00:00Z">
  <Issuer xmlns="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example.com</Issuer>
  <Status>
    <StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </Status>
  <Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="assert123" Version="2.0" IssueInstant="2024-01-01T00:00:00Z">
    <Issuer>https://idp.example.com</Issuer>
    <Subject>
      <NameID>testuser@example.com</NameID>
    </Subject>
    <Conditions NotBefore="` + notBefore + `" NotOnOrAfter="` + notOnOrAfter + `"/>
    <AttributeStatement>
      <Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress">
        <AttributeValue>testuser@example.com</AttributeValue>
      </Attribute>
    </AttributeStatement>
  </Assertion>
</Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(samlXML))

	user, err := auth.ProcessSAMLResponse(encoded)
	require.NoError(t, err)
	assert.Equal(t, "testuser@example.com", user.Email)
}

// ==================== Audience Validation Tests ====================

func TestSAMLAuthenticator_ValidateConditions_AudienceMatch(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		AllowedAudiences:    []string{"https://app.example.com", "https://other.example.com"},
	})

	conditions := &SAMLConditions{
		AudienceRestrictions: []SAMLAudienceRestr{
			{Audiences: []string{"https://app.example.com"}},
		},
	}

	err := auth.validateConditions(conditions)
	assert.NoError(t, err)
}

func TestSAMLAuthenticator_ValidateConditions_AudienceMismatch(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		AllowedAudiences:    []string{"https://app.example.com"},
	})

	conditions := &SAMLConditions{
		AudienceRestrictions: []SAMLAudienceRestr{
			{Audiences: []string{"https://wrong.example.com"}},
		},
	}

	err := auth.validateConditions(conditions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "audience does not match")
}

// ==================== Config Struct Tests ====================

func TestPasswordPolicy_Struct(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:        12,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumbers:   true,
		RequireSymbols:   true,
		MaxAge:           90 * 24 * time.Hour,
	}

	assert.Equal(t, 12, policy.MinLength)
	assert.True(t, policy.RequireUppercase)
	assert.Equal(t, 90*24*time.Hour, policy.MaxAge)
}

func TestSecurityConfig_Struct(t *testing.T) {
	config := SecurityConfig{
		HTTPSEnabled: true,
		CORSOrigins:  []string{"https://app.example.com"},
		RateLimiting: RateLimitConfig{
			Enabled:  true,
			Requests: 100,
			Window:   time.Minute,
		},
	}

	assert.True(t, config.HTTPSEnabled)
	assert.Len(t, config.CORSOrigins, 1)
	assert.True(t, config.RateLimiting.Enabled)
}

func TestTLSConfig_Struct(t *testing.T) {
	config := TLSConfig{
		Enabled:            true,
		CertFile:           "/path/to/cert.pem",
		KeyFile:            "/path/to/key.pem",
		CAFile:             "/path/to/ca.pem",
		ServerName:         "server.example.com",
		InsecureSkipVerify: false,
	}

	assert.True(t, config.Enabled)
	assert.Equal(t, "server.example.com", config.ServerName)
	assert.False(t, config.InsecureSkipVerify)
}

// ==================== LDAP Authenticator Tests ====================

func TestNewLDAPAuthenticator(t *testing.T) {
	config := LDAPConfig{
		Enabled:    true,
		URL:        "ldap://localhost:389",
		BaseDN:     "dc=example,dc=com",
		UserFilter: "(uid=%s)",
		Timeout:    30 * time.Second,
	}

	auth := NewLDAPAuthenticator(config)
	assert.NotNil(t, auth)
	assert.True(t, auth.config.Enabled)
	assert.Equal(t, "ldap://localhost:389", auth.config.URL)
}

func TestLDAPAuthenticator_Authenticate_Disabled(t *testing.T) {
	config := LDAPConfig{
		Enabled: false,
	}

	auth := NewLDAPAuthenticator(config)
	_, err := auth.Authenticate("user", "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

// Note: Actual LDAP dial tests require mocking the ldap package

// ==================== Alternative Time Format Tests ====================

func TestSAMLAuthenticator_ValidateConditions_AlternativeTimeFormat(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		MaxClockSkew:        5 * time.Minute,
	})

	// Use alternative time format with UTC time
	notBefore := time.Now().UTC().Add(-10 * time.Minute).Format("2006-01-02T15:04:05Z")
	notOnOrAfter := time.Now().UTC().Add(10 * time.Minute).Format("2006-01-02T15:04:05Z")

	conditions := &SAMLConditions{
		NotBefore:    notBefore,
		NotOnOrAfter: notOnOrAfter,
	}

	err := auth.validateConditions(conditions)
	assert.NoError(t, err)
}

func TestSAMLAuthenticator_ValidateConditions_InvalidNotOnOrAfterFormat(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	})

	conditions := &SAMLConditions{
		NotOnOrAfter: "invalid-format",
	}

	err := auth.validateConditions(conditions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid NotOnOrAfter time format")
}

// ==================== FriendlyName Attribute Tests ====================

func TestSAMLAuthenticator_ExtractUserFromAssertion_FriendlyName(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		AttributeMapping: map[string]string{
			"email": "Email",
		},
	})

	assertion := &SAMLAssertion{
		Subject: SAMLSubject{
			NameID: SAMLNameID{Value: "user123"},
		},
		AttributeStatement: SAMLAttributeStmt{
			Attributes: []SAMLAttribute{
				{
					Name:         "http://schemas.example.com/email",
					FriendlyName: "Email",
					Values:       []string{"friendly@example.com"},
				},
			},
		},
	}

	user, err := auth.extractUserFromAssertion(assertion)
	require.NoError(t, err)
	assert.Equal(t, "friendly@example.com", user.Email)
}

// ==================== GetClientIP Tests ====================

func TestEnterpriseAPI_GetClientIP(t *testing.T) {
	manager := &EnterpriseManager{
		RBAC: NewRBACManager(),
	}
	api := NewEnterpriseAPI(manager)

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For header",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Real-IP header",
			headers:    map[string]string{"X-Real-IP": "192.168.1.2"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.2",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "10.0.0.5:12345",
			expected:   "10.0.0.5:12345",
		},
		{
			name:       "X-Forwarded-For takes precedence",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1", "X-Real-IP": "192.168.1.2"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remoteAddr

			result := api.getClientIP(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== ResponseWriter Tests ====================

func TestResponseWriter_WriteHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     0,
	}

	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.statusCode)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

// ==================== URL-Safe Base64 Tests ====================

func TestSAMLAuthenticator_ProcessSAMLResponse_URLSafeBase64(t *testing.T) {
	auth := NewSAMLAuthenticator(SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		MaxClockSkew:        5 * time.Minute,
	})

	notBefore := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)
	notOnOrAfter := time.Now().Add(5 * time.Minute).Format(time.RFC3339)

	samlXML := `<?xml version="1.0"?>
<Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol" ID="resp123" Version="2.0">
  <Issuer xmlns="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example.com</Issuer>
  <Status>
    <StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </Status>
  <Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="assert123" Version="2.0">
    <Subject>
      <NameID>urlsafe@example.com</NameID>
    </Subject>
    <Conditions NotBefore="` + notBefore + `" NotOnOrAfter="` + notOnOrAfter + `"/>
    <AttributeStatement/>
  </Assertion>
</Response>`

	// Use URL-safe base64 encoding
	encoded := base64.URLEncoding.EncodeToString([]byte(samlXML))
	// Make sure standard decoding fails (to test URL-safe fallback)
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")

	user, err := auth.ProcessSAMLResponse(encoded)
	require.NoError(t, err)
	assert.Contains(t, user.ID, "saml_")
}
