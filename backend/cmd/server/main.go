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
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"github.com/weibh/taskmanager/infrastructure/cleanup"
	"github.com/weibh/taskmanager/infrastructure/config"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"github.com/weibh/taskmanager/infrastructure/llm"
	infraMcp "github.com/weibh/taskmanager/infrastructure/mcp"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/skill"
	infra_sm "github.com/weibh/taskmanager/infrastructure/statemachine"
	"github.com/weibh/taskmanager/infrastructure/utils"
	"github.com/weibh/taskmanager/infrastructure/workspace"
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

// checkAndUpdateWebhookURLs 检查所有启用的 webhook，更新过期的 webhook URL
func checkAndUpdateWebhookURLs(webhookGitHub *application.WebhookGitHubManager, webhookConfigRepo *_persistence.SQLiteGitHubWebhookConfigRepository, logger *zap.Logger) {
	ctx := context.Background()
	configs, err := webhookConfigRepo.FindAllEnabled(ctx)
	if err != nil {
		logger.Warn("failed to load enabled webhook configs", zap.Error(err))
		return
	}

	for _, config := range configs {
		needsUpdate, currentURL, err := webhookGitHub.CheckAndUpdateWebhook(ctx, config.Repo())
		if err != nil {
			logger.Warn("failed to check webhook",
				zap.String("repo", config.Repo()),
				zap.Error(err))
			continue
		}

		if needsUpdate {
			logger.Info("webhook URL mismatch, updating",
				zap.String("repo", config.Repo()),
				zap.String("current_url", currentURL),
				zap.String("expected_url", config.WebhookURL()))

			repoPath := config.Repo()
			if strings.HasPrefix(repoPath, "https://github.com/") {
				repoPath = strings.TrimPrefix(repoPath, "https://github.com/")
			}
			repoPath = strings.TrimSuffix(repoPath, ".git")

			webhookID, err := webhookGitHub.FindExistingWebhook(repoPath)
			if err != nil || webhookID == 0 {
				logger.Warn("webhook not found, skipping update",
					zap.String("repo", config.Repo()))
				continue
			}

			if err := webhookGitHub.UpdateWebhookURL(repoPath, webhookID, config.WebhookURL()); err != nil {
				logger.Warn("failed to update webhook URL",
					zap.String("repo", config.Repo()),
					zap.Error(err))
				continue
			}
			logger.Info("webhook URL updated successfully",
				zap.String("repo", config.Repo()))
		}
	}
}

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
	// 兼容旧数据库：添加 requirement_types.is_system 列
	if err := _persistence.MigrateRequirementTypeSystemColumn(db); err != nil {
		logger.Fatal("Failed to migrate requirement_types is_system column", zap.Error(err))
	}
	// 兼容旧数据库：添加 projects.max_concurrent_agents 列
	if err := _persistence.MigrateMaxConcurrentAgentsColumn(db); err != nil {
		logger.Fatal("Failed to migrate projects max_concurrent_agents column", zap.Error(err))
	}
	// 兼容旧数据库：添加 projects.default_agent_code 列
	if err := _persistence.MigrateDefaultAgentCodeColumn(db); err != nil {
		logger.Fatal("Failed to migrate projects default_agent_code column", zap.Error(err))
	}
	// 兼容旧数据库：将单心跳配置迁移到 heartbeats 表
	if err := _persistence.MigrateHeartbeatToTable(db); err != nil {
		logger.Fatal("Failed to migrate heartbeat to table", zap.Error(err))
	}
	// 兼容旧数据库：添加 requirements 表 agent 信息列
	if err := _persistence.MigrateRequirementAgentInfoColumns(db); err != nil {
		logger.Fatal("Failed to migrate requirement agent info columns", zap.Error(err))
	}
	// 预置默认心跳模板
	if err := _persistence.SeedHeartbeatTemplates(db); err != nil {
		logger.Fatal("Failed to seed heartbeat templates", zap.Error(err))
	}
	// 兼容旧数据库：添加 projects.heartbeat_scenario_code 列
	if err := _persistence.MigrateHeartbeatScenarioCodeColumn(db); err != nil {
		logger.Fatal("Failed to migrate projects heartbeat_scenario_code column", zap.Error(err))
	}
	// 兼容旧数据库：将 github_webhook_configs 表的 forwarder_pid 列改为 webhook_url 列
	if err := _persistence.MigrateGitHubWebhookConfigColumns(db); err != nil {
		logger.Fatal("Failed to migrate github_webhook_configs columns", zap.Error(err))
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
	heartbeatRepo := _persistence.NewSQLiteHeartbeatRepository(db)
	heartbeatTemplateRepo := _persistence.NewSQLiteHeartbeatTemplateRepository(db)
	heartbeatScenarioRepo := _persistence.NewSQLiteHeartbeatScenarioRepository(db)
	requirementRepo := _persistence.NewSQLiteRequirementRepository(db)
	mcpServerRepo := _persistence.NewSQLiteMCPServerRepository(db)
	bindingRepo := _persistence.NewSQLiteAgentMCPBindingRepository(db)
	webhookBindingRepo := _persistence.NewSQLiteWebhookHeartbeatBindingRepository(db)
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

	// 初始化状态机执行器（先创建，后续注入 trigger service）
	transitionExecutor := infra_sm.NewTransitionExecutor(logger)

	// 初始化状态机服务（供心跳调度器使用）
	stateMachineService := application.NewStateMachineService(stateMachineRepo, requirementRepo, transitionExecutor, replicaCleanupSvc, logger)

	// 8. 初始化心跳触发服务和调度器
	heartbeatTriggerService := application.NewHeartbeatTriggerService(
		heartbeatRepo,
		projectRepo,
		agentRepo,
		requirementRepo,
		idGenerator,
		gateway.messageBus,
		requirementDispatchService,
		stateMachineService,
	)
	heartbeatScheduler := application.NewHeartbeatSchedulerWithTriggerService(
		heartbeatRepo,
		projectRepo,
		agentRepo,
		requirementRepo,
		idGenerator,
		gateway.messageBus,
		requirementDispatchService,
		stateMachineService,
		heartbeatTriggerService,
	)

	// 将 trigger service 注入到状态机执行器
	transitionExecutor.SetHeartbeatTrigger(heartbeatTriggerService)

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
	heartbeatScenarioService := application.NewHeartbeatScenarioService(heartbeatScenarioRepo, projectRepo, heartbeatRepo, webhookBindingRepo, idGenerator, heartbeatScheduler)
	if err := heartbeatScenarioService.EnsureBuiltInScenarios(context.Background()); err != nil {
		logger.Fatal("Failed to ensure built-in heartbeat scenarios", zap.Error(err))
	}
	projectService := application.NewProjectApplicationService(projectRepo, requirementTypeRepo, idGenerator, heartbeatScenarioService)
	heartbeatService := application.NewHeartbeatApplicationService(heartbeatRepo, idGenerator, heartbeatScheduler)
	heartbeatTemplateService := application.NewHeartbeatTemplateApplicationService(heartbeatTemplateRepo, idGenerator)

	userHandler := httpHandler.NewUserHandler(userService)
	agentHandler := httpHandler.NewAgentHandler(agentService)
	providerHandler := httpHandler.NewLLMProviderHandler(providerService)
	channelHandler := httpHandler.NewChannelHandler(channelService)
	sessionHandler := httpHandler.NewSessionHandler(sessionService)
	conversationRecordHandler := httpHandler.NewConversationRecordHandler(conversationRecordService)
	projectHandler := httpHandler.NewProjectHandler(projectService)
	heartbeatHandler := httpHandler.NewHeartbeatHandlerWithTrigger(heartbeatService, heartbeatScheduler, heartbeatScheduler.TriggerService())
	heartbeatTemplateHandler := httpHandler.NewHeartbeatTemplateHandler(heartbeatTemplateService)
	heartbeatScenarioHandler := httpHandler.NewHeartbeatScenarioHandler(heartbeatScenarioService)
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
	systemLogHandler := httpHandler.NewSystemLogHandler()

	// 初始化 Webhook 相关组件
	webhookConfigRepo := _persistence.NewSQLiteGitHubWebhookConfigRepository(db)
	webhookEventLogRepo := _persistence.NewSQLiteWebhookEventLogRepository(db)
	// 使用 PublicURL（公网地址）作为 webhook 的回调地址
	// 优先从 ~/.taskmanager/config.json 读取（tunnel 创建时保存），否则使用配置文件中的 PublicURL
	webhookURL := config.GetPublicURL()
	if webhookURL == "" {
		webhookURL = cfg.API.BaseURL
		logger.Warn("Public URL not configured, using BaseURL for webhooks. GitHub webhooks require a public URL to work properly.")
	}
	webhookGitHub := application.NewWebhookGitHubManager(webhookURL)
	githubWebhookService := application.NewGitHubWebhookService(
		webhookConfigRepo,
		webhookEventLogRepo,
		webhookBindingRepo,
		heartbeatRepo,
		heartbeatTriggerService,
		idGenerator,
	)
	webhookHandler := httpHandler.NewWebhookHandler(githubWebhookService)
	githubWebhookHandler := httpHandler.NewGitHubWebhookHandler(githubWebhookService, webhookGitHub, authHandler)

	// 检查所有启用的 webhook 配置，更新过期的 webhook URL
	go checkAndUpdateWebhookURLs(webhookGitHub, webhookConfigRepo, logger)

	ginEngine := httpHandler.SetupRoutesWithManagement(
		userHandler, agentHandler, providerHandler,
		channelHandler, sessionHandler, conversationRecordHandler,
		authHandler, mcpHandler, skillHandler, projectHandler,
		requirementHandler, stateMachineHandler, projectStateMachineHandler,
		requirementTypeHandler, heartbeatHandler, heartbeatTemplateHandler,
		heartbeatScenarioHandler, systemLogHandler,
	)

	// 注册 Webhook 路由
	httpHandler.RegisterWebhookRoutes(ginEngine, webhookHandler, githubWebhookHandler, authHandler)

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
