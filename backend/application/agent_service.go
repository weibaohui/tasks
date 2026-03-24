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
	Description           string
	IdentityContent       string
	SoulContent           string
	AgentsContent         string
	UserContent           string
	ToolsContent          string
	Model                 string
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
	Name                  string
	Description           string
	IdentityContent       string
	SoulContent           string
	AgentsContent         string
	UserContent           string
	ToolsContent          string
	Model                 string
	MaxTokens             int
	Temperature           float64
	MaxIterations         int
	HistoryMessages       int
	SkillsList            []string
	ToolsList             []string
	IsActive              *bool
	IsDefault             *bool
	EnableThinkingProcess *bool
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

	if cmd.Name != "" {
		if err := agent.UpdateProfile(cmd.Name, cmd.Description); err != nil {
			return nil, err
		}
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
		boolValue(cmd.EnableThinkingProcess, agent.EnableThinkingProcess()),
	)
	if cmd.IsActive != nil {
		agent.SetActive(*cmd.IsActive)
	}
	if cmd.IsDefault != nil {
		agent.SetDefault(*cmd.IsDefault)
	}

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

func boolValue(ptr *bool, fallback bool) bool {
	if ptr == nil {
		return fallback
	}
	return *ptr
}
