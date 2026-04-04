/**
 * TaskContext 单元测试
 * 测试 TaskContext 的核心功能和并发安全性
 */
package application

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

// TestNewTaskContext 测试构造函数正确初始化所有字段
func TestNewTaskContext(t *testing.T) {
	tests := []struct {
		name         string
		taskID       string
		taskType     domain.TaskType
		goal         string
		spanID       string
		parentSpanID string
		traceContext *TraceContext
	}{
		{
			name:         "基本初始化",
			taskID:       "task-1",
			taskType:     domain.TaskTypeAgent,
			goal:         "测试目标",
			spanID:       "span-1",
			parentSpanID: "",
			traceContext: nil,
		},
		{
			name:         "带父Span的初始化",
			taskID:       "task-2",
			taskType:     domain.TaskTypeCoding,
			goal:         "编码任务",
			spanID:       "span-2",
			parentSpanID: "span-1",
			traceContext: NewTraceContext("trace-1", "task-root", nil),
		},
		{
			name:         "Custom类型任务",
			taskID:       "task-3",
			taskType:     domain.TaskTypeCustom,
			goal:         "自定义任务",
			spanID:       "span-3",
			parentSpanID: "span-2",
			traceContext: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTaskContext(tt.taskID, tt.taskType, tt.goal, tt.spanID, tt.parentSpanID, tt.traceContext)

			// 验证基本字段
			if tc.TaskID != tt.taskID {
				t.Errorf("期望 TaskID 为 '%s', 实际为 '%s'", tt.taskID, tc.TaskID)
			}
			if tc.TaskType != tt.taskType {
				t.Errorf("期望 TaskType 为 %v, 实际为 %v", tt.taskType, tc.TaskType)
			}
			if tc.Goal != tt.goal {
				t.Errorf("期望 Goal 为 '%s', 实际为 '%s'", tt.goal, tc.Goal)
			}
			if tc.SpanID != tt.spanID {
				t.Errorf("期望 SpanID 为 '%s', 实际为 '%s'", tt.spanID, tc.SpanID)
			}
			if tc.ParentSpanID != tt.parentSpanID {
				t.Errorf("期望 ParentSpanID 为 '%s', 实际为 '%s'", tt.parentSpanID, tc.ParentSpanID)
			}
			if tc.TraceContext != tt.traceContext {
				t.Errorf("期望 TraceContext 为 %v, 实际为 %v", tt.traceContext, tc.TraceContext)
			}

			// 验证 TodoList 自动创建
			if tc.TodoList == nil {
				t.Error("TodoList 应该自动创建，不应为 nil")
			} else if tc.TodoList.TaskID != tt.taskID {
				t.Errorf("TodoList.TaskID 应为 '%s', 实际为 '%s'", tt.taskID, tc.TodoList.TaskID)
			}

			// 验证回调函数初始为 nil
			if tc.OnProgress != nil {
				t.Error("OnProgress 初始应为 nil")
			}
			if tc.OnComplete != nil {
				t.Error("OnComplete 初始应为 nil")
			}
			if tc.OnSubTask != nil {
				t.Error("OnSubTask 初始应为 nil")
			}
			if tc.CancelFunc != nil {
				t.Error("CancelFunc 初始应为 nil")
			}
		})
	}
}

// TestTaskContext_SetOnProgress_ReportProgress 测试进度回调设置和报告
func TestTaskContext_SetOnProgress_ReportProgress(t *testing.T) {
	tests := []struct {
		name           string
		progress       int
		expectedCalled bool
	}{
		{"进度0", 0, true},
		{"进度50", 50, true},
		{"进度100", 100, true},
		{"负进度", -10, true},
		{"超范围进度", 150, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

			var receivedProgress int
			var called bool

			// 设置回调
			tc.SetOnProgress(func(progress int) {
				receivedProgress = progress
				called = true
			})

			// 报告进度
			tc.ReportProgress(tt.progress)

			if !called {
				t.Error("进度回调应该被调用")
			}
			if receivedProgress != tt.progress {
				t.Errorf("回调应接收到进度 %d, 实际为 %d", tt.progress, receivedProgress)
			}
		})
	}
}

// TestTaskContext_ReportProgress_NilCallback 测试进度回调为 nil 时安全调用
func TestTaskContext_ReportProgress_NilCallback(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	// 不设置回调，直接调用 ReportProgress
	// 不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ReportProgress 在回调为 nil 时不应该 panic, 实际 panic: %v", r)
		}
	}()

	tc.ReportProgress(50)
}

// TestTaskContext_SetOnComplete_Complete 测试完成回调设置和调用
func TestTaskContext_SetOnComplete_Complete(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	var called bool

	// 设置完成回调
	tc.SetOnComplete(func() {
		called = true
	})

	// 调用 Complete
	tc.Complete()

	if !called {
		t.Error("完成回调应该被调用")
	}
}

// TestTaskContext_Complete_NilCallback 测试完成回调为 nil 时安全调用
func TestTaskContext_Complete_NilCallback(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	// 不设置回调，直接调用 Complete
	// 不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Complete 在回调为 nil 时不应该 panic, 实际 panic: %v", r)
		}
	}()

	tc.Complete()
}

// TestTaskContext_SetOnSubTask_SpawnSubTask 测试子任务回调设置和调用
func TestTaskContext_SetOnSubTask_SpawnSubTask(t *testing.T) {
	tests := []struct {
		name         string
		goal         string
		taskType     domain.TaskType
		returnResult *SubTaskResult
	}{
		{
			name:         "生成子任务",
			goal:         "子任务目标",
			taskType:     domain.TaskTypeCoding,
			returnResult: &SubTaskResult{SubTaskID: "sub-1", Goal: "子任务目标", TaskType: domain.TaskTypeCoding},
		},
		{
			name:         "返回 nil",
			goal:         "测试目标",
			taskType:     domain.TaskTypeAgent,
			returnResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

			var receivedGoal string
			var receivedTaskType domain.TaskType
			var called bool

			// 设置子任务回调
			tc.SetOnSubTask(func(goal string, taskType domain.TaskType) *SubTaskResult {
				receivedGoal = goal
				receivedTaskType = taskType
				called = true
				return tt.returnResult
			})

			// 生成子任务
			result := tc.SpawnSubTask(tt.goal, tt.taskType)

			if !called {
				t.Error("子任务回调应该被调用")
			}
			if receivedGoal != tt.goal {
				t.Errorf("回调应接收到 goal '%s', 实际为 '%s'", tt.goal, receivedGoal)
			}
			if receivedTaskType != tt.taskType {
				t.Errorf("回调应接收到 taskType %v, 实际为 %v", tt.taskType, receivedTaskType)
			}
			if tt.returnResult != nil && result != tt.returnResult {
				t.Error("SpawnSubTask 应返回回调返回的结果")
			}
			if tt.returnResult == nil && result != nil {
				t.Error("当回调返回 nil 时，SpawnSubTask 也应返回 nil")
			}
		})
	}
}

// TestTaskContext_SpawnSubTask_NilCallback 测试子任务回调为 nil 时安全调用
func TestTaskContext_SpawnSubTask_NilCallback(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	// 不设置回调，直接调用 SpawnSubTask
	// 不应该 panic，应返回 nil
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SpawnSubTask 在回调为 nil 时不应该 panic, 实际 panic: %v", r)
		}
	}()

	result := tc.SpawnSubTask("目标", domain.TaskTypeCoding)
	if result != nil {
		t.Error("回调为 nil 时 SpawnSubTask 应返回 nil")
	}
}

// TestTaskContext_SetCancelFunc_Cancel 测试取消机制
func TestTaskContext_SetCancelFunc_Cancel(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	var called bool
	cancelFunc := func() {
		called = true
	}

	// 设置取消函数
	tc.SetCancelFunc(cancelFunc)

	// 调用 Cancel
	tc.Cancel()

	if !called {
		t.Error("取消函数应该被调用")
	}
}

// TestTaskContext_Cancel_NilFunc 测试取消函数为 nil 时安全调用
func TestTaskContext_Cancel_NilFunc(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	// 不设置取消函数，直接调用 Cancel
	// 不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Cancel 在 CancelFunc 为 nil 时不应该 panic, 实际 panic: %v", r)
		}
	}()

	tc.Cancel()
}

// TestTaskContext_Cancel_WithContext 测试使用真实 context.CancelFunc
func TestTaskContext_Cancel_WithContext(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	ctx, cancel := context.WithCancel(context.Background())
	tc.SetCancelFunc(cancel)

	// 验证 context 未取消
	select {
	case <-ctx.Done():
		t.Error("context 不应已被取消")
	default:
		// 正常
	}

	// 调用 Cancel
	tc.Cancel()

	// 验证 context 已取消
	select {
	case <-ctx.Done():
		// 正常
	default:
		t.Error("context 应该已被取消")
	}
}

// TestTaskContext_TodoList_AddTodoItem 测试添加待办项
func TestTaskContext_TodoList_AddTodoItem(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	// 添加待办项
	tc.AddTodoItem("sub-1", "子任务1", "thinking", "span-sub-1")
	tc.AddTodoItem("sub-2", "子任务2", "coding", "span-sub-2")

	// 验证 TodoList 中存在这些项
	items := tc.TodoList.GetAllItems()
	if len(items) != 2 {
		t.Errorf("期望有 2 个待办项, 实际有 %d 个", len(items))
	}

	// 验证第一个待办项
	item1 := tc.TodoList.GetItem("sub-1")
	if item1 == nil {
		t.Fatal("应该能找到 sub-1 待办项")
	}
	if item1.Goal != "子任务1" {
		t.Errorf("待办项 goal 应为 '子任务1', 实际为 '%s'", item1.Goal)
	}
	if item1.SubTaskType != "thinking" {
		t.Errorf("待办项 type 应为 'thinking', 实际为 '%s'", item1.SubTaskType)
	}
	if item1.Status != TodoStatusDistributed {
		t.Errorf("待办项状态应为 'distributed', 实际为 '%s'", item1.Status)
	}
}

// TestTaskContext_TodoList_UpdateTodoProgress 测试更新待办项进度
func TestTaskContext_TodoList_UpdateTodoProgress(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)
	tc.AddTodoItem("sub-1", "子任务1", "thinking", "span-sub-1")

	// 更新进度
	tc.UpdateTodoProgress("sub-1", 50)

	// 验证进度
	item := tc.TodoList.GetItem("sub-1")
	if item == nil {
		t.Fatal("应该能找到 sub-1 待办项")
	}
	if item.Progress != 50 {
		t.Errorf("待办项进度应为 50, 实际为 %d", item.Progress)
	}
	if item.Status != TodoStatusRunning {
		t.Errorf("进度更新后状态应为 'running', 实际为 '%s'", item.Status)
	}

	// 更新到 100
	tc.UpdateTodoProgress("sub-1", 100)
	item = tc.TodoList.GetItem("sub-1")
	if item.Progress != 100 {
		t.Errorf("待办项进度应为 100, 实际为 %d", item.Progress)
	}
}

// TestTaskContext_TodoList_UpdateTodoCompleted 测试标记待办项完成
func TestTaskContext_TodoList_UpdateTodoCompleted(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)
	tc.AddTodoItem("sub-1", "子任务1", "thinking", "span-sub-1")

	// 标记完成
	tc.UpdateTodoCompleted("sub-1")

	// 验证状态
	item := tc.TodoList.GetItem("sub-1")
	if item == nil {
		t.Fatal("应该能找到 sub-1 待办项")
	}
	if item.Status != TodoStatusCompleted {
		t.Errorf("待办项状态应为 'completed', 实际为 '%s'", item.Status)
	}
	if item.Progress != 100 {
		t.Errorf("待办项进度应为 100, 实际为 %d", item.Progress)
	}
	if item.CompletedAt == nil {
		t.Error("待办项 CompletedAt 不应为 nil")
	}
}

// TestTaskContext_TodoList_UpdateTodoFailed 测试标记待办项失败
func TestTaskContext_TodoList_UpdateTodoFailed(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)
	tc.AddTodoItem("sub-1", "子任务1", "thinking", "span-sub-1")

	// 标记失败
	tc.UpdateTodoFailed("sub-1")

	// 验证状态
	item := tc.TodoList.GetItem("sub-1")
	if item == nil {
		t.Fatal("应该能找到 sub-1 待办项")
	}
	if item.Status != TodoStatusFailed {
		t.Errorf("待办项状态应为 'failed', 实际为 '%s'", item.Status)
	}
	if item.CompletedAt == nil {
		t.Error("待办项 CompletedAt 不应为 nil")
	}
}

// TestTaskContext_AllSubTasksCompleted 测试全部子任务完成检查
func TestTaskContext_AllSubTasksCompleted(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(tc *TaskContext)
		expectedResult bool
	}{
		{
			name: "空列表",
			setupFunc: func(tc *TaskContext) {
				// 不添加任何待办项
			},
			expectedResult: false,
		},
		{
			name: "单个待办项已完成",
			setupFunc: func(tc *TaskContext) {
				tc.AddTodoItem("sub-1", "子任务1", "thinking", "span-sub-1")
				tc.UpdateTodoCompleted("sub-1")
			},
			expectedResult: true,
		},
		{
			name: "单个待办项未完成",
			setupFunc: func(tc *TaskContext) {
				tc.AddTodoItem("sub-1", "子任务1", "thinking", "span-sub-1")
			},
			expectedResult: false,
		},
		{
			name: "多个待办项全部完成",
			setupFunc: func(tc *TaskContext) {
				tc.AddTodoItem("sub-1", "子任务1", "thinking", "span-sub-1")
				tc.AddTodoItem("sub-2", "子任务2", "coding", "span-sub-2")
				tc.UpdateTodoCompleted("sub-1")
				tc.UpdateTodoCompleted("sub-2")
			},
			expectedResult: true,
		},
		{
			name: "多个待办项部分完成",
			setupFunc: func(tc *TaskContext) {
				tc.AddTodoItem("sub-1", "子任务1", "thinking", "span-sub-1")
				tc.AddTodoItem("sub-2", "子任务2", "coding", "span-sub-2")
				tc.UpdateTodoCompleted("sub-1")
				// sub-2 未完成
			},
			expectedResult: false,
		},
		{
			name: "包含失败任务",
			setupFunc: func(tc *TaskContext) {
				tc.AddTodoItem("sub-1", "子任务1", "thinking", "span-sub-1")
				tc.AddTodoItem("sub-2", "子任务2", "coding", "span-sub-2")
				tc.UpdateTodoCompleted("sub-1")
				tc.UpdateTodoFailed("sub-2")
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)
			tt.setupFunc(tc)

			result := tc.AllSubTasksCompleted()
			if result != tt.expectedResult {
				t.Errorf("AllSubTasksCompleted() 应返回 %v, 实际为 %v", tt.expectedResult, result)
			}
		})
	}
}

// TestTaskContext_Concurrent_ReportProgress 测试并发报告进度
func TestTaskContext_Concurrent_ReportProgress(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	var counter int32
	tc.SetOnProgress(func(progress int) {
		atomic.AddInt32(&counter, 1)
	})

	const numGoroutines = 100
	const numCallsPerGoroutine = 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < numCallsPerGoroutine; j++ {
				tc.ReportProgress(j)
			}
		}(i)
	}
	wg.Wait()

	expected := int32(numGoroutines * numCallsPerGoroutine)
	if counter != expected {
		t.Errorf("期望回调被调用 %d 次, 实际为 %d 次", expected, counter)
	}
}

// TestTaskContext_Concurrent_Complete 测试并发调用 Complete
func TestTaskContext_Concurrent_Complete(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	var counter int32
	tc.SetOnComplete(func() {
		atomic.AddInt32(&counter, 1)
	})

	const numGoroutines = 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tc.Complete()
		}()
	}
	wg.Wait()

	// Complete 不使用锁保护回调调用，所以可能被调用多次
	// 这里主要验证没有 panic 和 data race
	t.Logf("Complete 回调被调用了 %d 次", counter)
	if counter == 0 {
		t.Error("Complete 回调应该至少被调用一次")
	}
}

// TestTaskContext_Concurrent_SpawnSubTask 测试并发生成子任务
func TestTaskContext_Concurrent_SpawnSubTask(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	var counter int32
	tc.SetOnSubTask(func(goal string, taskType domain.TaskType) *SubTaskResult {
		atomic.AddInt32(&counter, 1)
		return &SubTaskResult{SubTaskID: "sub-task", Goal: goal, TaskType: taskType}
	})

	const numGoroutines = 100
	const numCallsPerGoroutine = 50

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < numCallsPerGoroutine; j++ {
				tc.SpawnSubTask("目标", domain.TaskTypeCoding)
			}
		}(i)
	}
	wg.Wait()

	expected := int32(numGoroutines * numCallsPerGoroutine)
	if counter != expected {
		t.Errorf("期望回调被调用 %d 次, 实际为 %d 次", expected, counter)
	}
}

// TestTaskContext_Concurrent_MixedOperations 测试并发混合操作
func TestTaskContext_Concurrent_MixedOperations(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	// 设置回调
	tc.SetOnProgress(func(progress int) {})
	tc.SetOnComplete(func() {})
	tc.SetOnSubTask(func(goal string, taskType domain.TaskType) *SubTaskResult {
		return nil
	})

	// 添加一些初始待办项
	for i := 0; i < 10; i++ {
		tc.AddTodoItem(strconv.Itoa(i), "子任务", "thinking", "span")
	}

	const numGoroutines = 50
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				switch j % 8 {
				case 0:
					tc.ReportProgress(j)
				case 1:
					tc.Complete()
				case 2:
					tc.SpawnSubTask("目标", domain.TaskTypeCoding)
				case 3:
					tc.AddTodoItem(strconv.Itoa(index)+"-"+strconv.Itoa(j%10), "子任务", "thinking", "span")
				case 4:
					tc.UpdateTodoProgress(strconv.Itoa(j%10), j%100)
				case 5:
					tc.UpdateTodoCompleted(strconv.Itoa(j%10))
				case 6:
					tc.UpdateTodoFailed(strconv.Itoa(j%10))
				case 7:
					_ = tc.AllSubTasksCompleted()
				}
			}
		}(i)
	}
	wg.Wait()

	// 如果没有 panic 或 data race，测试通过
	t.Log("并发混合操作测试完成，未发现竞态条件")
}

// TestTaskContext_FieldAccess 测试字段访问
func TestTaskContext_FieldAccess(t *testing.T) {
	traceContext := NewTraceContext("trace-1", "task-root", nil)
	tc := NewTaskContext("task-1", domain.TaskTypeCoding, "测试目标", "span-123", "parent-span-456", traceContext)

	// 验证所有字段可访问且值正确
	if tc.TaskID != "task-1" {
		t.Errorf("TaskID 应为 'task-1', 实际为 '%s'", tc.TaskID)
	}
	if tc.Goal != "测试目标" {
		t.Errorf("Goal 应为 '测试目标', 实际为 '%s'", tc.Goal)
	}
	if tc.TaskType != domain.TaskTypeCoding {
		t.Errorf("TaskType 应为 domain.TaskTypeCoding, 实际为 %v", tc.TaskType)
	}
	if tc.SpanID != "span-123" {
		t.Errorf("SpanID 应为 'span-123', 实际为 '%s'", tc.SpanID)
	}
	if tc.ParentSpanID != "parent-span-456" {
		t.Errorf("ParentSpanID 应为 'parent-span-456', 实际为 '%s'", tc.ParentSpanID)
	}
	if tc.TraceContext != traceContext {
		t.Error("TraceContext 不匹配")
	}
	if tc.TodoList == nil {
		t.Error("TodoList 不应为 nil")
	} else if tc.TodoList.TaskID != "task-1" {
		t.Errorf("TodoList.TaskID 应为 'task-1', 实际为 '%s'", tc.TodoList.TaskID)
	}
}

// TestTaskContext_MultipleCallbacks 测试多次设置回调（覆盖行为）
func TestTaskContext_MultipleCallbacks(t *testing.T) {
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "测试", "span-1", "", nil)

	var firstCalled, secondCalled bool

	// 设置第一个回调
	tc.SetOnProgress(func(progress int) {
		firstCalled = true
	})

	// 设置第二个回调（应该覆盖第一个）
	tc.SetOnProgress(func(progress int) {
		secondCalled = true
	})

	// 报告进度
	tc.ReportProgress(50)

	if firstCalled {
		t.Error("第一个回调不应该被调用（被覆盖）")
	}
	if !secondCalled {
		t.Error("第二个回调应该被调用")
	}
}

// TestTaskContext_ComplexWorkflow 测试完整工作流场景
func TestTaskContext_ComplexWorkflow(t *testing.T) {
	// 创建 TraceContext
	traceContext := NewTraceContext("trace-1", "task-root", nil)

	// 创建 TaskContext
	tc := NewTaskContext("task-1", domain.TaskTypeAgent, "完成某个复杂任务", "span-root", "", traceContext)

	// 设置回调
	progressValues := []int{}
	tc.SetOnProgress(func(progress int) {
		progressValues = append(progressValues, progress)
	})

	var completed bool
	tc.SetOnComplete(func() {
		completed = true
	})

	subTasks := []string{}
	tc.SetOnSubTask(func(goal string, taskType domain.TaskType) *SubTaskResult {
		subTasks = append(subTasks, goal)
		return &SubTaskResult{
			SubTaskID:    "sub-" + strconv.Itoa(len(subTasks)),
			SpanID:       "span-sub-" + strconv.Itoa(len(subTasks)),
			ParentSpanID: tc.SpanID,
			TaskType:     taskType,
			Goal:         goal,
		}
	})

	// 模拟工作流
	// 1. 报告进度
	tc.ReportProgress(10)
	tc.ReportProgress(20)

	// 2. 生成子任务
	result1 := tc.SpawnSubTask("分析需求", domain.TaskTypeAgent)
	if result1 == nil {
		t.Error("子任务结果不应为 nil")
	}

	// 3. 添加待办项
	tc.AddTodoItem("sub-1", "分析需求", "thinking", result1.SpanID)

	// 4. 更新子任务进度
	tc.UpdateTodoProgress("sub-1", 50)
	tc.UpdateTodoProgress("sub-1", 100)

	// 5. 标记子任务完成
	tc.UpdateTodoCompleted("sub-1")

	// 6. 检查所有子任务完成
	if !tc.AllSubTasksCompleted() {
		t.Error("所有子任务应该已完成")
	}

	// 7. 报告最终进度
	tc.ReportProgress(100)

	// 8. 完成任务
	tc.Complete()

	// 验证结果
	if len(progressValues) != 3 {
		t.Errorf("期望报告 3 次进度, 实际为 %d 次", len(progressValues))
	}
	if !completed {
		t.Error("任务应该已完成")
	}
	if len(subTasks) != 1 {
		t.Errorf("期望生成 1 个子任务, 实际为 %d 个", len(subTasks))
	}

	// 验证 TodoList 状态
	item := tc.TodoList.GetItem("sub-1")
	if item == nil {
		t.Fatal("应该能找到 sub-1 待办项")
	}
	if item.Status != TodoStatusCompleted {
		t.Errorf("待办项状态应为 'completed', 实际为 '%s'", item.Status)
	}
}
