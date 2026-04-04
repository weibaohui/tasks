/**
 * 值对象定义
 * 包含 TraceID, SpanID, Progress 等
 */
package domain

import (
	"time"
)

// TraceID 追踪ID值对象
type TraceID struct {
	value string
}

func NewTraceID(value string) TraceID {
	return TraceID{value: value}
}

func (id TraceID) String() string {
	return id.value
}

func (id TraceID) Equals(other TraceID) bool {
	return id.value == other.value
}

// SpanID 跨度ID值对象
type SpanID struct {
	value string
}

func NewSpanID(value string) SpanID {
	return SpanID{value: value}
}

func (id SpanID) String() string {
	return id.value
}

func (id SpanID) Equals(other SpanID) bool {
	return id.value == other.value
}

// Progress 进度值对象
type Progress struct {
	value     int
	updatedAt time.Time
}

// NewProgress 创建进度对象
func NewProgress() Progress {
	return Progress{
		value:     0,
		updatedAt: time.Now(),
	}
}

// Update 更新进度（自动 clamping 到 0-100 范围）
// 返回是否发生了 clamp（值为超出范围被调整）
func (p *Progress) Update(value int) bool {
	clamped := false
	if value < 0 {
		value = 0
		clamped = true
	}
	if value > 100 {
		value = 100
		clamped = true
	}
	p.value = value
	p.updatedAt = time.Now()
	return clamped
}

// Value 获取进度值
func (p Progress) Value() int {
	return p.value
}

// UpdatedAt 更新时间
func (p Progress) UpdatedAt() time.Time {
	return p.updatedAt
}

// ToMap 转换为map
func (p Progress) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"value":      p.value,
		"updated_at": p.updatedAt.Unix(),
	}
}
