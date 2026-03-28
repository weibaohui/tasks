/**
 * 领域事件定义
 */
package domain

import "time"

// DomainEvent 领域事件接口
type DomainEvent interface {
	// EventType 返回事件类型
	EventType() string
	// TraceID 返回追踪ID
	TraceID() TraceID
	// Timestamp 返回时间戳
	Timestamp() int64
}

// TaskCreatedEvent 任务创建事件
type TaskCreatedEvent struct {
	task *Task
}

func NewTaskCreatedEvent(task *Task) *TaskCreatedEvent {
	return &TaskCreatedEvent{task: task}
}

func (e *TaskCreatedEvent) EventType() string {
	return "TaskCreated"
}

func (e *TaskCreatedEvent) TraceID() TraceID {
	return e.task.TraceID()
}

func (e *TaskCreatedEvent) Timestamp() int64 {
	return e.task.CreatedAt().Unix()
}

// TaskStartedEvent 任务开始事件
type TaskStartedEvent struct {
	task *Task
}

func NewTaskStartedEvent(task *Task) *TaskStartedEvent {
	return &TaskStartedEvent{task: task}
}

func (e *TaskStartedEvent) EventType() string {
	return "TaskStarted"
}

func (e *TaskStartedEvent) TraceID() TraceID {
	return e.task.TraceID()
}

func (e *TaskStartedEvent) Timestamp() int64 {
	if e.task.StartedAt() != nil {
		return e.task.StartedAt().Unix()
	}
	return 0
}

// TaskCompletedEvent 任务完成事件
type TaskCompletedEvent struct {
	task *Task
}

func NewTaskCompletedEvent(task *Task) *TaskCompletedEvent {
	return &TaskCompletedEvent{task: task}
}

func (e *TaskCompletedEvent) EventType() string {
	return "TaskCompleted"
}

func (e *TaskCompletedEvent) TraceID() TraceID {
	return e.task.TraceID()
}

func (e *TaskCompletedEvent) Timestamp() int64 {
	if e.task.FinishedAt() != nil {
		return e.task.FinishedAt().Unix()
	}
	return 0
}

// TaskFailedEvent 任务失败事件
type TaskFailedEvent struct {
	task *Task
}

func NewTaskFailedEvent(task *Task) *TaskFailedEvent {
	return &TaskFailedEvent{task: task}
}

func (e *TaskFailedEvent) EventType() string {
	return "TaskFailed"
}

func (e *TaskFailedEvent) TraceID() TraceID {
	return e.task.TraceID()
}

func (e *TaskFailedEvent) Timestamp() int64 {
	if e.task.FinishedAt() != nil {
		return e.task.FinishedAt().Unix()
	}
	return 0
}

// TaskCancelledEvent 任务取消事件
type TaskCancelledEvent struct {
	task *Task
}

func NewTaskCancelledEvent(task *Task) *TaskCancelledEvent {
	return &TaskCancelledEvent{task: task}
}

func (e *TaskCancelledEvent) EventType() string {
	return "TaskCancelled"
}

func (e *TaskCancelledEvent) TraceID() TraceID {
	return e.task.TraceID()
}

func (e *TaskCancelledEvent) Timestamp() int64 {
	if e.task.FinishedAt() != nil {
		return e.task.FinishedAt().Unix()
	}
	return 0
}

// TaskPendingSummaryEvent 任务等待总结事件
type TaskPendingSummaryEvent struct {
	task *Task
}

func NewTaskPendingSummaryEvent(task *Task) *TaskPendingSummaryEvent {
	return &TaskPendingSummaryEvent{task: task}
}

func (e *TaskPendingSummaryEvent) EventType() string {
	return "TaskPendingSummary"
}

// Task 返回事件关联的任务
func (e *TaskPendingSummaryEvent) Task() *Task {
	return e.task
}

func (e *TaskPendingSummaryEvent) TraceID() TraceID {
	return e.task.TraceID()
}

func (e *TaskPendingSummaryEvent) Timestamp() int64 {
	return time.Now().Unix()
}

// TaskProgressUpdatedEvent 任务进度更新事件
type TaskProgressUpdatedEvent struct {
	task     *Task
	progress Progress
}

func NewTaskProgressUpdatedEvent(task *Task, progress Progress) *TaskProgressUpdatedEvent {
	return &TaskProgressUpdatedEvent{task: task, progress: progress}
}

func (e *TaskProgressUpdatedEvent) EventType() string {
	return "TaskProgressUpdated"
}

func (e *TaskProgressUpdatedEvent) TraceID() TraceID {
	return e.task.TraceID()
}

func (e *TaskProgressUpdatedEvent) Timestamp() int64 {
	return e.progress.UpdatedAt().Unix()
}

// GetProgress 获取进度
func (e *TaskProgressUpdatedEvent) GetProgress() Progress {
	return e.progress
}

// TodoPublishedEvent Todo列表发布事件
type TodoPublishedEvent struct {
	taskID       TaskID
	traceID      TraceID
	timestamp    int64
	todoListJSON string
}

func NewTodoPublishedEvent(taskID TaskID, traceID TraceID, todoListJSON string) *TodoPublishedEvent {
	return &TodoPublishedEvent{
		taskID:       taskID,
		traceID:      traceID,
		timestamp:    time.Now().UnixMilli(),
		todoListJSON: todoListJSON,
	}
}

func (e *TodoPublishedEvent) EventType() string {
	return "TodoPublished"
}

func (e *TodoPublishedEvent) TraceID() TraceID {
	return e.traceID
}

func (e *TodoPublishedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *TodoPublishedEvent) TodoListJSON() string {
	return e.todoListJSON
}

func (e *TodoPublishedEvent) TaskID() TaskID {
	return e.taskID
}

// TodoUpdatedEvent Todo列表更新事件
type TodoUpdatedEvent struct {
	taskID       TaskID
	traceID      TraceID
	timestamp    int64
	todoListJSON string
}

func NewTodoUpdatedEvent(taskID TaskID, traceID TraceID, todoListJSON string) *TodoUpdatedEvent {
	return &TodoUpdatedEvent{
		taskID:       taskID,
		traceID:      traceID,
		timestamp:    time.Now().UnixMilli(),
		todoListJSON: todoListJSON,
	}
}

func (e *TodoUpdatedEvent) EventType() string {
	return "TodoUpdated"
}

func (e *TodoUpdatedEvent) TraceID() TraceID {
	return e.traceID
}

func (e *TodoUpdatedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *TodoUpdatedEvent) TodoListJSON() string {
	return e.todoListJSON
}

func (e *TodoUpdatedEvent) TaskID() TaskID {
	return e.taskID
}

// SubTaskCompletedEvent 子任务完成事件（用于父任务更新Todo）
type SubTaskCompletedEvent struct {
	parentTaskID TaskID
	subTaskID    TaskID
	traceID      TraceID
	timestamp    int64
}

func NewSubTaskCompletedEvent(parentTaskID, subTaskID TaskID, traceID TraceID) *SubTaskCompletedEvent {
	return &SubTaskCompletedEvent{
		parentTaskID: parentTaskID,
		subTaskID:    subTaskID,
		traceID:      traceID,
		timestamp:    time.Now().UnixMilli(),
	}
}

func (e *SubTaskCompletedEvent) EventType() string {
	return "SubTaskCompleted"
}

func (e *SubTaskCompletedEvent) TraceID() TraceID {
	return e.traceID
}

func (e *SubTaskCompletedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *SubTaskCompletedEvent) ParentTaskID() TaskID {
	return e.parentTaskID
}

func (e *SubTaskCompletedEvent) SubTaskID() TaskID {
	return e.subTaskID
}

type TodoSubTaskCreatedEvent struct {
	parentTask TaskID
	subTask    TaskID
	trace      TraceID
	subTaskID  string
	subSpanID  string
	parentSpan string
	subType    TaskType
	goal       string
	timestamp  int64
}

func NewTodoSubTaskCreatedEvent(parentTask, subTask TaskID, trace TraceID, subTaskID, subSpanID, parentSpan string, subType TaskType, goal string) *TodoSubTaskCreatedEvent {
	return &TodoSubTaskCreatedEvent{
		parentTask: parentTask,
		subTask:    subTask,
		trace:      trace,
		subTaskID:  subTaskID,
		subSpanID:  subSpanID,
		parentSpan: parentSpan,
		subType:    subType,
		goal:       goal,
		timestamp:  time.Now().UnixMilli(),
	}
}

func (e *TodoSubTaskCreatedEvent) EventType() string {
	return "TodoSubTaskCreated"
}

func (e *TodoSubTaskCreatedEvent) TraceID() TraceID {
	return e.trace
}

func (e *TodoSubTaskCreatedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *TodoSubTaskCreatedEvent) ParentTaskID() TaskID {
	return e.parentTask
}

func (e *TodoSubTaskCreatedEvent) SubTaskID() TaskID {
	return e.subTask
}

func (e *TodoSubTaskCreatedEvent) SubTaskIDStr() string {
	return e.subTaskID
}

func (e *TodoSubTaskCreatedEvent) SubTaskSpanID() string {
	return e.subSpanID
}

func (e *TodoSubTaskCreatedEvent) ParentSpanID() string {
	return e.parentSpan
}

func (e *TodoSubTaskCreatedEvent) SubTaskType() TaskType {
	return e.subType
}

func (e *TodoSubTaskCreatedEvent) Goal() string {
	return e.goal
}
