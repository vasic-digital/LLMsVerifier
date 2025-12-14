package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"llm-verifier/config"
	"llm-verifier/database"
	"llm-verifier/llmverifier"
)

// Server represents the REST API server
type Server struct {
	router    *gin.Engine
	config    *config.Config
	database  *database.Database
	verifier  *llmverifier.Verifier
	jwtSecret []byte
}

// NewServer creates a new API server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize verifier
	verifier := llmverifier.New(cfg)

	server := &Server{
		router:    gin.Default(),
		config:    cfg,
		database:  db,
		verifier:  verifier,
		jwtSecret: []byte(cfg.API.JWTSecret),
	}

	server.setupRoutes()
	server.setupMiddleware()

	return server, nil
}

// setupMiddleware configures global middleware
func (s *Server) setupMiddleware() {
	// CORS middleware
	if s.config.API.EnableCORS {
		s.router.Use(corsMiddleware())
	}

	// Rate limiting middleware
	s.router.Use(s.rateLimitMiddleware())

	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logger middleware
	s.router.Use(gin.Logger())
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", s.healthCheck)

	// Authentication routes
	auth := s.router.Group("/auth")
	{
		auth.POST("/login", s.login)
		auth.POST("/refresh", s.refreshToken)
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	v1.Use(s.jwtAuthMiddleware())
	{
		// Models
		models := v1.Group("/models")
		{
			models.GET("", s.getModels)
			models.GET("/:id", s.getModel)
			models.POST("/:id/verify", s.verifyModel)
			models.DELETE("/:id", s.deleteModel)
		}

		// Providers
		providers := v1.Group("/providers")
		{
			providers.GET("", s.getProviders)
			providers.GET("/:id", s.getProvider)
			providers.POST("", s.createProvider)
			providers.PUT("/:id", s.updateProvider)
			providers.DELETE("/:id", s.deleteProvider)
		}

		// Verification results
		results := v1.Group("/verification-results")
		{
			results.GET("", s.getVerificationResults)
			results.GET("/:id", s.getVerificationResult)
			results.DELETE("/:id", s.deleteVerificationResult)
		}

		// Pricing
		pricing := v1.Group("/pricing")
		{
			pricing.GET("", s.getPricing)
			pricing.GET("/:id", s.getPricingByID)
			pricing.POST("", s.createPricing)
			pricing.PUT("/:id", s.updatePricing)
			pricing.DELETE("/:id", s.deletePricing)
		}

		// Limits
		limits := v1.Group("/limits")
		{
			limits.GET("", s.getLimits)
			limits.GET("/:id", s.getLimitByID)
			limits.POST("", s.createLimit)
			limits.PUT("/:id", s.updateLimit)
			limits.DELETE("/:id", s.deleteLimit)
		}

		// Issues
		issues := v1.Group("/issues")
		{
			issues.GET("", s.getIssues)
			issues.GET("/:id", s.getIssueByID)
			issues.POST("", s.createIssue)
			issues.PUT("/:id", s.updateIssue)
			issues.DELETE("/:id", s.deleteIssue)
		}

		// Events
		events := v1.Group("/events")
		{
			events.GET("", s.getEvents)
			events.GET("/:id", s.getEventByID)
			events.POST("", s.createEvent)
		}

		// Schedules
		schedules := v1.Group("/schedules")
		{
			schedules.GET("", s.getSchedules)
			schedules.GET("/:id", s.getScheduleByID)
			schedules.POST("", s.createSchedule)
			schedules.PUT("/:id", s.updateSchedule)
			schedules.DELETE("/:id", s.deleteSchedule)
		}

		// Config exports
		exports := v1.Group("/exports")
		{
			exports.GET("", s.getConfigExports)
			exports.GET("/:id", s.getConfigExportByID)
			exports.POST("", s.createConfigExport)
			exports.PUT("/:id", s.updateConfigExport)
			exports.DELETE("/:id", s.deleteConfigExport)
			exports.GET("/download/:id", s.downloadConfigExport)
		}

		// Logs
		logs := v1.Group("/logs")
		{
			logs.GET("", s.getLogs)
			logs.GET("/:id", s.getLogByID)
		}

		// Reports
		reports := v1.Group("/reports")
		{
			reports.POST("/generate", s.generateReport)
			reports.GET("/download/:id", s.downloadReport)
		}

		// Configuration
		config := v1.Group("/config")
		{
			config.GET("", s.getConfig)
			config.PUT("", s.updateConfig)
			config.POST("/export", s.exportConfig)
		}
	}

	// Swagger documentation
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// Start starts the HTTP server
func (s *Server) Start(port string) error {
	log.Printf("Starting LLM Verifier API server on port %s", port)
	return s.router.Run(":" + port)
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	if s.database != nil {
		return s.database.Close()
	}
	return nil
}

// Router returns the Gin router for testing
func (s *Server) Router() *gin.Engine {
	return s.router
}

// corsMiddleware handles CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// rateLimitMiddleware implements configurable rate limiting
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	// Simple in-memory rate limiter (in production, use Redis or similar)
	type client struct {
		requests  int
		resetTime time.Time
	}

	clients := make(map[string]*client)
	rateLimit := s.config.API.RateLimit
	if rateLimit <= 0 {
		rateLimit = 100 // default
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		if cli, exists := clients[ip]; exists {
			if now.After(cli.resetTime) {
				cli.requests = 1
				cli.resetTime = now.Add(time.Minute)
			} else if cli.requests >= rateLimit {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":             "Rate limit exceeded",
					"retry_after":       cli.resetTime.Unix(),
					"rate_limit":        rateLimit,
					"rate_limit_window": "60 seconds",
				})
				c.Abort()
				return
			} else {
				cli.requests++
			}
		} else {
			clients[ip] = &client{
				requests:  1,
				resetTime: now.Add(time.Minute),
			}
		}

		c.Next()
	}
}

// jwtAuthMiddleware validates JWT tokens
func (s *Server) jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		const bearerPrefix = "Bearer "
		if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := authHeader[len(bearerPrefix):]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.jwtSecret, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("user_id", claims["user_id"])
			c.Set("username", claims["username"])
			c.Set("role", claims["role"])
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Next()
	}
}
