# DDD 最佳实践约束

## ⚠️ AI 必须严格遵守的规则

本项目采用 DDD（Domain-Driven Design）架构。以下规则是 **强制约束**，任何代码变更必须遵守。

---

## 1. 核心原则

### 1.1 严格分层
```
interfaces/http    →  接收请求，调用 application service
application       →  编排领域对象，处理用例
domain           →  纯业务逻辑，无依赖
infrastructure    →  实现接口（repository、hook、external）
```

**绝对禁止**：
- `infrastructure` 引用 `interfaces`
- `domain` 引用任何其他层
- `application` 直接操作数据库

### 1.2 聚合根（Aggregate Root）

每个聚合有且只有一个 **聚合根**，外部对象只能通过聚合根访问内部实体。

**聚合根职责**：
- 创建时验证内部不变量
- 控制内部状态变更
- 对外暴露唯一入口

**项目中的聚合**：
| 聚合 | 聚合根 | 内部实体 |
|------|--------|---------|
| Task | Task | - |
| Agent | Agent | - |
| User | User | - |
| ConversationRecord | ConversationRecord | - |
| Provider | LLMProvider | ProviderModel |
| Channel | Channel | - |
| MCPServer | MCPServer | MCPTool, AgentMCPBinding |

### 1.3 值对象（Value Object）

值对象是 **不可变** 的，只有属性没有身份。

**规则**：
- 值对象必须是通过构造函数创建
- 所有字段是 `private` 或 `const`
- 禁止提供 setter
- 相同属性值 = 相同对象（用 `Equals` 判断）

**示例**：
```go
// ✅ 正确：值对象不可变
type UserCode struct {
    value string
}
func NewUserCode(code string) (*UserCode, error) {
    if code == "" {
        return nil, ErrUserCodeRequired
    }
    return &UserCode{value: code}, nil
}

// ❌ 错误：可变对象不是值对象
type UserCode struct {
    Code string // 可修改
}
```

---

## 2. 领域模型规则

### 2.1 实体（Entity）

实体有 **身份标识**，通过 ID 判断相等性。

```go
// ✅ 正确：实体有 ID
type Agent struct {
    id        AgentID
    userCode  string
    name      string
    ...
}

// ❌ 错误：缺少身份标识
type Agent struct {
    userCode  string
    name      string
    ...
}
```

### 2.2 工厂方法（Factory）

聚合的创建必须通过 **工厂方法**，确保创建时验证不变量。

```go
// ✅ 正确：工厂方法验证
func NewAgent(id AgentID, userCode string, name string) (*Agent, error) {
    if userCode == "" {
        return nil, ErrUserCodeRequired
    }
    if name == "" {
        return nil, ErrAgentNameRequired
    }
    return &Agent{id: id, userCode: userCode, name: name}, nil
}

// ❌ 错误：允许创建无效对象
type Agent struct {...}
agent := &Agent{} // 没有验证
```

### 2.3 领域服务（Domain Service）

当操作不属于任何实体时，使用 **领域服务**。

**规则**：
- 领域服务是 **无状态** 的
- 只处理跨多个聚合的业务逻辑
- 不要把领域服务变成事务脚本

---

## 3. 仓储模式（Repository）

### 3.1 仓储接口定义位置

**仓储接口必须在 `domain` 层定义**，实现放在 `infrastructure/persistence`。

```go
// ✅ 正确：接口在 domain
// domain/agent_repository.go
type AgentRepository interface {
    FindByID(ctx context.Context, id AgentID) (*Agent, error)
    Save(ctx context.Context, agent *Agent) error
    Delete(ctx context.Context, id AgentID) error
}

// ❌ 错误：接口在 infrastructure
// infrastructure/persistence/agent_repository.go
type AgentRepository interface {...} // 禁止
```

### 3.2 仓储方法命名

| 操作 | 方法名 |
|------|--------|
| 按 ID 查询 | `FindByID(ctx, id)` |
| 按条件查询 | `FindBy(ctx, spec)` |
| 保存（含新增/更新） | `Save(ctx, entity)` |
| 新增 | `Create(ctx, entity)` 或 `Save` |
| 更新 | `Update(ctx, entity)` 或 `Save` |
| 删除 | `Delete(ctx, id)` |
| 列表查询 | `List(ctx, filter)` 或 `FindAll(ctx)` |

### 3.3 查询与命令分离

- **CUD 操作**：通过 Repository
- **查询操作**：
  - 简单查询：Repository 方法
  - 复杂查询：使用 `Specification` 模式

---

## 4. 应用服务（Application Service）

### 4.1 应用服务职责

- 编排领域对象和仓储
- 处理 **用例**（use case）
- 处理 **事务边界**
- **不包含业务逻辑**

```go
// ✅ 正确：应用服务编排
func (s *AgentApplicationService) CreateAgent(ctx context.Context, cmd CreateAgentCommand) (*Agent, error) {
    // 1. 验证
    if cmd.UserCode == "" {
        return nil, ErrUserCodeRequired
    }

    // 2. 创建（通过工厂）
    agent, err := domain.NewAgent(...)
    if err != nil {
        return nil, err
    }

    // 3. 保存
    if err := s.agentRepo.Save(ctx, agent); err != nil {
        return nil, err
    }

    return agent, nil
}

// ❌ 错误：应用服务包含业务逻辑
func (s *AgentService) CreateAgent(...) {
    // 计算折扣？不应该在这
    price := price * 0.9
    // ...
}
```

### 4.2 命令对象（Command）

变更操作使用 **Command 对象**封装。

```go
type CreateAgentCommand struct {
    UserCode string
    Name     string
    Model    string
    // ...
}

type UpdateAgentCommand struct {
    ID           string
    Name         *string  // 可选字段用指针
    Model        *string
    // ...
}
```

---

## 5. 领域事件（Domain Event）

### 5.1 事件命名

```
<聚合名><事件名>Event

例如：
- AgentCreatedEvent
- TaskCompletedEvent
- UserActivatedEvent
```

### 5.2 事件内容

```go
type AgentCreatedEvent struct {
    ID        string
    UserCode  string
    Name      string
    Timestamp time.Time
}
```

### 5.3 事件发布

- 使用 **事件总线** 或 **发布-订阅**
- 事件是 **不可变的**
- 事件处理器在 `infrastructure` 层

---

## 6. 限界上下文（Bounded Context）

### 6.1 明确边界

项目划分的限界上下文：

| 上下文 | 核心域 | 职责 |
|--------|--------|------|
| Task | Task Aggregate | 任务管理 |
| Agent | Agent Aggregate | Agent 配置 |
| User | User Aggregate | 用户管理 |
| Conversation | ConversationRecord | 对话记录 |
| Channel | Channel Aggregate | 渠道管理 |
| LLM | Provider Aggregate | LLM 配置 |
| MCP | MCPServer Aggregate | MCP 工具管理 |

### 6.2 上下文间通信

- **同进程**：通过应用服务调用
- **跨进程**：通过消息队列

---

## 7. 常见错误与禁止模式

### 7.1 禁止的事务脚本

```go
// ❌ 错误：事务脚本，不是 DDD
func CreateAgent(name, code string) {
    db.Insert("agents", map[string]any{
        "name": name,
        "code": code,
    })
}
```

### 7.2 禁止的 Anemic Domain Model

```go
// ❌ 错误：贫血模型，只有 getter/setter
type Agent struct {
    ID   string
    Name string
}
func (a *Agent) SetName(name string) { a.Name = name }

// ✅ 正确：丰富领域模型
type Agent struct {
    id   AgentID
    name string
}
func (a *Agent) Rename(newName string) error {
    if newName == "" {
        return ErrAgentNameRequired
    }
    a.name = newName
    return nil
}
```

### 7.3 禁止的跨聚合直接修改

```go
// ❌ 错误：跨聚合直接修改
func ChangeAgentUser(agent *Agent, newUserCode string) {
    agent.UserCode = newUserCode  // 不应该直接修改
}

// ✅ 正确：通过聚合根方法
func (a *Agent) ChangeUser(newUserCode string) error {
    if newUserCode == "" {
        return ErrUserCodeRequired
    }
    a.userCode = newUserCode
    return nil
}
```

### 7.4 禁止在领域层引入技术细节

```go
// ❌ 错误：领域层依赖外部服务
type Agent struct {
    httpClient *http.Client  // 不应该
}

// ✅ 正确：领域模型纯净
type Agent struct {
    // 只有业务属性
}
```

---

## 8. 代码组织规范

### 8.1 文件命名

| 类型 | 命名规范 | 示例 |
|------|---------|------|
| 聚合 | `aggregate.go` | `agent.go` |
| 仓储接口 | `aggregate_repository.go` | `agent_repository.go` |
| 仓储实现 | `aggregate_sqlite_repository.go` | `agent_sqlite_repository.go` |
| 应用服务 | `aggregate_service.go` | `agent_service.go` |
| 命令 | `aggregate_commands.go` | `agent_commands.go` |
| 事件 | `aggregate_events.go` | `agent_events.go` |
| 值对象 | `value_objects.go` | `user_code.go` |

### 8.2 包结构

```
domain/
├── agent.go           # 聚合根
├── agent_repository.go # 仓储接口
├── agent_test.go     # 聚合测试
└── value_objects.go   # 值对象

infrastructure/
└── persistence/
    ├── agent_sqlite_repository.go  # 仓储实现
    └── migrations/                  # 数据库迁移
```

---

## 9. 测试规范

### 9.1 测试位置

- **聚合测试**：`domain/aggregate_test.go`
- **仓储测试**：`infrastructure/persistence/aggregate_repository_test.go`
- **应用服务测试**：`application/aggregate_service_test.go`

### 9.2 测试原则

- **聚合测试**：验证不变量和业务规则
- **仓储测试**：使用真实数据库（SQLite memory）
- **应用服务测试**：Mock 仓储，验证用例编排

---

## 10. 决策速查

| 场景 | 决策 |
|------|------|
| 需要保存数据 | Repository |
| 跨聚合操作 | Application Service |
| 业务规则验证 | Aggregate Factory |
| 状态变更 | Aggregate 方法 |
| 跨上下文通信 | Domain Event |
| 复杂查询 | Specification 模式 |
| 工具方法 | Domain Service |

---

## 违反规则的处理

当发现违反 DDD 规则的代码时：
1. **不要**在错误的基础上继续
2. **指出**违规之处
3. **提出**重构建议
4. **在获得确认后**再修改

---

## 参考资料

- 《Domain-Driven Design》- Eric Evans
- 《Implementing Domain-Driven Design》- Vaughn Vernon
- 《Get Your DDD Right》- Thomas Plishka
