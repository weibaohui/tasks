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
const (
	RequirementStatusTodo       RequirementStatus = "todo"
	RequirementStatusPreparing  RequirementStatus = "preparing"
	RequirementStatusCoding     RequirementStatus = "coding"
	RequirementStatusPROpened   RequirementStatus = "pr_opened"
	RequirementStatusFailed     RequirementStatus = "failed"
	RequirementStatusCompleted  RequirementStatus = "completed"
)

// Claude Runtime 状态常量
const (
	RuntimeStatusRunning   = "running"
	RuntimeStatusCompleted = "completed"
	RuntimeStatusFailed    = "failed"
)

// Normalize 规范化状态值，将旧值转换为 todo
// 注意：其他状态不再自动转换，改为由状态机定义
func (s RequirementStatus) Normalize() RequirementStatus {
	switch s {
	case "in_progress", "doing":
		return RequirementStatusPreparing
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
	// Agent Runtime 状态（持久化）
	agentRuntimeStatus    string // running, completed, failed, ""
	agentRuntimeStartedAt *time.Time
	agentRuntimeEndedAt   *time.Time
	agentRuntimeError     string
	agentRuntimeResult    string // Agent 执行结果摘要
	agentRuntimePrompt    string // Agent 执行提示词
	agentRuntimeAgentType string // 执行使用的 Agent 类型，如 CodingAgent / OpenCodeAgent
	traceId                string // Agent 执行时的 trace_id，用于关联对话记录
	// Token 消耗统计（从对话记录计算）
	promptTokens     int
	completionTokens int
	totalTokens      int
	// 进度数据（从对话记录中的 todo 工具提取）
	progressData string // JSON 格式存储的 ProgressData
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
func (r *Requirement) AgentRuntimeStatus() string      { return r.agentRuntimeStatus }
func (r *Requirement) AgentRuntimeStartedAt() *time.Time {
	return copyTimePtr(r.agentRuntimeStartedAt)
}
func (r *Requirement) AgentRuntimeEndedAt() *time.Time { return copyTimePtr(r.agentRuntimeEndedAt) }
func (r *Requirement) AgentRuntimeError() string       { return r.agentRuntimeError }
func (r *Requirement) AgentRuntimeResult() string      { return r.agentRuntimeResult }
func (r *Requirement) AgentRuntimePrompt() string      { return r.agentRuntimePrompt }
func (r *Requirement) AgentRuntimeAgentType() string   { return r.agentRuntimeAgentType }

// SetAgentRuntimeResult 设置 Agent 执行结果
func (r *Requirement) SetAgentRuntimeResult(result string) {
	r.agentRuntimeResult = result
}

// SetAgentRuntimePrompt 设置 Agent 执行提示词
func (r *Requirement) SetAgentRuntimePrompt(prompt string) {
	r.agentRuntimePrompt = prompt
}

// SetAgentRuntimeError 设置 Agent 错误信息
func (r *Requirement) SetAgentRuntimeError(errMsg string) {
	r.agentRuntimeError = errMsg
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

// ProgressData 返回进度数据（JSON 字符串）
func (r *Requirement) ProgressData() string {
	return r.progressData
}

// SetProgressData 设置进度数据（JSON 字符串）
func (r *Requirement) SetProgressData(data string) {
	r.progressData = data
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

// StartAgentRuntime 开始 Agent Runtime
func (r *Requirement) StartAgentRuntime(agentType string) {
	now := time.Now()
	r.agentRuntimeStatus = RuntimeStatusRunning
	r.agentRuntimeAgentType = agentType
	r.agentRuntimeStartedAt = &now
	r.agentRuntimeEndedAt = nil
	r.agentRuntimeError = ""
	r.updatedAt = now
}

// EndAgentRuntime 结束 Agent Runtime
func (r *Requirement) EndAgentRuntime(success bool, errMsg string) {
	now := time.Now()
	if success {
		r.agentRuntimeStatus = RuntimeStatusCompleted
	} else {
		r.agentRuntimeStatus = RuntimeStatusFailed
		r.agentRuntimeError = errMsg
	}
	r.agentRuntimeEndedAt = &now
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
	r.agentRuntimePrompt = ""
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
	newStatus := RequirementStatusPreparing
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
	if r.status != RequirementStatusPreparing {
		return ErrRequirementCannotDispatch
	}

	newStatus := RequirementStatusCoding
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
	newStatus := RequirementStatusPROpened
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

// StaleThreshold 需求超时判定阈值
const StaleThreshold = 30 * time.Minute

// StaleReplicaMissingThreshold 分身缺失时的较短超时阈值
const StaleReplicaMissingThreshold = 5 * time.Minute

// IsStale 判断处于 coding 状态的需求是否过期
// 返回 (shouldCleanup, reason)
func (r *Requirement) IsStale(now time.Time) (bool, string) {
	if r.status != RequirementStatusCoding {
		return false, ""
	}

	updatedAt := r.updatedAt
	if now.Sub(updatedAt) > StaleThreshold {
		return true, "timeout - no update for 30+ minutes"
	}
	return false, ""
}

// IsStaleWithReplicaCheck 在分身缺失时使用较短的阈值判定过期
// 需要调用方提供 replicaExists 信息
func (r *Requirement) IsStaleWithReplicaCheck(now time.Time, replicaExists bool) (bool, string) {
	if stale, reason := r.IsStale(now); stale {
		return true, reason
	}

	if r.replicaAgentCode != "" && !replicaExists {
		if now.Sub(r.updatedAt) > StaleReplicaMissingThreshold {
			return true, "replica agent missing - possible server crash during execution"
		}
	}
	return false, ""
}

// MarkFailed 标记失败
// 注意：此方法直接设置状态，应使用状态机 TriggerTransition 替代
// 注意：清理分身和workspace应由调用方负责，此方法只负责状态变更
// 如果状态发生变化，会保存前一个状态到 previousStatus
func (r *Requirement) MarkFailed(lastError string) {
	newStatus := RequirementStatusFailed
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
	newStatus := RequirementStatusCompleted
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
	AgentRuntimeStatus      string
	AgentRuntimeStartedAt   *time.Time
	AgentRuntimeEndedAt     *time.Time
	AgentRuntimeError       string
	AgentRuntimeResult      string
	AgentRuntimePrompt      string
	AgentRuntimeAgentType   string
	TraceID                 string
	PromptTokens           int
	CompletionTokens       int
	TotalTokens            int
	ProgressData           string
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
		AgentRuntimeStatus:      r.agentRuntimeStatus,
		AgentRuntimeStartedAt:   copyTimePtr(r.agentRuntimeStartedAt),
		AgentRuntimeEndedAt:     copyTimePtr(r.agentRuntimeEndedAt),
		AgentRuntimeError:       r.agentRuntimeError,
		AgentRuntimeResult:      r.agentRuntimeResult,
		AgentRuntimePrompt:      r.agentRuntimePrompt,
		AgentRuntimeAgentType:   r.agentRuntimeAgentType,
		TraceID:                 r.traceId,
		PromptTokens:           r.promptTokens,
		CompletionTokens:       r.completionTokens,
		TotalTokens:            r.totalTokens,
		ProgressData:           r.progressData,
	}
}

func (r *Requirement) FromSnapshot(s RequirementSnapshot) error {
	r.id = s.ID
	r.projectID = s.ProjectID
	r.title = s.Title
	r.description = s.Description
	r.acceptanceCriteria = s.AcceptanceCriteria
	r.tempWorkspaceRoot = strings.TrimSpace(s.TempWorkspaceRoot)
	r.status = s.Status.Normalize() // 规范化状态值，兼容旧数据（空字符串 -> todo）
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
	r.agentRuntimeStatus = s.AgentRuntimeStatus
	r.agentRuntimeStartedAt = copyTimePtr(s.AgentRuntimeStartedAt)
	r.agentRuntimeEndedAt = copyTimePtr(s.AgentRuntimeEndedAt)
	r.agentRuntimeError = s.AgentRuntimeError
	r.agentRuntimeResult = s.AgentRuntimeResult
	r.agentRuntimePrompt = s.AgentRuntimePrompt
	r.agentRuntimeAgentType = s.AgentRuntimeAgentType
	r.traceId = s.TraceID
	r.promptTokens = s.PromptTokens
	r.completionTokens = s.CompletionTokens
	r.totalTokens = s.TotalTokens
	r.progressData = s.ProgressData
	return nil
}

func copyTimePtr(input *time.Time) *time.Time {
	if input == nil {
		return nil
	}
	v := *input
	return &v
}
