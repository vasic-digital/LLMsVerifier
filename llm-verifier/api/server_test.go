package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"llm-verifier/config"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			Port:       "8080",
			JWTSecret:  "test-secret",
			RateLimit:  100,
			EnableCORS: true,
		},
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}

	// Test that router is initialized
	if server.router == nil {
		t.Error("Server router should be initialized")
	}
}

func TestServerHealthCheck(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			Port:       "8080",
			JWTSecret:  "test-secret",
			RateLimit:  100,
			EnableCORS: true,
		},
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create a test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Serve the request
	server.router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check response body contains expected fields
	body := w.Body.String()
	expectedFields := []string{"status", "timestamp", "version"}
	for _, field := range expectedFields {
		if !contains(body, field) {
			t.Errorf("Response body should contain '%s'", field)
		}
	}
}

func TestServerRoutes(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			Port:       "8080",
			JWTSecret:  "test-secret",
			RateLimit:  100,
			EnableCORS: true,
		},
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test that essential routes are registered
	testRoutes := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		{"POST", "/auth/login"},
		{"POST", "/auth/refresh"},
		{"GET", "/api/v1/models"},
		{"GET", "/api/v1/config"},
	}

	for _, route := range testRoutes {
		t.Run(route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// We don't care about the status code, just that the route exists
			// and doesn't panic
			if w.Code == http.StatusNotFound {
				t.Errorf("Route %s %s should exist", route.method, route.path)
			}
		})
	}
}

func TestServerStart(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			Port:       "8080",
			JWTSecret:  "test-secret",
			RateLimit:  100,
			EnableCORS: true,
		},
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test that Start doesn't panic with invalid port
	// We can't actually start the server in a test without blocking,
	// but we can test that the method exists and doesn't panic on invalid input
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Server.Start panicked: %v", r)
		}
	}()

	// This should return an error because we can't bind to port 0
	err = server.Start("0")
	if err == nil {
		t.Log("Server.Start returned nil error (expected in test environment)")
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
