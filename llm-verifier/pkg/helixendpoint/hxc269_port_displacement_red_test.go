package helixendpoint

import (
	"net/url"
	"os"
	"strconv"
	"testing"
)

// TestHXC269PortDisplacementRED — §11.4.115 RED-baseline-on-the-broken-artifact
// for HXC-269, registered in the §11.4.135 standing regression-guard suite.
//
// # The defect
//
// BaseURL composed its result as "http://" + net.JoinHostPort(host, port), and
// normalizeHost accepted any host that carried no colon. Four RFC 3986 gen-delims
// pass that acceptance while ENDING the host component, so the port appended after
// one of them lands in the wrong component. Measured on the pre-fix artifact at
// port 7061 — these are the redURL literals below:
//
//	"agent.internal?x"   -> http://agent.internal?x:7061   query    "x:7061", port EMPTY
//	"agent.internal#f"   -> http://agent.internal#f:7061   fragment "f:7061", port EMPTY
//	"agent.internal/p"   -> http://agent.internal/p:7061   path   "/p:7061", port EMPTY
//	"svcuser@gw.example" -> http://svcuser@gw.example:7061 port KEPT, host is "gw.example"
//
// The first three lose the port; the fourth keeps it and points at a DIFFERENT
// host. Every one parses cleanly, so nothing downstream can notice — the client
// connects to the default port, or to the wrong host, with no error anywhere.
//
// # Polarity switch (§11.4.115)
//
// RED_MODE=1 reproduces the defect on the pre-fix artifact and PASSes there.
// RED_MODE=0 (default) is the standing GREEN guard: it FAILs on the broken
// artifact and PASSes once malformed hosts are rejected.
//
// # Why the assertions are pinned to literals
//
// Every expected URL below is a test-local literal, never a value rendered by the
// code under test (not BaseURL, not DefaultHost, not JoinHostPort). A test that
// asks the code under test what it should have produced cannot fail.
//
// # Both mutation directions are covered
//
// Removing the fix (restoring the concatenation) FAILs the greenURL assertions on
// the four defect cases. The OPPOSITE mutation is covered too, which a
// port-did-not-move test alone would miss:
//
//   - a "fix" that rejects EVERYTHING fails the five ordinary controls, whose
//     greenURL is their own authority and not the placeholder;
//   - a "fix" that drops or zeroes the port fails assertPortOnAuthority, which
//     requires the exact port to be readable from the authority in every case,
//     including the rejected ones.
//
// So an implementation that produces no usable address is not a passing one.
func TestHXC269PortDisplacementRED(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}
	if redMode != "0" && redMode != "1" {
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}

	const port = 7061

	// placeholderURL is the literal a rejected host must fall back to. Written out
	// rather than built from DefaultHost so this pins the observable behaviour: if
	// the placeholder ever changes, that is a contract change a reader must see
	// here, not something the test silently absorbs.
	const placeholderURL = "http://localhost:7061"

	cases := []struct {
		name string
		host string
		// redURL is what the pre-fix artifact produced (measured, see above).
		redURL string
		// greenURL is what the fixed artifact must produce.
		greenURL string
		// defect marks the four shapes HXC-269 is about. Controls have
		// redURL == greenURL: they were correct before and must stay correct.
		defect bool
	}{
		// ---- the four silent-misdirection shapes ----
		{
			name:     "host carrying a query loses the port into the query",
			host:     "agent.internal?x",
			redURL:   "http://agent.internal?x:7061",
			greenURL: placeholderURL,
			defect:   true,
		},
		{
			name:     "host carrying a fragment loses the port into the fragment",
			host:     "agent.internal#f",
			redURL:   "http://agent.internal#f:7061",
			greenURL: placeholderURL,
			defect:   true,
		},
		{
			name:     "host carrying a path loses the port into the path",
			host:     "agent.internal/p",
			redURL:   "http://agent.internal/p:7061",
			greenURL: placeholderURL,
			defect:   true,
		},
		{
			// The fourth shape keeps the port and redirects the HOST, which is
			// worse than losing the port: the address is well-formed and points
			// somewhere else. It is also the credential shape — an operator who
			// pastes a token-bearing authority puts the token in the host.
			name:     "host carrying userinfo silently redirects to a different host",
			host:     "svcuser@gw.example",
			redURL:   "http://svcuser@gw.example:7061",
			greenURL: placeholderURL,
			defect:   true,
		},

		// ---- ordinary controls: the usefulness half of the pair ----
		//
		// These are what stop "reject everything" from passing. Each must resolve
		// to its OWN authority, never the placeholder.
		{
			name:     "ordinary hostname is unchanged",
			host:     "agent.internal",
			redURL:   "http://agent.internal:7061",
			greenURL: "http://agent.internal:7061",
		},
		{
			name:     "IPv4 literal is unchanged",
			host:     "10.0.0.7",
			redURL:   "http://10.0.0.7:7061",
			greenURL: "http://10.0.0.7:7061",
		},
		{
			name:     "IPv6 literal is still bracketed",
			host:     "::1",
			redURL:   "http://[::1]:7061",
			greenURL: "http://[::1]:7061",
		},
		{
			name:     "already-bracketed IPv6 literal is not double-bracketed",
			host:     "[::1]",
			redURL:   "http://[::1]:7061",
			greenURL: "http://[::1]:7061",
		},
		{
			// The zone ID carries a "%", which is NOT in the rejected set — the
			// fix must not over-reach into the RFC 6874 zone path.
			name:     "IPv6 zone ID still percent-encoded per RFC 6874",
			host:     "fe80::1%eth0",
			redURL:   "http://[fe80::1%25eth0]:7061",
			greenURL: "http://[fe80::1%25eth0]:7061",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BaseURL(tc.host, port)

			switch {
			case redMode == "1" && tc.defect:
				if got != tc.redURL {
					t.Fatalf("RED_MODE=1: expected the pre-fix artifact to produce the displaced address\n"+
						"  host     %q\n  want     %q\n  got      %q\n"+
						"Run RED_MODE=1 against the pre-fix build to see this PASS.",
						tc.host, tc.redURL, got)
				}
				assertPortDisplaced(t, got, port)
				t.Logf("RED_MODE=1 PASS: defect reproduced — %q -> %q", tc.host, got)

			case redMode == "1" && !tc.defect:
				// Controls must hold in RED mode too, or the instrument is invalid:
				// a build that mangles ordinary hosts would let the defect cases
				// "reproduce" for the wrong reason.
				if got != tc.greenURL {
					t.Fatalf("RED_MODE=1 instrument invalid: control case changed\n"+
						"  host %q\n  want %q\n  got  %q", tc.host, tc.greenURL, got)
				}

			default: // RED_MODE=0 — the standing guard
				if got != tc.greenURL {
					t.Fatalf("RED_MODE=0: HXC-269 not fixed for this shape\n"+
						"  host     %q\n  want     %q\n  got      %q\n"+
						"A host carrying \"/\", \"?\", \"#\" or \"@\" must be rejected so the port "+
						"stays on the authority; an ordinary host must be used as given.",
						tc.host, tc.greenURL, got)
				}
				// The invariant the whole item is about, asserted for EVERY case
				// including the rejected ones: the port is readable from the
				// authority. This is what a drop-the-port "fix" fails.
				assertPortOnAuthority(t, got, port)
			}
		})
	}
}

// assertPortOnAuthority is the contract HXC-269 exists to restore: whatever host
// was supplied, the port argument ends up in the authority where a standard
// parser finds it, and no component was displaced.
func assertPortOnAuthority(t *testing.T, rawURL string, wantPort int) {
	t.Helper()

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("result is not a parseable URL: %q: %v", rawURL, err)
	}
	if got, want := u.Port(), strconv.Itoa(wantPort); got != want {
		t.Errorf("port is not on the authority: want %q, got %q (from %q)", want, got, rawURL)
	}
	if u.Hostname() == "" {
		t.Errorf("URL has no host at all: %q", rawURL)
	}
	// The three displacement sinks plus userinfo. BaseURL returns
	// "scheme://authority" and callers JoinPath onto it, so any of these being
	// non-empty means a host smuggled a component through.
	if u.RawQuery != "" {
		t.Errorf("port or host displaced into the query %q: %q", u.RawQuery, rawURL)
	}
	if u.Fragment != "" {
		t.Errorf("port or host displaced into the fragment %q: %q", u.Fragment, rawURL)
	}
	if u.Path != "" {
		t.Errorf("port or host displaced into the path %q: %q", u.Path, rawURL)
	}
	if u.User != nil {
		t.Errorf("host smuggled userinfo %q into the authority: %q", u.User.String(), rawURL)
	}
}

// assertPortDisplaced is the RED-mode counterpart: it confirms the reproduction
// is the DEFECT and not merely a different string, so RED_MODE=1 cannot pass on a
// build that is broken in some unrelated way.
func assertPortDisplaced(t *testing.T, rawURL string, wantPort int) {
	t.Helper()

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("RED_MODE=1: expected a parseable-but-wrong URL, got a parse error: %q: %v", rawURL, err)
	}
	// Either the port is not on the authority (the first three shapes) or it is
	// but the host is not the one that was asked for (the userinfo shape). Both
	// are the silent misdirection; neither is acceptable output.
	portLost := u.Port() != strconv.Itoa(wantPort)
	displaced := u.RawQuery != "" || u.Fragment != "" || u.Path != "" || u.User != nil
	if !portLost && !displaced {
		t.Fatalf("RED_MODE=1: %q is a correct address — the defect did not reproduce", rawURL)
	}
}
