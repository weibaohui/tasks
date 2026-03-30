package application

import (
	"context"
	"errors"

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
}

type UpdateRequirementCommand struct {
	ID                 domain.RequirementID
	Title              string
	Description        string
	AcceptanceCriteria string
}

type ReportRequirementPRCommand struct {
	ID         domain.RequirementID
	PRURL      string
	BranchName string
}

type RequirementApplicationService struct {
	requirementRepo domain.RequirementRepository
	projectRepo     domain.ProjectRepository
	idGenerator     domain.IDGenerator
}

func NewRequirementApplicationService(requirementRepo domain.RequirementRepository, projectRepo domain.ProjectRepository, idGenerator domain.IDGenerator) *RequirementApplicationService {
	return &RequirementApplicationService{
		requirementRepo: requirementRepo,
		projectRepo:     projectRepo,
		idGenerator:     idGenerator,
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
	if err := requirement.UpdateContent(cmd.Title, cmd.Description, cmd.AcceptanceCriteria); err != nil {
		return nil, err
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}

func (s *RequirementApplicationService) ReportRequirementPROpened(ctx context.Context, cmd ReportRequirementPRCommand) (*domain.Requirement, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}
	requirement.MarkPROpened(cmd.PRURL, cmd.BranchName)
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}
