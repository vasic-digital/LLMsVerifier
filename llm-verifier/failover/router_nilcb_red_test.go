package failover

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.llmsverifier/database"
)

// RED test for nil-circuit-breaker dereference in the latency/weighted routers.
//
// GetCircuitBreaker returns nil for a provider that was never registered with
// the HealthChecker (a map miss). The routers dereference that result with
// .IsAvailable() without a nil guard, so routing a model whose provider has not
// been added to the health checker panics with a nil-pointer dereference
// instead of returning ErrNoHealthyProviders.
//
// Reproduces on the pre-fix code; passes once the routers nil-guard the
// circuit breaker.

func TestLatencyBasedRouter_RouteRequest_UnregisteredProvider_NoPanic(t *testing.T) {
	lt := NewLatencyTracker()
	hc := NewHealthChecker(nil) // no providers registered
	db := setupTestHealthCheckerDB(t)

	// A provider + model exist in the DB, but the provider was NEVER added to
	// the health checker — exactly the "model persisted, health checker not yet
	// populated" state.
	provider := &database.Provider{Name: "ghost", Endpoint: "http://localhost:9999", Description: "Test"}
	require.NoError(t, db.CreateProvider(provider))

	model := &database.Model{ProviderID: provider.ID, ModelID: "ghost-model", Name: "Ghost Model"}
	require.NoError(t, db.CreateModel(model))

	lbr := NewLatencyBasedRouter(lt, hc, db)

	// Must NOT panic; must surface the no-healthy-providers error.
	assert.NotPanics(t, func() {
		_, err := lbr.RouteRequest(model.ID)
		assert.ErrorIs(t, err, ErrNoHealthyProviders,
			"unregistered provider must yield ErrNoHealthyProviders, not a panic")
	})
}

func TestWeightedRouter_RouteRequest_UnregisteredProvider_NoPanic(t *testing.T) {
	lt := NewLatencyTracker()
	hc := NewHealthChecker(nil)
	db := setupTestHealthCheckerDB(t)

	provider := &database.Provider{Name: "ghost", Endpoint: "http://localhost:9999", Description: "Test"}
	require.NoError(t, db.CreateProvider(provider))

	model := &database.Model{ProviderID: provider.ID, ModelID: "ghost-model", Name: "Ghost Model"}
	require.NoError(t, db.CreateModel(model))

	wr := NewWeightedRouter(lt, hc)

	assert.NotPanics(t, func() {
		_, err := wr.RouteRequest(model.ID, db)
		assert.ErrorIs(t, err, ErrNoHealthyProviders,
			"unregistered provider must yield ErrNoHealthyProviders, not a panic")
	})
}

func TestWeightedRouter_CalculateProviderScore_UnregisteredProvider_NoPanic(t *testing.T) {
	lt := NewLatencyTracker()
	hc := NewHealthChecker(nil)
	db := setupTestHealthCheckerDB(t)

	wr := NewWeightedRouter(lt, hc)

	// Score for a provider the health checker has never seen must not panic,
	// and the health component (weight 0.4) must be excluded → only the
	// latency component (default 1.0 * 0.6) contributes.
	assert.NotPanics(t, func() {
		score := wr.CalculateProviderScore("999", db)
		assert.InDelta(t, 0.6, score, 1e-9,
			"unknown provider has no circuit breaker → unavailable → health component 0")
	})
}
