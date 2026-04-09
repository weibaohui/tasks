package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
)

// MockProjectRepository 项目仓库模拟
type MockProjectRepository struct {
	projects []*domain.Project
}

func (m *MockProjectRepository) FindAll(ctx context.Context) ([]*domain.Project, error) {
	return m.projects, nil
}

func (m *MockProjectRepository) FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	for _, p := range m.projects {
		if p.ID().String() == id.String() {
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

// MockAgentRepository Agent仓库模拟
type MockAgentRepository struct {
	agents []*domain.Agent
}

func (m *MockAgentRepository) FindAll(ctx context.Context) ([]*domain.Agent, error) {
	return m.agents, nil
}

func (m *MockAgentRepository) FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
	return nil, nil
}

func (m *MockAgentRepository) FindByAgentCode(ctx context.Context, code domain.AgentCode) (*domain.Agent, error) {
	for _, a := range m.agents {
		if a.AgentCode().String() == code.String() {
			return a, nil
		}
	}
	return nil, nil
}

func (m *MockAgentRepository) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Agent, error) {
	return nil, nil
}

func (m *MockAgentRepository) Save(ctx context.Context, agent *domain.Agent) error {
	return nil
}

func (m *MockAgentRepository) Delete(ctx context.Context, id domain.AgentID) error {
	return nil
}

// MockRequirementRepository 需求仓库模拟
type MockRequirementRepository struct {
	requirements []*domain.Requirement
}

func (m *MockRequirementRepository) FindAll(ctx context.Context) ([]*domain.Requirement, error) {
	return m.requirements, nil
}

func (m *MockRequirementRepository) FindByID(ctx context.Context, id domain.RequirementID) (*domain.Requirement, error) {
	for _, r := range m.requirements {
		if r.ID().String() == id.String() {
			return r, nil
		}
	}
	return nil, nil
}

func (m *MockRequirementRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Requirement, error) {
	return nil, nil
}

func (m *MockRequirementRepository) Save(ctx context.Context, req *domain.Requirement) error {
	m.requirements = append(m.requirements, req)
	return nil
}

func (m *MockRequirementRepository) Delete(ctx context.Context, id domain.RequirementID) error {
	return nil
}

func (m *MockRequirementRepository) List(ctx context.Context, filter domain.RequirementListFilter) ([]*domain.Requirement, error) {
	return nil, nil
}

func (m *MockRequirementRepository) Count(ctx context.Context, filter domain.RequirementListFilter) (int, error) {
	return 0, nil
}

func (m *MockRequirementRepository) GetStatusStats(ctx context.Context, projectID *domain.ProjectID) ([]domain.StatusStat, error) {
	return nil, nil
}

// MockIDGenerator ID生成器模拟
type MockIDGenerator struct {
	id string
}

func (m *MockIDGenerator) Generate() string {
	if m.id == "" {
		return "test-id-" + time.Now().Format("150405")
	}
	return m.id
}

// TestHeartbeatScheduler_ExecuteHeartbeat 测试心跳执行
func TestHeartbeatScheduler_ExecuteHeartbeat(t *testing.T) {
	// 创建测试项目
	project, err := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"测试项目",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	// 启用心跳
	enabled := true
	interval := 5
	mdContent := "心跳内容: ${project.name}"
	agentCode := "agent-001"
	project.UpdateHeartbeatConfig(&enabled, &interval, &mdContent, &agentCode)

	projectRepo := &MockProjectRepository{projects: []*domain.Project{project}}
	agentRepo := &MockAgentRepository{agents: []*domain.Agent{}}
	reqRepo := &MockRequirementRepository{requirements: []*domain.Requirement{}}
	idGen := &MockIDGenerator{id: "req-heartbeat-001"}

	scheduler := NewHeartbeatScheduler(
		projectRepo,
		agentRepo,
		reqRepo,
		idGen,
		nil,
		nil,
		nil, // stateMachineService
	)

	// 执行心跳
	scheduler.executeHeartbeat("proj-001")

	// 验证需求是否被创建
	requirements := reqRepo.requirements
	if len(requirements) != 1 {
		t.Fatalf("期望创建1个需求，实际创建了 %d 个", len(requirements))
	}

	req := requirements[0]
	if !strings.Contains(req.Title(), "[心跳]") {
		t.Fatalf("期望需求标题包含 [心跳]，实际为: %s", req.Title())
	}
	if req.RequirementType() != domain.RequirementTypeHeartbeat {
		t.Fatalf("期望需求类型为心跳，实际为: %s", req.RequirementType())
	}
}

// TestHeartbeatScheduler_RenderTemplate 测试模板渲染
func TestHeartbeatScheduler_RenderTemplate(t *testing.T) {
	project, err := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"测试项目",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	projectRepo := &MockProjectRepository{projects: []*domain.Project{project}}
	agentRepo := &MockAgentRepository{agents: []*domain.Agent{}}
	reqRepo := &MockRequirementRepository{requirements: []*domain.Requirement{}}
	idGen := &MockIDGenerator{}

	scheduler := NewHeartbeatScheduler(
		projectRepo,
		agentRepo,
		reqRepo,
		idGen,
		nil,
		nil,
		nil, // stateMachineService
	)

	template := "项目: ${project.name}, ID: ${project.id}, 仓库: ${project.git_repo_url}, 分支: ${project.default_branch}, 时间: ${timestamp}"
	result := scheduler.renderTemplate(template, project)

	if !strings.Contains(result, "测试项目") {
		t.Fatalf("模板渲染失败，未找到项目名称")
	}
	if !strings.Contains(result, "proj-001") {
		t.Fatalf("模板渲染失败，未找到项目ID")
	}
	if !strings.Contains(result, "git@github.com:test/repo.git") {
		t.Fatalf("模板渲染失败，未找到仓库URL")
	}
	if !strings.Contains(result, "main") {
		t.Fatalf("模板渲染失败，未找到默认分支")
	}
	if !strings.Contains(result, "20") { // 年份
		t.Fatalf("模板渲染失败，未找到时间戳")
	}
}