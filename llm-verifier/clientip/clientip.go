// Package clientip is the single canonical implementation of this
// codebase's caller-identity resolution algorithm: derive the real HTTP
// caller's address from an *http.Request, honouring X-Forwarded-For /
// X-Real-IP ONLY when the immediate TCP peer is on an operator-configured
// trusted-proxy allowlist, and deriving the RemoteAddr fallback with
// net.SplitHostPort — never a hand-rolled colon-split.
//
// # Why this package exists
//
// This algorithm existed as THREE independently-maintained copies, with
// nothing to stop them drifting apart. TWO were fixed independently and
// left keyed off the same LLM_VERIFIER_TRUSTED_PROXIES operator setting,
// free to drift from each other. The THIRD —
// enhanced/enterprise/api.go's EnterpriseAPI.getClientIP, feeding the
// enterprise RBAC audit log — consulted NO allowlist at all: it returned
// the LEFTMOST X-Forwarded-For entry from any caller whatsoever, then
// X-Real-IP, then the raw RemoteAddr complete with its ephemeral port. It
// was not a drift-prone copy of a gated rule; it was ungated, and any
// caller could dictate its own enterprise-audit identity. All three now
// delegate to Resolve below, so there is exactly one implementation of
// this algorithm, enforced structurally rather than by a cross-check that
// could itself fall out of sync.
//
// # Call sites
//
// Measured with `go list -deps ./cmd/...` (499 packages): this package
// and api ARE in the shipped binary's build graph, reached via
// cmd -> api -> clientip; security and enhanced/enterprise are NOT, and
// would each need cmd to gain a new import before they could run there.
//
// Being in the build graph is not the same as being live: api's call
// sites sit behind RateLimitMiddleware and AuditMiddleware, and
// Server.routes() registers against a bare *http.ServeMux with no
// middleware chain, so neither is applied today. Enumerate the current
// set with `go list -deps` plus a grep for the delegating functions
// rather than trusting a list here — a hardcoded census rots on the next
// edit, which is also why TestTrustedProxiesEnvVarLiteralValue
// deliberately refuses to assert a reference COUNT. Treat changes with
// production care anyway: activating an api caller requires ONLY wiring
// its middleware into Server.routes() — a routing change, not new
// caller-identity logic — and a regression here would then reach api's
// rate-limit key and its audit log with no further warning, because this
// package is already shipping.
//
// # Trust semantics
//
// The allowlist is entirely operator-supplied via TrustedProxiesEnvVar
// (LLM_VERIFIER_TRUSTED_PROXIES — a comma-separated list of bare IPs
// and/or CIDRs) and DEFAULTS TO EMPTY: nothing is trusted, both headers
// are ignored, and every caller's identity is its own RemoteAddr. See the
// original HXC-292 fix's doc comment (preserved in api/middleware.go) for
// the topology-finding rationale behind that default-deny posture, and
// resolveForwardedFor for why a trusted X-Forwarded-For is walked right to
// left rather than leftmost.
//
// # One dedupe slot, one log prefix
//
// Before this extraction api and security each deduplicated the
// malformed-allowlist warning independently, under their own prefixes
// ("llm-verifier: ..." and "llm-verifier/security: ..."). This package has
// ONE dedupe variable (lastWarnedTrustedProxies, below) and ONE prefix
// ("llm-verifier/clientip: ..."), used by every delegating caller. Two
// disclosed consequences. (1) A misconfigured value now warns at most once
// per CONSECUTIVE RUN of that value process-wide — per consecutive run,
// NOT per distinct value. Measured, A malformed and B clean: A→1 warning,
// A again→0, B→0, A again→1, i.e. TWO warnings for the one distinct value
// A. See TestWarnOnMalformedTrustedProxiesEntriesDedupesPerConsecutiveRun.
// (2) The old per-package prefixes no longer exist. No non-test code in
// this repository asserts on either prefix's exact text, so there is no
// known internal consumer — but it IS externally observable to log tooling
// that greps for them.
package clientip

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

// TrustedProxiesEnvVar is the environment variable operators use to
// declare which immediate peers are permitted to have their
// X-Forwarded-For / X-Real-IP headers trusted by Resolve. Every caller of
// this package (api/middleware.go, security/client_ip_trust.go,
// enhanced/enterprise/api.go) reads the SAME variable, so a single
// operator-set allowlist governs caller-identity resolution everywhere in
// this codebase. Defaults to empty (nothing trusted) — see the package
// doc comment for the full rationale.
const TrustedProxiesEnvVar = "LLM_VERIFIER_TRUSTED_PROXIES"

// Resolve extracts the real caller identity from r, honouring
// X-Forwarded-For / X-Real-IP ONLY when the immediate TCP peer
// (r.RemoteAddr) is on the operator-configured trusted-proxy allowlist
// (TrustedProxiesEnvVar). This is the sole implementation callers of this
// package delegate to.
//
// # Exported surface
//
// Resolve and TrustedProxiesEnvVar are this package's ENTIRE public API;
// every other function declared here is unexported (the exhaustive list is
// `grep '^func '` in this file, so it cannot silently drift out of date),
// and every delegating caller calls Resolve alone. Keep it that way:
// resolveForwardedFor and isTrustedProxyIP are precisely the primitives
// that walk the trusted-XFF chain WITHOUT the peer-trust gate Resolve
// applies before calling them, so exporting them would make this package's
// own defect class — an XFF / X-Real-IP value trusted without gating —
// reachable in a single external call.
func Resolve(r *http.Request) string {
	if isTrustedProxyPeer(r.RemoteAddr) {
		// An explicitly-EMPTY X-Forwarded-For joins to "" and is therefore
		// treated as ABSENT, falling through to X-Real-IP — the
		// xff_header_present_but_empty_value_treated_as_absent semantics.
		xff := strings.Join(r.Header.Values("X-Forwarded-For"), ", ")
		xriValues := r.Header.Values("X-Real-IP")

		// # Header PRECEDENCE, and its threat model
		//
		// When a trusted peer sends BOTH headers, X-Forwarded-For is
		// consulted FIRST and X-Real-IP is the fallback: strict precedence,
		// deliberately NOT a cross-header equality check.
		//
		// Precedence is safe under any APPENDING proxy. nginx documents
		// `$proxy_add_x_forwarded_for` as the client's X-Forwarded-For "with
		// the $remote_addr variable appended to it, separated by a comma",
		// so a caller-forged chain always sits to the LEFT of the
		// proxy-appended truth and the right-to-left walk reaches the
		// proxy-written entry first. Forgery does not win. That is the
		// prevailing convention rather than an obligation: X-Forwarded-For
		// is, in MDN's words, "not part of any current specification".
		//
		// Exactly one topology is NOT covered: a proxy that SETS X-Real-IP
		// but RELAYS an unmanaged X-Forwarded-For verbatim. There the whole
		// chain is caller-written, the walk returns the caller's rightmost
		// entry, and X-Real-IP — which was authoritative — is never reached.
		// That is a MISCONFIGURATION of the trust boundary, not a defect in
		// the walk: allowlisting a proxy asserts it SANITISES forwarding
		// headers, and one relaying caller-controlled X-Forwarded-For
		// untouched does not. Operators MUST ensure an allowlisted proxy
		// either appends to X-Forwarded-For or strips it.
		//
		// ## Do not add a corroboration guard
		//
		// Refusing BOTH headers when the rightmost X-Forwarded-For entry and
		// a single X-Real-IP disagree looks attractive, on the premise that a
		// proxy managing both derives both from its own peer. The premise is
		// false: nginx's realip module "is used to change the client address
		// ... to those sent in the specified header field" and adds
		// `$realip_remote_addr`, which "keeps the original client address",
		// so a deployment feeding the two headers from those two sources
		// makes them disagree BY DESIGN whenever an upstream hop contributed
		// an entry — Kubernetes ingress-nginx being exactly that deployment
		// UNDER its compute-full-forwarded-for option, which gates the
		// divergence jointly with use-forwarded-headers (both default false,
		// and in the else branch both headers render $remote_addr and AGREE;
		// verified in rootfs/etc/nginx/template/nginx.tmpl).
		// That also refutes the CONTRAPOSITIVE such a guard leans on: there
		// the peer manages BOTH headers and they still disagree, so a
		// mismatch does not prove the peer managed only one. Independently,
		// X-Real-IP is not a standard header and many proxies never manage
		// it: measured live 2026-08-13, ZERO occurrences under BOTH matchers
		// in HAProxy's configuration manual (`dev` tip and the current 3.4
		// LTS) and ZERO in RFC 7239, positive control X-Forwarded-For
		// non-zero in each. Pinned by
		// "xri_from_earlier_hop_with_later_appended_xff".
		//
		// Re-checkers beware one trap in that measurement: X-Forwarded-For's
		// OWN case-insensitive count steps 26 -> 27 at the 2.6 series and
		// holds through `dev` while its case-sensitive count stays flat at
		// 22, which reads like an X-Real-IP introduction at 2.6. It is not —
		// the added match is a lowercase `x-forwarded-for` argument in a
		// `balance hash` example.
		//
		// Such a guard is also SYMMETRIC by construction — nothing
		// distinguishes a forged X-Forwarded-For from a forged X-Real-IP, so
		// it fires on both — and where the walk has ALREADY produced the
		// right answer it discards it, handing any caller behind an appending
		// proxy a NEW capability: force itself, and everyone sharing that
		// proxy, into one unattributable audit identity and one rate-limit
		// bucket. Pinned by "xri_forged_vs_xff_appended_trusted_peer".

		// Check X-Forwarded-For first (for proxies/load balancers), read
		// with Header.Values and never Header.Get.
		//
		// Go's HTTP server preserves EVERY instance of a repeated header as
		// a separate element; Header.Get returns ONLY THE FIRST. Reading
		// X-Forwarded-For with Get therefore re-opened the exact
		// leftmost-selection bypass the right-to-left walk exists to close,
		// because the walk only ever saw the first line: it failed OPEN,
		// returning the attacker's forged first instance. Reproduced over a
		// real loopback TCP connection through Go's own HTTP parser — not a
		// hand-built http.Request — by
		// TestResolveMultiInstanceForwardingHeadersOverLoopback.
		//
		// Joining the instances with ", " is RFC 7230 §3.2.2's recipient
		// rule, quoted WHOLE because the permission carries exactly one
		// condition and it is load-bearing: a recipient "MAY combine
		// multiple header fields with the same field name into one
		// "field-name: field-value" pair, without changing the semantics of
		// the message, by appending each subsequent field value to the
		// combined field value in order, separated by a comma." Order
		// preservation is not an inference from that permission — the same
		// section states it normatively: "a proxy MUST NOT change the order
		// of these field values when forwarding a message."
		//
		// The list-form precondition in that section binds the SENDER, not
		// the recipient: "A sender MUST NOT generate multiple header fields
		// with the same field name in a message unless either the entire
		// field value for that header field is defined as a comma-separated
		// list [i.e., #(values)] or the header field is a well-known
		// exception". X-Forwarded-For is specified by no RFC, so no sender
		// is bound by that precondition and a repeated instance is a shape
		// this code must HANDLE rather than reject — while the combination
		// itself stays within the recipient permission, since under the
		// header's comma-list convention appending changes no semantics.
		// Both wire forms therefore enter the identical walk and
		// resolve to the identical identity. Pinned by
		// "xff_two_instances_rightmost_untrusted_wins" and
		// "xff_two_instances_non_ip_first_still_resolves" in every
		// delegating package's census test, plus
		// TestResolveMultiInstanceForwardingHeadersOverLoopback here.
		if xff != "" {
			if ip := resolveForwardedFor(xff); ip != "" {
				return ip
			}
			// No usable entry: either every entry was itself trusted (a
			// fully-trusted chain with no client beyond it) or the walk
			// hit an unparseable entry before finding a real one. Either
			// way, fall through rather than return "" or an unvalidated
			// string.
		}

		// Check X-Real-IP. Conventionally SET (not appended) by the
		// immediate proxy, so it carries no list/appending semantics and no
		// leftmost-selection risk. It is still validated as a real IP for
		// the same reason XFF entries are: an attacker-chosen non-IP string
		// must never reach an audit log or rate-limit key. This
		// net.ParseIP check had NO test asserting it anywhere in this
		// codebase — a mutation deleting it left ALL FOUR packages' suites
		// green, and a trusted peer's "X-Real-IP: '; DROP TABLE audit; --"
		// came back as the resolved identity. Pinned by
		// "xri_non_ip_value_rejected_trusted_peer" in all four census
		// tables, each reproduced RED against a mutated artifact before
		// being trusted as a guard.
		//
		// # Repeated instances are refused outright
		//
		// X-Real-IP had the identical Header.Get defect as X-Forwarded-For
		// above — the same loopback probe returned the forged FIRST instance
		// from a trusted peer — but the REMEDY differs deliberately, because
		// the two headers' semantics differ. X-Forwarded-For is a LIST each
		// hop APPENDS to, so combination is meaning-preserving and the
		// rightmost entry is principled. X-Real-IP is SET, not appended:
		// nginx's proxy_set_header "[a]llows redefining or appending fields
		// to the request header passed to the proxied server", where
		// "appending" means adding a FIELD to the header block rather than
		// concatenating a value, so the conventional single-line X-Real-IP
		// deployment redefines it — exactly ONE instance is the invariant,
		// and two or more mean that invariant is
		// ALREADY violated: at least one was not set by the trusted proxy,
		// and nothing in the message says which. Picking the last (or the
		// first) would be a GUESS about which hop wrote which line, so this
		// honours NONE of them and falls through to RemoteAddr.
		//
		// That reasoning is exact when the instances DISAGREE. It does not
		// hold when they are BYTE-IDENTICAL — "8.8.8.8" twice names one
		// address however the lines were split between hops — and this guard
		// refuses that case too: over-strict, deliberately, in the SAFE
		// direction, because agreement is a fact about the duplicate's
		// SPELLING rather than about who wrote it, and a caller able to
		// inject a second line at all can equally inject a MATCHING one.
		// Pinned by "xri_two_identical_instances_still_refused", so it is a
		// recorded decision rather than an unexamined consequence of the
		// len == 1 test.
		//
		// # Disclosed cost
		//
		// This is fail-CLOSED: the caller collapses onto the trusted proxy's
		// own address rather than adopting an attacker-chosen one as the
		// audit identity and rate-limit key. The cost is more than lost
		// precision — every caller behind that proxy collapses onto the SAME
		// identity and shares ONE rate-limit bucket, so a single abusive
		// caller can exhaust the quota for all of them, and the audit trail
		// cannot tell them apart afterwards. Two things bound it: it fires
		// only where the proxy ALREADY violates X-Real-IP's set semantics,
		// and X-Forwarded-For is checked FIRST and normally carries the
		// answer on such a topology anyway. Pinned by
		// "xri_two_instances_refused_falls_back_to_peer" plus
		// TestResolveMultiInstanceForwardingHeadersOverLoopback here.
		if len(xriValues) == 1 {
			if xri := normalizeHostLiteral(xriValues[0]); xri != "" {
				if ip := net.ParseIP(xri); ip != nil {
					return canonicalIPString(ip)
				}
			}
		}
	}

	// Fall back to RemoteAddr — always net.SplitHostPort, never a manual
	// split, and never the raw "host:port" string. The ephemeral source
	// port varies on every new TCP connection from the SAME caller; an
	// identity that includes it forks one real caller into a distinct
	// identity per connection.
	return normalizeRemoteAddr(r.RemoteAddr)
}

// resolveForwardedFor picks the caller's identity out of a trusted
// X-Forwarded-For header value.
//
// It walks the comma-separated entry list from RIGHT to LEFT — the order
// each hop appends in under an appending reverse proxy — skipping any entry
// that is ITSELF on the trusted allowlist, and returns the first entry that
// is not: the closest-to-the-client address this function cannot itself
// verify came from another hop it trusts.
//
// Selecting the LEFTMOST entry instead is a distinct, more serious defect:
// isTrustedProxyPeer confirms only the LAST hop (r.RemoteAddr, the immediate
// TCP peer), never any earlier hop recorded inside the header itself. Under
// an appending proxy, an attacker sending "X-Forwarded-For: 8.8.8.8" through
// that proxy causes this function to receive "8.8.8.8, <attacker's real
// IP>": the PROXY is the appender, and what it appends is the peer address
// IT observes — for a directly connected attacker, the ATTACKER's own
// address, not the proxy's. Leftmost selection then hands back the forged
// claim, exactly the bypass this right-to-left walk exists to close.
//
// Every candidate is validated with net.ParseIP before being trusted OR
// skipped: an entry that does not parse as an IP is neither confirmable as a
// trusted intermediary nor safe to hand out as a caller identity, so the
// walk stops there and Resolve falls through to X-Real-IP / RemoteAddr
// rather than guessing. The selected entry is returned in canonical form via
// canonicalIPString, not as the raw header literal — see that function's doc
// comment for the identity-forking defect it closes.
// Unexported — see Resolve's doc comment for why.
func resolveForwardedFor(xff string) string {
	entries := strings.Split(xff, ",")
	for i := len(entries) - 1; i >= 0; i-- {
		candidate := normalizeHostLiteral(entries[i])
		ip := net.ParseIP(candidate)
		if ip == nil {
			// Unparseable (or empty) entry: cannot be confirmed as a
			// trusted hop, and cannot be handed out as an identity
			// either. Stop here rather than skip past it or return it.
			return ""
		}
		if isTrustedProxyIP(ip) {
			// This hop is itself a declared, trusted intermediary —
			// presumed to have appended its own observed peer to its
			// right (already walked, or, if this was the rightmost
			// entry, IS the confirmed TCP peer). Keep walking left
			// through the chain.
			continue
		}
		return canonicalIPString(ip)
	}
	return ""
}

// isTrustedProxyPeer decides whether X-Forwarded-For / X-Real-IP may be
// honoured for a request whose immediate TCP peer is remoteAddr.
//
// Default-deny (see the package doc) matches OWASP ASVS 5.0 §4.1.3, whose
// scope is TWO conditions and not one: a header "used by the application
// and set by an intermediary layer" — the requirement names X-Real-IP and
// X-Forwarded-* among its example headers — "cannot be overridden by the
// end-user". It matches OWASP WSTG-CONF-14 too, cited by its stable ID
// rather than by URL-and-quote, which revision drift can invalidate:
// "Zero Trust: Avoid relying solely on hop-by-hop headers for critical
// security decisions". Do NOT compress that into the snappier
// "OWASP says never trust XFF unless you control the chain": that sentence
// is not OWASP's. Measured over the CheatSheetSeries cheatsheets/ corpus,
// X-Forwarded-For appears ZERO times, against positive controls proving
// the search works — XSS matches 33 files, X-Forwarded-Host 4 occurrences
// in one. No file-count denominator is quoted on purpose: it shifts with
// how a clone is scoped, while the zero and its controls do not.
//
// It matches where the ecosystem moved. Measured at chi v5.3.1, 2026-08-13:
// RealIP is DEPRECATED as "vulnerable to IP spoofing" (citing
// GHSA-3fxj-6jh8-hvhx, GHSA-rjr7-jggh-pgcp, GHSA-9g5q-2w5x-hmxf) for
// adopting True-Client-IP / X-Real-IP / the LEFTMOST X-Forwarded-For value
// "whether or not your infrastructure actually sets them", and its
// replacements take the trust boundary as a PARAMETER. Default-deny here is
// stricter still than Rails' ActionDispatch::RemoteIp, which ships a
// built-in TRUSTED_PROXIES set (loopback v4 and v6, the three RFC 1918
// ranges, fc00::/7, and the v4 and v6 link-locals) and so honours XFF
// behind any of them with no configuration at all.
// Unexported — see Resolve's doc comment for why.
//
// # The peerIP == nil guard is the fail-closed default
//
// The `peerIP == nil -> return false` line below is not defensive
// housekeeping; it is the default that makes an UNRECOGNISABLE peer
// untrusted BEFORE the allowlist is consulted at all, on the same sinks as
// the X-Real-IP validation in Resolve — the audit trail and the rate-limit
// key. Flipping it to `return true` makes a peer whose RemoteAddr is not an
// IP — a net.Listen("unix", …) peer ("@"), an empty or malformed
// RemoteAddr, or a host:port whose host is a NAME — adopt whatever
// X-Forwarded-For / X-Real-IP it sends, with NO allowlist involvement
// whatsoever.
//
// That mutation left ALL FOUR delegating packages' suites green: the
// censuses already carried non-IP peers AND already carried forwarding
// headers, but never the two TOGETHER, and this guard is only observable
// when a header is present to be wrongly honoured. Pinned by
// "xff_present_malformed_peer_stays_untrusted",
// "xff_present_unix_socket_peer_stays_untrusted",
// "xff_present_empty_peer_stays_untrusted",
// "xri_present_hostname_peer_stays_untrusted" and
// "xff_present_malformed_peer_stays_untrusted_with_allowlist_configured" in
// all four census tables, each reproduced RED against a mutated artifact
// with this guard flipped before being trusted as a guard.
//
// Ordering note (established by test, not assumed): because this guard
// returns BEFORE isTrustedProxyIP, a non-IP peer never reaches
// warnOnMalformedTrustedProxiesEntries either — so in a topology whose
// only traffic source is such a peer, a malformed allowlist produces
// fail-closed behaviour with no diagnostic at all. See
// TestZoneQualifiedIPv6AllowlistEntryIsRejectedAndReported.
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
// trusted-proxy allowlist (TrustedProxiesEnvVar). Used both for the
// immediate TCP peer (isTrustedProxyPeer, above) and, in
// resolveForwardedFor, for each earlier hop recorded inside a trusted
// X-Forwarded-For chain — an address this function has no independent way
// to verify actually sits where the header claims, but which the operator
// has explicitly declared trustworthy by address. Unexported — see
// Resolve's doc comment for why.
func isTrustedProxyIP(ip net.IP) bool {
	configured := strings.TrimSpace(os.Getenv(TrustedProxiesEnvVar))
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
			// A bracketed bare IPv6 allowlist entry ("[2001:db8::1]") must
			// be stripped before parsing — net.ParseIP rejects brackets
			// outright, so an unstripped bracketed entry would silently
			// never match its peer, regardless of how the operator wrote
			// it.
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

// lastWarnedTrustedProxies de-duplicates the malformed-allowlist
// diagnostic (below). It remembers the value most recently OBSERVED, not
// most recently warned about: the Store below runs unconditionally, ahead
// of the scan that decides whether anything is worth warning about, so a
// CLEAN value overwrites the slot just as a malformed one does. The dedupe
// is therefore per CONSECUTIVE RUN of a value rather than per distinct
// value — a fixed misconfiguration warns once while it remains configured
// (isTrustedProxyIP runs on every request that reaches a trust decision),
// but switching away and back warns again. The one-slot cache is a
// deliberate floor: the alternative is an unbounded set keyed by
// operator-supplied strings — unbounded process-lifetime growth driven by
// configuration churn, to suppress a diagnostic whose whole purpose is to
// be noticed.
var lastWarnedTrustedProxies atomic.Value // holds string

// warnOnMalformedTrustedProxiesEntries logs a single, de-duplicated
// diagnostic warning when TrustedProxiesEnvVar contains an entry that is
// neither a valid bare IP nor a valid CIDR. This is deliberately
// non-fatal and changes no trust decision whatsoever — a malformed entry
// is, and remains, silently skipped by isTrustedProxyIP above (fails
// closed for that one entry) — it exists solely so a misconfigured
// allowlist is diagnosable from logs rather than only from behaviour that
// looks identical to "not configured yet."
//
// This function was once entirely unasserted, and TWO independent mutations
// survived every suite: deleting the dedupe below, and deleting the call to
// this function altogether — the whole diagnostic was silently removable.
// Pinned now by TestWarnOnMalformedTrustedProxiesEntriesEmitsWarning (the
// warning fires and names both the offending entries and the variable) and
// TestWarnOnMalformedTrustedProxiesEntriesDedupesPerConsecutiveRun (warns
// once; silent while that value stays configured; warns AGAIN on a new
// value; and warns AGAIN on a RETURN to an already-warned value, including
// after an intervening CLEAN one — the leg that separates
// per-consecutive-run from per-distinct-value).
//
// The stakes are not merely cosmetic. net.ParseIP("fe80::1%eth0") returns
// nil, so a ZONE-QUALIFIED IPv6 allowlist entry is silently skipped and
// its peer can never be trusted even when an operator writes that exact
// address. It fails closed — the correct direction — and this warning is
// the ONLY signal telling the operator their link-local entry will never
// match. See TestZoneQualifiedIPv6AllowlistEntryIsRejectedAndReported.
//
// # CIDR entries with host bits set
//
// This function also reports a SECOND, distinct class, and the asymmetry
// with the one above is why it had to be added: the malformed class fails
// CLOSED (too little trust) and was already warned, while this one fails
// OPEN (too much trust) and was completely silent.
//
// net.ParseCIDR("10.0.0.5/8") returns err == nil with the prefix MASKED to
// 10.0.0.0/8 — it does not reject host bits, and isTrustedProxyIP then
// matches on that masked prefix. So an operator who writes "10.0.0.5/8"
// intending to trust ONE host silently trusts 16,777,214 addresses, every
// one of which may then dictate its own caller identity via
// X-Forwarded-For / X-Real-IP. Reproduced: with that allowlist,
// isTrustedProxyIP(10.255.255.254) returns true.
//
// It is DIAGNOSED, not rejected. Masking is Go's documented ParseCIDR
// contract, so refusing the entry would put this package at odds with its
// own parser, and silently narrowing it to the single host would fail
// CLOSED on an operator who genuinely meant the network and merely wrote a
// member address. There is direct precedent for masks-and-warns: nginx's
// set_real_ip_from masks the same notation and logs "low address bits of
// ... are meaningless" while still accepting the entry.
//
// The ecosystem is NOT unanimous, so do not restate this as one convention
// that every tool shares. iproute2 REFUSES host bits outright: `ip route
// add 10.0.0.5/8 dev lo` fails with "Error: Invalid prefix for given prefix
// length" (exit 2 — distinct from the exit-1 generic parse error it gives
// for a malformed address), while `ip route add 192.168.7.0/24 dev lo`
// succeeds. Masking is a defensible convention with real precedent, not a
// universal one; the warning below is what makes the choice visible either
// way. See TestCIDRAllowlistEntryWithHostBitsIsWidenedAndReported.
func warnOnMalformedTrustedProxiesEntries(configured string) {
	if last, ok := lastWarnedTrustedProxies.Load().(string); ok && last == configured {
		return
	}
	lastWarnedTrustedProxies.Store(configured)

	var malformed []string
	var hostBitsSet []string
	for _, entry := range strings.Split(configured, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			declared, cidr, err := net.ParseCIDR(entry)
			if err != nil {
				malformed = append(malformed, entry)
			} else if !declared.Equal(cidr.IP) {
				// Host bits set. See the doc comment above for why this
				// is warned but NOT rejected.
				hostBitsSet = append(hostBitsSet,
					entry+" (trusts the whole "+cidr.String()+")")
			}
			continue
		}
		if net.ParseIP(stripBrackets(entry)) == nil {
			malformed = append(malformed, entry)
		}
	}
	if len(malformed) > 0 {
		log.Printf("llm-verifier/clientip: %s contains %d entr(y/ies) that are neither a valid IP nor a "+
			"valid CIDR and will be ignored (fails closed for those entries): %v — check your deployment "+
			"configuration", TrustedProxiesEnvVar, len(malformed), malformed)
	}
	if len(hostBitsSet) > 0 {
		log.Printf("llm-verifier/clientip: %s contains %d CIDR entr(y/ies) whose HOST BITS are set, which "+
			"trusts the entire surrounding network and not the single host written — this WIDENS trust and "+
			"fails OPEN: %v — write a /32 (or /128) if you meant exactly one host, or the network address "+
			"itself if you meant the network",
			TrustedProxiesEnvVar, len(hostBitsSet), hostBitsSet)
	}
}

// stripBrackets removes a single matching pair of square brackets wrapping
// an otherwise-bare IPv6 literal (e.g. "[2001:db8::1]" -> "2001:db8::1").
// It is a no-op for anything else. This is the one shape
// net.SplitHostPort itself does not cover — a bracketed host with no port
// at all — so it is applied explicitly wherever that degenerate form can
// occur, preventing it from silently forking into a second, distinct
// identity from its "[2001:db8::1]:<port>" sibling. Unexported — see
// Resolve's doc comment for why.
func stripBrackets(host string) string {
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		return host[1 : len(host)-1]
	}
	return host
}

// normalizeRemoteAddr extracts the caller's address from
// http.Request.RemoteAddr, which Go's HTTP server ORDINARILY sets to
// "host:port" (bracketing an IPv6 host per net.SplitHostPort's own
// contract) — but "ordinarily", never "always": the field is a plain string
// with no guarantee attached, and this function exists partly because it is
// not one. Its own two non-"host:port" branches below handle a value that
// splits to an EMPTY host (":12345") and a value with no ":port" suffix at
// all, and the census pins a unix-socket peer "@", a bare hostname, an empty
// RemoteAddr and a zone-qualified IPv6 literal. net.SplitHostPort — never a
// hand-rolled split, never the raw string — is the only correct way to take
// it apart: it always returns the host UNbracketed and WITHOUT the ephemeral
// source port, which varies per connection and would otherwise fork one real
// caller into a distinct identity on every new connection. Unexported — see
// Resolve's doc comment for why.
func normalizeRemoteAddr(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		if host == "" {
			// A port was present but the host portion was empty (e.g.
			// ":12345"). There is no caller identity to extract here at
			// all; the original string is returned unmodified rather
			// than an invented sentinel.
			return remoteAddr
		}
		return host
	}
	// No ":port" suffix to split off. This is not automatically malformed
	// — a bare host with no port needs nothing stripped — so it is used
	// AS-IS, except for the one degenerate shape that would otherwise
	// silently fork into a second identity: a redundant bracket pair
	// around an otherwise-bare host ("[2001:db8::1]" with no port at
	// all).
	return stripBrackets(remoteAddr)
}

// normalizeHostLiteral trims and bracket-strips a caller-address literal
// taken from a forwarding header (an X-Forwarded-For entry or X-Real-IP).
// Unlike RemoteAddr, a header-supplied literal is not guaranteed to carry a
// port, so net.SplitHostPort is not the right tool here — but be precise
// about WHY, because the intuitive reason is wrong. SplitHostPort does NOT
// mis-split an unbracketed IPv6 literal on one of its colons: measured, it
// REJECTS it, returning an empty host and the error "address 2001:db8::1:
// too many colons in address" (likewise for "::1", "::" and
// "fe80::1%eth0"). The only shape that splits silently is a single-colon
// string such as "a:b", which is not an IPv6 literal. The hazard is
// therefore a hard error on every bare IPv6 address — noisy and detectable,
// not a silent misparse — but it is still the wrong tool for a value that
// legitimately has no port. The defensive bracket-strip alone is
// sufficient: it normalizes a bracketed header literal onto the SAME
// identity a direct connection from that address resolves to (see
// normalizeRemoteAddr). Unexported — see Resolve's doc comment for why.
func normalizeHostLiteral(literal string) string {
	return stripBrackets(strings.TrimSpace(literal))
}

// canonicalIPString renders an already-parsed, already-validated caller
// address in net.IP's single canonical textual form, and is applied to
// every identity Resolve derives from a FORWARDING HEADER (an
// X-Forwarded-For entry or X-Real-IP).
//
// Header literals are written by whatever is upstream — a proxy's
// configuration, or one hop further out the caller itself — so ONE real
// caller can arrive spelled several different ways. Returning the raw
// literal made each spelling a DISTINCT audit-trail identity and rate-limit
// key for the same caller: the same forking failure stripBrackets and
// normalizeRemoteAddr already exist to prevent (bracketing, and the
// ephemeral source port), on two further axes, both reproduced:
//
//   - IPv4-mapped IPv6: a trusted proxy's "X-Forwarded-For: ::ffff:8.8.8.8"
//     resolved to "::ffff:8.8.8.8" while plain "8.8.8.8" resolved to
//     "8.8.8.8" — two identities, one caller. Go's net stack already
//     canonicalises on the DIRECT path, so canonicalising here is what makes
//     the two agree rather than an independent convention: measured, an IPv4
//     client reaching a dual-stack "[::]" listener yields RemoteAddr
//     "127.0.0.1:<port>", and (&net.TCPAddr{IP:
//     net.ParseIP("::ffff:8.8.8.8"), Port: 443}).String() is "8.8.8.8:443".
//   - Hex case: "2001:DB8::1" vs "2001:db8::1" resolved to two identities.
//
// net.IP.String() collapses both — and any other spelling that parses to
// the same address — onto one value: the dotted quad for a v4 or v4-mapped
// address, lowercase RFC 5952 form for v6 (that RFC's "MUST be represented
// in lowercase"). Applying it is safe here BECAUSE the caller has already
// established ip != nil, so it can neither invent an identity nor rewrite
// one the validation rejected.
//
// Deliberately NOT applied to the RemoteAddr fallback (normalizeRemoteAddr):
// that path must pass through verbatim every value net.ParseIP does not
// accept — a unix-socket peer "@", a hostname, an empty or malformed
// RemoteAddr, AND a ZONE-QUALIFIED IPv6 literal such as "fe80::1%eth0"
// (pinned by "ipv6_zone_qualified_remoteaddr_with_port" in all four census
// tables; net.ParseIP returns nil for the zone form, so canonicalising there
// would destroy a real, connectable link-local peer identity). Unexported —
// see Resolve's doc comment for why.
func canonicalIPString(ip net.IP) string {
	return ip.String()
}
