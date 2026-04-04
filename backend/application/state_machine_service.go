package application

import (
	"context"

	"github.com/weibh/taskmanager/domain/state_machine"
	infra_sm "github.com/weibh/taskmanager/infrastructure/state_machine"
	"go.uber.org/zap"
)

// StateMachineService 应用服务
type StateMachineService struct {
	repo      state_machine.Repository
	executor  *infra_sm.TransitionExecutor
	logger    *zap.Logger
}

// NewStateMachineService 创建服务
func NewStateMachineService(repo state_machine.Repository, executor *infra_sm.TransitionExecutor, logger *zap.Logger) *StateMachineService {
	return &StateMachineService{
		repo:     repo,
		executor: executor,
		logger:   logger,
	}
}

// CreateStateMachine 创建状态机
func (s *StateMachineService) CreateStateMachine(ctx context.Context, projectID, name, description, yamlConfig string) (*state_machine.StateMachine, error) {
	cfg, err := state_machine.ParseConfig(yamlConfig)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	sm := state_machine.NewStateMachine(projectID, name, description, cfg)
	if err := s.repo.SaveStateMachine(ctx, sm); err != nil {
		return nil, err
	}

	return sm, nil
}

// GetStateMachine 获取状态机
func (s *StateMachineService) GetStateMachine(ctx context.Context, id string) (*state_machine.StateMachine, error) {
	return s.repo.GetStateMachine(ctx, id)
}

// ListStateMachines 列出状态机
func (s *StateMachineService) ListStateMachines(ctx context.Context, projectID string) ([]*state_machine.StateMachine, error) {
	return s.repo.ListStateMachines(ctx, projectID)
}

// DeleteStateMachine 删除状态机
func (s *StateMachineService) DeleteStateMachine(ctx context.Context, id string) error {
	return s.repo.DeleteStateMachine(ctx, id)
}

// BindType 绑定类型
func (s *StateMachineService) BindType(ctx context.Context, stateMachineID, requirementType string) error {
	// 验证状态机存在
	_, err := s.repo.GetStateMachine(ctx, stateMachineID)
	if err != nil {
		return err
	}

	binding := state_machine.NewTypeBinding(stateMachineID, requirementType)
	return s.repo.SaveTypeBinding(ctx, binding)
}

// UnbindType 解绑类型
func (s *StateMachineService) UnbindType(ctx context.Context, stateMachineID, requirementType string) error {
	return s.repo.DeleteTypeBinding(ctx, stateMachineID, requirementType)
}

// InitializeRequirementState 初始化需求状态（创建需求时调用）
func (s *StateMachineService) InitializeRequirementState(ctx context.Context, requirementID, projectID, requirementType string) (*state_machine.RequirementState, error) {
	// 查找绑定的状态机
	sm, err := s.repo.GetStateMachineByType(ctx, projectID, requirementType)
	if err != nil {
		return nil, err
	}
	if sm == nil {
		return nil, state_machine.ErrStateMachineNotFound("no state machine bound for type '" + requirementType + "'")
	}

	// 创建初始状态
	initialState := sm.Config.GetState(sm.Config.InitialState)
	if initialState == nil {
		return nil, state_machine.ErrStateNotFound(sm.Config.InitialState)
	}

	rs := state_machine.NewRequirementState(requirementID, sm.ID, initialState.ID, initialState.Name)
	if err := s.repo.SaveRequirementState(ctx, rs); err != nil {
		return nil, err
	}

	// 记录日志
	log := state_machine.NewTransitionLog(requirementID, "", initialState.ID, "init", "system", "requirement created")
	s.repo.SaveTransitionLog(ctx, log)

	return rs, nil
}

// TriggerTransition 触发转换
func (s *StateMachineService) TriggerTransition(ctx context.Context, requirementID, trigger, triggeredBy, remark string) (*state_machine.RequirementState, error) {
	// 获取当前状态
	rs, err := s.repo.GetRequirementState(ctx, requirementID)
	if err != nil {
		return nil, err
	}

	// 获取状态机
	sm, err := s.repo.GetStateMachine(ctx, rs.StateMachineID)
	if err != nil {
		return nil, err
	}

	// 查找转换规则
	transition := sm.Config.FindTransition(rs.CurrentState, trigger)
	if transition == nil {
		return nil, state_machine.ErrTransitionNotFound(rs.CurrentState, trigger)
	}

	// 获取目标状态
	toState := sm.Config.GetState(transition.ToState)
	if toState == nil {
		return nil, state_machine.ErrStateNotFound(transition.ToState)
	}

	// 记录日志
	log := state_machine.NewTransitionLog(requirementID, rs.CurrentState, toState.ID, trigger, triggeredBy, remark)

	// 更新状态
	rs.Transition(toState.ID, toState.Name)
	if err := s.repo.UpdateRequirementState(ctx, rs); err != nil {
		log.MarkFailed(err.Error())
		s.repo.SaveTransitionLog(ctx, log)
		return nil, err
	}

	// 保存日志
	if err := s.repo.SaveTransitionLog(ctx, log); err != nil {
		s.logger.Warn("failed to save transition log", zap.Error(err))
	}

	// 异步执行 hooks
	if len(transition.Hooks) > 0 {
		hookCtx := infra_sm.HookContext{
			RequirementID:   requirementID,
			ProjectID:       sm.ProjectID,
			StateMachineID:  sm.ID,
			FromState:       rs.CurrentState,
			ToState:         toState.ID,
			Trigger:         trigger,
			HookName:        "",
			HookType:        "",
		}
		s.executor.ExecuteHooks(ctx, transition.Hooks, hookCtx)
	}

	return rs, nil
}

// GetRequirementState 获取需求状态
func (s *StateMachineService) GetRequirementState(ctx context.Context, requirementID string) (*state_machine.RequirementState, error) {
	return s.repo.GetRequirementState(ctx, requirementID)
}

// GetTransitionHistory 获取转换历史
func (s *StateMachineService) GetTransitionHistory(ctx context.Context, requirementID string) ([]*state_machine.TransitionLog, error) {
	return s.repo.ListTransitionLogs(ctx, requirementID)
}

// StateSummary 状态统计
type StateSummary struct {
	StateID   string `json:"state_id"`
	StateName string `json:"state_name"`
	Count     int    `json:"count"`
}

// GetProjectStateSummary 获取项目下需求的状态统计
func (s *StateMachineService) GetProjectStateSummary(ctx context.Context, projectID string) ([]*StateSummary, error) {
	// 获取项目下所有状态机
	sms, err := s.repo.ListStateMachines(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var summaries []*StateSummary
	for _, sm := range sms {
		for _, state := range sm.Config.States {
			summaries = append(summaries, &StateSummary{
				StateID:   state.ID,
				StateName: state.Name,
				Count:     0, // TODO: 实现统计查询
			})
		}
	}

	return summaries, nil
}
