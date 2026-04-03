package application

import (
	"context"
	"strconv"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

type mockProjectIDGen struct {
	count int
}

func (m *mockProjectIDGen) Generate() string {
	m.count++
	return "project-id-" + strconv.Itoa(m.count)
}

func setupTestProjectSvc() *ProjectApplicationService {
	repo := newSharedMockProjectRepo()
	idGen := &mockProjectIDGen{}
	return NewProjectApplicationService(repo, idGen)
}

func TestProjectService_CreateProject(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	project, err := svc.CreateProject(ctx, CreateProjectCommand{
		Name:          "TestProject",
		GitRepoURL:    "https://github.com/weibaohui/tasks.git",
		DefaultBranch: "main",
		InitSteps:     []string{"make setup"},
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if project.Name() != "TestProject" {
		t.Errorf("期望 name 为 'TestProject', 实际为 '%s'", project.Name())
	}

	if project.GitRepoURL() != "https://github.com/weibaohui/tasks.git" {
		t.Errorf("期望 git_repo_url 为 'https://github.com/weibaohui/tasks.git', 实际为 '%s'", project.GitRepoURL())
	}

	if project.DefaultBranch() != "main" {
		t.Errorf("期望 default_branch 为 'main', 实际为 '%s'", project.DefaultBranch())
	}

	if len(project.InitSteps()) != 1 || project.InitSteps()[0] != "make setup" {
		t.Errorf("期望 init_steps 为 ['make setup'], 实际为 %v", project.InitSteps())
	}
}

func TestProjectService_CreateProject_DefaultBranch(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	project, err := svc.CreateProject(ctx, CreateProjectCommand{
		Name:       "TestProject",
		GitRepoURL: "https://github.com/weibaohui/tasks.git",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if project.DefaultBranch() != "main" {
		t.Errorf("期望 default_branch 为 'main', 实际为 '%s'", project.DefaultBranch())
	}
}

func TestProjectService_CreateProject_ValidationError(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	_, err := svc.CreateProject(ctx, CreateProjectCommand{
		Name:       "",
		GitRepoURL: "https://github.com/weibaohui/tasks.git",
	})

	if err != domain.ErrProjectNameRequired {
		t.Errorf("期望 ErrProjectNameRequired, 实际为 %v", err)
	}

	_, err = svc.CreateProject(ctx, CreateProjectCommand{
		Name:       "TestProject",
		GitRepoURL: "",
	})

	if err != domain.ErrProjectRepoURLRequired {
		t.Errorf("期望 ErrProjectRepoURLRequired, 实际为 %v", err)
	}
}

func TestProjectService_GetProject(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	created, _ := svc.CreateProject(ctx, CreateProjectCommand{
		Name:          "GetTestProject",
		GitRepoURL:    "https://github.com/weibaohui/tasks.git",
		DefaultBranch: "main",
	})

	project, err := svc.GetProject(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if project.Name() != "GetTestProject" {
		t.Errorf("期望 name 为 'GetTestProject', 实际为 '%s'", project.Name())
	}
}

func TestProjectService_GetProject_NotFound(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	_, err := svc.GetProject(ctx, domain.NewProjectID("non-existent"))
	if err != ErrProjectNotFound {
		t.Errorf("期望 ErrProjectNotFound, 实际为 %v", err)
	}
}

func TestProjectService_ListProjects(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	svc.CreateProject(ctx, CreateProjectCommand{Name: "Project1", GitRepoURL: "https://github.com/a/b.git"})
	svc.CreateProject(ctx, CreateProjectCommand{Name: "Project2", GitRepoURL: "https://github.com/c/d.git"})
	svc.CreateProject(ctx, CreateProjectCommand{Name: "Project3", GitRepoURL: "https://github.com/e/f.git"})

	projects, err := svc.ListProjects(ctx)
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("期望 3 个 projects, 实际为 %d", len(projects))
	}
}

func TestProjectService_UpdateProject(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	created, _ := svc.CreateProject(ctx, CreateProjectCommand{
		Name:          "OriginalName",
		GitRepoURL:    "https://github.com/original/repo.git",
		DefaultBranch: "master",
		InitSteps:     []string{"step1"},
	})

	updated, err := svc.UpdateProject(ctx, UpdateProjectCommand{
		ID:            created.ID(),
		Name:          "UpdatedName",
		GitRepoURL:    "https://github.com/updated/repo.git",
		DefaultBranch: "main",
		InitSteps:     []string{"step2", "step3"},
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Name() != "UpdatedName" {
		t.Errorf("期望 name 为 'UpdatedName', 实际为 '%s'", updated.Name())
	}

	if updated.GitRepoURL() != "https://github.com/updated/repo.git" {
		t.Errorf("期望 git_repo_url 为 'https://github.com/updated/repo.git', 实际为 '%s'", updated.GitRepoURL())
	}

	if updated.DefaultBranch() != "main" {
		t.Errorf("期望 default_branch 为 'main', 实际为 '%s'", updated.DefaultBranch())
	}

	if len(updated.InitSteps()) != 2 || updated.InitSteps()[0] != "step2" {
		t.Errorf("期望 init_steps 为 ['step2', 'step3'], 实际为 %v", updated.InitSteps())
	}
}

func TestProjectService_UpdateProject_NotFound(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	_, err := svc.UpdateProject(ctx, UpdateProjectCommand{
		ID:   domain.NewProjectID("non-existent"),
		Name: "UpdatedName",
	})
	if err != ErrProjectNotFound {
		t.Errorf("期望 ErrProjectNotFound, 实际为 %v", err)
	}
}

func TestProjectService_UpdateProject_HeartbeatConfig(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	created, _ := svc.CreateProject(ctx, CreateProjectCommand{
		Name:       "HeartbeatTestProject",
		GitRepoURL: "https://github.com/a/b.git",
	})

	enabled := true
	interval := 30
	mdContent := "# Heartbeat"
	agentCode := "agt_001"

	updated, err := svc.UpdateProject(ctx, UpdateProjectCommand{
		ID:                       created.ID(),
		Name:                     "HeartbeatTestProject",
		GitRepoURL:               "https://github.com/a/b.git",
		HeartbeatEnabled:         &enabled,
		HeartbeatIntervalMinutes: &interval,
		HeartbeatMDContent:       &mdContent,
		AgentCode:                &agentCode,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if !updated.HeartbeatEnabled() {
		t.Error("期望 heartbeat_enabled 为 true")
	}

	if updated.HeartbeatIntervalMinutes() != 30 {
		t.Errorf("期望 heartbeat_interval_minutes 为 30, 实际为 %d", updated.HeartbeatIntervalMinutes())
	}

	if updated.HeartbeatMDContent() != "# Heartbeat" {
		t.Errorf("期望 heartbeat_md_content 为 '# Heartbeat', 实际为 '%s'", updated.HeartbeatMDContent())
	}

	if updated.AgentCode() != "agt_001" {
		t.Errorf("期望 agent_code 为 'agt_001', 实际为 '%s'", updated.AgentCode())
	}
}

func TestProjectService_UpdateProject_DispatchConfig(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	created, _ := svc.CreateProject(ctx, CreateProjectCommand{
		Name:       "DispatchTestProject",
		GitRepoURL: "https://github.com/a/b.git",
	})

	channelCode := "channel_001"
	sessionKey := "session_001"

	updated, err := svc.UpdateProject(ctx, UpdateProjectCommand{
		ID:                  created.ID(),
		Name:                "DispatchTestProject",
		GitRepoURL:          "https://github.com/a/b.git",
		DispatchChannelCode: &channelCode,
		DispatchSessionKey:  &sessionKey,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.DispatchChannelCode() != "channel_001" {
		t.Errorf("期望 dispatch_channel_code 为 'channel_001', 实际为 '%s'", updated.DispatchChannelCode())
	}

	if updated.DispatchSessionKey() != "session_001" {
		t.Errorf("期望 dispatch_session_key 为 'session_001', 实际为 '%s'", updated.DispatchSessionKey())
	}
}

func TestProjectService_UpdateProject_DispatchConfig_EmptyStringNotOverwrite(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	// 先创建项目并设置派发配置
	created, _ := svc.CreateProject(ctx, CreateProjectCommand{
		Name:       "DispatchOverwriteProject",
		GitRepoURL: "https://github.com/a/b.git",
	})

	channelCode := "channel_001"
	sessionKey := "session_001"
	svc.UpdateProject(ctx, UpdateProjectCommand{
		ID:                  created.ID(),
		Name:                "DispatchOverwriteProject",
		GitRepoURL:          "https://github.com/a/b.git",
		DispatchChannelCode: &channelCode,
		DispatchSessionKey:  &sessionKey,
	})

	// 再用空字符串更新，不应覆盖现有配置
	emptyChannel := ""
	emptySession := ""
	updated, err := svc.UpdateProject(ctx, UpdateProjectCommand{
		ID:                  created.ID(),
		Name:                "DispatchOverwriteProject",
		GitRepoURL:          "https://github.com/a/b.git",
		DispatchChannelCode: &emptyChannel,
		DispatchSessionKey:  &emptySession,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.DispatchChannelCode() != "channel_001" {
		t.Errorf("期望 dispatch_channel_code 保持 'channel_001', 实际为 '%s'", updated.DispatchChannelCode())
	}

	if updated.DispatchSessionKey() != "session_001" {
		t.Errorf("期望 dispatch_session_key 保持 'session_001', 实际为 '%s'", updated.DispatchSessionKey())
	}
}

func TestProjectService_DeleteProject(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	created, _ := svc.CreateProject(ctx, CreateProjectCommand{
		Name:       "DeleteTestProject",
		GitRepoURL: "https://github.com/a/b.git",
	})

	err := svc.DeleteProject(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	_, err = svc.GetProject(ctx, created.ID())
	if err != ErrProjectNotFound {
		t.Errorf("期望 ErrProjectNotFound, 实际为 %v", err)
	}
}

func TestProjectService_DeleteProject_NotFound(t *testing.T) {
	svc := setupTestProjectSvc()
	ctx := context.Background()

	err := svc.DeleteProject(ctx, domain.NewProjectID("non-existent"))
	if err != ErrProjectNotFound {
		t.Errorf("期望 ErrProjectNotFound, 实际为 %v", err)
	}
}
