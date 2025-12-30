// Package api provides security middleware tests
package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== AuditLogger Tests ====================

func TestNewAuditLogger(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")

	assert.NotNil(t, al)
	assert.Equal(t, "/tmp/audit.log", al.logFile)
	assert.NotNil(t, al.events)
	assert.NotNil(t, al.stopCh)

	// Clean up
	al.Stop()
}

func TestAuditLogger_LogEvent(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	event := &AuditEvent{
		ID:        "test-event-1",
		Timestamp: time.Now(),
		EventType: "test",
		Severity:  "info",
	}

	// Should not panic
	al.LogEvent(event)

	// Give time for async processing
	time.Sleep(10 * time.Millisecond)
}

func TestAuditLogger_LogHTTPRequest(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	req := httptest.NewRequest("GET", "/api/models", nil)
	req.Header.Set("User-Agent", "test-agent")

	// Should not panic
	al.LogHTTPRequest(req, 200, time.Millisecond*100, 1024)

	time.Sleep(10 * time.Millisecond)
}

func TestAuditLogger_LogSecurityEvent(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	details := map[string]interface{}{
		"description": "Test security event",
		"ip_address":  "192.168.1.1",
	}

	// Should not panic
	al.LogSecurityEvent("security_alert", "warning", "Test alert", details)

	time.Sleep(10 * time.Millisecond)
}

func TestAuditLogger_LogAuthentication(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	// Test successful authentication
	al.LogAuthentication("192.168.1.1", "Mozilla/5.0", "testuser", true, nil)

	// Test failed authentication
	al.LogAuthentication("192.168.1.2", "Mozilla/5.0", "baduser", false, map[string]interface{}{
		"reason": "invalid password",
	})

	time.Sleep(10 * time.Millisecond)
}

func TestAuditLogger_LogAuthorization(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	// Test granted authorization
	al.LogAuthorization("192.168.1.1", "Mozilla/5.0", "/api/admin", "read", true, nil)

	// Test denied authorization
	al.LogAuthorization("192.168.1.2", "Mozilla/5.0", "/api/admin", "write", false, map[string]interface{}{
		"reason": "insufficient permissions",
	})

	time.Sleep(10 * time.Millisecond)
}

func TestAuditLogger_LogDataAccess(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	al.LogDataAccess("192.168.1.1", "Mozilla/5.0", "SELECT", "users", 100, nil)

	time.Sleep(10 * time.Millisecond)
}

func TestAuditLogger_LogError(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	err := errors.New("database connection failed")
	al.LogError("192.168.1.1", "Mozilla/5.0", "db_query", err, nil)

	time.Sleep(10 * time.Millisecond)
}

func TestAuditLogger_GetSeverityForStatus(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	tests := []struct {
		statusCode int
		expected   string
	}{
		{200, "info"},
		{201, "info"},
		{301, "info"},
		{400, "warning"},
		{401, "warning"},
		{403, "warning"},
		{404, "warning"},
		{500, "error"},
		{502, "error"},
		{503, "error"},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			severity := al.getSeverityForStatus(tt.statusCode)
			assert.Equal(t, tt.expected, severity)
		})
	}
}

func TestAuditLogger_IsSecurityRelevant(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	tests := []struct {
		method   string
		path     string
		expected bool
	}{
		{"GET", "/api/auth/login", true},
		{"GET", "/api/admin/users", true},
		{"GET", "/api/config", true},
		{"GET", "/api/export/data", true},
		{"GET", "/api/models", false},
		{"POST", "/api/models", true}, // Non-GET is security relevant
		{"DELETE", "/api/models", true},
		{"PUT", "/api/models", true},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			result := al.isSecurityRelevant(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditLogger_Stop(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")

	// Should not panic
	al.Stop()
}

func TestGenerateAuditID(t *testing.T) {
	id1 := generateAuditID()
	id2 := generateAuditID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.Contains(t, id1, "audit_")
	// IDs should be unique (or at least different with enough time)
}

func TestAuditMiddleware(t *testing.T) {
	al := NewAuditLogger("/tmp/audit.log")
	defer al.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := AuditMiddleware(al)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/api/models", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestAuditResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	arw := &auditResponseWriter{
		ResponseWriter: rr,
		statusCode:     200,
	}

	// Test WriteHeader
	arw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, arw.statusCode)

	// Test Write
	n, err := arw.Write([]byte("test data"))
	require.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, 9, arw.size)
}

// ==================== ComplianceChecker Tests ====================

func TestNewComplianceChecker(t *testing.T) {
	cc := NewComplianceChecker(true, 90)

	assert.NotNil(t, cc)
	assert.True(t, cc.gdprEnabled)
	assert.Equal(t, 90, cc.dataRetentionDays)
	assert.NotNil(t, cc.piiPatterns)
	assert.NotEmpty(t, cc.piiPatterns)
	assert.NotNil(t, cc.sensitiveFields)
	assert.NotEmpty(t, cc.sensitiveFields)
}

func TestComplianceChecker_CheckDataCompliance_Clean(t *testing.T) {
	cc := NewComplianceChecker(false, 90)

	data := map[string]interface{}{
		"name":        "Test Model",
		"description": "A test model for verification",
	}

	result := cc.CheckDataCompliance(data)

	assert.True(t, result.IsCompliant)
	assert.Empty(t, result.Violations)
	assert.False(t, result.PIIDetected)
}

func TestComplianceChecker_CheckDataCompliance_PIIDetected(t *testing.T) {
	cc := NewComplianceChecker(false, 90)

	data := map[string]interface{}{
		"email": "user@example.com",
		"phone": "+1-555-123-4567",
	}

	result := cc.CheckDataCompliance(data)

	assert.False(t, result.IsCompliant)
	assert.True(t, result.PIIDetected)
	assert.NotEmpty(t, result.Violations)
}

func TestComplianceChecker_CheckDataCompliance_SensitiveFields(t *testing.T) {
	cc := NewComplianceChecker(false, 90)

	data := map[string]interface{}{
		"password": "secret123",
		"api_key":  "sk-1234567890",
	}

	result := cc.CheckDataCompliance(data)

	assert.False(t, result.IsCompliant)
	assert.NotEmpty(t, result.Violations)

	// Check that sensitive field violations are found
	var foundSensitiveViolation bool
	for _, v := range result.Violations {
		if v.Type == "sensitive_field" {
			foundSensitiveViolation = true
			break
		}
	}
	assert.True(t, foundSensitiveViolation)
}

func TestComplianceChecker_CheckDataCompliance_GDPRViolation(t *testing.T) {
	cc := NewComplianceChecker(true, 90)

	data := map[string]interface{}{
		"user_email": "john.doe@example.com", // Contains email pattern
	}

	result := cc.CheckDataCompliance(data)

	assert.False(t, result.IsCompliant)
	assert.True(t, result.PIIDetected)

	// Check for GDPR violation (PII without consent)
	var foundGDPRViolation bool
	for _, v := range result.Violations {
		if v.Type == "gdpr_violation" {
			foundGDPRViolation = true
			break
		}
	}
	assert.True(t, foundGDPRViolation)
}

func TestComplianceChecker_CheckDataCompliance_WithConsent(t *testing.T) {
	cc := NewComplianceChecker(true, 90)

	data := map[string]interface{}{
		"user_email":   "john.doe@example.com",
		"gdpr_consent": true,
	}

	result := cc.CheckDataCompliance(data)

	// PII is still detected but no GDPR consent violation
	assert.True(t, result.PIIDetected)

	// No GDPR consent violation
	var foundConsentViolation bool
	for _, v := range result.Violations {
		if v.Description == "PII data processed without GDPR consent" {
			foundConsentViolation = true
			break
		}
	}
	assert.False(t, foundConsentViolation)
}

func TestComplianceChecker_CheckDataCompliance_RetentionViolation(t *testing.T) {
	cc := NewComplianceChecker(false, 30)

	// Data created 60 days ago (exceeds 30-day retention)
	oldDate := time.Now().Add(-60 * 24 * time.Hour)
	data := map[string]interface{}{
		"created_at": oldDate,
	}

	result := cc.CheckDataCompliance(data)

	assert.False(t, result.IsCompliant)
	assert.False(t, result.RetentionCheck)

	var foundRetentionViolation bool
	for _, v := range result.Violations {
		if v.Type == "retention_violation" {
			foundRetentionViolation = true
			break
		}
	}
	assert.True(t, foundRetentionViolation)
}

func TestComplianceChecker_CheckDataCompliance_RetentionStringFormat(t *testing.T) {
	cc := NewComplianceChecker(false, 30)

	// Data created 60 days ago in RFC3339 string format
	oldDate := time.Now().Add(-60 * 24 * time.Hour).Format(time.RFC3339)
	data := map[string]interface{}{
		"created_at": oldDate,
	}

	result := cc.CheckDataCompliance(data)

	assert.False(t, result.RetentionCheck)
}

func TestComplianceChecker_DetectPII(t *testing.T) {
	cc := NewComplianceChecker(false, 90)

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected bool
	}{
		{
			name:     "email",
			data:     map[string]interface{}{"field": "test@example.com"},
			expected: true,
		},
		{
			name:     "phone",
			data:     map[string]interface{}{"field": "+1-555-123-4567"},
			expected: true,
		},
		{
			name:     "ssn",
			data:     map[string]interface{}{"field": "123-45-6789"},
			expected: true,
		},
		{
			name:     "credit_card",
			data:     map[string]interface{}{"field": "4111-1111-1111-1111"},
			expected: true,
		},
		{
			name:     "ip_address",
			data:     map[string]interface{}{"field": "192.168.1.1"},
			expected: true,
		},
		{
			name:     "api_key",
			data:     map[string]interface{}{"field": "sk_live_1234567890abcdefghijk"},
			expected: true,
		},
		{
			name:     "clean",
			data:     map[string]interface{}{"field": "Hello World"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cc.detectPII(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComplianceChecker_CheckRequestCompliance(t *testing.T) {
	cc := NewComplianceChecker(false, 90)

	// Clean request
	req := httptest.NewRequest("GET", "/api/models", nil)
	result := cc.CheckRequestCompliance(req)
	assert.True(t, result.IsCompliant)

	// Request with PII in query string
	req = httptest.NewRequest("GET", "/api/search?email=user@example.com", nil)
	result = cc.CheckRequestCompliance(req)
	assert.False(t, result.IsCompliant)
	assert.True(t, result.PIIDetected)

	// Request with sensitive header
	req = httptest.NewRequest("GET", "/api/models", nil)
	req.Header.Set("password", "secret123")
	result = cc.CheckRequestCompliance(req)
	assert.False(t, result.IsCompliant)
}

func TestComplianceChecker_GetRetentionPolicy(t *testing.T) {
	cc := NewComplianceChecker(false, 90)

	// Test known policies
	userDataPolicy := cc.GetRetentionPolicy("user_data")
	assert.Equal(t, "user_data", userDataPolicy.DataType)
	assert.Equal(t, 2555, userDataPolicy.RetentionDays)

	logsPolicy := cc.GetRetentionPolicy("logs")
	assert.Equal(t, "logs", logsPolicy.DataType)
	assert.Equal(t, 90, logsPolicy.RetentionDays)
	assert.True(t, logsPolicy.AutoDelete)

	// Test unknown policy (should return default)
	unknownPolicy := cc.GetRetentionPolicy("unknown_type")
	assert.Equal(t, "unknown_type", unknownPolicy.DataType)
	assert.Equal(t, 90, unknownPolicy.RetentionDays) // Uses cc.dataRetentionDays
}

// ==================== ContentFilter Tests ====================

func TestNewContentFilter(t *testing.T) {
	cf := NewContentFilter()

	assert.NotNil(t, cf)
	assert.NotNil(t, cf.bannedWords)
	assert.NotEmpty(t, cf.bannedWords)
	assert.NotNil(t, cf.bannedPatterns)
	assert.NotEmpty(t, cf.bannedPatterns)
	assert.NotNil(t, cf.toxicityWords)
	assert.NotEmpty(t, cf.toxicityWords)
}

func TestContentFilter_AddBannedWord(t *testing.T) {
	cf := NewContentFilter()

	cf.AddBannedWord("newword")

	assert.True(t, cf.bannedWords["newword"])
}

func TestContentFilter_AddBannedPattern(t *testing.T) {
	cf := NewContentFilter()

	err := cf.AddBannedPattern(`test\d+`)
	require.NoError(t, err)

	// Test invalid regex
	err = cf.AddBannedPattern(`[invalid`)
	assert.Error(t, err)
}

func TestContentFilter_FilterContent_Clean(t *testing.T) {
	cf := NewContentFilter()

	result, err := cf.FilterContent("This is a clean message")

	require.NoError(t, err)
	assert.True(t, result.IsAllowed)
	assert.Empty(t, result.Violations)
	assert.Equal(t, 0.0, result.RiskScore)
}

func TestContentFilter_FilterContent_BannedWord(t *testing.T) {
	cf := NewContentFilter()

	result, err := cf.FilterContent("This is inappropriate content")

	require.NoError(t, err)
	assert.False(t, result.IsAllowed)
	assert.NotEmpty(t, result.Violations)

	var foundBannedWord bool
	for _, v := range result.Violations {
		if v.Type == "banned_word" {
			foundBannedWord = true
			break
		}
	}
	assert.True(t, foundBannedWord)
}

func TestContentFilter_FilterContent_ToxicWord(t *testing.T) {
	cf := NewContentFilter()

	result, err := cf.FilterContent("This message contains hate")

	require.NoError(t, err)
	assert.False(t, result.IsAllowed)

	var foundToxicity bool
	for _, v := range result.Violations {
		if v.Type == "toxicity" {
			foundToxicity = true
			break
		}
	}
	assert.True(t, foundToxicity)
}

func TestContentFilter_FilterContent_PatternMatch(t *testing.T) {
	cf := NewContentFilter()

	// Test URL pattern
	result, err := cf.FilterContent("Check out https://example.com")
	require.NoError(t, err)
	assert.False(t, result.IsAllowed)

	// Test script pattern
	result, err = cf.FilterContent("Hello <script>alert('xss')</script>")
	require.NoError(t, err)
	assert.False(t, result.IsAllowed)

	// Test email pattern
	result, err = cf.FilterContent("Contact me at user@example.com")
	require.NoError(t, err)
	assert.False(t, result.IsAllowed)
}

func TestContentFilter_FilterContent_Censored(t *testing.T) {
	cf := NewContentFilter()

	result, err := cf.FilterContent("This is inappropriate")

	require.NoError(t, err)
	// Filtered content should have banned word censored
	assert.Contains(t, result.FilteredContent, "*")
}

func TestContentFilter_CalculateRiskScore(t *testing.T) {
	cf := NewContentFilter()

	// No violations = 0 score
	score := cf.calculateRiskScore([]ContentViolation{})
	assert.Equal(t, 0.0, score)

	// Low severity
	score = cf.calculateRiskScore([]ContentViolation{
		{Severity: "low"},
	})
	assert.Equal(t, 0.2, score)

	// Medium severity
	score = cf.calculateRiskScore([]ContentViolation{
		{Severity: "medium"},
	})
	assert.Equal(t, 0.5, score)

	// High severity
	score = cf.calculateRiskScore([]ContentViolation{
		{Severity: "high"},
	})
	assert.Equal(t, 1.0, score)

	// Multiple - should cap at 1.0
	score = cf.calculateRiskScore([]ContentViolation{
		{Severity: "high"},
		{Severity: "high"},
	})
	assert.Equal(t, 1.0, score)
}

func TestContentFilter_CheckToxicity(t *testing.T) {
	cf := NewContentFilter()

	// Clean content
	result := cf.CheckToxicity("Hello, how are you?")
	assert.False(t, result.IsToxic)
	assert.Equal(t, 0.0, result.Score)
	assert.Empty(t, result.ToxicWords)

	// Toxic content
	result = cf.CheckToxicity("This message contains hate and violence")
	assert.True(t, result.IsToxic)
	assert.Greater(t, result.Score, 0.0)
	assert.NotEmpty(t, result.ToxicWords)

	// Content with threat keywords
	result = cf.CheckToxicity("I will threat you and cause harm")
	assert.True(t, result.IsToxic)
}

// ==================== Validation Tests ====================

func TestValidateAlphaNumSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"hello world", true},
		{"test-name", true},
		{"test_name", true},
		{"test.name", true},
		{"Hello123", true},
		{"test@name", false},
		{"test#name", false},
		{"test$name", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateRequest(&struct {
				Name string `binding:"required,alphanumspace"`
			}{Name: tt.input})

			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"", true}, // Empty is valid (optional)
		{"ftp://example.com", false},
		{"not-a-url", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateRequest(&struct {
				URL string `binding:"url"`
			}{URL: tt.input})

			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name@domain.org", true},
		{"", true}, // Empty is valid (optional)
		{"invalid-email", false},
		{"@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateRequest(&struct {
				Email string `binding:"email"`
			}{Email: tt.input})

			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"info", true},
		{"warning", true},
		{"error", true},
		{"critical", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateRequest(&struct {
				Severity string `binding:"severity"`
			}{Severity: tt.input})

			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"pending", true},
		{"running", true},
		{"completed", true},
		{"failed", true},
		{"cancelled", true},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateRequest(&struct {
				Status string `binding:"status"`
			}{Status: tt.input})

			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		wantErr  bool
	}{
		{"Passw0rd!", false},         // Valid
		{"shor1A!", true},            // Too short (7 chars)
		{"password1!", true},         // No uppercase
		{"PASSWORD1!", true},         // No lowercase
		{"Password!!", true},         // No digit
		{"Password12", true},         // No special char
		{"ValidP@ssw0rd123", false},  // Valid
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		username string
		wantErr  bool
	}{
		{"validuser", false},
		{"valid_user", false},
		{"valid-user", false},
		{"user123", false},
		{"ab", true},                              // Too short
		{"user with space", true},                 // Contains space
		{"user@name", true},                                                          // Invalid character
		{"verylongusernamethatexceedsfiftycharsverylonguserna", true},                // Too long (51 chars)
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePhoneNumber(t *testing.T) {
	tests := []struct {
		phone   string
		wantErr bool
	}{
		{"5551234567", false},
		{"+1-555-123-4567", false},
		{"(555) 123-4567", false},
		{"123", true}, // Too short
	}

	for _, tt := range tests {
		t.Run(tt.phone, func(t *testing.T) {
			err := ValidatePhoneNumber(tt.phone)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	tests := []struct {
		json    string
		wantErr bool
	}{
		{`{"key": "value"}`, false},
		{`{"array": [1, 2, 3]}`, false},
		{`{"nested": {"key": "value"}}`, false},
		{`{"unclosed": "value"`, true},
		{`{"extra": "bracket"}}`, true},
		{`[1, 2, 3]`, false},
		{`["unclosed"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.json, func(t *testing.T) {
			err := ValidateJSON(tt.json)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCronExpression(t *testing.T) {
	tests := []struct {
		cron    string
		wantErr bool
	}{
		{"* * * * *", false},
		{"0 0 * * *", false},
		{"*/5 * * * *", false},
		{"", false},                    // Empty is valid
		{"0 0 *", true},                // Not enough fields
		{"0 0 * * * *", true},          // Too many fields
	}

	for _, tt := range tests {
		t.Run(tt.cron, func(t *testing.T) {
			err := ValidateCronExpression(tt.cron)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDateRange(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		wantErr   bool
	}{
		{
			name:      "valid range",
			startDate: now,
			endDate:   now.Add(24 * time.Hour),
			wantErr:   false,
		},
		{
			name:      "start after end",
			startDate: now.Add(24 * time.Hour),
			endDate:   now,
			wantErr:   true,
		},
		{
			name:      "range too large",
			startDate: now,
			endDate:   now.Add(400 * 24 * time.Hour), // > 1 year
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDateRange(tt.startDate, tt.endDate)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		minItems int
		maxItems int
		minLen   int
		maxLen   int
		wantErr  bool
	}{
		{
			name:     "valid slice",
			slice:    []string{"one", "two", "three"},
			minItems: 1,
			maxItems: 5,
			minLen:   1,
			maxLen:   10,
			wantErr:  false,
		},
		{
			name:     "too few items",
			slice:    []string{"one"},
			minItems: 2,
			maxItems: 5,
			minLen:   1,
			maxLen:   10,
			wantErr:  true,
		},
		{
			name:     "too many items",
			slice:    []string{"one", "two", "three"},
			minItems: 1,
			maxItems: 2,
			minLen:   1,
			maxLen:   10,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringSlice(tt.slice, tt.minItems, tt.maxItems, tt.minLen, tt.maxLen)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIntegerSlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int64
		minItems int
		maxItems int
		minVal   int64
		maxVal   int64
		wantErr  bool
	}{
		{
			name:     "valid slice",
			slice:    []int64{1, 2, 3},
			minItems: 1,
			maxItems: 5,
			minVal:   1,
			maxVal:   100,
			wantErr:  false,
		},
		{
			name:     "value too small",
			slice:    []int64{0, 1, 2},
			minItems: 1,
			maxItems: 5,
			minVal:   1,
			maxVal:   100,
			wantErr:  true,
		},
		{
			name:     "value too large",
			slice:    []int64{1, 101},
			minItems: 1,
			maxItems: 5,
			minVal:   1,
			maxVal:   100,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIntegerSlice(tt.slice, tt.minItems, tt.maxItems, tt.minVal, tt.maxVal)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetValidationErrors(t *testing.T) {
	// Create a struct that will fail validation
	type TestStruct struct {
		Name  string `binding:"required,min=5"`
		Email string `binding:"required,email"`
	}

	err := ValidateRequest(&TestStruct{
		Name:  "ab", // Too short
		Email: "invalid", // Not a valid email
	})

	if err != nil {
		errors := GetValidationErrors(err)
		assert.NotEmpty(t, errors)
	}
}

// ==================== Request Struct Tests ====================

func TestLoginRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request LoginRequest
		wantErr bool
	}{
		{
			name: "valid",
			request: LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "missing username",
			request: LoginRequest{
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			request: LoginRequest{
				Username: "testuser",
			},
			wantErr: true,
		},
		{
			name: "username too short",
			request: LoginRequest{
				Username: "ab",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "password too short",
			request: LoginRequest{
				Username: "testuser",
				Password: "short",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequest(&tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
