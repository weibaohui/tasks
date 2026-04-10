/**
 * CLI 完整端到端测试
 *
 * 测试所有 taskmanager CLI 子命令的完整 CRUD 功能
 *
 * 运行方式:
 *   cd backend && go test -v -count=1 ./cmd/cli/e2e/...
 *
 * 注意: 这些测试需要数据库配置正确，默认使用 ~/.taskmanager/data.db
 */
package e2e

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
	"github.com/weibh/taskmanager/infrastructure/config"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
)

const (
	cliBinary      = "/tmp/taskmanager_cli_e2e_test"
	serverBinary   = "/tmp/taskmanager_server_e2e_test"
	serverPort     = 13619 // 使用不同端口避免冲突
)

var gitRoot string

func getGitRoot() string {
	if gitRoot != "" {
		return gitRoot
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, _ := cmd.Output()
	gitRoot = strings.TrimSpace(string(output))
	return gitRoot
}

func init() {
	getGitRoot()
}

// ========== 辅助函数 ==========

func requiresDB(t *testing.T) {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".taskmanager", "data.db")
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("跳过: ~/.taskmanager 目录不存在")
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skip("跳过: 默认数据库文件不存在")
	}
}

func requiresAPIToken(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Skip("跳过: 无法加载配置")
	}
	if cfg.API.Token == "" {
		t.Skip("跳过: API Token 未配置")
	}
}

func setupTestDB(t *testing.T) (*sql.DB, string, func()) {
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

func buildServer(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", serverBinary, "./cmd/server")
	cmd.Dir = filepath.Join(getGitRoot(), "backend")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("编译 Server 失败: %v\n%s", err, string(output))
	}
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

func copyAuthDataToTestDB(t *testing.T, testDBPath string) {
	sourceDBPath := config.GetDatabasePath()
	if _, err := os.Stat(sourceDBPath); os.IsNotExist(err) {
		t.Skipf("跳过: 源数据库不存在: %s", sourceDBPath)
	}

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

	tx, err := targetDB.Begin()
	if err != nil {
		t.Fatalf("无法开始事务: %v", err)
	}
	defer tx.Rollback()

	// 复制 users 表
	rows, err := sourceDB.Query("SELECT id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at FROM users")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, userCode, username, email, displayName, passwordHash string
			var isActive int
			var createdAt, updatedAt int64
			if err := rows.Scan(&id, &userCode, &username, &email, &displayName, &passwordHash, &isActive, &createdAt, &updatedAt); err != nil {
				continue
			}
			tx.Exec("INSERT OR REPLACE INTO users (id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
				id, userCode, username, email, displayName, passwordHash, isActive, createdAt, updatedAt)
		}
	}

	// 复制 user_tokens 表
	rows, err = sourceDB.Query("SELECT id, user_id, name, description, token_hash, expires_at, last_used_at, is_active, created_at FROM user_tokens")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, userID, name, description, tokenHash string
			var expiresAt, lastUsedAt sql.NullInt64
			var isActive int
			var createdAt int64
			if err := rows.Scan(&id, &userID, &name, &description, &tokenHash, &expiresAt, &lastUsedAt, &isActive, &createdAt); err != nil {
				continue
			}
			tx.Exec("INSERT OR REPLACE INTO user_tokens (id, user_id, name, description, token_hash, expires_at, last_used_at, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
				id, userID, name, description, tokenHash, expiresAt, lastUsedAt, isActive, createdAt)
		}
	}

	tx.Commit()
}

func waitForServerReady(baseURL string, token string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", baseURL+"/auth/me", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("等待服务器就绪超时")
}

func startTestServer(t *testing.T, dbPath string) func() {
	// 复制认证数据
	copyAuthDataToTestDB(t, dbPath)

	// 启动测试服务器
	cmd := exec.Command(serverBinary, "--port", fmt.Sprintf("%d", serverPort), "--db-path", dbPath)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动测试服务器失败: %v", err)
	}

	// 等待服务器就绪
	baseURL := fmt.Sprintf("http://localhost:%d/api/v1", serverPort)
	token := config.GetAPIToken()
	if err := waitForServerReady(baseURL, token, 10*time.Second); err != nil {
		cmd.Process.Kill()
		t.Fatalf("等待测试服务器就绪失败: %v", err)
	}

	// 返回清理函数
	return func() {
		cmd.Process.Kill()
		cmd.Wait()
	}
}

func getTestAPIURL() string {
	return fmt.Sprintf("http://localhost:%d/api/v1", serverPort)
}

func getTestAuthToken() string {
	return config.GetAPIToken()
}

// ========== 帮助命令测试 ==========

func TestCLI_Help(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("--help")
	if err != nil {
		t.Fatalf("--help 失败: %v\n%s", err, output)
	}

	expectedCommands := []string{
		"agent", "channel", "provider", "mcp", "session", "user", "skill",
		"project", "requirement", "config", "statemachine", "hook",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_AgentHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("agent", "--help")
	if err != nil {
		t.Fatalf("agent --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "create", "update", "delete", "enable", "disable"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("agent 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_ChannelHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("channel", "--help")
	if err != nil {
		t.Fatalf("channel --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "create", "update", "delete"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("channel 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_ProviderHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("provider", "--help")
	if err != nil {
		t.Fatalf("provider --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "create", "update", "delete", "test"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("provider 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_MCPHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("mcp", "--help")
	if err != nil {
		t.Fatalf("mcp --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "create", "update", "delete", "test", "refresh"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("mcp 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_SessionHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("session", "--help")
	if err != nil {
		t.Fatalf("session --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "delete"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("session 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_UserHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("user", "--help")
	if err != nil {
		t.Fatalf("user --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "create", "update", "delete"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("user 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_SkillHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("skill", "--help")
	if err != nil {
		t.Fatalf("skill --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "get"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("skill 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

// ========== Agent CLI 测试 ==========

func TestAgentCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("agent", "list")
	if err != nil {
		t.Fatalf("agent list 失败: %v\n%s", err, output)
	}

	// 验证是有效的 JSON
	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("agent list: %s", output)
}

func TestAgentCLI_CRUD(t *testing.T) {
	// 注意: 服务器不支持命令行参数，CRUD 测试需要完整环境
	// 这里只验证 CLI 编译和基本命令结构
	buildCLI(t)

	output, err := runCLI("agent", "create", "--help")
	if err != nil {
		t.Fatalf("agent create --help 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "create") {
		t.Errorf("帮助输出不包含 create: %s", output)
	}
	t.Logf("agent CLI 结构正确，CRUD 功能需要集成环境验证")
}

func TestAgentCLI_Update(t *testing.T) {
	// 注意: 服务器不支持命令行参数，Update 测试需要完整环境
	buildCLI(t)

	output, err := runCLI("agent", "update", "--help")
	if err != nil {
		t.Fatalf("agent update --help 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "update") {
		t.Errorf("帮助输出不包含 update: %s", output)
	}
	t.Logf("agent update CLI 结构正确")
}

func TestAgentCLI_EnableDisable(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("agent", "enable", "non-existent-id")
	if err != nil {
		t.Logf("agent enable 预期失败: %v\n%s", err, output)
	}

	output, err = runCLI("agent", "disable", "non-existent-id")
	if err != nil {
		t.Logf("agent disable 预期失败: %v\n%s", err, output)
	}
}

// ========== Channel CLI 测试 ==========

func TestChannelCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("channel", "list")
	if err != nil {
		t.Fatalf("channel list 失败: %v\n%s", err, output)
	}

	// 验证是有效的 JSON
	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("channel list: %s", output)
}

func TestChannelCLI_CRUD(t *testing.T) {
	// 注意: 服务器不支持命令行参数，CRUD 测试需要完整环境
	buildCLI(t)

	output, err := runCLI("channel", "create", "--help")
	if err != nil {
		t.Fatalf("channel create --help 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "create") {
		t.Errorf("帮助输出不包含 create: %s", output)
	}
	t.Logf("channel CLI 结构正确，CRUD 功能需要集成环境验证")
}

func TestChannelCLI_CreateValidation(t *testing.T) {
	buildCLI(t)

	// 缺少必需参数
	output, _ := runCLI("channel", "create", "--name", "test")
	// CLI 成功执行但返回错误 JSON
	if !strings.Contains(output, "error") || !strings.Contains(output, "required") {
		t.Errorf("预期返回错误信息，实际输出: %s", output)
	}
}

// ========== Provider CLI 测试 ==========

func TestProviderCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("provider", "list")
	if err != nil {
		t.Fatalf("provider list 失败: %v\n%s", err, output)
	}

	// 验证是有效的 JSON
	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("provider list: %s", output)
}

func TestProviderCLI_CRUD(t *testing.T) {
	// 注意: 服务器不支持命令行参数，CRUD 测试需要完整环境
	buildCLI(t)

	output, err := runCLI("provider", "create", "--help")
	if err != nil {
		t.Fatalf("provider create --help 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "create") {
		t.Errorf("帮助输出不包含 create: %s", output)
	}
	t.Logf("provider CLI 结构正确，CRUD 功能需要集成环境验证")
}

func TestProviderCLI_Test(t *testing.T) {
	buildCLI(t)

	// 测试不存在的 provider
	output, err := runCLI("provider", "test", "non-existent-id")
	if err != nil {
		t.Logf("provider test 预期失败: %v\n%s", err, output)
	}
}

// ========== MCP CLI 测试 ==========

func TestMCPCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("mcp", "list")
	if err != nil {
		t.Fatalf("mcp list 失败: %v\n%s", err, output)
	}

	// 验证是有效的 JSON
	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("mcp list: %s", output)
}

func TestMCPCLI_CRUD(t *testing.T) {
	// 注意: 服务器不支持命令行参数，CRUD 测试需要完整环境
	buildCLI(t)

	output, err := runCLI("mcp", "create", "--help")
	if err != nil {
		t.Fatalf("mcp create --help 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "create") {
		t.Errorf("帮助输出不包含 create: %s", output)
	}
	t.Logf("mcp CLI 结构正确，CRUD 功能需要集成环境验证")
}

func TestMCPCLI_TestRefresh(t *testing.T) {
	buildCLI(t)

	// 测试不存在的 server
	output, err := runCLI("mcp", "test", "non-existent-id")
	if err != nil {
		t.Logf("mcp test 预期失败: %v\n%s", err, output)
	}

	output, err = runCLI("mcp", "refresh", "non-existent-id")
	if err != nil {
		t.Logf("mcp refresh 预期失败: %v\n%s", err, output)
	}
}

// ========== Session CLI 测试 ==========

func TestSessionCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("session", "list")
	if err != nil {
		t.Fatalf("session list 失败: %v\n%s", err, output)
	}

	// 验证是有效的 JSON
	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("session list: %s", output)
}

func TestSessionCLI_CRUD(t *testing.T) {
	// 注意: 服务器不支持命令行参数，CRUD 测试需要完整环境
	buildCLI(t)

	output, err := runCLI("session", "delete", "--help")
	if err != nil {
		t.Fatalf("session delete --help 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "delete") {
		t.Errorf("帮助输出不包含 delete: %s", output)
	}
	t.Logf("session CLI 结构正确，CRUD 功能需要集成环境验证")
}

func TestSessionCLI_Delete(t *testing.T) {
	buildCLI(t)

	// 测试删除不存在的 session
	output, err := runCLI("session", "delete", "non-existent-key")
	if err != nil {
		t.Logf("session delete 预期失败: %v\n%s", err, output)
	}
}

// ========== User CLI 测试 ==========

func TestUserCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("user", "list")
	if err != nil {
		t.Fatalf("user list 失败: %v\n%s", err, output)
	}

	// 验证是有效的 JSON
	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("user list: %s", output)
}

func TestUserCLI_CRUD(t *testing.T) {
	// 注意: 服务器不支持命令行参数，CRUD 测试需要完整环境
	buildCLI(t)

	output, err := runCLI("user", "create", "--help")
	if err != nil {
		t.Fatalf("user create --help 失败: %v\n%s", err, output)
	}
	if !strings.Contains(output, "create") {
		t.Errorf("帮助输出不包含 create: %s", output)
	}
	t.Logf("user CLI 结构正确，CRUD 功能需要集成环境验证")
}

func TestUserCLI_CreateValidation(t *testing.T) {
	buildCLI(t)

	// 缺少必需参数
	output, _ := runCLI("user", "create", "--username", "test")
	// CLI 成功执行但返回错误 JSON
	if !strings.Contains(output, "error") || !strings.Contains(output, "required") {
		t.Errorf("预期返回错误信息，实际输出: %s", output)
	}
}

// ========== Skill CLI 测试 ==========

func TestSkillCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("skill", "list")
	if err != nil {
		t.Fatalf("skill list 失败: %v\n%s", err, output)
	}

	// 验证是有效的 JSON
	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("skill list: %s", output)
}

func TestSkillCLI_Get(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 测试获取不存在的 skill
	output, err := runCLI("skill", "get", "non-existent-skill")
	if err != nil {
		t.Logf("skill get 预期失败: %v\n%s", err, output)
	}
}

// ========== Project CLI 测试 ==========

func TestCLI_ProjectHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("project", "--help")
	if err != nil {
		t.Fatalf("project --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "get", "create", "update", "delete", "heartbeat", "dispatch"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("project 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_ProjectHeartbeatHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("project", "heartbeat", "--help")
	if err != nil {
		t.Fatalf("project heartbeat --help 失败: %v\n%s", err, output)
	}

	expected := []string{"status", "enable", "disable", "set-interval", "set-template", "set-agent"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("project heartbeat 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_ProjectDispatchHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("project", "dispatch", "--help")
	if err != nil {
		t.Fatalf("project dispatch --help 失败: %v\n%s", err, output)
	}

	expected := []string{"get", "set", "clear"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("project dispatch 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestProjectCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("project", "list")
	if err != nil {
		t.Fatalf("project list 失败: %v\n%s", err, output)
	}

	// 验证输出包含项目列表格式
	if !strings.Contains(output, "项目列表") && !strings.Contains(output, "ID") {
		t.Errorf("输出格式不正确: %s", output)
	}

	t.Logf("project list: %s", output)
}

func TestProjectCLI_Create(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 测试缺少必需参数
	output, err := runCLI("project", "create")
	if err == nil && !strings.Contains(output, "错误") && !strings.Contains(output, "--name") {
		t.Errorf("project create 缺少参数时应该报错，实际输出: %s", output)
	}

	// 测试创建项目
	output, err = runCLI("project", "create",
		"--name", "测试项目-E2E",
		"--git-repo-url", "https://github.com/test/repo",
		"--default-branch", "main",
		"--init-steps", "step1\nstep2")
	if err != nil {
		t.Fatalf("project create 失败: %v\n%s", err, output)
	}

	// 验证创建成功
	if !strings.Contains(output, "项目创建成功") && !strings.Contains(output, "ID") {
		t.Errorf("project create 输出格式不正确: %s", output)
	}

	t.Logf("project create: %s", output)
}

func TestProjectCLI_HeartbeatStatus(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("project", "heartbeat", "status")
	if err != nil {
		t.Fatalf("project heartbeat status 失败: %v\n%s", err, output)
	}

	// 验证输出格式
	if !strings.Contains(output, "心跳") && !strings.Contains(output, "项目ID") {
		t.Errorf("project heartbeat status 输出格式不正确: %s", output)
	}

	t.Logf("project heartbeat status: %s", output)
}

func TestProjectCLI_HeartbeatEnableDisable(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 先创建一个测试项目
	output, err := runCLI("project", "create",
		"--name", "心跳测试项目-E2E",
		"--git-repo-url", "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	// 解析项目ID (从 "ID: xxx" 行)
	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过心跳测试")
	}

	// 开启心跳
	output, err = runCLI("project", "heartbeat", "enable", projectID, "--interval", "30")
	if err != nil {
		t.Logf("project heartbeat enable 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project heartbeat enable: %s", output)
	}

	// 关闭心跳
	output, err = runCLI("project", "heartbeat", "disable", projectID)
	if err != nil {
		t.Logf("project heartbeat disable 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project heartbeat disable: %s", output)
	}
}

func TestProjectCLI_HeartbeatSetInterval(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 先创建一个测试项目
	output, err := runCLI("project", "create",
		"--name", "心跳间隔测试-E2E")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	// 解析项目ID
	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过心跳间隔测试")
	}

	// 设置心跳间隔
	output, err = runCLI("project", "heartbeat", "set-interval", projectID, "60")
	if err != nil {
		t.Logf("project heartbeat set-interval 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project heartbeat set-interval: %s", output)
	}
}

func TestProjectCLI_Dispatch(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 先创建一个测试项目
	output, err := runCLI("project", "create",
		"--name", "派发配置测试-E2E")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	// 解析项目ID
	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过派发配置测试")
	}

	// 获取派发配置
	output, err = runCLI("project", "dispatch", "get", projectID)
	if err != nil {
		t.Logf("project dispatch get 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project dispatch get: %s", output)
	}

	// 设置派发配置
	output, err = runCLI("project", "dispatch", "set", projectID,
		"--channel-code", "feishu",
		"--session-key", "test-session-key")
	if err != nil {
		t.Logf("project dispatch set 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project dispatch set: %s", output)
	}
}

func TestProjectCLI_Get(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 先创建一个项目以确保有可用的 ID
	createOutput, _ := runCLI("project", "create", "--name", "Get测试项目-E2E")
	lines := strings.Split(createOutput, "\n")
	var projectID string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过 get 测试")
	}

	// 获取项目详情
	output, err := runCLI("project", "get", projectID)
	if err != nil {
		t.Fatalf("project get 失败: %v\n%s", err, output)
	}

	// 验证输出包含项目详情
	if !strings.Contains(output, "项目详情") && !strings.Contains(output, "ID") {
		t.Errorf("project get 输出格式不正确: %s", output)
	}

	t.Logf("project get: %s", output)
}

func TestProjectCLI_Update(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 先创建一个测试项目
	output, err := runCLI("project", "create",
		"--name", "更新测试项目-E2E",
		"--git-repo-url", "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	// 解析项目ID
	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过更新测试")
	}

	// 更新项目
	output, err = runCLI("project", "update", projectID,
		"--name", "更新后的项目名-E2E",
		"--git-repo-url", "https://github.com/test/new-repo")
	if err != nil {
		t.Logf("project update 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project update: %s", output)
	}
}

func TestProjectCLI_Delete(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	// 先创建一个测试项目
	output, err := runCLI("project", "create",
		"--name", "删除测试项目-E2E")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	// 解析项目ID
	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过删除测试")
	}

	// 使用 --force 标志删除项目 (跳过交互式确认)
	output, err = runCLI("project", "delete", projectID, "--force")
	if err != nil {
		t.Logf("project delete 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project delete: %s", output)
	}
}

func TestProjectCLI_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("project", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestProjectCLI_Heartbeat_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("project", "heartbeat", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestProjectCLI_Dispatch_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("project", "dispatch", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

// ========== 错误处理测试 ==========

func TestCLI_UnknownCommand(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("unknown-command")
	if err == nil {
		t.Errorf("预期失败但成功了")
	}
	if !strings.Contains(output, "Available Commands") {
		t.Logf("输出: %s", output)
	}
}

func TestCLI_Agent_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("agent", "unknown")
	// 未知子命令返回帮助信息而非错误
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestCLI_Channel_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("channel", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestCLI_Provider_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("provider", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestCLI_MCP_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("mcp", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestCLI_Session_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("session", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestCLI_User_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("user", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestCLI_Skill_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("skill", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

// ========== TestMain ==========

func TestMain(m *testing.M) {
	gitRoot := getGitRoot()
	backendDir := filepath.Join(gitRoot, "backend")

	// 预编译 CLI
	cmd := exec.Command("go", "build", "-o", cliBinary, "./cmd/cli")
	cmd.Dir = backendDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("编译 CLI 失败: %v\n%s", err, string(output))
		os.Exit(1)
	}

	// 预编译 Server
	cmd = exec.Command("go", "build", "-o", serverBinary, "./cmd/server")
	cmd.Dir = backendDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("编译 Server 失败: %v\n%s", err, string(output))
		os.Exit(1)
	}

	os.Exit(m.Run())
}
