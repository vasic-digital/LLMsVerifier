package inmemory

import (
	"strings"
	"testing"
)

// TestNewSubscriptionID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form `fmt.Sprintf("sub-%s-%d", target, time.Now().UnixNano())`
// collided for two subscriptions registered against the same target within the
// same nanosecond. The subscription ID is the subscription's stored identity;
// the crypto/rand suffix guarantees uniqueness. RED on the UnixNano-only form
// (duplicates in a tight loop), GREEN after.
func TestNewSubscriptionID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newSubscriptionID("orders.created")
		if !strings.HasPrefix(id, "sub-orders.created-") {
			t.Fatalf("subscription ID lost its prefix: %q", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate subscription ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique subscription IDs, got %d", n, len(seen))
	}
}
