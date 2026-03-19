/**
 * 领域事件定义
 */
package domain

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
