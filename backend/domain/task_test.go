/**
 * Task 聚合根单元测试
 */
package domain

import (
	"errors"
	"strings"
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
		"测试目标",
		"测试验收标准",
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
		"测试目标",
		"测试验收标准",
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
		"测试目标",
		"测试验收标准",
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
	task.SetTaskConclusion("成功")
	task.Complete()

	err := task.Start()
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}
}

func TestTask_Complete(t *testing.T) {
	task := createTestTask()
	task.Start()
	task.SetTaskConclusion("测试结论")

	err := task.Complete()

	if err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}

	if task.Status() != TaskStatusCompleted {
		t.Errorf("期望状态为 Completed, 实际为 %v", task.Status())
	}

	if task.TaskConclusion() != "测试结论" {
		t.Errorf("期望任务结论为 '测试结论', 实际为 '%s'", task.TaskConclusion())
	}

	if task.FinishedAt() == nil {
		t.Error("期望 FinishedAt 不为 nil")
	}
}

func TestTask_Complete_InvalidTransition(t *testing.T) {
	task := createTestTask()
	task.SetTaskConclusion("测试结论") // 需要设置结论才能完成任务

	err := task.Complete()
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
	task.SetTaskConclusion("测试结论")
	task.Complete()

	err := task.Cancel()
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}
}

func TestTask_UpdateProgress(t *testing.T) {
	task := createTestTask()
	task.Start()

	task.UpdateProgress(50)

	progress := task.Progress()
	if progress.Value() != 50 {
		t.Errorf("期望进度为 50, 实际为 %d", progress.Value())
	}
}

func TestTask_UpdateProgress_ZeroTotal(t *testing.T) {
	task := createTestTask()
	task.Start()

	task.UpdateProgress(0)

	progress := task.Progress()
	if progress.Value() != 0 {
		t.Errorf("期望进度为 0, 实际为 %d", progress.Value())
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

	task, err := NewTask(
		taskID,
		traceID,
		spanID,
		nil,
		"Agent任务",
		"使用LLM Agent进行数据分析",
		TaskTypeAgent,
		"测试目标",
		"测试验收标准",
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

	task.SetTaskConclusion("分析完成，发现3个关键洞察")

	err = task.Complete()
	if err != nil {
		t.Fatalf("完成Agent任务失败: %v", err)
	}

	if task.Status() != TaskStatusCompleted {
		t.Errorf("期望状态为 Completed, 实际为 %v", task.Status())
	}

	if task.TaskConclusion() != "分析完成，发现3个关键洞察" {
		t.Errorf("期望任务结论为 '分析完成，发现3个关键洞察', 实际为 '%s'", task.TaskConclusion())
	}
}

func TestTask_AgentType_Progress(t *testing.T) {
	task := createAgentTestTask()
	task.Start()

	task.UpdateProgress(25)
	progress := task.Progress()
	if progress.Value() != 25 {
		t.Errorf("期望进度为 25, 实际为 %d", progress.Value())
	}

	task.UpdateProgress(50)
	progress = task.Progress()
	if progress.Value() != 50 {
		t.Errorf("期望进度为 50, 实际为 %d", progress.Value())
	}

	task.UpdateProgress(75)
	progress = task.Progress()
	if progress.Value() != 75 {
		t.Errorf("期望进度为 75, 实际为 %d", progress.Value())
	}

	task.UpdateProgress(100)
	progress = task.Progress()
	if progress.Value() != 100 {
		t.Errorf("期望进度为 100, 实际为 %d", progress.Value())
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
	task.UpdateProgress(50)

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

	if newTask.Progress().Value() != 50 {
		t.Errorf("恢复后进度不匹配: 期望 50, 实际 %d", newTask.Progress().Value())
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
	task.SetTaskConclusion("Agent任务完成")
	err = task.Complete()
	if err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}
	if task.Status() != TaskStatusCompleted {
		t.Errorf("期望 Completed, 实际 %v", task.Status())
	}
}

func TestParseTaskResultPairs_WithConclusionContainingDocSeparator(t *testing.T) {
	pair1 := TaskResultPair{
		TaskID:             "task-1",
		TaskName:           "子任务1",
		TaskRequirement:    "分析用户数据",
		AcceptanceCriteria: "产出结论",
		TaskConclusion:     "结论第一行\n---\n结论第三行",
		Status:             TaskStatusCompleted,
	}
	pair2 := TaskResultPair{
		TaskID:             "task-2",
		TaskName:           "子任务2",
		TaskRequirement:    "分析交易数据",
		AcceptanceCriteria: "产出结论",
		TaskConclusion:     "正常结论",
		Status:             TaskStatusCompleted,
	}

	records, err := AppendTaskResultPair("", pair1)
	if err != nil {
		t.Fatalf("追加第一条结果失败: %v", err)
	}
	records, err = AppendTaskResultPair(records, pair2)
	if err != nil {
		t.Fatalf("追加第二条结果失败: %v", err)
	}

	pairs, err := ParseTaskResultPairs(records)
	if err != nil {
		t.Fatalf("解析结果失败: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("期望解析出 2 条结果，实际为 %d", len(pairs))
	}
	if pairs[0].TaskConclusion != pair1.TaskConclusion {
		t.Fatalf("第一条结论不匹配，期望 %q，实际 %q", pair1.TaskConclusion, pairs[0].TaskConclusion)
	}
	if pairs[1].TaskID != "task-2" {
		t.Fatalf("第二条任务 ID 不匹配，期望 task-2，实际 %s", pairs[1].TaskID)
	}
}

func TestParseTaskResultPairs_WithCommentedYamlDocuments(t *testing.T) {
	records := strings.Join([]string{
		"# === 子任务 1 ===",
		"task_id: task-1",
		"task_name: 子任务1",
		"task_requirement: 任务1要求",
		"acceptance_criteria: 任务1验收",
		"task_conclusion: 任务1完成",
		"status: 2",
		"---",
		"# === 子任务 2 ===",
		"task_id: task-2",
		"task_name: 子任务2",
		"task_requirement: 任务2要求",
		"acceptance_criteria: 任务2验收",
		"task_conclusion: 任务2完成",
		"status: 2",
	}, "\n")

	pairs, err := ParseTaskResultPairs(records)
	if err != nil {
		t.Fatalf("解析带注释 YAML 文档失败: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("期望解析出 2 条结果，实际为 %d", len(pairs))
	}
	if pairs[0].TaskID != "task-1" || pairs[1].TaskID != "task-2" {
		t.Fatalf("解析结果任务 ID 不匹配: %+v", pairs)
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
		"测试目标",
		"测试验收标准",
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
		"测试目标",
		"测试验收标准",
		60*time.Second,
		0,
		0,
	)
	return task
}
