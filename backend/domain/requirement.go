package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrRequirementIDRequired        = errors.New("requirement id is required")
	ErrRequirementProjectIDRequired = errors.New("requirement project id is required")
	ErrRequirementTitleRequired     = errors.New("requirement title is required")
	ErrRequirementInvalidStatus     = errors.New("requirement status is invalid")
	ErrRequirementCannotDispatch    = errors.New("requirement cannot be dispatched in current state")
)

type RequirementID struct {
	value string
}

func NewRequirementID(value string) RequirementID {
	return RequirementID{value: value}
}

func (id RequirementID) String() string {
	return id.value
}

type RequirementStatus string

// 注意：状态值现在由状态机定义，不再硬编码。
// 这里只保留 todo 作为默认值，其他状态从状态机获取。
const RequirementStatusTodo RequirementStatus = "todo"

// IsValid 检查状态是否有效（兼容旧数据）
func (s RequirementStatus) IsValid() bool {
	// 任何非空字符串都是有效状态（状态由状态机定义）
	return s != ""
}

// Normalize 规范化状态值，将旧值转换为 todo
// 注意：其他状态不再自动转换，改为由状态机定义
func (s RequirementStatus) Normalize() RequirementStatus {
	switch s {
	case "in_progress", "doing":
		return RequirementStatus("preparing")
	case "preparing", "coding", "pr_opened", "failed", "completed", "done":
		// 这些旧状态保持不变（由状态机定义）
		return s
	case "": // 空字符串视为 todo
		return RequirementStatusTodo
	default:
		return s
	}
}

type RequirementType string

const (
	RequirementTypeNormal    RequirementType = "normal"
	RequirementTypeHeartbeat RequirementType = "heartbeat"
)

type Requirement struct {
	id                 RequirementID
	projectID          ProjectID
	title              string
	description        string
	acceptanceCriteria string
	tempWorkspaceRoot  string
	status             RequirementStatus
	previousStatus     RequirementStatus // 前一个状态，用于追踪状态转换历史
	assigneeAgentCode  string
	replicaAgentCode   string
	dispatchSessionKey string
	workspacePath      string
	lastError          string
	startedAt          *time.Time
	completedAt        *time.Time
	createdAt          time.Time
	updatedAt          time.Time
	// 需求类型：normal（普通需求，不自动触发）| heartbeat（心跳需求，自动触发）
	requirementType RequirementType
	// Claude Runtime 状态（持久化）
	claudeRuntimeStatus    string // running, completed, failed, ""
	claudeRuntimeStartedAt *time.Time
	claudeRuntimeEndedAt   *time.Time
	claudeRuntimeError     string
	claudeRuntimeResult    string // Claude Code 执行结果摘要
	claudeRuntimePrompt    string // Claude Code 执行提示词
	traceId                string // Claude Code 执行时的 trace_id，用于关联对话记录
	// Token 消耗统计（从对话记录计算）
	promptTokens     int
	completionTokens int
	totalTokens      int
}

// NewRedispatchedRequirement 创建重新派发的需求副本
// 标题会增加 "[重新派发]" 前缀
func NewRedispatchedRequirement(id RequirementID, original *Requirement) (*Requirement, error) {
	if original == nil {
		return nil, ErrRequirementProjectIDRequired
	}
	title := fmt.Sprintf("[重新派发] %s", original.Title())
	return NewRequirement(
		id,
		original.ProjectID(),
		title,
		original.Description(),
		original.AcceptanceCriteria(),
		original.TempWorkspaceRoot(),
	)
}

func NewRequirement(id RequirementID, projectID ProjectID, title, description, acceptanceCriteria, tempWorkspaceRoot string) (*Requirement, error) {
	if id.String() == "" {
		return nil, ErrRequirementIDRequired
	}
	if projectID.String() == "" {
		return nil, ErrRequirementProjectIDRequired
	}
	if strings.TrimSpace(title) == "" {
		return nil, ErrRequirementTitleRequired
	}
	now := time.Now()
	return &Requirement{
		id:                 id,
		projectID:          projectID,
		title:              title,
		description:        description,
		acceptanceCriteria: acceptanceCriteria,
		tempWorkspaceRoot:  strings.TrimSpace(tempWorkspaceRoot),
		status:             RequirementStatusTodo,
		requirementType:    RequirementTypeNormal,
		createdAt:          now,
		updatedAt:          now,
	}, nil
}

func (r *Requirement) ID() RequirementID                { return r.id }
func (r *Requirement) ProjectID() ProjectID             { return r.projectID }
func (r *Requirement) Title() string                    { return r.title }
func (r *Requirement) Description() string              { return r.description }
func (r *Requirement) AcceptanceCriteria() string       { return r.acceptanceCriteria }
func (r *Requirement) TempWorkspaceRoot() string        { return r.tempWorkspaceRoot }
func (r *Requirement) Status() RequirementStatus        { return r.status }
// PreviousStatus 返回前一个状态（用于追踪状态转换历史）
func (r *Requirement) PreviousStatus() RequirementStatus { return r.previousStatus }
func (r *Requirement) AssigneeAgentCode() string        { return r.assigneeAgentCode }
func (r *Requirement) ReplicaAgentCode() string         { return r.replicaAgentCode }
func (r *Requirement) DispatchSessionKey() string       { return r.dispatchSessionKey }
func (r *Requirement) WorkspacePath() string            { return r.workspacePath }
func (r *Requirement) LastError() string                { return r.lastError }
func (r *Requirement) StartedAt() *time.Time            { return copyTimePtr(r.startedAt) }
func (r *Requirement) CompletedAt() *time.Time          { return copyTimePtr(r.completedAt) }
func (r *Requirement) CreatedAt() time.Time             { return r.createdAt }
func (r *Requirement) UpdatedAt() time.Time             { return r.updatedAt }
func (r *Requirement) RequirementType() RequirementType { return r.requirementType }
func (r *Requirement) ClaudeRuntimeStatus() string      { return r.claudeRuntimeStatus }
func (r *Requirement) ClaudeRuntimeStartedAt() *time.Time {
	return copyTimePtr(r.claudeRuntimeStartedAt)
}
func (r *Requirement) ClaudeRuntimeEndedAt() *time.Time { return copyTimePtr(r.claudeRuntimeEndedAt) }
func (r *Requirement) ClaudeRuntimeError() string       { return r.claudeRuntimeError }
func (r *Requirement) ClaudeRuntimeResult() string      { return r.claudeRuntimeResult }
func (r *Requirement) ClaudeRuntimePrompt() string      { return r.claudeRuntimePrompt }

// SetClaudeRuntimeResult 设置 Claude Code 执行结果
func (r *Requirement) SetClaudeRuntimeResult(result string) {
	r.claudeRuntimeResult = result
}

// SetClaudeRuntimePrompt 设置 Claude Code 执行提示词
func (r *Requirement) SetClaudeRuntimePrompt(prompt string) {
	r.claudeRuntimePrompt = prompt
}

// SetClaudeRuntimeError 设置 Claude Code 错误信息
func (r *Requirement) SetClaudeRuntimeError(errMsg string) {
	r.claudeRuntimeError = errMsg
}

func (r *Requirement) TraceID() string { return r.traceId }

func (r *Requirement) SetTraceID(traceId string) {
	r.traceId = traceId
	r.updatedAt = time.Now()
}

func (r *Requirement) PromptTokens() int     { return r.promptTokens }
func (r *Requirement) CompletionTokens() int { return r.completionTokens }
func (r *Requirement) TotalTokens() int      { return r.totalTokens }

func (r *Requirement) SetTokenUsage(promptTokens, completionTokens, totalTokens int) {
	r.promptTokens = promptTokens
	r.completionTokens = completionTokens
	r.totalTokens = totalTokens
	r.updatedAt = time.Now()
}

// SetRequirementType 设置需求类型
func (r *Requirement) SetRequirementType(t RequirementType) {
	r.requirementType = t
}

// SyncStatusFromStateMachine 从状态机同步状态
// 这是状态机的值同步到需求的推荐方式
// 如果状态发生变化，会保存前一个状态到 previousStatus
func (r *Requirement) SyncStatusFromStateMachine(stateID string) {
	newStatus := RequirementStatus(stateID)
	if r.status != newStatus {
		r.previousStatus = r.status
		r.status = newStatus
		r.updatedAt = time.Now()
	}
}

func (r *Requirement) CanDispatch() bool {
	return r.status == RequirementStatusTodo
}

// CanRedispatch 检查是否可以重新派发
// 只要需求不是初始状态（todo），就可以重新派发
func (r *Requirement) CanRedispatch() bool {
	// 初始状态：todo -> 不需要重新派发
	// 其他状态都可以重新派发
	return r.status != RequirementStatusTodo
}

// StartClaudeRuntime 开始 Claude Runtime
func (r *Requirement) StartClaudeRuntime() {
	now := time.Now()
	r.claudeRuntimeStatus = "running"
	r.claudeRuntimeStartedAt = &now
	r.claudeRuntimeEndedAt = nil
	r.claudeRuntimeError = ""
	r.updatedAt = now
}

// EndClaudeRuntime 结束 Claude Runtime
func (r *Requirement) EndClaudeRuntime(success bool, errMsg string) {
	now := time.Now()
	if success {
		r.claudeRuntimeStatus = "completed"
	} else {
		r.claudeRuntimeStatus = "failed"
		r.claudeRuntimeError = errMsg
	}
	r.claudeRuntimeEndedAt = &now
	r.updatedAt = now
}

// Redispatch 重置需求状态，允许重新派发
// 注意：此方法直接设置状态，应使用状态机 TriggerTransition 替代
// 保留此方法用于向后兼容，新代码应使用状态机服务
// 如果状态发生变化，会保存前一个状态到 previousStatus
func (r *Requirement) Redispatch() error {
	if !r.CanRedispatch() {
		return ErrRequirementCannotDispatch
	}

	now := time.Now()
	if r.status != RequirementStatusTodo {
		r.previousStatus = r.status
	}
	r.status = RequirementStatusTodo
	r.assigneeAgentCode = ""
	r.replicaAgentCode = ""
	r.workspacePath = ""
	r.lastError = ""
	r.startedAt = nil
	r.completedAt = nil
	r.claudeRuntimePrompt = ""
	r.updatedAt = now

	return nil
}

func (r *Requirement) UpdateContent(title, description, acceptanceCriteria, tempWorkspaceRoot string) error {
	if strings.TrimSpace(title) == "" {
		return ErrRequirementTitleRequired
	}
	r.title = title
	r.description = description
	r.acceptanceCriteria = acceptanceCriteria
	r.tempWorkspaceRoot = strings.TrimSpace(tempWorkspaceRoot)
	r.updatedAt = time.Now()
	return nil
}

// StartDispatch 开始派发
// 注意：此方法直接设置状态，应使用状态机 TriggerTransition 替代
// 如果状态发生变化，会保存前一个状态到 previousStatus
func (r *Requirement) StartDispatch(assigneeAgentCode string) error {
	if !r.CanDispatch() {
		return ErrRequirementCannotDispatch
	}

	now := time.Now()
	newStatus := RequirementStatus("preparing")
	if r.status != newStatus {
		r.previousStatus = r.status
	}
	r.status = newStatus
	r.assigneeAgentCode = assigneeAgentCode
	r.startedAt = &now
	r.lastError = ""
	r.updatedAt = now

	return nil
}

// MarkCoding 标记编码中
// 注意：此方法直接设置状态，应使用状态机 TriggerTransition 替代
// 如果状态发生变化，会保存前一个状态到 previousStatus
func (r *Requirement) MarkCoding(workspacePath, replicaAgentCode string) error {
	if r.status != "preparing" {
		return ErrRequirementCannotDispatch
	}

	newStatus := RequirementStatus("coding")
	if r.status != newStatus {
		r.previousStatus = r.status
	}
	r.status = newStatus
	r.workspacePath = workspacePath
	r.replicaAgentCode = replicaAgentCode
	now := time.Now()
	r.updatedAt = now

	return nil
}

// MarkPROpened 标记 PR 已打开
// 注意：此方法直接设置状态，应使用状态机 TriggerTransition 替代
// 注意：清理分身和workspace应由调用方负责，此方法只负责状态变更
// 如果状态发生变化，会保存前一个状态到 previousStatus
func (r *Requirement) MarkPROpened() {
	now := time.Now()
	newStatus := RequirementStatus("pr_opened")
	if r.status != newStatus {
		r.previousStatus = r.status
	}
	r.status = newStatus
	r.lastError = ""
	r.completedAt = &now
	r.updatedAt = now
	r.replicaAgentCode = ""
	r.workspacePath = ""
}

// MarkFailed 标记失败
// 注意：此方法直接设置状态，应使用状态机 TriggerTransition 替代
// 注意：清理分身和workspace应由调用方负责，此方法只负责状态变更
// 如果状态发生变化，会保存前一个状态到 previousStatus
func (r *Requirement) MarkFailed(lastError string) {
	newStatus := RequirementStatus("failed")
	if r.status != newStatus {
		r.previousStatus = r.status
	}
	r.status = newStatus
	r.lastError = lastError
	now := time.Now()
	r.updatedAt = now
	r.replicaAgentCode = ""
	r.workspacePath = ""
}

// MarkCompleted 标记需求为已完成（Claude Code 正常结束）
// 注意：此方法直接设置状态，应使用状态机 TriggerTransition 替代
// 注意：清理分身和workspace应由调用方负责，此方法只负责状态变更
// 如果状态发生变化，会保存前一个状态到 previousStatus
func (r *Requirement) MarkCompleted() {
	newStatus := RequirementStatus("completed")
	if r.status != newStatus {
		r.previousStatus = r.status
	}
	r.status = newStatus
	now := time.Now()
	r.completedAt = &now
	r.updatedAt = now
	r.replicaAgentCode = ""
	r.workspacePath = ""
}

func (r *Requirement) SetDispatchSessionKey(sessionKey string) {
	r.dispatchSessionKey = strings.TrimSpace(sessionKey)
	r.updatedAt = time.Now()
}

func (r *Requirement) SetReplicaAgentCode(code string) {
	r.replicaAgentCode = code
	r.updatedAt = time.Now()
}

func (r *Requirement) SetWorkspacePath(path string) {
	r.workspacePath = path
	r.updatedAt = time.Now()
}

type RequirementSnapshot struct {
	ID                     RequirementID
	ProjectID              ProjectID
	Title                  string
	Description            string
	AcceptanceCriteria     string
	TempWorkspaceRoot      string
	Status                 RequirementStatus
	AssigneeAgentCode      string
	ReplicaAgentCode       string
	DispatchSessionKey     string
	WorkspacePath          string
	LastError              string
	StartedAt              *time.Time
	CompletedAt            *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	RequirementType        RequirementType
	ClaudeRuntimeStatus    string
	ClaudeRuntimeStartedAt *time.Time
	ClaudeRuntimeEndedAt   *time.Time
	ClaudeRuntimeError     string
	ClaudeRuntimeResult    string
	ClaudeRuntimePrompt    string
	TraceID                string
	PromptTokens           int
	CompletionTokens       int
	TotalTokens            int
}

func (r *Requirement) ToSnapshot() RequirementSnapshot {
	return RequirementSnapshot{
		ID:                     r.id,
		ProjectID:              r.projectID,
		Title:                  r.title,
		Description:            r.description,
		AcceptanceCriteria:     r.acceptanceCriteria,
		TempWorkspaceRoot:      r.tempWorkspaceRoot,
		Status:                 r.status,
		AssigneeAgentCode:      r.assigneeAgentCode,
		ReplicaAgentCode:       r.replicaAgentCode,
		DispatchSessionKey:     r.dispatchSessionKey,
		WorkspacePath:          r.workspacePath,
		LastError:              r.lastError,
		StartedAt:              copyTimePtr(r.startedAt),
		CompletedAt:            copyTimePtr(r.completedAt),
		CreatedAt:              r.createdAt,
		UpdatedAt:              r.updatedAt,
		RequirementType:        r.requirementType,
		ClaudeRuntimeStatus:    r.claudeRuntimeStatus,
		ClaudeRuntimeStartedAt: copyTimePtr(r.claudeRuntimeStartedAt),
		ClaudeRuntimeEndedAt:   copyTimePtr(r.claudeRuntimeEndedAt),
		ClaudeRuntimeError:     r.claudeRuntimeError,
		ClaudeRuntimeResult:    r.claudeRuntimeResult,
		ClaudeRuntimePrompt:    r.claudeRuntimePrompt,
		TraceID:                r.traceId,
		PromptTokens:           r.promptTokens,
		CompletionTokens:       r.completionTokens,
		TotalTokens:            r.totalTokens,
	}
}

func (r *Requirement) FromSnapshot(s RequirementSnapshot) error {
	if !s.Status.IsValid() {
		return ErrRequirementInvalidStatus
	}
	r.id = s.ID
	r.projectID = s.ProjectID
	r.title = s.Title
	r.description = s.Description
	r.acceptanceCriteria = s.AcceptanceCriteria
	r.tempWorkspaceRoot = strings.TrimSpace(s.TempWorkspaceRoot)
	r.status = s.Status.Normalize() // 规范化状态值，兼容旧数据
	r.assigneeAgentCode = s.AssigneeAgentCode
	r.replicaAgentCode = s.ReplicaAgentCode
	r.dispatchSessionKey = strings.TrimSpace(s.DispatchSessionKey)
	r.workspacePath = s.WorkspacePath
	r.lastError = s.LastError
	r.startedAt = copyTimePtr(s.StartedAt)
	r.completedAt = copyTimePtr(s.CompletedAt)
	r.createdAt = s.CreatedAt
	r.updatedAt = s.UpdatedAt
	r.requirementType = s.RequirementType
	r.claudeRuntimeStatus = s.ClaudeRuntimeStatus
	r.claudeRuntimeStartedAt = copyTimePtr(s.ClaudeRuntimeStartedAt)
	r.claudeRuntimeEndedAt = copyTimePtr(s.ClaudeRuntimeEndedAt)
	r.claudeRuntimeError = s.ClaudeRuntimeError
	r.claudeRuntimeResult = s.ClaudeRuntimeResult
	r.claudeRuntimePrompt = s.ClaudeRuntimePrompt
	r.traceId = s.TraceID
	r.promptTokens = s.PromptTokens
	r.completionTokens = s.CompletionTokens
	r.totalTokens = s.TotalTokens
	return nil
}

func copyTimePtr(input *time.Time) *time.Time {
	if input == nil {
		return nil
	}
	v := *input
	return &v
}
