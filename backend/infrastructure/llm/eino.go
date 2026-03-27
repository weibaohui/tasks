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
	"github.com/eino-contrib/jsonschema"
	"go.uber.org/zap"
)

// EinoProvider 使用 eino ChatModel 的 Provider
type EinoProvider struct {
	config        *Config
	chatModel     model.ToolCallingChatModel
	logger        *zap.Logger
	lastUsage     Usage
	toolHooks     []ToolHook        // 工具执行钩子
	toolObserver  ToolExecutionObserver // 工具执行观察者
	llmCallIndex  int               // 当前 LLM 调用索引
	// 当前工具执行的 span 信息（用于 trace 链路）
	currentSpanID     string
	currentParentSpanID string
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
	Content    string
	Usage      Usage
	ToolCalls  []string // tool names if any
	TraceID    string
	SpanID     string
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

// SetCurrentSpan 设置当前工具执行的 span 信息
func (p *EinoProvider) SetCurrentSpan(spanID, parentSpanID string) {
	p.currentSpanID = spanID
	p.currentParentSpanID = parentSpanID
}

// GetCurrentSpan 获取当前工具执行的 span 信息
func (p *EinoProvider) GetCurrentSpan() (spanID, parentSpanID string) {
	return p.currentSpanID, p.currentParentSpanID
}

var _ LLMProvider = (*EinoProvider)(nil)

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
		p.lastUsage = Usage{
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
func (p *EinoProvider) GenerateWithTools(ctx context.Context, prompt string, toolRegistries []*ToolRegistry, maxIterations int) (string, []ToolCall, error) {
	if maxIterations <= 0 {
		maxIterations = 5
	}

	// 收集所有工具
	toolMap := make(map[string]Tool)
	for _, registry := range toolRegistries {
		if registry != nil {
			for _, t := range registry.List() {
				toolMap[t.Name()] = t
			}
		}
	}

	// 转换为 eino 的 ToolInfo
	var einoTools []*schema.ToolInfo
	for _, t := range toolMap {
		// 解析参数 JSON schema
		var paramSchema *jsonschema.Schema
		if t.Parameters() != nil {
			_ = json.Unmarshal(t.Parameters(), &paramSchema)
		}

		einoTools = append(einoTools, &schema.ToolInfo{
			Name:        t.Name(),
			Desc:        t.Description(),
			ParamsOneOf: schema.NewParamsOneOfByJSONSchema(paramSchema),
		})
	}

	// 创建带工具的模型
	var err error
	boundModel := p.chatModel
	if len(einoTools) > 0 {
		boundModel, err = p.chatModel.WithTools(einoTools)
		if err != nil {
			return "", nil, fmt.Errorf("failed to bind tools: %w", err)
		}
	}

	// 构建消息历史
	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	var allToolCalls []ToolCall

	for iteration := 0; iteration < maxIterations; iteration++ {
		p.logger.Debug("EINO 迭代开始",
			zap.Int("迭代", iteration),
			zap.Int("消息数量", len(messages)),
		)

		resp, err := boundModel.Generate(ctx, messages)
		if err != nil {
			return "", nil, fmt.Errorf("eino generate failed: %w", err)
		}

		p.logger.Debug("EINO 收到响应",
			zap.Int("tool_calls", len(resp.ToolCalls)),
		)

		// 记录 token 使用量
		if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
			p.lastUsage = Usage{
				PromptTokens:     resp.ResponseMeta.Usage.PromptTokens,
				CompletionTokens: resp.ResponseMeta.Usage.CompletionTokens,
				TotalTokens:      resp.ResponseMeta.Usage.TotalTokens,
			}
		}

		// 提取 tool names
		var toolNames []string
		for _, tc := range resp.ToolCalls {
			toolNames = append(toolNames, tc.Function.Name)
		}

		// 如果没有工具调用，返回最终结果
		if len(resp.ToolCalls) == 0 {
			return resp.Content, allToolCalls, nil
		}

		// 通知 observer：LLM 返回了包含 tool_calls 的响应
		// 这是中间响应，应该被记录为 llm_response_with_tools
		if p.toolObserver != nil {
			p.toolObserver.OnLLMCalledWithTools(ctx, LLMCallContext{
				Content:   resp.Content,
				Usage:     p.lastUsage,
				ToolCalls: toolNames,
			})
		}

		messages = append(messages, resp)

		// 收集本轮工具执行上下文
		var toolContexts []ToolCallContext

		// 处理工具调用
		for _, tc := range resp.ToolCalls {
			p.logger.Info("EINO 收到工具调用",
				zap.String("id", tc.ID),
				zap.String("名称", tc.Function.Name),
				zap.String("参数", tc.Function.Arguments),
			)

			toolCall := ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
			}

			// 解析参数
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				args = map[string]interface{}{"raw": tc.Function.Arguments}
			}
			toolCall.Input, _ = json.Marshal(args)

			allToolCalls = append(allToolCalls, toolCall)

			// 查找并执行工具
			t, ok := toolMap[tc.Function.Name]
			if !ok {
				// 工具不存在，添加错误结果
				errMsg := fmt.Sprintf(`{"error": "tool %s not found"}`, tc.Function.Name)
				messages = append(messages, schema.ToolMessage(errMsg, tc.ID))
				// 调用 PostToolCall hooks
				p.callPostToolHooks(tc.Function.Name, toolCall.Input, errMsg, fmt.Errorf("tool not found"))
				toolContexts = append(toolContexts, ToolCallContext{
					ToolName:  tc.Function.Name,
					ToolInput: toolCall.Input,
					Output:    errMsg,
					Success:   false,
					Error:     fmt.Errorf("tool not found"),
				})
				continue
			}

			// 执行工具前调用 PreToolCall hooks
			modifiedInput := toolCall.Input
			toolCtx := ctx // 默认使用原始 context
			for _, hook := range p.toolHooks {
				if modInput, err := hook.PreToolCall(tc.Function.Name, modifiedInput); err == nil {
					modifiedInput = modInput
				}
				// 如果 hook 提供了 context（包含 span 信息），使用它
				if withCtx, ok := hook.(ToolHookWithContext); ok {
					if hookCtx := withCtx.GetCurrentCtx(); hookCtx != nil {
						toolCtx = hookCtx
					}
				}
			}

			// 执行工具
			result, err := t.Execute(toolCtx, modifiedInput)
			if err != nil {
				errMsg := fmt.Sprintf(`{"error": "%v"}`, err)
				messages = append(messages, schema.ToolMessage(errMsg, tc.ID))
				// 调用 PostToolCall hooks
				p.callPostToolHooks(tc.Function.Name, modifiedInput, errMsg, err)
				toolContexts = append(toolContexts, ToolCallContext{
					ToolName:  tc.Function.Name,
					ToolInput: modifiedInput,
					Output:    errMsg,
					Success:   false,
					Error:     err,
				})
				continue
			}

			// 添加工具结果到消息历史
			output := result.Output
			if result.Error != "" {
				output = fmt.Sprintf(`{"error": "%s", "output": "%s"}`, result.Error, output)
			}
			p.logger.Info("EINO 发送工具结果",
				zap.String("tool_call_id", tc.ID),
				zap.Int("输出长度", len(output)),
			)
			messages = append(messages, schema.ToolMessage(output, tc.ID))

			// 调用 PostToolCall hooks
			p.callPostToolHooks(tc.Function.Name, modifiedInput, output, nil)
			toolContexts = append(toolContexts, ToolCallContext{
				ToolName:  tc.Function.Name,
				ToolInput: modifiedInput,
				Output:    output,
				Success:   true,
			})
		}

		// 通知 observer：一轮工具调用完成
		// 此时下一个 LLM 调用将以 tool_call span 为 parent
		if p.toolObserver != nil && len(toolContexts) > 0 {
			p.toolObserver.OnToolExecutionComplete(ctx, toolContexts)
		}
	}

	// 如果循环结束还没返回，返回最后的响应
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		return lastMsg.Content, allToolCalls, nil
	}

	return "", allToolCalls, nil
}

// GenerateSubTasks 生成子任务计划
func (p *EinoProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error) {
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
func (p *EinoProvider) GetLastUsage() Usage {
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

// BindTools 绑定工具到模型
func BindTools(ctx context.Context, chatModel model.ToolCallingChatModel, regs []*ToolRegistry) (model.ToolCallingChatModel, error) {
	if len(regs) == 0 {
		return chatModel, nil
	}

	// 收集所有工具
	var einoTools []*schema.ToolInfo
	for _, registry := range regs {
		if registry != nil {
			for _, t := range registry.List() {
				// 解析参数 JSON schema
				var paramSchema *jsonschema.Schema
				if t.Parameters() != nil {
					_ = json.Unmarshal(t.Parameters(), &paramSchema)
				}

				einoTools = append(einoTools, &schema.ToolInfo{
					Name:        t.Name(),
					Desc:        t.Description(),
					ParamsOneOf: schema.NewParamsOneOfByJSONSchema(paramSchema),
				})
			}
		}
	}

	if len(einoTools) == 0 {
		return chatModel, nil
	}

	return chatModel.WithTools(einoTools)
}

// BuildMessages 构建消息列表
func BuildMessages(prompt string, history []*schema.Message) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history)+1)

	// 添加历史消息
	if len(history) > 0 {
		messages = append(messages, history...)
	}

	// 添加当前用户消息
	messages = append(messages, schema.UserMessage(prompt))

	return messages
}

// callPostToolHooks 调用所有 PostToolCall hooks
func (p *EinoProvider) callPostToolHooks(toolName string, input json.RawMessage, output string, toolErr error) {
	for _, hook := range p.toolHooks {
		hook.PostToolCall(toolName, input, output, toolErr)
	}
}
