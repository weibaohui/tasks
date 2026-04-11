package channel

import (
	"context"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/claudecode"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/skill"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// MessageProcessor 处理来自渠道的消息
type MessageProcessor struct {
	bus                  *bus.MessageBus
	logger               *zap.Logger
	sessionManager       *SessionManager
	agentConfigCache     *AgentConfigCache
	agentRepo            domain.AgentRepository
	providerRepo         domain.LLMProviderRepository
	sessionService       *application.SessionApplicationService
	idGenerator          domain.IDGenerator
	toolRegistry         *llm.ToolRegistry
	hookManager          *hook.Manager
	factory              domain.LLMProviderFactory
	mcpService           *application.MCPApplicationService
	skillsLoader         *skill.SkillsLoader
	requirementRepo      domain.RequirementRepository
	conversationRepo     domain.ConversationRecordRepository
	replicaCleanupSvc   domain.ReplicaCleanupService
	claudeCodeProcessor  claudecode.ClaudeCodeProcessorInterface
	commandProcessor     *CommandProcessor
}

// NewMessageProcessor 创建消息处理器
func NewMessageProcessor(
	messageBus *bus.MessageBus,
	sessionManager *SessionManager,
	logger *zap.Logger,
	agentRepo domain.AgentRepository,
	providerRepo domain.LLMProviderRepository,
	sessionService *application.SessionApplicationService,
	idGenerator domain.IDGenerator,
	hookManager *hook.Manager,
	factory domain.LLMProviderFactory,
	mcpService *application.MCPApplicationService,
	skillsLoader *skill.SkillsLoader,
	requirementRepo domain.RequirementRepository,
	conversationRepo domain.ConversationRecordRepository,
	replicaCleanupSvc domain.ReplicaCleanupService,
) *MessageProcessor {
	registry := llm.NewToolRegistry()
	// 注意：Bash 和 MCP 工具不全局注册，而是在 buildAgentToolsRegistry 中按 Agent 配置按需注册

	// 创建命令处理器并设置 sessionManager 引用
	commandProcessor := NewCommandProcessor(logger)
	SetSessionManager(sessionManager)

	return &MessageProcessor{
		bus:                  messageBus,
		logger:               logger,
		sessionManager:       sessionManager,
		agentConfigCache:     NewAgentConfigCache(),
		agentRepo:            agentRepo,
		providerRepo:         providerRepo,
		sessionService:       sessionService,
		idGenerator:          idGenerator,
		toolRegistry:         registry,
		hookManager:          hookManager,
		factory:              factory,
		mcpService:           mcpService,
		skillsLoader:         skillsLoader,
		requirementRepo:      requirementRepo,
		conversationRepo:     conversationRepo,
		replicaCleanupSvc:   replicaCleanupSvc,
		claudeCodeProcessor: claudecode.NewClaudeCodeProcessor(logger, hookManager, providerRepo, idGenerator, requirementRepo, replicaCleanupSvc, conversationRepo),
		commandProcessor:     commandProcessor,
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
