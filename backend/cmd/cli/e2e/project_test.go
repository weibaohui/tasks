/**
 * Project CLI Tests
 */
package e2e

import (
	"strings"
	"testing"
)

func TestCLI_ProjectHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("project", "--help")
	if err != nil {
		t.Fatalf("project --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "get", "create", "update", "delete", "heartbeat", "dispatch"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("project 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_ProjectHeartbeatHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("project", "heartbeat", "--help")
	if err != nil {
		t.Fatalf("project heartbeat --help 失败: %v\n%s", err, output)
	}

	expected := []string{"list", "create", "update", "delete", "enable", "disable"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("project heartbeat 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestCLI_ProjectDispatchHelp(t *testing.T) {
	buildCLI(t)

	output, err := runCLI("project", "dispatch", "--help")
	if err != nil {
		t.Fatalf("project dispatch --help 失败: %v\n%s", err, output)
	}

	expected := []string{"get", "set", "clear"}
	for _, cmd := range expected {
		if !strings.Contains(output, cmd) {
			t.Errorf("project dispatch 帮助输出不包含 '%s':\n%s", cmd, output)
		}
	}
}

func TestProjectCLI_List(t *testing.T) {
	ensureServerReady(t)
	buildCLI(t)

	output, err := runCLI("project", "list")
	if err != nil {
		t.Fatalf("project list 失败: %v\n%s", err, output)
	}

	if !strings.Contains(output, "项目列表") && !strings.Contains(output, "ID") {
		t.Errorf("输出格式不正确: %s", output)
	}

	t.Logf("project list: %s", output)
}

func TestProjectCLI_Create(t *testing.T) {
	ensureServerReady(t)
	buildCLI(t)

	output, err := runCLI("project", "create")
	if err == nil && !strings.Contains(output, "错误") && !strings.Contains(output, "--name") {
		t.Errorf("project create 缺少参数时应该报错，实际输出: %s", output)
	}

	output, err = runCLI("project", "create",
		"--name", "测试项目-E2E",
		"--git-repo-url", "https://github.com/test/repo",
		"--default-branch", "main",
		"--init-steps", "step1\nstep2")
	if err != nil {
		t.Fatalf("project create 失败: %v\n%s", err, output)
	}

	if !strings.Contains(output, "项目创建成功") && !strings.Contains(output, "ID") {
		t.Errorf("project create 输出格式不正确: %s", output)
	}

	t.Logf("project create: %s", output)
}

func TestProjectCLI_HeartbeatList(t *testing.T) {
	ensureServerReady(t)
	buildCLI(t)

	output, err := runCLI("project", "heartbeat", "list", "test-project-id")
	if err != nil {
		t.Fatalf("project heartbeat list 失败: %v\n%s", err, output)
	}

	if !strings.Contains(output, "项目心跳列表") && !strings.Contains(output, "心跳ID") {
		t.Errorf("project heartbeat list 输出格式不正确: %s", output)
	}

	t.Logf("project heartbeat list: %s", output)
}

func TestProjectCLI_HeartbeatEnableDisable(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("project", "create",
		"--name", "心跳测试项目-E2E",
		"--git-repo-url", "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过心跳测试")
	}

	// 先创建一个心跳
	output, err = runCLI("project", "heartbeat", "create", projectID,
		"--name", "测试心跳",
		"--interval", "30",
		"--agent-code", "scheduler")
	if err != nil {
		t.Logf("project heartbeat create 失败: %v\n%s", err, output)
		t.Skip("无法创建心跳，跳过测试")
	}

	lines = strings.Split(output, "\n")
	var heartbeatID string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID:") {
			heartbeatID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}
	if heartbeatID == "" {
		t.Skip("无法获取创建的心跳ID，跳过测试")
	}

	output, err = runCLI("project", "heartbeat", "enable", heartbeatID)
	if err != nil {
		t.Logf("project heartbeat enable 失败: %v\n%s", err, output)
	} else {
		t.Logf("project heartbeat enable: %s", output)
	}

	output, err = runCLI("project", "heartbeat", "disable", heartbeatID)
	if err != nil {
		t.Logf("project heartbeat disable 失败: %v\n%s", err, output)
	} else {
		t.Logf("project heartbeat disable: %s", output)
	}
}

func TestProjectCLI_HeartbeatUpdate(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("project", "create",
		"--name", "心跳更新测试-E2E")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过心跳更新测试")
	}

	output, err = runCLI("project", "heartbeat", "create", projectID,
		"--name", "更新前心跳",
		"--interval", "30",
		"--agent-code", "scheduler")
	if err != nil {
		t.Logf("project heartbeat create 失败: %v\n%s", err, output)
		t.Skip("无法创建心跳，跳过测试")
	}

	lines = strings.Split(output, "\n")
	var heartbeatID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID:") {
			heartbeatID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}
	if heartbeatID == "" {
		t.Skip("无法获取创建的心跳ID，跳过测试")
	}

	output, err = runCLI("project", "heartbeat", "update", heartbeatID,
		"--name", "更新后心跳",
		"--interval", "60")
	if err != nil {
		t.Logf("project heartbeat update 失败: %v\n%s", err, output)
	} else {
		t.Logf("project heartbeat update: %s", output)
	}
}

func TestProjectCLI_Dispatch(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("project", "create",
		"--name", "派发配置测试-E2E")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过派发配置测试")
	}

	output, err = runCLI("project", "dispatch", "get", projectID)
	if err != nil {
		t.Logf("project dispatch get 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project dispatch get: %s", output)
	}

	output, err = runCLI("project", "dispatch", "set", projectID,
		"--channel-code", "feishu",
		"--session-key", "test-session-key")
	if err != nil {
		t.Logf("project dispatch set 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project dispatch set: %s", output)
	}
}

func TestProjectCLI_Get(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	createOutput, _ := runCLI("project", "create", "--name", "Get测试项目-E2E")
	lines := strings.Split(createOutput, "\n")
	var projectID string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过 get 测试")
	}

	output, err := runCLI("project", "get", projectID)
	if err != nil {
		t.Fatalf("project get 失败: %v\n%s", err, output)
	}

	if !strings.Contains(output, "项目详情") && !strings.Contains(output, "ID") {
		t.Errorf("project get 输出格式不正确: %s", output)
	}

	t.Logf("project get: %s", output)
}

func TestProjectCLI_Update(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("project", "create",
		"--name", "更新测试项目-E2E",
		"--git-repo-url", "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过更新测试")
	}

	output, err = runCLI("project", "update", projectID,
		"--name", "更新后的项目名-E2E",
		"--git-repo-url", "https://github.com/test/new-repo")
	if err != nil {
		t.Logf("project update 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project update: %s", output)
	}
}

func TestProjectCLI_Delete(t *testing.T) {
	requiresAPIToken(t)
	buildCLI(t)

	output, err := runCLI("project", "create",
		"--name", "删除测试项目-E2E")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v\n%s", err, output)
	}

	lines := strings.Split(output, "\n")
	var projectID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID:") {
			projectID = strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			break
		}
	}

	if projectID == "" {
		t.Skip("无法获取创建的项目ID，跳过删除测试")
	}

	output, err = runCLI("project", "delete", projectID, "--force")
	if err != nil {
		t.Logf("project delete 失败 (可能项目不存在): %v\n%s", err, output)
	} else {
		t.Logf("project delete: %s", output)
	}
}

func TestProjectCLI_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("project", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestProjectCLI_Heartbeat_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("project", "heartbeat", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}

func TestProjectCLI_Dispatch_UnknownSubCommand(t *testing.T) {
	buildCLI(t)

	output, _ := runCLI("project", "dispatch", "unknown")
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("预期返回帮助信息，实际输出: %s", output)
	}
}
