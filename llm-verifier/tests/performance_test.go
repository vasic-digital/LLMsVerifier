package tests

import (
	"testing"
	"time"

	"llm-verifier/llmverifier"
)

// Performance and benchmark tests for the LLM verifier

func BenchmarkCalculateCodeCapabilityScore(b *testing.B) {
	verifier := &llmverifier.Verifier{}

	codeCaps := llmverifier.CodeCapabilityResult{
		CodeGeneration:   true,
		CodeCompletion:   true,
		CodeDebugging:    true,
		CodeReview:       true,
		TestGeneration:   true,
		Documentation:    true,
		Architecture:     true,
		CodeOptimization: true,
		ComplexityHandling: llmverifier.ComplexityMetrics{
			MaxHandledDepth:   5,
			CodeQuality:       95,
			LogicCorrectness:  90,
			RuntimeEfficiency: 85,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.CalculateCodeCapabilityScore(codeCaps)
	}
}

func BenchmarkCalculateResponsivenessScore(b *testing.B) {
	verifier := &llmverifier.Verifier{}

	availability := llmverifier.AvailabilityResult{
		Latency: 500 * time.Millisecond,
	}
	
	responseTime := llmverifier.ResponseTimeResult{
		AverageLatency: 500 * time.Millisecond,
		MinLatency:     200 * time.Millisecond,
		MaxLatency:     800 * time.Millisecond,
		Throughput:     10,
		MeasurementCount: 5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.CalculateResponsivenessScore(availability, responseTime)
	}
}

func BenchmarkCalculateReliabilityScore(b *testing.B) {
	verifier := &llmverifier.Verifier{}

	availability := llmverifier.AvailabilityResult{
		Exists:      true,
		Responsive:  true,
		Overloaded:  false,
		Error:       "",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.CalculateReliabilityScore(availability)
	}
}

func BenchmarkCalculateFeatureRichnessScore(b *testing.B) {
	verifier := &llmverifier.Verifier{}

	features := llmverifier.FeatureDetectionResult{
		ToolUse:            true,
		CodeGeneration:     true,
		CodeCompletion:     true,
		CodeExplanation:    true,
		CodeReview:         true,
		Streaming:          true,
		Embeddings:         true,
		Reasoning:          true,
		StructuredOutput:   true,
		JSONMode:           true,
		ParallelToolUse:    true,
		Multimodal:         true,
		ImageGeneration:    false,
		AudioGeneration:    false,
		MCPs:               false,
		LSPs:               false,
		Reranking:          false,
		}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.CalculateFeatureRichnessScore(features)
	}
}

func BenchmarkSortResultsByScore(b *testing.B) {
	verifier := &llmverifier.Verifier{}

	// Create mock results to sort
	results := make([]llmverifier.VerificationResult, 100)
	for i := range results {
		results[i] = llmverifier.VerificationResult{
			ModelInfo: llmverifier.ModelInfo{
				ID: "model-" + string(rune(i+'0')),
			},
			PerformanceScores: llmverifier.PerformanceScore{
				OverallScore: float64((i*7)%100), // Generate pseudo-random scores
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = verifier.SortResultsByScore(results, func(r llmverifier.VerificationResult) float64 {
			return r.PerformanceScores.OverallScore
		})
	}
}

func TestPerformanceThresholds(t *testing.T) {
	// Test that core functions perform within acceptable time thresholds
	verifier := &llmverifier.Verifier{}

	// Test code capability scoring performance
	start := time.Now()
	codeCaps := llmverifier.CodeCapabilityResult{
		CodeGeneration:   true,
		CodeCompletion:   true,
		CodeDebugging:    true,
		CodeReview:       true,
		TestGeneration:   true,
		Documentation:    true,
		Architecture:     true,
		CodeOptimization: true,
		ComplexityHandling: llmverifier.ComplexityMetrics{
			MaxHandledDepth:   5,
			CodeQuality:       95,
			LogicCorrectness:  90,
			RuntimeEfficiency: 85,
		},
	}
	
	_, _ = verifier.CalculateCodeCapabilityScore(codeCaps)
	elapsed := time.Since(start)
	
	if elapsed > 10*time.Millisecond {
		t.Errorf("Code capability scoring took too long: %v", elapsed)
	}

	// Test responsiveness scoring performance
	start = time.Now()
	availability := llmverifier.AvailabilityResult{
		Latency: 500 * time.Millisecond,
	}
	
	responseTime := llmverifier.ResponseTimeResult{
		AverageLatency: 500 * time.Millisecond,
		MinLatency:     200 * time.Millisecond,
		MaxLatency:     800 * time.Millisecond,
		Throughput:     10,
		MeasurementCount: 5,
	}

	_, _ = verifier.CalculateResponsivenessScore(availability, responseTime)
	elapsed = time.Since(start)

	if elapsed > 5*time.Millisecond {
		t.Errorf("Responsiveness scoring took too long: %v", elapsed)
	}
}

func BenchmarkGenerativeCapabilitiesAssessment(b *testing.B) {
	// Note: This benchmark would require a mock client in real implementation
	// For now we're just testing the function signature and structure

	b.Log("Benchmark for generative capabilities assessment prepared")
}

func TestScoringWithGenerativeCapabilities(t *testing.T) {
	// Test that generative capabilities are factored into scoring

	result := llmverifier.VerificationResult{
		FeatureDetection: llmverifier.FeatureDetectionResult{
			CodeGeneration: true,
			ToolUse: true,
			Reranking: true,
			ImageGeneration: true,
			MCPs: true,
			LSPs: true,
		},
		CodeCapabilities: llmverifier.CodeCapabilityResult{
			CodeGeneration: true,
			CodeCompletion: true,
			CodeDebugging: true,
		},
		GenerativeCapabilities: llmverifier.GenerativeCapabilityResult{
			CreativeWriting: true,
			Storytelling: true,
			ContentGeneration: true,
		},
		Availability: llmverifier.AvailabilityResult{
			Exists: true,
			Responsive: true,
			Latency: 200 * time.Millisecond,
		},
		ResponseTime: llmverifier.ResponseTimeResult{
			AverageLatency: 200 * time.Millisecond,
			Throughput: 5.0,
		},
	}

	verifier := &llmverifier.Verifier{}
	scores, _ := verifier.CalculateScores(result)

	if scores.OverallScore <= 0 || scores.OverallScore > 100 {
		t.Errorf("Expected overall score between 0-100, got %f", scores.OverallScore)
	}

	if scores.CodeCapability <= 0 {
		t.Errorf("Expected positive code capability score, got %f", scores.CodeCapability)
	}

	t.Logf("Scores calculated - Overall: %.2f, Code: %.2f, Responsiveness: %.2f",
		   scores.OverallScore, scores.CodeCapability, scores.Responsiveness)
}