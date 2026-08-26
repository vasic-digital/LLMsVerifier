package auth

import (
	"net"
	"os"
	"testing"
)

// TestHXC268DialAddressRED — §11.4.115 RED-baseline-on-the-broken-artifact for the
// LDAP half of HXC-268, registered in the §11.4.135 standing regression-guard suite.
//
// # The defect
//
// LDAPManager.connect composed its dial address with
// fmt.Sprintf("%s:%d", lm.config.Host, lm.config.Port). For an IPv6 host that is
// not a usable address: net.SplitHostPort — the parse every dialer performs —
// rejects it. Measured on the pre-fix artifact at port 389:
//
//	"fe80::1"      -> fe80::1:389        SplitHostPort: too many colons in address
//	"::1"          -> ::1:389            SplitHostPort: too many colons in address
//	"fe80::1%eth0" -> fe80::1%eth0:389   SplitHostPort: too many colons in address
//
// So an operator pointing this at an IPv6 directory server got a connection error
// with no indication that the address itself was malformed rather than the server
// unreachable.
//
// # Provenance of those measured values (§11.4.6)
//
// Stated exactly, because "measured on the pre-fix artifact" would otherwise
// imply a git-level revert, and that is not what was run. The pre-fix artifact
// has no dialAddress() method at all — connect composed the address inline — so
// the reproduction was performed with the original
// fmt.Sprintf("%s:%d", lm.config.Host, lm.config.Port) expression restored
// INSIDE the extracted helper. Same operands, same verbs, same call site, so the
// composition under measurement is byte-identical to the pre-fix inline one;
// what the rows above measure is the pre-fix COMPOSITION, not a checkout that
// predates the extraction.
//
// # Polarity switch (§11.4.115)
//
// RED_MODE=1 reproduces the defect on the pre-fix artifact and PASSes there.
// RED_MODE=0 (default) is the standing GREEN guard.
//
// # Both mutation directions are covered
//
// Restoring the raw Sprintf FAILs the three defect cases. The opposite mutation is
// covered by the four controls: a "fix" that brackets unconditionally turns
// "10.0.0.7" into "[10.0.0.7]:389" and an already-bracketed "[::1]" into
// "[[::1]]:389", both of which fail their pinned literal AND the SplitHostPort
// round-trip. The empty-host control additionally pins that this address is NOT
// given the HelixAgent placeholder — a directory client must fail loudly on a bad
// host, never be silently redirected.
func TestHXC268DialAddressRED(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}
	if redMode != "0" && redMode != "1" {
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}

	const port = 389

	cases := []struct {
		name string
		host string
		// red is what the pre-fix artifact produced (measured).
		red string
		// green is what the fixed artifact must produce. Test-local literals,
		// never values rendered by the code under test.
		green string
		// wantHost is what net.SplitHostPort must recover post-fix.
		wantHost string
		// defect marks the shapes HXC-268 is about; controls have red == green.
		defect bool
	}{
		{
			name:     "link-local IPv6 literal must be bracketed",
			host:     "fe80::1",
			red:      "fe80::1:389",
			green:    "[fe80::1]:389",
			wantHost: "fe80::1",
			defect:   true,
		},
		{
			name:     "loopback IPv6 literal must be bracketed",
			host:     "::1",
			red:      "::1:389",
			green:    "[::1]:389",
			wantHost: "::1",
			defect:   true,
		},
		{
			// The zone stays RAW here, unlike the URL class where RFC 6874
			// requires "%25": the net package resolves the literal zone.
			name:     "IPv6 zone ID must be bracketed and kept raw",
			host:     "fe80::1%eth0",
			red:      "fe80::1%eth0:389",
			green:    "[fe80::1%eth0]:389",
			wantHost: "fe80::1%eth0",
			defect:   true,
		},

		// ---- controls: correct before, must stay correct ----
		{
			name:     "already-bracketed IPv6 is not double-bracketed",
			host:     "[::1]",
			red:      "[::1]:389",
			green:    "[::1]:389",
			wantHost: "::1",
		},
		{
			name:     "IPv4 literal is left unbracketed",
			host:     "10.0.0.7",
			red:      "10.0.0.7:389",
			green:    "10.0.0.7:389",
			wantHost: "10.0.0.7",
		},
		{
			name:     "ordinary hostname is left unbracketed",
			host:     "ldap.internal",
			red:      "ldap.internal:389",
			green:    "ldap.internal:389",
			wantHost: "ldap.internal",
		},
		{
			// Pins the deliberate asymmetry with the URL class: a dial address
			// gets NO placeholder fallback.
			name:     "empty host is passed through, never given the HelixAgent placeholder",
			host:     "",
			red:      ":389",
			green:    ":389",
			wantHost: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lm := &LDAPManager{config: &LDAPConfig{Host: tc.host, Port: port}}
			got := lm.dialAddress()

			if redMode == "1" && tc.defect {
				if got != tc.red {
					t.Fatalf("RED_MODE=1: expected the pre-fix artifact to produce the malformed "+
						"address\n  host %q\n  want %q\n  got  %q\n"+
						"Run RED_MODE=1 against the pre-fix build to see this PASS.",
						tc.host, tc.red, got)
				}
				if _, _, err := net.SplitHostPort(got); err == nil {
					t.Fatalf("RED_MODE=1: %q is a usable address — the defect did not reproduce", got)
				}
				t.Logf("RED_MODE=1 PASS: defect reproduced — %q -> %q", tc.host, got)
				return
			}

			if got != tc.green {
				t.Fatalf("RED_MODE=0: HXC-268 not fixed for this shape\n  host %q\n  want %q\n  got  %q\n"+
					"An IPv6 literal must be bracketed per RFC 3986 §3.2.2; an IPv4 literal, a "+
					"hostname and an already-bracketed literal must be left alone.",
					tc.host, tc.green, got)
			}
			h, p, err := net.SplitHostPort(got)
			if err != nil {
				t.Fatalf("dial address is not parseable by the net package: %q: %v", got, err)
			}
			if h != tc.wantHost {
				t.Errorf("host did not survive the round-trip: want %q, got %q (from %q)",
					tc.wantHost, h, got)
			}
			if p != "389" {
				t.Errorf("port did not survive the round-trip: want %q, got %q (from %q)", "389", p, got)
			}
		})
	}
}
