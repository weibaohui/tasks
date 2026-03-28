/**
 * Task 聚合根
 * 是任务管理的核心实体，负责维护任务的完整生命周期
 */
package domain

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// 领域错误定义
var (
	ErrInvalidStatusTransition    = errors.New("invalid status transition")
	ErrTaskAlreadyStarted         = errors.New("task already started")
	ErrTaskNotRunning             = errors.New("task is not running")
	ErrTaskAlreadyFinished        = errors.New("task already finished")
	ErrTimeoutNotPositive         = errors.New("timeout must be positive")
	ErrTaskRequirementRequired    = errors.New("task requirement is required")
	ErrAcceptanceCriteriaRequired = errors.New("acceptance criteria is required")
	ErrTaskConclusionRequired     = errors.New("task conclusion is required to complete task")
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

	timeout    time.Duration
	maxRetries int
	priority   int

	status   TaskStatus
	progress Progress
	execErr  error

	// 独立字段（不再存储在 metadata 中）
	acceptanceCriteria string
	taskRequirement    string
	taskConclusion     string
	subtaskRecords     string // YAML: 子任务成对文档汇总（仅父任务使用）
	userCode           string
	agentCode          string
	channelCode        string
	sessionKey         string
	todoList           string // 待办列表
	analysis           string // Agent 分析结果

	// 任务层级和追踪字段
	depth      int    // 任务深度（从1开始）
	parentSpan string // 父任务的 span ID

	createdAt  time.Time
	startedAt  *time.Time
	finishedAt *time.Time

	domainEvents []DomainEvent

	mu sync.RWMutex
}

// TaskResultPair 子任务成对文档
// 包含任务要求、验收标准、任务名称和任务结论，用于父子任务结果汇总
type TaskResultPair struct {
	TaskName           string     `yaml:"task_name"`
	TaskRequirement    string     `yaml:"task_requirement"`
	AcceptanceCriteria string     `yaml:"acceptance_criteria"`
	TaskConclusion     string     `yaml:"task_conclusion"`
	CompletedAt        *time.Time `yaml:"completed_at,omitempty"`
	Status             TaskStatus `yaml:"status"`
}

// ToYAML 将 TaskResultPair 序列化为 YAML 格式
func (p *TaskResultPair) ToYAML() string {
	data, err := yaml.Marshal(p)
	if err != nil {
		return ""
	}
	return string(data)
}

// ParseTaskResultPairs 解析 YAML 格式的子任务成对文档
// records 是 YAML 格式的字符串，多个文档用 --- 分隔
func ParseTaskResultPairs(records string) ([]TaskResultPair, error) {
	if records == "" {
		return nil, nil
	}

	// 按 --- 分隔 YAML 文档
	docs := strings.Split(records, "---")
	var pairs []TaskResultPair

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" || strings.HasPrefix(doc, "#") {
			continue
		}
		var pair TaskResultPair
		if err := yaml.Unmarshal([]byte(doc), &pair); err != nil {
			return nil, fmt.Errorf("解析 TaskResultPair 失败: %w", err)
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// AppendTaskResultPair 将 TaskResultPair 添加到现有的 YAML 记录中
func AppendTaskResultPair(existingRecords string, pair TaskResultPair) (string, error) {
	pairYAML := pair.ToYAML()
	if pairYAML == "" {
		return existingRecords, nil
	}

	if existingRecords == "" {
		return pairYAML, nil
	}

	// 确保结尾有换行符
	existingRecords = strings.TrimRight(existingRecords, "\n")

	// 添加分隔符和新文档
	return existingRecords + "\n---\n" + pairYAML, nil
}

// NewTask 工厂方法：创建任务
// name: 任务名称
// taskRequirement: 任务目标/要求（必填）
// acceptanceCriteria: 验收标准（必填）
func NewTask(
	id TaskID,
	traceID TraceID,
	spanID SpanID,
	parentID *TaskID,
	name string,
	description string,
	taskType TaskType,
	taskRequirement string,
	acceptanceCriteria string,
	timeout time.Duration,
	maxRetries int,
	priority int,
) (*Task, error) {
	if name == "" {
		return nil, errors.New("task name is required")
	}
	if taskRequirement == "" {
		return nil, ErrTaskRequirementRequired
	}
	if acceptanceCriteria == "" {
		return nil, ErrAcceptanceCriteriaRequired
	}
	if timeout < 0 {
		return nil, ErrTimeoutNotPositive
	}

	task := &Task{
		id:                 id,
		traceID:            traceID,
		spanID:             spanID,
		parentID:           parentID,
		name:               name,
		description:        description,
		taskType:           taskType,
		taskRequirement:    taskRequirement,
		acceptanceCriteria: acceptanceCriteria,
		timeout:            timeout,
		maxRetries:         maxRetries,
		priority:           priority,
		status:             TaskStatusPending,
		progress:           NewProgress(),
		createdAt:          time.Now(),
	}

	task.recordEvent(NewTaskCreatedEvent(task))

	return task, nil
}

// 标识访问方法
func (t *Task) ID() TaskID          { return t.id }
func (t *Task) TraceID() TraceID    { return t.traceID }
func (t *Task) SpanID() SpanID      { return t.spanID }
func (t *Task) ParentID() *TaskID   { return t.parentID }
func (t *Task) Name() string        { return t.name }
func (t *Task) Description() string { return t.description }
func (t *Task) Type() TaskType      { return t.taskType }

// 独立字段访问方法
func (t *Task) AcceptanceCriteria() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.acceptanceCriteria
}
func (t *Task) SetAcceptanceCriteria(criteria string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.acceptanceCriteria = criteria
}

func (t *Task) TaskRequirement() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.taskRequirement
}
func (t *Task) SetTaskRequirement(requirement string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.taskRequirement = requirement
}

func (t *Task) TaskConclusion() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.taskConclusion
}
func (t *Task) SetTaskConclusion(conclusion string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.taskConclusion = conclusion
}

func (t *Task) SubtaskRecords() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.subtaskRecords
}
func (t *Task) SetSubtaskRecords(records string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subtaskRecords = records
}

func (t *Task) UserCode() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.userCode
}
func (t *Task) SetUserCode(code string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.userCode = code
}

func (t *Task) AgentCode() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.agentCode
}
func (t *Task) SetAgentCode(code string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.agentCode = code
}

func (t *Task) ChannelCode() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.channelCode
}
func (t *Task) SetChannelCode(code string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.channelCode = code
}

func (t *Task) SessionKey() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sessionKey
}
func (t *Task) SetSessionKey(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sessionKey = key
}

func (t *Task) TodoList() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.todoList
}
func (t *Task) SetTodoList(todoList string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.todoList = todoList
}

func (t *Task) Analysis() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.analysis
}
func (t *Task) SetAnalysis(analysis string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.analysis = analysis
}

func (t *Task) Depth() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.depth
}
func (t *Task) SetDepth(depth int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.depth = depth
}

func (t *Task) ParentSpan() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.parentSpan
}
func (t *Task) SetParentSpan(span string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.parentSpan = span
}

func (t *Task) Timeout() time.Duration { return t.timeout }
func (t *Task) MaxRetries() int        { return t.maxRetries }
func (t *Task) Priority() int          { return t.priority }
func (t *Task) CreatedAt() time.Time   { return t.createdAt }

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
		return target == TaskStatusCompleted || target == TaskStatusFailed || target == TaskStatusCancelled || target == TaskStatusPendingSummary
	case TaskStatusPendingSummary:
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

// Complete 完成任务（需要先设置任务结论）
func (t *Task) Complete() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.canTransitionTo(TaskStatusCompleted) {
		return ErrInvalidStatusTransition
	}

	// 验证任务结论必填
	if t.taskConclusion == "" {
		return ErrTaskConclusionRequired
	}

	now := time.Now()
	t.status = TaskStatusCompleted
	t.finishedAt = &now

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

// PendingSummary 进入等待总结状态（所有子任务完成，等待生成总结）
func (t *Task) PendingSummary() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.canTransitionTo(TaskStatusPendingSummary) {
		return ErrInvalidStatusTransition
	}

	t.status = TaskStatusPendingSummary
	t.recordEvent(NewTaskPendingSummaryEvent(t))

	return nil
}

// UpdateProgress 更新进度
func (t *Task) UpdateProgress(progress int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.Update(progress)
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
		ID:                 t.id,
		TraceID:            t.traceID,
		SpanID:             t.spanID,
		ParentID:           t.parentID,
		Name:               t.name,
		Description:        t.description,
		Type:               t.taskType,
		Timeout:            t.timeout,
		MaxRetries:         t.maxRetries,
		Priority:           t.priority,
		Status:             t.status,
		Progress:           t.progress,
		ErrorMsg:           t.getErrorMsg(),
		CreatedAt:          t.createdAt,
		StartedAt:          t.startedAt,
		FinishedAt:         t.finishedAt,
		AcceptanceCriteria: t.acceptanceCriteria,
		TaskRequirement:    t.taskRequirement,
		TaskConclusion:     t.taskConclusion,
		SubtaskRecords:     t.subtaskRecords,
		UserCode:           t.userCode,
		AgentCode:          t.agentCode,
		ChannelCode:        t.channelCode,
		SessionKey:         t.sessionKey,
		TodoList:           t.todoList,
		Analysis:           t.analysis,
		Depth:              t.depth,
		ParentSpan:         t.parentSpan,
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
	t.timeout = snap.Timeout
	t.maxRetries = snap.MaxRetries
	t.priority = snap.Priority
	t.status = snap.Status
	t.progress = snap.Progress
	t.execErr = t.errorFromMsg(snap.ErrorMsg)
	t.createdAt = snap.CreatedAt
	t.startedAt = snap.StartedAt
	t.finishedAt = snap.FinishedAt

	// 直接设置独立字段
	t.acceptanceCriteria = snap.AcceptanceCriteria
	t.taskRequirement = snap.TaskRequirement
	t.taskConclusion = snap.TaskConclusion
	t.subtaskRecords = snap.SubtaskRecords
	t.userCode = snap.UserCode
	t.agentCode = snap.AgentCode
	t.channelCode = snap.ChannelCode
	t.sessionKey = snap.SessionKey
	t.todoList = snap.TodoList
	t.analysis = snap.Analysis
	t.depth = snap.Depth
	t.parentSpan = snap.ParentSpan
}

// getErrorMsg 获取错误信息字符串
func (t *Task) getErrorMsg() string {
	if t.execErr != nil {
		return t.execErr.Error()
	}
	return ""
}

// errorFromMsg 从错误字符串恢复 error
func (t *Task) errorFromMsg(msg string) error {
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}
