package llmverifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GenerateMarkdownReport generates a human-readable markdown report
func (v *Verifier) GenerateMarkdownReport(results []VerificationResult, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	reportPath := filepath.Join(outputDir, "llm_verification_report.md")
	file, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	// Write report header
	fmt.Fprintf(file, "# %s\n\n", tr("report.title"))
	fmt.Fprintf(file, "%s %s\n\n", tr("report.generated_on"), time.Now().Format("2006-01-02 15:04:05"))

	// Generate summary
	summary := v.generateSummary(results)
	v.writeSummary(file, summary)

	// Write individual model reports
	for _, result := range results {
		if result.Error != "" {
			v.writeFailedModelReport(file, result)
		} else {
			v.writeModelReport(file, result)
		}
	}

	// Write category rankings
	v.writeCategoryRankings(file, results)

	return nil
}

// GenerateJSONReport generates a JSON report for programmatic use
func (v *Verifier) GenerateJSONReport(results []VerificationResult, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Add summary to results
	summary := v.generateSummary(results)
	jsonReport := map[string]interface{}{
		"summary": summary,
		"results": results,
		"metadata": map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"total_models": len(results),
		},
	}

	jsonPath := filepath.Join(outputDir, "llm_verification_report.json")
	file, err := os.Create(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON report file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonReport)
}

// generateSummary creates a summary of the verification results
func (v *Verifier) generateSummary(results []VerificationResult) Summary {
	startTime := v.GetStartTime()
	endTime := v.GetEndTime()

	summary := Summary{
		TotalModels:     len(results),
		StartTime:       startTime,
		EndTime:         endTime,
		Duration:        endTime.Sub(startTime),
		AvailableModels: 0,
		FailedModels:    0,
	}

	var totalScore float64
	var allPerformers []TopPerformer
	var brotliSupported int

	for i, result := range results {
		if result.Error != "" {
			summary.FailedModels++
		} else {
			summary.AvailableModels++
			totalScore += result.PerformanceScores.OverallScore

			// Count Brotli support
			if result.FeatureDetection.SupportsBrotli {
				brotliSupported++
			}

			// Add to performers list for rankings
			allPerformers = append(allPerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.OverallScore,
				Rank:      i + 1, // Will be re-ranked later
			})
		}
	}

	if summary.AvailableModels > 0 {
		summary.AverageScore = totalScore / float64(summary.AvailableModels)
		summary.BrotliSupportRate = float64(brotliSupported) / float64(summary.AvailableModels) * 100
	} else {
		summary.AverageScore = 0
		summary.BrotliSupportRate = 0
	}

	// Generate category rankings
	summary.CategoryRankings = v.generateCategoryRankings(results)

	return summary
}

// generateCategoryRankings generates rankings by different categories
func (v *Verifier) generateCategoryRankings(results []VerificationResult) CategoryRankings {
	rankings := CategoryRankings{}

	// Overall score rankings
	overallPerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			overallPerformers = append(overallPerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.OverallScore,
			})
		}
	}
	sort.Slice(overallPerformers, func(i, j int) bool {
		return overallPerformers[i].Score > overallPerformers[j].Score
	})
	// Assign ranks
	for i := range overallPerformers {
		overallPerformers[i].Rank = i + 1
	}
	rankings.ByCodeCapability = overallPerformers

	// Code capability rankings
	codePerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			codePerformers = append(codePerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.CodeCapability,
			})
		}
	}
	sort.Slice(codePerformers, func(i, j int) bool {
		return codePerformers[i].Score > codePerformers[j].Score
	})
	for i := range codePerformers {
		codePerformers[i].Rank = i + 1
	}
	rankings.ByCodeCapability = codePerformers

	// Responsiveness rankings
	responsivenessPerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			responsivenessPerformers = append(responsivenessPerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.Responsiveness,
			})
		}
	}
	sort.Slice(responsivenessPerformers, func(i, j int) bool {
		return responsivenessPerformers[i].Score > responsivenessPerformers[j].Score
	})
	for i := range responsivenessPerformers {
		responsivenessPerformers[i].Rank = i + 1
	}
	rankings.ByResponsiveness = responsivenessPerformers

	// Reliability rankings
	reliabilityPerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			reliabilityPerformers = append(reliabilityPerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.Reliability,
			})
		}
	}
	sort.Slice(reliabilityPerformers, func(i, j int) bool {
		return reliabilityPerformers[i].Score > reliabilityPerformers[j].Score
	})
	for i := range reliabilityPerformers {
		reliabilityPerformers[i].Rank = i + 1
	}
	rankings.ByReliability = reliabilityPerformers

	// Feature richness rankings
	featurePerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			featurePerformers = append(featurePerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.FeatureRichness,
			})
		}
	}
	sort.Slice(featurePerformers, func(i, j int) bool {
		return featurePerformers[i].Score > featurePerformers[j].Score
	})
	for i := range featurePerformers {
		featurePerformers[i].Rank = i + 1
	}
	rankings.ByFeatureRichness = featurePerformers

	// Value rankings
	valuePerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			valuePerformers = append(valuePerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.ValueProposition,
			})
		}
	}
	sort.Slice(valuePerformers, func(i, j int) bool {
		return valuePerformers[i].Score > valuePerformers[j].Score
	})
	for i := range valuePerformers {
		valuePerformers[i].Rank = i + 1
	}
	rankings.ByValue = valuePerformers

	return rankings
}

// writeSummary writes the summary section of the report
func (v *Verifier) writeSummary(file *os.File, summary Summary) {
	fmt.Fprintf(file, "## %s\n\n", tr("report.section.summary"))
	fmt.Fprintf(file, "- %s %d\n", tr("report.summary.total_models"), summary.TotalModels)
	fmt.Fprintf(file, "- %s %d\n", tr("report.summary.available_models"), summary.AvailableModels)
	fmt.Fprintf(file, "- %s %d\n", tr("report.summary.failed_models"), summary.FailedModels)
	fmt.Fprintf(file, "- %s %.2f\n", tr("report.summary.average_overall_score"), summary.AverageScore)
	fmt.Fprintf(file, "- %s %.2f%%\n", tr("report.summary.brotli_support_rate"), summary.BrotliSupportRate)
	fmt.Fprintf(file, "\n")

	// Show top performers by overall score
	fmt.Fprintf(file, "### %s\n\n", tr("report.section.top_performers_overall"))
	for i, performer := range summary.CategoryRankings.ByCodeCapability {
		if i >= 5 { // Show top 5
			break
		}
		fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelName, performer.Score)
	}
	fmt.Fprintf(file, "\n")
}

// writeModelReport writes the report for a single successfully verified model
func (v *Verifier) writeModelReport(file *os.File, result VerificationResult) {
	fmt.Fprintf(file, "## %s %s\n\n", tr("report.model.label"), result.ModelInfo.ID)

	// Basic information
	fmt.Fprintf(file, "### %s\n", tr("report.section.basic_information"))
	fmt.Fprintf(file, "- **%s**: %s\n", tr("report.field.endpoint"), result.ModelInfo.Endpoint)
	fmt.Fprintf(file, "- **%s**: %s\n", tr("report.field.verified_at"), result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "\n")

	// Scores
	fmt.Fprintf(file, "### %s\n", tr("report.section.performance_scores"))
	fmt.Fprintf(file, "- **%s**: %.2f\n", tr("report.field.overall_score"), result.PerformanceScores.OverallScore)
	fmt.Fprintf(file, "- **%s**: %.2f\n", tr("report.field.code_capability"), result.PerformanceScores.CodeCapability)
	fmt.Fprintf(file, "- **%s**: %.2f\n", tr("report.field.responsiveness"), result.PerformanceScores.Responsiveness)
	fmt.Fprintf(file, "- **%s**: %.2f\n", tr("report.field.reliability"), result.PerformanceScores.Reliability)
	fmt.Fprintf(file, "- **%s**: %.2f\n", tr("report.field.feature_richness"), result.PerformanceScores.FeatureRichness)
	fmt.Fprintf(file, "- **%s**: %.2f\n", tr("report.field.value_proposition"), result.PerformanceScores.ValueProposition)
	fmt.Fprintf(file, "\n")

	// Availability
	fmt.Fprintf(file, "### %s\n", tr("report.section.availability"))
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.exists"), result.Availability.Exists)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.responsive"), result.Availability.Responsive)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.overloaded"), result.Availability.Overloaded)
	fmt.Fprintf(file, "- **%s**: %s\n", tr("report.field.response_time"), result.Availability.Latency.String())
	fmt.Fprintf(file, "\n")

	// Response time metrics
	if result.ResponseTime.AverageLatency > 0 {
		fmt.Fprintf(file, "### %s\n", tr("report.section.response_time_metrics"))
		fmt.Fprintf(file, "- **%s**: %s\n", tr("report.field.average_latency"), result.ResponseTime.AverageLatency.String())
		// fmt.Fprintf(file, "- **P95 Latency**: %s\n", result.ResponseTime.P95Latency.String())  // We're not calculating P95 currently
		fmt.Fprintf(file, "- **%s**: %s\n", tr("report.field.throughput"), trData("report.value.requests_per_sec", map[string]any{"value": result.ResponseTime.Throughput}))
		fmt.Fprintf(file, "\n")
	}

	// Features
	fmt.Fprintf(file, "### %s\n", tr("report.section.supported_features"))
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.tool_use"), result.FeatureDetection.ToolUse)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.function_calling"), result.FeatureDetection.FunctionCalling)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.code_generation"), result.FeatureDetection.CodeGeneration)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.code_completion"), result.FeatureDetection.CodeCompletion)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.code_explanation"), result.FeatureDetection.CodeExplanation)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.code_review"), result.FeatureDetection.CodeReview)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.embeddings"), result.FeatureDetection.Embeddings)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.reranking"), result.FeatureDetection.Reranking)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.image_generation"), result.FeatureDetection.ImageGeneration)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.audio_generation"), result.FeatureDetection.AudioGeneration)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.video_generation"), result.FeatureDetection.VideoGeneration)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.mcps"), result.FeatureDetection.MCPs)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.lsps"), result.FeatureDetection.LSPs)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.acps"), result.FeatureDetection.ACPs)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.multimodal"), result.FeatureDetection.Multimodal)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.streaming"), result.FeatureDetection.Streaming)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.json_mode"), result.FeatureDetection.JSONMode)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.structured_output"), result.FeatureDetection.StructuredOutput)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.reasoning"), result.FeatureDetection.Reasoning)
	fmt.Fprintf(file, "- **%s**: %s\n", tr("report.field.parallel_tool_use"),
		trData("report.value.parallel_tool_use", map[string]any{
			"enabled": result.FeatureDetection.ParallelToolUse,
			"max":     result.FeatureDetection.MaxParallelCalls,
		}))
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.brotli_compression"), result.FeatureDetection.SupportsBrotli)
	fmt.Fprintf(file, "\n")

	// Code capabilities
	fmt.Fprintf(file, "### %s\n", tr("report.section.code_capabilities"))
	fmt.Fprintf(file, "- **%s**: %s\n", tr("report.field.language_support"), strings.Join(result.CodeCapabilities.LanguageSupport, ", "))
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.code_generation"), result.CodeCapabilities.CodeGeneration)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.code_completion"), result.CodeCapabilities.CodeCompletion)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.code_debugging"), result.CodeCapabilities.CodeDebugging)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.code_optimization"), result.CodeCapabilities.CodeOptimization)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.code_review"), result.CodeCapabilities.CodeReview)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.test_generation"), result.CodeCapabilities.TestGeneration)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.documentation"), result.CodeCapabilities.Documentation)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.refactoring"), result.CodeCapabilities.Refactoring)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.error_resolution"), result.CodeCapabilities.ErrorResolution)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.architecture_understanding"), result.CodeCapabilities.Architecture)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.security_assessment"), result.CodeCapabilities.SecurityAssessment)
	fmt.Fprintf(file, "- **%s**: %t\n", tr("report.field.pattern_recognition"), result.CodeCapabilities.PatternRecognition)

	// Complexity handling
	fmt.Fprintf(file, "- **%s**: %d/5\n", tr("report.field.complexity_level"), result.CodeCapabilities.ComplexityHandling.MaxHandledDepth)
	fmt.Fprintf(file, "- **%s**: %.2f\n", tr("report.field.code_quality_score"), result.CodeCapabilities.ComplexityHandling.CodeQuality)
	fmt.Fprintf(file, "- **%s**: %.2f\n", tr("report.field.logic_correctness_score"), result.CodeCapabilities.ComplexityHandling.LogicCorrectness)
	fmt.Fprintf(file, "- **%s**: %.2f\n", tr("report.field.runtime_efficiency_score"), result.CodeCapabilities.ComplexityHandling.RuntimeEfficiency)
	fmt.Fprintf(file, "\n")

	// Language-specific scores
	fmt.Fprintf(file, "### %s\n", tr("report.section.language_specific_performance"))
	fmt.Fprintf(file, "- **%s**: %.2f%%\n", tr("report.field.python_success_rate"), result.CodeCapabilities.PromptResponse.PythonSuccessRate)
	fmt.Fprintf(file, "- **%s**: %.2f%%\n", tr("report.field.javascript_success_rate"), result.CodeCapabilities.PromptResponse.JavascriptSuccessRate)
	fmt.Fprintf(file, "- **%s**: %.2f%%\n", tr("report.field.go_success_rate"), result.CodeCapabilities.PromptResponse.GoSuccessRate)
	fmt.Fprintf(file, "- **%s**: %.2f%%\n", tr("report.field.java_success_rate"), result.CodeCapabilities.PromptResponse.JavaSuccessRate)
	fmt.Fprintf(file, "- **%s**: %.2f%%\n", tr("report.field.cpp_success_rate"), result.CodeCapabilities.PromptResponse.CppSuccessRate)
	fmt.Fprintf(file, "- **%s**: %.2f%%\n", tr("report.field.typescript_success_rate"), result.CodeCapabilities.PromptResponse.TypescriptSuccessRate)
	fmt.Fprintf(file, "- **%s**: %.2f%%\n", tr("report.field.overall_success_rate"), result.CodeCapabilities.PromptResponse.OverallSuccessRate)
	fmt.Fprintf(file, "\n")
}

// writeFailedModelReport writes the report for a model that failed verification
func (v *Verifier) writeFailedModelReport(file *os.File, result VerificationResult) {
	fmt.Fprintf(file, "## %s %s (%s)\n\n", tr("report.model.label"), result.ModelInfo.ID, tr("report.model.failed"))
	fmt.Fprintf(file, "**%s**: %s\n\n", tr("report.field.error"), result.Error)
	fmt.Fprintf(file, "- **%s**: %s\n", tr("report.field.endpoint"), result.ModelInfo.Endpoint)
	fmt.Fprintf(file, "- **%s**: %s\n", tr("report.field.attempted_at"), result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "\n")
}

// writeCategoryRankings writes the category-wise rankings
func (v *Verifier) writeCategoryRankings(file *os.File, results []VerificationResult) {
	fmt.Fprintf(file, "## %s\n\n", tr("report.section.category_rankings"))

	// Create sorted lists for each category
	overallSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.OverallScore })
	codeSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.CodeCapability })
	responsivenessSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.Responsiveness })
	reliabilitySorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.Reliability })
	featureSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.FeatureRichness })
	valueSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.ValueProposition })

	// Overall rankings
	fmt.Fprintf(file, "### %s\n", tr("report.section.overall_performance"))
	for i, performer := range overallSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.OverallScore)
		}
	}
	fmt.Fprintf(file, "\n")

	// Code Capability Rankings
	fmt.Fprintf(file, "### %s\n", tr("report.section.by_code_capability"))
	for i, performer := range codeSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.CodeCapability)
		}
	}
	fmt.Fprintf(file, "\n")

	// Responsiveness Rankings
	fmt.Fprintf(file, "### %s\n", tr("report.section.by_responsiveness"))
	for i, performer := range responsivenessSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.Responsiveness)
		}
	}
	fmt.Fprintf(file, "\n")

	// Reliability Rankings
	fmt.Fprintf(file, "### %s\n", tr("report.section.by_reliability"))
	for i, performer := range reliabilitySorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.Reliability)
		}
	}
	fmt.Fprintf(file, "\n")

	// Feature Richness Rankings
	fmt.Fprintf(file, "### %s\n", tr("report.section.by_feature_richness"))
	for i, performer := range featureSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.FeatureRichness)
		}
	}
	fmt.Fprintf(file, "\n")

	// Value Proposition Rankings
	fmt.Fprintf(file, "### %s\n", tr("report.section.by_value_proposition"))
	for i, performer := range valueSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.ValueProposition)
		}
	}
	fmt.Fprintf(file, "\n")
}

// SortResultsByScore sorts verification results by a given score function
func (v *Verifier) SortResultsByScore(results []VerificationResult, scoreFunc func(VerificationResult) float64) []VerificationResult {
	// Create a copy to sort
	sorted := make([]VerificationResult, len(results))
	copy(sorted, results)

	// Sort by the provided score function in descending order
	sort.Slice(sorted, func(i, j int) bool {
		return scoreFunc(sorted[i]) > scoreFunc(sorted[j])
	})

	return sorted
}
