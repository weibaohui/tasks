/**
 * SQLite HeartbeatScenario Repository 集成测试
 */
package persistence

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/domain"
)

func setupHeartbeatScenarioTestDB(t *testing.T) (*sql.DB, func()) {
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

func createTestHeartbeatScenario(id, code, name string) *domain.HeartbeatScenario {
	scenario, _ := domain.NewHeartbeatScenario(
		domain.NewHeartbeatScenarioID(id),
		code,
		name,
		"测试描述",
		[]domain.HeartbeatScenarioItem{
			{Name: "心跳1", IntervalMinutes: 30, MDContent: "md1", AgentCode: "agent1", RequirementType: "normal", SortOrder: 1},
			{Name: "心跳2", IntervalMinutes: 60, MDContent: "md2", AgentCode: "agent2", RequirementType: "heartbeat", SortOrder: 2},
		},
	)
	return scenario
}

func TestSQLiteHeartbeatScenarioRepository_SaveAndFindByID(t *testing.T) {
	db, cleanup := setupHeartbeatScenarioTestDB(t)
	defer cleanup()

	repo := NewSQLiteHeartbeatScenarioRepository(db)
	ctx := context.Background()

	scenario := createTestHeartbeatScenario("sc-001", "dev_workflow", "开发工作流")
	err := repo.Save(ctx, scenario)
	if err != nil {
		t.Fatalf("保存场景失败: %v", err)
	}

	found, err := repo.FindByID(ctx, scenario.ID())
	if err != nil {
		t.Fatalf("查找场景失败: %v", err)
	}

	if found.Code() != "dev_workflow" {
		t.Errorf("期望 code 为 'dev_workflow', 实际为 '%s'", found.Code())
	}
	if found.Name() != "开发工作流" {
		t.Errorf("期望 name 为 '开发工作流', 实际为 '%s'", found.Name())
	}
	if len(found.Items()) != 2 {
		t.Errorf("期望 2 个 items, 实际为 %d", len(found.Items()))
	}
	if found.Items()[0].Name != "心跳1" {
		t.Errorf("期望 item[0].Name 为 '心跳1', 实际为 '%s'", found.Items()[0].Name)
	}
	if found.Items()[1].SortOrder != 2 {
		t.Errorf("期望 item[1].SortOrder 为 2, 实际为 %d", found.Items()[1].SortOrder)
	}
}

func TestSQLiteHeartbeatScenarioRepository_FindByCode(t *testing.T) {
	db, cleanup := setupHeartbeatScenarioTestDB(t)
	defer cleanup()

	repo := NewSQLiteHeartbeatScenarioRepository(db)
	ctx := context.Background()

	scenario := createTestHeartbeatScenario("sc-001", "github_dev", "GitHub开发")
	repo.Save(ctx, scenario)

	found, err := repo.FindByCode(ctx, "github_dev")
	if err != nil {
		t.Fatalf("查找场景失败: %v", err)
	}
	if found == nil {
		t.Fatal("期望找到场景, 实际为 nil")
	}
	if found.ID().String() != "sc-001" {
		t.Errorf("期望 ID 为 sc-001, 实际为 %s", found.ID().String())
	}
}

func TestSQLiteHeartbeatScenarioRepository_FindAll(t *testing.T) {
	db, cleanup := setupHeartbeatScenarioTestDB(t)
	defer cleanup()

	repo := NewSQLiteHeartbeatScenarioRepository(db)
	ctx := context.Background()

	repo.Save(ctx, createTestHeartbeatScenario("sc-001", "code1", "场景1"))
	repo.Save(ctx, createTestHeartbeatScenario("sc-002", "code2", "场景2"))
	repo.Save(ctx, createTestHeartbeatScenario("sc-003", "code3", "场景3"))

	scenarios, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("查找所有场景失败: %v", err)
	}

	if len(scenarios) != 3 {
		t.Errorf("期望 3 个场景, 实际为 %d", len(scenarios))
	}
}

func TestSQLiteHeartbeatScenarioRepository_Update(t *testing.T) {
	db, cleanup := setupHeartbeatScenarioTestDB(t)
	defer cleanup()

	repo := NewSQLiteHeartbeatScenarioRepository(db)
	ctx := context.Background()

	scenario := createTestHeartbeatScenario("sc-001", "github_dev", "原名称")
	repo.Save(ctx, scenario)

	scenario.SetEnabled(false)
	scenario.SetIsBuiltIn(true)
	scenario.Update("新名称", "新描述", []domain.HeartbeatScenarioItem{
		{Name: "心跳3", IntervalMinutes: 45, MDContent: "md3", AgentCode: "agent3", RequirementType: "pr_review"},
	})
	err := repo.Save(ctx, scenario)
	if err != nil {
		t.Fatalf("更新场景失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, scenario.ID())
	if found.Name() != "新名称" {
		t.Errorf("期望 name 为 '新名称', 实际为 '%s'", found.Name())
	}
	if found.Description() != "新描述" {
		t.Errorf("期望 description 为 '新描述', 实际为 '%s'", found.Description())
	}
	if found.Enabled() {
		t.Error("期望 enabled 为 false")
	}
	if !found.IsBuiltIn() {
		t.Error("期望 is_built_in 为 true")
	}
	if len(found.Items()) != 1 || found.Items()[0].Name != "心跳3" {
		t.Errorf("期望 items 已更新, 实际为 %v", found.Items())
	}
}

func TestSQLiteHeartbeatScenarioRepository_Delete(t *testing.T) {
	db, cleanup := setupHeartbeatScenarioTestDB(t)
	defer cleanup()

	repo := NewSQLiteHeartbeatScenarioRepository(db)
	ctx := context.Background()

	scenario := createTestHeartbeatScenario("sc-001", "github_dev", "GitHub开发")
	repo.Save(ctx, scenario)

	err := repo.Delete(ctx, scenario.ID())
	if err != nil {
		t.Fatalf("删除场景失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, scenario.ID())
	if found != nil {
		t.Error("期望场景已被删除")
	}
}

func TestSQLiteHeartbeatScenarioRepository_NotFound(t *testing.T) {
	db, cleanup := setupHeartbeatScenarioTestDB(t)
	defer cleanup()

	repo := NewSQLiteHeartbeatScenarioRepository(db)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, domain.NewHeartbeatScenarioID("non-existent"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if found != nil {
		t.Error("期望返回 nil")
	}

	found, err = repo.FindByCode(ctx, "non-existent")
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if found != nil {
		t.Error("期望返回 nil")
	}
}

func TestSQLiteHeartbeatScenarioRepository_EmptyItems(t *testing.T) {
	db, cleanup := setupHeartbeatScenarioTestDB(t)
	defer cleanup()

	repo := NewSQLiteHeartbeatScenarioRepository(db)
	ctx := context.Background()

	scenario, _ := domain.NewHeartbeatScenario(
		domain.NewHeartbeatScenarioID("sc-empty"),
		"empty_scenario",
		"空场景",
		"",
		[]domain.HeartbeatScenarioItem{},
	)
	err := repo.Save(ctx, scenario)
	if err != nil {
		t.Fatalf("保存空场景失败: %v", err)
	}

	found, err := repo.FindByID(ctx, scenario.ID())
	if err != nil {
		t.Fatalf("查找空场景失败: %v", err)
	}
	if len(found.Items()) != 0 {
		t.Errorf("期望 0 个 items, 实际为 %d", len(found.Items()))
	}
}
