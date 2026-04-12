/**
 * Agent CLI Tests
 */
package e2e

import (
	"strings"
	"testing"
)

func TestAgentCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("agent", "list")
	if err != nil {
		t.Fatalf("agent list 失败: %v\n%s", err, output)
	}

	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("agent list: %s", output)
}

func TestAgentCLI_CRUD(t *testing.T) {
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
