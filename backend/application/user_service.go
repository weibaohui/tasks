package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUsernameDuplicated = errors.New("username already exists")
)

type CreateUserCommand struct {
	Username     string
	Email        string
	DisplayName  string
	PasswordHash string
}

type UpdateUserCommand struct {
	ID          domain.UserID
	Email       string
	DisplayName string
	IsActive    *bool
}

type UserApplicationService struct {
	userRepo    domain.UserRepository
	idGenerator domain.IDGenerator
}

func NewUserApplicationService(
	userRepo domain.UserRepository,
	idGenerator domain.IDGenerator,
) *UserApplicationService {
	return &UserApplicationService{
		userRepo:    userRepo,
		idGenerator: idGenerator,
	}
}

func (s *UserApplicationService) CreateUser(ctx context.Context, cmd CreateUserCommand) (*domain.User, error) {
	exists, err := s.userRepo.FindByUsername(ctx, cmd.Username)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return nil, ErrUsernameDuplicated
	}

	user, err := domain.NewUser(
		domain.NewUserID(s.idGenerator.Generate()),
		domain.NewUserCode("usr_"+s.idGenerator.Generate()),
		cmd.Username,
		cmd.Email,
		cmd.DisplayName,
		cmd.PasswordHash,
	)
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}
	return user, nil
}

func (s *UserApplicationService) GetUser(ctx context.Context, id domain.UserID) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserApplicationService) ListUsers(ctx context.Context) ([]*domain.User, error) {
	return s.userRepo.FindAll(ctx)
}

func (s *UserApplicationService) UpdateUser(ctx context.Context, cmd UpdateUserCommand) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.UpdateProfile(cmd.Email, cmd.DisplayName)
	if cmd.IsActive != nil {
		if *cmd.IsActive {
			user.Activate()
		} else {
			user.Deactivate()
		}
	}

	if err := s.userRepo.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}
	return user, nil
}

func (s *UserApplicationService) DeleteUser(ctx context.Context, id domain.UserID) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	return s.userRepo.Delete(ctx, id)
}
