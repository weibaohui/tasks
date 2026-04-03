package application

import (
	"context"
	"strconv"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

type mockRequirementRepo struct {
	requirements map[string]*domain.Requirement
}

func newMockRequirementRepo() *mockRequirementRepo {
	return &mockRequirementRepo{
		requirements: make(map[string]*domain.Requirement),
	}
}

func (m *mockRequirementRepo) Save(ctx context.Context, requirement *domain.Requirement) error {
	m.requirements[requirement.ID().String()] = requirement
	return nil
}

func (m *mockRequirementRepo) FindByID(ctx context.Context, id domain.RequirementID) (*domain.Requirement, error) {
	requirement, ok := m.requirements[id.String()]
	if !ok {
		return nil, nil
	}
	return requirement, nil
}

func (m *mockRequirementRepo) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Requirement, error) {
	var result []*domain.Requirement
	for _, req := range m.requirements {
		if req.ProjectID().String() == projectID.String() {
			result = append(result, req)
		}
	}
	return result, nil
}

func (m *mockRequirementRepo) FindAll(ctx context.Context) ([]*domain.Requirement, error) {
	var result []*domain.Requirement
	for _, req := range m.requirements {
		result = append(result, req)
	}
	return result, nil
}

func (m *mockRequirementRepo) Delete(ctx context.Context, id domain.RequirementID) error {
	delete(m.requirements, id.String())
	return nil
}

type mockRequirementIDGen struct {
	count int
}

func (m *mockRequirementIDGen) Generate() string {
	m.count++
	return "req-id-" + strconv.Itoa(m.count)
}

func setupTestRequirementSvc() (*RequirementApplicationService, *mockRequirementRepo, *sharedMockProjectRepo) {
	reqRepo := newMockRequirementRepo()
	projRepo := newSharedMockProjectRepo()
	idGen := &mockRequirementIDGen{}

	svc := NewRequirementApplicationService(reqRepo, projRepo, idGen, nil, nil)
	return svc, reqRepo, projRepo
}

func createTestProject(repo *sharedMockProjectRepo) *domain.Project {
	project, _ := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"Test Project",
		"https://github.com/test/repo.git",
		"main",
		[]string{"make setup"},
	)
	repo.projects["proj-001"] = project
	return project
}

func TestRequirementService_CreateRequirement(t *testing.T) {
	svc, _, projRepo := setupTestRequirementSvc()
	ctx := context.Background()

	// 先创建一个项目
	createTestProject(projRepo)

	requirement, err := svc.CreateRequirement(ctx, CreateRequirementCommand{
		ProjectID:          domain.NewProjectID("proj-001"),
		Title:              "测试需求",
		Description:        "测试描述",
		AcceptanceCriteria: "测试验收标准",
		TempWorkspaceRoot:  "/tmp/workspace",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if requirement.Title() != "测试需求" {
		t.Errorf("期望 title 为 '测试需求', 实际为 '%s'", requirement.Title())
	}

	if requirement.Description() != "测试描述" {
		t.Errorf("期望 description 为 '测试描述', 实际为 '%s'", requirement.Description())
	}

	if requirement.AcceptanceCriteria() != "测试验收标准" {
		t.Errorf("期望 acceptanceCriteria 为 '测试验收标准', 实际为 '%s'", requirement.AcceptanceCriteria())
	}

	if requirement.ProjectID().String() != "proj-001" {
		t.Errorf("期望 projectID 为 'proj-001', 实际为 '%s'", requirement.ProjectID().String())
	}

	if requirement.TempWorkspaceRoot() != "/tmp/workspace" {
		t.Errorf("期望 tempWorkspaceRoot 为 '/tmp/workspace', 实际为 '%s'", requirement.TempWorkspaceRoot())
	}

	if requirement.Status() != domain.RequirementStatusTodo {
		t.Errorf("期望 status 为 'todo', 实际为 '%s'", requirement.Status())
	}
}

func TestRequirementService_CreateRequirement_ProjectNotFound(t *testing.T) {
	svc, _, _ := setupTestRequirementSvc()
	ctx := context.Background()

	_, err := svc.CreateRequirement(ctx, CreateRequirementCommand{
		ProjectID:          domain.NewProjectID("non-existent"),
		Title:              "测试需求",
		Description:        "测试描述",
		AcceptanceCriteria: "测试验收标准",
	})

	if err != ErrProjectNotFound {
		t.Errorf("期望 ErrProjectNotFound, 实际为 %v", err)
	}
}

func TestRequirementService_GetRequirement(t *testing.T) {
	svc, _, projRepo := setupTestRequirementSvc()
	ctx := context.Background()

	// 先创建一个项目
	createTestProject(projRepo)

	// 创建一个需求
	created, _ := svc.CreateRequirement(ctx, CreateRequirementCommand{
		ProjectID:   domain.NewProjectID("proj-001"),
		Title:       "GetTestRequirement",
		Description: "测试获取需求",
	})

	// 获取需求
	requirement, err := svc.GetRequirement(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if requirement.Title() != "GetTestRequirement" {
		t.Errorf("期望 title 为 'GetTestRequirement', 实际为 '%s'", requirement.Title())
	}

	if requirement.Description() != "测试获取需求" {
		t.Errorf("期望 description 为 '测试获取需求', 实际为 '%s'", requirement.Description())
	}
}

func TestRequirementService_GetRequirement_NotFound(t *testing.T) {
	svc, _, _ := setupTestRequirementSvc()
	ctx := context.Background()

	_, err := svc.GetRequirement(ctx, domain.NewRequirementID("non-existent"))
	if err != ErrRequirementNotFound {
		t.Errorf("期望 ErrRequirementNotFound, 实际为 %v", err)
	}
}

func TestRequirementService_ListRequirements(t *testing.T) {
	svc, _, projRepo := setupTestRequirementSvc()
	ctx := context.Background()

	// 创建两个项目
	project1 := createTestProject(projRepo)
	project2, _ := domain.NewProject(
		domain.NewProjectID("proj-002"),
		"Test Project 2",
		"https://github.com/test/repo2.git",
		"main",
		[]string{},
	)
	projRepo.projects["proj-002"] = project2

	// 为项目1创建需求
	_, _ = svc.CreateRequirement(ctx, CreateRequirementCommand{
		ProjectID:   project1.ID(),
		Title:       "Requirement 1",
		Description: "Desc 1",
	})
	_, _ = svc.CreateRequirement(ctx, CreateRequirementCommand{
		ProjectID:   project1.ID(),
		Title:       "Requirement 2",
		Description: "Desc 2",
	})

	// 为项目2创建需求
	_, _ = svc.CreateRequirement(ctx, CreateRequirementCommand{
		ProjectID:   project2.ID(),
		Title:       "Requirement 3",
		Description: "Desc 3",
	})

	// 测试按项目ID过滤
	proj1ID := domain.NewProjectID("proj-001")
	requirements, err := svc.ListRequirements(ctx, &proj1ID)
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(requirements) != 2 {
		t.Errorf("期望 2 个 requirements, 实际为 %d", len(requirements))
	}

	for _, req := range requirements {
		if req.ProjectID().String() != "proj-001" {
			t.Errorf("期望 projectID 为 'proj-001', 实际为 '%s'", req.ProjectID().String())
		}
	}
}

func TestRequirementService_ListRequirements_All(t *testing.T) {
	svc, _, projRepo := setupTestRequirementSvc()
	ctx := context.Background()

	// 创建两个项目
	project1 := createTestProject(projRepo)
	project2, _ := domain.NewProject(
		domain.NewProjectID("proj-002"),
		"Test Project 2",
		"https://github.com/test/repo2.git",
		"main",
		[]string{},
	)
	projRepo.projects["proj-002"] = project2

	// 创建需求
	_, _ = svc.CreateRequirement(ctx, CreateRequirementCommand{
		ProjectID: project1.ID(),
		Title:     "Requirement 1",
	})
	_, _ = svc.CreateRequirement(ctx, CreateRequirementCommand{
		ProjectID: project2.ID(),
		Title:     "Requirement 2",
	})

	// 测试查询全部
	requirements, err := svc.ListRequirements(ctx, nil)
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(requirements) != 2 {
		t.Errorf("期望 2 个 requirements, 实际为 %d", len(requirements))
	}
}

func TestRequirementService_UpdateRequirement(t *testing.T) {
	svc, _, projRepo := setupTestRequirementSvc()
	ctx := context.Background()

	// 先创建一个项目
	createTestProject(projRepo)

	// 创建一个需求
	created, _ := svc.CreateRequirement(ctx, CreateRequirementCommand{
		ProjectID:          domain.NewProjectID("proj-001"),
		Title:              "Original Title",
		Description:        "Original Description",
		AcceptanceCriteria: "Original Criteria",
	})

	// 更新需求
	newTitle := "Updated Title"
	newDescription := "Updated Description"
	newCriteria := "Updated Criteria"
	updated, err := svc.UpdateRequirement(ctx, UpdateRequirementCommand{
		ID:                 created.ID(),
		Title:              &newTitle,
		Description:        &newDescription,
		AcceptanceCriteria: &newCriteria,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Title() != "Updated Title" {
		t.Errorf("期望 title 为 'Updated Title', 实际为 '%s'", updated.Title())
	}

	if updated.Description() != "Updated Description" {
		t.Errorf("期望 description 为 'Updated Description', 实际为 '%s'", updated.Description())
	}

	if updated.AcceptanceCriteria() != "Updated Criteria" {
		t.Errorf("期望 acceptanceCriteria 为 'Updated Criteria', 实际为 '%s'", updated.AcceptanceCriteria())
	}
}

func TestRequirementService_UpdateRequirement_NotFound(t *testing.T) {
	svc, _, _ := setupTestRequirementSvc()
	ctx := context.Background()

	newTitle := "New Title"
	_, err := svc.UpdateRequirement(ctx, UpdateRequirementCommand{
		ID:    domain.NewRequirementID("non-existent"),
		Title: &newTitle,
	})

	if err != ErrRequirementNotFound {
		t.Errorf("期望 ErrRequirementNotFound, 实际为 %v", err)
	}
}

func TestRequirementService_ReportRequirementPROpened(t *testing.T) {
	svc, reqRepo, projRepo := setupTestRequirementSvc()
	ctx := context.Background()

	// 先创建一个项目
	createTestProject(projRepo)

	// 创建一个需求并设置为 coding 状态
	requirement, _ := domain.NewRequirement(
		domain.NewRequirementID("req-pr-test"),
		domain.NewProjectID("proj-001"),
		"PR Test Requirement",
		"Description",
		"Criteria",
		"",
	)
	requirement.StartDispatch("agent-001")
	requirement.MarkCoding("/tmp/workspace", "replica-001")
	reqRepo.requirements["req-pr-test"] = requirement

	// 报告 PR 已打开
	updated, err := svc.ReportRequirementPROpened(ctx, ReportRequirementPRCommand{
		ID: requirement.ID(),
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Status() != domain.RequirementStatusPROpened {
		t.Errorf("期望 status 为 'pr_opened', 实际为 '%s'", updated.Status())
	}
}

func TestRequirementService_ReportRequirementPROpened_NotFound(t *testing.T) {
	svc, _, _ := setupTestRequirementSvc()
	ctx := context.Background()

	_, err := svc.ReportRequirementPROpened(ctx, ReportRequirementPRCommand{
		ID: domain.NewRequirementID("non-existent"),
	})

	if err != ErrRequirementNotFound {
		t.Errorf("期望 ErrRequirementNotFound, 实际为 %v", err)
	}
}

func TestRequirementService_RedispatchRequirement(t *testing.T) {
	svc, reqRepo, projRepo := setupTestRequirementSvc()
	ctx := context.Background()

	// 先创建一个项目
	createTestProject(projRepo)

	// 创建一个需求并设置为 coding 状态
	requirement, _ := domain.NewRequirement(
		domain.NewRequirementID("req-redispatch-test"),
		domain.NewProjectID("proj-001"),
		"Redispatch Test Requirement",
		"Description",
		"Criteria",
		"",
	)
	requirement.StartDispatch("agent-001")
	requirement.MarkCoding("/tmp/workspace", "replica-001")
	reqRepo.requirements["req-redispatch-test"] = requirement

	// 重新派发
	updated, err := svc.RedispatchRequirement(ctx, RedispatchRequirementCommand{
		ID: requirement.ID(),
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Status() != domain.RequirementStatusTodo {
		t.Errorf("期望 status 为 'todo', 实际为 '%s'", updated.Status())
	}

	if updated.AssigneeAgentCode() != "" {
		t.Errorf("期望 assigneeAgentCode 为空, 实际为 '%s'", updated.AssigneeAgentCode())
	}

	if updated.WorkspacePath() != "" {
		t.Errorf("期望 workspacePath 为空, 实际为 '%s'", updated.WorkspacePath())
	}
}

func TestRequirementService_RedispatchRequirement_NotFound(t *testing.T) {
	svc, _, _ := setupTestRequirementSvc()
	ctx := context.Background()

	_, err := svc.RedispatchRequirement(ctx, RedispatchRequirementCommand{
		ID: domain.NewRequirementID("non-existent"),
	})

	if err != ErrRequirementNotFound {
		t.Errorf("期望 ErrRequirementNotFound, 实际为 %v", err)
	}
}

func TestRequirementService_CopyAndDispatchRequirement_NotFound(t *testing.T) {
	svc, _, _ := setupTestRequirementSvc()
	ctx := context.Background()

	// 创建一个 mock 的 dispatchService（这里我们只是测试找不到需求的情况）
	dispatchService := &RequirementDispatchService{}

	_, err := svc.CopyAndDispatchRequirement(ctx, CopyAndDispatchRequirementCommand{
		ID: domain.NewRequirementID("non-existent"),
	}, dispatchService)

	if err != ErrRequirementNotFound {
		t.Errorf("期望 ErrRequirementNotFound, 实际为 %v", err)
	}
}

func TestRequirementService_CopyAndDispatchRequirement_ProjectNotFound(t *testing.T) {
	svc, reqRepo, _ := setupTestRequirementSvc()
	ctx := context.Background()

	// 创建一个需求但不创建对应的项目
	requirement, _ := domain.NewRequirement(
		domain.NewRequirementID("req-copy-test"),
		domain.NewProjectID("proj-non-existent"),
		"Copy Test Requirement",
		"Description",
		"Criteria",
		"",
	)
	reqRepo.requirements["req-copy-test"] = requirement

	// 创建一个 mock 的 dispatchService
	dispatchService := &RequirementDispatchService{}

	_, err := svc.CopyAndDispatchRequirement(ctx, CopyAndDispatchRequirementCommand{
		ID: requirement.ID(),
	}, dispatchService)

	if err != ErrProjectNotFound {
		t.Errorf("期望 ErrProjectNotFound, 实际为 %v", err)
	}
}
