package security

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

// HXC-299: caller-identity trust resolution for extractIPAddress
// (security.go). This file is a deliberate MIRROR — not a fresh
// reimplementation — of the corrected algorithm HXC-292 landed for the
// sibling function api/middleware.go's getClientIP. See that function's doc
// comment (HXC-292, commit 9b9c4da9) for the full defect history, including
// the review-caught F1 finding (leftmost X-Forwarded-For selection is wrong
// under an appending proxy) this mirror inherits already-fixed rather than
// re-risking.
//
// # Why mirror instead of import/reuse
//
// getClientIP / resolveForwardedFor / isTrustedProxyPeer / isTrustedProxyIP
// in api/middleware.go are UNEXPORTED (lowercase) — this package cannot
// import them without api/middleware.go being modified to export them,
// which is out of scope for this fix (api/ is untouched here except to
// read). Even were they exported, importing FROM this lower-level
// security package INTO the higher-level api package would invert this
// codebase's normal dependency direction (api depends on security-package
// primitives for audit/rate-limit/credential concerns, never the reverse) —
// api/model_verifier.go, api/handlers.go, and api/server.go all import
// database/config/etc. downward, and no existing file in security/ imports
// api/, so adding that edge here would be the first of its kind. Both
// reasons independently rule out direct reuse; this file mirrors the
// CORRECTED algorithm instead of attempting a fresh one.
//
// # Trust semantics are IDENTICAL by design, not merely similar
//
// This mirror uses the SAME environment variable name
// (securityClientIPTrustedProxiesEnvVar == api's clientIPTrustedProxiesEnvVar
// == "LLM_VERIFIER_TRUSTED_PROXIES"), the SAME default-deny posture (empty
// configuration trusts nothing), and the SAME right-to-left
// skip-trusted-then-take-first-untrusted XFF walk. A single operator-set
// allowlist therefore governs BOTH this function's caller-identity
// resolution and api/middleware.go's — there is no deliberate divergence in
// trust semantics between the two copies, only the unavoidable duplication
// of the algorithm itself across the package/visibility boundary described
// above.
const securityClientIPTrustedProxiesEnvVar = "LLM_VERIFIER_TRUSTED_PROXIES"

// resolveClientIP extracts the real caller identity from r, honouring
// X-Forwarded-For / X-Real-IP ONLY when the immediate TCP peer
// (r.RemoteAddr) is on the operator-configured trusted-proxy allowlist
// (securityClientIPTrustedProxiesEnvVar). This is extractIPAddress's
// (security.go) sole implementation — see that function for the one call
// site (AuditTrail.LogRequest's AuditEntry.IPAddress) and its doc comment
// for a census of extractIPAddress's own callers. Structurally identical to
// api/middleware.go's getClientIP (mirrored, not reused — see this file's
// top-of-file doc comment for why).
func resolveClientIP(r *http.Request) string {
	if isTrustedProxyPeer(r.RemoteAddr) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if ip := resolveForwardedFor(xff); ip != "" {
				return ip
			}
			// No usable entry: either every entry was itself trusted, or the
			// walk hit an unparseable entry before finding a real one.
			// Either way, fall through rather than return "" or an
			// unvalidated string.
		}

		if xri := normalizeHostLiteral(r.Header.Get("X-Real-IP")); xri != "" {
			if net.ParseIP(xri) != nil {
				return xri
			}
		}
	}

	// Fall back to RemoteAddr — always net.SplitHostPort, never a manual
	// split (the fix for defect 1: strings.Split(remoteAddr, ":")[0] split
	// on EVERY colon, truncating a bracketed IPv6 literal to its opening
	// bracket + first hextet and collapsing distinct callers that share
	// only that hextet onto one shared identity).
	return normalizeRemoteAddr(r.RemoteAddr)
}

// resolveForwardedFor picks the caller's identity out of a trusted
// X-Forwarded-For header value. It walks the comma-separated entry list
// from RIGHT to LEFT — the order each hop actually appends in under a
// standard appending reverse proxy (e.g. nginx's
// $proxy_add_x_forwarded_for) — skipping any entry that is ITSELF on the
// trusted allowlist, and returns the first entry that is not. This is the
// HXC-292 F1 fix, mirrored: extractIPAddress's pre-fix behaviour took the
// LEFTMOST entry unconditionally (strings.Split(xff, ",")[0]), which is the
// exact bypass an attacker-prepended forged entry exploits behind any
// appending proxy — mirrored here already-corrected rather than
// reintroducing that exact mistake in a fresh implementation.
//
// Every candidate is validated with net.ParseIP before being trusted or
// skipped — an entry that does not parse as an IP cannot be confirmed as a
// trusted hop nor safely handed out as an identity (it must never reach the
// audit log or a rate-limit key), so the walk stops there.
func resolveForwardedFor(xff string) string {
	entries := strings.Split(xff, ",")
	for i := len(entries) - 1; i >= 0; i-- {
		candidate := normalizeHostLiteral(entries[i])
		ip := net.ParseIP(candidate)
		if ip == nil {
			return ""
		}
		if isTrustedProxyIP(ip) {
			continue
		}
		return candidate
	}
	return ""
}

// isTrustedProxyPeer decides whether X-Forwarded-For / X-Real-IP may be
// honoured for a request whose immediate TCP peer is remoteAddr.
//
// The allowlist is entirely operator-supplied via
// securityClientIPTrustedProxiesEnvVar and DEFAULTS TO EMPTY: nothing is
// trusted, both headers are ignored, and every caller's identity is its own
// RemoteAddr. See api/middleware.go's isTrustedProxyPeer doc comment
// (HXC-292) for the full topology-finding rationale (this repository
// sanctions both a direct-exposure and a reverse-proxied deployment, no
// single guessed default is safe across both, CONST-045 forbids hardcoding
// a distribution host regardless) — that rationale applies identically here
// and is not repeated per call site.
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
// trusted-proxy allowlist. Used both for the immediate TCP peer
// (isTrustedProxyPeer, above) and, in resolveForwardedFor, for each earlier
// hop recorded inside a trusted X-Forwarded-For chain.
func isTrustedProxyIP(ip net.IP) bool {
	configured := strings.TrimSpace(os.Getenv(securityClientIPTrustedProxiesEnvVar))
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

// securityClientIPLastWarnedTrustedProxies de-duplicates the
// malformed-allowlist diagnostic below: it remembers the exact
// securityClientIPTrustedProxiesEnvVar value most recently warned about, so
// a fixed (mis)configuration produces AT MOST one warning per distinct
// value rather than once per request.
var securityClientIPLastWarnedTrustedProxies atomic.Value // holds string

// warnOnMalformedTrustedProxiesEntries logs a single, de-duplicated
// diagnostic warning when securityClientIPTrustedProxiesEnvVar contains an
// entry that is neither a valid bare IP nor a valid CIDR. Deliberately
// non-fatal and changes no trust decision — a malformed entry is, and
// remains, silently skipped by isTrustedProxyIP above (fails closed for
// that one entry).
func warnOnMalformedTrustedProxiesEntries(configured string) {
	if last, ok := securityClientIPLastWarnedTrustedProxies.Load().(string); ok && last == configured {
		return
	}
	securityClientIPLastWarnedTrustedProxies.Store(configured)

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
		log.Printf("llm-verifier/security: %s contains %d entr(y/ies) that are neither a valid IP nor a "+
			"valid CIDR and will be ignored (fails closed for those entries): %v — check your "+
			"deployment configuration", securityClientIPTrustedProxiesEnvVar, len(malformed), malformed)
	}
}

// stripBrackets removes a single matching pair of square brackets wrapping
// an otherwise-bare IPv6 literal (e.g. "[2001:db8::1]" -> "2001:db8::1").
// It is a no-op for anything else.
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
// always returns the host UNbracketed. This is the fix for defect 1: the
// pre-fix strings.Split(remoteAddr, ":")[0] split on EVERY colon in the
// string, not just the port separator, truncating "[2001:db8::1]:54321" to
// "[2001" — an opening bracket and the first hextet — and collapsing any
// two callers that happen to share that first hextet onto one shared,
// non-address identity.
func normalizeRemoteAddr(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		if host == "" {
			// A port was present but the host portion was empty (e.g.
			// ":12345"). There is no caller identity to extract here at
			// all; the original string is returned unmodified rather than
			// an invented sentinel, matching this function's pre-fix
			// behaviour for this exact degenerate shape.
			return remoteAddr
		}
		return host
	}
	// No ":port" suffix to split off. This is not automatically malformed
	// — a bare host with no port needs nothing stripped — so it is used
	// AS-IS, except for the one degenerate shape that would otherwise
	// silently fork into a second identity: a redundant bracket pair
	// around an otherwise-bare host ("[2001:db8::1]" with no port at all).
	return stripBrackets(remoteAddr)
}

// normalizeHostLiteral trims and bracket-strips a caller-address literal
// taken from a forwarding header (an X-Forwarded-For entry or X-Real-IP).
// Unlike RemoteAddr, a header-supplied literal is not guaranteed to carry a
// port, so net.SplitHostPort is not the right tool here — an unbracketed
// IPv6 literal such as "2001:db8::1" contains colons that SplitHostPort
// would misinterpret as a host:port separator. The defensive bracket-strip
// alone is sufficient: it normalizes a bracketed header literal onto the
// SAME identity a direct connection from the same address would resolve to
// (see normalizeRemoteAddr) without risking that misparse.
func normalizeHostLiteral(literal string) string {
	return stripBrackets(strings.TrimSpace(literal))
}
