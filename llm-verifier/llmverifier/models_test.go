package llmverifier

import (
	"encoding/json"
	"testing"
	"time"
)

func TestVerificationResult_JsonMarshal(t *testing.T) {
	result := VerificationResult{
		ModelInfo: ModelInfo{
			ID:       "test-model",
			Object:   "model",
			Created:  1234567890,
			OwnedBy:  "test-owner",
			Endpoint: "https://api.test.com",
		},
		Availability: AvailabilityResult{
			Exists:      true,
			Responsive:  true,
			Overloaded:  false,
			Latency:     100 * time.Millisecond,
			LastChecked: time.Now(),
		},
		ResponseTime: ResponseTimeResult{
			AverageLatency:   150 * time.Millisecond,
			P95Latency:       200 * time.Millisecond,
			MinLatency:       50 * time.Millisecond,
			MaxLatency:       300 * time.Millisecond,
			Throughput:       10.5,
			MeasurementCount: 100,
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:          true,
			CodeGeneration:   true,
			CodeCompletion:   true,
			Embeddings:       true,
			Streaming:        true,
			JSONMode:         true,
			StructuredOutput: true,
			Reasoning:       true,
			ParallelToolUse:  true,
			MaxParallelCalls: 5,
			BatchProcessing:  true,
			Modalities:       []string{"text", "image"},
		},
		CodeCapabilities: CodeCapabilityResult{
			LanguageSupport: []string{"python", "javascript", "go", "java"},
			CodeGeneration:  true,
			CodeCompletion:  true,
			CodeDebugging:   true,
			CodeOptimization: true,
			CodeReview:      true,
			CodeExplanation: true,
			TestGeneration:  true,
			Documentation:   true,
			Refactoring:    true,
			ErrorResolution: true,
			Architecture:   true,
			SecurityAssessment: true,
			PatternRecognition: true,
			DebuggingAccuracy: 95.5,
			ComplexityHandling: ComplexityMetrics{
				MaxHandledDepth: 10,
				MaxTokens:       4096,
				CodeQuality:     85.0,
				LogicCorrectness: 90.0,
				RuntimeEfficiency: 88.0,
			},
			PromptResponse: PromptResponseTest{
				PythonSuccessRate:       95.0,
				JavascriptSuccessRate: 92.0,
				GoSuccessRate:          90.0,
				JavaSuccessRate:        88.0,
				CppSuccessRate:        85.0,
				TypescriptSuccessRate: 93.0,
				OverallSuccessRate:     90.5,
				AvgResponseTime:       120 * time.Millisecond,
			},
		},
		GenerativeCapabilities: GenerativeCapabilityResult{
			CreativeWriting:        true,
			Storytelling:          true,
			ContentGeneration:     true,
			ArtisticCreativity:    true,
			ProblemSolving:       true,
			MultimodalGenerative:  true,
			OriginalityScore:      85.5,
			CreativityScore:      88.0,
		},
		PerformanceScores: PerformanceScore{
			OverallScore:     85.5,
			CodeCapability:   90.0,
			Responsiveness:   88.0,
			Reliability:      92.0,
			FeatureRichness:  87.0,
			ValueProposition: 86.5,
		},
		Timestamp: time.Now(),
	}

	// Test JSON marshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal VerificationResult: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled VerificationResult
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal VerificationResult: %v", err)
	}

	// Verify key fields match
	if unmarshaled.ModelInfo.ID != result.ModelInfo.ID {
		t.Errorf("Expected ID %s, got %s", result.ModelInfo.ID, unmarshaled.ModelInfo.ID)
	}

	if unmarshaled.Availability.Exists != result.Availability.Exists {
		t.Errorf("Expected Exists %v, got %v", result.Availability.Exists, unmarshaled.Availability.Exists)
	}

	if unmarshaled.PerformanceScores.OverallScore != result.PerformanceScores.OverallScore {
		t.Errorf("Expected OverallScore %f, got %f", result.PerformanceScores.OverallScore, unmarshaled.PerformanceScores.OverallScore)
	}
}

func TestModelInfo_Fields(t *testing.T) {
	modelInfo := ModelInfo{
		ID:                 "gpt-4",
		Object:             "model",
		Created:            1677610602,
		OwnedBy:           "openai",
		Root:              "",
		Parent:            "",
		Permissions:       []Permission{},
		ScalingPolicy:     &ScalingPolicy{},
		Capabilities:      Capabilities{},
		ContextWindow:     ContextWindow{},
		MaxOutputTokens:   4096,
		InputPrices:       InputPrices{},
		OutputPrices:      OutputPrices{},
		HasTrainingData:   false,
		Description:       "GPT-4 model",
		Architecture:      Architecture{},
		Tokenizer:         "cl100k_base",
		Organization:      "openai",
		ReleaseDate:       "2023-03-14",
		LanguageSupport:   []string{"en", "es", "fr", "de"},
		UseCase:           "chat",
		Version:           "1.0",
		MaxInputTokens:    8192,
		SupportsVision:    false,
		SupportsAudio:     false,
		SupportsVideo:     false,
		SupportsReasoning: true,
		OpenSource:        false,
		Deprecated:        false,
		Tags:              []string{"chat", "completion"},
		Endpoint:          "https://api.openai.com/v1",
	}

	// Test that all fields can be set and accessed
	if modelInfo.ID != "gpt-4" {
		t.Errorf("Expected ID 'gpt-4', got '%s'", modelInfo.ID)
	}

	if modelInfo.Object != "model" {
		t.Errorf("Expected Object 'model', got '%s'", modelInfo.Object)
	}

	if modelInfo.Created != 1677610602 {
		t.Errorf("Expected Created 1677610602, got %d", modelInfo.Created)
	}

	if !modelInfo.SupportsReasoning {
		t.Error("Expected SupportsReasoning to be true")
	}

	if len(modelInfo.LanguageSupport) != 4 {
		t.Errorf("Expected 4 languages, got %d", len(modelInfo.LanguageSupport))
	}
}

func TestPermission_Fields(t *testing.T) {
	permission := Permission{
		ID:                 "perm_123",
		Object:             "model",
		Created:            1640995200,
		AllowCreate_engine: true,
		AllowSampling:      true,
		AllowLogprobs:      true,
		AllowSearchIndices: false,
		AllowView:          true,
		AllowFineTuning:    false,
		Organization:       "org_test",
		Group:              "group_1",
		IsBlocking:         false,
		Type:               "custom",
	}

	if permission.ID != "perm_123" {
		t.Errorf("Expected ID 'perm_123', got '%s'", permission.ID)
	}

	if !permission.AllowCreate_engine {
		t.Error("Expected AllowCreate_engine to be true")
	}

	if permission.IsBlocking {
		t.Error("Expected IsBlocking to be false")
	}
}

func TestAvailabilityResult_Fields(t *testing.T) {
	now := time.Now()
	result := AvailabilityResult{
		Exists:      true,
		Responsive:  true,
		Overloaded:  false,
		Latency:     250 * time.Millisecond,
		LastChecked: now,
		Error:       "",
	}

	if !result.Exists {
		t.Error("Expected Exists to be true")
	}

	if !result.Responsive {
		t.Error("Expected Responsive to be true")
	}

	if result.Overloaded {
		t.Error("Expected Overloaded to be false")
	}

	if result.Latency != 250*time.Millisecond {
		t.Errorf("Expected Latency 250ms, got %v", result.Latency)
	}

	if !result.LastChecked.Equal(now) {
		t.Error("Expected LastChecked to match now")
	}

	if result.Error != "" {
		t.Errorf("Expected empty Error, got '%s'", result.Error)
	}
}

func TestResponseTimeResult_Fields(t *testing.T) {
	result := ResponseTimeResult{
		AverageLatency:   180 * time.Millisecond,
		P95Latency:       300 * time.Millisecond,
		MinLatency:       50 * time.Millisecond,
		MaxLatency:       500 * time.Millisecond,
		Throughput:       15.5,
		MeasurementCount: 1000,
	}

	if result.AverageLatency != 180*time.Millisecond {
		t.Errorf("Expected AverageLatency 180ms, got %v", result.AverageLatency)
	}

	if result.P95Latency != 300*time.Millisecond {
		t.Errorf("Expected P95Latency 300ms, got %v", result.P95Latency)
	}

	if result.Throughput != 15.5 {
		t.Errorf("Expected Throughput 15.5, got %f", result.Throughput)
	}

	if result.MeasurementCount != 1000 {
		t.Errorf("Expected MeasurementCount 1000, got %d", result.MeasurementCount)
	}
}

func TestFeatureDetectionResult_Fields(t *testing.T) {
	result := FeatureDetectionResult{
		ToolUse:          true,
		Functions:        []FunctionDefinition{},
		CodeGeneration:   true,
		CodeCompletion:   true,
		CodeReview:       true,
		CodeExplanation:  true,
		Embeddings:       false,
		Reranking:        false,
		ImageGeneration:  false,
		AudioGeneration:  false,
		VideoGeneration:  false,
		MCPs:             false,
		LSPs:             false,
		Multimodal:       false,
		Streaming:        true,
		JSONMode:         true,
		StructuredOutput: true,
		Reasoning:       true,
		FunctionCalling:  true,
		ParallelToolUse:  true,
		MaxParallelCalls: 3,
		Modalities:       []string{"text"},
		BatchProcessing:  true,
	}

	if !result.ToolUse {
		t.Error("Expected ToolUse to be true")
	}

	if !result.CodeGeneration {
		t.Error("Expected CodeGeneration to be true")
	}

	if !result.Streaming {
		t.Error("Expected Streaming to be true")
	}

	if result.MaxParallelCalls != 3 {
		t.Errorf("Expected MaxParallelCalls 3, got %d", result.MaxParallelCalls)
	}

	if len(result.Modalities) != 1 || result.Modalities[0] != "text" {
		t.Errorf("Expected Modalities ['text'], got %v", result.Modalities)
	}
}

func TestGenerativeCapabilityResult_Fields(t *testing.T) {
	result := GenerativeCapabilityResult{
		CreativeWriting:        true,
		Storytelling:          true,
		ContentGeneration:     true,
		ArtisticCreativity:    true,
		ProblemSolving:       true,
		MultimodalGenerative:  false,
		OriginalityScore:      92.5,
		CreativityScore:      88.0,
	}

	if !result.CreativeWriting {
		t.Error("Expected CreativeWriting to be true")
	}

	if !result.ProblemSolving {
		t.Error("Expected ProblemSolving to be true")
	}

	if result.OriginalityScore != 92.5 {
		t.Errorf("Expected OriginalityScore 92.5, got %f", result.OriginalityScore)
	}

	if result.CreativityScore != 88.0 {
		t.Errorf("Expected CreativityScore 88.0, got %f", result.CreativityScore)
	}
}

func TestCodeCapabilityResult_Fields(t *testing.T) {
	result := CodeCapabilityResult{
		LanguageSupport: []string{"python", "javascript", "go", "rust", "java"},
		CodeGeneration:  true,
		CodeCompletion:  true,
		CodeDebugging:   true,
		CodeOptimization: true,
		CodeReview:      true,
		CodeExplanation: true,
		TestGeneration:  true,
		Documentation:   true,
		Refactoring:    true,
		ErrorResolution: true,
		Architecture:   true,
		SecurityAssessment: true,
		PatternRecognition: true,
		DebuggingAccuracy: 96.5,
		ComplexityHandling: ComplexityMetrics{
			MaxHandledDepth: 15,
			MaxTokens:       8192,
			CodeQuality:     92.0,
			LogicCorrectness: 94.0,
			RuntimeEfficiency: 89.5,
		},
		PromptResponse: PromptResponseTest{
			PythonSuccessRate:       98.0,
			JavascriptSuccessRate: 95.0,
			GoSuccessRate:          93.0,
			JavaSuccessRate:        90.0,
			CppSuccessRate:        87.0,
			TypescriptSuccessRate: 94.0,
			OverallSuccessRate:     93.0,
			AvgResponseTime:       100 * time.Millisecond,
		},
	}

	if len(result.LanguageSupport) != 5 {
		t.Errorf("Expected 5 languages, got %d", len(result.LanguageSupport))
	}

	if !result.CodeGeneration {
		t.Error("Expected CodeGeneration to be true")
	}

	if result.DebuggingAccuracy != 96.5 {
		t.Errorf("Expected DebuggingAccuracy 96.5, got %f", result.DebuggingAccuracy)
	}

	if result.ComplexityHandling.MaxHandledDepth != 15 {
		t.Errorf("Expected MaxHandledDepth 15, got %d", result.ComplexityHandling.MaxHandledDepth)
	}

	if result.PromptResponse.OverallSuccessRate != 93.0 {
		t.Errorf("Expected OverallSuccessRate 93.0, got %f", result.PromptResponse.OverallSuccessRate)
	}
}

func TestPerformanceScore_Fields(t *testing.T) {
	score := PerformanceScore{
		OverallScore:     89.5,
		CodeCapability:   92.0,
		Responsiveness:   88.0,
		Reliability:      95.0,
		FeatureRichness:  87.5,
		ValueProposition: 86.0,
	}

	if score.OverallScore != 89.5 {
		t.Errorf("Expected OverallScore 89.5, got %f", score.OverallScore)
	}

	if score.CodeCapability != 92.0 {
		t.Errorf("Expected CodeCapability 92.0, got %f", score.CodeCapability)
	}

	if score.Reliability != 95.0 {
		t.Errorf("Expected Reliability 95.0, got %f", score.Reliability)
	}
}

func TestComplexityMetrics_Fields(t *testing.T) {
	metrics := ComplexityMetrics{
		MaxHandledDepth:   20,
		MaxTokens:         16384,
		CodeQuality:       91.5,
		LogicCorrectness: 93.0,
		RuntimeEfficiency: 89.0,
	}

	if metrics.MaxHandledDepth != 20 {
		t.Errorf("Expected MaxHandledDepth 20, got %d", metrics.MaxHandledDepth)
	}

	if metrics.MaxTokens != 16384 {
		t.Errorf("Expected MaxTokens 16384, got %d", metrics.MaxTokens)
	}

	if metrics.CodeQuality != 91.5 {
		t.Errorf("Expected CodeQuality 91.5, got %f", metrics.CodeQuality)
	}

	if metrics.RuntimeEfficiency != 89.0 {
		t.Errorf("Expected RuntimeEfficiency 89.0, got %f", metrics.RuntimeEfficiency)
	}
}

func TestPromptResponseTest_Fields(t *testing.T) {
	result := PromptResponseTest{
		PythonSuccessRate:       97.5,
		JavascriptSuccessRate: 94.0,
		GoSuccessRate:          92.5,
		JavaSuccessRate:        89.0,
		CppSuccessRate:        86.0,
		TypescriptSuccessRate: 93.0,
		OverallSuccessRate:     92.0,
		AvgResponseTime:       110 * time.Millisecond,
	}

	if result.PythonSuccessRate != 97.5 {
		t.Errorf("Expected PythonSuccessRate 97.5, got %f", result.PythonSuccessRate)
	}

	if result.OverallSuccessRate != 92.0 {
		t.Errorf("Expected OverallSuccessRate 92.0, got %f", result.OverallSuccessRate)
	}

	if result.AvgResponseTime != 110*time.Millisecond {
		t.Errorf("Expected AvgResponseTime 110ms, got %v", result.AvgResponseTime)
	}
}

func TestSummary_Fields(t *testing.T) {
	startTime := time.Now().Add(-5 * time.Minute)
	endTime := time.Now()
	topPerformers := []TopPerformer{
		{ModelName: "model-1", Score: 95.0, Rank: 1},
		{ModelName: "model-2", Score: 92.0, Rank: 2},
		{ModelName: "model-3", Score: 88.0, Rank: 3},
	}
	categoryRankings := CategoryRankings{
		ByCodeCapability: topPerformers[:2],
		ByResponsiveness:  topPerformers[:2],
		ByReliability:     topPerformers[:2],
		ByFeatureRichness: topPerformers[:2],
		ByValue:           topPerformers[:2],
	}

	summary := Summary{
		TotalModels:      10,
		AvailableModels:  8,
		FailedModels:     2,
		StartTime:        startTime,
		EndTime:          endTime,
		Duration:         5 * time.Minute,
		AverageScore:     91.7,
		TopPerformers:    topPerformers,
		CategoryRankings: categoryRankings,
	}

	if summary.TotalModels != 10 {
		t.Errorf("Expected TotalModels 10, got %d", summary.TotalModels)
	}

	if summary.AvailableModels != 8 {
		t.Errorf("Expected AvailableModels 8, got %d", summary.AvailableModels)
	}

	if summary.FailedModels != 2 {
		t.Errorf("Expected FailedModels 2, got %d", summary.FailedModels)
	}

	if summary.Duration != 5*time.Minute {
		t.Errorf("Expected Duration 5m, got %v", summary.Duration)
	}

	if summary.AverageScore != 91.7 {
		t.Errorf("Expected AverageScore 91.7, got %f", summary.AverageScore)
	}

	if len(summary.TopPerformers) != 3 {
		t.Errorf("Expected 3 top performers, got %d", len(summary.TopPerformers))
	}
}

func TestModels_EdgeCases(t *testing.T) {
	t.Run("empty verification result", func(t *testing.T) {
		result := VerificationResult{
			Timestamp: time.Now(),
		}

		if result.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set")
		}
	})

	t.Run("zero values", func(t *testing.T) {
		result := AvailabilityResult{
			Exists:      false,
			Responsive:  false,
			Overloaded:  false,
			Latency:     0,
		}

		if result.Exists {
			t.Error("Expected Exists to be false")
		}

		if result.Latency != 0 {
			t.Errorf("Expected Latency 0, got %v", result.Latency)
		}
	})

	t.Run("maximum values", func(t *testing.T) {
		score := PerformanceScore{
			OverallScore:     100.0,
			CodeCapability:   100.0,
			Responsiveness:   100.0,
			Reliability:      100.0,
			FeatureRichness:  100.0,
			ValueProposition: 100.0,
		}

		if score.OverallScore != 100.0 {
			t.Errorf("Expected OverallScore 100.0, got %f", score.OverallScore)
		}
	})
}

func TestModels_ScoreDetails_Fields(t *testing.T) {
	codeBreakdown := CodeCapabilityBreakdown{
		GenerationScore:    95.0,
		CompletionScore:    92.0,
		DebuggingScore:     90.0,
		ReviewScore:        88.0,
		TestGenScore:       85.0,
		DocumentScore:      87.0,
		ArchitectureScore:  89.0,
		OptimizationScore:  91.0,
		ComplexityHandling: 86.0,
		WeightedAverage:    89.0,
	}

	responseBreakdown := ResponseTimeBreakdown{
		LatencyScore:     94.0,
		ThroughputScore:  92.0,
		ConsistencyScore: 90.0,
		WeightedAverage:  92.0,
	}

	featureBreakdown := FeatureSupportBreakdown{
		CoreFeaturesScore:         96.0,
		AdvancedFeaturesScore:     88.0,
		ExperimentalFeaturesScore: 75.0,
		WeightedAverage:           86.3,
	}

	reliabilityBreakdown := ReliabilityBreakdown{
		AvailabilityScore: 98.0,
		ConsistencyScore:  95.0,
		ErrorRateScore:    92.0,
		StabilityScore:    94.0,
		WeightedAverage:   94.75,
	}

	scoreDetails := ScoreDetails{
		CodeCapabilityBreakdown: codeBreakdown,
		ResponseTimeBreakdown:   responseBreakdown,
		FeatureSupportBreakdown: featureBreakdown,
		ReliabilityBreakdown:    reliabilityBreakdown,
	}

	// Test all breakdown scores
	if scoreDetails.CodeCapabilityBreakdown.GenerationScore != 95.0 {
		t.Errorf("Expected GenerationScore 95.0, got %f", scoreDetails.CodeCapabilityBreakdown.GenerationScore)
	}

	if scoreDetails.ResponseTimeBreakdown.LatencyScore != 94.0 {
		t.Errorf("Expected LatencyScore 94.0, got %f", scoreDetails.ResponseTimeBreakdown.LatencyScore)
	}

	if scoreDetails.FeatureSupportBreakdown.CoreFeaturesScore != 96.0 {
		t.Errorf("Expected CoreFeaturesScore 96.0, got %f", scoreDetails.FeatureSupportBreakdown.CoreFeaturesScore)
	}

	if scoreDetails.ReliabilityBreakdown.AvailabilityScore != 98.0 {
		t.Errorf("Expected AvailabilityScore 98.0, got %f", scoreDetails.ReliabilityBreakdown.AvailabilityScore)
	}
}

func TestModels_OptionalFields(t *testing.T) {
	// Test optional fields when they are nil/not set
	result := VerificationResult{
		ModelInfo: ModelInfo{
			ID:      "test",
			Root:    "", // Empty optional field
			Parent:  "", // Empty optional field
		},
		Availability: AvailabilityResult{
			Exists: true,
			Error: "", // Empty optional error field
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:         true,
			MaxParallelCalls: 0, // Zero value for optional field
		},
		Timestamp: time.Now(),
	}

	// Should not panic when marshaling/unmarshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result with optional fields: %v", err)
	}

	var unmarshaled VerificationResult
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal result with optional fields: %v", err)
	}

	if unmarshaled.ModelInfo.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", unmarshaled.ModelInfo.ID)
	}
}