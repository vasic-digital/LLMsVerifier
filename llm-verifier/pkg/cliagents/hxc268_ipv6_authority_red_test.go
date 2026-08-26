package cliagents

import (
	"context"
	"encoding/json"
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
// HXC-250 routed generator.go's endpoint sites through pkg/helixendpoint but left
// SIX per-agent sites on a raw fmt.Sprintf — crush.go, kilocode.go, opencode.go,
// helixcode.go, additional_agents.go (the generic generator behind ~44 agents),
// and formatters_config.go. Each built "http://%s:%d/v1" (formatters:
// "/v1/format") straight from HelixAgentHost/HelixAgentPort. That concatenation is
// not a valid URL authority for an IPv6 host: RFC 3986 §3.2.2 requires the literal
// to be bracketed. Measured on the pre-fix artifact at port 7061:
//
//	"fe80::1"      -> http://fe80::1:7061/v1
//	"::1"          -> http://::1:7061/v1
//	"fe80::1%eth0" -> http://fe80::1%eth0:7061/v1   url.Parse: invalid URL escape "%et"
//	""             -> http://:7061/v1               PARSES, but with NO host at all
//
// That asymmetry is exactly what this item closes: the same generated config
// carried correctly-bracketed MCP/extension URLs (HXC-250, via helixendpoint)
// alongside malformed provider base URLs (these six), so a config could be
// half-usable with nothing reporting a problem.
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
// Under this module's current language version Go itself papers over two of the
// four shapes — but these configs are consumed by NON-Go CLI agents (Node, Python,
// Rust) whose parsers follow the RFC, the zone shape is broken TODAY in both
// modes, and a `go` directive bump to 1.26 breaks the remaining two for Go
// consumers too.
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
// artifact and PASSes once all six sites route through pkg/helixendpoint.
//
// GREEN is UNIVERSAL (every emitted URL must be correct); RED is EXISTENTIAL (the
// defect must appear somewhere in the emitted set). RED cannot be universal here
// precisely because of the asymmetry above — the HXC-250 URLs stay correct on the
// pre-fix HXC-268 artifact.
//
// # Why the assertions are pinned to literals
//
// Every expected authority below is a test-local literal, never a value rendered
// by the code under test. A test that asks the code under test what it should
// have produced cannot fail.
//
// # Both mutation directions are covered
//
// Restoring the raw Sprintf FAILs the GREEN defect cases. A "fix" that brackets
// unconditionally renders IPv4 as "[10.0.0.7]" and a hostname as
// "[agent.internal]" — both fail the pinned authority AND are rejected by
// url.Parse — and a "fix" that collapses everything to the placeholder fails the
// five non-empty host cases.
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
			all := hxc268GeneratedURLs(t, tc.host, port)

			if redMode == "1" && tc.defect {
				hxc268AssertDefectPresent(t, tc, all)
				return
			}
			for _, got := range all {
				hxc268AssertGreen(t, tc, got, port)
			}
		})
	}
}

// hxc268GeneratedURLs drives the real generator entry points for host:port and
// returns every "http://" URL they emit.
//
// It covers all six HXC-268 sites: GenerateAll reaches the four per-agent
// generators plus the generic one behind every remaining agent, and
// DefaultFormattersConfig is asserted directly so the formatters site is pinned
// by name rather than only reached incidentally.
func hxc268GeneratedURLs(t *testing.T, host string, port int) []string {
	t.Helper()

	cfg := &GeneratorConfig{HelixAgentHost: host, HelixAgentPort: port}
	results, err := NewUnifiedGenerator(cfg).GenerateAll(context.Background())
	if err != nil {
		t.Fatalf("instrument invalid: GenerateAll failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("instrument invalid: GenerateAll produced no results to inspect")
	}

	var all []string
	configured := 0
	for _, r := range results {
		if r.Config == nil {
			continue
		}
		configured++

		// Marshal to JSON and walk the generic tree: the per-agent config types
		// are unrelated structs, and this reaches every string field in all of
		// them without the guard having to know any of their shapes.
		raw, err := json.Marshal(r.Config)
		if err != nil {
			t.Fatalf("instrument invalid: could not marshal %s config: %v", r.AgentType, err)
		}
		var tree interface{}
		if err := json.Unmarshal(raw, &tree); err != nil {
			t.Fatalf("instrument invalid: could not walk %s config: %v", r.AgentType, err)
		}
		all = append(all, hxc268CollectHTTPURLs(tree)...)
	}
	if configured == 0 {
		t.Fatal("instrument invalid: no agent produced a config to inspect")
	}

	// The formatters site, pinned explicitly rather than only incidentally.
	svc := DefaultFormattersConfig(host, port).ServiceURL
	if svc == "" {
		t.Fatal("instrument invalid: DefaultFormattersConfig emitted no ServiceURL")
	}
	all = append(all, svc)

	if len(all) == 0 {
		t.Fatal("instrument invalid: no URL was inspected")
	}
	t.Logf("inspected %d generated URL(s) across %d agent configs for host %q",
		len(all), configured, host)
	return all
}

// hxc268HostCase is one operator-supplied host form and what every generated URL
// must look like for it, before and after the fix.
type hxc268HostCase struct {
	name string
	host string

	// wantURLPrefix is the exact scheme+authority every generated URL must carry
	// after the fix. Written out rather than built from helixendpoint so this pins
	// the observable contract: if the authority ever changes, that is a contract
	// change a reader must see here, not something the test silently absorbs.
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
			// The shape that was broken in BOTH parser modes pre-fix.
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

// hxc268CollectHTTPURLs walks a marshalled config tree and returns every
// "http://" string in it, at any nesting depth.
//
// Scoped to "http://" deliberately: the generated configs also carry vendor
// "https://" literals this package does not compose — schema URLs
// ("https://opencode.ai/config.json", "https://charm.land/crush.json") and
// third-party MCP endpoints ("https://mcp.context7.com/mcp") — which carry no
// host:port of ours. Every "http://" string, by contrast, IS one of the endpoints
// this item is about (measured: 12 distinct, all on the configured host:port), so
// widening the net would only add noise while narrowing it would let a site
// escape the guard.
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
		}
	}
	walk(v)
	return out
}
