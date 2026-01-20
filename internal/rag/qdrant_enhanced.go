package rag

import (
	"context"
	"log"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// SearchResult represents a search result
type SearchResult struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Score      float64                `json:"score"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Source     string                 `json:"source,omitempty"` // "dense", "sparse", "hybrid"
}

// SearchOptions configures search behavior
type SearchOptions struct {
	TopK          int     `json:"top_k"`
	MinScore      float64 `json:"min_score"`
	DenseWeight   float64 `json:"dense_weight"`
	SparseWeight  float64 `json:"sparse_weight"`
	UseReranker   bool    `json:"use_reranker"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
}

// DefaultSearchOptions returns default options
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		TopK:         10,
		MinScore:     0.0,
		DenseWeight:  0.7,
		SparseWeight: 0.3,
		UseReranker:  true,
	}
}

// FusionMethod defines how to combine results
type FusionMethod string

const (
	FusionRRF      FusionMethod = "rrf"      // Reciprocal Rank Fusion
	FusionWeighted FusionMethod = "weighted" // Weighted combination
)

// Document represents a document to index
type Document struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Vector   []float32              `json:"vector,omitempty"`
}

// Reranker interface for reranking results
type Reranker interface {
	Rerank(ctx context.Context, query string, results []*SearchResult) ([]*SearchResult, error)
}

// DebateEvaluator interface for debate-based evaluation
type DebateEvaluator interface {
	EvaluateRelevance(ctx context.Context, query, content string) (float64, error)
}

// EmbeddingProvider generates embeddings
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// QdrantDenseRetriever handles dense vector retrieval
type QdrantDenseRetriever struct {
	collection string
	embedder   EmbeddingProvider
	logger     *log.Logger
	mu         sync.RWMutex
	docs       map[string]*Document // In-memory storage for demo
}

// NewQdrantDenseRetriever creates a new dense retriever
func NewQdrantDenseRetriever(collection string, embedder EmbeddingProvider, logger *log.Logger) *QdrantDenseRetriever {
	if logger == nil {
		logger = log.Default()
	}
	return &QdrantDenseRetriever{
		collection: collection,
		embedder:   embedder,
		logger:     logger,
		docs:       make(map[string]*Document),
	}
}

// Index indexes documents
func (r *QdrantDenseRetriever) Index(ctx context.Context, docs []*Document) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, doc := range docs {
		if doc.ID == "" {
			doc.ID = uuid.New().String()
		}
		r.docs[doc.ID] = doc
	}
	return nil
}

// Search performs dense search
func (r *QdrantDenseRetriever) Search(ctx context.Context, query string, opts *SearchOptions) ([]*SearchResult, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*SearchResult
	for _, doc := range r.docs {
		// Simulate similarity score
		score := cosineSimilarity(query, doc.Content)
		if score >= opts.MinScore {
			results = append(results, &SearchResult{
				ID:       doc.ID,
				Content:  doc.Content,
				Score:    score,
				Metadata: doc.Metadata,
				Source:   "dense",
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > opts.TopK {
		results = results[:opts.TopK]
	}

	return results, nil
}

// BM25Index implements BM25 sparse retrieval
type BM25Index struct {
	mu         sync.RWMutex
	docs       map[string]*Document
	termFreqs  map[string]map[string]float64 // docID -> term -> freq
	docFreqs   map[string]int                // term -> doc count
	docLengths map[string]int                // docID -> length
	avgDocLen  float64
	k1         float64
	b          float64
	logger     *log.Logger
}

// NewBM25Index creates a new BM25 index
func NewBM25Index(logger *log.Logger) *BM25Index {
	if logger == nil {
		logger = log.Default()
	}
	return &BM25Index{
		docs:       make(map[string]*Document),
		termFreqs:  make(map[string]map[string]float64),
		docFreqs:   make(map[string]int),
		docLengths: make(map[string]int),
		k1:         1.5,
		b:          0.75,
		logger:     logger,
	}
}

// Index indexes documents for BM25
func (idx *BM25Index) Index(ctx context.Context, docs []*Document) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	var totalLen int
	for _, doc := range docs {
		if doc.ID == "" {
			doc.ID = uuid.New().String()
		}
		idx.docs[doc.ID] = doc

		terms := tokenize(doc.Content)
		idx.docLengths[doc.ID] = len(terms)
		totalLen += len(terms)

		// Calculate term frequencies
		termCounts := make(map[string]float64)
		for _, term := range terms {
			termCounts[term]++
		}
		idx.termFreqs[doc.ID] = termCounts

		// Update document frequencies
		seen := make(map[string]bool)
		for _, term := range terms {
			if !seen[term] {
				idx.docFreqs[term]++
				seen[term] = true
			}
		}
	}

	if len(idx.docs) > 0 {
		idx.avgDocLen = float64(totalLen) / float64(len(idx.docs))
	}

	return nil
}

// Search performs BM25 search
func (idx *BM25Index) Search(ctx context.Context, query string, opts *SearchOptions) ([]*SearchResult, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	queryTerms := tokenize(query)
	N := float64(len(idx.docs))

	var results []*SearchResult
	for docID, doc := range idx.docs {
		score := 0.0
		for _, term := range queryTerms {
			tf := idx.termFreqs[docID][term]
			df := float64(idx.docFreqs[term])
			if df == 0 {
				continue
			}

			idf := math.Log((N - df + 0.5) / (df + 0.5))
			docLen := float64(idx.docLengths[docID])
			tfNorm := (tf * (idx.k1 + 1)) / (tf + idx.k1*(1-idx.b+idx.b*docLen/idx.avgDocLen))
			score += idf * tfNorm
		}

		if score >= opts.MinScore {
			results = append(results, &SearchResult{
				ID:       docID,
				Content:  doc.Content,
				Score:    score,
				Metadata: doc.Metadata,
				Source:   "sparse",
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > opts.TopK {
		results = results[:opts.TopK]
	}

	return results, nil
}

// EnhancedQdrantRetriever provides hybrid retrieval
type EnhancedQdrantRetriever struct {
	denseRetriever  *QdrantDenseRetriever
	sparseIndex     *BM25Index
	reranker        Reranker
	debateEvaluator DebateEvaluator
	fusionMethod    FusionMethod
	logger          *log.Logger
	mu              sync.RWMutex
}

// NewEnhancedQdrantRetriever creates a new enhanced retriever
func NewEnhancedQdrantRetriever(collection string, embedder EmbeddingProvider, logger *log.Logger) *EnhancedQdrantRetriever {
	if logger == nil {
		logger = log.Default()
	}
	return &EnhancedQdrantRetriever{
		denseRetriever: NewQdrantDenseRetriever(collection, embedder, logger),
		sparseIndex:    NewBM25Index(logger),
		fusionMethod:   FusionRRF,
		logger:         logger,
	}
}

// SetReranker sets the reranker
func (r *EnhancedQdrantRetriever) SetReranker(reranker Reranker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reranker = reranker
}

// SetDebateEvaluator sets the debate evaluator
func (r *EnhancedQdrantRetriever) SetDebateEvaluator(eval DebateEvaluator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.debateEvaluator = eval
}

// Index indexes documents for hybrid search
func (r *EnhancedQdrantRetriever) Index(ctx context.Context, docs []*Document) error {
	if err := r.denseRetriever.Index(ctx, docs); err != nil {
		return err
	}
	return r.sparseIndex.Index(ctx, docs)
}

// HybridRetrieve performs hybrid retrieval
func (r *EnhancedQdrantRetriever) HybridRetrieve(ctx context.Context, query string, opts *SearchOptions) ([]*SearchResult, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	// Get dense results
	denseResults, err := r.denseRetriever.Search(ctx, query, opts)
	if err != nil {
		return nil, err
	}

	// Get sparse results
	sparseResults, err := r.sparseIndex.Search(ctx, query, opts)
	if err != nil {
		return nil, err
	}

	// Fuse results
	var fused []*SearchResult
	switch r.fusionMethod {
	case FusionRRF:
		fused = r.fusionRRF(denseResults, sparseResults, opts)
	case FusionWeighted:
		fused = r.fusionWeighted(denseResults, sparseResults, opts)
	default:
		fused = r.fusionRRF(denseResults, sparseResults, opts)
	}

	// Rerank if enabled
	r.mu.RLock()
	reranker := r.reranker
	r.mu.RUnlock()

	if opts.UseReranker && reranker != nil {
		fused, err = reranker.Rerank(ctx, query, fused)
		if err != nil {
			r.logger.Printf("Reranking failed: %v", err)
			// Continue with unfused results
		}
	}

	if len(fused) > opts.TopK {
		fused = fused[:opts.TopK]
	}

	return fused, nil
}

// fusionRRF performs Reciprocal Rank Fusion
func (r *EnhancedQdrantRetriever) fusionRRF(dense, sparse []*SearchResult, opts *SearchOptions) []*SearchResult {
	const k = 60.0 // RRF constant

	scores := make(map[string]float64)
	results := make(map[string]*SearchResult)

	// Add dense results
	for rank, res := range dense {
		rrfScore := opts.DenseWeight / (k + float64(rank+1))
		scores[res.ID] += rrfScore
		results[res.ID] = res
	}

	// Add sparse results
	for rank, res := range sparse {
		rrfScore := opts.SparseWeight / (k + float64(rank+1))
		scores[res.ID] += rrfScore
		if _, ok := results[res.ID]; !ok {
			results[res.ID] = res
		}
	}

	// Create fused list
	var fused []*SearchResult
	for id, res := range results {
		fusedRes := &SearchResult{
			ID:       res.ID,
			Content:  res.Content,
			Score:    scores[id],
			Metadata: res.Metadata,
			Source:   "hybrid",
		}
		fused = append(fused, fusedRes)
	}

	sort.Slice(fused, func(i, j int) bool {
		return fused[i].Score > fused[j].Score
	})

	return fused
}

// fusionWeighted performs weighted score fusion
func (r *EnhancedQdrantRetriever) fusionWeighted(dense, sparse []*SearchResult, opts *SearchOptions) []*SearchResult {
	scores := make(map[string]float64)
	results := make(map[string]*SearchResult)

	// Normalize and add dense scores
	if len(dense) > 0 {
		maxDense := dense[0].Score
		for _, res := range dense {
			normScore := res.Score / maxDense
			scores[res.ID] += normScore * opts.DenseWeight
			results[res.ID] = res
		}
	}

	// Normalize and add sparse scores
	if len(sparse) > 0 {
		maxSparse := sparse[0].Score
		for _, res := range sparse {
			normScore := res.Score / maxSparse
			scores[res.ID] += normScore * opts.SparseWeight
			if _, ok := results[res.ID]; !ok {
				results[res.ID] = res
			}
		}
	}

	var fused []*SearchResult
	for id, res := range results {
		fusedRes := &SearchResult{
			ID:       res.ID,
			Content:  res.Content,
			Score:    scores[id],
			Metadata: res.Metadata,
			Source:   "hybrid",
		}
		fused = append(fused, fusedRes)
	}

	sort.Slice(fused, func(i, j int) bool {
		return fused[i].Score > fused[j].Score
	})

	return fused
}

// QdrantRAGPipeline provides a complete RAG pipeline
type QdrantRAGPipeline struct {
	retriever   *EnhancedQdrantRetriever
	debateEval  DebateEvaluator
	logger      *log.Logger
}

// NewQdrantRAGPipeline creates a new RAG pipeline
func NewQdrantRAGPipeline(collection string, embedder EmbeddingProvider, logger *log.Logger) *QdrantRAGPipeline {
	if logger == nil {
		logger = log.Default()
	}
	return &QdrantRAGPipeline{
		retriever: NewEnhancedQdrantRetriever(collection, embedder, logger),
		logger:    logger,
	}
}

// SetDebateEvaluator sets the debate evaluator
func (p *QdrantRAGPipeline) SetDebateEvaluator(eval DebateEvaluator) {
	p.debateEval = eval
	p.retriever.SetDebateEvaluator(eval)
}

// Retrieve retrieves relevant documents
func (p *QdrantRAGPipeline) Retrieve(ctx context.Context, query string, opts *SearchOptions) ([]*SearchResult, error) {
	return p.retriever.HybridRetrieve(ctx, query, opts)
}

// Index indexes documents
func (p *QdrantRAGPipeline) Index(ctx context.Context, docs []*Document) error {
	return p.retriever.Index(ctx, docs)
}

// Helper functions

func tokenize(text string) []string {
	text = strings.ToLower(text)
	words := strings.Fields(text)
	var tokens []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:\"'()[]{}") 
		if len(w) > 0 {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

func cosineSimilarity(a, b string) float64 {
	tokensA := tokenize(a)
	tokensB := tokenize(b)

	termSetA := make(map[string]float64)
	for _, t := range tokensA {
		termSetA[t]++
	}

	termSetB := make(map[string]float64)
	for _, t := range tokensB {
		termSetB[t]++
	}

	var dotProduct, normA, normB float64
	for t, countA := range termSetA {
		normA += countA * countA
		if countB, ok := termSetB[t]; ok {
			dotProduct += countA * countB
		}
	}
	for _, countB := range termSetB {
		normB += countB * countB
	}

	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
