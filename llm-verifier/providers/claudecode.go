package providers

// claudecode.go implements the Claude-Code-CLI-bridge adapter — the
// sink-side probe that shells out to `claude -p` so we can ask "is this
// claudeN/provider-router/provider-native alias actually answering, or is
// it Fair-Usage/rate-limit capped?" the same way the worker itself would
// ask (§11.4.69 sink-side positive evidence; §11.4.13 out-of-band
// captured-evidence). Cloned from providers/kimicode.go's exec/parse
// pattern (§11.4.74 extend-don't-reimplement) — the near-exact template
// cited by the incorporation plan
// (docs/research/llmsverifier_incorporation_20260707/ANALYSIS_AND_PLAN.md
// PART C.2, ATMOSphere-Android-15 repo).
//
// Three ALIAS KINDS (per-invocation env matrix — §11.4.6 no-guessing: the
// exact env each kind needs is FACT, derived from the operator's own
// ~/.local/share/claude-multi-account/aliases.sh + ~/.claude-code-router
// wiring, not guessed):
//
//   - native:          CLAUDE_CONFIG_DIR=<per-account dir>, every
//     ANTHROPIC_* var UNSET so the CLI falls back to its own
//     CLI-refreshed OAuth credentials under that config dir.
//   - provider-router:  ANTHROPIC_BASE_URL=http://127.0.0.1:3456 (ccr),
//     ANTHROPIC_API_KEY=ccr (ccr's front accepts any non-empty key and
//     translates Anthropic-protocol -> the configured OpenAI-compatible
//     backend).
//   - provider-native:  ANTHROPIC_BASE_URL/ANTHROPIC_AUTH_TOKEN (and
//     optionally ANTHROPIC_MODEL) pointing straight at a provider's own
//     Anthropic-compatible endpoint, bypassing ccr.
//
// The probe is judged by a DETERMINISTIC SENTINEL SUBSTRING match — no LLM
// judge, no false pass (mirrors
// cmd/semantic-code-visibility/main.go's round-1 discipline, fetched
// read-only from origin/main since the checked-out submodule commit
// predates that command — see the PWU-1 report for the fetch-first
// finding). A 429 / rate-limit / Fair-Usage / overloaded signal in
// either the CLI's stderr OR its `--output-format json` result payload
// is classified as an INFRA condition (ErrClaudeCodeRateLimited) and is
// NEVER folded into a false PASS.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ClaudeCodeAliasKind is the closed set of env-matrix shapes the bridge
// knows how to build. Any other value is a config error, never guessed.
type ClaudeCodeAliasKind string

const (
	// ClaudeCodeAliasNative drives `claude -p` under a native Max/Pro
	// account's own CLAUDE_CONFIG_DIR, with every ANTHROPIC_* override
	// stripped so the CLI falls back to its own CLI-refreshed OAuth.
	ClaudeCodeAliasNative ClaudeCodeAliasKind = "native"
	// ClaudeCodeAliasProviderNative points `claude -p` straight at a
	// provider's own Anthropic-compatible endpoint (ANTHROPIC_BASE_URL +
	// ANTHROPIC_AUTH_TOKEN, optionally ANTHROPIC_MODEL), bypassing ccr.
	ClaudeCodeAliasProviderNative ClaudeCodeAliasKind = "provider-native"
	// ClaudeCodeAliasProviderRouter routes `claude -p` through the local
	// claude-code-router (ccr) front (ANTHROPIC_BASE_URL=ccr,
	// ANTHROPIC_API_KEY=ccr).
	ClaudeCodeAliasProviderRouter ClaudeCodeAliasKind = "provider-router"
)

const (
	// ClaudeCodeDefaultModel is used when the caller (or the alias
	// config) does not name a model explicitly.
	ClaudeCodeDefaultModel = "claude-opus-4-8"
)

// ClaudeCodeAliasConfig names ONE alias's env matrix. Exactly the fields
// each ClaudeCodeAliasKind needs are consulted; unrelated fields are
// ignored so a caller may keep one struct populated with every possible
// field and just flip Kind.
type ClaudeCodeAliasConfig struct {
	// Kind selects which env-matrix branch buildEnv() takes.
	Kind ClaudeCodeAliasKind
	// ConfigDir is CLAUDE_CONFIG_DIR for ClaudeCodeAliasNative.
	ConfigDir string
	// AnthropicBaseURL is ANTHROPIC_BASE_URL for ProviderNative/ProviderRouter.
	AnthropicBaseURL string
	// AnthropicAuthToken is ANTHROPIC_AUTH_TOKEN for ProviderNative.
	AnthropicAuthToken string
	// AnthropicAPIKey is ANTHROPIC_API_KEY for ProviderRouter (ccr
	// accepts the literal "ccr" as a non-empty placeholder key).
	AnthropicAPIKey string
	// AnthropicModel optionally overrides the model requested via
	// ANTHROPIC_MODEL for ProviderNative; also used as the ListModels()
	// fallback model id when the caller does not specify one.
	AnthropicModel string
}

// ErrClaudeCodeRateLimited is returned (wrapped, via errors.Is) whenever
// the CLI invocation itself completed or failed in a way that positively
// indicates 429 / rate-limit / Fair-Usage / overloaded — NEVER folded
// into a generic failure and NEVER mistaken for a genuine content
// determination (§11.4.69 / §11.4.107: an infra condition is a different
// class of non-pass than "the model actually replied without the
// sentinel").
var ErrClaudeCodeRateLimited = errors.New("claude code cli: rate limited or fair-usage capped")

// claudeCodeRateLimitSignals is the CLOSED, case-insensitive substring
// set that classifies a claude -p failure/response as infra rather than
// a genuine determination. Extending this list is a deliberate,
// evidence-driven change (§11.4.6) — not a place to guess new phrases.
var claudeCodeRateLimitSignals = []string{
	"429",
	"rate limit",
	"rate_limit",
	"ratelimit",
	"fair usage",
	"fair-usage",
	"fairusage",
	"overloaded",
	"retry-after",
	"too many requests",
}

// isRateLimitSignal reports whether s contains any of the closed-set
// rate-limit/Fair-Usage phrases, case-insensitively.
func isRateLimitSignal(s string) bool {
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	for _, sig := range claudeCodeRateLimitSignals {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

// ErrClaudeCodeQuotaExceeded is returned (wrapped, via errors.Is)
// whenever the CLI invocation completed or failed in a way that
// positively indicates a QUOTA condition -- a subscription / balance /
// weekly-or-session-usage-limit that will NOT clear on a short retry --
// as distinct from the TRANSIENT ErrClaudeCodeRateLimited condition
// (§11.4.69 / §11.4.107: an infra condition is a different class of
// non-pass than "the model actually replied without the sentinel", and a
// quota condition is itself a different sub-class of infra condition than
// a transient rate limit -- a caller acting on this distinction applies a
// materially longer cooldown for quota than for rate-limit). PWU-2 of the
// LLMsVerifier incorporation
// (docs/research/llmsverifier_incorporation_20260707/ANALYSIS_AND_PLAN.md
// PART C.3, ATMOSphere-Android-15 repo).
var ErrClaudeCodeQuotaExceeded = errors.New("claude code cli: quota or subscription usage cap exceeded")

// claudeCodeQuotaSignals is the CLOSED, case-insensitive substring set
// that classifies a claude-code-cli-bridge failure as a QUOTA condition
// rather than a transient rate limit. Checked BEFORE
// claudeCodeRateLimitSignals at every call site (§11.4.6 -- quota is the
// more specific, higher-cost-to-misclassify condition). Real captured
// FACTs backing this list: HTTP 402 body
// `{"detail":"Subscription usage cap exceeded. Please add balance to
// continue."}` (qa-results/multitrack/logs/T4_iter2_20260706T223846Z.log,
// this repo) and the "weekly limit"/"session limit" quota-exhausted
// markers already recognised by
// constitution/scripts/multitrack/multitrack_fallback_monitor.sh
// (MT_FBMON_QUOTA_DEFAULT). UNCONFIRMED (§11.4.6): the exact live
// claude -p stderr/exit text for the native weekly-limit UX message was
// not re-captured during this PWU (forcing it live would burn the quota
// this bridge exists to protect) -- "weekly limit" is the operator-cited
// literal phrase. Extending this list is a deliberate, evidence-driven
// change -- not a place to guess new phrases.
var claudeCodeQuotaSignals = []string{
	"weekly limit",
	"session limit",
	"monthly limit",
	"subscription usage cap",
	"usage cap exceeded",
	"add balance",
	"402",
}

// isQuotaSignal reports whether s contains any of the closed-set quota
// phrases, case-insensitively.
func isQuotaSignal(s string) bool {
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	for _, sig := range claudeCodeQuotaSignals {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

// claudeCodeCLIResult models the JSON object `claude -p --output-format
// json` emits on stdout. FACT, captured live 2026-07-07 (PWU-1) via:
//
//	claude -p "Reply with exactly: PONG" --output-format json \
//	  --max-turns 1 --dangerously-skip-permissions
//
// which returned (redacted-irrelevant fields elided):
//
//	{"type":"result","subtype":"success","is_error":false,
//	 "api_error_status":null,"duration_ms":20061,"num_turns":1,
//	 "result":"PONG","stop_reason":"end_turn",
//	 "session_id":"...","total_cost_usd":2.49,"usage":{...},
//	 "modelUsage":{...},"permission_denials":[],
//	 "terminal_reason":"completed","fast_mode_state":"off","uuid":"..."}
//
// UNCONFIRMED (§11.4.6): the exact non-null shape of api_error_status on
// a genuine error response was not captured live (forcing a real 429
// would burn the quota this bridge exists to protect) — it is kept as
// json.RawMessage so a future error-shaped payload never fails to parse;
// PWU-2 (rate-limit-verdict wiring) is the tracked follow-up that closes
// this gap with a captured error sample.
type claudeCodeCLIResult struct {
	Type           string          `json:"type"`
	Subtype        string          `json:"subtype"`
	IsError        bool            `json:"is_error"`
	APIErrorStatus json.RawMessage `json:"api_error_status"`
	DurationMs     int64           `json:"duration_ms"`
	NumTurns       int             `json:"num_turns"`
	Result         string          `json:"result"`
	StopReason     string          `json:"stop_reason"`
	SessionID      string          `json:"session_id"`
	TotalCostUSD   float64         `json:"total_cost_usd"`
	TerminalReason string          `json:"terminal_reason"`
}

// ClaudeCodeCLIAdapter shells out to the `claude` CLI (Claude Code) in
// one-shot print mode, per alias env matrix, and judges the reply by
// deterministic sentinel substring at the caller layer (this adapter
// only surfaces the raw completion text — see cmd/semantic-code-
// visibility/main.go's round-1 pattern for how a caller performs the
// sentinel judgement).
type ClaudeCodeCLIAdapter struct {
	BaseAdapter
	alias        ClaudeCodeAliasConfig
	cliPath      string
	cliAvailable bool
	checkOnce    sync.Once
	checkErr     error
	timeout      time.Duration
}

// NewClaudeCodeCLIAdapter builds a bridge adapter bound to one alias's
// env matrix. timeout<=0 defaults to 180s (mirrors kimicode.go).
func NewClaudeCodeCLIAdapter(alias ClaudeCodeAliasConfig, timeout time.Duration) *ClaudeCodeCLIAdapter {
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	return &ClaudeCodeCLIAdapter{
		BaseAdapter: BaseAdapter{
			client:   nil,
			endpoint: alias.AnthropicBaseURL,
			apiKey:   "",
			headers:  map[string]string{},
		},
		alias:   alias,
		timeout: timeout,
	}
}

// NewClaudeCodeCLIAdapterFromClient adapts the 3-argument
// NewXXXAdapter(httpClient, endpoint, apiKey) construction shape used by
// ModelProviderService.fetchFromProviderAPIEnhanced's registration
// switch (model_provider_service.go ~:711) to this CLI-subprocess
// bridge. It is a ListModels()-oriented compatibility shim: the switch
// call site only carries client/endpoint/apiKey, so it builds a
// ClaudeCodeAliasProviderRouter config from those three values.
// ChatCompletion callers that need a native or provider-native alias
// MUST construct via NewClaudeCodeCLIAdapter with an explicit
// ClaudeCodeAliasConfig instead — this shim exists only so the switch's
// `case "claude-code":` can register a working adapter without changing
// the switch's calling convention.
func NewClaudeCodeCLIAdapterFromClient(client interface{}, endpoint, apiKey string) *ClaudeCodeCLIAdapter {
	_ = client // signature parity with the switch's call site only.
	alias := ClaudeCodeAliasConfig{
		Kind:             ClaudeCodeAliasProviderRouter,
		AnthropicBaseURL: endpoint,
		AnthropicAPIKey:  apiKey,
	}
	return NewClaudeCodeCLIAdapter(alias, 0)
}

// IsAvailable reports whether the `claude` binary resolves on PATH and
// responds to --version. It does NOT invoke a real completion (that
// would spend the quota this bridge exists to protect) — auth validity
// surfaces at ChatCompletion time via is_error / a rate-limit signal.
func (p *ClaudeCodeCLIAdapter) IsAvailable() bool {
	p.checkOnce.Do(func() {
		path, err := exec.LookPath("claude")
		if err != nil {
			p.checkErr = fmt.Errorf("claude command not found: %w", err)
			p.cliAvailable = false
			return
		}
		p.cliPath = path

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, path, "--version")
		if _, err := cmd.CombinedOutput(); err != nil {
			p.checkErr = fmt.Errorf("claude command failed: %w", err)
			p.cliAvailable = false
			return
		}

		p.cliAvailable = true
	})
	return p.cliAvailable
}

// GetError returns the last availability-check error, if any.
func (p *ClaudeCodeCLIAdapter) GetError() error {
	p.IsAvailable()
	return p.checkErr
}

// buildEnv constructs the per-alias env matrix: os.Environ() with every
// ANTHROPIC_* variable stripped, then the alias kind's own subset
// re-added. Stripping first (rather than only overriding) is load-
// bearing for the native case — a leftover ANTHROPIC_BASE_URL from the
// parent shell must not silently redirect a "native account" probe.
func (p *ClaudeCodeCLIAdapter) buildEnv() []string {
	base := os.Environ()
	filtered := make([]string, 0, len(base)+4)
	for _, kv := range base {
		key := kv
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			key = kv[:idx]
		}
		if strings.HasPrefix(key, "ANTHROPIC_") {
			continue
		}
		filtered = append(filtered, kv)
	}

	switch p.alias.Kind {
	case ClaudeCodeAliasNative:
		if p.alias.ConfigDir != "" {
			filtered = append(filtered, "CLAUDE_CONFIG_DIR="+p.alias.ConfigDir)
		}
	case ClaudeCodeAliasProviderRouter:
		filtered = append(filtered, "ANTHROPIC_BASE_URL="+p.alias.AnthropicBaseURL)
		filtered = append(filtered, "ANTHROPIC_API_KEY="+p.alias.AnthropicAPIKey)
	case ClaudeCodeAliasProviderNative:
		filtered = append(filtered, "ANTHROPIC_BASE_URL="+p.alias.AnthropicBaseURL)
		filtered = append(filtered, "ANTHROPIC_AUTH_TOKEN="+p.alias.AnthropicAuthToken)
		if p.alias.AnthropicModel != "" {
			filtered = append(filtered, "ANTHROPIC_MODEL="+p.alias.AnthropicModel)
		}
	}
	return filtered
}

// buildClaudeCodePrompt flattens an OpenAI-style message list into the
// single -p prompt string `claude -p` expects, mirroring kimicode.go's
// role-prefixing convention exactly so the two CLI-bridge adapters stay
// consistent for anyone reading both.
func buildClaudeCodePrompt(messages []Message) string {
	var b strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			b.WriteString("System: ")
			b.WriteString(msg.Content)
			b.WriteString("\n\n")
		case "user":
			b.WriteString(msg.Content)
			b.WriteString("\n")
		case "assistant":
			b.WriteString("Assistant: ")
			b.WriteString(msg.Content)
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

// ChatCompletion drives one `claude -p --output-format json --max-turns
// 1 --dangerously-skip-permissions` invocation under this adapter's
// alias env matrix and returns the assistant's reply as an
// OpenAIChatResponse. A rate-limit/Fair-Usage signal in stderr or the
// JSON payload is surfaced as ErrClaudeCodeRateLimited (infra) rather
// than folded into a generic error or a false pass (§11.4.69/§11.4.107).
func (p *ClaudeCodeCLIAdapter) ChatCompletion(ctx context.Context, request OpenAIChatRequest) (*OpenAIChatResponse, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("Claude Code CLI not available: %v", p.checkErr)
	}

	prompt := buildClaudeCodePrompt(request.Messages)
	if prompt == "" {
		return nil, fmt.Errorf("no prompt provided")
	}

	model := ClaudeCodeDefaultModel
	if len(request.Model) > 0 {
		model = request.Model
	} else if p.alias.AnthropicModel != "" {
		model = p.alias.AnthropicModel
	}

	cmdCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--model", model,
		"--max-turns", "1",
		"--dangerously-skip-permissions",
	}

	cmd := exec.CommandContext(cmdCtx, p.cliPath, args...)
	cmd.Env = p.buildEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	stderrText := stderr.String()

	if runErr != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("claude code CLI timed out after %v", p.timeout)
		}
		detail := stderrText
		if detail == "" {
			detail = runErr.Error()
		}
		// PWU-2: quota checked BEFORE rate-limit (§11.4.6 -- quota is the
		// more specific, higher-cost-to-misclassify condition; the
		// closed-set signal lists do not currently overlap, but ordering
		// is preserved as defence-in-depth for future additions).
		if isQuotaSignal(stderrText) || isQuotaSignal(runErr.Error()) {
			return nil, fmt.Errorf("%w: %s", ErrClaudeCodeQuotaExceeded, firstNChars(detail, 200))
		}
		if isRateLimitSignal(stderrText) || isRateLimitSignal(runErr.Error()) {
			return nil, fmt.Errorf("%w: %s", ErrClaudeCodeRateLimited, firstNChars(detail, 200))
		}
		return nil, fmt.Errorf("claude code CLI failed: %w (stderr: %s)", runErr, firstNChars(stderrText, 200))
	}

	var result claudeCodeCLIResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("claude code CLI: failed to parse --output-format json result: %w (raw: %s)", err, firstNChars(stdout.String(), 200))
	}

	if result.IsError {
		raw := stdout.String()
		if isQuotaSignal(raw) || isQuotaSignal(stderrText) {
			return nil, fmt.Errorf("%w: subtype=%s terminal_reason=%s", ErrClaudeCodeQuotaExceeded, result.Subtype, result.TerminalReason)
		}
		if isRateLimitSignal(raw) || isRateLimitSignal(stderrText) {
			return nil, fmt.Errorf("%w: subtype=%s terminal_reason=%s", ErrClaudeCodeRateLimited, result.Subtype, result.TerminalReason)
		}
		return nil, fmt.Errorf("claude code CLI reported is_error=true: subtype=%s stop_reason=%s terminal_reason=%s", result.Subtype, result.StopReason, result.TerminalReason)
	}

	if strings.TrimSpace(result.Result) == "" {
		return nil, fmt.Errorf("claude code CLI: empty result content")
	}

	promptTokens := len(prompt) / 4
	completionTokens := len(result.Result) / 4

	response := &OpenAIChatResponse{
		ID:      newClaudeCodeResponseID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: result.Result,
				},
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}

	return response, nil
}

// StreamChatCompletion emulates streaming by running one ChatCompletion
// and emitting its result as a single terminal chunk (`claude -p
// --output-format json` is not a token-streamed transport) — mirrors
// kimicode.go's StreamChatCompletion wrapper exactly.
func (p *ClaudeCodeCLIAdapter) StreamChatCompletion(ctx context.Context, request OpenAIChatRequest) (<-chan OpenAIStreamResponse, <-chan error) {
	responseChan := make(chan OpenAIStreamResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		resp, err := p.ChatCompletion(ctx, request)
		if err != nil {
			errorChan <- err
			return
		}

		if len(resp.Choices) > 0 {
			finishReason := "stop"
			streamResp := OpenAIStreamResponse{
				ID:      resp.ID,
				Object:  "chat.completion.chunk",
				Created: resp.Created,
				Model:   resp.Model,
				Choices: []OpenAIChoice{
					{
						Index: resp.Choices[0].Index,
						Delta: OpenAIDelta{
							Role:    resp.Choices[0].Message.Role,
							Content: resp.Choices[0].Message.Content,
						},
						FinishReason: &finishReason,
					},
				},
			}
			responseChan <- streamResp
		}
	}()

	return responseChan, errorChan
}

// claudeCodeModel holds model metadata for the Claude-Code-CLI-bridge
// provider entry (mirrors kimicode.go's kimiCodeModel).
type claudeCodeModel struct {
	ID      string
	Object  string
	Created int64
	OwnedBy string
}

// GetKnownModels returns the single model this alias is configured for
// (or ClaudeCodeDefaultModel when the alias does not name one).
func (p *ClaudeCodeCLIAdapter) GetKnownModels() []claudeCodeModel {
	model := p.alias.AnthropicModel
	if model == "" {
		model = ClaudeCodeDefaultModel
	}
	return []claudeCodeModel{
		{ID: model, Object: "model", Created: time.Now().Unix(), OwnedBy: "claude-code-cli"},
	}
}

// ListModels satisfies the ListModels(ctx) (*OpenAIModelsResponse, error)
// interface ModelProviderService.fetchModelsFromAdapter type-asserts for
// (model_provider_service.go's ListModelsInterface).
func (p *ClaudeCodeCLIAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	models := p.GetKnownModels()
	resp := &OpenAIModelsResponse{Object: "list"}
	for _, m := range models {
		resp.Data = append(resp.Data, struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			ID:      m.ID,
			Object:  m.Object,
			Created: m.Created,
			OwnedBy: m.OwnedBy,
		})
	}
	return resp, nil
}

// GetProviderName identifies this adapter in provider-keyed maps/config.
func (p *ClaudeCodeCLIAdapter) GetProviderName() string {
	return "claude-code-cli"
}

// SupportsStreaming reports true because StreamChatCompletion is wired
// (as a single-final-chunk emulation — see its doc comment); mirrors
// kimicode.go's declared capability shape.
func (p *ClaudeCodeCLIAdapter) SupportsStreaming() bool {
	return true
}

// SupportsTools reports false: this text-shell bridge does not expose
// function/tool-calling.
func (p *ClaudeCodeCLIAdapter) SupportsTools() bool {
	return false
}

// HealthCheck drives one minimal ChatCompletion ("Reply with just
// 'OK'") to prove the alias is genuinely answering, mirroring
// kimicode.go's HealthCheck. Callers on a cadence budget (§11.4.69
// sink-side probing) should NOT invoke this more often than their
// project's already-defined health-probe interval — every invocation is
// a real, billed `claude -p` call.
func (p *ClaudeCodeCLIAdapter) HealthCheck(ctx context.Context) error {
	if !p.IsAvailable() {
		return p.checkErr
	}

	checkCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	model := p.alias.AnthropicModel
	if model == "" {
		model = ClaudeCodeDefaultModel
	}

	_, err := p.ChatCompletion(checkCtx, OpenAIChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "user", Content: "Reply with just 'OK'"},
		},
		MaxTokens: 10,
	})

	return err
}

// randomClaudeCodeIDSuffix and newClaudeCodeResponseID build a unique
// OpenAI-compatible response ID for this adapter (mirrors kimicode.go's
// randomIDSuffix/newKimiResponseID naming pattern; package-level
// randomIDSuffix from kimicode.go is reused directly rather than
// duplicated — §11.4.74 extend-don't-reimplement within the same
// package).
func newClaudeCodeResponseID() string {
	return fmt.Sprintf("claude-code-cli-%d-%s", time.Now().UnixNano(), randomIDSuffix())
}

// firstNChars returns the first n runes of s (UTF-8 aware), used to keep
// error messages bounded when embedding raw CLI stdout/stderr.
func firstNChars(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
