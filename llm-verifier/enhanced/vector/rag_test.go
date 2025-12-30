package vector

import (
	"context"
	"testing"
	"time"

	ctxt "llm-verifier/enhanced/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== InMemoryVectorDB Tests ====================

func TestNewInMemoryVectorDB(t *testing.T) {
	db := NewInMemoryVectorDB()
	assert.NotNil(t, db)
	assert.NotNil(t, db.documents)
}

func TestInMemoryVectorDB_Store(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	doc := &Document{
		ID:      "doc-1",
		Content: "Test content",
		Vector:  Vector{0.1, 0.2, 0.3},
	}

	err := db.Store(ctx, doc)
	assert.NoError(t, err)

	// Verify timestamp was set
	assert.False(t, doc.Timestamp.IsZero())
}

func TestInMemoryVectorDB_Get(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	doc := &Document{
		ID:      "doc-1",
		Content: "Test content",
		Vector:  Vector{0.1, 0.2, 0.3},
	}
	db.Store(ctx, doc)

	// Get existing document
	retrieved, err := db.Get(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, doc.ID, retrieved.ID)
	assert.Equal(t, doc.Content, retrieved.Content)

	// Get non-existent document
	_, err = db.Get(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestInMemoryVectorDB_Delete(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	doc := &Document{
		ID:      "doc-1",
		Content: "Test content",
		Vector:  Vector{0.1, 0.2, 0.3},
	}
	db.Store(ctx, doc)

	err := db.Delete(ctx, "doc-1")
	assert.NoError(t, err)

	// Verify deleted
	_, err = db.Get(ctx, "doc-1")
	assert.Error(t, err)
}

func TestInMemoryVectorDB_List(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	db.Store(ctx, &Document{ID: "doc-1", Vector: Vector{0.1}})
	db.Store(ctx, &Document{ID: "doc-2", Vector: Vector{0.2}})
	db.Store(ctx, &Document{ID: "doc-3", Vector: Vector{0.3}})

	ids, err := db.List(ctx)
	require.NoError(t, err)
	assert.Len(t, ids, 3)
}

func TestInMemoryVectorDB_Search(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	// Store documents with different vectors
	db.Store(ctx, &Document{ID: "doc-1", Content: "Similar", Vector: Vector{0.9, 0.1, 0.1}})
	db.Store(ctx, &Document{ID: "doc-2", Content: "Different", Vector: Vector{0.1, 0.9, 0.1}})
	db.Store(ctx, &Document{ID: "doc-3", Content: "Very Similar", Vector: Vector{0.95, 0.1, 0.05}})

	// Search for similar documents
	query := Vector{1.0, 0.1, 0.1}
	results, err := db.Search(ctx, query, 10, 0.0)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// First result should have highest similarity
	assert.True(t, results[0].Score >= results[1].Score)
}

func TestInMemoryVectorDB_SearchWithThreshold(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	db.Store(ctx, &Document{ID: "doc-1", Vector: Vector{0.9, 0.1}})
	db.Store(ctx, &Document{ID: "doc-2", Vector: Vector{0.1, 0.9}})

	// High threshold - should exclude dissimilar
	query := Vector{1.0, 0.0}
	results, err := db.Search(ctx, query, 10, 0.9)
	require.NoError(t, err)

	// Only similar documents should be returned
	for _, r := range results {
		assert.GreaterOrEqual(t, r.Score, 0.9)
	}
}

func TestInMemoryVectorDB_SearchWithLimit(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		db.Store(ctx, &Document{ID: string(rune('a'+i)), Vector: Vector{float64(i) / 10}})
	}

	query := Vector{0.5}
	results, err := db.Search(ctx, query, 3, 0.0)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 3)
}

func TestInMemoryVectorDB_Close(t *testing.T) {
	db := NewInMemoryVectorDB()
	err := db.Close()
	assert.NoError(t, err)
}

// ==================== Cosine Similarity Tests ====================

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        Vector
		b        Vector
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        Vector{1, 0, 0},
			b:        Vector{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        Vector{1, 0},
			b:        Vector{0, 1},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        Vector{1, 0},
			b:        Vector{-1, 0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        Vector{1, 0.1},
			b:        Vector{1, 0},
			expected: 0.9950, // Approximately
		},
		{
			name:     "different lengths",
			a:        Vector{1, 2, 3},
			b:        Vector{1, 2},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			a:        Vector{0, 0, 0},
			b:        Vector{1, 2, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

// ==================== MockEmbeddingService Tests ====================

func TestMockEmbeddingService(t *testing.T) {
	svc := NewMockEmbeddingService(128)
	assert.NotNil(t, svc)
	assert.Equal(t, 128, svc.GetDimension())
}

func TestMockEmbeddingService_GenerateEmbedding(t *testing.T) {
	svc := NewMockEmbeddingService(64)
	ctx := context.Background()

	vec, err := svc.GenerateEmbedding(ctx, "test text")
	require.NoError(t, err)
	assert.Len(t, vec, 64)

	// Same text should produce same embedding (deterministic)
	vec2, err := svc.GenerateEmbedding(ctx, "test text")
	require.NoError(t, err)
	assert.Equal(t, vec, vec2)

	// Different text should produce different embedding
	vec3, err := svc.GenerateEmbedding(ctx, "different text")
	require.NoError(t, err)
	assert.NotEqual(t, vec, vec3)
}

func TestMockEmbeddingService_GenerateEmbeddings(t *testing.T) {
	svc := NewMockEmbeddingService(32)
	ctx := context.Background()

	texts := []string{"text1", "text2", "text3"}
	vectors, err := svc.GenerateEmbeddings(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, vectors, 3)

	for i, vec := range vectors {
		assert.Len(t, vec, 32)
		// Each text produces unique embedding
		for j := i + 1; j < len(vectors); j++ {
			assert.NotEqual(t, vec, vectors[j])
		}
	}
}

// ==================== RAGService Tests ====================

func TestNewRAGService(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(128)

	rag := NewRAGService(db, embeddings, nil)
	assert.NotNil(t, rag)
	assert.Equal(t, 5, rag.maxResults)
	assert.Equal(t, 0.1, rag.similarityThreshold)
	assert.True(t, rag.enableHybridSearch)
}

func TestRAGService_GetStats(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(128)
	rag := NewRAGService(db, embeddings, nil)

	stats := rag.GetStats()
	assert.Equal(t, 128, stats["embedding_dimension"])
	assert.Equal(t, 5, stats["max_results"])
	assert.Equal(t, 0.1, stats["similarity_threshold"])
}

// ==================== Retrieval Strategy Tests ====================

func TestVectorRetrievalStrategy(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)
	strategy := NewVectorRetrievalStrategy(db, embeddings)

	ctx := context.Background()

	// Store some documents
	vec1, _ := embeddings.GenerateEmbedding(ctx, "machine learning")
	db.Store(ctx, &Document{ID: "doc-1", Content: "machine learning", Vector: vec1})

	vec2, _ := embeddings.GenerateEmbedding(ctx, "deep learning")
	db.Store(ctx, &Document{ID: "doc-2", Content: "deep learning", Vector: vec2})

	// Search
	options := RetrievalOptions{MaxResults: 10, SimilarityThreshold: 0.0}
	results, err := strategy.Retrieve(ctx, "machine learning", options)
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestKeywordRetrievalStrategy(t *testing.T) {
	strategy := NewKeywordRetrievalStrategy()
	ctx := context.Background()

	options := RetrievalOptions{MaxResults: 10}
	results, err := strategy.Retrieve(ctx, "test query", options)
	require.NoError(t, err)
	// Keyword strategy returns empty (placeholder)
	assert.Empty(t, results)
}

func TestSemanticRetrievalStrategy(t *testing.T) {
	embeddings := NewMockEmbeddingService(64)
	strategy := NewSemanticRetrievalStrategy(embeddings)
	ctx := context.Background()

	options := RetrievalOptions{MaxResults: 10}
	results, err := strategy.Retrieve(ctx, "test query", options)
	require.NoError(t, err)
	// Semantic strategy returns empty (placeholder)
	assert.Empty(t, results)
}

// ==================== Query Expander Tests ====================

func TestBasicQueryExpander(t *testing.T) {
	expander := NewBasicQueryExpander()

	// Regular query
	expansions := expander.ExpandQuery("hello world")
	assert.Contains(t, expansions, "hello world")

	// Query with "model" keyword
	expansions = expander.ExpandQuery("AI model performance")
	assert.GreaterOrEqual(t, len(expansions), 2)
}

// ==================== Relevance Ranker Tests ====================

func TestRelevanceRanker(t *testing.T) {
	ranker := NewRelevanceRanker()

	results := []SearchResult{
		{Document: Document{ID: "a"}, Score: 0.5},
		{Document: Document{ID: "b"}, Score: 0.9},
		{Document: Document{ID: "c"}, Score: 0.7},
	}

	ranked := ranker.RankContext(results, "query")

	// Should be sorted by score descending
	assert.Equal(t, "b", ranked[0].Document.ID)
	assert.Equal(t, "c", ranked[1].Document.ID)
	assert.Equal(t, "a", ranked[2].Document.ID)
}

func TestRelevanceRanker_Limit(t *testing.T) {
	ranker := NewRelevanceRanker()

	var results []SearchResult
	for i := 0; i < 20; i++ {
		results = append(results, SearchResult{
			Document: Document{ID: string(rune('a' + i))},
			Score:    float64(i) / 20,
		})
	}

	ranked := ranker.RankContext(results, "query")
	assert.LessOrEqual(t, len(ranked), 10)
}

// ==================== Struct Tests ====================

func TestDocument_Struct(t *testing.T) {
	now := time.Now()
	doc := Document{
		ID:      "doc-1",
		Content: "Test content",
		Metadata: map[string]interface{}{
			"key": "value",
		},
		Vector:    Vector{0.1, 0.2, 0.3},
		Timestamp: now,
		Source:    "test",
	}

	assert.Equal(t, "doc-1", doc.ID)
	assert.Equal(t, "Test content", doc.Content)
	assert.Equal(t, "value", doc.Metadata["key"])
	assert.Len(t, doc.Vector, 3)
}

func TestSearchResult_Struct(t *testing.T) {
	result := SearchResult{
		Document: Document{ID: "doc-1"},
		Score:    0.95,
		Source:   "vector_search",
		Rank:     1,
	}

	assert.Equal(t, "doc-1", result.Document.ID)
	assert.Equal(t, 0.95, result.Score)
	assert.Equal(t, 1, result.Rank)
}

func TestRetrievalOptions_Struct(t *testing.T) {
	now := time.Now()
	options := RetrievalOptions{
		MaxResults:          10,
		SimilarityThreshold: 0.8,
		UseHybridSearch:     true,
		Sources:             []string{"source1", "source2"},
		TimeFilter: &TimeRange{
			Start: now.Add(-24 * time.Hour),
			End:   now,
		},
	}

	assert.Equal(t, 10, options.MaxResults)
	assert.Equal(t, 0.8, options.SimilarityThreshold)
	assert.True(t, options.UseHybridSearch)
	assert.NotNil(t, options.TimeFilter)
}

// ==================== Interface Compliance Tests ====================

func TestVectorDatabaseInterface(t *testing.T) {
	var _ VectorDatabase = (*InMemoryVectorDB)(nil)
}

func TestEmbeddingServiceInterface(t *testing.T) {
	var _ EmbeddingService = (*MockEmbeddingService)(nil)
}

func TestRetrievalStrategyInterface(t *testing.T) {
	var _ RetrievalStrategy = (*VectorRetrievalStrategy)(nil)
	var _ RetrievalStrategy = (*KeywordRetrievalStrategy)(nil)
	var _ RetrievalStrategy = (*SemanticRetrievalStrategy)(nil)
}

func TestQueryExpanderInterface(t *testing.T) {
	var _ QueryExpander = (*BasicQueryExpander)(nil)
}

func TestContextRankerInterface(t *testing.T) {
	var _ ContextRanker = (*RelevanceRanker)(nil)
}

// ==================== RAGService Advanced Tests ====================

func TestRAGService_IndexMessage(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)
	rag := NewRAGService(db, embeddings, nil)
	ctx := context.Background()

	msg := &ctxt.Message{
		ID:        "msg-1",
		Role:      "user",
		Content:   "Test message content for indexing",
		Timestamp: time.Now(),
	}

	err := rag.IndexMessage(ctx, msg)
	require.NoError(t, err)

	// Verify message was indexed
	ids, err := db.List(ctx)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
}

func TestRAGService_IndexMultipleMessages(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)
	rag := NewRAGService(db, embeddings, nil)
	ctx := context.Background()

	messages := []*ctxt.Message{
		{ID: "msg-1", Role: "user", Content: "First message", Timestamp: time.Now()},
		{ID: "msg-2", Role: "assistant", Content: "Second message", Timestamp: time.Now().Add(time.Second)},
		{ID: "msg-3", Role: "user", Content: "Third message", Timestamp: time.Now().Add(2 * time.Second)},
	}

	for _, msg := range messages {
		err := rag.IndexMessage(ctx, msg)
		require.NoError(t, err)
	}

	ids, err := db.List(ctx)
	require.NoError(t, err)
	assert.Len(t, ids, 3)
}

func TestRAGService_RetrieveContext(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)
	rag := NewRAGService(db, embeddings, nil)
	ctx := context.Background()

	// Index some messages
	messages := []*ctxt.Message{
		{ID: "msg-1", Role: "user", Content: "machine learning concepts", Timestamp: time.Now()},
		{ID: "msg-2", Role: "assistant", Content: "deep learning neural networks", Timestamp: time.Now().Add(time.Second)},
		{ID: "msg-3", Role: "user", Content: "cooking recipes for dinner", Timestamp: time.Now().Add(2 * time.Second)},
	}

	for _, msg := range messages {
		rag.IndexMessage(ctx, msg)
	}

	// Retrieve context without conversation ID
	docs, err := rag.RetrieveContext(ctx, "machine learning", "")
	require.NoError(t, err)
	assert.NotEmpty(t, docs)
}

func TestRAGService_RetrieveContextEmpty(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)
	rag := NewRAGService(db, embeddings, nil)
	ctx := context.Background()

	// Retrieve from empty database
	docs, err := rag.RetrieveContext(ctx, "query", "")
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestRAGService_OptimizePrompt(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)
	rag := NewRAGService(db, embeddings, nil)
	ctx := context.Background()

	// Index some messages first
	msg := &ctxt.Message{
		ID:        "msg-1",
		Role:      "user",
		Content:   "Previous conversation about AI",
		Timestamp: time.Now(),
	}
	rag.IndexMessage(ctx, msg)

	// Optimize prompt
	originalPrompt := "Tell me about AI"
	optimized, err := rag.OptimizePrompt(ctx, originalPrompt, "")
	require.NoError(t, err)

	// Should contain context information
	assert.Contains(t, optimized, "Context information")
	assert.Contains(t, optimized, originalPrompt)
}

func TestRAGService_OptimizePromptNoContext(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)
	rag := NewRAGService(db, embeddings, nil)
	ctx := context.Background()

	// Optimize prompt with no indexed data
	originalPrompt := "Tell me about AI"
	optimized, err := rag.OptimizePrompt(ctx, originalPrompt, "")
	require.NoError(t, err)

	// Should return original prompt unchanged
	assert.Equal(t, originalPrompt, optimized)
}

func TestRAGService_RetrieveContextAdvanced(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)

	// Create strategies
	vectorStrategy := NewVectorRetrievalStrategy(db, embeddings)
	expander := NewBasicQueryExpander()
	ranker := NewRelevanceRanker()

	rag := NewRAGService(db, embeddings, nil)
	rag.retrievalStrategies = append(rag.retrievalStrategies, vectorStrategy)
	rag.queryExpander = expander
	rag.contextRanker = ranker

	ctx := context.Background()

	// Index some documents
	messages := []*ctxt.Message{
		{ID: "msg-1", Role: "user", Content: "AI model training", Timestamp: time.Now()},
		{ID: "msg-2", Role: "assistant", Content: "machine learning optimization", Timestamp: time.Now().Add(time.Second)},
	}

	for _, msg := range messages {
		rag.IndexMessage(ctx, msg)
	}

	// Advanced retrieval
	options := RetrievalOptions{
		MaxResults:          10,
		SimilarityThreshold: 0.0,
	}
	docs, err := rag.RetrieveContextAdvanced(ctx, "AI training", "", options)
	require.NoError(t, err)
	assert.NotNil(t, docs)
}

func TestRAGService_RetrieveContextAdvancedDefaultOptions(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)
	rag := NewRAGService(db, embeddings, nil)
	ctx := context.Background()

	// Use zero values - should use defaults
	options := RetrievalOptions{}
	docs, err := rag.RetrieveContextAdvanced(ctx, "query", "", options)
	require.NoError(t, err)
	// Empty database returns empty slice (may be nil or empty)
	assert.Empty(t, docs)
}

func TestRAGService_WithStrategies(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)

	rag := NewRAGService(db, embeddings, nil)

	// Add multiple strategies
	rag.retrievalStrategies = []RetrievalStrategy{
		NewVectorRetrievalStrategy(db, embeddings),
		NewKeywordRetrievalStrategy(),
		NewSemanticRetrievalStrategy(embeddings),
	}

	assert.Len(t, rag.retrievalStrategies, 3)
}

func TestRAGService_WithQueryExpander(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)

	rag := NewRAGService(db, embeddings, nil)
	rag.queryExpander = NewBasicQueryExpander()

	assert.NotNil(t, rag.queryExpander)
}

func TestRAGService_WithContextRanker(t *testing.T) {
	db := NewInMemoryVectorDB()
	embeddings := NewMockEmbeddingService(64)

	rag := NewRAGService(db, embeddings, nil)
	rag.contextRanker = NewRelevanceRanker()

	assert.NotNil(t, rag.contextRanker)
}

// ==================== InMemoryVectorDB Edge Cases ====================

func TestInMemoryVectorDB_StoreOverwrite(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	// Store a document
	doc := &Document{ID: "doc-1", Content: "Original", Vector: Vector{0.1, 0.2}}
	db.Store(ctx, doc)

	// Overwrite with same ID
	doc2 := &Document{ID: "doc-1", Content: "Updated", Vector: Vector{0.3, 0.4}}
	err := db.Store(ctx, doc2)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := db.Get(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated", retrieved.Content)
}

func TestInMemoryVectorDB_SearchEmptyDB(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	results, err := db.Search(ctx, Vector{0.1, 0.2, 0.3}, 10, 0.0)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestInMemoryVectorDB_SearchSingleDocument(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	db.Store(ctx, &Document{ID: "only-one", Content: "Only doc", Vector: Vector{1.0, 0.0, 0.0}})

	results, err := db.Search(ctx, Vector{1.0, 0.0, 0.0}, 10, 0.0)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "only-one", results[0].Document.ID)
}

func TestInMemoryVectorDB_DeleteNonExistent(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	// Delete non-existent should not error
	err := db.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestInMemoryVectorDB_ListEmpty(t *testing.T) {
	db := NewInMemoryVectorDB()
	ctx := context.Background()

	ids, err := db.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

// ==================== MockEmbeddingService Edge Cases ====================

func TestMockEmbeddingService_EmptyText(t *testing.T) {
	svc := NewMockEmbeddingService(32)
	ctx := context.Background()

	vec, err := svc.GenerateEmbedding(ctx, "")
	require.NoError(t, err)
	assert.Len(t, vec, 32)
}

func TestMockEmbeddingService_LongText(t *testing.T) {
	svc := NewMockEmbeddingService(64)
	ctx := context.Background()

	longText := ""
	for i := 0; i < 1000; i++ {
		longText += "word "
	}

	vec, err := svc.GenerateEmbedding(ctx, longText)
	require.NoError(t, err)
	assert.Len(t, vec, 64)
}

func TestMockEmbeddingService_BatchEmpty(t *testing.T) {
	svc := NewMockEmbeddingService(32)
	ctx := context.Background()

	vectors, err := svc.GenerateEmbeddings(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, vectors)
}

// ==================== Cosine Similarity Edge Cases ====================

func TestCosineSimilarity_EmptyVectors(t *testing.T) {
	result := cosineSimilarity(Vector{}, Vector{})
	assert.Equal(t, 0.0, result)
}

func TestCosineSimilarity_SingleElement(t *testing.T) {
	result := cosineSimilarity(Vector{1.0}, Vector{1.0})
	assert.InDelta(t, 1.0, result, 0.01)
}

func TestCosineSimilarity_LargeVectors(t *testing.T) {
	// Create large vectors
	size := 1000
	a := make(Vector, size)
	b := make(Vector, size)
	for i := 0; i < size; i++ {
		a[i] = float64(i) / float64(size)
		b[i] = float64(i) / float64(size)
	}

	result := cosineSimilarity(a, b)
	assert.InDelta(t, 1.0, result, 0.01) // Identical vectors
}
