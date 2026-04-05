package statemachine

import (
	"context"
	"testing"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain/state_machine"
	infra_sm "github.com/weibh/taskmanager/infrastructure/state_machine"
	"go.uber.org/zap"
)

// MockRepository 测试用 Mock Repository
type MockRepository struct {
	stateMachines      map[string]*state_machine.StateMachine
	requirementStates map[string]*state_machine.RequirementState
	transitionLogs    []*state_machine.TransitionLog
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		stateMachines:      make(map[string]*state_machine.StateMachine),
		requirementStates: make(map[string]*state_machine.RequirementState),
		transitionLogs:    []*state_machine.TransitionLog{},
	}
}

func (r *MockRepository) SaveStateMachine(ctx context.Context, sm *state_machine.StateMachine) error {
	r.stateMachines[sm.ID] = sm
	return nil
}

func (r *MockRepository) GetStateMachine(ctx context.Context, id string) (*state_machine.StateMachine, error) {
	sm, ok := r.stateMachines[id]
	if !ok {
		return nil, state_machine.ErrStateMachineNotFound(id)
	}
	return sm, nil
}

func (r *MockRepository) ListStateMachines(ctx context.Context) ([]*state_machine.StateMachine, error) {
	var result []*state_machine.StateMachine
	for _, sm := range r.stateMachines {
		result = append(result, sm)
	}
	return result, nil
}

func (r *MockRepository) DeleteStateMachine(ctx context.Context, id string) error {
	delete(r.stateMachines, id)
	return nil
}

func (r *MockRepository) SaveRequirementState(ctx context.Context, rs *state_machine.RequirementState) error {
	r.requirementStates[rs.RequirementID] = rs
	return nil
}

func (r *MockRepository) GetRequirementState(ctx context.Context, requirementID string) (*state_machine.RequirementState, error) {
	rs, ok := r.requirementStates[requirementID]
	if !ok {
		return nil, state_machine.ErrRequirementStateNotFound(requirementID)
	}
	return rs, nil
}

func (r *MockRepository) UpdateRequirementState(ctx context.Context, rs *state_machine.RequirementState) error {
	r.requirementStates[rs.RequirementID] = rs
	return nil
}

func (r *MockRepository) SaveTransitionLog(ctx context.Context, log *state_machine.TransitionLog) error {
	r.transitionLogs = append(r.transitionLogs, log)
	return nil
}

func (r *MockRepository) ListTransitionLogs(ctx context.Context, requirementID string) ([]*state_machine.TransitionLog, error) {
	var result []*state_machine.TransitionLog
	for _, log := range r.transitionLogs {
		if log.RequirementID == requirementID {
			result = append(result, log)
		}
	}
	return result, nil
}

const testYAML = `
name: 测试流程
description: SDK 测试用流程

initial_state: created

states:
  - id: created
    name: 已创建
    is_final: false
  - id: in_progress
    name: 进行中
    is_final: false
  - id: done
    name: 已完成
    is_final: true

transitions:
  - from: created
    to: in_progress
    trigger: start
    description: 开始
  - from: in_progress
    to: done
    trigger: complete
    description: 完成
`

func TestSDK(t *testing.T) {
	// 创建 Mock 服务
	logger := zap.NewNop()
	repo := NewMockRepository()
	executor := infra_sm.NewTransitionExecutor(logger)
	svc := application.NewStateMachineService(repo, executor, logger)

	// 使用 WithService 注入
	sm := New(WithService(svc))
	defer sm.Close()

	ctx := context.Background()

	// 创建状态机
	t.Log("创建状态机...")
	machine, err := sm.Create(ctx, "测试流程", "测试", testYAML)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	t.Logf("状态机创建成功: ID=%s", machine.ID)

	// 获取状态机
	t.Log("获取状态机...")
	got, err := sm.Get(ctx, machine.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != machine.ID {
		t.Errorf("Get returned wrong ID: %s != %s", got.ID, machine.ID)
	}

	// 列出状态机
	t.Log("列出状态机...")
	list, err := sm.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List returned %d items, want 1", len(list))
	}

	// 初始化需求
	t.Log("初始化需求...")
	reqID := "test-req-001"
	rs, err := sm.Initialize(ctx, reqID, machine.ID)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if rs.CurrentState != "created" {
		t.Errorf("Initial state: %s != created", rs.CurrentState)
	}

	// 触发转换 start
	t.Log("触发转换 start...")
	rs, err = sm.Transition(ctx, reqID, "start", "tester", "开始测试")
	if err != nil {
		t.Fatalf("Transition(start) failed: %v", err)
	}
	if rs.CurrentState != "in_progress" {
		t.Errorf("State after start: %s != in_progress", rs.CurrentState)
	}

	// 触发转换 complete
	t.Log("触发转换 complete...")
	rs, err = sm.Transition(ctx, reqID, "complete", "tester", "完成测试")
	if err != nil {
		t.Fatalf("Transition(complete) failed: %v", err)
	}
	if rs.CurrentState != "done" {
		t.Errorf("State after complete: %s != done", rs.CurrentState)
	}

	// 获取状态
	t.Log("获取状态...")
	state, err := sm.GetState(ctx, reqID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state.CurrentState != "done" {
		t.Errorf("GetState state: %s != done", state.CurrentState)
	}

	// 获取历史
	t.Log("获取历史...")
	history, err := sm.GetHistory(ctx, reqID)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	// init + start + complete = 3
	if len(history) != 3 {
		t.Errorf("History count: %d != 3", len(history))
	}

	// 删除状态机
	t.Log("删除状态机...")
	err = sm.Delete(ctx, machine.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	t.Log("测试完成!")
}
