package scheduler

import "testing"

// TestGenerateScheduleAndRunID_Unique is a §11.4.115 reproduce-first uniqueness
// guard. The pre-fix forms `sched_%d` / `run_%d` from time.Now().UnixNano()
// collide in a tight loop, which would let two schedules (or two runs) share an
// ID — corrupting the schedule map keyed by ID. The crypto/rand suffix
// guarantees uniqueness. RED on UnixNano-only, GREEN after.
func TestGenerateScheduleAndRunID_Unique(t *testing.T) {
	const n = 10000
	t.Run("schedule_id", func(t *testing.T) {
		seen := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			id := generateScheduleID()
			if _, dup := seen[id]; dup {
				t.Fatalf("duplicate schedule ID at iteration %d: %q", i, id)
			}
			seen[id] = struct{}{}
		}
		if len(seen) != n {
			t.Fatalf("expected %d unique schedule IDs, got %d", n, len(seen))
		}
	})
	t.Run("run_id", func(t *testing.T) {
		seen := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			id := generateRunID()
			if _, dup := seen[id]; dup {
				t.Fatalf("duplicate run ID at iteration %d: %q", i, id)
			}
			seen[id] = struct{}{}
		}
		if len(seen) != n {
			t.Fatalf("expected %d unique run IDs, got %d", n, len(seen))
		}
	})
}
