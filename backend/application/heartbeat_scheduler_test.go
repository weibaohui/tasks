package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

// MockProjectRepository is a mock implementation of ProjectRepository
type MockProjectRepository struct {
	projects []*domain.Project
}

func (m *MockProjectRepository) FindAll(ctx context.Context) ([]*domain.Project, error) {
	return m.projects, nil
}

func (m *MockProjectRepository) FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	for _, p := range m.projects {
		if p.ID() == id {
			return p, nil
		}
	}
	return nil, nil
}

func (m *MockProjectRepository) Save(ctx context.Context, project *domain.Project) error {
	return nil
}

func (m *MockProjectRepository) Delete(ctx context.Context, id domain.ProjectID) error {
	return nil
}

// MockAgentRepository is a mock implementation of AgentRepository
type MockAgentRepository struct {
	agents []*domain.Agent
}

func (m *MockAgentRepository) FindAll(ctx context.Context) ([]*domain.Agent, error) {
	return m.agents, nil
}

func (m *MockAgentRepository) FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
	for _, a := range m.agents {
		if a.ID() == id {
			return a, nil
		}
	}
	return nil, nil
}

func (m *MockAgentRepository) FindByAgentCode(ctx context.Context, code domain.AgentCode) (*domain.Agent, error) {
	for _, a := range m.agents {
		if a.AgentCode() == code {
			return a, nil
		}
	}
	return nil, nil
}

func (m *MockAgentRepository) Save(ctx context.Context, agent *domain.Agent) error {
	return nil
}

func (m *MockAgentRepository) Delete(ctx context.Context, id domain.AgentID) error {
	return nil
}

// MockRequirementRepository is a mock implementation of RequirementRepository
type MockRequirementRepository struct {
	requirements []*domain.Requirement
}

func (m *MockRequirementRepository) FindAll(ctx context.Context) ([]*domain.Requirement, error) {
	return m.requirements, nil
}

func (m *MockRequirementRepository) FindByID(ctx context.Context, id domain.RequirementID) (*domain.Requirement, error) {
	for _, r := range m.requirements {
		if r.ID() == id {
			return r, nil
		}
	}
	return nil, nil
}

func (m *MockRequirementRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Requirement, error) {
	var result []*domain.Requirement
	for _, r := range m.requirements {
		if r.ProjectID() == projectID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MockRequirementRepository) Save(ctx context.Context, req *domain.Requirement) error {
	return nil
}

func (m *MockRequirementRepository) Delete(ctx context.Context, id domain.RequirementID) error {
	return nil
}

// MockIDGenerator is a mock implementation of IDGenerator
type MockIDGenerator struct {
	id string
}

func (m *MockIDGenerator) Generate() string {
	if m.id == "" {
		return "test-id-" + time.Now().Format("20060102150405")
	}
	return m.id
}

// MockInboundPublisher is a mock implementation of inbound publisher
type MockInboundPublisher struct {
	messages []*channelBus.InboundMessage
}

func (m *MockInboundPublisher) PublishInbound(msg *channelBus.InboundMessage) {
	m.messages = append(m.messages, msg)
}

// MockRequirementDispatchService is a mock implementation
type MockRequirementDispatchService struct {
	dispatched []DispatchRequirementCommand
}

func (m *MockRequirementDispatchService) DispatchRequirement(ctx context.Context, cmd DispatchRequirementCommand) (*DispatchRequirementResult, error) {
	m.dispatched = append(m.dispatched, cmd)
	return &DispatchRequirementResult{
		RequirementID:    "req-123",
		Status:           "dispatched",
		WorkspacePath:    "/tmp/workspace",
		ReplicaAgentCode: "test-agent",
		TaskID:           "task-123",
	}, nil
}

func createTestProject() *domain.Project {
	project, _ := domain.NewProject(
		domain.NewProjectID("project-123"),
		"TestProject",
		"git@github.com:test/repo.git",
		"main",
		[]string{},
	)
	project.UpdateHeartbeatConfig(true, 60, "Heartbeat content for ${project.name}", "agent-001")
	project.UpdateDispatchConfig("feishu", "test-session")
	return project
}

func TestRenderTemplate(t *testing.T) {
	scheduler := &HeartbeatScheduler{}
	project := createTestProject()

	tests := []struct {
		name     string
		template string
		check    func(string) bool
	}{
		{
			name:     "project id placeholder",
			template: "project id is ${project.id}",
			check:    func(s string) bool { return strings.Contains(s, project.ID().String()) },
		},
		{
			name:     "project name placeholder",
			template: "project name is ${project.name}",
			check:    func(s string) bool { return strings.Contains(s, project.Name()) },
		},
		{
			name:     "git repo url placeholder",
			template: "repo is ${project.git_repo_url}",
			check:    func(s string) bool { return strings.Contains(s, project.GitRepoURL()) },
		},
		{
			name:     "default branch placeholder",
			template: "branch is ${project.default_branch}",
			check:    func(s string) bool { return strings.Contains(s, project.DefaultBranch()) },
		},
		{
			name:     "timestamp placeholder",
			template: "time is ${timestamp}",
			check: func(s string) bool {
				// timestamp should be in format YYYY-MM-DD HH:MM:SS
				return strings.Contains(s, time.Now().Format("2006-01-02"))
			},
		},
		{
			name:     "multiple placeholders",
			template: "${project.id} - ${project.name} - ${timestamp}",
			check: func(s string) bool {
				return strings.Contains(s, project.ID().String()) &&
					strings.Contains(s, project.Name())
			},
		},
		{
			name:     "no placeholder",
			template: "static content",
			check:    func(s string) bool { return s == "static content" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scheduler.renderTemplate(tt.template, project)
			if !tt.check(result) {
				t.Errorf("renderTemplate(%q) = %q, want to pass check", tt.template, result)
			}
		})
	}
}
