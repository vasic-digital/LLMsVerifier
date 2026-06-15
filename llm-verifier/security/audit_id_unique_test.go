package security

import "testing"

// TestGenerateAuditID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form `fmt.Sprintf("audit_%d", time.Now().UnixNano())` collides
// heavily in a tight loop (same nanosecond reused), corrupting the audit trail
// because two distinct audit events would share an ID. The crypto/rand suffix
// (randomIDSuffix) guarantees uniqueness. RED on UnixNano-only, GREEN after.
func TestGenerateAuditID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := generateAuditID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate audit ID generated at iteration %d: %q (collision corrupts the audit trail)", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique audit IDs, got %d", n, len(seen))
	}
}
