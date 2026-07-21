package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// truncatedServer returns an OpenAI-compatible response whose content is EMPTY
// because the completion hit the token budget: finish_reason="length". This is
// the exact wire shape a reasoning model produces when its reasoning tokens
// consume the whole max_tokens allowance before any answer is emitted.
func truncatedServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message":       map[string]any{"role": "assistant", "content": ""},
				"finish_reason": "length",
			}},
		})
	}))
}

// TestChatCompleteNamesBudgetTruncation is the RED baseline (§11.4.115) for a
// layer-3 false-negative measured against a live reasoning model.
//
// FORENSICS (captured, live, exact driver round-2 shape, varying ONLY max_tokens):
//
//	512  -> content=0    finish=length   (reasoning consumed the whole budget)
//	512  -> content=1058 finish=length   (same call, shorter reasoning — a coin flip)
//	1024 -> content=1105 finish=stop
//	2048 -> content=1589 finish=stop
//	4096 -> content=1007 finish=stop     (max observed completion: 1364 tokens)
//
// Reasoning length varies per sampling (194-1099 tokens observed), so a FIXED
// 512-token round-2 budget makes "did we get an answer?" non-deterministic.
// When it truncates, the caller reported only "response message content was
// empty" — which reads as "the model said nothing" and hides the real cause,
// so the failure is neither diagnosable nor attributable to the budget.
//
// The error MUST name the truncation. Without the fix the message is the bare
// empty-content string and this test FAILS.
func TestChatCompleteNamesBudgetTruncation(t *testing.T) {
	srv := truncatedServer(t)
	defer srv.Close()

	_, err := chatComplete(context.Background(), srv.Client(), srv.URL, "k", "m",
		[]chatMessage{{Role: "user", Content: "describe this code"}}, 512)
	if err == nil {
		t.Fatal("a truncated, empty-content response must be an error, not a silent pass")
	}
	got := strings.ToLower(err.Error())
	if !strings.Contains(got, "truncat") && !strings.Contains(got, "max_tokens") {
		t.Errorf("error must name the BUDGET TRUNCATION so the failure is diagnosable "+
			"and attributable (not a bare empty-content message); got: %v", err)
	}
}

// TestChatCompleteStillReportsGenuinelyEmptyContent guards the other direction:
// an empty content that did NOT truncate (finish_reason="stop") must keep its
// original empty-content diagnosis. Without this, a fix could mislabel every
// empty response as a budget problem — trading one wrong story for another.
func TestChatCompleteStillReportsGenuinelyEmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message":       map[string]any{"role": "assistant", "content": ""},
				"finish_reason": "stop",
			}},
		})
	}))
	defer srv.Close()

	_, err := chatComplete(context.Background(), srv.Client(), srv.URL, "k", "m",
		[]chatMessage{{Role: "user", Content: "hi"}}, 512)
	if err == nil {
		t.Fatal("an empty content must still be an error")
	}
	if strings.Contains(strings.ToLower(err.Error()), "truncat") {
		t.Errorf("a non-truncated empty response must NOT be blamed on the token budget; got: %v", err)
	}
}

// TestRound2BudgetAccommodatesReasoning pins the measured budget. The forensics
// above show 512 truncates non-deterministically while every sample at >=1024
// finished "stop", with a maximum observed completion of 1364 tokens. 2048 is
// the smallest power-of-two clearing that ceiling with headroom.
//
// max_tokens is a CAP, not a charge — billing follows tokens actually generated
// — so raising it removes truncation without raising cost for short answers.
func TestRound2BudgetAccommodatesReasoning(t *testing.T) {
	if round2MaxTokens < 2048 {
		t.Errorf("round-2 budget %d is below the measured requirement: reasoning tokens "+
			"(194-1099 observed) plus the answer must fit, max observed completion 1364",
			round2MaxTokens)
	}
}

// TestRound1BudgetAccommodatesReasoning pins the round-1 budget.
//
// MEASURED (deepseek-v4-pro, real fixture + real prompt template, 3 samples per
// budget, varying ONLY max_tokens): reasoning 58-97 tokens; 64 -> 0/3 pass,
// 128 -> 2/3, 256 -> 3/3. Max completion over n=10 uncensored at 2048: 128.
//
// Round 1 at 256 was NOT failing for THIS model. It is raised because reasoning
// distributions do not transfer between models: a live run-proof leg saw
// siliconflow return the sentinel PREFIX "ZETA-9-ORANGE-" for the sentinel
// "ZETA-9-ORANGE-7f3a" — severed mid-token — and be marked "cannot see code"
// while its layer-4 TUI turn PASSED. Round 1 is where truncation is most
// expensive: its failure is a definitive exit-1 that DE-VERIFIES a working
// provider, where round 2 merely skips.
func TestRound1BudgetAccommodatesReasoning(t *testing.T) {
	if round1MaxTokens < 512 {
		t.Errorf("round-1 budget %d leaves too little headroom: max observed completion "+
			"was 128 tokens for this model, but a truncated round 1 de-verifies a working "+
			"provider, so the budget must clear other models' reasoning too", round1MaxTokens)
	}
}

// TestJudgeBudgetAccommodatesReasoning pins the judge budget.
//
// MEASURED (same setup, judge prompt built from the real rubric with a real
// round-2 description held fixed): reasoning 78-361 tokens; 64 -> 0/3 pass,
// 128 -> 0/3, 256 -> 1/3, 512 -> 3/3. Every failing sample was
// finish_reason="length" with EMPTY content — the documented truncation shape.
//
// 512 is deliberately NOT the floor here. It passed 3/3 only as an artifact of
// small n: widening to 9 uncensored samples surfaced a 500-token completion,
// leaving 512 just 12 tokens of headroom. This is the concrete case where three
// samples would have produced the wrong constant.
func TestJudgeBudgetAccommodatesReasoning(t *testing.T) {
	if judgeMaxTokens < 1024 {
		t.Errorf("judge budget %d is below the measured requirement: a reasoning judge spent "+
			"up to 500 completion tokens to emit a single integer (n=9), and 64 truncated "+
			"3/3 with empty content", judgeMaxTokens)
	}
}

// TestChatCompleteSurfacesFinishReason guards the mechanism the two diagnoses
// above depend on: the stop reason must travel back to the caller alongside the
// content. Returning content alone is what made a severed round-1 reply
// indistinguishable from a model that ignored the instruction — the caller then
// blamed the model ("sentinel not found") for a budget WE chose.
func TestChatCompleteSurfacesFinishReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				// Non-empty but SEVERED: a strict prefix of the sentinel, exactly
				// the live siliconflow shape.
				"message":       map[string]any{"role": "assistant", "content": "ZETA-9-ORANGE-"},
				"finish_reason": "length",
			}},
		})
	}))
	defer srv.Close()

	out, err := chatComplete(context.Background(), srv.Client(), srv.URL, "k", "m",
		[]chatMessage{{Role: "user", Content: "echo the sentinel"}}, 256)
	if err != nil {
		t.Fatalf("a non-empty truncated response is not an error at this layer: %v", err)
	}
	if out.FinishReason != "length" {
		t.Errorf("finish_reason must reach the caller so a severed reply can be told apart "+
			"from a non-compliant model; got %q", out.FinishReason)
	}
	if out.Content != "ZETA-9-ORANGE-" {
		t.Errorf("content must survive intact alongside the stop reason; got %q", out.Content)
	}
}
