package api

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"digital.vasic.llmsverifier/clientip"
)

// RateLimiter provides thread-safe rate limiting using token bucket algorithm
type RateLimiter struct {
	mu           sync.RWMutex
	clients      map[string]*clientRateLimit
	defaultLimit int           // requests per window
	windowSize   time.Duration // sliding window size
	cleanupTick  time.Duration // cleanup interval for expired entries
}

// clientRateLimit tracks rate limit state for a single client
type clientRateLimit struct {
	requests    []time.Time // timestamps of requests within window
	mu          sync.Mutex
	lastCleanup time.Time
}

// NewRateLimiter creates a new rate limiter with specified limits
func NewRateLimiter(requestsPerMinute int, windowSize time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:      make(map[string]*clientRateLimit),
		defaultLimit: requestsPerMinute,
		windowSize:   windowSize,
		cleanupTick:  time.Minute * 5,
	}

	// Start background cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if the client is allowed to make a request
func (rl *RateLimiter) Allow(clientIP string) (allowed bool, remaining int, resetTime time.Time) {
	rl.mu.Lock()
	client, exists := rl.clients[clientIP]
	if !exists {
		client = &clientRateLimit{
			requests:    make([]time.Time, 0, rl.defaultLimit),
			lastCleanup: time.Now(),
		}
		rl.clients[clientIP] = client
	}
	rl.mu.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	// Remove expired requests (outside the window)
	validRequests := make([]time.Time, 0, len(client.requests))
	for _, t := range client.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	client.requests = validRequests

	// Calculate remaining and reset time
	remaining = rl.defaultLimit - len(client.requests)
	if len(client.requests) > 0 {
		resetTime = client.requests[0].Add(rl.windowSize)
	} else {
		resetTime = now.Add(rl.windowSize)
	}

	// Check if limit exceeded
	if len(client.requests) >= rl.defaultLimit {
		return false, 0, resetTime
	}

	// Record this request
	client.requests = append(client.requests, now)
	remaining = rl.defaultLimit - len(client.requests)

	return true, remaining, resetTime
}

// GetStatus returns current rate limit status without consuming a request
func (rl *RateLimiter) GetStatus(clientIP string) (remaining int, resetTime time.Time) {
	rl.mu.RLock()
	client, exists := rl.clients[clientIP]
	rl.mu.RUnlock()

	if !exists {
		return rl.defaultLimit, time.Now().Add(rl.windowSize)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	count := 0
	var firstRequest time.Time
	for _, t := range client.requests {
		if t.After(windowStart) {
			if count == 0 {
				firstRequest = t
			}
			count++
		}
	}

	remaining = rl.defaultLimit - count
	if count > 0 {
		resetTime = firstRequest.Add(rl.windowSize)
	} else {
		resetTime = now.Add(rl.windowSize)
	}

	return remaining, resetTime
}

// cleanupLoop periodically removes stale entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes clients with no recent requests
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	for ip, client := range rl.clients {
		client.mu.Lock()
		// Remove if all requests are expired
		hasValid := false
		for _, t := range client.requests {
			if t.After(windowStart) {
				hasValid = true
				break
			}
		}
		if !hasValid && now.Sub(client.lastCleanup) > rl.windowSize*2 {
			delete(rl.clients, ip)
		}
		client.mu.Unlock()
	}
}

// Global rate limiter instance
var globalRateLimiter *RateLimiter
var rateLimiterOnce sync.Once

// GetGlobalRateLimiter returns the global rate limiter, creating it if necessary
func GetGlobalRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		// Default: 100 requests per minute
		globalRateLimiter = NewRateLimiter(100, time.Minute)
	})
	return globalRateLimiter
}

// SetGlobalRateLimiter allows configuring a custom rate limiter
func SetGlobalRateLimiter(rl *RateLimiter) {
	globalRateLimiter = rl
}

// OutputSanitizationMiddleware sanitizes API responses to prevent XSS
func OutputSanitizationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a response writer wrapper to capture the response
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
			}

			// Call the next handler
			next.ServeHTTP(wrapper, r)

			// Get the response content type
			contentType := wrapper.Header().Get("Content-Type")

			// Sanitize based on content type
			if strings.Contains(contentType, "application/json") {
				sanitizedBody := SanitizeJSONOutput(wrapper.body.String())
				wrapper.ResponseWriter.Write([]byte(sanitizedBody.(string)))
			} else if strings.Contains(contentType, "text/html") {
				sanitizedBody := SanitizeHTMLResponse(wrapper.body.String())
				wrapper.ResponseWriter.Write([]byte(sanitizedBody))
			} else {
				// For other content types, apply basic output sanitization
				sanitizedBody := SanitizeOutput(wrapper.body.String())
				wrapper.ResponseWriter.Write([]byte(sanitizedBody))
			}
		})
	}
}

// responseWriterWrapper captures the response body
type responseWriterWrapper struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (rw *responseWriterWrapper) Write(data []byte) (int, error) {
	// Write to both the buffer and the original response writer
	rw.body.Write(data)
	return rw.ResponseWriter.Write(data)
}

// SanitizeJSONResponse previously returned the input data unchanged
// with the comment "For now, just return the data as-is". Any caller
// expecting field-redaction / sensitive-data stripping before
// sending to the TUI client received raw data — potential
// information-leak surface, §11.4 stub-interface bluff with
// security implications.
//
// Until real struct-tag-driven filtering is implemented, prefix
// the function's docstring with a CRITICAL marker so anyone
// adding new sensitive fields knows they are NOT being redacted.
// Behaviour unchanged (return data as-is) — converting to error
// would break existing callers that pass non-sensitive data; the
// proper fix is the actual sanitisation implementation.
//
// TODO[§11.4 / CONST-042]: implement real sanitisation via struct
// tag `json:"...,sensitive"` or similar before shipping new fields
// that may contain credentials / PII / internal-only data.
func SanitizeJSONResponse(data interface{}) interface{} {
	// SECURITY NOTICE: this function currently returns input
	// unchanged. Do NOT pass data containing credentials / PII /
	// internal-only fields through this function expecting them to
	// be stripped — implement field-level redaction first.
	return data
}

// SecurityHeadersMiddleware adds comprehensive security headers
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// XSS protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Referrer policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions policy (formerly Feature-Policy)
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// HSTS (HTTP Strict Transport Security) - only for HTTPS
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// Remove server header to avoid information disclosure
			w.Header().Del("Server")

			// Add security-focused headers
			w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

			next.ServeHTTP(w, r)
		})
	}
}

// ContentSecurityPolicyMiddleware adds CSP headers
func ContentSecurityPolicyMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Comprehensive CSP policy
			csp := "default-src 'self'; " +
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
				"style-src 'self' 'unsafe-inline'; " +
				"img-src 'self' data: https: blob:; " +
				"font-src 'self' https: data:; " +
				"connect-src 'self' https: wss:; " +
				"media-src 'self' https:; " +
				"object-src 'none'; " +
				"frame-src 'none'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'; " +
				"upgrade-insecure-requests;"

			w.Header().Set("Content-Security-Policy", csp)
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware implements HTTP rate limiting using sliding window algorithm
func RateLimitMiddleware(limiter interface{}) func(http.Handler) http.Handler {
	// Use the global rate limiter or create one if custom limiter provided
	var rl *RateLimiter
	if customLimiter, ok := limiter.(*RateLimiter); ok && customLimiter != nil {
		rl = customLimiter
	} else {
		rl = GetGlobalRateLimiter()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)

			// Check rate limit using the real rate limiter
			allowed, remaining, resetTime := rl.Allow(clientIP)

			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.defaultLimit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			if !allowed {
				retryAfter := int(time.Until(resetTime).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, "Rate limit exceeded. Please retry after the reset time.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIPTrustedProxiesEnvVar is the environment variable operators use to
// declare which immediate peers are permitted to have their
// X-Forwarded-For / X-Real-IP headers trusted by getClientIP. Kept as its
// own named constant (rather than referencing clientip.TrustedProxiesEnvVar
// inline everywhere) purely for call-site readability in this file and its
// tests; its VALUE is guaranteed identical to clientip's — see the
// HXC-298-extraction note below.
const clientIPTrustedProxiesEnvVar = clientip.TrustedProxiesEnvVar

// getClientIP extracts the real client IP from the request.
//
// HXC-292 fixed two independent defects here, plus one review-caught defect
// (F1) in the fix for the second.
//
// Defect 1 (bracket-fork / identity split): the RemoteAddr fallback used a
// hand-rolled strings.LastIndex(r.RemoteAddr, ":") split. It correctly
// located the port separator, but for a direct IPv6 connection — where Go
// sets RemoteAddr to "[2001:db8::1]:54321" — it left the surrounding
// brackets in the returned host, yielding the identity "[2001:db8::1]",
// while the X-Forwarded-For / X-Real-IP paths returned the header value
// verbatim (conventionally UNbracketed). The SAME real caller therefore
// forked into two distinct rate-limit / audit identities (see
// RateLimitMiddleware's rl.Allow(clientIP) below and
// api/audit_logger.go's LogHTTPRequest ClientIP field) depending only on
// whether a forwarding header happened to be present on a given request.
// External precedent for the exact hazard of comparing a bracketed literal
// against an unbracketed one without reconciling them first:
// CVE-2026-39361 (OpenObserve, GHSA-gcwf-3p7h-wm79). Fixed by taking
// RemoteAddr apart with net.SplitHostPort — which always returns the host
// UNbracketed — never a hand-rolled split; see clientip.Resolve's internal
// normalizeRemoteAddr step (unexported — see that package's doc comment
// for why only Resolve itself is exported).
//
// Defect 2 (unconditional forwarding-header trust): X-Forwarded-For and
// X-Real-IP were honoured verbatim with NO permitted-intermediary list. Any
// caller able to reach this service — not merely one behind a legitimate
// reverse proxy — could state any client identity it liked. Fixed by
// gating BOTH headers on clientip.Resolve's internal peer-trust check,
// which trusts nothing unless the immediate TCP peer is explicitly
// allow-listed via
// clientIPTrustedProxiesEnvVar; see that function's doc comment (in
// clientip/clientip.go) for why the default is empty (deny) rather than a
// value guessed into source, and see this file's own topology-finding
// rationale below for why no single guessed default would even be SAFE for
// every deployment this repository sanctions.
//
// Topology finding (HXC-292, captured against this repository's own
// deployment configs): BOTH a direct-exposure topology and a
// reverse-proxied topology are sanctioned here. docker-compose.yml — the
// README "quick start" path — exposes this service with no proxy in
// front. docker-compose.prod.yml, llm-verifier/docker-compose.yml,
// k8s-manifests.yaml, and every guide under docs/deployment/ instead put
// nginx, Traefik, or an nginx-ingress controller in front of it. Which one
// is live for any given deployment is not knowable from this source tree
// — and even where a proxy IS present, its actual network identity varies
// by topology: a docker-compose bridge subnet is operator-chosen (e.g.
// docker-compose.prod.yml's own "subnet: 172.20.0.0/16"), a Kubernetes
// ingress controller's pod/service CIDR is cluster-specific, and an AWS
// ALB's source ranges are AWS-managed and change over time. CONST-045 (no
// hardcoded distribution hosts) and this project's general
// configuration-injection discipline both forbid baking any single one of
// those into source — and no guessed default would even be SAFE across
// all of them: every other container sharing docker-compose.prod.yml's
// bridge network (postgres, redis, prometheus, grafana, watchtower) can
// already reach this service's port directly, so a blanket
// "trust all of RFC 1918" default would let any ONE of those, if
// compromised, forge a client identity exactly as easily as the nginx
// container such a default would have been trying to describe. Operators
// deploying behind docker-compose.prod.yml, llm-verifier/docker-compose.yml,
// or k8s-manifests.yaml's ingress-nginx MUST set LLM_VERIFIER_TRUSTED_PROXIES
// to their actual proxy's address/CIDR to regain per-client granularity;
// the empty default's cost is a bounded, instantly-reversible degradation
// in identity GRANULARITY (every caller behind an unconfigured proxy
// resolves to the proxy's own address), never an outage and never a
// forged-identity hole. See HXC-293 for the framework-wide version of this
// same default-deny posture in the sibling helix_agent submodule; this
// item (HXC-292) is scoped to this function only and does not widen
// HXC-293's scope.
//
// F1 (review-caught, the more serious half): the FIRST cut of the defect-2
// fix selected the LEFTMOST X-Forwarded-For entry once the peer was
// trusted, on the theory that "every entry to its right was appended by an
// intermediary this function has already confirmed trusted." That theory
// was false: the peer-trust check confirms only the LAST hop —
// r.RemoteAddr, the immediate TCP peer — never any earlier hop recorded
// inside the header itself. With nginx's common
// `$proxy_add_x_forwarded_for` configuration, the trusted proxy APPENDS to
// whatever XFF it received rather than replacing it, so an attacker
// sending "X-Forwarded-For: 8.8.8.8" through that proxy causes this
// function to receive "8.8.8.8, <attacker's real IP>" — leftmost selection
// handed back the attacker's own forged claim, reopening defect 2 behind
// precisely the deployment topology this fix's default-deny posture was
// written to respect (configuring the allowlist, exactly as this
// function's own documentation instructs an operator to do, is what
// exposed the bypass). Fixed by clientip.Resolve's internal
// resolveForwardedFor step, which walks the list RIGHT to LEFT — the order
// proxies actually append in — skipping any entry that is ITSELF on the
// trusted allowlist, and returning the first entry that is not.
//
// # Defect-class census, not just this function (HXC-292 F2 follow-up)
//
// This function's OWN two consumers (RateLimitMiddleware below,
// api/audit_logger.go's LogHTTPRequest) were examined completely and
// correctly — but "how many consumers does THIS function have" is a
// different question from "how many functions in this component do THIS."
// A census of the latter found two further, independent instances of the
// same defect class elsewhere in this repository: HXC-299
// (security/security.go's extractIPAddress — which THEN split at the
// FIRST colon, so a bracketed IPv6 literal truncated to "[2001",
// collapsing genuinely different callers onto one shared identity, worse
// than this function's pre-fix behaviour, which at least kept identities
// distinct, and which also trusted its header verbatim) and HXC-298
// (enhanced/enterprise/api.go's EnterpriseAPI.getClientIP — verbatim
// header trust plus a raw RemoteAddr, port included, feeding the RBAC
// audit log). Both are cited by SYMBOL rather than by file:line, because
// a line citation here drifts: this comment previously named
// security.go:482, which sat 57 lines above extractIPAddress at HEAD and
// 67 above it in the working tree — and line 482 is itself only a comment
// fragment. Both defects are since FIXED — extractIPAddress, this
// function, and EnterpriseAPI.getClientIP now all resolve through
// clientip.Resolve. Per §11.4.146 STEP 3 / §11.4.118: fixing a function's
// identified callers is not the same as censusing the defect CLASS across
// the component — the second question is the one that surfaces siblings
// like these.
//
// # HXC-298 extraction (closes HXC-308)
//
// HXC-299's own fix could not import this (then-unexported) function
// without inverting this codebase's dependency direction, so it mirrored
// the corrected algorithm instead — leaving two independently-maintained
// copies with nothing to stop them drifting apart (HXC-308). Fixing
// HXC-298's third, independent copy in enhanced/enterprise/api.go by
// re-deriving yet another mirror would only have widened that risk.
// Instead, the corrected algorithm was extracted into the new clientip
// package (digital.vasic.llmsverifier/clientip) — a dependency-neutral
// leaf package with no import-cycle risk against api/, security/, or
// enhanced/enterprise/ (none of the three import each other). This
// function, security/client_ip_trust.go's resolveClientIP, and
// enhanced/enterprise/api.go's EnterpriseAPI.getClientIP now all delegate
// to clientip.Resolve: there is exactly one implementation of this
// algorithm in this codebase. See clientip/clientip.go's package doc
// comment for the full extraction rationale.
func getClientIP(r *http.Request) string {
	return clientip.Resolve(r)
}

// isRateLimited checks if the client is rate limited using the global rate limiter
func isRateLimited(clientIP string) bool {
	rl := GetGlobalRateLimiter()
	allowed, _, _ := rl.Allow(clientIP)
	return !allowed
}
