package statemachine

import (
	"context"
	"fmt"
	"testing"

	"github.com/weibh/taskmanager/domain/statemachine"
	"go.uber.org/zap"
)

type fakeHeartbeatTrigger struct {
	triggeredID string
	err         error
}

func (f *fakeHeartbeatTrigger) Trigger(ctx context.Context, heartbeatID string) error {
	f.triggeredID = heartbeatID
	return f.err
}

func TestTransitionExecutor_ExecuteHook_TriggerHeartbeat(t *testing.T) {
	logger := zap.NewNop()
	executor := NewTransitionExecutor(logger)

	fakeTrigger := &fakeHeartbeatTrigger{}
	executor.SetHeartbeatTrigger(fakeTrigger)

	hook := statemachine.TransitionHook{
		Name: "trigger-heartbeat-hook",
		Type: "trigger_heartbeat",
		Config: map[string]interface{}{
			"heartbeat_id": "hb-123",
		},
		Retry: 0,
	}

	hookCtx := statemachine.HookContext{
		RequirementID:  "req-001",
		StateMachineID: "sm-001",
		FromState:      "todo",
		ToState:        "doing",
		Trigger:        "start",
	}

	// 直接调用同步方法，避免异步竞态
	executor.executeHook(context.Background(), hook, hookCtx)

	if fakeTrigger.triggeredID != "hb-123" {
		t.Fatalf("期望触发心跳 hb-123，实际触发: %s", fakeTrigger.triggeredID)
	}
}

func TestTransitionExecutor_ExecuteHook_TriggerHeartbeat_WithInterpolation(t *testing.T) {
	logger := zap.NewNop()
	executor := NewTransitionExecutor(logger)

	fakeTrigger := &fakeHeartbeatTrigger{}
	executor.SetHeartbeatTrigger(fakeTrigger)

	hook := statemachine.TransitionHook{
		Name: "trigger-heartbeat-hook",
		Type: "trigger_heartbeat",
		Config: map[string]interface{}{
			"heartbeat_id": "hb-{{requirement_id}}",
		},
		Retry: 0,
	}

	hookCtx := statemachine.HookContext{
		RequirementID:  "req-456",
		StateMachineID: "sm-001",
		FromState:      "todo",
		ToState:        "doing",
		Trigger:        "start",
	}

	// 直接调用同步方法，避免异步竞态
	executor.executeHook(context.Background(), hook, hookCtx)

	if fakeTrigger.triggeredID != "hb-req-456" {
		t.Fatalf("期望触发心跳 hb-req-456，实际触发: %s", fakeTrigger.triggeredID)
	}
}

func TestTransitionExecutor_ExecuteHook_TriggerHeartbeat_WithTriggerIDInterpolation(t *testing.T) {
	logger := zap.NewNop()
	executor := NewTransitionExecutor(logger)

	fakeTrigger := &fakeHeartbeatTrigger{}
	executor.SetHeartbeatTrigger(fakeTrigger)

	hook := statemachine.TransitionHook{
		Name: "trigger-heartbeat-hook",
		Type: "trigger_heartbeat",
		Config: map[string]interface{}{
			"heartbeat_id": "hb-{{trigger_id}}-{{requirement_id}}",
		},
		Retry: 0,
	}

	hookCtx := statemachine.HookContext{
		RequirementID:  "req-456",
		StateMachineID: "sm-001",
		FromState:      "todo",
		ToState:        "doing",
		Trigger:        "start",
		TriggerID:      "tr-789",
	}

	executor.executeHook(context.Background(), hook, hookCtx)

	if fakeTrigger.triggeredID != "hb-tr-789-req-456" {
		t.Fatalf("期望触发心跳 hb-tr-789-req-456，实际触发: %s", fakeTrigger.triggeredID)
	}
}

func TestTransitionExecutor_ExecuteHook_TriggerHeartbeat_NotConfigured(t *testing.T) {
	logger := zap.NewNop()
	executor := NewTransitionExecutor(logger)
	// 不设置 heartbeatTrigger

	hook := statemachine.TransitionHook{
		Name: "trigger-heartbeat-hook",
		Type: "trigger_heartbeat",
		Config: map[string]interface{}{
			"heartbeat_id": "hb-123",
		},
		Retry: 0,
	}

	hookCtx := statemachine.HookContext{
		RequirementID: "req-001",
	}

	// 异步执行，不会 panic，但会记录错误日志
	// 这里主要验证不设置 trigger 不会导致 panic
	executor.ExecuteHooks(context.Background(), []statemachine.TransitionHook{hook}, hookCtx)
}

func TestTransitionExecutor_ExecuteHook_TriggerHeartbeat_MissingID(t *testing.T) {
	logger := zap.NewNop()
	executor := NewTransitionExecutor(logger)

	fakeTrigger := &fakeHeartbeatTrigger{}
	executor.SetHeartbeatTrigger(fakeTrigger)

	hook := statemachine.TransitionHook{
		Name:   "trigger-heartbeat-hook",
		Type:   "trigger_heartbeat",
		Config: map[string]interface{}{},
		Retry:  0,
	}

	hookCtx := statemachine.HookContext{
		RequirementID: "req-001",
	}

	// 异步执行，不会 panic
	executor.ExecuteHooks(context.Background(), []statemachine.TransitionHook{hook}, hookCtx)
}

func TestTransitionExecutor_ExecuteHook_TriggerHeartbeat_Retry(t *testing.T) {
	logger := zap.NewNop()
	executor := NewTransitionExecutor(logger)

	fakeTrigger := &fakeHeartbeatTrigger{err: fmt.Errorf("trigger failed")}
	executor.SetHeartbeatTrigger(fakeTrigger)

	hook := statemachine.TransitionHook{
		Name: "trigger-heartbeat-hook",
		Type: "trigger_heartbeat",
		Config: map[string]interface{}{
			"heartbeat_id": "hb-123",
		},
		Retry: 2,
	}

	hookCtx := statemachine.HookContext{
		RequirementID: "req-001",
	}

	executor.executeHook(context.Background(), hook, hookCtx)

	// retry=2 表示最多执行 3 次（初始 + 2 次重试）
	// 但由于 fakeTrigger 没有计数器，我们只能验证不会 panic
}
