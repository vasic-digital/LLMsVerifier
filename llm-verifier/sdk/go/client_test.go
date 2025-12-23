package llmverifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLLMVerifierClient(t *testing.T) {
	client := NewLLMVerifierClient("http://localhost:8080", "test-key")
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080", client.baseURL)
	assert.Equal(t, "test-key", client.apiKey)
	assert.NotNil(t, client.httpClient)
}

func TestNewLLMVerifierClientWithEmptyURL(t *testing.T) {
	client := NewLLMVerifierClient("", "")
	assert.NotNil(t, client)
	assert.Equal(t, "", client.baseURL)
}

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/test", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	var result map[string]string
	err := client.get("/test", nil, &result)
	assert.NoError(t, err)
	assert.Equal(t, "ok", result["message"])
}

func TestClientGetWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "1", r.URL.Query().Get("limit"))
		assert.Equal(t, "test", r.URL.Query().Get("provider"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]int{1, 2, 3})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	params := map[string]string{"limit": "1", "provider": "test"}
	var result []int
	err := client.get("/test", params, &result)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, result)
}

func TestClientGetWithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "test-token")
	var result map[string]string
	err := client.get("/test", nil, &result)
	assert.NoError(t, err)
}

func TestClientGetError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	var result map[string]string
	err := client.get("/test", nil, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		body, _ := io.ReadAll(r.Body)
		var data map[string]string
		json.Unmarshal(body, &data)
		assert.Equal(t, "value", data["key"])
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	var result map[string]string
	err := client.post("/test", map[string]string{"key": "value"}, &result)
	assert.NoError(t, err)
}

func TestClientPostWithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "test-token")
	var result map[string]string
	err := client.post("/test", nil, &result)
	assert.NoError(t, err)
}

func TestClientPostError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	var result map[string]string
	err := client.post("/test", nil, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestClientPostNilResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	err := client.post("/test", nil, nil)
	assert.NoError(t, err)
}

func TestLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var data map[string]string
		json.Unmarshal(body, &data)
		assert.Equal(t, "user", data["username"])
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AuthResponse{
			Token: "new-token",
			ExpiresAt: "2024-01-01T00:00:00Z",
			User: User{ID: 1, Username: "user", Email: "user@example.com", Role: "admin"},
		})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "old-token")
	resp, err := client.Login("user", "pass")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "new-token", resp.Token)
	assert.Equal(t, "new-token", client.apiKey)
}

func TestLoginError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	resp, err := client.Login("user", "pass")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGetModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/models", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]Model{
			{ID: 1, Name: "GPT-4", ProviderID: 1, ModelID: "gpt-4", Score: 95.5, Status: "verified", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: 2, Name: "Claude", ProviderID: 2, ModelID: "claude", Score: 93.0, Status: "verified", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	models, err := client.GetModels(10, 0, "")
	assert.NoError(t, err)
	assert.Len(t, models, 2)
	assert.Equal(t, "GPT-4", models[0].Name)
}

func TestGetModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/models/1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Model{
			ID: 1, Name: "GPT-4", ProviderID: 1, ModelID: "gpt-4", Description: "Test",
			Architecture: "transformer", Score: 95.5, Status: "verified", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	model, err := client.GetModel(1)
	assert.NoError(t, err)
	assert.NotNil(t, model)
	assert.Equal(t, 1, model.ID)
}

func TestGetModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "model not found"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	_, err := client.GetModel(999)
	assert.Error(t, err)
}

func TestVerifyModel(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VerificationResult{
			ID: 1, ModelID: 1, Status: "completed", Score: 95.5, StartedAt: now,
		})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	result, err := client.VerifyModel("gpt-4")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]Provider{
			{ID: 1, Name: "OpenAI", Endpoint: "https://api.openai.com/v1", Status: "active", CreatedAt: time.Now()},
		})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	providers, err := client.GetProviders()
	assert.NoError(t, err)
	assert.Len(t, providers, 1)
}

func TestGetHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthStatus{Status: "healthy", Timestamp: time.Now(), Uptime: "24h", Version: "1.0.0"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	health, err := client.GetHealth()
	assert.NoError(t, err)
	assert.NotNil(t, health)
}

func TestGetSystemInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SystemInfo{Version: "1.0.0", GoVersion: "1.21.0", ModelsCount: 10, ProvidersCount: 3, Uptime: "24h"})
	}))
	defer server.Close()

	client := NewLLMVerifierClient(server.URL, "")
	info, err := client.GetSystemInfo()
	assert.NoError(t, err)
	assert.NotNil(t, info)
}
