package config

import (
	"context"
	"strings"
	"testing"
)

// fakeConfigTranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// i18n-routed sentinel without coupling to the English bundle text. Anti-bluff
// per CONST-035 / Article XI §11.9: a test asserting the original literal would
// silently pass if the call-site bypassed the translator.
type fakeConfigTranslator struct{}

func (fakeConfigTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeConfigTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeConfigTranslator installs the fakeConfigTranslator, runs fn, then
// restores the prior translator.
func withFakeConfigTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeConfigTranslator{}
	defer func() { translator = prior }()
	fn()
}

// invalidConfig builds a Config that deliberately fails every validator branch
// migrated in round-415 so the assertion can prove each error message is routed
// through the i18n seam.
func invalidConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			BaseURL:      "ftp://bad-scheme.example.com",
			DefaultModel: "", // setDefaults will repopulate; cleared again below
		},
		LLMs: []LLMConfig{
			{
				Name:     "", // llm_name_empty
				Endpoint: "ftp://bad.example.com",
				APIKey:   "", // api_key_required (non-well-known)
			},
		},
		API: APIConfig{
			Port:      "not-a-number",
			JWTSecret: "short", // jwt_secret_too_short
			RateLimit: -1,
		},
		Monitoring: MonitoringConfig{
			EnableMetrics: true,
			MetricsPort:   "",
			EnableHealth:  true,
			HealthPort:    "",
		},
		Logging: LoggingConfig{
			Level:   "trace",   // log_level_invalid
			Format:  "xml",     // log_format_invalid
			Output:  "syslog",  // log_output_invalid
			MaxSize: -1,
			MaxAge:  -1,
		},
	}
}

// TestValidationMessages_Routed proves every migrated validation error message
// flows through the i18n seam. With the fake translator installed, each error
// must carry the "<TRANSLATED:config.validation.*>" prefix — if a branch still
// held an English literal, the assertion fails.
func TestValidationMessages_Routed(t *testing.T) {
	withFakeConfigTranslator(t, func() {
		cfg := invalidConfig()
		result := ValidateConfig(cfg)
		if result.Valid {
			t.Fatal("expected invalid config, got Valid=true")
		}
		if len(result.Errors) == 0 {
			t.Fatal("expected validation errors, got none")
		}
		for _, ve := range result.Errors {
			// ValidationError.Error() routes through trData.
			if !strings.HasPrefix(ve.Error(), "<TRANSLATED:config.validation.error_for_field>") {
				t.Errorf("ValidationError.Error() not i18n-routed: %q", ve.Error())
			}
			if !strings.HasPrefix(ve.Message, "<TRANSLATED:config.validation.") {
				t.Errorf("validation message for %q not i18n-routed: %q", ve.Field, ve.Message)
			}
		}
		// ValidationResult.Error() routes the failure summary.
		if !strings.HasPrefix(result.Error(), "<TRANSLATED:config.validation.failed_summary>") {
			t.Errorf("ValidationResult.Error() not i18n-routed: %q", result.Error())
		}
	})
}

// TestConfigMutationGuard is the paired-mutation test per §1.1. With the
// production-default NoopTranslator, the verbatim message ID is returned — NOT
// an English literal. A regression that re-hardcoded "JWT secret cannot be
// empty" would make the message differ from the bare ID, failing this test.
func TestConfigMutationGuard(t *testing.T) {
	if got := tr("config.validation.jwt_secret_empty"); got != "config.validation.jwt_secret_empty" {
		t.Fatalf("NoopTranslator must return the bare id; got %q", got)
	}
	if got := trData("config.validation.error_for_field", map[string]any{"field": "x", "message": "y"}); got != "config.validation.error_for_field" {
		t.Fatalf("NoopTranslator (trData) must return the bare id; got %q", got)
	}

	cfg := invalidConfig()
	result := ValidateConfig(cfg)
	for _, ve := range result.Errors {
		// Under NoopTranslator the message is the bare message ID — never an
		// English literal that regressed past the seam.
		if strings.Contains(ve.Message, "cannot be empty") ||
			strings.Contains(ve.Message, "must be one of") ||
			strings.Contains(ve.Message, "must be at least") {
			t.Fatalf("validation message regressed to a hardcoded English literal: %q", ve.Message)
		}
		if !strings.HasPrefix(ve.Message, "config.validation.") {
			t.Fatalf("validation message not routed through the i18n seam: %q", ve.Message)
		}
	}
}
