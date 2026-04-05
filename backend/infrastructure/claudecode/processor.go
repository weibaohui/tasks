package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	claudecode "github.com/severity1/claude-agent-sdk-go"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

const defaultTokenAggregationLimit = 1000 // Token 聚合查询默认限制

// StreamingCallback 回调接口用于流式输出
type StreamingCallback interface {
	OnThinking(thinking string)
	OnToolCall(toolName string, input map[string]any)
	OnToolResult(toolName string, result string)
	OnText(text string)
	OnComplete(finalResult string)
	GetFinalResult() string
}

// toolHookAdapter bridges Claude Code SDK hooks to the domain hook system
type toolHookAdapter struct {
	hookManager *hook.Manager
	logger      *zap.Logger
	hookCtx     *domain.HookContext
	sessionKey  string
	userCode    string
	agentCode   string
	channelCode string
	channelType string
	traceID     string
}

// preToolUseAdapter converts Claude Code PreToolUse hook to domain.ToolHook
func (a *toolHookAdapter) preToolUseAdapter(ctx context.Context, input any, toolUseID *string, hookCtx claudecode.HookContext) (claudecode.HookJSONOutput, error) {
	preInput, ok := input.(*claudecode.PreToolUseHookInput)
	if !ok {
		a.logger.Warn("ClaudeCode PreToolUse: unexpected input type")
		return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
	}

	// 在 Claude Code SDK 路径中，PostLLMCall（会设置 ToolParentSpanKey）在工具调用之后才执行（defer）。
	// 因此首次 PreToolCall 时 ToolParentSpanKey 还未设置，需要在这里主动设置，
	// 确保连续工具调用共享同一个父级（llm_response_with_tools），而不是互相嵌套。
	if a.hookCtx.Get(hooks.ToolParentSpanKey) == nil {
		if currentSpan, ok := a.hookCtx.Get(hooks.SpanKey).(string); ok && currentSpan != "" {
			a.hookCtx.WithValue(hooks.ToolParentSpanKey, currentSpan)
		}
	}

	// Convert to domain.ToolCallContext
	callCtx := &domain.ToolCallContext{
		ToolName:     preInput.ToolName,
		ToolInput:    preInput.ToolInput,
		SessionID:    a.sessionKey,
		TraceID:      a.traceID,
		SpanID:       "",
		ParentSpanID: "",
	}

	// Execute PreToolCall hooks
	_, err := a.hookManager.PreToolCall(a.hookCtx, callCtx)
	if err != nil {
		a.logger.Error("ClaudeCode PreToolUse hook failed",
			zap.String("tool", preInput.ToolName),
			zap.Error(err))
	}

	// Return continue=true to allow tool execution
	return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
}

// postToolUseAdapter converts Claude Code PostToolUse hook to domain.ToolHook
func (a *toolHookAdapter) postToolUseAdapter(ctx context.Context, input any, toolUseID *string, hookCtx claudecode.HookContext) (claudecode.HookJSONOutput, error) {
	postInput, ok := input.(*claudecode.PostToolUseHookInput)
	if !ok {
		a.logger.Warn("ClaudeCode PostToolUse: unexpected input type")
		return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
	}

	// Convert to domain.ToolCallContext
	callCtx := &domain.ToolCallContext{
		ToolName:     postInput.ToolName,
		ToolInput:    postInput.ToolInput,
		SessionID:    a.sessionKey,
		TraceID:      a.traceID,
		SpanID:       "",
		ParentSpanID: "",
	}

	// Convert tool response to ToolExecutionResult
	execResult := &domain.ToolExecutionResult{
		Success:  true,
		Duration: 0,
	}

	// Handle tool response
	if postInput.ToolResponse != nil {
		if respBytes, err := json.Marshal(postInput.ToolResponse); err == nil {
			execResult.Output = string(respBytes)
		} else {
			execResult.Output = fmt.Sprintf("%v", postInput.ToolResponse)
		}
	}

	// Execute PostToolCall hooks
	_, err := a.hookManager.PostToolCall(a.hookCtx, callCtx, execResult)
	if err != nil {
		a.logger.Error("ClaudeCode PostToolUse hook failed",
			zap.String("tool", postInput.ToolName),
			zap.Error(err))
	}

	// Return continue=true
	return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
}

func boolPtr(b bool) *bool {
	return &b
}

// ClaudeCodeProcessor 处理 CodingAgent 类型消息的 Claude Code 会话
type ClaudeCodeProcessor struct {
	logger              *zap.Logger
	hookManager         *hook.Manager
	providerRepo        domain.LLMProviderRepository
	idGenerator         domain.IDGenerator
	requirementRepo     domain.RequirementRepository
	conversationRepo    domain.ConversationRecordRepository
	replicaAgentManager *domain.ReplicaAgentManager
}

// ClaudeCodeProcessorInterface 定义 Claude Code 处理器的接口
type ClaudeCodeProcessorInterface interface {
	Process(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agent *domain.Agent) (string, error)
	ProcessWithStreaming(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agent *domain.Agent, callback StreamingCallback) error
}

// ClaudeCodeSession 会话上下文（包含 CLI Session ID）
type ClaudeCodeSession struct {
	SessionKey   string
	CliSessionID string
}

// NewClaudeCodeProcessor 创建 ClaudeCodeProcessor
func NewClaudeCodeProcessor(
	logger *zap.Logger,
	hookManager *hook.Manager,
	providerRepo domain.LLMProviderRepository,
	idGenerator domain.IDGenerator,
	requirementRepo domain.RequirementRepository,
	replicaAgentManager *domain.ReplicaAgentManager,
	conversationRepo domain.ConversationRecordRepository,
) *ClaudeCodeProcessor {
	return &ClaudeCodeProcessor{
		logger:              logger,
		hookManager:         hookManager,
		providerRepo:        providerRepo,
		idGenerator:         idGenerator,
		requirementRepo:     requirementRepo,
		conversationRepo:    conversationRepo,
		replicaAgentManager: replicaAgentManager,
	}
}

// Process 处理 CodingAgent 消息
func (p *ClaudeCodeProcessor) Process(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agent *domain.Agent) (string, error) {
	// 优先从 context 获取已有的 trace_id，如果不存在则生成新的
	traceID := trace.MustGetTraceID(ctx)
	spanID := trace.MustGetSpanID(ctx)
	if traceID == "" {
		var newCtx context.Context
		newCtx, traceID, spanID = trace.StartTrace(ctx)
		ctx = newCtx
	}
	_ = spanID // 未使用

	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	agentCode := ""
	if agent != nil {
		agentCode = agent.AgentCode().String()
	}
	p.logger.Info("ClaudeCode 处理消息",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("session_key", msg.SessionKey()),
		zap.String("agent_code", agentCode),
		zap.String("内容", preview),
	)

	// 保存 trace_id 到需求表
	if msg.Metadata != nil {
		if requirementIDStr, ok := msg.Metadata["requirement_id"].(string); ok && requirementIDStr != "" {
			requirementID := domain.NewRequirementID(requirementIDStr)
			requirement, err := p.requirementRepo.FindByID(ctx, requirementID)
			if err != nil {
				p.logger.Warn("查找需求失败，无法保存 trace_id", zap.Error(err))
			} else if requirement != nil {
				requirement.SetTraceID(traceID)
				if err := p.requirementRepo.Save(ctx, requirement); err != nil {
					p.logger.Warn("保存 trace_id 到需求表失败", zap.Error(err))
				} else {
					p.logger.Info("已保存 trace_id 到需求表",
						zap.String("requirement_id", requirementIDStr),
						zap.String("trace_id", traceID),
					)
				}
			}
		}
	}

	// 获取 Provider 配置（优先使用 Agent 指定的 Provider）
	provider, err := p.resolveProvider(ctx, agent)
	if err != nil {
		p.logger.Warn("获取 LLM Provider 失败，使用默认配置", zap.Error(err))
		provider = nil
	}

	// 获取超时配置
	timeout := 120
	if agent != nil && agent.ClaudeCodeConfig() != nil && agent.ClaudeCodeConfig().Timeout > 0 {
		timeout = agent.ClaudeCodeConfig().Timeout
	}

	// 调用 Claude Code（使用独立的 context，避免被取消）
	queryCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 从会话获取 CLI Session UUID，用于会话恢复
	cliSessionID := ""
	if session != nil {
		cliSessionID = session.CliSessionID
	}

	// 调用 Claude Code
	response, newCliSessionID, err := p.queryClaudeCode(queryCtx, msg, cliSessionID, traceID, provider, agent)
	if err != nil {
		p.logger.Error("Claude Code 调用失败", zap.Error(err))
		return "", fmt.Errorf("Claude Code 调用失败: %w", err)
	}

	// 如果返回了新的 CLI Session ID，更新会话
	if newCliSessionID != "" && session != nil {
		session.CliSessionID = newCliSessionID
		p.logger.Info("Claude Code 会话已保存",
			zap.String("session_key", msg.SessionKey()),
			zap.String("cli_session_id", newCliSessionID),
		)
	}

	p.logger.Info("Claude Code 返回响应",
		zap.String("session_key", msg.SessionKey()),
		zap.String("response_length", fmt.Sprintf("%d", len(response))),
		zap.String("response_preview", func() string {
			if len(response) > 100 {
				return response[:100] + "..."
			}
			return response
		}()),
	)

	return response, nil
}

// ProcessWithStreaming 处理 CodingAgent 消息，带流式回调
func (p *ClaudeCodeProcessor) ProcessWithStreaming(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agent *domain.Agent, callback StreamingCallback) error {
	// 优先从 context 获取已有的 trace_id，如果不存在则生成新的
	traceID := trace.MustGetTraceID(ctx)
	spanID := trace.MustGetSpanID(ctx)
	if traceID == "" {
		var newCtx context.Context
		newCtx, traceID, spanID = trace.StartTrace(ctx)
		ctx = newCtx
	}
	_ = spanID // 未使用

	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	agentCode := ""
	if agent != nil {
		agentCode = agent.AgentCode().String()
	}
	p.logger.Info("ClaudeCode 流式处理消息",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("session_key", msg.SessionKey()),
		zap.String("agent_code", agentCode),
		zap.String("内容", preview),
	)

	// 保存 trace_id 到需求表
	if msg.Metadata != nil {
		if requirementIDStr, ok := msg.Metadata["requirement_id"].(string); ok && requirementIDStr != "" {
			requirementID := domain.NewRequirementID(requirementIDStr)
			requirement, err := p.requirementRepo.FindByID(ctx, requirementID)
			if err != nil {
				p.logger.Warn("查找需求失败，无法保存 trace_id", zap.Error(err))
			} else if requirement != nil {
				requirement.SetTraceID(traceID)
				if err := p.requirementRepo.Save(ctx, requirement); err != nil {
					p.logger.Warn("保存 trace_id 到需求表失败", zap.Error(err))
				} else {
					p.logger.Info("已保存 trace_id 到需求表",
						zap.String("requirement_id", requirementIDStr),
						zap.String("trace_id", traceID),
					)
				}
			}
		}
	}

	// 获取 Provider 配置（优先使用 Agent 指定的 Provider）
	provider, err := p.resolveProvider(ctx, agent)
	if err != nil {
		p.logger.Warn("获取 LLM Provider 失败，使用默认配置", zap.Error(err))
		provider = nil
	}

	// 获取超时配置
	timeout := 120
	if agent != nil && agent.ClaudeCodeConfig() != nil && agent.ClaudeCodeConfig().Timeout > 0 {
		timeout = agent.ClaudeCodeConfig().Timeout
	}

	// 调用 Claude Code（使用独立的 context）
	queryCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cliSessionID := ""
	if session != nil {
		cliSessionID = session.CliSessionID
	}

	// 流式调用 Claude Code
	newCliSessionID, err := p.queryClaudeCodeStreaming(queryCtx, msg, msg.Content, cliSessionID, traceID, provider, agent, callback)
	if err != nil {
		p.logger.Error("Claude Code 流式调用失败", zap.Error(err))
		// 即使出错也要触发清理 hook
		p.triggerClaudeCodeFinishedHook(ctx, msg, agent, false, "")
		return fmt.Errorf("Claude Code 调用失败: %w", err)
	}

	// 如果返回了新的 CLI Session ID，更新会话
	if newCliSessionID != "" && session != nil {
		session.CliSessionID = newCliSessionID
		p.logger.Info("Claude Code 会话已保存",
			zap.String("session_key", msg.SessionKey()),
			zap.String("cli_session_id", newCliSessionID),
		)
	}

	// 获取最终结果
	finalResult := ""
	if callback != nil {
		finalResult = callback.GetFinalResult()
	}

	// Claude Code 执行完成，触发 claude_code_finished hook
	p.triggerClaudeCodeFinishedHook(ctx, msg, agent, true, finalResult)

	return nil
}

// queryClaudeCodeStreaming 流式调用 Claude Code SDK
func (p *ClaudeCodeProcessor) queryClaudeCodeStreaming(ctx context.Context, msg *bus.InboundMessage, userInput, cliSessionID, traceID string, provider *domain.LLMProvider, agent *domain.Agent, callback StreamingCallback) (string, error) {
	sessionKey := msg.SessionKey()

	// 创建工具钩子适配器
	var ccToolHookAdapter *toolHookAdapter
	var hookCtx *domain.HookContext
	var result string
	var llmCallCtx *domain.LLMCallContext
	var llmUsage = &domain.Usage{}

	if p.hookManager != nil {
		hookCtx = domain.NewHookContext(ctx)
		hookCtx.SetMetadata("session_key", sessionKey)
		hookCtx.SetMetadata("trace_id", traceID)
		// 启用思考过程，让 FeishuThinkingProcessHook 发送中间过程卡片
		hookCtx.SetMetadata("enable_thinking_process", "true")
		// 设置渠道信息，供 sendThinkingMessage 使用
		hookCtx.SetMetadata("chat_id", msg.ChatID)
		hookCtx.SetMetadata("channel_type", msg.Channel)

		userCode := ""
		agentCode := ""
		if agent != nil {
			agentCode = agent.AgentCode().String()
			userCode = agent.UserCode()
		}

		// 从 msg.Metadata 提取 channel_code
		channelCode := ""
		if msg.Metadata != nil {
			if v, ok := msg.Metadata["channel_code"].(string); ok {
				channelCode = v
			}
		}

		ccToolHookAdapter = &toolHookAdapter{
			hookManager: p.hookManager,
			logger:      p.logger,
			hookCtx:     hookCtx,
			sessionKey:  sessionKey,
			userCode:    userCode,
			agentCode:   agentCode,
			channelCode: channelCode,
			channelType: msg.Channel,
			traceID:     traceID,
		}

		// 构建 LLMCallContext 并调用 PreLLMCall hooks
		llmCallCtx = &domain.LLMCallContext{
			Prompt:    userInput,
			UserInput: userInput,
			Model:     "claude_code",
			SessionID: sessionKey,
			TraceID:   traceID,
			Metadata: map[string]string{
				"session_key":  sessionKey,
				"trace_id":     traceID,
				"user_code":    userCode,
				"agent_code":   agentCode,
				"channel_code": channelCode,
				"channel_type": msg.Channel,
				"chat_id":      msg.ChatID,
			},
		}

		// 调用 PreLLMCall hooks
		modifiedCtx, err := p.hookManager.PreLLMCall(hookCtx, llmCallCtx)
		if err != nil {
			p.logger.Error("PreLLMCall failed", zap.Error(err))
		}
		if modifiedCtx != nil {
			llmCallCtx = modifiedCtx
		}

		// 确保 PostLLMCall 和 OnToolExecutionComplete 被调用
		// 使用 hookCtx（而非新建 context），确保 span 状态在 PreToolCall/PostToolCall 之间正确共享
		defer func() {
			resp := &domain.LLMResponse{Content: result, Usage: domain.Usage{}}
			if llmUsage != nil {
				resp.Usage = *llmUsage
			}
			p.hookManager.PostLLMCall(hookCtx, llmCallCtx, resp)
			// 工具执行完成后，写入延迟的最终 llm_response
			p.hookManager.OnToolExecutionComplete(hookCtx)
		}()
	}

	// 构建选项
	opts := p.buildOptions(provider, cliSessionID, agent, ccToolHookAdapter)

	p.logger.Info("开始 Claude Code 流式查询",
		zap.String("session_key", sessionKey),
		zap.String("cli_session_id", cliSessionID),
	)

	startTime := time.Now()

	// 使用 Client 接口进行流式处理
	client := claudecode.NewClient(opts...)

	if err := client.Connect(ctx); err != nil {
		p.logger.Error("Claude Code Connect 失败",
			zap.String("session_key", sessionKey),
			zap.Duration("duration", time.Since(startTime)),
			zap.Error(err),
		)
		return "", fmt.Errorf("Claude Code Connect 失败: %w", err)
	}
	defer client.Disconnect()

	p.logger.Info("Claude Code 连接成功，开始流式查询",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
	)

	// 使用 QueryWithSession 发送查询
	sessionID := cliSessionID
	if sessionID == "" {
		sessionID = "default"
	}
	if err := client.QueryWithSession(ctx, userInput, sessionID); err != nil {
		p.logger.Error("Claude Code QueryWithSession 失败",
			zap.String("session_key", sessionKey),
			zap.Duration("duration", time.Since(startTime)),
			zap.Error(err),
		)
		return "", fmt.Errorf("Claude Code QueryWithSession 失败: %w", err)
	}

	// 流式接收消息
	var cliSessionIDResult string
	var mu sync.Mutex

	msgChan := client.ReceiveMessages(ctx)
	for msg := range msgChan {
		if msg == nil {
			continue
		}

		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case *claudecode.TextBlock:
					mu.Lock()
					result += b.Text
					mu.Unlock()
					callback.OnText(b.Text)
				case *claudecode.ToolUseBlock:
					p.logger.Info("Claude Code 工具调用",
						zap.String("session_key", sessionKey),
						zap.String("tool_name", b.Name),
					)
					toolInput := make(map[string]any)
					if b.Input != nil {
						toolInput = b.Input
					}
					callback.OnToolCall(b.Name, toolInput)
				case *claudecode.ToolResultBlock:
					p.logger.Info("Claude Code 工具结果",
						zap.String("session_key", sessionKey),
						zap.Any("content", b.Content),
					)
					content := fmt.Sprintf("%v", b.Content)
					mu.Lock()
					result += content
					mu.Unlock()
					callback.OnToolResult("", content)
				case *claudecode.ThinkingBlock:
					// 只发送思考卡片，不累积到 result
					callback.OnThinking(b.Thinking)
				}
			}
		case *claudecode.ResultMessage:
			if m.SessionID != "" {
				cliSessionIDResult = m.SessionID
			}
			if m.IsError && m.Result != nil {
				result += fmt.Sprintf("\n[错误: %s]", *m.Result)
			}
			// 捕获 token usage（Claude CLI 在流式模式下 output_tokens 始终为 0）
			// 使用所有可用 token 字段求和作为兜底方案
			if m.Usage != nil && llmUsage != nil {
				llmUsage.PromptTokens = getUsageInt(*m.Usage, "input_tokens")
				llmUsage.CompletionTokens = getUsageInt(*m.Usage, "output_tokens")
				cacheRead := getUsageInt(*m.Usage, "cache_read_input_tokens")
				cacheCreate := getUsageInt(*m.Usage, "cache_creation_input_tokens")
				llmUsage.TotalTokens = llmUsage.PromptTokens + llmUsage.CompletionTokens + cacheRead + cacheCreate
				p.logger.Info("Token usage captured",
					zap.Any("usage", m.Usage),
					zap.Int("prompt", llmUsage.PromptTokens),
					zap.Int("completion", llmUsage.CompletionTokens),
					zap.Int("cache_read", cacheRead),
					zap.Int("cache_create", cacheCreate),
					zap.Int("total", llmUsage.TotalTokens),
				)
			} else {
				p.logger.Warn("Token usage not available",
					zap.Any("m.Usage", m.Usage),
					zap.Any("llmUsage", llmUsage),
				)
			}
			// ResultMessage 表示流式结束，立即调用 OnComplete 并退出
			callback.OnComplete(result)
			return cliSessionIDResult, nil
		case *claudecode.UserMessage:
			// 用户消息，不处理
		}
	}

	p.logger.Info("Claude Code 流式接收完成",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
	)

	// 如果循环正常结束（channel 关闭）但没有收到 ResultMessage，也调用 OnComplete
	callback.OnComplete(result)
	return cliSessionIDResult, nil
}

// queryClaudeCode 调用 Claude Code SDK
func (p *ClaudeCodeProcessor) queryClaudeCode(ctx context.Context, msg *bus.InboundMessage, cliSessionID, traceID string, provider *domain.LLMProvider, agent *domain.Agent) (string, string, error) {
	sessionKey := msg.SessionKey()
	userInput := msg.Content

	// 创建工具钩子适配器，用于将 Claude Code SDK 工具调用桥接到现有 hook 系统
	var ccToolHookAdapter *toolHookAdapter
	if p.hookManager != nil {
		hookCtx := domain.NewHookContext(ctx)
		hookCtx.SetMetadata("session_key", sessionKey)
		hookCtx.SetMetadata("trace_id", traceID)

		userCode := ""
		agentCode := ""
		if agent != nil {
			agentCode = agent.AgentCode().String()
			userCode = agent.UserCode()
		}

		// 从 msg.Metadata 提取 channel_code
		channelCode := ""
		if msg.Metadata != nil {
			if v, ok := msg.Metadata["channel_code"].(string); ok {
				channelCode = v
			}
		}

		ccToolHookAdapter = &toolHookAdapter{
			hookManager: p.hookManager,
			logger:      p.logger,
			hookCtx:     hookCtx,
			sessionKey:  sessionKey,
			userCode:    userCode,
			agentCode:   agentCode,
			channelCode: channelCode,
			channelType: msg.Channel,
			traceID:     traceID,
		}
	}

	// 构建 claudecode 选项
	opts := p.buildOptions(provider, cliSessionID, agent, ccToolHookAdapter)

	p.logger.Info("开始 Claude Code 查询",
		zap.String("session_key", sessionKey),
		zap.String("cli_session_id", cliSessionID),
		zap.String("provider", func() string {
			if provider != nil {
				return provider.ProviderKey()
			}
			return "default"
		}()),
	)

	startTime := time.Now()

	// 使用 Client 接口进行流式处理
	client := claudecode.NewClient(opts...)

	// 使用 Connect 建立连接
	if err := client.Connect(ctx); err != nil {
		p.logger.Error("Claude Code Connect 失败",
			zap.String("session_key", sessionKey),
			zap.Duration("duration", time.Since(startTime)),
			zap.Error(err),
		)
		return "", "", fmt.Errorf("Claude Code Connect 失败: %w", err)
	}
	defer client.Disconnect()

	p.logger.Info("Claude Code 连接成功，开始查询",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
	)

	// 使用 QueryWithSession 发送查询
	sessionID := cliSessionID
	if sessionID == "" {
		sessionID = "default"
	}
	if err := client.QueryWithSession(ctx, userInput, sessionID); err != nil {
		p.logger.Error("Claude Code QueryWithSession 失败",
			zap.String("session_key", sessionKey),
			zap.Duration("duration", time.Since(startTime)),
			zap.Error(err),
		)
		return "", "", fmt.Errorf("Claude Code QueryWithSession 失败: %w", err)
	}

	// 使用 ReceiveMessages 接收流式消息
	var result string
	var cliSessionIDResult string

	msgChan := client.ReceiveMessages(ctx)
	resultChan := make(chan string, 1)
	sessionChan := make(chan string, 1)

	// 启动 goroutine 处理消息
	go func() {
		var result string
		var cliSessionIDResult string
		for msg := range msgChan {
			if msg == nil {
				continue
			}

			switch m := msg.(type) {
			case *claudecode.AssistantMessage:
				for _, block := range m.Content {
					switch b := block.(type) {
					case *claudecode.TextBlock:
						result += b.Text
					case *claudecode.ToolUseBlock:
						// 记录工具调用
						p.logger.Info("Claude Code 工具调用",
							zap.String("session_key", sessionKey),
							zap.String("tool_name", b.Name),
						)
						result += fmt.Sprintf("\n[调用工具: %s]\n", b.Name)
					case *claudecode.ToolResultBlock:
						// 记录工具结果
						p.logger.Info("Claude Code 工具结果",
							zap.String("session_key", sessionKey),
							zap.Any("content", b.Content),
						)
						result += fmt.Sprintf("%v\n", b.Content)
					case *claudecode.ThinkingBlock:
						// 思考过程
						result += fmt.Sprintf("\n[思考: %s]\n", b.Thinking)
					}
				}
			case *claudecode.ResultMessage:
				if m.SessionID != "" {
					cliSessionIDResult = m.SessionID
				}
				if m.IsError && m.Result != nil {
					result += fmt.Sprintf("\n[错误: %s]", *m.Result)
				}
				p.logger.Info("Claude Code ResultMessage",
					zap.String("session_key", sessionKey),
					zap.String("cli_session_id", cliSessionIDResult),
					zap.Bool("is_error", m.IsError),
				)
				// ResultMessage 表示会话结束，跳出循环
				resultChan <- result
				sessionChan <- cliSessionIDResult
				return
			case *claudecode.UserMessage:
				// 用户消息，不处理
			}
		}
		// 如果通道正常关闭但没有 ResultMessage，发送空结果
		resultChan <- result
		sessionChan <- cliSessionIDResult
	}()

	// 等待结果或超时
	select {
	case result = <-resultChan:
		cliSessionIDResult = <-sessionChan
	case <-ctx.Done():
		return "", "", fmt.Errorf("Claude Code 查询超时: %w", ctx.Err())
	}

	p.logger.Info("Claude Code 流式接收完成",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
	)

	return result, cliSessionIDResult, nil
}

// buildOptions 根据 Provider 类型构建 claudecode 选项
func (p *ClaudeCodeProcessor) buildOptions(provider *domain.LLMProvider, cliSessionID string, agent *domain.Agent, toolHookAdapter *toolHookAdapter) []claudecode.Option {
	opts := []claudecode.Option{}

	// 获取配置
	var config *domain.ClaudeCodeConfig
	if agent != nil {
		config = agent.ClaudeCodeConfig()
	}
	if config == nil {
		config = domain.DefaultClaudeCodeConfig()
	}

	// 注册工具钩子以记录对话
	if toolHookAdapter != nil {
		opts = append(opts, claudecode.WithPreToolUseHook("", toolHookAdapter.preToolUseAdapter))
		opts = append(opts, claudecode.WithPostToolUseHook("", toolHookAdapter.postToolUseAdapter))
	}

	// 设置 Env（API Key 和 Base URL）
	env := config.Env
	if env == nil {
		env = make(map[string]string)
	}

	// 设置模型
	model := config.Model
	if model == "" {
		if provider != nil {
			// 当模型为空时，从 provider 获取 API Key 和 Base URL
			if provider.APIKey() != "" {
				env["ANTHROPIC_API_KEY"] = provider.APIKey()
			}
			if provider.APIBase() != "" {
				env["ANTHROPIC_BASE_URL"] = provider.APIBase()
			}
			// 模型保持为空，让 Claude Code 使用默认模型
		} else {
			// 没有 provider 时，使用默认模型
			model = "MiniMax-M2.7-highspeed"
		}
	}

	opts = append(opts, claudecode.WithEnv(env))
	opts = append(opts, claudecode.WithModel(model))

	// 设置系统提示词
	if config.SystemPrompt != "" {
		opts = append(opts, claudecode.WithSystemPrompt(config.SystemPrompt))
	}

	// 设置最大思考 Token
	if config.MaxThinkingTokens > 0 {
		opts = append(opts, claudecode.WithMaxThinkingTokens(config.MaxThinkingTokens))
	}

	// 设置权限模式
	switch config.PermissionMode {
	case domain.PermissionModeAcceptEdits:
		opts = append(opts, claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits))
	case domain.PermissionModePlan:
		opts = append(opts, claudecode.WithPermissionMode(claudecode.PermissionModePlan))
	case domain.PermissionModeBypassPermissions:
		opts = append(opts, claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions))
	default:
		opts = append(opts, claudecode.WithPermissionMode(claudecode.PermissionModeDefault))
	}

	// 设置允许的工具
	if len(config.AllowedTools) > 0 {
		opts = append(opts, claudecode.WithAllowedTools(config.AllowedTools...))
	}

	// 设置禁止的工具
	if len(config.DisallowedTools) > 0 {
		opts = append(opts, claudecode.WithDisallowedTools(config.DisallowedTools...))
	}

	// 设置最大对话轮次
	if config.MaxTurns > 0 {
		opts = append(opts, claudecode.WithMaxTurns(config.MaxTurns))
	}

	// 设置工作目录
	if config.Cwd != "" {
		opts = append(opts, claudecode.WithCwd(config.Cwd))
	}

	// 设置会话恢复
	if config.Resume != nil && *config.Resume && cliSessionID != "" {
		opts = append(opts, claudecode.WithResume(cliSessionID))
	}

	// 设置继续会话
	if config.ContinueConversation != nil && *config.ContinueConversation {
		opts = append(opts, claudecode.WithContinueConversation(true))
	}

	// 设置文件检查点
	if config.FileCheckpointing != nil && *config.FileCheckpointing {
		opts = append(opts, claudecode.WithFileCheckpointing())
	}

	// 设置备用模型
	if config.FallbackModel != "" {
		opts = append(opts, claudecode.WithFallbackModel(config.FallbackModel))
	}

	// 追加系统提示词
	if config.AppendSystemPrompt != "" {
		opts = append(opts, claudecode.WithAppendSystemPrompt(config.AppendSystemPrompt))
	}

	// 设置沙箱
	if config.SandboxEnabled != nil && *config.SandboxEnabled {
		opts = append(opts, claudecode.WithSandboxEnabled(true))
		if config.AutoAllowBashIfSandboxed != nil && *config.AutoAllowBashIfSandboxed {
			opts = append(opts, claudecode.WithAutoAllowBashIfSandboxed(true))
		}
		if len(config.ExcludedCommands) > 0 {
			opts = append(opts, claudecode.WithSandboxExcludedCommands(config.ExcludedCommands...))
		}
	}

	// 设置 MCP 服务器
	if len(config.McpServers) > 0 {
		mcpServers := make(map[string]claudecode.McpServerConfig)
		for name, server := range config.McpServers {
			// 只支持 stdio 类型
			mcpServers[name] = &claudecode.McpStdioServerConfig{
				Command: server.Command,
				Args:    server.Args,
				Env:     server.Env,
			}
		}
		opts = append(opts, claudecode.WithMcpServers(mcpServers))
	}

	// 设置插件
	if len(config.Plugins) > 0 {
		// 需要将 string 转换为 SdkPluginConfig
		// 这里暂时跳过，实际使用时需要根据 SDK 定义转换
	}

	// 设置本地插件
	if config.LocalPlugin != "" {
		opts = append(opts, claudecode.WithLocalPlugin(config.LocalPlugin))
	}

	// 设置 JSON Schema
	if len(config.JSONSchema) > 0 {
		opts = append(opts, claudecode.WithJSONSchema(config.JSONSchema))
	}

	// 设置部分消息
	if config.IncludePartialMessages != nil && *config.IncludePartialMessages {
		opts = append(opts, claudecode.WithIncludePartialMessages(true))
	}

	// 设置最大预算
	if config.MaxBudgetUSD > 0 {
		opts = append(opts, claudecode.WithMaxBudgetUSD(config.MaxBudgetUSD))
	}

	// 设置 Beta 功能
	if len(config.Betas) > 0 {
		// 需要将 string 转换为 SdkBeta
		// 这里暂时跳过，实际使用时需要根据 SDK 定义转换
	}

	// 设置 CLI 路径
	if config.CLIPath != "" {
		opts = append(opts, claudecode.WithCLIPath(config.CLIPath))
	}

	// 设置额外参数
	if len(config.ExtraArgs) > 0 {
		args := make(map[string]*string)
		for k, v := range config.ExtraArgs {
			args[k] = &v
		}
		opts = append(opts, claudecode.WithExtraArgs(args))
	}

	// 设置来源
	if len(config.SettingSources) > 0 {
		// 需要将 string 转换为 SettingSource
		// 这里暂时跳过，实际使用时需要根据 SDK 定义转换
	}

	p.logger.Info("Claude Code 选项配置",
		zap.String("provider", func() string {
			if provider != nil {
				return provider.ProviderKey()
			}
			return "default"
		}()),
		zap.String("api_base_url", func() string {
			if provider != nil {
				return provider.APIBase()
			}
			return ""
		}()),
		zap.String("cli_session_id", cliSessionID),
		zap.String("model", model),
		zap.Int("options_count", len(opts)),
	)

	return opts
}

// toFloat64 converts an interface{} to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

// getUsageInt extracts an int value from a usage map
func getUsageInt(usage map[string]any, key string) int {
	if v, ok := usage[key]; ok {
		if f, ok := toFloat64(v); ok {
			return int(f)
		}
	}
	return 0
}

// resolveProvider 解析 Agent 使用的 LLM Provider
// 优先级：1. Agent 指定的 Provider > 2. 用户默认 Provider
func (p *ClaudeCodeProcessor) resolveProvider(ctx context.Context, agent *domain.Agent) (*domain.LLMProvider, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent is nil")
	}

	// 1. 优先使用 Agent 指定的 Provider
	llmProviderID := agent.LLMProviderID()
	if llmProviderID.String() != "" {
		provider, err := p.providerRepo.FindByID(ctx, llmProviderID)
		if err != nil {
			p.logger.Warn("获取 Agent 指定的 LLM Provider 失败，将使用用户默认 Provider",
				zap.String("agent_code", agent.AgentCode().String()),
				zap.String("llm_provider_id", llmProviderID.String()),
				zap.Error(err))
		} else if provider != nil {
			p.logger.Info("使用 Agent 指定的 LLM Provider",
				zap.String("agent_code", agent.AgentCode().String()),
				zap.String("provider_key", provider.ProviderKey()),
				zap.String("llm_provider_id", llmProviderID.String()))
			return provider, nil
		}
	}

	// 2. 使用用户默认 Provider
	userCode := agent.UserCode()
	provider, err := p.providerRepo.FindDefaultActive(ctx, userCode)
	if err != nil {
		return nil, fmt.Errorf("获取用户默认 Provider 失败: %w", err)
	}
	if provider != nil {
		p.logger.Info("使用用户默认 LLM Provider",
			zap.String("user_code", userCode),
			zap.String("provider_key", provider.ProviderKey()))
	}
	return provider, nil
}

// triggerClaudeCodeFinishedHook 触发 Claude Code 完成 hook
// success 参数表示 Claude Code 是否成功完成（不是错误退出）
// finalResult 参数是 Claude Code 的最终执行结果
func (p *ClaudeCodeProcessor) triggerClaudeCodeFinishedHook(ctx context.Context, msg *bus.InboundMessage, agent *domain.Agent, success bool, finalResult string) {
	// 检查是否有 requirement_id 元数据
	if msg.Metadata == nil {
		p.logger.Debug("Claude Code 完成，无 requirement_id 元数据")
		return
	}

	requirementIDStr, ok := msg.Metadata["requirement_id"].(string)
	if !ok || requirementIDStr == "" {
		p.logger.Debug("Claude Code 完成，requirement_id 为空或不存在",
			zap.Any("metadata", msg.Metadata))
		return
	}

	// 查找 requirement
	requirementID := domain.NewRequirementID(requirementIDStr)
	requirement, err := p.requirementRepo.FindByID(ctx, requirementID)
	if err != nil {
		p.logger.Error("Claude Code 完成，查找 requirement 失败",
			zap.String("requirement_id", requirementIDStr),
			zap.Error(err))
		return
	}
	if requirement == nil {
		p.logger.Warn("Claude Code 完成，requirement 不存在",
			zap.String("requirement_id", requirementIDStr))
		return
	}

	p.logger.Info("Claude Code 完成，触发 claude_code_finished hook",
		zap.String("requirement_id", requirementIDStr),
		zap.String("requirement_title", requirement.Title()),
		zap.Bool("success", success))

	// **立即清理分身**（代码约束，不是 Hook）
	// 在触发任何 hook 之前清理分身，确保清理一定会执行
	if p.replicaAgentManager != nil {
		p.replicaAgentManager.EnsureDisposed(ctx, requirement.ReplicaAgentCode(), requirement.WorkspacePath())
		requirement.SetReplicaAgentCode("")
		requirement.SetWorkspacePath("")
		if err := p.requirementRepo.Save(ctx, requirement); err != nil {
			p.logger.Error("Claude Code 完成，保存 requirement 失败",
				zap.String("requirement_id", requirementIDStr),
				zap.Error(err))
			return
		}
	}

	// 成功完成时，标记需求为 completed 状态并保存执行结果
	if success {
		requirement.MarkCompleted()
		requirement.SetClaudeRuntimeResult(finalResult)
		if err := p.requirementRepo.Save(ctx, requirement); err != nil {
			p.logger.Error("Claude Code 完成，保存 requirement 失败",
				zap.String("requirement_id", requirementIDStr),
				zap.Error(err))
			return
		}
		p.logger.Info("Claude Code 完成，requirement 已标记为 completed",
			zap.String("requirement_id", requirementIDStr),
			zap.Any("result_length", len(finalResult)))
	}

	// 按 trace_id 查询对话记录并计算 token
	traceID := requirement.TraceID()
	if traceID != "" && p.conversationRepo != nil {
		records, err := p.conversationRepo.FindByTraceID(ctx, traceID, defaultTokenAggregationLimit) // 最多查询1000条
		if err != nil {
			p.logger.Warn("查询对话记录失败", zap.String("trace_id", traceID), zap.Error(err))
		} else {
			var totalPrompt, totalCompletion, totalTokens int
			for _, record := range records {
				totalPrompt += record.PromptTokens()
				totalCompletion += record.CompletionTokens()
				totalTokens += record.TotalTokens()
			}
			requirement.SetTokenUsage(totalPrompt, totalCompletion, totalTokens)
			if err := p.requirementRepo.Save(ctx, requirement); err != nil {
				p.logger.Warn("保存 token 使用量到需求表失败", zap.Error(err))
			} else {
				p.logger.Info("已计算并保存 token 使用量",
					zap.String("requirement_id", requirementIDStr),
					zap.String("trace_id", traceID),
					zap.Int("prompt_tokens", totalPrompt),
					zap.Int("completion_tokens", totalCompletion),
					zap.Int("total_tokens", totalTokens),
					zap.Int("records_count", len(records)),
				)
			}
		}
	}
}
