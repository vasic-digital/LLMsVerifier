package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"llm-verifier/database"
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

	// TODO: Implement async verification
	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Verification started",
		"model_id": id,
	})
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

// deleteVerificationResult deletes a verification result
func (s *Server) deleteVerificationResult(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Delete verification result not implemented"})
}

// generateReport generates a new report
func (s *Server) generateReport(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Report generation not implemented"})
}

// downloadReport downloads a generated report
func (s *Server) downloadReport(c *gin.Context) {
	reportID := c.Param("id")
	c.JSON(http.StatusNotFound, gin.H{"error": "Report " + reportID + " not found"})
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
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Configuration update not implemented"})
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
