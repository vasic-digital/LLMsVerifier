package messaging

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateUUID(t *testing.T) {
	t.Run("generates valid UUID", func(t *testing.T) {
		uuid := generateUUID()
		assert.NotEmpty(t, uuid)
		assert.Len(t, uuid, 36) // UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	})

	t.Run("generates unique UUIDs", func(t *testing.T) {
		uuids := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			uuid := generateUUID()
			assert.False(t, uuids[uuid], "duplicate UUID generated")
			uuids[uuid] = true
		}
	})
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 1, -1},
		{0, 0, 0},
	}

	for _, tc := range tests {
		result := min(tc.a, tc.b)
		assert.Equal(t, tc.expected, result)
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{5, 5, 5},
		{-1, 1, 1},
		{0, 0, 0},
	}

	for _, tc := range tests {
		result := max(tc.a, tc.b)
		assert.Equal(t, tc.expected, result)
	}
}

func TestCalculateBackoff(t *testing.T) {
	baseBackoff := 100 * time.Millisecond
	maxBackoff := 10 * time.Second

	t.Run("first attempt returns base", func(t *testing.T) {
		backoff := calculateBackoff(0, baseBackoff, maxBackoff)
		assert.Equal(t, baseBackoff, backoff)
	})

	t.Run("increases exponentially", func(t *testing.T) {
		backoff1 := calculateBackoff(1, baseBackoff, maxBackoff)
		backoff2 := calculateBackoff(2, baseBackoff, maxBackoff)
		backoff3 := calculateBackoff(3, baseBackoff, maxBackoff)

		// With jitter, we can't check exact values, but should be > base
		assert.Greater(t, float64(backoff1), float64(baseBackoff)*0.8)
		assert.Greater(t, float64(backoff2), float64(baseBackoff))
		assert.Greater(t, float64(backoff3), float64(backoff2)*0.8)
	})

	t.Run("caps at max", func(t *testing.T) {
		backoff := calculateBackoff(100, baseBackoff, maxBackoff)
		// Should be around maxBackoff (with jitter)
		assert.LessOrEqual(t, backoff, maxBackoff+maxBackoff/10)
	})

	t.Run("handles negative attempt", func(t *testing.T) {
		backoff := calculateBackoff(-1, baseBackoff, maxBackoff)
		assert.Equal(t, baseBackoff, backoff)
	})
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate", "hello world", 8, "hello..."},
		{"very short max", "hello", 2, "he"},
		{"max 3", "hello", 3, "hel"},
		{"empty string", "", 5, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateString(tc.input, tc.maxLen)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCopyHeaders(t *testing.T) {
	t.Run("copy nil headers", func(t *testing.T) {
		result := copyHeaders(nil)
		assert.Nil(t, result)
	})

	t.Run("copy empty headers", func(t *testing.T) {
		result := copyHeaders(make(map[string]string))
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("copy headers", func(t *testing.T) {
		headers := map[string]string{
			"Content-Type": "application/json",
			"X-Custom":     "value",
		}
		result := copyHeaders(headers)

		assert.Equal(t, headers, result)

		// Modifying result shouldn't affect original
		result["New-Key"] = "new-value"
		assert.NotContains(t, headers, "New-Key")
	})
}

func TestMergeHeaders(t *testing.T) {
	t.Run("merge nil maps", func(t *testing.T) {
		result := mergeHeaders(nil, nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("merge with nil override", func(t *testing.T) {
		base := map[string]string{"key": "value"}
		result := mergeHeaders(base, nil)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("merge with nil base", func(t *testing.T) {
		override := map[string]string{"key": "value"}
		result := mergeHeaders(nil, override)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("override takes precedence", func(t *testing.T) {
		base := map[string]string{
			"key1": "base1",
			"key2": "base2",
		}
		override := map[string]string{
			"key2": "override2",
			"key3": "override3",
		}
		result := mergeHeaders(base, override)

		assert.Equal(t, "base1", result["key1"])
		assert.Equal(t, "override2", result["key2"])
		assert.Equal(t, "override3", result["key3"])
	})
}
