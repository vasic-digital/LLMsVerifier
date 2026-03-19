// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStress_ConcurrentScoreModel(t *testing.T) {
	ds := NewDefaultStrategy()
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results := []TestResult{
				{Category: TestCategoryResponsiveness, Score: float64(idx) / 100.0},
				{Category: TestCategoryCode, Score: 0.5},
			}
			_, err := ds.ScoreModel(context.Background(), ModelInfo{ID: fmt.Sprintf("model-%d", idx)}, results)
			if err != nil {
				errors <- err
			}
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent ScoreModel failed: %v", err)
	}
}

func TestStress_ConcurrentStrategyBuilder(t *testing.T) {
	var wg sync.WaitGroup
	errors := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := NewStrategyBuilder(fmt.Sprintf("strategy-%d", idx)).
				WithWeights(WeightConfig{Responsiveness: 0.5, Reliability: 0.5}).
				RequireCapability("vision").
				MinScore(0.5).
				Build()
			if err != nil {
				errors <- err
			}
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent Build failed: %v", err)
	}
}

func TestStress_LargeModelList_FilterModels(t *testing.T) {
	s, err := NewStrategyBuilder("large").
		WithWeights(WeightConfig{VisionCapability: 1.0}).
		RequireCapability("vision").
		Build()
	require.NoError(t, err)

	var models []ModelInfo
	for i := 0; i < 10000; i++ {
		models = append(models, ModelInfo{
			ID:             fmt.Sprintf("model-%d", i),
			SupportsVision: i%2 == 0, // half have vision
		})
	}

	start := time.Now()
	filtered := s.FilterModels(models)
	elapsed := time.Since(start)

	assert.Len(t, filtered, 5000)
	assert.Less(t, elapsed, 100*time.Millisecond, "filtering 10K models should be fast")
}

func TestStress_LargeResultSet_ScoreModel(t *testing.T) {
	s, err := NewStrategyBuilder("large").
		WithWeights(WeightConfig{
			Responsiveness:  0.25,
			CodeCapability:  0.25,
			FeatureRichness: 0.25,
			Reliability:     0.25,
		}).
		Build()
	require.NoError(t, err)

	var results []TestResult
	categories := []TestCategory{TestCategoryResponsiveness, TestCategoryCode, TestCategoryFeature, TestCategoryExistence}
	for i := 0; i < 100000; i++ {
		results = append(results, TestResult{
			Category: categories[i%4],
			Score:    0.8,
		})
	}

	start := time.Now()
	score, err := s.ScoreModel(context.Background(), ModelInfo{}, results)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.InDelta(t, 0.8, score.Overall, 0.001)
	assert.Less(t, elapsed, 500*time.Millisecond, "scoring 100K results should be fast")
}

func TestStress_RapidBuilderCreation(t *testing.T) {
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_, err := NewStrategyBuilder(fmt.Sprintf("s-%d", i)).
			WithWeights(WeightConfig{Responsiveness: 1.0}).
			Build()
		require.NoError(t, err)
	}
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 1*time.Second, "building 10K strategies should be fast")
}

func TestStress_WeightConfig_Validate_Rapid(t *testing.T) {
	wc := WeightConfig{Responsiveness: 0.5, Reliability: 0.5}
	start := time.Now()
	for i := 0; i < 100000; i++ {
		err := wc.Validate()
		require.NoError(t, err)
	}
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 500*time.Millisecond, "100K validations should be fast")
}
