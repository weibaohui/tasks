/**
 * EventBus 单元测试
 */
package bus

import (
	"sync"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type mockTask struct {
	id      domain.TaskID
	traceID domain.TraceID
	status  domain.TaskStatus
}

func (m *mockTask) ID() domain.TaskID         { return m.id }
func (m *mockTask) TraceID() domain.TraceID   { return m.traceID }
func (m *mockTask) Status() domain.TaskStatus { return m.status }
func (m *mockTask) StartedAt() *time.Time     { return nil }
func (m *mockTask) FinishedAt() *time.Time    { return nil }

type mockEvent struct {
	task      *mockTask
	eventType string
	timestamp int64
}

func (e *mockEvent) EventType() string       { return e.eventType }
func (e *mockEvent) TraceID() domain.TraceID { return e.task.traceID }
func (e *mockEvent) Timestamp() int64        { return e.timestamp }

func TestEventBus_Subscribe(t *testing.T) {
	eb := NewEventBus()

	var mu sync.Mutex
	eventReceived := false

	handler := func(event domain.DomainEvent) {
		mu.Lock()
		eventReceived = true
		mu.Unlock()
	}

	eb.Subscribe("TaskCreated", handler)

	task := &mockTask{
		id:      domain.NewTaskID("test-1"),
		traceID: domain.NewTraceID("trace-1"),
	}
	event := &mockEvent{task: task, eventType: "TaskCreated", timestamp: time.Now().Unix()}

	eb.Publish(event)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !eventReceived {
		t.Error("期望事件被接收，但实际没有")
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	eb := NewEventBus()

	var mu sync.Mutex
	callCount := 0

	handler := func(event domain.DomainEvent) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	eb.Subscribe("TaskCreated", handler)

	task := &mockTask{
		id:      domain.NewTaskID("test-1"),
		traceID: domain.NewTraceID("trace-1"),
	}
	event := &mockEvent{task: task, eventType: "TaskCreated", timestamp: time.Now().Unix()}

	eb.Publish(event)
	time.Sleep(10 * time.Millisecond)

	eb.Unsubscribe("TaskCreated", handler)

	eb.Publish(event)
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 1 {
		t.Errorf("期望调用次数为 1, 实际为 %d", callCount)
	}
}

func TestEventBus_SubscribeFunc(t *testing.T) {
	eb := NewEventBus()

	var mu sync.Mutex
	eventCount := 0

	unsubscribe := eb.SubscribeFunc("TaskCreated", func(event domain.DomainEvent) {
		mu.Lock()
		eventCount++
		mu.Unlock()
	})

	task := &mockTask{
		id:      domain.NewTaskID("test-1"),
		traceID: domain.NewTraceID("trace-1"),
	}

	eb.Publish(&mockEvent{task: task, eventType: "TaskCreated", timestamp: time.Now().Unix()})
	time.Sleep(10 * time.Millisecond)
	eb.Publish(&mockEvent{task: task, eventType: "TaskCreated", timestamp: time.Now().Unix()})
	time.Sleep(10 * time.Millisecond)

	unsubscribe()

	eb.Publish(&mockEvent{task: task, eventType: "TaskCreated", timestamp: time.Now().Unix()})
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if eventCount != 2 {
		t.Errorf("期望事件计数为 2, 实际为 %d", eventCount)
	}
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	eb := NewEventBus()

	var mu sync.Mutex
	count1 := 0
	count2 := 0

	handler1 := func(event domain.DomainEvent) {
		mu.Lock()
		count1++
		mu.Unlock()
	}
	handler2 := func(event domain.DomainEvent) {
		mu.Lock()
		count2++
		mu.Unlock()
	}

	eb.Subscribe("TaskCreated", handler1)
	eb.Subscribe("TaskCreated", handler2)

	task := &mockTask{
		id:      domain.NewTaskID("test-1"),
		traceID: domain.NewTraceID("trace-1"),
	}
	event := &mockEvent{task: task, eventType: "TaskCreated", timestamp: time.Now().Unix()}

	eb.Publish(event)
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count1 != 1 {
		t.Errorf("期望 handler1 计数为 1, 实际为 %d", count1)
	}
	if count2 != 1 {
		t.Errorf("期望 handler2 计数为 1, 实际为 %d", count2)
	}
}

func TestEventBus_NoHandler(t *testing.T) {
	eb := NewEventBus()

	task := &mockTask{
		id:      domain.NewTaskID("test-1"),
		traceID: domain.NewTraceID("trace-1"),
	}
	event := &mockEvent{task: task, eventType: "TaskCreated", timestamp: time.Now().Unix()}

	eb.Publish(event)
}

func TestEventBus_Concurrent(t *testing.T) {
	eb := NewEventBus()

	var mu sync.Mutex
	eventCount := 0

	handler := func(event domain.DomainEvent) {
		mu.Lock()
		eventCount++
		mu.Unlock()
	}

	eb.Subscribe("TaskCreated", handler)

	task := &mockTask{
		id:      domain.NewTaskID("test-1"),
		traceID: domain.NewTraceID("trace-1"),
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			eb.Publish(&mockEvent{task: task, eventType: "TaskCreated", timestamp: time.Now().Unix()})
		}()
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if eventCount != 100 {
		t.Errorf("期望事件计数为 100, 实际为 %d", eventCount)
	}
}
