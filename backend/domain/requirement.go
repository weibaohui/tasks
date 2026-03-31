package domain

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var (
	ErrRequirementIDRequired        = errors.New("requirement id is required")
	ErrRequirementProjectIDRequired = errors.New("requirement project id is required")
	ErrRequirementTitleRequired     = errors.New("requirement title is required")
	ErrRequirementInvalidStatus     = errors.New("requirement status is invalid")
	ErrRequirementInvalidDevState   = errors.New("requirement dev state is invalid")
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

const (
	RequirementStatusTodo       RequirementStatus = "todo"
	RequirementStatusInProgress RequirementStatus = "in_progress"
	RequirementStatusDone       RequirementStatus = "done"
)

func (s RequirementStatus) IsValid() bool {
	switch s {
	case RequirementStatusTodo, RequirementStatusInProgress, RequirementStatusDone:
		return true
	default:
		return false
	}
}

type RequirementDevState string

const (
	RequirementDevStateIdle      RequirementDevState = "idle"
	RequirementDevStatePreparing RequirementDevState = "preparing"
	RequirementDevStateCoding    RequirementDevState = "coding"
	RequirementDevStatePROpened  RequirementDevState = "pr_opened"
	RequirementDevStateFailed    RequirementDevState = "failed"
)

func (s RequirementDevState) IsValid() bool {
	switch s {
	case RequirementDevStateIdle, RequirementDevStatePreparing, RequirementDevStateCoding, RequirementDevStatePROpened, RequirementDevStateFailed:
		return true
	default:
		return false
	}
}

type Requirement struct {
	id                 RequirementID
	projectID          ProjectID
	title              string
	description        string
	acceptanceCriteria string
	tempWorkspaceRoot  string
	status             RequirementStatus
	devState           RequirementDevState
	assigneeAgentID    string
	replicaAgentID     string
	dispatchSessionKey string
	workspacePath      string
	branchName         string
	prURL              string
	lastError          string
	startedAt          *time.Time
	completedAt        *time.Time
	createdAt          time.Time
	updatedAt          time.Time

	// stateChangeCallbacks 状态变更回调列表（不持久化）
	stateChangeCallbacks []StateChangeCallback

	// replicaAgentManager 分身管理器（不持久化）
	// 通过 SetReplicaAgentManager 设置
	replicaAgentManager *ReplicaAgentManager
}

// ReplicaAgentManager 分身管理器
// 负责强制销毁分身，这是代码约束而非 Hook
type ReplicaAgentManager struct {
	agentRepo AgentRepository
}

// NewReplicaAgentManager 创建分身管理器
func NewReplicaAgentManager(agentRepo AgentRepository) *ReplicaAgentManager {
	return &ReplicaAgentManager{agentRepo: agentRepo}
}

// EnsureDisposed 确保分身已销毁（幂等方法）
// 这是一个幂等操作，调用多次和调用一次效果相同
func (m *ReplicaAgentManager) EnsureDisposed(ctx context.Context, replicaAgentID, workspacePath string) {
	if replicaAgentID == "" {
		return
	}

	// 1. 删除分身 Agent
	if err := m.agentRepo.Delete(ctx, NewAgentID(replicaAgentID)); err != nil {
		log.Printf("failed to delete replica agent %s: %v", replicaAgentID, err)
	} else {
		log.Printf("replica agent %s disposed", replicaAgentID)
	}

	// 2. 清理工作目录
	if workspacePath != "" {
		if err := os.RemoveAll(workspacePath); err != nil {
			log.Printf("failed to cleanup workspace %s: %v", workspacePath, err)
		} else {
			log.Printf("workspace %s cleaned", workspacePath)
		}
	}
}

// SetReplicaAgentManager 设置分身管理器
func (r *Requirement) SetReplicaAgentManager(manager *ReplicaAgentManager) {
	r.replicaAgentManager = manager
}

// StateChangeCallback 状态变更回调函数类型
type StateChangeCallback func(change *StateChange)

// SetStateChangeCallback 设置状态变更回调
func (r *Requirement) SetStateChangeCallback(cb StateChangeCallback) {
	r.stateChangeCallbacks = append(r.stateChangeCallbacks, cb)
}

// ClearStateChangeCallbacks 清除所有回调
func (r *Requirement) ClearStateChangeCallbacks() {
	r.stateChangeCallbacks = nil
}

// fireStateChange 触发状态变更回调
func (r *Requirement) fireStateChange(change *StateChange) {
	fmt.Printf("[DEBUG] fireStateChange: trigger=%s, callbacks=%d\n", change.Trigger, len(r.stateChangeCallbacks))
	for _, cb := range r.stateChangeCallbacks {
		cb(change)
	}
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
		devState:           RequirementDevStateIdle,
		createdAt:          now,
		updatedAt:          now,
	}, nil
}

func (r *Requirement) ID() RequirementID             { return r.id }
func (r *Requirement) ProjectID() ProjectID          { return r.projectID }
func (r *Requirement) Title() string                 { return r.title }
func (r *Requirement) Description() string           { return r.description }
func (r *Requirement) AcceptanceCriteria() string    { return r.acceptanceCriteria }
func (r *Requirement) TempWorkspaceRoot() string     { return r.tempWorkspaceRoot }
func (r *Requirement) Status() RequirementStatus     { return r.status }
func (r *Requirement) DevState() RequirementDevState { return r.devState }
func (r *Requirement) AssigneeAgentID() string       { return r.assigneeAgentID }
func (r *Requirement) ReplicaAgentID() string        { return r.replicaAgentID }
func (r *Requirement) DispatchSessionKey() string    { return r.dispatchSessionKey }
func (r *Requirement) WorkspacePath() string         { return r.workspacePath }
func (r *Requirement) BranchName() string            { return r.branchName }
func (r *Requirement) PRURL() string                 { return r.prURL }
func (r *Requirement) LastError() string             { return r.lastError }
func (r *Requirement) StartedAt() *time.Time         { return copyTimePtr(r.startedAt) }
func (r *Requirement) CompletedAt() *time.Time       { return copyTimePtr(r.completedAt) }
func (r *Requirement) CreatedAt() time.Time          { return r.createdAt }
func (r *Requirement) UpdatedAt() time.Time          { return r.updatedAt }
func (r *Requirement) CanDispatch() bool {
	return r.status == RequirementStatusTodo && r.devState == RequirementDevStateIdle
}

// CanRedispatch 检查是否可以重新派发
// 只要需求不是初始状态（todo + idle），就可以重新派发
func (r *Requirement) CanRedispatch() bool {
	// 初始状态：todo + idle -> 不需要重新派发
	// 其他状态都可以重新派发
	return !(r.status == RequirementStatusTodo && r.devState == RequirementDevStateIdle)
}

// Redispatch 重置需求状态，允许重新派发
func (r *Requirement) Redispatch() error {
	if !r.CanRedispatch() {
		return ErrRequirementCannotDispatch
	}

	fromStatus := r.status
	fromDevState := r.devState

	now := time.Now()
	r.status = RequirementStatusTodo
	r.devState = RequirementDevStateIdle
	r.assigneeAgentID = ""
	r.replicaAgentID = ""
	r.workspacePath = ""
	r.branchName = ""
	r.prURL = ""
	r.lastError = ""
	r.startedAt = nil
	r.completedAt = nil
	r.updatedAt = now

	r.fireStateChange(&StateChange{
		FromStatus:   fromStatus,
		ToStatus:     r.status,
		FromDevState: fromDevState,
		ToDevState:   r.devState,
		Trigger:      "redispatch",
		Reason:       "manual redispatch",
		Timestamp:    now,
	})

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

func (r *Requirement) StartDispatch(assigneeAgentID string) error {
	if !r.CanDispatch() {
		return ErrRequirementCannotDispatch
	}

	fromStatus := r.status
	fromDevState := r.devState

	now := time.Now()
	r.status = RequirementStatusInProgress
	r.devState = RequirementDevStatePreparing
	r.assigneeAgentID = assigneeAgentID
	r.startedAt = &now
	r.lastError = ""
	r.updatedAt = now

	r.fireStateChange(&StateChange{
		FromStatus:   fromStatus,
		ToStatus:     r.status,
		FromDevState: fromDevState,
		ToDevState:   r.devState,
		Trigger:      "start_dispatch",
		Reason:       "",
		Timestamp:    now,
	})

	return nil
}

func (r *Requirement) MarkCoding(workspacePath, replicaAgentID, branchName string) error {
	if r.status != RequirementStatusInProgress {
		return ErrRequirementCannotDispatch
	}

	fromDevState := r.devState

	r.devState = RequirementDevStateCoding
	r.workspacePath = workspacePath
	r.replicaAgentID = replicaAgentID
	r.branchName = branchName
	now := time.Now()
	r.updatedAt = now

	r.fireStateChange(&StateChange{
		FromStatus:   r.status,
		ToStatus:     r.status,
		FromDevState: fromDevState,
		ToDevState:   r.devState,
		Trigger:      "mark_coding",
		Reason:       "",
		Timestamp:    now,
	})

	return nil
}

func (r *Requirement) MarkPROpened(prURL, branchName string) {
	fromStatus := r.status
	fromDevState := r.devState

	now := time.Now()
	r.status = RequirementStatusDone
	r.devState = RequirementDevStatePROpened
	r.prURL = prURL
	if branchName != "" {
		r.branchName = branchName
	}
	r.lastError = ""
	r.completedAt = &now
	r.updatedAt = now

	// Claude Code 结束 - 成功
	r.fireStateChange(&StateChange{
		FromStatus:   fromStatus,
		ToStatus:     r.status,
		FromDevState: fromDevState,
		ToDevState:   r.devState,
		Trigger:      "claude_code_finished",
		Reason:       "",
		Timestamp:    now,
	})

	r.fireStateChange(&StateChange{
		FromStatus:   fromStatus,
		ToStatus:     r.status,
		FromDevState: fromDevState,
		ToDevState:   r.devState,
		Trigger:      "mark_pr_opened",
		Reason:       "",
		Timestamp:    now,
	})

	// 强制销毁分身（代码约束）
	if r.replicaAgentManager != nil {
		r.replicaAgentManager.EnsureDisposed(context.Background(), r.replicaAgentID, r.workspacePath)
	}
}

func (r *Requirement) MarkFailed(lastError string) {
	fromStatus := r.status
	fromDevState := r.devState

	r.status = RequirementStatusInProgress
	r.devState = RequirementDevStateFailed
	r.lastError = lastError
	now := time.Now()
	r.updatedAt = now

	// Claude Code 结束 - 失败
	r.fireStateChange(&StateChange{
		FromStatus:   fromStatus,
		ToStatus:     r.status,
		FromDevState: fromDevState,
		ToDevState:   r.devState,
		Trigger:      "claude_code_finished",
		Reason:       lastError,
		Timestamp:    now,
	})

	r.fireStateChange(&StateChange{
		FromStatus:   fromStatus,
		ToStatus:     r.status,
		FromDevState: fromDevState,
		ToDevState:   r.devState,
		Trigger:      "mark_failed",
		Reason:       lastError,
		Timestamp:    now,
	})

	// 强制销毁分身（代码约束）
	if r.replicaAgentManager != nil {
		r.replicaAgentManager.EnsureDisposed(context.Background(), r.replicaAgentID, r.workspacePath)
	}
}

func (r *Requirement) SetDispatchSessionKey(sessionKey string) {
	r.dispatchSessionKey = strings.TrimSpace(sessionKey)
	r.updatedAt = time.Now()
}

type RequirementSnapshot struct {
	ID                 RequirementID
	ProjectID          ProjectID
	Title              string
	Description        string
	AcceptanceCriteria string
	TempWorkspaceRoot  string
	Status             RequirementStatus
	DevState           RequirementDevState
	AssigneeAgentID    string
	ReplicaAgentID     string
	DispatchSessionKey string
	WorkspacePath      string
	BranchName         string
	PRURL              string
	LastError          string
	StartedAt          *time.Time
	CompletedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (r *Requirement) ToSnapshot() RequirementSnapshot {
	return RequirementSnapshot{
		ID:                 r.id,
		ProjectID:          r.projectID,
		Title:              r.title,
		Description:        r.description,
		AcceptanceCriteria: r.acceptanceCriteria,
		TempWorkspaceRoot:  r.tempWorkspaceRoot,
		Status:             r.status,
		DevState:           r.devState,
		AssigneeAgentID:    r.assigneeAgentID,
		ReplicaAgentID:     r.replicaAgentID,
		DispatchSessionKey: r.dispatchSessionKey,
		WorkspacePath:      r.workspacePath,
		BranchName:         r.branchName,
		PRURL:              r.prURL,
		LastError:          r.lastError,
		StartedAt:          copyTimePtr(r.startedAt),
		CompletedAt:        copyTimePtr(r.completedAt),
		CreatedAt:          r.createdAt,
		UpdatedAt:          r.updatedAt,
	}
}

func (r *Requirement) FromSnapshot(s RequirementSnapshot) error {
	if !s.Status.IsValid() {
		return ErrRequirementInvalidStatus
	}
	if !s.DevState.IsValid() {
		return ErrRequirementInvalidDevState
	}
	r.id = s.ID
	r.projectID = s.ProjectID
	r.title = s.Title
	r.description = s.Description
	r.acceptanceCriteria = s.AcceptanceCriteria
	r.tempWorkspaceRoot = strings.TrimSpace(s.TempWorkspaceRoot)
	r.status = s.Status
	r.devState = s.DevState
	r.assigneeAgentID = s.AssigneeAgentID
	r.replicaAgentID = s.ReplicaAgentID
	r.dispatchSessionKey = strings.TrimSpace(s.DispatchSessionKey)
	r.workspacePath = s.WorkspacePath
	r.branchName = s.BranchName
	r.prURL = s.PRURL
	r.lastError = s.LastError
	r.startedAt = copyTimePtr(s.StartedAt)
	r.completedAt = copyTimePtr(s.CompletedAt)
	r.createdAt = s.CreatedAt
	r.updatedAt = s.UpdatedAt
	return nil
}

func copyTimePtr(input *time.Time) *time.Time {
	if input == nil {
		return nil
	}
	v := *input
	return &v
}
