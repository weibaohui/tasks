# Hook 系统测试文档

## 1. 测试策略

### 1.1 测试金字塔

```
         ┌───────────┐
         │   E2E     │  ← 端到端测试，验证完整流程
         │   Tests   │
         ├───────────┤
         │ Integration│  ← 集成测试，验证 Hook 链执行
         │   Tests    │
         ├───────────┤
         │   Unit     │  ← 单元测试，验证单个 Hook
         │   Tests    │
         └───────────┘
```

### 1.2 测试覆盖目标

| 类别 | 覆盖率目标 |
|------|------------|
| Registry | 100% |
| Executor | 100% |
| Manager | 100% |
| Hook Context | 100% |
| 内置 Hooks | 90% |
| 整体 | 95% |

## 2. 单元测试

### 2.1 Registry 测试 (registry_test.go)

```go
package hook

import (
    "errors"
    "testing"

    "github.com/weibh/taskmanager/domain"
    "go.uber.org/zap"
)

// mockHook 测试用 Mock Hook
type mockHook struct {
    name     string
    priority int
    enabled  bool
    hookType domain.HookType
}

func (m *mockHook) Name() string              { return m.name }
func (m *mockHook) Priority() int             { return m.priority }
func (m *mockHook) Enabled() bool             { return m.enabled }
func (m *mockHook) SetEnabled(b bool)         { m.enabled = b }
func (m *mockHook) HookType() domain.HookType { return m.hookType }

func TestRegistry_Register(t *testing.T) {
    registry := NewRegistry()
    logger, _ := zap.NewDevelopment()

    // Test: 注册成功
    hook := &mockHook{name: "test-hook", hookType: domain.HookTypeLLM}
    err := registry.Register(hook)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }

    // Test: 注册 nil hook 应返回错误
    err = registry.Register(nil)
    if err == nil {
        t.Fatal("expected error for nil hook")
    }

    // Test: 重复注册应返回错误
    err = registry.Register(hook)
    if err == nil {
        t.Fatal("expected error for duplicate registration")
    }

    // Test: 获取已注册的 hook
    retrieved := registry.Get("test-hook")
    if retrieved == nil {
        t.Fatal("expected to retrieve hook")
    }
    if retrieved.Name() != "test-hook" {
        t.Fatalf("expected name 'test-hook', got '%s'", retrieved.Name())
    }

    // Test: 获取不存在的 hook 应返回 nil
    notFound := registry.Get("non-existent")
    if notFound != nil {
        t.Fatal("expected nil for non-existent hook")
    }
}

func TestRegistry_Unregister(t *testing.T) {
    registry := NewRegistry()

    // 注册 hook
    hook := &mockHook{name: "test-hook", hookType: domain.HookTypeLLM}
    registry.Register(hook)

    // Test: 注销成功
    err := registry.Unregister("test-hook")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }

    // Test: 注销不存在的 hook 应返回错误
    err = registry.Unregister("non-existent")
    if err == nil {
        t.Fatal("expected error for unregister non-existent")
    }
}

func TestRegistry_List(t *testing.T) {
    registry := NewRegistry()

    // Test: 空注册表
    hooks := registry.List()
    if len(hooks) != 0 {
        t.Fatalf("expected 0 hooks, got %d", len(hooks))
    }

    // 注册多个 hooks
    hook1 := &mockHook{name: "hook-1", hookType: domain.HookTypeLLM}
    hook2 := &mockHook{name: "hook-2", hookType: domain.HookTypeTool}
    hook3 := &mockHook{name: "hook-3", hookType: domain.HookTypeLLM}

    registry.Register(hook1)
    registry.Register(hook2)
    registry.Register(hook3)

    // Test: List 返回所有 hooks
    hooks = registry.List()
    if len(hooks) != 3 {
        t.Fatalf("expected 3 hooks, got %d", len(hooks))
    }

    // Test: ListByType 过滤
    llmHooks := registry.ListByType(domain.HookTypeLLM)
    if len(llmHooks) != 2 {
        t.Fatalf("expected 2 LLM hooks, got %d", len(llmHooks))
    }

    toolHooks := registry.ListByType(domain.HookTypeTool)
    if len(toolHooks) != 1 {
        t.Fatalf("expected 1 Tool hook, got %d", len(toolHooks))
    }
}

func TestRegistry_EnableDisable(t *testing.T) {
    registry := NewRegistry()

    hook := &mockHook{name: "test-hook", enabled: true, hookType: domain.HookTypeLLM}
    registry.Register(hook)

    // Test: 禁用 hook
    err := registry.Disable("test-hook")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if registry.Get("test-hook").Enabled() {
        t.Fatal("expected hook to be disabled")
    }

    // Test: 启用 hook
    err = registry.Enable("test-hook")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if !registry.Get("test-hook").Enabled() {
        t.Fatal("expected hook to be enabled")
    }

    // Test: 启用/禁用不存在的 hook 应返回错误
    err = registry.Enable("non-existent")
    if err == nil {
        t.Fatal("expected error for enable non-existent")
    }
}

func TestRegistry_Clear(t *testing.T) {
    registry := NewRegistry()

    registry.Register(&mockHook{name: "hook-1", hookType: domain.HookTypeLLM})
    registry.Register(&mockHook{name: "hook-2", hookType: domain.HookTypeLLM})

    // Test: Clear
    registry.Clear()
    if len(registry.List()) != 0 {
        t.Fatal("expected 0 hooks after clear")
    }
}
```

### 2.2 Executor 测试 (executor_test.go)

```go
package hook

import (
    "context"
    "errors"
    "sync"
    "testing"
    "time"

    "github.com/weibh/taskmanager/domain"
    "go.uber.org/zap"
)

// mockLLMHook 测试用 LLM Hook
type mockLLMHook struct {
    name            string
    priority        int
    enabled         bool
    preCallFn       func(ctx *HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error)
    postCallFn      func(ctx *HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error)
    callOrder       *[]string
    callOrderMu     *sync.Mutex
}

func (m *mockLLMHook) Name() string                              { return m.name }
func (m *mockLLMHook) Priority() int                             { return m.priority }
func (m *mockLLMHook) Enabled() bool                             { return m.enabled }
func (m *mockLLMHook) SetEnabled(b bool)                          { m.enabled = b }
func (m *mockLLMHook) HookType() domain.HookType                  { return domain.HookTypeLLM }
func (m *mockLLMHook) PreLLMCall(ctx *HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
    if m.callOrder != nil {
        m.callOrderMu.Lock()
        *m.callOrder = append(*m.callOrder, m.name+"-pre")
        m.callOrderMu.Unlock()
    }
    if m.preCallFn != nil {
        return m.preCallFn(ctx, callCtx)
    }
    return callCtx, nil
}
func (m *mockLLMHook) PostLLMCall(ctx *HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
    if m.callOrder != nil {
        m.callOrderMu.Lock()
        *m.callOrder = append(*m.callOrder, m.name+"-post")
        m.callOrderMu.Unlock()
    }
    if m.postCallFn != nil {
        return m.postCallFn(ctx, callCtx, resp)
    }
    return resp, nil
}

func TestExecutor_ExecutePreLLMCall(t *testing.T) {
    registry := NewRegistry()
    logger, _ := zap.NewDevelopment()
    executor := NewExecutor(registry, logger)

    // 注册一个修改 prompt 的 Hook
    registry.Register(&mockLLMHook{
        name:     "modifier",
        priority: 10,
        enabled:  true,
        preCallFn: func(ctx *HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
            callCtx.Prompt = callCtx.Prompt + " [modified]"
            return callCtx, nil
        },
    })

    // Test: 执行 PreLLMCall
    ctx := NewHookContext(context.Background())
    callCtx := &domain.LLMCallContext{Prompt: "original"}
    result, err := executor.ExecutePreLLMCall(ctx, callCtx)

    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if result.Prompt != "original [modified]" {
        t.Fatalf("expected prompt 'original [modified]', got '%s'", result.Prompt)
    }
}

func TestExecutor_PriorityOrdering(t *testing.T) {
    registry := NewRegistry()
    logger, _ := zap.NewDevelopment()
    executor := NewExecutor(registry, logger)

    var callOrder []string
    var mu sync.Mutex

    // 注册多个 hooks，优先级不同
    registry.Register(&mockLLMHook{
        name:      "hook-priority-20",
        priority:  20,
        enabled:   true,
        callOrder: &callOrder,
        callOrderMu: &mu,
    })
    registry.Register(&mockLLMHook{
        name:      "hook-priority-10",
        priority:  10,
        enabled:   true,
        callOrder: &callOrder,
        callOrderMu: &mu,
    })
    registry.Register(&mockLLMHook{
        name:      "hook-priority-30",
        priority:  30,
        enabled:   true,
        callOrder: &callOrder,
        callOrderMu: &mu,
    })

    ctx := NewHookContext(context.Background())
    executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

    // Test: 按优先级顺序执行
    expected := []string{"hook-priority-10-pre", "hook-priority-20-pre", "hook-priority-30-pre"}
    if len(callOrder) != len(expected) {
        t.Fatalf("expected %d calls, got %d", len(expected), len(callOrder))
    }
    for i, e := range expected {
        if callOrder[i] != e {
            t.Fatalf("at index %d: expected '%s', got '%s'", i, e, callOrder[i])
        }
    }
}

func TestExecutor_DisabledHookSkipped(t *testing.T) {
    registry := NewRegistry()
    logger, _ := zap.NewDevelopment()
    executor := NewExecutor(registry, logger)

    var callOrder []string
    var mu sync.Mutex

    registry.Register(&mockLLMHook{
        name:        "enabled-hook",
        priority:    10,
        enabled:     true,
        callOrder:   &callOrder,
        callOrderMu: &mu,
    })
    registry.Register(&mockLLMHook{
        name:        "disabled-hook",
        priority:    5, // 更高优先级但被禁用
        enabled:     false,
        callOrder:   &callOrder,
        callOrderMu: &mu,
    })

    ctx := NewHookContext(context.Background())
    executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

    // Test: 禁用的 hook 应被跳过
    if len(callOrder) != 1 {
        t.Fatalf("expected 1 call, got %d", len(callOrder))
    }
    if callOrder[0] != "enabled-hook-pre" {
        t.Fatalf("expected 'enabled-hook-pre', got '%s'", callOrder[0])
    }
}

func TestExecutor_ErrorHandling_Continue(t *testing.T) {
    registry := NewRegistry()
    logger, _ := zap.NewDevelopment()
    executor := NewExecutor(registry, logger)
    executor.SetErrorStrategy(ErrorStrategyContinue)

    var callOrder []string
    var mu sync.Mutex

    registry.Register(&mockLLMHook{
        name:        "error-hook",
        priority:    10,
        enabled:     true,
        callOrder:   &callOrder,
        callOrderMu: &mu,
        preCallFn: func(ctx *HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
            return nil, errors.New("hook error")
        },
    })
    registry.Register(&mockLLMHook{
        name:        "after-error-hook",
        priority:    20,
        enabled:     true,
        callOrder:   &callOrder,
        callOrderMu: &mu,
    })

    ctx := NewHookContext(context.Background())
    result, err := executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

    // Test: Continue 模式下，错误后继续执行
    if err != nil {
        t.Fatalf("expected no returned error in continue mode, got %v", err)
    }
    if len(callOrder) != 1 {
        t.Fatalf("expected 1 call (error hook), got %d", len(callOrder))
    }
    if ctx.HasErrors() {
        t.Fatal("expected errors to be recorded in context")
    }
}

func TestExecutor_ErrorHandling_StopOnFirst(t *testing.T) {
    registry := NewRegistry()
    logger, _ := zap.NewDevelopment()
    executor := NewExecutor(registry, logger)
    executor.SetErrorStrategy(ErrorStrategyStopOnFirst)

    registry.Register(&mockLLMHook{
        name:     "error-hook",
        priority: 10,
        enabled:  true,
        preCallFn: func(ctx *HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
            return nil, errors.New("hook error")
        },
    })
    registry.Register(&mockLLMHook{
        name:      "after-error-hook",
        priority:  20,
        enabled:   true,
        callOrder: &[]string{}, // dummy
    })

    ctx := NewHookContext(context.Background())
    _, err := executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

    // Test: StopOnFirst 模式下，遇到错误立即返回
    if err == nil {
        t.Fatal("expected error in stop on first mode")
    }
}

func TestExecutor_HookContextPropagation(t *testing.T) {
    registry := NewRegistry()
    logger, _ := zap.NewDevelopment()
    executor := NewExecutor(registry, logger)

    registry.Register(&mockLLMHook{
        name:     "hook-1",
        priority: 10,
        enabled:  true,
        preCallFn: func(ctx *HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
            ctx.SetMetadata("key1", "value1")
            return callCtx, nil
        },
    })
    registry.Register(&mockLLMHook{
        name:     "hook-2",
        priority: 20,
        enabled:  true,
        preCallFn: func(ctx *HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
            // 应该能访问 hook-1 设置的值
            val := ctx.GetMetadata("key1")
            if val != "value1" {
                t.Errorf("expected 'value1', got '%v'", val)
            }
            return callCtx, nil
        },
    })

    ctx := NewHookContext(context.Background())
    executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

    // Test: Hook 列表应该包含所有执行的 hook
    hooks := ctx.GetHooks()
    if len(hooks) != 2 {
        t.Fatalf("expected 2 hooks, got %d", len(hooks))
    }
}
```

### 2.3 HookContext 测试 (context_test.go)

```go
package hook

import (
    "context"
    "errors"
    "testing"
    "time"
)

func TestHookContext_WithValue(t *testing.T) {
    ctx := NewHookContext(context.Background())

    ctx.WithValue("key1", "value1")
    ctx.WithValue("key2", 123)

    if val := ctx.Get("key1"); val != "value1" {
        t.Errorf("expected 'value1', got '%v'", val)
    }
    if val := ctx.Get("key2"); val != 123 {
        t.Errorf("expected 123, got '%v'", val)
    }
}

func TestHookContext_AddHook(t *testing.T) {
    ctx := NewHookContext(context.Background())

    ctx.AddHook("hook-1")
    ctx.AddHook("hook-2")

    hooks := ctx.GetHooks()
    if len(hooks) != 2 {
        t.Fatalf("expected 2 hooks, got %d", len(hooks))
    }
    if hooks[0] != "hook-1" || hooks[1] != "hook-2" {
        t.Errorf("unexpected hook order: %v", hooks)
    }
}

func TestHookContext_AddError(t *testing.T) {
    ctx := NewHookContext(context.Background())

    err1 := errors.New("error 1")
    err2 := errors.New("error 2")

    ctx.AddError(err1, "hook-1", "pre")
    ctx.AddError(err2, "hook-2", "post")

    if !ctx.HasErrors() {
        t.Fatal("expected HasErrors to return true")
    }

    errs := ctx.GetErrors()
    if len(errs) != 2 {
        t.Fatalf("expected 2 errors, got %d", len(errs))
    }
    if errs[0].HookName != "hook-1" || errs[0].Phase != "pre" {
        t.Errorf("unexpected error: %+v", errs[0])
    }
}

func TestHookContext_Duration(t *testing.T) {
    ctx := NewHookContext(context.Background())

    time.Sleep(10 * time.Millisecond)

    duration := ctx.Duration()
    if duration < 10*time.Millisecond {
        t.Errorf("expected duration >= 10ms, got %v", duration)
    }
}

func TestHookContext_Metadata(t *testing.T) {
    ctx := NewHookContext(context.Background())

    ctx.SetMetadata("session_id", "sess-123")
    ctx.SetMetadata("trace_id", "trace-456")

    if val := ctx.GetMetadata("session_id"); val != "sess-123" {
        t.Errorf("expected 'sess-123', got '%s'", val)
    }
    if val := ctx.GetMetadata("trace_id"); val != "trace-456" {
        t.Errorf("expected 'trace-456', got '%s'", val)
    }
    if val := ctx.GetMetadata("non-existent"); val != "" {
        t.Errorf("expected '', got '%s'", val)
    }
}
```

### 2.4 内置 Hook 测试 (hooks_test.go)

```go
package hooks

import (
    "context"
    "testing"
    "time"

    "github.com/weibh/taskmanager/domain"
    "github.com/weibh/taskmanager/infrastructure/hook"
    "go.uber.org/zap"
)

func TestLoggingHook(t *testing.T) {
    logger, _ := zap.NewDevelopment()
    h := NewLoggingHook(logger)

    if h.Name() != "logging" {
        t.Errorf("expected name 'logging', got '%s'", h.Name())
    }
    if h.Priority() != 100 {
        t.Errorf("expected priority 100, got %d", h.Priority())
    }
    if !h.Enabled() {
        t.Error("expected enabled")
    }
}

func TestMetricsHook(t *testing.T) {
    logger, _ := zap.NewDevelopment()
    h := NewMetricsHook(logger)

    // Test: PreLLMCall 记录开始时间
    ctx := hook.NewHookContext(context.Background())
    callCtx := &domain.LLMCallContext{
        Prompt: "test prompt",
        Model:  "gpt-4",
    }

    result, err := h.PreLLMCall(ctx, callCtx)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if result != callCtx {
        t.Error("expected same context returned")
    }

    // Test: PostLLMCall 记录耗时
    resp := &domain.LLMResponse{
        Content: "test response",
        Usage: domain.Usage{
            PromptTokens:     10,
            CompletionTokens: 20,
            TotalTokens:      30,
        },
    }

    time.Sleep(10 * time.Millisecond)

    resultResp, err := h.PostLLMCall(ctx, callCtx, resp)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if resultResp != resp {
        t.Error("expected same response returned")
    }
}

func TestRateLimitHook(t *testing.T) {
    logger, _ := zap.NewDevelopment()
    h := NewRateLimitHook(10, 20, logger) // 10 req/s, burst 20

    if !h.Enabled() {
        t.Error("expected enabled")
    }

    // Test: 前 20 个请求应该通过
    ctx := hook.NewHookContext(context.Background())
    callCtx := &domain.LLMCallContext{Prompt: "test"}

    for i := 0; i < 20; i++ {
        _, err := h.PreLLMCall(ctx, callCtx)
        if err != nil {
            t.Fatalf("request %d: expected no error, got %v", i, err)
        }
    }

    // Test: 第 21 个请求应该被限流
    _, err := h.PreLLMCall(ctx, callCtx)
    if err == nil {
        t.Fatal("expected rate limit error")
    }
    if err != hook.ErrRateLimited {
        t.Errorf("expected ErrRateLimited, got %v", err)
    }
}
```

## 3. 集成测试

### 3.1 Hook 链集成测试

```go
package hook_test

import (
    "context"
    "fmt"
    "sync"
    "testing"

    "github.com/weibh/taskmanager/domain"
    hook "github.com/weibh/taskmanager/infrastructure/hook"
    "github.com/weibh/taskmanager/infrastructure/hook/hooks"
    "go.uber.org/zap"
)

func TestHookManager_FullChain(t *testing.T) {
    logger, _ := zap.NewDevelopment()
    manager := hook.NewManager(logger, nil)

    // 注册 Hooks
    manager.Register(hooks.NewLoggingHook(logger))
    manager.Register(hooks.NewMetricsHook(logger))

    // 创建 mock LLM Provider
    mockProvider := &mockLLMProvider{
        generateFn: func(prompt string) (string, error) {
            return fmt.Sprintf("response to: %s", prompt), nil
        },
    }

    // 创建 hookable provider
    hookableProvider := &hookableProvider{
        wrapped:  mockProvider,
        hookMgr: manager,
    }

    // Test: 调用 LLM
    ctx := context.Background()
    resp, err := hookableProvider.Generate(ctx, "test prompt")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if resp != "response to: test prompt" {
        t.Errorf("unexpected response: %s", resp)
    }
}

// mockLLMProvider 测试用 Provider
type mockLLMProvider struct {
    generateFn func(prompt string) (string, error)
}

func (m *mockLLMProvider) Generate(ctx context.Context, prompt string) (string, error) {
    return m.generateFn(prompt)
}

func (m *mockLLMProvider) Name() string {
    return "mock"
}

// hookableProvider 测试用包装器
type hookableProvider struct {
    wrapped  *mockLLMProvider
    hookMgr  hook.HookManagerInterface
}

func (p *hookableProvider) Generate(ctx context.Context, prompt string) (string, error) {
    if p.hookMgr != nil {
        callCtx := &domain.LLMCallContext{Prompt: prompt, Model: "mock"}
        modifiedCtx, err := p.hookMgr.PreLLMCall(ctx, callCtx)
        if err != nil {
            return "", err
        }
        prompt = modifiedCtx.Prompt
    }

    response, err := p.wrapped.Generate(ctx, prompt)
    if err != nil {
        return "", err
    }

    if p.hookMgr != nil {
        resp := &domain.LLMResponse{Content: response}
        modifiedResp, err := p.hookMgr.PostLLMCall(ctx, &domain.LLMCallContext{Prompt: prompt}, resp)
        if err != nil {
            return "", err
        }
        response = modifiedResp.Content
    }

    return response, nil
}
```

### 3.2 性能测试

```go
package hook_test

import (
    "context"
    "testing"
    "time"

    "github.com/weibh/taskmanager/domain"
    hook "github.com/weibh/taskmanager/infrastructure/hook"
    "go.uber.org/zap"
)

func BenchmarkHookExecution(b *testing.B) {
    logger, _ := zap.NewDevelopment()
    manager := hook.NewManager(logger, nil)

    // 注册多个 hooks
    for i := 0; i < 10; i++ {
        manager.Register(&benchmarkHook{name: fmt.Sprintf("hook-%d", i), priority: i})
    }

    ctx := context.Background()
    callCtx := &domain.LLMCallContext{Prompt: "benchmark test"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        manager.PreLLMCall(ctx, callCtx)
    }
}

type benchmarkHook struct {
    name     string
    priority int
}

func (h *benchmarkHook) Name() string              { return h.name }
func (h *benchmarkHook) Priority() int             { return h.priority }
func (h *benchmarkHook) Enabled() bool             { return true }
func (h *benchmarkHook) SetEnabled(bool)            {}
func (h *benchmarkHook) HookType() domain.HookType { return domain.HookTypeLLM }
func (h *benchmarkHook) PreLLMCall(ctx *hook.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
    time.Sleep(time.Microsecond) // 模拟一些处理
    return callCtx, nil
}
func (h *benchmarkHook) PostLLMCall(ctx *hook.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
    return resp, nil
}
```

## 4. E2E 测试

### 4.1 完整流程 E2E 测试

```go
package e2e

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/weibh/taskmanager/domain"
    hook "github.com/weibh/taskmanager/infrastructure/hook"
    "github.com/weibh/taskmanager/infrastructure/hook/hooks"
    "github.com/weibh/taskmanager/infrastructure/llm"
    "go.uber.org/zap"
)

func TestHookSystem_E2E(t *testing.T) {
    if os.Getenv("E2E_TEST") != "1" {
        t.Skip("skipping e2e test")
    }

    logger, _ := zap.NewProduction()
    manager := hook.NewManager(logger, nil)

    // 注册内置 hooks
    manager.Register(hooks.NewLoggingHook(logger))
    manager.Register(hooks.NewMetricsHook(logger))

    // 创建真实的 LLM Provider
    config := &llm.Config{
        ProviderType: "openai",
        Model:        "gpt-4",
        APIKey:       os.Getenv("OPENAI_API_KEY"),
        BaseURL:      os.Getenv("OPENAI_BASE_URL"),
    }

    provider, err := llm.NewLLMProvider(config)
    if err != nil {
        t.Fatalf("failed to create provider: %v", err)
    }

    // 包装为 hookable
    hookableProvider := &hookableLLMProvider{
        wrapped:  provider,
        hookMgr: manager,
    }

    // Test: 执行 LLM 调用
    ctx := context.Background()
    start := time.Now()

    resp, err := hookableProvider.Generate(ctx, "What is 2+2?")
    if err != nil {
        t.Fatalf("LLM call failed: %v", err)
    }

    duration := time.Since(start)

    t.Logf("LLM response: %s", resp)
    t.Logf("Duration: %v", duration)

    // 验证响应不为空
    if len(resp) == 0 {
        t.Error("expected non-empty response")
    }
}

type hookableLLMProvider struct {
    wrapped  llm.LLMProvider
    hookMgr  hook.HookManagerInterface
}

func (p *hookableLLMProvider) Generate(ctx context.Context, prompt string) (string, error) {
    if p.hookMgr != nil {
        callCtx := &domain.LLMCallContext{Prompt: prompt}
        modifiedCtx, err := p.hookMgr.PreLLMCall(ctx, callCtx)
        if err != nil {
            return "", err
        }
        prompt = modifiedCtx.Prompt
    }

    resp, err := p.wrapped.Generate(ctx, prompt)
    if err != nil {
        return "", err
    }

    if p.hookMgr != nil {
        llmResp := &domain.LLMResponse{Content: resp}
        modifiedResp, err := p.hookMgr.PostLLMCall(ctx, &domain.LLMCallContext{Prompt: prompt}, llmResp)
        if err != nil {
            return "", err
        }
        resp = modifiedResp.Content
    }

    return resp, nil
}
```

## 5. 测试运行

```bash
# 运行所有测试
go test ./infrastructure/hook/... -v

# 运行单元测试
go test ./infrastructure/hook/... -v -short

# 运行集成测试
go test ./infrastructure/hook/... -v -run Integration

# 运行性能测试
go test ./infrastructure/hook/... -bench=. -benchmem

# 运行 E2E 测试
E2E_TEST=1 go test ./infrastructure/hook/... -v -run E2E

# 生成覆盖率报告
go test ./infrastructure/hook/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 6. Mock 数据

### 6.1 Mock LLM Provider

```go
// mockLLMProvider 用于测试的 Mock LLM Provider
type mockLLMProvider struct {
    GenerateFunc        func(ctx context.Context, prompt string) (string, error)
    GenerateSubTasksFunc func(ctx context.Context, taskName, taskDesc string, depth, maxDepth int) (*llm.SubTaskPlan, error)
}

func (m *mockLLMProvider) Generate(ctx context.Context, prompt string) (string, error) {
    if m.GenerateFunc != nil {
        return m.GenerateFunc(ctx, prompt)
    }
    return "mock response", nil
}

func (m *mockLLMProvider) GenerateSubTasks(ctx context.Context, taskName, taskDesc string, depth, maxDepth int) (*llm.SubTaskPlan, error) {
    if m.GenerateSubTasksFunc != nil {
        return m.GenerateSubTasksFunc(ctx, taskName, taskDesc, depth, maxDepth)
    }
    return &llm.SubTaskPlan{}, nil
}

func (m *mockLLMProvider) Name() string {
    return "mock"
}
```

### 6.2 Mock Tool Hook

```go
// mockToolHook 用于测试的 Mock Tool Hook
type mockToolHook struct {
    name            string
    priority        int
    enabled         bool
    preToolCallFn   func(ctx *hook.HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error)
    postToolCallFn  func(ctx *hook.HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error)
}

func (m *mockToolHook) Name() string                              { return m.name }
func (m *mockToolHook) Priority() int                             { return m.priority }
func (m *mockToolHook) Enabled() bool                             { return m.enabled }
func (m *mockToolHook) SetEnabled(b bool)                          { m.enabled = b }
func (m *mockToolHook) HookType() domain.HookType                  { return domain.HookTypeTool }
func (m *mockToolHook) PreToolCall(ctx *hook.HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
    if m.preToolCallFn != nil {
        return m.preToolCallFn(ctx, callCtx)
    }
    return callCtx, nil
}
func (m *mockToolHook) PostToolCall(ctx *hook.HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
    if m.postToolCallFn != nil {
        return m.postToolCallFn(ctx, callCtx, result)
    }
    return result, nil
}
func (m *mockToolHook) OnToolError(ctx *hook.HookContext, callCtx *domain.ToolCallContext, err error) (*domain.ToolExecutionResult, error) {
    return &domain.ToolExecutionResult{Success: false, Error: err}, nil
}
```
