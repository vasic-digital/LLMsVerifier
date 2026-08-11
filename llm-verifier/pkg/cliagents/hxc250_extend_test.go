package cliagents

// HXC-250 STEP-3 (§11.4.146) — extend the reproduce/confirm pair across the
// full case-space of endpoint injection in this package: every injection
// surface, boundary and malformed input, IPv6 topology, consumer-supplied MCP
// entries, and the capability-preservation invariant (§11.4.122).

import (
	"net/url"
	"strings"
	"testing"

	crush_config "digital.vasic.llmsverifier/pkg/crush/config"
	"digital.vasic.llmsverifier/pkg/helixendpoint"
)

func helixAgentURLs(servers []MCPServerConfig) []string {
	var out []string
	for _, s := range servers {
		if s.Type == "remote" && strings.HasPrefix(s.Name, helixAgentMCPNamePrefix) {
			out = append(out, s.URL)
		}
	}
	return out
}

// ---- Case group A: DefaultMCPServersForHost across host/port topologies ----

func TestHXC250Extend_DefaultMCPServersForHostTopologies(t *testing.T) {
	cases := []struct {
		name       string
		host       string
		port       int
		wantPrefix string
	}{
		{"plain host", "agent.internal", 7061, "http://agent.internal:7061/v1/"},
		{"IPv4 literal", "10.0.0.5", 7061, "http://10.0.0.5:7061/v1/"},
		{"IPv6 literal is bracketed", "::1", 7061, "http://[::1]:7061/v1/"},
		{"already-bracketed IPv6", "[::1]", 7061, "http://[::1]:7061/v1/"},
		{"blank host falls back", "", 7061, "http://localhost:7061/v1/"},
		{"invalid port falls back", "agent.internal", 0, "http://agent.internal:8100/v1/"},
		{"max port", "agent.internal", 65535, "http://agent.internal:65535/v1/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			urls := helixAgentURLs(DefaultMCPServersForHost(tc.host, tc.port))
			if len(urls) == 0 {
				t.Fatal("instrument invalid: no HelixAgent MCP entries produced")
			}
			for _, u := range urls {
				if !strings.HasPrefix(u, tc.wantPrefix) {
					t.Errorf("URL %q does not start with %q", u, tc.wantPrefix)
				}
				parsed, err := url.Parse(u)
				if err != nil {
					t.Errorf("emitted unparseable URL %q: %v", u, err)
					continue
				}
				if parsed.Host == "" {
					t.Errorf("emitted URL %q has empty host (malformed)", u)
				}
			}
		})
	}
}

// ---- Case group B: capability preservation (§11.4.122) ----

func TestHXC250Extend_CapabilityPreserved(t *testing.T) {
	servers := DefaultMCPServersForHost("agent.internal", 7061)

	if got := len(servers); got != DefaultMCPServersCount() {
		t.Errorf("MCP server count = %d, want documented %d", got, DefaultMCPServersCount())
	}

	// All six named HelixAgent capabilities must still ship.
	required := []string{
		"helixagent-mcp", "helixagent-acp", "helixagent-lsp",
		"helixagent-embeddings", "helixagent-vision", "helixagent-cognee",
	}
	present := map[string]bool{}
	for _, s := range servers {
		present[s.Name] = true
	}
	for _, name := range required {
		if !present[name] {
			t.Errorf("capability %q was removed — §11.4.122 silent-removal violation", name)
		}
	}

	// Containerized set keeps the same six.
	cPresent := map[string]bool{}
	for _, s := range ContainerizedMCPServersForHostPort("agent.internal", 7061) {
		cPresent[s.Name] = true
	}
	for _, name := range required {
		if !cPresent[name] {
			t.Errorf("containerized capability %q was removed", name)
		}
	}
	if got := len(ContainerizedMCPServers("agent.internal")); got != ContainerizedMCPServersCount() {
		t.Errorf("containerized count = %d, want documented %d", got, ContainerizedMCPServersCount())
	}
}

// ---- Case group C: ContainerizedMCPServers honours its own host ----

func TestHXC250Extend_ContainerizedHonoursHost(t *testing.T) {
	const host = "containers.internal"
	servers := ContainerizedMCPServersForHostPort(host, 7061)

	for _, s := range servers {
		if s.Type != "remote" {
			continue
		}
		parsed, err := url.Parse(s.URL)
		if err != nil {
			t.Errorf("%s: unparseable URL %q: %v", s.Name, s.URL, err)
			continue
		}
		// Third-party remote MCPs (context7 etc.) legitimately point elsewhere.
		if parsed.Scheme == "https" {
			continue
		}
		if !strings.HasPrefix(parsed.Host, host+":") {
			t.Errorf("%s: URL %q does not target the supplied host %q", s.Name, s.URL, host)
		}
	}
}

func TestHXC250Extend_ContainerizedIPv6IsBracketed(t *testing.T) {
	for _, s := range ContainerizedMCPServersForHostPort("::1", 7061) {
		if s.Type != "remote" || strings.HasPrefix(s.URL, "https://") {
			continue
		}
		if strings.Contains(s.URL, "http://::1:") {
			t.Errorf("%s: emitted malformed unbracketed IPv6 URL %q", s.Name, s.URL)
		}
		if _, err := url.Parse(s.URL); err != nil {
			t.Errorf("%s: unparseable URL %q: %v", s.Name, s.URL, err)
		}
	}
}

// ---- Case group D: RetargetHelixAgentMCPServers semantics ----

func TestHXC250Extend_RetargetPreservesNonHelixEntries(t *testing.T) {
	custom := []MCPServerConfig{
		{Name: "helixagent-mcp", Type: "remote", URL: "http://stale.old:1/v1/mcp"},
		{Name: "helixagent-vision", Type: "remote", URL: "http://stale.old:1/v1/vision?x=1"},
		{Name: "context7", Type: "remote", URL: "https://mcp.context7.com/mcp"},
		{Name: "filesystem", Type: "local", Command: []string{"npx", "-y", "srv"}},
		{Name: "consumer-own", Type: "remote", URL: "http://consumer.example:4321/thing"},
	}

	got := RetargetHelixAgentMCPServers(custom, "agent.internal", 7061)
	if len(got) != len(custom) {
		t.Fatalf("entry count changed: %d -> %d", len(custom), len(got))
	}

	byName := map[string]MCPServerConfig{}
	for _, s := range got {
		byName[s.Name] = s
	}

	if want := "http://agent.internal:7061/v1/mcp"; byName["helixagent-mcp"].URL != want {
		t.Errorf("helixagent-mcp = %q, want %q", byName["helixagent-mcp"].URL, want)
	}
	if want := "http://agent.internal:7061/v1/vision?x=1"; byName["helixagent-vision"].URL != want {
		t.Errorf("query not preserved: got %q, want %q", byName["helixagent-vision"].URL, want)
	}
	if want := "https://mcp.context7.com/mcp"; byName["context7"].URL != want {
		t.Errorf("third-party MCP was rewritten: got %q, want %q", byName["context7"].URL, want)
	}
	if want := "http://consumer.example:4321/thing"; byName["consumer-own"].URL != want {
		t.Errorf("consumer-owned MCP was rewritten: got %q, want %q", byName["consumer-own"].URL, want)
	}
	if got := byName["filesystem"]; got.Type != "local" || len(got.Command) != 3 {
		t.Errorf("local MCP entry was mutated: %+v", got)
	}

	// Input slice must not be mutated in place.
	if custom[0].URL != "http://stale.old:1/v1/mcp" {
		t.Errorf("caller's slice was mutated in place: %q", custom[0].URL)
	}
}

func TestHXC250Extend_RetargetEmptyYieldsDefaults(t *testing.T) {
	for _, in := range [][]MCPServerConfig{nil, {}} {
		got := RetargetHelixAgentMCPServers(in, "agent.internal", 7061)
		if len(got) != DefaultMCPServersCount() {
			t.Errorf("empty input yielded %d entries, want the default %d",
				len(got), DefaultMCPServersCount())
		}
		for _, u := range helixAgentURLs(got) {
			if !strings.HasPrefix(u, "http://agent.internal:7061/") {
				t.Errorf("default-filled entry %q does not target the injected endpoint", u)
			}
		}
	}
}

func TestHXC250Extend_RetargetUnparseableURLStillInjected(t *testing.T) {
	// A malformed stored URL must not be left pointing at a stale endpoint.
	in := []MCPServerConfig{{Name: "helixagent-cognee", Type: "remote", URL: "http://%zz"}}
	got := RetargetHelixAgentMCPServers(in, "agent.internal", 7061)
	if !strings.HasPrefix(got[0].URL, "http://agent.internal:7061/") {
		t.Errorf("unparseable URL not re-injected: got %q", got[0].URL)
	}
}

// ---- Case group E: crush provider config ----

func TestHXC250Extend_CrushDefaultConfigFollowsInjection(t *testing.T) {
	t.Run("env host+port", func(t *testing.T) {
		t.Setenv(helixendpoint.EnvBaseURL, "")
		t.Setenv(helixendpoint.EnvHost, "agent.internal")
		t.Setenv(helixendpoint.EnvPort, "7061")
		if got := crush_config.CreateDefaultConfig().Providers["helixagent"].BaseURL; got != "http://agent.internal:7061/v1" {
			t.Errorf("BaseURL = %q, want %q", got, "http://agent.internal:7061/v1")
		}
	})

	t.Run("env base URL with trailing slash", func(t *testing.T) {
		t.Setenv(helixendpoint.EnvBaseURL, "https://gw.example:8443/")
		if got := crush_config.CreateDefaultConfig().Providers["helixagent"].BaseURL; got != "https://gw.example:8443/v1" {
			t.Errorf("BaseURL = %q, want %q", got, "https://gw.example:8443/v1")
		}
	})

	t.Run("explicit override still wins", func(t *testing.T) {
		t.Setenv(helixendpoint.EnvHost, "agent.internal")
		t.Setenv(helixendpoint.EnvPort, "7061")
		cfg := crush_config.CreateHelixAgentConfig("", "http://explicit.example:1234/v1", nil, nil)
		if got := cfg.Providers["helixagent"].BaseURL; got != "http://explicit.example:1234/v1" {
			t.Errorf("explicit baseURL was overridden: got %q", got)
		}
	})
}

// ---- Case group F: generator config resolution ----

func TestHXC250Extend_DefaultGeneratorConfigFollowsEnv(t *testing.T) {
	t.Setenv(helixendpoint.EnvHost, "agent.internal")
	t.Setenv(helixendpoint.EnvPort, "7061")

	cfg := DefaultGeneratorConfig()
	if cfg.HelixAgentHost != "agent.internal" || cfg.HelixAgentPort != 7061 {
		t.Errorf("config endpoint = %s:%d, want agent.internal:7061", cfg.HelixAgentHost, cfg.HelixAgentPort)
	}
	for _, u := range helixAgentURLs(cfg.MCPServers) {
		if !strings.HasPrefix(u, "http://agent.internal:7061/") {
			t.Errorf("default MCP entry %q does not follow the injected endpoint", u)
		}
	}
}

func TestHXC250Extend_NewUnifiedGeneratorRetargets(t *testing.T) {
	cfg := DefaultGeneratorConfig()
	cfg.HelixAgentHost = "agent.internal"
	cfg.HelixAgentPort = 7061
	// MCPServers deliberately left as built by DefaultGeneratorConfig.

	ug := NewUnifiedGenerator(cfg)
	urls := helixAgentURLs(ug.config.MCPServers)
	if len(urls) == 0 {
		t.Fatal("instrument invalid: no HelixAgent MCP entries after retarget")
	}
	for _, u := range urls {
		if !strings.HasPrefix(u, "http://agent.internal:7061/") {
			t.Errorf("NewUnifiedGenerator did not retarget %q", u)
		}
	}
}

func TestHXC250Extend_NilConfigStillUsable(t *testing.T) {
	// Boundary: NewUnifiedGenerator(nil) falls back to DefaultGeneratorConfig.
	ug := NewUnifiedGenerator(nil)
	if len(ug.config.MCPServers) == 0 {
		t.Fatal("nil config produced no MCP servers — capability lost")
	}
	if len(ug.ListSupportedAgents()) == 0 {
		t.Fatal("nil config produced no generators")
	}
}

// ---- Case group G: review-round-2 remediation (§11.4.134 iterate-to-GO) ----
// Cases added after the independent review flagged them; each asserts the
// NEW behaviour so the fix cannot silently regress.

func TestHXC250Extend_RetargetPreservesFragment(t *testing.T) {
	in := []MCPServerConfig{
		{Name: "helixagent-mcp", Type: "remote", URL: "http://stale:1/v1/mcp#section"},
		{Name: "helixagent-acp", Type: "remote", URL: "http://stale:1/v1/acp?q=1#frag"},
	}
	got := RetargetHelixAgentMCPServers(in, "agent.internal", 7061)
	if want := "http://agent.internal:7061/v1/mcp#section"; got[0].URL != want {
		t.Errorf("fragment dropped: got %q, want %q", got[0].URL, want)
	}
	if want := "http://agent.internal:7061/v1/acp?q=1#frag"; got[1].URL != want {
		t.Errorf("query+fragment not preserved: got %q, want %q", got[1].URL, want)
	}
}

func TestHXC250Extend_RetargetSchemelessAndOpaqueRebuilt(t *testing.T) {
	// A scheme-less or opaque URL parses without error but has no recoverable
	// host/path; it must be rebuilt from the entry name, not silently reduced
	// to a bare base URL with the path dropped.
	cases := []struct{ name, url, want string }{
		{"helixagent-mcp", "localhost:8100/v1/mcp", "http://agent.internal:7061/v1/mcp"},
		{"helixagent-vision", "mailto:someone@example.com", "http://agent.internal:7061/v1/vision"},
		{"helixagent-lsp", "/v1/lsp", "http://agent.internal:7061/v1/lsp"},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/"+tc.url, func(t *testing.T) {
			got := RetargetHelixAgentMCPServers(
				[]MCPServerConfig{{Name: tc.name, Type: "remote", URL: tc.url}},
				"agent.internal", 7061)
			if got[0].URL != tc.want {
				t.Errorf("got %q, want %q", got[0].URL, tc.want)
			}
		})
	}
}

func TestHXC250Extend_ContainerizedCountIsHostIndependent(t *testing.T) {
	t.Setenv(helixendpoint.EnvHost, "somewhere.else")
	t.Setenv(helixendpoint.EnvPort, "7061")
	if got, want := ContainerizedMCPServersCount(), len(ContainerizedMCPServers("other.host")); got != want {
		t.Errorf("count is host-dependent: %d vs %d", got, want)
	}
}

// ---- Case group H: the BASE_URL-vs-HOST/PORT split (review finding F1) ----
//
// README.md ("Which variable reaches which generated artifact") makes a
// load-bearing operator promise: HELIX_AGENT_BASE_URL reaches the Crush provider
// base_url and does NOT reach the pkg/cliagents surfaces, which resolve through
// helixendpoint.Host()+Port(). Nothing pinned that promise, so the split could
// drift in EITHER direction with no test failing and the documentation rotting
// silently (§11.4.226 — prose does not bind, seams do).
//
// Measured on the real generators with BASE_URL as the only injection:
//
//	emitted=50  carrying-placeholder-8100=49  carrying-injected-BASE_URL=1
//
// i.e. an operator who sets only BASE_URL ships 49 of 50 artifacts pointing at
// the placeholder — byte-identical to the defect HXC-250 exists to fix.
//
// The split itself is DEFENSIBLE and is what this test pins, not a defect:
// cliagents synthesises URLs across many sibling ports, so one opaque BASE_URL
// (scheme + path + a single port) cannot be decomposed onto them without
// inventing semantics, while Crush is a single-URL surface where a verbatim
// BASE_URL is complete.
//
// BOTH directions are asserted, because either drift is a defect: cliagents
// starting to honour EnvBaseURL would make the README wrong, and Crush ceasing
// to honour it would strand the one artifact that does follow it.
func TestHXC250Extend_BaseURLReachesCrushButNotCLIAgents(t *testing.T) {
	const injectedBase = "https://gw.injected.invalid:8443"
	const injectedHost = "gw.injected.invalid"

	// ONLY the base URL is injected — the exact operator mistake under test.
	t.Setenv(helixendpoint.EnvBaseURL, injectedBase)
	t.Setenv(helixendpoint.EnvHost, "")
	t.Setenv(helixendpoint.EnvPort, "")

	placeholder := helixendpoint.BaseURL(helixendpoint.DefaultHost, helixendpoint.DefaultPort)

	// Direction 1 — Crush DOES follow the injected base URL.
	t.Run("crush provider follows BASE_URL", func(t *testing.T) {
		got := crush_config.CreateDefaultConfig().Providers["helixagent"].BaseURL
		if want := injectedBase + "/v1"; got != want {
			t.Errorf("crush base_url = %q, want %q — %s no longer reaches the Crush provider, "+
				"contradicting README \"Which variable reaches which generated artifact\"",
				got, want, helixendpoint.EnvBaseURL)
		}
	})

	// Direction 2 — cliagents does NOT: it stays on the documented placeholder.
	t.Run("cliagents MCP URLs stay on the placeholder", func(t *testing.T) {
		urls := helixAgentURLs(DefaultMCPServers())
		if len(urls) == 0 {
			t.Fatal("instrument invalid: no HelixAgent MCP entries produced, so any verdict is vacuous")
		}
		for _, u := range urls {
			if strings.Contains(u, injectedHost) {
				t.Errorf("MCP URL %q followed %s — pkg/cliagents now honours it, contradicting the "+
					"README table that documents it as HOST/PORT-only", u, helixendpoint.EnvBaseURL)
			}
			if !strings.HasPrefix(u, placeholder+"/") {
				t.Errorf("MCP URL %q does not resolve to the documented placeholder %q", u, placeholder)
			}
		}
	})

	t.Run("generator config endpoint stays on the placeholder", func(t *testing.T) {
		cfg := DefaultGeneratorConfig()
		if cfg.HelixAgentHost != helixendpoint.DefaultHost {
			t.Errorf("HelixAgentHost = %q, want the placeholder %q — %s must not reach this surface",
				cfg.HelixAgentHost, helixendpoint.DefaultHost, helixendpoint.EnvBaseURL)
		}
		if cfg.HelixAgentPort != helixendpoint.DefaultPort {
			t.Errorf("HelixAgentPort = %d, want the placeholder %d",
				cfg.HelixAgentPort, helixendpoint.DefaultPort)
		}
	})

	// The pairing the README prescribes as the fix must reach BOTH surfaces —
	// otherwise the advice the split forces on operators is itself unpinned.
	t.Run("setting HOST+PORT alongside reaches both surfaces", func(t *testing.T) {
		t.Setenv(helixendpoint.EnvHost, injectedHost)
		t.Setenv(helixendpoint.EnvPort, "8443")

		if got := crush_config.CreateDefaultConfig().Providers["helixagent"].BaseURL; got != injectedBase+"/v1" {
			t.Errorf("crush base_url = %q, want %q", got, injectedBase+"/v1")
		}
		urls := helixAgentURLs(DefaultMCPServers())
		if len(urls) == 0 {
			t.Fatal("instrument invalid: no HelixAgent MCP entries produced")
		}
		for _, u := range urls {
			if !strings.HasPrefix(u, "http://"+injectedHost+":8443/") {
				t.Errorf("MCP URL %q did not follow the HOST/PORT injection", u)
			}
		}
	})
}
