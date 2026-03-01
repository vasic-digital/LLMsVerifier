package verification

import (
	"testing"
)

func TestValidateMeaningfulResponse(t *testing.T) {
	cvs := &CodeVerificationService{}

	tests := []struct {
		name     string
		response string
		expected bool
	}{
		// Valid meaningful responses
		{"simple hello", "Hello! How can I help you today?", true},
		{"greeting", "Hi there! I'm ready to assist.", true},
		{"short but valid", "Hello!", true},
		{"response with punctuation", "Hey! What's up?", true},
		{"casual response", "Hi! What would you like to work on?", true},

		// Empty or too short
		{"empty", "", false},
		{"single char", "a", false},
		{"too short", "hi", false},
		{"two chars", "ok", false},

		// Error indicators
		{"error message", "Error: Connection failed", false},
		{"api error", "API Error: Invalid key", false},
		{"cannot respond", "I cannot respond to that.", false},
		{"not available", "Service not available", false},
		{"authentication failed", "Authentication failed", false},
		{"rate limit", "Rate limit exceeded", false},
		{"timeout", "Request timeout", false},
		{"internal error", "Internal server error", false},
		{"bad gateway", "502 Bad Gateway", false},
		{"forbidden", "403 Forbidden", false},
		{"unauthorized", "401 Unauthorized", false},
		{"not found", "404 Not Found", false},

		// Refusal patterns
		{"im sorry", "I'm sorry, I can't help with that.", false},
		{"i cannot", "I cannot provide that information.", false},
		{"as an ai", "As an AI, I am not able to do that.", false},
		{"i dont have", "I don't have access to that.", false},
		{"i was not trained", "I wasn't trained on that data.", false},

		// Placeholder content
		{"placeholder", "This is placeholder content.", false},
		{"todo", "TODO: Implement this feature.", false},
		{"tbd", "TBD - To be determined.", false},
		{"coming soon", "Coming soon!", false},
		{"not implemented", "Not implemented yet.", false},

		// Edge cases
		{"null response", "null", false},
		{"nil response", "nil", false},
		{"undefined", "undefined", false},
		{"error brackets", "[error] Something went wrong", false},
		{"unicode error", "⚠️ Error occurred", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cvs.validateMeaningfulResponse(tt.response)
			if result != tt.expected {
				t.Errorf("validateMeaningfulResponse(%q) = %v, want %v", tt.response, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input     string
		maxLength int
		expected  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"test", 4, "test"},
		{"", 5, ""},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLength)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLength, result, tt.expected)
			}
		})
	}
}
