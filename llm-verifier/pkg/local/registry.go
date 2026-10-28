// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package local provides discovery and scoring for locally hosted
// LLM models running on Ollama or llama.cpp instances. It probes
// remote hosts via SSH and parses their model inventories so that
// the unified selector can rank local models alongside cloud
// providers using identical scoring dimensions.
package local

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
)

// LocalModelInfo describes a model available via Ollama or llama.cpp
// on a local or remote host.
type LocalModelInfo struct {
	// Name is the model identifier (e.g. "minicpm-v:8b", "llava:13b").
	Name string `json:"name"`

	// Provider is the inference backend: "ollama" or "llamacpp".
	Provider string `json:"provider"`

	// Host is the hostname or IP where the model is available
	// (e.g. "thinker.local", "amber.local", "localhost").
	Host string `json:"host"`

	// SupportsVision indicates whether the model can process
	// images (e.g. LLaVA, MiniCPM-V, Qwen-VL).
	SupportsVision bool `json:"supports_vision"`

	// ParameterCount is the total parameter count in the model
	// (e.g. 8_000_000_000 for an 8B model).
	ParameterCount int64 `json:"parameter_count"`

	// QuantLevel is the quantization level string
	// (e.g. "Q4_K_M", "Q8_0", "F16").
	QuantLevel string `json:"quant_level"`

	// RAMRequired is the estimated RAM in bytes needed to run
	// the model.
	RAMRequired int64 `json:"ram_required"`

	// GPURequired is the estimated VRAM in bytes needed. Zero
	// means CPU-only inference is sufficient.
	GPURequired int64 `json:"gpu_required"`
}

// visionModelPatterns matches known vision-capable model families.
var visionModelPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)llava`),
	regexp.MustCompile(`(?i)minicpm.*v`),
	regexp.MustCompile(`(?i)qwen.*vl`),
	regexp.MustCompile(`(?i)bakllava`),
	regexp.MustCompile(`(?i)cogvlm`),
	regexp.MustCompile(`(?i)internvl`),
	regexp.MustCompile(`(?i)phi.*vision`),
	regexp.MustCompile(`(?i)moondream`),
}

// isVisionModel checks if a model name matches any known vision
// model family.
func isVisionModel(name string) bool {
	for _, pat := range visionModelPatterns {
		if pat.MatchString(name) {
			return true
		}
	}
	return false
}

// sizePattern matches Ollama model size strings like "4.1 GB"
// or "8.0 GB".
var sizePattern = regexp.MustCompile(`([\d.]+)\s*(GB|MB|KB)`)

// parseSizeBytes parses a human-readable size string into bytes.
func parseSizeBytes(s string) int64 {
	m := sizePattern.FindStringSubmatch(s)
	if len(m) < 3 {
		return 0
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(m[2]) {
	case "GB":
		return int64(val * 1024 * 1024 * 1024)
	case "MB":
		return int64(val * 1024 * 1024)
	case "KB":
		return int64(val * 1024)
	}
	return 0
}

// paramPattern matches parameter count strings like "8B", "70B",
// "1.5B".
var paramPattern = regexp.MustCompile(`([\d.]+)\s*[Bb]`)

// parseParameterCount extracts parameter count from a model name
// or tag (e.g. "llava:7b" -> 7_000_000_000).
func parseParameterCount(name string) int64 {
	m := paramPattern.FindStringSubmatch(name)
	if len(m) < 2 {
		return 0
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	return int64(val * 1_000_000_000)
}

// ProbeResult contains the outcome of probing a single host.
type ProbeResult struct {
	Host   string
	Models []LocalModelInfo
	Error  error
}

// CommandRunner abstracts command execution for testing.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// defaultRunner uses os/exec.
type defaultRunner struct{}

func (d *defaultRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// DefaultRunner returns the standard command runner.
func DefaultRunner() CommandRunner {
	return &defaultRunner{}
}

// KnownHostsEnv names the optional environment variable that supplies
// a managed known_hosts file for SSH probing (see ProbeOptionsFromEnv).
const KnownHostsEnv = "LLMSVERIFIER_SSH_KNOWN_HOSTS"

// ProbeOptions tunes SSH probing. The zero value uses the invoking
// user's default known_hosts files. Host key verification is ALWAYS
// enforced (StrictHostKeyChecking=yes); there is no option to disable it.
type ProbeOptions struct {
	// KnownHostsFile, when non-empty, is passed to ssh as
	// UserKnownHostsFile. It must be an absolute path made of
	// [A-Za-z0-9._/-] only, without ".." or a trailing "/", and not under
	// /dev, /proc or /sys.
	KnownHostsFile string
}

// ProbeOptionsFromEnv builds ProbeOptions from LLMSVERIFIER_SSH_KNOWN_HOSTS.
func ProbeOptionsFromEnv() ProbeOptions {
	return ProbeOptions{KnownHostsFile: os.Getenv(KnownHostsEnv)}
}

var (
	hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]{0,251}[A-Za-z0-9])?$`)
	userPattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9._-]{0,31}$`)
	knownHostsChars = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
)

// validateHost accepts a DNS hostname, IPv4 or IPv6 literal. Anything that
// could be parsed by ssh as an option or a user@host/port/URI form
// (leading '-', '@', whitespace, shell or control characters) is rejected.
func validateHost(host string) error {
	if host == "" || len(host) > 253 {
		return fmt.Errorf("local: invalid host %q", host)
	}
	if hostnamePattern.MatchString(host) && !strings.Contains(host, "..") {
		return nil
	}
	if net.ParseIP(host) != nil {
		return nil
	}
	return fmt.Errorf("local: invalid host %q", host)
}

// validateUser accepts a POSIX-style login name.
func validateUser(user string) error {
	if user != "" && !userPattern.MatchString(user) {
		return fmt.Errorf("local: invalid ssh user %q", user)
	}
	return nil
}

func validateKnownHostsFile(p string) error {
	if p == "" {
		return nil
	}
	if !filepath.IsAbs(p) || !knownHostsChars.MatchString(p) ||
		strings.Contains(p, "..") || strings.HasSuffix(p, "/") ||
		strings.HasPrefix(filepath.Clean(p), "/dev/") || strings.HasPrefix(filepath.Clean(p), "/proc/") ||
		strings.HasPrefix(filepath.Clean(p), "/sys/") {
		return fmt.Errorf("local: invalid known_hosts file %q", p)
	}
	return nil
}

// buildSSHArgs returns the argv (after the "ssh" program name) for probing
// one host. Host key checking is always strict and the target follows "--"
// so it can never be interpreted as an ssh option.
func buildSSHArgs(host, user string, opts ProbeOptions) ([]string, error) {
	if err := validateHost(host); err != nil {
		return nil, err
	}
	if err := validateUser(user); err != nil {
		return nil, err
	}
	if err := validateKnownHostsFile(opts.KnownHostsFile); err != nil {
		return nil, err
	}
	target := host
	if user != "" {
		target = user + "@" + host
	}
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=yes",
	}
	if opts.KnownHostsFile != "" {
		args = append(args, "-o", "UserKnownHostsFile="+opts.KnownHostsFile)
	}
	return append(args, "--", target, "ollama", "list"), nil
}

// ProbeLocalModels discovers models available on configured hosts
// by SSH-ing to each host and running "ollama list". It returns
// all discovered models across all reachable hosts. Host keys must
// already be trusted (see KnownHostsEnv / ProbeLocalModelsWithOptions).
func ProbeLocalModels(hosts []string, user string) ([]LocalModelInfo, error) {
	return probeLocalModels(hosts, user, DefaultRunner())
}

// probeLocalModels is ProbeLocalModels with an injectable runner so the
// env-var wiring of the public entry point is testable.
func probeLocalModels(hosts []string, user string, runner CommandRunner) ([]LocalModelInfo, error) {
	return ProbeLocalModelsWithOptions(hosts, user, ProbeOptionsFromEnv(), runner)
}

// ProbeLocalModelsWithRunner is like ProbeLocalModels but accepts
// a custom CommandRunner for testing.
func ProbeLocalModelsWithRunner(hosts []string, user string, runner CommandRunner) ([]LocalModelInfo, error) {
	return ProbeLocalModelsWithOptions(hosts, user, ProbeOptions{}, runner)
}

// ProbeLocalModelsWithOptions probes hosts with explicit ProbeOptions.
func ProbeLocalModelsWithOptions(hosts []string, user string, opts ProbeOptions, runner CommandRunner) ([]LocalModelInfo, error) {
	if len(hosts) == 0 {
		return nil, fmt.Errorf("local: no hosts provided")
	}

	var allModels []LocalModelInfo
	var lastErr error

	for _, host := range hosts {
		models, err := probeHost(host, user, opts, runner)
		if err != nil {
			log.Printf("local: skipping host %q: %v", host, err)
			lastErr = err
			continue
		}
		allModels = append(allModels, models...)
	}

	if len(allModels) == 0 && lastErr != nil {
		return nil, fmt.Errorf("local: no models discovered: %w", lastErr)
	}

	return allModels, nil
}

// probeHost discovers models on a single host via SSH + "ollama list".
func probeHost(host, user string, opts ProbeOptions, runner CommandRunner) ([]LocalModelInfo, error) {
	args, err := buildSSHArgs(host, user, opts)
	if err != nil {
		return nil, err
	}
	out, err := runner.Run("ssh", args...)
	if err != nil {
		return nil, fmt.Errorf("ssh to %s: %w", host, err)
	}

	return parseOllamaList(string(out), host)
}

// parseOllamaList parses the output of "ollama list" into
// LocalModelInfo entries. The expected format is tab-separated:
//
//	NAME           ID           SIZE    MODIFIED
//	llava:7b       abc123       4.5 GB  3 days ago
func parseOllamaList(output, host string) ([]LocalModelInfo, error) {
	var models []LocalModelInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Skip header line.
	if scanner.Scan() {
		// header consumed
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Split on whitespace — Ollama output is typically
		// tab/space separated with variable widths.
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]

		// Try to find the size field — it's typically the 3rd
		// or later field and contains "GB" or "MB".
		var sizeStr string
		for i := 1; i < len(fields)-1; i++ {
			combined := fields[i] + " " + fields[i+1]
			if sizePattern.MatchString(combined) {
				sizeStr = combined
				break
			}
		}

		ramRequired := parseSizeBytes(sizeStr)
		paramCount := parseParameterCount(name)

		// Estimate GPU requirement: roughly same as RAM for
		// full GPU offload, less for partial.
		gpuRequired := ramRequired

		model := LocalModelInfo{
			Name:           name,
			Provider:       "ollama",
			Host:           host,
			SupportsVision: isVisionModel(name),
			ParameterCount: paramCount,
			RAMRequired:    ramRequired,
			GPURequired:    gpuRequired,
		}

		// Try to extract quantization from the name/tag.
		model.QuantLevel = extractQuantLevel(name)

		models = append(models, model)
	}

	return models, nil
}

// quantPatterns matches common quantization suffixes.
var quantPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(Q\d+_K_[MSL])`),
	regexp.MustCompile(`(?i)(Q\d+_\d+)`),
	regexp.MustCompile(`(?i)(Q\d+_K)`),
	regexp.MustCompile(`(?i)(Q\d+)`),
	regexp.MustCompile(`(?i)(F16|F32|BF16)`),
}

// extractQuantLevel tries to find a quantization level string
// in a model name or tag.
func extractQuantLevel(name string) string {
	for _, pat := range quantPatterns {
		if m := pat.FindString(name); m != "" {
			return strings.ToUpper(m)
		}
	}
	return ""
}

// ScoreLocalModel scores a local model using the same conceptual
// dimensions as cloud models: quality (estimated from parameter
// count), reliability (based on host reachability, assumed high
// for discovered models), and cost (always zero).
//
// The score is normalized to [0, 1].
func ScoreLocalModel(model LocalModelInfo) float64 {
	// Quality: estimated from parameter count.
	// 70B+ = 0.80, 13B = 0.70, 8B = 0.65, 7B = 0.60,
	// smaller = 0.50.
	qualityScore := estimateQuality(model.ParameterCount)

	// Vision bonus: vision-capable models get a bump.
	visionBonus := 0.0
	if model.SupportsVision {
		visionBonus = 0.10
	}

	// Reliability: local models that were successfully probed
	// are assumed to be highly reliable.
	reliabilityScore := 0.90

	// Cost: always zero for local models — represented as a
	// perfect cost score.
	costScore := 1.0

	// Weighted combination matching the strategy dimensions.
	// quality=0.35, cost=0.20, reliability=0.20, vision=0.25
	score := 0.35*qualityScore +
		0.20*costScore +
		0.20*reliabilityScore +
		0.25*visionBonus

	// Clamp to [0, 1].
	return math.Min(1.0, math.Max(0.0, score))
}

// estimateQuality maps parameter count to an estimated quality
// score. Larger models generally produce better results.
func estimateQuality(params int64) float64 {
	if params <= 0 {
		return 0.50
	}
	billions := float64(params) / 1_000_000_000.0
	switch {
	case billions >= 70:
		return 0.82
	case billions >= 30:
		return 0.75
	case billions >= 13:
		return 0.70
	case billions >= 8:
		return 0.65
	case billions >= 7:
		return 0.60
	case billions >= 3:
		return 0.55
	default:
		return 0.50
	}
}

// ToModelInfo converts a LocalModelInfo to the unified
// strategy.ModelInfo used by the scoring and selection system.
func ToModelInfo(local LocalModelInfo) strategy.ModelInfo {
	qualityScore := estimateQuality(local.ParameterCount)
	if local.SupportsVision {
		qualityScore = math.Min(1.0, qualityScore+0.05)
	}

	// Estimate context window from parameter count.
	contextWindow := 4096
	if local.ParameterCount >= 13_000_000_000 {
		contextWindow = 8192
	}
	if local.ParameterCount >= 30_000_000_000 {
		contextWindow = 16384
	}

	// Estimate latency from parameter count.
	latencyMs := 3000
	if local.ParameterCount >= 30_000_000_000 {
		latencyMs = 8000
	}
	if local.GPURequired > 0 {
		latencyMs = latencyMs / 2 // GPU roughly halves latency
	}

	capabilities := []string{"text"}
	if local.SupportsVision {
		capabilities = append(capabilities, "vision", "local")
	} else {
		capabilities = append(capabilities, "local")
	}

	return strategy.ModelInfo{
		ID:                fmt.Sprintf("local-%s-%s", local.Host, local.Name),
		Name:              fmt.Sprintf("%s (%s/%s)", local.Name, local.Provider, local.Host),
		Provider:          local.Provider,
		Model:             local.Name,
		SupportsVision:    local.SupportsVision,
		SupportsStreaming: true,
		ContextWindow:     contextWindow,
		MaxOutputTokens:   2048,
		AvgLatencyMs:      latencyMs,
		InputCostPer1k:    0.0,
		OutputCostPer1k:   0.0,
		QualityScore:      qualityScore,
		ReliabilityScore:  0.90,
		LastVerified:      time.Now(),
		Capabilities:      capabilities,
		Metadata: map[string]any{
			"host":            local.Host,
			"parameter_count": local.ParameterCount,
			"quant_level":     local.QuantLevel,
			"ram_required":    local.RAMRequired,
			"gpu_required":    local.GPURequired,
			"is_local":        true,
		},
	}
}

// SortByScore sorts local models by their score in descending
// order.
func SortByScore(models []LocalModelInfo) {
	sort.SliceStable(models, func(i, j int) bool {
		return ScoreLocalModel(models[i]) > ScoreLocalModel(models[j])
	})
}
