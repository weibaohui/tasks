package application

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

// PatchAgentCommand 局部更新 Agent 的命令，仅非 nil 字段会被应用
type PatchAgentCommand struct {
	ID                    domain.AgentID
	Name                  *string
	Description           *string
	IdentityContent       *string
	SoulContent           *string
	AgentsContent         *string
	UserContent           *string
	ToolsContent          *string
	Model                 *string
	LLMProviderID         *string
	MaxTokens             *int
	Temperature           *float64
	MaxIterations         *int
	HistoryMessages       *int
	SkillsList            *[]string
	ToolsList             *[]string
	IsActive              *bool
	IsDefault             *bool
	EnableThinkingProcess *bool
	AgentType             *string
	ClaudeCodeConfig      *domain.ClaudeCodeConfig
}

// agentConfigPatch 通用接口，用于提取 agent config 的 patch 字段
type agentConfigPatch interface {
	hasConfigField() bool
	applyTo(agent *domain.Agent) domain.AgentConfigUpdate
}

// updateAgentPatch adapts UpdateAgentCommand to agentConfigPatch
type updateAgentPatch UpdateAgentCommand

func (p updateAgentPatch) hasConfigField() bool {
	return p.IdentityContent != nil || p.SoulContent != nil ||
		p.AgentsContent != nil || p.UserContent != nil || p.ToolsContent != nil ||
		p.Model != nil || p.MaxTokens != nil || p.Temperature != nil ||
		p.MaxIterations != nil || p.HistoryMessages != nil ||
		p.SkillsList != nil || p.ToolsList != nil ||
		p.EnableThinkingProcess != nil
}

func (p updateAgentPatch) applyTo(agent *domain.Agent) domain.AgentConfigUpdate {
	cfg := domain.AgentConfigUpdate{
		IdentityContent:      agent.IdentityContent(),
		SoulContent:          agent.SoulContent(),
		AgentsContent:        agent.AgentsContent(),
		UserContent:          agent.UserContent(),
		ToolsContent:         agent.ToolsContent(),
		Model:                agent.Model(),
		MaxTokens:            agent.MaxTokens(),
		Temperature:          agent.Temperature(),
		MaxIterations:        agent.MaxIterations(),
		HistoryMessages:      agent.HistoryMessages(),
		SkillsList:           agent.SkillsList(),
		ToolsList:            agent.ToolsList(),
		EnableThinkingProcess: agent.EnableThinkingProcess(),
	}
	if p.IdentityContent != nil {
		cfg.IdentityContent = *p.IdentityContent
	}
	if p.SoulContent != nil {
		cfg.SoulContent = *p.SoulContent
	}
	if p.AgentsContent != nil {
		cfg.AgentsContent = *p.AgentsContent
	}
	if p.UserContent != nil {
		cfg.UserContent = *p.UserContent
	}
	if p.ToolsContent != nil {
		cfg.ToolsContent = *p.ToolsContent
	}
	if p.Model != nil {
		cfg.Model = *p.Model
	}
	if p.MaxTokens != nil {
		cfg.MaxTokens = *p.MaxTokens
	}
	if p.Temperature != nil {
		cfg.Temperature = *p.Temperature
	}
	if p.MaxIterations != nil {
		cfg.MaxIterations = *p.MaxIterations
	}
	if p.HistoryMessages != nil {
		cfg.HistoryMessages = *p.HistoryMessages
	}
	if p.SkillsList != nil {
		cfg.SkillsList = *p.SkillsList
	}
	if p.ToolsList != nil {
		cfg.ToolsList = *p.ToolsList
	}
	if p.EnableThinkingProcess != nil {
		cfg.EnableThinkingProcess = *p.EnableThinkingProcess
	}
	return cfg
}

// patchAgentPatch adapts PatchAgentCommand to agentConfigPatch
type patchAgentPatch PatchAgentCommand

func (p patchAgentPatch) hasConfigField() bool {
	return p.IdentityContent != nil || p.SoulContent != nil ||
		p.AgentsContent != nil || p.UserContent != nil || p.ToolsContent != nil ||
		p.Model != nil || p.MaxTokens != nil || p.Temperature != nil ||
		p.MaxIterations != nil || p.HistoryMessages != nil ||
		p.SkillsList != nil || p.ToolsList != nil ||
		p.EnableThinkingProcess != nil
}

func (p patchAgentPatch) applyTo(agent *domain.Agent) domain.AgentConfigUpdate {
	cfg := domain.AgentConfigUpdate{
		IdentityContent:      agent.IdentityContent(),
		SoulContent:          agent.SoulContent(),
		AgentsContent:        agent.AgentsContent(),
		UserContent:          agent.UserContent(),
		ToolsContent:         agent.ToolsContent(),
		Model:                agent.Model(),
		MaxTokens:            agent.MaxTokens(),
		Temperature:          agent.Temperature(),
		MaxIterations:        agent.MaxIterations(),
		HistoryMessages:      agent.HistoryMessages(),
		SkillsList:           agent.SkillsList(),
		ToolsList:            agent.ToolsList(),
		EnableThinkingProcess: agent.EnableThinkingProcess(),
	}
	if p.IdentityContent != nil {
		cfg.IdentityContent = *p.IdentityContent
	}
	if p.SoulContent != nil {
		cfg.SoulContent = *p.SoulContent
	}
	if p.AgentsContent != nil {
		cfg.AgentsContent = *p.AgentsContent
	}
	if p.UserContent != nil {
		cfg.UserContent = *p.UserContent
	}
	if p.ToolsContent != nil {
		cfg.ToolsContent = *p.ToolsContent
	}
	if p.Model != nil {
		cfg.Model = *p.Model
	}
	if p.MaxTokens != nil {
		cfg.MaxTokens = *p.MaxTokens
	}
	if p.Temperature != nil {
		cfg.Temperature = *p.Temperature
	}
	if p.MaxIterations != nil {
		cfg.MaxIterations = *p.MaxIterations
	}
	if p.HistoryMessages != nil {
		cfg.HistoryMessages = *p.HistoryMessages
	}
	if p.SkillsList != nil {
		cfg.SkillsList = *p.SkillsList
	}
	if p.ToolsList != nil {
		cfg.ToolsList = *p.ToolsList
	}
	if p.EnableThinkingProcess != nil {
		cfg.EnableThinkingProcess = *p.EnableThinkingProcess
	}
	return cfg
}

func (s *AgentApplicationService) UpdateAgent(ctx context.Context, cmd UpdateAgentCommand) (*domain.Agent, error) {
	agent, err := s.agentRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}

	if err := applyProfileUpdate(agent, cmd.Name, cmd.Description); err != nil {
		return nil, err
	}

	patch := updateAgentPatch(cmd)
	if patch.hasConfigField() {
		agent.UpdateConfig(patch.applyTo(agent))
	}

	if cmd.IsActive != nil {
		agent.SetActive(*cmd.IsActive)
	}
	if cmd.IsDefault != nil {
		agent.SetDefault(*cmd.IsDefault)
	}
	if cmd.AgentType != nil {
		if err := agent.SetAgentType(domain.AgentType(*cmd.AgentType)); err != nil {
			return nil, err
		}
	}
	agent.ApplyLLMProvider(cmd.LLMProviderID)

	if err := s.agentRepo.Save(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to save agent: %w", err)
	}
	return agent, nil
}

func (s *AgentApplicationService) PatchAgent(ctx context.Context, cmd PatchAgentCommand) (*domain.Agent, error) {
	agent, err := s.agentRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}

	if err := applyProfileUpdate(agent, cmd.Name, cmd.Description); err != nil {
		return nil, err
	}

	patch := patchAgentPatch(cmd)
	if patch.hasConfigField() {
		agent.UpdateConfig(patch.applyTo(agent))
	}

	if cmd.IsActive != nil {
		agent.SetActive(*cmd.IsActive)
	}
	if cmd.IsDefault != nil {
		agent.SetDefault(*cmd.IsDefault)
	}
	if cmd.AgentType != nil {
		if err := agent.SetAgentType(domain.AgentType(*cmd.AgentType)); err != nil {
			return nil, err
		}
	}

	if cmd.ClaudeCodeConfig != nil {
		agent.UpdateClaudeCodeConfig(cmd.ClaudeCodeConfig)
	}
	agent.ApplyLLMProvider(cmd.LLMProviderID)

	if err := s.agentRepo.Save(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to save agent: %w", err)
	}
	return agent, nil
}

// applyProfileUpdate 按需更新 Agent 的 profile 字段
func applyProfileUpdate(agent *domain.Agent, name, description *string) error {
	if name != nil {
		desc := description
		if desc == nil {
			d := agent.Description()
			desc = &d
		}
		if err := agent.UpdateProfile(*name, *desc); err != nil {
			return err
		}
	} else if description != nil {
		if err := agent.UpdateProfile(agent.Name(), *description); err != nil {
			return err
		}
	}
	return nil
}

func boolValue(ptr *bool, fallback bool) bool {
	if ptr == nil {
		return fallback
	}
	return *ptr
}
