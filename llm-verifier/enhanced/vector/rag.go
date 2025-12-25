package vector

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
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
	Source   string   `json:"source"`
	Rank     int      `json:"rank"`
}

// RetrievalStrategy defines different retrieval approaches
type RetrievalStrategy interface {
	Retrieve(ctx context.Context, query string, options RetrievalOptions) ([]SearchResult, error)
}

// QueryExpander expands queries for better retrieval
type QueryExpander interface {
	ExpandQuery(query string) []string
}

// ContextRanker ranks and filters retrieved context
type ContextRanker interface {
	RankContext(results []SearchResult, query string) []SearchResult
}

// RetrievalOptions configures retrieval behavior
type RetrievalOptions struct {
	MaxResults          int        `json:"max_results"`
	SimilarityThreshold float64    `json:"similarity_threshold"`
	UseHybridSearch     bool       `json:"use_hybrid_search"`
	Sources             []string   `json:"sources"`
	TimeFilter          *TimeRange `json:"time_filter,omitempty"`
}

// TimeRange represents a time range for filtering
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
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
	enableHybridSearch  bool
	retrievalStrategies []RetrievalStrategy
	queryExpander       QueryExpander
	contextRanker       ContextRanker
}

// NewRAGService creates a new RAG service
func NewRAGService(vectorDB VectorDatabase, embeddings EmbeddingService, contextMgr *ctxt.ConversationManager) *RAGService {
	return &RAGService{
		vectorDB:            vectorDB,
		embeddings:          embeddings,
		contextMgr:          contextMgr,
		maxResults:          5,
		similarityThreshold: 0.1,
		enableHybridSearch:  true,
		retrievalStrategies: []RetrievalStrategy{
			NewVectorRetrievalStrategy(vectorDB, embeddings),
			NewKeywordRetrievalStrategy(),
			NewSemanticRetrievalStrategy(embeddings),
		},
		queryExpander: NewBasicQueryExpander(),
		contextRanker: NewRelevanceRanker(),
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

// RetrieveContextAdvanced performs advanced multi-source context retrieval
func (rag *RAGService) RetrieveContextAdvanced(ctx context.Context, query string, conversationID string, options RetrievalOptions) ([]Document, error) {
	if options.MaxResults == 0 {
		options.MaxResults = rag.maxResults
	}
	if options.SimilarityThreshold == 0 {
		options.SimilarityThreshold = rag.similarityThreshold
	}

	var allResults []SearchResult

	// Use multiple retrieval strategies
	for _, strategy := range rag.retrievalStrategies {
		results, err := strategy.Retrieve(ctx, query, options)
		if err != nil {
			continue // Skip failed strategies
		}
		allResults = append(allResults, results...)
	}

	// Expand query for better retrieval
	if rag.queryExpander != nil {
		expandedQueries := rag.queryExpander.ExpandQuery(query)
		for _, expandedQuery := range expandedQueries {
			for _, strategy := range rag.retrievalStrategies {
				results, err := strategy.Retrieve(ctx, expandedQuery, options)
				if err != nil {
					continue
				}
				// Mark as expanded results with lower weight
				for i := range results {
					results[i].Score *= 0.8
				}
				allResults = append(allResults, results...)
			}
		}
	}

	// Rank and filter results
	if rag.contextRanker != nil {
		allResults = rag.contextRanker.RankContext(allResults, query)
	}

	// Deduplicate and limit results
	seen := make(map[string]bool)
	var finalResults []Document

	for _, result := range allResults {
		if len(finalResults) >= options.MaxResults {
			break
		}

		docID := result.Document.ID
		if !seen[docID] {
			seen[docID] = true
			finalResults = append(finalResults, result.Document)
		}
	}

	return finalResults, nil
}

// VectorRetrievalStrategy implements vector-based retrieval
type VectorRetrievalStrategy struct {
	vectorDB   VectorDatabase
	embeddings EmbeddingService
}

func NewVectorRetrievalStrategy(vectorDB VectorDatabase, embeddings EmbeddingService) *VectorRetrievalStrategy {
	return &VectorRetrievalStrategy{
		vectorDB:   vectorDB,
		embeddings: embeddings,
	}
}

func (vrs *VectorRetrievalStrategy) Retrieve(ctx context.Context, query string, options RetrievalOptions) ([]SearchResult, error) {
	// Generate embedding for query
	vector, err := vrs.embeddings.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}

	// Search vector database
	results, err := vrs.vectorDB.Search(ctx, vector, options.MaxResults*2, options.SimilarityThreshold)
	if err != nil {
		return nil, err
	}

	// Convert to SearchResult format
	searchResults := make([]SearchResult, len(results))
	for i, result := range results {
		searchResults[i] = SearchResult{
			Document: result.Document,
			Score:    result.Score,
			Source:   "vector_search",
			Rank:     i + 1,
		}
	}

	return searchResults, nil
}

// KeywordRetrievalStrategy implements keyword-based retrieval
type KeywordRetrievalStrategy struct{}

func NewKeywordRetrievalStrategy() *KeywordRetrievalStrategy {
	return &KeywordRetrievalStrategy{}
}

func (krs *KeywordRetrievalStrategy) Retrieve(ctx context.Context, query string, options RetrievalOptions) ([]SearchResult, error) {
	// Simple keyword matching - in production, this would use a text search index
	// For now, return empty results as this is a placeholder
	return []SearchResult{}, nil
}

// SemanticRetrievalStrategy implements semantic search
type SemanticRetrievalStrategy struct {
	embeddings EmbeddingService
}

func NewSemanticRetrievalStrategy(embeddings EmbeddingService) *SemanticRetrievalStrategy {
	return &SemanticRetrievalStrategy{embeddings: embeddings}
}

func (srs *SemanticRetrievalStrategy) Retrieve(ctx context.Context, query string, options RetrievalOptions) ([]SearchResult, error) {
	// This could implement more advanced semantic search
	// For now, delegate to vector search
	return []SearchResult{}, nil
}

// BasicQueryExpander implements simple query expansion
type BasicQueryExpander struct{}

func NewBasicQueryExpander() *BasicQueryExpander {
	return &BasicQueryExpander{}
}

func (bqe *BasicQueryExpander) ExpandQuery(query string) []string {
	// Simple query expansion - add synonyms and related terms
	expansions := []string{query}

	// Add some basic expansions
	if strings.Contains(strings.ToLower(query), "model") {
		expansions = append(expansions, strings.ReplaceAll(query, "model", "AI model"))
		expansions = append(expansions, strings.ReplaceAll(query, "model", "language model"))
	}

	return expansions
}

// RelevanceRanker ranks results by relevance
type RelevanceRanker struct{}

func NewRelevanceRanker() *RelevanceRanker {
	return &RelevanceRanker{}
}

func (rr *RelevanceRanker) RankContext(results []SearchResult, query string) []SearchResult {
	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	maxResults := 10
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
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
