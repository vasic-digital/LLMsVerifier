package logging

import "testing"

// TestGenerateLogID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form `fmt.Sprintf("log_%d", time.Now().UnixNano())` collides in a
// tight loop, which would let two distinct log entries share an ID. The
// crypto/rand suffix guarantees uniqueness. RED on UnixNano-only, GREEN after.
func TestGenerateLogID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := generateLogID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate log ID generated at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique log IDs, got %d", n, len(seen))
	}
}
