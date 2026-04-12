/**
 * Claude LLM Provider 实现
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

	"github.com/weibh/taskmanager/domain"
)

// ClaudeProvider Anthropic Claude provider
type ClaudeProvider struct {
	config    *Config
	client    *http.Client
	lastUsage domain.Usage // 上次调用的 token 使用量
}

var _ domain.LLMClient = (*ClaudeProvider)(nil)

// ClaudeMessage Claude 消息格式
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest Claude 请求格式
type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
	Messages    []ClaudeMessage `json:"messages"`
}

// ClaudeResponse Claude 响应格式
type ClaudeResponse struct {
	ID      string `json:"id"`
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// NewClaudeProvider 创建 Claude Provider
func NewClaudeProvider(config *Config) (*ClaudeProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1/messages"
	}

	return &ClaudeProvider{
		config: config,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Generate 生成文本
func (p *ClaudeProvider) Generate(ctx context.Context, prompt string) (string, error) {
	messages := []ClaudeMessage{
		{Role: "user", Content: prompt},
	}

	reqBody := ClaudeRequest{
		Model:       p.config.Model,
		Messages:    messages,
		Temperature: p.config.Temperature,
		MaxTokens:   p.config.MaxTokens,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL, bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from claude")
	}

	return claudeResp.Content[0].Text, nil
}

// GenerateSubTasks 生成子任务计划
func (p *ClaudeProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*domain.SubTaskPlan, error) {
	prompt := SubTaskPrompt(taskName, taskDesc, depth, maxDepth)

	resp, err := p.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	yamlStr := ExtractYAML(resp)

	plan, err := TryParseYAML(yamlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sub task plan: %w", err)
	}

	return plan, nil
}

// GenerateWithTools 生成文本，支持工具调用 (Claude 版本)
func (p *ClaudeProvider) GenerateWithTools(ctx context.Context, prompt string, tools []*domain.ToolRegistry, maxIterations int) (string, []domain.ToolCall, error) {
	// Claude 的工具调用实现较为复杂，这里暂时委托给普通 Generate
	// TODO: 实现完整的 Claude 工具调用支持
	resp, err := p.Generate(ctx, prompt)
	return resp, nil, err
}

// Name 返回 provider 名称
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// GetLastUsage 返回上次调用的 token 使用量
func (p *ClaudeProvider) GetLastUsage() domain.Usage {
	return p.lastUsage
}
