package domain

import (
	"strings"
	"testing"
	"time"
)

func TestNewHeartbeat(t *testing.T) {
	tests := []struct {
		name            string
		id              HeartbeatID
		projectID       ProjectID
		heartbeatName   string
		intervalMinutes int
		mdContent       string
		agentCode       string
		requirementType string
		wantErr         error
	}{
		{
			name:            "正常创建",
			id:              NewHeartbeatID("hb-001"),
			projectID:       NewProjectID("proj-001"),
			heartbeatName:   "需求派发",
			intervalMinutes: 30,
			mdContent:       "# Prompt",
			agentCode:       "scheduler",
			requirementType: "normal",
			wantErr:         nil,
		},
		{
			name:            "空ID",
			id:              NewHeartbeatID(""),
			projectID:       NewProjectID("proj-001"),
			heartbeatName:   "需求派发",
			intervalMinutes: 30,
			mdContent:       "# Prompt",
			agentCode:       "scheduler",
			requirementType: "normal",
			wantErr:         ErrHeartbeatIDRequired,
		},
		{
			name:            "空项目ID",
			id:              NewHeartbeatID("hb-001"),
			projectID:       NewProjectID(""),
			heartbeatName:   "需求派发",
			intervalMinutes: 30,
			mdContent:       "# Prompt",
			agentCode:       "scheduler",
			requirementType: "normal",
			wantErr:         ErrHeartbeatProjectIDRequired,
		},
		{
			name:            "空名称",
			id:              NewHeartbeatID("hb-001"),
			projectID:       NewProjectID("proj-001"),
			heartbeatName:   "",
			intervalMinutes: 30,
			mdContent:       "# Prompt",
			agentCode:       "scheduler",
			requirementType: "normal",
			wantErr:         ErrHeartbeatNameRequired,
		},
		{
			name:            "只有空格名称",
			id:              NewHeartbeatID("hb-001"),
			projectID:       NewProjectID("proj-001"),
			heartbeatName:   "   ",
			intervalMinutes: 30,
			mdContent:       "# Prompt",
			agentCode:       "scheduler",
			requirementType: "normal",
			wantErr:         ErrHeartbeatNameRequired,
		},
		{
			name:            "非法间隔",
			id:              NewHeartbeatID("hb-001"),
			projectID:       NewProjectID("proj-001"),
			heartbeatName:   "需求派发",
			intervalMinutes: 0,
			mdContent:       "# Prompt",
			agentCode:       "scheduler",
			requirementType: "normal",
			wantErr:         ErrHeartbeatIntervalInvalid,
		},
		{
			name:            "空AgentCode",
			id:              NewHeartbeatID("hb-001"),
			projectID:       NewProjectID("proj-001"),
			heartbeatName:   "需求派发",
			intervalMinutes: 30,
			mdContent:       "# Prompt",
			agentCode:       "",
			requirementType: "normal",
			wantErr:         ErrHeartbeatAgentCodeRequired,
		},
		{
			name:            "只有空格AgentCode",
			id:              NewHeartbeatID("hb-001"),
			projectID:       NewProjectID("proj-001"),
			heartbeatName:   "需求派发",
			intervalMinutes: 30,
			mdContent:       "# Prompt",
			agentCode:       "   ",
			requirementType: "normal",
			wantErr:         ErrHeartbeatAgentCodeRequired,
		},
		{
			name:            "空requirementType使用默认值",
			id:              NewHeartbeatID("hb-001"),
			projectID:       NewProjectID("proj-001"),
			heartbeatName:   "需求派发",
			intervalMinutes: 30,
			mdContent:       "# Prompt",
			agentCode:       "scheduler",
			requirementType: "",
			wantErr:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hb, err := NewHeartbeat(tt.id, tt.projectID, tt.heartbeatName, tt.intervalMinutes, tt.mdContent, tt.agentCode, tt.requirementType)
			if err != tt.wantErr {
				t.Fatalf("期望错误 %v, 实际 %v", tt.wantErr, err)
			}
			if err != nil {
				return
			}
			if hb.ID().String() != tt.id.String() {
				t.Errorf("期望ID %s, 实际 %s", tt.id.String(), hb.ID().String())
			}
			if hb.ProjectID().String() != tt.projectID.String() {
				t.Errorf("期望ProjectID %s, 实际 %s", tt.projectID.String(), hb.ProjectID().String())
			}
			if hb.Name() != strings.TrimSpace(tt.heartbeatName) {
				t.Errorf("期望名称 %s, 实际 %s", tt.heartbeatName, hb.Name())
			}
			if hb.IntervalMinutes() != tt.intervalMinutes {
				t.Errorf("期望间隔 %d, 实际 %d", tt.intervalMinutes, hb.IntervalMinutes())
			}
			if hb.MDContent() != tt.mdContent {
				t.Errorf("期望MD内容 %s, 实际 %s", tt.mdContent, hb.MDContent())
			}
			if hb.AgentCode() != strings.TrimSpace(tt.agentCode) {
				t.Errorf("期望AgentCode %s, 实际 %s", tt.agentCode, hb.AgentCode())
			}
			expectedType := tt.requirementType
			if expectedType == "" {
				expectedType = "heartbeat"
			}
			if hb.RequirementType() != expectedType {
				t.Errorf("期望RequirementType %s, 实际 %s", expectedType, hb.RequirementType())
			}
			if !hb.Enabled() {
				t.Error("期望默认启用")
			}
			if hb.CreatedAt().IsZero() {
				t.Error("CreatedAt 不应为零")
			}
			if hb.UpdatedAt().IsZero() {
				t.Error("UpdatedAt 不应为零")
			}
		})
	}
}

func TestHeartbeatUpdate(t *testing.T) {
	hb, err := NewHeartbeat(
		NewHeartbeatID("hb-001"),
		NewProjectID("proj-001"),
		"原名称",
		30,
		"# Old",
		"old-agent",
		"normal",
	)
	if err != nil {
		t.Fatalf("创建Heartbeat失败: %v", err)
	}

	oldUpdatedAt := hb.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	err = hb.Update("新名称", 60, "# New", "new-agent", "pr_review")
	if err != nil {
		t.Fatalf("更新失败: %v", err)
	}

	if hb.Name() != "新名称" {
		t.Errorf("期望名称 新名称, 实际 %s", hb.Name())
	}
	if hb.IntervalMinutes() != 60 {
		t.Errorf("期望间隔 60, 实际 %d", hb.IntervalMinutes())
	}
	if hb.MDContent() != "# New" {
		t.Errorf("期望MD内容 # New, 实际 %s", hb.MDContent())
	}
	if hb.AgentCode() != "new-agent" {
		t.Errorf("期望AgentCode new-agent, 实际 %s", hb.AgentCode())
	}
	if hb.RequirementType() != "pr_review" {
		t.Errorf("期望RequirementType pr_review, 实际 %s", hb.RequirementType())
	}
	if !hb.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestHeartbeatUpdateValidation(t *testing.T) {
	hb, err := NewHeartbeat(
		NewHeartbeatID("hb-001"),
		NewProjectID("proj-001"),
		"原名称",
		30,
		"# Old",
		"old-agent",
		"normal",
	)
	if err != nil {
		t.Fatalf("创建Heartbeat失败: %v", err)
	}

	tests := []struct {
		name            string
		heartbeatName   string
		intervalMinutes int
		agentCode       string
		wantErr         error
	}{
		{
			name:            "空名称",
			heartbeatName:   "",
			intervalMinutes: 30,
			agentCode:       "agent",
			wantErr:         ErrHeartbeatNameRequired,
		},
		{
			name:            "非法间隔",
			heartbeatName:   "名称",
			intervalMinutes: 0,
			agentCode:       "agent",
			wantErr:         ErrHeartbeatIntervalInvalid,
		},
		{
			name:            "空AgentCode",
			heartbeatName:   "名称",
			intervalMinutes: 30,
			agentCode:       "",
			wantErr:         ErrHeartbeatAgentCodeRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hb.Update(tt.heartbeatName, tt.intervalMinutes, "# Content", tt.agentCode, "normal")
			if err != tt.wantErr {
				t.Errorf("期望错误 %v, 实际 %v", tt.wantErr, err)
			}
			// 验证失败后字段未被修改
			if hb.Name() != "原名称" {
				t.Errorf("失败时不应修改名称")
			}
		})
	}
}

func TestHeartbeatSetEnabled(t *testing.T) {
	hb, _ := NewHeartbeat(NewHeartbeatID("hb-001"), NewProjectID("proj-001"), "测试", 30, "", "agent", "heartbeat")
	oldUpdatedAt := hb.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	hb.SetEnabled(false)
	if hb.Enabled() {
		t.Error("期望禁用")
	}
	if !hb.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}

	oldUpdatedAt = hb.UpdatedAt()
	time.Sleep(10 * time.Millisecond)
	hb.SetEnabled(true)
	if !hb.Enabled() {
		t.Error("期望启用")
	}
	if !hb.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestHeartbeatSetSortOrder(t *testing.T) {
	hb, _ := NewHeartbeat(NewHeartbeatID("hb-001"), NewProjectID("proj-001"), "测试", 30, "", "agent", "heartbeat")
	hb.SetSortOrder(5)
	if hb.SortOrder() != 5 {
		t.Errorf("期望SortOrder 5, 实际 %d", hb.SortOrder())
	}
}

func TestHeartbeatRenderPrompt(t *testing.T) {
	project, _ := NewProject(NewProjectID("proj-001"), "测试项目", "https://github.com/test/project.git", "main", []string{"step1"})
	hb, _ := NewHeartbeat(
		NewHeartbeatID("hb-001"),
		project.ID(),
		"测试",
		30,
		"ID=${project.id} Name=${project.name} Repo=${project.git_repo_url} Branch=${project.default_branch} Time=${timestamp}",
		"agent",
		"heartbeat",
	)

	result := hb.RenderPrompt(project)
	if !strings.Contains(result, "ID=proj-001") {
		t.Errorf("期望包含 ID=proj-001, 实际 %s", result)
	}
	if !strings.Contains(result, "Name=测试项目") {
		t.Errorf("期望包含 Name=测试项目, 实际 %s", result)
	}
	if !strings.Contains(result, "Repo=https://github.com/test/project.git") {
		t.Errorf("期望包含 Repo, 实际 %s", result)
	}
	if !strings.Contains(result, "Branch=main") {
		t.Errorf("期望包含 Branch=main, 实际 %s", result)
	}
	if !strings.Contains(result, "Time=") {
		t.Errorf("期望包含 Time=, 实际 %s", result)
	}
}

func TestHeartbeatRenderPromptNilProject(t *testing.T) {
	hb, _ := NewHeartbeat(
		NewHeartbeatID("hb-001"),
		NewProjectID("proj-001"),
		"测试",
		30,
		"raw content",
		"agent",
		"heartbeat",
	)

	result := hb.RenderPrompt(nil)
	if result != "raw content" {
		t.Errorf("nil project 应返回原始内容, 实际 %s", result)
	}
}

func TestHeartbeatSnapshotRoundTrip(t *testing.T) {
	hb, _ := NewHeartbeat(
		NewHeartbeatID("hb-001"),
		NewProjectID("proj-001"),
		"快照测试",
		45,
		"# Snapshot",
		"snap-agent",
		"pr_review",
	)
	hb.SetEnabled(false)
	hb.SetSortOrder(3)

	snap := hb.ToSnapshot()
	if snap.ID.String() != "hb-001" {
		t.Errorf("快照ID错误")
	}
	if snap.Name != "快照测试" {
		t.Errorf("快照名称错误")
	}
	if snap.IntervalMinutes != 45 {
		t.Errorf("快照间隔错误")
	}
	if snap.Enabled {
		t.Error("快照Enabled应为false")
	}
	if snap.SortOrder != 3 {
		t.Errorf("快照排序错误")
	}

	restored := &Heartbeat{}
	restored.FromSnapshot(snap)
	if restored.ID().String() != "hb-001" {
		t.Errorf("恢复后ID错误")
	}
	if restored.Name() != "快照测试" {
		t.Errorf("恢复后名称错误")
	}
	if restored.IntervalMinutes() != 45 {
		t.Errorf("恢复后间隔错误")
	}
	if restored.Enabled() {
		t.Error("恢复后Enabled应为false")
	}
	if restored.SortOrder() != 3 {
		t.Errorf("恢复后排序错误")
	}
	if restored.RequirementType() != "pr_review" {
		t.Errorf("恢复后RequirementType错误")
	}
}
