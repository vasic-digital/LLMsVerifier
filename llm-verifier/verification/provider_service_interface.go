package verification

// ProviderServiceInterface defines the interface for provider services
type ProviderServiceInterface interface {
	GetAllProviders() map[string]ProviderClientInfo
	GetModels(providerID string) ([]ModelInfo, error)
}

// ProviderClientInfo contains basic provider client information
type ProviderClientInfo struct {
	ProviderID string
	BaseURL    string
	APIKey     string
}

// ModelInfo contains basic model information
type ModelInfo struct {
	ID         string
	Name       string
	ProviderID string
	Features   map[string]interface{}
}