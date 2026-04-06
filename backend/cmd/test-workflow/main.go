package main

import (
	"context"
	"fmt"
	"os"

	"github.com/weibh/taskmanager/application"
	infra_sm "github.com/weibh/taskmanager/infrastructure/state_machine"
	"github.com/weibh/taskmanager/domain/state_machine"
	"go.uber.org/zap"
)

// MockStateMachineRepository 测试用 Mock
type MockStateMachineRepository struct {
	stateMachines      map[string]*state_machine.StateMachine
	requirementStates map[string]*state_machine.RequirementState
	transitionLogs    []*state_machine.TransitionLog
}

func NewMockStateMachineRepository() *MockStateMachineRepository {
	return &MockStateMachineRepository{
		stateMachines:      make(map[string]*state_machine.StateMachine),
		requirementStates: make(map[string]*state_machine.RequirementState),
		transitionLogs:    []*state_machine.TransitionLog{},
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
	return nil
}

func (r *MockStateMachineRepository) GetProjectStateMachine(ctx context.Context, projectID string, requirementType state_machine.RequirementType) (*state_machine.ProjectStateMachine, error) {
	return nil, state_machine.ErrProjectStateMachineNotFound
}

func (r *MockStateMachineRepository) ListProjectStateMachines(ctx context.Context, projectID string) ([]*state_machine.ProjectStateMachine, error) {
	return nil, nil
}

func (r *MockStateMachineRepository) DeleteProjectStateMachine(ctx context.Context, id string) error {
	return nil
}

func (r *MockStateMachineRepository) DeleteProjectStateMachinesByProject(ctx context.Context, projectID string) error {
	return nil
}

// 简化版状态机 YAML
const yamlConfig = `
name: simple_workflow
description: 简化版开发流程，用于E2E测试

initial_state: submitted

states:
  - id: submitted
    name: 已提交
    is_final: false
  - id: in_review
    name: 审查中
    is_final: false
  - id: building
    name: 构建中
    is_final: false
  - id: testing
    name: 测试中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: submitted
    to: in_review
    trigger: submit_review
    description: 提交审查
    hooks:
      - name: 通知审查者
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: in_review
    to: building
    trigger: approve
    description: 审查通过
    hooks:
      - name: 触发构建
        type: command
        config:
          command: echo "Building {{requirement_id}}..."
        timeout: 30
        retry: 0

  - from: in_review
    to: submitted
    trigger: reject
    description: 审查拒绝
    hooks:
      - name: 通知开发者
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: building
    to: testing
    trigger: build_success
    description: 构建成功
    hooks:
      - name: 触发测试
        type: command
        config:
          command: echo "Testing {{requirement_id}}..."
        timeout: 30
        retry: 0

  - from: building
    to: submitted
    trigger: build_failed
    description: 构建失败
    hooks:
      - name: 通知失败
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: testing
    to: completed
    trigger: test_pass
    description: 测试通过
    hooks:
      - name: 发送完成通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: testing
    to: building
    trigger: test_failed
    description: 测试失败
    hooks:
      - name: 通知测试失败
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1
`

func main() {
	logger, _ := zap.NewDevelopment()
	repo := NewMockStateMachineRepository()
	executor := infra_sm.NewTransitionExecutor(logger)
	svc := application.NewStateMachineService(repo, executor, logger)
	ctx := context.Background()

	fmt.Println("=== 状态机 E2E 测试 ===\n")

	// 1. 创建状态机
	fmt.Println("Step 1: 创建状态机")
	sm, err := svc.CreateStateMachine(ctx, "simple_workflow", "简化版工作流", yamlConfig)
	if err != nil {
		fmt.Printf("❌ 创建状态机失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 状态机创建成功: ID=%s, Name=%s\n", sm.ID, sm.Name)
	fmt.Printf("   初始状态: %s\n\n", sm.Config.InitialState)

	// 2. 初始化需求状态
	fmt.Println("Step 2: 初始化需求状态")
	requirementID := "req-001"
	rs, err := svc.InitializeRequirementState(ctx, requirementID, sm.ID)
	if err != nil {
		fmt.Printf("❌ 初始化需求状态失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 需求状态初始化成功: RequirementID=%s, CurrentState=%s\n\n", rs.RequirementID, rs.CurrentState)

	// 3. 提交审查
	fmt.Println("Step 3: 提交审查 (submitted -> in_review)")
	rs, err = svc.TriggerTransition(ctx, requirementID, "submit_review", "developer", "提交代码审查")
	if err != nil {
		fmt.Printf("❌ 提交审查失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 状态转换成功: CurrentState=%s\n\n", rs.CurrentState)

	// 4. 审查通过
	fmt.Println("Step 4: 审查通过 (in_review -> building)")
	rs, err = svc.TriggerTransition(ctx, requirementID, "approve", "reviewer", "代码审查通过")
	if err != nil {
		fmt.Printf("❌ 审查通过失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 状态转换成功: CurrentState=%s\n\n", rs.CurrentState)

	// 5. 构建成功
	fmt.Println("Step 5: 构建成功 (building -> testing)")
	rs, err = svc.TriggerTransition(ctx, requirementID, "build_success", "ci_system", "构建完成")
	if err != nil {
		fmt.Printf("❌ 构建成功转换失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 状态转换成功: CurrentState=%s\n\n", rs.CurrentState)

	// 6. 测试通过
	fmt.Println("Step 6: 测试通过 (testing -> completed)")
	rs, err = svc.TriggerTransition(ctx, requirementID, "test_pass", "qa_engineer", "所有测试通过")
	if err != nil {
		fmt.Printf("❌ 测试通过转换失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 状态转换成功: CurrentState=%s\n\n", rs.CurrentState)

	// 7. 获取转换历史
	fmt.Println("Step 7: 获取转换历史")
	logs, err := svc.GetTransitionHistory(ctx, requirementID)
	if err != nil {
		fmt.Printf("❌ 获取转换历史失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("转换历史记录数: %d\n", len(logs))
	for i, log := range logs {
		status := "✅"
		if log.Result != "success" {
			status = "❌"
		}
		fmt.Printf("  [%d] %s %s -> %s, trigger=%s, by=%s\n",
			i+1, status, log.FromState, log.ToState, log.Trigger, log.TriggeredBy)
	}

	fmt.Println("\n=== 完整流程测试通过! ===")
}
