package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== ComplianceManager Tests ====================

func TestNewComplianceManager(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	cm, err := NewComplianceManager(logPath)
	require.NoError(t, err)
	assert.NotNil(t, cm)
	defer cm.Close()
}

func TestNewComplianceManager_InvalidPath(t *testing.T) {
	// Try to open a file in a non-existent directory
	_, err := NewComplianceManager("/nonexistent/path/audit.log")
	assert.Error(t, err)
}

func TestComplianceManager_LogAuditEvent(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	cm, err := NewComplianceManager(logPath)
	require.NoError(t, err)
	defer cm.Close()

	event := &AuditEvent{
		ID:        "event-123",
		Timestamp: time.Now(),
		EventType: "api_request",
		UserID:    "user123",
		Resource:  "/api/models",
		Action:    "read",
		Status:    "success",
		IPAddress: "192.168.1.1",
	}

	err = cm.LogAuditEvent(event)
	assert.NoError(t, err)

	// Give time for async write
	time.Sleep(100 * time.Millisecond)

	// Verify file was written
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "event-123")
}

func TestComplianceManager_Close(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	cm, err := NewComplianceManager(logPath)
	require.NoError(t, err)

	err = cm.Close()
	assert.NoError(t, err)
}

func TestComplianceManager_QueryAuditLogs(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	cm, err := NewComplianceManager(logPath)
	require.NoError(t, err)
	defer cm.Close()

	// Query returns empty slice in demo implementation
	logs, err := cm.QueryAuditLogs(nil, 10)
	require.NoError(t, err)
	assert.Empty(t, logs)
}

// ==================== DataRetentionManager Tests ====================

func TestNewDataRetentionManager(t *testing.T) {
	drm := NewDataRetentionManager()
	assert.NotNil(t, drm)
	assert.NotEmpty(t, drm.policies)
}

func TestDataRetentionManager_SetRetentionPolicy(t *testing.T) {
	drm := NewDataRetentionManager()

	policy := RetentionPolicy{
		Resource:   "custom_resource",
		Retention:  30 * 24 * time.Hour, // 30 days
		Action:     "delete",
		Compliance: "Custom",
	}

	drm.SetRetentionPolicy("custom_resource", policy)

	retrieved, exists := drm.GetRetentionPolicy("custom_resource")
	assert.True(t, exists)
	assert.Equal(t, "custom_resource", retrieved.Resource)
	assert.Equal(t, "delete", retrieved.Action)
}

func TestDataRetentionManager_GetRetentionPolicy(t *testing.T) {
	drm := NewDataRetentionManager()

	// Get existing policy (default)
	policy, exists := drm.GetRetentionPolicy("audit_logs")
	assert.True(t, exists)
	assert.Equal(t, "audit_logs", policy.Resource)
	assert.Equal(t, "SOC2", policy.Compliance)

	// Get non-existent policy
	_, exists = drm.GetRetentionPolicy("nonexistent")
	assert.False(t, exists)
}

func TestDataRetentionManager_CheckRetention(t *testing.T) {
	drm := NewDataRetentionManager()

	// Check recent data (should be retained)
	retain, _ := drm.CheckRetention("audit_logs", time.Now())
	assert.True(t, retain)

	// Check old data (should not be retained)
	oldDate := time.Now().Add(-10 * 365 * 24 * time.Hour) // 10 years ago
	retain, action := drm.CheckRetention("audit_logs", oldDate)
	assert.False(t, retain)
	assert.NotEmpty(t, action)

	// Check non-existent resource (default keep)
	retain, _ = drm.CheckRetention("nonexistent", time.Now())
	assert.True(t, retain)
}

// ==================== GDPRManager Tests ====================

func TestNewGDPRManager(t *testing.T) {
	gm := NewGDPRManager()
	assert.NotNil(t, gm)
}

func TestGDPRManager_HandleDataSubjectRequest_Access(t *testing.T) {
	gm := NewGDPRManager()

	err := gm.HandleDataSubjectRequest("access", "user123")
	assert.NoError(t, err)
}

func TestGDPRManager_HandleDataSubjectRequest_Erasure(t *testing.T) {
	gm := NewGDPRManager()

	err := gm.HandleDataSubjectRequest("erasure", "user123")
	assert.NoError(t, err)
}

func TestGDPRManager_HandleDataSubjectRequest_Rectification(t *testing.T) {
	gm := NewGDPRManager()

	err := gm.HandleDataSubjectRequest("rectification", "user123")
	assert.NoError(t, err)
}

func TestGDPRManager_HandleDataSubjectRequest_Portability(t *testing.T) {
	gm := NewGDPRManager()

	err := gm.HandleDataSubjectRequest("portability", "user123")
	assert.NoError(t, err)
}

func TestGDPRManager_HandleDataSubjectRequest_Restriction(t *testing.T) {
	gm := NewGDPRManager()

	err := gm.HandleDataSubjectRequest("restriction", "user123")
	assert.NoError(t, err)
}

func TestGDPRManager_HandleDataSubjectRequest_Objection(t *testing.T) {
	gm := NewGDPRManager()

	err := gm.HandleDataSubjectRequest("objection", "user123")
	assert.NoError(t, err)
}

func TestGDPRManager_HandleDataSubjectRequest_Unknown(t *testing.T) {
	gm := NewGDPRManager()

	err := gm.HandleDataSubjectRequest("unknown_type", "user123")
	assert.Error(t, err)
}

// ==================== ConsentManager Tests ====================

func TestNewConsentManager(t *testing.T) {
	cm := NewConsentManager()
	assert.NotNil(t, cm)
}

func TestConsentManager_GrantConsent(t *testing.T) {
	cm := NewConsentManager()

	// Grant with no expiry
	cm.GrantConsent("user123", "marketing", nil)

	// Verify grant
	hasConsent := cm.CheckConsent("user123", "marketing")
	assert.True(t, hasConsent)
}

func TestConsentManager_WithdrawConsent(t *testing.T) {
	cm := NewConsentManager()

	// Grant first
	cm.GrantConsent("user123", "marketing", nil)

	// Verify granted
	hasConsent := cm.CheckConsent("user123", "marketing")
	assert.True(t, hasConsent)

	// Withdraw
	cm.WithdrawConsent("user123", "marketing")

	// Should no longer have consent
	hasConsent = cm.CheckConsent("user123", "marketing")
	assert.False(t, hasConsent)
}

func TestConsentManager_CheckConsent(t *testing.T) {
	cm := NewConsentManager()

	// Grant consent
	cm.GrantConsent("user123", "analytics", nil)
	cm.GrantConsent("user123", "marketing", nil)

	// Check granted
	hasAnalytics := cm.CheckConsent("user123", "analytics")
	assert.True(t, hasAnalytics)

	hasMarketing := cm.CheckConsent("user123", "marketing")
	assert.True(t, hasMarketing)

	// Check not granted
	hasOther := cm.CheckConsent("user123", "other")
	assert.False(t, hasOther)

	// Check unknown user
	hasUnknown := cm.CheckConsent("unknown", "analytics")
	assert.False(t, hasUnknown)
}

func TestConsentManager_CheckConsent_ExpiredConsent(t *testing.T) {
	cm := NewConsentManager()

	// Grant with past expiry
	pastExpiry := time.Now().Add(-24 * time.Hour)
	cm.GrantConsent("user123", "expired", &pastExpiry)

	// Should not have consent (expired)
	hasConsent := cm.CheckConsent("user123", "expired")
	assert.False(t, hasConsent)
}

// ==================== DataManager Tests ====================

func TestNewDataManager(t *testing.T) {
	dm := NewDataManager()
	assert.NotNil(t, dm)
}

func TestDataManager_ExportUserData(t *testing.T) {
	dm := NewDataManager()

	data, err := dm.ExportUserData("user123")
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.NotEmpty(t, data)
	// Data contains JSON with user_id
	assert.Contains(t, string(data), "user123")
}

func TestDataManager_DeleteUserData(t *testing.T) {
	dm := NewDataManager()

	err := dm.DeleteUserData("user123")
	assert.NoError(t, err)
}

// ==================== Struct Tests ====================

func TestAuditEvent_Struct(t *testing.T) {
	event := AuditEvent{
		ID:          "event-1",
		Timestamp:   time.Now(),
		EventType:   "auth",
		UserID:      "user1",
		ClientID:    "client1",
		Resource:    "/api/test",
		Action:      "create",
		Status:      "success",
		IPAddress:   "10.0.0.1",
		UserAgent:   "TestAgent/1.0",
		Details:     map[string]interface{}{"key": "value"},
		DataSubject: "subject1",
	}

	assert.Equal(t, "event-1", event.ID)
	assert.Equal(t, "auth", event.EventType)
	assert.Equal(t, "create", event.Action)
}

func TestRetentionPolicy_Struct(t *testing.T) {
	policy := RetentionPolicy{
		Resource:   "logs",
		Retention:  365 * 24 * time.Hour,
		Action:     "archive",
		Compliance: "SOC2",
	}

	assert.Equal(t, "logs", policy.Resource)
	assert.Equal(t, "archive", policy.Action)
	assert.Equal(t, "SOC2", policy.Compliance)
}
