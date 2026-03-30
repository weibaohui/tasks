package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/weibh/taskmanager/domain"
)

var (
	ErrBaseAgentNotFound = errors.New("base agent not found")
)

type DispatchRequirementCommand struct {
	RequirementID domain.RequirementID
	AgentID       domain.AgentID
}

type DispatchRequirementResult struct {
	RequirementID  string `json:"requirement_id"`
	Status         string `json:"status"`
	DevState       string `json:"dev_state"`
	WorkspacePath  string `json:"workspace_path"`
	ReplicaAgentID string `json:"replica_agent_id"`
	TaskID         string `json:"task_id"`
}

type RequirementDispatchService struct {
	requirementRepo domain.RequirementRepository
	projectRepo     domain.ProjectRepository
	agentRepo       domain.AgentRepository
	taskService     *TaskApplicationService
	idGenerator     domain.IDGenerator
}

func NewRequirementDispatchService(
	requirementRepo domain.RequirementRepository,
	projectRepo domain.ProjectRepository,
	agentRepo domain.AgentRepository,
	taskService *TaskApplicationService,
	idGenerator domain.IDGenerator,
) *RequirementDispatchService {
	return &RequirementDispatchService{
		requirementRepo: requirementRepo,
		projectRepo:     projectRepo,
		agentRepo:       agentRepo,
		taskService:     taskService,
		idGenerator:     idGenerator,
	}
}

func (s *RequirementDispatchService) DispatchRequirement(ctx context.Context, cmd DispatchRequirementCommand) (*DispatchRequirementResult, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.RequirementID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}
	project, err := s.projectRepo.FindByID(ctx, requirement.ProjectID())
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	baseAgent, err := s.agentRepo.FindByID(ctx, cmd.AgentID)
	if err != nil {
		return nil, err
	}
	if baseAgent == nil {
		return nil, ErrBaseAgentNotFound
	}
	if err := requirement.StartDispatch(cmd.AgentID.String()); err != nil {
		return nil, err
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	workspacePath := filepath.Join(workspaceRootPath(), requirement.ProjectID().String(), requirement.ID().String())
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		return nil, err
	}
	replicaAgent, err := s.createReplicaAgent(ctx, baseAgent, requirement, workspacePath)
	if err != nil {
		requirement.MarkFailed(err.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		return nil, err
	}
	branchName := fmt.Sprintf("feature/%s", requirement.ID().String())
	if err := requirement.MarkCoding(workspacePath, replicaAgent.ID().String(), branchName); err != nil {
		return nil, err
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	task, err := s.taskService.CreateTask(ctx, CreateTaskCommand{
		Name:               requirement.Title(),
		Description:        requirement.Description(),
		Type:               domain.TaskTypeAgent,
		TaskRequirement:    firstNonEmpty(requirement.Description(), requirement.Title()),
		AcceptanceCriteria: firstNonEmpty(requirement.AcceptanceCriteria(), "完成需求并创建 PR"),
		Timeout:            1800,
		MaxRetries:         0,
		Priority:           0,
		AgentCode:          replicaAgent.AgentCode().String(),
		UserCode:           replicaAgent.UserCode(),
	})
	if err != nil {
		requirement.MarkFailed(err.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		return nil, err
	}
	if err := s.taskService.StartTask(ctx, task.ID()); err != nil {
		requirement.MarkFailed(err.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		return nil, err
	}
	return &DispatchRequirementResult{
		RequirementID:  requirement.ID().String(),
		Status:         string(requirement.Status()),
		DevState:       string(requirement.DevState()),
		WorkspacePath:  requirement.WorkspacePath(),
		ReplicaAgentID: requirement.ReplicaAgentID(),
		TaskID:         task.ID().String(),
	}, nil
}

func (s *RequirementDispatchService) createReplicaAgent(ctx context.Context, baseAgent *domain.Agent, requirement *domain.Requirement, workspacePath string) (*domain.Agent, error) {
	snap := baseAgent.ToSnapshot()
	now := time.Now()
	snap.ID = domain.NewAgentID(s.idGenerator.Generate())
	snap.AgentCode = domain.NewAgentCode("agt_" + s.idGenerator.Generate())
	snap.Name = fmt.Sprintf("%s-replica-%s", baseAgent.Name(), requirement.ID().String())
	snap.IsDefault = false
	snap.IsActive = true
	snap.AgentType = domain.AgentTypeCoding
	snap.CreatedAt = now
	snap.UpdatedAt = now
	if snap.ClaudeCodeConfig == nil {
		snap.ClaudeCodeConfig = domain.DefaultClaudeCodeConfig()
	} else {
		cfg := *snap.ClaudeCodeConfig
		snap.ClaudeCodeConfig = &cfg
	}
	snap.ClaudeCodeConfig.Cwd = workspacePath
	continueConversation := false
	forkSession := true
	snap.ClaudeCodeConfig.ContinueConversation = &continueConversation
	snap.ClaudeCodeConfig.ForkSession = &forkSession
	replica := &domain.Agent{}
	replica.FromSnapshot(snap)
	if err := s.agentRepo.Save(ctx, replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func workspaceRootPath() string {
	if p := os.Getenv("AI_DEVOPS_WORKSPACE_ROOT"); p != "" {
		return p
	}
	return "/tmp/ai-devops"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
