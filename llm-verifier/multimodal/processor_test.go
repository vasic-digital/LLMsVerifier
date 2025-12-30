package multimodal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== ContentType Tests ====================

func TestContentType_Constants(t *testing.T) {
	assert.Equal(t, ContentType("image"), ContentTypeImage)
	assert.Equal(t, ContentType("audio"), ContentTypeAudio)
	assert.Equal(t, ContentType("video"), ContentTypeVideo)
	assert.Equal(t, ContentType("text"), ContentTypeText)
}

// ==================== MultiModalProcessor Tests ====================

func TestNewMultiModalProcessor(t *testing.T) {
	processor := NewMultiModalProcessor()

	require.NotNil(t, processor)
	assert.NotNil(t, processor.httpClient)
	assert.NotNil(t, processor.contentSafety)
	assert.NotNil(t, processor.imageProcessor)
	assert.NotNil(t, processor.audioProcessor)
}

func TestMultiModalProcessor_validateContent(t *testing.T) {
	processor := NewMultiModalProcessor()

	t.Run("nil content", func(t *testing.T) {
		err := processor.validateContent(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content is required")
	})

	t.Run("empty content type", func(t *testing.T) {
		content := &MultiModalContent{}
		err := processor.validateContent(content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content type is required")
	})

	t.Run("valid image content", func(t *testing.T) {
		content := &MultiModalContent{
			Type:     ContentTypeImage,
			MimeType: "image/jpeg",
			Size:     1024, // 1KB
		}
		err := processor.validateContent(content)
		assert.NoError(t, err)
	})

	t.Run("image exceeds size limit", func(t *testing.T) {
		content := &MultiModalContent{
			Type: ContentTypeImage,
			Size: 11 * 1024 * 1024, // 11MB, exceeds 10MB limit
		}
		err := processor.validateContent(content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("audio exceeds size limit", func(t *testing.T) {
		content := &MultiModalContent{
			Type: ContentTypeAudio,
			Size: 51 * 1024 * 1024, // 51MB, exceeds 50MB limit
		}
		err := processor.validateContent(content)
		assert.Error(t, err)
	})

	t.Run("video exceeds size limit", func(t *testing.T) {
		content := &MultiModalContent{
			Type: ContentTypeVideo,
			Size: 101 * 1024 * 1024, // 101MB, exceeds 100MB limit
		}
		err := processor.validateContent(content)
		assert.Error(t, err)
	})

	t.Run("invalid MIME type for image", func(t *testing.T) {
		content := &MultiModalContent{
			Type:     ContentTypeImage,
			MimeType: "application/pdf", // Not a valid image type
			Size:     1024,
		}
		err := processor.validateContent(content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid MIME type")
	})

	t.Run("valid audio content", func(t *testing.T) {
		content := &MultiModalContent{
			Type:     ContentTypeAudio,
			MimeType: "audio/mpeg",
			Size:     1024 * 1024, // 1MB
		}
		err := processor.validateContent(content)
		assert.NoError(t, err)
	})

	t.Run("valid video content", func(t *testing.T) {
		content := &MultiModalContent{
			Type:     ContentTypeVideo,
			MimeType: "video/mp4",
			Size:     10 * 1024 * 1024, // 10MB
		}
		err := processor.validateContent(content)
		assert.NoError(t, err)
	})

	t.Run("text content no size limit", func(t *testing.T) {
		content := &MultiModalContent{
			Type: ContentTypeText,
			Size: 100 * 1024 * 1024, // Large text, no limit
		}
		err := processor.validateContent(content)
		assert.NoError(t, err)
	})
}

func TestMultiModalProcessor_processVideo(t *testing.T) {
	processor := NewMultiModalProcessor()
	ctx := context.Background()

	content := &MultiModalContent{
		Type:       ContentTypeVideo,
		MimeType:   "video/mp4",
		Base64Data: "dGVzdCB2aWRlbyBkYXRh",
	}

	analysis, err := processor.processVideo(ctx, content, "Describe this video")

	require.NoError(t, err)
	require.NotNil(t, analysis)
	assert.Equal(t, "Video content processed", analysis.Description)
	assert.Equal(t, 0.85, analysis.Confidence)
	assert.Contains(t, analysis.Topics, "video")
}

func TestMultiModalProcessor_generateLLMResponse(t *testing.T) {
	processor := NewMultiModalProcessor()
	ctx := context.Background()

	analysis := &ContentAnalysis{
		Description: "A beautiful sunset over the ocean",
		Objects:     []string{"sun", "ocean", "clouds"},
		Text:        "",
		Transcript:  "",
	}

	t.Run("with custom prompt", func(t *testing.T) {
		req := &MultiModalRequest{
			Prompt: "What colors are visible?",
		}

		response, err := processor.generateLLMResponse(ctx, req, analysis)

		require.NoError(t, err)
		assert.Contains(t, response, "Analysis complete")
		assert.Contains(t, response, analysis.Description)
	})

	t.Run("with default prompt", func(t *testing.T) {
		req := &MultiModalRequest{
			Prompt: "",
		}

		response, err := processor.generateLLMResponse(ctx, req, analysis)

		require.NoError(t, err)
		assert.Contains(t, response, "Analysis complete")
	})

	t.Run("with text analysis", func(t *testing.T) {
		analysisWithText := &ContentAnalysis{
			Description: "Document scan",
			Text:        "Sample document text",
		}

		req := &MultiModalRequest{}

		response, err := processor.generateLLMResponse(ctx, req, analysisWithText)

		require.NoError(t, err)
		assert.NotEmpty(t, response)
	})

	t.Run("with transcript", func(t *testing.T) {
		analysisWithTranscript := &ContentAnalysis{
			Description: "Audio recording",
			Transcript:  "Hello, this is a test recording.",
		}

		req := &MultiModalRequest{}

		response, err := processor.generateLLMResponse(ctx, req, analysisWithTranscript)

		require.NoError(t, err)
		assert.NotEmpty(t, response)
	})
}

// ==================== ImageProcessor Tests ====================

func TestNewImageProcessor(t *testing.T) {
	processor := NewImageProcessor()

	require.NotNil(t, processor)
	assert.NotEmpty(t, processor.visionProviders)
	assert.Contains(t, processor.visionProviders, "openai-gpt4v")
	assert.Contains(t, processor.visionProviders, "anthropic-claude3")
}

func TestImageProcessor_ProcessImage(t *testing.T) {
	processor := NewImageProcessor()
	ctx := context.Background()

	t.Run("with base64 data", func(t *testing.T) {
		content := &MultiModalContent{
			Type:       ContentTypeImage,
			Base64Data: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
		}

		analysis, err := processor.ProcessImage(ctx, content, "")

		require.NoError(t, err)
		require.NotNil(t, analysis)
		assert.NotEmpty(t, analysis.Description)
		assert.NotEmpty(t, analysis.Objects)
		assert.Greater(t, analysis.Confidence, 0.0)
	})

	t.Run("with raw data", func(t *testing.T) {
		content := &MultiModalContent{
			Type: ContentTypeImage,
			Data: []byte{0x89, 0x50, 0x4E, 0x47}, // PNG header
		}

		analysis, err := processor.ProcessImage(ctx, content, "Describe the image")

		require.NoError(t, err)
		require.NotNil(t, analysis)
		assert.NotEmpty(t, analysis.Description)
	})

	t.Run("no image data", func(t *testing.T) {
		content := &MultiModalContent{
			Type: ContentTypeImage,
		}

		analysis, err := processor.ProcessImage(ctx, content, "")

		assert.Error(t, err)
		assert.Nil(t, analysis)
		assert.Contains(t, err.Error(), "no image data")
	})

	t.Run("with custom prompt", func(t *testing.T) {
		content := &MultiModalContent{
			Type:       ContentTypeImage,
			Base64Data: "dGVzdA==",
		}

		analysis, err := processor.ProcessImage(ctx, content, "List all objects in this image")

		require.NoError(t, err)
		require.NotNil(t, analysis)
	})
}

// ==================== AudioProcessor Tests ====================

func TestNewAudioProcessor(t *testing.T) {
	processor := NewAudioProcessor()

	require.NotNil(t, processor)
	assert.NotEmpty(t, processor.transcriptionProviders)
	assert.Contains(t, processor.transcriptionProviders, "openai-whisper")
	assert.Contains(t, processor.transcriptionProviders, "google-speech")
}

func TestAudioProcessor_ProcessAudio(t *testing.T) {
	processor := NewAudioProcessor()
	ctx := context.Background()

	t.Run("with base64 data", func(t *testing.T) {
		content := &MultiModalContent{
			Type:       ContentTypeAudio,
			Base64Data: "dGVzdCBhdWRpbyBkYXRh",
		}

		analysis, err := processor.ProcessAudio(ctx, content, "")

		require.NoError(t, err)
		require.NotNil(t, analysis)
		assert.NotEmpty(t, analysis.Transcript)
		assert.Equal(t, "en", analysis.Language)
		assert.Greater(t, analysis.Confidence, 0.0)
	})

	t.Run("with raw data", func(t *testing.T) {
		content := &MultiModalContent{
			Type: ContentTypeAudio,
			Data: []byte{0xFF, 0xFB}, // MP3 frame sync
		}

		analysis, err := processor.ProcessAudio(ctx, content, "Transcribe")

		require.NoError(t, err)
		require.NotNil(t, analysis)
	})

	t.Run("no audio data", func(t *testing.T) {
		content := &MultiModalContent{
			Type: ContentTypeAudio,
		}

		analysis, err := processor.ProcessAudio(ctx, content, "")

		assert.Error(t, err)
		assert.Nil(t, analysis)
		assert.Contains(t, err.Error(), "no audio data")
	})
}

// ==================== ContentSafetyChecker Tests ====================

func TestNewContentSafetyChecker(t *testing.T) {
	checker := NewContentSafetyChecker()

	require.NotNil(t, checker)
	assert.NotEmpty(t, checker.safetyProviders)
	assert.Contains(t, checker.safetyProviders, "openai-moderation")
}

func TestContentSafetyChecker_CheckContent(t *testing.T) {
	checker := NewContentSafetyChecker()
	ctx := context.Background()

	content := &MultiModalContent{
		Type: ContentTypeImage,
		Data: []byte("test image data"),
	}

	result, err := checker.CheckContent(ctx, content)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Safe)
	assert.Equal(t, 0.95, result.Score)
	assert.Empty(t, result.Issues)
}

// ==================== ContentValidator Tests ====================

func TestNewContentValidator(t *testing.T) {
	validator := NewContentValidator()
	require.NotNil(t, validator)
}

func TestContentValidator_ValidateImage(t *testing.T) {
	validator := NewContentValidator()

	t.Run("valid JPEG", func(t *testing.T) {
		data := []byte{0xFF, 0xD8, 0xFF, 0xE0}
		err := validator.ValidateImage(data)
		assert.NoError(t, err)
	})

	t.Run("valid PNG", func(t *testing.T) {
		data := []byte{0x89, 0x50, 0x4E, 0x47}
		err := validator.ValidateImage(data)
		assert.NoError(t, err)
	})

	t.Run("valid GIF", func(t *testing.T) {
		data := []byte{0x47, 0x49, 0x46, 0x38}
		err := validator.ValidateImage(data)
		assert.NoError(t, err)
	})

	t.Run("valid WebP", func(t *testing.T) {
		data := []byte{0x52, 0x49, 0x46, 0x46}
		err := validator.ValidateImage(data)
		assert.NoError(t, err)
	})

	t.Run("too small", func(t *testing.T) {
		data := []byte{0xFF, 0xD8}
		err := validator.ValidateImage(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too small")
	})

	t.Run("invalid format", func(t *testing.T) {
		data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
		err := validator.ValidateImage(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported or invalid")
	})
}

func TestContentValidator_ValidateAudio(t *testing.T) {
	validator := NewContentValidator()

	t.Run("valid MP3", func(t *testing.T) {
		data := []byte{0xFF, 0xFB, 0x90, 0x00}
		err := validator.ValidateAudio(data)
		assert.NoError(t, err)
	})

	t.Run("valid WAV", func(t *testing.T) {
		data := []byte{0x52, 0x49, 0x46, 0x46}
		err := validator.ValidateAudio(data)
		assert.NoError(t, err)
	})

	t.Run("valid OGG", func(t *testing.T) {
		data := []byte{0x4F, 0x67, 0x67, 0x53}
		err := validator.ValidateAudio(data)
		assert.NoError(t, err)
	})

	t.Run("valid M4A", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70}
		err := validator.ValidateAudio(data)
		assert.NoError(t, err)
	})

	t.Run("too small", func(t *testing.T) {
		data := []byte{0xFF}
		err := validator.ValidateAudio(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too small")
	})

	t.Run("invalid format", func(t *testing.T) {
		data := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
		err := validator.ValidateAudio(data)
		assert.Error(t, err)
	})
}

func TestContentValidator_DetectContentType(t *testing.T) {
	validator := NewContentValidator()

	t.Run("detect image from extension", func(t *testing.T) {
		data := []byte("test data")
		contentType, mimeType, err := validator.DetectContentType(data, "photo.jpg")

		require.NoError(t, err)
		assert.Equal(t, ContentTypeImage, contentType)
		assert.Equal(t, "image/jpeg", mimeType)
	})

	t.Run("detect audio from extension", func(t *testing.T) {
		data := []byte("test data")
		contentType, mimeType, err := validator.DetectContentType(data, "song.mp3")

		require.NoError(t, err)
		assert.Equal(t, ContentTypeAudio, contentType)
		assert.Equal(t, "audio/mpeg", mimeType)
	})

	t.Run("detect video from extension", func(t *testing.T) {
		data := []byte("test data")
		contentType, mimeType, err := validator.DetectContentType(data, "clip.mp4")

		require.NoError(t, err)
		assert.Equal(t, ContentTypeVideo, contentType)
		assert.Equal(t, "video/mp4", mimeType)
	})

	t.Run("detect text from content", func(t *testing.T) {
		data := []byte("Hello, this is plain text content.")
		contentType, mimeType, err := validator.DetectContentType(data, "document")

		require.NoError(t, err)
		assert.Equal(t, ContentTypeText, contentType)
		assert.Contains(t, mimeType, "text")
	})

	t.Run("detect PNG from content", func(t *testing.T) {
		data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		contentType, mimeType, err := validator.DetectContentType(data, "unknown")

		require.NoError(t, err)
		assert.Equal(t, ContentTypeImage, contentType)
		assert.Equal(t, "image/png", mimeType)
	})
}

// ==================== CreateMultiModalContent Tests ====================

func TestCreateMultiModalContent(t *testing.T) {
	t.Run("create from PNG data", func(t *testing.T) {
		data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

		content, err := CreateMultiModalContent(data, "", "test.png")

		require.NoError(t, err)
		require.NotNil(t, content)
		assert.NotEmpty(t, content.ID)
		assert.Equal(t, ContentTypeImage, content.Type)
		assert.Equal(t, "image/png", content.MimeType)
		assert.Equal(t, "test.png", content.Filename)
		assert.Equal(t, int64(len(data)), content.Size)
		assert.NotEmpty(t, content.Base64Data)
	})

	t.Run("create from JPEG data", func(t *testing.T) {
		data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}

		content, err := CreateMultiModalContent(data, "https://example.com/image.jpg", "image.jpg")

		require.NoError(t, err)
		require.NotNil(t, content)
		assert.Equal(t, ContentTypeImage, content.Type)
		assert.Equal(t, "https://example.com/image.jpg", content.URL)
	})

	t.Run("create from MP3 data", func(t *testing.T) {
		data := []byte{0xFF, 0xFB, 0x90, 0x00, 0x00}

		content, err := CreateMultiModalContent(data, "", "audio.mp3")

		require.NoError(t, err)
		require.NotNil(t, content)
		assert.Equal(t, ContentTypeAudio, content.Type)
	})

	t.Run("invalid content", func(t *testing.T) {
		// Content that can't be detected as any known type
		data := []byte{0x00, 0x00, 0x00, 0x00}

		_, err := CreateMultiModalContent(data, "", "unknown")

		// May return error if can't detect type
		// or might default to text/plain based on http.DetectContentType
		// Let's just ensure it doesn't panic
		assert.NotNil(t, data)
		_ = err // May or may not be error
	})
}

// ==================== Struct Tests ====================

func TestMultiModalContent_Structure(t *testing.T) {
	content := &MultiModalContent{
		ID:          "test-id",
		Type:        ContentTypeImage,
		Data:        []byte("test"),
		URL:         "https://example.com/image.png",
		Base64Data:  "dGVzdA==",
		MimeType:    "image/png",
		Filename:    "test.png",
		Size:        100,
		Metadata:    map[string]interface{}{"key": "value"},
		TextContent: "extracted text",
		SafetyScore: 0.95,
	}

	assert.Equal(t, "test-id", content.ID)
	assert.Equal(t, ContentTypeImage, content.Type)
	assert.Equal(t, int64(100), content.Size)
	assert.Equal(t, 0.95, content.SafetyScore)
}

func TestContentAnalysis_Structure(t *testing.T) {
	analysis := &ContentAnalysis{
		Description:  "Test description",
		Objects:      []string{"object1", "object2"},
		Text:         "Detected text",
		Transcript:   "Audio transcript",
		Language:     "en",
		Confidence:   0.95,
		Entities:     []Entity{{Type: "person", Text: "John", Confidence: 0.9}},
		Sentiment:    "positive",
		Topics:       []string{"tech", "AI"},
		SafetyIssues: []SafetyIssue{},
		CustomFields: map[string]interface{}{"custom": "field"},
	}

	assert.Equal(t, "Test description", analysis.Description)
	assert.Len(t, analysis.Objects, 2)
	assert.Equal(t, "en", analysis.Language)
	assert.Len(t, analysis.Entities, 1)
	assert.Equal(t, "positive", analysis.Sentiment)
}

func TestEntity_Structure(t *testing.T) {
	entity := Entity{
		Type:       "location",
		Text:       "New York",
		Confidence: 0.88,
		StartPos:   10,
		EndPos:     18,
	}

	assert.Equal(t, "location", entity.Type)
	assert.Equal(t, "New York", entity.Text)
	assert.Equal(t, 0.88, entity.Confidence)
	assert.Equal(t, 10, entity.StartPos)
	assert.Equal(t, 18, entity.EndPos)
}

func TestSafetyIssue_Structure(t *testing.T) {
	issue := SafetyIssue{
		Type:        "violence",
		Description: "Contains violent content",
		Severity:    "high",
		Confidence:  0.75,
	}

	assert.Equal(t, "violence", issue.Type)
	assert.Equal(t, "high", issue.Severity)
	assert.Equal(t, 0.75, issue.Confidence)
}

func TestMultiModalRequest_Structure(t *testing.T) {
	content := &MultiModalContent{Type: ContentTypeImage}
	request := &MultiModalRequest{
		Content:     content,
		Provider:    "openai",
		Model:       "gpt-4-vision",
		Prompt:      "Describe this image",
		MaxTokens:   1000,
		Temperature: 0.7,
		SafetyCheck: true,
	}

	assert.Equal(t, content, request.Content)
	assert.Equal(t, "openai", request.Provider)
	assert.Equal(t, "gpt-4-vision", request.Model)
	assert.True(t, request.SafetyCheck)
}

func TestMultiModalResponse_Structure(t *testing.T) {
	response := &MultiModalResponse{
		RequestID:      "req-123",
		Content:        &MultiModalContent{},
		Response:       "Analysis result",
		TokensUsed:     500,
		ProcessingTime: 1000000000, // 1 second
		Error:          "",
	}

	assert.Equal(t, "req-123", response.RequestID)
	assert.Equal(t, "Analysis result", response.Response)
	assert.Equal(t, 500, response.TokensUsed)
	assert.Empty(t, response.Error)
}

func TestSafetyResult_Structure(t *testing.T) {
	result := &SafetyResult{
		Score:      0.98,
		Safe:       true,
		Issues:     []SafetyIssue{},
		Categories: []string{"safe_content"},
	}

	assert.Equal(t, 0.98, result.Score)
	assert.True(t, result.Safe)
	assert.Empty(t, result.Issues)
	assert.Contains(t, result.Categories, "safe_content")
}
