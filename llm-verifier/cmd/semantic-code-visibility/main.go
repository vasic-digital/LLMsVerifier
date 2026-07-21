// Command semantic-code-visibility is a generic, consumer-agnostic verifier
// that probes whether an OpenAI-compatible chat model can actually SEE a piece
// of code shown to it, and (optionally) whether it can DESCRIBE that code well
// enough to satisfy an independent judge model.
//
// It is deliberately project-not-aware: it hardcodes no consumer project name,
// path, prompt, fixture, or sentinel. Everything consumer-specific is supplied
// at runtime through CLI flags (the code fixture, the prompt template, the
// sentinel token, the provider endpoint/model, and — for round 2 — the judge
// endpoint/model). This keeps it reusable as shared infrastructure by any
// number of unrelated consumer projects.
//
// It depends only on the Go standard library (flag, os, net/http,
// encoding/json, ...) — it does NOT import the verification/providers/database
// packages, so it builds without the module's cgo/sqlite surface.
//
// Behaviour:
//
//	Round 1 (sentinel): interpolate the prompt template ({{FIXTURE_CONTENT}} ->
//	fixture contents, {{SENTINEL}} -> sentinel), POST it to the provider's
//	chat-completions endpoint (resolved from --base-url by chatCompletionsURL:
//	a base already ending in /chat/completions is used verbatim, a base ending
//	in a version segment /v[0-9]+ gets /chat/completions appended, anything
//	else gets /v1/chat/completions appended), and pass iff the reply contains
//	the exact sentinel substring.
//
//	Round 2 (judge, optional): only attempted when round 1 passes AND a full
//	judge flag set (--judge-base-url/--judge-model/--judge-api-key-env) is given.
//	The model-under-test is asked to describe the code, then the judge model
//	scores that description 0-3; round 2 passes iff score >= --judge-threshold.
//	Without judge flags, round 2 is reported "skipped" (never "failed").
//
// Anti-bluff: a transport error, timeout, non-200 status, or empty body ALWAYS
// yields pass=false with a reason — never a false pass. A round-1 reply that
// contains the sentinel but ALSO regurgitates a >=60-character verbatim slice
// of the fixture is treated as a prompt echo / bluff (genuine failure), not a
// pass. The bearer token is read ONLY from the env var named by --api-key-env
// (never taken as a flag value, never echoed into argv or the JSON output).
//
// Exit codes:
//
//	0  overall pass (round 1 passed, and round 2 passed or was skipped).
//	1  genuine verification failure — a round-1 or round-2 API call
//	   COMPLETED and yielded a real negative determination: the sentinel was
//	   not reflected back, the reply was a prompt echo / bluff, or the judge
//	   score was below --judge-threshold. This ALSO covers definitive
//	   provider rejections on the MODEL-UNDER-TEST calls (round 1 and the
//	   round-2 description call): HTTP 401, 402, 403, 404. Auth failure,
//	   depleted billing credits, and model-not-found are deterministic
//	   states, not transient infra — the model genuinely cannot work.
//	2  usage/config error — missing/invalid flags, unreadable fixture or
//	   prompt files, unset --api-key-env, etc.
//	3  infra/transport error — a round-1 or round-2 model/judge API call did
//	   NOT complete: HTTP 429 or 5xx, network/timeout error, or empty
//	   response/content. CRITICAL: ALL failures of the JUDGE call (round-2
//	   grading) stay exit 3 regardless of status code — a broken judge must
//	   never demote the model-under-test.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Generic (consumer-agnostic) defaults used only when the optional round-2
// flags are omitted. They name no consumer project and carry no branding.
const (
	defaultRound2Instruction = "Based on the code you were shown, describe in detail what it does, how it is structured, and its purpose."
	defaultJudgePrompt       = "You are grading whether a description accurately reflects some reference code.\n\n" +
		"Reference code:\n{{FIXTURE_CONTENT}}\n\n" +
		"Description to grade:\n{{DESCRIPTION}}\n\n" +
		"Rate how accurately the description reflects the reference code on an integer scale from 0 to 3 " +
		"(0 = unrelated, 1 = mostly wrong, 2 = mostly right, 3 = fully accurate). Reply with ONLY the single integer."

	maxResponseBytes = 1 << 20 // 1 MiB cap on any provider response body
)

// chatMessage / chatRequest / chatResponse model the minimal OpenAI-compatible
// chat-completions wire shapes this command needs.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		// FinishReason distinguishes "the model answered" from "the model ran
		// out of budget mid-generation" ("length"). Without it, a completion
		// truncated before it emitted any content is indistinguishable from a
		// model that genuinely returned nothing.
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// round2MaxTokens is the completion budget for the round-2 "describe the code"
// call. It must cover the model's REASONING tokens plus the answer: reasoning
// models emit reasoning first, and if it consumes the whole allowance the
// response comes back finish_reason="length" with EMPTY content.
//
// Measured live against a reasoning model on the real fixture prompt (varying
// ONLY this value): 512 truncated non-deterministically (one sample content=0,
// another content=1058 — reasoning length varies 194-1099 tokens per sampling);
// every sample at >=1024 finished "stop"; maximum observed completion was 1364
// tokens. 2048 is the smallest power of two clearing that with headroom.
//
// This is a CAP, not a charge — billing follows tokens actually generated — so
// raising it removes truncation without increasing cost for short answers.
const round2MaxTokens = 2048

// round1MaxTokens and judgeMaxTokens complete the same correction at the two
// remaining call sites. Reasoning models emit reasoning tokens BEFORE any
// answer, and max_tokens caps the WHOLE completion, so each budget must cover
// that site's reasoning plus its answer.
//
// Reasoning length is per-TASK, not merely per-model — measured on
// deepseek-v4-pro, 3 samples per budget:
//
//	round 1 (echo an 18-char sentinel): reasoning 58-97 tokens; 64 -> 0/3,
//	  128 -> 2/3, 256 -> 3/3. Max completion over n=10 at 2048: 128 tokens.
//	judge  (emit one integer):          reasoning 78-361 tokens; 64 -> 0/3,
//	  128 -> 0/3, 256 -> 1/3, 512 -> 3/3. Max completion over n=9: 500 tokens.
//
// So the two sites differ in kind. The JUDGE at 64 is a MEASURED ACTIVE defect:
// 0/3, every sample finish_reason="length" with empty content — the documented
// truncation shape. 512 would be the wrong fix: it passed 3/3 only as an
// artifact of small n, and widening to 9 samples surfaced a 500-token
// completion, leaving 12 tokens of headroom. 2048 (matching round2MaxTokens)
// removes the question.
//
// ROUND 1 at 256 was NOT broken for this model — it passed 3/3, and the
// deepseek false-negative came from round 2, not here. It is raised because
// reasoning distributions do not transfer between models: siliconflow returned
// the sentinel PREFIX "ZETA-9-ORANGE-" (severed mid-token) and was marked
// "cannot see code" while its layer-4 TUI turn PASSED. Round 1 is the site
// where truncation costs most — its failure is a definitive exit-1 that
// DE-VERIFIES a working provider, where round 2 only skips.
//
// Honest bound on all of this: one model, one provider, one prompt pair. A
// sample maximum is not a distributional ceiling (the 500-token judge sample
// appeared only after 3->9 samples), so these clear the observed tail with
// headroom rather than claiming a proven bound.
//
// max_tokens is a CAP, not a charge — billing follows tokens actually
// generated — so raising these removes truncation without raising the cost of
// short answers.
const (
	round1MaxTokens = 2048
	judgeMaxTokens  = 2048
)

// chatOutcome carries BOTH what the model said and why it stopped saying it.
//
// Returning the content alone loses the single fact needed to tell a
// non-compliant model from a severed response. Measured case (siliconflow,
// live run-proof): round 1 returned the observed string "ZETA-9-ORANGE-" for
// the sentinel "ZETA-9-ORANGE-7f3a" — a strict PREFIX, cut mid-token by the
// completion budget. With only the content in hand the caller reported
// "sentinel not found in response", which reads as "the model did not comply"
// and de-verified a provider whose live layer-4 TUI turn PASSED. The stop
// reason is what distinguishes the two, so it travels with the content.
type chatOutcome struct {
	Content      string
	FinishReason string
}

// round1Result is the sentinel-visibility outcome. infra is unexported (never
// serialized) and is true iff Pass=false because the round-1 API call could
// not complete (transport error, HTTP 429/5xx, empty body/content) rather
// than because it completed and genuinely lacked the sentinel, echoed the
// prompt, or was definitively rejected by the provider (HTTP 401/402/403/404).
type round1Result struct {
	Pass     bool   `json:"pass"`
	Observed string `json:"observed"`
	Reason   string `json:"reason,omitempty"`
	infra    bool
}

// round2Result is the judge outcome. Pass/Score are pointers so they serialize
// as JSON null when round 2 was not evaluated. infra is unexported (never
// serialized) and is true iff Pass=false because the round-2 model-describe
// call or the judge call could not complete, rather than because a completed
// call yielded a genuine below-threshold score.
type round2Result struct {
	Pass    *bool  `json:"pass"`
	Score   *int   `json:"score"`
	Skipped bool   `json:"skipped"`
	Reason  string `json:"reason,omitempty"`
	infra   bool
}

// report is the machine-readable output document.
type report struct {
	Round1  round1Result `json:"round1_sentinel"`
	Round2  round2Result `json:"round2_judge"`
	Overall bool         `json:"overall_pass"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point. Exit codes: 0 overall pass, 1 genuine
// verification fail (completed call with a real negative verdict, prompt
// echo / bluff, or a definitive provider rejection — HTTP 401/402/403/404 —
// on a model-under-test call), 2 usage/config error, 3 infra/transport error
// (a model or judge call could not complete: HTTP 429/5xx, timeout,
// transport, empty body; judge-call failures are ALWAYS exit 3).
// See the package doc comment above for the full table.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("semantic-code-visibility", flag.ContinueOnError)
	fs.SetOutput(stderr)

	baseURL := fs.String("base-url", "", "Provider chat endpoint base (e.g. https://api.deepseek.com)")
	model := fs.String("model", "", "Model id to send to the provider")
	apiKeyEnv := fs.String("api-key-env", "", "NAME of the env var holding the bearer token (read via os.Getenv; never the key itself)")
	fixturePath := fs.String("fixture", "", "Path to a file whose contents are the code to show the model")
	promptPath := fs.String("prompt", "", "Path to a prompt-template file containing {{FIXTURE_CONTENT}} and {{SENTINEL}}")
	sentinel := fs.String("sentinel", "", "Exact token to look for in the round-1 reply")
	timeoutSec := fs.Int("timeout", 60, "Per-request timeout in seconds")
	format := fs.String("format", "json", "Output format (only 'json' is supported)")

	round2PromptPath := fs.String("round2-prompt", "", "Optional path to a round-2 describe-prompt template (defaults to a generic instruction)")
	judgeBaseURL := fs.String("judge-base-url", "", "Optional judge provider chat endpoint base (enables round 2)")
	judgeModel := fs.String("judge-model", "", "Optional judge model id (enables round 2)")
	judgeKeyEnv := fs.String("judge-api-key-env", "", "Optional NAME of the env var holding the judge bearer token (enables round 2)")
	judgePromptPath := fs.String("judge-prompt", "", "Optional path to a judge scoring-prompt template with {{FIXTURE_CONTENT}} and {{DESCRIPTION}} (defaults to a generic rubric)")
	judgeThreshold := fs.Int("judge-threshold", 2, "Minimum judge score (0-3) required for round 2 to pass")

	if err := fs.Parse(args); err != nil {
		// flag already printed usage/err to stderr (includes -h ErrHelp).
		return 2
	}

	// --- config validation (exit 2 on any config/usage error) ---
	var missing []string
	for name, val := range map[string]string{
		"--base-url":    *baseURL,
		"--model":       *model,
		"--api-key-env": *apiKeyEnv,
		"--fixture":     *fixturePath,
		"--prompt":      *promptPath,
		"--sentinel":    *sentinel,
	} {
		if val == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(stderr, "config error: missing required flag(s): %s\n", strings.Join(missing, ", "))
		return 2
	}
	if *format != "json" {
		fmt.Fprintf(stderr, "config error: unsupported --format %q (only 'json' is supported)\n", *format)
		return 2
	}
	if *timeoutSec <= 0 {
		fmt.Fprintf(stderr, "config error: --timeout must be positive, got %d\n", *timeoutSec)
		return 2
	}

	apiKey := os.Getenv(*apiKeyEnv)
	if apiKey == "" {
		fmt.Fprintf(stderr, "config error: env var %q (from --api-key-env) is empty or unset\n", *apiKeyEnv)
		return 2
	}

	fixtureBytes, err := os.ReadFile(*fixturePath)
	if err != nil {
		fmt.Fprintf(stderr, "config error: cannot read --fixture: %v\n", err)
		return 2
	}
	promptBytes, err := os.ReadFile(*promptPath)
	if err != nil {
		fmt.Fprintf(stderr, "config error: cannot read --prompt: %v\n", err)
		return 2
	}

	// Judge flags are all-or-none. Any subset that is not the full set is a
	// config error rather than a silent partial-round-2.
	judgeAny := *judgeBaseURL != "" || *judgeModel != "" || *judgeKeyEnv != ""
	judgeAll := *judgeBaseURL != "" && *judgeModel != "" && *judgeKeyEnv != ""
	if judgeAny && !judgeAll {
		fmt.Fprintf(stderr, "config error: --judge-base-url, --judge-model and --judge-api-key-env must be set together (or all omitted)\n")
		return 2
	}

	var (
		judgeKey            string
		round2Instruction   = defaultRound2Instruction
		judgePromptTemplate = defaultJudgePrompt
	)
	if judgeAll {
		judgeKey = os.Getenv(*judgeKeyEnv)
		if judgeKey == "" {
			fmt.Fprintf(stderr, "config error: judge env var %q (from --judge-api-key-env) is empty or unset\n", *judgeKeyEnv)
			return 2
		}
		if *round2PromptPath != "" {
			b, rerr := os.ReadFile(*round2PromptPath)
			if rerr != nil {
				fmt.Fprintf(stderr, "config error: cannot read --round2-prompt: %v\n", rerr)
				return 2
			}
			round2Instruction = string(b)
		}
		if *judgePromptPath != "" {
			b, rerr := os.ReadFile(*judgePromptPath)
			if rerr != nil {
				fmt.Fprintf(stderr, "config error: cannot read --judge-prompt: %v\n", rerr)
				return 2
			}
			judgePromptTemplate = string(b)
		}
	}

	// --- build round-1 prompt ---
	round1Prompt := strings.ReplaceAll(string(promptBytes), "{{FIXTURE_CONTENT}}", string(fixtureBytes))
	round1Prompt = strings.ReplaceAll(round1Prompt, "{{SENTINEL}}", *sentinel)

	hc := &http.Client{Timeout: time.Duration(*timeoutSec) * time.Second}
	ctx := context.Background()

	rep := report{Round2: round2Result{Skipped: true}}

	// --- round 1 ---
	r1Out, err := chatComplete(ctx, hc, *baseURL, apiKey, *model, []chatMessage{
		{Role: "user", Content: round1Prompt},
	}, round1MaxTokens)
	if err != nil {
		// Anti-bluff: any failed/empty call is a fail with a reason, never a pass.
		// A definitive provider rejection (HTTP 401/402/403/404) is a genuine
		// negative determination (exit 1): auth failure, depleted billing
		// credits, and model-not-found are deterministic states, not transient
		// infra. Everything else (429/5xx/timeout/transport/empty) is an
		// infra/transport failure (exit 3) — the call never completed.
		if isDefinitiveRejection(err) {
			rep.Round1 = round1Result{Pass: false, Observed: "", Reason: "provider definitively rejected the request: " + err.Error()}
		} else {
			rep.Round1 = round1Result{Pass: false, Observed: "", Reason: err.Error(), infra: true}
		}
	} else {
		r1Content := r1Out.Content
		rep.Round1 = round1Result{
			Pass:     strings.Contains(r1Content, *sentinel),
			Observed: firstNChars(r1Content, 80),
		}
		if !rep.Round1.Pass {
			// A sentinel no-match is TWO different failures wearing one face.
			// If the completion hit its budget, the reply was severed mid-stream
			// and the sentinel can be half-present — measured live on
			// siliconflow: observed "ZETA-9-ORANGE-" against the sentinel
			// "ZETA-9-ORANGE-7f3a", a strict prefix cut mid-token. Reporting
			// that as "sentinel not found" blames the model for a budget WE
			// chose, and de-verified a provider whose layer-4 TUI turn PASSED.
			//
			// Classified infra (exit 3 => honest SKIP), NOT a definitive fail:
			// a truncated probe never completed, so it yields no verdict about
			// the model. Demoting on our own budget is the bluff to avoid.
			if r1Out.FinishReason == "length" {
				rep.Round1.Reason = fmt.Sprintf(
					"round-1 response truncated at max_tokens=%d (finish_reason=length): the "+
						"sentinel was cut off mid-stream, so this is a completion-budget "+
						"failure, not a model that cannot see the code", round1MaxTokens)
				rep.Round1.infra = true
			} else {
				rep.Round1.Reason = "sentinel not found in response"
			}
		} else if echoBluffDetected(r1Content, string(fixtureBytes)) {
			// Echo/bluff guard: a reply that contains the sentinel AND a
			// >=60-char verbatim slice of the fixture merely regurgitated the
			// prompt back — it never demonstrated code visibility. That is a
			// genuine negative determination (exit 1), not a pass.
			rep.Round1.Pass = false
			rep.Round1.Reason = "prompt echo / bluff: reply contains the sentinel plus a verbatim slice of the fixture"
		}
	}

	// --- round 2 (judge), only if enabled AND round 1 passed ---
	switch {
	case !judgeAll:
		rep.Round2 = round2Result{Skipped: true}
	case !rep.Round1.Pass:
		rep.Round2 = round2Result{Skipped: true, Reason: "round 1 did not pass; round 2 not attempted"}
	default:
		rep.Round2 = runRound2(ctx, hc, round2Params{
			modelBaseURL: *baseURL,
			modelKey:     apiKey,
			model:        *model,
			round1Prompt: round1Prompt,
			round1Reply:  r1Out.Content,
			round2Instr:  interpolate(round2Instruction, string(fixtureBytes), *sentinel),
			judgeBaseURL: *judgeBaseURL,
			judgeKey:     judgeKey,
			judgeModel:   *judgeModel,
			judgePrompt:  judgePromptTemplate,
			fixture:      string(fixtureBytes),
			threshold:    *judgeThreshold,
		})
	}

	// --- overall verdict ---
	if rep.Round2.Skipped {
		rep.Overall = rep.Round1.Pass
	} else {
		rep.Overall = rep.Round1.Pass && rep.Round2.Pass != nil && *rep.Round2.Pass
	}

	out, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "internal error: marshal report: %v\n", err)
		return 2
	}
	fmt.Fprintln(stdout, string(out))

	if rep.Overall {
		return 0
	}
	// Anti-bluff exit-code precision: an infra/transport failure (the model or
	// judge call could not complete: HTTP 429/5xx, timeout, transport, empty
	// body) is a different class of non-pass than a genuine verification
	// failure (sentinel not reflected, prompt echo / bluff, judge score below
	// threshold, or a definitive provider rejection — HTTP 401/402/403/404 —
	// on a model-under-test call). Exit 3 means "the verifier could not reach
	// a determination"; exit 1 means "the verifier reached a determination and
	// it was negative." Judge-call failures are ALWAYS infra (exit 3),
	// regardless of status code: a broken judge never demotes the model.
	if rep.Round1.infra || (!rep.Round2.Skipped && rep.Round2.infra) {
		return 3
	}
	return 1
}

type round2Params struct {
	modelBaseURL string
	modelKey     string
	model        string
	round1Prompt string
	round1Reply  string
	round2Instr  string
	judgeBaseURL string
	judgeKey     string
	judgeModel   string
	judgePrompt  string
	fixture      string
	threshold    int
}

// runRound2 continues the round-1 conversation to obtain a description, then
// asks the judge model to score it. Every failure yields pass=false with a
// reason (anti-bluff) — never a null/pass on a broken call.
func runRound2(ctx context.Context, hc *http.Client, p round2Params) round2Result {
	// failInfra marks the failure as infra/transport (exit 3): the model or
	// judge API call itself did not complete.
	failInfra := func(reason string) round2Result {
		f := false
		return round2Result{Pass: &f, Skipped: false, Reason: reason, infra: true}
	}
	// fail marks a genuine determination (exit 1): either the call completed
	// but the content could not be turned into a usable verdict (e.g. the
	// judge's reply had no parseable score), or the model-under-test call was
	// definitively rejected by the provider (HTTP 401/402/403/404) — both are
	// real negative determinations, not infra failures.
	fail := func(reason string) round2Result {
		f := false
		return round2Result{Pass: &f, Skipped: false, Reason: reason}
	}

	// Ask the model-under-test to describe the code (continued conversation).
	descOut, err := chatComplete(ctx, hc, p.modelBaseURL, p.modelKey, p.model, []chatMessage{
		{Role: "user", Content: p.round1Prompt},
		{Role: "assistant", Content: p.round1Reply},
		{Role: "user", Content: p.round2Instr},
	}, round2MaxTokens)
	if err != nil {
		// This is a MODEL-UNDER-TEST call: a definitive provider rejection
		// (HTTP 401/402/403/404) is a genuine negative determination (exit 1),
		// not transient infra.
		if isDefinitiveRejection(err) {
			return fail("round-2 model call definitively rejected by provider: " + err.Error())
		}
		return failInfra("round-2 model call failed: " + err.Error())
	}
	description := descOut.Content

	// Ask the judge to score the description. ALL judge-call failures are
	// infra (exit 3) regardless of status code — a broken judge must never
	// demote the model-under-test.
	judgePrompt := strings.ReplaceAll(p.judgePrompt, "{{FIXTURE_CONTENT}}", p.fixture)
	judgePrompt = strings.ReplaceAll(judgePrompt, "{{DESCRIPTION}}", description)
	judgeOut, err := chatComplete(ctx, hc, p.judgeBaseURL, p.judgeKey, p.judgeModel, []chatMessage{
		{Role: "user", Content: judgePrompt},
	}, judgeMaxTokens)
	if err != nil {
		return failInfra("judge call failed: " + err.Error())
	}
	judgeReply := judgeOut.Content

	score, ok := parseScore(judgeReply)
	if !ok {
		// A judge reply severed by its own completion budget is a JUDGE-SIDE
		// failure, and the policy stated directly above is that a broken judge
		// must never demote the model-under-test. Without this branch a
		// truncated judge reply fell through to fail() — exit 1 — de-verifying
		// a model on the strength of the judge running out of tokens.
		if judgeOut.FinishReason == "length" {
			return failInfra(fmt.Sprintf(
				"judge reply truncated at max_tokens=%d (finish_reason=length) before a "+
					"parseable score was emitted — judge-side budget failure, no verdict "+
					"about the model under test", judgeMaxTokens))
		}
		return fail("could not parse judge score from reply: " + firstNChars(judgeReply, 80))
	}

	pass := score >= p.threshold
	res := round2Result{Pass: &pass, Score: &score, Skipped: false}
	if !pass {
		res.Reason = fmt.Sprintf("judge score %d below threshold %d", score, p.threshold)
	}
	return res
}

// versionPathSegmentRe matches a trailing API-version path segment such as
// /v1, /v4, or /v12 at the end of a (slash-trimmed) base URL.
var versionPathSegmentRe = regexp.MustCompile(`/v[0-9]+$`)

// chatCompletionsURL resolves the chat-completions endpoint for a provider
// base URL. Provider bases come in three shapes:
//
//   - a base already ending in /chat/completions is used verbatim (no doubled
//     suffix);
//   - a base ending in a version segment /v[0-9]+ (e.g. /v1, /v4, /v12) gets
//     /chat/completions appended — e.g. https://api.z.ai/api/coding/paas/v4 ->
//     https://api.z.ai/api/coding/paas/v4/chat/completions (NOT .../paas/v4/v1/...);
//   - anything else (bare host or unversioned path) gets /v1/chat/completions
//     appended.
//
// Trailing slashes on the base are always trimmed first.
func chatCompletionsURL(base string) string {
	b := strings.TrimRight(base, "/")
	if strings.HasSuffix(b, "/chat/completions") {
		return b
	}
	if versionPathSegmentRe.MatchString(b) {
		return b + "/chat/completions"
	}
	return b + "/v1/chat/completions"
}

// chatComplete POSTs an OpenAI-compatible chat completion to the endpoint
// resolved by chatCompletionsURL(baseURL) with an Authorization: Bearer header
// and returns choices[0].message.content. It treats transport errors, non-200
// statuses, empty bodies, and empty content as errors (anti-bluff). A non-200
// status is returned as a *statusError so callers can classify definitive
// provider rejections (401/402/403/404) separately from transient ones
// (429/5xx) via isDefinitiveRejection.
func chatComplete(ctx context.Context, hc *http.Client, baseURL, apiKey, model string, messages []chatMessage, maxTokens int) (chatOutcome, error) {
	url := chatCompletionsURL(baseURL)
	payload, err := json.Marshal(chatRequest{Model: model, Messages: messages, MaxTokens: maxTokens})
	if err != nil {
		return chatOutcome{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return chatOutcome{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := hc.Do(req)
	if err != nil {
		return chatOutcome{}, fmt.Errorf("http call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return chatOutcome{}, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return chatOutcome{}, &statusError{Status: resp.StatusCode, Body: firstNChars(string(body), 200)}
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return chatOutcome{}, fmt.Errorf("empty response body")
	}

	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return chatOutcome{}, fmt.Errorf("decode response json: %w", err)
	}
	if len(cr.Choices) == 0 {
		return chatOutcome{}, fmt.Errorf("response had no choices")
	}
	content := cr.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		// A completion that hit the token ceiling BEFORE emitting any content is
		// a budget condition, not a silent model. Reasoning models spend tokens
		// on reasoning first, so a too-small max_tokens produces exactly this
		// shape — and reasoning length varies per sampling, which makes the
		// outcome non-deterministic. Reporting it as a bare "content was empty"
		// reads as "the model said nothing" and hides the real, fixable cause.
		if cr.Choices[0].FinishReason == "length" {
			return chatOutcome{}, fmt.Errorf("response truncated before any content was emitted: "+
				"finish_reason=length at max_tokens=%d (the completion budget was consumed, "+
				"typically by reasoning tokens, before an answer began)", maxTokens)
		}
		return chatOutcome{}, fmt.Errorf("response message content was empty")
	}
	return chatOutcome{Content: content, FinishReason: cr.Choices[0].FinishReason}, nil
}

// statusError is a non-200 provider response. Carrying the status code (and a
// truncated body for diagnostics) lets callers split deterministic rejections
// from transient infra failures instead of lumping every non-2xx together.
type statusError struct {
	Status int
	Body   string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("non-200 status %d: %s", e.Status, e.Body)
}

// isDefinitiveRejection reports whether err is a definitive provider rejection
// (HTTP 401, 402, 403, 404): auth failure, depleted billing credits, and
// model-not-found are deterministic states — retrying cannot help — so they
// count as a genuine negative determination (exit 1) on model-under-test
// calls, unlike HTTP 429/5xx/timeout/transport failures (exit 3).
func isDefinitiveRejection(err error) bool {
	var se *statusError
	if errors.As(err, &se) {
		switch se.Status {
		case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden, http.StatusNotFound:
			return true
		}
	}
	return false
}

// echoBluffRunes is the minimum verbatim rune overlap with the fixture that
// marks a sentinel-containing reply as a prompt echo / bluff.
const echoBluffRunes = 60

// echoBluffDetected reports whether the response regurgitates a >=60-rune
// verbatim slice of the fixture — the signature of a model that echoed the
// prompt back (sentinel included) instead of demonstrating that it can see
// the code. Whitespace runs are normalized to a single space on both sides
// before comparing, so re-wrapped or re-indented echoes are still caught.
func echoBluffDetected(response, fixture string) bool {
	normResp := normalizeWhitespace(response)
	fixRunes := []rune(normalizeWhitespace(fixture))
	if len(fixRunes) < echoBluffRunes {
		// Fixture too short for a reliable echo signature.
		return false
	}
	for i := 0; i+echoBluffRunes <= len(fixRunes); i++ {
		if strings.Contains(normResp, string(fixRunes[i:i+echoBluffRunes])) {
			return true
		}
	}
	return false
}

// normalizeWhitespace collapses every whitespace run to a single space.
func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// interpolate applies the same {{FIXTURE_CONTENT}}/{{SENTINEL}} substitutions
// used for the round-1 prompt (so a consumer-supplied round-2 template may
// reference them too).
func interpolate(tmpl, fixture, sentinel string) string {
	s := strings.ReplaceAll(tmpl, "{{FIXTURE_CONTENT}}", fixture)
	return strings.ReplaceAll(s, "{{SENTINEL}}", sentinel)
}

// parseScore extracts the first integer found in the judge reply.
func parseScore(s string) (int, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			continue
		}
		j := i
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		n, err := strconv.Atoi(s[i:j])
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// firstNChars returns the first n runes of s (UTF-8 aware).
func firstNChars(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
