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
	"path/filepath"
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
	"github.com/weibh/taskmanager/infrastructure/skill"
	"github.com/weibh/taskmanager/infrastructure/utils"
	httpHandler "github.com/weibh/taskmanager/interfaces/http"
	ws "github.com/weibh/taskmanager/interfaces/ws"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
	"github.com/weibh/taskmanager/pkg/channel"
	"go.uber.org/zap"
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
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	if runAdminSubcommandIfMatched(logger) {
		return
	}

	logger.Info("启动任务管理服务...")

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
	eventBus := bus.NewEventBus()
	taskRepo := _persistence.NewSQLiteTaskRepository(db)
	userRepo := _persistence.NewSQLiteUserRepository(db)
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
	// 注册对话记录 Hook
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

	// 5. 初始化任务执行器
	executor := application.NewTaskExecutor()
	executor.RegisterHandler(domain.TaskTypeAgent, application.AgentHandlerFunc)

	// 6. 初始化工作池
	workerPool := application.NewWorkerPool(3, logger)

	// 6.1 初始化自动任务执行器
	autoExecutor := application.NewAutoTaskExecutor(taskRepo, eventBus, application.GetTaskRegistry(), workerPool, hookManager)
	// 设置仓库用于动态 LLM 查找
	llmFactory := llm.NewLLMProviderFactory()
	autoExecutor.SetRepositories(agentRepo, providerRepo, channelRepo, llmFactory)

	// 6.2 初始化任务总结器
	summarizer := application.NewTaskSummarizer(taskRepo, autoExecutor, eventBus)
	summarizer.Start()

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
	projectService := application.NewProjectApplicationService(projectRepo, idGenerator)
	requirementService := application.NewRequirementApplicationService(requirementRepo, projectRepo, idGenerator)
	requirementDispatchService := application.NewRequirementDispatchService(requirementRepo, projectRepo, agentRepo, taskService, idGenerator)
	mcpService := application.NewMCPApplicationService(mcpServerRepo, agentRepo, bindingRepo, mcpToolRepo, mcpToolLogRepo, idGenerator)
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
	projectHandler := httpHandler.NewProjectHandler(projectService)
	requirementHandler := httpHandler.NewRequirementHandler(requirementService, requirementDispatchService)
	mcpHandler := httpHandler.NewMCPHandler(mcpService)
	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		authSecret = "taskmanager-dev-secret"
	}
	authHandler := httpHandler.NewAuthHandler(userService, authSecret, 7*24*time.Hour)

	// 7.1 初始化技能加载器
	skillsLoader := skill.NewSkillsLoader(resolveWorkspace())
	skillHandler := httpHandler.NewSkillHandler(skillsLoader)

	mux := httpHandler.SetupRoutesWithManagement(taskHandler, userHandler, agentHandler, providerHandler, channelHandler, sessionHandler, conversationRecordHandler, authHandler, mcpHandler, skillHandler, projectHandler, requirementHandler)

	// 8. 初始化 WebSocket
	wsHandler := ws.NewWebSocketHandler(eventBus)
	wsHandler.SubscribeToEvents()

	// 添加 WebSocket 路由
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler.HandleWebSocket(w, r)
	})

	// 9. 初始化渠道网关
	gateway := initGateway(channelService, agentRepo, providerRepo, taskService, workerPool, idGenerator, hookManager, logger, mcpService, skillsLoader)
	requirementDispatchService.SetInboundPublisher(gateway.messageBus)

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

// runAdminSubcommandIfMatched 在以 taskmanager 方式运行时，优先处理管理员相关子命令，避免启动主服务。
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

// runCreateAdmin 创建默认管理员用户（admin/admin123），只执行一次性操作并退出进程。
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

// runDeleteAdmin 删除默认管理员用户（admin），只执行一次性操作并退出进程。
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

// getDBAndUserRepo 初始化数据库与用户仓库，并返回清理函数用于关闭数据库连接。
func getDBAndUserRepo(logger *zap.Logger) (domain.UserRepository, domain.IDGenerator, func(), error) {
	dbPath := resolveDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("打开数据库失败(%s): %w", dbPath, err)
	}

	if err := _persistence.InitSchema(db); err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("初始化数据库结构失败(%s): %w", dbPath, err)
	}

	idGenerator := utils.NewNanoIDGenerator(21)
	userRepo := _persistence.NewSQLiteUserRepository(db)

	cleanup := func() {
		_ = db.Close()
	}

	return userRepo, idGenerator, cleanup, nil
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

// resolveWorkspace 解析工作区目录路径
func resolveWorkspace() string {
	if p := os.Getenv("TASKMANAGER_WORKSPACE"); p != "" {
		return p
	}
	// 如果当前目录存在 backend 目录，使用 backend（适配从仓库根目录执行）
	// 注意：这可能导致工作区技能与内置技能目录重叠
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend")
	}
	// 否则使用当前工作目录
	return "."
}

// initGateway 初始化渠道网关
func initGateway(
	channelService *application.ChannelApplicationService,
	agentRepo domain.AgentRepository,
	providerRepo domain.LLMProviderRepository,
	taskService *application.TaskApplicationService,
	workerPool *application.WorkerPool,
	idGenerator *utils.NanoIDGenerator,
	hookManager *hook.Manager,
	logger *zap.Logger,
	mcpService *application.MCPApplicationService,
	skillsLoader *skill.SkillsLoader,
) *Gateway {
	gw := &Gateway{
		logger:         logger,
		messageBus:     channelBus.NewMessageBus(logger),
		sessionManager: channel.NewSessionManager(logger),
		channelManager: channel.NewManager(nil),
	}

	// 注册 FeishuThinkingProcessHook（需要 messageBus）
	feishuThinkingHook := hooks.NewFeishuThinkingProcessHook(gw.messageBus, logger)
	hookManager.Register(feishuThinkingHook)
	logger.Info("已注册 FeishuThinkingProcessHook")

	// 创建消息处理器
	gw.processor = channel.NewMessageProcessor(gw.messageBus, gw.sessionManager, logger, agentRepo, providerRepo, taskService, workerPool, idGenerator, hookManager, llm.NewLLMProviderFactory(), mcpService, skillsLoader)

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
