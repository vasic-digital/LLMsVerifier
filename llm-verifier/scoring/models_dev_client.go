package scoring

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/quic-go/quic-go/http3"
	"llm-verifier/logging"
)

// ModelsDevClient handles API calls to models.dev with HTTP/3 and Brotli support
type ModelsDevClient struct {
	httpClient     *http.Client
	http3Client    *http.Client
	baseURL        string
	logger         *logging.Logger
	useHTTP3       bool
	useBrotli      bool
	requestTimeout time.Duration
}

// ModelsDevAPIResponse represents the API response from models.dev
type ModelsDevAPIResponse struct {
	Models []ModelsDevModel `json:"models"`
}

// ModelsDevModel represents a single model from models.dev
type ModelsDevModel struct {
	Provider            string                `json:"provider"`
	Model               string                `json:"model"`
	Family              string                `json:"family"`
	ProviderID          string                `json:"provider_id"`
	ModelID             string                `json:"model_id"`
	ToolCall            bool                  `json:"tool_call"`
	Reasoning           bool                  `json:"reasoning"`
	Input               float64               `json:"input"`
	Output              float64               `json:"output"`
	InputCostPer1M      float64               `json:"input_cost_per_1m_tokens"`
	OutputCostPer1M     float64               `json:"output_cost_per_1m_tokens"`
	ReasoningCostPer1M  float64               `json:"reasoning_cost_per_1m_tokens"`
	CacheReadCostPer1M  float64               `json:"cache_read_cost_per_1m_tokens"`
	CacheWriteCostPer1M float64               `json:"cache_write_cost_per_1m_tokens"`
	AudioInputCostPer1M float64               `json:"audio_input_cost_per_1m_tokens"`
	AudioOutputCostPer1M float64              `json:"audio_output_cost_per_1m_tokens"`
	ContextLimit        int                   `json:"context_limit"`
	InputLimit          int                   `json:"input_limit"`
	OutputLimit         int                   `json:"output_limit"`
	StructuredOutput    bool                  `json:"structured_output"`
	Temperature         bool                  `json:"temperature"`
	Weights             string                `json:"weights"`
	Knowledge           string                `json:"knowledge"`
	ReleaseDate         string                `json:"release_date"`
	LastUpdated         string                `json:"last_updated"`
	AdditionalData      ModelsDevAdditionalData `json:"additional_data,omitempty"`
}

// ModelsDevAdditionalData contains extra information about the model
type ModelsDevAdditionalData struct {
	ParameterCount      int64    `json:"parameter_count,omitempty"`
	Architecture        string   `json:"architecture,omitempty"`
	TrainingDataCutoff  string   `json:"training_data_cutoff,omitempty"`
	OpenWeights         bool     `json:"open_weights,omitempty"`
	Multimodal          bool     `json:"multimodal,omitempty"`
	Vision              bool     `json:"vision,omitempty"`
	Audio               bool     `json:"audio,omitempty"`
	Video               bool     `json:"video,omitempty"`
	Languages           []string `json:"languages,omitempty"`
	Tags                []string `json:"tags,omitempty"`
	License             string   `json:"license,omitempty"`
	DocumentationURL    string   `json:"documentation_url,omitempty"`
	APIEndpoint         string   `json:"api_endpoint,omitempty"`
}

// ClientConfig holds configuration for the ModelsDevClient
type ClientConfig struct {
	BaseURL        string
	RequestTimeout time.Duration
	UseHTTP3       bool
	UseBrotli      bool
	EnableLogging  bool
}

// DefaultClientConfig returns default configuration
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BaseURL:        "https://models.dev",
		RequestTimeout: 30 * time.Second,
		UseHTTP3:       true,
		UseBrotli:      true,
		EnableLogging:  true,
	}
}

// NewModelsDevClient creates a new client with HTTP/3 and Brotli support
func NewModelsDevClient(config ClientConfig, logger *logging.Logger) (*ModelsDevClient, error) {
	client := &ModelsDevClient{
		baseURL:        config.BaseURL,
		logger:         logger,
		useHTTP3:       config.UseHTTP3,
		useBrotli:      config.UseBrotli,
		requestTimeout: config.RequestTimeout,
	}

	// Create HTTP/2 client (fallback)
	http2Transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false, // Enable gzip compression
	}

	client.httpClient = &http.Client{
		Transport: http2Transport,
		Timeout:   config.RequestTimeout,
	}

	// Create HTTP/3 client if enabled
	if config.UseHTTP3 {
		http3Transport := &http3.Transport{
			EnableDatagrams: true,
		}

		client.http3Client = &http.Client{
			Transport: http3Transport,
			Timeout:   config.RequestTimeout,
		}
	}

	return client, nil
}

// FetchAllModels fetches all models from models.dev API
func (c *ModelsDevClient) FetchAllModels(ctx context.Context) (*ModelsDevAPIResponse, error) {
	endpoint := fmt.Sprintf("%s/api.json", c.baseURL)
	return c.makeRequest(ctx, endpoint)
}

// FetchModelByID fetches a specific model by its model ID
func (c *ModelsDevClient) FetchModelByID(ctx context.Context, modelID string) (*ModelsDevModel, error) {
	// First fetch all models, then filter by model ID
	response, err := c.FetchAllModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}

	for _, model := range response.Models {
		if model.ModelID == modelID {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model with ID %s not found", modelID)
}

// FetchModelsByProvider fetches models from a specific provider
func (c *ModelsDevClient) FetchModelsByProvider(ctx context.Context, providerID string) ([]ModelsDevModel, error) {
	response, err := c.FetchAllModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}

	var providerModels []ModelsDevModel
	for _, model := range response.Models {
		if model.ProviderID == providerID {
			providerModels = append(providerModels, model)
		}
	}

	return providerModels, nil
}

// makeRequest performs the actual HTTP request with compression support
func (c *ModelsDevClient) makeRequest(ctx context.Context, endpoint string) (*ModelsDevAPIResponse, error) {
	var resp *http.Response
	var err error

	// Try HTTP/3 first if enabled
	if c.useHTTP3 && c.http3Client != nil {
		resp, err = c.doRequest(ctx, c.http3Client, endpoint)
		if err == nil {
			if c.logger != nil {
				c.logger.Info("Successfully used HTTP/3 for models.dev API call")
			}
		} else if c.logger != nil {
			c.logger.Warn("HTTP/3 request failed, falling back to HTTP/2", "error", err)
		}
	}

	// Fallback to HTTP/2 if HTTP/3 failed or not enabled
	if resp == nil && err != nil {
		resp, err = c.doRequest(ctx, c.httpClient, endpoint)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
	}

	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Handle response based on content encoding
	var reader io.Reader = resp.Body
	switch resp.Header.Get("Content-Encoding") {
	case "br":
		if c.useBrotli {
			reader = brotli.NewReader(resp.Body)
			if c.logger != nil {
				c.logger.Info("Successfully decoded Brotli compressed response")
			}
		}
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
	}

	// Decode JSON response
	var apiResponse ModelsDevAPIResponse
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return &apiResponse, nil
}

// doRequest performs the HTTP request with proper headers
func (c *ModelsDevClient) doRequest(ctx context.Context, client *http.Client, endpoint string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for compression support
	if c.useBrotli {
		req.Header.Set("Accept-Encoding", "br, gzip, deflate")
	} else {
		req.Header.Set("Accept-Encoding", "gzip, deflate")
	}
	
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "LLM-Verifier/1.0")

	return client.Do(req)
}

// GetProviderLogo fetches the provider logo SVG
func (c *ModelsDevClient) GetProviderLogo(ctx context.Context, providerID string) (string, error) {
	endpoint := fmt.Sprintf("%s/logos/%s.svg", c.baseURL, providerID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "LLM-Verifier/1.0")

	// Use HTTP/2 client for logo requests (HTTP/3 might be overkill for small SVG files)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch logo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("logo request failed with status %d", resp.StatusCode)
	}

	logoData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read logo data: %w", err)
	}

	return string(logoData), nil
}

// ConvertToInternalModel converts ModelsDevModel to internal Model format
func (c *ModelsDevClient) ConvertToInternalModel(devModel ModelsDevModel) *Model {
	return &Model{
		ProviderID:          devModel.ProviderID,
		ModelID:             devModel.ModelID,
		Name:                devModel.Model,
		Description:         fmt.Sprintf("%s model from %s", devModel.Model, devModel.Provider),
		Family:              devModel.Family,
		SupportsToolCall:    devModel.ToolCall,
		SupportsReasoning:   devModel.Reasoning,
		InputCostPer1M:      devModel.InputCostPer1M,
		OutputCostPer1M:     devModel.OutputCostPer1M,
		ReasoningCostPer1M:  devModel.ReasoningCostPer1M,
		CacheReadCostPer1M:  devModel.CacheReadCostPer1M,
		CacheWriteCostPer1M: devModel.CacheWriteCostPer1M,
		AudioInputCostPer1M: devModel.AudioInputCostPer1M,
		AudioOutputCostPer1M: devModel.AudioOutputCostPer1M,
		ContextLimit:        devModel.ContextLimit,
		InputLimit:          devModel.InputLimit,
		OutputLimit:         devModel.OutputLimit,
		SupportsStructuredOutput: devModel.StructuredOutput,
		SupportsTemperature: devModel.Temperature,
		Weights:             devModel.Weights,
		KnowledgeCutoff:     devModel.Knowledge,
		ReleaseDate:         devModel.ReleaseDate,
		LastUpdated:         devModel.LastUpdated,
		AdditionalData:      devModel.AdditionalData,
	}
}

// Model represents the internal model structure
type Model struct {
	ProviderID               string
	ModelID                  string
	Name                     string
	Description              string
	Family                   string
	SupportsToolCall         bool
	SupportsReasoning        bool
	InputCostPer1M           float64
	OutputCostPer1M          float64
	ReasoningCostPer1M       float64
	CacheReadCostPer1M       float64
	CacheWriteCostPer1M      float64
	AudioInputCostPer1M      float64
	AudioOutputCostPer1M     float64
	ContextLimit             int
	InputLimit               int
	OutputLimit              int
	SupportsStructuredOutput bool
	SupportsTemperature      bool
	Weights                  string
	KnowledgeCutoff          string
	ReleaseDate              string
	LastUpdated              string
	AdditionalData           ModelsDevAdditionalData
}