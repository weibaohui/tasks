package domain

import "testing"

func TestRequirementDispatchSessionKeySnapshotRoundTrip(t *testing.T) {
	req, err := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)
	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	req.SetDispatchSessionKey("  feishu:chat-001  ")
	if req.DispatchSessionKey() != "feishu:chat-001" {
		t.Fatalf("期望派发会话键被去空格，实际为: %s", req.DispatchSessionKey())
	}

	snap := req.ToSnapshot()
	var restored Requirement
	if err := restored.FromSnapshot(snap); err != nil {
		t.Fatalf("从快照恢复需求失败: %v", err)
	}
	if restored.DispatchSessionKey() != "feishu:chat-001" {
		t.Fatalf("期望恢复后的派发会话键正确，实际为: %s", restored.DispatchSessionKey())
	}
}
