/**
 * 服务端入口 - 核心业务服务
 * 包含 HTTP API、渠道网关、调度器等所有功能
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

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"github.com/weibh/taskmanager/infrastructure/config"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"github.com/weibh/taskmanager/infrastructure/cleanup"
	"github.com/weibh/taskmanager/infrastructure/llm"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/workspace"
	"github.com/weibh/taskmanager/infrastructure/skill"
	infra_sm "github.com/weibh/taskmanager/infrastructure/statemachine"
	infraMcp "github.com/weibh/taskmanager/infrastructure/mcp"
	"github.com/weibh/taskmanager/infrastructure/utils"
	httpHandler "github.com/weibh/taskmanager/interfaces/http"
	ws "github.com/weibh/taskmanager/interfaces/ws"
	"github.com/weibh/taskmanager/internal/embed"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin123"
)

func main() {
	// 1. 加载配置（先加载配置以获取日志级别）
	cfg, err := config.Load()
	if err != nil {
		// 配置加载失败前，使用默认 logger
		fallbackLogger, _ := zap.NewDevelopment()
		defer fallbackLogger.Sync()
		fallbackLogger.Fatal("加载配置失败", zap.Error(err))
	}

	// 2. 根据配置初始化日志
	logger := initLogger(cfg.Logging.Level)
	defer logger.Sync()

	if runAdminSubcommandIfMatched(logger) {
		return
	}

	logger.Info("启动任务管理核心服务...")
	logger.Info("配置加载完成", zap.String("api_base_url", cfg.API.BaseURL))

	// 2. 初始化数据库
	dbPath := config.ExpandPath(cfg.Database.Path)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal("Failed to open database", zap.String("path", dbPath), zap.Error(err))
	}
	defer db.Close()

	// 初始化数据库 Schema
	if err := _persistence.InitSchema(db); err != nil {
		logger.Fatal("Failed to init schema", zap.Error(err))
	}
	// 兼容旧数据库：迁移 claude_runtime 列到 agent_runtime
	if err := _persistence.MigrateClaudeRuntimeColumns(db); err != nil {
		logger.Fatal("Failed to migrate schema", zap.Error(err))
	}
	// 兼容旧数据库：添加 progress_data 列
	if err := _persistence.MigrateProgressDataColumn(db); err != nil {
		logger.Fatal("Failed to migrate progress_data column", zap.Error(err))
	}
	logger.Info("数据库初始化完成", zap.String("db_path", dbPath))

	// 3. 初始化依赖
	idGenerator := utils.NewNanoIDGenerator(utils.DefaultIDSize)

	eventBus := bus.NewEventBus()
	userRepo := _persistence.NewSQLiteUserRepository(db)
	userTokenRepo := _persistence.NewSQLiteUserTokenRepository(db)
	agentRepo := _persistence.NewSQLiteAgentRepository(db)
	providerRepo := _persistence.NewSQLiteLLMProviderRepository(db)
	channelRepo := _persistence.NewSQLiteChannelRepository(db)
	sessionRepo := _persistence.NewSQLiteSessionRepository(db)
	conversationRecordRepo := _persistence.NewSQLiteConversationRecordRepository(db)
	projectRepo := _persistence.NewSQLiteProjectRepository(db)
	requirementRepo := _persistence.NewSQLiteRequirementRepository(db)
	mcpServerRepo := _persistence.NewSQLiteMCPServerRepository(db)
	bindingRepo := _persistence.NewSQLiteAgentMCPBindingRepository(db)
	mcpToolRepo := _persistence.NewSQLiteMCPToolRepository(db)
	mcpToolLogRepo := _persistence.NewSQLiteMCPToolLogRepository(db)

	// 4. 初始化 Hook Manager
	hookManager := hook.NewManager(logger, nil)
	hookManager.Register(hooks.NewLoggingHook(logger))
	hookManager.Register(hooks.NewMetricsHook(logger))
	hookManager.Register(hooks.NewRateLimitHook(rate.Limit(60), 100, logger))
	convRecordHook := hooks.NewConversationRecordHook(conversationRecordRepo, idGenerator, logger, &hooks.ConversationRecordHookConfig{
		SessionKeyExtractor: func(ctx *domain.HookContext) string {
			return ctx.GetMetadata("session_key")
		},
		UserCodeExtractor: func(ctx *domain.HookContext) string {
			return ctx.GetMetadata("user_code")
		},
		AgentCodeExtractor: func(ctx *domain.HookContext) string {
			return ctx.GetMetadata("agent_code")
		},
		ChannelCodeExtractor: func(ctx *domain.HookContext) string {
			return ctx.GetMetadata("channel_code")
		},
		ChannelTypeExtractor: func(ctx *domain.HookContext) string {
			return ctx.GetMetadata("channel_type")
		},
	})
	hookManager.Register(convRecordHook)
	progressTrackingHook := hooks.NewProgressTrackingHook(requirementRepo, logger)
	hookManager.Register(progressTrackingHook)
	logger.Info("Hook Manager 初始化完成", zap.Int("hooks", len(hookManager.List())))

	// 5. 初始化应用服务
	channelService := application.NewChannelApplicationService(channelRepo, idGenerator)
	sessionService := application.NewSessionApplicationService(sessionRepo, idGenerator)

	replicaCleanupSvc := cleanup.NewReplicaCleanupService(agentRepo, &workspace.OSWorkspaceManager{})
	logger.Info("ReplicaCleanupService 初始化完成")

	// 初始化状态机仓库（提前初始化，供 requirementDispatchService 使用）
	stateMachineRepo := _persistence.NewSQLiteStateMachineRepository(db)

	mcpService := application.NewMCPApplicationService(mcpServerRepo, agentRepo, bindingRepo, mcpToolRepo, mcpToolLogRepo, idGenerator, &infraMcp.MCPClientFactoryImpl{})

	// 初始化需求类型仓储
	requirementTypeRepo := _persistence.NewSQLiteRequirementTypeEntityRepository(db)

	// 6. 初始化技能加载器（gateway 需要）
	skillsLoader := skill.NewSkillsLoader(resolveWorkspace())

	// 7. 初始化渠道网关
	gateway := initGateway(channelService, sessionService, agentRepo, providerRepo, idGenerator, hookManager, logger, mcpService, skillsLoader, requirementRepo, conversationRecordRepo, replicaCleanupSvc)

	// 初始化需求派发服务（需要 gateway.messageBus，所以放在 gateway 之后）
	requirementDispatchService := application.NewRequirementDispatchService(
		requirementRepo,
		projectRepo,
		agentRepo,
		sessionService,
		idGenerator,
		replicaCleanupSvc,
		stateMachineRepo,
		&config.ConfigWorkspaceProvider{},
		&workspace.OSWorkspaceManager{},
		gateway.messageBus,
	)

	// 初始化状态机执行器和服务（供心跳调度器使用）
	transitionExecutor := infra_sm.NewTransitionExecutor(logger)
	stateMachineService := application.NewStateMachineService(stateMachineRepo, requirementRepo, transitionExecutor, logger)

	// 8. 初始化心跳调度器
	heartbeatScheduler := application.NewHeartbeatScheduler(
		projectRepo,
		agentRepo,
		requirementRepo,
		idGenerator,
		gateway.messageBus,
		requirementDispatchService,
		stateMachineService,
	)

	// 启动心跳调度器
	heartbeatCtx := context.Background()
	if err := heartbeatScheduler.Start(heartbeatCtx); err != nil {
		logger.Error("心跳调度器启动失败", zap.Error(err))
	} else {
		logger.Info("心跳调度器启动完成")
	}

	logger.Info("核心服务已启动",
		zap.Int("channels", gateway.ChannelCount()),
	)

	// 9. 初始化 HTTP API Handler
	userService := application.NewUserApplicationService(userRepo, idGenerator)
	agentService := application.NewAgentApplicationService(agentRepo, idGenerator)
	providerService := application.NewLLMProviderApplicationService(providerRepo, idGenerator, llm.TestLLMConnection)
	conversationRecordService := application.NewConversationRecordApplicationService(conversationRecordRepo, idGenerator)
	projectService := application.NewProjectApplicationService(projectRepo, requirementTypeRepo, idGenerator)

	userHandler := httpHandler.NewUserHandler(userService)
	agentHandler := httpHandler.NewAgentHandler(agentService)
	providerHandler := httpHandler.NewLLMProviderHandler(providerService)
	channelHandler := httpHandler.NewChannelHandler(channelService)
	sessionHandler := httpHandler.NewSessionHandler(sessionService)
	conversationRecordHandler := httpHandler.NewConversationRecordHandler(conversationRecordService)
	projectHandler := httpHandler.NewProjectHandler(projectService, heartbeatScheduler)
	requirementService := application.NewRequirementApplicationService(
		requirementRepo,
		projectRepo,
		idGenerator,
		replicaCleanupSvc,
		stateMachineRepo,
	)
	requirementHandler := httpHandler.NewRequirementHandler(requirementService, requirementDispatchService, agentRepo)
	mcpHandler := httpHandler.NewMCPHandler(mcpService)

	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		authSecret = "taskmanager-dev-secret"
	}
	authHandler := httpHandler.NewAuthHandler(userService, userTokenRepo, idGenerator, authSecret)
	skillHandler := httpHandler.NewSkillHandler(skillsLoader)

	// 初始化状态机 handler（stateMachineService 已在前面初始化）
	stateMachineHandler := httpHandler.NewStateMachineHandler(stateMachineService)

	// 初始化项目状态机应用服务
	projectStateMachineService := application.NewProjectStateMachineApplicationService(stateMachineRepo)
	projectStateMachineHandler := httpHandler.NewProjectStateMachineHandler(projectStateMachineService)

	// 初始化需求类型 handler
	requirementTypeHandler := httpHandler.NewRequirementTypeHandler(requirementTypeRepo)

	ginEngine := httpHandler.SetupRoutesWithManagement(
		userHandler, agentHandler, providerHandler,
		channelHandler, sessionHandler, conversationRecordHandler,
		authHandler, mcpHandler, skillHandler, projectHandler,
		requirementHandler, stateMachineHandler, projectStateMachineHandler,
		requirementTypeHandler,
	)

	// 10. 初始化 WebSocket（用于前端实时通知）
	wsHandler := ws.NewWebSocketHandler(eventBus)
	wsHandler.SubscribeToEvents()
	ginEngine.GET("/ws", gin.WrapF(wsHandler.HandleWebSocket))

	// 11. 添加前端静态文件路由（SPA）
	embed.SetupFrontendRoutes(ginEngine)

	// 12. 启动 HTTP Server
	webPort := cfg.Server.Port
	addr := fmt.Sprintf(":%d", webPort)
	server := &http.Server{
		Addr:         addr,
		Handler:      ginEngine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	logger.Info("HTTP API 服务启动", zap.String("addr", addr))

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP Server 启动失败", zap.Error(err))
		}
	}()

	// 13. 等待中断信号优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭核心服务...")

	// 关闭 HTTP Server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("HTTP Server 关闭失败", zap.Error(err))
	}

	// 关闭心跳调度器
	heartbeatScheduler.Stop()

	// 关闭渠道网关
	gateway.Shutdown()

	logger.Info("核心服务已关闭")
}
