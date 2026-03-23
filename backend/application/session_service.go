package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/weibh/taskmanager/domain"
)

var (
	ErrSessionNotFound = errors.New("session not found")
)

type CreateSessionCommand struct {
	UserCode    string
	ChannelCode string
	AgentCode   string
	SessionKey  string
	ExternalID  string
	Metadata    map[string]interface{}
}

type UpdateSessionMetadataCommand struct {
	SessionKey string
	Metadata   map[string]interface{}
}

type SessionApplicationService struct {
	sessionRepo domain.SessionRepository
	idGenerator domain.IDGenerator
}

func NewSessionApplicationService(
	sessionRepo domain.SessionRepository,
	idGenerator domain.IDGenerator,
) *SessionApplicationService {
	return &SessionApplicationService{
		sessionRepo: sessionRepo,
		idGenerator: idGenerator,
	}
}

func (s *SessionApplicationService) CreateSession(ctx context.Context, cmd CreateSessionCommand) (*domain.Session, error) {
	session, err := domain.NewSession(
		domain.NewSessionID(s.idGenerator.Generate()),
		cmd.UserCode,
		cmd.ChannelCode,
		cmd.SessionKey,
		cmd.ExternalID,
		cmd.AgentCode,
	)
	if err != nil {
		return nil, err
	}
	session.SetMetadata(cmd.Metadata)
	if err := s.sessionRepo.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}
	return session, nil
}

func (s *SessionApplicationService) GetSessionByKey(ctx context.Context, sessionKey string) (*domain.Session, error) {
	session, err := s.sessionRepo.FindBySessionKey(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (s *SessionApplicationService) GetSessionByID(ctx context.Context, id domain.SessionID) (*domain.Session, error) {
	session, err := s.sessionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (s *SessionApplicationService) ListUserSessions(ctx context.Context, userCode string) ([]*domain.Session, error) {
	if userCode == "" {
		return nil, errors.New("user_code is required")
	}
	return s.sessionRepo.FindActiveByUserCode(ctx, userCode)
}

func (s *SessionApplicationService) ListChannelSessions(ctx context.Context, channelCode string) ([]*domain.Session, error) {
	if channelCode == "" {
		return nil, errors.New("channel_code is required")
	}
	return s.sessionRepo.FindByChannelCode(ctx, channelCode)
}

func (s *SessionApplicationService) DeleteSession(ctx context.Context, sessionKey string) error {
	session, err := s.sessionRepo.FindBySessionKey(ctx, sessionKey)
	if err != nil {
		return err
	}
	if session == nil {
		return ErrSessionNotFound
	}
	return s.sessionRepo.DeleteBySessionKey(ctx, sessionKey)
}

func (s *SessionApplicationService) DeleteChannelSessions(ctx context.Context, channelCode string) error {
	return s.sessionRepo.DeleteByChannelCode(ctx, channelCode)
}

func (s *SessionApplicationService) TouchSession(ctx context.Context, sessionKey string) error {
	session, err := s.GetSessionByKey(ctx, sessionKey)
	if err != nil {
		return err
	}
	session.Touch()
	return s.sessionRepo.Save(ctx, session)
}

func (s *SessionApplicationService) GetLastActive(ctx context.Context, sessionKey string) (*time.Time, error) {
	session, err := s.GetSessionByKey(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	return session.LastActiveAt(), nil
}

func (s *SessionApplicationService) GetSessionMetadata(ctx context.Context, sessionKey string) (map[string]interface{}, error) {
	session, err := s.GetSessionByKey(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	return session.Metadata(), nil
}

func (s *SessionApplicationService) UpdateSessionMetadata(ctx context.Context, cmd UpdateSessionMetadataCommand) error {
	session, err := s.GetSessionByKey(ctx, cmd.SessionKey)
	if err != nil {
		return err
	}
	session.SetMetadata(cmd.Metadata)
	return s.sessionRepo.Save(ctx, session)
}
