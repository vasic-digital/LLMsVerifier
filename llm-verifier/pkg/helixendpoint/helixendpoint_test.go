package helixendpoint

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// HXC-250 STEP-3 extend case-set for the endpoint resolver: every documented
// resolution path, plus the malformed/boundary inputs a consuming project can
// realistically inject.

func TestResolutionOrder(t *testing.T) {
	cases := []struct {
		name             string
		baseURL, h, port string
		want             string
	}{
		{"all unset -> documented placeholder", "", "", "", "http://localhost:8100"},
		{"host only", "", "agent.internal", "", "http://agent.internal:8100"},
		{"port only", "", "", "7061", "http://localhost:7061"},
		{"host and port", "", "agent.internal", "7061", "http://agent.internal:7061"},
		{"base URL wins over host+port", "http://from-base:1234", "ignored", "9999", "http://from-base:1234"},
		{"base URL trailing slash trimmed", "http://from-base:1234/", "", "", "http://from-base:1234"},
		{"base URL with path preserved", "https://gw.example/helix", "", "", "https://gw.example/helix"},
		{"base URL non-http scheme preserved", "https://gw.example:8443", "", "", "https://gw.example:8443"},
		{"whitespace-only host falls back", "", "   ", "", "http://localhost:8100"},
		{"whitespace-only port falls back", "", "", "  ", "http://localhost:8100"},
		{"non-numeric port falls back", "", "", "not-a-port", "http://localhost:8100"},
		{"zero port falls back", "", "", "0", "http://localhost:8100"},
		{"negative port falls back", "", "", "-1", "http://localhost:8100"},
		{"above-range port falls back", "", "", "65536", "http://localhost:8100"},
		{"max valid port accepted", "", "", "65535", "http://localhost:65535"},
		{"min valid port accepted", "", "", "1", "http://localhost:1"},
		{"IPv6 literal from env is bracketed", "", "::1", "7061", "http://[::1]:7061"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvBaseURL, tc.baseURL)
			t.Setenv(EnvHost, tc.h)
			t.Setenv(EnvPort, tc.port)
			if got := DefaultBaseURL(); got != tc.want {
				t.Errorf("DefaultBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBaseURLExplicitArgs(t *testing.T) {
	cases := []struct {
		name string
		host string
		port int
		want string
	}{
		{"plain host", "agent.internal", 7061, "http://agent.internal:7061"},
		{"IPv4 literal", "10.0.0.5", 7061, "http://10.0.0.5:7061"},
		{"IPv6 literal bracketed", "::1", 7061, "http://[::1]:7061"},
		{"IPv6 full literal bracketed", "fe80::1ff:fe23:4567:890a", 80, "http://[fe80::1ff:fe23:4567:890a]:80"},
		{"already-bracketed IPv6 not double-bracketed", "[::1]", 7061, "http://[::1]:7061"},
		{"blank host falls back", "", 7061, "http://localhost:7061"},
		{"whitespace host falls back", "  ", 7061, "http://localhost:7061"},
		{"invalid low port falls back", "agent.internal", 0, "http://agent.internal:8100"},
		{"invalid high port falls back", "agent.internal", 70000, "http://agent.internal:8100"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BaseURL(tc.host, tc.port); got != tc.want {
				t.Errorf("BaseURL(%q, %d) = %q, want %q", tc.host, tc.port, got, tc.want)
			}
		})
	}
}

// TestBaseURLIPv6IsParseable is the anti-bluff check behind the bracketing:
// the naive fmt.Sprintf("http://%s:%d") form produces a URL whose host is not
// recoverable. Assert the emitted URL round-trips through net/url with the
// host intact.
func TestBaseURLIPv6IsParseable(t *testing.T) {
	got := BaseURL("::1", 7061)
	if !strings.Contains(got, "[::1]") {
		t.Fatalf("IPv6 literal not bracketed: %q", got)
	}
	if strings.Contains(got, "http://::1:") {
		t.Fatalf("emitted the known-malformed unbracketed form: %q", got)
	}
}

func TestJoinPath(t *testing.T) {
	cases := []struct{ base, path, want string }{
		{"http://h:1", "v1/mcp", "http://h:1/v1/mcp"},
		{"http://h:1/", "v1/mcp", "http://h:1/v1/mcp"},
		{"http://h:1", "/v1/mcp", "http://h:1/v1/mcp"},
		{"http://h:1/", "/v1/mcp", "http://h:1/v1/mcp"},
		{"http://h:1///", "///v1/mcp", "http://h:1/v1/mcp"},
		{"http://h:1", "", "http://h:1"},
		{"http://h:1/", "", "http://h:1"},
	}
	for _, tc := range cases {
		if got := JoinPath(tc.base, tc.path); got != tc.want {
			t.Errorf("JoinPath(%q, %q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}

// TestHostPortHelpers pins the individual accessors used by callers that need
// host and port separately rather than a composed URL.
func TestHostPortHelpers(t *testing.T) {
	t.Run("defaults when unset", func(t *testing.T) {
		t.Setenv(EnvHost, "")
		t.Setenv(EnvPort, "")
		if Host() != DefaultHost {
			t.Errorf("Host() = %q, want %q", Host(), DefaultHost)
		}
		if Port() != DefaultPort {
			t.Errorf("Port() = %d, want %d", Port(), DefaultPort)
		}
	})
	t.Run("injected values", func(t *testing.T) {
		t.Setenv(EnvHost, " agent.internal ")
		t.Setenv(EnvPort, " 7061 ")
		if Host() != "agent.internal" {
			t.Errorf("Host() = %q, want trimmed %q", Host(), "agent.internal")
		}
		if Port() != 7061 {
			t.Errorf("Port() = %d, want 7061", Port())
		}
	})
}

// ---- Review-round-2 remediation (§11.4.134 iterate-to-GO) ----

func TestHostRejectsHostPortMistake(t *testing.T) {
	// The plausible operator mistake: port placed in the host variable.
	// Must fall back rather than emit "http://[agent.internal:7061]:8100".
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvPort, "")
	for _, bad := range []string{"agent.internal:7061", "example.com:80", "a:b:c"} {
		t.Setenv(EnvHost, bad)
		if got := Host(); got != DefaultHost {
			t.Errorf("Host() with %q = %q, want fallback %q", bad, got, DefaultHost)
		}
		if got := DefaultBaseURL(); strings.Contains(got, "[") {
			t.Errorf("DefaultBaseURL() with host %q emitted bracketed nonsense: %q", bad, got)
		}
	}
}

func TestBaseURLIPv6ZoneIDEncoded(t *testing.T) {
	got := BaseURL("fe80::1%eth0", 7061)
	if want := "http://[fe80::1%25eth0]:7061"; got != want {
		t.Errorf("BaseURL zone-id = %q, want %q", got, want)
	}
	// Already-encoded zone must not be double-encoded.
	if got2 := BaseURL("fe80::1%25eth0", 7061); got2 != got {
		t.Errorf("already-encoded zone changed: %q vs %q", got2, got)
	}
}

func TestBaseURLRejectsNonIPv6Colon(t *testing.T) {
	if got, want := BaseURL("agent.internal:7061", 7061), "http://localhost:7061"; got != want {
		t.Errorf("BaseURL(host:port mistake) = %q, want %q", got, want)
	}
}

func TestDefaultBaseURLSeparatorOnlyFallsBack(t *testing.T) {
	t.Setenv(EnvHost, "")
	t.Setenv(EnvPort, "")
	for _, sep := range []string{"/", "//", "   /  "} {
		t.Setenv(EnvBaseURL, sep)
		if got, want := DefaultBaseURL(), "http://localhost:8100"; got != want {
			t.Errorf("DefaultBaseURL() with EnvBaseURL=%q = %q, want %q", sep, got, want)
		}
	}
}

func TestZoneIDWithoutIPv6Rejected(t *testing.T) {
	if got, want := BaseURL("host%eth0", 7061), "http://localhost:7061"; got != want {
		t.Errorf("BaseURL(non-IPv6 zone) = %q, want %q", got, want)
	}
}

// ---- Silent-fallback diagnostics (HXC-250 review F2) ----
//
// Three fallbacks used to be reached in total silence even though the consumer
// HAD injected something: a malformed EnvHost, an invalid EnvPort, and an
// EnvBaseURL set without an EnvHost. The third is the one that matters most —
// measured on the real generators, BASE_URL alone leaves 49 of 50 emitted
// artifacts on the placeholder, which is byte-identical to the defect HXC-250
// exists to fix and therefore indistinguishable from it without a diagnostic.
//
// The decision is pinned by pure table tests (no globals, no stderr, every
// branch); the WIRING and the once-per-process dedup are pinned separately by
// the capture tests below. Each layer carries a negative control, so an
// implementation that warned unconditionally would fail rather than pass.

// resetEndpointWarnings clears the once-per-process dedup so each capture test
// starts from a known state and the suite is order-independent. It lives in the
// test file so production carries no test hook.
func resetEndpointWarnings(t *testing.T) {
	t.Helper()
	warnedMu.Lock()
	defer warnedMu.Unlock()
	warned = map[string]bool{}
}

// captureStderr runs fn with os.Stderr redirected to a temp file and returns
// what was written.
//
// NOTE: this swaps the PROCESS-GLOBAL os.Stderr, so it is not safe under
// t.Parallel(). No test in this package calls t.Parallel(); if that ever
// changes, the diagnostics must move to an injected io.Writer instead. A file
// rather than a pipe so a warning can never block on a full pipe buffer.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stderr.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create capture file: %v", err)
	}
	saved := os.Stderr
	os.Stderr = f
	defer func() {
		os.Stderr = saved
		_ = f.Close()
	}()

	fn()

	// Writes to an *os.File are unbuffered, so the bytes are already on disk.
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read capture file: %v", err)
	}
	return string(out)
}

func TestHostFallbackWarnings(t *testing.T) {
	cases := []struct {
		name         string
		rawHost      string
		baseURLSet   bool
		wantCount    int
		wantContains []string
	}{
		{"resolving host, no base URL", "agent.internal", false, 0, nil},
		{"resolving host WITH base URL is the documented pairing", "agent.internal", true, 0, nil},
		{"resolving IPv6 host", "::1", true, 0, nil},
		{"unset host, no base URL is zero-config", "", false, 0, nil},
		{"blank host, no base URL is zero-config", "   ", false, 0, nil},
		{
			"host:port mistake", "agent.internal:7061", false, 1,
			[]string{`HELIX_AGENT_HOST="agent.internal:7061"`, "not a usable URL host", "localhost"},
		},
		{
			"zone id without IPv6", "host%eth0", false, 1,
			[]string{`HELIX_AGENT_HOST="host%eth0"`, "not a usable URL host"},
		},
		{
			"unset host WITH base URL is the 49-of-50 case", "", true, 1,
			[]string{"HELIX_AGENT_BASE_URL is set but HELIX_AGENT_HOST does not resolve", "Host()+Port()"},
		},
		{
			"blank host WITH base URL", "  ", true, 1,
			[]string{"HELIX_AGENT_BASE_URL is set but HELIX_AGENT_HOST does not resolve"},
		},
		{
			"malformed host WITH base URL reports both independent facts",
			"agent.internal:7061", true, 2,
			[]string{"not a usable URL host", "does not resolve"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hostFallbackWarnings(tc.rawHost, tc.baseURLSet)
			if len(got) != tc.wantCount {
				t.Fatalf("hostFallbackWarnings(%q, %v) returned %d warnings, want %d: %q",
					tc.rawHost, tc.baseURLSet, len(got), tc.wantCount, got)
			}
			joined := strings.Join(got, "\n")
			for _, want := range tc.wantContains {
				if !strings.Contains(joined, want) {
					t.Errorf("warning text %q does not mention %q", joined, want)
				}
			}
		})
	}
}

// TestHostFallbackWarningOmitsBaseURLValue is the §11.4.10 guard: a base URL can
// carry userinfo, so the diagnostic names the VARIABLE and never echoes its value.
func TestHostFallbackWarningOmitsBaseURLValue(t *testing.T) {
	got := strings.Join(hostFallbackWarnings("", true), "\n")
	if got == "" {
		t.Fatal("instrument invalid: expected a warning to inspect")
	}
	for _, leak := range []string{"://", "@"} {
		if strings.Contains(got, leak) {
			t.Errorf("warning appears to echo a URL (contains %q), which could leak userinfo: %q", leak, got)
		}
	}
}

// syntheticSecret is a fabricated fixture, not a credential of any real system.
// It exists only so a leak is detectable: if it appears on stderr, a real secret
// in the same position would have appeared too.
const syntheticSecret = "s3cr3t-synthetic-not-a-real-credential"

// credentialBearingValues enumerates the shapes an operator can realistically
// paste into EnvHost or EnvPort that embed a secret. None of them parses as a
// port, so every one reaches the port diagnostic and must be withheld there.
//
// reachesHostWarning records a fact about normalizeHost rather than an intent:
// a value it ACCEPTS is used as the host and never reported, so that path emits
// nothing and cannot leak. Only a value it REJECTS reaches the host diagnostic,
// and only then does redaction have anything to do. Recording this per-fixture
// keeps the guard from silently skipping the shapes it cannot exercise.
//
// The neighbouring variable's URL is the likeliest wrong paste: the host warning
// itself tells the operator to "use HELIX_AGENT_BASE_URL to inject host and port
// as one string", so it anticipates this confusion in its own text.
var credentialBearingValues = []struct {
	name               string
	raw                string
	reachesHostWarning bool
}{
	{"scheme-ful URL with userinfo", "https://svcuser:" + syntheticSecret + "@gw.example", true},
	{"bare userinfo@host", "svcuser:" + syntheticSecret + "@gw.example", true},
	{"scheme-relative URL with userinfo", "//svcuser:" + syntheticSecret + "@gw.example", true},
	{"bare user:pass pair, no @ at all", "svcuser:" + syntheticSecret, true},
	{"query-string token", "https://gw.example/v1?token=" + syntheticSecret, true},
	{"fragment token", "https://gw.example#access_token=" + syntheticSecret, true},
	{"percent-encoded userinfo", "svcuser%3A" + syntheticSecret + "%40gw.example", true},

	// Uppercase "%3A" and NOTHING else: no "@", no "/", no "?", no "#", no bare
	// colon, and no digits-only "%40" to fall back on. The ONLY thing that can
	// withhold this value is the case-fold in front of the escape check, so this
	// fixture is what makes that fold falsifiable — without it the fold can be
	// deleted and the whole suite still passes, because every other encoded
	// fixture carries a "%40" whose two characters are digits and therefore match
	// unfolded.
	{"percent-encoded userinfo, uppercase escape only", "svcuser%3A" + syntheticSecret, true},

	// Doubly-encoded: "%2540" is "%40" with its own "%" re-encoded, and "%253A"
	// likewise. Neither "%40" nor "%3A" occurs anywhere in this string, so a check
	// looking for those two literals waves a structured credential straight onto
	// stderr. Recognising ANY percent-escape catches it — and because "%25" is
	// itself an escape, it catches every deeper encoding by the same rule.
	//
	// DESIGNATED prover that the escape check must be GENERAL. It was the sole one
	// until the hex-bound fixtures below landed, and the sharing they introduce is
	// structural: the two-literal mutant keeps exactly "%40" and "%3a", so ANY
	// fixture whose only escape sits on a bound those two do not occupy — "f" and
	// "9" — necessarily leaks under it too. Giving those fixtures a second, retained
	// escape would defeat them, since the retained one would keep them redacted
	// under the very bound mutation they exist to catch. The digit-LOWER fixture
	// shows the mitigation where it IS available: it is built on "%40" precisely so
	// it stays redacted here and leaves this fixture's job undiluted.
	{"double-percent-encoded userinfo", "svcuser%253A" + syntheticSecret + "%2540gw.example", true},

	// The ONLY escape in this value is its last three bytes, so it is the only
	// fixture whose withholding depends on containsPercentEscape scanning to the
	// final possible escape position. Every other encoded fixture carries an
	// escape mid-string, which a scan that stops one index early still finds — so
	// without this fixture the loop bound can be tightened from "i+2 < len(s)" to
	// "i+3 < len(s)" and the whole suite still passes, while a terminal-escape
	// value ("svcuser%3a", "s3cr3t-x%21") is echoed verbatim onto stderr.
	//
	// It carries nothing else the predicate could catch: no "@", "/", "?" or "#",
	// and no colon at all — so the escape scan is the sole thing standing between
	// this value and the operator's terminal.
	//
	// The escape is deliberately LOWERCASE so this fixture pins the loop BOUND and
	// nothing else: an uppercase "%3A" here would also die to a deleted case-fold,
	// which would leave two fixtures answering for one clause and none uniquely for
	// the other. The fixture above stays the sole prover of the fold; this one the
	// sole prover of the bound.
	{"percent-encoded, escape terminal", syntheticSecret + "%3a", true},

	// The mirror of the fixture above at the OTHER end of the same scan: the only
	// escape is this value's FIRST three bytes, so it is the only fixture whose
	// withholding depends on containsPercentEscape starting at index 0. Every
	// other encoded fixture carries its escape at a later index, which a scan
	// beginning one byte in still finds — so without this fixture the loop start
	// can be moved from "i := 0" to "i := 1" and the whole suite still passes,
	// while any value that OPENS with an escape is echoed verbatim.
	//
	// That shape is the URL-encoded twin of the `curl -u :<token>` paste at the
	// end of this table: an encoder handed a fragment that BEGINS with a
	// delimiter emits that delimiter's escape first and the credential after it.
	//
	// The escape is "%3b" — an encoded ";" — rather than the more obvious "%3a".
	// Both of its hex characters are then INTERIOR to all four isLowerHexDigit
	// bounds ("3" is neither "0" nor "9"; "b" is neither "a" nor "f"), so this
	// fixture answers for the loop START and nothing else. "%3a" is an equally
	// good witness for the start index but its "a" sits ON the letter lower
	// bound, which would dilute that row's designated prover for no gain.
	{"escape at index 0, encoded leading delimiter", "%3b" + syntheticSecret, true},

	// A "user:pass:port" paste — the operator meant to give host and port, and
	// carried the userinfo along. The secret sits in the MIDDLE, so the last colon
	// is the port separator and its all-digit right-hand side satisfies the port
	// heuristic; a check that inspects only the last colon therefore waves this
	// through and echoes the secret in BOTH diagnostics.
	//
	// The marker the doc defines is "a ':' whose right-hand side is not a plausible
	// port", and such a colon plainly EXISTS here — the first one, whose RHS is
	// "<secret>:7061". Inspecting only the last colon under-implements that marker
	// rather than bounding it, so this fixture is what holds the predicate to the
	// contract its own documentation states.
	//
	// This is also the DESIGNATED prover that the multi-colon clause EXISTS: delete
	// the clause outright and this is the fixture whose leak the suite reports. The
	// two fixtures immediately below die to that same deletion as well — unavoidably,
	// since a value probing the clause's THRESHOLD or its ANCHOR must first reach the
	// clause at all — so removal of THIS clause is answered by three fixtures rather
	// than one. The one-to-one property is preserved everywhere it can be: each of
	// those two is the SOLE killer of its own mutation, and this one stays the sole
	// fixture whose ONLY reason for being withheld is the clause.
	//
	// The same reach-it-before-you-probe-it rule recurs verbatim one clause down: the
	// last-colon clause's removal is likewise answered by three — the bare user:pass
	// pair, which is its designated prover, plus the two `curl -u` edge fixtures that
	// must reach it to probe its two remaining terms.
	{"user:pass:port paste, secret mid-string", "svcuser:" + syntheticSecret + ":7061", true},

	// THRESHOLD. The clause reads "MORE THAN ONE colon", and the fixture above has
	// exactly two — so it cannot tell "> 1" from "== 2". A password that itself
	// contains a colon produces three, which the equality form waves through: the
	// clause declines, the last colon's RHS is the all-digit "7061", and the secret
	// reaches both diagnostics verbatim. Three colons is the smallest count that
	// distinguishes the two forms, and this is the only fixture that carries it.
	//
	// It probes nothing else: no "@", "/", "?" or "#", and no percent-escape — the
	// multi-colon clause is the sole thing withholding it.
	{"three-colon paste, password contains a colon", "svcuser:" + syntheticSecret + ":x2:7061", true},

	// ANCHOR. The clause's IPv6 exemption is a PREFIX test — the bracket must OPEN
	// the value — and the only bracketed fixture in the suite ("[::1]:7061", among
	// the ordinary-value controls) opens with one, so it cannot tell HasPrefix from
	// Contains. A bracket that appears mid-string satisfies the loosened form and
	// disarms the exemption for a value that is no kind of IPv6 literal: the clause
	// declines, the trailing "7061" satisfies the port heuristic, and the secret is
	// printed. Captured live before this fixture existed —
	// HELIX_AGENT_PORT="svc[1]:<secret>:7061" echoed the value in full.
	//
	// Deliberately keeps its bracket AFTER the first character and its colon count at
	// two, so it answers for the anchor alone and leaves the threshold to the fixture
	// above.
	{"bracket mid-string, not an IPv6 literal", "svc[1]:" + syntheticSecret + ":7061", true},

	// ---- One marker each: the four members of ContainsAny("@/?#") ----
	//
	// The lesson these four encode is that COVERAGE and FALSIFIABILITY are different
	// properties. Every marker in that set is exercised many times over by the
	// fixtures above — the lattice LOOKS complete — but each of those values carries
	// SEVERAL markers at once, so deleting any single character from the set changes
	// no verdict: a different clause still withholds the value and the suite stays
	// green while the marker it was meant to prove is gone. A clause is only proven
	// by a value that ISOLATES it, and until this block only "?" had one.
	//
	// Each fixture below therefore carries exactly ONE marker and nothing else the
	// predicate could catch — no second marker, no percent-escape, no colon whose
	// right-hand side fails the port heuristic. Delete its character from the set and
	// it is the sole fixture that leaks.

	// "@" — the userinfo delimiter, and the most realistic leak of the four: a real
	// URL authority routinely carries a token AS the username, so this is the exact
	// shape an operator pastes when they put a credentialed authority in the wrong
	// variable. Its single colon introduces the all-digit "7061", so neither colon
	// clause fires and the "@" alone stands between it and the terminal.
	{"token-as-username authority, only marker is @", syntheticSecret + "@gw.example:7061", true},

	// "/" — a bare path paste, no scheme and no authority-delimiter. normalizeHost
	// ACCEPTS it (no colon, no zone marker), so the host path stays silent and only
	// the port diagnostic can leak it; the assertion below pins that silence rather
	// than skipping the case.
	{"bare-path paste, only marker is /", "gw.example/" + syntheticSecret, false},

	// "#" — a fragment-carried token with the scheme stripped off. The scheme-ful
	// sibling above ("fragment token") also carries "/" twice over, which is why it
	// cannot prove this marker. Host-accepted for the same reason as the "/" case.
	{"schemeless fragment token, only marker is #", "gw.example#access_token=" + syntheticSecret, false},

	// "?" — the fourth member, and the one that already had its isolating fixture
	// before the block above was written. Accepted by normalizeHost (no colon, no
	// zone marker), so the host path reports nothing at all — asserted below rather
	// than skipped.
	{"query-string token, no scheme", "gw.example?token=" + syntheticSecret, false},

	// ---- One bound each: the four bounds of isLowerHexDigit's two ranges ----
	//
	// The lesson of the block above — coverage is not falsifiability — recurs one
	// level down, inside the character class the escape check is built from. Every
	// escape in every fixture above happens to use hex letter "a" and digits
	// 0,2,3,4,5, so the class is entered from both of its ranges many times over
	// and yet NEITHER upper bound moves any verdict: measured, `c <= 'f'` could be
	// tightened all the way to `c <= 'a'`, and `c <= '9'` to `c <= '5'`, with the
	// entire suite still green while "svcuser%2F<secret>" and "svcuser%3F<secret>"
	// — the encoded "/" and "?", precisely the clause-1 markers that percent-
	// encoding hides — were echoed verbatim onto stderr.
	//
	// A bound is moved only by a value whose escape sits ON it. Each fixture below
	// therefore places exactly ONE hex character on exactly ONE bound and keeps
	// every other bound comfortably interior, so it answers for that bound alone.

	// LETTER UPPER BOUND ("f"). The sole escape is "%3f" — an encoded "?" — whose
	// letter is the last member of a-f while its digit "3" sits interior to 0-9.
	// Tighten `c <= 'f'` by any amount and this is the fixture that leaks.
	//
	// Deliberately LOWERCASE, for the same reason the "escape terminal" fixture
	// above is: the uppercase "%3F" the leak was originally measured with would ALSO
	// die to a deleted case-fold, leaving the fold answered by two fixtures and this
	// bound by none uniquely. Lowercase keeps each one-to-one.
	{"encoded '?', hex letter upper bound", "svcuser%3f" + syntheticSecret, true},

	// DIGIT UPPER BOUND ("9"). "%39" is an over-encoded digit — the shape an
	// over-eager encoder produces when it escapes characters that never needed it.
	// Both its hex characters are digits, so it is blind to BOTH letter bounds and
	// to the case-fold, and "3" keeps it clear of the digit lower bound: the only
	// bound it can answer for is the one its "9" sits on.
	{"over-encoded digit, hex digit upper bound", "svcuser%39" + syntheticSecret, true},

	// DIGIT LOWER BOUND ("0"). A token pasted as the userinfo of an authority whose
	// "@" is percent-encoded — the encoded twin of the "token-as-username authority"
	// fixture above. Its "0" is the first member of 0-9 and its "4" is interior.
	//
	// Alone among these three it does NOT widen the generalization mutant's kill
	// set: "%40" is one of the two literals that mutant keeps, so it stays redacted
	// there and "double-percent-encoded" remains the sole fixture proving the escape
	// check must be GENERAL rather than a two-literal lookup.
	{"encoded-@ authority, hex digit lower bound", syntheticSecret + "%40gw.example", true},

	// ---- The last-colon rule's two edge positions ----
	//
	// Both are the `curl -u` form, which is why they are realistic rather than
	// contrived: a great many API vendors document authentication as literally
	// `curl -u <api_key>:` (token as username, empty password) or `curl -u :<token>`
	// (empty username, token as password), so a copied credential arriving with a
	// colon welded to one end is an ordinary paste, not an exotic one.
	//
	// Each sits at an index the interior fixtures cannot reach, and the rule reads
	// the colon's position and its right-hand side — so the ends are exactly where
	// its two remaining terms live.

	// EMPTY RIGHT-HAND SIDE. The trailing colon leaves "" to the right, and the rule
	// withholds it only because isAllDigits reports false for the empty string. Let
	// that report true — a one-word change, and the intuitive reading of "all of the
	// zero characters here are digits" — and the colon is read as a port separator
	// with a satisfied port, so the token to its LEFT is echoed in full.
	//
	// This is the sole fixture with a colon at the last index, so it is the only one
	// whose verdict that word can move.
	{"curl -u <token>: — empty-password basic auth", syntheticSecret + ":", true},

	// LEADING COLON. The mirror case: the colon at index 0 is what an empty USERNAME
	// leaves behind. The rule admits it through `i >= 0`; tighten that to `i > 0` —
	// the ordinary defensive-looking off-by-one, since index 0 reads like "not
	// found" in many APIs — and the clause declines entirely, leaving nothing else
	// to catch a value that carries no other marker.
	{"curl -u :<token> — empty-username basic auth", ":" + syntheticSecret, true},

	// ---- One position each: the three loop terms of isAllDigits' scan ----
	//
	// The lesson recurs a third time, now one level below the last-colon rule
	// itself. A scan loop is not one decision but THREE — where it starts, where
	// it stops, and how it steps — and a fixture falsifies only the term whose
	// miss it can survive. Every right-hand side above is either all-digits or
	// fails on its FIRST byte, so all three terms could be moved with the suite
	// still green: measured before these fixtures existed, start "i := 1", end
	// "i < len(s)-1" and step "i += 2" EACH left the entire suite passing while a
	// password one stray character away from a port was echoed in full.
	//
	// All three below are the same realistic paste — a credential in the host or
	// port variable followed by a mistyped port — differing only in WHERE the
	// stray non-digit lands. The "7061x" entry in the port control set below is
	// that same typo without a credential attached, which is what makes the class
	// ordinary rather than contrived.

	// START. The stray character is FIRST, so a scan beginning at index 1 sees
	// only "7061", reports all-digits, and the last colon is read as a port
	// separator with a satisfied port — echoing the token to its left.
	{"mistyped port, stray char leading", syntheticSecret + ":x7061", true},

	// END. The stray character is LAST, so a scan stopping one byte early sees
	// only "7061" and reaches the same wrong verdict from the opposite end.
	{"mistyped port, stray char trailing", syntheticSecret + ":7061x", true},

	// STEP. The stray character sits at an ODD index, so a scan stepping by two
	// walks straight over it and every byte it looks at IS a digit. Both ends are
	// digits here precisely so neither the start nor the end term can answer for
	// this one — only the step can.
	{"mistyped port, stray char at an odd index", syntheticSecret + ":9x99", true},
}

// TestFallbackWarningsRedactCredentialBearingValues is the §11.4.10 guard for the
// two diagnostics that DO echo their rejected value.
//
// TestHostFallbackWarningOmitsBaseURLValue guards the base-URL message, which
// never echoes anything. These two messages are the other half: they echo, and
// the value they echo is by definition NOT the shape the variable expects, so it
// may well be the URL that belongs in the neighbouring variable.
func TestFallbackWarningsRedactCredentialBearingValues(t *testing.T) {
	for _, tc := range credentialBearingValues {
		t.Run("host/"+tc.name, func(t *testing.T) {
			got := strings.Join(hostFallbackWarnings(tc.raw, false), "\n")

			if !tc.reachesHostWarning {
				// normalizeHost accepts it, so nothing is reported and nothing can
				// leak from this path. Asserting the silence keeps the fixture
				// honest: if normalizeHost later starts rejecting this shape, the
				// value would begin reaching stderr and this fails rather than
				// quietly passing.
				if got != "" {
					t.Fatalf("expected no host warning for an accepted value, got %q", got)
				}
				if strings.Contains(got, syntheticSecret) {
					t.Errorf("warning echoes the injected secret verbatim: %q", got)
				}
				return
			}

			if got == "" {
				t.Fatal("instrument invalid: expected a warning to inspect")
			}
			assertRedacted(t, got, EnvHost)
		})

		t.Run("port/"+tc.name, func(t *testing.T) {
			got := portFallbackWarning(tc.raw)
			if got == "" {
				t.Fatal("instrument invalid: expected a warning to inspect")
			}
			assertRedacted(t, got, EnvPort)
		})
	}
}

// assertRedacted checks the three things a redacted diagnostic must satisfy at
// once: the secret is gone, the redaction is announced rather than silent, and
// the message still identifies which variable was rejected.
func assertRedacted(t *testing.T, warning, wantVar string) {
	t.Helper()
	if strings.Contains(warning, syntheticSecret) {
		t.Errorf("warning echoes the injected secret verbatim: %q", warning)
	}
	if !strings.Contains(warning, redactedValue) {
		t.Errorf("warning does not announce the redaction: %q", warning)
	}
	if !strings.Contains(warning, wantVar) {
		t.Errorf("warning no longer names the rejected variable %s: %q", wantVar, warning)
	}
}

// TestFallbackWarningsStillQuoteOrdinaryValues is the negative control without
// which the guard above passes vacuously: an implementation that redacted
// unconditionally would satisfy every assertion in it while destroying the
// diagnostic. An ordinary rejected value is what the operator needs to see.
func TestFallbackWarningsStillQuoteOrdinaryValues(t *testing.T) {
	// "agent.internal:7061" is the case the host message exists to explain, so it
	// is the one that would hurt most to lose.
	//
	// Note the set is drawn only from values normalizeHost REJECTS. Plain tokens
	// like "abc" — or even "not a host" — are ACCEPTED as hosts and never reach
	// this diagnostic, so they cannot serve as a control here; "abc" plays that
	// role in the port set below, where it genuinely is rejected.
	//
	// "[::1]:7061" carries a second job since the multi-colon clause landed: it is
	// the only entry here with more than one colon, so it is what proves the "["
	// guard on that clause still spares a bracketed IPv6 literal. Drop the guard
	// and this control is the assertion that notices.
	//
	// The last two close the over-redaction side of two terms whose UNDER-redaction
	// side the credential fixtures already cover, and which nothing here could reach
	// before:
	//
	//   - "host%if1" — a "%" followed by a NON-hex byte and then a hex one. The
	//     escape check reads TWO bytes after the "%", and "host%eth0" only proves it
	//     reads the SECOND: with "%e" the first byte is hex, so dropping the second
	//     check over-redacts and "host%eth0" notices, but dropping the FIRST check
	//     leaves "%..t" failing on the second byte and nothing notices. Here the
	//     bytes are reversed — "i" is not hex, "f" is — so this is the control that
	//     notices the first byte going unread, and a malformed zone ID is exactly
	//     where a stray "%" legitimately turns up.
	//
	//   - "agent.internal:9090" — a port containing a "9". Every other port here is
	//     built from 7,0,6,1,8, so isAllDigits' upper bound could be tightened from
	//     '9' to '8' and no control would notice, silently redacting every value
	//     whose port contains a 9.
	hostOrdinary := []string{
		"agent.internal:7061", "host%eth0", "[::1]:7061", "example.com:80",
		"host%if1", "agent.internal:9090",
	}
	for _, raw := range hostOrdinary {
		t.Run("host/"+raw, func(t *testing.T) {
			got := strings.Join(hostFallbackWarnings(raw, false), "\n")
			assertQuotedVerbatim(t, got, raw)
		})
	}

	portOrdinary := []string{"not-a-port", "0", "-1", "65536", "70000", "80.5", "7061x", "abc"}
	for _, raw := range portOrdinary {
		t.Run("port/"+raw, func(t *testing.T) {
			assertQuotedVerbatim(t, portFallbackWarning(raw), raw)
		})
	}
}

func assertQuotedVerbatim(t *testing.T, warning, raw string) {
	t.Helper()
	if warning == "" {
		t.Fatalf("instrument invalid: %q produced no warning to inspect", raw)
	}
	if !strings.Contains(warning, strconv.Quote(raw)) {
		t.Errorf("ordinary rejected value %q is no longer quoted in the warning: %q", raw, warning)
	}
	if strings.Contains(warning, redactedValue) {
		t.Errorf("ordinary rejected value %q was redacted, which removes the diagnostic: %q", raw, warning)
	}
}

// TestFallbackWarningsRedactOnStderr pins the WIRING of the redaction. The pure
// functions above prove the decision; this proves the bytes that actually reach
// the operator's terminal carry it — the leak was a stderr leak.
func TestFallbackWarningsRedactOnStderr(t *testing.T) {
	leaky := "https://svcuser:" + syntheticSecret + "@gw.example"

	t.Run("host", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvBaseURL, "")
		t.Setenv(EnvPort, "")
		t.Setenv(EnvHost, leaky)

		out := captureStderr(t, func() { _ = Host() })
		if !strings.Contains(out, "not a usable URL host") {
			t.Fatalf("instrument invalid: expected the host fallback warning; stderr was %q", out)
		}
		assertRedacted(t, out, EnvHost)
	})

	t.Run("port", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvBaseURL, "")
		t.Setenv(EnvHost, "")
		t.Setenv(EnvPort, leaky)

		out := captureStderr(t, func() { _ = Port() })
		if !strings.Contains(out, "is not an integer in the range") {
			t.Fatalf("instrument invalid: expected the port fallback warning; stderr was %q", out)
		}
		assertRedacted(t, out, EnvPort)
	})
}

func TestPortFallbackWarning(t *testing.T) {
	silent := []string{"", "   ", "1", "7061", "65535", " 7061 "}
	for _, raw := range silent {
		if got := portFallbackWarning(raw); got != "" {
			t.Errorf("portFallbackWarning(%q) = %q, want no warning", raw, got)
		}
	}

	noisy := []string{"not-a-port", "0", "-1", "65536", "70000", "80.5", "7061x"}
	for _, raw := range noisy {
		got := portFallbackWarning(raw)
		if got == "" {
			t.Errorf("portFallbackWarning(%q) returned no warning, want one", raw)
			continue
		}
		if !strings.Contains(got, "HELIX_AGENT_PORT="+strconv.Quote(strings.TrimSpace(raw))) {
			t.Errorf("warning for %q does not quote the offending value: %q", raw, got)
		}
		if !strings.Contains(got, "8100") {
			t.Errorf("warning for %q does not name the placeholder port: %q", raw, got)
		}
	}
}

// TestHostWarnsOnMalformedInjection pins the WIRING: Host() must actually route
// its fallback through the diagnostic, not merely have one available.
func TestHostWarnsOnMalformedInjection(t *testing.T) {
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvPort, "")

	t.Run("malformed host warns", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvHost, "agent.internal:7061")
		var got string
		out := captureStderr(t, func() { got = Host() })
		if got != DefaultHost {
			t.Errorf("Host() = %q, want fallback %q", got, DefaultHost)
		}
		if !strings.Contains(out, "not a usable URL host") {
			t.Errorf("Host() fell back silently; stderr was %q", out)
		}
	})

	// Negative control: without it, an implementation that warned on every call
	// would pass the assertion above.
	t.Run("valid host is silent", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvHost, "agent.internal")
		out := captureStderr(t, func() { _ = Host() })
		if out != "" {
			t.Errorf("valid host produced stderr output: %q", out)
		}
	})

	t.Run("unset host is silent", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvHost, "")
		out := captureStderr(t, func() { _ = Host() })
		if out != "" {
			t.Errorf("zero-config produced stderr output: %q", out)
		}
	})
}

func TestPortWarnsOnInvalidInjection(t *testing.T) {
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvHost, "")

	t.Run("invalid port warns", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvPort, "not-a-port")
		var got int
		out := captureStderr(t, func() { got = Port() })
		if got != DefaultPort {
			t.Errorf("Port() = %d, want fallback %d", got, DefaultPort)
		}
		if !strings.Contains(out, "HELIX_AGENT_PORT") {
			t.Errorf("Port() fell back silently; stderr was %q", out)
		}
	})

	t.Run("valid port is silent", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvPort, "7061")
		out := captureStderr(t, func() { _ = Port() })
		if out != "" {
			t.Errorf("valid port produced stderr output: %q", out)
		}
	})

	t.Run("unset port is silent", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvPort, "")
		out := captureStderr(t, func() { _ = Port() })
		if out != "" {
			t.Errorf("zero-config produced stderr output: %q", out)
		}
	})
}

// TestHostWarnsWhenBaseURLSetWithoutHost is the case that must land: its failure
// mode is byte-identical to the original HXC-250 defect, so documentation alone
// cannot distinguish it. Host() returns the placeholder while the operator
// believes their injection is in force.
func TestHostWarnsWhenBaseURLSetWithoutHost(t *testing.T) {
	t.Setenv(EnvPort, "")

	t.Run("base URL without host warns", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvBaseURL, "https://gw.example:8443")
		t.Setenv(EnvHost, "")

		var got string
		out := captureStderr(t, func() { got = Host() })
		if got != DefaultHost {
			t.Fatalf("instrument invalid: Host() = %q, expected the placeholder %q", got, DefaultHost)
		}
		for _, want := range []string{EnvBaseURL, EnvHost, "does not resolve"} {
			if !strings.Contains(out, want) {
				t.Errorf("warning does not mention %q; stderr was %q", want, out)
			}
		}
	})

	// Negative control: the documented pairing must NOT nag.
	t.Run("base URL WITH host is silent", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvBaseURL, "https://gw.example:8443")
		t.Setenv(EnvHost, "agent.internal")
		out := captureStderr(t, func() { _ = Host() })
		if out != "" {
			t.Errorf("the documented BASE_URL+HOST pairing warned: %q", out)
		}
	})

	// A separator-only base URL is treated as unset by DefaultBaseURL, so it must
	// be treated as unset here too — otherwise the resolver and the diagnostic
	// disagree about what "set" means.
	t.Run("separator-only base URL is unset for the diagnostic too", func(t *testing.T) {
		resetEndpointWarnings(t)
		t.Setenv(EnvBaseURL, "/")
		t.Setenv(EnvHost, "")
		out := captureStderr(t, func() { _ = Host() })
		if out != "" {
			t.Errorf("separator-only base URL warned even though it resolves as unset: %q", out)
		}
	})
}

// TestHostEmitsEveryWarningWhenBothFire pins the WIRING of the two-warning case.
//
// hostFallbackWarnings' contract is that a malformed host WITH a base URL set
// reports BOTH facts because the operator needs each, and TestHostFallbackWarnings
// proves the pure function returns both. Nothing proved Host() EMITS both: every
// other stderr test drives a configuration producing exactly ONE warning, so the
// loop forwarding them to warnOnce could skip its first element, stop one short,
// or step by two with the suite still green while the operator silently lost one
// of the two facts. Measured before this test existed: the step form survived the
// entire suite. It is the same three-term loop-position rule the fixture blocks
// above apply to the predicate, applied here to the diagnostics wiring.
func TestHostEmitsEveryWarningWhenBothFire(t *testing.T) {
	resetEndpointWarnings(t)
	t.Setenv(EnvBaseURL, "https://gw.example:8443")
	t.Setenv(EnvHost, "agent.internal:7061")
	t.Setenv(EnvPort, "")

	var got string
	out := captureStderr(t, func() { got = Host() })
	if got != DefaultHost {
		t.Fatalf("instrument invalid: Host() = %q, expected the placeholder %q", got, DefaultHost)
	}
	if n := strings.Count(out, warnPrefix); n != 2 {
		t.Fatalf("Host() emitted %d warnings, want both independent facts: %q", n, out)
	}
	for _, want := range []string{"not a usable URL host", "does not resolve"} {
		if !strings.Contains(out, want) {
			t.Errorf("stderr is missing the %q fact: %q", want, out)
		}
	}
}

// TestWarningsAreOncePerProcess pins the dedup: the endpoint is re-resolved once
// per generated artifact (~50 per run), so an unguarded warning would bury itself.
func TestWarningsAreOncePerProcess(t *testing.T) {
	resetEndpointWarnings(t)
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvHost, "agent.internal:7061")
	t.Setenv(EnvPort, "not-a-port")

	first := captureStderr(t, func() {
		_ = Host()
		_ = Port()
	})
	if n := strings.Count(first, warnPrefix); n != 2 {
		t.Fatalf("first resolution emitted %d warnings, want 2 (one host, one port): %q", n, first)
	}

	repeat := captureStderr(t, func() {
		for i := 0; i < 50; i++ {
			_ = Host()
			_ = Port()
		}
	})
	if repeat != "" {
		t.Errorf("50 further resolutions re-warned instead of staying quiet: %q", repeat)
	}
}

// TestDistinctWarningsAreNotSuppressedByEachOther: dedup is per-message, so a
// second, different misconfiguration is not hidden behind the first.
func TestDistinctWarningsAreNotSuppressedByEachOther(t *testing.T) {
	resetEndpointWarnings(t)
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvPort, "")

	t.Setenv(EnvHost, "first.bad:1")
	out1 := captureStderr(t, func() { _ = Host() })
	if !strings.Contains(out1, "first.bad:1") {
		t.Fatalf("instrument invalid: first warning missing: %q", out1)
	}

	t.Setenv(EnvHost, "second.bad:2")
	out2 := captureStderr(t, func() { _ = Host() })
	if !strings.Contains(out2, "second.bad:2") {
		t.Errorf("a second, different misconfiguration was suppressed by the first: %q", out2)
	}
}

// TestZoneIDTwentyFiveNotDropped: a bare "%25" is the genuine numeric zone index
// 25, not an empty already-encoded zone. Trimming it away silently emitted a
// DIFFERENT address (the zone vanished).
func TestZoneIDTwentyFiveNotDropped(t *testing.T) {
	if got, want := BaseURL("fe80::1%25", 7061), "http://[fe80::1%2525]:7061"; got != want {
		t.Errorf("BaseURL(zone 25) = %q, want %q — the zone must survive", got, want)
	}
	// The already-encoded form is still decoded exactly once.
	if got, want := BaseURL("fe80::1%25eth0", 7061), "http://[fe80::1%25eth0]:7061"; got != want {
		t.Errorf("BaseURL(encoded zone) = %q, want %q", got, want)
	}
}
