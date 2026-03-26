/**
 * 领域服务
 * 包含 LLM Provider 选择策略服务
 */
package domain

import (
	"context"
	"fmt"
)

// LLMProviderFactory 基础设施层实现，用于创建实际的 LLM Provider
// Build 返回的 interface{} 是 infrastructure/llm.LLMProvider，由调用方进行类型断言
type LLMProviderFactory interface {
	// Build 根据配置创建 LLM Provider
	Build(config *LLMProviderConfig) (interface{}, error)
}

// LLMProviderSelectionService 领域服务：LLM Provider 选择策略
// 封装 Provider 选择优先级、回退顺序、模型兜底逻辑
type LLMProviderSelectionService struct {
	agentRepo    AgentRepository
	providerRepo LLMProviderRepository
	channelRepo  ChannelRepository
}

// NewLLMProviderSelectionService 创建 LLM Provider 选择服务
func NewLLMProviderSelectionService(
	agentRepo AgentRepository,
	providerRepo LLMProviderRepository,
	channelRepo ChannelRepository,
) *LLMProviderSelectionService {
	return &LLMProviderSelectionService{
		agentRepo:    agentRepo,
		providerRepo: providerRepo,
		channelRepo:  channelRepo,
	}
}

// SelectProviderForTask 根据任务元数据选择合适的 LLM Provider 配置
// 优先从 agent_code 查找，其次从 channel_code 查找，最后使用 user_code
func (s *LLMProviderSelectionService) SelectProviderForTask(ctx context.Context, task *Task) (*LLMProviderConfig, error) {
	metadata := task.Metadata()
	if metadata == nil {
		return nil, fmt.Errorf("任务元数据为空")
	}

	// 0. 尝试直接从 agent_code 获取（最高优先级，用于 Agent 工具创建的任务）
	agentCode, hasAgentCode := metadata["agent_code"].(string)
	if hasAgentCode && s.agentRepo != nil && s.providerRepo != nil {
		agent, err := s.agentRepo.FindByAgentCode(ctx, NewAgentCode(agentCode))
		if err == nil && agent != nil {
			provider, err := s.providerRepo.FindDefaultActive(ctx, agent.UserCode())
			if err == nil && provider != nil {
				return s.buildConfigFromProvider(provider, agent.Model()), nil
			}
		}
	}

	// 1. 尝试从 channel_code 获取
	channelCode, hasChannel := metadata["channel_code"].(string)
	userCode, hasUser := metadata["user_code"].(string)

	if hasChannel && s.channelRepo != nil && s.agentRepo != nil && s.providerRepo != nil {
		channel, err := s.channelRepo.FindByCode(ctx, NewChannelCode(channelCode))
		if err == nil && channel != nil {
			agentCode := channel.AgentCode()
			if agentCode != "" {
				agent, err := s.agentRepo.FindByAgentCode(ctx, NewAgentCode(agentCode))
				if err == nil && agent != nil {
					provider, err := s.providerRepo.FindDefaultActive(ctx, agent.UserCode())
					if err == nil && provider != nil {
						return s.buildConfigFromProvider(provider, agent.Model()), nil
					}
				}
			}
		}
	}

	// 2. 尝试直接从 user_code 获取
	if hasUser && s.providerRepo != nil {
		provider, err := s.providerRepo.FindDefaultActive(ctx, userCode)
		if err == nil && provider != nil {
			return s.buildConfigFromProvider(provider, ""), nil
		}
	}

	return nil, fmt.Errorf("未找到可用的 LLM Provider")
}

// buildConfigFromProvider 从 domain LLMProvider 构建配置
func (s *LLMProviderSelectionService) buildConfigFromProvider(provider *LLMProvider, model string) *LLMProviderConfig {
	cfg := NewLLMProviderConfig(
		provider.ProviderKey(),
		model,
		provider.APIKey(),
		provider.APIBase(),
	)

	// 设置 API 类型（默认为 openai）
	if provider.ProviderType() != "" {
		cfg.SetProviderType(provider.ProviderType())
	} else {
		cfg.SetProviderType(ProviderTypeOpenAI)
	}

	// 模型兜底：如果未指定模型，使用 provider 默认模型
	if cfg.Model == "" {
		cfg.Model = provider.DefaultModel()
	}

	return cfg
}

// ValidateConfig 验证配置是否有效
func (s *LLMProviderSelectionService) ValidateConfig(config *LLMProviderConfig) error {
	if config == nil {
		return fmt.Errorf("provider config is nil")
	}
	if config.ProviderKey() == "" {
		return fmt.Errorf("provider type is required")
	}
	if config.ModelName() == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}