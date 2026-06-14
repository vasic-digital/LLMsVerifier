package providers

import (
	"sync"
	"testing"
	"time"
)

// RED test for the read-lock-protected map write in getFromCache.
//
// getFromCache takes cacheMutex.RLock() (a READ lock) but, when an entry is
// expired, calls delete(mps.cache, providerID) — a MAP WRITE under a read lock.
// GetAllModels fans GetModels (→ getFromCache) across many goroutines, so two
// goroutines hitting expired entries concurrently race on the map. Under -race
// this is reported as a data race; in production it can trip the unrecoverable
// "fatal error: concurrent map writes".
//
// Run with -race. Reproduces on the pre-fix code; passes once the expired-entry
// eviction no longer mutates the map under the read lock.

func TestGetFromCache_ConcurrentExpiredEviction_NoRace(t *testing.T) {
	mps := &ModelProviderService{
		cache:    make(map[string]*providerCacheEntry),
		cacheTTL: 24,
	}

	// Seed many already-expired entries (timestamp > 24h ago).
	const n = 64
	stale := time.Now().Add(-48 * time.Hour)
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		k := "provider-" + time.Duration(i).String()
		keys[i] = k
		mps.cache[k] = &providerCacheEntry{
			models:     []Model{{ID: "m", ProviderID: k}},
			timestamp:  stale,
			providerID: k,
		}
	}

	var wg sync.WaitGroup
	for _, k := range keys {
		// Two concurrent readers per key maximise the expired-eviction race.
		for r := 0; r < 2; r++ {
			wg.Add(1)
			go func(key string) {
				defer wg.Done()
				_ = mps.getFromCache(key)
			}(k)
		}
	}
	wg.Wait()
}
