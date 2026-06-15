package monitoring

import (
	"strings"
	"sync"
	"testing"
)

// TestNewAlertID_Unique is a §11.4.115 reproduce-first uniqueness guard. The
// pre-fix form `fmt.Sprintf("%s-%d", metric, time.Now().Unix())` used
// second-resolution time, so the same metric crossing its threshold more than
// once within the same second produced duplicate alert IDs. The crypto/rand
// suffix guarantees uniqueness. RED on the Unix()-only form (massive
// duplicates), GREEN after.
func TestNewAlertID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newAlertID("cpu_usage")
		if !strings.HasPrefix(id, "cpu_usage-") {
			t.Fatalf("alert ID lost its metric prefix: %q", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate alert ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique alert IDs, got %d", n, len(seen))
	}
}

// TestNewAlertID_Unique_Concurrent exercises concurrent threshold breaches —
// the real shape of the defect (a monitor evaluating many metrics in parallel).
func TestNewAlertID_Unique_Concurrent(t *testing.T) {
	const goroutines = 20
	const perG = 500
	var mu sync.Mutex
	seen := make(map[string]struct{}, goroutines*perG)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				id := newAlertID("mem_usage")
				mu.Lock()
				if _, dup := seen[id]; dup {
					mu.Unlock()
					t.Errorf("duplicate alert ID under concurrency: %q", id)
					return
				}
				seen[id] = struct{}{}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(seen) != goroutines*perG {
		t.Fatalf("expected %d unique concurrent alert IDs, got %d", goroutines*perG, len(seen))
	}
}
