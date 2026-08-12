package security

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

// securityClientIPTrustedProxiesEnvVar is defined in client_ip_trust.go
// (the production fix). RED-capture evidence for this test (RED_MODE=1,
// proving both defects on the pre-fix artifact) was captured BEFORE that
// file existed, using a temporary local placeholder of the same name/value
// — see docs/qa evidence for this ticket for the captured pre-fix output.

// TestHXC299ExtractIPAddressTrustRED — §11.4.115 RED-baseline-on-the-broken-artifact
// for HXC-299, registered in the §11.4.135 standing regression-guard suite.
//
// # The two defects
//
// extractIPAddress (security/security.go) has two independent defects — a
// worse sibling of HXC-292's getClientIP (api/middleware.go):
//
// Defect 1 (identity-collapse truncation): the RemoteAddr fallback split the
// address with strings.Split(r.RemoteAddr, ":")[0] — splitting on EVERY
// colon, not just the port separator. For a direct IPv6 connection Go sets
// RemoteAddr to "[2001:db8::1]:54321"; splitting on ":" yields
// ["[2001","db8","","1]","54321"], and index [0] is "[2001" — an opening
// bracket and the first hextet, discarding everything else. TWO GENUINELY
// DIFFERENT callers sharing only their first hextet ("[2001:db8::1]:11111"
// and "[2001:dead:beef::99]:22222") both collapse onto the SAME identity
// "[2001". This is WORSE than HXC-292's pre-fix defect: HXC-292's defect kept
// identities DISTINCT (merely inconsistently bracketed); this defect makes
// distinct callers INDISTINGUISHABLE — anything counting or recording per
// caller (e.g. AuditTrail.LogRequest's AuditEntry.IPAddress, this function's
// one production call site) attributes multiple real callers' actions to one
// shared, truncated, non-address string.
//
// Defect 2 (unconditional forwarding-header trust): X-Forwarded-For and
// X-Real-IP are honoured verbatim with NO permitted-intermediary list — an
// even more direct instance of HXC-292's defect 2, since this function does
// not even select consistently (it always takes the leftmost X-Forwarded-For
// entry, unconditionally). Any caller able to reach this function's consumer
// could state any client identity it liked.
//
// # Polarity switch (§11.4.115)
//
// RED_MODE=1 reproduces each defect against the CURRENT pre-fix behaviour and
// PASSes there. RED_MODE=0 (default) is the standing GREEN guard: it FAILs if
// either defect is reintroduced.
//
// # Why the assertions are pinned to literals
//
// Every expected identity below is a test-local literal, never a value
// re-derived from the code under test — a test that asks the code under test
// what it should have produced cannot fail.
func TestHXC299ExtractIPAddressTrustRED(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}
	if redMode != "0" && redMode != "1" {
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}

	t.Run("defect1_identity_collapse_truncation", func(t *testing.T) {
		hxc299RunIdentityCollapseCase(t, redMode)
	})

	t.Run("defect2_header_forgery", func(t *testing.T) {
		hxc299RunHeaderForgeryCase(t, redMode)
	})
}

// hxc299RunIdentityCollapseCase reproduces (RED_MODE=1) or guards against
// (RED_MODE=0) defect 1: two GENUINELY DIFFERENT direct IPv6 callers, sharing
// only their leading hextet, MUST resolve to two DISTINCT identities. The
// pre-fix artifact truncates both to the shared literal "[2001" — a
// collapsed, shared, non-address identity.
func hxc299RunIdentityCollapseCase(t *testing.T, redMode string) {
	t.Helper()

	const (
		callerARemoteAddr = "[2001:db8::1]:11111"
		callerBRemoteAddr = "[2001:dead:beef::99]:22222"
		collapsedIdentity = "[2001"
		callerAIdentity   = "2001:db8::1"
		callerBIdentity   = "2001:dead:beef::99"
	)

	reqA := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqA.RemoteAddr = callerARemoteAddr
	gotA := extractIPAddress(reqA)

	reqB := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqB.RemoteAddr = callerBRemoteAddr
	gotB := extractIPAddress(reqB)

	if redMode == "1" {
		if gotA != collapsedIdentity || gotB != collapsedIdentity {
			t.Fatalf("RED_MODE=1: defect1 did NOT reproduce — callers %q and %q resolved to %q and %q, "+
				"want both to collapse onto the pre-fix truncated identity %q. Run RED_MODE=1 against the "+
				"pre-fix artifact to see this PASS.", callerARemoteAddr, callerBRemoteAddr, gotA, gotB,
				collapsedIdentity)
		}
		if gotA != gotB {
			t.Fatalf("RED_MODE=1: defect1 did NOT reproduce — expected both distinct callers to collapse "+
				"onto ONE shared identity, got %q vs %q (they did not collapse)", gotA, gotB)
		}
		t.Logf("RED_MODE=1 PASS: defect1 reproduced — two genuinely different callers (%q, %q) both "+
			"truncated to the SAME shared identity %q", callerARemoteAddr, callerBRemoteAddr, gotA)
		return
	}

	// GREEN: the two distinct callers MUST resolve to two distinct,
	// canonical, unbracketed identities — never collapsed onto a shared
	// truncated prefix.
	if gotA != callerAIdentity {
		t.Fatalf("defect1 regression: caller A RemoteAddr %q resolved to %q, want the canonical "+
			"unbracketed identity %q", callerARemoteAddr, gotA, callerAIdentity)
	}
	if gotB != callerBIdentity {
		t.Fatalf("defect1 regression: caller B RemoteAddr %q resolved to %q, want the canonical "+
			"unbracketed identity %q", callerBRemoteAddr, gotB, callerBIdentity)
	}
	if gotA == gotB {
		t.Fatalf("defect1 regression: two genuinely different callers (%q, %q) collapsed onto the SAME "+
			"identity %q — net.SplitHostPort must be used, never a naive strings.Split on every colon",
			callerARemoteAddr, callerBRemoteAddr, gotA)
	}
	t.Logf("GREEN: caller A %q -> %q, caller B %q -> %q — distinct identities preserved",
		callerARemoteAddr, gotA, callerBRemoteAddr, gotB)
}

// hxc299RunHeaderForgeryCase reproduces (RED_MODE=1) or guards against
// (RED_MODE=0) defect 2: a caller with NO trusted-proxy relationship to this
// service's consumer must NOT be able to mint an arbitrary client identity
// merely by sending X-Forwarded-For.
func hxc299RunHeaderForgeryCase(t *testing.T, redMode string) {
	t.Helper()

	// Explicitly unset — the untrusted-by-default posture must hold even if
	// some other test in this process happened to set the allowlist and a
	// t.Setenv cleanup raced (defence in depth; each subtest that DOES need
	// trust uses t.Setenv with its own scoped cleanup).
	t.Setenv(securityClientIPTrustedProxiesEnvVar, "")

	const (
		realPeer   = "203.0.113.9:53211" // the attacker's own TCP connection
		forgedIP   = "198.51.100.66"     // the identity the attacker claims via the header
		realPeerIP = "203.0.113.9"
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = realPeer
	req.Header.Set("X-Forwarded-For", forgedIP)
	got := extractIPAddress(req)

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

// TestHXC299ExtractIPAddressCensus is the §11.4.146 STEP-3 fan-out: every
// address form and header/trust combination that can reach extractIPAddress,
// each with its explicit expected outcome. This is the standing GREEN suite
// (not RED_MODE-gated) — see TestHXC299ExtractIPAddressTrustRED above for the
// two defect-specific polarity-switch guards.
func TestHXC299ExtractIPAddressCensus(t *testing.T) {
	for _, tc := range hxc299ExtractIPAddressCases() {
		t.Run(tc.name, func(t *testing.T) {
			if tc.trustedProxies != "" {
				t.Setenv(securityClientIPTrustedProxiesEnvVar, tc.trustedProxies)
			} else {
				t.Setenv(securityClientIPTrustedProxiesEnvVar, "")
			}

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" || tc.xffPresentEmpty {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xri != "" || tc.xriPresentEmpty {
				req.Header.Set("X-Real-IP", tc.xri)
			}

			got := extractIPAddress(req)
			if got != tc.want {
				t.Fatalf("%s: extractIPAddress() = %q, want %q\n  remoteAddr=%q xff=%q(present=%v) "+
					"xri=%q(present=%v) trustedProxies=%q",
					tc.outcome, got, tc.want, tc.remoteAddr, tc.xff,
					tc.xff != "" || tc.xffPresentEmpty, tc.xri, tc.xri != "" || tc.xriPresentEmpty,
					tc.trustedProxies)
			}
			t.Logf("outcome: %s -> %q", tc.outcome, got)
		})
	}
}

// hxc299ExtractIPAddressCase is one enumerated (address-form x header x
// trust) combination and its required outcome.
type hxc299ExtractIPAddressCase struct {
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

func hxc299ExtractIPAddressCases() []hxc299ExtractIPAddressCase {
	return []hxc299ExtractIPAddressCase{
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
			outcome:    "defect1 guard: bracketed IPv6 RemoteAddr must resolve UNbracketed, not truncated",
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
			outcome:    "control: a non-IP host:port form is unaffected by the fix",
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
			outcome:    "boundary: a port with no host portion carries no identity; original string preserved unmodified",
		},
		{
			// The exact HXC-299 forensic case: two genuinely different
			// callers sharing only their first hextet must NOT collapse.
			name:       "ipv6_shared_first_hextet_distinct_callers_second",
			remoteAddr: "[2001:dead:beef::99]:22222",
			want:       "2001:dead:beef::99",
			outcome:    "defect1 regression guard: a caller sharing ONLY its first hextet with another caller must still resolve to its OWN distinct identity",
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

		// ---- F1-class: XFF list semantics — RIGHTMOST NON-TRUSTED entry
		// wins, walking right to left and skipping entries that are
		// themselves trusted. NEVER leftmost — extractIPAddress's pre-fix
		// behaviour picked the leftmost entry unconditionally, which is the
		// exact bypass HXC-292's F1 review round caught and fixed in the
		// sibling function; mirrored here rather than re-risking the same
		// mistake in a fresh implementation. ----
		{
			name:           "xff_multiple_ips_rightmost_untrusted_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.7, 172.20.0.5, 10.0.0.1",
			trustedProxies: "172.20.0.0/16",
			want:           "10.0.0.1",
			outcome:        "F1-class: walking right to left, the rightmost entry (10.0.0.1) is not itself trusted, so it wins immediately -- the leftmost entry is never even consulted",
		},
		{
			// The critical F1-class regression guard: an attacker prepends a
			// forged claim; the trusted, appending proxy appends the
			// attacker's REAL address to the right of it. The real
			// (rightmost, untrusted) client MUST win.
			name:           "xff_attacker_prepended_forged_real_client_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "8.8.8.8, 203.0.113.9",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.9",
			outcome:        "F1-class regression guard: attacker-forged leftmost entry (8.8.8.8) discarded; the real, proxy-appended, non-trusted client (203.0.113.9) wins",
		},
		{
			name:           "xff_two_trusted_hops_client_beyond_both",
			remoteAddr:     "172.20.0.9:443",
			xff:            "203.0.113.50, 172.20.0.5",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.50",
			outcome:        "F1-class: two chained trusted proxies are both walked past; the real client beyond them wins",
		},
		{
			name:           "xff_malformed_entry_beyond_trusted_hop_falls_through",
			remoteAddr:     "172.20.0.5:443",
			xff:            "not-an-ip, 172.20.0.5",
			trustedProxies: "172.20.0.0/16",
			want:           "172.20.0.5",
			outcome:        "boundary: a non-IP entry beyond a trusted hop cannot be confirmed trusted nor safely returned -> walk stops, falls through to RemoteAddr",
		},
		{
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
			outcome:        "boundary: a LEADING empty entry is harmless under right-to-left walking",
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
			name:           "xff_trusted_ipv6_peer_bracketed_allowlist_entry",
			remoteAddr:     "[2001:db8::5]:443",
			xff:            "203.0.113.77",
			trustedProxies: "[2001:db8::5]",
			want:           "203.0.113.77",
			outcome:        "a bracketed IPv6 allowlist entry ([2001:db8::5]) matches an IPv6 peer at the same address -> header honoured",
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

// TestHXC299ExtractIPAddressConcurrent is the §11.4.85 concurrent-contention
// case: N goroutines simultaneously resolving distinct callers (some
// trusted, some not, some forging headers) must each get their own correct,
// uncorrupted identity — no shared mutable state in extractIPAddress /
// isTrustedProxyPeer (security-package copy) may leak one caller's
// resolution into another's.
func TestHXC299ExtractIPAddressConcurrent(t *testing.T) {
	t.Setenv(securityClientIPTrustedProxiesEnvVar, "172.20.0.0/16")

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

			got := extractIPAddress(req)
			if got != want {
				errs[i] = fmt.Sprintf("goroutine %d: extractIPAddress() = %q, want %q", i, got, want)
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
