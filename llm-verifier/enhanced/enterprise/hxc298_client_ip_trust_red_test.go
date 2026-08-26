package enterprise

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"digital.vasic.llmsverifier/clientip"
)

// TestHXC298ClientIPTrustRED — §11.4.115 RED-baseline-on-the-broken-artifact
// for HXC-298, registered in the §11.4.135 standing regression-guard suite.
//
// # The defects
//
// EnterpriseAPI.getClientIP (enhanced/enterprise/api.go) had two independent
// defects, both worse than either already-repaired sibling:
//
// Defect 1 (raw RemoteAddr, port included): the RemoteAddr fallback
// returned r.RemoteAddr completely unprocessed — no net.SplitHostPort, not
// even a hand-rolled split. Every direct connection therefore resolved to
// "host:port" verbatim. Since the ephemeral source port is assigned fresh
// by the OS on every new TCP connection, the SAME caller forks into a
// DIFFERENT identity on every single connection it makes — a much wider
// blast radius than HXC-292's bracket-fork (which only forked identity
// depending on whether a forwarding header happened to be present) or
// HXC-299's hextet-collapse (which only forked/collapsed identity for
// bracketed IPv6 literals). Here that identity feeds the enterprise
// access-control audit log (RBACManager.logAudit's ipAddress parameter, via
// auditMiddleware, handleLogin's failure/success paths, handleLogout, and
// the token-refresh handler) — the record of who did what for permission
// decisions — so a single real caller's own history fragments into
// unrelated-looking entries purely because of which ephemeral port its OS
// happened to assign.
//
// Defect 2 (unconditional forwarding-header trust, leftmost selection):
// X-Forwarded-For and X-Real-IP were honoured verbatim with NO
// permitted-intermediary list, and — like HXC-299's pre-fix state, unlike
// HXC-292's — the leftmost X-Forwarded-For entry was taken unconditionally
// (strings.Split(xff, ",")[0]). Any caller able to reach this API — not
// merely one behind a legitimate reverse proxy — could dictate any client
// identity it liked into the audit log, and even a hardened deployment that
// puts a trusted, APPENDING reverse proxy in front (nginx's
// $proxy_add_x_forwarded_for) remains exploitable: the attacker's own
// forged, prepended entry is what leftmost selection returns.
//
// # Wiring status (§11.4.108 / §11.4.124 — established, not assumed)
//
// A full import-graph census at HXC-298 fix time found NO production entry
// point constructs EnterpriseManager / EnterpriseAPI or calls Start(ctx):
// cmd/main.go (the shipped llm-verifier binary) imports api, client,
// database, llmverifier, seed, and tui — never enhanced/enterprise. The
// config/{development,staging,production}.yaml files each declare an
// "enterprise:" section, but cmd/main.go never reads that key, so it cannot
// activate this subsystem either. The ONLY importers of enhanced/enterprise
// anywhere in this module are three files under tests/:
// integration_simple_test.go, working_components_test.go, and
// system_validation_test_fixed.go — each constructing an EnterpriseConfig
// for unit-style coverage, none starting the HTTP server. NOTE:
// system_validation_test_fixed.go does NOT end in "_test.go" (it ends in
// "_fixed.go") and so is NOT itself a Go test file — `go list` places it in
// the tests package's regular GoFiles, compiled like any other source file.
// This does not change the latency conclusion: the load-bearing fact is
// that digital.vasic.llmsverifier/tests is never imported by any main
// package (`go list -deps ./cmd/...` has no entry for llmsverifier/tests
// or llmsverifier/enhanced/enterprise), not the file-suffix of any one
// importer inside it. This subsystem is therefore LATENT in the
// currently-shipped binary — same class of finding as HXC-299's
// AuditTrail, and per that fix's own citation of §11.4.124: not a
// reason to leave it broken. It is a complete, self-consistent RBAC + audit
// feature (handleAudit, gated by PermissionLogsView, surfaces exactly the
// log this fix protects) that could be wired into cmd/main.go at any time,
// and per §11.4.124 it must be correct when that happens rather than
// silently forging or collapsing caller identities the moment it is.
//
// # Reuse-vs-mirror-vs-extract (HXC-308)
//
// enhanced/enterprise, api, and security all live in the SAME Go module
// (digital.vasic.llmsverifier) — there is no module boundary here. api's
// getClientIP and security's resolveClientIP are unexported, so neither can
// be imported directly without exporting something; HXC-299's own doc
// comment additionally notes that importing FROM security INTO api would
// have inverted that codebase's dependency direction (api/ was frozen for
// that fix's scope). Rather than adding a THIRD independent copy — which
// would only widen the HXC-308 drift risk this fix instead closes — the
// corrected algorithm was extracted into the new clientip package (a leaf
// package with no dependents among api/, security/, or enhanced/enterprise/
// at extraction time, so no cycle is possible). api/middleware.go's
// getClientIP, security/client_ip_trust.go's resolveClientIP, and this
// package's EnterpriseAPI.getClientIP all now delegate to clientip.Resolve
// — one implementation, not three that can drift.
//
// # Polarity switch (§11.4.115)
//
// RED_MODE=1 reproduces each defect against the CURRENT pre-fix behaviour
// and PASSes there. RED_MODE=0 (default) is the standing GREEN guard: it
// FAILs if either defect is reintroduced.
func TestHXC298ClientIPTrustRED(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}
	if redMode != "0" && redMode != "1" {
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}

	t.Run("defect1_raw_remoteaddr_port_identity_fork", func(t *testing.T) {
		hxc298RunRawRemoteAddrCase(t, redMode)
	})

	t.Run("defect2_header_forgery", func(t *testing.T) {
		hxc298RunHeaderForgeryCase(t, redMode)
	})

	t.Run("f1_appending_proxy_leftmost_bypass", func(t *testing.T) {
		hxc298RunAppendingProxyBypassCase(t, redMode)
	})
}

// hxc298RunRawRemoteAddrCase reproduces (RED_MODE=1) or guards against
// (RED_MODE=0) defect 1: the SAME direct caller connecting twice, on two
// DIFFERENT ephemeral source ports, must resolve to the SAME identity. The
// pre-fix artifact instead echoes r.RemoteAddr verbatim, so the port itself
// becomes part of the "identity" and forks every connection apart.
func hxc298RunRawRemoteAddrCase(t *testing.T, redMode string) {
	t.Helper()

	api := &EnterpriseAPI{}

	const (
		sameCallerIP  = "203.0.113.9"
		connectionOne = "203.0.113.9:54321"
		connectionTwo = "203.0.113.9:60002"
	)

	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = connectionOne
	got1 := api.getClientIP(req1)

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = connectionTwo
	got2 := api.getClientIP(req2)

	if redMode == "1" {
		if got1 != connectionOne || got2 != connectionTwo {
			t.Fatalf("RED_MODE=1: defect1 did NOT reproduce as expected — want raw RemoteAddr echoed "+
				"verbatim (port included): got1=%q want %q; got2=%q want %q. Run RED_MODE=1 against the "+
				"pre-fix artifact to see this PASS.", got1, connectionOne, got2, connectionTwo)
		}
		if got1 == got2 {
			t.Fatalf("RED_MODE=1: defect1 did NOT reproduce — same caller %q on two connections resolved "+
				"to the SAME identity (%q == %q); want the pre-fix behaviour where identity forks with "+
				"the ephemeral source port", sameCallerIP, got1, got2)
		}
		t.Logf("RED_MODE=1 PASS: defect1 reproduced — same caller %q forked into two distinct audit-log "+
			"identities %q and %q purely because the ephemeral source port differed", sameCallerIP, got1, got2)
		return
	}

	// GREEN: the SAME caller must resolve to the SAME identity regardless
	// of which ephemeral source port its TCP connection happened to use,
	// and that identity must be the bare IP — never host:port.
	if got1 != sameCallerIP || got2 != sameCallerIP {
		t.Fatalf("defect1 regression: same caller %q on two connections resolved to %q and %q, want both "+
			"to resolve to the bare IP %q (net.SplitHostPort must be used, never the raw RemoteAddr)",
			sameCallerIP, got1, got2, sameCallerIP)
	}
	if got1 != got2 {
		t.Fatalf("defect1 regression: same caller %q forked into two identities (%q vs %q) across two "+
			"connections that differ only by ephemeral source port", sameCallerIP, got1, got2)
	}
	t.Logf("GREEN: same caller %q on two connections (ports 54321 and 60002) both resolve to the stable "+
		"identity %q", sameCallerIP, got1)
}

// hxc298RunHeaderForgeryCase reproduces (RED_MODE=1) or guards against
// (RED_MODE=0) defect 2: a caller with NO trusted-proxy relationship to
// this API must NOT be able to mint an arbitrary client identity merely by
// sending X-Forwarded-For.
func hxc298RunHeaderForgeryCase(t *testing.T, redMode string) {
	t.Helper()

	// Explicitly unset — the untrusted-by-default posture must hold even
	// if some other test in this process happened to set the allowlist and
	// a t.Setenv cleanup raced (defence in depth; each subtest that DOES
	// need trust uses t.Setenv with its own scoped cleanup).
	t.Setenv(clientip.TrustedProxiesEnvVar, "")

	api := &EnterpriseAPI{}

	const (
		realPeer   = "203.0.113.9:53211" // the attacker's own TCP connection
		forgedIP   = "198.51.100.66"     // the identity the attacker claims via the header
		realPeerIP = "203.0.113.9"
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = realPeer
	req.Header.Set("X-Forwarded-For", forgedIP)
	got := api.getClientIP(req)

	if redMode == "1" {
		if got != forgedIP {
			t.Fatalf("RED_MODE=1: defect2 did NOT reproduce — forged X-Forwarded-For %q from untrusted "+
				"peer %q resolved to %q, want the forged identity %q to have been trusted (the pre-fix "+
				"defect). Run RED_MODE=1 against the pre-fix artifact to see this PASS.",
				forgedIP, realPeer, got, forgedIP)
		}
		t.Logf("RED_MODE=1 PASS: defect2 reproduced — forged X-Forwarded-For %q from untrusted peer %q "+
			"was trusted verbatim into the audit log", forgedIP, realPeer)
		return
	}

	// GREEN: with no trusted-proxy allowlist configured, the header MUST be
	// ignored and the caller's own TCP peer address used instead.
	if got != realPeerIP {
		t.Fatalf("defect2 regression: forged X-Forwarded-For %q from untrusted peer %q resolved to %q, "+
			"want the real peer identity %q (an unconfigured/non-matching trusted-proxy allowlist must "+
			"mean the header is NOT honoured)", forgedIP, realPeer, got, realPeerIP)
	}
	t.Logf("GREEN: forged X-Forwarded-For %q from untrusted peer %q was correctly ignored; resolved "+
		"identity is the real peer %q", forgedIP, realPeer, got)
}

// hxc298RunAppendingProxyBypassCase reproduces (RED_MODE=1) or guards
// against (RED_MODE=0) the leftmost-selection bypass: under a trusted,
// APPENDING reverse proxy, an attacker's own forged, prepended
// X-Forwarded-For entry must never win over the real client the proxy
// appended to its right.
func hxc298RunAppendingProxyBypassCase(t *testing.T, redMode string) {
	t.Helper()

	t.Setenv(clientip.TrustedProxiesEnvVar, "172.20.0.0/16")

	api := &EnterpriseAPI{}

	const (
		trustedPeer = "172.20.0.5:443" // the trusted, appending reverse proxy
		forgedEntry = "8.8.8.8"        // the attacker's own forged, prepended claim
		realClient  = "203.0.113.9"    // the attacker's true address, appended by the trusted proxy
	)
	xff := forgedEntry + ", " + realClient

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = trustedPeer
	req.Header.Set("X-Forwarded-For", xff)
	got := api.getClientIP(req)

	if redMode == "1" {
		if got != forgedEntry {
			t.Fatalf("RED_MODE=1: leftmost-selection bypass did NOT reproduce — XFF %q from trusted peer "+
				"%q resolved to %q, want the forged leftmost entry %q. Run RED_MODE=1 against the pre-fix "+
				"artifact to see this PASS.", xff, trustedPeer, got, forgedEntry)
		}
		t.Logf("RED_MODE=1 PASS: leftmost-selection bypass reproduced — XFF %q resolved to the "+
			"attacker-forged entry %q, discarding the real, proxy-appended client %q", xff, got, realClient)
		return
	}

	// GREEN: the real, proxy-appended, non-trusted client MUST win — never
	// the attacker's own forged, prepended claim.
	if got != realClient {
		t.Fatalf("leftmost-selection regression: XFF %q from trusted peer %q resolved to %q, want the "+
			"real (rightmost, proxy-appended, non-trusted) client %q", xff, trustedPeer, got, realClient)
	}
	t.Logf("GREEN: appending-proxy forgery correctly defeated — XFF %q from trusted peer %q resolved to "+
		"the real client %q, not the attacker's forged entry %q", xff, trustedPeer, got, forgedEntry)
}

// TestHXC298ClientIPCensus is the §11.4.146 STEP-3 fan-out: every address
// form and header/trust combination that can reach EnterpriseAPI.getClientIP,
// each with its explicit expected outcome. This is the standing GREEN suite
// (not RED_MODE-gated) — see TestHXC298ClientIPTrustRED above for the
// defect-specific polarity-switch guards.
//
// This table is also the HXC-308 drift detector this fix's scope permits:
// it is derived from the SAME algorithm now shared via the clientip
// package, so if a future change ever made EnterpriseAPI.getClientIP stop
// delegating to clientip.Resolve, this census — not merely the RED tests
// above — would immediately catch any behavioural divergence across the
// full enumerated case space, not just the three headline defects.
func TestHXC298ClientIPCensus(t *testing.T) {
	for _, tc := range hxc298ClientIPCases() {
		t.Run(tc.name, func(t *testing.T) {
			if tc.trustedProxies != "" {
				t.Setenv(clientip.TrustedProxiesEnvVar, tc.trustedProxies)
			} else {
				t.Setenv(clientip.TrustedProxiesEnvVar, "")
			}

			api := &EnterpriseAPI{}

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			// F11: xffInstances / xriInstances drive SEPARATE header
			// instances via Header.Add -- the exact shape Go's HTTP server
			// produces from repeated header LINES on the wire, and the shape
			// r.Header.Get silently truncates to its FIRST element. Header.Set
			// cannot express it (it replaces), so before these fields the
			// census could not reach the defect at all.
			if len(tc.xffInstances) > 0 {
				for _, v := range tc.xffInstances {
					req.Header.Add("X-Forwarded-For", v)
				}
			} else if tc.xff != "" || tc.xffPresentEmpty {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			// R11-F2: xriPresentEmpty is emitted as its OWN header
			// instance and may COMBINE with xriInstances. Header.Set could
			// not express that (it replaces), which is why the field was
			// previously unfalsifiable: every row that set it produced a
			// single-instance request, and Resolve treats a lone
			// present-but-empty X-Real-IP exactly like an absent one. Adding
			// it as a SEPARATE instance alongside a valid one is
			// observable -- it pushes Header.Values to len 2 and trips the
			// multi-instance refusal. Behaviour-preserving for every
			// pre-existing row: with no xriInstances and an empty tc.xri
			// this Adds exactly the one empty value Set used to write, and
			// with a non-empty tc.xri on a fresh request Add and Set are
			// indistinguishable.
			if tc.xriPresentEmpty {
				req.Header.Add("X-Real-IP", "")
			}
			if len(tc.xriInstances) > 0 {
				for _, v := range tc.xriInstances {
					req.Header.Add("X-Real-IP", v)
				}
			} else if tc.xri != "" {
				req.Header.Add("X-Real-IP", tc.xri)
			}

			got := api.getClientIP(req)
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

// hxc298ClientIPCase is one enumerated (address-form x header x trust)
// combination and its required outcome.
type hxc298ClientIPCase struct {
	name            string
	remoteAddr      string
	xff             string
	xffPresentEmpty bool // set the header to "" explicitly (distinct from absent)
	xri             string
	xriPresentEmpty bool
	// F11: repeated header INSTANCES (separate wire lines). Non-empty
	// takes precedence over the single-value xff/xri fields above.
	xffInstances   []string
	xriInstances   []string
	trustedProxies string // LLM_VERIFIER_TRUSTED_PROXIES value; "" = unset (default-deny)
	want           string
	outcome        string // human-readable description of why `want` is correct
}

func hxc298ClientIPCases() []hxc298ClientIPCase {
	return []hxc298ClientIPCase{
		// ---- valid address forms, no headers, no trust configured ----
		{
			name:       "ipv4_remoteaddr_with_port",
			remoteAddr: "192.0.2.10:54321",
			want:       "192.0.2.10",
			outcome:    "plain IPv4 RemoteAddr resolves to the bare host, never host:port",
		},
		{
			name:       "ipv6_bracketed_remoteaddr_with_port",
			remoteAddr: "[2001:db8::1]:54321",
			want:       "2001:db8::1",
			outcome:    "bracketed IPv6 RemoteAddr resolves UNbracketed and without the port",
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
			outcome:    "control: a non-IP host:port form is unaffected by the port-strip fix",
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
			outcome:    "boundary: no identity to extract, no invented sentinel",
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

		// ---- leftmost-vs-rightmost XFF list semantics — RIGHTMOST
		// NON-TRUSTED entry wins, walking right to left and skipping
		// entries that are themselves trusted. NEVER leftmost. ----
		{
			name:           "xff_multiple_ips_rightmost_untrusted_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.7, 172.20.0.5, 10.0.0.1",
			trustedProxies: "172.20.0.0/16",
			want:           "10.0.0.1",
			outcome:        "walking right to left, the rightmost entry (10.0.0.1) is not itself trusted, so it wins immediately -- the leftmost entry is never even consulted",
		},
		{
			name:           "xff_attacker_prepended_forged_real_client_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "8.8.8.8, 203.0.113.9",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.9",
			outcome:        "leftmost-selection regression guard: attacker-forged leftmost entry (8.8.8.8) discarded; the real, proxy-appended, non-trusted client (203.0.113.9) wins",
		},
		{
			name:           "xff_two_trusted_hops_client_beyond_both",
			remoteAddr:     "172.20.0.9:443",
			xff:            "203.0.113.50, 172.20.0.5",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.50",
			outcome:        "two chained trusted proxies are both walked past; the real client beyond them wins",
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
			name:           "xff_trusted_ipv6_peer_bracketed_allowlist_entry",
			remoteAddr:     "[2001:db8::5]:443",
			xff:            "203.0.113.77",
			trustedProxies: "[2001:db8::5]",
			want:           "203.0.113.77",
			outcome:        "a bracketed IPv6 allowlist entry matches an IPv6 peer at the same address -> header honoured",
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

		// ---- both forwarding headers present from a TRUSTED peer, on the
		// topologies where PRECEDENCE and fallback agree ----
		{
			name:           "both_headers_trusted_peer_agreeing_xff_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.30",
			xri:            "198.51.100.30",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.30",
			outcome:        "the canonical single-hop nginx pair (`proxy_set_header X-Real-IP $remote_addr` + `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for`) emits BOTH headers from the SAME $remote_addr, so precedence and fallback would return the same address either way -- the topology where the two headers agree and nothing rides on which is consulted first",
		},
		{
			name:           "xff_multihop_xri_names_middle_hop",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.30, 10.9.9.9",
			xri:            "10.9.9.9",
			trustedProxies: "172.20.0.0/16,10.9.9.9",
			want:           "198.51.100.30",
			outcome:        "this is why X-Forwarded-For takes PRECEDENCE over X-Real-IP rather than the reverse. On a two-hop chain (client -> CDN 10.9.9.9 -> nginx -> us) the proxy's X-Real-IP names the CDN, not the client; the XFF walk skips the allowlisted CDN and returns the real client. Preferring X-Real-IP here would merge EVERY client behind that CDN into one audit identity and one rate-limit bucket",
		},

		// ---- R7-1 / R10 (round-7 finding; round-8 guard REMOVED in
		// round 10, §11.4.120 reconciliation): X-Forwarded-For takes
		// strict PRECEDENCE, and the residual exposure is pinned, not
		// guarded ----
		//
		// Round 8 added a cross-header equality guard that refused BOTH
		// headers on a mismatch, and the rows here asserted that refusal.
		// Round 9 proved the guard a live regression on UNFORGED traffic,
		// and round 10 removed it: the premise it rested on (a proxy
		// managing both headers derives both from its own peer) is false
		// for ingress-nginx with realip active and for every mixed-vendor
		// chain, and the guard was SYMMETRIC by construction so it also
		// fired on the mirror topology where the walk had already produced
		// the correct answer. Per §11.4.120 these rows are RECONCILED to
		// the restored behaviour rather than deleted -- they now pin BOTH
		// that behaviour AND the one exposure precedence leaves open, so
		// neither can change silently. See Resolve's "# Header PRECEDENCE,
		// and its threat model" doc comment for the full reasoning and
		// cited sources; mirrored from clientip's TestResolveCensus, where
		// the algorithm actually lives.
		{
			name:           "xff_forged_vs_xri_authoritative_trusted_peer",
			remoteAddr:     "172.20.0.5:443",
			xff:            "1.2.3.4",
			xri:            "203.0.113.9",
			trustedProxies: "172.20.0.0/16",
			want:           "1.2.3.4",
			outcome:        "R10 DOCUMENTED RESIDUAL EXPOSURE, deliberately pinned so it cannot change unnoticed: a trusted proxy that SETS X-Real-IP but RELAYS an unmanaged X-Forwarded-For verbatim lets the caller's own forged chain win on precedence. This is a misconfigured trust boundary -- allowlisting a proxy asserts it sanitises forwarding headers, and one that passes caller-controlled XFF through does not. The remedy is configuration (the proxy must append to or strip XFF), not a cross-header guard: the guard that covered this row refused BOTH headers whenever the rightmost X-Forwarded-For entry and a single X-Real-IP did not compare equal, counting not-comparable as disagreement, so it ALSO refused these six ZERO-FORGERY rows of this same table -- xff_rightmost_has_port_with_xri_present, xff_rightmost_unparseable_with_valid_xri_present, xri_from_earlier_hop_with_later_appended_xff, xri_from_earlier_hop_with_two_later_appended_xff_hops, xri_present_but_empty_value_treated_as_absent, xri_whitespace_only_value_treated_as_absent -- while handing callers behind an appending proxy a new unattributability capability (xri_forged_vs_xff_appended_trusted_peer). Those six are NAMED, not counted. The guard's source survives in NO commit -- this package is untracked, and the guard's own identifier has zero commits in all history (`git log --all -S forwardingHeadersCorroborate`, run from the repository root: 0), while the bare word `corroborate` has zero hits under llm-verifier/ at HEAD (`git grep -il corroborate HEAD -- llm-verifier/`: 0). BOTH measurements name their scope deliberately, because the UNSCOPED forms do NOT return zero: `git grep -il corroborate HEAD` matches 4 tracked governance/docs files outside llm-verifier/, and `git log --all -S corroborate` matches 2 commits -- none of them this guard. An unscoped claim here would be refuted by the very command a reader would run to check it, which is worse than an uncheckable one. So any TALLY is a property of a RECONSTRUCTION, not a recoverable fact. Measured: an IP-equality reconstruction and a literal-equality reconstruction refuse exactly the six above in common; xff_ipv4_mapped_entry_canonicalises_with_xri_present is refused by the literal-equality one ONLY, because ::ffff:198.51.100.30 and 198.51.100.30 are the SAME hop in two spellings. That single reconstruction-dependent row is the whole reason successive revisions of this line disagreed about the figure, which is why this one names rows instead of counting them -- the disagreement is demonstrated in the sentence above rather than asserted as a tally of past edits, which this package being untracked would make uncheckable anyway",
		},
		{
			name:           "xff_forged_multi_entry_vs_xri_authoritative_trusted_peer",
			remoteAddr:     "172.20.0.5:443",
			xff:            "1.2.3.4, 5.6.7.8",
			xri:            "203.0.113.9",
			trustedProxies: "172.20.0.0/16",
			want:           "5.6.7.8",
			outcome:        "R10: the same documented residual exposure with a padded chain -- the right-to-left walk returns the rightmost non-allowlisted entry (5.6.7.8), confirming the exposure is the RELAYED-XFF topology itself and not an artefact of a single-entry chain",
		},
		{
			name:           "xri_forged_vs_xff_appended_trusted_peer",
			remoteAddr:     "172.20.0.5:443",
			xff:            "1.2.3.4, 198.51.100.77",
			xri:            "9.9.9.9",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.77",
			outcome:        "R10 MIRROR topology, and the reason the corroboration guard could not be narrowed: on a proxy that APPENDS X-Forwarded-For but does not manage X-Real-IP, the proxy-appended 198.51.100.77 is authoritative and the caller's bogus X-Real-IP is simply never reached. Precedence returns the correct client here. The removed guard saw only a mismatch -- indistinguishable from the row above -- and collapsed this onto the proxy, letting any caller behind an appending proxy make itself and everyone sharing that proxy unattributable by adding one header",
		},
		{
			name:           "xri_from_earlier_hop_with_later_appended_xff",
			remoteAddr:     "172.20.0.5:443",
			xff:            "172.20.0.9",
			xri:            "203.0.113.77",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.77",
			outcome:        "R10 regression guard for the removed corroboration guard, ZERO forgery: an edge hop sets X-Real-IP to the real client while a later hop appends X-Forwarded-For naming only allowlisted infrastructure. The XFF walk exhausts on allowlisted entries and correctly falls through to X-Real-IP. The removed guard saw two disagreeing headers and collapsed this legitimate caller onto 172.20.0.5 -- the shape Kubernetes ingress-nginx ships deliberately (X-Real-IP from $remote_addr, X-Forwarded-For from $full_x_forwarded_for on $realip_remote_addr) and that every mixed-vendor chain produces, since Envoy/HAProxy/ALB append XFF and never manage the non-standard X-Real-IP",
		},
		{
			name:           "xri_from_earlier_hop_with_two_later_appended_xff_hops",
			remoteAddr:     "172.20.0.5:443",
			xff:            "172.20.0.9, 172.20.0.7",
			xri:            "203.0.113.77",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.77",
			outcome:        "R10: the same legitimate multi-hop shape with THREE hops of allowlisted infrastructure in the chain -- confirming the fall-through to X-Real-IP is driven by allowlist exhaustion and not by the chain having exactly one entry",
		},
		{
			name:           "xff_ipv4_mapped_entry_canonicalises_with_xri_present",
			remoteAddr:     "172.20.0.5:443",
			xff:            "::ffff:198.51.100.30",
			xri:            "198.51.100.30",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.30",
			outcome:        "D1: an IPv4-mapped X-Forwarded-For entry resolves to the canonical dotted-quad form, so a proxy that spells the two headers differently does not fork one caller into two identities -- the precedence path returns the XFF entry, canonicalised, rather than the identically-valued X-Real-IP",
		},
		{
			name:           "xri_two_instances_with_xff_present_xff_still_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.30",
			xriInstances:   []string{"198.51.100.30", "8.8.8.8"},
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.30",
			outcome:        "F11 + R10: two X-Real-IP instances are refused outright as a X-Real-IP value, but that refusal is confined to the X-Real-IP fallback -- a usable X-Forwarded-For is consulted FIRST and still resolves, so an attacker cannot suppress a proxy-appended chain by injecting a second X-Real-IP instance",
		},
		{
			name:            "xri_present_but_empty_value_treated_as_absent",
			remoteAddr:      "172.20.0.5:443",
			xff:             "203.0.113.77",
			xriPresentEmpty: true,
			trustedProxies:  "172.20.0.0/16",
			want:            "203.0.113.77",
			outcome:         "R9-2/R10: an explicitly-EMPTY X-Real-IP is treated as absent, mirroring the xff_header_present_but_empty_value_treated_as_absent row on the other header -- one misconfigured `proxy_set_header X-Real-IP` emitting an empty value must not unattribute an entire deployment, which is exactly what the removed corroboration guard did by treating not-comparable as disagreement",
		},
		{
			name:           "xri_whitespace_only_value_treated_as_absent",
			remoteAddr:     "172.20.0.5:443",
			xff:            "203.0.113.77",
			xri:            "   ",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.77",
			outcome:        "R9-2/R10 boundary: a whitespace-only X-Real-IP is equally unusable and equally must not suppress a valid X-Forwarded-For -- the trim-then-parse path rejects it as a VALUE without taking the whole request down with it",
		},
		{
			name:            "xri_empty_instance_suppresses_authoritative_xri_and_collapses_onto_peer",
			remoteAddr:      "172.20.0.5:443",
			xriPresentEmpty: true,
			xriInstances:    []string{"8.8.8.8"},
			trustedProxies:  "172.20.0.0/16",
			want:            "172.20.0.5",
			outcome:         "R11-F2 CAPABILITY, previously pinned by no row in any table: a caller who can inject ONE EMPTY X-Real-IP line alongside the proxy's authoritative one pushes Header.Values(\"X-Real-IP\") to len 2, trips the multi-instance refusal, and SUPPRESSES the proxy's authoritative X-Real-IP entirely -- collapsing itself and everyone sharing that proxy onto the single peer identity 172.20.0.5: one unattributable audit identity, one rate-limit bucket. The reachability envelope is the SAME one the F11 separate-header-line defect already treats as real (a trusted peer plus a proxy with add-header rather than replace-header semantics), and the capability is the SAME unattributability capability R10 cited as its reason for REMOVING the corroboration guard, reached by another route -- so it is pinned here rather than left to be rediscovered. This row is also what makes xriPresentEmpty falsifiable: it carries NO X-Forwarded-For, so deleting the harness's `if tc.xriPresentEmpty` branch leaves one valid instance and resolves 8.8.8.8 instead of the peer",
		},
		{
			name:           "xff_rightmost_unparseable_with_valid_xri_present",
			remoteAddr:     "172.20.0.5:443",
			xff:            "203.0.113.9, unknown",
			xri:            "198.51.100.44",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.44",
			outcome:        "R9-3/R10: a non-IP rightmost X-Forwarded-For entry (`unknown`, which real proxies do emit) STOPS the right-to-left walk -- resolveForwardedFor returns \"\" rather than stepping past an entry it cannot validate -- so Resolve falls through to X-Real-IP, and it is X-Real-IP that supplies the answer here, NOT a leftward continuation of the walk. The X-Real-IP value is deliberately DIFFERENT from the leftward XFF entry 203.0.113.9 so the row discriminates the two mechanisms: a skip-and-continue walk would return 203.0.113.9 and fail. The removed guard instead read not-comparable as disagreement and discarded that valid authoritative X-Real-IP too",
		},
		{
			name:           "xff_rightmost_has_port_with_xri_present",
			remoteAddr:     "172.20.0.5:443",
			xff:            "203.0.113.9, 198.51.100.7:1234",
			xri:            "198.51.100.44",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.44",
			outcome:        "R9-3/R10 boundary: a ported rightmost X-Forwarded-For entry is likewise not a bare IP, so it STOPS the walk exactly as `unknown` does; the answer again arrives via the X-Real-IP fall-through, not via a leftward continuation, and the distinct X-Real-IP value proves which of the two supplied it. Refusing the whole request, as the removed guard did, would have discarded that fall-through as well",
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

		// ---- HXC-298 own defect-1 focus: same caller across two
		// connections, distinguished only by ephemeral source port ----
		{
			name:       "same_caller_different_ports_no_headers",
			remoteAddr: "198.51.100.42:33001",
			want:       "198.51.100.42",
			outcome:    "defect1 guard: the ephemeral source port is never part of the resolved identity",
		},

		// ---- F1 (post-extraction review finding): the net.ParseIP(xri)
		// validation guard on X-Real-IP, deleted by a reviewer's mutation,
		// left ALL FOUR delegating packages' suites green -- and, from a
		// trusted peer, "X-Real-IP: '; DROP TABLE audit; --" came back as
		// the RESOLVED IDENTITY, landing directly in the enterprise RBAC
		// audit log (RBACManager.logAudit's ipAddress parameter). This case
		// pins the guard directly: a non-IP X-Real-IP value from a TRUSTED
		// peer must NEVER be returned -- it must fall through to the
		// normalized RemoteAddr, exactly as an untrusted peer's forged
		// header would. ----
		{
			name:           "xri_non_ip_value_rejected_trusted_peer",
			remoteAddr:     "172.20.0.5:443",
			xri:            "'; DROP TABLE audit; --",
			trustedProxies: "172.20.0.0/16",
			want:           "172.20.0.5",
			outcome:        "F1 guard: a trusted peer's X-Real-IP MUST still be validated with net.ParseIP -- a non-IP value (here, an injection-style string) must never be returned as the resolved identity",
		},

		// ---- F2 (post-extraction review finding): the FAIL-CLOSED default
		// inside isTrustedProxyPeer -- `if peerIP == nil { return false }`
		// -- is what makes an UNRECOGNISABLE peer untrusted BEFORE the
		// allowlist is consulted at all. It is the same defect class as F1,
		// one guard over, feeding the same sink (the enterprise RBAC audit
		// log, via RBACManager.logAudit's ipAddress parameter), and a
		// reviewer's mutation flipping it to `return true` left ALL FOUR
		// delegating packages' suites green.
		//
		// The gap was structural: this census already carried non-IP peers
		// (hostname_remoteaddr_with_port, malformed_no_colon_remoteaddr,
		// empty_remoteaddr) and already carried forwarding headers -- but
		// never the two TOGETHER, and the guard is only observable when a
		// header is present to be wrongly honoured. These rows supply that
		// pairing. Mirrored from clientip's own TestResolveCensus, where the
		// guard actually lives. ----
		{
			name:       "xff_present_malformed_peer_stays_untrusted",
			remoteAddr: "not-an-address",
			xff:        "9.9.9.9",
			want:       "not-an-address",
			outcome:    "F2 guard: a peer whose RemoteAddr does not parse as an IP is UNTRUSTED by default, so its X-Forwarded-For is ignored and the unparseable RemoteAddr is used as-is",
		},
		{
			name:       "xff_present_unix_socket_peer_stays_untrusted",
			remoteAddr: "@",
			xff:        "9.9.9.9",
			want:       "@",
			outcome:    "F2 guard: a unix-socket peer (net.Listen(\"unix\") sets RemoteAddr to \"@\") has no IP to match against the allowlist and MUST stay untrusted, never adopting the header's claim",
		},
		{
			name:       "xff_present_empty_peer_stays_untrusted",
			remoteAddr: "",
			xff:        "9.9.9.9",
			want:       "",
			outcome:    "F2 guard: an empty RemoteAddr yields no peer IP, so trust MUST fail closed and the header MUST NOT become the caller identity",
		},
		{
			name:       "xri_present_hostname_peer_stays_untrusted",
			remoteAddr: "attacker.example:1234",
			xri:        "9.9.9.9",
			want:       "attacker.example",
			outcome:    "F2 guard: a syntactically-valid host:port whose host is a NAME, not an IP, still yields no peer IP -- X-Real-IP MUST be ignored and the hostname used",
		},
		{
			name:           "xff_present_malformed_peer_stays_untrusted_with_allowlist_configured",
			remoteAddr:     "not-an-address",
			xff:            "9.9.9.9",
			trustedProxies: "172.20.0.0/16",
			want:           "not-an-address",
			outcome:        "F2 guard: the fail-closed default holds independently of allowlist state -- a configured (but unmatchable) allowlist must not turn an unparseable peer into a trusted one",
		},

		// ---- F6 / F7 / F8 (round-3 review findings): three more guards
		// whose observability each requires a COMBINATION of conditions no
		// earlier census row paired -- the same structural miss-pattern as
		// F2, which is why three consecutive review rounds each found one
		// more. Each was PROVED by a surviving mutation (the guard deleted,
		// ALL FOUR delegating packages' suites still green), and each row
		// below was reproduced RED against its own mutant before being
		// trusted as a guard. All three feed this package's sink, the
		// enterprise RBAC audit log (RBACManager.logAudit's ipAddress
		// parameter).
		//
		// F6 is the most serious of the three: F1, F2, F7 and F8 all fail
		// CLOSED when their guard is removed, but F6's surviving mutant
		// WIDENS TRUST -- it grants trusted-proxy status to a peer the
		// operator never allowlisted, then writes that peer's
		// attacker-supplied X-Forwarded-For into the RBAC audit log.
		// Mirrored from clientip's own TestResolveCensus, where the guards
		// actually live. ----
		{
			name:           "xff_present_unterminated_bracket_allowlist_entry_stays_untrusted",
			remoteAddr:     "10.0.0.1:5000",
			xff:            "9.9.9.9",
			trustedProxies: "[10.0.0.10",
			want:           "10.0.0.1",
			outcome:        "F6 guard: an UNTERMINATED bracketed allowlist entry ([10.0.0.10) is not a matching pair, so stripBrackets leaves it alone, it parses as neither IP nor CIDR, and the peer stays UNTRUSTED -- dropping the closing-bracket conjunct would truncate it to the different host 10.0.0.1, match this peer, and honour its forged X-Forwarded-For (9.9.9.9)",
		},
		{
			name:           "xri_bracketed_ipv6_trusted_peer",
			remoteAddr:     "10.0.0.1:5000",
			xri:            "[2001:db8::5]",
			trustedProxies: "10.0.0.1",
			want:           "2001:db8::5",
			outcome:        "F7 guard: a trusted peer's BRACKETED IPv6 X-Real-IP is bracket-stripped before validation, resolving to the same identity a direct connection from that address would -- without the strip it fails net.ParseIP and the caller collapses onto the proxy's own address (10.0.0.1)",
		},
		{
			name:           "xri_whitespace_padded_trusted_peer",
			remoteAddr:     "10.0.0.1:5000",
			xri:            "  8.8.8.8  ",
			trustedProxies: "10.0.0.1",
			want:           "8.8.8.8",
			outcome:        "F7 guard: a trusted peer's whitespace-padded X-Real-IP is TRIMMED before validation -- without the trim it fails net.ParseIP and the caller collapses onto the proxy's own address (10.0.0.1)",
		},
		{
			name:           "xff_present_bracketed_noport_peer_on_allowlist",
			remoteAddr:     "[2001:db8::1]",
			xff:            "9.9.9.9",
			trustedProxies: "2001:db8::1",
			want:           "9.9.9.9",
			outcome:        "F8 guard: a bracketed IPv6 peer with NO port takes net.SplitHostPort's error path, where the brackets MUST still be stripped before net.ParseIP -- otherwise this explicitly-allowlisted proxy silently stops being trusted and its X-Forwarded-For is ignored, resolving the caller to the proxy (2001:db8::1) instead of the real client",
		},

		// F9: isTrustedProxyIP TRIMS EACH ALLOWLIST ENTRY, not just the
		// env var as a whole. The reason it needed a row at all is a
		// property of every allowlist written above it: none of them puts
		// a SPACE after a comma. The outer
		// strings.TrimSpace(os.Getenv(...)) strips only the two ends of
		// the whole env var, so with no INTERIOR padding anywhere, no
		// entry ever reached the split loop needing a trim of its own and
		// the per-entry trim was never exercised BY THE CENSUS TABLE. That
		// premise is checkable in one command rather than taken on trust:
		//
		//	grep -c 'trustedProxies: *"[^"]*, ' <this file>
		//
		// is 1, and the single hit IS this row. The trim only becomes
		// observable when the allowlist has TWO OR MORE entries, the
		// matching one is not the first, AND the list is written with the
		// space that "a, b" leaves after the comma -- the way every
		// operator actually writes a list. Note the
		// asymmetry this closes: the X-Forwarded-For entry list DID get
		// whitespace coverage (xff_multiple_ips_rightmost_untrusted_wins
		// feeds "198.51.100.7, 172.20.0.5, 10.0.0.1" through
		// normalizeHostLiteral's trim), so the identical
		// comma-space-separated shape was tested on the header side and
		// never on the allowlist side. Without the trim, " 172.20.0.5"
		// fails net.ParseIP, this explicitly-allowlisted proxy silently
		// stops being trusted, and its X-Forwarded-For is discarded --
		// fails closed, like F7 and F8.
		//
		// # R7-2 (post-review finding, closed by this note)
		//
		// Three earlier revisions of this comment justified the row with
		// claims that were either self-invalidating or never checkable.
		// They are recorded here, corrected, because the same shapes are
		// easy to reintroduce:
		//
		//   - A hard-coded COUNT of the rows above ("35 census rows
		//     precede this one", corrected once to 40/41 per package) was
		//     wrong AGAIN, in every one of the four tables, by the time the
		//     F11 and R7-1 rows had been inserted above it. A number that
		//     must be re-derived on every row addition is a
		//     self-description guaranteed to drift, so it is REMOVED
		//     rather than fixed a third time -- deliberately NOT replaced
		//     with today's correct count, which would drift the same way.
		//     The space-after-comma property stated above does not move
		//     when rows are added and carries the argument by itself.
		//   - "every preceding row configured a SINGLE-entry allowlist" --
		//     false here (multi-entry values DO precede this row, e.g.
		//     "10.0.0.0/8,172.20.0.0/16", "not-a-cidr,172.20.0.0/16" and
		//     "172.20.0.0/16,10.9.9.9"), and, since the F11 rows landed,
		//     false in clientip/clientip_test.go where it originated too.
		//     Entry COUNT was never the load-bearing condition; interior
		//     PADDING is, which is why the premise above is stated in
		//     those terms instead.
		//   - "dropping it survived all four suites" -- an unqualified
		//     claim about a pre-F9 census that no longer exists and, in an
		//     as-yet-uncommitted package, cannot be re-checked against any
		//     history. It is also plainly false of the tree it sits in:
		//     with the per-entry trim deleted today, ALL FOUR suites FAIL
		//     -- this row in each of the four census tables, plus, in
		//     clientip only, TestWarnOnMalformedTrustedProxiesEntriesEmitsWarning
		//     -- its "the one VALID allowlist entry must still match the
		//     peer" assertion, Resolve() = "203.0.113.240", want
		//     "198.51.100.60". (Cited by assertion text, not by line
		//     number, deliberately: the first draft of this very note
		//     cited clientip_test.go:763 and the note's own insertion
		//     pushed that assertion to a different line before the edit
		//     was finished.) That warn test asserts the trust decision
		//     for the space-padded allowlist
		//     "f4-emit-probe-not-an-ip, 203.0.113.240,10.0.0.0/33", so it
		//     guards the per-entry trim INDEPENDENTLY of this row -- which
		//     means "never the thing under test" was never true at suite
		//     level for clientip, whatever the census did or did not
		//     cover. The scoped claim -- never exercised by the CENSUS
		//     TABLES -- is the only half that was ever verifiable, and it
		//     is exactly what this row fixes.
		{
			name:           "xff_present_second_allowlist_entry_space_separated_trusted",
			remoteAddr:     "172.20.0.5:443",
			xff:            "9.9.9.9",
			trustedProxies: "10.0.0.1, 172.20.0.5",
			want:           "9.9.9.9",
			outcome:        "F9 guard: each allowlist entry is trimmed individually, so the second entry of the conventional comma-space list \"10.0.0.1, 172.20.0.5\" matches this peer and its X-Forwarded-For is honoured -- without the per-entry trim \" 172.20.0.5\" fails net.ParseIP, the allowlisted proxy silently stops being trusted, and the caller collapses onto the proxy (172.20.0.5)",
		},
		// R7-3: a CIDR allowlist entry whose HOST BITS ARE SET silently
		// widens trust. net.ParseCIDR("10.0.0.5/8") returns err == nil
		// with the prefix MASKED to 10.0.0.0/8, so an operator who writes
		// one host's address allowlists 16,777,214 of them -- and every
		// one may then dictate its own caller identity through
		// X-Forwarded-For. Note the asymmetry that made this worth a row:
		// the SAFE-direction allowlist mistake (a zone-qualified entry,
		// which fails CLOSED) was already diagnosed, while this
		// DANGEROUS-direction one was completely silent, because the entry
		// IS a valid CIDR and nothing looked past that.
		//
		// The BEHAVIOUR is deliberately unchanged -- masking is Go's
		// documented ParseCIDR contract and matches how nginx's
		// set_real_ip_from, iproute2 and most ACL parsers read the same
		// notation, so narrowing or refusing it here would make this
		// package disagree with every other tool reading the operator's
		// own configuration. What closes R7-3 is the operator-facing
		// WARNING (see clientip's warnOnMalformedTrustedProxiesEntries doc
		// comment and TestCIDRAllowlistEntryWithHostBitsIsWidenedAndReported).
		// This row is the guard that the warning is not warning about
		// nothing: it pins the widening as REAL, observable end-to-end
		// through this package's own entry point. If the masking ever
		// stops, this row fails and the warning's premise is gone with it.
		{
			name:           "cidr_allowlist_entry_with_host_bits_widens_trust",
			remoteAddr:     "10.255.255.254:443",
			xff:            "198.51.100.62",
			trustedProxies: "10.0.0.5/8",
			want:           "198.51.100.62",
			outcome:        "R7-3 guard: the allowlist entry \"10.0.0.5/8\" reads as a single host but net.ParseCIDR MASKS it to 10.0.0.0/8, so this peer -- 16.7M addresses away from the 10.0.0.5 the operator actually wrote -- is trusted and dictates its own identity through X-Forwarded-For (198.51.100.62); the widening is real, which is what makes the host-bits warning load-bearing rather than decorative",
		},

		// ---- F11: repeated header INSTANCES (separate lines on the wire),
		// the shape r.Header.Get truncates to its FIRST element. ----
		{
			name:           "xff_two_instances_rightmost_untrusted_wins",
			remoteAddr:     "10.0.0.1:443",
			xffInstances:   []string{"8.8.8.8", "203.0.113.9"},
			trustedProxies: "10.0.0.1",
			want:           "203.0.113.9",
			outcome:        "F11 guard: TWO separate X-Forwarded-For header instances are joined per RFC 7230 3.2.2 before the right-to-left walk, so the RIGHTMOST untrusted entry (the one the appending proxy contributed) wins -- reading only Header.Get returns the attacker's forged FIRST instance (8.8.8.8) and re-opens the leftmost-selection bypass the walk exists to close",
		},
		{
			name:           "xff_two_instances_non_ip_first_still_resolves",
			remoteAddr:     "10.0.0.1:443",
			xffInstances:   []string{"not-an-ip", "203.0.113.9"},
			trustedProxies: "10.0.0.1",
			want:           "203.0.113.9",
			outcome:        "F11 guard: a NON-IP first instance must not discard the real client -- with Header.Get the walk saw only \"not-an-ip\", stopped, and the caller collapsed onto the proxy (10.0.0.1); joining every instance lets the walk reach the real rightmost entry",
		},
		{
			name:           "xri_two_instances_refused_falls_back_to_peer",
			remoteAddr:     "10.0.0.1:443",
			xriInstances:   []string{"8.8.8.8", "203.0.113.9"},
			trustedProxies: "10.0.0.1",
			want:           "10.0.0.1",
			outcome:        "F11 guard: X-Real-IP is SET (not appended) by the immediate proxy, so two instances mean that invariant is ALREADY violated and nothing in the message says which line the proxy wrote -- every instance is refused and the identity falls back to the peer, never the attacker's forged FIRST instance (8.8.8.8) that Header.Get returned",
		},
		{
			name:           "xri_two_identical_instances_still_refused",
			remoteAddr:     "10.0.0.1:443",
			xriInstances:   []string{"8.8.8.8", "8.8.8.8"},
			trustedProxies: "10.0.0.1",
			want:           "10.0.0.1",
			outcome:        "R7 nit guard: TWO BYTE-IDENTICAL X-Real-IP instances are refused as well, deliberately and in the SAFE direction -- the refusal's stated reason (nothing says which hop wrote which line) is unanswerable here too, and \"the values agree\" describes the duplicate's SPELLING rather than its provenance, since a caller able to inject a second instance can equally inject a matching one; the rule stays uniform with no attacker-selectable branch and the caller collapses onto the peer (10.0.0.1) instead of adopting 8.8.8.8",
		},

		// ---- D1: one caller, several spellings. The resolved identity is
		// canonicalised so it cannot fork one caller into two audit-trail
		// identities / two rate-limit keys. ----
		{
			name:           "xff_v4_mapped_ipv6_entry_canonicalised",
			remoteAddr:     "10.0.0.1:443",
			xff:            "::ffff:8.8.8.8",
			trustedProxies: "10.0.0.1",
			want:           "8.8.8.8",
			outcome:        "D1 guard: an IPv4-mapped IPv6 X-Forwarded-For entry resolves to the SAME identity as its plain dotted-quad spelling -- returning the raw literal \"::ffff:8.8.8.8\" forks one caller into two identities, and also disagrees with the identity that caller gets on a DIRECT connection (Go's net stack canonicalises there)",
		},
		{
			name:           "xff_uppercase_ipv6_entry_canonicalised",
			remoteAddr:     "10.0.0.1:443",
			xff:            "2001:DB8::1",
			trustedProxies: "10.0.0.1",
			want:           "2001:db8::1",
			outcome:        "D1 guard: an upper-case-hex IPv6 X-Forwarded-For entry resolves to the canonical lower-case RFC 5952 form -- returning the raw literal forks \"2001:DB8::1\" and \"2001:db8::1\" into two identities for one caller",
		},
	}
}

// TestHXC298ClientIPConcurrent is the §11.4.85 concurrent-contention case:
// N goroutines simultaneously resolving distinct callers (some trusted,
// some not, some forging headers) must each get their own correct,
// uncorrupted identity — no shared mutable state in getClientIP /
// clientip.Resolve may leak one caller's resolution into another's.
func TestHXC298ClientIPConcurrent(t *testing.T) {
	t.Setenv(clientip.TrustedProxiesEnvVar, "172.20.0.0/16")

	api := &EnterpriseAPI{}

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
				// peer address (port stripped) used.
				peer := fmt.Sprintf("203.0.113.%d:%d", i%256, 1000+i)
				req = httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = peer
				req.Header.Set("X-Forwarded-For", "6.6.6.6")
				want = peer[:len(peer)-len(fmt.Sprintf(":%d", 1000+i))]
			}

			got := api.getClientIP(req)
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
