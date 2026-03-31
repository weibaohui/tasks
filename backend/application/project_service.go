package application

import (
	"context"
	"errors"

	"github.com/weibh/taskmanager/domain"
)

var (
	ErrProjectNotFound = errors.New("project not found")
)

type CreateProjectCommand struct {
	Name          string
	GitRepoURL    string
	DefaultBranch string
	InitSteps     []string
}

type UpdateProjectCommand struct {
	ID                        domain.ProjectID
	Name                      string
	GitRepoURL                string
	DefaultBranch             string
	InitSteps                 []string
	HeartbeatEnabled          bool
	HeartbeatIntervalMinutes  int
	HeartbeatMDContent        string
	AgentCode                string
	DispatchChannelCode      string
	DispatchSessionKey       string
}

type ProjectApplicationService struct {
	projectRepo  domain.ProjectRepository
	idGenerator  domain.IDGenerator
}

func NewProjectApplicationService(projectRepo domain.ProjectRepository, idGenerator domain.IDGenerator) *ProjectApplicationService {
	return &ProjectApplicationService{projectRepo: projectRepo, idGenerator: idGenerator}
}

func (s *ProjectApplicationService) CreateProject(ctx context.Context, cmd CreateProjectCommand) (*domain.Project, error) {
	project, err := domain.NewProject(
		domain.NewProjectID(s.idGenerator.Generate()),
		cmd.Name,
		cmd.GitRepoURL,
		cmd.DefaultBranch,
		cmd.InitSteps,
	)
	if err != nil {
		return nil, err
	}
	if err := s.projectRepo.Save(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectApplicationService) GetProject(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	return project, nil
}

func (s *ProjectApplicationService) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	return s.projectRepo.FindAll(ctx)
}

func (s *ProjectApplicationService) UpdateProject(ctx context.Context, cmd UpdateProjectCommand) (*domain.Project, error) {
	project, err := s.projectRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	if err := project.Update(cmd.Name, cmd.GitRepoURL, cmd.DefaultBranch, cmd.InitSteps); err != nil {
		return nil, err
	}
	project.UpdateHeartbeatConfig(cmd.HeartbeatEnabled, cmd.HeartbeatIntervalMinutes, cmd.HeartbeatMDContent, cmd.AgentCode)
	project.UpdateDispatchConfig(cmd.DispatchChannelCode, cmd.DispatchSessionKey)
	if err := s.projectRepo.Save(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectApplicationService) DeleteProject(ctx context.Context, id domain.ProjectID) error {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if project == nil {
		return ErrProjectNotFound
	}
	return s.projectRepo.Delete(ctx, id)
}
