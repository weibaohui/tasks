/**
 * EventBus 内存事件总线
 * 负责发布领域事件到订阅者
 */
package bus

import (
	"fmt"
	"sync"

	"github.com/weibh/taskmanager/domain"
)

// EventBus 事件总线
type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]domain.EventHandler
}

// NewEventBus 创建事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]domain.EventHandler),
	}
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(eventType string, handler domain.EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Unsubscribe 取消订阅
func (eb *EventBus) Unsubscribe(eventType string, handler domain.EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	handlers := eb.handlers[eventType]
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			return
		}
	}
}

// Publish 发布事件
func (eb *EventBus) Publish(event domain.DomainEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	eventType := event.EventType()
	handlers := eb.handlers[eventType]
	for _, handler := range handlers {
		go handler(event)
	}
}

// SubscribeFunc 返回取消订阅函数
func (eb *EventBus) SubscribeFunc(eventType string, handler domain.EventHandler) func() {
	eb.Subscribe(eventType, handler)
	return func() {
		eb.Unsubscribe(eventType, handler)
	}
}
