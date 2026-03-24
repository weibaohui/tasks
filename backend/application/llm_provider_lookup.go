/**
 * LLM Provider 查找逻辑
 * 封装任务到 LLM Provider 的映射关系
 */
package application

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

// taskLLMProvider 任务与 LLM Provider 的关联查找器
type taskLLMProvider struct {
	agentRepo    domain.AgentRepository
	providerRepo domain.LLMProviderRepository
	channelRepo  domain.ChannelRepository
}

// newTaskLLMProvider 创建 LLM Provider 查找器
func newTaskLLMProvider(
	agentRepo domain.AgentRepository,
	providerRepo domain.LLMProviderRepository,
	channelRepo domain.ChannelRepository,
) *taskLLMProvider {
	return &taskLLMProvider{
		agentRepo:    agentRepo,
		providerRepo: providerRepo,
		channelRepo:  channelRepo,
	}
}

// getProviderForTask 根据任务元数据获取 LLM Provider
// 优先从 channel_code 查找，否则使用默认 provider
func (t *taskLLMProvider) getProviderForTask(ctx context.Context, task *domain.Task) (llm.LLMProvider, error) {
	metadata := task.Metadata()
	if metadata == nil {
		return nil, fmt.Errorf("任务元数据为空")
	}

	// 1. 尝试从 channel_code 获取
	channelCode, hasChannel := metadata["channel_code"].(string)
	userCode, hasUser := metadata["user_code"].(string)

	if hasChannel && hasUser && t.channelRepo != nil && t.agentRepo != nil && t.providerRepo != nil {
		channel, err := t.channelRepo.FindByCode(ctx, domain.NewChannelCode(channelCode))
		if err == nil && channel != nil {
			agentCode := channel.AgentCode()
			if agentCode != "" {
				agent, err := t.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(agentCode))
				if err == nil && agent != nil {
					provider, err := t.providerRepo.FindDefaultActive(ctx, agent.UserCode())
					if err == nil && provider != nil {
						return t.createProviderFromDomain(provider, agent.Model())
					}
				}
			}
		}
	}

	// 2. 尝试直接从 user_code 获取
	if hasUser && t.providerRepo != nil {
		provider, err := t.providerRepo.FindDefaultActive(ctx, userCode)
		if err == nil && provider != nil {
			return t.createProviderFromDomain(provider, "")
		}
	}

	return nil, fmt.Errorf("未找到可用的 LLM Provider")
}

// createProviderFromDomain 从 domain LLMProvider 创建 infrastructure LLMProvider
func (t *taskLLMProvider) createProviderFromDomain(provider *domain.LLMProvider, model string) (llm.LLMProvider, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider is nil")
	}

	cfg := &llm.Config{
		ProviderType: provider.ProviderKey(),
		Model:        model,
		APIKey:       provider.APIKey(),
		BaseURL:      provider.APIBase(),
		Temperature:  0.7,
		MaxTokens:    4096,
	}

	if cfg.Model == "" {
		cfg.Model = provider.DefaultModel()
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4"
	}

	return llm.NewLLMProvider(cfg)
}