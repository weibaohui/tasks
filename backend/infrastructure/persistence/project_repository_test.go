/**
 * SQLite Project Repository 集成测试
 */
package persistence

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/domain"
)

func setupProjectTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	err = InitSchema(db)
	if err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func createTestProject(id, name, gitRepoURL string) *domain.Project {
	return createTestProjectWithBranch(id, name, gitRepoURL, "main")
}

func createTestProjectWithBranch(id, name, gitRepoURL, defaultBranch string) *domain.Project {
	project, _ := domain.NewProject(
		domain.NewProjectID(id),
		name,
		gitRepoURL,
		defaultBranch,
		[]string{"make setup"},
	)
	return project
}

func TestSQLiteProjectRepository_SaveAndFindByID(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	repo := NewSQLiteProjectRepository(db)
	ctx := context.Background()

	project := createTestProject("proj-001", "测试项目", "https://github.com/test/repo.git")
	err := repo.Save(ctx, project)
	if err != nil {
		t.Fatalf("保存项目失败: %v", err)
	}

	found, err := repo.FindByID(ctx, project.ID())
	if err != nil {
		t.Fatalf("查找项目失败: %v", err)
	}

	if found.Name() != "测试项目" {
		t.Errorf("期望 name 为 '测试项目', 实际为 '%s'", found.Name())
	}
	if found.GitRepoURL() != "https://github.com/test/repo.git" {
		t.Errorf("期望 git_repo_url 为 'https://github.com/test/repo.git', 实际为 '%s'", found.GitRepoURL())
	}
	if found.DefaultBranch() != "main" {
		t.Errorf("期望 default_branch 为 'main', 实际为 '%s'", found.DefaultBranch())
	}
	if len(found.InitSteps()) != 1 || found.InitSteps()[0] != "make setup" {
		t.Errorf("期望 init_steps 为 ['make setup'], 实际为 %v", found.InitSteps())
	}
	if found.HeartbeatScenarioCode() != "" {
		t.Errorf("期望 heartbeat_scenario_code 为空, 实际为 '%s'", found.HeartbeatScenarioCode())
	}
}

func TestSQLiteProjectRepository_FindAll(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	repo := NewSQLiteProjectRepository(db)
	ctx := context.Background()

	repo.Save(ctx, createTestProject("proj-001", "项目1", "https://github.com/a/b.git"))
	repo.Save(ctx, createTestProject("proj-002", "项目2", "https://github.com/c/d.git"))

	projects, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("查找所有项目失败: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("期望 2 个项目, 实际为 %d", len(projects))
	}
}

func TestSQLiteProjectRepository_UpdateHeartbeatScenarioCode(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	repo := NewSQLiteProjectRepository(db)
	ctx := context.Background()

	project := createTestProject("proj-001", "测试项目", "https://github.com/test/repo.git")
	repo.Save(ctx, project)

	project.SetHeartbeatScenarioCode("github_dev_workflow")
	err := repo.Save(ctx, project)
	if err != nil {
		t.Fatalf("更新项目失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, project.ID())
	if found.HeartbeatScenarioCode() != "github_dev_workflow" {
		t.Errorf("期望 heartbeat_scenario_code 为 'github_dev_workflow', 实际为 '%s'", found.HeartbeatScenarioCode())
	}
}

func TestSQLiteProjectRepository_Delete(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	repo := NewSQLiteProjectRepository(db)
	ctx := context.Background()

	project := createTestProject("proj-001", "测试项目", "https://github.com/test/repo.git")
	repo.Save(ctx, project)

	err := repo.Delete(ctx, project.ID())
	if err != nil {
		t.Fatalf("删除项目失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, project.ID())
	if found != nil {
		t.Error("期望项目已被删除")
	}
}

func TestSQLiteProjectRepository_NotFound(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	repo := NewSQLiteProjectRepository(db)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, domain.NewProjectID("non-existent"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if found != nil {
		t.Error("期望返回 nil")
	}
}

func TestSQLiteProjectRepository_UpdateOtherFields(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	repo := NewSQLiteProjectRepository(db)
	ctx := context.Background()

	project := createTestProjectWithBranch("proj-001", "原名称", "https://github.com/original/repo.git", "master")
	repo.Save(ctx, project)

	project.Update("新名称", "https://github.com/new/repo.git", "main", []string{"make build", "make test"})
	channelCode := "channel_001"
	sessionKey := "session_001"
	agentCode := "agent_001"
	project.UpdateDispatchConfig(&channelCode, &sessionKey, &agentCode)
	project.SetMaxConcurrentAgents(4)
	project.SetHeartbeatScenarioCode("dev_workflow")
	err := repo.Save(ctx, project)
	if err != nil {
		t.Fatalf("更新项目失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, project.ID())
	if found.Name() != "新名称" {
		t.Errorf("期望 name 为 '新名称', 实际为 '%s'", found.Name())
	}
	if found.GitRepoURL() != "https://github.com/new/repo.git" {
		t.Errorf("期望 git_repo_url 为 'https://github.com/new/repo.git', 实际为 '%s'", found.GitRepoURL())
	}
	if found.DefaultBranch() != "main" {
		t.Errorf("期望 default_branch 为 'main', 实际为 '%s'", found.DefaultBranch())
	}
	if len(found.InitSteps()) != 2 || found.InitSteps()[0] != "make build" {
		t.Errorf("期望 init_steps 为 ['make build', 'make test'], 实际为 %v", found.InitSteps())
	}
	if found.DispatchChannelCode() != "channel_001" {
		t.Errorf("期望 dispatch_channel_code 为 'channel_001', 实际为 '%s'", found.DispatchChannelCode())
	}
	if found.DispatchSessionKey() != "session_001" {
		t.Errorf("期望 dispatch_session_key 为 'session_001', 实际为 '%s'", found.DispatchSessionKey())
	}
	if found.DefaultAgentCode() != "agent_001" {
		t.Errorf("期望 default_agent_code 为 'agent_001', 实际为 '%s'", found.DefaultAgentCode())
	}
	if found.MaxConcurrentAgents() != 4 {
		t.Errorf("期望 max_concurrent_agents 为 4, 实际为 %d", found.MaxConcurrentAgents())
	}
	if found.HeartbeatScenarioCode() != "dev_workflow" {
		t.Errorf("期望 heartbeat_scenario_code 为 'dev_workflow', 实际为 '%s'", found.HeartbeatScenarioCode())
	}
}
