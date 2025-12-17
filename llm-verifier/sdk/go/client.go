package llmverifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMVerifierClient provides a Go client for the LLM Verifier REST API
type LLMVerifierClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewLLMVerifierClient creates a new client instance
func NewLLMVerifierClient(baseURL, apiKey string) *LLMVerifierClient {
	return &LLMVerifierClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
	}
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	User      User   `json:"user"`
}

// User represents a user
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Login performs user authentication
func (c *LLMVerifierClient) Login(username, password string) (*AuthResponse, error) {
	req := map[string]string{
		"username": username,
		"password": password,
	}

	var resp AuthResponse
	err := c.post("/auth/login", req, &resp)
	if err != nil {
		return nil, err
	}

	// Set the token for future requests
	c.apiKey = resp.Token
	return &resp, nil
}

// Model represents an LLM model
type Model struct {
	ID             int       `json:"id"`
	ProviderID     int       `json:"provider_id"`
	ModelID        string    `json:"model_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	Architecture   string    `json:"architecture,omitempty"`
	ParameterCount int       `json:"parameter_count,omitempty"`
	ContextWindow  int       `json:"context_window_tokens,omitempty"`
	MaxOutput      int       `json:"max_output_tokens,omitempty"`
	Score          float64   `json:"overall_score,omitempty"`
	Status         string    `json:"verification_status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// GetModels retrieves all models with optional filtering
func (c *LLMVerifierClient) GetModels(limit, offset int, provider string) ([]Model, error) {
	params := make(map[string]string)
	if limit > 0 {
		params["limit"] = fmt.Sprintf("%d", limit)
	}
	if offset > 0 {
		params["offset"] = fmt.Sprintf("%d", offset)
	}
	if provider != "" {
		params["provider"] = provider
	}

	var models []Model
	err := c.get("/api/v1/models", params, &models)
	return models, err
}

// GetModel retrieves a specific model by ID
func (c *LLMVerifierClient) GetModel(id int) (*Model, error) {
	var model Model
	err := c.get(fmt.Sprintf("/api/v1/models/%d", id), nil, &model)
	return &model, err
}

// VerifyModel triggers verification for a specific model
func (c *LLMVerifierClient) VerifyModel(modelID string) (*VerificationResult, error) {
	req := map[string]string{
		"model_id": modelID,
	}

	var result VerificationResult
	err := c.post(fmt.Sprintf("/api/v1/models/%s/verify", modelID), req, &result)
	return &result, err
}

// VerificationResult represents a verification result
type VerificationResult struct {
	ID               int        `json:"id"`
	ModelID          int        `json:"model_id"`
	Status           string     `json:"status"`
	Score            float64    `json:"overall_score"`
	CodeScore        float64    `json:"code_capability_score"`
	Responsiveness   float64    `json:"responsiveness_score"`
	Reliability      float64    `json:"reliability_score"`
	FeatureRichness  float64    `json:"feature_richness_score"`
	ValueProposition float64    `json:"value_proposition_score"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
}

// GetVerificationResults retrieves verification results
func (c *LLMVerifierClient) GetVerificationResults(limit, offset int) ([]VerificationResult, error) {
	params := make(map[string]string)
	if limit > 0 {
		params["limit"] = fmt.Sprintf("%d", limit)
	}
	if offset > 0 {
		params["offset"] = fmt.Sprintf("%d", offset)
	}

	var results []VerificationResult
	err := c.get("/api/v1/verification-results", params, &results)
	return results, err
}

// Provider represents an LLM provider
type Provider struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Endpoint    string    `json:"endpoint"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetProviders retrieves all providers
func (c *LLMVerifierClient) GetProviders() ([]Provider, error) {
	var providers []Provider
	err := c.get("/api/v1/providers", nil, &providers)
	return providers, err
}

// HealthStatus represents system health status
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services,omitempty"`
}

// GetHealth retrieves system health status
func (c *LLMVerifierClient) GetHealth() (*HealthStatus, error) {
	var health HealthStatus
	err := c.get("/health", nil, &health)
	return &health, err
}

// SystemInfo represents system information
type SystemInfo struct {
	Version        string `json:"version"`
	GoVersion      string `json:"go_version"`
	BuildTime      string `json:"build_time"`
	DatabaseSize   int64  `json:"database_size"`
	ModelsCount    int    `json:"models_count"`
	ProvidersCount int    `json:"providers_count"`
	Uptime         string `json:"uptime"`
}

// GetSystemInfo retrieves system information
func (c *LLMVerifierClient) GetSystemInfo() (*SystemInfo, error) {
	var info SystemInfo
	err := c.get("/api/v1/system/info", nil, &info)
	return &info, err
}

// Helper methods

func (c *LLMVerifierClient) get(endpoint string, params map[string]string, result interface{}) error {
	url := c.baseURL + endpoint
	if params != nil && len(params) > 0 {
		url += "?"
		for k, v := range params {
			url += fmt.Sprintf("%s=%s&", k, v)
		}
		url = url[:len(url)-1] // Remove trailing &
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *LLMVerifierClient) post(endpoint string, data interface{}, result interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Example usage:
//
// client := NewLLMVerifierClient("http://localhost:8080", "")
// auth, err := client.Login("admin", "password")
// if err != nil {
//     log.Fatal(err)
// }
//
// models, err := client.GetModels(10, 0, "")
// if err != nil {
//     log.Fatal(err)
// }
//
// fmt.Printf("Found %d models\n", len(models))
