package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"llm-verifier/client"
	"llm-verifier/logging"
)

// EnhancedModelProviderService extends ModelProviderService with mandatory verification
type EnhancedModelProviderService struct {
	*ModelProviderService
	verificationService *ModelVerificationService
	logger              *logging.Logger
	verificationConfig  VerificationConfig
}

// NewEnhancedModelProviderService creates a new enhanced model provider service with verification
func NewEnhancedModelProviderService(configPath string, logger *logging.Logger, verificationConfig VerificationConfig) *EnhancedModelProviderService {
	// Create the base model provider service
	baseService := NewModelProviderService(configPath, logger)
	
	// Create HTTP client for verification service
	httpClient := client.NewHTTPClient(30 * time.Second)
	
	// Create the verification service
	verificationService := NewModelVerificationService(httpClient, logger, verificationConfig)
	
	return &EnhancedModelProviderService{
		ModelProviderService: baseService,
		verificationService:  verificationService,
		logger:               logger,
		verificationConfig:   verificationConfig,
	}
}

// GetModelsWithVerification retrieves models with mandatory verification
func (emps *EnhancedModelProviderService) GetModelsWithVerification(ctx context.Context, providerID string) ([]Model, error) {
	emps.logger.Info(fmt.Sprintf("Getting models with verification for provider: %s", providerID), nil)
	
	// First, get models using the standard 3-tier system
	models, err := emps.ModelProviderService.GetModels(providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get models for provider %s: %w", providerID, err)
	}
	
	if len(models) == 0 {
		return models, nil // No models to verify
	}
	
	emps.logger.Info(fmt.Sprintf("Found %d models for provider %s, starting mandatory verification", len(models), providerID), nil)
	
	// Get the provider client for verification
	providerClient, exists := emps.ModelProviderService.providerClients[providerID]
	if !exists {
		emps.logger.Warning(fmt.Sprintf("Provider client not found for %s, skipping verification", providerID), nil)
		return models, nil
	}
	
	// Perform mandatory verification for all models
	verificationResults := emps.verificationService.VerifyModels(ctx, models, map[string]*ProviderClient{
		providerID: providerClient,
	})
	
	// Filter models based on verification results
	verifiedModels := emps.filterVerifiedModels(models, verificationResults)
	
	emps.logger.Info(fmt.Sprintf("Verification complete for provider %s: %d/%d models verified", 
		providerID, len(verifiedModels), len(models)), nil)
	
	return verifiedModels, nil
}

// GetAllModelsWithVerification retrieves all models with mandatory verification
func (emps *EnhancedModelProviderService) GetAllModelsWithVerification(ctx context.Context) (map[string][]Model, error) {
	emps.logger.Info("Getting all models with mandatory verification", nil)
	
	// Get all models using the standard method
	allModels, err := emps.ModelProviderService.GetAllModels()
	if err != nil {
		return nil, fmt.Errorf("failed to get all models: %w", err)
	}
	
	if len(allModels) == 0 {
		return allModels, nil
	}
	
	emps.logger.Info(fmt.Sprintf("Found models for %d providers, starting mandatory verification", len(allModels)), nil)
	
	// Prepare all models for verification
	var allModelsList []Model
	providerModelsMap := make(map[string][]Model)
	
	for providerID, models := range allModels {
		providerModelsMap[providerID] = models
		allModelsList = append(allModelsList, models...)
	}
	
	// Perform verification for all models
	verificationResults := emps.verificationService.VerifyModels(ctx, allModelsList, emps.ModelProviderService.providerClients)
	
	// Filter models based on verification results
	verifiedModels := make(map[string][]Model)
	totalVerified := 0
	
	for providerID, models := range providerModelsMap {
		var verifiedProviderModels []Model
		for _, model := range models {
			verificationKey := fmt.Sprintf("%s:%s", providerID, model.ID)
			if result, exists := verificationResults[verificationKey]; exists {
				if result.VerificationStatus == "verified" && result.CanSeeCode {
					verifiedProviderModels = append(verifiedProviderModels, model)
					totalVerified++
				}
			}
		}
		if len(verifiedProviderModels) > 0 {
			verifiedModels[providerID] = verifiedProviderModels
		}
	}
	
	emps.logger.Info(fmt.Sprintf("Mandatory verification complete: %d verified models across %d providers", 
		totalVerified, len(verifiedModels)), nil)
	
	return verifiedModels, nil
}

// QuickVerifyModels performs a quick verification check for models
func (emps *EnhancedModelProviderService) QuickVerifyModels(ctx context.Context, models []Model) ([]Model, error) {
	if len(models) == 0 {
		return models, nil
	}
	
	emps.logger.Info(fmt.Sprintf("Performing quick verification for %d models", len(models)), nil)
	
	// Group models by provider
	modelsByProvider := make(map[string][]Model)
	for _, model := range models {
		modelsByProvider[model.ProviderID] = append(modelsByProvider[model.ProviderID], model)
	}
	
	var verifiedModels []Model
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for providerID, providerModels := range modelsByProvider {
		providerClient, exists := emps.ModelProviderService.providerClients[providerID]
		if !exists {
			emps.logger.Warning(fmt.Sprintf("Provider client not found for %s, skipping verification", providerID), nil)
			continue
		}
		
		wg.Add(1)
		go func(pid string, pmodels []Model, pclient *ProviderClient) {
			defer wg.Done()
			
			// Perform quick verification for this provider's models
			results := emps.verificationService.VerifyModels(ctx, pmodels, map[string]*ProviderClient{pid: pclient})
			
			// Collect verified models
			for _, model := range pmodels {
				verificationKey := fmt.Sprintf("%s:%s", pid, model.ID)
				if result, exists := results[verificationKey]; exists {
					if result.VerificationStatus == "verified" && result.CanSeeCode {
						mu.Lock()
						verifiedModels = append(verifiedModels, model)
						mu.Unlock()
					}
				}
			}
		}(providerID, providerModels, providerClient)
	}
	
	wg.Wait()
	
	emps.logger.Info(fmt.Sprintf("Quick verification complete: %d/%d models verified", 
		len(verifiedModels), len(models)), nil)
	
	return verifiedModels, nil
}

// GetVerificationResults returns all verification results
func (emps *EnhancedModelProviderService) GetVerificationResults() map[string]*ModelVerificationResult {
	return emps.verificationService.GetAllVerificationResults()
}

// GetModelVerificationResult returns verification result for a specific model
func (emps *EnhancedModelProviderService) GetModelVerificationResult(providerID, modelID string) *ModelVerificationResult {
	return emps.verificationService.GetVerificationResult(providerID, modelID)
}

// IsModelVerified checks if a model is verified
func (emps *EnhancedModelProviderService) IsModelVerified(providerID, modelID string) bool {
	return emps.verificationService.IsModelVerified(providerID, modelID)
}

// EnableVerification enables or disables verification
func (emps *EnhancedModelProviderService) EnableVerification(enabled bool) {
	emps.verificationService.EnableVerification(enabled)
}

// SetStrictMode sets strict mode for verification
func (emps *EnhancedModelProviderService) SetStrictMode(strict bool) {
	emps.verificationService.SetStrictMode(strict)
}

// ClearVerificationResults clears all verification results
func (emps *EnhancedModelProviderService) ClearVerificationResults() {
	emps.verificationService.ClearVerificationResults()
}

// GetVerificationService returns the underlying verification service
func (emps *EnhancedModelProviderService) GetVerificationService() *ModelVerificationService {
	return emps.verificationService
}

// Helper method to filter verified models
func (emps *EnhancedModelProviderService) filterVerifiedModels(models []Model, verificationResults map[string]*ModelVerificationResult) []Model {
	if !emps.verificationConfig.Enabled {
		return models
	}
	
	var verifiedModels []Model
	
	for _, model := range models {
		verificationKey := fmt.Sprintf("%s:%s", model.ProviderID, model.ID)
		result, exists := verificationResults[verificationKey]
		
		if exists && result.VerificationStatus == "verified" && result.CanSeeCode {
			verifiedModels = append(verifiedModels, model)
		} else if !exists {
			emps.logger.Debug(fmt.Sprintf("No verification result for model %s, excluding", model.ID), nil)
		} else {
			emps.logger.Debug(fmt.Sprintf("Model %s failed verification: status=%s, can_see_code=%t", 
				model.ID, result.VerificationStatus, result.CanSeeCode), nil)
		}
	}
	
	return verifiedModels
}

// CreateDefaultVerificationConfig creates a default verification configuration
func CreateDefaultVerificationConfig() VerificationConfig {
	return VerificationConfig{
		Enabled:               true,
		StrictMode:            true,
		MaxRetries:            3,
		TimeoutSeconds:        30,
		RequireAffirmative:    true,
		MinVerificationScore:  0.7,
	}
}