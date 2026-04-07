package application

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

type mockAgentRepo struct {
	agents map[string]*domain.Agent
}

func newMockAgentRepo() *mockAgentRepo {
	return &mockAgentRepo{
		agents: make(map[string]*domain.Agent),
	}
}

func (m *mockAgentRepo) Save(ctx context.Context, agent *domain.Agent) error {
	m.agents[agent.ID().String()] = agent
	return nil
}

func (m *mockAgentRepo) FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
	agent, ok := m.agents[id.String()]
	if !ok {
		return nil, nil
	}
	return agent, nil
}

func (m *mockAgentRepo) FindAll(ctx context.Context) ([]*domain.Agent, error) {
	var result []*domain.Agent
	for _, agent := range m.agents {
		result = append(result, agent)
	}
	return result, nil
}

func (m *mockAgentRepo) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Agent, error) {
	var result []*domain.Agent
	for _, agent := range m.agents {
		if agent.UserCode() == userCode {
			result = append(result, agent)
		}
	}
	return result, nil
}

func (m *mockAgentRepo) FindByAgentCode(ctx context.Context, code domain.AgentCode) (*domain.Agent, error) {
	for _, agent := range m.agents {
		if agent.AgentCode().String() == code.String() {
			return agent, nil
		}
	}
	return nil, nil
}

func (m *mockAgentRepo) Delete(ctx context.Context, id domain.AgentID) error {
	delete(m.agents, id.String())
	return nil
}

type mockAgentIDGen struct {
	count int
}

func (m *mockAgentIDGen) Generate() string {
	m.count++
	return "agent-id-" + strconv.Itoa(m.count)
}

func setupTestAgentSvc() *AgentApplicationService {
	repo := newMockAgentRepo()
	idGen := &mockAgentIDGen{}
	return NewAgentApplicationService(repo, idGen)
}

func TestAgentService_CreateAgent(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	agent, err := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "测试Agent",
		Model:    "gpt-4",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if agent.Name() != "测试Agent" {
		t.Errorf("期望 name 为 '测试Agent', 实际为 '%s'", agent.Name())
	}

	if agent.UserCode() != "usr_001" {
		t.Errorf("期望 user_code 为 'usr_001', 实际为 '%s'", agent.UserCode())
	}

	if agent.Model() != "gpt-4" {
		t.Errorf("期望 model 为 'gpt-4', 实际为 '%s'", agent.Model())
	}

	if !agent.IsActive() {
		t.Error("新创建的 agent 应该是激活状态")
	}
}

func TestAgentService_CreateAgent_WithFullConfig(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	agent, err := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:              "usr_001",
		Name:                  "完整配置Agent",
		Description:           "测试描述",
		IdentityContent:       "# Identity",
		SoulContent:           "# Soul",
		AgentsContent:         "# Agents",
		UserContent:           "# User",
		ToolsContent:          "# Tools",
		Model:                 "gpt-4",
		MaxTokens:             8000,
		Temperature:           0.9,
		MaxIterations:         20,
		HistoryMessages:       20,
		SkillsList:            []string{"skill1", "skill2"},
		ToolsList:             []string{"tool1", "tool2"},
		IsDefault:             true,
		EnableThinkingProcess: true,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if agent.Description() != "测试描述" {
		t.Errorf("期望 description 为 '测试描述', 实际为 '%s'", agent.Description())
	}

	if agent.MaxTokens() != 8000 {
		t.Errorf("期望 max_tokens 为 8000, 实际为 %d", agent.MaxTokens())
	}

	if agent.Temperature() != 0.9 {
		t.Errorf("期望 temperature 为 0.9, 实际为 %f", agent.Temperature())
	}

	if agent.MaxIterations() != 20 {
		t.Errorf("期望 max_iterations 为 20, 实际为 %d", agent.MaxIterations())
	}

	if !agent.IsDefault() {
		t.Error("agent 应该是默认状态")
	}

	if !agent.EnableThinkingProcess() {
		t.Error("agent 应该启用思考过程")
	}
}

func TestAgentService_CreateAgent_DefaultConfig(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	// 只提供最小配置
	agent, err := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "最小配置Agent",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证默认值
	if agent.Description() != "默认 Agent" {
		t.Errorf("期望 description 为 '默认 Agent', 实际为 '%s'", agent.Description())
	}

	if agent.MaxTokens() != 4096 {
		t.Errorf("期望 max_tokens 为 4096, 实际为 %d", agent.MaxTokens())
	}

	if agent.Temperature() != 0.7 {
		t.Errorf("期望 temperature 为 0.7, 实际为 %f", agent.Temperature())
	}

	if agent.MaxIterations() != 15 {
		t.Errorf("期望 max_iterations 为 15, 实际为 %d", agent.MaxIterations())
	}

	if agent.HistoryMessages() != 10 {
		t.Errorf("期望 history_messages 为 10, 实际为 %d", agent.HistoryMessages())
	}

	if agent.IdentityContent() == "" {
		t.Error("identity_content 不应该为空")
	}

	if agent.SoulContent() == "" {
		t.Error("soul_content 不应该为空")
	}
}

func TestAgentService_GetAgent(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "GetTestAgent",
	})

	agent, err := svc.GetAgent(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if agent.Name() != "GetTestAgent" {
		t.Errorf("期望 name 为 'GetTestAgent', 实际为 '%s'", agent.Name())
	}
}

func TestAgentService_GetAgent_NotFound(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	_, err := svc.GetAgent(ctx, domain.NewAgentID("non-existent"))
	if err != ErrAgentNotFound {
		t.Errorf("期望 ErrAgentNotFound, 实际为 %v", err)
	}
}

func TestAgentService_GetAgentByCode(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "GetByCodeAgent",
	})

	agent, err := svc.GetAgentByCode(ctx, created.AgentCode())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if agent.Name() != "GetByCodeAgent" {
		t.Errorf("期望 name 为 'GetByCodeAgent', 实际为 '%s'", agent.Name())
	}
}

func TestAgentService_GetAgentByCode_NotFound(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	_, err := svc.GetAgentByCode(ctx, domain.NewAgentCode("agt_non-existent"))
	if err != ErrAgentNotFound {
		t.Errorf("期望 ErrAgentNotFound, 实际为 %v", err)
	}
}

func TestAgentService_ListAgents(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	svc.CreateAgent(ctx, CreateAgentCommand{UserCode: "usr_001", Name: "Agent1"})
	svc.CreateAgent(ctx, CreateAgentCommand{UserCode: "usr_001", Name: "Agent2"})
	svc.CreateAgent(ctx, CreateAgentCommand{UserCode: "usr_002", Name: "Agent3"})

	agents, err := svc.ListAgents(ctx, "")
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(agents) != 3 {
		t.Errorf("期望 3 个 agents, 实际为 %d", len(agents))
	}
}

func TestAgentService_ListAgents_FilterByUserCode(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	svc.CreateAgent(ctx, CreateAgentCommand{UserCode: "usr_001", Name: "Agent1"})
	svc.CreateAgent(ctx, CreateAgentCommand{UserCode: "usr_001", Name: "Agent2"})
	svc.CreateAgent(ctx, CreateAgentCommand{UserCode: "usr_002", Name: "Agent3"})

	agents, err := svc.ListAgents(ctx, "usr_001")
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("期望 2 个 agents, 实际为 %d", len(agents))
	}

	for _, agent := range agents {
		if agent.UserCode() != "usr_001" {
			t.Errorf("期望 user_code 为 'usr_001', 实际为 '%s'", agent.UserCode())
		}
	}
}

func TestAgentService_UpdateAgent(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:    "usr_001",
		Name:        "OriginalName",
		Description: "OriginalDesc",
	})

	isActive := false
	updated, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:          created.ID(),
		Name:        strPtr("NewName"),
		Description: strPtr("NewDesc"),
		IsActive:    &isActive,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Name() != "NewName" {
		t.Errorf("期望 name 为 'NewName', 实际为 '%s'", updated.Name())
	}

	if updated.Description() != "NewDesc" {
		t.Errorf("期望 description 为 'NewDesc', 实际为 '%s'", updated.Description())
	}

	if updated.IsActive() {
		t.Error("agent 应该是非激活状态")
	}
}

func TestAgentService_UpdateAgent_NotFound(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	_, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:   domain.NewAgentID("non-existent"),
		Name: strPtr("NewName"),
	})
	if err != ErrAgentNotFound {
		t.Errorf("期望 ErrAgentNotFound, 实际为 %v", err)
	}
}

func TestAgentService_UpdateAgent_Config(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "ConfigTestAgent",
		Model:    "gpt-4",
	})

	updated, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:                    created.ID(),
		Model:                 strPtr("gpt-3.5"),
		MaxTokens:             intPtr(6000),
		Temperature:           float64Ptr(0.5),
		MaxIterations:         intPtr(10),
		HistoryMessages:       intPtr(15),
		EnableThinkingProcess: boolPtr(true),
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Model() != "gpt-3.5" {
		t.Errorf("期望 model 为 'gpt-3.5', 实际为 '%s'", updated.Model())
	}

	if updated.MaxTokens() != 6000 {
		t.Errorf("期望 max_tokens 为 6000, 实际为 %d", updated.MaxTokens())
	}
}

func TestAgentService_UpdateAgent_SetDefault(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "DefaultTestAgent",
	})

	isDefault := true
	updated, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:        created.ID(),
		IsDefault: &isDefault,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if !updated.IsDefault() {
		t.Error("agent 应该是默认状态")
	}
}

func TestAgentService_DeleteAgent(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "DeleteTestAgent",
	})

	err := svc.DeleteAgent(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	_, err = svc.GetAgent(ctx, created.ID())
	if err != ErrAgentNotFound {
		t.Errorf("期望 ErrAgentNotFound, 实际为 %v", err)
	}
}

func TestAgentService_DeleteAgent_NotFound(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	err := svc.DeleteAgent(ctx, domain.NewAgentID("non-existent"))
	if err != ErrAgentNotFound {
		t.Errorf("期望 ErrAgentNotFound, 实际为 %v", err)
	}
}

func TestApplyDefaultAgentCreateConfig(t *testing.T) {
	cmd := &CreateAgentCommand{
		Name:     "TestAgent",
		UserCode: "usr_001",
	}

	applyDefaultAgentCreateConfig(cmd)

	if cmd.Description != "默认 Agent" {
		t.Errorf("期望 description 为 '默认 Agent', 实际为 '%s'", cmd.Description)
	}

	if cmd.MaxTokens != 4096 {
		t.Errorf("期望 max_tokens 为 4096, 实际为 %d", cmd.MaxTokens)
	}

	if cmd.Temperature != 0.7 {
		t.Errorf("期望 temperature 为 0.7, 实际为 %f", cmd.Temperature)
	}

	if cmd.MaxIterations != 15 {
		t.Errorf("期望 max_iterations 为 15, 实际为 %d", cmd.MaxIterations)
	}

	if cmd.HistoryMessages != 10 {
		t.Errorf("期望 history_messages 为 10, 实际为 %d", cmd.HistoryMessages)
	}
}

func TestApplyDefaultAgentCreateConfig_PreservesProvidedValues(t *testing.T) {
	cmd := &CreateAgentCommand{
		Name:        "TestAgent",
		UserCode:    "usr_001",
		Description: "自定义描述",
		Model:       "gpt-3.5",
		MaxTokens:   8000,
	}

	applyDefaultAgentCreateConfig(cmd)

	if cmd.Description != "自定义描述" {
		t.Errorf("期望 description 为 '自定义描述', 实际为 '%s'", cmd.Description)
	}

	if cmd.Model != "gpt-3.5" {
		t.Errorf("期望 model 为 'gpt-3.5', 实际为 '%s'", cmd.Model)
	}

	if cmd.MaxTokens != 8000 {
		t.Errorf("期望 max_tokens 为 8000, 实际为 %d", cmd.MaxTokens)
	}
}

func TestBoolValue(t *testing.T) {
	trueVal := true
	falseVal := false

	if boolValue(nil, true) != true {
		t.Error("nil pointer 应该返回 fallback 值")
	}

	if boolValue(&trueVal, false) != true {
		t.Error("true pointer 应该返回 true")
	}

	if boolValue(&falseVal, true) != false {
		t.Error("false pointer 应该返回 false")
	}
}

func boolPtr(b bool) *bool   { return &b }
func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func float64Ptr(f float64) *float64 { return &f }

func TestAgentService_PatchAgent_WithClaudeCodeConfig(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "PatchTestAgent",
		Model:    "gpt-4",
	})

	// 获取初始配置
	initialTimeout := created.ClaudeCodeConfig().Timeout
	if initialTimeout != 120 {
		t.Errorf("期望初始 Timeout 为 120, 实际为 %d", initialTimeout)
	}

	// 初始 Model 是从环境或默认获取的
	initialModel := created.Model()

	// Patch 更新 Timeout
	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID: created.ID(),
		ClaudeCodeConfig: &domain.ClaudeCodeConfig{
			Timeout: 600,
		},
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if patched.ClaudeCodeConfig().Timeout != 600 {
		t.Errorf("期望 Patch 后 Timeout 为 600, 实际为 %d", patched.ClaudeCodeConfig().Timeout)
	}

	// 验证 Agent Model 保持不变
	if patched.Model() != initialModel {
		t.Errorf("期望 Model 保持为 %s, 实际为 %s", initialModel, patched.Model())
	}
}

func TestAgentService_PatchAgent_ClaudeCodeConfigMerge(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, err := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "MergeTestAgent",
	})
	if err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	// 先 Patch 一个完整配置
	if _, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID: created.ID(),
		ClaudeCodeConfig: &domain.ClaudeCodeConfig{
			Timeout:       600,
			Model:         "claude-3-5-sonnet",
			MaxThinkingTokens: 8000,
		},
	}); err != nil {
		t.Fatalf("首次 Patch 失败: %v", err)
	}

	// 再 Patch 只更新 Timeout
	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID: created.ID(),
		ClaudeCodeConfig: &domain.ClaudeCodeConfig{
			Timeout: 300,
		},
	})
	if err != nil {
		t.Fatalf("二次 Patch 失败: %v", err)
	}

	config := patched.ClaudeCodeConfig()
	// Timeout 应该被更新
	if config.Timeout != 300 {
		t.Errorf("期望 Timeout 为 300, 实际为 %d", config.Timeout)
	}
	// Model 应该保持之前的值
	if config.Model != "claude-3-5-sonnet" {
		t.Errorf("期望 Model 为 claude-3-5-sonnet, 实际为 %s", config.Model)
	}
	// MaxThinkingTokens 应该保持之前的值
	if config.MaxThinkingTokens != 8000 {
		t.Errorf("期望 MaxThinkingTokens 为 8000, 实际为 %d", config.MaxThinkingTokens)
	}
}

func TestAgentService_PatchAgent_NotFound(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	_, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID: domain.NewAgentID("non-existent"),
		ClaudeCodeConfig: &domain.ClaudeCodeConfig{
			Timeout: 600,
		},
	})

	if err != ErrAgentNotFound {
		t.Errorf("期望 ErrAgentNotFound, 实际为 %v", err)
	}
}

func TestAgentService_PatchAgent_OtherFieldsStillWork(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "OriginalName",
		Model:    "gpt-4",
	})

	// Patch 更新名称和 ClaudeCodeConfig
	newName := "PatchedName"
	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID:   created.ID(),
		Name: &newName,
		ClaudeCodeConfig: &domain.ClaudeCodeConfig{
			Timeout: 600,
		},
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if patched.Name() != "PatchedName" {
		t.Errorf("期望 Name 为 PatchedName, 实际为 %s", patched.Name())
	}

	if patched.ClaudeCodeConfig().Timeout != 600 {
		t.Errorf("期望 Timeout 为 600, 实际为 %d", patched.ClaudeCodeConfig().Timeout)
	}
}

// TestAgentService_CreateAgent_DuplicateCode 测试重复AgentCode检测
func TestAgentService_CreateAgent_DuplicateCode(t *testing.T) {
	repo := newMockAgentRepo()
	idGen := &mockAgentIDGen{}
	svc := NewAgentApplicationService(repo, idGen)
	ctx := context.Background()

	// 首先手动添加一个agent到repo，使用idGen将生成的第一个code
	agentCode := domain.NewAgentCode("agt_agent-id-1")
	existingAgent, err := domain.NewAgent(
		domain.NewAgentID("existing-id"),
		agentCode,
		"usr_002",
		"ExistingAgent",
		"描述",
		domain.AgentTypeBareLLM,
	)
	if err != nil {
		t.Fatalf("创建 existingAgent 失败: %v", err)
	}
	if err := repo.Save(ctx, existingAgent); err != nil {
		t.Fatalf("保存 existingAgent 失败: %v", err)
	}

	// 尝试创建agent，应该检测到重复的AgentCode
	_, err = svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "Agent1",
	})

	if err != ErrAgentCodeDuplicated {
		t.Errorf("期望 ErrAgentCodeDuplicated, 实际为 %v", err)
	}
}

// TestAgentService_PatchAgent_SetAgentType 测试PatchAgent更新AgentType
func TestAgentService_PatchAgent_SetAgentType(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:  "usr_001",
		Name:      "TestAgent",
		AgentType: string(domain.AgentTypeBareLLM),
	})

	newType := string(domain.AgentTypeCoding)
	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID:        created.ID(),
		AgentType: &newType,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if string(patched.AgentType()) != string(domain.AgentTypeCoding) {
		t.Errorf("期望 AgentType 为 Coding, 实际为 %s", patched.AgentType())
	}
}

// TestAgentService_PatchAgent_SetIsActive 测试PatchAgent更新IsActive
func TestAgentService_PatchAgent_SetIsActive(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, err := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "TestAgent",
	})
	if err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	// 初始为激活状态
	if !created.IsActive() {
		t.Error("新创建的agent应该是激活状态")
	}

	// 更新为非激活
	isActive := false
	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID:       created.ID(),
		IsActive: &isActive,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if patched.IsActive() {
		t.Error("Patch后agent应该是非激活状态")
	}

	// 再更新为激活
	isActive = true
	patched2, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID:       patched.ID(),
		IsActive: &isActive,
	})
	if err != nil {
		t.Fatalf("再次 Patch 失败: %v", err)
	}

	if !patched2.IsActive() {
		t.Error("再次Patch后agent应该是激活状态")
	}
}

// TestAgentService_PatchAgent_SetIsDefault 测试PatchAgent更新IsDefault
func TestAgentService_PatchAgent_SetIsDefault(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "TestAgent",
	})

	// 初始为非默认
	if created.IsDefault() {
		t.Error("新创建的agent默认不应是默认状态")
	}

	// 更新为默认
	isDefault := true
	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID:        created.ID(),
		IsDefault: &isDefault,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if !patched.IsDefault() {
		t.Error("Patch后agent应该是默认状态")
	}
}

// TestAgentService_PatchAgent_UpdateConfig 测试PatchAgent更新配置字段
func TestAgentService_PatchAgent_UpdateConfig(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "TestAgent",
		Model:    "gpt-4",
	})

	// Patch更新多个配置字段
	newModel := "claude-3"
	newMaxTokens := 8000
	newTemp := 0.9
	newMaxIter := 20
	newHistoryMsg := 25
	newIdentity := "新身份"
	newSoul := "新灵魂"
	newAgents := "新agents内容"
	newUser := "新user内容"
	newTools := "新tools内容"
	enableThinking := true

	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID:                    created.ID(),
		Model:                 &newModel,
		MaxTokens:             &newMaxTokens,
		Temperature:           &newTemp,
		MaxIterations:         &newMaxIter,
		HistoryMessages:       &newHistoryMsg,
		IdentityContent:       &newIdentity,
		SoulContent:           &newSoul,
		AgentsContent:         &newAgents,
		UserContent:           &newUser,
		ToolsContent:          &newTools,
		EnableThinkingProcess: &enableThinking,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if patched.Model() != "claude-3" {
		t.Errorf("期望 Model 为 claude-3, 实际为 %s", patched.Model())
	}
	if patched.MaxTokens() != 8000 {
		t.Errorf("期望 MaxTokens 为 8000, 实际为 %d", patched.MaxTokens())
	}
	if patched.Temperature() != 0.9 {
		t.Errorf("期望 Temperature 为 0.9, 实际为 %f", patched.Temperature())
	}
	if patched.MaxIterations() != 20 {
		t.Errorf("期望 MaxIterations 为 20, 实际为 %d", patched.MaxIterations())
	}
	if patched.HistoryMessages() != 25 {
		t.Errorf("期望 HistoryMessages 为 25, 实际为 %d", patched.HistoryMessages())
	}
	if patched.IdentityContent() != "新身份" {
		t.Errorf("期望 IdentityContent 为 '新身份', 实际为 %s", patched.IdentityContent())
	}
	if patched.SoulContent() != "新灵魂" {
		t.Errorf("期望 SoulContent 为 '新灵魂', 实际为 %s", patched.SoulContent())
	}
	if patched.AgentsContent() != "新agents内容" {
		t.Errorf("期望 AgentsContent 为 '新agents内容', 实际为 %s", patched.AgentsContent())
	}
	if patched.UserContent() != "新user内容" {
		t.Errorf("期望 UserContent 为 '新user内容', 实际为 %s", patched.UserContent())
	}
	if patched.ToolsContent() != "新tools内容" {
		t.Errorf("期望 ToolsContent 为 '新tools内容', 实际为 %s", patched.ToolsContent())
	}
	if !patched.EnableThinkingProcess() {
		t.Error("期望 EnableThinkingProcess 为 true")
	}
}

// TestAgentService_PatchAgent_SkillsAndTools 测试PatchAgent更新技能和工具列表
func TestAgentService_PatchAgent_SkillsAndTools(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:   "usr_001",
		Name:       "TestAgent",
		SkillsList: []string{},
		ToolsList:  []string{},
	})

	// Patch更新技能和工具列表
	newSkills := []string{"skill1", "skill2", "skill3"}
	newTools := []string{"tool1", "tool2"}

	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID:         created.ID(),
		SkillsList: &newSkills,
		ToolsList:  &newTools,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(patched.SkillsList()) != 3 {
		t.Errorf("期望 SkillsList 长度为 3, 实际为 %d", len(patched.SkillsList()))
	}
	if len(patched.ToolsList()) != 2 {
		t.Errorf("期望 ToolsList 长度为 2, 实际为 %d", len(patched.ToolsList()))
	}
}

// TestAgentService_PatchAgent_ApplyLLMProvider 测试PatchAgent应用LLMProvider
func TestAgentService_PatchAgent_ApplyLLMProvider(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "TestAgent",
	})

	// 应用LLMProvider
	providerID := "provider-123"
	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID:            created.ID(),
		LLMProviderID: &providerID,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if patched.LLMProviderID().String() != "provider-123" {
		t.Errorf("期望 LLMProviderID 为 provider-123, 实际为 %s", patched.LLMProviderID().String())
	}
}

// TestAgentService_UpdateAgent_PartialFields 测试UpdateAgent部分字段更新
func TestAgentService_UpdateAgent_PartialFields(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:    "usr_001",
		Name:        "OriginalName",
		Description: "OriginalDesc",
	})

	// 只更新Name，Description应该保持不变
	updated, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:   created.ID(),
		Name: strPtr("NewName"),
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Name() != "NewName" {
		t.Errorf("期望 Name 为 NewName, 实际为 %s", updated.Name())
	}
	// Description应该保持不变（通过读取原始值填充）
	if updated.Description() != "OriginalDesc" {
		t.Errorf("期望 Description 保持为 OriginalDesc, 实际为 %s", updated.Description())
	}
}

// TestAgentService_UpdateAgent_OnlyDescription 测试UpdateAgent只更新Description
func TestAgentService_UpdateAgent_OnlyDescription(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:    "usr_001",
		Name:        "OriginalName",
		Description: "OriginalDesc",
	})

	// 只更新Description，Name应该保持不变
	updated, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:          created.ID(),
		Description: strPtr("NewDesc"),
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Name() != "OriginalName" {
		t.Errorf("期望 Name 保持为 OriginalName, 实际为 %s", updated.Name())
	}
	if updated.Description() != "NewDesc" {
		t.Errorf("期望 Description 为 NewDesc, 实际为 %s", updated.Description())
	}
}

// TestAgentService_UpdateAgent_SetAgentType 测试UpdateAgent更新AgentType
func TestAgentService_UpdateAgent_SetAgentType(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:  "usr_001",
		Name:      "TestAgent",
		AgentType: string(domain.AgentTypeBareLLM),
	})

	newType := string(domain.AgentTypeCoding)
	updated, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:        created.ID(),
		AgentType: &newType,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if string(updated.AgentType()) != string(domain.AgentTypeCoding) {
		t.Errorf("期望 AgentType 为 Coding, 实际为 %s", updated.AgentType())
	}
}

// TestAgentService_UpdateAgent_SkillsAndTools 测试UpdateAgent更新技能和工具列表
func TestAgentService_UpdateAgent_SkillsAndTools(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:   "usr_001",
		Name:       "TestAgent",
		SkillsList: []string{"skill1"},
		ToolsList:  []string{"tool1"},
	})

	newSkills := []string{"skill2", "skill3"}
	newTools := []string{"tool2", "tool3", "tool4"}

	updated, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:         created.ID(),
		SkillsList: &newSkills,
		ToolsList:  &newTools,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(updated.SkillsList()) != 2 {
		t.Errorf("期望 SkillsList 长度为 2, 实际为 %d", len(updated.SkillsList()))
	}
	if len(updated.ToolsList()) != 3 {
		t.Errorf("期望 ToolsList 长度为 3, 实际为 %d", len(updated.ToolsList()))
	}
}

// TestAgentService_CreateAgent_WithLLMProvider 测试创建Agent时设置LLMProvider
func TestAgentService_CreateAgent_WithLLMProvider(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	providerID := "provider-abc"
	agent, err := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:      "usr_001",
		Name:          "TestAgent",
		LLMProviderID: &providerID,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if agent.LLMProviderID().String() != "provider-abc" {
		t.Errorf("期望 LLMProviderID 为 provider-abc, 实际为 %s", agent.LLMProviderID().String())
	}
}

// TestApplyDefaultAgentCreateConfig_AllDefaults 测试applyDefaultAgentCreateConfig所有默认值
func TestApplyDefaultAgentCreateConfig_AllDefaults(t *testing.T) {
	cmd := &CreateAgentCommand{
		Name:     "TestAgent",
		UserCode: "usr_001",
	}

	applyDefaultAgentCreateConfig(cmd)

	// 验证所有默认内容字段
	if cmd.IdentityContent != domain.DefaultIdentityContent {
		t.Errorf("期望 IdentityContent 为默认值, 实际为 '%s'", cmd.IdentityContent)
	}
	if cmd.SoulContent != domain.DefaultSoulContent {
		t.Errorf("期望 SoulContent 为默认值, 实际为 '%s'", cmd.SoulContent)
	}
	if cmd.AgentsContent != domain.DefaultAgentsContent {
		t.Errorf("期望 AgentsContent 为默认值, 实际为 '%s'", cmd.AgentsContent)
	}
	if cmd.UserContent != domain.DefaultUserContent {
		t.Errorf("期望 UserContent 为默认值, 实际为 '%s'", cmd.UserContent)
	}
	if cmd.ToolsContent != domain.DefaultToolsContent {
		t.Errorf("期望 ToolsContent 为默认值, 实际为 '%s'", cmd.ToolsContent)
	}
	// 验证 SkillsList 和 ToolsList 被初始化为空数组
	if cmd.SkillsList == nil {
		t.Error("期望 SkillsList 被初始化为空数组, 实际为 nil")
	}
	if cmd.ToolsList == nil {
		t.Error("期望 ToolsList 被初始化为空数组, 实际为 nil")
	}
}

// TestApplyDefaultAgentCreateConfig_EmptyWhitespace 测试空白字符串被正确处理
func TestApplyDefaultAgentCreateConfig_EmptyWhitespace(t *testing.T) {
	cmd := &CreateAgentCommand{
		Name:            "TestAgent",
		UserCode:        "usr_001",
		Description:     "   ",   // 空白字符串
		IdentityContent: "\t\n", // 制表符和换行
		Model:           "  ",    // 空白
	}

	applyDefaultAgentCreateConfig(cmd)

	// 空白字符串应该被替换为默认值
	if cmd.Description != domain.DefaultAgentDescription {
		t.Errorf("期望空白 Description 被替换为默认值, 实际为 '%s'", cmd.Description)
	}
	if cmd.IdentityContent != domain.DefaultIdentityContent {
		t.Errorf("期望空白 IdentityContent 被替换为默认值, 实际为 '%s'", cmd.IdentityContent)
	}
	// Model应该根据环境变量或默认值设置
	// 注意：这里取决于环境变量设置，不直接断言具体值
	if strings.TrimSpace(cmd.Model) == "" {
		t.Error("期望空白 Model 被替换为非空值")
	}
}

// TestApplyDefaultAgentCreateConfig_ZeroValues 测试零值被正确处理
func TestApplyDefaultAgentCreateConfig_ZeroValues(t *testing.T) {
	cmd := &CreateAgentCommand{
		Name:            "TestAgent",
		UserCode:        "usr_001",
		MaxTokens:       0,
		Temperature:     0,
		MaxIterations:   0,
		HistoryMessages: 0,
	}

	applyDefaultAgentCreateConfig(cmd)

	// 零值应该被替换为默认值
	if cmd.MaxTokens != domain.DefaultMaxTokens {
		t.Errorf("期望 MaxTokens 为 %d, 实际为 %d", domain.DefaultMaxTokens, cmd.MaxTokens)
	}
	if cmd.Temperature != domain.DefaultTemperature {
		t.Errorf("期望 Temperature 为 %f, 实际为 %f", domain.DefaultTemperature, cmd.Temperature)
	}
	if cmd.MaxIterations != domain.DefaultMaxIterations {
		t.Errorf("期望 MaxIterations 为 %d, 实际为 %d", domain.DefaultMaxIterations, cmd.MaxIterations)
	}
	if cmd.HistoryMessages != domain.DefaultHistoryMessages {
		t.Errorf("期望 HistoryMessages 为 %d, 实际为 %d", domain.DefaultHistoryMessages, cmd.HistoryMessages)
	}
}

// TestAgentService_PatchAgent_OnlyDescription 测试PatchAgent只更新Description
func TestAgentService_PatchAgent_OnlyDescription(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:    "usr_001",
		Name:        "OriginalName",
		Description: "OriginalDesc",
	})

	// 只更新Description
	newDesc := "NewDescription"
	patched, err := svc.PatchAgent(ctx, PatchAgentCommand{
		ID:          created.ID(),
		Description: &newDesc,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if patched.Description() != "NewDescription" {
		t.Errorf("期望 Description 为 NewDescription, 实际为 %s", patched.Description())
	}
	// Name应该保持不变
	if patched.Name() != "OriginalName" {
		t.Errorf("期望 Name 保持为 OriginalName, 实际为 %s", patched.Name())
	}
}

// TestAgentService_UpdateAgent_ApplyLLMProvider 测试UpdateAgent应用LLMProvider
func TestAgentService_UpdateAgent_ApplyLLMProvider(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "TestAgent",
	})

	// 初始没有LLMProvider
	if created.LLMProviderID().String() != "" {
		t.Error("新创建的agent应该没有LLMProvider")
	}

	// 应用LLMProvider
	providerID := "provider-456"
	updated, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:            created.ID(),
		LLMProviderID: &providerID,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.LLMProviderID().String() != "provider-456" {
		t.Errorf("期望 LLMProviderID 为 provider-456, 实际为 %s", updated.LLMProviderID().String())
	}
}

// TestAgentService_UpdateAgent_EnableThinkingProcess 测试UpdateAgent更新EnableThinkingProcess
func TestAgentService_UpdateAgent_EnableThinkingProcess(t *testing.T) {
	svc := setupTestAgentSvc()
	ctx := context.Background()

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode:              "usr_001",
		Name:                  "TestAgent",
		EnableThinkingProcess: false,
	})

	// 初始为false
	if created.EnableThinkingProcess() {
		t.Error("新创建的agent EnableThinkingProcess 初始应为 false")
	}

	// 更新为true
	enable := true
	updated, err := svc.UpdateAgent(ctx, UpdateAgentCommand{
		ID:                    created.ID(),
		EnableThinkingProcess: &enable,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if !updated.EnableThinkingProcess() {
		t.Error("更新后 EnableThinkingProcess 应为 true")
	}
}
