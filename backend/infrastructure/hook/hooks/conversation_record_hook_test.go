/**
 * ConversationRecordHook 单独方法测试
 * 每个测试只验证一个方法的行为
 */
package hooks

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/persistence"
	"go.uber.org/zap"
)

// mockIDGen 简单的 ID 生成器
type mockIDGen struct {
	nextID int
}

func (m *mockIDGen) Generate() string {
	id := m.nextID
	m.nextID++
	return "span-" + string(rune('0'+id))
}

func newMockIDGen() *mockIDGen {
	return &mockIDGen{nextID: 0}
}

func setupHookTestDB(t *testing.T) (*sql.DB, func()) {
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

// TestPreLLMCall_Basic 测试 PreLLMCall 基本功能
func TestPreLLMCall_Basic(t *testing.T) {
	db, cleanup := setupHookTestDB(t)
	defer cleanup()

	logger := zap.NewNop()
	idGen := newMockIDGen()
	repo := persistence.NewSQLiteConversationRecordRepository(db)
	hook := NewConversationRecordHook(repo, idGen, logger, nil)

	ctx := context.Background()
	hookCtx := domain.NewHookContext(ctx)

	callCtx := &domain.LLMCallContext{
		TraceID:   "trace-123",
		SessionID: "session-abc",
		Prompt:    "你好",
		UserInput: "你好",
		Metadata: map[string]string{
			"session_key":  "session-abc",
			"user_code":    "user-001",
			"agent_code":   "agent-002",
			"channel_code": "channel-003",
			"channel_type": "feishu",
		},
	}

	result, err := hook.PreLLMCall(hookCtx, callCtx)
	if err != nil {
		t.Fatalf("PreLLMCall 失败: %v", err)
	}
	if result == nil {
		t.Fatal("结果不应为空")
	}

	// 验证记录是否创建
	records, err := repo.FindByTraceID(ctx, "trace-123", 10)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("期望 1 条记录，实际 %d 条", len(records))
	}

	record := records[0]
	if record.EventType() != "llm_call" {
		t.Errorf("期望 event_type 为 llm_call，实际为 %s", record.EventType())
	}
	if record.Role() != "user" {
		t.Errorf("期望 role 为 user，实际为 %s", record.Role())
	}
	if record.Content() != "你好" {
		t.Errorf("期望 content 为 你好，实际为 %s", record.Content())
	}
	if record.SessionKey() != "session-abc" {
		t.Errorf("期望 session_key 为 session-abc，实际为 %s", record.SessionKey())
	}
	if record.AgentCode() != "agent-002" {
		t.Errorf("期望 agent_code 为 agent-002，实际为 %s", record.AgentCode())
	}
}

// TestPostLLMCall_Basic 测试 PostLLMCall 基本功能
func TestPostLLMCall_Basic(t *testing.T) {
	db, cleanup := setupHookTestDB(t)
	defer cleanup()

	logger := zap.NewNop()
	idGen := newMockIDGen()
	repo := persistence.NewSQLiteConversationRecordRepository(db)
	hook := NewConversationRecordHook(repo, idGen, logger, nil)

	ctx := context.Background()
	hookCtx := domain.NewHookContext(ctx)

	// 先调用 PreLLMCall
	callCtx := &domain.LLMCallContext{
		TraceID:   "trace-456",
		SessionID: "session-xyz",
		Prompt:    "你好",
		UserInput: "你好",
		Metadata: map[string]string{
			"session_key": "session-xyz",
		},
	}
	hook.PreLLMCall(hookCtx, callCtx)

	// 调用 PostLLMCall
	resp := &domain.LLMResponse{
		Content: "你好，我是助手",
		Usage: domain.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	result, err := hook.PostLLMCall(hookCtx, callCtx, resp)
	if err != nil {
		t.Fatalf("PostLLMCall 失败: %v", err)
	}
	if result == nil {
		t.Fatal("结果不应为空")
	}

	// 验证记录
	records, _ := repo.FindByTraceID(ctx, "trace-456", 10)
	// 应该有两条记录：user input 和 assistant response
	if len(records) != 2 {
		t.Fatalf("期望 2 条记录，实际 %d 条", len(records))
	}

	// 找到 assistant 的记录
	var assistantRecord *domain.ConversationRecord
	for _, r := range records {
		if r.Role() == "assistant" {
			assistantRecord = r
			break
		}
	}
	if assistantRecord == nil {
		t.Fatal("找不到 assistant 记录")
	}
	if assistantRecord.Content() != "你好，我是助手" {
		t.Errorf("期望 content 为 你好，我是助手，实际为 %s", assistantRecord.Content())
	}
	if assistantRecord.PromptTokens() != 10 {
		t.Errorf("期望 prompt_tokens 为 10，实际为 %d", assistantRecord.PromptTokens())
	}
	if assistantRecord.TotalTokens() != 30 {
		t.Errorf("期望 total_tokens 为 30，实际为 %d", assistantRecord.TotalTokens())
	}
}

// TestPreToolCall_Basic 测试 PreToolCall 基本功能
func TestPreToolCall_Basic(t *testing.T) {
	db, cleanup := setupHookTestDB(t)
	defer cleanup()

	logger := zap.NewNop()
	idGen := newMockIDGen()
	repo := persistence.NewSQLiteConversationRecordRepository(db)
	hook := NewConversationRecordHook(repo, idGen, logger, nil)

	ctx := context.Background()
	hookCtx := domain.NewHookContext(ctx)

	// 设置 scope（模拟真实环境中 toolHookAdapter 的行为）
	hookCtx = hookCtx.WithValue(scopeKey, scopeInfo{
		SessionKey:  "session-tool",
		UserCode:    "user-tool",
		AgentCode:   "agent-tool",
		ChannelCode: "channel-tool",
		ChannelType: "feishu",
	})

	callCtx := &domain.ToolCallContext{
		TraceID:      "trace-tool",
		ToolName:     "bash",
		ToolInput:    map[string]interface{}{"command": "ls"},
		ParentSpanID: "span-parent",
	}

	result, err := hook.PreToolCall(hookCtx, callCtx)
	if err != nil {
		t.Fatalf("PreToolCall 失败: %v", err)
	}
	if result == nil {
		t.Fatal("结果不应为空")
	}

	// 验证记录
	records, _ := repo.FindByTraceID(ctx, "trace-tool", 10)
	if len(records) != 1 {
		t.Fatalf("期望 1 条记录，实际 %d 条", len(records))
	}

	record := records[0]
	if record.EventType() != "tool_call" {
		t.Errorf("期望 event_type 为 tool_call，实际为 %s", record.EventType())
	}
	if record.Role() != "tool" {
		t.Errorf("期望 role 为 tool，实际为 %s", record.Role())
	}
	if record.ParentSpanID() != "span-parent" {
		t.Errorf("期望 parent_span_id 为 span-parent，实际为 %s", record.ParentSpanID())
	}
	// 验证 scope
	if record.SessionKey() != "session-tool" {
		t.Errorf("期望 session_key 为 session-tool，实际为 %s", record.SessionKey())
	}
}

// TestPostToolCall_Success 测试 PreToolCall + PostToolCall 成功流程
func TestPostToolCall_Success(t *testing.T) {
	db, cleanup := setupHookTestDB(t)
	defer cleanup()

	logger := zap.NewNop()
	idGen := newMockIDGen()
	repo := persistence.NewSQLiteConversationRecordRepository(db)
	hook := NewConversationRecordHook(repo, idGen, logger, nil)

	ctx := context.Background()
	hookCtx := domain.NewHookContext(ctx)

	// 设置 scope
	hookCtx = hookCtx.WithValue(scopeKey, scopeInfo{
		SessionKey:  "session-post",
		UserCode:    "user-post",
		AgentCode:   "agent-post",
		ChannelCode: "channel-post",
		ChannelType: "feishu",
	})

	callCtx := &domain.ToolCallContext{
		TraceID:      "trace-post",
		ToolName:     "bash",
		ToolInput:    map[string]interface{}{"command": "ls"},
		ParentSpanID: "span-llm",
	}

	// PreToolCall
	hook.PreToolCall(hookCtx, callCtx)

	// PostToolCall
	result := &domain.ToolExecutionResult{
		Success: true,
		Output:  "file1.txt\nfile2.txt",
	}

	_, err := hook.PostToolCall(hookCtx, callCtx, result)
	if err != nil {
		t.Fatalf("PostToolCall 失败: %v", err)
	}

	// 验证记录 - 应该有 tool_call 和 tool_result 两条
	records, _ := repo.FindByTraceID(ctx, "trace-post", 10)
	if len(records) != 2 {
		t.Fatalf("期望 2 条记录，实际 %d 条", len(records))
	}

	// 验证 tool_result
	var toolResult *domain.ConversationRecord
	for _, r := range records {
		if r.EventType() == "tool_result" {
			toolResult = r
			break
		}
	}
	if toolResult == nil {
		t.Fatal("找不到 tool_result 记录")
	}
	if toolResult.Content() != "file1.txt\nfile2.txt" {
		t.Errorf("期望 content 为 file1.txt\\nfile2.txt，实际为 %s", toolResult.Content())
	}
}

// TestOnToolError_Basic 测试 OnToolError 基本功能
func TestOnToolError_Basic(t *testing.T) {
	db, cleanup := setupHookTestDB(t)
	defer cleanup()

	logger := zap.NewNop()
	idGen := newMockIDGen()
	repo := persistence.NewSQLiteConversationRecordRepository(db)
	hook := NewConversationRecordHook(repo, idGen, logger, nil)

	ctx := context.Background()
	hookCtx := domain.NewHookContext(ctx)

	// 设置 scope
	hookCtx = hookCtx.WithValue(scopeKey, scopeInfo{
		SessionKey:  "session-err",
		UserCode:    "user-err",
		AgentCode:   "agent-err",
		ChannelCode: "channel-err",
		ChannelType: "feishu",
	})

	callCtx := &domain.ToolCallContext{
		TraceID:      "trace-error",
		ToolName:     "bash",
		ToolInput:    map[string]interface{}{"command": "ls"},
		ParentSpanID: "span-parent",
	}

	// PreToolCall
	hook.PreToolCall(hookCtx, callCtx)

	// OnToolError
	execErr := &testError{msg: "command not found"}
	result, err := hook.OnToolError(hookCtx, callCtx, execErr)
	if err != nil {
		t.Fatalf("OnToolError 失败: %v", err)
	}
	if result == nil {
		t.Fatal("结果不应为空")
	}
	if result.Success {
		t.Error("期望 Success 为 false")
	}

	// 验证记录 - 应该有 tool_call 和 tool_error 两条
	records, _ := repo.FindByTraceID(ctx, "trace-error", 10)
	if len(records) != 2 {
		t.Fatalf("期望 2 条记录，实际 %d 条", len(records))
	}

	// 验证 tool_error
	var toolError *domain.ConversationRecord
	for _, r := range records {
		if r.EventType() == "tool_error" {
			toolError = r
			break
		}
	}
	if toolError == nil {
		t.Fatal("找不到 tool_error 记录")
	}
	if toolError.Role() != "tool_error" {
		t.Errorf("期望 role 为 tool_error，实际为 %s", toolError.Role())
	}
}

// testError 模拟错误
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
