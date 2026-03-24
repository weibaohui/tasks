/**
 * OpenAI LLM Provider 实现
 * 使用直接 HTTP 调用 OpenAI API
 */
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIProvider OpenAI GPT 系列 provider
type OpenAIProvider struct {
	config     *Config
	client     *http.Client
	lastUsage  Usage  // 上次调用的 token 使用量
}

var _ LLMProvider = (*OpenAIProvider)(nil)

// OpenAIMessage OpenAI 消息格式
type OpenAIMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []OpenAIMessageToolCall `json:"tool_calls,omitempty"`
}

// OpenAIMessageToolCall OpenAI 消息中的工具调用
type OpenAIMessageToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function OpenAIMessageFunction `json:"function"`
}

// OpenAIMessageFunction 函数调用
type OpenAIMessageFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAIRequest OpenAI 请求格式
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Tools       []ToolInfo      `json:"tools,omitempty"`
}

// OpenAIResponse OpenAI 响应格式
type OpenAIResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Usage OpenAI token 使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Choice 选择
type Choice struct {
	Message       OpenAIMessage `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

// NewOpenAIProvider 创建 OpenAI Provider
func NewOpenAIProvider(config *Config) (*OpenAIProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIProvider{
		config: config,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Generate 生成文本
func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	messages := []OpenAIMessage{
		{Role: "user", Content: prompt},
	}

	reqBody := OpenAIRequest{
		Model:       p.config.Model,
		Messages:    messages,
		Temperature: p.config.Temperature,
		MaxTokens:   p.config.MaxTokens,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.config.BaseURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	} else {
		url = strings.TrimSuffix(url, "/")
		if !strings.HasSuffix(url, "/chat/completions") {
			url = url + "/chat/completions"
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from openai")
	}

	// 保存 Usage
	p.lastUsage = openAIResp.Usage

	return openAIResp.Choices[0].Message.Content, nil
}

// GetLastUsage 返回上次调用的 token 使用量
func (p *OpenAIProvider) GetLastUsage() Usage {
	return p.lastUsage
}

// GenerateWithTools 生成文本，支持工具调用
func (p *OpenAIProvider) GenerateWithTools(ctx context.Context, prompt string, toolRegistries []*ToolRegistry, maxIterations int) (string, []ToolCall, error) {
	if maxIterations <= 0 {
		maxIterations = 5
	}

	// 构建消息历史
	messages := []OpenAIMessage{
		{Role: "user", Content: prompt},
	}

	// 收集所有工具
	var allTools []ToolInfo
	for _, registry := range toolRegistries {
		if registry != nil {
			allTools = append(allTools, registry.GetToolInfos()...)
		}
	}

	var toolCalls []ToolCall

	for iteration := 0; iteration < maxIterations; iteration++ {
		// 构建请求
		reqBody := OpenAIRequest{
			Model:       p.config.Model,
			Messages:    messages,
			Temperature: p.config.Temperature,
			MaxTokens:   p.config.MaxTokens,
			Tools:       allTools,
		}

		reqJSON, err := json.Marshal(reqBody)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		url := p.config.BaseURL
		if url == "" {
			url = "https://api.openai.com/v1/chat/completions"
		} else {
			url = strings.TrimSuffix(url, "/")
			if !strings.HasSuffix(url, "/chat/completions") {
				url = url + "/chat/completions"
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqJSON))
		if err != nil {
			return "", nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

		resp, err := p.client.Do(req)
		if err != nil {
			return "", nil, fmt.Errorf("openai request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(body))
		}

		var openAIResp OpenAIResponse
		if err := json.Unmarshal(body, &openAIResp); err != nil {
			return "", nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if len(openAIResp.Choices) == 0 {
			return "", nil, fmt.Errorf("empty response from openai")
		}

		choice := openAIResp.Choices[0]
		p.lastUsage = openAIResp.Usage

		// 添加助手消息到历史
		assistantMsg := OpenAIMessage{
			Role:      choice.Message.Role,
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// 如果没有工具调用，返回最终结果
		if len(choice.Message.ToolCalls) == 0 {
			return choice.Message.Content, toolCalls, nil
		}

		// 处理工具调用
		for _, tc := range choice.Message.ToolCalls {
			toolCall := ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			}
			toolCalls = append(toolCalls, toolCall)

			// 在消息历史中添加 tool 类型消息
			messages = append(messages, OpenAIMessage{
				Role: "tool",
				Content: "", // 暂不填充，等执行完再填充
			})
		}

		// 注意：这里需要执行工具并填充结果
		// 由于需要访问 registry，这部分逻辑应该在调用方处理
		// 这里先返回，让调用方执行工具后继续迭代
		break
	}

	// 如果循环结束还没返回，返回最后的消息内容
	lastMsg := messages[len(messages)-1]
	return lastMsg.Content, toolCalls, nil
}

// GenerateSubTasks 生成子任务计划
func (p *OpenAIProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error) {
	prompt := SubTaskPrompt(taskName, taskDesc, depth, maxDepth)

	resp, err := p.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// 解析 YAML 响应
	yamlStr := ExtractYAML(resp)

	plan, err := TryParseYAML(yamlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sub task plan: %w", err)
	}

	return plan, nil
}

// Name 返回 provider 名称
func (p *OpenAIProvider) Name() string {
	return "openai"
}
