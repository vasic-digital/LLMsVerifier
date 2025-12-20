package supervisor

import (
	"encoding/json"
	"fmt"
	"log"
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
