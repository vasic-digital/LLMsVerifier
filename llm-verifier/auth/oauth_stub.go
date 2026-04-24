//go:build !real_oauth
// +build !real_oauth

// Package auth — OAuth credential-reader STUBS (default build).
//
// This is the default build for the open-source tree: fresh clones do NOT
// ship `oauth_credentials.go` (it's gitignored and lives only on maintainer
// machines that carry the full CLI OAuth integration). So the stubs here
// provide the required symbol surface for `providers/` to compile.
//
// To build with the real, full OAuth integration (Claude Code + Qwen CLI
// token readers), drop the real `oauth_credentials.go` into this directory
// and build with `-tags=real_oauth`. The tag both activates the real file
// and de-activates this stub, avoiding duplicate declarations.
package auth

import "fmt"

// OAuthCredentialReader is the handle the providers carry for lazy token
// refresh. The stub form has no state because no refresh ever succeeds.
type OAuthCredentialReader struct{}

var globalOAuthReader = &OAuthCredentialReader{}

// GetGlobalOAuthReader returns the process-wide reader. Callers check
// Has*Credentials before pulling tokens so this is safe to return a
// singleton.
func GetGlobalOAuthReader() *OAuthCredentialReader {
	return globalOAuthReader
}

// IsClaudeOAuthEnabled reports whether the operator has opted into the
// Claude Code CLI OAuth flow. The stub always returns false — callers
// fall back to API-key authentication.
func IsClaudeOAuthEnabled() bool { return false }

// IsQwenOAuthEnabled reports whether the operator has opted into the
// Qwen CLI OAuth flow. Stubbed to false for the same reason as the
// Claude variant.
func IsQwenOAuthEnabled() bool { return false }

// HasValidClaudeCredentials reports whether the on-disk Claude OAuth
// token is present and unexpired. Stubbed to false.
func (r *OAuthCredentialReader) HasValidClaudeCredentials() bool { return false }

// HasValidQwenCredentials reports whether the on-disk Qwen OAuth
// token is present and unexpired. Stubbed to false.
func (r *OAuthCredentialReader) HasValidQwenCredentials() bool { return false }

// GetClaudeAccessToken returns the current Claude OAuth bearer token.
// Stubbed to return an actionable error so callers see WHY OAuth is
// unavailable instead of silently succeeding with an empty token.
func (r *OAuthCredentialReader) GetClaudeAccessToken() (string, error) {
	return "", fmt.Errorf("Claude OAuth integration is not compiled into this build; use an API key")
}

// GetQwenAccessToken mirrors GetClaudeAccessToken for the Qwen CLI
// OAuth flow.
func (r *OAuthCredentialReader) GetQwenAccessToken() (string, error) {
	return "", fmt.Errorf("Qwen OAuth integration is not compiled into this build; use an API key")
}

// ClaudeOAuthCredentials minimal stub matching the `real_oauth` file's shape
// enough for other default-build files (e.g. token_refresh.go) to compile.
// Fields mirror the live type so a future transition to the full integration
// is a drop-in replacement.
type ClaudeOAuthCredentials struct {
	ClaudeAiOauth *ClaudeAiOAuth `json:"claudeAiOauth"`
}

// ClaudeAiOAuth stub — see oauth_credentials.go (real_oauth build) for the live
// version. Field set kept identical so callers relying on access-token fields
// compile cleanly under both build tags.
type ClaudeAiOAuth struct {
	AccessToken      string   `json:"accessToken"`
	RefreshToken     string   `json:"refreshToken"`
	ExpiresAt        int64    `json:"expiresAt"`
	Scopes           []string `json:"scopes"`
	SubscriptionType string   `json:"subscriptionType"`
	RateLimitTier    string   `json:"rateLimitTier"`
}

// QwenOAuthCredentials stub — see oauth_credentials.go.
type QwenOAuthCredentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiryDate   int64  `json:"expiry_date"`
	TokenType    string `json:"token_type"`
	ResourceURL  string `json:"resource_url,omitempty"`
}

// ReadClaudeCredentials stub — always reports the integration as unavailable.
// The real implementation reads ~/.claude/credentials.json.
func (r *OAuthCredentialReader) ReadClaudeCredentials() (*ClaudeOAuthCredentials, error) {
	return nil, fmt.Errorf("Claude OAuth integration is not compiled into this build; use an API key")
}

// ReadQwenCredentials stub — mirrors ReadClaudeCredentials for the Qwen path.
func (r *OAuthCredentialReader) ReadQwenCredentials() (*QwenOAuthCredentials, error) {
	return nil, fmt.Errorf("Qwen OAuth integration is not compiled into this build; use an API key")
}

// ClearCache stub — real implementation flushes the credential cache so the
// next read re-consults the on-disk file. Stub is a no-op because there is
// no cache (reads always fail).
func (r *OAuthCredentialReader) ClearCache() {}

// GetClaudeCredentialsPath stub — real impl returns the platform-specific
// ~/.claude/credentials.json path. Stub returns an empty string so callers
// can at least report "no credentials path" without panicking.
func GetClaudeCredentialsPath() string { return "" }

// GetQwenCredentialsPath stub — real impl returns the Qwen CLI credentials
// path (typically ~/.qwen/credentials.json). Stub returns an empty string.
func GetQwenCredentialsPath() string { return "" }
