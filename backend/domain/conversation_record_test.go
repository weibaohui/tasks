/**
 * ConversationRecord Domain 实体单元测试
 */
package domain

import (
	"testing"
	"time"
)

func TestNewConversationRecord(t *testing.T) {
	id := NewConversationRecordID("record-1")
	traceID := "trace-abc"

	record, err := NewConversationRecord(id, traceID, "llm_call")
	if err != nil {
		t.Fatalf("创建 ConversationRecord 失败: %v", err)
	}

	if record.ID() != id {
		t.Errorf("期望 ID 为 %s，实际为 %s", id, record.ID())
	}
	if record.TraceID() != traceID {
		t.Errorf("期望 TraceID 为 %s，实际为 %s", traceID, record.TraceID())
	}
	if record.EventType() != "llm_call" {
		t.Errorf("期望 EventType 为 llm_call，实际为 %s", record.EventType())
	}
	if record.Timestamp().IsZero() {
		t.Error("Timestamp 不应为零")
	}
}

func TestNewConversationRecord_InvalidID(t *testing.T) {
	_, err := NewConversationRecord(NewConversationRecordID(""), "trace-1", "llm_call")
	if err != ErrConversationRecordIDRequired {
		t.Errorf("期望 ErrConversationRecordIDRequired，实际为 %v", err)
	}
}

func TestNewConversationRecord_InvalidTraceID(t *testing.T) {
	_, err := NewConversationRecord(NewConversationRecordID("id-1"), "", "llm_call")
	if err != ErrConversationRecordTraceIDRequired {
		t.Errorf("期望 ErrConversationRecordTraceIDRequired，实际为 %v", err)
	}
}

func TestNewConversationRecord_InvalidEventType(t *testing.T) {
	_, err := NewConversationRecord(NewConversationRecordID("id-1"), "trace-1", "")
	if err != ErrConversationRecordEventTypeRequired {
		t.Errorf("期望 ErrConversationRecordEventTypeRequired，实际为 %v", err)
	}
}

func TestConversationRecord_SetSpan(t *testing.T) {
	record, _ := NewConversationRecord(NewConversationRecordID("id-1"), "trace-1", "llm_call")

	record.SetSpan("span-123", "parent-456")

	if record.SpanID() != "span-123" {
		t.Errorf("期望 SpanID 为 span-123，实际为 %s", record.SpanID())
	}
	if record.ParentSpanID() != "parent-456" {
		t.Errorf("期望 ParentSpanID 为 parent-456，实际为 %s", record.ParentSpanID())
	}
}

func TestConversationRecord_SetMessage(t *testing.T) {
	record, _ := NewConversationRecord(NewConversationRecordID("id-1"), "trace-1", "llm_call")

	record.SetMessage("user", "你好")

	if record.Role() != "user" {
		t.Errorf("期望 Role 为 user，实际为 %s", record.Role())
	}
	if record.Content() != "你好" {
		t.Errorf("期望 Content 为 你好，实际为 %s", record.Content())
	}
}

func TestConversationRecord_SetScope(t *testing.T) {
	record, _ := NewConversationRecord(NewConversationRecordID("id-1"), "trace-1", "llm_call")

	record.SetScope("session-1", "user-1", "agent-1", "channel-1", "feishu")

	if record.SessionKey() != "session-1" {
		t.Errorf("期望 SessionKey 为 session-1，实际为 %s", record.SessionKey())
	}
	if record.UserCode() != "user-1" {
		t.Errorf("期望 UserCode 为 user-1，实际为 %s", record.UserCode())
	}
	if record.AgentCode() != "agent-1" {
		t.Errorf("期望 AgentCode 为 agent-1，实际为 %s", record.AgentCode())
	}
	if record.ChannelCode() != "channel-1" {
		t.Errorf("期望 ChannelCode 为 channel-1，实际为 %s", record.ChannelCode())
	}
	if record.ChannelType() != "feishu" {
		t.Errorf("期望 ChannelType 为 feishu，实际为 %s", record.ChannelType())
	}
}

func TestConversationRecord_SetTokenUsage(t *testing.T) {
	record, _ := NewConversationRecord(NewConversationRecordID("id-1"), "trace-1", "llm_response")

	record.SetTokenUsage(100, 50, 150, 10, 5)

	if record.PromptTokens() != 100 {
		t.Errorf("期望 PromptTokens 为 100，实际为 %d", record.PromptTokens())
	}
	if record.CompletionTokens() != 50 {
		t.Errorf("期望 CompletionTokens 为 50，实际为 %d", record.CompletionTokens())
	}
	if record.TotalTokens() != 150 {
		t.Errorf("期望 TotalTokens 为 150，实际为 %d", record.TotalTokens())
	}
	if record.ReasoningTokens() != 10 {
		t.Errorf("期望 ReasoningTokens 为 10，实际为 %d", record.ReasoningTokens())
	}
	if record.CachedTokens() != 5 {
		t.Errorf("期望 CachedTokens 为 5，实际为 %d", record.CachedTokens())
	}
}

func TestConversationRecord_ToSnapshot(t *testing.T) {
	record, _ := NewConversationRecord(NewConversationRecordID("id-1"), "trace-1", "llm_call")
	record.SetSpan("span-1", "")
	record.SetMessage("user", "hello")
	record.SetScope("session-1", "user-1", "agent-1", "channel-1", "feishu")
	record.SetTokenUsage(10, 20, 30, 0, 0)

	snap := record.ToSnapshot()

	if snap.ID != record.ID() {
		t.Errorf("Snapshot ID 不匹配")
	}
	if snap.TraceID != "trace-1" {
		t.Errorf("Snapshot TraceID 不匹配")
	}
	if snap.SpanID != "span-1" {
		t.Errorf("Snapshot SpanID 不匹配")
	}
	if snap.Role != "user" {
		t.Errorf("Snapshot Role 不匹配")
	}
	if snap.Content != "hello" {
		t.Errorf("Snapshot Content 不匹配")
	}
	if snap.SessionKey != "session-1" {
		t.Errorf("Snapshot SessionKey 不匹配")
	}
}

func TestConversationRecord_FromSnapshot(t *testing.T) {
	snap := ConversationRecordSnapshot{
		ID:           NewConversationRecordID("id-1"),
		TraceID:      "trace-1",
		SpanID:       "span-1",
		ParentSpanID: "parent-1",
		EventType:    "llm_call",
		Timestamp:    time.Now(),
		SessionKey:   "session-1",
		Role:         "user",
		Content:      "hello",
		PromptTokens: 10,
		TotalTokens:  30,
		UserCode:     "user-1",
		AgentCode:    "agent-1",
		ChannelCode:  "channel-1",
		ChannelType:  "feishu",
	}

	record := &ConversationRecord{}
	record.FromSnapshot(snap)

	if record.ID() != snap.ID {
		t.Errorf("ID 不匹配")
	}
	if record.TraceID() != snap.TraceID {
		t.Errorf("TraceID 不匹配")
	}
	if record.SpanID() != snap.SpanID {
		t.Errorf("SpanID 不匹配")
	}
	if record.Role() != snap.Role {
		t.Errorf("Role 不匹配")
	}
	if record.Content() != snap.Content {
		t.Errorf("Content 不匹配")
	}
}
