package verification

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"digital.vasic.llmsverifier/client"
	"digital.vasic.llmsverifier/logging"
)

// randomIDSuffix returns a short, collision-resistant suffix for verification
// IDs. 8 bytes from crypto/rand (base64url ~11 chars); on RNG failure falls back
// to a doubled-nanosecond value so the suffix is non-empty and varies.
func randomIDSuffix() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano()*2654435761)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// newVerificationID builds a unique verification-run ID. The pre-fix form
// `fmt.Sprintf("<prefix>_%s_%s_%d", providerID, modelID, time.Now().Unix())`
// used second-resolution time, so re-verifying the SAME model within the same
// second (retries, back-to-back runs, or concurrent VerifyAll) produced a
// duplicate verification_id — and that ID is persisted into the result record
// + `model.Features["verification_id"]`, so a collision aliases two distinct
// runs. The crypto/rand suffix guarantees uniqueness. Named helper so its
// uniqueness is directly testable (§11.4.50 / §11.4.115).
func newVerificationID(prefix, providerID, modelID string) string {
	return fmt.Sprintf("%s_%s_%s_%d_%s", prefix, providerID, modelID, time.Now().UnixNano(), randomIDSuffix())
}

// CodeVerificationService handles mandatory code visibility verification for models
type CodeVerificationService struct {
	httpClient *client.HTTPClient
	logger     *logging.Logger
}

// CodeVerificationRequest represents a code verification test request
type CodeVerificationRequest struct {
	ModelID    string `json:"model_id"`
	ProviderID string `json:"provider_id"`
	Code       string `json:"code"`
	Language   string `json:"language"`
}

// CodeVerificationResponse represents the result of a code verification test
type CodeVerificationResponse struct {
	ModelID             string    `json:"model_id"`
	ProviderID          string    `json:"provider_id"`
	Verified            bool      `json:"verified"`
	Response            string    `json:"response"`
	CanSeeCode          bool      `json:"can_see_code"`
	AffirmativeResponse bool      `json:"affirmative_response"`
	CodeUnderstanding   float64   `json:"code_understanding"`
	ResponseTime        int64     `json:"response_time_ms"`
	Error               string    `json:"error,omitempty"`
	TestTimestamp       time.Time `json:"test_timestamp"`
}

// CodeVerificationResult represents a comprehensive verification result
type CodeVerificationResult struct {
	VerificationID          string               `json:"verification_id"`
	ModelID                 string               `json:"model_id"`
	ProviderID              string               `json:"provider_id"`
	Status                  string               `json:"status"` // pending, verified, failed, error
	CodeVisibility          bool                 `json:"code_visibility"`
	ToolSupport             bool                 `json:"tool_support"`
	AffirmativeConfirmation bool                 `json:"affirmative_confirmation"`
	ResponseAnalysis        CodeResponseAnalysis `json:"response_analysis"`
	VerificationScore       float64              `json:"verification_score"`
	TestedAt                time.Time            `json:"tested_at"`
	CompletedAt             *time.Time           `json:"completed_at,omitempty"`
	ErrorMessage            string               `json:"error_message,omitempty"`
}

// CodeResponseAnalysis analyzes the model's response to code visibility questions
type CodeResponseAnalysis struct {
	ContainsAffirmative bool     `json:"contains_affirmative"`
	ContainsNegative    bool     `json:"contains_negative"`
	CodeReferences      []string `json:"code_references"`
	LanguageDetection   string   `json:"language_detection"`
	UnderstandingLevel  string   `json:"understanding_level"` // none, basic, intermediate, advanced
	ConfidenceScore     float64  `json:"confidence_score"`
}

// NewCodeVerificationService creates a new code verification service
func NewCodeVerificationService(httpClient *client.HTTPClient, logger *logging.Logger) *CodeVerificationService {
	return &CodeVerificationService{
		httpClient: httpClient,
		logger:     logger,
	}
}

// VerifyModelCodeVisibility performs mandatory code visibility verification for a specific model
func (cvs *CodeVerificationService) VerifyModelCodeVisibility(ctx context.Context, modelID, providerID string, providerClient ProviderClientInterface) (*CodeVerificationResult, error) {
	verificationID := newVerificationID("code_verify", providerID, modelID)

	result := &CodeVerificationResult{
		VerificationID: verificationID,
		ModelID:        modelID,
		ProviderID:     providerID,
		Status:         "pending",
		TestedAt:       time.Now(),
	}

	if cvs.logger != nil {
		cvs.logger.Info(fmt.Sprintf("Starting code visibility verification for model %s from provider %s", modelID, providerID), map[string]interface{}{
			"verification_id": verificationID,
			"model_id":        modelID,
			"provider_id":     providerID,
		})
	}

	if providerClient == nil {
		result.Status = "error"
		result.ErrorMessage = "Provider client cannot be nil"
		result.CompletedAt = ptrTime(time.Now())
		return result, fmt.Errorf("provider client cannot be nil")
	}

	// Test code visibility with different code samples
	codeSamples := cvs.getTestCodeSamples()
	verificationResponses := make([]CodeVerificationResponse, 0)

	for _, sample := range codeSamples {
		response, err := cvs.testCodeVisibility(ctx, providerID, modelID, providerClient, sample)
		if err != nil {
			if cvs.logger != nil {
				cvs.logger.Warning(fmt.Sprintf("Failed to test code visibility for sample %s: %v", sample.Language, err), map[string]interface{}{
					"model_id":    modelID,
					"provider_id": providerID,
					"language":    sample.Language,
				})
			}
			continue
		}
		verificationResponses = append(verificationResponses, *response)
	}

	// Analyze all responses
	result.ResponseAnalysis = cvs.analyzeVerificationResponses(verificationResponses)
	// STRICT VERIFICATION: code visibility requires real confidence (> 0.5),
	// which a bare affirmative without any code-specific content cannot reach.
	// A model that merely responds — or parrots "Yes, I can see it" — does NOT
	// demonstrate code visibility and must fail the gate.
	result.CodeVisibility = result.ResponseAnalysis.ConfidenceScore > 0.5
	result.AffirmativeConfirmation = result.ResponseAnalysis.ContainsAffirmative
	// Report the real computed score. There is intentionally NO score floor —
	// not even a conditional one: flooring to 0.7 (in any form) makes every
	// strict gate downstream of VerificationScore vacuous and certifies
	// sub-threshold models as high-scoring. A code-blind, verification-failed
	// model keeps its true (low) measured confidence; a visible model reports
	// its real confidence, which a strict >= 0.7 gate can meaningfully check.
	result.VerificationScore = result.ResponseAnalysis.ConfidenceScore

	// Apply timeout penalty for slow responses
	avgResponseTime := cvs.calculateAverageResponseTime(verificationResponses)
	if adjustedScore, timeoutPenalty, applied := applyVerificationTimeoutPenalty(result.VerificationScore, avgResponseTime, result.CodeVisibility); applied {
		result.VerificationScore = adjustedScore
		if cvs.logger != nil {
			cvs.logger.Warning(fmt.Sprintf("Applied timeout penalty to model %s: avg response time=%dms, penalty=%.2f, adjusted score=%.2f",
				modelID, avgResponseTime, timeoutPenalty, result.VerificationScore), nil)
		}
	}

	// STATUS DETERMINATION: a model is "verified" ONLY when its code
	// visibility was actually established. A model that returned responses
	// but did NOT demonstrate code visibility (CodeVisibility == false —
	// e.g. it answered "No, I cannot see your code" to every sample) is the
	// canonical negative case this gate exists to catch and MUST be "failed".
	// Certifying such a model as "verified" is a §11.4 / §11.4.1 PASS-bluff:
	// the gate's non-strict IsModelVerified treats any non-"error" status as
	// verified, so a code-blind model would silently pass the gate.
	if len(verificationResponses) == 0 {
		result.Status = "failed"
		result.ErrorMessage = "No successful verification responses"
	} else if result.CodeVisibility {
		result.Status = "verified"
	} else {
		result.Status = "failed"
		result.ErrorMessage = "Model responded but did not demonstrate code visibility"
	}

	completedAt := time.Now()
	result.CompletedAt = &completedAt

	if cvs.logger != nil {
		cvs.logger.Info(fmt.Sprintf("Code visibility verification completed for model %s: %s (score: %.2f)",
			modelID, result.Status, result.VerificationScore), map[string]interface{}{
			"verification_id": verificationID,
			"model_id":        modelID,
			"provider_id":     providerID,
			"status":          result.Status,
			"score":           result.VerificationScore,
			"code_visibility": result.CodeVisibility,
		})
	}

	return result, nil
}

// Helper function to get max of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// applyVerificationTimeoutPenalty computes the timeout-penalised
// VerificationScore for a slow model. Extracted to a pure function so the
// score-floor-for-a-failed-model bug class (fixed above for the initial 0.7
// floor) is independently unit-testable without a real 20s+ HTTP round trip.
//
// The 0.3 post-penalty floor exists to avoid needlessly zeroing out a model
// that DID demonstrate code visibility (codeVisible == true) but was merely
// slow to respond. It MUST NOT be applied when the model never demonstrated
// code visibility in the first place (codeVisible == false) -- otherwise a
// blind-AND-slow model (score already 0.0) would be bumped from 0.0 up to
// 0.3 by the very act of being slow, the identical inflate-a-failed-model's
// score bluff. For codeVisible == false the penalised score is clamped at 0
// (a score can never go negative) instead of floored at 0.3.
//
// Returns the adjusted score, the penalty magnitude that was applied (for
// logging), and whether any penalty was applied at all (avgResponseTimeMs <=
// 20000 is a no-op).
func applyVerificationTimeoutPenalty(score float64, avgResponseTimeMs int64, codeVisible bool) (adjusted float64, penalty float64, applied bool) {
	if avgResponseTimeMs <= 20000 {
		return score, 0, false
	}

	switch {
	case avgResponseTimeMs > 60000:
		penalty = 0.4
	case avgResponseTimeMs > 45000:
		penalty = 0.3
	case avgResponseTimeMs > 30000:
		penalty = 0.2
	default:
		penalty = 0.1
	}

	adjusted = score - penalty
	if codeVisible {
		adjusted = max(adjusted, 0.3)
	} else if adjusted < 0 {
		adjusted = 0
	}
	return adjusted, penalty, true
}

// TestCodeSample represents a code sample for testing
type TestCodeSample struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	Purpose  string `json:"purpose"`
}

// testCodeVisibility tests if a model can see and understand code
func (cvs *CodeVerificationService) testCodeVisibility(ctx context.Context, providerID, modelID string, providerClient ProviderClientInterface, sample TestCodeSample) (*CodeVerificationResponse, error) {
	startTime := time.Now()

	// Create the verification prompt
	prompt := cvs.createCodeVerificationPrompt(sample)

	// Make API request to model
	response, err := cvs.makeVerificationRequest(ctx, providerClient, modelID, prompt)
	if err != nil {
		return &CodeVerificationResponse{
			ModelID:       modelID,
			ProviderID:    providerID,
			Verified:      false,
			Error:         err.Error(),
			ResponseTime:  time.Since(startTime).Milliseconds(),
			TestTimestamp: time.Now(),
		}, err // Propagate error so caller can distinguish API failure from negative verification
	}

	responseTime := time.Since(startTime).Milliseconds()

	// Analyze the response
	analysis := cvs.analyzeCodeResponse(response, sample)

	verificationResponse := &CodeVerificationResponse{
		ModelID:             modelID,
		ProviderID:          providerID,
		Verified:            analysis.ContainsAffirmative && !analysis.ContainsNegative,
		Response:            response,
		CanSeeCode:          analysis.ContainsAffirmative,
		AffirmativeResponse: analysis.ContainsAffirmative,
		CodeUnderstanding:   analysis.ConfidenceScore,
		ResponseTime:        responseTime,
		TestTimestamp:       time.Now(),
	}

	return verificationResponse, nil
}

// createCodeVerificationPrompt creates a prompt to test code visibility
func (cvs *CodeVerificationService) createCodeVerificationPrompt(sample TestCodeSample) string {
	return fmt.Sprintf("Do you see my code? Please respond with \"Yes, I can see your %s code\" if you can see the code below, or \"No, I cannot see your code\" if you cannot see it.\n\n%s code:\n```%s\n%s\n```\n\nPlease confirm if you can see this code and understand what it does.", sample.Language, strings.Title(sample.Language), sample.Language, sample.Code)
}

// makeVerificationRequest makes an API request to the model
func (cvs *CodeVerificationService) makeVerificationRequest(ctx context.Context, providerClient ProviderClientInterface, modelID, prompt string) (string, error) {
	// Create the request payload
	requestPayload := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  150,
		"temperature": 0.1,
	}

	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Make HTTP request with the provider Authorization header. Without the
	// Bearer key the provider rejects the call (401), so verification could
	// never pass even with a valid key — the auth header is mandatory.
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/chat/completions", providerClient.GetBaseURL()),
		strings.NewReader(string(jsonPayload)),
	)
	if err != nil {
		return "", fmt.Errorf("failed to build API request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := providerClient.GetAPIKey(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := providerClient.GetHTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return "", fmt.Errorf("failed to decode API response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("no response choices available")
	}

	return apiResponse.Choices[0].Message.Content, nil
}

// affirmativePatterns / negativePatterns match whole words/phrases only (\b
// word boundaries). Bare substring matching let the keyword "no" match inside
// "not", "know", and "cannot", so affirmative answers containing those words
// were misclassified as negative — and, conversely, a denial like "I cannot
// see it" contains the substring "can see" only across the "cannot" boundary,
// which word boundaries reject.
var (
	affirmativePatterns = compileKeywordPatterns([]string{"yes", "i can see", "i see", "visible", "can see"})
	negativePatterns    = compileKeywordPatterns([]string{"no", "cannot see", "can't see", "not visible", "do not see"})
)

// compileKeywordPatterns wraps each keyword in \b word boundaries so matching
// is whole-word/whole-phrase. regexp.QuoteMeta keeps multi-word phrases and
// apostrophes literal.
func compileKeywordPatterns(keywords []string) []*regexp.Regexp {
	patterns := make([]*regexp.Regexp, 0, len(keywords))
	for _, keyword := range keywords {
		patterns = append(patterns, regexp.MustCompile(`\b`+regexp.QuoteMeta(keyword)+`\b`))
	}
	return patterns
}

// matchesAnyPattern reports whether any of the precompiled word-boundary
// patterns matches the (already lower-cased) response.
func matchesAnyPattern(responseLower string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(responseLower) {
			return true
		}
	}
	return false
}

// analyzeCodeResponse analyzes the model's response for code visibility confirmation
func (cvs *CodeVerificationService) analyzeCodeResponse(response string, sample TestCodeSample) CodeResponseAnalysis {
	responseLower := strings.ToLower(response)

	// Check for affirmative responses (whole-word/phrase matching)
	containsAffirmative := matchesAnyPattern(responseLower, affirmativePatterns)

	// Check for negative responses (whole-word/phrase matching)
	containsNegative := matchesAnyPattern(responseLower, negativePatterns)

	// Detect code references
	codeReferences := cvs.extractCodeReferences(response, sample)

	// Detect language understanding
	languageDetection := cvs.detectLanguageUnderstanding(response, sample.Language)

	// Calculate understanding level
	understandingLevel := cvs.calculateUnderstandingLevel(containsAffirmative, len(codeReferences), languageDetection)

	// Calculate confidence score
	confidenceScore := cvs.calculateConfidenceScore(containsAffirmative, containsNegative, len(codeReferences), understandingLevel)

	return CodeResponseAnalysis{
		ContainsAffirmative: containsAffirmative,
		ContainsNegative:    containsNegative,
		CodeReferences:      codeReferences,
		LanguageDetection:   languageDetection,
		UnderstandingLevel:  understandingLevel,
		ConfidenceScore:     confidenceScore,
	}
}

// extractCodeReferences extracts references to code elements from the response
func (cvs *CodeVerificationService) extractCodeReferences(response string, sample TestCodeSample) []string {
	var references []string

	// Look for function names, variable names, etc.
	// This is a simplified implementation
	codeWords := []string{"function", "class", "method", "variable", "return", "if", "else", "for", "while"}

	responseLower := strings.ToLower(response)
	for _, word := range codeWords {
		if strings.Contains(responseLower, word) {
			references = append(references, word)
		}
	}

	return references
}

// detectLanguageUnderstanding detects if the model understands the programming language
func (cvs *CodeVerificationService) detectLanguageUnderstanding(response string, expectedLanguage string) string {
	responseLower := strings.ToLower(response)

	languageKeywords := map[string][]string{
		"python":     {"python", "def", "import", "print"},
		"javascript": {"javascript", "function", "const", "let", "var"},
		"go":         {"go", "func", "package", "import"},
		"java":       {"java", "class", "public", "private"},
		"csharp":     {"csharp", "class", "public", "static"},
	}

	if keywords, exists := languageKeywords[expectedLanguage]; exists {
		for _, keyword := range keywords {
			if strings.Contains(responseLower, keyword) {
				return expectedLanguage
			}
		}
	}

	return "unknown"
}

// calculateUnderstandingLevel calculates the level of code understanding
func (cvs *CodeVerificationService) calculateUnderstandingLevel(affirmative bool, codeRefCount int, languageDetection string) string {
	if !affirmative {
		return "none"
	}

	if languageDetection != "unknown" && codeRefCount >= 3 {
		return "advanced"
	} else if codeRefCount >= 2 {
		return "intermediate"
	} else if codeRefCount >= 1 {
		return "basic"
	}

	return "none"
}

// calculateConfidenceScore calculates a confidence score for the verification.
// Weights are calibrated so a bare affirmative with zero code-specific content
// ("Yes, I can see it") tops out at 0.4 — BELOW the 0.5 strict-verification
// threshold. Reaching 0.5 requires actual evidence: code references and/or a
// demonstrated understanding level.
func (cvs *CodeVerificationService) calculateConfidenceScore(affirmative, negative bool, codeRefCount int, understandingLevel string) float64 {
	score := 0.0

	if affirmative {
		score += 0.3
	}

	if !negative {
		score += 0.1
	}

	// Add score based on code references
	score += float64(codeRefCount) * 0.1
	if score > 0.9 {
		score = 0.9
	}

	// Add score based on understanding level
	switch understandingLevel {
	case "advanced":
		score += 0.3
	case "intermediate":
		score += 0.2
	case "basic":
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// analyzeVerificationResponses analyzes multiple verification responses
func (cvs *CodeVerificationService) analyzeVerificationResponses(responses []CodeVerificationResponse) CodeResponseAnalysis {
	if len(responses) == 0 {
		return CodeResponseAnalysis{
			UnderstandingLevel: "none",
			ConfidenceScore:    0.0,
		}
	}

	totalAffirmative := 0
	totalNegative := 0
	totalCodeRefs := 0
	totalScore := 0.0

	for _, response := range responses {
		if response.AffirmativeResponse {
			totalAffirmative++
		}
		if response.Response != "" {
			lower := strings.ToLower(response.Response)
			// Check for actual negative phrases, not bare "no" which matches "know", "function", etc.
			negativeIndicators := []string{"i cannot see", "not visible", "i don't have access", "no code", "cannot access", "i do not see"}
			for _, indicator := range negativeIndicators {
				if strings.Contains(lower, indicator) {
					totalNegative++
					break
				}
			}
		}
		totalCodeRefs += len(cvs.extractCodeReferences(response.Response, TestCodeSample{}))
		totalScore += response.CodeUnderstanding
	}

	avgScore := totalScore / float64(len(responses))
	containsAffirmative := totalAffirmative > len(responses)/2
	containsNegative := totalNegative > len(responses)/2

	understandingLevel := "none"
	if avgScore >= 0.8 {
		understandingLevel = "advanced"
	} else if avgScore >= 0.6 {
		understandingLevel = "intermediate"
	} else if avgScore >= 0.3 {
		understandingLevel = "basic"
	}

	return CodeResponseAnalysis{
		ContainsAffirmative: containsAffirmative,
		ContainsNegative:    containsNegative,
		CodeReferences:      []string{fmt.Sprintf("avg_refs: %d", totalCodeRefs/len(responses))},
		UnderstandingLevel:  understandingLevel,
		ConfidenceScore:     avgScore,
	}
}

// getTestCodeSamples returns test code samples for verification
func (cvs *CodeVerificationService) getTestCodeSamples() []TestCodeSample {
	return []TestCodeSample{
		{
			Code: `def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

print(fibonacci(10))`,
			Language: "python",
			Purpose:  "Test basic function definition and recursion",
		},
		{
			Code: `function quickSort(arr) {
    if (arr.length <= 1) return arr;
    
    const pivot = arr[Math.floor(arr.length / 2)];
    const left = arr.filter(x => x < pivot);
    const middle = arr.filter(x => x === pivot);
    const right = arr.filter(x => x > pivot);
    
    return [...quickSort(left), ...middle, ...quickSort(right)];
}`,
			Language: "javascript",
			Purpose:  "Test function declaration and array operations",
		},
		{
			Code: `package main

import "fmt"

func main() {
    message := "Hello, World!"
    fmt.Println(message)
}`,
			Language: "go",
			Purpose:  "Test package declaration and basic syntax",
		},
		{
			Code: `public class Calculator {
    public static int add(int a, int b) {
        return a + b;
    }
    
    public static void main(String[] args) {
        int result = add(5, 3);
        System.out.println("Result: " + result);
    }
}`,
			Language: "java",
			Purpose:  "Test class definition and static methods",
		},
		{
			Code: `using System;

class Program {
    static void Main() {
        string name = "LLM Verifier";
        Console.WriteLine($"Hello, {name}!");
    }
}`,
			Language: "csharp",
			Purpose:  "Test class and string interpolation",
		},
	}
}

// Helper function to get pointer to time
func ptrTime(t time.Time) *time.Time {
	return &t
}

// MeaningfulResponseVerificationResult represents the result of verifying a model gives meaningful responses
type MeaningfulResponseVerificationResult struct {
	VerificationID        string    `json:"verification_id"`
	ModelID               string    `json:"model_id"`
	ProviderID            string    `json:"provider_id"`
	Status                string    `json:"status"` // pending, verified, failed, error
	HasMeaningfulResponse bool      `json:"has_meaningful_response"`
	ResponseLength        int       `json:"response_length"`
	ResponseContent       string    `json:"response_content,omitempty"`
	ResponseTimeMs        int64     `json:"response_time_ms"`
	ErrorMessage          string    `json:"error_message,omitempty"`
	VerifiedAt            time.Time `json:"verified_at"`
}

// VerifyMeaningfulResponse tests if the model gives meaningful responses to basic prompts
// A meaningful response must:
// 1. Not be empty
// 2. Not be an error message
// 3. Have at least 5 characters
// 4. Not contain common error indicators
// Implements retry logic for timeouts to distinguish temporary vs persistent issues
func (cvs *CodeVerificationService) VerifyMeaningfulResponse(ctx context.Context, modelID, providerID string, providerClient ProviderClientInterface) (*MeaningfulResponseVerificationResult, error) {
	verificationID := newVerificationID("meaningful_verify", providerID, modelID)

	result := &MeaningfulResponseVerificationResult{
		VerificationID: verificationID,
		ModelID:        modelID,
		ProviderID:     providerID,
		Status:         "pending",
		VerifiedAt:     time.Now(),
	}

	if cvs.logger != nil {
		cvs.logger.Info(fmt.Sprintf("Starting meaningful response verification for model %s from provider %s", modelID, providerID), map[string]interface{}{
			"verification_id": verificationID,
			"model_id":        modelID,
			"provider_id":     providerID,
		})
	}

	if providerClient == nil {
		result.Status = "error"
		result.ErrorMessage = "Provider client cannot be nil"
		return result, fmt.Errorf("provider client cannot be nil")
	}

	const maxRetries = 3
	const timeoutThreshold = 30000
	var lastErr error
	var response string
	var responseTimes []int64

	for attempt := 1; attempt <= maxRetries; attempt++ {
		startTime := time.Now()
		response, lastErr = cvs.makeVerificationRequest(ctx, providerClient, modelID, "hello!")
		responseTime := time.Since(startTime).Milliseconds()
		responseTimes = append(responseTimes, responseTime)

		if lastErr == nil {
			result.ResponseTimeMs = responseTime
			break
		}

		isTimeout := strings.Contains(strings.ToLower(lastErr.Error()), "timeout") ||
			strings.Contains(strings.ToLower(lastErr.Error()), "deadline exceeded") ||
			responseTime >= timeoutThreshold

		if isTimeout && attempt < maxRetries {
			if cvs.logger != nil {
				cvs.logger.Warning(fmt.Sprintf("Timeout on attempt %d/%d for model %s, retrying...", attempt, maxRetries, modelID), map[string]interface{}{
					"model_id":      modelID,
					"provider_id":   providerID,
					"attempt":       attempt,
					"response_time": responseTime,
				})
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		result.ResponseTimeMs = responseTime
		break
	}

	// Only compute average response time when all retries failed;
	// if a successful response was obtained, keep its actual response time.
	if lastErr != nil && len(responseTimes) > 0 {
		var totalTime int64
		for _, rt := range responseTimes {
			totalTime += rt
		}
		result.ResponseTimeMs = totalTime / int64(len(responseTimes))
	}

	if lastErr != nil {
		result.Status = "failed"
		timeoutCount := 0
		for _, rt := range responseTimes {
			if rt >= timeoutThreshold {
				timeoutCount++
			}
		}

		if timeoutCount >= 2 {
			result.ErrorMessage = fmt.Sprintf("Consistent timeout (%d/%d attempts). Model is too slow for reliable use. Avg response time: %dms", timeoutCount, maxRetries, result.ResponseTimeMs)
		} else {
			result.ErrorMessage = fmt.Sprintf("API request failed: %v (avg response time: %dms)", lastErr, result.ResponseTimeMs)
		}

		if cvs.logger != nil {
			cvs.logger.Warning(fmt.Sprintf("Meaningful response verification failed for model %s: %v", modelID, lastErr), map[string]interface{}{
				"model_id":        modelID,
				"provider_id":     providerID,
				"attempts":        len(responseTimes),
				"avg_response_ms": result.ResponseTimeMs,
			})
		}
		return result, nil
	}

	result.ResponseContent = response
	result.ResponseLength = len(response)

	isMeaningful := cvs.validateMeaningfulResponse(response)

	if result.ResponseTimeMs > 20000 && isMeaningful {
		if cvs.logger != nil {
			cvs.logger.Warning(fmt.Sprintf("Model %s responded meaningfully but is slow (%dms), applying timeout penalty", modelID, result.ResponseTimeMs), nil)
		}
	}

	result.HasMeaningfulResponse = isMeaningful

	if isMeaningful {
		result.Status = "verified"
		if cvs.logger != nil {
			cvs.logger.Info(fmt.Sprintf("Meaningful response verification PASSED for model %s: length=%d, time=%dms",
				modelID, result.ResponseLength, result.ResponseTimeMs), map[string]interface{}{
				"model_id":         modelID,
				"provider_id":      providerID,
				"response_length":  result.ResponseLength,
				"response_time_ms": result.ResponseTimeMs,
			})
		}
	} else {
		result.Status = "failed"
		result.ErrorMessage = "Response is empty, too short, or contains error indicators"
		if cvs.logger != nil {
			cvs.logger.Warning(fmt.Sprintf("Meaningful response verification FAILED for model %s: response='%s' (length=%d)",
				modelID, truncateString(response, 100), result.ResponseLength), map[string]interface{}{
				"model_id":        modelID,
				"provider_id":     providerID,
				"response_length": result.ResponseLength,
			})
		}
	}

	return result, nil
}

// validateMeaningfulResponse checks if a response is meaningful
// Returns true if:
// - Response is not empty
// - Response length >= 5 characters
// - Does not contain common error indicators
// - Is not a refusal or error message
func (cvs *CodeVerificationService) validateMeaningfulResponse(response string) bool {
	if response == "" {
		return false
	}

	responseLower := strings.ToLower(strings.TrimSpace(response))

	// Check minimum length (use trimmed response to avoid whitespace-only passing)
	if len(responseLower) < 5 {
		return false
	}

	// Check for common error indicators
	errorIndicators := []string{
		"error:",
		"failed to",
		"cannot",
		"unable to",
		"not available",
		"service unavailable",
		"api error",
		"authentication failed",
		"invalid api",
		"rate limit",
		"timeout",
		"connection error",
		"internal server error",
		"bad gateway",
		"forbidden",
		"unauthorized",
		"not found",
		"null",
		"nil",
		"undefined",
		"[error]",
		"**error**",
		"⚠️",
		"❌",
	}

	for _, indicator := range errorIndicators {
		if strings.Contains(responseLower, indicator) {
			return false
		}
	}

	// Check for refusal patterns
	refusalPatterns := []string{
		"i'm sorry",
		"i cannot",
		"i can't",
		"sorry, i",
		"i'm not able",
		"as an ai",
		"as a language model",
		"i don't have",
		"i do not have",
		"i was not trained",
		"i wasn't trained",
	}

	for _, pattern := range refusalPatterns {
		if strings.Contains(responseLower, pattern) {
			return false
		}
	}

	// Check for placeholder content
	placeholderPatterns := []string{
		"placeholder",
		"todo",
		"tbd",
		"coming soon",
		"not implemented",
		"under construction",
	}

	for _, pattern := range placeholderPatterns {
		if strings.Contains(responseLower, pattern) {
			return false
		}
	}

	return true
}

// truncateString truncates a string to maxLength and adds "..." if truncated
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

// VerifyRealisticDebatePrompt tests if the model can handle realistic debate prompts
// This uses a longer, more complex prompt similar to actual debate usage
// CRITICAL: This catches slow providers that pass simple "hello!" tests but timeout on real prompts
func (cvs *CodeVerificationService) VerifyRealisticDebatePrompt(ctx context.Context, modelID, providerID string, providerClient ProviderClientInterface) (*MeaningfulResponseVerificationResult, error) {
	verificationID := newVerificationID("debate_verify", providerID, modelID)

	result := &MeaningfulResponseVerificationResult{
		VerificationID: verificationID,
		ModelID:        modelID,
		ProviderID:     providerID,
		Status:         "pending",
		VerifiedAt:     time.Now(),
	}

	if cvs.logger != nil {
		cvs.logger.Info(fmt.Sprintf("Starting realistic debate prompt verification for model %s from provider %s", modelID, providerID), map[string]interface{}{
			"verification_id": verificationID,
			"model_id":        modelID,
			"provider_id":     providerID,
		})
	}

	if providerClient == nil {
		result.Status = "error"
		result.ErrorMessage = "Provider client cannot be nil"
		return result, fmt.Errorf("provider client cannot be nil")
	}

	realisticPrompt := `You are part of HelixAgent, an AI coding assistant that provides responses through an AI Debate Ensemble.

IMPORTANT CONTEXT:
- You are integrated with AI coding tools like Claude Code, OpenCode, and Qwen Code
- The user's coding assistant HAS FULL ACCESS to their codebase through tools
- Tools available: Read files, Write/Edit files, Search code (grep), List files (glob), Execute shell commands
- When the user asks about their code, the assistant CAN see and access their files
- NEVER say "I cannot see your codebase" - the tools handle file access
- Provide SPECIFIC, ACTIONABLE coding advice

Your role is THE ARCHITECT: Design system architecture, identify components, define data flows, and plan technology choices. For coding questions, provide architectural guidance on structure, patterns, and best practices.

Topic: How should I structure a Go microservice for user authentication?

Provide your analysis in 2-3 sentences, focused on your role.`

	startTime := time.Now()
	response, err := cvs.makeVerificationRequest(ctx, providerClient, modelID, realisticPrompt)
	result.ResponseTimeMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Realistic prompt failed: %v (time: %dms)", err, result.ResponseTimeMs)
		if cvs.logger != nil {
			cvs.logger.Warning(fmt.Sprintf("Realistic debate prompt verification failed for model %s: %v", modelID, err), map[string]interface{}{
				"model_id":         modelID,
				"provider_id":      providerID,
				"response_time_ms": result.ResponseTimeMs,
			})
		}
		return result, err // Propagate error to caller for proper handling
	}

	result.ResponseContent = response
	result.ResponseLength = len(response)
	result.HasMeaningfulResponse = len(response) > 50

	if result.HasMeaningfulResponse {
		result.Status = "verified"
		if cvs.logger != nil {
			cvs.logger.Info(fmt.Sprintf("Realistic debate prompt verification PASSED for model %s: length=%d, time=%dms",
				modelID, result.ResponseLength, result.ResponseTimeMs), map[string]interface{}{
				"model_id":         modelID,
				"provider_id":      providerID,
				"response_length":  result.ResponseLength,
				"response_time_ms": result.ResponseTimeMs,
			})
		}
	} else {
		result.Status = "failed"
		result.ErrorMessage = "Response too short for realistic prompt"
	}

	return result, nil
}

// calculateAverageResponseTime calculates the average response time from verification responses
func (cvs *CodeVerificationService) calculateAverageResponseTime(responses []CodeVerificationResponse) int64 {
	if len(responses) == 0 {
		return 0
	}

	var totalTime int64
	for _, resp := range responses {
		totalTime += resp.ResponseTime
	}

	return totalTime / int64(len(responses))
}
