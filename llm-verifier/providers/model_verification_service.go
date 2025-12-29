package providers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"llm-verifier/client"
	"llm-verifier/logging"
	"llm-verifier/verification"
)

// ModelVerificationService handles mandatory "Do you see my code?" verification for all models
type ModelVerificationService struct {
	httpClient           *client.HTTPClient
	logger               *logging.Logger
	codeVerificationSvc  *verification.CodeVerificationService
	verificationResults  map[string]*ModelVerificationResult
	resultsMutex         sync.RWMutex
	verificationEnabled  bool
	strictMode           bool // If true, only verified models are usable
}

// ModelVerificationResult represents the result of mandatory model verification
type ModelVerificationResult struct {
	ModelID               string    `json:"model_id"`
	ProviderID            string    `json:"provider_id"`
	VerificationStatus    string    `json:"verification_status"` // "verified", "failed", "pending", "error"
	CanSeeCode            bool      `json:"can_see_code"`
	AffirmativeResponse   bool      `json:"affirmative_response"`
	VerificationScore     float64   `json:"verification_score"`
	LastVerifiedAt        time.Time `json:"last_verified_at"`
	ErrorMessage          string    `json:"error_message,omitempty"`
	VerificationAttempts  int       `json:"verification_attempts"`
}

// VerificationConfig configures the model verification service
type VerificationConfig struct {
	Enabled               bool
	StrictMode            bool
	MaxRetries            int
	TimeoutSeconds        int
	RequireAffirmative    bool
	MinVerificationScore  float64
}

// NewModelVerificationService creates a new model verification service
func NewModelVerificationService(httpClient *client.HTTPClient, logger *logging.Logger, config VerificationConfig) *ModelVerificationService {
	codeVerificationSvc := verification.NewCodeVerificationService(httpClient, logger)
	
	return &ModelVerificationService{
		httpClient:           httpClient,
		logger:               logger,
		codeVerificationSvc:  codeVerificationSvc,
		verificationResults:  make(map[string]*ModelVerificationResult),
		verificationEnabled:  config.Enabled,
		strictMode:           config.StrictMode,
	}
}

// VerifyModel performs mandatory verification for a single model
func (mvs *ModelVerificationService) VerifyModel(ctx context.Context, model Model, providerClient *ProviderClient) (*ModelVerificationResult, error) {
	if !mvs.verificationEnabled {
		mvs.logger.Info(fmt.Sprintf("Model verification disabled, skipping verification for %s", model.ID), nil)
		return &ModelVerificationResult{
			ModelID:            model.ID,
			ProviderID:         model.ProviderID,
			VerificationStatus: "skipped",
			CanSeeCode:         true, // Assume true when verification is disabled
			AffirmativeResponse: true,
			VerificationScore:  1.0,
			LastVerifiedAt:     time.Now(),
		}, nil
	}

	verificationKey := fmt.Sprintf("%s:%s", model.ProviderID, model.ID)
	
	mvs.logger.Info(fmt.Sprintf("Starting mandatory verification for model %s from provider %s", model.ID, model.ProviderID), map[string]interface{}{
		"model_id":     model.ID,
		"provider_id":  model.ProviderID,
		"verification_key": verificationKey,
	})

	// Create provider client interface for verification
	providerClientInterface := &verificationProviderClient{
		baseURL:    providerClient.BaseURL,
		apiKey:     providerClient.APIKey,
		httpClient: providerClient.HTTPClient,
	}

	// Perform code visibility verification
	codeResult, err := mvs.codeVerificationSvc.VerifyModelCodeVisibility(
		ctx,
		model.ID,
		model.ProviderID,
		providerClientInterface,
	)

	result := &ModelVerificationResult{
		ModelID:              model.ID,
		ProviderID:           model.ProviderID,
		LastVerifiedAt:       time.Now(),
		VerificationAttempts: 1,
	}

	if err != nil {
		mvs.logger.Error(fmt.Sprintf("Failed to verify model %s: %v", model.ID, err), map[string]interface{}{
			"model_id":     model.ID,
			"provider_id":  model.ProviderID,
			"error":        err.Error(),
		})
		result.VerificationStatus = "error"
		result.ErrorMessage = err.Error()
		result.CanSeeCode = false
		result.AffirmativeResponse = false
		result.VerificationScore = 0.0
	} else {
		// Process successful verification
		result.CanSeeCode = codeResult.CodeVisibility
		result.AffirmativeResponse = codeResult.AffirmativeConfirmation
		result.VerificationScore = codeResult.VerificationScore
		result.VerificationStatus = codeResult.Status
		
		if codeResult.ErrorMessage != "" {
			result.ErrorMessage = codeResult.ErrorMessage
		}
	}

	// Store the result
	mvs.storeVerificationResult(verificationKey, result)

	mvs.logger.Info(fmt.Sprintf("Mandatory verification completed for model %s: status=%s, can_see_code=%t, score=%.2f", 
		model.ID, result.VerificationStatus, result.CanSeeCode, result.VerificationScore), map[string]interface{}{
		"model_id":     model.ID,
		"provider_id":  model.ProviderID,
		"status":       result.VerificationStatus,
		"can_see_code": result.CanSeeCode,
		"score":        result.VerificationScore,
	})

	return result, nil
}

// VerifyModels performs mandatory verification for multiple models
func (mvs *ModelVerificationService) VerifyModels(ctx context.Context, models []Model, providerClients map[string]*ProviderClient) map[string]*ModelVerificationResult {
	results := make(map[string]*ModelVerificationResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	mvs.logger.Info(fmt.Sprintf("Starting mandatory verification for %d models", len(models)), nil)

	for _, model := range models {
		wg.Add(1)
		
		go func(m Model) {
			defer wg.Done()
			
			providerClient, exists := providerClients[m.ProviderID]
			if !exists {
				mvs.logger.Warning(fmt.Sprintf("No provider client found for %s", m.ProviderID), map[string]interface{}{
					"model_id":    m.ID,
					"provider_id": m.ProviderID,
				})
				
				result := &ModelVerificationResult{
					ModelID:            m.ID,
					ProviderID:         m.ProviderID,
					VerificationStatus: "error",
					ErrorMessage:       "Provider client not found",
					LastVerifiedAt:     time.Now(),
				}
				
				mu.Lock()
				results[fmt.Sprintf("%s:%s", m.ProviderID, m.ID)] = result
				mu.Unlock()
				return
			}
			
			result, err := mvs.VerifyModel(ctx, m, providerClient)
			if err != nil {
				mvs.logger.Error(fmt.Sprintf("Error verifying model %s: %v", m.ID, err), nil)
			}
			
			mu.Lock()
			results[fmt.Sprintf("%s:%s", m.ProviderID, m.ID)] = result
			mu.Unlock()
		}(model)
	}

	wg.Wait()
	
	mvs.logger.Info(fmt.Sprintf("Completed mandatory verification for %d models", len(results)), nil)
	
	return results
}

// IsModelVerified checks if a model has been verified and meets the verification criteria
func (mvs *ModelVerificationService) IsModelVerified(providerID, modelID string) bool {
	if !mvs.verificationEnabled {
		return true // If verification is disabled, consider all models verified
	}

	if mvs.strictMode {
		result := mvs.getVerificationResult(providerID, modelID)
		if result == nil {
			return false // No verification result means not verified
		}
		
		// In strict mode, model must be explicitly verified with affirmative response
		return result.VerificationStatus == "verified" && 
			   result.CanSeeCode && 
			   result.AffirmativeResponse &&
			   result.VerificationScore >= 0.7 // Minimum verification score
	}
	
	// In non-strict mode, any verification result is acceptable
	result := mvs.getVerificationResult(providerID, modelID)
	return result != nil && result.VerificationStatus != "error"
}

// GetVerificationResult gets the verification result for a specific model
func (mvs *ModelVerificationService) GetVerificationResult(providerID, modelID string) *ModelVerificationResult {
	return mvs.getVerificationResult(providerID, modelID)
}

// GetAllVerificationResults gets all verification results
func (mvs *ModelVerificationService) GetAllVerificationResults() map[string]*ModelVerificationResult {
	mvs.resultsMutex.RLock()
	defer mvs.resultsMutex.RUnlock()
	
	results := make(map[string]*ModelVerificationResult)
	for key, result := range mvs.verificationResults {
		results[key] = result
	}
	
	return results
}

// GetVerifiedModels filters models to only include verified ones
func (mvs *ModelVerificationService) GetVerifiedModels(models []Model) []Model {
	if !mvs.verificationEnabled {
		return models // Return all models if verification is disabled
	}

	var verifiedModels []Model
	
	for _, model := range models {
		if mvs.IsModelVerified(model.ProviderID, model.ID) {
			verifiedModels = append(verifiedModels, model)
		} else {
			mvs.logger.Debug(fmt.Sprintf("Model %s from %s is not verified, excluding from results", 
				model.ID, model.ProviderID), nil)
		}
	}
	
	mvs.logger.Info(fmt.Sprintf("Filtered %d models down to %d verified models", 
		len(models), len(verifiedModels)), nil)
	
	return verifiedModels
}

// EnableVerification enables or disables model verification
func (mvs *ModelVerificationService) EnableVerification(enabled bool) {
	mvs.verificationEnabled = enabled
	mvs.logger.Info(fmt.Sprintf("Model verification %s", map[bool]string{true: "enabled", false: "disabled"}[enabled]), nil)
}

// SetStrictMode sets strict mode for verification
func (mvs *ModelVerificationService) SetStrictMode(strict bool) {
	mvs.strictMode = strict
	mvs.logger.Info(fmt.Sprintf("Strict mode %s", map[bool]string{true: "enabled", false: "disabled"}[strict]), nil)
}

// ClearVerificationResults clears all verification results
func (mvs *ModelVerificationService) ClearVerificationResults() {
	mvs.resultsMutex.Lock()
	defer mvs.resultsMutex.Unlock()
	
	mvs.verificationResults = make(map[string]*ModelVerificationResult)
	mvs.logger.Info("Cleared all verification results", nil)
}

// Helper methods

func (mvs *ModelVerificationService) getVerificationResult(providerID, modelID string) *ModelVerificationResult {
	mvs.resultsMutex.RLock()
	defer mvs.resultsMutex.RUnlock()
	
	key := fmt.Sprintf("%s:%s", providerID, modelID)
	return mvs.verificationResults[key]
}

func (mvs *ModelVerificationService) storeVerificationResult(key string, result *ModelVerificationResult) {
	mvs.resultsMutex.Lock()
	defer mvs.resultsMutex.Unlock()
	
	mvs.verificationResults[key] = result
}

// verificationProviderClient adapts ProviderClient to ProviderClientInterface
type verificationProviderClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func (v *verificationProviderClient) GetBaseURL() string {
	return v.baseURL
}

func (v *verificationProviderClient) GetAPIKey() string {
	return v.apiKey
}

func (v *verificationProviderClient) GetHTTPClient() *http.Client {
	return v.httpClient
}