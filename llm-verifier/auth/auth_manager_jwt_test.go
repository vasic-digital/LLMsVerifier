package auth

// Round-59 §11.4 anti-bluff JWT JWKS-verification tests.
//
// These tests close round-28's ErrJWTSignatureVerificationNotImplemented
// HIGH-security gap by proving that SSOManager.ValidateToken with a
// configured JWKSURL performs REAL asymmetric-signature verification
// against a REAL HTTP-served JWKS endpoint (httptest.NewServer).
//
// CONST-035 / CONST-042 / CONST-050(A) / Article XI §11.9 anchors apply.
//
// Verbatim operator mandate (2026-05-19) preserved in the commit body
// per CONST-049 §11.4.17:
// "all existing tests and Challenges do work in anti-bluff manner - they
// MUST confirm that all tested codebase really works as expected! We had
// been in position that all tests do execute with success and all
// Challenges as well, but in reality the most of the features does not
// work and can't be used! This MUST NOT be the case and execution of
// tests and Challenges MUST guarantee the quality, the completition and
// full usability by end users of the product!"

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Test fixtures =======================================================

// testJWKSServer wraps an httptest.Server that serves a JWKS document
// constructed from one or more (kid, public-key) entries. fetchCount
// records every successful GET so cache-behaviour tests can assert.
type testJWKSServer struct {
	server     *httptest.Server
	fetchCount int64
}

func newTestJWKSServer(t *testing.T, entries map[string]interface{}) *testJWKSServer {
	t.Helper()
	s := &testJWKSServer{}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&s.fetchCount, 1)
		doc := jwksDocument{Keys: make([]jwkRaw, 0, len(entries))}
		for kid, k := range entries {
			j, err := publicKeyToJWK(kid, k)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			doc.Keys = append(doc.Keys, j)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *testJWKSServer) URL() string  { return s.server.URL }
func (s *testJWKSServer) Fetches() int { return int(atomic.LoadInt64(&s.fetchCount)) }

func publicKeyToJWK(kid string, k interface{}) (jwkRaw, error) {
	switch pk := k.(type) {
	case *rsa.PublicKey:
		nBytes := pk.N.Bytes()
		eBytes := big.NewInt(int64(pk.E)).Bytes()
		return jwkRaw{
			Kid: kid,
			Kty: "RSA",
			Alg: "RS256",
			N:   base64.RawURLEncoding.EncodeToString(nBytes),
			E:   base64.RawURLEncoding.EncodeToString(eBytes),
		}, nil
	case *ecdsa.PublicKey:
		var crv string
		switch pk.Curve {
		case elliptic.P256():
			crv = "P-256"
		case elliptic.P384():
			crv = "P-384"
		case elliptic.P521():
			crv = "P-521"
		default:
			return jwkRaw{}, fmt.Errorf("unsupported EC curve")
		}
		return jwkRaw{
			Kid: kid,
			Kty: "EC",
			Alg: "ES256",
			Crv: crv,
			X:   base64.RawURLEncoding.EncodeToString(pk.X.Bytes()),
			Y:   base64.RawURLEncoding.EncodeToString(pk.Y.Bytes()),
		}, nil
	default:
		return jwkRaw{}, fmt.Errorf("unsupported key type %T", k)
	}
}

// signRSAToken signs `claims` with `priv` using RS256 + kid in header.
func signRSAToken(t *testing.T, priv *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(priv)
	require.NoError(t, err)
	return s
}

// signECToken signs claims with ES256 + kid.
func signECToken(t *testing.T, priv *ecdsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(priv)
	require.NoError(t, err)
	return s
}

// signHMACToken signs claims with HS256 — should be REJECTED by SSO path.
func signHMACToken(t *testing.T, secret []byte, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(secret)
	require.NoError(t, err)
	return s
}

// noneToken builds an unsigned `alg: none` token — MUST be rejected.
func noneToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	header := map[string]interface{}{"alg": "none", "typ": "JWT", "kid": "anykid"}
	hJSON, _ := json.Marshal(header)
	cJSON, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(hJSON) + "." +
		base64.RawURLEncoding.EncodeToString(cJSON) + "."
}

func defaultClaims(iss, aud string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss":   iss,
		"aud":   aud,
		"sub":   "user-12345",
		"email": "user@example.com",
		"name":  "Test User",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
}

// ===== Tests ==============================================================

// TestValidateToken_NoSSOConfig_ReturnsSentinel preserves round-28
// semantics for providers registered WITHOUT a JWKSURL.
func TestValidateToken_NoSSOConfig_ReturnsSentinel(t *testing.T) {
	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{Provider: "google", Issuer: "https://accounts.google.com"})

	wellFormed := "aGVhZGVy.eyJzdWIiOiJ2aWN0aW0ifQ.c2ln"
	info, err := sm.ValidateToken("google", wellFormed)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWTSignatureVerificationNotImplemented,
		"round-28 sentinel MUST fire when provider configured without JWKSURL")
}

// TestValidateToken_NoneAlgorithm_Rejected — `alg: none` is a classic
// auth-bypass; MUST be refused with ErrJWTAlgorithmNotAllowed regardless
// of any other config.
func TestValidateToken_NoneAlgorithm_Rejected(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	tok := noneToken(t, defaultClaims("https://idp.example.com", "client-123"))
	info, err := sm.ValidateToken("idp", tok)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWTAlgorithmNotAllowed,
		"`none` algorithm MUST be permanently forbidden for SSO (CONST-035 / CONST-042)")
}

// TestValidateToken_HS256_Rejected — HMAC algorithms imply a shared
// secret; using them for SSO is an alg-confusion bypass vector.
func TestValidateToken_HS256_Rejected(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	tok := signHMACToken(t, []byte("attacker-known-secret"), "k1",
		defaultClaims("https://idp.example.com", "client-123"))
	info, err := sm.ValidateToken("idp", tok)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWTAlgorithmNotAllowed,
		"HS* (HMAC) algorithms MUST be permanently forbidden for SSO — alg-confusion bypass risk")
}

// TestValidateToken_ValidRSASignedToken_AcceptedWithClaims is the happy
// path: real RSA-2048 signature against a real JWKS endpoint passes
// verification and produces SSOUserInfo synthesised from VERIFIED claims.
func TestValidateToken_ValidRSASignedToken_AcceptedWithClaims(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	claims := defaultClaims("https://idp.example.com", "client-123")
	claims["groups"] = []interface{}{"engineers", "admins"}
	tok := signRSAToken(t, priv, "k1", claims)

	info, err := sm.ValidateToken("idp", tok)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "idp", info.Provider)
	assert.Equal(t, "https://idp.example.com", info.Issuer)
	assert.Equal(t, "user-12345", info.Subject)
	assert.Equal(t, "user@example.com", info.Email)
	assert.Equal(t, "Test User", info.Name)
	assert.ElementsMatch(t, []string{"engineers", "admins"}, info.Groups)
}

// TestValidateToken_ValidECSignedToken_AcceptedWithClaims — same happy
// path but with ES256 / P-256 to prove EC keys parse correctly too.
func TestValidateToken_ValidECSignedToken_AcceptedWithClaims(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"ec1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	tok := signECToken(t, priv, "ec1", defaultClaims("https://idp.example.com", "client-123"))
	info, err := sm.ValidateToken("idp", tok)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "user-12345", info.Subject)
}

// TestValidateToken_TamperedSignature_Rejected: flip a byte in the
// signature and assert ErrJWTSignatureInvalid surfaces.
func TestValidateToken_TamperedSignature_Rejected(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	tok := signRSAToken(t, priv, "k1", defaultClaims("https://idp.example.com", "client-123"))
	parts := strings.Split(tok, ".")
	require.Len(t, parts, 3)
	// Flip a byte in the signature segment (decode, mutate, re-encode).
	sigBytes, _ := base64.RawURLEncoding.DecodeString(parts[2])
	require.NotEmpty(t, sigBytes)
	sigBytes[0] ^= 0xFF
	parts[2] = base64.RawURLEncoding.EncodeToString(sigBytes)
	tampered := strings.Join(parts, ".")

	info, err := sm.ValidateToken("idp", tampered)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWTSignatureInvalid,
		"tampered signature MUST be rejected with ErrJWTSignatureInvalid")
}

// TestValidateToken_UnknownKid_Rejected: token signed with key not in JWKS.
func TestValidateToken_UnknownKid_Rejected(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	tok := signRSAToken(t, priv, "rotated-out", defaultClaims("https://idp.example.com", "client-123"))
	info, err := sm.ValidateToken("idp", tok)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWTKeyNotFound,
		"kid not in JWKS MUST surface ErrJWTKeyNotFound")
}

// TestValidateToken_ExpiredToken_Rejected: exp in the past -> claims invalid.
func TestValidateToken_ExpiredToken_Rejected(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	claims := defaultClaims("https://idp.example.com", "client-123")
	claims["iat"] = time.Now().Add(-2 * time.Hour).Unix()
	claims["exp"] = time.Now().Add(-1 * time.Hour).Unix()
	tok := signRSAToken(t, priv, "k1", claims)

	info, err := sm.ValidateToken("idp", tok)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWTClaimsInvalid)
}

// TestValidateToken_FutureIatToken_Rejected_OutsideSkewWindow: iat far
// in the future should be rejected (replay-window signal).
func TestValidateToken_FutureIatToken_Rejected_OutsideSkewWindow(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider:  "idp",
		Issuer:    "https://idp.example.com",
		Audience:  "client-123",
		JWKSURL:   js.URL(),
		ClockSkew: 30 * time.Second, // explicit, tight
	})

	claims := defaultClaims("https://idp.example.com", "client-123")
	// 10 minutes in the future — way outside 30s skew.
	claims["iat"] = time.Now().Add(10 * time.Minute).Unix()
	// exp must still be after iat or jwt/v5 rejects on other grounds.
	claims["exp"] = time.Now().Add(20 * time.Minute).Unix()
	tok := signRSAToken(t, priv, "k1", claims)

	info, err := sm.ValidateToken("idp", tok)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWTClaimsInvalid,
		"iat far in the future MUST be rejected with ErrJWTClaimsInvalid")
}

// TestValidateToken_WrongIssuer_Rejected.
func TestValidateToken_WrongIssuer_Rejected(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	claims := defaultClaims("https://attacker.example.com", "client-123")
	tok := signRSAToken(t, priv, "k1", claims)

	info, err := sm.ValidateToken("idp", tok)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWTClaimsInvalid)
}

// TestValidateToken_WrongAudience_Rejected.
func TestValidateToken_WrongAudience_Rejected(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	claims := defaultClaims("https://idp.example.com", "different-client")
	tok := signRSAToken(t, priv, "k1", claims)

	info, err := sm.ValidateToken("idp", tok)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWTClaimsInvalid)
}

// TestJWKSCache_RefreshOnTTLExpiry: with a tiny TTL, a second call
// after TTL elapses MUST trigger a second fetch.
func TestJWKSCache_RefreshOnTTLExpiry(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	cfg := (&JWKSConfig{
		JWKSURL:  js.URL(),
		Issuer:   "iss",
		Audience: "aud",
		CacheTTL: 50 * time.Millisecond,
	}).resolveDefaults()
	ks := newJWKSKeySet(cfg)

	// First call -> refresh #1
	_, err = ks.keyFor(context.Background(), "k1")
	require.NoError(t, err)
	assert.Equal(t, 1, js.Fetches())

	// Second call within TTL -> NO refresh
	_, err = ks.keyFor(context.Background(), "k1")
	require.NoError(t, err)
	assert.Equal(t, 1, js.Fetches())

	// Wait past TTL
	time.Sleep(80 * time.Millisecond)

	// Third call -> refresh #2
	_, err = ks.keyFor(context.Background(), "k1")
	require.NoError(t, err)
	assert.Equal(t, 2, js.Fetches(), "stale cache MUST trigger refetch")
}

// TestJWKSCache_DoesNotRefetchOnEveryCall: 100 calls within TTL must
// produce exactly ONE fetch (DoS-against-IdP protection).
func TestJWKSCache_DoesNotRefetchOnEveryCall(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	cfg := (&JWKSConfig{
		JWKSURL:  js.URL(),
		Issuer:   "iss",
		Audience: "aud",
		CacheTTL: time.Hour,
	}).resolveDefaults()
	ks := newJWKSKeySet(cfg)

	for i := 0; i < 100; i++ {
		_, err := ks.keyFor(context.Background(), "k1")
		require.NoError(t, err)
	}
	assert.Equal(t, 1, js.Fetches(), "100 calls within TTL MUST produce exactly 1 fetch")
}

// TestRound59Sentinels_DistinguishableFromRound28Sentinel: errors.Is
// MUST cleanly distinguish round-59 sentinels from the preserved
// round-28 sentinel.
func TestRound59Sentinels_DistinguishableFromRound28Sentinel(t *testing.T) {
	round59 := []error{
		ErrJWTSignatureInvalid,
		ErrJWTKeyNotFound,
		ErrJWTAlgorithmNotAllowed,
		ErrJWKSEndpointUnreachable,
		ErrJWTClaimsInvalid,
		ErrJWTMalformed,
	}
	for _, s := range round59 {
		assert.False(t, errors.Is(s, ErrJWTSignatureVerificationNotImplemented),
			"round-59 sentinel %v MUST NOT be confused with round-28 sentinel", s)
		assert.False(t, errors.Is(ErrJWTSignatureVerificationNotImplemented, s),
			"round-28 sentinel MUST NOT match round-59 sentinel %v", s)
	}
}

// TestJWKSEndpointUnreachable_ReturnsSentinel: when the configured
// JWKS URL is unreachable AND no cached key exists, the verifier MUST
// surface ErrJWKSEndpointUnreachable rather than silently allowing.
func TestJWKSEndpointUnreachable_ReturnsSentinel(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	sm := NewSSOManager()
	sm.AddProvider(&SSOConfig{
		Provider: "idp",
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		// Point at a guaranteed-unreachable endpoint.
		JWKSURL: "http://127.0.0.1:1/non-existent-jwks",
	})

	tok := signRSAToken(t, priv, "k1", defaultClaims("https://idp.example.com", "client-123"))
	info, err := sm.ValidateToken("idp", tok)
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrJWKSEndpointUnreachable,
		"unreachable JWKS endpoint MUST surface ErrJWKSEndpointUnreachable")
}

// TestAuthenticateWithSSO_WithJWKS_ProducesAuthenticatedClient: end-to-end
// via the AuthManager wrapper — proves round-28 sentinel no longer
// short-circuits the SSO flow when JWKS is properly configured.
func TestAuthenticateWithSSO_WithJWKS_ProducesAuthenticatedClient(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	js := newTestJWKSServer(t, map[string]interface{}{"k1": &priv.PublicKey})

	sm := GetSSOManager()
	// Use unique provider name so we don't collide with parallel tests
	// in this file (GetSSOManager returns a process-global singleton).
	providerName := fmt.Sprintf("idp-round59-%d", time.Now().UnixNano())
	sm.AddProvider(&SSOConfig{
		Provider: providerName,
		Issuer:   "https://idp.example.com",
		Audience: "client-123",
		JWKSURL:  js.URL(),
	})

	claims := defaultClaims("https://idp.example.com", "client-123")
	claims["name"] = "Round-59 User"
	claims["groups"] = []interface{}{"editor"}
	tok := signRSAToken(t, priv, "k1", claims)

	am := NewAuthManager("test-secret")
	am.EnableSSO()
	client, err := am.AuthenticateWithSSO(providerName, tok)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "Round-59 User", client.Name)
	assert.Contains(t, client.Permissions, "read")
	assert.Contains(t, client.Permissions, "write")
}
