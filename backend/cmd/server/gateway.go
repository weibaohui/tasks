package main

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/cleanup"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/utils"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
	"github.com/weibh/taskmanager/pkg/channel"
	"go.uber.org/zap"
)

// Gateway 渠道网关组件
type Gateway struct {
	logger         *zap.Logger
	messageBus     *channelBus.MessageBus
	sessionManager *channel.SessionManager
	processor      *channel.MessageProcessor
	channelManager *channel.Manager
}

func initGateway(
	channelService *application.ChannelApplicationService,
	sessionService *application.SessionApplicationService,
	agentRepo domain.AgentRepository,
	providerRepo domain.LLMProviderRepository,
	idGenerator *utils.NanoIDGenerator,
	hookManager *hook.Manager,
	logger *zap.Logger,
	mcpService *application.MCPApplicationService,
	skillsLoader domain.SkillsLoader,
	requirementRepo domain.RequirementRepository,
	conversationRecordRepo domain.ConversationRecordRepository,
	replicaCleanupSvc *cleanup.ReplicaCleanupService,
) *Gateway {
	gw := &Gateway{
		logger:         logger,
		messageBus:     channelBus.NewMessageBus(logger),
		sessionManager: channel.NewSessionManager(logger),
		channelManager: channel.NewManager(nil),
	}

	feishuThinkingHook := hooks.NewFeishuThinkingProcessHook(gw.messageBus, logger)
	hookManager.Register(feishuThinkingHook)
	logger.Info("已注册 FeishuThinkingProcessHook")

	gw.processor = channel.NewMessageProcessor(gw.messageBus, gw.sessionManager, logger, agentRepo, providerRepo, sessionService, idGenerator, hookManager, llm.NewLLMProviderFactory(), mcpService, skillsLoader, requirementRepo, conversationRecordRepo, replicaCleanupSvc)
	gw.channelManager = channel.NewManager(gw.messageBus)

	ctx, cancel := context.WithCancel(context.Background())
	gw.messageBus.StartDispatcher(ctx)
	_ = cancel

	if err := gw.channelManager.StartAll(ctx); err != nil {
		logger.Error("启动渠道失败", zap.Error(err))
	}

	go gw.runMessageLoop(ctx, channelService)
	logger.Info("渠道网关初始化完成", zap.Int("channels", gw.ChannelCount()))

	return gw
}

func (g *Gateway) loadChannels(channelService *application.ChannelApplicationService) {
	registry := channel.DefaultRegistry(g.messageBus, g.logger)

	ctx := context.Background()
	channels, err := channelService.ListActiveChannels(ctx)
	if err != nil {
		g.logger.Error("加载渠道配置失败", zap.Error(err))
		return
	}

	for _, ch := range channels {
		chType := string(ch.Type())
		chInstance, err := registry.CreateChannel(chType, ch.Config())
		if err != nil {
			g.logger.Warn("创建渠道实例失败",
				zap.String("name", ch.Name()),
				zap.String("type", chType),
				zap.Error(err),
			)
			continue
		}
		g.channelManager.Register(chInstance)
		g.logger.Info("已注册渠道",
			zap.String("name", ch.Name()),
			zap.String("type", chType),
		)
	}
}

func (g *Gateway) runMessageLoop(ctx context.Context, channelService *application.ChannelApplicationService) {
	g.logger.Info("消息处理循环已启动")
	for {
		msg, err := g.messageBus.ConsumeInbound(ctx)
		if err != nil {
			if ctx.Err() != nil {
				g.logger.Info("消息处理循环上下文已取消")
				return
			}
			g.logger.Error("消费消息失败", zap.Error(err))
			continue
		}

		if err := g.processor.Process(ctx, msg); err != nil {
			g.logger.Error("处理消息失败", zap.Error(err))
			metadata := make(map[string]any)
			for k, v := range msg.Metadata {
				metadata[k] = v
			}
			outMsg := &channelBus.OutboundMessage{
				Channel:  msg.Channel,
				ChatID:   msg.ChatID,
				Content:  fmt.Sprintf("处理消息时出错: %v", err),
				Metadata: metadata,
			}
			g.messageBus.PublishOutbound(outMsg)
		}
	}
}

func (g *Gateway) Shutdown() {
	g.logger.Info("正在关闭渠道网关...")
	g.channelManager.StopAll()
	g.messageBus.Stop()
	g.logger.Info("渠道网关已关闭")
}

func (g *Gateway) ChannelCount() int {
	return len(g.channelManager.List())
}
