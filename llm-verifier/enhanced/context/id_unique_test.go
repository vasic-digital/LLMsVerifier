package context

import "testing"

// TestNewSummaryID_Unique + TestNewMessageID_Unique are §11.4.115 reproduce-first
// uniqueness guards. The pre-fix forms (`summary_%d` / `msg_%d` from
// time.Now().UnixNano()) collide in a tight loop — two summaries / two messages
// created in the same nanosecond would share an ID, corrupting long-term + short-
// term context history. The crypto/rand suffix guarantees uniqueness. RED on
// UnixNano-only, GREEN after.
func TestNewSummaryID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newSummaryID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate summary ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique summary IDs, got %d", n, len(seen))
	}
}

func TestNewMessageID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newMessageID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate message ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique message IDs, got %d", n, len(seen))
	}
}
