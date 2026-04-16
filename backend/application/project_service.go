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
	ID                  domain.ProjectID
	Name                string
	GitRepoURL          string
	DefaultBranch       string
	InitSteps           []string
	DispatchChannelCode *string
	DispatchSessionKey  *string
	DefaultAgentCode    *string
	MaxConcurrentAgents *int
}

type ProjectApplicationService struct {
	projectRepo           domain.ProjectRepository
	requirementTypeRepo  domain.RequirementTypeEntityRepository
	idGenerator          domain.IDGenerator
}

func NewProjectApplicationService(projectRepo domain.ProjectRepository, requirementTypeRepo domain.RequirementTypeEntityRepository, idGenerator domain.IDGenerator) *ProjectApplicationService {
	return &ProjectApplicationService{
		projectRepo:          projectRepo,
		requirementTypeRepo:  requirementTypeRepo,
		idGenerator:          idGenerator,
	}
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

	// 为项目创建默认的需求类型
	if err := s.ensureDefaultRequirementTypes(ctx, project.ID()); err != nil {
		// 日志记录错误，但不阻塞项目创建
		// 可能是需求类型表不存在或其他原因
	}

	return project, nil
}

// ensureDefaultRequirementTypes 确保项目有所需的默认类型（normal, heartbeat）
func (s *ProjectApplicationService) ensureDefaultRequirementTypes(ctx context.Context, projectID domain.ProjectID) error {
	if s.requirementTypeRepo == nil {
		return nil
	}

	defaultTypes := []struct {
		code        string
		name        string
		description string
		color       string
	}{
		{"normal", "普通需求", "普通流程需求，需要人工触发", "blue"},
		{"heartbeat", "心跳需求", "自动触发的心跳任务", "green"},
	}

	for _, dt := range defaultTypes {
		// 检查是否已存在
		existing, err := s.requirementTypeRepo.FindByCode(ctx, projectID, dt.code)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}

		// 创建默认类型
		rt, err := domain.NewRequirementTypeEntity(
			domain.NewRequirementTypeEntityID(s.idGenerator.Generate()),
			projectID,
			dt.code,
			dt.name,
			dt.description,
		)
		if err != nil {
			return err
		}
		rt.SetColor(dt.color)
		rt.SetIsSystem(true)
		if err := s.requirementTypeRepo.Save(ctx, rt); err != nil {
			return err
		}
	}
	return nil
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
	// 仅在提供了非空值时才更新派发配置
	// 防止前端发送空字符串覆盖现有有效配置
	if cmd.DispatchChannelCode != nil && *cmd.DispatchChannelCode != "" ||
		cmd.DispatchSessionKey != nil && *cmd.DispatchSessionKey != "" ||
		cmd.DefaultAgentCode != nil && *cmd.DefaultAgentCode != "" {
		project.UpdateDispatchConfig(cmd.DispatchChannelCode, cmd.DispatchSessionKey, cmd.DefaultAgentCode)
	}
	if cmd.MaxConcurrentAgents != nil {
		if err := project.SetMaxConcurrentAgents(*cmd.MaxConcurrentAgents); err != nil {
			return nil, err
		}
	}
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
