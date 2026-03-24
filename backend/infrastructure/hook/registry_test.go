/**
 * Hook Registry 单元测试
 */
package hook

import (
	"testing"

	"github.com/weibh/taskmanager/domain"
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

// mockLLMHook 测试用 LLM Hook
type mockLLMHook struct {
	name       string
	priority   int
	enabled    bool
	preCallFn  func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error)
	postCallFn func(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error)
}

func (m *mockLLMHook) Name() string              { return m.name }
func (m *mockLLMHook) Priority() int             { return m.priority }
func (m *mockLLMHook) Enabled() bool             { return m.enabled }
func (m *mockLLMHook) SetEnabled(b bool)         { m.enabled = b }
func (m *mockLLMHook) HookType() domain.HookType { return domain.HookTypeLLM }
func (m *mockLLMHook) PreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
	if m.preCallFn != nil {
		return m.preCallFn(ctx, callCtx)
	}
	return callCtx, nil
}
func (m *mockLLMHook) PostLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
	if m.postCallFn != nil {
		return m.postCallFn(ctx, callCtx, resp)
	}
	return resp, nil
}
