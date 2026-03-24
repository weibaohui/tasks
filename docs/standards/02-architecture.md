# 技术架构规范

## 1. DDD 分层架构

```
interfaces/http    →  接收请求，调用 application service
application       →  编排领域对象，处理用例
domain           →  纯业务逻辑，无依赖
infrastructure    →  实现接口（repository、hook、external）
```

### 1.1 分层依赖规则

| 层 | 可依赖 |
|----|--------|
| interfaces | application, domain, infrastructure |
| application | domain, infrastructure |
| domain | 无（纯业务） |
| infrastructure | domain |

**绝对禁止**：
- `infrastructure` 引用 `interfaces`
- `domain` 引用其他任何层
- 循环依赖

## 2. 限界上下文

| 上下文 | 核心域 | 聚合 |
|--------|--------|------|
| Task | 任务管理 | Task |
| Agent | Agent 配置 | Agent |
| User | 用户管理 | User |
| Conversation | 对话记录 | ConversationRecord |
| Channel | 渠道管理 | Channel |
| LLM | LLM 配置 | LLMProvider |
| MCP | MCP 工具 | MCPServer |

## 3. 聚合根规则

### 3.1 聚合根职责
- 创建时验证内部不变量
- 控制内部状态变更
- 对外暴露唯一入口

### 3.2 项目中的聚合

| 聚合 | 聚合根 | 值对象 |
|------|--------|--------|
| Task | Task | TaskID |
| Agent | Agent | AgentID, UserCode |
| User | User | UserID |
| ConversationRecord | ConversationRecord | RecordID |
| Provider | LLMProvider | ProviderID |
| Channel | Channel | ChannelID |
| MCPServer | MCPServer | ServerID |

## 4. 仓储模式

### 4.1 接口定义位置
**仓储接口必须在 `domain` 层定义**

```
✅ domain/agent_repository.go     # 接口定义
✅ infrastructure/persistence/agent_sqlite.go  # 实现
❌ infrastructure/agent_repository.go  # 禁止
```

### 4.2 仓储方法命名

| 操作 | 方法名 |
|------|--------|
| 按 ID 查询 | `FindByID(ctx, id)` |
| 保存 | `Save(ctx, entity)` |
| 删除 | `Delete(ctx, id)` |
| 列表查询 | `List(ctx, filter)` |

## 5. 领域事件

### 5.1 命名规范
```
<聚合><事件名>Event

示例：
AgentCreatedEvent
TaskCompletedEvent
UserActivatedEvent
```

### 5.2 事件规则
- 事件是**不可变**的
- 使用**事件总线**发布
- 事件处理器在 `infrastructure` 层

## 6. 值对象规则

### 6.1 值对象特征
- **不可变**
- 通过构造函数创建
- 无身份标识
- 用 `Equals` 判断相等

### 6.2 示例
```go
// ✅ 正确：不可变值对象
type UserCode struct { value string }
func NewUserCode(code string) (*UserCode, error) {
    if code == "" { return nil, ErrUserCodeRequired }
    return &UserCode{value: code}, nil
}

// ❌ 错误：可变对象
type UserCode struct { Code string }  // 可修改
```

## 7. 应用服务职责

- 编排领域对象和仓储
- 处理用例和事务边界
- **不包含业务逻辑**

```go
// ✅ 正确：纯编排
func (s *AgentService) CreateAgent(ctx, cmd) (*Agent, error) {
    agent, err := domain.NewAgent(...)
    if err != nil { return nil, err }
    return agent, s.repo.Save(ctx, agent)
}

// ❌ 错误：包含业务逻辑
func (s *AgentService) CreateAgent(ctx, cmd) {
    price := price * 0.9  // ❌ 业务逻辑不应在这
}
```

## 8. 禁止模式

| 模式 | 错误示例 | 正确做法 |
|------|----------|----------|
| 事务脚本 | 直接操作 DB | 通过 Repository |
| 贫血模型 | 只有 getter/setter | 丰富领域模型 |
| 跨聚合修改 | 直接修改其他聚合 | 通过聚合根方法 |
| 领域层技术依赖 | `domain` 层用 `http.Client` | 技术细节放 `infrastructure` |

## 9. 前后端分离

### 9.1 后端 (Go)
- 只提供 API，不渲染页面
- 使用 HTTP/JSON 通信
- 遵循 REST 风格

### 9.2 前端 (React)
- 只负责页面渲染
- 通过 API 与后端交互
- 状态管理使用 React hooks/zustand

### 9.3 API 规范
- 路径：`/api/v1/{resource}`
- 认证：`Authorization: Bearer {token}`
- 错误响应：`{"code": 400, "message": "错误信息"}`
