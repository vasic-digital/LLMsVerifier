package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthHandler_WithoutDatabase_StatusNotConfigured(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/health", nil)

	server := &Server{}
	server.HealthHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"database":"not_configured"`)
}

func TestHealthHandler_InvalidMethod(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/health", nil)

	server := &Server{}
	server.HealthHandler(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestListModelsHandler_NoDatabase_Returns503(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/models", nil)

	server := &Server{database: nil}
	server.ListModelsHandler(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "Database not available")
}

func TestListProvidersHandler_NoDatabase_Returns503(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/providers", nil)

	server := &Server{database: nil}
	server.ListProvidersHandler(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "Database not available")
}

func TestAddProviderHandler_NoDatabase_Returns503(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/providers", nil)

	server := &Server{database: nil}
	server.AddProviderHandler(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "Database not available")
}

func TestGetModelHandler_NoDatabase_Returns503(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/models/1", nil)

	server := &Server{database: nil}
	server.GetModelHandler(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "Database not available")
}

func TestVerifyModelHandler_NoDatabase_Returns503(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/models/1/verify", nil)

	server := &Server{database: nil}
	server.VerifyModelHandler(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "Database not available")
}

func TestProvidersHandler_UnknownMethod_Returns405(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/providers", nil)

	server := &Server{}
	server.ProvidersHandler(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}
