package checkpointing

import "testing"

// TestNewCheckpointID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form `fmt.Sprintf("chk_%s_%d", agentID, time.Now().UnixNano())`
// collides in a tight loop for the same agent — and the ID is used as the
// on-disk checkpoint filename + in-memory map key, so a collision overwrites a
// prior checkpoint (lost agent state). The crypto/rand suffix guarantees
// uniqueness. RED on UnixNano-only, GREEN after.
func TestNewCheckpointID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newCheckpointID("agent-1")
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate checkpoint ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique checkpoint IDs, got %d", n, len(seen))
	}
}
