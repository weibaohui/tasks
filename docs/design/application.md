# 应用层设计 (Application Layer)

应用层协调领域对象完成用例，不包含业务规则，只负责流程编排。

## 目录

- [TaskApplicationService](#taskapplicationservice)
- [TaskRuntime](#taskruntime)
- [QueryService (CQRS)](#queryservice-cqrs)
- [DTO定义](#dto定义)
- [WorkerPool](#workerpool)

---

## TaskRuntime

TaskRuntime 负责管理任务的运行时上下文（context），因为领域层的 Task 实体不应持有 context。

```go
// application/runtime.go
package application

import (
    "context"
    "sync"
    "time"
    
    "github.com/example/taskmanager/domain"
)

// TaskRuntime 任务运行时管理器
type TaskRuntime struct {
    mu          sync.RWMutex
    running     map[domain.TaskID]context.CancelFunc
    contexts    map[domain.TaskID]context.Context
}

// NewTaskRuntime 创建任务运行时管理器
func NewTaskRuntime() *TaskRuntime {
    return &TaskRuntime{
        running:  make(map[domain.TaskID]context.CancelFunc),
        contexts: make(map[domain.TaskID]context.Context),
    }
}

// Register 注册任务运行时的 context
func (rt *TaskRuntime) Register(taskID domain.TaskID, ctx context.Context, cancel context.CancelFunc) {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    rt.running[taskID] = cancel
    rt.contexts[taskID] = ctx
}

// GetContext 获取任务的 context
func (rt *TaskRuntime) GetContext(taskID domain.TaskID) (context.Context, bool) {
    rt.mu.RLock()
    defer rt.mu.RUnlock()
    ctx, ok := rt.contexts[taskID]
    return ctx, ok
}

// Cancel 取消任务
func (rt *TaskRuntime) Cancel(taskID domain.TaskID) bool {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    if cancel, ok := rt.running[taskID]; ok {
        cancel()
        delete(rt.running, taskID)
        delete(rt.contexts, taskID)
        return true
    }
    return false
}

// Unregister 注销任务
func (rt *TaskRuntime) Unregister(taskID domain.TaskID) {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    delete(rt.running, taskID)
    delete(rt.contexts, taskID)
}

// CreateContext 为任务创建带超时的 context
func (rt *TaskRuntime) CreateContext(taskID domain.TaskID, timeout time.Duration) (context.Context, context.CancelFunc) {
    var ctx context.Context
    var cancel context.CancelFunc
    
    if timeout > 0 {
        ctx, cancel = context.WithTimeout(context.Background(), timeout)
    } else {
        ctx, cancel = context.WithCancel(context.Background())
    }
    
    rt.Register(taskID, ctx, cancel)
    return ctx, cancel
}
```

---

## TaskApplicationService

```go
// application/task_service.go
package application

import (
    "context"
    "errors"
    "fmt"
    "time"
    
    "go.uber.org/zap"
    
    "github.com/example/taskmanager/domain"
    "github.com/example/taskmanager/infrastructure/bus"
)

// TaskApplicationService 任务应用服务
type TaskApplicationService struct {
    taskRepo      domain.TaskRepository
    eventStore    domain.EventStore
    idGenerator   domain.IDGenerator
    hookRegistry  domain.HookRegistry
    hookExecutor  *domain.HookExecutor
    treeBuilder   *domain.TaskTreeBuilder
    eventBus      *bus.EventBus
    executor      *domain.TaskExecutor
    workerPool    *WorkerPool
    taskRuntime   *TaskRuntime  // 运行时管理器
    logger        *zap.Logger
}

func NewTaskApplicationService(
    taskRepo domain.TaskRepository,
    eventStore domain.EventStore,
    idGenerator domain.IDGenerator,
    hookRegistry domain.HookRegistry,
    eventBus *bus.EventBus,
    workerPool *WorkerPool,
    logger *zap.Logger,
) *TaskApplicationService {
    hookExecutor := domain.NewHookExecutor(hookRegistry)
    return &TaskApplicationService{
        taskRepo:      taskRepo,
        eventStore:    eventStore,
        idGenerator:   idGenerator,
        hookRegistry:  hookRegistry,
        hookExecutor:  hookExecutor,
        treeBuilder:   domain.NewTaskTreeBuilder(taskRepo),
        eventBus:      eventBus,
        executor:      domain.NewTaskExecutor(hookExecutor),
        workerPool:    workerPool,
        taskRuntime:   NewTaskRuntime(),  // 初始化运行时管理器
        logger:        logger,
    }
}
```

### 创建任务用例

```go
// CreateTaskCommand 创建任务命令 (DTO)
type CreateTaskCommand struct {
    Name        string
    Description string
    Type        domain.TaskType
    Metadata    map[string]interface{}
    Timeout     int64  // 毫秒
    MaxRetries  int
    Priority    int
    ParentID    *domain.TaskID  // 可选，创建子任务时使用
    Hooks       domain.TaskHooks
}

// CreateTask 创建任务用例
func (s *TaskApplicationService) CreateTask(
    ctx context.Context,
    cmd CreateTaskCommand,
) (*domain.Task, error) {
    // 1. 生成ID
    taskID := domain.TaskID(s.idGenerator.Generate())
    spanID := domain.SpanID(s.idGenerator.Generate())
    
    // 2. 确定TraceID
    var traceID domain.TraceID
    if cmd.ParentID != nil {
        // 子任务：继承父任务的TraceID
        parent, err := s.taskRepo.FindByID(ctx, *cmd.ParentID)
        if err != nil {
            return nil, fmt.Errorf("parent task not found: %w", err)
        }
        traceID = parent.TraceID()
    } else {
        // 根任务：生成新TraceID
        traceID = domain.TraceID(s.idGenerator.Generate())
    }
    
    // 3. 创建领域实体
    task, err := domain.NewTask(
        taskID,
        traceID,
        spanID,
        cmd.ParentID,
        cmd.Name,
        cmd.Description,
        cmd.Type,
        cmd.Metadata,
        time.Duration(cmd.Timeout)*time.Millisecond,
        cmd.MaxRetries,
        cmd.Priority,
    )
    if err != nil {
        return nil, err
    }
    
    // 4. 执行创建前钩子
    if err := s.hookExecutor.ExecuteBeforeCreate(ctx, task, cmd.Hooks); err != nil {
        return nil, err
    }
    
    // 5. 持久化任务
    if err := s.taskRepo.Save(ctx, task); err != nil {
        return nil, err
    }
    
    // 6. 执行创建后钩子
    s.hookExecutor.ExecuteAfterCreate(ctx, task, cmd.Hooks)
    
    // 7. 发布领域事件
    s.publishDomainEvents(task)
    
    s.logger.Info("task created",
        zap.String("task_id", task.ID().String()),
        zap.String("trace_id", task.TraceID().String()),
        zap.String("span_id", task.SpanID().String()),
    )
    
    return task, nil
}
```

### 启动任务用例

```go
// StartTask 启动任务用例
func (s *TaskApplicationService) StartTask(
    ctx context.Context,
    taskID domain.TaskID,
    handler domain.TaskHandler,
    hooks domain.TaskHooks,
) error {
    // 1. 获取任务
    task, err := s.taskRepo.FindByID(ctx, taskID)
    if err != nil {
        return err
    }

    // 2. 预检查：能否启动
    if task.Status() != domain.TaskStatusPending {
        return domain.ErrTaskAlreadyStarted
    }

    // 3. 先更新任务状态（确保状态变更先于 worker pool 提交）
    if err := task.Start(); err != nil {
        return err
    }

    // 4. 持久化状态变更（如果持久化失败，任务不会提交到 worker pool）
    if err := s.taskRepo.Save(ctx, task); err != nil {
        return err
    }

    // 5. 提交到工作池（使用新的 context 避免原 context 被取消）
    submitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    submitted := make(chan bool, 1)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                s.logger.Error("worker pool submit panic", zap.Any("recover", r))
                submitted <- false
            }
        }()
        s.workerPool.Submit(func() {
            s.executeTask(task, handler, hooks)
        })
        submitted <- true
    }()

    select {
    case ok := <-submitted:
        if !ok {
            // 提交失败，但状态已是 Running，需要回滚或标记为失败
            s.logger.Error("failed to submit task to worker pool, rolling back task status")
            task.RollbackStart() // 需要添加此方法将状态回滚到 Pending
            s.taskRepo.Save(ctx, task)
            return errors.New("failed to submit task to worker pool")
        }
    case <-submitCtx.Done():
        // 提交超时，但状态已是 Running，需要回滚
        s.logger.Error("timeout submitting task to worker pool, rolling back task status")
        task.RollbackStart()
        s.taskRepo.Save(ctx, task)
        return errors.New("timeout submitting task to worker pool")
    }

    // 6. 发布领域事件
    s.publishDomainEvents(task)

    return nil
}
```

### 执行任务

```go
// executeTask 执行任务（内部方法）
func (s *TaskApplicationService) executeTask(
    task *domain.Task,
    handler domain.TaskHandler,
    hooks domain.TaskHooks,
) {
    defer s.taskRuntime.Unregister(task.ID())  // 执行完成后注销
    
    // 创建带超时的 context
    ctx, cancel := s.taskRuntime.CreateContext(task.ID(), task.GetTimeout())
    defer cancel()
    
    // 1. 执行前钩子
    if err := s.hookExecutor.ExecuteBeforeExecute(ctx, task, hooks); err != nil {
        s.logger.Error("before execute hook failed", zap.Error(err))
        s.finishTaskWithError(ctx, task, hooks, err)
        return
    }
    
    // 2. 执行任务（带重试）
    err := s.executeWithRetry(ctx, task, handler, hooks)
    
    // 3. 执行后钩子
    s.hookExecutor.ExecuteAfterExecute(ctx, task, hooks, err)
    
    // 4. 处理结果
    if err != nil {
        s.finishTaskWithError(ctx, task, hooks, err)
    } else {
        s.finishTaskWithSuccess(ctx, task, hooks)
    }
}

// executeWithRetry 带重试的执行
func (s *TaskApplicationService) executeWithRetry(
    ctx context.Context,
    task *domain.Task,
    handler domain.TaskHandler,
    hooks domain.TaskHooks,
) error {
    maxRetries := task.MaxRetries()
    if maxRetries < 0 {
        maxRetries = 0
    }
    
    var lastErr error
    for attempt := 0; attempt <= maxRetries; attempt++ {
        if attempt > 0 {
            // 检查 context 是否已取消
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
            }
            
            // 指数退避
            backoff := time.Duration(attempt*attempt) * time.Second
            s.logger.Info("retrying task",
                zap.String("task_id", task.ID().String()),
                zap.Int("attempt", attempt),
                zap.Duration("backoff", backoff),
            )
            
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return ctx.Err()
            }
        }
        
        // 执行
        taskCtx := domain.NewTaskContext(ctx, task)
        err := handler(taskCtx)
        if err == nil {
            return nil  // 成功
        }
        
        lastErr = err
        
        // 只有特定错误才重试
        if !isRetryableError(err) {
            break
        }
    }
    
    return lastErr
}

func isRetryableError(err error) bool {
    if err == nil {
        return false
    }
    // 可重试的错误：超时、临时网络错误等
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }
    // 其他错误不重试
    return false
}
```

### 完成任务

```go
// finishTaskWithSuccess 成功完成任务
func (s *TaskApplicationService) finishTaskWithSuccess(
    ctx context.Context,
    task *domain.Task,
    hooks domain.TaskHooks,
) {
    // 1. 完成前钩子
    if err := s.hookExecutor.ExecuteBeforeFinish(ctx, task, hooks); err != nil {
        s.logger.Error("before finish hook failed", zap.Error(err))
    }
    
    // 2. 标记完成（领域方法）
    result := domain.NewResult(true, nil, "completed successfully")
    if err := task.Complete(result); err != nil {
        s.logger.Error("complete task failed", zap.Error(err))
        return
    }
    
    // 3. 持久化
    if err := s.taskRepo.Save(ctx, task); err != nil {
        s.logger.Error("save task failed", zap.Error(err))
        return
    }
    
    // 4. 完成后钩子
    s.hookExecutor.ExecuteAfterFinish(ctx, task, hooks)
    
    // 5. 发布领域事件
    s.publishDomainEvents(task)
}

// finishTaskWithError 失败完成任务
func (s *TaskApplicationService) finishTaskWithError(
    ctx context.Context,
    task *domain.Task,
    hooks domain.TaskHooks,
    execErr error,
) {
    // 1. 完成前钩子
    if err := s.hookExecutor.ExecuteBeforeFinish(ctx, task, hooks); err != nil {
        s.logger.Error("before finish hook failed", zap.Error(err))
    }
    
    // 2. 标记失败（领域方法）
    if err := task.Fail(execErr); err != nil {
        s.logger.Error("fail task failed", zap.Error(err))
        return
    }
    
    // 3. 持久化
    if err := s.taskRepo.Save(ctx, task); err != nil {
        s.logger.Error("save task failed", zap.Error(err))
        return
    }
    
    // 4. 完成后钩子
    s.hookExecutor.ExecuteAfterFinish(ctx, task, hooks)
    
    // 5. 发布领域事件
    s.publishDomainEvents(task)
}
```

### 取消任务

```go
// CancelTask 取消任务用例
func (s *TaskApplicationService) CancelTask(
    ctx context.Context,
    taskID domain.TaskID,
    reason string,
) error {
    // 1. 获取任务
    task, err := s.taskRepo.FindByID(ctx, taskID)
    if err != nil {
        return err
    }
    
    // 2. 取消运行时（发送 cancel 信号）
    s.taskRuntime.Cancel(taskID)
    
    // 3. 取消任务（领域方法）
    if err := task.Cancel(reason); err != nil {
        return err
    }
    
    // 4. 持久化
    if err := s.taskRepo.Save(ctx, task); err != nil {
        return err
    }
    
    // 5. 发布领域事件
    s.publishDomainEvents(task)
    
    // 6. 级联取消子任务
    children, err := s.taskRepo.FindByParentID(ctx, taskID)
    if err != nil {
        s.logger.Error("find children failed", zap.Error(err))
    }
    for _, child := range children {
        if err := s.CancelTask(ctx, child.ID(), "parent cancelled"); err != nil {
            s.logger.Error("cancel child task failed", zap.Error(err))
        }
    }
    
    return nil
}

// GetTaskTree 获取任务树用例
func (s *TaskApplicationService) GetTaskTree(
    ctx context.Context,
    traceID domain.TraceID,
) (*domain.TaskTree, error) {
    // 获取根任务
    tasks, err := s.taskRepo.FindByTraceID(ctx, traceID)
    if err != nil {
        return nil, err
    }
    
    var rootTask *domain.Task
    for _, task := range tasks {
        if task.IsRoot() {
            rootTask = task
            break
        }
    }
    
    if rootTask == nil {
        return nil, fmt.Errorf("root task not found for trace %s", traceID)
    }
    
    return s.treeBuilder.BuildTree(ctx, rootTask)
}

// publishDomainEvents 发布领域事件
func (s *TaskApplicationService) publishDomainEvents(task *domain.Task) {
    events := task.PullDomainEvents()
    for _, event := range events {
        // 持久化事件
        if s.eventStore != nil {
            if err := s.eventStore.Save(context.Background(), event); err != nil {
                s.logger.Error("save event failed", zap.Error(err))
            }
        }
        // 发布到事件总线
        s.eventBus.Publish(event)
    }
}
```

---

## QueryService (CQRS)

查询服务与命令分离，专门处理查询。

```go
// application/query_service.go
package application

import (
    "context"
    
    "github.com/example/taskmanager/domain"
)

// QueryService 查询服务（CQRS分离）
type QueryService struct {
    taskRepo domain.TaskRepository
}

func NewQueryService(taskRepo domain.TaskRepository) *QueryService {
    return &QueryService{taskRepo: taskRepo}
}

// GetTask 获取任务查询
func (s *QueryService) GetTask(ctx context.Context, id domain.TaskID) (*GetTaskDTO, error) {
    task, err := s.taskRepo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    return s.toDTO(task), nil
}

// ListTasksByTrace 根据TraceID列任务查询
func (s *QueryService) ListTasksByTrace(ctx context.Context, traceID domain.TraceID) ([]*GetTaskDTO, error) {
    tasks, err := s.taskRepo.FindByTraceID(ctx, traceID)
    if err != nil {
        return nil, err
    }
    
    result := make([]*GetTaskDTO, len(tasks))
    for i, task := range tasks {
        result[i] = s.toDTO(task)
    }
    return result, nil
}

// toDTO 转换为DTO
func (s *QueryService) toDTO(task *domain.Task) *GetTaskDTO {
    dto := &GetTaskDTO{
        ID:          task.ID().String(),
        TraceID:     task.TraceID().String(),
        SpanID:      task.SpanID().String(),
        Name:        task.Name(),
        Description: task.Description(),
        Type:        task.Type().String(),
        Status:      task.Status().String(),
        Progress: ProgressDTO{
            Total:      task.Progress().Total(),
            Current:    task.Progress().Current(),
            Percentage: task.Progress().Percentage(),
            Stage:      task.Progress().Stage(),
            Detail:     task.Progress().Detail(),
        },
        CreatedAt: task.CreatedAt().Unix(),
    }
    
    if task.ParentID() != nil {
        parentID := task.ParentID().String()
        dto.ParentID = &parentID
    }
    
    if task.Status().IsFinished() {
        err := task.Error()
        if err != nil {
            dto.Error = err.Error()
        }
        
        result := task.Result()
        if result != nil {
            dto.Result = &ResultDTO{
                Success: result.Success(),
                Data:    result.Data(),
                Message: result.Message(),
            }
        }
    }
    
    if task.StartedAt() != nil {
        t := task.StartedAt().Unix()
        dto.StartedAt = &t
    }
    
    if task.FinishedAt() != nil {
        t := task.FinishedAt().Unix()
        dto.FinishedAt = &t
    }
    
    return dto
}
```

---

## DTO定义

```go
// application/dto.go
package application

import (
    "github.com/example/taskmanager/domain"
)

// CreateTaskCommand 创建任务命令
type CreateTaskCommand struct {
    Name        string
    Description string
    Type        domain.TaskType
    Metadata    map[string]interface{}
    Timeout     int64
    MaxRetries  int
    Priority    int
    ParentID    *domain.TaskID
    Hooks       domain.TaskHooks
}

// GetTaskDTO 任务查询DTO
type GetTaskDTO struct {
    ID          string
    TraceID     string
    SpanID      string
    ParentID    *string
    Name        string
    Description string
    Type        string
    Status      string
    Progress    ProgressDTO
    Result      *ResultDTO
    Error       string
    CreatedAt   int64
    StartedAt   *int64
    FinishedAt  *int64
}

// ProgressDTO 进度DTO
type ProgressDTO struct {
    Total      int64   `json:"total"`
    Current    int64   `json:"current"`
    Percentage float64 `json:"percentage"`
    Stage      string  `json:"stage"`
    Detail     string  `json:"detail"`
}

// ResultDTO 结果DTO
type ResultDTO struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
    Message string      `json:"message"`
}
```

---

## WorkerPool

```go
// application/worker_pool.go
package application

import (
    "context"
    "sync"
)

// WorkerPool 工作池
type WorkerPool struct {
    workers  int
    jobQueue chan func()
    wg       sync.WaitGroup
    
    // 关闭控制
    ctx    context.Context
    cancel context.CancelFunc
    closed chan struct{}
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    pool := &WorkerPool{
        workers:  workers,
        jobQueue: make(chan func(), queueSize),
        ctx:      ctx,
        cancel:   cancel,
        closed:   make(chan struct{}),
    }
    pool.start()
    return pool
}

func (p *WorkerPool) start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go func() {
            defer p.wg.Done()
            for {
                select {
                case job, ok := <-p.jobQueue:
                    if !ok {
                        return  // 队列关闭
                    }
                    // 执行 Job，带 panic 恢复
                    func() {
                        defer func() {
                            if r := recover(); r != nil {
                                // 记录 panic，但不影响其他 worker
                            }
                        }()
                        job()
                    }()
                case <-p.ctx.Done():
                    return  // 强制关闭
                }
            }
        }()
    }
}

// Submit 提交任务，可能阻塞
func (p *WorkerPool) Submit(job func()) {
    select {
    case p.jobQueue <- job:
    case <-p.closed:
        // 已关闭，不执行
    }
}

// TrySubmit 尝试提交任务，非阻塞，返回是否成功
func (p *WorkerPool) TrySubmit(job func()) bool {
    select {
    case p.jobQueue <- job:
        return true
    case <-p.closed:
        return false
    default:
        return false  // 队列满
    }
}

// GracefulStop 优雅关闭，等待所有任务完成
func (p *WorkerPool) GracefulStop() {
    close(p.jobQueue)
    p.wg.Wait()
    close(p.closed)
}

// ForceStop 强制关闭，中断正在执行的任务
func (p *WorkerPool) ForceStop() {
    p.cancel()  // 取消 context
    close(p.jobQueue)
    p.wg.Wait()
    close(p.closed)
}
```
