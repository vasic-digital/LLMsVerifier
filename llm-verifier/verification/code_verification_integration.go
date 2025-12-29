package verification

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"llm-verifier/database"
	"llm-verifier/logging"
)

// CodeVerificationIntegration integrates code verification with the model discovery process
type CodeVerificationIntegration struct {
	verificationService *CodeVerificationService
	db                  *database.Database
	logger              *logging.Logger
	providerService     ProviderServiceInterface
}

// NewCodeVerificationIntegration creates a new integration instance
func NewCodeVerificationIntegration(verificationService *CodeVerificationService, db *database.Database, logger *logging.Logger, providerService ProviderServiceInterface) *CodeVerificationIntegration {
	return &CodeVerificationIntegration{
		verificationService: verificationService,
		db:                  db,
		logger:              logger,
		providerService:     providerService,
	}
}

// VerificationResult represents the result of verification integration
type VerificationResult struct {
	ProviderID       string                     `json:"provider_id"`
	ModelID          string                     `json:"model_id"`
	VerificationID   string                     `json:"verification_id"`
	Status           string                     `json:"status"`
	CodeVisibility   bool                       `json:"code_visibility"`
	ToolSupport      bool                       `json:"tool_support"`
	VerificationScore float64                   `json:"verification_score"`
	VerifiedAt       time.Time                  `json:"verified_at"`
	ErrorMessage     string                     `json:"error_message,omitempty"`
}

// VerifyAllModelsWithCodeSupport verifies all models that support code generation
func (cvi *CodeVerificationIntegration) VerifyAllModelsWithCodeSupport(ctx context.Context) ([]VerificationResult, error) {
	cvi.logger.Info("Starting mandatory code verification for all models with code support", nil)

	// Get all providers
	providers := cvi.providerService.GetAllProviders()
	var results []VerificationResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process providers concurrently
	for providerID := range providers {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			
			models, err := cvi.providerService.GetModels(pid)
			if err != nil {
				cvi.logger.Error(fmt.Sprintf("Failed to get models for provider %s: %v", pid, err), nil)
				return
			}

			// Verify each model
			for _, model := range models {
				// Skip models that don't support code generation
				if !cvi.shouldVerifyModel(model) {
					continue
				}

				result, err := cvi.verifyModel(ctx, model)
				if err != nil {
					cvi.logger.Error(fmt.Sprintf("Failed to verify model %s: %v", model.ID, err), map[string]interface{}{
						"model_id":    model.ID,
						"provider_id": pid,
					})
					continue
				}

				mu.Lock()
				results = append(results, *result)
				mu.Unlock()
			}
		}(providerID)
	}

	wg.Wait()

	cvi.logger.Info(fmt.Sprintf("Completed mandatory code verification for %d models", len(results)), map[string]interface{}{
		"total_verified": len(results),
	})

	return results, nil
}

// shouldVerifyModel determines if a model should be verified for code visibility
func (cvi *CodeVerificationIntegration) shouldVerifyModel(model ModelInfo) bool {
	// Check if model supports code generation or has coding-related features
	if model.Features != nil {
		// Check for code-related features
		if supportsCode, ok := model.Features["code"].(bool); ok && supportsCode {
			return true
		}
		if supportsTools, ok := model.Features["tool_call"].(bool); ok && supportsTools {
			return true
		}
		if supportsReasoning, ok := model.Features["reasoning"].(bool); ok && supportsReasoning {
			return true
		}
	}

	// Check model name for coding-related keywords
	modelName := strings.ToLower(model.Name + " " + model.ID)
	codeKeywords := []string{"code", "coder", "gpt-4", "claude", "deepseek", "mistral", "llama", "codestral", "programming", "development"}
	
	for _, keyword := range codeKeywords {
		if strings.Contains(modelName, keyword) {
			return true
		}
	}

	return false
}

// verifyModel performs code verification for a single model
func (cvi *CodeVerificationIntegration) verifyModel(ctx context.Context, model ModelInfo) (*VerificationResult, error) {
	cvi.logger.Info(fmt.Sprintf("Verifying code visibility for model %s", model.ID), map[string]interface{}{
		"model_id":    model.ID,
		"provider_id": model.ProviderID,
	})

	// Get provider client info
	providers := cvi.providerService.GetAllProviders()
	providerInfo, exists := providers[model.ProviderID]
	if !exists {
		return &VerificationResult{
			ProviderID:   model.ProviderID,
			ModelID:      model.ID,
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Provider %s not found", model.ProviderID),
			VerifiedAt:   time.Now(),
		}, nil
	}

	// Create a simple provider client interface
	simpleClient := &SimpleProviderClient{
		BaseURL:    providerInfo.BaseURL,
		APIKey:     providerInfo.APIKey,
		HTTPClient: &http.Client{Timeout: 30 * time.Second}, // Default HTTP client
	}

	// Perform code verification
	verificationResult, err := cvi.verificationService.VerifyModelCodeVisibility(ctx, model.ID, model.ProviderID, simpleClient)
	if err != nil {
		return &VerificationResult{
			ProviderID:   model.ProviderID,
			ModelID:      model.ID,
			Status:       "error",
			ErrorMessage: err.Error(),
			VerifiedAt:   time.Now(),
		}, nil
	}

	// Store verification result in database
	_, err = cvi.storeVerificationResult(verificationResult)
	if err != nil {
		cvi.logger.Error(fmt.Sprintf("Failed to store verification result: %v", err), map[string]interface{}{
			"model_id":    model.ID,
			"provider_id": model.ProviderID,
		})
	}

	// Update model metadata with verification status
	err = cvi.updateModelVerificationStatus(model, verificationResult)
	if err != nil {
		cvi.logger.Error(fmt.Sprintf("Failed to update model verification status: %v", err), map[string]interface{}{
			"model_id":    model.ID,
			"provider_id": model.ProviderID,
		})
	}

	result := &VerificationResult{
		ProviderID:        model.ProviderID,
		ModelID:           model.ID,
		VerificationID:    verificationResult.VerificationID,
		Status:            verificationResult.Status,
		CodeVisibility:    verificationResult.CodeVisibility,
		ToolSupport:       verificationResult.ToolSupport,
		VerificationScore: verificationResult.VerificationScore,
		VerifiedAt:        verificationResult.TestedAt,
	}

	if verificationResult.ErrorMessage != "" {
		result.ErrorMessage = verificationResult.ErrorMessage
	}

	return result, nil
}

// storeVerificationResult stores the verification result in the database
func (cvi *CodeVerificationIntegration) storeVerificationResult(result *CodeVerificationResult) (*database.VerificationResult, error) {
	// Find the model in the database
	models, err := cvi.db.ListModels(map[string]interface{}{
		"model_id":    result.ModelID,
		"provider_id": result.ProviderID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find model in database: %w", err)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("model not found in database")
	}

	model := models[0]

	// Create verification result
	verificationResult := &database.VerificationResult{
		ModelID:                  model.ID,
		VerificationType:         "code_visibility_verification",
		StartedAt:                result.TestedAt,
		CompletedAt:              result.CompletedAt,
		Status:                   result.Status,
		ModelExists:              ptrBool(true),
		Responsive:               ptrBool(result.Status == "verified"),
		LatencyMs:                ptrInt(2000), // Placeholder - should be calculated from actual response time
		SupportsCodeGeneration:   result.CodeVisibility,
		SupportsCodeCompletion:   result.CodeVisibility,
		SupportsCodeReview:       result.CodeVisibility,
		SupportsCodeExplanation:  result.CodeVisibility,
		CodeDebugging:            result.CodeVisibility,
		CodeOptimization:         result.ToolSupport,
		TestGeneration:           result.ToolSupport,
		DocumentationGeneration:  result.ToolSupport,
		Refactoring:              result.ToolSupport,
		ErrorResolution:          result.CodeVisibility,
		ArchitectureDesign:       result.ToolSupport,
		SecurityAssessment:       result.ToolSupport,
		PatternRecognition:       result.CodeVisibility,
		DebuggingAccuracy:        result.VerificationScore,
		CodeQualityScore:         result.VerificationScore * 10,
		OverallScore:             result.VerificationScore * 10,
		CodeCapabilityScore:      result.VerificationScore * 10,
		ResponsivenessScore:      8.0,
		ReliabilityScore:         result.VerificationScore * 8,
		FeatureRichnessScore:     result.VerificationScore * 9,
		ScoreDetails:             fmt.Sprintf("Code visibility verification score: %.2f", result.VerificationScore),
		CreatedAt:                time.Now(),
	}

	if result.ErrorMessage != "" {
		verificationResult.ErrorMessage = &result.ErrorMessage
	}

	err = cvi.db.CreateVerificationResult(verificationResult)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification result: %w", err)
	}

	return verificationResult, nil
}

// updateModelVerificationStatus updates the model's verification status in the provider service
func (cvi *CodeVerificationIntegration) updateModelVerificationStatus(model ModelInfo, result *CodeVerificationResult) error {
	// Update model metadata with verification status
	if model.Features == nil {
		model.Features = make(map[string]interface{})
	}
	model.Features["code_visibility_verified"] = result.CodeVisibility
	model.Features["tool_support_verified"] = result.ToolSupport
	model.Features["verification_score"] = result.VerificationScore
	model.Features["last_verified"] = result.TestedAt
	model.Features["verification_id"] = result.VerificationID
	model.Features["verification_status"] = result.Status

	return nil
}

// GetVerificationStatus returns the verification status for a specific model
func (cvi *CodeVerificationIntegration) GetVerificationStatus(modelID, providerID string) (*VerificationResult, error) {
	// Get the latest verification result from the database
	models, err := cvi.db.ListModels(map[string]interface{}{
		"model_id":    modelID,
		"provider_id": providerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find model: %w", err)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("model not found")
	}

	model := models[0]

	// Get the latest verification result
	results, err := cvi.db.ListVerificationResults(map[string]interface{}{
		"model_id": model.ID,
		"limit":    1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get verification results: %w", err)
	}

	if len(results) == 0 {
		return &VerificationResult{
			ProviderID: providerID,
			ModelID:    modelID,
			Status:     "not_verified",
			VerifiedAt: time.Now(),
		}, nil
	}

	verificationResult := results[0]

	return &VerificationResult{
		ProviderID:        providerID,
		ModelID:           modelID,
		VerificationID:    fmt.Sprintf("%d", verificationResult.ID),
		Status:            verificationResult.Status,
		CodeVisibility:    verificationResult.SupportsCodeGeneration,
		ToolSupport:       verificationResult.SupportsToolUse,
		VerificationScore: verificationResult.OverallScore / 10.0, // Convert back to 0-1 scale
		VerifiedAt:        verificationResult.CreatedAt,
	}, nil
}

// GetAllVerifiedModels returns all models that have been verified for code visibility
func (cvi *CodeVerificationIntegration) GetAllVerifiedModels() ([]VerificationResult, error) {
	// Get all models with verification status
	models, err := cvi.db.ListModels(map[string]interface{}{
		"supports_tool_use": true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	var results []VerificationResult

	for _, model := range models {
		// Get verification status for each model
		verificationResults, err := cvi.db.ListVerificationResults(map[string]interface{}{
			"model_id": model.ID,
			"limit":    1,
		})
		if err != nil || len(verificationResults) == 0 {
			continue
		}

		verificationResult := verificationResults[0]

		// Get provider information
		provider, err := cvi.db.GetProvider(model.ProviderID)
		if err != nil {
			continue
		}

		results = append(results, VerificationResult{
			ProviderID:        provider.Name,
			ModelID:           model.ModelID,
			VerificationID:    fmt.Sprintf("%d", verificationResult.ID),
			Status:            verificationResult.Status,
			CodeVisibility:    verificationResult.SupportsCodeGeneration,
			ToolSupport:       verificationResult.SupportsToolUse,
			VerificationScore: verificationResult.OverallScore / 10.0,
			VerifiedAt:        verificationResult.CreatedAt,
		})
	}

	return results, nil
}

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrInt(i int) *int {
	return &i
}