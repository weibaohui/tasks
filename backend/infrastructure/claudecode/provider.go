package claudecode

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

func (p *ClaudeCodeProcessor) resolveProvider(ctx context.Context, agent *domain.Agent) (*domain.LLMProvider, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent is nil")
	}

	// 1. 优先使用 Agent 指定的 Provider
	llmProviderID := agent.LLMProviderID()
	if llmProviderID.String() != "" {
		provider, err := p.providerRepo.FindByID(ctx, llmProviderID)
		if err != nil {
			p.logger.Warn("获取 Agent 指定的 LLM Provider 失败，将使用用户默认 Provider",
				zap.String("agent_code", agent.AgentCode().String()),
				zap.String("llm_provider_id", llmProviderID.String()),
				zap.Error(err))
		} else if provider != nil {
			p.logger.Info("使用 Agent 指定的 LLM Provider",
				zap.String("agent_code", agent.AgentCode().String()),
				zap.String("provider_key", provider.ProviderKey()),
				zap.String("llm_provider_id", llmProviderID.String()))
			return provider, nil
		}
	}

	// 2. 使用用户默认 Provider
	userCode := agent.UserCode()
	provider, err := p.providerRepo.FindDefaultActive(ctx, userCode)
	if err != nil {
		return nil, fmt.Errorf("获取用户默认 Provider 失败: %w", err)
	}
	if provider != nil {
		p.logger.Info("使用用户默认 LLM Provider",
			zap.String("user_code", userCode),
			zap.String("provider_key", provider.ProviderKey()))
	}
	return provider, nil
}

// triggerClaudeCodeFinishedHook 触发 Claude Code 完成 hook
// success 参数表示 Claude Code 是否成功完成（不是错误退出）
// finalResult 参数是 Claude Code 的最终执行结果
