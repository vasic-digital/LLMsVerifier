package vectordb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// QdrantConfig holds Qdrant connection configuration
type QdrantConfig struct {
	Host                      string        `yaml:"host" json:"host"`
	HTTPPort                  int           `yaml:"http_port" json:"http_port"`
	GRPCPort                  int           `yaml:"grpc_port" json:"grpc_port"`
	APIKey                    string        `yaml:"api_key" json:"api_key"`
	Timeout                   time.Duration `yaml:"timeout" json:"timeout"`
	VerificationCollection    string        `yaml:"verification_collection" json:"verification_collection"`
	ModelEmbeddingsCollection string        `yaml:"model_embeddings_collection" json:"model_embeddings_collection"`
	VectorDimension           int           `yaml:"vector_dimension" json:"vector_dimension"`
}

// DefaultQdrantConfig returns default Qdrant configuration
func DefaultQdrantConfig() *QdrantConfig {
	return &QdrantConfig{
		Host:                      "localhost",
		HTTPPort:                  6333,
		GRPCPort:                  6334,
		Timeout:                   30 * time.Second,
		VerificationCollection:    "verification_embeddings",
		ModelEmbeddingsCollection: "model_embeddings",
		VectorDimension:           1536,
	}
}

// Validate validates the Qdrant configuration
func (c *QdrantConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.HTTPPort <= 0 || c.HTTPPort > 65535 {
		return fmt.Errorf("http_port must be between 1 and 65535")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.VectorDimension <= 0 {
		return fmt.Errorf("vector_dimension must be positive")
	}
	return nil
}

// GetHTTPURL returns the HTTP URL for Qdrant
func (c *QdrantConfig) GetHTTPURL() string {
	return fmt.Sprintf("http://%s:%d", c.Host, c.HTTPPort)
}

// QdrantClient wraps the Qdrant client for LLMsVerifier vector operations
type QdrantClient struct {
	config    *QdrantConfig
	client    *http.Client
	mu        sync.RWMutex
	connected bool
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(config *QdrantConfig) (*QdrantClient, error) {
	if config == nil {
		config = DefaultQdrantConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &QdrantClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		connected: false,
	}, nil
}

// Connect establishes connection to Qdrant
func (c *QdrantClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify connection
	if err := c.healthCheck(ctx); err != nil {
		return fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	// Ensure collections exist
	if err := c.ensureCollections(ctx); err != nil {
		return fmt.Errorf("failed to ensure collections: %w", err)
	}

	c.connected = true
	return nil
}

func (c *QdrantClient) healthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.config.GetHTTPURL())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c *QdrantClient) ensureCollections(ctx context.Context) error {
	collections := []struct {
		name string
		dim  int
	}{
		{c.config.VerificationCollection, c.config.VectorDimension},
		{c.config.ModelEmbeddingsCollection, c.config.VectorDimension},
	}

	for _, col := range collections {
		exists, err := c.collectionExists(ctx, col.name)
		if err != nil {
			return fmt.Errorf("failed to check collection %s: %w", col.name, err)
		}
		if !exists {
			if err := c.createCollection(ctx, col.name, col.dim); err != nil {
				return fmt.Errorf("failed to create collection %s: %w", col.name, err)
			}
		}
	}
	return nil
}

func (c *QdrantClient) collectionExists(ctx context.Context, name string) (bool, error) {
	url := fmt.Sprintf("%s/collections/%s", c.config.GetHTTPURL(), name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (c *QdrantClient) createCollection(ctx context.Context, name string, dimension int) error {
	url := fmt.Sprintf("%s/collections/%s", c.config.GetHTTPURL(), name)

	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     dimension,
			"distance": "Cosine",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %s", string(body))
	}

	return nil
}

// Close closes the client connection
func (c *QdrantClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	return nil
}

// IsConnected returns whether the client is connected
func (c *QdrantClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// HealthCheck performs a health check on Qdrant
func (c *QdrantClient) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Qdrant")
	}

	return c.healthCheck(ctx)
}

// VerificationEmbedding represents a verification result embedding
type VerificationEmbedding struct {
	ID         string                 `json:"id"`
	ProviderID string                 `json:"provider_id"`
	ModelID    string                 `json:"model_id"`
	Vector     []float32              `json:"vector"`
	Payload    map[string]interface{} `json:"payload"`
	Score      float64                `json:"score"`
	Timestamp  time.Time              `json:"timestamp"`
}

// UpsertVerificationEmbedding upserts a verification embedding
func (c *QdrantClient) UpsertVerificationEmbedding(ctx context.Context, embedding *VerificationEmbedding) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Qdrant")
	}

	url := fmt.Sprintf("%s/collections/%s/points", c.config.GetHTTPURL(), c.config.VerificationCollection)

	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":     embedding.ID,
				"vector": embedding.Vector,
				"payload": map[string]interface{}{
					"provider_id": embedding.ProviderID,
					"model_id":    embedding.ModelID,
					"score":       embedding.Score,
					"timestamp":   embedding.Timestamp.Format(time.RFC3339),
					"metadata":    embedding.Payload,
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert embedding: %s", string(body))
	}

	return nil
}

// SimilarVerification represents a similar verification result
type SimilarVerification struct {
	ID         string                 `json:"id"`
	ProviderID string                 `json:"provider_id"`
	ModelID    string                 `json:"model_id"`
	Score      float64                `json:"score"`
	Similarity float32                `json:"similarity"`
	Payload    map[string]interface{} `json:"payload"`
}

// SearchSimilarVerifications searches for similar verifications
func (c *QdrantClient) SearchSimilarVerifications(ctx context.Context, vector []float32, limit int) ([]*SimilarVerification, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to Qdrant")
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", c.config.GetHTTPURL(), c.config.VerificationCollection)

	payload := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s", string(body))
	}

	var result struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var verifications []*SimilarVerification
	for _, r := range result.Result {
		v := &SimilarVerification{
			ID:         fmt.Sprintf("%v", r.ID),
			Similarity: r.Score,
			Payload:    r.Payload,
		}
		if providerID, ok := r.Payload["provider_id"].(string); ok {
			v.ProviderID = providerID
		}
		if modelID, ok := r.Payload["model_id"].(string); ok {
			v.ModelID = modelID
		}
		if score, ok := r.Payload["score"].(float64); ok {
			v.Score = score
		}
		verifications = append(verifications, v)
	}

	return verifications, nil
}

// ModelEmbedding represents a model embedding
type ModelEmbedding struct {
	ID           string                 `json:"id"`
	ProviderID   string                 `json:"provider_id"`
	ModelID      string                 `json:"model_id"`
	ModelName    string                 `json:"model_name"`
	Vector       []float32              `json:"vector"`
	Capabilities map[string]interface{} `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// UpsertModelEmbedding upserts a model embedding
func (c *QdrantClient) UpsertModelEmbedding(ctx context.Context, embedding *ModelEmbedding) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Qdrant")
	}

	url := fmt.Sprintf("%s/collections/%s/points", c.config.GetHTTPURL(), c.config.ModelEmbeddingsCollection)

	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":     embedding.ID,
				"vector": embedding.Vector,
				"payload": map[string]interface{}{
					"provider_id":  embedding.ProviderID,
					"model_id":     embedding.ModelID,
					"model_name":   embedding.ModelName,
					"capabilities": embedding.Capabilities,
					"metadata":     embedding.Metadata,
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert model embedding: %s", string(body))
	}

	return nil
}

// SimilarModel represents a similar model
type SimilarModel struct {
	ID           string                 `json:"id"`
	ProviderID   string                 `json:"provider_id"`
	ModelID      string                 `json:"model_id"`
	ModelName    string                 `json:"model_name"`
	Similarity   float32                `json:"similarity"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

// SearchSimilarModels searches for similar models
func (c *QdrantClient) SearchSimilarModels(ctx context.Context, vector []float32, limit int) ([]*SimilarModel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to Qdrant")
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", c.config.GetHTTPURL(), c.config.ModelEmbeddingsCollection)

	payload := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s", string(body))
	}

	var result struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var models []*SimilarModel
	for _, r := range result.Result {
		m := &SimilarModel{
			ID:         fmt.Sprintf("%v", r.ID),
			Similarity: r.Score,
		}
		if providerID, ok := r.Payload["provider_id"].(string); ok {
			m.ProviderID = providerID
		}
		if modelID, ok := r.Payload["model_id"].(string); ok {
			m.ModelID = modelID
		}
		if modelName, ok := r.Payload["model_name"].(string); ok {
			m.ModelName = modelName
		}
		if capabilities, ok := r.Payload["capabilities"].(map[string]interface{}); ok {
			m.Capabilities = capabilities
		}
		models = append(models, m)
	}

	return models, nil
}

// DeleteVerificationEmbedding deletes a verification embedding
func (c *QdrantClient) DeleteVerificationEmbedding(ctx context.Context, id string) error {
	return c.deletePoint(ctx, c.config.VerificationCollection, id)
}

// DeleteModelEmbedding deletes a model embedding
func (c *QdrantClient) DeleteModelEmbedding(ctx context.Context, id string) error {
	return c.deletePoint(ctx, c.config.ModelEmbeddingsCollection, id)
}

func (c *QdrantClient) deletePoint(ctx context.Context, collection, id string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Qdrant")
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", c.config.GetHTTPURL(), collection)

	payload := map[string]interface{}{
		"points": []string{id},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete point: %s", string(body))
	}

	return nil
}
