// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingRunner captures the exact argv handed to the runner by the
// real production code path.
type recordingRunner struct {
	calls [][]string
}

func (r *recordingRunner) Run(name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return []byte("NAME ID SIZE MODIFIED\nllava:7b abc 4.5 GB 1 day ago\n"), nil
}

const okList = "NAME ID SIZE MODIFIED\nllava:7b abc 4.5 GB 1 day ago\n"

func TestProbe_ExactArgv_StrictHostKeyChecking(t *testing.T) {
	r := &recordingRunner{}
	_, err := ProbeLocalModelsWithRunner([]string{"thinker.local"}, "milos", r)
	require.NoError(t, err)
	require.Len(t, r.calls, 1)
	assert.Equal(t, []string{
		"ssh", "-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=yes",
		"--", "milos@thinker.local", "ollama", "list",
	}, r.calls[0])
}

func TestProbe_NeverDisablesHostKeyChecking(t *testing.T) {
	r := &recordingRunner{}
	_, err := ProbeLocalModelsWithRunner([]string{"a.local", "10.0.0.5", "::1"}, "", r)
	require.NoError(t, err)
	require.Len(t, r.calls, 3)
	for _, c := range r.calls {
		joined := strings.Join(c, " ")
		assert.NotContains(t, joined, "StrictHostKeyChecking=no")
		assert.NotContains(t, joined, "accept-new")
		assert.NotContains(t, joined, "UserKnownHostsFile=/dev/null")
	}
}

func TestProbe_DoubleDashPrecedesTarget(t *testing.T) {
	r := &recordingRunner{}
	_, err := ProbeLocalModelsWithRunner([]string{"thinker.local"}, "", r)
	require.NoError(t, err)
	args := r.calls[0]
	dd := indexOf(args, "--")
	require.GreaterOrEqual(t, dd, 0, "argv must contain --")
	assert.Equal(t, "thinker.local", args[dd+1])
	assert.Equal(t, []string{"ollama", "list"}, args[dd+2:])
}

func TestProbe_RejectsOptionInjectionHosts(t *testing.T) {
	bad := []string{
		"-oProxyCommand=touch /tmp/pwn",
		"-o", "-F/etc/passwd", "--", "-",
		"host name", "host;rm", "host$(id)", "a@b", "host\nx", "",
		"host`id`", "ho st", "host/../x", ".leading", "trailing.", "-leading",
		strings.Repeat("a", 254),
	}
	for _, h := range bad {
		r := &recordingRunner{}
		_, err := ProbeLocalModelsWithRunner([]string{h}, "", r)
		assert.Error(t, err, "host %q must be rejected", h)
		assert.Empty(t, r.calls, "runner must not be invoked for host %q", h)
	}
}

func TestProbe_RejectsBadUsers(t *testing.T) {
	bad := []string{"-oProxyCommand=x", "-l", "a b", "a@b", "x;y", "u\n", "$(id)", "-", strings.Repeat("u", 33)}
	for _, u := range bad {
		r := &recordingRunner{}
		_, err := ProbeLocalModelsWithRunner([]string{"thinker.local"}, u, r)
		assert.Error(t, err, "user %q must be rejected", u)
		assert.Empty(t, r.calls, "runner must not be invoked for user %q", u)
	}
}

func TestProbe_AcceptsLegitimateTargets(t *testing.T) {
	for _, h := range []string{"localhost", "thinker.local", "amber-1.lan", "192.168.1.10", "::1", "2001:db8::1", "A1"} {
		r := &recordingRunner{}
		_, err := ProbeLocalModelsWithRunner([]string{h}, "svc_user-1.x", r)
		assert.NoError(t, err, "host %q must be accepted", h)
		assert.Len(t, r.calls, 1)
	}
}

func TestProbe_OneBadHostDoesNotBlockGoodOnes(t *testing.T) {
	r := &recordingRunner{}
	models, err := ProbeLocalModelsWithRunner([]string{"-oProxyCommand=x", "good.local"}, "", r)
	require.NoError(t, err)
	assert.Len(t, r.calls, 1)
	assert.Len(t, models, 1)
	assert.Equal(t, "good.local", models[0].Host)
}

func TestProbeWithOptions_KnownHostsFile(t *testing.T) {
	r := &recordingRunner{}
	_, err := ProbeLocalModelsWithOptions([]string{"thinker.local"}, "milos",
		ProbeOptions{KnownHostsFile: "/etc/llmsverifier/known_hosts"}, r)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"ssh", "-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=/etc/llmsverifier/known_hosts",
		"--", "milos@thinker.local", "ollama", "list",
	}, r.calls[0])
}

func TestProbeWithOptions_EmptyOptionsMatchDefaultArgv(t *testing.T) {
	r1, r2 := &recordingRunner{}, &recordingRunner{}
	_, _ = ProbeLocalModelsWithRunner([]string{"h.local"}, "u", r1)
	_, _ = ProbeLocalModelsWithOptions([]string{"h.local"}, "u", ProbeOptions{}, r2)
	assert.Equal(t, r1.calls, r2.calls)
}

func TestProbeWithOptions_RejectsBadKnownHostsFile(t *testing.T) {
	for _, p := range []string{"-oProxyCommand=x", "relative/known_hosts", "/a b/kh", "/a\nb", "/dev/null", "/a,b", "/a=b"} {
		r := &recordingRunner{}
		_, err := ProbeLocalModelsWithOptions([]string{"h.local"}, "", ProbeOptions{KnownHostsFile: p}, r)
		assert.Error(t, err, "known_hosts %q must be rejected", p)
		assert.Empty(t, r.calls)
	}
}

func TestKnownHostsFileFromEnv(t *testing.T) {
	t.Setenv(KnownHostsEnv, "/srv/kh")
	assert.Equal(t, ProbeOptions{KnownHostsFile: "/srv/kh"}, ProbeOptionsFromEnv())
	t.Setenv(KnownHostsEnv, "")
	assert.Equal(t, ProbeOptions{}, ProbeOptionsFromEnv())
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}
