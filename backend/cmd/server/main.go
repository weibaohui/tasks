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
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"github.com/weibh/taskmanager/infrastructure/config"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/skill"
	infra_sm "github.com/weibh/taskmanager/infrastructure/state_machine"
	"github.com/weibh/taskmanager/infrastructure/utils"
	httpHandler "github.com/weibh/taskmanager/interfaces/http"
	ws "github.com/weibh/taskmanager/interfaces/ws"
	"github.com/weibh/taskmanager/internal/embed"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
	"github.com/weibh/taskmanager/pkg/channel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
)

const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin123"
)

// Gateway 渠道网关组件
type Gateway struct {
	logger         *zap.Logger
	messageBus     *channelBus.MessageBus
	sessionManager *channel.SessionManager
	processor      *channel.MessageProcessor
	channelManager *channel.Manager
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
	logger.Info("数据库初始化完成", zap.String("db_path", dbPath))

	// 3. 初始化依赖
	idGenerator := utils.NewNanoIDGenerator(21)

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
	logger.Info("Hook Manager 初始化完成", zap.Int("hooks", len(hookManager.List())))

	// 5. 初始化应用服务
	channelService := application.NewChannelApplicationService(channelRepo, idGenerator)
	sessionService := application.NewSessionApplicationService(sessionRepo, idGenerator)

	// 初始化 Hook 配置仓储
	hookConfigRepo := _persistence.NewSQLiteRequirementHookConfigRepository(db)
	hookLogRepo := _persistence.NewSQLiteRequirementHookActionLogRepository(db)
	logger.Info("Hook 仓储初始化完成")

	hookLogger := &zapRequirementLogger{logger: logger}
	hookExecutor := domain.NewConfigurableHookExecutor(
		hookConfigRepo,
		hookLogRepo,
		nil,
		hookLogger,
		idGenerator,
	)
	logger.Info("ConfigurableHookExecutor 初始化完成")

	replicaAgentManager := domain.NewReplicaAgentManager(agentRepo)
	logger.Info("ReplicaAgentManager 初始化完成")

	requirementDispatchService := application.NewRequirementDispatchService(
		requirementRepo,
		projectRepo,
		agentRepo,
		nil, // taskService - no longer used
		sessionService,
		idGenerator,
		replicaAgentManager,
		hookExecutor,
	)
	mcpService := application.NewMCPApplicationService(mcpServerRepo, agentRepo, bindingRepo, mcpToolRepo, mcpToolLogRepo, idGenerator)

	// 6. 初始化技能加载器（gateway 需要）
	skillsLoader := skill.NewSkillsLoader(resolveWorkspace())

	// 7. 初始化渠道网关
	gateway := initGateway(channelService, sessionService, agentRepo, providerRepo, idGenerator, hookManager, logger, mcpService, skillsLoader, requirementRepo, conversationRecordRepo, hookExecutor, replicaAgentManager)
	requirementDispatchService.SetInboundPublisher(gateway.messageBus)

	// 8. 初始化心跳调度器
	heartbeatScheduler := application.NewHeartbeatScheduler(
		projectRepo,
		agentRepo,
		requirementRepo,
		idGenerator,
		gateway.messageBus,
		requirementDispatchService,
	)

	// 添加 TriggerAgentExecutor 到 Hook 执行器
	triggerAgentExecutor := hook.NewTriggerAgentExecutor(agentRepo, requirementRepo, projectRepo, idGenerator, gateway.messageBus)
	hookExecutor.AddExecutor(triggerAgentExecutor)
	fmt.Printf("[DEBUG] TriggerAgentExecutor 注册完成\n")
	logger.Info("TriggerAgentExecutor 注册完成")

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
	providerService := application.NewLLMProviderApplicationService(providerRepo, idGenerator)
	conversationRecordService := application.NewConversationRecordApplicationService(conversationRecordRepo, idGenerator)
	projectService := application.NewProjectApplicationService(projectRepo, idGenerator)

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
		hookExecutor,
		replicaAgentManager,
	)
	requirementHandler := httpHandler.NewRequirementHandler(requirementService, requirementDispatchService)
	mcpHandler := httpHandler.NewMCPHandler(mcpService)

	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		authSecret = "taskmanager-dev-secret"
	}
	authHandler := httpHandler.NewAuthHandler(userService, userTokenRepo, idGenerator, authSecret)
	skillHandler := httpHandler.NewSkillHandler(skillsLoader)
	hookHandler := httpHandler.NewHookHandler(hookConfigRepo, hookLogRepo, idGenerator)

	// 初始化状态机
	stateMachineRepo := _persistence.NewSQLiteStateMachineRepository(db)
	transitionExecutor := infra_sm.NewTransitionExecutor(logger)
	stateMachineService := application.NewStateMachineService(stateMachineRepo, transitionExecutor, logger)
	stateMachineHandler := httpHandler.NewStateMachineHandler(stateMachineService)

	mux := httpHandler.SetupRoutesWithManagement(
		userHandler, agentHandler, providerHandler,
		channelHandler, sessionHandler, conversationRecordHandler,
		authHandler, mcpHandler, skillHandler, projectHandler,
		requirementHandler, hookHandler, stateMachineHandler,
	)

	// 10. 初始化 WebSocket（用于前端实时通知）
	wsHandler := ws.NewWebSocketHandler(eventBus)
	wsHandler.SubscribeToEvents()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler.HandleWebSocket(w, r)
	})

	// 11. 添加前端静态文件路由（SPA）
	frontendHandler := embed.SetupFrontendRoutes()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// API 路由和 WebSocket 路由不走前端
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws") {
			http.NotFound(w, r)
			return
		}
		frontendHandler.ServeHTTP(w, r)
	})

	// 12. 启动 HTTP Server
	webPort := cfg.Server.Port
	addr := fmt.Sprintf(":%d", webPort)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
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

func runAdminSubcommandIfMatched(logger *zap.Logger) bool {
	if len(os.Args) < 2 {
		return false
	}

	switch os.Args[1] {
	case "create-admin":
		if err := runCreateAdmin(logger); err != nil {
			logger.Fatal("创建默认管理员用户失败", zap.Error(err))
		}
		return true
	case "delete-admin":
		if err := runDeleteAdmin(logger); err != nil {
			logger.Fatal("删除默认管理员用户失败", zap.Error(err))
		}
		return true
	default:
		return false
	}
}

func runCreateAdmin(logger *zap.Logger) error {
	userRepo, idGen, cleanup, err := getDBAndUserRepo(logger)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	existingUser, err := userRepo.FindByUsername(ctx, DefaultAdminUsername)
	if err != nil {
		return fmt.Errorf("检查用户失败: %w", err)
	}
	if existingUser != nil {
		logger.Info("管理员用户已存在", zap.String("username", DefaultAdminUsername))
		return nil
	}

	userService := application.NewUserApplicationService(userRepo, idGen)
	user, err := userService.CreateUser(ctx, application.CreateUserCommand{
		Username:    DefaultAdminUsername,
		DisplayName: "系统管理员",
		Email:       "admin@local.dev",
		Password:    DefaultAdminPassword,
	})
	if err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	logger.Info("管理员用户创建成功",
		zap.String("username", user.Username()),
		zap.String("userCode", user.UserCode().String()),
	)
	fmt.Printf("初始密码: %s (请登录后立即修改)\n", DefaultAdminPassword)
	return nil
}

func runDeleteAdmin(logger *zap.Logger) error {
	userRepo, _, cleanup, err := getDBAndUserRepo(logger)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	existingUser, err := userRepo.FindByUsername(ctx, DefaultAdminUsername)
	if err != nil {
		return fmt.Errorf("查找用户失败: %w", err)
	}
	if existingUser == nil {
		logger.Info("管理员用户不存在", zap.String("username", DefaultAdminUsername))
		return nil
	}

	if err := userRepo.Delete(ctx, existingUser.ID()); err != nil {
		return fmt.Errorf("删除管理员用户失败: %w", err)
	}

	logger.Info("管理员用户已删除", zap.String("username", DefaultAdminUsername))
	return nil
}

func getDBAndUserRepo(logger *zap.Logger) (domain.UserRepository, domain.IDGenerator, func(), error) {
	dbPath := resolveDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("打开数据库失败(%s): %w", dbPath, err)
	}

	if err := _persistence.InitSchema(db); err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("初始化数据库结构失败(%s): %w", dbPath, err)
	}

	idGenerator := utils.NewNanoIDGenerator(21)

	userRepo := _persistence.NewSQLiteUserRepository(db)
	cleanup := func() {
		db.Close()
	}

	return userRepo, idGenerator, cleanup, nil
}

func resolveDBPath() string {
	if p := os.Getenv("TASKMANAGER_DB_PATH"); p != "" {
		return p
	}
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	if st, err := os.Stat("./cmd/server"); err == nil && st.IsDir() {
		return filepath.FromSlash("./tasks.db")
	}
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend/tasks.db")
	}
	return filepath.FromSlash("./tasks.db")
}

func resolveWorkspace() string {
	if p := os.Getenv("TASKMANAGER_WORKSPACE"); p != "" {
		return p
	}
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend")
	}
	return "."
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
	skillsLoader *skill.SkillsLoader,
	requirementRepo domain.RequirementRepository,
	conversationRecordRepo domain.ConversationRecordRepository,
	hookExecutor *domain.ConfigurableHookExecutor,
	replicaAgentManager *domain.ReplicaAgentManager,
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

	gw.processor = channel.NewMessageProcessor(gw.messageBus, gw.sessionManager, logger, agentRepo, providerRepo, nil, sessionService, nil, idGenerator, hookManager, llm.NewLLMProviderFactory(), mcpService, skillsLoader, requirementRepo, conversationRecordRepo, hookExecutor, replicaAgentManager)
	gw.channelManager = channel.NewManager(gw.messageBus)
	gw.loadChannels(channelService)

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

type zapRequirementLogger struct {
	logger *zap.Logger
}

func (l *zapRequirementLogger) Debug(msg string, fields ...domain.RequirementStateHookLogField) {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		if sf, ok := f.(domain.RequirementStateHookLogField); ok {
			zapFields[i] = l.toZapField(sf)
		}
	}
	l.logger.Debug(msg, zapFields...)
}

func (l *zapRequirementLogger) Info(msg string, fields ...domain.RequirementStateHookLogField) {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		if sf, ok := f.(domain.RequirementStateHookLogField); ok {
			zapFields[i] = l.toZapField(sf)
		}
	}
	l.logger.Info(msg, zapFields...)
}

func (l *zapRequirementLogger) Error(msg string, fields ...domain.RequirementStateHookLogField) {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		if sf, ok := f.(domain.RequirementStateHookLogField); ok {
			zapFields[i] = l.toZapField(sf)
		}
	}
	l.logger.Error(msg, zapFields...)
}

func (l *zapRequirementLogger) toZapField(f domain.RequirementStateHookLogField) zap.Field {
	switch v := f.(type) {
	case domain.StringField:
		return zap.String(v.Key, v.Val)
	default:
		if af, ok := f.(domain.AnyField); ok {
			return zap.Any(af.Key, af.Val)
		}
		return zap.Any("unknown", f)
	}
}

// initLogger 根据配置的日志级别初始化 zap logger
func initLogger(level string) *zap.Logger {
	// 解析日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn", "warning":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 创建自定义配置
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapLevel),
		Development: true,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		// 构建失败时返回默认开发模式 logger
		fallback, _ := zap.NewDevelopment()
		return fallback
	}
	return logger
}
