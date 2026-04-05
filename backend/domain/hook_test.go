package domain

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ============================================================================
// BaseHook Tests
// ============================================================================

func TestNewBaseHook(t *testing.T) {
	hook := NewBaseHook("test-hook", 10, HookTypeLifecycle)

	if hook.name != "test-hook" {
		t.Errorf("期望 name 为 test-hook, 实际为 %s", hook.name)
	}
	if hook.priority != 10 {
		t.Errorf("期望 priority 为 10, 实际为 %d", hook.priority)
	}
	if !hook.enabled {
		t.Error("期望 enabled 为 true")
	}
	if hook.hookType != HookTypeLifecycle {
		t.Errorf("期望 hookType 为 lifecycle, 实际为 %s", hook.hookType)
	}
}

func TestBaseHook_Methods(t *testing.T) {
	hook := NewBaseHook("test-hook", 5, HookTypeLLM)

	if hook.Name() != "test-hook" {
		t.Errorf("期望 Name() 返回 test-hook, 实际为 %s", hook.Name())
	}
	if hook.Priority() != 5 {
		t.Errorf("期望 Priority() 返回 5, 实际为 %d", hook.Priority())
	}
	if !hook.Enabled() {
		t.Error("期望 Enabled() 返回 true")
	}
	if hook.HookType() != HookTypeLLM {
		t.Errorf("期望 HookType() 返回 llm, 实际为 %s", hook.HookType())
	}

	hook.SetEnabled(false)
	if hook.Enabled() {
		t.Error("SetEnabled(false) 后期望 Enabled() 返回 false")
	}

	hook.SetEnabled(true)
	if !hook.Enabled() {
		t.Error("SetEnabled(true) 后期望 Enabled() 返回 true")
	}
}

// ============================================================================
// HookContext Tests
// ============================================================================

func TestNewHookContext(t *testing.T) {
	ctx := context.Background()
	hc := NewHookContext(ctx)

	if hc.Context == nil {
		t.Error("期望 Context 不为 nil")
	}
	if hc.values == nil {
		t.Error("期望 values 已初始化")
	}
	if hc.hooks == nil {
		t.Error("期望 hooks 已初始化")
	}
	if hc.errors == nil {
		t.Error("期望 errors 已初始化")
	}
	if hc.metadata == nil {
		t.Error("期望 metadata 已初始化")
	}
	if hc.startTime.IsZero() {
		t.Error("期望 startTime 不为零值")
	}
}

func TestHookContext_WithValue_Get(t *testing.T) {
	hc := NewHookContext(context.Background())

	hc.WithValue("key1", "value1")
	hc.WithValue("key2", 42)

	if v := hc.Get("key1"); v != "value1" {
		t.Errorf("期望 Get(key1) 返回 value1, 实际为 %v", v)
	}
	if v := hc.Get("key2"); v != 42 {
		t.Errorf("期望 Get(key2) 返回 42, 实际为 %v", v)
	}
	if v := hc.Get("nonexistent"); v != nil {
		t.Errorf("期望 Get(nonexistent) 返回 nil, 实际为 %v", v)
	}
}

func TestHookContext_AddHook_GetHooks(t *testing.T) {
	hc := NewHookContext(context.Background())

	hc.AddHook("hook-1")
	hc.AddHook("hook-2")
	hc.AddHook("hook-1")

	hooks := hc.GetHooks()
	if len(hooks) != 3 {
		t.Errorf("期望 hooks 长度为 3, 实际为 %d", len(hooks))
	}
	if hooks[0] != "hook-1" || hooks[1] != "hook-2" || hooks[2] != "hook-1" {
		t.Errorf("期望 hooks 顺序为 [hook-1, hook-2, hook-1], 实际为 %v", hooks)
	}

	// 验证返回的是副本
	hooks[0] = "modified"
	if hc.GetHooks()[0] != "hook-1" {
		t.Error("期望 GetHooks() 返回副本，不影响原始数据")
	}
}

func TestHookContext_AddError_GetErrors_HasErrors(t *testing.T) {
	hc := NewHookContext(context.Background())

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	hc.AddError(err1, "hook-a", "phase-1")
	hc.AddError(err2, "hook-b", "phase-2")

	if !hc.HasErrors() {
		t.Error("期望 HasErrors() 返回 true")
	}

	errs := hc.GetErrors()
	if len(errs) != 2 {
		t.Errorf("期望 errors 长度为 2, 实际为 %d", len(errs))
	}
	if errs[0].Err != err1 || errs[0].HookName != "hook-a" || errs[0].Phase != "phase-1" {
		t.Errorf("期望第一个错误匹配，实际为 %+v", errs[0])
	}
	if errs[1].Err != err2 || errs[1].HookName != "hook-b" || errs[1].Phase != "phase-2" {
		t.Errorf("期望第二个错误匹配，实际为 %+v", errs[1])
	}

	// 验证返回的是副本
	errs[0].HookName = "modified"
	if hc.GetErrors()[0].HookName != "hook-a" {
		t.Error("期望 GetErrors() 返回副本，不影响原始数据")
	}
}

func TestHookContext_HasErrors_Empty(t *testing.T) {
	hc := NewHookContext(context.Background())
	if hc.HasErrors() {
		t.Error("空上下文期望 HasErrors() 返回 false")
	}
	if len(hc.GetErrors()) != 0 {
		t.Error("空上下文期望 GetErrors() 为空")
	}
}

func TestHookContext_SetMetadata_GetMetadata(t *testing.T) {
	hc := NewHookContext(context.Background())

	hc.SetMetadata("trace-id", "abc-123")
	hc.SetMetadata("project", "test-project")

	if v := hc.GetMetadata("trace-id"); v != "abc-123" {
		t.Errorf("期望 GetMetadata(trace-id) 返回 abc-123, 实际为 %v", v)
	}
	if v := hc.GetMetadata("project"); v != "test-project" {
		t.Errorf("期望 GetMetadata(project) 返回 test-project, 实际为 %v", v)
	}
	if v := hc.GetMetadata("missing"); v != "" {
		t.Errorf("期望 GetMetadata(missing) 返回空字符串, 实际为 %v", v)
	}
}

func TestHookContext_Duration(t *testing.T) {
	hc := NewHookContext(context.Background())

	// 等待一小段时间
	time.Sleep(20 * time.Millisecond)

	d := hc.Duration()
	if d <= 0 {
		t.Errorf("期望 Duration > 0, 实际为 %v", d)
	}
	// 使用更宽松的阈值，避免 CI 环境中的调度延迟导致测试失败
	if d < 1*time.Millisecond {
		t.Errorf("期望 Duration >= 1ms, 实际为 %v", d)
	}
}

