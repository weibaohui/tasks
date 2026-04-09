package application

import (
	"context"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain/state_machine"
	infra_sm "github.com/weibh/taskmanager/infrastructure/state_machine"
	"go.uber.org/zap"
)

// MockStateMachineRepository Mock 仓储
type MockStateMachineRepository struct {
	stateMachines        map[string]*state_machine.StateMachine
	requirementStates    map[string]*state_machine.RequirementState
	transitionLogs       []*state_machine.TransitionLog
	projectStateMachines map[string]*state_machine.ProjectStateMachine
}

func NewMockStateMachineRepository() *MockStateMachineRepository {
	return &MockStateMachineRepository{
		stateMachines:        make(map[string]*state_machine.StateMachine),
		requirementStates:    make(map[string]*state_machine.RequirementState),
		transitionLogs:       []*state_machine.TransitionLog{},
		projectStateMachines: make(map[string]*state_machine.ProjectStateMachine),
	}
}

func (r *MockStateMachineRepository) SaveStateMachine(ctx context.Context, sm *state_machine.StateMachine) error {
	r.stateMachines[sm.ID] = sm
	return nil
}

func (r *MockStateMachineRepository) GetStateMachine(ctx context.Context, id string) (*state_machine.StateMachine, error) {
	sm, ok := r.stateMachines[id]
	if !ok {
		return nil, state_machine.ErrStateMachineNotFound(id)
	}
	return sm, nil
}

func (r *MockStateMachineRepository) ListStateMachines(ctx context.Context) ([]*state_machine.StateMachine, error) {
	var result []*state_machine.StateMachine
	for _, sm := range r.stateMachines {
		result = append(result, sm)
	}
	return result, nil
}

func (r *MockStateMachineRepository) DeleteStateMachine(ctx context.Context, id string) error {
	delete(r.stateMachines, id)
	return nil
}

func (r *MockStateMachineRepository) SaveRequirementState(ctx context.Context, rs *state_machine.RequirementState) error {
	r.requirementStates[rs.RequirementID] = rs
	return nil
}

func (r *MockStateMachineRepository) GetRequirementState(ctx context.Context, requirementID string) (*state_machine.RequirementState, error) {
	rs, ok := r.requirementStates[requirementID]
	if !ok {
		return nil, state_machine.ErrRequirementStateNotFound(requirementID)
	}
	return rs, nil
}

func (r *MockStateMachineRepository) UpdateRequirementState(ctx context.Context, rs *state_machine.RequirementState) error {
	r.requirementStates[rs.RequirementID] = rs
	return nil
}

func (r *MockStateMachineRepository) SaveTransitionLog(ctx context.Context, log *state_machine.TransitionLog) error {
	r.transitionLogs = append(r.transitionLogs, log)
	return nil
}

func (r *MockStateMachineRepository) ListTransitionLogs(ctx context.Context, requirementID string) ([]*state_machine.TransitionLog, error) {
	var result []*state_machine.TransitionLog
	for _, log := range r.transitionLogs {
		if log.RequirementID == requirementID {
			result = append(result, log)
		}
	}
	return result, nil
}

func (r *MockStateMachineRepository) SaveProjectStateMachine(ctx context.Context, psm *state_machine.ProjectStateMachine) error {
	key := psm.ProjectID() + "_" + string(psm.RequirementType())
	r.projectStateMachines[key] = psm
	return nil
}

func (r *MockStateMachineRepository) GetProjectStateMachine(ctx context.Context, projectID string, requirementType state_machine.RequirementType) (*state_machine.ProjectStateMachine, error) {
	key := projectID + "_" + string(requirementType)
	psm, ok := r.projectStateMachines[key]
	if !ok {
		return nil, state_machine.ErrProjectStateMachineNotFound
	}
	return psm, nil
}

func (r *MockStateMachineRepository) ListProjectStateMachines(ctx context.Context, projectID string) ([]*state_machine.ProjectStateMachine, error) {
	var result []*state_machine.ProjectStateMachine
	for _, psm := range r.projectStateMachines {
		if psm.ProjectID() == projectID {
			result = append(result, psm)
		}
	}
	return result, nil
}

func (r *MockStateMachineRepository) DeleteProjectStateMachine(ctx context.Context, id string) error {
	for key, psm := range r.projectStateMachines {
		if psm.ID() == id {
			delete(r.projectStateMachines, key)
			return nil
		}
	}
	return nil
}

func (r *MockStateMachineRepository) DeleteProjectStateMachinesByProject(ctx context.Context, projectID string) error {
	for key, psm := range r.projectStateMachines {
		if psm.ProjectID() == projectID {
			delete(r.projectStateMachines, key)
		}
	}
	return nil
}

func (r *MockStateMachineRepository) Clear() {
	r.stateMachines = make(map[string]*state_machine.StateMachine)
	r.requirementStates = make(map[string]*state_machine.RequirementState)
	r.transitionLogs = []*state_machine.TransitionLog{}
}

const testYAML = `
name: test_flow
description: 测试流程
initial_state: created

states:
  - id: created
    name: 已创建
    is_final: false
  - id: in_progress
    name: 进行中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: created
    to: in_progress
    trigger: start
    description: 开始处理
  - from: in_progress
    to: completed
    trigger: complete
    description: 完成
`

func TestStateMachineService_CreateStateMachine(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	sm, err := svc.CreateStateMachine(ctx, "test", "测试", testYAML)
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}

	if sm.Name != "test" {
		t.Errorf("期望名称为 test, 实际为 %s", sm.Name)
	}
}

func TestStateMachineService_CreateStateMachine_InvalidYAML(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	_, err := svc.CreateStateMachine(ctx, "test", "测试", "invalid: yaml")
	if err == nil {
		t.Error("期望创建失败")
	}
}

func TestStateMachineService_CreateStateMachine_InvalidConfig(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	_, err := svc.CreateStateMachine(ctx, "test", "测试", `
name: test
initial_state: not_exist
states:
  - id: created
    name: 已创建
    is_final: false
`)
	if err == nil {
		t.Error("期望创建失败")
	}
}

func TestStateMachineService_GetStateMachine(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	sm, _ := svc.CreateStateMachine(ctx, "test", "测试", testYAML)

	found, err := svc.GetStateMachine(ctx, sm.ID)
	if err != nil {
		t.Fatalf("获取失败: %v", err)
	}

	if found.Name != "test" {
		t.Errorf("期望名称为 test, 实际为 %s", found.Name)
	}
}

func TestStateMachineService_GetStateMachine_NotFound(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	_, err := svc.GetStateMachine(ctx, "not-exist")
	if err == nil {
		t.Error("期望未找到")
	}
}

func TestStateMachineService_InitializeRequirementState(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	sm, _ := svc.CreateStateMachine(ctx, "test", "测试", testYAML)

	rs, err := svc.InitializeRequirementState(ctx, "req-1", sm.ID)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	if rs.CurrentState != "created" {
		t.Errorf("期望初始状态为 created, 实际为 %s", rs.CurrentState)
	}

	if len(repo.transitionLogs) != 1 {
		t.Errorf("期望1条日志, 实际为 %d", len(repo.transitionLogs))
	}
}

func TestStateMachineService_TriggerTransition(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	sm, _ := svc.CreateStateMachine(ctx, "test", "测试", testYAML)
	svc.InitializeRequirementState(ctx, "req-1", sm.ID)

	metadata := map[string]interface{}{"project_id": "project-1"}
	ctxWithMeta := infra_sm.WithMetadata(ctx, metadata)
	rs, err := svc.TriggerTransition(ctxWithMeta, "req-1", "start", "user", "开始处理")
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	if rs.CurrentState != "in_progress" {
		t.Errorf("期望状态为 in_progress, 实际为 %s", rs.CurrentState)
	}
}

func TestStateMachineService_TriggerTransition_InvalidTrigger(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	sm, _ := svc.CreateStateMachine(ctx, "test", "测试", testYAML)
	svc.InitializeRequirementState(ctx, "req-1", sm.ID)

	metadata := map[string]interface{}{"project_id": "project-1"}
	ctxWithMeta := infra_sm.WithMetadata(ctx, metadata)
	_, err := svc.TriggerTransition(ctxWithMeta, "req-1", "invalid", "user", "")
	if err == nil {
		t.Error("期望失败：无效的触发器")
	}
}

func TestStateMachineService_TriggerTransition_StateNotFound(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	sm, _ := svc.CreateStateMachine(ctx, "test", "测试", testYAML)
	rs, _ := svc.InitializeRequirementState(ctx, "req-1", sm.ID)
	rs.CurrentState = "not_exist"
	repo.UpdateRequirementState(ctx, rs)

	metadata := map[string]interface{}{"project_id": "project-1"}
	ctxWithMeta := infra_sm.WithMetadata(ctx, metadata)
	_, err := svc.TriggerTransition(ctxWithMeta, "req-1", "start", "user", "")
	if err == nil {
		t.Error("期望失败")
	}
}

func TestStateMachineService_GetRequirementState(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	sm, _ := svc.CreateStateMachine(ctx, "test", "测试", testYAML)
	svc.InitializeRequirementState(ctx, "req-1", sm.ID)

	rs, err := svc.GetRequirementState(ctx, "req-1")
	if err != nil {
		t.Fatalf("获取失败: %v", err)
	}

	if rs.RequirementID != "req-1" {
		t.Errorf("期望需求ID为 req-1, 实际为 %s", rs.RequirementID)
	}
}

func TestStateMachineService_GetRequirementState_NotFound(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	_, err := svc.GetRequirementState(ctx, "not-exist")
	if err == nil {
		t.Error("期望未找到")
	}
}

func TestStateMachineService_GetTransitionHistory(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	sm, _ := svc.CreateStateMachine(ctx, "test", "测试", testYAML)
	svc.InitializeRequirementState(ctx, "req-1", sm.ID)

	metadata := map[string]interface{}{"project_id": "project-1"}
	ctxWithMeta := infra_sm.WithMetadata(ctx, metadata)
	svc.TriggerTransition(ctxWithMeta, "req-1", "start", "user", "")

	logs, err := svc.GetTransitionHistory(ctx, "req-1")
	if err != nil {
		t.Fatalf("获取失败: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("期望2条日志, 实际为 %d", len(logs))
	}
}

func TestStateMachineService_DeleteStateMachine(t *testing.T) {
	repo := NewMockStateMachineRepository()
	logger, _ := zap.NewDevelopment()
	svc := NewStateMachineService(repo, nil, nil, logger)

	ctx := context.Background()
	sm, _ := svc.CreateStateMachine(ctx, "test", "测试", testYAML)

	err := svc.DeleteStateMachine(ctx, sm.ID)
	if err != nil {
		t.Fatalf("删除失败: %v", err)
	}

	_, err = svc.GetStateMachine(ctx, sm.ID)
	if err == nil {
		t.Error("期望未找到")
	}
}

func TestNewTransitionLog(t *testing.T) {
	log := state_machine.NewTransitionLog("req-1", "created", "in_progress", "start", "user", "开始")
	if log.Result != "success" {
		t.Errorf("期望结果为 success, 实际为 %s", log.Result)
	}

	if log.FromState != "created" {
		t.Errorf("期望源状态为 created, 实际为 %s", log.FromState)
	}

	if log.ToState != "in_progress" {
		t.Errorf("期望目标状态为 in_progress, 实际为 %s", log.ToState)
	}
}

func TestTransitionLog_MarkFailed(t *testing.T) {
	log := state_machine.NewTransitionLog("req-1", "created", "in_progress", "start", "user", "")
	log.MarkFailed("error message")

	if log.Result != "failed" {
		t.Errorf("期望结果为 failed, 实际为 %s", log.Result)
	}

	if log.ErrorMessage != "error message" {
		t.Errorf("期望错误信息为 error message, 实际为 %s", log.ErrorMessage)
	}
}

func TestNewRequirementState(t *testing.T) {
	rs := state_machine.NewRequirementState("req-1", "sm-1", "created", "已创建")
	if rs.RequirementID != "req-1" {
		t.Errorf("期望需求ID为 req-1, 实际为 %s", rs.RequirementID)
	}

	if rs.CurrentState != "created" {
		t.Errorf("期望状态为 created, 实际为 %s", rs.CurrentState)
	}
}

func TestRequirementState_Transition(t *testing.T) {
	rs := state_machine.NewRequirementState("req-1", "sm-1", "created", "已创建")
	oldTime := rs.UpdatedAt
	time.Sleep(time.Millisecond)

	rs.Transition("in_progress", "进行中")

	if rs.CurrentState != "in_progress" {
		t.Errorf("期望状态为 in_progress, 实际为 %s", rs.CurrentState)
	}

	if rs.CurrentStateName != "进行中" {
		t.Errorf("期望状态名称为 进行中, 实际为 %s", rs.CurrentStateName)
	}

	if !rs.UpdatedAt.After(oldTime) {
		t.Error("期望更新时间更新")
	}
}
