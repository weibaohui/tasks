/**
 * 领域事件定义
 */
package domain

import "time"

// DomainEvent 领域事件接口
type DomainEvent interface {
	// EventType 返回事件类型
	EventType() string
	// TraceID 返回追踪ID
	TraceID() TraceID
	// Timestamp 返回时间戳
	Timestamp() int64
}

// ReplenishRequiredEvent 需求补充信息事件
type ReplenishRequiredEvent struct {
	requirement *Requirement
	timestamp   int64
}

func NewReplenishRequiredEvent(requirement *Requirement) *ReplenishRequiredEvent {
	return &ReplenishRequiredEvent{
		requirement: requirement,
		timestamp:   time.Now().Unix(),
	}
}

func (e *ReplenishRequiredEvent) EventType() string {
	return "ReplenishRequired"
}

func (e *ReplenishRequiredEvent) TraceID() TraceID {
	return NewTraceID(e.requirement.TraceID())
}

func (e *ReplenishRequiredEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *ReplenishRequiredEvent) Requirement() *Requirement {
	return e.requirement
}
