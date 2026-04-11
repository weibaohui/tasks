package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestNewRequirement_Success(t *testing.T) {
	req, err := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"/tmp/workspace",
	)

	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	if req.ID().String() != "req-001" {
		t.Errorf("期望ID为 req-001, 实际为 %s", req.ID().String())
	}

	if req.ProjectID().String() != "proj-001" {
		t.Errorf("期望ProjectID为 proj-001, 实际为 %s", req.ProjectID().String())
	}

	if req.Title() != "需求标题" {
		t.Errorf("期望Title为 需求标题, 实际为 %s", req.Title())
	}

	if req.Description() != "需求描述" {
		t.Errorf("期望Description为 需求描述, 实际为 %s", req.Description())
	}

	if req.AcceptanceCriteria() != "验收标准" {
		t.Errorf("期望AcceptanceCriteria为 验收标准, 实际为 %s", req.AcceptanceCriteria())
	}

	if req.TempWorkspaceRoot() != "/tmp/workspace" {
		t.Errorf("期望TempWorkspaceRoot为 /tmp/workspace, 实际为 %s", req.TempWorkspaceRoot())
	}

	// 验证默认状态
	if req.Status() != RequirementStatusTodo {
		t.Errorf("期望默认状态为 todo, 实际为 %s", req.Status())
	}

	// 验证默认类型
	if req.RequirementType() != RequirementTypeNormal {
		t.Errorf("期望默认类型为 normal, 实际为 %s", req.RequirementType())
	}

	// 验证时间戳
	if req.CreatedAt().IsZero() {
		t.Error("期望CreatedAt不为零值")
	}

	if req.UpdatedAt().IsZero() {
		t.Error("期望UpdatedAt不为零值")
	}
}

func TestNewRequirement_EmptyID(t *testing.T) {
	_, err := NewRequirement(
		NewRequirementID(""),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	if err != ErrRequirementIDRequired {
		t.Errorf("期望返回 ErrRequirementIDRequired, 实际返回 %v", err)
	}
}

func TestNewRequirement_EmptyProjectID(t *testing.T) {
	_, err := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID(""),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	if err != ErrRequirementProjectIDRequired {
		t.Errorf("期望返回 ErrRequirementProjectIDRequired, 实际返回 %v", err)
	}
}

func TestNewRequirement_EmptyTitle(t *testing.T) {
	_, err := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"",
		"需求描述",
		"验收标准",
		"",
	)

	if err != ErrRequirementTitleRequired {
		t.Errorf("期望返回 ErrRequirementTitleRequired, 实际返回 %v", err)
	}
}

func TestNewRequirement_TitleWithOnlySpaces(t *testing.T) {
	_, err := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"   ",
		"需求描述",
		"验收标准",
		"",
	)

	if err != ErrRequirementTitleRequired {
		t.Errorf("期望返回 ErrRequirementTitleRequired, 实际返回 %v", err)
	}
}

func TestRequirementStatus_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		status   RequirementStatus
		expected RequirementStatus
	}{
		{"旧状态 in_progress 转换为 preparing", "in_progress", RequirementStatusPreparing},
		{"todo 保持不变", RequirementStatusTodo, RequirementStatusTodo},
		{"preparing 保持不变", "preparing", RequirementStatusPreparing},
		{"coding 保持不变", "coding", RequirementStatusCoding},
		{"pr_opened 保持不变", "pr_opened", RequirementStatusPROpened},
		{"failed 保持不变", "failed", RequirementStatusFailed},
		{"completed 保持不变", "completed", RequirementStatusCompleted},
		{"done 保持不变", "done", RequirementStatus("done")},
		{"无效状态保持不变", "invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.Normalize()
			if result != tt.expected {
				t.Errorf("状态 %s: 期望 %s, 实际 %s", tt.status, tt.expected, result)
			}
		})
	}
}

func TestRequirement_CanDispatch(t *testing.T) {
	tests := []struct {
		name     string
		status   RequirementStatus
		expected bool
	}{
		{"todo 状态可以派发", RequirementStatusTodo, true},
		{"preparing 状态不可派发", RequirementStatusPreparing, false},
		{"coding 状态不可派发", RequirementStatusCoding, false},
		{"pr_opened 状态不可派发", RequirementStatusPROpened, false},
		{"failed 状态不可派发", RequirementStatusFailed, false},
		{"completed 状态不可派发", RequirementStatusCompleted, false},
		{"done 状态不可派发", RequirementStatus("done"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequirementWithStatus(tt.status)
			result := req.CanDispatch()
			if result != tt.expected {
				t.Errorf("状态 %s: 期望 CanDispatch=%v, 实际 %v", tt.status, tt.expected, result)
			}
		})
	}
}

func TestRequirement_CanRedispatch(t *testing.T) {
	tests := []struct {
		name     string
		status   RequirementStatus
		expected bool
	}{
		{"todo 状态不可重新派发", RequirementStatusTodo, false},
		{"preparing 状态可以重新派发", RequirementStatusPreparing, true},
		{"coding 状态可以重新派发", RequirementStatusCoding, true},
		{"pr_opened 状态可以重新派发", RequirementStatusPROpened, true},
		{"failed 状态可以重新派发", RequirementStatusFailed, true},
		{"completed 状态可以重新派发", RequirementStatusCompleted, true},
		{"done 状态可以重新派发", RequirementStatus("done"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequirementWithStatus(tt.status)
			result := req.CanRedispatch()
			if result != tt.expected {
				t.Errorf("状态 %s: 期望 CanRedispatch=%v, 实际 %v", tt.status, tt.expected, result)
			}
		})
	}
}

func TestRequirement_StartDispatch_Success(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	time.Sleep(10 * time.Millisecond) // 确保时间戳不同
	err := req.StartDispatch("agent-001")

	if err != nil {
		t.Fatalf("StartDispatch 失败: %v", err)
	}

	// 验证状态变更
	if req.Status() != RequirementStatusPreparing {
		t.Errorf("期望状态变为 preparing, 实际为 %s", req.Status())
	}

	// 验证指派Agent
	if req.AssigneeAgentCode() != "agent-001" {
		t.Errorf("期望AssigneeAgentCode为 agent-001, 实际为 %s", req.AssigneeAgentCode())
	}

	// 验证 StartedAt 被设置
	if req.StartedAt() == nil {
		t.Error("期望 StartedAt 被设置")
	}

	// 验证 LastError 被清空
	if req.LastError() != "" {
		t.Errorf("期望 LastError 被清空, 实际为 %s", req.LastError())
	}
}

func TestRequirement_StartDispatch_InvalidState(t *testing.T) {
	tests := []struct {
		name   string
		status RequirementStatus
	}{
		{"preparing 状态", RequirementStatusPreparing},
		{"coding 状态", RequirementStatusCoding},
		{"pr_opened 状态", RequirementStatusPROpened},
		{"failed 状态", RequirementStatusFailed},
		{"completed 状态", RequirementStatusCompleted},
		{"done 状态", RequirementStatus("done")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequirementWithStatus(tt.status)
			err := req.StartDispatch("agent-001")

			if err != ErrRequirementCannotDispatch {
				t.Errorf("期望返回 ErrRequirementCannotDispatch, 实际返回 %v", err)
			}
		})
	}
}

func TestRequirement_MarkCoding_Success(t *testing.T) {
	req := createRequirementWithStatus(RequirementStatusPreparing)

	time.Sleep(10 * time.Millisecond)
	err := req.MarkCoding("/workspace/test", "replica-001")

	if err != nil {
		t.Fatalf("MarkCoding 失败: %v", err)
	}

	// 验证状态变更
	if req.Status() != RequirementStatusCoding {
		t.Errorf("期望状态变为 coding, 实际为 %s", req.Status())
	}

	// 验证工作目录
	if req.WorkspacePath() != "/workspace/test" {
		t.Errorf("期望WorkspacePath为 /workspace/test, 实际为 %s", req.WorkspacePath())
	}

	// 验证分身代码
	if req.ReplicaAgentCode() != "replica-001" {
		t.Errorf("期望ReplicaAgentCode为 replica-001, 实际为 %s", req.ReplicaAgentCode())
	}
}

func TestRequirement_MarkCoding_InvalidState(t *testing.T) {
	invalidStates := []RequirementStatus{
		RequirementStatusTodo,
		RequirementStatusCoding,
		RequirementStatusPROpened,
		RequirementStatusFailed,
		RequirementStatusCompleted,
		RequirementStatus("done"),
	}

	for _, status := range invalidStates {
		t.Run(string(status), func(t *testing.T) {
			req := createRequirementWithStatus(status)
			err := req.MarkCoding("/workspace/test", "replica-001")

			if err != ErrRequirementCannotDispatch {
				t.Errorf("期望返回 ErrRequirementCannotDispatch, 实际返回 %v", err)
			}
		})
	}
}

func TestRequirement_MarkPROpened(t *testing.T) {
	req := createRequirementWithStatus(RequirementStatusCoding)

	time.Sleep(10 * time.Millisecond)
	req.MarkPROpened()

	// 验证状态变更
	if req.Status() != RequirementStatusPROpened {
		t.Errorf("期望状态变为 pr_opened, 实际为 %s", req.Status())
	}

	// 验证 LastError 被清空
	if req.LastError() != "" {
		t.Errorf("期望 LastError 被清空, 实际为 %s", req.LastError())
	}

	// 验证 CompletedAt 被设置
	if req.CompletedAt() == nil {
		t.Error("期望 CompletedAt 被设置")
	}
}

func TestRequirement_MarkFailed(t *testing.T) {
	req := createRequirementWithStatus(RequirementStatusCoding)

	time.Sleep(10 * time.Millisecond)
	req.MarkFailed("执行失败: 网络错误")

	// 验证状态变更
	if req.Status() != RequirementStatusFailed {
		t.Errorf("期望状态变为 failed, 实际为 %s", req.Status())
	}

	// 验证 LastError 被设置
	if req.LastError() != "执行失败: 网络错误" {
		t.Errorf("期望 LastError 为 '执行失败: 网络错误', 实际为 %s", req.LastError())
	}
}

func TestRequirement_MarkCompleted(t *testing.T) {
	req := createRequirementWithStatus(RequirementStatusCoding)

	time.Sleep(10 * time.Millisecond)
	req.MarkCompleted()

	// 验证状态变更
	if req.Status() != RequirementStatusCompleted {
		t.Errorf("期望状态变为 completed, 实际为 %s", req.Status())
	}

	// 验证 CompletedAt 被设置
	if req.CompletedAt() == nil {
		t.Error("期望 CompletedAt 被设置")
	}
}

func TestRequirement_Redispatch_Success(t *testing.T) {
	// 创建一个非 todo 状态的需求
	req := createRequirementWithStatus(RequirementStatusFailed)
	req.SetDispatchSessionKey("session-001")
	req.SetWorkspacePath("/workspace/test")
	req.SetReplicaAgentCode("replica-001")

	time.Sleep(10 * time.Millisecond)
	err := req.Redispatch()

	if err != nil {
		t.Fatalf("Redispatch 失败: %v", err)
	}

	// 验证状态重置为 todo
	if req.Status() != RequirementStatusTodo {
		t.Errorf("期望状态变为 todo, 实际为 %s", req.Status())
	}

	// 验证字段被清空
	if req.AssigneeAgentCode() != "" {
		t.Errorf("期望 AssigneeAgentCode 被清空, 实际为 %s", req.AssigneeAgentCode())
	}

	if req.ReplicaAgentCode() != "" {
		t.Errorf("期望 ReplicaAgentCode 被清空, 实际为 %s", req.ReplicaAgentCode())
	}

	if req.WorkspacePath() != "" {
		t.Errorf("期望 WorkspacePath 被清空, 实际为 %s", req.WorkspacePath())
	}

	if req.LastError() != "" {
		t.Errorf("期望 LastError 被清空, 实际为 %s", req.LastError())
	}

	if req.StartedAt() != nil {
		t.Error("期望 StartedAt 被清空")
	}

	if req.CompletedAt() != nil {
		t.Error("期望 CompletedAt 被清空")
	}

	if req.ClaudeRuntimePrompt() != "" {
		t.Errorf("期望 ClaudeRuntimePrompt 被清空, 实际为 %s", req.ClaudeRuntimePrompt())
	}
}

func TestRequirement_Redispatch_InvalidState(t *testing.T) {
	req := createRequirementWithStatus(RequirementStatusTodo)

	err := req.Redispatch()

	if err != ErrRequirementCannotDispatch {
		t.Errorf("期望返回 ErrRequirementCannotDispatch, 实际返回 %v", err)
	}
}

func TestRequirement_UpdateContent_Success(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"原标题",
		"原描述",
		"原验收标准",
		"/tmp/old",
	)

	oldUpdatedAt := req.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	err := req.UpdateContent("新标题", "新描述", "新验收标准", "/tmp/new")

	if err != nil {
		t.Fatalf("UpdateContent 失败: %v", err)
	}

	if req.Title() != "新标题" {
		t.Errorf("期望 Title 更新为 '新标题', 实际为 %s", req.Title())
	}

	if req.Description() != "新描述" {
		t.Errorf("期望 Description 更新为 '新描述', 实际为 %s", req.Description())
	}

	if req.AcceptanceCriteria() != "新验收标准" {
		t.Errorf("期望 AcceptanceCriteria 更新为 '新验收标准', 实际为 %s", req.AcceptanceCriteria())
	}

	if req.TempWorkspaceRoot() != "/tmp/new" {
		t.Errorf("期望 TempWorkspaceRoot 更新为 '/tmp/new', 实际为 %s", req.TempWorkspaceRoot())
	}

	if !req.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestRequirement_UpdateContent_EmptyTitle(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"原标题",
		"原描述",
		"原验收标准",
		"",
	)

	err := req.UpdateContent("", "新描述", "新验收标准", "/tmp/new")

	if err != ErrRequirementTitleRequired {
		t.Errorf("期望返回 ErrRequirementTitleRequired, 实际返回 %v", err)
	}

	// 验证内容未被修改
	if req.Title() != "原标题" {
		t.Errorf("标题不应被修改, 期望 '原标题', 实际为 %s", req.Title())
	}
}

func TestRequirement_UpdateContent_TitleWithOnlySpaces(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"原标题",
		"原描述",
		"原验收标准",
		"",
	)

	err := req.UpdateContent("   ", "新描述", "新验收标准", "/tmp/new")

	if err != ErrRequirementTitleRequired {
		t.Errorf("期望返回 ErrRequirementTitleRequired, 实际返回 %v", err)
	}

	// 验证内容未被修改
	if req.Title() != "原标题" {
		t.Errorf("标题不应被修改, 期望 '原标题', 实际为 %s", req.Title())
	}
}

func TestRequirement_StartClaudeRuntime(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	oldUpdatedAt := req.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	req.StartClaudeRuntime()

	// 验证状态
	if req.ClaudeRuntimeStatus() != RuntimeStatusRunning {
		t.Errorf("期望 ClaudeRuntimeStatus 为 'running', 实际为 %s", req.ClaudeRuntimeStatus())
	}

	// 验证开始时间
	if req.ClaudeRuntimeStartedAt() == nil {
		t.Error("期望 ClaudeRuntimeStartedAt 被设置")
	}

	// 验证结束时间被清空
	if req.ClaudeRuntimeEndedAt() != nil {
		t.Error("期望 ClaudeRuntimeEndedAt 被清空")
	}

	// 验证错误被清空
	if req.ClaudeRuntimeError() != "" {
		t.Errorf("期望 ClaudeRuntimeError 被清空, 实际为 %s", req.ClaudeRuntimeError())
	}

	// 验证 UpdatedAt
	if !req.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestRequirement_EndClaudeRuntime_Success(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	req.StartClaudeRuntime()
	time.Sleep(10 * time.Millisecond)

	oldUpdatedAt := req.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	req.EndClaudeRuntime(true, "")

	// 验证状态
	if req.ClaudeRuntimeStatus() != RuntimeStatusCompleted {
		t.Errorf("期望 ClaudeRuntimeStatus 为 'completed', 实际为 %s", req.ClaudeRuntimeStatus())
	}

	// 验证结束时间
	if req.ClaudeRuntimeEndedAt() == nil {
		t.Error("期望 ClaudeRuntimeEndedAt 被设置")
	}

	// 验证错误为空
	if req.ClaudeRuntimeError() != "" {
		t.Errorf("期望 ClaudeRuntimeError 为空, 实际为 %s", req.ClaudeRuntimeError())
	}

	// 验证 UpdatedAt
	if !req.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestRequirement_EndClaudeRuntime_Failure(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	req.StartClaudeRuntime()
	time.Sleep(10 * time.Millisecond)

	req.EndClaudeRuntime(false, "执行超时")

	// 验证状态
	if req.ClaudeRuntimeStatus() != RuntimeStatusFailed {
		t.Errorf("期望 ClaudeRuntimeStatus 为 'failed', 实际为 %s", req.ClaudeRuntimeStatus())
	}

	// 验证错误信息
	if req.ClaudeRuntimeError() != "执行超时" {
		t.Errorf("期望 ClaudeRuntimeError 为 '执行超时', 实际为 %s", req.ClaudeRuntimeError())
	}

	// 验证结束时间
	if req.ClaudeRuntimeEndedAt() == nil {
		t.Error("期望 ClaudeRuntimeEndedAt 被设置")
	}
}

func TestRequirement_SetTokenUsage(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	oldUpdatedAt := req.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	req.SetTokenUsage(1000, 500, 1500)

	// 验证 Token 统计
	if req.PromptTokens() != 1000 {
		t.Errorf("期望 PromptTokens 为 1000, 实际为 %d", req.PromptTokens())
	}

	if req.CompletionTokens() != 500 {
		t.Errorf("期望 CompletionTokens 为 500, 实际为 %d", req.CompletionTokens())
	}

	if req.TotalTokens() != 1500 {
		t.Errorf("期望 TotalTokens 为 1500, 实际为 %d", req.TotalTokens())
	}

	// 验证 UpdatedAt
	if !req.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestRequirement_SetClaudeRuntimeResult(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	req.SetClaudeRuntimeResult("执行成功，创建了3个文件")

	if req.ClaudeRuntimeResult() != "执行成功，创建了3个文件" {
		t.Errorf("期望 ClaudeRuntimeResult 为 '执行成功，创建了3个文件', 实际为 %s", req.ClaudeRuntimeResult())
	}
}

func TestRequirement_SetClaudeRuntimePrompt(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	req.SetClaudeRuntimePrompt("请实现登录功能")

	if req.ClaudeRuntimePrompt() != "请实现登录功能" {
		t.Errorf("期望 ClaudeRuntimePrompt 为 '请实现登录功能', 实际为 %s", req.ClaudeRuntimePrompt())
	}
}

func TestRequirement_SetTraceID(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	oldUpdatedAt := req.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	req.SetTraceID("trace-001")

	if req.TraceID() != "trace-001" {
		t.Errorf("期望 TraceID 为 'trace-001', 实际为 %s", req.TraceID())
	}

	// 验证 UpdatedAt
	if !req.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestRequirement_SetRequirementType(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	req.SetRequirementType(RequirementTypeHeartbeat)

	if req.RequirementType() != RequirementTypeHeartbeat {
		t.Errorf("期望 RequirementType 为 heartbeat, 实际为 %s", req.RequirementType())
	}
}

func TestRequirement_SetDispatchSessionKey(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	oldUpdatedAt := req.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	req.SetDispatchSessionKey("  feishu:chat-001  ")

	// 验证空格被去除
	if req.DispatchSessionKey() != "feishu:chat-001" {
		t.Errorf("期望 DispatchSessionKey 为 'feishu:chat-001', 实际为 %s", req.DispatchSessionKey())
	}

	// 验证 UpdatedAt
	if !req.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestRequirement_SetReplicaAgentCode(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	oldUpdatedAt := req.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	req.SetReplicaAgentCode("replica-001")

	if req.ReplicaAgentCode() != "replica-001" {
		t.Errorf("期望 ReplicaAgentCode 为 'replica-001', 实际为 %s", req.ReplicaAgentCode())
	}

	// 验证 UpdatedAt
	if !req.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestRequirement_SetWorkspacePath(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	oldUpdatedAt := req.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	req.SetWorkspacePath("/workspace/test")

	if req.WorkspacePath() != "/workspace/test" {
		t.Errorf("期望 WorkspacePath 为 '/workspace/test', 实际为 %s", req.WorkspacePath())
	}

	// 验证 UpdatedAt
	if !req.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestNewRedispatchedRequirement_Success(t *testing.T) {
	original, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"原始需求",
		"原始描述",
		"原始验收标准",
		"/tmp/original",
	)

	redispatched, err := NewRedispatchedRequirement(
		NewRequirementID("req-002"),
		original,
	)

	if err != nil {
		t.Fatalf("创建重新派发需求失败: %v", err)
	}

	// 验证新ID
	if redispatched.ID().String() != "req-002" {
		t.Errorf("期望 ID 为 req-002, 实际为 %s", redispatched.ID().String())
	}

	// 验证标题前缀
	if redispatched.Title() != "[重新派发] 原始需求" {
		t.Errorf("期望 Title 为 '[重新派发] 原始需求', 实际为 %s", redispatched.Title())
	}

	// 验证 ProjectID 继承
	if redispatched.ProjectID().String() != "proj-001" {
		t.Errorf("期望 ProjectID 为 proj-001, 实际为 %s", redispatched.ProjectID().String())
	}

	// 验证描述继承
	if redispatched.Description() != "原始描述" {
		t.Errorf("期望 Description 为 '原始描述', 实际为 %s", redispatched.Description())
	}

	// 验证验收标准继承
	if redispatched.AcceptanceCriteria() != "原始验收标准" {
		t.Errorf("期望 AcceptanceCriteria 为 '原始验收标准', 实际为 %s", redispatched.AcceptanceCriteria())
	}

	// 验证工作目录继承
	if redispatched.TempWorkspaceRoot() != "/tmp/original" {
		t.Errorf("期望 TempWorkspaceRoot 为 '/tmp/original', 实际为 %s", redispatched.TempWorkspaceRoot())
	}

	// 验证状态为 todo
	if redispatched.Status() != RequirementStatusTodo {
		t.Errorf("期望 Status 为 todo, 实际为 %s", redispatched.Status())
	}
}

func TestNewRedispatchedRequirement_NilOriginal(t *testing.T) {
	_, err := NewRedispatchedRequirement(
		NewRequirementID("req-002"),
		nil,
	)

	if err != ErrRequirementProjectIDRequired {
		t.Errorf("期望返回 ErrRequirementProjectIDRequired, 实际返回 %v", err)
	}
}

func TestRequirement_ToSnapshot(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"/tmp/workspace",
	)

	// 设置一些额外字段
	req.SetRequirementType(RequirementTypeHeartbeat)
	req.SetDispatchSessionKey("session-001")
	req.SetTraceID("trace-001")
	req.SetTokenUsage(1000, 500, 1500)
	req.SetClaudeRuntimeResult("执行结果")
	req.SetClaudeRuntimePrompt("执行提示词")

	snap := req.ToSnapshot()

	// 验证快照字段
	if snap.ID.String() != "req-001" {
		t.Errorf("快照 ID 期望 req-001, 实际 %s", snap.ID.String())
	}

	if snap.ProjectID.String() != "proj-001" {
		t.Errorf("快照 ProjectID 期望 proj-001, 实际 %s", snap.ProjectID.String())
	}

	if snap.Title != "需求标题" {
		t.Errorf("快照 Title 期望 '需求标题', 实际 %s", snap.Title)
	}

	if snap.Description != "需求描述" {
		t.Errorf("快照 Description 期望 '需求描述', 实际 %s", snap.Description)
	}

	if snap.AcceptanceCriteria != "验收标准" {
		t.Errorf("快照 AcceptanceCriteria 期望 '验收标准', 实际 %s", snap.AcceptanceCriteria)
	}

	if snap.TempWorkspaceRoot != "/tmp/workspace" {
		t.Errorf("快照 TempWorkspaceRoot 期望 '/tmp/workspace', 实际 %s", snap.TempWorkspaceRoot)
	}

	if snap.Status != RequirementStatusTodo {
		t.Errorf("快照 Status 期望 todo, 实际 %s", snap.Status)
	}

	if snap.RequirementType != RequirementTypeHeartbeat {
		t.Errorf("快照 RequirementType 期望 heartbeat, 实际 %s", snap.RequirementType)
	}

	if snap.DispatchSessionKey != "session-001" {
		t.Errorf("快照 DispatchSessionKey 期望 'session-001', 实际 %s", snap.DispatchSessionKey)
	}

	if snap.TraceID != "trace-001" {
		t.Errorf("快照 TraceID 期望 'trace-001', 实际 %s", snap.TraceID)
	}

	if snap.PromptTokens != 1000 {
		t.Errorf("快照 PromptTokens 期望 1000, 实际 %d", snap.PromptTokens)
	}

	if snap.CompletionTokens != 500 {
		t.Errorf("快照 CompletionTokens 期望 500, 实际 %d", snap.CompletionTokens)
	}

	if snap.TotalTokens != 1500 {
		t.Errorf("快照 TotalTokens 期望 1500, 实际 %d", snap.TotalTokens)
	}

	if snap.ClaudeRuntimeResult != "执行结果" {
		t.Errorf("快照 ClaudeRuntimeResult 期望 '执行结果', 实际 %s", snap.ClaudeRuntimeResult)
	}

	if snap.ClaudeRuntimePrompt != "执行提示词" {
		t.Errorf("快照 ClaudeRuntimePrompt 期望 '执行提示词', 实际 %s", snap.ClaudeRuntimePrompt)
	}
}

func TestRequirement_FromSnapshot(t *testing.T) {
	startedAt := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	completedAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	claudeStartedAt := time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC)
	claudeEndedAt := time.Date(2024, 1, 1, 11, 30, 0, 0, time.UTC)
	createdAt := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	snap := RequirementSnapshot{
		ID:                     NewRequirementID("req-001"),
		ProjectID:              NewProjectID("proj-001"),
		Title:                  "快照标题",
		Description:            "快照描述",
		AcceptanceCriteria:     "快照验收标准",
		TempWorkspaceRoot:      "/tmp/snapshot",
		Status:                 RequirementStatusCoding,
		AssigneeAgentCode:      "agent-001",
		ReplicaAgentCode:       "replica-001",
		DispatchSessionKey:     "session-001",
		WorkspacePath:          "/workspace/test",
		LastError:              "",
		StartedAt:              &startedAt,
		CompletedAt:            &completedAt,
		CreatedAt:              createdAt,
		UpdatedAt:              updatedAt,
		RequirementType:        RequirementTypeHeartbeat,
		ClaudeRuntimeStatus:    RuntimeStatusCompleted,
		ClaudeRuntimeStartedAt: &claudeStartedAt,
		ClaudeRuntimeEndedAt:   &claudeEndedAt,
		ClaudeRuntimeError:     "",
		ClaudeRuntimeResult:    "执行完成",
		ClaudeRuntimePrompt:    "提示词",
		TraceID:                "trace-001",
		PromptTokens:           1000,
		CompletionTokens:       500,
		TotalTokens:            1500,
	}

	req := &Requirement{}
	err := req.FromSnapshot(snap)

	if err != nil {
		t.Fatalf("从快照恢复失败: %v", err)
	}

	// 验证所有字段
	if req.ID().String() != "req-001" {
		t.Errorf("ID 期望 req-001, 实际 %s", req.ID().String())
	}

	if req.ProjectID().String() != "proj-001" {
		t.Errorf("ProjectID 期望 proj-001, 实际 %s", req.ProjectID().String())
	}

	if req.Title() != "快照标题" {
		t.Errorf("Title 期望 '快照标题', 实际 %s", req.Title())
	}

	if req.Description() != "快照描述" {
		t.Errorf("Description 期望 '快照描述', 实际 %s", req.Description())
	}

	if req.AcceptanceCriteria() != "快照验收标准" {
		t.Errorf("AcceptanceCriteria 期望 '快照验收标准', 实际 %s", req.AcceptanceCriteria())
	}

	if req.TempWorkspaceRoot() != "/tmp/snapshot" {
		t.Errorf("TempWorkspaceRoot 期望 '/tmp/snapshot', 实际 %s", req.TempWorkspaceRoot())
	}

	if req.Status() != RequirementStatusCoding {
		t.Errorf("Status 期望 coding, 实际 %s", req.Status())
	}

	if req.AssigneeAgentCode() != "agent-001" {
		t.Errorf("AssigneeAgentCode 期望 'agent-001', 实际 %s", req.AssigneeAgentCode())
	}

	if req.ReplicaAgentCode() != "replica-001" {
		t.Errorf("ReplicaAgentCode 期望 'replica-001', 实际 %s", req.ReplicaAgentCode())
	}

	if req.DispatchSessionKey() != "session-001" {
		t.Errorf("DispatchSessionKey 期望 'session-001', 实际 %s", req.DispatchSessionKey())
	}

	if req.WorkspacePath() != "/workspace/test" {
		t.Errorf("WorkspacePath 期望 '/workspace/test', 实际 %s", req.WorkspacePath())
	}

	if req.RequirementType() != RequirementTypeHeartbeat {
		t.Errorf("RequirementType 期望 heartbeat, 实际 %s", req.RequirementType())
	}

	if req.ClaudeRuntimeStatus() != RuntimeStatusCompleted {
		t.Errorf("ClaudeRuntimeStatus 期望 'completed', 实际 %s", req.ClaudeRuntimeStatus())
	}

	if req.ClaudeRuntimeResult() != "执行完成" {
		t.Errorf("ClaudeRuntimeResult 期望 '执行完成', 实际 %s", req.ClaudeRuntimeResult())
	}

	if req.ClaudeRuntimePrompt() != "提示词" {
		t.Errorf("ClaudeRuntimePrompt 期望 '提示词', 实际 %s", req.ClaudeRuntimePrompt())
	}

	if req.TraceID() != "trace-001" {
		t.Errorf("TraceID 期望 'trace-001', 实际 %s", req.TraceID())
	}

	if req.PromptTokens() != 1000 {
		t.Errorf("PromptTokens 期望 1000, 实际 %d", req.PromptTokens())
	}

	if req.CompletionTokens() != 500 {
		t.Errorf("CompletionTokens 期望 500, 实际 %d", req.CompletionTokens())
	}

	if req.TotalTokens() != 1500 {
		t.Errorf("TotalTokens 期望 1500, 实际 %d", req.TotalTokens())
	}

	// 验证时间字段
	if req.StartedAt() == nil || !req.StartedAt().Equal(startedAt) {
		t.Error("StartedAt 不匹配")
	}

	if req.CompletedAt() == nil || !req.CompletedAt().Equal(completedAt) {
		t.Error("CompletedAt 不匹配")
	}

	if req.ClaudeRuntimeStartedAt() == nil || !req.ClaudeRuntimeStartedAt().Equal(claudeStartedAt) {
		t.Error("ClaudeRuntimeStartedAt 不匹配")
	}

	if req.ClaudeRuntimeEndedAt() == nil || !req.ClaudeRuntimeEndedAt().Equal(claudeEndedAt) {
		t.Error("ClaudeRuntimeEndedAt 不匹配")
	}

	if !req.CreatedAt().Equal(createdAt) {
		t.Error("CreatedAt 不匹配")
	}

	if !req.UpdatedAt().Equal(updatedAt) {
		t.Error("UpdatedAt 不匹配")
	}
}

func TestRequirement_FromSnapshot_AnyStatusAccepted(t *testing.T) {
	// 状态现在由状态机定义，FromSnapshot 接受任何非空状态
	snap := RequirementSnapshot{
		ID:          NewRequirementID("req-001"),
		ProjectID:   NewProjectID("proj-001"),
		Title:       "快照标题",
		Description: "快照描述",
		Status:      "custom_state_from_sm",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := &Requirement{}
	err := req.FromSnapshot(snap)

	if err != nil {
		t.Errorf("期望不返回错误, 实际返回 %v", err)
	}
	if req.Status() != "custom_state_from_sm" {
		t.Errorf("期望状态 custom_state_from_sm, 实际 %s", req.Status())
	}
}

func TestRequirement_FromSnapshot_NormalizeOldStatus(t *testing.T) {
	snap := RequirementSnapshot{
		ID:          NewRequirementID("req-001"),
		ProjectID:   NewProjectID("proj-001"),
		Title:       "快照标题",
		Description: "快照描述",
		Status:      "in_progress", // 旧状态值
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	req := &Requirement{}
	err := req.FromSnapshot(snap)

	if err != nil {
		t.Fatalf("从快照恢复失败: %v", err)
	}

	// 验证旧状态被转换为新状态
	if req.Status() != RequirementStatusPreparing {
		t.Errorf("旧状态 in_progress 应转换为 preparing, 实际为 %s", req.Status())
	}
}

func TestRequirement_SnapshotRoundTrip(t *testing.T) {
	startedAt := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	completedAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	claudeStartedAt := time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC)
	claudeEndedAt := time.Date(2024, 1, 1, 11, 30, 0, 0, time.UTC)

	original := &Requirement{}

	// 使用快照设置所有字段
	snap := RequirementSnapshot{
		ID:                     NewRequirementID("req-001"),
		ProjectID:              NewProjectID("proj-001"),
		Title:                  "原始标题",
		Description:            "原始描述",
		AcceptanceCriteria:     "原始验收标准",
		TempWorkspaceRoot:      "/tmp/original",
		Status:                 RequirementStatusCoding,
		AssigneeAgentCode:      "agent-001",
		ReplicaAgentCode:       "replica-001",
		DispatchSessionKey:     "session-001",
		WorkspacePath:          "/workspace/test",
		LastError:              "",
		StartedAt:              &startedAt,
		CompletedAt:            &completedAt,
		CreatedAt:              time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		UpdatedAt:              time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		RequirementType:        RequirementTypeNormal,
		ClaudeRuntimeStatus:    RuntimeStatusCompleted,
		ClaudeRuntimeStartedAt: &claudeStartedAt,
		ClaudeRuntimeEndedAt:   &claudeEndedAt,
		ClaudeRuntimeError:     "",
		ClaudeRuntimeResult:    "执行完成",
		ClaudeRuntimePrompt:    "提示词",
		TraceID:                "trace-001",
		PromptTokens:           1000,
		CompletionTokens:       500,
		TotalTokens:            1500,
	}

	original.FromSnapshot(snap)

	// 转换为快照
	restoredSnap := original.ToSnapshot()

	// 从快照恢复
	restored := &Requirement{}
	err := restored.FromSnapshot(restoredSnap)

	if err != nil {
		t.Fatalf("从快照恢复失败: %v", err)
	}

	// 验证所有字段一致
	if !reflect.DeepEqual(restoredSnap, snap) {
		t.Error("快照往返后字段不一致")
	}

	// 验证恢复的 Requirement 字段
	if restored.ID().String() != original.ID().String() {
		t.Error("ID 不匹配")
	}

	if restored.Title() != original.Title() {
		t.Error("Title 不匹配")
	}

	if restored.Status() != original.Status() {
		t.Error("Status 不匹配")
	}

	if restored.PromptTokens() != original.PromptTokens() {
		t.Error("PromptTokens 不匹配")
	}
}

func TestRequirement_ClaudeRuntimeTimeCopy(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	req.StartClaudeRuntime()

	// 获取时间点
	startedAt1 := req.ClaudeRuntimeStartedAt()
	startedAt2 := req.ClaudeRuntimeStartedAt()

	// 验证返回的是不同的指针
	if startedAt1 == startedAt2 {
		t.Error("ClaudeRuntimeStartedAt 应返回不同的指针")
	}

	// 验证值相同
	if !startedAt1.Equal(*startedAt2) {
		t.Error("ClaudeRuntimeStartedAt 返回的时间值应相同")
	}
}

func TestRequirement_TimeCopy(t *testing.T) {
	req := createRequirementWithStatus(RequirementStatusPreparing)

	startedAt1 := req.StartedAt()
	startedAt2 := req.StartedAt()

	// 验证返回的是不同的指针
	if startedAt1 == startedAt2 {
		t.Error("StartedAt 应返回不同的指针")
	}

	// 验证值相同
	if !startedAt1.Equal(*startedAt2) {
		t.Error("StartedAt 返回的时间值应相同")
	}

	completedAt1 := req.CompletedAt()
	completedAt2 := req.CompletedAt()

	// 验证 CompletedAt 也返回不同的指针
	if completedAt1 != completedAt2 {
		t.Error("CompletedAt nil 情况下应返回相同的 nil")
	}
}

func TestNewRequirement_Success_WithEmptyWorkspace(t *testing.T) {
	req, err := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"", // 空工作目录
	)

	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	if req.TempWorkspaceRoot() != "" {
		t.Errorf("期望 TempWorkspaceRoot 为空, 实际为 %s", req.TempWorkspaceRoot())
	}
}

func TestNewRequirement_Success_WithWhitespaceWorkspace(t *testing.T) {
	req, err := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"  /tmp/workspace  ", // 带空格的工作目录
	)

	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	// 空格应被去除
	if req.TempWorkspaceRoot() != "/tmp/workspace" {
		t.Errorf("期望 TempWorkspaceRoot 去除空格为 '/tmp/workspace', 实际为 '%s'", req.TempWorkspaceRoot())
	}
}

func TestRequirement_UpdateContent_WithWhitespaceWorkspace(t *testing.T) {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"原标题",
		"原描述",
		"原验收标准",
		"/tmp/old",
	)

	err := req.UpdateContent("新标题", "新描述", "新验收标准", "  /tmp/new  ")

	if err != nil {
		t.Fatalf("UpdateContent 失败: %v", err)
	}

	// 空格应被去除
	if req.TempWorkspaceRoot() != "/tmp/new" {
		t.Errorf("期望 TempWorkspaceRoot 去除空格为 '/tmp/new', 实际为 '%s'", req.TempWorkspaceRoot())
	}
}

func TestRequirement_FromSnapshot_WithWhitespaceWorkspace(t *testing.T) {
	snap := RequirementSnapshot{
		ID:                 NewRequirementID("req-001"),
		ProjectID:          NewProjectID("proj-001"),
		Title:              "快照标题",
		Description:        "快照描述",
		AcceptanceCriteria: "快照验收标准",
		TempWorkspaceRoot:  "  /tmp/workspace  ", // 带空格
		DispatchSessionKey: "  session-001  ",    // 带空格
		Status:             RequirementStatusTodo,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	req := &Requirement{}
	err := req.FromSnapshot(snap)

	if err != nil {
		t.Fatalf("从快照恢复失败: %v", err)
	}

	// 空格应被去除
	if req.TempWorkspaceRoot() != "/tmp/workspace" {
		t.Errorf("期望 TempWorkspaceRoot 去除空格, 实际为 '%s'", req.TempWorkspaceRoot())
	}

	if req.DispatchSessionKey() != "session-001" {
		t.Errorf("期望 DispatchSessionKey 去除空格, 实际为 '%s'", req.DispatchSessionKey())
	}
}

// 辅助函数：创建指定状态的需求
func createRequirementWithStatus(status RequirementStatus) *Requirement {
	req, _ := NewRequirement(
		NewRequirementID("req-001"),
		NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)

	switch status {
	case RequirementStatusTodo:
		// 默认状态
	case RequirementStatusPreparing:
		req.StartDispatch("agent-001")
	case RequirementStatusCoding:
		req.StartDispatch("agent-001")
		req.MarkCoding("/workspace", "replica-001")
	case RequirementStatusPROpened:
		req.StartDispatch("agent-001")
		req.MarkCoding("/workspace", "replica-001")
		req.MarkPROpened()
	case RequirementStatusFailed:
		req.StartDispatch("agent-001")
		req.MarkCoding("/workspace", "replica-001")
		req.MarkFailed("执行失败")
	case RequirementStatusCompleted:
		req.StartDispatch("agent-001")
		req.MarkCoding("/workspace", "replica-001")
		req.MarkCompleted()
	case RequirementStatus("done"):
		// Done 状态需要手动设置或通过其他流程
		req.StartDispatch("agent-001")
		req.MarkCoding("/workspace", "replica-001")
		req.MarkCompleted() // 使用 Completed 代替
	}

	return req
}
