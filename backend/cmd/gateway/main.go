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
	"sync"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
	httpHandler "github.com/weibh/taskmanager/interfaces/http"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
	"github.com/weibh/taskmanager/pkg/channel"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("启动渠道网关服务...")

	// 1. 初始化数据库
	db, err := sql.Open("sqlite3", "./tasks.db")
	if err != nil {
		logger.Fatal("Failed to open database", zap.Error(err))
	}
	defer db.Close()

	// 2. 初始化数据库 Schema
	if err := _persistence.InitSchema(db); err != nil {
		logger.Fatal("Failed to init schema", zap.Error(err))
	}
	logger.Info("数据库初始化完成")

	// 3. 初始化依赖
	idGenerator := utils.NewNanoIDGenerator(21)
	eventBus := bus.NewEventBus()
	channelRepo := _persistence.NewSQLiteChannelRepository(db)
	sessionRepo := _persistence.NewSQLiteSessionRepository(db)
	agentRepo := _persistence.NewSQLiteAgentRepository(db)

	// 4. 初始化 Message Bus
	messageBus := channelBus.NewMessageBus(logger)
	logger.Info("Message Bus 初始化完成")

	// 5. 初始化 Hook Manager
	hookManager := hook.NewManager(logger, nil)
	hookManager.Register(hooks.NewLoggingHook(logger))
	hookManager.Register(hooks.NewMetricsHook(logger))
	hookManager.Register(hooks.NewRateLimitHook(rate.Limit(60), 100, logger))
	logger.Info("Hook Manager 初始化完成", zap.Int("hooks", len(hookManager.List())))

	// 6. 初始化应用服务
	sessionService := application.NewSessionApplicationService(sessionRepo, idGenerator)
	agentService := application.NewAgentApplicationService(agentRepo, idGenerator)
	channelService := application.NewChannelApplicationService(channelRepo, idGenerator)
	logger.Info("应用服务初始化完成")

	// 7. 初始化渠道管理器
	channelRegistry := channel.DefaultRegistry(logger)
	channelManager := channel.NewManager(messageBus)

	// 从数据库加载渠道配置
	if err := registerChannelsFromDB(channelService, channelRegistry, channelManager, logger); err != nil {
		logger.Error("加载渠道配置失败", zap.Error(err))
	}

	// 8. 启动消息分发器
	ctx, cancel := context.WithCancel(context.Background())
	messageBus.StartDispatcher(ctx)

	// 9. 启动所有渠道
	if err := channelManager.StartAll(ctx); err != nil {
		logger.Error("启动渠道失败", zap.Error(err))
	}

	// 10. 初始化 HTTP Handler (仅管理API)
	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		authSecret = "taskmanager-dev-secret"
	}

	// 创建最小化的 mux 用于管理 API
	taskRepo := _persistence.NewSQLiteTaskRepository(db)
	taskService := application.NewTaskApplicationService(taskRepo, idGenerator, eventBus, logger)
	taskHandler := httpHandler.NewTaskHandler(taskService, nil)
	userService := application.NewUserApplicationService(nil, idGenerator)
	authHandler := httpHandler.NewAuthHandler(userService, authSecret, 7*24*time.Hour)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/tasks", taskHandler.CreateTask)
	mux.HandleFunc("/api/tasks/", taskHandler.GetTask)

	// 11. 启动 HTTP Server
	server := &http.Server{
		Addr:         ":8889",
		Handler:      mux,
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

	// 12. 启动消息处理循环
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		processMessages(ctx, messageBus, sessionService, agentService, logger)
	}()

	logger.Info("渠道网关服务已启动",
		zap.Int("channels", len(channelManager.List())),
	)

	// 13. 等待中断信号
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
		factory, ok := registry.GetFactory(chType)
		if !ok {
			logger.Warn("未注册的渠道类型", zap.String("type", chType))
			continue
		}

		chInstance, err := factory(ch.Config())
		if err != nil {
			logger.Error("创建渠道实例失败",
				zap.String("name", ch.Name()),
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

// processMessages 处理来自渠道的消息
func processMessages(
	ctx context.Context,
	messageBus *channelBus.MessageBus,
	sessionService *application.SessionApplicationService,
	agentService *application.AgentApplicationService,
	logger *zap.Logger,
) {
	for {
		msg, err := messageBus.ConsumeInbound(ctx)
		if err != nil {
			return
		}
		handleInboundMessage(msg, sessionService, agentService, logger)
	}
}

// handleInboundMessage 处理入站消息
func handleInboundMessage(
	msg *channelBus.InboundMessage,
	sessionService *application.SessionApplicationService,
	agentService *application.AgentApplicationService,
	logger *zap.Logger,
) {
	logger.Info("处理入站消息",
		zap.String("channel", msg.Channel),
		zap.String("sender", msg.SenderID),
		zap.String("content", msg.Content),
	)

	// TODO: 实现消息处理逻辑
	// 1. 获取或创建 Session
	// 2. 记录 Conversation
	// 3. 调用 Agent 处理
	// 4. 发送响应
}
