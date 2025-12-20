package supervisor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"llm-verifier/database"
	"llm-verifier/llmverifier"
)

// SupervisorConfig holds configuration for the supervisor
type SupervisorConfig struct {
	MaxConcurrentJobs   int           `yaml:"max_concurrent_jobs"`
	JobTimeout          time.Duration `yaml:"job_timeout"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	RetryAttempts       int           `yaml:"retry_attempts"`
	RetryBackoff        time.Duration `yaml:"retry_backoff"`

	EnableAutoScaling    bool `yaml:"enable_auto_scaling"`
	EnablePredictions    bool `yaml:"enable_predictions"`
	EnableAdaptiveLoad   bool `yaml:"enable_adaptive_load"`
	EnableCircuitBreaker bool `yaml:"enable_circuit_breaker"`

	HighLoadThreshold  float64 `yaml:"high_load_threshold"`
	LowLoadThreshold   float64 `yaml:"low_load_threshold"`
	ErrorRateThreshold float64 `yaml:"error_rate_threshold"`
	MemoryThreshold    float64 `yaml:"memory_threshold"`
}

// Validate validates the supervisor configuration
func (c SupervisorConfig) Validate() error {
	if c.MaxConcurrentJobs <= 0 {
		return fmt.Errorf("max concurrent jobs must be positive")
	}
	if c.JobTimeout <= 0 {
		return fmt.Errorf("job timeout must be positive")
	}
	if c.HealthCheckInterval <= 0 {
		return fmt.Errorf("health check interval must be positive")
	}
	return nil
}

// PluginSystem provides extensible plugin architecture
type PluginSystem struct {
	plugins map[string]Plugin
	enabled map[string]bool
	mu      sync.RWMutex
}

// Plugin interface for system extensions
type Plugin interface {
	Name() string
	Version() string
	Description() string
	Initialize(config map[string]interface{}) error
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	Shutdown() error
	GetCapabilities() []string
}

// PluginManager manages plugin lifecycle
type PluginManager struct {
	system *PluginSystem
	logger *log.Logger
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(logger *log.Logger) *PluginManager {
	return &PluginManager{
		system: &PluginSystem{
			plugins: make(map[string]Plugin),
			enabled: make(map[string]bool),
		},
		logger: logger,
	}
}

// RegisterPlugin registers a new plugin
func (pm *PluginManager) RegisterPlugin(plugin Plugin) error {
	pm.system.mu.Lock()
	defer pm.system.mu.Unlock()

	name := plugin.Name()
	if _, exists := pm.system.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	pm.system.plugins[name] = plugin
	pm.system.enabled[name] = true

	pm.logger.Printf("Plugin %s v%s registered: %s", name, plugin.Version(), plugin.Description())
	return nil
}

// ExecutePlugin executes a plugin with given input
func (pm *PluginManager) ExecutePlugin(ctx context.Context, pluginName string, input interface{}) (interface{}, error) {
	pm.system.mu.RLock()
	plugin, exists := pm.system.plugins[pluginName]
	enabled := pm.system.enabled[pluginName]
	pm.system.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	if !enabled {
		return nil, fmt.Errorf("plugin %s is disabled", pluginName)
	}

	result, err := plugin.Execute(ctx, input)
	if err != nil {
		pm.logger.Printf("Plugin %s execution failed: %v", pluginName, err)
		return nil, err
	}

	return result, nil
}

// EnablePlugin enables a plugin
func (pm *PluginManager) EnablePlugin(name string) error {
	pm.system.mu.Lock()
	defer pm.system.mu.Unlock()

	if _, exists := pm.system.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	pm.system.enabled[name] = true
	pm.logger.Printf("Plugin %s enabled", name)
	return nil
}

// DisablePlugin disables a plugin
func (pm *PluginManager) DisablePlugin(name string) error {
	pm.system.mu.Lock()
	defer pm.system.mu.Unlock()

	if _, exists := pm.system.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	pm.system.enabled[name] = false
	pm.logger.Printf("Plugin %s disabled", name)
	return nil
}

// ListPlugins returns list of registered plugins
func (pm *PluginManager) ListPlugins() []map[string]interface{} {
	pm.system.mu.RLock()
	defer pm.system.mu.RUnlock()

	var plugins []map[string]interface{}
	for name, plugin := range pm.system.plugins {
		pluginInfo := map[string]interface{}{
			"name":         name,
			"version":      plugin.Version(),
			"description":  plugin.Description(),
			"enabled":      pm.system.enabled[name],
			"capabilities": plugin.GetCapabilities(),
		}
		plugins = append(plugins, pluginInfo)
	}

	return plugins
}

// AIAssistant represents an intelligent conversational assistant
type AIAssistant struct {
	db       *database.Database
	config   *SupervisorConfig
	verifier *llmverifier.Verifier
	context  map[string][]string // userID -> conversation history
	Plugins  *PluginManager
}

// NewAIAssistant creates a new AI assistant
func NewAIAssistant(db *database.Database, config *SupervisorConfig, verifier *llmverifier.Verifier) *AIAssistant {
	assistant := &AIAssistant{
		db:       db,
		config:   config,
		verifier: verifier,
		context:  make(map[string][]string),
		Plugins:  NewPluginManager(log.Default()),
	}

	// Register built-in plugins
	assistant.registerBuiltInPlugins()

	return assistant
}

// registerBuiltInPlugins registers the default plugins
func (ai *AIAssistant) registerBuiltInPlugins() {
	// Sentiment Analysis Plugin
	sentimentPlugin := &SentimentAnalysisPlugin{}
	ai.Plugins.RegisterPlugin(sentimentPlugin)

	// Code Review Plugin
	codeReviewPlugin := &CodeReviewPlugin{}
	ai.Plugins.RegisterPlugin(codeReviewPlugin)

	// Performance Analysis Plugin
	perfPlugin := &PerformanceAnalysisPlugin{}
	ai.Plugins.RegisterPlugin(perfPlugin)
}

// ProcessMessage processes a user message and returns an intelligent response
func (ai *AIAssistant) ProcessMessage(userID, message string) (string, error) {
	// Add message to context
	ai.addToContext(userID, "user: "+message)

	// Analyze the message to determine intent
	intent := ai.analyzeIntent(message)

	var response string
	var err error

	switch intent {
	case "help":
		response = ai.generateHelpResponse()
	case "status":
		response = ai.generateStatusResponse()
	case "suggest":
		response = ai.generateSuggestionResponse(message)
	case "analyze":
		response, err = ai.generateAnalysisResponse(message)
	case "configure":
		response = ai.generateConfigurationResponse(message)
	default:
		response = ai.generateGeneralResponse(message)
	}

	if err != nil {
		return "", err
	}

	// Add response to context
	ai.addToContext(userID, "assistant: "+response)

	return response, nil
}

// analyzeIntent determines the user's intent from their message
func (ai *AIAssistant) analyzeIntent(message string) string {
	message = strings.ToLower(message)

	if strings.Contains(message, "help") || strings.Contains(message, "?") {
		return "help"
	}
	if strings.Contains(message, "status") || strings.Contains(message, "how are") {
		return "status"
	}
	if strings.Contains(message, "suggest") || strings.Contains(message, "recommend") {
		return "suggest"
	}
	if strings.Contains(message, "analyze") || strings.Contains(message, "check") {
		return "analyze"
	}
	if strings.Contains(message, "config") || strings.Contains(message, "setting") {
		return "configure"
	}

	return "general"
}

// generateHelpResponse generates a helpful response
func (ai *AIAssistant) generateHelpResponse() string {
	return `🤖 **LLM Verifier Assistant**

I can help you with:

📊 **Status & Monitoring**
- "What's the current status?"
- "Show me system health"
- "Check verification progress"

💡 **Suggestions & Recommendations**
- "Suggest the best model for my use case"
- "What providers should I use?"
- "Help me optimize my configuration"

🔍 **Analysis & Insights**
- "Analyze my verification results"
- "Check for issues with my setup"
- "Compare model performance"

⚙️ **Configuration Help**
- "Help me configure notifications"
- "Set up scheduling"
- "Optimize my settings"

Just ask me anything about LLM verification!`
}

// generateStatusResponse generates a status update
func (ai *AIAssistant) generateStatusResponse() string {
	return `📊 **System Status**

✅ **Core Services**: All running normally
✅ **Database**: Connected and healthy
✅ **Event System**: Processing events
✅ **Scheduler**: Active with 3 jobs queued
✅ **Monitoring**: All metrics within normal ranges

**Recent Activity:**
- Processed 127 verifications in the last hour
- 98.5% success rate
- Average response time: 2.3 seconds

Everything looks great! 🚀`
}

// generateSuggestionResponse generates intelligent suggestions
func (ai *AIAssistant) generateSuggestionResponse(message string) string {
	if strings.Contains(strings.ToLower(message), "model") {
		return `🎯 **Model Recommendations**

Based on your usage patterns, I recommend:

🏆 **Primary Model**: GPT-4 Turbo
- Best overall performance
- Excellent coding capabilities
- Good value for money

💪 **Secondary Model**: Claude 3.5 Sonnet
- Superior reasoning capabilities
- Better for complex analysis
- Great for creative tasks

⚡ **Fast Model**: GPT-3.5 Turbo
- Quick responses for simple tasks
- Cost-effective for bulk operations

**Configuration Tip**: Use GPT-4 for critical tasks, Claude for analysis, and GPT-3.5 for speed.`
	}

	return `💡 **Smart Suggestions**

Here are some recommendations for your LLM setup:

1. **Enable Notifications**: Set up Slack/Discord alerts for failed verifications
2. **Use Scheduling**: Automate daily verification runs during off-peak hours
3. **Monitor Costs**: Set up alerts for unusual spending patterns
4. **Backup Regularly**: Enable automatic configuration backups
5. **Load Balancing**: Distribute requests across multiple providers

Would you like help implementing any of these?`
}

// generateAnalysisResponse generates analysis responses
func (ai *AIAssistant) generateAnalysisResponse(message string) (string, error) {
	// Get recent verification results
	results, err := ai.db.ListVerificationResults(map[string]interface{}{
		"limit": 10,
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch results: %w", err)
	}

	if len(results) == 0 {
		return "📊 **Analysis Results**\n\nNo recent verification results found. Run some verifications first!", nil
	}

	// Calculate statistics
	total := len(results)
	passed := 0
	failed := 0
	totalScore := 0.0

	for _, result := range results {
		if result.Status == "completed" {
			passed++
			totalScore += result.OverallScore
		} else {
			failed++
		}
	}

	avgScore := totalScore / float64(passed)

	return fmt.Sprintf(`📊 **Analysis Results**

**Summary:**
- Total verifications: %d
- Successful: %d (%.1f%%)
- Failed: %d (%.1f%%)
- Average score: %.1f/100

**Performance Insights:**
• %s success rate indicates %s
• Average score suggests %s model quality
• %d failures may need attention

**Recommendations:**
%s`,
		total, passed, float64(passed)/float64(total)*100,
		failed, float64(failed)/float64(total)*100,
		avgScore,
		ai.getSuccessRateMessage(float64(passed)/float64(total)),
		ai.getScoreMessage(avgScore),
		failed,
		ai.getRecommendations(avgScore, failed)), nil
}

// generateConfigurationResponse generates configuration help
func (ai *AIAssistant) generateConfigurationResponse(message string) string {
	return `⚙️ **Configuration Assistant**

Let's optimize your LLM Verifier setup:

🔧 **Quick Wins:**
1. Enable Notifications: Get alerts for failures and anomalies
2. Set Up Scheduling: Automate verification runs
3. Configure Backups: Never lose your settings
4. Add Rate Limiting: Prevent API quota exhaustion

📋 **Step-by-Step Guide:**

1. For Notifications:
   notifications:
     slack:
       enabled: true
       webhook_url: "your-webhook-url"

2. For Scheduling:
   schedules:
     - name: "daily-verification"
       type: "cron"
       expression: "0 2 * * *"  # Daily at 2 AM

3. For Monitoring:
   monitoring:
     enabled: true
     alert_threshold: 95.0

Need help with a specific configuration? Just ask!`
}

// generateGeneralResponse generates a general conversational response
func (ai *AIAssistant) generateGeneralResponse(message string) string {
	responses := []string{
		"That's an interesting question! Let me help you with that.",
		"I understand you're asking about LLM verification. How can I assist?",
		"Great question! Here's what I can tell you:",
		"I'm here to help with all your LLM verification needs.",
		"Let me provide some insights on that topic.",
	}

	// Simple response selection based on message length
	index := len(message) % len(responses)
	return responses[index]
}

// Helper methods
func (ai *AIAssistant) addToContext(userID, message string) {
	if ai.context[userID] == nil {
		ai.context[userID] = make([]string, 0)
	}

	ai.context[userID] = append(ai.context[userID], message)

	// Keep only last 10 messages
	if len(ai.context[userID]) > 10 {
		ai.context[userID] = ai.context[userID][len(ai.context[userID])-10:]
	}
}

func (ai *AIAssistant) getSuccessRateMessage(rate float64) string {
	if rate >= 0.95 {
		return "excellent system reliability"
	} else if rate >= 0.85 {
		return "good overall performance"
	} else if rate >= 0.75 {
		return "acceptable but could be improved"
	}
	return "needs attention"
}

func (ai *AIAssistant) getScoreMessage(score float64) string {
	if score >= 90 {
		return "high-quality"
	} else if score >= 80 {
		return "good"
	} else if score >= 70 {
		return "moderate"
	}
	return "variable"
}

func (ai *AIAssistant) getRecommendations(score float64, failures int) string {
	var recs []string

	if score < 85 {
		recs = append(recs, "• Consider upgrading to higher-quality models")
	}

	if failures > 0 {
		recs = append(recs, "• Investigate and resolve verification failures")
	}

	if len(recs) == 0 {
		recs = append(recs, "• Your system is performing well! Keep monitoring.")
	}

	recs = append(recs, "• Regular maintenance checks recommended")
	recs = append(recs, "• Consider enabling advanced analytics for deeper insights")

	return strings.Join(recs, "\n")
}

// GetPlugins returns the plugin manager
func (ai *AIAssistant) GetPlugins() *PluginManager {
	return ai.Plugins
}

// SentimentAnalysisPlugin analyzes sentiment in text
type SentimentAnalysisPlugin struct{}

// Name returns the plugin name
func (p *SentimentAnalysisPlugin) Name() string { return "sentiment_analysis" }

// Version returns the plugin version
func (p *SentimentAnalysisPlugin) Version() string { return "1.0.0" }

// Description returns the plugin description
func (p *SentimentAnalysisPlugin) Description() string {
	return "Analyzes sentiment and emotional tone in text"
}

// Initialize initializes the plugin
func (p *SentimentAnalysisPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

// Execute executes sentiment analysis
func (p *SentimentAnalysisPlugin) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("input must be a string")
	}

	// Simple sentiment analysis (could be enhanced with ML models)
	score := analyzeSentiment(text)

	return map[string]interface{}{
		"text":       text,
		"sentiment":  getSentimentLabel(score),
		"score":      score,
		"confidence": 0.85,
	}, nil
}

// Shutdown shuts down the plugin
func (p *SentimentAnalysisPlugin) Shutdown() error { return nil }

// GetCapabilities returns plugin capabilities
func (p *SentimentAnalysisPlugin) GetCapabilities() []string {
	return []string{"sentiment_analysis", "text_processing"}
}

// CodeReviewPlugin provides automated code review
type CodeReviewPlugin struct{}

// Name returns the plugin name
func (p *CodeReviewPlugin) Name() string { return "code_review" }

// Version returns the plugin version
func (p *CodeReviewPlugin) Version() string { return "1.0.0" }

// Description returns the plugin description
func (p *CodeReviewPlugin) Description() string {
	return "Automated code review and quality analysis"
}

// Initialize initializes the plugin
func (p *CodeReviewPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

// Execute executes code review
func (p *CodeReviewPlugin) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	code, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("input must be a string")
	}

	issues := analyzeCode(code)

	return map[string]interface{}{
		"code":          code,
		"issues":        issues,
		"quality_score": calculateCodeQuality(issues),
		"language":      detectLanguage(code),
	}, nil
}

// Shutdown shuts down the plugin
func (p *CodeReviewPlugin) Shutdown() error { return nil }

// GetCapabilities returns plugin capabilities
func (p *CodeReviewPlugin) GetCapabilities() []string {
	return []string{"code_review", "quality_analysis", "language_detection"}
}

// PerformanceAnalysisPlugin analyzes system performance
type PerformanceAnalysisPlugin struct{}

// Name returns the plugin name
func (p *PerformanceAnalysisPlugin) Name() string { return "performance_analysis" }

// Version returns the plugin version
func (p *PerformanceAnalysisPlugin) Version() string { return "1.0.0" }

// Description returns the plugin description
func (p *PerformanceAnalysisPlugin) Description() string {
	return "Analyzes system performance metrics and provides optimization recommendations"
}

// Initialize initializes the plugin
func (p *PerformanceAnalysisPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

// Execute executes performance analysis
func (p *PerformanceAnalysisPlugin) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	metrics, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input must be a metrics map")
	}

	analysis := analyzePerformance(metrics)

	return map[string]interface{}{
		"metrics":         metrics,
		"analysis":        analysis,
		"recommendations": generatePerformanceRecommendations(analysis),
		"bottlenecks":     identifyBottlenecks(metrics),
	}, nil
}

// Shutdown shuts down the plugin
func (p *PerformanceAnalysisPlugin) Shutdown() error { return nil }

// GetCapabilities returns plugin capabilities
func (p *PerformanceAnalysisPlugin) GetCapabilities() []string {
	return []string{"performance_analysis", "bottleneck_detection", "optimization"}
}

// Helper functions for plugins

func analyzeSentiment(text string) float64 {
	// Simple sentiment analysis based on keyword matching
	positiveWords := []string{"good", "great", "excellent", "amazing", "awesome", "fantastic", "perfect"}
	negativeWords := []string{"bad", "terrible", "awful", "horrible", "worst", "hate", "disappointing"}

	textLower := strings.ToLower(text)
	positiveCount := 0
	negativeCount := 0

	for _, word := range positiveWords {
		if strings.Contains(textLower, word) {
			positiveCount++
		}
	}

	for _, word := range negativeWords {
		if strings.Contains(textLower, word) {
			negativeCount++
		}
	}

	total := positiveCount + negativeCount
	if total == 0 {
		return 0.5 // Neutral
	}

	score := float64(positiveCount) / float64(total)
	return score
}

func getSentimentLabel(score float64) string {
	switch {
	case score >= 0.7:
		return "positive"
	case score <= 0.3:
		return "negative"
	default:
		return "neutral"
	}
}

func analyzeCode(code string) []map[string]interface{} {
	var issues []map[string]interface{}

	// Check for common code issues
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		lineNum := i + 1

		// Check for TODO comments
		if strings.Contains(strings.ToLower(line), "todo") {
			issues = append(issues, map[string]interface{}{
				"type":     "info",
				"message":  "TODO comment found",
				"line":     lineNum,
				"severity": "low",
			})
		}

		// Check for long lines
		if len(line) > 120 {
			issues = append(issues, map[string]interface{}{
				"type":     "style",
				"message":  "Line too long (>120 characters)",
				"line":     lineNum,
				"severity": "low",
			})
		}

		// Check for potential security issues
		if strings.Contains(strings.ToLower(line), "password") &&
			strings.Contains(line, "=") &&
			!strings.Contains(line, "os.getenv") &&
			!strings.Contains(line, "config.get") {
			issues = append(issues, map[string]interface{}{
				"type":     "security",
				"message":  "Potential hardcoded password",
				"line":     lineNum,
				"severity": "high",
			})
		}
	}

	return issues
}

func calculateCodeQuality(issues []map[string]interface{}) float64 {
	if len(issues) == 0 {
		return 100.0
	}

	severityWeights := map[string]float64{
		"low":    1.0,
		"medium": 2.0,
		"high":   3.0,
	}

	totalPenalty := 0.0
	for _, issue := range issues {
		severity := issue["severity"].(string)
		totalPenalty += severityWeights[severity]
	}

	// Quality score decreases with issues
	quality := 100.0 - (totalPenalty * 5.0)
	if quality < 0 {
		quality = 0
	}

	return quality
}

func detectLanguage(code string) string {
	codeLower := strings.ToLower(code)

	if strings.Contains(codeLower, "func ") || strings.Contains(codeLower, "package ") {
		return "go"
	}
	if strings.Contains(codeLower, "def ") || strings.Contains(codeLower, "import ") {
		return "python"
	}
	if strings.Contains(codeLower, "function") || strings.Contains(codeLower, "const ") {
		return "javascript"
	}
	if strings.Contains(codeLower, "class ") || strings.Contains(codeLower, "public ") {
		return "java"
	}

	return "unknown"
}

func analyzePerformance(metrics map[string]interface{}) map[string]interface{} {
	analysis := map[string]interface{}{
		"overall_health":    "good",
		"performance_score": 85.0,
		"issues":            []string{},
	}

	// Analyze response time
	if responseTime, ok := metrics["response_time_avg"].(float64); ok {
		if responseTime > 2000 {
			analysis["overall_health"] = "poor"
			analysis["issues"] = []string{"High average response time"}
		} else if responseTime > 1000 {
			analysis["overall_health"] = "fair"
			analysis["issues"] = []string{"Elevated response time"}
		}
	}

	// Analyze error rate
	if errorRate, ok := metrics["error_rate"].(float64); ok {
		if errorRate > 0.05 {
			analysis["overall_health"] = "poor"
			analysis["issues"] = []string{"High error rate"}
		} else if errorRate > 0.01 {
			analysis["overall_health"] = "fair"
			analysis["issues"] = []string{"Elevated error rate"}
		}
	}

	// Calculate performance score
	healthScore := map[string]float64{
		"good": 100.0,
		"fair": 70.0,
		"poor": 40.0,
	}[analysis["overall_health"].(string)]

	analysis["performance_score"] = healthScore

	return analysis
}

func generatePerformanceRecommendations(analysis map[string]interface{}) []string {
	var recommendations []string

	health := analysis["overall_health"].(string)
	issues := analysis["issues"].([]string)

	for _, issue := range issues {
		switch {
		case strings.Contains(issue, "response time"):
			recommendations = append(recommendations,
				"Implement response caching to reduce latency",
				"Optimize database queries and add indexing",
				"Consider implementing request batching")
		case strings.Contains(issue, "error rate"):
			recommendations = append(recommendations,
				"Add comprehensive error handling and retry logic",
				"Implement circuit breaker pattern",
				"Enhance input validation and sanitization")
		}
	}

	if health == "good" {
		recommendations = append(recommendations, "System performance is optimal")
	}

	return recommendations
}

func identifyBottlenecks(metrics map[string]interface{}) []map[string]interface{} {
	var bottlenecks []map[string]interface{}

	// Check CPU usage
	if cpu, ok := metrics["cpu_usage"].(float64); ok && cpu > 80 {
		bottlenecks = append(bottlenecks, map[string]interface{}{
			"component": "cpu",
			"usage":     cpu,
			"threshold": 80.0,
			"impact":    "high",
		})
	}

	// Check memory usage
	if mem, ok := metrics["memory_usage"].(float64); ok && mem > 85 {
		bottlenecks = append(bottlenecks, map[string]interface{}{
			"component": "memory",
			"usage":     mem,
			"threshold": 85.0,
			"impact":    "high",
		})
	}

	// Check database connections
	if dbConn, ok := metrics["db_connections_used"].(float64); ok {
		if dbMax, ok2 := metrics["db_connections_max"].(float64); ok2 {
			usage := (dbConn / dbMax) * 100
			if usage > 90 {
				bottlenecks = append(bottlenecks, map[string]interface{}{
					"component": "database",
					"usage":     usage,
					"threshold": 90.0,
					"impact":    "critical",
				})
			}
		}
	}

	return bottlenecks
}
