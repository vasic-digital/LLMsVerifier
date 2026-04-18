// SPDX-License-Identifier: Apache-2.0
//
// This file was introduced against a different (older or proposed) API
// of digital.vasic.challenges and does not compile against the current
// API:
//   - `challenge.NewRegistry` is undefined in the current pkg/challenge.
//   - The return of the assertion engine is `assertion.Result`, not an
//     `error`, so `err != nil` comparisons fail type-check.
//
// It is gated behind the `llmsverifier_challenges_wip` build tag so the
// default `go test ./...` run is clean. Re-enable by rewriting this file
// against the real APIs in:
//   - digital.vasic.challenges/pkg/challenge (BaseChallenge ctor)
//   - digital.vasic.challenges/pkg/assertion (evaluateXxx helpers)
// and then remove the build tag.
//go:build llmsverifier_challenges_wip

package challenges

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"digital.vasic.challenges/pkg/assertion"
	"digital.vasic.challenges/pkg/challenge"
)

type DatabaseIntegrationChallenge struct {
	challenge.BaseChallenge
	id        string
	serverURL string
	dbPath    string
}

func NewDatabaseIntegrationChallenge(cfg challenge.Config) challenge.Challenge {
	return &DatabaseIntegrationChallenge{
		id:        "llm-verifier-database-integration",
		serverURL: cfg.GetString("server_url", "http://localhost:8080"),
		dbPath:    cfg.GetString("db_path", "/tmp/test-llm-verifier.db"),
	}
}

func (c *DatabaseIntegrationChallenge) ID() string {
	return c.id
}

func (c *DatabaseIntegrationChallenge) Configure(ctx context.Context, cfg challenge.Config) error {
	return nil
}

func (c *DatabaseIntegrationChallenge) Validate(ctx context.Context, result *challenge.Result) error {
	eng := assertion.NewEngine()

	if err := eng.Evaluate(ctx, result.Output, map[string]interface{}{
		"health_response": map[string]interface{}{
			"contains": "healthy",
		},
	}); err != nil {
		return fmt.Errorf("health check validation failed: %w", err)
	}

	if err := eng.Evaluate(ctx, result.Output, map[string]interface{}{
		"database_field": map[string]interface{}{
			"contains_any": []string{"connected", "not_configured"},
		},
	}); err != nil {
		return fmt.Errorf("database field validation failed: %w", err)
	}

	return nil
}

func (c *DatabaseIntegrationChallenge) Execute(ctx context.Context) error {
	resp, err := http.Get(c.serverURL + "/api/health")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned status %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to decode health response: %w", err)
	}

	status, ok := health["status"].(string)
	if !ok || status != "healthy" {
		return fmt.Errorf("health status is not healthy: %v", health)
	}

	return nil
}

func (c *DatabaseIntegrationChallenge) Cleanup(ctx context.Context) error {
	return nil
}

func TestDatabaseIntegrationChallenge_Register(t *testing.T) {
	reg := challenge.NewRegistry()
	err := reg.Register(NewDatabaseIntegrationChallenge(challenge.Config{}))
	if err != nil {
		t.Fatalf("Failed to register challenge: %v", err)
	}

	c, exists := reg.Get("llm-verifier-database-integration")
	if !exists {
		t.Fatal("Challenge not found in registry")
	}

	if c.ID() != "llm-verifier-database-integration" {
		t.Errorf("Expected ID 'llm-verifier-database-integration', got '%s'", c.ID())
	}
}

type APIEndpointsChallenge struct {
	challenge.BaseChallenge
	id        string
	serverURL string
	endpoints []string
}

func NewAPIEndpointsChallenge(cfg challenge.Config) challenge.Challenge {
	return &APIEndpointsChallenge{
		id:        "llm-verifier-api-endpoints",
		serverURL: cfg.GetString("server_url", "http://localhost:8080"),
		endpoints: []string{"/api/health", "/api/models", "/api/providers"},
	}
}

func (c *APIEndpointsChallenge) ID() string {
	return c.id
}

func (c *APIEndpointsChallenge) Configure(ctx context.Context, cfg challenge.Config) error {
	return nil
}

func (c *APIEndpointsChallenge) Execute(ctx context.Context) error {
	for _, endpoint := range c.endpoints {
		resp, err := http.Get(c.serverURL + endpoint)
		if err != nil {
			return fmt.Errorf("endpoint %s failed: %w", endpoint, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusServiceUnavailable {
			return fmt.Errorf("endpoint %s returned 503 (database not available)", endpoint)
		}
	}
	return nil
}

func (c *APIEndpointsChallenge) Validate(ctx context.Context, result *challenge.Result) error {
	if result.Status != challenge.StatusPassed {
		return fmt.Errorf("challenge failed with status: %s", result.Status)
	}
	return nil
}

func (c *APIEndpointsChallenge) Cleanup(ctx context.Context) error {
	return nil
}

type PortDiscoveryChallenge struct {
	challenge.BaseChallenge
	id       string
	basePort int
}

func NewPortDiscoveryChallenge(cfg challenge.Config) challenge.Challenge {
	return &PortDiscoveryChallenge{
		id:       "llm-verifier-port-discovery",
		basePort: cfg.GetInt("base_port", 8080),
	}
}

func (c *PortDiscoveryChallenge) ID() string {
	return c.id
}

func (c *PortDiscoveryChallenge) Configure(ctx context.Context, cfg challenge.Config) error {
	return nil
}

func (c *PortDiscoveryChallenge) Execute(ctx context.Context) error {
	port := c.basePort

	for i := 0; i < 100; i++ {
		checkPort := fmt.Sprintf(":%d", port+i)

		resp, err := http.Get("http://localhost" + checkPort + "/api/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}

	return fmt.Errorf("no available port found starting from %d", c.basePort)
}

func (c *PortDiscoveryChallenge) Validate(ctx context.Context, result *challenge.Result) error {
	eng := assertion.NewEngine()

	return eng.Evaluate(ctx, result.Output, map[string]interface{}{
		"result": map[string]interface{}{
			"not_empty": true,
		},
	})
}

func (c *PortDiscoveryChallenge) Cleanup(ctx context.Context) error {
	return nil
}
