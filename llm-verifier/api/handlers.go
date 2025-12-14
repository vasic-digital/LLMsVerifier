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
// @Summary Health check endpoint
// @Description Check if the API server is running
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
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
// @Param credentials body map[string]string true "User credentials"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (s *Server) login(c *gin.Context) {
	var credentials struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// TODO: Implement proper user authentication
	// For now, accept any username/password combination
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
		"expires_in": 86400, // 24 hours
		"user": gin.H{
			"id":       1,
			"username": credentials.Username,
			"role":     "admin",
		},
	})
}

// refreshToken handles token refresh
// @Summary Refresh JWT token
// @Description Refresh an expired JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh_token body map[string]string true "Refresh token"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/refresh [post]
func (s *Server) refreshToken(c *gin.Context) {
	// TODO: Implement token refresh logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Token refresh not implemented"})
}

// getModels retrieves all models
// @Summary Get all models
// @Description Retrieve a list of all verified models
// @Tags models
// @Accept json
// @Produce json
// @Param limit query int false "Limit number of results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/v1/models [get]
func (s *Server) getModels(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	if modelIDStr != "" {
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
		// TODO: Implement general pricing list
		c.JSON(http.StatusNotImplemented, gin.H{"error": "General pricing list not implemented"})
	}
}

// getPricingByID retrieves specific pricing by ID
// @Summary Get pricing by ID
// @Description Retrieve specific pricing information by ID
// @Tags pricing
// @Accept json
// @Produce json
// @Param id path int true "Pricing ID"
// @Success 200 {object} database.Pricing
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/pricing/{id} [get]
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

// createPricing creates new pricing
// @Summary Create pricing
// @Description Create new pricing information
// @Tags pricing
// @Accept json
// @Produce json
// @Param pricing body database.Pricing true "Pricing data"
// @Success 201 {object} database.Pricing
// @Failure 400 {object} map[string]string
// @Router /api/v1/pricing [post]
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

// updatePricing updates existing pricing
// @Summary Update pricing
// @Description Update existing pricing information
// @Tags pricing
// @Accept json
// @Produce json
// @Param id path int true "Pricing ID"
// @Param pricing body database.Pricing true "Updated pricing data"
// @Success 200 {object} database.Pricing
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/pricing/{id} [put]
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

// deletePricing deletes pricing
// @Summary Delete pricing
// @Description Delete pricing information
// @Tags pricing
// @Accept json
// @Produce json
// @Param id path int true "Pricing ID"
// @Success 204 {object} nil
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/pricing/{id} [delete]
func (s *Server) deletePricing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pricing ID"})
		return
	}

	// TODO: Implement DeletePricing in database package
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Delete pricing not implemented"})
}

// getLimits retrieves limits information
// @Summary Get limits
// @Description Retrieve limits information with optional filtering
// @Tags limits
// @Accept json
// @Produce json
// @Param model_id query int false "Filter by model ID"
// @Param limit query int false "Limit number of results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/v1/limits [get]
func (s *Server) getLimits(c *gin.Context) {
	modelIDStr := c.Query("model_id")
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

	if modelIDStr != "" {
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
		// TODO: Implement general limits list
		c.JSON(http.StatusNotImplemented, gin.H{"error": "General limits list not implemented"})
	}
}

// getLimitByID retrieves specific limit by ID
// @Summary Get limit by ID
// @Description Retrieve specific limit information by ID
// @Tags limits
// @Accept json
// @Produce json
// @Param id path int true "Limit ID"
// @Success 200 {object} database.Limit
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/limits/{id} [get]
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

// createLimit creates new limit
// @Summary Create limit
// @Description Create new limit information
// @Tags limits
// @Accept json
// @Produce json
// @Param limit body database.Limit true "Limit data"
// @Success 201 {object} database.Limit
// @Failure 400 {object} map[string]string
// @Router /api/v1/limits [post]
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

// updateLimit updates existing limit
// @Summary Update limit
// @Description Update existing limit information
// @Tags limits
// @Accept json
// @Produce json
// @Param id path int true "Limit ID"
// @Param limit body database.Limit true "Updated limit data"
// @Success 200 {object} database.Limit
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/limits/{id} [put]
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

	// TODO: Implement UpdateLimit in database package
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Update limit not implemented"})
}

// deleteLimit deletes limit
// @Summary Delete limit
// @Description Delete limit information
// @Tags limits
// @Accept json
// @Produce json
// @Param id path int true "Limit ID"
// @Success 204 {object} nil
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/limits/{id} [delete]
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

// getIssues retrieves issues information
// @Summary Get issues
// @Description Retrieve issues with optional filtering
// @Tags issues
// @Accept json
// @Produce json
// @Param model_id query int false "Filter by model ID"
// @Param severity query string false "Filter by severity"
// @Param resolved query bool false "Include resolved issues"
// @Param limit query int false "Limit number of results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/v1/issues [get]
func (s *Server) getIssues(c *gin.Context) {
	modelIDStr := c.Query("model_id")
	severity := c.Query("severity")
	resolvedStr := c.Query("resolved")
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

	// Build filters
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
		"issues":  issues,
		"filters": filters,
	})
}

// getIssueByID retrieves specific issue by ID
// @Summary Get issue by ID
// @Description Retrieve specific issue information by ID
// @Tags issues
// @Accept json
// @Produce json
// @Param id path int true "Issue ID"
// @Success 200 {object} database.Issue
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/issues/{id} [get]
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

// createIssue creates new issue
// @Summary Create issue
// @Description Create new issue report
// @Tags issues
// @Accept json
// @Produce json
// @Param issue body database.Issue true "Issue data"
// @Success 201 {object} database.Issue
// @Failure 400 {object} map[string]string
// @Router /api/v1/issues [post]
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

// updateIssue updates existing issue
// @Summary Update issue
// @Description Update existing issue information
// @Tags issues
// @Accept json
// @Produce json
// @Param id path int true "Issue ID"
// @Param issue body database.Issue true "Updated issue data"
// @Success 200 {object} database.Issue
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/issues/{id} [put]
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

// deleteIssue deletes issue
// @Summary Delete issue
// @Description Delete issue information
// @Tags issues
// @Accept json
// @Produce json
// @Param id path int true "Issue ID"
// @Success 204 {object} nil
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/issues/{id} [delete]
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
