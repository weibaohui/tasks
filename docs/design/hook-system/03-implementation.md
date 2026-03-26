# Hook 系统实现文档

## 1. 项目结构

```
backend/
├── domain/
│   └── hook.go              # Hook 核心接口定义
├── infrastructure/
│   └── hook/
│       ├── registry.go      # Hook 注册表实现
│       ├── executor.go      # Hook 执行器实现
│       ├── manager.go       # Hook 管理器实现
│       ├── context.go       # Hook 上下文实现
│       ├── config.go        # Hook 配置定义
│       └── hooks/
│           ├── logging.go       # 日志 Hook
│           ├── metrics.go       # 指标 Hook
│           ├── cache.go         # 缓存 Hook
│           └── rate_limit.go    # 限流 Hook
└── application/
    └── llm_hook_integration.go  # LLM 集成
```

## 2. 核心实现

### 2.1 Hook 接口定义 (domain/hook.go)

```go
package domain

import (
    "context"
    "time"
)

// Hook 基础接口
type Hook interface {
    Name() string
    Priority() int
    Enabled() bool
    SetEnabled(bool)
    HookType() HookType
}

// HookType Hook 类型
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

// BaseHook 基础实现
type BaseHook struct {
    name     string
    priority int
    enabled  bool
    hookType HookType
}

func (h *BaseHook) Name() string              { return h.name }
func (h *BaseHook) Priority() int             { return h.priority }
func (h *BaseHook) Enabled() bool            { return h.enabled }
func (h *BaseHook) SetEnabled(b bool)         { h.enabled = b }
func (h *BaseHook) HookType() HookType       { return h.hookType }

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
    TotalTokens      int
}

// LLMHook LLM 钩子接口
type LLMHook interface {
    Hook
    PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error)
    PostLLMCall(ctx *HookContext, callCtx *LLMCallContext, response *LLMResponse) (*LLMResponse, error)
}

// ToolCallContext 工具调用上下文
type ToolCallContext struct {
    ToolName      string
    ToolInput     map[string]interface{}
    SessionID     string
    TraceID       string
    SpanID        string
    ParentSpanID  string
}

// ToolExecutionResult 工具执行结果
type ToolExecutionResult struct {
    Success  bool
    Output   interface{}
    Error    error
    Duration time.Duration
    CacheHit bool
    SpanID   string
}

// ToolHook 工具钩子接口
type ToolHook interface {
    Hook
    PreToolCall(ctx *HookContext, callCtx *ToolCallContext) (*ToolCallContext, error)
    PostToolCall(ctx *HookContext, callCtx *ToolCallContext, result *ToolExecutionResult) (*ToolExecutionResult, error)
    OnToolError(ctx *HookContext, callCtx *ToolCallContext, err error) (*ToolExecutionResult, error)
}
```

### 2.2 Hook Context (infrastructure/hook/context.go)

```go
package hook

import (
    "context"
    "sync"
    "time"
)

// HookContext 钩子执行上下文
type HookContext struct {
    context.Context
    values    map[interface{}]interface{}
    hooks     []string
    errors    []HookError
    metadata  map[string]string
    startTime time.Time
    mu        sync.RWMutex
}

type ContextKey string

const (
    ContextKeySessionID ContextKey = "session_id"
    ContextKeyTraceID   ContextKey = "trace_id"
    ContextKeyRequestID ContextKey = "request_id"
)

type HookError struct {
    Err      error
    HookName string
    Phase    string
}

func NewHookContext(ctx context.Context) *HookContext {
    return &HookContext{
        Context:   ctx,
        values:    make(map[interface{}]interface{}),
        hooks:     make([]string, 0),
        errors:    make([]HookError, 0),
        metadata:  make(map[string]string),
        startTime: time.Now(),
    }
}

func (c *HookContext) WithValue(key, val interface{}) *HookContext {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.values[key] = val
    return c
}

func (c *HookContext) Get(key interface{}) interface{} {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.values[key]
}

func (c *HookContext) AddHook(name string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.hooks = append(c.hooks, name)
}

func (c *HookContext) GetHooks() []string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    result := make([]string, len(c.hooks))
    copy(result, c.hooks)
    return result
}

func (c *HookContext) AddError(err error, hookName, phase string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.errors = append(c.errors, HookError{Err: err, HookName: hookName, Phase: phase})
}

func (c *HookContext) GetErrors() []HookError {
    c.mu.RLock()
    defer c.mu.RUnlock()
    result := make([]HookError, len(c.errors))
    copy(result, c.errors)
    return result
}

func (c *HookContext) HasErrors() bool {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return len(c.errors) > 0
}

func (c *HookContext) SetMetadata(key, val string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.metadata[key] = val
}

func (c *HookContext) GetMetadata(key string) string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.metadata[key]
}

func (c *HookContext) Duration() time.Duration {
    return time.Since(c.startTime)
}
```

### 2.3 Registry (infrastructure/hook/registry.go)

```go
package hook

import (
    "errors"
    "fmt"
    "sync"

    "github.com/weibh/taskmanager/domain"
)

// Registry Hook 注册表
type Registry interface {
    Register(hook domain.Hook) error
    Unregister(name string) error
    Get(name string) domain.Hook
    List() []domain.Hook
    ListByType(hookType domain.HookType) []domain.Hook
    Enable(name string) error
    Disable(name string) error
    Clear()
}

// SimpleRegistry 简单注册表实现
type SimpleRegistry struct {
    mu    sync.RWMutex
    hooks map[string]domain.Hook
}

func NewRegistry() Registry {
    return &SimpleRegistry{
        hooks: make(map[string]domain.Hook),
    }
}

func (r *SimpleRegistry) Register(hook domain.Hook) error {
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

func (r *SimpleRegistry) Unregister(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.hooks[name]; !exists {
        return fmt.Errorf("hook %s not found", name)
    }

    delete(r.hooks, name)
    return nil
}

func (r *SimpleRegistry) Get(name string) domain.Hook {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.hooks[name]
}

func (r *SimpleRegistry) List() []domain.Hook {
    r.mu.RLock()
    defer r.mu.RUnlock()

    hooks := make([]domain.Hook, 0, len(r.hooks))
    for _, hook := range r.hooks {
        hooks = append(hooks, hook)
    }
    return hooks
}

func (r *SimpleRegistry) ListByType(hookType domain.HookType) []domain.Hook {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var hooks []domain.Hook
    for _, hook := range r.hooks {
        if hook.HookType() == hookType {
            hooks = append(hooks, hook)
        }
    }
    return hooks
}

func (r *SimpleRegistry) Enable(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    hook, exists := r.hooks[name]
    if !exists {
        return fmt.Errorf("hook %s not found", name)
    }
    hook.SetEnabled(true)
    return nil
}

func (r *SimpleRegistry) Disable(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    hook, exists := r.hooks[name]
    if !exists {
        return fmt.Errorf("hook %s not found", name)
    }
    hook.SetEnabled(false)
    return nil
}

func (r *SimpleRegistry) Clear() {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.hooks = make(map[string]domain.Hook)
}
```

### 2.4 Executor (infrastructure/hook/executor.go)

```go
package hook

import (
    "fmt"
    "sort"

    "github.com/weibh/taskmanager/domain"
    "go.uber.org/zap"
)

// ErrorStrategy 错误处理策略
type ErrorStrategy int

const (
    ErrorStrategyStopOnFirst ErrorStrategy = iota
    ErrorStrategyContinue
)

// Executor Hook 执行器
type Executor struct {
    registry      Registry
    logger        *zap.Logger
    errorStrategy ErrorStrategy
}

func NewExecutor(registry Registry, logger *zap.Logger) *Executor {
    return &Executor{
        registry:      registry,
        logger:        logger,
        errorStrategy: ErrorStrategyContinue,
    }
}

func (e *Executor) SetErrorStrategy(strategy ErrorStrategy) {
    e.errorStrategy = strategy
}

// ExecutePreLLMCall 执行 PreLLMCall 钩子
func (e *Executor) ExecutePreLLMCall(ctx *HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
    hooks := e.getEnabledHooks(domain.HookTypeLLM)
    hooks = e.sortByPriority(hooks)

    var filterFunc func(domain.Hook) bool = func(h domain.Hook) bool {
        _, ok := h.(domain.LLMHook)
        return ok
    }

    modifiedCtx := callCtx
    for _, hook := range hooks {
        if !filterFunc(hook) {
            continue
        }

        llmHook := hook.(domain.LLMHook)
        e.logger.Debug("executing PreLLMCall",
            zap.String("hook", hook.Name()),
            zap.Int("priority", hook.Priority()))

        result, err := llmHook.PreLLMCall(ctx, modifiedCtx)
        if err != nil {
            e.logger.Error("PreLLMCall failed",
                zap.String("hook", hook.Name()),
                zap.Error(err))

            ctx.AddError(err, hook.Name(), "pre_llm_call")

            if e.errorStrategy == ErrorStrategyStopOnFirst {
                return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
            }
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
func (e *Executor) ExecutePostLLMCall(ctx *HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
    hooks := e.getEnabledHooks(domain.HookTypeLLM)
    hooks = e.sortByPriority(hooks)

    var filterFunc func(domain.Hook) bool = func(h domain.Hook) bool {
        _, ok := h.(domain.LLMHook)
        return ok
    }

    modifiedResp := resp
    for _, hook := range hooks {
        if !filterFunc(hook) {
            continue
        }

        llmHook := hook.(domain.LLMHook)
        e.logger.Debug("executing PostLLMCall",
            zap.String("hook", hook.Name()),
            zap.Int("priority", hook.Priority()))

        result, err := llmHook.PostLLMCall(ctx, callCtx, modifiedResp)
        if err != nil {
            e.logger.Error("PostLLMCall failed",
                zap.String("hook", hook.Name()),
                zap.Error(err))

            ctx.AddError(err, hook.Name(), "post_llm_call")

            if e.errorStrategy == ErrorStrategyStopOnFirst {
                return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
            }
            continue
        }

        if result != nil {
            modifiedResp = result
        }
        ctx.AddHook(hook.Name())
    }

    return modifiedResp, nil
}

// ExecutePreToolCall 执行 PreToolCall 钩子
func (e *Executor) ExecutePreToolCall(ctx *HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
    hooks := e.getEnabledHooks(domain.HookTypeTool)
    hooks = e.sortByPriority(hooks)

    var filterFunc func(domain.Hook) bool = func(h domain.Hook) bool {
        _, ok := h.(domain.ToolHook)
        return ok
    }

    modifiedCtx := callCtx
    for _, hook := range hooks {
        if !filterFunc(hook) {
            continue
        }

        toolHook := hook.(domain.ToolHook)
        e.logger.Debug("executing PreToolCall",
            zap.String("hook", hook.Name()),
            zap.String("tool", callCtx.ToolName))

        result, err := toolHook.PreToolCall(ctx, modifiedCtx)
        if err != nil {
            e.logger.Error("PreToolCall failed",
                zap.String("hook", hook.Name()),
                zap.Error(err))

            ctx.AddError(err, hook.Name(), "pre_tool_call")

            if e.errorStrategy == ErrorStrategyStopOnFirst {
                return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
            }
            continue
        }

        if result != nil {
            modifiedCtx = result
        }
        ctx.AddHook(hook.Name())
    }

    return modifiedCtx, nil
}

// ExecutePostToolCall 执行 PostToolCall 钩子
func (e *Executor) ExecutePostToolCall(ctx *HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
    hooks := e.getEnabledHooks(domain.HookTypeTool)
    hooks = e.sortByPriority(hooks)

    var filterFunc func(domain.Hook) bool = func(h domain.Hook) bool {
        _, ok := h.(domain.ToolHook)
        return ok
    }

    modifiedResult := result
    for _, hook := range hooks {
        if !filterFunc(hook) {
            continue
        }

        toolHook := hook.(domain.ToolHook)
        e.logger.Debug("executing PostToolCall",
            zap.String("hook", hook.Name()),
            zap.String("tool", callCtx.ToolName))

        res, err := toolHook.PostToolCall(ctx, callCtx, modifiedResult)
        if err != nil {
            e.logger.Error("PostToolCall failed",
                zap.String("hook", hook.Name()),
                zap.Error(err))

            ctx.AddError(err, hook.Name(), "post_tool_call")

            if e.errorStrategy == ErrorStrategyStopOnFirst {
                return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
            }
            continue
        }

        if res != nil {
            modifiedResult = res
        }
        ctx.AddHook(hook.Name())
    }

    return modifiedResult, nil
}

func (e *Executor) getEnabledHooks(hookType domain.HookType) []domain.Hook {
    hooks := e.registry.ListByType(hookType)
    var enabled []domain.Hook
    for _, hook := range hooks {
        if hook.Enabled() {
            enabled = append(enabled, hook)
        }
    }
    return enabled
}

func (e *Executor) sortByPriority(hooks []domain.Hook) []domain.Hook {
    sorted := make([]domain.Hook, len(hooks))
    copy(sorted, hooks)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].Priority() < sorted[j].Priority()
    })
    return sorted
}
```

### 2.5 Manager (infrastructure/hook/manager.go)

```go
package hook

import (
    "context"
    "sync"

    "github.com/weibh/taskmanager/domain"
    "go.uber.org/zap"
)

// ManagerConfig Manager 配置
type ManagerConfig struct {
    ErrorStrategy ErrorStrategy
    EnableLogging bool
}

// Manager Hook 管理器
type Manager struct {
    mu       sync.RWMutex
    registry Registry
    executor *Executor
    logger   *zap.Logger
    config   *ManagerConfig
}

func NewManager(logger *zap.Logger, config *ManagerConfig) *Manager {
    if config == nil {
        config = &ManagerConfig{
            ErrorStrategy: ErrorStrategyContinue,
            EnableLogging: true,
        }
    }

    registry := NewRegistry()
    executor := NewExecutor(registry, logger)
    executor.SetErrorStrategy(config.ErrorStrategy)

    return &Manager{
        registry: registry,
        executor: executor,
        logger:   logger,
        config:   config,
    }
}

// Register 注册 Hook
func (m *Manager) Register(h domain.Hook) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.registry.Register(h)
}

// Unregister 注销 Hook
func (m *Manager) Unregister(name string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.registry.Unregister(name)
}

// List 列出所有 Hook
func (m *Manager) List() []domain.Hook {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.registry.List()
}

// PreLLMCall 执行 PreLLMCall 钩子
func (m *Manager) PreLLMCall(ctx context.Context, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
    hookCtx := NewHookContext(ctx)
    return m.executor.ExecutePreLLMCall(hookCtx, callCtx)
}

// PostLLMCall 执行 PostLLMCall 钩子
func (m *Manager) PostLLMCall(ctx context.Context, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
    hookCtx := NewHookContext(ctx)
    return m.executor.ExecutePostLLMCall(hookCtx, callCtx, resp)
}

// PreToolCall 执行 PreToolCall 钩子
func (m *Manager) PreToolCall(ctx context.Context, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
    hookCtx := NewHookContext(ctx)
    return m.executor.ExecutePreToolCall(hookCtx, callCtx)
}

// PostToolCall 执行 PostToolCall 钩子
func (m *Manager) PostToolCall(ctx context.Context, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
    hookCtx := NewHookContext(ctx)
    return m.executor.ExecutePostToolCall(hookCtx, callCtx, result)
}
```

### 2.6 内置 Hook 实现

#### LoggingHook (infrastructure/hook/hooks/logging.go)

```go
package hooks

import (
    "github.com/weibh/taskmanager/domain"
    "github.com/weibh/taskmanager/infrastructure/hook"
    "go.uber.org/zap"
)

// LoggingHook 记录所有 Hook 调用日志
type LoggingHook struct {
    *domain.BaseHook
    logger *zap.Logger
}

func NewLoggingHook(logger *zap.Logger) *LoggingHook {
    return &LoggingHook{
        BaseHook: &domain.BaseHook{
            name:     "logging",
            priority: 100,
            enabled:  true,
            hookType: domain.HookTypeLifecycle,
        },
        logger: logger,
    }
}

func (h *LoggingHook) OnInitialize(ctx *hook.HookContext, config interface{}) error {
    h.logger.Info("Hook system initializing")
    return nil
}

func (h *LoggingHook) OnShutdown(ctx *hook.HookContext) error {
    h.logger.Info("Hook system shutting down")
    return nil
}

func (h *LoggingHook) PreLLMCall(ctx *hook.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
    h.logger.Debug("PreLLMCall",
        zap.String("model", callCtx.Model),
        zap.Int("prompt_len", len(callCtx.Prompt)))
    return callCtx, nil
}

func (h *LoggingHook) PostLLMCall(ctx *hook.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
    h.logger.Debug("PostLLMCall",
        zap.String("model", resp.Model),
        zap.Int("content_len", len(resp.Content)),
        zap.Int("total_tokens", resp.Usage.TotalTokens))
    return resp, nil
}
```

#### MetricsHook (infrastructure/hook/hooks/metrics.go)

```go
package hooks

import (
    "sync"
    "time"

    "github.com/weibh/taskmanager/domain"
    "github.com/weibh/taskmanager/infrastructure/hook"
    "go.uber.org/zap"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
    mu           sync.RWMutex
    counters     map[string]int64
    gauges       map[string]interface{}
    histograms   map[string][]time.Duration
}

func NewMetricsCollector() *MetricsCollector {
    return &MetricsCollector{
        counters:   make(map[string]int64),
        gauges:     make(map[string]interface{}),
        histograms: make(map[string][]time.Duration),
    }
}

func (m *MetricsCollector) Increment(name string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.counters[name]++
}

func (m *MetricsCollector) Set(name string, val interface{}) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.gauges[name] = val
}

func (m *MetricsCollector) Record(name string, duration time.Duration) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.histograms[name] = append(m.histograms[name], duration)
}

// MetricsHook 收集指标
type MetricsHook struct {
    *domain.BaseHook
    metrics    *MetricsCollector
    startTimes map[string]time.Time
    mu         sync.Mutex
    logger     *zap.Logger
}

func NewMetricsHook(logger *zap.Logger) *MetricsHook {
    return &MetricsHook{
        BaseHook: &domain.BaseHook{
            name:     "metrics",
            priority: 50,
            enabled:  true,
            hookType: domain.HookTypeLLM,
        },
        metrics:    NewMetricsCollector(),
        startTimes: make(map[string]time.Time),
        logger:     logger,
    }
}

func (h *MetricsHook) PreLLMCall(ctx *hook.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
    h.mu.Lock()
    h.startTimes["llm"] = time.Now()
    h.mu.Unlock()

    h.metrics.Increment("llm_call_total")
    h.metrics.Set("llm_model", callCtx.Model)
    return callCtx, nil
}

func (h *MetricsHook) PostLLMCall(ctx *hook.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
    h.mu.Lock()
    if start, ok := h.startTimes["llm"]; ok {
        h.metrics.Record("llm_call_duration", time.Since(start))
        delete(h.startTimes, "llm")
    }
    h.mu.Unlock()

    h.metrics.Set("llm_prompt_tokens", resp.Usage.PromptTokens)
    h.metrics.Set("llm_completion_tokens", resp.Usage.CompletionTokens)
    h.metrics.Set("llm_total_tokens", resp.Usage.TotalTokens)

    h.logger.Info("LLM call completed",
        zap.Int("prompt_tokens", resp.Usage.PromptTokens),
        zap.Int("completion_tokens", resp.Usage.CompletionTokens),
        zap.Int("total_tokens", resp.Usage.TotalTokens))

    return resp, nil
}
```

#### RateLimitHook (infrastructure/hook/hooks/rate_limit.go)

```go
package hooks

import (
    "errors"
    "time"

    "github.com/weibh/taskmanager/domain"
    "github.com/weibh/taskmanager/infrastructure/hook"
    "go.uber.org/zap"
    "golang.org/x/time/rate"
)

var ErrRateLimited = errors.New("rate limit exceeded")

// RateLimitHook 限流
type RateLimitHook struct {
    *domain.BaseHook
    limiter *rate.Limiter
    burst   int
    logger  *zap.Logger
}

func NewRateLimitHook(limit rate.Limit, burst int, logger *zap.Logger) *RateLimitHook {
    return &RateLimitHook{
        BaseHook: &domain.BaseHook{
            name:     "rate_limit",
            priority: 5, // 最先执行
            enabled:  true,
            hookType: domain.HookTypeLLM,
        },
        limiter: rate.NewLimiter(limit, burst),
        burst:   burst,
        logger:  logger,
    }
}

func (h *RateLimitHook) PreLLMCall(ctx *hook.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
    if !h.limiter.Allow() {
        h.logger.Warn("rate limit exceeded, request blocked")
        return nil, ErrRateLimited
    }
    return callCtx, nil
}

// Wait implements backpressure
func (h *RateLimitHook) Wait(ctx *hook.HookContext) error {
    return h.limiter.Wait(ctx)
}
```

## 3. LLM 集成

### 3.1 Provider 集成 (infrastructure/llm/provider.go)

```go
package llm

import (
    "context"

    "github.com/weibh/taskmanager/domain"
    hook "github.com/weibh/taskmanager/infrastructure/hook"
)

// HookManagerInterface Hook 管理器接口
type HookManagerInterface interface {
    PreLLMCall(ctx context.Context, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error)
    PostLLMCall(ctx context.Context, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error)
}

// LLMProvider LLM provider 接口
type LLMProvider interface {
    Generate(ctx context.Context, prompt string) (string, error)
    GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error)
    Name() string
    SetHookManager(manager HookManagerInterface)
}

// hookableProvider 包装器，为普通 Provider 添加 Hook 支持
type hookableProvider struct {
    wrapped   LLMProvider
    hookMgr  HookManagerInterface
}

func (p *hookableProvider) SetHookManager(manager HookManagerInterface) {
    p.hookMgr = manager
}

func (p *hookableProvider) Generate(ctx context.Context, prompt string) (string, error) {
    // 1. Pre Hook
    if p.hookMgr != nil {
        callCtx := &domain.LLMCallContext{
            Prompt: prompt,
            Model:  "default",
        }
        modifiedCtx, err := p.hookMgr.PreLLMCall(ctx, callCtx)
        if err != nil {
            return "", err
        }
        prompt = modifiedCtx.Prompt
    }

    // 2. Actual LLM Call
    response, err := p.wrapped.Generate(ctx, prompt)
    if err != nil {
        return "", err
    }

    // 3. Post Hook
    if p.hookMgr != nil {
        resp := &domain.LLMResponse{
            Content: response,
        }
        modifiedResp, err := p.hookMgr.PostLLMCall(ctx, &domain.LLMCallContext{Prompt: prompt}, resp)
        if err != nil {
            return "", err
        }
        response = modifiedResp.Content
    }

    return response, nil
}

func (p *hookableProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error) {
    return p.wrapped.GenerateSubTasks(ctx, taskName, taskDesc, depth, maxDepth)
}

func (p *hookableProvider) Name() string {
    return p.wrapped.Name()
}
```

## 4. 配置加载

```go
package hook

import (
    "os"

    "gopkg.in/yaml.v3"
)

// Config Hook 系统配置
type Config struct {
    Enabled       bool             `yaml:"enabled"`
    ErrorStrategy string           `yaml:"error_strategy"`
    Hooks         []HookConfig     `yaml:"hooks"`
}

type HookConfig struct {
    Name     string                 `yaml:"name"`
    Enabled  bool                  `yaml:"enabled"`
    Priority int                    `yaml:"priority"`
    Config   map[string]interface{} `yaml:"config"`
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

## 5. 使用示例

```go
func Example() {
    // 1. 创建 Logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // 2. 创建 Manager
    manager := hook.NewManager(logger, nil)

    // 3. 注册内置 Hooks
    manager.Register(hooks.NewLoggingHook(logger))
    manager.Register(hooks.NewMetricsHook(logger))
    manager.Register(hooks.NewRateLimitHook(rate.Limit(10), 20, logger))

    // 4. 创建 LLM Provider 并设置 Hook Manager
    config := DefaultConfig()
    provider, _ := NewLLMProvider(config)

    if hp, ok := provider.(*hookableProvider); ok {
        hp.SetHookManager(manager)
    }

    // 5. 使用 Provider（会自动触发 Hooks）
    response, err := provider.Generate(ctx, "Hello, world!")
}
```
