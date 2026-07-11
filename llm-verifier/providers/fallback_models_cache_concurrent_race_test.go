package providers

import (
	"fmt"
	"sync"
	"testing"
)

// §11.4.169(10)/(11) coverage gap-fill — concurrency/atomicity + race-condition
// test-type coverage for the package-level `modelCache` singleton
// (providers/fallback_models.go).
//
// modelCache is a process-wide *ModelCache guarded by a sync.RWMutex, written
// via SetCachedModels and read via GetCachedModels (which GetFallbackModels
// consults first). It is exercised concurrently in production: the sibling
// ModelProviderService.GetAllModels fan-out (model_provider_service.go) spans
// a goroutine per configured provider (`var wg sync.WaitGroup` + per-provider
// `go func(...)`), and any of those goroutines' discovery paths may fall back
// to GetFallbackModels/SetCachedModels for the SAME or DIFFERENT provider IDs
// at the same wall-clock moment. Despite that, providers/fallback_models_test.go
// only ever exercised modelCache single-threaded (TestModelCache_SetAndGet /
// TestModelCache_NotFound) — zero -race coverage existed for this shared
// mutable cache before this test, unlike the sibling
// cache_expiry_race_red_test.go which already covers a DIFFERENT cache
// (ModelProviderService.cache) in this same package.
//
// This is the thinnest genuine §11.4.169 gap found this session (see the
// session's own baseline `go test ./... -count=1` run: 58 ok / 3 pre-existing,
// out-of-scope FAILs in tests/automation_test.go unrelated to this package).
// capabilities.Detector's cache (the other mutex-guarded map candidate
// surveyed) was independently confirmed lock-correct (RLock-then-RUnlock
// before any write, writes only under Lock) with no matching gap this severe.
//
// Run with -race. This test is the STANDING GREEN guard for the CURRENT
// (already-correct) implementation; its §1.1 self-validation is documented in
// the accompanying evidence note (docs/qa/) rather than left as residue in
// this file per §11.4.84 (no mutation markers may land in a committed
// artifact) — the guard was manually verified, in this session, to FAIL under
// `go test -race` when SetCachedModels's `mu.Lock()/mu.Unlock()` pair is
// temporarily removed, and to PASS once restored (git diff was reverted
// before commit; see evidence note for the exact captured transcript).
func TestModelCache_ConcurrentGetSet_NoRace(t *testing.T) {
	const providers = 12
	const writersPerProvider = 6
	const readersPerProvider = 6

	var wg sync.WaitGroup

	for p := 0; p < providers; p++ {
		providerID := fmt.Sprintf("race-provider-%d", p)
		models := []Model{
			{ID: fmt.Sprintf("%s-model-a", providerID), Name: "A", ProviderID: providerID, ProviderName: providerID, MaxTokens: 4096},
			{ID: fmt.Sprintf("%s-model-b", providerID), Name: "B", ProviderID: providerID, ProviderName: providerID, MaxTokens: 8192},
		}

		// Concurrent writers hammering the SAME provider key — exercises the
		// Lock()-guarded double-map write (models + timestamps) for races
		// between writers.
		for w := 0; w < writersPerProvider; w++ {
			wg.Add(1)
			go func(pid string, ms []Model) {
				defer wg.Done()
				SetCachedModels(pid, ms)
			}(providerID, models)
		}

		// Concurrent readers hammering the SAME provider key while writers
		// are in flight — exercises the RLock()-guarded double-map read
		// racing against the writers above.
		for r := 0; r < readersPerProvider; r++ {
			wg.Add(1)
			go func(pid string) {
				defer wg.Done()
				_, _ = GetCachedModels(pid)
			}(providerID)
		}

		// GetFallbackModels itself calls GetCachedModels first — exercise the
		// real production call path (not just the two primitives directly),
		// concurrently, for both a cached AND an uncached (fallback-literal)
		// provider ID so both branches race together.
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			_ = GetFallbackModels(pid)
		}(providerID)

		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = GetFallbackModels("openai") // known static-fallback branch, no cache hit path required
		}()
	}

	wg.Wait()

	// Sanity (non-bluff): confirm at least one of the concurrently-written
	// entries is actually retrievable post-storm — a cache that silently
	// dropped every write under contention would still pass a bare -race
	// scan but would be a functional (not just concurrency-safety) bluff.
	cached, ok := GetCachedModels("race-provider-0")
	if !ok {
		t.Fatalf("expected race-provider-0 to be cached after concurrent writers, got ok=false")
	}
	if len(cached) != 2 {
		t.Fatalf("expected 2 cached models for race-provider-0, got %d", len(cached))
	}
}
