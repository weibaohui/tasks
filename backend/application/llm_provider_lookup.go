/**
 * LLM Provider 查找逻辑
 * 封装任务到 LLM Provider 的映射关系
 *
 * 选择策略已下沉到 domain 层的 LLMProviderSelectionService
 * Provider 构造由基础设施层 LLMProviderFactory 实现
 */
package application

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

// taskLLMProvider LLM Provider 查找器
// 协调 domain service 和 infrastructure factory
type taskLLMProvider struct {
	selectionService *domain.LLMProviderSelectionService
	factory         domain.LLMProviderFactory
}

// newTaskLLMProvider 创建 LLM Provider 查找器
func newTaskLLMProvider(
	agentRepo domain.AgentRepository,
	providerRepo domain.LLMProviderRepository,
	channelRepo domain.ChannelRepository,
	factory domain.LLMProviderFactory,
) *taskLLMProvider {
	return &taskLLMProvider{
		selectionService: domain.NewLLMProviderSelectionService(
			agentRepo,
			providerRepo,
			channelRepo,
		),
		factory: factory,
	}
}

// getProviderForTask 根据任务元数据获取 LLM Provider
// 1. 调用 domain service 选择合适的 provider 配置
// 2. 调用 infrastructure factory 创建实际的 provider
func (t *taskLLMProvider) getProviderForTask(ctx context.Context, task *domain.Task) (llm.LLMProvider, error) {
	// 1. 调用 domain service 获取 provider 配置
	config, err := t.selectionService.SelectProviderForTask(ctx, task)
	if err != nil {
		return nil, err
	}

	// 2. 验证配置
	if err := t.selectionService.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid provider config: %w", err)
	}

	// 3. 调用 infrastructure factory 创建实际的 provider
	provider, err := t.factory.Build(config)
	if err != nil {
		return nil, err
	}

	// 4. 类型断言为 llm.LLMProvider
	if p, ok := provider.(llm.LLMProvider); ok {
		return p, nil
	}
	return nil, fmt.Errorf("provider type assertion failed")
}