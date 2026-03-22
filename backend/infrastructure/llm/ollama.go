/**
 * Ollama LLM Provider 实现
 */
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaProvider Ollama 本地模型 provider
type OllamaProvider struct {
	config *Config
	client *http.Client
}

var _ LLMProvider = (*OllamaProvider)(nil)

// OllamaRequest Ollama 请求格式
type OllamaRequest struct {
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool   `json:"stream"`
}

// OllamaResponse Ollama 响应格式
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// NewOllamaProvider 创建 Ollama Provider
func NewOllamaProvider(config *Config) (*OllamaProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaProvider{
		config: config,
		client: &http.Client{
			Timeout: 300 * time.Second, // Ollama 可能需要更长时间
		},
	}, nil
}

// Generate 生成文本
func (p *OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:       p.config.Model,
		Prompt:      prompt,
		Temperature: p.config.Temperature,
		Stream:      false,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", p.config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return ollamaResp.Response, nil
}

// GenerateSubTasks 生成子任务计划
func (p *OllamaProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error) {
	prompt := subTaskPrompt(taskName, taskDesc, depth, maxDepth)

	resp, err := p.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var plan SubTaskPlan

	jsonStr := extractJSON(resp)

	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		planPtr, err := tryFixAndParseJSON(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sub task plan: %w", err)
		}
		plan = *planPtr
	}

	return &plan, nil
}

// Name 返回 provider 名称
func (p *OllamaProvider) Name() string {
	return "ollama"
}
