package vector

import (
	"context"
)

// MockEmbeddingService is a mock embedding service for testing
type MockEmbeddingService struct {
	dimension int
}

// NewMockEmbeddingService creates a new mock embedding service
func NewMockEmbeddingService(dimension int) *MockEmbeddingService {
	return &MockEmbeddingService{dimension: dimension}
}

// GenerateEmbedding generates a mock embedding
func (es *MockEmbeddingService) GenerateEmbedding(ctx context.Context, text string) (Vector, error) {
	// Create a simple hash-based vector for demonstration
	vector := make(Vector, es.dimension)
	hash := 0

	for _, char := range text {
		hash = (hash*31 + int(char)) % 1000
	}

	for i := 0; i < es.dimension; i++ {
		// Create pseudo-random but deterministic values
		vector[i] = float64((hash+i*17)%1000) / 1000.0
	}

	return vector, nil
}

// GenerateEmbeddings generates multiple embeddings
func (es *MockEmbeddingService) GenerateEmbeddings(ctx context.Context, texts []string) ([]Vector, error) {
	vectors := make([]Vector, len(texts))
	for i, text := range texts {
		vector, err := es.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		vectors[i] = vector
	}
	return vectors, nil
}

// GetDimension returns the embedding dimension
func (es *MockEmbeddingService) GetDimension() int {
	return es.dimension
}
