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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/config"
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

// requiresAPIToken 如果 API Token 未配置则跳过测试
func requiresAPIToken(t *testing.T) {
	// 使用项目配置加载器读取配置
	cfg, err := config.Load()
	if err != nil {
		t.Skip("跳过: 无法加载配置")
	}
	if cfg.API.Token == "" {
		t.Skip("跳过: API Token 未配置 (需要在 ~/.taskmanager/config.yaml 中设置 api.token)")
	}
}

func setupTestDB(t *testing.T) (*sql.DB, string, func()) {
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

	return db, dbPath, cleanup
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
	return runCLIWithEnv(nil, args...)
}

func runCLIWithEnv(env map[string]string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cliBinary, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// useTestDB 使用指定数据库路径启动临时 server，测试结束后恢复默认 server
// copyAuthDataToTestDB 将默认数据库中的认证数据复制到临时数据库，
// 确保使用临时数据库的 server 能够验证现有 API Token。
func copyAuthDataToTestDB(t *testing.T, testDBPath string) {
	// 使用 config.GetDatabasePath() 获取源数据库路径，尊重 TASKMANAGER_DB_PATH 环境变量
	sourceDBPath := config.GetDatabasePath()

	// 验证源数据库存在
	if _, err := os.Stat(sourceDBPath); os.IsNotExist(err) {
		t.Skipf("跳过: 源数据库不存在: %s", sourceDBPath)
	}

	// 打开源数据库和目标数据库
	sourceDB, err := sql.Open("sqlite3", sourceDBPath)
	if err != nil {
		t.Fatalf("无法打开源数据库: %v", err)
	}
	defer sourceDB.Close()

	targetDB, err := sql.Open("sqlite3", testDBPath)
	if err != nil {
		t.Fatalf("无法打开目标数据库: %v", err)
	}
	defer targetDB.Close()

	// 开始事务
	tx, err := targetDB.Begin()
	if err != nil {
		t.Fatalf("无法开始事务: %v", err)
	}

	// 复制 users 表
	rows, err := sourceDB.Query("SELECT id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at FROM users")
	if err != nil {
		tx.Rollback()
		t.Fatalf("无法查询源数据库 users 表: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, userCode, username, email, displayName, passwordHash string
		var isActive int
		var createdAt, updatedAt int64
		if err := rows.Scan(&id, &userCode, &username, &email, &displayName, &passwordHash, &isActive, &createdAt, &updatedAt); err != nil {
			tx.Rollback()
			t.Fatalf("无法读取 users 数据: %v", err)
		}
		_, err = tx.Exec("INSERT OR REPLACE INTO users (id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			id, userCode, username, email, displayName, passwordHash, isActive, createdAt, updatedAt)
		if err != nil {
			tx.Rollback()
			t.Fatalf("无法插入 users 数据: %v", err)
		}
	}
	rows.Close()

	// 复制 user_tokens 表
	rows, err = sourceDB.Query("SELECT id, user_id, name, description, token_hash, expires_at, last_used_at, is_active, created_at FROM user_tokens")
	if err != nil {
		tx.Rollback()
		t.Fatalf("无法查询源数据库 user_tokens 表: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, userID, name, description, tokenHash string
		var expiresAt, lastUsedAt sql.NullInt64
		var isActive int
		var createdAt int64
		if err := rows.Scan(&id, &userID, &name, &description, &tokenHash, &expiresAt, &lastUsedAt, &isActive, &createdAt); err != nil {
			tx.Rollback()
			t.Fatalf("无法读取 user_tokens 数据: %v", err)
		}
		_, err = tx.Exec("INSERT OR REPLACE INTO user_tokens (id, user_id, name, description, token_hash, expires_at, last_used_at, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			id, userID, name, description, tokenHash, expiresAt, lastUsedAt, isActive, createdAt)
		if err != nil {
			tx.Rollback()
			t.Fatalf("无法插入 user_tokens 数据: %v", err)
		}
	}
	rows.Close()

	// 提交事务
	if err := tx.Commit(); err != nil {
		t.Fatalf("无法提交事务: %v", err)
	}
}

// waitForServerReady 使用轮询方式检查服务器是否就绪
// 每100ms检查一次，总超时时间5秒
func waitForServerReady(t *testing.T, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	baseURL := config.GetAPIBaseURL()
	token := config.GetAPIToken()

	for time.Now().Before(deadline) {
		// 尝试访问 /api/v1/auth/me 端点来检查服务器是否就绪
		req, err := http.NewRequest("GET", baseURL+"/auth/me", nil)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			// 只要服务器响应（无论是 200 还是 401），都认为服务器已就绪
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("等待服务器就绪超时（%v）", timeout)
}

func useTestDB(t *testing.T, dbPath string) {
	copyAuthDataToTestDB(t, dbPath)

	output, err := runCLIWithEnv(map[string]string{"TASKMANAGER_DB_PATH": dbPath}, "server", "restart")
	if err != nil {
		t.Fatalf("启动临时 server 失败: %v\n%s", err, output)
	}

	// 使用轮询方式等待 server 就绪，总超时5秒
	if err := waitForServerReady(t, 5*time.Second); err != nil {
		t.Fatalf("等待临时 server 就绪失败: %v", err)
	}

	t.Cleanup(func() {
		// 恢复默认数据库的 server
		output, err := runCLI("server", "restart")
		if err != nil {
			t.Fatalf("恢复默认 server 失败: %v\n%s", err, output)
		}
		// 使用轮询方式等待 server 恢复，总超时5秒
		if err := waitForServerReady(t, 5*time.Second); err != nil {
			t.Fatalf("等待默认 server 恢复失败: %v", err)
		}
	})
}

// ========== Admin 命令测试 ==========

func TestCLI_CreateAdmin(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("create-admin")
	if err != nil {
		t.Fatalf("create-admin 失败: %v\n%s", err, output)
	}

	// 验证输出包含成功提示
	if !strings.Contains(output, "管理员用户创建成功") && !strings.Contains(output, "管理员用户已存在") {
		t.Errorf("输出不包含成功提示:\n%s", output)
	}
}

func TestCLI_DeleteAdmin(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("delete-admin")
	if err != nil {
		t.Fatalf("delete-admin 失败: %v\n%s", err, output)
	}

	// 验证输出包含成功提示
	if !strings.Contains(output, "管理员用户已删除") && !strings.Contains(output, "管理员用户不存在") {
		t.Errorf("输出不包含成功提示:\n%s", output)
	}
}

// ========== Agent 命令测试 ==========

func TestCLI_AgentList(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("agent", "list")
	if err != nil {
		t.Fatalf("agent list 失败: %v\n%s", err, output)
	}

	// 验证输出包含 JSON 数组（CLI 输出纯 JSON）
	if !strings.HasPrefix(strings.TrimSpace(output), "[") {
		t.Errorf("输出不是 JSON 数组:\n%s", output)
	}

	// 验证输出包含基本字段
	if !strings.Contains(output, `"name"`) {
		t.Errorf("输出不包含 'name' 字段:\n%s", output)
	}

	t.Logf("agent list 输出:\n%s", output)
}

// ========== Project 命令测试 ==========

func TestCLI_ProjectList(t *testing.T) {
	requiresAPIToken(t)
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
	requiresAPIToken(t)
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
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("requirement", "list")
	if err != nil {
		t.Fatalf("requirement list 失败: %v\n%s", err, output)
	}

	// 验证输出是有效的 JSON 数组
	if !strings.HasPrefix(output, "[") {
		t.Errorf("输出不是 JSON 数组:\n%s", output)
	}

	t.Logf("requirement list 输出:\n%s", output)
}

func TestCLI_RequirementListWithHeartbeat(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 测试 --all 标志
	output, err := runCLI("requirement", "list", "--all")
	if err != nil {
		t.Fatalf("requirement list --all 失败: %v\n%s", err, output)
	}

	// 验证输出是有效的 JSON 数组
	if !strings.HasPrefix(output, "[") {
		t.Errorf("输出不是 JSON 数组:\n%s", output)
	}

	t.Logf("requirement list --all 输出:\n%s", output)
}

func TestCLI_RequirementCreate(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 需要先有一个项目 ID
	// 创建测试项目
	db, dbPath, dbCleanup := setupTestDB(t)
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

	// 切换到临时数据库的 server
	useTestDB(t, dbPath)

	// 使用项目 ID 创建需求，注入临时数据库路径
	output, err := runCLIWithEnv(
		map[string]string{"TASKMANAGER_DB_PATH": dbPath},
		"requirement", "create",
		"--project-id", project.ID().String(),
		"--title", "E2E 测试需求",
		"--description", "这是一个 E2E 测试需求")
	if err != nil {
		t.Fatalf("requirement create 失败: %v\n%s", err, output)
	}

	// 验证输出包含成功信息 (JSON 格式)
	if !strings.Contains(output, `"message":"created"`) && !strings.Contains(output, `"message": "created"`) {
		t.Errorf("输出不包含创建成功信息:\n%s", output)
	}

	t.Logf("requirement create 输出:\n%s", output)
}

func TestCLI_RequirementGet(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 先创建一个测试需求
	db, dbPath, dbCleanup := setupTestDB(t)
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

	// 切换到临时数据库的 server
	useTestDB(t, dbPath)

	// 使用 requirement get 获取详情，注入临时数据库路径
	output, err := runCLIWithEnv(
		map[string]string{"TASKMANAGER_DB_PATH": dbPath},
		"requirement", "get", "--id", requirement.ID().String())
	if err != nil {
		t.Fatalf("requirement get 失败: %v\n%s", err, output)
	}

	// 验证输出是有效的 JSON 且包含需求标题
	if !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON 对象:\n%s", output)
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
	requiresAPIToken(t)
	buildCLI(t)

	// 1. 创建临时数据库并准备测试数据
	db, dbPath, dbCleanup := setupTestDB(t)
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

	// 切换到临时数据库的 server，统一测试数据和 CLI 调用的数据库路径
	useTestDB(t, dbPath)

	// 2. 列出项目验证
	output, err := runCLIWithEnv(
		map[string]string{"TASKMANAGER_DB_PATH": dbPath},
		"project", "list")
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
	output, err = runCLIWithEnv(
		map[string]string{"TASKMANAGER_DB_PATH": dbPath},
		"requirement", "get", "--id", requirement.ID().String())
	if err != nil {
		t.Fatalf("requirement get 失败: %v\n%s", err, output)
	}
	// 验证输出是有效的 JSON 且包含需求标题
	if !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON 对象:\n%s", output)
	}
	if !strings.Contains(output, "E2E 工作流测试需求") {
		t.Errorf("需求详情不包含标题:\n%s", output)
	}

	// 5. 列出需求
	output, err = runCLIWithEnv(
		map[string]string{"TASKMANAGER_DB_PATH": dbPath},
		"requirement", "list")
	if err != nil {
		t.Fatalf("requirement list 失败: %v\n%s", err, output)
	}
	// 验证输出是有效的 JSON 数组
	if !strings.HasPrefix(output, "[") {
		t.Errorf("输出不是 JSON 数组:\n%s", output)
	}

	t.Log("完整工作流测试通过!")
}

// Helper: 确保 CLI 和 Server 已编译
func TestMain(m *testing.M) {
	gitRoot := getGitRoot()
	backendDir := filepath.Join(gitRoot, "backend")

	// 确保 CLI 可以编译
	cmd := exec.Command("go", "build", "-o", cliBinary, "./cmd/cli")
	cmd.Dir = backendDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("编译 CLI 失败: %v\n%s", err, string(output))
		os.Exit(1)
	}

	// 将 server 编译到 CLI 同级目录，供 server restart 优先使用
	serverBinary := filepath.Join(filepath.Dir(cliBinary), "taskmanager-server")
	cmd = exec.Command("go", "build", "-o", serverBinary, "./cmd/server")
	cmd.Dir = backendDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("编译 Server 失败: %v\n%s", err, string(output))
		os.Exit(1)
	}

	os.Exit(m.Run())
}
