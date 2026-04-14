package application

import (
	"testing"

	"github.com/weibh/taskmanager/domain"
)

func TestResolveReplicaAgentCwd(t *testing.T) {
	workspacePath := "/tmp/ai-devops/proj-001/req-001"

	requirementWithTemp, err := domain.NewRequirement(
		domain.NewRequirementID("req-001"),
		domain.NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		" /tmp/custom-workspace ",
	)
	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	if got := resolveReplicaAgentCwd(requirementWithTemp, workspacePath); got != workspacePath {
		t.Fatalf("期望始终使用完整派发工作目录，实际为: %s", got)
	}

	requirementWithoutTemp, err := domain.NewRequirement(
		domain.NewRequirementID("req-002"),
		domain.NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)
	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	if got := resolveReplicaAgentCwd(requirementWithoutTemp, workspacePath); got != workspacePath {
		t.Fatalf("期望回退到派发工作目录，实际为: %s", got)
	}

	if got := resolveReplicaAgentCwd(nil, workspacePath); got != workspacePath {
		t.Fatalf("期望 nil 需求回退到派发工作目录，实际为: %s", got)
	}
}
