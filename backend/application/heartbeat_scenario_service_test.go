package application

import (
	"context"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

// mockHeartbeatScenarioRepo 模拟 HeartbeatScenarioRepository
type mockHeartbeatScenarioRepo struct {
	scenarios map[string]*domain.HeartbeatScenario
}

func newMockHeartbeatScenarioRepo() *mockHeartbeatScenarioRepo {
	return &mockHeartbeatScenarioRepo{
		scenarios: make(map[string]*domain.HeartbeatScenario),
	}
}

func (m *mockHeartbeatScenarioRepo) Save(ctx context.Context, scenario *domain.HeartbeatScenario) error {
	m.scenarios[scenario.ID().String()] = scenario
	return nil
}

func (m *mockHeartbeatScenarioRepo) FindByID(ctx context.Context, id domain.HeartbeatScenarioID) (*domain.HeartbeatScenario, error) {
	return m.scenarios[id.String()], nil
}

func (m *mockHeartbeatScenarioRepo) FindByCode(ctx context.Context, code string) (*domain.HeartbeatScenario, error) {
	for _, s := range m.scenarios {
		if s.Code() == code {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockHeartbeatScenarioRepo) FindAll(ctx context.Context) ([]*domain.HeartbeatScenario, error) {
	var result []*domain.HeartbeatScenario
	for _, s := range m.scenarios {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockHeartbeatScenarioRepo) Delete(ctx context.Context, id domain.HeartbeatScenarioID) error {
	delete(m.scenarios, id.String())
	return nil
}

// mockHeartbeatRepo 模拟 HeartbeatRepository
type mockHeartbeatRepo struct {
	heartbeats map[string]*domain.Heartbeat
}

func newMockHeartbeatRepo() *mockHeartbeatRepo {
	return &mockHeartbeatRepo{
		heartbeats: make(map[string]*domain.Heartbeat),
	}
}

func (m *mockHeartbeatRepo) Save(ctx context.Context, hb *domain.Heartbeat) error {
	m.heartbeats[hb.ID().String()] = hb
	return nil
}

func (m *mockHeartbeatRepo) FindByID(ctx context.Context, id domain.HeartbeatID) (*domain.Heartbeat, error) {
	return m.heartbeats[id.String()], nil
}

func (m *mockHeartbeatRepo) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Heartbeat, error) {
	var result []*domain.Heartbeat
	for _, hb := range m.heartbeats {
		if hb.ProjectID().String() == projectID.String() {
			result = append(result, hb)
		}
	}
	return result, nil
}

func (m *mockHeartbeatRepo) FindAllEnabled(ctx context.Context) ([]*domain.Heartbeat, error) {
	var result []*domain.Heartbeat
	for _, hb := range m.heartbeats {
		if hb.Enabled() {
			result = append(result, hb)
		}
	}
	return result, nil
}

func (m *mockHeartbeatRepo) Delete(ctx context.Context, id domain.HeartbeatID) error {
	delete(m.heartbeats, id.String())
	return nil
}

// mockWebhookHeartbeatBindingRepo 模拟 WebhookHeartbeatBindingRepository
type mockWebhookHeartbeatBindingRepo struct{}

func (m *mockWebhookHeartbeatBindingRepo) Save(ctx context.Context, binding *domain.WebhookHeartbeatBinding) error {
	return nil
}

func (m *mockWebhookHeartbeatBindingRepo) FindByID(ctx context.Context, id domain.WebhookHeartbeatBindingID) (*domain.WebhookHeartbeatBinding, error) {
	return nil, nil
}

func (m *mockWebhookHeartbeatBindingRepo) FindByConfigID(ctx context.Context, configID domain.GitHubWebhookConfigID) ([]*domain.WebhookHeartbeatBinding, error) {
	return nil, nil
}

func (m *mockWebhookHeartbeatBindingRepo) FindByConfigIDAndEventType(ctx context.Context, configID domain.GitHubWebhookConfigID, eventType string) ([]*domain.WebhookHeartbeatBinding, error) {
	return nil, nil
}

func (m *mockWebhookHeartbeatBindingRepo) Delete(ctx context.Context, id domain.WebhookHeartbeatBindingID) error {
	return nil
}

func (m *mockWebhookHeartbeatBindingRepo) DeleteByHeartbeatID(ctx context.Context, heartbeatID domain.HeartbeatID) error {
	return nil
}

func setupTestScenarioSvc() (*HeartbeatScenarioService, *mockHeartbeatScenarioRepo, *sharedMockProjectRepo, *mockHeartbeatRepo, *mockIDGenerator) {
	scenarioRepo := newMockHeartbeatScenarioRepo()
	projectRepo := newSharedMockProjectRepo()
	heartbeatRepo := newMockHeartbeatRepo()
	bindingRepo := &mockWebhookHeartbeatBindingRepo{}
	idGen := &mockIDGenerator{}
	svc := NewHeartbeatScenarioService(scenarioRepo, projectRepo, heartbeatRepo, bindingRepo, idGen, nil)
	return svc, scenarioRepo, projectRepo, heartbeatRepo, idGen
}

func TestHeartbeatScenarioService_CreateScenario(t *testing.T) {
	svc, _, _, _, _ := setupTestScenarioSvc()
	ctx := context.Background()

	scenario, err := svc.CreateScenario(ctx, "test_code", "测试场景", "描述", []domain.HeartbeatScenarioItem{
		{Name: "心跳1", IntervalMinutes: 30, MDContent: "md", AgentCode: "agent", RequirementType: "normal"},
	}, true)
	if err != nil {
		t.Fatalf("创建场景失败: %v", err)
	}
	if scenario.Code() != "test_code" {
		t.Errorf("期望Code test_code, 实际 %s", scenario.Code())
	}
	if scenario.Name() != "测试场景" {
		t.Errorf("期望Name 测试场景, 实际 %s", scenario.Name())
	}
}

func TestHeartbeatScenarioService_GetScenarioByCode(t *testing.T) {
	svc, _, _, _, _ := setupTestScenarioSvc()
	ctx := context.Background()

	_, err := svc.CreateScenario(ctx, "test_code", "测试场景", "描述", []domain.HeartbeatScenarioItem{}, true)
	if err != nil {
		t.Fatalf("创建场景失败: %v", err)
	}

	scenario, err := svc.GetScenarioByCode(ctx, "test_code")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if scenario == nil {
		t.Fatal("期望找到场景")
	}
	if scenario.Name() != "测试场景" {
		t.Errorf("期望Name 测试场景, 实际 %s", scenario.Name())
	}

	notFound, err := svc.GetScenarioByCode(ctx, "non_existent")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if notFound != nil {
		t.Error("期望未找到场景")
	}
}

func TestHeartbeatScenarioService_ListScenarios(t *testing.T) {
	svc, _, _, _, _ := setupTestScenarioSvc()
	ctx := context.Background()

	svc.CreateScenario(ctx, "code1", "场景1", "", []domain.HeartbeatScenarioItem{}, true)
	svc.CreateScenario(ctx, "code2", "场景2", "", []domain.HeartbeatScenarioItem{}, true)

	scenarios, err := svc.ListScenarios(ctx)
	if err != nil {
		t.Fatalf("列出失败: %v", err)
	}
	if len(scenarios) != 2 {
		t.Errorf("期望2个场景, 实际 %d", len(scenarios))
	}
}

func TestHeartbeatScenarioService_DeleteScenario(t *testing.T) {
	svc, _, _, _, _ := setupTestScenarioSvc()
	ctx := context.Background()

	scenario, _ := svc.CreateScenario(ctx, "del_code", "待删除", "", []domain.HeartbeatScenarioItem{}, true)

	err := svc.DeleteScenario(ctx, scenario.ID().String())
	if err != nil {
		t.Fatalf("删除失败: %v", err)
	}

	notFound, _ := svc.GetScenarioByCode(ctx, "del_code")
	if notFound != nil {
		t.Error("期望已删除")
	}
}

func TestHeartbeatScenarioService_DeleteBuiltInScenario(t *testing.T) {
	svc, scenarioRepo, _, _, _ := setupTestScenarioSvc()
	ctx := context.Background()

	scenario, _ := domain.NewHeartbeatScenario(domain.NewHeartbeatScenarioID("builtin"), "builtin_code", "内置", "", []domain.HeartbeatScenarioItem{})
	scenario.SetIsBuiltIn(true)
	scenarioRepo.Save(ctx, scenario)

	err := svc.DeleteScenario(ctx, scenario.ID().String())
	if err == nil {
		t.Fatal("期望删除内置场景失败")
	}
}

func TestHeartbeatScenarioService_ApplyScenarioToProject(t *testing.T) {
	_, _, projectRepo, heartbeatRepo, _ := setupTestScenarioSvc()
	ctx := context.Background()

	// 创建项目
	project, _ := domain.NewProject(domain.NewProjectID("proj-001"), "测试项目", "https://github.com/test/repo.git", "main", []string{})
	projectRepo.Save(ctx, project)

	// 创建场景
	scenario, _ := domain.NewHeartbeatScenario(
		domain.NewHeartbeatScenarioID("sc-001"),
		"dev_workflow",
		"开发工作流",
		"",
		[]domain.HeartbeatScenarioItem{
			{Name: "心跳1", IntervalMinutes: 30, MDContent: "md1", AgentCode: "agent1", RequirementType: "normal"},
			{Name: "心跳2", IntervalMinutes: 60, MDContent: "md2", AgentCode: "agent2", RequirementType: "heartbeat"},
		},
	)
	scenarioRepo := newMockHeartbeatScenarioRepo()
	scenarioRepo.Save(ctx, scenario)

	// 重新构造 service 使用新的 scenarioRepo
	svc2 := NewHeartbeatScenarioService(scenarioRepo, projectRepo, heartbeatRepo, &mockWebhookHeartbeatBindingRepo{}, &mockIDGenerator{}, nil)

	err := svc2.ApplyScenarioToProject(ctx, "proj-001", "dev_workflow")
	if err != nil {
		t.Fatalf("应用场景失败: %v", err)
	}

	// 验证项目下心跳数量
	hbs, _ := heartbeatRepo.FindByProjectID(ctx, domain.NewProjectID("proj-001"))
	if len(hbs) != 2 {
		t.Errorf("期望2条心跳, 实际 %d", len(hbs))
	}

	// 验证项目场景编码已更新
	updatedProject, _ := projectRepo.FindByID(ctx, domain.NewProjectID("proj-001"))
	if updatedProject.HeartbeatScenarioCode() != "dev_workflow" {
		t.Errorf("期望项目场景编码 dev_workflow, 实际 %s", updatedProject.HeartbeatScenarioCode())
	}
}

func TestHeartbeatScenarioService_ApplyScenarioToProject_ReplaceExisting(t *testing.T) {
	_, _, projectRepo, heartbeatRepo, idGen := setupTestScenarioSvc()
	ctx := context.Background()

	project, _ := domain.NewProject(domain.NewProjectID("proj-001"), "测试项目", "https://github.com/test/repo.git", "main", []string{})
	projectRepo.Save(ctx, project)

	// 先添加旧心跳
	oldHb, _ := domain.NewHeartbeat(domain.NewHeartbeatID("old-hb"), domain.NewProjectID("proj-001"), "旧心跳", 30, "md", "agent", "normal")
	heartbeatRepo.Save(ctx, oldHb)

	scenario, _ := domain.NewHeartbeatScenario(
		domain.NewHeartbeatScenarioID("sc-001"),
		"dev_workflow",
		"开发工作流",
		"",
		[]domain.HeartbeatScenarioItem{
			{Name: "新心跳", IntervalMinutes: 45, MDContent: "new_md", AgentCode: "new_agent", RequirementType: "normal"},
		},
	)
	scenarioRepo := newMockHeartbeatScenarioRepo()
	scenarioRepo.Save(ctx, scenario)

	svc2 := NewHeartbeatScenarioService(scenarioRepo, projectRepo, heartbeatRepo, &mockWebhookHeartbeatBindingRepo{}, idGen, nil)

	err := svc2.ApplyScenarioToProject(ctx, "proj-001", "dev_workflow")
	if err != nil {
		t.Fatalf("应用场景失败: %v", err)
	}

	hbs, _ := heartbeatRepo.FindByProjectID(ctx, domain.NewProjectID("proj-001"))
	if len(hbs) != 1 {
		t.Errorf("期望1条心跳(旧被替换), 实际 %d", len(hbs))
	}
	if hbs[0].Name() != "开发工作流 - 新心跳" {
		t.Errorf("期望新心跳名称, 实际 %s", hbs[0].Name())
	}
}

func TestHeartbeatScenarioService_ApplyScenarioToProject_ProjectNotFound(t *testing.T) {
	svc, _, _, _, _ := setupTestScenarioSvc()
	ctx := context.Background()

	err := svc.ApplyScenarioToProject(ctx, "non-existent", "dev_workflow")
	if err == nil {
		t.Fatal("期望项目不存在错误")
	}
}

func TestHeartbeatScenarioService_ApplyScenarioToProject_ScenarioNotFound(t *testing.T) {
	svc, _, projectRepo, _, _ := setupTestScenarioSvc()
	ctx := context.Background()

	project, _ := domain.NewProject(domain.NewProjectID("proj-001"), "测试项目", "https://github.com/test/repo.git", "main", []string{})
	projectRepo.Save(ctx, project)

	err := svc.ApplyScenarioToProject(ctx, "proj-001", "non-existent")
	if err == nil {
		t.Fatal("期望场景不存在错误")
	}
}

func TestHeartbeatScenarioService_EnsureBuiltInScenarios(t *testing.T) {
	svc, scenarioRepo, _, _, _ := setupTestScenarioSvc()
	ctx := context.Background()

	err := svc.EnsureBuiltInScenarios(ctx)
	if err != nil {
		t.Fatalf("确保内置场景失败: %v", err)
	}

	scenario, _ := scenarioRepo.FindByCode(ctx, "github_dev_workflow")
	if scenario == nil {
		t.Fatal("期望找到内置场景")
	}
	if !scenario.IsBuiltIn() {
		t.Error("期望是内置场景")
	}
	if len(scenario.Items()) != 8 {
		t.Errorf("期望8个心跳项, 实际 %d", len(scenario.Items()))
	}

	// 再次调用应幂等
	err = svc.EnsureBuiltInScenarios(ctx)
	if err != nil {
		t.Fatalf("幂等调用失败: %v", err)
	}
	scenarios, _ := scenarioRepo.FindAll(ctx)
	if len(scenarios) != 1 {
		t.Errorf("期望仍只有1个内置场景, 实际 %d", len(scenarios))
	}
}
