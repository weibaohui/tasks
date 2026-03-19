# 领域层设计 (Domain Layer)

领域层是核心业务逻辑所在，不依赖其他层，只依赖Go标准库。

## 目录

- [实体：Task聚合根](#实体task聚合根)
- [值对象](#值对象)
- [领域事件](#领域事件)
- [仓储接口](#仓储接口)
- [领域服务](#领域服务)
- [Hook系统](#hook系统)

---

## 实体：Task聚合根

```go
// domain/task.go
package domain

import (
    "context"
    "errors"
    "sync"
    "time"
)

// 领域错误定义
var (
    ErrInvalidStatusTransition = errors.New("invalid status transition")
    ErrTaskAlreadyStarted      = errors.New("task already started")
    ErrTaskNotRunning          = errors.New("task is not running")
    ErrTaskAlreadyFinished     = errors.New("task already finished")
    ErrTimeoutNotPositive      = errors.New("timeout must be positive")
)

// Task 任务聚合根
type Task struct {
    // 标识 (实体的唯一标识)
    id       TaskID
    traceID  TraceID
    spanID   SpanID
    parentID *TaskID  // 可选
    
    // 基本信息
    name        string
    description string
    taskType    TaskType
    metadata    map[string]interface{}
    
    // 执行配置
    timeout    time.Duration
    maxRetries int
    priority   int
    
    // 执行状态 (可变状态)
    status   TaskStatus
    progress Progress
    result   *Result
    execErr  error
    
    // 注意：context 不在领域层管理，由应用层的 TaskRuntime 管理
    
    // 时间戳
    createdAt  time.Time
    startedAt  *time.Time
    finishedAt *time.Time
    
    // 领域事件 (待发布)
    domainEvents []DomainEvent
    
    // 并发保护
    mu sync.RWMutex
}
```

### 工厂方法

```go
// NewTask 工厂方法：创建任务
func NewTask(
    id TaskID,
    traceID TraceID,
    spanID SpanID,
    parentID *TaskID,
    name string,
    description string,
    taskType TaskType,
    metadata map[string]interface{},
    timeout time.Duration,
    maxRetries int,
    priority int,
) (*Task, error) {
    if name == "" {
        return nil, errors.New("task name is required")
    }
    if timeout < 0 {
        return nil, ErrTimeoutNotPositive
    }
    
    task := &Task{
        id:          id,
        traceID:     traceID,
        spanID:      spanID,
        parentID:    parentID,
        name:        name,
        description: description,
        taskType:    taskType,
        metadata:    metadata,
        timeout:     timeout,
        maxRetries:  maxRetries,
        priority:    priority,
        status:      TaskStatusPending,
        progress:    NewProgress(),
        createdAt:   time.Now(),
    }
    
    // 记录领域事件
    task.recordEvent(NewTaskCreatedEvent(task))
    
    return task, nil
}
```

### 属性访问方法

```go
// 实体标识获取方法
func (t *Task) ID() TaskID           { return t.id }
func (t *Task) TraceID() TraceID     { return t.traceID }
func (t *Task) SpanID() SpanID       { return t.spanID }
func (t *Task) ParentID() *TaskID    { return t.parentID }
func (t *Task) Name() string         { return t.name }
func (t *Task) Description() string  { return t.description }
func (t *Task) Type() TaskType       { return t.taskType }
func (t *Task) Metadata() map[string]interface{} { return t.metadata }
func (t *Task) Timeout() time.Duration { return t.timeout }
func (t *Task) MaxRetries() int      { return t.maxRetries }
func (t *Task) Priority() int        { return t.priority }
func (t *Task) Status() TaskStatus   { 
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.status 
}
func (t *Task) Progress() Progress   { 
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.progress 
}
func (t *Task) Result() *Result      { 
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.result 
}
func (t *Task) Error() error         { 
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.execErr 
}
func (t *Task) CreatedAt() time.Time { return t.createdAt }
func (t *Task) StartedAt() *time.Time { 
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.startedAt 
}
func (t *Task) FinishedAt() *time.Time { 
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.finishedAt 
}
```

### 领域方法（业务行为）

```go
// Start 开始任务
func (t *Task) Start() error {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if !t.canTransitionTo(TaskStatusRunning) {
        return ErrInvalidStatusTransition
    }
    
    now := time.Now()
    t.status = TaskStatusRunning
    t.startedAt = &now
    
    // 记录领域事件
    t.recordEvent(NewTaskStartedEvent(t))
    
    return nil
}

// GetTimeout 获取超时时间
func (t *Task) GetTimeout() time.Duration {
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.timeout
}

// UpdateProgress 更新进度
func (t *Task) UpdateProgress(current int64, stage, detail string) error {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if t.status != TaskStatusRunning {
        return ErrTaskNotRunning
    }
    
    oldProgress := t.progress
    t.progress = t.progress.Update(current, stage, detail)
    
    // 进度有变化时记录事件
    if oldProgress.Percentage() != t.progress.Percentage() {
        t.recordEvent(NewTaskProgressUpdatedEvent(t, t.progress))
    }
    
    return nil
}

// Complete 完成任务
func (t *Task) Complete(result *Result) error {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if !t.canTransitionTo(TaskStatusCompleted) {
        return ErrInvalidStatusTransition
    }
    
    now := time.Now()
    t.status = TaskStatusCompleted
    t.result = result
    t.finishedAt = &now
    
    if t.cancelFunc != nil {
        t.cancelFunc()
    }
    
    t.recordEvent(NewTaskCompletedEvent(t))
    
    return nil
}

// Fail 标记失败
func (t *Task) Fail(err error) error {
    t.mu.Lock()
    defer t.mu.Unlock()

    if !t.canTransitionTo(TaskStatusFailed) {
        return ErrInvalidStatusTransition
    }

    now := time.Now()
    t.status = TaskStatusFailed
    t.execErr = err
    t.finishedAt = &now

    t.recordEvent(NewTaskFailedEvent(t, err))

    return nil
}

// Cancel 取消任务
func (t *Task) Cancel(reason string) error {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if t.isFinished() {
        return ErrTaskAlreadyFinished
    }
    
    now := time.Now()
    t.status = TaskStatusCancelled
    t.execErr = errors.New(reason)
    t.finishedAt = &now
    
    t.recordEvent(NewTaskCancelledEvent(t, reason))
    
    return nil
}

// IsRoot 是否是根任务
func (t *Task) IsRoot() bool {
    return t.parentID == nil
}

// IsFinished 是否已完成（终态）
func (t *Task) IsFinished() bool {
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.isFinished()
}

func (t *Task) isFinished() bool {
    return t.status == TaskStatusCompleted ||
           t.status == TaskStatusFailed ||
           t.status == TaskStatusCancelled
}

// 状态机规则
func (t *Task) canTransitionTo(newStatus TaskStatus) bool {
    transitions := map[TaskStatus][]TaskStatus{
        TaskStatusPending:   {TaskStatusRunning},
        TaskStatusRunning:   {TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled, TaskStatusStopping},
        TaskStatusStopping:  {TaskStatusCancelled, TaskStatusFailed},
    }
    
    allowed, ok := transitions[t.status]
    if !ok {
        return false
    }
    
    for _, s := range allowed {
        if s == newStatus {
            return true
        }
    }
    return false
}
```

### 领域事件相关

```go
// recordEvent 记录领域事件
func (t *Task) recordEvent(event DomainEvent) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.domainEvents = append(t.domainEvents, event)
}

// PullDomainEvents 拉取并清空领域事件 (由应用层调用)
func (t *Task) PullDomainEvents() []DomainEvent {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    events := t.domainEvents
    t.domainEvents = nil
    return events
}
```

### 快照（用于持久化）

```go
// ToSnapshot 转换为快照
func (t *Task) ToSnapshot() *TaskSnapshot {
    t.mu.RLock()
    defer t.mu.RUnlock()
    
    return &TaskSnapshot{
        ID:          t.id,
        TraceID:     t.traceID,
        SpanID:      t.spanID,
        ParentID:    t.parentID,
        Name:        t.name,
        Description: t.description,
        Type:        t.taskType,
        Metadata:    t.metadata,
        Timeout:     t.timeout,
        MaxRetries:  t.maxRetries,
        Priority:    t.priority,
        Status:      t.status,
        Progress:    t.progress,
        Result:      t.result,
        ErrorMsg:    errorToString(t.execErr),
        CreatedAt:   t.createdAt,
        StartedAt:   t.startedAt,
        FinishedAt:  t.finishedAt,
    }
}

// TaskSnapshot 任务快照
type TaskSnapshot struct {
    ID          TaskID
    TraceID     TraceID
    SpanID      SpanID
    ParentID    *TaskID
    Name        string
    Description string
    Type        TaskType
    Metadata    map[string]interface{}
    Timeout     time.Duration
    MaxRetries  int
    Priority    int
    Status      TaskStatus
    Progress    Progress
    Result      *Result
    ErrorMsg    string
    CreatedAt   time.Time
    StartedAt   *time.Time
    FinishedAt  *time.Time
}

// TaskFromSnapshot 从快照重建实体
// 注意：重建的任务不包含 context，需要应用层通过 TaskRuntime 管理运行时上下文
func TaskFromSnapshot(snap *TaskSnapshot) *Task {
    task := &Task{
        id:          snap.ID,
        traceID:     snap.TraceID,
        spanID:      snap.SpanID,
        parentID:    snap.ParentID,
        name:        snap.Name,
        description: snap.Description,
        taskType:    snap.Type,
        metadata:    snap.Metadata,
        timeout:     snap.Timeout,
        maxRetries:  snap.MaxRetries,
        priority:    snap.Priority,
        status:      snap.Status,
        progress:    snap.Progress,
        result:      snap.Result,
        execErr:     stringToError(snap.ErrorMsg),
        createdAt:   snap.CreatedAt,
        startedAt:   snap.StartedAt,
        finishedAt:  snap.FinishedAt,
        // 注意：不恢复 context 和 cancelFunc，由应用层管理
    }
    return task
}
```

---

## 值对象

```go
// domain/value_object.go
package domain

import (
    "time"
)

// TaskID 任务ID值对象
type TaskID string
func (id TaskID) String() string { return string(id) }

// TraceID 链路追踪ID值对象
type TraceID string
func (id TraceID) String() string { return string(id) }

// SpanID 跨度ID值对象
type SpanID string
func (id SpanID) String() string { return string(id) }

// TaskType 任务类型值对象
type TaskType string
func (t TaskType) String() string { return string(t) }

// TaskStatus 任务状态值对象
type TaskStatus int

const (
    TaskStatusPending    TaskStatus = iota // 等待中
    TaskStatusRunning                      // 执行中
    TaskStatusStopping                     // 停止中
    TaskStatusCompleted                    // 已完成
    TaskStatusFailed                       // 失败
    TaskStatusCancelled                    // 已取消
)

func (s TaskStatus) String() string {
    names := []string{"Pending", "Running", "Stopping", "Completed", "Failed", "Cancelled"}
    if int(s) < len(names) {
        return names[s]
    }
    return "Unknown"
}

func (s TaskStatus) IsFinished() bool {
    return s == TaskStatusCompleted || s == TaskStatusFailed || s == TaskStatusCancelled
}

// Progress 进度值对象 (不可变)
type Progress struct {
    total      int64
    current    int64
    percentage float64
    stage      string
    detail     string
    updatedAt  time.Time
}

// NewProgress 创建新进度（无总量限制）
func NewProgress() Progress {
    return Progress{ updatedAt: time.Now() }
}

// NewProgressWithTotal 创建带总量的进度
func NewProgressWithTotal(total int64) Progress {
    if total < 0 {
        total = 0
    }
    return Progress{
        total:     total,
        updatedAt: time.Now(),
    }
}

// Update 返回新的进度值对象
func (p Progress) Update(current int64, stage, detail string) Progress {
    // 计算百分比：优先使用 total，若无 total 但有 current 则用 current 作参考
    percentage := float64(0)
    if p.total > 0 {
        percentage = float64(current) / float64(p.total) * 100
        if percentage > 100 {
            percentage = 100
        }
    } else if current > 0 {
        // 没有设置 total，但至少有进度数据，使用 current 作为进度参考值
        percentage = float64(current)
    }

    // 限制在 0-100 之间
    if percentage > 100 {
        percentage = 100
    }
    if percentage < 0 {
        percentage = 0
    }

    newStage, newDetail := p.stage, p.detail
    if stage != "" { newStage = stage }
    if detail != "" { newDetail = detail }

    return Progress{
        total:      p.total,
        current:    current,
        percentage: percentage,
        stage:      newStage,
        detail:     newDetail,
        updatedAt:  time.Now(),
    }
}

func (p Progress) Total() int64       { return p.total }
func (p Progress) Current() int64     { return p.current }
func (p Progress) Percentage() float64 { return p.percentage }
func (p Progress) Stage() string      { return p.stage }
func (p Progress) Detail() string     { return p.detail }
func (p Progress) UpdatedAt() time.Time { return p.updatedAt }

// Result 结果值对象
type Result struct {
    success  bool
    data     interface{}
    message  string
    metadata map[string]interface{}
    output   string
}

func NewResult(success bool, data interface{}, message string) Result {
    return Result{ success: success, data: data, message: message }
}

func (r Result) Success() bool { return r.success }
func (r Result) Data() interface{} { return r.data }
func (r Result) Message() string { return r.message }
func (r Result) WithMetadata(m map[string]interface{}) Result {
    r.metadata = m
    return r
}
```

---

## 领域事件

```go
// domain/event.go
package domain

import (
    "time"
    "github.com/aidarkhanov/nanoid/v2"
)

// DomainEvent 领域事件接口
type DomainEvent interface {
    EventID() string
    EventType() string
    OccurredAt() time.Time
    AggregateID() string
    AggregateType() string
}

// BaseDomainEvent 基础实现
type BaseDomainEvent struct {
    eventID       string
    eventType     string
    occurredAt    time.Time
    aggregateID   string
    aggregateType string
}

func (e BaseDomainEvent) EventID() string       { return e.eventID }
func (e BaseDomainEvent) EventType() string     { return e.eventType }
func (e BaseDomainEvent) OccurredAt() time.Time { return e.occurredAt }
func (e BaseDomainEvent) AggregateID() string   { return e.aggregateID }
func (e BaseDomainEvent) AggregateType() string { return e.aggregateType }

// 各具体事件类型...

// TaskCreatedEvent 任务创建事件
type TaskCreatedEvent struct {
    BaseDomainEvent
    TraceID  TraceID
    SpanID   SpanID
    ParentID *TaskID
    Name     string
    Type     TaskType
}

// TaskStartedEvent 任务开始事件
type TaskStartedEvent struct {
    BaseDomainEvent
    TraceID TraceID
    SpanID  SpanID
}

// TaskProgressUpdatedEvent 进度更新事件
type TaskProgressUpdatedEvent struct {
    BaseDomainEvent
    TraceID  TraceID
    SpanID   SpanID
    Progress Progress
}

// TaskCompletedEvent 任务完成事件
type TaskCompletedEvent struct {
    BaseDomainEvent
    TraceID TraceID
    SpanID  SpanID
    Result  *Result
}

// TaskFailedEvent 任务失败事件
type TaskFailedEvent struct {
    BaseDomainEvent
    TraceID TraceID
    SpanID  SpanID
    Error   string
}

// TaskCancelledEvent 任务取消事件
type TaskCancelledEvent struct {
    BaseDomainEvent
    TraceID TraceID
    SpanID  SpanID
    Reason  string
}
```

---

## 仓储接口

```go
// domain/repository.go
package domain

import "context"

// TaskRepository 任务仓储接口
type TaskRepository interface {
    Save(ctx context.Context, task *Task) error
    FindByID(ctx context.Context, id TaskID) (*Task, error)
    FindByTraceID(ctx context.Context, traceID TraceID) ([]*Task, error)
    FindByParentID(ctx context.Context, parentID TaskID) ([]*Task, error)
    FindByStatus(ctx context.Context, status TaskStatus) ([]*Task, error)
    FindRunningTasks(ctx context.Context) ([]*Task, error)
    Delete(ctx context.Context, id TaskID) error
    Exists(ctx context.Context, id TaskID) (bool, error)
}

// EventStore 事件存储接口
type EventStore interface {
    Save(ctx context.Context, event DomainEvent) error
    FindByAggregateID(ctx context.Context, aggregateID string) ([]DomainEvent, error)
    FindByTraceID(ctx context.Context, traceID TraceID) ([]DomainEvent, error)
}

// IDGenerator ID生成器接口
type IDGenerator interface {
    Generate() string
}
```

---

## 领域服务

```go
// domain/service.go
package domain

import (
    "context"
)

// TaskTreeBuilder 任务树构建服务
type TaskTreeBuilder struct {
    taskRepo TaskRepository
}

func NewTaskTreeBuilder(taskRepo TaskRepository) *TaskTreeBuilder {
    return &TaskTreeBuilder{taskRepo: taskRepo}
}

// BuildTree 构建任务树
func (b *TaskTreeBuilder) BuildTree(ctx context.Context, rootTask *Task) (*TaskTree, error) {
    tree := &TaskTree{
        Root:    nil,
        TraceID: rootTask.TraceID(),
        Total:   0,
    }
    
    tasks, err := b.taskRepo.FindByTraceID(ctx, rootTask.TraceID())
    if err != nil {
        return nil, err
    }
    
    // 构建树结构...
    
    return tree, nil
}

// TaskTree 任务树值对象
type TaskTree struct {
    Root     *TaskNode
    TraceID  TraceID
    Total    int
    Complete int
}

// TaskNode 任务树节点
type TaskNode struct {
    Task     *Task
    Children []*TaskNode
    Depth    int
}

// TaskExecutor 任务执行领域服务
type TaskExecutor struct {
    hookExecutor *HookExecutor
}

func NewTaskExecutor(hookExecutor *HookExecutor) *TaskExecutor {
    return &TaskExecutor{hookExecutor: hookExecutor}
}

// Execute 执行任务
func (e *TaskExecutor) Execute(
    ctx context.Context,
    task *Task,
    handler TaskHandler,
    hooks TaskHooks,
) error {
    // 1. 执行前钩子
    if err := e.hookExecutor.ExecuteBeforeExecute(ctx, task, hooks); err != nil {
        return err
    }
    
    // 2. 执行任务
    taskCtx := NewTaskContext(ctx, task)
    err := handler(taskCtx)
    
    // 3. 执行后钩子
    e.hookExecutor.ExecuteAfterExecute(ctx, task, hooks, err)
    
    return err
}

// TaskHandler 任务处理器函数
type TaskHandler func(ctx *TaskContext) error

// TaskContext 任务执行上下文
type TaskContext struct {
    context.Context
    task *Task
}

func NewTaskContext(ctx context.Context, task *Task) *TaskContext {
    return &TaskContext{Context: ctx, task: task}
}

func (c *TaskContext) Task() *Task { return c.task }

func (c *TaskContext) ReportProgress(current int64, stage, detail string) error {
    return c.task.UpdateProgress(current, stage, detail)
}
```

---

## Hook系统

```go
// domain/hook.go
package domain

import "context"

// HookPoint 钩子触发点
type HookPoint int

const (
    HookBeforeCreate HookPoint = iota
    HookAfterCreate
    HookBeforeExecute
    HookAfterExecute
    HookBeforeFinish
    HookAfterFinish
)

// HookFunc 钩子函数
type HookFunc func(ctx context.Context, task *Task) error

// TaskHooks 任务钩子集合
type TaskHooks struct {
    BeforeCreate  []HookFunc
    AfterCreate   []HookFunc
    BeforeExecute []HookFunc
    AfterExecute  []HookFunc
    BeforeFinish  []HookFunc
    AfterFinish   []HookFunc
}

var EmptyHooks = TaskHooks{}

// HookRegistry 钩子注册表接口
type HookRegistry interface {
    RegisterGlobal(point HookPoint, hook HookFunc)
    RegisterForType(taskType TaskType, hooks TaskHooks)
    GetGlobalHooks(point HookPoint) []HookFunc
    GetTypeHooks(taskType TaskType) TaskHooks
}
```
