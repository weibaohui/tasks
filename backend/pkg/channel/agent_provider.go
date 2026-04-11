package channel

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

func (p *MessageProcessor) getAgentAndProvider(ctx context.Context, msg *bus.InboundMessage) (*domain.Agent, *domain.LLMProvider, error) {
	if msg.Metadata == nil {
		return nil, nil, fmt.Errorf("消息元数据为空")
	}

	// 获取 agent_code
	agentCode, ok := msg.Metadata["agent_code"].(string)
	if !ok || agentCode == "" {
		// 尝试从 channel_code 获取 channel 再获取 agent
		p.logger.Debug("消息中未包含 agent_code")
		return nil, nil, fmt.Errorf("消息中未包含 agent_code")
	}

	// 获取 Agent
	agent, err := p.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(agentCode))
	if err != nil || agent == nil {
		p.logger.Debug("获取 Agent 失败", zap.String("agent_code", agentCode), zap.Error(err))
		return nil, nil, err
	}

	// 获取用户的默认 LLM Provider
	userCode := agent.UserCode()
	provider, err := p.providerRepo.FindDefaultActive(ctx, userCode)
	if err != nil || provider == nil {
		p.logger.Debug("获取 LLM Provider 失败", zap.String("user_code", userCode), zap.Error(err))
		return agent, nil, err
	}

	return agent, provider, nil
}

