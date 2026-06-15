package providers

import (
	"strings"
	"testing"
)

// TestNewReplicateResponseID_Unique is a §11.4.115 reproduce-first uniqueness
// guard. The pre-fix form `"replicate-" + time.Now().Unix()` used
// second-resolution time, so two responses produced within the same second
// (concurrent requests, or a stream chunk + the final response) shared an
// OpenAI-compatible response ID — a wire-format identity callers may key on.
// The crypto/rand suffix guarantees uniqueness. RED on the Unix()-only form
// (massive duplicates), GREEN after.
func TestNewReplicateResponseID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newReplicateResponseID()
		if !strings.HasPrefix(id, "replicate-") {
			t.Fatalf("replicate response ID lost its prefix: %q", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate replicate response ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique replicate response IDs, got %d", n, len(seen))
	}
}
