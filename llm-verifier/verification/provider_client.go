package verification

import (
	"net/http"
)

// ProviderClientInterface defines the interface for provider clients
type ProviderClientInterface interface {
	GetBaseURL() string
	GetAPIKey() string
	GetHTTPClient() *http.Client
}

// SimpleProviderClient is a simplified provider client for verification
type SimpleProviderClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func (c *SimpleProviderClient) GetBaseURL() string {
	return c.BaseURL
}

func (c *SimpleProviderClient) GetAPIKey() string {
	return c.APIKey
}

func (c *SimpleProviderClient) GetHTTPClient() *http.Client {
	return c.HTTPClient
}