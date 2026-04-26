package cliagents

import (
	"context"
	"testing"
)

// TestHelixCodeGenerator_AuthFromJWTSecretEnv covers Finding #24:
// the generator must include an `auth.jwt_secret` block populated from
// JWT_SECRET so the installed HelixCode CLI can mint bearer tokens that
// HelixAgent's middleware accepts. Without this, every install required
// a manual hand-patch.
func TestHelixCodeGenerator_AuthFromJWTSecretEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-jwt-secret-from-env")
	t.Setenv("HELIXAGENT_JWT_SECRET", "")

	gen := NewHelixCodeGenerator()
	result, err := gen.Generate(context.Background(), &GeneratorConfig{
		HelixAgentHost: "localhost",
		HelixAgentPort: 8100,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	cfg, ok := result.Config.(*HelixCodeConfig)
	if !ok {
		t.Fatalf("Config is not *HelixCodeConfig (got %T)", result.Config)
	}
	if cfg.Auth == nil {
		t.Fatal("expected Auth block to be populated, got nil")
	}
	if cfg.Auth.JWTSecret != "test-jwt-secret-from-env" {
		t.Errorf("Auth.JWTSecret = %q, want %q",
			cfg.Auth.JWTSecret, "test-jwt-secret-from-env")
	}
}

// TestHelixCodeGenerator_AuthFromHelixagentFallback verifies that
// HELIXAGENT_JWT_SECRET is honored when JWT_SECRET is absent.
func TestHelixCodeGenerator_AuthFromHelixagentFallback(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("HELIXAGENT_JWT_SECRET", "fallback-jwt-secret")

	gen := NewHelixCodeGenerator()
	result, err := gen.Generate(context.Background(), &GeneratorConfig{
		HelixAgentHost: "localhost",
		HelixAgentPort: 8100,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	cfg := result.Config.(*HelixCodeConfig)
	if cfg.Auth == nil || cfg.Auth.JWTSecret != "fallback-jwt-secret" {
		t.Errorf("Auth.JWTSecret = %#v, want fallback-jwt-secret",
			cfg.Auth)
	}
}

// TestHelixCodeGenerator_AuthPlaceholderWhenNoEnv verifies the generator
// emits a discoverable placeholder when neither env var is set, so the
// installer can spot the missing value rather than silently shipping an
// empty secret.
func TestHelixCodeGenerator_AuthPlaceholderWhenNoEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("HELIXAGENT_JWT_SECRET", "")

	gen := NewHelixCodeGenerator()
	result, err := gen.Generate(context.Background(), &GeneratorConfig{
		HelixAgentHost: "localhost",
		HelixAgentPort: 8100,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	cfg := result.Config.(*HelixCodeConfig)
	if cfg.Auth == nil || cfg.Auth.JWTSecret != "<YOUR_JWT_SECRET>" {
		t.Errorf("Auth.JWTSecret = %#v, want placeholder", cfg.Auth)
	}
}
