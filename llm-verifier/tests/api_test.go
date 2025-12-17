package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"llm-verifier/config"
	"llm-verifier/llmverifier"
)

// TestAPIEndpoints tests enterprise API endpoints
func TestAPIEndpoints(t *testing.T) {
	t.Run("API Endpoints", func(t *testing.T) {
		// Setup test server
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			
			// Authentication endpoint test
			if r.URL.Path == "/api/enterprise/auth/login" && r.Method == http.MethodPost {
				t.Logf("Testing login endpoint")
				
				// Test login with valid credentials
				loginReq := map[string]string{
					"username": "testuser",
					"password": "testpassword",
				}
				loginBody, _ := json.Marshal(loginReq)
				loginResp := httptest.NewRecorder()
				testServer.Config.Handler.ServeHTTP(loginResp, r)
				
				if loginResp.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", loginResp.Code)
				}
				
				// Parse response
				var loginRespData map[string]interface{}
				if err := json.Unmarshal(loginResp.Body.Bytes(), &loginRespData); err != nil {
					t.Errorf("Failed to parse login response: %v", err)
				}
				
				if !loginRespData["success"].(bool) {
					t.Errorf("Expected successful login")
				}
				
				// Verify token is present
				if _, ok := loginRespData["token"].(string); !ok {
					t.Errorf("Expected token in response")
				}
				
				t.Logf("Login test passed")
			}
			
			// Users endpoint test
			if r.URL.Path == "/api/enterprise/users" && r.Method == http.MethodGet {
				t.Logf("Testing users endpoint")
				
				usersResp := httptest.NewRecorder()
				testServer.Config.Handler.ServeHTTP(usersResp, r)
				
				if usersResp.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", usersResp.Code)
				}
				
				var usersData []interface{}
				if err := json.Unmarshal(usersResp.Body.Bytes(), &usersData); err != nil {
					t.Errorf("Failed to parse users response: %v", err)
				}
				
				if usersData["users"].([]interface{}) == nil {
					t.Error("Expected users array in response")
				}
				
				userCount := len(usersData["users"].([]interface{}))
				if userCount < 2 {
					t.Errorf("Expected at least 2 users")
				}
				
				t.Logf("Users test passed - %d users found", userCount)
			}
			
			// Audit endpoint test
			if r.URL.Path == "/api/enterprise/audit" && r.Method == http.MethodGet {
				t.Logf("Testing audit endpoint")
				
				auditResp := httptest.NewRecorder()
				testServer.Config.Handler.ServeHTTP(auditResp, r)
				
				if auditResp.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", auditResp.Code)
				}
				
				var auditData []interface{}
				if err := json.Unmarshal(auditResp.Body.Bytes(), &auditData); err != nil {
					t.Errorf("Failed to parse audit response: %v", err)
				}
				
				if len(auditData["audit"].([]interface{})) == 0 {
					t.Error("Expected audit entries in response")
				}
				
				t.Logf("Audit test passed - %d audit entries found", len(auditData["audit"].([]interface{})))
			}
			
			// Metrics endpoint test
			if r.URL.Path == "/api/enterprise/metrics" && r.Method == http.MethodGet {
				t.Logf("Testing metrics endpoint")
				
				metricsResp := httptest.NewRecorder()
				testServer.Config.Handler.ServeHTTP(metricsResp, r)
				
				if metricsResp.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", metricsResp.Code)
				}
				
				var metricsData map[string]interface{}
				if err := json.Unmarshal(metricsResp.Body.Bytes(), &metricsData); err != nil {
					t.Errorf("Failed to parse metrics response: %v", err)
				}
				
				// Verify metrics structure
				expectedMetrics := map[string]interface{}{
					"total_users":   2,
					"total_tenants":   1,
					"uptime_seconds": 3600,
					"requests_per_second": 25.5,
				"error_rate": 0.02,
					"memory_usage_mb": 128,
					"health_score": 0.95,
				"features": map[string]bool{
						"rbac":          true,
						"multi_tenant":     true,
						"audit_logging":    true,
						"security":       false,
						"rate_limiting": true,
						"authentication": true,
					},
				}
				
				for key, expectedValue := range expectedMetrics {
					if actualValue, ok := metricsData[key]; ok && actualValue != expectedValue {
						t.Errorf("Metrics mismatch for '%s': expected %v, got %v", key, expectedValue, actualValue)
					}
				}
				
				t.Logf("Metrics test passed - all metrics verified")
			}
			
			// Health endpoint test
			if r.URL.Path == "/api/enterprise/health" && r.Method == http.MethodGet {
				t.Logf("Testing health endpoint")
				
				healthResp := httptest.NewRecorder()
				testServer.Config.Handler.ServeHTTP(healthResp, r)
				
				if healthResp.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", healthResp.Code)
				}
				
				var healthData map[string]interface{}
				if err := json.Unmarshal(healthResp.Body.Bytes(), &healthData); err != nil {
					t.Errorf("Failed to parse health response: %v", err)
				}
				
				// Verify health status structure
				if healthData["status"].(string) != "healthy" {
					t.Errorf("Expected 'healthy' status, got '%s'", healthData["status"])
				}
				
				// Verify features are enabled
				features, ok := healthData["features"].(map[string]interface{})
				expectedFeatures := map[string]bool{
					"rbac":          true,
					"multi_tenant":     true,
					"audit_logging":    true,
					"rate_limiting": true,
					"authentication": true,
				}
				
				for key, expectedEnabled := range expectedFeatures {
					if actualEnabled, ok := features[key]; !ok || actualEnabled != expectedEnabled {
						t.Errorf("Feature '%s' expected %v, got %v", key, expectedEnabled, actualEnabled)
					}
				}
				
				// Verify checks
				checks, checks := healthData["checks"].(map[string]interface{})
				for _, checkName := range []string{"database", "api_server", "file_system", "memory", "cache"} {
					if checkData, ok := checks[checkName].(map[string]interface{}); ok {
						if checkStatus, ok := checkData["status"].(string); !ok {
							t.Errorf("Health check '%s' failed: %s", checkName, checkData["status"])
						}
					}
				}
				
				t.Logf("Health test passed - all checks passed")
			}

			default:
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
			}
			}
		})
		})
}

/*
// TestAPIErrorHandling tests API error scenarios
func TestAPIErrorHandling(t *testing.T) {
	t.Run("API Error Handling", func(t *testing.T) {
		// Test 404 Not Found
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Resource not found"})
		})
		
		req := httptest.NewRequest("GET", "/api/enterprise/nonexistent")
		resp, err := http.DefaultClient.Do(req)
		
		if err != nil {
			t.Errorf("Request failed: %v", err)
		}
		
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
		
		t.Logf("404 Not Found test passed")
	})
	
	t.Run("API Error Handling - 500 Internal Error", func(t *testing.T) {
		// Simulate internal server error
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
		})
		
		req := httptest.NewRequest("GET", "/api/enterprise/error/internal")
		resp, err := http.DefaultClient.Do(req)
		
		if err != nil {
			t.Errorf("Request failed: %v", err)
		}
		
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", resp.StatusCode)
		}
		
		t.Logf("500 Internal Error test passed")
	})
	
	t.Run("API Error Handling - Method Not Allowed", func(t *testing.T) {
		// Test unsupported HTTP methods
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().sET("Content-Type", "application/json")
			
			switch r.Method {
			case http.MethodPost:
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			case http.MethodPut:
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			case http.MethodDelete:
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			}
		})
			}
		}
		})
		})
		
			req := httptest.NewRequest("DELETE", "/api/enterprise/users")
		resp, err := http.DefaultClient.Do(req)
		
		if err != nil {
			t.Errorf("DELETE request failed: %v", err)
		}
		
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 Method Not Allowed, got %d", resp.StatusCode)
		}
		
		t.Logf("Method Not Allowed test passed")
	})
	
	t.Run("API Error Handling - Timeout", func(t *testing.T) {
		// Test request timeout scenarios
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Simulate timeout by not responding
			// In a real scenario, this would have a timeout handler
		w.WriteHeader(http.StatusRequestTimeout)
			json.NewEncoder(w).Encode(map[string]string{"error": "Request timeout"})
		})
		
		req := httptest.NewRequest("POST", "/api/enterprise/login", strings.NewReader("{\"username\":\"testuser\",\"password\":\"testpassword\"}"))
		req.Header.Set("Content-Type", "application/json")
		
		// Set a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
			
		resp, err := http.DefaultClient.Do(req)
		
		if ctx.Err() == context.Canceled {
			t.Logf("Request timed out as expected")
			return
		}
		
		if err != nil {
			t.Errorf("Request failed: %v", err)
		}
		
		if resp.StatusCode != http.StatusRequestTimeout {
			t.Errorf("Expected status 408 Request Timeout, got %d", resp.StatusCode)
		}
		
		t.Logf("Request Timeout test passed")
	})
}

// TestAPIResponseFormat tests response structure and format
func TestAPIResponseFormat(t *testing.T) {
	t.Run("API Response Format", func(t *testing.T) {
		// Test that all responses have consistent JSON structure
		testEndpoints := []string{
			"/api/enterprise/auth/login",
			"/api/enterprise/users",
			"/api/enterprise/tenants",
			"/api/enterprise/audit",
			"/api/enterprise/metrics",
			"/api/enterprise/health",
		}
		
		for _, endpoint := range testEndpoints {
			t.Run("Response Format - "+endpoint, func(t *testing.T) {
				req := httptest.NewRequest("GET", endpoint, nil)
				resp, err := http.DefaultClient.Do(req)
				
				if err != nil {
					t.Errorf("Request failed for %s: %v", err)
				}
				
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected status 200, got %d for %s", endpoint, resp.StatusCode)
				}
				
				var responseData map[string]interface{}
				if err := json.Unmarshal(resp.Body.Bytes(), &responseData); err != nil {
					t.Errorf("Failed to parse response for %s: %v", err)
				}
				
				// Verify response has required fields
				if endpoint != "/api/enterprise/metrics" {
					requiredFields := []string{"status", "timestamp"}
					for field := range requiredFields {
						if _, exists := responseData[field]; !exists {
							t.Errorf("Missing required field '%s' in response for %s", field, endpoint)
						}
					}
				}
				
				// Verify data types
				switch endpoint {
				case "/api/enterprise/auth/login":
					if _, ok := responseData["success"].(bool); !ok {
						t.Errorf("Login response missing success field")
					}
					if _, ok := responseData["token"].(string); !ok {
						t.Errorf("Login response missing token field")
					}
					
				case "/api/enterprise/users":
					if _, ok := responseData["users"]; !ok {
						t.Errorf("Users response missing users array")
					}
					users := responseData["users"].([]interface{})
					for _, user := range users {
						if userMap, ok := user.(map[string]interface{}); ok {
							if _, ok := userMap["id"].(string); !ok {
								t.Errorf("User missing id field")
							}
							if _, ok := userMap["username"].(string); !ok {
								t.Errorf("User missing username field")
							}
						}
					}
				
				case "/api/enterprise/tenants":
					if _, ok := responseData["tenants"]; !ok {
						t.Errorf("Tenants response missing tenants array")
					}
					tenants := responseData["tenants"].([]interface{})
					for _, tenant := range tenants {
						if tenantMap, ok := tenant.(map[string]interface{}); ok {
							if _, ok := tenantMap["id"].(string); !ok {
								t.Errorf("Tenant missing id field")
							}
							if _, ok := tenantMap["name"].(string); !ok {
								t.Errorf("Tenant missing name field")
							}
						}
					}
				default:
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
				}
				
				t.Logf("Response format test passed for %s", endpoint)
			})
		}
	})
	}
}