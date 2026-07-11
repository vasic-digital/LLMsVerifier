package capabilities

import (
	"os"
	"testing"
)

// TestHelixLLM_CapabilitySeedRegistered_Polarity is the §11.4.115 RED/GREEN
// polarity guard for registering HelixLLM as a tracked provider in the
// CONST-040 capability-seed registry (registry.go's providerCapabilitySeeds)
// — Phase A of the providers-coverage plan
// (docs/research/07.2026/06_providers_coverage/EXPANSION_PLAN_v2.md §3 Phase
// A). CONST-036/037/039: LLMsVerifier is the single source of truth for
// every provider — including the in-repo HelixLLM local coder — so it must
// be discoverable via GetProviderBaseCapabilities exactly like every other
// provider.
//
//	RED_MODE=1 (reproduce-on-broken-artifact): asserts the seed is ABSENT —
//	           PASSes ONLY on a pre-fix build (seed not yet added) and FAILs
//	           on the current fixed build (seed present).
//	RED_MODE=0 (default, standing regression guard): asserts the seed IS
//	           present and unverified-by-construction (Verified=false, C3).
func TestHelixLLM_CapabilitySeedRegistered_Polarity(t *testing.T) {
	caps := GetProviderBaseCapabilities("helixllm")

	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}

	switch redMode {
	case "1":
		if caps != nil {
			t.Fatalf("RED_MODE=1: expected the helixllm capability seed to be ABSENT " +
				"(reproducing the pre-fix defect), but it is present")
		}
		t.Log("RED_MODE=1 PASS: reproduced absence of the helixllm capability seed (pre-fix artifact)")
	case "0":
		if caps == nil {
			t.Fatalf("RED_MODE=0: helixllm capability seed not registered — HelixLLM is not " +
				"discoverable via GetProviderBaseCapabilities (CONST-040)")
		}
		if caps.Provider != "helixllm" {
			t.Fatalf("seed Provider field must be %q, got %q", "helixllm", caps.Provider)
		}
		if caps.Verified {
			t.Fatalf("seed MUST be unverified by construction (C3, CONST-036/037/040); got Verified=true")
		}
		t.Log("RED_MODE=0 PASS: helixllm capability seed registered, unverified-by-construction")
	default:
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}
}
