// Command claude-alias-probe is the generic, consumer-agnostic CLI
// exposure of providers.ClaudeCodeCLIAdapter (providers/claudecode.go) --
// the Claude-Code-CLI-bridge PWU-1+2 of the LLMsVerifier incorporation
// landed as a Go-level adapter with no CLI entry point of its own
// (docs/research/llmsverifier_incorporation_20260707/ANALYSIS_AND_PLAN.md
// PART D / PART F PWU-3, ATMOSphere-Android-15 repo). PWU-3's
// project-agnostic health-probe utility (constitution/scripts/
// llm-alias-health/) is designed to EXEC the LLMsVerifier binary rather
// than import it as a Go dependency (§11.4.28(C) -- the consuming
// project's constitution submodule has an empty .gitmodules, so nested
// own-org submodules are forbidden; exec-decoupling is the sanctioned
// path per the incorporation plan PART C.4/PART G risk 7). This command
// is that exec target for the three claude-code alias kinds
// (native/provider-native/provider-router) -- the CLI-driven counterpart
// to cmd/semantic-code-visibility's HTTP-only prober.
//
// It is deliberately project-not-aware: no consumer project name, path,
// alias id, or credential value is hardcoded. Every alias-specific value
// (config dir, base URL, model, env-var NAMES holding secrets) is
// supplied at runtime via CLI flags, mirroring
// cmd/semantic-code-visibility's flag-injection convention (§11.4.10 --
// secrets are read ONLY from the env var NAMED by a flag, never taken as
// a flag value, never echoed into argv or the JSON output).
//
// Behaviour: build one alias's env matrix from --kind + the kind-specific
// flags, drive ONE `claude -p --output-format json --max-turns 1` call
// (via providers.ClaudeCodeCLIAdapter) asking the model to echo back
// --sentinel verbatim, and judge the reply by exact substring match -- no
// LLM judge, no false pass (mirrors cmd/semantic-code-visibility's
// round-1 discipline). A rate-limit/quota signal surfaced by the adapter
// (providers.ErrClaudeCodeRateLimited / ErrClaudeCodeQuotaExceeded, wired
// through providers.ClassifyProbeError -- PWU-2 of this incorporation)
// is reported as a DISTINCT verdict, never folded into a generic failure
// or a false "ok" (§11.4.69/§11.4.107).
//
// Exit codes:
//
//	0  verdict=="ok"      -- the sentinel was found in a genuine reply.
//	1  verdict=="failed"  -- the call completed but the sentinel was NOT
//	   found (a genuine content determination, not an infra condition).
//	2  usage/config error -- missing/invalid flags, unset/empty env var
//	   named by a --*-env flag, non-positive --timeout, etc.
//	3  infra/transport error -- verdict=="rate_limited" or
//	   verdict=="quota_exceeded", OR the CLI call itself could not
//	   complete (binary missing, timeout, unparseable --output-format
//	   json payload). Distinct from exit 1: the verifier could not reach
//	   a content determination at all.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"digital.vasic.llmsverifier/providers"
)

const defaultPromptTemplate = "Reply with EXACTLY this token and nothing else: {{SENTINEL}}"

// probeReport is the machine-readable output document. infra is
// unexported (never serialized) and mirrors
// cmd/semantic-code-visibility's round1Result.infra convention: true iff
// Verdict!="ok" because the call itself did not yield a genuine content
// determination (rate-limited/quota-exceeded/transport failure), as
// opposed to a completed call that genuinely lacked the sentinel.
type probeReport struct {
	Verdict           string `json:"verdict"`
	RetryAfterSeconds int    `json:"retry_after_seconds,omitempty"`
	Detail            string `json:"detail,omitempty"`
	LatencyMs         int64  `json:"latency_ms"`
	Model             string `json:"model"`
	SentinelMatched   bool   `json:"sentinel_matched"`
	Observed          string `json:"observed,omitempty"`
	infra             bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point. See the package doc comment for the
// full exit-code table.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("claude-alias-probe", flag.ContinueOnError)
	fs.SetOutput(stderr)

	kind := fs.String("kind", "", "Alias kind: native | provider-native | provider-router")
	configDir := fs.String("config-dir", "", "CLAUDE_CONFIG_DIR for kind=native")
	baseURL := fs.String("base-url", "", "ANTHROPIC_BASE_URL for kind=provider-native|provider-router")
	authTokenEnv := fs.String("auth-token-env", "", "NAME of the env var holding ANTHROPIC_AUTH_TOKEN (kind=provider-native, required)")
	apiKeyEnv := fs.String("api-key-env", "", "NAME of the env var holding ANTHROPIC_API_KEY (kind=provider-router; falls back to the literal \"ccr\" placeholder when unset -- ccr's front accepts any non-empty key, not a secret)")
	model := fs.String("model", "", "Model id (optional; falls back to the alias's own default)")
	sentinel := fs.String("sentinel", "", "Exact token to look for in the reply (required)")
	prompt := fs.String("prompt", "", "Optional inline prompt template containing {{SENTINEL}} (defaults to a generic instruction)")
	timeoutSec := fs.Int("timeout", 60, "Per-probe timeout in seconds")
	format := fs.String("format", "json", "Output format (only 'json' is supported)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *format != "json" {
		fmt.Fprintf(stderr, "config error: unsupported --format %q (only 'json' is supported)\n", *format)
		return 2
	}
	if *sentinel == "" {
		fmt.Fprintln(stderr, "config error: --sentinel is required")
		return 2
	}
	if *timeoutSec <= 0 {
		fmt.Fprintf(stderr, "config error: --timeout must be positive, got %d\n", *timeoutSec)
		return 2
	}

	aliasCfg, err := buildAliasConfig(*kind, *configDir, *baseURL, *authTokenEnv, *apiKeyEnv, *model)
	if err != nil {
		fmt.Fprintf(stderr, "config error: %v\n", err)
		return 2
	}

	promptTemplate := defaultPromptTemplate
	if *prompt != "" {
		promptTemplate = *prompt
	}
	finalPrompt := strings.ReplaceAll(promptTemplate, "{{SENTINEL}}", *sentinel)

	timeout := time.Duration(*timeoutSec) * time.Second
	adapter := providers.NewClaudeCodeCLIAdapter(aliasCfg, timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout+10*time.Second)
	defer cancel()

	start := time.Now()
	resp, callErr := adapter.ChatCompletion(ctx, providers.OpenAIChatRequest{
		Model:     *model,
		Messages:  []providers.Message{{Role: "user", Content: finalPrompt}},
		MaxTokens: 64,
	})
	latency := time.Since(start)

	var content, respModel string
	if callErr == nil && resp != nil && len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
		respModel = resp.Model
	}

	rep := judgeOutcome(content, callErr, *sentinel)
	rep.LatencyMs = latency.Milliseconds()
	rep.Model = *model
	if rep.Model == "" {
		rep.Model = respModel
	}

	out, merr := json.MarshalIndent(rep, "", "  ")
	if merr != nil {
		fmt.Fprintf(stderr, "internal error: marshal report: %v\n", merr)
		return 2
	}
	fmt.Fprintln(stdout, string(out))

	switch {
	case rep.Verdict == "ok":
		return 0
	case rep.infra:
		return 3
	default:
		return 1
	}
}

// buildAliasConfig validates --kind + its required companion flags and
// returns the providers.ClaudeCodeAliasConfig the adapter needs. Every
// branch reads secrets ONLY from the named env var (§11.4.10) -- never
// from a flag value.
func buildAliasConfig(kind, configDir, baseURL, authTokenEnv, apiKeyEnv, model string) (providers.ClaudeCodeAliasConfig, error) {
	switch kind {
	case string(providers.ClaudeCodeAliasNative):
		if configDir == "" {
			return providers.ClaudeCodeAliasConfig{}, fmt.Errorf("--config-dir is required for --kind=native")
		}
		return providers.ClaudeCodeAliasConfig{
			Kind:           providers.ClaudeCodeAliasNative,
			ConfigDir:      configDir,
			AnthropicModel: model,
		}, nil

	case string(providers.ClaudeCodeAliasProviderNative):
		if baseURL == "" {
			return providers.ClaudeCodeAliasConfig{}, fmt.Errorf("--base-url is required for --kind=provider-native")
		}
		if authTokenEnv == "" {
			return providers.ClaudeCodeAliasConfig{}, fmt.Errorf("--auth-token-env is required for --kind=provider-native")
		}
		token := os.Getenv(authTokenEnv)
		if token == "" {
			return providers.ClaudeCodeAliasConfig{}, fmt.Errorf("env var %q (from --auth-token-env) is empty or unset", authTokenEnv)
		}
		return providers.ClaudeCodeAliasConfig{
			Kind:               providers.ClaudeCodeAliasProviderNative,
			AnthropicBaseURL:   baseURL,
			AnthropicAuthToken: token,
			AnthropicModel:     model,
		}, nil

	case string(providers.ClaudeCodeAliasProviderRouter):
		if baseURL == "" {
			return providers.ClaudeCodeAliasConfig{}, fmt.Errorf("--base-url is required for --kind=provider-router")
		}
		key := "ccr"
		if apiKeyEnv != "" {
			key = os.Getenv(apiKeyEnv)
			if key == "" {
				return providers.ClaudeCodeAliasConfig{}, fmt.Errorf("env var %q (from --api-key-env) is empty or unset", apiKeyEnv)
			}
		}
		return providers.ClaudeCodeAliasConfig{
			Kind:             providers.ClaudeCodeAliasProviderRouter,
			AnthropicBaseURL: baseURL,
			AnthropicAPIKey:  key,
			AnthropicModel:   model,
		}, nil

	default:
		return providers.ClaudeCodeAliasConfig{}, fmt.Errorf("unknown --kind %q (must be native|provider-native|provider-router)", kind)
	}
}

// judgeOutcome maps a completed call's (content, sentinel) pair OR a
// failed call's error into the closed-set probeReport verdict. It is a
// pure function (no exec, no I/O) so unit tests can exercise every
// branch -- including the rate_limited/quota_exceeded classes -- without
// spending any real LLM quota (§11.4.6 G.1 quota-minimal probing).
//
// Anti-bluff (§11.4/§11.4.69): a call error is ALWAYS infra=true (the
// verifier could not reach a content determination); only a completed
// call that genuinely lacks the sentinel is a non-infra "failed"
// (a real, negative determination).
func judgeOutcome(content string, callErr error, sentinel string) probeReport {
	if callErr != nil {
		outcome := providers.ClassifyProbeError(callErr)
		return probeReport{
			Verdict:           string(outcome.Verdict),
			RetryAfterSeconds: outcome.RetryAfterSeconds,
			Detail:            outcome.Detail,
			infra:             true,
		}
	}

	matched := strings.Contains(content, sentinel)
	rep := probeReport{
		Verdict:         "ok",
		SentinelMatched: matched,
		Observed:        firstNChars(content, 120),
	}
	if !matched {
		rep.Verdict = "failed"
		rep.Detail = "sentinel not found in response"
	}
	return rep
}

// firstNChars returns the first n runes of s (UTF-8 aware), used to keep
// the JSON payload bounded.
func firstNChars(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
