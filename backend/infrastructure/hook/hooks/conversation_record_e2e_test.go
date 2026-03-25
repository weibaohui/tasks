/**
 * ConversationRecordHook 端到端测试
 * 简化版：验证最终数据库中的记录关系，不预设具体 span 值
 */
package hooks

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"go.uber.org/zap"
)

// setupE2eDB 创建测试数据库
func setupE2eDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}
	err = persistence.InitSchema(db)
	if err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}
	return db, func() { db.Close() }
}

// mockIDGenE2E 简单的 ID 生成器
type mockIDGenE2E struct {
	counter int
}

func (m *mockIDGenE2E) Generate() string {
	m.counter++
	return "span-" + string(rune('0'+m.counter))
}

// TestE2E_TraceTree 数据库端到端测试（简化版）
// 验证追踪树的父子关系，不预设具体 span 值
func TestE2E_TraceTree(t *testing.T) {
	db, cleanup := setupE2eDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := persistence.NewSQLiteConversationRecordRepository(db)
	idGen := &mockIDGenE2E{}
	hook := NewConversationRecordHook(repo, idGen, logger, nil)

	ctx := context.Background()
	traceID := "trace-e2e-simple"

	// 用于保存关键 span ID
	var llmCallSpanID string
	var toolCallSpanID string

	// ========== Step 1: 用户输入 (PreLLMCall) ==========
	// 设置 span 到 Go context
	ctx1 := trace.WithTraceID(ctx, traceID)
	ctx1 = trace.WithSpanID(ctx1, "span-1")
	ctx1 = trace.WithParentSpanID(ctx1, "")
	hookCtx1 := domain.NewHookContext(ctx1)

	callCtx1 := &domain.LLMCallContext{
		TraceID:   traceID,
		SessionID: "session-e2e",
		Prompt:    "帮我执行 ls 命令",
		UserInput: "帮我执行 ls 命令",
		Metadata: map[string]string{
			"session_key":  "session-e2e",
			"user_code":    "user-e2e",
			"agent_code":   "agent-e2e",
			"channel_code": "channel-e2e",
			"channel_type": "feishu",
		},
	}

	_, err := hook.PreLLMCall(hookCtx1, callCtx1)
	if err != nil {
		t.Fatalf("PreLLMCall 失败: %v", err)
	}
	// 从 ctx 中获取生成的 span
	llmCallSpanID = trace.GetSpanID(hookCtx1.Context)

	// ========== Step 2: LLM 响应 (PostLLMCall) ==========
	ctx2 := trace.WithTraceID(ctx, traceID)
	ctx2 = trace.WithSpanID(ctx2, "span-2")
	ctx2 = trace.WithParentSpanID(ctx2, llmCallSpanID) // parent 指向 llm_call
	hookCtx2 := domain.NewHookContext(ctx2)
	// 将 llmCallSpanID 存入 spanKey（模拟 PreLLMCall 的行为）
	hookCtx2 = hookCtx2.WithValue(spanKey, llmCallSpanID)

	resp := &domain.LLMResponse{
		Content: "我将帮你执行 ls 命令",
		Usage: domain.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	_, err = hook.PostLLMCall(hookCtx2, callCtx1, resp)
	if err != nil {
		t.Fatalf("PostLLMCall 失败: %v", err)
	}
	// PostLLMCall 会将新生成的 span 存入 spanKey，我们需要获取它
	llmResponseSpanID, _ := hookCtx2.Get(spanKey).(string)

	// ========== Step 3: 工具调用 (PreToolCall) ==========
	// 注意：ctx3 用于传递 Go context，但 spanKey 的值来自 PostLLMCall 存储
	ctx3 := trace.WithTraceID(ctx, traceID)
	ctx3 = trace.WithSpanID(ctx3, "span-3") // 仅用于测试设置，不用于 spanKey 传递
	ctx3 = trace.WithParentSpanID(ctx3, "span-2")
	hookCtx3 := domain.NewHookContext(ctx3)
	// PostLLMCall 通过 hookCtx2.WithValue(spanKey, spanID) 存储了 llmResponseSpanID
	// PreToolCall 需要从 hookCtx2 传递到 hookCtx3，这里模拟传递
	hookCtx3 = hookCtx3.WithValue(spanKey, llmResponseSpanID) // 关键：传递 llmResponseSpanID
	hookCtx3 = hookCtx3.WithValue(ScopeKey, ScopeInfo{
		SessionKey:  "session-e2e",
		UserCode:    "user-e2e",
		AgentCode:   "agent-e2e",
		ChannelCode: "channel-e2e",
		ChannelType: "feishu",
	})

	callCtx2 := &domain.ToolCallContext{
		TraceID:      traceID,
		ToolName:     "bash",
		ToolInput:    map[string]interface{}{"command": "ls"},
		ParentSpanID: llmResponseSpanID, // 指向 LLM response
	}

	_, err = hook.PreToolCall(hookCtx3, callCtx2)
	if err != nil {
		t.Fatalf("PreToolCall 失败: %v", err)
	}
	// PreToolCall 会生成新的 span 并存入 spanKey
	toolCallSpanID, _ = hookCtx3.Get(spanKey).(string)

	// ========== Step 4: 工具结果 (PostToolCall) ==========
	ctx4 := trace.WithTraceID(ctx, traceID)
	ctx4 = trace.WithSpanID(ctx4, "span-4")
	ctx4 = trace.WithParentSpanID(ctx4, "span-3") // parent 指向 tool_call
	hookCtx4 := domain.NewHookContext(ctx4)
	hookCtx4 = hookCtx4.WithValue(ScopeKey, ScopeInfo{
		SessionKey:  "session-e2e",
		UserCode:    "user-e2e",
		AgentCode:   "agent-e2e",
		ChannelCode: "channel-e2e",
		ChannelType: "feishu",
	})
	// PostToolCall 需要 tool_call 的 span_id
	hookCtx4 = hookCtx4.WithValue(spanKey, toolCallSpanID)

	result := &domain.ToolExecutionResult{
		Success: true,
		Output:  "file1.txt\nfile2.txt",
	}

	_, err = hook.PostToolCall(hookCtx4, callCtx2, result)
	if err != nil {
		t.Fatalf("PostToolCall 失败: %v", err)
	}

	// ========== 验证数据库 ==========
	records, err := repo.FindByTraceID(context.Background(), traceID, 100)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	t.Logf("共 %d 条记录:", len(records))
	for _, r := range records {
		t.Logf("  span=%s, event=%s, role=%s, parent=%s, content=%s",
			r.SpanID(), r.EventType(), r.Role(), r.ParentSpanID(), r.Content())
	}

	if len(records) != 4 {
		t.Errorf("期望 4 条记录，实际 %d 条", len(records))
	}

	// 建立 span -> record 映射
	spanMap := make(map[string]*domain.ConversationRecord)
	for _, r := range records {
		spanMap[r.SpanID()] = r
	}

	// 验证追踪树关系
	// 1. 找到 llm_call (用户输入)
	var llmCallRecord *domain.ConversationRecord
	var llmResponseRecord *domain.ConversationRecord
	var toolCallRecord *domain.ConversationRecord
	var toolResultRecord *domain.ConversationRecord

	for _, r := range records {
		switch r.EventType() {
		case "llm_call":
			llmCallRecord = r
		case "llm_response":
			llmResponseRecord = r
		case "tool_call":
			toolCallRecord = r
		case "tool_result":
			toolResultRecord = r
		}
	}

	// 验证 llm_response 的 parent 是 llm_call
	if llmResponseRecord != nil && llmCallRecord != nil {
		if llmResponseRecord.ParentSpanID() != llmCallRecord.SpanID() {
			t.Errorf("llm_response 的 parent 应为 llm_call 的 span=%s，实际 parent=%s",
				llmCallRecord.SpanID(), llmResponseRecord.ParentSpanID())
		} else {
			t.Logf("✓ llm_response.parent -> llm_call.span (%s)", llmCallRecord.SpanID())
		}
	} else {
		t.Error("缺少 llm_call 或 llm_response 记录")
	}

	// 验证 tool_call 的 parent 是 llm_response
	if toolCallRecord != nil && llmResponseRecord != nil {
		if toolCallRecord.ParentSpanID() != llmResponseRecord.SpanID() {
			t.Errorf("tool_call 的 parent 应为 llm_response 的 span=%s，实际 parent=%s",
				llmResponseRecord.SpanID(), toolCallRecord.ParentSpanID())
		} else {
			t.Logf("✓ tool_call.parent -> llm_response.span (%s)", llmResponseRecord.SpanID())
		}
	} else {
		t.Error("缺少 tool_call 或 llm_response 记录")
	}

	// 验证 tool_result 的 parent 是 tool_call
	if toolResultRecord != nil && toolCallRecord != nil {
		if toolResultRecord.ParentSpanID() != toolCallRecord.SpanID() {
			t.Errorf("tool_result 的 parent 应为 tool_call 的 span=%s，实际 parent=%s",
				toolCallRecord.SpanID(), toolResultRecord.ParentSpanID())
		} else {
			t.Logf("✓ tool_result.parent -> tool_call.span (%s)", toolCallRecord.SpanID())
		}
	} else {
		t.Error("缺少 tool_result 或 tool_call 记录")
	}

	// 验证 Scope 信息
	for _, r := range records {
		if r.SessionKey() != "session-e2e" {
			t.Errorf("SessionKey 应为 session-e2e，实际 %s", r.SessionKey())
		}
		if r.UserCode() != "user-e2e" {
			t.Errorf("UserCode 应为 user-e2e，实际 %s", r.UserCode())
		}
	}

	t.Log("E2E 测试通过！")
}

// TestE2E_LLMResponseWithTools 测试 LLM 返回包含 tool_calls 时记录中间响应
func TestE2E_LLMResponseWithTools(t *testing.T) {
	db, cleanup := setupE2eDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := persistence.NewSQLiteConversationRecordRepository(db)
	idGen := &mockIDGenE2E{}
	hook := NewConversationRecordHook(repo, idGen, logger, nil)

	ctx := context.Background()
	traceID := "trace-llm-with-tools"

	// ========== Step 1: 用户输入 (PreLLMCall) ==========
	ctx1 := trace.WithTraceID(ctx, traceID)
	ctx1 = trace.WithSpanID(ctx1, "span-1")
	ctx1 = trace.WithParentSpanID(ctx1, "")
	hookCtx := domain.NewHookContext(ctx1)

	callCtx1 := &domain.LLMCallContext{
		TraceID:   traceID,
		SessionID: "session-tools",
		Prompt:    "帮我执行 ls 命令",
		UserInput: "帮我执行 ls 命令",
		Metadata: map[string]string{
			"session_key":  "session-tools",
			"user_code":    "user-tools",
			"agent_code":   "agent-tools",
			"channel_code": "channel-tools",
			"channel_type": "feishu",
		},
	}

	_, err := hook.PreLLMCall(hookCtx, callCtx1)
	if err != nil {
		t.Fatalf("PreLLMCall 失败: %v", err)
	}

	// ========== Step 2: LLM 响应，包含 tool_calls (PostLLMCall) ==========
	// 模拟 LLM 返回包含 tool_calls 的情况
	// 注意：在真实系统中，GenerateWithTools 内部会先调用 OnLLMCalledWithTools
	// 然后执行工具，最后调用 OnToolExecutionComplete
	// PostLLMCall 会延迟记录最终的 llm_response
	ctx2 := trace.WithTraceID(ctx, traceID)
	ctx2 = trace.WithSpanID(ctx2, "span-2")
	ctx2 = trace.WithParentSpanID(ctx2, "span-1")
	hookCtx2 := domain.NewHookContext(ctx2)
	hookCtx2 = hookCtx2.WithValue(spanKey, "span-1")
	// 传递 scope
	hookCtx2 = hookCtx2.WithValue(ScopeKey, ScopeInfo{
		SessionKey:  "session-tools",
		UserCode:    "user-tools",
		AgentCode:   "agent-tools",
		ChannelCode: "channel-tools",
		ChannelType: "feishu",
	})

	// 模拟包含 tool_calls 的 RawResponse
	resp := &domain.LLMResponse{
		Content: "我将帮你执行 ls 命令",
		Usage: domain.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		RawResponse: `{"tool_calls":[{"id":"call_123","function":{"name":"bash","arguments":"{\"command\":\"ls\"}"}}]}`,
	}

	_, err = hook.PostLLMCall(hookCtx2, callCtx1, resp)
	if err != nil {
		t.Fatalf("PostLLMCall 失败: %v", err)
	}

	// ========== Step 3: 工具调用 (PreToolCall) ==========
	// 获取 llm_response_with_tools 的 span
	llmWithToolsSpan, _ := hookCtx2.Get(spanKey).(string)

	ctx3 := trace.WithTraceID(ctx, traceID)
	ctx3 = trace.WithSpanID(ctx3, "span-3")
	hookCtx3 := domain.NewHookContext(ctx3)
	hookCtx3 = hookCtx3.WithValue(spanKey, llmWithToolsSpan)
	hookCtx3 = hookCtx3.WithValue(ScopeKey, ScopeInfo{
		SessionKey:  "session-tools",
		UserCode:    "user-tools",
		AgentCode:   "agent-tools",
		ChannelCode: "channel-tools",
		ChannelType: "feishu",
	})

	callCtx2 := &domain.ToolCallContext{
		TraceID:      traceID,
		ToolName:     "bash",
		ToolInput:    map[string]interface{}{"command": "ls"},
		ParentSpanID: llmWithToolsSpan,
	}

	_, err = hook.PreToolCall(hookCtx3, callCtx2)
	if err != nil {
		t.Fatalf("PreToolCall 失败: %v", err)
	}

	// ========== Step 4: 工具结果 (PostToolCall) ==========
	// 获取 tool_call 的 span
	toolCallSpanID, _ := hookCtx3.Get(spanKey).(string)

	ctx4 := trace.WithTraceID(ctx, traceID)
	ctx4 = trace.WithSpanID(ctx4, "span-4")
	hookCtx4 := domain.NewHookContext(ctx4)
	hookCtx4 = hookCtx4.WithValue(spanKey, toolCallSpanID)
	hookCtx4 = hookCtx4.WithValue(ScopeKey, ScopeInfo{
		SessionKey:  "session-tools",
		UserCode:    "user-tools",
		AgentCode:   "agent-tools",
		ChannelCode: "channel-tools",
		ChannelType: "feishu",
	})

	result := &domain.ToolExecutionResult{
		Success: true,
		Output:  "file1.txt\nfile2.txt",
	}

	_, err = hook.PostToolCall(hookCtx4, callCtx2, result)
	if err != nil {
		t.Fatalf("PostToolCall 失败: %v", err)
	}

	// ========== Step 5: 工具执行完成，触发延迟的 llm_response 记录 ==========
	// OnToolExecutionComplete 应该记录最终的 llm_response，parent 是 tool_call 的 span
	// 需要把 deferredResponseKey 从 hookCtx2 传递过来
	hookCtx4 = hookCtx4.WithValue(spanKey, toolCallSpanID)
	hookCtx4 = hookCtx4.WithValue(deferredResponseKey, hookCtx2.Get(deferredResponseKey))

	hook.OnToolExecutionComplete(hookCtx4)

	// ========== 验证数据库 ==========
	records, err := repo.FindByTraceID(context.Background(), traceID, 100)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	t.Logf("共 %d 条记录:", len(records))
	for _, r := range records {
		t.Logf("  span=%s, event=%s, role=%s, parent=%s, content=%s",
			r.SpanID(), r.EventType(), r.Role(), r.ParentSpanID(), r.Content())
	}

	// 应该有 5 条记录：llm_call, llm_response_with_tools, tool_call, tool_result, llm_response
	if len(records) != 5 {
		t.Errorf("期望 5 条记录，实际 %d 条", len(records))
	}

	// 建立 span -> record 映射
	spanMap := make(map[string]*domain.ConversationRecord)
	for _, r := range records {
		spanMap[r.SpanID()] = r
	}

	// 验证 llm_response_with_tools 存在
	var foundWithTools bool
	for _, r := range records {
		if r.EventType() == "llm_response_with_tools" {
			foundWithTools = true
			// 其 parent 应该是 llm_call (span-1)
			if r.ParentSpanID() != "span-1" {
				t.Errorf("llm_response_with_tools 的 parent 应为 span-1，实际 %s", r.ParentSpanID())
			}
			t.Logf("✓ llm_response_with_tools 存在，parent=%s", r.ParentSpanID())
		}
	}

	if !foundWithTools {
		t.Error("缺少 llm_response_with_tools 记录")
	}

	// 验证最终的 llm_response 存在且 parent 正确
	var foundFinalLLMResponse bool
	for _, r := range records {
		if r.EventType() == "llm_response" && r.SpanID() == "span-2" {
			foundFinalLLMResponse = true
			// 最终的 llm_response 的 parent 应该是 tool_call (span-5)
			if r.ParentSpanID() != "span-5" {
				t.Errorf("最终 llm_response 的 parent 应为 span-5，实际 %s", r.ParentSpanID())
			} else {
				t.Logf("✓ 最终 llm_response 存在，parent=%s", r.ParentSpanID())
			}
		}
	}

	if !foundFinalLLMResponse {
		t.Error("缺少最终的 llm_response 记录")
	}

	t.Log("LLM Response With Tools 测试通过！")
}
