package ai

import (
	"testing"
	"time"
)

func TestSimpleRecommender(t *testing.T) {
	recommender := NewSimpleRecommender()

	// Test coding task recommendation
	req := RecRequest{
		TaskType:   "coding",
		Complexity: "medium",
		MaxCost:    1.0, // High budget to avoid cost filtering
	}

	recommendations := recommender.Recommend(req)

	if len(recommendations) == 0 {
		t.Fatal("Expected at least one recommendation")
	}

	// Check that we get some recommendations
	if len(recommendations) > 4 {
		t.Errorf("Too many recommendations: %d", len(recommendations))
	}

	// Check scores are reasonable
	for _, rec := range recommendations {
		if rec.Score < 0 || rec.Score > 1 {
			t.Errorf("Invalid score %f for model %s", rec.Score, rec.ModelID)
		}
		if rec.Cost < 0 {
			t.Errorf("Invalid cost %f for model %s", rec.Cost, rec.ModelID)
		}
		if rec.Time <= 0 {
			t.Errorf("Invalid time estimation for model %s", rec.ModelID)
		}
		if rec.Reasoning == "" {
			t.Errorf("Missing reasoning for model %s", rec.ModelID)
		}
	}

	// Check that scores are sorted descending
	for i := 1; i < len(recommendations); i++ {
		if recommendations[i].Score > recommendations[i-1].Score {
			t.Errorf("Recommendations not sorted by score: %f > %f",
				recommendations[i].Score, recommendations[i-1].Score)
		}
	}
}

func TestRecommendationScoring(t *testing.T) {
	recommender := NewSimpleRecommender()

	tests := []struct {
		name       string
		taskType   string
		complexity string
	}{
		{
			name:       "Coding task",
			taskType:   "coding",
			complexity: "medium",
		},
		{
			name:       "Writing task",
			taskType:   "writing",
			complexity: "medium",
		},
		{
			name:       "Simple task",
			taskType:   "chat",
			complexity: "simple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := RecRequest{
				TaskType:   tt.taskType,
				Complexity: tt.complexity,
				MaxCost:    1.0, // High budget
			}

			recommendations := recommender.Recommend(req)
			if len(recommendations) == 0 {
				t.Fatal("Expected recommendations")
			}

			// Just verify we get reasonable results
			if len(recommendations) > 4 {
				t.Errorf("Too many recommendations for %s: %d", tt.name, len(recommendations))
			}

			// Verify the top recommendation has good reasoning
			if recommendations[0].Reasoning == "" {
				t.Errorf("Missing reasoning for top recommendation in %s", tt.name)
			}
		})
	}
}

func TestCostConstraints(t *testing.T) {
	recommender := NewSimpleRecommender()

	// Test with reasonable budget
	req := RecRequest{
		TaskType:   "coding",
		Complexity: "simple", // Simple task should be cheaper
		MaxCost:    0.01,     // 1 cent budget
	}

	recommendations := recommender.Recommend(req)

	// Should still get recommendations (cheaper models)
	if len(recommendations) == 0 {
		t.Fatal("Expected recommendations even with low budget")
	}

	// All recommendations should be within budget
	for _, rec := range recommendations {
		if rec.Cost > req.MaxCost {
			t.Errorf("Recommendation cost %f exceeds budget %f", rec.Cost, req.MaxCost)
		}
	}
}

func TestFeatureRequirements(t *testing.T) {
	recommender := NewSimpleRecommender()

	// Test with streaming requirement
	req := RecRequest{
		TaskType:         "coding",
		Complexity:       "medium",
		RequiredFeatures: []string{"streaming"},
	}

	recommendations := recommender.Recommend(req)

	// All recommendations should have streaming
	for _, rec := range recommendations {
		model := recommender.models[rec.ModelID]
		if !model.Features["streaming"] {
			t.Errorf("Model %s recommended but doesn't have streaming feature", rec.ModelID)
		}
	}
}

func TestTimeEstimations(t *testing.T) {
	recommender := NewSimpleRecommender()

	req := RecRequest{
		TaskType:   "coding",
		Complexity: "medium",
	}

	recommendations := recommender.Recommend(req)

	for _, rec := range recommendations {
		if rec.Time <= 0 {
			t.Errorf("Invalid time estimation for model %s: %v", rec.ModelID, rec.Time)
		}
		// Should be reasonable (under 10 minutes)
		if rec.Time > 10*time.Minute {
			t.Errorf("Unreasonably long time estimation for model %s: %v", rec.ModelID, rec.Time)
		}
	}
}
