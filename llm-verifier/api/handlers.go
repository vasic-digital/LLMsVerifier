package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"llm-verifier/config"
	"llm-verifier/database"
	"llm-verifier/llmverifier"
)

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// login handles user authentication
func (s *Server) login(c *gin.Context) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// TODO: Implement proper user authentication
	if credentials.Username == "" || credentials.Password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  1,
		"username": credentials.Username,
		"role":     "admin",
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      tokenString,
		"expires_in": 86400,
		"user": gin.H{
			"id":       1,
			"username": credentials.Username,
			"role":     "admin",
		},
	})
}

// refreshToken handles token refresh
func (s *Server) refreshToken(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Token refresh not implemented"})
}

// getModels retrieves all models
func (s *Server) getModels(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	filters := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	models, err := s.database.ListModels(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"models": models,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// createModel creates a new model
func (s *Server) createModel(c *gin.Context) {
	var model database.Model
	if err := c.ShouldBindJSON(&model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model data"})
		return
	}

	err := s.database.CreateModel(&model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create model"})
		return
	}

	c.JSON(http.StatusCreated, model)
}

// getModel retrieves a specific model by ID
func (s *Server) getModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	model, err := s.database.GetModel(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	c.JSON(http.StatusOK, model)
}

// verifyModel triggers verification for a specific model
func (s *Server) verifyModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	// Get model from database
	model, err := s.database.GetModel(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	// Start verification in background
	go func() {
		// Create LLM client from model's provider
		provider, err := s.database.GetProvider(model.ProviderID)
		if err != nil {
			// Log error and create event
			_ = s.database.CreateEvent(&database.Event{
				EventType: "verification_failed",
				Severity:  "error",
				Title:     "Verification Failed",
				Message:   fmt.Sprintf("Failed to get provider for model %s: %v", model.Name, err),
				ModelID:   &id,
				CreatedAt: time.Now(),
			})
			return
		}

		// Create temporary config for single model verification
		tempConfig := &config.Config{
			LLMs: []config.LLMConfig{
				{
					Name:     model.Name,
					Endpoint: provider.Endpoint,
					APIKey:   provider.APIKeyEncrypted,
					Model:    model.ModelID,
				},
			},
			Concurrency: 1,
			Timeout:     30 * time.Second,
		}

		// Create temporary verifier for this model
		tempVerifier := llmverifier.New(tempConfig)
		results, err := tempVerifier.Verify()
		if err != nil {
			// Log error and create event
			_ = s.database.CreateEvent(&database.Event{
				EventType: "verification_failed",
				Severity:  "error",
				Title:     "Verification Failed",
				Message:   fmt.Sprintf("Verification failed for model %s: %v", model.Name, err),
				ModelID:   &id,
				CreatedAt: time.Now(),
			})
			return
		}

		// Convert first result to database format
		if len(results) > 0 {
			result := results[0] // Take first result

			// Save verification result to database
			err = s.database.CreateVerificationResult(&database.VerificationResult{
				ModelID:                  id,
				VerificationType:         "comprehensive",
				StartedAt:                result.Timestamp,
				CompletedAt:              &result.Timestamp,
				Status:                   "completed",
				ModelExists:              &result.Availability.Exists,
				Responsive:               &result.Availability.Responsive,
				Overloaded:               &result.Availability.Overloaded,
				LatencyMs:                func() *int { v := int(result.Availability.Latency.Milliseconds()); return &v }(),
				SupportsToolUse:          result.FeatureDetection.ToolUse,
				SupportsFunctionCalling:  result.FeatureDetection.FunctionCalling,
				SupportsCodeGeneration:   result.FeatureDetection.CodeGeneration,
				SupportsCodeCompletion:   result.FeatureDetection.CodeCompletion,
				SupportsCodeReview:       result.FeatureDetection.CodeReview,
				SupportsCodeExplanation:  result.FeatureDetection.CodeExplanation,
				SupportsEmbeddings:       result.FeatureDetection.Embeddings,
				SupportsReranking:        result.FeatureDetection.Reranking,
				SupportsImageGeneration:  result.FeatureDetection.ImageGeneration,
				SupportsAudioGeneration:  result.FeatureDetection.AudioGeneration,
				SupportsVideoGeneration:  result.FeatureDetection.VideoGeneration,
				SupportsMCPs:             result.FeatureDetection.MCPs,
				SupportsLSPs:             result.FeatureDetection.LSPs,
				SupportsMultimodal:       result.FeatureDetection.Multimodal,
				SupportsStreaming:        result.FeatureDetection.Streaming,
				SupportsJSONMode:         result.FeatureDetection.JSONMode,
				SupportsStructuredOutput: result.FeatureDetection.StructuredOutput,
				SupportsReasoning:        result.FeatureDetection.Reasoning,
				SupportsParallelToolUse:  result.FeatureDetection.ParallelToolUse,
				MaxParallelCalls:         result.FeatureDetection.MaxParallelCalls,
				SupportsBatchProcessing:  result.FeatureDetection.BatchProcessing,
				CodeLanguageSupport:      result.CodeCapabilities.LanguageSupport,
				CodeDebugging:            result.CodeCapabilities.CodeDebugging,
				CodeOptimization:         result.CodeCapabilities.CodeOptimization,
				TestGeneration:           result.CodeCapabilities.TestGeneration,
				DocumentationGeneration:  result.CodeCapabilities.Documentation,
				Refactoring:              result.CodeCapabilities.Refactoring,
				ErrorResolution:          result.CodeCapabilities.ErrorResolution,
				ArchitectureDesign:       result.CodeCapabilities.Architecture,
				SecurityAssessment:       result.CodeCapabilities.SecurityAssessment,
				PatternRecognition:       result.CodeCapabilities.PatternRecognition,
				DebuggingAccuracy:        result.CodeCapabilities.ComplexityHandling.CodeQuality,
				MaxHandledDepth:          result.CodeCapabilities.ComplexityHandling.MaxHandledDepth,
				CodeQualityScore:         result.CodeCapabilities.ComplexityHandling.CodeQuality,
				LogicCorrectnessScore:    result.CodeCapabilities.ComplexityHandling.LogicCorrectness,
				RuntimeEfficiencyScore:   result.CodeCapabilities.ComplexityHandling.RuntimeEfficiency,
				OverallScore:             result.PerformanceScores.OverallScore,
				CodeCapabilityScore:      result.PerformanceScores.CodeCapability,
				ResponsivenessScore:      result.PerformanceScores.Responsiveness,
				ReliabilityScore:         result.PerformanceScores.Reliability,
				FeatureRichnessScore:     result.PerformanceScores.FeatureRichness,
				ValuePropositionScore:    result.PerformanceScores.ValueProposition,
			})
			if err != nil {
				// Log error
				_ = s.database.CreateEvent(&database.Event{
					EventType: "verification_failed",
					Severity:  "error",
					Title:     "Database Error",
					Message:   fmt.Sprintf("Failed to save verification result for model %s: %v", model.Name, err),
					ModelID:   &id,
					CreatedAt: time.Now(),
				})
				return
			}

			// Create success event
			_ = s.database.CreateEvent(&database.Event{
				EventType: "verification_completed",
				Severity:  "info",
				Title:     "Verification Completed",
				Message:   fmt.Sprintf("Verification completed for model %s with score %.2f", model.Name, result.PerformanceScores.OverallScore),
				ModelID:   &id,
				CreatedAt: time.Now(),
			})
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Verification started",
		"model_id": id,
		"model":    model.Name,
	})
}

// updateModel updates an existing model
func (s *Server) updateModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	var model database.Model
	if err := c.ShouldBindJSON(&model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model data"})
		return
	}

	model.ID = id
	err = s.database.UpdateModel(&model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update model"})
		return
	}

	c.JSON(http.StatusOK, model)
}

// deleteModel deletes a specific model
func (s *Server) deleteModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	err = s.database.DeleteModel(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete model"})
		return
	}

	c.Status(http.StatusNoContent)
}

// getProviders retrieves all providers
func (s *Server) getProviders(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	filters := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	providers, err := s.database.ListProviders(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve providers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// getProvider retrieves a specific provider by ID
func (s *Server) getProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	provider, err := s.database.GetProvider(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// createProvider creates a new provider
func (s *Server) createProvider(c *gin.Context) {
	var provider database.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider data"})
		return
	}

	err := s.database.CreateProvider(&provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create provider"})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// updateProvider updates an existing provider
func (s *Server) updateProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	var provider database.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider data"})
		return
	}

	provider.ID = id
	err = s.database.UpdateProvider(&provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// deleteProvider deletes a specific provider
func (s *Server) deleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	err = s.database.DeleteProvider(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		return
	}

	c.Status(http.StatusNoContent)
}

// getVerificationResults retrieves verification results
func (s *Server) getVerificationResults(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	filters := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	results, err := s.database.ListVerificationResults(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve verification results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// createVerificationResult creates a new verification result
func (s *Server) createVerificationResult(c *gin.Context) {
	var verificationResult database.VerificationResult
	if err := c.ShouldBindJSON(&verificationResult); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification result data"})
		return
	}

	err := s.database.CreateVerificationResult(&verificationResult)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create verification result"})
		return
	}

	c.JSON(http.StatusCreated, verificationResult)
}

// getVerificationResult retrieves a specific verification result
func (s *Server) getVerificationResult(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification result ID"})
		return
	}

	result, err := s.database.GetVerificationResult(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Verification result not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// updateVerificationResult updates an existing verification result
func (s *Server) updateVerificationResult(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification result ID"})
		return
	}

	var verificationResult database.VerificationResult
	if err := c.ShouldBindJSON(&verificationResult); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification result data"})
		return
	}

	verificationResult.ID = id
	err = s.database.UpdateVerificationResult(&verificationResult)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update verification result"})
		return
	}

	c.JSON(http.StatusOK, verificationResult)
}

// deleteVerificationResult deletes a verification result
func (s *Server) deleteVerificationResult(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification result ID"})
		return
	}

	err = s.database.DeleteVerificationResult(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete verification result"})
		return
	}

	c.Status(http.StatusNoContent)
}

// generateReport generates a new report
func (s *Server) generateReport(c *gin.Context) {
	var request struct {
		ReportType string  `json:"report_type" binding:"required,oneof=summary detailed comparison"`
		ModelIDs   []int64 `json:"model_ids,omitempty"`
		StartDate  string  `json:"start_date,omitempty"`
		EndDate    string  `json:"end_date,omitempty"`
		Format     string  `json:"format" binding:"required,oneof=json html pdf"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Parse dates if provided
	var startTime, endTime time.Time
	var err error

	if request.StartDate != "" {
		startTime, err = time.Parse(time.RFC3339, request.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use RFC3339 format"})
			return
		}
	} else {
		startTime = time.Now().AddDate(0, -1, 0) // Default to last month
	}

	if request.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, request.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use RFC3339 format"})
			return
		}
	} else {
		endTime = time.Now()
	}

	// Generate report based on type
	report := gin.H{
		"report_id":    fmt.Sprintf("report_%d", time.Now().Unix()),
		"generated_at": time.Now().Format(time.RFC3339),
		"report_type":  request.ReportType,
		"format":       request.Format,
		"date_range":   gin.H{"start": startTime.Format(time.RFC3339), "end": endTime.Format(time.RFC3339)},
	}

	// Get models for the report
	var models []*database.Model
	if len(request.ModelIDs) > 0 {
		for _, modelID := range request.ModelIDs {
			model, err := s.database.GetModel(modelID)
			if err == nil {
				models = append(models, model)
			}
		}
	} else {
		// Get all models
		allModels, err := s.database.ListModels(map[string]interface{}{"limit": 1000})
		if err == nil {
			models = allModels
		}
	}

	// Get verification results for the date range
	filters := map[string]interface{}{
		"start_date": startTime,
		"end_date":   endTime,
		"limit":      1000,
	}

	verificationResults, err := s.database.ListVerificationResults(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve verification results: " + err.Error()})
		return
	}

	// Get issues
	issues, err := s.database.ListIssues(map[string]interface{}{"limit": 1000})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issues: " + err.Error()})
		return
	}

	// Build report data based on report type
	switch request.ReportType {
	case "summary":
		report["summary"] = gin.H{
			"total_models":              len(models),
			"total_verifications":       len(verificationResults),
			"total_issues":              len(issues),
			"open_issues":               countOpenIssues(issues),
			"average_model_score":       calculateAverageModelScore(models),
			"verification_success_rate": calculateVerificationSuccessRate(verificationResults),
		}
	case "detailed":
		report["detailed"] = gin.H{
			"models":               models,
			"verification_results": verificationResults,
			"issues":               issues,
		}
	case "comparison":
		// For comparison reports, we need at least 2 models
		if len(models) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Comparison report requires at least 2 models"})
			return
		}
		report["comparison"] = gin.H{
			"models_comparison": compareModels(models, verificationResults),
		}
	}

	// Store the report (in a real implementation, this would save to database or file system)
	reportID := fmt.Sprintf("report_%d", time.Now().Unix())

	c.JSON(http.StatusOK, gin.H{
		"message":      "Report generated successfully",
		"report_id":    reportID,
		"report":       report,
		"download_url": fmt.Sprintf("/api/v1/reports/%s/download", reportID),
	})
}

// Helper functions for report generation
func countOpenIssues(issues []*database.Issue) int {
	count := 0
	for _, issue := range issues {
		if issue.ResolvedAt == nil || issue.ResolvedAt.IsZero() {
			count++
		}
	}
	return count
}

func calculateAverageModelScore(models []*database.Model) float64 {
	if len(models) == 0 {
		return 0.0
	}

	total := 0.0
	count := 0
	for _, model := range models {
		if model.OverallScore > 0 {
			total += model.OverallScore
			count++
		}
	}

	if count == 0 {
		return 0.0
	}
	return total / float64(count)
}

func calculateVerificationSuccessRate(results []*database.VerificationResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	successful := 0
	for _, result := range results {
		if result.Status == "completed" && (result.ErrorMessage == nil || *result.ErrorMessage == "") {
			successful++
		}
	}

	return float64(successful) / float64(len(results)) * 100.0
}

func compareModels(models []*database.Model, results []*database.VerificationResult) []gin.H {
	var comparisons []gin.H

	// Group results by model
	resultsByModel := make(map[int64][]*database.VerificationResult)
	for _, result := range results {
		resultsByModel[result.ModelID] = append(resultsByModel[result.ModelID], result)
	}

	for _, model := range models {
		modelResults := resultsByModel[model.ID]

		// Calculate average scores for this model
		var avgOverallScore, avgLatency float64
		if len(modelResults) > 0 {
			totalScore := 0.0
			totalLatency := 0.0
			validResults := 0

			for _, result := range modelResults {
				if result.Status == "completed" {
					totalScore += result.OverallScore
					totalLatency += float64(result.AvgLatencyMs)
					validResults++
				}
			}

			if validResults > 0 {
				avgOverallScore = totalScore / float64(validResults)
				avgLatency = totalLatency / float64(validResults)
			}
		}

		comparisons = append(comparisons, gin.H{
			"model_id":               model.ID,
			"model_name":             model.Name,
			"provider_name":          getProviderNameForModel(*model),
			"overall_score":          model.OverallScore,
			"verification_count":     len(modelResults),
			"avg_verification_score": avgOverallScore,
			"avg_latency_ms":         avgLatency,
			"verification_status":    model.VerificationStatus,
		})
	}

	return comparisons
}

func getProviderNameForModel(model database.Model) string {
	// This would need to fetch provider name from database
	// For now, return empty string
	return ""
}

// downloadReport downloads a generated report
func (s *Server) downloadReport(c *gin.Context) {
	reportID := c.Param("id")
	format := c.Query("format")
	if format == "" {
		format = "json" // Default format
	}

	// In a real implementation, this would retrieve the report from storage
	// For now, we'll return a mock report based on the ID
	if !strings.HasPrefix(reportID, "report_") {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report " + reportID + " not found"})
		return
	}

	// Parse timestamp from report ID
	timestampStr := strings.TrimPrefix(reportID, "report_")
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid report ID format"})
		return
	}

	reportTime := time.Unix(timestamp, 0)

	// Create a mock report
	report := gin.H{
		"report_id":    reportID,
		"generated_at": reportTime.Format(time.RFC3339),
		"title":        "LLM Verification Report",
		"summary": gin.H{
			"total_models":        5,
			"verified_models":     3,
			"average_score":       8.2,
			"total_verifications": 42,
			"success_rate":        92.5,
			"open_issues":         2,
		},
		"models": []gin.H{
			{"id": 1, "name": "GPT-4", "provider": "OpenAI", "score": 9.5, "status": "verified"},
			{"id": 2, "name": "Claude-3", "provider": "Anthropic", "score": 9.2, "status": "verified"},
			{"id": 3, "name": "Gemini Pro", "provider": "Google", "score": 8.8, "status": "verified"},
			{"id": 4, "name": "Llama 3", "provider": "Meta", "score": 8.5, "status": "pending"},
			{"id": 5, "name": "Mistral Large", "provider": "Mistral AI", "score": 8.1, "status": "verified"},
		},
		"recommendations": []string{
			"Consider upgrading to GPT-4 for complex code generation tasks",
			"Use Claude-3 for documentation and explanation tasks",
			"Monitor Llama 3 for future improvements in verification status",
		},
	}

	switch strings.ToLower(format) {
	case "json":
		c.JSON(http.StatusOK, report)
	case "html":
		// Generate simple HTML report
		html := generateHTMLReport(report)
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	case "pdf":
		// In a real implementation, this would generate a PDF
		c.JSON(http.StatusNotImplemented, gin.H{"error": "PDF export not yet implemented"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported format: " + format})
	}
}

func generateHTMLReport(report gin.H) string {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>LLM Verification Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        .summary { background: #f5f5f5; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .summary-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 15px; }
        .summary-item { background: white; padding: 15px; border-radius: 3px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .summary-value { font-size: 24px; font-weight: bold; color: #007bff; }
        .summary-label { color: #666; font-size: 14px; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f8f9fa; }
        .status-verified { color: #28a745; }
        .status-pending { color: #ffc107; }
        .recommendations { background: #e7f3ff; padding: 20px; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <h1>LLM Verification Report</h1>
    <p><strong>Report ID:</strong> ` + report["report_id"].(string) + `</p>
    <p><strong>Generated:</strong> ` + report["generated_at"].(string) + `</p>
    
    <div class="summary">
        <h2>Summary</h2>
        <div class="summary-grid">`

	summary := report["summary"].(gin.H)
	html += fmt.Sprintf(`
            <div class="summary-item">
                <div class="summary-value">%d</div>
                <div class="summary-label">Total Models</div>
            </div>
            <div class="summary-item">
                <div class="summary-value">%.1f</div>
                <div class="summary-label">Average Score</div>
            </div>
            <div class="summary-item">
                <div class="summary-value">%.1f%%</div>
                <div class="summary-label">Success Rate</div>
            </div>
            <div class="summary-item">
                <div class="summary-value">%d</div>
                <div class="summary-label">Verified Models</div>
            </div>
            <div class="summary-item">
                <div class="summary-value">%d</div>
                <div class="summary-label">Total Verifications</div>
            </div>
            <div class="summary-item">
                <div class="summary-value">%d</div>
                <div class="summary-label">Open Issues</div>
            </div>`,
		summary["total_models"].(int),
		summary["average_score"].(float64),
		summary["success_rate"].(float64),
		summary["verified_models"].(int),
		summary["total_verifications"].(int),
		summary["open_issues"].(int))

	html += `
        </div>
    </div>
    
    <h2>Models</h2>
    <table>
        <tr>
            <th>Name</th>
            <th>Provider</th>
            <th>Score</th>
            <th>Status</th>
        </tr>`

	models := report["models"].([]gin.H)
	for _, model := range models {
		statusClass := "status-" + model["status"].(string)
		html += fmt.Sprintf(`
        <tr>
            <td>%s</td>
            <td>%s</td>
            <td>%.1f</td>
            <td class="%s">%s</td>
        </tr>`,
			model["name"].(string),
			model["provider"].(string),
			model["score"].(float64),
			statusClass,
			strings.ToUpper(model["status"].(string)))
	}

	html += `
    </table>
    
    <div class="recommendations">
        <h2>Recommendations</h2>
        <ul>`

	recommendations := report["recommendations"].([]string)
	for _, rec := range recommendations {
		html += fmt.Sprintf(`<li>%s</li>`, rec)
	}

	html += `
        </ul>
    </div>
</body>
</html>`

	return html
}

// getConfig retrieves current configuration
func (s *Server) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"concurrency": s.config.Concurrency,
		"timeout":     s.config.Timeout,
		"api": gin.H{
			"port":        s.config.API.Port,
			"rate_limit":  s.config.API.RateLimit,
			"enable_cors": s.config.API.EnableCORS,
		},
	})
}

// updateConfig updates system configuration
func (s *Server) updateConfig(c *gin.Context) {
	var updateData struct {
		Concurrency *int           `json:"concurrency,omitempty"`
		Timeout     *time.Duration `json:"timeout,omitempty"`
		API         *struct {
			Port       *string `json:"port,omitempty"`
			RateLimit  *int    `json:"rate_limit,omitempty"`
			EnableCORS *bool   `json:"enable_cors,omitempty"`
		} `json:"api,omitempty"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Update concurrency if provided
	if updateData.Concurrency != nil {
		if *updateData.Concurrency < 1 || *updateData.Concurrency > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Concurrency must be between 1 and 100"})
			return
		}
		s.config.Concurrency = *updateData.Concurrency
	}

	// Update timeout if provided
	if updateData.Timeout != nil {
		if *updateData.Timeout < time.Second || *updateData.Timeout > 10*time.Minute {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Timeout must be between 1 second and 10 minutes"})
			return
		}
		s.config.Timeout = *updateData.Timeout
	}

	// Update API settings if provided
	if updateData.API != nil {
		if updateData.API.Port != nil {
			s.config.API.Port = *updateData.API.Port
		}
		if updateData.API.RateLimit != nil {
			if *updateData.API.RateLimit < 1 || *updateData.API.RateLimit > 1000 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Rate limit must be between 1 and 1000 requests per minute"})
				return
			}
			s.config.API.RateLimit = *updateData.API.RateLimit
		}
		if updateData.API.EnableCORS != nil {
			s.config.API.EnableCORS = *updateData.API.EnableCORS
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
		"config": gin.H{
			"concurrency": s.config.Concurrency,
			"timeout":     s.config.Timeout,
			"api": gin.H{
				"port":        s.config.API.Port,
				"rate_limit":  s.config.API.RateLimit,
				"enable_cors": s.config.API.EnableCORS,
			},
		},
	})
}

// exportConfig exports configuration for external tools
func (s *Server) exportConfig(c *gin.Context) {
	format := c.Query("format")
	if format == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format parameter required"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration exported in " + format + " format",
		"format":  format,
	})
}

// getPricing retrieves pricing information with optional filtering
func (s *Server) getPricing(c *gin.Context) {
	modelIDStr := c.Query("model_id")

	if modelIDStr != "" {
		// If model_id is provided, get pricing for that specific model
		modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model_id parameter"})
			return
		}

		pricing, err := s.database.ListPricing(modelID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pricing"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"pricing":  pricing,
			"model_id": modelID,
		})
	} else {
		// TODO: Implement general pricing listing without model filter
		c.JSON(http.StatusNotImplemented, gin.H{"error": "General pricing listing not implemented"})
	}
}

// getPricingByID retrieves a specific pricing record by ID
func (s *Server) getPricingByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pricing ID"})
		return
	}

	pricing, err := s.database.GetPricing(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pricing not found"})
		return
	}

	c.JSON(http.StatusOK, pricing)
}

// createPricing creates a new pricing record
func (s *Server) createPricing(c *gin.Context) {
	var pricing database.Pricing
	if err := c.ShouldBindJSON(&pricing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pricing data"})
		return
	}

	err := s.database.CreatePricing(&pricing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pricing"})
		return
	}

	c.JSON(http.StatusCreated, pricing)
}

// updatePricing updates an existing pricing record
func (s *Server) updatePricing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pricing ID"})
		return
	}

	var pricing database.Pricing
	if err := c.ShouldBindJSON(&pricing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pricing data"})
		return
	}

	pricing.ID = id
	err = s.database.UpdatePricing(&pricing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pricing"})
		return
	}

	c.JSON(http.StatusOK, pricing)
}

// deletePricing deletes a pricing record
func (s *Server) deletePricing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pricing ID"})
		return
	}

	err = s.database.DeletePricing(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pricing"})
		return
	}

	c.Status(http.StatusNoContent)
}

// getLimits retrieves limits with optional filtering
func (s *Server) getLimits(c *gin.Context) {
	modelIDStr := c.Query("model_id")

	if modelIDStr != "" {
		// If model_id is provided, get limits for that specific model
		modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model_id parameter"})
			return
		}

		limits, err := s.database.GetLimitsForModel(modelID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve limits"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"limits":   limits,
			"model_id": modelID,
		})
	} else {
		// TODO: Implement general limits listing without model filter
		c.JSON(http.StatusNotImplemented, gin.H{"error": "General limits listing not implemented"})
	}
}

// getLimitByID retrieves a specific limit by ID
func (s *Server) getLimitByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit ID"})
		return
	}

	limit, err := s.database.GetLimit(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Limit not found"})
		return
	}

	c.JSON(http.StatusOK, limit)
}

// createLimit creates a new limit record
func (s *Server) createLimit(c *gin.Context) {
	var limit database.Limit
	if err := c.ShouldBindJSON(&limit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit data"})
		return
	}

	err := s.database.CreateLimit(&limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create limit"})
		return
	}

	c.JSON(http.StatusCreated, limit)
}

// updateLimit updates an existing limit record
func (s *Server) updateLimit(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit ID"})
		return
	}

	var limit database.Limit
	if err := c.ShouldBindJSON(&limit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit data"})
		return
	}

	limit.ID = id
	err = s.database.UpdateLimit(&limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update limit"})
		return
	}

	c.JSON(http.StatusOK, limit)
}

// deleteLimit deletes a limit record
func (s *Server) deleteLimit(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit ID"})
		return
	}

	err = s.database.DeleteLimit(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete limit"})
		return
	}

	c.Status(http.StatusNoContent)
}

// getIssues retrieves issues with optional filtering
func (s *Server) getIssues(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	modelIDStr := c.Query("model_id")
	severity := c.Query("severity")
	issueType := c.Query("issue_type")
	resolvedStr := c.Query("resolved")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	filters := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	if modelIDStr != "" {
		modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model_id parameter"})
			return
		}
		filters["model_id"] = modelID
	}

	if severity != "" {
		filters["severity"] = severity
	}

	if issueType != "" {
		filters["issue_type"] = issueType
	}

	if resolvedStr != "" {
		resolved, err := strconv.ParseBool(resolvedStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resolved parameter"})
			return
		}
		filters["resolved"] = resolved
	}

	issues, err := s.database.ListIssues(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issues"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"issues": issues,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// getIssueByID retrieves a specific issue by ID
func (s *Server) getIssueByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	issue, err := s.database.GetIssue(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		return
	}

	c.JSON(http.StatusOK, issue)
}

// createIssue creates a new issue
func (s *Server) createIssue(c *gin.Context) {
	var issue database.Issue
	if err := c.ShouldBindJSON(&issue); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue data"})
		return
	}

	err := s.database.CreateIssue(&issue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create issue"})
		return
	}

	c.JSON(http.StatusCreated, issue)
}

// updateIssue updates an existing issue
func (s *Server) updateIssue(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	var issue database.Issue
	if err := c.ShouldBindJSON(&issue); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue data"})
		return
	}

	issue.ID = id
	err = s.database.UpdateIssue(&issue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update issue"})
		return
	}

	c.JSON(http.StatusOK, issue)
}

// deleteIssue deletes an issue
func (s *Server) deleteIssue(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	err = s.database.DeleteIssue(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete issue"})
		return
	}

	c.Status(http.StatusNoContent)
}

// getEvents retrieves events with optional filtering
func (s *Server) getEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	eventType := c.Query("event_type")
	severity := c.Query("severity")
	modelIDStr := c.Query("model_id")
	providerIDStr := c.Query("provider_id")
	fromDateStr := c.Query("from_date")
	toDateStr := c.Query("to_date")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	filters := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	if eventType != "" {
		filters["event_type"] = eventType
	}

	if severity != "" {
		filters["severity"] = severity
	}

	if modelIDStr != "" {
		modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model_id parameter"})
			return
		}
		filters["model_id"] = modelID
	}

	if providerIDStr != "" {
		providerID, err := strconv.ParseInt(providerIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider_id parameter"})
			return
		}
		filters["provider_id"] = providerID
	}

	if fromDateStr != "" {
		filters["from_date"] = fromDateStr
	}

	if toDateStr != "" {
		filters["to_date"] = toDateStr
	}

	events, err := s.database.ListEvents(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// getEventByID retrieves a specific event by ID
func (s *Server) getEventByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	event, err := s.database.GetEvent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// createEvent creates a new event
func (s *Server) createEvent(c *gin.Context) {
	var event database.Event
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event data"})
		return
	}

	err := s.database.CreateEvent(&event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// updateEvent updates an existing event
func (s *Server) updateEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var event database.Event
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event data"})
		return
	}

	event.ID = id
	err = s.database.UpdateEvent(&event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// deleteEvent deletes an event
func (s *Server) deleteEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	err = s.database.DeleteEvent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event"})
		return
	}

	c.Status(http.StatusNoContent)
}

// getSchedules retrieves schedules with optional filtering
func (s *Server) getSchedules(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	scheduleType := c.Query("schedule_type")
	targetType := c.Query("target_type")
	targetIDStr := c.Query("target_id")
	isActiveStr := c.Query("is_active")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	filters := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	if scheduleType != "" {
		filters["schedule_type"] = scheduleType
	}

	if targetType != "" {
		filters["target_type"] = targetType
	}

	if targetIDStr != "" {
		targetID, err := strconv.ParseInt(targetIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target_id parameter"})
			return
		}
		filters["target_id"] = targetID
	}

	if isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid is_active parameter"})
			return
		}
		filters["is_active"] = isActive
	}

	schedules, err := s.database.ListSchedules(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schedules": schedules,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// getScheduleByID retrieves a specific schedule by ID
func (s *Server) getScheduleByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	schedule, err := s.database.GetSchedule(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// createSchedule creates a new schedule
func (s *Server) createSchedule(c *gin.Context) {
	var schedule database.Schedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule data"})
		return
	}

	err := s.database.CreateSchedule(&schedule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule"})
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

// updateSchedule updates an existing schedule
func (s *Server) updateSchedule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	var schedule database.Schedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule data"})
		return
	}

	schedule.ID = id
	err = s.database.UpdateSchedule(&schedule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// deleteSchedule deletes a schedule
func (s *Server) deleteSchedule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	err = s.database.DeleteSchedule(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete schedule"})
		return
	}

	c.Status(http.StatusNoContent)
}

// getConfigExports retrieves config exports with optional filtering
func (s *Server) getConfigExports(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	exportType := c.Query("export_type")
	isVerifiedStr := c.Query("is_verified")
	createdBy := c.Query("created_by")
	search := c.Query("search")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	filters := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	if exportType != "" {
		filters["export_type"] = exportType
	}

	if isVerifiedStr != "" {
		isVerified, err := strconv.ParseBool(isVerifiedStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid is_verified parameter"})
			return
		}
		filters["is_verified"] = isVerified
	}

	if createdBy != "" {
		filters["created_by"] = createdBy
	}

	if search != "" {
		filters["search"] = search
	}

	configExports, err := s.database.ListConfigExports(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve config exports"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"config_exports": configExports,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// getConfigExportByID retrieves a specific config export by ID
func (s *Server) getConfigExportByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config export ID"})
		return
	}

	configExport, err := s.database.GetConfigExport(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Config export not found"})
		return
	}

	c.JSON(http.StatusOK, configExport)
}

// createConfigExport creates a new config export
func (s *Server) createConfigExport(c *gin.Context) {
	var configExport database.ConfigExport
	if err := c.ShouldBindJSON(&configExport); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config export data"})
		return
	}

	err := s.database.CreateConfigExport(&configExport)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create config export"})
		return
	}

	c.JSON(http.StatusCreated, configExport)
}

// updateConfigExport updates an existing config export
func (s *Server) updateConfigExport(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config export ID"})
		return
	}

	var configExport database.ConfigExport
	if err := c.ShouldBindJSON(&configExport); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config export data"})
		return
	}

	configExport.ID = id
	err = s.database.UpdateConfigExport(&configExport)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config export"})
		return
	}

	c.JSON(http.StatusOK, configExport)
}

// deleteConfigExport deletes a config export
func (s *Server) deleteConfigExport(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config export ID"})
		return
	}

	err = s.database.DeleteConfigExport(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete config export"})
		return
	}

	c.Status(http.StatusNoContent)
}

// downloadConfigExport downloads a config export file
func (s *Server) downloadConfigExport(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config export ID"})
		return
	}

	configExport, err := s.database.GetConfigExport(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Config export not found"})
		return
	}

	// Increment download count
	err = s.database.IncrementDownloadCount(id)
	if err != nil {
		// Log error but don't fail the download
		// TODO: Add proper logging
	}

	// Set headers for file download
	c.Header("Content-Disposition", "attachment; filename="+configExport.Name+".json")
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", []byte(configExport.ConfigData))
}

// getLogs retrieves logs with optional filtering
func (s *Server) getLogs(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	level := c.Query("level")
	logger := c.Query("logger")
	requestID := c.Query("request_id")
	modelIDStr := c.Query("model_id")
	providerIDStr := c.Query("provider_id")
	verificationResultIDStr := c.Query("verification_result_id")
	fromDateStr := c.Query("from_date")
	toDateStr := c.Query("to_date")
	search := c.Query("search")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	filters := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	if level != "" {
		filters["level"] = level
	}

	if logger != "" {
		filters["logger"] = logger
	}

	if requestID != "" {
		filters["request_id"] = requestID
	}

	if modelIDStr != "" {
		modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model_id parameter"})
			return
		}
		filters["model_id"] = modelID
	}

	if providerIDStr != "" {
		providerID, err := strconv.ParseInt(providerIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider_id parameter"})
			return
		}
		filters["provider_id"] = providerID
	}

	if verificationResultIDStr != "" {
		verificationResultID, err := strconv.ParseInt(verificationResultIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification_result_id parameter"})
			return
		}
		filters["verification_result_id"] = verificationResultID
	}

	if fromDateStr != "" {
		filters["from_date"] = fromDateStr
	}

	if toDateStr != "" {
		filters["to_date"] = toDateStr
	}

	if search != "" {
		filters["search"] = search
	}

	logs, err := s.database.ListLogs(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// getLogByID retrieves a specific log entry by ID
func (s *Server) getLogByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log ID"})
		return
	}

	logEntry, err := s.database.GetLog(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Log entry not found"})
		return
	}

	c.JSON(http.StatusOK, logEntry)
}
