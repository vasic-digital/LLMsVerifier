package client

import (
	"encoding/json"
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

func TestLoginSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"token": "new-token"})
	}))
	defer server.Close()
	client := New(server.URL)
	err := client.Login("user", "pass")
	assert.NoError(t, err)
	assert.Equal(t, "new-token", client.token)
}

func TestLoginFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	client := New(server.URL)
	err := client.Login("user", "wrong")
	assert.Error(t, err)
}

func TestGetModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string][]map[string]interface{}{
			"models": {{"id": "1", "name": "GPT-4"}},
		})
	}))
	defer server.Close()
	client := New(server.URL)
	models, err := client.GetModels()
	assert.NoError(t, err)
	assert.Len(t, models, 1)
}

func TestGetModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "1", "name": "GPT-4"})
	}))
	defer server.Close()
	client := New(server.URL)
	model, err := client.GetModel("1")
	assert.NoError(t, err)
	assert.Equal(t, "GPT-4", model["name"])
}

func TestCreateModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "1", "name": "New"})
	}))
	defer server.Close()
	client := New(server.URL)
	model, err := client.CreateModel(map[string]interface{}{"name": "New"})
	assert.NoError(t, err)
	assert.Equal(t, "New", model["name"])
}

func TestVerifyModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "completed"})
	}))
	defer server.Close()
	client := New(server.URL)
	result, err := client.VerifyModel("1")
	assert.NoError(t, err)
	assert.Equal(t, "completed", result["status"])
}

func TestGetProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string][]map[string]interface{}{
			"providers": {{"id": "1", "name": "OpenAI"}},
		})
	}))
	defer server.Close()
	client := New(server.URL)
	providers, err := client.GetProviders()
	assert.NoError(t, err)
	assert.Len(t, providers, 1)
}

func TestGetProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "1", "name": "OpenAI"})
	}))
	defer server.Close()
	client := New(server.URL)
	_, err := client.GetProvider("1")
	assert.NoError(t, err)
}

func TestGetVerificationResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "1"}})
	}))
	defer server.Close()
	client := New(server.URL)
	results, err := client.GetVerificationResults()
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestGetVerificationResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "1"})
	}))
	defer server.Close()
	client := New(server.URL)
	_, err := client.GetVerificationResult("1")
	assert.NoError(t, err)
}

func TestGetPricing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string][]map[string]interface{}{
			"pricing": {{"model": "gpt-4"}},
		})
	}))
	defer server.Close()
	client := New(server.URL)
	pricing, err := client.GetPricing()
	assert.NoError(t, err)
	assert.Len(t, pricing, 1)
}

func TestGetLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{{"limit": "100"}})
	}))
	defer server.Close()
	client := New(server.URL)
	_, err := client.GetLimits()
	assert.NoError(t, err)
}

func TestGetIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "1"}})
	}))
	defer server.Close()
	client := New(server.URL)
	issues, err := client.GetIssues()
	assert.NoError(t, err)
	assert.Len(t, issues, 1)
}

func TestGetEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "1"}})
	}))
	defer server.Close()
	client := New(server.URL)
	events, err := client.GetEvents()
	assert.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestGetSchedules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "1"}})
	}))
	defer server.Close()
	client := New(server.URL)
	schedules, err := client.GetSchedules()
	assert.NoError(t, err)
	assert.Len(t, schedules, 1)
}

func TestGetConfigExports(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "1"}})
	}))
	defer server.Close()
	client := New(server.URL)
	_, err := client.GetConfigExports()
	assert.NoError(t, err)
}

func TestDownloadConfigExport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "test"}`))
	}))
	defer server.Close()
	client := New(server.URL)
	data, err := client.DownloadConfigExport("1")
	assert.NoError(t, err)
	assert.Equal(t, `{"data": "test"}`, string(data))
}

func TestGetLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{{"level": "info"}})
	}))
	defer server.Close()
	client := New(server.URL)
	_, err := client.GetLogs()
	assert.NoError(t, err)
}

func TestGetConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.0"})
	}))
	defer server.Close()
	client := New(server.URL)
	_, err := client.GetConfig()
	assert.NoError(t, err)
}

func TestExportConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"url": "/download"})
	}))
	defer server.Close()
	client := New(server.URL)
	_, err := client.ExportConfig("json")
	assert.NoError(t, err)
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := New(server.URL)
	models, err := client.GetModels()
	assert.Error(t, err)
	assert.Nil(t, models)
}

func TestClientMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid"))
	}))
	defer server.Close()
	client := New(server.URL)
	_, err := client.GetModels()
	assert.Error(t, err)
}

func TestClientContentTypeHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()
	client := New(server.URL)
	_, err := client.CreateModel(map[string]interface{}{})
	assert.NoError(t, err)
}

func TestClientAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string][]map[string]interface{}{"models": {}})
	}))
	defer server.Close()
	client := New(server.URL)
	client.SetToken("test-token")
	_, err := client.GetModels()
	assert.NoError(t, err)
}
