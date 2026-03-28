/**
 * 值对象单元测试
 */
package domain

import (
	"testing"
	"time"
)

func TestTaskID(t *testing.T) {
	id := NewTaskID("test-id")

	if id.String() != "test-id" {
		t.Errorf("期望 String() 返回 'test-id', 实际为 '%s'", id.String())
	}

	if !id.Equals(NewTaskID("test-id")) {
		t.Error("期望两个相同的 TaskID 相等")
	}

	if id.Equals(NewTaskID("other-id")) {
		t.Error("期望两个不同的 TaskID 不相等")
	}
}

func TestTraceID(t *testing.T) {
	id := NewTraceID("trace-123")

	if id.String() != "trace-123" {
		t.Errorf("期望 String() 返回 'trace-123', 实际为 '%s'", id.String())
	}

	if !id.Equals(NewTraceID("trace-123")) {
		t.Error("期望两个相同的 TraceID 相等")
	}
}

func TestSpanID(t *testing.T) {
	id := NewSpanID("span-456")

	if id.String() != "span-456" {
		t.Errorf("期望 String() 返回 'span-456', 实际为 '%s'", id.String())
	}

	if !id.Equals(NewSpanID("span-456")) {
		t.Error("期望两个相同的 SpanID 相等")
	}
}

func TestTaskStatus(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusRunning, "running"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusFailed, "failed"},
		{TaskStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if tt.status.String() != tt.expected {
			t.Errorf("期望状态字符串为 '%s', 实际为 '%s'", tt.expected, tt.status.String())
		}
	}
}

func TestParseTaskStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected TaskStatus
		hasError bool
	}{
		{"pending", TaskStatusPending, false},
		{"running", TaskStatusRunning, false},
		{"completed", TaskStatusCompleted, false},
		{"failed", TaskStatusFailed, false},
		{"cancelled", TaskStatusCancelled, false},
		{"unknown", TaskStatusPending, true},
	}

	for _, tt := range tests {
		status, err := ParseTaskStatus(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("期望解析 '%s' 返回错误, 但实际没有", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("解析 '%s' 失败: %v", tt.input, err)
			}
			if status != tt.expected {
				t.Errorf("期望状态为 %v, 实际为 %v", tt.expected, status)
			}
		}
	}
}

func TestTaskType(t *testing.T) {
	tests := []struct {
		taskType TaskType
		expected string
	}{
		{TaskTypeAgent, "agent"},
		{TaskTypeCoding, "coding"},
		{TaskTypeCustom, "custom"},
	}

	for _, tt := range tests {
		if tt.taskType.String() != tt.expected {
			t.Errorf("期望类型字符串为 '%s', 实际为 '%s'", tt.expected, tt.taskType.String())
		}
	}
}

func TestParseTaskType(t *testing.T) {
	tests := []struct {
		input    string
		expected TaskType
		hasError bool
	}{
		{"agent", TaskTypeAgent, false},
		{"coding", TaskTypeCoding, false},
		{"custom", TaskTypeCustom, false},
		{"unknown", TaskTypeCustom, true},
	}

	for _, tt := range tests {
		taskType, err := ParseTaskType(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("期望解析 '%s' 返回错误, 但实际没有", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("解析 '%s' 失败: %v", tt.input, err)
			}
			if taskType != tt.expected {
				t.Errorf("期望类型为 %v, 实际为 %v", tt.expected, taskType)
			}
		}
	}
}

func TestProgress(t *testing.T) {
	p := NewProgress()

	if p.Value() != 0 {
		t.Errorf("期望初始进度为 0, 实际为 %d", p.Value())
	}
}

func TestProgress_Update(t *testing.T) {
	p := NewProgress()

	clamped := p.Update(50)

	if p.Value() != 50 {
		t.Errorf("期望进度为 50, 实际为 %d", p.Value())
	}
	if clamped {
		t.Error("期望未发生 clamp")
	}
}

func TestProgress_Update_ZeroTotal(t *testing.T) {
	p := NewProgress()

	clamped := p.Update(0)

	if p.Value() != 0 {
		t.Errorf("期望进度为 0, 实际为 %d", p.Value())
	}
	if clamped {
		t.Error("期望未发生 clamp")
	}
}

func TestProgress_Update_ClampBoundary(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
		clamped  bool
	}{
		{"负数 clamp 到 0", -1, 0, true},
		{"零值不变", 0, 0, false},
		{"正常值 1 不变", 1, 1, false},
		{"正常值 99 不变", 99, 99, false},
		{"正常值 100 不变", 100, 100, false},
		{"超过 100 clamp 到 100", 101, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProgress()
			clamped := p.Update(tt.input)

			if p.Value() != tt.expected {
				t.Errorf("期望进度为 %d, 实际为 %d", tt.expected, p.Value())
			}
			if clamped != tt.clamped {
				t.Errorf("期望 clamp=%v, 实际为 %v", tt.clamped, clamped)
			}
		})
	}
}

func TestProgress_ToMap(t *testing.T) {
	p := NewProgress()
	p.Update(75)

	m := p.ToMap()

	if m["value"].(int) != 75 {
		t.Errorf("ToMap value 期望 75, 实际 %v", m["value"])
	}
}

func TestProgress_UpdatedAt(t *testing.T) {
	p := NewProgress()
	before := time.Now()

	p.Update(50)

	after := time.Now()

	if p.UpdatedAt().Before(before) || p.UpdatedAt().After(after) {
		t.Error("UpdatedAt 时间不在预期范围内")
	}
}
