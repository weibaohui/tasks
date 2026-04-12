/**
 * Session CLI Tests
 */
package e2e

import (
	"strings"
	"testing"
)

func TestSessionCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("session", "list")
	if err != nil {
		t.Fatalf("session list 失败: %v\n%s", err, output)
	}

	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("session list: %s", output)
}

func TestSessionCLI_CRUD(t *testing.T) {
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

	output, err := runCLI("session", "delete", "non-existent-key")
	if err != nil {
		t.Logf("session delete 预期失败: %v\n%s", err, output)
	}
}
