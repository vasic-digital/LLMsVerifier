package providers

import "llm-verifier/verification"

// ProviderServiceAdapter adapts ModelProviderService to ProviderServiceInterface
type ProviderServiceAdapter struct {
	service *ModelProviderService
}

// NewProviderServiceAdapter creates a new adapter
func NewProviderServiceAdapter(service *ModelProviderService) verification.ProviderServiceInterface {
	return &ProviderServiceAdapter{service: service}
}

// GetAllProviders returns all providers as ProviderClientInfo
func (a *ProviderServiceAdapter) GetAllProviders() map[string]verification.ProviderClientInfo {
	providers := a.service.GetAllProviders()
	result := make(map[string]verification.ProviderClientInfo)
	
	for id, client := range providers {
		result[id] = verification.ProviderClientInfo{
			ProviderID: id,
			BaseURL:    client.BaseURL,
			APIKey:     client.APIKey,
		}
	}
	
	return result
}

// GetModels returns models for a provider as ModelInfo
func (a *ProviderServiceAdapter) GetModels(providerID string) ([]verification.ModelInfo, error) {
	models, err := a.service.GetModels(providerID)
	if err != nil {
		return nil, err
	}
	
	result := make([]verification.ModelInfo, len(models))
	for i, model := range models {
		result[i] = verification.ModelInfo{
			ID:         model.ID,
			Name:       model.Name,
			ProviderID: model.ProviderID,
			Features:   model.Features,
		}
	}
	
	return result, nil
}