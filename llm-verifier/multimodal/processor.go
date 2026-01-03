// Package multimodal provides multi-modal content processing and verification
package multimodal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProviderConfig holds configuration for an LLM provider
type ProviderConfig struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

// MultiModalProcessor handles multi-modal content processing
type MultiModalProcessor struct {
	httpClient     *http.Client
	contentSafety  *ContentSafetyChecker
	imageProcessor *ImageProcessor
	audioProcessor *AudioProcessor
	providers      map[string]*ProviderConfig
	defaultVision  string
	defaultAudio   string
	defaultSafety  string
	mu             sync.RWMutex
}

// ContentType represents the type of multi-modal content
type ContentType string

const (
	ContentTypeImage ContentType = "image"
	ContentTypeAudio ContentType = "audio"
	ContentTypeVideo ContentType = "video"
	ContentTypeText  ContentType = "text"
)

// MultiModalContent represents multi-modal content with metadata
type MultiModalContent struct {
	ID          string                 `json:"id"`
	Type        ContentType            `json:"type"`
	Data        []byte                 `json:"data,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Base64Data  string                 `json:"base64_data,omitempty"`
	MimeType    string                 `json:"mime_type"`
	Filename    string                 `json:"filename,omitempty"`
	Size        int64                  `json:"size"`
	Metadata    map[string]interface{} `json:"metadata"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	TextContent string                 `json:"text_content,omitempty"`
	SafetyScore float64                `json:"safety_score,omitempty"`
	Analysis    *ContentAnalysis       `json:"analysis,omitempty"`
}

// ContentAnalysis represents AI analysis of content
type ContentAnalysis struct {
	Description  string                 `json:"description"`
	Objects      []string               `json:"objects,omitempty"`
	Text         string                 `json:"text,omitempty"`
	Transcript   string                 `json:"transcript,omitempty"`
	Language     string                 `json:"language,omitempty"`
	Confidence   float64                `json:"confidence"`
	Entities     []Entity               `json:"entities,omitempty"`
	Sentiment    string                 `json:"sentiment,omitempty"`
	Topics       []string               `json:"topics,omitempty"`
	SafetyIssues []SafetyIssue          `json:"safety_issues,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// Entity represents a detected entity in content
type Entity struct {
	Type       string  `json:"type"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	StartPos   int     `json:"start_pos,omitempty"`
	EndPos     int     `json:"end_pos,omitempty"`
}

// SafetyIssue represents a content safety concern
type SafetyIssue struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	Confidence  float64 `json:"confidence"`
}

// MultiModalRequest represents a request for multi-modal processing
type MultiModalRequest struct {
	Content     *MultiModalContent `json:"content"`
	Provider    string             `json:"provider"`
	Model       string             `json:"model"`
	Prompt      string             `json:"prompt,omitempty"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	SafetyCheck bool               `json:"safety_check"`
}

// MultiModalResponse represents a response from multi-modal processing
type MultiModalResponse struct {
	RequestID      string             `json:"request_id"`
	Content        *MultiModalContent `json:"content"`
	Response       string             `json:"response"`
	TokensUsed     int                `json:"tokens_used"`
	ProcessingTime time.Duration      `json:"processing_time"`
	Error          string             `json:"error,omitempty"`
}

// NewMultiModalProcessor creates a new multi-modal processor
func NewMultiModalProcessor() *MultiModalProcessor {
	mmp := &MultiModalProcessor{
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for multimodal processing
		},
		providers: make(map[string]*ProviderConfig),
	}
	mmp.contentSafety = NewContentSafetyChecker(mmp)
	mmp.imageProcessor = NewImageProcessor(mmp)
	mmp.audioProcessor = NewAudioProcessor(mmp)
	return mmp
}

// RegisterProvider registers an LLM provider for multimodal processing
func (mmp *MultiModalProcessor) RegisterProvider(name string, config *ProviderConfig) {
	mmp.mu.Lock()
	defer mmp.mu.Unlock()
	mmp.providers[name] = config
}

// SetDefaultVisionProvider sets the default provider for vision tasks
func (mmp *MultiModalProcessor) SetDefaultVisionProvider(name string) {
	mmp.mu.Lock()
	defer mmp.mu.Unlock()
	mmp.defaultVision = name
}

// SetDefaultAudioProvider sets the default provider for audio tasks
func (mmp *MultiModalProcessor) SetDefaultAudioProvider(name string) {
	mmp.mu.Lock()
	defer mmp.mu.Unlock()
	mmp.defaultAudio = name
}

// SetDefaultSafetyProvider sets the default provider for safety checking
func (mmp *MultiModalProcessor) SetDefaultSafetyProvider(name string) {
	mmp.mu.Lock()
	defer mmp.mu.Unlock()
	mmp.defaultSafety = name
}

// GetProvider returns a provider by name
func (mmp *MultiModalProcessor) GetProvider(name string) (*ProviderConfig, bool) {
	mmp.mu.RLock()
	defer mmp.mu.RUnlock()
	provider, ok := mmp.providers[name]
	return provider, ok
}

// GetHTTPClient returns the HTTP client
func (mmp *MultiModalProcessor) GetHTTPClient() *http.Client {
	return mmp.httpClient
}

// ProcessContent processes multi-modal content
func (mmp *MultiModalProcessor) ProcessContent(ctx context.Context, req *MultiModalRequest) (*MultiModalResponse, error) {
	startTime := time.Now()
	requestID := uuid.New().String()

	response := &MultiModalResponse{
		RequestID: requestID,
		Content:   req.Content,
	}

	// Validate content
	if err := mmp.validateContent(req.Content); err != nil {
		response.Error = fmt.Sprintf("content validation failed: %v", err)
		return response, err
	}

	// Safety check if requested
	if req.SafetyCheck {
		safetyResult, err := mmp.contentSafety.CheckContent(ctx, req.Content)
		if err != nil {
			response.Error = fmt.Sprintf("safety check failed: %v", err)
			return response, err
		}

		req.Content.SafetyScore = safetyResult.Score
		if len(safetyResult.Issues) > 0 {
			req.Content.Analysis.SafetyIssues = safetyResult.Issues
		}
	}

	// Process based on content type
	var analysis *ContentAnalysis
	var err error

	switch req.Content.Type {
	case ContentTypeImage:
		analysis, err = mmp.imageProcessor.ProcessImage(ctx, req.Content, req.Prompt)
	case ContentTypeAudio:
		analysis, err = mmp.audioProcessor.ProcessAudio(ctx, req.Content, req.Prompt)
	case ContentTypeVideo:
		analysis, err = mmp.processVideo(ctx, req.Content, req.Prompt)
	default:
		err = fmt.Errorf("unsupported content type: %s", req.Content.Type)
	}

	if err != nil {
		response.Error = fmt.Sprintf("content processing failed: %v", err)
		return response, err
	}

	req.Content.Analysis = analysis
	req.Content.ProcessedAt = &startTime

	// Generate response using specified provider
	llmResponse, err := mmp.generateLLMResponse(ctx, req, analysis)
	if err != nil {
		response.Error = fmt.Sprintf("LLM response generation failed: %v", err)
		return response, err
	}

	response.Response = llmResponse
	response.ProcessingTime = time.Since(startTime)

	return response, nil
}

// validateContent validates multi-modal content
func (mmp *MultiModalProcessor) validateContent(content *MultiModalContent) error {
	if content == nil {
		return fmt.Errorf("content is required")
	}

	if content.Type == "" {
		return fmt.Errorf("content type is required")
	}

	// Check size limits
	maxSizes := map[ContentType]int64{
		ContentTypeImage: 10 * 1024 * 1024,  // 10MB
		ContentTypeAudio: 50 * 1024 * 1024,  // 50MB
		ContentTypeVideo: 100 * 1024 * 1024, // 100MB
	}

	if maxSize, exists := maxSizes[content.Type]; exists && content.Size > maxSize {
		return fmt.Errorf("content size %d exceeds maximum %d for type %s", content.Size, maxSize, content.Type)
	}

	// Validate MIME type
	if content.MimeType != "" {
		expectedTypes := map[ContentType][]string{
			ContentTypeImage: {"image/jpeg", "image/png", "image/gif", "image/webp"},
			ContentTypeAudio: {"audio/mpeg", "audio/wav", "audio/ogg", "audio/mp4"},
			ContentTypeVideo: {"video/mp4", "video/avi", "video/mov", "video/webm"},
		}

		if expected, exists := expectedTypes[content.Type]; exists {
			valid := false
			for _, mimeType := range expected {
				if content.MimeType == mimeType {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid MIME type %s for content type %s", content.MimeType, content.Type)
			}
		}
	}

	return nil
}

// generateLLMResponse generates a response using the specified LLM provider
func (mmp *MultiModalProcessor) generateLLMResponse(ctx context.Context, req *MultiModalRequest, analysis *ContentAnalysis) (string, error) {
	// Create prompt with analysis
	prompt := req.Prompt
	if prompt == "" {
		prompt = "Based on the content analysis provided, give a comprehensive summary and any relevant insights."
	}

	// Add analysis context
	contextInfo := fmt.Sprintf("Content Analysis:\n- Description: %s\n", analysis.Description)
	if analysis.Text != "" {
		contextInfo += fmt.Sprintf("- Detected Text: %s\n", analysis.Text)
	}
	if analysis.Transcript != "" {
		contextInfo += fmt.Sprintf("- Transcript: %s\n", analysis.Transcript)
	}
	if len(analysis.Objects) > 0 {
		contextInfo += fmt.Sprintf("- Detected Objects: %s\n", strings.Join(analysis.Objects, ", "))
	}
	if len(analysis.Topics) > 0 {
		contextInfo += fmt.Sprintf("- Topics: %s\n", strings.Join(analysis.Topics, ", "))
	}
	if analysis.Sentiment != "" {
		contextInfo += fmt.Sprintf("- Sentiment: %s\n", analysis.Sentiment)
	}
	if analysis.Language != "" {
		contextInfo += fmt.Sprintf("- Language: %s\n", analysis.Language)
	}

	fullPrompt := fmt.Sprintf("%s\n\nUser Request: %s", contextInfo, prompt)

	// Get provider for LLM response
	providerName := req.Provider
	if providerName == "" {
		mmp.mu.RLock()
		providerName = mmp.defaultVision // Use vision provider for general LLM
		mmp.mu.RUnlock()
		if providerName == "" {
			providerName = "openai"
		}
	}

	provider, ok := mmp.GetProvider(providerName)
	if !ok {
		// Return formatted analysis if no provider configured
		return fmt.Sprintf("Analysis Summary:\n%s", contextInfo), nil
	}

	// Build chat completion request
	model := req.Model
	if model == "" {
		model = provider.Model
	}
	if model == "" {
		model = "gpt-4o"
	}

	request := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are an expert content analyst. Provide clear, actionable insights based on the content analysis provided."},
			{"role": "user", "content": fullPrompt},
		},
		"max_tokens": req.MaxTokens,
	}

	if request["max_tokens"] == 0 {
		request["max_tokens"] = 1024
	}

	if req.Temperature > 0 {
		request["temperature"] = req.Temperature
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(endpoint, "/"))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.APIKey))

	resp, err := mmp.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return fmt.Sprintf("Analysis Summary:\n%s", contextInfo), nil
	}

	return chatResp.Choices[0].Message.Content, nil
}

// processVideo processes video content by extracting and analyzing frames and audio
func (mmp *MultiModalProcessor) processVideo(ctx context.Context, content *MultiModalContent, prompt string) (*ContentAnalysis, error) {
	// Video processing involves:
	// 1. Extract keyframes for image analysis
	// 2. Extract audio track for transcription
	// 3. Combine results

	analysis := &ContentAnalysis{
		Topics:     []string{},
		Entities:   []Entity{},
		CustomFields: map[string]interface{}{
			"content_type": "video",
			"processing":   "frame_and_audio_analysis",
		},
	}

	// For video, we need actual frame extraction which requires ffmpeg
	// Check if we have video data
	if len(content.Data) == 0 && content.Base64Data == "" {
		return nil, fmt.Errorf("no video data provided")
	}

	// Get vision provider for frame analysis
	mmp.mu.RLock()
	visionProviderName := mmp.defaultVision
	audioProviderName := mmp.defaultAudio
	mmp.mu.RUnlock()

	if visionProviderName == "" {
		visionProviderName = "openai"
	}

	_, hasVision := mmp.GetProvider(visionProviderName)

	// For video analysis, we use the vision model to describe the video
	// based on a sample frame (first frame analysis)
	if hasVision {
		// Create a video analysis prompt
		videoPrompt := prompt
		if videoPrompt == "" {
			videoPrompt = `Analyze this video content. This is a keyframe from the video.
Describe:
1. The main subjects and action visible
2. The setting and context
3. Any text or graphics visible
4. The overall mood and style
5. Estimated content category (educational, entertainment, news, etc.)`
		}

		// Note: For production, you would extract actual keyframes using ffmpeg
		// For now, if we have base64 data that could be a preview frame, use it
		if content.Base64Data != "" {
			// Attempt to analyze as an image (video thumbnail/keyframe)
			frameContent := &MultiModalContent{
				Type:       ContentTypeImage,
				Base64Data: content.Base64Data,
				MimeType:   "image/jpeg", // Assume JPEG for keyframe
			}

			frameAnalysis, err := mmp.imageProcessor.ProcessImage(ctx, frameContent, videoPrompt)
			if err == nil {
				analysis.Description = "Video Analysis (from keyframe): " + frameAnalysis.Description
				analysis.Objects = frameAnalysis.Objects
				analysis.Topics = append(analysis.Topics, frameAnalysis.Topics...)
				analysis.Sentiment = frameAnalysis.Sentiment
				analysis.Confidence = frameAnalysis.Confidence * 0.9 // Slightly lower for video
			}
		}
	}

	// Process audio track if audio provider is configured
	if audioProviderName != "" {
		if _, hasAudio := mmp.GetProvider(audioProviderName); hasAudio {
			// In production, you would extract the audio track using ffmpeg
			// For now, note that audio processing would be done here
			analysis.CustomFields["audio_extraction"] = "requires_ffmpeg"
		}
	}

	// If no analysis was performed, provide metadata-based analysis
	if analysis.Description == "" {
		analysis.Description = fmt.Sprintf("Video file (%s, %d bytes)", content.MimeType, content.Size)
		analysis.Confidence = 0.5

		// Basic topic inference from MIME type
		switch content.MimeType {
		case "video/mp4":
			analysis.Topics = append(analysis.Topics, "video", "multimedia")
		case "video/webm":
			analysis.Topics = append(analysis.Topics, "web video", "multimedia")
		case "video/avi", "video/x-msvideo":
			analysis.Topics = append(analysis.Topics, "video", "legacy format")
		case "video/quicktime", "video/mov":
			analysis.Topics = append(analysis.Topics, "video", "quicktime")
		}
	}

	// Add video-specific metadata
	analysis.CustomFields["mime_type"] = content.MimeType
	analysis.CustomFields["file_size"] = content.Size
	if content.Filename != "" {
		analysis.CustomFields["filename"] = content.Filename
	}

	// Note about production implementation
	analysis.CustomFields["note"] = "For full video analysis, integrate with ffmpeg for frame extraction and audio separation"

	return analysis, nil
}

// VideoFrameExtractor handles video frame extraction (requires ffmpeg in production)
type VideoFrameExtractor struct {
	ffmpegPath string
}

// NewVideoFrameExtractor creates a new video frame extractor
func NewVideoFrameExtractor(ffmpegPath string) *VideoFrameExtractor {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // Use system ffmpeg
	}
	return &VideoFrameExtractor{
		ffmpegPath: ffmpegPath,
	}
}

// ExtractKeyframes extracts keyframes from video at specified intervals
// This is a production-ready interface; implementation requires ffmpeg
func (vfe *VideoFrameExtractor) ExtractKeyframes(videoPath string, intervalSeconds float64, maxFrames int) ([][]byte, error) {
	// In production, this would:
	// 1. Use ffmpeg to extract frames: ffmpeg -i video.mp4 -vf "fps=1/interval" -frames:v maxFrames frame_%03d.jpg
	// 2. Read the extracted frames into memory
	// 3. Return as slice of byte arrays

	// For now, return an error indicating ffmpeg is required
	return nil, fmt.Errorf("video frame extraction requires ffmpeg integration")
}

// ExtractAudioTrack extracts audio from video file
// This is a production-ready interface; implementation requires ffmpeg
func (vfe *VideoFrameExtractor) ExtractAudioTrack(videoPath, outputFormat string) ([]byte, error) {
	// In production, this would:
	// 1. Use ffmpeg to extract audio: ffmpeg -i video.mp4 -vn -acodec copy audio.mp3
	// 2. Read the extracted audio into memory
	// 3. Return as byte array

	return nil, fmt.Errorf("audio extraction requires ffmpeg integration")
}

// ImageProcessor handles image processing and analysis
type ImageProcessor struct {
	processor *MultiModalProcessor
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(processor *MultiModalProcessor) *ImageProcessor {
	return &ImageProcessor{
		processor: processor,
	}
}

// OpenAIVisionRequest represents a vision request for OpenAI GPT-4V
type OpenAIVisionRequest struct {
	Model       string                   `json:"model"`
	Messages    []OpenAIVisionMessage    `json:"messages"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
}

// OpenAIVisionMessage represents a message with multimodal content
type OpenAIVisionMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

// OpenAIVisionTextContent represents text content in a vision message
type OpenAIVisionTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// OpenAIVisionImageContent represents image content in a vision message
type OpenAIVisionImageContent struct {
	Type     string                 `json:"type"`
	ImageURL OpenAIVisionImageURL   `json:"image_url"`
}

// OpenAIVisionImageURL represents an image URL in vision content
type OpenAIVisionImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// OpenAIVisionResponse represents a response from OpenAI vision API
type OpenAIVisionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// ProcessImage processes an image using vision-capable models
func (ip *ImageProcessor) ProcessImage(ctx context.Context, content *MultiModalContent, prompt string) (*ContentAnalysis, error) {
	// Convert image to base64 if needed
	if content.Base64Data == "" && len(content.Data) > 0 {
		content.Base64Data = base64.StdEncoding.EncodeToString(content.Data)
	} else if content.Base64Data == "" {
		return nil, fmt.Errorf("no image data provided")
	}

	// Create vision request
	visionPrompt := prompt
	if visionPrompt == "" {
		visionPrompt = "Analyze this image comprehensively. Describe: 1) Main subjects and objects visible, 2) Any text or writing, 3) Colors and composition, 4) Context and setting, 5) Any notable details. Format your response as structured analysis."
	}

	// Get the default vision provider
	ip.processor.mu.RLock()
	providerName := ip.processor.defaultVision
	ip.processor.mu.RUnlock()

	if providerName == "" {
		providerName = "openai"
	}

	provider, ok := ip.processor.GetProvider(providerName)
	if !ok {
		return nil, fmt.Errorf("vision provider '%s' not configured", providerName)
	}

	// Build the vision request based on provider type
	var analysis *ContentAnalysis
	var err error

	switch {
	case strings.Contains(strings.ToLower(providerName), "openai"):
		analysis, err = ip.processWithOpenAI(ctx, provider, content, visionPrompt)
	case strings.Contains(strings.ToLower(providerName), "anthropic"):
		analysis, err = ip.processWithAnthropic(ctx, provider, content, visionPrompt)
	case strings.Contains(strings.ToLower(providerName), "google"), strings.Contains(strings.ToLower(providerName), "gemini"):
		analysis, err = ip.processWithGemini(ctx, provider, content, visionPrompt)
	default:
		// Default to OpenAI-compatible format
		analysis, err = ip.processWithOpenAI(ctx, provider, content, visionPrompt)
	}

	if err != nil {
		return nil, fmt.Errorf("vision processing failed: %w", err)
	}

	return analysis, nil
}

// processWithOpenAI processes image using OpenAI GPT-4V API
func (ip *ImageProcessor) processWithOpenAI(ctx context.Context, provider *ProviderConfig, content *MultiModalContent, prompt string) (*ContentAnalysis, error) {
	model := provider.Model
	if model == "" {
		model = "gpt-4o" // Default to GPT-4o for vision
	}

	// Build data URL for image
	dataURL := fmt.Sprintf("data:%s;base64,%s", content.MimeType, content.Base64Data)

	// Create the vision request
	request := OpenAIVisionRequest{
		Model:     model,
		MaxTokens: 4096,
		Messages: []OpenAIVisionMessage{
			{
				Role: "user",
				Content: []interface{}{
					OpenAIVisionTextContent{
						Type: "text",
						Text: prompt,
					},
					OpenAIVisionImageContent{
						Type: "image_url",
						ImageURL: OpenAIVisionImageURL{
							URL:    dataURL,
							Detail: "high",
						},
					},
				},
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(endpoint, "/"))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.APIKey))

	resp, err := ip.processor.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var visionResp OpenAIVisionResponse
	if err := json.Unmarshal(body, &visionResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if visionResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", visionResp.Error.Message)
	}

	if len(visionResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from vision model")
	}

	// Parse the response into structured analysis
	responseText := visionResp.Choices[0].Message.Content
	analysis := ip.parseVisionResponse(responseText)

	return analysis, nil
}

// processWithAnthropic processes image using Anthropic Claude API
func (ip *ImageProcessor) processWithAnthropic(ctx context.Context, provider *ProviderConfig, content *MultiModalContent, prompt string) (*ContentAnalysis, error) {
	model := provider.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	// Build Anthropic vision request
	request := map[string]interface{}{
		"model":      model,
		"max_tokens": 4096,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]string{
							"type":       "base64",
							"media_type": content.MimeType,
							"data":       content.Base64Data,
						},
					},
					{
						"type": "text",
						"text": prompt,
					},
				},
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1"
	}
	url := fmt.Sprintf("%s/messages", strings.TrimSuffix(endpoint, "/"))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", provider.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := ip.processor.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if anthropicResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", anthropicResp.Error.Message)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no response from vision model")
	}

	responseText := ""
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			responseText += c.Text
		}
	}

	analysis := ip.parseVisionResponse(responseText)
	return analysis, nil
}

// processWithGemini processes image using Google Gemini API
func (ip *ImageProcessor) processWithGemini(ctx context.Context, provider *ProviderConfig, content *MultiModalContent, prompt string) (*ContentAnalysis, error) {
	model := provider.Model
	if model == "" {
		model = "gemini-1.5-pro"
	}

	// Build Gemini vision request
	request := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"inline_data": map[string]string{
							"mime_type": content.MimeType,
							"data":      content.Base64Data,
						},
					},
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 4096,
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "https://generativelanguage.googleapis.com/v1beta"
	}
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", strings.TrimSuffix(endpoint, "/"), model, provider.APIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ip.processor.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from vision model")
	}

	responseText := geminiResp.Candidates[0].Content.Parts[0].Text
	analysis := ip.parseVisionResponse(responseText)
	return analysis, nil
}

// parseVisionResponse parses raw LLM response into structured analysis
func (ip *ImageProcessor) parseVisionResponse(responseText string) *ContentAnalysis {
	analysis := &ContentAnalysis{
		Description: responseText,
		Confidence:  0.9,
	}

	// Extract objects (simple heuristic - look for common object patterns)
	lowerText := strings.ToLower(responseText)
	commonObjects := []string{"person", "people", "car", "building", "tree", "animal", "dog", "cat", "table", "chair", "computer", "phone", "book", "food", "water", "sky", "road", "mountain", "flower"}
	for _, obj := range commonObjects {
		if strings.Contains(lowerText, obj) {
			analysis.Objects = append(analysis.Objects, obj)
		}
	}

	// Extract sentiment
	positiveWords := []string{"beautiful", "happy", "bright", "colorful", "pleasant", "peaceful", "cheerful"}
	negativeWords := []string{"dark", "gloomy", "sad", "damaged", "broken", "destroyed"}

	positiveCount := 0
	negativeCount := 0
	for _, word := range positiveWords {
		if strings.Contains(lowerText, word) {
			positiveCount++
		}
	}
	for _, word := range negativeWords {
		if strings.Contains(lowerText, word) {
			negativeCount++
		}
	}

	if positiveCount > negativeCount {
		analysis.Sentiment = "positive"
	} else if negativeCount > positiveCount {
		analysis.Sentiment = "negative"
	} else {
		analysis.Sentiment = "neutral"
	}

	// Extract topics
	topicPatterns := map[string][]string{
		"nature":     {"tree", "forest", "mountain", "river", "ocean", "flower", "animal"},
		"urban":      {"building", "city", "street", "car", "road", "traffic"},
		"people":     {"person", "people", "face", "group", "crowd"},
		"technology": {"computer", "phone", "screen", "device", "electronic"},
		"food":       {"food", "restaurant", "meal", "dish", "cooking"},
	}

	for topic, keywords := range topicPatterns {
		for _, keyword := range keywords {
			if strings.Contains(lowerText, keyword) {
				analysis.Topics = append(analysis.Topics, topic)
				break
			}
		}
	}

	return analysis
}

// AudioProcessor handles audio processing and transcription
type AudioProcessor struct {
	processor *MultiModalProcessor
}

// NewAudioProcessor creates a new audio processor
func NewAudioProcessor(processor *MultiModalProcessor) *AudioProcessor {
	return &AudioProcessor{
		processor: processor,
	}
}

// OpenAIWhisperResponse represents a response from OpenAI Whisper API
type OpenAIWhisperResponse struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Segments []struct {
		ID               int     `json:"id"`
		Start            float64 `json:"start"`
		End              float64 `json:"end"`
		Text             string  `json:"text"`
		AvgLogprob       float64 `json:"avg_logprob"`
		CompressionRatio float64 `json:"compression_ratio"`
		NoSpeechProb     float64 `json:"no_speech_prob"`
	} `json:"segments,omitempty"`
}

// ProcessAudio processes audio content for transcription and analysis
func (ap *AudioProcessor) ProcessAudio(ctx context.Context, content *MultiModalContent, prompt string) (*ContentAnalysis, error) {
	// Convert audio to base64 if needed
	if content.Base64Data == "" && len(content.Data) > 0 {
		content.Base64Data = base64.StdEncoding.EncodeToString(content.Data)
	} else if content.Base64Data == "" && len(content.Data) == 0 {
		return nil, fmt.Errorf("no audio data provided")
	}

	// Get the default audio provider
	ap.processor.mu.RLock()
	providerName := ap.processor.defaultAudio
	ap.processor.mu.RUnlock()

	if providerName == "" {
		providerName = "openai"
	}

	provider, ok := ap.processor.GetProvider(providerName)
	if !ok {
		return nil, fmt.Errorf("audio provider '%s' not configured", providerName)
	}

	// Process based on provider type
	var analysis *ContentAnalysis
	var err error

	switch {
	case strings.Contains(strings.ToLower(providerName), "openai"), strings.Contains(strings.ToLower(providerName), "whisper"):
		analysis, err = ap.processWithOpenAIWhisper(ctx, provider, content, prompt)
	case strings.Contains(strings.ToLower(providerName), "google"):
		analysis, err = ap.processWithGoogleSpeech(ctx, provider, content, prompt)
	default:
		// Default to OpenAI Whisper
		analysis, err = ap.processWithOpenAIWhisper(ctx, provider, content, prompt)
	}

	if err != nil {
		return nil, fmt.Errorf("audio processing failed: %w", err)
	}

	return analysis, nil
}

// processWithOpenAIWhisper transcribes audio using OpenAI Whisper API
func (ap *AudioProcessor) processWithOpenAIWhisper(ctx context.Context, provider *ProviderConfig, content *MultiModalContent, prompt string) (*ContentAnalysis, error) {
	// Decode base64 audio data
	var audioData []byte
	var err error
	if len(content.Data) > 0 {
		audioData = content.Data
	} else {
		audioData, err = base64.StdEncoding.DecodeString(content.Base64Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode audio data: %w", err)
		}
	}

	// Create multipart form data for Whisper API
	var requestBody bytes.Buffer
	boundary := "----WebKitFormBoundary" + uuid.New().String()[:8]

	// Add file field
	requestBody.WriteString("--" + boundary + "\r\n")
	filename := content.Filename
	if filename == "" {
		// Determine extension from mime type
		switch content.MimeType {
		case "audio/mpeg":
			filename = "audio.mp3"
		case "audio/wav":
			filename = "audio.wav"
		case "audio/ogg":
			filename = "audio.ogg"
		case "audio/mp4":
			filename = "audio.m4a"
		default:
			filename = "audio.mp3"
		}
	}
	requestBody.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\n", filename))
	requestBody.WriteString(fmt.Sprintf("Content-Type: %s\r\n\r\n", content.MimeType))
	requestBody.Write(audioData)
	requestBody.WriteString("\r\n")

	// Add model field
	model := provider.Model
	if model == "" {
		model = "whisper-1"
	}
	requestBody.WriteString("--" + boundary + "\r\n")
	requestBody.WriteString("Content-Disposition: form-data; name=\"model\"\r\n\r\n")
	requestBody.WriteString(model + "\r\n")

	// Add response_format field for detailed response
	requestBody.WriteString("--" + boundary + "\r\n")
	requestBody.WriteString("Content-Disposition: form-data; name=\"response_format\"\r\n\r\n")
	requestBody.WriteString("verbose_json\r\n")

	// Add prompt if provided
	if prompt != "" {
		requestBody.WriteString("--" + boundary + "\r\n")
		requestBody.WriteString("Content-Disposition: form-data; name=\"prompt\"\r\n\r\n")
		requestBody.WriteString(prompt + "\r\n")
	}

	requestBody.WriteString("--" + boundary + "--\r\n")

	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	url := fmt.Sprintf("%s/audio/transcriptions", strings.TrimSuffix(endpoint, "/"))

	req, err := http.NewRequestWithContext(ctx, "POST", url, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.APIKey))

	resp, err := ap.processor.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var whisperResp OpenAIWhisperResponse
	if err := json.Unmarshal(body, &whisperResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Calculate confidence from segment probabilities
	confidence := 0.9
	if len(whisperResp.Segments) > 0 {
		avgLogprob := 0.0
		for _, seg := range whisperResp.Segments {
			avgLogprob += seg.AvgLogprob
		}
		avgLogprob /= float64(len(whisperResp.Segments))
		// Convert log probability to confidence (rough approximation)
		confidence = 1.0 / (1.0 + (-avgLogprob / 10.0))
		if confidence > 1.0 {
			confidence = 0.95
		}
	}

	// Parse transcript for entities and topics
	analysis := ap.parseTranscript(whisperResp.Text)
	analysis.Transcript = whisperResp.Text
	analysis.Language = whisperResp.Language
	analysis.Confidence = confidence
	analysis.Description = fmt.Sprintf("Audio transcription (%s, %.1f seconds)", whisperResp.Language, whisperResp.Duration)

	return analysis, nil
}

// processWithGoogleSpeech transcribes audio using Google Cloud Speech-to-Text API
func (ap *AudioProcessor) processWithGoogleSpeech(ctx context.Context, provider *ProviderConfig, content *MultiModalContent, prompt string) (*ContentAnalysis, error) {
	// Decode base64 audio data if needed
	audioBase64 := content.Base64Data
	if audioBase64 == "" && len(content.Data) > 0 {
		audioBase64 = base64.StdEncoding.EncodeToString(content.Data)
	}

	// Determine encoding from mime type
	encoding := "LINEAR16"
	switch content.MimeType {
	case "audio/mpeg":
		encoding = "MP3"
	case "audio/ogg":
		encoding = "OGG_OPUS"
	case "audio/wav":
		encoding = "LINEAR16"
	}

	// Build Google Speech API request
	request := map[string]interface{}{
		"config": map[string]interface{}{
			"encoding":                   encoding,
			"languageCode":               "en-US",
			"enableAutomaticPunctuation": true,
			"enableWordTimeOffsets":      true,
			"model":                      "latest_long",
		},
		"audio": map[string]string{
			"content": audioBase64,
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "https://speech.googleapis.com/v1"
	}
	url := fmt.Sprintf("%s/speech:recognize?key=%s", strings.TrimSuffix(endpoint, "/"), provider.APIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ap.processor.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var googleResp struct {
		Results []struct {
			Alternatives []struct {
				Transcript string  `json:"transcript"`
				Confidence float64 `json:"confidence"`
			} `json:"alternatives"`
		} `json:"results"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if googleResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", googleResp.Error.Message)
	}

	// Combine all transcripts
	var fullTranscript strings.Builder
	var totalConfidence float64
	count := 0
	for _, result := range googleResp.Results {
		if len(result.Alternatives) > 0 {
			fullTranscript.WriteString(result.Alternatives[0].Transcript)
			fullTranscript.WriteString(" ")
			totalConfidence += result.Alternatives[0].Confidence
			count++
		}
	}

	avgConfidence := 0.9
	if count > 0 {
		avgConfidence = totalConfidence / float64(count)
	}

	transcript := strings.TrimSpace(fullTranscript.String())
	analysis := ap.parseTranscript(transcript)
	analysis.Transcript = transcript
	analysis.Language = "en"
	analysis.Confidence = avgConfidence
	analysis.Description = "Audio transcription via Google Speech-to-Text"

	return analysis, nil
}

// parseTranscript parses transcript for entities and topics
func (ap *AudioProcessor) parseTranscript(transcript string) *ContentAnalysis {
	analysis := &ContentAnalysis{}
	lowerTranscript := strings.ToLower(transcript)

	// Extract entities (simple NER heuristics)
	// Look for capitalized words that might be names/places
	words := strings.Fields(transcript)
	for i, word := range words {
		if len(word) > 1 && word[0] >= 'A' && word[0] <= 'Z' {
			// Skip if it's the first word in a sentence
			if i > 0 && !strings.HasSuffix(words[i-1], ".") {
				cleanWord := strings.Trim(word, ".,!?;:")
				if len(cleanWord) > 2 {
					analysis.Entities = append(analysis.Entities, Entity{
						Type:       "entity",
						Text:       cleanWord,
						Confidence: 0.7,
					})
				}
			}
		}
	}

	// Extract topics
	topicKeywords := map[string][]string{
		"technology":  {"computer", "software", "technology", "ai", "artificial intelligence", "machine learning", "algorithm", "data", "digital"},
		"business":    {"company", "business", "market", "investment", "revenue", "profit", "customer", "sales"},
		"science":     {"research", "study", "experiment", "theory", "hypothesis", "discovery", "scientist"},
		"health":      {"health", "medical", "doctor", "patient", "treatment", "disease", "hospital"},
		"education":   {"school", "university", "student", "teacher", "learning", "education", "course"},
		"environment": {"climate", "environment", "nature", "pollution", "sustainable", "energy"},
	}

	for topic, keywords := range topicKeywords {
		for _, keyword := range keywords {
			if strings.Contains(lowerTranscript, keyword) {
				analysis.Topics = append(analysis.Topics, topic)
				break
			}
		}
	}

	// Determine sentiment
	positiveWords := []string{"great", "excellent", "wonderful", "amazing", "fantastic", "happy", "good", "success", "love", "best"}
	negativeWords := []string{"bad", "terrible", "awful", "horrible", "sad", "fail", "worst", "hate", "problem", "issue"}

	positiveCount := 0
	negativeCount := 0
	for _, word := range positiveWords {
		if strings.Contains(lowerTranscript, word) {
			positiveCount++
		}
	}
	for _, word := range negativeWords {
		if strings.Contains(lowerTranscript, word) {
			negativeCount++
		}
	}

	if positiveCount > negativeCount+1 {
		analysis.Sentiment = "positive"
	} else if negativeCount > positiveCount+1 {
		analysis.Sentiment = "negative"
	} else {
		analysis.Sentiment = "neutral"
	}

	return analysis
}

// ContentSafetyChecker handles content safety and moderation
type ContentSafetyChecker struct {
	processor *MultiModalProcessor
}

// NewContentSafetyChecker creates a new content safety checker
func NewContentSafetyChecker(processor *MultiModalProcessor) *ContentSafetyChecker {
	return &ContentSafetyChecker{
		processor: processor,
	}
}

// SafetyResult represents the result of a safety check
type SafetyResult struct {
	Score      float64       `json:"score"`
	Safe       bool          `json:"safe"`
	Issues     []SafetyIssue `json:"issues"`
	Categories []string      `json:"categories"`
}

// OpenAIModerationResponse represents a response from OpenAI Moderation API
type OpenAIModerationResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Results []struct {
		Flagged        bool               `json:"flagged"`
		Categories     map[string]bool    `json:"categories"`
		CategoryScores map[string]float64 `json:"category_scores"`
	} `json:"results"`
}

// CheckContent checks content for safety issues
func (csc *ContentSafetyChecker) CheckContent(ctx context.Context, content *MultiModalContent) (*SafetyResult, error) {
	// Get the default safety provider
	csc.processor.mu.RLock()
	providerName := csc.processor.defaultSafety
	csc.processor.mu.RUnlock()

	if providerName == "" {
		providerName = "openai"
	}

	provider, ok := csc.processor.GetProvider(providerName)
	if !ok {
		// If no provider configured, perform basic local check
		return csc.performBasicSafetyCheck(content)
	}

	// For text content in images/audio, use moderation API
	var textToCheck string
	if content.TextContent != "" {
		textToCheck = content.TextContent
	} else if content.Analysis != nil && content.Analysis.Transcript != "" {
		textToCheck = content.Analysis.Transcript
	} else if content.Analysis != nil && content.Analysis.Text != "" {
		textToCheck = content.Analysis.Text
	}

	if textToCheck == "" {
		// No text to check, perform image-based safety check if applicable
		if content.Type == ContentTypeImage {
			return csc.checkImageSafety(ctx, provider, content)
		}
		return &SafetyResult{
			Score:  0.95,
			Safe:   true,
			Issues: []SafetyIssue{},
		}, nil
	}

	// Use OpenAI Moderation API for text
	return csc.checkWithOpenAIModeration(ctx, provider, textToCheck)
}

// checkWithOpenAIModeration checks text using OpenAI Moderation API
func (csc *ContentSafetyChecker) checkWithOpenAIModeration(ctx context.Context, provider *ProviderConfig, text string) (*SafetyResult, error) {
	request := map[string]string{
		"input": text,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	url := fmt.Sprintf("%s/moderations", strings.TrimSuffix(endpoint, "/"))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.APIKey))

	resp, err := csc.processor.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var moderationResp OpenAIModerationResponse
	if err := json.Unmarshal(body, &moderationResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := &SafetyResult{
		Score:  1.0,
		Safe:   true,
		Issues: []SafetyIssue{},
	}

	if len(moderationResp.Results) > 0 {
		modResult := moderationResp.Results[0]
		result.Safe = !modResult.Flagged

		// Calculate overall safety score
		maxScore := 0.0
		for category, score := range modResult.CategoryScores {
			if score > maxScore {
				maxScore = score
			}
			if modResult.Categories[category] {
				result.Categories = append(result.Categories, category)
				severity := "low"
				if score > 0.8 {
					severity = "high"
				} else if score > 0.5 {
					severity = "medium"
				}
				result.Issues = append(result.Issues, SafetyIssue{
					Type:        category,
					Description: fmt.Sprintf("Content flagged for %s", category),
					Severity:    severity,
					Confidence:  score,
				})
			}
		}
		result.Score = 1.0 - maxScore
	}

	return result, nil
}

// checkImageSafety checks image content for safety using vision models
func (csc *ContentSafetyChecker) checkImageSafety(ctx context.Context, provider *ProviderConfig, content *MultiModalContent) (*SafetyResult, error) {
	// Use vision model to analyze image for safety
	safetyPrompt := `Analyze this image for content safety. Check for:
1. Explicit or adult content
2. Violence or gore
3. Hate symbols or imagery
4. Dangerous or illegal activities
5. Personal information exposure

Respond with a JSON object in this exact format:
{"safe": true/false, "issues": [{"type": "category", "description": "details", "severity": "low/medium/high"}]}`

	// Build vision request
	dataURL := fmt.Sprintf("data:%s;base64,%s", content.MimeType, content.Base64Data)

	request := OpenAIVisionRequest{
		Model:     "gpt-4o",
		MaxTokens: 500,
		Messages: []OpenAIVisionMessage{
			{
				Role: "user",
				Content: []interface{}{
					OpenAIVisionTextContent{Type: "text", Text: safetyPrompt},
					OpenAIVisionImageContent{
						Type:     "image_url",
						ImageURL: OpenAIVisionImageURL{URL: dataURL, Detail: "low"},
					},
				},
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(endpoint, "/"))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.APIKey))

	resp, err := csc.processor.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var visionResp OpenAIVisionResponse
	if err := json.Unmarshal(body, &visionResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(visionResp.Choices) == 0 {
		return &SafetyResult{Score: 0.95, Safe: true}, nil
	}

	// Parse the JSON response
	responseText := visionResp.Choices[0].Message.Content

	// Try to extract JSON from the response
	result := &SafetyResult{
		Score:  0.95,
		Safe:   true,
		Issues: []SafetyIssue{},
	}

	// Look for JSON in the response
	startIdx := strings.Index(responseText, "{")
	endIdx := strings.LastIndex(responseText, "}")
	if startIdx >= 0 && endIdx > startIdx {
		jsonStr := responseText[startIdx : endIdx+1]
		var safetyJSON struct {
			Safe   bool `json:"safe"`
			Issues []struct {
				Type        string `json:"type"`
				Description string `json:"description"`
				Severity    string `json:"severity"`
			} `json:"issues"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &safetyJSON); err == nil {
			result.Safe = safetyJSON.Safe
			for _, issue := range safetyJSON.Issues {
				confidence := 0.7
				if issue.Severity == "high" {
					confidence = 0.9
				} else if issue.Severity == "medium" {
					confidence = 0.8
				}
				result.Issues = append(result.Issues, SafetyIssue{
					Type:        issue.Type,
					Description: issue.Description,
					Severity:    issue.Severity,
					Confidence:  confidence,
				})
			}
			if !result.Safe {
				result.Score = 0.3
			}
		}
	}

	return result, nil
}

// performBasicSafetyCheck performs basic safety checks without external API
func (csc *ContentSafetyChecker) performBasicSafetyCheck(content *MultiModalContent) (*SafetyResult, error) {
	result := &SafetyResult{
		Score:  0.95,
		Safe:   true,
		Issues: []SafetyIssue{},
	}

	// Check file size limits
	maxSizes := map[ContentType]int64{
		ContentTypeImage: 20 * 1024 * 1024,  // 20MB
		ContentTypeAudio: 100 * 1024 * 1024, // 100MB
		ContentTypeVideo: 500 * 1024 * 1024, // 500MB
	}

	if maxSize, ok := maxSizes[content.Type]; ok && content.Size > maxSize {
		result.Issues = append(result.Issues, SafetyIssue{
			Type:        "file_size",
			Description: fmt.Sprintf("File size exceeds maximum allowed (%d bytes)", maxSize),
			Severity:    "medium",
			Confidence:  1.0,
		})
		result.Score = 0.7
	}

	// Check for suspicious MIME types
	suspiciousMimeTypes := []string{"application/x-executable", "application/x-msdownload", "application/x-dosexec"}
	for _, suspicious := range suspiciousMimeTypes {
		if content.MimeType == suspicious {
			result.Safe = false
			result.Score = 0.0
			result.Issues = append(result.Issues, SafetyIssue{
				Type:        "suspicious_file",
				Description: "File type is potentially malicious",
				Severity:    "high",
				Confidence:  1.0,
			})
			break
		}
	}

	return result, nil
}

// ContentValidator provides content validation utilities
type ContentValidator struct{}

// NewContentValidator creates a new content validator
func NewContentValidator() *ContentValidator {
	return &ContentValidator{}
}

// ValidateImage validates image content
func (cv *ContentValidator) ValidateImage(data []byte) error {
	// Check file signature
	if len(data) < 4 {
		return fmt.Errorf("image data too small")
	}

	// Check common image signatures
	signatures := map[string][]byte{
		"jpeg": {0xFF, 0xD8, 0xFF},
		"png":  {0x89, 0x50, 0x4E, 0x47},
		"gif":  {0x47, 0x49, 0x46},
		"webp": {0x52, 0x49, 0x46, 0x46},
	}

	for _, sig := range signatures {
		if len(data) >= len(sig) && bytes.Equal(data[:len(sig)], sig) {
			return nil // Valid format
		}
	}

	return fmt.Errorf("unsupported or invalid image format")
}

// ValidateAudio validates audio content
func (cv *ContentValidator) ValidateAudio(data []byte) error {
	// Check file signatures for common audio formats
	if len(data) < 4 {
		return fmt.Errorf("audio data too small")
	}

	signatures := map[string][]byte{
		"mp3": {0xFF, 0xFB},                                     // MP3 frame sync
		"wav": {0x52, 0x49, 0x46, 0x46},                         // RIFF
		"ogg": {0x4F, 0x67, 0x67, 0x53},                         // OggS
		"m4a": {0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70}, // M4A
	}

	for _, sig := range signatures {
		if len(data) >= len(sig) && bytes.Equal(data[:len(sig)], sig) {
			return nil // Valid format
		}
	}

	return fmt.Errorf("unsupported or invalid audio format")
}

// DetectContentType detects content type from data
func (cv *ContentValidator) DetectContentType(data []byte, filename string) (ContentType, string, error) {
	// Try MIME type detection first
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	// Map MIME type to content type
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return ContentTypeImage, mimeType, nil
	case strings.HasPrefix(mimeType, "audio/"):
		return ContentTypeAudio, mimeType, nil
	case strings.HasPrefix(mimeType, "video/"):
		return ContentTypeVideo, mimeType, nil
	case strings.HasPrefix(mimeType, "text/"):
		return ContentTypeText, mimeType, nil
	default:
		return "", mimeType, fmt.Errorf("unable to detect content type for MIME type: %s", mimeType)
	}
}

// CreateMultiModalContent creates a MultiModalContent from various inputs
func CreateMultiModalContent(data []byte, url, filename string) (*MultiModalContent, error) {
	content := &MultiModalContent{
		ID:       uuid.New().String(),
		Data:     data,
		URL:      url,
		Filename: filename,
		Size:     int64(len(data)),
		Metadata: make(map[string]interface{}),
	}

	// Detect content type
	validator := NewContentValidator()
	contentType, mimeType, err := validator.DetectContentType(data, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to detect content type: %w", err)
	}

	content.Type = contentType
	content.MimeType = mimeType

	// Validate based on type
	switch contentType {
	case ContentTypeImage:
		if err := validator.ValidateImage(data); err != nil {
			return nil, fmt.Errorf("invalid image: %w", err)
		}
	case ContentTypeAudio:
		if err := validator.ValidateAudio(data); err != nil {
			return nil, fmt.Errorf("invalid audio: %w", err)
		}
	}

	// Create base64 representation for API calls
	content.Base64Data = base64.StdEncoding.EncodeToString(data)

	return content, nil
}
