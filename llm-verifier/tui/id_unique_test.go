package tui

import "testing"

// TestNewNotificationID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form `fmt.Sprintf("notif-%d", time.Now().UnixNano())` collides in
// a tight loop — two TUI notifications added in the same nanosecond would share
// an ID. The crypto/rand suffix guarantees uniqueness. RED on UnixNano-only,
// GREEN after.
func TestNewNotificationID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newNotificationID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate TUI notification ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique TUI notification IDs, got %d", n, len(seen))
	}
}
