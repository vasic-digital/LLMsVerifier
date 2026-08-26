package helixendpoint

import (
	"net"
	"testing"
)

// TestHXC268DialAddressContract pins DialAddress, the chokepoint HXC-268 added so
// the module's net.Dial-style call sites (LDAP, SMTP) stop composing addresses
// with fmt.Sprintf("%s:%d").
//
// # No RED polarity here, deliberately
//
// The §11.4.115 polarity switch belongs where the defective composition actually
// existed — the call sites, which carry TestHXC268DialAddressRED in auth/,
// events/ and scoring/. DialAddress itself is NEW, so there is no pre-fix
// artifact on which a RED run could reproduce anything; a RED mode here would be
// asserting a property of code that never shipped, which is not a reproduction
// (§11.4.6). This is the contract test for the shared primitive; the guards are
// at the seams.
//
// # What the pinned values encode
//
// Every want below is a test-local literal, never a value rendered by the code
// under test. Three properties that distinguish DialAddress from BaseURL are
// asserted directly, because each would be a defect if the URL behaviour were
// borrowed:
//
//   - the ZONE stays RAW ("%eth0"), NOT RFC 6874 percent-encoded — the net
//     package wants the literal zone, and "%25eth0" would not resolve;
//   - an EMPTY host stays empty (":587"), with NO placeholder substitution — this
//     address belongs to a foreign service, and silently redirecting it to the
//     HelixAgent placeholder is exactly the failure the separation prevents;
//   - an ALREADY-BRACKETED host is not double-bracketed — net.JoinHostPort
//     brackets unconditionally, and "[[::1]]:587" is rejected by SplitHostPort.
//
// # Why every row also pins BaseURL
//
// The two builders share exactly one thing — the unbracket trim — and differ
// deliberately everywhere else. Pinning only DialAddress leaves that shared trim
// asserted from one side, so a change to it that happens to keep the dial side
// intact while breaking the URL side (or vice versa) passes. Asserting BOTH
// outputs per row makes each row a statement about the SHARED rule and about the
// two divergences it feeds: the zone row and the empty-host row read differently
// in the two columns, which is the separation the package doc claims, now
// mechanically pinned rather than described.
func TestHXC268DialAddressContract(t *testing.T) {
	const port = 587

	cases := []struct {
		name string
		host string
		// want is the exact address DialAddress must return.
		want string
		// wantBaseURL is the exact URL BaseURL must return for the SAME host, so
		// each row states both builders' readings of one input side by side.
		wantBaseURL string
		// wantHost is what net.SplitHostPort must recover from want.
		wantHost string
		// wantSplitErr marks a row whose composed address is deliberately NOT
		// parseable: the host was unusable, DialAddress passed it through rather
		// than inventing a reading, and the net package rejecting it IS the loud
		// failure that pass-through exists to produce.
		wantSplitErr bool
	}{
		{
			name:        "link-local IPv6 literal is bracketed",
			host:        "fe80::1",
			want:        "[fe80::1]:587",
			wantBaseURL: "http://[fe80::1]:587",
			wantHost:    "fe80::1",
		},
		{
			name:        "loopback IPv6 literal is bracketed",
			host:        "::1",
			want:        "[::1]:587",
			wantBaseURL: "http://[::1]:587",
			wantHost:    "::1",
		},
		{
			// The regression net.JoinHostPort would cause on its own: it brackets
			// whenever the host holds a colon, so an already-bracketed literal
			// becomes "[[::1]]:587", which SplitHostPort rejects outright
			// ("missing port in address"). Pre-fix, the raw Sprintf got this case
			// RIGHT — so a naive JoinHostPort swap would have traded one broken
			// host form for another.
			name:        "already-bracketed IPv6 literal is not double-bracketed",
			host:        "[::1]",
			want:        "[::1]:587",
			wantBaseURL: "http://[::1]:587",
			wantHost:    "::1",
		},
		{
			// The row that makes unbracket's "one layer only" rule FALSIFIABLE.
			//
			// Every other bracketed fixture strips to a bracket-free host, so a
			// mutant that stripped REPEATEDLY instead of once would agree with all
			// of them — and did: the repeat-strip mutation survived the whole
			// 8-package suite before this row existed. Two brackets is the
			// shortest input on which "once" and "until none remain" disagree.
			//
			// Neither column asserts that "[[::1]]" is a host anyone can mean; both
			// assert that the code does NOT invent a meaning for it. One strip
			// leaves "[::1]", which is not a parseable IP, so the two builders
			// refuse it in the two ways their contracts require:
			//
			//   - BaseURL REJECTS to the documented placeholder, because a URL must
			//     stay parseable and it owns the endpoint it is falling back to;
			//   - DialAddress PASSES THROUGH to "[[::1]]:587" — unparseable on
			//     purpose (wantSplitErr), because this address belongs to a foreign
			//     service and a loud dial-time failure is the honest outcome, where
			//     substituting a placeholder would point the client somewhere real
			//     and wrong.
			//
			// A repeated strip would yield "::1" and both columns would silently
			// start reporting a working loopback address the operator never wrote.
			name:         "double-bracketed host is stripped one layer only, never re-read as a literal",
			host:         "[[::1]]",
			want:         "[[::1]]:587",
			wantBaseURL:  "http://" + DefaultHost + ":587",
			wantSplitErr: true,
		},
		{
			// Isolates the LENGTH guard (len > 1). "[]" is the only input whose
			// length is exactly the boundary, so a guard shifted to len > 2 keeps
			// the pair instead of stripping it, and both builders then read a
			// two-character bracket pair as a hostname. Pristine, the strip empties
			// it, which BaseURL rejects to the placeholder and DialAddress passes
			// through as the portless ":587" a dial will refuse.
			name:        "empty bracket pair strips to nothing rather than becoming a host",
			host:        "[]",
			want:        ":587",
			wantBaseURL: "http://" + DefaultHost + ":587",
			wantHost:    "",
		},
		{
			// Isolates the OPENING-bracket conjunct. Ends with ']' but does not
			// begin with '[' — a literal whose opening bracket the operator
			// dropped. Without the conjunct the trim fires anyway and eats the
			// leading ':' , turning "::1]" into ":1": a different address, silently.
			// Pristine refuses to repair it, so the malformed host survives intact
			// into an address the net package rejects — the loud outcome.
			name:         "half-bracketed literal missing its opening bracket is not trimmed",
			host:         "::1]",
			want:         "[::1]]:587",
			wantBaseURL:  "http://" + DefaultHost + ":587",
			wantSplitErr: true,
		},
		{
			// Isolates the CLOSING-bracket conjunct, and is the sharpest of the
			// three: without it "[::1" strips to "::", which is a VALID address —
			// the all-zeros host. The mutant would not fail loudly, it would
			// quietly dial a real and entirely different destination, which is the
			// silent-misdirection class this package exists to prevent. Pristine
			// keeps the truncated literal, so BaseURL rejects it to the placeholder
			// and the dial address stays visibly malformed: net.SplitHostPort
			// refuses "[[::1]:587" with "unexpected '[' in address" (MEASURED —
			// the inner bracket survives the leading-'[' scan, which is not what
			// reading JoinHostPort alone predicts).
			name:         "truncated literal missing its closing bracket is not trimmed",
			host:         "[::1",
			want:         "[[::1]:587",
			wantBaseURL:  "http://" + DefaultHost + ":587",
			wantSplitErr: true,
		},
		{
			// RAW zone, not RFC 6874 "%25eth0" — see the doc comment above. The
			// wantBaseURL column is where the two builders visibly diverge.
			name:        "IPv6 zone ID is kept raw for the net package",
			host:        "fe80::1%eth0",
			want:        "[fe80::1%eth0]:587",
			wantBaseURL: "http://[fe80::1%25eth0]:587",
			wantHost:    "fe80::1%eth0",
		},
		{
			name:        "IPv4 literal is left unbracketed",
			host:        "10.0.0.7",
			want:        "10.0.0.7:587",
			wantBaseURL: "http://10.0.0.7:587",
			wantHost:    "10.0.0.7",
		},
		{
			name:        "ordinary hostname is left unbracketed",
			host:        "smtp.internal",
			want:        "smtp.internal:587",
			wantBaseURL: "http://smtp.internal:587",
			wantHost:    "smtp.internal",
		},
		{
			// NO placeholder substitution on the dial side — see the doc comment
			// above. The second divergence row: BaseURL DOES substitute here.
			name:        "empty host is passed through, never replaced by the placeholder",
			host:        "",
			want:        ":587",
			wantBaseURL: "http://" + DefaultHost + ":587",
			wantHost:    "",
		},
		{
			name:        "surrounding whitespace is trimmed",
			host:        "  ::1  ",
			want:        "[::1]:587",
			wantBaseURL: "http://[::1]:587",
			wantHost:    "::1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DialAddress(tc.host, port)
			if got != tc.want {
				t.Fatalf("DialAddress(%q, %d)\n  want %q\n  got  %q", tc.host, port, tc.want, got)
			}

			// The same host through the OTHER builder. Asserted per row so the
			// shared unbracket trim is pinned from both sides at once, and so the
			// rows where the two contracts deliberately disagree say so explicitly.
			if gotURL := BaseURL(tc.host, port); gotURL != tc.wantBaseURL {
				t.Errorf("BaseURL(%q, %d)\n  want %q\n  got  %q", tc.host, port, tc.wantBaseURL, gotURL)
			}

			if tc.wantSplitErr {
				// The pass-through rows: the address is unparseable BY DESIGN, and
				// that rejection is the loud failure. A mutant that made this host
				// parseable would be silently inventing an address, so the absence
				// of an error is the defect here.
				if h, p, err := net.SplitHostPort(got); err == nil {
					t.Errorf("net.SplitHostPort(%q) unexpectedly succeeded (host %q, port %q): "+
						"an unusable host must stay unusable so the dial fails loudly, "+
						"never be silently re-read as a valid address", got, h, p)
				}
				return
			}

			// The dial-class analogue of the URL round-trip: whatever host was
			// supplied, the net package must recover it and the port from the
			// composed address.
			h, p, err := net.SplitHostPort(got)
			if err != nil {
				t.Fatalf("net.SplitHostPort(%q) failed: %v", got, err)
			}
			if h != tc.wantHost {
				t.Errorf("host did not survive the round-trip: want %q, got %q (from %q)",
					tc.wantHost, h, got)
			}
			if p != "587" {
				t.Errorf("port did not survive the round-trip: want %q, got %q (from %q)",
					"587", p, got)
			}
		})
	}
}

// TestHXC268DialAddressIsNotBaseURL pins the separation itself: DialAddress must
// NOT acquire BaseURL's placeholder fallback.
//
// Without this, a later "simplification" that routed DialAddress through
// normalizeHost would compile, keep every case above passing except the empty
// one, and silently point an SMTP or LDAP client at the HelixAgent placeholder
// instead of failing on a host the operator got wrong.
func TestHXC268DialAddressIsNotBaseURL(t *testing.T) {
	if got := DialAddress("", 587); got != ":587" {
		t.Errorf("DialAddress must pass an empty host through so the dial fails loudly, "+
			"not substitute the HelixAgent placeholder: want %q, got %q", ":587", got)
	}
	if got := DialAddress("", 587); got == DefaultHost+":587" {
		t.Errorf("DialAddress substituted DefaultHost %q — a foreign service address must "+
			"never fall back to the HelixAgent endpoint", DefaultHost)
	}
}
