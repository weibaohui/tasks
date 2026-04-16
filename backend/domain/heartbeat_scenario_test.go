package domain

import (
	"testing"
	"time"
)

type fakeIDGenerator struct {
	idx int
}

func (g *fakeIDGenerator) Generate() string {
	g.idx++
	return "id-" + string(rune('0'+g.idx))
}

func TestNewHeartbeatScenario(t *testing.T) {
	tests := []struct {
		name        string
		id          HeartbeatScenarioID
		code        string
		scenarioName string
		description string
		items       []HeartbeatScenarioItem
		wantErr     error
	}{
		{
			name:         "正常创建",
			id:           NewHeartbeatScenarioID("sc-001"),
			code:         "github_dev",
			scenarioName: "GitHub开发",
			description:  "测试场景",
			items: []HeartbeatScenarioItem{
				{Name: "心跳1", IntervalMinutes: 30, MDContent: "content", AgentCode: "agent1", RequirementType: "normal"},
			},
			wantErr: nil,
		},
		{
			name:         "空ID",
			id:           NewHeartbeatScenarioID(""),
			code:         "github_dev",
			scenarioName: "GitHub开发",
			description:  "测试场景",
			items:        []HeartbeatScenarioItem{},
			wantErr:      ErrHeartbeatScenarioIDRequired,
		},
		{
			name:         "空code",
			id:           NewHeartbeatScenarioID("sc-001"),
			code:         "",
			scenarioName: "GitHub开发",
			description:  "测试场景",
			items:        []HeartbeatScenarioItem{},
			wantErr:      ErrHeartbeatScenarioCodeRequired,
		},
		{
			name:         "空格code",
			id:           NewHeartbeatScenarioID("sc-001"),
			code:         "   ",
			scenarioName: "GitHub开发",
			description:  "测试场景",
			items:        []HeartbeatScenarioItem{},
			wantErr:      ErrHeartbeatScenarioCodeRequired,
		},
		{
			name:         "空name",
			id:           NewHeartbeatScenarioID("sc-001"),
			code:         "github_dev",
			scenarioName: "",
			description:  "测试场景",
			items:        []HeartbeatScenarioItem{},
			wantErr:      ErrHeartbeatScenarioNameRequired,
		},
		{
			name:         "items为空允许",
			id:           NewHeartbeatScenarioID("sc-001"),
			code:         "github_dev",
			scenarioName: "GitHub开发",
			description:  "测试场景",
			items:        []HeartbeatScenarioItem{},
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario, err := NewHeartbeatScenario(tt.id, tt.code, tt.scenarioName, tt.description, tt.items)
			if err != tt.wantErr {
				t.Fatalf("期望错误 %v, 实际 %v", tt.wantErr, err)
			}
			if err != nil {
				return
			}
			if scenario.ID().String() != tt.id.String() {
				t.Errorf("期望ID %s, 实际 %s", tt.id.String(), scenario.ID().String())
			}
			if scenario.Code() != tt.code {
				t.Errorf("期望Code %s, 实际 %s", tt.code, scenario.Code())
			}
			if scenario.Name() != tt.scenarioName {
				t.Errorf("期望Name %s, 实际 %s", tt.scenarioName, scenario.Name())
			}
			if !scenario.Enabled() {
				t.Error("期望默认启用")
			}
			if scenario.IsBuiltIn() {
				t.Error("期望默认非内置")
			}
			if scenario.CreatedAt().IsZero() {
				t.Error("CreatedAt 不应为零")
			}
		})
	}
}

func TestHeartbeatScenarioApplyToProject(t *testing.T) {
	scenario, err := NewHeartbeatScenario(
		NewHeartbeatScenarioID("sc-001"),
		"github_dev",
		"GitHub开发",
		"描述",
		[]HeartbeatScenarioItem{
			{Name: "Issue分析", IntervalMinutes: 30, MDContent: "md1", AgentCode: "agent1", RequirementType: "github_issue", SortOrder: 1},
			{Name: "代码编写", IntervalMinutes: 60, MDContent: "md2", AgentCode: "agent2", RequirementType: "github_coding", SortOrder: 2},
		},
	)
	if err != nil {
		t.Fatalf("创建场景失败: %v", err)
	}

	project, _ := NewProject(NewProjectID("proj-001"), "测试项目", "https://github.com/test/repo.git", "main", []string{})
	idGen := &fakeIDGenerator{}

	heartbeats, err := scenario.ApplyToProject(project.ID(), idGen)
	if err != nil {
		t.Fatalf("应用失败: %v", err)
	}
	if len(heartbeats) != 2 {
		t.Fatalf("期望2条心跳, 实际 %d", len(heartbeats))
	}

	for i, hb := range heartbeats {
		if hb.ProjectID().String() != "proj-001" {
			t.Errorf("心跳%d projectID错误", i)
		}
		expectedName := "GitHub开发 - " + scenario.Items()[i].Name
		if hb.Name() != expectedName {
			t.Errorf("心跳%d name期望 %s, 实际 %s", i, expectedName, hb.Name())
		}
		if hb.IntervalMinutes() != scenario.Items()[i].IntervalMinutes {
			t.Errorf("心跳%d interval错误", i)
		}
		if hb.MDContent() != scenario.Items()[i].MDContent {
			t.Errorf("心跳%d mdContent错误", i)
		}
		if hb.AgentCode() != scenario.Items()[i].AgentCode {
			t.Errorf("心跳%d agentCode错误", i)
		}
		if hb.RequirementType() != scenario.Items()[i].RequirementType {
			t.Errorf("心跳%d requirementType错误", i)
		}
		if hb.SortOrder() != scenario.Items()[i].SortOrder {
			t.Errorf("心跳%d sortOrder错误", i)
		}
	}
}

func TestHeartbeatScenarioApplyToProjectEmptyProjectID(t *testing.T) {
	scenario, _ := NewHeartbeatScenario(
		NewHeartbeatScenarioID("sc-001"),
		"github_dev",
		"GitHub开发",
		"描述",
		[]HeartbeatScenarioItem{{Name: "心跳1", IntervalMinutes: 30, MDContent: "md", AgentCode: "agent", RequirementType: "normal"}},
	)
	_, err := scenario.ApplyToProject(NewProjectID(""), &fakeIDGenerator{})
	if err != ErrHeartbeatProjectIDRequired {
		t.Errorf("期望 ErrHeartbeatProjectIDRequired, 实际 %v", err)
	}
}

func TestHeartbeatScenarioUpdate(t *testing.T) {
	scenario, _ := NewHeartbeatScenario(
		NewHeartbeatScenarioID("sc-001"),
		"github_dev",
		"原名称",
		"原描述",
		[]HeartbeatScenarioItem{{Name: "心跳1", IntervalMinutes: 30, MDContent: "md", AgentCode: "agent", RequirementType: "normal"}},
	)
	oldUpdatedAt := scenario.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	err := scenario.Update("新名称", "新描述", []HeartbeatScenarioItem{
		{Name: "心跳2", IntervalMinutes: 60, MDContent: "new md", AgentCode: "newAgent", RequirementType: "pr_review"},
	})
	if err != nil {
		t.Fatalf("更新失败: %v", err)
	}

	if scenario.Name() != "新名称" {
		t.Errorf("期望名称 新名称, 实际 %s", scenario.Name())
	}
	if scenario.Description() != "新描述" {
		t.Errorf("期望描述 新描述, 实际 %s", scenario.Description())
	}
	if len(scenario.Items()) != 1 || scenario.Items()[0].Name != "心跳2" {
		t.Errorf("期望items更新")
	}
	if !scenario.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestHeartbeatScenarioUpdateValidation(t *testing.T) {
	scenario, _ := NewHeartbeatScenario(
		NewHeartbeatScenarioID("sc-001"),
		"github_dev",
		"原名称",
		"原描述",
		[]HeartbeatScenarioItem{{Name: "心跳1", IntervalMinutes: 30, MDContent: "md", AgentCode: "agent", RequirementType: "normal"}},
	)

	err := scenario.Update("", "新描述", []HeartbeatScenarioItem{})
	if err != ErrHeartbeatScenarioNameRequired {
		t.Errorf("期望 ErrHeartbeatScenarioNameRequired, 实际 %v", err)
	}
	if scenario.Name() != "原名称" {
		t.Errorf("失败时不应修改名称")
	}
}

func TestHeartbeatScenarioSetEnabledAndBuiltIn(t *testing.T) {
	scenario, _ := NewHeartbeatScenario(
		NewHeartbeatScenarioID("sc-001"),
		"github_dev",
		"名称",
		"描述",
		[]HeartbeatScenarioItem{},
	)

	scenario.SetEnabled(false)
	if scenario.Enabled() {
		t.Error("期望禁用")
	}

	scenario.SetIsBuiltIn(true)
	if !scenario.IsBuiltIn() {
		t.Error("期望内置")
	}
}

func TestHeartbeatScenarioSnapshotRoundTrip(t *testing.T) {
	scenario, _ := NewHeartbeatScenario(
		NewHeartbeatScenarioID("sc-001"),
		"github_dev",
		"GitHub开发",
		"描述",
		[]HeartbeatScenarioItem{
			{Name: "心跳1", IntervalMinutes: 30, MDContent: "md", AgentCode: "agent", RequirementType: "normal", SortOrder: 1},
		},
	)
	scenario.SetEnabled(false)
	scenario.SetIsBuiltIn(true)

	snap := scenario.ToSnapshot()
	if snap.ID.String() != "sc-001" {
		t.Errorf("快照ID错误")
	}
	if snap.Code != "github_dev" {
		t.Errorf("快照Code错误")
	}
	if snap.Enabled {
		t.Error("快照Enabled应为false")
	}
	if !snap.IsBuiltIn {
		t.Error("快照IsBuiltIn应为true")
	}
	if len(snap.Items) != 1 {
		t.Errorf("快照Items长度错误")
	}

	restored := &HeartbeatScenario{}
	if err := restored.FromSnapshot(snap); err != nil {
		t.Fatalf("恢复快照失败: %v", err)
	}
	if restored.ID().String() != "sc-001" {
		t.Errorf("恢复后ID错误")
	}
	if restored.Code() != "github_dev" {
		t.Errorf("恢复后Code错误")
	}
	if restored.Enabled() {
		t.Error("恢复后Enabled应为false")
	}
	if !restored.IsBuiltIn() {
		t.Error("恢复后IsBuiltIn应为true")
	}
	if len(restored.Items()) != 1 || restored.Items()[0].Name != "心跳1" {
		t.Errorf("恢复后Items错误")
	}
}

func TestHeartbeatScenarioFromSnapshotValidation(t *testing.T) {
	tests := []struct {
		name    string
		snap    HeartbeatScenarioSnapshot
		wantErr error
	}{
		{
			name:    "空ID",
			snap:    HeartbeatScenarioSnapshot{ID: NewHeartbeatScenarioID(""), Code: "c", Name: "n"},
			wantErr: ErrHeartbeatScenarioIDRequired,
		},
		{
			name:    "空Code",
			snap:    HeartbeatScenarioSnapshot{ID: NewHeartbeatScenarioID("id"), Code: "", Name: "n"},
			wantErr: ErrHeartbeatScenarioCodeRequired,
		},
		{
			name:    "空Name",
			snap:    HeartbeatScenarioSnapshot{ID: NewHeartbeatScenarioID("id"), Code: "c", Name: ""},
			wantErr: ErrHeartbeatScenarioNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &HeartbeatScenario{}
			err := s.FromSnapshot(tt.snap)
			if err != tt.wantErr {
				t.Errorf("期望错误 %v, 实际 %v", tt.wantErr, err)
			}
		})
	}
}
