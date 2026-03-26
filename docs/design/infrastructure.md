# 基础设施层设计 (Infrastructure Layer)

基础设施层提供技术实现，包括仓储实现、消息总线、ID生成器等。

## 目录

- [基础设施层设计 (Infrastructure Layer)](#基础设施层设计-infrastructure-layer)
  - [目录](#目录)
  - [SQLite仓储实现](#sqlite仓储实现)
  - [数据库Schema](#数据库schema)
  - [事件总线实现](#事件总线实现)
  - [ID生成器实现](#id生成器实现)
  - [Hook注册表实现](#hook注册表实现)

---

## SQLite仓储实现

```go
// infrastructure/persistence/task_repository.go
package persistence

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"
    
    "github.com/example/taskmanager/domain"
    _ "modernc.org/sqlite"
)

// SQLiteTaskRepository SQLite任务仓储实现
type SQLiteTaskRepository struct {
    db *sql.DB
}

// NewSQLiteTaskRepository 创建SQLite任务仓储
func NewSQLiteTaskRepository(db *sql.DB) *SQLiteTaskRepository {
    return &SQLiteTaskRepository{db: db}
}

// Save 保存任务
func (r *SQLiteTaskRepository) Save(ctx context.Context, task *domain.Task) error {
    snap := task.ToSnapshot()
    
    metadata, _ := json.Marshal(snap.Metadata)
    progress, _ := json.Marshal(snap.Progress)
    result, _ := json.Marshal(snap.Result)
    
    query := `
        INSERT INTO tasks (id, trace_id, span_id, parent_id, name, description, type,
            metadata, timeout, max_retries, priority, status, progress, result,
            error_msg, created_at, started_at, finished_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            status=excluded.status,
            progress=excluded.progress,
            result=excluded.result,
            error_msg=excluded.error_msg,
            started_at=excluded.started_at,
            finished_at=excluded.finished_at
    `
    
    var parentID interface{}
    if snap.ParentID != nil {
        parentID = snap.ParentID.String()
    }
    
    var startedAt, finishedAt interface{}
    if snap.StartedAt != nil {
        startedAt = snap.StartedAt.Unix()
    }
    if snap.FinishedAt != nil {
        finishedAt = snap.FinishedAt.Unix()
    }
    
    _, err := r.db.ExecContext(ctx, query,
        snap.ID.String(), snap.TraceID.String(), snap.SpanID.String(), parentID,
        snap.Name, snap.Description, snap.Type.String(), metadata,
        snap.Timeout.Milliseconds(), snap.MaxRetries, snap.Priority, int(snap.Status),
        progress, result, snap.ErrorMsg, snap.CreatedAt.Unix(),
        startedAt, finishedAt,
    )
    
    return err
}

// FindByID 根据ID查找任务
func (r *SQLiteTaskRepository) FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
    query := `
        SELECT id, trace_id, span_id, parent_id, name, description, type,
               metadata, timeout, max_retries, priority, status, progress, result,
               error_msg, created_at, started_at, finished_at
        FROM tasks WHERE id = ?`
    
    row := r.db.QueryRowContext(ctx, query, id.String())
    return r.scanToTask(row)
}

// FindByTraceID 根据TraceID查找所有任务
func (r *SQLiteTaskRepository) FindByTraceID(ctx context.Context, traceID domain.TraceID) ([]*domain.Task, error) {
    query := `
        SELECT id, trace_id, span_id, parent_id, name, description, type,
               metadata, timeout, max_retries, priority, status, progress, result,
               error_msg, created_at, started_at, finished_at
        FROM tasks WHERE trace_id = ? ORDER BY created_at`
    
    rows, err := r.db.QueryContext(ctx, query, traceID.String())
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    return r.scanToTasks(rows)
}

// FindByParentID 根据父任务ID查找子任务
func (r *SQLiteTaskRepository) FindByParentID(ctx context.Context, parentID domain.TaskID) ([]*domain.Task, error) {
    query := `
        SELECT id, trace_id, span_id, parent_id, name, description, type,
               metadata, timeout, max_retries, priority, status, progress, result,
               error_msg, created_at, started_at, finished_at
        FROM tasks WHERE parent_id = ?`
    
    rows, err := r.db.QueryContext(ctx, query, parentID.String())
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    return r.scanToTasks(rows)
}

// FindByStatus 根据状态查找任务
func (r *SQLiteTaskRepository) FindByStatus(ctx context.Context, status domain.TaskStatus) ([]*domain.Task, error) {
    query := `
        SELECT id, trace_id, span_id, parent_id, name, description, type,
               metadata, timeout, max_retries, priority, status, progress, result,
               error_msg, created_at, started_at, finished_at
        FROM tasks WHERE status = ?`
    
    rows, err := r.db.QueryContext(ctx, query, int(status))
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    return r.scanToTasks(rows)
}

// FindRunningTasks 查找所有运行中的任务
func (r *SQLiteTaskRepository) FindRunningTasks(ctx context.Context) ([]*domain.Task, error) {
    return r.FindByStatus(ctx, domain.TaskStatusRunning)
}

// Delete 删除任务
func (r *SQLiteTaskRepository) Delete(ctx context.Context, id domain.TaskID) error {
    query := `DELETE FROM tasks WHERE id = ?`
    _, err := r.db.ExecContext(ctx, query, id.String())
    return err
}

// Exists 判断任务是否存在
func (r *SQLiteTaskRepository) Exists(ctx context.Context, id domain.TaskID) (bool, error) {
    query := `SELECT 1 FROM tasks WHERE id = ? LIMIT 1`
    var n int
    err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&n)
    if err == sql.ErrNoRows {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return true, nil
}

// scan helpers
func (r *SQLiteTaskRepository) scanToTask(row *sql.Row) (*domain.Task, error) {
    var snap domain.TaskSnapshot
    var metadataJSON, progressJSON, resultJSON []byte
    var parentIDStr *string
    var typeStr string
    var statusInt int
    var createdAtUnix int64
    var startedAtUnix, finishedAtUnix *int64
    var timeoutMs int64
    
    err := row.Scan(
        &snap.ID, &snap.TraceID, &snap.SpanID, &parentIDStr,
        &snap.Name, &snap.Description, &typeStr, &metadataJSON,
        &timeoutMs, &snap.MaxRetries, &snap.Priority, &statusInt,
        &progressJSON, &resultJSON, &snap.ErrorMsg, &createdAtUnix,
        &startedAtUnix, &finishedAtUnix,
    )
    if err != nil {
        return nil, err
    }
    
    // 反序列化
    json.Unmarshal(metadataJSON, &snap.Metadata)
    json.Unmarshal(progressJSON, &snap.Progress)
    json.Unmarshal(resultJSON, &snap.Result)
    
    snap.Type = domain.TaskType(typeStr)
    snap.Status = domain.TaskStatus(statusInt)
    snap.Timeout = time.Duration(timeoutMs) * time.Millisecond
    
    if parentIDStr != nil {
        id := domain.TaskID(*parentIDStr)
        snap.ParentID = &id
    }
    
    snap.CreatedAt = time.Unix(createdAtUnix, 0)
    if startedAtUnix != nil {
        t := time.Unix(*startedAtUnix, 0)
        snap.StartedAt = &t
    }
    if finishedAtUnix != nil {
        t := time.Unix(*finishedAtUnix, 0)
        snap.FinishedAt = &t
    }
    
    return domain.TaskFromSnapshot(&snap), nil
}

func (r *SQLiteTaskRepository) scanToTasks(rows *sql.Rows) ([]*domain.Task, error) {
    var tasks []*domain.Task
    for rows.Next() {
        var snap domain.TaskSnapshot
        var metadataJSON, progressJSON, resultJSON []byte
        var parentIDStr *string
        var typeStr string
        var statusInt int
        var createdAtUnix int64
        var startedAtUnix, finishedAtUnix *int64
        var timeoutMs int64
        
        err := rows.Scan(
            &snap.ID, &snap.TraceID, &snap.SpanID, &parentIDStr,
            &snap.Name, &snap.Description, &typeStr, &metadataJSON,
            &timeoutMs, &snap.MaxRetries, &snap.Priority, &statusInt,
            &progressJSON, &resultJSON, &snap.ErrorMsg, &createdAtUnix,
            &startedAtUnix, &finishedAtUnix,
        )
        if err != nil {
            return nil, err
        }
        
        json.Unmarshal(metadataJSON, &snap.Metadata)
        json.Unmarshal(progressJSON, &snap.Progress)
        json.Unmarshal(resultJSON, &snap.Result)
        
        snap.Type = domain.TaskType(typeStr)
        snap.Status = domain.TaskStatus(statusInt)
        snap.Timeout = time.Duration(timeoutMs) * time.Millisecond
        
        if parentIDStr != nil {
            id := domain.TaskID(*parentIDStr)
            snap.ParentID = &id
        }
        
        snap.CreatedAt = time.Unix(createdAtUnix, 0)
        if startedAtUnix != nil {
            t := time.Unix(*startedAtUnix, 0)
            snap.StartedAt = &t
        }
        if finishedAtUnix != nil {
            t := time.Unix(*finishedAtUnix, 0)
            snap.FinishedAt = &t
        }
        
        tasks = append(tasks, domain.TaskFromSnapshot(&snap))
    }
    return tasks, rows.Err()
}
```

---

## 数据库Schema

```go
// infrastructure/persistence/schema.go
package persistence

import "database/sql"

// Migrate 执行数据库迁移
func Migrate(db *sql.DB) error {
    // 启用WAL模式
    if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
        return err
    }
    
    // 创建任务表
    tasksTable := `
    CREATE TABLE IF NOT EXISTS tasks (
        id TEXT PRIMARY KEY,
        trace_id TEXT NOT NULL,
        span_id TEXT NOT NULL,
        parent_id TEXT,
        name TEXT NOT NULL,
        description TEXT,
        type TEXT,
        metadata TEXT,          -- JSON
        timeout INTEGER,        -- 毫秒
        max_retries INTEGER DEFAULT 0,
        priority INTEGER DEFAULT 0,
        status INTEGER NOT NULL,
        progress TEXT,          -- JSON
        result TEXT,            -- JSON
        error_msg TEXT,
        created_at INTEGER NOT NULL,
        started_at INTEGER,
        finished_at INTEGER
    );`
    
    if _, err := db.Exec(tasksTable); err != nil {
        return err
    }
    
    // 创建索引
    indexes := []string{
        `CREATE INDEX IF NOT EXISTS idx_tasks_trace_id ON tasks(trace_id);`,
        `CREATE INDEX IF NOT EXISTS idx_tasks_parent_id ON tasks(parent_id);`,
        `CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
        `CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);`,
    }
    
    for _, idx := range indexes {
        if _, err := db.Exec(idx); err != nil {
            return err
        }
    }
    
    // 创建事件表（可选，用于事件持久化）
    eventsTable := `
    CREATE TABLE IF NOT EXISTS events (
        id TEXT PRIMARY KEY,
        type TEXT NOT NULL,
        trace_id TEXT NOT NULL,
        task_id TEXT NOT NULL,
        span_id TEXT NOT NULL,
        timestamp INTEGER NOT NULL,
        payload TEXT           -- JSON
    );`
    
    if _, err := db.Exec(eventsTable); err != nil {
        return err
    }
    
    // 事件表索引
    eventIndexes := []string{
        `CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);`,
        `CREATE INDEX IF NOT EXISTS idx_events_task_id ON events(task_id);`,
        `CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);`,
    }
    
    for _, idx := range eventIndexes {
        if _, err := db.Exec(idx); err != nil {
            return err
        }
    }
    
    return nil
}
```

---

## 事件总线实现

```go
// infrastructure/bus/event_bus.go
package bus

import (
    "fmt"
    "sync"
    
    "github.com/example/taskmanager/domain"
)

// EventBus 内存事件总线实现
type EventBus struct {
    mu       sync.RWMutex
    handlers map[string]map[string]EventHandler  // eventType -> handlerID -> handler
    counter  int64
}

// Subscription 订阅句柄
type Subscription struct {
    ID        string
    EventType string
}

// EventHandler 事件处理器
type EventHandler func(event domain.DomainEvent)

// NewEventBus 创建事件总线
func NewEventBus() *EventBus {
    return &EventBus{
        handlers: make(map[string]map[string]EventHandler),
    }
}

// Subscribe 订阅事件，返回订阅句柄用于取消订阅
func (b *EventBus) Subscribe(eventType string, handler EventHandler) *Subscription {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    b.counter++
    sub := &Subscription{
        ID:        fmt.Sprintf("sub_%d", b.counter),
        EventType: eventType,
    }
    
    if b.handlers[eventType] == nil {
        b.handlers[eventType] = make(map[string]EventHandler)
    }
    b.handlers[eventType][sub.ID] = handler
    
    return sub
}

// Unsubscribe 取消订阅
func (b *EventBus) Unsubscribe(sub *Subscription) {
    if sub == nil {
        return
    }
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if handlers, ok := b.handlers[sub.EventType]; ok {
        delete(handlers, sub.ID)
    }
}

// Publish 发布事件
func (b *EventBus) Publish(event domain.DomainEvent) {
    b.mu.RLock()
    handlers := b.handlers[event.EventType()]
    b.mu.RUnlock()
    
    for _, handler := range handlers {
        go func(h EventHandler) {
            defer func() {
                if r := recover(); r != nil {
                    // 记录 panic，但不影响其他 handler
                    // 实际应使用 logger
                    log.Printf("Event handler panic: %v", r)
                }
            }()
            h(event)
        }(handler)
    }
}
```

---

## ID生成器实现

```go
// infrastructure/utils/id_generator.go
package utils

import (
    "github.com/aidarkhanov/nanoid/v2"
    "github.com/example/taskmanager/domain"
)

// NanoIDGenerator nanoID生成器
type NanoIDGenerator struct {
    alphabet string
    size     int
}

// NewNanoIDGenerator 创建nanoID生成器
func NewNanoIDGenerator() *NanoIDGenerator {
    return &NanoIDGenerator{
        alphabet: "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
        size:     21,
    }
}

// Generate 生成唯一ID
func (g *NanoIDGenerator) Generate() string {
    id, _ := nanoid.Generate(g.alphabet, g.size)
    return id
}

// 编译时接口检查
var _ domain.IDGenerator = (*NanoIDGenerator)(nil)
```

---

## Hook注册表实现

```go
// infrastructure/hook/registry.go
package hook

import (
    "sync"
    
    "github.com/example/taskmanager/domain"
)

// Registry Hook注册表实现
type Registry struct {
    mu          sync.RWMutex
    globalHooks map[domain.HookPoint][]domain.HookFunc
    typeHooks   map[domain.TaskType]domain.TaskHooks
}

// NewRegistry 创建Hook注册表
func NewRegistry() *Registry {
    return &Registry{
        globalHooks: make(map[domain.HookPoint][]domain.HookFunc),
        typeHooks:   make(map[domain.TaskType]domain.TaskHooks),
    }
}

// RegisterGlobal 注册全局钩子
func (r *Registry) RegisterGlobal(point domain.HookPoint, hook domain.HookFunc) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.globalHooks[point] = append(r.globalHooks[point], hook)
}

// RegisterForType 注册任务类型钩子
func (r *Registry) RegisterForType(taskType domain.TaskType, hooks domain.TaskHooks) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.typeHooks[taskType] = hooks
}

// GetGlobalHooks 获取全局钩子
func (r *Registry) GetGlobalHooks(point domain.HookPoint) []domain.HookFunc {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.globalHooks[point]
}

// GetTypeHooks 获取任务类型钩子
func (r *Registry) GetTypeHooks(taskType domain.TaskType) domain.TaskHooks {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.typeHooks[taskType]
}

// 编译时接口检查
var _ domain.HookRegistry = (*Registry)(nil)
```
