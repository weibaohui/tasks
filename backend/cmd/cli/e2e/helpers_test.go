/**
 * CLI E2E Test Helpers
 *
 * 提供 CLI 测试的辅助函数和共享常量和变量
 *
 * 运行方式:
 *   cd backend && go test -v -count=1 ./cmd/cli/e2e/...
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
	cliBinary    = "/tmp/taskmanager_cli_e2e_test"
	serverBinary = "/tmp/taskmanager_server_e2e_test"
	serverPort   = 13619
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

func ensureServerReady(t *testing.T) {
	requiresAPIToken(t)
	cfg, _ := config.Load()
	if err := waitForServerReady(cfg.API.BaseURL, cfg.API.Token, 20*time.Second); err != nil {
		t.Fatalf("服务器未就绪: %v", err)
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
	copyAuthDataToTestDB(t, dbPath)

	cmd := exec.Command(serverBinary, "--port", fmt.Sprintf("%d", serverPort), "--db-path", dbPath)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动测试服务器失败: %v", err)
	}

	baseURL := fmt.Sprintf("http://localhost:%d/api/v1", serverPort)
	token := config.GetAPIToken()
	if err := waitForServerReady(baseURL, token, 10*time.Second); err != nil {
		cmd.Process.Kill()
		t.Fatalf("等待测试服务器就绪失败: %v", err)
	}

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
