# 项目目录结构

## 完整目录结构

```
taskmanager/
├── cmd/                          # 应用程序入口
│   └── server/                   # 服务端启动入口
│       └── main.go               # main函数
│
├── internal/                     # 内部实现
│   └── config/                   # 配置管理
│       └── config.go
│
├── domain/                       # 领域层 (核心业务)
│   ├── entity.go                 # 实体基础接口
│   ├── task.go                   # Task聚合根
│   ├── value_object.go           # 值对象
│   │                             #   - TaskID, TraceID, SpanID
│   │                             #   - TaskStatus, TaskType
│   │                             #   - Progress, Result
│   ├── event.go                  # 领域事件
│   ├── repository.go             # 仓储接口
│   ├── service.go                # 领域服务
│   └── hook.go                   # Hook系统接口
│
├── application/                  # 应用层 (用例编排)
│   ├── task_service.go           # TaskApplicationService
│   ├── query_service.go          # QueryService (CQRS)
│   ├── dto.go                    # DTO定义
│   └── worker_pool.go            # WorkerPool
│
├── infrastructure/               # 基础设施层
│   ├── persistence/              # 持久化实现
│   │   ├── sqlite_repository.go  # SQLite仓储实现
│   │   ├── event_store.go        # 事件存储实现
│   │   └── schema.go             # 数据库Schema
│   │
│   ├── bus/                      # 消息总线
│   │   └── event_bus.go          # 内存事件总线
│   │
│   ├── hook/                     # Hook实现
│   │   └── registry.go           # Hook注册表
│   │
│   └── utils/                    # 工具类
│       └── id_generator.go       # nanoID生成器
│
├── interfaces/                   # 接口层 (适配器)
│   ├── http/                     # HTTP API
│   │   ├── handler.go            # HTTP处理器
│   │   └── router.go             # 路由定义
│   │
│   └── ws/                       # WebSocket
│       └── handler.go            # WebSocket处理器
│
├── go.mod                        # Go模块定义
├── go.sum                        # Go模块校验
└── README.md                     # 项目说明
```

---

## go.mod 依赖

```go
module github.com/example/taskmanager

go 1.21

require (
    github.com/aidarkhanov/nanoid/v2 v2.0.5
    go.uber.org/zap v1.26.0
    modernc.org/sqlite v1.28.0
)
```

---

## 分层依赖关系

```
interfaces/
    ↓ (依赖)
application/
    ↓ (依赖)
domain/              ← 核心，不依赖其他层
    ↑ (由...实现)
infrastructure/
```

**关键原则：**
- 领域层不依赖任何其他层
- 领域层定义接口（仓储、HookRegistry）
- 基础设施层实现领域层定义的接口
- 依赖倒置：高层模块不依赖低层模块，都依赖抽象

---

## 各层文件职责

### Domain Layer (domain/)

| 文件 | 职责 |
|------|------|
| task.go | Task聚合根实体，包含业务规则、状态机、领域方法 |
| value_object.go | 值对象定义：ID、Status、Progress、Result等 |
| event.go | 领域事件：TaskCreatedEvent、TaskProgressUpdatedEvent等 |
| repository.go | 仓储接口定义：TaskRepository、EventStore |
| service.go | 领域服务：TaskTreeBuilder、TaskExecutor |
| hook.go | Hook系统接口：HookRegistry、TaskHooks |

### Application Layer (application/)

| 文件 | 职责 |
|------|------|
| task_service.go | TaskApplicationService，编排创建、启动、取消用例 |
| query_service.go | QueryService，处理查询（CQRS） |
| dto.go | DTO定义：CreateTaskCommand、GetTaskDTO等 |
| worker_pool.go | WorkerPool，goroutine池管理 |

### Infrastructure Layer (infrastructure/)

| 文件 | 职责 |
|------|------|
| persistence/sqlite_repository.go | TaskRepository的SQLite实现 |
| persistence/event_store.go | EventStore的SQLite实现 |
| persistence/schema.go | 数据库表结构和迁移 |
| bus/event_bus.go | 内存事件总线实现 |
| hook/registry.go | HookRegistry实现 |
| utils/id_generator.go | nanoID生成器 |

### Interface Layer (interfaces/)

| 文件 | 职责 |
|------|------|
| http/handler.go | HTTP请求处理器 |
| http/router.go | 路由注册 |
| ws/handler.go | WebSocket连接处理 |

---

## 代码示例

### 领域层示例 (domain/task.go)

```go
package domain

// Task 聚合根
type Task struct {
    id TaskID
    // ... 其他字段
}

// 领域方法
func (t *Task) Start() error {
    // 业务规则验证
    if !t.canTransitionTo(TaskStatusRunning) {
        return ErrInvalidStatusTransition
    }
    // ... 执行逻辑
    t.recordEvent(NewTaskStartedEvent(t))
    return nil
}
```

### 应用层示例 (application/task_service.go)

```go
package application

func (s *TaskApplicationService) CreateTask(ctx context.Context, cmd CreateTaskCommand) (*domain.Task, error) {
    // 1. 生成ID
    taskID := domain.TaskID(s.idGenerator.Generate())
    
    // 2. 创建领域实体
    task, err := domain.NewTask(taskID, ...)
    
    // 3. 执行前钩子
    s.hookExecutor.ExecuteBeforeCreate(ctx, task, cmd.Hooks)
    
    // 4. 持久化
    s.taskRepo.Save(ctx, task)
    
    // 5. 发布领域事件
    s.publishDomainEvents(task)
    
    return task, nil
}
```

### 基础设施层示例 (infrastructure/persistence/sqlite_repository.go)

```go
package persistence

type SQLiteTaskRepository struct {
    db *sql.DB
}

func (r *SQLiteTaskRepository) Save(ctx context.Context, task *domain.Task) error {
    snap := task.ToSnapshot()
    // SQL执行...
}

// 编译时检查接口实现
var _ domain.TaskRepository = (*SQLiteTaskRepository)(nil)
```

---

## 扩展指南

### 添加新的存储实现

1. 创建新的仓储文件：`infrastructure/persistence/mysql_repository.go`
2. 实现 `domain.TaskRepository` 接口
3. 在 `main.go` 中替换使用

### 添加新的接口适配器

1. 创建适配器目录：`interfaces/grpc/`
2. 实现gRPC服务
3. 在 `main.go` 中注册

### 添加新的领域事件

1. 在 `domain/event.go` 中定义新的事件类型
2. 在 `Task` 聚合根中添加触发点
3. 在 `eventBus` 中订阅处理
