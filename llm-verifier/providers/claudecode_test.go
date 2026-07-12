package providers

// claudecode_test.go — unit tests for the Claude-Code-CLI-bridge adapter
// (providers/claudecode.go). Mocks are permitted here per §11.4.27 (unit
// tests only); every other test type for this adapter (the ccr-routed
// live sentinel probe + the garbage-route negative case) lives in
// claudecode_integration_test.go behind //go:build integration and
// drives the real `claude` binary — §11.4.27(A) no-fakes-beyond-unit.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// --- buildEnv (§11.4.6: FACT-derived per-alias env matrix) ----------------

func TestClaudeCodeBuildEnv_Native_StripsAmbientAnthropicVars(t *testing.T) {
	// A leftover ANTHROPIC_BASE_URL in the parent shell must NOT leak into
	// a "native account" probe — this is the exact leak §11.4.111 warns
	// against (silently redirecting a supposedly-native call).
	t.Setenv("ANTHROPIC_BASE_URL", "http://127.0.0.1:9999")
	t.Setenv("ANTHROPIC_API_KEY", "should-not-appear")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "should-not-appear-either")

	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{
		Kind:      ClaudeCodeAliasNative,
		ConfigDir: "/home/example/.claude-example",
	}, 0)

	env := adapter.buildEnv()

	foundConfigDir := false
	for _, kv := range env {
		if strings.HasPrefix(kv, "ANTHROPIC_") {
			t.Fatalf("native alias env leaked an ANTHROPIC_* var: %q", kv)
		}
		if kv == "CLAUDE_CONFIG_DIR=/home/example/.claude-example" {
			foundConfigDir = true
		}
	}
	if !foundConfigDir {
		t.Fatalf("native alias env missing CLAUDE_CONFIG_DIR=/home/example/.claude-example; env=%v", env)
	}
}

func TestClaudeCodeBuildEnv_ProviderRouter_SetsCcrVars(t *testing.T) {
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "stale-value-must-be-stripped")

	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{
		Kind:             ClaudeCodeAliasProviderRouter,
		AnthropicBaseURL: "http://127.0.0.1:3456",
		AnthropicAPIKey:  "ccr",
	}, 0)

	env := adapter.buildEnv()

	wantBaseURL := "ANTHROPIC_BASE_URL=http://127.0.0.1:3456"
	wantAPIKey := "ANTHROPIC_API_KEY=ccr"
	haveBaseURL, haveAPIKey := false, false
	for _, kv := range env {
		if kv == wantBaseURL {
			haveBaseURL = true
		}
		if kv == wantAPIKey {
			haveAPIKey = true
		}
		if strings.HasPrefix(kv, "ANTHROPIC_AUTH_TOKEN=") {
			t.Fatalf("provider-router alias must not carry a stale ANTHROPIC_AUTH_TOKEN, got %q", kv)
		}
	}
	if !haveBaseURL {
		t.Fatalf("provider-router env missing %q; env=%v", wantBaseURL, env)
	}
	if !haveAPIKey {
		t.Fatalf("provider-router env missing %q; env=%v", wantAPIKey, env)
	}
}

func TestClaudeCodeBuildEnv_ProviderNative_SetsBaseURLAuthTokenAndModel(t *testing.T) {
	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{
		Kind:               ClaudeCodeAliasProviderNative,
		AnthropicBaseURL:   "https://api.example-provider.test/v1",
		AnthropicAuthToken: "sk-example-redacted",
		AnthropicModel:     "example-model-v1",
	}, 0)

	env := adapter.buildEnv()

	want := map[string]bool{
		"ANTHROPIC_BASE_URL=https://api.example-provider.test/v1": false,
		"ANTHROPIC_AUTH_TOKEN=sk-example-redacted":                false,
		"ANTHROPIC_MODEL=example-model-v1":                        false,
	}
	for _, kv := range env {
		if _, ok := want[kv]; ok {
			want[kv] = true
		}
	}
	for k, found := range want {
		if !found {
			t.Fatalf("provider-native env missing %q; env=%v", k, env)
		}
	}
}

func TestClaudeCodeBuildEnv_ProviderNative_OmitsModelWhenUnset(t *testing.T) {
	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{
		Kind:               ClaudeCodeAliasProviderNative,
		AnthropicBaseURL:   "https://api.example-provider.test/v1",
		AnthropicAuthToken: "sk-example-redacted",
	}, 0)

	env := adapter.buildEnv()
	for _, kv := range env {
		if strings.HasPrefix(kv, "ANTHROPIC_MODEL=") {
			t.Fatalf("provider-native env set ANTHROPIC_MODEL despite AnthropicModel being empty: %q", kv)
		}
	}
}

// --- buildClaudeCodePrompt -------------------------------------------------

func TestBuildClaudeCodePrompt_RolePrefixing(t *testing.T) {
	prompt := buildClaudeCodePrompt([]Message{
		{Role: "system", Content: "You are terse."},
		{Role: "user", Content: "Say hi."},
		{Role: "assistant", Content: "Hi."},
		{Role: "user", Content: "Say bye."},
	})

	if !strings.Contains(prompt, "System: You are terse.") {
		t.Fatalf("expected system-prefixed line, got: %q", prompt)
	}
	if !strings.Contains(prompt, "Say hi.") {
		t.Fatalf("expected user content present verbatim, got: %q", prompt)
	}
	if !strings.Contains(prompt, "Assistant: Hi.") {
		t.Fatalf("expected assistant-prefixed line, got: %q", prompt)
	}
	if !strings.Contains(prompt, "Say bye.") {
		t.Fatalf("expected final user content present, got: %q", prompt)
	}
}

func TestBuildClaudeCodePrompt_EmptyForNoMessages(t *testing.T) {
	if got := buildClaudeCodePrompt(nil); got != "" {
		t.Fatalf("expected empty prompt for nil messages, got %q", got)
	}
	if got := buildClaudeCodePrompt([]Message{{Role: "tool", Content: "ignored role"}}); got != "" {
		t.Fatalf("expected unrecognised role to contribute nothing, got %q", got)
	}
}

// --- isRateLimitSignal (§11.4.69 infra-vs-determination classifier) -------

func TestIsRateLimitSignal_ClosedSetPositives(t *testing.T) {
	positives := []string{
		"HTTP 429 Too Many Requests",
		"error: rate limit exceeded, please retry",
		"rate_limit_error: your account has hit a rate_limit",
		"You are being rate limited",
		"Fair Usage limit reached for this account",
		"fair-usage cap hit",
		"the upstream is currently OVERLOADED",
		"Retry-After: 30",
		"Too Many Requests",
	}
	for _, s := range positives {
		if !isRateLimitSignal(s) {
			t.Errorf("isRateLimitSignal(%q) = false, want true", s)
		}
	}
}

func TestIsRateLimitSignal_NegativesDoNotFalsePositive(t *testing.T) {
	negatives := []string{
		"",
		"PONG",
		"the model replied successfully with a normal answer",
		"connection refused",
		"invalid API key",
	}
	for _, s := range negatives {
		if isRateLimitSignal(s) {
			t.Errorf("isRateLimitSignal(%q) = true, want false (false-positive would mask a genuine determination as infra)", s)
		}
	}
}

// --- claudeCodeCLIResult JSON shape (captured LIVE sample, §11.4.6) -------

// claudeCodePWU1CapturedSample is the EXACT stdout captured 2026-07-07
// during PWU-1 via:
//
//	claude -p "Reply with exactly: PONG" --output-format json \
//	  --max-turns 1 --dangerously-skip-permissions
//
// This is real captured evidence, not a hand-authored fixture — parsing
// it is the anti-bluff proof that claudeCodeCLIResult matches the
// CLI's actual wire shape (§11.4.6 no-guessing).
const claudeCodePWU1CapturedSample = `{"type":"result","subtype":"success","is_error":false,"api_error_status":null,"duration_ms":20061,"duration_api_ms":4825,"ttft_ms":11938,"ttft_stream_ms":11864,"time_to_request_ms":7136,"num_turns":1,"result":"PONG","stop_reason":"end_turn","session_id":"c48fb51d-cbea-4282-95ba-f7360766d798","total_cost_usd":2.4888570000000003,"usage":{"input_tokens":49732,"cache_creation_input_tokens":223253,"cache_read_input_tokens":15084,"output_tokens":5},"terminal_reason":"completed","fast_mode_state":"off","uuid":"00c52a28-f6d0-465e-82c8-3aa3e67cac9e"}`

func TestClaudeCodeCLIResult_ParsesLiveCapturedSample(t *testing.T) {
	var result claudeCodeCLIResult
	if err := json.Unmarshal([]byte(claudeCodePWU1CapturedSample), &result); err != nil {
		t.Fatalf("failed to parse live-captured claude -p --output-format json sample: %v", err)
	}
	if result.Type != "result" {
		t.Errorf("Type = %q, want %q", result.Type, "result")
	}
	if result.Subtype != "success" {
		t.Errorf("Subtype = %q, want %q", result.Subtype, "success")
	}
	if result.IsError {
		t.Errorf("IsError = true, want false for a successful sample")
	}
	if result.Result != "PONG" {
		t.Errorf("Result = %q, want %q", result.Result, "PONG")
	}
	if result.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", result.StopReason, "end_turn")
	}
	if result.TerminalReason != "completed" {
		t.Errorf("TerminalReason = %q, want %q", result.TerminalReason, "completed")
	}
	if result.SessionID == "" {
		t.Errorf("SessionID is empty, want a real session id from the captured sample")
	}
	if result.TotalCostUSD <= 0 {
		t.Errorf("TotalCostUSD = %v, want > 0", result.TotalCostUSD)
	}
	// api_error_status is JSON null in the captured success sample; the
	// json.RawMessage field must accept it without error (already proven
	// by Unmarshal succeeding above) and MUST render as the literal
	// "null" when re-marshalled untouched.
	if string(result.APIErrorStatus) != "null" {
		t.Errorf("APIErrorStatus raw = %q, want literal \"null\" for the captured success sample", string(result.APIErrorStatus))
	}
}

// --- response-ID generation (mirrors kimicode_id_unique_test.go) ---------

func TestNewClaudeCodeResponseID_Unique(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newClaudeCodeResponseID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate claude-code response ID at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique claude-code response IDs, got %d", n, len(seen))
	}
}

// --- GetKnownModels / ListModels -------------------------------------------

func TestClaudeCodeGetKnownModels_DefaultsWhenAliasModelUnset(t *testing.T) {
	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{Kind: ClaudeCodeAliasNative}, 0)
	models := adapter.GetKnownModels()
	if len(models) != 1 {
		t.Fatalf("expected exactly 1 known model, got %d", len(models))
	}
	if models[0].ID != ClaudeCodeDefaultModel {
		t.Errorf("known model ID = %q, want default %q", models[0].ID, ClaudeCodeDefaultModel)
	}
	if models[0].OwnedBy != "claude-code-cli" {
		t.Errorf("known model OwnedBy = %q, want %q", models[0].OwnedBy, "claude-code-cli")
	}
}

func TestClaudeCodeGetKnownModels_UsesAliasModelOverride(t *testing.T) {
	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{
		Kind:           ClaudeCodeAliasProviderNative,
		AnthropicModel: "claude-example-override",
	}, 0)
	models := adapter.GetKnownModels()
	if len(models) != 1 || models[0].ID != "claude-example-override" {
		t.Fatalf("expected the alias-configured model override to be used, got %+v", models)
	}
}

func TestClaudeCodeListModels_MapsToOpenAIModelsResponse(t *testing.T) {
	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{Kind: ClaudeCodeAliasNative}, 0)
	resp, err := adapter.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}
	if resp.Object != "list" {
		t.Errorf("Object = %q, want %q", resp.Object, "list")
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != ClaudeCodeDefaultModel {
		t.Fatalf("expected exactly 1 model entry with ID %q, got %+v", ClaudeCodeDefaultModel, resp.Data)
	}
}

// --- adapter identity / capability flags -----------------------------------

func TestClaudeCodeCLIAdapter_ProviderIdentityAndCapabilities(t *testing.T) {
	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{Kind: ClaudeCodeAliasNative}, 0)
	if got := adapter.GetProviderName(); got != "claude-code-cli" {
		t.Errorf("GetProviderName() = %q, want %q", got, "claude-code-cli")
	}
	if !adapter.SupportsStreaming() {
		t.Errorf("SupportsStreaming() = false, want true (single-final-chunk emulation is wired)")
	}
	if adapter.SupportsTools() {
		t.Errorf("SupportsTools() = true, want false (this bridge does not expose tool-calling)")
	}
}

// --- default-timeout normalisation -----------------------------------------

func TestNewClaudeCodeCLIAdapter_DefaultsTimeoutWhenNonPositive(t *testing.T) {
	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{Kind: ClaudeCodeAliasNative}, 0)
	if adapter.timeout <= 0 {
		t.Fatalf("expected a positive default timeout, got %v", adapter.timeout)
	}
}

// --- ChatCompletion guard: no prompt => explicit error, never a silent PASS

func TestClaudeCodeCLIAdapter_ChatCompletion_EmptyPromptIsError(t *testing.T) {
	adapter := NewClaudeCodeCLIAdapter(ClaudeCodeAliasConfig{Kind: ClaudeCodeAliasNative}, 0)
	if !adapter.IsAvailable() {
		t.Skip("SKIP: claude CLI not available on this host — cannot reach the no-prompt guard without a resolvable binary")
	}
	_, err := adapter.ChatCompletion(context.Background(), OpenAIChatRequest{Messages: nil})
	if err == nil {
		t.Fatalf("expected an error for an empty/no-message prompt, got nil (this would be a silent bluff)")
	}
	if !strings.Contains(err.Error(), "no prompt provided") {
		t.Errorf("error = %v, want it to mention the empty-prompt guard", err)
	}
}
