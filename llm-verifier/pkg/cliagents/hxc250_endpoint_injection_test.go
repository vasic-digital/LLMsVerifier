package cliagents

// HXC-250 — CONST-051(B) decoupling: a consuming project's HelixAgent endpoint
// was hardcoded into this reusable submodule, so generated CLI-agent configs
// pointed every end user at one fixed host:port no matter what the consumer
// deployed.
//
// §11.4.115 polarity switch — ONE source, TWO roles:
//
//	RED_MODE=1 (default) — reproduce the defect on the CURRENT artifact: assert
//	                       the hardcoded endpoint REACHES the emitted config
//	                       artifacts despite a different endpoint being injected.
//	RED_MODE=0           — standing GREEN regression guard: assert the injected
//	                       endpoint reaches the artifacts and the hardcoded
//	                       endpoint appears ZERO times.
//
// Also a §11.4.146 STEP-1 investigation instrument: it enumerates, in BOTH
// polarities, which emitted artifacts carry a wrong endpoint (blast radius).
//
// Two injection modes are exercised, because they failed differently:
//
//	config — DefaultGeneratorConfig() then set HelixAgentHost/Port (the natural
//	         consumer usage). Pre-fix this leaked in 48/48 agent configs,
//	         because MCPServers had already been built from the literal.
//	env    — inject via HELIX_AGENT_HOST / HELIX_AGENT_PORT and touch no field.
//	         Pre-fix the env was never consulted at all.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	crush_config "digital.vasic.llmsverifier/pkg/crush/config"
	"digital.vasic.llmsverifier/pkg/helixendpoint"
)

// The endpoint literal this submodule must never emit on its own initiative.
// Written split so this test file is not itself a hit for the source census.
const (
	hxc250LeakedHost = "local" + "host"
	hxc250LeakedPort = "81" + "00"
	hxc250LeakedURL  = "http://" + hxc250LeakedHost + ":" + hxc250LeakedPort
)

// Endpoint the test INJECTS. Deliberately not any real deployment.
const (
	hxc250InjectedHost = "helix-agent.injected.invalid"
	hxc250InjectedPort = 19099
)

// hxc250RedMode reads the §11.4.115 polarity switch.
//
// The fix has landed, so the STANDING default is GREEN (RED_MODE unset or 0):
// the same source now runs as the permanent regression guard in the normal
// suite. RED_MODE=1 is the opt-in reproduction mode, used to re-prove the
// guard against a pre-fix artifact (revert the source fix, run with
// RED_MODE=1, observe the defect reproduce).
func hxc250RedMode(t *testing.T) bool {
	t.Helper()
	switch os.Getenv("RED_MODE") {
	case "", "0":
		return false
	case "1":
		return true
	default:
		t.Fatalf("RED_MODE must be 0 or 1, got %q", os.Getenv("RED_MODE"))
		return false
	}
}

type hxc250Artifact struct {
	Label       string
	Path        string
	Leaks       int
	HasInjected bool
}

func hxc250Census(t *testing.T, label, path string) hxc250Artifact {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	body := string(raw)
	return hxc250Artifact{
		Label:       label,
		Path:        path,
		Leaks:       strings.Count(body, hxc250LeakedURL),
		HasInjected: strings.Contains(body, hxc250InjectedHost),
	}
}

func hxc250WriteJSON(t *testing.T, root, label string, v any) hxc250Artifact {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", label, err)
	}
	safe := strings.NewReplacer("/", "_", "(", "", ")", "", ".", "_", " ", "_").Replace(label)
	path := filepath.Join(root, safe+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return hxc250Census(t, label, path)
}

// hxc250EmitAgents drives the REAL generator for every supported agent and
// writes REAL config files through the real SaveConfig path, one directory per
// agent so agents sharing a ConfigFileName do not overwrite each other.
func hxc250EmitAgents(t *testing.T, root string, mutate func(*GeneratorConfig)) []hxc250Artifact {
	t.Helper()

	// Silence the generator's stderr DEBUG spew so the census stays readable.
	//
	// NOTE: this swaps the PROCESS-GLOBAL os.Stderr, so it is not safe under
	// t.Parallel(). No test in this package calls t.Parallel(); if that ever
	// changes, this must move to a per-generator io.Writer instead.
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	saved := os.Stderr
	os.Stderr = devnull
	defer func() {
		os.Stderr = saved
		_ = devnull.Close()
	}()

	probeCfg := DefaultGeneratorConfig()
	probeCfg.OutputDir = root
	agents := NewUnifiedGenerator(probeCfg).ListSupportedAgents()
	sort.Slice(agents, func(i, j int) bool { return agents[i] < agents[j] })

	var artifacts []hxc250Artifact
	ctx := context.Background()

	for _, agentType := range agents {
		agentDir := filepath.Join(root, string(agentType))
		if err := os.MkdirAll(agentDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", agentDir, err)
		}

		cfg := DefaultGeneratorConfig()
		cfg.OutputDir = agentDir
		if mutate != nil {
			mutate(cfg)
		}

		ug := NewUnifiedGenerator(cfg)
		result, err := ug.Generate(ctx, agentType)
		if err != nil || result == nil || result.Config == nil {
			// Emits no config, so no endpoint to leak. Recorded rather than
			// silently dropped (§11.4.6).
			artifacts = append(artifacts, hxc250Artifact{Label: string(agentType) + " (no config emitted)"})
			continue
		}
		if err := ug.SaveConfig(result); err != nil {
			t.Fatalf("SaveConfig(%s): %v", agentType, err)
		}
		artifacts = append(artifacts, hxc250Census(t, string(agentType), result.ConfigPath))
	}

	return artifacts
}

func hxc250Report(t *testing.T, scenario string, artifacts []hxc250Artifact) (emitted, leaking, injected int, labels []string) {
	t.Helper()
	for _, a := range artifacts {
		if a.Path == "" {
			continue
		}
		emitted++
		if a.Leaks > 0 {
			leaking++
			labels = append(labels, fmt.Sprintf("%s(x%d)", a.Label, a.Leaks))
		}
		if a.HasInjected {
			injected++
		}
	}
	t.Logf("[%s] census: emitted=%d  carrying-hardcoded-endpoint=%d  carrying-injected-endpoint=%d",
		scenario, emitted, leaking, injected)
	if len(labels) > 0 {
		t.Logf("[%s] leaking artifacts: %s", scenario, strings.Join(labels, ", "))
	}
	return
}

func hxc250Assert(t *testing.T, scenario string, red bool, emitted, leaking, injected int, labels []string) {
	t.Helper()

	// §11.4.201 instrument validation: a zero-leak verdict is only meaningful
	// if artifacts were actually produced and actually contain the endpoint.
	if emitted == 0 {
		t.Fatalf("[%s] instrument invalid: zero artifacts emitted — any leak count would be vacuous", scenario)
	}

	if red {
		if leaking == 0 {
			t.Fatalf("[%s] RED_MODE=1 expected the hardcoded endpoint %s to reach the emitted artifacts, "+
				"but all %d were clean — defect not reproduced", scenario, hxc250LeakedURL, emitted)
		}
		t.Logf("[%s] RED confirmed: %d/%d artifacts carry %s despite an injected endpoint of %s:%d",
			scenario, leaking, emitted, hxc250LeakedURL, hxc250InjectedHost, hxc250InjectedPort)
		return
	}

	if leaking != 0 {
		t.Errorf("[%s] RED_MODE=0: %d/%d artifacts still carry the hardcoded endpoint %s: %s",
			scenario, leaking, emitted, hxc250LeakedURL, strings.Join(labels, ", "))
	}
	// EVERY emitted artifact must carry the injected endpoint, not merely one.
	//
	// A weaker `injected == 0` check is not enough, and this is measured rather
	// than assumed: a paired §1.1 mutation that made helixendpoint.Host() ignore
	// the injected environment left 1 of 51 artifacts carrying the injected host
	// — the one surface that receives it as an explicit argument — and the
	// weaker check passed the mutant GREEN. Partial injection is exactly the
	// regression this guard exists to catch, and with no leaked literal to trip
	// the leak counter it is otherwise indistinguishable from a clean run.
	if injected != emitted {
		t.Errorf("[%s] RED_MODE=0: the injected endpoint %s reached only %d of %d artifacts — "+
			"%d carry neither the injected nor the hardcoded endpoint, so the zero-leak "+
			"result is vacuous for them (partial-injection regression)",
			scenario, hxc250InjectedHost, injected, emitted, emitted-injected)
	}
	if !t.Failed() {
		t.Logf("[%s] GREEN confirmed: 0 leaks across %d artifacts; injected endpoint present in %d",
			scenario, emitted, injected)
	}
}

// TestHXC250_ConfigInjectionReachesEmittedArtifacts exercises the natural
// consumer usage: take the default config, set the endpoint fields, generate.
func TestHXC250_ConfigInjectionReachesEmittedArtifacts(t *testing.T) {
	red := hxc250RedMode(t)
	root := t.TempDir()

	artifacts := hxc250EmitAgents(t, root, func(cfg *GeneratorConfig) {
		cfg.HelixAgentHost = hxc250InjectedHost
		cfg.HelixAgentPort = hxc250InjectedPort
		// Deliberately does NOT re-derive cfg.MCPServers — a consumer should
		// not have to know that setting the endpoint requires a second call.
	})

	// NOTE: this scenario deliberately references ONLY API that existed before
	// the fix, so the same source compiles against the broken artifact and its
	// RED result is a real reproduction rather than a build failure
	// (§11.4.201 — a broken instrument's "failure" is not evidence).

	emitted, leaking, injected, labels := hxc250Report(t, "config-injection", artifacts)
	hxc250Assert(t, "config-injection", red, emitted, leaking, injected, labels)
}

// TestHXC250_EnvInjectionReachesEmittedArtifacts exercises injection through
// the environment with no field assignment at all — the path a consuming
// project uses when it configures the submodule from its own deployment env.
func TestHXC250_EnvInjectionReachesEmittedArtifacts(t *testing.T) {
	red := hxc250RedMode(t)
	t.Setenv(helixendpoint.EnvHost, hxc250InjectedHost)
	t.Setenv(helixendpoint.EnvPort, fmt.Sprintf("%d", hxc250InjectedPort))

	root := t.TempDir()
	artifacts := hxc250EmitAgents(t, root, nil)

	// Zero-argument surfaces: these can only follow the injected environment.
	artifacts = append(artifacts,
		hxc250WriteJSON(t, root, "DefaultMCPServers()", DefaultMCPServers()),
		hxc250WriteJSON(t, root, "ContainerizedMCPServers(host)", ContainerizedMCPServers(hxc250InjectedHost)),
		hxc250WriteJSON(t, root, "crush_config.CreateDefaultConfig()", crush_config.CreateDefaultConfig()),
	)

	emitted, leaking, injected, labels := hxc250Report(t, "env-injection", artifacts)
	hxc250Assert(t, "env-injection", red, emitted, leaking, injected, labels)
}
