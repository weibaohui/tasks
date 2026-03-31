# Requirement 状态变更 Hook 系统设计文档

## 1. 架构设计

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Requirement State Change Hook Architecture               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Application Service                              │   │
│  │              RequirementDispatchService / RequirementService            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                      │                                      │
│                                      ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       Requirement Domain                              │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐   │   │
│  │  │   Model     │  │State Change │  │  ReplicaAgentManager     │   │   │
│  │  │ - StartDispatch│ │ - From/To  │  │  - Dispose() 强制销毁   │   │   │
│  │  │ - MarkCoding │  │ - Trigger   │  │  - EnsureDisposed()     │   │   │
│  │  │ - MarkFailed │  │ - Reason    │  │                        │   │   │
│  │  └──────────────┘  └──────────────┘  └──────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                      │                                      │
│                                      ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │              ConfigurableHookExecutor (数据库配置驱动)                  │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐   │   │
│  │  │HookConfigRepo│  │   Executor  │  │    ActionExecutors       │   │   │
│  │  │ - 从DB加载配置 │  │ - 按优先级执行│  │ - TriggerAgentExecutor │   │   │
│  │  │ - 缓存配置    │  │ - 错误继续   │  │ - NotificationExecutor │   │   │
│  │  └──────────────┘  └──────────────┘  │ - WebhookExecutor       │   │   │
│  │                                     └──────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 模块职责

| 模块 | 职责 |
|------|------|
| Requirement | 领域模型，管理状态 |
| ReplicaAgentManager | 强制销毁分身，做成代码约束 |
| HookConfigRepository | 从数据库加载 Hook 配置 |
| ConfigurableHookExecutor | 执行数据库配置的 Hook |
| ActionExecutors | 执行具体动作（触发 Agent、通知等） |

## 2. 核心设计变更

### 2.1 移除：CleanupHook 概念

**之前的设计**：通过 `RequirementCleanupHook` 监听状态变更来清理分身。

**问题**：
- Hook 可能被禁用，导致分身永不销毁
- 清理是可选行为，不够可靠

**新设计**：
- 分身销毁是**强制代码约束**，不通过 Hook 实现
- `ReplicaAgentManager.EnsureDisposed()` 在 `MarkFailed` 和 `MarkPROpened` 后**必须**调用
- Hook 系统仅用于**可配置的动作**（触发其他 Agent、通知等）

### 2.2 ReplicaAgentManager - 强制销毁分身

```go
// ReplicaAgentManager 分身管理器
// 负责分身的创建和销毁，销毁是强制行为
type ReplicaAgentManager struct {
    agentRepo domain.AgentRepository
}

// NewReplicaAgentManager 创建管理器
func NewReplicaAgentManager(agentRepo domain.AgentRepository) *ReplicaAgentManager {
    return &ReplicaAgentManager{agentRepo: agentRepo}
}

// EnsureDisposed 确保分身已销毁（幂等方法）
// 这是一个幂等操作，调用多次和调用一次效果相同
func (m *ReplicaAgentManager) EnsureDisposed(ctx context.Context, replicaAgentID, workspacePath string) {
    if replicaAgentID == "" {
        return
    }

    // 1. 删除分身 Agent
    if err := m.agentRepo.Delete(ctx, domain.NewAgentID(replicaAgentID)); err != nil {
        // 记录错误但不阻塞
        log.Printf("failed to delete replica agent %s: %v", replicaAgentID, err)
    }

    // 2. 清理工作目录
    if workspacePath != "" {
        if err := os.RemoveAll(workspacePath); err != nil {
            log.Printf("failed to cleanup workspace %s: %v", workspacePath, err)
        }
    }
}
```

### 2.3 领域模型改造

#### 2.3.1 Requirement 模型变更

```go
type Requirement struct {
    // ... 现有字段 ...

    // replicaAgentManager 分身管理器（注入依赖）
    replicaAgentManager *ReplicaAgentManager
}

// SetReplicaAgentManager 设置分身管理器
func (r *Requirement) SetReplicaAgentManager(manager *ReplicaAgentManager) {
    r.replicaAgentManager = manager
}
```

#### 2.3.2 MarkFailed 改造

```go
func (r *Requirement) MarkFailed(lastError string) {
    fromStatus := r.status
    fromDevState := r.devState

    r.status = RequirementStatusInProgress
    r.devState = RequirementDevStateFailed
    r.lastError = lastError
    now := time.Now()
    r.updatedAt = now

    // 触发状态变更回调
    r.fireStateChange(&StateChange{
        FromStatus:   fromStatus,
        ToStatus:     r.status,
        FromDevState: fromDevState,
        ToDevState:   r.devState,
        Trigger:      "MarkFailed",
        Reason:       lastError,
        Timestamp:    now,
    })

    // 强制销毁分身（代码约束，不可跳过）
    if r.replicaAgentManager != nil {
        r.replicaAgentManager.EnsureDisposed(context.Background(), r.replicaAgentID, r.workspacePath)
    }
}
```

#### 2.3.3 MarkPROpened 改造

```go
func (r *Requirement) MarkPROpened(prURL, branchName string) {
    fromStatus := r.status
    fromDevState := r.devState

    now := time.Now()
    r.status = RequirementStatusDone
    r.devState = RequirementDevStatePROpened
    r.prURL = prURL
    if branchName != "" {
        r.branchName = branchName
    }
    r.lastError = ""
    r.completedAt = &now
    r.updatedAt = now

    r.fireStateChange(&StateChange{
        FromStatus:   fromStatus,
        ToStatus:     r.status,
        FromDevState: fromDevState,
        ToDevState:   r.devState,
        Trigger:      "MarkPROpened",
        Reason:       "",
        Timestamp:    now,
    })

    // 强制销毁分身（代码约束，不可跳过）
    if r.replicaAgentManager != nil {
        r.replicaAgentManager.EnsureDisposed(context.Background(), r.replicaAgentID, r.workspacePath)
    }
}
```

## 3. 可配置 Hook 系统设计

### 3.1 数据库 Schema

```sql
-- Hook 配置表
CREATE TABLE requirement_hook_configs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    trigger_point TEXT NOT NULL,           -- start_dispatch, mark_coding, mark_failed, mark_pr_opened
    action_type TEXT NOT NULL,             -- trigger_agent, notification, webhook
    action_config TEXT NOT NULL,           -- JSON 配置
    enabled INTEGER DEFAULT 1,
    priority INTEGER DEFAULT 50,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Hook 执行日志表
CREATE TABLE requirement_hook_action_logs (
    id TEXT PRIMARY KEY,
    hook_config_id TEXT NOT NULL,
    requirement_id TEXT NOT NULL,
    trigger_point TEXT NOT NULL,
    action_type TEXT NOT NULL,
    status TEXT NOT NULL,                 -- pending, running, success, failed
    input_context TEXT,                   -- 执行时的上下文
    result TEXT,
    error TEXT,
    started_at TEXT,
    completed_at TEXT,
    FOREIGN KEY (hook_config_id) REFERENCES requirement_hook_configs(id),
    FOREIGN KEY (requirement_id) REFERENCES requirements(id)
);

-- 索引
CREATE INDEX idx_hook_configs_trigger ON requirement_hook_configs(trigger_point, enabled);
CREATE INDEX idx_hook_logs_requirement ON requirement_hook_action_logs(requirement_id);
CREATE INDEX idx_hook_logs_status ON requirement_hook_action_logs(status);
```

### 3.2 Hook 配置模型

```go
// RequirementHookConfig Hook 配置
type RequirementHookConfig struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    TriggerPoint string    `json:"trigger_point"` // start_dispatch, mark_coding, mark_failed, mark_pr_opened
    ActionType   string    `json:"action_type"`   // trigger_agent, notification, webhook
    ActionConfig string    `json:"action_config"` // JSON
    Enabled      bool      `json:"enabled"`
    Priority     int       `json:"priority"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// RequirementHookActionLog 执行日志
type RequirementHookActionLog struct {
    ID             string    `json:"id"`
    HookConfigID   string    `json:"hook_config_id"`
    RequirementID  string    `json:"requirement_id"`
    TriggerPoint   string    `json:"trigger_point"`
    ActionType     string    `json:"action_type"`
    Status         string    `json:"status"` // pending, running, success, failed
    InputContext   string    `json:"input_context"`
    Result         string    `json:"result"`
    Error          string    `json:"error"`
    StartedAt      time.Time `json:"started_at"`
    CompletedAt    *time.Time `json:"completed_at"`
}
```

### 3.3 Hook 配置仓储

```go
// RequirementHookConfigRepository Hook 配置仓储接口
type RequirementHookConfigRepository interface {
    Save(ctx context.Context, config *RequirementHookConfig) error
    FindByID(ctx context.Context, id string) (*RequirementHookConfig, error)
    FindByTriggerPoint(ctx context.Context, triggerPoint string) ([]*RequirementHookConfig, error)
    FindEnabledByTriggerPoint(ctx context.Context, triggerPoint string) ([]*RequirementHookConfig, error)
    Delete(ctx context.Context, id string) error
}

// RequirementHookActionLogRepository 执行日志仓储接口
type RequirementHookActionLogRepository interface {
    Save(ctx context.Context, log *RequirementHookActionLog) error
    FindByRequirementID(ctx context.Context, requirementID string) ([]*RequirementHookActionLog, error)
    FindByHookConfigID(ctx context.Context, hookConfigID string, limit int) ([]*RequirementHookActionLog, error)
}
```

### 3.4 动作配置结构

#### trigger_agent 配置

```go
type TriggerAgentActionConfig struct {
    AgentID         string            `json:"agent_id"`          // 目标 Agent ID
    PromptTemplate  string            `json:"prompt_template"`   // Prompt 模板
    TimeoutMinutes  int               `json:"timeout_minutes"`   // 超时时间
    WorkspaceTemplate string          `json:"workspace_template"` // 工作目录模板
    Context         ActionContext     `json:"context"`           // 上下文配置
}

type ActionContext struct {
    IncludeProject      bool `json:"include_project"`
    IncludeRequirement  bool `json:"include_requirement"`
    IncludeHistory      bool `json:"include_history"`
}
```

#### notification 配置

```go
type NotificationActionConfig struct {
    Channel  string `json:"channel"`  // feishu, email
    Template string `json:"template"`  // 通知模板
}
```

#### webhook 配置

```go
type WebhookActionConfig struct {
    URL           string            `json:"url"`
    Method       string            `json:"method"` // GET, POST, PUT
    Headers      map[string]string `json:"headers"`
    BodyTemplate string            `json:"body_template"`
}
```

### 3.5 ConfigurableHookExecutor

```go
// ConfigurableHookExecutor 可配置 Hook 执行器
// 从数据库加载配置，按配置执行动作
type ConfigurableHookExecutor struct {
    configRepo  RequirementHookConfigRepository
    logRepo     RequirementHookActionLogRepository
    actionRuns  []ActionExecutor
    logger      Logger
}

type ActionExecutor interface {
    Execute(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error)
}

type ActionResult struct {
    Success bool
    Output  string
    Error   error
}

// Execute 执行指定触发点的所有已配置 Hook
func (e *ConfigurableHookExecutor) Execute(
    ctx context.Context,
    triggerPoint string,
    req *Requirement,
    change *StateChange,
) {
    // 1. 从数据库加载该触发点的配置
    configs, err := e.configRepo.FindEnabledByTriggerPoint(ctx, triggerPoint)
    if err != nil {
        e.logger.Error("failed to load hook configs", "trigger", triggerPoint, "error", err)
        return
    }

    // 2. 按优先级排序
    sort.Slice(configs, func(i, j int) bool {
        return configs[i].Priority < configs[j].Priority
    })

    // 3. 遍历执行
    for _, config := range configs {
        e.executeConfig(ctx, config, req, change)
    }
}

func (e *ConfigurableHookExecutor) executeConfig(
    ctx context.Context,
    config *RequirementHookConfig,
    req *Requirement,
    change *StateChange,
) {
    // 1. 创建执行日志
    log := &RequirementHookActionLog{
        ID:            generateID(),
        HookConfigID:  config.ID,
        RequirementID: req.ID().String(),
        TriggerPoint:  change.Trigger,
        ActionType:    config.ActionType,
        Status:        "pending",
        StartedAt:     time.Now(),
    }
    _ = e.logRepo.Save(ctx, log)

    // 2. 查找对应的动作执行器
    var executor ActionExecutor
    for _, ae := range e.actionRuns {
        if ae.Supports(config.ActionType) {
            executor = ae
            break
        }
    }
    if executor == nil {
        log.Status = "failed"
        log.Error = fmt.Sprintf("no executor for action type: %s", config.ActionType)
        _ = e.logRepo.Save(ctx, log)
        return
    }

    // 3. 执行动作
    log.Status = "running"
    _ = e.logRepo.Save(ctx, log)

    result, err := executor.Execute(ctx, config, req, change)

    // 4. 更新日志
    if err != nil {
        log.Status = "failed"
        log.Error = err.Error()
    } else {
        log.Status = "success"
        log.Result = result.Output
    }
    now := time.Now()
    log.CompletedAt = &now
    _ = e.logRepo.Save(ctx, log)
}
```

### 3.6 TriggerAgentExecutor

```go
// TriggerAgentExecutor 触发 Agent 动作执行器
type TriggerAgentExecutor struct {
    agentRepo  domain.AgentRepository
    idGen     domain.IDGenerator
    publisher  MessagePublisher
}

func (e *TriggerAgentExecutor) Supports(actionType string) bool {
    return actionType == "trigger_agent"
}

func (e *TriggerAgentExecutor) Execute(
    ctx context.Context,
    config *RequirementHookConfig,
    req *Requirement,
    change *StateChange,
) (*ActionResult, error) {
    // 1. 解析配置
    var actionConfig TriggerAgentActionConfig
    if err := json.Unmarshal([]byte(config.ActionConfig), &actionConfig); err != nil {
        return nil, fmt.Errorf("invalid action config: %w", err)
    }

    // 2. 构建 Prompt
    prompt := e.renderPrompt(actionConfig.PromptTemplate, req, change)

    // 3. 创建新分身
    replicaAgent, err := e.createReplica(ctx, actionConfig, req)
    if err != nil {
        return nil, fmt.Errorf("failed to create replica: %w", err)
    }

    // 4. 发送任务消息
    err = e.publisher.PublishInbound(&InboundMessage{
        Channel:   "internal",
        SenderID:  "hook_system",
        ChatID:    replicaAgent.ID().String(),
        Content:   prompt,
        Timestamp: time.Now(),
        Metadata: map[string]any{
            "agent_code":     replicaAgent.AgentCode().String(),
            "requirement_id": req.ID().String(),
            "hook_config_id": config.ID,
        },
    })

    if err != nil {
        return nil, fmt.Errorf("failed to publish task: %w", err)
    }

    return &ActionResult{
        Success: true,
        Output:  fmt.Sprintf("triggered agent %s", replicaAgent.ID().String()),
    }, nil
}

func (e *TriggerAgentExecutor) renderPrompt(template string, req *Requirement, change *StateChange) string {
    // 替换变量
    result := template
    result = strings.ReplaceAll(result, "${requirement.id}", req.ID().String())
    result = strings.ReplaceAll(result, "${requirement.title}", req.Title())
    result = strings.ReplaceAll(result, "${requirement.description}", req.Description())
    result = strings.ReplaceAll(result, "${requirement.acceptance_criteria}", req.AcceptanceCriteria())
    result = strings.ReplaceAll(result, "${project.id}", req.ProjectID().String())
    // ... 其他变量
    return result
}
```

## 4. 服务集成设计

### 4.1 应用服务改造

```go
type RequirementDispatchService struct {
    requirementRepo    domain.RequirementRepository
    projectRepo       domain.ProjectRepository
    agentRepo         domain.AgentRepository
    taskService       *TaskApplicationService
    sessionService    *SessionApplicationService
    idGenerator       domain.IDGenerator
    inboundPublisher  interface {
        PublishInbound(msg *channelBus.InboundMessage)
    }

    // 强制销毁分身的管理器
    replicaAgentManager *ReplicaAgentManager

    // 可配置 Hook 执行器
    hookExecutor *ConfigurableHookExecutor
}

func NewRequirementDispatchService(
    requirementRepo domain.RequirementRepository,
    projectRepo domain.ProjectRepository,
    agentRepo domain.AgentRepository,
    taskService *TaskApplicationService,
    sessionService *SessionApplicationService,
    idGenerator domain.IDGenerator,
    replicaAgentManager *ReplicaAgentManager,
    hookExecutor *ConfigurableHookExecutor,
) *RequirementDispatchService {
    return &RequirementDispatchService{
        requirementRepo:       requirementRepo,
        projectRepo:          projectRepo,
        agentRepo:            agentRepo,
        taskService:          taskService,
        sessionService:       sessionService,
        idGenerator:          idGenerator,
        replicaAgentManager:  replicaAgentManager,
        hookExecutor:         hookExecutor,
    }
}
```

### 4.2 依赖注入配置

```go
// main.go

// 1. 创建 ReplicaAgentManager（强制销毁）
replicaAgentManager := NewReplicaAgentManager(agentRepo)

// 2. 创建 Hook 配置仓储
hookConfigRepo := persistence.NewSQLiteRequirementHookConfigRepository(db)
hookLogRepo := persistence.NewSQLiteRequirementHookActionLogRepository(db)

// 3. 创建动作执行器
triggerAgentExecutor := NewTriggerAgentExecutor(agentRepo, idGenerator, messageBus)
notificationExecutor := NewNotificationExecutor(notifyService)
webhookExecutor := NewWebhookExecutor(httpClient)

// 4. 创建可配置 Hook 执行器
hookExecutor := NewConfigurableHookExecutor(
    hookConfigRepo,
    hookLogRepo,
    []ActionExecutor{triggerAgentExecutor, notificationExecutor, webhookExecutor},
    logger,
)

// 5. 创建服务
dispatchService := NewRequirementDispatchService(
    requirementRepo,
    projectRepo,
    agentRepo,
    taskService,
    sessionService,
    idGenerator,
    replicaAgentManager,  // 强制销毁
    hookExecutor,         // 可配置 Hook
)

// 6. 在加载 Requirement 后设置管理器
requirement, _ := requirementRepo.FindByID(ctx, reqID)
requirement.SetReplicaAgentManager(replicaAgentManager)
```

## 5. 文件结构变更

```
backend/domain/
├── hook.go              # 新增 RequirementStateHook 接口（保留，用于内置 Hook）
├── requirement.go       # 添加 ReplicaAgentManager 注入和强制销毁

backend/infrastructure/
├── persistence/
│   ├── hook_config_repository.go      # 新增 Hook 配置仓储
│   └── hook_action_log_repository.go  # 新增执行日志仓储

backend/application/
├── requirement_dispatch_service.go  # 改造：注入管理器，执行 Hook

backend/interfaces/http/
├── hook_handler.go                 # 新增 Hook 配置管理 HTTP 接口
├── hook_config_handler.go          # 新增 Hook 配置 CRUD

frontend/src/
├── pages/
│   ├── HookConfigListPage.tsx       # 新增 Hook 配置列表页
│   ├── HookConfigEditPage.tsx       # 新增 Hook 配置编辑页
│   └── HookLogPage.tsx             # 新增 Hook 执行日志页
```

## 6. 执行流程图

### 6.1 状态变更完整流程

```
状态变更触发流程：

1. ApplicationService 调用领域方法
   └─ requirement.MarkFailed(error)
   或
   └─ requirement.MarkPROpened(prURL, branch)
           │
           ▼
2. 领域方法执行状态变更
   - 更新内部状态
   - 触发 fireStateChange()
           │
           ▼
3. 强制销毁分身（代码约束）
   └─ replicaAgentManager.EnsureDisposed(ctx, replicaAgentID, workspacePath)
   - 删除 Agent 记录
   - 清理工作目录
           │
           ▼
4. 触发可配置的 Hook（如果配置了）
   └─ hookExecutor.Execute(ctx, triggerPoint, req, change)
   - 从数据库加载配置
   - 按优先级执行动作
           │
           ▼
5. ActionExecutor 执行具体动作
   ├─ TriggerAgentExecutor: 创建新分身执行任务
   ├─ NotificationExecutor: 发送通知
   └─ WebhookExecutor: 发送 Webhook
```

## 7. API 设计

### 7.1 Hook 配置 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/hook-configs | 获取所有配置 |
| POST | /api/v1/hook-configs | 创建配置 |
| GET | /api/v1/hook-configs/:id | 获取单个配置 |
| PUT | /api/v1/hook-configs/:id | 更新配置 |
| DELETE | /api/v1/hook-configs/:id | 删除配置 |
| PATCH | /api/v1/hook-configs/:id/enable | 启用配置 |
| PATCH | /api/v1/hook-configs/:id/disable | 禁用配置 |

### 7.2 Hook 日志 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/hook-logs | 获取执行日志 |
| GET | /api/v1/hook-logs/:id | 获取单个日志详情 |

## 8. 测试策略

### 8.1 单元测试

```go
// ReplicaAgentManager 测试
func TestReplicaAgentManager_EnsureDisposed_Idempotent(t *testing.T) {
    manager := NewReplicaAgentManager(mockAgentRepo)
    // 调用多次应该和调用一次效果相同
    manager.EnsureDisposed(ctx, "agent-1", "/tmp/workspace")
    manager.EnsureDisposed(ctx, "agent-1", "/tmp/workspace")
    // 验证只删除一次
    assert.Equal(t, 1, len(mockAgentRepo.deletedIDs))
}

// ConfigurableHookExecutor 测试
func TestConfigurableHookExecutor_Execute_LoadsFromDB(t *testing.T) {
    executor := NewConfigurableHookExecutor(mockConfigRepo, mockLogRepo, executors, logger)
    executor.Execute(ctx, "mark_failed", req, change)
    // 验证从数据库加载了配置
    assert.True(t, mockConfigRepo.findEnabledByTriggerPointCalled)
}
```

### 8.2 集成测试

```go
func TestRequirement_MarkFailed_EnsuresReplicaDisposed(t *testing.T) {
    // 集成测试验证 MarkFailed 后分身被销毁
    req := createTestRequirement()
    req.SetReplicaAgentManager(manager)
    req.MarkCoding("/tmp/workspace", "agent-replica", "feature/test")

    req.MarkFailed("test error")

    // 验证分身被销毁
    mockAgentRepo.AssertDeleted("agent-replica")
}
```
