package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Full OpenAI API adapter implementation with all endpoints

// CreateEmbedding creates an embedding vector for input text
func (o *OpenAIAdapter) CreateEmbedding(ctx context.Context, model, input string) (*EmbeddingResponse, error) {
	request := EmbeddingRequest{
		Model: model,
		Input: input,
	}

	// Validate request
	if request.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if request.Input == "" {
		return nil, fmt.Errorf("input is required")
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/embeddings", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// CreateCompletion creates a text completion (legacy endpoint)
func (o *OpenAIAdapter) CreateCompletion(ctx context.Context, request CompletionRequest) (*CompletionResponse, error) {
	// Validate request
	if request.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if request.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/completions", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// CreateModeration creates a moderation for input text
func (o *OpenAIAdapter) CreateModeration(ctx context.Context, input string, model string) (*ModerationResponse, error) {
	request := ModerationRequest{
		Input: input,
		Model: model, // Optional, defaults to text-moderation-latest
	}

	// Validate request
	if request.Input == "" {
		return nil, fmt.Errorf("input is required")
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/moderations", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response ModerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// CreateImage generates an image from text prompt
func (o *OpenAIAdapter) CreateImage(ctx context.Context, request ImageRequest) (*ImageResponse, error) {
	// Validate request
	if request.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	// Set defaults
	if request.N <= 0 {
		request.N = 1
	}
	if request.Size == "" {
		request.Size = "1024x1024"
	}
	if request.ResponseFormat == "" {
		request.ResponseFormat = "url"
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/images/generations", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// CreateImageEdit edits an image based on a prompt and mask
func (o *OpenAIAdapter) CreateImageEdit(ctx context.Context, request ImageEditRequest) (*ImageResponse, error) {
	// Validate request
	if request.Image == nil && request.ImagePath == "" {
		return nil, fmt.Errorf("image is required")
	}
	if request.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add image file
	if request.Image != nil {
		// Use provided image data
		part, err := writer.CreateFormFile("image", "image.png")
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}
		_, err = io.Copy(part, request.Image)
		if err != nil {
			return nil, fmt.Errorf("failed to copy image data: %w", err)
		}
	} else if request.ImagePath != "" {
		// Use image file path
		file, err := os.Open(request.ImagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open image file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("image", filepath.Base(request.ImagePath))
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}
		_, err = io.Copy(part, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy image file: %w", err)
		}
	}

	// Add mask file if provided
	if request.Mask != nil {
		part, err := writer.CreateFormFile("mask", "mask.png")
		if err != nil {
			return nil, fmt.Errorf("failed to create mask form file: %w", err)
		}
		_, err = io.Copy(part, request.Mask)
		if err != nil {
			return nil, fmt.Errorf("failed to copy mask data: %w", err)
		}
	} else if request.MaskPath != "" {
		file, err := os.Open(request.MaskPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open mask file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("mask", filepath.Base(request.MaskPath))
		if err != nil {
			return nil, fmt.Errorf("failed to create mask form file: %w", err)
		}
		_, err = io.Copy(part, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy mask file: %w", err)
		}
	}

	// Add form fields
	writer.WriteField("prompt", request.Prompt)
	if request.N > 0 {
		writer.WriteField("n", fmt.Sprintf("%d", request.N))
	}
	if request.Size != "" {
		writer.WriteField("size", request.Size)
	}
	if request.ResponseFormat != "" {
		writer.WriteField("response_format", request.ResponseFormat)
	}

	err := writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close form writer: %w", err)
	}

	url := fmt.Sprintf("%s/images/edits", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// CreateImageVariation creates variations of an image
func (o *OpenAIAdapter) CreateImageVariation(ctx context.Context, request ImageVariationRequest) (*ImageResponse, error) {
	// Validate request
	if request.Image == nil && request.ImagePath == "" {
		return nil, fmt.Errorf("image is required")
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add image file
	if request.Image != nil {
		// Use provided image data
		part, err := writer.CreateFormFile("image", "image.png")
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}
		_, err = io.Copy(part, request.Image)
		if err != nil {
			return nil, fmt.Errorf("failed to copy image data: %w", err)
		}
	} else if request.ImagePath != "" {
		// Use image file path
		file, err := os.Open(request.ImagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open image file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("image", filepath.Base(request.ImagePath))
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}
		_, err = io.Copy(part, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy image file: %w", err)
		}
	}

	// Add form fields
	if request.N > 0 {
		writer.WriteField("n", fmt.Sprintf("%d", request.N))
	}
	if request.Size != "" {
		writer.WriteField("size", request.Size)
	}
	if request.ResponseFormat != "" {
		writer.WriteField("response_format", request.ResponseFormat)
	}

	err := writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close form writer: %w", err)
	}

	url := fmt.Sprintf("%s/images/variations", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// CreateTranscription transcribes audio to text
func (o *OpenAIAdapter) CreateTranscription(ctx context.Context, request TranscriptionRequest) (*TranscriptionResponse, error) {
	// Validate request
	if request.File == nil && request.FilePath == "" {
		return nil, fmt.Errorf("file is required")
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add audio file
	if request.File != nil {
		// Use provided file data
		part, err := writer.CreateFormFile("file", "audio.mp3")
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}
		_, err = io.Copy(part, request.File)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file data: %w", err)
		}
	} else if request.FilePath != "" {
		// Use file path
		file, err := os.Open(request.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open audio file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("file", filepath.Base(request.FilePath))
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}
		_, err = io.Copy(part, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file: %w", err)
		}
	}

	// Add form fields
	if request.Model != "" {
		writer.WriteField("model", request.Model)
	}
	if request.Language != "" {
		writer.WriteField("language", request.Language)
	}
	if request.Prompt != "" {
		writer.WriteField("prompt", request.Prompt)
	}
	if request.ResponseFormat != "" {
		writer.WriteField("response_format", request.ResponseFormat)
	}
	if request.Temperature != 0 {
		writer.WriteField("temperature", fmt.Sprintf("%.1f", request.Temperature))
	}

	err := writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close form writer: %w", err)
	}

	url := fmt.Sprintf("%s/audio/transcriptions", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	// Check response format
	if request.ResponseFormat == "json" {
		var response TranscriptionResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &response, nil
	} else {
		// Plain text response
		text, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return &TranscriptionResponse{Text: string(text)}, nil
	}
}

// CreateSpeech synthesizes text to speech
func (o *OpenAIAdapter) CreateSpeech(ctx context.Context, request SpeechRequest) ([]byte, error) {
	// Validate request
	if request.Model == "" {
		request.Model = "tts-1" // Default model
	}
	if request.Input == "" {
		return nil, fmt.Errorf("input is required")
	}
	if request.Voice == "" {
		request.Voice = "alloy" // Default voice
	}

	// Validate voice
	validVoices := []string{"alloy", "echo", "fable", "onyx", "nova", "shimmer"}
	voiceValid := false
	for _, v := range validVoices {
		if request.Voice == v {
			voiceValid = true
			break
		}
	}
	if !voiceValid {
		return nil, fmt.Errorf("invalid voice: %s, must be one of: %s", request.Voice, strings.Join(validVoices, ", "))
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/audio/speech", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	// Return audio data
	return io.ReadAll(resp.Body)
}

// CreateFineTuningJob creates a fine-tuning job
func (o *OpenAIAdapter) CreateFineTuningJob(ctx context.Context, request FineTuningJobRequest) (*FineTuningJobResponse, error) {
	// Validate request
	if request.TrainingFile == "" {
		return nil, fmt.Errorf("training_file is required")
	}
	if request.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/fine_tuning/jobs", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response FineTuningJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// ListFineTuningJobs lists fine-tuning jobs
func (o *OpenAIAdapter) ListFineTuningJobs(ctx context.Context, limit int, after string) (*FineTuningJobsResponse, error) {
	params := make([]string, 0)
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if after != "" {
		params = append(params, fmt.Sprintf("after=%s", after))
	}

	url := fmt.Sprintf("%s/fine_tuning/jobs", o.endpoint)
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response FineTuningJobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// RetrieveFineTuningJob retrieves a fine-tuning job
func (o *OpenAIAdapter) RetrieveFineTuningJob(ctx context.Context, jobID string) (*FineTuningJobResponse, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job_id is required")
	}

	url := fmt.Sprintf("%s/fine_tuning/jobs/%s", o.endpoint, jobID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response FineTuningJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// ListModels lists all available models
func (o *OpenAIAdapter) ListModels(ctx context.Context) (*ModelsResponse, error) {
	url := fmt.Sprintf("%s/models", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// CreateAssistant creates an assistant
func (o *OpenAIAdapter) CreateAssistant(ctx context.Context, request AssistantRequest) (*AssistantResponse, error) {
	// Set default model if not provided
	if request.Model == "" {
		request.Model = "gpt-3.5-turbo"
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/assistants", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response AssistantResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// ListAssistants lists all assistants
func (o *OpenAIAdapter) ListAssistants(ctx context.Context, limit int, order string) (*AssistantsResponse, error) {
	params := make([]string, 0)
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if order != "" {
		params = append(params, fmt.Sprintf("order=%s", order))
	}

	url := fmt.Sprintf("%s/assistants", o.endpoint)
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response AssistantsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// CreateThread creates a thread
func (o *OpenAIAdapter) CreateThread(ctx context.Context, request ThreadRequest) (*ThreadResponse, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/threads", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response ThreadResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// UploadFile uploads a file
func (o *OpenAIAdapter) UploadFile(ctx context.Context, filePath, purpose string) (*FileResponse, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}
	if purpose == "" {
		purpose = "fine-tune" // Default purpose
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	// Add purpose
	writer.WriteField("purpose", purpose)

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close form writer: %w", err)
	}

	url := fmt.Sprintf("%s/files", o.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response FileResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// ListFiles lists all uploaded files
func (o *OpenAIAdapter) ListFiles(ctx context.Context, purpose string) (*FilesResponse, error) {
	params := ""
	if purpose != "" {
		params = fmt.Sprintf("?purpose=%s", purpose)
	}

	url := fmt.Sprintf("%s/files%s", o.endpoint, params)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response FilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// DeleteFile deletes a file
func (o *OpenAIAdapter) DeleteFile(ctx context.Context, fileID string) (*DeleteFileResponse, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	url := fmt.Sprintf("%s/files/%s", o.endpoint, fileID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response DeleteFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// Response structures for all endpoints

// EmbeddingRequest represents request for embeddings
type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// EmbeddingResponse represents response from embeddings
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// CompletionRequest represents request for legacy completions
type CompletionRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
	Logprobs    *bool    `json:"logprobs,omitempty"`
	Echo        bool     `json:"echo,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// CompletionResponse represents response from legacy completions
type CompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		Logprobs     any    `json:"logprobs"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ModerationRequest represents request for moderations
type ModerationRequest struct {
	Input string `json:"input"`
	Model string `json:"model,omitempty"`
}

// ModerationResponse represents response from moderations
type ModerationResponse struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Model  string `json:"model"`
	Results []struct {
		Flagged bool `json:"flagged"`
		Categories struct {
			Sexual             bool `json:"sexual"`
			Hate              bool `json:"hate"`
			Harassment         bool `json:"harassment"`
			SelfHarm          bool `json:"self-harm"`
			SexualMinors       bool `json:"sexual/minors"`
			HateThreatening   bool `json:"hate/threatening"`
			ViolenceGraphic   bool `json:"violence/graphic"`
			SelfHarmIntent    bool `json:"self-harm/intent"`
			SelfHarmInstructions bool `json:"self-harm/instructions"`
			HarassmentThreatening bool `json:"harassment/threatening"`
			Violence          bool `json:"violence"`
		} `json:"categories"`
		CategoryScores struct {
			Sexual             float64 `json:"sexual"`
			Hate              float64 `json:"hate"`
			Harassment         float64 `json:"harassment"`
			SelfHarm          float64 `json:"self-harm"`
			SexualMinors       float64 `json:"sexual/minors"`
			HateThreatening   float64 `json:"hate/threatening"`
			ViolenceGraphic   float64 `json:"violence/graphic"`
			SelfHarmIntent    float64 `json:"self-harm/intent"`
			SelfHarmInstructions float64 `json:"self-harm/instructions"`
			HarassmentThreatening float64 `json:"harassment/threatening"`
			Violence          float64 `json:"violence"`
		} `json:"category_scores"`
	} `json:"results"`
}

// ImageRequest represents request for image generation
type ImageRequest struct {
	Prompt         string `json:"prompt"`
	Model          string `json:"model,omitempty"`
	N              int    `json:"n,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Size           string `json:"size,omitempty"`
	Style          string `json:"style,omitempty"`
	User           string `json:"user,omitempty"`
}

// ImageResponse represents response from image generation
type ImageResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL          string `json:"url,omitempty"`
		B64JSON      string `json:"b64_json,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
}

// ImageEditRequest represents request for image editing
type ImageEditRequest struct {
	Image          io.Reader `json:"-"`
	ImagePath      string    `json:"-"`
	Mask           io.Reader `json:"-"`
	MaskPath       string    `json:"-"`
	Prompt         string    `json:"prompt"`
	N              int       `json:"n,omitempty"`
	Size           string    `json:"size,omitempty"`
	ResponseFormat string    `json:"response_format,omitempty"`
	User           string    `json:"user,omitempty"`
}

// ImageVariationRequest represents request for image variations
type ImageVariationRequest struct {
	Image          io.Reader `json:"-"`
	ImagePath      string    `json:"-"`
	N              int       `json:"n,omitempty"`
	Size           string    `json:"size,omitempty"`
	ResponseFormat string    `json:"response_format,omitempty"`
	User           string    `json:"user,omitempty"`
}

// TranscriptionRequest represents request for speech-to-text
type TranscriptionRequest struct {
	File           io.Reader `json:"-"`
	FilePath       string    `json:"-"`
	Model          string    `json:"model,omitempty"`
	Language       string    `json:"language,omitempty"`
	Prompt         string    `json:"prompt,omitempty"`
	ResponseFormat string    `json:"response_format,omitempty"`
	Temperature    float64   `json:"temperature,omitempty"`
}

// TranscriptionResponse represents response from speech-to-text
type TranscriptionResponse struct {
	Text string `json:"text"`
}

// SpeechRequest represents request for text-to-speech
type SpeechRequest struct {
	Model       string  `json:"model"`
	Input       string  `json:"input"`
	Voice       string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed       float64 `json:"speed,omitempty"`
}

// FineTuningJobRequest represents request for fine-tuning
type FineTuningJobRequest struct {
	TrainingFile  string  `json:"training_file"`
	ValidationFile string  `json:"validation_file,omitempty"`
	Model         string  `json:"model"`
	Hyperparameters struct {
		BatchSize      int    `json:"batch_size,omitempty"`
		LearningRateMultiplier float64 `json:"learning_rate_multiplier,omitempty"`
		NEpochs       int    `json:"n_epochs,omitempty"`
	} `json:"hyperparameters,omitempty"`
	IntegrationSuffix string `json:"integration_suffix,omitempty"`
	Seed           int64  `json:"seed,omitempty"`
}

// FineTuningJobResponse represents response from fine-tuning
type FineTuningJobResponse struct {
	ID           string `json:"id"`
	Object       string `json:"object"`
	Model        string `json:"model"`
	CreatedAt    int64  `json:"created_at"`
	FinishedAt   *int64 `json:"finished_at,omitempty"`
	TrainingFile string `json:"training_file"`
	ValidationFile *string `json:"validation_file,omitempty"`
	TrainedTokens *int   `json:"trained_tokens,omitempty"`
	Hyperparameters struct {
		BatchSize      int    `json:"batch_size"`
		LearningRateMultiplier float64 `json:"learning_rate_multiplier"`
		NEpochs       int    `json:"n_epochs"`
	} `json:"hyperparameters"`
	OrganizationID string `json:"organization_id,omitempty"`
	Status        string `json:"status"`
	ValidationFileID *string `json:"validation_file_id,omitempty"`
	TrainingFileID string `json:"training_file_id"`
	Error         *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   string `json:"param"`
	} `json:"error,omitempty"`
	UserProvidedSuffix string `json:"user_provided_suffix,omitempty"`
	TrainedModel     *string `json:"trained_model,omitempty"`
	Integrations     []struct {
		Type     string `json:"type"`
		Workflow string `json:"workflow"`
	} `json:"integrations,omitempty"`
	Seed int64 `json:"seed,omitempty"`
	EstimatedMetrics struct {
		FullLoss           float64 `json:"full_loss"`
		TrainingLoss       float64 `json:"training_loss"`
		TrainingPrecision  float64 `json:"training_precision"`
		ValidationLoss    float64 `json:"validation_loss"`
	} `json:"estimated_metrics,omitempty"`
}

// FineTuningJobsResponse represents response from listing fine-tuning jobs
type FineTuningJobsResponse struct {
	Object string `json:"object"`
	Data   []FineTuningJobResponse `json:"data"`
	HasMore bool `json:"has_more"`
}

// ModelsResponse represents response from listing models
type ModelsResponse struct {
	Object string        `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// AssistantRequest represents request for creating assistant
type AssistantRequest struct {
	Model            string                 `json:"model,omitempty"`
	Name             string                 `json:"name,omitempty"`
	Description       string                 `json:"description,omitempty"`
	Instructions     string                 `json:"instructions,omitempty"`
	Tools            []interface{}          `json:"tools,omitempty"`
	ToolResources    map[string]interface{} `json:"tool_resources,omitempty"`
	Temperature      float64               `json:"temperature,omitempty"`
	TopP             float64               `json:"top_p,omitempty"`
	ResponseFormat   interface{}           `json:"response_format,omitempty"`
}

// AssistantResponse represents response from assistant
type AssistantResponse struct {
	ID             string                 `json:"id"`
	Object         string                 `json:"object"`
	CreatedAt      int64                  `json:"created_at"`
	Name           *string                `json:"name,omitempty"`
	Description    *string                `json:"description,omitempty"`
	Model          string                 `json:"model"`
	Instructions   *string                `json:"instructions,omitempty"`
	Tools          []interface{}          `json:"tools,omitempty"`
	ToolResources map[string]interface{} `json:"tool_resources,omitempty"`
	Temperature    *float64               `json:"temperature,omitempty"`
	TopP           *float64               `json:"top_p,omitempty"`
	ResponseFormat interface{}           `json:"response_format,omitempty"`
}

// AssistantsResponse represents response from listing assistants
type AssistantsResponse struct {
	Object string            `json:"object"`
	Data   []AssistantResponse `json:"data"`
	HasMore bool              `json:"has_more"`
}

// ThreadRequest represents request for creating thread
type ThreadRequest struct {
	Messages []struct {
		Role         string                 `json:"role"`
		Content      []interface{}          `json:"content"`
		FileIds      []string               `json:"file_ids,omitempty"`
		Metadata     map[string]interface{} `json:"metadata,omitempty"`
	} `json:"messages,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ThreadResponse represents response from thread
type ThreadResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	CreatedAt int64  `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// FileResponse represents response from file upload
type FileResponse struct {
	ID           string `json:"id"`
	Object       string `json:"object"`
	Bytes        int64  `json:"bytes"`
	CreatedAt    int64  `json:"created_at"`
	Filename     string `json:"filename"`
	Purpose      string `json:"purpose"`
}

// FilesResponse represents response from listing files
type FilesResponse struct {
	Object string `json:"object"`
	Data   []FileResponse `json:"data"`
	HasMore bool `json:"has_more"`
}

// DeleteFileResponse represents response from file deletion
type DeleteFileResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}