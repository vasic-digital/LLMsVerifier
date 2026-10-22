// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRunner implements CommandRunner for testing.
type mockRunner struct {
	output map[string][]byte
	err    map[string]error
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		output: make(map[string][]byte),
		err:    make(map[string]error),
	}
}

func (m *mockRunner) SetOutput(host string, out []byte, err error) {
	m.output[host] = out
	m.err[host] = err
}

func (m *mockRunner) Run(name string, args ...string) ([]byte, error) {
	// Extract host from SSH args — it's the second-to-last arg
	// (last arg is the command).
	for i, a := range args {
		if a == "-o" || a == "ssh" {
			continue
		}
		// The target host is the first non-flag arg after SSH options.
		if !hasPrefix(a, "-") && i > 0 {
			if out, ok := m.output[a]; ok {
				return out, m.err[a]
			}
		}
	}
	// Try matching on last-but-one arg (the user@host part).
	if len(args) >= 2 {
		target := args[len(args)-2]
		if out, ok := m.output[target]; ok {
			return out, m.err[target]
		}
	}
	return nil, fmt.Errorf("mock: no output configured")
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// --- parseOllamaList tests ---

func TestParseOllamaList_StandardOutput(t *testing.T) {
	output := `NAME                ID              SIZE      MODIFIED
llava:7b            abc123def456    4.5 GB    3 days ago
llama3.1:8b         def789abc012    4.7 GB    1 day ago
minicpm-v:8b        ghi345jkl678    5.1 GB    2 hours ago
`
	models, err := parseOllamaList(output, "thinker.local")
	require.NoError(t, err)
	assert.Len(t, models, 3)

	// llava:7b should be vision-capable.
	assert.Equal(t, "llava:7b", models[0].Name)
	assert.Equal(t, "ollama", models[0].Provider)
	assert.Equal(t, "thinker.local", models[0].Host)
	assert.True(t, models[0].SupportsVision)
	assert.Greater(t, models[0].RAMRequired, int64(0))

	// llama3.1:8b should not be vision-capable.
	assert.Equal(t, "llama3.1:8b", models[1].Name)
	assert.False(t, models[1].SupportsVision)

	// minicpm-v:8b should be vision-capable.
	assert.Equal(t, "minicpm-v:8b", models[2].Name)
	assert.True(t, models[2].SupportsVision)
}

func TestParseOllamaList_EmptyOutput(t *testing.T) {
	output := `NAME                ID              SIZE      MODIFIED
`
	models, err := parseOllamaList(output, "localhost")
	require.NoError(t, err)
	assert.Empty(t, models)
}

func TestParseOllamaList_SingleModel(t *testing.T) {
	output := `NAME                ID              SIZE      MODIFIED
qwen2-vl:7b         xyz123          3.8 GB    1 hour ago
`
	models, err := parseOllamaList(output, "amber.local")
	require.NoError(t, err)
	assert.Len(t, models, 1)
	assert.Equal(t, "qwen2-vl:7b", models[0].Name)
	assert.True(t, models[0].SupportsVision)
	assert.Equal(t, "amber.local", models[0].Host)
}

// --- isVisionModel tests ---

func TestIsVisionModel(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"llava:7b", true},
		{"llava:13b", true},
		{"LLaVA-Next:7b", true},
		{"minicpm-v:8b", true},
		{"MiniCPM-V-2.6:8b", true},
		{"qwen2-vl:7b", true},
		{"bakllava:7b", true},
		{"cogvlm2:19b", true},
		{"internvl:25b", true},
		{"moondream:1.8b", true},
		{"llama3.1:8b", false},
		{"mistral:7b", false},
		{"codellama:13b", false},
		{"phi3:3.8b", false},
		{"gemma:2b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isVisionModel(tt.name),
				"isVisionModel(%q)", tt.name)
		})
	}
}

// --- parseSizeBytes tests ---

func TestParseSizeBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"4.5 GB", 4831838208},
		{"8.0 GB", 8589934592},
		{"512 MB", 536870912},
		{"100 KB", 102400},
		{"", 0},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSizeBytes(tt.input)
			assert.Equal(t, tt.expected, result,
				"parseSizeBytes(%q)", tt.input)
		})
	}
}

// --- parseParameterCount tests ---

func TestParseParameterCount(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"llava:7b", 7_000_000_000},
		{"llama3.1:8b", 8_000_000_000},
		{"minicpm-v:8b", 8_000_000_000},
		{"llava:13b", 13_000_000_000},
		{"llama:70b", 70_000_000_000},
		{"phi3:3.8b", 3_800_000_000},
		{"moondream:1.8b", 1_800_000_000},
		{"mistral:latest", 0}, // no size in name
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseParameterCount(tt.input)
			assert.Equal(t, tt.expected, result,
				"parseParameterCount(%q)", tt.input)
		})
	}
}

// --- extractQuantLevel tests ---

func TestExtractQuantLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"model-q4_k_m", "Q4_K_M"},
		{"model-Q8_0", "Q8_0"},
		{"model-f16", "F16"},
		{"model-Q4_K", "Q4_K"},
		{"model-q4", "Q4"},
		{"model:latest", ""},
		{"llava:7b", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractQuantLevel(tt.input)
			assert.Equal(t, tt.expected, result,
				"extractQuantLevel(%q)", tt.input)
		})
	}
}

// --- ScoreLocalModel tests ---

func TestScoreLocalModel_VisionModel(t *testing.T) {
	model := LocalModelInfo{
		Name:           "llava:7b",
		Provider:       "ollama",
		Host:           "thinker.local",
		SupportsVision: true,
		ParameterCount: 7_000_000_000,
		RAMRequired:    4_500_000_000,
		GPURequired:    4_500_000_000,
	}

	score := ScoreLocalModel(model)
	assert.Greater(t, score, 0.0, "score should be positive")
	assert.LessOrEqual(t, score, 1.0, "score should be <= 1")
}

func TestScoreLocalModel_TextOnlyModel(t *testing.T) {
	model := LocalModelInfo{
		Name:           "llama3.1:8b",
		Provider:       "ollama",
		Host:           "localhost",
		SupportsVision: false,
		ParameterCount: 8_000_000_000,
		RAMRequired:    5_000_000_000,
	}

	score := ScoreLocalModel(model)
	assert.Greater(t, score, 0.0, "score should be positive")
	assert.LessOrEqual(t, score, 1.0, "score should be <= 1")
}

func TestScoreLocalModel_VisionBetterThanText(t *testing.T) {
	vision := LocalModelInfo{
		Name:           "llava:7b",
		Provider:       "ollama",
		Host:           "thinker.local",
		SupportsVision: true,
		ParameterCount: 7_000_000_000,
	}

	text := LocalModelInfo{
		Name:           "llama:7b",
		Provider:       "ollama",
		Host:           "thinker.local",
		SupportsVision: false,
		ParameterCount: 7_000_000_000,
	}

	visionScore := ScoreLocalModel(vision)
	textScore := ScoreLocalModel(text)
	assert.Greater(t, visionScore, textScore,
		"vision model should score higher than text-only at same size")
}

func TestScoreLocalModel_LargerModelBetter(t *testing.T) {
	small := LocalModelInfo{
		Name:           "llava:7b",
		Provider:       "ollama",
		Host:           "thinker.local",
		SupportsVision: true,
		ParameterCount: 7_000_000_000,
	}

	large := LocalModelInfo{
		Name:           "llava:13b",
		Provider:       "ollama",
		Host:           "thinker.local",
		SupportsVision: true,
		ParameterCount: 13_000_000_000,
	}

	smallScore := ScoreLocalModel(small)
	largeScore := ScoreLocalModel(large)
	assert.Greater(t, largeScore, smallScore,
		"larger model should score higher")
}

// --- estimateQuality tests ---

func TestEstimateQuality(t *testing.T) {
	tests := []struct {
		params   int64
		minScore float64
		maxScore float64
	}{
		{0, 0.49, 0.51},
		{1_000_000_000, 0.49, 0.51},
		{3_000_000_000, 0.54, 0.56},
		{7_000_000_000, 0.59, 0.61},
		{8_000_000_000, 0.64, 0.66},
		{13_000_000_000, 0.69, 0.71},
		{30_000_000_000, 0.74, 0.76},
		{70_000_000_000, 0.81, 0.83},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%dB", tt.params/1_000_000_000), func(t *testing.T) {
			score := estimateQuality(tt.params)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

// --- ToModelInfo tests ---

func TestToModelInfo(t *testing.T) {
	local := LocalModelInfo{
		Name:           "llava:13b",
		Provider:       "ollama",
		Host:           "thinker.local",
		SupportsVision: true,
		ParameterCount: 13_000_000_000,
		QuantLevel:     "Q4_K_M",
		RAMRequired:    8_000_000_000,
		GPURequired:    8_000_000_000,
	}

	info := ToModelInfo(local)

	assert.Equal(t, "local-thinker.local-llava:13b", info.ID)
	assert.Contains(t, info.Name, "llava:13b")
	assert.Contains(t, info.Name, "thinker.local")
	assert.Equal(t, "ollama", info.Provider)
	assert.Equal(t, "llava:13b", info.Model)
	assert.True(t, info.SupportsVision)
	assert.True(t, info.SupportsStreaming)
	assert.Equal(t, 0.0, info.InputCostPer1k)
	assert.Equal(t, 0.0, info.OutputCostPer1k)
	assert.Greater(t, info.QualityScore, 0.0)
	assert.Greater(t, info.ReliabilityScore, 0.0)
	assert.Greater(t, info.ContextWindow, 0)
	assert.Greater(t, info.MaxOutputTokens, 0)
	assert.Contains(t, info.Capabilities, "vision")
	assert.Contains(t, info.Capabilities, "local")
	assert.Contains(t, info.Capabilities, "text")

	// Check metadata.
	assert.Equal(t, "thinker.local", info.Metadata["host"])
	assert.Equal(t, int64(13_000_000_000), info.Metadata["parameter_count"])
	assert.Equal(t, "Q4_K_M", info.Metadata["quant_level"])
	assert.Equal(t, true, info.Metadata["is_local"])
}

func TestToModelInfo_TextOnly(t *testing.T) {
	local := LocalModelInfo{
		Name:           "llama3.1:8b",
		Provider:       "ollama",
		Host:           "localhost",
		SupportsVision: false,
		ParameterCount: 8_000_000_000,
	}

	info := ToModelInfo(local)

	assert.False(t, info.SupportsVision)
	assert.NotContains(t, info.Capabilities, "vision")
	assert.Contains(t, info.Capabilities, "local")
	assert.Contains(t, info.Capabilities, "text")
}

// --- SortByScore tests ---

func TestSortByScore(t *testing.T) {
	models := []LocalModelInfo{
		{Name: "small:3b", ParameterCount: 3_000_000_000},
		{Name: "llava:13b", ParameterCount: 13_000_000_000, SupportsVision: true},
		{Name: "medium:8b", ParameterCount: 8_000_000_000},
	}

	SortByScore(models)

	// Vision 13b should be first due to vision bonus + larger size.
	assert.Equal(t, "llava:13b", models[0].Name)
	// 8b should be before 3b.
	assert.Equal(t, "medium:8b", models[1].Name)
	assert.Equal(t, "small:3b", models[2].Name)
}

// --- ProbeLocalModelsWithRunner tests ---

func TestProbeLocalModelsWithRunner_Success(t *testing.T) {
	runner := newMockRunner()
	runner.SetOutput("user@thinker.local", []byte(`NAME                ID              SIZE      MODIFIED
llava:7b            abc123def456    4.5 GB    3 days ago
llama3.1:8b         def789abc012    4.7 GB    1 day ago
`), nil)

	models, err := ProbeLocalModelsWithRunner(
		[]string{"thinker.local"}, "user", runner,
	)
	require.NoError(t, err)
	assert.Len(t, models, 2)
	assert.Equal(t, "llava:7b", models[0].Name)
	assert.Equal(t, "thinker.local", models[0].Host)
}

func TestProbeLocalModelsWithRunner_HostUnreachable(t *testing.T) {
	runner := newMockRunner()
	runner.SetOutput("user@unreachable.local", nil,
		fmt.Errorf("ssh: connect to host unreachable.local: No route to host"))

	_, err := ProbeLocalModelsWithRunner(
		[]string{"unreachable.local"}, "user", runner,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no models discovered")
}

func TestProbeLocalModelsWithRunner_NoHosts(t *testing.T) {
	runner := newMockRunner()

	_, err := ProbeLocalModelsWithRunner(
		[]string{}, "user", runner,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no hosts provided")
}

func TestProbeLocalModelsWithRunner_MultipleHosts(t *testing.T) {
	runner := newMockRunner()
	runner.SetOutput("user@host1.local", []byte(`NAME                ID              SIZE      MODIFIED
llava:7b            abc123          4.5 GB    3 days ago
`), nil)
	runner.SetOutput("user@host2.local", []byte(`NAME                ID              SIZE      MODIFIED
minicpm-v:8b        def456          5.1 GB    1 day ago
`), nil)

	models, err := ProbeLocalModelsWithRunner(
		[]string{"host1.local", "host2.local"}, "user", runner,
	)
	require.NoError(t, err)
	assert.Len(t, models, 2)

	hosts := map[string]bool{}
	for _, m := range models {
		hosts[m.Host] = true
	}
	assert.True(t, hosts["host1.local"])
	assert.True(t, hosts["host2.local"])
}

func TestProbeLocalModelsWithRunner_PartialFailure(t *testing.T) {
	runner := newMockRunner()
	runner.SetOutput("user@good.local", []byte(`NAME                ID              SIZE      MODIFIED
llava:7b            abc123          4.5 GB    3 days ago
`), nil)
	runner.SetOutput("user@bad.local", nil,
		fmt.Errorf("connection refused"))

	models, err := ProbeLocalModelsWithRunner(
		[]string{"good.local", "bad.local"}, "user", runner,
	)
	require.NoError(t, err)
	assert.Len(t, models, 1)
	assert.Equal(t, "good.local", models[0].Host)
}
