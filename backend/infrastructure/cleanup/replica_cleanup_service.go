package cleanup

import (
	"context"
	"log"

	"github.com/weibh/taskmanager/domain"
)

// ReplicaCleanupService 负责清理分身Agent和工作区
// 实现 domain.ReplicaCleanupService 接口
type ReplicaCleanupService struct {
	agentRepo       domain.AgentRepository
	workspaceManager domain.WorkspaceManager
}

var _ domain.ReplicaCleanupService = (*ReplicaCleanupService)(nil)

func NewReplicaCleanupService(agentRepo domain.AgentRepository, workspaceManager domain.WorkspaceManager) *ReplicaCleanupService {
	return &ReplicaCleanupService{agentRepo: agentRepo, workspaceManager: workspaceManager}
}

// CleanupReplica 清理分身和工作区（幂等方法）
func (s *ReplicaCleanupService) CleanupReplica(ctx context.Context, replicaAgentCode, workspacePath string) error {
	if replicaAgentCode == "" && workspacePath == "" {
		return nil
	}

	// 1. 删除分身 Agent
	if replicaAgentCode != "" {
		agent, err := s.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(replicaAgentCode))
		if err != nil {
			log.Printf("failed to find replica agent %s: %v", replicaAgentCode, err)
		} else if agent != nil {
			if err := s.agentRepo.Delete(ctx, agent.ID()); err != nil {
				log.Printf("failed to delete replica agent %s: %v", agent.AgentCode().String(), err)
			} else {
				log.Printf("replica agent %s disposed", agent.AgentCode().String())
			}
		}
	}

	// 2. 清理工作目录
	if workspacePath != "" {
		if err := s.workspaceManager.RemoveWorkspace(workspacePath); err != nil {
			log.Printf("failed to cleanup workspace %s: %v", workspacePath, err)
			return err
		}
		log.Printf("workspace %s cleaned", workspacePath)
	}

	return nil
}
