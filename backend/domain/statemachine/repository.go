package statemachine

import "context"

// Repository 接口
type Repository interface {
	// 状态机
	SaveStateMachine(ctx context.Context, sm *StateMachine) error
	GetStateMachine(ctx context.Context, id string) (*StateMachine, error)
	ListStateMachines(ctx context.Context) ([]*StateMachine, error)
	DeleteStateMachine(ctx context.Context, id string) error

	// 需求状态
	SaveRequirementState(ctx context.Context, rs *RequirementState) error
	GetRequirementState(ctx context.Context, requirementID string) (*RequirementState, error)
	UpdateRequirementState(ctx context.Context, rs *RequirementState) error

	// 转换日志
	SaveTransitionLog(ctx context.Context, log *TransitionLog) error
	ListTransitionLogs(ctx context.Context, requirementID string) ([]*TransitionLog, error)

	// 项目状态机关联
	SaveProjectStateMachine(ctx context.Context, psm *ProjectStateMachine) error
	GetProjectStateMachine(ctx context.Context, projectID string, requirementType RequirementType) (*ProjectStateMachine, error)
	ListProjectStateMachines(ctx context.Context, projectID string) ([]*ProjectStateMachine, error)
	DeleteProjectStateMachine(ctx context.Context, id string) error
	DeleteProjectStateMachinesByProject(ctx context.Context, projectID string) error
}
