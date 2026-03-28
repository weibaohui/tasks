/**
 * FeishuThinkingProcessHook 单元测试
 */
package hooks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

func TestFeishuThinkingProcessHook_New(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)

	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	assert.NotNil(t, hook)
	assert.Equal(t, "feishu_thinking_process", hook.Name())
	assert.Equal(t, 60, hook.Priority())
	assert.Equal(t, domain.HookTypeLLM, hook.HookType())
	assert.True(t, hook.Enabled())
}

func TestFeishuThinkingProcessHook_PreLLMCall_Disabled(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	ctx := domain.NewHookContext(t.Context())
	callCtx := &domain.LLMCallContext{
		Prompt:    "test prompt",
		SessionID: "session-123",
		Metadata: map[string]string{
			"enable_thinking_process": "false",
		},
	}

	result, err := hook.PreLLMCall(ctx, callCtx)

	assert.NoError(t, err)
	assert.Equal(t, callCtx, result)
}

func TestFeishuThinkingProcessHook_PreLLMCall_Enabled(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	ctx := domain.NewHookContext(t.Context())
	ctx.SetMetadata("session_key", "session-123")

	callCtx := &domain.LLMCallContext{
		Prompt:    "test prompt",
		SessionID: "session-123",
		Metadata: map[string]string{
			"enable_thinking_process": "true",
			"channel_type":            "feishu",
			"chat_id":                 "chat-456",
		},
	}

	result, err := hook.PreLLMCall(ctx, callCtx)

	assert.NoError(t, err)
	assert.Equal(t, callCtx, result)
	// 验证缓存已更新
	info, exists := hook.sessionCache["session-123"]
	assert.True(t, exists)
	assert.True(t, info.EnableThinkingProcess)
	assert.Equal(t, "feishu", info.Channel)
	assert.Equal(t, "chat-456", info.ChatID)
}

func TestFeishuThinkingProcessHook_PostLLMCall_WithToolCalls(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	// 先设置 session 缓存
	ctx := domain.NewHookContext(t.Context())
	ctx.SetMetadata("session_key", "session-123")

	// 先调用 PreLLMCall 来设置缓存
	preCallCtx := &domain.LLMCallContext{
		SessionID: "session-123",
		Metadata: map[string]string{
			"enable_thinking_process": "true",
			"channel_type":            "feishu",
			"chat_id":                 "chat-456",
		},
	}
	hook.PreLLMCall(ctx, preCallCtx)

	// 再调用 PostLLMCall
	resp := &domain.LLMResponse{
		Content:           "I will call a tool",
		ContainsToolCalls: true,
		RawResponse:       `{"tool_calls": [{"id": "call-1", "function": {"name": "bash", "arguments": "{}"}}]}`,
	}

	result, err := hook.PostLLMCall(ctx, preCallCtx, resp)

	assert.NoError(t, err)
	assert.Equal(t, resp, result)
}

func TestFeishuThinkingProcessHook_PreToolCall(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	ctx := domain.NewHookContext(t.Context())
	ctx.SetMetadata("session_key", "session-123")

	// 设置缓存
	hook.sessionCache["session-123"] = &sessionInfo{
		SessionKey:            "session-123",
		ChatID:                "chat-456",
		Channel:               "feishu",
		EnableThinkingProcess: true,
		UpdatedAt:             time.Now(),
	}

	callCtx := &domain.ToolCallContext{
		ToolName:  "bash",
		ToolInput: map[string]interface{}{"command": "ls"},
	}

	result, err := hook.PreToolCall(ctx, callCtx)

	assert.NoError(t, err)
	assert.Equal(t, callCtx, result)
}

func TestFeishuThinkingProcessHook_PostToolCall_Success(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	ctx := domain.NewHookContext(t.Context())
	ctx.SetMetadata("session_key", "session-123")

	// 设置缓存
	hook.sessionCache["session-123"] = &sessionInfo{
		SessionKey:            "session-123",
		ChatID:                "chat-456",
		Channel:               "feishu",
		EnableThinkingProcess: true,
		UpdatedAt:             time.Now(),
	}

	callCtx := &domain.ToolCallContext{
		ToolName: "bash",
	}

	result := &domain.ToolExecutionResult{
		Success:  true,
		Output:   "file1.txt\nfile2.txt",
		Duration: 100 * time.Millisecond,
	}

	resp, err := hook.PostToolCall(ctx, callCtx, result)

	assert.NoError(t, err)
	assert.Equal(t, result, resp)
}

func TestFeishuThinkingProcessHook_PostToolCall_Error(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	ctx := domain.NewHookContext(t.Context())
	ctx.SetMetadata("session_key", "session-123")

	// 设置缓存
	hook.sessionCache["session-123"] = &sessionInfo{
		SessionKey:            "session-123",
		ChatID:                "chat-456",
		Channel:               "feishu",
		EnableThinkingProcess: true,
		UpdatedAt:             time.Now(),
	}

	callCtx := &domain.ToolCallContext{
		ToolName: "bash",
	}

	result := &domain.ToolExecutionResult{
		Success:  false,
		Error:    assert.AnError,
		Duration: 50 * time.Millisecond,
	}

	resp, err := hook.PostToolCall(ctx, callCtx, result)

	assert.NoError(t, err)
	assert.Equal(t, result, resp)
}

func TestFeishuThinkingProcessHook_OnToolError(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	ctx := domain.NewHookContext(t.Context())
	ctx.SetMetadata("session_key", "session-123")

	// 设置缓存
	hook.sessionCache["session-123"] = &sessionInfo{
		SessionKey:            "session-123",
		ChatID:                "chat-456",
		Channel:               "feishu",
		EnableThinkingProcess: true,
		UpdatedAt:             time.Now(),
	}

	callCtx := &domain.ToolCallContext{
		ToolName: "bash",
	}

	testErr := assert.AnError

	result, err := hook.OnToolError(ctx, callCtx, testErr)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, testErr, result.Error)
}

func TestFeishuThinkingProcessHook_isThinkingProcessEnabled(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	// 测试从 callCtx.Metadata 检查
	ctx := domain.NewHookContext(t.Context())
	callCtx := &domain.LLMCallContext{
		Metadata: map[string]string{
			"enable_thinking_process": "true",
		},
	}

	assert.True(t, hook.isThinkingProcessEnabled(ctx, callCtx))

	// 测试从 sessionCache 检查
	ctx.SetMetadata("session_key", "session-123")
	hook.sessionCache["session-123"] = &sessionInfo{
		SessionKey:            "session-123",
		EnableThinkingProcess: true,
		UpdatedAt:             time.Now(),
	}

	assert.True(t, hook.isThinkingProcessEnabled(ctx, nil))

	// 测试禁用状态
	callCtx2 := &domain.LLMCallContext{
		Metadata: map[string]string{
			"enable_thinking_process": "false",
		},
	}
	assert.False(t, hook.isThinkingProcessEnabled(ctx, callCtx2))
}

func TestFeishuThinkingProcessHook_extractToolNames(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	// 测试正常提取
	rawResponse := `{"tool_calls": [{"id": "call-1", "function": {"name": "bash", "arguments": "{}"}}, {"id": "call-2", "function": {"name": "read_file", "arguments": "{}"}}]}`
	names := hook.extractToolNames(rawResponse)
	assert.Len(t, names, 2)
	assert.Contains(t, names, "`bash`")
	assert.Contains(t, names, "`read_file`")

	// 测试空响应
	names = hook.extractToolNames("")
	assert.Nil(t, names)

	// 测试无效 JSON
	names = hook.extractToolNames("invalid json")
	assert.Nil(t, names)

	// 测试没有 tool_calls
	names = hook.extractToolNames(`{"content": "hello"}`)
	assert.Nil(t, names)
}

func TestFeishuThinkingProcessHook_NonFeishuChannel(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	ctx := domain.NewHookContext(t.Context())
	ctx.SetMetadata("session_key", "session-123")
	ctx.SetMetadata("chat_id", "chat-456")

	// 设置缓存为非飞书渠道
	hook.sessionCache["session-123"] = &sessionInfo{
		SessionKey:            "session-123",
		ChatID:                "chat-456",
		Channel:               "dingtalk", // 非飞书渠道
		EnableThinkingProcess: true,
		UpdatedAt:             time.Now(),
	}

	callCtx := &domain.LLMCallContext{
		Prompt:    "test",
		SessionID: "session-123",
		Metadata: map[string]string{
			"enable_thinking_process": "true",
		},
	}

	// 调用 PreLLMCall，应该不会发送消息到非飞书渠道
	hook.PreLLMCall(ctx, callCtx)

	// 由于是非飞书渠道，消息应该不会发送（但会更新缓存）
	// 这里主要验证没有 panic
	assert.True(t, hook.isThinkingProcessEnabled(ctx, callCtx))
}

func TestFeishuThinkingProcessHook_getSessionInfo(t *testing.T) {
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)
	hook := NewFeishuThinkingProcessHook(messageBus, logger)

	ctx := domain.NewHookContext(t.Context())

	// 测试空 ctx
	info := hook.getSessionInfo(nil)
	assert.Nil(t, info)

	// 测试无 session_key
	info = hook.getSessionInfo(ctx)
	assert.Nil(t, info)

	// 设置 session_key 但没有缓存
	ctx.SetMetadata("session_key", "session-123")
	info = hook.getSessionInfo(ctx)
	assert.Nil(t, info)

	// 设置缓存
	hook.sessionCache["session-123"] = &sessionInfo{
		SessionKey:            "session-123",
		ChatID:                "chat-456",
		Channel:               "feishu",
		EnableThinkingProcess: true,
		UpdatedAt:             time.Now(),
	}

	info = hook.getSessionInfo(ctx)
	assert.NotNil(t, info)
	assert.Equal(t, "chat-456", info.ChatID)

	// 测试过期缓存
	hook.sessionCache["session-old"] = &sessionInfo{
		SessionKey:            "session-old",
		ChatID:                "chat-old",
		Channel:               "feishu",
		EnableThinkingProcess: true,
		UpdatedAt:             time.Now().Add(-31 * time.Minute),
	}
	ctx.SetMetadata("session_key", "session-old")
	info = hook.getSessionInfo(ctx)
	assert.Nil(t, info) // 应该返回 nil，因为缓存已过期
}
