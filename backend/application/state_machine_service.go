package application

import (
	"log"
	"context"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/statemachine"
	"go.uber.org/zap"
)

// StateMachineService 应用服务
type StateMachineService struct {
	repo            statemachine.Repository
	requirementRepo domain.RequirementRepository
	executor        statemachine.HookExecutor
	logger          *zap.Logger
}

// NewStateMachineService 创建服务
func NewStateMachineService(repo statemachine.Repository, requirementRepo domain.RequirementRepository, executor statemachine.HookExecutor, logger *zap.Logger) *StateMachineService {
	return &StateMachineService{
		repo:            repo,
		requirementRepo: requirementRepo,
		executor:        executor,
		logger:          logger,
	}
}

// CreateStateMachine 创建状态机
func (s *StateMachineService) CreateStateMachine(ctx context.Context, name, description, yamlConfig string) (*statemachine.StateMachine, error) {
	cfg, err := statemachine.ParseConfig(yamlConfig)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	sm := statemachine.NewStateMachine(name, description, cfg)
	if err := s.repo.SaveStateMachine(ctx, sm); err != nil {
		return nil, err
	}

	return sm, nil
}

// UpdateStateMachine 更新状态机
func (s *StateMachineService) UpdateStateMachine(ctx context.Context, id, name, description, yamlConfig string) (*statemachine.StateMachine, error) {
	cfg, err := statemachine.ParseConfig(yamlConfig)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 验证状态机存在
	_, err = s.repo.GetStateMachine(ctx, id)
	if err != nil {
		return nil, err
	}

	// 使用相同 ID 创建新对象（SaveStateMachine 使用 UPSERT）
	sm := statemachine.NewStateMachine(name, description, cfg)
	sm.ID = id

	if err := s.repo.SaveStateMachine(ctx, sm); err != nil {
		return nil, err
	}

	return sm, nil
}

// GetStateMachine 获取状态机
func (s *StateMachineService) GetStateMachine(ctx context.Context, id string) (*statemachine.StateMachine, error) {
	return s.repo.GetStateMachine(ctx, id)
}

// ListStateMachines 列出状态机
func (s *StateMachineService) ListStateMachines(ctx context.Context) ([]*statemachine.StateMachine, error) {
	return s.repo.ListStateMachines(ctx)
}

// DeleteStateMachine 删除状态机
func (s *StateMachineService) DeleteStateMachine(ctx context.Context, id string) error {
	return s.repo.DeleteStateMachine(ctx, id)
}

// GetProjectStateMachine 获取项目关联的状态机
func (s *StateMachineService) GetProjectStateMachine(ctx context.Context, projectID string, requirementType statemachine.RequirementType) (*statemachine.ProjectStateMachine, error) {
	return s.repo.GetProjectStateMachine(ctx, projectID, requirementType)
}

// InitializeRequirementState 初始化需求状态（创建需求时调用）
// 注意：此方法需要调用方传入 stateMachineID，因为状态机不再绑定到特定项目
func (s *StateMachineService) InitializeRequirementState(ctx context.Context, requirementID, stateMachineID string) (*statemachine.RequirementState, error) {
	// 获取状态机
	sm, err := s.repo.GetStateMachine(ctx, stateMachineID)
	if err != nil {
		return nil, err
	}

	// 创建初始状态
	initialState := sm.Config.GetState(sm.Config.InitialState)
	if initialState == nil {
		return nil, statemachine.ErrStateNotFound(sm.Config.InitialState)
	}

	rs := statemachine.NewRequirementState(requirementID, sm.ID, initialState.ID, initialState.Name)
	if err := s.repo.SaveRequirementState(ctx, rs); err != nil {
		return nil, err
	}

	// 记录日志
	logEntry := statemachine.NewTransitionLog(requirementID, "", initialState.ID, "init", "system", "requirement created")
	s.repo.SaveTransitionLog(ctx, logEntry)

	return rs, nil
}

// TriggerTransition 触发转换
// metadata 通过 context 传递，用于 hook 上下文的模板变量替换
func (s *StateMachineService) TriggerTransition(ctx context.Context, requirementID, trigger, triggeredBy, remark string) (*statemachine.RequirementState, error) {
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
		return nil, statemachine.ErrTransitionNotFound(rs.CurrentState, trigger)
	}

	// 获取目标状态
	toState := sm.Config.GetState(transition.ToState)
	if toState == nil {
		return nil, statemachine.ErrStateNotFound(transition.ToState)
	}

	// 记录日志
	logEntry := statemachine.NewTransitionLog(requirementID, rs.CurrentState, toState.ID, trigger, triggeredBy, remark)

	// 更新状态
	rs.Transition(toState.ID, toState.Name)
	if err := s.repo.UpdateRequirementState(ctx, rs); err != nil {
		logEntry.MarkFailed(err.Error())
		s.repo.SaveTransitionLog(ctx, logEntry)
		return nil, err
	}

	// 保存日志
	if err := s.repo.SaveTransitionLog(ctx, logEntry); err != nil {
		s.logger.Warn("failed to save transition log", zap.Error(err))
	}

	// 同步更新 Requirement 的状态
	if s.requirementRepo != nil {
		requirement, err := s.requirementRepo.FindByID(ctx, domain.NewRequirementID(requirementID))
		if err == nil && requirement != nil {
			requirement.SyncStatusFromStateMachine(toState.ID)
			if errSave := s.requirementRepo.Save(ctx, requirement); errSave != nil {
				log.Printf("requirementRepo.Save failed: %v", errSave)
			}
		}
	}

	// 异步执行 hooks
	if len(transition.Hooks) > 0 && s.executor != nil {
		metadata := statemachine.MetadataFromContext(ctx)
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		hookCtx := statemachine.HookContext{
			RequirementID:  requirementID,
			StateMachineID: sm.ID,
			FromState:      rs.CurrentState,
			ToState:        toState.ID,
			Trigger:        trigger,
			HookName:       "",
			HookType:       "",
			Metadata:       metadata,
		}
		s.executor.ExecuteHooks(ctx, transition.Hooks, hookCtx)
	}

	return rs, nil
}

// GetRequirementState 获取需求状态
func (s *StateMachineService) GetRequirementState(ctx context.Context, requirementID string) (*statemachine.RequirementState, error) {
	return s.repo.GetRequirementState(ctx, requirementID)
}

// GetTransitionHistory 获取转换历史
func (s *StateMachineService) GetTransitionHistory(ctx context.Context, requirementID string) ([]*statemachine.TransitionLog, error) {
	return s.repo.ListTransitionLogs(ctx, requirementID)
}

// StateSummary 状态统计
type StateSummary struct {
	StateID   string `json:"state_id"`
	StateName string `json:"state_name"`
	Count     int    `json:"count"`
}

// GetStateSummary 获取所有状态机的状态统计
func (s *StateMachineService) GetStateSummary(ctx context.Context) ([]*StateSummary, error) {
	sms, err := s.repo.ListStateMachines(ctx)
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
