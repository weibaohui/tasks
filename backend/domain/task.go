/**
 * Task 聚合根
 * 是任务管理的核心实体，负责维护任务的完整生命周期
 */
package domain

import (
	"errors"
	"sync"
	"time"
)

// 领域错误定义
var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrTaskAlreadyStarted      = errors.New("task already started")
	ErrTaskNotRunning          = errors.New("task is not running")
	ErrTaskAlreadyFinished     = errors.New("task already finished")
	ErrTimeoutNotPositive      = errors.New("timeout must be positive")
)

// Task 任务聚合根
type Task struct {
	id       TaskID
	traceID  TraceID
	spanID   SpanID
	parentID *TaskID

	name        string
	description string
	taskType    TaskType
	metadata    map[string]interface{}

	timeout    time.Duration
	maxRetries int
	priority   int

	status   TaskStatus
	progress Progress
	result   *Result
	execErr  error

	createdAt  time.Time
	startedAt  *time.Time
	finishedAt *time.Time

	domainEvents []DomainEvent

	mu sync.RWMutex
}

// NewTask 工厂方法：创建任务
func NewTask(
	id TaskID,
	traceID TraceID,
	spanID SpanID,
	parentID *TaskID,
	name string,
	description string,
	taskType TaskType,
	metadata map[string]interface{},
	timeout time.Duration,
	maxRetries int,
	priority int,
) (*Task, error) {
	if name == "" {
		return nil, errors.New("task name is required")
	}
	if timeout < 0 {
		return nil, ErrTimeoutNotPositive
	}

	task := &Task{
		id:          id,
		traceID:     traceID,
		spanID:      spanID,
		parentID:    parentID,
		name:        name,
		description: description,
		taskType:    taskType,
		metadata:    metadata,
		timeout:     timeout,
		maxRetries:  maxRetries,
		priority:    priority,
		status:      TaskStatusPending,
		progress:    NewProgress(),
		createdAt:   time.Now(),
	}

	task.recordEvent(NewTaskCreatedEvent(task))

	return task, nil
}

// 标识访问方法
func (t *Task) ID() TaskID                       { return t.id }
func (t *Task) TraceID() TraceID                 { return t.traceID }
func (t *Task) SpanID() SpanID                   { return t.spanID }
func (t *Task) ParentID() *TaskID                { return t.parentID }
func (t *Task) Name() string                     { return t.name }
func (t *Task) Description() string              { return t.description }
func (t *Task) Type() TaskType                   { return t.taskType }
func (t *Task) Metadata() map[string]interface{} { return t.metadata }
func (t *Task) SetMetadata(m map[string]interface{}) { t.metadata = m }
func (t *Task) Timeout() time.Duration           { return t.timeout }
func (t *Task) MaxRetries() int                  { return t.maxRetries }
func (t *Task) Priority() int                    { return t.priority }
func (t *Task) CreatedAt() time.Time             { return t.createdAt }

func (t *Task) Status() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

func (t *Task) Progress() Progress {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.progress
}

func (t *Task) Result() *Result {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.result
}

func (t *Task) Error() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.execErr
}

func (t *Task) StartedAt() *time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.startedAt
}

func (t *Task) FinishedAt() *time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.finishedAt
}

// canTransitionTo 检查状态转换是否合法
func (t *Task) canTransitionTo(target TaskStatus) bool {
	switch t.status {
	case TaskStatusPending:
		return target == TaskStatusRunning || target == TaskStatusCancelled
	case TaskStatusRunning:
		return target == TaskStatusCompleted || target == TaskStatusFailed || target == TaskStatusCancelled
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		return false
	default:
		return false
	}
}

// Start 开始任务
func (t *Task) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.canTransitionTo(TaskStatusRunning) {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	t.status = TaskStatusRunning
	t.startedAt = &now

	t.recordEvent(NewTaskStartedEvent(t))

	return nil
}

// Complete 完成任务
func (t *Task) Complete(result Result) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.canTransitionTo(TaskStatusCompleted) {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	t.status = TaskStatusCompleted
	t.finishedAt = &now
	t.result = &result

	t.recordEvent(NewTaskCompletedEvent(t))

	return nil
}

// Fail 任务失败
func (t *Task) Fail(err error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.canTransitionTo(TaskStatusFailed) {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	t.status = TaskStatusFailed
	t.finishedAt = &now
	t.execErr = err

	t.recordEvent(NewTaskFailedEvent(t))

	return nil
}

// Cancel 取消任务
func (t *Task) Cancel() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.canTransitionTo(TaskStatusCancelled) {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	t.status = TaskStatusCancelled
	t.finishedAt = &now

	t.recordEvent(NewTaskCancelledEvent(t))

	return nil
}

// UpdateProgress 更新进度
func (t *Task) UpdateProgress(total, current int, stage, detail string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.Update(total, current, stage, detail)
	t.recordEvent(NewTaskProgressUpdatedEvent(t, t.progress))
}

// recordEvent 记录领域事件（调用者需持有锁）
func (t *Task) recordEvent(event DomainEvent) {
	t.domainEvents = append(t.domainEvents, event)
}

// PopEvents 弹出所有领域事件
func (t *Task) PopEvents() []DomainEvent {
	t.mu.Lock()
	defer t.mu.Unlock()

	events := t.domainEvents
	t.domainEvents = nil
	return events
}

// ToSnapshot 转换为快照
func (t *Task) ToSnapshot() TaskSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return TaskSnapshot{
		ID:          t.id,
		TraceID:     t.traceID,
		SpanID:      t.spanID,
		ParentID:    t.parentID,
		Name:        t.name,
		Description: t.description,
		Type:        t.taskType,
		Metadata:    t.metadata,
		Timeout:     t.timeout,
		MaxRetries:  t.maxRetries,
		Priority:    t.priority,
		Status:      t.status,
		Progress:    t.progress,
		Result:      t.result,
		ErrorMsg:    "",
		CreatedAt:   t.createdAt,
		StartedAt:   t.startedAt,
		FinishedAt:  t.finishedAt,
	}
}

// FromSnapshot 从快照恢复
func (t *Task) FromSnapshot(snap *TaskSnapshot) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.id = snap.ID
	t.traceID = snap.TraceID
	t.spanID = snap.SpanID
	t.parentID = snap.ParentID
	t.name = snap.Name
	t.description = snap.Description
	t.taskType = snap.Type
	t.metadata = snap.Metadata
	t.timeout = snap.Timeout
	t.maxRetries = snap.MaxRetries
	t.priority = snap.Priority
	t.status = snap.Status
	t.progress = snap.Progress
	t.result = snap.Result
	t.execErr = nil
	t.createdAt = snap.CreatedAt
	t.startedAt = snap.StartedAt
	t.finishedAt = snap.FinishedAt
}
