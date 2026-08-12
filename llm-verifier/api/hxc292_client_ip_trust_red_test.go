package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

// TestHXC292ClientIPTrustRED — §11.4.115 RED-baseline-on-the-broken-artifact
// for HXC-292, registered in the §11.4.135 standing regression-guard suite.
//
// # The two defects
//
// getClientIP (api/middleware.go) had two independent defects:
//
// Defect 1 (bracket-fork / identity split): the RemoteAddr fallback split
// the address with a hand-rolled strings.LastIndex(r.RemoteAddr, ":")
// instead of net.SplitHostPort. For a direct IPv6 connection Go sets
// RemoteAddr to "[2001:db8::1]:54321"; the manual split correctly located
// the port separator (the rightmost ':') but left the surrounding brackets
// in the returned host, producing the identity "[2001:db8::1]". Meanwhile
// the X-Forwarded-For / X-Real-IP paths returned the header value verbatim
// — conventionally UNbracketed. The SAME real caller therefore forked into
// TWO distinct identities depending only on whether a forwarding header
// happened to be present on a given request. External precedent for the
// exact hazard of comparing a bracketed literal against an unbracketed one:
// CVE-2026-39361 (OpenObserve, GHSA-gcwf-3p7h-wm79), an SSRF filter bypassed
// because "[::1]" was never reconciled with a bare "::1" blocklist entry.
//
// Defect 2 (unconditional forwarding-header trust): X-Forwarded-For and
// X-Real-IP were honoured verbatim with NO permitted-intermediary list. Any
// caller able to reach this service — not merely a caller behind a
// legitimate reverse proxy — could state any client identity it liked by
// sending either header directly, defeating whatever getClientIP's output
// feeds (see api/audit_logger.go's LogHTTPRequest ClientIP field and
// RateLimitMiddleware's rl.Allow(clientIP) key in this same file).
//
// # Polarity switch (§11.4.115)
//
// RED_MODE=1 reproduces each defect against the CURRENT pre-fix behaviour
// and PASSes there. RED_MODE=0 (default) is the standing GREEN guard: it
// FAILs if either defect is reintroduced.
//
// # Why the assertions are pinned to literals
//
// Every expected identity below is a test-local literal, never a value
// re-derived from the code under test — a test that asks the code under
// test what it should have produced cannot fail.
func TestHXC292ClientIPTrustRED(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}
	if redMode != "0" && redMode != "1" {
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}

	t.Run("defect1_bracket_fork", func(t *testing.T) {
		hxc292RunBracketForkCase(t, redMode)
	})

	t.Run("defect2_header_forgery", func(t *testing.T) {
		hxc292RunHeaderForgeryCase(t, redMode)
	})

	t.Run("f1_appending_proxy_leftmost_bypass", func(t *testing.T) {
		hxc292RunAppendingProxyBypassCase(t, redMode)
	})
}

// hxc292RunBracketForkCase reproduces (RED_MODE=1) or guards against
// (RED_MODE=0) defect 1: the SAME direct IPv6 caller must resolve to the
// SAME identity whether or not a forwarding header happens to be present on
// the request. The pre-fix artifact resolves the RemoteAddr-only path to a
// bracketed "[2001:db8::1]" while a header-driven path (once headers are
// trusted at all) resolves to the unbracketed "2001:db8::1" — two distinct
// identities for one real caller.
func hxc292RunBracketForkCase(t *testing.T, redMode string) {
	t.Helper()

	const (
		v6RemoteAddr = "[2001:db8::1]:54321"
		v6Bracketed  = "[2001:db8::1]"
		v6Bare       = "2001:db8::1"
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = v6RemoteAddr
	got := getClientIP(req)

	if redMode == "1" {
		if got != v6Bracketed {
			t.Fatalf("RED_MODE=1: defect1 did NOT reproduce — direct IPv6 RemoteAddr %q resolved to "+
				"%q, want the pre-fix bracketed identity %q. Run RED_MODE=1 against the pre-fix "+
				"artifact to see this PASS.", v6RemoteAddr, got, v6Bracketed)
		}
		t.Logf("RED_MODE=1 PASS: defect1 reproduced — RemoteAddr %q resolved to bracketed identity %q",
			v6RemoteAddr, got)
		return
	}

	// GREEN: the identity MUST be the canonical unbracketed form, matching
	// what the same caller would resolve to via a trusted forwarding header
	// (see hxc292ClientIPCases's "ipv6_bracketed_remoteaddr_with_port" and
	// "xff_bracketed_ipv6_entry_trusted_peer" rows — both must agree).
	if got != v6Bare {
		t.Fatalf("defect1 regression: direct IPv6 RemoteAddr %q resolved to %q, want the canonical "+
			"unbracketed identity %q (net.SplitHostPort must be used, never a manual colon-split)",
			v6RemoteAddr, got, v6Bare)
	}
	t.Logf("GREEN: RemoteAddr %q correctly resolves to unbracketed identity %q", v6RemoteAddr, got)
}

// hxc292RunHeaderForgeryCase reproduces (RED_MODE=1) or guards against
// (RED_MODE=0) defect 2: a caller with NO trusted-proxy relationship to this
// service must NOT be able to mint an arbitrary client identity merely by
// sending X-Forwarded-For.
func hxc292RunHeaderForgeryCase(t *testing.T, redMode string) {
	t.Helper()

	// Explicitly unset — the untrusted-by-default posture must hold even if
	// some other test in this process happened to set the allowlist and a
	// t.Setenv cleanup raced (defence in depth; each subtest that DOES need
	// trust uses t.Setenv with its own scoped cleanup).
	t.Setenv(clientIPTrustedProxiesEnvVar, "")

	const (
		realPeer   = "203.0.113.9:53211" // the attacker's own TCP connection
		forgedIP   = "198.51.100.66"     // the identity the attacker claims via the header
		realPeerIP = "203.0.113.9"
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = realPeer
	req.Header.Set("X-Forwarded-For", forgedIP)
	got := getClientIP(req)

	if redMode == "1" {
		if got != forgedIP {
			t.Fatalf("RED_MODE=1: defect2 did NOT reproduce — forged X-Forwarded-For %q from untrusted "+
				"peer %q resolved to %q, want the forged identity %q to have been trusted (the "+
				"pre-fix defect). Run RED_MODE=1 against the pre-fix artifact to see this PASS.",
				forgedIP, realPeer, got, forgedIP)
		}
		t.Logf("RED_MODE=1 PASS: defect2 reproduced — forged X-Forwarded-For %q from untrusted peer "+
			"%q was trusted verbatim", forgedIP, realPeer)
		return
	}

	// GREEN: with no trusted-proxy allowlist configured, the header MUST be
	// ignored and the caller's own TCP peer address used instead.
	if got != realPeerIP {
		t.Fatalf("defect2 regression: forged X-Forwarded-For %q from untrusted peer %q resolved to "+
			"%q, want the real peer identity %q (an unconfigured/non-matching trusted-proxy "+
			"allowlist must mean the header is NOT honoured)", forgedIP, realPeer, got, realPeerIP)
	}
	t.Logf("GREEN: forged X-Forwarded-For %q from untrusted peer %q was correctly ignored; resolved "+
		"identity is the real peer %q", forgedIP, realPeer, got)
}

// hxc292RunAppendingProxyBypassCase reproduces (RED_MODE=1) or guards
// against (RED_MODE=0) F1: leftmost-entry XFF selection re-opens defect 2
// behind ANY appending proxy.
//
// isTrustedProxyPeer confirms only the LAST hop (the immediate TCP peer,
// i.e. r.RemoteAddr) — never any earlier hop recorded inside the header
// value itself. nginx's common `$proxy_add_x_forwarded_for` configuration
// APPENDS to whatever X-Forwarded-For it received rather than replacing it.
// An attacker who connects through such a proxy and sends
// "X-Forwarded-For: 8.8.8.8" therefore causes the backend to receive
// "8.8.8.8, <attacker's real IP>" — the trusted proxy has appended the
// attacker's REAL address to the RIGHT of the attacker's OWN forged claim.
// Blind "leftmost wins" selection returns the forged "8.8.8.8", handing the
// attacker exactly the bypass HXC-292 was opened to close: rate-limit
// evasion (their real bucket never fills) and audit-log identity forgery
// (api/audit_logger.go's ClientIP field records the forged value).
func hxc292RunAppendingProxyBypassCase(t *testing.T, redMode string) {
	t.Helper()

	t.Setenv(clientIPTrustedProxiesEnvVar, "172.20.0.0/16")

	const (
		trustedPeer = "172.20.0.5:443" // the trusted, appending reverse proxy
		forgedEntry = "8.8.8.8"        // the attacker's own forged, prepended claim
		realClient  = "203.0.113.9"    // the attacker's true address, appended by the trusted proxy
	)
	// nginx $proxy_add_x_forwarded_for style: appends the proxy's own
	// observed peer (the attacker) to whatever it received (the attacker's
	// forged claim) rather than replacing it.
	xff := forgedEntry + ", " + realClient

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = trustedPeer
	req.Header.Set("X-Forwarded-For", xff)
	got := getClientIP(req)

	if redMode == "1" {
		if got != forgedEntry {
			t.Fatalf("RED_MODE=1: F1 did NOT reproduce — leftmost-selection bypass not present; XFF %q "+
				"from trusted peer %q resolved to %q, want the forged leftmost entry %q. Run RED_MODE=1 "+
				"against the pre-F1-fix artifact to see this PASS.", xff, trustedPeer, got, forgedEntry)
		}
		t.Logf("RED_MODE=1 PASS: F1 reproduced — leftmost XFF selection returned the attacker-forged "+
			"entry %q from XFF %q, discarding the real client %q the trusted proxy appended",
			got, xff, realClient)
		return
	}

	// GREEN: the real, proxy-appended, non-trusted client MUST win — never
	// the attacker's own forged, prepended claim.
	if got != realClient {
		t.Fatalf("F1 regression: XFF %q from trusted peer %q resolved to %q, want the real "+
			"(rightmost, proxy-appended, non-trusted) client %q — an earlier entry in the chain must "+
			"NEVER be trusted merely because the LAST hop happens to be", xff, trustedPeer, got, realClient)
	}
	t.Logf("GREEN: appending-proxy forgery correctly defeated — XFF %q from trusted peer %q resolved "+
		"to the real client %q, not the attacker's forged entry %q", xff, trustedPeer, got, forgedEntry)
}

// TestHXC292ClientIPCensus is the §11.4.146 STEP-3 fan-out: every address
// form and header/trust combination that can reach getClientIP, each with
// its explicit expected outcome. This is the standing GREEN suite (not
// RED_MODE-gated) — see TestHXC292ClientIPTrustRED above for the two
// defect-specific polarity-switch guards.
func TestHXC292ClientIPCensus(t *testing.T) {
	for _, tc := range hxc292ClientIPCases() {
		t.Run(tc.name, func(t *testing.T) {
			if tc.trustedProxies != "" {
				t.Setenv(clientIPTrustedProxiesEnvVar, tc.trustedProxies)
			} else {
				t.Setenv(clientIPTrustedProxiesEnvVar, "")
			}

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" || tc.xffPresentEmpty {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xri != "" || tc.xriPresentEmpty {
				req.Header.Set("X-Real-IP", tc.xri)
			}

			got := getClientIP(req)
			if got != tc.want {
				t.Fatalf("%s: getClientIP() = %q, want %q\n  remoteAddr=%q xff=%q(present=%v) "+
					"xri=%q(present=%v) trustedProxies=%q",
					tc.outcome, got, tc.want, tc.remoteAddr, tc.xff,
					tc.xff != "" || tc.xffPresentEmpty, tc.xri, tc.xri != "" || tc.xriPresentEmpty,
					tc.trustedProxies)
			}
			t.Logf("outcome: %s -> %q", tc.outcome, got)
		})
	}
}

// hxc292ClientIPCase is one enumerated (address-form x header x trust)
// combination and its required outcome.
type hxc292ClientIPCase struct {
	name            string
	remoteAddr      string
	xff             string
	xffPresentEmpty bool // set the header to "" explicitly (distinct from absent)
	xri             string
	xriPresentEmpty bool
	trustedProxies  string // LLM_VERIFIER_TRUSTED_PROXIES value; "" = unset (default-deny)
	want            string
	outcome         string // human-readable description of why `want` is correct
}

func hxc292ClientIPCases() []hxc292ClientIPCase {
	return []hxc292ClientIPCase{
		// ---- valid address forms, no headers, no trust configured ----
		{
			name:       "ipv4_remoteaddr_with_port",
			remoteAddr: "192.0.2.10:54321",
			want:       "192.0.2.10",
			outcome:    "plain IPv4 RemoteAddr resolves to the bare host",
		},
		{
			name:       "ipv6_bracketed_remoteaddr_with_port",
			remoteAddr: "[2001:db8::1]:54321",
			want:       "2001:db8::1",
			outcome:    "defect1 guard: bracketed IPv6 RemoteAddr must resolve UNbracketed",
		},
		{
			name:       "ipv6_loopback_remoteaddr_with_port",
			remoteAddr: "[::1]:8080",
			want:       "::1",
			outcome:    "IPv6 loopback resolves UNbracketed",
		},
		{
			name:       "ipv6_zone_qualified_remoteaddr_with_port",
			remoteAddr: "[fe80::1%eth0]:8080",
			want:       "fe80::1%eth0",
			outcome:    "zone-qualified IPv6 literal survives net.SplitHostPort with brackets removed",
		},
		{
			name:       "hostname_remoteaddr_with_port",
			remoteAddr: "attacker.example:1234",
			want:       "attacker.example",
			outcome:    "control: a non-IP host:port form is unaffected by the bracket fix",
		},

		// ---- boundary: no port, empty, malformed ----
		{
			name:       "bare_ipv4_remoteaddr_no_port",
			remoteAddr: "203.0.113.5",
			want:       "203.0.113.5",
			outcome:    "boundary: no port suffix to strip, used as-is",
		},
		{
			name:       "bare_bracketed_ipv6_no_port_degenerate",
			remoteAddr: "[2001:db8::1]",
			want:       "2001:db8::1",
			outcome:    "boundary: degenerate bracketed-with-no-port form still unbrackets",
		},
		{
			name:       "empty_remoteaddr",
			remoteAddr: "",
			want:       "",
			outcome:    "boundary: no identity to extract, matches pre-fix behaviour (no invented sentinel)",
		},
		{
			name:       "malformed_no_colon_remoteaddr",
			remoteAddr: "not-an-address",
			want:       "not-an-address",
			outcome:    "boundary: unparseable-as-host:port string used as-is, never collapsed onto a shared bucket",
		},
		{
			name:       "remoteaddr_empty_host_with_port",
			remoteAddr: ":12345",
			want:       ":12345",
			outcome:    "boundary: a port with no host portion carries no identity; original string preserved unmodified (matches pre-fix behaviour for this exact shape)",
		},

		// ---- defect 2: forged / untrusted headers must be ignored by default ----
		{
			name:       "xff_untrusted_peer_ignored",
			remoteAddr: "203.0.113.9:1111",
			xff:        "9.9.9.9",
			want:       "203.0.113.9",
			outcome:    "defect2 guard: no trusted-proxy allowlist configured -> XFF ignored, real peer used",
		},
		{
			name:       "xri_untrusted_peer_ignored",
			remoteAddr: "203.0.113.9:2222",
			xri:        "9.9.9.9",
			want:       "203.0.113.9",
			outcome:    "defect2 guard: no trusted-proxy allowlist configured -> X-Real-IP ignored, real peer used",
		},
		{
			name:       "both_headers_untrusted_peer_ignored",
			remoteAddr: "203.0.113.9:3333",
			xff:        "9.9.9.9",
			xri:        "8.8.8.8",
			want:       "203.0.113.9",
			outcome:    "defect2 guard: neither header is honoured when the peer is untrusted",
		},
		{
			name:           "xff_peer_not_in_configured_range_ignored",
			remoteAddr:     "203.0.113.9:4444",
			xff:            "9.9.9.9",
			trustedProxies: "10.0.0.0/8",
			want:           "203.0.113.9",
			outcome:        "defect2 guard: a non-empty allowlist that simply does not match the peer still denies trust",
		},

		// ---- trusted-peer paths: bare IP entry ----
		{
			name:           "xff_trusted_peer_bare_ip_match",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.7",
			trustedProxies: "172.20.0.5",
			want:           "198.51.100.7",
			outcome:        "allowlist bare-IP entry matches the peer exactly -> header honoured",
		},

		// ---- trusted-peer paths: CIDR entry ----
		{
			name:           "xff_trusted_peer_cidr_match",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.7",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.7",
			outcome:        "allowlist CIDR entry contains the peer -> header honoured",
		},
		{
			name:           "xff_trusted_peer_multiple_allowlist_entries",
			remoteAddr:     "172.20.5.5:80",
			xff:            "198.51.100.9",
			trustedProxies: "10.0.0.0/8,172.20.0.0/16",
			want:           "198.51.100.9",
			outcome:        "comma-separated allowlist, second entry matches -> header honoured",
		},
		{
			name:           "xff_trusted_peer_malformed_allowlist_entry_ignored",
			remoteAddr:     "172.20.5.5:80",
			xff:            "198.51.100.9",
			trustedProxies: "not-a-cidr,172.20.0.0/16",
			want:           "198.51.100.9",
			outcome:        "a malformed allowlist entry is skipped, not fatal; the remaining valid entry still matches",
		},

		// ---- F1: XFF list semantics — RIGHTMOST NON-TRUSTED entry wins,
		// walking right to left and skipping entries that are themselves
		// trusted. NEVER leftmost: isTrustedProxyPeer confirms only the
		// LAST hop, so a leftmost-wins selection lets an attacker prepend
		// an arbitrary forged claim ahead of whatever a trusted, APPENDING
		// proxy (nginx's $proxy_add_x_forwarded_for) adds for its own
		// observed peer. See hxc292RunAppendingProxyBypassCase for the
		// dedicated RED/GREEN polarity guard for this exact bypass. ----
		{
			name:           "xff_multiple_ips_rightmost_untrusted_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.7, 172.20.0.5, 10.0.0.1",
			trustedProxies: "172.20.0.0/16",
			want:           "10.0.0.1",
			outcome:        "F1: walking right to left, the rightmost entry (10.0.0.1) is not itself trusted, so it wins immediately -- the leftmost entry is never even consulted",
		},
		{
			// The critical F1 regression guard: an attacker prepends a
			// forged claim; the trusted, appending proxy appends the
			// attacker's REAL address to the right of it. The real
			// (rightmost, untrusted) client MUST win.
			name:           "xff_attacker_prepended_forged_real_client_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "8.8.8.8, 203.0.113.9",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.9",
			outcome:        "F1 regression guard: attacker-forged leftmost entry (8.8.8.8) discarded; the real, proxy-appended, non-trusted client (203.0.113.9) wins",
		},
		{
			// Two trusted hops in the chain (a two-proxy deployment, both
			// within the same allow-listed CIDR): the walk must skip PAST
			// both trusted entries and land on the real client beyond them,
			// proving the walk is genuinely recursive, not single-hop.
			name:           "xff_two_trusted_hops_client_beyond_both",
			remoteAddr:     "172.20.0.9:443",
			xff:            "203.0.113.50, 172.20.0.5",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.50",
			outcome:        "F1: two chained trusted proxies (172.20.0.9 the confirmed peer, 172.20.0.5 the earlier trusted hop) are both walked past; the real client beyond them wins",
		},
		{
			// A malformed entry encountered PAST a trusted hop must break
			// the walk (fall through) rather than be skipped-as-if-trusted
			// or returned as an unvalidated identity.
			name:           "xff_malformed_entry_beyond_trusted_hop_falls_through",
			remoteAddr:     "172.20.0.5:443",
			xff:            "not-an-ip, 172.20.0.5",
			trustedProxies: "172.20.0.0/16",
			want:           "172.20.0.5",
			outcome:        "boundary: a non-IP entry beyond a trusted hop cannot be confirmed trusted nor safely returned -> walk stops, falls through to RemoteAddr",
		},
		{
			// A trailing comma (empty rightmost entry) must not crash, must
			// not be treated as trusted, and must not be returned as an
			// identity -- it breaks the walk exactly like any other
			// unparseable rightmost entry.
			name:           "xff_trailing_empty_entry_falls_through",
			remoteAddr:     "172.20.0.5:443",
			xff:            "203.0.113.5,",
			trustedProxies: "172.20.0.0/16",
			want:           "172.20.0.5",
			outcome:        "boundary: trailing comma yields an empty rightmost entry -> unparseable, walk stops, falls through to RemoteAddr",
		},
		{
			name:           "xff_leading_empty_entry_ignored_rightmost_used",
			remoteAddr:     "172.20.0.5:443",
			xff:            ",10.0.0.1",
			trustedProxies: "172.20.0.0/16",
			want:           "10.0.0.1",
			outcome:        "boundary: a LEADING empty entry is harmless under right-to-left walking -- the rightmost (only real) entry is consulted first and wins",
		},
		{
			name:           "xff_multiple_ips_with_surrounding_whitespace",
			remoteAddr:     "172.20.0.5:443",
			xff:            " 198.51.100.8 , 172.20.0.5",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.8",
			outcome:        "rightmost entry (172.20.0.5) is trusted and skipped; the next entry left is trimmed of surrounding whitespace before use",
		},
		{
			name:           "xff_bracketed_ipv6_entry_trusted_peer",
			remoteAddr:     "172.20.0.5:443",
			xff:            "[2001:db8::99]",
			trustedProxies: "172.20.0.0/16",
			want:           "2001:db8::99",
			outcome:        "a header-supplied bracketed IPv6 literal is normalised unbracketed, matching the direct-connection identity for the same address",
		},
		{
			// F5: a bracketed IPv6 literal used as a bare allowlist ENTRY
			// (not a peer/header value) must still match -- net.ParseIP
			// rejects brackets outright, so isTrustedProxyIP must strip
			// them before parsing the entry, exactly as it already does
			// for the peer/header side.
			name:           "xff_trusted_ipv6_peer_bracketed_allowlist_entry",
			remoteAddr:     "[2001:db8::5]:443",
			xff:            "203.0.113.77",
			trustedProxies: "[2001:db8::5]",
			want:           "203.0.113.77",
			outcome:        "F5: a bracketed IPv6 allowlist entry ([2001:db8::5]) matches an IPv6 peer at the same address -> header honoured",
		},
		{
			name:            "xff_header_present_but_empty_value_treated_as_absent",
			remoteAddr:      "172.20.0.5:443",
			xffPresentEmpty: true,
			xri:             "198.51.100.20",
			trustedProxies:  "172.20.0.0/16",
			want:            "198.51.100.20",
			outcome:         "an explicitly-empty X-Forwarded-For header is treated as absent, falling through to X-Real-IP",
		},

		// ---- precedence: XFF beats X-Real-IP when both present + trusted ----
		{
			name:           "both_headers_trusted_peer_xff_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.30",
			xri:            "198.51.100.31",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.30",
			outcome:        "existing precedence preserved: X-Forwarded-For is checked before X-Real-IP",
		},

		// ---- X-Real-IP alone, trusted peer ----
		{
			name:           "xri_only_trusted_peer",
			remoteAddr:     "172.20.0.5:443",
			xri:            "198.51.100.20",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.20",
			outcome:        "X-Real-IP alone is honoured once the peer is trusted",
		},
	}
}

// TestHXC292ClientIPConcurrent is the §11.4.85 concurrent-contention case:
// N goroutines simultaneously resolving distinct callers (some trusted, some
// not, some forging headers) must each get their own correct, uncorrupted
// identity — no shared mutable state in getClientIP / isTrustedProxyPeer may
// leak one caller's resolution into another's.
func TestHXC292ClientIPConcurrent(t *testing.T) {
	t.Setenv(clientIPTrustedProxiesEnvVar, "172.20.0.0/16")

	const n = 64
	var wg sync.WaitGroup
	errs := make([]string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			var (
				req  *http.Request
				want string
			)
			if i%2 == 0 {
				// Trusted peer, forwarded identity must be honoured.
				peer := fmt.Sprintf("172.20.%d.%d:443", (i/256)%256, i%256)
				forwarded := fmt.Sprintf("198.51.100.%d", i%256)
				req = httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = peer
				req.Header.Set("X-Forwarded-For", forwarded)
				want = forwarded
			} else {
				// Untrusted peer forging a header: must be ignored, own
				// peer address used.
				peer := fmt.Sprintf("203.0.113.%d:%d", i%256, 1000+i)
				req = httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = peer
				req.Header.Set("X-Forwarded-For", "6.6.6.6")
				want = peer[:len(peer)-len(fmt.Sprintf(":%d", 1000+i))]
			}

			got := getClientIP(req)
			if got != want {
				errs[i] = fmt.Sprintf("goroutine %d: getClientIP() = %q, want %q", i, got, want)
			}
		}(i)
	}
	wg.Wait()

	for _, e := range errs {
		if e != "" {
			t.Error(e)
		}
	}
}
