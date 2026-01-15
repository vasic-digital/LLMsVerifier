package vectordb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultQdrantConfig(t *testing.T) {
	config := DefaultQdrantConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 6333, config.HTTPPort)
	assert.Equal(t, 6334, config.GRPCPort)
	assert.Empty(t, config.APIKey)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, "verification_embeddings", config.VerificationCollection)
	assert.Equal(t, "model_embeddings", config.ModelEmbeddingsCollection)
	assert.Equal(t, 1536, config.VectorDimension)
}

func TestQdrantConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*QdrantConfig)
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default config",
			modify:      func(c *QdrantConfig) {},
			expectError: false,
		},
		{
			name: "empty host",
			modify: func(c *QdrantConfig) {
				c.Host = ""
			},
			expectError: true,
			errorMsg:    "host is required",
		},
		{
			name: "invalid http port",
			modify: func(c *QdrantConfig) {
				c.HTTPPort = 0
			},
			expectError: true,
			errorMsg:    "http_port must be between 1 and 65535",
		},
		{
			name: "invalid timeout",
			modify: func(c *QdrantConfig) {
				c.Timeout = 0
			},
			expectError: true,
			errorMsg:    "timeout must be positive",
		},
		{
			name: "invalid vector dimension",
			modify: func(c *QdrantConfig) {
				c.VectorDimension = 0
			},
			expectError: true,
			errorMsg:    "vector_dimension must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultQdrantConfig()
			tt.modify(config)

			err := config.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQdrantConfigGetHTTPURL(t *testing.T) {
	config := DefaultQdrantConfig()
	config.Host = "qdrant-server"
	config.HTTPPort = 6333

	assert.Equal(t, "http://qdrant-server:6333", config.GetHTTPURL())
}

func TestNewQdrantClient(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		client, err := NewQdrantClient(nil)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.False(t, client.IsConnected())
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &QdrantConfig{
			Host:            "qdrant.example.com",
			HTTPPort:        6333,
			GRPCPort:        6334,
			Timeout:         60 * time.Second,
			VectorDimension: 768,
		}
		client, err := NewQdrantClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("with invalid config", func(t *testing.T) {
		config := &QdrantConfig{
			Host: "",
		}
		client, err := NewQdrantClient(config)
		require.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestVerificationEmbedding(t *testing.T) {
	embedding := &VerificationEmbedding{
		ID:         "emb-123",
		ProviderID: "openai",
		ModelID:    "gpt-4",
		Vector:     make([]float32, 1536),
		Payload: map[string]interface{}{
			"test_type": "accuracy",
		},
		Score:     0.95,
		Timestamp: time.Now(),
	}

	assert.Equal(t, "emb-123", embedding.ID)
	assert.Equal(t, "openai", embedding.ProviderID)
	assert.Equal(t, "gpt-4", embedding.ModelID)
	assert.Len(t, embedding.Vector, 1536)
	assert.Equal(t, 0.95, embedding.Score)
}

func TestSimilarVerification(t *testing.T) {
	similar := &SimilarVerification{
		ID:         "sim-123",
		ProviderID: "anthropic",
		ModelID:    "claude-3",
		Score:      0.92,
		Similarity: 0.98,
		Payload: map[string]interface{}{
			"tests_passed": 7,
		},
	}

	assert.Equal(t, "sim-123", similar.ID)
	assert.Equal(t, "anthropic", similar.ProviderID)
	assert.Equal(t, float32(0.98), similar.Similarity)
}

func TestModelEmbedding(t *testing.T) {
	embedding := &ModelEmbedding{
		ID:         "model-emb-123",
		ProviderID: "openai",
		ModelID:    "gpt-4",
		ModelName:  "GPT-4 Turbo",
		Vector:     make([]float32, 1536),
		Capabilities: map[string]interface{}{
			"context_length": 128000,
			"vision":         true,
		},
		Metadata: map[string]interface{}{
			"release_date": "2024-01-01",
		},
	}

	assert.Equal(t, "model-emb-123", embedding.ID)
	assert.Equal(t, "GPT-4 Turbo", embedding.ModelName)
	assert.Len(t, embedding.Vector, 1536)
}

func TestSimilarModel(t *testing.T) {
	similar := &SimilarModel{
		ID:         "sim-model-123",
		ProviderID: "anthropic",
		ModelID:    "claude-3-opus",
		ModelName:  "Claude 3 Opus",
		Similarity: 0.95,
		Capabilities: map[string]interface{}{
			"context_length": 200000,
		},
	}

	assert.Equal(t, "sim-model-123", similar.ID)
	assert.Equal(t, "Claude 3 Opus", similar.ModelName)
	assert.Equal(t, float32(0.95), similar.Similarity)
}
