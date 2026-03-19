// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"digital.vasic.llmsverifier/llmverifier"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStress_ConcurrentRecipeApply(t *testing.T) {
	var wg sync.WaitGroup
	errors := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := llmverifier.NewStrategyBuilder(fmt.Sprintf("s-%d", idx)).
				WithWeights(llmverifier.WeightConfig{
					VisionCapability:     0.50,
					InstructionFollowing: 0.50,
				}).
				WithRecipe(VisionTest()).
				WithRecipe(InstructionFollowingTest()).
				WithRecipe(ContextWindowTest(100000)).
				WithRecipe(StreamingReliabilityTest()).
				WithRecipe(LongSessionStabilityTest(time.Duration(idx) * time.Minute)).
				Build()
			if err != nil {
				errors <- err
			}
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent recipe apply failed: %v", err)
	}
}

func TestStress_ConcurrentRecipeTestRun(t *testing.T) {
	s, err := llmverifier.NewStrategyBuilder("test").
		WithWeights(llmverifier.WeightConfig{VisionCapability: 1.0}).
		WithRecipe(VisionTest()).
		Build()
	require.NoError(t, err)

	test := s.CustomTests()[0]
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := test.Run(context.Background(), nil)
			assert.NoError(t, err)
			assert.False(t, result.Passed) // nil client
		}()
	}
	wg.Wait()
}

func TestStress_RapidRecipeConstruction(t *testing.T) {
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_ = VisionTest()
		_ = InstructionFollowingTest()
		_ = ContextWindowTest(50000)
		_ = StreamingReliabilityTest()
		_ = LongSessionStabilityTest(30 * time.Minute)
	}
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 100*time.Millisecond, "constructing 50K recipes should be fast")
}
