/**
 * 渠道网关服务入口
 * 支持飞书、钉钉、微信等多渠道消息接收和发送
 */
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"github.com/weibh/taskmanager/infrastructure/llm"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/skill"
	"github.com/weibh/taskmanager/infrastructure/utils"
	httpHandler "github.com/weibh/taskmanager/interfaces/http"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
	"github.com/weibh/taskmanager/pkg/channel"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// resolveGatewayWorkspace 解析工作区目录路径
func resolveGatewayWorkspace() string {
	if p := os.Getenv("TASKMANAGER_WORKSPACE"); p != "" {
		return p
	}
	// 如果当前目录存在 backend 目录，使用 backend（适配从仓库根目录执行）
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend")
	}
	// 否则使用当前工作目录
	return "."
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("启动渠道网关服务...")

	// 1. 初始化数据库
	dbPath := resolveDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal("Failed to open database", zap.String("path", dbPath), zap.Error(err))
	}
	defer db.Close()

	// 2. 初始化数据库 Schema
	if err := _persistence.InitSchema(db); err != nil {
		logger.Fatal("Failed to init schema", zap.Error(err))
	}
	logger.Info("数据库初始化完成")

	// 3. 初始化依赖
	idGenerator := utils.NewNanoIDGenerator(21)
	channelRepo := _persistence.NewSQLiteChannelRepository(db)
	sessionRepo := _persistence.NewSQLiteSessionRepository(db)
	agentRepo := _persistence.NewSQLiteAgentRepository(db)
	providerRepo := _persistence.NewSQLiteLLMProviderRepository(db)
	userTokenRepo := _persistence.NewSQLiteUserTokenRepository(db)
	requirementRepo := _persistence.NewSQLiteRequirementRepository(db)
	conversationRecordRepo := _persistence.NewSQLiteConversationRecordRepository(db)

	// 4. 初始化 Message Bus
	messageBus := channelBus.NewMessageBus(logger)
	logger.Info("Message Bus 初始化完成")

	// 5. 初始化 Session Manager
	sessionManager := channel.NewSessionManager(logger)
	logger.Info("Session Manager 初始化完成")

	// 6. 初始化应用服务
	sessionService := application.NewSessionApplicationService(sessionRepo, idGenerator)
	logger.Info("Session 服务初始化完成")

	// 6.5 初始化 ReplicaCleanupService
	replicaCleanupSvc := application.NewReplicaCleanupService(agentRepo)
	logger.Info("ReplicaCleanupService 初始化完成")

	// 7. 初始化 Hook Manager
	hookManager := hook.NewManager(logger, nil)
	hookManager.Register(hooks.NewLoggingHook(logger))
	hookManager.Register(hooks.NewMetricsHook(logger))
	hookManager.Register(hooks.NewRateLimitHook(rate.Limit(60), 100, logger))
	hookManager.Register(hooks.NewFeishuThinkingProcessHook(messageBus, logger))
	logger.Info("Hook Manager 初始化完成", zap.Int("hooks", len(hookManager.List())))

	// 8. 初始化技能加载器
	gatewayWorkspace := resolveGatewayWorkspace()
	gatewaySkillsLoader := skill.NewSkillsLoader(gatewayWorkspace)
	logger.Info("技能加载器初始化完成", zap.String("workspace", gatewayWorkspace))

	// 9. 初始化消息处理器 (gateway 不创建 workerPool，任务由 server 执行)
	processor := channel.NewMessageProcessor(messageBus, sessionManager, logger, agentRepo, providerRepo, nil, sessionService, nil, idGenerator, hookManager, llm.NewLLMProviderFactory(), nil, gatewaySkillsLoader, requirementRepo, conversationRecordRepo, replicaCleanupSvc)
	logger.Info("消息处理器初始化完成")

	// 10. 初始化应用服务
	agentService := application.NewAgentApplicationService(agentRepo, idGenerator)
	channelService := application.NewChannelApplicationService(channelRepo, idGenerator)
	logger.Info("应用服务初始化完成")

	// 11. 初始化渠道管理器
	channelRegistry := channel.DefaultRegistry(messageBus, logger)
	channelManager := channel.NewManager(messageBus)

	// 从数据库加载渠道配置
	if err := registerChannelsFromDB(channelService, channelRegistry, channelManager, logger); err != nil {
		logger.Error("加载渠道配置失败", zap.Error(err))
	}

	// 12. 启动消息分发器
	ctx, cancel := context.WithCancel(context.Background())
	messageBus.StartDispatcher(ctx)

	// 13. 启动所有渠道
	if err := channelManager.StartAll(ctx); err != nil {
		logger.Error("启动渠道失败", zap.Error(err))
	}

	// 14. 初始化 HTTP Handler (仅管理API)
	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		authSecret = "taskmanager-dev-secret"
	}

	// 创建 Gin Engine 用于管理 API
	userService := application.NewUserApplicationService(nil, idGenerator)
	authHandler := httpHandler.NewAuthHandler(userService, userTokenRepo, idGenerator, authSecret)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.POST("/api/auth/login", authHandler.Login)

	// 15. 启动 HTTP Server
	server := &http.Server{
		Addr:         ":8889",
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("HTTP Server 启动在 :8889")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP Server 启动失败", zap.Error(err))
		}
	}()

	// 16. 启动消息处理循环
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runMessageLoop(ctx, messageBus, processor, sessionService, agentService, logger)
	}()

	logger.Info("渠道网关服务已启动",
		zap.Int("channels", len(channelManager.List())),
	)

	// 17. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("正在关闭...")
	cancel()

	// 等待关闭
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("消息处理循环已正常停止")
	case <-time.After(5 * time.Second):
		logger.Warn("消息处理循环停止超时")
	}

	// 停止所有渠道
	channelManager.StopAll()

	// 停止 HTTP Server
	if err := server.Shutdown(context.Background()); err != nil {
		logger.Error("停止 HTTP Server 失败", zap.Error(err))
	}

	logger.Info("渠道网关服务已关闭")
}

// registerChannelsFromDB 从数据库注册渠道
func registerChannelsFromDB(
	channelService *application.ChannelApplicationService,
	registry *channel.Registry,
	manager *channel.Manager,
	logger *zap.Logger,
) error {
	ctx := context.Background()

	// 获取所有启用的渠道
	channels, err := channelService.ListActiveChannels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active channels: %w", err)
	}

	for _, ch := range channels {
		chType := string(ch.Type())

		chInstance, err := registry.CreateChannel(chType, ch.Config())
		if err != nil {
			logger.Warn("创建渠道实例失败",
				zap.String("name", ch.Name()),
				zap.String("type", chType),
				zap.Error(err),
			)
			continue
		}

		manager.Register(chInstance)
		logger.Info("已注册渠道",
			zap.String("name", ch.Name()),
			zap.String("type", chType),
		)
	}

	return nil
}

// runMessageLoop 运行消息处理循环
func runMessageLoop(
	ctx context.Context,
	messageBus *channelBus.MessageBus,
	processor *channel.MessageProcessor,
	sessionService *application.SessionApplicationService,
	agentService *application.AgentApplicationService,
	logger *zap.Logger,
) {
	logger.Info("消息处理循环已启动")
	for {
		msg, err := messageBus.ConsumeInbound(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("消息处理循环上下文已取消")
				return
			}
			logger.Error("消费消息失败", zap.Error(err))
			continue
		}

		if err := processor.Process(ctx, msg); err != nil {
			logger.Error("处理消息失败", zap.Error(err))
			// 发送错误响应
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
			messageBus.PublishOutbound(outMsg)
		}
	}
}

// resolveDBPath 解析数据库文件路径，支持通过环境变量配置，默认在后端目录下
func resolveDBPath() string {
	if p := os.Getenv("TASKMANAGER_DB_PATH"); p != "" {
		return p
	}
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	// 如果当前目录存在 backend 目录，优先写入 backend/tasks.db（适配从仓库根目录执行）
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend/tasks.db")
	}
	// 否则使用当前工作目录
	return filepath.FromSlash("./tasks.db")
}
