package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrRequirementTypeIDRequired   = errors.New("requirement type id is required")
	ErrRequirementTypeNameRequired = errors.New("requirement type name is required")
	ErrRequirementTypeCodeInvalid  = errors.New("requirement type code is invalid")
)

type RequirementTypeEntityID struct {
	value string
}

func NewRequirementTypeEntityID(value string) RequirementTypeEntityID {
	return RequirementTypeEntityID{value: value}
}

func (id RequirementTypeEntityID) String() string {
	return id.value
}

// RequirementTypeEntity 需求类型实体
type RequirementTypeEntity struct {
	id          RequirementTypeEntityID
	projectID   ProjectID
	code        string           // 类型代码，如 "normal", "heartbeat", "pr_review", "optimization"
	name        string           // 类型名称
	description string           // 类型描述
	icon        string           // 图标
	color       string           // 颜色
	sortOrder   int              // 排序
	stateMachineID string        // 绑定的状态机ID（可选）
	isSystem    bool             // 是否为系统内置类型，不可删除
	createdAt   time.Time
	updatedAt   time.Time
}

func NewRequirementTypeEntity(id RequirementTypeEntityID, projectID ProjectID, code, name, description string) (*RequirementTypeEntity, error) {
	if id.String() == "" {
		return nil, ErrRequirementTypeIDRequired
	}
	if projectID.String() == "" {
		return nil, ErrRequirementProjectIDRequired
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrRequirementTypeCodeInvalid
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrRequirementTypeNameRequired
	}

	now := time.Now()
	return &RequirementTypeEntity{
		id:          id,
		projectID:   projectID,
		code:        strings.ToLower(code),
		name:        name,
		description: strings.TrimSpace(description),
		sortOrder:   0,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func (rt *RequirementTypeEntity) ID() RequirementTypeEntityID      { return rt.id }
func (rt *RequirementTypeEntity) ProjectID() ProjectID        { return rt.projectID }
func (rt *RequirementTypeEntity) Code() string               { return rt.code }
func (rt *RequirementTypeEntity) Name() string               { return rt.name }
func (rt *RequirementTypeEntity) Description() string         { return rt.description }
func (rt *RequirementTypeEntity) Icon() string               { return rt.icon }
func (rt *RequirementTypeEntity) Color() string              { return rt.color }
func (rt *RequirementTypeEntity) SortOrder() int             { return rt.sortOrder }
func (rt *RequirementTypeEntity) StateMachineID() string     { return rt.stateMachineID }
func (rt *RequirementTypeEntity) CreatedAt() time.Time       { return rt.createdAt }
func (rt *RequirementTypeEntity) UpdatedAt() time.Time       { return rt.updatedAt }

func (rt *RequirementTypeEntity) SetDescription(desc string) {
	rt.description = strings.TrimSpace(desc)
	rt.updatedAt = time.Now()
}

func (rt *RequirementTypeEntity) SetIcon(icon string) {
	rt.icon = strings.TrimSpace(icon)
	rt.updatedAt = time.Now()
}

func (rt *RequirementTypeEntity) SetColor(color string) {
	rt.color = strings.TrimSpace(color)
	rt.updatedAt = time.Now()
}

func (rt *RequirementTypeEntity) SetSortOrder(order int) {
	rt.sortOrder = order
	rt.updatedAt = time.Now()
}

func (rt *RequirementTypeEntity) SetStateMachineID(smID string) {
	rt.stateMachineID = strings.TrimSpace(smID)
	rt.updatedAt = time.Now()
}

func (rt *RequirementTypeEntity) IsSystem() bool {
	return rt.isSystem
}

func (rt *RequirementTypeEntity) SetIsSystem(v bool) {
	rt.isSystem = v
	rt.updatedAt = time.Now()
}

// RequirementTypeEntitySnapshot 需求类型快照
type RequirementTypeEntitySnapshot struct {
	ID             RequirementTypeEntityID
	ProjectID      ProjectID
	Code           string
	Name           string
	Description    string
	Icon           string
	Color          string
	SortOrder      int
	StateMachineID string
	IsSystem       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (rt *RequirementTypeEntity) ToSnapshot() RequirementTypeEntitySnapshot {
	return RequirementTypeEntitySnapshot{
		ID:             rt.id,
		ProjectID:      rt.projectID,
		Code:           rt.code,
		Name:           rt.name,
		Description:    rt.description,
		Icon:           rt.icon,
		Color:          rt.color,
		SortOrder:      rt.sortOrder,
		StateMachineID: rt.stateMachineID,
		IsSystem:       rt.isSystem,
		CreatedAt:      rt.createdAt,
		UpdatedAt:      rt.updatedAt,
	}
}

func (rt *RequirementTypeEntity) FromSnapshot(s RequirementTypeEntitySnapshot) {
	rt.id = s.ID
	rt.projectID = s.ProjectID
	rt.code = s.Code
	rt.name = s.Name
	rt.description = s.Description
	rt.icon = s.Icon
	rt.color = s.Color
	rt.sortOrder = s.SortOrder
	rt.stateMachineID = s.StateMachineID
	rt.isSystem = s.IsSystem
	rt.createdAt = s.CreatedAt
	rt.updatedAt = s.UpdatedAt
}

// RequirementTypeEntityRepository 需求类型仓储接口
type RequirementTypeEntityRepository interface {
	FindByID(ctx context.Context, id RequirementTypeEntityID) (*RequirementTypeEntity, error)
	FindByProjectID(ctx context.Context, projectID ProjectID) ([]*RequirementTypeEntity, error)
	FindByCode(ctx context.Context, projectID ProjectID, code string) (*RequirementTypeEntity, error)
	Save(ctx context.Context, rt *RequirementTypeEntity) error
	Delete(ctx context.Context, id RequirementTypeEntityID) error
}