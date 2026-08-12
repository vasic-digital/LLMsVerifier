package capabilities

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
)

// TestHXC268IPv6AuthorityRED — §11.4.115 RED-baseline-on-the-broken-artifact for
// HXC-268, registered in the §11.4.135 standing regression-guard suite.
//
// # The defect
//
// Every generated CLI-agent config in this package composed its endpoint with
// fmt.Sprintf("http://%s:%d…", cg.baseHost, cg.basePort) — 15 sites across three
// shapes ("http://%s:%d", "http://%s:%d/v1", "http://%s:%d/v1/messages"). That
// concatenation is not a valid URL authority for an IPv6 host: RFC 3986 §3.2.2
// requires the literal to be bracketed, so a port appended to a bare literal is
// unreadable. Measured on the pre-fix artifact at port 7061:
//
//	"fe80::1"      -> http://fe80::1:7061/v1
//	"::1"          -> http://::1:7061/v1
//	"fe80::1%eth0" -> http://fe80::1%eth0/v1        url.Parse: invalid URL escape "%et"
//	""             -> http://:7061/v1               PARSES, but with NO host at all
//
// # Why the RED assertion is on the STRING, not on url.Parse
//
// Go's own acceptance of these is VERSION-GATED, so a parse-based reproduction
// would be an invalid instrument. MEASURED in this module, both modes:
//
//	                              urlstrictcolons=0        urlstrictcolons=1
//	                              (this module's go 1.25.3) (Go >= 1.26 language ver)
//	http://fe80::1:7061/v1        accepted, host recovered  REJECTED: invalid port
//	http://::1:7061/v1            accepted, host recovered  REJECTED: invalid port
//	http://fe80::1%eth0:7061/v1   REJECTED: invalid escape  REJECTED: invalid port
//	http://:7061/v1               accepted, NO host         accepted, NO host
//
// So under this module's current language version Go itself papers over two of
// the four shapes. That does NOT make them correct: the generated configs are
// consumed by NON-Go CLI agents (Node, Python, Rust) whose parsers follow the RFC,
// the zone shape is broken TODAY in both modes, and the moment this module's `go`
// directive reaches 1.26 the remaining two break for Go consumers too. Fixing this
// pre-empts that.
//
// The RED assertion therefore keys on the emitted STRING carrying the UNBRACKETED
// authority — true in every mode and every Go version — rather than on a parse
// failure that only some configurations produce. GREEN keys on the bracketed
// authority plus a parse round-trip, and MEASURED above, every post-fix URL parses
// correctly under BOTH settings.
//
// # Polarity switch (§11.4.115)
//
// RED_MODE=1 reproduces the defect on the pre-fix artifact and PASSes there.
// RED_MODE=0 (default) is the standing GREEN guard: it FAILs on the broken
// artifact and PASSes once every site routes through pkg/helixendpoint.
//
// The two modes are deliberately asymmetric in strength:
//
//   - GREEN is UNIVERSAL — EVERY emitted URL must be correct. That is the
//     standing contract, and it is what a regression at any single site trips.
//   - RED is EXISTENTIAL — the defect must appear SOMEWHERE in the emitted set.
//     It cannot be universal, because a generated config may also carry URLs
//     built by already-fixed code (the HXC-250 endpoint-injection sites), which
//     stay correct on the pre-fix HXC-268 artifact. Demanding that those be
//     broken too would make RED unsatisfiable on the very artifact it exists to
//     characterise.
//
// # Why the assertions are pinned to literals
//
// Every expected authority below is a test-local literal, never a value rendered
// by the code under test (not helixendpoint.BaseURL, not DefaultHost, not
// JoinHostPort). A test that asks the code under test what it should have
// produced cannot fail.
//
// # Both mutation directions are covered
//
// Restoring the raw Sprintf FAILs the GREEN defect cases. The OPPOSITE mutation
// is covered too, which an IPv6-only test would miss:
//
//   - a "fix" that brackets UNCONDITIONALLY renders IPv4 as "[10.0.0.7]" and a
//     hostname as "[agent.internal]"; both fail wantURLPrefix, and both are
//     rejected outright by url.Parse ("invalid IP-literal");
//   - a "fix" that rejects everything into the placeholder renders every case as
//     "localhost:7061", failing wantURLPrefix on all five non-empty hosts.
//
// So an implementation that mangles ordinary hosts is not a passing one.
func TestHXC268IPv6AuthorityRED(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}
	if redMode != "0" && redMode != "1" {
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}

	const port = 7061

	for _, tc := range hxc268HostCases() {
		t.Run(tc.name, func(t *testing.T) {
			gen := NewConfigGenerator(tc.host, port)

			agents := GetAllCLIAgents()
			if len(agents) == 0 {
				t.Fatal("instrument invalid: GetAllCLIAgents() returned nothing to drive")
			}

			var all []string
			for _, agent := range agents {
				cfg, err := gen.GenerateForAgent(agent, nil)
				if err != nil {
					t.Fatalf("instrument invalid: GenerateForAgent(%q) failed: %v", agent, err)
				}

				urls := hxc268CollectHTTPURLs(cfg.Content)
				if len(urls) == 0 && !hxc268EndpointlessAgents[agent] {
					t.Errorf("agent %q emitted no http:// URL — either the site this guard "+
						"exists to cover stopped being exercised, or a new endpoint-less agent "+
						"needs adding to hxc268EndpointlessAgents with the reason measured", agent)
				}
				all = append(all, urls...)
			}

			if len(all) == 0 {
				t.Fatal("instrument invalid: no URL was inspected for any agent")
			}
			t.Logf("inspected %d generated URL(s) across %d agents for host %q",
				len(all), len(agents), tc.host)

			if redMode == "1" && tc.defect {
				hxc268AssertDefectPresent(t, tc, all)
				return
			}
			// RED_MODE=1 controls fall through to the same GREEN assertions:
			// a build that mangled ordinary hosts would let the defect cases
			// "reproduce" for the wrong reason, so the instrument is only valid
			// while the controls hold in both modes.
			for _, got := range all {
				hxc268AssertGreen(t, tc, got, port)
			}
		})
	}
}

// hxc268EndpointlessAgents are the agents that compose NO host:port endpoint at
// all, so there is nothing for this guard to inspect and an empty result is
// correct rather than a regression.
//
// MEASURED against the generators, not assumed:
//
//   - "deepseekcli" — generateDeepSeekCLIConfig emits only environment variables
//     (DEEPSEEK_USE_LOCAL / DEEPSEEK_API_KEY / DEEPSEEK_MODEL); the CLI resolves
//     its own endpoint, so this package never builds one.
//   - "gptengineer" — generateGPTEngineerConfig emits only run/paths/caching keys;
//     it carries no endpoint field of any kind.
//
// The list is deliberately an explicit allowlist rather than a blanket "skip
// agents with no URL": that blanket form would silently absorb a regression in
// which a real endpoint stopped being emitted, which is precisely the failure
// this guard is meant to catch.
var hxc268EndpointlessAgents = map[string]bool{
	"deepseekcli": true,
	"gptengineer": true,
}

// hxc268HostCase is one operator-supplied host form and what every generated URL
// must look like for it, before and after the fix.
type hxc268HostCase struct {
	name string
	host string

	// wantURLPrefix is the exact scheme+authority every generated URL must carry
	// after the fix. Written out rather than built from helixendpoint or
	// DefaultHost so this pins the observable contract: if the authority ever
	// changes, that is a contract change a reader must see here, not something
	// the test silently absorbs.
	wantURLPrefix string

	// wantHostname is what url.Parse must recover from that authority. For the
	// zone-ID case this is the DECODED form: the URL carries RFC 6874 "%25" on
	// the wire and url.Parse hands back the raw zone.
	wantHostname string

	// defect marks the shapes HXC-268 is about. Controls have defect=false: they
	// were correct before and must stay correct.
	defect bool

	// redURLPrefix is the exact scheme+authority the PRE-FIX artifact produced —
	// the unbracketed form. Set for every defect case; unused for controls, whose
	// pre-fix and post-fix output are the same.
	redURLPrefix string
}

func hxc268HostCases() []hxc268HostCase {
	return []hxc268HostCase{
		// ---- the IPv6 shapes HXC-268 is about ----
		{
			name:          "link-local IPv6 literal must be bracketed",
			host:          "fe80::1",
			wantURLPrefix: "http://[fe80::1]:7061",
			wantHostname:  "fe80::1",
			defect:        true,
			redURLPrefix:  "http://fe80::1:7061",
		},
		{
			name:          "loopback IPv6 literal must be bracketed",
			host:          "::1",
			wantURLPrefix: "http://[::1]:7061",
			wantHostname:  "::1",
			defect:        true,
			redURLPrefix:  "http://::1:7061",
		},
		{
			// The zone ID must survive as RFC 6874 "%25" on the wire. A fix that
			// dropped the zone, or left a bare "%eth0" that url.Parse reads as a
			// malformed escape, fails here. This is the shape that was broken in
			// BOTH parser modes pre-fix.
			name:          "IPv6 zone ID must be percent-encoded per RFC 6874",
			host:          "fe80::1%eth0",
			wantURLPrefix: "http://[fe80::1%25eth0]:7061",
			wantHostname:  "fe80::1%eth0",
			defect:        true,
			redURLPrefix:  "http://fe80::1%eth0:7061",
		},
		{
			// The silent member of the class: it parsed cleanly and pointed at
			// nothing. Post-fix it resolves to the documented placeholder.
			name:          "empty host must not yield a host-less authority",
			host:          "",
			wantURLPrefix: "http://localhost:7061",
			wantHostname:  "localhost",
			defect:        true,
			redURLPrefix:  "http://:7061",
		},

		// ---- controls: the usefulness half of the pair ----
		//
		// These are what stop "bracket everything" and "reject everything" from
		// passing. Each must resolve to its OWN authority, unbracketed.
		{
			name:          "already-bracketed IPv6 is not double-bracketed",
			host:          "[::1]",
			wantURLPrefix: "http://[::1]:7061",
			wantHostname:  "::1",
		},
		{
			name:          "IPv4 literal is left unbracketed",
			host:          "10.0.0.7",
			wantURLPrefix: "http://10.0.0.7:7061",
			wantHostname:  "10.0.0.7",
		},
		{
			name:          "ordinary hostname is left unbracketed",
			host:          "agent.internal",
			wantURLPrefix: "http://agent.internal:7061",
			wantHostname:  "agent.internal",
		},
	}
}

// hxc268AssertGreen is the contract HXC-268 exists to establish: whatever host
// was supplied, the generated URL carries the expected authority verbatim AND
// survives a round-trip through url.Parse with its host and port intact.
func hxc268AssertGreen(t *testing.T, tc hxc268HostCase, got string, wantPort int) {
	t.Helper()

	// Full-string authority match against the test-local literal. This is what
	// catches a mutation that still PARSES but changed the authority — an
	// unconditional bracket, or a fallback that swallowed the real host.
	if got != tc.wantURLPrefix && !strings.HasPrefix(got, tc.wantURLPrefix+"/") {
		t.Errorf("HXC-268 not fixed for host %q\n  want authority %q\n  got URL       %q\n"+
			"An IPv6 literal must be bracketed per RFC 3986 §3.2.2; an IPv4 literal and a "+
			"hostname must NOT be.", tc.host, tc.wantURLPrefix, got)
		return
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Errorf("generated URL is not parseable: %q: %v", got, err)
		return
	}
	if u.Hostname() != tc.wantHostname {
		t.Errorf("host did not survive the round-trip\n  want %q\n  got  %q (from %q)",
			tc.wantHostname, u.Hostname(), got)
	}
	if want := fmt.Sprintf("%d", wantPort); u.Port() != want {
		t.Errorf("port is not on the authority: want %q, got %q (from %q)", want, u.Port(), got)
	}
}

// hxc268AssertDefectPresent is the RED-mode counterpart: it confirms the
// reproduction is the DEFECT and not merely a different string, so RED_MODE=1
// cannot pass on a build that is broken in some unrelated way.
func hxc268AssertDefectPresent(t *testing.T, tc hxc268HostCase, urls []string) {
	t.Helper()

	if tc.redURLPrefix == "" {
		t.Fatalf("instrument invalid: defect case %q carries no redURLPrefix to reproduce", tc.name)
	}

	for _, got := range urls {
		if got != tc.redURLPrefix && !strings.HasPrefix(got, tc.redURLPrefix+"/") {
			continue
		}
		// Found the pre-fix authority. Report what the CURRENT parser makes of
		// it too — informational only, because that verdict is version-gated
		// (see the urlstrictcolons table in the test doc) and must never be the
		// thing the reproduction depends on.
		if _, err := url.Parse(got); err != nil {
			t.Logf("RED_MODE=1 PASS: defect reproduced for host %q — %q (unparseable here: %v)",
				tc.host, got, err)
		} else {
			t.Logf("RED_MODE=1 PASS: defect reproduced for host %q — %q (this parser accepts the "+
				"unbracketed authority; RFC-conformant parsers and urlstrictcolons=1 do not)",
				tc.host, got)
		}
		return
	}

	t.Fatalf("RED_MODE=1: the defect did NOT reproduce for host %q — none of the %d emitted URLs "+
		"carried the pre-fix authority %q. Run RED_MODE=1 against the pre-fix build to see this PASS.",
		tc.host, len(urls), tc.redURLPrefix)
}

// hxc268CollectHTTPURLs walks a generated config's Content and returns every
// "http://" string in it, at any nesting depth.
//
// Scoped to "http://" deliberately: the generated configs also carry vendor
// "https://" schema URLs ("https://opencode.ai/config.json") which are literals
// this package does not compose and which carry no host:port of ours. Every
// "http://" string, by contrast, IS one of the endpoints this item is about, so
// widening the net here would only add noise while narrowing it further would
// let a site escape the guard.
func hxc268CollectHTTPURLs(v interface{}) []string {
	var out []string
	var walk func(interface{})
	walk = func(node interface{}) {
		switch n := node.(type) {
		case string:
			if strings.HasPrefix(n, "http://") {
				out = append(out, n)
			}
		case map[string]interface{}:
			for _, child := range n {
				walk(child)
			}
		case []interface{}:
			for _, child := range n {
				walk(child)
			}
		case []string:
			for _, child := range n {
				walk(child)
			}
		}
	}
	walk(v)
	return out
}
