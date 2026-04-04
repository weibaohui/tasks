package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// Mock Implementations
// ============================================================================

type mockRequirementStateHook struct {
	BaseHook
	onChangeFunc func(ctx context.Context, req *Requirement, change *StateChange) error
	callCount    int
}

func newMockRequirementStateHook(name string, priority int, enabled bool) *mockRequirementStateHook {
	h := &mockRequirementStateHook{
		BaseHook: *NewBaseHook(name, priority, HookTypeRequirement),
	}
	h.SetEnabled(enabled)
	return h
}

func (m *mockRequirementStateHook) OnRequirementStateChanged(ctx context.Context, req *Requirement, change *StateChange) error {
	m.callCount++
	if m.onChangeFunc != nil {
		return m.onChangeFunc(ctx, req, change)
	}
	return nil
}

type mockRequirementStateHookLogger struct {
	debugLogs []string
	infoLogs  []string
	errorLogs []string
}

func (m *mockRequirementStateHookLogger) Debug(msg string, fields ...RequirementStateHookLogField) {
	m.debugLogs = append(m.debugLogs, msg)
}
func (m *mockRequirementStateHookLogger) Info(msg string, fields ...RequirementStateHookLogField) {
	m.infoLogs = append(m.infoLogs, msg)
}
func (m *mockRequirementStateHookLogger) Error(msg string, fields ...RequirementStateHookLogField) {
	m.errorLogs = append(m.errorLogs, msg)
}

type mockRequirementHookConfigRepository struct {
	configs map[string]*RequirementHookConfig
	findErr error
}

func newMockRequirementHookConfigRepository() *mockRequirementHookConfigRepository {
	return &mockRequirementHookConfigRepository{
		configs: make(map[string]*RequirementHookConfig),
	}
}

func (m *mockRequirementHookConfigRepository) Save(ctx context.Context, config *RequirementHookConfig) error {
	m.configs[config.ID] = config
	return nil
}
func (m *mockRequirementHookConfigRepository) FindByID(ctx context.Context, id string) (*RequirementHookConfig, error) {
	return m.configs[id], nil
}
func (m *mockRequirementHookConfigRepository) FindByTriggerPoint(ctx context.Context, triggerPoint string) ([]*RequirementHookConfig, error) {
	var result []*RequirementHookConfig
	for _, c := range m.configs {
		if c.TriggerPoint == triggerPoint {
			result = append(result, c)
		}
	}
	return result, nil
}
func (m *mockRequirementHookConfigRepository) FindByProjectID(ctx context.Context, projectID string) ([]*RequirementHookConfig, error) {
	var result []*RequirementHookConfig
	for _, c := range m.configs {
		if c.ProjectID == projectID {
			result = append(result, c)
		}
	}
	return result, nil
}
func (m *mockRequirementHookConfigRepository) FindEnabledByTriggerPoint(ctx context.Context, triggerPoint string) ([]*RequirementHookConfig, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*RequirementHookConfig
	for _, c := range m.configs {
		if c.TriggerPoint == triggerPoint && c.Enabled {
			result = append(result, c)
		}
	}
	return result, nil
}
func (m *mockRequirementHookConfigRepository) Delete(ctx context.Context, id string) error {
	delete(m.configs, id)
	return nil
}

type mockRequirementHookActionLogRepository struct {
	logs    map[string]*RequirementHookActionLog
	findErr error
}

func newMockRequirementHookActionLogRepository() *mockRequirementHookActionLogRepository {
	return &mockRequirementHookActionLogRepository{
		logs: make(map[string]*RequirementHookActionLog),
	}
}

func (m *mockRequirementHookActionLogRepository) Save(ctx context.Context, log *RequirementHookActionLog) error {
	key := log.HookConfigID + "_" + log.RequirementID
	m.logs[key] = log
	return nil
}
func (m *mockRequirementHookActionLogRepository) FindByRequirementID(ctx context.Context, requirementID string) ([]*RequirementHookActionLog, error) {
	var result []*RequirementHookActionLog
	for _, log := range m.logs {
		if log.RequirementID == requirementID {
			result = append(result, log)
		}
	}
	return result, nil
}
func (m *mockRequirementHookActionLogRepository) FindByHookConfigID(ctx context.Context, hookConfigID string, limit int) ([]*RequirementHookActionLog, error) {
	var result []*RequirementHookActionLog
	for _, log := range m.logs {
		if log.HookConfigID == hookConfigID {
			result = append(result, log)
		}
	}
	return result, nil
}
func (m *mockRequirementHookActionLogRepository) FindByHookConfigAndRequirement(ctx context.Context, hookConfigID, requirementID string) (*RequirementHookActionLog, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	key := hookConfigID + "_" + requirementID
	return m.logs[key], nil
}

type mockActionExecutor struct {
	supportedType string
	executeFunc   func(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error)
	callCount     int
}

func (m *mockActionExecutor) Supports(actionType string) bool {
	return m.supportedType == actionType
}
func (m *mockActionExecutor) Execute(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error) {
	m.callCount++
	if m.executeFunc != nil {
		return m.executeFunc(ctx, config, req, change)
	}
	return &ActionResult{Success: true, Output: "mock-output"}, nil
}

type mockConfigurableHookLogger struct {
	debugLogs []string
	infoLogs  []string
	errorLogs []string
}

func (m *mockConfigurableHookLogger) Debug(msg string, fields ...RequirementStateHookLogField) {
	m.debugLogs = append(m.debugLogs, msg)
}
func (m *mockConfigurableHookLogger) Info(msg string, fields ...RequirementStateHookLogField) {
	m.infoLogs = append(m.infoLogs, msg)
}
func (m *mockConfigurableHookLogger) Error(msg string, fields ...RequirementStateHookLogField) {
	m.errorLogs = append(m.errorLogs, msg)
}

type mockIDGenerator struct {
	counter int
}

func (m *mockIDGenerator) Generate() string {
	m.counter++
	return fmt.Sprintf("id-%d", m.counter)
}

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

func TestHookContext_ConcurrentSafety(t *testing.T) {
	hc := NewHookContext(context.Background())
	var wg sync.WaitGroup
	numGoroutines := 50
	numOps := 100

	// 并发写入 values
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				hc.WithValue(fmt.Sprintf("key-%d-%d", id, j), j)
			}
		}(i)
	}

	// 并发读取 values
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				_ = hc.Get(fmt.Sprintf("key-%d-%d", id, j))
			}
		}(i)
	}

	// 并发添加 hook
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				hc.AddHook(fmt.Sprintf("hook-%d", id))
			}
		}(i)
	}

	// 并发读取 hooks
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				_ = hc.GetHooks()
			}
		}(i)
	}

	// 并发添加 error
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				hc.AddError(fmt.Errorf("error-%d-%d", id, j), "hook", "phase")
			}
		}(i)
	}

	// 并发读取 errors
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				_ = hc.GetErrors()
				_ = hc.HasErrors()
			}
		}(i)
	}

	// 并发写入 metadata
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				hc.SetMetadata(fmt.Sprintf("meta-%d", id), fmt.Sprintf("val-%d", j))
			}
		}(i)
	}

	// 并发读取 metadata
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				_ = hc.GetMetadata(fmt.Sprintf("meta-%d", id))
			}
		}(i)
	}

	wg.Wait()

	// 验证最终结果
	hooks := hc.GetHooks()
	if len(hooks) != numGoroutines*numOps {
		t.Errorf("期望 hooks 长度为 %d, 实际为 %d", numGoroutines*numOps, len(hooks))
	}

	errs := hc.GetErrors()
	if len(errs) != numGoroutines*numOps {
		t.Errorf("期望 errors 长度为 %d, 实际为 %d", numGoroutines*numOps, len(errs))
	}
}

// ============================================================================
// requirementStateHookRegistry Tests
// ============================================================================

func TestRequirementStateHookRegistry_Register(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	hook := newMockRequirementStateHook("hook-1", 10, true)

	err := registry.Register(hook)
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	retrieved := registry.Get("hook-1")
	if retrieved == nil {
		t.Fatal("期望 Get(hook-1) 返回已注册的 hook")
	}
	if retrieved.Name() != "hook-1" {
		t.Errorf("期望返回 hook-1, 实际为 %s", retrieved.Name())
	}
}

func TestRequirementStateHookRegistry_RegisterNil(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	err := registry.Register(nil)
	if err == nil {
		t.Fatal("期望 Register(nil) 返回错误")
	}
}

func TestRequirementStateHookRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	hook := newMockRequirementStateHook("hook-1", 10, true)

	if err := registry.Register(hook); err != nil {
		t.Fatalf("第一次注册失败: %v", err)
	}

	err := registry.Register(hook)
	if err == nil {
		t.Fatal("期望重复注册返回错误")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Errorf("期望错误包含 already registered, 实际为 %v", err)
	}
}

func TestRequirementStateHookRegistry_Unregister(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	hook := newMockRequirementStateHook("hook-1", 10, true)
	registry.Register(hook)

	err := registry.Unregister("hook-1")
	if err != nil {
		t.Fatalf("注销失败: %v", err)
	}

	if registry.Get("hook-1") != nil {
		t.Error("期望注销后 Get(hook-1) 返回 nil")
	}
}

func TestRequirementStateHookRegistry_UnregisterNotExist(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	err := registry.Unregister("nonexistent")
	if err == nil {
		t.Fatal("期望注销不存在的 hook 返回错误")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("期望错误包含 not found, 实际为 %v", err)
	}
}

func TestRequirementStateHookRegistry_Get_List(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	hook1 := newMockRequirementStateHook("hook-1", 10, true)
	hook2 := newMockRequirementStateHook("hook-2", 20, true)

	registry.Register(hook1)
	registry.Register(hook2)

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("期望 List() 返回 2 个 hook, 实际为 %d", len(list))
	}

	// Get 不存在的 hook
	if registry.Get("nonexistent") != nil {
		t.Error("期望 Get(nonexistent) 返回 nil")
	}
}

func TestRequirementStateHookRegistry_Enable_Disable(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	hook := newMockRequirementStateHook("hook-1", 10, true)
	registry.Register(hook)

	if err := registry.Disable("hook-1"); err != nil {
		t.Fatalf("禁用失败: %v", err)
	}
	if registry.Get("hook-1").Enabled() {
		t.Error("期望 Disable 后 hook 为 disabled")
	}

	if err := registry.Enable("hook-1"); err != nil {
		t.Fatalf("启用失败: %v", err)
	}
	if !registry.Get("hook-1").Enabled() {
		t.Error("期望 Enable 后 hook 为 enabled")
	}

	if err := registry.Enable("nonexistent"); err == nil {
		t.Error("期望 Enable 不存在的 hook 返回错误")
	}
	if err := registry.Disable("nonexistent"); err == nil {
		t.Error("期望 Disable 不存在的 hook 返回错误")
	}
}

// ============================================================================
// RequirementStateHookExecutor Tests
// ============================================================================

func createTestRequirement(t *testing.T) *Requirement {
	req, err := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"测试需求",
		"测试描述",
		"验收标准",
		"/tmp/workspace",
	)
	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}
	return req
}

func TestRequirementStateHookExecutor_Execute(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	logger := &mockRequirementStateHookLogger{}
	executor := NewRequirementStateHookExecutor(registry, logger)

	hook1 := newMockRequirementStateHook("hook-1", 20, true)
	hook2 := newMockRequirementStateHook("hook-2", 10, true)

	registry.Register(hook1)
	registry.Register(hook2)

	req := createTestRequirement(t)
	change := &StateChange{
		FromStatus: RequirementStatusTodo,
		ToStatus:   RequirementStatusCoding,
		Trigger:    "test-trigger",
		Timestamp:  time.Now(),
	}

	executor.Execute(context.Background(), req, change)

	if hook1.callCount != 1 {
		t.Errorf("期望 hook-1 被调用 1 次, 实际为 %d", hook1.callCount)
	}
	if hook2.callCount != 1 {
		t.Errorf("期望 hook-2 被调用 1 次, 实际为 %d", hook2.callCount)
	}

	// 验证日志
	if len(logger.debugLogs) != 2 {
		t.Errorf("期望 2 条 debug 日志, 实际为 %d", len(logger.debugLogs))
	}
}

func TestRequirementStateHookExecutor_Execute_PrioritySort(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	logger := &mockRequirementStateHookLogger{}
	executor := NewRequirementStateHookExecutor(registry, logger)

	var executionOrder []string
	hook1 := newMockRequirementStateHook("hook-1", 30, true)
	hook1.onChangeFunc = func(ctx context.Context, req *Requirement, change *StateChange) error {
		executionOrder = append(executionOrder, "hook-1")
		return nil
	}
	hook2 := newMockRequirementStateHook("hook-2", 10, true)
	hook2.onChangeFunc = func(ctx context.Context, req *Requirement, change *StateChange) error {
		executionOrder = append(executionOrder, "hook-2")
		return nil
	}
	hook3 := newMockRequirementStateHook("hook-3", 20, true)
	hook3.onChangeFunc = func(ctx context.Context, req *Requirement, change *StateChange) error {
		executionOrder = append(executionOrder, "hook-3")
		return nil
	}

	registry.Register(hook1)
	registry.Register(hook2)
	registry.Register(hook3)

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "test"}
	executor.Execute(context.Background(), req, change)

	if len(executionOrder) != 3 {
		t.Fatalf("期望执行 3 个 hook, 实际顺序: %v", executionOrder)
	}
	if executionOrder[0] != "hook-2" || executionOrder[1] != "hook-3" || executionOrder[2] != "hook-1" {
		t.Errorf("期望按优先级排序 [hook-2, hook-3, hook-1], 实际为 %v", executionOrder)
	}
}

func TestRequirementStateHookExecutor_Execute_OnlyEnabled(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	logger := &mockRequirementStateHookLogger{}
	executor := NewRequirementStateHookExecutor(registry, logger)

	hook1 := newMockRequirementStateHook("hook-1", 10, true)
	hook2 := newMockRequirementStateHook("hook-2", 20, false)

	registry.Register(hook1)
	registry.Register(hook2)

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "test"}
	executor.Execute(context.Background(), req, change)

	if hook1.callCount != 1 {
		t.Errorf("期望 hook-1 被调用 1 次, 实际为 %d", hook1.callCount)
	}
	if hook2.callCount != 0 {
		t.Errorf("期望被禁用的 hook-2 不被调用, 实际为 %d", hook2.callCount)
	}
}

func TestRequirementStateHookExecutor_Execute_ErrorHandling(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	logger := &mockRequirementStateHookLogger{}
	executor := NewRequirementStateHookExecutor(registry, logger)

	hook1 := newMockRequirementStateHook("hook-1", 10, true)
	hook1.onChangeFunc = func(ctx context.Context, req *Requirement, change *StateChange) error {
		return errors.New("hook-1 error")
	}
	hook2 := newMockRequirementStateHook("hook-2", 20, true)

	registry.Register(hook1)
	registry.Register(hook2)

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "test"}
	executor.Execute(context.Background(), req, change)

	// 单个 hook 失败不应影响其他 hook 执行
	if hook2.callCount != 1 {
		t.Errorf("期望 hook-2 仍然被调用 1 次, 实际为 %d", hook2.callCount)
	}

	// 验证错误日志
	if len(logger.errorLogs) != 1 {
		t.Errorf("期望 1 条 error 日志, 实际为 %d", len(logger.errorLogs))
	}
}

func TestRequirementStateHookExecutor_sortByPriority(t *testing.T) {
	registry := NewRequirementStateHookRegistry()
	executor := NewRequirementStateHookExecutor(registry, nil)

	hooks := []RequirementStateHook{
		newMockRequirementStateHook("hook-c", 30, true),
		newMockRequirementStateHook("hook-a", 10, true),
		newMockRequirementStateHook("hook-b", 20, true),
	}

	sorted := executor.sortByPriority(hooks)

	if sorted[0].Name() != "hook-a" {
		t.Errorf("期望第一个为 hook-a, 实际为 %s", sorted[0].Name())
	}
	if sorted[1].Name() != "hook-b" {
		t.Errorf("期望第二个为 hook-b, 实际为 %s", sorted[1].Name())
	}
	if sorted[2].Name() != "hook-c" {
		t.Errorf("期望第三个为 hook-c, 实际为 %s", sorted[2].Name())
	}
}

// ============================================================================
// ConfigurableHookExecutor Tests
// ============================================================================

func TestConfigurableHookExecutor_Execute(t *testing.T) {
	configRepo := newMockRequirementHookConfigRepository()
	logRepo := newMockRequirementHookActionLogRepository()
	logger := &mockConfigurableHookLogger{}
	idGen := &mockIDGenerator{}

	executor := NewConfigurableHookExecutor(configRepo, logRepo, nil, logger, idGen)

	actionExec := &mockActionExecutor{
		supportedType: "coding_agent",
		executeFunc: func(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error) {
			return &ActionResult{Success: true, Output: "done"}, nil
		},
	}
	executor.AddExecutor(actionExec)

	config := &RequirementHookConfig{
		ID:           "config-1",
		ProjectID:    "proj-001",
		Name:         "测试Hook",
		TriggerPoint: "mark_coding",
		ActionType:   "coding_agent",
		ActionConfig: "{}",
		Enabled:      true,
		Priority:     10,
	}
	configRepo.Save(context.Background(), config)

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "mark_coding"}

	executor.Execute(context.Background(), "mark_coding", req, change)

	// 验证日志
	savedLog, _ := logRepo.FindByHookConfigAndRequirement(context.Background(), "config-1", req.ID().String())
	if savedLog == nil {
		t.Fatal("期望执行日志被保存")
	}
	if savedLog.Status != "success" {
		t.Errorf("期望日志状态为 success, 实际为 %s", savedLog.Status)
	}
	if savedLog.Result != "done" {
		t.Errorf("期望日志结果为 done, 实际为 %s", savedLog.Result)
	}
	if savedLog.Error != "" {
		t.Errorf("期望日志错误为空, 实际为 %s", savedLog.Error)
	}
}

func TestConfigurableHookExecutor_Execute_NoConfigRepo(t *testing.T) {
	logger := &mockConfigurableHookLogger{}
	executor := NewConfigurableHookExecutor(nil, nil, nil, logger, nil)

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "mark_coding"}

	// 不应 panic
	executor.Execute(context.Background(), "mark_coding", req, change)
}

func TestConfigurableHookExecutor_Execute_LoadError(t *testing.T) {
	configRepo := newMockRequirementHookConfigRepository()
	configRepo.findErr = errors.New("db error")
	logger := &mockConfigurableHookLogger{}
	executor := NewConfigurableHookExecutor(configRepo, nil, nil, logger, nil)

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "mark_coding"}
	executor.Execute(context.Background(), "mark_coding", req, change)

	if len(logger.errorLogs) != 1 {
		t.Errorf("期望 1 条 error 日志, 实际为 %d", len(logger.errorLogs))
	}
}

func TestConfigurableHookExecutor_Execute_NoConfigs(t *testing.T) {
	configRepo := newMockRequirementHookConfigRepository()
	logger := &mockConfigurableHookLogger{}
	executor := NewConfigurableHookExecutor(configRepo, nil, nil, logger, nil)

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "mark_coding"}
	executor.Execute(context.Background(), "mark_coding", req, change)

	// 没有配置时不应产生 error 日志
	if len(logger.errorLogs) != 0 {
		t.Errorf("期望 0 条 error 日志, 实际为 %d", len(logger.errorLogs))
	}
}

func TestConfigurableHookExecutor_Execute_PrioritySort(t *testing.T) {
	configRepo := newMockRequirementHookConfigRepository()
	logRepo := newMockRequirementHookActionLogRepository()
	logger := &mockConfigurableHookLogger{}
	idGen := &mockIDGenerator{}

	executor := NewConfigurableHookExecutor(configRepo, logRepo, nil, logger, idGen)

	var execOrder []string
	actionExec := &mockActionExecutor{
		supportedType: "coding_agent",
		executeFunc: func(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error) {
			execOrder = append(execOrder, config.ID)
			return &ActionResult{Success: true}, nil
		},
	}
	executor.AddExecutor(actionExec)

	configRepo.Save(context.Background(), &RequirementHookConfig{
		ID: "config-high", TriggerPoint: "test", ActionType: "coding_agent", Enabled: true, Priority: 30,
	})
	configRepo.Save(context.Background(), &RequirementHookConfig{
		ID: "config-low", TriggerPoint: "test", ActionType: "coding_agent", Enabled: true, Priority: 10,
	})
	configRepo.Save(context.Background(), &RequirementHookConfig{
		ID: "config-mid", TriggerPoint: "test", ActionType: "coding_agent", Enabled: true, Priority: 20,
	})

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "test"}
	executor.Execute(context.Background(), "test", req, change)

	if len(execOrder) != 3 {
		t.Fatalf("期望执行 3 个配置, 实际顺序: %v", execOrder)
	}
	if execOrder[0] != "config-low" || execOrder[1] != "config-mid" || execOrder[2] != "config-high" {
		t.Errorf("期望按优先级排序 [config-low, config-mid, config-high], 实际为 %v", execOrder)
	}
}

func TestConfigurableHookExecutor_Execute_NoExecutor(t *testing.T) {
	configRepo := newMockRequirementHookConfigRepository()
	logRepo := newMockRequirementHookActionLogRepository()
	logger := &mockConfigurableHookLogger{}
	idGen := &mockIDGenerator{}

	executor := NewConfigurableHookExecutor(configRepo, logRepo, []ActionExecutor{}, logger, idGen)

	configRepo.Save(context.Background(), &RequirementHookConfig{
		ID: "config-1", TriggerPoint: "test", ActionType: "unknown_type", Enabled: true, Priority: 10,
	})

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "test"}
	executor.Execute(context.Background(), "test", req, change)

	savedLog, _ := logRepo.FindByHookConfigAndRequirement(context.Background(), "config-1", req.ID().String())
	if savedLog == nil {
		t.Fatal("期望执行日志被保存")
	}
	if savedLog.Status != "failed" {
		t.Errorf("期望日志状态为 failed, 实际为 %s", savedLog.Status)
	}
	if !strings.Contains(savedLog.Error, "no executor for action type") {
		t.Errorf("期望错误包含 no executor for action type, 实际为 %s", savedLog.Error)
	}
}

func TestConfigurableHookExecutor_Execute_AlreadyExecuted(t *testing.T) {
	configRepo := newMockRequirementHookConfigRepository()
	logRepo := newMockRequirementHookActionLogRepository()
	logger := &mockConfigurableHookLogger{}
	idGen := &mockIDGenerator{}

	executor := NewConfigurableHookExecutor(configRepo, logRepo, nil, logger, idGen)

	actionExec := &mockActionExecutor{
		supportedType: "coding_agent",
		executeFunc: func(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error) {
			return &ActionResult{Success: true}, nil
		},
	}
	executor.AddExecutor(actionExec)

	configRepo.Save(context.Background(), &RequirementHookConfig{
		ID: "config-1", TriggerPoint: "test", ActionType: "coding_agent", Enabled: true, Priority: 10,
	})

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "test"}

	// 预先保存一个成功的执行日志
	now := time.Now()
	logRepo.Save(context.Background(), &RequirementHookActionLog{
		ID:            "log-1",
		HookConfigID:  "config-1",
		RequirementID: req.ID().String(),
		Status:        "success",
		CompletedAt:   &now,
	})

	executor.Execute(context.Background(), "test", req, change)

	// 验证没有新增执行记录（即跳过了执行）
	logs, _ := logRepo.FindByHookConfigID(context.Background(), "config-1", 10)
	if len(logs) != 1 {
		t.Errorf("期望只有 1 条日志记录, 实际为 %d", len(logs))
	}

	// 验证 executor 未被调用（避免 mock 按 key 覆盖导致的假阳性）
	if actionExec.callCount != 0 {
		t.Errorf("期望 executor 未被调用, 实际调用 %d 次", actionExec.callCount)
	}
}

func TestConfigurableHookExecutor_Execute_ExecutorError(t *testing.T) {
	configRepo := newMockRequirementHookConfigRepository()
	logRepo := newMockRequirementHookActionLogRepository()
	logger := &mockConfigurableHookLogger{}
	idGen := &mockIDGenerator{}

	executor := NewConfigurableHookExecutor(configRepo, logRepo, nil, logger, idGen)

	actionExec := &mockActionExecutor{
		supportedType: "coding_agent",
		executeFunc: func(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error) {
			return nil, errors.New("execution failed")
		},
	}
	executor.AddExecutor(actionExec)

	configRepo.Save(context.Background(), &RequirementHookConfig{
		ID: "config-1", TriggerPoint: "test", ActionType: "coding_agent", Enabled: true, Priority: 10,
	})

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "test"}
	executor.Execute(context.Background(), "test", req, change)

	savedLog, _ := logRepo.FindByHookConfigAndRequirement(context.Background(), "config-1", req.ID().String())
	if savedLog == nil {
		t.Fatal("期望执行日志被保存")
	}
	if savedLog.Status != "failed" {
		t.Errorf("期望日志状态为 failed, 实际为 %s", savedLog.Status)
	}
	if savedLog.Error != "execution failed" {
		t.Errorf("期望日志错误为 execution failed, 实际为 %s", savedLog.Error)
	}
}

func TestConfigurableHookExecutor_AddExecutor(t *testing.T) {
	executor := NewConfigurableHookExecutor(nil, nil, nil, nil, nil)
	actionExec := &mockActionExecutor{supportedType: "test"}

	executor.AddExecutor(actionExec)

	if len(executor.executors) != 1 {
		t.Errorf("期望 executors 长度为 1, 实际为 %d", len(executor.executors))
	}
}

// ============================================================================
// Integration/Other Tests
// ============================================================================

func TestHookTypeConstants(t *testing.T) {
	tests := []struct {
		actual   HookType
		expected string
	}{
		{HookTypeLifecycle, "lifecycle"},
		{HookTypeLLM, "llm"},
		{HookTypeTool, "tool"},
		{HookTypeMessage, "message"},
		{HookTypeSkill, "skill"},
		{HookTypeMCP, "mcp"},
		{HookTypePrompt, "prompt"},
		{HookTypeSession, "session"},
		{HookTypeRequirement, "requirement"},
	}

	for _, tc := range tests {
		if string(tc.actual) != tc.expected {
			t.Errorf("期望 HookType 为 %s, 实际为 %s", tc.expected, tc.actual)
		}
	}
}

func TestLLMCallContext_DefaultValues(t *testing.T) {
	ctx := &LLMCallContext{
		Prompt:    "test prompt",
		UserInput: "hello",
		Model:     "claude",
	}
	if ctx.Prompt != "test prompt" {
		t.Error("Prompt 不匹配")
	}
	if ctx.UserInput != "hello" {
		t.Error("UserInput 不匹配")
	}
}

func TestToolCallContext_DefaultValues(t *testing.T) {
	ctx := &ToolCallContext{
		ToolName: "test-tool",
		SessionID: "sess-1",
	}
	if ctx.ToolName != "test-tool" {
		t.Error("ToolName 不匹配")
	}
	if ctx.SessionID != "sess-1" {
		t.Error("SessionID 不匹配")
	}
}

func TestUsage_TokenCounts(t *testing.T) {
	u := Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}
	if u.PromptTokens != 10 || u.CompletionTokens != 20 || u.TotalTokens != 30 {
		t.Error("Usage 字段值不匹配")
	}
}

func TestActionResult_Fields(t *testing.T) {
	result := &ActionResult{
		Success: true,
		Output:  "test-output",
		Error:   errors.New("test-error"),
	}
	if !result.Success || result.Output != "test-output" || result.Error == nil {
		t.Error("ActionResult 字段值不匹配")
	}
}

func TestTriggerAgentActionConfig_Fields(t *testing.T) {
	cfg := &TriggerAgentActionConfig{
		PromptTemplate:    "tpl",
		TimeoutMinutes:    10,
		WorkspaceTemplate: "ws",
	}
	if cfg.PromptTemplate != "tpl" || cfg.TimeoutMinutes != 10 || cfg.WorkspaceTemplate != "ws" {
		t.Error("TriggerAgentActionConfig 字段值不匹配")
	}
}

func TestNotificationActionConfig_Fields(t *testing.T) {
	cfg := &NotificationActionConfig{
		Channel:  "feishu",
		Template: "tmpl",
	}
	if cfg.Channel != "feishu" || cfg.Template != "tmpl" {
		t.Error("NotificationActionConfig 字段值不匹配")
	}
}

func TestWebhookActionConfig_Fields(t *testing.T) {
	cfg := &WebhookActionConfig{
		URL:          "http://example.com",
		Method:       "POST",
		Headers:      map[string]string{"X-Test": "1"},
		BodyTemplate: "{}",
	}
	if cfg.URL != "http://example.com" || cfg.Method != "POST" || cfg.BodyTemplate != "{}" {
		t.Error("WebhookActionConfig 字段值不匹配")
	}
}

func TestStateChange_Fields(t *testing.T) {
	sc := &StateChange{
		FromStatus: RequirementStatusTodo,
		ToStatus:   RequirementStatusCoding,
		Trigger:    "test",
		Reason:     "reason",
		Timestamp:  time.Now(),
	}
	if sc.FromStatus != RequirementStatusTodo || sc.ToStatus != RequirementStatusCoding {
		t.Error("StateChange 状态字段值不匹配")
	}
	if sc.Trigger != "test" || sc.Reason != "reason" {
		t.Error("StateChange 其他字段值不匹配")
	}
}

func TestRequirementStateEventConstants(t *testing.T) {
	events := []struct {
		actual   RequirementStateEvent
		expected string
	}{
		{RequirementEventDispatching, "dispatching"},
		{RequirementEventDispatched, "dispatched"},
		{RequirementEventDispatchFailed, "dispatch_failed"},
		{RequirementEventCodingStarted, "coding_started"},
		{RequirementEventCodingCompleted, "coding_completed"},
		{RequirementEventCodingFailed, "coding_failed"},
		{RequirementEventCompleted, "completed"},
	}

	for _, tc := range events {
		if string(tc.actual) != tc.expected {
			t.Errorf("期望 RequirementStateEvent 为 %s, 实际为 %s", tc.expected, tc.actual)
		}
	}
}

// ============================================================================
// HookExecutor (TaskHook) Tests
// ============================================================================

type mockTaskHook struct {
	name           string
	onCreatedErr   error
	onStartedErr   error
	onCompletedErr error
	onFailedErr    error
	onCancelledErr error
	onProgressErr  error
	callRecord     map[string]int
}

func newMockTaskHook(name string) *mockTaskHook {
	return &mockTaskHook{
		name:       name,
		callRecord: make(map[string]int),
	}
}

func (m *mockTaskHook) Name() string { return m.name }
func (m *mockTaskHook) OnTaskCreated(ctx context.Context, task *Task) error {
	m.callRecord["OnTaskCreated"]++
	return m.onCreatedErr
}
func (m *mockTaskHook) OnTaskStarted(ctx context.Context, task *Task) error {
	m.callRecord["OnTaskStarted"]++
	return m.onStartedErr
}
func (m *mockTaskHook) OnTaskCompleted(ctx context.Context, task *Task) error {
	m.callRecord["OnTaskCompleted"]++
	return m.onCompletedErr
}
func (m *mockTaskHook) OnTaskFailed(ctx context.Context, task *Task, err error) error {
	m.callRecord["OnTaskFailed"]++
	return m.onFailedErr
}
func (m *mockTaskHook) OnTaskCancelled(ctx context.Context, task *Task) error {
	m.callRecord["OnTaskCancelled"]++
	return m.onCancelledErr
}
func (m *mockTaskHook) OnTaskProgressUpdated(ctx context.Context, task *Task) error {
	m.callRecord["OnTaskProgressUpdated"]++
	return m.onProgressErr
}

type mockHookRegistry struct {
	hooks TaskHooks
}

func (m *mockHookRegistry) Register(hook TaskHook) error {
	m.hooks = append(m.hooks, hook)
	return nil
}
func (m *mockHookRegistry) Unregister(name string) error {
	return nil
}
func (m *mockHookRegistry) GetHooks() TaskHooks {
	return m.hooks
}

func TestNewHookExecutor(t *testing.T) {
	registry := &mockHookRegistry{}
	executor := NewHookExecutor(registry)
	if executor == nil {
		t.Fatal("期望 NewHookExecutor 返回非 nil")
	}
	if executor.registry != registry {
		t.Error("期望 registry 被正确设置")
	}
}

func TestHookExecutor_ExecuteOnTaskCreated(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	task := &Task{}
	err := executor.ExecuteOnTaskCreated(context.Background(), task)
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if hook.callRecord["OnTaskCreated"] != 1 {
		t.Errorf("期望 OnTaskCreated 被调用 1 次, 实际为 %d", hook.callRecord["OnTaskCreated"])
	}
}

func TestHookExecutor_ExecuteOnTaskCreated_Error(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	hook.onCreatedErr = errors.New("create error")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskCreated(context.Background(), &Task{})
	if err != hook.onCreatedErr {
		t.Errorf("期望返回 create error, 实际为 %v", err)
	}
}

func TestHookExecutor_ExecuteOnTaskStarted(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskStarted(context.Background(), &Task{})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if hook.callRecord["OnTaskStarted"] != 1 {
		t.Errorf("期望 OnTaskStarted 被调用 1 次, 实际为 %d", hook.callRecord["OnTaskStarted"])
	}
}

func TestHookExecutor_ExecuteOnTaskStarted_Error(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	hook.onStartedErr = errors.New("start error")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskStarted(context.Background(), &Task{})
	if err != hook.onStartedErr {
		t.Errorf("期望返回 start error, 实际为 %v", err)
	}
}

func TestHookExecutor_ExecuteOnTaskCompleted(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskCompleted(context.Background(), &Task{})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if hook.callRecord["OnTaskCompleted"] != 1 {
		t.Errorf("期望 OnTaskCompleted 被调用 1 次, 实际为 %d", hook.callRecord["OnTaskCompleted"])
	}
}

func TestHookExecutor_ExecuteOnTaskCompleted_Error(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	hook.onCompletedErr = errors.New("complete error")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskCompleted(context.Background(), &Task{})
	if err != hook.onCompletedErr {
		t.Errorf("期望返回 complete error, 实际为 %v", err)
	}
}

func TestHookExecutor_ExecuteOnTaskFailed(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	origErr := errors.New("original failure")
	err := executor.ExecuteOnTaskFailed(context.Background(), &Task{}, origErr)
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if hook.callRecord["OnTaskFailed"] != 1 {
		t.Errorf("期望 OnTaskFailed 被调用 1 次, 实际为 %d", hook.callRecord["OnTaskFailed"])
	}
}

func TestHookExecutor_ExecuteOnTaskFailed_Error(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	hook.onFailedErr = errors.New("failed hook error")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskFailed(context.Background(), &Task{}, errors.New("orig"))
	if err != hook.onFailedErr {
		t.Errorf("期望返回 failed hook error, 实际为 %v", err)
	}
}

func TestHookExecutor_ExecuteOnTaskCancelled(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskCancelled(context.Background(), &Task{})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if hook.callRecord["OnTaskCancelled"] != 1 {
		t.Errorf("期望 OnTaskCancelled 被调用 1 次, 实际为 %d", hook.callRecord["OnTaskCancelled"])
	}
}

func TestHookExecutor_ExecuteOnTaskCancelled_Error(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	hook.onCancelledErr = errors.New("cancel error")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskCancelled(context.Background(), &Task{})
	if err != hook.onCancelledErr {
		t.Errorf("期望返回 cancel error, 实际为 %v", err)
	}
}

func TestHookExecutor_ExecuteOnTaskProgressUpdated(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskProgressUpdated(context.Background(), &Task{})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if hook.callRecord["OnTaskProgressUpdated"] != 1 {
		t.Errorf("期望 OnTaskProgressUpdated 被调用 1 次, 实际为 %d", hook.callRecord["OnTaskProgressUpdated"])
	}
}

func TestHookExecutor_ExecuteOnTaskProgressUpdated_Error(t *testing.T) {
	hook := newMockTaskHook("task-hook-1")
	hook.onProgressErr = errors.New("progress error")
	registry := &mockHookRegistry{hooks: TaskHooks{hook}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskProgressUpdated(context.Background(), &Task{})
	if err != hook.onProgressErr {
		t.Errorf("期望返回 progress error, 实际为 %v", err)
	}
}

func TestHookExecutor_MultipleHooks(t *testing.T) {
	hook1 := newMockTaskHook("task-hook-1")
	hook2 := newMockTaskHook("task-hook-2")
	registry := &mockHookRegistry{hooks: TaskHooks{hook1, hook2}}
	executor := NewHookExecutor(registry)

	err := executor.ExecuteOnTaskCreated(context.Background(), &Task{})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if hook1.callRecord["OnTaskCreated"] != 1 || hook2.callRecord["OnTaskCreated"] != 1 {
		t.Error("期望所有 hook 都被调用")
	}
}

// ============================================================================
// ConfigurableHookExecutor edge case: logRepo save error during init
// ============================================================================

type mockFailingLogRepo struct {
	mockRequirementHookActionLogRepository
	saveShouldFail bool
}

func (m *mockFailingLogRepo) Save(ctx context.Context, log *RequirementHookActionLog) error {
	if m.saveShouldFail {
		return errors.New("save failed")
	}
	return m.mockRequirementHookActionLogRepository.Save(ctx, log)
}

// callCountLogRepo 跟踪 Save 调用次数，第一次（初始保存）成功，第二次（更新保存）失败
type callCountLogRepo struct {
	mockRequirementHookActionLogRepository
	saveCount int
}

func (m *callCountLogRepo) Save(ctx context.Context, log *RequirementHookActionLog) error {
	m.saveCount++
	if m.saveCount == 2 {
		return errors.New("save failed on second call (update)")
	}
	return m.mockRequirementHookActionLogRepository.Save(ctx, log)
}

func TestConfigurableHookExecutor_Execute_LogSaveInitError(t *testing.T) {
	configRepo := newMockRequirementHookConfigRepository()
	logRepo := &mockFailingLogRepo{}
	logRepo.logs = make(map[string]*RequirementHookActionLog)
	logger := &mockConfigurableHookLogger{}
	idGen := &mockIDGenerator{}

	executor := NewConfigurableHookExecutor(configRepo, logRepo, nil, logger, idGen)

	actionExec := &mockActionExecutor{
		supportedType: "coding_agent",
		executeFunc: func(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error) {
			return &ActionResult{Success: true, Output: "done"}, nil
		},
	}
	executor.AddExecutor(actionExec)

	configRepo.Save(context.Background(), &RequirementHookConfig{
		ID: "config-1", TriggerPoint: "test", ActionType: "coding_agent", Enabled: true, Priority: 10,
	})

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "test"}

	// First call: save fails on initial log creation
	logRepo.saveShouldFail = true
	executor.Execute(context.Background(), "test", req, change)

	if len(logger.errorLogs) != 1 {
		t.Errorf("期望 1 条 error 日志(保存初始日志失败), 实际为 %d", len(logger.errorLogs))
	}
}

func TestConfigurableHookExecutor_Execute_LogSaveUpdateError(t *testing.T) {
	configRepo := newMockRequirementHookConfigRepository()
	logRepo := &callCountLogRepo{}
	logRepo.logs = make(map[string]*RequirementHookActionLog)
	logger := &mockConfigurableHookLogger{}
	idGen := &mockIDGenerator{}

	executor := NewConfigurableHookExecutor(configRepo, logRepo, nil, logger, idGen)

	actionExec := &mockActionExecutor{
		supportedType: "coding_agent",
		executeFunc: func(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error) {
			return &ActionResult{Success: true, Output: "done"}, nil
		},
	}
	executor.AddExecutor(actionExec)

	configRepo.Save(context.Background(), &RequirementHookConfig{
		ID: "config-1", TriggerPoint: "test", ActionType: "coding_agent", Enabled: true, Priority: 10,
	})

	req := createTestRequirement(t)
	change := &StateChange{Trigger: "test"}

	// First call: initial save succeeds, update save fails
	executor.Execute(context.Background(), "test", req, change)

	// Verify Save was called 3 times (initial save + running update + final update)
	// Only the 2nd call (running update) should fail
	if logRepo.saveCount != 3 {
		t.Errorf("期望 Save 被调用 3 次，实际调用 %d 次", logRepo.saveCount)
	}

	// No error log expected because the log update failures are silently ignored
	if len(logger.errorLogs) >= 1 {
		t.Errorf("期望没有 error 日志，实际有 %d 条", len(logger.errorLogs))
	}
}
