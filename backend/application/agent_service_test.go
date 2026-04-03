package application

import (
	"context"
	"strconv"
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

	created, _ := svc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "MergeTestAgent",
	})

	// 先 Patch 一个完整配置
	svc.PatchAgent(ctx, PatchAgentCommand{
		ID: created.ID(),
		ClaudeCodeConfig: &domain.ClaudeCodeConfig{
			Timeout:       600,
			Model:         "claude-3-5-sonnet",
			MaxThinkingTokens: 8000,
		},
	})

	// 再 Patch 只更新 Timeout
	patched, _ := svc.PatchAgent(ctx, PatchAgentCommand{
		ID: created.ID(),
		ClaudeCodeConfig: &domain.ClaudeCodeConfig{
			Timeout: 300,
		},
	})

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
