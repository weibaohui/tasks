package application

import (
	"context"
	"errors"
	"fmt"
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
	IsActive              *bool
	IsDefault             bool
	EnableThinkingProcess bool
	ClaudeCodeConfig      *domain.ClaudeCodeConfig
	OpenCodeConfig        *domain.OpenCodeConfig
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
	ClaudeCodeConfig      *domain.ClaudeCodeConfig
	OpenCodeConfig        *domain.OpenCodeConfig
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
	cmd.Model = strings.TrimSpace(cmd.Model)
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

	agent.UpdateConfig(domain.AgentConfigUpdate{
		IdentityContent:       cmd.IdentityContent,
		SoulContent:           cmd.SoulContent,
		AgentsContent:         cmd.AgentsContent,
		UserContent:           cmd.UserContent,
		ToolsContent:          cmd.ToolsContent,
		Model:                 cmd.Model,
		MaxTokens:             cmd.MaxTokens,
		Temperature:           cmd.Temperature,
		MaxIterations:         cmd.MaxIterations,
		HistoryMessages:       cmd.HistoryMessages,
		SkillsList:            cmd.SkillsList,
		ToolsList:             cmd.ToolsList,
		EnableThinkingProcess: cmd.EnableThinkingProcess,
	})

	if cmd.ClaudeCodeConfig != nil {
		agent.UpdateClaudeCodeConfig(cmd.ClaudeCodeConfig)
	}
	if cmd.OpenCodeConfig != nil {
		agent.UpdateOpenCodeConfig(cmd.OpenCodeConfig)
	}

	if cmd.IsActive != nil {
		agent.SetActive(*cmd.IsActive)
	}
	if cmd.IsDefault {
		agent.SetDefault(true)
	}
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
