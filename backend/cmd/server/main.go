/**
 * 服务端入口
 */
package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
	httpHandler "github.com/weibh/taskmanager/interfaces/http"
	ws "github.com/weibh/taskmanager/interfaces/ws"
	"go.uber.org/zap"
)

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

	// 4. 初始化任务执行器
	executor := application.NewTaskExecutor()
	executor.RegisterHandler(domain.TaskTypeDataProcessing, application.DataProcessingHandler)
	executor.RegisterHandler(domain.TaskTypeFileOperation, application.FileOperationHandler)
	executor.RegisterHandler(domain.TaskTypeAPICall, application.APICallHandler)
	executor.RegisterHandler(domain.TaskTypeCustom, application.CustomHandler)

	// 5. 初始化工作池
	workerPool := application.NewWorkerPool(3, logger)

	// 5.1 初始化自动任务执行器
	autoExecutor := application.NewAutoTaskExecutor(taskRepo, eventBus, application.GetTaskRegistry(), workerPool)

	// 5.2 注册任务处理器
	executor.RegisterHandler(domain.TaskTypeDataProcessing, application.DataProcessingHandler)
	executor.RegisterHandler(domain.TaskTypeFileOperation, application.FileOperationHandler)
	executor.RegisterHandler(domain.TaskTypeAPICall, application.APICallHandler)
	executor.RegisterHandler(domain.TaskTypeCustom, application.CustomHandler)

	workerPool.SetExecuteFunc(func(ctx context.Context, task *domain.Task) {
		// 根任务（无 parent_id）使用自动执行器，会分发子任务
		// 子任务（有 parent_id）使用普通执行器，正常执行
		parentID := task.ParentID()
		if parentID == nil {
			// 根任务：使用自动执行器
			if err := autoExecutor.ExecuteAutoTask(ctx, task); err != nil {
				if ctx.Err() != context.Canceled {
					logger.Error("自动任务执行失败", zap.String("taskID", task.ID().String()), zap.Error(err))
				}
			}
		} else {
			// 子任务：使用普通执行器
			if err := executor.Execute(ctx, task, taskRepo); err != nil {
				if ctx.Err() != context.Canceled {
					logger.Error("任务执行失败", zap.String("taskID", task.ID().String()), zap.Error(err))
				}
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
	taskService.SetWorkerPool(workerPool)
	queryService := application.NewQueryService(taskRepo)

	// 6.1 订阅 TodoSubTaskCreatedEvent 事件
	eventBus.Subscribe("TodoSubTaskCreated", func(event domain.DomainEvent) {
		if e, ok := event.(*domain.TodoSubTaskCreatedEvent); ok {
			autoExecutor.HandleSubTaskCompleted(e.SubTaskIDStr(), e.ParentTaskID().String())
		}
	})

	// 7. 初始化 HTTP Handler
	taskHandler := httpHandler.NewTaskHandler(taskService, queryService)
	mux := httpHandler.SetupRoutes(taskHandler)

	// 8. 初始化 WebSocket
	wsHandler := ws.NewWebSocketHandler(eventBus)
	wsHandler.SubscribeToEvents()

	// 添加 WebSocket 路由
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler.HandleWebSocket(w, r)
	})

	// 7. 创建 HTTP Server
	server := &http.Server{
		Addr:         ":8888",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. 启动服务器
	go func() {
		logger.Info("HTTP Server 启动在 :8888")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP Server 启动失败", zap.Error(err))
		}
	}()

	// 9. 等待中断信号优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	ctx := context.Background()
	err = server.Shutdown(ctx)
	if err != nil {
		logger.Fatal("服务器关闭失败", zap.Error(err))
	}

	logger.Info("服务器已关闭")
}
