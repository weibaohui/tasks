package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/llm/tools"
	"github.com/weibh/taskmanager/infrastructure/llm/tools/mcp"
	"github.com/weibh/taskmanager/infrastructure/skill"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// contextKey 用于在 HookContext 中存储和获取数据
type contextKey string

const spanKey contextKey = "conversation_span"

// MessageProcessor 处理来自渠道的消息
type MessageProcessor struct {
	bus              *bus.MessageBus
	logger           *zap.Logger
	sessionManager   *SessionManager
	agentConfigCache *AgentConfigCache
	agentRepo        domain.AgentRepository
	providerRepo     domain.LLMProviderRepository
	taskService      *application.TaskApplicationService
	workerPool       *application.WorkerPool
	idGenerator      domain.IDGenerator
	toolRegistry     *llm.ToolRegistry
	hookManager      *hook.Manager
	factory          domain.LLMProviderFactory
	mcpService       *application.MCPApplicationService
	skillsLoader     *skill.SkillsLoader
}

// NewMessageProcessor 创建消息处理器
func NewMessageProcessor(
	messageBus *bus.MessageBus,
	sessionManager *SessionManager,
	logger *zap.Logger,
	agentRepo domain.AgentRepository,
	providerRepo domain.LLMProviderRepository,
	taskService *application.TaskApplicationService,
	workerPool *application.WorkerPool,
	idGenerator domain.IDGenerator,
	hookManager *hook.Manager,
	factory domain.LLMProviderFactory,
	mcpService *application.MCPApplicationService,
	skillsLoader *skill.SkillsLoader,
) *MessageProcessor {
	registry := llm.NewToolRegistry()
	// 注意：Bash 和 MCP 工具不全局注册，而是在 buildAgentToolsRegistry 中按 Agent 配置按需注册

	return &MessageProcessor{
		bus:              messageBus,
		logger:           logger,
		sessionManager:   sessionManager,
		agentConfigCache: NewAgentConfigCache(),
		agentRepo:        agentRepo,
		providerRepo:     providerRepo,
		taskService:      taskService,
		workerPool:       workerPool,
		idGenerator:      idGenerator,
		toolRegistry:     registry,
		hookManager:      hookManager,
		factory:          factory,
		mcpService:       mcpService,
		skillsLoader:     skillsLoader,
	}
}

// Process 处理入站消息
func (p *MessageProcessor) Process(ctx context.Context, msg *bus.InboundMessage) error {
	// 开始新的 Trace，生成 trace_id 和 span_id
	ctx, traceID, spanID := trace.StartTrace(ctx)

	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	p.logger.Info("处理消息",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("渠道", msg.Channel),
		zap.String("发送者", msg.SenderID),
		zap.String("内容", preview),
	)

	// 获取或创建会话
	session := p.sessionManager.GetOrCreate(msg.SessionKey())

	// 为当前会话创建独立的 cancellable context，并注入 trace 信息
	sessionCtx, cancel := context.WithCancel(ctx)
	sessionCtx = trace.WithTraceID(sessionCtx, traceID)
	sessionCtx = trace.WithSpanID(sessionCtx, spanID)
	sessionCtx = trace.WithSessionInfo(sessionCtx, msg.SessionKey(), msg.Channel)
	session.SetContext(sessionCtx, cancel)

	// 处理完成后清理
	defer func() {
		session.SetContext(nil, nil)
	}()

	// 保存用户消息到会话历史
	session.AddMessage(Message{
		Role:    "user",
		Content: msg.Content,
		TraceID: traceID,
		SpanID:  spanID,
	})

	// 注意：不在这里自动创建任务
	// 任务应该在明确请求时才创建，例如通过 /task 命令触发

	// 生成响应
	response := p.generateResponse(sessionCtx, msg, session, traceID, spanID)

	// 发布响应消息
	outMsg := &bus.OutboundMessage{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		Content:  response,
		Metadata: make(map[string]any),
	}

	// 传递原始消息的 metadata 用于渠道特定功能
	if msg.Metadata != nil {
		if msgID, ok := msg.Metadata["message_id"].(string); ok {
			outMsg.Metadata["reply_to_message_id"] = msgID
		}
		if appID, ok := msg.Metadata["app_id"].(string); ok {
			outMsg.Metadata["app_id"] = appID
		}
		if senderID, ok := msg.Metadata["sender_id"].(string); ok {
			outMsg.Metadata["sender_id"] = senderID
		}
		if chatType, ok := msg.Metadata["chat_type"].(string); ok {
			outMsg.Metadata["chat_type"] = chatType
		}
	}

	// 传递 trace 信息
	outMsg.Metadata["trace_id"] = traceID
	outMsg.Metadata["span_id"] = spanID

	// 保存助手响应到会话历史
	session.AddMessage(Message{
		Role:    "assistant",
		Content: response,
		TraceID: traceID,
		SpanID:  spanID,
	})

	p.bus.PublishOutbound(outMsg)
	return nil
}

// generateResponse 生成响应
func (p *MessageProcessor) generateResponse(ctx context.Context, msg *bus.InboundMessage, session *Session, traceID, parentSpanID string) string {
	content := strings.TrimSpace(msg.Content)

	// 简单的命令处理
	if strings.HasPrefix(content, "/help") {
		return "可用命令:\n/help - 显示帮助信息\n/status - 显示状态"
	}

	if strings.HasPrefix(content, "/status") {
		return fmt.Sprintf("状态正常\n会话: %s\n渠道: %s", msg.SessionKey(), msg.Channel)
	}

	// 获取 Agent 和 LLM 配置
	agent, provider, err := p.getAgentAndProvider(msg)
	if err != nil {
		p.logger.Debug("获取 Agent/LLM 配置失败", zap.Error(err))
		return fmt.Sprintf("收到消息: %s\n(Agent 或 LLM 配置未找到)", content)
	}

	// 如果没有 Provider，返回默认响应
	if provider == nil {
		return fmt.Sprintf("收到消息: %s\n(LLM Provider 未配置)", content)
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
		model = "gpt-4"
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

	result, err := p.factory.Build(providerConfig)
	if err != nil {
		p.logger.Error("创建 LLM Provider 失败", zap.Error(err))
		return fmt.Sprintf("收到消息: %s\n(LLM 配置错误)", content)
	}

	llmProvider, ok := result.(llm.LLMProvider)
	if !ok {
		p.logger.Error("LLM Provider 类型转换失败")
		return fmt.Sprintf("收到消息: %s\n(LLM 配置错误)", content)
	}

	// 构建对话历史 prompt
	prompt := p.buildPrompt(session, content, agent)

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
	var toolCalls []llm.ToolCall

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
	toolRegistries := []*llm.ToolRegistry{p.toolRegistry}
	if agent != nil {
		if agentToolsRegistry := p.buildAgentToolsRegistry(agent); agentToolsRegistry != nil {
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
				TotalTokens:     usage.TotalTokens,
			},
		}
		// 构造 RawResponse 包含工具调用信息，供 hook 分析
		if len(toolCalls) > 0 {
			toolCallsInfo := make([]map[string]interface{}, 0, len(toolCalls))
			for _, tc := range toolCalls {
				argsStr := string(tc.Input)
				toolCallsInfo = append(toolCallsInfo, map[string]interface{}{
					"id":   tc.ID,
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

// getAgentAndProvider 根据消息获取 Agent 和 LLMProvider
func (p *MessageProcessor) getAgentAndProvider(msg *bus.InboundMessage) (*domain.Agent, *domain.LLMProvider, error) {
	if msg.Metadata == nil {
		return nil, nil, fmt.Errorf("消息元数据为空")
	}

	// 获取 agent_code
	agentCode, ok := msg.Metadata["agent_code"].(string)
	if !ok || agentCode == "" {
		// 尝试从 channel_code 获取 channel 再获取 agent
		p.logger.Debug("消息中未包含 agent_code")
		return nil, nil, fmt.Errorf("消息中未包含 agent_code")
	}

	// 获取 Agent
	agent, err := p.agentRepo.FindByAgentCode(context.Background(), domain.NewAgentCode(agentCode))
	if err != nil || agent == nil {
		p.logger.Debug("获取 Agent 失败", zap.String("agent_code", agentCode), zap.Error(err))
		return nil, nil, err
	}

	// 获取用户的默认 LLM Provider
	userCode := agent.UserCode()
	provider, err := p.providerRepo.FindDefaultActive(context.Background(), userCode)
	if err != nil || provider == nil {
		p.logger.Debug("获取 LLM Provider 失败", zap.String("user_code", userCode), zap.Error(err))
		return agent, nil, err
	}

	return agent, provider, nil
}

// buildPrompt 构建 LLM prompt
func (p *MessageProcessor) buildPrompt(session *Session, userInput string, agent *domain.Agent) string {
	var sb strings.Builder

	// 添加系统提示
	sb.WriteString("你是一个智能助手，请根据对话历史回答用户的问题。\n\n")

	// 添加 MCP Server 列表（如果有绑定）
	if agent != nil && p.mcpService != nil {
		mcpInfo := p.getAgentMCPServers(agent)
		if mcpInfo != "" {
			sb.WriteString(mcpInfo)
			sb.WriteString("\n")
		}
	}

	// 添加对话历史
	messages := session.Messages()
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("用户: %s\n", msg.Content))
		case "assistant":
			sb.WriteString(fmt.Sprintf("助手: %s\n", msg.Content))
		case "system":
			sb.WriteString(fmt.Sprintf("系统: %s\n", msg.Content))
		}
	}

	// 添加当前用户输入
	sb.WriteString(fmt.Sprintf("用户: %s\n助手:", userInput))

	return sb.String()
}

// getAgentMCPServers 获取 Agent 绑定的 MCP Server 列表，生成提示词
func (p *MessageProcessor) getAgentMCPServers(agent *domain.Agent) string {
	if agent == nil {
		return ""
	}

	ctx := context.Background()
	bindings, err := p.mcpService.ListAgentBindings(ctx, agent.ID())
	if err != nil || len(bindings) == 0 {
		return ""
	}

	var servers []string
	for _, binding := range bindings {
		if !binding.IsActive() {
			continue
		}
		server, err := p.mcpService.GetServer(ctx, binding.MCPServerID())
		if err != nil || server == nil {
			continue
		}
		if server.Status() != "active" {
			continue
		}
		desc := server.Description()
		if desc == "" {
			desc = "无描述"
		}
		servers = append(servers, fmt.Sprintf("- **%s** (%s): %s", server.Code(), server.Name(), desc))
	}

	if len(servers) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## 可用的 MCP Servers\n")
	sb.WriteString("你可以使用 `use_mcp` 工具加载以下 MCP Server 的工具:\n\n")
	for _, s := range servers {
		sb.WriteString(s)
		sb.WriteString("\n")
	}
	sb.WriteString("\n使用示例: use_mcp(server_code=\"服务器编码\", action=\"load\")")

	return sb.String()
}

// buildAgentToolsRegistry 为 Agent 构建工具注册表
// 包括 Bash（按 agent.ToolsList 配置）、MCP（按 agent 绑定）、Skills（按 agent.SkillsList 配置）
// 如果各项配置都为空，则不注册任何工具
func (p *MessageProcessor) buildAgentToolsRegistry(agent *domain.Agent) *llm.ToolRegistry {
	if agent == nil {
		return nil
	}

	registry := llm.NewToolRegistry()
	registered := false

	// 1. 注册 Bash 工具（如果 agent.ToolsList 包含 "bash"）
	agentTools := agent.ToolsList()
	for _, t := range agentTools {
		if t == "bash" {
			registry.Register(tools.NewBashTool())
			registered = true
			break
		}
	}

	// 2. 注册 MCP 工具（如果 agent 有 MCP 绑定）
	if p.mcpService != nil {
		bindings, err := p.mcpService.ListAgentBindings(context.Background(), agent.ID())
		if err == nil && len(bindings) > 0 {
			// 检查是否有任何启用的绑定
			hasActiveBinding := false
			for _, b := range bindings {
				if b.IsActive() {
					hasActiveBinding = true
					break
				}
			}
			if hasActiveBinding {
				registry.Register(mcp.NewUseMCPTool(p.mcpService))
				registry.Register(mcp.NewCallMCPTool(p.mcpService))
				registered = true
			}
		}
	}

	// 3. 注册 Skills 工具（如果 agent.SkillsList 非空）
	if p.skillsLoader != nil {
		skills := p.skillsLoader.ListSkills()
		agentSkills := agent.SkillsList()
		if len(agentSkills) > 0 && len(skills) > 0 {
			enabledSkills := make(map[string]bool)
			for _, s := range agentSkills {
				enabledSkills[s] = true
			}

			skillToolsRegistry := tools.NewSkillToolsAdapterRegistry(p.skillsLoader)
			// 使用 GetToolsForSkills 避免重复发现技能（复用已获取的 skills 列表）
			for _, t := range skillToolsRegistry.GetToolsForSkills(skills) {
				toolName := t.Name()
				if strings.HasPrefix(toolName, "skill__") {
					skillName := strings.TrimPrefix(toolName, "skill__")
					if enabledSkills[skillName] {
						registry.Register(t)
						registered = true
					}
				}
			}
		}
	}

	// 如果没有注册任何工具，返回 nil
	if !registered {
		return nil
	}

	return registry
}

// AgentConfigCache 缓存 Agent 配置
type AgentConfigCache struct {
	cache map[string]*AgentConfig
}

func NewAgentConfigCache() *AgentConfigCache {
	return &AgentConfigCache{
		cache: make(map[string]*AgentConfig),
	}
}

// AgentConfig Agent 配置
type AgentConfig struct {
	AgentCode    string
	Name         string
	Instructions string
	Tools        []string
	MCPs         []string
}

// Get 获取配置
func (c *AgentConfigCache) Get(key string) (*AgentConfig, bool) {
	cfg, ok := c.cache[key]
	return cfg, ok
}

// Set 设置配置
func (c *AgentConfigCache) Set(key string, cfg *AgentConfig) {
	c.cache[key] = cfg
}

// Clear 清除缓存
func (c *AgentConfigCache) Clear(key string) {
	delete(c.cache, key)
}

// toolHookAdapter 将 domain.ToolHook 适配为 llm.ToolHook
type toolHookAdapter struct {
	processor    *MessageProcessor
	hookCtx      *domain.HookContext
	sessionID    string
	traceID      string
	spanID       string
	parentSpanID string
	// scope 信息
	sessionKey   string
	userCode     string
	agentCode    string
	channelCode  string
	channelType  string
}

func (p *MessageProcessor) newToolHookAdapter(hookCtx *domain.HookContext, sessionID, traceID, parentSpanID, sessionKey, userCode, agentCode, channelCode, channelType string) *toolHookAdapter {
	return &toolHookAdapter{
		processor:    p,
		hookCtx:      hookCtx,
		sessionID:    sessionID,
		traceID:      traceID,
		spanID:       p.idGenerator.Generate(),
		parentSpanID: parentSpanID,
		sessionKey:   sessionKey,
		userCode:     userCode,
		agentCode:    agentCode,
		channelCode:  channelCode,
		channelType:  channelType,
	}
}

func (a *toolHookAdapter) PreToolCall(toolName string, input json.RawMessage) (json.RawMessage, error) {
	// 构建 ToolCallContext
	var args map[string]interface{}
	if err := json.Unmarshal(input, &args); err != nil {
		args = map[string]interface{}{"raw": string(input)}
	}

	callCtx := &domain.ToolCallContext{
		ToolName:     toolName,
		ToolInput:    args,
		SessionID:    a.sessionID,
		TraceID:      a.traceID,
		SpanID:       a.spanID,
		ParentSpanID: a.parentSpanID,
	}

	// 将 tool_call 的 span_id 和 scope 设置到 ctx 中，供 PostToolCall 使用
	ctxWithSpan := a.hookCtx.WithValue(spanKey, a.spanID)
	ctxWithSpan = ctxWithSpan.WithValue(hooks.ScopeKey, hooks.ScopeInfo{
		SessionKey:  a.sessionKey,
		UserCode:    a.userCode,
		AgentCode:   a.agentCode,
		ChannelCode: a.channelCode,
		ChannelType: a.channelType,
	})

	// 调用 PreToolCall hooks
	if a.processor.hookManager != nil {
		modifiedCtx, err := a.processor.hookManager.PreToolCall(ctxWithSpan, callCtx)
		if err != nil {
			a.processor.logger.Error("PreToolCall hook failed", zap.Error(err))
		} else if modifiedCtx != nil {
			// 如果 hook 修改了输入，返回修改后的输入
			if modifiedCtx.ToolInput != nil {
				newInput, err := json.Marshal(modifiedCtx.ToolInput)
				if err == nil {
					return newInput, nil
				}
			}
		}
	}

	return input, nil
}

func (a *toolHookAdapter) PostToolCall(toolName string, input json.RawMessage, output string, toolErr error) {
	// 构建 ToolCallContext
	var args map[string]interface{}
	if err := json.Unmarshal(input, &args); err != nil {
		args = map[string]interface{}{"raw": string(input)}
	}

	callCtx := &domain.ToolCallContext{
		ToolName:     toolName,
		ToolInput:    args,
		SessionID:    a.sessionID,
		TraceID:      a.traceID,
		SpanID:       a.spanID,
		ParentSpanID: a.parentSpanID,
	}

	// 构建 ToolExecutionResult
	var resultOutput interface{} = output
	if toolErr != nil {
		resultOutput = fmt.Sprintf("error: %v", toolErr)
	}
	result := &domain.ToolExecutionResult{
		Success: toolErr == nil,
		Output:  resultOutput,
		Error:   toolErr,
		SpanID:  a.spanID,
	}

	// 调用 PostToolCall hooks - 使用带有 scope 信息的 ctx
	if a.processor.hookManager != nil {
		ctxWithScope := a.hookCtx.WithValue(hooks.ScopeKey, hooks.ScopeInfo{
			SessionKey:  a.sessionKey,
			UserCode:    a.userCode,
			AgentCode:   a.agentCode,
			ChannelCode: a.channelCode,
			ChannelType: a.channelType,
		})
		_, err := a.processor.hookManager.PostToolCall(ctxWithScope, callCtx, result)
		if err != nil {
			a.processor.logger.Error("PostToolCall hook failed", zap.Error(err))
		}
	}
}

// OnLLMCalledWithTools 实现 llm.ToolExecutionObserver
func (a *toolHookAdapter) OnLLMCalledWithTools(ctx context.Context, callCtx llm.LLMCallContext) {
	if a.processor.hookManager == nil {
		return
	}

	// 转换为 domain.LLMCallContext
	domainCallCtx := &domain.LLMCallContext{
		TraceID:   a.traceID,
		SessionID: a.sessionID,
		Metadata: map[string]string{
			"session_key":  a.sessionKey,
			"user_code":    a.userCode,
			"agent_code":   a.agentCode,
			"channel_code": a.channelCode,
			"channel_type": a.channelType,
		},
	}

	domainResp := &domain.LLMResponse{
		Content: callCtx.Content,
		Usage: domain.Usage{
			PromptTokens:     callCtx.Usage.PromptTokens,
			CompletionTokens: callCtx.Usage.CompletionTokens,
			TotalTokens:      callCtx.Usage.TotalTokens,
		},
	}

	domainHookCtx := domain.NewHookContext(ctx)
	domainHookCtx.SetMetadata("trace_id", a.traceID)
	domainHookCtx.SetMetadata("session_key", a.sessionKey)

	a.processor.hookManager.OnLLMCalledWithTools(domainHookCtx, domainCallCtx, domainResp)
}

// OnToolExecutionComplete 实现 llm.ToolExecutionObserver
func (a *toolHookAdapter) OnToolExecutionComplete(ctx context.Context, tools []llm.ToolCallContext) {
	if a.processor.hookManager == nil {
		return
	}

	domainHookCtx := domain.NewHookContext(ctx)
	domainHookCtx.SetMetadata("trace_id", a.traceID)
	domainHookCtx.SetMetadata("session_key", a.sessionKey)

	a.processor.hookManager.OnToolExecutionComplete(domainHookCtx)
}

// createTaskFromMessage 从消息创建任务
func (p *MessageProcessor) createTaskFromMessage(ctx context.Context, msg *bus.InboundMessage, traceID, spanID string, session *Session) {
	// 构建任务元数据，包含会话和渠道信息
	metadata := make(map[string]interface{})
	metadata["session_key"] = msg.SessionKey()
	metadata["channel"] = msg.Channel
	metadata["sender_id"] = msg.SenderID
	metadata["content"] = msg.Content

	// 从消息 metadata 中提取 agent_code 和其他信息
	if msg.Metadata != nil {
		if agentCode, ok := msg.Metadata["agent_code"].(string); ok {
			metadata["agent_code"] = agentCode
		}
		if channelCode, ok := msg.Metadata["channel_code"].(string); ok {
			metadata["channel_code"] = channelCode
		}
		if userCode, ok := msg.Metadata["user_code"].(string); ok {
			metadata["user_code"] = userCode
		}
	}

	// 使用消息的 trace_id 和 span_id
	taskTraceID := domain.NewTraceID(traceID)
	taskSpanID := domain.NewSpanID(spanID)

	// 创建任务命令
	cmd := application.CreateTaskCommand{
		Name:        fmt.Sprintf("会话任务: %s", msg.SessionKey()),
		Description: msg.Content,
		Type:        domain.TaskTypeAgent,
		Metadata:    metadata,
		Timeout:     60000, // 60秒超时
		MaxRetries:  0,
		Priority:    0,
		TraceID:     &taskTraceID,
		SpanID:      &taskSpanID,
	}

	// 创建任务
	task, err := p.taskService.CreateTask(ctx, cmd)
	if err != nil {
		p.logger.Error("创建任务失败", zap.Error(err), zap.String("trace_id", traceID))
		return
	}

	// 启动任务并提交到工作池
	if err := p.taskService.StartTask(ctx, task.ID()); err != nil {
		p.logger.Error("启动任务失败", zap.Error(err), zap.String("task_id", task.ID().String()))
		return
	}

	p.logger.Info("任务已创建并提交",
		zap.String("task_id", task.ID().String()),
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("task_span_id", task.SpanID().String()),
	)
}
