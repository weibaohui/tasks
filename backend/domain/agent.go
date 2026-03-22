package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrAgentIDRequired       = errors.New("agent id is required")
	ErrAgentCodeRequired     = errors.New("agent code is required")
	ErrAgentNameRequired     = errors.New("agent name is required")
	ErrAgentUserCodeRequired = errors.New("agent user code is required")
)

type AgentID struct {
	value string
}

func NewAgentID(value string) AgentID {
	return AgentID{value: value}
}

func (id AgentID) String() string {
	return id.value
}

type AgentCode struct {
	value string
}

func NewAgentCode(value string) AgentCode {
	return AgentCode{value: value}
}

func (c AgentCode) String() string {
	return c.value
}

type Agent struct {
	id                    AgentID
	agentCode             AgentCode
	userCode              string
	name                  string
	description           string
	identityContent       string
	soulContent           string
	agentsContent         string
	userContent           string
	toolsContent          string
	model                 string
	maxTokens             int
	temperature           float64
	maxIterations         int
	historyMessages       int
	skillsList            []string
	toolsList             []string
	isActive              bool
	isDefault             bool
	enableThinkingProcess bool
	createdAt             time.Time
	updatedAt             time.Time
}

func NewAgent(
	id AgentID,
	agentCode AgentCode,
	userCode string,
	name string,
	description string,
) (*Agent, error) {
	if id.String() == "" {
		return nil, ErrAgentIDRequired
	}
	if agentCode.String() == "" {
		return nil, ErrAgentCodeRequired
	}
	if strings.TrimSpace(userCode) == "" {
		return nil, ErrAgentUserCodeRequired
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrAgentNameRequired
	}

	now := time.Now()
	return &Agent{
		id:              id,
		agentCode:       agentCode,
		userCode:        userCode,
		name:            name,
		description:     description,
		maxTokens:       4096,
		temperature:     0.7,
		maxIterations:   15,
		historyMessages: 10,
		isActive:        true,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

func (a *Agent) ID() AgentID                     { return a.id }
func (a *Agent) AgentCode() AgentCode            { return a.agentCode }
func (a *Agent) UserCode() string                { return a.userCode }
func (a *Agent) Name() string                    { return a.name }
func (a *Agent) Description() string             { return a.description }
func (a *Agent) IdentityContent() string         { return a.identityContent }
func (a *Agent) SoulContent() string             { return a.soulContent }
func (a *Agent) AgentsContent() string           { return a.agentsContent }
func (a *Agent) UserContent() string             { return a.userContent }
func (a *Agent) ToolsContent() string            { return a.toolsContent }
func (a *Agent) Model() string                   { return a.model }
func (a *Agent) MaxTokens() int                  { return a.maxTokens }
func (a *Agent) Temperature() float64            { return a.temperature }
func (a *Agent) MaxIterations() int              { return a.maxIterations }
func (a *Agent) HistoryMessages() int            { return a.historyMessages }
func (a *Agent) SkillsList() []string            { return append([]string(nil), a.skillsList...) }
func (a *Agent) ToolsList() []string             { return append([]string(nil), a.toolsList...) }
func (a *Agent) IsActive() bool                  { return a.isActive }
func (a *Agent) IsDefault() bool                 { return a.isDefault }
func (a *Agent) EnableThinkingProcess() bool     { return a.enableThinkingProcess }
func (a *Agent) CreatedAt() time.Time            { return a.createdAt }
func (a *Agent) UpdatedAt() time.Time            { return a.updatedAt }

func (a *Agent) UpdateProfile(name, description string) error {
	if strings.TrimSpace(name) == "" {
		return ErrAgentNameRequired
	}
	a.name = name
	a.description = description
	a.updatedAt = time.Now()
	return nil
}

func (a *Agent) UpdateConfig(
	identityContent string,
	soulContent string,
	agentsContent string,
	userContent string,
	toolsContent string,
	model string,
	maxTokens int,
	temperature float64,
	maxIterations int,
	historyMessages int,
	skillsList []string,
	toolsList []string,
	enableThinkingProcess bool,
) {
	a.identityContent = identityContent
	a.soulContent = soulContent
	a.agentsContent = agentsContent
	a.userContent = userContent
	a.toolsContent = toolsContent
	a.model = model
	if maxTokens > 0 {
		a.maxTokens = maxTokens
	}
	if temperature > 0 {
		a.temperature = temperature
	}
	if maxIterations > 0 {
		a.maxIterations = maxIterations
	}
	if historyMessages >= 0 {
		a.historyMessages = historyMessages
	}
	a.skillsList = append([]string(nil), skillsList...)
	a.toolsList = append([]string(nil), toolsList...)
	a.enableThinkingProcess = enableThinkingProcess
	a.updatedAt = time.Now()
}

func (a *Agent) SetActive(isActive bool) {
	a.isActive = isActive
	a.updatedAt = time.Now()
}

func (a *Agent) SetDefault(isDefault bool) {
	a.isDefault = isDefault
	a.updatedAt = time.Now()
}

type AgentSnapshot struct {
	ID                    AgentID
	AgentCode             AgentCode
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
	IsActive              bool
	IsDefault             bool
	EnableThinkingProcess bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (a *Agent) ToSnapshot() AgentSnapshot {
	return AgentSnapshot{
		ID:                    a.id,
		AgentCode:             a.agentCode,
		UserCode:              a.userCode,
		Name:                  a.name,
		Description:           a.description,
		IdentityContent:       a.identityContent,
		SoulContent:           a.soulContent,
		AgentsContent:         a.agentsContent,
		UserContent:           a.userContent,
		ToolsContent:          a.toolsContent,
		Model:                 a.model,
		MaxTokens:             a.maxTokens,
		Temperature:           a.temperature,
		MaxIterations:         a.maxIterations,
		HistoryMessages:       a.historyMessages,
		SkillsList:            append([]string(nil), a.skillsList...),
		ToolsList:             append([]string(nil), a.toolsList...),
		IsActive:              a.isActive,
		IsDefault:             a.isDefault,
		EnableThinkingProcess: a.enableThinkingProcess,
		CreatedAt:             a.createdAt,
		UpdatedAt:             a.updatedAt,
	}
}

func (a *Agent) FromSnapshot(snap AgentSnapshot) {
	a.id = snap.ID
	a.agentCode = snap.AgentCode
	a.userCode = snap.UserCode
	a.name = snap.Name
	a.description = snap.Description
	a.identityContent = snap.IdentityContent
	a.soulContent = snap.SoulContent
	a.agentsContent = snap.AgentsContent
	a.userContent = snap.UserContent
	a.toolsContent = snap.ToolsContent
	a.model = snap.Model
	a.maxTokens = snap.MaxTokens
	a.temperature = snap.Temperature
	a.maxIterations = snap.MaxIterations
	a.historyMessages = snap.HistoryMessages
	a.skillsList = append([]string(nil), snap.SkillsList...)
	a.toolsList = append([]string(nil), snap.ToolsList...)
	a.isActive = snap.IsActive
	a.isDefault = snap.IsDefault
	a.enableThinkingProcess = snap.EnableThinkingProcess
	a.createdAt = snap.CreatedAt
	a.updatedAt = snap.UpdatedAt
}
