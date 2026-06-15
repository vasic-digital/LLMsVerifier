package scoring

import "testing"

// TestGenerateBatchID_Unique + TestNewAlertID_Unique are §11.4.115
// reproduce-first uniqueness guards. The pre-fix forms (`batch_%d` / `alert_%d`
// from time.Now().UnixNano()) collide in a tight loop — two batch jobs / two
// alerts in the same nanosecond would share an ID, corrupting batch status
// lookup and alert dedup. The crypto/rand suffix guarantees uniqueness. RED on
// UnixNano-only, GREEN after.
func TestGenerateBatchID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := generateBatchID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate batch ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique batch IDs, got %d", n, len(seen))
	}
}

func TestNewAlertID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newAlertID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate alert ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique alert IDs, got %d", n, len(seen))
	}
}
