package vector

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	ctxt "llm-verifier/enhanced/context"
)

// Vector represents a vector embedding
type Vector []float64

// Document represents a document with its vector embedding
type Document struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Vector    Vector                 `json:"vector"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
}

// VectorDatabase interface for vector operations
type VectorDatabase interface {
	// Store stores a document with its vector
	Store(ctx context.Context, doc *Document) error

	// Search searches for similar documents
	Search(ctx context.Context, queryVector Vector, limit int, threshold float64) ([]SearchResult, error)

	// Delete removes a document by ID
	Delete(ctx context.Context, id string) error

	// Get retrieves a document by ID
	Get(ctx context.Context, id string) (*Document, error)

	// List returns all document IDs
	List(ctx context.Context) ([]string, error)

	// Close closes the database connection
	Close() error
}

// InMemoryVectorDB is a simple in-memory vector database for demonstration
type InMemoryVectorDB struct {
	documents map[string]*Document
	mu        sync.RWMutex
}

// NewInMemoryVectorDB creates a new in-memory vector database
func NewInMemoryVectorDB() *InMemoryVectorDB {
	return &InMemoryVectorDB{
		documents: make(map[string]*Document),
	}
}

// Store stores a document
func (db *InMemoryVectorDB) Store(ctx context.Context, doc *Document) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if doc.Timestamp.IsZero() {
		doc.Timestamp = time.Now()
	}

	db.documents[doc.ID] = doc
	return nil
}

// Search searches for similar documents using cosine similarity
func (db *InMemoryVectorDB) Search(ctx context.Context, queryVector Vector, limit int, threshold float64) ([]SearchResult, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var results []SearchResult

	for _, doc := range db.documents {
		similarity := cosineSimilarity(queryVector, doc.Vector)
		if similarity >= threshold {
			results = append(results, SearchResult{
				Document: *doc,
				Score:    similarity,
			})
		}
	}

	// Sort by similarity score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Delete removes a document
func (db *InMemoryVectorDB) Delete(ctx context.Context, id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.documents, id)
	return nil
}

// Get retrieves a document
func (db *InMemoryVectorDB) Get(ctx context.Context, id string) (*Document, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	doc, exists := db.documents[id]
	if !exists {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	// Return a copy
	docCopy := *doc
	return &docCopy, nil
}

// List returns all document IDs
func (db *InMemoryVectorDB) List(ctx context.Context) ([]string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	ids := make([]string, 0, len(db.documents))
	for id := range db.documents {
		ids = append(ids, id)
	}
	return ids, nil
}

// Close is a no-op for in-memory database
func (db *InMemoryVectorDB) Close() error {
	return nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b Vector) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// EmbeddingService interface for generating embeddings
type EmbeddingService interface {
	// GenerateEmbedding generates a vector embedding for text
	GenerateEmbedding(ctx context.Context, text string) (Vector, error)

	// GenerateEmbeddings generates embeddings for multiple texts
	GenerateEmbeddings(ctx context.Context, texts []string) ([]Vector, error)

	// GetDimension returns the dimension of embeddings
	GetDimension() int
}

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

// RAGService provides Retrieval-Augmented Generation capabilities
type RAGService struct {
	vectorDB            VectorDatabase
	embeddings          EmbeddingService
	contextMgr          *ctxt.ConversationManager
	maxResults          int
	similarityThreshold float64
}

// NewRAGService creates a new RAG service
func NewRAGService(vectorDB VectorDatabase, embeddings EmbeddingService, contextMgr *ctxt.ConversationManager) *RAGService {
	return &RAGService{
		vectorDB:            vectorDB,
		embeddings:          embeddings,
		contextMgr:          contextMgr,
		maxResults:          5,
		similarityThreshold: 0.1,
	}
}

// IndexMessage indexes a message for retrieval
func (rag *RAGService) IndexMessage(ctx context.Context, message *ctxt.Message) error {
	// Generate embedding for the message content
	vector, err := rag.embeddings.GenerateEmbedding(ctx, message.Content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Create document
	doc := &Document{
		ID:      fmt.Sprintf("msg_%s_%d", message.ID, message.Timestamp.Unix()),
		Content: message.Content,
		Metadata: map[string]interface{}{
			"role":       message.Role,
			"timestamp":  message.Timestamp,
			"message_id": message.ID,
		},
		Vector:    vector,
		Timestamp: message.Timestamp,
		Source:    "conversation",
	}

	// Store in vector database
	return rag.vectorDB.Store(ctx, doc)
}

// RetrieveContext retrieves relevant context for a query
func (rag *RAGService) RetrieveContext(ctx context.Context, query string, conversationID string) ([]Document, error) {
	// Generate embedding for the query
	queryVector, err := rag.embeddings.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search for similar documents
	results, err := rag.vectorDB.Search(ctx, queryVector, rag.maxResults, rag.similarityThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector database: %w", err)
	}

	// Also get recent conversation context
	var conversationContext []Document
	if conversationID != "" {
		if conv := rag.contextMgr.GetConversation(conversationID); conv != nil {
			recentMessages := conv.GetRecentMessages(10) // Last 10 messages

			for _, msg := range recentMessages {
				doc := Document{
					ID:      msg.ID,
					Content: msg.Content,
					Metadata: map[string]interface{}{
						"role":      msg.Role,
						"timestamp": msg.Timestamp,
					},
					Timestamp: msg.Timestamp,
					Source:    "conversation_recent",
				}
				conversationContext = append(conversationContext, doc)
			}
		}
	}

	// Combine and deduplicate results
	allDocs := make([]Document, 0)

	// Add search results
	for _, result := range results {
		allDocs = append(allDocs, result.Document)
	}

	// Add recent conversation context (avoid duplicates)
	messageIDs := make(map[string]bool)
	for _, doc := range allDocs {
		if msgID, ok := doc.Metadata["message_id"].(string); ok {
			messageIDs[msgID] = true
		}
	}

	for _, doc := range conversationContext {
		if msgID, ok := doc.Metadata["message_id"].(string); ok {
			if !messageIDs[msgID] {
				allDocs = append(allDocs, doc)
			}
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(allDocs, func(i, j int) bool {
		return allDocs[i].Timestamp.After(allDocs[j].Timestamp)
	})

	// Limit results
	if len(allDocs) > rag.maxResults {
		allDocs = allDocs[:rag.maxResults]
	}

	return allDocs, nil
}

// OptimizePrompt enhances a prompt with retrieved context
func (rag *RAGService) OptimizePrompt(ctx context.Context, originalPrompt string, conversationID string) (string, error) {
	// Retrieve relevant context
	contextDocs, err := rag.RetrieveContext(ctx, originalPrompt, conversationID)
	if err != nil {
		return originalPrompt, fmt.Errorf("failed to retrieve context: %w", err)
	}

	if len(contextDocs) == 0 {
		return originalPrompt, nil // No context to add
	}

	// Build enhanced prompt
	var contextText string
	for _, doc := range contextDocs {
		source := doc.Source
		if source == "conversation_recent" {
			source = "recent conversation"
		}
		contextText += fmt.Sprintf("[From %s]: %s\n", source, doc.Content)
	}

	enhancedPrompt := fmt.Sprintf(`Context information:
%s

Original request: %s

Please use the context above to provide a more informed and relevant response.`, contextText, originalPrompt)

	return enhancedPrompt, nil
}

// GetStats returns statistics about the RAG system
func (rag *RAGService) GetStats() map[string]interface{} {
	// This would return statistics about indexed documents, search performance, etc.
	return map[string]interface{}{
		"embedding_dimension":  rag.embeddings.GetDimension(),
		"max_results":          rag.maxResults,
		"similarity_threshold": rag.similarityThreshold,
	}
}
