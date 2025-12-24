package client

import (
	"time"
)

// ErrorSummary represents error statistics
type ErrorSummary struct {
	TotalErrors int64            `json:"total_errors"`
	ErrorRate   float64          `json:"error_rate"`
	ErrorTypes  map[string]int64 `json:"error_types"`
	GeneratedAt time.Time        `json:"generated_at"`
}
