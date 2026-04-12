/**
 * Channel CLI Tests
 */
package e2e

import (
	"strings"
	"testing"
)

func TestChannelCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("channel", "list")
	if err != nil {
		t.Fatalf("channel list 失败: %v\n%s", err, output)
	}

	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("channel list: %s", output)
}

func TestChannelCLI_CRUD(t *testing.T) {
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

	output, _ := runCLI("channel", "create", "--name", "test")
	if !strings.Contains(output, "error") || !strings.Contains(output, "required") {
		t.Errorf("预期返回错误信息，实际输出: %s", output)
	}
}
