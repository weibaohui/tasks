/**
 * Skill CLI Tests
 */
package e2e

import (
	"strings"
	"testing"
)

func TestSkillCLI_List(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("skill", "list")
	if err != nil {
		t.Fatalf("skill list 失败: %v\n%s", err, output)
	}

	if !strings.HasPrefix(output, "[") && !strings.HasPrefix(output, "{") {
		t.Errorf("输出不是 JSON: %s", output)
	}

	t.Logf("skill list: %s", output)
}

func TestSkillCLI_Get(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("skill", "get", "non-existent-skill")
	if err != nil {
		t.Logf("skill get 预期失败: %v\n%s", err, output)
	}
}
