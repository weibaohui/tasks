/**
 * LLM Provider 工厂实现
 * 将 domain.LLMProviderConfig 转换为 infrastructure LLMProvider
 * 支持 OpenAI 格式和 Anthropic 格式
 */
package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/weibh/taskmanager/domain"
)

// LLMProviderFactoryImpl 基础设施层实现：创建 LLM Provider
type LLMProviderFactoryImpl struct{}

// NewLLMProviderFactory 创建 LLM Provider 工厂
func NewLLMProviderFactory() *LLMProviderFactoryImpl {
	return &LLMProviderFactoryImpl{}
}

// Build 根据 domain 配置创建 LLM Provider
func (f *LLMProviderFactoryImpl) Build(config *domain.LLMProviderConfig) (interface{}, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	if err := f.validate(config); err != nil {
		return nil, err
	}

	// 根据 API 类型选择不同的创建方式
	switch config.APIFormat() {
	case domain.APITypeAnthropic:
		return f.createClaudeProvider(config)
	default:
		return f.createOpenAIProvider(config)
	}
}

// createOpenAIProvider 创建 OpenAI 兼容格式的 Provider
func (f *LLMProviderFactoryImpl) createOpenAIProvider(config *domain.LLMProviderConfig) (*EinoProvider, error) {
	baseURL := config.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  config.GetAPIKey(),
		Model:   config.ModelName(),
		BaseURL: baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create openai chat model: %w", err)
	}

	cfg := &Config{
		ProviderType: config.ProviderKey(),
		Model:       config.ModelName(),
		APIKey:      config.GetAPIKey(),
		BaseURL:     baseURL,
		Temperature: config.GetTemperature(),
		MaxTokens:   config.GetMaxTokens(),
	}

	return &EinoProvider{
		config:    cfg,
		chatModel: chatModel,
	}, nil
}

// createClaudeProvider 创建 Anthropic/Claude 原生格式的 Provider
func (f *LLMProviderFactoryImpl) createClaudeProvider(config *domain.LLMProviderConfig) (*EinoProvider, error) {
	baseURL := config.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	claudeConfig := &claude.Config{
		APIKey:     config.GetAPIKey(),
		Model:      config.ModelName(),
		BaseURL:    &baseURL,
		HTTPClient: NewClaudeHTTPClient(), // 使用 Claude Code 伪装 HTTP Client
	}

	chatModel, err := claude.NewChatModel(context.Background(), claudeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create claude chat model: %w", err)
	}

	cfg := &Config{
		ProviderType: config.ProviderKey(),
		Model:       config.ModelName(),
		APIKey:      config.GetAPIKey(),
		BaseURL:     baseURL,
		Temperature: config.GetTemperature(),
		MaxTokens:   config.GetMaxTokens(),
	}

	return &EinoProvider{
		config:    cfg,
		chatModel: chatModel,
	}, nil
}

// validate 验证配置
func (f *LLMProviderFactoryImpl) validate(config *domain.LLMProviderConfig) error {
	if config.ProviderKey() == "" {
		return fmt.Errorf("provider type is required")
	}
	if config.ModelName() == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}