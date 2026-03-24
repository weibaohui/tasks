package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// TestEinoProvider_GenerateWithTools_AppendAssistantToolCallsFirst 验证工具调用的消息顺序：
// 先写入包含 ToolCalls 的 assistant 消息，再写入 tool 消息（携带 tool_call_id）。
func TestEinoProvider_GenerateWithTools_AppendAssistantToolCallsFirst(t *testing.T) {
	fakeModel := &fakeToolCallingChatModel{t: t}
	provider := &EinoProvider{
		config:    &Config{ProviderType: "eino", Model: "fake"},
		chatModel: fakeModel,
		logger:    zap.NewNop(),
	}

	reg := NewToolRegistry()
	reg.Register(&stubTool{
		name:   "bash",
		output: "Tue Mar 24 10:54:43 CST 2026",
	})

	got, toolCalls, err := provider.GenerateWithTools(context.Background(), "请执行 date", []*ToolRegistry{reg}, 3)
	if err != nil {
		t.Fatalf("GenerateWithTools 返回错误: %v", err)
	}
	if got != "最终答案" {
		t.Fatalf("GenerateWithTools 返回内容不符合预期: got=%q", got)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("toolCalls 数量不符合预期: got=%d", len(toolCalls))
	}
	if toolCalls[0].ID != "call_1" || toolCalls[0].Name != "bash" {
		t.Fatalf("toolCalls 内容不符合预期: %+v", toolCalls[0])
	}
}

type fakeToolCallingChatModel struct {
	t     *testing.T
	calls int
}

// Generate 伪造两次调用：第一次返回 ToolCalls，第二次验证消息顺序并返回最终答案。
func (m *fakeToolCallingChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.calls++

	switch m.calls {
	case 1:
		if len(input) != 1 {
			m.t.Fatalf("第一次 Generate 期望 1 条消息，实际=%d", len(input))
		}
		if input[0].Role != schema.User {
			m.t.Fatalf("第一次 Generate 期望 role=user，实际=%s", input[0].Role)
		}

		return &schema.Message{
			Role:    schema.Assistant,
			Content: "",
			ToolCalls: []schema.ToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: schema.FunctionCall{
					Name:      "bash",
					Arguments: `{"command":"date"}`,
				},
			}},
		}, nil
	case 2:
		if len(input) != 3 {
			m.t.Fatalf("第二次 Generate 期望 3 条消息（user + assistant(tool_calls) + tool），实际=%d", len(input))
		}
		if input[0].Role != schema.User {
			m.t.Fatalf("第二次 Generate 第 1 条消息期望 role=user，实际=%s", input[0].Role)
		}
		if input[1].Role != schema.Assistant || len(input[1].ToolCalls) != 1 || input[1].ToolCalls[0].ID != "call_1" {
			m.t.Fatalf("第二次 Generate 第 2 条消息期望包含 tool_calls(call_1)，实际=%+v", input[1])
		}
		if input[2].Role != schema.Tool || input[2].ToolCallID != "call_1" {
			m.t.Fatalf("第二次 Generate 第 3 条消息期望 role=tool 且 tool_call_id=call_1，实际=%+v", input[2])
		}

		return &schema.Message{
			Role:    schema.Assistant,
			Content: "最终答案",
		}, nil
	default:
		return nil, fmt.Errorf("fakeToolCallingChatModel 收到超出预期的 Generate 调用次数: %d", m.calls)
	}
}

// Stream 在该单元测试里不使用。
func (m *fakeToolCallingChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("not implemented")
}

// WithTools 返回自身即可，满足 EinoProvider.GenerateWithTools 的绑定流程。
func (m *fakeToolCallingChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

type stubTool struct {
	name   string
	output string
}

// Name 返回工具名称。
func (t *stubTool) Name() string {
	return t.name
}

// Description 返回工具描述。
func (t *stubTool) Description() string {
	return "测试用工具"
}

// Parameters 返回工具参数 schema（测试不关注，返回 nil）。
func (t *stubTool) Parameters() json.RawMessage {
	return nil
}

// Execute 返回固定输出，模拟工具执行成功。
func (t *stubTool) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	return &ToolResult{
		Output: t.output,
	}, nil
}

