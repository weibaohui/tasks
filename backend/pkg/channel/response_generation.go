package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/claudecode"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"go.uber.org/zap"
	"github.com/weibh/taskmanager/pkg/bus"
	"github.com/weibh/taskmanager/infrastructure/trace"
)

// generateResponse 生成响应
func (p *MessageProcessor) generateResponse(ctx context.Context, msg *bus.InboundMessage, session *Session, traceID, parentSpanID string) string {
	content := strings.TrimSpace(msg.Content)

	// 检查是否是命令，优先处理命令
	if p.commandProcessor != nil && p.commandProcessor.IsCommand(content) {
		p.logger.Info("执行命令",
			zap.String("session_key", msg.SessionKey()),
			zap.String("content", content),
		)
		return p.commandProcessor.Process(ctx, msg)
	}

	// 获取 Agent 和 LLM 配置
	agent, provider, err := p.getAgentAndProvider(ctx, msg)
	if err != nil {
		p.logger.Debug("获取 Agent/LLM 配置失败", zap.Error(err))
		return fmt.Sprintf("收到消息: %s\n(Agent 或 LLM 配置未找到)", content)
	}

	// 如果没有 Provider，返回默认响应
	if provider == nil {
		return fmt.Sprintf("收到消息: %s\n(LLM Provider 未配置)", content)
	}

	// 检查 Agent 类型，如果是 CodingAgent，使用 ClaudeCodeProcessor 流式处理
	if agent != nil && agent.AgentType().String() == "CodingAgent" {
		p.logger.Info("使用 ClaudeCodeProcessor 流式处理 CodingAgent",
			zap.String("agent_code", agent.AgentCode().String()),
			zap.String("session_key", msg.SessionKey()),
		)
		// 创建 ClaudeCodeSession
		ccSession := &claudecode.ClaudeCodeSession{
			SessionKey:   msg.SessionKey(),
			CliSessionID: session.GetCliSessionID(),
		}
		// 创建流式回调
		callback := newFeishuStreamingCallback(p.bus, p.logger, msg, traceID, parentSpanID, p.hookManager)
		p.updateClaudeCodeRuntimeStatus(ctx, msg.SessionKey(), "running", "")
		// 更新需求的 Claude Runtime 状态
		if requirementID, ok := msg.Metadata["requirement_id"].(string); ok {
			p.updateRequirementClaudeRuntimeStatus(ctx, requirementID, "running", "")
		}
		// 使用流式处理
		err := p.claudeCodeProcessor.ProcessWithStreaming(ctx, msg, ccSession, agent, callback)
		if err != nil {
			p.updateClaudeCodeRuntimeStatus(ctx, msg.SessionKey(), "failed", err.Error())
			if requirementID, ok := msg.Metadata["requirement_id"].(string); ok {
				p.updateRequirementClaudeRuntimeStatus(ctx, requirementID, "failed", err.Error())
			}
			p.logger.Error("ClaudeCodeProcessor 流式处理失败", zap.Error(err))
			return fmt.Sprintf("收到消息: %s\n(Claude Code 处理失败: %v)", content, err)
		}
		p.updateClaudeCodeRuntimeStatus(ctx, msg.SessionKey(), "completed", "")
		if requirementID, ok := msg.Metadata["requirement_id"].(string); ok {
			p.updateRequirementClaudeRuntimeStatus(ctx, requirementID, "completed", "")
		}
		// 更新 session 的 CLI Session ID
		if ccSession.CliSessionID != "" {
			session.SetCliSessionID(ccSession.CliSessionID)
		}
		// 流式处理的消息已通过回调发送，这里返回空字符串避免重复发送
		return ""
	}

	// 构建 LLM 配置
	model := ""
	if agent != nil {
		model = agent.Model()
	}
	if model == "" {
		model = provider.DefaultModel()
	}
	if model == "" {
		supported := provider.SupportedModels()
		if len(supported) > 0 {
			model = supported[0].ID
		}
	}

	// 使用工厂模式创建 LLM Provider
	providerConfig := domain.NewLLMProviderConfig(
		provider.ProviderKey(),
		model,
		provider.APIKey(),
		provider.APIBase(),
	)
	if provider.ProviderType() != "" {
		providerConfig.SetProviderType(provider.ProviderType())
	}

	llmProvider, err := p.factory.Build(providerConfig)
	if err != nil {
		p.logger.Error("创建 LLM Provider 失败", zap.Error(err))
		return fmt.Sprintf("收到消息: %s\n(LLM 配置错误)", content)
	}

	// 构建对话历史 prompt
	prompt := p.buildPrompt(ctx, session, content, agent)

	// 开始 LLM 调用 span
	ctx, llmSpanID := trace.StartSpan(ctx)
	p.logger.Debug("LLM 调用",
		zap.String("trace_id", traceID),
		zap.String("parent_span_id", parentSpanID),
		zap.String("span_id", llmSpanID),
	)

	// 设置 hook context 元数据
	var hookCtx *domain.HookContext
	if p.hookManager != nil {
		hookCtx = domain.NewHookContext(ctx)
		hookCtx.SetMetadata("trace_id", traceID)
		hookCtx.SetMetadata("session_key", msg.SessionKey())
		// channel_code 从 msg.Metadata 获取
		if msg.Metadata != nil {
			if v, ok := msg.Metadata["channel_code"].(string); ok {
				hookCtx.SetMetadata("channel_code", v)
			}
		}
		// channel_type 统一使用 msg.Channel（feishu/dingtalk/wechat 等）
		if msg.Channel != "" {
			hookCtx.SetMetadata("channel_type", msg.Channel)
		}
		if agent != nil {
			hookCtx.SetMetadata("agent_code", agent.AgentCode().String())
			hookCtx.SetMetadata("user_code", agent.UserCode())
		}
		ctx = hookCtx
	}

	// 如果是 EinoProvider，设置工具执行钩子（在 StartSpan 之后创建 adapter）
	if einoProvider, ok := llmProvider.(*llm.EinoProvider); ok && p.hookManager != nil && hookCtx != nil {
		agentCode := ""
		userCode := ""
		if agent != nil {
			agentCode = agent.AgentCode().String()
			userCode = agent.UserCode()
		}
		// 从 msg.Metadata 获取 channel_code
		channelCode := ""
		if msg.Metadata != nil {
			if v, ok := msg.Metadata["channel_code"].(string); ok {
				channelCode = v
			}
		}
		// channel_type 统一使用 msg.Channel（feishu/dingtalk/wechat 等）
		channelType := msg.Channel
		toolHookAdapter := p.newToolHookAdapter(hookCtx, msg.SessionKey(), traceID, llmSpanID, msg.SessionKey(), userCode, agentCode, channelCode, channelType)
		einoProvider.SetToolHooks([]llm.ToolHook{toolHookAdapter})
		einoProvider.SetToolExecutionObserver(toolHookAdapter) // 设置 observer 以监听工具执行
	}

	// 调用 LLM (带工具支持)
	var response string
	var toolCalls []domain.ToolCall

	// 构建 call metadata 从 msg.Metadata
	callMetadata := make(map[string]string)
	if msg.SessionKey() != "" {
		callMetadata["session_key"] = msg.SessionKey()
	}
	// channel_type 统一使用 msg.Channel（feishu/dingtalk/wechat 等）
	if msg.Channel != "" {
		callMetadata["channel_type"] = msg.Channel
	}
	if msg.Metadata != nil {
		if v, ok := msg.Metadata["channel_code"].(string); ok {
			callMetadata["channel_code"] = v
		}
		if v, ok := msg.Metadata["chat_id"].(string); ok {
			callMetadata["chat_id"] = v
		}
	}
	// Agent 查询结果优先（覆盖 msg.Metadata 中的值）
	if agent != nil {
		callMetadata["agent_code"] = agent.AgentCode().String()
		callMetadata["user_code"] = agent.UserCode()
		// 传递思考过程配置
		if agent.EnableThinkingProcess() {
			callMetadata["enable_thinking_process"] = "true"
		}
	}

	// PreLLMCall hook
	if p.hookManager != nil {
		callCtx := &domain.LLMCallContext{
			Prompt:    prompt,
			UserInput: content, // 用户原始输入
			Model:     model,
			SessionID: msg.SessionKey(),
			TraceID:   traceID,
			Metadata:  callMetadata,
		}
		// 复用之前创建的 hookCtx，确保元数据一致性
		if hookCtx == nil {
			hookCtx = domain.NewHookContext(ctx)
			hookCtx.SetMetadata("trace_id", traceID)
			hookCtx.SetMetadata("session_key", msg.SessionKey())
		}
		modifiedCtx, err := p.hookManager.PreLLMCall(hookCtx, callCtx)
		if err != nil {
			p.logger.Error("PreLLMCall hook failed", zap.Error(err))
		} else if modifiedCtx != nil {
			prompt = modifiedCtx.Prompt
		}
	}

	// 构建工具注册表（包括 Agent 指定的 Bash、MCP、Skills 工具）
	toolRegistries := []*domain.ToolRegistry{p.toolRegistry}
	if agent != nil {
		// 构建上下文参数
		contextParams := map[string]string{
			"agentCode":  agent.AgentCode().String(),
			"userCode":   agent.UserCode(),
			"sessionKey": msg.SessionKey(),
			"traceID":    traceID,
			"spanID":     llmSpanID,
		}
		if msg.Metadata != nil {
			if v, ok := msg.Metadata["channel_code"].(string); ok {
				contextParams["channelCode"] = v
			}
		}
		if agentToolsRegistry := p.buildAgentToolsRegistry(ctx, agent, contextParams); agentToolsRegistry != nil {
			toolRegistries = append(toolRegistries, agentToolsRegistry)
		}
	}

	response, toolCalls, err = llmProvider.GenerateWithTools(ctx, prompt, toolRegistries, 5)

	// 获取 token 使用量
	usage := llmProvider.GetLastUsage()

	// PostLLMCall hook
	if p.hookManager != nil {
		callCtx := &domain.LLMCallContext{
			Prompt:    prompt,
			Model:     model,
			SessionID: msg.SessionKey(),
			TraceID:   traceID,
			Metadata:  callMetadata,
		}
		resp := &domain.LLMResponse{
			Content: response,
			Model:   model,
			Usage: domain.Usage{
				PromptTokens:     usage.PromptTokens,
				CompletionTokens: usage.CompletionTokens,
				TotalTokens:      usage.TotalTokens,
			},
		}
		// 构造 RawResponse 包含工具调用信息，供 hook 分析
		if len(toolCalls) > 0 {
			toolCallsInfo := make([]map[string]interface{}, 0, len(toolCalls))
			for _, tc := range toolCalls {
				argsStr := string(tc.Input)
				toolCallsInfo = append(toolCallsInfo, map[string]interface{}{
					"id": tc.ID,
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": argsStr,
					},
				})
			}
			rawJSON, _ := json.Marshal(map[string]interface{}{
				"tool_calls": toolCallsInfo,
			})
			resp.RawResponse = string(rawJSON)
		}
		// 复用之前创建的 hookCtx
		if hookCtx == nil {
			hookCtx = domain.NewHookContext(ctx)
			hookCtx.SetMetadata("trace_id", traceID)
			hookCtx.SetMetadata("session_key", msg.SessionKey())
		}
		_, err := p.hookManager.PostLLMCall(hookCtx, callCtx, resp)
		if err != nil {
			p.logger.Error("PostLLMCall hook failed", zap.Error(err))
		}
	}

	if err != nil {
		p.logger.Error("LLM 调用失败",
			zap.String("trace_id", traceID),
			zap.String("span_id", llmSpanID),
			zap.Error(err),
		)
		return fmt.Sprintf("抱歉，LLM 处理失败: %v", err)
	}

	p.logger.Info("LLM 调用成功",
		zap.String("trace_id", traceID),
		zap.String("span_id", llmSpanID),
		zap.Int("response_length", len(response)),
		zap.Int("tool_calls", len(toolCalls)),
	)

	// 如果有工具调用，在响应中说明
	if len(toolCalls) > 0 {
		p.logger.Info("执行了工具调用",
			zap.String("trace_id", traceID),
			zap.String("span_id", llmSpanID),
			zap.Int("count", len(toolCalls)),
		)
	}

	return response
}
