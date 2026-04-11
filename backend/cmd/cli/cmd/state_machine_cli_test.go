package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// DevOps CLI E2E 测试

const devOpsYAML = `
name: DevOps流程
description: 从代码提交到上线的完整DevOps流程

initial_state: code_submitted

states:
  - id: todo
    name: 待办
    is_final: false
  - id: code_submitted
    name: 代码已提交
    is_final: false
  - id: code_review
    name: 代码审查
    is_final: false
  - id: unit_test
    name: 单元测试
    is_final: false
  - id: integration_test
    name: 集成测试
    is_final: false
  - id: deploy_staging
    name: 部署预发布
    is_final: false
  - id: deploy_production
    name: 部署生产
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: todo
    to: code_submitted
    trigger: create
    description: 创建
  - from: code_submitted
    to: code_review
    trigger: start_review
    description: 开始代码审查
  - from: code_review
    to: unit_test
    trigger: review_passed
    description: 审查通过，开始单元测试
  - from: code_review
    to: code_submitted
    trigger: review_failed
    description: 审查失败，打回重做
  - from: unit_test
    to: integration_test
    trigger: unit_passed
    description: 单元测试通过
  - from: unit_test
    to: code_submitted
    trigger: unit_failed
    description: 单元测试失败
  - from: integration_test
    to: deploy_staging
    trigger: integration_passed
    description: 集成测试通过，部署预发布
  - from: integration_test
    to: code_submitted
    trigger: integration_failed
    description: 集成测试失败
  - from: deploy_staging
    to: deploy_production
    trigger: staging_verified
    description: 预发布验证通过，部署生产
  - from: deploy_staging
    to: integration_test
    trigger: staging_failed
    description: 预发布验证失败，回滚重测
  - from: deploy_production
    to: completed
    trigger: finish
    description: 完成
`

func TestStateMachineDevOpsCLI(t *testing.T) {
	// 检查 CLI 是否可用
	if _, err := exec.LookPath("taskmanager"); err != nil {
		t.Skip("taskmanager CLI not found in PATH, skipping E2E test")
	}

	// 检查 statemachine create 命令是否存在
	output, _ := exec.Command("taskmanager", "statemachine", "--help").CombinedOutput()
	if !strings.Contains(string(output), "create") {
		t.Skip("statemachine create command not available, skipping E2E test")
	}

	ctx := context.Background()
	var smID, reqID string

	// Step 1: 创建 DevOps 状态机
	t.Log("[Step 1] 创建 DevOps 状态机...")
	sm, err := createSM(ctx, "DevOps流程-E2E", "E2E测试", devOpsYAML)
	if err != nil {
		t.Fatalf("创建状态机失败: %v", err)
	}
	smID = sm.ID
	t.Logf("状态机创建成功: ID=%s", smID)

	// defer 清理
	defer func() {
		deleteSM(ctx, smID)
		t.Logf("已清理状态机: %s", smID)
	}()

	// Step 2: 验证状态机列表
	t.Log("[Step 2] 验证状态机列表...")
	list, err := listSM(ctx)
	if err != nil {
		t.Fatalf("列出状态机失败: %v", err)
	}
	found := false
	for _, s := range list {
		if s.ID == smID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("创建的状态机未出现在列表中")
	}

	// Step 3: 获取可用触发器
	t.Log("[Step 3] 获取可用触发器...")
	triggers, err := getTriggersSM(ctx, smID, "")
	if err != nil {
		t.Fatalf("获取触发器失败: %v", err)
	}
	if len(triggers) != 11 {
		t.Errorf("预期 11 个触发器，实际 %d 个", len(triggers))
	}

	// Step 4: 初始化需求
	t.Log("[Step 4] 初始化需求...")
	reqID = "DEV-E2E-001"
	rs, err := initReq(ctx, reqID, smID)
	if err != nil {
		t.Fatalf("初始化需求状态失败: %v", err)
	}
	if rs.CurrentState != "code_submitted" {
		t.Errorf("预期初始状态 code_submitted，实际 %s", rs.CurrentState)
	}

	// Step 5: 执行 DevOps 流程
	t.Log("[Step 5] 执行 DevOps 流程...")

	steps := []struct {
		name      string
		trigger   string
		wantState string
	}{
		{"提交代码审查", "start_review", "code_review"},
		{"审查通过", "review_passed", "unit_test"},
		{"单元测试通过", "unit_passed", "integration_test"},
		{"集成测试通过", "integration_passed", "deploy_staging"},
		{"预发布验证通过", "staging_verified", "deploy_production"},
	}

	for i, step := range steps {
		// 获取当前状态可用的触发器
		availTriggers, _ := getTriggersSM(ctx, smID, rs.CurrentState)
		triggerFound := false
		for _, t := range availTriggers {
			if t.Trigger == step.trigger {
				triggerFound = true
				break
			}
		}
		if !triggerFound {
			t.Errorf("[%d] 触发器 %s 不可用于当前状态 %s", i, step.trigger, rs.CurrentState)
		}

		// 执行转换
		tr, err := transitionReq(ctx, reqID, step.trigger, "e2e-test", step.name)
		if err != nil {
			t.Errorf("[%d] 转换 %s 失败: %v", i, step.trigger, err)
			continue
		}

		if tr.ToState != step.wantState {
			t.Errorf("[%d] 预期状态 %s，实际 %s", i, step.wantState, tr.ToState)
		}
		t.Logf("  [%d/%d] %s: %s → %s", i+1, len(steps), step.name, tr.FromState, tr.ToState)

		// 更新状态
		rs, _ = getReqState(ctx, reqID)
	}

	// Step 6: 验证最终状态
	t.Log("[Step 6] 验证最终状态...")
	finalState, err := getReqState(ctx, reqID)
	if err != nil {
		t.Fatalf("获取最终状态失败: %v", err)
	}
	if finalState.CurrentState != "deploy_production" {
		t.Errorf("预期最终状态 deploy_production，实际 %s", finalState.CurrentState)
	}

	// Step 7: 验证转换历史
	t.Log("[Step 7] 验证转换历史...")
	history, err := getReqHistory(ctx, reqID)
	if err != nil {
		t.Fatalf("获取转换历史失败: %v", err)
	}
	// init + 5 次转换 = 6 条记录
	if len(history) != 6 {
		t.Errorf("预期 6 条历史记录，实际 %d 条", len(history))
	}
}

// ============ CLI 调用封装 ============

type smInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type triggerInfo struct {
	Trigger    string `json:"trigger"`
	FromState string `json:"from_state"`
	ToState   string `json:"to_state"`
}

type reqStateInfo struct {
	RequirementID   string `json:"requirement_id"`
	CurrentState    string `json:"current_state"`
	CurrentStateName string `json:"current_state_name"`
}

type transResultInfo struct {
	FromState   string `json:"from_state"`
	ToState     string `json:"to_state"`
	Trigger     string `json:"trigger"`
	TriggeredBy string `json:"triggered_by"`
}

type transHistoryInfo struct {
	FromState   string `json:"from_state"`
	ToState     string `json:"to_state"`
	Trigger     string `json:"trigger"`
	TriggeredBy string `json:"triggered_by"`
	Result      string `json:"result"`
}

func runCLI(args ...string) (string, error) {
	cmd := exec.Command("taskmanager", args...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func createSM(ctx context.Context, name, desc, yaml string) (*smInfo, error) {
	output, err := runCLI("statemachine", "create", "-n", name, "-d", desc, "-c", yaml)
	if err != nil {
		return nil, err
	}
	var sm smInfo
	if err := json.Unmarshal([]byte(output), &sm); err != nil {
		return nil, fmt.Errorf("parse error: %v, output: %s", err, output)
	}
	return &sm, nil
}

func listSM(ctx context.Context) ([]smInfo, error) {
	output, err := runCLI("statemachine", "list")
	if err != nil {
		return nil, err
	}
	var list []smInfo
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	return list, nil
}

func getTriggersSM(ctx context.Context, smID, state string) ([]triggerInfo, error) {
	args := []string{"statemachine", "triggers", smID}
	if state != "" {
		args = append(args, "-s", state)
	}
	output, err := runCLI(args...)
	if err != nil {
		return nil, err
	}
	var triggers []triggerInfo
	if err := json.Unmarshal([]byte(output), &triggers); err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	return triggers, nil
}

func initReq(ctx context.Context, reqID, smID string) (*reqStateInfo, error) {
	output, err := runCLI("statemachine", "init", reqID, "-s", smID)
	if err != nil {
		return nil, err
	}
	var rs reqStateInfo
	if err := json.Unmarshal([]byte(output), &rs); err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	return &rs, nil
}

func getReqState(ctx context.Context, reqID string) (*reqStateInfo, error) {
	output, err := runCLI("statemachine", "state", reqID)
	if err != nil {
		return nil, err
	}
	var rs reqStateInfo
	if err := json.Unmarshal([]byte(output), &rs); err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	return &rs, nil
}

func transitionReq(ctx context.Context, reqID, trigger, triggeredBy, remark string) (*transResultInfo, error) {
	args := []string{"statemachine", "transition", reqID, "-t", trigger, "-b", triggeredBy, "-r", remark}
	output, err := runCLI(args...)
	if err != nil {
		return nil, err
	}
	var tr transResultInfo
	if err := json.Unmarshal([]byte(output), &tr); err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	return &tr, nil
}

func getReqHistory(ctx context.Context, reqID string) ([]transHistoryInfo, error) {
	output, err := runCLI("statemachine", "history", reqID)
	if err != nil {
		return nil, err
	}
	var history []transHistoryInfo
	if err := json.Unmarshal([]byte(output), &history); err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	return history, nil
}

func deleteSM(ctx context.Context, smID string) error {
	_, err := runCLI("statemachine", "delete", smID)
	return err
}
