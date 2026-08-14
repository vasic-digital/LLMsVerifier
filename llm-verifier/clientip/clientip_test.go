package clientip

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestResolveCensus is the canonical case table for this package's single
// implementation of the caller-identity resolution algorithm. Every
// delegating consumer (api/middleware.go's getClientIP,
// security/client_ip_trust.go's resolveClientIP,
// enhanced/enterprise/api.go's EnterpriseAPI.getClientIP) mirrors a subset
// of these cases in its own package for call-site-level regression
// guarding; this table is the ground truth they are all mirrored from.
func TestResolveCensus(t *testing.T) {
	cases := []struct {
		name            string
		remoteAddr      string
		xff             string
		xffPresentEmpty bool
		xri             string
		xriPresentEmpty bool
		// F11: repeated header INSTANCES (separate wire lines). Non-empty
		// takes precedence over the single-value xff/xri fields above.
		xffInstances   []string
		xriInstances   []string
		trustedProxies string
		want           string
		outcome        string
	}{
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
			outcome:    "control: a non-IP host:port form is unaffected",
		},
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
			outcome:    "boundary: unparseable-as-host:port string used as-is",
		},
		{
			name:       "remoteaddr_empty_host_with_port",
			remoteAddr: ":12345",
			want:       ":12345",
			outcome:    "boundary: a port with no host portion carries no identity, preserved unmodified",
		},
		{
			name:       "xff_untrusted_peer_ignored",
			remoteAddr: "203.0.113.9:1111",
			xff:        "9.9.9.9",
			want:       "203.0.113.9",
			outcome:    "no trusted-proxy allowlist configured -> XFF ignored, real peer used",
		},
		{
			name:       "xri_untrusted_peer_ignored",
			remoteAddr: "203.0.113.9:2222",
			xri:        "9.9.9.9",
			want:       "203.0.113.9",
			outcome:    "no trusted-proxy allowlist configured -> X-Real-IP ignored, real peer used",
		},
		{
			name:           "xff_trusted_peer_bare_ip_match",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.7",
			trustedProxies: "172.20.0.5",
			want:           "198.51.100.7",
			outcome:        "allowlist bare-IP entry matches the peer exactly -> header honoured",
		},
		{
			name:           "xff_trusted_peer_cidr_match",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.7",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.7",
			outcome:        "allowlist CIDR entry contains the peer -> header honoured",
		},
		{
			name:           "xff_multiple_ips_rightmost_untrusted_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "198.51.100.7, 172.20.0.5, 10.0.0.1",
			trustedProxies: "172.20.0.0/16",
			want:           "10.0.0.1",
			outcome:        "walking right to left, the rightmost entry is not itself trusted, so it wins immediately",
		},
		{
			name:           "xff_attacker_prepended_forged_real_client_wins",
			remoteAddr:     "172.20.0.5:443",
			xff:            "8.8.8.8, 203.0.113.9",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.9",
			outcome:        "leftmost-selection regression guard: attacker-forged leftmost entry discarded",
		},
		{
			name:           "xff_two_trusted_hops_client_beyond_both",
			remoteAddr:     "172.20.0.9:443",
			xff:            "203.0.113.50, 172.20.0.5",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.50",
			outcome:        "two chained trusted proxies are both walked past",
		},
		{
			name:           "xff_malformed_entry_beyond_trusted_hop_falls_through",
			remoteAddr:     "172.20.0.5:443",
			xff:            "not-an-ip, 172.20.0.5",
			trustedProxies: "172.20.0.0/16",
			want:           "172.20.0.5",
			outcome:        "a non-IP entry beyond a trusted hop -> walk stops, falls through to RemoteAddr",
		},
		{
			name:           "xff_trailing_empty_entry_falls_through",
			remoteAddr:     "172.20.0.5:443",
			xff:            "203.0.113.5,",
			trustedProxies: "172.20.0.0/16",
			want:           "172.20.0.5",
			outcome:        "trailing comma yields an empty rightmost entry -> falls through",
		},
		{
			name:           "xff_leading_empty_entry_ignored_rightmost_used",
			remoteAddr:     "172.20.0.5:443",
			xff:            ",10.0.0.1",
			trustedProxies: "172.20.0.0/16",
			want:           "10.0.0.1",
			outcome:        "a leading empty entry is harmless under right-to-left walking",
		},
		{
			name:           "xff_bracketed_ipv6_entry_trusted_peer",
			remoteAddr:     "172.20.0.5:443",
			xff:            "[2001:db8::99]",
			trustedProxies: "172.20.0.0/16",
			want:           "2001:db8::99",
			outcome:        "a header-supplied bracketed IPv6 literal is normalised unbracketed",
		},
		{
			name:           "xff_trusted_ipv6_peer_bracketed_allowlist_entry",
			remoteAddr:     "[2001:db8::5]:443",
			xff:            "203.0.113.77",
			trustedProxies: "[2001:db8::5]",
			want:           "203.0.113.77",
			outcome:        "a bracketed IPv6 allowlist entry matches an IPv6 peer at the same address",
		},
		{
			name:            "xff_header_present_but_empty_value_treated_as_absent",
			remoteAddr:      "172.20.0.5:443",
			xffPresentEmpty: true,
			xri:             "198.51.100.20",
			trustedProxies:  "172.20.0.0/16",
			want:            "198.51.100.20",
			outcome:         "an explicitly-empty X-Forwarded-For header is treated as absent",
		},
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
		// headers on a mismatch. Round 9 proved it a live regression on
		// UNFORGED traffic, and round 10 removed it: the premise it rested
		// on (a proxy managing both derives both from its own peer) is
		// false for ingress-nginx with realip active and for every
		// mixed-vendor chain, and the guard was SYMMETRIC by construction
		// so it also fired on the mirror topology where the walk had
		// already produced the correct answer. See Resolve's "# Header
		// PRECEDENCE, and its threat model" doc comment for the full
		// reasoning and cited sources. The rows below pin BOTH the
		// restored behaviour AND the one exposure precedence leaves open,
		// so neither can change silently.
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
		// §1.1 FALSIFIABILITY of xriPresentEmpty — an earlier claim here,
		// RETRACTED. The rows below SET xriPresentEmpty, which was declared
		// and harness-wired in all four tables while ZERO rows set it (its
		// twin xffPresentEmpty was set everywhere) — §11.4.124 dead
		// scaffolding, now wired.
		//
		// The retracted claim was that the FIELD could never be made
		// falsifiable, reasoning that Resolve treats a present-but-empty
		// X-Real-IP and an ABSENT one identically, so no assertion on
		// Resolve's OUTPUT could separate them. That was FALSE, and
		// measuring it is what surfaced a live capability. Driven through
		// Go's real HTTP parser with LLM_VERIFIER_TRUSTED_PROXIES set to
		// 172.20.0.0/16 and the peer at 172.20.0.5:
		//
		//	ABSENT        + one valid X-Real-IP instance -> 8.8.8.8
		//	PRESENT-EMPTY + one valid X-Real-IP instance -> 172.20.0.5 (peer)
		//
		// They differ. A present-but-empty X-Real-IP arrives as its own
		// header INSTANCE, so alongside a valid one it pushes
		// Header.Values("X-Real-IP") to len 2 and trips the multi-instance
		// refusal in Resolve's X-Real-IP fallback. Only the LONE
		// present-but-empty case is output-indistinguishable from absence.
		// That is the genuine limit, and it is far narrower than what was
		// claimed — it is the same limit the pre-existing
		// xff_header_present_but_empty_value_treated_as_absent row has on
		// the other header, and it predates this change.
		//
		// xri_empty_instance_suppresses_authoritative_xri_and_collapses_onto_peer
		// (below) pins the difference and makes the field falsifiable:
		// deleting the harness's `if tc.xriPresentEmpty` branch flips that
		// row from the peer to 8.8.8.8.
		//
		// What the two rows immediately below pin is unchanged: the property
		// that regressed — an unusable X-Real-IP must not SUPPRESS a valid
		// X-Forwarded-For. Both still fail against a reinstated corroboration
		// guard, which is the mutation they exist to catch.
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
		{
			name:           "xff_peer_not_in_configured_range_ignored",
			remoteAddr:     "203.0.113.200:443",
			xff:            "1.2.3.4",
			trustedProxies: "172.20.0.0/16",
			want:           "203.0.113.200",
			outcome:        "R9-6: a WELL-FORMED allowlist that simply does not contain the peer -- the peer-trust gate refuses the forwarding headers and the caller resolves to its own RemoteAddr. Distinct from the malformed-entry and matching-entry rows: this is the ordinary non-match path, and without it every allowlist row in this table either matched the peer or was malformed",
		},

		// ---- R9-6 (round-9 review finding, closed): rows that existed
		// ONLY in the delegating packages' mirrors ----
		//
		// This table's own doc comment calls it "the ground truth they are
		// all mirrored from". That was FALSE while the mirrors held seven
		// case names this table did not (and this table held none the
		// mirrors lacked) -- a ground-truth table that is a strict SUBSET
		// of what it supposedly grounds. The six below plus
		// xff_peer_not_in_configured_range_ignored above close that gap so
		// the claim is true as written rather than aspirational.
		{
			name:       "both_headers_untrusted_peer_ignored",
			remoteAddr: "203.0.113.9:3333",
			xff:        "9.9.9.9",
			xri:        "8.8.8.8",
			want:       "203.0.113.9",
			outcome:    "neither forwarding header is honoured when the peer is untrusted -- the peer-trust gate runs BEFORE either header is read, so the presence of both changes nothing",
		},
		{
			name:       "same_caller_different_ports_no_headers",
			remoteAddr: "198.51.100.42:33001",
			want:       "198.51.100.42",
			outcome:    "the ephemeral source port is never part of the resolved identity -- two connections from one caller must not fork into two audit identities or two rate-limit buckets",
		},
		{
			name:       "ipv6_shared_first_hextet_distinct_callers_second",
			remoteAddr: "[2001:dead:beef::99]:22222",
			want:       "2001:dead:beef::99",
			outcome:    "a caller sharing ONLY its first hextet with another caller must still resolve to its OWN distinct identity -- guards the hand-rolled colon-split defect class, where an IPv6 literal was truncated at its first colon and every caller in a /16 collapsed into one identity",
		},
		{
			name:           "xff_multiple_ips_with_surrounding_whitespace",
			remoteAddr:     "172.20.0.5:443",
			xff:            " 198.51.100.8 , 172.20.0.5",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.8",
			outcome:        "rightmost entry (172.20.0.5) is allowlisted and skipped; the next entry left is trimmed of surrounding whitespace before use -- the padded form real proxies emit",
		},
		{
			name:           "xff_trusted_peer_multiple_allowlist_entries",
			remoteAddr:     "172.20.5.5:80",
			xff:            "198.51.100.9",
			trustedProxies: "10.0.0.0/8,172.20.0.0/16",
			want:           "198.51.100.9",
			outcome:        "comma-separated allowlist whose SECOND entry matches the peer -> the header is honoured, proving the allowlist split walks past a non-matching first entry",
		},
		{
			name:           "xff_trusted_peer_malformed_allowlist_entry_ignored",
			remoteAddr:     "172.20.5.5:80",
			xff:            "198.51.100.9",
			trustedProxies: "not-a-cidr,172.20.0.0/16",
			want:           "198.51.100.9",
			outcome:        "a malformed allowlist entry is skipped rather than being fatal to the whole allowlist; the remaining valid entry still matches, so one typo in the operator's configuration does not silently untrust every proxy after it",
		},
		{
			name:           "xri_only_trusted_peer",
			remoteAddr:     "172.20.0.5:443",
			xri:            "198.51.100.20",
			trustedProxies: "172.20.0.0/16",
			want:           "198.51.100.20",
			outcome:        "X-Real-IP alone is honoured once the peer is trusted",
		},

		// ---- F1 (post-extraction review finding): the net.ParseIP(xri)
		// validation guard on X-Real-IP, deleted by a reviewer's mutation,
		// left ALL FOUR delegating packages' suites green -- and, from a
		// trusted peer, "X-Real-IP: '; DROP TABLE audit; --" came back as
		// the RESOLVED IDENTITY. This case pins the guard directly, in
		// this package's OWN test suite -- not merely in the three
		// delegating packages' mirrored census tables -- since this is
		// where the guard actually lives. ----
		{
			name:           "xri_non_ip_value_rejected_trusted_peer",
			remoteAddr:     "172.20.0.5:443",
			xri:            "'; DROP TABLE audit; --",
			trustedProxies: "172.20.0.0/16",
			want:           "172.20.0.5",
			outcome:        "F1 guard: a trusted peer's X-Real-IP MUST still be validated with net.ParseIP -- a non-IP value (here, an injection-style string) must never be returned as the resolved identity",
		},

		// ---- F2 (post-extraction review finding): isTrustedProxyPeer's
		// FAIL-CLOSED guard — `if peerIP == nil { return false }` — is the
		// default that makes an UNRECOGNISABLE peer untrusted BEFORE the
		// allowlist is ever consulted. It is the same defect class as F1,
		// one guard over, on the same sink (audit trail + rate-limit key),
		// and a reviewer's mutation flipping it to `return true` left ALL
		// FOUR delegating packages' suites green.
		//
		// The gap was structural, not accidental: the census already
		// carried non-IP peers (hostname_remoteaddr_with_port,
		// malformed_no_colon_remoteaddr, empty_remoteaddr) and already
		// carried forwarding headers — but never the two TOGETHER, and the
		// guard only has an observable effect when a header is present to
		// be wrongly honoured. These rows supply exactly that pairing.
		//
		// Reachability is real: net/http sets RemoteAddr from the accepted
		// conn, so a TCP listener always yields "ip:port" — but a
		// net.Listen("unix", ...) listener yields a non-IP RemoteAddr, and
		// security/security.go's AuditTrail.LogRequest (plus every census
		// here) takes a caller-constructed *http.Request whose RemoteAddr
		// is whatever the caller set. ----
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
		// trusted as a guard.
		//
		// F6 is the most serious of the three: F1, F2, F7 and F8 all fail
		// CLOSED when their guard is removed (an identity is lost or a
		// header ignored), but F6's surviving mutant WIDENS TRUST -- it
		// grants trusted-proxy status to a peer the operator never
		// allowlisted, and then honours that peer's attacker-supplied
		// X-Forwarded-For. ----

		// F6: stripBrackets removes only a MATCHING pair. Its two
		// conjuncts were only ever tested together: M08 (whole function
		// no-op) and M10 (off-by-one) are both CAUGHT, so the function
		// LOOKED covered -- but no census row ever fed it a '['-prefixed
		// literal WITHOUT a closing ']', so the matched-pair requirement
		// itself was untested. Dropping ONLY the `host[len(host)-1] == ']'`
		// conjunct truncates the last character of any such literal: the
		// operator typo "[10.0.0.10" (one missing bracket) becomes the
		// DIFFERENT host "10.0.0.1", which then matches this peer exactly,
		// makes it a trusted proxy, and hands the resolved identity to its
		// forged X-Forwarded-For. Pristine rejects the entry, fails closed,
		// AND warns the operator it is unparseable.
		{
			name:           "xff_present_unterminated_bracket_allowlist_entry_stays_untrusted",
			remoteAddr:     "10.0.0.1:5000",
			xff:            "9.9.9.9",
			trustedProxies: "[10.0.0.10",
			want:           "10.0.0.1",
			outcome:        "F6 guard: an UNTERMINATED bracketed allowlist entry ([10.0.0.10) is not a matching pair, so stripBrackets leaves it alone, it parses as neither IP nor CIDR, and the peer stays UNTRUSTED -- dropping the closing-bracket conjunct would truncate it to the different host 10.0.0.1, match this peer, and honour its forged X-Forwarded-For (9.9.9.9)",
		},

		// F7: normalizeHostLiteral is applied to X-Real-IP, not just to
		// X-Forwarded-For entries. The census carried
		// xff_bracketed_ipv6_entry_trusted_peer for the XFF path but NO
		// X-Real-IP counterpart and NO whitespace row at all, so replacing
		// the whole normalizeHostLiteral(...) call with a raw
		// Header.Get("X-Real-IP") survived every suite. It fails closed
		// (net.ParseIP rejects the un-normalised value, so Resolve falls
		// through), which is why it is not a bypass -- but it silently
		// discards a legitimate caller identity and resolves the caller to
		// the PROXY instead, which is exactly the "forks into a second,
		// distinct identity" outcome stripBrackets' own doc comment exists
		// to prevent, in the opposite direction.
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

		// F8: isTrustedProxyPeer bracket-strips on the
		// net.SplitHostPort-ERROR path, which is the only path a bracketed
		// peer with NO port takes. Structurally the identical miss to F2:
		// the census carried a bracketed-no-port peer
		// (bare_bracketed_ipv6_no_port_degenerate) AND carried headers AND
		// carried allowlists -- but never a bracketed-no-port peer that is
		// ON the allowlist WITH a header present, the only combination in
		// which this strip is observable. Without it the peer literal keeps
		// its brackets, net.ParseIP returns nil, and an explicitly
		// allowlisted proxy silently stops being trusted (fails closed).
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
		// property of every OTHER allowlist in this file, wherever it
		// sits relative to this row: none of them puts
		// a SPACE after a comma. (R9-7: the premise is stated
		// file-wide because the command below scans the WHOLE file --
		// an earlier revision scoped the prose to "every allowlist
		// written ABOVE it" while the command checked everything, so a
		// comma-space row added BELOW would have made the printed
		// command report a count the prose still called correct.) The outer
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
		// This comment is the ORIGIN of the wording copied into the three
		// delegating packages, and three of its earlier revisions
		// justified the row with claims that were either self-invalidating
		// or never checkable. They are recorded here, corrected, because
		// the same shapes are easy to reintroduce:
		//
		//   - A hard-coded COUNT of the rows above ("35 census rows
		//     precede this one", corrected once to 40/41 in the copies)
		//     was wrong AGAIN, here and in all three copies, by the time
		//     the F11 and R7-1 rows had been inserted above it. A number
		//     that must be re-derived on every row addition is a
		//     self-description guaranteed to drift, so it is REMOVED
		//     rather than fixed a third time -- deliberately NOT replaced
		//     with today's correct count, which would drift the same way.
		//     The space-after-comma property stated above does not move
		//     when rows are added and carries the argument by itself.
		//   - "every preceding row configured a SINGLE-entry allowlist" --
		//     stated here as true-in-this-package, and no longer true even
		//     here: "172.20.0.0/16,10.9.9.9" precedes this row. Entry
		//     COUNT was never the load-bearing condition; interior PADDING
		//     is, which is why the premise above is stated in those terms
		//     instead. The "IN THIS PACKAGE" scoping added by F13 stopped
		//     the claim being copied, but could not stop it going stale
		//     where it stood.
		//   - "dropping it survived all four suites" -- an unqualified
		//     claim about a pre-F9 census that no longer exists and, in an
		//     as-yet-uncommitted package, cannot be re-checked against any
		//     history. It is also plainly false of the tree it sits in:
		//     with the `entry = strings.TrimSpace(entry)` inside
		//     isTrustedProxyIP's split loop deleted today, ALL FOUR suites
		//     FAIL -- this row in each of the four census
		//     tables, and here additionally
		//     TestWarnOnMalformedTrustedProxiesEntriesEmitsWarning
		//     -- its "the one VALID allowlist entry must still match the
		//     peer" assertion, Resolve() = "203.0.113.240", want
		//     "198.51.100.60". (Cited by assertion text, not by line
		//     number, deliberately: the first draft of this very note
		//     cited clientip_test.go:763 and the note's own insertion
		//     pushed that assertion to a different line before the edit
		//     was finished.) That warn test asserts the trust decision
		//     for the space-padded allowlist
		//     "f4-emit-probe-not-an-ip, 203.0.113.240,10.0.0.0/33", so its
		//     F10 fixture guards the per-entry trim INDEPENDENTLY of this
		//     row -- which means "never the thing under test" was never
		//     true at suite level for THIS package, whatever the census
		//     did or did not cover. The scoped claim -- never exercised by
		//     the CENSUS TABLE -- is the only half that was ever
		//     verifiable, and it is exactly what this row fixes.
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
		// which fails CLOSED) was already diagnosed by
		// warnOnMalformedTrustedProxiesEntries, while this
		// DANGEROUS-direction one was completely silent, because the entry
		// IS a valid CIDR and nothing looked past that.
		//
		// The BEHAVIOUR is deliberately unchanged -- masking is Go's
		// documented ParseCIDR contract and matches how nginx's
		// set_real_ip_from, iproute2 and most ACL parsers read the same
		// notation, so narrowing or refusing it here would make this
		// package disagree with every other tool reading the operator's
		// own configuration. What closes R7-3 is the operator-facing
		// WARNING -- see warnOnMalformedTrustedProxiesEntries' own doc
		// comment for that argument in full.
		//
		// This row deliberately OVERLAPS
		// TestCIDRAllowlistEntryWithHostBitsIsWidenedAndReported, which
		// asserts the same widening AND the warning AND a
		// correctly-written-network negative control. The split is the
		// same one F10 records: the dedicated test is the unit-level
		// assertion and exists only here, while this row is the
		// cross-package surface -- it is registered identically in all
		// four census tables, so each delegating package proves the
		// widening is observable through ITS OWN entry point rather than
		// inheriting the claim from this package's test. If the masking
		// ever stops, all four rows fail and the warning's premise is gone
		// with them.
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.trustedProxies != "" {
				t.Setenv(TrustedProxiesEnvVar, tc.trustedProxies)
			} else {
				t.Setenv(TrustedProxiesEnvVar, "")
			}

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

			got := Resolve(req)
			if got != tc.want {
				t.Fatalf("%s: Resolve() = %q, want %q", tc.outcome, got, tc.want)
			}
			t.Logf("outcome: %s -> %q", tc.outcome, got)
		})
	}
}

// TestResolveConcurrent is the §11.4.85 concurrent-contention case: N
// goroutines simultaneously resolving distinct callers must each get their
// own correct, uncorrupted identity — no shared mutable state in Resolve
// or its internal isTrustedProxyPeer / warnOnMalformedTrustedProxiesEntries
// steps may leak one caller's resolution into another's.
func TestResolveConcurrent(t *testing.T) {
	t.Setenv(TrustedProxiesEnvVar, "172.20.0.0/16")

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
				peer := fmt.Sprintf("172.20.%d.%d:443", (i/256)%256, i%256)
				forwarded := fmt.Sprintf("198.51.100.%d", i%256)
				req = httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = peer
				req.Header.Set("X-Forwarded-For", forwarded)
				want = forwarded
			} else {
				peer := fmt.Sprintf("203.0.113.%d:%d", i%256, 1000+i)
				req = httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = peer
				req.Header.Set("X-Forwarded-For", "6.6.6.6")
				want = peer[:len(peer)-len(fmt.Sprintf(":%d", 1000+i))]
			}

			got := Resolve(req)
			if got != want {
				errs[i] = fmt.Sprintf("goroutine %d: Resolve() = %q, want %q", i, got, want)
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

// TestIsTrustedProxyIPMalformedAllowlistEntries proves a malformed entry in
// TrustedProxiesEnvVar is skipped (fails closed for that one entry) rather
// than being fatal to the whole allowlist evaluation.
func TestIsTrustedProxyIPMalformedAllowlistEntries(t *testing.T) {
	t.Setenv(TrustedProxiesEnvVar, "not-a-cidr,172.20.0.0/16")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "172.20.5.5:80"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")

	got := Resolve(req)
	if got != "198.51.100.9" {
		t.Fatalf("Resolve() = %q, want %q (a malformed allowlist entry must not be fatal to the "+
			"remaining valid entry)", got, "198.51.100.9")
	}
}

// captureLogOutput runs fn with the standard logger redirected into a
// buffer and returns everything fn logged. Flags and prefix are zeroed so
// the returned text is exactly the message, then every logger setting is
// restored before returning — nested/repeated calls within one test are
// therefore safe.
//
// warnOnMalformedTrustedProxiesEntries writes through the package-level
// `log` logger (log.Printf), which is the only observable channel it has:
// it changes NO trust decision, returns nothing, and mutates only the
// unexported lastWarnedTrustedProxies dedupe cell. Asserting on the emitted
// text is therefore the only way to test it at all.
func captureLogOutput(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	origOut, origFlags, origPrefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&buf)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() {
		log.SetOutput(origOut)
		log.SetFlags(origFlags)
		log.SetPrefix(origPrefix)
	}()

	fn()
	return buf.String()
}

// resetMalformedAllowlistWarnDedupe clears the package-level
// lastWarnedTrustedProxies cell so a test that asserts on the FIRST
// observation of a value starts from a known state.
//
// This is required for re-runnability (§11.4.98): the dedupe cell is
// process-global and deliberately survives individual tests, so without
// this reset a test asserting "this value warns" passes on the first
// `go test` iteration and FAILS on the second under `-count=2` — verified
// by running exactly that before adding this helper. Relying instead on
// each test happening to use a value different from whatever the
// previously-executed test left behind would make these tests
// order-dependent and filter-dependent, which is its own bluff surface.
//
// "" is a safe sentinel rather than a magic value: isTrustedProxyIP
// returns before calling the diagnostic when the configured value is
// empty, so "" is never a value the dedupe could legitimately be holding.
func resetMalformedAllowlistWarnDedupe(t *testing.T) {
	t.Helper()
	lastWarnedTrustedProxies.Store("")
}

// TestWarnOnMalformedTrustedProxiesEntriesEmitsWarning asserts the
// malformed-allowlist diagnostic is ACTUALLY EMITTED, names the offending
// entries, and names the environment variable an operator must fix.
//
// F4 (post-review finding, closed by this test and its dedupe sibling
// below): before these tests, warnOnMalformedTrustedProxiesEntries was
// entirely unasserted — a reviewer's mutation DELETING THE WHOLE CALL to it
// left every suite green, i.e. the entire diagnostic was silently
// deletable. TestIsTrustedProxyIPMalformedAllowlistEntries (above) covers
// only that a malformed entry is non-fatal to the REMAINING valid entry; it
// never observes whether an operator is told anything at all.
//
// This matters beyond tidiness: a malformed entry FAILS CLOSED silently —
// from behaviour alone, "this entry is malformed and was skipped" is
// indistinguishable from "no allowlist is configured yet". This warning is
// the ONLY signal that separates them.
func TestWarnOnMalformedTrustedProxiesEntriesEmitsWarning(t *testing.T) {
	// Deliberately unique to this test: the diagnostic dedupes per distinct
	// configured value process-wide, so a value shared with another test
	// could be silently suppressed here depending on execution order.
	//
	// F10: the VALID entry is deliberately NOT first and deliberately
	// SPACE-PADDED ("…, 203.0.113.240,…"), which is how an operator
	// actually writes a list. Every earlier revision of this fixture wrote
	// the entries comma-separated with NO spaces, so this function's
	// per-entry strings.TrimSpace was never the thing under test and
	// deleting it left all four suites green. Only a space-padded VALID
	// entry can catch it, and only when that entry is not first (the outer
	// TrimSpace on the whole env var already strips the leading one): the
	// untrimmed " 203.0.113.240" fails net.ParseIP, so the diagnostic
	// FALSELY names the operator's one CORRECT entry as malformed and
	// reports three bad entries instead of two — sending them to fix the
	// only line that was right. Both the "contains 2 entr" count assertion
	// and the valid-entry-not-named assertion below catch it.
	//
	// # F14 (post-review finding, closed by this note): F10 is 1x-covered, by design
	//
	// F10 is a fixture STRENGTHENING inside this one warn-diagnostic test —
	// it is NOT a census row, so it is caught by THIS package only, and an
	// earlier "5 rows x 4 tables = 20" framing that swept it in with the
	// census work overstated its reach: under the F10 mutation the api,
	// security and enhanced/enterprise suites stay GREEN (rc=0, all three),
	// because none of them has a warn-diagnostic test at all. The 20
	// row-instances from F6 / F7 (x2) / F8 / F9 are real and were verified
	// row-by-row across the four census tables; F10 is a 21st change with
	// 1x coverage.
	//
	// The decision (recorded, not left implicit): F10 STAYS 1x-covered, and
	// the warn-diagnostic tests are deliberately NOT mirrored into the three
	// delegating packages. warnOnMalformedTrustedProxiesEntries is
	// unexported and lives only here; a delegating package could reach it
	// solely by asserting on process-global log output produced through a
	// pure one-line delegation to Resolve, so a mirror would re-assert THIS
	// package's own unit-level behaviour without adding any call-site
	// information. The one thing a mirror could genuinely add — proof that
	// the delegating package still routes through clientip.Resolve at all —
	// is already carried by every trusted-proxy census row in that package,
	// since Resolve is the only route to the diagnostic. Honest boundary
	// (§11.4.6): this means a future defect confined to the diagnostic is
	// caught once, not four times; that is accepted deliberately here, not
	// an oversight, and it is stated so a later reader does not re-derive
	// the "why is this only in one package?" question as a finding.
	const configured = "f4-emit-probe-not-an-ip, 203.0.113.240,10.0.0.0/33"
	resetMalformedAllowlistWarnDedupe(t)
	t.Setenv(TrustedProxiesEnvVar, configured)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "203.0.113.240:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.60")

	var got string
	out := captureLogOutput(t, func() { got = Resolve(req) })

	// The valid bare-IP entry still matches, so the two malformed entries
	// really were skipped rather than poisoning the whole allowlist.
	if got != "198.51.100.60" {
		t.Fatalf("Resolve() = %q, want %q — the one VALID allowlist entry must still match the peer "+
			"even though two sibling entries are malformed", got, "198.51.100.60")
	}

	if out == "" {
		t.Fatalf("no diagnostic was logged for malformed LLM_VERIFIER_TRUSTED_PROXIES entries in %q — "+
			"a malformed entry fails closed SILENTLY, so an operator whose allowlist never matches has "+
			"no way to tell a typo apart from an unconfigured allowlist", configured)
	}
	for _, want := range []string{
		TrustedProxiesEnvVar,      // names the variable the operator must fix
		"f4-emit-probe-not-an-ip", // the neither-IP-nor-CIDR entry
		"10.0.0.0/33",             // the /-bearing entry that is not a valid CIDR
		"contains 2 entr",         // exactly two entries were rejected, not one, not three
		"fails closed",            // states the consequence, not merely the fact
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("malformed-allowlist diagnostic does not mention %q.\nfull log output:\n%s", want, out)
		}
	}

	// The one VALID entry must NOT be reported as malformed.
	if strings.Contains(out, "203.0.113.240") {
		t.Fatalf("the diagnostic names the VALID allowlist entry %q as malformed — only genuinely "+
			"unparseable entries may be reported.\nfull log output:\n%s", "203.0.113.240", out)
	}

	t.Logf("outcome: malformed entries reported to the operator -> %s", strings.TrimSpace(out))
}

// TestWarnOnMalformedTrustedProxiesEntriesDedupesPerConsecutiveRun asserts
// what the dedupe ACTUALLY does — warn once for a configured string, stay
// silent while that same string remains configured, warn AGAIN the moment
// the value changes, and warn AGAIN on a RETURN to a value already warned
// about.
//
// R11-F4: this test was named ...DedupesPerDistinctValue and stopped after
// the A→A→B legs, which is exactly the prefix that CANNOT distinguish
// per-distinct-value from per-consecutive-run. The distinguishing leg is
// the REVISIT, and it was never driven. lastWarnedTrustedProxies is one
// slot written on every observation, so any intervening different value —
// INCLUDING a perfectly clean one that warns about nothing — clears it and
// a later A warns a second time. Measured before the fix: A→1, A→0, B→0,
// A→1, i.e. TWO warnings for the single distinct value A. The name and the
// package doc both said "per distinct value"; the code never did.
//
// F4 (post-review finding, closed by this test): the package doc comment
// makes a specific, externally-observable behavioural claim about this
// extraction — the warning now fires "AT MOST ONCE process-wide per
// distinct value, rather than once per copy". Nothing tested the old
// behaviour, the new behaviour, or the transition; a reviewer's mutation
// DELETING the per-value dedupe left every suite green.
//
// Both failure directions are pinned deliberately:
//   - no dedupe at all -> isTrustedProxyIP runs on EVERY request that
//     reaches a trust decision, so a misconfigured allowlist would flood
//     the log once per request (`second` catches this);
//   - dedupe once per PROCESS rather than per observed value -> an operator
//     who edits the variable and restarts nothing would never be told their
//     NEW value is also malformed (`third` catches this);
//   - the dedupe silently strengthening into a true per-distinct-value
//     memo -> the documented "warns again on revisit" behaviour would
//     regress and this test's own claim would go stale (`fourth` and
//     `sixth` catch this, via a malformed and then a CLEAN intervening
//     value respectively).
func TestWarnOnMalformedTrustedProxiesEntriesDedupesPerConsecutiveRun(t *testing.T) {
	const (
		valueOne = "203.0.113.241,f4-dedupe-probe-value-one"
		valueTwo = "203.0.113.241,f4-dedupe-probe-value-two"
		// A WELL-FORMED value, so it warns about nothing at all. It is here
		// to prove the reset is driven by OBSERVATION and not by warning:
		// lastWarnedTrustedProxies is stored before the malformed scan runs.
		valueClean = "203.0.113.241"
	)

	resetMalformedAllowlistWarnDedupe(t)

	resolveOnce := func() {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "203.0.113.241:443"
		req.Header.Set("X-Forwarded-For", "198.51.100.61")
		_ = Resolve(req)
	}

	t.Setenv(TrustedProxiesEnvVar, valueOne)
	first := captureLogOutput(t, resolveOnce)
	if !strings.Contains(first, "f4-dedupe-probe-value-one") {
		t.Fatalf("the FIRST observation of malformed value %q logged no diagnostic naming it.\n"+
			"full log output:\n%s", valueOne, first)
	}

	second := captureLogOutput(t, resolveOnce)
	if strings.Contains(second, TrustedProxiesEnvVar) || strings.Contains(second, "f4-dedupe-probe-value-one") {
		t.Fatalf("the SAME malformed value %q warned a SECOND time — the per-value dedupe is gone, so a "+
			"misconfigured allowlist now logs once per request that reaches a trust decision.\n"+
			"second-run log output:\n%s", valueOne, second)
	}

	t.Setenv(TrustedProxiesEnvVar, valueTwo)
	third := captureLogOutput(t, resolveOnce)
	if !strings.Contains(third, "f4-dedupe-probe-value-two") {
		t.Fatalf("a DIFFERENT malformed value %q logged no diagnostic naming it — the dedupe must be per "+
			"distinct VALUE, not once per process, or an operator who edits a still-broken allowlist is "+
			"never told.\nthird-run log output:\n%s", valueTwo, third)
	}
	if strings.Contains(third, "f4-dedupe-probe-value-one") {
		t.Fatalf("the diagnostic for %q reported the PREVIOUS value's entry — the warning must describe "+
			"the CURRENTLY configured value.\nthird-run log output:\n%s", valueTwo, third)
	}

	// R11-F4, the leg that was missing: RETURN to valueOne. Under a true
	// per-distinct-value dedupe this stays silent (valueOne was warned
	// about already). Under the one-slot per-consecutive-run dedupe the
	// code actually implements, valueTwo overwrote the slot, so valueOne
	// warns a SECOND time. This is the assertion that separates the two.
	t.Setenv(TrustedProxiesEnvVar, valueOne)
	fourth := captureLogOutput(t, resolveOnce)
	if !strings.Contains(fourth, "f4-dedupe-probe-value-one") {
		t.Fatalf("RETURNING to the already-warned value %q logged nothing. The dedupe is a SINGLE slot "+
			"overwritten on every observation, so an intervening different value must reset it and this "+
			"revisit must warn again — if it no longer does, the dedupe has changed shape and both this "+
			"test's name and the package doc comment are now wrong.\nfourth-run log output:\n%s", valueOne, fourth)
	}

	// And the sharper form: the intervening value need not be malformed.
	// A CLEAN value warns about nothing, yet still overwrites the slot,
	// because the Store runs ahead of the scan that decides whether to warn.
	t.Setenv(TrustedProxiesEnvVar, valueClean)
	fifth := captureLogOutput(t, resolveOnce)
	if strings.Contains(fifth, TrustedProxiesEnvVar) {
		t.Fatalf("a WELL-FORMED allowlist %q emitted a diagnostic; only malformed or host-bits entries "+
			"may warn.\nfifth-run log output:\n%s", valueClean, fifth)
	}

	t.Setenv(TrustedProxiesEnvVar, valueOne)
	sixth := captureLogOutput(t, resolveOnce)
	if !strings.Contains(sixth, "f4-dedupe-probe-value-one") {
		t.Fatalf("returning to %q after a CLEAN value logged nothing — a clean value warns about nothing "+
			"but still overwrites the dedupe slot, so this revisit must warn. Total warnings for the ONE "+
			"distinct value %q across this test is therefore three, which is why the dedupe is documented "+
			"as per-CONSECUTIVE-RUN and not per-distinct-value.\nsixth-run log output:\n%s",
			valueOne, valueOne, sixth)
	}

	t.Logf("outcome: warn-once-per-CONSECUTIVE-RUN confirmed (first warned, repeat silent, new value warned, " +
		"revisit warned again, clean value silent, revisit after clean warned again)")
}

// TestZoneQualifiedIPv6AllowlistEntryIsRejectedAndReported pins the
// operator-visible consequence of a Go standard-library behaviour that is
// easy to be surprised by: net.ParseIP("fe80::1%eth0") returns nil — a
// ZONE-QUALIFIED IPv6 literal is not a parseable IP to net.ParseIP at all
// (verified empirically; net.ResolveIPAddr, a different API, is what
// handles zones).
//
// Two consequences follow for an operator who writes a link-local address
// into LLM_VERIFIER_TRUSTED_PROXIES, and this test asserts BOTH:
//
//  1. The entry can never match ANY peer — including a peer connecting from
//     exactly that address, whose own RemoteAddr host is the identical
//     zone-qualified literal (isTrustedProxyPeer parses the peer with the
//     same net.ParseIP and also gets nil). It fails CLOSED, which is the
//     correct direction, and is the F2 guard observed from the allowlist
//     side.
//  2. The malformed-entry diagnostic is the ONLY thing that tells the
//     operator this — which is precisely why F4's deletion of that
//     diagnostic had to be closed. Without it, "my link-local proxy is
//     allowlisted but its headers are still ignored" produces no signal
//     whatsoever.
//
// Honest boundary, established by running this test rather than assumed:
// the diagnostic is reachable only once a peer with a PARSEABLE IP reaches
// a trust decision, because isTrustedProxyPeer's own `peerIP == nil` guard
// (the F2 fail-closed default) returns BEFORE isTrustedProxyIP — the only
// caller of the diagnostic — is ever invoked. So in the exact topology of
// sub-case (2) below, where the link-local peer is the ONLY traffic
// source, the operator gets the fail-closed behaviour with NO warning at
// all. The two sub-cases are therefore asserted separately, and the
// warning is asserted only where it can actually fire.
func TestZoneQualifiedIPv6AllowlistEntryIsRejectedAndReported(t *testing.T) {
	const configured = "fe80::1%eth0"

	t.Run("reported_to_operator_when_a_parseable_peer_reaches_a_trust_decision", func(t *testing.T) {
		resetMalformedAllowlistWarnDedupe(t)
		t.Setenv(TrustedProxiesEnvVar, configured)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "203.0.113.242:443"
		req.Header.Set("X-Forwarded-For", "198.51.100.62")

		var got string
		out := captureLogOutput(t, func() { got = Resolve(req) })

		if got != "203.0.113.242" {
			t.Fatalf("Resolve() = %q, want the peer's own address %q — a zone-qualified allowlist entry "+
				"parses as neither IP nor CIDR, so it matches nothing and this peer stays untrusted",
				got, "203.0.113.242")
		}
		if !strings.Contains(out, configured) {
			t.Fatalf("no diagnostic named the zone-qualified allowlist entry %q. This warning is the ONLY "+
				"signal an operator gets that a link-local entry they explicitly configured will never "+
				"match any peer.\nfull log output:\n%s", configured, out)
		}
		t.Logf("outcome: zone-qualified allowlist entry reported to the operator -> %s", strings.TrimSpace(out))
	})

	t.Run("never_trusts_even_the_identical_zone_qualified_peer", func(t *testing.T) {
		t.Setenv(TrustedProxiesEnvVar, configured)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "[fe80::1%eth0]:443"
		req.Header.Set("X-Forwarded-For", "198.51.100.62")

		got := Resolve(req)
		if got != "fe80::1%eth0" {
			t.Fatalf("Resolve() = %q, want the peer's own RemoteAddr host %q — the peer's address is "+
				"parsed with the SAME net.ParseIP that rejected the allowlist entry, so an operator "+
				"cannot allowlist a link-local proxy by writing its exact address; it MUST fail closed "+
				"and ignore X-Forwarded-For", got, "fe80::1%eth0")
		}
		t.Logf("outcome: zone-qualified peer fails closed -> %q (and, per this test's doc comment, "+
			"receives NO warning in this topology because the peer-parse guard returns first)", got)
	})
}

// TestCIDRAllowlistEntryWithHostBitsIsWidenedAndReported closes R7-3.
//
// net.ParseCIDR("10.0.0.5/8") returns err == nil and MASKS the prefix to
// 10.0.0.0/8, so an operator writing what reads as one host's address
// actually allowlists the whole /8 — and every address in it may then
// dictate its own caller identity through X-Forwarded-For / X-Real-IP.
//
// Note the asymmetry this test exists to close: the SAFE-direction mistake
// (a zone-qualified entry, which fails CLOSED) was already warned about by
// TestZoneQualifiedIPv6AllowlistEntryIsRejectedAndReported, while this
// DANGEROUS-direction one was completely silent —
// warnOnMalformedTrustedProxiesEntries never looked at it, because the
// entry IS a valid CIDR.
//
// Both halves are asserted deliberately: the widening is REAL (so the
// warning is not warning about nothing), and the operator is TOLD (so the
// widening is not silent). Behaviour is unchanged by design — see
// warnOnMalformedTrustedProxiesEntries' own doc comment for why the
// ecosystem-standard masking is kept rather than narrowed or refused.
func TestCIDRAllowlistEntryWithHostBitsIsWidenedAndReported(t *testing.T) {
	const configured = "10.0.0.5/8"

	t.Run("widening_is_real_a_far_off_address_in_the_masked_prefix_is_trusted", func(t *testing.T) {
		resetMalformedAllowlistWarnDedupe(t)
		t.Setenv(TrustedProxiesEnvVar, configured)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// Not the host the operator wrote, and not adjacent to it — the
		// far end of the /8 that "10.0.0.5/8" silently masks to.
		req.RemoteAddr = "10.255.255.254:443"
		req.Header.Set("X-Forwarded-For", "198.51.100.62")

		got := Resolve(req)
		if got != "198.51.100.62" {
			t.Fatalf("Resolve() = %q, want %q — 10.255.255.254 is 16.7M addresses away from the single "+
				"host 10.0.0.5 the operator wrote, yet net.ParseCIDR masked the entry to 10.0.0.0/8 and "+
				"isTrustedProxyIP matched it, so this peer IS trusted and its X-Forwarded-For IS honoured. "+
				"If this assertion ever fails the widening has stopped happening and the warning below is "+
				"warning about nothing", got, "198.51.100.62")
		}
		t.Logf("outcome: host-bits entry %q silently trusts %s -> forwarded identity honoured (%q)",
			configured, "10.255.255.254", got)
	})

	t.Run("reported_to_operator", func(t *testing.T) {
		resetMalformedAllowlistWarnDedupe(t)
		t.Setenv(TrustedProxiesEnvVar, configured)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.255.255.254:443"
		req.Header.Set("X-Forwarded-For", "198.51.100.62")

		out := captureLogOutput(t, func() { _ = Resolve(req) })

		if !strings.Contains(out, configured) {
			t.Fatalf("no diagnostic named the host-bits allowlist entry %q. This warning is the ONLY "+
				"signal an operator gets that their entry trusts an entire network rather than the one "+
				"host it reads as.\nfull log output:\n%s", configured, out)
		}
		if !strings.Contains(out, "10.0.0.0/8") {
			t.Fatalf("the diagnostic named the entry but not the network it actually trusts (10.0.0.0/8). "+
				"Naming only the entry leaves the operator to work out the blast radius themselves, which "+
				"is the whole thing they got wrong.\nfull log output:\n%s", out)
		}
		t.Logf("outcome: host-bits entry reported to the operator -> %s", strings.TrimSpace(out))
	})

	t.Run("a_correctly_written_network_entry_is_not_warned_about", func(t *testing.T) {
		resetMalformedAllowlistWarnDedupe(t)
		t.Setenv(TrustedProxiesEnvVar, "10.0.0.0/8")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.255.255.254:443"
		req.Header.Set("X-Forwarded-For", "198.51.100.62")

		out := captureLogOutput(t, func() { _ = Resolve(req) })

		if strings.Contains(out, "HOST BITS") {
			t.Fatalf("a correctly-written network entry was warned about. A diagnostic that fires on "+
				"correct configuration trains operators to ignore it, which costs the real warning its "+
				"only value.\nfull log output:\n%s", out)
		}
		t.Logf("outcome: correctly-written network entry produces no host-bits warning (log: %q)",
			strings.TrimSpace(out))
	})
}

// TestTrustedProxiesEnvVarLiteralValue pins the operator-facing environment
// variable NAME as a hardcoded literal — deliberately NOT
// TrustedProxiesEnvVar itself (that comparison would be tautological and
// would follow any rename silently, catching nothing).
//
// F1 (post-extraction review finding, closed by this test): after the
// HXC-298 extraction, "LLM_VERIFIER_TRUSTED_PROXIES" exists as a literal in
// exactly ONE place in this codebase's non-test source
// (clientip.go's `const TrustedProxiesEnvVar = "LLM_VERIFIER_TRUSTED_PROXIES"`).
// Every one of this module's other references go through the symbolic
// constant (TrustedProxiesEnvVar / clientIPTrustedProxiesEnvVar /
// securityClientIPTrustedProxiesEnvVar, the latter two now aliases of this
// one) and therefore follow a rename of the underlying string AUTOMATICALLY
// and SILENTLY: a reviewer's rename-the-value mutation left all four
// consumer suites green, because every test's setenv/assert pair moved
// together with the constant.
//
// The complete file set carrying those symbolic references, across ALL
// THREE delegating packages plus this one (an earlier draft of this
// comment omitted enhanced/enterprise's three files and cited a count that
// went stale on the very next edit — a hardcoded reference count is
// fragile exactly like the rename this test guards against, so it is
// deliberately NOT repeated here; reverify with
// `grep -rn "TrustedProxiesEnvVar" --include="*_test.go" .` from the
// llm-verifier module root, whose count moves every time a census row is
// added and is therefore never quoted as a fixed number in this comment):
// api/hxc292_client_ip_trust_red_test.go, api/middleware_test.go,
// security/hxc299_extract_ip_trust_red_test.go, security/security_test.go,
// enhanced/enterprise/hxc298_client_ip_trust_red_test.go,
// enhanced/enterprise/api_handlers_test.go,
// enhanced/enterprise/enterprise_extended_test.go, and this package's own
// clientip_test.go.
//
// LLM_VERIFIER_TRUSTED_PROXIES is not an internal implementation detail —
// it is a documented deployment contract operators set on real,
// already-running deployments: .env.example, docs/administrator-manual.md,
// docs/ENVIRONMENT_VARIABLES.md, docs/deployment/docker.md, and
// docs/deployment/kubernetes.md all instruct an operator to export this
// EXACT name. A rename of the Go constant's value would compile clean,
// pass every existing test (all of which reference the constant
// symbolically), and silently VOID every already-deployed operator's
// allowlist — the renamed code would read a differently-named environment
// variable that no deployed operator has set, silently reverting every
// proxied deployment to default-deny (see the empty-default cost discussion
// preserved in api/middleware.go from the original HXC-292 fix: a bounded,
// instantly-reversible loss of identity GRANULARITY, never an outage and
// never a forged-identity hole) with zero test signal that anything changed.
//
// This test is therefore deliberately anti-symbolic: it is the ONE place in
// this codebase's test suite that does NOT reference TrustedProxiesEnvVar
// on both sides of a comparison, so it is the one test a rename of the
// underlying string cannot silently carry along with it.
func TestTrustedProxiesEnvVarLiteralValue(t *testing.T) {
	const documentedEnvVarName = "LLM_VERIFIER_TRUSTED_PROXIES"

	if TrustedProxiesEnvVar != documentedEnvVarName {
		t.Fatalf("TrustedProxiesEnvVar = %q, want the documented deployment-contract literal %q -- "+
			"renaming this value silently voids every already-deployed operator's "+
			"LLM_VERIFIER_TRUSTED_PROXIES allowlist across every deployment documented in "+
			".env.example, docs/administrator-manual.md, docs/ENVIRONMENT_VARIABLES.md, "+
			"docs/deployment/docker.md, and docs/deployment/kubernetes.md -- with every other "+
			"test in this module (all symbolic references to TrustedProxiesEnvVar) staying green",
			TrustedProxiesEnvVar, documentedEnvVarName)
	}
}

// TestWarnOnMalformedTrustedProxiesEntriesIgnoresEmptyEntries asserts the
// diagnostic stays SILENT for an allowlist whose only irregularity is an
// empty element — a trailing comma ("10.0.0.1,"), an interior double comma
// ("a,,b"), or a whitespace-only element (" , ").
//
// # F12 (post-review finding, closed by this test)
//
// The `entry == ""` skip exists at TWO sites, and round 4 tested one and
// classified BOTH:
//
//   - isTrustedProxyIP's skip: mutating it away survives, and is GENUINELY
//     equivalent — an empty string fails net.ParseIP and net.ParseCIDR
//     anyway, so the trust decision is identical either way. That
//     classification was correct.
//
//   - warnOnMalformedTrustedProxiesEntries's skip (the one THIS test
//     covers): mutating it away ALSO survived every suite, but is NOT
//     equivalent. Captured behavioural difference on the same input, with
//     allowlist "10.0.0.1," resolving the same request to the same
//     "9.9.9.9":
//
//     PRISTINE: (silent)
//     MUTANT:   ... contains 1 entr(y/ies) that are neither a valid IP nor
//     a valid CIDR ... : []
//
// An operator's harmless trailing comma is thereby reported as a malformed
// entry — and NAMED as the empty string, which corresponds to no line they
// can find — sending them to fix configuration that is already correct.
// That is the SAME defect shape as F10 (the diagnostic falsely naming a
// correct entry), in the SAME function, one guard over; F10's own
// remediation did not sweep for it.
//
// Deliberately asserts SILENCE rather than a message: this diagnostic's
// entire value is that it fires ONLY for entries an operator must act on,
// so a false positive is as harmful as a false negative (§11.4.201-class
// reasoning — a guard that fires on a correct configuration is a FAIL-bluff
// even though nothing "passed" wrongly).
func TestWarnOnMalformedTrustedProxiesEntriesIgnoresEmptyEntries(t *testing.T) {
	cases := []struct {
		name       string
		configured string
		peer       string
		want       string
		outcome    string
	}{
		{
			name:       "trailing_comma",
			configured: "10.0.0.1,",
			peer:       "10.0.0.1:443",
			want:       "9.9.9.9",
			outcome:    "a trailing comma leaves ONE empty element, which is skipped silently",
		},
		{
			name:       "interior_double_comma",
			configured: "10.0.0.2,,172.20.0.0/16",
			peer:       "10.0.0.2:443",
			want:       "9.9.9.9",
			outcome:    "an interior double comma leaves ONE empty element between two VALID entries",
		},
		{
			name:       "whitespace_only_element",
			configured: "10.0.0.3, ,172.20.0.0/16",
			peer:       "10.0.0.3:443",
			want:       "9.9.9.9",
			outcome:    "a whitespace-only element trims to empty and is skipped silently",
		},
		{
			name:       "leading_comma",
			configured: ",10.0.0.4",
			peer:       "10.0.0.4:443",
			want:       "9.9.9.9",
			outcome:    "a leading comma leaves ONE empty element before a valid entry",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetMalformedAllowlistWarnDedupe(t)
			t.Setenv(TrustedProxiesEnvVar, tc.configured)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tc.peer
			req.Header.Set("X-Forwarded-For", tc.want)

			var got string
			out := captureLogOutput(t, func() { got = Resolve(req) })

			// The allowlist's real entries must still work — this proves
			// the diagnostic was actually REACHED (isTrustedProxyIP calls
			// it only once a peer with a parseable IP reaches a trust
			// decision), so the silence below is genuine silence and not
			// merely an un-executed code path.
			if got != tc.want {
				t.Fatalf("%s: Resolve() = %q, want %q -- the VALID entries of allowlist %q must still "+
					"match this peer, otherwise the diagnostic below was never reached and its "+
					"silence proves nothing", tc.outcome, got, tc.want, tc.configured)
			}

			if strings.TrimSpace(out) != "" {
				t.Fatalf("%s: allowlist %q logged a malformed-entry diagnostic, but its only "+
					"irregularity is an EMPTY element -- an operator's harmless trailing/extra comma "+
					"must never be reported as malformed (the report names the empty string, which "+
					"matches no line they can find, sending them to fix correct configuration).\n"+
					"full log output:\n%s", tc.outcome, tc.configured, out)
			}

			t.Logf("outcome: %s -> resolved %q, diagnostic silent", tc.outcome, got)
		})
	}
}

// TestResolveMultiInstanceForwardingHeadersOverLoopback drives REPEATED
// forwarding-header lines over a REAL loopback TCP connection through Go's
// own HTTP server parser — not a hand-assembled http.Request — and asserts
// the resolved identity for each.
//
// # F11 (post-review finding, closed by this test and the census rows)
//
// Go's server preserves EVERY instance of a repeated header; r.Header.Get
// returns only the FIRST. Reading X-Forwarded-For with Get therefore
// re-opened the leftmost-selection bypass resolveForwardedFor exists to
// close, and X-Real-IP had the identical shape. Both failed OPEN: a
// trusted peer forwarding an attacker's forged first instance had that
// forged value returned as the caller identity.
//
// This test is deliberately WIRE-LEVEL rather than
// httptest.NewRequest+Header.Add, because the defect is precisely about
// what Go's parser does with repeated header LINES. The census rows
// (xff_two_instances_*, xri_two_instances_*) cover the same invariant
// cheaply in every delegating package; this one proves the wire shape they
// simulate is the shape a real client can actually put on a socket.
func TestResolveMultiInstanceForwardingHeadersOverLoopback(t *testing.T) {
	cases := []struct {
		name       string
		headerName string
		instances  []string
		want       string
		outcome    string
	}{
		{
			name:       "xff_two_instances_rightmost_untrusted_wins",
			headerName: "X-Forwarded-For",
			instances:  []string{"8.8.8.8", "203.0.113.9"},
			want:       "203.0.113.9",
			outcome:    "both X-Forwarded-For instances enter the right-to-left walk, so the proxy-contributed rightmost entry wins over the attacker's forged first instance",
		},
		{
			name:       "xff_three_instances_rightmost_untrusted_wins",
			headerName: "X-Forwarded-For",
			instances:  []string{"1.2.3.4", "8.8.8.8", "203.0.113.9"},
			want:       "203.0.113.9",
			outcome:    "the walk reaches the rightmost entry regardless of how many forged instances precede it (Header.Get returned 1.2.3.4)",
		},
		{
			name:       "xri_two_instances_refused",
			headerName: "X-Real-IP",
			instances:  []string{"8.8.8.8", "203.0.113.9"},
			want:       "127.0.0.1",
			outcome:    "X-Real-IP is SET not appended, so repeated instances are unattributable and ALL are refused -- the identity falls back to the loopback peer, never the forged first instance",
		},
		{
			name:       "xri_single_instance_still_honoured",
			headerName: "X-Real-IP",
			instances:  []string{"203.0.113.9"},
			want:       "203.0.113.9",
			outcome:    "control: the single-instance X-Real-IP path is unchanged by the multi-instance refusal",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(TrustedProxiesEnvVar, "127.0.0.0/8")

			type observation struct {
				values   []string
				get      string
				resolved string
			}
			obsCh := make(chan observation, 1)

			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("loopback listen failed: %v", err)
			}
			defer ln.Close()

			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				obsCh <- observation{
					values:   append([]string(nil), r.Header.Values(tc.headerName)...),
					get:      r.Header.Get(tc.headerName),
					resolved: Resolve(r),
				}
				w.WriteHeader(http.StatusNoContent)
			})}
			go func() { _ = srv.Serve(ln) }()
			defer func() { _ = srv.Close() }()

			conn, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatalf("loopback dial failed: %v", err)
			}
			defer conn.Close()

			var wire strings.Builder
			wire.WriteString("GET /probe HTTP/1.1\r\nHost: " + ln.Addr().String() + "\r\n")
			for _, v := range tc.instances {
				// One SEPARATE header line per instance -- the shape
				// Header.Set cannot produce and Header.Get truncates.
				wire.WriteString(tc.headerName + ": " + v + "\r\n")
			}
			wire.WriteString("Connection: close\r\n\r\n")

			if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
				t.Fatalf("set deadline failed: %v", err)
			}
			if _, err := conn.Write([]byte(wire.String())); err != nil {
				t.Fatalf("wire write failed: %v", err)
			}
			if _, err := bufio.NewReader(conn).ReadString('\n'); err != nil {
				t.Fatalf("response read failed: %v", err)
			}

			var obs observation
			select {
			case obs = <-obsCh:
			case <-time.After(10 * time.Second):
				t.Fatal("handler never observed the request")
			}

			// Precondition: Go really did preserve every instance. If this
			// ever stops holding, the rest of the assertions would be
			// testing a different situation than the one they describe.
			if len(obs.values) != len(tc.instances) {
				t.Fatalf("Go's parser preserved %d %s instance(s) %q, want %d %q -- this test's premise "+
					"(repeated header LINES survive parsing as separate values) no longer holds",
					len(obs.values), tc.headerName, obs.values, len(tc.instances), tc.instances)
			}

			if obs.resolved != tc.want {
				t.Fatalf("%s: Resolve() = %q, want %q\n  wire %s instances=%q\n  Header.Values()=%q "+
					"Header.Get()=%q (Get sees only the FIRST instance -- reading it here is the F11 "+
					"defect)", tc.outcome, obs.resolved, tc.want, tc.headerName, tc.instances,
					obs.values, obs.get)
			}

			t.Logf("outcome: %s\n  wire %s instances=%q -> Header.Values()=%q Header.Get()=%q "+
				"Resolve()=%q", tc.outcome, tc.headerName, tc.instances, obs.values, obs.get,
				obs.resolved)
		})
	}
}

// TestResolveIdentityDoesNotForkAcrossSpellings asserts that one caller
// reaching this service by several different spellings of its own address
// — and by the two different PATHS (direct connection vs forwarded through
// a trusted proxy) — resolves to exactly ONE identity string.
//
// # D1 (post-review finding, closed by this test and the census rows)
//
// Resolve's return value is an audit-trail identity AND a rate-limit key.
// Before canonicalIPString, a trusted proxy's "X-Forwarded-For:
// ::ffff:8.8.8.8" resolved to the literal "::ffff:8.8.8.8" while plain
// "8.8.8.8" resolved to "8.8.8.8" — two distinct identities and two
// distinct rate-limit buckets for ONE caller, and the header-derived one
// disagreed with what that same caller gets on a DIRECT connection. Hex
// case forked it a second way ("2001:DB8::1" vs "2001:db8::1"). That is
// the identical failure stripBrackets and normalizeRemoteAddr already
// exist to prevent, on two axes nothing covered.
//
// Asserted as an EQUIVALENCE (all spellings agree with each other and with
// the direct-connection path) rather than against hardcoded strings, so it
// keeps holding if the canonical form itself is ever revisited.
func TestResolveIdentityDoesNotForkAcrossSpellings(t *testing.T) {
	groups := []struct {
		name      string
		spellings []string
		outcome   string
	}{
		{
			name:      "ipv4_and_its_v4_mapped_ipv6_form",
			spellings: []string{"8.8.8.8", "::ffff:8.8.8.8", "[::ffff:8.8.8.8]"},
			outcome:   "a dotted quad, its IPv4-mapped IPv6 form, and that form bracketed are ONE caller",
		},
		{
			name:      "ipv6_hex_case_and_bracketing",
			spellings: []string{"2001:db8::1", "2001:DB8::1", "[2001:DB8::1]", " 2001:Db8::1 "},
			outcome:   "IPv6 hex case, bracketing, and surrounding whitespace are ONE caller",
		},
		{
			name:      "ipv6_zero_run_compression",
			spellings: []string{"2001:db8:0:0:0:0:0:2", "2001:db8::2"},
			outcome:   "an expanded IPv6 zero run and its :: compressed form are ONE caller",
		},
	}

	const trustedProxy = "10.0.0.1"

	resolveViaProxy := func(t *testing.T, spelling string) string {
		t.Helper()
		t.Setenv(TrustedProxiesEnvVar, trustedProxy)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = trustedProxy + ":443"
		req.Header.Set("X-Forwarded-For", spelling)
		return Resolve(req)
	}

	for _, g := range groups {
		t.Run(g.name, func(t *testing.T) {
			first := resolveViaProxy(t, g.spellings[0])
			if first == "" {
				t.Fatalf("%s: spelling %q resolved to the empty string", g.outcome, g.spellings[0])
			}

			for _, spelling := range g.spellings[1:] {
				got := resolveViaProxy(t, spelling)
				if got != first {
					t.Fatalf("%s: spelling %q resolved to %q but %q resolved to %q -- ONE caller must "+
						"never fork into two audit-trail identities / two rate-limit keys",
						g.outcome, spelling, got, g.spellings[0], first)
				}
			}

			// The forwarded path must also agree with the identity the
			// SAME caller gets connecting DIRECTLY, or the two paths keep
			// separate books on one caller.
			t.Setenv(TrustedProxiesEnvVar, "")
			direct := httptest.NewRequest(http.MethodGet, "/test", nil)
			direct.RemoteAddr = net.JoinHostPort(first, "54321")
			if got := Resolve(direct); got != first {
				t.Fatalf("%s: forwarded-path identity %q but a DIRECT connection from that same "+
					"address resolved to %q -- the proxied and direct paths must agree",
					g.outcome, first, got)
			}

			t.Logf("outcome: %s -> single identity %q across %d spellings + the direct path",
				g.outcome, first, len(g.spellings))
		})
	}
}
