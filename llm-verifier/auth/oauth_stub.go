// Package auth — OAuth credential-reader stubs.
//
// The providers/ package references a small surface (OAuthCredentialReader,
// GetGlobalOAuthReader, IsClaudeOAuthEnabled, IsQwenOAuthEnabled) that was
// previously delivered by an OAuth integration with the Claude Code CLI and
// the Qwen CLI. That integration is not part of the open-source LLMsVerifier
// build and its implementation file is no longer checked in — without these
// stubs the providers/ package fails to compile and every downstream test
// fails with `[setup failed]`.
//
// The stubs keep the providers/ package compiling and make the OAuth path a
// clean no-op: `Is*OAuthEnabled` returns false, so `NewAnthropicAdapterAuto`
// and `NewQwenAdapterAuto` fall through to their API-key paths. Anyone
// re-introducing the real CLI integration should replace this file (or
// override these symbols via a build tag) with their full implementation.
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
