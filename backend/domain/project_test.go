package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestNewProject(t *testing.T) {
	tests := []struct {
		name          string
		id            ProjectID
		projectName   string
		gitRepoURL    string
		defaultBranch string
		initSteps     []string
		wantErr       error
	}{
		{
			name:          "正常创建项目",
			id:            NewProjectID("proj-001"),
			projectName:   "测试项目",
			gitRepoURL:    "https://github.com/test/project.git",
			defaultBranch: "main",
			initSteps:     []string{"step1", "step2"},
			wantErr:       nil,
		},
		{
			name:          "空默认分支使用main",
			id:            NewProjectID("proj-002"),
			projectName:   "测试项目2",
			gitRepoURL:    "https://github.com/test/project2.git",
			defaultBranch: "",
			initSteps:     []string{},
			wantErr:       nil,
		},
		{
			name:          "空ID返回错误",
			id:            NewProjectID(""),
			projectName:   "测试项目",
			gitRepoURL:    "https://github.com/test/project.git",
			defaultBranch: "main",
			initSteps:     []string{},
			wantErr:       ErrProjectIDRequired,
		},
		{
			name:          "空名称返回错误",
			id:            NewProjectID("proj-003"),
			projectName:   "",
			gitRepoURL:    "https://github.com/test/project.git",
			defaultBranch: "main",
			initSteps:     []string{},
			wantErr:       ErrProjectNameRequired,
		},
		{
			name:          "只有空格名称返回错误",
			id:            NewProjectID("proj-004"),
			projectName:   "   ",
			gitRepoURL:    "https://github.com/test/project.git",
			defaultBranch: "main",
			initSteps:     []string{},
			wantErr:       ErrProjectNameRequired,
		},
		{
			name:          "空仓库URL返回错误",
			id:            NewProjectID("proj-005"),
			projectName:   "测试项目",
			gitRepoURL:    "",
			defaultBranch: "main",
			initSteps:     []string{},
			wantErr:       ErrProjectRepoURLRequired,
		},
		{
			name:          "只有空格仓库URL返回错误",
			id:            NewProjectID("proj-006"),
			projectName:   "测试项目",
			gitRepoURL:    "   ",
			defaultBranch: "main",
			initSteps:     []string{},
			wantErr:       ErrProjectRepoURLRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := NewProject(tt.id, tt.projectName, tt.gitRepoURL, tt.defaultBranch, tt.initSteps)
			if err != tt.wantErr {
				t.Fatalf("期望错误 %v, 实际 %v", tt.wantErr, err)
			}
			if err != nil {
				return
			}

			if project.ID().String() != tt.id.String() {
				t.Errorf("期望ID %s, 实际 %s", tt.id.String(), project.ID().String())
			}
			if project.Name() != tt.projectName {
				t.Errorf("期望名称 %s, 实际 %s", tt.projectName, project.Name())
			}
			if project.GitRepoURL() != tt.gitRepoURL {
				t.Errorf("期望仓库URL %s, 实际 %s", tt.gitRepoURL, project.GitRepoURL())
			}
			expectedBranch := tt.defaultBranch
			if expectedBranch == "" {
				expectedBranch = "main"
			}
			if project.DefaultBranch() != expectedBranch {
				t.Errorf("期望默认分支 %s, 实际 %s", expectedBranch, project.DefaultBranch())
			}

			// 验证心跳默认值
			if project.HeartbeatEnabled() != false {
				t.Errorf("期望心跳默认禁用, 实际 %v", project.HeartbeatEnabled())
			}
			if project.HeartbeatIntervalMinutes() != 60 {
				t.Errorf("期望心跳间隔默认60分钟, 实际 %d", project.HeartbeatIntervalMinutes())
			}
		})
	}
}

func TestProjectAccessors(t *testing.T) {
	id := NewProjectID("proj-001")
	name := "测试项目"
	gitRepoURL := "https://github.com/test/project.git"
	defaultBranch := "develop"
	initSteps := []string{"step1", "step2", "step3"}

	project, err := NewProject(id, name, gitRepoURL, defaultBranch, initSteps)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	// 测试所有访问器
	if project.ID().String() != "proj-001" {
		t.Errorf("ID() 期望 proj-001, 实际 %s", project.ID().String())
	}
	if project.Name() != name {
		t.Errorf("Name() 期望 %s, 实际 %s", name, project.Name())
	}
	if project.GitRepoURL() != gitRepoURL {
		t.Errorf("GitRepoURL() 期望 %s, 实际 %s", gitRepoURL, project.GitRepoURL())
	}
	if project.DefaultBranch() != defaultBranch {
		t.Errorf("DefaultBranch() 期望 %s, 实际 %s", defaultBranch, project.DefaultBranch())
	}
	if !reflect.DeepEqual(project.InitSteps(), initSteps) {
		t.Errorf("InitSteps() 期望 %v, 实际 %v", initSteps, project.InitSteps())
	}
	if project.HeartbeatEnabled() != false {
		t.Errorf("HeartbeatEnabled() 期望 false, 实际 %v", project.HeartbeatEnabled())
	}
	if project.HeartbeatIntervalMinutes() != 60 {
		t.Errorf("HeartbeatIntervalMinutes() 期望 60, 实际 %d", project.HeartbeatIntervalMinutes())
	}
	if project.HeartbeatMDContent() != "" {
		t.Errorf("HeartbeatMDContent() 期望空字符串, 实际 %s", project.HeartbeatMDContent())
	}
	if project.AgentCode() != "" {
		t.Errorf("AgentCode() 期望空字符串, 实际 %s", project.AgentCode())
	}
	if project.DispatchChannelCode() != "" {
		t.Errorf("DispatchChannelCode() 期望空字符串, 实际 %s", project.DispatchChannelCode())
	}
	if project.DispatchSessionKey() != "" {
		t.Errorf("DispatchSessionKey() 期望空字符串, 实际 %s", project.DispatchSessionKey())
	}
	if project.CreatedAt().IsZero() {
		t.Error("CreatedAt() 不应为零时间")
	}
	if project.UpdatedAt().IsZero() {
		t.Error("UpdatedAt() 不应为零时间")
	}
}

func TestProjectUpdate(t *testing.T) {
	project, err := NewProject(
		NewProjectID("proj-001"),
		"原名称",
		"https://github.com/test/old.git",
		"main",
		[]string{"old-step"},
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	oldUpdatedAt := project.UpdatedAt()
	time.Sleep(10 * time.Millisecond) // 确保时间有变化

	err = project.Update("新名称", "https://github.com/test/new.git", "develop", []string{"new-step1", "new-step2"})
	if err != nil {
		t.Fatalf("更新项目失败: %v", err)
	}

	if project.Name() != "新名称" {
		t.Errorf("期望名称更新为 新名称, 实际 %s", project.Name())
	}
	if project.GitRepoURL() != "https://github.com/test/new.git" {
		t.Errorf("期望仓库URL更新, 实际 %s", project.GitRepoURL())
	}
	if project.DefaultBranch() != "develop" {
		t.Errorf("期望默认分支更新为 develop, 实际 %s", project.DefaultBranch())
	}
	expectedSteps := []string{"new-step1", "new-step2"}
	if !reflect.DeepEqual(project.InitSteps(), expectedSteps) {
		t.Errorf("期望初始化步骤更新为 %v, 实际 %v", expectedSteps, project.InitSteps())
	}
	if !project.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestProjectUpdateValidation(t *testing.T) {
	project, err := NewProject(
		NewProjectID("proj-001"),
		"原名称",
		"https://github.com/test/old.git",
		"main",
		[]string{},
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	tests := []struct {
		name          string
		projectName   string
		gitRepoURL    string
		defaultBranch string
		wantErr       error
	}{
		{
			name:          "空名称",
			projectName:   "",
			gitRepoURL:    "https://github.com/test/project.git",
			defaultBranch: "main",
			wantErr:       ErrProjectNameRequired,
		},
		{
			name:          "只有空格名称",
			projectName:   "   ",
			gitRepoURL:    "https://github.com/test/project.git",
			defaultBranch: "main",
			wantErr:       ErrProjectNameRequired,
		},
		{
			name:          "空仓库URL",
			projectName:   "测试项目",
			gitRepoURL:    "",
			defaultBranch: "main",
			wantErr:       ErrProjectRepoURLRequired,
		},
		{
			name:          "只有空格仓库URL",
			projectName:   "测试项目",
			gitRepoURL:    "   ",
			defaultBranch: "main",
			wantErr:       ErrProjectRepoURLRequired,
		},
		{
			name:          "空默认分支使用main",
			projectName:   "测试项目",
			gitRepoURL:    "https://github.com/test/project.git",
			defaultBranch: "",
			wantErr:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := project.Update(tt.projectName, tt.gitRepoURL, tt.defaultBranch, []string{})
			if err != tt.wantErr {
				t.Errorf("期望错误 %v, 实际 %v", tt.wantErr, err)
			}
			if err == nil && tt.defaultBranch == "" {
				if project.DefaultBranch() != "main" {
					t.Errorf("期望默认分支为 main, 实际 %s", project.DefaultBranch())
				}
			}
		})
	}
}

func TestProjectUpdateHeartbeatConfig(t *testing.T) {
	project, err := NewProject(
		NewProjectID("proj-001"),
		"测试项目",
		"https://github.com/test/project.git",
		"main",
		[]string{},
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	oldUpdatedAt := project.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	// 测试更新所有字段
	enabled := true
	interval := 30
	mdContent := "# Heartbeat Content"
	agentCode := "agent-001"

	project.UpdateHeartbeatConfig(&enabled, &interval, &mdContent, &agentCode)

	if project.HeartbeatEnabled() != true {
		t.Errorf("期望心跳启用, 实际 %v", project.HeartbeatEnabled())
	}
	if project.HeartbeatIntervalMinutes() != 30 {
		t.Errorf("期望心跳间隔30分钟, 实际 %d", project.HeartbeatIntervalMinutes())
	}
	if project.HeartbeatMDContent() != "# Heartbeat Content" {
		t.Errorf("期望心跳内容正确, 实际 %s", project.HeartbeatMDContent())
	}
	if project.AgentCode() != "agent-001" {
		t.Errorf("期望Agent代码正确, 实际 %s", project.AgentCode())
	}
	if !project.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestProjectUpdateHeartbeatConfigNilValues(t *testing.T) {
	project, err := NewProject(
		NewProjectID("proj-001"),
		"测试项目",
		"https://github.com/test/project.git",
		"main",
		[]string{},
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	// 先更新一次
	enabled := true
	interval := 30
	mdContent := "# Content"
	agentCode := "agent-001"
	project.UpdateHeartbeatConfig(&enabled, &interval, &mdContent, &agentCode)

	oldUpdatedAt := project.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	// 测试 nil 值不更新
	project.UpdateHeartbeatConfig(nil, nil, nil, nil)

	if project.HeartbeatEnabled() != true {
		t.Errorf("nil enabled 不应改变值, 实际 %v", project.HeartbeatEnabled())
	}
	if project.HeartbeatIntervalMinutes() != 30 {
		t.Errorf("nil interval 不应改变值, 实际 %d", project.HeartbeatIntervalMinutes())
	}
	if project.HeartbeatMDContent() != "# Content" {
		t.Errorf("nil mdContent 不应改变值, 实际 %s", project.HeartbeatMDContent())
	}
	if project.AgentCode() != "agent-001" {
		t.Errorf("nil agentCode 不应改变值, 实际 %s", project.AgentCode())
	}
	// nil 值不更新，但 UpdatedAt 仍应被更新
	if !project.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestProjectUpdateDispatchConfig(t *testing.T) {
	project, err := NewProject(
		NewProjectID("proj-001"),
		"测试项目",
		"https://github.com/test/project.git",
		"main",
		[]string{},
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	oldUpdatedAt := project.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	channelCode := "feishu"
	sessionKey := "session-001"

	project.UpdateDispatchConfig(&channelCode, &sessionKey)

	if project.DispatchChannelCode() != "feishu" {
		t.Errorf("期望派发通道代码正确, 实际 %s", project.DispatchChannelCode())
	}
	if project.DispatchSessionKey() != "session-001" {
		t.Errorf("期望派发会话键正确, 实际 %s", project.DispatchSessionKey())
	}
	if !project.UpdatedAt().After(oldUpdatedAt) {
		t.Error("期望 UpdatedAt 被更新")
	}
}

func TestProjectUpdateDispatchConfigNilAndEmpty(t *testing.T) {
	project, err := NewProject(
		NewProjectID("proj-001"),
		"测试项目",
		"https://github.com/test/project.git",
		"main",
		[]string{},
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	// 先设置值
	channelCode := "feishu"
	sessionKey := "session-001"
	project.UpdateDispatchConfig(&channelCode, &sessionKey)

	// 测试 nil 值不更新
	project.UpdateDispatchConfig(nil, nil)

	if project.DispatchChannelCode() != "feishu" {
		t.Errorf("nil channelCode 不应改变值, 实际 %s", project.DispatchChannelCode())
	}
	if project.DispatchSessionKey() != "session-001" {
		t.Errorf("nil sessionKey 不应改变值, 实际 %s", project.DispatchSessionKey())
	}

	// 测试空字符串不更新
	emptyString := ""
	project.UpdateDispatchConfig(&emptyString, &emptyString)

	if project.DispatchChannelCode() != "feishu" {
		t.Errorf("空字符串 channelCode 不应改变值, 实际 %s", project.DispatchChannelCode())
	}
	if project.DispatchSessionKey() != "session-001" {
		t.Errorf("空字符串 sessionKey 不应改变值, 实际 %s", project.DispatchSessionKey())
	}
}

func TestProjectToSnapshot(t *testing.T) {
	id := NewProjectID("proj-001")
	name := "测试项目"
	gitRepoURL := "https://github.com/test/project.git"
	defaultBranch := "develop"
	initSteps := []string{"step1", "step2"}

	project, err := NewProject(id, name, gitRepoURL, defaultBranch, initSteps)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	// 设置额外字段
	enabled := true
	interval := 30
	mdContent := "# Content"
	agentCode := "agent-001"
	channelCode := "feishu"
	sessionKey := "session-001"
	project.UpdateHeartbeatConfig(&enabled, &interval, &mdContent, &agentCode)
	project.UpdateDispatchConfig(&channelCode, &sessionKey)

	snap := project.ToSnapshot()

	if snap.ID.String() != "proj-001" {
		t.Errorf("快照ID期望 proj-001, 实际 %s", snap.ID.String())
	}
	if snap.Name != name {
		t.Errorf("快照名称期望 %s, 实际 %s", name, snap.Name)
	}
	if snap.GitRepoURL != gitRepoURL {
		t.Errorf("快照仓库URL期望 %s, 实际 %s", gitRepoURL, snap.GitRepoURL)
	}
	if snap.DefaultBranch != defaultBranch {
		t.Errorf("快照默认分支期望 %s, 实际 %s", defaultBranch, snap.DefaultBranch)
	}
	if !reflect.DeepEqual(snap.InitSteps, initSteps) {
		t.Errorf("快照初始化步骤期望 %v, 实际 %v", initSteps, snap.InitSteps)
	}
	if snap.HeartbeatEnabled != true {
		t.Errorf("快照心跳启用期望 true, 实际 %v", snap.HeartbeatEnabled)
	}
	if snap.HeartbeatIntervalMinutes != 30 {
		t.Errorf("快照心跳间隔期望 30, 实际 %d", snap.HeartbeatIntervalMinutes)
	}
	if snap.HeartbeatMDContent != "# Content" {
		t.Errorf("快照心跳内容期望 '# Content', 实际 %s", snap.HeartbeatMDContent)
	}
	if snap.AgentCode != "agent-001" {
		t.Errorf("快照Agent代码期望 'agent-001', 实际 %s", snap.AgentCode)
	}
	if snap.DispatchChannelCode != "feishu" {
		t.Errorf("快照派发通道代码期望 'feishu', 实际 %s", snap.DispatchChannelCode)
	}
	if snap.DispatchSessionKey != "session-001" {
		t.Errorf("快照派发会话键期望 'session-001', 实际 %s", snap.DispatchSessionKey)
	}
	if snap.CreatedAt.IsZero() {
		t.Error("快照创建时间不应为零")
	}
	if snap.UpdatedAt.IsZero() {
		t.Error("快照更新时间不应为零")
	}
}

func TestProjectFromSnapshot(t *testing.T) {
	snap := ProjectSnapshot{
		ID:                       NewProjectID("proj-001"),
		Name:                     "快照项目",
		GitRepoURL:               "https://github.com/snapshot/project.git",
		DefaultBranch:            "feature-branch",
		InitSteps:                []string{"snap-step1", "snap-step2"},
		HeartbeatEnabled:         true,
		HeartbeatIntervalMinutes: 45,
		HeartbeatMDContent:       "# Snapshot Content",
		AgentCode:                "snap-agent",
		DispatchChannelCode:      "snap-channel",
		DispatchSessionKey:       "snap-session",
		CreatedAt:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:                time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	}

	project := &Project{}
	project.FromSnapshot(snap)

	if project.ID().String() != "proj-001" {
		t.Errorf("ID期望 proj-001, 实际 %s", project.ID().String())
	}
	if project.Name() != "快照项目" {
		t.Errorf("名称期望 快照项目, 实际 %s", project.Name())
	}
	if project.GitRepoURL() != "https://github.com/snapshot/project.git" {
		t.Errorf("仓库URL期望 snapshot URL, 实际 %s", project.GitRepoURL())
	}
	if project.DefaultBranch() != "feature-branch" {
		t.Errorf("默认分支期望 feature-branch, 实际 %s", project.DefaultBranch())
	}
	expectedSteps := []string{"snap-step1", "snap-step2"}
	if !reflect.DeepEqual(project.InitSteps(), expectedSteps) {
		t.Errorf("初始化步骤期望 %v, 实际 %v", expectedSteps, project.InitSteps())
	}
	if project.HeartbeatEnabled() != true {
		t.Errorf("心跳启用期望 true, 实际 %v", project.HeartbeatEnabled())
	}
	if project.HeartbeatIntervalMinutes() != 45 {
		t.Errorf("心跳间隔期望 45, 实际 %d", project.HeartbeatIntervalMinutes())
	}
	if project.HeartbeatMDContent() != "# Snapshot Content" {
		t.Errorf("心跳内容期望 '# Snapshot Content', 实际 %s", project.HeartbeatMDContent())
	}
	if project.AgentCode() != "snap-agent" {
		t.Errorf("Agent代码期望 snap-agent, 实际 %s", project.AgentCode())
	}
	if project.DispatchChannelCode() != "snap-channel" {
		t.Errorf("派发通道代码期望 snap-channel, 实际 %s", project.DispatchChannelCode())
	}
	if project.DispatchSessionKey() != "snap-session" {
		t.Errorf("派发会话键期望 snap-session, 实际 %s", project.DispatchSessionKey())
	}
	if !project.CreatedAt().Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("创建时间期望 2024-01-01, 实际 %v", project.CreatedAt())
	}
	if !project.UpdatedAt().Equal(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("更新时间期望 2024-06-01, 实际 %v", project.UpdatedAt())
	}
}

func TestProjectSnapshotRoundTrip(t *testing.T) {
	// 创建一个完整配置的项目
	id := NewProjectID("proj-001")
	name := "原始项目"
	gitRepoURL := "https://github.com/original/project.git"
	defaultBranch := "main"
	initSteps := []string{"step1", "step2", "step3"}

	original, err := NewProject(id, name, gitRepoURL, defaultBranch, initSteps)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	// 设置所有配置
	enabled := true
	interval := 120
	mdContent := "# Original MD"
	agentCode := "original-agent"
	channelCode := "original-channel"
	sessionKey := "original-session"
	original.UpdateHeartbeatConfig(&enabled, &interval, &mdContent, &agentCode)
	original.UpdateDispatchConfig(&channelCode, &sessionKey)

	// 转换为快照
	snap := original.ToSnapshot()

	// 从快照恢复
	restored := &Project{}
	restored.FromSnapshot(snap)

	// 验证所有字段
	if restored.ID().String() != original.ID().String() {
		t.Errorf("ID不匹配: 期望 %s, 实际 %s", original.ID().String(), restored.ID().String())
	}
	if restored.Name() != original.Name() {
		t.Errorf("名称不匹配: 期望 %s, 实际 %s", original.Name(), restored.Name())
	}
	if restored.GitRepoURL() != original.GitRepoURL() {
		t.Errorf("仓库URL不匹配")
	}
	if restored.DefaultBranch() != original.DefaultBranch() {
		t.Errorf("默认分支不匹配")
	}
	if !reflect.DeepEqual(restored.InitSteps(), original.InitSteps()) {
		t.Errorf("初始化步骤不匹配: 期望 %v, 实际 %v", original.InitSteps(), restored.InitSteps())
	}
	if restored.HeartbeatEnabled() != original.HeartbeatEnabled() {
		t.Errorf("心跳启用不匹配")
	}
	if restored.HeartbeatIntervalMinutes() != original.HeartbeatIntervalMinutes() {
		t.Errorf("心跳间隔不匹配: 期望 %d, 实际 %d", original.HeartbeatIntervalMinutes(), restored.HeartbeatIntervalMinutes())
	}
	if restored.HeartbeatMDContent() != original.HeartbeatMDContent() {
		t.Errorf("心跳内容不匹配")
	}
	if restored.AgentCode() != original.AgentCode() {
		t.Errorf("Agent代码不匹配")
	}
	if restored.DispatchChannelCode() != original.DispatchChannelCode() {
		t.Errorf("派发通道代码不匹配")
	}
	if restored.DispatchSessionKey() != original.DispatchSessionKey() {
		t.Errorf("派发会话键不匹配")
	}
	if !restored.CreatedAt().Equal(original.CreatedAt()) {
		t.Errorf("创建时间不匹配")
	}
	if !restored.UpdatedAt().Equal(original.UpdatedAt()) {
		t.Errorf("更新时间不匹配")
	}
}

func TestProjectInitStepsCopy(t *testing.T) {
	// 测试 InitSteps 返回的是副本，修改不影响原始数据
	originalSteps := []string{"step1", "step2", "step3"}
	project, err := NewProject(
		NewProjectID("proj-001"),
		"测试项目",
		"https://github.com/test/project.git",
		"main",
		originalSteps,
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	// 获取步骤并修改
	steps := project.InitSteps()
	steps[0] = "modified-step"
	steps = append(steps, "new-step")

	// 验证原始项目未受影响
	retrievedSteps := project.InitSteps()
	if retrievedSteps[0] != "step1" {
		t.Errorf("InitSteps 应返回副本，修改不应影响原始数据: 期望 step1, 实际 %s", retrievedSteps[0])
	}
	if len(retrievedSteps) != 3 {
		t.Errorf("InitSteps 长度应保持不变: 期望 3, 实际 %d", len(retrievedSteps))
	}
}

func TestProjectInitStepsUpdateCopy(t *testing.T) {
	// 测试 Update 方法中的 steps 也是复制的
	project, err := NewProject(
		NewProjectID("proj-001"),
		"测试项目",
		"https://github.com/test/project.git",
		"main",
		[]string{},
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	newSteps := []string{"step1", "step2"}
	err = project.Update("新名称", "https://github.com/test/new.git", "main", newSteps)
	if err != nil {
		t.Fatalf("更新项目失败: %v", err)
	}

	// 修改原始切片
	newSteps[0] = "modified"

	// 验证项目中的步骤未受影响
	retrievedSteps := project.InitSteps()
	if retrievedSteps[0] != "step1" {
		t.Errorf("Update 应复制 steps，修改原始切片不应影响项目: 期望 step1, 实际 %s", retrievedSteps[0])
	}
}

func TestProjectSnapshotStepsCopy(t *testing.T) {
	// 测试快照中的步骤也是复制的
	originalSteps := []string{"step1", "step2"}
	project, err := NewProject(
		NewProjectID("proj-001"),
		"测试项目",
		"https://github.com/test/project.git",
		"main",
		originalSteps,
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	snap := project.ToSnapshot()

	// 修改快照中的步骤
	snap.InitSteps[0] = "modified"

	// 验证项目中的步骤未受影响
	retrievedSteps := project.InitSteps()
	if retrievedSteps[0] != "step1" {
		t.Errorf("ToSnapshot 应复制 steps: 期望 step1, 实际 %s", retrievedSteps[0])
	}
}

func TestProjectFromSnapshotStepsCopy(t *testing.T) {
	// 测试从快照恢复时也复制步骤
	snapSteps := []string{"step1", "step2"}
	snap := ProjectSnapshot{
		ID:            NewProjectID("proj-001"),
		Name:          "快照项目",
		GitRepoURL:    "https://github.com/test/project.git",
		DefaultBranch: "main",
		InitSteps:     snapSteps,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	project := &Project{}
	project.FromSnapshot(snap)

	// 修改快照中的步骤
	snap.InitSteps[0] = "modified"

	// 验证项目中的步骤未受影响
	retrievedSteps := project.InitSteps()
	if retrievedSteps[0] != "step1" {
		t.Errorf("FromSnapshot 应复制 steps: 期望 step1, 实际 %s", retrievedSteps[0])
	}
}

func TestProjectNewProjectStepsCopy(t *testing.T) {
	// 测试 NewProject 时传入的 steps 被复制
	inputSteps := []string{"step1", "step2"}
	project, err := NewProject(
		NewProjectID("proj-001"),
		"测试项目",
		"https://github.com/test/project.git",
		"main",
		inputSteps,
	)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}

	// 修改原始切片
	inputSteps[0] = "modified"

	// 验证项目中的步骤未受影响
	retrievedSteps := project.InitSteps()
	if retrievedSteps[0] != "step1" {
		t.Errorf("NewProject 应复制 steps: 期望 step1, 实际 %s", retrievedSteps[0])
	}
}
