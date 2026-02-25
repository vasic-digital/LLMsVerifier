package providers

import (
	"bufio"
	"context"
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

	content, thinking := p.parseJSONResponse(string(output))

	promptTokens := len(prompt) / 4
	completionTokens := len(content) / 4

	model := KimiCodeDefaultModel
	if len(request.Model) > 0 {
		model = request.Model
	}

	response := &OpenAIChatResponse{
		ID:      fmt.Sprintf("kimi-code-cli-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: OpenAIUsage{
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
			streamResp := OpenAIStreamResponse{
				ID:      resp.ID,
				Object:  "chat.completion.chunk",
				Created: resp.Created,
				Model:   resp.Model,
				Choices: []OpenAIStreamChoice{
					{
						Index: resp.Choices[0].Index,
						Delta: OpenAIDelta{
							Role:    resp.Choices[0].Message.Role,
							Content: resp.Choices[0].Message.Content,
						},
						FinishReason: resp.Choices[0].FinishReason,
					},
				},
			}
			responseChan <- streamResp
		}
	}()

	return responseChan, errorChan
}

func (p *KimiCodeCLIAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	return &OpenAIModelsResponse{
		Object: "list",
		Data:   p.GetKnownModels(),
	}, nil
}

func (p *KimiCodeCLIAdapter) GetKnownModels() []OpenAIModel {
	var models []OpenAIModel
	for _, m := range knownKimiCodeModels {
		models = append(models, OpenAIModel{
			ID:        m,
			Object:    "model",
			Created:   time.Now().Unix(),
			OwnedBy:   "kimi-code",
			Provider:  "kimi-code-cli",
			MaxTokens: 32768,
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
		Messages: []OpenAIMessage{
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
