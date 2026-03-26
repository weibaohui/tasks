/**
 * MessageBus 单元测试
 */
package bus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewMessageBus(t *testing.T) {
	bus := NewMessageBus(nil)
	if bus == nil {
		t.Fatal("NewMessageBus 不应返回 nil")
	}
	if bus.InboundSize() != 0 {
		t.Errorf("期望 inbound 初始大小为 0, 实际为 %d", bus.InboundSize())
	}
	if bus.OutboundSize() != 0 {
		t.Errorf("期望 outbound 初始大小为 0, 实际为 %d", bus.OutboundSize())
	}
	if bus.StreamSize() != 0 {
		t.Errorf("期望 stream 初始大小为 0, 实际为 %d", bus.StreamSize())
	}
}

func TestMessageBus_PublishInbound(t *testing.T) {
	bus := NewMessageBus(nil)
	msg := NewInboundMessage("feishu", "user-001", "chat-001", "hello")

	bus.PublishInbound(msg)

	if bus.InboundSize() != 1 {
		t.Errorf("期望 inbound 大小为 1, 实际为 %d", bus.InboundSize())
	}
}

func TestMessageBus_ConsumeInbound(t *testing.T) {
	bus := NewMessageBus(nil)
	msg := NewInboundMessage("feishu", "user-001", "chat-001", "hello")
	bus.PublishInbound(msg)

	ctx := context.Background()
	consumed, err := bus.ConsumeInbound(ctx)
	if err != nil {
		t.Fatalf("ConsumeInbound 失败: %v", err)
	}

	if consumed.Content != "hello" {
		t.Errorf("期望 content 为 hello, 实际为 %s", consumed.Content)
	}

	if bus.InboundSize() != 0 {
		t.Errorf("期望 inbound 大小为 0, 实际为 %d", bus.InboundSize())
	}
}

func TestMessageBus_ConsumeInbound_ContextCancel(t *testing.T) {
	bus := NewMessageBus(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := bus.ConsumeInbound(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("期望 context.DeadlineExceeded, 实际为 %v", err)
	}
}

func TestMessageBus_PublishOutbound(t *testing.T) {
	bus := NewMessageBus(nil)
	msg := NewOutboundMessage("feishu", "chat-001", "response")

	bus.PublishOutbound(msg)

	if bus.OutboundSize() != 1 {
		t.Errorf("期望 outbound 大小为 1, 实际为 %d", bus.OutboundSize())
	}
}

func TestMessageBus_ConsumeOutbound(t *testing.T) {
	bus := NewMessageBus(nil)
	msg := NewOutboundMessage("feishu", "chat-001", "response")
	bus.PublishOutbound(msg)

	ctx := context.Background()
	consumed, err := bus.ConsumeOutbound(ctx)
	if err != nil {
		t.Fatalf("ConsumeOutbound 失败: %v", err)
	}

	if consumed.Content != "response" {
		t.Errorf("期望 content 为 response, 实际为 %s", consumed.Content)
	}
}

func TestMessageBus_PublishStream(t *testing.T) {
	bus := NewMessageBus(nil)
	chunk := NewStreamChunk("feishu", "chat-001", "hello", "hello", false)

	bus.PublishStream(chunk)

	if bus.StreamSize() != 1 {
		t.Errorf("期望 stream 大小为 1, 实际为 %d", bus.StreamSize())
	}
}

func TestMessageBus_SubscribeOutbound(t *testing.T) {
	bus := NewMessageBus(nil)
	var wg sync.WaitGroup
	wg.Add(1)

	callback := func(msg *OutboundMessage) error {
		if msg.Content != "test" {
			t.Errorf("期望 content 为 test, 实际为 %s", msg.Content)
		}
		wg.Done()
		return nil
	}

	bus.SubscribeOutbound("feishu", callback)

	ctx := context.Background()
	bus.StartDispatcher(ctx)
	defer bus.Stop()

	bus.PublishOutbound(NewOutboundMessage("feishu", "chat-001", "test"))

	// Wait for callback with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("回调未在预期时间内执行")
	}
}

func TestMessageBus_SubscribeStream(t *testing.T) {
	bus := NewMessageBus(nil)
	var wg sync.WaitGroup
	wg.Add(1)

	callback := func(chunk *StreamChunk) error {
		if chunk.Delta != "hello" {
			t.Errorf("期望 delta 为 hello, 实际为 %s", chunk.Delta)
		}
		wg.Done()
		return nil
	}

	bus.SubscribeStream("feishu", callback)

	ctx := context.Background()
	bus.StartDispatcher(ctx)
	defer bus.Stop()

	bus.PublishStream(NewStreamChunk("feishu", "chat-001", "hello", "hello", false))

	// Wait for callback with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("回调未在预期时间内执行")
	}
}

func TestMessageBus_Stop(t *testing.T) {
	bus := NewMessageBus(nil)
	ctx := context.Background()
	bus.StartDispatcher(ctx)
	bus.Stop()
	// No panic means success
}

func TestNewInboundMessage(t *testing.T) {
	msg := NewInboundMessage("feishu", "user-001", "chat-001", "hello")

	if msg.Channel != "feishu" {
		t.Errorf("期望 channel 为 feishu, 实际为 %s", msg.Channel)
	}
	if msg.SenderID != "user-001" {
		t.Errorf("期望 sender_id 为 user-001, 实际为 %s", msg.SenderID)
	}
	if msg.ChatID != "chat-001" {
		t.Errorf("期望 chat_id 为 chat-001, 实际为 %s", msg.ChatID)
	}
	if msg.Content != "hello" {
		t.Errorf("期望 content 为 hello, 实际为 %s", msg.Content)
	}
	if len(msg.Media) != 0 {
		t.Errorf("期望 media 为空, 实际为 %v", msg.Media)
	}
	if msg.Metadata == nil {
		t.Error("期望 metadata 不为 nil")
	}
}

func TestNewOutboundMessage(t *testing.T) {
	msg := NewOutboundMessage("feishu", "chat-001", "response")

	if msg.Channel != "feishu" {
		t.Errorf("期望 channel 为 feishu, 实际为 %s", msg.Channel)
	}
	if msg.ChatID != "chat-001" {
		t.Errorf("期望 chat_id 为 chat-001, 实际为 %s", msg.ChatID)
	}
	if msg.Content != "response" {
		t.Errorf("期望 content 为 response, 实际为 %s", msg.Content)
	}
}

func TestNewStreamChunk(t *testing.T) {
	chunk := NewStreamChunk("feishu", "chat-001", "hello", "hello world", true)

	if chunk.Channel != "feishu" {
		t.Errorf("期望 channel 为 feishu, 实际为 %s", chunk.Channel)
	}
	if chunk.ChatID != "chat-001" {
		t.Errorf("期望 chat_id 为 chat-001, 实际为 %s", chunk.ChatID)
	}
	if chunk.Delta != "hello" {
		t.Errorf("期望 delta 为 hello, 实际为 %s", chunk.Delta)
	}
	if chunk.Content != "hello world" {
		t.Errorf("期望 content 为 hello world, 实际为 %s", chunk.Content)
	}
	if !chunk.Done {
		t.Error("期望 done 为 true")
	}
}

func TestInboundMessage_SessionKey(t *testing.T) {
	msg := NewInboundMessage("feishu", "user-001", "chat-001", "hello")

	sk := msg.SessionKey()
	if sk != "feishu:chat-001" {
		t.Errorf("期望 session_key 为 feishu:chat-001, 实际为 %s", sk)
	}
}