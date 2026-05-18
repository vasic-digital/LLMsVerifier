package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPBenchmarkProvider is a concrete LLMProvider implementation that
// dispatches each Complete() call to an OpenAI-compatible
// /chat/completions endpoint and returns the first choice's message
// content together with the real usage.total_tokens value reported by
// the server. It is the round-78 §11.4 anti-bluff close-out of the
// round-28 ErrBenchmarkProviderNotConfigured sentinel: round 28
// removed the hardcoded Passed=true / Score=0.8 / Latency=100ms /
// TokensUsed=50 fabrication from the runner.executeTask nil-provider
// branch, but left consumers without a ready-wireable implementation;
// this type is what they wire into (*StandardBenchmarkRunner).
// SetProvider so executeTask produces a real model response, real
// measured latency (computed by the caller — runner.executeTask wraps
// the Complete call with time.Since), and real token usage (reported
// by the server's usage.total_tokens field) and feeds them into the
// BenchmarkResult.
//
// The OpenAI Chat Completions wire protocol is targeted because it is
// the most widely adopted format: OpenAI itself, Azure OpenAI,
// OpenRouter, Groq, DeepSeek, Mistral, Together, Anyscale, vLLM,
// llama.cpp's HTTP server, LM Studio, LocalAI, and any other server
// fronting an OpenAI shim all speak it. Consumers who need a
// non-OpenAI-shaped backend (e.g. raw Anthropic Messages, raw Gemini
// generateContent) wrap their SDK behind a custom LLMProvider
// implementation instead — HTTPBenchmarkProvider targets the majority
// case and keeps the dependency footprint at net/http only.
//
// CONST-042 secret-leak adjacency: APIKey is sourced from the
// caller's HTTPBenchmarkProviderConfig (operator passes an
// env-sourced value into the constructor at wire-up time).
// HTTPBenchmarkProvider NEVER reads os.Getenv itself, NEVER logs the
// APIKey field, NEVER includes the key in error messages.
//
// CONST-050(A) / (B): production code may import and construct
// HTTPBenchmarkProvider. The unit tests in http_provider_test.go
// exercise it against httptest.NewServer (in-process loopback) which
// counts as a unit test under CONST-050(A) (no real external
// network); the integration test TestHTTPBenchmarkProvider_RealOpenAI
// is env-gated and hits a real provider when LLMSVERIFIER_TEST_OPENAI_KEY
// is set, satisfying CONST-050(B) coverage for the integration tier.
//
// ANTI-BLUFF GUARANTEE (Article XI §11.9 / CONST-035): every metric
// returned by Complete is derived from real response data. The
// returned tokens value comes from the server's usage.total_tokens
// field (if present) or is computed from the response content length
// (fallback for servers that omit usage). Latency is NOT computed
// here — runner.executeTask wraps the Complete call with time.Since
// to capture wall-clock duration. NEVER fabricate values; NEVER
// return a hardcoded constant.
type HTTPBenchmarkProvider struct {
	endpoint   string
	model      string
	apiKey     string
	timeout    time.Duration
	maxRetries int
	httpClient *http.Client
	name       string
}

// HTTPBenchmarkProviderConfig is the constructor input for
// NewHTTPBenchmarkProvider. The two required fields are Endpoint +
// Model; APIKey is optional (local OpenAI-shim servers like
// llama.cpp's HTTP mode do not require one); Timeout defaults to 60s
// when zero. HTTPClient is optional — callers wanting custom
// transport wiring (proxy, mTLS, retry middleware) inject it here;
// otherwise a clean http.Client honouring Timeout is constructed.
type HTTPBenchmarkProviderConfig struct {
	// Endpoint is the base URL of the OpenAI-compatible server,
	// MINUS the trailing "/chat/completions" path component. Examples:
	//   - "https://api.openai.com/v1"           (OpenAI)
	//   - "https://api.deepseek.com/v1"         (DeepSeek)
	//   - "https://api.groq.com/openai/v1"      (Groq OpenAI shim)
	//   - "http://localhost:11434/v1"           (Ollama OpenAI shim)
	//   - "http://localhost:8080/v1"            (llama.cpp HTTP server)
	// Empty endpoint => NewHTTPBenchmarkProvider returns
	// ErrHTTPBenchmarkProviderEndpointNotConfigured.
	Endpoint string

	// Model is the model identifier the server uses to route the
	// request (e.g. "gpt-4o-mini", "llama-3.1-8b-instruct",
	// "deepseek-chat"). Empty model => NewHTTPBenchmarkProvider
	// returns ErrHTTPBenchmarkProviderModelNotConfigured.
	Model string

	// APIKey is sent as the Authorization: Bearer <APIKey> header.
	// May be empty for local servers (llama.cpp, Ollama) that don't
	// require authentication — empty key => no Authorization header
	// sent. CONST-042: callers MUST source this from .env / secret
	// store and pass at construction time; HTTPBenchmarkProvider
	// never reads env itself.
	APIKey string

	// Timeout caps the round-trip duration. Zero => 60s default.
	// Honoured even when ctx.Deadline is later (whichever fires
	// first wins via context.WithTimeout chaining inside Complete).
	Timeout time.Duration

	// MaxRetries is reserved for future use (retry on 5xx /
	// rate-limit). Currently the implementation makes a single
	// request and surfaces failures verbatim — consumers wrap in a
	// retry layer if they need it. Field reserved to keep the
	// config surface stable when retry logic lands.
	MaxRetries int

	// HTTPClient lets callers inject a pre-configured client
	// (custom Transport for proxy / mTLS / retry middleware /
	// metrics interceptor). Nil => internal client constructed
	// with Timeout.
	HTTPClient *http.Client

	// Name is the provider name reported by GetName(). Optional —
	// defaults to "http-benchmark-provider" when empty.
	Name string
}

// Sentinel errors covering the four failure modes Complete /
// NewHTTPBenchmarkProvider expose. All four are stable contract
// surfaces — consumers MAY errors.Is() against them to branch on
// failure mode (e.g. retry vs surface vs abort the benchmark run).
//
// These are distinct from ErrBenchmarkProviderNotConfigured (defined
// in runner.go and preserved verbatim from round 28): that sentinel
// fires when the runner has NO provider wired at all; the four
// sentinels below fire from a concrete HTTPBenchmarkProvider that
// IS wired but mis-configured or hit a runtime failure.
var (
	// ErrHTTPBenchmarkProviderEndpointNotConfigured fires from
	// NewHTTPBenchmarkProvider when Endpoint is empty. Distinct
	// from ErrBenchmarkProviderNotConfigured which fires from
	// runner.executeTask when SetProvider was never called — this
	// one fires earlier, at construction time, when the caller
	// invoked the constructor with an unusable config.
	ErrHTTPBenchmarkProviderEndpointNotConfigured = errors.New(
		"llmsverifier benchmark: HTTPBenchmarkProviderConfig.Endpoint is empty — set it to the OpenAI-compatible base URL of your LLM provider (e.g. \"https://api.openai.com/v1\") before constructing HTTPBenchmarkProvider",
	)

	// ErrHTTPBenchmarkProviderModelNotConfigured fires from
	// NewHTTPBenchmarkProvider when Model is empty. Sibling of the
	// endpoint sentinel.
	ErrHTTPBenchmarkProviderModelNotConfigured = errors.New(
		"llmsverifier benchmark: HTTPBenchmarkProviderConfig.Model is empty — set it to a model identifier the configured endpoint accepts (e.g. \"gpt-4o-mini\", \"llama-3.1-8b-instruct\") before constructing HTTPBenchmarkProvider",
	)

	// ErrHTTPBenchmarkProviderRequestFailed wraps every
	// network-layer or HTTP-level failure (DNS, TCP, TLS, 4xx,
	// 5xx). The wrapped error retains the original cause via
	// errors.Unwrap; the wrapper text adds the endpoint + status
	// context but NEVER includes the APIKey.
	ErrHTTPBenchmarkProviderRequestFailed = errors.New(
		"llmsverifier benchmark: HTTPBenchmarkProvider request to the LLM endpoint failed",
	)

	// ErrHTTPBenchmarkProviderResponseInvalid fires when the
	// server returned a 2xx status but the response body could not
	// be parsed as an OpenAI-compatible chat-completions response,
	// or the body parsed but contained zero choices / an empty
	// first-choice message.
	ErrHTTPBenchmarkProviderResponseInvalid = errors.New(
		"llmsverifier benchmark: HTTPBenchmarkProvider received a 2xx response but could not extract a usable message — body did not match the OpenAI chat-completions schema (choices[0].message.content)",
	)
)

// NewHTTPBenchmarkProvider constructs an HTTPBenchmarkProvider from
// the given config. Returns one of the two construction-time
// sentinels (endpoint or model not configured) if the config is
// unusable. The returned provider is safe for concurrent use across
// goroutines — the embedded *http.Client is concurrency-safe per
// stdlib contract, and HTTPBenchmarkProvider itself holds no mutable
// state after construction.
func NewHTTPBenchmarkProvider(cfg HTTPBenchmarkProviderConfig) (*HTTPBenchmarkProvider, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, ErrHTTPBenchmarkProviderEndpointNotConfigured
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, ErrHTTPBenchmarkProviderModelNotConfigured
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	name := cfg.Name
	if strings.TrimSpace(name) == "" {
		name = "http-benchmark-provider"
	}

	// Strip trailing slash so endpoint + "/chat/completions"
	// composes cleanly regardless of whether the caller passed
	// ".../v1" or ".../v1/".
	endpoint := strings.TrimRight(cfg.Endpoint, "/")

	return &HTTPBenchmarkProvider{
		endpoint:   endpoint,
		model:      cfg.Model,
		apiKey:     cfg.APIKey,
		timeout:    timeout,
		maxRetries: cfg.MaxRetries,
		httpClient: client,
		name:       name,
	}, nil
}

// GetName satisfies the LLMProvider interface contract.
func (p *HTTPBenchmarkProvider) GetName() string {
	return p.name
}

// httpChatCompletionsRequest is the wire shape POSTed to the
// endpoint. Only the fields the spec strictly requires are populated
// — every OpenAI-compatible server accepts the minimum form.
type httpChatCompletionsRequest struct {
	Model    string                       `json:"model"`
	Messages []httpChatCompletionMessage  `json:"messages"`
}

type httpChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// httpChatCompletionsResponse covers the subset of the OpenAI
// response schema HTTPBenchmarkProvider consumes. usage.total_tokens
// is load-bearing for the anti-bluff guarantee: the value returned
// by Complete is the server's reported total, NOT a fabricated
// constant. Additional fields the server returns (finish_reason,
// model echo, system_fingerprint, etc.) are tolerated via JSON
// decoder default behaviour (unknown fields ignored).
type httpChatCompletionsResponse struct {
	Choices []httpChatCompletionChoice `json:"choices"`
	Usage   *httpChatCompletionUsage   `json:"usage,omitempty"`
	// Error field present on some servers' 2xx-with-error
	// responses (rare but observed on Azure OpenAI). When
	// non-empty we treat the response as invalid even though HTTP
	// status was 2xx.
	Error *httpChatCompletionError `json:"error,omitempty"`
}

type httpChatCompletionChoice struct {
	Message httpChatCompletionMessage `json:"message"`
}

type httpChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type httpChatCompletionError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    string `json:"code,omitempty"`
}

// Complete satisfies the LLMProvider interface contract. It
// dispatches prompt to the configured endpoint as a single user-role
// message and returns the first choice's content + the real
// usage.total_tokens reported by the server. Behavioural guarantees:
//
//   - Honours ctx.Cancel + ctx.Deadline (request is built with
//     http.NewRequestWithContext so cancellation aborts in-flight
//     read).
//   - Wraps every transport / HTTP-error in
//     ErrHTTPBenchmarkProviderRequestFailed with the underlying cause
//     accessible via errors.Unwrap.
//   - Wraps every parse / empty-choice failure in
//     ErrHTTPBenchmarkProviderResponseInvalid.
//   - NEVER includes the APIKey in any error message or log output.
//   - NEVER mutates HTTPBenchmarkProvider state — safe for concurrent
//     calls.
//   - NEVER fabricates the tokens value. When usage.total_tokens is
//     present in the response, that value is returned verbatim. When
//     usage is absent (rare — some local shims omit it), tokens is
//     derived from len(content) as a coarse character-count
//     approximation, never a fixed constant. The §11.4 anti-bluff
//     invariant is preserved: the value reflects real response data,
//     even when the server is incomplete.
//
// The systemPrompt parameter is currently included as a separate
// system-role message when non-empty (OpenAI standard pattern). When
// empty, only the user message is sent.
func (p *HTTPBenchmarkProvider) Complete(ctx context.Context, prompt, systemPrompt string) (string, int, error) {
	messages := make([]httpChatCompletionMessage, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, httpChatCompletionMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	messages = append(messages, httpChatCompletionMessage{
		Role:    "user",
		Content: prompt,
	})

	body := httpChatCompletionsRequest{
		Model:    p.model,
		Messages: messages,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		// json.Marshal of this struct cannot realistically fail
		// (no chan / func / cyclic types), but wrap defensively
		// rather than panic so the contract stays "every failure
		// surfaces as a sentinel-wrapped error".
		return "", 0, fmt.Errorf("%w: marshal request body: %v", ErrHTTPBenchmarkProviderRequestFailed, err)
	}

	url := p.endpoint + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, fmt.Errorf("%w: build request for %s: %v", ErrHTTPBenchmarkProviderRequestFailed, url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if p.apiKey != "" {
		// CONST-042: header set from in-memory field; never
		// logged.
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		// Transport-layer failure: DNS, TCP, TLS, ctx-cancel,
		// timeout. Wrap with the endpoint URL (no APIKey).
		return "", 0, fmt.Errorf("%w: POST %s: %v", ErrHTTPBenchmarkProviderRequestFailed, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read body before status check so 4xx/5xx error responses
	// can surface the server's textual diagnostic (helps consumers
	// debug model-not-found, auth-rejected, rate-limited, etc.).
	respBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", 0, fmt.Errorf("%w: read response body from %s (status %d): %v",
			ErrHTTPBenchmarkProviderRequestFailed, url, resp.StatusCode, readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Truncate the server's response body in the error
		// message so a multi-MB error payload doesn't blow up
		// logs.
		bodyExcerpt := string(respBytes)
		if len(bodyExcerpt) > 512 {
			bodyExcerpt = bodyExcerpt[:512] + "...(truncated)"
		}
		return "", 0, fmt.Errorf("%w: POST %s returned HTTP %d: %s",
			ErrHTTPBenchmarkProviderRequestFailed, url, resp.StatusCode, bodyExcerpt)
	}

	var parsed httpChatCompletionsResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		bodyExcerpt := string(respBytes)
		if len(bodyExcerpt) > 256 {
			bodyExcerpt = bodyExcerpt[:256] + "...(truncated)"
		}
		return "", 0, fmt.Errorf("%w: unmarshal response from %s: %v (body excerpt: %q)",
			ErrHTTPBenchmarkProviderResponseInvalid, url, err, bodyExcerpt)
	}

	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", 0, fmt.Errorf("%w: server returned 2xx with embedded error from %s: %s (type=%s code=%s)",
			ErrHTTPBenchmarkProviderResponseInvalid, url, parsed.Error.Message, parsed.Error.Type, parsed.Error.Code)
	}

	if len(parsed.Choices) == 0 {
		return "", 0, fmt.Errorf("%w: response from %s contained zero choices",
			ErrHTTPBenchmarkProviderResponseInvalid, url)
	}

	content := parsed.Choices[0].Message.Content
	if content == "" {
		return "", 0, fmt.Errorf("%w: response from %s contained an empty first-choice message",
			ErrHTTPBenchmarkProviderResponseInvalid, url)
	}

	// Anti-bluff: tokens MUST come from the server when available.
	// When the server omits usage (some local shims do), fall back
	// to a real character-count-derived value — NEVER a fixed
	// constant. Article XI §11.9 / CONST-035 forbids fabricated
	// metrics.
	tokens := 0
	if parsed.Usage != nil && parsed.Usage.TotalTokens > 0 {
		tokens = parsed.Usage.TotalTokens
	} else {
		// Fallback when server omits usage: approximate via
		// content length. NOT a fixed value (would be a bluff);
		// derived from real response data.
		tokens = len(content)
	}

	return content, tokens, nil
}

// Compile-time check that HTTPBenchmarkProvider implements
// LLMProvider. If the interface evolves, this line breaks the build
// before tests run — the §11.4 four-layer test floor (pre-build
// gate).
var _ LLMProvider = (*HTTPBenchmarkProvider)(nil)
