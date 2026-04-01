package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

var (
	ErrRequirementNotFound = errors.New("requirement not found")
)

type CreateRequirementCommand struct {
	ProjectID          domain.ProjectID
	Title              string
	Description        string
	AcceptanceCriteria string
	TempWorkspaceRoot  string
}

type UpdateRequirementCommand struct {
	ID                 domain.RequirementID
	Title              string
	Description        string
	AcceptanceCriteria string
	TempWorkspaceRoot  string
}

type ReportRequirementPRCommand struct {
	ID domain.RequirementID
}

type RedispatchRequirementCommand struct {
	ID domain.RequirementID
}

type RequirementApplicationService struct {
	requirementRepo     domain.RequirementRepository
	projectRepo         domain.ProjectRepository
	idGenerator         domain.IDGenerator
	hookExecutor        *domain.ConfigurableHookExecutor
	replicaAgentManager *domain.ReplicaAgentManager
}

func NewRequirementApplicationService(
	requirementRepo domain.RequirementRepository,
	projectRepo domain.ProjectRepository,
	idGenerator domain.IDGenerator,
	hookExecutor *domain.ConfigurableHookExecutor,
	replicaAgentManager *domain.ReplicaAgentManager,
) *RequirementApplicationService {
	return &RequirementApplicationService{
		requirementRepo:     requirementRepo,
		projectRepo:         projectRepo,
		idGenerator:         idGenerator,
		hookExecutor:        hookExecutor,
		replicaAgentManager: replicaAgentManager,
	}
}

func (s *RequirementApplicationService) CreateRequirement(ctx context.Context, cmd CreateRequirementCommand) (*domain.Requirement, error) {
	project, err := s.projectRepo.FindByID(ctx, cmd.ProjectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	requirement, err := domain.NewRequirement(
		domain.NewRequirementID(s.idGenerator.Generate()),
		cmd.ProjectID,
		cmd.Title,
		cmd.Description,
		cmd.AcceptanceCriteria,
		cmd.TempWorkspaceRoot,
	)
	if err != nil {
		return nil, err
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}

func (s *RequirementApplicationService) GetRequirement(ctx context.Context, id domain.RequirementID) (*domain.Requirement, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}
	return requirement, nil
}

func (s *RequirementApplicationService) ListRequirements(ctx context.Context, projectID *domain.ProjectID) ([]*domain.Requirement, error) {
	if projectID != nil {
		return s.requirementRepo.FindByProjectID(ctx, *projectID)
	}
	return s.requirementRepo.FindAll(ctx)
}

func (s *RequirementApplicationService) UpdateRequirement(ctx context.Context, cmd UpdateRequirementCommand) (*domain.Requirement, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}
	if err := requirement.UpdateContent(cmd.Title, cmd.Description, cmd.AcceptanceCriteria, cmd.TempWorkspaceRoot); err != nil {
		return nil, err
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}

func (s *RequirementApplicationService) ReportRequirementPROpened(ctx context.Context, cmd ReportRequirementPRCommand) (*domain.Requirement, error) {
	fmt.Printf("[DEBUG] ReportRequirementPROpened CALLED: id=%s\n", cmd.ID)
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}

	// 设置分身管理器（用于清理）
	requirement.SetReplicaAgentManager(s.replicaAgentManager)

	// 设置状态变更回调
	requirement.ClearStateChangeCallbacks()
	if s.hookExecutor != nil {
		requirement.SetStateChangeCallback(func(change *domain.StateChange) {
			fmt.Printf("[DEBUG] Hook callback triggered: trigger=%s, requirement=%s\n", change.Trigger, requirement.ID())
			s.hookExecutor.Execute(ctx, change.Trigger, requirement, change)
		})
		fmt.Println("[DEBUG] Hook executor available, callback set")
	} else {
		fmt.Println("[DEBUG] Hook executor is NIL!")
	}

	requirement.MarkPROpened()
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}

// RedispatchRequirement 重新派发需求
func (s *RequirementApplicationService) RedispatchRequirement(ctx context.Context, cmd RedispatchRequirementCommand) (*domain.Requirement, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}

	if err := requirement.Redispatch(); err != nil {
		return nil, err
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}
