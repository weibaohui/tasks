/**
 * 值对象定义
 * 包含 TaskID, TraceID, SpanID, TaskStatus, TaskType, Progress, Result 等
 */
package domain

import (
	"fmt"
	"time"
)

// TaskID 任务ID值对象
type TaskID struct {
	value string
}

func NewTaskID(value string) TaskID {
	return TaskID{value: value}
}

func (id TaskID) String() string {
	return id.value
}

func (id TaskID) Equals(other TaskID) bool {
	return id.value == other.value
}

// TraceID 追踪ID值对象
type TraceID struct {
	value string
}

func NewTraceID(value string) TraceID {
	return TraceID{value: value}
}

func (id TraceID) String() string {
	return id.value
}

func (id TraceID) Equals(other TraceID) bool {
	return id.value == other.value
}

// SpanID 跨度ID值对象
type SpanID struct {
	value string
}

func NewSpanID(value string) SpanID {
	return SpanID{value: value}
}

func (id SpanID) String() string {
	return id.value
}

func (id SpanID) Equals(other SpanID) bool {
	return id.value == other.value
}

// TaskStatus 任务状态枚举
type TaskStatus int

const (
	TaskStatusPending    TaskStatus = 0
	TaskStatusRunning    TaskStatus = 1
	TaskStatusCompleted  TaskStatus = 2
	TaskStatusFailed     TaskStatus = 3
	TaskStatusCancelled  TaskStatus = 4
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "pending"
	case TaskStatusRunning:
		return "running"
	case TaskStatusCompleted:
		return "completed"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// ParseTaskStatus 解析任务状态字符串
func ParseTaskStatus(s string) (TaskStatus, error) {
	switch s {
	case "pending":
		return TaskStatusPending, nil
	case "running":
		return TaskStatusRunning, nil
	case "completed":
		return TaskStatusCompleted, nil
	case "failed":
		return TaskStatusFailed, nil
	case "cancelled":
		return TaskStatusCancelled, nil
	default:
		return TaskStatusPending, fmt.Errorf("unknown status: %s", s)
	}
}

// TaskType 任务类型枚举
type TaskType int

const (
	TaskTypeDataProcessing TaskType = 0
	TaskTypeFileOperation  TaskType = 1
	TaskTypeAPICall        TaskType = 2
	TaskTypeCustom         TaskType = 3
)

func (t TaskType) String() string {
	switch t {
	case TaskTypeDataProcessing:
		return "data_processing"
	case TaskTypeFileOperation:
		return "file_operation"
	case TaskTypeAPICall:
		return "api_call"
	case TaskTypeCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// ParseTaskType 解析任务类型字符串
func ParseTaskType(s string) (TaskType, error) {
	switch s {
	case "data_processing":
		return TaskTypeDataProcessing, nil
	case "file_operation":
		return TaskTypeFileOperation, nil
	case "api_call":
		return TaskTypeAPICall, nil
	case "custom":
		return TaskTypeCustom, nil
	default:
		return TaskTypeCustom, fmt.Errorf("unknown type: %s", s)
	}
}

// Progress 进度值对象
type Progress struct {
	total      int
	current    int
	percentage float64
	stage      string
	detail     string
	updatedAt  time.Time
}

// NewProgress 创建进度对象
func NewProgress() Progress {
	return Progress{
		total:      0,
		current:    0,
		percentage: 0,
		stage:      "",
		detail:     "",
		updatedAt:  time.Now(),
	}
}

// Update 更新进度
func (p *Progress) Update(total, current int, stage, detail string) {
	p.total = total
	p.current = current
	if total > 0 {
		p.percentage = float64(current) / float64(total) * 100
	} else if current > 0 {
		p.percentage = 0
	}
	p.stage = stage
	p.detail = detail
	p.updatedAt = time.Now()
}

// Total 总数
func (p Progress) Total() int {
	return p.total
}

// Current 当前数
func (p Progress) Current() int {
	return p.current
}

// Percentage 百分比
func (p Progress) Percentage() float64 {
	return p.percentage
}

// Stage 阶段
func (p Progress) Stage() string {
	return p.stage
}

// Detail 详情
func (p Progress) Detail() string {
	return p.detail
}

// UpdatedAt 更新时间
func (p Progress) UpdatedAt() time.Time {
	return p.updatedAt
}

// ToMap 转换为map
func (p Progress) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"total":      p.total,
		"current":    p.current,
		"percentage": p.percentage,
		"stage":      p.stage,
		"detail":     p.detail,
		"updated_at": p.updatedAt.Unix(),
	}
}

// Result 结果值对象
type Result struct {
	data    interface{}
	message string
}

// NewResult 创建结果对象
func NewResult(data interface{}, message string) Result {
	return Result{
		data:    data,
		message: message,
	}
}

// Data 数据
func (r Result) Data() interface{} {
	return r.data
}

// Message 消息
func (r Result) Message() string {
	return r.message
}

// ToMap 转换为map
func (r Result) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"data":    r.data,
		"message": r.message,
	}
}

// TaskSnapshot 任务快照，用于持久化
type TaskSnapshot struct {
	ID          TaskID
	TraceID     TraceID
	SpanID      SpanID
	ParentID    *TaskID
	Name        string
	Description string
	Type        TaskType
	Metadata    map[string]interface{}
	Timeout     time.Duration
	MaxRetries  int
	Priority    int
	Status      TaskStatus
	Progress    Progress
	Result      *Result
	ErrorMsg    string
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}
