/**
 * Eino LLM Provider 实现
 * 使用 cloudwego/eino 的 ChatModel，支持 OpenAI、Claude 等多种 Provider
 */
package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/eino-contrib/jsonschema"
	"go.uber.org/zap"
)

// EinoProvider 使用 eino ChatModel 的 Provider
type EinoProvider struct {
	config    *Config
	chatModel model.ToolCallingChatModel
	logger    *zap.Logger
	lastUsage Usage
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

// GenerateWithTools 生成文本，支持工具调用
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
			Name: t.Name(),
			Desc: t.Description(),
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
		fmt.Printf("[EINO] 迭代 %d, 消息数量: %d\n", iteration, len(messages))

		resp, err := boundModel.Generate(ctx, messages)
		if err != nil {
			return "", nil, fmt.Errorf("eino generate failed: %w", err)
		}

		fmt.Printf("[EINO] 收到响应，ToolCalls数量: %d\n", len(resp.ToolCalls))

		// 记录 token 使用量
		if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
			p.lastUsage = Usage{
				PromptTokens:     resp.ResponseMeta.Usage.PromptTokens,
				CompletionTokens: resp.ResponseMeta.Usage.CompletionTokens,
				TotalTokens:      resp.ResponseMeta.Usage.TotalTokens,
			}
		}

		// 如果没有工具调用，返回最终结果
		if len(resp.ToolCalls) == 0 {
			return resp.Content, allToolCalls, nil
		}

		// 处理工具调用
		for _, tc := range resp.ToolCalls {
			fmt.Printf("[EINO] 收到工具调用: ID=%s, Name=%s, Arguments=%s\n", tc.ID, tc.Function.Name, tc.Function.Arguments)

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
				messages = append(messages, schema.ToolMessage(
					fmt.Sprintf(`{"error": "tool %s not found"}`, tc.Function.Name),
					tc.ID,
				))
				continue
			}

			// 执行工具
			result, err := t.Execute(ctx, toolCall.Input)
			if err != nil {
				messages = append(messages, schema.ToolMessage(
					fmt.Sprintf(`{"error": "%v"}`, err),
					tc.ID,
				))
				continue
			}

			// 添加工具结果到消息历史
			output := result.Output
			if result.Error != "" {
				output = fmt.Sprintf(`{"error": "%s", "output": "%s"}`, result.Error, output)
			}
			fmt.Printf("[EINO] 发送工具结果: ToolCallID=%s, Output=%s\n", tc.ID, output)
			messages = append(messages, schema.ToolMessage(output, tc.ID))
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
					Name: t.Name(),
					Desc: t.Description(),
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
