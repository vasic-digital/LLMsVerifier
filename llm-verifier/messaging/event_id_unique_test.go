package messaging

import "testing"

// TestGenerateEventID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form combined millisecond timestamp + nanosecond%1000, which
// collides heavily for events emitted within the same millisecond — two
// distinct verification events would share an event ID, corrupting downstream
// message-broker consumers that key on event ID. The crypto/rand suffix
// guarantees uniqueness. RED on the old scheme, GREEN after.
func TestGenerateEventID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := generateEventID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate event ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique event IDs, got %d", n, len(seen))
	}
}
