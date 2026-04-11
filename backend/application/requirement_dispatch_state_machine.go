package application

import (
	"context"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/state_machine"
)

func (s *RequirementDispatchService) getProjectStateMachineName(ctx context.Context, projectID string, reqType domain.RequirementType) string {
	if s.stateMachineRepo == nil {
		return ""
	}

	psm, err := s.stateMachineRepo.GetProjectStateMachine(ctx, projectID, state_machine.RequirementType(reqType))
	if err != nil {
		return ""
	}

	snap := psm.ToSnapshot()
	sm, err := s.stateMachineRepo.GetStateMachine(ctx, snap.StateMachineID)
	if err != nil {
		return ""
	}

	return sm.Name
}

// getStateMachineGuide 获取当前状态机状态和 AI Guide
// 返回当前状态 ID 和 AI Guide 信息
func (s *RequirementDispatchService) getStateMachineGuide(ctx context.Context, projectID, requirementID string, reqType domain.RequirementType) (string, map[string]interface{}) {
	if s.stateMachineRepo == nil {
		return "", nil
	}

	// 获取项目状态机映射
	psm, err := s.stateMachineRepo.GetProjectStateMachine(ctx, projectID, state_machine.RequirementType(reqType))
	if err != nil {
		return "", nil
	}

	snap := psm.ToSnapshot()

	// 获取状态机配置
	sm, err := s.stateMachineRepo.GetStateMachine(ctx, snap.StateMachineID)
	if err != nil {
		return "", nil
	}

	// 获取需求当前状态（从 RequirementState）
	reqState, err := s.stateMachineRepo.GetRequirementState(ctx, requirementID)
	if err != nil {
		// 如果没有 RequirementState，返回初始状态
		return sm.Config.InitialState, sm.Config.GetStateAIGuide(sm.Config.InitialState)
	}

	// 返回当前状态和 AI Guide
	return reqState.CurrentState, sm.Config.GetStateAIGuide(reqState.CurrentState)
}

// saveRequirementState 保存需求状态到状态机
func (s *RequirementDispatchService) saveRequirementState(ctx context.Context, requirement *domain.Requirement, currentState string) {
	// 获取项目状态机映射
	psm, err := s.stateMachineRepo.GetProjectStateMachine(ctx, requirement.ProjectID().String(), state_machine.RequirementType(requirement.RequirementType()))
	if err != nil {
		return
	}

	snap := psm.ToSnapshot()

	// 获取状态机配置
	sm, err := s.stateMachineRepo.GetStateMachine(ctx, snap.StateMachineID)
	if err != nil {
		return
	}

	// 获取状态信息
	stateInfo := sm.Config.GetState(currentState)
	if stateInfo == nil {
		return
	}

	// 创建或更新 RequirementState
	rs := state_machine.NewRequirementState(requirement.ID().String(), sm.ID, currentState, stateInfo.Name)
	_ = s.stateMachineRepo.SaveRequirementState(ctx, rs)
}

// getStateMachineConfig 获取完整的状态机配置
func (s *RequirementDispatchService) getStateMachineConfig(ctx context.Context, projectID string, reqType domain.RequirementType) *state_machine.Config {
	if s.stateMachineRepo == nil {
		return nil
	}

	// 获取项目状态机映射
	psm, err := s.stateMachineRepo.GetProjectStateMachine(ctx, projectID, state_machine.RequirementType(reqType))
	if err != nil {
		return nil
	}

	snap := psm.ToSnapshot()

	// 获取状态机
	sm, err := s.stateMachineRepo.GetStateMachine(ctx, snap.StateMachineID)
	if err != nil {
		return nil
	}

	return sm.Config
}
