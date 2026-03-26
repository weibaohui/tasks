/**
 * 领域事件单元测试
 */
package domain

import (
	"testing"
	"time"
)

func TestTodoPublishedEvent(t *testing.T) {
	taskID := NewTaskID("task-001")
	traceID := NewTraceID("trace-001")
	todoJSON := `{"todos": [{"content": "test"}]}`

	event := NewTodoPublishedEvent(taskID, traceID, todoJSON)

	if event.EventType() != "TodoPublished" {
		t.Errorf("期望 EventType 为 TodoPublished, 实际为 %s", event.EventType())
	}

	if event.TaskID() != taskID {
		t.Errorf("TaskID 不匹配")
	}

	if event.TraceID() != traceID {
		t.Errorf("TraceID 不匹配")
	}

	if event.TodoListJSON() != todoJSON {
		t.Errorf("TodoListJSON 不匹配")
	}

	if event.Timestamp() == 0 {
		t.Error("Timestamp 不应为 0")
	}
}

func TestTodoUpdatedEvent(t *testing.T) {
	taskID := NewTaskID("task-001")
	traceID := NewTraceID("trace-001")
	todoJSON := `{"todos": [{"content": "updated"}]}`

	event := NewTodoUpdatedEvent(taskID, traceID, todoJSON)

	if event.EventType() != "TodoUpdated" {
		t.Errorf("期望 EventType 为 TodoUpdated, 实际为 %s", event.EventType())
	}

	if event.TaskID() != taskID {
		t.Errorf("TaskID 不匹配")
	}

	if event.TodoListJSON() != todoJSON {
		t.Errorf("TodoListJSON 不匹配")
	}
}

func TestSubTaskCompletedEvent(t *testing.T) {
	parentID := NewTaskID("parent-001")
	subTaskID := NewTaskID("sub-001")
	traceID := NewTraceID("trace-001")

	event := NewSubTaskCompletedEvent(parentID, subTaskID, traceID)

	if event.EventType() != "SubTaskCompleted" {
		t.Errorf("期望 EventType 为 SubTaskCompleted, 实际为 %s", event.EventType())
	}

	if event.ParentTaskID() != parentID {
		t.Errorf("ParentTaskID 不匹配")
	}

	if event.SubTaskID() != subTaskID {
		t.Errorf("SubTaskID 不匹配")
	}

	if event.TraceID() != traceID {
		t.Errorf("TraceID 不匹配")
	}

	if event.Timestamp() == 0 {
		t.Error("Timestamp 不应为 0")
	}
}

func TestTodoSubTaskCreatedEvent(t *testing.T) {
	parentTask := NewTaskID("parent-001")
	subTask := NewTaskID("sub-001")
	traceID := NewTraceID("trace-001")

	event := NewTodoSubTaskCreatedEvent(
		parentTask,
		subTask,
		traceID,
		"sub-task-id",
		"sub-span-id",
		"parent-span",
		TaskTypeAgent,
		"Goal description",
	)

	if event.EventType() != "TodoSubTaskCreated" {
		t.Errorf("期望 EventType 为 TodoSubTaskCreated, 实际为 %s", event.EventType())
	}

	if event.ParentTaskID() != parentTask {
		t.Errorf("ParentTaskID 不匹配")
	}

	if event.SubTaskID() != subTask {
		t.Errorf("SubTaskID 不匹配")
	}

	if event.TraceID() != traceID {
		t.Errorf("TraceID 不匹配")
	}

	if event.SubTaskIDStr() != "sub-task-id" {
		t.Errorf("SubTaskIDStr 不匹配")
	}

	if event.SubTaskSpanID() != "sub-span-id" {
		t.Errorf("SubTaskSpanID 不匹配")
	}

	if event.ParentSpanID() != "parent-span" {
		t.Errorf("ParentSpanID 不匹配")
	}

	if event.SubTaskType() != TaskTypeAgent {
		t.Errorf("SubTaskType 不匹配")
	}

	if event.Goal() != "Goal description" {
		t.Errorf("Goal 不匹配")
	}

	if event.Timestamp() == 0 {
		t.Error("Timestamp 不应为 0")
	}
}

func TestTodoSubTaskCreatedEvent_TimestampSet(t *testing.T) {
	parentTask := NewTaskID("parent-001")
	subTask := NewTaskID("sub-001")
	traceID := NewTraceID("trace-001")

	before := time.Now().UnixMilli()
	event := NewTodoSubTaskCreatedEvent(
		parentTask,
		subTask,
		traceID,
		"id",
		"span",
		"parent",
		TaskTypeAgent,
		"goal",
	)
	after := time.Now().UnixMilli()

	if event.Timestamp() < before || event.Timestamp() > after {
		t.Error("Timestamp 应在合理范围内")
	}
}
