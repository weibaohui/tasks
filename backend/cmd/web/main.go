/**
 * Web 服务入口 - HTTP API 和前端服务
 * 仅提供 HTTP API 接口，不运行业务核心（网关、调度器等）
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
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/skill"
	"github.com/weibh/taskmanager/infrastructure/utils"
	httpHandler "github.com/weibh/taskmanager/interfaces/http"
	ws "github.com/weibh/taskmanager/interfaces/ws"
	"github.com/weibh/taskmanager/internal/embed"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	logger.Info("启动 Web 管理服务...")
	logger.Info("配置加载完成", zap.Int("server_port", cfg.Server.Port), zap.String("api_base_url", cfg.API.BaseURL))

	// 2. 初始化数据库
	dbPath := config.ExpandPath(cfg.Database.Path)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal("打开数据库失败", zap.String("path", dbPath), zap.Error(err))
	}
	defer db.Close()

	// 初始化数据库 Schema
	if err := _persistence.InitSchema(db); err != nil {
		logger.Fatal("初始化数据库结构失败", zap.Error(err))
	}
	logger.Info("数据库初始化完成", zap.String("db_path", dbPath))

	// 3. 初始化依赖
	idGenerator := utils.NewNanoIDGenerator(21)
	eventBus := bus.NewEventBus()
	taskRepo := _persistence.NewSQLiteTaskRepository(db)
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

	// 4. 初始化应用服务（仅查询和命令，不启动工作池和调度器）
	taskService := application.NewTaskApplicationService(taskRepo, idGenerator, eventBus, logger)
	userService := application.NewUserApplicationService(userRepo, idGenerator)
	agentService := application.NewAgentApplicationService(agentRepo, idGenerator)
	providerService := application.NewLLMProviderApplicationService(providerRepo, idGenerator)
	channelService := application.NewChannelApplicationService(channelRepo, idGenerator)
	sessionService := application.NewSessionApplicationService(sessionRepo, idGenerator)
	conversationRecordService := application.NewConversationRecordApplicationService(conversationRecordRepo, idGenerator)
	projectService := application.NewProjectApplicationService(projectRepo, idGenerator)

	// Hook 仓储
	hookConfigRepo := _persistence.NewSQLiteRequirementHookConfigRepository(db)
	hookLogRepo := _persistence.NewSQLiteRequirementHookActionLogRepository(db)
	logger.Info("Hook 仓储初始化完成")

	queryService := application.NewQueryService(taskRepo)

	// 5. 初始化 HTTP Handler
	taskHandler := httpHandler.NewTaskHandler(taskService, queryService)
	userHandler := httpHandler.NewUserHandler(userService)
	agentHandler := httpHandler.NewAgentHandler(agentService)
	providerHandler := httpHandler.NewLLMProviderHandler(providerService)
	channelHandler := httpHandler.NewChannelHandler(channelService)
	sessionHandler := httpHandler.NewSessionHandler(sessionService)
	conversationRecordHandler := httpHandler.NewConversationRecordHandler(conversationRecordService)
	// web 服务不管理心跳调度器，传入 nil
	projectHandler := httpHandler.NewProjectHandler(projectService, nil)

	// Requirement 服务（完整版，支持 Hook 和派发）
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

	requirementService := application.NewRequirementApplicationService(
		requirementRepo,
		projectRepo,
		idGenerator,
		hookExecutor,
		replicaAgentManager,
	)

	requirementDispatchService := application.NewRequirementDispatchService(
		requirementRepo,
		projectRepo,
		agentRepo,
		taskService,
		sessionService,
		idGenerator,
		replicaAgentManager,
		hookExecutor,
	)
	logger.Info("RequirementDispatchService 初始化完成")

	requirementHandler := httpHandler.NewRequirementHandler(requirementService, requirementDispatchService)

	mcpService := application.NewMCPApplicationService(mcpServerRepo, agentRepo, bindingRepo, mcpToolRepo, mcpToolLogRepo, idGenerator)
	mcpHandler := httpHandler.NewMCPHandler(mcpService)

	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		authSecret = "taskmanager-dev-secret"
	}
	authHandler := httpHandler.NewAuthHandler(userService, userTokenRepo, idGenerator, authSecret)

	// 技能加载器
	skillsLoader := skill.NewSkillsLoader(resolveWorkspace())
	skillHandler := httpHandler.NewSkillHandler(skillsLoader)

	// Hook Handler
	hookHandler := httpHandler.NewHookHandler(hookConfigRepo, hookLogRepo, idGenerator)
	logger.Info("Hook Handler 初始化完成")

	mux := httpHandler.SetupRoutesWithManagement(
		taskHandler, userHandler, agentHandler, providerHandler,
		channelHandler, sessionHandler, conversationRecordHandler,
		authHandler, mcpHandler, skillHandler, projectHandler,
		requirementHandler, hookHandler,
	)

	// 6. 初始化 WebSocket（仅用于前端实时通知）
	wsHandler := ws.NewWebSocketHandler(eventBus)
	wsHandler.SubscribeToEvents()

	// 添加 WebSocket 路由
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler.HandleWebSocket(w, r)
	})

	// 7. 添加前端静态文件路由（SPA）
	// API 路由优先，非 API 路由交给前端处理
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

	// 8. 创建 HTTP Server
	webPort := cfg.Server.Port

	addr := fmt.Sprintf(":%d", webPort)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. 启动服务器
	go func() {
		logger.Info("HTTP Server 启动", zap.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP Server 启动失败", zap.Error(err))
		}
	}()

	// 9. 等待中断信号优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭 Web 服务...")

	ctx := context.Background()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭失败", zap.Error(err))
	}

	logger.Info("Web 服务已关闭")
}

// zapRequirementLogger 实现 domain.RequirementStateHookLogger 接口
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

// resolveWorkspace 解析工作区目录路径
func resolveWorkspace() string {
	if p := os.Getenv("TASKMANAGER_WORKSPACE"); p != "" {
		return p
	}
	// 如果当前目录存在 backend 目录，使用 backend
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend")
	}
	// 否则使用当前工作目录
	return "."
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
