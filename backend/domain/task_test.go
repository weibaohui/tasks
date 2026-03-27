/**
 * Task 聚合根单元测试
 */
package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewTask(t *testing.T) {
	taskID := NewTaskID("test-task-1")
	traceID := NewTraceID("test-trace-1")
	spanID := NewSpanID("test-span-1")

	task, err := NewTask(
		taskID,
		traceID,
		spanID,
		nil,
		"测试任务",
		"任务描述",
		TaskTypeCustom,
		map[string]interface{}{"key": "value"},
		60*time.Second,
		3,
		5,
	)

	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}

	if task.ID() != taskID {
		t.Errorf("期望任务ID为 %v, 实际为 %v", taskID, task.ID())
	}

	if task.TraceID() != traceID {
		t.Errorf("期望追踪ID为 %v, 实际为 %v", traceID, task.TraceID())
	}

	if task.Name() != "测试任务" {
		t.Errorf("期望任务名称为 '测试任务', 实际为 '%s'", task.Name())
	}

	if task.Status() != TaskStatusPending {
		t.Errorf("期望初始状态为 Pending, 实际为 %v", task.Status())
	}

	if task.Priority() != 5 {
		t.Errorf("期望优先级为 5, 实际为 %d", task.Priority())
	}
}

func TestNewTask_EmptyName(t *testing.T) {
	taskID := NewTaskID("test-task-1")
	traceID := NewTraceID("test-trace-1")
	spanID := NewSpanID("test-span-1")

	_, err := NewTask(
		taskID,
		traceID,
		spanID,
		nil,
		"", // 空名称
		"",
		TaskTypeCustom,
		nil,
		0,
		0,
		0,
	)

	if err == nil {
		t.Error("期望返回错误，但实际返回 nil")
	}
}

func TestNewTask_NegativeTimeout(t *testing.T) {
	taskID := NewTaskID("test-task-1")
	traceID := NewTraceID("test-trace-1")
	spanID := NewSpanID("test-span-1")

	_, err := NewTask(
		taskID,
		traceID,
		spanID,
		nil,
		"测试任务",
		"",
		TaskTypeCustom,
		nil,
		-1*time.Second, // 负数超时
		0,
		0,
	)

	if err != ErrTimeoutNotPositive {
		t.Errorf("期望返回 ErrTimeoutNotPositive, 实际返回 %v", err)
	}
}

func TestTask_Start(t *testing.T) {
	task := createTestTask()

	err := task.Start()
	if err != nil {
		t.Fatalf("启动任务失败: %v", err)
	}

	if task.Status() != TaskStatusRunning {
		t.Errorf("期望状态为 Running, 实际为 %v", task.Status())
	}

	if task.StartedAt() == nil {
		t.Error("期望 StartedAt 不为 nil")
	}

	events := task.PopEvents()
	if len(events) != 2 {
		t.Errorf("期望 2 个领域事件, 实际为 %d", len(events))
	}
}

func TestTask_Start_InvalidTransition(t *testing.T) {
	task := createTestTask()

	err := task.Start()
	if err != nil {
		t.Fatalf("第一次启动任务失败: %v", err)
	}

	err = task.Start()
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}
}

func TestTask_Start_FromCompleted(t *testing.T) {
	task := createTestTask()

	task.Start()
	result := NewResult("result", "成功")
	task.Complete(result)

	err := task.Start()
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}
}

func TestTask_Complete(t *testing.T) {
	task := createTestTask()
	task.Start()

	result := NewResult(map[string]interface{}{"data": "test"}, "处理完成")
	err := task.Complete(result)

	if err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}

	if task.Status() != TaskStatusCompleted {
		t.Errorf("期望状态为 Completed, 实际为 %v", task.Status())
	}

	if task.Result() == nil {
		t.Fatal("期望 Result 不为 nil")
	}

	if task.Result().Message() != "处理完成" {
		t.Errorf("期望结果消息为 '处理完成', 实际为 '%s'", task.Result().Message())
	}

	if task.FinishedAt() == nil {
		t.Error("期望 FinishedAt 不为 nil")
	}
}

func TestTask_Complete_InvalidTransition(t *testing.T) {
	task := createTestTask()

	result := NewResult(nil, "")
	err := task.Complete(result)
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}
}

func TestTask_Fail(t *testing.T) {
	task := createTestTask()
	task.Start()

	taskErr := errors.New("处理失败")
	err := task.Fail(taskErr)

	if err != nil {
		t.Fatalf("标记任务失败: %v", err)
	}

	if task.Status() != TaskStatusFailed {
		t.Errorf("期望状态为 Failed, 实际为 %v", task.Status())
	}

	if task.Error() == nil {
		t.Error("期望 Error 不为 nil")
	}
}

func TestTask_Fail_InvalidTransition(t *testing.T) {
	task := createTestTask()

	taskErr := errors.New("处理失败")
	err := task.Fail(taskErr)
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}
}

func TestTask_Cancel(t *testing.T) {
	task := createTestTask()
	task.Start()

	err := task.Cancel()
	if err != nil {
		t.Fatalf("取消任务失败: %v", err)
	}

	if task.Status() != TaskStatusCancelled {
		t.Errorf("期望状态为 Cancelled, 实际为 %v", task.Status())
	}
}

func TestTask_Cancel_FromPending(t *testing.T) {
	task := createTestTask()

	err := task.Cancel()
	if err != nil {
		t.Fatalf("取消待处理任务失败: %v", err)
	}

	if task.Status() != TaskStatusCancelled {
		t.Errorf("期望状态为 Cancelled, 实际为 %v", task.Status())
	}
}

func TestTask_Cancel_InvalidTransition(t *testing.T) {
	task := createTestTask()
	task.Start()
	result := NewResult(nil, "")
	task.Complete(result)

	err := task.Cancel()
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}
}

func TestTask_UpdateProgress(t *testing.T) {
	task := createTestTask()
	task.Start()

	task.UpdateProgress(100, 50, "处理中", "已处理50项")

	progress := task.Progress()
	if progress.Total() != 100 {
		t.Errorf("期望总数为 100, 实际为 %d", progress.Total())
	}

	if progress.Current() != 50 {
		t.Errorf("期望当前为 50, 实际为 %d", progress.Current())
	}

	if progress.Percentage() != 50.0 {
		t.Errorf("期望百分比为 50.0, 实际为 %f", progress.Percentage())
	}

	if progress.Stage() != "处理中" {
		t.Errorf("期望阶段为 '处理中', 实际为 '%s'", progress.Stage())
	}
}

func TestTask_UpdateProgress_ZeroTotal(t *testing.T) {
	task := createTestTask()
	task.Start()

	task.UpdateProgress(0, 0, "准备中", "初始化")

	progress := task.Progress()
	if progress.Percentage() != 0.0 {
		t.Errorf("期望百分比为 0.0, 实际为 %f", progress.Percentage())
	}
}

func TestTask_ToSnapshot(t *testing.T) {
	task := createTestTask()
	task.Start()

	snap := task.ToSnapshot()

	if snap.ID != task.ID() {
		t.Errorf("快照ID不匹配")
	}

	if snap.Status != TaskStatusRunning {
		t.Errorf("期望快照状态为 Running, 实际为 %v", snap.Status)
	}
}

func TestTask_FromSnapshot(t *testing.T) {
	task := createTestTask()
	task.Start()

	snap := task.ToSnapshot()

	newTask := &Task{}
	newTask.FromSnapshot(&snap)

	if newTask.ID() != task.ID() {
		t.Errorf("恢复后ID不匹配")
	}

	if newTask.Status() != task.Status() {
		t.Errorf("恢复后状态不匹配: 期望 %v, 实际 %v", task.Status(), newTask.Status())
	}

	if newTask.Name() != task.Name() {
		t.Errorf("恢复后名称不匹配")
	}
}

func TestTask_PopEvents(t *testing.T) {
	task := createTestTask()

	events := task.PopEvents()
	if len(events) != 1 {
		t.Errorf("期望 1 个初始事件, 实际为 %d", len(events))
	}

	events = task.PopEvents()
	if len(events) != 0 {
		t.Errorf("期望 0 个事件, 实际为 %d", len(events))
	}
}

func TestTask_ConcurrentAccess(t *testing.T) {
	task := createTestTask()

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			task.Start()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.Progress()
		}
		done <- true
	}()

	<-done
	<-done
}

// Agent 模式测试用例

func TestNewTask_AgentType(t *testing.T) {
	taskID := NewTaskID("agent-task-1")
	traceID := NewTraceID("agent-trace-1")
	spanID := NewSpanID("agent-span-1")

	metadata := map[string]interface{}{
		"model":       "claude-3-opus",
		"prompt":      "请帮我分析这个数据",
		"max_tokens":  4096,
		"temperature": 0.7,
	}

	task, err := NewTask(
		taskID,
		traceID,
		spanID,
		nil,
		"Agent任务",
		"使用LLM Agent进行数据分析",
		TaskTypeAgent,
		metadata,
		120*time.Second,
		3,
		10,
	)

	if err != nil {
		t.Fatalf("创建Agent任务失败: %v", err)
	}

	if task.Type() != TaskTypeAgent {
		t.Errorf("期望任务类型为 Agent, 实际为 %v", task.Type())
	}

	if task.Name() != "Agent任务" {
		t.Errorf("期望任务名称为 'Agent任务', 实际为 '%s'", task.Name())
	}

	if task.Metadata()["model"] != "claude-3-opus" {
		t.Errorf("期望 model 为 'claude-3-opus', 实际为 '%v'", task.Metadata()["model"])
	}
}

func TestTask_AgentType_Lifecycle(t *testing.T) {
	task := createAgentTestTask()

	if task.Type() != TaskTypeAgent {
		t.Errorf("期望任务类型为 Agent, 实际为 %v", task.Type())
	}

	err := task.Start()
	if err != nil {
		t.Fatalf("启动Agent任务失败: %v", err)
	}

	if task.Status() != TaskStatusRunning {
		t.Errorf("期望状态为 Running, 实际为 %v", task.Status())
	}

	result := NewResult(map[string]interface{}{
		"response": "分析完成，发现3个关键洞察",
		"insights": []string{"趋势1", "趋势2", "趋势3"},
	}, "Agent任务执行成功")

	err = task.Complete(result)
	if err != nil {
		t.Fatalf("完成Agent任务失败: %v", err)
	}

	if task.Status() != TaskStatusCompleted {
		t.Errorf("期望状态为 Completed, 实际为 %v", task.Status())
	}

	if task.Result().Data().(map[string]interface{})["response"] != "分析完成，发现3个关键洞察" {
		t.Error("Agent任务结果不正确")
	}
}

func TestTask_AgentType_Progress(t *testing.T) {
	task := createAgentTestTask()
	task.Start()

	task.UpdateProgress(4, 1, "思考中", "分析问题...")
	progress := task.Progress()
	if progress.Stage() != "思考中" {
		t.Errorf("期望阶段为 '思考中', 实际为 '%s'", progress.Stage())
	}

	task.UpdateProgress(4, 2, "执行中", "调用工具...")
	progress = task.Progress()
	if progress.Percentage() != 50.0 {
		t.Errorf("期望百分比为 50.0, 实际为 %f", progress.Percentage())
	}

	task.UpdateProgress(4, 3, "生成中", "整合结果...")
	progress = task.Progress()
	if progress.Current() != 3 {
		t.Errorf("期望当前为 3, 实际为 %d", progress.Current())
	}

	task.UpdateProgress(4, 4, "完成", "任务结束")
	progress = task.Progress()
	if progress.Percentage() != 100.0 {
		t.Errorf("期望百分比为 100.0, 实际为 %f", progress.Percentage())
	}
}

func TestTask_AgentType_FailAndRetry(t *testing.T) {
	task := createAgentTestTask()
	task.Start()

	err := task.Fail(errors.New("LLM API超时"))
	if err != nil {
		t.Fatalf("标记Agent任务失败失败: %v", err)
	}

	if task.Status() != TaskStatusFailed {
		t.Errorf("期望状态为 Failed, 实际为 %v", task.Status())
	}

	if task.Error() == nil {
		t.Error("期望 Error 不为 nil")
	}
}

func TestTask_AgentType_ToFromSnapshot(t *testing.T) {
	task := createAgentTestTask()
	task.Start()
	task.UpdateProgress(100, 50, "处理中", "已完成一半")

	snap := task.ToSnapshot()

	if snap.Type != TaskTypeAgent {
		t.Errorf("期望快照类型为 Agent, 实际为 %v", snap.Type)
	}

	newTask := &Task{}
	newTask.FromSnapshot(&snap)

	if newTask.Type() != TaskTypeAgent {
		t.Errorf("恢复后类型不匹配")
	}

	if newTask.Status() != TaskStatusRunning {
		t.Errorf("恢复后状态不匹配")
	}

	if newTask.Progress().Current() != 50 {
		t.Errorf("恢复后进度不匹配: 期望 50, 实际 %d", newTask.Progress().Current())
	}
}

func TestTaskType_Agent_AllTransitions(t *testing.T) {
	task := createAgentTestTask()

	// Pending -> Running
	err := task.Start()
	if err != nil {
		t.Fatalf("启动任务失败: %v", err)
	}
	if task.Status() != TaskStatusRunning {
		t.Errorf("期望 Running, 实际 %v", task.Status())
	}

	// Running -> Completed
	result := NewResult("success", "完成")
	err = task.Complete(result)
	if err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}
	if task.Status() != TaskStatusCompleted {
		t.Errorf("期望 Completed, 实际 %v", task.Status())
	}
}

func createAgentTestTask() *Task {
	task, _ := NewTask(
		NewTaskID("agent-test-task"),
		NewTraceID("agent-test-trace"),
		NewSpanID("agent-test-span"),
		nil,
		"Agent测试任务",
		"Agent模式测试",
		TaskTypeAgent,
		map[string]interface{}{
			"model":      "claude-3-opus",
			"prompt":     "测试prompt",
			"max_tokens": 4096,
		},
		120*time.Second,
		3,
		10,
	)
	return task
}

func createTestTask() *Task {
	task, _ := NewTask(
		NewTaskID("test-task"),
		NewTraceID("test-trace"),
		NewSpanID("test-span"),
		nil,
		"测试任务",
		"",
		TaskTypeCustom,
		nil,
		60*time.Second,
		0,
		0,
	)
	return task
}
