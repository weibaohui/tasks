package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/weibh/taskmanager/domain"
)

var (
	ErrAgentNotFound       = errors.New("agent not found")
	ErrAgentCodeDuplicated = errors.New("agent code already exists")
)

type CreateAgentCommand struct {
	UserCode              string
	Name                  string
	AgentType             string
	Description           string
	IdentityContent       string
	SoulContent           string
	AgentsContent         string
	UserContent           string
	ToolsContent          string
	Model                 string
	LLMProviderID         *string
	MaxTokens             int
	Temperature           float64
	MaxIterations         int
	HistoryMessages       int
	SkillsList            []string
	ToolsList             []string
	IsDefault             bool
	EnableThinkingProcess bool
}

type UpdateAgentCommand struct {
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
}

type AgentApplicationService struct {
	agentRepo   domain.AgentRepository
	idGenerator domain.IDGenerator
}

func NewAgentApplicationService(
	agentRepo domain.AgentRepository,
	idGenerator domain.IDGenerator,
) *AgentApplicationService {
	return &AgentApplicationService{
		agentRepo:   agentRepo,
		idGenerator: idGenerator,
	}
}

// applyDefaultAgentCreateConfig 为新建 Agent 填充基础默认配置
func applyDefaultAgentCreateConfig(cmd *CreateAgentCommand) {
	if strings.TrimSpace(cmd.Description) == "" {
		cmd.Description = domain.DefaultAgentDescription
	}
	if strings.TrimSpace(cmd.IdentityContent) == "" {
		cmd.IdentityContent = domain.DefaultIdentityContent
	}
	if strings.TrimSpace(cmd.SoulContent) == "" {
		cmd.SoulContent = domain.DefaultSoulContent
	}
	if strings.TrimSpace(cmd.AgentsContent) == "" {
		cmd.AgentsContent = domain.DefaultAgentsContent
	}
	if strings.TrimSpace(cmd.UserContent) == "" {
		cmd.UserContent = domain.DefaultUserContent
	}
	if strings.TrimSpace(cmd.ToolsContent) == "" {
		cmd.ToolsContent = domain.DefaultToolsContent
	}
	if strings.TrimSpace(cmd.Model) == "" {
		cmd.Model = defaultAgentModelFromEnv()
	}
	if cmd.MaxTokens <= 0 {
		cmd.MaxTokens = domain.DefaultMaxTokens
	}
	if cmd.Temperature <= 0 {
		cmd.Temperature = domain.DefaultTemperature
	}
	if cmd.MaxIterations <= 0 {
		cmd.MaxIterations = domain.DefaultMaxIterations
	}
	if cmd.HistoryMessages <= 0 {
		cmd.HistoryMessages = domain.DefaultHistoryMessages
	}
	if cmd.SkillsList == nil {
		cmd.SkillsList = []string{}
	}
	if cmd.ToolsList == nil {
		cmd.ToolsList = []string{}
	}
}

// defaultAgentModelFromEnv 从环境变量中推断默认模型名称
func defaultAgentModelFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("LLM_MODEL")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("OPENAI_MODEL")); v != "" {
		return v
	}
	return domain.DefaultModel
}

func (s *AgentApplicationService) CreateAgent(ctx context.Context, cmd CreateAgentCommand) (*domain.Agent, error) {
	applyDefaultAgentCreateConfig(&cmd)

	agentCode := domain.NewAgentCode("agt_" + s.idGenerator.Generate())
	exists, err := s.agentRepo.FindByAgentCode(ctx, agentCode)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return nil, ErrAgentCodeDuplicated
	}

	agent, err := domain.NewAgent(
		domain.NewAgentID(s.idGenerator.Generate()),
		agentCode,
		cmd.UserCode,
		cmd.Name,
		cmd.Description,
		domain.AgentType(cmd.AgentType),
	)
	if err != nil {
		return nil, err
	}

	agent.UpdateConfig(
		cmd.IdentityContent,
		cmd.SoulContent,
		cmd.AgentsContent,
		cmd.UserContent,
		cmd.ToolsContent,
		cmd.Model,
		cmd.MaxTokens,
		cmd.Temperature,
		cmd.MaxIterations,
		cmd.HistoryMessages,
		cmd.SkillsList,
		cmd.ToolsList,
		cmd.EnableThinkingProcess,
	)
	agent.SetDefault(cmd.IsDefault)
	agent.ApplyLLMProvider(cmd.LLMProviderID)

	if err := s.agentRepo.Save(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to save agent: %w", err)
	}
	return agent, nil
}

func (s *AgentApplicationService) GetAgent(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
	agent, err := s.agentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	return agent, nil
}

func (s *AgentApplicationService) GetAgentByCode(ctx context.Context, code domain.AgentCode) (*domain.Agent, error) {
	agent, err := s.agentRepo.FindByAgentCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	return agent, nil
}

func (s *AgentApplicationService) ListAgents(ctx context.Context, userCode string) ([]*domain.Agent, error) {
	if userCode == "" {
		return s.agentRepo.FindAll(ctx)
	}
	return s.agentRepo.FindByUserCode(ctx, userCode)
}

func (s *AgentApplicationService) UpdateAgent(ctx context.Context, cmd UpdateAgentCommand) (*domain.Agent, error) {
	agent, err := s.agentRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}

	// 按需更新 profile 字段（仅检查 nil，空字符串由 domain 层校验）
	if cmd.Name != nil {
		desc := cmd.Description
		if desc == nil {
			d := agent.Description()
			desc = &d
		}
		if err := agent.UpdateProfile(*cmd.Name, *desc); err != nil {
			return nil, err
		}
	} else if cmd.Description != nil {
		if err := agent.UpdateProfile(agent.Name(), *cmd.Description); err != nil {
			return nil, err
		}
	}

	// 按需更新 config 字段
	hasConfigField := cmd.IdentityContent != nil || cmd.SoulContent != nil ||
		cmd.AgentsContent != nil || cmd.UserContent != nil || cmd.ToolsContent != nil ||
		cmd.Model != nil || cmd.MaxTokens != nil || cmd.Temperature != nil ||
		cmd.MaxIterations != nil || cmd.HistoryMessages != nil ||
		cmd.SkillsList != nil || cmd.ToolsList != nil ||
		cmd.EnableThinkingProcess != nil

	if hasConfigField {
		identityContent := agent.IdentityContent()
		soulContent := agent.SoulContent()
		agentsContent := agent.AgentsContent()
		userContent := agent.UserContent()
		toolsContent := agent.ToolsContent()
		model := agent.Model()
		maxTokens := agent.MaxTokens()
		temperature := agent.Temperature()
		maxIterations := agent.MaxIterations()
		historyMessages := agent.HistoryMessages()
		skillsList := agent.SkillsList()
		toolsList := agent.ToolsList()
		enableThinkingProcess := agent.EnableThinkingProcess()

		if cmd.IdentityContent != nil {
			identityContent = *cmd.IdentityContent
		}
		if cmd.SoulContent != nil {
			soulContent = *cmd.SoulContent
		}
		if cmd.AgentsContent != nil {
			agentsContent = *cmd.AgentsContent
		}
		if cmd.UserContent != nil {
			userContent = *cmd.UserContent
		}
		if cmd.ToolsContent != nil {
			toolsContent = *cmd.ToolsContent
		}
		if cmd.Model != nil {
			model = *cmd.Model
		}
		if cmd.MaxTokens != nil {
			maxTokens = *cmd.MaxTokens
		}
		if cmd.Temperature != nil {
			temperature = *cmd.Temperature
		}
		if cmd.MaxIterations != nil {
			maxIterations = *cmd.MaxIterations
		}
		if cmd.HistoryMessages != nil {
			historyMessages = *cmd.HistoryMessages
		}
		if cmd.SkillsList != nil {
			skillsList = *cmd.SkillsList
		}
		if cmd.ToolsList != nil {
			toolsList = *cmd.ToolsList
		}
		if cmd.EnableThinkingProcess != nil {
			enableThinkingProcess = *cmd.EnableThinkingProcess
		}

		agent.UpdateConfig(
			identityContent, soulContent, agentsContent, userContent, toolsContent,
			model, maxTokens, temperature, maxIterations, historyMessages,
			skillsList, toolsList, enableThinkingProcess,
		)
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

func (s *AgentApplicationService) DeleteAgent(ctx context.Context, id domain.AgentID) error {
	agent, err := s.agentRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if agent == nil {
		return ErrAgentNotFound
	}
	return s.agentRepo.Delete(ctx, id)
}

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

func (s *AgentApplicationService) PatchAgent(ctx context.Context, cmd PatchAgentCommand) (*domain.Agent, error) {
	agent, err := s.agentRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}

	// 按需更新 profile 字段（仅检查 nil，空字符串由 domain 层校验）
	if cmd.Name != nil {
		desc := cmd.Description
		if desc == nil {
			d := agent.Description()
			desc = &d
		}
		if err := agent.UpdateProfile(*cmd.Name, *desc); err != nil {
			return nil, err
		}
	} else if cmd.Description != nil {
		if err := agent.UpdateProfile(agent.Name(), *cmd.Description); err != nil {
			return nil, err
		}
	}

	// 按需更新 config 字段：只要任意一个 config 字段被提供，就构建完整参数调用 UpdateConfig
	configFields := []*bool{
		func() *bool { b := true; return &b }(), // 占位，表示有 config 字段
	}
	hasConfigField := cmd.IdentityContent != nil || cmd.SoulContent != nil ||
		cmd.AgentsContent != nil || cmd.UserContent != nil || cmd.ToolsContent != nil ||
		cmd.Model != nil || cmd.MaxTokens != nil || cmd.Temperature != nil ||
		cmd.MaxIterations != nil || cmd.HistoryMessages != nil ||
		cmd.SkillsList != nil || cmd.ToolsList != nil ||
		cmd.EnableThinkingProcess != nil
	_ = configFields // 避免 unused 警告

	if hasConfigField {
		identityContent := agent.IdentityContent()
		soulContent := agent.SoulContent()
		agentsContent := agent.AgentsContent()
		userContent := agent.UserContent()
		toolsContent := agent.ToolsContent()
		model := agent.Model()
		maxTokens := agent.MaxTokens()
		temperature := agent.Temperature()
		maxIterations := agent.MaxIterations()
		historyMessages := agent.HistoryMessages()
		skillsList := agent.SkillsList()
		toolsList := agent.ToolsList()
		enableThinkingProcess := agent.EnableThinkingProcess()

		if cmd.IdentityContent != nil {
			identityContent = *cmd.IdentityContent
		}
		if cmd.SoulContent != nil {
			soulContent = *cmd.SoulContent
		}
		if cmd.AgentsContent != nil {
			agentsContent = *cmd.AgentsContent
		}
		if cmd.UserContent != nil {
			userContent = *cmd.UserContent
		}
		if cmd.ToolsContent != nil {
			toolsContent = *cmd.ToolsContent
		}
		if cmd.Model != nil {
			model = *cmd.Model
		}
		if cmd.MaxTokens != nil {
			maxTokens = *cmd.MaxTokens
		}
		if cmd.Temperature != nil {
			temperature = *cmd.Temperature
		}
		if cmd.MaxIterations != nil {
			maxIterations = *cmd.MaxIterations
		}
		if cmd.HistoryMessages != nil {
			historyMessages = *cmd.HistoryMessages
		}
		if cmd.SkillsList != nil {
			skillsList = *cmd.SkillsList
		}
		if cmd.ToolsList != nil {
			toolsList = *cmd.ToolsList
		}
		if cmd.EnableThinkingProcess != nil {
			enableThinkingProcess = *cmd.EnableThinkingProcess
		}

		agent.UpdateConfig(
			identityContent, soulContent, agentsContent, userContent, toolsContent,
			model, maxTokens, temperature, maxIterations, historyMessages,
			skillsList, toolsList, enableThinkingProcess,
		)
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

func boolValue(ptr *bool, fallback bool) bool {
	if ptr == nil {
		return fallback
	}
	return *ptr
}
