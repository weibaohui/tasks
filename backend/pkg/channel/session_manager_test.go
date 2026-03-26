/**
 * SessionManager 单元测试
 */
package channel

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	session := NewSession("test-key", nil)
	if session == nil {
		t.Fatal("NewSession 不应返回 nil")
	}
	if session.Key() != "test-key" {
		t.Errorf("期望 key 为 test-key, 实际为 %s", session.Key())
	}
	if len(session.Messages()) != 0 {
		t.Errorf("期望消息为空, 实际为 %d", len(session.Messages()))
	}
}

func TestSession_AddMessage(t *testing.T) {
	session := NewSession("test-key", nil)

	session.AddMessage(Message{
		Role:    "user",
		Content: "hello",
	})

	msgs := session.Messages()
	if len(msgs) != 1 {
		t.Errorf("期望 1 条消息, 实际为 %d", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Errorf("期望 content 为 hello, 实际为 %s", msgs[0].Content)
	}
	if msgs[0].Role != "user" {
		t.Errorf("期望 role 为 user, 实际为 %s", msgs[0].Role)
	}
}

func TestSession_AddMessage_DefaultTimestamp(t *testing.T) {
	session := NewSession("test-key", nil)
	before := time.Now()

	session.AddMessage(Message{
		Role:    "user",
		Content: "hello",
	})

	msgs := session.Messages()
	if msgs[0].Timestamp.Before(before) {
		t.Error("时间戳应该在添加之后或相等")
	}
}

func TestSession_Messages_ReturnsCopy(t *testing.T) {
	session := NewSession("test-key", nil)
	session.AddMessage(Message{Role: "user", Content: "hello"})

	msgs1 := session.Messages()
	msgs2 := session.Messages()

	if msgs1[0].Content != msgs2[0].Content {
		t.Error("两次获取的消息应该相同")
	}

	// 修改一个不应该影响另一个
	msgs1[0].Content = "modified"
	if msgs2[0].Content == "modified" {
		t.Error("Messages 应返回拷贝，不应受外部修改影响")
	}
}

func TestSession_SetContext(t *testing.T) {
	session := NewSession("test-key", nil)
	ctx, cancel := context.WithCancel(context.Background())

	session.SetContext(ctx, cancel)

	if session.Context() != ctx {
		t.Error("Context 不匹配")
	}
}

func TestSession_Cancel(t *testing.T) {
	session := NewSession("test-key", nil)
	ctx, cancel := context.WithCancel(context.Background())
	session.SetContext(ctx, cancel)

	session.Cancel()

	// Context should be cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

func TestSession_CreatedAt(t *testing.T) {
	before := time.Now()
	session := NewSession("test-key", nil)
	after := time.Now()

	if session.CreatedAt().Before(before) || session.CreatedAt().After(after) {
		t.Error("CreatedAt 不在预期范围内")
	}
}

func TestNewSessionManager(t *testing.T) {
	manager := NewSessionManager(nil)
	if manager == nil {
		t.Fatal("NewSessionManager 不应返回 nil")
	}
	if manager.Count() != 0 {
		t.Errorf("期望计数为 0, 实际为 %d", manager.Count())
	}
}

func TestSessionManager_GetOrCreate(t *testing.T) {
	manager := NewSessionManager(nil)

	session1 := manager.GetOrCreate("key-1")
	session2 := manager.GetOrCreate("key-1")

	// Same key should return same session
	if session1 != session2 {
		t.Error("相同 key 应返回相同 session")
	}

	if manager.Count() != 1 {
		t.Errorf("期望计数为 1, 实际为 %d", manager.Count())
	}
}

func TestSessionManager_Get(t *testing.T) {
	manager := NewSessionManager(nil)
	manager.GetOrCreate("key-1")

	session := manager.Get("key-1")
	if session == nil {
		t.Error("应返回 session")
	}

	nonexistent := manager.Get("nonexistent")
	if nonexistent != nil {
		t.Error("不应返回不存在的 session")
	}
}

func TestSessionManager_Delete(t *testing.T) {
	manager := NewSessionManager(nil)
	manager.GetOrCreate("key-1")

	manager.Delete("key-1")

	if manager.Count() != 0 {
		t.Errorf("期望计数为 0, 实际为 %d", manager.Count())
	}
}

func TestSessionManager_Delete_Nonexistent(t *testing.T) {
	manager := NewSessionManager(nil)
	// Should not panic
	manager.Delete("nonexistent")
}

func TestSessionManager_List(t *testing.T) {
	manager := NewSessionManager(nil)
	manager.GetOrCreate("key-1")
	manager.GetOrCreate("key-2")
	manager.GetOrCreate("key-3")

	keys := manager.List()
	if len(keys) != 3 {
		t.Errorf("期望 3 个 keys, 实际为 %d", len(keys))
	}
}

func TestSessionManager_CleanupInactive(t *testing.T) {
	manager := NewSessionManager(nil)
	manager.GetOrCreate("key-1")

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Should cleanup sessions older than 5ms
	removed := manager.CleanupInactive(5 * time.Millisecond)
	if removed != 1 {
		t.Errorf("期望清理 1 个 session, 实际为 %d", removed)
	}
}

func TestSessionManager_CleanupInactive_NoneRemoved(t *testing.T) {
	manager := NewSessionManager(nil)
	manager.GetOrCreate("key-1")

	// Should not cleanup - session is still active
	removed := manager.CleanupInactive(time.Hour)
	if removed != 0 {
		t.Errorf("期望清理 0 个 session, 实际为 %d", removed)
	}
}

func TestSessionManager_ConcurrentAccess(t *testing.T) {
	manager := NewSessionManager(nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			manager.GetOrCreate("key")
			wg.Done()
		}(i)
	}
	wg.Wait()

	if manager.Count() != 1 {
		t.Errorf("期望 1 个 session, 实际为 %d", manager.Count())
	}
}

func TestMessage_Fields(t *testing.T) {
	now := time.Now()
	msg := Message{
		Role:      "user",
		Content:   "test content",
		Timestamp: now,
		TraceID:   "trace-001",
		SpanID:    "span-001",
	}

	if msg.Role != "user" {
		t.Errorf("期望 Role 为 user, 实际为 %s", msg.Role)
	}
	if msg.Content != "test content" {
		t.Errorf("期望 Content 为 test content, 实际为 %s", msg.Content)
	}
	if msg.Timestamp != now {
		t.Errorf("Timestamp 不匹配")
	}
	if msg.TraceID != "trace-001" {
		t.Errorf("期望 TraceID 为 trace-001, 实际为 %s", msg.TraceID)
	}
	if msg.SpanID != "span-001" {
		t.Errorf("期望 SpanID 为 span-001, 实际为 %s", msg.SpanID)
	}
}