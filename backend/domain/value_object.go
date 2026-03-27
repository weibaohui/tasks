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
	TaskStatusPending   TaskStatus = 0
	TaskStatusRunning   TaskStatus = 1
	TaskStatusCompleted TaskStatus = 2
	TaskStatusFailed    TaskStatus = 3
	TaskStatusCancelled TaskStatus = 4
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
// 模式：agent（智能体）、coding（编码）、custom（自定义）
type TaskType int

const (
	TaskTypeAgent  TaskType = 0
	TaskTypeCoding TaskType = 1 // 编码模式（待实现）
	TaskTypeCustom TaskType = 2
)

func (t TaskType) String() string {
	switch t {
	case TaskTypeAgent:
		return "agent"
	case TaskTypeCoding:
		return "coding"
	case TaskTypeCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// ParseTaskType 解析任务类型字符串
func ParseTaskType(s string) (TaskType, error) {
	switch s {
	case "agent":
		return TaskTypeAgent, nil
	case "coding":
		return TaskTypeCoding, nil
	case "custom":
		return TaskTypeCustom, nil
	default:
		return TaskTypeCustom, fmt.Errorf("unknown type: %s", s)
	}
}

// Progress 进度值对象
type Progress struct {
	value     int
	updatedAt time.Time
}

// NewProgress 创建进度对象
func NewProgress() Progress {
	return Progress{
		value:     0,
		updatedAt: time.Now(),
	}
}

// Update 更新进度
func (p *Progress) Update(value int) {
	p.value = value
	p.updatedAt = time.Now()
}

// Value 获取进度值
func (p Progress) Value() int {
	return p.value
}

// UpdatedAt 更新时间
func (p Progress) UpdatedAt() time.Time {
	return p.updatedAt
}

// ToMap 转换为map
func (p Progress) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"value":      p.value,
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
	ID                 TaskID
	TraceID            TraceID
	SpanID             SpanID
	ParentID           *TaskID
	Name               string
	Description        string
	Type               TaskType
	AcceptanceCriteria string
	TaskRequirement    string
	TaskConclusion     string
	UserCode           string
	AgentCode          string
	ChannelCode        string
	SessionKey         string
	ExecutionSummary   map[string]interface{} // 执行摘要
	TodoList           string                 // 待办列表
	Analysis           string                 // Agent 分析结果
	Depth              int                    // 任务深度
	ParentSpan         string                 // 父任务的 span ID
	Timeout            time.Duration
	MaxRetries         int
	Priority           int
	Status             TaskStatus
	Progress           Progress
	Result             *Result
	ErrorMsg           string
	CreatedAt          time.Time
	StartedAt          *time.Time
	FinishedAt         *time.Time
}
