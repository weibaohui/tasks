package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
)

// MockHeartbeatRepository Heartbeat仓库模拟
type MockHeartbeatRepository struct {
	heartbeats []*domain.Heartbeat
}

func (m *MockHeartbeatRepository) Save(ctx context.Context, hb *domain.Heartbeat) error {
	for i, existing := range m.heartbeats {
		if existing.ID().String() == hb.ID().String() {
			m.heartbeats[i] = hb
			return nil
		}
	}
	m.heartbeats = append(m.heartbeats, hb)
	return nil
}

func (m *MockHeartbeatRepository) FindByID(ctx context.Context, id domain.HeartbeatID) (*domain.Heartbeat, error) {
	for _, hb := range m.heartbeats {
		if hb.ID().String() == id.String() {
			return hb, nil
		}
	}
	return nil, nil
}

func (m *MockHeartbeatRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Heartbeat, error) {
	var result []*domain.Heartbeat
	for _, hb := range m.heartbeats {
		if hb.ProjectID().String() == projectID.String() {
			result = append(result, hb)
		}
	}
	return result, nil
}

func (m *MockHeartbeatRepository) FindAllEnabled(ctx context.Context) ([]*domain.Heartbeat, error) {
	var result []*domain.Heartbeat
	for _, hb := range m.heartbeats {
		if hb.Enabled() {
			result = append(result, hb)
		}
	}
	return result, nil
}

func (m *MockHeartbeatRepository) Delete(ctx context.Context, id domain.HeartbeatID) error {
	for i, hb := range m.heartbeats {
		if hb.ID().String() == id.String() {
			m.heartbeats = append(m.heartbeats[:i], m.heartbeats[i+1:]...)
			return nil
		}
	}
	return nil
}

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

func (m *MockRequirementRepository) FindByTraceID(ctx context.Context, traceID string) (*domain.Requirement, error) {
	for _, r := range m.requirements {
		if r.TraceID() == traceID {
			return r, nil
		}
	}
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

	hb, err := domain.NewHeartbeat(
		domain.NewHeartbeatID("hb-001"),
		project.ID(),
		"测试心跳",
		5,
		"心跳内容: ${project.name}",
		"agent-001",
		"heartbeat",
	)
	if err != nil {
		t.Fatalf("创建心跳失败: %v", err)
	}

	hbRepo := &MockHeartbeatRepository{heartbeats: []*domain.Heartbeat{hb}}
	projectRepo := &MockProjectRepository{projects: []*domain.Project{project}}
	agentRepo := &MockAgentRepository{agents: []*domain.Agent{}}
	reqRepo := &MockRequirementRepository{requirements: []*domain.Requirement{}}
	idGen := &MockIDGenerator{id: "req-heartbeat-001"}

	scheduler := NewHeartbeatScheduler(
		hbRepo,
		projectRepo,
		agentRepo,
		reqRepo,
		idGen,
		nil,
		nil,
		nil,
	)

	// 执行心跳
	scheduler.executeHeartbeat(context.Background(), "hb-001")

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
	if !strings.Contains(req.Description(), "测试项目") {
		t.Fatalf("期望需求描述包含项目名称，实际为: %s", req.Description())
	}
}

// TestHeartbeatScheduler_ExecuteHeartbeat_PrReviewType 测试 PR Review 类型心跳
func TestHeartbeatScheduler_ExecuteHeartbeat_PrReviewType(t *testing.T) {
	project, _ := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"测试项目",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)

	hb, _ := domain.NewHeartbeat(
		domain.NewHeartbeatID("hb-002"),
		project.ID(),
		"PR检查",
		5,
		"检查PR",
		"agent-001",
		"pr_review",
	)

	hbRepo := &MockHeartbeatRepository{heartbeats: []*domain.Heartbeat{hb}}
	projectRepo := &MockProjectRepository{projects: []*domain.Project{project}}
	agentRepo := &MockAgentRepository{}
	reqRepo := &MockRequirementRepository{}
	idGen := &MockIDGenerator{id: "req-pr-001"}

	scheduler := NewHeartbeatScheduler(
		hbRepo,
		projectRepo,
		agentRepo,
		reqRepo,
		idGen,
		nil,
		nil,
		nil,
	)

	scheduler.executeHeartbeat(context.Background(), "hb-002")

	requirements := reqRepo.requirements
	if len(requirements) != 1 {
		t.Fatalf("期望创建1个需求，实际创建了 %d 个", len(requirements))
	}

	req := requirements[0]
	if req.RequirementType() != domain.RequirementType("pr_review") {
		t.Fatalf("期望需求类型为 pr_review，实际为: %s", req.RequirementType())
	}
}

// TestHeartbeatScheduler_RefreshSchedule 测试刷新调度
func TestHeartbeatScheduler_RefreshSchedule(t *testing.T) {
	project, _ := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"测试项目",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)

	hb, _ := domain.NewHeartbeat(
		domain.NewHeartbeatID("hb-001"),
		project.ID(),
		"测试心跳",
		5,
		"内容",
		"agent-001",
		"heartbeat",
	)

	hbRepo := &MockHeartbeatRepository{heartbeats: []*domain.Heartbeat{hb}}
	projectRepo := &MockProjectRepository{projects: []*domain.Project{project}}
	scheduler := NewHeartbeatScheduler(
		hbRepo,
		projectRepo,
		nil,
		nil,
		&MockIDGenerator{},
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("启动调度器失败: %v", err)
	}

	// 验证已注册
	if _, exists := scheduler.entries["hb-001"]; !exists {
		t.Fatal("期望 hb-001 已注册")
	}

	// 更新间隔并刷新
	hb.Update("测试心跳", 10, "内容", "agent-001", "heartbeat")
	if err := scheduler.RefreshSchedule(ctx, "hb-001"); err != nil {
		t.Fatalf("刷新调度失败: %v", err)
	}

	if _, exists := scheduler.entries["hb-001"]; !exists {
		t.Fatal("刷新后期望 hb-001 仍已注册")
	}

	// 禁用后刷新
	hb.SetEnabled(false)
	if err := scheduler.RefreshSchedule(ctx, "hb-001"); err != nil {
		t.Fatalf("刷新调度失败: %v", err)
	}

	if _, exists := scheduler.entries["hb-001"]; exists {
		t.Fatal("禁用后期望 hb-001 未注册")
	}

	scheduler.Stop()
}

// TestHeartbeatScheduler_RenderTemplate 测试模板渲染
func TestHeartbeatScheduler_RenderTemplate(t *testing.T) {
	project, _ := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"测试项目",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)

	projectRepo := &MockProjectRepository{projects: []*domain.Project{project}}
	scheduler := NewHeartbeatScheduler(
		nil,
		projectRepo,
		nil,
		nil,
		&MockIDGenerator{},
		nil,
		nil,
		nil,
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
