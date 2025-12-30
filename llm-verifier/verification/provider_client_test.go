package verification

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleProviderClient_GetBaseURL(t *testing.T) {
	client := &SimpleProviderClient{
		BaseURL: "https://api.example.com",
		APIKey:  "test-key",
	}

	assert.Equal(t, "https://api.example.com", client.GetBaseURL())
}

func TestSimpleProviderClient_GetAPIKey(t *testing.T) {
	client := &SimpleProviderClient{
		BaseURL: "https://api.example.com",
		APIKey:  "test-api-key-12345",
	}

	assert.Equal(t, "test-api-key-12345", client.GetAPIKey())
}

func TestSimpleProviderClient_GetHTTPClient(t *testing.T) {
	httpClient := &http.Client{}
	client := &SimpleProviderClient{
		BaseURL:    "https://api.example.com",
		APIKey:     "test-key",
		HTTPClient: httpClient,
	}

	assert.Equal(t, httpClient, client.GetHTTPClient())
}

func TestSimpleProviderClient_GetHTTPClient_Nil(t *testing.T) {
	client := &SimpleProviderClient{
		BaseURL: "https://api.example.com",
		APIKey:  "test-key",
	}

	assert.Nil(t, client.GetHTTPClient())
}

func TestSimpleProviderClient_ImplementsInterface(t *testing.T) {
	client := &SimpleProviderClient{
		BaseURL:    "https://api.example.com",
		APIKey:     "test-key",
		HTTPClient: &http.Client{},
	}

	var _ ProviderClientInterface = client
}

func TestSimpleProviderClient_EmptyValues(t *testing.T) {
	client := &SimpleProviderClient{}

	assert.Empty(t, client.GetBaseURL())
	assert.Empty(t, client.GetAPIKey())
	assert.Nil(t, client.GetHTTPClient())
}

func TestProviderServiceInterface_Types(t *testing.T) {
	// Test ProviderClientInfo struct
	info := ProviderClientInfo{
		ProviderID: "openai",
		BaseURL:    "https://api.openai.com",
		APIKey:     "sk-test-key",
	}

	assert.Equal(t, "openai", info.ProviderID)
	assert.Equal(t, "https://api.openai.com", info.BaseURL)
	assert.Equal(t, "sk-test-key", info.APIKey)
}

func TestModelInfo_Struct(t *testing.T) {
	features := map[string]interface{}{
		"max_tokens": 4096,
		"supports_vision": true,
	}

	info := ModelInfo{
		ID:         "gpt-4",
		Name:       "GPT-4",
		ProviderID: "openai",
		Features:   features,
	}

	assert.Equal(t, "gpt-4", info.ID)
	assert.Equal(t, "GPT-4", info.Name)
	assert.Equal(t, "openai", info.ProviderID)
	assert.Equal(t, 4096, info.Features["max_tokens"])
	assert.Equal(t, true, info.Features["supports_vision"])
}

func TestModelInfo_EmptyFeatures(t *testing.T) {
	info := ModelInfo{
		ID:         "test-model",
		Name:       "Test Model",
		ProviderID: "test-provider",
	}

	assert.Nil(t, info.Features)
}
