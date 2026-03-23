# Hook 系统架构设计

## 1. 设计目标

构建一个**可扩展、分层、事件驱动**的 Hook 系统，支持 LLM 执行过程中的所有细节作为 Hook，实现：

- 插件式架构：随意增减 Hook
- 分层设计：生命周期、LLM 调用、工具执行、消息处理等
- 事件驱动：各 Hook 独立，不相互依赖
- 同步/异步支持：按需选择
- 优先级控制：保证执行顺序

## 2. Hook 系统分层架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Hook System (顶层)                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ Lifecycle   │  │ LLM         │  │ Tool                │ │
│  │ Hooks       │  │ Hooks       │  │ Hooks               │ │
│  │ (5个)       │  │ (8个)       │  │ (10个)              │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ Message    │  │ Skill      │  │ MCP                  │ │
│  │ Hooks      │  │ Hooks       │  │ Hooks               │ │
│  │ (5个)      │  │ (6个)       │  │ (6个)               │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐                           │
│  │ Prompt     │  │ Session    │                           │
│  │ Hooks      │  │ Hooks       │                           │
│  │ (6个)       │  │ (5个)       │                           │
│  └─────────────┘  └─────────────┘                           │
└─────────────────────────────────────────────────────────────┘
```

## 3. Hook 事件分类与详细定义

### 3.1 Lifecycle Hooks (生命周期)

| Hook 名称 | 触发时机 | 输入参数 | 返回值 |
|-----------|---------|---------|--------|
| `OnInitialize` | 系统初始化时 | config | error |
| `OnShutdown` | 系统关闭时 | - | error |
| `OnStart` | 会话开始时 | session_id | error |
| `OnStop` | 会话结束时 | session_id, reason | error |
| `OnError` | 错误发生时 | error, context | error |

### 3.2 LLM Hooks (LLM 调用)

| Hook 名称 | 触发时机 | 输入参数 | 返回值 |
|-----------|---------|---------|--------|
| `PreLLMCall` | LLM 调用前 | prompt, model, params | ModifiedPrompt, error |
| `PostLLMCall` | LLM 调用后 | prompt, response, usage | ModifiedResponse, error |
| `PrePromptGeneration` | Prompt 生成前 | template, vars | ModifiedTemplate, error |
| `PostPromptGeneration` | Prompt 生成后 | final_prompt | ModifiedPrompt, error |
| `PreParseResponse` | 响应解析前 | raw_response | ModifiedRaw, error |
| `PostParseResponse` | 响应解析后 | parsed_response | ModifiedResponse, error |
| `OnLLMRetry` | LLM 重试时 | attempt, error | RetryConfig, error |
| `OnLLMTimeout` | LLM 超时时 | timeout, prompt | NewTimeout, error |

### 3.3 Tool Hooks (工具执行)

| Hook 名称 | 触发时机 | 输入参数 | 返回值 |
|-----------|---------|---------|--------|
| `PreToolCall` | 工具调用前 | tool_name, tool_input | ModifiedInput, error |
| `PostToolCall` | 工具调用后 | tool_result | ModifiedResult, error |
| `OnToolError` | 工具执行错误 | error, tool_name | RetryResult, error |
| `PreToolValidation` | 工具参数验证前 | tool_name, params | ModifiedParams, error |
| `OnToolRegistered` | 工具注册时 | tool_definition | error |
| `OnToolUnregistered` | 工具注销时 | tool_name | error |
| `PreToolExecution` | 工具实际执行前 | tool_context | ModifiedContext, error |
| `PostToolExecution` | 工具执行完成后 | execution_result | ModifiedResult, error |
| `OnToolCacheHit` | 工具缓存命中 | cache_key, result | ModifiedResult, error |
| `OnToolRateLimit` | 工具限流时 | tool_name, retry_after | WaitDuration, error |

### 3.4 Message Hooks (消息处理)

| Hook 名称 | 触发时机 | 输入参数 | 返回值 |
|-----------|---------|---------|--------|
| `PreMessageReceive` | 消息接收前 | raw_message | ModifiedMessage, error |
| `PostMessageReceive` | 消息接收后 | message | ModifiedMessage, error |
| `PreMessageSend` | 消息发送前 | message | ModifiedMessage, error |
| `PostMessageSend` | 消息发送后 | message | error |
| `OnMessageError` | 消息处理错误 | error, message | error |

### 3.5 Skill Hooks (技能系统)

| Hook 名称 | 触发时机 | 输入参数 | 返回值 |
|-----------|---------|---------|--------|
| `PreSkillInvoke` | Skill 调用前 | skill_name, params | ModifiedParams, error |
| `PostSkillInvoke` | Skill 调用后 | skill_result | ModifiedResult, error |
| `OnSkillError` | Skill 执行错误 | error, skill_name | error |
| `OnSkillLoaded` | Skill 加载时 | skill_definition | error |
| `OnSkillUnloaded` | Skill 卸载时 | skill_name | error |
| `PreSkillExecution` | Skill 实际执行前 | execution_context | ModifiedContext, error |

### 3.6 MCP Hooks (Model Context Protocol)

| Hook 名称 | 触发时机 | 输入参数 | 返回值 |
|-----------|---------|---------|--------|
| `PreMCPRequest` | MCP 请求前 | request_type, params | ModifiedParams, error |
| `PostMCPResponse` | MCP 响应后 | response | ModifiedResponse, error |
| `OnMCPError` | MCP 错误时 | error, request | error |
| `OnMCPServerStart` | MCP 服务器启动 | server_config | error |
| `OnMCPServerStop` | MCP 服务器停止 | server_name | error |
| `PreMCPStream` | MCP 流式请求前 | stream_data | ModifiedData, error |

### 3.7 Prompt Hooks (提示词系统)

| Hook 名称 | 触发时机 | 输入参数 | 返回值 |
|-----------|---------|---------|--------|
| `PrePromptRender` | Prompt 渲染前 | template, vars | ModifiedTemplate, error |
| `PostPromptRender` | Prompt 渲染后 | final_prompt | ModifiedPrompt, error |
| `PrePromptMerge` | Prompt 合并前 | prompts[] | ModifiedPrompts, error |
| `PostPromptMerge` | Prompt 合并后 | merged_prompt | ModifiedPrompt, error |
| `OnTemplateLoaded` | 模板加载时 | template_path | error |
| `OnTemplateError` | 模板错误时 | error, template | error |

### 3.8 Session Hooks (会话管理)

| Hook 名称 | 触发时机 | 输入参数 | 返回值 |
|-----------|---------|---------|--------|
| `PreSessionCreate` | 会话创建前 | session_config | ModifiedConfig, error |
| `PostSessionCreate` | 会话创建后 | session | error |
| `OnSessionResume` | 会话恢复时 | session_id | error |
| `OnSessionExpired` | 会话过期时 | session_id | error |
| `OnSessionSave` | 会话保存时 | session_state | error |

## 4. 核心接口设计

### 4.1 Hook 接口定义

```go
// Hook 基础接口
type Hook interface {
    // Name 返回 Hook 名称
    Name() string
    // Priority 返回优先级 (0-100, 越小越先执行)
    Priority() int
    // Enabled 是否启用
    Enabled() bool
}

// BaseHook 提供基础实现
type BaseHook struct {
    name     string
    priority int
    enabled  bool
}

func (h *BaseHook) Name() string     { return h.name }
func (h *BaseHook) Priority() int    { return h.priority }
func (h *BaseHook) Enabled() bool    { return h.enabled }
```

### 4.2 生命周期 Hook

```go
// LifecycleHook 生命周期钩子
type LifecycleHook interface {
    Hook
    OnInitialize(ctx context.Context, config *Config) error
    OnShutdown(ctx context.Context) error
    OnStart(ctx context.Context, sessionID string) error
    OnStop(ctx context.Context, sessionID string, reason string) error
    OnError(ctx context.Context, err error, details map[string]interface{}) error
}
```

### 4.3 LLM Hook

```go
// LLMCallContext LLM 调用上下文
type LLMCallContext struct {
    Prompt       string
    Model        string
    Temperature  float64
    MaxTokens    int
    StopSequences []string
    SystemPrompt string
    // ... 其他参数
}

// LLMResponse LLM 响应
type LLMResponse struct {
    Content      string
    Usage        Usage
    Model        string
    FinishReason string
    // ... 其他字段
}

// LLMHook LLM 钩子
type LLMHook interface {
    Hook
    PreLLMCall(ctx context.Context, callCtx *LLMCallContext) (*LLMCallContext, error)
    PostLLMCall(ctx context.Context, callCtx *LLMCallContext, response *LLMResponse) (*LLMResponse, error)
    PrePromptGeneration(ctx context.Context, template string, vars map[string]interface{}) (string, error)
    PostPromptGeneration(ctx context.Context, prompt string) (string, error)
    PreParseResponse(ctx context.Context, raw string) (string, error)
    PostParseResponse(ctx context.Context, parsed interface{}) (interface{}, error)
    OnLLMRetry(ctx context.Context, attempt int, err error) (*RetryConfig, error)
    OnLLMTimeout(ctx context.Context, timeout time.Duration, prompt string) (time.Duration, error)
}
```

### 4.4 Tool Hook

```go
// ToolCallContext 工具调用上下文
type ToolCallContext struct {
    ToolName string
    ToolInput map[string]interface{}
    SessionID string
    TraceID   string
    // ... 其他上下文
}

// ToolExecutionResult 工具执行结果
type ToolExecutionResult struct {
    Success bool
    Output  interface{}
    Error   error
    Duration time.Duration
    // ... 其他字段
}

// ToolHook 工具钩子
type ToolHook interface {
    Hook
    PreToolCall(ctx context.Context, callCtx *ToolCallContext) (*ToolCallContext, error)
    PostToolCall(ctx context.Context, callCtx *ToolCallContext, result *ToolExecutionResult) (*ToolExecutionResult, error)
    OnToolError(ctx context.Context, callCtx *ToolCallContext, err error) (*ToolExecutionResult, error)
    PreToolValidation(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)
    OnToolCacheHit(ctx context.Context, cacheKey string, result interface{}) (interface{}, error)
    OnToolRateLimit(ctx context.Context, toolName string, retryAfter time.Duration) (time.Duration, error)
}
```

## 5. Hook Registry 设计

```go
// Registry Hook 注册表
type Registry interface {
    // Register 注册 Hook
    Register(hook Hook) error
    // Unregister 注销 Hook
    Unregister(name string) error
    // Get 获取 Hook
    Get(name string) Hook
    // List 列出所有 Hook
    List() []Hook
    // ListByType 按类型获取 Hooks
    ListByType(hookType string) []Hook
}

// HookManager Hook 管理器
type HookManager struct {
    mu       sync.RWMutex
    registry map[string]Hook
    chains   map[string][]Hook // 按事件类型分组
}

func (m *HookManager) Register(hook Hook) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if _, exists := m.registry[hook.Name()]; exists {
        return fmt.Errorf("hook %s already registered", hook.Name())
    }

    m.registry[hook.Name()] = hook
    m.rebuildChains()
    return nil
}

func (m *HookManager) rebuildChains() {
    // 按优先级排序，重新构建执行链
}
```

## 6. Hook Executor 设计

```go
// ContextKey Hook 上下文键
type ContextKey string

const (
    HookContextSessionID ContextKey = "session_id"
    HookContextTraceID   ContextKey = "trace_id"
    HookContextRequestID ContextKey = "request_id"
)

// HookContext Hook 执行上下文
type HookContext struct {
    context.Context
    values map[ContextKey]interface{}
}

func (c *HookContext) WithValue(key ContextKey, val interface{}) *HookContext {
    c.values[key] = val
    return c
}

func (c *HookContext) Get(key ContextKey) interface{} {
    return c.values[key]
}

// Executor Hook 执行器
type Executor struct {
    registry Registry
    logger   *zap.Logger
}

func (e *Executor) ExecutePreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error) {
    hooks := e.registry.ListByType("pre_llm_call")
    var err error
    modifiedCtx := callCtx

    for _, hook := range hooks {
        if !hook.Enabled() {
            continue
        }
        llmHook := hook.(LLMHook)
        modifiedCtx, err = llmHook.PreLLMCall(ctx, modifiedCtx)
        if err != nil {
            return nil, fmt.Errorf("hook %s failed: %w", hook.Name(), err)
        }
    }

    return modifiedCtx, nil
}
```

## 7. 内置 Hook 实现

### 7.1 日志 Hook

```go
// LoggingHook 记录所有 Hook 调用日志
type LoggingHook struct {
    *BaseHook
    logger *zap.Logger
}

func (h *LoggingHook) PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error) {
    h.logger.Info("PreLLMCall",
        zap.String("model", callCtx.Model),
        zap.Int("prompt_len", len(callCtx.Prompt)))
    return callCtx, nil
}
```

### 7.2 指标 Hook

```go
// MetricsHook 收集指标
type MetricsHook struct {
    *BaseHook
    metrics *MetricsCollector
}

func (h *MetricsHook) PostLLMCall(ctx *HookContext, callCtx *LLMCallContext, resp *LLMResponse) (*LLMResponse, error) {
    h.metrics.Record("llm_call_duration", time.Since(ctx.Value(HookContextStartTime).(time.Time)))
    h.metrics.Increment("llm_call_total")
    return resp, nil
}
```

### 7.3 限流 Hook

```go
// RateLimitHook 限流
type RateLimitHook struct {
    *BaseHook
    limiter *rate.Limiter
}

func (h *RateLimitHook) PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error) {
    if !h.limiter.Allow() {
        return nil, ErrRateLimited
    }
    return callCtx, nil
}
```

## 8. 配置格式

```yaml
hooks:
  # 全局启用/禁用
  enabled: true

  # 各类型 Hook 配置
  lifecycle:
    - name: logging
      enabled: true
      priority: 100

  llm:
    - name: metrics
      enabled: true
      priority: 50
    - name: rate_limit
      enabled: true
      priority: 10
      config:
        max_calls_per_minute: 60

  tool:
    - name: tool_cache
      enabled: true
      priority: 20
      config:
        cache_ttl: 5m
```

## 9. 执行顺序示例

```
LLM 调用执行流程：

1. PreToolCall (Tool Hook)
   ↓
2. PreToolValidation (Tool Hook)
   ↓
3. PrePromptGeneration (LLM Hook)
   ↓
4. PrePromptRender (Prompt Hook)
   ↓
5. PostPromptRender (Prompt Hook)
   ↓
6. PostPromptGeneration (LLM Hook)
   ↓
7. PrePromptMerge (Prompt Hook)
   ↓
8. PreLLMCall (LLM Hook) ← 这里可以修改 prompt
   ↓
   [LLM Actual Call]
   ↓
9. PostLLMCall (LLM Hook) ← 这里可以修改 response
   ↓
10. PreParseResponse (LLM Hook)
    ↓
11. PostParseResponse (LLM Hook)
    ↓
12. PostToolCall (Tool Hook)
```

## 10. 错误处理策略

```go
// ErrorStrategy 错误处理策略
type ErrorStrategy int

const (
    // ErrorStrategyStopOnFirst 遇到错误立即停止
    ErrorStrategyStopOnFirst ErrorStrategy = iota
    // ErrorStrategyContinue 继续执行后续 Hook
    ErrorStrategyContinue
    // ErrorStrategySkipType 跳过同类型 Hook
    ErrorStrategySkipType
)

// HookExecutionResult 执行结果
type HookExecutionResult struct {
    Success   bool
    Results   []HookResult
    FinalError error
}
```

## 11. 未来扩展

- **异步 Hook 支持**：部分 Hook 可异步执行，不阻塞主流程
- **Hook 链式调用**：支持 Hook 返回值影响下一个 Hook 的输入
- **条件触发**：基于上下文条件决定是否触发 Hook
- **分布式 Hook**：支持跨进程的 Hook 调用
- **Hook 调试 UI**：可视化 Hook 执行状态和耗时

## 12. 实现计划

### Phase 1: 核心框架
- [ ] 定义 Hook 接口和基础结构
- [ ] 实现 Registry 和 Manager
- [ ] 实现基础 Executor
- [ ] 迁移现有 TaskHook 实现

### Phase 2: LLM Hooks
- [ ] 实现 PreLLMCall / PostLLMCall
- [ ] 实现 Prompt Generation Hooks
- [ ] 实现 Response Parsing Hooks

### Phase 3: Tool Hooks
- [ ] 实现 PreToolCall / PostToolCall
- [ ] 实现 Tool Validation Hooks
- [ ] 实现 Tool Error Handling

### Phase 4: 高级功能
- [ ] 异步 Hook 支持
- [ ] 条件触发
- [ ] 分布式 Hook
