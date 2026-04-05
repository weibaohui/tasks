package application

import (
	"context"

	"github.com/weibh/taskmanager/domain/state_machine"
)

// ProjectStateMachineMapping 项目状态机映射DTO
type ProjectStateMachineMapping struct {
	ID              string `json:"id"`
	RequirementType string `json:"requirement_type"`
	StateMachineID  string `json:"state_machine_id"`
	StateMachineName string `json:"state_machine_name,omitempty"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

// SetProjectStateMachineCommand 设置项目状态机命令
type SetProjectStateMachineCommand struct {
	ProjectID       string
	RequirementType string
	StateMachineID  string
}

// GetProjectStateMachinesQuery 获取项目状态机查询
type GetProjectStateMachinesQuery struct {
	ProjectID string
}

// ProjectStateMachineApplicationService 项目状态机应用服务
type ProjectStateMachineApplicationService struct {
	repo state_machine.Repository
}

// NewProjectStateMachineApplicationService 创建应用服务
func NewProjectStateMachineApplicationService(repo state_machine.Repository) *ProjectStateMachineApplicationService {
	return &ProjectStateMachineApplicationService{repo: repo}
}

// SetProjectStateMachine 设置项目状态机映射
func (s *ProjectStateMachineApplicationService) SetProjectStateMachine(ctx context.Context, cmd SetProjectStateMachineCommand) (*ProjectStateMachineMapping, error) {
	// 验证状态机是否存在
	_, err := s.repo.GetStateMachine(ctx, cmd.StateMachineID)
	if err != nil {
		return nil, err
	}

	reqType := state_machine.RequirementType(cmd.RequirementType)

	// 尝试获取现有映射
	existing, err := s.repo.GetProjectStateMachine(ctx, cmd.ProjectID, reqType)
	if err != nil && err != state_machine.ErrProjectStateMachineNotFound {
		return nil, err
	}

	var psm *state_machine.ProjectStateMachine
	if err == state_machine.ErrProjectStateMachineNotFound {
		// 创建新映射
		psm, err = state_machine.NewProjectStateMachine(cmd.ProjectID, reqType, cmd.StateMachineID)
		if err != nil {
			return nil, err
		}
	} else {
		// 更新现有映射
		psm = existing
		if err := psm.UpdateStateMachine(cmd.StateMachineID); err != nil {
			return nil, err
		}
	}

	if err := s.repo.SaveProjectStateMachine(ctx, psm); err != nil {
		return nil, err
	}

	snap := psm.ToSnapshot()
	return &ProjectStateMachineMapping{
		ID:               snap.ID,
		RequirementType:  string(snap.RequirementType),
		StateMachineID:   snap.StateMachineID,
		CreatedAt:        snap.CreatedAt.UnixMilli(),
		UpdatedAt:        snap.UpdatedAt.UnixMilli(),
	}, nil
}

// GetProjectStateMachines 获取项目的所有状态机映射
func (s *ProjectStateMachineApplicationService) GetProjectStateMachines(ctx context.Context, query GetProjectStateMachinesQuery) ([]*ProjectStateMachineMapping, error) {
	psms, err := s.repo.ListProjectStateMachines(ctx, query.ProjectID)
	if err != nil {
		return nil, err
	}

	result := make([]*ProjectStateMachineMapping, 0, len(psms))
	for _, psm := range psms {
		snap := psm.ToSnapshot()

		// 获取状态机名称
		var stateMachineName string
		sm, err := s.repo.GetStateMachine(ctx, snap.StateMachineID)
		if err == nil {
			stateMachineName = sm.Name
		}

		result = append(result, &ProjectStateMachineMapping{
			ID:               snap.ID,
			RequirementType:  string(snap.RequirementType),
			StateMachineID:   snap.StateMachineID,
			StateMachineName: stateMachineName,
			CreatedAt:        snap.CreatedAt.UnixMilli(),
			UpdatedAt:        snap.UpdatedAt.UnixMilli(),
		})
	}

	return result, nil
}

// DeleteProjectStateMachine 删除项目状态机映射
func (s *ProjectStateMachineApplicationService) DeleteProjectStateMachine(ctx context.Context, id string) error {
	return s.repo.DeleteProjectStateMachine(ctx, id)
}

// GetAvailableRequirementTypes 获取可用的需求类型列表
func (s *ProjectStateMachineApplicationService) GetAvailableRequirementTypes() []string {
	return []string{
		string(state_machine.RequirementTypeNormal),
		string(state_machine.RequirementTypeHeartbeat),
	}
}

// GetProjectStateMachineByType 获取指定类型的项目状态机映射
func (s *ProjectStateMachineApplicationService) GetProjectStateMachineByType(ctx context.Context, projectID string, requirementType string) (*ProjectStateMachineMapping, error) {
	reqType := state_machine.RequirementType(requirementType)
	psm, err := s.repo.GetProjectStateMachine(ctx, projectID, reqType)
	if err != nil {
		return nil, err
	}

	snap := psm.ToSnapshot()

	// 获取状态机名称
	var stateMachineName string
	sm, err := s.repo.GetStateMachine(ctx, snap.StateMachineID)
	if err == nil {
		stateMachineName = sm.Name
	}

	return &ProjectStateMachineMapping{
		ID:               snap.ID,
		RequirementType:  string(snap.RequirementType),
		StateMachineID:   snap.StateMachineID,
		StateMachineName: stateMachineName,
		CreatedAt:        snap.CreatedAt.UnixMilli(),
		UpdatedAt:        snap.UpdatedAt.UnixMilli(),
	}, nil
}
