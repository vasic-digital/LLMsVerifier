package enhanced

import "testing"

// TestNewTaskID_Unique + TestNewMessageID_Unique are §11.4.115 reproduce-first
// uniqueness guards. The pre-fix forms (`task_%d` / `task_%d_sub1` / `msg_%d`
// from time.Now().UnixNano()) collide in a tight loop — and DecomposeTask emits
// a primary task plus two subtasks back-to-back with no I/O between them, so the
// three would routinely share the same nanosecond. Colliding task/message IDs
// corrupt task tracking + context history. The crypto/rand suffix guarantees
// uniqueness. RED on UnixNano-only, GREEN after.
func TestNewTaskID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n*3)
	for i := 0; i < n; i++ {
		for _, id := range []string{newTaskID(""), newTaskID("sub1"), newTaskID("sub2")} {
			if _, dup := seen[id]; dup {
				t.Fatalf("duplicate task ID at iteration %d: %q", i, id)
			}
			seen[id] = struct{}{}
		}
	}
	if len(seen) != n*3 {
		t.Fatalf("expected %d unique task IDs, got %d", n*3, len(seen))
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
