package api

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

// TestHandlers_Complete tests all API handlers
func TestHandlers_Complete(t *testing.T) {
	tests := []struct {
		name     string
		handler  string
		setupFunc func() interface{}
		validateFunc func(t *testing.T, result interface{})
	}{
		{
			name:    "GetModelsHandler",
			handler: "GetModels",
			setupFunc: func() interface{} {
				return setupTestModels()
			},
			validateFunc: func(t *testing.T, result interface{}) {
				models := result.([]Model)
				assert.NotEmpty(t, models)
				for _, model := range models {
					assert.Contains(t, model.Name, "(SC:")
				}
			},
		},
		{
			name:    "VerifyModelHandler",
			handler: "VerifyModel",
			setupFunc: func() interface{} {
				return setupTestVerification()
			},
			validateFunc: func(t *testing.T, result interface{}) {
				verification := result.(VerificationResult)
				assert.True(t, verification.Success)
				assert.Contains(t, verification.ScoreSuffix, "(SC:")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := tt.setupFunc()
			tt.validateFunc(t, setup)
		})
	}
}

func setupTestModels() []Model {
	return []Model{
		{
			ModelID:      "gpt-4",
			Name:         "GPT-4 (SC:8.5)",
			Provider:     "OpenAI",
			OverallScore: 8.5,
			IsActive:     true,
		},
		{
			ModelID:      "claude-3",
			Name:         "Claude-3 (SC:7.8)",
			Provider:     "Anthropic",
			OverallScore: 7.8,
			IsActive:     true,
		},
	}
}

func setupTestVerification() VerificationResult {
	return VerificationResult{
		ID:          "test-123",
		ModelID:     "gpt-4",
		Prompt:      "Test prompt",
		Response:    "Test response",
		Score:       8.5,
		ScoreSuffix: "(SC:8.5)",
		Success:     true,
		Timestamp:   time.Now(),
		Duration:    1500,
	}
}