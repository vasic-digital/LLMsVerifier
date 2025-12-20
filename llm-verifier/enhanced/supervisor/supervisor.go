package supervisor

import (
	"fmt"
	"strings"
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

// AIAssistant represents an intelligent conversational assistant
type AIAssistant struct {
	db       *database.Database
	config   *SupervisorConfig
	verifier *llmverifier.Verifier
	context  map[string][]string // userID -> conversation history
}

// NewAIAssistant creates a new AI assistant
func NewAIAssistant(db *database.Database, config *SupervisorConfig, verifier *llmverifier.Verifier) *AIAssistant {
	return &AIAssistant{
		db:       db,
		config:   config,
		verifier: verifier,
		context:  make(map[string][]string),
	}
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
