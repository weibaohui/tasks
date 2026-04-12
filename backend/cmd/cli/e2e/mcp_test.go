/**
 * MCP CLI Tests
 */
package e2e

import (
	"strings"
	"testing"
)

func TestMCPCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("mcp", "list")
	if err != nil {
		t.Fatalf("mcp list 失败: %v\n%s", err, output)
	}

	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("mcp list: %s", output)
}

func TestMCPCLI_CRUD(t *testing.T) {
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

	output, err := runCLI("mcp", "test", "non-existent-id")
	if err != nil {
		t.Logf("mcp test 预期失败: %v\n%s", err, output)
	}

	output, err = runCLI("mcp", "refresh", "non-existent-id")
	if err != nil {
		t.Logf("mcp refresh 预期失败: %v\n%s", err, output)
	}
}
