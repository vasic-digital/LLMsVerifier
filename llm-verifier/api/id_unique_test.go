package api

import "testing"

// TestGenerateAuditID_Unique + TestGenerateUUID_Unique are §11.4.115
// reproduce-first uniqueness guards. The pre-fix forms (`audit_%d` from
// time.Now().UnixNano() and `%x` of UnixNano for request IDs) collide in a
// tight loop — two audit events / two HTTP requests served in the same
// nanosecond would share an ID (X-Request-ID collision breaks request tracing).
// The crypto/rand suffix guarantees uniqueness. RED on UnixNano-only, GREEN
// after.
func TestGenerateAuditID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := generateAuditID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate api audit ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique api audit IDs, got %d", n, len(seen))
	}
}

func TestGenerateUUID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := generateUUID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate request ID at iteration %d: %q (X-Request-ID collision)", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique request IDs, got %d", n, len(seen))
	}
}
