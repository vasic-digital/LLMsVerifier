package events

import (
	"strings"
	"testing"
)

// TestNewConnectionID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form `fmt.Sprintf("ws_%d", time.Now().UnixNano())` collides in a
// tight loop. The connection ID keys both the subscriber map and the connection
// wrapper, so a collision corrupts WebSocket fan-out (events delivered to the
// wrong client). The crypto/rand suffix guarantees uniqueness. RED on
// UnixNano-only, GREEN after.
func TestNewConnectionID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newConnectionID()
		if !strings.HasPrefix(id, "ws_") {
			t.Fatalf("connection ID lost its ws_ prefix: %q", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate connection ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique connection IDs, got %d", n, len(seen))
	}
}
