# DDD 架构约束

## ⚠️ 强制约束

以下规则是**必须严格遵守**的，任何代码变更不得违反。

## 1. 分层约束

```
interfaces/http    →  接收请求
application       →  编排用例
domain           →  纯业务逻辑（无依赖）
infrastructure    →  实现接口
```

### 禁止的依赖

| 层 | 禁止依赖 |
|----|----------|
| `domain` | 任何其他层 |
| `infrastructure` | `interfaces` |

## 2. 聚合根约束

### 2.1 聚合根必须
- 在**创建时**验证不变量
- 通过**工厂方法**创建
- 控制所有内部状态变更

### 2.2 项目中的聚合

| 聚合 | 文件位置 | 工厂方法 |
|------|----------|----------|
| Task | `domain/task.go` | `NewTask()` |
| Agent | `domain/agent.go` | `NewAgent()` |
| User | `domain/user.go` | `NewUser()` |
| ConversationRecord | `domain/conversation_record.go` | `NewConversationRecord()` |
| LLMProvider | `domain/provider.go` | `NewLLMProvider()` |
| Channel | `domain/channel.go` | `NewChannel()` |
| MCPServer | `domain/mcp_server.go` | `NewMCPServer()` |

## 3. 仓储约束

### 3.1 接口定义位置
**仓储接口必须在 `domain` 层定义**

```
✅ domain/agent_repository.go
✅ domain/conversation_record_repository.go

❌ infrastructure/persistence/agent_repository.go  # 禁止
```

### 3.2 接口命名
```go
type AgentRepository interface {
    FindByID(ctx context.Context, id AgentID) (*Agent, error)
    Save(ctx context.Context, agent *Agent) error
    Delete(ctx context.Context, id AgentID) error
}
```

## 4. 值对象约束

值对象必须**不可变**：

```go
// ✅ 正确
type UserCode struct {
    value string
}
func NewUserCode(code string) (*UserCode, error) {
    if code == "" {
        return nil, ErrUserCodeRequired
    }
    return &UserCode{value: code}, nil
}

// ❌ 错误：可变对象
type UserCode struct {
    Code string  // 可修改
}
```

## 5. 应用服务约束

应用服务**只负责编排**，不包含业务逻辑：

```go
// ✅ 正确
func (s *AgentAppService) CreateAgent(ctx context.Context, cmd CreateAgentCommand) (*Agent, error) {
    agent, err := domain.NewAgent(...)
    if err != nil {
        return nil, err
    }
    return agent, s.repo.Save(ctx, agent)
}

// ❌ 错误：包含业务逻辑
func (s *AgentAppService) CreateAgent(ctx context.Context, cmd CreateAgentCommand) (*Agent, error) {
    cmd.Name = strings.TrimSpace(cmd.Name)  // 业务逻辑不应该在这
    // ...
}
```

## 6. 禁止模式

| 禁止 | 说明 |
|------|------|
| 贫血模型 | 只有 getter/setter，无业务方法 |
| 事务脚本 | 直接操作数据库，无领域逻辑 |
| 跨聚合修改 | 直接修改其他聚合的状态 |
| 领域层技术依赖 | `domain` 层引用 `net/http`、`database/sql` 等 |

## 7. 限界上下文

| 上下文 | 核心域 | 边界 |
|--------|--------|------|
| Task | 任务管理 | 只管理任务 |
| Agent | Agent 配置 | 只管理 Agent |
| User | 用户管理 | 只管理用户 |
| Conversation | 对话记录 | 只管理对话记录 |
| Channel | 渠道管理 | 只管理渠道 |
| LLM | LLM 配置 | 只管理 Provider |
| MCP | MCP 工具 | 只管理 MCP Server |

## 8. 决策速查

| 场景 | 决策 |
|------|------|
| 需要持久化 | Repository |
| 跨聚合操作 | Application Service |
| 业务规则验证 | Aggregate Factory |
| 状态变更 | Aggregate 方法 |
| 跨上下文通信 | Domain Event |
| 复杂查询 | Specification |
| 工具方法 | Domain Service（无状态） |

## 9. 违规处理

当发现违规代码时：
1. **不要**在错误基础上继续
2. **指出**违规之处
3. **提出**重构建议
4. **获得确认后**再修改
