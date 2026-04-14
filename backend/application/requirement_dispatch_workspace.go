package application

import (
	"github.com/weibh/taskmanager/domain"
)

func (s *RequirementDispatchService) workspaceRootPath() string {
	return s.workspaceConfig.WorkspaceRoot()
}

func (s *RequirementDispatchService) requirementWorkspaceRoot(requirement *domain.Requirement) string {
	if requirement != nil && requirement.TempWorkspaceRoot() != "" {
		return requirement.TempWorkspaceRoot()
	}
	return s.workspaceRootPath()
}

func resolveReplicaAgentCwd(requirement *domain.Requirement, workspacePath string) string {
	// workspacePath 已经由 DispatchRequirement 计算为包含 project_id 和 requirement_id 的完整路径
	// 无需再被 TempWorkspaceRoot 覆盖
	return workspacePath
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
