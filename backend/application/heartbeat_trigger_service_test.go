package application

import (
	"context"
	"strings"
	"testing"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/statemachine"
)

func TestHeartbeatTriggerService_Trigger_Success(t *testing.T) {
	project, _ := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"测试项目",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)
	channelCode := "channel-001"
	sessionKey := "session-001"
	project.UpdateDispatchConfig(&channelCode, &sessionKey)

	hb, _ := domain.NewHeartbeat(
		domain.NewHeartbeatID("hb-001"),
		project.ID(),
		"测试心跳",
		5,
		"心跳内容: ${project.name}",
		"agent-001",
		"heartbeat",
	)

	hbRepo := &MockHeartbeatRepository{heartbeats: []*domain.Heartbeat{hb}}
	projectRepo := &MockProjectRepository{projects: []*domain.Project{project}}
	agentRepo := &MockAgentRepository{}
	reqRepo := &MockRequirementRepository{requirements: []*domain.Requirement{}}
	idGen := &MockIDGenerator{id: "req-trigger-001"}

	service := NewHeartbeatTriggerService(
		hbRepo,
		projectRepo,
		agentRepo,
		reqRepo,
		idGen,
		nil,
		nil,
		nil,
	)

	// triggerService 没有 requirementDispatchService，所以会返回错误
	err := service.Trigger(context.Background(), "hb-001")
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("期望返回 dispatch service not available 错误，实际: %v", err)
	}

	// 但需求应该已经被创建
	if len(reqRepo.requirements) != 1 {
		t.Fatalf("期望创建1个需求，实际创建了 %d 个", len(reqRepo.requirements))
	}

	req := reqRepo.requirements[0]
	if !strings.Contains(req.Title(), "[心跳]") {
		t.Fatalf("期望需求标题包含 [心跳]，实际为: %s", req.Title())
	}
	if req.RequirementType() != domain.RequirementTypeHeartbeat {
		t.Fatalf("期望需求类型为 heartbeat，实际为: %s", req.RequirementType())
	}
	if !strings.Contains(req.Description(), "测试项目") {
		t.Fatalf("期望需求描述包含项目名称，实际为: %s", req.Description())
	}
}

func TestHeartbeatTriggerService_Trigger_HeartbeatNotFound(t *testing.T) {
	hbRepo := &MockHeartbeatRepository{heartbeats: []*domain.Heartbeat{}}
	projectRepo := &MockProjectRepository{}
	reqRepo := &MockRequirementRepository{}

	service := NewHeartbeatTriggerService(
		hbRepo,
		projectRepo,
		nil,
		reqRepo,
		&MockIDGenerator{},
		nil,
		nil,
		nil,
	)

	err := service.Trigger(context.Background(), "hb-not-exist")
	if err == nil || !strings.Contains(err.Error(), "failed to find heartbeat") {
		t.Fatalf("期望返回找不到心跳错误，实际: %v", err)
	}

	if len(reqRepo.requirements) != 0 {
		t.Fatalf("期望没有创建需求，实际创建了 %d 个", len(reqRepo.requirements))
	}
}

func TestHeartbeatTriggerService_Trigger_HeartbeatDisabled(t *testing.T) {
	project, _ := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"测试项目",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)

	hb, _ := domain.NewHeartbeat(
		domain.NewHeartbeatID("hb-001"),
		project.ID(),
		"测试心跳",
		5,
		"内容",
		"agent-001",
		"heartbeat",
	)
	hb.SetEnabled(false)

	hbRepo := &MockHeartbeatRepository{heartbeats: []*domain.Heartbeat{hb}}
	projectRepo := &MockProjectRepository{projects: []*domain.Project{project}}
	reqRepo := &MockRequirementRepository{}

	service := NewHeartbeatTriggerService(
		hbRepo,
		projectRepo,
		nil,
		reqRepo,
		&MockIDGenerator{},
		nil,
		nil,
		nil,
	)

	err := service.Trigger(context.Background(), "hb-001")
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("期望返回心跳已禁用错误，实际: %v", err)
	}

	if len(reqRepo.requirements) != 0 {
		t.Fatalf("期望没有创建需求，实际创建了 %d 个", len(reqRepo.requirements))
	}
}

func TestHeartbeatTriggerService_Trigger_ProjectNotFound(t *testing.T) {
	project, _ := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"测试项目",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)

	hb, _ := domain.NewHeartbeat(
		domain.NewHeartbeatID("hb-001"),
		project.ID(),
		"测试心跳",
		5,
		"内容",
		"agent-001",
		"heartbeat",
	)

	hbRepo := &MockHeartbeatRepository{heartbeats: []*domain.Heartbeat{hb}}
	projectRepo := &MockProjectRepository{projects: []*domain.Project{}} // 空项目列表
	reqRepo := &MockRequirementRepository{}

	service := NewHeartbeatTriggerService(
		hbRepo,
		projectRepo,
		nil,
		reqRepo,
		&MockIDGenerator{},
		nil,
		nil,
		nil,
	)

	err := service.Trigger(context.Background(), "hb-001")
	if err == nil || !strings.Contains(err.Error(), "failed to find project") {
		t.Fatalf("期望返回找不到项目错误，实际: %v", err)
	}

	if len(reqRepo.requirements) != 0 {
		t.Fatalf("期望没有创建需求，实际创建了 %d 个", len(reqRepo.requirements))
	}
}

func TestHeartbeatTriggerService_Trigger_WithStateMachine(t *testing.T) {
	project, _ := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"测试项目",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)
	channelCode := "channel-001"
	sessionKey := "session-001"
	project.UpdateDispatchConfig(&channelCode, &sessionKey)

	hb, _ := domain.NewHeartbeat(
		domain.NewHeartbeatID("hb-001"),
		project.ID(),
		"测试心跳",
		5,
		"内容",
		"agent-001",
		"feature",
	)

	hbRepo := &MockHeartbeatRepository{heartbeats: []*domain.Heartbeat{hb}}
	projectRepo := &MockProjectRepository{projects: []*domain.Project{project}}
	agentRepo := &MockAgentRepository{}
	reqRepo := &MockRequirementRepository{}
	idGen := &MockIDGenerator{id: "req-sm-001"}

	cfg := &statemachine.Config{
		InitialState: "todo",
		States: []statemachine.State{
			{ID: "todo", Name: "待办"},
			{ID: "completed", Name: "已完成", IsFinal: true},
		},
		Transitions: []statemachine.Transition{
			{FromState: "todo", ToState: "completed", Trigger: "finish"},
		},
	}
	sm := statemachine.NewStateMachine("测试状态机", "", cfg)

	psm, _ := statemachine.NewProjectStateMachine("proj-001", statemachine.RequirementType("feature"), sm.ID)

	mockSMRepo := &mockStateMachineRepo{
		stateMachines: map[string]*statemachine.StateMachine{
			sm.ID: sm,
		},
		projectStateMachines: map[string]*statemachine.ProjectStateMachine{
			"proj-001:feature": psm,
		},
	}

	stateMachineService := NewStateMachineService(mockSMRepo, reqRepo, nil, nil)

	service := NewHeartbeatTriggerService(
		hbRepo,
		projectRepo,
		agentRepo,
		reqRepo,
		idGen,
		nil,
		nil,
		stateMachineService,
	)

	// 没有 dispatch service，会返回错误
	_ = service.Trigger(context.Background(), "hb-001")

	// 验证需求被创建且需求类型正确
	if len(reqRepo.requirements) != 1 {
		t.Fatalf("期望创建1个需求，实际创建了 %d 个", len(reqRepo.requirements))
	}

	req := reqRepo.requirements[0]
	if req.RequirementType() != domain.RequirementType("feature") {
		t.Fatalf("期望需求类型为 feature，实际为: %s", req.RequirementType())
	}
}

// mockStateMachineRepo 模拟状态机仓库
type mockStateMachineRepo struct {
	stateMachines        map[string]*statemachine.StateMachine
	requirementStates    map[string]*statemachine.RequirementState
	projectStateMachines map[string]*statemachine.ProjectStateMachine
}

func (m *mockStateMachineRepo) SaveStateMachine(ctx context.Context, sm *statemachine.StateMachine) error {
	if m.stateMachines == nil {
		m.stateMachines = make(map[string]*statemachine.StateMachine)
	}
	m.stateMachines[sm.ID] = sm
	return nil
}

func (m *mockStateMachineRepo) GetStateMachine(ctx context.Context, id string) (*statemachine.StateMachine, error) {
	return m.stateMachines[id], nil
}

func (m *mockStateMachineRepo) ListStateMachines(ctx context.Context) ([]*statemachine.StateMachine, error) {
	var result []*statemachine.StateMachine
	for _, sm := range m.stateMachines {
		result = append(result, sm)
	}
	return result, nil
}

func (m *mockStateMachineRepo) DeleteStateMachine(ctx context.Context, id string) error {
	delete(m.stateMachines, id)
	return nil
}

func (m *mockStateMachineRepo) SaveRequirementState(ctx context.Context, rs *statemachine.RequirementState) error {
	if m.requirementStates == nil {
		m.requirementStates = make(map[string]*statemachine.RequirementState)
	}
	m.requirementStates[rs.RequirementID] = rs
	return nil
}

func (m *mockStateMachineRepo) GetRequirementState(ctx context.Context, requirementID string) (*statemachine.RequirementState, error) {
	return m.requirementStates[requirementID], nil
}

func (m *mockStateMachineRepo) UpdateRequirementState(ctx context.Context, rs *statemachine.RequirementState) error {
	if m.requirementStates == nil {
		m.requirementStates = make(map[string]*statemachine.RequirementState)
	}
	m.requirementStates[rs.RequirementID] = rs
	return nil
}

func (m *mockStateMachineRepo) SaveTransitionLog(ctx context.Context, log *statemachine.TransitionLog) error {
	return nil
}

func (m *mockStateMachineRepo) ListTransitionLogs(ctx context.Context, requirementID string) ([]*statemachine.TransitionLog, error) {
	return nil, nil
}

func (m *mockStateMachineRepo) SaveProjectStateMachine(ctx context.Context, psm *statemachine.ProjectStateMachine) error {
	if m.projectStateMachines == nil {
		m.projectStateMachines = make(map[string]*statemachine.ProjectStateMachine)
	}
	key := psm.ProjectID() + ":" + string(psm.RequirementType())
	m.projectStateMachines[key] = psm
	return nil
}

func (m *mockStateMachineRepo) GetProjectStateMachine(ctx context.Context, projectID string, requirementType statemachine.RequirementType) (*statemachine.ProjectStateMachine, error) {
	if m.projectStateMachines == nil {
		return nil, nil
	}
	key := projectID + ":" + string(requirementType)
	return m.projectStateMachines[key], nil
}

func (m *mockStateMachineRepo) ListProjectStateMachines(ctx context.Context, projectID string) ([]*statemachine.ProjectStateMachine, error) {
	var result []*statemachine.ProjectStateMachine
	for _, psm := range m.projectStateMachines {
		if psm.ProjectID() == projectID {
			result = append(result, psm)
		}
	}
	return result, nil
}

func (m *mockStateMachineRepo) DeleteProjectStateMachine(ctx context.Context, id string) error {
	for key, psm := range m.projectStateMachines {
		if psm.ID() == id {
			delete(m.projectStateMachines, key)
			break
		}
	}
	return nil
}

func (m *mockStateMachineRepo) DeleteProjectStateMachinesByProject(ctx context.Context, projectID string) error {
	for key, psm := range m.projectStateMachines {
		if psm.ProjectID() == projectID {
			delete(m.projectStateMachines, key)
		}
	}
	return nil
}
