// Package helixendpoint resolves the HelixAgent service endpoint that generated
// CLI-agent configurations point at.
//
// # Why this package exists (HXC-250 / CONST-051(B))
//
// LLMsVerifier is shared infrastructure consumed by N >= 2 independent
// projects. A consuming project's deployment topology — the host and port its
// HelixAgent server actually listens on — is NOT knowable here and MUST NOT be
// baked into this submodule. Before HXC-250 the endpoint was written as a
// literal in nine places, so every generated config pointed every end user at
// one fixed host:port no matter what the consumer deployed.
//
// The endpoint is therefore INJECTED, never reached for. Resolution order,
// highest precedence first:
//
//  1. EnvBaseURL ("HELIX_AGENT_BASE_URL") — a complete base URL, used verbatim
//     (minus any trailing slash). Covers schemes, paths and non-default forms
//     this package should not try to reconstruct.
//  2. EnvHost / EnvPort ("HELIX_AGENT_HOST" / "HELIX_AGENT_PORT") — host and
//     port injected separately; each falls back independently.
//  3. DefaultHost / DefaultPort — a documented loopback PLACEHOLDER.
//
// The step-3 placeholder is a last-resort default so a zero-configuration call
// still produces a syntactically valid config. It is NOT a claim about any
// consumer's real deployment: a consuming project whose server listens
// elsewhere MUST inject via step 1 or 2 (or via the caller-supplied
// host/port parameters that every exported builder here accepts). No value
// here is derived from, or intended to match, any particular consumer.
//
// Callers that already hold a configured host/port (for example a generator
// config field populated by the consuming project) should pass it to BaseURL
// directly and never consult the env fallbacks.
//
// # Second responsibility: host:port composition for FOREIGN services (HXC-268)
//
// Besides resolving the HelixAgent endpoint, this package owns the module's
// host:port COMPOSITION rules, because they are the same rules wherever an
// address is built: bracket an IPv6 literal per RFC 3986 §3.2.2, and do not
// double-bracket one the caller already bracketed. Before HXC-268 those rules
// were re-implemented per call site, which is wrong for every IPv6 host.
//
// The census is 31 sites across both modules of this repository — 24 routed
// through this package by HXC-268, and 7 tracked for follow-up under HXC-280
// (see below). It reached 31 only after four corrections (15 -> 26 -> 30 -> 31),
// and the reason is worth recording, because it is a fact about SEARCHING rather
// than about this package: each earlier count was taken by looking for the shape
// the known sites used. The 31st was found only by sweeping IDIOM FAMILIES
// deliberately — it composes its authority with "+=" and a local intToString
// helper, so it is invisible to every fmt.Sprintf-shaped search, including the
// ones that found the other thirty. A census of this defect class is only as
// complete as the set of idioms it enumerates.
//
// The 7 deferred sites are NOT servable by either builder here: they carry
// userinfo in the authority (an AMQP URI, Postgres and MySQL DSNs) or belong to
// foreign services whose URL needs no placeholder fallback (a vector database, a
// log-analysis collector, a stream processor). BaseURL substitutes a host and
// DialAddress is not a URL, so both would be wrong; HXC-280 covers them with a
// defaults-free builder. One of them (the AMQP URI) lives in the ROOT module,
// but the deferral is about semantics, not reachability: importing this
// package across the module seam is mechanically available (the root go.mod
// already requires and replaces digital.vasic.llmsverifier, and other
// root-module files already import its subpackages) — the reason neither
// builder here serves it is that the AMQP URI needs userinfo (user:pass@) and
// a trailing vhost path segment, and neither BaseURL nor DialAddress composes
// either.
//
// Two builders, deliberately separate, because their fallback semantics differ
// and confusing them is itself a defect:
//
//   - BaseURL — for the HelixAgent endpoint THIS package resolves. Applies the
//     DefaultHost/DefaultPort placeholder fallback and RFC 6874 zone encoding.
//   - DialAddress — for net.Dial-style addresses of services this package does
//     NOT own (LDAP directories, SMTP relays). NO placeholder fallback, no zone
//     encoding, no scheme; a bad host fails loudly at dial time rather than being
//     silently redirected at the HelixAgent port.
//
// So this package is not "HelixAgent-only" — it is the module's single
// address-composition chokepoint, and DialAddress lives here rather than beside
// each dialer precisely so the bracketing rules cannot drift apart.
package helixendpoint

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Environment variables through which a consuming project injects its endpoint.
const (
	EnvBaseURL = "HELIX_AGENT_BASE_URL"
	EnvHost    = "HELIX_AGENT_HOST"
	EnvPort    = "HELIX_AGENT_PORT"
)

// Documented last-resort placeholders. See the package doc: these are a
// syntactically-valid fallback, not a consumer's real deployment.
const (
	DefaultHost = "localhost"
	DefaultPort = 8100
)

// --- Silent-fallback diagnostics -------------------------------------------
//
// Three fallbacks below are reached only when a consumer DID inject something
// and it was not usable. Falling back without a word makes an injection mistake
// indistinguishable from the zero-configuration path: the generated configs stay
// syntactically valid, and every client they configure quietly points at a
// placeholder that serves something else. Each of those three emits a warning.
//
// Warn, never error. Host and Port have infallible signatures precisely because
// zero-configuration generation is a documented, supported mode; returning an
// error would break it, and changing what a generator emits on a fallback would
// be a behaviour change no diagnostic needs. A warning leaves every generated
// byte identical and only adds a line of stderr.

// warnPrefix marks these diagnostics in a stream that already carries other
// generator output.
const warnPrefix = "helixendpoint: WARNING: "

var (
	warnedMu sync.Mutex
	warned   = map[string]bool{}
)

// warnOnce writes msg to stderr the first time this process produces that exact
// message, and never again.
//
// Deduplication is per-message rather than per-process: the endpoint is
// re-resolved once per generated artifact (~50 per run), so an unguarded warning
// would bury itself, while a single process-wide flag would hide a second,
// different misconfiguration behind the first.
//
// The key set is therefore bounded by the number of DISTINCT bad values a
// process sees, not by call count — one or two in practice, since the
// environment is fixed for a process's lifetime.
//
// os.Stderr is read at call time so a test can redirect it.
func warnOnce(msg string) {
	warnedMu.Lock()
	seen := warned[msg]
	warned[msg] = true
	warnedMu.Unlock()
	if seen {
		return
	}
	fmt.Fprintln(os.Stderr, warnPrefix+msg)
}

// redactedValue replaces a rejected value that may carry credentials. It names
// the fact of redaction rather than staying silent, so an operator reading the
// warning knows the value was seen and deliberately withheld.
const redactedValue = "<value withheld: may carry credentials>"

// renderRejected formats a rejected env value for stderr: quoted when it cannot
// plausibly carry a credential, replaced by redactedValue when it can.
//
// A rejected value is echoed because the operator needs it to spot the typo, but
// the reason a value reaches this path at all is that it is NOT the shape the
// variable expects — and the likeliest wrong shape is the neighbouring variable's
// URL, which can carry userinfo. Echoing it verbatim would put on stderr exactly
// what the EnvBaseURL diagnostic already refuses to print.
func renderRejected(v string) string {
	if mayCarryCredentials(v) {
		return redactedValue
	}
	return strconv.Quote(v)
}

// mayCarryCredentials reports whether a value contains any marker of a shape that
// can embed a secret. It is deliberately over-inclusive: a false positive costs
// one diagnostic's precision, a false negative puts a credential on stderr.
//
// The markers, each closing one credential-bearing shape:
//
//   - "@"      — the userinfo delimiter, in both "user:pass@host" and
//     "scheme://user:pass@host".
//   - "/"      — any scheme-ful ("://"), scheme-relative ("//user:pass@host") or
//     path-bearing form; a URL is the shape that carries userinfo and
//     "?token=" alike.
//   - ANY percent-escape — a "%" followed by two hex digits, case-folded. The
//     shapes this closes are the percent-encoded "@" ("%40") and ":" ("%3A"), so
//     a fully-encoded "user%3Apass%40host" is not waved through. Matching the
//     GENERAL form rather than those two literals is what makes the rule hold at
//     every encoding depth: "%25" — an encoded "%" — is itself an escape, so a
//     doubly-encoded "user%253Apass%2540host", which contains neither "%40" nor
//     "%3A" anywhere, is caught by the same test, and so is every deeper layer.
//   - a ":" whose right-hand side is not a plausible port — this is what separates
//     the "host:port" mistake the diagnostic exists to explain (echoed) from a
//     bare "user:pass" pair (withheld). Reading only the LAST colon does not settle
//     that question: in "user:pass:7061" the last colon really is a plausible port
//     separator, while the FIRST one — whose right-hand side is "pass:7061" — is
//     exactly the marker described here. So a second clause withholds any value
//     carrying MORE THAN ONE colon unless it OPENS with "[" — the bracket being the
//     form in which a run of colons is most idiomatically an IPv6 literal rather
//     than a smuggled userinfo pair. The guard is a cheap prefix test, not an IPv6
//     parse, so it spares any "["-opening value whether or not a literal follows.
//     That leaves a residual hole — "[a:pass:7061" keeps its echo — which is
//     consistent with the contract rather than an oversight in it: no realistic
//     paste lands there, because a real URL authority puts userinfo BEFORE the "@"
//     and outside any brackets, so a genuine credential-bearing value carries an
//     "@" or a "/" and is withheld by the first clause regardless.
//
// Honest boundary (§11.4.6): this catches STRUCTURE, not secrecy. A secret with no
// structure at all — a bare "sk-live-…" token pasted into either variable — is
// textually indistinguishable from a mistyped hostname and is still echoed;
// suppressing it would mean echoing nothing, which removes the diagnostic. So is a
// "user:1234" whose all-digit right-hand side is a numeric password rather than a
// port. Both are documented rather than papered over.
//
// The multi-colon clause spends precision in the over-inclusive direction the rest
// of this predicate already chooses, and two shapes pay for it — differently.
//
// An UNBRACKETED IPv6 literal carrying a port ("::1:7061") is ACCEPTED by
// normalizeHost, so it never reaches the host diagnostic at all; only the PORT
// diagnostic, which rejects it, now withholds it where it used to quote it. That
// costs ONE lost echo, on a shape an operator would more idiomatically bracket.
//
// A multi-colon value that is NOT a parseable address ("1:2:3", "foo:1:2") is
// rejected by normalizeHost as well, so it reaches BOTH diagnostics and loses its
// echo in BOTH — the larger of the two costs, and the honest count.
//
// In either case the alternative — trusting a lone trailing port to vouch for every
// colon before it — is what put a credential on stderr.
//
// # Term → falsifier map
//
// Every decision this predicate makes, the mutation that negates that decision, and
// the fixture whose failure reports it. MEASURED — each mutation was applied to this
// file and the package suite re-run — not derived by reading.
//
// The map exists because the same gap recurred at four depths, each round closing
// one layer and exposing the next: first the clause-1 markers, then the colon
// clause's terms, then the hex character class below, then the scan loops' index
// positions. Every time the cause was identical — the fixtures ENTERED the term from
// every side but no single value ISOLATED it, so the term could be deleted with the
// suite still green. Coverage of a term is not falsification of it.
//
// The fourth round showed why "enumerate the whole term set at once" is not something
// anyone can do by eye: what COUNTS as a term is itself a rule, and a rule gets applied
// where it is obvious and skipped where it is not. So the map is built by applying the
// rules below to every site each one names, and each rule carries its site count — an
// omission is then arithmetic rather than judgement:
//
//	R1 — every SCAN LOOP contributes THREE positional terms: its START index, its END
//	     condition, and its STEP. Sites: the 2 loops in this predicate
//	     (containsPercentEscape, isAllDigits) = 6 terms. Earlier revisions listed
//	     exactly ONE of the six — containsPercentEscape's END bound — and four of the
//	     five unlisted terms proved un-killed, every one of them a leak.
//	R2 — every CLAUSE contributes one term per MEMBER plus a clause-present term.
//	     Sites: clauses 1-4 = 4 clause-present terms, plus clause 1's 4 members.
//	R3 — every CHARACTER-RANGE test contributes one term per BOUND. Sites:
//	     isLowerHexDigit's two ranges = 4 bounds; isAllDigits' one range = 2 bounds.
//	R4 — every INDEX or COUNT literal contributes one term per comparison it can be
//	     shifted across. Sites: the colon threshold (3), the LastIndex guard and its
//	     boundary (2), the split point (1), the two lookahead offsets (2).
//
// R2, R3 and R4 were re-audited site by site when R1 was written — a rule applied only
// where it is convenient is precisely the failure this map exists to end — and all
// three were already uniform. The audit also turned up two boolean-operator mutants
// outside the four rules (isLowerHexDigit's "||" and isAllDigits' "||", each flipped
// to "&&"); both are killed by existing fixtures, so they are covered rather than
// gaps. R1 was the only uneven rule.
//
// "unique" = the named fixture is the only one whose failure reports that mutation.
// "shared/N" = N do, unavoidably, and the designated prover is named. N counts distinct
// fixture VALUES rather than subtests, because one value is usually exercised by both
// the host and the port diagnostic. Sharing is structural wherever a value must first
// REACH a clause in order to probe one of its terms: such a value necessarily also
// answers for the clause's existence.
//
//	clause 1 — strings.ContainsAny(v, "@/?#")
//	  "@" member        drop from set      token-as-username authority       unique
//	  "/" member        drop from set      bare-path paste                   unique
//	  "?" member        drop from set      query-string token, no scheme     unique
//	  "#" member        drop from set      schemeless fragment token         unique
//	  clause present    ContainsAny(v,"")  shared/4 — prover: the "@" fixture
//
//	case fold — strings.ToLower(v)
//	  the fold          lower := v         uppercase escape only             unique
//
//	clause 2 — containsPercentEscape(lower)
//	  clause present    disable clause     shared/8 — prover: percent-encoded userinfo
//	  generality        two-literal lookup shared/4 — prover: double-percent-encoded
//	  "%" sentinel      match '!' instead  shared/8 — prover: percent-encoded userinfo
//	  lookahead byte 1  drop s[i+1] check  host%if1 (ordinary)               unique
//	  lookahead byte 2  drop s[i+2] check  host%eth0 (ordinary)              unique
//	  loop START (R1)   i := 0 → i := 1    escape at index 0                 unique
//	  loop END   (R1)   i+2 < → i+3 <      percent-encoded, escape terminal  unique
//	  loop STEP  (R1)   i++ → i += 2       shared/3 — prover: uppercase escape only
//
//	isLowerHexDigit — the two ranges, four bounds
//	  digit low  "0"    c >= '1'           encoded-@ authority               unique
//	  digit high "9"    c <= '8'           over-encoded digit                unique
//	  letter low "a"    c >= 'b'           shared/2 — prover: uppercase escape only
//	  letter high "f"   c <= 'e'           encoded "?"                       unique
//
//	clause 3 — Count(v,":") > 1 && !HasPrefix(v,"[")
//	  threshold vs >0   > 1 → > 0          shared/3 — prover: agent.internal:7061 (ord.)
//	  threshold vs >2   > 1 → > 2          shared/2 — prover: user:pass:port paste
//	  threshold vs ==2  > 1 → == 2         three-colon paste                 unique
//	  "[" anchor        drop the guard     [::1]:7061 (ordinary)             unique
//	  "[" anchored      HasPrefix→Contains bracket mid-string                unique
//	  clause present    threshold > 99     shared/3 — prover: user:pass:port paste
//
//	clause 4 — LastIndex(v,":"); i >= 0 && !isAllDigits(v[i+1:])
//	  split point       v[i+1:] → v[i:]    shared/4 — prover: agent.internal:7061 (ord.)
//	  i >= 0 guard      drop the guard     shared/7 — prover: not-a-port (ordinary)
//	  i >= 0 boundary   i >= 0 → i > 0     curl -u :<token>                  unique
//	  the negation      !isAllDigits → is  shared, both directions at once
//	  clause present    i >= 99            shared/6 — prover: bare user:pass pair
//
//	isAllDigits
//	  empty string      "" → true          curl -u <token>:                  unique
//	  digit low  "0"    s[i] < '1'         shared/4 — prover: agent.internal:7061 (ord.)
//	  digit high "9"    s[i] > '8'         agent.internal:9090 (ordinary)    unique
//	  loop START (R1)   i := 0 → i := 1    mistyped port, stray leading      unique
//	  loop END   (R1)   < len(s) → -1      mistyped port, stray trailing     unique
//	  loop STEP  (R1)   i++ → i += 2       mistyped port, stray odd index    unique
//
//	renderRejected dispatch (the caller)
//	  redact branch     always redact      shared/14 ordinary controls
//	  quote branch      never redact       shared/all credential fixtures
//
//	clause ORDER
//	  any permutation   reorder clauses    NO KILLER — equivalent mutant, see below
//
// ARITHMETIC. The table holds 38 rows and stands for 38 distinct named mutations, one
// per row: 37 were each applied to this file and the suite re-run, and the 38th is the
// clause-ORDER row, which stands for all 23 non-identity permutations of the four
// clauses and is argued equivalent below rather than measured against a killer. No row
// abbreviates more than one mutation, so the count and the table check each other.
//
// Two cautions this restatement exists to prevent, both learned from the revision it
// replaces. First, an earlier revision headlined "37 mutations / 1 equivalent survivor"
// over a 33-row table; the two numbers cannot be reconciled from the artifact, and an
// unreconcilable headline is exactly the unverified figure this whole exercise removes,
// so it is restated rather than carried forward. Second, a shared/N count silently goes
// stale when a later round adds a fixture that happens to fall into an existing kill
// set: clause 4's "i >= 0 guard" read shared/6 until host%if1 was added for the
// lookahead-byte-1 row, and nothing re-measured it. Every N above was re-measured
// against the current fixture set in the round that added R1.
//
// The clause-order mutant is EQUIVALENT rather than a gap: each clause is a pure
// read of v that returns the same constant true, none has a side effect, and the
// only computed value (lower) is derived solely from v. Any permutation therefore
// decides every input identically, so no fixture can distinguish one — the absence
// of a killer is a proof about the code, not a hole in the suite.
//
// Honest boundary (§11.4.6): this map is closed over the terms of THIS predicate as
// written. It says nothing about markers the predicate does not implement — a
// credential shape carrying none of these structures is still echoed, exactly as the
// boundary note above records.
//
// R1 has a THIRD site in this package, outside the predicate and so outside this
// table: the loop in Host() that forwards hostFallbackWarnings' output to warnOnce.
// It was enumerated under the same rule anyway, since a rule applied only where the
// table already reaches is the failure R1 exists to end. Its START and END terms were
// already killed by the existing stderr tests; its STEP term was not — a loop stepping
// by two emits the first of the two warnings and drops the second, which no test
// noticed. TestHostEmitsEveryWarningWhenBothFire closes it.
func mayCarryCredentials(v string) bool {
	if strings.ContainsAny(v, "@/?#") {
		return true
	}
	lower := strings.ToLower(v)
	if containsPercentEscape(lower) {
		return true
	}
	if strings.Count(v, ":") > 1 && !strings.HasPrefix(v, "[") {
		return true
	}
	if i := strings.LastIndex(v, ":"); i >= 0 && !isAllDigits(v[i+1:]) {
		return true
	}
	return false
}

// containsPercentEscape reports whether s holds a percent-escape: a "%" followed
// by two hex digits.
//
// s MUST already be lower-cased. Only lowercase hex letters are recognised, so
// the caller's fold is what makes an uppercase "%3A" match — the fold is part of
// this predicate, not decoration around it.
//
// Deliberately narrower than "contains a %": a lone "%" is ordinary text in a
// mistyped host (the zone-ID marker in "host%eth0", for one), and withholding
// those would cost the diagnostic for a whole class of typo that carries no
// structure at all.
func containsPercentEscape(s string) bool {
	for i := 0; i+2 < len(s); i++ {
		if s[i] == '%' && isLowerHexDigit(s[i+1]) && isLowerHexDigit(s[i+2]) {
			return true
		}
	}
	return false
}

// isLowerHexDigit reports whether c is an ASCII digit or a lowercase hex letter.
func isLowerHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}

// isAllDigits reports whether s is non-empty and made only of ASCII digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// hostFallbackWarnings returns every warning implied by an EnvHost value, given
// whether EnvBaseURL carries a usable value.
//
// Pure by design — the caller supplies the raw inputs — so every branch is
// exercised without touching process state or capturing stderr. It re-derives
// whether the host resolves rather than trusting a caller precondition, so it is
// correct for any input and cannot warn about a host that was in fact used.
//
// Both conditions can hold at once (a malformed host WITH a base URL set), and
// then both are reported: they are independent facts and the operator needs
// each. A malformed host is treated as "does not resolve" for the second
// warning because it produces exactly the same outcome as an unset one.
func hostFallbackWarnings(rawHost string, baseURLSet bool) []string {
	if _, ok := normalizeHost(rawHost); ok {
		// The host resolved, so nothing fell back and there is nothing to warn
		// about — not even with a base URL also set, which is the documented way
		// to make one injection reach every generated artifact.
		return nil
	}

	var out []string

	if trimmed := strings.TrimSpace(rawHost); trimmed != "" {
		out = append(out, fmt.Sprintf(
			"%s=%s is not a usable URL host, so the placeholder host %q is used instead. "+
				"This variable takes a BARE host — no scheme, path, query, fragment, userinfo or port. "+
				"A colon is accepted only inside a genuine IPv6 literal, so a \"host:port\" value is "+
				"rejected rather than re-interpreted, and a value carrying \"/\", \"?\", \"#\" or \"@\" is "+
				"rejected for the same reason: appending the port after one of those puts it in the "+
				"wrong URL component and the address silently points elsewhere. "+
				"Use %s to inject host and port as one string.",
			EnvHost, renderRejected(trimmed), DefaultHost, EnvBaseURL))
	}

	if baseURLSet {
		// Deliberately does NOT echo the base URL: a URL can carry userinfo, and
		// diagnostics must not put credentials on stderr. A rejected host or port
		// is echoed only after renderRejected clears it — the same value pasted
		// into the wrong variable is the same secret.
		out = append(out, fmt.Sprintf(
			"%s is set but %s does not resolve, so this endpoint falls back to the placeholder host %q. "+
				"%s is honoured only where the endpoint resolves through DefaultBaseURL() — the Crush "+
				"provider base_url — while CLI-agent MCP, skill and extension URLs resolve through "+
				"Host()+Port() and will point at the placeholder. Set %s (and %s) alongside %s so one "+
				"injection reaches every generated artifact.",
			EnvBaseURL, EnvHost, DefaultHost, EnvBaseURL, EnvHost, EnvPort, EnvBaseURL))
	}

	return out
}

// portFallbackWarning returns the warning implied by an EnvPort value, or "" when
// there is nothing to warn about. Pure, for the same reason as
// hostFallbackWarnings.
//
// An unset or blank value is the documented zero-configuration path, not a
// mistake, and is silent.
func portFallbackWarning(rawPort string) string {
	trimmed := strings.TrimSpace(rawPort)
	if trimmed == "" {
		return ""
	}
	if _, ok := parsePort(trimmed); ok {
		return ""
	}
	return fmt.Sprintf(
		"%s=%s is not an integer in the range 1-65535, so the placeholder port %d is used instead. "+
			"A valid %s is still honoured — host and port fall back independently.",
		EnvPort, renderRejected(trimmed), DefaultPort, EnvHost)
}

// Host returns the injected HelixAgent host, or DefaultHost when unset, blank,
// or malformed.
//
// "Malformed" specifically includes the plausible operator mistake of pasting
// something that is not a BARE host. Two shapes, one rule:
//
//   - the port in the host variable (HELIX_AGENT_HOST="agent.internal:7061") — a
//     colon is only accepted in a genuine IPv6 literal;
//   - a value carrying "/", "?", "#" or "@" (HXC-269) — a path, query, fragment
//     or userinfo. Appending ":<port>" after any of those puts the port in the
//     wrong URL component, so the address parses cleanly and points somewhere
//     else entirely. See normalizeHost for the measured enumeration.
//
// Rejecting rather than re-interpreting keeps this from silently emitting a
// nonsense host into every generated config, and matches Port's
// fallback-on-malformed-input philosophy. A consumer that means "host and port
// together" has EnvBaseURL for that.
//
// Every fallback taken here after the consumer injected SOMETHING warns once on
// stderr; see the silent-fallback diagnostics block above.
//
// Two warnings can be due at once (a malformed host WITH a base URL set), and EVERY
// one is forwarded — dropping either would cost the operator an independent fact
// hostFallbackWarnings deliberately reports separately. That is a property of the
// loop below, not of the pure function, so it is pinned by its own wiring test
// (TestHostEmitsEveryWarningWhenBothFire) rather than by the table tests.
func Host() string {
	raw := os.Getenv(EnvHost)
	if h, ok := normalizeHost(raw); ok {
		return h
	}
	for _, w := range hostFallbackWarnings(raw, injectedBaseURL() != "") {
		warnOnce(w)
	}
	return DefaultHost
}

// unbracket trims surrounding whitespace and strips one layer of IPv6 literal
// brackets, so "[::1]" becomes "::1".
//
// It exists because net.JoinHostPort brackets UNCONDITIONALLY whenever the host
// contains a colon: handing it an already-bracketed literal emits "[[::1]]:389",
// which net.SplitHostPort then rejects outright ("missing port in address") and
// url.Parse reads as an invalid IP-literal. Both address builders in this package
// end in JoinHostPort, so both need this trim, and it lives here — once — rather
// than being restated at each of them (§11.4.74: one chokepoint, no parallel copy
// that can drift out of agreement with the other).
//
// One layer only: "[[::1]]" is not a host an operator can mean, so a repeated
// strip would be inventing an interpretation rather than accepting one.
func unbracket(host string) string {
	host = strings.TrimSpace(host)
	if len(host) > 1 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}
	return host
}

// DialAddress builds a "host:port" address for net.Dial-style consumers
// (net.Dial, smtp.SendMail, tls.Dial, ldap.Dial), bracketing an IPv6 literal per
// RFC 3986 §3.2.2 so "fe80::1" + port 389 yields "[fe80::1]:389" and not the
// unusable "fe80::1:389" that fmt.Sprintf("%s:%d") produces. An already-bracketed
// host is left alone rather than double-bracketed.
//
// # Why this is NOT BaseURL
//
// Three deliberate differences, each of which would be a defect if borrowed from
// the URL path:
//
//  1. NO placeholder fallback. BaseURL substitutes DefaultHost/DefaultPort for an
//     unusable value, which is right for the HelixAgent endpoint this package
//     resolves. A dial address belongs to a service this package does not own — an
//     LDAP directory, an SMTP relay — so substituting the HelixAgent placeholder
//     would silently redirect a mail or directory client to the placeholder HOST.
//     Note what that does and does not mean: BaseURL falls back on host and port
//     INDEPENDENTLY, and DefaultPort replaces only a port outside 1-65535, so the
//     usual outcome is this service's OWN port on the WRONG machine — a connection
//     that may well succeed against something else entirely, which is worse than
//     one that fails. The host and port are passed through as given instead; a
//     wrong one fails loudly at dial time, which is the correct outcome and the
//     honest one (§11.4.6).
//  2. NO RFC 6874 zone encoding. BaseURL percent-encodes a zone ID
//     ("fe80::1%eth0" -> "fe80::1%25eth0") because a URL must stay parseable. The
//     net package expects the zone RAW, so encoding it here would break exactly
//     the addresses it was meant to preserve.
//  3. NO scheme. The result is an authority, not a URL.
//
// What it DOES share with BaseURL is the bracketing rule and the unbracket trim
// above — the two facts that were getting reimplemented per call site.
func DialAddress(host string, port int) string {
	return net.JoinHostPort(unbracket(host), strconv.Itoa(port))
}

// normalizeHost validates and canonicalises a host for URL construction.
// It reports false when the input cannot be a valid URL host.
//
// Returned IPv6 hosts carry any zone ID percent-encoded per RFC 6874
// ("fe80::1%eth0" -> "fe80::1%25eth0") so the resulting URL stays parseable.
func normalizeHost(host string) (string, bool) {
	host = unbracket(host)
	if host == "" {
		return "", false
	}

	// HXC-269: a host is a BARE host — never a URL fragment carrying a path,
	// query, fragment or userinfo. Each of the four bytes below is an RFC 3986
	// gen-delim that ENDS the host component, so appending ":<port>" after one of
	// them puts the port in the wrong component entirely. MEASURED over the whole
	// printable ASCII range against BaseURL + url.Parse: these four are the
	// complete set of bytes that normalizeHost accepted while producing a URL that
	// still PARSES and is therefore silently wrong —
	//
	//	"?"  "hx?yz" -> "http://hx?yz:7061"  host "hx", port EMPTY, query "yz:7061"
	//	"#"  "hx#yz" -> "http://hx#yz:7061"  host "hx", port EMPTY, fragment "yz:7061"
	//	"/"  "hx/yz" -> "http://hx/yz:7061"  host "hx", port EMPTY, path "/yz:7061"
	//	"@"  "hx@yz" -> "http://hx@yz:7061"  host "yz" — the WRONG host — userinfo "hx"
	//
	// The first three lose the port; the fourth keeps it and redirects the host,
	// which is worse. All four are silent: every standard parser accepts the
	// result, so nothing downstream can notice.
	//
	// REJECT rather than strip or re-interpret, for three reasons. (1) Consistency:
	// this function ALREADY rejects the sibling mistakes — a "host:port" value and a
	// scheme-ful URL — rather than re-interpreting them, and Host's doc gives the
	// reason ("keeps this from silently emitting a nonsense host into every
	// generated config"); these four are the same mistake through a different byte.
	// (2) Stripping would silently discard what the operator wrote, which is the
	// very silent-misdirection class this rejection exists to end. (3) Preserving
	// the extra components is not even expressible here: BaseURL returns
	// "scheme://authority" and callers JoinPath onto it, so a preserved query would
	// produce "http://h:7061?x" and then "http://h:7061?x/v1" — broken either way.
	// A consumer that means host-and-port-together has EnvBaseURL, which the
	// rejection message names.
	//
	// Rejecting also closes a credential leak this same acceptance opened, and that
	// is not a side benefit but the sharper half of the defect: because the value
	// was ACCEPTED, no diagnostic fired, so a "gw.example?token=<secret>" pasted
	// into EnvHost was written VERBATIM into every generated artifact on disk.
	// Rejected, it now reaches the host diagnostic, where renderRejected withholds
	// it — all four bytes are clause-1 markers of mayCarryCredentials, so the
	// rejection message announces the redaction instead of echoing the secret.
	//
	// Honest boundary (§11.4.6): this closes the SILENT class only. Eight further
	// bytes (space, "[", "\\", "^", "`", "{", "|", "}") are still accepted here and
	// make url.Parse FAIL — a loud, visible error rather than a wrong address, so a
	// different defect class and deliberately out of scope; see the report.
	if strings.ContainsAny(host, "/?#@") {
		return "", false
	}

	bare, zone := host, ""
	if i := strings.Index(host, "%"); i >= 0 {
		bare, zone = host[:i], host[i+1:]
		// Tolerate a zone that already carries the RFC 6874 "%25" delimiter, but
		// only when something remains after it: a bare "%25" is the genuine
		// numeric zone index 25, and trimming it away would drop the zone
		// entirely and silently emit a different address. The residual
		// ambiguity is inherent to accepting both forms — "%251" could be
		// encoded zone "1" or literal zone "251" — and is resolved in favour of
		// the encoded reading, which is what round-tripping our own output needs.
		if rest := strings.TrimPrefix(zone, "25"); rest != "" {
			zone = rest
		}
	}

	if strings.Contains(bare, ":") {
		// Only a real IPv6 literal may contain a colon. Anything else is a
		// host:port mistake or garbage.
		if net.ParseIP(bare) == nil {
			return "", false
		}
		if zone != "" {
			return bare + "%25" + zone, true
		}
		return bare, true
	}

	// A zone ID is only meaningful for IPv6.
	if zone != "" {
		return "", false
	}
	return bare, true
}

// Port returns the injected HelixAgent port, or DefaultPort when unset, blank,
// non-numeric, or outside the valid TCP range. A malformed injection falls back
// rather than panicking or emitting an unusable ":0" URL — the generated config
// stays syntactically valid and the caller can still override explicitly. A
// set-but-invalid value warns once on stderr; an unset one is silent.
func Port() int {
	raw := os.Getenv(EnvPort)
	if p, ok := parsePort(raw); ok {
		return p
	}
	if w := portFallbackWarning(raw); w != "" {
		warnOnce(w)
	}
	return DefaultPort
}

// parsePort is the single definition of "a usable injected port", shared by
// Port and portFallbackWarning so the value used and the value warned about can
// never disagree. ok is false for absent, blank, non-numeric, and out-of-range.
func parsePort(raw string) (int, bool) {
	p, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || p < 1 || p > 65535 {
		return 0, false
	}
	return p, true
}

// BaseURL builds "http://host:port" for an explicitly supplied host and port.
//
// IPv6 literals are bracketed per RFC 3986 §3.2.2, so "::1" yields
// "http://[::1]:8100" and not the malformed "http://::1:8100" that naive
// fmt.Sprintf("http://%s:%d") produces. An already-bracketed host is left
// alone. A blank host falls back to DefaultHost; a port outside the valid TCP
// range falls back to DefaultPort.
//
// A host that is not a BARE host — one carrying a port, scheme, path, query,
// fragment or userinfo — also falls back to DefaultHost rather than being
// re-interpreted (HXC-269; see normalizeHost). That fallback is what guarantees
// this function's contract: the returned port is ALWAYS the port argument, in
// the authority component where a parser will find it, never displaced into a
// query, fragment or path by a delimiter smuggled in through the host.
func BaseURL(host string, port int) string {
	normalized, ok := normalizeHost(host)
	if !ok {
		normalized = DefaultHost
	}
	if port < 1 || port > 65535 {
		port = DefaultPort
	}
	return "http://" + net.JoinHostPort(normalized, strconv.Itoa(port))
}

// DefaultBaseURL resolves the endpoint from the environment per the package
// resolution order and returns it as a base URL with no trailing slash.
func DefaultBaseURL() string {
	if v := injectedBaseURL(); v != "" {
		return v
	}
	return BaseURL(Host(), Port())
}

// injectedBaseURL returns the EnvBaseURL value as this package would use it, or
// "" when it is absent or unusable. It is the single definition of "a base URL
// is set", shared by DefaultBaseURL and the Host diagnostics so the resolver and
// the warning can never disagree about it.
//
// A value that is nothing but separators (e.g. "/") trims to the empty string,
// which would emit relative "/v1" provider URLs; it counts as unset.
func injectedBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv(EnvBaseURL)), "/")
}

// JoinPath appends a path segment to a base URL without doubling or dropping
// the separator, so a base URL injected with or without a trailing slash
// produces the same result.
func JoinPath(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if path == "" {
		return baseURL
	}
	return baseURL + "/" + strings.TrimLeft(path, "/")
}
