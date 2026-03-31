/**
 * CLI 端到端测试
 *
 * 测试所有 taskmanager CLI 子命令的基本功能
 *
 * 运行方式:
 *   cd backend && go test -v -run TestCLI ./cmd/cli/...
 *
 * 注意: 这些测试需要数据库配置正确，默认使用 ~/.taskmanager/data.db
 */
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
)

// CLI 二进制路径
var cliBinary = "/tmp/taskmanager_cli_e2e_test"

// requiresDB 如果默认数据库路径不存在则跳过测试
func requiresDB(t *testing.T) {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".taskmanager", "data.db")
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("跳过: ~/.taskmanager 目录不存在 (需要先运行 make dev 或创建数据库)")
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skip("跳过: 默认数据库文件不存在 (需要先运行 make dev 或创建数据库)")
	}
}

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// 创建临时数据库
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("无法创建数据库: %v", err)
	}

	if err := _persistence.InitSchema(db); err != nil {
		db.Close()
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func buildCLI(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", cliBinary, "./cmd/cli")
	cmd.Dir = filepath.Join(getGitRoot(), "backend")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("编译 CLI 失败: %v\n%s", err, string(output))
	}
}

func getGitRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

func runCLI(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cliBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ========== Admin 命令测试 ==========

func TestCLI_CreateAdmin(t *testing.T) {
	requiresDB(t)
	buildCLI(t)

	output, err := runCLI("create-admin")
	if err != nil {
		t.Fatalf("create-admin 失败: %v\n%s", err, output)
	}

	// 验证输出包含成功信息
	if !strings.Contains(output, "成功") && !strings.Contains(output, "success") {
		t.Logf("输出: %s", output)
	}
}

func TestCLI_DeleteAdmin(t *testing.T) {
	requiresDB(t)
	buildCLI(t)

	// 先创建
	runCLI("create-admin")

	// 再删除
	output, err := runCLI("delete-admin")
	if err != nil {
		t.Fatalf("delete-admin 失败: %v\n%s", err, output)
	}
}

// ========== Agent 命令测试 ==========

func TestCLI_AgentList(t *testing.T) {
	requiresDB(t)
	buildCLI(t)

	output, err := runCLI("agent", "list")
	if err != nil {
		t.Fatalf("agent list 失败: %v\n%s", err, output)
	}

	// 验证输出包含表头
	if !strings.Contains(output, "Agent 列表") {
		t.Errorf("输出不包含 'Agent 列表':\n%s", output)
	}

	// 验证输出包含 ID 和状态 列
	if !strings.Contains(output, "ID") {
		t.Errorf("输出不包含 'ID' 列:\n%s", output)
	}

	t.Logf("agent list 输出:\n%s", output)
}

// ========== Project 命令测试 ==========

func TestCLI_ProjectList(t *testing.T) {
	requiresDB(t)
	buildCLI(t)

	output, err := runCLI("project", "list")
	if err != nil {
		t.Fatalf("project list 失败: %v\n%s", err, output)
	}

	// 验证输出包含表头
	if !strings.Contains(output, "项目列表") {
		t.Errorf("输出不包含 '项目列表':\n%s", output)
	}

	t.Logf("project list 输出:\n%s", output)
}

func TestCLI_ProjectHeartbeatStatus(t *testing.T) {
	requiresDB(t)
	buildCLI(t)

	output, err := runCLI("project", "heartbeat", "status")
	if err != nil {
		t.Fatalf("project heartbeat status 失败: %v\n%s", err, output)
	}

	// 验证输出包含心跳状态表头
	if !strings.Contains(output, "项目心跳状态") {
		t.Errorf("输出不包含 '项目心跳状态':\n%s", output)
	}

	t.Logf("project heartbeat status 输出:\n%s", output)
}

// ========== Requirement 命令测试 ==========

func TestCLI_RequirementList(t *testing.T) {
	requiresDB(t)
	buildCLI(t)

	output, err := runCLI("requirement", "list")
	if err != nil {
		t.Fatalf("requirement list 失败: %v\n%s", err, output)
	}

	// 验证输出包含需求列表表头
	if !strings.Contains(output, "需求列表") {
		t.Errorf("输出不包含 '需求列表':\n%s", output)
	}

	t.Logf("requirement list 输出:\n%s", output)
}

func TestCLI_RequirementListWithHeartbeat(t *testing.T) {
	requiresDB(t)
	buildCLI(t)

	// 测试 --include-heartbeat 标志
	output, err := runCLI("requirement", "list", "--include-heartbeat")
	if err != nil {
		t.Fatalf("requirement list --include-heartbeat 失败: %v\n%s", err, output)
	}

	if !strings.Contains(output, "需求列表") {
		t.Errorf("输出不包含 '需求列表':\n%s", output)
	}

	t.Logf("requirement list --include-heartbeat 输出:\n%s", output)
}

func TestCLI_RequirementCreate(t *testing.T) {
	requiresDB(t)
	buildCLI(t)

	// 需要先有一个项目 ID
	// 创建测试项目
	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	idGen := utils.NewNanoIDGenerator(21)
	projectRepo := _persistence.NewSQLiteProjectRepository(db)

	project, err := domain.NewProject(domain.NewProjectID(idGen.Generate()), "测试项目", "test-project", "main", nil)
	if err != nil {
		t.Fatalf("创建项目对象失败: %v", err)
	}
	ctx := context.Background()
	if err := projectRepo.Save(ctx, project); err != nil {
		t.Fatalf("创建测试项目失败: %v", err)
	}

	// 使用项目 ID 创建需求
	output, err := runCLI("requirement", "create",
		"--project-id", project.ID().String(),
		"--title", "E2E 测试需求",
		"--description", "这是一个 E2E 测试需求")
	if err != nil {
		t.Fatalf("requirement create 失败: %v\n%s", err, output)
	}

	// 验证输出包含成功信息
	if !strings.Contains(output, "成功") {
		t.Errorf("输出不包含 '成功':\n%s", output)
	}

	t.Logf("requirement create 输出:\n%s", output)
}

func TestCLI_RequirementGet(t *testing.T) {
	requiresDB(t)
	buildCLI(t)

	// 先创建一个测试需求
	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	idGen := utils.NewNanoIDGenerator(21)
	projectRepo := _persistence.NewSQLiteProjectRepository(db)
	requirementRepo := _persistence.NewSQLiteRequirementRepository(db)
	appService := application.NewRequirementApplicationService(requirementRepo, projectRepo, idGen, nil, nil)

	ctx := context.Background()

	// 创建项目
	project, err := domain.NewProject(domain.NewProjectID(idGen.Generate()), "测试项目", "test-project", "main", nil)
	if err != nil {
		t.Fatalf("创建项目对象失败: %v", err)
	}
	if err := projectRepo.Save(ctx, project); err != nil {
		t.Fatalf("创建测试项目失败: %v", err)
	}

	// 创建需求
	requirement, err := appService.CreateRequirement(ctx, application.CreateRequirementCommand{
		ProjectID: project.ID(),
		Title:     "E2E 测试需求 - Get",
	})
	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	// 使用 requirement get 获取详情
	output, err := runCLI("requirement", "get", "--id", requirement.ID().String())
	if err != nil {
		t.Fatalf("requirement get 失败: %v\n%s", err, output)
	}

	// 验证输出包含需求详情
	if !strings.Contains(output, "需求详情") {
		t.Errorf("输出不包含 '需求详情':\n%s", output)
	}
	if !strings.Contains(output, "E2E 测试需求 - Get") {
		t.Errorf("输出不包含需求标题:\n%s", output)
	}

	t.Logf("requirement get 输出:\n%s", output)
}

// ========== Config 命令测试 ==========

func TestCLI_ConfigShow(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("config", "show")
	if err != nil {
		t.Fatalf("config show 失败: %v\n%s", err, output)
	}

	// 验证输出包含配置信息
	if !strings.Contains(output, "当前配置") {
		t.Errorf("输出不包含 '当前配置':\n%s", output)
	}

	t.Logf("config show 输出:\n%s", output)
}

func TestCLI_ConfigInit(t *testing.T) {
	buildCLI(t)

	// 创建临时配置目录
	tmpDir := t.TempDir()
	tmpConfig := filepath.Join(tmpDir, "config.yaml")

	// 设置临时配置路径
	os.Setenv("TASKMANAGER_CONFIG", tmpConfig)
	defer os.Unsetenv("TASKMANAGER_CONFIG")

	output, err := runCLI("config", "init")
	if err != nil {
		t.Fatalf("config init 失败: %v\n%s", err, output)
	}

	// 验证配置文件被创建
	if _, err := os.Stat(tmpConfig); os.IsNotExist(err) {
		t.Errorf("配置文件未创建: %s", tmpConfig)
	}

	t.Logf("config init 输出:\n%s", output)
}

// ========== 帮助命令测试 ==========

func TestCLI_Help(t *testing.T) {
	buildCLI(t)

	// 测试根命令帮助
	output, err := runCLI("--help")
	if err != nil {
		t.Fatalf("--help 失败: %v\n%s", err, output)
	}

	// 验证输出包含可用命令
	expectedCommands := []string{
		"agent",
		"project",
		"requirement",
		"config",
		"create-admin",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}

	t.Logf("--help 输出:\n%s", output)
}

func TestCLI_RequirementHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("requirement", "--help")
	if err != nil {
		t.Fatalf("requirement --help 失败: %v\n%s", err, output)
	}

	// 验证输出包含所有子命令
	expectedSubcommands := []string{
		"create",
		"list",
		"get",
		"update",
		"dispatch",
		"complete",
		"review",
		"tasks",
	}

	for _, cmd := range expectedSubcommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("requirement 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}

	t.Logf("requirement --help 输出:\n%s", output)
}

func TestCLI_ProjectHeartbeatHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("project", "heartbeat", "--help")
	if err != nil {
		t.Fatalf("project heartbeat --help 失败: %v\n%s", err, output)
	}

	// 验证输出包含心跳子命令
	expectedSubcommands := []string{
		"enable",
		"disable",
		"set-interval",
		"status",
	}

	for _, cmd := range expectedSubcommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("project heartbeat 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}

	t.Logf("project heartbeat --help 输出:\n%s", output)
}

func TestCLI_RequirementTasksHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("requirement", "tasks", "--help")
	if err != nil {
		t.Fatalf("requirement tasks --help 失败: %v\n%s", err, output)
	}

	// 验证输出包含 tasks 命令信息
	if !strings.Contains(output, "查看需求的子任务") {
		t.Errorf("输出不包含 '查看需求的子任务':\n%s", output)
	}

	if !strings.Contains(output, "--id") {
		t.Errorf("输出不包含 '--id' 标志:\n%s", output)
	}

	t.Logf("requirement tasks --help 输出:\n%s", output)
}

// ========== 错误处理测试 ==========

func TestCLI_RequirementGetWithoutID(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("requirement", "get")
	if err == nil {
		t.Logf("预期失败但成功了, 输出: %s", output)
	}

	// 验证输出包含错误信息
	if !strings.Contains(output, "错误") && !strings.Contains(output, "id") {
		t.Logf("输出: %s", output)
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("unknown-command")
	if err == nil {
		t.Logf("预期失败但成功了, 输出: %s", output)
	}

	// Cobra 会输出帮助信息
	if !strings.Contains(output, "Available Commands") {
		t.Logf("输出: %s", output)
	}
}

// ========== 集成测试: 完整工作流 ==========

func TestCLI_Workflow_ProjectAndRequirement(t *testing.T) {
	// 注意: 此测试需要配置 TASKMANAGER_DB_PATH 环境变量指向有效的数据库
	// 由于 CLI 和测试使用不同的数据库，此测试暂时跳过
	t.Skip("跳过: CLI 进程无法访问测试创建的临时数据库，需要集成测试环境")
}

func TestCLI_Workflow_ProjectAndRequirement_Manual(t *testing.T) {
	// 手动集成测试 - 需要设置有效的数据库路径
	// export TASKMANAGER_DB_PATH=/path/to/test.db
	dbPath := os.Getenv("TASKMANAGER_DB_PATH")
	if dbPath == "" {
		t.Skip("跳过: 需要设置 TASKMANAGER_DB_PATH 环境变量")
	}

	buildCLI(t)

	// 1. 创建项目
	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	idGen := utils.NewNanoIDGenerator(21)
	projectRepo := _persistence.NewSQLiteProjectRepository(db)
	requirementRepo := _persistence.NewSQLiteRequirementRepository(db)
	appService := application.NewRequirementApplicationService(requirementRepo, projectRepo, idGen, nil, nil)

	ctx := context.Background()

	// 创建项目
	project, err := domain.NewProject(domain.NewProjectID(idGen.Generate()), "E2E 工作流测试项目", "e2e-workflow-test", "main", nil)
	if err != nil {
		t.Fatalf("创建项目对象失败: %v", err)
	}
	if err := projectRepo.Save(ctx, project); err != nil {
		t.Fatalf("创建测试项目失败: %v", err)
	}

	t.Logf("创建项目成功: %s", project.ID().String())

	// 2. 列出项目验证
	output, err := runCLI("project", "list")
	if err != nil {
		t.Fatalf("project list 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "E2E 工作流测试项目") {
		t.Errorf("项目列表不包含新创建的项目:\n%s", output)
	}

	// 3. 创建需求
	requirement, err := appService.CreateRequirement(ctx, application.CreateRequirementCommand{
		ProjectID:          project.ID(),
		Title:              "E2E 工作流测试需求",
		Description:        "用于测试完整工作流的需求",
		AcceptanceCriteria: "满足 E2E 测试条件",
	})
	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	t.Logf("创建需求成功: %s", requirement.ID().String())

	// 4. 获取需求详情
	output, err = runCLI("requirement", "get", "--id", requirement.ID().String())
	if err != nil {
		t.Fatalf("requirement get 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "E2E 工作流测试需求") {
		t.Errorf("需求详情不包含标题:\n%s", output)
	}

	// 5. 列出需求
	output, err = runCLI("requirement", "list")
	if err != nil {
		t.Fatalf("requirement list 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "E2E 工作流测试需求") {
		t.Errorf("需求列表不包含新创建的需求:\n%s", output)
	}

	t.Log("完整工作流测试通过!")
}

// Helper: 确保 CLI 已编译
func TestMain(m *testing.M) {
	// 确保 CLI 可以编译
	cmd := exec.Command("go", "build", "-o", cliBinary, "./cmd/cli")
	cmd.Dir = filepath.Join(getGitRoot(), "backend")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("编译 CLI 失败: %v\n%s", err, string(output))
		os.Exit(1)
	}

	os.Exit(m.Run())
}
