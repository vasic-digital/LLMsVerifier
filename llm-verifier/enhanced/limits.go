package enhanced

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"llm-verifier/database"
)

// LimitsDetector detects rate limits and quotas for different providers
type LimitsDetector struct {
	httpClient *http.Client
}

// NewLimitsDetector creates a new limits detector
func NewLimitsDetector() *LimitsDetector {
	return &LimitsDetector{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LimitsInfo represents rate limit and quota information
type LimitsInfo struct {
	RequestsPerMinute *int                   `json:"requests_per_minute,omitempty"`
	RequestsPerHour   *int                   `json:"requests_per_hour,omitempty"`
	RequestsPerDay    *int                   `json:"requests_per_day,omitempty"`
	TokensPerMinute   *int                   `json:"tokens_per_minute,omitempty"`
	TokensPerHour     *int                   `json:"tokens_per_hour,omitempty"`
	TokensPerDay      *int                   `json:"tokens_per_day,omitempty"`
	CurrentUsage      map[string]int         `json:"current_usage,omitempty"`
	ResetPeriod       string                 `json:"reset_period,omitempty"`
	ResetTime         *time.Time             `json:"reset_time,omitempty"`
	IsHardLimit       bool                   `json:"is_hard_limit"`
	AdditionalLimits  map[string]interface{} `json:"additional_limits,omitempty"`
}

// DetectLimits detects rate limits and quotas for a given provider and model
func (ld *LimitsDetector) DetectLimits(providerName, modelID string, headers http.Header) (*LimitsInfo, error) {
	switch strings.ToLower(providerName) {
	case "openai":
		return ld.detectOpenAILimits(headers)
	case "anthropic":
		return ld.detectAnthropicLimits(headers)
	case "azure", "azure openai":
		return ld.detectAzureOpenAILimits(headers)
	case "google", "google cloud", "gcp":
		return ld.detectGoogleLimits(headers)
	case "cohere":
		return ld.detectCohereLimits(headers)
	case "groq":
		return ld.detectGroqLimits(headers)
	case "togetherai":
		return ld.detectTogetherAILimits(headers)
	case "fireworks":
		return ld.detectFireworksLimits(headers)
	case "poe":
		return ld.detectPoeLimits(headers)
	case "navigator":
		return ld.detectNavigatorLimits(headers)
	case "mistral":
		return ld.detectMistralLimits(headers)
	default:
		return ld.detectGenericLimits(headers)
	}
}

// detectOpenAILimits detects limits for OpenAI
func (ld *LimitsDetector) detectOpenAILimits(headers http.Header) (*LimitsInfo, error) {
	limits := &LimitsInfo{
		IsHardLimit:      true,
		CurrentUsage:     make(map[string]int),
		AdditionalLimits: make(map[string]interface{}),
	}

	// OpenAI uses standard rate limit headers
	if rpm := headers.Get("x-ratelimit-limit-requests"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil {
			limits.RequestsPerMinute = &val
		}
	}

	if tpm := headers.Get("x-ratelimit-limit-tokens"); tpm != "" {
		if val, err := strconv.Atoi(tpm); err == nil {
			limits.TokensPerMinute = &val
		}
	}

	// Current usage
	if used := headers.Get("x-ratelimit-remaining-requests"); used != "" {
		if val, err := strconv.Atoi(used); err == nil && limits.RequestsPerMinute != nil {
			limits.CurrentUsage["requests"] = *limits.RequestsPerMinute - val
		}
	}

	if used := headers.Get("x-ratelimit-remaining-tokens"); used != "" {
		if val, err := strconv.Atoi(used); err == nil && limits.TokensPerMinute != nil {
			limits.CurrentUsage["tokens"] = *limits.TokensPerMinute - val
		}
	}

	// Reset information
	if reset := headers.Get("x-ratelimit-reset"); reset != "" {
		if timestamp, err := strconv.ParseInt(reset, 10, 64); err == nil {
			resetTime := time.Unix(timestamp, 0)
			limits.ResetTime = &resetTime
		}
	}

	return limits, nil
}

// detectAnthropicLimits detects limits for Anthropic
func (ld *LimitsDetector) detectAnthropicLimits(headers http.Header) (*LimitsInfo, error) {
	limits := &LimitsInfo{
		IsHardLimit:      true,
		CurrentUsage:     make(map[string]int),
		AdditionalLimits: make(map[string]interface{}),
	}

	// Anthropic uses custom rate limit headers
	if rpm := headers.Get("anthropic-ratelimit-requests-limit"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil {
			limits.RequestsPerHour = &val // Anthropic typically uses hourly limits
		}
	}

	if tpm := headers.Get("anthropic-ratelimit-tokens-limit"); tpm != "" {
		if val, err := strconv.Atoi(tpm); err == nil {
			limits.TokensPerHour = &val
		}
	}

	// Current usage
	if used := headers.Get("anthropic-ratelimit-requests-remaining"); used != "" {
		if val, err := strconv.Atoi(used); err == nil && limits.RequestsPerHour != nil {
			limits.CurrentUsage["requests"] = *limits.RequestsPerHour - val
		}
	}

	if used := headers.Get("anthropic-ratelimit-tokens-remaining"); used != "" {
		if val, err := strconv.Atoi(used); err == nil && limits.TokensPerHour != nil {
			limits.CurrentUsage["tokens"] = *limits.TokensPerHour - val
		}
	}

	// Reset information
	if reset := headers.Get("anthropic-ratelimit-reset"); reset != "" {
		if timestamp, err := time.Parse(time.RFC3339, reset); err == nil {
			limits.ResetTime = &timestamp
		}
	}

	return limits, nil
}

// detectAzureOpenAILimits detects limits for Azure OpenAI
func (ld *LimitsDetector) detectAzureOpenAILimits(headers http.Header) (*LimitsInfo, error) {
	limits := &LimitsInfo{
		IsHardLimit:      true,
		CurrentUsage:     make(map[string]int),
		AdditionalLimits: make(map[string]interface{}),
	}

	// Azure OpenAI uses standard rate limit headers but with different values
	if rpm := headers.Get("x-rate-limit-limit"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil {
			limits.RequestsPerMinute = &val
		}
	}

	if tpm := headers.Get("x-rate-limit-limit-tokens"); tpm != "" {
		if val, err := strconv.Atoi(tpm); err == nil {
			limits.TokensPerMinute = &val
		}
	}

	// Current usage
	if used := headers.Get("x-rate-limit-remaining"); used != "" {
		if val, err := strconv.Atoi(used); err == nil && limits.RequestsPerMinute != nil {
			limits.CurrentUsage["requests"] = *limits.RequestsPerMinute - val
		}
	}

	// Azure-specific limits
	if tpmUsed := headers.Get("x-rate-limit-remaining-tokens"); tpmUsed != "" {
		if val, err := strconv.Atoi(tpmUsed); err == nil && limits.TokensPerMinute != nil {
			limits.CurrentUsage["tokens"] = *limits.TokensPerMinute - val
		}
	}

	return limits, nil
}

// detectGoogleLimits detects limits for Google Cloud
func (ld *LimitsDetector) detectGoogleLimits(headers http.Header) (*LimitsInfo, error) {
	limits := &LimitsInfo{
		IsHardLimit:      true,
		CurrentUsage:     make(map[string]int),
		AdditionalLimits: make(map[string]interface{}),
	}

	// Google Cloud uses quota project headers
	if quota := headers.Get("x-goog-quota-user"); quota != "" {
		limits.AdditionalLimits["quota_user"] = quota
	}

	// Rate limit information (if available)
	if rpm := headers.Get("x-rate-limit-limit"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil {
			limits.RequestsPerMinute = &val
		}
	}

	if tpm := headers.Get("x-rate-limit-limit-tokens"); tpm != "" {
		if val, err := strconv.Atoi(tpm); err == nil {
			limits.TokensPerMinute = &val
		}
	}

	return limits, nil
}

// detectCohereLimits detects limits for Cohere
func (ld *LimitsDetector) detectCohereLimits(headers http.Header) (*LimitsInfo, error) {
	limits := &LimitsInfo{
		IsHardLimit:      true,
		CurrentUsage:     make(map[string]int),
		AdditionalLimits: make(map[string]interface{}),
	}

	// Cohere uses standard rate limit headers
	if rpm := headers.Get("x-ratelimit-limit"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil {
			limits.RequestsPerMinute = &val
		}
	}

	// Current usage
	if remaining := headers.Get("x-ratelimit-remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil && limits.RequestsPerMinute != nil {
			limits.CurrentUsage["requests"] = *limits.RequestsPerMinute - val
		}
	}

	return limits, nil
}

// detectGroqLimits detects limits for Groq
func (ld *LimitsDetector) detectGroqLimits(headers http.Header) (*LimitsInfo, error) {
	rpm := 30
	limits := &LimitsInfo{
		RequestsPerMinute: &rpm,
		IsHardLimit:       true,
		CurrentUsage:      make(map[string]int),
		AdditionalLimits:  make(map[string]interface{}),
	}
	return limits, nil
}

// detectTogetherAILimits detects limits for Together AI
func (ld *LimitsDetector) detectTogetherAILimits(headers http.Header) (*LimitsInfo, error) {
	rpm := 10
	limits := &LimitsInfo{
		RequestsPerMinute: &rpm,
		IsHardLimit:       true,
		CurrentUsage:      make(map[string]int),
		AdditionalLimits:  make(map[string]interface{}),
	}
	return limits, nil
}

// detectFireworksLimits detects limits for Fireworks AI
func (ld *LimitsDetector) detectFireworksLimits(headers http.Header) (*LimitsInfo, error) {
	rpm := 100
	limits := &LimitsInfo{
		RequestsPerMinute: &rpm,
		IsHardLimit:       true,
		CurrentUsage:      make(map[string]int),
		AdditionalLimits:  make(map[string]interface{}),
	}
	return limits, nil
}

// detectPoeLimits detects limits for Poe
func (ld *LimitsDetector) detectPoeLimits(headers http.Header) (*LimitsInfo, error) {
	rpm := 60
	limits := &LimitsInfo{
		RequestsPerMinute: &rpm,
		IsHardLimit:       true,
		CurrentUsage:      make(map[string]int),
		AdditionalLimits:  make(map[string]interface{}),
	}
	return limits, nil
}

// detectNavigatorLimits detects limits for NaviGator AI
func (ld *LimitsDetector) detectNavigatorLimits(headers http.Header) (*LimitsInfo, error) {
	rpm := 20
	limits := &LimitsInfo{
		RequestsPerMinute: &rpm,
		IsHardLimit:       true,
		CurrentUsage:      make(map[string]int),
		AdditionalLimits:  make(map[string]interface{}),
	}
	return limits, nil
}

// detectMistralLimits detects limits for Mistral
func (ld *LimitsDetector) detectMistralLimits(headers http.Header) (*LimitsInfo, error) {
	rpm := 50
	limits := &LimitsInfo{
		RequestsPerMinute: &rpm,
		IsHardLimit:       true,
		CurrentUsage:      make(map[string]int),
		AdditionalLimits:  make(map[string]interface{}),
	}
	return limits, nil
}

// detectGenericLimits attempts to detect limits from standard headers
func (ld *LimitsDetector) detectGenericLimits(headers http.Header) (*LimitsInfo, error) {
	limits := &LimitsInfo{
		IsHardLimit:      true,
		CurrentUsage:     make(map[string]int),
		AdditionalLimits: make(map[string]interface{}),
	}

	// Standard rate limit headers
	if rpm := headers.Get("x-rate-limit-limit"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil {
			limits.RequestsPerMinute = &val
		}
	}

	if tpm := headers.Get("x-rate-limit-limit-tokens"); tpm != "" {
		if val, err := strconv.Atoi(tpm); err == nil {
			limits.TokensPerMinute = &val
		}
	}

	if rph := headers.Get("x-rate-limit-limit-requests-per-hour"); rph != "" {
		if val, err := strconv.Atoi(rph); err == nil {
			limits.RequestsPerHour = &val
		}
	}

	if rpd := headers.Get("x-rate-limit-limit-requests-per-day"); rpd != "" {
		if val, err := strconv.Atoi(rpd); err == nil {
			limits.RequestsPerDay = &val
		}
	}

	// Current usage
	if remaining := headers.Get("x-rate-limit-remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil && limits.RequestsPerMinute != nil {
			limits.CurrentUsage["requests"] = *limits.RequestsPerMinute - val
		}
	}

	// Reset information
	if reset := headers.Get("x-rate-limit-reset"); reset != "" {
		if timestamp, err := strconv.ParseInt(reset, 10, 64); err == nil {
			resetTime := time.Unix(timestamp, 0)
			limits.ResetTime = &resetTime
		}
	}

	return limits, nil
}

// SaveLimits saves limits information to the database
func SaveLimits(db *database.Database, modelID int64, limits *LimitsInfo) error {
	// Save each limit type separately
	limitTypes := []struct {
		limitType string
		value     *int
		current   int
	}{
		{"requests_per_minute", limits.RequestsPerMinute, limits.CurrentUsage["requests"]},
		{"requests_per_hour", limits.RequestsPerHour, limits.CurrentUsage["requests"]},
		{"requests_per_day", limits.RequestsPerDay, limits.CurrentUsage["requests"]},
		{"tokens_per_minute", limits.TokensPerMinute, limits.CurrentUsage["tokens"]},
		{"tokens_per_hour", limits.TokensPerHour, limits.CurrentUsage["tokens"]},
		{"tokens_per_day", limits.TokensPerDay, limits.CurrentUsage["tokens"]},
	}

	for _, lt := range limitTypes {
		if lt.value != nil && *lt.value > 0 {
			limit := &database.Limit{
				ModelID:      modelID,
				LimitType:    lt.limitType,
				LimitValue:   *lt.value,
				CurrentUsage: lt.current,
				ResetPeriod:  limits.ResetPeriod,
				ResetTime:    limits.ResetTime,
				IsHardLimit:  limits.IsHardLimit,
			}

			if err := db.CreateLimit(limit); err != nil {
				return fmt.Errorf("failed to save %s limit: %w", lt.limitType, err)
			}
		}
	}

	return nil
}

// UpdateLimitsFromHeaders updates limits based on response headers
func (ld *LimitsDetector) UpdateLimitsFromHeaders(db *database.Database, modelID int64, headers http.Header) error {
	// Get provider information from database
	model, err := db.GetModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	provider, err := db.GetProvider(model.ProviderID)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// Detect limits from headers
	limits, err := ld.DetectLimits(provider.Name, model.ModelID, headers)
	if err != nil {
		return fmt.Errorf("failed to detect limits: %w", err)
	}

	// Save detected limits
	return SaveLimits(db, modelID, limits)
}

// CheckRateLimitStatus checks if we're approaching rate limits
func CheckRateLimitStatus(db *database.Database, modelID int64) (map[string]float64, error) {
	limits, err := db.GetLimitsForModel(modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get limits: %w", err)
	}

	status := make(map[string]float64)

	for _, limit := range limits {
		if limit.LimitValue > 0 {
			usagePercentage := float64(limit.CurrentUsage) / float64(limit.LimitValue) * 100.0
			status[limit.LimitType] = usagePercentage
		}
	}

	return status, nil
}

// GetRateLimitAdvice provides advice based on rate limit status
func GetRateLimitAdvice(status map[string]float64) []string {
	var advice []string

	for limitType, percentage := range status {
		switch {
		case percentage >= 90:
			advice = append(advice, fmt.Sprintf("Critical: %s usage at %.1f%% - consider reducing request rate", limitType, percentage))
		case percentage >= 80:
			advice = append(advice, fmt.Sprintf("Warning: %s usage at %.1f%% - monitor closely", limitType, percentage))
		case percentage >= 60:
			advice = append(advice, fmt.Sprintf("Info: %s usage at %.1f%% - normal range", limitType, percentage))
		}
	}

	if len(advice) == 0 {
		advice = append(advice, "Rate limits are within normal ranges")
	}

	return advice
}

// BatchUpdateLimits updates limits for multiple models
func (ld *LimitsDetector) BatchUpdateLimits(db *database.Database, limitsData map[int64]http.Header) error {
	for modelID, headers := range limitsData {
		if err := ld.UpdateLimitsFromHeaders(db, modelID, headers); err != nil {
			// Log error but continue with other models
			fmt.Printf("Warning: Failed to update limits for model %d: %v\n", modelID, err)
		}
	}
	return nil
}

// ResetExpiredLimits resets limits that have expired
func ResetExpiredLimits(db *database.Database) error {
	return db.ResetLimitUsage()
}

// GetLimitComparison compares limits between different models
func GetLimitComparison(db *database.Database, modelIDs []int64) (map[int64][]*database.Limit, error) {
	limitMap := make(map[int64][]*database.Limit)

	for _, modelID := range modelIDs {
		limits, err := db.GetLimitsForModel(modelID)
		if err != nil {
			// Skip models without limits
			continue
		}

		limitMap[modelID] = limits
	}

	return limitMap, nil
}

// ValidateLimits validates limits information
func ValidateLimits(limits *LimitsInfo) error {
	// Check that at least one limit is specified
	hasLimit := false

	if limits.RequestsPerMinute != nil && *limits.RequestsPerMinute > 0 {
		hasLimit = true
	}
	if limits.RequestsPerHour != nil && *limits.RequestsPerHour > 0 {
		hasLimit = true
	}
	if limits.RequestsPerDay != nil && *limits.RequestsPerDay > 0 {
		hasLimit = true
	}
	if limits.TokensPerMinute != nil && *limits.TokensPerMinute > 0 {
		hasLimit = true
	}
	if limits.TokensPerHour != nil && *limits.TokensPerHour > 0 {
		hasLimit = true
	}
	if limits.TokensPerDay != nil && *limits.TokensPerDay > 0 {
		hasLimit = true
	}

	if !hasLimit {
		return fmt.Errorf("at least one limit must be specified")
	}

	// Validate current usage doesn't exceed limits
	for limitType, usage := range limits.CurrentUsage {
		var limit *int

		switch limitType {
		case "requests":
			if limits.RequestsPerMinute != nil {
				limit = limits.RequestsPerMinute
			} else if limits.RequestsPerHour != nil {
				limit = limits.RequestsPerHour
			} else if limits.RequestsPerDay != nil {
				limit = limits.RequestsPerDay
			}
		case "tokens":
			if limits.TokensPerMinute != nil {
				limit = limits.TokensPerMinute
			} else if limits.TokensPerHour != nil {
				limit = limits.TokensPerHour
			} else if limits.TokensPerDay != nil {
				limit = limits.TokensPerDay
			}
		}

		if limit != nil && usage > *limit {
			return fmt.Errorf("current usage (%d) exceeds limit (%d) for %s", usage, *limit, limitType)
		}
	}

	return nil
}

// EstimateWaitTime estimates how long to wait before making more requests
func EstimateWaitTime(limits *LimitsInfo) time.Duration {
	if limits.ResetTime == nil {
		return 1 * time.Minute // Default wait time
	}

	now := time.Now()
	waitTime := limits.ResetTime.Sub(now)

	if waitTime <= 0 {
		return 1 * time.Second // Minimum wait time
	}

	// Add 10% buffer to be safe
	return waitTime + (waitTime / 10)
}

// GetOptimalRequestRate calculates the optimal request rate to stay within limits
func GetOptimalRequestRate(limits *LimitsInfo) (int, time.Duration) {
	// Find the most restrictive limit
	var minLimit *int
	var period time.Duration

	if limits.RequestsPerMinute != nil && *limits.RequestsPerMinute > 0 {
		minLimit = limits.RequestsPerMinute
		period = 1 * time.Minute
	}

	if limits.RequestsPerHour != nil && *limits.RequestsPerHour > 0 {
		if minLimit == nil || *limits.RequestsPerHour < *minLimit {
			minLimit = limits.RequestsPerHour
			period = 1 * time.Hour
		}
	}

	if limits.RequestsPerDay != nil && *limits.RequestsPerDay > 0 {
		if minLimit == nil || *limits.RequestsPerDay < *minLimit {
			minLimit = limits.RequestsPerDay
			period = 24 * time.Hour
		}
	}

	if minLimit == nil {
		return 1, 1 * time.Second // Default conservative rate
	}

	// Use 80% of the limit to be safe
	safeLimit := int(float64(*minLimit) * 0.8)

	return safeLimit, period
}
