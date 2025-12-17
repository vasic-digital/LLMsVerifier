// Package api LLM Verifier REST API
//
// This API provides comprehensive endpoints for managing and verifying Large Language Models (LLMs).
// It supports model discovery, verification workflows, reporting, and administrative functions.
//
// Terms Of Service: https://llm-verifier.ai/terms
//
// Schemes: http, https
// Host: localhost:8080
// BasePath: /api/v1
// Version: 1.0.0
//
// Consumes:
// - application/json
//
// Produces:
// - application/json
//
// SecurityDefinitions:
// BearerAuth:
//
//	type: apiKey
//	name: Authorization
//	in: header
//	description: "JWT Authorization header using the Bearer scheme. Example: 'Bearer {token}'"
//
// swagger:meta
package api

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"llm-verifier/config"
	"llm-verifier/database"
	"llm-verifier/enhanced"
	"llm-verifier/events"
	"llm-verifier/llmverifier"
	"llm-verifier/monitoring"
	"llm-verifier/notifications"
	"llm-verifier/scheduler"
)

// Server represents the REST API server
type Server struct {
	router          *gin.Engine
	config          *config.Config
	database        *database.Database
	verifier        *llmverifier.Verifier
	healthChecker   *monitoring.HealthChecker
	eventBus        *events.EventBus
	notificationMgr *notifications.NotificationManager
	scheduler       *scheduler.Scheduler
	issueMgr        *enhanced.IssueManager
	jwtSecret       []byte
	startTime       time.Time
}

// NewServer creates a new API server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Setup Gin to use our custom validator
	SetupGinValidator()

	// Initialize database
	var db *database.Database
	var err error
	if cfg.Database.EncryptionKey != "" {
		db, err = database.NewEncrypted(cfg.Database.Path, cfg.Database.EncryptionKey)
	} else {
		db, err = database.New(cfg.Database.Path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize verifier
	verifier := llmverifier.New(cfg)

	// Initialize health checker
	healthChecker := monitoring.NewHealthChecker(db)
	healthChecker.Start(30 * time.Second) // Check every 30 seconds

	// Initialize event bus
	eventBus := events.NewEventBus(db)

	// Initialize notification manager
	notificationMgr := notifications.NewNotificationManager(cfg, eventBus)

	// Initialize scheduler
	sched := scheduler.NewScheduler(db)

	// Initialize issue manager
	issueMgr := enhanced.NewIssueManager(db)

	server := &Server{
		router:          gin.Default(),
		config:          cfg,
		database:        db,
		verifier:        verifier,
		healthChecker:   healthChecker,
		eventBus:        eventBus,
		notificationMgr: notificationMgr,
		scheduler:       sched,
		issueMgr:        issueMgr,
		jwtSecret:       []byte(cfg.API.JWTSecret),
		startTime:       time.Now(),
	}

	server.setupMiddleware()
	server.setupRoutes()

	// Register comprehensive health endpoints
	server.healthChecker.RegisterHealthEndpoints(server.router)

	// Set up scheduler job handler
	server.setupScheduler()

	// Start scheduler
	if err := server.scheduler.Start(); err != nil {
		return nil, fmt.Errorf("failed to start scheduler: %w", err)
	}

	return server, nil
}

// setupMiddleware configures global middleware
func (s *Server) setupMiddleware() {
	// CORS middleware
	if s.config.API.EnableCORS {
		s.router.Use(corsMiddleware())
	}

	// Security headers middleware
	s.router.Use(s.securityHeadersMiddleware())

	// Rate limiting middleware
	s.router.Use(s.rateLimitMiddleware())

	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logger middleware
	s.router.Use(gin.Logger())
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check endpoints are registered by the health checker

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
		// Users
		users := v1.Group("/users")
		{
			users.GET("", s.getUsers)
			users.GET("/:id", s.getUserByID)
			users.POST("", s.createUser, s.requireRoleMiddleware("admin"))
			users.PUT("/:id", s.updateUser, s.requireRoleMiddleware("admin"))
			users.DELETE("/:id", s.deleteUser, s.requireRoleMiddleware("admin"))
			users.GET("/me", s.getCurrentUser)
			users.PUT("/me", s.updateCurrentUser)
		}

		// Models
		models := v1.Group("/models")
		{
			models.GET("", s.getModels)
			models.GET("/:id", s.getModel)
			models.POST("", s.createModel, s.requireRoleMiddleware("admin"))
			models.PUT("/:id", s.updateModel, s.requireRoleMiddleware("admin"))
			models.POST("/:id/verify", s.verifyModel)
			models.DELETE("/:id", s.deleteModel, s.requireRoleMiddleware("admin"))
		}

		// Providers
		providers := v1.Group("/providers")
		{
			providers.GET("", s.getProviders)
			providers.GET("/:id", s.getProvider)
			providers.POST("", s.createProvider, s.requireRoleMiddleware("admin"))
			providers.PUT("/:id", s.updateProvider, s.requireRoleMiddleware("admin"))
			providers.DELETE("/:id", s.deleteProvider, s.requireRoleMiddleware("admin"))
		}

		// Verification results
		results := v1.Group("/verification-results")
		{
			results.GET("", s.getVerificationResults)
			results.GET("/:id", s.getVerificationResult)
			results.POST("", s.createVerificationResult)
			results.PUT("/:id", s.updateVerificationResult)
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
			events.PUT("/:id", s.updateEvent)
			events.DELETE("/:id", s.deleteEvent)
			events.GET("/ws", s.handleWebSocket)
			events.GET("/subscribers", s.getEventSubscribers)
			events.POST("/subscribers/notifications", s.createNotificationSubscriber)
		}

		// Notifications
		notifications := v1.Group("/notifications")
		{
			notifications.GET("/channels", s.getNotificationChannels)
			notifications.POST("/test", s.testNotification)
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
			exports.POST("/:id/verify", s.verifyConfigExport)
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

		// System
		system := v1.Group("/system")
		{
			system.GET("/info", s.getSystemInfo)
			system.GET("/database-stats", s.getDatabaseStats)
		}
	}
}

// securityHeadersMiddleware adds security headers to all responses
func (s *Server) securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (restrictive for API)
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'none'; style-src 'none'; img-src 'none'; font-src 'none'; connect-src 'self'; media-src 'none'; object-src 'none'; frame-src 'none'")

		// HSTS (HTTP Strict Transport Security) - only for HTTPS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Remove server header for security
		c.Header("Server", "")

		c.Next()
	}
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

// rateLimitMiddleware implements configurable rate limiting with enhanced features
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	// Enhanced in-memory rate limiter with sliding window support
	type client struct {
		requests    int
		resetTime   time.Time
		windowStart time.Time
	}

	clients := make(map[string]*client)
	rateLimit := s.config.API.RateLimit
	if rateLimit <= 0 {
		rateLimit = 100 // default
	}

	burstLimit := s.config.API.BurstLimit
	if burstLimit <= 0 {
		burstLimit = rateLimit * 2 // default burst is 2x rate limit
	}

	windowSeconds := s.config.API.RateLimitWindow
	if windowSeconds <= 0 {
		windowSeconds = 60 // default 60-second window
	}
	windowSize := time.Duration(windowSeconds) * time.Second

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		// Skip rate limiting for certain paths (health checks, metrics)
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/health") ||
			strings.HasPrefix(c.Request.URL.Path, "/api/v1/metrics") {
			c.Next()
			return
		}

		// Get or create client record
		cli, exists := clients[ip]
		if !exists {
			cli = &client{
				requests:    1,
				resetTime:   now.Add(windowSize),
				windowStart: now,
			}
			clients[ip] = cli
		} else {
			// Check if window has expired
			if now.After(cli.resetTime) {
				cli.requests = 1
				cli.resetTime = now.Add(windowSize)
				cli.windowStart = now
			} else {
				// Check if rate limit is exceeded
				timeInWindow := now.Sub(cli.windowStart).Seconds()
				expectedRequests := int(float64(rateLimit) * (timeInWindow / float64(windowSeconds)))

				// Check burst limit first (more strict)
				if cli.requests > expectedRequests+burstLimit {
					// Burst limit exceeded - immediate rejection
					retryAfter := int(cli.resetTime.Sub(now).Seconds())

					// Set standard rate limit headers
					c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimit))
					c.Header("X-RateLimit-Remaining", "0")
					c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", cli.resetTime.Unix()))
					c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
					c.Header("X-RateLimit-Burst-Exceeded", "true")

					// Clean up old entries periodically
					if len(clients) > 10000 {
						for k, v := range clients {
							if now.After(v.resetTime.Add(5 * time.Minute)) {
								delete(clients, k)
							}
						}
					}

					c.JSON(http.StatusTooManyRequests, gin.H{
						"error":             "Burst rate limit exceeded",
						"retry_after":       retryAfter,
						"rate_limit":        rateLimit,
						"burst_limit":       burstLimit,
						"rate_limit_window": fmt.Sprintf("%d seconds", windowSeconds),
						"window_start":      cli.windowStart.Unix(),
						"window_end":        cli.resetTime.Unix(),
					})
					c.Abort()
					return
				} else if cli.requests >= rateLimit {
					// Regular rate limit exceeded
					retryAfter := int(cli.resetTime.Sub(now).Seconds())

					// Set standard rate limit headers
					c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimit))
					c.Header("X-RateLimit-Remaining", "0")
					c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", cli.resetTime.Unix()))
					c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))

					// Clean up old entries periodically
					if len(clients) > 10000 {
						for k, v := range clients {
							if now.After(v.resetTime.Add(5 * time.Minute)) {
								delete(clients, k)
							}
						}
					}

					c.JSON(http.StatusTooManyRequests, gin.H{
						"error":             "Rate limit exceeded",
						"retry_after":       retryAfter,
						"rate_limit":        rateLimit,
						"burst_limit":       burstLimit,
						"rate_limit_window": fmt.Sprintf("%d seconds", windowSeconds),
						"window_start":      cli.windowStart.Unix(),
						"window_end":        cli.resetTime.Unix(),
					})
					c.Abort()
					return
				} else {
					// Within limits, increment request count
					cli.requests++
				}
			}
		}

		// Set rate limit headers for successful requests
		remaining := max(0, rateLimit-cli.requests)

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", cli.resetTime.Unix()))

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

// requireRoleMiddleware creates middleware that requires specific roles
func (s *Server) requireRoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid user role type"})
			c.Abort()
			return
		}

		// Check if user role is in allowed roles
		if slices.Contains(allowedRoles, roleStr) {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// setupScheduler configures the scheduler with job handlers
func (s *Server) setupScheduler() {
	s.scheduler.SetJobHandler(func(jobType scheduler.JobType, targets []string, options map[string]interface{}) error {
		switch jobType {
		case scheduler.JobTypeVerification:
			return s.handleScheduledVerification(targets, options)
		case scheduler.JobTypeExport:
			return s.handleScheduledExport(targets, options)
		case scheduler.JobTypeCleanup:
			return s.handleScheduledCleanup(targets, options)
		case scheduler.JobTypeReport:
			return s.handleScheduledReport(targets, options)
		default:
			return fmt.Errorf("unknown job type: %s", jobType)
		}
	})
}

// handleScheduledVerification executes scheduled model verification
func (s *Server) handleScheduledVerification(targets []string, options map[string]interface{}) error {
	log.Printf("Executing scheduled verification for targets: %v", targets)

	// If targets contains "all", verify all models
	if len(targets) == 1 && targets[0] == "all" {
		models, err := s.database.ListModels(map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("failed to list models: %w", err)
		}

		for _, model := range models {
			if model.VerificationStatus == "verified" || model.VerificationStatus == "pending" {
				go func(m *database.Model) {
					// Trigger verification for this model
					if err := s.verifyModelHelper(m); err != nil {
						log.Printf("Scheduled verification failed for model %s: %v", m.Name, err)
					}
				}(model)
			}
		}
	} else {
		// Verify specific models
		for _, target := range targets {
			if modelID, err := strconv.ParseInt(target, 10, 64); err == nil {
				model, err := s.database.GetModel(modelID)
				if err != nil {
					log.Printf("Failed to get model %s: %v", target, err)
					continue
				}

				go func(m *database.Model) {
					if err := s.verifyModelHelper(m); err != nil {
						log.Printf("Scheduled verification failed for model %s: %v", m.Name, err)
					}
				}(model)
			}
		}
	}

	return nil
}

// handleScheduledExport executes scheduled configuration export
func (s *Server) handleScheduledExport(targets []string, options map[string]interface{}) error {
	log.Printf("Executing scheduled export for targets: %v", targets)

	// Export configurations for all supported formats
	formats := []string{"opencode", "claude", "crush", "vscode", "json", "yaml"}

	for _, format := range formats {
		_, err := s.exportConfiguration(format)
		if err != nil {
			log.Printf("Failed to export configuration in format %s: %v", format, err)
			// Continue with other formats
		}
	}

	return nil
}

// handleScheduledCleanup executes scheduled cleanup tasks
func (s *Server) handleScheduledCleanup(targets []string, options map[string]interface{}) error {
	log.Printf("Executing scheduled cleanup for targets: %v", targets)

	// Clean up old verification results (older than 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	// This would require additional database cleanup methods
	// For now, just log the intent
	log.Printf("Scheduled cleanup would remove old data older than %v", thirtyDaysAgo)

	return nil
}

// handleScheduledReport executes scheduled report generation
func (s *Server) handleScheduledReport(targets []string, options map[string]interface{}) error {
	log.Printf("Executing scheduled report generation for targets: %v", targets)

	// Generate comprehensive report
	reportData := map[string]interface{}{
		"generated_at": time.Now(),
		"report_type":  "scheduled",
		"targets":      targets,
	}

	// This would generate and save a report
	// For now, just log the intent
	log.Printf("Scheduled report generated: %v", reportData)

	return nil
}

// verifyModelHelper is a helper function for scheduled verification
func (s *Server) verifyModelHelper(model *database.Model) error {
	// Get provider
	provider, err := s.database.GetProvider(model.ProviderID)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// Create temporary config
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

	// Create temporary verifier
	tempVerifier := llmverifier.New(tempConfig)
	results, err := tempVerifier.Verify()
	if err != nil {
		return err
	}

	if len(results) > 0 {
		result := results[0]
		// Save result (simplified version)
		log.Printf("Verification completed for model %s: score %.2f", model.Name, result.PerformanceScores.OverallScore)
	}

	return nil
}

// Shutdown gracefully shuts down the server and its components
func (s *Server) Shutdown() {
	log.Println("Shutting down LLM Verifier API server...")

	// Shutdown scheduler
	if s.scheduler != nil {
		s.scheduler.Stop()
	}

	// Shutdown notification manager
	if s.notificationMgr != nil {
		s.notificationMgr.Shutdown()
	}

	// Shutdown event bus
	if s.eventBus != nil {
		s.eventBus.Shutdown()
	}

	// Shutdown health checker
	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}

	log.Println("LLM Verifier API server shutdown complete")
}

// Start starts the HTTP server
func (s *Server) Start(port string) error {
	log.Printf("Starting LLM Verifier API server on port %s", port)
	return s.router.Run(":" + port)
}

// Router returns the Gin router for testing
func (s *Server) Router() *gin.Engine {
	return s.router
}
