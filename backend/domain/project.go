package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrProjectIDRequired      = errors.New("project id is required")
	ErrProjectNameRequired    = errors.New("project name is required")
	ErrProjectRepoURLRequired = errors.New("project git repo url is required")
)

type ProjectID struct {
	value string
}

func NewProjectID(value string) ProjectID {
	return ProjectID{value: value}
}

func (id ProjectID) String() string {
	return id.value
}

type Project struct {
	id                       ProjectID
	name                     string
	gitRepoURL               string
	defaultBranch            string
	initSteps                []string
	heartbeatEnabled         bool
	heartbeatIntervalMinutes  int
	heartbeatMDContent       string
	heartbeatAgentID         string
	dispatchChannelCode      string
	dispatchSessionKey       string
	createdAt                time.Time
	updatedAt                time.Time
}

func NewProject(id ProjectID, name, gitRepoURL, defaultBranch string, initSteps []string) (*Project, error) {
	if id.String() == "" {
		return nil, ErrProjectIDRequired
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrProjectNameRequired
	}
	if strings.TrimSpace(gitRepoURL) == "" {
		return nil, ErrProjectRepoURLRequired
	}
	if strings.TrimSpace(defaultBranch) == "" {
		defaultBranch = "main"
	}
	now := time.Now()
	return &Project{
		id:                       id,
		name:                     name,
		gitRepoURL:               gitRepoURL,
		defaultBranch:            defaultBranch,
		initSteps:                append([]string(nil), initSteps...),
		heartbeatEnabled:         false,
		heartbeatIntervalMinutes:  60,
		heartbeatMDContent:       "",
		heartbeatAgentID:         "",
		dispatchChannelCode:      "",
		dispatchSessionKey:       "",
		createdAt:                now,
		updatedAt:                now,
	}, nil
}

func (p *Project) ID() ProjectID            { return p.id }
func (p *Project) Name() string             { return p.name }
func (p *Project) GitRepoURL() string       { return p.gitRepoURL }
func (p *Project) DefaultBranch() string    { return p.defaultBranch }
func (p *Project) InitSteps() []string      { return append([]string(nil), p.initSteps...) }
func (p *Project) HeartbeatEnabled() bool    { return p.heartbeatEnabled }
func (p *Project) HeartbeatIntervalMinutes() int { return p.heartbeatIntervalMinutes }
func (p *Project) HeartbeatMDContent() string { return p.heartbeatMDContent }
func (p *Project) HeartbeatAgentID() string   { return p.heartbeatAgentID }
func (p *Project) DispatchChannelCode() string { return p.dispatchChannelCode }
func (p *Project) DispatchSessionKey() string { return p.dispatchSessionKey }
func (p *Project) CreatedAt() time.Time       { return p.createdAt }
func (p *Project) UpdatedAt() time.Time      { return p.updatedAt }

func (p *Project) Update(name, gitRepoURL, defaultBranch string, initSteps []string) error {
	if strings.TrimSpace(name) == "" {
		return ErrProjectNameRequired
	}
	if strings.TrimSpace(gitRepoURL) == "" {
		return ErrProjectRepoURLRequired
	}
	if strings.TrimSpace(defaultBranch) == "" {
		defaultBranch = "main"
	}
	p.name = name
	p.gitRepoURL = gitRepoURL
	p.defaultBranch = defaultBranch
	p.initSteps = append([]string(nil), initSteps...)
	p.updatedAt = time.Now()
	return nil
}

// UpdateHeartbeatConfig 更新心跳配置
func (p *Project) UpdateHeartbeatConfig(enabled bool, intervalMinutes int, mdContent, agentID string) {
	p.heartbeatEnabled = enabled
	p.heartbeatIntervalMinutes = intervalMinutes
	p.heartbeatMDContent = mdContent
	p.heartbeatAgentID = agentID
	p.updatedAt = time.Now()
}

type ProjectSnapshot struct {
	ID                        ProjectID
	Name                      string
	GitRepoURL                string
	DefaultBranch             string
	InitSteps                 []string
	HeartbeatEnabled          bool
	HeartbeatIntervalMinutes   int
	HeartbeatMDContent        string
	HeartbeatAgentID          string
	DispatchChannelCode       string
	DispatchSessionKey       string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

func (p *Project) ToSnapshot() ProjectSnapshot {
	return ProjectSnapshot{
		ID:                        p.id,
		Name:                      p.name,
		GitRepoURL:                p.gitRepoURL,
		DefaultBranch:             p.defaultBranch,
		InitSteps:                 append([]string(nil), p.initSteps...),
		HeartbeatEnabled:          p.heartbeatEnabled,
		HeartbeatIntervalMinutes:   p.heartbeatIntervalMinutes,
		HeartbeatMDContent:        p.heartbeatMDContent,
		HeartbeatAgentID:          p.heartbeatAgentID,
		DispatchChannelCode:       p.dispatchChannelCode,
		DispatchSessionKey:        p.dispatchSessionKey,
		CreatedAt:                 p.createdAt,
		UpdatedAt:                 p.updatedAt,
	}
}

func (p *Project) FromSnapshot(s ProjectSnapshot) {
	p.id = s.ID
	p.name = s.Name
	p.gitRepoURL = s.GitRepoURL
	p.defaultBranch = s.DefaultBranch
	p.initSteps = append([]string(nil), s.InitSteps...)
	p.heartbeatEnabled = s.HeartbeatEnabled
	p.heartbeatIntervalMinutes = s.HeartbeatIntervalMinutes
	p.heartbeatMDContent = s.HeartbeatMDContent
	p.heartbeatAgentID = s.HeartbeatAgentID
	p.createdAt = s.CreatedAt
	p.updatedAt = s.UpdatedAt
}
