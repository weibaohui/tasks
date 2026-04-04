package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestNewTaskRuntime 测试构造函数
func TestNewTaskRuntime(t *testing.T) {
	rt := NewTaskRuntime()
	if rt == nil {
		t.Fatal("NewTaskRuntime() returned nil")
	}
	if rt.running == nil {
		t.Error("running map not initialized")
	}
	if rt.contexts == nil {
		t.Error("contexts map not initialized")
	}
}

// TestRegister 测试任务注册
func TestRegister(t *testing.T) {
	rt := NewTaskRuntime()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rt.Register("task-1", ctx, cancel)

	// 验证是否成功注册
	gotCtx, ok := rt.GetContext("task-1")
	if !ok {
		t.Error("Register failed: task not found")
	}
	if gotCtx != ctx {
		t.Error("Register failed: context mismatch")
	}
}

// TestRegisterOverride 测试重复注册同一 taskID 应覆盖旧值
func TestRegisterOverride(t *testing.T) {
	rt := NewTaskRuntime()

	// 第一次注册
	ctx1, cancel1 := context.WithCancel(context.Background())
	rt.Register("task-1", ctx1, cancel1)

	// 第二次注册同一个 taskID
	ctx2, cancel2 := context.WithCancel(context.Background())
	rt.Register("task-1", ctx2, cancel2)

	// 验证是否被覆盖
	gotCtx, ok := rt.GetContext("task-1")
	if !ok {
		t.Fatal("Register override failed: task not found")
	}
	if gotCtx != ctx2 {
		t.Error("Register override failed: context not updated")
	}

	// 清理
	cancel1()
	cancel2()
}

// TestGetContext_NotFound 测试获取未注册任务的 context
func TestGetContext_NotFound(t *testing.T) {
	rt := NewTaskRuntime()

	_, ok := rt.GetContext("non-existent-task")
	if ok {
		t.Error("GetContext should return false for non-existent task")
	}
}

// TestCancel 测试取消任务
func TestCancel(t *testing.T) {
	rt := NewTaskRuntime()
	ctx, cancel := context.WithCancel(context.Background())
	rt.Register("task-1", ctx, cancel)

	// 取消任务
	ok := rt.Cancel("task-1")
	if !ok {
		t.Error("Cancel should return true for existing task")
	}

	// 验证 context.Done() 被关闭
	select {
	case <-ctx.Done():
		// 预期行为
	case <-time.After(time.Second):
		t.Error("context.Done() should be closed after Cancel")
	}

	// 验证任务已被清理
	_, found := rt.GetContext("task-1")
	if found {
		t.Error("Cancel should remove task from contexts")
	}
}

// TestCancel_NotFound 测试取消不存在的任务
func TestCancel_NotFound(t *testing.T) {
	rt := NewTaskRuntime()

	ok := rt.Cancel("non-existent-task")
	if ok {
		t.Error("Cancel should return false for non-existent task")
	}
}

// TestCancel_DoubleCancel 测试重复取消任务
func TestCancel_DoubleCancel(t *testing.T) {
	rt := NewTaskRuntime()
	ctx, cancel := context.WithCancel(context.Background())
	rt.Register("task-1", ctx, cancel)

	// 第一次取消
	ok1 := rt.Cancel("task-1")
	if !ok1 {
		t.Error("First Cancel should return true")
	}

	// 第二次取消（任务已不存在）
	ok2 := rt.Cancel("task-1")
	if ok2 {
		t.Error("Second Cancel should return false")
	}
}

// TestUnregister 测试注销任务
func TestUnregister(t *testing.T) {
	rt := NewTaskRuntime()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rt.Register("task-1", ctx, cancel)

	rt.Unregister("task-1")

	// 验证任务已被清理
	_, found := rt.GetContext("task-1")
	if found {
		t.Error("Unregister should remove task")
	}
}

// TestUnregister_NotFound 测试注销不存在的任务
func TestUnregister_NotFound(t *testing.T) {
	rt := NewTaskRuntime()

	// 不应 panic
	rt.Unregister("non-existent-task")
}

// TestCreateContext_WithTimeout 测试带超时的 context 创建
func TestCreateContext_WithTimeout(t *testing.T) {
	rt := NewTaskRuntime()
	ctx, cancel := rt.CreateContext("task-1", 100*time.Millisecond)
	defer cancel()

	// 验证是否自动注册
	_, ok := rt.GetContext("task-1")
	if !ok {
		t.Error("CreateContext should auto-register task")
	}

	// 验证超时后 context.Done() 被关闭
	select {
	case <-ctx.Done():
		// 预期行为
	case <-time.After(time.Second):
		t.Error("context.Done() should be closed after timeout")
	}
}

// TestCreateContext_NoTimeout 测试无超时的 context 创建
func TestCreateContext_NoTimeout(t *testing.T) {
	rt := NewTaskRuntime()
	ctx, cancel := rt.CreateContext("task-1", 0)
	defer cancel()

	// 验证是否自动注册
	_, ok := rt.GetContext("task-1")
	if !ok {
		t.Error("CreateContext should auto-register task")
	}

	// 验证 context 未被立即取消
	select {
	case <-ctx.Done():
		t.Error("context should not be done immediately without timeout")
	case <-time.After(50 * time.Millisecond):
		// 预期行为
	}

	// 手动取消
	cancel()

	// 验证取消后 context.Done() 被关闭
	select {
	case <-ctx.Done():
		// 预期行为
	case <-time.After(time.Second):
		t.Error("context.Done() should be closed after cancel")
	}
}

// TestCreateContext_Override 测试重复创建同一 taskID 的 context
func TestCreateContext_Override(t *testing.T) {
	rt := NewTaskRuntime()

	// 第一次创建
	ctx1, cancel1 := rt.CreateContext("task-1", time.Hour)
	defer cancel1()

	// 第二次创建（应覆盖）
	ctx2, cancel2 := rt.CreateContext("task-1", time.Hour)
	defer cancel2()

	// 验证是否被覆盖
	gotCtx, _ := rt.GetContext("task-1")
	if gotCtx != ctx2 {
		t.Error("CreateContext should override existing task")
	}

	// 验证旧 context 没有被取消（只是从管理器中移除）
	select {
	case <-ctx1.Done():
		t.Error("Old context should not be cancelled when overridden")
	case <-time.After(50 * time.Millisecond):
		// 预期行为
	}
}

// TestEmptyTaskID 测试空 taskID 的处理
func TestEmptyTaskID(t *testing.T) {
	rt := NewTaskRuntime()

	// 注册空 taskID
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rt.Register("", ctx, cancel)

	// 应该能够获取
	gotCtx, ok := rt.GetContext("")
	if !ok {
		t.Error("Empty taskID should be valid")
	}
	if gotCtx != ctx {
		t.Error("Context mismatch for empty taskID")
	}

	// 能够取消
	ok = rt.Cancel("")
	if !ok {
		t.Error("Cancel should work for empty taskID")
	}
}

// TestConcurrentRegisterCancel 测试并发注册和取消
func TestConcurrentRegisterCancel(t *testing.T) {
	rt := NewTaskRuntime()
	const numTasks = 100

	var wg sync.WaitGroup

	// 并发注册
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := fmt.Sprintf("task-%d", id)
			ctx, cancel := context.WithCancel(context.Background())
			rt.Register(taskID, ctx, cancel)
		}(i)
	}
	wg.Wait()

	// 验证所有任务都已注册
	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		_, ok := rt.GetContext(taskID)
		if !ok {
			t.Errorf("Task %s not found after concurrent register", taskID)
		}
	}

	// 并发取消
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := fmt.Sprintf("task-%d", id)
			rt.Cancel(taskID)
		}(i)
	}
	wg.Wait()

	// 验证所有任务都已取消
	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		_, ok := rt.GetContext(taskID)
		if ok {
			t.Errorf("Task %s should be removed after concurrent cancel", taskID)
		}
	}
}

// TestConcurrentMixedOperations 测试并发混合操作
func TestConcurrentMixedOperations(t *testing.T) {
	rt := NewTaskRuntime()
	const numWorkers = 50
	const operationsPerWorker = 20

	var wg sync.WaitGroup

	// 多个 goroutine 执行混合操作
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerWorker; j++ {
				taskID := fmt.Sprintf("task-%d-%d", workerID, j%5)

				switch j % 4 {
				case 0: // Register
					ctx, cancel := context.WithCancel(context.Background())
					rt.Register(taskID, ctx, cancel)
				case 1: // GetContext
					rt.GetContext(taskID)
				case 2: // Cancel
					rt.Cancel(taskID)
				case 3: // Unregister
					rt.Unregister(taskID)
				}
			}
		}(i)
	}
	wg.Wait()
}

// TestConcurrentCreateContext 测试并发创建多个任务的 context
func TestConcurrentCreateContext(t *testing.T) {
	rt := NewTaskRuntime()
	const numTasks = 100

	var wg sync.WaitGroup

	// 并发创建 context
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := fmt.Sprintf("task-%d", id)
			ctx, cancel := rt.CreateContext(taskID, time.Hour)
			defer cancel()

			// 验证 context 可用
			select {
			case <-ctx.Done():
				t.Errorf("Context for task %s should not be done immediately", taskID)
			default:
				// 预期行为
			}
		}(i)
	}
	wg.Wait()

	// 验证所有任务都已注册
	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		_, ok := rt.GetContext(taskID)
		if !ok {
			t.Errorf("Task %s not found after concurrent CreateContext", taskID)
		}
	}
}

// TestLargeScaleRegisterCleanup 测试大量任务注册和清理的性能
func TestLargeScaleRegisterCleanup(t *testing.T) {
	rt := NewTaskRuntime()
	const numTasks = 1000

	// 注册大量任务
	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		ctx, cancel := context.WithCancel(context.Background())
		rt.Register(taskID, ctx, cancel)
	}

	// 验证所有任务都已注册
	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		_, ok := rt.GetContext(taskID)
		if !ok {
			t.Errorf("Task %s not found after large scale register", taskID)
		}
	}

	// 清理所有任务
	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		rt.Cancel(taskID)
	}

	// 验证所有任务都已清理
	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		_, ok := rt.GetContext(taskID)
		if ok {
			t.Errorf("Task %s should be removed after Cancel", taskID)
		}
	}
}

// TestUnregisterAndCancelIndependence 测试 Unregister 和 Cancel 的独立性
func TestUnregisterAndCancelIndependence(t *testing.T) {
	rt := NewTaskRuntime()
	ctx, cancel := context.WithCancel(context.Background())
	rt.Register("task-1", ctx, cancel)

	// Unregister 任务
	rt.Unregister("task-1")

	// 验证任务已从管理器移除
	_, ok := rt.GetContext("task-1")
	if ok {
		t.Error("Task should be removed after Unregister")
	}

	// 但 context 本身没有被取消
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled by Unregister")
	case <-time.After(50 * time.Millisecond):
		// 预期行为
	}

	// 手动取消验证 cancel 函数仍然有效
	cancel()
	select {
	case <-ctx.Done():
		// 预期行为
	case <-time.After(time.Second):
		t.Error("Context should be cancelled after manual cancel")
	}
}
