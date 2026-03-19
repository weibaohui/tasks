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

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/infrastructure/bus"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
	httpHandler "github.com/weibh/taskmanager/interfaces/http"
	ws "github.com/weibh/taskmanager/interfaces/ws"
	_ "github.com/mattn/go-sqlite3"
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

	// 4. 初始化应用服务
	taskService := application.NewTaskApplicationService(
		taskRepo,
		idGenerator,
		eventBus,
		logger,
	)
	queryService := application.NewQueryService(taskRepo)

	// 5. 初始化 HTTP Handler
	taskHandler := httpHandler.NewTaskHandler(taskService, queryService)
	mux := httpHandler.SetupRoutes(taskHandler)

	// 6. 初始化 WebSocket
	wsHandler := ws.NewWebSocketHandler(eventBus)
	wsHandler.SubscribeToEvents()

	// 添加 WebSocket 路由
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler.HandleWebSocket(w, r)
	})

	// 7. 创建 HTTP Server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. 启动服务器
	go func() {
		logger.Info("HTTP Server 启动在 :8080")
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
