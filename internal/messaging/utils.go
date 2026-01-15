package messaging

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

// generateUUID generates a UUID v4 string.
func generateUUID() string {
	uuid := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, uuid)
	if err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("msg-%d", time.Now().UnixNano())
	}

	// Set version (4) and variant (2) bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(uuid[0:4]),
		hex.EncodeToString(uuid[4:6]),
		hex.EncodeToString(uuid[6:8]),
		hex.EncodeToString(uuid[8:10]),
		hex.EncodeToString(uuid[10:16]))
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// calculateBackoff calculates exponential backoff with jitter.
func calculateBackoff(attempt int, baseBackoff, maxBackoff time.Duration) time.Duration {
	if attempt <= 0 {
		return baseBackoff
	}

	// Exponential backoff: base * 2^attempt
	backoff := baseBackoff
	for i := 0; i < attempt; i++ {
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
			break
		}
	}

	// Add jitter (±10%)
	jitter := time.Duration(float64(backoff) * 0.1)
	jitterBytes := make([]byte, 1)
	_, _ = rand.Read(jitterBytes)
	if jitterBytes[0]%2 == 0 {
		backoff += time.Duration(float64(jitter) * float64(jitterBytes[0]) / 255.0)
	} else {
		backoff -= time.Duration(float64(jitter) * float64(jitterBytes[0]) / 255.0)
	}

	return backoff
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// copyHeaders creates a copy of a headers map.
func copyHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	copy := make(map[string]string, len(headers))
	for k, v := range headers {
		copy[k] = v
	}
	return copy
}

// mergeHeaders merges two header maps, with the second map taking precedence.
func mergeHeaders(base, override map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}
