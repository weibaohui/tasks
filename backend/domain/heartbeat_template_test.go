package domain

import (
	"strings"
	"testing"
)

func TestNewHeartbeatTemplate(t *testing.T) {
	tests := []struct {
		name            string
		templateID      string
		templateName    string
		mdContent       string
		requirementType string
		wantErr         bool
		errMsg          string
	}{
		{
			name: "正常创建", templateID: "ht-001", templateName: "测试模板",
			mdContent: "内容", requirementType: "normal", wantErr: false,
		},
		{
			name: "空ID", templateID: "", templateName: "测试模板",
			mdContent: "内容", requirementType: "normal", wantErr: true,
			errMsg: ErrHeartbeatTemplateIDRequired.Error(),
		},
		{
			name: "空名称", templateID: "ht-001", templateName: "",
			mdContent: "内容", requirementType: "normal", wantErr: true,
			errMsg: ErrHeartbeatTemplateNameRequired.Error(),
		},
		{
			name: "只有空格名称", templateID: "ht-001", templateName: "   ",
			mdContent: "内容", requirementType: "normal", wantErr: true,
			errMsg: ErrHeartbeatTemplateNameRequired.Error(),
		},
		{
			name: "空requirementType使用默认值", templateID: "ht-001", templateName: "测试模板",
			mdContent: "内容", requirementType: "", wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewHeartbeatTemplate(NewHeartbeatTemplateID(tt.templateID), tt.templateName, tt.mdContent, tt.requirementType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望错误但没有返回")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("错误信息不匹配: got %v, want contain %v", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf(" unexpected error: %v", err)
			}
			if template.ID().String() != tt.templateID {
				t.Errorf("ID = %v, want %v", template.ID().String(), tt.templateID)
			}
			if template.Name() != strings.TrimSpace(tt.templateName) {
				t.Errorf("Name = %v, want %v", template.Name(), strings.TrimSpace(tt.templateName))
			}
			if template.MDContent() != tt.mdContent {
				t.Errorf("MDContent = %v, want %v", template.MDContent(), tt.mdContent)
			}
			wantType := tt.requirementType
			if wantType == "" {
				wantType = "heartbeat"
			}
			if template.RequirementType() != wantType {
				t.Errorf("RequirementType = %v, want %v", template.RequirementType(), wantType)
			}
		})
	}
}

func TestHeartbeatTemplateUpdate(t *testing.T) {
	template, _ := NewHeartbeatTemplate(NewHeartbeatTemplateID("ht-001"), "旧名称", "旧内容", "normal")

	err := template.Update("新名称", "新内容", "pr_review")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if template.Name() != "新名称" {
		t.Errorf("Name = %v, want 新名称", template.Name())
	}
	if template.MDContent() != "新内容" {
		t.Errorf("MDContent = %v, want 新内容", template.MDContent())
	}
	if template.RequirementType() != "pr_review" {
		t.Errorf("RequirementType = %v, want pr_review", template.RequirementType())
	}

	err = template.Update("", "内容", "optimization")
	if err == nil {
		t.Fatalf("空名称更新应返回错误")
	}
}

func TestHeartbeatTemplateSnapshotRoundTrip(t *testing.T) {
	original, _ := NewHeartbeatTemplate(NewHeartbeatTemplateID("ht-001"), "模板", "内容", "optimization")

	snap := original.ToSnapshot()
	restored := &HeartbeatTemplate{}
	if err := restored.FromSnapshot(snap); err != nil {
		t.Fatalf("FromSnapshot failed: %v", err)
	}

	if restored.ID().String() != original.ID().String() {
		t.Errorf("ID mismatch")
	}
	if restored.Name() != original.Name() {
		t.Errorf("Name mismatch")
	}
	if restored.MDContent() != original.MDContent() {
		t.Errorf("MDContent mismatch")
	}
	if restored.RequirementType() != original.RequirementType() {
		t.Errorf("RequirementType mismatch")
	}
	if !restored.CreatedAt().Equal(original.CreatedAt()) {
		t.Errorf("CreatedAt mismatch")
	}
}
