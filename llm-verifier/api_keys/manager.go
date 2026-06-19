package api_keys

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	FaultyAPIKeysFile      = ".faulty_api_keys"
	UnsupportedAPIKeysFile = ".unsupported_api_keys"
)

type APIKeyEntry struct {
	KeyName   string
	Reason    string
	Timestamp time.Time
}

func GetProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func GetFaultyAPIKeysPath() string {
	return filepath.Join(GetProjectRoot(), FaultyAPIKeysFile)
}

func GetUnsupportedAPIKeysPath() string {
	return filepath.Join(GetProjectRoot(), UnsupportedAPIKeysFile)
}

func ReadFaultyAPIKeys() (map[string]APIKeyEntry, error) {
	return readAPIKeysFile(GetFaultyAPIKeysPath())
}

func ReadUnsupportedAPIKeys() (map[string]APIKeyEntry, error) {
	return readAPIKeysFile(GetUnsupportedAPIKeysPath())
}

func readAPIKeysFile(path string) (map[string]APIKeyEntry, error) {
	result := make(map[string]APIKeyEntry)

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		keyName := strings.TrimSpace(parts[0])
		rawValue := strings.TrimSpace(parts[1])

		if !strings.HasPrefix(rawValue, "\"") || !strings.HasSuffix(rawValue, "\"") {
			continue
		}

		value := strings.Trim(rawValue, "\"")

		entry, err := parseEntry(keyName, value)
		if err != nil {
			continue
		}

		result[keyName] = entry
	}

	return result, nil
}

func parseEntry(keyName, value string) (APIKeyEntry, error) {
	entry := APIKeyEntry{
		KeyName: keyName,
	}

	parts := strings.SplitN(value, " ", 2)
	if len(parts) < 2 {
		entry.Reason = value
		return entry, nil
	}

	timestampPart := parts[0]
	reasonPart := parts[1]

	parsedTime, err := time.Parse("2006-01-02:15-04", timestampPart)
	if err != nil {
		entry.Reason = value
		return entry, nil
	}

	entry.Timestamp = parsedTime
	entry.Reason = reasonPart

	return entry, nil
}

func WriteFaultyAPIKey(keyName, reason string) error {
	return writeAPIKeyEntry(GetFaultyAPIKeysPath(), keyName, reason)
}

func WriteUnsupportedAPIKey(keyName, reason string) error {
	return writeAPIKeyEntry(GetUnsupportedAPIKeysPath(), keyName, reason)
}

func writeAPIKeyEntry(path, keyName, reason string) error {
	entries, err := readAPIKeysFile(path)
	if err != nil {
		return err
	}

	entries[keyName] = APIKeyEntry{
		KeyName:   keyName,
		Reason:    reason,
		Timestamp: time.Now(),
	}

	var lines []string
	lines = append(lines, "# Faulty API Keys - DO NOT COMMIT VALUES")
	lines = append(lines, "# Format: PROVIDER_NAME_API_KEY=\"YEAR-MONTH-DAY:HOUR-MINUTE: Reason\"")
	lines = append(lines, "")

	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := entries[k]
		ts := v.Timestamp.Format("2006-01-02:15-04")
		lines = append(lines, fmt.Sprintf("%s=\"%s %s\"", k, ts, v.Reason))
	}

	content := strings.Join(lines, "\n") + "\n"

	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

func RemoveFaultyAPIKey(keyName string) error {
	return removeAPIKeyEntry(GetFaultyAPIKeysPath(), keyName)
}

func RemoveUnsupportedAPIKey(keyName string) error {
	return removeAPIKeyEntry(GetUnsupportedAPIKeysPath(), keyName)
}

func removeAPIKeyEntry(path, keyName string) error {
	entries, err := readAPIKeysFile(path)
	if err != nil {
		return err
	}

	delete(entries, keyName)

	if len(entries) == 0 {
		err = os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
		return nil
	}

	var lines []string
	lines = append(lines, "# Faulty API Keys - DO NOT COMMIT VALUES")
	lines = append(lines, "# Format: PROVIDER_NAME_API_KEY=\"YEAR-MONTH-DAY:HOUR-MINUTE: Reason\"")
	lines = append(lines, "")

	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := entries[k]
		ts := v.Timestamp.Format("2006-01-02:15-04")
		lines = append(lines, fmt.Sprintf("%s=\"%s %s\"", k, ts, v.Reason))
	}

	content := strings.Join(lines, "\n") + "\n"

	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

func GetKnownProviderEnvVars() map[string]string {
	return map[string]string{
		"OPENAI_API_KEY":      "openai",
		"ANTHROPIC_API_KEY":   "anthropic",
		"HUGGINGFACE_API_KEY": "huggingface",
		"GROQ_API_KEY":        "groq",
		"GEMINI_API_KEY":      "gemini",
		"DEEPSEEK_API_KEY":    "deepseek",
		"NVIDIA_API_KEY":      "nvidia",
		"OPENROUTER_API_KEY":  "openrouter",
		"REPLICATE_API_KEY":   "replicate",
		"FIREWORKS_API_KEY":   "fireworks",
		"TOGETHER_API_KEY":    "together",
		"PERPLEXITY_API_KEY":  "perplexity",
		"MISTRAL_API_KEY":     "mistral",
		"CODESTRAL_API_KEY":   "codestral",
		"CLOUDFLARE_API_KEY":  "cloudflare",
		"SAMBANOVA_API_KEY":   "sambanova",
		"CEREBRAS_API_KEY":    "cerebras",
		"MODAL_API_KEY":       "modal",
		"INFERENCE_API_KEY":   "inference",
		"SILICONFLOW_API_KEY": "siliconflow",
		"NOVITA_API_KEY":      "novita",
		"UPSTAGE_API_KEY":     "upstage",
		"NLP_API_KEY":         "nlpcloud",
		"HYPERBOLIC_API_KEY":  "hyperbolic",
		"ZAI_API_KEY":         "zai",
		"BASETEN_API_KEY":     "baseten",
		"TWELVELABS_API_KEY":  "twelvelabs",
		"CHUTES_API_KEY":      "chutes",
		"KIMI_API_KEY":        "kimi",
		"XIAOMI_MIMO_API_KEY": "xiaomi",
		"SARVAM_API_KEY":      "sarvam",
		"VULAVULA_API_KEY":    "vulavula",
		"VERCEL_API_KEY":      "vercel",
		"COHERE_API_KEY":      "cohere",
		"AI21_API_KEY":        "ai21",
		"ALEPH_ALPHA_API_KEY": "aleph-alpha",
		"WRITER_API_KEY":      "writer",
		"GOOSECAI_API_KEY":    "gooseai",
		"STABILITY_API_KEY":   "stability",
		"MIDJOURNEY_API_KEY":  "midjourney",
		"RUNWAY_API_KEY":      "runway",
		"ELEVENLABS_API_KEY":  "elevenlabs",
		"ASSEMBLYAI_API_KEY":  "assemblyai",
		"GLADIA_API_KEY":      "gladia",
		"FAL_API_KEY":         "fal",
		"RELEVANCE_API_KEY":   "relevance",
		"XAI_API_KEY":         "xai",
		"TOGETHERAI_API_KEY":  "togetherai",
		"OPENCODE_API_KEY":    "opencode",
		"CLAUDE_CODE_API_KEY": "claude-code",
		"QWEN_API_KEY":        "qwen",
		"MCP_API_KEY":         "mcp",
		"LSP_API_KEY":         "lsp",
	}
}

func IsKnownProviderEnvVar(varName string) bool {
	knownVars := GetKnownProviderEnvVars()
	_, exists := knownVars[varName]
	return exists
}

func GetProviderFromEnvVar(varName string) string {
	knownVars := GetKnownProviderEnvVars()
	provider, exists := knownVars[varName]
	if !exists {
		return ""
	}
	return provider
}
