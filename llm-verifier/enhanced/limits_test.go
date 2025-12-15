package enhanced

import (
	"net/http"
	"testing"
	"time"
)

func TestNewLimitsDetector(t *testing.T) {
	detector := NewLimitsDetector()
	if detector == nil {
		t.Fatal("Expected LimitsDetector to be created, got nil")
	}
	if detector.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
	if detector.httpClient.Timeout.Seconds() != 30 {
		t.Errorf("Expected timeout to be 30 seconds, got %v", detector.httpClient.Timeout)
	}
}

func TestDetectOpenAILimits(t *testing.T) {
	detector := NewLimitsDetector()

	tests := []struct {
		name           string
		headers        http.Header
		expectedRPM    *int
		expectedTPM    *int
		expectedUsage  map[string]int
		shouldHaveTime bool
	}{
		{
			name: "Complete OpenAI headers",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("x-ratelimit-limit-requests", "60")
				h.Set("x-ratelimit-limit-tokens", "40000")
				h.Set("x-ratelimit-remaining-requests", "45")
				h.Set("x-ratelimit-remaining-tokens", "30000")
				h.Set("x-ratelimit-reset", "1700000000")
				return h
			}(),
			expectedRPM: intPtr(60),
			expectedTPM: intPtr(40000),
			expectedUsage: map[string]int{
				"requests": 15,    // 60 - 45
				"tokens":   10000, // 40000 - 30000
			},
			shouldHaveTime: true,
		},
		{
			name: "Partial OpenAI headers",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("x-ratelimit-limit-requests", "30")
				h.Set("x-ratelimit-remaining-requests", "10")
				return h
			}(),
			expectedRPM: intPtr(30),
			expectedTPM: nil,
			expectedUsage: map[string]int{
				"requests": 20, // 30 - 10
			},
			shouldHaveTime: false,
		},
		{
			name:           "No headers",
			headers:        http.Header{},
			expectedRPM:    nil,
			expectedTPM:    nil,
			expectedUsage:  map[string]int{},
			shouldHaveTime: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits, err := detector.DetectLimits("openai", "test-model", tt.headers)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check requests per minute
			if tt.expectedRPM != nil {
				if limits.RequestsPerMinute == nil {
					t.Error("Expected RequestsPerMinute to be set")
				} else if *limits.RequestsPerMinute != *tt.expectedRPM {
					t.Errorf("Expected RequestsPerMinute %v, got %v", *tt.expectedRPM, *limits.RequestsPerMinute)
				}
			} else if limits.RequestsPerMinute != nil {
				t.Errorf("Expected RequestsPerMinute to be nil, got %v", *limits.RequestsPerMinute)
			}

			// Check tokens per minute
			if tt.expectedTPM != nil {
				if limits.TokensPerMinute == nil {
					t.Error("Expected TokensPerMinute to be set")
				} else if *limits.TokensPerMinute != *tt.expectedTPM {
					t.Errorf("Expected TokensPerMinute %v, got %v", *tt.expectedTPM, *limits.TokensPerMinute)
				}
			} else if limits.TokensPerMinute != nil {
				t.Errorf("Expected TokensPerMinute to be nil, got %v", *limits.TokensPerMinute)
			}

			// Check current usage
			for key, expected := range tt.expectedUsage {
				if actual, ok := limits.CurrentUsage[key]; !ok {
					t.Errorf("Expected current usage for %s to be set", key)
				} else if actual != expected {
					t.Errorf("Expected current usage for %s to be %v, got %v", key, expected, actual)
				}
			}

			// Check reset time
			if tt.shouldHaveTime {
				if limits.ResetTime == nil {
					t.Error("Expected ResetTime to be set")
				}
			} else if limits.ResetTime != nil {
				t.Error("Expected ResetTime to be nil")
			}

			// Check default values
			if !limits.IsHardLimit {
				t.Error("Expected IsHardLimit to be true")
			}
		})
	}
}

func TestDetectAnthropicLimits(t *testing.T) {
	detector := NewLimitsDetector()

	tests := []struct {
		name           string
		headers        http.Header
		expectedRPH    *int
		expectedTPH    *int
		expectedUsage  map[string]int
		shouldHaveTime bool
	}{
		{
			name: "Complete Anthropic headers",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("anthropic-ratelimit-requests-limit", "1000")
				h.Set("anthropic-ratelimit-tokens-limit", "100000")
				h.Set("anthropic-ratelimit-requests-remaining", "800")
				h.Set("anthropic-ratelimit-tokens-remaining", "90000")
				h.Set("anthropic-ratelimit-reset", "2024-01-01T00:00:00Z")
				return h
			}(),
			expectedRPH: intPtr(1000),
			expectedTPH: intPtr(100000),
			expectedUsage: map[string]int{
				"requests": 200,   // 1000 - 800
				"tokens":   10000, // 100000 - 90000
			},
			shouldHaveTime: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits, err := detector.DetectLimits("anthropic", "test-model", tt.headers)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check requests per hour
			if tt.expectedRPH != nil {
				if limits.RequestsPerHour == nil {
					t.Error("Expected RequestsPerHour to be set")
				} else if *limits.RequestsPerHour != *tt.expectedRPH {
					t.Errorf("Expected RequestsPerHour %v, got %v", *tt.expectedRPH, *limits.RequestsPerHour)
				}
			}

			// Check tokens per hour
			if tt.expectedTPH != nil {
				if limits.TokensPerHour == nil {
					t.Error("Expected TokensPerHour to be set")
				} else if *limits.TokensPerHour != *tt.expectedTPH {
					t.Errorf("Expected TokensPerHour %v, got %v", *tt.expectedTPH, *limits.TokensPerHour)
				}
			}

			// Check current usage
			for key, expected := range tt.expectedUsage {
				if actual, ok := limits.CurrentUsage[key]; !ok {
					t.Errorf("Expected current usage for %s to be set", key)
				} else if actual != expected {
					t.Errorf("Expected current usage for %s to be %v, got %v", key, expected, actual)
				}
			}

			// Check reset time
			if tt.shouldHaveTime {
				if limits.ResetTime == nil {
					t.Error("Expected ResetTime to be set")
				}
			}
		})
	}
}

func TestDetectGenericLimits(t *testing.T) {
	detector := NewLimitsDetector()

	tests := []struct {
		name           string
		headers        http.Header
		expectedRPM    *int
		expectedTPM    *int
		expectedRPH    *int
		expectedRPD    *int
		expectedUsage  map[string]int
		shouldHaveTime bool
	}{
		{
			name: "Standard rate limit headers",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("x-rate-limit-limit", "60")
				h.Set("x-rate-limit-limit-tokens", "40000")
				h.Set("x-rate-limit-remaining", "30")
				h.Set("x-rate-limit-reset", "1700000000")
				return h
			}(),
			expectedRPM: intPtr(60),
			expectedTPM: intPtr(40000),
			expectedUsage: map[string]int{
				"requests": 30, // 60 - 30
			},
			shouldHaveTime: true,
		},
		{
			name: "Hourly and daily limits",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("x-rate-limit-limit-requests-per-hour", "3600")
				h.Set("x-rate-limit-limit-requests-per-day", "86400")
				return h
			}(),
			expectedRPH:    intPtr(3600),
			expectedRPD:    intPtr(86400),
			expectedUsage:  map[string]int{},
			shouldHaveTime: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits, err := detector.DetectLimits("unknown-provider", "test-model", tt.headers)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check requests per minute
			if tt.expectedRPM != nil {
				if limits.RequestsPerMinute == nil {
					t.Error("Expected RequestsPerMinute to be set")
				} else if *limits.RequestsPerMinute != *tt.expectedRPM {
					t.Errorf("Expected RequestsPerMinute %v, got %v", *tt.expectedRPM, *limits.RequestsPerMinute)
				}
			}

			// Check tokens per minute
			if tt.expectedTPM != nil {
				if limits.TokensPerMinute == nil {
					t.Error("Expected TokensPerMinute to be set")
				} else if *limits.TokensPerMinute != *tt.expectedTPM {
					t.Errorf("Expected TokensPerMinute %v, got %v", *tt.expectedTPM, *limits.TokensPerMinute)
				}
			}

			// Check requests per hour
			if tt.expectedRPH != nil {
				if limits.RequestsPerHour == nil {
					t.Error("Expected RequestsPerHour to be set")
				} else if *limits.RequestsPerHour != *tt.expectedRPH {
					t.Errorf("Expected RequestsPerHour %v, got %v", *tt.expectedRPH, *limits.RequestsPerHour)
				}
			}

			// Check requests per day
			if tt.expectedRPD != nil {
				if limits.RequestsPerDay == nil {
					t.Error("Expected RequestsPerDay to be set")
				} else if *limits.RequestsPerDay != *tt.expectedRPD {
					t.Errorf("Expected RequestsPerDay %v, got %v", *tt.expectedRPD, *limits.RequestsPerDay)
				}
			}

			// Check current usage
			for key, expected := range tt.expectedUsage {
				if actual, ok := limits.CurrentUsage[key]; !ok {
					t.Errorf("Expected current usage for %s to be set", key)
				} else if actual != expected {
					t.Errorf("Expected current usage for %s to be %v, got %v", key, expected, actual)
				}
			}

			// Check reset time
			if tt.shouldHaveTime {
				if limits.ResetTime == nil {
					t.Error("Expected ResetTime to be set")
				}
			}
		})
	}
}

func TestDetectLimits(t *testing.T) {
	detector := NewLimitsDetector()

	tests := []struct {
		name         string
		providerName string
		headers      http.Header
		shouldError  bool
	}{
		{
			name:         "OpenAI provider",
			providerName: "openai",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("x-ratelimit-limit-requests", "60")
				return h
			}(),
			shouldError: false,
		},
		{
			name:         "Anthropic provider",
			providerName: "anthropic",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("anthropic-ratelimit-requests-limit", "1000")
				return h
			}(),
			shouldError: false,
		},
		{
			name:         "Google provider",
			providerName: "google",
			headers:      http.Header{},
			shouldError:  false,
		},
		{
			name:         "Cohere provider",
			providerName: "cohere",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("x-ratelimit-limit", "60")
				return h
			}(),
			shouldError: false,
		},
		{
			name:         "Azure provider",
			providerName: "azure",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("x-rate-limit-limit", "60")
				return h
			}(),
			shouldError: false,
		},
		{
			name:         "Unknown provider",
			providerName: "unknown-provider",
			headers:      http.Header{},
			shouldError:  false, // Should not error, should return generic limits
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits, err := detector.DetectLimits(tt.providerName, "test-model", tt.headers)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if limits == nil {
					t.Error("Expected limits info but got nil")
				}
				if limits.CurrentUsage == nil {
					t.Error("Expected CurrentUsage map to be initialized")
				}
				if limits.AdditionalLimits == nil {
					t.Error("Expected AdditionalLimits map to be initialized")
				}
			}
		})
	}
}

func TestGetRateLimitAdvice(t *testing.T) {
	tests := []struct {
		name     string
		status   map[string]float64
		expected []string
	}{
		{
			name: "Critical usage",
			status: map[string]float64{
				"requests_per_minute": 95.0,
				"tokens_per_minute":   85.0,
			},
			expected: []string{
				"Critical: requests_per_minute usage at 95.0% - consider reducing request rate",
				"Warning: tokens_per_minute usage at 85.0% - monitor closely",
			},
		},
		{
			name: "Warning usage",
			status: map[string]float64{
				"requests_per_hour": 85.0,
				"tokens_per_hour":   70.0,
			},
			expected: []string{
				"Warning: requests_per_hour usage at 85.0% - monitor closely",
				"Info: tokens_per_hour usage at 70.0% - normal range",
			},
		},
		{
			name: "Normal usage",
			status: map[string]float64{
				"requests_per_day": 65.0,
				"tokens_per_day":   70.0,
			},
			expected: []string{
				"Info: requests_per_day usage at 65.0% - normal range",
				"Info: tokens_per_day usage at 70.0% - normal range",
			},
		},
		{
			name:     "No usage data",
			status:   map[string]float64{},
			expected: []string{"Rate limits are within normal ranges"},
		},
		{
			name: "Low usage",
			status: map[string]float64{
				"requests_per_minute": 30.0,
			},
			expected: []string{"Rate limits are within normal ranges"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advice := GetRateLimitAdvice(tt.status)

			if len(advice) != len(tt.expected) {
				t.Errorf("Expected %d advice items, got %d", len(tt.expected), len(advice))
			}

			// Check each expected advice item is present
			for _, expected := range tt.expected {
				found := false
				for _, actual := range advice {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected advice item not found: %s", expected)
				}
			}
		})
	}
}

func TestEstimateWaitTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		limits   *LimitsInfo
		expected time.Duration
	}{
		{
			name: "With reset time in future",
			limits: &LimitsInfo{
				ResetTime: timePtr(now.Add(5 * time.Minute)),
			},
			expected: 5*time.Minute + 30*time.Second, // 5 minutes + 10% buffer
		},
		{
			name: "With reset time in past",
			limits: &LimitsInfo{
				ResetTime: timePtr(now.Add(-5 * time.Minute)),
			},
			expected: 1 * time.Second, // Minimum wait time
		},
		{
			name:     "No reset time",
			limits:   &LimitsInfo{},
			expected: 1 * time.Minute, // Default wait time
		},
		{
			name: "Reset time exactly now",
			limits: &LimitsInfo{
				ResetTime: &now,
			},
			expected: 1 * time.Second, // Minimum wait time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			waitTime := EstimateWaitTime(tt.limits)

			// Allow small margin of error for timing
			lowerBound := tt.expected - (tt.expected / 10)
			upperBound := tt.expected + (tt.expected / 10)

			if waitTime < lowerBound || waitTime > upperBound {
				t.Errorf("Expected wait time between %v and %v, got %v", lowerBound, upperBound, waitTime)
			}
		})
	}
}

func TestGetOptimalRequestRate(t *testing.T) {
	tests := []struct {
		name           string
		limits         *LimitsInfo
		expectedRate   int
		expectedPeriod time.Duration
	}{
		{
			name: "Minute limit only",
			limits: &LimitsInfo{
				RequestsPerMinute: intPtr(60),
			},
			expectedRate:   48, // 80% of 60
			expectedPeriod: 1 * time.Minute,
		},
		{
			name: "Hour limit only",
			limits: &LimitsInfo{
				RequestsPerHour: intPtr(3600),
			},
			expectedRate:   2880, // 80% of 3600
			expectedPeriod: 1 * time.Hour,
		},
		{
			name: "Day limit only",
			limits: &LimitsInfo{
				RequestsPerDay: intPtr(86400),
			},
			expectedRate:   69120, // 80% of 86400
			expectedPeriod: 24 * time.Hour,
		},
		{
			name: "Multiple limits (minute is most restrictive)",
			limits: &LimitsInfo{
				RequestsPerMinute: intPtr(60),
				RequestsPerHour:   intPtr(1000),
				RequestsPerDay:    intPtr(10000),
			},
			expectedRate:   48, // 80% of 60 (most restrictive)
			expectedPeriod: 1 * time.Minute,
		},
		{
			name: "Multiple limits (minute is most restrictive)",
			limits: &LimitsInfo{
				RequestsPerMinute: intPtr(60),
				RequestsPerHour:   intPtr(1000),
				RequestsPerDay:    intPtr(10000),
			},
			expectedRate:   48, // 80% of 60 (most restrictive)
			expectedPeriod: 1 * time.Minute,
		},
		{
			name:           "No limits",
			limits:         &LimitsInfo{},
			expectedRate:   1,
			expectedPeriod: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, period := GetOptimalRequestRate(tt.limits)

			if rate != tt.expectedRate {
				t.Errorf("Expected rate %v, got %v", tt.expectedRate, rate)
			}

			if period != tt.expectedPeriod {
				t.Errorf("Expected period %v, got %v", tt.expectedPeriod, period)
			}
		})
	}
}

func TestValidateLimits(t *testing.T) {
	tests := []struct {
		name        string
		limits      *LimitsInfo
		shouldError bool
	}{
		{
			name: "Valid limits with requests per minute",
			limits: &LimitsInfo{
				RequestsPerMinute: intPtr(60),
				CurrentUsage: map[string]int{
					"requests": 30,
				},
			},
			shouldError: false,
		},
		{
			name: "Valid limits with tokens per hour",
			limits: &LimitsInfo{
				TokensPerHour: intPtr(100000),
				CurrentUsage: map[string]int{
					"tokens": 50000,
				},
			},
			shouldError: false,
		},
		{
			name: "No limits specified",
			limits: &LimitsInfo{
				CurrentUsage: map[string]int{},
			},
			shouldError: true,
		},
		{
			name: "Usage exceeds limit",
			limits: &LimitsInfo{
				RequestsPerMinute: intPtr(60),
				CurrentUsage: map[string]int{
					"requests": 70,
				},
			},
			shouldError: true,
		},
		{
			name: "Multiple valid limits",
			limits: &LimitsInfo{
				RequestsPerMinute: intPtr(60),
				TokensPerMinute:   intPtr(40000),
				CurrentUsage: map[string]int{
					"requests": 30,
					"tokens":   20000,
				},
			},
			shouldError: false,
		},
		{
			name: "Zero limit value",
			limits: &LimitsInfo{
				RequestsPerMinute: intPtr(0),
			},
			shouldError: true,
		},
		{
			name: "Negative limit value",
			limits: &LimitsInfo{
				RequestsPerMinute: intPtr(-10),
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLimits(tt.limits)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLimitsInfoStructure(t *testing.T) {
	// Test that LimitsInfo struct fields are properly defined
	now := time.Now()
	limits := &LimitsInfo{
		RequestsPerMinute: intPtr(60),
		RequestsPerHour:   intPtr(3600),
		RequestsPerDay:    intPtr(86400),
		TokensPerMinute:   intPtr(40000),
		TokensPerHour:     intPtr(2400000),
		TokensPerDay:      intPtr(57600000),
		CurrentUsage: map[string]int{
			"requests": 30,
			"tokens":   20000,
		},
		ResetPeriod: "minute",
		ResetTime:   &now,
		IsHardLimit: true,
		AdditionalLimits: map[string]interface{}{
			"quota_user": "test-project",
		},
	}

	if limits.RequestsPerMinute == nil || *limits.RequestsPerMinute != 60 {
		t.Errorf("Expected RequestsPerMinute 60, got %v", limits.RequestsPerMinute)
	}
	if limits.RequestsPerHour == nil || *limits.RequestsPerHour != 3600 {
		t.Errorf("Expected RequestsPerHour 3600, got %v", limits.RequestsPerHour)
	}
	if limits.RequestsPerDay == nil || *limits.RequestsPerDay != 86400 {
		t.Errorf("Expected RequestsPerDay 86400, got %v", limits.RequestsPerDay)
	}
	if limits.TokensPerMinute == nil || *limits.TokensPerMinute != 40000 {
		t.Errorf("Expected TokensPerMinute 40000, got %v", limits.TokensPerMinute)
	}
	if limits.TokensPerHour == nil || *limits.TokensPerHour != 2400000 {
		t.Errorf("Expected TokensPerHour 2400000, got %v", limits.TokensPerHour)
	}
	if limits.TokensPerDay == nil || *limits.TokensPerDay != 57600000 {
		t.Errorf("Expected TokensPerDay 57600000, got %v", limits.TokensPerDay)
	}
	if limits.CurrentUsage["requests"] != 30 {
		t.Errorf("Expected CurrentUsage[requests] 30, got %v", limits.CurrentUsage["requests"])
	}
	if limits.CurrentUsage["tokens"] != 20000 {
		t.Errorf("Expected CurrentUsage[tokens] 20000, got %v", limits.CurrentUsage["tokens"])
	}
	if limits.ResetPeriod != "minute" {
		t.Errorf("Expected ResetPeriod 'minute', got %s", limits.ResetPeriod)
	}
	if limits.ResetTime != &now {
		t.Errorf("Expected ResetTime to match provided time")
	}
	if !limits.IsHardLimit {
		t.Error("Expected IsHardLimit to be true")
	}
	if limits.AdditionalLimits["quota_user"] != "test-project" {
		t.Errorf("Expected AdditionalLimits[quota_user] 'test-project', got %v", limits.AdditionalLimits["quota_user"])
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
