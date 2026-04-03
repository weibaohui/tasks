# Agent LLM Provider 关联重构设计文档

## 1. 背景与问题

### 1.1 当前问题
1. **Agent 的 `provider_key` 字段未被使用**
   - Agent 表有 `provider_key` 字段，但代码中通过 `FindDefaultActive` 获取用户的默认 Provider
   - 忽略了 Agent 自身配置的 Provider 关联

2. **Claude Code Config 缺少 API Key 和 API Base**
   - `ClaudeCodeConfig` 只包含 `Model` 字段
   - API Key 和 API Base 依赖用户默认 Provider，而非 Agent 指定的 Provider

3. **配置来源不清晰**
   - 用户无法为不同 Agent 配置不同的 LLM Provider
   - 所有 Agent 共享同一个默认 Provider 配置

### 1.2 目标
- 支持为每个 Agent 独立配置 LLM Provider
- 明确配置优先级：Agent 指定 Provider > 用户默认 Provider
- Claude Code 使用 Agent 指定 Provider 的 API Key/Base + Config 中的 Model

---

## 2. 设计方案

### 2.1 数据库变更

#### Agent 表添加字段
```sql
-- 添加 llm_provider_id 外键
ALTER TABLE agents ADD COLUMN llm_provider_id TEXT REFERENCES llm_providers(id);

-- 迁移数据：将 provider_key 转换为 llm_provider_id
-- （需要查找 llm_providers 表中匹配的 provider_key）
```

#### 数据迁移策略
1. 新字段 `llm_provider_id` 允许 NULL
2. 如果 Agent 没有指定 Provider，回退到用户默认 Provider
3. 原有 `provider_key` 字段标记为废弃（后续版本移除）

---

### 2.2 Domain 层变更

#### Agent 实体
```go
type Agent struct {
    id                    AgentID
    agentCode             AgentCode
    agentType             AgentType
    userCode              string
    name                  string
    description           string
    
    // LLM Provider 关联
    llmProviderID         LLMProviderID  // 新增：关联的 LLM Provider ID
    
    // 通用 LLM 配置（BareLLM 使用）
    model                 string
    maxTokens             int
    temperature           float64
    maxIterations         int
    historyMessages       int
    
    // Claude Code 特有配置
    claudeCodeConfig      *ClaudeCodeConfig
    
    // 其他字段...
    skillsList            []string
    toolsList             []string
    isActive              bool
    isDefault             bool
    enableThinkingProcess bool
    shadowFrom            string
    createdAt             time.Time
    updatedAt             time.Time
}
```

#### 新增方法
```go
// LLMProviderID 获取关联的 LLM Provider ID
func (a *Agent) LLMProviderID() LLMProviderID { return a.llmProviderID }

// SetLLMProviderID 设置关联的 LLM Provider ID
func (a *Agent) SetLLMProviderID(id LLMProviderID) {
    a.llmProviderID = id
    a.updatedAt = time.Now()
}
```

---

### 2.3 Repository 层变更

#### Agent Repository
- 更新 `Save()` 方法：保存 `llm_provider_id`
- 更新 `FindByID()` / `FindAll()`：加载 `llm_provider_id`
- 更新 Snapshot 转换逻辑

#### 新增查询方法（可选）
```go
// FindByLLMProviderID 按 LLM Provider ID 查找 Agents
FindByLLMProviderID(ctx context.Context, providerID LLMProviderID) ([]*Agent, error)
```

---

### 2.4 Application 层变更

#### Agent Service
- 创建/更新 Agent 时支持指定 `llm_provider_id`
- 验证 LLM Provider 是否存在且属于该用户

#### Claude Code Processor
修改 `buildOptions` 函数：

```go
// 配置优先级：
// 1. 如果 Agent 指定了 LLMProvider，使用该 Provider
// 2. 否则使用用户的默认 Provider
// 3. Model 优先使用 ClaudeCodeConfig.Model，否则使用 Provider.DefaultModel

func (p *ClaudeCodeProcessor) resolveProvider(ctx context.Context, agent *domain.Agent) (*domain.LLMProvider, error) {
    if agent == nil {
        return nil, nil
    }
    
    // 1. 尝试获取 Agent 指定的 Provider
    if agent.LLMProviderID().String() != "" {
        provider, err := p.providerRepo.FindByID(ctx, agent.LLMProviderID())
        if err == nil && provider != nil {
            return provider, nil
        }
        p.logger.Warn("Agent 指定的 Provider 不存在或已禁用，回退到默认 Provider",
            zap.String("agent_id", agent.ID().String()),
            zap.String("llm_provider_id", agent.LLMProviderID().String()))
    }
    
    // 2. 回退到用户默认 Provider
    return p.providerRepo.FindDefaultActive(ctx, agent.UserCode())
}
```

---

### 2.5 Interface 层变更

#### HTTP Handler
更新 Agent API：
- `POST /api/v1/agents` - 创建 Agent 支持 `llm_provider_id`
- `PUT /api/v1/agents/:id` - 更新 Agent 支持 `llm_provider_id`
- `GET /api/v1/agents/:id` - 返回包含 `llm_provider_id` 和 Provider 基本信息

#### 请求/响应 DTO
```go
// CreateAgentRequest / UpdateAgentRequest
type AgentRequest struct {
    Name              string              `json:"name"`
    Description       string              `json:"description"`
    AgentType         string              `json:"agent_type"`
    LLMProviderID     string              `json:"llm_provider_id,omitempty"`  // 新增
    Model             string              `json:"model"`
    MaxTokens         int                 `json:"max_tokens"`
    Temperature       float64             `json:"temperature"`
    MaxIterations     int                 `json:"max_iterations"`
    // ... 其他字段
    ClaudeCodeConfig  *ClaudeCodeConfig   `json:"claude_code_config,omitempty"`
}

// AgentResponse
type AgentResponse struct {
    ID                string              `json:"id"`
    AgentCode         string              `json:"agent_code"`
    Name              string              `json:"name"`
    // ... 其他字段
    LLMProviderID     string              `json:"llm_provider_id,omitempty"`  // 新增
    LLMProviderName   string              `json:"llm_provider_name,omitempty"` // 新增（冗余，方便展示）
}
```

---

### 2.6 前端变更

#### Agent 编辑页面重构

**Tab 1: 基础信息**
- 名称
- 描述
- Agent Code（只读）
- Agent 类型（BareLLM / CodingAgent）
- **关联的 LLM Provider**（下拉选择框，可选）

**Tab 2: Claude Code 配置（仅 CodingAgent 显示）**
- 模型选择（输入框，提示：留空使用 Provider 默认模型）
- 系统提示词
- 权限模式（default / acceptEdits / plan / bypassPermissions）
- 最大思考 Token
- 最大对话轮次
- 超时时间（秒）
- 沙箱启用
- 其他高级设置

**Tab 3: 通用 LLM 配置（BareLLM 使用）**
- 模型名称
- Max Tokens
- Temperature
- Max Iterations
- History Messages

#### 提示信息
- 如果未选择 LLM Provider，显示提示："未选择时将使用用户默认的 LLM Provider"
- 显示已选 Provider 的基本信息（名称、API Base）

---

## 3. 配置优先级规则

### Claude Code 运行时配置解析

| 配置项 | 优先级 1 | 优先级 2 | 优先级 3 |
|--------|---------|---------|---------|
| **API Key** | Agent.LLMProvider.api_key | 用户默认 Provider.api_key | 环境变量 |
| **API Base** | Agent.LLMProvider.api_base | 用户默认 Provider.api_base | 默认值 |
| **Model** | ClaudeCodeConfig.Model | Agent.LLMProvider.default_model | Claude Code 默认 |
| **Provider Type** | Agent.LLMProvider.provider_type | 用户默认 Provider.provider_type | openai |

### 配置解析流程图

```text
开始
  │
  ▼
Agent 是否指定 LLMProviderID?
  │
  ├─ 是 ──▶ 查找该 Provider
  │           │
  │           ├─ 存在且启用 ──▶ 使用该 Provider
  │           │
  │           └─ 不存在/禁用 ──▶ 警告日志
  │                               │
  ▼                               ▼
否                               回退到用户默认 Provider
  │                               │
  ▼                               ▼
使用用户默认 Provider ◀───────────┘
  │
  ▼
构建 Claude Code 配置
  │
  ├─ API Key = Provider.APIKey()
  ├─ API Base = Provider.APIBase()
  ├─ Model = ClaudeCodeConfig.Model ?? Provider.DefaultModel()
  └─ 其他参数 = ClaudeCodeConfig
  │
  ▼
结束
```

---

## 4. 迁移计划

### 阶段 1: 数据库迁移
1. 添加 `llm_provider_id` 字段
2. 数据迁移脚本（provider_key → llm_provider_id）
3. 测试数据完整性

### 阶段 2: Domain & Repository 层
1. 更新 Agent 实体
2. 更新 Repository 查询和保存逻辑
3. 添加单元测试

### 阶段 3: Application & Interface 层
1. 更新 Agent Service
2. 更新 HTTP Handler
3. 更新 Claude Code Processor

### 阶段 4: 前端
1. 重构 Agent 编辑页面
2. 添加 LLM Provider 选择组件
3. 根据 Agent 类型动态显示不同配置 Tab

### 阶段 5: 测试 & 发布
1. 集成测试
2. 回归测试
3. 文档更新
4. 部署

---

## 5. 兼容性考虑

### 向后兼容
- `llm_provider_id` 为可选字段
- 未指定时保持现有行为（使用用户默认 Provider）
- API 响应中 `llm_provider_id` 为空字符串表示未指定

### API 版本控制
- 本次变更为向后兼容的增量更新
- 不需要 API 版本号升级

---

## 6. 待办任务

- [ ] 创建数据库迁移脚本
- [ ] 更新 Agent Domain 实体
- [ ] 更新 Agent Repository
- [ ] 更新 Agent Service
- [ ] 更新 HTTP Handler
- [ ] 更新 Claude Code Processor
- [ ] 重构前端 Agent 编辑页面
- [ ] 编写单元测试
- [ ] 编写集成测试
- [ ] 更新 API 文档

---

## 7. 附录

### 相关代码文件

**Backend:**
- `backend/domain/agent.go`
- `backend/domain/llm_provider.go`
- `backend/infrastructure/persistence/agent_repository.go`
- `backend/infrastructure/persistence/schema.go`
- `backend/application/agent_service.go`
- `backend/interfaces/http/agent_handler.go`
- `backend/infrastructure/claudecode/processor.go`

**Frontend:**
- `frontend/src/pages/AgentEditPage.tsx` (需创建)
- `frontend/src/pages/AgentListPage.tsx` (需更新)
- `frontend/src/types/agent.ts` (需更新)

### 数据库表结构

```sql
-- agents 表（变更后）
CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    agent_code TEXT NOT NULL UNIQUE,
    user_code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    agent_type TEXT NOT NULL DEFAULT 'BareLLM',
    
    -- LLM Provider 关联（新增）
    llm_provider_id TEXT REFERENCES llm_providers(id),
    
    -- 通用 LLM 配置
    model TEXT,
    max_tokens INTEGER NOT NULL,
    temperature REAL NOT NULL,
    max_iterations INTEGER NOT NULL,
    history_messages INTEGER NOT NULL,
    
    -- Claude Code 配置
    claude_code_config TEXT,
    
    -- 其他字段...
    provider_key TEXT,  -- 废弃，保留用于迁移
    
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```
