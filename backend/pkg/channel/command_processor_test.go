package channel

import (
	"context"
	"testing"

	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// setupCommandProcessorTest 初始化全局 sessionManager
func setupCommandProcessorTest() *CommandProcessor {
	sm := NewSessionManager(zap.NewNop())
	SetSessionManager(sm)
	return NewCommandProcessor(zap.NewNop())
}

func TestCommandProcessor_IsCommand(t *testing.T) {
	cp := NewCommandProcessor(zap.NewNop())

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "有效命令 /help",
			content:  "/help",
			expected: true,
		},
		{
			name:     "有效命令 /status",
			content:  "/status",
			expected: true,
		},
		{
			name:     "有效命令 /new",
			content:  "/new",
			expected: true,
		},
		{
			name:     "有效命令 /stop",
			content:  "/stop",
			expected: true,
		},
		{
			name:     "有效命令 /clear",
			content:  "/clear",
			expected: true,
		},
		{
			name:     "有效命令 /models",
			content:  "/models",
			expected: true,
		},
		{
			name:     "命令带参数",
			content:  "/help foo bar",
			expected: true,
		},
		{
			name:     "普通消息不是命令",
			content:  "Hello, how are you?",
			expected: false,
		},
		{
			name:     "空消息",
			content:  "",
			expected: false,
		},
		{
			name:     "非斜杠开头",
			content:  "help",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cp.IsCommand(tt.content)
			if result != tt.expected {
				t.Errorf("IsCommand(%q) = %v, 期望 %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestCommandProcessor_Process_UnknownCommand(t *testing.T) {
	cp := NewCommandProcessor(zap.NewNop())
	ctx := context.Background()

	msg := &bus.InboundMessage{
		Content: "/unknown",
	}

	result := cp.Process(ctx, msg)

	if result != "未知命令: /unknown" {
		t.Errorf("期望 '未知命令: /unknown', 实际为 %s", result)
	}
}

func TestCommandProcessor_Process_Help(t *testing.T) {
	cp := NewCommandProcessor(zap.NewNop())
	ctx := context.Background()

	msg := &bus.InboundMessage{
		Content: "/help",
	}

	result := cp.Process(ctx, msg)

	if result == "" {
		t.Error("期望非空结果")
	}
	if !contains(result, "可用命令") {
		t.Error("结果应包含 '可用命令'")
	}
	if !contains(result, "/help") {
		t.Error("结果应包含 /help")
	}
	if !contains(result, "/status") {
		t.Error("结果应包含 /status")
	}
}

func TestCommandProcessor_Process_Status(t *testing.T) {
	cp := setupCommandProcessorTest()
	ctx := context.Background()

	msg := &bus.InboundMessage{
		Content: "/status",
		Channel: "test",
		ChatID:  "test-session",
	}

	result := cp.Process(ctx, msg)

	if result == "" {
		t.Error("期望非空结果")
	}
	if !contains(result, "状态") {
		t.Error("结果应包含 '状态'")
	}
}

func TestCommandProcessor_Process_Models(t *testing.T) {
	cp := NewCommandProcessor(zap.NewNop())
	ctx := context.Background()

	msg := &bus.InboundMessage{
		Content: "/models",
	}

	result := cp.Process(ctx, msg)

	if result == "" {
		t.Error("期望非空结果")
	}
	if !contains(result, "支持的模型") {
		t.Error("结果应包含 '支持的模型'")
	}
	if !contains(result, "MiniMax-M2.7-highspeed") {
		t.Error("结果应包含 MiniMax-M2.7-highspeed")
	}
}

func TestCommandProcessor_Process_New(t *testing.T) {
	cp := setupCommandProcessorTest()
	ctx := context.Background()

	msg := &bus.InboundMessage{
		Content: "/new",
	}

	result := cp.Process(ctx, msg)

	if result == "" {
		t.Error("期望非空结果")
	}
	if !contains(result, "新会话") {
		t.Error("结果应包含 '新会话'")
	}
}

func TestCommandProcessor_Process_Stop(t *testing.T) {
	cp := setupCommandProcessorTest()
	ctx := context.Background()

	msg := &bus.InboundMessage{
		Content: "/stop",
	}

	result := cp.Process(ctx, msg)

	if result == "" {
		t.Error("期望非空结果")
	}
}

func TestCommandProcessor_Process_Clear(t *testing.T) {
	cp := setupCommandProcessorTest()
	ctx := context.Background()

	msg := &bus.InboundMessage{
		Content: "/clear",
	}

	result := cp.Process(ctx, msg)

	if result == "" {
		t.Error("期望非空结果")
	}
	if !contains(result, "清除") {
		t.Error("结果应包含 '清除'")
	}
}

func TestCommandProcessor_Register(t *testing.T) {
	cp := NewCommandProcessor(zap.NewNop())

	// 注册新命令
	cp.Register("/test", func(ctx context.Context, msg *bus.InboundMessage, args string) string {
		return "test command executed with args: " + args
	})

	// 验证新命令被识别
	if !cp.IsCommand("/test") {
		t.Error("新注册的命令应该被 IsCommand 识别")
	}

	// 验证命令执行
	ctx := context.Background()
	msg := &bus.InboundMessage{
		Content: "/test myargs",
	}

	result := cp.Process(ctx, msg)
	expected := "test command executed with args: myargs"
	if result != expected {
		t.Errorf("期望 %q, 实际为 %q", expected, result)
	}
}

func TestCommandProcessor_Register_Override(t *testing.T) {
	cp := NewCommandProcessor(zap.NewNop())

	// 注册同名命令（覆盖）
	cp.Register("/help", func(ctx context.Context, msg *bus.InboundMessage, args string) string {
		return "custom help"
	})

	ctx := context.Background()
	msg := &bus.InboundMessage{
		Content: "/help",
	}

	result := cp.Process(ctx, msg)
	if result != "custom help" {
		t.Errorf("期望 'custom help', 实际为 %s", result)
	}
}

func TestCommandProcessor_Process_EmptyContent(t *testing.T) {
	cp := NewCommandProcessor(zap.NewNop())
	ctx := context.Background()

	msg := &bus.InboundMessage{
		Content: "",
	}

	result := cp.Process(ctx, msg)

	if result != "无效命令" {
		t.Errorf("期望 '无效命令', 实际为 %s", result)
	}
}

func TestCommandProcessor_Process_WhitespaceOnly(t *testing.T) {
	cp := NewCommandProcessor(zap.NewNop())
	ctx := context.Background()

	msg := &bus.InboundMessage{
		Content: "   ",
	}

	result := cp.Process(ctx, msg)

	if result != "无效命令" {
		t.Errorf("期望 '无效命令', 实际为 %s", result)
	}
}

func TestCommandProcessor_CaseInsensitive(t *testing.T) {
	cp := NewCommandProcessor(zap.NewNop())
	ctx := context.Background()

	testCases := []string{"/HELP", "/Help", "/hElP", "/help"}

	for _, content := range testCases {
		msg := &bus.InboundMessage{
			Content: content,
		}
		result := cp.Process(ctx, msg)
		if !contains(result, "可用命令") {
			t.Errorf("命令 %q 应该被识别并执行", content)
		}
	}
}

func TestSetSessionManager(t *testing.T) {
	// 测试全局 sessionManager 设置
	sm := &SessionManager{}
	SetSessionManager(sm)

	if sessionManager != sm {
		t.Error("全局 sessionManager 应该被设置")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}