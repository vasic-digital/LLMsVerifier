package cliagents

import (
	"encoding/json"
	"strings"
	"testing"
)

// hxc269SyntheticSecret is a fabricated fixture, not a credential of any real
// system. It exists only so a leak is detectable: if it appears in a generated
// artifact, a real secret in the same position would have appeared too.
const hxc269SyntheticSecret = "s3cr3t-synthetic-not-a-real-credential"

// TestHXC269NoDisplacementEndToEnd is the end-to-end half of the HXC-269 guard.
//
// The unit guard (pkg/helixendpoint TestHXC269PortDisplacementRED) proves the
// address BUILDER is correct. This one proves the property the item actually
// describes — "the value flows end to end from the operator-facing host setting
// into generated configuration with nothing reporting a problem" — by driving the
// real operator-facing input (the HELIX_AGENT_HOST environment variable) through
// the real generator entry points and inspecting what they emit.
//
// It is a distinct layer, not a duplicate: a regression that bypassed
// helixendpoint in DefaultGeneratorConfig — reading the environment directly, or
// re-introducing a fmt.Sprintf on the raw value — would leave the unit guard GREEN
// and be caught only here.
//
// # What the pre-fix artifact did (measured)
//
// With HELIX_AGENT_HOST="gw.example?token=<secret>" the generator emitted, for
// each of six MCP entries:
//
//	http://gw.example?token=<secret>:8100/v1/mcp
//
// Two failures at once. The port is not on the authority — a parser reads host
// "gw.example", NO port, and query "token=<secret>:8100" — so the client silently
// used the default port. And the secret was written VERBATIM into every generated
// artifact on disk, with no diagnostic anywhere, because normalizeHost ACCEPTED
// the shape and therefore nothing reported a problem.
//
// # What is asserted
//
// For each of the four displacing shapes: the injected value reaches NEITHER the
// resolved host NOR any generated URL, and the injected PORT still arrives intact.
// That second assertion is the usefulness half — a "fix" that discarded the whole
// endpoint would satisfy the no-leak check while breaking every generated config.
func TestHXC269NoDisplacementEndToEnd(t *testing.T) {
	const injectedPort = 7061

	// Each shape carries the secret in the position an operator would actually
	// paste it, one per displacing delimiter.
	shapes := []struct {
		name string
		host string
	}{
		{"query-carried token", "gw.example?token=" + hxc269SyntheticSecret},
		{"fragment-carried token", "gw.example#access_token=" + hxc269SyntheticSecret},
		{"path-carried token", "gw.example/" + hxc269SyntheticSecret},
		{"token as userinfo", hxc269SyntheticSecret + "@gw.example"},
	}

	for _, tc := range shapes {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HELIX_AGENT_BASE_URL", "")
			t.Setenv("HELIX_AGENT_PORT", "7061")
			t.Setenv("HELIX_AGENT_HOST", tc.host)

			cfg := DefaultGeneratorConfig()

			servers, err := json.Marshal(DefaultMCPServers())
			if err != nil {
				t.Fatalf("instrument invalid: could not marshal generated servers: %v", err)
			}

			if strings.Contains(cfg.HelixAgentHost, hxc269SyntheticSecret) {
				t.Errorf("resolved host still carries the injected value: %q", cfg.HelixAgentHost)
			}
			if strings.Contains(string(servers), hxc269SyntheticSecret) {
				t.Errorf("injected value reached a generated MCP server URL: %s", servers)
			}

			// Usefulness: the endpoint must still be configured, not discarded.
			// A rejected HOST must not take the valid PORT down with it — the two
			// fall back independently, and a generator that emitted no port would
			// point every client at the wrong place just as surely.
			if cfg.HelixAgentPort != injectedPort {
				t.Errorf("injected port did not survive host rejection: want %d, got %d",
					injectedPort, cfg.HelixAgentPort)
			}
			if cfg.HelixAgentHost == "" {
				t.Error("resolved host is empty; generated configs would carry no host at all")
			}
			if !strings.Contains(string(servers), "/v1/mcp") {
				t.Errorf("generated servers lost their endpoints entirely: %s", servers)
			}
		})
	}
}
