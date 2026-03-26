# 使用示例

## 目录

- [基础使用](#基础使用)
- [创建子任务（链路追踪）](#创建子任务链路追踪)
- [查看任务树](#查看任务树)
- [使用Hook](#使用hook)
- [强制停止任务](#强制停止任务)
- [WebSocket订阅事件](#websocket订阅事件)

---

## 基础使用

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "go.uber.org/zap"
    
    "github.com/example/taskmanager/application"
    "github.com/example/taskmanager/domain"
    "github.com/example/taskmanager/infrastructure/bus"
    "github.com/example/taskmanager/infrastructure/persistence"
    "github.com/example/taskmanager/infrastructure/utils"
    _ "modernc.org/sqlite"
)

func main() {
    // 1. 初始化日志
    logger, _ := zap.NewDevelopment()
    defer logger.Sync()
    
    // 2. 初始化数据库
    db, err := sql.Open("sqlite", "./taskmanager.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // 3. 执行数据库迁移
    if err := persistence.Migrate(db); err != nil {
        panic(err)
    }
    
    // 4. 初始化基础设施
    taskRepo := persistence.NewSQLiteTaskRepository(db)
    eventBus := bus.NewEventBus()
    idGenerator := utils.NewNanoIDGenerator()
    workerPool := application.NewWorkerPool(10, 100)
    
    // 5. 初始化应用服务
    taskService := application.NewTaskApplicationService(
        taskRepo,
        nil,  // eventStore (可选)
        idGenerator,
        nil,  // hookRegistry (可选)
        eventBus,
        workerPool,
        logger,
    )
    
    queryService := application.NewQueryService(taskRepo)
    
    // 6. 订阅事件
    eventBus.Subscribe("task:created", func(e domain.DomainEvent) {
        fmt.Printf("Task created: %s\n", e.AggregateID())
    })
    
    eventBus.Subscribe("task:progress_updated", func(e domain.DomainEvent) {
        evt := e.(*domain.TaskProgressUpdatedEvent)
        fmt.Printf("Progress: %.1f%% - %s\n", 
            evt.Progress.Percentage(), 
            evt.Progress.Stage())
    })
    
    // 7. 创建任务
    ctx := context.Background()
    task, err := taskService.CreateTask(ctx, application.CreateTaskCommand{
        Name:        "数据处理任务",
        Description: "处理一批重要数据",
        Type:        "data_processing",
        Timeout:     60000, // 1分钟
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Task created: %s\n", task.ID())
    fmt.Printf("TraceID: %s\n", task.TraceID())
    fmt.Printf("SpanID: %s\n", task.SpanID())
    
    // 8. 定义任务处理器
    handler := func(ctx *domain.TaskContext) error {
        for i := 1; i <= 5; i++ {
            time.Sleep(500 * time.Millisecond)
            
            // 报告进度
            if err := ctx.ReportProgress(int64(i)*20, 
                fmt.Sprintf("阶段%d", i), 
                "处理中..."); err != nil {
                return err
            }
        }
        return nil
    }
    
    // 9. 启动任务
    if err := taskService.StartTask(ctx, task.ID(), handler, domain.EmptyHooks); err != nil {
        panic(err)
    }
    
    // 10. 等待完成
    time.Sleep(5 * time.Second)
    
    // 11. 查询结果
    dto, _ := queryService.GetTask(ctx, task.ID())
    fmt.Printf("Final status: %s\n", dto.Status)
    fmt.Printf("Progress: %.1f%%\n", dto.Progress.Percentage)
    
    // 12. 清理
    workerPool.Stop()
}
```

---

## 创建子任务（链路追踪）

```go
func parentTaskHandler(
    taskService *application.TaskApplicationService,
) domain.TaskHandler {
    return func(ctx *domain.TaskContext) error {
        parentTask := ctx.Task()
        
        fmt.Printf("父任务开始执行: %s\n", parentTask.ID())
        
        // 创建子任务1
        parentID := parentTask.ID()
        child1, err := taskService.CreateTask(ctx, application.CreateTaskCommand{
            Name:     "子任务-数据清洗",
            Type:     "data_cleaning",
            ParentID: &parentID,  // 设置父任务ID
            Timeout:  30000,
        })
        if err != nil {
            return err
        }
        
        // 创建子任务2
        parentID2 := parentTask.ID()  // 必须取地址前先赋值给变量
        child2, err := taskService.CreateTask(ctx, application.CreateTaskCommand{
            Name:     "子任务-数据转换",
            Type:     "data_transform",
            ParentID: &parentID2,
            Timeout:  30000,
        })
        if err != nil {
            return err
        }
        
        // 启动子任务
        taskService.StartTask(ctx, child1.ID(), childHandler1, domain.EmptyHooks)
        taskService.StartTask(ctx, child2.ID(), childHandler2, domain.EmptyHooks)
        
        // 父任务继续自己的工作
        for i := 1; i <= 3; i++ {
            time.Sleep(1 * time.Second)
            ctx.ReportProgress(int64(i)*33, "协调子任务", fmt.Sprintf("子任务进度检查 %d/3", i))
        }
        
        return nil
    }
}

func childHandler1(ctx *domain.TaskContext) error {
    task := ctx.Task()
    fmt.Printf("子任务1执行: %s, TraceID: %s\n", task.ID(), task.TraceID())
    
    // 模拟工作
    for i := 1; i <= 5; i++ {
        time.Sleep(200 * time.Millisecond)
        ctx.ReportProgress(int64(i)*20, "清洗中", "")
    }
    
    return nil
}

func childHandler2(ctx *domain.TaskContext) error {
    task := ctx.Task()
    fmt.Printf("子任务2执行: %s, TraceID: %s\n", task.ID(), task.TraceID())
    
    for i := 1; i <= 5; i++ {
        time.Sleep(300 * time.Millisecond)
        ctx.ReportProgress(int64(i)*20, "转换中", "")
    }
    
    return nil
}
```

---

## 查看任务树

```go
func printTaskTree(ctx context.Context, taskService *application.TaskApplicationService, traceID domain.TraceID) {
    // 获取任务树
    tree, err := taskService.GetTaskTree(ctx, traceID)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("=== Task Tree for Trace: %s ===\n", traceID)
    fmt.Printf("Total: %d, Complete: %d\n\n", tree.Total, tree.Complete)
    
    // 打印树形结构
    if tree.Root != nil {
        printNode(tree.Root, 0)
    }
    
    // 状态汇总
    summary := tree.GetStatusSummary()
    fmt.Println("\n=== Status Summary ===")
    for status, count := range summary {
        fmt.Printf("%s: %d\n", status, count)
    }
}

func printNode(node *domain.TaskNode, depth int) {
    indent := ""
    for i := 0; i < depth; i++ {
        indent += "  "
    }
    
    task := node.Task
    fmt.Printf("%s[%s] %s (Status: %s, Progress: %.1f%%)\n",
        indent,
        task.SpanID(),
        task.Name(),
        task.Status(),
        task.Progress().Percentage(),
    )
    
    for _, child := range node.Children {
        printNode(child, depth+1)
    }
}
```

**输出示例:**
```
=== Task Tree for Trace: xyz789... ===
Total: 4, Complete: 2

[root123] 数据处理任务 (Status: Running, Progress: 66.7%)
  [span001] 子任务-数据清洗 (Status: Completed, Progress: 100.0%)
    [span003] 孙任务-验证 (Status: Completed, Progress: 100.0%)
  [span002] 子任务-数据转换 (Status: Running, Progress: 40.0%)

=== Status Summary ===
Pending: 0
Running: 2
Completed: 2
```

---

## 使用Hook

```go
func main() {
    // ... 初始化代码 ...
    
    // 创建Hook注册表
    hookRegistry := hook.NewRegistry()
    
    // 注册全局钩子
    hookRegistry.RegisterGlobal(domain.HookBeforeExecute, func(ctx context.Context, task *domain.Task) error {
        fmt.Printf("[Hook] Task %s is about to start\n", task.ID())
        // 可以在这里做权限检查、资源预留等
        return nil
    })
    
    hookRegistry.RegisterGlobal(domain.HookAfterFinish, func(ctx context.Context, task *domain.Task) error {
        fmt.Printf("[Hook] Task %s finished with status %v\n", task.ID(), task.Status())
        // 可以在这里做清理、通知、记录审计日志等
        return nil
    })
    
    // 注册特定类型任务的钩子
    hookRegistry.RegisterForType("data_processing", domain.TaskHooks{
        BeforeExecute: []domain.HookFunc{
            func(ctx context.Context, task *domain.Task) error {
                fmt.Printf("[Hook] Data processing task %s starting\n", task.ID())
                return nil
            },
        },
        AfterFinish: []domain.HookFunc{
            func(ctx context.Context, task *domain.Task) error {
                // 发送通知
                if task.Status() == domain.TaskStatusCompleted {
                    fmt.Printf("[Hook] Sending success notification for task %s\n", task.ID())
                }
                return nil
            },
        },
    })
    
    // 初始化服务时传入Hook注册表
    taskService := application.NewTaskApplicationService(
        taskRepo,
        nil,
        idGenerator,
        hookRegistry,  // 传入注册表
        eventBus,
        workerPool,
        logger,
    )
    
    // 创建任务时也可以指定任务级钩子
    task, _ := taskService.CreateTask(ctx, application.CreateTaskCommand{
        Name: "特殊任务",
        Type: "special",
    })
    
    // 启动时传入任务级钩子
    taskHooks := domain.TaskHooks{
        BeforeExecute: []domain.HookFunc{
            func(ctx context.Context, t *domain.Task) error {
                fmt.Printf("[Task Hook] Only for this task: %s\n", t.ID())
                return nil
            },
        },
    }
    
    taskService.StartTask(ctx, task.ID(), handler, taskHooks)
}
```

---

## 强制停止任务

```go
func stopTaskDemo(ctx context.Context, taskService *application.TaskApplicationService) {
    // 创建一个长时间运行的任务
    longTaskHandler := func(ctx *domain.TaskContext) error {
        for i := 1; i <= 100; i++ {
            select {
            case <-ctx.Done():  // 监听取消信号
                fmt.Println("Task cancelled!")
                return ctx.Err()
            default:
                time.Sleep(100 * time.Millisecond)
                ctx.ReportProgress(int64(i), "处理中", "")
            }
        }
        return nil
    }
    
    // 创建任务
    task, _ := taskService.CreateTask(ctx, application.CreateTaskCommand{
        Name:    "长时间任务",
        Type:    "long_running",
        Timeout: 0, // 不设置超时，手动取消
    })
    
    // 启动任务
    taskService.StartTask(ctx, task.ID(), longTaskHandler, domain.EmptyHooks)
    
    // 等待3秒后取消
    time.Sleep(3 * time.Second)
    
    fmt.Printf("Cancelling task %s...\n", task.ID())
    if err := taskService.CancelTask(ctx, task.ID(), "user requested"); err != nil {
        fmt.Printf("Cancel failed: %v\n", err)
    }
    
    // 取消会级联传播到所有子任务
}
```

---

## WebSocket订阅事件

```javascript
// 前端JavaScript示例

// 连接到WebSocket
const traceId = 'xyz789...';
const ws = new WebSocket(`ws://localhost:8080/ws?trace_id=${traceId}`);

ws.onopen = function() {
    console.log('WebSocket connected');
};

ws.onmessage = function(event) {
    const msg = JSON.parse(event.data);
    
    switch(msg.event_type) {
        case 'task:created':
            console.log('Task created:', msg.aggregate_id);
            break;
            
        case 'task:started':
            console.log('Task started:', msg.aggregate_id);
            break;
            
        case 'task:progress_updated':
            const progress = msg.progress;
            console.log(`Progress: ${progress.percentage}% - ${progress.stage}`);
            updateProgressBar(msg.aggregate_id, progress.percentage);
            break;
            
        case 'task:completed':
            console.log('Task completed:', msg.aggregate_id);
            showSuccessNotification(msg.aggregate_id);
            break;
            
        case 'task:failed':
            console.log('Task failed:', msg.aggregate_id, msg.error);
            showErrorNotification(msg.aggregate_id, msg.error);
            break;
    }
};

ws.onclose = function() {
    console.log('WebSocket disconnected');
};

// 更新进度条
function updateProgressBar(taskId, percentage) {
    const bar = document.getElementById(`progress-${taskId}`);
    if (bar) {
        bar.style.width = `${percentage}%`;
        bar.textContent = `${percentage.toFixed(1)}%`;
    }
}
```

---

## 完整Web服务端示例

```go
package main

import (
    "database/sql"
    "net/http"
    
    "go.uber.org/zap"
    
    "github.com/example/taskmanager/application"
    "github.com/example/taskmanager/infrastructure/bus"
    "github.com/example/taskmanager/infrastructure/hook"
    "github.com/example/taskmanager/infrastructure/persistence"
    "github.com/example/taskmanager/infrastructure/utils"
    httphandler "github.com/example/taskmanager/interfaces/http"
    _ "modernc.org/sqlite"
)

func main() {
    // 1. 初始化
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    db, _ := sql.Open("sqlite", "./taskmanager.db")
    defer db.Close()
    persistence.Migrate(db)
    
    // 2. 基础设施
    taskRepo := persistence.NewSQLiteTaskRepository(db)
    eventBus := bus.NewEventBus()
    idGenerator := utils.NewNanoIDGenerator()
    workerPool := application.NewWorkerPool(10, 100)
    hookRegistry := hook.NewRegistry()
    
    // 3. 应用服务
    taskService := application.NewTaskApplicationService(
        taskRepo, nil, idGenerator, hookRegistry,
        eventBus, workerPool, logger,
    )
    queryService := application.NewQueryService(taskRepo)
    
    // 4. HTTP处理器
    handler := httphandler.NewTaskHandler(taskService, queryService)
    router := httphandler.SetupRoutes(handler)
    
    // 5. 启动服务器
    logger.Info("Starting server on :8080")
    if err := http.ListenAndServe(":8080", router); err != nil {
        logger.Fatal("server failed", zap.Error(err))
    }
}
```
