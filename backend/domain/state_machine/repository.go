package state_machine

import "context"

// Repository 接口
type Repository interface {
	// 状态机
	SaveStateMachine(ctx context.Context, sm *StateMachine) error
	GetStateMachine(ctx context.Context, id string) (*StateMachine, error)
	ListStateMachines(ctx context.Context, projectID string) ([]*StateMachine, error)
	DeleteStateMachine(ctx context.Context, id string) error

	// 类型绑定
	SaveTypeBinding(ctx context.Context, binding *TypeBinding) error
	GetTypeBinding(ctx context.Context, stateMachineID, requirementType string) (*TypeBinding, error)
	DeleteTypeBinding(ctx context.Context, stateMachineID, requirementType string) error
	GetStateMachineByType(ctx context.Context, projectID, requirementType string) (*StateMachine, error)

	// 需求状态
	SaveRequirementState(ctx context.Context, rs *RequirementState) error
	GetRequirementState(ctx context.Context, requirementID string) (*RequirementState, error)
	UpdateRequirementState(ctx context.Context, rs *RequirementState) error

	// 转换日志
	SaveTransitionLog(ctx context.Context, log *TransitionLog) error
	ListTransitionLogs(ctx context.Context, requirementID string) ([]*TransitionLog, error)
}
