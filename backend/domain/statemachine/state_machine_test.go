package statemachine

import (
	"testing"
)

const validYAML = `
name: test_flow
description: 测试流程
initial_state: todo

states:
  - id: todo
    name: 待办
    is_final: false
  - id: analyzing
    name: 分析中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: todo
    to: analyzing
    trigger: start
    description: 开始处理
  - from: analyzing
    to: completed
    trigger: complete
    description: 完成
`

func TestParseConfig_Success(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if cfg.Name != "test_flow" {
		t.Errorf("期望名称为 test_flow, 实际为 %s", cfg.Name)
	}

	if cfg.InitialState != "todo" {
		t.Errorf("期望初始状态为 todo, 实际为 %s", cfg.InitialState)
	}

	if len(cfg.States) != 3 {
		t.Errorf("期望3个状态, 实际为 %d", len(cfg.States))
	}

	if len(cfg.Transitions) != 2 {
		t.Errorf("期望2个转换, 实际为 %d", len(cfg.Transitions))
	}
}

func TestParseConfig_InvalidYAML(t *testing.T) {
	_, err := ParseConfig("invalid: [yaml:")
	if err == nil {
		t.Error("期望解析失败")
	}
}

func TestConfig_Validate_Success(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("校验失败: %v", err)
	}
}

func TestConfig_Validate_InitialStateNotFound(t *testing.T) {
	yaml := `
name: test
initial_state: not_exist
states:
  - id: todo
    name: 待办
    is_final: false
  - id: processing
    name: 处理中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true
`
	cfg, err := ParseConfig(yaml)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	err = cfg.Validate()
	if err == nil {
		t.Error("期望校验失败")
	}
}

func TestConfig_Validate_TransitionFromStateNotFound(t *testing.T) {
	yaml := `
name: test
initial_state: todo
states:
  - id: todo
    name: 待办
    is_final: false
  - id: processing
    name: 处理中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true
transitions:
  - from: not_exist
    to: todo
    trigger: start
`
	cfg, err := ParseConfig(yaml)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	err = cfg.Validate()
	if err == nil {
		t.Error("期望校验失败")
	}
}

func TestConfig_Validate_FinalStateCannotTransition(t *testing.T) {
	yaml := `
name: test
initial_state: todo
states:
  - id: todo
    name: 待办
    is_final: false
  - id: processing
    name: 处理中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true
transitions:
  - from: completed
    to: todo
    trigger: reopen
`
	cfg, err := ParseConfig(yaml)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	err = cfg.Validate()
	if err == nil {
		t.Error("期望校验失败：终态不能作为转换源")
	}
}

func TestConfig_Validate_MandatoryStates(t *testing.T) {
	// 缺少 todo 状态
	yaml1 := `
name: test
initial_state: analyzing
states:
  - id: analyzing
    name: 分析中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true
transitions:
  - from: analyzing
    to: completed
    trigger: complete
`
	cfg1, err := ParseConfig(yaml1)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	err = cfg1.Validate()
	if err == nil {
		t.Error("期望校验失败：缺少 todo 状态")
	}

	// 缺少 completed 状态
	yaml2 := `
name: test
initial_state: todo
states:
  - id: todo
    name: 待办
    is_final: false
  - id: analyzing
    name: 分析中
    is_final: false
transitions:
  - from: todo
    to: analyzing
    trigger: start
`
	cfg2, err := ParseConfig(yaml2)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	err = cfg2.Validate()
	if err == nil {
		t.Error("期望校验失败：缺少 completed 状态")
	}

	// todo 没有出向转换
	yaml3 := `
name: test
initial_state: todo
states:
  - id: todo
    name: 待办
    is_final: false
  - id: completed
    name: 已完成
    is_final: true
`
	cfg3, err := ParseConfig(yaml3)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	err = cfg3.Validate()
	if err == nil {
		t.Error("期望校验失败：todo 没有出向转换")
	}

	// completed 未标记为终态
	yaml4 := `
name: test
initial_state: todo
states:
  - id: todo
    name: 待办
    is_final: false
  - id: analyzing
    name: 分析中
    is_final: false
  - id: completed
    name: 已完成
    is_final: false
transitions:
  - from: todo
    to: analyzing
    trigger: start
`
	cfg4, err := ParseConfig(yaml4)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	err = cfg4.Validate()
	if err == nil {
		t.Error("期望校验失败：completed 未标记为终态")
	}
}

func TestConfig_FindTransition_Success(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	transition := cfg.FindTransition("todo", "start")
	if transition == nil {
		t.Fatal("期望找到转换")
	}

	if transition.ToState != "analyzing" {
		t.Errorf("期望目标状态为 analyzing, 实际为 %s", transition.ToState)
	}
}

func TestConfig_FindTransition_NotFound(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	transition := cfg.FindTransition("todo", "invalid_trigger")
	if transition != nil {
		t.Error("期望未找到转换")
	}
}

func TestConfig_GetState_Success(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	state := cfg.GetState("analyzing")
	if state == nil {
		t.Fatal("期望找到状态")
	}

	if state.Name != "分析中" {
		t.Errorf("期望名称为 分析中, 实际为 %s", state.Name)
	}

	if state.IsFinal {
		t.Error("期望不是终态")
	}
}

func TestConfig_GetState_NotFound(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	state := cfg.GetState("not_exist")
	if state != nil {
		t.Error("期望未找到状态")
	}
}

func TestNewStateMachine(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	sm := NewStateMachine("test", "测试状态机", cfg)

	if sm.ID == "" {
		t.Error("期望ID不为空")
	}


	if sm.Name != "test" {
		t.Errorf("期望名称为 test, 实际为 %s", sm.Name)
	}

	if sm.Config.InitialState != "todo" {
		t.Errorf("期望初始状态为 todo, 实际为 %s", sm.Config.InitialState)
	}
}
