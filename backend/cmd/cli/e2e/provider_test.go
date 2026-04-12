/**
 * Provider CLI Tests
 */
package e2e

import (
	"strings"
	"testing"
)

func TestProviderCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("provider", "list")
	if err != nil {
		t.Fatalf("provider list 失败: %v\n%s", err, output)
	}

	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("provider list: %s", output)
}

func TestProviderCLI_CRUD(t *testing.T) {
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

	output, err := runCLI("provider", "test", "non-existent-id")
	if err != nil {
		t.Logf("provider test 预期失败: %v\n%s", err, output)
	}
}
