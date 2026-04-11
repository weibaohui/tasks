package application

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

func (s *RequirementDispatchService) createReplicaAgent(ctx context.Context, baseAgent *domain.Agent, requirement *domain.Requirement, workspacePath string) (*domain.Agent, error) {
	cwd := resolveReplicaAgentCwd(requirement, workspacePath)
	replica := domain.NewReplicaAgent(
		baseAgent,
		domain.NewAgentID(s.idGenerator.Generate()),
		domain.NewAgentCode("agt_"+s.idGenerator.Generate()),
		requirement.ID().String(),
		cwd,
	)
	if err := s.agentRepo.Save(ctx, replica); err != nil {
		return nil, fmt.Errorf("save replica agent: %w", err)
	}
	return replica, nil
}
