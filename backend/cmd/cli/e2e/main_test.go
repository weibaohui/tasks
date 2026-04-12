/**
 * CLI E2E TestMain
 *
 * 运行方式:
 *   cd backend && go test -v -count=1 ./cmd/cli/e2e/...
 */
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/weibh/taskmanager/infrastructure/config"
)

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

	// 等待服务器就绪（处理并行测试时其他包重启服务器的情况）
	cfg, err := config.Load()
	if err == nil && cfg.API.Token != "" {
		_ = waitForServerReady(cfg.API.BaseURL, cfg.API.Token, 15*time.Second)
	}

	os.Exit(m.Run())
}
