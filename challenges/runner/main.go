// Command runner is the LLMsVerifier round-296 Challenge runner
// (mirrors the LLMProvider round-292 / DocProcessor round-220
// templates).
//
// It exercises the real crush_config.SchemaValidator + ConfigLoader
// surfaces across five locale fixtures. Every PASS line is backed
// by a runtime invariant on the real validator — never a metadata-
// only check (CONST-035 / Article XI §11.9, CONST-050(A) paired
// mutation per §1.1).
//
// LLMsVerifier is the constitutional single source of truth for
// model and provider metadata (CONST-036/037/038/039/040 in
// consuming projects). The crush_config package is one of its
// public surfaces — it validates third-party tool configurations
// (Crush, OpenCode, etc.) against the verifier's schema before
// the consuming project trusts the embedded provider/model list.
// A bluff in this validator silently lets unvalidated providers
// reach end users — exactly the failure mode the operator
// mandated against on 2026-04-28.
//
// Anti-bluff invariants enforced (one batch per locale fixture):
//
//  1. crush_config.NewSchemaValidator() returns a non-nil
//     validator (real constructor contract).
//  2. ValidateFromReader on a minimal valid config (providers +
//     models present) returns Valid=true with zero errors.
//  3. ValidateFromReader on a config WITHOUT providers returns
//     Valid=false with at least one error mentioning the missing
//     field — proves structural validation actually runs (not a
//     hardcoded true).
//  4. ValidateFromReader on syntactically invalid JSON returns a
//     non-nil error containing "invalid JSON" — proves the JSON
//     parser is wired and surfaced.
//  5. ConfigLoader.LoadFromFile + SaveToFile round-trip on a real
//     temp file preserves the provider ID — proves persistence
//     wire (not in-memory map echo).
//
// Mutation hook: when env LLMSVERIFIER_MUTATE_RUNNER=1 is set,
// the runner inverts invariant (3) (treats a validator that
// SILENTLY ACCEPTS a config with no providers as PASS instead of
// FAIL). The paired Challenge wrapper asserts the runner exits
// non-zero under mutation and rewrites that to exit 99
// (paired-mutation success).
//
// Verbatim 2026-05-19 operator mandate (preserved per
// CONST-049 §11.4.17):
//
//	"all existing tests and Challenges do work in anti-bluff
//	manner - they MUST confirm that all tested codebase really
//	works as expected! We had been in position that all tests
//	do execute with success and all Challenges as well, but
//	in reality the most of the features does not work and
//	can't be used! This MUST NOT be the case and execution
//	of tests and Challenges MUST guarantee the quality, the
//	completition and full usability by end users of the
//	product!"
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	crush_config "digital.vasic.llmsverifier/pkg/crush/config"
)

// fixture is a 7-field projection of challenges/fixtures/<locale>.yaml.
// Minimal in-process parse keeps the runner free of new yaml deps
// (CONST-051(B): no transitive deps creeping into a reusable
// submodule).
type fixture struct {
	locale                       string
	providerID                   string
	prompt                       string
	expectValidMinimalConfig     bool
	expectInvalidNoProviders     bool // expected Valid for no-providers (false = expect Valid=false)
	expectInvalidJSONError       bool
	expectRoundtripPreserves     bool
}

func main() {
	if code := run(os.Stdout); code != 0 {
		os.Exit(code)
	}
}

func run(out io.Writer) int {
	fmt.Fprintln(out, "=== LLMsVerifier Challenge Runner (round-296) ===")

	fixDir := os.Getenv("LLMSVERIFIER_FIXTURES_DIR")
	if fixDir == "" {
		fixDir = filepath.Join("challenges", "fixtures")
	}

	fixtures, err := loadFixtures(fixDir)
	if err != nil {
		fmt.Fprintf(out, "FAIL: load fixtures from %s: %v\n",
			fixDir, err)
		return 1
	}
	if len(fixtures) < 5 {
		fmt.Fprintf(out, "FAIL: expected >=5 fixtures, got %d\n",
			len(fixtures))
		return 1
	}
	fmt.Fprintf(out, "[setup] loaded %d locale fixtures from %s\n",
		len(fixtures), fixDir)

	mutate := os.Getenv("LLMSVERIFIER_MUTATE_RUNNER") == "1"
	if mutate {
		fmt.Fprintln(out, "[setup] MUTATION MODE: invariant 3 polarity"+
			" inverted (PASS when validator silently accepts a"+
			" config with no providers)")
	}

	pass, fail := 0, 0
	step := func(name string, ok bool, detail string) {
		if ok {
			pass++
			fmt.Fprintf(out, "  PASS  %-58s  %s\n", name, detail)
			return
		}
		fail++
		fmt.Fprintf(out, "  FAIL  %-58s  %s\n", name, detail)
	}

	for _, f := range fixtures {
		// Invariant 1: real constructor.
		sv := crush_config.NewSchemaValidator()
		step("validator.constructor."+f.locale,
			sv != nil,
			fmt.Sprintf("nonNil=%v", sv != nil))
		if sv == nil {
			continue
		}

		// Invariant 2: validate a minimal-but-valid config.
		// Use the fixture's provider_id so each locale exercises
		// a distinct payload (no shared state across locales).
		validConfig := fmt.Sprintf(`{
  "providers": {
    "%s": {
      "type": "openai",
      "base_url": "https://api.example.com",
      "api_key": "sk-test",
      "models": [{"id": "test-model", "name": "Test Model"}]
    }
  },
  "models": {
    "default": {"model": "test-model", "provider": "%s"}
  }
}`, f.providerID, f.providerID)
		validResult, vErr := sv.ValidateFromReader(
			bytes.NewReader([]byte(validConfig)))
		gotValidMinimal := vErr == nil && validResult != nil &&
			validResult.Valid
		step("validator.valid_minimal_config."+f.locale,
			gotValidMinimal == f.expectValidMinimalConfig,
			fmt.Sprintf("got=%v want=%v err=%v errors=%d",
				gotValidMinimal, f.expectValidMinimalConfig, vErr,
				countErrors(validResult)))

		// Invariant 3: validate a config missing providers.
		noProvidersConfig := `{"models": {"default": {"model": "x", "provider": "y"}}}`
		npResult, npErr := sv.ValidateFromReader(
			bytes.NewReader([]byte(noProvidersConfig)))
		// Real surface should produce Valid=false (config without
		// providers is invalid per validateStructure's "required
		// field" check).
		gotInvalidNP := npErr == nil && npResult != nil &&
			!npResult.Valid && countErrors(npResult) > 0
		// f.expectInvalidNoProviders is false (we expect Valid=false).
		// So invariantOK = gotInvalidNP == true.
		invariantOK := gotInvalidNP
		if mutate {
			step("validator.rejects_no_providers."+f.locale+"[MUTATED]",
				!invariantOK,
				fmt.Sprintf("mutation-inverted: gotInvalidNP=%v"+
					" errors=%d", gotInvalidNP,
					countErrors(npResult)))
		} else {
			step("validator.rejects_no_providers."+f.locale,
				invariantOK,
				fmt.Sprintf("gotInvalidNP=%v errors=%d err=%v",
					gotInvalidNP, countErrors(npResult), npErr))
		}

		// Invariant 4: invalid JSON surfaces error.
		invalidJSON := `{this is not json`
		_, ijErr := sv.ValidateFromReader(
			bytes.NewReader([]byte(invalidJSON)))
		gotInvalidJSON := ijErr != nil &&
			strings.Contains(ijErr.Error(), "invalid JSON")
		step("validator.invalid_json_error."+f.locale,
			gotInvalidJSON == f.expectInvalidJSONError,
			fmt.Sprintf("got=%v want=%v err=%v",
				gotInvalidJSON, f.expectInvalidJSONError, ijErr))

		// Invariant 5: ConfigLoader round-trip preserves provider.
		// Real file system, real JSON marshal, real disk read.
		gotRoundtrip := configRoundtripPreservesProvider(
			f.providerID)
		step("loader.roundtrip_preserves_provider."+f.locale,
			gotRoundtrip == f.expectRoundtripPreserves,
			fmt.Sprintf("got=%v want=%v",
				gotRoundtrip, f.expectRoundtripPreserves))
	}

	fmt.Fprintf(out, "\n=== Summary: PASS=%d FAIL=%d ===\n",
		pass, fail)
	if fail > 0 {
		return 1
	}
	return 0
}

// countErrors returns the number of errors on a ValidationResult,
// gracefully handling nil.
func countErrors(r *crush_config.ValidationResult) int {
	if r == nil {
		return 0
	}
	return len(r.Errors)
}

// configRoundtripPreservesProvider performs a real file-system
// round-trip through ConfigLoader and returns true iff the
// provider ID is preserved across save+load.
func configRoundtripPreservesProvider(providerID string) bool {
	loader := crush_config.ConfigLoader{}
	cfg := &crush_config.Config{
		Providers: map[string]crush_config.ProviderConfig{
			providerID: {
				Type:    "openai",
				BaseURL: "https://api.example.com",
				APIKey:  "sk-test",
				Models: []crush_config.Model{
					{ID: "test-model", Name: "Test Model"},
				},
			},
		},
		Models: map[string]crush_config.SelectedModel{
			"default": {
				Model:    "test-model",
				Provider: providerID,
			},
		},
	}

	tmp, err := os.CreateTemp("", "llmsverifier-roundtrip-*.json")
	if err != nil {
		return false
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)

	if err := loader.SaveToFile(cfg, path); err != nil {
		return false
	}
	loaded, err := loader.LoadFromFile(path)
	if err != nil || loaded == nil {
		return false
	}
	_, ok := loaded.Providers[providerID]
	return ok
}

// loadFixtures parses every *.yaml in dir using a tiny line-based
// parser. We only support the 7 keys our fixtures use; anything
// else is ignored. Keeping the parser in-runner avoids pulling
// yaml.v3 into the runtime path of a submodule that other projects
// reuse (CONST-051(B)).
func loadFixtures(dir string) ([]fixture, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []fixture
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		f := parseFixture(string(data))
		if f.locale == "" {
			return nil, fmt.Errorf(
				"%s: missing locale key", e.Name())
		}
		out = append(out, f)
	}
	return out, nil
}

func parseFixture(text string) fixture {
	f := fixture{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		k := strings.TrimSpace(line[:colon])
		v := strings.TrimSpace(line[colon+1:])
		v = strings.Trim(v, "\"'")
		switch k {
		case "locale":
			f.locale = v
		case "provider_id":
			f.providerID = v
		case "prompt":
			f.prompt = v
		case "expect_valid_minimal_config":
			if b, err := strconv.ParseBool(v); err == nil {
				f.expectValidMinimalConfig = b
			}
		case "expect_invalid_no_providers":
			if b, err := strconv.ParseBool(v); err == nil {
				f.expectInvalidNoProviders = b
			}
		case "expect_invalid_json_error":
			if b, err := strconv.ParseBool(v); err == nil {
				f.expectInvalidJSONError = b
			}
		case "expect_roundtrip_preserves_provider":
			if b, err := strconv.ParseBool(v); err == nil {
				f.expectRoundtripPreserves = b
			}
		}
	}
	return f
}
