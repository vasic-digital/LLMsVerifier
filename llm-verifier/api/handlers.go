package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"

	"llm-verifier/config"
	"llm-verifier/database"
	"llm-verifier/events"
	"llm-verifier/llmverifier"
)

// healthCheck handles health check requests
// @Summary Health check endpoint
// @Description Returns the health status of the API
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{} "Health status"
// @Router /health [get]
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// login handles user authentication
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{} "Login response with JWT token"
// @Failure 400 {object} map[string]interface{} "Validation error"
// @Failure 401 {object} map[string]interface{} "Authentication failed"
// @Router /auth/login [post]
func (s *Server) login(c *gin.Context) {
	var credentials LoginRequest

	if err := c.ShouldBindJSON(&credentials); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Sanitize inputs
	username, err := ValidateAndSanitizeString(credentials.Username, 3, 50, false)
	if err != nil {
		SendError(c, http.StatusBadRequest, ErrCodeValidation, err.Error(), nil)
		return
	}

	// Validate password strength
	if err := ValidatePassword(credentials.Password); err != nil {
		SendError(c, http.StatusBadRequest, ErrCodeValidation, err.Error(), nil)
		return
	}

	// Get user from database
	user, err := s.database.GetUserByUsername(username)
	if err != nil {
		// User not found or database error
		log.Printf("Login failed: user '%s' not found or database error: %v", username, err)
		HandleUnauthorizedError(c, "Invalid credentials")
		return
	}

	// Check if user is active
	if !user.IsActive {
		log.Printf("Login failed: user '%s' account is disabled", username)
		HandleUnauthorizedError(c, "Account is disabled")
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(credentials.Password))
	if err != nil {
		log.Printf("Login failed: invalid password for user '%s'", username)
		HandleUnauthorizedError(c, "Invalid credentials")
		return
	}

	// Update last login time
	currentTime := time.Now()
	user.LastLogin = &currentTime
	if err := s.database.UpdateUser(user); err != nil {
		log.Printf("Failed to update last login for user '%s': %v", username, err)
	}

	log.Printf("User '%s' (ID: %d) logged in successfully", username, user.ID)

	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
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
	// Get the current token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}

	// Extract token from "Bearer <token>" format
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
		return
	}

	currentTokenString := tokenParts[1]

	// Parse and validate current token (even if expired, we need the claims)
	token, err := jwt.Parse(currentTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		// For any token parsing error, we can try to extract claims manually
		// This handles expired tokens as well
		if strings.Contains(err.Error(), "token is expired") ||
			strings.Contains(err.Error(), "expired") {
			// Token is expired, but we can still extract claims
			if token != nil {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					s.createNewTokenFromClaims(c, claims)
					return
				}
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Token is valid, extract claims and create new token
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		s.createNewTokenFromClaims(c, claims)
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
}

// createNewTokenFromClaims creates a new token from existing claims
func (s *Server) createNewTokenFromClaims(c *gin.Context, claims jwt.MapClaims) {
	// Extract user information from claims
	userID, ok := claims["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	username, ok := claims["username"].(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username in token"})
		return
	}

	role, ok := claims["role"].(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid role in token"})
		return
	}

	// Verify user still exists and is active
	user, err := s.database.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account is inactive"})
		return
	}

	// Create new JWT token with fresh expiration
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  int(userID),
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
		"iat":      time.Now().Unix(),
	})

	newTokenString, err := newToken.SignedString(s.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      newTokenString,
		"expires_in": 86400,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

// getModels retrieves all models
// @Summary Get models
// @Description Retrieves a list of all LLM models with pagination
// @Tags models
// @Produce json
// @Param limit query int false "Number of results per page (default: 50)"
// @Param offset query int false "Offset for pagination (default: 0)"
// @Param provider query string false "Filter by provider name"
// @Param status query string false "Filter by model status"
// @Success 200 {object} map[string]interface{} "List of models"
// @Failure 400 {object} map[string]interface{} "Invalid parameters"
// @Router /models [get]
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

	filters := map[string]any{
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
// @Summary Create model
// @Description Creates a new LLM model (admin only)
// @Tags models
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param model body CreateModelRequest true "Model data"
// @Success 201 {object} map[string]interface{} "Created model"
// @Failure 400 {object} map[string]interface{} "Validation error"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden (admin only)"
// @Router /models [post]
func (s *Server) createModel(c *gin.Context) {
	var request CreateModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Convert request to database model
	model := database.Model{
		ProviderID:            request.ProviderID,
		ModelID:               request.ModelID,
		Name:                  request.Name,
		Description:           request.Description,
		Version:               request.Version,
		Architecture:          request.Architecture,
		ParameterCount:        request.ParameterCount,
		ContextWindowTokens:   request.ContextWindowTokens,
		MaxOutputTokens:       request.MaxOutputTokens,
		TrainingDataCutoff:    request.TrainingDataCutoff,
		ReleaseDate:           request.ReleaseDate,
		IsMultimodal:          request.IsMultimodal,
		SupportsVision:        request.SupportsVision,
		SupportsAudio:         request.SupportsAudio,
		SupportsVideo:         request.SupportsVideo,
		SupportsReasoning:     request.SupportsReasoning,
		OpenSource:            request.OpenSource,
		Deprecated:            request.Deprecated,
		Tags:                  request.Tags,
		LanguageSupport:       request.LanguageSupport,
		UseCase:               request.UseCase,
		VerificationStatus:    request.VerificationStatus,
		OverallScore:          request.OverallScore,
		CodeCapabilityScore:   request.CodeCapabilityScore,
		ResponsivenessScore:   request.ResponsivenessScore,
		ReliabilityScore:      request.ReliabilityScore,
		FeatureRichnessScore:  request.FeatureRichnessScore,
		ValuePropositionScore: request.ValuePropositionScore,
	}

	err := s.database.CreateModel(&model)
	if err != nil {
		log.Printf("Failed to create model '%s': %v", request.Name, err)
		HandleDatabaseError(c, err)
		return
	}

	log.Printf("Model created: ID=%d, Name='%s', ProviderID=%d", model.ID, model.Name, model.ProviderID)
	SendSuccess(c, http.StatusCreated, model, "Model created successfully")
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

	// Publish verification started event
	s.eventBus.Publish(events.NewEvent(
		events.EventTypeVerificationStarted,
		events.EventSeverityInfo,
		fmt.Sprintf("Verification started for model %s", model.Name),
		"api",
	).WithData("model_id", id).WithData("model_name", model.Name))

	// Start verification in background
	go func() {
		// Create LLM client from model's provider
		provider, err := s.database.GetProvider(model.ProviderID)
		if err != nil {
			// Publish verification failed event
			s.eventBus.Publish(events.NewEvent(
				events.EventTypeModelVerificationFailed,
				events.EventSeverityError,
				fmt.Sprintf("Failed to get provider for model %s: %v", model.Name, err),
				"api",
			).WithData("model_id", id).WithData("model_name", model.Name).WithData("error", err.Error()))
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
			// Publish verification failed event
			s.eventBus.Publish(events.NewEvent(
				events.EventTypeModelVerificationFailed,
				events.EventSeverityError,
				fmt.Sprintf("Verification failed for model %s: %v", model.Name, err),
				"verifier",
			).WithData("model_id", id).WithData("model_name", model.Name).WithData("error", err.Error()))
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
				// Publish database error event
				s.eventBus.Publish(events.NewEvent(
					events.EventTypeErrorOccurred,
					events.EventSeverityError,
					fmt.Sprintf("Failed to save verification result for model %s: %v", model.Name, err),
					"database",
				).WithData("model_id", id).WithData("model_name", model.Name).WithData("error", err.Error()))
				return
			}

			// Publish verification completed event
			s.eventBus.Publish(events.NewEvent(
				events.EventTypeModelVerified,
				events.EventSeverityInfo,
				fmt.Sprintf("Verification completed for model %s with score %.2f", model.Name, result.PerformanceScores.OverallScore),
				"verifier",
			).WithData("model_id", id).WithData("model_name", model.Name).WithData("overall_score", result.PerformanceScores.OverallScore))
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
		SendError(c, http.StatusBadRequest, ErrCodeValidation, "Invalid model ID", nil)
		return
	}

	var request UpdateModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Get existing model to merge updates
	existingModel, err := s.database.GetModel(id)
	if err != nil {
		HandleNotFoundError(c, "Model", id)
		return
	}

	// Update fields if provided in request
	if request.ProviderID > 0 {
		existingModel.ProviderID = request.ProviderID
	}
	if request.ModelID != "" {
		existingModel.ModelID = request.ModelID
	}
	if request.Name != "" {
		existingModel.Name = request.Name
	}
	if request.Description != "" {
		existingModel.Description = request.Description
	}
	if request.Version != "" {
		existingModel.Version = request.Version
	}
	if request.Architecture != "" {
		existingModel.Architecture = request.Architecture
	}
	if request.ParameterCount != nil {
		existingModel.ParameterCount = request.ParameterCount
	}
	if request.ContextWindowTokens != nil {
		existingModel.ContextWindowTokens = request.ContextWindowTokens
	}
	if request.MaxOutputTokens != nil {
		existingModel.MaxOutputTokens = request.MaxOutputTokens
	}
	if request.TrainingDataCutoff != nil {
		existingModel.TrainingDataCutoff = request.TrainingDataCutoff
	}
	if request.ReleaseDate != nil {
		existingModel.ReleaseDate = request.ReleaseDate
	}
	if request.IsMultimodal != nil {
		existingModel.IsMultimodal = *request.IsMultimodal
	}
	if request.SupportsVision != nil {
		existingModel.SupportsVision = *request.SupportsVision
	}
	if request.SupportsAudio != nil {
		existingModel.SupportsAudio = *request.SupportsAudio
	}
	if request.SupportsVideo != nil {
		existingModel.SupportsVideo = *request.SupportsVideo
	}
	if request.SupportsReasoning != nil {
		existingModel.SupportsReasoning = *request.SupportsReasoning
	}
	if request.OpenSource != nil {
		existingModel.OpenSource = *request.OpenSource
	}
	if request.Deprecated != nil {
		existingModel.Deprecated = *request.Deprecated
	}
	if request.Tags != nil {
		existingModel.Tags = request.Tags
	}
	if request.LanguageSupport != nil {
		existingModel.LanguageSupport = request.LanguageSupport
	}
	if request.UseCase != "" {
		existingModel.UseCase = request.UseCase
	}
	if request.VerificationStatus != "" {
		existingModel.VerificationStatus = request.VerificationStatus
	}
	if request.OverallScore > 0 {
		existingModel.OverallScore = request.OverallScore
	}
	if request.CodeCapabilityScore > 0 {
		existingModel.CodeCapabilityScore = request.CodeCapabilityScore
	}
	if request.ResponsivenessScore > 0 {
		existingModel.ResponsivenessScore = request.ResponsivenessScore
	}
	if request.ReliabilityScore > 0 {
		existingModel.ReliabilityScore = request.ReliabilityScore
	}
	if request.FeatureRichnessScore > 0 {
		existingModel.FeatureRichnessScore = request.FeatureRichnessScore
	}
	if request.ValuePropositionScore > 0 {
		existingModel.ValuePropositionScore = request.ValuePropositionScore
	}

	err = s.database.UpdateModel(existingModel)
	if err != nil {
		HandleDatabaseError(c, err)
		return
	}

	SendSuccess(c, http.StatusOK, existingModel, "Model updated successfully")
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

	filters := map[string]any{
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
	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	provider := database.Provider{
		Name:                  req.Name,
		Endpoint:              req.Endpoint,
		APIKeyEncrypted:       req.APIKeyEncrypted,
		Description:           req.Description,
		Website:               req.Website,
		SupportEmail:          req.SupportEmail,
		DocumentationURL:      req.DocumentationURL,
		IsActive:              req.IsActive,
		ReliabilityScore:      req.ReliabilityScore,
		AverageResponseTimeMs: req.AverageResponseTimeMs,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	err := s.database.CreateProvider(&provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create provider"})
		return
	}

	// Publish provider added event
	s.eventBus.Publish(events.NewEvent(
		events.EventTypeProviderAdded,
		events.EventSeverityInfo,
		fmt.Sprintf("Provider %s added with endpoint %s", provider.Name, provider.Endpoint),
		"api",
	).WithData("provider_id", provider.ID).WithData("provider_name", provider.Name).WithData("endpoint", provider.Endpoint))

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

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Get existing provider to preserve fields not in request
	existingProvider, err := s.database.GetProvider(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Update fields from request
	if req.Name != "" {
		existingProvider.Name = req.Name
	}
	if req.Endpoint != "" {
		existingProvider.Endpoint = req.Endpoint
	}
	if req.APIKeyEncrypted != "" {
		existingProvider.APIKeyEncrypted = req.APIKeyEncrypted
	}
	if req.Description != "" {
		existingProvider.Description = req.Description
	}
	if req.Website != "" {
		existingProvider.Website = req.Website
	}
	if req.SupportEmail != "" {
		existingProvider.SupportEmail = req.SupportEmail
	}
	if req.DocumentationURL != "" {
		existingProvider.DocumentationURL = req.DocumentationURL
	}
	if req.IsActive != nil {
		existingProvider.IsActive = *req.IsActive
	}
	if req.ReliabilityScore != 0 {
		existingProvider.ReliabilityScore = req.ReliabilityScore
	}
	if req.AverageResponseTimeMs != 0 {
		existingProvider.AverageResponseTimeMs = req.AverageResponseTimeMs
	}
	existingProvider.UpdatedAt = time.Now()

	err = s.database.UpdateProvider(existingProvider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider"})
		return
	}

	// Publish provider updated event
	s.eventBus.Publish(events.NewEvent(
		events.EventTypeProviderUpdated,
		events.EventSeverityInfo,
		fmt.Sprintf("Provider %s updated", existingProvider.Name),
		"api",
	).WithData("provider_id", existingProvider.ID).WithData("provider_name", existingProvider.Name))

	c.JSON(http.StatusOK, existingProvider)
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

	filters := map[string]any{
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
	var req CreateVerificationResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	verificationResult := database.VerificationResult{
		ModelID:                  req.ModelID,
		VerificationType:         req.VerificationType,
		StartedAt:                req.StartedAt,
		CompletedAt:              req.CompletedAt,
		Status:                   req.Status,
		ErrorMessage:             req.ErrorMessage,
		ModelExists:              req.ModelExists,
		Responsive:               req.Responsive,
		Overloaded:               req.Overloaded,
		LatencyMs:                req.LatencyMs,
		SupportsToolUse:          req.SupportsToolUse,
		SupportsFunctionCalling:  req.SupportsFunctionCalling,
		SupportsCodeGeneration:   req.SupportsCodeGeneration,
		SupportsCodeCompletion:   req.SupportsCodeCompletion,
		SupportsCodeReview:       req.SupportsCodeReview,
		SupportsCodeExplanation:  req.SupportsCodeExplanation,
		SupportsEmbeddings:       req.SupportsEmbeddings,
		SupportsReranking:        req.SupportsReranking,
		SupportsImageGeneration:  req.SupportsImageGeneration,
		SupportsAudioGeneration:  req.SupportsAudioGeneration,
		SupportsVideoGeneration:  req.SupportsVideoGeneration,
		SupportsMCPs:             req.SupportsMCPs,
		SupportsLSPs:             req.SupportsLSPs,
		SupportsMultimodal:       req.SupportsMultimodal,
		SupportsStreaming:        req.SupportsStreaming,
		SupportsJSONMode:         req.SupportsJSONMode,
		SupportsStructuredOutput: req.SupportsStructuredOutput,
		SupportsReasoning:        req.SupportsReasoning,
		SupportsParallelToolUse:  req.SupportsParallelToolUse,
		MaxParallelCalls:         req.MaxParallelCalls,
		SupportsBatchProcessing:  req.SupportsBatchProcessing,
		CodeLanguageSupport:      req.CodeLanguageSupport,
		CodeDebugging:            req.CodeDebugging,
		CodeOptimization:         req.CodeOptimization,
		TestGeneration:           req.TestGeneration,
		DocumentationGeneration:  req.DocumentationGeneration,
		Refactoring:              req.Refactoring,
		ErrorResolution:          req.ErrorResolution,
		ArchitectureDesign:       req.ArchitectureDesign,
		SecurityAssessment:       req.SecurityAssessment,
		PatternRecognition:       req.PatternRecognition,
		DebuggingAccuracy:        req.DebuggingAccuracy,
		MaxHandledDepth:          req.MaxHandledDepth,
		CodeQualityScore:         req.CodeQualityScore,
		LogicCorrectnessScore:    req.LogicCorrectnessScore,
		RuntimeEfficiencyScore:   req.RuntimeEfficiencyScore,
		OverallScore:             req.OverallScore,
		CodeCapabilityScore:      req.CodeCapabilityScore,
		ResponsivenessScore:      req.ResponsivenessScore,
		ReliabilityScore:         req.ReliabilityScore,
		FeatureRichnessScore:     req.FeatureRichnessScore,
		ValuePropositionScore:    req.ValuePropositionScore,
		ScoreDetails:             req.ScoreDetails,
		AvgLatencyMs:             req.AvgLatencyMs,
		P95LatencyMs:             req.P95LatencyMs,
		MinLatencyMs:             req.MinLatencyMs,
		MaxLatencyMs:             req.MaxLatencyMs,
		ThroughputRPS:            req.ThroughputRPS,
		RawRequest:               req.RawRequest,
		RawResponse:              req.RawResponse,
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

	var req UpdateVerificationResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Get existing verification result to preserve fields not in request
	existingResult, err := s.database.GetVerificationResult(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Verification result not found"})
		return
	}

	// Update fields from request
	if req.ModelID != 0 {
		existingResult.ModelID = req.ModelID
	}
	if req.VerificationType != "" {
		existingResult.VerificationType = req.VerificationType
	}
	if !req.StartedAt.IsZero() {
		existingResult.StartedAt = req.StartedAt
	}
	if req.CompletedAt != nil {
		existingResult.CompletedAt = req.CompletedAt
	}
	if req.Status != "" {
		existingResult.Status = req.Status
	}
	if req.ErrorMessage != nil {
		existingResult.ErrorMessage = req.ErrorMessage
	}
	if req.ModelExists != nil {
		existingResult.ModelExists = req.ModelExists
	}
	if req.Responsive != nil {
		existingResult.Responsive = req.Responsive
	}
	if req.Overloaded != nil {
		existingResult.Overloaded = req.Overloaded
	}
	if req.LatencyMs != nil {
		existingResult.LatencyMs = req.LatencyMs
	}
	if req.SupportsToolUse != nil {
		existingResult.SupportsToolUse = *req.SupportsToolUse
	}
	if req.SupportsFunctionCalling != nil {
		existingResult.SupportsFunctionCalling = *req.SupportsFunctionCalling
	}
	if req.SupportsCodeGeneration != nil {
		existingResult.SupportsCodeGeneration = *req.SupportsCodeGeneration
	}
	if req.SupportsCodeCompletion != nil {
		existingResult.SupportsCodeCompletion = *req.SupportsCodeCompletion
	}
	if req.SupportsCodeReview != nil {
		existingResult.SupportsCodeReview = *req.SupportsCodeReview
	}
	if req.SupportsCodeExplanation != nil {
		existingResult.SupportsCodeExplanation = *req.SupportsCodeExplanation
	}
	if req.SupportsEmbeddings != nil {
		existingResult.SupportsEmbeddings = *req.SupportsEmbeddings
	}
	if req.SupportsReranking != nil {
		existingResult.SupportsReranking = *req.SupportsReranking
	}
	if req.SupportsImageGeneration != nil {
		existingResult.SupportsImageGeneration = *req.SupportsImageGeneration
	}
	if req.SupportsAudioGeneration != nil {
		existingResult.SupportsAudioGeneration = *req.SupportsAudioGeneration
	}
	if req.SupportsVideoGeneration != nil {
		existingResult.SupportsVideoGeneration = *req.SupportsVideoGeneration
	}
	if req.SupportsMCPs != nil {
		existingResult.SupportsMCPs = *req.SupportsMCPs
	}
	if req.SupportsLSPs != nil {
		existingResult.SupportsLSPs = *req.SupportsLSPs
	}
	if req.SupportsMultimodal != nil {
		existingResult.SupportsMultimodal = *req.SupportsMultimodal
	}
	if req.SupportsStreaming != nil {
		existingResult.SupportsStreaming = *req.SupportsStreaming
	}
	if req.SupportsJSONMode != nil {
		existingResult.SupportsJSONMode = *req.SupportsJSONMode
	}
	if req.SupportsStructuredOutput != nil {
		existingResult.SupportsStructuredOutput = *req.SupportsStructuredOutput
	}
	if req.SupportsReasoning != nil {
		existingResult.SupportsReasoning = *req.SupportsReasoning
	}
	if req.SupportsParallelToolUse != nil {
		existingResult.SupportsParallelToolUse = *req.SupportsParallelToolUse
	}
	if req.MaxParallelCalls != 0 {
		existingResult.MaxParallelCalls = req.MaxParallelCalls
	}
	if req.SupportsBatchProcessing != nil {
		existingResult.SupportsBatchProcessing = *req.SupportsBatchProcessing
	}
	if req.CodeLanguageSupport != nil {
		existingResult.CodeLanguageSupport = req.CodeLanguageSupport
	}
	if req.CodeDebugging != nil {
		existingResult.CodeDebugging = *req.CodeDebugging
	}
	if req.CodeOptimization != nil {
		existingResult.CodeOptimization = *req.CodeOptimization
	}
	if req.TestGeneration != nil {
		existingResult.TestGeneration = *req.TestGeneration
	}
	if req.DocumentationGeneration != nil {
		existingResult.DocumentationGeneration = *req.DocumentationGeneration
	}
	if req.Refactoring != nil {
		existingResult.Refactoring = *req.Refactoring
	}
	if req.ErrorResolution != nil {
		existingResult.ErrorResolution = *req.ErrorResolution
	}
	if req.ArchitectureDesign != nil {
		existingResult.ArchitectureDesign = *req.ArchitectureDesign
	}
	if req.SecurityAssessment != nil {
		existingResult.SecurityAssessment = *req.SecurityAssessment
	}
	if req.PatternRecognition != nil {
		existingResult.PatternRecognition = *req.PatternRecognition
	}
	if req.DebuggingAccuracy != 0 {
		existingResult.DebuggingAccuracy = req.DebuggingAccuracy
	}
	if req.MaxHandledDepth != 0 {
		existingResult.MaxHandledDepth = req.MaxHandledDepth
	}
	if req.CodeQualityScore != 0 {
		existingResult.CodeQualityScore = req.CodeQualityScore
	}
	if req.LogicCorrectnessScore != 0 {
		existingResult.LogicCorrectnessScore = req.LogicCorrectnessScore
	}
	if req.RuntimeEfficiencyScore != 0 {
		existingResult.RuntimeEfficiencyScore = req.RuntimeEfficiencyScore
	}
	if req.OverallScore != 0 {
		existingResult.OverallScore = req.OverallScore
	}
	if req.CodeCapabilityScore != 0 {
		existingResult.CodeCapabilityScore = req.CodeCapabilityScore
	}
	if req.ResponsivenessScore != 0 {
		existingResult.ResponsivenessScore = req.ResponsivenessScore
	}
	if req.ReliabilityScore != 0 {
		existingResult.ReliabilityScore = req.ReliabilityScore
	}
	if req.FeatureRichnessScore != 0 {
		existingResult.FeatureRichnessScore = req.FeatureRichnessScore
	}
	if req.ValuePropositionScore != 0 {
		existingResult.ValuePropositionScore = req.ValuePropositionScore
	}
	if req.ScoreDetails != "" {
		existingResult.ScoreDetails = req.ScoreDetails
	}
	if req.AvgLatencyMs != 0 {
		existingResult.AvgLatencyMs = req.AvgLatencyMs
	}
	if req.P95LatencyMs != 0 {
		existingResult.P95LatencyMs = req.P95LatencyMs
	}
	if req.MinLatencyMs != 0 {
		existingResult.MinLatencyMs = req.MinLatencyMs
	}
	if req.MaxLatencyMs != 0 {
		existingResult.MaxLatencyMs = req.MaxLatencyMs
	}
	if req.ThroughputRPS != 0 {
		existingResult.ThroughputRPS = req.ThroughputRPS
	}
	if req.RawRequest != nil {
		existingResult.RawRequest = req.RawRequest
	}
	if req.RawResponse != nil {
		existingResult.RawResponse = req.RawResponse
	}

	err = s.database.UpdateVerificationResult(existingResult)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update verification result"})
		return
	}

	c.JSON(http.StatusOK, existingResult)
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
	var request GenerateReportRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		HandleValidationError(c, err)
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
		allModels, err := s.database.ListModels(map[string]any{"limit": 1000})
		if err == nil {
			models = allModels
		}
	}

	// Get verification results for the date range
	filters := map[string]any{
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
	issues, err := s.database.ListIssues(map[string]any{"limit": 1000})
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
			"models_comparison": s.compareModels(models, verificationResults),
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

func (s *Server) compareModels(models []*database.Model, results []*database.VerificationResult) []gin.H {
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
			"provider_name":          s.getProviderNameForModel(*model),
			"overall_score":          model.OverallScore,
			"verification_count":     len(modelResults),
			"avg_verification_score": avgOverallScore,
			"avg_latency_ms":         avgLatency,
			"verification_status":    model.VerificationStatus,
		})
	}

	return comparisons
}

func (s *Server) getProviderNameForModel(model database.Model) string {
	// Look up provider by ID
	provider, err := s.database.GetProvider(model.ProviderID)
	if err != nil {
		// If lookup fails, return a fallback name
		return fmt.Sprintf("Provider-%d", model.ProviderID)
	}
	return provider.Name
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
	var request UpdateSystemConfigRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Update concurrency if provided
	if request.Concurrency != nil {
		s.config.Concurrency = *request.Concurrency
	}

	// Update timeout if provided
	if request.Timeout != nil {
		s.config.Timeout = *request.Timeout
	}

	// Update API settings if provided
	if request.API != nil {
		if request.API.Port != nil {
			s.config.API.Port = *request.API.Port
		}
		if request.API.RateLimit != nil {
			s.config.API.RateLimit = *request.API.RateLimit
		}
		if request.API.EnableCORS != nil {
			s.config.API.EnableCORS = *request.API.EnableCORS
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
		log.Printf("Export config failed: format parameter required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format parameter required. Supported formats: opencode, claude, crush, vscode, json, yaml"})
		return
	}

	// Check if multiple formats requested (comma-separated)
	formats := strings.Split(format, ",")
	if len(formats) > 1 {
		// Bulk export
		results := make(map[string]string)
		for _, f := range formats {
			f = strings.TrimSpace(f)
			exported, err := s.exportConfiguration(f)
			if err != nil {
				log.Printf("Export config failed for format '%s': %v", f, err)
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to export configuration for format '%s': %s", f, err.Error())})
				return
			}
			results[f] = exported
		}

		log.Printf("Bulk configuration export completed for formats: %v", formats)
		c.JSON(http.StatusOK, gin.H{
			"exports": results,
			"formats": formats,
		})
		return
	}

	// Single format export
	exported, err := s.exportConfiguration(format)
	if err != nil {
		log.Printf("Export config failed for format '%s': %v", format, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to export configuration: " + err.Error()})
		return
	}

	log.Printf("Configuration exported successfully in '%s' format", format)

	// Set appropriate content type based on format
	contentType := "application/json"
	filename := "config.json"

	switch format {
	case "opencode":
		contentType = "application/json"
		filename = "opencode-config.json"
	case "claude":
		contentType = "application/json"
		filename = "claude-config.json"
	case "vscode":
		contentType = "application/json"
		filename = "vscode-settings.json"
	case "yaml":
		contentType = "application/x-yaml"
		filename = "config.yaml"
	default: // json
		contentType = "application/json"
		filename = "config.json"
	}

	// Set headers for file download
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.String(http.StatusOK, exported)
}

// getPricing retrieves pricing information with optional filtering
func (s *Server) getPricing(c *gin.Context) {
	// Build filters from query parameters
	filters := make(map[string]any)

	// Model ID filter
	if modelIDStr := c.Query("model_id"); modelIDStr != "" {
		modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model_id parameter"})
			return
		}
		filters["model_id"] = modelID
	}

	// Pricing model filter
	if pricingModel := c.Query("pricing_model"); pricingModel != "" {
		filters["pricing_model"] = pricingModel
	}

	// Currency filter
	if currency := c.Query("currency"); currency != "" {
		filters["currency"] = currency
	}

	// Get pricing with filters
	pricing, err := s.database.ListPricing(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pricing"})
		return
	}

	// Prepare response
	response := gin.H{
		"pricing": pricing,
		"filters": filters,
	}

	// Add model_id to response if it was filtered
	if modelID, ok := filters["model_id"]; ok {
		response["model_id"] = modelID
	}

	c.JSON(http.StatusOK, response)
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
	var req CreatePricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	pricing := database.Pricing{
		ModelID:              req.ModelID,
		InputTokenCost:       req.InputTokenCost,
		OutputTokenCost:      req.OutputTokenCost,
		CachedInputTokenCost: req.CachedInputTokenCost,
		StorageCost:          req.StorageCost,
		RequestCost:          req.RequestCost,
		Currency:             req.Currency,
		PricingModel:         req.PricingModel,
		EffectiveFrom:        req.EffectiveFrom,
		EffectiveTo:          req.EffectiveTo,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
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

	var req UpdatePricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Get existing pricing to preserve fields not in request
	existingPricing, err := s.database.GetPricing(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pricing not found"})
		return
	}

	// Update fields from request
	if req.ModelID != 0 {
		existingPricing.ModelID = req.ModelID
	}
	if req.InputTokenCost != 0 {
		existingPricing.InputTokenCost = req.InputTokenCost
	}
	if req.OutputTokenCost != 0 {
		existingPricing.OutputTokenCost = req.OutputTokenCost
	}
	if req.CachedInputTokenCost != 0 {
		existingPricing.CachedInputTokenCost = req.CachedInputTokenCost
	}
	if req.StorageCost != 0 {
		existingPricing.StorageCost = req.StorageCost
	}
	if req.RequestCost != 0 {
		existingPricing.RequestCost = req.RequestCost
	}
	if req.Currency != "" {
		existingPricing.Currency = req.Currency
	}
	if req.PricingModel != "" {
		existingPricing.PricingModel = req.PricingModel
	}
	if req.EffectiveFrom != nil {
		existingPricing.EffectiveFrom = req.EffectiveFrom
	}
	if req.EffectiveTo != nil {
		existingPricing.EffectiveTo = req.EffectiveTo
	}
	existingPricing.UpdatedAt = time.Now()

	err = s.database.UpdatePricing(existingPricing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pricing"})
		return
	}

	c.JSON(http.StatusOK, existingPricing)
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
	// Build filters from query parameters
	filters := make(map[string]any)

	// Model ID filter
	if modelIDStr := c.Query("model_id"); modelIDStr != "" {
		modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model_id parameter"})
			return
		}
		filters["model_id"] = modelID
	}

	// Limit type filter
	if limitType := c.Query("limit_type"); limitType != "" {
		filters["limit_type"] = limitType
	}

	// Hard limit filter
	if isHardLimitStr := c.Query("is_hard_limit"); isHardLimitStr != "" {
		isHardLimit, err := strconv.ParseBool(isHardLimitStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid is_hard_limit parameter"})
			return
		}
		filters["is_hard_limit"] = isHardLimit
	}

	// Get limits with filters
	limits, err := s.database.ListLimits(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve limits"})
		return
	}

	// Prepare response
	response := gin.H{
		"limits":  limits,
		"filters": filters,
	}

	// Add model_id to response if it was filtered
	if modelID, ok := filters["model_id"]; ok {
		response["model_id"] = modelID
	}

	c.JSON(http.StatusOK, response)
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
	var req CreateLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	limit := database.Limit{
		ModelID:      req.ModelID,
		LimitType:    req.LimitType,
		LimitValue:   req.LimitValue,
		CurrentUsage: req.CurrentUsage,
		ResetPeriod:  req.ResetPeriod,
		ResetTime:    req.ResetTime,
		IsHardLimit:  req.IsHardLimit,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
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

	var req UpdateLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Get existing limit to preserve fields not in request
	existingLimit, err := s.database.GetLimit(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Limit not found"})
		return
	}

	// Update fields from request
	if req.ModelID != 0 {
		existingLimit.ModelID = req.ModelID
	}
	if req.LimitType != "" {
		existingLimit.LimitType = req.LimitType
	}
	if req.LimitValue != 0 {
		existingLimit.LimitValue = req.LimitValue
	}
	if req.CurrentUsage != 0 {
		existingLimit.CurrentUsage = req.CurrentUsage
	}
	if req.ResetPeriod != "" {
		existingLimit.ResetPeriod = req.ResetPeriod
	}
	if req.ResetTime != nil {
		existingLimit.ResetTime = req.ResetTime
	}
	if req.IsHardLimit != nil {
		existingLimit.IsHardLimit = *req.IsHardLimit
	}
	existingLimit.UpdatedAt = time.Now()

	err = s.database.UpdateLimit(existingLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update limit"})
		return
	}

	c.JSON(http.StatusOK, existingLimit)
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

	filters := map[string]any{
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
	var req CreateIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	issue := database.Issue{
		ModelID:              req.ModelID,
		IssueType:            req.IssueType,
		Severity:             req.Severity,
		Title:                req.Title,
		Description:          req.Description,
		Symptoms:             req.Symptoms,
		Workarounds:          req.Workarounds,
		AffectedFeatures:     req.AffectedFeatures,
		FirstDetected:        req.FirstDetected,
		LastOccurred:         req.LastOccurred,
		ResolvedAt:           req.ResolvedAt,
		ResolutionNotes:      req.ResolutionNotes,
		VerificationResultID: req.VerificationResultID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
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

	var req UpdateIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Get existing issue to preserve fields not in request
	existingIssue, err := s.database.GetIssue(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		return
	}

	// Update fields from request
	if req.ModelID != 0 {
		existingIssue.ModelID = req.ModelID
	}
	if req.IssueType != "" {
		existingIssue.IssueType = req.IssueType
	}
	if req.Severity != "" {
		existingIssue.Severity = req.Severity
	}
	if req.Title != "" {
		existingIssue.Title = req.Title
	}
	if req.Description != "" {
		existingIssue.Description = req.Description
	}
	if req.Symptoms != nil {
		existingIssue.Symptoms = req.Symptoms
	}
	if req.Workarounds != nil {
		existingIssue.Workarounds = req.Workarounds
	}
	if req.AffectedFeatures != nil {
		existingIssue.AffectedFeatures = req.AffectedFeatures
	}
	if !req.FirstDetected.IsZero() {
		existingIssue.FirstDetected = req.FirstDetected
	}
	if req.LastOccurred != nil {
		existingIssue.LastOccurred = req.LastOccurred
	}
	if req.ResolvedAt != nil {
		existingIssue.ResolvedAt = req.ResolvedAt
	}
	if req.ResolutionNotes != nil {
		existingIssue.ResolutionNotes = req.ResolutionNotes
	}
	if req.VerificationResultID != nil {
		existingIssue.VerificationResultID = req.VerificationResultID
	}
	existingIssue.UpdatedAt = time.Now()

	err = s.database.UpdateIssue(existingIssue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update issue"})
		return
	}

	c.JSON(http.StatusOK, existingIssue)
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

	filters := map[string]any{
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
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	event := database.Event{
		EventType:            req.EventType,
		Severity:             req.Severity,
		Title:                req.Title,
		Message:              req.Message,
		Details:              req.Details,
		ModelID:              req.ModelID,
		ProviderID:           req.ProviderID,
		VerificationResultID: req.VerificationResultID,
		IssueID:              req.IssueID,
		CreatedAt:            time.Now(),
	}

	err := s.database.CreateEvent(&event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	// Publish event via event bus
	if s.eventBus != nil {
		busEvent := events.Event{
			ID:        fmt.Sprintf("db_%d", event.ID),
			Type:      events.EventType(event.EventType),
			Severity:  events.EventSeverity(event.Severity),
			Message:   event.Message,
			Data:      make(map[string]any),
			Timestamp: event.CreatedAt,
			Source:    "api",
		}

		// Add event details to data if present
		if event.Details != nil {
			busEvent.Data["details"] = *event.Details
		}
		if event.ModelID != nil {
			busEvent.Data["model_id"] = *event.ModelID
		}
		if event.ProviderID != nil {
			busEvent.Data["provider_id"] = *event.ProviderID
		}

		go func() {
			if err := s.eventBus.Publish(busEvent); err != nil {
				log.Printf("Failed to publish event: %v", err)
			}
		}()
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

	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Get existing event to preserve fields not in request
	existingEvent, err := s.database.GetEvent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Update fields from request
	if req.EventType != "" {
		existingEvent.EventType = req.EventType
	}
	if req.Severity != "" {
		existingEvent.Severity = req.Severity
	}
	if req.Title != "" {
		existingEvent.Title = req.Title
	}
	if req.Message != "" {
		existingEvent.Message = req.Message
	}
	if req.Details != nil {
		existingEvent.Details = req.Details
	}
	if req.ModelID != nil {
		existingEvent.ModelID = req.ModelID
	}
	if req.ProviderID != nil {
		existingEvent.ProviderID = req.ProviderID
	}
	if req.VerificationResultID != nil {
		existingEvent.VerificationResultID = req.VerificationResultID
	}
	if req.IssueID != nil {
		existingEvent.IssueID = req.IssueID
	}

	err = s.database.UpdateEvent(existingEvent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event"})
		return
	}

	c.JSON(http.StatusOK, existingEvent)
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

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, implement proper origin checking
		return true
	},
}

// handleWebSocket handles WebSocket connections for real-time event streaming
func (s *Server) handleWebSocket(c *gin.Context) {
	// Get event types to subscribe to from query parameters
	eventTypesStr := c.Query("types")
	var eventTypes []events.EventType

	if eventTypesStr != "" {
		// Parse comma-separated event types
		typeStrings := strings.Split(eventTypesStr, ",")
		for _, ts := range typeStrings {
			ts = strings.TrimSpace(ts)
			switch ts {
			case "model.verified":
				eventTypes = append(eventTypes, events.EventTypeModelVerified)
			case "model.verification.failed":
				eventTypes = append(eventTypes, events.EventTypeModelVerificationFailed)
			case "model.score.changed":
				eventTypes = append(eventTypes, events.EventTypeScoreChanged)
			case "provider.added":
				eventTypes = append(eventTypes, events.EventTypeProviderAdded)
			case "provider.updated":
				eventTypes = append(eventTypes, events.EventTypeProviderUpdated)
			case "system.health.changed":
				eventTypes = append(eventTypes, events.EventTypeSystemHealthChanged)
			case "verification.started":
				eventTypes = append(eventTypes, events.EventTypeVerificationStarted)
			case "verification.completed":
				eventTypes = append(eventTypes, events.EventTypeVerificationCompleted)
			case "schedule.executed":
				eventTypes = append(eventTypes, events.EventTypeScheduleExecuted)
			case "error.occurred":
				eventTypes = append(eventTypes, events.EventTypeErrorOccurred)
			}
		}
	} else {
		// Subscribe to all event types if none specified
		eventTypes = []events.EventType{
			events.EventTypeModelVerified,
			events.EventTypeModelVerificationFailed,
			events.EventTypeScoreChanged,
			events.EventTypeProviderAdded,
			events.EventTypeProviderUpdated,
			events.EventTypeSystemHealthChanged,
			events.EventTypeVerificationStarted,
			events.EventTypeVerificationCompleted,
			events.EventTypeScheduleExecuted,
			events.EventTypeErrorOccurred,
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Create WebSocket subscriber
	subscriberID := fmt.Sprintf("ws_%d", time.Now().UnixNano())
	wsSubscriber := &events.WebSocketSubscriber{
		ID:     subscriberID,
		Conn:   conn,
		Types:  eventTypes,
		Active: true,
	}

	// Subscribe to event bus
	s.eventBus.Subscribe(wsSubscriber)
	defer s.eventBus.Unsubscribe(subscriberID)

	log.Printf("WebSocket subscriber %s connected, subscribed to %d event types", subscriberID, len(eventTypes))

	// Keep connection alive and handle incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for subscriber %s: %v", subscriberID, err)
			}
			break
		}
		// For now, we don't handle incoming messages from client
		// Could be extended to handle subscription changes, etc.
	}

	log.Printf("WebSocket subscriber %s disconnected", subscriberID)
}

// getEventSubscribers returns information about active event subscribers
func (s *Server) getEventSubscribers(c *gin.Context) {
	subscribers := s.eventBus.GetSubscribers()

	// Convert subscribers to API-friendly format
	subscriberInfo := make([]gin.H, len(subscribers))
	for i, sub := range subscribers {
		subscriberInfo[i] = gin.H{
			"id":          sub.GetID(),
			"active":      sub.IsActive(),
			"event_types": sub.GetTypes(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"subscribers": subscriberInfo,
		"total":       len(subscribers),
	})
}

// createNotificationSubscriber creates a new notification subscriber
func (s *Server) createNotificationSubscriber(c *gin.Context) {
	var req struct {
		Channels   []string `json:"channels" binding:"required"`
		EventTypes []string `json:"event_types" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Convert string event types to EventType
	var eventTypes []events.EventType
	for _, et := range req.EventTypes {
		switch et {
		case "model.verified":
			eventTypes = append(eventTypes, events.EventTypeModelVerified)
		case "model.verification.failed":
			eventTypes = append(eventTypes, events.EventTypeModelVerificationFailed)
		case "model.score.changed":
			eventTypes = append(eventTypes, events.EventTypeScoreChanged)
		case "provider.added":
			eventTypes = append(eventTypes, events.EventTypeProviderAdded)
		case "provider.updated":
			eventTypes = append(eventTypes, events.EventTypeProviderUpdated)
		case "system.health.changed":
			eventTypes = append(eventTypes, events.EventTypeSystemHealthChanged)
		case "verification.started":
			eventTypes = append(eventTypes, events.EventTypeVerificationStarted)
		case "verification.completed":
			eventTypes = append(eventTypes, events.EventTypeVerificationCompleted)
		case "schedule.executed":
			eventTypes = append(eventTypes, events.EventTypeScheduleExecuted)
		case "error.occurred":
			eventTypes = append(eventTypes, events.EventTypeErrorOccurred)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event type: " + et})
			return
		}
	}

	// Create notification subscriber
	subscriberID := fmt.Sprintf("notification_%d", time.Now().UnixNano())
	notificationSub := &events.NotificationSubscriber{
		ID:       subscriberID,
		Channels: req.Channels,
		Types:    eventTypes,
		Active:   true,
	}

	// Subscribe to event bus
	s.eventBus.Subscribe(notificationSub)

	c.JSON(http.StatusCreated, gin.H{
		"id":          subscriberID,
		"channels":    req.Channels,
		"event_types": req.EventTypes,
		"active":      true,
	})
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

	filters := map[string]any{
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
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	schedule := database.Schedule{
		Name:            req.Name,
		Description:     req.Description,
		ScheduleType:    req.ScheduleType,
		CronExpression:  req.CronExpression,
		IntervalSeconds: req.IntervalSeconds,
		TargetType:      req.TargetType,
		TargetID:        req.TargetID,
		IsActive:        req.IsActive,
		MaxRuns:         req.MaxRuns,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
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

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Get existing schedule to preserve fields not in request
	existingSchedule, err := s.database.GetSchedule(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	// Update fields from request
	if req.Name != "" {
		existingSchedule.Name = req.Name
	}
	if req.Description != nil {
		existingSchedule.Description = req.Description
	}
	if req.ScheduleType != "" {
		existingSchedule.ScheduleType = req.ScheduleType
	}
	if req.CronExpression != nil {
		existingSchedule.CronExpression = req.CronExpression
	}
	if req.IntervalSeconds != nil {
		existingSchedule.IntervalSeconds = req.IntervalSeconds
	}
	if req.TargetType != "" {
		existingSchedule.TargetType = req.TargetType
	}
	if req.TargetID != nil {
		existingSchedule.TargetID = req.TargetID
	}
	if req.IsActive != nil {
		existingSchedule.IsActive = *req.IsActive
	}
	if req.MaxRuns != nil {
		existingSchedule.MaxRuns = req.MaxRuns
	}
	existingSchedule.UpdatedAt = time.Now()

	err = s.database.UpdateSchedule(existingSchedule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule"})
		return
	}

	c.JSON(http.StatusOK, existingSchedule)
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

	filters := map[string]any{
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
	var req CreateConfigExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	configExport := database.ConfigExport{
		ExportType:        req.ExportType,
		Name:              req.Name,
		Description:       req.Description,
		ConfigData:        req.ConfigData,
		TargetModels:      req.TargetModels,
		TargetProviders:   req.TargetProviders,
		Filters:           req.Filters,
		IsVerified:        req.IsVerified,
		VerificationNotes: req.VerificationNotes,
		CreatedBy:         req.CreatedBy,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
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

	var req UpdateConfigExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	// Get existing config export to preserve fields not in request
	existingExport, err := s.database.GetConfigExport(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Config export not found"})
		return
	}

	// Update fields from request
	if req.ExportType != "" {
		existingExport.ExportType = req.ExportType
	}
	if req.Name != "" {
		existingExport.Name = req.Name
	}
	if req.Description != "" {
		existingExport.Description = req.Description
	}
	if req.ConfigData != "" {
		existingExport.ConfigData = req.ConfigData
	}
	if req.TargetModels != nil {
		existingExport.TargetModels = req.TargetModels
	}
	if req.TargetProviders != nil {
		existingExport.TargetProviders = req.TargetProviders
	}
	if req.Filters != nil {
		existingExport.Filters = req.Filters
	}
	if req.IsVerified != nil {
		existingExport.IsVerified = *req.IsVerified
	}
	if req.VerificationNotes != nil {
		existingExport.VerificationNotes = req.VerificationNotes
	}
	if req.CreatedBy != nil {
		existingExport.CreatedBy = req.CreatedBy
	}
	existingExport.UpdatedAt = time.Now()

	err = s.database.UpdateConfigExport(existingExport)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config export"})
		return
	}

	c.JSON(http.StatusOK, existingExport)
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
		log.Printf("Failed to increment download count for config export ID %d: %v", id, err)
	}

	// Set headers for file download
	c.Header("Content-Disposition", "attachment; filename="+configExport.Name+".json")
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", []byte(configExport.ConfigData))
}

// verifyConfigExport verifies a config export configuration
func (s *Server) verifyConfigExport(c *gin.Context) {
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

	// Verify the configuration
	isValid, message := s.verifyConfiguration(configExport.ExportType, configExport.ConfigData)

	// Update the verification status
	configExport.IsVerified = isValid
	configExport.VerificationNotes = &message
	configExport.UpdatedAt = time.Now()

	err = s.database.UpdateConfigExport(configExport)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config export verification status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                 configExport.ID,
		"export_type":        configExport.ExportType,
		"is_verified":        configExport.IsVerified,
		"verification_notes": configExport.VerificationNotes,
		"updated_at":         configExport.UpdatedAt,
	})
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

	filters := map[string]any{
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

// exportConfiguration exports the current configuration in the specified format
func (s *Server) exportConfiguration(format string) (string, error) {
	switch format {
	case "json":
		return s.exportAsJSON()
	case "yaml":
		return s.exportAsYAML()
	case "opencode":
		return s.exportAsOpenCode()
	case "claude":
		return s.exportAsClaude()
	case "crush":
		return s.exportAsCrush()
	case "vscode":
		return s.exportAsVSCode()
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
}

// verifyConfiguration verifies that an exported configuration is valid for the specified format
func (s *Server) verifyConfiguration(format, configData string) (bool, string) {
	switch format {
	case "json":
		return s.verifyJSONConfiguration(configData)
	case "yaml":
		return s.verifyYAMLConfiguration(configData)
	case "opencode":
		return s.verifyOpenCodeConfiguration(configData)
	case "claude":
		return s.verifyClaudeConfiguration(configData)
	case "crush":
		return s.verifyCrushConfiguration(configData)
	case "vscode":
		return s.verifyVSCodeConfiguration(configData)
	default:
		return false, "Unsupported format for verification"
	}
}

// exportAsJSON exports configuration as JSON
func (s *Server) exportAsJSON() (string, error) {
	// Convert config to JSON
	jsonBytes, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// exportAsYAML exports configuration as YAML
func (s *Server) exportAsYAML() (string, error) {
	// Convert config to YAML
	yamlBytes, err := yaml.Marshal(s.config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to YAML: %w", err)
	}
	return string(yamlBytes), nil
}

// exportAsOpenCode exports configuration in OpenCode format
func (s *Server) exportAsOpenCode() (string, error) {
	// OpenCode format: JSON with specific structure for OpenCode AI assistant
	opencodeConfig := map[string]any{
		"name":    "LLM Verifier Configuration",
		"version": "1.0",
		"config": map[string]any{
			"llms": s.config.LLMs,
			"global": map[string]any{
				"base_url":      s.config.Global.BaseURL,
				"default_model": s.config.Global.DefaultModel,
				"max_retries":   s.config.Global.MaxRetries,
				"request_delay": s.config.Global.RequestDelay.String(),
				"timeout":       s.config.Global.Timeout.String(),
			},
			"api": map[string]any{
				"port":        s.config.API.Port,
				"rate_limit":  s.config.API.RateLimit,
				"enable_cors": s.config.API.EnableCORS,
			},
			"concurrency": s.config.Concurrency,
			"timeout":     s.config.Timeout.String(),
		},
		"instructions": "This configuration file is for the LLM Verifier tool. It defines LLM endpoints, API settings, and global configuration.",
	}

	jsonBytes, err := json.MarshalIndent(opencodeConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenCode config: %w", err)
	}
	return string(jsonBytes), nil
}

// exportAsClaude exports configuration in Claude Code format
func (s *Server) exportAsClaude() (string, error) {
	// Claude Code format: Simplified JSON structure for Claude AI assistant
	claudeConfig := map[string]any{
		"llm_verifier_config": map[string]any{
			"llm_endpoints": s.config.LLMs,
			"settings": map[string]any{
				"concurrency":           s.config.Concurrency,
				"timeout_seconds":       int(s.config.Timeout.Seconds()),
				"api_port":              s.config.API.Port,
				"rate_limit_per_minute": s.config.API.RateLimit,
			},
			"global": map[string]any{
				"default_model": s.config.Global.DefaultModel,
				"max_retries":   s.config.Global.MaxRetries,
			},
		},
		"description": "LLM Verifier configuration for Claude Code assistant integration",
	}

	jsonBytes, err := json.MarshalIndent(claudeConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Claude config: %w", err)
	}
	return string(jsonBytes), nil
}

// exportAsCrush exports configuration in Crush format
func (s *Server) exportAsCrush() (string, error) {
	// Crush format: JSON with MCP and LLM configuration structure
	// Based on Crush's configuration schema at https://charm.land/crush.json
	crushConfig := map[string]any{
		"$schema": "https://charm.land/crush.json",
		"mcp": map[string]any{
			"llm-verifier": map[string]any{
				"type":    "http",
				"command": "http://localhost:" + s.config.API.Port + "/api/v1",
				"headers": map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		"llm": map[string]any{
			"endpoints": s.config.LLMs,
			"settings": map[string]any{
				"concurrency":           s.config.Concurrency,
				"timeout_seconds":       int(s.config.Timeout.Seconds()),
				"default_model":         s.config.Global.DefaultModel,
				"max_retries":           s.config.Global.MaxRetries,
				"request_delay_seconds": s.config.Global.RequestDelay.Seconds(),
			},
		},
		"description": "LLM Verifier configuration for Crush AI assistant integration",
	}

	jsonBytes, err := json.MarshalIndent(crushConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Crush config: %w", err)
	}
	return string(jsonBytes), nil
}

// verifyJSONConfiguration validates JSON configuration format
func (s *Server) verifyJSONConfiguration(configData string) (bool, string) {
	var data map[string]any
	if err := json.Unmarshal([]byte(configData), &data); err != nil {
		return false, "Invalid JSON format: " + err.Error()
	}
	return true, "Valid JSON configuration"
}

// verifyYAMLConfiguration validates YAML configuration format
func (s *Server) verifyYAMLConfiguration(configData string) (bool, string) {
	var data map[string]any
	if err := yaml.Unmarshal([]byte(configData), &data); err != nil {
		return false, "Invalid YAML format: " + err.Error()
	}
	return true, "Valid YAML configuration"
}

// verifyOpenCodeConfiguration validates OpenCode configuration format
func (s *Server) verifyOpenCodeConfiguration(configData string) (bool, string) {
	var data map[string]any
	if err := json.Unmarshal([]byte(configData), &data); err != nil {
		return false, "Invalid JSON format: " + err.Error()
	}

	// Check required fields
	requiredFields := []string{"name", "version", "config"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			return false, "Missing required field: " + field
		}
	}

	// Check config structure
	config, ok := data["config"].(map[string]any)
	if !ok {
		return false, "Config field must be an object"
	}

	if _, exists := config["llms"]; !exists {
		return false, "Config must contain llms field"
	}

	return true, "Valid OpenCode configuration"
}

// verifyClaudeConfiguration validates Claude Code configuration format
func (s *Server) verifyClaudeConfiguration(configData string) (bool, string) {
	var data map[string]any
	if err := json.Unmarshal([]byte(configData), &data); err != nil {
		return false, "Invalid JSON format: " + err.Error()
	}

	// Check required fields
	if _, exists := data["llm_verifier_config"]; !exists {
		return false, "Missing required field: llm_verifier_config"
	}

	config, ok := data["llm_verifier_config"].(map[string]any)
	if !ok {
		return false, "llm_verifier_config field must be an object"
	}

	if _, exists := config["llm_endpoints"]; !exists {
		return false, "Config must contain llm_endpoints field"
	}

	return true, "Valid Claude Code configuration"
}

// verifyCrushConfiguration validates Crush configuration format
func (s *Server) verifyCrushConfiguration(configData string) (bool, string) {
	var data map[string]any
	if err := json.Unmarshal([]byte(configData), &data); err != nil {
		return false, "Invalid JSON format: " + err.Error()
	}

	// Check schema
	if schema, exists := data["$schema"]; !exists || schema != "https://charm.land/crush.json" {
		return false, "Missing or invalid $schema field"
	}

	// Check MCP configuration
	mcp, ok := data["mcp"].(map[string]any)
	if !ok {
		return false, "Missing or invalid mcp field"
	}

	if _, exists := mcp["llm-verifier"]; !exists {
		return false, "MCP configuration must contain llm-verifier"
	}

	// Check LLM configuration
	llm, ok := data["llm"].(map[string]any)
	if !ok {
		return false, "Missing or invalid llm field"
	}

	if _, exists := llm["endpoints"]; !exists {
		return false, "LLM configuration must contain endpoints field"
	}

	return true, "Valid Crush configuration"
}

// verifyVSCodeConfiguration validates VS Code configuration format
func (s *Server) verifyVSCodeConfiguration(configData string) (bool, string) {
	var data map[string]any
	if err := json.Unmarshal([]byte(configData), &data); err != nil {
		return false, "Invalid JSON format: " + err.Error()
	}

	// Check for llmVerifier configuration
	if _, exists := data["llmVerifier"]; !exists {
		return false, "Missing required field: llmVerifier"
	}

	return true, "Valid VS Code configuration"
}

// exportAsVSCode exports configuration in VS Code settings format
func (s *Server) exportAsVSCode() (string, error) {
	// VS Code format: JSON with VS Code settings structure
	vscodeConfig := map[string]any{
		"llmVerifier": map[string]any{
			"llms":        s.config.LLMs,
			"concurrency": s.config.Concurrency,
			"timeout":     s.config.Timeout.String(),
		},
		"llmVerifier.api": map[string]any{
			"port":       s.config.API.Port,
			"rateLimit":  s.config.API.RateLimit,
			"enableCors": s.config.API.EnableCORS,
		},
		"llmVerifier.global": map[string]any{
			"defaultModel": s.config.Global.DefaultModel,
			"maxRetries":   s.config.Global.MaxRetries,
			"requestDelay": s.config.Global.RequestDelay.String(),
		},
		"[json]": map[string]any{
			"editor.defaultFormatter": "esbenp.prettier-vscode",
		},
	}

	jsonBytes, err := json.MarshalIndent(vscodeConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal VS Code config: %w", err)
	}
	return string(jsonBytes), nil
}

// getUsers retrieves all users
func (s *Server) getUsers(c *gin.Context) {
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

	filters := map[string]any{
		"limit":  limit,
		"offset": offset,
	}

	users, err := s.database.ListUsers(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// getUserByID retrieves a specific user by ID
func (s *Server) getUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := s.database.GetUser(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// createUser creates a new user
func (s *Server) createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	user := database.User{
		Username:  req.Username,
		Email:     req.Email,
		Role:      req.Role,
		IsActive:  req.IsActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.PasswordHash = string(hashedPassword)

	err = s.database.CreateUser(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// updateUser updates an existing user
func (s *Server) updateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	existingUser, err := s.database.GetUser(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if req.Username != "" {
		existingUser.Username = req.Username
	}
	if req.Email != "" {
		existingUser.Email = req.Email
	}
	if req.Role != "" {
		existingUser.Role = req.Role
	}
	if req.IsActive != nil {
		existingUser.IsActive = *req.IsActive
	}
	existingUser.UpdatedAt = time.Now()

	err = s.database.UpdateUser(existingUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, existingUser)
}

// deleteUser deletes a user
func (s *Server) deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = s.database.DeleteUser(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.Status(http.StatusNoContent)
}

// getCurrentUser retrieves the current authenticated user
func (s *Server) getCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, ok := userID.(float64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	user, err := s.database.GetUser(int64(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// updateCurrentUser updates the current authenticated user
func (s *Server) updateCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, ok := userID.(float64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	var req UpdateCurrentUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleValidationError(c, err)
		return
	}

	existingUser, err := s.database.GetUser(int64(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update allowed fields
	if req.Email != "" {
		existingUser.Email = req.Email
	}
	existingUser.UpdatedAt = time.Now()

	err = s.database.UpdateUser(existingUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, existingUser)
}

// getSystemInfo returns system information
// @Summary Get system information
// @Description Returns system version, statistics, and counts
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{} "System information"
// @Router /system/info [get]
func (s *Server) getSystemInfo(c *gin.Context) {
	// Get basic system stats
	var totalModels, totalProviders, totalVerifications int64

	// Get counts from database
	if s.database != nil {
		totalModels, _ = s.database.GetModelCount()
		totalProviders, _ = s.database.GetProviderCount()
		totalVerifications, _ = s.database.GetVerificationResultCount()
	}

	// Build system info response
	systemInfo := map[string]any{
		"version":             "1.0.0",
		"build_time":          "2024-01-01T00:00:00Z", // This should be set at build time
		"go_version":          "1.21.0",
		"git_commit":          "abc123", // This should be set at build time
		"database_version":    "3.40.0",
		"total_models":        totalModels,
		"total_providers":     totalProviders,
		"total_verifications": totalVerifications,
		"system_stats": map[string]any{
			"cpu_usage":    15.2, // Placeholder - in production, get actual stats
			"memory_usage": 45.8, // Placeholder
			"disk_usage":   23.1, // Placeholder
		},
	}

	SendSuccess(c, http.StatusOK, systemInfo, "")
}

// getDatabaseStats returns database statistics
// @Summary Get database statistics
// @Description Returns database performance and size statistics
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{} "Database statistics"
// @Failure 503 {object} map[string]interface{} "Database not available"
// @Router /system/database-stats [get]
func (s *Server) getDatabaseStats(c *gin.Context) {
	if s.database == nil {
		SendError(c, http.StatusServiceUnavailable, ErrCodeDatabase, "Database not available", nil)
		return
	}

	// Get database stats
	stats, err := s.database.GetDatabaseStats()
	if err != nil {
		SendError(c, http.StatusInternalServerError, ErrCodeDatabase, "Failed to get database stats", map[string]string{
			"error": err.Error(),
		})
		return
	}

	SendSuccess(c, http.StatusOK, stats, "")
}

// Request/Response types for user operations
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required,oneof=admin user"`
	IsActive bool   `json:"is_active"`
}

type UpdateUserRequest struct {
	Username string `json:"username,omitempty" binding:"omitempty,min=3,max=50"`
	Email    string `json:"email,omitempty" binding:"omitempty,email"`
	Role     string `json:"role,omitempty" binding:"omitempty,oneof=admin user"`
	IsActive *bool  `json:"is_active,omitempty"`
}

type UpdateCurrentUserRequest struct {
	Email string `json:"email,omitempty" binding:"omitempty,email"`
}
