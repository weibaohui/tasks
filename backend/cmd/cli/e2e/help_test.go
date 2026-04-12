/**
 * CLI Help Tests
 */
package e2e

import (
	"strings"
	"testing"
)

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
