package auth

// JWKS (JSON Web Key Set) client + verifier for SSO JWT validation.
//
// Round-59 §11.4 anti-bluff: this file implements the real asymmetric
// signature verification that closes round-28's
// ErrJWTSignatureVerificationNotImplemented HIGH-security gap. Until
// round-28, SSOManager.ValidateToken base64-decoded the JWT payload
// directly and accepted any well-formed 3-part dot-delimited string as
// a verified SSO authentication — a silent authentication bypass.
// Round-28 refused such tokens via a sentinel error. Round-59 wires
// real verification:
//
//   1. Fetch the IdP's JSON Web Key Set from the configured jwks_uri
//      (cached per kid with bounded TTL — typical 1h to avoid DoS).
//   2. Resolve the token's `kid` header against the JWKS.
//   3. Reconstruct the asymmetric public key (RSA n+e, or EC x+y+crv).
//   4. Verify the JWT signature using the resolved key + the token's
//      `alg` header — restricted to a strict allowlist that EXCLUDES
//      `none` and EXCLUDES all HMAC algorithms (those imply a shared
//      secret, which is wrong for SSO and is the historical source of
//      catastrophic JWT-confusion vulnerabilities).
//   5. Validate registered claims (`exp`, `iat`, `iss`, `aud`).
//
// CONST-035 / CONST-042 / CONST-050(A) / Article XI §11.9 anchors apply.
// No JWKS URL is hardcoded; every IdP-specific detail lives in
// JWKSConfig supplied by the operator at runtime.

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ===== Round-59 sentinel errors ============================================

// ErrJWTSignatureInvalid is returned when JWKS-resolved key verification
// rejects the token's signature. Distinct from
// ErrJWTSignatureVerificationNotImplemented (which now signals the
// "JWKS not configured / IdP unreachable" path).
var ErrJWTSignatureInvalid = errors.New("llmsverifier auth: JWT signature verification failed against configured JWKS key (§11.4 anti-bluff: CONST-035 / CONST-042)")

// ErrJWTKeyNotFound is returned when the token's `kid` header is not
// present in the fetched JWKS. After cache refresh, persistent
// not-found is a hard error — never silently fall back to no
// verification.
var ErrJWTKeyNotFound = errors.New("llmsverifier auth: JWT `kid` not present in configured JWKS — refusing to authenticate (§11.4 anti-bluff: CONST-035)")

// ErrJWTAlgorithmNotAllowed is returned when the token's `alg` header
// is not in the SSO allowlist. `none` and HMAC algorithms (`HS*`) are
// permanently forbidden for SSO regardless of operator config —
// accepting them is the classic "alg-confusion" auth bypass.
var ErrJWTAlgorithmNotAllowed = errors.New("llmsverifier auth: JWT `alg` not in SSO allowlist — `none` and HMAC algorithms are permanently rejected for SSO (§11.4 anti-bluff: CONST-035 / CONST-042)")

// ErrJWKSEndpointUnreachable is returned when the configured JWKS URL
// could not be fetched (network error, non-2xx status, malformed JSON).
// Callers MUST treat this as a hard auth failure — never silently
// proceed without verification.
var ErrJWKSEndpointUnreachable = errors.New("llmsverifier auth: JWKS endpoint unreachable or returned invalid response — refusing to authenticate (§11.4 anti-bluff: CONST-035)")

// ErrJWTClaimsInvalid is returned when one of the registered claims
// (`exp`, `iat`, `iss`, `aud`) fails validation against SSOConfig.
var ErrJWTClaimsInvalid = errors.New("llmsverifier auth: JWT claims failed validation (exp/iat/iss/aud)")

// ErrJWTMalformed is returned when the token does not parse as a
// 3-part JWT or its header is unreadable.
var ErrJWTMalformed = errors.New("llmsverifier auth: JWT malformed or unparseable")

// ===== Allowed algorithm set ===============================================

// defaultAllowedAlgs is the asymmetric-only SSO allowlist applied when
// JWKSConfig.AllowedAlgs is empty. Permanently excludes `none` + all
// HMAC algorithms (`HS*`) — those are wrong for SSO regardless of any
// per-deployment policy.
var defaultAllowedAlgs = []string{
	"RS256", "RS384", "RS512",
	"PS256", "PS384", "PS512",
	"ES256", "ES384", "ES512",
}

// forbiddenAlgs is the closed set of algorithms permanently rejected
// for SSO no matter what operator config says. This is the safety net
// behind the AllowedAlgs allowlist.
var forbiddenAlgs = map[string]struct{}{
	"none":  {},
	"NONE":  {},
	"None":  {},
	"HS256": {},
	"HS384": {},
	"HS512": {},
}

// ===== Public configuration ================================================

// JWKSConfig carries everything ValidateTokenWithJWKS needs. Operator
// supplies one of these per SSO provider — NEVER hardcoded.
type JWKSConfig struct {
	// JWKSURL is the IdP's published `jwks_uri` (typically discovered
	// from the IdP's `<issuer>/.well-known/openid-configuration`
	// document). Required.
	JWKSURL string

	// Issuer is the expected `iss` claim. Required — empty issuer
	// disables `iss` validation which is unsafe for SSO.
	Issuer string

	// Audience is the expected `aud` claim. Required — empty audience
	// disables `aud` validation which is unsafe for SSO.
	Audience string

	// AllowedAlgs restricts the accepted `alg` header. Empty -> use
	// defaultAllowedAlgs. Always sanitised against forbiddenAlgs.
	AllowedAlgs []string

	// CacheTTL bounds how long a fetched JWKS document is reused
	// before refresh. Zero -> 1 hour (typical IdP rotation cadence).
	// Mandatory cap of 24h to avoid stuck-key DoS scenarios.
	CacheTTL time.Duration

	// ClockSkew tolerance for `exp` / `iat` / `nbf` checks. Zero ->
	// 60 seconds.
	ClockSkew time.Duration

	// HTTPClient is an optional custom transport (useful for tests via
	// httptest.NewServer). Nil -> http.DefaultClient with a 10s timeout.
	HTTPClient *http.Client
}

// resolveDefaults returns a JWKSConfig with zero-valued fields filled
// from sensible defaults. Forbidden algs are stripped from AllowedAlgs.
func (c *JWKSConfig) resolveDefaults() *JWKSConfig {
	out := *c
	if len(out.AllowedAlgs) == 0 {
		out.AllowedAlgs = append([]string{}, defaultAllowedAlgs...)
	} else {
		filtered := out.AllowedAlgs[:0]
		for _, a := range out.AllowedAlgs {
			if _, banned := forbiddenAlgs[a]; banned {
				continue
			}
			filtered = append(filtered, a)
		}
		out.AllowedAlgs = filtered
	}
	if out.CacheTTL <= 0 {
		out.CacheTTL = time.Hour
	}
	if out.CacheTTL > 24*time.Hour {
		out.CacheTTL = 24 * time.Hour
	}
	if out.ClockSkew <= 0 {
		out.ClockSkew = 60 * time.Second
	}
	if out.HTTPClient == nil {
		out.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &out
}

// ===== JWK / JWKS wire structs (RFC 7517) ==================================

type jwkRaw struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg,omitempty"`
	Use string `json:"use,omitempty"`

	// RSA
	N string `json:"n,omitempty"`
	E string `json:"e,omitempty"`

	// EC
	Crv string `json:"crv,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
}

type jwksDocument struct {
	Keys []jwkRaw `json:"keys"`
}

// ===== JWKS cache ==========================================================

// jwksKeySet is a refresh-on-TTL-expiry cache of parsed public keys.
// Concurrency-safe via mu (RWMutex). Bounded TTL prevents DoS-by-poll
// against the IdP.
type jwksKeySet struct {
	cfg *JWKSConfig

	mu         sync.RWMutex
	keys       map[string]interface{} // kid -> *rsa.PublicKey | *ecdsa.PublicKey
	fetchedAt  time.Time
	lastErr    error
	fetchCount int // exposed for tests verifying we do NOT refetch on every call
}

// newJWKSKeySet builds a cache from cfg. Does not fetch eagerly.
func newJWKSKeySet(cfg *JWKSConfig) *jwksKeySet {
	return &jwksKeySet{
		cfg:  cfg.resolveDefaults(),
		keys: make(map[string]interface{}),
	}
}

// keyFor returns the parsed public key for the given kid, refreshing
// the cache if past TTL. On any IdP-side failure returns a wrapped
// ErrJWKSEndpointUnreachable / ErrJWTKeyNotFound rather than silently
// returning nil.
func (s *jwksKeySet) keyFor(ctx context.Context, kid string) (interface{}, error) {
	s.mu.RLock()
	stale := time.Since(s.fetchedAt) > s.cfg.CacheTTL
	k, hit := s.keys[kid]
	s.mu.RUnlock()

	if hit && !stale {
		return k, nil
	}

	// Either we've never fetched OR the cache is stale OR the kid is
	// missing (could be a freshly-rotated key). Try a refresh.
	if err := s.refresh(ctx); err != nil {
		// Surface the cached key if we have one — better than 100%
		// outage when IdP blips. But ONLY for non-stale entries; a
		// stale entry + IdP failure is hard fail.
		if hit && !stale {
			return k, nil
		}
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if k, ok := s.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("%w: kid=%q", ErrJWTKeyNotFound, kid)
}

// refresh fetches the JWKS document, parses each JWK into a
// crypto/{rsa,ecdsa}.PublicKey, and atomically replaces the cache.
// Failures leave the previous cache in place.
func (s *jwksKeySet) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.JWKSURL, nil)
	if err != nil {
		return fmt.Errorf("%w: building JWKS request: %v", ErrJWKSEndpointUnreachable, err)
	}
	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		s.mu.Lock()
		s.lastErr = err
		s.mu.Unlock()
		return fmt.Errorf("%w: %v", ErrJWKSEndpointUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: JWKS endpoint returned status %d", ErrJWKSEndpointUnreachable, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: reading JWKS body: %v", ErrJWKSEndpointUnreachable, err)
	}
	var doc jwksDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("%w: parsing JWKS JSON: %v", ErrJWKSEndpointUnreachable, err)
	}

	parsed := make(map[string]interface{}, len(doc.Keys))
	for _, k := range doc.Keys {
		pk, err := jwkToPublicKey(&k)
		if err != nil {
			// Skip individual unparseable key; do not abort the whole
			// refresh — the IdP often publishes mixed key types.
			continue
		}
		if k.Kid == "" {
			continue
		}
		parsed[k.Kid] = pk
	}

	s.mu.Lock()
	s.keys = parsed
	s.fetchedAt = time.Now()
	s.lastErr = nil
	s.fetchCount++
	s.mu.Unlock()
	return nil
}

// jwkToPublicKey converts a single JWK to a Go public key. Only RSA
// and EC are supported (the asymmetric set in defaultAllowedAlgs).
func jwkToPublicKey(k *jwkRaw) (interface{}, error) {
	switch strings.ToUpper(k.Kty) {
	case "RSA":
		if k.N == "" || k.E == "" {
			return nil, fmt.Errorf("RSA JWK missing n/e")
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("RSA n decode: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("RSA e decode: %w", err)
		}
		// e is a positive integer; per RFC 7518 it's big-endian.
		e := 0
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}
		if e <= 0 {
			return nil, fmt.Errorf("RSA e non-positive")
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: e,
		}, nil
	case "EC":
		if k.Crv == "" || k.X == "" || k.Y == "" {
			return nil, fmt.Errorf("EC JWK missing crv/x/y")
		}
		var curve elliptic.Curve
		switch k.Crv {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("EC JWK unsupported curve %q", k.Crv)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, fmt.Errorf("EC x decode: %w", err)
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, fmt.Errorf("EC y decode: %w", err)
		}
		return &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported kty %q", k.Kty)
	}
}

// peekJWTAlg decodes only the JWT header segment and extracts `alg`.
// Used to enforce the asymmetric-only allowlist BEFORE jwt.Parse so
// the precise CONST-035 / CONST-042 sentinel surfaces (jwt.Parse would
// otherwise lump alg violations into a generic ErrSignatureInvalid).
func peekJWTAlg(tokenString string) (string, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("token does not have 3 segments")
	}
	hBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("header base64 decode: %w", err)
	}
	var header map[string]interface{}
	if err := json.Unmarshal(hBytes, &header); err != nil {
		return "", fmt.Errorf("header JSON parse: %w", err)
	}
	alg, _ := header["alg"].(string)
	if alg == "" {
		return "", fmt.Errorf("header has no alg")
	}
	return alg, nil
}

// ===== Verifier ============================================================

// VerifyJWTWithJWKS verifies tokenString using the configured JWKS
// endpoint, returning parsed jwt.MapClaims on success and a wrapped
// round-59 sentinel on failure.
//
// CONST-035 / CONST-042 guarantees:
//   - `none` alg always rejected (ErrJWTAlgorithmNotAllowed)
//   - HMAC algs always rejected for SSO (ErrJWTAlgorithmNotAllowed)
//   - Any alg outside AllowedAlgs rejected (ErrJWTAlgorithmNotAllowed)
//   - Signature failure surfaces ErrJWTSignatureInvalid
//   - Unknown kid surfaces ErrJWTKeyNotFound
//   - exp/iat/iss/aud failures surface ErrJWTClaimsInvalid
func verifyJWTWithJWKS(ctx context.Context, cfg *JWKSConfig, keySet *jwksKeySet, tokenString string) (jwt.MapClaims, error) {
	if cfg.JWKSURL == "" {
		// Per round-28 semantics: an unconfigured SSO flow must surface
		// the original sentinel — distinct from a CONFIGURED-but-broken
		// flow which surfaces ErrJWKSEndpointUnreachable.
		return nil, ErrJWTSignatureVerificationNotImplemented
	}

	// Defensive resolve in case caller passed a raw config.
	cfg = cfg.resolveDefaults()

	// Pre-screen the algorithm at the header level BEFORE handing the
	// token to jwt.Parse so we can return the precise sentinel.
	// jwt.Parse's WithValidMethods would surface a generic "signing
	// method invalid" error that loses the CONST-035 / CONST-042
	// context we want to preserve.
	headerAlg, headerErr := peekJWTAlg(tokenString)
	if headerErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrJWTMalformed, headerErr)
	}
	if _, banned := forbiddenAlgs[headerAlg]; banned {
		return nil, fmt.Errorf("%w: alg=%q", ErrJWTAlgorithmNotAllowed, headerAlg)
	}
	allowedAlg := false
	for _, a := range cfg.AllowedAlgs {
		if a == headerAlg {
			allowedAlg = true
			break
		}
	}
	if !allowedAlg {
		return nil, fmt.Errorf("%w: alg=%q not in allowlist", ErrJWTAlgorithmNotAllowed, headerAlg)
	}

	parser := jwt.NewParser(jwt.WithValidMethods(cfg.AllowedAlgs))

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		alg, _ := token.Header["alg"].(string)
		if _, banned := forbiddenAlgs[alg]; banned {
			return nil, fmt.Errorf("%w: alg=%q", ErrJWTAlgorithmNotAllowed, alg)
		}
		// Enforce allowlist defensively (parser does this too, but a
		// belt-and-braces check guards against parser-internal default
		// drift if jwt/v5 changes WithValidMethods semantics).
		allowed := false
		for _, a := range cfg.AllowedAlgs {
			if a == alg {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("%w: alg=%q not in allowlist", ErrJWTAlgorithmNotAllowed, alg)
		}

		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("%w: token header has no kid", ErrJWTKeyNotFound)
		}
		return keySet.keyFor(ctx, kid)
	}

	token, err := parser.Parse(tokenString, keyFunc)
	if err != nil {
		// Surface our domain sentinels untouched if already wrapped.
		if errors.Is(err, ErrJWTAlgorithmNotAllowed) ||
			errors.Is(err, ErrJWTKeyNotFound) ||
			errors.Is(err, ErrJWKSEndpointUnreachable) ||
			errors.Is(err, ErrJWTSignatureVerificationNotImplemented) {
			return nil, err
		}
		// jwt/v5 surfaces signature failure via ErrSignatureInvalid.
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, fmt.Errorf("%w: %v", ErrJWTSignatureInvalid, err)
		}
		// Token-validity errors (exp/nbf/iat) — surface as claims-invalid.
		if errors.Is(err, jwt.ErrTokenExpired) ||
			errors.Is(err, jwt.ErrTokenNotValidYet) ||
			errors.Is(err, jwt.ErrTokenUsedBeforeIssued) {
			return nil, fmt.Errorf("%w: %v", ErrJWTClaimsInvalid, err)
		}
		// Malformed input or signing-method mismatch from WithValidMethods.
		if errors.Is(err, jwt.ErrTokenMalformed) || errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, fmt.Errorf("%w: %v", ErrJWTSignatureInvalid, err)
		}
		if errors.Is(err, jwt.ErrTokenUnverifiable) {
			return nil, fmt.Errorf("%w: %v", ErrJWTSignatureInvalid, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrJWTMalformed, err)
	}
	if !token.Valid {
		return nil, ErrJWTSignatureInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("%w: claims not MapClaims", ErrJWTClaimsInvalid)
	}

	// Issuer / audience / iat-with-skew validation. exp/nbf already
	// enforced by jwt/v5's default parser path.
	now := time.Now()

	if cfg.Issuer != "" {
		iss, _ := claims["iss"].(string)
		if iss != cfg.Issuer {
			return nil, fmt.Errorf("%w: iss=%q expected=%q", ErrJWTClaimsInvalid, iss, cfg.Issuer)
		}
	}

	if cfg.Audience != "" {
		matched := false
		switch a := claims["aud"].(type) {
		case string:
			matched = a == cfg.Audience
		case []interface{}:
			for _, v := range a {
				if s, ok := v.(string); ok && s == cfg.Audience {
					matched = true
					break
				}
			}
		}
		if !matched {
			return nil, fmt.Errorf("%w: aud mismatch (expected %q)", ErrJWTClaimsInvalid, cfg.Audience)
		}
	}

	// iat sanity: not too far in the future. jwt/v5 does not enforce
	// this by default — we add it because future-iat is a classic
	// replay-window signal.
	if iatRaw, ok := claims["iat"]; ok {
		var iat time.Time
		switch v := iatRaw.(type) {
		case float64:
			iat = time.Unix(int64(v), 0)
		case json.Number:
			n, _ := v.Int64()
			iat = time.Unix(n, 0)
		}
		if !iat.IsZero() && iat.After(now.Add(cfg.ClockSkew)) {
			return nil, fmt.Errorf("%w: iat in the future (iat=%s, now=%s, skew=%s)", ErrJWTClaimsInvalid, iat, now, cfg.ClockSkew)
		}
	}

	return claims, nil
}
