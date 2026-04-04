package application

import (
	"sync"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
)

// setupTestRegistry 创建一个新的 TaskRegistry 实例用于测试
// 注意：由于 GetTaskRegistry 是单例，我们需要重置 defaultRegistry
func setupTestRegistry(t *testing.T) *TaskRegistry {
	registry := GetTaskRegistry()
	// 获取锁并直接重新初始化所有 map，确保测试之间相互独立
	registry.mu.Lock()
	registry.taskContexts = make(map[string]*TaskContext)
	registry.traceContexts = make(map[string]*TraceContext)
	registry.todoLists = make(map[string]*TodoList)
	registry.mu.Unlock()
	return registry
}

// createTestTraceContext 创建一个用于测试的 TraceContext
func createTestTraceContext(traceID, rootTaskID string) *TraceContext {
	return NewTraceContext(traceID, rootTaskID, nil)
}

// createTestTaskContext 创建一个用于测试的 TaskContext
func createTestTaskContext(taskID string, taskType domain.TaskType, goal string) *TaskContext {
	return NewTaskContext(taskID, taskType, goal, "span-1", "", nil)
}

// createTestTodoList 创建一个用于测试的 TodoList
func createTestTodoList(taskID string) *TodoList {
	return NewTodoList(taskID)
}

// TestGetTaskRegistry_Singleton 测试 GetTaskRegistry 返回单例实例
func TestGetTaskRegistry_Singleton(t *testing.T) {
	registry1 := GetTaskRegistry()
	registry2 := GetTaskRegistry()

	if registry1 != registry2 {
		t.Error("GetTaskRegistry 应该返回同一个实例")
	}
}

// TestGetTaskRegistry_Concurrent 测试 GetTaskRegistry 的并发安全性
func TestGetTaskRegistry_Concurrent(t *testing.T) {
	const numGoroutines = 100
	var wg sync.WaitGroup
	registries := make([]*TaskRegistry, numGoroutines)
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			registry := GetTaskRegistry()
			mu.Lock()
			registries[index] = registry
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// 验证所有 goroutine 获取的都是同一个实例
	first := registries[0]
	for i, r := range registries {
		if r != first {
			t.Errorf("goroutine %d 获取的 registry 与其他不同", i)
		}
	}
}

// TestTaskRegistry_RegisterTraceContext 测试注册 TraceContext
func TestTaskRegistry_RegisterTraceContext(t *testing.T) {
	registry := setupTestRegistry(t)

	tc := createTestTraceContext("trace-1", "task-1")
	registry.RegisterTraceContext(tc)

	// 验证可以获取到注册的 TraceContext
	got := registry.GetTraceContext("trace-1")
	if got == nil {
		t.Fatal("应该能获取到已注册的 TraceContext")
	}
	if got.TraceID != "trace-1" {
		t.Errorf("期望 TraceID 为 'trace-1', 实际为 '%s'", got.TraceID)
	}
	if got.RootTaskID != "task-1" {
		t.Errorf("期望 RootTaskID 为 'task-1', 实际为 '%s'", got.RootTaskID)
	}
}

// TestTaskRegistry_RegisterTraceContext_Overwrite 测试重复注册 TraceContext 会覆盖
func TestTaskRegistry_RegisterTraceContext_Overwrite(t *testing.T) {
	registry := setupTestRegistry(t)

	tc1 := createTestTraceContext("trace-1", "task-1")
	registry.RegisterTraceContext(tc1)

	tc2 := createTestTraceContext("trace-1", "task-2")
	registry.RegisterTraceContext(tc2)

	// 验证被覆盖
	got := registry.GetTraceContext("trace-1")
	if got.RootTaskID != "task-2" {
		t.Errorf("重复注册应该覆盖，期望 RootTaskID 为 'task-2', 实际为 '%s'", got.RootTaskID)
	}
}

// TestTaskRegistry_GetTraceContext_NotFound 测试获取不存在的 TraceContext
func TestTaskRegistry_GetTraceContext_NotFound(t *testing.T) {
	registry := setupTestRegistry(t)

	got := registry.GetTraceContext("non-existent-trace")
	if got != nil {
		t.Error("获取不存在的 traceID 应该返回 nil")
	}
}

// TestTaskRegistry_GetTraceContextByTaskID 测试通过 RootTaskID 查找 TraceContext
func TestTaskRegistry_GetTraceContextByTaskID(t *testing.T) {
	registry := setupTestRegistry(t)

	tc1 := createTestTraceContext("trace-1", "task-1")
	tc2 := createTestTraceContext("trace-2", "task-2")
	registry.RegisterTraceContext(tc1)
	registry.RegisterTraceContext(tc2)

	// 通过 RootTaskID 查找
	got := registry.GetTraceContextByTaskID("task-1")
	if got == nil {
		t.Fatal("应该能找到对应的 TraceContext")
	}
	if got.TraceID != "trace-1" {
		t.Errorf("期望找到 trace-1, 实际为 '%s'", got.TraceID)
	}
}

// TestTaskRegistry_GetTraceContextByTaskID_NotFound 测试通过不存在的 RootTaskID 查找
func TestTaskRegistry_GetTraceContextByTaskID_NotFound(t *testing.T) {
	registry := setupTestRegistry(t)

	tc := createTestTraceContext("trace-1", "task-1")
	registry.RegisterTraceContext(tc)

	got := registry.GetTraceContextByTaskID("non-existent-task")
	if got != nil {
		t.Error("查找不存在的 taskID 应该返回 nil")
	}
}

// TestTaskRegistry_UnregisterTraceContext 测试注销 TraceContext
func TestTaskRegistry_UnregisterTraceContext(t *testing.T) {
	registry := setupTestRegistry(t)

	tc := createTestTraceContext("trace-1", "task-1")
	registry.RegisterTraceContext(tc)

	// 验证已注册
	if registry.GetTraceContext("trace-1") == nil {
		t.Fatal("TraceContext 应该已注册")
	}

	// 注销
	registry.UnregisterTraceContext("trace-1")

	// 验证已注销
	if registry.GetTraceContext("trace-1") != nil {
		t.Error("注销后应该无法获取到 TraceContext")
	}
}

// TestTaskRegistry_UnregisterTraceContext_NotFound 测试注销不存在的 TraceContext 不 panic
func TestTaskRegistry_UnregisterTraceContext_NotFound(t *testing.T) {
	registry := setupTestRegistry(t)

	// 不应该 panic
	registry.UnregisterTraceContext("non-existent-trace")
}

// TestTaskRegistry_RegisterTaskContext 测试注册 TaskContext
func TestTaskRegistry_RegisterTaskContext(t *testing.T) {
	registry := setupTestRegistry(t)

	ctx := createTestTaskContext("task-1", domain.TaskTypeAgent, "测试目标")
	registry.RegisterTaskContext("task-1", ctx)

	// 验证可以获取
	got := registry.GetTaskContext("task-1")
	if got == nil {
		t.Fatal("应该能获取到已注册的 TaskContext")
	}
	if got.TaskID != "task-1" {
		t.Errorf("期望 TaskID 为 'task-1', 实际为 '%s'", got.TaskID)
	}
	if got.Goal != "测试目标" {
		t.Errorf("期望 Goal 为 '测试目标', 实际为 '%s'", got.Goal)
	}
}

// TestTaskRegistry_RegisterTaskContext_Overwrite 测试重复注册 TaskContext 会覆盖
func TestTaskRegistry_RegisterTaskContext_Overwrite(t *testing.T) {
	registry := setupTestRegistry(t)

	ctx1 := createTestTaskContext("task-1", domain.TaskTypeAgent, "目标1")
	ctx2 := createTestTaskContext("task-1", domain.TaskTypeCoding, "目标2")
	registry.RegisterTaskContext("task-1", ctx1)
	registry.RegisterTaskContext("task-1", ctx2)

	// 验证被覆盖
	got := registry.GetTaskContext("task-1")
	if got.Goal != "目标2" {
		t.Errorf("重复注册应该覆盖，期望 Goal 为 '目标2', 实际为 '%s'", got.Goal)
	}
}

// TestTaskRegistry_GetTaskContext_NotFound 测试获取不存在的 TaskContext
func TestTaskRegistry_GetTaskContext_NotFound(t *testing.T) {
	registry := setupTestRegistry(t)

	got := registry.GetTaskContext("non-existent-task")
	if got != nil {
		t.Error("获取不存在的 taskID 应该返回 nil")
	}
}

// TestTaskRegistry_UnregisterTaskContext 测试注销 TaskContext
func TestTaskRegistry_UnregisterTaskContext(t *testing.T) {
	registry := setupTestRegistry(t)

	ctx := createTestTaskContext("task-1", domain.TaskTypeAgent, "测试目标")
	registry.RegisterTaskContext("task-1", ctx)

	// 验证已注册
	if registry.GetTaskContext("task-1") == nil {
		t.Fatal("TaskContext 应该已注册")
	}

	// 注销
	registry.UnregisterTaskContext("task-1")

	// 验证已注销
	if registry.GetTaskContext("task-1") != nil {
		t.Error("注销后应该无法获取到 TaskContext")
	}
}

// TestTaskRegistry_UnregisterTaskContext_NotFound 测试注销不存在的 TaskContext 不 panic
func TestTaskRegistry_UnregisterTaskContext_NotFound(t *testing.T) {
	registry := setupTestRegistry(t)

	// 不应该 panic
	registry.UnregisterTaskContext("non-existent-task")
}

// TestTaskRegistry_RegisterTodoList 测试注册 TodoList
func TestTaskRegistry_RegisterTodoList(t *testing.T) {
	registry := setupTestRegistry(t)

	tl := createTestTodoList("task-1")
	registry.RegisterTodoList("task-1", tl)

	// 验证可以获取
	got := registry.GetTodoList("task-1")
	if got == nil {
		t.Fatal("应该能获取到已注册的 TodoList")
	}
	if got.TaskID != "task-1" {
		t.Errorf("期望 TaskID 为 'task-1', 实际为 '%s'", got.TaskID)
	}
}

// TestTaskRegistry_RegisterTodoList_Overwrite 测试重复注册 TodoList 会覆盖
func TestTaskRegistry_RegisterTodoList_Overwrite(t *testing.T) {
	registry := setupTestRegistry(t)

	tl1 := createTestTodoList("task-1")
	tl1.AddItem("sub-1", "子任务1", "thinking", "span-1", TodoStatusDistributed)

	tl2 := createTestTodoList("task-1")
	tl2.AddItem("sub-2", "子任务2", "coding", "span-2", TodoStatusDistributed)

	registry.RegisterTodoList("task-1", tl1)
	registry.RegisterTodoList("task-1", tl2)

	// 验证被覆盖
	got := registry.GetTodoList("task-1")
	if len(got.Items) != 1 || got.Items[0].SubTaskID != "sub-2" {
		t.Error("重复注册应该覆盖 TodoList")
	}
}

// TestTaskRegistry_GetTodoList_NotFound 测试获取不存在的 TodoList
func TestTaskRegistry_GetTodoList_NotFound(t *testing.T) {
	registry := setupTestRegistry(t)

	got := registry.GetTodoList("non-existent-task")
	if got != nil {
		t.Error("获取不存在的 taskID 应该返回 nil")
	}
}

// TestTaskRegistry_UnregisterTodoList 测试注销 TodoList
func TestTaskRegistry_UnregisterTodoList(t *testing.T) {
	registry := setupTestRegistry(t)

	tl := createTestTodoList("task-1")
	registry.RegisterTodoList("task-1", tl)

	// 验证已注册
	if registry.GetTodoList("task-1") == nil {
		t.Fatal("TodoList 应该已注册")
	}

	// 注销
	registry.UnregisterTodoList("task-1")

	// 验证已注销
	if registry.GetTodoList("task-1") != nil {
		t.Error("注销后应该无法获取到 TodoList")
	}
}

// TestTaskRegistry_UnregisterTodoList_NotFound 测试注销不存在的 TodoList 不 panic
func TestTaskRegistry_UnregisterTodoList_NotFound(t *testing.T) {
	registry := setupTestRegistry(t)

	// 不应该 panic
	registry.UnregisterTodoList("non-existent-task")
}

// TestTaskRegistry_GetAllTraceContexts 测试获取所有 TraceContext
func TestTaskRegistry_GetAllTraceContexts(t *testing.T) {
	registry := setupTestRegistry(t)

	// 注册多个 TraceContext
	tc1 := createTestTraceContext("trace-1", "task-1")
	tc2 := createTestTraceContext("trace-2", "task-2")
	registry.RegisterTraceContext(tc1)
	registry.RegisterTraceContext(tc2)

	// 获取所有
	all := registry.GetAllTraceContexts()

	if len(all) != 2 {
		t.Errorf("期望获取 2 个 TraceContext, 实际为 %d", len(all))
	}

	if _, ok := all["trace-1"]; !ok {
		t.Error("应该包含 trace-1")
	}
	if _, ok := all["trace-2"]; !ok {
		t.Error("应该包含 trace-2")
	}
}

// TestTaskRegistry_GetAllTraceContexts_Copy 测试返回的是副本而非引用
func TestTaskRegistry_GetAllTraceContexts_Copy(t *testing.T) {
	registry := setupTestRegistry(t)

	tc := createTestTraceContext("trace-1", "task-1")
	registry.RegisterTraceContext(tc)

	// 获取所有
	all1 := registry.GetAllTraceContexts()

	// 验证获取到了数据
	if len(all1) != 1 {
		t.Fatal("应该获取到 1 个 TraceContext")
	}

	// 修改返回的 map 本身（添加/删除元素），不影响原始 map
	all1["trace-2"] = createTestTraceContext("trace-2", "task-2")
	delete(all1, "trace-1")

	// 再次获取
	all2 := registry.GetAllTraceContexts()

	// 验证原始数据未被修改（map 结构是独立的）
	if len(all2) != 1 {
		t.Errorf("修改返回的 map 不应影响原始数据，期望 1 个，实际 %d 个", len(all2))
	}
	if _, ok := all2["trace-1"]; !ok {
		t.Error("原始数据中的 trace-1 应该仍然存在")
	}
	if _, ok := all2["trace-2"]; ok {
		t.Error("在副本中添加的 trace-2 不应出现在原始数据中")
	}
}

// TestTaskRegistry_GetAllTodoLists 测试获取所有 TodoList
func TestTaskRegistry_GetAllTodoLists(t *testing.T) {
	registry := setupTestRegistry(t)

	// 注册多个 TodoList
	tl1 := createTestTodoList("task-1")
	tl2 := createTestTodoList("task-2")
	registry.RegisterTodoList("task-1", tl1)
	registry.RegisterTodoList("task-2", tl2)

	// 获取所有
	all := registry.GetAllTodoLists()

	if len(all) != 2 {
		t.Errorf("期望获取 2 个 TodoList, 实际为 %d", len(all))
	}

	if _, ok := all["task-1"]; !ok {
		t.Error("应该包含 task-1")
	}
	if _, ok := all["task-2"]; !ok {
		t.Error("应该包含 task-2")
	}
}

// TestTaskRegistry_GetAllTodoLists_Copy 测试返回的是副本而非引用
func TestTaskRegistry_GetAllTodoLists_Copy(t *testing.T) {
	registry := setupTestRegistry(t)

	tl := createTestTodoList("task-1")
	registry.RegisterTodoList("task-1", tl)

	// 获取所有
	all1 := registry.GetAllTodoLists()

	// 修改返回的 map
	all1["task-1"] = nil

	// 再次获取
	all2 := registry.GetAllTodoLists()

	// 验证原始数据未被修改
	if all2["task-1"] == nil {
		t.Error("GetAllTodoLists 应该返回副本，修改副本不应影响原始数据")
	}
}

// TestTaskRegistry_ConcurrentOperations 测试并发操作的安全性
func TestTaskRegistry_ConcurrentOperations(t *testing.T) {
	registry := setupTestRegistry(t)

	const numGoroutines = 50
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup

	// 并发注册
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				taskID := "task-" + string(rune('0'+index)) + "-" + string(rune('0'+j%10))

				// 注册 TraceContext
				tc := createTestTraceContext("trace-"+taskID, taskID)
				registry.RegisterTraceContext(tc)

				// 注册 TaskContext
				ctx := createTestTaskContext(taskID, domain.TaskTypeAgent, "goal")
				registry.RegisterTaskContext(taskID, ctx)

				// 注册 TodoList
				tl := createTestTodoList(taskID)
				registry.RegisterTodoList(taskID, tl)
			}
		}(i)
	}

	wg.Wait()

	// 验证所有注册的数据都可以被正确获取
	allTraces := registry.GetAllTraceContexts()
	allTodos := registry.GetAllTodoLists()

	// 数据量可能因并发覆盖而不确定，但至少应该有一些数据
	if len(allTraces) == 0 {
		t.Error("并发注册后应该有 TraceContext 数据")
	}
	if len(allTodos) == 0 {
		t.Error("并发注册后应该有 TodoList 数据")
	}

	// 验证没有数据损坏（通过遍历所有数据）
	for traceID, tc := range allTraces {
		if tc == nil {
			t.Errorf("traceID %s 对应的 TraceContext 为 nil", traceID)
		}
		if tc.TraceID != traceID {
			t.Errorf("traceID %s 对应的 TraceContext.TraceID 不匹配", traceID)
		}
	}

	for taskID, tl := range allTodos {
		if tl == nil {
			t.Errorf("taskID %s 对应的 TodoList 为 nil", taskID)
		}
		if tl.TaskID != taskID {
			t.Errorf("taskID %s 对应的 TodoList.TaskID 不匹配", taskID)
		}
	}
}

// TestTaskRegistry_ConcurrentReadWrite 测试并发读写
func TestTaskRegistry_ConcurrentReadWrite(t *testing.T) {
	registry := setupTestRegistry(t)

	const numGoroutines = 30
	const duration = 100 * time.Millisecond

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// 先注册一些初始数据
	for i := 0; i < 10; i++ {
		tc := createTestTraceContext("trace-"+string(rune('0'+i)), "task-"+string(rune('0'+i)))
		registry.RegisterTraceContext(tc)
	}

	// 启动读写 goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			counter := 0
			for {
				select {
				case <-stopCh:
					return
				default:
					counter++
					taskID := "task-" + string(rune('0'+index)) + "-" + string(rune('0'+counter%10))

					switch counter % 6 {
					case 0:
						// 写 TraceContext
						tc := createTestTraceContext("trace-"+taskID, taskID)
						registry.RegisterTraceContext(tc)
					case 1:
						// 读 TraceContext
						_ = registry.GetTraceContext("trace-" + taskID)
					case 2:
						// 注销 TraceContext
						registry.UnregisterTraceContext("trace-" + taskID)
					case 3:
						// 批量读取
						_ = registry.GetAllTraceContexts()
					case 4:
						// 通过 TaskID 查找
						_ = registry.GetTraceContextByTaskID(taskID)
					case 5:
						// 读写 TodoList
						tl := createTestTodoList(taskID)
						registry.RegisterTodoList(taskID, tl)
						_ = registry.GetTodoList(taskID)
						registry.UnregisterTodoList(taskID)
					}
				}
			}
		}(i)
	}

	// 运行一段时间
	time.Sleep(duration)
	close(stopCh)
	wg.Wait()

	// 如果没有 panic 或死锁，测试通过
	t.Log("并发读写测试完成，未发现竞态条件")
}

// TestTaskRegistry_EmptyRegistry 测试空注册表的行为
func TestTaskRegistry_EmptyRegistry(t *testing.T) {
	registry := setupTestRegistry(t)

	// 确保注册表为空
	allTraces := registry.GetAllTraceContexts()
	for id := range allTraces {
		registry.UnregisterTraceContext(id)
	}
	allTodos := registry.GetAllTodoLists()
	for id := range allTodos {
		registry.UnregisterTodoList(id)
	}

	// 验证空注册表的行为
	if len(registry.GetAllTraceContexts()) != 0 {
		t.Error("空注册表应该返回空的 TraceContext map")
	}
	if len(registry.GetAllTodoLists()) != 0 {
		t.Error("空注册表应该返回空的 TodoList map")
	}
	if registry.GetTraceContext("any") != nil {
		t.Error("空注册表获取不存在的 trace 应该返回 nil")
	}
	if registry.GetTaskContext("any") != nil {
		t.Error("空注册表获取不存在的 task 应该返回 nil")
	}
	if registry.GetTodoList("any") != nil {
		t.Error("空注册表获取不存在的 todolist 应该返回 nil")
	}
	if registry.GetTraceContextByTaskID("any") != nil {
		t.Error("空注册表通过 taskID 查找应该返回 nil")
	}
}

// TestTaskRegistry_MixedOperations 测试混合操作场景
func TestTaskRegistry_MixedOperations(t *testing.T) {
	registry := setupTestRegistry(t)

	// 场景：完整的任务生命周期
	traceID := "trace-lifecycle"
	rootTaskID := "task-root"
	childTaskID := "task-child"

	// 1. 注册 TraceContext
	tc := createTestTraceContext(traceID, rootTaskID)
	registry.RegisterTraceContext(tc)

	// 2. 注册 Root TaskContext
	rootCtx := createTestTaskContext(rootTaskID, domain.TaskTypeAgent, "根任务")
	registry.RegisterTaskContext(rootTaskID, rootCtx)

	// 3. 注册 Root TodoList
	rootTodo := createTestTodoList(rootTaskID)
	registry.RegisterTodoList(rootTaskID, rootTodo)

	// 4. 注册 Child TaskContext
	childCtx := createTestTaskContext(childTaskID, domain.TaskTypeCoding, "子任务")
	registry.RegisterTaskContext(childTaskID, childCtx)

	// 5. 验证通过 taskID 查找 TraceContext
	foundTc := registry.GetTraceContextByTaskID(rootTaskID)
	if foundTc == nil || foundTc.TraceID != traceID {
		t.Error("应该能通过 rootTaskID 找到 TraceContext")
	}

	// 6. 验证所有数据都可以获取
	if registry.GetTraceContext(traceID) == nil {
		t.Error("应该能获取 TraceContext")
	}
	if registry.GetTaskContext(rootTaskID) == nil {
		t.Error("应该能获取 Root TaskContext")
	}
	if registry.GetTaskContext(childTaskID) == nil {
		t.Error("应该能获取 Child TaskContext")
	}
	if registry.GetTodoList(rootTaskID) == nil {
		t.Error("应该能获取 Root TodoList")
	}

	// 7. 模拟任务完成，注销相关数据
	registry.UnregisterTaskContext(childTaskID)
	registry.UnregisterTaskContext(rootTaskID)
	registry.UnregisterTodoList(rootTaskID)
	registry.UnregisterTraceContext(traceID)

	// 8. 验证所有数据已被清理
	if registry.GetTraceContext(traceID) != nil {
		t.Error("TraceContext 应该已被注销")
	}
	if registry.GetTaskContext(rootTaskID) != nil {
		t.Error("Root TaskContext 应该已被注销")
	}
	if registry.GetTaskContext(childTaskID) != nil {
		t.Error("Child TaskContext 应该已被注销")
	}
	if registry.GetTodoList(rootTaskID) != nil {
		t.Error("Root TodoList 应该已被注销")
	}
}
