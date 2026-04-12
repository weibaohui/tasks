package domain

// EventHandler 事件处理函数
type EventHandler func(event DomainEvent)

// EventBus 领域事件总线接口
type EventBus interface {
	Subscribe(eventType string, handler EventHandler)
	Unsubscribe(eventType string, handler EventHandler)
	Publish(event DomainEvent)
	SubscribeFunc(eventType string, handler EventHandler) func()
}
