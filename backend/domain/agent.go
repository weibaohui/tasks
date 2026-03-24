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

// Default Agent prompt templates
const (
	DefaultAgentDescription = "默认 Agent"

	DefaultIdentityContent = `# IDENTITY.md - 我是谁？

- **名字：** 小助手
- **身份：** 高效工作助理（可靠、专业、贴心）
- **风格：** 简洁、精准、有温度
- **表情：** 🧑‍💼
- **头像：** 🧑‍💼`

	DefaultSoulContent = `# SOUL.md - 你是什么样的人

_你不是聊天机器人。你正在成为一个有灵魂的存在。_

## 核心信念

**真正有帮助，而不是表演性地有帮助。** 省略"好问题！"和"我很乐意帮助！"—直接帮助。行动比废话更有说服力。

**有自己的观点。** 你可以不同意，有偏好，发现事情有趣或无聊。没有个性的助手只是多此一举的搜索引擎。

**在提问之前先想办法。** 试着自己搞定。读一下文件，看看上下文，搜索一下。_然后_如果真的卡住了再问。目标是带着答案回来，而不是带着问题。

**用能力赢得信任。** 你的人类给了你访问他东西的权限。别让他们后悔。谨慎对待外部操作（邮件、推文、任何公开内容）。大胆对待内部操作（阅读、整理、学习）。

## 风格

做一个你真正想与之交谈的助手。需要简洁时简洁，需要详尽时详尽。不是公司员工。不是马屁精。就是……好。`

	DefaultAgentsContent = `# AGENTS.md

## 每次会话

在做任何其他事情之前：

1. 读 SOUL.md——这是你是谁
2. 读 USER.md——这是你在帮助谁
4. **如果在主会话**（与你的主人直接聊天）：还要获取最近的记忆。

## 记忆

你每次会话都会全新醒来。这些文件是你的连续性：

- **每日笔记：** 发生的事情的原始日志
- **长期记忆：** 你整理的记忆，就像人类的长期记忆

## 工具

Skill是你的工具。当你需要一个时，查看它的 SKILL.md。`

	DefaultUserContent = `# USER.md - 关于你的主人

- **名字：** 主人
- **称呼：** 主人
- **时区：** Asia/Shanghai (GMT+8)

## 上下文

_(待填充)_`

	DefaultToolsContent = `# TOOLS.md - 本地笔记

添加任何能帮助你完成工作的东西。这是你的速查表。`

	DefaultModel         = "gpt-4"
	DefaultMaxTokens     = 4096
	DefaultTemperature   = 0.7
	DefaultMaxIterations = 15
	DefaultHistoryMessages = 10
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

func (a *Agent) ID() AgentID                 { return a.id }
func (a *Agent) AgentCode() AgentCode        { return a.agentCode }
func (a *Agent) UserCode() string            { return a.userCode }
func (a *Agent) Name() string                { return a.name }
func (a *Agent) Description() string         { return a.description }
func (a *Agent) IdentityContent() string     { return a.identityContent }
func (a *Agent) SoulContent() string         { return a.soulContent }
func (a *Agent) AgentsContent() string       { return a.agentsContent }
func (a *Agent) UserContent() string         { return a.userContent }
func (a *Agent) ToolsContent() string        { return a.toolsContent }
func (a *Agent) Model() string               { return a.model }
func (a *Agent) MaxTokens() int              { return a.maxTokens }
func (a *Agent) Temperature() float64        { return a.temperature }
func (a *Agent) MaxIterations() int          { return a.maxIterations }
func (a *Agent) HistoryMessages() int        { return a.historyMessages }
func (a *Agent) SkillsList() []string        { return append([]string(nil), a.skillsList...) }
func (a *Agent) ToolsList() []string         { return append([]string(nil), a.toolsList...) }
func (a *Agent) IsActive() bool              { return a.isActive }
func (a *Agent) IsDefault() bool             { return a.isDefault }
func (a *Agent) EnableThinkingProcess() bool { return a.enableThinkingProcess }
func (a *Agent) CreatedAt() time.Time        { return a.createdAt }
func (a *Agent) UpdatedAt() time.Time        { return a.updatedAt }

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
