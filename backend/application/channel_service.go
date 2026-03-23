package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
)

type CreateChannelCommand struct {
	UserCode  string
	Name      string
	Type      domain.ChannelType
	Config    map[string]interface{}
	AllowFrom []string
	AgentCode string
}

type UpdateChannelCommand struct {
	ID        domain.ChannelID
	Name      *string
	Config    *map[string]interface{}
	AllowFrom *[]string
	IsActive  *bool
	AgentCode *string
}

type ChannelApplicationService struct {
	channelRepo domain.ChannelRepository
	idGenerator domain.IDGenerator
}

func NewChannelApplicationService(
	channelRepo domain.ChannelRepository,
	idGenerator domain.IDGenerator,
) *ChannelApplicationService {
	return &ChannelApplicationService{
		channelRepo: channelRepo,
		idGenerator: idGenerator,
	}
}

func (s *ChannelApplicationService) CreateChannel(ctx context.Context, cmd CreateChannelCommand) (*domain.Channel, error) {
	channel, err := domain.NewChannel(
		domain.NewChannelID(s.idGenerator.Generate()),
		domain.NewChannelCode("chn_"+s.idGenerator.Generate()),
		cmd.UserCode,
		cmd.Name,
		cmd.Type,
	)
	if err != nil {
		return nil, err
	}
	channel.UpdateConfig(cmd.Config)
	channel.SetAllowFrom(cmd.AllowFrom)
	channel.BindAgent(cmd.AgentCode)

	if err := s.channelRepo.Save(ctx, channel); err != nil {
		return nil, fmt.Errorf("failed to save channel: %w", err)
	}
	return channel, nil
}

func (s *ChannelApplicationService) GetChannel(ctx context.Context, id domain.ChannelID) (*domain.Channel, error) {
	channel, err := s.channelRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}
	return channel, nil
}

func (s *ChannelApplicationService) GetChannelByCode(ctx context.Context, code domain.ChannelCode) (*domain.Channel, error) {
	channel, err := s.channelRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}
	return channel, nil
}

func (s *ChannelApplicationService) ListChannels(ctx context.Context, userCode string) ([]*domain.Channel, error) {
	if userCode == "" {
		return nil, errors.New("user_code is required")
	}
	return s.channelRepo.FindByUserCode(ctx, userCode)
}

func (s *ChannelApplicationService) UpdateChannel(ctx context.Context, cmd UpdateChannelCommand) (*domain.Channel, error) {
	channel, err := s.channelRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	if cmd.Name != nil {
		if err := channel.UpdateName(*cmd.Name); err != nil {
			return nil, err
		}
	}
	if cmd.Config != nil {
		channel.UpdateConfig(*cmd.Config)
	}
	if cmd.AllowFrom != nil {
		channel.SetAllowFrom(*cmd.AllowFrom)
	}
	if cmd.AgentCode != nil {
		channel.BindAgent(*cmd.AgentCode)
	}
	if cmd.IsActive != nil {
		channel.SetActive(*cmd.IsActive)
	}

	if err := s.channelRepo.Save(ctx, channel); err != nil {
		return nil, fmt.Errorf("failed to save channel: %w", err)
	}
	return channel, nil
}

func (s *ChannelApplicationService) DeleteChannel(ctx context.Context, id domain.ChannelID) error {
	channel, err := s.channelRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}
	return s.channelRepo.Delete(ctx, id)
}

// ListActiveChannels returns all active channels
func (s *ChannelApplicationService) ListActiveChannels(ctx context.Context) ([]*domain.Channel, error) {
	return s.channelRepo.FindActive(ctx)
}
