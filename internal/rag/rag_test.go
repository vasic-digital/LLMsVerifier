package rag

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// DefaultSearchOptions
// ============================================================================

func TestDefaultSearchOptions(t *testing.T) {
	opts := DefaultSearchOptions()

	assert.Equal(t, 10, opts.TopK)
	assert.Equal(t, 0.0, opts.MinScore)
	assert.Equal(t, 0.7, opts.DenseWeight)
	assert.Equal(t, 0.3, opts.SparseWeight)
	assert.True(t, opts.UseReranker)
}

// ============================================================================
// tokenize
// ============================================================================

func TestTokenize_Basic(t *testing.T) {
	tokens := tokenize("Hello, World!")
	assert.Contains(t, tokens, "hello")
	assert.Contains(t, tokens, "world")
}

func TestTokenize_Empty(t *testing.T) {
	tokens := tokenize("")
	assert.Empty(t, tokens)
}

func TestTokenize_Punctuation(t *testing.T) {
	tokens := tokenize("foo. bar! baz?")
	assert.Contains(t, tokens, "foo")
	assert.Contains(t, tokens, "bar")
	assert.Contains(t, tokens, "baz")
}

// ============================================================================
// cosineSimilarity
// ============================================================================

func TestCosineSimilarity_Identical(t *testing.T) {
	score := cosineSimilarity("hello world", "hello world")
	assert.InDelta(t, 1.0, score, 0.01)
}

func TestCosineSimilarity_Unrelated(t *testing.T) {
	score := cosineSimilarity("hello world", "foo bar baz")
	assert.InDelta(t, 0.0, score, 0.01)
}

func TestCosineSimilarity_Partial(t *testing.T) {
	score := cosineSimilarity("hello world", "hello earth")
	assert.Greater(t, score, 0.0)
	assert.Less(t, score, 1.0)
}

func TestCosineSimilarity_Empty(t *testing.T) {
	score := cosineSimilarity("", "hello")
	assert.Equal(t, 0.0, score)
}

// ============================================================================
// QdrantDenseRetriever
// ============================================================================

func TestNewQdrantDenseRetriever(t *testing.T) {
	r := NewQdrantDenseRetriever("test-collection", nil, nil)
	assert.NotNil(t, r)
	assert.Equal(t, "test-collection", r.collection)
}

func TestQdrantDenseRetriever_Index(t *testing.T) {
	r := NewQdrantDenseRetriever("test", nil, nil)
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc-1", Content: "The quick brown fox"},
		{ID: "doc-2", Content: "jumped over the lazy dog"},
	}

	err := r.Index(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, r.docs, 2)
}

func TestQdrantDenseRetriever_Index_GeneratesID(t *testing.T) {
	r := NewQdrantDenseRetriever("test", nil, nil)
	ctx := context.Background()

	docs := []*Document{
		{Content: "Document without ID"},
	}

	err := r.Index(ctx, docs)
	require.NoError(t, err)
	assert.NotEmpty(t, docs[0].ID)
}

func TestQdrantDenseRetriever_Search(t *testing.T) {
	r := NewQdrantDenseRetriever("test", nil, nil)
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc-1", Content: "machine learning algorithms"},
		{ID: "doc-2", Content: "deep learning neural networks"},
		{ID: "doc-3", Content: "cooking recipes pasta"},
	}
	require.NoError(t, r.Index(ctx, docs))

	results, err := r.Search(ctx, "machine learning", nil)
	require.NoError(t, err)
	assert.True(t, len(results) >= 1)
}

func TestQdrantDenseRetriever_Search_LimitsTopK(t *testing.T) {
	r := NewQdrantDenseRetriever("test", nil, nil)
	ctx := context.Background()

	for i := 0; i < 20; i++ {
		docs := []*Document{{Content: "test document content here"}}
		require.NoError(t, r.Index(ctx, docs))
	}

	opts := &SearchOptions{TopK: 5, MinScore: 0.0}
	results, err := r.Search(ctx, "test document", opts)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 5)
}

func TestQdrantDenseRetriever_Search_MinScore(t *testing.T) {
	r := NewQdrantDenseRetriever("test", nil, nil)
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc-1", Content: "completely unrelated content xyz"},
	}
	require.NoError(t, r.Index(ctx, docs))

	opts := &SearchOptions{TopK: 10, MinScore: 1.0} // only exact matches
	results, err := r.Search(ctx, "nothing in common", opts)
	require.NoError(t, err)
	// Nothing should have score >= 1.0 (which would require identical strings)
	assert.Empty(t, results)
}

// ============================================================================
// BM25Index
// ============================================================================

func TestNewBM25Index(t *testing.T) {
	idx := NewBM25Index(nil)
	assert.NotNil(t, idx)
	assert.Equal(t, 1.5, idx.k1)
	assert.Equal(t, 0.75, idx.b)
}

func TestBM25Index_Index(t *testing.T) {
	idx := NewBM25Index(nil)
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc-1", Content: "information retrieval systems"},
		{ID: "doc-2", Content: "machine learning for search"},
	}

	err := idx.Index(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, idx.docs, 2)
	assert.NotZero(t, idx.avgDocLen)
}

func TestBM25Index_Index_GeneratesID(t *testing.T) {
	idx := NewBM25Index(nil)
	ctx := context.Background()

	docs := []*Document{{Content: "no id given"}}
	require.NoError(t, idx.Index(ctx, docs))
	assert.NotEmpty(t, docs[0].ID)
}

func TestBM25Index_Search(t *testing.T) {
	idx := NewBM25Index(nil)
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc-1", Content: "information retrieval and search"},
		{ID: "doc-2", Content: "machine learning algorithms"},
		{ID: "doc-3", Content: "information retrieval systems query"},
	}
	require.NoError(t, idx.Index(ctx, docs))

	results, err := idx.Search(ctx, "information retrieval", nil)
	require.NoError(t, err)
	assert.True(t, len(results) >= 1)
	// The first result should be one of the "information retrieval" docs
	assert.NotEmpty(t, results[0].ID)
}

func TestBM25Index_Search_Empty(t *testing.T) {
	idx := NewBM25Index(nil)
	ctx := context.Background()

	results, err := idx.Search(ctx, "anything", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestBM25Index_Search_LimitsTopK(t *testing.T) {
	idx := NewBM25Index(nil)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		docs := []*Document{{Content: "search retrieval systems"}}
		require.NoError(t, idx.Index(ctx, docs))
	}

	opts := &SearchOptions{TopK: 3, MinScore: 0.0}
	results, err := idx.Search(ctx, "search", opts)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 3)
}

// ============================================================================
// EnhancedQdrantRetriever
// ============================================================================

func TestNewEnhancedQdrantRetriever(t *testing.T) {
	r := NewEnhancedQdrantRetriever("test-collection", nil, nil)
	assert.NotNil(t, r)
	assert.NotNil(t, r.denseRetriever)
	assert.NotNil(t, r.sparseIndex)
	assert.Equal(t, FusionRRF, r.fusionMethod)
}

func TestEnhancedQdrantRetriever_SetReranker(t *testing.T) {
	r := NewEnhancedQdrantRetriever("test", nil, nil)

	reranker := &mockReranker{}
	r.SetReranker(reranker)

	r.mu.RLock()
	assert.Equal(t, reranker, r.reranker)
	r.mu.RUnlock()
}

func TestEnhancedQdrantRetriever_SetDebateEvaluator(t *testing.T) {
	r := NewEnhancedQdrantRetriever("test", nil, nil)

	eval := &mockDebateEvaluator{}
	r.SetDebateEvaluator(eval)

	r.mu.RLock()
	assert.Equal(t, eval, r.debateEvaluator)
	r.mu.RUnlock()
}

func TestEnhancedQdrantRetriever_Index(t *testing.T) {
	r := NewEnhancedQdrantRetriever("test", nil, nil)
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc-1", Content: "hybrid retrieval search"},
		{ID: "doc-2", Content: "vector database indexing"},
	}

	err := r.Index(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, r.denseRetriever.docs, 2)
}

func TestEnhancedQdrantRetriever_HybridRetrieve_RRF(t *testing.T) {
	r := NewEnhancedQdrantRetriever("test", nil, nil)
	r.fusionMethod = FusionRRF
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc-1", Content: "neural network deep learning"},
		{ID: "doc-2", Content: "machine learning algorithms data"},
		{ID: "doc-3", Content: "cooking pasta recipes"},
	}
	require.NoError(t, r.Index(ctx, docs))

	results, err := r.HybridRetrieve(ctx, "machine learning", nil)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestEnhancedQdrantRetriever_HybridRetrieve_Weighted(t *testing.T) {
	r := NewEnhancedQdrantRetriever("test", nil, nil)
	r.fusionMethod = FusionWeighted
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc-1", Content: "neural network deep learning"},
		{ID: "doc-2", Content: "machine learning algorithms"},
	}
	require.NoError(t, r.Index(ctx, docs))

	opts := &SearchOptions{TopK: 10, MinScore: 0.0, DenseWeight: 0.6, SparseWeight: 0.4}
	results, err := r.HybridRetrieve(ctx, "learning", opts)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestEnhancedQdrantRetriever_HybridRetrieve_WithReranker(t *testing.T) {
	r := NewEnhancedQdrantRetriever("test", nil, nil)
	reranker := &mockReranker{}
	r.SetReranker(reranker)
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc-1", Content: "information retrieval systems"},
	}
	require.NoError(t, r.Index(ctx, docs))

	opts := &SearchOptions{TopK: 10, MinScore: 0.0, UseReranker: true}
	results, err := r.HybridRetrieve(ctx, "information", opts)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestEnhancedQdrantRetriever_HybridRetrieve_LimitsTopK(t *testing.T) {
	r := NewEnhancedQdrantRetriever("test", nil, nil)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		docs := []*Document{{Content: "information retrieval data"}}
		require.NoError(t, r.Index(ctx, docs))
	}

	opts := &SearchOptions{TopK: 3, MinScore: 0.0}
	results, err := r.HybridRetrieve(ctx, "information", opts)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 3)
}

// ============================================================================
// QdrantRAGPipeline
// ============================================================================

func TestNewQdrantRAGPipeline(t *testing.T) {
	p := NewQdrantRAGPipeline("test-col", nil, nil)
	assert.NotNil(t, p)
	assert.NotNil(t, p.retriever)
}

func TestQdrantRAGPipeline_SetDebateEvaluator(t *testing.T) {
	p := NewQdrantRAGPipeline("test", nil, nil)
	eval := &mockDebateEvaluator{}
	p.SetDebateEvaluator(eval)
	assert.Equal(t, eval, p.debateEval)
}

func TestQdrantRAGPipeline_Index(t *testing.T) {
	p := NewQdrantRAGPipeline("test", nil, nil)
	ctx := context.Background()

	docs := []*Document{
		{ID: "d1", Content: "test document content"},
	}

	err := p.Index(ctx, docs)
	require.NoError(t, err)
}

func TestQdrantRAGPipeline_Retrieve(t *testing.T) {
	p := NewQdrantRAGPipeline("test", nil, nil)
	ctx := context.Background()

	docs := []*Document{
		{ID: "d1", Content: "test document content search retrieval"},
	}
	require.NoError(t, p.Index(ctx, docs))

	results, err := p.Retrieve(ctx, "test document", nil)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

// ============================================================================
// Mock implementations
// ============================================================================

type mockReranker struct{}

func (m *mockReranker) Rerank(ctx context.Context, query string, results []*SearchResult) ([]*SearchResult, error) {
	return results, nil
}

type mockDebateEvaluator struct{}

func (m *mockDebateEvaluator) EvaluateRelevance(ctx context.Context, query, content string) (float64, error) {
	return 0.8, nil
}

// ============================================================================
// FusionMethod constants
// ============================================================================

func TestFusionMethodConstants(t *testing.T) {
	assert.Equal(t, FusionMethod("rrf"), FusionRRF)
	assert.Equal(t, FusionMethod("weighted"), FusionWeighted)
}
