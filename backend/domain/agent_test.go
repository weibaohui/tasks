/**
 * Agent 聚合根单元测试
 */
package domain

import (
	"testing"
	"time"
)

func TestNewAgent(t *testing.T) {
	agent, err := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"这是一个测试Agent",
		AgentTypeBareLLM,
	)

	if err != nil {
		t.Fatalf("创建Agent失败: %v", err)
	}

	if agent.ID() != NewAgentID("agent-001") {
		t.Errorf("期望AgentID为 agent-001, 实际为 %s", agent.ID())
	}

	if agent.AgentCode() != NewAgentCode("my-agent") {
		t.Errorf("期望AgentCode为 my-agent, 实际为 %s", agent.AgentCode())
	}

	if agent.UserCode() != "user-001" {
		t.Errorf("期望UserCode为 user-001, 实际为 %s", agent.UserCode())
	}

	if agent.Name() != "测试Agent" {
		t.Errorf("期望Name为 测试Agent, 实际为 %s", agent.Name())
	}

	if agent.Description() != "这是一个测试Agent" {
		t.Errorf("期望Description为 这是一个测试Agent, 实际为 %s", agent.Description())
	}

	// 验证默认值
	if agent.MaxTokens() != 4096 {
		t.Errorf("期望MaxTokens为 4096, 实际为 %d", agent.MaxTokens())
	}

	if agent.Temperature() != 0.7 {
		t.Errorf("期望Temperature为 0.7, 实际为 %f", agent.Temperature())
	}

	if agent.MaxIterations() != 15 {
		t.Errorf("期望MaxIterations为 15, 实际为 %d", agent.MaxIterations())
	}

	if agent.HistoryMessages() != 10 {
		t.Errorf("期望HistoryMessages为 10, 实际为 %d", agent.HistoryMessages())
	}

	if !agent.IsActive() {
		t.Error("期望IsActive为true")
	}

	if agent.CreatedAt().IsZero() {
		t.Error("期望CreatedAt不为零值")
	}

	if agent.UpdatedAt().IsZero() {
		t.Error("期望UpdatedAt不为零值")
	}
}

func TestNewAgent_EmptyID(t *testing.T) {
	_, err := NewAgent(
		NewAgentID(""),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	if err != ErrAgentIDRequired {
		t.Errorf("期望返回 ErrAgentIDRequired, 实际返回 %v", err)
	}
}

func TestNewAgent_EmptyAgentCode(t *testing.T) {
	_, err := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode(""),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	if err != ErrAgentCodeRequired {
		t.Errorf("期望返回 ErrAgentCodeRequired, 实际返回 %v", err)
	}
}

func TestNewAgent_EmptyUserCode(t *testing.T) {
	_, err := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"   ", // 空白字符串
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	if err != ErrAgentUserCodeRequired {
		t.Errorf("期望返回 ErrAgentUserCodeRequired, 实际返回 %v", err)
	}
}

func TestNewAgent_EmptyName(t *testing.T) {
	_, err := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"   ", // 空白字符串
		"",
		AgentTypeBareLLM,
	)

	if err != ErrAgentNameRequired {
		t.Errorf("期望返回 ErrAgentNameRequired, 实际返回 %v", err)
	}
}

func TestAgent_UpdateProfile(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"原名称",
		"原描述",
		AgentTypeBareLLM,
	)

	originalUpdatedAt := agent.UpdatedAt()
	time.Sleep(10 * time.Millisecond) // 确保时间戳不同

	err := agent.UpdateProfile("新名称", "新描述")
	if err != nil {
		t.Fatalf("UpdateProfile失败: %v", err)
	}

	if agent.Name() != "新名称" {
		t.Errorf("期望Name为 新名称, 实际为 %s", agent.Name())
	}

	if agent.Description() != "新描述" {
		t.Errorf("期望Description为 新描述, 实际为 %s", agent.Description())
	}

	if !agent.UpdatedAt().After(originalUpdatedAt) {
		t.Error("期望UpdatedAt已更新")
	}
}

func TestAgent_UpdateProfile_EmptyName(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"原名称",
		"原描述",
		AgentTypeBareLLM,
	)

	err := agent.UpdateProfile("", "新描述")
	if err != ErrAgentNameRequired {
		t.Errorf("期望返回 ErrAgentNameRequired, 实际返回 %v", err)
	}

	// 名称不应被修改
	if agent.Name() != "原名称" {
		t.Errorf("名称不应被修改，期望 原名称, 实际为 %s", agent.Name())
	}
}

func TestAgent_UpdateConfig(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	agent.UpdateConfig(
		"身份设定",
		"灵魂设定",
		"Agents设定",
		"用户设定",
		"工具设定",
		"gpt-4",
		8192,
		0.9,
		20,
		20,
		[]string{"skill1", "skill2"},
		[]string{"tool1", "tool2"},
		true,
	)

	if agent.IdentityContent() != "身份设定" {
		t.Errorf("期望IdentityContent为 身份设定, 实际为 %s", agent.IdentityContent())
	}

	if agent.SoulContent() != "灵魂设定" {
		t.Errorf("期望SoulContent为 灵魂设定, 实际为 %s", agent.SoulContent())
	}

	if agent.AgentsContent() != "Agents设定" {
		t.Errorf("期望AgentsContent为 Agents设定, 实际为 %s", agent.AgentsContent())
	}

	if agent.UserContent() != "用户设定" {
		t.Errorf("期望UserContent为 用户设定, 实际为 %s", agent.UserContent())
	}

	if agent.ToolsContent() != "工具设定" {
		t.Errorf("期望ToolsContent为 工具设定, 实际为 %s", agent.ToolsContent())
	}

	if agent.Model() != "gpt-4" {
		t.Errorf("期望Model为 gpt-4, 实际为 %s", agent.Model())
	}

	if agent.MaxTokens() != 8192 {
		t.Errorf("期望MaxTokens为 8192, 实际为 %d", agent.MaxTokens())
	}

	if agent.Temperature() != 0.9 {
		t.Errorf("期望Temperature为 0.9, 实际为 %f", agent.Temperature())
	}

	if agent.MaxIterations() != 20 {
		t.Errorf("期望MaxIterations为 20, 实际为 %d", agent.MaxIterations())
	}

	if agent.HistoryMessages() != 20 {
		t.Errorf("期望HistoryMessages为 20, 实际为 %d", agent.HistoryMessages())
	}

	if len(agent.SkillsList()) != 2 {
		t.Errorf("期望SkillsList长度为 2, 实际为 %d", len(agent.SkillsList()))
	}

	if len(agent.ToolsList()) != 2 {
		t.Errorf("期望ToolsList长度为 2, 实际为 %d", len(agent.ToolsList()))
	}

	if !agent.EnableThinkingProcess() {
		t.Error("期望EnableThinkingProcess为true")
	}
}

func TestAgent_UpdateConfig_PreservesDefaultValues(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	// 只更新部分配置，其他保持不变
	agent.UpdateConfig(
		"新身份",
		"", // 空，保持原值
		"", // 空，保持原值
		"", // 空，保持原值
		"", // 空，保持原值
		"claude-3",
		0,  // 0 不应被设置
		0,  // 0 不应被设置
		0,  // 0 不应被设置
		-1, // 负数不应被设置
		nil,
		nil,
		false,
	)

	// identityContent 被更新
	if agent.IdentityContent() != "新身份" {
		t.Errorf("期望IdentityContent为 新身份, 实际为 %s", agent.IdentityContent())
	}

	// 以下保持默认值
	if agent.SoulContent() != "" {
		t.Errorf("期望SoulContent为空, 实际为 %s", agent.SoulContent())
	}

	// MaxTokens 应该是默认值 4096，不是 0
	if agent.MaxTokens() != 4096 {
		t.Errorf("期望MaxTokens为 4096, 实际为 %d", agent.MaxTokens())
	}

	// Temperature 应该是默认值 0.7，不是 0
	if agent.Temperature() != 0.7 {
		t.Errorf("期望Temperature为 0.7, 实际为 %f", agent.Temperature())
	}

	// MaxIterations 应该是默认值 15，不是 0
	if agent.MaxIterations() != 15 {
		t.Errorf("期望MaxIterations为 15, 实际为 %d", agent.MaxIterations())
	}

	// HistoryMessages 应该是默认值 10，不是 -1
	if agent.HistoryMessages() != 10 {
		t.Errorf("期望HistoryMessages为 10, 实际为 %d", agent.HistoryMessages())
	}

	// EnableThinkingProcess 应该是 false
	if agent.EnableThinkingProcess() {
		t.Error("期望EnableThinkingProcess为false")
	}
}

func TestAgent_SetActive(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	if agent.IsActive() != true {
		t.Error("初始IsActive应为true")
	}

	agent.SetActive(false)
	if agent.IsActive() {
		t.Error("期望IsActive为false")
	}

	agent.SetActive(true)
	if !agent.IsActive() {
		t.Error("期望IsActive为true")
	}
}

func TestAgent_SetDefault(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	if agent.IsDefault() {
		t.Error("初始IsDefault应为false")
	}

	agent.SetDefault(true)
	if !agent.IsDefault() {
		t.Error("期望IsDefault为true")
	}

	agent.SetDefault(false)
	if agent.IsDefault() {
		t.Error("期望IsDefault为false")
	}
}

func TestAgent_ToSnapshot(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"测试描述",
		AgentTypeBareLLM,
	)

	agent.SetActive(false)
	agent.SetDefault(true)

	agent.UpdateConfig(
		"身份",
		"灵魂",
		"",
		"",
		"",
		"gpt-4",
		8192,
		0.8,
		25,
		15,
		[]string{"skill1"},
		[]string{"tool1"},
		true,
	)

	snap := agent.ToSnapshot()

	if snap.ID != agent.ID() {
		t.Errorf("ID不匹配")
	}

	if snap.AgentCode != agent.AgentCode() {
		t.Errorf("AgentCode不匹配")
	}

	if snap.UserCode != agent.UserCode() {
		t.Errorf("UserCode不匹配")
	}

	if snap.Name != agent.Name() {
		t.Errorf("Name不匹配")
	}

	if snap.Description != agent.Description() {
		t.Errorf("Description不匹配")
	}

	if snap.IdentityContent != agent.IdentityContent() {
		t.Errorf("IdentityContent不匹配")
	}

	if snap.SoulContent != agent.SoulContent() {
		t.Errorf("SoulContent不匹配")
	}

	if snap.Model != agent.Model() {
		t.Errorf("Model不匹配")
	}

	if snap.MaxTokens != agent.MaxTokens() {
		t.Errorf("MaxTokens不匹配")
	}

	if snap.Temperature != agent.Temperature() {
		t.Errorf("Temperature不匹配")
	}

	if snap.MaxIterations != agent.MaxIterations() {
		t.Errorf("MaxIterations不匹配")
	}

	if snap.HistoryMessages != agent.HistoryMessages() {
		t.Errorf("HistoryMessages不匹配")
	}

	if snap.IsActive != agent.IsActive() {
		t.Errorf("IsActive不匹配")
	}

	if snap.IsDefault != agent.IsDefault() {
		t.Errorf("IsDefault不匹配")
	}

	if snap.EnableThinkingProcess != agent.EnableThinkingProcess() {
		t.Errorf("EnableThinkingProcess不匹配")
	}

	// 验证切片是深拷贝
	if len(snap.SkillsList) != len(agent.SkillsList()) {
		t.Errorf("SkillsList长度不匹配")
	}

	if len(snap.ToolsList) != len(agent.ToolsList()) {
		t.Errorf("ToolsList长度不匹配")
	}
}

func TestAgent_FromSnapshot(t *testing.T) {
	originalAgent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"测试描述",
		AgentTypeBareLLM,
	)

	originalAgent.SetActive(false)
	originalAgent.SetDefault(true)

	originalAgent.UpdateConfig(
		"身份",
		"灵魂",
		"agents",
		"user",
		"tools",
		"gpt-4",
		8192,
		0.8,
		25,
		15,
		[]string{"skill1", "skill2"},
		[]string{"tool1", "tool2"},
		true,
	)

	snap := originalAgent.ToSnapshot()

	// 从快照恢复
	restoredAgent := &Agent{}
	restoredAgent.FromSnapshot(snap)

	if restoredAgent.ID() != originalAgent.ID() {
		t.Errorf("ID不匹配")
	}

	if restoredAgent.AgentCode() != originalAgent.AgentCode() {
		t.Errorf("AgentCode不匹配")
	}

	if restoredAgent.UserCode() != originalAgent.UserCode() {
		t.Errorf("UserCode不匹配")
	}

	if restoredAgent.Name() != originalAgent.Name() {
		t.Errorf("Name不匹配")
	}

	if restoredAgent.Description() != originalAgent.Description() {
		t.Errorf("Description不匹配")
	}

	if restoredAgent.IdentityContent() != originalAgent.IdentityContent() {
		t.Errorf("IdentityContent不匹配")
	}

	if restoredAgent.SoulContent() != originalAgent.SoulContent() {
		t.Errorf("SoulContent不匹配")
	}

	if restoredAgent.Model() != originalAgent.Model() {
		t.Errorf("Model不匹配")
	}

	if restoredAgent.MaxTokens() != originalAgent.MaxTokens() {
		t.Errorf("MaxTokens不匹配")
	}

	if restoredAgent.Temperature() != originalAgent.Temperature() {
		t.Errorf("Temperature不匹配")
	}

	if restoredAgent.MaxIterations() != originalAgent.MaxIterations() {
		t.Errorf("MaxIterations不匹配")
	}

	if restoredAgent.HistoryMessages() != originalAgent.HistoryMessages() {
		t.Errorf("HistoryMessages不匹配")
	}

	if restoredAgent.IsActive() != originalAgent.IsActive() {
		t.Errorf("IsActive不匹配")
	}

	if restoredAgent.IsDefault() != originalAgent.IsDefault() {
		t.Errorf("IsDefault不匹配")
	}

	if restoredAgent.EnableThinkingProcess() != originalAgent.EnableThinkingProcess() {
		t.Errorf("EnableThinkingProcess不匹配")
	}

	// 验证切片是深拷贝
	if len(restoredAgent.SkillsList()) != len(originalAgent.SkillsList()) {
		t.Errorf("SkillsList长度不匹配")
	}

	if len(restoredAgent.ToolsList()) != len(originalAgent.ToolsList()) {
		t.Errorf("ToolsList长度不匹配")
	}

	// 验证修改原始快照不会影响恢复的Agent
	snap.SkillsList[0] = "modified"
	if restoredAgent.SkillsList()[0] == "modified" {
		t.Error("SkillsList应该是深拷贝，不应受影响")
	}
}

func TestAgent_SkillsList_ReturnsCopy(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	agent.UpdateConfig("", "", "", "", "", "", 0, 0, 0, 0, []string{"skill1", "skill2"}, nil, false)

	skills1 := agent.SkillsList()
	skills1[0] = "modified"

	skills2 := agent.SkillsList()
	if skills2[0] == "modified" {
		t.Error("SkillsList应该返回拷贝，不应受外部修改影响")
	}
}

func TestAgent_ToolsList_ReturnsCopy(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	agent.UpdateConfig("", "", "", "", "", "", 0, 0, 0, 0, nil, []string{"tool1", "tool2"}, false)

	tools1 := agent.ToolsList()
	tools1[0] = "modified"

	tools2 := agent.ToolsList()
	if tools2[0] == "modified" {
		t.Error("ToolsList应该返回拷贝，不应受外部修改影响")
	}
}

func TestAgent_UpdateClaudeCodeConfig(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	// 初始默认配置
	initialConfig := agent.ClaudeCodeConfig()
	if initialConfig.Timeout != 120 {
		t.Errorf("期望初始 Timeout 为 120, 实际为 %d", initialConfig.Timeout)
	}

	// 更新配置
	newConfig := &ClaudeCodeConfig{
		Timeout:       600,
		Model:         "claude-3-5-sonnet",
		MaxThinkingTokens: 10000,
	}
	agent.UpdateClaudeCodeConfig(newConfig)

	// 验证合并后的配置
	updatedConfig := agent.ClaudeCodeConfig()
	if updatedConfig.Timeout != 600 {
		t.Errorf("期望 Timeout 为 600, 实际为 %d", updatedConfig.Timeout)
	}
	if updatedConfig.Model != "claude-3-5-sonnet" {
		t.Errorf("期望 Model 为 claude-3-5-sonnet, 实际为 %s", updatedConfig.Model)
	}
	if updatedConfig.MaxThinkingTokens != 10000 {
		t.Errorf("期望 MaxThinkingTokens 为 10000, 实际为 %d", updatedConfig.MaxThinkingTokens)
	}
}

func TestAgent_UpdateClaudeCodeConfig_NilConfig(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	// 保存原始 Timeout
	originalTimeout := agent.ClaudeCodeConfig().Timeout

	// 传入 nil 不应修改配置
	agent.UpdateClaudeCodeConfig(nil)

	if agent.ClaudeCodeConfig().Timeout != originalTimeout {
		t.Errorf("nil 配置不应修改 Timeout, 期望 %d, 实际 %d", originalTimeout, agent.ClaudeCodeConfig().Timeout)
	}
}

func TestAgent_UpdateClaudeCodeConfig_MergeBehavior(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	// 设置初始配置
	initialConfig := &ClaudeCodeConfig{
		Model:             "claude-3",
		Timeout:           120,
		MaxThinkingTokens: 8000,
		PermissionMode:    PermissionModeDefault,
	}
	agent.SetClaudeCodeConfig(initialConfig)

	// 用新配置合并（只更新 Timeout）
	updateConfig := &ClaudeCodeConfig{
		Timeout: 600,
	}
	agent.UpdateClaudeCodeConfig(updateConfig)

	// 验证：Timeout 被更新，其他字段保留
	config := agent.ClaudeCodeConfig()
	if config.Timeout != 600 {
		t.Errorf("期望 Timeout 为 600, 实际为 %d", config.Timeout)
	}
	if config.Model != "claude-3" {
		t.Errorf("期望 Model 为 claude-3, 实际为 %s", config.Model)
	}
	if config.MaxThinkingTokens != 8000 {
		t.Errorf("期望 MaxThinkingTokens 为 8000, 实际为 %d", config.MaxThinkingTokens)
	}
	if config.PermissionMode != PermissionModeDefault {
		t.Errorf("期望 PermissionMode 为 Default, 实际为 %s", config.PermissionMode)
	}
}

func TestAgent_SetClaudeCodeConfig(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	config := &ClaudeCodeConfig{
		Timeout:       300,
		Model:         "claude-3-opus",
		MaxThinkingTokens: 12000,
	}
	agent.SetClaudeCodeConfig(config)

 retrieved := agent.ClaudeCodeConfig()
	if retrieved.Timeout != 300 {
		t.Errorf("期望 Timeout 为 300, 实际为 %d", retrieved.Timeout)
	}
	if retrieved.Model != "claude-3-opus" {
		t.Errorf("期望 Model 为 claude-3-opus, 实际为 %s", retrieved.Model)
	}
}

func TestAgent_ClaudeCodeConfig_ReturnsDefault(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	// 未设置配置时，应该返回默认配置
	config := agent.ClaudeCodeConfig()
	if config == nil {
		t.Fatal("ClaudeCodeConfig 不应返回 nil")
	}
	if config.Timeout != 120 {
		t.Errorf("期望默认 Timeout 为 120, 实际为 %d", config.Timeout)
	}
}

func TestAgent_ToSnapshot_WithClaudeCodeConfig(t *testing.T) {
	agent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	config := &ClaudeCodeConfig{
		Timeout:           600,
		Model:             "claude-3-5-sonnet",
		MaxThinkingTokens: 10000,
	}
	agent.SetClaudeCodeConfig(config)

	snap := agent.ToSnapshot()

	if snap.ClaudeCodeConfig == nil {
		t.Fatal("Snapshot 的 ClaudeCodeConfig 不应为 nil")
	}
	if snap.ClaudeCodeConfig.Timeout != 600 {
		t.Errorf("期望 Snapshot Timeout 为 600, 实际为 %d", snap.ClaudeCodeConfig.Timeout)
	}
	if snap.ClaudeCodeConfig.Model != "claude-3-5-sonnet" {
		t.Errorf("期望 Snapshot Model 为 claude-3-5-sonnet, 实际为 %s", snap.ClaudeCodeConfig.Model)
	}
}

func TestAgent_FromSnapshot_WithClaudeCodeConfig(t *testing.T) {
	originalAgent, _ := NewAgent(
		NewAgentID("agent-001"),
		NewAgentCode("my-agent"),
		"user-001",
		"测试Agent",
		"",
		AgentTypeBareLLM,
	)

	config := &ClaudeCodeConfig{
		Timeout:           600,
		Model:             "claude-3-5-sonnet",
		MaxThinkingTokens: 10000,
	}
	originalAgent.SetClaudeCodeConfig(config)

	snap := originalAgent.ToSnapshot()

	restoredAgent := &Agent{}
	restoredAgent.FromSnapshot(snap)

	retrievedConfig := restoredAgent.ClaudeCodeConfig()
	if retrievedConfig.Timeout != 600 {
		t.Errorf("期望恢复的 Timeout 为 600, 实际为 %d", retrievedConfig.Timeout)
	}
	if retrievedConfig.Model != "claude-3-5-sonnet" {
		t.Errorf("期望恢复的 Model 为 claude-3-5-sonnet, 实际为 %s", retrievedConfig.Model)
	}
}
