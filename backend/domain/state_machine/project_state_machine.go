package state_machine

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrProjectStateMachineIDRequired     = errors.New("project state machine id is required")
	ErrProjectIDRequired                 = errors.New("project id is required")
	ErrStateMachineIDRequired            = errors.New("state machine id is required")
	ErrRequirementTypeRequired           = errors.New("requirement type is required")
	ErrInvalidRequirementType            = errors.New("invalid requirement type")
	ErrDuplicateRequirementTypeMapping   = errors.New("duplicate requirement type mapping")
)

// RequirementType 需求类型
// 与 domain.RequirementType 保持一致
type RequirementType string

const (
	RequirementTypeNormal    RequirementType = "normal"
	RequirementTypeHeartbeat RequirementType = "heartbeat"
)

// IsValidRequirementType 检查需求类型是否有效
// 注意：需求类型现在通过 requirement_types 表动态管理，不再硬编码验证
// 此函数保留用于向后兼容，任何非空字符串都视为有效类型
func IsValidRequirementType(t string) bool {
	return t != ""
}

// ProjectStateMachine 项目状态机关联
// 记录项目与状态机之间的关联关系，按需求类型区分
type ProjectStateMachine struct {
	id               string
	projectID        string
	requirementType  RequirementType
	stateMachineID   string
	createdAt        time.Time
	updatedAt        time.Time
}

// NewProjectStateMachine 创建项目状态机关联
func NewProjectStateMachine(projectID string, requirementType RequirementType, stateMachineID string) (*ProjectStateMachine, error) {
	if projectID == "" {
		return nil, ErrProjectIDRequired
	}
	if stateMachineID == "" {
		return nil, ErrStateMachineIDRequired
	}
	if requirementType == "" {
		return nil, ErrRequirementTypeRequired
	}
	if !IsValidRequirementType(string(requirementType)) {
		return nil, ErrInvalidRequirementType
	}

	now := time.Now()
	return &ProjectStateMachine{
		id:              uuid.New().String(),
		projectID:       projectID,
		requirementType: requirementType,
		stateMachineID:  stateMachineID,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// ID 返回关联ID
func (p *ProjectStateMachine) ID() string {
	return p.id
}

// ProjectID 返回项目ID
func (p *ProjectStateMachine) ProjectID() string {
	return p.projectID
}

// RequirementType 返回需求类型
func (p *ProjectStateMachine) RequirementType() RequirementType {
	return p.requirementType
}

// StateMachineID 返回状态机ID
func (p *ProjectStateMachine) StateMachineID() string {
	return p.stateMachineID
}

// CreatedAt 返回创建时间
func (p *ProjectStateMachine) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt 返回更新时间
func (p *ProjectStateMachine) UpdatedAt() time.Time {
	return p.updatedAt
}

// UpdateStateMachine 更新关联的状态机
func (p *ProjectStateMachine) UpdateStateMachine(stateMachineID string) error {
	if stateMachineID == "" {
		return ErrStateMachineIDRequired
	}
	p.stateMachineID = stateMachineID
	p.updatedAt = time.Now()
	return nil
}

// ProjectStateMachineSnapshot 项目状态机关联快照
// 用于序列化和持久化
type ProjectStateMachineSnapshot struct {
	ID              string
	ProjectID       string
	RequirementType RequirementType
	StateMachineID  string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ToSnapshot 转换为快照
func (p *ProjectStateMachine) ToSnapshot() ProjectStateMachineSnapshot {
	return ProjectStateMachineSnapshot{
		ID:              p.id,
		ProjectID:       p.projectID,
		RequirementType: p.requirementType,
		StateMachineID:  p.stateMachineID,
		CreatedAt:       p.createdAt,
		UpdatedAt:       p.updatedAt,
	}
}

// FromSnapshot 从快照恢复
func (p *ProjectStateMachine) FromSnapshot(s ProjectStateMachineSnapshot) {
	p.id = s.ID
	p.projectID = s.ProjectID
	p.requirementType = s.RequirementType
	p.stateMachineID = s.StateMachineID
	p.createdAt = s.CreatedAt
	p.updatedAt = s.UpdatedAt
}
