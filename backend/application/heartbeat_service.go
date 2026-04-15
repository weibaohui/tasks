package application

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

type CreateHeartbeatCommand struct {
	ProjectID       string
	Name            string
	IntervalMinutes int
	MDContent       string
	AgentCode       string
	RequirementType string
}

type UpdateHeartbeatCommand struct {
	ID              string
	Name            string
	IntervalMinutes int
	MDContent       string
	AgentCode       string
	RequirementType string
	Enabled         bool
}

type HeartbeatApplicationService struct {
	heartbeatRepo domain.HeartbeatRepository
	idGenerator   domain.IDGenerator
	scheduler     *HeartbeatScheduler
}

func NewHeartbeatApplicationService(
	heartbeatRepo domain.HeartbeatRepository,
	idGenerator domain.IDGenerator,
	scheduler *HeartbeatScheduler,
) *HeartbeatApplicationService {
	return &HeartbeatApplicationService{
		heartbeatRepo: heartbeatRepo,
		idGenerator:   idGenerator,
		scheduler:     scheduler,
	}
}

func (s *HeartbeatApplicationService) CreateHeartbeat(ctx context.Context, cmd CreateHeartbeatCommand) (*domain.Heartbeat, error) {
	hb, err := domain.NewHeartbeat(
		domain.NewHeartbeatID(s.idGenerator.Generate()),
		domain.NewProjectID(cmd.ProjectID),
		cmd.Name,
		cmd.IntervalMinutes,
		cmd.MDContent,
		cmd.AgentCode,
		cmd.RequirementType,
	)
	if err != nil {
		return nil, err
	}
	if err := s.heartbeatRepo.Save(ctx, hb); err != nil {
		return nil, fmt.Errorf("failed to save heartbeat: %w", err)
	}
	if s.scheduler != nil {
		if err := s.scheduler.RefreshSchedule(ctx, hb.ID().String()); err != nil {
			return nil, fmt.Errorf("failed to refresh schedule: %w", err)
		}
	}
	return hb, nil
}

func (s *HeartbeatApplicationService) UpdateHeartbeat(ctx context.Context, cmd UpdateHeartbeatCommand) (*domain.Heartbeat, error) {
	hb, err := s.heartbeatRepo.FindByID(ctx, domain.NewHeartbeatID(cmd.ID))
	if err != nil {
		return nil, err
	}
	if hb == nil {
		return nil, fmt.Errorf("heartbeat not found")
	}
	if err := hb.Update(cmd.Name, cmd.IntervalMinutes, cmd.MDContent, cmd.AgentCode, cmd.RequirementType); err != nil {
		return nil, err
	}
	hb.SetEnabled(cmd.Enabled)
	if err := s.heartbeatRepo.Save(ctx, hb); err != nil {
		return nil, fmt.Errorf("failed to save heartbeat: %w", err)
	}
	if s.scheduler != nil {
		if err := s.scheduler.RefreshSchedule(ctx, hb.ID().String()); err != nil {
			return nil, fmt.Errorf("failed to refresh schedule: %w", err)
		}
	}
	return hb, nil
}

func (s *HeartbeatApplicationService) DeleteHeartbeat(ctx context.Context, id string) error {
	hb, err := s.heartbeatRepo.FindByID(ctx, domain.NewHeartbeatID(id))
	if err != nil {
		return err
	}
	if hb == nil {
		return fmt.Errorf("heartbeat not found")
	}
	if err := s.heartbeatRepo.Delete(ctx, domain.NewHeartbeatID(id)); err != nil {
		return fmt.Errorf("failed to delete heartbeat: %w", err)
	}
	if s.scheduler != nil {
		if err := s.scheduler.RefreshSchedule(ctx, id); err != nil {
			return fmt.Errorf("failed to refresh schedule: %w", err)
		}
	}
	return nil
}

func (s *HeartbeatApplicationService) ListHeartbeatsByProject(ctx context.Context, projectID string) ([]*domain.Heartbeat, error) {
	return s.heartbeatRepo.FindByProjectID(ctx, domain.NewProjectID(projectID))
}

func (s *HeartbeatApplicationService) GetHeartbeat(ctx context.Context, id string) (*domain.Heartbeat, error) {
	return s.heartbeatRepo.FindByID(ctx, domain.NewHeartbeatID(id))
}
