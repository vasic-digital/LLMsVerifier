package providers

import "testing"

// TestNewKimiResponseID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form `fmt.Sprintf("kimi-code-cli-%d", time.Now().UnixNano())`
// collides in a tight loop — two responses produced in the same nanosecond
// would share an OpenAI-compatible response ID, which downstream clients key on.
// The crypto/rand suffix guarantees uniqueness. RED on UnixNano-only, GREEN
// after.
func TestNewKimiResponseID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newKimiResponseID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate kimi response ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique kimi response IDs, got %d", n, len(seen))
	}
}
