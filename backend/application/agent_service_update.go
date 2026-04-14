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
	OpenCodeConfig        *domain.OpenCodeConfig
}

// agentConfigPatchFields 通用配置 patch 字段提取接口
type agentConfigPatchFields interface {
	configPointers() (identityContent, soulContent, agentsContent, userContent, toolsContent *string,
		model *string, maxTokens *int, temperature *float64, maxIterations *int, historyMessages *int,
		skillsList *[]string, toolsList *[]string, enableThinkingProcess *bool)
}

// updatePatch adapts UpdateAgentCommand to agentConfigPatchFields
type updatePatch UpdateAgentCommand

func (p updatePatch) configPointers() (identityContent, soulContent, agentsContent, userContent, toolsContent *string,
	model *string, maxTokens *int, temperature *float64, maxIterations *int, historyMessages *int,
	skillsList *[]string, toolsList *[]string, enableThinkingProcess *bool) {
	return p.IdentityContent, p.SoulContent, p.AgentsContent, p.UserContent, p.ToolsContent,
		p.Model, p.MaxTokens, p.Temperature, p.MaxIterations, p.HistoryMessages,
		p.SkillsList, p.ToolsList, p.EnableThinkingProcess
}

// patchCmd adapts PatchAgentCommand to agentConfigPatchFields
type patchCmd PatchAgentCommand

func (p patchCmd) configPointers() (identityContent, soulContent, agentsContent, userContent, toolsContent *string,
	model *string, maxTokens *int, temperature *float64, maxIterations *int, historyMessages *int,
	skillsList *[]string, toolsList *[]string, enableThinkingProcess *bool) {
	return p.IdentityContent, p.SoulContent, p.AgentsContent, p.UserContent, p.ToolsContent,
		p.Model, p.MaxTokens, p.Temperature, p.MaxIterations, p.HistoryMessages,
		p.SkillsList, p.ToolsList, p.EnableThinkingProcess
}

// hasConfigField 检查是否有任意一个配置字段被提供
func hasConfigField(p agentConfigPatchFields) bool {
	ic, sc, ac, uc, tc, m, mt, t, mi, hm, sl, tl, etp := p.configPointers()
	return ic != nil || sc != nil || ac != nil || uc != nil || tc != nil ||
		m != nil || mt != nil || t != nil || mi != nil || hm != nil ||
		sl != nil || tl != nil || etp != nil
}

// buildAgentConfigUpdate 从 patch 字段构建完整的 AgentConfigUpdate
func buildAgentConfigUpdate(agent *domain.Agent, p agentConfigPatchFields) domain.AgentConfigUpdate {
	ic, sc, ac, uc, tc, m, mt, t, mi, hm, sl, tl, etp := p.configPointers()
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
	if ic != nil {
		cfg.IdentityContent = *ic
	}
	if sc != nil {
		cfg.SoulContent = *sc
	}
	if ac != nil {
		cfg.AgentsContent = *ac
	}
	if uc != nil {
		cfg.UserContent = *uc
	}
	if tc != nil {
		cfg.ToolsContent = *tc
	}
	if m != nil {
		cfg.Model = *m
	}
	if mt != nil {
		cfg.MaxTokens = *mt
	}
	if t != nil {
		cfg.Temperature = *t
	}
	if mi != nil {
		cfg.MaxIterations = *mi
	}
	if hm != nil {
		cfg.HistoryMessages = *hm
	}
	if sl != nil {
		cfg.SkillsList = *sl
	}
	if tl != nil {
		cfg.ToolsList = *tl
	}
	if etp != nil {
		cfg.EnableThinkingProcess = *etp
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

	p := updatePatch(cmd)
	if hasConfigField(p) {
		agent.UpdateConfig(buildAgentConfigUpdate(agent, p))
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
	if cmd.OpenCodeConfig != nil {
		agent.UpdateOpenCodeConfig(cmd.OpenCodeConfig)
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

	p := patchCmd(cmd)
	if hasConfigField(p) {
		agent.UpdateConfig(buildAgentConfigUpdate(agent, p))
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
	if cmd.OpenCodeConfig != nil {
		agent.UpdateOpenCodeConfig(cmd.OpenCodeConfig)
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
