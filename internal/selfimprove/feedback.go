package selfimprove

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryFeedbackCollector implements FeedbackCollector with in-memory storage
type InMemoryFeedbackCollector struct {
	mu            sync.RWMutex
	feedback      []*Feedback
	bySession     map[string][]*Feedback
	byPrompt      map[string][]*Feedback
	maxBufferSize int
	logger        *log.Logger
}

// NewInMemoryFeedbackCollector creates a new in-memory feedback collector
func NewInMemoryFeedbackCollector(logger *log.Logger, maxBufferSize int) *InMemoryFeedbackCollector {
	if logger == nil {
		logger = log.Default()
	}
	if maxBufferSize <= 0 {
		maxBufferSize = 10000
	}
	return &InMemoryFeedbackCollector{
		feedback:      make([]*Feedback, 0),
		bySession:     make(map[string][]*Feedback),
		byPrompt:      make(map[string][]*Feedback),
		maxBufferSize: maxBufferSize,
		logger:        logger,
	}
}

// Collect adds new feedback
func (c *InMemoryFeedbackCollector) Collect(ctx context.Context, feedback *Feedback) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if feedback.ID == "" {
		feedback.ID = uuid.New().String()
	}
	if feedback.CreatedAt.IsZero() {
		feedback.CreatedAt = time.Now()
	}

	// Trim if buffer is full
	if len(c.feedback) >= c.maxBufferSize {
		c.feedback = c.feedback[len(c.feedback)/2:]
		c.rebuildIndexes()
	}

	c.feedback = append(c.feedback, feedback)
	
	if feedback.SessionID != "" {
		c.bySession[feedback.SessionID] = append(c.bySession[feedback.SessionID], feedback)
	}
	if feedback.PromptID != "" {
		c.byPrompt[feedback.PromptID] = append(c.byPrompt[feedback.PromptID], feedback)
	}

	return nil
}

// rebuildIndexes rebuilds the session and prompt indexes
func (c *InMemoryFeedbackCollector) rebuildIndexes() {
	c.bySession = make(map[string][]*Feedback)
	c.byPrompt = make(map[string][]*Feedback)
	for _, f := range c.feedback {
		if f.SessionID != "" {
			c.bySession[f.SessionID] = append(c.bySession[f.SessionID], f)
		}
		if f.PromptID != "" {
			c.byPrompt[f.PromptID] = append(c.byPrompt[f.PromptID], f)
		}
	}
}

// GetBySession retrieves feedback for a session
func (c *InMemoryFeedbackCollector) GetBySession(ctx context.Context, sessionID string) ([]*Feedback, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result := make([]*Feedback, len(c.bySession[sessionID]))
	copy(result, c.bySession[sessionID])
	return result, nil
}

// GetByPrompt retrieves feedback for a prompt
func (c *InMemoryFeedbackCollector) GetByPrompt(ctx context.Context, promptID string) ([]*Feedback, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result := make([]*Feedback, len(c.byPrompt[promptID]))
	copy(result, c.byPrompt[promptID])
	return result, nil
}

// GetAggregated returns aggregated feedback statistics
func (c *InMemoryFeedbackCollector) GetAggregated(ctx context.Context, filter *FeedbackFilter) (*AggregatedFeedback, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	filtered := c.applyFilter(filter)

	agg := &AggregatedFeedback{
		TypeDistribution:   make(map[FeedbackType]int),
		SourceDistribution: make(map[FeedbackSource]int),
		DimensionAverages:  make(map[DimensionType]float64),
		ProviderStats:      make(map[string]*ProviderFeedbackStats),
		ScoreDistribution:  make(map[string]int),
	}

	if len(filtered) == 0 {
		return agg, nil
	}

	var totalScore float64
	dimCounts := make(map[DimensionType]int)
	dimSums := make(map[DimensionType]float64)

	for _, f := range filtered {
		agg.TotalCount++
		totalScore += f.Score
		agg.TypeDistribution[f.Type]++
		agg.SourceDistribution[f.Source]++

		// Track dimensions
		for dim, score := range f.Dimensions {
			dimSums[dim] += score
			dimCounts[dim]++
		}

		// Track provider stats
		if f.ProviderName != "" {
			if _, ok := agg.ProviderStats[f.ProviderName]; !ok {
				agg.ProviderStats[f.ProviderName] = &ProviderFeedbackStats{
					ProviderName: f.ProviderName,
					Dimensions:   make(map[DimensionType]float64),
				}
			}
			agg.ProviderStats[f.ProviderName].TotalCount++
			agg.ProviderStats[f.ProviderName].AverageScore += f.Score
		}
	}

	agg.AverageScore = totalScore / float64(agg.TotalCount)

	for dim, sum := range dimSums {
		agg.DimensionAverages[dim] = sum / float64(dimCounts[dim])
	}

	for _, ps := range agg.ProviderStats {
		if ps.TotalCount > 0 {
			ps.AverageScore /= float64(ps.TotalCount)
		}
	}

	return agg, nil
}

// applyFilter filters feedback based on criteria
func (c *InMemoryFeedbackCollector) applyFilter(filter *FeedbackFilter) []*Feedback {
	if filter == nil {
		return c.feedback
	}

	var result []*Feedback
	for _, f := range c.feedback {
		if c.matchesFilter(f, filter) {
			result = append(result, f)
		}
	}
	return result
}

// matchesFilter checks if feedback matches filter criteria
func (c *InMemoryFeedbackCollector) matchesFilter(f *Feedback, filter *FeedbackFilter) bool {
	if len(filter.Types) > 0 {
		found := false
		for _, t := range filter.Types {
			if f.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filter.MinScore != nil && f.Score < *filter.MinScore {
		return false
	}
	if filter.MaxScore != nil && f.Score > *filter.MaxScore {
		return false
	}
	if filter.StartTime != nil && f.CreatedAt.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && f.CreatedAt.After(*filter.EndTime) {
		return false
	}

	return true
}

// Export exports feedback as training examples
func (c *InMemoryFeedbackCollector) Export(ctx context.Context, filter *FeedbackFilter) ([]*TrainingExample, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	filtered := c.applyFilter(filter)
	
	// Group by prompt
	byPrompt := make(map[string][]*Feedback)
	for _, f := range filtered {
		byPrompt[f.PromptID] = append(byPrompt[f.PromptID], f)
	}

	var examples []*TrainingExample
	for promptID, feedbacks := range byPrompt {
		example := &TrainingExample{
			ID:        uuid.New().String(),
			Prompt:    promptID,
			Feedback:  feedbacks,
			CreatedAt: time.Now(),
		}

		// Calculate average reward score
		var totalScore float64
		for _, f := range feedbacks {
			totalScore += f.Score
		}
		example.RewardScore = totalScore / float64(len(feedbacks))

		examples = append(examples, example)
	}

	return examples, nil
}

// AutoFeedbackCollector automatically collects feedback using AI evaluation
type AutoFeedbackCollector struct {
	collector   FeedbackCollector
	rewardModel RewardModel
	config      *SelfImprovementConfig
	logger      *log.Logger
}

// NewAutoFeedbackCollector creates a new auto feedback collector
func NewAutoFeedbackCollector(collector FeedbackCollector, rewardModel RewardModel, config *SelfImprovementConfig, logger *log.Logger) *AutoFeedbackCollector {
	if logger == nil {
		logger = log.Default()
	}
	return &AutoFeedbackCollector{
		collector:   collector,
		rewardModel: rewardModel,
		config:      config,
		logger:      logger,
	}
}

// CollectAuto automatically evaluates and collects feedback for a response
func (afc *AutoFeedbackCollector) CollectAuto(ctx context.Context, sessionID, promptID, prompt, response, providerName, model string) (*Feedback, error) {
	score, err := afc.rewardModel.Score(ctx, prompt, response)
	if err != nil {
		return nil, err
	}

	feedbackType := FeedbackTypeNeutral
	if score >= 0.7 {
		feedbackType = FeedbackTypePositive
	} else if score <= 0.3 {
		feedbackType = FeedbackTypeNegative
	}

	feedback := &Feedback{
		SessionID:    sessionID,
		PromptID:     promptID,
		Type:         feedbackType,
		Source:       FeedbackSourceAI,
		Score:        score,
		ProviderName: providerName,
		Model:        model,
		CreatedAt:    time.Now(),
	}

	if err := afc.collector.Collect(ctx, feedback); err != nil {
		return nil, err
	}

	return feedback, nil
}
