package providers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// randomIDSuffix returns a short, collision-resistant suffix for IDs in the
// providers package. 8 bytes from crypto/rand (base64url ~11 chars); on RNG
// failure falls back to a doubled-nanosecond value so the suffix is non-empty
// and varies. §11.4.50: two Kimi-Code responses produced within the same
// nanosecond must not share an OpenAI-compatible response ID.
func randomIDSuffix() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano()*2654435761)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// newKimiResponseID builds a unique OpenAI-compatible response ID for the
// Kimi-Code provider. Kept as a named helper so its uniqueness is directly
// testable (§11.4.115).
func newKimiResponseID() string {
	return fmt.Sprintf("kimi-code-cli-%d-%s", time.Now().UnixNano(), randomIDSuffix())
}

const (
	KimiCodeDefaultModel   = "kimi-for-coding"
	KimiCodeAPIEndpoint    = "https://api.kimi.com/coding/v1"
	KimiCodeCredentialPath = ".kimi/credentials/kimi-code.json"
)

var knownKimiCodeModels = []string{
	"kimi-for-coding",
}

type kimiCodeCredential struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	ExpiresAt    float64 `json:"expires_at"`
	Scope        string  `json:"scope"`
	TokenType    string  `json:"token_type"`
}

type KimiCodeCLIAdapter struct {
	BaseAdapter
	cliPath      string
	cliAvailable bool
	checkOnce    sync.Once
	checkErr     error
	timeout      time.Duration
}

type kimiCodeJSONResponse struct {
	Role    string                    `json:"role"`
	Content []kimiCodeContentBlockCLI `json:"content"`
}

type kimiCodeContentBlockCLI struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Think     string `json:"think,omitempty"`
	Encrypted string `json:"encrypted,omitempty"`
}

func NewKimiCodeCLIAdapter(timeout time.Duration) *KimiCodeCLIAdapter {
	if timeout == 0 {
		timeout = 180 * time.Second
	}
	return &KimiCodeCLIAdapter{
		BaseAdapter: BaseAdapter{
			client:   nil,
			endpoint: KimiCodeAPIEndpoint,
			apiKey:   "",
			headers:  map[string]string{},
		},
		timeout: timeout,
	}
}

func (p *KimiCodeCLIAdapter) IsAvailable() bool {
	p.checkOnce.Do(func() {
		path, err := exec.LookPath("kimi")
		if err != nil {
			p.checkErr = fmt.Errorf("kimi command not found: %w", err)
			p.cliAvailable = false
			return
		}
		p.cliPath = path

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, path, "--version")
		if _, err := cmd.CombinedOutput(); err != nil {
			p.checkErr = fmt.Errorf("kimi command failed: %w", err)
			p.cliAvailable = false
			return
		}

		if !p.isAuthenticated() {
			p.checkErr = fmt.Errorf("kimi CLI not authenticated")
			p.cliAvailable = false
			return
		}

		p.cliAvailable = true
	})

	return p.cliAvailable
}

func (p *KimiCodeCLIAdapter) isAuthenticated() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	credPath := filepath.Join(homeDir, KimiCodeCredentialPath)
	data, err := os.ReadFile(credPath)
	if err != nil {
		return false
	}

	var creds kimiCodeCredential
	if err := json.Unmarshal(data, &creds); err != nil {
		return false
	}

	if creds.AccessToken == "" {
		return false
	}

	if creds.ExpiresAt > 0 {
		expiryTime := time.Unix(int64(creds.ExpiresAt), 0)
		if time.Now().After(expiryTime) {
			return false
		}
	}

	return true
}

func (p *KimiCodeCLIAdapter) GetError() error {
	p.IsAvailable()
	return p.checkErr
}

func (p *KimiCodeCLIAdapter) ChatCompletion(ctx context.Context, request OpenAIChatRequest) (*OpenAIChatResponse, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("Kimi Code CLI not available: %v", p.checkErr)
	}

	var promptBuilder strings.Builder
	for _, msg := range request.Messages {
		switch msg.Role {
		case "system":
			promptBuilder.WriteString("System: ")
			promptBuilder.WriteString(msg.Content)
			promptBuilder.WriteString("\n\n")
		case "user":
			promptBuilder.WriteString(msg.Content)
			promptBuilder.WriteString("\n")
		case "assistant":
			promptBuilder.WriteString("Assistant: ")
			promptBuilder.WriteString(msg.Content)
			promptBuilder.WriteString("\n\n")
		}
	}

	prompt := promptBuilder.String()
	if prompt == "" {
		return nil, fmt.Errorf("no prompt provided")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	args := []string{
		"--print",
		"--output-format", "stream-json",
		"-p", prompt,
	}

	cmd := exec.CommandContext(cmdCtx, p.cliPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("kimi CLI timed out after %v", p.timeout)
		}
		return nil, fmt.Errorf("kimi CLI failed: %w (output: %s)", err, string(output))
	}

	content, _ := p.parseJSONResponse(string(output))

	promptTokens := len(prompt) / 4
	completionTokens := len(content) / 4

	model := KimiCodeDefaultModel
	if len(request.Model) > 0 {
		model = request.Model
	}

	response := &OpenAIChatResponse{
		ID:      newKimiResponseID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: content,
				},
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}

	return response, nil
}

func (p *KimiCodeCLIAdapter) parseJSONResponse(rawOutput string) (string, string) {
	rawOutput = strings.TrimSpace(rawOutput)
	var thinking strings.Builder
	var text strings.Builder

	lines := strings.Split(rawOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var resp kimiCodeJSONResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			text.WriteString(line)
			text.WriteString("\n")
			continue
		}

		for _, block := range resp.Content {
			switch block.Type {
			case "think":
				if block.Think != "" {
					thinking.WriteString(block.Think)
					thinking.WriteString("\n")
				}
			case "text":
				if block.Text != "" {
					text.WriteString(block.Text)
				}
			}
		}
	}

	return text.String(), thinking.String()
}

func (p *KimiCodeCLIAdapter) StreamChatCompletion(ctx context.Context, request OpenAIChatRequest) (<-chan OpenAIStreamResponse, <-chan error) {
	responseChan := make(chan OpenAIStreamResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		resp, err := p.ChatCompletion(ctx, request)
		if err != nil {
			errorChan <- err
			return
		}

		if len(resp.Choices) > 0 {
			finishReason := "stop"
			streamResp := OpenAIStreamResponse{
				ID:      resp.ID,
				Object:  "chat.completion.chunk",
				Created: resp.Created,
				Model:   resp.Model,
				Choices: []OpenAIChoice{
					{
						Index: resp.Choices[0].Index,
						Delta: OpenAIDelta{
							Role:    resp.Choices[0].Message.Role,
							Content: resp.Choices[0].Message.Content,
						},
						FinishReason: &finishReason,
					},
				},
			}
			responseChan <- streamResp
		}
	}()

	return responseChan, errorChan
}

func (p *KimiCodeCLIAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	models := p.GetKnownModels()
	resp := &OpenAIModelsResponse{
		Object: "list",
	}
	for _, m := range models {
		resp.Data = append(resp.Data, struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			ID:      m.ID,
			Object:  m.Object,
			Created: m.Created,
			OwnedBy: m.OwnedBy,
		})
	}
	return resp, nil
}

// kimiCodeModel holds model metadata for Kimi Code models.
type kimiCodeModel struct {
	ID      string
	Object  string
	Created int64
	OwnedBy string
}

func (p *KimiCodeCLIAdapter) GetKnownModels() []kimiCodeModel {
	var models []kimiCodeModel
	for _, m := range knownKimiCodeModels {
		models = append(models, kimiCodeModel{
			ID:      m,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "kimi-code",
		})
	}
	return models
}

func (p *KimiCodeCLIAdapter) GetProviderName() string {
	return "kimi-code-cli"
}

func (p *KimiCodeCLIAdapter) SupportsStreaming() bool {
	return true
}

func (p *KimiCodeCLIAdapter) SupportsTools() bool {
	return false
}

func (p *KimiCodeCLIAdapter) HealthCheck(ctx context.Context) error {
	if !p.IsAvailable() {
		return p.checkErr
	}

	checkCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	_, err := p.ChatCompletion(checkCtx, OpenAIChatRequest{
		Model: KimiCodeDefaultModel,
		Messages: []Message{
			{Role: "user", Content: "Reply with just 'OK'"},
		},
		MaxTokens: 10,
	})

	return err
}

func IsKimiCodeCLIInstalled() bool {
	_, err := exec.LookPath("kimi")
	return err == nil
}

func IsKimiCodeCLIAuthenticated() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	credPath := filepath.Join(homeDir, KimiCodeCredentialPath)
	data, err := os.ReadFile(credPath)
	if err != nil {
		return false
	}

	var creds kimiCodeCredential
	if err := json.Unmarshal(data, &creds); err != nil {
		return false
	}

	if creds.AccessToken == "" {
		return false
	}

	if creds.ExpiresAt > 0 {
		expiryTime := time.Unix(int64(creds.ExpiresAt), 0)
		if time.Now().After(expiryTime) {
			return false
		}
	}

	return true
}

func CanUseKimiCodeCLI() bool {
	if os.Getenv("KIMI_CODE_USE_OAUTH_CREDENTIALS") != "true" {
		return false
	}

	return IsKimiCodeCLIInstalled() && IsKimiCodeCLIAuthenticated()
}

func GetKimiCodeCredential() (*kimiCodeCredential, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	credPath := filepath.Join(homeDir, KimiCodeCredentialPath)
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds kimiCodeCredential
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

func (p *KimiCodeCLIAdapter) ReadCloser() io.ReadCloser {
	return nil
}
