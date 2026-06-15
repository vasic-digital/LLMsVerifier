package events

import (
	"strings"
	"testing"
)

// TestNewNotificationSubscriberID_Unique is a §11.4.115 reproduce-first
// uniqueness guard. The pre-fix form
// `fmt.Sprintf("notify_%s_%d", serviceType, time.Now().Unix())` used
// second-resolution time, so two subscribers of the same serviceType created
// within the same second collided. The ID keys the subscriber registry, so a
// collision silently aliases/overwrites a subscriber and misroutes
// notifications. The crypto/rand suffix guarantees uniqueness. This loop mints
// many IDs for the same serviceType within well under a second — RED on the
// Unix()-only form (massive duplicates), GREEN after the suffix.
func TestNewNotificationSubscriberID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newNotificationSubscriberID("slack")
		if !strings.HasPrefix(id, "notify_slack_") {
			t.Fatalf("notification subscriber ID lost its prefix: %q", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate notification subscriber ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique notification subscriber IDs, got %d", n, len(seen))
	}
}
