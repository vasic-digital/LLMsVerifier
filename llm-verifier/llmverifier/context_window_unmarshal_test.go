package llmverifier

import (
	"encoding/json"
	"testing"
)

// §11.4.135 standing regression guard for the ContextWindow decode bug
// (2026-06-13): some providers (e.g. Groq) emit "context_window": <number>
// while others emit the structured object. The original struct-only decode
// returned `json: cannot unmarshal number into Go struct field ...ContextWindow`,
// which aborted the ENTIRE model-list decode for that provider — so a provider
// with a valid key surfaced ZERO models. UnmarshalJSON now tolerates both forms.
//
// Mutation-proven: removing ContextWindow.UnmarshalJSON makes the bare-int case
// below fail to decode (the exact production crash).

func TestContextWindow_UnmarshalJSON_BareInteger(t *testing.T) {
	// The Groq form that crashed the whole provider.
	var cw ContextWindow
	if err := json.Unmarshal([]byte(`131072`), &cw); err != nil {
		t.Fatalf("bare-integer context_window must decode (Groq form); got error: %v", err)
	}
	if cw.TotalMaxTokens != 131072 {
		t.Errorf("bare integer should map to TotalMaxTokens; got %d", cw.TotalMaxTokens)
	}
}

func TestContextWindow_UnmarshalJSON_ObjectForm(t *testing.T) {
	var cw ContextWindow
	if err := json.Unmarshal([]byte(`{"total_max_tokens":200000}`), &cw); err != nil {
		t.Fatalf("object-form context_window must still decode; got error: %v", err)
	}
	if cw.TotalMaxTokens != 200000 {
		t.Errorf("object form total_max_tokens not decoded; got %d", cw.TotalMaxTokens)
	}
}

func TestContextWindow_UnmarshalJSON_Null(t *testing.T) {
	var cw ContextWindow
	if err := json.Unmarshal([]byte(`null`), &cw); err != nil {
		t.Fatalf("null context_window must be tolerated; got error: %v", err)
	}
}

// TestContextWindow_InModelList proves the real failure mode: a model object
// carrying a numeric context_window decodes cleanly (it used to abort the list).
func TestContextWindow_InModelList(t *testing.T) {
	payload := `[{"context_window":131072},{"context_window":{"total_max_tokens":8192}}]`
	var holders []struct {
		ContextWindow ContextWindow `json:"context_window"`
	}
	if err := json.Unmarshal([]byte(payload), &holders); err != nil {
		t.Fatalf("mixed-form model list must decode (the production crash); got: %v", err)
	}
	if len(holders) != 2 || holders[0].ContextWindow.TotalMaxTokens != 131072 || holders[1].ContextWindow.TotalMaxTokens != 8192 {
		t.Fatalf("mixed-form decode wrong: %+v", holders)
	}
}
