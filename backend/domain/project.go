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
	id            ProjectID
	name          string
	gitRepoURL    string
	defaultBranch string
	initSteps     []string
	createdAt     time.Time
	updatedAt     time.Time
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
		id:            id,
		name:          name,
		gitRepoURL:    gitRepoURL,
		defaultBranch: defaultBranch,
		initSteps:     append([]string(nil), initSteps...),
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

func (p *Project) ID() ProjectID         { return p.id }
func (p *Project) Name() string          { return p.name }
func (p *Project) GitRepoURL() string    { return p.gitRepoURL }
func (p *Project) DefaultBranch() string { return p.defaultBranch }
func (p *Project) InitSteps() []string   { return append([]string(nil), p.initSteps...) }
func (p *Project) CreatedAt() time.Time  { return p.createdAt }
func (p *Project) UpdatedAt() time.Time  { return p.updatedAt }

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

type ProjectSnapshot struct {
	ID            ProjectID
	Name          string
	GitRepoURL    string
	DefaultBranch string
	InitSteps     []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (p *Project) ToSnapshot() ProjectSnapshot {
	return ProjectSnapshot{
		ID:            p.id,
		Name:          p.name,
		GitRepoURL:    p.gitRepoURL,
		DefaultBranch: p.defaultBranch,
		InitSteps:     append([]string(nil), p.initSteps...),
		CreatedAt:     p.createdAt,
		UpdatedAt:     p.updatedAt,
	}
}

func (p *Project) FromSnapshot(s ProjectSnapshot) {
	p.id = s.ID
	p.name = s.Name
	p.gitRepoURL = s.GitRepoURL
	p.defaultBranch = s.DefaultBranch
	p.initSteps = append([]string(nil), s.InitSteps...)
	p.createdAt = s.CreatedAt
	p.updatedAt = s.UpdatedAt
}
