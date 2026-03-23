/**
 * 服务端入口
 */
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"github.com/weibh/taskmanager/infrastructure/llm"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
	httpHandler "github.com/weibh/taskmanager/interfaces/http"
	ws "github.com/weibh/taskmanager/interfaces/ws"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
	"github.com/weibh/taskmanager/pkg/channel"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Gateway 渠道网关组件
type Gateway struct {
	logger          *zap.Logger
	messageBus      *channelBus.MessageBus
	sessionManager  *channel.SessionManager
	processor      *channel.MessageProcessor
	channelManager *channel.Manager
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("启动任务管理服务...")

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
	taskRepo := _persistence.NewSQLiteTaskRepository(db)
	userRepo := _persistence.NewSQLiteUserRepository(db)
	agentRepo := _persistence.NewSQLiteAgentRepository(db)
	providerRepo := _persistence.NewSQLiteLLMProviderRepository(db)
	channelRepo := _persistence.NewSQLiteChannelRepository(db)
	sessionRepo := _persistence.NewSQLiteSessionRepository(db)
	conversationRecordRepo := _persistence.NewSQLiteConversationRecordRepository(db)
	mcpServerRepo := _persistence.NewSQLiteMCPServerRepository(db)
	bindingRepo := _persistence.NewSQLiteAgentMCPBindingRepository(db)
	mcpToolRepo := _persistence.NewSQLiteMCPToolRepository(db)
	mcpToolLogRepo := _persistence.NewSQLiteMCPToolLogRepository(db)

	// 4. 初始化 LLM Provider
	llmConfig := llm.DefaultConfig()
	var llmProvider llm.LLMProvider
	shouldInitLLM := llmConfig.ProviderType == "ollama" || llmConfig.APIKey != "" || llmConfig.BaseURL != "" || os.Getenv("LLM_PROVIDER") != ""

	// 4.1 初始化 Hook Manager
	hookManager := hook.NewManager(logger, nil)
	hookManager.Register(hooks.NewLoggingHook(logger))
	hookManager.Register(hooks.NewMetricsHook(logger))
	hookManager.Register(hooks.NewRateLimitHook(rate.Limit(60), 100, logger))
	logger.Info("Hook Manager 初始化完成", zap.Int("hooks", len(hookManager.List())))

	if shouldInitLLM {
		var err error
		provider, err := llm.NewLLMProvider(llmConfig)
		if err != nil {
			logger.Warn("LLM Provider 初始化失败，将使用默认子任务生成", zap.Error(err))
		} else {
			// 4.2 包装为 HookableProvider
			llmProvider = llm.NewHookableProvider(provider)
			llmProvider.(*llm.HookableProvider).SetHookManager(hookManager)
			logger.Info("LLM Provider 初始化成功（带 Hook 支持）", zap.String("provider", llmProvider.Name()), zap.String("model", llmConfig.Model))
		}
	} else {
		logger.Warn("未配置 LLM API Key，子任务生成将使用默认逻辑")
	}

	// 5. 初始化任务执行器
	executor := application.NewTaskExecutor()
	executor.RegisterHandler(domain.TaskTypeAgent, application.AgentHandlerFunc)

	// 6. 初始化工作池
	workerPool := application.NewWorkerPool(3, logger)

	// 6.1 初始化自动任务执行器
	autoExecutor := application.NewAutoTaskExecutor(taskRepo, eventBus, application.GetTaskRegistry(), workerPool)
	if llmProvider != nil {
		autoExecutor.SetLLMProvider(llmProvider)
	}

	workerPool.SetExecuteFunc(func(ctx context.Context, task *domain.Task) {
		// 所有任务都使用自动执行器，支持递归创建子任务
		if err := autoExecutor.ExecuteAutoTask(ctx, task); err != nil {
			if ctx.Err() != context.Canceled {
				logger.Error("自动任务执行失败", zap.String("taskID", task.ID().String()), zap.Error(err))
			}
		}
		// 确保任务状态被持久化
		if err := taskRepo.Save(context.Background(), task); err != nil {
			logger.Error("任务状态保存失败", zap.String("taskID", task.ID().String()), zap.Error(err))
		}
	})
	workerPool.Start()

	// 6. 初始化应用服务并连接工作池
	taskService := application.NewTaskApplicationService(
		taskRepo,
		idGenerator,
		eventBus,
		logger,
	)
	userService := application.NewUserApplicationService(userRepo, idGenerator)
	agentService := application.NewAgentApplicationService(agentRepo, idGenerator)
	providerService := application.NewLLMProviderApplicationService(providerRepo, idGenerator)
	channelService := application.NewChannelApplicationService(channelRepo, idGenerator)
	sessionService := application.NewSessionApplicationService(sessionRepo, idGenerator)
	conversationRecordService := application.NewConversationRecordApplicationService(conversationRecordRepo, idGenerator)
	mcpService := application.NewMCPApplicationService(mcpServerRepo, agentRepo, bindingRepo, mcpToolRepo, mcpToolLogRepo, idGenerator)
	ensureDefaultAdminUser(userService, userRepo, logger)
	taskService.SetWorkerPool(workerPool)
	queryService := application.NewQueryService(taskRepo)

	// 7. 初始化 HTTP Handler
	taskHandler := httpHandler.NewTaskHandler(taskService, queryService)
	userHandler := httpHandler.NewUserHandler(userService)
	agentHandler := httpHandler.NewAgentHandler(agentService)
	providerHandler := httpHandler.NewLLMProviderHandler(providerService)
	channelHandler := httpHandler.NewChannelHandler(channelService)
	sessionHandler := httpHandler.NewSessionHandler(sessionService)
	conversationRecordHandler := httpHandler.NewConversationRecordHandler(conversationRecordService)
	mcpHandler := httpHandler.NewMCPHandler(mcpService)
	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		authSecret = "taskmanager-dev-secret"
	}
	authHandler := httpHandler.NewAuthHandler(userService, authSecret, 7*24*time.Hour)
	mux := httpHandler.SetupRoutesWithManagement(taskHandler, userHandler, agentHandler, providerHandler, channelHandler, sessionHandler, conversationRecordHandler, authHandler, mcpHandler)

	// 8. 初始化 WebSocket
	wsHandler := ws.NewWebSocketHandler(eventBus)
	wsHandler.SubscribeToEvents()

	// 添加 WebSocket 路由
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler.HandleWebSocket(w, r)
	})

	// 9. 初始化渠道网关
	gateway := initGateway(channelService, logger)

	// 10. 创建 HTTP Server
	server := &http.Server{
		Addr:         ":8888",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 11. 启动服务器
	go func() {
		logger.Info("HTTP Server 启动在 :8888")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP Server 启动失败", zap.Error(err))
		}
	}()

	logger.Info("服务已启动",
		zap.Int("channels", gateway.ChannelCount()),
	)

	// 12. 等待中断信号优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	// 关闭渠道网关
	gateway.Shutdown()

	ctx := context.Background()
	err = server.Shutdown(ctx)
	if err != nil {
		logger.Fatal("服务器关闭失败", zap.Error(err))
	}

	logger.Info("服务器已关闭")
}

// initGateway 初始化渠道网关
func initGateway(channelService *application.ChannelApplicationService, logger *zap.Logger) *Gateway {
	gw := &Gateway{
		logger:          logger,
		messageBus:      channelBus.NewMessageBus(logger),
		sessionManager:  channel.NewSessionManager(logger),
		channelManager: channel.NewManager(nil),
	}

	// 创建消息处理器
	gw.processor = channel.NewMessageProcessor(gw.messageBus, gw.sessionManager, logger)

	// 初始化渠道管理器
	gw.channelManager = channel.NewManager(gw.messageBus)

	// 加载渠道配置
	gw.loadChannels(channelService)

	// 启动消息分发器
	ctx, cancel := context.WithCancel(context.Background())
	gw.messageBus.StartDispatcher(ctx)
	_ = cancel // 保留 cancel 函数用于清理

	// 启动所有渠道
	if err := gw.channelManager.StartAll(ctx); err != nil {
		logger.Error("启动渠道失败", zap.Error(err))
	}

	// 启动消息处理循环
	go gw.runMessageLoop(ctx, channelService)

	logger.Info("渠道网关初始化完成", zap.Int("channels", gw.ChannelCount()))

	return gw
}

// loadChannels 从数据库加载渠道
func (g *Gateway) loadChannels(channelService *application.ChannelApplicationService) {
	registry := channel.DefaultRegistry(g.logger)

	ctx := context.Background()
	channels, err := channelService.ListActiveChannels(ctx)
	if err != nil {
		g.logger.Error("加载渠道配置失败", zap.Error(err))
		return
	}

	for _, ch := range channels {
		chType := string(ch.Type())
		factory, ok := registry.GetFactory(chType)
		if !ok {
			g.logger.Warn("未注册的渠道类型", zap.String("type", chType))
			continue
		}

		chInstance, err := factory(ch.Config())
		if err != nil {
			g.logger.Error("创建渠道实例失败",
				zap.String("name", ch.Name()),
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

// runMessageLoop 运行消息处理循环
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
			outMsg := &channelBus.OutboundMessage{
				Channel:  msg.Channel,
				ChatID:   msg.ChatID,
				Content:  fmt.Sprintf("处理消息时出错: %v", err),
				Metadata: make(map[string]any),
			}
			g.messageBus.PublishOutbound(outMsg)
		}
	}
}

// Shutdown 关闭渠道网关
func (g *Gateway) Shutdown() {
	g.logger.Info("正在关闭渠道网关...")
	g.channelManager.StopAll()
	g.messageBus.Stop()
	g.logger.Info("渠道网关已关闭")
}

// ChannelCount 返回已注册渠道数量
func (g *Gateway) ChannelCount() int {
	return len(g.channelManager.List())
}

func ensureDefaultAdminUser(userService *application.UserApplicationService, userRepo domain.UserRepository, logger *zap.Logger) {
	ctx := context.Background()
	existingUser, err := userRepo.FindByUsername(ctx, "admin")
	if err != nil {
		logger.Warn("检查默认管理员用户失败", zap.Error(err))
		return
	}
	if existingUser != nil {
		if err := existingUser.ChangePasswordHash("admin123"); err != nil {
			logger.Warn("重置默认管理员密码失败", zap.Error(err))
			return
		}
		existingUser.Activate()
		if err := userRepo.Save(ctx, existingUser); err != nil {
			logger.Warn("保存默认管理员用户失败", zap.Error(err))
			return
		}
		logger.Info("默认管理员密码已重置", zap.String("username", "admin"))
		return
	}
	_, err = userService.CreateUser(ctx, application.CreateUserCommand{
		Username:    "admin",
		DisplayName: "系统管理员",
		Email:       "admin@local.dev",
		Password:    "admin123",
	})
	if err != nil {
		logger.Warn("创建默认管理员用户失败", zap.Error(err))
		return
	}
	logger.Info("默认管理员用户已创建", zap.String("username", "admin"))
}
