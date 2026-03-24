package channel

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// MessageProcessor 处理来自渠道的消息
type MessageProcessor struct {
	bus               *bus.MessageBus
	logger            *zap.Logger
	sessionManager    *SessionManager
	agentConfigCache *AgentConfigCache
	agentRepo        domain.AgentRepository
	providerRepo     domain.LLMProviderRepository
	taskService      *application.TaskApplicationService
	workerPool       *application.WorkerPool
	idGenerator      domain.IDGenerator
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
) *MessageProcessor {
	return &MessageProcessor{
		bus:               messageBus,
		logger:            logger,
		sessionManager:    sessionManager,
		agentConfigCache: NewAgentConfigCache(),
		agentRepo:        agentRepo,
		providerRepo:     providerRepo,
		taskService:      taskService,
		workerPool:       workerPool,
		idGenerator:      idGenerator,
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

	// 如果配置了任务服务和工作者池，创建任务
	if p.taskService != nil && p.workerPool != nil && p.idGenerator != nil {
		p.createTaskFromMessage(ctx, msg, traceID, spanID, session)
	}

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

	llmConfig := &llm.Config{
		ProviderType: provider.ProviderKey(),
		Model:        model,
		APIKey:       provider.APIKey(),
		BaseURL:      provider.APIBase(),
		Temperature:  0.7,
		MaxTokens:    4096,
	}

	// 创建 LLM Provider
	llmProvider, err := llm.NewLLMProvider(llmConfig)
	if err != nil {
		p.logger.Error("创建 LLM Provider 失败", zap.Error(err))
		return fmt.Sprintf("收到消息: %s\n(LLM 配置错误)", content)
	}

	// 构建对话历史 prompt
	prompt := p.buildPrompt(session, content)

	// 开始 LLM 调用 span
	ctx, llmSpanID := trace.StartSpan(ctx)
	p.logger.Debug("LLM 调用",
		zap.String("trace_id", traceID),
		zap.String("parent_span_id", parentSpanID),
		zap.String("span_id", llmSpanID),
	)

	// 调用 LLM
	response, err := llmProvider.Generate(ctx, prompt)
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
	)

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
func (p *MessageProcessor) buildPrompt(session *Session, userInput string) string {
	var sb strings.Builder

	// 添加系统提示
	sb.WriteString("你是一个智能助手，请根据对话历史回答用户的问题。\n\n")

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
	AgentCode   string
	Name        string
	Instructions string
	Tools       []string
	MCPs        []string
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
