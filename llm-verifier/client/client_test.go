package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	baseURL := "http://localhost:8080"
	client := New(baseURL)

	assert.NotNil(t, client)
	assert.Equal(t, baseURL, client.baseURL)
	assert.NotNil(t, client.httpClient)
}

func TestSetToken(t *testing.T) {
	client := New("http://localhost:8080")
	token := "test-token-123"

	client.SetToken(token)
	assert.Equal(t, token, client.token)
}

func TestDoRequest(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := New(server.URL)

	resp, err := client.doRequest("GET", "/test", nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp.Body.Close()
}

func TestDoRequest_WithToken(t *testing.T) {
	// Mock server that checks for Authorization header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-token", authHeader)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "authenticated"}`))
	}))
	defer server.Close()

	client := New(server.URL)
	client.SetToken("test-token")

	resp, err := client.doRequest("GET", "/protected", nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp.Body.Close()
}

func TestDoRequest_WithBody(t *testing.T) {
	// Mock server that checks request body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/test", r.URL.Path)

		// Check content type
		contentType := r.Header.Get("Content-Type")
		assert.Equal(t, "application/json", contentType)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	client := New(server.URL)
	testData := map[string]interface{}{"name": "test"}

	resp, err := client.doRequest("POST", "/test", testData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	resp.Body.Close()
}

func TestDoRequest_ServerError(t *testing.T) {
	// Mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := New(server.URL)

	resp, err := client.doRequest("GET", "/error", nil)
	// doRequest returns an error for HTTP status codes >= 400
	assert.Error(t, err)
	assert.Nil(t, resp) // Response should be nil on error
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}

func TestGetModels_MethodExists(t *testing.T) {
	client := New("http://localhost:8080")

	// Test that the method exists and has the right signature
	// We don't test actual HTTP call here to avoid dependency on a running server
	assert.NotNil(t, client)

	// The method should exist without panicking
	assert.NotPanics(t, func() {
		client.GetModels()
	})
}

func TestGetProviders_MethodExists(t *testing.T) {
	client := New("http://localhost:8080")

	// Test that the method exists
	assert.NotNil(t, client)

	// The method should exist without panicking
	assert.NotPanics(t, func() {
		client.GetProviders()
	})
}

func TestGetVerificationResults_MethodExists(t *testing.T) {
	client := New("http://localhost:8080")

	// Test that the method exists
	assert.NotNil(t, client)

	// The method should exist without panicking
	assert.NotPanics(t, func() {
		client.GetVerificationResults()
	})
}
