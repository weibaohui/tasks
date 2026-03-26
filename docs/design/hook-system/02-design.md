# Hook 系统设计文档

## 1. 架构设计

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Hook System Architecture                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Hook Consumer (使用者)                         │   │
│  │   LLM Provider / Tool Executor / Message Handler / Skill System      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                      │                                      │
│                                      ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Hook Manager                                  │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐    │   │
│  │  │  Registry   │  │   Executor   │  │     Event Dispatcher      │    │   │
│  │  │  - Register │  │  - Execute  │  │  - Sync / Async          │    │   │
│  │  │  - Unreg    │  │  - Chain    │  │  - Error Handling        │    │   │
│  │  │  - Get/List │  │  - Priority │  │  - Context Propagation   │    │   │
│  │  └──────────────┘  └──────────────┘  └──────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                      │                                      │
│                                      ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Hook Categories                                 │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐                │   │
│  │  │Lifecycle │ │   LLM    │ │   Tool   │ │ Message  │                │   │
│  │  │  Hooks  │ │  Hooks   │ │  Hooks   │ │  Hooks   │                │   │
│  │  │   (5)   │ │   (8)   │ │  (10)   │ │   (5)   │                │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘                │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐                │   │
│  │  │  Skill   │ │   MCP    │ │ Prompt   │ │ Session  │                │   │
│  │  │  Hooks   │ │  Hooks   │ │  Hooks   │ │  Hooks   │                │   │
│  │  │   (6)   │ │   (6)   │ │   (6)   │ │   (5)   │                │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 模块职责

| 模块 | 职责 |
|------|------|
| Hook Manager | 统一管理所有 Hook，提供注册、注销、查询接口 |
| Registry | 存储和管理 Hook 实例，支持按类型、名称查询 |
| Executor | 执行 Hook 链，支持优先级排序、错误处理 |
| Event Dispatcher | 事件分发器，支持同步/异步分发 |
| Hook Categories | 按功能分类的 Hook 接口定义 |

## 2. 核心接口设计

### 2.1 Hook 基础接口

```go
// Hook 基础接口，所有 Hook 都必须实现
type Hook interface {
    // Name 返回 Hook 名称，唯一标识
    Name() string
    // Priority 返回优先级 (0-100, 越小越先执行)
    Priority() int
    // Enabled 返回是否启用
    Enabled() bool
    // HookType 返回 Hook 类型
    HookType() HookType
}

// BaseHook 提供基础实现
type BaseHook struct {
    name     string
    priority int
    enabled  bool
    hookType HookType
}

func (h *BaseHook) Name() string              { return h.name }
func (h *BaseHook) Priority() int             { return h.priority }
func (h *BaseHook) Enabled() bool             { return h.enabled }
func (h *BaseHook) HookType() HookType        { return h.hookType }
func (h *BaseHook) SetEnabled(b bool)         { h.enabled = b }

// HookType Hook 类型枚举
type HookType string

const (
    HookTypeLifecycle HookType = "lifecycle"
    HookTypeLLM       HookType = "llm"
    HookTypeTool      HookType = "tool"
    HookTypeMessage   HookType = "message"
    HookTypeSkill    HookType = "skill"
    HookTypeMCP      HookType = "mcp"
    HookTypePrompt   HookType = "prompt"
    HookTypeSession  HookType = "session"
)
```

### 2.2 Lifecycle Hook 接口

```go
// LifecycleHook 生命周期钩子
type LifecycleHook interface {
    Hook
    // OnInitialize 系统初始化时调用
    OnInitialize(ctx context.Context, config *Config) error
    // OnShutdown 系统关闭时调用
    OnShutdown(ctx context.Context) error
    // OnStart 会话开始时调用
    OnStart(ctx context.Context, sessionID string) error
    // OnStop 会话结束时调用
    OnStop(ctx context.Context, sessionID string, reason string) error
    // OnError 错误发生时调用
    OnError(ctx context.Context, err error, details map[string]interface{}) error
}
```

### 2.3 LLM Hook 接口

```go
// LLMCallContext LLM 调用上下文
type LLMCallContext struct {
    Prompt        string
    Model         string
    Temperature   float64
    MaxTokens     int
    StopSequences []string
    SystemPrompt  string
    SessionID     string
    TraceID       string
}

// LLMResponse LLM 响应
type LLMResponse struct {
    Content      string
    Usage        Usage
    Model        string
    FinishReason string
    RawResponse  string
}

// Usage token 使用量
type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens     int
}

// RetryConfig 重试配置
type RetryConfig struct {
    MaxAttempts int
    Delay      time.Duration
    Backoff    float64
}

// LLMHook LLM 钩子接口
type LLMHook interface {
    Hook
    // PreLLMCall LLM 调用前，可修改调用参数
    PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error)
    // PostLLMCall LLM 调用后，可修改响应
    PostLLMCall(ctx *HookContext, callCtx *LLMCallContext, response *LLMResponse) (*LLMResponse, error)
    // PrePromptGeneration Prompt 生成前
    PrePromptGeneration(ctx *HookContext, template string, vars map[string]interface{}) (string, error)
    // PostPromptGeneration Prompt 生成后
    PostPromptGeneration(ctx *HookContext, prompt string) (string, error)
    // PreParseResponse 响应解析前
    PreParseResponse(ctx *HookContext, raw string) (string, error)
    // PostParseResponse 响应解析后
    PostParseResponse(ctx *HookContext, parsed interface{}) (interface{}, error)
    // OnLLMRetry LLM 重试时
    OnLLMRetry(ctx *HookContext, attempt int, err error) (*RetryConfig, error)
    // OnLLMTimeout LLM 超时时
    OnLLMTimeout(ctx *HookContext, timeout time.Duration, prompt string) (time.Duration, error)
}
```

### 2.4 Tool Hook 接口

```go
// ToolCallContext 工具调用上下文
type ToolCallContext struct {
    ToolName  string
    ToolInput map[string]interface{}
    SessionID string
    TraceID   string
    SpanID    string
    ParentSpanID string
}

// ToolDefinition 工具定义
type ToolDefinition struct {
    Name        string
    Description string
    Parameters  map[string]interface{}
}

// ToolExecutionResult 工具执行结果
type ToolExecutionResult struct {
    Success   bool
    Output    interface{}
    Error     error
    Duration  time.Duration
    CacheHit  bool
    SpanID    string
}

// ToolHook 工具钩子接口
type ToolHook interface {
    Hook
    // PreToolCall 工具调用前，可修改输入参数
    PreToolCall(ctx *HookContext, callCtx *ToolCallContext) (*ToolCallContext, error)
    // PostToolCall 工具调用后，可修改结果
    PostToolCall(ctx *HookContext, callCtx *ToolCallContext, result *ToolExecutionResult) (*ToolExecutionResult, error)
    // OnToolError 工具执行错误
    OnToolError(ctx *HookContext, callCtx *ToolCallContext, err error) (*ToolExecutionResult, error)
    // PreToolValidation 工具参数验证前
    PreToolValidation(ctx *HookContext, toolName string, params map[string]interface{}) (map[string]interface{}, error)
    // OnToolRegistered 工具注册时
    OnToolRegistered(ctx *HookContext, def *ToolDefinition) error
    // OnToolUnregistered 工具注销时
    OnToolUnregistered(ctx *HookContext, toolName string) error
    // PreToolExecution 工具实际执行前
    PreToolExecution(ctx *HookContext, callCtx *ToolCallContext) (*ToolCallContext, error)
    // PostToolExecution 工具执行完成后
    PostToolExecution(ctx *HookContext, callCtx *ToolCallContext, result *ToolExecutionResult) (*ToolExecutionResult, error)
    // OnToolCacheHit 工具缓存命中
    OnToolCacheHit(ctx *HookContext, cacheKey string, result interface{}) (interface{}, error)
    // OnToolRateLimit 工具限流时
    OnToolRateLimit(ctx *HookContext, toolName string, retryAfter time.Duration) (time.Duration, error)
}
```

### 2.5 其他 Hook 接口（简化版）

```go
// MessageHook 消息钩子接口
type MessageHook interface {
    Hook
    PreMessageReceive(ctx *HookContext, raw []byte) ([]byte, error)
    PostMessageReceive(ctx *HookContext, msg *Message) (*Message, error)
    PreMessageSend(ctx *HookContext, msg *Message) (*Message, error)
    PostMessageSend(ctx *HookContext, msg *Message) error
    OnMessageError(ctx *HookContext, err error, msg *Message) error
}

// SkillHook 技能钩子接口
type SkillHook interface {
    Hook
    PreSkillInvoke(ctx *HookContext, skillName string, params map[string]interface{}) (string, map[string]interface{}, error)
    PostSkillInvoke(ctx *HookContext, skillName string, result interface{}) (interface{}, error)
    OnSkillError(ctx *HookContext, err error, skillName string) error
    OnSkillLoaded(ctx *HookContext, skillDef *SkillDefinition) error
    OnSkillUnloaded(ctx *HookContext, skillName string) error
    PreSkillExecution(ctx *HookContext, execCtx *SkillExecutionContext) (*SkillExecutionContext, error)
}

// MCPHook MCP 钩子接口
type MCPHook interface {
    Hook
    PreMCPRequest(ctx *HookContext, reqType string, params map[string]interface{}) (string, map[string]interface{}, error)
    PostMCPResponse(ctx *HookContext, resp *MCPResponse) (*MCPResponse, error)
    OnMCPError(ctx *HookContext, err error, req *MCPRequest) error
    OnMCPServerStart(ctx *HookContext, config *MCPServerConfig) error
    OnMCPServerStop(ctx *HookContext, serverName string) error
    PreMCPStream(ctx *HookContext, streamData []byte) ([]byte, error)
}

// PromptHook Prompt 钩子接口
type PromptHook interface {
    Hook
    PrePromptRender(ctx *HookContext, template string, vars map[string]interface{}) (string, map[string]interface{}, error)
    PostPromptRender(ctx *HookContext, prompt string) (string, error)
    PrePromptMerge(ctx *HookContext, prompts []string) ([]string, error)
    PostPromptMerge(ctx *HookContext, merged string) (string, error)
    OnTemplateLoaded(ctx *HookContext, path string) error
    OnTemplateError(ctx *HookContext, err error, template string) error
}

// SessionHook 会话钩子接口
type SessionHook interface {
    Hook
    PreSessionCreate(ctx *HookContext, config *SessionConfig) (*SessionConfig, error)
    PostSessionCreate(ctx *HookContext, session *Session) error
    OnSessionResume(ctx *HookContext, sessionID string) error
    OnSessionExpired(ctx *HookContext, sessionID string) error
    OnSessionSave(ctx *HookContext, state *SessionState) error
}
```

## 3. Hook Context 设计

```go
// HookContext 钩子执行上下文
type HookContext struct {
    context.Context
    values     map[interface{}]interface{}
    hooks      []string          // 已执行的 Hook 名称列表
    errors     []HookError       // 执行过程中的错误
    metadata   map[string]string // 元数据
}

func NewHookContext(ctx context.Context) *HookContext {
    return &HookContext{
        Context: ctx,
        values:  make(map[interface{}]interface{}),
        hooks:   make([]string, 0),
        errors:  make([]HookError, 0),
        metadata: make(map[string]string),
    }
}

// WithValue 设置上下文值
func (c *HookContext) WithValue(key, val interface{}) *HookContext {
    c.values[key] = val
    return c
}

// Get 获取上下文值
func (c *HookContext) Get(key interface{}) interface{} {
    return c.values[key]
}

// AddHook 记录已执行的 Hook
func (c *HookContext) AddHook(name string) {
    c.hooks = append(c.hooks, name)
}

// GetHooks 获取已执行的 Hook 列表
func (c *HookContext) GetHooks() []string {
    return c.hooks
}

// AddError 添加错误
func (c *HookContext) AddError(err error, hookName string) {
    c.errors = append(c.errors, HookError{Err: err, HookName: hookName})
}

// GetErrors 获取所有错误
func (c *HookContext) GetErrors() []HookError {
    return c.errors
}

// SetMetadata 设置元数据
func (c *HookContext) SetMetadata(key, val string) {
    c.metadata[key] = val
}

// GetMetadata 获取元数据
func (c *HookContext) GetMetadata(key string) string {
    return c.metadata[key]
}

// HookError Hook 执行错误
type HookError struct {
    Err      error
    HookName string
    Phase    string // "pre", "post", "main"
}
```

## 4. Registry 设计

```go
// Registry Hook 注册表接口
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
    ListByType(hookType HookType) []Hook
    // ListByEvent 按事件类型获取 Hooks
    ListByEvent(event string) []Hook
    // Enable 启用 Hook
    Enable(name string) error
    // Disable 禁用 Hook
    Disable(name string) error
    // Clear 清空所有 Hook
    Clear()
}

// hookRegistry 注册表实现
type hookRegistry struct {
    mu    sync.RWMutex
    hooks map[string]Hook
}

func NewRegistry() Registry {
    return &hookRegistry{
        hooks: make(map[string]Hook),
    }
}

func (r *hookRegistry) Register(hook Hook) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if hook == nil {
        return errors.New("hook cannot be nil")
    }

    if _, exists := r.hooks[hook.Name()]; exists {
        return fmt.Errorf("hook %s already registered", hook.Name())
    }

    r.hooks[hook.Name()] = hook
    return nil
}

func (r *hookRegistry) Unregister(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.hooks[name]; !exists {
        return fmt.Errorf("hook %s not found", name)
    }

    delete(r.hooks, name)
    return nil
}

func (r *hookRegistry) Get(name string) Hook {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.hooks[name]
}

func (r *hookRegistry) List() []Hook {
    r.mu.RLock()
    defer r.mu.RUnlock()

    hooks := make([]Hook, 0, len(r.hooks))
    for _, hook := range r.hooks {
        hooks = append(hooks, hook)
    }
    return hooks
}

func (r *hookRegistry) ListByType(hookType HookType) []Hook {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var hooks []Hook
    for _, hook := range r.hooks {
        if hook.HookType() == hookType {
            hooks = append(hooks, hook)
        }
    }
    return hooks
}

func (r *hookRegistry) Enable(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    hook, exists := r.hooks[name]
    if !exists {
        return fmt.Errorf("hook %s not found", name)
    }
    hook.SetEnabled(true)
    return nil
}

func (r *hookRegistry) Disable(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    hook, exists := r.hooks[name]
    if !exists {
        return fmt.Errorf("hook %s not found", name)
    }
    hook.SetEnabled(false)
    return nil
}

func (r *hookRegistry) Clear() {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.hooks = make(map[string]Hook)
}
```

## 5. Executor 设计

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

// Executor Hook 执行器
type Executor struct {
    registry      Registry
    logger        *zap.Logger
    errorStrategy ErrorStrategy
    asyncPool     *workerPool
}

func NewExecutor(registry Registry, logger *zap.Logger) *Executor {
    return &Executor{
        registry:      registry,
        logger:        logger,
        errorStrategy: ErrorStrategyContinue,
        asyncPool:     newWorkerPool(10), // 10 个 worker
    }
}

// ExecutePreLLMCall 执行 PreLLMCall 钩子
func (e *Executor) ExecutePreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error) {
    hooks := e.getEnabledHooks(HookTypeLLM, "pre_llm_call")
    hooks = e.sortByPriority(hooks)

    modifiedCtx := callCtx
    for _, hook := range hooks {
        llmHook := hook.(LLMHook)
        e.logger.Debug("executing PreLLMCall", zap.String("hook", hook.Name()))

        result, err := llmHook.PreLLMCall(ctx, modifiedCtx)
        if err != nil {
            e.logger.Error("PreLLMCall failed",
                zap.String("hook", hook.Name()),
                zap.Error(err))

            if e.errorStrategy == ErrorStrategyStopOnFirst {
                return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
            }
            ctx.AddError(err, hook.Name())
            continue
        }

        if result != nil {
            modifiedCtx = result
        }
        ctx.AddHook(hook.Name())
    }

    return modifiedCtx, nil
}

// ExecutePostLLMCall 执行 PostLLMCall 钩子
func (e *Executor) ExecutePostLLMCall(ctx *HookContext, callCtx *LLMCallContext, resp *LLMResponse) (*LLMResponse, error) {
    hooks := e.getEnabledHooks(HookTypeLLM, "post_llm_call")
    hooks = e.sortByPriority(hooks)

    modifiedResp := resp
    for _, hook := range hooks {
        llmHook := hook.(LLMHook)
        e.logger.Debug("executing PostLLMCall", zap.String("hook", hook.Name()))

        result, err := llmHook.PostLLMCall(ctx, callCtx, modifiedResp)
        if err != nil {
            e.logger.Error("PostLLMCall failed",
                zap.String("hook", hook.Name()),
                zap.Error(err))

            if e.errorStrategy == ErrorStrategyStopOnFirst {
                return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
            }
            ctx.AddError(err, hook.Name())
            continue
        }

        if result != nil {
            modifiedResp = result
        }
        ctx.AddHook(hook.Name())
    }

    return modifiedResp, nil
}

// getEnabledHooks 获取指定类型的启用状态的 Hooks
func (e *Executor) getEnabledHooks(hookType HookType, eventName string) []Hook {
    hooks := e.registry.ListByType(hookType)
    var enabled []Hook
    for _, hook := range hooks {
        if hook.Enabled() {
            enabled = append(enabled, hook)
        }
    }
    return enabled
}

// sortByPriority 按优先级排序
func (e *Executor) sortByPriority(hooks []Hook) []Hook {
    sort.Slice(hooks, func(i, j int) bool {
        return hooks[i].Priority() < hooks[j].Priority()
    })
    return hooks
}

// ExecuteAsync 异步执行 Hook
func (e *Executor) ExecuteAsync(ctx *HookContext, fn func() error) {
    e.asyncPool.Submit(func() {
        if err := fn(); err != nil {
            e.logger.Error("async hook execution failed", zap.Error(err))
        }
    })
}
```

## 6. Manager 设计

```go
// Manager Hook 管理器
type Manager struct {
    registry  Registry
    executor  *Executor
    publisher *eventPublisher
    logger    *zap.Logger
    config    *ManagerConfig
}

type ManagerConfig struct {
    ErrorStrategy   ErrorStrategy
    AsyncPoolSize   int
    EnableMetrics   bool
    EnableLogging   bool
}

// NewManager 创建 Manager
func NewManager(config *ManagerConfig, logger *zap.Logger) *Manager {
    registry := NewRegistry()
    executor := NewExecutor(registry, logger)

    if config == nil {
        config = &ManagerConfig{
            ErrorStrategy: ErrorStrategyContinue,
            AsyncPoolSize: 10,
            EnableMetrics: true,
            EnableLogging: true,
        }
    }

    return &Manager{
        registry:  registry,
        executor:  executor,
        publisher: newEventPublisher(),
        logger:    logger,
        config:    config,
    }
}

// Register 注册 Hook
func (m *Manager) Register(hook Hook) error {
    return m.registry.Register(hook)
}

// Unregister 注销 Hook
func (m *Manager) Unregister(name string) error {
    return m.registry.Unregister(name)
}

// PreLLMCall 执行 PreLLMCall 钩子
func (m *Manager) PreLLMCall(ctx context.Context, callCtx *LLMCallContext) (*LLMCallContext, error) {
    hookCtx := NewHookContext(ctx)
    return m.executor.ExecutePreLLMCall(hookCtx, callCtx)
}

// PostLLMCall 执行 PostLLMCall 钩子
func (m *Manager) PostLLMCall(ctx context.Context, callCtx *LLMCallContext, resp *LLMResponse) (*LLMResponse, error) {
    hookCtx := NewHookContext(ctx)
    return m.executor.ExecutePostLLMCall(hookCtx, callCtx, resp)
}

// PreToolCall 执行 PreToolCall 钩子
func (m *Manager) PreToolCall(ctx context.Context, callCtx *ToolCallContext) (*ToolCallContext, error) {
    hookCtx := NewHookContext(ctx)
    return m.executor.ExecutePreToolCall(hookCtx, callCtx)
}

// PostToolCall 执行 PostToolCall 钩子
func (m *Manager) PostToolCall(ctx context.Context, callCtx *ToolCallContext, result *ToolExecutionResult) (*ToolExecutionResult, error) {
    hookCtx := NewHookContext(ctx)
    return m.executor.ExecutePostToolCall(hookCtx, callCtx, result)
}
```

## 7. 配置格式

```yaml
hook_system:
  # 全局配置
  enabled: true
  error_strategy: "continue"  # stop_on_first, continue, skip_type

  # 各类型 Hook 配置
  lifecycle:
    - name: logging
      enabled: true
      priority: 100
      config:
        log_level: "info"

  llm:
    - name: metrics
      enabled: true
      priority: 50
      config:
        record_usage: true
    - name: rate_limit
      enabled: true
      priority: 10
      config:
        max_calls_per_minute: 60
    - name: cache
      enabled: true
      priority: 20
      config:
        cache_ttl: 5m
        cache_size: 1000

  tool:
    - name: tool_cache
      enabled: true
      priority: 20
      config:
        cache_ttl: 5m
    - name: tool_logging
      enabled: true
      priority: 100
```

## 8. 执行流程图

### 8.1 LLM 调用完整流程

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
8. PostPromptMerge (Prompt Hook)
   ↓
9. PreLLMCall (LLM Hook) ← 这里可以修改 prompt
   ↓
   [LLM Actual Call]
   ↓
10. PostLLMCall (LLM Hook) ← 这里可以修改 response
    ↓
11. PreParseResponse (LLM Hook)
    ↓
12. PostParseResponse (LLM Hook)
    ↓
13. PostToolCall (Tool Hook)
```

### 8.2 工具调用完整流程

```
工具调用执行流程：

1. PreToolCall
   ↓
2. PreToolValidation
   ↓
3. PreToolExecution
   ↓
   [Tool Actual Execution]
   ↓
4. PostToolExecution
   ↓
5. PostToolCall
```

## 9. 错误处理策略

```go
// HookExecutionResult Hook 执行结果
type HookExecutionResult struct {
    Success     bool
    HookResults []HookResult
    FinalError  error
    Duration    time.Duration
}

// HookResult 单个 Hook 执行结果
type HookResult struct {
    HookName   string
    Success    bool
    Error      error
    Duration   time.Duration
    Modified   bool  // 是否修改了输入/输出
}

// 执行策略示例

// ErrorStrategyStopOnFirst: 遇到错误立即停止
func (e *Executor) executeStopOnFirst(hooks []Hook, ctx *HookContext) (*HookExecutionResult, error) {
    for _, hook := range hooks {
        result, err := e.executeHook(hook, ctx)
        if err != nil {
            return result, err
        }
    }
    return &HookExecutionResult{Success: true}, nil
}

// ErrorStrategyContinue: 继续执行所有 Hook
func (e *Executor) executeContinue(hooks []Hook, ctx *HookContext) *HookExecutionResult {
    result := &HookExecutionResult{Success: true}
    for _, hook := range hooks {
        hookResult, _ := e.executeHook(hook, ctx)
        result.HookResults = append(result.HookResults, *hookResult)
        if hookResult.Error != nil {
            result.Success = false
        }
    }
    return result
}
```

## 10. 内置 Hook 实现

### 10.1 日志 Hook

```go
// LoggingHook 记录所有 Hook 调用日志
type LoggingHook struct {
    *BaseHook
    logger  *zap.Logger
    logLevel string
}

func NewLoggingHook(logger *zap.Logger) *LoggingHook {
    return &LoggingHook{
        BaseHook: &BaseHook{
            name:     "logging",
            priority: 100,
            enabled:  true,
            hookType: HookTypeLifecycle,
        },
        logger:   logger,
        logLevel: "info",
    }
}

func (h *LoggingHook) OnInitialize(ctx *HookContext, config *Config) error {
    h.logger.Info("Hook system initializing", zap.Any("config", config))
    return nil
}

func (h *LoggingHook) PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error) {
    h.logger.Debug("PreLLMCall",
        zap.String("model", callCtx.Model),
        zap.Int("prompt_len", len(callCtx.Prompt)))
    return callCtx, nil
}

func (h *LoggingHook) PostLLMCall(ctx *HookContext, callCtx *LLMCallContext, resp *LLMResponse) (*LLMResponse, error) {
    h.logger.Debug("PostLLMCall",
        zap.String("model", resp.Model),
        zap.Int("content_len", len(resp.Content)),
        zap.Int("total_tokens", resp.Usage.TotalTokens))
    return resp, nil
}
```

### 10.2 指标 Hook

```go
// MetricsHook 收集指标
type MetricsHook struct {
    *BaseHook
    metrics    *MetricsCollector
    startTimes map[string]time.Time
    mu         sync.Mutex
}

func NewMetricsHook() *MetricsHook {
    return &MetricsHook{
        BaseHook: &BaseHook{
            name:     "metrics",
            priority: 50,
            enabled:  true,
            hookType: HookTypeLLM,
        },
        metrics:    NewMetricsCollector(),
        startTimes: make(map[string]time.Time),
    }
}

func (h *MetricsHook) PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error) {
    h.mu.Lock()
    h.startTimes["llm"] = time.Now()
    h.mu.Unlock()

    h.metrics.Increment("llm_call_total")
    h.metrics.Set("llm_model", callCtx.Model)
    return callCtx, nil
}

func (h *MetricsHook) PostLLMCall(ctx *HookContext, callCtx *LLMCallContext, resp *LLMResponse) (*LLMResponse, error) {
    h.mu.Lock()
    if start, ok := h.startTimes["llm"]; ok {
        h.metrics.Record("llm_call_duration", time.Since(start))
        delete(h.startTimes, "llm")
    }
    h.mu.Unlock()

    h.metrics.Set("llm_prompt_tokens", resp.Usage.PromptTokens)
    h.metrics.Set("llm_completion_tokens", resp.Usage.CompletionTokens)
    h.metrics.Set("llm_total_tokens", resp.Usage.TotalTokens)
    return resp, nil
}
```

### 10.3 缓存 Hook

```go
// CacheHook 缓存 LLM 响应
type CacheHook struct {
    *BaseHook
    cache  *ccache.Cache
    ttl    time.Duration
}

func NewCacheHook(ttl time.Duration) *CacheHook {
    return &CacheHook{
        BaseHook: &BaseHook{
            name:     "cache",
            priority: 10, // 优先执行，在 PreLLMCall 中检查缓存
            enabled:  true,
            hookType: HookTypeLLM,
        },
        cache: ccache.New(ccache.Configure().MaxSize(1000)),
        ttl:   ttl,
    }
}

func (h *CacheHook) PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error) {
    // 生成缓存 key
    cacheKey := h.generateCacheKey(callCtx)

    // 尝试获取缓存
    if cached, found := h.cache.Get(cacheKey); found {
        h.logger.Info("cache hit", zap.String("key", cacheKey))
        // 直接返回缓存的响应，绕过实际 LLM 调用
        ctx.Set(cacheKeyContextKey, cached.(*LLMResponse))
        return callCtx, nil
    }

    return callCtx, nil
}

func (h *CacheHook) PostLLMCall(ctx *HookContext, callCtx *LLMCallContext, resp *LLMResponse) (*LLMResponse, error) {
    cacheKey := h.generateCacheKey(callCtx)
    h.cache.Set(cacheKey, resp, h.ttl)
    return resp, nil
}

func (h *CacheHook) generateCacheKey(callCtx *LLMCallContext) string {
    h := md5.New()
    h.Write([]byte(callCtx.Prompt))
    h.Write([]byte(callCtx.Model))
    return hex.EncodeToString(h.Sum(nil))
}
```

### 10.4 限流 Hook

```go
// RateLimitHook 限流
type RateLimitHook struct {
    *BaseHook
    limiter *rate.Limiter
    burst   int
}

func NewRateLimitHook(limit rate.Limit, burst int) *RateLimitHook {
    return &RateLimitHook{
        BaseHook: &BaseHook{
            name:     "rate_limit",
            priority: 5, // 最先执行
            enabled:  true,
            hookType: HookTypeLLM,
        },
        limiter: rate.NewLimiter(limit, burst),
        burst:   burst,
    }
}

func (h *RateLimitHook) PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error) {
    if !h.limiter.Allow() {
        return nil, ErrRateLimited
    }
    return callCtx, nil
}
```
