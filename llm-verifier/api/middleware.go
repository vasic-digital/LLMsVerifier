package api

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
// X-Forwarded-For / X-Real-IP headers trusted by getClientIP. See
// isTrustedProxyPeer for the full rationale behind defaulting this to
// empty (nothing trusted) rather than a guessed value.
const clientIPTrustedProxiesEnvVar = "LLM_VERIFIER_TRUSTED_PROXIES"

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
// CVE-2026-39361 (OpenObserve, GHSA-gcwf-3p7h-wm79). Fixed below by taking
// RemoteAddr apart with net.SplitHostPort — which always returns the host
// UNbracketed — never a hand-rolled split; see normalizeRemoteAddr.
//
// Defect 2 (unconditional forwarding-header trust): X-Forwarded-For and
// X-Real-IP were honoured verbatim with NO permitted-intermediary list. Any
// caller able to reach this service — not merely one behind a legitimate
// reverse proxy — could state any client identity it liked. Fixed below by
// gating BOTH headers on isTrustedProxyPeer, which trusts nothing unless
// the immediate TCP peer is explicitly allow-listed via
// clientIPTrustedProxiesEnvVar; see that function's doc comment for why the
// default is empty (deny) rather than a value guessed into source.
//
// F1 (review-caught, the more serious half): the FIRST cut of the defect-2
// fix selected the LEFTMOST X-Forwarded-For entry once the peer was
// trusted, on the theory that "every entry to its right was appended by an
// intermediary this function has already confirmed trusted." That theory
// was false: isTrustedProxyPeer confirms only the LAST hop — r.RemoteAddr,
// the immediate TCP peer — never any earlier hop recorded inside the
// header itself. With nginx's common `$proxy_add_x_forwarded_for`
// configuration, the trusted proxy APPENDS to whatever XFF it received
// rather than replacing it, so an attacker sending
// "X-Forwarded-For: 8.8.8.8" through that proxy causes this function to
// receive "8.8.8.8, <attacker's real IP>" — leftmost selection handed back
// the attacker's own forged claim, reopening defect 2 behind precisely the
// deployment topology this fix's default-deny posture was written to
// respect (configuring the allowlist, exactly as this function's own
// documentation instructs an operator to do, is what exposed the bypass).
// Fixed below by resolveForwardedFor, which walks the list RIGHT to LEFT —
// the order proxies actually append in — skipping any entry that is
// ITSELF on the trusted allowlist, and returning the first entry that is
// not: the address closest to the client this function cannot itself
// verify came from a hop it trusts. This is safe under both an appending
// proxy (the forged leftmost entry is walked past once its real, appended,
// untrusted origin is reached) and a replacing proxy (there is only ever
// one entry, and it is not trusted, so it is returned immediately); worst
// case — an unparseable or all-trusted chain — it degrades to RemoteAddr,
// never forges. Every candidate is validated with net.ParseIP so an
// attacker-chosen non-IP string can never land in the audit-log ClientIP
// field.
//
// # Defect-class census, not just this function (HXC-292 F2 follow-up)
//
// This function's OWN two consumers (RateLimitMiddleware below,
// api/audit_logger.go's LogHTTPRequest) were examined completely and
// correctly — but "how many consumers does THIS function have" is a
// different question from "how many functions in this component do THIS."
// A census of the latter found two further, independent instances of the
// same defect class elsewhere in this repository, tracked separately and
// deliberately NOT touched by this fix (scope discipline, HXC-293 note
// below): HXC-299 (security/security.go:482 extractIPAddress — splits at
// the FIRST colon, so a bracketed IPv6 literal truncates to "[2001",
// collapsing genuinely different callers onto one shared identity, worse
// than this function's pre-fix behaviour, which at least kept identities
// distinct; also trusts its header verbatim) and HXC-298
// (enhanced/enterprise/api.go:817 — verbatim header trust plus a raw
// RemoteAddr, port included, feeding the RBAC audit log). Per §11.4.146
// STEP 3 / §11.4.118: fixing a function's identified callers is not the
// same as censusing the defect CLASS across the component — the second
// question is the one that surfaces siblings like these.
func getClientIP(r *http.Request) string {
	if isTrustedProxyPeer(r.RemoteAddr) {
		// Check X-Forwarded-For header first (for proxies/load balancers).
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if ip := resolveForwardedFor(xff); ip != "" {
				return ip
			}
			// No usable entry: either every entry was itself trusted (a
			// fully-trusted chain with no client beyond it) or the walk hit
			// an unparseable entry before finding a real one. Either way,
			// fall through rather than return "" or an unvalidated string.
		}

		// Check X-Real-IP header. Conventionally set (not appended) by the
		// immediate proxy — e.g. nginx's `proxy_set_header X-Real-IP
		// $remote_addr` overwrites rather than accumulates — so it carries
		// no list/appending semantics and no F1-class risk; it is still
		// validated as a real IP for the same reason XFF entries are: an
		// attacker-chosen non-IP string must never reach the audit log.
		if xri := normalizeHostLiteral(r.Header.Get("X-Real-IP")); xri != "" {
			if net.ParseIP(xri) != nil {
				return xri
			}
		}
	}

	// Fall back to RemoteAddr — always net.SplitHostPort, never a manual split.
	return normalizeRemoteAddr(r.RemoteAddr)
}

// resolveForwardedFor picks the caller's identity out of a trusted
// X-Forwarded-For header value.
//
// It walks the comma-separated entry list from RIGHT to LEFT — the order
// each hop actually appends in (see getClientIP's F1 doc section) —
// skipping any entry that is ITSELF on the trusted-proxy allowlist, and
// returns the first entry that is not: the closest-to-the-client address
// this function cannot itself verify came from another hop it trusts. This
// mirrors the standard approach mature reverse-proxy stacks use once they
// bother validating XFF trust at all (nginx's realip module with
// `real_ip_recursive on`; Express's `trust proxy` with a numeric hop count
// or CIDR list) — walk backward through the chain, trust only as far as
// the configured proxies go, and stop at the first untrusted hop.
//
// Every candidate is validated with net.ParseIP before being trusted OR
// skipped: an entry that does not parse as an IP at all is neither
// something this function can confirm is a trusted intermediary NOR
// something safe to hand out as a caller identity (an attacker-chosen
// non-IP string must never reach the audit log or the rate-limit key), so
// the walk stops there and getClientIP falls through to X-Real-IP /
// RemoteAddr rather than guessing.
func resolveForwardedFor(xff string) string {
	entries := strings.Split(xff, ",")
	for i := len(entries) - 1; i >= 0; i-- {
		candidate := normalizeHostLiteral(entries[i])
		ip := net.ParseIP(candidate)
		if ip == nil {
			// Unparseable (or empty) entry: cannot be confirmed as a
			// trusted hop, and cannot be handed out as an identity either.
			// Stop here rather than skip past it or return it.
			return ""
		}
		if isTrustedProxyIP(ip) {
			// This hop is itself a declared, trusted intermediary —
			// presumed to have appended its own observed peer to its
			// right (already walked, or, if this was the rightmost entry,
			// IS the confirmed TCP peer per isTrustedProxyPeer). Keep
			// walking left through the chain.
			continue
		}
		return candidate
	}
	return ""
}

// isTrustedProxyPeer decides whether X-Forwarded-For / X-Real-IP may be
// honoured for a request whose immediate TCP peer is remoteAddr.
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
// container such a default would have been trying to describe.
//
// The allowlist is therefore entirely operator-supplied via
// clientIPTrustedProxiesEnvVar (LLM_VERIFIER_TRUSTED_PROXIES — a
// comma-separated list of bare IPs and/or CIDRs, e.g. "172.20.0.0/16" for
// the docker-compose.prod.yml topology, or the ingress-controller's pod
// CIDR on Kubernetes) and DEFAULTS TO EMPTY: nothing is trusted,
// X-Forwarded-For / X-Real-IP are ignored, and every caller's identity is
// its own RemoteAddr. This mirrors where the ecosystem has moved after
// repeated XFF-spoofing incidents (Go's own chi middleware's RealIP no
// longer trusts XFF unless a trusted-proxy list is configured; Express's
// `trust proxy` defaults to false; Django / Rails require explicit
// configuration before SECURE_PROXY_SSL_HEADER / trusted_proxies take
// effect) and matches OWASP's guidance: do not use X-Forwarded-For for
// security-relevant decisions unless you control and know the proxies in
// the chain.
//
// What the empty default costs a genuinely-proxied deployment that has
// not yet set the env var: every caller behind that proxy resolves to the
// PROXY's own peer address rather than its own — i.e. exactly the
// behaviour this service would already have with no proxy in front of it
// at all. Rate limiting (RateLimitMiddleware, below) becomes shared
// across every user of that proxy, and audit-log ClientIP values
// (api/audit_logger.go) collapse to the proxy's address, until the
// operator configures the allowlist — but the service keeps answering
// every request either way. That is a deliberate, bounded, and instantly
// reversible degradation in identity GRANULARITY, never an outage, and
// never a forged-identity hole. It is traded against the alternative of
// guessing one specific network's identity into source code, which
// CONST-045 forbids outright and which would silently be WRONG — and
// therefore either still-forgeable (an over-broad guess like "all of RFC
// 1918") or entirely inert (an under-broad guess that doesn't match the
// operator's actual topology) — for every deployment except the one
// guessed.
//
// Operators deploying behind docker-compose.prod.yml,
// llm-verifier/docker-compose.yml, or k8s-manifests.yaml's ingress-nginx
// MUST set LLM_VERIFIER_TRUSTED_PROXIES to their actual proxy's
// address/CIDR to regain per-client granularity. See HXC-293 for the
// framework-wide version of this same default-deny posture in the
// sibling helix_agent submodule; this item (HXC-292) is scoped to this
// function only and does not widen HXC-293's scope.
func isTrustedProxyPeer(remoteAddr string) bool {
	peerHost, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		peerHost = stripBrackets(remoteAddr)
	}
	peerIP := net.ParseIP(peerHost)
	if peerIP == nil {
		return false
	}
	return isTrustedProxyIP(peerIP)
}

// isTrustedProxyIP decides whether ip itself is on the operator-configured
// trusted-proxy allowlist (clientIPTrustedProxiesEnvVar). Used both for the
// immediate TCP peer (isTrustedProxyPeer, above) and, in
// resolveForwardedFor, for each earlier hop recorded inside a trusted
// X-Forwarded-For chain — an address this function has no independent way
// to verify actually sits where the header claims, but which the operator
// has explicitly declared trustworthy by address.
func isTrustedProxyIP(ip net.IP) bool {
	configured := strings.TrimSpace(os.Getenv(clientIPTrustedProxiesEnvVar))
	if configured == "" {
		return false
	}
	warnOnMalformedTrustedProxiesEntries(configured)

	for _, entry := range strings.Split(configured, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			// F5: strip a redundant bracket pair around a bare IPv6
			// allowlist entry ("[2001:db8::1]") before parsing —
			// net.ParseIP rejects brackets outright, so an unstripped
			// bracketed entry would silently never match its peer,
			// regardless of how the operator wrote it.
			if candidate := net.ParseIP(stripBrackets(entry)); candidate != nil && candidate.Equal(ip) {
				return true
			}
			continue
		}
		if _, cidr, err := net.ParseCIDR(entry); err == nil && cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// clientIPLastWarnedTrustedProxies de-duplicates the malformed-allowlist
// diagnostic (below): it remembers the exact clientIPTrustedProxiesEnvVar
// value most recently warned about, so a fixed (mis)configuration produces
// AT MOST one warning per distinct value rather than once per request —
// isTrustedProxyIP runs on every request that reaches a trust decision.
var clientIPLastWarnedTrustedProxies atomic.Value // holds string

// warnOnMalformedTrustedProxiesEntries logs a single, de-duplicated
// diagnostic warning when clientIPTrustedProxiesEnvVar contains an entry
// that is neither a valid bare IP nor a valid CIDR. This is deliberately
// non-fatal and changes no trust decision whatsoever — a malformed entry
// is, and remains, silently skipped by isTrustedProxyIP above (fails
// closed for that one entry) — it exists solely so a misconfigured
// allowlist is diagnosable from logs rather than only from behaviour that
// looks identical to "not configured yet."
func warnOnMalformedTrustedProxiesEntries(configured string) {
	if last, ok := clientIPLastWarnedTrustedProxies.Load().(string); ok && last == configured {
		return
	}
	clientIPLastWarnedTrustedProxies.Store(configured)

	var malformed []string
	for _, entry := range strings.Split(configured, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				malformed = append(malformed, entry)
			}
			continue
		}
		if net.ParseIP(stripBrackets(entry)) == nil {
			malformed = append(malformed, entry)
		}
	}
	if len(malformed) > 0 {
		log.Printf("llm-verifier: %s contains %d entr(y/ies) that are neither a valid IP nor a valid "+
			"CIDR and will be ignored (fails closed for those entries): %v — check your deployment "+
			"configuration", clientIPTrustedProxiesEnvVar, len(malformed), malformed)
	}
}

// stripBrackets removes a single matching pair of square brackets wrapping
// an otherwise-bare IPv6 literal (e.g. "[2001:db8::1]" -> "2001:db8::1").
// It is a no-op for anything else. This is the one shape
// net.SplitHostPort itself does not cover — a bracketed host with no port
// at all — so it is applied explicitly wherever that degenerate form can
// occur, preventing it from silently forking into a second, distinct
// identity from its "[2001:db8::1]:<port>" sibling.
func stripBrackets(host string) string {
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		return host[1 : len(host)-1]
	}
	return host
}

// normalizeRemoteAddr extracts the caller's address from
// http.Request.RemoteAddr, which Go always sets to "host:port" (bracketing
// an IPv6 host per net.SplitHostPort's own contract). net.SplitHostPort —
// never a hand-rolled split — is the only correct way to take it apart: it
// always returns the host UNbracketed, which is the fix for HXC-292 defect
// 1 (see getClientIP's doc comment for the full defect description).
func normalizeRemoteAddr(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		if host == "" {
			// A port was present but the host portion was empty (e.g.
			// ":12345"). There is no caller identity to extract here at all;
			// the original string is returned unmodified rather than an
			// invented sentinel, matching this function's pre-fix behaviour
			// for this exact degenerate shape.
			return remoteAddr
		}
		return host
	}
	// No ":port" suffix to split off. This is not automatically malformed —
	// a bare host with no port needs nothing stripped — so it is used AS-IS,
	// except for the one degenerate shape that would otherwise silently fork
	// into a second identity: a redundant bracket pair around an otherwise-
	// bare host ("[2001:db8::1]" with no port at all).
	return stripBrackets(remoteAddr)
}

// normalizeHostLiteral trims and bracket-strips a caller-address literal
// taken from a forwarding header (an X-Forwarded-For entry or
// X-Real-IP). Unlike RemoteAddr, a header-supplied literal is not
// guaranteed to carry a port, so net.SplitHostPort is not the right tool
// here — an unbracketed IPv6 literal such as "2001:db8::1" contains
// colons that SplitHostPort would misinterpret as a host:port separator.
// The defensive bracket-strip alone is sufficient: it normalizes a
// bracketed header literal onto the SAME identity a direct connection from
// the same address would resolve to (see normalizeRemoteAddr) without
// risking that misparse.
func normalizeHostLiteral(literal string) string {
	return stripBrackets(strings.TrimSpace(literal))
}

// isRateLimited checks if the client is rate limited using the global rate limiter
func isRateLimited(clientIP string) bool {
	rl := GetGlobalRateLimiter()
	allowed, _, _ := rl.Allow(clientIP)
	return !allowed
}
