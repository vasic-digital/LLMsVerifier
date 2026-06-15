package auth

import "testing"

// TestGenerateClientID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form assigned `client.ID = time.Now().UnixNano()` for LDAP/SSO
// clients; two clients created in the same nanosecond would share a client ID —
// and that ID keys per-client rate-limiting + usage attribution, so a collision
// merges two clients' quotas. generateClientID draws from crypto/rand (positive
// int64). RED on UnixNano-only, GREEN after.
func TestGenerateClientID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[int64]struct{}, n)
	for i := 0; i < n; i++ {
		id := generateClientID()
		if id <= 0 {
			t.Fatalf("client ID must be positive, got %d at iteration %d", id, i)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate client ID at iteration %d: %d", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique client IDs, got %d", n, len(seen))
	}
}
