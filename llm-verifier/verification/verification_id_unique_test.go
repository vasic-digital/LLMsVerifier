package verification

import (
	"strings"
	"testing"
)

// TestNewVerificationID_Unique is a §11.4.115 reproduce-first uniqueness guard.
// The pre-fix form
// `fmt.Sprintf("<prefix>_%s_%s_%d", providerID, modelID, time.Now().Unix())`
// used second-resolution time, so re-verifying the SAME model within the same
// second (retries, back-to-back runs, concurrent VerifyAll) produced a
// duplicate verification_id. That ID is persisted into the result record and
// into `model.Features["verification_id"]`, so a collision aliases two distinct
// runs. The crypto/rand suffix guarantees uniqueness. RED on the Unix()-only
// form (massive duplicates for a fixed provider+model), GREEN after.
func TestNewVerificationID_Unique(t *testing.T) {
	cases := []string{"code_verify", "meaningful_verify", "debate_verify", "coding_cap"}
	for _, prefix := range cases {
		prefix := prefix
		t.Run(prefix, func(t *testing.T) {
			const n = 10000
			seen := make(map[string]struct{}, n)
			for i := 0; i < n; i++ {
				id := newVerificationID(prefix, "openai", "gpt-4")
				want := prefix + "_openai_gpt-4_"
				if !strings.HasPrefix(id, want) {
					t.Fatalf("verification ID lost its prefix: got %q want prefix %q", id, want)
				}
				if _, dup := seen[id]; dup {
					t.Fatalf("duplicate verification ID at iteration %d: %q", i, id)
				}
				seen[id] = struct{}{}
			}
			if len(seen) != n {
				t.Fatalf("expected %d unique verification IDs, got %d", n, len(seen))
			}
		})
	}
}
