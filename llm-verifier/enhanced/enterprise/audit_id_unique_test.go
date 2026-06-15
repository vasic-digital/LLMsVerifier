package enterprise

import "testing"

// TestNewAuditID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form `fmt.Sprintf("audit_%d", time.Now().UnixNano())` collides in
// a tight loop — two RBAC audit entries written in the same nanosecond would
// share an ID, corrupting the audit trail. The crypto/rand suffix guarantees
// uniqueness. RED on UnixNano-only, GREEN after.
func TestNewAuditID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newAuditID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate RBAC audit ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique RBAC audit IDs, got %d", n, len(seen))
	}
}
