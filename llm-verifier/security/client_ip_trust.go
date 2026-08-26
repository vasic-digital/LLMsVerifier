package security

import (
	"net/http"

	"digital.vasic.llmsverifier/clientip"
)

// HXC-299: caller-identity trust resolution for extractIPAddress
// (security.go). This file originally MIRRORED — rather than importing —
// the corrected algorithm HXC-292 landed for the sibling function
// api/middleware.go's getClientIP, because getClientIP was unexported and
// api/ was deliberately frozen (out of scope) for HXC-299's fix. That left
// two independently-maintained copies of one security-relevant algorithm
// with nothing to stop them drifting apart (HXC-308). HXC-298 found a
// THIRD, independent, still-broken copy in
// enhanced/enterprise/api.go — fixing it with a fourth mirror would only
// have widened that risk, so the corrected algorithm was extracted into
// the new clientip package (digital.vasic.llmsverifier/clientip) instead:
// a dependency-neutral leaf package with no import-cycle risk against
// api/, security/, or enhanced/enterprise/. See clientip/clientip.go's
// package doc comment for the full extraction rationale and the original
// F1 defect history (leftmost X-Forwarded-For selection under an
// appending proxy) this package inherits already-fixed.
//
// resolveClientIP below now delegates to clientip.Resolve. There is
// exactly one implementation of this algorithm in this codebase;
// api/middleware.go's getClientIP and enhanced/enterprise/api.go's
// EnterpriseAPI.getClientIP delegate to the same function.
const securityClientIPTrustedProxiesEnvVar = clientip.TrustedProxiesEnvVar

// resolveClientIP extracts the real caller identity from r, honouring
// X-Forwarded-For / X-Real-IP ONLY when the immediate TCP peer
// (r.RemoteAddr) is on the operator-configured trusted-proxy allowlist
// (securityClientIPTrustedProxiesEnvVar). This is extractIPAddress's
// (security.go) sole implementation — see that function for the one call
// site (AuditTrail.LogRequest's AuditEntry.IPAddress) and its doc comment
// for a census of extractIPAddress's own callers.
func resolveClientIP(r *http.Request) string {
	return clientip.Resolve(r)
}
