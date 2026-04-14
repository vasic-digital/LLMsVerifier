package api_keys

import (
	"os"
	"strings"
)

type EnvVarScanner struct {
	knownProviders map[string]string
}

func NewEnvVarScanner() *EnvVarScanner {
	return &EnvVarScanner{
		knownProviders: GetKnownProviderEnvVars(),
	}
}

func (e *EnvVarScanner) ScanEnvForUnsupportedKeys() (map[string]string, error) {
	unsupported := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if !strings.HasSuffix(key, "_API_KEY") && !strings.HasSuffix(key, "_KEY") {
			continue
		}

		if key == "API_KEY" || key == "KEY" || key == "SECRET" {
			continue
		}

		if _, known := e.knownProviders[key]; known {
			continue
		}

		unsupported[key] = value
	}

	return unsupported, nil
}

func (e *EnvVarScanner) GetEnvVarValue(key string) string {
	return os.Getenv(key)
}

func (e *EnvVarScanner) HasValue(key string) bool {
	value := os.Getenv(key)
	return value != "" && value != "your-"+strings.ToLower(strings.TrimSuffix(key, "_KEY"))+"-here"
}
