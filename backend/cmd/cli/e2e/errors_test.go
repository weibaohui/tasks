/**
 * CLI Error Handling Tests
 */
package e2e

import (
	"strings"
	"testing"
)

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
