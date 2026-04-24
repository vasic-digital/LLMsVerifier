//go:build oauth_stub
// +build oauth_stub

// Package auth — OAuth credential-reader STUBS.
//
// This file compiles only with `-tags=oauth_stub`. The real, non-stub
// implementation in `oauth_credentials.go` compiles by default and provides
// the full OAuth path for the Claude Code and Qwen CLI integrations.
//
// Keep this stub around so that operators who don't have (or don't want to
// compile) the full OAuth integration can build with `-tags=oauth_stub`
// and still have the providers/ package link correctly. Without the tag,
// this file is skipped and there is no symbol collision with
// oauth_credentials.go.
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
