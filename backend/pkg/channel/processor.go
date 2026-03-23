package channel

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// MessageProcessor 处理来自渠道的消息
type MessageProcessor struct {
	bus               *bus.MessageBus
	logger            *zap.Logger
	sessionManager    *SessionManager
	agentConfigCache *AgentConfigCache
}

// NewMessageProcessor 创建消息处理器
func NewMessageProcessor(
	messageBus *bus.MessageBus,
	sessionManager *SessionManager,
	logger *zap.Logger,
) *MessageProcessor {
	return &MessageProcessor{
		bus:               messageBus,
		logger:            logger,
		sessionManager:    sessionManager,
		agentConfigCache: NewAgentConfigCache(),
	}
}

// Process 处理入站消息
func (p *MessageProcessor) Process(ctx context.Context, msg *bus.InboundMessage) error {
	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	p.logger.Info("处理消息",
		zap.String("渠道", msg.Channel),
		zap.String("发送者", msg.SenderID),
		zap.String("内容", preview),
	)

	// 获取或创建会话
	session := p.sessionManager.GetOrCreate(msg.SessionKey())

	// 为当前会话创建独立的 cancellable context
	sessionCtx, cancel := context.WithCancel(ctx)
	session.SetContext(sessionCtx, cancel)

	// 处理完成后清理
	defer func() {
		session.SetContext(nil, nil)
	}()

	// 保存用户消息到会话历史
	session.AddMessage(Message{
		Role:    "user",
		Content: msg.Content,
	})

	// TODO: 调用 LLM 处理消息
	// 目前返回简单的响应
	response := p.generateResponse(msg)

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

	// 保存助手响应到会话历史
	session.AddMessage(Message{
		Role:    "assistant",
		Content: response,
	})

	p.bus.PublishOutbound(outMsg)
	return nil
}

// generateResponse 生成响应（临时实现，后续接入 LLM）
func (p *MessageProcessor) generateResponse(msg *bus.InboundMessage) string {
	content := strings.TrimSpace(msg.Content)

	// 简单的命令处理
	if strings.HasPrefix(content, "/help") {
		return "可用命令:\n/help - 显示帮助信息\n/status - 显示状态"
	}

	if strings.HasPrefix(content, "/status") {
		return fmt.Sprintf("状态正常\n会话: %s\n渠道: %s", msg.SessionKey(), msg.Channel)
	}

	// 默认响应
	return fmt.Sprintf("收到消息: %s\n(LLM 处理功能待实现)", content)
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
