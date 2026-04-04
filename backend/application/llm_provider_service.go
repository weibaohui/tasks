package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

const DefaultFallbackModel = "gpt-3.5-turbo"

var (
	ErrProviderNotFound = errors.New("provider not found")
)

// TestConnectionRunner 定义测试连接的接口，便于测试时注入 mock
type TestConnectionRunner interface {
	RunTest(ctx context.Context, config *llm.Config) error
}

// defaultTestConnectionRunner 默认的测试连接实现
type defaultTestConnectionRunner struct{}

func (r *defaultTestConnectionRunner) RunTest(ctx context.Context, config *llm.Config) error {
	client, err := llm.NewLLMProvider(config)
	if err != nil {
		return fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err = client.Generate(testCtx, "Hi, please respond with 'OK' if you receive this message.")
	return err
}

// ChooseModelForProvider 根据 provider 信息选择用于测试连接的模型
// 这是一个纯函数，便于独立测试
func ChooseModelForProvider(defaultModel string, supportedModels []domain.ModelInfo) string {
	model := defaultModel
	if model == "" && len(supportedModels) > 0 {
		model = supportedModels[0].ID
	}
	if model == "" {
		model = DefaultFallbackModel // 默认模型
	}
	return model
}

type CreateProviderCommand struct {
	UserCode        string
	ProviderKey     string
	ProviderName    string
	APIKey          string
	APIBase         string
	ProviderType    string // API 格式：openai, anthropic
	ExtraHeaders    map[string]string
	SupportedModels []domain.ModelInfo
	DefaultModel    string
	IsDefault       bool
	Priority        int
	AutoMerge       *bool
}

type UpdateProviderCommand struct {
	ID              domain.LLMProviderID
	ProviderKey     *string
	ProviderName    *string
	APIKey          *string
	APIBase         *string
	ProviderType    *string
	ExtraHeaders    *map[string]string
	SupportedModels *[]domain.ModelInfo
	DefaultModel    *string
	IsDefault       *bool
	Priority        *int
	AutoMerge       *bool
	IsActive        *bool
}

type LLMProviderApplicationService struct {
	repo        domain.LLMProviderRepository
	idGenerator domain.IDGenerator
	testRunner  TestConnectionRunner
}

func NewLLMProviderApplicationService(
	repo domain.LLMProviderRepository,
	idGenerator domain.IDGenerator,
) *LLMProviderApplicationService {
	return &LLMProviderApplicationService{
		repo:        repo,
		idGenerator: idGenerator,
		testRunner:  &defaultTestConnectionRunner{},
	}
}

// NewLLMProviderApplicationServiceWithRunner 允许注入自定义 testRunner，用于测试
func NewLLMProviderApplicationServiceWithRunner(
	repo domain.LLMProviderRepository,
	idGenerator domain.IDGenerator,
	testRunner TestConnectionRunner,
) *LLMProviderApplicationService {
	return &LLMProviderApplicationService{
		repo:        repo,
		idGenerator: idGenerator,
		testRunner:  testRunner,
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
	if cmd.ProviderType != "" {
		provider.SetProviderType(cmd.ProviderType)
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
	if cmd.ProviderType != nil {
		provider.SetProviderType(*cmd.ProviderType)
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

	// 使用纯函数选择模型
	model := ChooseModelForProvider(provider.DefaultModel(), provider.SupportedModels())

	config := &llm.Config{
		ProviderType: provider.ProviderKey(),
		Model:        model,
		APIKey:       provider.APIKey(),
		BaseURL:      provider.APIBase(),
		Temperature:  0.7,
		MaxTokens:    1024,
	}

	// 使用注入的 testRunner 进行测试
	err = s.testRunner.RunTest(ctx, config)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("API 调用失败: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success":     true,
		"message":     "连接测试成功",
		"provider":    provider.ProviderName(),
		"providerKey": provider.ProviderKey(),
		"api_base":    provider.APIBase(),
		"model":       model,
		"model_count": len(provider.SupportedModels()),
	}, nil
}
