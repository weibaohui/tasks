/**
 * Eino LLM Provider 实现
 * 使用 cloudwego/eino 的 ChatModel，支持 OpenAI、Claude 等多种 Provider
 */
package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

// EinoProvider 使用 eino ChatModel 的 Provider
type EinoProvider struct {
	config       *Config
	chatModel    model.ToolCallingChatModel
	logger       *zap.Logger
	lastUsage    domain.Usage
	toolHooks    []ToolHook            // 工具执行钩子
	toolObserver ToolExecutionObserver // 工具执行观察者
	llmCallIndex int                   // 当前 LLM 调用索引
}

// ToolHook 工具执行钩子接口
type ToolHook interface {
	PreToolCall(toolName string, input json.RawMessage) (json.RawMessage, error)
	PostToolCall(toolName string, input json.RawMessage, output string, err error)
}

// ToolHookWithContext 工具执行钩子接口（带 context）
// 用于在工具执行时获取包含 span 信息的 context
type ToolHookWithContext interface {
	GetCurrentCtx() context.Context
}

// ToolCallContext 工具调用上下文（传递给 observer）
type ToolCallContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	ToolName     string
	ToolInput    json.RawMessage
	Output       string
	Success      bool
	Error        error
}

// LLMCallContext LLM 调用上下文（传递给 observer）
type LLMCallContext struct {
	Content   string
	Usage     domain.Usage
	ToolCalls []string // tool names if any
	TraceID   string
	SpanID    string
}

// ToolExecutionObserver 工具执行观察者接口
// 用于在工具执行完成后通知观察者，让其记录相关信息
type ToolExecutionObserver interface {
	// OnLLMCalledWithTools 当 LLM 返回包含 tool_calls 时通知
	// 此时应该记录 llm_response_with_tools
	OnLLMCalledWithTools(ctx context.Context, callCtx LLMCallContext)
	// OnToolExecutionComplete 当一轮工具调用完成后通知
	// 此时应该更新 span 链，供下一个 LLM 调用使用
	OnToolExecutionComplete(ctx context.Context, tools []ToolCallContext)
}

// SetToolHooks 设置工具执行钩子
func (p *EinoProvider) SetToolHooks(hooks []ToolHook) {
	p.toolHooks = hooks
}

// SetToolExecutionObserver 设置工具执行观察者
func (p *EinoProvider) SetToolExecutionObserver(observer ToolExecutionObserver) {
	p.toolObserver = observer
}

var _ domain.LLMClient = (*EinoProvider)(nil)

// NewEinoProvider 创建 Eino Provider
func NewEinoProvider(config *Config, logger *zap.Logger) (*EinoProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  config.APIKey,
		Model:   config.Model,
		BaseURL: baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create eino chat model: %w", err)
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	return &EinoProvider{
		config:    config,
		chatModel: chatModel,
		logger:    logger,
	}, nil
}

// Generate 生成文本
func (p *EinoProvider) Generate(ctx context.Context, prompt string) (string, error) {
	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	resp, err := p.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("eino generate failed: %w", err)
	}

	if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
		p.lastUsage = domain.Usage{
			PromptTokens:     resp.ResponseMeta.Usage.PromptTokens,
			CompletionTokens: resp.ResponseMeta.Usage.CompletionTokens,
			TotalTokens:      resp.ResponseMeta.Usage.TotalTokens,
		}
	}

	return resp.Content, nil
}

// GenerateWithTools 生成文本，支持工具调用。
//
// 注意：当模型返回 ToolCalls 时，必须先把包含 ToolCalls 的 assistant 消息写入 messages，
// 再追加 tool 消息（携带 tool_call_id）。否则部分 OpenAI 兼容网关（例如 Minimax）会报
// “tool result's tool id not found / tool_call_id 找不到”等错误。
func (p *EinoProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*domain.SubTaskPlan, error) {
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

// GetLastUsage 返回上次调用的 token 使用量
func (p *EinoProvider) GetLastUsage() domain.Usage {
	return p.lastUsage
}

// Name 返回 provider 名称
func (p *EinoProvider) Name() string {
	return "eino"
}

// NewEinoChatModel 创建 eino ChatModel（供其他组件直接使用）
func NewEinoChatModel(ctx context.Context, config *Config) (model.ToolCallingChatModel, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  config.APIKey,
		Model:   config.Model,
		BaseURL: baseURL,
	})
	if err != nil {
		return nil, err
	}

	return chatModel, nil
}

