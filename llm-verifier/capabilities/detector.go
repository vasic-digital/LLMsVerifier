package capabilities

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Detector dynamically detects LLM provider capabilities
type Detector struct {
	httpClient  *http.Client
	cache       map[string]*ProviderCapabilities
	cacheExpiry map[string]time.Time
	cacheTTL    time.Duration
	mu          sync.RWMutex
}

// NewDetector creates a new capability detector
func NewDetector() *Detector {
	return &Detector{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:       make(map[string]*ProviderCapabilities),
		cacheExpiry: make(map[string]time.Time),
		cacheTTL:    15 * time.Minute,
	}
}

// DetectProviderCapabilities detects capabilities for a specific provider
func (d *Detector) DetectProviderCapabilities(ctx context.Context, provider, apiKey string) (*ProviderCapabilities, error) {
	// Check cache first
	d.mu.RLock()
	if caps, found := d.cache[provider]; found {
		if time.Now().Before(d.cacheExpiry[provider]) {
			d.mu.RUnlock()
			return caps, nil
		}
	}
	d.mu.RUnlock()

	// Get base capabilities from registry
	caps := GetProviderBaseCapabilities(provider)
	if caps == nil {
		caps = &ProviderCapabilities{
			Provider: provider,
		}
	}

	// Run dynamic detection tests. A probe is attempted ONLY when an apiKey is
	// supplied; without one, no request reaches any provider endpoint.
	probed := false
	if apiKey != "" {
		d.detectStreaming(ctx, caps, provider, apiKey)
		d.detectModels(ctx, caps, provider, apiKey)
		probed = true
	}

	// Fail-closed: only stamp Verified/VerifiedAt when a probe actually ran.
	// Stamping Verified=true with no probe (empty apiKey) would be a self-cert
	// bluff, inconsistent with the C3 seed fail-closed invariant.
	if probed {
		caps.Verified = true
		caps.VerifiedAt = time.Now()
	}

	// Update cache
	d.mu.Lock()
	d.cache[provider] = caps
	d.cacheExpiry[provider] = time.Now().Add(d.cacheTTL)
	d.mu.Unlock()

	return caps, nil
}

// detectStreaming tests if streaming is actually supported
func (d *Detector) detectStreaming(ctx context.Context, caps *ProviderCapabilities, provider, apiKey string) {
	endpoint := getProviderStreamEndpoint(provider)
	if endpoint == "" {
		return
	}

	// Create a minimal streaming request
	body := getMinimalStreamRequest(provider)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(body))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(req, provider, apiKey)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		caps.Streaming.Supported = false
		return
	}
	defer resp.Body.Close()

	// Check for SSE content type
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		caps.Streaming.Supported = true
		if !containsStreamingType(caps.Streaming.Types, StreamingTypeSSE) {
			caps.Streaming.Types = append(caps.Streaming.Types, StreamingTypeSSE)
		}
	}
}

// detectModels queries the models endpoint to detect available models
func (d *Detector) detectModels(ctx context.Context, caps *ProviderCapabilities, provider, apiKey string) {
	endpoint := getProviderModelsEndpoint(provider)
	if endpoint == "" {
		return
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return
	}

	setAuthHeader(req, provider, apiKey)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	// Parse models response
	body, _ := io.ReadAll(resp.Body)
	models := parseModelsResponse(provider, body)

	if len(models) > 0 && caps.Custom == nil {
		caps.Custom = make(map[string]interface{})
	}
	if len(models) > 0 {
		caps.Custom["detected_models"] = models
	}
}

// Query queries capabilities with specific requirements
func (d *Detector) Query(ctx context.Context, query *CapabilityQuery) (*CapabilityQueryResult, error) {
	result := &CapabilityQueryResult{
		Query:        query,
		Matches:      false,
		PartialMatch: 0.0,
	}

	// Get provider capabilities
	if query.Provider != "" {
		caps := GetProviderBaseCapabilities(query.Provider)
		if caps == nil {
			result.MissingCaps = append(result.MissingCaps, "provider_not_found")
			return result, nil
		}
		result.Provider = caps

		// Check requirements
		matchCount := 0
		totalReqs := 0

		if query.RequireStreaming != nil {
			totalReqs++
			if caps.Streaming.Supported && containsStreamingType(caps.Streaming.Types, *query.RequireStreaming) {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, fmt.Sprintf("streaming:%s", *query.RequireStreaming))
			}
		}

		if query.RequireHTTP3 {
			totalReqs++
			if caps.Network.HTTP3Supported {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, "http3")
				result.Recommendations = append(result.Recommendations, tr("llmsverifier_capabilities_rec_http3_unsupported_any"))
			}
		}

		if query.RequireCompression != nil {
			totalReqs++
			if caps.Compression.Supported && containsCompressionType(caps.Compression.Types, *query.RequireCompression) {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, fmt.Sprintf("compression:%s", *query.RequireCompression))
			}
		}

		if query.RequireCaching != nil {
			totalReqs++
			if caps.Caching.Supported && containsCachingType(caps.Caching.Types, *query.RequireCaching) {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, fmt.Sprintf("caching:%s", *query.RequireCaching))
			}
		}

		if query.RequireProtocol != nil {
			totalReqs++
			if containsProtocol(caps.Protocols, *query.RequireProtocol) {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, fmt.Sprintf("protocol:%s", *query.RequireProtocol))
			}
		}

		if query.RequireOAuth {
			totalReqs++
			if caps.Auth.OAuthSupported {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, "oauth")
			}
		}

		if query.RequireVision {
			totalReqs++
			if caps.Model_.Vision {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, "vision")
			}
		}

		if query.RequireFunctions {
			totalReqs++
			if caps.Model_.FunctionCalling {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, "function_calling")
			}
		}

		if query.RequireReasoning {
			totalReqs++
			if caps.Model_.Reasoning {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, "reasoning")
			}
		}

		if query.MinContextTokens > 0 {
			totalReqs++
			if caps.Model_.MaxContextTokens >= query.MinContextTokens {
				matchCount++
			} else {
				result.MissingCaps = append(result.MissingCaps, fmt.Sprintf("context_tokens:%d", query.MinContextTokens))
			}
		}

		if totalReqs > 0 {
			result.PartialMatch = float64(matchCount) / float64(totalReqs)
			result.Matches = matchCount == totalReqs
		} else {
			result.Matches = true
			result.PartialMatch = 1.0
		}
	}

	// Get CLI agent capabilities
	if query.CLIAgent != "" {
		agentCaps := GetCLIAgentCapabilities(query.CLIAgent)
		if agentCaps == nil {
			result.MissingCaps = append(result.MissingCaps, "cli_agent_not_found")
			return result, nil
		}
		result.CLIAgent = agentCaps
	}

	return result, nil
}

// GetCapabilityMatrix generates a complete capability matrix
func (d *Detector) GetCapabilityMatrix() *CapabilityMatrix {
	matrix := &CapabilityMatrix{
		GeneratedAt:   time.Now(),
		Providers:     make(map[string]*ProviderCapabilities),
		CLIAgents:     make(map[string]*CLIAgentCapabilities),
		ByStreaming:   make(map[StreamingType][]string),
		ByProtocol:    make(map[ProtocolType][]string),
		ByCompression: make(map[CompressionType][]string),
		ByCaching:     make(map[CachingType][]string),
		ByAuth:        make(map[AuthType][]string),
	}

	// Add all providers
	for _, provider := range GetAllProviders() {
		caps := GetProviderBaseCapabilities(provider)
		if caps != nil {
			matrix.Providers[provider] = caps

			// Index by streaming type
			for _, st := range caps.Streaming.Types {
				matrix.ByStreaming[st] = append(matrix.ByStreaming[st], provider)
			}

			// Index by protocol
			for _, p := range caps.Protocols {
				matrix.ByProtocol[p] = append(matrix.ByProtocol[p], provider)
			}

			// Index by compression
			for _, c := range caps.Compression.Types {
				matrix.ByCompression[c] = append(matrix.ByCompression[c], provider)
			}

			// Index by caching
			for _, c := range caps.Caching.Types {
				matrix.ByCaching[c] = append(matrix.ByCaching[c], provider)
			}

			// Index by auth
			for _, a := range caps.Auth.Types {
				matrix.ByAuth[a] = append(matrix.ByAuth[a], provider)
			}
		}
	}

	// Add all CLI agents
	for _, agent := range GetAllCLIAgents() {
		caps := GetCLIAgentCapabilities(agent)
		if caps != nil {
			matrix.CLIAgents[agent] = caps
		}
	}

	return matrix
}

// Helper functions
func containsStreamingType(types []StreamingType, t StreamingType) bool {
	for _, st := range types {
		if st == t {
			return true
		}
	}
	return false
}

func containsCompressionType(types []CompressionType, t CompressionType) bool {
	for _, ct := range types {
		if ct == t {
			return true
		}
	}
	return false
}

func containsCachingType(types []CachingType, t CachingType) bool {
	for _, ct := range types {
		if ct == t {
			return true
		}
	}
	return false
}

func containsProtocol(protocols []ProtocolType, p ProtocolType) bool {
	for _, pt := range protocols {
		if pt == p {
			return true
		}
	}
	return false
}

func getProviderStreamEndpoint(provider string) string {
	endpoints := map[string]string{
		"openai":    "https://api.openai.com/v1/chat/completions",
		"anthropic": "https://api.anthropic.com/v1/messages",
		"deepseek":  "https://api.deepseek.com/v1/chat/completions",
		"gemini":    "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:streamGenerateContent",
		"qwen":      "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation",
		"groq":      "https://api.groq.com/openai/v1/chat/completions",
		"mistral":   "https://api.mistral.ai/v1/chat/completions",
	}
	return endpoints[provider]
}

func getProviderModelsEndpoint(provider string) string {
	endpoints := map[string]string{
		"openai":   "https://api.openai.com/v1/models",
		"deepseek": "https://api.deepseek.com/v1/models",
		"groq":     "https://api.groq.com/openai/v1/models",
		"mistral":  "https://api.mistral.ai/v1/models",
	}
	return endpoints[provider]
}

func getMinimalStreamRequest(provider string) string {
	switch provider {
	case "anthropic":
		return `{"model":"claude-3-haiku-20240307","max_tokens":1,"messages":[{"role":"user","content":"Hi"}],"stream":true}`
	default:
		return `{"model":"gpt-3.5-turbo","max_tokens":1,"messages":[{"role":"user","content":"Hi"}],"stream":true}`
	}
}

func setAuthHeader(req *http.Request, provider, apiKey string) {
	switch provider {
	case "anthropic":
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case "gemini":
		// API key is typically in URL for Gemini
		req.URL.RawQuery = fmt.Sprintf("key=%s", apiKey)
	default:
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}
}

func parseModelsResponse(provider string, body []byte) []string {
	var models []string

	// Generic OpenAI-compatible response
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err == nil {
		for _, m := range resp.Data {
			models = append(models, m.ID)
		}
	}

	return models
}
