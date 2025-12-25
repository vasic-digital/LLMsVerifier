package providers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGroqAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.groq.com/openai/v1"
	apiKey := "test-key"

	adapter := NewGroqAdapter(client, endpoint, apiKey)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
	assert.Equal(t, endpoint, adapter.endpoint)
	assert.Equal(t, apiKey, adapter.apiKey)
	assert.Contains(t, adapter.headers, "Authorization")
	assert.Equal(t, "Bearer test-key", adapter.headers["Authorization"])
}

func TestGroqAdapterBaseAdapter(t *testing.T) {
	client := &http.Client{}
	endpoint := "https://api.groq.com/openai/v1"
	apiKey := "test-key"

	adapter := NewGroqAdapter(client, endpoint, apiKey)

	// Test base adapter methods
	assert.NotNil(t, adapter.GetClient())
	assert.Equal(t, endpoint, adapter.GetEndpoint())
	assert.Equal(t, apiKey, adapter.GetAPIKey())
	assert.NotNil(t, adapter.GetHeaders())
}
