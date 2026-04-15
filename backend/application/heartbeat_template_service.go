package application

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

type CreateHeartbeatTemplateCommand struct {
	Name            string
	MDContent       string
	RequirementType string
}

type HeartbeatTemplateApplicationService struct {
	repo        domain.HeartbeatTemplateRepository
	idGenerator domain.IDGenerator
}

func NewHeartbeatTemplateApplicationService(
	repo domain.HeartbeatTemplateRepository,
	idGenerator domain.IDGenerator,
) *HeartbeatTemplateApplicationService {
	return &HeartbeatTemplateApplicationService{
		repo:        repo,
		idGenerator: idGenerator,
	}
}

func (s *HeartbeatTemplateApplicationService) CreateHeartbeatTemplate(ctx context.Context, cmd CreateHeartbeatTemplateCommand) (*domain.HeartbeatTemplate, error) {
	t, err := domain.NewHeartbeatTemplate(
		domain.NewHeartbeatTemplateID(s.idGenerator.Generate()),
		cmd.Name,
		cmd.MDContent,
		cmd.RequirementType,
	)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("failed to save heartbeat template: %w", err)
	}
	return t, nil
}

func (s *HeartbeatTemplateApplicationService) ListHeartbeatTemplates(ctx context.Context) ([]*domain.HeartbeatTemplate, error) {
	return s.repo.FindAll(ctx)
}

func (s *HeartbeatTemplateApplicationService) DeleteHeartbeatTemplate(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, domain.NewHeartbeatTemplateID(id)); err != nil {
		return fmt.Errorf("failed to delete heartbeat template: %w", err)
	}
	return nil
}
