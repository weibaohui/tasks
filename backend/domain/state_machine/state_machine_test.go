package state_machine

import (
	"testing"
)

const validYAML = `
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

func TestParseConfig_Success(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if cfg.Name != "test_flow" {
		t.Errorf("期望名称为 test_flow, 实际为 %s", cfg.Name)
	}

	if cfg.InitialState != "created" {
		t.Errorf("期望初始状态为 created, 实际为 %s", cfg.InitialState)
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
  - id: created
    name: 已创建
    is_final: false
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
initial_state: created
states:
  - id: created
    name: 已创建
    is_final: false
transitions:
  - from: not_exist
    to: created
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
initial_state: created
states:
  - id: created
    name: 已创建
    is_final: false
  - id: completed
    name: 已完成
    is_final: true
transitions:
  - from: completed
    to: created
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

func TestConfig_FindTransition_Success(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	transition := cfg.FindTransition("created", "start")
	if transition == nil {
		t.Fatal("期望找到转换")
	}

	if transition.ToState != "in_progress" {
		t.Errorf("期望目标状态为 in_progress, 实际为 %s", transition.ToState)
	}
}

func TestConfig_FindTransition_NotFound(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	transition := cfg.FindTransition("created", "invalid_trigger")
	if transition != nil {
		t.Error("期望未找到转换")
	}
}

func TestConfig_GetState_Success(t *testing.T) {
	cfg, err := ParseConfig(validYAML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	state := cfg.GetState("in_progress")
	if state == nil {
		t.Fatal("期望找到状态")
	}

	if state.Name != "进行中" {
		t.Errorf("期望名称为 进行中, 实际为 %s", state.Name)
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

	if sm.Config.InitialState != "created" {
		t.Errorf("期望初始状态为 created, 实际为 %s", sm.Config.InitialState)
	}
}
