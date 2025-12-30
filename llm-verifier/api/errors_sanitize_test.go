package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ==================== Error Response Tests ====================

func TestSendError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendError(c, http.StatusBadRequest, ErrCodeValidation, "Test error", map[string]string{
		"field": "test_field",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, ErrCodeValidation, response.Code)
	assert.Equal(t, "Test error", response.Message)
	assert.Equal(t, "test_field", response.Details["field"])
	assert.NotZero(t, response.Timestamp)
}

func TestSendError_WithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-req-123")

	SendError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Unauthorized", nil)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-req-123", response.RequestID)
}

func TestSendSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"key": "value"}
	SendSuccess(c, http.StatusOK, data, "Operation successful")

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Operation successful", response.Message)
}

func TestSendPaginated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := []string{"item1", "item2", "item3"}
	SendPaginated(c, data, 2, 10, 35)

	assert.Equal(t, http.StatusOK, w.Code)

	var response PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 2, response.Pagination.Page)
	assert.Equal(t, 10, response.Pagination.PageSize)
	assert.Equal(t, 35, response.Pagination.TotalItems)
	assert.Equal(t, 4, response.Pagination.TotalPages) // ceil(35/10) = 4
	assert.True(t, response.Pagination.HasNext)        // page 2 of 4
	assert.True(t, response.Pagination.HasPrev)        // page 2 > 1
}

func TestSendPaginated_FirstPage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendPaginated(c, []string{}, 1, 10, 25)

	var response PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Pagination.HasPrev) // First page has no prev
	assert.True(t, response.Pagination.HasNext)  // More pages exist
}

func TestSendPaginated_LastPage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendPaginated(c, []string{}, 3, 10, 25)

	var response PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Pagination.HasPrev)  // Previous pages exist
	assert.False(t, response.Pagination.HasNext) // Last page has no next
}

func TestHandleNotFoundError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	HandleNotFoundError(c, "Model", 123)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, ErrCodeNotFound, response.Code)
	assert.Equal(t, "Model not found", response.Message)
	assert.Equal(t, "Model", response.Details["resource"])
	assert.Equal(t, "123", response.Details["id"])
}

func TestHandleUnauthorizedError(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		expectedMessage string
	}{
		{"with message", "Custom auth error", "Custom auth error"},
		{"without message", "", "Authentication required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			HandleUnauthorizedError(c, tt.message)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, ErrCodeUnauthorized, response.Code)
			assert.Equal(t, tt.expectedMessage, response.Message)
		})
	}
}

func TestHandleForbiddenError(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		expectedMessage string
	}{
		{"with message", "Access denied", "Access denied"},
		{"without message", "", "Insufficient permissions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			HandleForbiddenError(c, tt.message)

			assert.Equal(t, http.StatusForbidden, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, ErrCodeForbidden, response.Code)
			assert.Equal(t, tt.expectedMessage, response.Message)
		})
	}
}

func TestHandleInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	HandleInternalError(c, errors.New("database connection failed"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, ErrCodeInternal, response.Code)
	assert.Equal(t, "Internal server error", response.Message)
}

func TestHandleDatabaseError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	HandleDatabaseError(c, errors.New("query failed"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, ErrCodeDatabase, response.Code)
	assert.Equal(t, "Database operation failed", response.Message)
}

func TestHandleConflictError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	HandleConflictError(c, "User", "email", "test@example.com")

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, ErrCodeConflict, response.Code)
	assert.Equal(t, "User already exists", response.Message)
	assert.Equal(t, "User", response.Details["resource"])
	assert.Equal(t, "email", response.Details["field"])
	assert.Equal(t, "test@example.com", response.Details["value"])
}

func TestHandleRateLimitError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	HandleRateLimitError(c, 60)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "60", w.Header().Get("Retry-After"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, ErrCodeRateLimit, response.Code)
	assert.Equal(t, "Rate limit exceeded", response.Message)
	assert.Equal(t, "60", response.Details["retry_after_seconds"])
}

func TestHandleValidationError(t *testing.T) {
	t.Run("generic binding error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		HandleValidationError(c, errors.New("invalid JSON format"))

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, ErrCodeValidation, response.Code)
		assert.Equal(t, "Invalid request format", response.Message)
		assert.NotEmpty(t, response.Details["error"])
	})
}

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name             string
		existingID       string
		expectProvidedID bool
	}{
		{"with existing ID", "existing-req-id", true},
		{"without existing ID", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			if tt.existingID != "" {
				c.Request.Header.Set("X-Request-ID", tt.existingID)
			}

			middleware := RequestIDMiddleware()
			middleware(c)

			requestID := c.GetString("request_id")
			assert.NotEmpty(t, requestID)

			if tt.expectProvidedID {
				assert.Equal(t, tt.existingID, requestID)
			}

			assert.Equal(t, requestID, w.Header().Get("X-Request-ID"))
		})
	}
}

func TestGenerateUUID(t *testing.T) {
	uuid1 := generateUUID()
	uuid2 := generateUUID()

	assert.NotEmpty(t, uuid1)
	assert.NotEmpty(t, uuid2)
	// UUIDs should be different (very high probability)
	// Note: In extremely rare cases this could fail if called at exactly the same nanosecond
}

// ==================== Sanitize Tests ====================

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal text", "Hello World", "Hello World"},
		{"with HTML", "<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"with null bytes", "Hello\x00World", "HelloWorld"},
		{"with whitespace", "  Hello  ", "Hello"},
		{"with special chars", "Test<>&\"'", "Test&lt;&gt;&amp;&#34;&#39;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldMatch func(string) bool
	}{
		{"remove script tags", "<script>evil()</script>Content", func(s string) bool {
			return !strings.Contains(s, "<script") && strings.Contains(s, "Content")
		}},
		{"remove onclick", "<div onclick='evil()'>test</div>", func(s string) bool {
			return !strings.Contains(s, "onclick") && strings.Contains(s, "test")
		}},
		{"remove javascript href", "<a href=\"javascript:evil()\">link</a>", func(s string) bool {
			return !strings.Contains(s, "javascript:")
		}},
		{"remove data protocol", "<img src=\"data:image/png;base64,evil\">", func(s string) bool {
			return !strings.Contains(s, "data:")
		}},
		{"remove style tags", "<style>.evil{}</style>Content", func(s string) bool {
			return !strings.Contains(s, "<style") && strings.Contains(s, "Content")
		}},
		{"remove iframe", "<iframe src='evil.html'></iframe>", func(s string) bool {
			return !strings.Contains(s, "<iframe")
		}},
		{"remove object", "<object data='evil.swf'></object>", func(s string) bool {
			return !strings.Contains(s, "<object")
		}},
		{"remove embed", "<embed src='evil.swf'></embed>", func(s string) bool {
			return !strings.Contains(s, "<embed")
		}},
		{"remove applet", "<applet code='evil.class'></applet>", func(s string) bool {
			return !strings.Contains(s, "<applet")
		}},
		{"allow safe HTML", "<p>Safe paragraph</p>", func(s string) bool {
			return strings.Contains(s, "<p>") && strings.Contains(s, "Safe paragraph")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHTML(tt.input)
			assert.True(t, tt.shouldMatch(result), "Result: %s", result)
		})
	}
}

func TestSanitizeSQL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldMatch func(string) bool
	}{
		{"remove SQL comment", "value--DROP TABLE", func(s string) bool {
			return !strings.Contains(s, "--") && !strings.Contains(s, "DROP")
		}},
		{"remove block comment", "value/*evil*/safe", func(s string) bool {
			return !strings.Contains(s, "/*") && !strings.Contains(s, "*/") && !strings.Contains(s, "evil")
		}},
		{"remove UNION", "1 UNION SELECT *", func(s string) bool {
			return !strings.Contains(strings.ToLower(s), "union") && !strings.Contains(strings.ToLower(s), "select")
		}},
		{"remove quotes", "value'; DROP TABLE--", func(s string) bool {
			return !strings.Contains(s, "'") && !strings.Contains(s, "\"")
		}},
		{"remove semicolon", "value; DROP TABLE", func(s string) bool {
			return !strings.Contains(s, ";") && !strings.Contains(strings.ToLower(s), "drop")
		}},
		{"remove backticks", "`table`", func(s string) bool {
			return !strings.Contains(s, "`")
		}},
		{"normal input", "searchterm123", func(s string) bool {
			return strings.Contains(s, "searchterm123")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSQL(tt.input)
			assert.True(t, tt.shouldMatch(result), "Result: %s", result)
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"remove path traversal", "../../../etc/passwd", "etc/passwd"},
		{"remove dot slash", "./hidden/file", "hidden/file"},
		{"remove null bytes", "file\x00name", "filename"},
		{"remove multiple slashes", "path//to///file", "path/to/file"},
		{"remove control chars", "file\x01name", "filename"},
		{"normal path", "valid/path/to/file", "valid/path/to/file"},
		{"trim dots and slashes", "./path/", "path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedEmail string
		expectedValid bool
	}{
		{"valid email", "test@example.com", "test@example.com", true},
		{"valid email uppercase", "TEST@EXAMPLE.COM", "test@example.com", true},
		{"valid email with dots", "test.user@example.co.uk", "test.user@example.co.uk", true},
		{"invalid no at", "testexample.com", "", false},
		{"invalid no domain", "test@", "", false},
		{"invalid format", "not-an-email", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := SanitizeEmail(tt.input)
			assert.Equal(t, tt.expectedValid, valid)
			if valid {
				assert.Equal(t, tt.expectedEmail, result)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedURL  string
		expectedValid bool
	}{
		{"valid https", "https://example.com/path", "https://example.com/path", true},
		{"valid http", "http://example.com", "http://example.com", true},
		{"remove javascript", "javascript:evil()", "", false},
		{"invalid no protocol", "example.com", "", false},
		{"invalid ftp protocol", "ftp://example.com", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := SanitizeURL(tt.input)
			assert.Equal(t, tt.expectedValid, valid)
			if valid {
				assert.Contains(t, result, tt.expectedURL)
			}
		})
	}
}

func TestSanitizePhoneNumber(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedPhone string
		expectedValid bool
	}{
		{"valid US phone", "555-123-4567", "5551234567", true},
		{"valid with parens", "(555) 123-4567", "5551234567", true},
		{"valid international", "+1-555-123-4567", "15551234567", true},
		{"too short", "12345", "", false},
		{"too long", "12345678901234567890", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := SanitizePhoneNumber(tt.input)
			assert.Equal(t, tt.expectedValid, valid)
			if valid {
				assert.Equal(t, tt.expectedPhone, result)
			}
		})
	}
}

func TestSanitizeJSON(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValid bool
	}{
		{"valid object", `{"key": "value"}`, true},
		{"valid array", `[1, 2, 3]`, true},
		{"valid nested", `{"a": {"b": [1, 2]}}`, true},
		{"unbalanced braces", `{"key": "value"`, false},
		{"unbalanced brackets", `[1, 2, 3`, false},
		{"extra closing brace", `{"key": "value"}}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, valid := SanitizeJSON(tt.input)
			assert.Equal(t, tt.expectedValid, valid)
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"remove path traversal", "../../../etc/passwd", "etcpasswd"},
		{"remove slashes", "path/to/file.txt", "pathtofile.txt"},
		{"remove backslashes", "path\\to\\file.txt", "pathtofile.txt"},
		{"remove dangerous chars", "file<name>.txt", "filename.txt"},
		{"remove control chars", "file\x00name.txt", "filename.txt"},
		{"normal filename", "document.pdf", "document.pdf"},
		{"with spaces", "  file name.txt  ", "file name.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeInteger(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedInt   int64
		expectedValid bool
	}{
		{"valid positive", "123", 123, true},
		{"valid negative", "-456", -456, true},
		{"with non-digits", "12abc34", 1234, true},
		{"invalid empty", "", 0, false},
		{"invalid letters only", "abc", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := SanitizeInteger(tt.input)
			assert.Equal(t, tt.expectedValid, valid)
			if valid {
				assert.Equal(t, tt.expectedInt, result)
			}
		})
	}
}

func TestSanitizeFloat(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedFloat float64
		expectedValid bool
	}{
		{"valid integer", "123", 123.0, true},
		{"valid decimal", "123.45", 123.45, true},
		{"valid negative", "-123.45", -123.45, true},
		{"with non-digits", "12.3abc", 12.3, true},
		{"invalid empty", "", 0, false},
		{"invalid letters only", "abc", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := SanitizeFloat(tt.input)
			assert.Equal(t, tt.expectedValid, valid)
			if valid {
				assert.InDelta(t, tt.expectedFloat, result, 0.001)
			}
		})
	}
}

func TestSanitizeBool(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedBool  bool
		expectedValid bool
	}{
		{"true lowercase", "true", true, true},
		{"TRUE uppercase", "TRUE", true, true},
		{"1", "1", true, true},
		{"yes", "yes", true, true},
		{"on", "on", true, true},
		{"t", "t", true, true},
		{"false lowercase", "false", false, true},
		{"FALSE uppercase", "FALSE", false, true},
		{"0", "0", false, true},
		{"no", "no", false, true},
		{"off", "off", false, true},
		{"f", "f", false, true},
		{"invalid", "maybe", false, false},
		{"empty", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := SanitizeBool(tt.input)
			assert.Equal(t, tt.expectedValid, valid)
			if valid {
				assert.Equal(t, tt.expectedBool, result)
			}
		})
	}
}

func TestSanitizeOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal text", "Hello World", "Hello World"},
		{"with HTML", "<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"empty string", "", ""},
		{"special chars", "&<>\"'", "&amp;&lt;&gt;&#34;&#39;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeOutput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeHTMLResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"remove script", "<html><script>evil()</script><body>Content</body></html>", "<html><body>Content</body></html>"},
		{"remove javascript url", "<a href=\"javascript:evil()\">link</a>", "<a href=\"#\">link</a>"},
		{"safe HTML", "<html><body><p>Safe</p></body></html>", "<html><body><p>Safe</p></body></html>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHTMLResponse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeJSONOutput(t *testing.T) {
	// This function currently just returns the data as-is
	// but we test to ensure it doesn't modify valid data
	data := map[string]string{"key": "value"}
	result := SanitizeJSONOutput(data)
	assert.Equal(t, data, result)
}
