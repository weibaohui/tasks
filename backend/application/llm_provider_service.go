package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
)

type CreateProviderCommand struct {
	UserCode        string
	ProviderKey     string
	ProviderName    string
	APIKey          string
	APIBase         string
	APIType         string // API 格式：openai, anthropic
	ExtraHeaders    map[string]string
	SupportedModels []domain.ModelInfo
	DefaultModel    string
	IsDefault       bool
	Priority        int
	AutoMerge       *bool
}

type UpdateProviderCommand struct {
	ID                    domain.LLMProviderID
	ProviderKey           *string
	ProviderName          *string
	APIKey                *string
	APIBase               *string
	APIType               *string
	ExtraHeaders          *map[string]string
	SupportedModels       *[]domain.ModelInfo
	DefaultModel          *string
	IsDefault             *bool
	Priority              *int
	AutoMerge             *bool
	IsActive              *bool
	EmbeddingModels       *[]domain.EmbeddingModelInfo
	DefaultEmbeddingModel *string
}

type LLMProviderApplicationService struct {
	repo        domain.LLMProviderRepository
	idGenerator domain.IDGenerator
}

func NewLLMProviderApplicationService(
	repo domain.LLMProviderRepository,
	idGenerator domain.IDGenerator,
) *LLMProviderApplicationService {
	return &LLMProviderApplicationService{
		repo:        repo,
		idGenerator: idGenerator,
	}
}

func (s *LLMProviderApplicationService) Create(ctx context.Context, cmd CreateProviderCommand) (*domain.LLMProvider, error) {
	provider, err := domain.NewLLMProvider(
		domain.NewLLMProviderID(s.idGenerator.Generate()),
		cmd.UserCode,
		cmd.ProviderKey,
		cmd.ProviderName,
		cmd.APIKey,
		cmd.APIBase,
	)
	if err != nil {
		return nil, err
	}

	if cmd.ExtraHeaders != nil {
		provider.SetExtraHeaders(cmd.ExtraHeaders)
	}
	provider.SetSupportedModels(cmd.SupportedModels)
	provider.SetDefaultModel(cmd.DefaultModel)
	provider.SetPriority(cmd.Priority)
	provider.SetDefault(cmd.IsDefault)
	if cmd.AutoMerge != nil {
		provider.SetAutoMerge(*cmd.AutoMerge)
	}
	if cmd.APIType != "" {
		provider.SetAPIType(cmd.APIType)
	}

	if cmd.IsDefault {
		if err := s.repo.ClearDefaultByUserCode(ctx, cmd.UserCode, nil); err != nil {
			return nil, err
		}
	}

	if err := s.repo.Save(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to save provider: %w", err)
	}
	return provider, nil
}

func (s *LLMProviderApplicationService) Get(ctx context.Context, id domain.LLMProviderID) (*domain.LLMProvider, error) {
	provider, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	return provider, nil
}

func (s *LLMProviderApplicationService) List(ctx context.Context, userCode string) ([]*domain.LLMProvider, error) {
	return s.repo.FindByUserCode(ctx, userCode)
}

func (s *LLMProviderApplicationService) Update(ctx context.Context, cmd UpdateProviderCommand) (*domain.LLMProvider, error) {
	provider, err := s.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	newProviderKey := provider.ProviderKey()
	newProviderName := provider.ProviderName()
	newAPIKey := provider.APIKey()
	newAPIBase := provider.APIBase()
	if cmd.ProviderKey != nil {
		newProviderKey = *cmd.ProviderKey
	}
	if cmd.ProviderName != nil {
		newProviderName = *cmd.ProviderName
	}
	if cmd.APIKey != nil {
		newAPIKey = *cmd.APIKey
	}
	if cmd.APIBase != nil {
		newAPIBase = *cmd.APIBase
	}
	if err := provider.UpdateProfile(newProviderKey, newProviderName, newAPIKey, newAPIBase); err != nil {
		return nil, err
	}

	if cmd.ExtraHeaders != nil {
		provider.SetExtraHeaders(*cmd.ExtraHeaders)
	}
	if cmd.SupportedModels != nil {
		provider.SetSupportedModels(*cmd.SupportedModels)
	}
	if cmd.DefaultModel != nil {
		provider.SetDefaultModel(*cmd.DefaultModel)
	}
	if cmd.Priority != nil {
		provider.SetPriority(*cmd.Priority)
	}
	if cmd.AutoMerge != nil {
		provider.SetAutoMerge(*cmd.AutoMerge)
	}
	if cmd.IsActive != nil {
		provider.SetActive(*cmd.IsActive)
	}
	if cmd.APIType != nil {
		provider.SetAPIType(*cmd.APIType)
	}
	if cmd.EmbeddingModels != nil || cmd.DefaultEmbeddingModel != nil {
		embeddingModels := provider.EmbeddingModels()
		defaultEmbeddingModel := provider.DefaultEmbeddingModel()
		if cmd.EmbeddingModels != nil {
			embeddingModels = *cmd.EmbeddingModels
		}
		if cmd.DefaultEmbeddingModel != nil {
			defaultEmbeddingModel = *cmd.DefaultEmbeddingModel
		}
		provider.SetEmbeddingModels(embeddingModels, defaultEmbeddingModel)
	}
	if cmd.IsDefault != nil {
		if *cmd.IsDefault {
			excludeID := provider.ID()
			if err := s.repo.ClearDefaultByUserCode(ctx, provider.UserCode(), &excludeID); err != nil {
				return nil, err
			}
		}
		provider.SetDefault(*cmd.IsDefault)
	}

	if err := s.repo.Save(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to save provider: %w", err)
	}
	return provider, nil
}

func (s *LLMProviderApplicationService) Delete(ctx context.Context, id domain.LLMProviderID) error {
	provider, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if provider == nil {
		return ErrProviderNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *LLMProviderApplicationService) TestConnection(ctx context.Context, id domain.LLMProviderID) (map[string]interface{}, error) {
	provider, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if provider.APIKey() == "" {
		return map[string]interface{}{
			"success": false,
			"message": "API Key 未配置",
		}, nil
	}
	return map[string]interface{}{
		"success":     true,
		"message":     "连接配置检查通过",
		"provider":    provider.ProviderName(),
		"providerKey": provider.ProviderKey(),
		"api_base":    provider.APIBase(),
		"model_count": len(provider.SupportedModels()),
	}, nil
}
