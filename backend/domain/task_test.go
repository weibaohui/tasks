/**
 * Task 聚合根单元测试
 */
package domain

import (
	"errors"
	"fmt"
	"strings"
	"sync"
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

// ==================== Getter/Setter 测试 ====================

func TestTask_Getters(t *testing.T) {
	task := createTestTask()

	// 测试基本 getter
	if task.SpanID().String() != "test-span" {
		t.Errorf("期望 SpanID 为 'test-span', 实际为 '%s'", task.SpanID().String())
	}

	if task.ParentID() != nil {
		t.Error("期望 ParentID 为 nil")
	}

	if task.Description() != "" {
		t.Errorf("期望 Description 为空, 实际为 '%s'", task.Description())
	}

	if task.Timeout() != 60*time.Second {
		t.Errorf("期望 Timeout 为 60s, 实际为 %v", task.Timeout())
	}

	if task.MaxRetries() != 0 {
		t.Errorf("期望 MaxRetries 为 0, 实际为 %d", task.MaxRetries())
	}

	if task.CreatedAt().IsZero() {
		t.Error("期望 CreatedAt 不为零值")
	}
}

func TestTask_AcceptanceCriteria(t *testing.T) {
	task := createTestTask()

	// 初始值
	if task.AcceptanceCriteria() != "测试验收标准" {
		t.Errorf("期望初始 AcceptanceCriteria 为 '测试验收标准', 实际为 '%s'", task.AcceptanceCriteria())
	}

	// 设置新值
	task.SetAcceptanceCriteria("新的验收标准")
	if task.AcceptanceCriteria() != "新的验收标准" {
		t.Errorf("期望 AcceptanceCriteria 为 '新的验收标准', 实际为 '%s'", task.AcceptanceCriteria())
	}
}

func TestTask_TaskRequirement(t *testing.T) {
	task := createTestTask()

	// 初始值
	if task.TaskRequirement() != "测试目标" {
		t.Errorf("期望初始 TaskRequirement 为 '测试目标', 实际为 '%s'", task.TaskRequirement())
	}

	// 设置新值
	task.SetTaskRequirement("新的任务要求")
	if task.TaskRequirement() != "新的任务要求" {
		t.Errorf("期望 TaskRequirement 为 '新的任务要求', 实际为 '%s'", task.TaskRequirement())
	}
}

func TestTask_SubtaskRecords(t *testing.T) {
	task := createTestTask()

	// 初始值应为空
	if task.SubtaskRecords() != "" {
		t.Errorf("期望初始 SubtaskRecords 为空, 实际为 '%s'", task.SubtaskRecords())
	}

	// 设置新值
	task.SetSubtaskRecords("task1: done")
	if task.SubtaskRecords() != "task1: done" {
		t.Errorf("期望 SubtaskRecords 为 'task1: done', 实际为 '%s'", task.SubtaskRecords())
	}
}

func TestTask_UserCode(t *testing.T) {
	task := createTestTask()

	// 初始值应为空
	if task.UserCode() != "" {
		t.Errorf("期望初始 UserCode 为空, 实际为 '%s'", task.UserCode())
	}

	// 设置新值
	task.SetUserCode("USER001")
	if task.UserCode() != "USER001" {
		t.Errorf("期望 UserCode 为 'USER001', 实际为 '%s'", task.UserCode())
	}
}

func TestTask_AgentCode(t *testing.T) {
	task := createTestTask()

	// 初始值应为空
	if task.AgentCode() != "" {
		t.Errorf("期望初始 AgentCode 为空, 实际为 '%s'", task.AgentCode())
	}

	// 设置新值
	task.SetAgentCode("AGENT001")
	if task.AgentCode() != "AGENT001" {
		t.Errorf("期望 AgentCode 为 'AGENT001', 实际为 '%s'", task.AgentCode())
	}
}

func TestTask_ChannelCode(t *testing.T) {
	task := createTestTask()

	// 初始值应为空
	if task.ChannelCode() != "" {
		t.Errorf("期望初始 ChannelCode 为空, 实际为 '%s'", task.ChannelCode())
	}

	// 设置新值
	task.SetChannelCode("CHANNEL001")
	if task.ChannelCode() != "CHANNEL001" {
		t.Errorf("期望 ChannelCode 为 'CHANNEL001', 实际为 '%s'", task.ChannelCode())
	}
}

func TestTask_SessionKey(t *testing.T) {
	task := createTestTask()

	// 初始值应为空
	if task.SessionKey() != "" {
		t.Errorf("期望初始 SessionKey 为空, 实际为 '%s'", task.SessionKey())
	}

	// 设置新值
	task.SetSessionKey("SESSION001")
	if task.SessionKey() != "SESSION001" {
		t.Errorf("期望 SessionKey 为 'SESSION001', 实际为 '%s'", task.SessionKey())
	}
}

func TestTask_TodoList(t *testing.T) {
	task := createTestTask()

	// 初始值应为空
	if task.TodoList() != "" {
		t.Errorf("期望初始 TodoList 为空, 实际为 '%s'", task.TodoList())
	}

	// 设置新值
	task.SetTodoList("[{'id': 1, 'done': false}]")
	if task.TodoList() != "[{'id': 1, 'done': false}]" {
		t.Errorf("期望 TodoList 为 '[{'id': 1, 'done': false}]', 实际为 '%s'", task.TodoList())
	}
}

func TestTask_Analysis(t *testing.T) {
	task := createTestTask()

	// 初始值应为空
	if task.Analysis() != "" {
		t.Errorf("期望初始 Analysis 为空, 实际为 '%s'", task.Analysis())
	}

	// 设置新值
	task.SetAnalysis("任务分析结果")
	if task.Analysis() != "任务分析结果" {
		t.Errorf("期望 Analysis 为 '任务分析结果', 实际为 '%s'", task.Analysis())
	}
}

func TestTask_Depth(t *testing.T) {
	task := createTestTask()

	// 初始值应为0
	if task.Depth() != 0 {
		t.Errorf("期望初始 Depth 为 0, 实际为 %d", task.Depth())
	}

	// 设置新值
	task.SetDepth(3)
	if task.Depth() != 3 {
		t.Errorf("期望 Depth 为 3, 实际为 %d", task.Depth())
	}
}

func TestTask_ParentSpan(t *testing.T) {
	task := createTestTask()

	// 初始值应为空
	if task.ParentSpan() != "" {
		t.Errorf("期望初始 ParentSpan 为空, 实际为 '%s'", task.ParentSpan())
	}

	// 设置新值
	task.SetParentSpan("parent-span-123")
	if task.ParentSpan() != "parent-span-123" {
		t.Errorf("期望 ParentSpan 为 'parent-span-123', 实际为 '%s'", task.ParentSpan())
	}
}

// ==================== 状态转换测试 ====================

func TestTask_PendingSummary(t *testing.T) {
	task := createTestTask()

	// 必须先启动才能进入 PendingSummary
	err := task.Start()
	if err != nil {
		t.Fatalf("启动任务失败: %v", err)
	}

	// 进入等待总结状态
	err = task.PendingSummary()
	if err != nil {
		t.Fatalf("进入等待总结状态失败: %v", err)
	}

	if task.Status() != TaskStatusPendingSummary {
		t.Errorf("期望状态为 PendingSummary, 实际为 %v", task.Status())
	}

	// 验证事件
	events := task.PopEvents()
	found := false
	for _, e := range events {
		if e.EventType() == "TaskPendingSummary" {
			found = true
			break
		}
	}
	if !found {
		t.Error("期望找到 TaskPendingSummary 事件")
	}
}

func TestTask_PendingSummary_InvalidTransition(t *testing.T) {
	task := createTestTask()

	// 直接从 Pending 状态进入 PendingSummary 应该失败
	err := task.PendingSummary()
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}

	// 从 Completed 状态进入 PendingSummary 也应该失败
	task.Start()
	task.SetTaskConclusion("完成")
	task.Complete()

	err = task.PendingSummary()
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}
}

func TestTask_Complete_WithoutConclusion(t *testing.T) {
	task := createTestTask()
	task.Start()

	// 未设置结论时完成任务应该失败
	err := task.Complete()
	if err != ErrTaskConclusionRequired {
		t.Errorf("期望返回 ErrTaskConclusionRequired, 实际返回 %v", err)
	}
}

func TestTask_Complete_FromPendingSummary(t *testing.T) {
	task := createTestTask()
	task.Start()
	task.SetTaskConclusion("任务总结")
	task.PendingSummary()

	err := task.Complete()
	if err != nil {
		t.Fatalf("从 PendingSummary 完成任务失败: %v", err)
	}

	if task.Status() != TaskStatusCompleted {
		t.Errorf("期望状态为 Completed, 实际为 %v", task.Status())
	}
}

func TestTask_Fail_FromPendingSummary(t *testing.T) {
	task := createTestTask()
	task.Start()
	task.PendingSummary()

	err := task.Fail(errors.New("总结失败"))
	if err != nil {
		t.Fatalf("从 PendingSummary 标记失败失败: %v", err)
	}

	if task.Status() != TaskStatusFailed {
		t.Errorf("期望状态为 Failed, 实际为 %v", task.Status())
	}
}

func TestTask_Cancel_FromPendingSummary(t *testing.T) {
	task := createTestTask()
	task.Start()
	task.PendingSummary()

	err := task.Cancel()
	if err != nil {
		t.Fatalf("从 PendingSummary 取消任务失败: %v", err)
	}

	if task.Status() != TaskStatusCancelled {
		t.Errorf("期望状态为 Cancelled, 实际为 %v", task.Status())
	}
}

func TestTask_Cancel_FromRunning(t *testing.T) {
	task := createTestTask()
	task.Start()

	err := task.Cancel()
	if err != nil {
		t.Fatalf("从 Running 取消任务失败: %v", err)
	}

	if task.Status() != TaskStatusCancelled {
		t.Errorf("期望状态为 Cancelled, 实际为 %v", task.Status())
	}
}

func TestTask_Fail_FromPending(t *testing.T) {
	task := createTestTask()

	// 从 Pending 直接 Fail 应该失败
	err := task.Fail(errors.New("失败"))
	if err != ErrInvalidStatusTransition {
		t.Errorf("期望返回 ErrInvalidStatusTransition, 实际返回 %v", err)
	}
}

func TestTask_AllStatusTransitions(t *testing.T) {
	testCases := []struct {
		name           string
		initialStatus  TaskStatus
		targetStatus   TaskStatus
		setupTask      func(*Task)
		expectedError  error
	}{
		{"Pending->Running", TaskStatusPending, TaskStatusRunning, nil, nil},
		{"Pending->Cancelled", TaskStatusPending, TaskStatusCancelled, nil, nil},
		{"Running->Completed", TaskStatusRunning, TaskStatusCompleted, func(t *Task) { t.SetTaskConclusion("完成") }, nil},
		{"Running->Failed", TaskStatusRunning, TaskStatusFailed, nil, nil},
		{"Running->Cancelled", TaskStatusRunning, TaskStatusCancelled, nil, nil},
		{"Running->PendingSummary", TaskStatusRunning, TaskStatusPendingSummary, nil, nil},
		{"PendingSummary->Completed", TaskStatusPendingSummary, TaskStatusCompleted, func(t *Task) { t.SetTaskConclusion("完成") }, nil},
		{"PendingSummary->Failed", TaskStatusPendingSummary, TaskStatusFailed, nil, nil},
		{"PendingSummary->Cancelled", TaskStatusPendingSummary, TaskStatusCancelled, nil, nil},
		{"Completed->Running", TaskStatusCompleted, TaskStatusRunning, nil, ErrInvalidStatusTransition},
		{"Failed->Running", TaskStatusFailed, TaskStatusRunning, nil, ErrInvalidStatusTransition},
		{"Cancelled->Running", TaskStatusCancelled, TaskStatusRunning, nil, ErrInvalidStatusTransition},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task := createTestTask()

			// 设置初始状态
			switch tc.initialStatus {
			case TaskStatusRunning:
				task.Start()
			case TaskStatusPendingSummary:
				task.Start()
				task.PendingSummary()
			case TaskStatusCompleted:
				task.Start()
				task.SetTaskConclusion("完成")
				task.Complete()
			case TaskStatusFailed:
				task.Start()
				task.Fail(errors.New("失败"))
			case TaskStatusCancelled:
				task.Cancel()
			}

			// 执行前置设置
			if tc.setupTask != nil {
				tc.setupTask(task)
			}

			// 尝试状态转换
			var err error
			switch tc.targetStatus {
			case TaskStatusRunning:
				err = task.Start()
			case TaskStatusCompleted:
				err = task.Complete()
			case TaskStatusFailed:
				err = task.Fail(errors.New("失败"))
			case TaskStatusCancelled:
				err = task.Cancel()
			case TaskStatusPendingSummary:
				err = task.PendingSummary()
			}

			if err != tc.expectedError {
				t.Errorf("期望错误 %v, 实际 %v", tc.expectedError, err)
			}
		})
	}
}

// ==================== TaskResultPair 测试 ====================

func TestTaskResultPair_ToYAML(t *testing.T) {
	now := time.Now()
	pair := TaskResultPair{
		TaskID:             "task-1",
		TaskName:           "测试任务",
		TaskRequirement:    "任务要求",
		AcceptanceCriteria: "验收标准",
		TaskConclusion:     "任务结论",
		CompletedAt:        &now,
		Status:             TaskStatusCompleted,
	}

	yamlStr := pair.ToYAML()
	if yamlStr == "" {
		t.Error("ToYAML 返回空字符串")
	}

	// 验证 YAML 包含关键字段
	if !strings.Contains(yamlStr, "task_id: task-1") {
		t.Error("YAML 中不包含 task_id")
	}
	if !strings.Contains(yamlStr, "task_name: 测试任务") {
		t.Error("YAML 中不包含 task_name")
	}
}

func TestTaskResultPair_ToYAML_Empty(t *testing.T) {
	// 测试空结构体的 YAML 序列化
	pair := TaskResultPair{}
	yamlStr := pair.ToYAML()
	if yamlStr == "" {
		t.Error("空结构体的 ToYAML 返回空字符串")
	}
}

func TestParseTaskResultPairs_Empty(t *testing.T) {
	pairs, err := ParseTaskResultPairs("")
	if err != nil {
		t.Errorf("解析空字符串失败: %v", err)
	}
	if len(pairs) != 0 {
		t.Errorf("期望返回空切片, 实际为 %d 个元素", len(pairs))
	}
}

func TestParseTaskResultPairs_Single(t *testing.T) {
	pair := TaskResultPair{
		TaskID:             "task-1",
		TaskName:           "单任务",
		TaskRequirement:    "要求",
		AcceptanceCriteria: "标准",
		TaskConclusion:     "结论",
		Status:             TaskStatusCompleted,
	}

	yamlStr := pair.ToYAML()
	pairs, err := ParseTaskResultPairs(yamlStr)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("期望 1 个结果, 实际为 %d", len(pairs))
	}
	if pairs[0].TaskID != "task-1" {
		t.Errorf("TaskID 不匹配")
	}
}

func TestParseTaskResultPairs_InvalidYAML(t *testing.T) {
	invalidYAML := "invalid: yaml: [: content"
	_, err := ParseTaskResultPairs(invalidYAML)
	if err == nil {
		t.Error("期望解析无效 YAML 返回错误")
	}
}

func TestAppendTaskResultPair_EmptyExisting(t *testing.T) {
	pair := TaskResultPair{
		TaskID:             "task-1",
		TaskName:           "任务1",
		TaskRequirement:    "要求1",
		AcceptanceCriteria: "标准1",
		TaskConclusion:     "结论1",
		Status:             TaskStatusCompleted,
	}

	result, err := AppendTaskResultPair("", pair)
	if err != nil {
		t.Fatalf("追加失败: %v", err)
	}

	// 验证结果只包含一个文档（没有分隔符）
	if strings.Contains(result, "---") {
		t.Error("单个文档不应包含分隔符")
	}
}

func TestAppendTaskResultPair_Multiple(t *testing.T) {
	pair1 := TaskResultPair{
		TaskID:             "task-1",
		TaskName:           "任务1",
		TaskRequirement:    "要求1",
		AcceptanceCriteria: "标准1",
		TaskConclusion:     "结论1",
		Status:             TaskStatusCompleted,
	}
	pair2 := TaskResultPair{
		TaskID:             "task-2",
		TaskName:           "任务2",
		TaskRequirement:    "要求2",
		AcceptanceCriteria: "标准2",
		TaskConclusion:     "结论2",
		Status:             TaskStatusFailed,
	}

	result, err := AppendTaskResultPair("", pair1)
	if err != nil {
		t.Fatalf("追加第一个失败: %v", err)
	}

	result, err = AppendTaskResultPair(result, pair2)
	if err != nil {
		t.Fatalf("追加第二个失败: %v", err)
	}

	// 验证包含分隔符
	if !strings.Contains(result, "---") {
		t.Error("多个文档应包含分隔符")
	}

	// 解析验证
	pairs, err := ParseTaskResultPairs(result)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(pairs) != 2 {
		t.Errorf("期望 2 个结果, 实际为 %d", len(pairs))
	}
}

// ==================== 快照测试 ====================

func TestTask_Snapshot_AllFields(t *testing.T) {
	task := createTestTask()
	task.SetAcceptanceCriteria("验收标准")
	task.SetTaskRequirement("任务要求")
	task.SetTaskConclusion("任务结论")
	task.SetSubtaskRecords("子任务记录")
	task.SetUserCode("USER001")
	task.SetAgentCode("AGENT001")
	task.SetChannelCode("CHANNEL001")
	task.SetSessionKey("SESSION001")
	task.SetTodoList("TODO列表")
	task.SetAnalysis("分析结果")
	task.SetDepth(2)
	task.SetParentSpan("parent-span")

	task.Start()
	task.UpdateProgress(75)

	snap := task.ToSnapshot()

	// 验证所有字段
	if snap.AcceptanceCriteria != "验收标准" {
		t.Error("AcceptanceCriteria 不匹配")
	}
	if snap.TaskRequirement != "任务要求" {
		t.Error("TaskRequirement 不匹配")
	}
	if snap.TaskConclusion != "任务结论" {
		t.Error("TaskConclusion 不匹配")
	}
	if snap.SubtaskRecords != "子任务记录" {
		t.Error("SubtaskRecords 不匹配")
	}
	if snap.UserCode != "USER001" {
		t.Error("UserCode 不匹配")
	}
	if snap.AgentCode != "AGENT001" {
		t.Error("AgentCode 不匹配")
	}
	if snap.ChannelCode != "CHANNEL001" {
		t.Error("ChannelCode 不匹配")
	}
	if snap.SessionKey != "SESSION001" {
		t.Error("SessionKey 不匹配")
	}
	if snap.TodoList != "TODO列表" {
		t.Error("TodoList 不匹配")
	}
	if snap.Analysis != "分析结果" {
		t.Error("Analysis 不匹配")
	}
	if snap.Depth != 2 {
		t.Error("Depth 不匹配")
	}
	if snap.ParentSpan != "parent-span" {
		t.Error("ParentSpan 不匹配")
	}
	if snap.Progress.Value() != 75 {
		t.Error("Progress 不匹配")
	}
}

func TestTask_Snapshot_WithError(t *testing.T) {
	task := createTestTask()
	task.Start()
	task.Fail(errors.New("任务执行失败"))

	snap := task.ToSnapshot()
	if snap.ErrorMsg != "任务执行失败" {
		t.Errorf("期望 ErrorMsg 为 '任务执行失败', 实际为 '%s'", snap.ErrorMsg)
	}
}

func TestTask_FromSnapshot_WithError(t *testing.T) {
	snap := TaskSnapshot{
		ID:        NewTaskID("test-id"),
		TraceID:   NewTraceID("test-trace"),
		SpanID:    NewSpanID("test-span"),
		Name:      "测试任务",
		Type:      TaskTypeCustom,
		Status:    TaskStatusFailed,
		ErrorMsg:  "快照错误信息",
		CreatedAt: time.Now(),
	}

	newTask := &Task{}
	newTask.FromSnapshot(&snap)

	if newTask.Error() == nil {
		t.Error("期望 Error 不为 nil")
	}
	if newTask.Error().Error() != "快照错误信息" {
		t.Errorf("期望错误信息为 '快照错误信息', 实际为 '%s'", newTask.Error().Error())
	}
}

func TestTask_FromSnapshot_EmptyError(t *testing.T) {
	snap := TaskSnapshot{
		ID:        NewTaskID("test-id"),
		TraceID:   NewTraceID("test-trace"),
		SpanID:    NewSpanID("test-span"),
		Name:      "测试任务",
		Type:      TaskTypeCustom,
		Status:    TaskStatusPending,
		ErrorMsg:  "",
		CreatedAt: time.Now(),
	}

	newTask := &Task{}
	newTask.FromSnapshot(&snap)

	if newTask.Error() != nil {
		t.Error("期望 Error 为 nil")
	}
}

func TestTask_FromSnapshot_WithParentID(t *testing.T) {
	parentID := NewTaskID("parent-id")
	snap := TaskSnapshot{
		ID:        NewTaskID("test-id"),
		TraceID:   NewTraceID("test-trace"),
		SpanID:    NewSpanID("test-span"),
		ParentID:  &parentID,
		Name:      "子任务",
		Type:      TaskTypeCustom,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	newTask := &Task{}
	newTask.FromSnapshot(&snap)

	if newTask.ParentID() == nil {
		t.Fatal("期望 ParentID 不为 nil")
	}
	if newTask.ParentID().String() != "parent-id" {
		t.Errorf("期望 ParentID 为 'parent-id', 实际为 '%s'", newTask.ParentID().String())
	}
}

// ==================== 领域事件测试 ====================

func TestTask_Events(t *testing.T) {
	task := createTestTask()

	// 初始应该有 TaskCreated 事件
	events := task.PopEvents()
	if len(events) != 1 {
		t.Fatalf("期望 1 个初始事件, 实际为 %d", len(events))
	}
	if events[0].EventType() != "TaskCreated" {
		t.Errorf("期望事件类型为 TaskCreated, 实际为 %s", events[0].EventType())
	}

	// 启动任务
	task.Start()
	events = task.PopEvents()
	if len(events) != 1 {
		t.Fatalf("期望 1 个启动事件, 实际为 %d", len(events))
	}
	if events[0].EventType() != "TaskStarted" {
		t.Errorf("期望事件类型为 TaskStarted, 实际为 %s", events[0].EventType())
	}

	// 更新进度
	task.UpdateProgress(50)
	events = task.PopEvents()
	if len(events) != 1 {
		t.Fatalf("期望 1 个进度更新事件, 实际为 %d", len(events))
	}
	if events[0].EventType() != "TaskProgressUpdated" {
		t.Errorf("期望事件类型为 TaskProgressUpdated, 实际为 %s", events[0].EventType())
	}

	// 完成任务
	task.SetTaskConclusion("完成")
	task.Complete()
	events = task.PopEvents()
	if len(events) != 1 {
		t.Fatalf("期望 1 个完成事件, 实际为 %d", len(events))
	}
	if events[0].EventType() != "TaskCompleted" {
		t.Errorf("期望事件类型为 TaskCompleted, 实际为 %s", events[0].EventType())
	}
}

func TestTaskProgressUpdatedEvent_GetProgress(t *testing.T) {
	task := createTestTask()
	task.Start()
	task.UpdateProgress(66)

	events := task.PopEvents()
	for _, e := range events {
		if e.EventType() == "TaskProgressUpdated" {
			progressEvent, ok := e.(*TaskProgressUpdatedEvent)
			if !ok {
				t.Error("无法转换为 TaskProgressUpdatedEvent")
				continue
			}
			if progressEvent.GetProgress().Value() != 66 {
				t.Errorf("期望进度为 66, 实际为 %d", progressEvent.GetProgress().Value())
			}
		}
	}
}

// ==================== 并发安全测试 ====================

func TestTask_ConcurrentSetters(t *testing.T) {
	task := createTestTask()

	done := make(chan bool, 10)

	// 并发写入不同的字段
	go func() {
		for i := 0; i < 100; i++ {
			task.SetAcceptanceCriteria(fmt.Sprintf("criteria-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.SetTaskRequirement(fmt.Sprintf("requirement-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.SetTaskConclusion(fmt.Sprintf("conclusion-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.SetUserCode(fmt.Sprintf("user-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.SetAgentCode(fmt.Sprintf("agent-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.SetChannelCode(fmt.Sprintf("channel-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.SetSessionKey(fmt.Sprintf("session-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.SetTodoList(fmt.Sprintf("todo-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.SetAnalysis(fmt.Sprintf("analysis-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.SetDepth(i)
		}
		done <- true
	}()

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestTask_ConcurrentReadWrite(t *testing.T) {
	task := createTestTask()
	task.Start()

	done := make(chan bool, 4)

	// 并发读写
	go func() {
		for i := 0; i < 100; i++ {
			task.SetTaskConclusion(fmt.Sprintf("conclusion-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = task.TaskConclusion()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.UpdateProgress(i % 100)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = task.Progress()
		}
		done <- true
	}()

	for i := 0; i < 4; i++ {
		<-done
	}
}

func TestTask_ConcurrentSnapshot(t *testing.T) {
	task := createTestTask()
	task.Start()

	done := make(chan bool, 3)

	// 同时进行快照和修改
	go func() {
		for i := 0; i < 100; i++ {
			task.SetTaskConclusion(fmt.Sprintf("conclusion-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			task.UpdateProgress(i % 100)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = task.ToSnapshot()
		}
		done <- true
	}()

	for i := 0; i < 3; i++ {
		<-done
	}
}

// ==================== 边界情况测试 ====================

func TestNewTask_MissingTaskRequirement(t *testing.T) {
	_, err := NewTask(
		NewTaskID("test"),
		NewTraceID("trace"),
		NewSpanID("span"),
		nil,
		"任务名称",
		"",
		TaskTypeCustom,
		"", // 空任务要求
		"验收标准",
		0,
		0,
		0,
	)
	if err != ErrTaskRequirementRequired {
		t.Errorf("期望返回 ErrTaskRequirementRequired, 实际返回 %v", err)
	}
}

func TestNewTask_MissingAcceptanceCriteria(t *testing.T) {
	_, err := NewTask(
		NewTaskID("test"),
		NewTraceID("trace"),
		NewSpanID("span"),
		nil,
		"任务名称",
		"",
		TaskTypeCustom,
		"任务要求",
		"", // 空验收标准
		0,
		0,
		0,
	)
	if err != ErrAcceptanceCriteriaRequired {
		t.Errorf("期望返回 ErrAcceptanceCriteriaRequired, 实际返回 %v", err)
	}
}

func TestTask_UpdateProgress_Concurrent(t *testing.T) {
	task := createTestTask()
	task.Start()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			task.UpdateProgress(val)
		}(i)
	}
	wg.Wait()

	// 验证进度在有效范围内
	progress := task.Progress()
	if progress.Value() < 0 || progress.Value() > 100 {
		t.Errorf("进度值 %d 超出有效范围 [0, 100]", progress.Value())
	}
}

func TestTaskResultPair_WithNilTime(t *testing.T) {
	pair := TaskResultPair{
		TaskID:             "task-1",
		TaskName:           "测试任务",
		TaskRequirement:    "要求",
		AcceptanceCriteria: "标准",
		TaskConclusion:     "结论",
		CompletedAt:        nil, // nil 时间
		Status:             TaskStatusCompleted,
	}

	yamlStr := pair.ToYAML()
	if yamlStr == "" {
		t.Error("nil CompletedAt 的 ToYAML 返回空字符串")
	}

	// 验证可以正确解析回来
	pairs, err := ParseTaskResultPairs(yamlStr)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("期望 1 个结果, 实际为 %d", len(pairs))
	}
	if pairs[0].CompletedAt != nil {
		t.Error("期望 CompletedAt 为 nil")
	}
}

func TestParseTaskResultPairs_WithEmptyDocuments(t *testing.T) {
	// 包含空文档的 YAML
	records := "---\n---\ntask_id: task-1\ntask_name: 任务1\n"

	pairs, err := ParseTaskResultPairs(records)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	// 应该跳过空文档
	if len(pairs) != 1 {
		t.Errorf("期望 1 个结果（跳过空文档）, 实际为 %d", len(pairs))
	}
}

// ==================== 辅助函数 ====================

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
